package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/cucumber/godog"
)

// credState is the per-scenario fixture for the Credential Storage (006)
// executable acceptance. Everything is confined to temp directories and
// injected sources — no step reads the real stdin/TTY/env or the developer's
// real home directory (ADR-3; the test-isolation risk in tasks.md T005).
type credState struct {
	home    string
	start   string
	created []string // temp dirs to chmod-restore + remove after the scenario

	isTTY      bool
	env        string
	stdin      string
	stdinPiped bool
	arg        string
	argGiven   bool
	cwd        bool
	overwrite  bool

	promptValue string
	confirm     bool
	chosen      []string
	inter       *fakeInteractor

	// secrets are token values that must never appear in produced output.
	secrets []string

	outcome Outcome
	stdout  string
	stderr  string
}

func (cs *credState) homeDir() string {
	if cs.home == "" {
		d, err := os.MkdirTemp("", "gf-home-*")
		if err != nil {
			panic(err)
		}
		cs.home, cs.created = d, append(cs.created, d)
	}
	return cs.home
}

func (cs *credState) startDir() string {
	if cs.start == "" {
		d, err := os.MkdirTemp("", "gf-cwd-*")
		if err != nil {
			panic(err)
		}
		cs.start, cs.created = d, append(cs.created, d)
	}
	return cs.start
}

func (cs *credState) homePath() string { return filepath.Join(cs.homeDir(), auth.CredentialsFileName) }
func (cs *credState) cwdPath() string  { return filepath.Join(cs.startDir(), auth.CredentialsFileName) }

func (cs *credState) cleanup() {
	for _, d := range cs.created {
		_ = os.Chmod(d, 0o700) // an unwritable-dir scenario left it 0500
		_ = os.RemoveAll(d)
	}
}

