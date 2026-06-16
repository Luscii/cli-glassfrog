package cli

// The published, frozen process-exit convention: the codes a caller reads in
// $? after any glassfrog invocation (interface-cli.md). These constants are the
// single source of truth and are pinned by exitcode_test.go — a renumber breaks
// loudly there, so it can never happen silently.
//
// Per ADR-2 the full 0–6 convention is published now, but the live Outcome enum
// (dispatch.go) carries a category only once its producer exists. Identity Read
// (011) is the first consuming command: it added APIError(3) and
// NetworkUnavailable(6) to the Outcome enum and their cases to ExitCode below.
// API Error Extraction (015) is the producer that filled codes 4 (permission)
// and 5 (rate-limited): it split APIError(3) into 401/403→permission(4) and
// 429→rate-limit(5) at this one registry site, each taking its already-reserved
// code, without renumbering existing codes (interface-cli.md "Extension").
// Rate-Limit Handling (017) owns the 429 retry/backoff above the Execute seam;
// 015 only classifies the 429 into RateLimited(5).
//
// Stale-Write Surfacing (054) adds codeStaleWrite(7) — the first code beyond the
// originally-published 0–6 band — by branching the 412 out of the generic
// APIError bucket at the same status-driven classifier. It is exactly the
// extension mechanism 004 designed (new category → single registry site → new
// previously-unused code → never renumber), so no existing code changes meaning.
const (
	codeSuccess            = 0 // a command completed, or help/listing/--version resolved
	codeInternalError      = 1 // safety net: a resolved action failed, a panic, or any unmapped/future category
	codeUsageError         = 2 // unknown command, unknown/missing flag, or unexpected positional argument
	codeAPIError           = 3 // a generic non-2xx API response (Identity Read 011 onward)
	codePermissionError    = 4 // an API auth/membership rejection: 401/403 (API Error Extraction 015 onward)
	codeRateLimited        = 5 // the API rate limit was exceeded: 429 (API Error Extraction 015 onward)
	codeNetworkUnavailable = 6 // the API could not be reached at the wire (Identity Read 011 onward)
	codeStaleWrite         = 7 // a guarded write was refused: the resource changed since it was read, 412 (Stale-Write Surfacing 054 onward)
)

// ExitCode maps a code-free Outcome category (dispatch.go) to its process exit
// code. It is the single registry binding a category to a code — a pure lookup
// that never classifies, renders text, retries, or inspects an error (ADR-1):
// producers classify into an Outcome, ExitCode only maps.
//
// The default arm returns codeInternalError (1) so any unmapped or future
// category can never accidentally exit 0 (Fail Safe, CONSTITUTION III). Only the
// categories with producers today are mapped explicitly; the operational
// categories (codes 3–6) gain their cases with the future API client.
func ExitCode(o Outcome) int {
	switch o {
	case Success:
		return codeSuccess
	case UsageError:
		return codeUsageError
	case RuntimeError:
		return codeInternalError
	case APIError:
		return codeAPIError
	case PermissionError:
		return codePermissionError
	case RateLimited:
		return codeRateLimited
	case NetworkUnavailable:
		return codeNetworkUnavailable
	case StaleWrite:
		return codeStaleWrite
	default:
		return codeInternalError
	}
}
