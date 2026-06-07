package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

// TestRateLimitHandlingFeatures runs the Rate-Limit Handling (017) executable
// acceptance scenarios against NewRetryExecutor/Execute, driving them over a fake
// base http.RoundTripper + a recording fake-sleep (never blocks) + a buffer
// progress sink — no real network, no real sleep, no real home or filesystem.
//
// The suite is scoped to *only* rate-limit-handling.feature. godog binds steps
// per-suite, so a directory-globbing Paths would pull in the 007/008/009/010
// suites' scenarios and fail with undefined steps (LEARNINGS 2026-06-04). This is
// the fifth apiclient suite (007 → TestFeatures, 008 → TestBaseURLFeatures, 009 →
// TestConnectionContextFeatures, 010 → TestRequestExecutionFeatures, this → 017),
// each pointed at its own file. The four @validation scenarios stay @wip — held
// out for the validate skill, not implemented by the Builder.
func TestRateLimitHandlingFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRateLimitScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/no-shared-api-client/rate-limit-handling.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// rateLimitWorld is the per-scenario state. A Given installs the retry policy and
// the fake base's canned sequence; a When builds the executor over that base +
// the recording sleep + the progress buffer and runs it; the Thens assert on the
// captured outcome, the recorded waits, and the note. Step helpers return errors,
// never panic (LEARNINGS).
type rateLimitWorld struct {
	policy   RetryPolicy
	base     *sequenceBase
	sleep    *recordingSleep
	progress strings.Builder

	resp    *Response
	out     map[string]any
	execErr error
}

// execute builds the executor over the configured base + recording sleep + buffer
// and runs one request through it. The client is built from a complete context so
// the AuthTransport attaches the (secret) token — proving the note carries no
// secret even though one is in play.
func (w *rateLimitWorld) execute(method string) error {
	client, err := NewClient(completeContext(secretToken), w.base)
	if err != nil {
		return fmt.Errorf("building client: %w", err)
	}
	exec := NewRetryExecutor(client, w.policy, w.sleep.sleep, &w.progress)
	w.out = map[string]any{}
	w.resp, w.execErr = exec.Execute(context.Background(), Request{Method: method, Path: "/me"}, &w.out)
	return nil
}

func initializeRateLimitScenario(sc *godog.ScenarioContext) {
	w := &rateLimitWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = rateLimitWorld{
			policy: DefaultRetryPolicy,
			base:   &sequenceBase{steps: []cannedResp{{status: 200, body: `{}`}}},
			sleep:  &recordingSleep{},
		}
		return ctx, nil
	})

	// --- Givens: the executor + the API's canned sequence ---
	sc.Step(`^a retrying executor wrapping a client built from a complete connection context$`, w.givenExecutor)
	sc.Step(`^the API would return a 429 with "Retry-After: (\d+)" then a 200 response$`, w.given429RetryAfterThen200)
	sc.Step(`^the API would return a 200 response$`, w.givenAPI200)
	sc.Step(`^the API would return a 429 with no Retry-After then a 200 response$`, w.given429NoRetryAfterThen200)
	sc.Step(`^the API could not be reached$`, w.givenAPIUnreachable)
	sc.Step(`^the API would return a 403 response$`, w.givenAPI403)
	sc.Step(`^the API would return a 429 response$`, w.givenAPI429)
	sc.Step(`^the API would return a 429 on every attempt$`, w.givenAPI429Always)

	// --- Whens ---
	sc.Step(`^a GET request is executed through the retrying executor$`, func() error { return w.execute(http.MethodGet) })
	sc.Step(`^a POST request is executed through the retrying executor$`, func() error { return w.execute(http.MethodPost) })

	// --- Thens ---
	sc.Step(`^the executor will wait about (\d+) seconds and re-attempt the request$`, w.thenWaitedSecondsAndReattempted)
	sc.Step(`^the eventual 200 response will be returned$`, w.then200Returned)
	sc.Step(`^the 200 response will be returned$`, w.then200Returned)
	sc.Step(`^no wait will be imposed and only one attempt will be made$`, w.thenNoWaitOneAttempt)
	sc.Step(`^the executor will wait the fallback backoff interval and re-attempt$`, w.thenWaitedFallbackAndReattempted)
	sc.Step(`^a transport error will be returned$`, w.thenTransportErrorReturned)
	sc.Step(`^the 403 response error will be returned unchanged$`, w.then403Unchanged)
	sc.Step(`^the 429 response error will be returned on the first occurrence$`, w.then429OnFirstOccurrence)
	sc.Step(`^no wait will be imposed and the request will not be re-sent$`, w.thenNoWaitNotResent)
	sc.Step(`^the executor will stop after the maximum number of attempts$`, w.thenStoppedAtMaxAttempts)
	sc.Step(`^the most recent 429 response error carrying its status, rate-limit headers, and body will be returned$`, w.thenLast429Full)
	sc.Step(`^the error will not be classified by failure kind$`, w.thenResponseErrorGeneric)
	sc.Step(`^a progress note naming the wait and the next attempt will be written to standard error$`, w.thenProgressNote)
	sc.Step(`^the note will contain no token or secret$`, w.thenNoSecretInNote)
}

