# Plan: Pagination

**Feature**: 016-pagination
**Role**: Shaper
**Inputs**: spec.md (016-pagination); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: the API-client seam is a `Client` built once from the `ConnectionContext`, `(*Client).Execute(reqCtx, req, out) (*Response, error)` is the single send seam, 011–017 build on it — 010 ADR-1/§85; 010 produces typed code-free errors (`TransportError`, generic `ResponseError{status,headers,body}`, `DecodeError`) and the consuming command does `errors.As`/maps — `apiclient` never imports `internal/cli` — 010 ADR-3/§91; `classifyClientError(err) Outcome` is the single `errors.As` chain reused by 012–017 with a `default→1` fail-safe, transport→6, non-2xx→3, and 015/017 SPLIT 4/5 without renumbering — 011 §101; **paginated list reads share one `glassfrog.Pagination` (`PerPage`,`HasNextPage`,`NextCursor`) + one `{data,meta:{pagination}}` envelope, first-page-only + a `HasNextPage` signal, the multi-page walk deferred to 016 — 013 §109**; API models live in the leaf `internal/glassfrog` which imports nothing internal so both `cli` and `apiclient` may import it — 011 ADR-1; inject seams + fail-fast on nil, no nil-default — 005/008/PR #20); `.score/memory/LEARNINGS.md` (hermetic offline tests via an injected seam; godog suites point at their own feature file; a change-detector map needs a `len`+comma-ok guard — PR #10). No SOUL.md. DEPRECATION.md carries only the 007/009-seam note.

