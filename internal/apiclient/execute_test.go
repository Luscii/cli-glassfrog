package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// trackingBody is a response body that records whether it was closed, so a test
// can pin the always-close-the-body guarantee on every branch.
type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

// respondingBase is a fake base http.RoundTripper that returns a canned response
// built per request, counting its calls (the one-attempt tripwire). It records
// the X-Auth-Token it observed so a test can confirm the transport authenticated
// the request.
type respondingBase struct {
	calls          int
	gotToken       string
	gotContentType string
	contentTypeSet bool
	status         int
	header         http.Header
	body           *trackingBody
}

func (b *respondingBase) RoundTrip(req *http.Request) (*http.Response, error) {
	b.calls++
	b.gotToken = req.Header.Get(AuthHeaderName)
	// Record presence-and-value so a test can distinguish "no Content-Type header"
	// (the bodyless reads) from "set to application/json" (the write, 042 ADR-1).
	_, b.contentTypeSet = req.Header["Content-Type"]
	b.gotContentType = req.Header.Get("Content-Type")
	header := b.header
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{
		StatusCode: b.status,
		Header:     header,
		Body:       b.body,
	}, nil
}

// erroringBase is a fake base that fails at the wire, returning a network-level
// error (no response), like a connection refusal.
type erroringBase struct {
	calls int
	err   error
}

func (b *erroringBase) RoundTrip(*http.Request) (*http.Response, error) {
	b.calls++
	return nil, b.err
}

// blockingBase blocks until the request context is cancelled, then returns its
// error — modelling a server that accepts the connection but never responds. The
// injected short deadline (the request context) is what fails the request.
type blockingBase struct {
	calls int
}

func (b *blockingBase) RoundTrip(req *http.Request) (*http.Response, error) {
	b.calls++
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func mustClient(t *testing.T, ctx ConnectionContext, base http.RoundTripper) *Client {
	t.Helper()
	c, err := NewClient(ctx, base)
	if err != nil {
		t.Fatalf("NewClient errored: %v", err)
	}
	return c
}

func bodyOf(s string) *trackingBody { return &trackingBody{Reader: strings.NewReader(s)} }

func TestExecute2xxDecodesIntoTarget(t *testing.T) {
	base := &respondingBase{status: 200, body: bodyOf(`{"id":"per_1","name":"Ada"}`)}
	client := mustClient(t, completeContext(secretToken), base)

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &out)
	if err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if out.ID != "per_1" || out.Name != "Ada" {
		t.Fatalf("decoded target = %+v, want id per_1 / name Ada", out)
	}
	if !base.body.closed {
		t.Fatal("response body was not closed on the 2xx-decode branch")
	}
}

func TestExecute2xxNoTargetSkipsDecode(t *testing.T) {
	base := &respondingBase{status: 204, body: bodyOf("")}
	client := mustClient(t, completeContext(secretToken), base)

	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/health"}, nil)
	if err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if resp.StatusCode != 204 {
		t.Fatalf("StatusCode = %d, want 204", resp.StatusCode)
	}
	if !base.body.closed {
		t.Fatal("response body was not closed on the 2xx-no-target branch")
	}
}

func TestExecute2xxUndecodableIsDecodeError(t *testing.T) {
	base := &respondingBase{status: 200, body: bodyOf(`not json at all`)}
	client := mustClient(t, completeContext(secretToken), base)

	var out struct {
		ID string `json:"id"`
	}
	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &out)
	if resp != nil {
		t.Fatalf("response = %v, want nil on a decode failure", resp)
	}
	var decErr *DecodeError
	if !errors.As(err, &decErr) {
		t.Fatalf("err = %v, want *DecodeError", err)
	}
	if decErr.StatusCode != 200 {
		t.Fatalf("DecodeError.StatusCode = %d, want 200", decErr.StatusCode)
	}
	if !base.body.closed {
		t.Fatal("response body was not closed on the decode-error branch")
	}
}

func TestExecuteNon2xxIsGenericResponseError(t *testing.T) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	base := &respondingBase{status: 403, header: header, body: bodyOf(`{"error":"forbidden"}`)}
	client := mustClient(t, completeContext(secretToken), base)

	var out struct {
		Error string `json:"error"`
	}
	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &out)
	if resp != nil {
		t.Fatalf("response = %v, want nil on a non-2xx", resp)
	}
	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("err = %v, want *ResponseError", err)
	}
	if respErr.StatusCode != 403 {
		t.Fatalf("ResponseError.StatusCode = %d, want 403", respErr.StatusCode)
	}
	if string(respErr.Body) != `{"error":"forbidden"}` {
		t.Fatalf("ResponseError.Body = %q, want the raw body", respErr.Body)
	}
	if respErr.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("ResponseError.Header missing Content-Type")
	}
	// The body must never be decoded into the success target.
	if out.Error != "" {
		t.Fatalf("the non-2xx body was decoded into the target: %+v", out)
	}
	if !base.body.closed {
		t.Fatal("response body was not closed on the non-2xx branch")
	}
}

