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
	// ContentType is the optional media type of Body, set as the request's
	// Content-Type header by Execute only when non-empty (042 ADR-1). Empty for
	// every bodyless read (the landed GETs), so their outbound request carries no
	// Content-Type header and stays byte-identical; the first write sets it to
	// "application/json" so the API parses the JSON body rather than ignoring it
	// (a silent 422 the codebase designs against). A narrow field, not a general
	// Header bag — generalize when a second header (If-Match, deferred) has a real
	// consumer.
	ContentType string
	// IfMatch is the optional resource version sent as the request's If-Match
	// precondition header by Execute, only when non-empty (mirrors ContentType,
	// 042 ADR-1). Empty for every request that is not guarded — the landed reads
	// and writes leave it "", so their outbound request carries no If-Match header
	// and stays byte-identical; a guarded write sets it to the version a prior
	// single-resource read captured (apiclient.Response.Version(), 052) so the
	// server refuses a stale write (412 Precondition Failed) instead of overwriting
	// it last-write-wins. The value is sent verbatim — no quoting, unquoting,
	// weak-validator ("W/…") handling, or normalization — because the precondition
	// must echo the server's token byte-for-byte or risk a spurious 412. The set is
	// method-agnostic: a guarded DELETE (Tension Discard) is guarded like a
	// PUT/PATCH; the caller populates IfMatch only on requests it intends to guard.
	// A narrow field, not a general Header bag — this is the deferred If-Match
	// consumer 042 named; generalize only when a second request header lands. The
	// intended consumers are each write command's own read-then-write retrofit
	// (Tension Update/Discard, Proposal write-flow); 053 wires none of them.
	IfMatch string
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
// must not own. The nil check runs first, unconditionally — a wiring bug must
// fail the same way whether or not the context also carries a base-URL error.
func NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error) {
	if base == nil {
		// Argument precondition, checked before any other branch so the panic is
		// deterministic and a nil transport is never masked by a base-URL error.
		panic("apiclient.NewClient: base must not be nil")
	}
	if ctx.BaseURLErr != nil {
		// Base-URL fail-fast: no usable endpoint, so build nothing and return the
		// carried error verbatim. The token is never inspected on this branch.
		return nil, ctx.BaseURLErr
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
// http.DefaultTransport — the per-request ceiling is the Client's Timeout. If a
// dependency has replaced http.DefaultTransport with a non-*http.Transport, the
// type assertion would panic, so fall back to using it directly rather than
// failing a legitimate construction; the Client's Timeout still bounds it.
func NewClientFromOS(ctx ConnectionContext) (*Client, error) {
	base := http.DefaultTransport
	if t, ok := base.(*http.Transport); ok {
		base = t.Clone()
	}
	return NewClient(ctx, base)
}
