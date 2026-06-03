package cli

import "github.com/spf13/cobra"

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
		return "Outcome(unknown)"
	}
}

// Run resolves args against the assembled root command tree, executes the
// resolved command (or routes a group/root to a help/listing outcome), and
// returns the categorized Outcome plus any error. It is the single dispatch
// boundary: the entrypoint calls Run instead of executing the tree directly
// (ADR-2). Run never emits an exit code and never terminates the process — the
// caller acts on the returned Outcome.
//
// Outcome derivation lives in T002; this seam currently maps any execution
// error to UsageError as a placeholder.
func Run(root *cobra.Command, args []string) (Outcome, error) {
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return UsageError, err
	}
	return Success, nil
}
