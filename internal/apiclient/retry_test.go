package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// --- T001: isSafeMethod / parseRetryAfter / DefaultRetryPolicy (pure) -----

func TestIsSafeMethod(t *testing.T) {
	safe := []string{http.MethodGet, http.MethodHead}
	for _, m := range safe {
		if !isSafeMethod(m) {
			t.Errorf("isSafeMethod(%q) = false, want true (idempotent read)", m)
		}
	}
	unsafe := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, m := range unsafe {
		if isSafeMethod(m) {
			t.Errorf("isSafeMethod(%q) = true, want false (non-idempotent)", m)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		name    string
		value   string // empty name "absent" means no header at all
		header  bool   // whether to set the header
		wantDur time.Duration
		wantOK  bool
	}{
		{name: "integer seconds", value: "42", header: true, wantDur: 42 * time.Second, wantOK: true},
		{name: "zero means immediate", value: "0", header: true, wantDur: 0, wantOK: true},
		{name: "empty value", value: "", header: true, wantDur: 0, wantOK: false},
		{name: "absent header", header: false, wantDur: 0, wantOK: false},
		{name: "negative", value: "-1", header: true, wantDur: 0, wantOK: false},
		{name: "non-integer", value: "abc", header: true, wantDur: 0, wantOK: false},
		{name: "http-date form", value: "Wed, 21 Oct 2015 07:28:00 GMT", header: true, wantDur: 0, wantOK: false},
		// A value so large it would overflow time.Duration when scaled to seconds is
		// unusable — never honored as a negative/garbage wait that bypasses the budget.
		{name: "overflows time.Duration", value: "99999999999999", header: true, wantDur: 0, wantOK: false},
		{name: "largest non-overflowing", value: strconv.FormatInt(maxRetryAfterSeconds, 10), header: true, wantDur: time.Duration(maxRetryAfterSeconds) * time.Second, wantOK: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := make(http.Header)
			if tc.header {
				h.Set("Retry-After", tc.value)
			}
			gotDur, gotOK := parseRetryAfter(h)
			if gotDur != tc.wantDur || gotOK != tc.wantOK {
				t.Errorf("parseRetryAfter(%q) = (%v, %v), want (%v, %v)", tc.value, gotDur, gotOK, tc.wantDur, tc.wantOK)
			}
		})
	}
}

func TestDefaultRetryPolicyFloors(t *testing.T) {
	if DefaultRetryPolicy.MaxAttempts < 2 {
		t.Errorf("MaxAttempts = %d, want ≥ 2", DefaultRetryPolicy.MaxAttempts)
	}
	if DefaultRetryPolicy.MaxTotalWait <= 0 {
		t.Errorf("MaxTotalWait = %v, want positive", DefaultRetryPolicy.MaxTotalWait)
	}
	if DefaultRetryPolicy.FallbackBackoff <= 0 {
		t.Errorf("FallbackBackoff = %v, want positive", DefaultRetryPolicy.FallbackBackoff)
	}
}

// --- T002 test scaffolding ------------------------------------------------

// cannedResp is one canned reply a sequenceBase returns on a single attempt.
type cannedResp struct {
	status int
	header http.Header
	body   string
}

// sequenceBase is a fake base http.RoundTripper that returns canned replies in
// order — the i-th attempt gets steps[i], and once the list is exhausted it
// repeats the last entry (so "always 429" is one 429 step). When netErr is set it
// fails at the wire on every call (a transport failure). It counts its calls so a
// test can pin the exact number of attempts (the retry tripwire).
type sequenceBase struct {
	calls  int
	steps  []cannedResp
	netErr error
}