func (w *world) registerCredStorageSteps(sc *godog.ScenarioContext) {
	sc.After(func(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
		if w.cred != nil {
			w.cred.cleanup()
		}
		return ctx, err
	})

	// --- Givens ---
	sc.Step(`^no "\.glassfrogrc" existed in the home directory$`, w.credNoHomeFile)
	sc.Step(`^the operator supplied the token "([^"]*)" as a command argument$`, w.credArgToken)
	sc.Step(`^the home "\.glassfrogrc" location could not be written$`, w.credHomeUnwritable)
	sc.Step(`^the home "\.glassfrogrc" held a line that was neither blank, a comment, nor a "key=value" pair$`, w.credHomeMalformed)
	sc.Step(`^the operator supplied a token that was only whitespace$`, w.credBlankToken)
	sc.Step(`^the home "\.glassfrogrc" held the token "([^"]*)" and an unrelated entry$`, w.credHomeTokenAndOther)
	sc.Step(`^the session was interactive \(standard input is a terminal\)$`, w.credInteractive)
	sc.Step(`^the operator confirmed replacing the token with "([^"]*)"$`, w.credConfirmReplaceWith)
	sc.Step(`^no token was supplied as an argument, on standard input, or in GLASSFROG_TOKEN$`, w.credNoSource)
	sc.Step(`^GLASSFROG_TOKEN was set to "([^"]*)"$`, w.credEnvToken)
	sc.Step(`^the operator supplied no argument and piped nothing to standard input$`, w.credNoSource)
	sc.Step(`^the session had no terminal on standard input$`, w.credNonInteractive)
	sc.Step(`^the home "\.glassfrogrc" already held the token "([^"]*)"$`, w.credHomeHeldToken)
	sc.Step(`^the --overwrite flag was not given$`, w.credNoOverwriteFlag)
	sc.Step(`^the token "([^"]*)" was piped to standard input$`, w.credPipedToken)
	sc.Step(`^the operator supplied no token argument$`, func() error { return nil })
	sc.Step(`^the --cwd flag was given$`, w.credCwdFlag)
	sc.Step(`^a "\.glassfrogrc" in the current directory and one in the home directory each held a token$`, w.credBothLocationsHeldToken)

	// --- When ---
	sc.Step(`^the CLI stores the credential$`, w.credStore)

	// --- Thens ---
	sc.Step(`^it will create the home "\.glassfrogrc" holding the token "([^"]*)"$`, w.credHomeHoldsToken)
	sc.Step(`^it will report the path it wrote$`, w.credReportedPath)
	sc.Step(`^the token value will not appear in the output$`, w.credNoTokenInOutput)
	sc.Step(`^it will report a write error naming that location$`, w.credWriteErrorNamingLocation)
	sc.Step(`^the filesystem will be unchanged afterward$`, w.credFilesystemUnchanged)
	sc.Step(`^it will report a format error naming that file$`, w.credFormatErrorNamingFile)
	sc.Step(`^it will not overwrite the file$`, w.credDidNotOverwrite)
	sc.Step(`^it will reject the token as unusable$`, w.credRejectedUnusable)
	sc.Step(`^it will not write any file$`, w.credWroteNoFile)
	sc.Step(`^it will replace only the token entry with "([^"]*)"$`, w.credReplacedTokenWith)
	sc.Step(`^it will leave the unrelated entry unchanged$`, w.credUnrelatedPreserved)
	sc.Step(`^it will prompt for the token without echoing the typed characters$`, w.credPromptedNoEcho)
	sc.Step(`^it will write the entered token to the home "\.glassfrogrc"$`, w.credHomeHoldsPrompted)
	sc.Step(`^it will write the token "([^"]*)" to the home "\.glassfrogrc"$`, w.credHomeHoldsToken)
	sc.Step(`^it will report that there is no token to store$`, w.credReportedNoToken)
	sc.Step(`^it will report an error$`, w.credReportedAnError)
	sc.Step(`^it will leave the existing credentials unchanged$`, w.credExistingUnchanged)
	sc.Step(`^it will write the token "([^"]*)" to the current directory's "\.glassfrogrc"$`, w.credCwdHoldsToken)
	sc.Step(`^it will confirm before changing the existing tokens$`, w.credConfirmed)
	sc.Step(`^it will offer to write the current-directory file, the home file, or both$`, w.credOfferedLocations)
}

// --- Given implementations ---

func (w *world) credNoHomeFile() error { w.cred.homeDir(); return nil }

func (w *world) credArgToken(tok string) error {
	w.cred.arg, w.cred.argGiven = tok, true
	w.cred.secrets = append(w.cred.secrets, tok)
	return nil
}

func (w *world) credHomeUnwritable() error {
	dir := w.cred.homeDir()
	return os.Chmod(dir, 0o500)
}

func (w *world) credHomeMalformed() error {
	return os.WriteFile(w.cred.homePath(), []byte("a line that is not a comment and has no equals\n"), 0o600)
}

func (w *world) credBlankToken() error {
	w.cred.arg, w.cred.argGiven = "   ", true
	return nil
}

func (w *world) credHomeTokenAndOther(tok string) error {
	w.cred.secrets = append(w.cred.secrets, tok)
	return os.WriteFile(w.cred.homePath(), []byte("# my credentials\ntoken="+tok+"\nother=keep_me\n"), 0o600)
}

func (w *world) credInteractive() error {
	w.cred.isTTY = true
	// Scenarios that prompt for a missing token quote no value; default the
	// entered token so the "write the entered token" assertion has a target.
	// A later "confirmed replacing with X" step overrides it.
	if w.cred.promptValue == "" {
		w.cred.promptValue = "gf_prompted_token"
		w.cred.secrets = append(w.cred.secrets, "gf_prompted_token")
	}
	return nil
}
func (w *world) credNonInteractive() error { w.cred.isTTY = false; return nil }
func (w *world) credNoSource() error       { return nil }