func TestExecute429CarriesRateLimitHeaders(t *testing.T) {
	header := http.Header{
		"Retry-After":           []string{"60"},
		"X-Ratelimit-Remaining": []string{"0"},
	}
	base := &respondingBase{status: 429, header: header, body: bodyOf(`{"error":"rate limited"}`)}
	client := mustClient(t, completeContext(secretToken), base)

	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil)
	if resp != nil {
		t.Fatalf("response = %v, want nil on a 429", resp)
	}
	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("err = %v, want *ResponseError", err)
	}
	if respErr.StatusCode != 429 {
		t.Fatalf("ResponseError.StatusCode = %d, want 429", respErr.StatusCode)
	}
	if respErr.Header.Get("Retry-After") != "60" || respErr.Header.Get("X-Ratelimit-Remaining") != "0" {
		t.Fatalf("ResponseError did not carry the rate-limit headers: %v", respErr.Header)
	}
	// Exactly one attempt — no sleep, no backoff, no retry.
	if base.calls != 1 {
		t.Fatalf("base reached %d times on a 429, want exactly 1 (no retry)", base.calls)
	}
}

func TestExecuteWireFailureIsTransportError(t *testing.T) {
	base := &erroringBase{err: errors.New("dial tcp 127.0.0.1:443: connect: connection refused")}
	client := mustClient(t, completeContext(secretToken), base)

	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil)
	if resp != nil {
		t.Fatalf("response = %v, want nil on a wire failure", resp)
	}
	var transErr *TransportError
	if !errors.As(err, &transErr) {
		t.Fatalf("err = %v, want *TransportError", err)
	}
}

func TestExecutePropagatesAuthErrorNotTransportError(t *testing.T) {
	// A usable base URL but no token: the AuthTransport refuses at send time with
	// a typed *AuthError, which must propagate unchanged — never mislabeled as a
	// *TransportError — and no request may reach the base.
	ctx := ConnectionContext{
		BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceDefault},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	base := &respondingBase{status: 200, body: bodyOf("{}")}
	client := mustClient(t, ctx, base)

	resp, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil)
	if resp != nil {
		t.Fatalf("response = %v, want nil on the auth fail-safe", resp)
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %v, want the propagated *AuthError", err)
	}
	if authErr.Kind != NoCredentials {
		t.Fatalf("AuthError.Kind = %v, want NoCredentials", authErr.Kind)
	}
	var transErr *TransportError
	if errors.As(err, &transErr) {
		t.Fatal("the auth fail-safe was mislabeled as a *TransportError")
	}
	if base.calls != 0 {
		t.Fatalf("base reached %d times; no unauthenticated request may be sent", base.calls)
	}
}

func TestExecuteMakesExactlyOneAttempt(t *testing.T) {
	base := &respondingBase{status: 200, body: bodyOf("{}")}
	client := mustClient(t, completeContext(secretToken), base)

	var out map[string]any
	if _, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &out); err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if base.calls != 1 {
		t.Fatalf("base reached %d times, want exactly 1 (no retry)", base.calls)
	}
}

func TestExecuteHungConnectionTimesOutAsTransportError(t *testing.T) {
	base := &blockingBase{}
	client := mustClient(t, completeContext(secretToken), base)

	// A short injected deadline stands in for the request timeout, kept fast.
	reqCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := client.Execute(reqCtx, Request{Method: http.MethodGet, Path: "/me"}, nil)
	if resp != nil {
		t.Fatalf("response = %v, want nil on a hung connection", resp)
	}
	var transErr *TransportError
	if !errors.As(err, &transErr) {
		t.Fatalf("err = %v, want *TransportError on the timeout", err)
	}
	if base.calls != 1 {
		t.Fatalf("base reached %d times, want exactly 1 (no retry on timeout)", base.calls)
	}
}

func TestExecuteSendsAuthenticatedRequest(t *testing.T) {
	base := &respondingBase{status: 200, body: bodyOf("{}")}
	client := mustClient(t, completeContext(secretToken), base)

	if _, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil); err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if base.gotToken != secretToken {
		t.Fatalf("base saw X-Auth-Token = %q, want %q attached by the transport", base.gotToken, secretToken)
	}
}