**Readiness**: Must met + Should substantial — behavioral accord with four When/Then groups, three happy-path + two error + three edge scenarios, nine non-behaviors, integration boundaries naming every collaborator (010/012-014/015/017/API), three user scenarios, five assumptions, no open ambiguities. Strong foundation. The four behavioral forks were resolved during `/score:define` (walker-not-command; partial-flagged-incomplete on a cut-short walk; unbounded walk with a non-advancing-cursor guard; page size defaults to the API max, flag-overridable). **No architectural unknown needed a resolve conversation**: the seam to build on (010's `Client`/`Execute`), the shared `Pagination`/envelope shape, the typed-error discipline, and the classify/exit-code split are all fixed by DECISIONS; Go 1.26 supplies generics. One cross-spec decision — a generic `glassfrog.Page[T]` envelope vs the per-resource concrete `MyRolesResponse` named in 012/013's not-yet-landed plans — is recorded as an **announced divergence** (ADR-2) with a `/score:deprecate` suggestion in the handoff.

---

## System Architecture

Pagination is the **"page through them"** half of CONSTITUTION VI — the reusable walker that turns 010's single-response seam into a complete, multi-page result set. It is a **mechanism, not a command** (like 010/015/017): purely additive, registers no cobra command, prints nothing, calls no `os.Exit`. Its consumers are sibling Go capabilities (the list commands 012–014 when they adopt it, and the future Governance Reads), so its boundary is a **specification boundary**: the package API — the walk function, its options, and the typed result/error — is the structural contract consumers depend on; protocol-level shape is the interface skill's concern.

The defining architectural fact, and a **correction to 010's plan**: the v5 API carries paging in the **response body** at `meta.pagination`, *not* in a `Link` response header (010's plan.md §120 and `execute.go`'s comment say "reads Link/paging headers" — that is wrong for this API; 012 already reads `meta.pagination.has_next_page` from the body). 010's `Execute` decodes the body into the caller's `out` target and exposes only status+headers on `*Response`. So the walker reads paging by **decoding each page into an enveloped target** (`{data, meta:{pagination}}`) — it never inspects a header for cursors.

A new package **`internal/paging`** houses the walk. It composes 010 and the `glassfrog` schema, keeping `apiclient` schema-agnostic (its `Execute(out any)` stays generic) and `glassfrog` logic-free:

```
internal/glassfrog            (leaf schema — imports nothing internal; 011 ADR-1)
  Pagination{PerPage, HasNextPage, NextCursor}        ── shared, reuse-or-create (013 §109)
  Page[T]{ Data []T; Meta Meta{ Pagination } }        ── generic list envelope  [ADR-2]

internal/apiclient            (transport — Execute(out any), typed code-free errors; 010)
  (*Client).Execute(reqCtx, Request, out any) (*Response, error)   ── one bounded send, no retry

internal/paging               (NEW — the walk; imports glassfrog + apiclient)        [ADR-1]
  type Executor interface { Execute(ctx, apiclient.Request, out any) (*apiclient.Response, error) }
        │  satisfied by *apiclient.Client (prod); a fake Executor in tests (hermetic, offline)
  All[T any](reqCtx, ex Executor, req apiclient.Request, opts ...Option) Result[T]   ── the walker  [ADR-1]
    │  size := 500 (API max, conserve rate budget); WithPageSize(n) overrides            [ADR-4]
    │  loop:
    │    q := clone(req.Query); q.Set("per_page", size); if cursor != "" { q.Set("cursor", cursor) }
    │    var page glassfrog.Page[T]
    │    resp, err := ex.Execute(reqCtx, req{Query:q}, &page)
    │    if err != nil        → return Result{Records: acc, Complete:false, Stop: err}   ── partial  [ADR-3]
    │    acc = append(acc, page.Data...)                                                  ── API order, no dedup
    │    pg := page.Meta.Pagination
    │    if !pg.HasNextPage   → return Result{Records: acc, Complete:true}                ── done (incl. absent meta)
    │    if pg.NextCursor==""  → return Result{Records: acc, Complete:false, Stop:&MalformedPageError{}} ── no loop [ADR-5]
    │    cursor = pg.NextCursor
  type Result[T any]{ Records []T; Complete bool; Stop error; Pages int }                ── explicit  [ADR-3]
  type MalformedPageError struct{ Page int }                                             ── classify→default(1)

consumed by ──► 012-014 list reads (when they adopt the walk) + future Governance Reads:
                render Records; if !Complete → stderr "incomplete: <Stop>" + classifyClientError(Stop)→ExitCode
```

The walker is the orchestrator and nothing more: it issues page requests through the seam, concatenates `Data` in API order, and stops — complete, or partial with a named cause. It never touches the token (rides 010's `Client`, which rides 007's transport), decides no exit code, and renders nothing.

---

## Architecture Decisions

### ADR-1: Pagination is a library-only generic walker in a new `internal/paging` package over the 010 `Client` seam; no command, no exit-code edits

**Context**: 016 is "a mechanism, not a command" (spec System Overview) sitting on 010's `(*Client).Execute` (DECISIONS §85). 010 produces typed code-free errors and `apiclient` must not import `internal/cli` (§91). The walk loops over `Execute`, accumulates typed records, and needs the shared `glassfrog.Pagination` (§109). The home must compose `apiclient` (seam, errors, request) **and** `glassfrog` (envelope, pagination) without inverting either's role.

**Options considered**:
1. **A new `internal/paging` package** holding the walk, an `Executor` seam interface (satisfied by `*apiclient.Client`), `Result[T]`, and the typed error. It imports `apiclient` + `glassfrog`; `apiclient` stays `out any`-generic transport and `glassfrog` stays logic-free schema. Tests inject a fake `Executor` — hermetic, no HTTP.
2. **A file in `internal/apiclient`** (keep the "seam family" together). Rejected: forces `apiclient` (transport) to import `glassfrog` (schema), inverting 010's schema-agnostic `Execute(out any)` boundary and mixing the typed-record walk into the byte-level seam.
3. **A method on `*apiclient.Client`** (`client.Paginate[T]`). Rejected: Go does not allow type-parameterized methods, and it would still drag schema into `apiclient`.

**Decision**: Option 1. `internal/paging` is the walk's home. `All[T any](reqCtx, ex Executor, req apiclient.Request, opts ...Option) Result[T]` is the seam; `Executor` is the one-method interface `*apiclient.Client` already satisfies. Production passes the real `Client`; tests pass a fake `Executor` that returns canned pages by decoding into `out` — the project's inject-the-seam pattern (005/006/008/011), here at the page granularity.

**Consequences**: `apiclient` and `glassfrog` are untouched in role; the cross-cutting walk lives in one discoverable place that 012–014 and Governance Reads import. 016 is purely additive — **no cobra command, no `Outcome`/`ExitCode` edit** (like 010): the walker's `Stop` error is classified by the *consumer* via the existing `classifyClientError` (§101). Tests run fully offline against a fake `Executor`, with no `net/http` in the walk's unit tests. *Precedent-setting: the multi-page walk lives in `internal/paging` as a generic function over an `Executor` seam; list commands depend on it, not a hand-rolled loop.*

### ADR-2: The walker reads paging from the decoded body (`meta.pagination`), decoding a generic `glassfrog.Page[T]` that reuses the shared `glassfrog.Pagination` — superseding the per-resource concrete envelope

**Context**: The v5 spec carries paging in the response **body** (`{data, meta:{pagination:{per_page, has_next_page, next_cursor}}}`), and 010's `Execute` surfaces the body only by decoding it into the caller's `out` (status+headers on `*Response`, nothing else). 010's plan and `execute.go` comment claim 016 reads a `Link` *header* — incorrect for this API. DECISIONS §109 fixed "one shared `glassfrog.Pagination` + one `{data,meta}` envelope," but 012/013 plans named a **per-resource concrete** envelope (`MyRolesResponse{Data []Role; Meta…}`); none of 012–014 has landed yet (all at *Analyzed*). Go 1.26 has generics.

**Options considered**:
1. **A generic `glassfrog.Page[T]{Data []T; Meta Meta}`** reusing the shared `glassfrog.Pagination`; the walker decodes each page into `Page[T]` and reads `page.Meta.Pagination`. One envelope shape for every resource — "each read adds only its resource model" (the type arg `T`), which is §109's *intent* expressed generically, with zero per-resource boilerplate.
2. **Per-resource concrete envelopes** (`MyRolesResponse`, `MyActionsResponse`, …) plus an interface the walker ranges over (`Items() []T; Page() Pagination`). Rejected: needs accessor methods + an allocator factory on every resource — exactly the boilerplate generics remove — and three near-identical envelope types.
3. **Decode `data` as `json.RawMessage` and aggregate raw bytes**, leaving the caller to decode. Rejected: pushes per-record decoding onto every consumer, loses type safety, and complicates "concatenate in API order."

**Decision**: Option 1. Introduce `glassfrog.Page[T any]` over the shared `glassfrog.Pagination`. The walker decodes `&glassfrog.Page[T]{}` per page (header reading is **not** how cursors are found — correcting 010's note). This **supersedes** the per-resource concrete-envelope wording in 012/013's plans: the canonical shared list envelope is the generic `Page[T]`. Because no paginated read has landed, this is a forward choice, not a refactor; whichever of 016/012-014 lands first creates `glassfrog.Pagination` + `Page[T]` (the §109 first-to-land rule), the others reuse.

**Consequences**: One envelope, no boilerplate; new list commands `paging.All[glassfrog.Role](…)` for free. This is an **announced divergence** from 012/013's concrete-envelope decision — flagged in the handoff with a `/score:deprecate` suggestion so the precedent is formally retired; when 012–014 implement, they decode `Page[Role]`/`Page[Action]`/`Page[Project]` (single-page or walked) rather than a bespoke `MyRolesResponse`. The `meta`-is-body correction must also be carried into 010's stale `execute.go` comment (Risks). *Precedent-setting: the shared paginated list envelope is the generic `glassfrog.Page[T]`; paging state is read from the decoded body, never a response header.*

### ADR-3: The walk outcome is an explicit `Result[T]{Records, Complete, Stop, Pages}` — partial-flagged-incomplete, not `([]T, error)`

**Context**: Spec decision 2 (the defining fork): a cut-short walk must return **the records gathered so far, flagged incomplete, carrying the stopping cause** — never all-or-nothing, and never a partial set passed off as complete (the silent truncation CONSTITUTION VI forbids). Go's reflexive idiom `records, err := …; if err != nil { return err }` *discards* the partial records — the precise footgun the spec warns against.

**Options considered**:
1. **An explicit `Result[T]{Records []T; Complete bool; Stop error; Pages int}`** returned by value (never a bare error). `Complete` is the contract; `Stop` is non-nil iff `!Complete` and names the cause; `Records` is always the set gathered (possibly partial). The consumer cannot accidentally drop the partial set — it is a named field, not a value shadowed by an `if err != nil` early return.
2. **`([]T, error)` with "records may be non-empty on error" documented** (the `io.Reader` idiom). Rejected: relies on every consumer remembering the non-standard contract; the standard reflex throws the records away on error — defeating decision 2.
3. **A typed error that *embeds* the partial records** (`PartialError[T]{Records, Cause}`). Rejected: records-inside-an-error is awkward to consume (`errors.As` to read data), and still invites the discard reflex on the happy `err != nil` path.

**Decision**: Option 1. `All[T]` returns `Result[T]` by value. `Complete == (Stop == nil)`; `Pages` is the page count (≥1) for observability and tests. The consumer renders `Records` always, and on `!Complete` writes the incompleteness signal (stderr) and maps `Stop` for the exit code. The struct makes "never mistake partial for complete" structurally hard to get wrong.

**Consequences**: Consumers get a single value carrying everything VI needs (records + completeness + cause). The walker never returns a naked `error`, so the partial set is never silently dropped. `Stop` holds whatever stopped the walk — a 010 typed error (`*TransportError`/`*ResponseError`/`*AuthError`) or `*MalformedPageError` — so the consumer's existing `classifyClientError` maps it unchanged. *Precedent-setting: paginated reads surface an explicit complete/partial result value, not a bare slice+error.*

### ADR-4: Page size defaults to the API max (500) via a `WithPageSize` option; the walker clones the caller's query per page and sets only `per_page`+`cursor`, with no silent clamp

**Context**: Spec decision 4 — request a large page size to minimize round-trips against the rate limit, overridable, *for now only via a command-line flag*. The API: `per_page` 1–500 (default 100), over-max → `400`, blank/zero/negative/non-numeric → API falls back to 100; "callers should not assume the server silently clamped." `apiclient.Request.Query` is a shared `url.Values` map the walker must not mutate. The flag itself is a CLI surface (interface's concern), not the walker's.

**Options considered**:
1. **Default `per_page=500`; a functional `WithPageSize(n)` option overrides; per page, clone `req.Query`, set `per_page` and `cursor`, leave all other params (`q`, `include`) intact; pass an out-of-range size through and surface the API's `400` as the `Stop` cause** (no client-side clamp). Conserves the rate budget, preserves the caller's filter, and honors "no silent clamp."
2. **Hardcode `per_page=500`, no override**. Rejected: spec decision 4 requires operator override (the flag).
3. **Default to the API default (100)**. Rejected: 5× the round-trips of 500 against a rolling rate limit (50/hr on Free) — the opposite of decision 4's intent.
4. **Clamp/validate the size inside the walker** (reject >500 before sending). Rejected here: the spec's "no silent clamp" + "surface the API's rejection" makes the API the bound's owner; optional 1–500 flag validation belongs to the *flag* (interface), not the walk. The walker stays a faithful pass-through.

**Decision**: Option 1. `All[T]` defaults the page size to `500` and accepts `WithPageSize(n)`. Each iteration **clones** the caller's `req.Query` (never mutates the shared map), sets `per_page` to the configured size and `cursor` to the prior page's `next_cursor` (omitted on the first request), and preserves every other parameter. An out-of-range size is sent as-is; the resulting `400` stops the walk as a `Stop` cause (ADR-3), never a silent clamp.

**Consequences**: Fewer requests per walk; the caller's filter/include survive every page; the no-silent-clamp guarantee holds. The `--per-page` flag (name/validation) and its wiring are deferred to the consumer/interface — the walker only needs the resolved integer. *Feature-local; the default-500 + clone-query rule is noted for consumers but not broad precedent.*

### ADR-5: Unbounded walk with a non-advancing-cursor guard; no hard page cap; the malformed-paging stop classifies to the default exit code

**Context**: Spec decision 3 — walk to completion (no mandatory cap), but never loop on a malformed paging response: `has_next_page=true` with an absent/blank `next_cursor` must fail loud, not spin. The stop must flow through the consumer's existing `classifyClientError` (§101), which has a `default→1` (RuntimeError) fail-safe and reserved 3–6 for genuine API/network outcomes.

**Options considered**:
1. **Unbounded loop (until `has_next_page=false`) + a guard**: when `has_next_page=true` but `next_cursor==""`, stop with a typed `MalformedPageError` (`Complete:false`), never re-issuing with an empty cursor. No fixed page/item cap.
2. **Impose a hard max-pages cap** that stops loudly when hit. Rejected: the user declined a mandatory ceiling (decision 3); a silent cap would itself be truncation, and a loud one adds a knob nothing yet needs. (A future `WithMaxPages` option remains open without changing the contract.)
3. **No guard, trust the API to terminate**. Rejected: a buggy/adversarial `has_next_page=true`+empty-cursor response would loop forever — the exact fail-loud-not-spin hazard decision 3 names.

**Decision**: Option 1. The loop runs until a page reports `has_next_page=false` (which also covers an **absent** `meta.pagination` — the zero-valued `HasNextPage` is `false`, so a non-paginated endpoint like the org role tree completes in one page). The non-advancing-cursor guard yields `Result{Complete:false, Stop:&MalformedPageError{Page:n}}`. `MalformedPageError` carries no status, so it falls to `classifyClientError`'s `default→RuntimeError(1)` — **no new `Outcome`/`ExitCode` case, no renumbering** (§101).

**Consequences**: Large reads complete; non-paginated endpoints work unchanged; a malformed cursor fails loud as a partial-incomplete result instead of hanging. The default classification means 016 needs zero `internal/cli` edits — the consumer maps `MalformedPageError` via the fail-safe it already has. A bounded variant stays a clean future addition. *Feature-local.*

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — upstream dependency, landed/Complete)**: the walker calls `Execute(reqCtx, req, &glassfrog.Page[T]{})` once per page through the `Executor` seam. A 2xx decodes into the envelope (walker reads `Meta.Pagination`, appends `Data`); any 010 typed error (`*TransportError`/`*ResponseError`/`*AuthError`/`*DecodeError`) stops the walk and becomes `Stop`. The walker re-resolves nothing and never reads the token. **Correction to carry upstream**: 010's `execute.go` comment and plan say 016 "reads Link/paging headers" — 016 reads the decoded body's `meta.pagination`; the comment should be amended (Risks).
- **`internal/glassfrog` (schema — leaf, importable by `paging`)**: owns the shared `Pagination{PerPage,HasNextPage,NextCursor}` and the new generic `Page[T]{Data []T; Meta Meta}` (ADR-2). Reuse-or-create per the §109 first-to-land rule.
- **List commands 012 My Roles / 013 My Actions / 014 My Projects + future Governance Reads (`internal/cli` — downstream consumers)**: call `paging.All[T]`, render `Result.Records`, and on `!Complete` emit the incompleteness signal (stderr, mirroring 012's `has_next_page` note) and map `Result.Stop` via `classifyClientError`→`ExitCode`. Today these single-page and signal incompleteness themselves; adopting the walk (and switching to `Page[T]`) is each command's own later change, not 016's.
- **API Error Extraction (015 — sibling, unbuilt)**: when a page's `Stop` is a non-2xx `*ResponseError`, 015 interprets it; the walker carries it raw.
- **Rate-Limit Handling (017 — sibling, unbuilt)**: 017's backoff wraps the per-page `Execute` (it "layers backoff above the seam" — §91); the walker treats each page atomically and never sleeps/retries. A page that ultimately fails with `429` stops the walk with the partial set — the rate-limited large read this capability exists to survive.
- **Exit-Code Convention (004, `internal/cli` — downstream)**: the walker wires no exit code; `Result.Stop` maps through the consumer's `classifyClientError` (transport→6, non-2xx→3, `MalformedPageError`→default 1). No `apiclient`/`paging`→`cli` dependency.

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: the walker never holds or reads the token — it rides `*apiclient.Client`, which rides 007's `AuthTransport`. It logs nothing and constructs no diagnostic from request headers. `Result.Stop` carries only 010's typed errors (response-side status/headers/body and net causes — never the request auth header) or `MalformedPageError` (a page index). Pinned by a test asserting no `paging` output renders a token and the walk reads no credential.

**Faithful aggregation & no silent truncation (CONSTITUTION I + VI)**: records are concatenated in the API's order across pages — **no reorder, no dedup, no transform** (ADR-2/§ Non-Behaviors). Every outcome states `Complete`; an incomplete walk always carries a `Stop` cause (ADR-3). The caller's `q`/`include` params survive every page (ADR-4). These are the spec's load-bearing invariants and each gets a dedicated test.

**Error handling (CONSTITUTION III)**: fail loud, never spin. A mid-walk failure stops and surfaces the cause with the partial set; a non-advancing cursor stops with `MalformedPageError` rather than looping (ADR-5); a `reqCtx` cancellation propagates from `Execute` as the `Stop` cause. No path returns a partial set that reads as complete.

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network. A **fake `Executor`** returns canned pages by decoding scripted JSON into `out`, keyed by the request's `cursor` — exercising: single-page-complete; multi-page (assert cursor threaded, `Data` concatenated in order, `Pages` count); non-paginated (absent `meta` → complete in one page); empty (`data:[]` → complete, zero records); mid-walk failure (page 3 returns `*ResponseError`/`429` → partial + `Stop`, records from pages 1–2 retained); first-page failure (→ empty + `Stop`); non-advancing cursor (`has_next_page:true`+blank `next_cursor` → `MalformedPageError`, no loop — a call-counting tripwire pins "did not spin"); query preservation (caller's `q`/`include` present on every page, only `per_page`/`cursor` added); default size 500 and `WithPageSize` override observed on the issued request; token-never-rendered. Pure helpers (`clone` query, cursor selection) unit-tested directly. The driving scenarios become a **new `internal/paging` godog suite** pointed at its **own** feature file `features/silent-truncation/pagination.feature` (LEARNINGS — never the whole `features/` dir); step helpers return errors, never panic.

---

## Implementation Strategy

Two phases, linear; likely a **single PR-sized unit** (016 is small, self-contained, and library-only). Depends only on landed code (010 `Client`/`Execute`/typed errors, Complete on `main`) plus the shared `glassfrog.Pagination` (reuse-or-create). Purely additive — no existing file modified except the one-line `execute.go` comment correction (Phase 2).

- **Phase 1 — schema + walker**: in `internal/glassfrog`, add (reuse if a paginated read already landed) `Pagination{PerPage,HasNextPage,NextCursor}` and the generic `Page[T]{Data []T; Meta Meta{Pagination}}`. In the new `internal/paging`, define the `Executor` interface, `Result[T]`, `MalformedPageError`, the `Option`/`WithPageSize` functional option, and `All[T]` — the clone-query + cursor loop + completion/guard/failure branches (ADRs 3/4/5). RED-first unit tests via the fake `Executor` for every branch above, plus the non-advancing-cursor tripwire. *Depends on: 010 (landed).*
- **Phase 2 — executable acceptance + comment correction**: godog step definitions for the driving scenarios in the new `internal/paging` suite pointed at `features/silent-truncation/pagination.feature`; amend 010's stale `execute.go` "reads Link/paging headers" comment to "paging is read from the decoded body's `meta.pagination`" (ADR-2). *Depends on: Phase 1.*

Consumer adoption (012–014 switching from single-page-signal to `paging.All`, and the `--per-page` flag) is **out of this plan** — each is the consuming command's own change.

---

## Risks

- **`cursor` vs `after` spec inconsistency** (medium likelihood, low impact): the v5 spec's prose intro says page with `after`, but the defined `Cursor` parameter is `name: cursor` and the `Pagination` schema says "Pass as `?cursor=`". *Mitigation*: the walker uses `cursor` (two of three sources agree); a focused integration check (or a doc-confirmation) against the live API before consumers ship; the param name is a one-line change if wrong.
- **016/017 composition — where backoff wraps** (medium likelihood, medium impact): if 017 wraps the *whole walk* instead of the *per-page* `Execute`, a mid-walk 429 would be retried inconsistently. *Mitigation*: this plan fixes the seam at per-page `Execute` (§91 "above the seam"); the walker is retry-agnostic and treats each page atomically — 017 layers on the `Executor` it injects. Recorded so 017's plan honors it.
- **Generic-envelope divergence from 012/013** (medium likelihood, low impact): 012/013 plans name a concrete `MyRolesResponse`; this plan makes the shared envelope generic `Page[T]`. *Mitigation*: none has landed (all *Analyzed*), so there is no refactor; announced as ADR-2 with a `/score:deprecate` suggestion; first-to-land creates the shared types, others reuse.
- **A consumer drops the partial set on `Stop`** (low likelihood, high impact): the whole point of decision 2 defeated if a command does `if res.Stop != nil { return }` without rendering `res.Records`. *Mitigation*: `Result[T]` makes `Records` a first-class field independent of `Stop` (ADR-3); a consumer-side scenario (when 012–014 adopt) asserts partial records are printed; documented on the type.
- **Stale "Link header" assumption spreading** (low likelihood, low impact): 010's plan and `execute.go` comment misstate the paging mechanism. *Mitigation*: ADR-2 corrects it and Phase 2 amends the comment; this plan is the canonical statement (body `meta.pagination`).

---

## What This Plan Does Not Cover

- **The exact package API shape** — the `All`/`Executor`/`Option`/`Result`/`MalformedPageError` signatures and the `glassfrog.Page[T]`/`Pagination` field tags — `/score:interface` pins these (the specification boundary).
- **The `--per-page` flag** (name, 1–500 validation, help text) and its wiring into a command — a consuming command's change; the walker only consumes the resolved integer.
- **Consumer adoption** — 012–014 switching from first-page-signal to `paging.All` (and from `MyRolesResponse` to `Page[T]`) — each list command's own later spec/change.
- **Interpreting a non-2xx `Stop`** into typed API/permission errors (015) and **`429` backoff** (017) — sibling capabilities; the walker carries the raw stop cause and makes one attempt per page.
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/silent-truncation/pagination.feature`.
- **A bounded-walk option** (`WithMaxPages`) — intentionally deferred (ADR-5); the contract leaves room without change.