// --- Given implementations ---

func (w *rateLimitWorld) givenExecutor() error {
	// The Before hook already seeded a complete-context executor with the default
	// policy and a 200 base; the API Given that follows overrides the base.
	return nil
}

func (w *rateLimitWorld) given429RetryAfterThen200(secs string) error {
	w.base = &sequenceBase{steps: []cannedResp{
		{status: 429, header: retryAfter(secs), body: `{"error":"rate limited"}`},
		{status: 200, body: `{"id":"per_1"}`},
	}}
	return nil
}

func (w *rateLimitWorld) givenAPI200() error {
	w.base = &sequenceBase{steps: []cannedResp{{status: 200, body: `{"id":"per_1"}`}}}
	return nil
}

func (w *rateLimitWorld) given429NoRetryAfterThen200() error {
	w.base = &sequenceBase{steps: []cannedResp{
		{status: 429, body: `{"error":"rate limited"}`}, // no Retry-After
		{status: 200, body: `{"id":"per_1"}`},
	}}
	return nil
}

func (w *rateLimitWorld) givenAPIUnreachable() error {
	w.base = &sequenceBase{netErr: errors.New("dial tcp 127.0.0.1:443: connect: connection refused")}
	return nil
}

func (w *rateLimitWorld) givenAPI403() error {
	w.base = &sequenceBase{steps: []cannedResp{{status: 403, body: `{"error":"forbidden"}`}}}
	return nil
}

func (w *rateLimitWorld) givenAPI429() error {
	w.base = &sequenceBase{steps: []cannedResp{{status: 429, header: retryAfter("2"), body: `{"error":"rate limited"}`}}}
	return nil
}

func (w *rateLimitWorld) givenAPI429Always() error {
	header := retryAfter("1")
	header.Set("X-Ratelimit-Remaining", "0")
	// One 429 step, repeated by the sequenceBase on every attempt.
	w.base = &sequenceBase{steps: []cannedResp{{status: 429, header: header, body: `{"error":"rate limited"}`}}}
	return nil
}

// --- Then implementations ---

func (w *rateLimitWorld) thenWaitedSecondsAndReattempted(secs string) error {
	n, err := strconv.Atoi(secs)
	if err != nil {
		return fmt.Errorf("bad seconds %q: %w", secs, err)
	}
	want := time.Duration(n) * time.Second
	if len(w.sleep.durs) != 1 || w.sleep.durs[0] != want {
		return fmt.Errorf("waited %v, want a single %v wait (the Retry-After interval)", w.sleep.durs, want)
	}
	if w.base.calls != 2 {
		return fmt.Errorf("base called %d times, want 2 (the 429 then the re-attempt)", w.base.calls)
	}
	return nil
}

func (w *rateLimitWorld) then200Returned() error {
	if w.execErr != nil {
		return fmt.Errorf("unexpected error: %v", w.execErr)
	}
	if w.resp == nil || w.resp.StatusCode != 200 {
		return fmt.Errorf("resp = %v, want a 200 response", w.resp)
	}
	return nil
}

func (w *rateLimitWorld) thenNoWaitOneAttempt() error {
	if len(w.sleep.durs) != 0 {
		return fmt.Errorf("slept %v, want no wait", w.sleep.durs)
	}
	if w.base.calls != 1 {
		return fmt.Errorf("base called %d times, want exactly 1 (no retry)", w.base.calls)
	}
	return nil
}

