package apiclient

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// AuthTransport is the auth round-tripper: an http.RoundTripper that
// authenticates each outgoing request by attaching the X-Auth-Token header, and
// refuses to send when no usable credential exists. It wraps a base transport
// (supplied by Connection Configuration) and reaches it only on the
// authenticated branch — so no request is ever sent unauthenticated (the
// Fail-Safe guarantee, ADR-1).
//
// The credential is resolved once per invocation and cached: resolution is
// deterministic (005), so every request in one invocation carries the same
// identity and the filesystem walk is not repeated. The constructor name and
// the package are [ASSUMED] pending reconciliation with Connection
// Configuration.
type AuthTransport struct {
	base    http.RoundTripper
	resolve func() (auth.Resolution, error)

	once sync.Once
	res  auth.Resolution
	err  error
}

// NewAuthTransport builds an auth round-tripper that wraps base and consumes the
// injected resolver. Production binds Credential Discovery's resolver
// (auth.Resolve); tests bind a fake. 007 never resolves credentials itself
// (ADR-3) — the resolver is the only source of the token.
//
// Both base and resolve are required and must be non-nil; a nil argument is a
// wiring error and RoundTrip will panic. This is deliberate fail-fast: the
// transport must not substitute a default base (e.g. http.DefaultTransport),
// because Connection Configuration — not 007 — owns the HTTP transport, its base
// URL, and its timeouts (spec § Non-Behaviors; ADR-1). Silently defaulting would
// also degrade quietly where the spec requires failing loud (CONSTITUTION III).
func NewAuthTransport(base http.RoundTripper, resolve func() (auth.Resolution, error)) *AuthTransport {
	return &AuthTransport{base: base, resolve: resolve}
}

// ensureResolved consults the resolver exactly once for the transport's lifetime
// (one invocation) and caches the outcome. sync.Once makes the cache safe under
// the concurrent RoundTrip calls an http.Client may issue.
func (t *AuthTransport) ensureResolved() (auth.Resolution, error) {
	t.once.Do(func() {
		t.res, t.err = t.resolve()
	})
	return t.res, t.err
}

// RoundTrip authenticates and delegates. On a usable token it sets X-Auth-Token
// verbatim and calls the base transport, returning its result unchanged. On no
// credential or a credential error it returns the typed AuthError without ever
// reaching the base transport.
func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, authErr := authorize(t.ensureResolved())
	if authErr != nil {
		return nil, authErr
	}

	// An http.RoundTripper must not modify the request it is given; clone before
	// attaching the header so the caller's request is untouched (net/http
	// convention). The token is set verbatim — no trimming or re-encoding.
	clone := req.Clone(req.Context())
	clone.Header.Set(AuthHeaderName, token)
	return t.base.RoundTrip(clone)
}

// Identity is the operator-facing "acting as" view of the active credential. It
// carries only the safe-to-display Source and Path — never the token — so it can
// be printed in diagnostics without leaking the secret.
type Identity struct {
	Source auth.Source
	Path   string
}

func (i Identity) String() string {
	if i.Path != "" {
		return fmt.Sprintf("%s (%s)", i.Source, i.Path)
	}
	return i.Source.String()
}

// ActiveIdentity reports the identity the transport is acting as, resolving once
// (and caching) like RoundTrip does. It returns the same typed AuthError when no
// usable credential exists, so a caller reports an "acting as" line only when
// authentication actually succeeded. The token is never part of the result.
func (t *AuthTransport) ActiveIdentity() (Identity, *AuthError) {
	res, err := t.ensureResolved()
	if _, authErr := authorize(res, err); authErr != nil {
		return Identity{}, authErr
	}
	return Identity{Source: res.Source, Path: res.Path}, nil
}

// Compile-time assurance that AuthTransport satisfies http.RoundTripper, so
// Connection Configuration can compose it into an http.Client.
var _ http.RoundTripper = (*AuthTransport)(nil)
