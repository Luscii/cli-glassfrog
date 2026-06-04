// Package apiclient is the API-client/transport layer of the Glassfrog CLI: the
// point where the resolved identity meets the wire. Request Authentication (007)
// lives here — it decorates each outgoing API request with the X-Auth-Token
// header for the org + person the token is scoped to, and refuses to send when
// no usable credential exists, so no request can ever leave the process
// unauthenticated.
//
// It consumes Credential Discovery's Resolution (internal/auth, 005) through an
// injected resolver seam and never resolves credentials itself. It owns neither
// the HTTP client (base URL, timeouts, retries, response parsing) — that is
// Connection Configuration, a sibling capability — nor any cobra command; the
// command that triggers API calls is a future spec.
//
// Secret hygiene is a package invariant carried from internal/auth: the token
// appears only as the X-Auth-Token header value. It is never logged and never
// placed in an AuthError — a CredentialError wraps Discovery's path-only error,
// which names the offending file and never the token.
package apiclient

import (
	"fmt"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// AuthHeaderName is the request header carrying the Glassfrog API credential.
// It is fixed by the Glassfrog v5 API scheme (a PROJECT constraint, not
// provisional) and centralized here as the single source of truth.
const AuthHeaderName = "X-Auth-Token"

// AuthErrorKind discriminates the two ways Request Authentication can fail to
// authenticate a request. They are kept distinct so the consuming command can
// give different guidance ("no token — store one" vs "the credentials file is
// broken at <path>") and map each to an exit code (007 decides none).
type AuthErrorKind int

const (
	// NoCredentials means no usable credential exists anywhere — Discovery
	// reported Source None with no error. A normal, expected precondition.
	NoCredentials AuthErrorKind = iota
	// CredentialError means a credentials file exists but could not be read or
	// parsed — Discovery returned a typed read/format error naming the file.
	CredentialError
)

func (k AuthErrorKind) String() string {
	switch k {
	case NoCredentials:
		return "no-credentials"
	case CredentialError:
		return "credential-error"
	default:
		return "unknown"
	}
}

// AuthError is the typed, code-free failure Request Authentication returns when
// it cannot authenticate a request. It carries a discriminable Kind; the
// consuming command (not 007) maps it to an exit code. For CredentialError it
// wraps Discovery's read/format error — unwrappable via errors.As / Unwrap —
// which names only the path, never the token. The token value never appears in
// Error(): NoCredentials carries no cause, and the CredentialError cause is
// path-only by the internal/auth contract.
type AuthError struct {
	Kind  AuthErrorKind
	cause error // the wrapped Discovery error for CredentialError; nil otherwise
}

func (e *AuthError) Error() string {
	switch e.Kind {
	case NoCredentials:
		return "cannot authenticate: no credentials found"
	case CredentialError:
		return fmt.Sprintf("cannot authenticate: %v", e.cause)
	default:
		return "cannot authenticate"
	}
}

// Unwrap exposes the wrapped Discovery error so callers can errors.As it back to
// the concrete *auth.ReadError / *auth.FormatError. It is nil for NoCredentials.
func (e *AuthError) Unwrap() error { return e.cause }

// authorize is the pure mapping from Credential Discovery's (Resolution, error)
// outcome to either a usable token or a typed AuthError. It performs no I/O and
// reads no environment or filesystem — it only classifies what Discovery
// returned (ADR-4):
//
//   - a resolver error → CredentialError wrapping it (the request must not be
//     sent); the error wins even if a token is somehow present;
//   - Source None → NoCredentials (no usable credential);
//   - Source Environment / File → the token, with no AuthError.
func authorize(res auth.Resolution, err error) (token string, authErr *AuthError) {
	if err != nil {
		return "", &AuthError{Kind: CredentialError, cause: err}
	}
	if res.Source == auth.SourceNone {
		return "", &AuthError{Kind: NoCredentials}
	}
	return res.Token, nil
}
