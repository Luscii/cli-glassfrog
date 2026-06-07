# Tasks: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-shared-api-client/rate-limit-handling.feature

---

## Dependency Graph

Phase 1: Retry policy + `isSafeMethod` + `parseRetryAfter` in `internal/apiclient` (1 task, no phase dependencies) [Shared]
Phase 2: `RetryExecutor` — the bounded retry loop over `(*Client).Execute` with injected seams (1 task, depends on Phase 1) [Shared]
Phase 3: Wire the `me` read path through the executor + executable acceptance via godog (2 tasks, depend on Phase 2 — parallel with each other) [Shared]

4 tasks total | Phase 3's two tasks parallelizable | Builder: pipeline

> Every task is `[Shared]`: the retry layer is infrastructure serving all three user scenarios (ride-out-transient-throttles / bounded-wait-then-surface / announce-each-wait-on-stderr) rather than any single one.
>
> **Cross-spec note**: this slice is purely additive in `internal/apiclient` (a new file plus its tests); the only edit to landed code is routing 011's `runMe` send through the executor (T003). It depends on landed code on main: 010's `Client`, `(*Client).Execute`, `Request`, and the generic `ResponseError{StatusCode, Header, Body}` (which already carries a 429's `Retry-After`/`X-RateLimit-*`); and 011's `runMe` send site + `classifyClientError`. It realizes 010's "017 layers backoff above the seam" (ADR-1). It registers **no cobra command**, decides **no exit code**, and adds **no `Outcome` category / no `classifyClientError` edit**: a surfaced (capped-out) 429 stays a generic `*ResponseError` → `APIError(3)` unchanged, and the `429 → RateLimited(5)` split (code 5 reserved by 004) is API Error Extraction (015)'s job (ADR-5). `apiclient` still never imports `internal/cli`. No new `.glassfrogrc` key, env var, or flag — only the `RetryPolicy` `[ASSUMED]` constants.

---

## Branching Guidance

**Pipeline mode**: `spec/017-rate-limit-handling/base` → `spec/017-rate-limit-handling/task-1`, `…/task-2`, `…/task-3`, `…/task-4` (one task branch per T-id, merged back into the spec base). T003 and T004 touch different files and can run on parallel branches off the Phase 2 result.

**Parallel-spec awareness**: 001–011 are Complete (010's seam and 011's `me` read path are landed on main — 017's dependencies). 012–014 are specified (Analyzed) but not yet implemented; 015 (API Error Extraction) and 016 (Pagination) are sibling Should-tier specs not present in this workspace (parallel branches). 017 builds only on landed 010/011 and does not block on the unbuilt siblings.

---

## Phase 1: Retry policy + `isSafeMethod` + `parseRetryAfter` [Shared]

- [x] **T001** [Shared] Add the `RetryPolicy` value (+ `DefaultRetryPolicy` constants), the `isSafeMethod` predicate, and the `parseRetryAfter` header parser — RED-first unit tests — `retry.go`; unit tests for both predicates + parser cases + policy floors
  - **Scope**: In `internal/apiclient` (a new file, e.g. `retry.go`), define the code-free `RetryPolicy{ MaxAttempts int; MaxTotalWait time.Duration; FallbackBackoff time.Duration }` and an exported `DefaultRetryPolicy` value built from named `[ASSUMED]` constants (exact values deferred; `MaxAttempts ≥ 2`, positive `MaxTotalWait`/`FallbackBackoff`). Add `isSafeMethod(method string) bool` — `GET`/`HEAD` true, everything else false (idempotency, ADR-3). Add `parseRetryAfter(h http.Header) (time.Duration, bool)` — parse `Retry-After` as a **non-negative integer number of seconds** (the spec's `RetryAfter` schema is `integer`); return `(0, false)` for absent / empty / non-integer / negative / HTTP-date form (caller falls back to `FallbackBackoff`). No I/O, no sleeping, no transport here.
  - **Acceptance criteria**:
    - `isSafeMethod` returns true for `GET`/`HEAD` and false for `POST`/`PUT`/`PATCH`/`DELETE` (case per the project's method constants)
    - `parseRetryAfter` returns `(42s, true)` for `"42"`, `(0, true)` for `"0"`, and `(0, false)` for `""`, `"-1"`, `"abc"`, and an HTTP-date string
    - `DefaultRetryPolicy` exposes `MaxAttempts ≥ 2` and positive `MaxTotalWait` and `FallbackBackoff` from named `[ASSUMED]` constants
    - RED-first unit tests cover each predicate/parse case; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (additive in `internal/apiclient`)
  - **Plan reference**: Phase 1; ADR-2 (honor `Retry-After`, fallback, caps), ADR-3 (safe-method gate)
  - **Interface references**: interface-spec.md — Configuration contract `RetryPolicy`, Internal helpers (`isSafeMethod`, `parseRetryAfter`)
  - **Scenario references**: rate-limit-handling.feature: "A 429 without a usable Retry-After uses the fallback backoff", "A write is not auto-retried on a 429"
  - **Risk**: ⚠️ Parser robustness — treat negative / non-integer / HTTP-date `Retry-After` as **unusable** (fall back to backoff), never crash or sleep a garbage duration (companion to the URL-parse-robustness LEARNINGS). Integer seconds only, per the spec schema.

## Phase 2: `RetryExecutor` — the bounded retry loop over `Execute` [Shared]

- [x] **T002** [Shared] Implement `RetryExecutor` + `NewRetryExecutor`: wrap `(*Client).Execute` in a bounded `429` retry loop with injected `sleep` + progress seams, surfacing the raw 429 at the caps — RED-first unit tests over a fake base + recording sleep + buffer — `retry.go`; 8 unit tests incl. attempts-cap + total-wait-cap tripwires, non-safe-method, passthrough table, token-never-in-note, nil-seam panics
  - **Scope**: In `internal/apiclient`, add `RetryExecutor` (holds the `*Client`, a `RetryPolicy`, an injected `sleep func(time.Duration)`, and an `io.Writer` progress sink) and `NewRetryExecutor(client *Client, policy RetryPolicy, sleep func(time.Duration), progress io.Writer) *RetryExecutor` — `client`/`sleep`/`progress` required and non-nil; a nil seam panics (fail-fast, no nil-default — DECISIONS/PR #20). Add `(*RetryExecutor).Execute(reqCtx context.Context, req Request, out any) (*Response, error)` — **signature-identical** to `(*Client).Execute`. The loop (plan System Architecture): for each attempt up to `MaxAttempts`, call `client.Execute`; if the error is not a `429` `*ResponseError` (`errors.As` + `StatusCode == 429`) → return it unchanged; if `!isSafeMethod(req.Method)` → return the 429 unchanged; if this was the last attempt → return the 429 unchanged; else compute `wait` (`parseRetryAfter`, else `FallbackBackoff`); if `accumulated + wait > MaxTotalWait` → return the 429 unchanged (give up, **no truncated sleep**); else write the secret-free progress note, `sleep(wait)`, add to the accumulator, loop. The package reaches for no `time.Sleep`/`os.Stderr`. Add **no** new outcome type and **no** classification — a surfaced 429 is the unchanged `*ResponseError` (ADR-5).
  - **Acceptance criteria**:
    - 429-then-200 for a `GET` → waits the `Retry-After` interval (assert the recorded sleep equals the header), re-attempts, returns the 200 `Response` with `out` populated
    - A 429 carrying no usable `Retry-After` → the recorded wait equals `FallbackBackoff`
    - Any non-429 outcome (200, 403/other non-2xx, `*TransportError`, `*DecodeError`, `*AuthError`) → returned unchanged with **exactly one** `Execute` call and **no** sleep
    - Always-429 `GET` → **exactly `MaxAttempts`** `Execute` calls (call-counting tripwire), then the most recent `*ResponseError{429,…}` returned unchanged, not classified
    - `Retry-After`s that together exceed `MaxTotalWait` → gives up within budget, total recorded sleep ≤ `MaxTotalWait`, no sleep beyond the cap, the last 429 returned
    - A `POST` 429 → **exactly one** attempt, the 429 returned on first occurrence, not re-sent
    - A progress note is written to the buffer naming the wait and the next attempt; no token/secret appears in the note or any surfaced error (token-never-in-output assertion)
    - A nil `client`/`sleep`/`progress` panics (documented precondition); RED-first unit tests over a fake base `RoundTripper` + a recording fake-sleep (never blocks) + a buffer; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-1 (retry above the seam, not a transport), ADR-2 (honor/fallback/caps), ADR-3 (safe gate), ADR-4 (injected `sleep`+writer, fail-fast on nil, secret-free note), ADR-5 (no classification; surfaced 429 stays `*ResponseError`); Cross-cutting (secret hygiene, error handling, observability, testing)
  - **Interface references**: interface-spec.md — Entry points (`NewRetryExecutor`, `(*RetryExecutor).Execute`), Outcome contract, Interactions (retry loop, honoring `Retry-After`, eligibility, progress note, per-attempt timeout boundary), Error Communication
  - **Scenario references**: rate-limit-handling.feature: "A 429 is honored and the retry succeeds", "A response that is not a 429 is returned without waiting", "A 429 without a usable Retry-After uses the fallback backoff", "A transport error is returned without a retry", "A non-429 non-2xx response is passed through without retrying", "A write is not auto-retried on a 429", "The 429 is surfaced when attempts are exhausted", "A progress note is written to stderr before re-attempting", and the four `@validation` scenarios
  - **Risk**: ⚠️ Keep retry **above** `Execute` — never sleep inside a transport, which would count against `client.Timeout` (ADR-1). ⚠️ Inject `sleep` so cap/backoff tests assert durations without blocking (no real `time.Sleep` in tests — CONSTITUTION IV). ⚠️ Do **not** classify the 429 (no new error type, no `Outcome` edit — that is 015's, ADR-5). ⚠️ Bound **accumulated sleep** and give up rather than take a truncated sleep when the next wait would exceed the budget.

## Phase 3: Wire the read path + executable acceptance [Shared]

- [ ] **T003** [P] [Shared] Route 011's `runMe` send through the retry executor, threading `time.Sleep` and the command's stderr in production — RED-first unit tests over a fake transport + fake sleep
  - **Scope**: In `internal/cli/me.go`, route `runMe`'s single `client.Execute(cfg.reqCtx, req, &me)` through `apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, sleep, cfg.stderr).Execute(...)`. Thread the `sleep func(time.Duration)` seam into `meConfig`/`meSeam` so it is injectable: the `productionSeam` binds `time.Sleep` and the command binds `cmd.ErrOrStderr()` as the progress sink (the existing `cfg.stderr`); tests bind a recording fake-sleep + buffer. `classifyClientError` is **unchanged** (a surfaced 429 stays `APIError(3)` — ADR-5). The projection rendering and every other branch stay as-is.
  - **Acceptance criteria**:
    - `runMe` sends through the `RetryExecutor`, not the bare `client.Execute`; the production seam binds `time.Sleep` and the command's stderr
    - A `me` run whose transport returns 429-then-200 renders the identity projection after one bounded wait (fake sleep — no real delay), exit `Success`
    - A `me` run with a non-429 outcome (200 success, 403, transport error, no-token) behaves exactly as before — `classifyClientError` mapping unchanged (403/other non-2xx → `APIError(3)`, including a capped-out 429)
    - RED-first unit tests over the existing fake-transport `meSeam` plus a recording fake-sleep; `go build`/`go vet` clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 (wire the read path); Integration Design (Identity Read 011 consumer); ADR-4 (prod binds `time.Sleep` + stderr), ADR-5 (`classifyClientError` unchanged)
  - **Interface references**: interface-spec.md — Interactions (build-once / wrap-once flow); Error Communication (no classification here)
  - **Scenario references**: rate-limit-handling.feature: "A 429 is honored and the retry succeeds" (end-to-end through `me`)
  - **Risk**: ⚠️ The `sleep` seam must be **injectable** into `runMe` (not a direct `time.Sleep`) or the `me` retry test sleeps for real. ⚠️ Do not edit `classifyClientError` or add an `Outcome` category (ADR-5). ⚠️ This is the one edit to landed 011 code — keep it additive (route the send + thread the seams), changing no other `me` branch; future reads (012–014) adopt the same wrapper at their send site.

- [ ] **T004** [P] [Shared] Make the 017 driving scenarios pass as executable acceptance via godog, driving `NewRetryExecutor`/`Execute` over a fake base + recording sleep in a new suite scoped to this spec's own feature file
  - **Scope**: Add godog step definitions for `features/no-shared-api-client/rate-limit-handling.feature` (all three Rule blocks), in a **new** `internal/apiclient` godog suite (`TestRateLimitHandlingFeatures`) whose `Paths` names **only** that feature file — the package now has five suites (007/008/009/010 + this), each pointed at its specific file (LEARNINGS: a suite must point at its own file, never the `features/` directory). Drive `NewRetryExecutor`/`Execute` over a **fake** base `http.RoundTripper` returning the canned 429/200/403/transport sequences + a **recording fake-sleep** (never blocks) + a **buffer** progress sink. Step helpers return errors, never panic (LEARNINGS). **Reuse existing step phrasing** where an assertion already exists — grep the package's `sc.Step(` registrations before adding bindings. Remove `@wip` from the 8 behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 017 scenario (429-honored-then-success / non-429-no-wait / no-Retry-After-fallback / transport-not-retried / non-429-non-2xx-passthrough / write-not-retried / attempts-exhausted-surfaces-429 / stderr-progress-note) has an executable, passing path over the fake base + recording sleep + buffer
    - `@wip` removed from those 8 scenarios; the four `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `rate-limit-handling.feature`; all five `apiclient` suites run and report their own independent `N scenarios (N passed)` counts
    - No real network beyond loopback `httptest`, **no real sleep**, and no real home/filesystem are touched (fake base + recording sleep + buffer only); `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 (executable acceptance); Cross-cutting Concerns (testing strategy)
  - **Scenario references**: rate-limit-handling.feature: all 017 behavioral Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — a fifth feature file in the `apiclient` package must keep every suite pointed at specific files (not the directory), or un-wipping one spec's scenarios breaks another suite; verify all five report their own counts. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings (LEARNINGS); step helpers return errors, never panic. ⚠️ Bind a recording fake-sleep so the suite never waits real seconds.