func (w *world) credConfirmReplaceWith(tok string) error {
	// The replacement token enters interactively (no arg/stdin/env), so it is
	// the prompted value; confirming the replace drives the confirm path.
	w.cred.confirm = true
	w.cred.promptValue = tok
	w.cred.secrets = append(w.cred.secrets, tok)
	return nil
}

func (w *world) credEnvToken(tok string) error {
	w.cred.env = tok
	w.cred.secrets = append(w.cred.secrets, tok)
	return nil
}

func (w *world) credHomeHeldToken(tok string) error {
	w.cred.secrets = append(w.cred.secrets, tok)
	return os.WriteFile(w.cred.homePath(), []byte("token="+tok+"\n"), 0o600)
}

func (w *world) credNoOverwriteFlag() error { w.cred.overwrite = false; return nil }

func (w *world) credPipedToken(tok string) error {
	w.cred.stdin, w.cred.stdinPiped = tok+"\n", true
	w.cred.secrets = append(w.cred.secrets, tok)
	return nil
}

func (w *world) credCwdFlag() error { w.cred.cwd = true; return nil }

func (w *world) credBothLocationsHeldToken() error {
	// This scenario is the interactive confirm-and-choose flow: the operator
	// confirms the change, which is what surfaces the location choice.
	w.cred.confirm = true
	w.cred.secrets = append(w.cred.secrets, "gf_home_old", "gf_cwd_old")
	if err := os.WriteFile(w.cred.homePath(), []byte("token=gf_home_old\n"), 0o600); err != nil {
		return err
	}
	return os.WriteFile(w.cred.cwdPath(), []byte("token=gf_cwd_old\n"), 0o600)
}

// --- When implementation ---

func (w *world) credStore() error {
	cs := w.cred
	cs.inter = &fakeInteractor{promptValue: cs.promptValue, confirm: cs.confirm, chosen: cs.chosen}
	seam := &fakeSeam{
		home:  cs.homeDir(),
		start: cs.startDir(),
		inter: cs.inter,
		inputs: tokenInputs{
			stdin:      cs.stdin,
			stdinPiped: cs.stdinPiped,
			env:        cs.env,
			isTTY:      cs.isTTY,
		},
	}
	root := NewRootCommand()
	MustRegister(root, newAuthCommand(seam))

	args := []string{"auth", "login"}
	if cs.argGiven {
		args = append(args, cs.arg)
	}
	if cs.cwd {
		args = append(args, "--cwd")
	}
	if cs.overwrite {
		args = append(args, "--overwrite")
	}

	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	cs.outcome, _ = Run(root, args)
	cs.stdout, cs.stderr = out.String(), errb.String()
	return nil
}

// --- Then implementations ---

func (w *world) credHomeHoldsToken(want string) error { return tokenAt(w.cred.homePath(), want) }
func (w *world) credCwdHoldsToken(want string) error  { return tokenAt(w.cred.cwdPath(), want) }
func (w *world) credReplacedTokenWith(want string) error {
	return tokenAt(w.cred.homePath(), want)
}

func (w *world) credHomeHoldsPrompted() error { return tokenAt(w.cred.homePath(), w.cred.promptValue) }

func (w *world) credReportedPath() error {
	if !strings.Contains(w.cred.stdout, w.cred.homePath()) {
		return fmt.Errorf("stdout should name the written path %q, got %q", w.cred.homePath(), w.cred.stdout)
	}
	return nil
}

func (w *world) credNoTokenInOutput() error {
	combined := w.cred.stdout + w.cred.stderr
	for _, s := range w.cred.secrets {
		if s != "" && strings.Contains(combined, s) {
			return fmt.Errorf("token value %q leaked into output: %q", s, combined)
		}
	}
	return nil
}

func (w *world) credWriteErrorNamingLocation() error {
	if !strings.Contains(w.cred.stderr, "write error") || !strings.Contains(w.cred.stderr, w.cred.homePath()) {
		return fmt.Errorf("stderr should report a write error naming %q, got %q", w.cred.homePath(), w.cred.stderr)
	}
	return nil
}

