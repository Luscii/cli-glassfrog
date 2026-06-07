# Validate: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/no-shared-api-client/rate-limit-handling.feature, PROJECT.md
**Implementation files**: 2 production (`internal/apiclient/retry.go`, `internal/cli/me.go`) + 3 test (`internal/apiclient/retry_test.go`, `internal/apiclient/rate_limit_handling_bdd_test.go`, `internal/cli/me_test.go`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 4 of 4 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All 8 driving scenarios are concretized in `rate-limit-handling.feature`, referenced by checked tasks T002/T004, and have identifiable code paths plus passing executable acceptance.

| Scenario | Status | Implementation |
|---|---|---|
| a single 429 is honored and the retry succeeds | ✓ Covered | `retry.go:RetryExecutor.Execute` (429 → `parseRetryAfter` → `sleep` → re-attempt → 200) |
| a non-429 outcome passes straight through | ✓ Covered | `retry.go:134` — `errors.As`/non-429 returns `resp, err` on first occurrence |
| a 429 without Retry-After uses the fallback backoff | ✓ Covered | `retry.go` — `parseRetryAfter` false → `wait = e.policy.FallbackBackoff` |
| caps reached — the 429 is surfaced unchanged | ✓ Covered | `retry.go` — `attempt >= MaxAttempts` and `waited+wait > MaxTotalWait` both return the raw `*ResponseError` |
| a transport error is not retried | ✓ Covered | `retry.go` — `*TransportError` is not a 429 `*ResponseError` → returned, one attempt |
| a non-safe request is not auto-retried on a 429 | ✓ Covered | `retry.go` — `!isSafeMethod(req.Method)` → 429 returned on first occurrence |
| a non-429 non-2xx is passed through, not retried | ✓ Covered | `retry.go` — `respErr.StatusCode != 429` → returned, no wait |
| a wait note is emitted to stderr before re-attempting | ✓ Covered | `retry.go` — `fmt.Fprintf(e.progress, …)` before each `sleep` |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — RetryPolicy + isSafeMethod + parseRetryAfter | ✓ Met | `retry.go`; `TestIsSafeMethod`, `TestParseRetryAfter` ((42s,true)/(0,true)/false for ""/"-1"/"abc"/HTTP-date), `TestDefaultRetryPolicyFloors` (4 / 60s / 2s) |
| T002 — RetryExecutor + NewRetryExecutor | ✓ Met | `retry.go`; 8 unit tests incl. attempts-cap tripwire (exactly `MaxAttempts` calls), total-wait-cap (no truncated sleep), non-safe POST (one attempt), passthrough table (200/403/transport/decode/auth, one call, no sleep), token-never-in-note, nil-seam panics |
| T003 — route runMe through RetryExecutor | ✓ Met | `me.go:112-114` (executor at send site, not bare `client.Execute`); `productionSeam.sleep()` binds `time.Sleep`; `cfg.stderr = cmd.ErrOrStderr()`; `TestRunMe_RetriesOn429ThenSucceeds`; `classifyClientError` unchanged |
| T004 — godog executable acceptance | ✓ Met | `rate_limit_handling_bdd_test.go` (5th apiclient suite, `Paths` names only its own file); 8 scenarios un-wipped + passing; all five suites report independent counts (auth 8, baseURL 10, context 8, rate-limit 8, request-execution 10); fake base + recording sleep + buffer (no real network/sleep/fs) |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Implementation |
|---|---|---|
| `NewRetryExecutor(client *Client, policy RetryPolicy, sleep func(time.Duration), progress io.Writer) *RetryExecutor` | ✓ Conformant | `retry.go` — signature exact; nil `client`/`sleep`/`progress` panic (fail-fast) |
| `(*RetryExecutor).Execute(reqCtx context.Context, req Request, out any) (*Response, error)` | ✓ Conformant | `retry.go` — signature-identical to `(*Client).Execute` |
| `RetryPolicy{MaxAttempts int; MaxTotalWait time.Duration; FallbackBackoff time.Duration}` | ✓ Conformant | `retry.go` — field shape exact |
| `DefaultRetryPolicy` (exported value, `[ASSUMED]` constants) | ✓ Conformant | `retry.go` — 4 / 60s / 2s from named constants |
| `isSafeMethod(method string) bool` | ✓ Conformant | `retry.go` — GET/HEAD true, else false |
| `parseRetryAfter(h http.Header) (time.Duration, bool)` | ✓ Conformant | `retry.go` — non-negative integer seconds; absent/negative/non-integer/HTTP-date → `(0,false)` |
| Outcome contract (same types; surfaced 429 unchanged; no new type) | ✓ Conformant | `retry.go` returns 010's `*Response`/`*ResponseError`; no `Outcome` edit |
| Error Communication table (2xx / retried-2xx / exhausted-429 / non-safe-429 / non-429 non-2xx / transport / decode / auth) | ✓ Conformant | All eight rows traced to the loop's return points |
| No command, no entry point (invocation/instructional N/A) | ✓ Conformant | No cobra command registered; progress note travels through the injected `io.Writer` |

---

## Non-Behavior Absence

**Status**: Pass (0 of 8 exclusions violated)

| Non-behavior | Status | Evidence |
|---|---|---|
| No proactive throttle on `X-RateLimit-Remaining` | ✓ Absent | Executor only reacts to a returned 429; never reads `X-RateLimit-Remaining` for pacing |
| No unbounded wait / no honoring Retry-After past budget | ✓ Absent | `waited+wait > MaxTotalWait` gives up before sleeping (no truncated sleep) |
| No auto-retry of a non-safe request on 429 | ✓ Absent | `!isSafeMethod` gate returns the 429 on first occurrence |
| No classify/interpret/rename of 429; no read/rewrite of body | ✓ Absent | Returns the unmodified `*ResponseError`; never inspects/edits `Body`; `clienterror.go` has no 429/RateLimited case |
| No initial send itself / no attach identity / no resolve URL / no decode | ✓ Absent | Delegates entirely to `e.client.Execute`; `out` passed through untouched |
| No retry on transport/decode/non-429 | ✓ Absent | Only a 429 `*ResponseError` triggers the retry branch |
| No token/full-request/secret in progress notes | ✓ Absent | Note carries only wait/attempt/cap; `thenNoSecretInNote` + `TestRetryExecutor_ProgressNoteNamesWaitAndAttemptNoSecret` pin it |
| No exit-code decision / no final user-facing message | ✓ Absent | Executor returns the error; `classifyClientError` (011, unchanged) maps a capped-out 429 → `APIError(3)`; 017 adds no `ExitCode`/`Outcome` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 8 behavioral scenarios (referenced by checked tasks T002/T004) have their `@wip` tags removed and run in the suite. The 4 `@validation` scenarios correctly retain `@validation @wip` — held out for this validation pass per T004's scope, not stranded work.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| every send goes through the 010 seam | ✓ Satisfied | `retry.go:134` — the only send is `e.client.Execute`; no `http.NewRequest`/`RoundTrip`/`Do` exists in the executor. Base-call-count tests confirm N attempts = N seam calls. |
| total wait is bounded regardless of Retry-After size | ✓ Satisfied | `waited+wait > MaxTotalWait` returns the last 429 without sleeping further; `TestRetryExecutor_TotalWaitBudgetGivesUpWithoutTruncatedSleep` asserts total ≤ budget and exactly one wait taken |
| only 429 triggers a retry | ✓ Satisfied | `if !errors.As(err, &respErr) || respErr.StatusCode != 429 { return resp, err }`; `TestRetryExecutor_NonRetryOutcomesPassThroughOnce` (200/403/transport/decode) + `TestRetryExecutor_AuthErrorPassesThrough` confirm one attempt, no sleep |
| the surfaced 429 is the raw outcome, untyped | ✓ Satisfied | Returns the unmodified `*ResponseError`; no rate-limit error type defined; `TestRetryExecutor_AttemptsExhaustedSurfacesRaw429` asserts status + rate-limit headers + body intact |

**Test execution note**: the 4 `@validation` scenarios retain `@wip` and have no step definitions, so godog does not execute them — verification is inspection-based per the validate baseline. Supplementary confidence comes from the unit tests cited above, which pin the same four properties and pass.

---

## Verdict: Ready

All 5 conformance dimensions pass with 0 findings. All 4 held-out validation scenarios are satisfied through independent inspection, corroborated by passing unit tests. The implementation conforms to the specification: it retries only safe-method 429s, honors `Retry-After` (else a bounded fallback), caps both attempts and accumulated wait without a truncated sleep, surfaces the raw `*ResponseError` unchanged (no classification — ADR-5), emits a secret-free per-wait note through the injected writer, and sends exclusively through the 010 `Execute` seam. The full test suite is green; no real network, sleep, or filesystem is touched.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 017 is closed.

Note for a future cycle: the surfaced (capped-out) 429 currently maps to `APIError(3)`, not the reserved `RateLimited(5)` — this is the deliberate ADR-5 boundary, picked up by API Error Extraction (015) when that capability lands.
