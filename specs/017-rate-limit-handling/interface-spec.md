# Interface Accord: Rate-Limit Handling — Specification

**Feature**: 017-rate-limit-handling
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4/5 — the `internal/apiclient` retrying executor that wraps `(*Client).Execute` (010) in a bounded loop, the `RetryPolicy` value, the injected `sleep` + progress-`io.Writer` seams, and the unchanged pass-through of 010's typed outcomes (the surfaced `429` stays a generic `*ResponseError`, classified by 015 — not here).

---

This accord pins the Go API surface of the rate-limit retry layer: the **`RetryExecutor`** that decorates a `*Client` with bounded `429` retry, its **`NewRetryExecutor`** constructor and injected seams, the **`RetryPolicy`** configuration value, and the **`Execute`** method whose signature is **identical to `(*Client).Execute`** so the read path's send call site is unchanged in shape. There is **no command and no entry point** in this slice — retry is a function call layered above the existing send seam, and the stderr progress note travels through an **injected `io.Writer`** (the consuming command's stderr), not a new command surface — so the *invocation* and *instructional* surfaces are **N/A**. The capability introduces **no new `.glassfrogrc` key, env var, or flag**: its only configuration is the `RetryPolicy` caps, fixed `[ASSUMED]` constants (`DefaultRetryPolicy`) whose values and eventual configurability are deferred (Consistency Notes). This accord defines the **retry decorator over the `Execute` seam** — the mirror of 010's accord, which defined the seam itself.

---

## Surface

### Entry points

