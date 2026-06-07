package apiclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// The retry-policy caps are named [ASSUMED] constants — like 010's requestTimeout
// (the only other tunable this package introduces), their values and eventual
// configurability are deferred (as 008 deferred the default URL). They satisfy
// the contract floors: MaxAttempts ≥ 2, positive MaxTotalWait and FallbackBackoff.
const (
	// defaultMaxAttempts is the total number of attempts including the first. 4
	// gives three bounded retries before a transient throttle is surfaced.
	defaultMaxAttempts = 4
	// defaultMaxTotalWait bounds the accumulated sleep across all waits, so a
	// far-out window reset can never hang a command for an unbounded stretch.
	defaultMaxTotalWait = 60 * time.Second
	// defaultFallbackBackoff is the wait used for a 429 that carries no usable
	// Retry-After header.
	defaultFallbackBackoff = 2 * time.Second
)

// RetryPolicy is the code-free retry configuration. Its field shape is the
// contract; the constant values are [ASSUMED] and deferred (interface-spec). The
// "no retry" behavior is just MaxAttempts == 1 — the same code path serves a
// future caller that opts out.
type RetryPolicy struct {
	// MaxAttempts is the total attempts incl. the first (≥1). 1 = no retry.
	MaxAttempts int
	// MaxTotalWait is the upper bound on ACCUMULATED sleep across all waits. A wait
	// that would push the running total past this is not taken — the executor gives
	// up instead (no truncated sleep).
	MaxTotalWait time.Duration
	// FallbackBackoff is the wait used for a 429 whose Retry-After is unusable.
	FallbackBackoff time.Duration
}

// DefaultRetryPolicy is the exported default value built from the named [ASSUMED]
// constants; the production wiring uses it.
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts:     defaultMaxAttempts,
	MaxTotalWait:    defaultMaxTotalWait,
	FallbackBackoff: defaultFallbackBackoff,
}

// isSafeMethod reports whether a request method is idempotent and therefore safe
// to auto-retry on a 429 (ADR-3): GET/HEAD are retryable, everything else
// (POST/PUT/PATCH/DELETE) surfaces the 429 on first occurrence so a
// non-idempotent operation is never silently re-sent. Matched against the
// project's uppercase net/http method constants.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

// parseRetryAfter parses the Retry-After header as a NON-NEGATIVE INTEGER number
// of seconds (the spec's RetryAfter schema is integer — spec/glassfrog-api-v5.yaml).
// It returns (duration, true) for a usable value ("0" is usable and means retry
// immediately) and (0, false) for an absent / empty / non-integer / negative /
// HTTP-date value — "unusable", so the caller falls back to FallbackBackoff. It
// never sleeps a garbage duration (parser-robustness, companion to the URL-parse
// LEARNINGS).
func parseRetryAfter(h http.Header) (time.Duration, bool) {
	v := h.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		// Non-integer (incl. the HTTP-date form this API does not produce) or a
		// negative count: unusable. Fall back to the bounded backoff.
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

// RetryExecutor decorates a *Client (010) with a bounded 429 retry loop ABOVE the
// Execute send seam (ADR-1) — never an http.RoundTripper inside the transport
// stack, so each attempt stays one timed Do bounded by client.Timeout and the
// sleep sits BETWEEN attempts, outside any single attempt's timeout. It holds the
// client, the policy, and two injected seams: sleep (prod binds time.Sleep; tests
// bind a recording fake that never blocks) and a progress io.Writer (prod binds
// the command's stderr; tests bind a buffer) — so the package reaches for no
// time.Sleep / os.Stderr directly (ADR-4).
type RetryExecutor struct {
	client   *Client
	policy   RetryPolicy
	sleep    func(time.Duration)
	progress io.Writer
}

// NewRetryExecutor is the pure constructor. It wraps a built *Client (010) with
// the retry policy and the two injected seams. client, sleep, and progress are
// required and must be non-nil; a nil seam is a wiring bug and panics (fail-fast,
// no nil-default — DECISIONS/PR #20). The package reaches for no time.Sleep /
// os.Stderr directly.
func NewRetryExecutor(client *Client, policy RetryPolicy, sleep func(time.Duration), progress io.Writer) *RetryExecutor {
	if client == nil {
		panic("apiclient.NewRetryExecutor: client must not be nil")
	}
	if sleep == nil {
		panic("apiclient.NewRetryExecutor: sleep must not be nil")
	}
	if progress == nil {
		panic("apiclient.NewRetryExecutor: progress must not be nil")
	}
	return &RetryExecutor{client: client, policy: policy, sleep: sleep, progress: progress}
}

// Execute is the send seam WITH retry — signature-identical to (*Client).Execute,
// so a caller swaps the bare client for the executor without changing the call
// site. Per attempt it calls client.Execute once (010: exactly one Do); on a 429
// *ResponseError for a safe method it honors the wait (Retry-After, else
// FallbackBackoff) and re-attempts within the policy caps; on any other outcome
// (success, transport, decode, auth, non-429 non-2xx, non-safe 429) it returns it
// UNCHANGED on the first occurrence. A surfaced 429 — attempts exhausted or the
// next wait would exceed the accumulated-wait budget — is the unchanged
// *ResponseError{429, Header, Body}; 017 does not classify it (ADR-5). Before each
// sleep it writes one secret-free progress note (the wait, the next attempt, the
// cap) to the injected writer.
func (e *RetryExecutor) Execute(reqCtx context.Context, req Request, out any) (*Response, error) {
	var waited time.Duration
	for attempt := 1; ; attempt++ {
		resp, err := e.client.Execute(reqCtx, req, out)

		// Anything that is not a 429 *ResponseError passes through unchanged:
		// success, transport, decode, auth, and every non-429 non-2xx status.
		var respErr *ResponseError
		if !errors.As(err, &respErr) || respErr.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}
		// A 429 on a non-safe (non-idempotent) method: surface on first occurrence,
		// never auto-retry (ADR-3).
		if !isSafeMethod(req.Method) {
			return resp, err
		}
		// Attempts exhausted: surface the raw 429 unchanged (ADR-2).
		if attempt >= e.policy.MaxAttempts {
			return resp, err
		}
		// Honor Retry-After (integer seconds), else fall back to the bounded backoff.
		wait, ok := parseRetryAfter(respErr.Header)
		if !ok {
			wait = e.policy.FallbackBackoff
		}
		// Taking this wait would push accumulated sleep past the budget: give up
		// rather than take a truncated sleep (ADR-2).
		if waited+wait > e.policy.MaxTotalWait {
			return resp, err
		}
		// One non-secret line per wait: the wait, the next attempt index, and the
		// cap. The *ResponseError carries only the response side; the X-Auth-Token
		// request header is never echoed (ADR-4 / CONSTITUTION II).
		fmt.Fprintf(e.progress, "rate limited; waiting %s before retry %d/%d\n", wait, attempt+1, e.policy.MaxAttempts)
		e.sleep(wait)
		waited += wait
	}
}
