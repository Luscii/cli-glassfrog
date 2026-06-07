# Specification: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Rate-Limit Handling is a downstream sibling in the **API Client** solution (problem: *No Shared API Client* — without shared transport conventions, every endpoint command would reinvent paging, error, and rate-limit plumbing). The Glassfrog API enforces a **per-organization rolling 1-hour rate limit** that varies by plan; when it is exceeded the API answers `429 Too Many Requests` with a `Retry-After` header (whole seconds) and the rate-limit headers (`X-RateLimit-Limit` / `-Remaining` / `-Reset`) it attaches to every response. Request Execution (010) makes exactly one send attempt and surfaces a `429` as a generic non-2xx outcome carrying its status, headers, and body — it never sleeps or retries. **This capability owns that retry.**

It composes *around* the 010 send seam: hand it a request, and on a `429` it waits the interval the API asks for and re-attempts, within a hard cap on both the number of attempts and the total time spent waiting. Any non-`429` outcome — success, transport error, or any other non-2xx — passes straight through unchanged. It does not classify or interpret the `429` into a meaningful typed error; that stays with API Error Extraction (015). Its single job is to honor the API's "wait then try again" signal so a momentary throttle becomes a brief pause rather than a failed command — while never blocking unboundedly.

---

## Behavioral Accord

### Passing non-rate-limit outcomes through

- When the wrapped send returns any outcome that is not a `429` — a success, a transport error, a decode outcome, or any other non-2xx status — the system returns that outcome unchanged, having made no additional attempt and imposed no wait.

### Reacting to a 429

- When the wrapped send returns a `429`, the system reads the `Retry-After` header to determine how long the API asks it to wait, and — provided the request is eligible to retry and the wait fits within the remaining caps — waits that interval and then re-attempts the same request through the same send seam.
- When a `429` carries no usable `Retry-After` value (absent, non-numeric, or empty), the system falls back to a bounded backoff interval for that attempt rather than treating the response as un-retryable.
- When a re-attempt itself returns a `429`, the system repeats the same decision (read the wait, check the caps, wait, re-attempt) until it gets a non-`429` outcome or a cap is reached.

### Bounding the wait

- When honoring a `429` would require waiting longer than the remaining total-wait budget, or would exceed the maximum number of attempts, the system stops retrying and surfaces the most recent `429` outcome unchanged — carrying its status, its rate-limit headers, and its body — without sleeping any further.
- When attempts are exhausted on a `429`, the system surfaces that same raw `429` outcome rather than synthesizing a different error; deciding what the surfaced `429` *means* is left to the caller and to API Error Extraction (015).

### Eligibility

- When the request being sent is not a safe (read) request, the system does not auto-retry on a `429`; it surfaces the `429` outcome unchanged on the first occurrence so a non-idempotent operation is never silently re-sent.

### Observability while waiting

- When the system waits before a re-attempt, it emits a non-secret progress note to the standard error stream stating that it is rate-limited and roughly how long it will wait (and which attempt is next), so an operator — human or agent — sees a deliberate pause rather than a silent hang.
- The system never prompts interactively and never blocks on input; a pause is bounded by the caps and proceeds on its own.

---

## User Scenarios

**In order to** have a command ride out a brief per-org throttle instead of failing the moment the API says "slow down,"
**as a** practitioner running a sequence of reads,
**I want** the CLI to honor the API's `Retry-After` and re-attempt automatically, within sane bounds.

**In order to** never have an automated run sleep for an unbounded stretch when the window is far from resetting,
**as an** operator (human or AI agent) driving the CLI,
**I want** the wait capped in both attempts and total time, with the `429` surfaced once the cap is hit.

**In order to** know the command is deliberately pausing rather than hung,
**as an** operator watching a command,
**I want** a short, non-secret note on stderr each time it waits before retrying.

---

## Non-Behaviors

