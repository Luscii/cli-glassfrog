package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/cucumber/godog"
)

// TestRequestExecutionFeatures runs the Request Execution (010) executable
// acceptance scenarios against NewClient/Execute, driving them over a fake base
// http.RoundTripper — no real network beyond what the fakes model, no real home
// or filesystem.
//
// The suite is scoped to *only* request-execution.feature. godog binds steps
// per-suite, so a directory-globbing Paths would pull in the 007/008/009 suites'
// scenarios and fail with undefined steps (LEARNINGS 2026-06-04). This is the
// fourth apiclient suite (007 → TestFeatures, 008 → TestBaseURLFeatures, 009 →
// TestConnectionContextFeatures, this → 010), each pointed at its own file. The
// four @validation scenarios stay @wip — held out for the validate skill.
func TestRequestExecutionFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRequestExecutionScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/no-shared-api-client/request-execution.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// reqWorld is the per-scenario state. A Given sets up the connection context and
// the fake base; a When builds the client and (usually) executes; the Thens
// assert on the captured outcome. Step helpers return errors, never panic
// (LEARNINGS).
type reqWorld struct {
	ctx  ConnectionContext
	base http.RoundTripper

	// Concrete refs to whichever fake base a Given installed, for call-count and
	// header assertions. Exactly one is non-nil per scenario.
	responding *respondingBase
	erroring   *erroringBase
	blocking   *blockingBase

	hang  bool   // the base never responds → use a short injected deadline
	token string // the token a complete/present context carries, for the auth check

	client   *Client
	buildErr error

	targetSupplied bool
	target         map[string]any
	resp           *Response
	execErr        error
}

func (w *reqWorld) baseCalls() int {
	switch {
	case w.responding != nil:
		return w.responding.calls
	case w.erroring != nil:
		return w.erroring.calls
	case w.blocking != nil:
		return w.blocking.calls
	}
	return 0
}

func (w *reqWorld) build() {
	w.client, w.buildErr = NewClient(w.ctx, w.base)
}

func (w *reqWorld) execute(withTarget bool) {
	w.build()
	if w.client == nil {
		return // build failed (e.g. base-URL fail-fast); nothing to send
	}
	reqCtx := context.Background()
	if w.hang {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(reqCtx, 50*time.Millisecond)
		defer cancel()
	}
	var out any
	if withTarget {
		w.targetSupplied = true
		w.target = map[string]any{}
		out = &w.target
	}
	w.resp, w.execErr = w.client.Execute(reqCtx, Request{Method: http.MethodGet, Path: "/me"}, out)
}

func initializeRequestExecutionScenario(sc *godog.ScenarioContext) {
	w := &reqWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = reqWorld{}
		return ctx, nil
	})

	// --- Givens: the client / connection context ---
	sc.Step(`^a client had been built from a complete connection context$`, w.givenCompleteContext)
	sc.Step(`^a client had been built from a connection context carrying a present token$`, w.givenPresentTokenContext)
	sc.Step(`^a client had been built from a connection context with a usable base URL but no token$`, w.givenNoTokenContext)
	sc.Step(`^a connection context had carried a base-URL error$`, w.givenBaseURLErrorContext)

	// --- Givens: the API's canned reply ---
	sc.Step(`^the API would return a 200 response with a JSON body$`, w.givenAPI200JSON)
	sc.Step(`^the API would return a 204 response with no body$`, w.givenAPI204)
	sc.Step(`^the API would return a 403 response with a body$`, w.givenAPI403)
	sc.Step(`^the API would return a 200 response whose body does not match the target$`, w.givenAPI200Undecodable)
	sc.Step(`^the API would return a 429 rate-limit response$`, w.givenAPI429)
	sc.Step(`^the API could not be reached$`, w.givenAPIUnreachable)
	sc.Step(`^the API accepts the connection but never responds$`, w.givenAPIHangs)

	// --- Whens ---
	sc.Step(`^an endpoint command executes a request with a decode target$`, func() error { w.execute(true); return nil })
	sc.Step(`^an endpoint command executes a request with no decode target$`, func() error { w.execute(false); return nil })
	sc.Step(`^an endpoint command executes a request$`, func() error { w.execute(false); return nil })
	sc.Step(`^an endpoint command builds a client for the request$`, func() error { w.build(); return nil })

	// --- Thens: success ---
	sc.Step(`^the response status and headers will be returned$`, w.thenStatusAndHeaders)
	sc.Step(`^the body will be decoded into the target$`, w.thenBodyDecoded)
	sc.Step(`^no body will be decoded$`, w.thenNoBodyDecoded)
	sc.Step(`^the outgoing request will carry the X-Auth-Token header attached by the transport$`, w.thenHeaderAttachedByTransport)
	sc.Step(`^the client will not attach the header itself$`, w.thenClientDoesNotAttach)

	// --- Thens: typed failures ---
	sc.Step(`^a transport error naming the failure will be returned$`, w.thenTransportError)
	sc.Step(`^no response or decoded body will be returned$`, w.thenNoResponseOrBody)
	sc.Step(`^a response error carrying the status, headers, and raw body will be returned$`, w.thenResponseErrorFull)
	sc.Step(`^the body will not be decoded into the target$`, w.thenTargetUntouched)
	sc.Step(`^the error will not be classified by failure kind$`, w.thenResponseErrorGeneric)
	sc.Step(`^the base-URL error will be returned$`, w.thenBaseURLErrorReturned)
	sc.Step(`^no client will be built and no request will reach the API$`, w.thenNoClientNoRequest)
	sc.Step(`^a decode error will be returned rather than a success$`, w.thenDecodeError)
	sc.Step(`^the request timeout will elapse$`, w.thenTimeoutElapsed)
	sc.Step(`^a transport error will be returned without a retry$`, w.thenTransportErrorNoRetry)
	sc.Step(`^the transport's authentication failure will be propagated$`, w.thenAuthErrorPropagated)
	sc.Step(`^no unauthenticated request will be sent$`, w.thenNoRequestSent)

	// --- Thens: status + headers exposed (429) ---
	sc.Step(`^a response error carrying the 429 status and rate-limit headers will be returned$`, w.thenResponseError429)
	sc.Step(`^the client will not sleep, back off, or retry$`, w.thenNoRetry)
}

