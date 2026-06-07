# Interface Accord: Pagination — Specification

**Feature**: 016-pagination
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4/5 — the new `internal/paging` package's generic `All[T]` walker over an `Executor` seam, its `Result[T]` / `MalformedPageError` / `WithPageSize` surface, and the generic `glassfrog.Page[T]` envelope over the shared `glassfrog.Pagination`, consumed by the list commands (012–014 on adoption) and future Governance Reads.

---

This accord pins the Go API surface of the multi-page pagination walker: the **`internal/paging`** package's **`All[T]`** function, the **`Executor`** seam it walks through (satisfied by `*apiclient.Client`), the **`Result[T]`** value it returns (partial-flagged-incomplete), the **`Option`/`WithPageSize`** configuration, the typed **`MalformedPageError`**, and the **`glassfrog.Page[T]`** envelope (over the shared **`glassfrog.Pagination`**) it decodes. There is **no command and no entry point** in this slice — pagination is a library function consumers call — so the *invocation* surface is **N/A**; the **`--per-page` flag** that will set the page size is a *consumer command's* surface (a later change), not this capability's. The capability introduces **no configuration of its own** beyond the in-process `WithPageSize` option (default the API max, 500); no `.glassfrogrc` key, env var, or flag. This accord defines the **walk built on the 010 seam** — the mirror of 010's accord, which defined the single-request seam this one loops over.

---

## Surface

### Entry points

| Function | Signature (shape) | Description |
|---|---|---|
| `All` | `All[T any](reqCtx context.Context, ex Executor, req apiclient.Request, opts ...Option) Result[T]` | The walker. Issues page requests through `ex` (one `Execute` per page), decoding each into a `glassfrog.Page[T]`, concatenating `page.Data` in API order until a page reports `has_next_page=false` (or carries no `meta.pagination`). Returns a `Result[T]` by value — **never a bare `error`** (ADR-3). Defaults the page size to **500** (the API max), overridable via `opts`. |
| `WithPageSize` | `WithPageSize(n int) Option` | Functional option overriding the per-request `per_page`. The walker passes `n` through **as given** — no client-side clamp; an out-of-range value (e.g. > 500) surfaces the API's `400` as the walk's `Stop` cause (ADR-4). Flag-level 1–500 validation, if any, belongs to the consumer's `--per-page` flag, not here. |

### Seam contract — `Executor`

The one-method interface `All` walks through. `*apiclient.Client` satisfies it as-is (its `Execute` has this exact shape); production passes the real `*Client`, tests pass a fake that decodes canned pages into `out`.

| Method | Signature (shape) | Description |
|---|---|---|
| `Execute` | `Execute(reqCtx context.Context, req apiclient.Request, out any) (*apiclient.Response, error)` | 010's single send seam — one bounded request, decode-into-`out` on 2xx, typed code-free error otherwise. `All` supplies `out = &glassfrog.Page[T]{}` per page. |

### Output contract — `Result[T]`

Returned by value from `All`. `Complete == (Stop == nil)`; `Records` is always the set gathered (possibly partial); a consumer renders `Records` regardless and, when `!Complete`, signals incompleteness and maps `Stop`.

| Field | Type | Description |
|---|---|---|
| `Records` | `[]T` | All records gathered across pages, concatenated in API order — never reordered, de-duplicated, or transformed. May be partial when `!Complete`; empty on a first-page failure. |
| `Complete` | `bool` | `true` iff the walk reached the API's end (`has_next_page=false`, or no `meta.pagination`). `false` on any early stop. |
| `Stop` | `error` | `nil` iff `Complete`. Otherwise the cause that stopped the walk — a 010 typed error (`*apiclient.TransportError` / `*apiclient.ResponseError` / `*apiclient.AuthError` / `*apiclient.DecodeError`) or `*MalformedPageError`. `errors.As`-discriminable; carries no token. |
| `Pages` | `int` | Number of page requests issued (≥ 1) — observability + tests. |

### Envelope contract — `glassfrog.Page[T]` (+ shared `glassfrog.Pagination`)

The generic list envelope `All` decodes each page into. Reuse-or-create per the §109 first-to-land rule; `Pagination` is the shared struct 012–014 also use.

