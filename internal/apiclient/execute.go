package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Response is Request Execution's success value (a 2xx outcome). It carries the
// status code and the response headers so the sibling capabilities can build on
// the same seam — Rate-Limit Handling (017) reads rate-limit headers off the
// non-2xx path. Pagination (016) does NOT read a paging header: the v5 API carries
// paging in the response BODY at meta.pagination, which the walker decodes via an
// enveloped target (glassfrog.Page[T]) — Response exposes only status+headers. The
// decoded body is written into the caller's out target, not stored here; when out
// is nil the body is drained and closed without decoding.
type Response struct {
	// StatusCode is the 2xx status the API returned.
	StatusCode int
	// Header is the response headers, exposed for the sibling capabilities.
	Header http.Header
}

// Version returns the resource version captured from this read response — the
// ETag header, verbatim — for a later guarded write (Guarded Writes, 053) to
// send back as If-Match. The value is returned exactly as the server stated it:
// no unquoting, no weak-validator ("W/…") prefix stripping, no normalization,
// because an If-Match precondition must echo the server's token byte-for-byte or
// risk a spurious 412. When the response carries no ETag, it returns "" — the
// "no version captured" sentinel, indistinguishable from an empty ETag (neither
// is a usable precondition). Header lookup is case-insensitive (net/http
// canonicalizes header names), so ETag/Etag/etag all match.
//
// Version is purely derived from the Header the Response already holds: it stores
// nothing, mutates nothing, sends nothing, and renders nothing. It is the
// read-side capture seam Guarded Writes (053) — the intended consumer — will call
// on its in-process pre-write read; no existing call site invokes it yet (ADR-2).
func (r *Response) Version() string {
	return r.Header.Get("ETag")
}

// TransportError is the typed, code-free failure for a request that could not
// reach the API or complete at the wire — connection refused, DNS failure, TLS
// failure, or the request timeout elapsing. It wraps the underlying network
// cause (host/port/network-level, never the token) and is discriminable via
// errors.As. The consuming command (004), not 010, maps it to an exit code
// (transport → network-unavailable).
type TransportError struct {
	cause error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("request failed: %v", e.cause)
}

// Unwrap exposes the underlying network cause for errors.As / errors.Is.
func (e *TransportError) Unwrap() error { return e.cause }

// ResponseError is the typed, code-free outcome for a non-2xx response. It is
// deliberately GENERIC and uncategorized — it carries the status, headers, and
// raw body so API Error Extraction (015) can extract API detail and Rate-Limit
// Handling (017) can read a 429's rate-limit headers off the same value. 010 does
// not classify it by failure kind and never decodes the body into the caller's
// target.
type ResponseError struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("the API returned a non-2xx response: status %d", e.StatusCode)
}

// DecodeError is the typed, code-free outcome for a 2xx response whose body could
// not be decoded into the supplied target. It is surfaced loud rather than
// returning a zero-valued target (CONSTITUTION III). It carries the status and
// wraps the decode cause (a parse failure, never the token).
type DecodeError struct {
	StatusCode int
	cause      error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("decoding the %d response body failed: %v", e.StatusCode, e.cause)
}

// Unwrap exposes the underlying decode cause for errors.As / errors.Is.
func (e *DecodeError) Unwrap() error { return e.cause }

