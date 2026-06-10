package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// commandUsageError marks an error returned by a resolved command's own action
// as a usage problem the operator can fix (e.g. "no token to store",
// "credential already exists — pass --overwrite") rather than a runtime
// failure. A runnable command returns it (wrapped) when the *invocation* is at
// fault even though the action itself ran far enough to detect it; dispatch
// classifies it as UsageError so Exit-Code Convention maps it to code 2.
//
// It is the single sanctioned way for a command's action to yield a UsageError:
// every other UsageError source is cobra-level (unknown command/flag, an
// arg-validator rejection). Introduced with the first command that produces
// command-originated usage errors (Credential Storage 006).
type commandUsageError struct{ err error }

func (e *commandUsageError) Error() string { return e.err.Error() }
func (e *commandUsageError) Unwrap() error { return e.err }

// outcomeError carries an explicit operational Outcome on dispatch's error
// channel, for the categories dispatch cannot re-derive from an untyped error.
// commandUsageError handles the UsageError case; outcomeError generalizes it to
// the operational categories a resolved action produces (APIError,
// NetworkUnavailable) — introduced with the first command that classifies API
// client outcomes (Identity Read 011). Dispatch unwraps it to return the carried
// Outcome verbatim, so Exit-Code Convention maps it to the right code (3/6)
// rather than collapsing every action failure to RuntimeError(1).
//
// The command writes its own controlled, token-free message before returning
// this (it sets SilenceErrors), so dispatch only reads the category — it never
// re-renders. Success and UsageError keep their existing channels (nil and
// *commandUsageError); RuntimeError still travels as a bare error.
type outcomeError struct {
	outcome Outcome
	err     error
}

func (e *outcomeError) Error() string { return e.err.Error() }
func (e *outcomeError) Unwrap() error { return e.err }

// Outcome is the code-free classification dispatch assigns to an invocation.
// It names *what kind* of outcome occurred — never a process exit code (that
// is Exit-Code Convention's concern) and never a rendered message (Help &
// Version's). Exit-Code Convention (004) is the consumer that maps this
// category to a process code via ExitCode (exitcode.go) without re-deriving it
// from an untyped error (ADR-1).
//
// The category named outcomes with a producer today: Success, UsageError, and
// RuntimeError were the original three; Identity Read (011) is the first
// consuming command, so it adds the operational categories its API-client
// errors produce — NetworkUnavailable and APIError (codes 6 and 3, which 004
// reserved). The remaining reserved categories (permission/rate-limit, codes
// 4/5) gain an Outcome value when their producer lands (API Error Extraction
// 015 / Rate-Limit Handling 017), splitting APIError without renumbering.
type Outcome int

const (
	// Success means a command was routed and dispatched: it ran and its action
	// returned no error, or a group/root resolved to a help/listing outcome.
	Success Outcome = iota
	// UsageError means the invocation did not name a valid command, or carried
	// an unknown flag / unexpected argument — nothing ran, or running was
	// refused.
	UsageError
	// RuntimeError means a resolved command's own action returned an error: the
	// command ran but did not succeed. Distinct from UsageError (nothing ran)
	// and from Success (ran cleanly). Exit-Code Convention maps it to the
	// catch-all internal-error code 1 (ADR-3); the error itself still travels
	// via Run's error return.
	RuntimeError
	// NetworkUnavailable means the API could not be reached at the wire
	// (connection/DNS/TLS/timeout). Produced by the read surface classifying an
	// *apiclient.TransportError (011, ADR-3/4); Exit-Code Convention maps it to
	// the reserved code 6.
	NetworkUnavailable
	// APIError means the API answered with a non-2xx status. It is the generic,
	// uninterpreted "general API error" bucket — produced by classifying an
	// *apiclient.ResponseError (011, ADR-4) whose status is neither 401/403 nor
	// 429. Exit-Code Convention maps it to the reserved code 3.
	APIError
	// PermissionError means the API rejected the call as an auth/membership
	// failure (HTTP 401 or 403). Produced by API Error Extraction (015) splitting
	// APIError on the status the *apiclient.ProblemError carries (ADR-3);
	// Exit-Code Convention maps it to the reserved code 4. (403 plan-limit
	// messaging stays with Unsignalled Plan Limits — 015 carries it generically.)
	PermissionError
	// RateLimited means the API rejected the call for exceeding the rate limit
	// (HTTP 429). Produced by API Error Extraction (015) splitting APIError on the
	// status (ADR-3); Exit-Code Convention maps it to the reserved code 5. 015
	// only classifies the 429 — Rate-Limit Handling (017) owns the retry/backoff
	// above the Execute seam; 015 never sleeps or retries.
	RateLimited
)

// String renders the category name for legibility in logs and test failures.
func (o Outcome) String() string {
	switch o {
	case Success:
		return "Success"
	case UsageError:
		return "UsageError"
	case RuntimeError:
		return "RuntimeError"
	case NetworkUnavailable:
		return "NetworkUnavailable"
	case APIError:
		return "APIError"
	case PermissionError:
		return "PermissionError"
	case RateLimited:
		return "RateLimited"
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
//     the command ran but failed, so the category is RuntimeError and the error
//     is returned (Exit-Code Convention maps RuntimeError to code 1, ADR-3).
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
		// Unknown or malformed flag: the resolved command did not run. A command
		// that silenced cobra's own error output (SilenceErrors) printed nothing —
		// surface the cause here so the operator isn't left with a bare exit code.
		// This is symmetric with the Args-validator branch below; commands that let
		// cobra print keep their default output, so this never double-prints. (Role
		// Policies' two-command split, 034, is the first to reject a list-only flag
		// purely at this cobra level on a SilenceErrors leaf — see LEARNINGS.)
		if executed != nil && executed.SilenceErrors {
			fmt.Fprintf(executed.ErrOrStderr(), "Error: %s\nRun '%s --help' for usage.\n", err, executed.CommandPath())
		}
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
		// unexpected argument (a usage error — the action never ran).
		if err != nil && executed != nil && executed.ValidateArgs(executed.Flags().Args()) != nil {
			// The command's Args validator rejected the input before the action
			// ran. A command that silenced cobra's own error output (SilenceErrors)
			// printed nothing — surface the cause here so the operator isn't left
			// with a bare exit code. Commands that let cobra print keep their
			// default output, so this never double-prints. The Args validator is
			// side-effect-free, so re-checking it above is safe.
			if executed.SilenceErrors {
				fmt.Fprintf(executed.ErrOrStderr(), "Error: %s\nRun '%s --help' for usage.\n", err, executed.CommandPath())
			}
			return UsageError, err
		}
		// The validator passed. A resolved action may still classify its own
		// error as a usage problem by returning a *commandUsageError (e.g. "no
		// token to store") — that is a UsageError (→ code 2). Any other error is
		// the action's own runtime failure (RuntimeError → code 1, ADR-3); no
		// error is a clean run.
		if err != nil {
			var ue *commandUsageError
			if errors.As(err, &ue) {
				return UsageError, err
			}
			// An action may classify its own failure into an operational category
			// dispatch cannot re-derive (APIError, NetworkUnavailable) by returning
			// a *outcomeError; honor the carried category so Exit-Code Convention
			// maps it (3/6) rather than the RuntimeError(1) catch-all.
			var oe *outcomeError
			if errors.As(err, &oe) {
				return oe.outcome, err
			}
			return RuntimeError, err
		}
		return Success, nil
	}
}
