package apiclient

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// requestTimeout bounds a single API request so a hung connection fails loud
// rather than blocking forever (ADR-4). It is set on the built *http.Client, so
// it covers connection, any redirects, and reading the body. The value is an
// [ASSUMED] default — chosen as a generous ceiling for a healthy API call — and
// its eventual configurability is deferred (as 008 deferred the default URL).
// It is the only tunable this slice introduces.
const requestTimeout = 30 * time.Second

// Request is the code-free request descriptor a caller hands to Execute. It names
// the HTTP method, the path joined onto the connection context's base URL, and
// optional query parameters and body. It carries no credential and no base URL —
// identity rides the transport (007) and the endpoint was captured on the Client
// at NewClient — so the descriptor is purely the per-call shape.
type Request struct {
	// Method is the HTTP method (GET, POST, …). Required.
	Method string
	// Path is the request path joined onto the context's base URL. It is used as
	// given (008 pass-through-as-given); the join rule is consistent with 008's
	// contract. Required.
	Path string
	// Query is the optional query parameters appended to the URL. Nil for none.
	Query url.Values
	// Body is the optional request body. Nil for bodyless requests (GET, DELETE).
	Body io.Reader
}

// Client is the API-client request seam: a configured HTTP client built once from
// a ConnectionContext (009) and threaded to every request in an invocation
// (ADR-1). It holds the configured *http.Client — whose transport is 007's
// AuthTransport over the injected base, with the request timeout set — and the
// base URL captured from the context for request-time URL joins. Build it with
// NewClient (tests inject a fake base) or NewClientFromOS (production binds the
// real transport); call Execute per request.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds the request client once from the assembled ConnectionContext,
// wrapping the injected base transport. It is the base-URL fail-fast site (ADR-2):
// when ctx.BaseURLErr is set it returns that error verbatim and a nil *Client,
// building nothing and never inspecting the token — a context with no usable
// endpoint cannot produce a client. On the usable branch it builds an
// *http.Client whose transport is NewAuthTransport(base, replayThunk) — wrapping
// base in 007's AuthTransport over the replay thunk (009 ADR-2) that returns the
// context's already-resolved credential, never a fresh Discovery walk — with the
// request timeout set. The resolved base URL is captured on the *Client for
// request-time joins.
//
// The token fail-safe is NOT checked here; it stays in 007's AuthTransport,
// firing at send time on an absent or broken credential. 010 never reads
// ctx.Cred.Token itself.
//
// Precondition: base is required and must be non-nil; a nil base is a wiring bug
// and panics (fail-fast — no nil-default, per DECISIONS/PR #20). Defaulting to
// http.DefaultTransport would silently give this layer an untimed transport it
// must not own.
func NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error) {
	if ctx.BaseURLErr != nil {
		// Base-URL fail-fast: no usable endpoint, so build nothing and return the
		// carried error verbatim. The token is never inspected on this branch.
		return nil, ctx.BaseURLErr
	}
	if base == nil {
		panic("apiclient.NewClient: base must not be nil")
	}

	replay := func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr }
	httpClient := &http.Client{
		Timeout:   requestTimeout,
		Transport: NewAuthTransport(base, replay),
	}
	return &Client{httpClient: httpClient, baseURL: ctx.BaseURL.Value}, nil
}

// NewClientFromOS is the thin production seam over NewClient: it binds base to a
// real, configured base http.RoundTripper and delegates. It is intended to be
// called ONCE per invocation, paired with the single AssembleFromOS call — the
// client is built once and threaded to every request.
//
// The base transport is a clone of the standard transport's defaults (connection
// pooling, dial/TLS handshake timeouts) owned by this client, not the shared
// http.DefaultTransport — the per-request ceiling is the Client's Timeout.
func NewClientFromOS(ctx ConnectionContext) (*Client, error) {
	base := http.DefaultTransport.(*http.Transport).Clone()
	return NewClient(ctx, base)
}