- The system must not proactively throttle or pre-emptively pause based on `X-RateLimit-Remaining` nearing zero. **Why**: proactive pacing needs cross-request shared state that doesn't fit a per-invocation CLI; reacting to the `429` the API actually returns keeps the behavior local and predictable. Proactive pacing is a candidate for a later capability, not this one.
- The system must not wait an unbounded amount of time, nor honor a `Retry-After` that exceeds its total-wait budget. **Why**: a rolling window can reset up to an hour out; sleeping that long would hang a command or an automated run indefinitely. The caps guarantee the call always returns in bounded time.
- The system must not auto-retry a non-safe (write) request on a `429`. **Why**: re-sending a non-idempotent operation risks double-applying it; for writes the `429` is surfaced for the caller to decide, not silently retried.
- The system must not classify, interpret, or rename the `429` into a meaningful typed error, nor read or rewrite the error body. **Why**: API Error Extraction (015) owns turning a non-2xx into a typed, meaningful error; this capability only decides *whether and when to retry* and surfaces the raw outcome otherwise.
- The system must not make the initial send attempt itself or attach identity, resolve the base URL, or decode bodies. **Why**: Request Execution (010) owns sending one authenticated request and surfacing its outcome; this capability wraps that seam and re-invokes it, never duplicating it.
- The system must not retry on transport errors, decode errors, or non-`429` statuses. **Why**: only a `429` carries the API's explicit "wait then retry" contract; retrying other failures would mask real errors and is out of this capability's concern.
- The system must not emit the token, the full request, or any secret in its progress notes. **Why**: the wait note is printed on the request path where secrets leak; the same secret-never-emitted rule that governs Request Execution (010) and Request Authentication (007) applies here.
- The system must not decide the process exit code or the final user-facing message. **Why**: Exit-Code Convention (004) and the consuming command own classification of the surfaced outcome.

---

## Integration Boundaries

- **Request Execution (010 — upstream dependency)**: provides the single-attempt send seam. This capability calls it, inspects the outcome, and re-invokes it for each retry. 010 surfaces the `429` as a generic non-2xx carrying status, the rate-limit headers, and the body; this capability reads `Retry-After` (and the rate-limit headers) from that outcome and never sends a request except through 010.
- **API Error Extraction (015 — downstream sibling)**: classifies whatever non-2xx outcome finally survives — including a `429` this capability gave up retrying — into a typed, meaningful error. Composition order is: 010 sends → 017 retries/backs off → 015 classifies the final outcome. This capability surfaces the raw `429` onward; it does not type it.
- **Pagination (016 — sibling)**: walks multi-page reads by issuing more requests; each such request can itself be rate-limited, so paging composes with this capability's per-request retry. Neither owns the other; both sit on the 010 seam.
- **Standard error stream (operator-facing surface)**: the progress note on each wait flows to stderr. No other output surface is touched.
- **Exit-Code Convention (004 — downstream)**: the surfaced outcome ultimately informs the command's exit code, but the mapping belongs to that capability and the consuming command.

---

## Driving Scenarios

### Happy path

**Scenario: a single 429 is honored and the retry succeeds**
Given a safe request whose first send returns a `429` with `Retry-After: 2`
When the request is sent through this capability
Then the system waits about 2 seconds, re-attempts the same request through the send seam, and returns the subsequent success outcome unchanged.

**Scenario: a non-429 outcome passes straight through**
Given a safe request whose first send returns a `200` success
When the request is sent through this capability
Then the system returns the success unchanged, having imposed no wait and made no extra attempt.

**Scenario: a 429 without Retry-After uses the fallback backoff**
Given a safe request whose first send returns a `429` with no usable `Retry-After`
When the request is sent through this capability
Then the system waits a bounded fallback interval, re-attempts, and returns the subsequent outcome.

### Error scenarios

**Scenario: caps reached — the 429 is surfaced unchanged**
Given a safe request that keeps returning `429` past the maximum attempts (or whose required wait would exceed the total-wait budget)
When the request is sent through this capability
Then the system stops retrying, sleeps no further, and surfaces the most recent `429` outcome carrying its status, rate-limit headers, and body
And does not classify it into a typed error.