func (b *sequenceBase) RoundTrip(req *http.Request) (*http.Response, error) {
	b.calls++
	if b.netErr != nil {
		return nil, b.netErr
	}
	i := b.calls - 1
	if i >= len(b.steps) {
		i = len(b.steps) - 1 // repeat the last canned reply
	}
	s := b.steps[i]
	header := s.header
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{
		StatusCode: s.status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

// recordingSleep records the durations it is asked to sleep without ever
// blocking, so cap/backoff tests assert durations in milliseconds (CONSTITUTION
// IV — no real time.Sleep).
type recordingSleep struct {
	durs []time.Duration
}

func (r *recordingSleep) sleep(d time.Duration) { r.durs = append(r.durs, d) }

func (r *recordingSleep) total() time.Duration {
	var t time.Duration
	for _, d := range r.durs {
		t += d
	}
	return t
}

func retryAfter(secs string) http.Header {
	h := make(http.Header)
	h.Set("Retry-After", secs)
	return h
}

// newExecutorOver builds a RetryExecutor over a fake base + recording sleep +
// buffer, using the given policy. It returns the executor, the recording sleep,
// and the progress buffer for assertions.
func newExecutorOver(t *testing.T, base http.RoundTripper, policy RetryPolicy) (*RetryExecutor, *recordingSleep, *strings.Builder) {
	t.Helper()
	client := mustClient(t, completeContext(secretToken), base)
	sleep := &recordingSleep{}
	var progress strings.Builder
	exec := NewRetryExecutor(client, policy, sleep.sleep, &progress)
	return exec, sleep, &progress
}

func getReq(method string) Request { return Request{Method: method, Path: "/me"} }

// --- T002: the retry loop -------------------------------------------------

func TestRetryExecutor_429ThenSuccessHonorsRetryAfter(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{
		{status: 429, header: retryAfter("2"), body: `{"error":"rate limited"}`},
		{status: 200, body: `{"id":"per_1"}`},
	}}
	exec, sleep, _ := newExecutorOver(t, base, DefaultRetryPolicy)

	var out map[string]any
	resp, err := exec.Execute(context.Background(), getReq(http.MethodGet), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("resp = %v, want a 200 response", resp)
	}
	if out["id"] != "per_1" {
		t.Errorf("out = %v, want the 200 body decoded", out)
	}
	if base.calls != 2 {
		t.Errorf("base called %d times, want 2 (429 then 200)", base.calls)
	}
	if len(sleep.durs) != 1 || sleep.durs[0] != 2*time.Second {
		t.Errorf("slept %v, want exactly [2s] (the Retry-After interval)", sleep.durs)
	}
}

func TestRetryExecutor_429NoRetryAfterUsesFallback(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{
		{status: 429, body: `{}`}, // no Retry-After
		{status: 200, body: `{}`},
	}}
	exec, sleep, _ := newExecutorOver(t, base, DefaultRetryPolicy)

	if _, err := exec.Execute(context.Background(), getReq(http.MethodGet), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sleep.durs) != 1 || sleep.durs[0] != DefaultRetryPolicy.FallbackBackoff {
		t.Errorf("slept %v, want exactly [%v] (the fallback backoff)", sleep.durs, DefaultRetryPolicy.FallbackBackoff)
	}
}

func TestRetryExecutor_NonRetryOutcomesPassThroughOnce(t *testing.T) {
	cases := []struct {
		name string
		base *sequenceBase
		ctx  ConnectionContext
		want func(error) bool
	}{
		{
			name: "200 success",
			base: &sequenceBase{steps: []cannedResp{{status: 200, body: `{}`}}},
			want: func(err error) bool { return err == nil },
		},
		{
			name: "403 non-429 non-2xx",
			base: &sequenceBase{steps: []cannedResp{{status: 403, body: `{"error":"forbidden"}`}}},
			want: func(err error) bool {
				var re *ResponseError
				return errors.As(err, &re) && re.StatusCode == 403
			},
		},
		{
			name: "transport error",
			base: &sequenceBase{netErr: errors.New("dial tcp: connection refused")},
			want: func(err error) bool {
				var te *TransportError
				return errors.As(err, &te)
			},
		},
		{
			name: "decode error on a 2xx",
			base: &sequenceBase{steps: []cannedResp{{status: 200, body: `not json`}}},
			want: func(err error) bool {
				var de *DecodeError
				return errors.As(err, &de)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec, sleep, _ := newExecutorOver(t, tc.base, DefaultRetryPolicy)
			var out map[string]any
			_, err := exec.Execute(context.Background(), getReq(http.MethodGet), &out)
			if !tc.want(err) {
				t.Fatalf("err = %v, did not match the expected unchanged outcome", err)
			}
			if tc.base.calls != 1 {
				t.Errorf("base called %d times, want exactly 1 (no retry)", tc.base.calls)
			}
			if len(sleep.durs) != 0 {
				t.Errorf("slept %v, want no sleep on a non-429 outcome", sleep.durs)
			}
		})
	}
}

// An *AuthError surfaced by the transport (no-token context) passes straight
// through, exactly one attempt, no sleep — it is not a 429.
func TestRetryExecutor_AuthErrorPassesThrough(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{{status: 200, body: `{}`}}}
	ctx := ConnectionContext{
		BaseURL: BaseURL{Value: "https://example.test/api/v5", Source: SourceDefault},
	}
	client := mustClient(t, ctx, base) // no token → AuthTransport fail-safe fires
	sleep := &recordingSleep{}
	var progress strings.Builder
	exec := NewRetryExecutor(client, DefaultRetryPolicy, sleep.sleep, &progress)

	_, err := exec.Execute(context.Background(), getReq(http.MethodGet), nil)
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %v, want the propagated *AuthError", err)
	}
	if base.calls != 0 {
		t.Errorf("base reached %d times; an unauthenticated request must not be sent", base.calls)
	}
	if len(sleep.durs) != 0 {
		t.Errorf("slept %v, want no sleep", sleep.durs)
	}
}