func (w *rateLimitWorld) thenWaitedFallbackAndReattempted() error {
	if len(w.sleep.durs) != 1 || w.sleep.durs[0] != w.policy.FallbackBackoff {
		return fmt.Errorf("waited %v, want a single fallback-backoff %v wait", w.sleep.durs, w.policy.FallbackBackoff)
	}
	if w.base.calls != 2 {
		return fmt.Errorf("base called %d times, want 2 (the 429 then the re-attempt)", w.base.calls)
	}
	return nil
}

func (w *rateLimitWorld) thenTransportErrorReturned() error {
	var te *TransportError
	if !errors.As(w.execErr, &te) {
		return fmt.Errorf("err = %v, want *TransportError", w.execErr)
	}
	return nil
}

func (w *rateLimitWorld) then403Unchanged() error {
	var re *ResponseError
	if !errors.As(w.execErr, &re) {
		return fmt.Errorf("err = %v, want *ResponseError", w.execErr)
	}
	if re.StatusCode != 403 {
		return fmt.Errorf("status = %d, want 403 (passed through unchanged)", re.StatusCode)
	}
	return nil
}

func (w *rateLimitWorld) then429OnFirstOccurrence() error {
	var re *ResponseError
	if !errors.As(w.execErr, &re) {
		return fmt.Errorf("err = %v, want *ResponseError", w.execErr)
	}
	if re.StatusCode != 429 {
		return fmt.Errorf("status = %d, want 429", re.StatusCode)
	}
	return nil
}

func (w *rateLimitWorld) thenNoWaitNotResent() error {
	if len(w.sleep.durs) != 0 {
		return fmt.Errorf("slept %v, want no wait for a write", w.sleep.durs)
	}
	if w.base.calls != 1 {
		return fmt.Errorf("base called %d times, want exactly 1 (the write is not re-sent)", w.base.calls)
	}
	return nil
}

func (w *rateLimitWorld) thenStoppedAtMaxAttempts() error {
	if w.base.calls != w.policy.MaxAttempts {
		return fmt.Errorf("base called %d times, want exactly MaxAttempts=%d", w.base.calls, w.policy.MaxAttempts)
	}
	return nil
}

func (w *rateLimitWorld) thenLast429Full() error {
	var re *ResponseError
	if !errors.As(w.execErr, &re) {
		return fmt.Errorf("err = %v, want the raw *ResponseError", w.execErr)
	}
	if re.StatusCode != 429 {
		return fmt.Errorf("status = %d, want 429", re.StatusCode)
	}
	if re.Header.Get("Retry-After") == "" && re.Header.Get("X-Ratelimit-Remaining") == "" {
		return errors.New("the surfaced 429 carried no rate-limit headers")
	}
	if len(re.Body) == 0 {
		return errors.New("the surfaced 429 carried no body")
	}
	return nil
}

// thenResponseErrorGeneric pins that the surfaced error is the generic
// *ResponseError carrier — not a classified rate-limit type (that is 015's, ADR-5).
// Matches 010's step vocabulary for the same assertion.
func (w *rateLimitWorld) thenResponseErrorGeneric() error {
	var re *ResponseError
	if !errors.As(w.execErr, &re) {
		return fmt.Errorf("err = %v, want the generic *ResponseError (no failure-kind classification)", w.execErr)
	}
	return nil
}

func (w *rateLimitWorld) thenProgressNote() error {
	note := w.progress.String()
	if strings.TrimSpace(note) == "" {
		return errors.New("no progress note was written before the retry")
	}
	if !strings.Contains(note, "retry") {
		return fmt.Errorf("progress note %q should name the re-attempt", note)
	}
	if !strings.Contains(note, "2/") {
		return fmt.Errorf("progress note %q should name the next attempt index", note)
	}
	return nil
}

func (w *rateLimitWorld) thenNoSecretInNote() error {
	if strings.Contains(w.progress.String(), secretToken) {
		return fmt.Errorf("the token leaked into the progress note: %q", w.progress.String())
	}
	if w.execErr != nil && strings.Contains(w.execErr.Error(), secretToken) {
		return fmt.Errorf("the token leaked into the surfaced error: %q", w.execErr.Error())
	}
	return nil
}
