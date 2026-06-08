# Plan: Rate-Limit Handling

**Feature**: 017-rate-limit-handling
**Role**: Shaper
**Inputs**: spec.md (017-rate-limit-handling); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: `internal/apiclient` owns the client/transport — base URL, **timeouts, retries** — 008 ADR-1; 010's `(*Client).Execute` is the single send seam, makes **exactly one `Do`, no retry**, and surfaces a non-2xx as a generic `*ResponseError{StatusCode, Header, Body}` that **carries the 429 + rate-limit headers** so "017 reads them to back off" / "017 layers backoff **above** the seam" — 010 ADR-3/ADR-4; 011's shared `classifyClientError` maps `*ResponseError` → `APIError(3)` and notes "015/017 refine it later"; 004 froze code **5 = rate-limit** as a reserved constant whose `Outcome` category lands "when its producer exists"; inject seams + fail-fast-on-nil for the pure layer, OS seam degrades — 005/008/009/010 + PR #20/#30; the producer-classifies/consumer-maps split — 002/004/005/007/008/009/010/011); `.score/memory/LEARNINGS.md` (relevant: a godog suite points at its OWN feature file; step helpers return errors never panic; install a tripwire to assert a *negative* property — "exactly N attempts", "nothing slept"; an error message must not interpret a status a Non-Behavior reserves for a sibling — 2026-06-07 #33-r2). `.score/memory/DEPRECATION.md`: only the settled 007/009 seam note — nothing 017 supersedes. No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord with When/Then grouped by concern, 3 happy + 2 error + 3 edge driving scenarios, 8 non-behaviors with reasoning, integration boundaries naming 010/015/016/004 and the stderr surface, user scenarios, 6 `[ASSUMED]` planning-tuning items, no remaining ambiguities. Strong foundation. The five behavioral forks were resolved during `/score:define` (reactive-only; honor `Retry-After`; cap attempts **and** total wait then surface the raw 429; non-secret stderr progress note, no prompt; auto-retry scoped to safe reads). **No architectural unknown required a resolve conversation**: the placement (above the `Execute` seam), the injected-seam discipline, the no-exit-code-change boundary, and the `internal/apiclient` home are all fixed by DECISIONS precedent and the approved spec; the `[ASSUMED]` items (cap values, fallback-backoff shape, the executor/policy API names, the safe-method signal) are interface/detail-level, not behavioral gaps.

---

## System Architecture

Rate-Limit Handling is a Should-tier sibling in the **API Client** solution (problem: *No Shared API Client*). It adds the one thing 010 deliberately left out: when the Glassfrog API answers `429 Too Many Requests` (per-org rolling 1-hour window — `spec/glassfrog-api-v5.yaml:87`), wait the interval the API asks for and try again, within a hard cap. 010 makes **exactly one `Do` per `Execute`** and surfaces a 429 as a generic `*apiclient.ResponseError{StatusCode, Header, Body}` carrying the `Retry-After` / `X-RateLimit-*` headers; 011's `me` read sends through that seam. 017 is the retry loop **around** `Execute` — the layer 010 forecast ("017 layers backoff above the seam").

It is **purely additive in `internal/apiclient`** — a retrying executor plus a retry-policy value — and it routes the landed read path's send through that executor. It registers no cobra command and decides no exit code. It does **not** classify the surfaced 429 into a meaningful error or a rate-limit exit code: a given-up 429 stays a generic `*ResponseError` → `APIError(3)` via 011's unchanged `classifyClientError`; typing the non-2xx (the `429 → RateLimited(5)` split 004 reserved) is API Error Extraction (015)'s job, per the spec's Non-Behaviors (ADR-5).

The parts (all in `internal/apiclient`):