| Type | Shape | Description |
|---|---|---|
| `glassfrog.Page[T any]` | `{ Data []T \`json:"data"\`; Meta Meta \`json:"meta"\` }` where `Meta` is `{ Pagination Pagination \`json:"pagination"\` }` | The `{data, meta:{pagination}}` body shape. `T` is the caller's resource model (`glassfrog.Role`, `Action`, `Project`, …) — the only per-read addition. |
| `glassfrog.Pagination` | `{ PerPage int \`json:"per_page"\`; HasNextPage bool \`json:"has_next_page"\`; NextCursor string \`json:"next_cursor"\` }` | Paging state read from the **response body** (not a header). An **absent** `meta.pagination` decodes to the zero value (`HasNextPage=false`) → the page is treated as complete (non-paginated endpoints like the org role tree). |

### Error type — `MalformedPageError`

| Type | Shape | Returned (as `Result.Stop`) when |
|---|---|---|
| `*MalformedPageError` | `{ Page int }` | A page reports `has_next_page=true` but the cursor does not advance — `next_cursor` is blank/absent **or identical to the cursor just used** — so the walker stops rather than re-issuing or looping (ADR-5). `Page` is the 1-based page index at which the non-advancing cursor was seen. Carries no status, so a consumer's `classifyClientError` maps it via `default → RuntimeError(1)`. |

**Example (shapes, not literal values)**:
```
// ex = *apiclient.Client (or a fake Executor); reqCtx = the per-request context.Context.
complete:     paging.All[glassfrog.Role](reqCtx, ex, apiclient.Request{Method:"GET", Path:"/me/roles"})
                 → Result{Records:[…all roles…], Complete:true,  Stop:nil,                       Pages:3}
sized:        paging.All[glassfrog.Action](reqCtx, ex, req, paging.WithPageSize(250))
                 → Result{Records:[…],          Complete:true,  Stop:nil,                       Pages:2}
mid-walk 429: paging.All[glassfrog.Role](reqCtx, ex, req)
                 → Result{Records:[…pages 1–2…],Complete:false, Stop:&apiclient.ResponseError{StatusCode:429,…}, Pages:3}
first-page:   paging.All[glassfrog.Role](reqCtx, ex, req)
                 → Result{Records:[],           Complete:false, Stop:&apiclient.TransportError{…}, Pages:1}
malformed:    paging.All[glassfrog.Role](reqCtx, ex, req)
                 → Result{Records:[…so far…],   Complete:false, Stop:&paging.MalformedPageError{Page:2}, Pages:2}
non-paginated:paging.All[glassfrog.Role](reqCtx, ex, req)   // response has no meta.pagination
                 → Result{Records:[…all…],      Complete:true,  Stop:nil,                       Pages:1}
```

---

## Interactions

**Walk flow**: `All` loops — build the per-page request, `ex.Execute` it into `&glassfrog.Page[T]{}`, append `Data`, inspect `Meta.Pagination`, and either continue (with the next cursor), complete, or stop. The consumer hands in the request once (method, path, its own query/filters); `All` owns only the paging params across iterations.