func TestRetryExecutor_AttemptsExhaustedSurfacesRaw429(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{
		{status: 429, header: retryAfter("0"), body: `{"error":"rate limited"}`},
	}} // repeats the 429 forever
	exec, sleep, _ := newExecutorOver(t, base, DefaultRetryPolicy)

	resp, err := exec.Execute(context.Background(), getReq(http.MethodGet), nil)
	if resp != nil {
		t.Errorf("resp = %v, want nil on a surfaced 429", resp)
	}
	if base.calls != DefaultRetryPolicy.MaxAttempts {
		t.Errorf("base called %d times, want exactly MaxAttempts=%d", base.calls, DefaultRetryPolicy.MaxAttempts)
	}
	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("err = %v, want the raw *ResponseError", err)
	}
	if respErr.StatusCode != 429 {
		t.Errorf("status = %d, want 429", respErr.StatusCode)
	}
	if respErr.Header.Get("Retry-After") == "" {
		t.Error("the surfaced 429 lost its rate-limit headers")
	}
	if len(respErr.Body) == 0 {
		t.Error("the surfaced 429 lost its body")
	}
	// MaxAttempts attempts → MaxAttempts-1 waits.
	if len(sleep.durs) != DefaultRetryPolicy.MaxAttempts-1 {
		t.Errorf("slept %d times, want MaxAttempts-1=%d", len(sleep.durs), DefaultRetryPolicy.MaxAttempts-1)
	}
}

func TestRetryExecutor_TotalWaitBudgetGivesUpWithoutTruncatedSleep(t *testing.T) {
	// Each Retry-After is 40s; with a 60s budget the first wait fits (40 ≤ 60) but
	// the second (40+40=80 > 60) does not — so the executor gives up before a
	// second sleep and never sleeps beyond the cap.
	policy := RetryPolicy{MaxAttempts: 10, MaxTotalWait: 60 * time.Second, FallbackBackoff: 2 * time.Second}
	base := &sequenceBase{steps: []cannedResp{
		{status: 429, header: retryAfter("40"), body: `{}`},
	}}
	exec, sleep, _ := newExecutorOver(t, base, policy)

	_, err := exec.Execute(context.Background(), getReq(http.MethodGet), nil)
	var respErr *ResponseError
	if !errors.As(err, &respErr) || respErr.StatusCode != 429 {
		t.Fatalf("err = %v, want the last 429 *ResponseError", err)
	}
	if sleep.total() > policy.MaxTotalWait {
		t.Errorf("total slept %v exceeds the budget %v", sleep.total(), policy.MaxTotalWait)
	}
	if len(sleep.durs) != 1 {
		t.Errorf("slept %v, want exactly one wait (the second would exceed the budget)", sleep.durs)
	}
}

func TestRetryExecutor_NonSafeMethodNotRetried(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{{status: 429, header: retryAfter("2"), body: `{}`}}}
	exec, sleep, _ := newExecutorOver(t, base, DefaultRetryPolicy)

	_, err := exec.Execute(context.Background(), getReq(http.MethodPost), nil)
	var respErr *ResponseError
	if !errors.As(err, &respErr) || respErr.StatusCode != 429 {
		t.Fatalf("err = %v, want the 429 returned on first occurrence", err)
	}
	if base.calls != 1 {
		t.Errorf("base called %d times, want exactly 1 (a write is not auto-retried)", base.calls)
	}
	if len(sleep.durs) != 0 {
		t.Errorf("slept %v, want no sleep for a write", sleep.durs)
	}
}

func TestRetryExecutor_ProgressNoteNamesWaitAndAttemptNoSecret(t *testing.T) {
	base := &sequenceBase{steps: []cannedResp{
		{status: 429, header: retryAfter("5"), body: `{}`},
		{status: 200, body: `{}`},
	}}
	exec, _, progress := newExecutorOver(t, base, DefaultRetryPolicy)

	if _, err := exec.Execute(context.Background(), getReq(http.MethodGet), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	note := progress.String()
	if !strings.Contains(note, "5s") {
		t.Errorf("progress note %q should name the wait (5s)", note)
	}
	if !strings.Contains(note, "2/") {
		t.Errorf("progress note %q should name the next attempt index", note)
	}
	if strings.Contains(note, secretToken) {
		t.Errorf("the token leaked into the progress note: %q", note)
	}
}

func TestNewRetryExecutor_NilSeamsPanic(t *testing.T) {
	client := mustClient(t, completeContext(secretToken), &sequenceBase{steps: []cannedResp{{status: 200, body: `{}`}}})
	sleep := func(time.Duration) {}
	var progress strings.Builder

	cases := map[string]func(){
		"nil client":   func() { NewRetryExecutor(nil, DefaultRetryPolicy, sleep, &progress) },
		"nil sleep":    func() { NewRetryExecutor(client, DefaultRetryPolicy, nil, &progress) },
		"nil progress": func() { NewRetryExecutor(client, DefaultRetryPolicy, sleep, nil) },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: NewRetryExecutor did not panic (fail-fast on a nil seam)", name)
				}
			}()
			build()
		})
	}
}
