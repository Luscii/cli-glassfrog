package cli

import (
	"errors"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
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
//   - *apiclient.ResponseError         → APIError (a generic non-2xx; 015/017 refine it later)
//   - *apiclient.DecodeError           → RuntimeError (a 2xx body that would not parse — a contract failure)
//   - a base-URL error (*apiclient.BaseURLError or an internal/rcfile read/format
//     error surfaced as ctx.BaseURLErr) → UsageError (a correctable endpoint input)
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

	var responseErr *apiclient.ResponseError
	if errors.As(err, &responseErr) {
		return APIError
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

	// Fail-safe: an unrecognized error is never silently a success. Exit-Code
	// Convention maps RuntimeError to the catch-all internal-error code 1
	// (CONSTITUTION III).
	return RuntimeError
}