**Query handling (never mutate the caller's map)**: each iteration **clones** `req.Query` (an `url.Values`/map is shared by reference), then sets `per_page` to the configured size and `cursor` to the prior page's `next_cursor` (omitted on the first request). Every other caller parameter (`q`, `include`, …) is preserved unchanged on every page (spec Non-Behavior). The caller's own `req.Query` is never modified.

**Page size**: defaults to **500** (the API max — fewer round-trips against the rolling rate limit, spec decision 4). `WithPageSize(n)` overrides; `n` is sent as-is (no clamp).

**Completion vs. continuation**: after appending a page's `Data` — if `Meta.Pagination.HasNextPage` is `false` (including the zero-value case where the body had no `meta.pagination`), the walk completes (`Complete:true`). If `true` and `NextCursor` **advances** (non-blank and different from the cursor just used), the loop continues with that cursor. If `true` but the cursor does **not** advance — `NextCursor` is blank/absent **or equal to the cursor just used** — the walk stops with a `*MalformedPageError` — it does **not** re-issue and does **not** loop (ADR-5). The walker tracks the prior cursor to make this comparison; the repeated-cursor case guards against an API that ignores an unrecognized `cursor` param (the `cursor`-vs-`after` ambiguity) returning the same page forever. There is no fixed page/item cap (unbounded; a `WithMaxPages` option is a deferred future addition).

**Stop-and-retain on failure**: if any page's `Execute` returns an error (transport, non-2xx incl. `429`, decode, or a propagated `AuthError`), the walk stops and returns the records gathered so far with `Complete:false` and `Stop` set to that error — the partial set is **retained**, never discarded (spec decision 2). A cancelled `reqCtx` surfaces from `Execute` as the `Stop` cause the same way.

**Retry/rate-limit composition**: `All` makes one `Execute` call per page and never sleeps or retries. Rate-Limit Handling (017), when present, wraps the **per-page** `Execute` (it injects the `Executor`), so backoff is transparent to the walker; a page that ultimately fails with `429` stops the walk with the partial set.

**Secret hygiene**: `All` never reads or holds the token — identity rides `*apiclient.Client` → 007's `AuthTransport`. It logs nothing and builds no diagnostic from request headers; `Result.Stop` carries only response-side / network causes or a page index.

---

## Error Communication

`All` returns exactly one `Result[T]` per call; failure is communicated through `Complete`/`Stop`, **not** a separate return value — so a consumer cannot drop the partial set with a reflexive `if err != nil { return }`.

| Condition | Outcome |
|---|---|
| Walk reaches `has_next_page=false` (or no `meta.pagination`) | `Result{Records, Complete:true, Stop:nil, Pages}`. |
| Page `Execute` returns a typed error (transport / non-2xx / decode / auth) | `Result{Records:<gathered so far>, Complete:false, Stop:<that 010 error>, Pages}`. Partial set retained. |
| `has_next_page=true` + non-advancing cursor (blank/absent **or** equal to the prior cursor) | `Result{Records:<gathered so far>, Complete:false, Stop:&MalformedPageError{Page}, Pages}`. No loop. |
| First page fails | `Result{Records:[], Complete:false, Stop:<error>, Pages:1}`. |
| `reqCtx` cancelled mid-walk | Surfaces as the `Execute` error in `Stop`; partial set retained. |

**Code-free, consumer-maps**: `All` wires **no exit code** and prints nothing. The consuming command renders `Result.Records`, and when `!Complete` writes a single incompleteness note to **stderr** (mirroring 012's first-page-`has_next_page` signal — never silent, CONSTITUTION VI) and maps `Result.Stop` through the **existing** `classifyClientError` (011 §101): `*TransportError` → code 6 (network-unavailable); `*ResponseError` → 3/4/5 (once 015/017 classify); `*MalformedPageError` (and any unrecognized cause) → the `default → RuntimeError(1)` fail-safe. **No new `Outcome`/`ExitCode` case and no renumbering** are required by this capability — `internal/paging` does not import `internal/cli`.

---

## Consistency Notes

- **Mirrors the established seam patterns**: `All[T]` over an injected `Executor` follows the inject-the-seam shape of 010's `NewClient(ctx, base)` and 009's `Assemble`/`AssembleFromOS` (prod binds the real `*Client`, tests a fake), here at the page granularity; the typed `MalformedPageError` follows the code-free, `errors.As`-able, token-free `AuthError`/`TransportError` precedent.
- **Reuses the shared envelope, generically (ADR-2)**: `glassfrog.Pagination` is the shared struct §109 fixed; this accord makes the envelope generic `glassfrog.Page[T]` — "each read adds only its resource model" as the type arg — **superseding** the per-resource `MyRolesResponse` named in 012/013's not-yet-landed plans. Whichever of 016/012-014 lands first creates `Pagination` + `Page[T]`; the others reuse. (`/score:deprecate` suggested for the per-resource-envelope note.)
- **Corrects 010's header assumption**: 010's accord said Pagination "reads `Response.Header` Link/paging headers." The v5 API carries paging in the **response body** at `meta.pagination`, so this walker reads the decoded envelope, not a header — 010's `Response` exposes only status+headers, and the stale `execute.go` comment is amended in implementation.
- **No command surface, no new configuration**: like 005/006/007/008/009/010, this slice registers no cobra command and prints nothing; the invocation and instructional surfaces are N/A. The only tunable is the in-process `WithPageSize` (default 500); the `--per-page` *flag* and consumer adoption (012–014 switching from first-page-signal to `paging.All`, and from `MyRolesResponse` to `Page[T]`) are each the consuming command's own change.
- **Specification touchpoint, like its siblings**: a specification accord (the Go package API), not a CLI one. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
