package cli

import (
	"errors"
	"net/http"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/output"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// classifyClientError is the single errors.As chain the read surface uses to
// turn an API-client error (010's typed, code-free errors, plus the base-URL
// error surfaced at client construction) into a code-free Outcome category. It
// is produced by the first consuming command (Identity Read 011, ADR-3) and
// reused verbatim by every later read (012–017) so the error→category mapping
// lives in exactly one place and never drifts. It never inspects an exit code,
// renders a message, or touches the token — it only maps.
//
// The mapping (plan ADR-4, interface-spec Error Communication):
//
//   - nil                              → Success
//   - *apiclient.AuthError{NoCredentials}   → UsageError (not authenticated — a correctable invocation)
//   - *apiclient.AuthError{CredentialError} → RuntimeError (a malformed .glassfrogrc — an internal failure)
//   - *apiclient.TransportError        → NetworkUnavailable (the wire failed)
//   - *apiclient.ProblemError / *apiclient.ResponseError → branch on the status
//     (API Error Extraction 015, ADR-3): 401/403 → PermissionError (code 4),
//     429 → RateLimited (code 5), everything else → APIError (the generic
//     non-2xx, code 3)
//   - *apiclient.DecodeError           → RuntimeError (a 2xx body that would not parse — a contract failure)
//   - a base-URL error (*apiclient.BaseURLError or an internal/rcfile read/format
//     error surfaced as ctx.BaseURLErr) → UsageError (a correctable endpoint input)
//   - *output.FormatError (an invalid --output/GLASSFROG_OUTPUT/.glassfrogrc output
//     selector, 020) → UsageError (a correctable selection input)
//   - anything else                    → RuntimeError (the fail-safe; never Success)
//
// Discrimination order matters: *AuthError is matched BEFORE *TransportError
// (007's fail-safe surfaces as the Do error and must not be mislabelled
// transport — 010's discipline), and BEFORE the rcfile arms (a CredentialError
// wraps an rcfile error via Unwrap, so an unmatched rcfile arm would otherwise
// catch it as a base-URL UsageError). An rcfile error reaches the rcfile arms
// only unwrapped — i.e. via the base-URL path from NewClient — so mapping it to
// UsageError there is correct.
func classifyClientError(err error) Outcome {
	if err == nil {
		return Success
	}

	var authErr *apiclient.AuthError
	if errors.As(err, &authErr) {
		if authErr.Kind == apiclient.NoCredentials {
			return UsageError
		}
		// CredentialError: a credentials file exists but could not be read/parsed.
		return RuntimeError
	}

	var transportErr *apiclient.TransportError
	if errors.As(err, &transportErr) {
		return NetworkUnavailable
	}

	// A non-2xx, whether still the bare *ResponseError or refined into a
	// *ProblemError (which wraps it), matches here — errors.As reaches the wrapped
	// value and its StatusCode is the authoritative HTTP status (ExtractProblem
	// never promotes a disagreeing body status). Branching on the status IN this
	// one arm — rather than a separate *ProblemError arm before a bare
	// *ResponseError arm — sidesteps the wrapping discrimination-order hazard: the
	// switch's default IS the generic-APIError bucket, so a 401/403/429 can never
	// fall through to it (plan Risks). API Error Extraction (015) split (ADR-3).
	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		switch responseErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return PermissionError
		case http.StatusTooManyRequests:
			return RateLimited
		default:
			return APIError
		}
	}

	var decodeErr *apiclient.DecodeError
	if errors.As(err, &decodeErr) {
		return RuntimeError
	}

	// Base-URL configuration error from client construction: a malformed
	// configured endpoint (*BaseURLError) or an unreadable/malformed .glassfrogrc
	// the base-URL resolver surfaced (an rcfile error). Both are correctable
	// operator input → UsageError.
	var baseURLErr *apiclient.BaseURLError
	if errors.As(err, &baseURLErr) {
		return UsageError
	}
	var rcReadErr *rcfile.ReadError
	if errors.As(err, &rcReadErr) {
		return UsageError
	}
	var rcFormatErr *rcfile.FormatError
	if errors.As(err, &rcFormatErr) {
		return UsageError
	}

	// Output Format Selection (020, ADR-4): a present-but-invalid --output /
	// GLASSFROG_OUTPUT / .glassfrogrc output value, surfaced as *output.FormatError
	// before any request. Symmetric with the base-URL arms above — a correctable
	// selection input → UsageError, so an invalid selector's category and message
	// agree and it exits with the conventional usage code (no new exit code).
	var formatErr *output.FormatError
	if errors.As(err, &formatErr) {
		return UsageError
	}

	// Fail-safe: an unrecognized error is never silently a success. Exit-Code
	// Convention maps RuntimeError to the catch-all internal-error code 1
	// (CONSTITUTION III).
	return RuntimeError
}
