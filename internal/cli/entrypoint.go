package cli

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Main is the testable process entrypoint: it assembles the command tree,
// dispatches os.Args through Run, and maps the resulting Outcome to a process
// exit code via the canonical registry (ExitCode). It returns the code rather
// than calling os.Exit, so main() stays a one-line wrapper and the exit-code
// and panic paths remain observable in-process (a subprocess smoke test covers
// the real os.Exit wiring).
//
// The recover that turns a panic into code 1 lives in runToExitCode, the core
// Main delegates to — so the in-process tests and the BDD harness exercise the
// exact same recover+map code Main runs, against a custom tree, without
// re-implementing it.
func Main() int {
	return runToExitCode(Assemble(), os.Args[1:])
}

// runToExitCode dispatches args against root, maps the outcome to its exit code,
// and recovers an unrecovered panic to codeInternalError (ADR-4). Go terminates
// an unrecovered panic with status 2, which would collide with UsageError = 2
// and let an internal crash masquerade as the operator's input error; the
// recover closes that gap. It writes the panic value and stack to stderr so the
// crash stays diagnosable (Action Transparency, CONSTITUTION II) — the one place
// this capability writes text; the numeric code is its only outcome signal.
//
// The returned error from Run is intentionally discarded: 004 renders no text,
// and any error message has already been written by cobra or dispatch (002).
// The exit code is derived purely from the code-free Outcome category.
func runToExitCode(root *cobra.Command, args []string) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "glassfrog: internal error: %v\n\n%s", r, debug.Stack())
			code = codeInternalError
		}
	}()
	outcome, _ := Run(root, args)
	return ExitCode(outcome)
}