**Scenario: a transport error is not retried**
Given a safe request whose send returns a typed transport error
When the request is sent through this capability
Then the system returns that transport error unchanged and makes no additional attempt.

### Edge cases

**Scenario: a non-safe request is not auto-retried on a 429**
Given a non-safe (write) request whose first send returns a `429`
When the request is sent through this capability
Then the system surfaces the `429` outcome unchanged on the first occurrence and never re-sends the request.

**Scenario: a non-429 non-2xx is passed through, not retried**
Given a safe request whose send returns a `403` (or any non-`429` non-2xx)
When the request is sent through this capability
Then the system returns that non-2xx outcome unchanged for API Error Extraction (015) to classify, with no wait and no retry.

**Scenario: a wait note is emitted to stderr before re-attempting**
Given a safe request whose first send returns a `429` with `Retry-After: 5`
When the system decides to wait and re-attempt
Then a non-secret progress note is written to standard error indicating the pause and the next attempt
And the note contains no token or secret.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: every send goes through the 010 seam**
Given any number of attempts (initial plus retries)
When the request is handled
Then each attempt is made only through Request Execution's send seam
And this capability never constructs or sends a request by itself.

**Scenario: total wait is bounded regardless of Retry-After size**
Given a sequence of `429`s whose `Retry-After` values would sum past the total-wait budget
When the request is handled
Then the system returns within the bounded budget and surfaces the last `429`
And never sleeps beyond the cap.

**Scenario: only 429 triggers a retry**
Given the full range of outcomes (success, transport error, decode error, and each non-2xx status)
When each is returned by the send seam
Then a retry is attempted only for a `429`
And every other outcome is returned unchanged on the first attempt.

**Scenario: the surfaced 429 is the raw outcome, untyped**
Given attempts are exhausted on a `429`
When the outcome is surfaced
Then it carries the original status, rate-limit headers, and body
And no rate-limit-specific error type or message has been synthesized by this capability.

---

## Assumptions

- **Caps are planning-tunable** `[ASSUMED]`: a maximum attempt count and a maximum total-wait budget exist with sensible defaults. The exact values and whether they are configurable are planning details; the behavior — bounded retries, then surface the `429` — is fixed.
- **Fallback backoff** `[ASSUMED]`: when `Retry-After` is unusable, a bounded backoff interval is used. Its base, growth, and any jitter are planning details; the behavior (a bounded wait rather than no retry, and never exceeding the total-wait budget) is fixed.
- **Safe-request signal** `[ASSUMED]`: the capability can tell a safe (read) request from a non-safe one. The most likely signal is the HTTP method (GET/HEAD safe), but how eligibility is provided is a planning detail; the behavior (writes are not auto-retried) is fixed.
- **Retry-After units**: `Retry-After` is interpreted as whole seconds, per the API spec (`spec/glassfrog-api-v5.yaml`), not an HTTP-date.
- **Progress-note wording** `[ASSUMED]`: the exact text and format of the stderr note are adjustable at planning; the behavior (a non-secret note on each wait, on stderr, no prompt) is fixed.
- **Composition seam**: this capability wraps the 010 send and sits inside API Error Extraction (015), which classifies whatever outcome finally survives. The ordering is fixed; the wiring mechanism is a planning detail.

---

## Ambiguity Warnings

_None remaining — the five behavioral forks were resolved during the defining conversation: (1) reactive-only, no proactive throttling on `X-RateLimit-Remaining`; (2) honor `Retry-After` exactly, with a bounded fallback when it is absent; (3) cap both attempts and total wait, then surface the raw `429`; (4) a non-secret progress note on stderr while waiting, no interactive prompt, operator may be human or agent; (5) auto-retry scoped to safe reads, writes surface the `429` without re-sending. The remaining `[ASSUMED]` items are planning-time tuning details, not behavioral gaps._
