package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Outcome is the code-free classification dispatch assigns to an invocation.
// It names *what kind* of outcome occurred — never a process exit code (that
// is Exit-Code Convention's concern, a later spec) and never a rendered
// message (Help & Version's). Exit-Code Convention (004) is the consumer that
// will map this category to a process code without re-deriving it from an
// untyped error (ADR-2).
//
// The category is deliberately minimal — two values now. A third value for a
// resolved command whose own action fails (a future RuntimeError) is deferred
// to Exit-Code Convention, the capability that needs to tell runtime failures
// apart. Until then, a resolved command that ran is Success and its action
// error, if any, travels via the error returned by Run.
type Outcome int

const (
	// Success means a command was routed and dispatched: it ran (its action
	// may itself have returned an error, surfaced via Run's error return), or a
	// group/root resolved to a help/listing outcome.
	Success Outcome = iota
	// UsageError means the invocation did not name a valid command, or carried
	// an unknown flag / unexpected argument — nothing ran, or running was
	// refused.
	UsageError
)

// String renders the category name for legibility in logs and test failures.
func (o Outcome) String() string {
	switch o {
	case Success:
		return "Success"
	case UsageError:
		return "UsageError"
	default:
		// Preserve the underlying value for an unexpected Outcome (e.g. a future
		// enum extension) rather than collapsing it to a constant — keeps logs
		// and test failures debuggable.
		return fmt.Sprintf("Outcome(%d)", int(o))
	}
}

// Run resolves args against the assembled root command tree, executes the
// resolved command (or routes a group/root to a help/listing outcome), and
// returns the categorized Outcome plus any error. It is the single dispatch
// boundary: the entrypoint calls Run instead of executing the tree directly
// (ADR-2). Run never emits an exit code and never terminates the process — the
// caller acts on the returned Outcome.
//
// Matching is exact at every level: cobra's EnablePrefixMatching package-global
// is left at its default of false (ADR-1), so a prefix never resolves to a
// longer command. Unknown-flag rejection is cobra's default and is left on.
//
// Outcome derivation classifies at the seams cobra exposes, because cobra has
// no typed "command not found" (see plan Risks):
//
//   - a flag-parse failure (unknown/malformed flag) → UsageError. A
//     SetFlagErrorFunc hook records it because the failing command may itself be
//     runnable, so the error alone can't be told apart from a runtime failure.
//   - an error against a non-runnable node (the root rejecting an unknown first
//     token) → UsageError.
//   - a non-runnable group that cobra resolved to its help output while
//     positional tokens went unmatched → UsageError naming the first such token.
//     cobra's default arg validator only rejects unknown subcommands at the
//     root; a nested group prints help and returns nil, so dispatch surfaces the
//     swallowed token from the group's leftover positional args itself. Because
//     cobra never printed this synthesized error, dispatch writes it (plus a
//     help pointer) to stderr — matching how cobra reports its own
//     unknown-command errors, so the operator always sees the unrecognized token.
//   - an error on a runnable command whose own arg validator rejected the input
//     (an unexpected positional argument the command does not accept) → the
//     action never ran, so this is a UsageError. Each command declares what it
//     accepts via its cobra Args validator; dispatch re-checks it to tell an
//     arg rejection apart from a runtime failure.
//   - anything else with an error came from a resolved command's own action →
//     the error is returned but the category stays Success. Distinguishing that
//     runtime failure (a future RuntimeError) is deferred to Exit-Code
//     Convention (004).
func Run(root *cobra.Command, args []string) (Outcome, error) {
	root.SetArgs(args)

	// A flag-parse failure is a usage error even when the command it occurs on
	// is runnable, so the returned error can't classify it on its own. The hook
	// propagates to descendants (cobra falls back to the nearest ancestor's
	// FlagErrorFunc), so setting it on the root covers the whole tree.
	flagFailed := false
	root.SetFlagErrorFunc(func(_ *cobra.Command, ferr error) error {
		flagFailed = true
		return ferr
	})

	executed, err := root.ExecuteC()

	switch {
	case flagFailed:
		// Unknown or malformed flag: the resolved command did not run.
		return UsageError, err
	case err != nil && executed != nil && !executed.Runnable():
		// Resolution failed against a non-runnable node (the root rejecting an
		// unknown first token); no command action ran.
		return UsageError, err
	case err == nil && executed != nil && !executed.Runnable():
		// cobra routed a non-runnable group/root to its help output. If
		// positional tokens were left unmatched, the caller typed an unknown
		// subcommand that cobra silently swallowed — surface it. cobra returned
		// nil here, so it printed no error of its own; dispatch writes the
		// synthesized error and a help pointer to stderr, the same way cobra
		// reports its own unknown-command errors, so the operator sees the
		// unrecognized token rather than only the group's help.
		if leftover := executed.Flags().Args(); len(leftover) > 0 {
			usageErr := fmt.Errorf("unknown command %q for %q", leftover[0], executed.CommandPath())
			fmt.Fprintf(executed.ErrOrStderr(), "Error: %s\nRun '%s --help' for usage.\n", usageErr, executed.CommandPath())
			return UsageError, usageErr
		}
		return Success, nil
	default:
		// A command ran, or its arg validator rejected the input before the
		// action ran. Re-check the resolved command's Args against its parsed
		// positional args: a non-nil result means the command refused an
		// unexpected argument (a usage error — the action never ran). Otherwise
		// the action ran and any error is its own runtime failure, returned
		// uncategorized (RuntimeError deferred to 004).
		if err != nil && executed != nil && executed.ValidateArgs(executed.Flags().Args()) != nil {
			return UsageError, err
		}
		return Success, err
	}
}
