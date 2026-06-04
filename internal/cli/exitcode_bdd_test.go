package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/cucumber/godog"
	"github.com/spf13/cobra"
)

// registerExitCodeSteps wires the Exit-Code Convention (004) Given/Then steps
// onto the shared world. The When ("the caller invokes …") is the existing 002
// step, which now routes through the production recover+map core and records
// the resulting exit code (see whenCallerInvokes). Only the genuinely-new
// assertions — registering a failing/panicking command and asserting the
// process exit code — need fresh bindings; everything else reuses the existing
// dispatch vocabulary.
func (w *world) registerExitCodeSteps(sc *godog.ScenarioContext) {
	// --- Givens (failure producers the success-path tree lacks) ---
	sc.Step(`^a "([^"]*)" command whose action fails for a reason matching no known category is registered$`, w.givenFailingCommand)
	sc.Step(`^a "([^"]*)" command whose action panics is registered$`, w.givenPanickingCommand)

	// --- Thens (the process exit code; the only new assertion 004 introduces) ---
	sc.Step(`^the process will exit with code (\d+)$`, w.thenExitWithCode)
	sc.Step(`^it will not exit with code (\d+)$`, w.thenNotExitWithCode)
}

// --- Given implementations ---

// givenFailingCommand registers a valid leaf whose action returns an error —
// a resolved command that ran but failed, classified RuntimeError → code 1.
func (w *world) givenFailingCommand(name string) error {
	return Register(w.root, &cobra.Command{
		Use:   name,
		Short: "the " + name + " command",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			w.ran[name] = true
			return errors.New(name + " failed")
		},
	})
}

// givenPanickingCommand registers a valid leaf whose action panics — an
// unexpected internal failure the entrypoint must recover to code 1, never
// Go's default panic status 2 (ADR-4).
func (w *world) givenPanickingCommand(name string) error {
	return Register(w.root, &cobra.Command{
		Use:   name,
		Short: "the " + name + " command",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			w.ran[name] = true
			panic(name + " panicked")
		},
	})
}

// --- Then implementations ---

func (w *world) thenExitWithCode(code int) error {
	if !w.exitCodeSet {
		return fmt.Errorf("no invocation was dispatched")
	}
	if w.exitCode != code {
		return fmt.Errorf("process exit code = %d, want %d", w.exitCode, code)
	}
	return nil
}

func (w *world) thenNotExitWithCode(code int) error {
	if !w.exitCodeSet {
		return fmt.Errorf("no invocation was dispatched")
	}
	if w.exitCode == code {
		return fmt.Errorf("process exit code = %d, but it must not be %d", w.exitCode, code)
	}
	return nil
}

// silenceOSStderr redirects os.Stderr to the null device and returns a restore
// func, so the panic diagnostic the recover writes (recoverToCode) does not
// pollute the test log. Best-effort: if the null device cannot be opened the
// original stderr is left in place. The unit test pins the diagnostic's content
// directly (TestRunToExitCode_PanicYieldsOneNotTwo).
func silenceOSStderr() func() {
	orig := os.Stderr
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}
	}
	os.Stderr = devnull
	return func() {
		os.Stderr = orig
		devnull.Close()
	}
}
