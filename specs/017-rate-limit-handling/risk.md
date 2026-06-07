# Risk: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Round**: 1
**Date**: 2026-06-07
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A retry storm amplifies the throttle (re-sending too fast worsens the org-wide rate limit) | spec Behavioral Accord (react to 429); CONSTITUTION X | High | Low | RC-1 | Yellow |
| H-2 | An unbounded wait hangs the command / automated run | spec Non-Behaviors (no unbounded wait); plan ADR-2 | Medium | Low | RC-2 | Green |
| H-3 | A non-idempotent write is silently re-sent on a 429 (double-apply) | spec Non-Behaviors (no write-retry); plan ADR-3; CONSTITUTION III/IX | High | Low | RC-3 | Yellow |
| H-4 | A non-429 error is masked / the capped-out 429 reported as success (Fail-Safe breach) | spec Behavioral Accord (pass-through); CONSTITUTION III | Medium | Low | RC-4 | Green |
| H-5 | The token leaks into the progress note or a surfaced error | plan § Cross-cutting (secret hygiene); CONSTITUTION II | High | Low | RC-5 | Yellow |
| H-6 | A malformed `Retry-After` causes a garbage/negative sleep or a crash | interface § Internal helpers (`parseRetryAfter`); LEARNINGS parser-robustness | Medium | Low | RC-6 | Green |
| H-7 | A rate-limit exhaustion exits as generic `APIError(3)`, not the reserved code 5 (agent can't distinguish) | plan ADR-5; checklist P2; CONSTITUTION II/X | Low | Medium | RC-7 | Green |
| H-8 | A cap/backoff test sleeps for real and hangs CI | plan § Cross-cutting/Risks; CONSTITUTION IV | Low | Low | RC-8 | Green |

No residual risk is Red.

---

## Hazard Detail

### H-1 — Retry storm amplifies the throttle
A retry loop that re-sends faster than the API allows (ignoring or mis-reading `Retry-After`) hammers an already-throttled org and *deepens* the very rate limit it is trying to ride out — the org-wide harm CONSTITUTION X guards against. **Severity High**: throttling affects the whole organization, not just this call. **Probability Low**: 017's design honors the API's `Retry-After` interval verbatim, makes at most `MaxAttempts` attempts, and only retries safe reads — there is no faster-than-instructed or unbounded re-send.
- **RC-1**: The loop waits the `Retry-After` interval (bounded fallback when absent) before each re-attempt, caps the attempt count, and never retries below the API's requested interval — so request volume after a 429 stays within what the API asked for (spec Behavioral Accord "Reacting to a 429"; plan ADR-2; feature "A 429 is honored and the retry succeeds").
- **Residual: Yellow** (High×Low) — acceptable: the API's own `Retry-After` paces every re-send and the attempt cap bounds the total; honoring the server signal is the intended X behavior.

### H-2 — Unbounded wait hangs the command
Honoring a far-out `Retry-After` (a window resetting up to an hour out), or accumulating many waits, could block a command — or an automated agent run — indefinitely. **Severity Medium**: a hung invocation stalls automation but does not corrupt data. **Probability Low**: ADR-2 caps **both** the attempt count and the accumulated wait, and gives up (surfacing the raw 429) rather than taking a truncated sleep when the next wait would exceed the budget.
- **RC-2**: Dual caps — `MaxAttempts` and `MaxTotalWait` — bound every call to a finite time; a `Retry-After` larger than the remaining budget triggers give-up, not a sleep (spec Behavioral Accord "Bounding the wait"; plan ADR-2; validation scenario "Total wait stays within the budget regardless of Retry-After size").
- **Residual: Green** (Medium×Low) — the bounded-wait property is structural and pinned by the total-wait-cap tripwire test.

### H-3 — Non-idempotent write silently re-sent
Auto-retrying a `429` on a write (`POST`/`PATCH`/`DELETE`) could re-apply a mutation that the API actually processed before answering 429 — a double-applied governance change, the partial/duplicate-state harm CONSTITUTION III and IX guard against. **Severity High**: a duplicated mutation corrupts live governance data. **Probability Low**: ADR-3 gates retry on `isSafeMethod` (`GET`/`HEAD` only); a `429` on a non-safe method is surfaced on first occurrence and never re-sent — and the current surface is all reads (no write caller exists yet).
- **RC-3**: The `isSafeMethod` gate restricts auto-retry to idempotent reads; a non-safe `429` returns the unchanged `*ResponseError` on the first attempt, never re-sending (spec Non-Behaviors; plan ADR-3; feature "A write is not auto-retried on a 429", pinned by a `POST`-429 exactly-one-attempt test).
- **Residual: Yellow** (High×Low) — acceptable with the hard safe-method gate and its dedicated test; the high severity keeps it above Green despite the low probability, so the gate must not be relaxed without revisiting this hazard.

### H-4 — Masked error / false success
A retry loop is a classic place to accidentally swallow a non-429 error (returning a stale success) or to report the final capped-out 429 as a success. **Severity Medium**: a hidden failure misleads the operator (Fail-Safe, III). **Probability Low**: the design returns every non-429 outcome unchanged on the first occurrence and surfaces the capped-out 429 as the unchanged `*ResponseError` — never as success.
- **RC-4**: Non-429 outcomes (success, transport, decode, other non-2xx, `*AuthError`) pass through unchanged; the exhausted 429 is surfaced as the raw error, not a success; a nil seam fails fast rather than degrading silently (spec Behavioral Accord "Passing non-rate-limit outcomes through" + "Bounding the wait"; plan ADR-2/ADR-4; validation scenario "Only a 429 triggers a retry").
- **Residual: Green** (Medium×Low).

### H-5 — Token leakage on the retry path
017 sits on the live request path and emits a stderr note — both places a secret could leak (CONSTITUTION II). **Severity High**: token disclosure. **Probability Low**: 017 never reads the token (it rides 007's `AuthTransport`, wired in 010, below this layer); the progress note carries only timing, and the `*ResponseError` it inspects holds only the *response* side (the `X-Auth-Token` request header is not echoed).
- **RC-5**: 017 reads no credential; the progress note names only the wait/attempt/cap; a token-never-in-output test covers the note and every surfaced error (plan § Cross-cutting; interface § Error Communication "No secret anywhere"; T002 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path control and the explicit output test (mirrors 010/011/012's token-leak posture).

### H-6 — Malformed `Retry-After` mishandled
A `429` could carry a `Retry-After` that is negative, non-integer, or an HTTP-date — which, if mishandled, could produce a negative/garbage sleep duration or a parse panic. **Severity Medium**: a wrong sleep (instant retry storm, or a crash). **Probability Low**: `parseRetryAfter` parses non-negative integer seconds only (the spec's `RetryAfter` schema) and returns "unusable" on anything else, so the caller falls back to the bounded `FallbackBackoff`; the bad inputs are unit-tested.
- **RC-6**: `parseRetryAfter` accepts only a non-negative integer; absent/negative/non-integer/HTTP-date → fallback backoff (never a negative or unbounded sleep), pinned by unit tests over each bad input (interface § Internal helpers; plan ADR-2; T001 acceptance). Companion to the URL-parse-robustness LEARNINGS.
- **Residual: Green** (Medium×Low).

### H-7 — Exhaustion exits 3, not the reserved 5
A capped-out 429 is surfaced as a generic `*ResponseError` → `APIError(3)`; an agent keying on exit codes cannot distinguish a rate-limit exhaustion from any other API error until API Error Extraction (015) types it (ADR-5). **Severity Low**: the exit code is coarser, not *wrong* — and the stderr retry notes already make the rate-limiting legible to the operator. **Probability Medium**: this occurs on every exhaustion, which is the expected outcome whenever the cap is genuinely hit.
- **RC-7**: The per-wait stderr notes convey the rate-limiting at the operator level (partial Action Transparency); exit-code-level classification (`429 → RateLimited(5)`, code reserved by 004) is deliberately deferred to 015, the capability that types non-2xx responses (plan ADR-5; checklist P2). 017 surfaces the raw 429 so 015 can type it without 017 re-deciding.
- **Residual: Green** (Low×Medium) — accepted cross-spec deferral; revisit when 015 lands. Same shape as the 429-classification deferral 011/012 recorded, now one layer closer to closure.

### H-8 — Test hangs on real sleep
A cap/backoff test that calls the real `time.Sleep` would block CI for the (possibly large) `Retry-After` durations under test. **Severity Low**: CI slowness, never runtime safety. **Probability Low**: the `sleep` is an injected seam; tests bind a non-blocking recording fake (ADR-4).
- **RC-8**: `NewRetryExecutor` takes an injected `sleep func(time.Duration)`; tests bind a recording fake that asserts durations without blocking; production binds `time.Sleep` (plan ADR-4; T002/T004 acceptance — "no real sleep").
- **Residual: Green** (Low×Low).

---

## Residual Risk Summary

8 hazards, 8 controls, **0 Red**. Three Yellows — H-1 (retry storm), H-3 (write double-apply), H-5 (token leak) — are all High-severity × Low-probability, acceptable with their structural controls (server-paced `Retry-After` + attempt cap; the `isSafeMethod` gate; the no-token-path + output test). The dual-cap give-up (H-2), pass-through discipline (H-4), `parseRetryAfter` robustness (H-6), exit-code deferral (H-7), and injected-sleep seam (H-8) are Green. **Dependencies are landed** — 010 (the `Execute` seam + generic `ResponseError`) and 011 (the `runMe` send site) are Complete on `main`, so there is no unbuilt-dependency hazard. The one cross-spec item to track is H-7: the reserved rate-limit exit code 5 lands with API Error Extraction (015), not here.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Behavioral Accord (Reacting to a 429); plan ADR-2; CONSTITUTION X |
| H-2 | spec § Non-Behaviors (no unbounded wait); plan ADR-2 |
| H-3 | spec § Non-Behaviors (no write-retry); plan ADR-3; CONSTITUTION III/IX |
| H-4 | spec § Behavioral Accord (pass-through); plan ADR-2/ADR-4; CONSTITUTION III |
| H-5 | plan § Cross-cutting (secret hygiene); interface § Error Communication; CONSTITUTION II |
| H-6 | interface § Internal helpers (`parseRetryAfter`); plan ADR-2 |
| H-7 | plan ADR-5; checklist.md P2; CONSTITUTION II/X |
| H-8 | plan § Cross-cutting / Risks; CONSTITUTION IV |
| RC-1 | spec accord + feature scenario; plan ADR-2 (honor `Retry-After` + attempt cap) |
| RC-2 | spec accord + validation scenario; plan ADR-2 (dual caps, give-up not truncated sleep) |
| RC-3 | spec Non-Behaviors; plan ADR-3 (`isSafeMethod` gate); `POST`-429 one-attempt test |
| RC-4 | spec accord; plan ADR-2/ADR-4 (pass-through; fail-fast on nil) |
| RC-5 | plan § Cross-cutting (no-token-path, token-never-in-output test); interface § Error Communication |
| RC-6 | interface § Internal helpers (`parseRetryAfter` non-negative-integer-only); plan ADR-2 |
| RC-7 | plan ADR-5 (surfaced 429 untyped; stderr notes; code 5 → 015) |
| RC-8 | plan ADR-4 (injected `sleep` seam; recording fake in tests) |