func (w *world) credFilesystemUnchanged() error {
	_ = os.Chmod(w.cred.homeDir(), 0o700)
	if _, err := os.Stat(w.cred.homePath()); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("home credentials file should not exist after a failed write, stat err = %v", err)
	}
	entries, err := os.ReadDir(w.cred.homeDir())
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("a temp file was left behind: %v", entries)
	}
	return nil
}

func (w *world) credFormatErrorNamingFile() error {
	if !strings.Contains(w.cred.stderr, "format error") || !strings.Contains(w.cred.stderr, w.cred.homePath()) {
		return fmt.Errorf("stderr should report a format error naming %q, got %q", w.cred.homePath(), w.cred.stderr)
	}
	return nil
}

func (w *world) credDidNotOverwrite() error {
	got, err := os.ReadFile(w.cred.homePath())
	if err != nil {
		return err
	}
	if !strings.Contains(string(got), "no equals") {
		return fmt.Errorf("the malformed file was overwritten, got:\n%s", got)
	}
	return nil
}

func (w *world) credRejectedUnusable() error {
	if w.cred.outcome != UsageError {
		return fmt.Errorf("outcome = %v, want UsageError for an unusable token", w.cred.outcome)
	}
	return nil
}

func (w *world) credWroteNoFile() error {
	if err := assertAbsent(w.cred.homePath()); err != nil {
		return err
	}
	return assertAbsent(w.cred.cwdPath())
}

func (w *world) credUnrelatedPreserved() error {
	got, err := os.ReadFile(w.cred.homePath())
	if err != nil {
		return err
	}
	if !strings.Contains(string(got), "other=keep_me") {
		return fmt.Errorf("unrelated entry not preserved, got:\n%s", got)
	}
	return nil
}

func (w *world) credPromptedNoEcho() error {
	if w.cred.inter == nil || !w.cred.inter.promptCalled {
		return errors.New("the interactive prompt was not used for a missing token")
	}
	return w.credNoTokenInOutput()
}

func (w *world) credReportedNoToken() error {
	if w.cred.outcome != UsageError {
		return fmt.Errorf("outcome = %v, want UsageError", w.cred.outcome)
	}
	if !strings.Contains(w.cred.stderr, "no token to store") {
		return fmt.Errorf("stderr should report no token to store, got %q", w.cred.stderr)
	}
	return nil
}

func (w *world) credReportedAnError() error {
	if w.cred.outcome == Success {
		return errors.New("expected an error outcome, got Success")
	}
	if strings.TrimSpace(w.cred.stderr) == "" {
		return errors.New("an error should be reported on stderr")
	}
	return nil
}

func (w *world) credExistingUnchanged() error {
	// The existing credential (gf_old_token in these scenarios) is intact.
	return tokenAt(w.cred.homePath(), "gf_old_token")
}

func (w *world) credConfirmed() error {
	if w.cred.inter == nil || !w.cred.inter.confirmCalled {
		return errors.New("the store did not confirm before changing the existing token")
	}
	return nil
}

func (w *world) credOfferedLocations() error {
	in := w.cred.inter
	if in == nil || !in.chooseCalled {
		return errors.New("the store did not offer a location choice")
	}
	if in.choseHome != w.cred.homePath() || in.choseCwd != w.cred.cwdPath() {
		return fmt.Errorf("location choice should offer home %q and cwd %q, got home=%q cwd=%q",
			w.cred.homePath(), w.cred.cwdPath(), in.choseHome, in.choseCwd)
	}
	return nil
}

// --- shared assertion helpers ---

func tokenAt(path, want string) error {
	tok, has, err := auth.ReadCredentialsFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if !has || tok != want {
		return fmt.Errorf("token at %s = %q (present=%v), want %q", path, tok, has, want)
	}
	return nil
}

func assertAbsent(path string) error {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("expected no file at %s, stat err = %v", path, err)
	}
	return nil
}