func TestExecuteErrorsNeverRenderTheToken(t *testing.T) {
	const token = "gf_live_secret123"
	ctx := completeContext(token)

	// Across the error-producing branches, no rendered error may contain the token.
	cases := map[string]func() error{
		"transport": func() error {
			base := &erroringBase{err: errors.New("dial tcp: connection refused")}
			_, err := mustClient(t, ctx, base).Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil)
			return err
		},
		"response": func() error {
			base := &respondingBase{status: 500, body: bodyOf("boom")}
			_, err := mustClient(t, ctx, base).Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil)
			return err
		},
		"decode": func() error {
			base := &respondingBase{status: 200, body: bodyOf("not json")}
			var out struct {
				ID string `json:"id"`
			}
			_, err := mustClient(t, ctx, base).Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, &out)
			return err
		},
	}
	for name, run := range cases {
		err := run()
		if err == nil {
			t.Fatalf("%s: expected an error to inspect", name)
		}
		if strings.Contains(err.Error(), token) {
			t.Fatalf("%s: the token leaked into the error: %q", name, err.Error())
		}
	}
}

func TestExecuteJoinsURLWithQuery(t *testing.T) {
	// A capturing base records the URL Execute built, confirming the base+path
	// join and the appended query.
	var gotURL string
	base := capturingBase(func(req *http.Request) { gotURL = req.URL.String() })
	client := mustClient(t, completeContext(secretToken), base)

	_, err := client.Execute(
		context.Background(),
		Request{Method: http.MethodGet, Path: "/roles", Query: map[string][]string{"page": {"2"}}},
		nil,
	)
	if err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	want := "https://glassfrog.com/api/v5/roles?page=2"
	if gotURL != want {
		t.Fatalf("request URL = %q, want %q", gotURL, want)
	}
}

func TestExecuteJoinsURLPreservingBaseQueryAndFragment(t *testing.T) {
	// A base that carries its own query string and fragment is accepted by
	// isUsableURL (scheme+host only). The path must be appended onto the base's
	// path component, req.Query merged with the base's existing query, and the
	// fragment preserved — not mangled by string concatenation.
	var gotURL string
	base := capturingBase(func(req *http.Request) { gotURL = req.URL.String() })
	ctx := ConnectionContext{
		BaseURL: BaseURL{Value: "https://example.com/api/v5?token=abc#frag", Source: SourceEnvironment},
		Cred:    auth.Resolution{Token: secretToken, Source: auth.SourceFile},
	}
	client := mustClient(t, ctx, base)

	_, err := client.Execute(
		context.Background(),
		Request{Method: http.MethodGet, Path: "/roles", Query: map[string][]string{"page": {"2"}}},
		nil,
	)
	if err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	// q.Encode() sorts keys, so page precedes token; the fragment is reattached.
	want := "https://example.com/api/v5/roles?page=2&token=abc#frag"
	if gotURL != want {
		t.Fatalf("request URL = %q, want %q", gotURL, want)
	}
}

// capturingBase adapts an inspect func into a base RoundTripper returning a bare
// 200, so a test can assert what request was built.
func capturingBase(inspect func(*http.Request)) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		inspect(req)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: bodyOf("")}, nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestExecuteSetsContentTypeWhenPresent pins the 042 ADR-1 write-body seam: a
// Request carrying a non-empty ContentType produces an outbound request whose
// Content-Type header is exactly that value (so the API parses a JSON body rather
// than ignoring it and answering 422). The body and the rest of the send path are
// unchanged.
func TestExecuteSetsContentTypeWhenPresent(t *testing.T) {
	base := &respondingBase{status: 201, body: bodyOf(`{"data":{"id":"ten_1"}}`)}
	client := mustClient(t, completeContext(secretToken), base)

	req := Request{
		Method:      http.MethodPost,
		Path:        "/roles/role_1/tensions",
		Body:        strings.NewReader(`{"tension":{"body":"x"}}`),
		ContentType: "application/json",
	}
	if _, err := client.Execute(context.Background(), req, nil); err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if !base.contentTypeSet {
		t.Fatal("Content-Type header was not set for a non-empty ContentType")
	}
	if base.gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", base.gotContentType)
	}
}

// TestExecuteOmitsContentTypeWhenEmpty pins that the empty default (every landed
// GET read) sets NO Content-Type header on the outbound request — the reads' wire
// behavior is byte-identical to before the field existed (042 ADR-1).
func TestExecuteOmitsContentTypeWhenEmpty(t *testing.T) {
	base := &respondingBase{status: 200, body: bodyOf(`{"id":"per_1"}`)}
	client := mustClient(t, completeContext(secretToken), base)

	if _, err := client.Execute(context.Background(), Request{Method: http.MethodGet, Path: "/me"}, nil); err != nil {
		t.Fatalf("Execute errored: %v", err)
	}
	if base.contentTypeSet {
		t.Fatalf("a bodyless read must set no Content-Type header, got %q", base.gotContentType)
	}
}