// --- Given implementations ---

func (w *reqWorld) setResponding(b *respondingBase) {
	w.responding = b
	w.base = b
}

func (w *reqWorld) givenCompleteContext() error {
	w.ctx = completeContext(secretToken)
	w.token = secretToken
	// Default reply unless a later API Given overrides it.
	w.setResponding(&respondingBase{status: 200, body: bodyOf("{}")})
	return nil
}

func (w *reqWorld) givenPresentTokenContext() error {
	w.ctx = completeContext(secretToken)
	w.token = secretToken
	w.setResponding(&respondingBase{status: 200, body: bodyOf("{}")})
	return nil
}

func (w *reqWorld) givenNoTokenContext() error {
	w.ctx = ConnectionContext{
		BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceDefault},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	// A base that would record a call, to prove none is made.
	w.setResponding(&respondingBase{status: 200, body: bodyOf("{}")})
	return nil
}

func (w *reqWorld) givenBaseURLErrorContext() error {
	w.ctx = ConnectionContext{BaseURLErr: &BaseURLError{Source: "--base-url"}}
	w.setResponding(&respondingBase{status: 200, body: bodyOf("{}")})
	return nil
}

func (w *reqWorld) givenAPI200JSON() error {
	w.setResponding(&respondingBase{status: 200, body: bodyOf(`{"id":"per_1","name":"Ada"}`)})
	return nil
}

func (w *reqWorld) givenAPI204() error {
	w.setResponding(&respondingBase{status: 204, body: bodyOf("")})
	return nil
}

func (w *reqWorld) givenAPI403() error {
	header := http.Header{"Content-Type": []string{"application/json"}}
	w.setResponding(&respondingBase{status: 403, header: header, body: bodyOf(`{"error":"forbidden"}`)})
	return nil
}

func (w *reqWorld) givenAPI200Undecodable() error {
	w.setResponding(&respondingBase{status: 200, body: bodyOf(`this is not json`)})
	return nil
}

func (w *reqWorld) givenAPI429() error {
	header := http.Header{
		"Retry-After":           []string{"60"},
		"X-Ratelimit-Remaining": []string{"0"},
	}
	w.setResponding(&respondingBase{status: 429, header: header, body: bodyOf(`{"error":"rate limited"}`)})
	return nil
}

func (w *reqWorld) givenAPIUnreachable() error {
	w.responding = nil // this Given overrides the default responding base
	w.erroring = &erroringBase{err: errors.New("dial tcp 127.0.0.1:443: connect: connection refused")}
	w.base = w.erroring
	return nil
}

func (w *reqWorld) givenAPIHangs() error {
	w.responding = nil // this Given overrides the default responding base
	w.blocking = &blockingBase{}
	w.base = w.blocking
	w.hang = true
	return nil
}

// --- Then implementations ---

func (w *reqWorld) thenStatusAndHeaders() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.resp == nil {
		return errors.New("no response was returned")
	}
	if w.resp.StatusCode == 0 {
		return errors.New("response carried no status code")
	}
	if w.resp.Header == nil {
		return errors.New("response carried no headers")
	}
	return nil
}

func (w *reqWorld) thenBodyDecoded() error {
	if !w.targetSupplied {
		return errors.New("no decode target was supplied")
	}
	if len(w.target) == 0 {
		return errors.New("the body was not decoded into the target")
	}
	return nil
}

func (w *reqWorld) thenNoBodyDecoded() error {
	if w.resp == nil {
		return errors.New("no response was returned")
	}
	if w.targetSupplied {
		return errors.New("a decode target was supplied; this scenario expects none")
	}
	return nil
}

