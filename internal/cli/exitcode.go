package cli

// The published, frozen process-exit convention: the codes a caller reads in
// $? after any glassfrog invocation (interface-cli.md). These constants are the
// single source of truth and are pinned by exitcode_test.go — a renumber breaks
// loudly there, so it can never happen silently.
//
// Per ADR-2 the full 0–6 convention is published now, but the live Outcome enum
// (dispatch.go) carries a category only once its producer exists. So codes 3–6
// are reserved for the future API client: when it lands it will add
// APIError/PermissionError/RateLimited/NetworkUnavailable to the Outcome enum
// and the matching cases to ExitCode below — at this one registry site, each
// taking its already-reserved code. Existing codes are never renumbered
// (interface-cli.md "Extension"). The asymmetry — codes exist as constants
// before their categories exist as enum values — is intentional, not
// incomplete.
const (
	codeSuccess            = 0 // a command completed, or help/listing/--version resolved
	codeInternalError      = 1 // safety net: a resolved action failed, a panic, or any unmapped/future category
	codeUsageError         = 2 // unknown command, unknown/missing flag, or unexpected positional argument
	codeAPIError           = 3 // reserved — future API client: a general API error
	codePermissionError    = 4 // reserved — future API client: auth/membership rejection
	codeRateLimited        = 5 // reserved — future API client: rate limit exceeded
	codeNetworkUnavailable = 6 // reserved — future API client: the API could not be reached
)

// ExitCode maps a code-free Outcome category (dispatch.go) to its process exit
// code. It is the single registry binding a category to a code — a pure lookup
// that never classifies, renders text, retries, or inspects an error (ADR-1):
// producers classify into an Outcome, ExitCode only maps.
//
// The default arm returns codeInternalError (1) so any unmapped or future
// category can never accidentally exit 0 (Fail Safe, CONSTITUTION III). Only the
// categories with producers today are mapped explicitly; RuntimeError gains its
// case when its producer is reclassified (T002), and the operational categories
// (codes 3–6) gain theirs with the future API client.
func ExitCode(o Outcome) int {
	switch o {
	case Success:
		return codeSuccess
	case UsageError:
		return codeUsageError
	default:
		return codeInternalError
	}
}
