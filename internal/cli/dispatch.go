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
//     swallowed token from the group's leftover positional args itself.
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
		// subcommand that cobra silently swallowed — surface it.
		if leftover := executed.Flags().Args(); len(leftover) > 0 {
			return UsageError, fmt.Errorf("unknown command %q for %q", leftover[0], executed.CommandPath())
		}
		return Success, nil
	default:
		// A command ran. Its action's error (if any) travels via err; the
		// category stays Success (RuntimeError deferred to 004).
		return Success, err
	}
}