func (w *reqWorld) thenHeaderAttachedByTransport() error {
	if w.responding == nil {
		return errors.New("no responding base recorded the outgoing request")
	}
	if w.responding.gotToken != w.token {
		return fmt.Errorf("outgoing X-Auth-Token = %q, want %q attached by the transport", w.responding.gotToken, w.token)
	}
	return nil
}

func (w *reqWorld) thenClientDoesNotAttach() error {
	// The guarantee is that Execute never reads the token (it has no access — only
	// the replay thunk inside 007's AuthTransport does) and relies on the transport
	// to attach the header, so the only X-Auth-Token the base saw is the one
	// AuthTransport set.
	if w.responding == nil || w.responding.gotToken != w.token {
		return errors.New("the credential was not attached by the transport")
	}
	return nil
}

func (w *reqWorld) thenTransportError() error {
	var transErr *TransportError
	if !errors.As(w.execErr, &transErr) {
		return fmt.Errorf("err = %v, want *TransportError", w.execErr)
	}
	if transErr.Error() == "" {
		return errors.New("the transport error names no failure")
	}
	return nil
}

func (w *reqWorld) thenNoResponseOrBody() error {
	if w.resp != nil {
		return fmt.Errorf("response = %v, want nil", w.resp)
	}
	if len(w.target) != 0 {
		return fmt.Errorf("target was decoded into: %v, want untouched", w.target)
	}
	return nil
}

func (w *reqWorld) thenResponseErrorFull() error {
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want *ResponseError", w.execErr)
	}
	if respErr.StatusCode == 0 {
		return errors.New("response error carried no status")
	}
	if respErr.Header == nil {
		return errors.New("response error carried no headers")
	}
	if len(respErr.Body) == 0 {
		return errors.New("response error carried no raw body")
	}
	return nil
}

func (w *reqWorld) thenTargetUntouched() error {
	if len(w.target) != 0 {
		return fmt.Errorf("the body was decoded into the target: %v, want untouched", w.target)
	}
	return nil
}

func (w *reqWorld) thenResponseErrorGeneric() error {
	// Generic means it is the plain *ResponseError carrier — not a classified
	// API/permission/rate-limit type (those are 015/017). The type has no Kind.
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want the generic *ResponseError", w.execErr)
	}
	return nil
}

func (w *reqWorld) thenBaseURLErrorReturned() error {
	var baseErr *BaseURLError
	if !errors.As(w.buildErr, &baseErr) {
		return fmt.Errorf("build err = %v, want the carried *BaseURLError", w.buildErr)
	}
	return nil
}

func (w *reqWorld) thenNoClientNoRequest() error {
	if w.client != nil {
		return errors.New("a client was built on the base-URL fail-fast branch")
	}
	if w.baseCalls() != 0 {
		return fmt.Errorf("the base was reached %d times; no request may be sent", w.baseCalls())
	}
	return nil
}

func (w *reqWorld) thenDecodeError() error {
	if w.resp != nil {
		return fmt.Errorf("response = %v, want nil on a decode failure", w.resp)
	}
	var decErr *DecodeError
	if !errors.As(w.execErr, &decErr) {
		return fmt.Errorf("err = %v, want *DecodeError", w.execErr)
	}
	return nil
}

func (w *reqWorld) thenTimeoutElapsed() error {
	if w.execErr == nil {
		return errors.New("the request did not fail; the timeout did not elapse")
	}
	return nil
}

func (w *reqWorld) thenTransportErrorNoRetry() error {
	if err := w.thenTransportError(); err != nil {
		return err
	}
	if w.baseCalls() != 1 {
		return fmt.Errorf("the base was reached %d times, want exactly 1 (no retry)", w.baseCalls())
	}
	return nil
}

func (w *reqWorld) thenAuthErrorPropagated() error {
	var authErr *AuthError
	if !errors.As(w.execErr, &authErr) {
		return fmt.Errorf("err = %v, want the propagated *AuthError", w.execErr)
	}
	var transErr *TransportError
	if errors.As(w.execErr, &transErr) {
		return errors.New("the auth fail-safe was mislabeled as a *TransportError")
	}
	return nil
}

func (w *reqWorld) thenNoRequestSent() error {
	if w.baseCalls() != 0 {
		return fmt.Errorf("the base was reached %d times; no unauthenticated request may be sent", w.baseCalls())
	}
	return nil
}

func (w *reqWorld) thenResponseError429() error {
	var respErr *ResponseError
	if !errors.As(w.execErr, &respErr) {
		return fmt.Errorf("err = %v, want *ResponseError", w.execErr)
	}
	if respErr.StatusCode != 429 {
		return fmt.Errorf("status = %d, want 429", respErr.StatusCode)
	}
	if respErr.Header.Get("Retry-After") == "" && respErr.Header.Get("X-Ratelimit-Remaining") == "" {
		return errors.New("the response error carried no rate-limit headers")
	}
	return nil
}

func (w *reqWorld) thenNoRetry() error {
	if w.baseCalls() != 1 {
		return fmt.Errorf("the base was reached %d times, want exactly 1 (no sleep/backoff/retry)", w.baseCalls())
	}
	return nil
}
