package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/spf13/cobra"
)

// loginConfig carries everything runLogin needs, gathered by the seam. Keeping
// runLogin a function of injected values (not the seam) makes the whole
// store decision — resolve, guard, write, report — testable over temp dirs
// with a fake interactor, exercising the real auth writer/reader.
type loginConfig struct {
	inputs    tokenInputs
	homeDir   string
	startDir  string
	cwd       bool
	overwrite bool
	interact  interactor
	stdout    io.Writer
	stderr    io.Writer
}

// runLogin resolves the token, applies the existing-token guard, writes through
// the internal/auth writer, and reports the written path — returning a code-free
// Outcome the command maps for Exit-Code Convention (ADR-5). It never emits an
// exit code and never prints the token: the success line names the path only,
// and every error message names paths only.
func runLogin(cfg loginConfig) (Outcome, error) {
	// 1. Resolve the token by precedence; prompt only when interactive.
	raw, source := resolveTokenSource(cfg.inputs)
	switch source {
	case tokenNone:
		fmt.Fprintln(cfg.stderr, "no token to store — supply a token via argument, stdin, or GLASSFROG_TOKEN")
		return UsageError, errors.New("no token to store")
	case tokenNeedsPrompt:
		prompted, err := cfg.interact.promptToken()
		if err != nil {
			fmt.Fprintln(cfg.stderr, "could not read the token from the prompt")
			return RuntimeError, err
		}
		raw = prompted
	}

	token, ok := usableToken(raw)
	if !ok {
		fmt.Fprintln(cfg.stderr, "the supplied token is empty — supply a non-empty token")
		return UsageError, errors.New("empty token")
	}

	// 2. Select the target and detect an existing token (a malformed existing
	// file fails loud here, before any write).
	target := targetPath(cfg.homeDir, cfg.startDir, cfg.cwd)
	_, hasExisting, rerr := auth.ReadCredentialsFile(target)
	// An absent target is not an error — it means "no existing token, proceed".
	// The shared reader (005) reports absence as a *ReadError wrapping
	// os.ErrNotExist; only a genuine read/format failure is surfaced.
	if rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		return classifyAuthError(cfg.stderr, target, rerr)
	}

	// 3. Existing-token guard.
	targets := []string{target}
	switch existingTokenGuard(hasExisting, cfg.inputs.isTTY, cfg.overwrite) {
	case guardBlocked:
		fmt.Fprintf(cfg.stderr, "a credential already exists at %s — pass --overwrite to replace it\n", target)
		return UsageError, errors.New("existing credential, no --overwrite")
	case guardInteractive:
		confirmed, err := cfg.interact.confirmReplace(target)
		if err != nil {
			fmt.Fprintln(cfg.stderr, "could not read the confirmation")
			return RuntimeError, err
		}
		if !confirmed {
			fmt.Fprintln(cfg.stdout, "No changes made.")
			return Success, nil
		}
		chosen, err := cfg.interact.chooseLocations(homeTokenPath(cfg.homeDir), cwdTokenPath(cfg.startDir))
		if err != nil {
			fmt.Fprintln(cfg.stderr, "could not read the location choice")
			return RuntimeError, err
		}
		if len(chosen) > 0 {
			targets = chosen
		}
	case guardProceed:
		// targets stays the single selected path.
	}

	// 4. Write to each target and report the path (never the token).
	for _, path := range targets {
		if err := auth.WriteCredentials(path, token); err != nil {
			return classifyAuthError(cfg.stderr, path, err)
		}
		fmt.Fprintf(cfg.stdout, "Stored credentials in %s\n", path)
	}
	return Success, nil
}

// classifyAuthError maps an internal/auth error to a RuntimeError outcome and
// writes the operator-facing message (path only, never the token). It
// discriminates the three distinct failures so the message names the real cause
// and never over-claims about the filesystem:
//   - *FormatError — an existing file is malformed; it was not overwritten.
//   - *WriteError — the write to this path failed; the atomic temp+rename
//     guarantees this path was not partially written. The claim is scoped to
//     path only: when writing several targets (the interactive "both" choice),
//     an earlier target may already have been written, so no global
//     "filesystem unchanged" claim is made.
//   - anything else — e.g. an existing credentials file that could not be read
//     during the pre-write guard: report it as a read/access failure, not a
//     write error, since no write was attempted.
func classifyAuthError(stderr io.Writer, path string, err error) (Outcome, error) {
	var fe *auth.FormatError
	if errors.As(err, &fe) {
		fmt.Fprintf(stderr, "format error: %s is malformed — fix or remove it; it was not overwritten\n", path)
		return RuntimeError, err
	}
	var we *auth.WriteError
	if errors.As(err, &we) {
		fmt.Fprintf(stderr, "write error: could not write credentials to %s — check write permission on the directory; %s was not partially written\n", path, path)
		return RuntimeError, err
	}
	fmt.Fprintf(stderr, "error: could not read the existing credentials at %s — check its permissions; no changes were made\n", path)
	return RuntimeError, err
}

// newAuthCommand assembles the `auth` credential command group and its `login`
// leaf, registered through the guard (group-has-children, non-empty Short).
// The seam is injected so tests drive a fake one; production passes
// productionSeam{} from Assemble.
func newAuthCommand(seam loginSeam) *cobra.Command {
	group := &cobra.Command{
		Use:   "auth",
		Short: "Manage Glassfrog API credentials",
	}
	MustRegister(group, newAuthLoginCommand(seam))
	return group
}

// newAuthLoginCommand builds `auth login [TOKEN]`: at most one positional
// (MaximumNArgs(1)), --cwd and --overwrite flags, delegating the write to
// internal/auth and classifying the outcome for Exit-Code Convention. cobra's
// own error/usage dump is silenced because runLogin writes its own controlled,
// token-free messages.
func newAuthLoginCommand(seam loginSeam) *cobra.Command {
	var cwd, overwrite bool
	cmd := &cobra.Command{
		Use:           "login [TOKEN]",
		Short:         "Store an API token to the credentials file",
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs, err := seam.gatherInputs(args)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "could not read the token input")
				return err
			}
			home, err := seam.homeDir()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "could not determine the home directory")
				return err
			}
			start, err := seam.startDir()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "could not determine the working directory")
				return err
			}
			outcome, oerr := runLogin(loginConfig{
				inputs:    inputs,
				homeDir:   home,
				startDir:  start,
				cwd:       cwd,
				overwrite: overwrite,
				interact:  seam.interactor(),
				stdout:    cmd.OutOrStdout(),
				stderr:    cmd.ErrOrStderr(),
			})
			// Map the code-free Outcome onto the error channel dispatch reads: a
			// UsageError is wrapped so dispatch classifies it (→ code 2); a
			// RuntimeError travels as-is (→ code 1); Success is a clean return.
			switch outcome {
			case Success:
				return nil
			case UsageError:
				return &commandUsageError{oerr}
			default:
				return oerr
			}
		},
	}
	cmd.Flags().BoolVar(&cwd, "cwd", false, "Write ./.glassfrogrc in the current directory instead of the home file")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Replace an existing stored token without prompting")
	return cmd
}