| Function | Signature (shape) | Description |
|---|---|---|
| `NewRetryExecutor` | `(client *Client, policy RetryPolicy, sleep func(time.Duration), progress io.Writer) *RetryExecutor` | The pure constructor. Wraps a built `*Client` (010) with the retry policy and two **injected** seams: `sleep` (prod binds `time.Sleep`; tests bind a recording fake that never blocks) and `progress` (prod binds the command's `cmd.ErrOrStderr()`; tests bind a buffer). `client`, `sleep`, and `progress` are required and must be non-nil; a nil seam is a wiring bug and panics (fail-fast, no nil-default — DECISIONS/PR #20). The package reaches for no `time.Sleep`/`os.Stderr` directly. |
| `(*RetryExecutor).Execute` | `(reqCtx context.Context, req Request, out any) (*Response, error)` | The send seam **with retry** — signature-identical to `(*Client).Execute`, so a caller swaps the bare client for the executor without changing the call site. Calls `client.Execute` once per attempt (each one timed `Do`, bounded by `client.Timeout`); on a `429` `*ResponseError` for a **safe** method, honors the wait and re-attempts within the policy caps; on any other outcome (success, transport, decode, non-`429` non-2xx, non-safe `429`) returns it **unchanged** on the first occurrence. |

### Configuration contract — `RetryPolicy`

The code-free retry policy. (Field shape is the contract; the constant values are `[ASSUMED]`, deferred.)

| Field | Type | Description |
|---|---|---|
| `MaxAttempts` | `int` | Total attempts incl. the first (≥1). `1` = no retry (the opt-out path the same code serves). |
| `MaxTotalWait` | `time.Duration` | Upper bound on **accumulated sleep** across all waits. A wait that would push the running total past this is not taken — the executor gives up instead. |
| `FallbackBackoff` | `time.Duration` | The wait used for an attempt whose `429` carries no usable `Retry-After`. |

`DefaultRetryPolicy` is the exported default value built from the named `[ASSUMED]` constants; the production wiring uses it.

### Internal helpers (not part of the consumed surface)

| Function | Signature (shape) | Description |
|---|---|---|
| `isSafeMethod` | `(method string) bool` | `GET`/`HEAD` are retryable; anything else is not (idempotency — ADR-3). Read off `req.Method`. |
| `parseRetryAfter` | `(h http.Header) (time.Duration, bool)` | Parses `Retry-After` as a **non-negative integer number of seconds** (the spec's `RetryAfter` schema is `integer`). Absent / non-integer / negative → `(0, false)` — "unusable", caller falls back to `FallbackBackoff`. |

### Outcome contract

`RetryExecutor.Execute` returns the **same** `*Response` / `error` types as `(*Client).Execute` (010) — it adds **no new outcome type**. The surfaced `429` after retry is the **unchanged** `*ResponseError{StatusCode:429, Header, Body}` (status + `Retry-After`/`X-RateLimit-*` headers + body intact). 017 does **not** classify it (ADR-5).

**Example (shapes, not literal values)**:
```
// client = a *Client built once (010); exec wraps it with the policy + seams.
exec := NewRetryExecutor(client, DefaultRetryPolicy, time.Sleep, cmd.ErrOrStderr())

429→200 (safe):  exec.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)   → &Response{StatusCode:200, …}, nil   // me populated after 1 bounded wait
429 capped:      exec.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)   → nil, &ResponseError{StatusCode:429, Header:{Retry-After,…}, Body:[…]}  // unchanged
429 (write):     exec.Execute(reqCtx, Request{Method:"POST", Path:"/…"}, nil)   → nil, &ResponseError{StatusCode:429, …}   // not retried, first occurrence
403 (non-429):   exec.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)   → nil, &ResponseError{StatusCode:403, …}   // passed through, no wait
200 (no 429):    exec.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)   → &Response{StatusCode:200, …}, nil        // no wait, one attempt
```

---

## Interactions

**Build-once / send flow**: the command layer builds the `*Client` once (010 `NewClientFromOS`), wraps it once with `NewRetryExecutor(client, DefaultRetryPolicy, time.Sleep, stderr)`, then calls `(*RetryExecutor).Execute` per request — the read seam routes its single send through the executor instead of the bare client.

**Retry loop** (ADR-1/ADR-2): per attempt, call `client.Execute`. If the error is not a `429` `*ResponseError` → return it. If `!isSafeMethod(req.Method)` → return the `429` unchanged. If this was the last attempt (`MaxAttempts`) → return the `429` unchanged. Else compute the wait — `parseRetryAfter` seconds, else `FallbackBackoff`; if `accumulated + wait > MaxTotalWait` → return the `429` unchanged (give up, **no truncated sleep**); else emit the progress note, `sleep(wait)`, add to the accumulator, and loop.

**Honoring `Retry-After`** (ADR-2): the API hands whole seconds (`spec/glassfrog-api-v5.yaml:5339`); `parseRetryAfter` uses them verbatim. `0` means retry immediately (zero wait); a missing/garbled value falls back. HTTP-date form is not produced by this API and is treated as unusable.

**Eligibility** (ADR-3): only `GET`/`HEAD` auto-retry. A non-safe `429` is surfaced on first occurrence so a non-idempotent operation is never silently re-sent.

**Progress note** (ADR-4): before each `sleep`, one line to the injected `progress` writer naming the wait, the next attempt index, and the cap (e.g. `rate limited; waiting 2s before retry 2/4`). No interactive prompt; the operator (human or agent) sees a deliberate pause. The note carries **no** token, request, or secret.

**Per-attempt timeout boundary** (ADR-1): each `client.Execute` is one `Do` bounded by `client.Timeout`; the `sleep` happens **between** attempts, outside any single attempt's timeout — so a multi-second backoff never trips the request timeout.

---

## Error Communication

`RetryExecutor.Execute` returns exactly one outcome per call, surfacing 010's typed outcomes unchanged:

| Condition | Outcome |
|---|---|
| 2xx (on any attempt) | `*Response{StatusCode, Header}`, nil; `out` populated by that attempt's decode. |
| `429`, safe method, retried, then 2xx | The eventual `*Response`, nil — after one or more bounded waits. |
| `429`, safe method, attempts exhausted **or** next wait over budget | `nil, *ResponseError{StatusCode:429, Header, Body}` — the **last** response, unchanged. |
| `429`, non-safe method | `nil, *ResponseError{429, …}` — unchanged, first occurrence, no wait. |
| Non-`429` non-2xx (any status) | `nil, *ResponseError{…}` — passed through, no wait, no retry. |
| Wire failure / timeout | `nil, *TransportError{cause}` — passed through, no retry. |
| 2xx body does not decode | `nil, *DecodeError{…}` — passed through. |
| No usable token / credential-file error | `nil, *AuthError{…}` — passed through unchanged (from 007 via 010). |

**No classification here** (ADR-5): the surfaced `429` is a generic `*ResponseError`. 017 adds no `Outcome` category and no `ExitCode` edit, and `internal/apiclient` still never imports `internal/cli`; the consuming `classifyClientError` owns the mapping. The `429 → RateLimited(5)` split (code 5 reserved by 004) is API Error Extraction (015)'s job — now landed, so a surfaced 429 maps to `RateLimited(5)` (it mapped to `APIError(3)` before 015 existed), with no change to 017's code.

**No secret anywhere**: 017 never reads the token (it rides 007's `AuthTransport`, wired in 010, below this layer). The progress note carries only timing; the `*ResponseError` it inspects holds the *response* side (the `X-Auth-Token` *request* header is not echoed). Pinned by a token-never-in-output test across the note and the surfaced error.

---

## Consistency Notes

- **Mirrors the established seam patterns**: `NewRetryExecutor(client, policy, sleep, progress)` follows the pure-constructor + injected-seam shape of 010's `NewClient(ctx, base)`, 009's `Assemble`, and 007's `NewAuthTransport` — required seams non-nil, fail-fast (DECISIONS/PR #20). `Execute`'s signature is deliberately identical to `(*Client).Execute` so the read seam's send call site is shape-unchanged (a future `executor` interface could unify both).
- **Wraps the seam from above, by design** (ADR-1): retry is a decorator over `Execute`, not an `http.RoundTripper` in the client stack — the only placement compatible with 010's one-`Do`-per-`Execute` invariant and the per-attempt `client.Timeout`. Realizes 010's "017 layers backoff above the seam."
- **Generic `429` carrier, untyped here** (ADR-5): the surfaced `429` stays 010's generic `*ResponseError`. API Error Extraction (015) types it (incl. the reserved `RateLimited(5)`); Pagination (016) composes above this (each page request is independently retried). Keeping 017 retry-only is what lets 015 own classification — mirrors the LEARNINGS rule (2026-06-07 #33-r2) that a read must not interpret a status its Non-Behaviors reserve for a sibling.
- **No command surface, no new configuration**: like 005/006/007/008/009/010, this slice registers no cobra command. Invocation and instructional surfaces are N/A. The only tunables are the `RetryPolicy` caps — `[ASSUMED]` constants (`DefaultRetryPolicy`), configurability deferred (as 008 deferred the default URL and 010 the timeout). No new `.glassfrogrc` key, env var, or flag.
- **Fifth specification touchpoint in this project**: like 005/007/008/009/010, a specification accord, not a CLI one. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