// Execute is the single send seam every endpoint command calls through (ADR-1).
// It joins the Client's captured base URL with req.Path (008 pass-through-as-
// given), appends req.Query, builds the *http.Request (method, body, reqCtx for
// cancellation), makes EXACTLY ONE Do call (no retry — ADR-4), and maps the
// result to a *Response (2xx) or one of the typed code-free errors:
//
//   - a Do error: 007's *AuthError is discriminated via errors.As FIRST and
//     returned unchanged (never mislabeled as a transport failure); any other
//     error is wrapped as *TransportError;
//   - a 2xx response: a *Response{StatusCode, Header}; when out != nil the body is
//     decoded into out (a decode failure → *DecodeError), when out == nil the body
//     is drained without decoding;
//   - a non-2xx response: a generic *ResponseError{StatusCode, Header, Body} — the
//     body is never decoded into out and the error is never classified.
//
// The response body is always closed on every branch (no fd/connection leak). 010
// never reads the token — identity rides 007's AuthTransport, wired at NewClient.
func (c *Client) Execute(reqCtx context.Context, req Request, out any) (*Response, error) {
	urlStr, err := buildURL(c.baseURL, req.Path, req.Query)
	if err != nil {
		// The base URL is 008-validated as parseable, so this is a guard: a base
		// that won't parse is a build-time failure, not a wire failure.
		return nil, &TransportError{cause: err}
	}

	httpReq, err := http.NewRequestWithContext(reqCtx, req.Method, urlStr, req.Body)
	if err != nil {
		// A malformed method/URL is a build-time wiring failure, not a wire failure;
		// surface it as a transport error rather than attempting a doomed send.
		return nil, &TransportError{cause: err}
	}

	if req.ContentType != "" {
		// Set the body's media type only when the caller supplied one (042 ADR-1):
		// a bodyless read leaves ContentType empty and the request carries no
		// Content-Type header (unchanged from every landed GET). 007's AuthTransport
		// owns only X-Auth-Token, so the content type is set here on the built
		// request, before Do, never in the auth layer.
		httpReq.Header.Set("Content-Type", req.ContentType)
	}

	if req.IfMatch != "" {
		// Set the precondition only when the caller supplied a version (mirrors the
		// ContentType block above, 042 ADR-1). The value is sent verbatim — no
		// quoting, unquoting, weak-validator ("W/…") handling, or normalization — so
		// it echoes the server's token (captured by Response.Version(), 052)
		// byte-for-byte or risks a spurious 412. Method-agnostic: depends only on the
		// field, so a DELETE is guarded like a PUT/PATCH. 007's AuthTransport owns
		// only X-Auth-Token, so If-Match is set here on the built request, before Do.
		httpReq.Header.Set("If-Match", req.IfMatch)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Discriminate 007's fail-safe BEFORE wrapping: a no-/broken-credential
		// AuthError and a genuine wire failure both surface as the Do error, and the
		// AuthError must keep its taxonomy (ADR-4).
		var authErr *AuthError
		if errors.As(err, &authErr) {
			return nil, authErr
		}
		return nil, &TransportError{cause: err}
	}
	// One close covers every branch below — non-2xx and decode-error included.
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Non-2xx: carry the status, headers, and raw body generically. Never
		// decoded into out, never classified by kind (015/017 refine it).
		body, _ := io.ReadAll(resp.Body)
		return nil, &ResponseError{StatusCode: resp.StatusCode, Header: resp.Header, Body: body}
	}

	response := &Response{StatusCode: resp.StatusCode, Header: resp.Header}
	if out == nil {
		// No decode target: drain the body so the connection can be reused, then the
		// deferred Close runs.
		_, _ = io.Copy(io.Discard, resp.Body)
		return response, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		// A 2xx the API contract says should parse but did not — fail loud.
		return nil, &DecodeError{StatusCode: resp.StatusCode, cause: err}
	}
	return response, nil
}

// buildURL joins the captured base URL with the request path and query as 008
// resolved the base (pass-through-as-given). It parses the base so a base that
// carries its own query string or fragment is handled correctly rather than
// mangled: req.Path is appended onto the base's path component only (a single
// slash between them, no normalization), req.Query is merged with any query the
// base already carried, and the base's fragment is preserved. An empty path
// leaves the base path unchanged. The base is 008-validated as a parseable
// absolute URL; a parse failure is returned as a guard.
func buildURL(base, path string, query url.Values) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if path != "" {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	if len(query) > 0 {
		merged := u.Query() // the base's own query params, if any
		for key, vals := range query {
			for _, v := range vals {
				merged.Add(key, v)
			}
		}
		u.RawQuery = merged.Encode()
	}
	return u.String(), nil
}