- **`RetryPolicy`** — a code-free config value: `MaxAttempts`, `MaxTotalWait`, and a `FallbackBackoff` base, all named `[ASSUMED]` constants with sane defaults (mirroring 010's `requestTimeout`). The "no retry" behavior is just `MaxAttempts == 1` — the same code path serves a future caller that opts out.
- **The retrying executor** (`RetryExecutor`, built by `NewRetryExecutor` — name pinned by interface-spec) — decorates a `*Client`, wrapping `(*Client).Execute` in a bounded loop. Its `Execute` method (signature-identical to the client's) takes the `Request` and optional decode target; the executor holds the client, the policy, and two **injected seams**: a `sleep func(time.Duration)` (so tests never really sleep — CONSTITUTION IV) and an `io.Writer` progress sink (the command's stderr in production; a buffer in tests — the package reaches for no `os.Stderr`). Per iteration:

```
(*RetryExecutor).Execute(reqCtx, req, out):    // executor holds client, policy, sleep, progress
  waited := 0
  for attempt := 1 .. policy.MaxAttempts:
      resp, err := client.Execute(reqCtx, req, out)        ── 010: exactly one Do, bounded by client.Timeout
      if err is not a 429 *ResponseError:  return resp, err  ── success / transport / decode / other non-2xx pass THROUGH
      if !isSafeMethod(req.Method):         return resp, err  ── writes: surface the 429 on first occurrence (ADR-3)
      if attempt == policy.MaxAttempts:     return resp, err  ── attempts exhausted → surface the raw 429 (ADR-2)
      wait := parseRetryAfter(respErr.Header)  or policy.FallbackBackoff   ── honor Retry-After (int seconds), else fallback
      if waited + wait > policy.MaxTotalWait: return resp, err        ── would exceed budget → give up, no further sleep
      fprintf(progress, "rate limited; waiting %s before retry %d/%d", wait, attempt+1, MaxAttempts)  ── stderr, no secret
      sleep(wait); waited += wait
```

- **`isSafeMethod(method)`** — `GET`/`HEAD` are retryable; anything else surfaces the 429 unretried (idempotency). Read off `Request.Method`.
- **`parseRetryAfter(header)`** — parses `Retry-After` as a non-negative integer number of seconds (the spec's `RetryAfter` schema is `integer`); absent / non-integer / negative → "unusable", caller falls back to `FallbackBackoff`.

The read seam (today 011's `runMe`, which calls `client.Execute(cfg.reqCtx, req, &me)` directly) routes that single call through the `RetryExecutor`, threading the `sleep` and stderr seams. Because the read surface shares one send shape, future reads (013/014 when built, and the rest of 012–017's pattern) adopt the same wrapper — the retry lives in one `apiclient` helper, not re-inlined per command (the `classifyClientError` "one place, never drifts" discipline). Each `Execute` remains one timed attempt; the sleep happens **between** attempts, outside any single attempt's `client.Timeout`.

---

## Architecture Decisions

### ADR-1: 017 is a retry loop *above* the `Execute` seam — a shared `apiclient` helper — not an `http.RoundTripper` in the transport stack

**Context**: 010 fixed `(*Client).Execute` as the single send seam, pinned "**exactly one `Do`, no retry**" with a call-counting tripwire, and set `client.Timeout` to bound one attempt. Its plan and the DECISIONS record repeatedly place 017 "**above** the seam." Retry could instead be a transport middleware (like 007's `AuthTransport`) layered into the client `NewClient` builds.

**Options considered**:
1. **A retrying executor above `Execute`** — a shared `apiclient` function that calls `client.Execute` in a bounded loop and inspects the typed `*ResponseError`. 010 is untouched; each attempt is one timed `Do`; the sleep sits between attempts, outside the per-attempt timeout. Reads the 429 + headers off exactly the value 010 designed to carry them.
2. **A `RetryTransport` `http.RoundTripper` in the client's stack** — retry below `Execute`, so no read-command change. Rejected: it breaks 010's one-`Do`-per-`Execute` invariant and its tripwire, forces editing 010's frozen `NewClient`, and — fatally — a `Retry-After` sleep *inside* a `RoundTrip` is counted against `client.Timeout`, so a legitimate multi-second backoff would trip the request timeout.
3. **Bake retry into `Execute` itself** — rejected: directly contradicts 010's "no retry" Non-Behavior and the same timeout problem; it also removes the caller's ability to opt out.

**Decision**: Option 1. `RetryExecutor` (built by `NewRetryExecutor`; name pinned by interface-spec) wraps `client.Execute`. This is silent conformance to 010's "backoff above the seam" precedent and the only placement compatible with the per-attempt `client.Timeout`. 010's code and its one-attempt tripwire stay intact; 017's "no unbounded wait" is a property of a loop that sleeps between independent timed attempts.

**Consequences**: 017 is purely additive in `internal/apiclient`; the read path changes only at its send call site (today `me.go:96`). The exact executor/seam signature is interface-level. *Precedent-setting: cross-cutting request behaviors that need to sleep or re-send (retry, backoff) wrap the `Execute` seam from above; the per-`Do` transport stack stays one-shot so `client.Timeout` keeps bounding a single attempt.*

### ADR-2: Honor `Retry-After`, fall back to a bounded backoff, and cap **both** attempts and accumulated wait — then surface the raw 429

**Context**: The API returns `Retry-After` (integer seconds) plus `X-RateLimit-*` on a 429 (`spec/glassfrog-api-v5.yaml:105`, `:5339`). The window can reset up to an hour out, so blind honoring would hang a command. Spec forks 2+3 (approved): honor `Retry-After` exactly, bounded fallback when absent, cap attempts **and** total wait, then surface the raw 429.

**Options considered**:
1. **Honor `Retry-After` with a fallback backoff and dual caps** — wait the header's seconds; if absent/unusable use `FallbackBackoff`; stop when attempts are exhausted **or** the next wait would push accumulated sleep past `MaxTotalWait`, surfacing the last `*ResponseError` unchanged. Predictable, bounded, uses the API's own signal.
2. **Pure exponential backoff, ignore `Retry-After`** — rejected: discards the precise interval the API hands us, risking waiting too little (re-throttle) or too much.
3. **Honor `Retry-After` with no upper bound** — rejected: a far-out window reset would sleep unboundedly; the spec's central Non-Behavior forbids it.

**Decision**: Option 1. `MaxAttempts`, `MaxTotalWait`, `FallbackBackoff` are named `[ASSUMED]` constants (exact values and any future configurability deferred, as 008 deferred the default URL and 010 the timeout). The budget bounds **accumulated sleep** (not wall-clock incl. requests) — simple, and a single `Retry-After` larger than the remaining budget triggers give-up rather than a truncated sleep.

**Consequences**: Every call returns in bounded time; most transient 429s resolve transparently. A surfaced 429 is the *unchanged* `*ResponseError` (status + rate-limit headers + body intact for 015). `parseRetryAfter` parses non-negative integer seconds only (the spec's schema); HTTP-date form is not produced by this API and is treated as "unusable → fallback" (a tunable robustness detail, not a behavior gap). *Feature-local beyond the tunable-constants point 008/010 already set.*

### ADR-3: Only safe (idempotent) methods auto-retry; non-safe requests surface the 429 on first occurrence

**Context**: 017 wraps the seam *every* request would call through. Re-sending a non-idempotent write on a 429 risks double-applying it. The read surface is all `GET` today; the propose/write path is later (PROJECT scope).

**Options considered**:
1. **Retry only safe methods (`GET`/`HEAD`), read off `Request.Method`** — writes surface the 429 unretried, so a non-idempotent op is never silently re-sent. Zero caller ceremony; correct by default.
2. **Retry every method** — rejected: silently re-sends writes; a double-applied mutation is exactly the cost the spec's Non-Behavior names.
3. **A caller opt-in retry flag per request** — rejected as premature: no write caller exists yet, and method-safety is the standard, self-evident signal; a flag can be added later without changing this layer's shape if a safe-but-non-GET case ever needs it.

**Decision**: Option 1. `isSafeMethod(req.Method)` gates the retry; a non-safe 429 returns immediately as the unchanged `*ResponseError`. 

**Consequences**: The whole landed read surface gets retry for free; the future write path is protected by default. *Feature-local.*

### ADR-4: The `sleep` and the progress sink are injected seams; `apiclient` reaches for no process global

**Context**: Backoff must sleep (real time) and emit a non-secret progress note to stderr (spec: operator may be human or agent; no prompt). CONSTITUTION IV wants hermetic tests; 010/009/008/005 inject their OS-touching seams and fail-fast on nil in the pure layer (PR #20/#30); `internal/apiclient` "prints nothing / no `os.Exit`" (010) — it touches no `os.Stdout`/`os.Stderr` directly.

**Options considered**:
1. **Inject `sleep func(time.Duration)` and an `io.Writer` progress sink** — production binds `time.Sleep` and the command's `cmd.ErrOrStderr()`; tests bind a recording fake-sleep (asserts durations, never blocks) and a buffer. Keeps every branch offline and the package free of process globals.
2. **Call `time.Sleep` and `os.Stderr` directly** — rejected: makes the cap tests slow/flaky (real sleeps) and reintroduces the `os`-global reach 010 kept out of the package; the note would bypass the command's stderr.

**Decision**: Option 1. The pure executor takes the seams (fail-fast on nil, no nil-default — PR #20); the production wiring binds the real sleep and the command's stderr. The note names the wait, the next attempt index, and the cap — never the token, the request, or any secret (the `*ResponseError` carries only the *response* side; the auth header is a request header and is not echoed).

**Consequences**: Cap/backoff behavior is tested in milliseconds; the progress note is a buffer assertion. *Precedent-setting: a cross-cutting `apiclient` behavior that must sleep or emit diagnostics injects a clock/sleep + writer seam rather than reaching for `time.Sleep`/`os.Stderr`, keeping the transport package hermetic and command-free.*

### ADR-5: 017 does **not** classify the surfaced 429 or add a rate-limit exit code — that split is API Error Extraction (015)'s

**Context**: 004 reserved code **5 = rate-limit**; 011's `classifyClientError` maps a generic `*ResponseError` → `APIError(3)` with a comment "015/017 refine it later." But 017's spec Non-Behavior is explicit: 017 "must not classify, interpret, or rename the 429 into a meaningful typed error" — that is API Error Extraction (015)'s job. The user-approved composition note: when 017 gives up it "surfaces the raw non-2xx onward, leaving the typed-error classification to 015."

**Options considered**:
1. **017 is retry-only; it touches no `internal/cli` classification** — a given-up 429 stays a generic `*ResponseError` → `APIError(3)` via the unchanged `classifyClientError`. The `429 → RateLimited(5)` split lands with 015, the capability that types non-2xx responses. Faithful to 017's spec and the approved composition note.
2. **017 also adds `RateLimited(5)` to the `Outcome` enum + a `classifyClientError` case** — rejected: it makes 017 classify a status its own spec reserves for 015 (and the same over-reach LEARNINGS #33-r2 flagged for error messages), and couples 017 to `internal/cli` for no behavioral gain (the retry value is the *transparent* recovery; the exit-code typing of a *surfaced* 429 is a separate axis 015 owns).

**Decision**: Option 1. 017 adds **no** `Outcome` category and **no** `classifyClientError`/`ExitCode` edit. Code 5 stays a reserved constant until 015 produces its category. This clarifies the 011 DECISIONS "015/017" shorthand in favor of 015 — retry behavior is 017's; non-2xx typing (incl. `429 → 5`) is 015's.

**Consequences**: 017 stays a clean transport-layer behavior; `internal/apiclient` does not import `internal/cli`. A surfaced (capped-out) 429 exits via whatever `classifyClientError` maps it to — when 017 was authored, before 015 landed, that was `APIError(3)`; now that API Error Extraction (015) has landed it is `RateLimited(5)`, with no change to 017's code. *Precedent-setting: a 429 surfaced after retry exhaustion is typed by 015 (→ `RateLimited(5)`); rate-limit *handling* (retry) and rate-limit *classification* (exit code 5) are separate capabilities.*

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — upstream dependency, landed)**: the `RetryExecutor` calls `(*Client).Execute` once per attempt and inspects the returned error via `errors.As(err, &*ResponseError)` + `StatusCode == 429`. 010 is unchanged: it still makes exactly one `Do` per call and carries the 429's status/headers/body. The rate-limit headers (`Retry-After`, `X-RateLimit-*`) are read off `ResponseError.Header`.
- **Identity Read (011, `internal/cli` — landed consumer)**: `runMe` routes its single `client.Execute` send through the `RetryExecutor`, threading the `sleep` and stderr seams (prod binds `time.Sleep` + `cmd.ErrOrStderr()`). 017 makes **no** `classifyClientError` edit (ADR-5); the shared classifier maps a given-up 429 to whatever it currently owns — `RateLimited(5)` now that 015 has landed. Future reads (013/014 when built; the 012–017 pattern) adopt the same wrapper at their send site.
- **API Error Extraction (015 — downstream sibling, not in this workspace)**: types the non-2xx that finally survives — including a 429 017 gave up retrying — into a meaningful error and the reserved `RateLimited(5)` exit code. Composition order: 010 sends → **017 retries/backs off** → 015 classifies the final outcome. 017 produces the raw 429; 015 types it.
- **Pagination (016 — sibling, not in this workspace)**: walks multi-page reads by issuing more requests; each can be rate-limited, so paging composes with 017's per-request retry. Neither owns the other; both sit on the 010 seam.
- **Exit-Code Convention (004, `internal/cli` — downstream)**: code 5 (rate-limit) stays a reserved constant until 015's producer exists (ADR-5). 017 wires no exit code.
- **Operator stderr (the command's `io.Writer`)**: the only output surface — the per-wait progress note, injected (ADR-4).

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: 017 never reads the token (the secret rides 007's `AuthTransport`, wired in 010, well below this layer). The progress note carries only the wait duration, attempt index, and cap; the `*ResponseError` it inspects holds the *response* side (status, `Retry-After`/`X-RateLimit-*`, body) — the auth header is a *request* header and is not echoed. Pinned by a test asserting no progress note or surfaced error renders the token.

**Error handling (CONSTITUTION III)**: fail loud, never silently. A non-429 outcome passes through unchanged (no masking). A capped-out 429 is surfaced as the unchanged `*ResponseError` — not swallowed, not retried forever. The wait is always bounded (ADR-2). A nil injected `sleep`/writer is a wiring bug → fail-fast (no nil-default — PR #20).

**Observability (spec)**: a single non-secret stderr note per wait, via the injected writer (ADR-4); no interactive prompt; the operator (human or agent) sees a deliberate pause, not a hang.

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network and **no real sleep**. Over a fake base `http.RoundTripper` (010's injected seam) + a recording fake-sleep + a buffer: single-429-then-200 succeeds after one bounded wait; `Retry-After` honored (assert the slept duration equals the header); absent `Retry-After` → `FallbackBackoff`; **attempts-cap tripwire** (always-429 transport → assert *exactly* `MaxAttempts` `Do` calls, then a surfaced 429 — the negative-property tripwire LEARNINGS prescribes); **total-wait-cap tripwire** (`Retry-After`s summing past budget → give up, assert total slept ≤ budget and no further sleep); non-safe method (`POST` 429 → exactly one attempt, returned unchanged); non-429 passthrough (`403` → one attempt, unchanged); the progress-note buffer carries the wait/attempt and **no** token. The driving scenarios become a **new feature file** `features/no-shared-api-client/rate-limit-handling.feature` (the 017 capability under the shared *No Shared API Client* problem dir, beside `request-execution.feature`) wired to a godog suite pointed at **its own** file (LEARNINGS — never the directory); step helpers return errors, never panic (LEARNINGS); reuse the existing `apiclient` step vocabulary where an assertion already exists (grep `sc.Step(` before adding phrasings).

---

## Implementation Strategy

Three phases, linear. Depends only on landed code: 010 (`Client`, `Execute`, `Request`, `ResponseError{StatusCode,Header,Body}`) and 011 (`runMe` send site). Purely additive in `internal/apiclient`; the only edit to landed code is routing the read send through the new executor.

- **Phase 1 — policy + helpers**: define the code-free `RetryPolicy` (`MaxAttempts`/`MaxTotalWait`/`FallbackBackoff` as named `[ASSUMED]` constants), `isSafeMethod`, and `parseRetryAfter` (non-negative integer seconds; absent/negative/non-integer → unusable). RED-first unit tests: safe vs non-safe methods; `parseRetryAfter` parses `"42"`, rejects `""`/`"-1"`/`"abc"`/HTTP-date. *Depends on: nothing new.*
- **Phase 2 — `RetryExecutor`**: the bounded loop over `client.Execute` with injected `sleep` + progress `io.Writer` (fail-fast on nil). Inspect `*ResponseError`+429; honor `Retry-After`/fallback; enforce both caps; surface the raw 429 on exhaustion/over-budget/non-safe; pass every non-429 outcome through unchanged; emit the secret-free note per wait. RED-first unit tests over the fake base + recording sleep + buffer for every branch, including the attempts-cap and total-wait-cap **tripwires** and the token-never-in-output assertion. *Depends on: Phase 1.*
- **Phase 3 — wire the read path + executable acceptance**: route 011's `runMe` send through the `RetryExecutor` (thread `sleep`=`time.Sleep`, progress=`cmd.ErrOrStderr()` in the production seam; tests bind fakes); confirm a 429-then-200 `me` run renders the projection after one bounded wait and a non-429 `me` run is unchanged. godog step definitions for the driving scenarios in the new `internal/apiclient` feature file. *Depends on: Phase 2.*

---

## Risks

- **A `Retry-After` sleep eats `client.Timeout` if retry is mis-placed** (low likelihood given ADR-1, high impact): a transport-level retry would count the backoff against the per-attempt timeout. *Mitigation*: ADR-1 keeps retry strictly *above* `Execute`, so the sleep sits between independent timed attempts; a test with a multi-second fake-sleep confirms no attempt is timed out by the wait.
- **A cap test that sleeps for real → slow/flaky suite** (medium likelihood, medium impact): honoring large `Retry-After`s in a test could block. *Mitigation*: the `sleep` seam is injected and recording — tests assert durations without blocking (ADR-4); CI never waits.
- **Non-idempotent write silently re-sent** (low likelihood today, high impact): no write caller exists yet, but the seam is shared. *Mitigation*: ADR-3 gates retry on `isSafeMethod`; a `POST`-429 test pins exactly one attempt.
- **Surfaced 429 exit-code ownership** (medium likelihood of reviewer/operator surprise, low impact): before 015 landed, a capped-out 429 mapped to `APIError(3)`; 015 has since typed it to the reserved `RateLimited(5)`. *Mitigation*: ADR-5 makes this an explicit, spec-faithful boundary — 017 surfaces the raw 429 and 015 owns the typing; the retry value (transparent recovery) is unaffected either way.
- **Wrong default caps for the real API** (low likelihood, low impact): too-low caps give up on a recoverable throttle; too-high delays a command. *Mitigation*: named `[ASSUMED]` constants, tunable, configurability deferred (as 008/010 deferred their defaults); does not block the slice.

---

## What This Plan Does Not Cover

- **Typing the surfaced 429** into a meaningful error and the reserved `RateLimited(5)` exit code — API Error Extraction (015), which consumes the raw `*ResponseError` 017 surfaces (ADR-5). Not in this workspace.
- **Following pagination** across pages — Pagination (016); each page request composes with 017's retry but the walk is 016's. Not in this workspace.
- **Proactive throttling** on `X-RateLimit-Remaining` — a spec Non-Behavior; a candidate for a later capability, not this one.
- **The exact executor / policy / seam API shape** — `RetryExecutor`'s API (constructor, `Execute` method), the `RetryPolicy` fields, how the `sleep`/progress seams are threaded into the read seam — pinned by interface-spec.md (the specification boundary).
- **The cap *values*, fallback-backoff curve, and any future `--`flag configurability** — named `[ASSUMED]` constants here; tuning/exposure is deferred.
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/no-shared-api-client/rate-limit-handling.feature`.
