# Tasks: Pagination

**Feature**: 016-pagination
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/silent-truncation/pagination.feature

---

## Dependency Graph

Phase 1: Shared list envelope + the `internal/paging` walker (2 tasks, no phase dependencies) [Shared]
Phase 2: Executable acceptance + 010 comment correction (2 tasks, depends on Phase 1) [Shared]

4 tasks total | phases run sequentially (Phase 2 depends on Phase 1); within Phase 2, T004 can run parallel to T003 | Builder: pipeline (role-based aware)

> Every task is `[Shared]`: the walker is infrastructure serving all three user scenarios (one-walker-returns-the-complete-set / a-cut-short-walk-returns-the-partial-set-flagged-incomplete / a-partial-set-is-never-presented-as-complete) rather than any single one.
>
> **Cross-spec note**: 016 is purely additive and **library-only** — no cobra command, no exit-code edit, no existing file changed except a one-line comment correction in 010's `execute.go` (T004). It depends only on **landed** code: 010's `(*Client).Execute(reqCtx, req, out) (*Response, error)` seam and its typed code-free errors (`internal/apiclient`, Complete on main). The walker's `Result.Stop` is mapped by the *consumer*'s existing `classifyClientError` (011 §101) — `MalformedPageError` falls to the `default → RuntimeError(1)` fail-safe, so **no `Outcome`/`ExitCode` change and no renumbering**. `internal/paging` imports `apiclient` + `glassfrog`; neither imports `internal/cli`.
>
> **First-to-land coordination (role-based awareness)**: specs 012/013/014 (My Roles/Actions/Projects) are concurrently *Analyzed* and also depend on the shared `glassfrog.Pagination`. Per DECISIONS §109, whichever of 016/012-014 lands first **creates** `glassfrog.Pagination`; the others **reuse** it (T001 is reuse-or-create). 016 additionally introduces the generic `glassfrog.Page[T]` envelope, **superseding** the per-resource concrete `MyRolesResponse` named in 012/013's not-yet-landed plans (ADR-2; `/score:deprecate` suggested) — no refactor, since none has landed.

---

## Branching Guidance

**Pipeline mode**: `spec/016-pagination/base` → `spec/016-pagination/task-1`, `…/task-2`, `…/task-3`, `…/task-4` (one task branch per T-id, merged back into the spec base).

**Role-based awareness**: specs 012–014 are in-progress in parallel sessions and share the `glassfrog.Pagination` type T001 touches. Coordinate the first-to-land creation of `glassfrog.Pagination` (and the generic `Page[T]`) so the others reuse rather than fork a second envelope — grep `internal/glassfrog` for an existing `Pagination`/`Page` before adding, and reuse if present.

---

## Phase 1: Shared list envelope + the `internal/paging` walker [Shared]

- [x] **T001** [Shared] Add (reuse-or-create) the shared `glassfrog.Pagination` and the generic `glassfrog.Page[T]` list envelope — `Pagination` reused from 012 (now landed); added generic `Page[T]`+`Meta` in new `page.go`, 2 decode tests (envelope + absent-meta)
  - **Scope**: In `internal/glassfrog` (the leaf schema package — imports nothing internal), add the generic list envelope `Page[T any]{ Data []T \`json:"data"\`; Meta Meta \`json:"meta"\` }` where `Meta` is `{ Pagination Pagination \`json:"pagination"\` }`, over the shared `Pagination{ PerPage int \`json:"per_page"\`; HasNextPage bool \`json:"has_next_page"\`; NextCursor string \`json:"next_cursor"\` }`. **Reuse-or-create**: if a concurrently-landed paginated read (012–014) already added `Pagination`, reuse it; otherwise create it (DECISIONS §109 first-to-land rule). Decoding must be tolerant of an **absent** `meta.pagination` (it decodes to the zero value → `HasNextPage=false`, the non-paginated-endpoint case). No logic, no internal imports.
  - **Acceptance criteria**:
    - `glassfrog.Page[T any]` and `glassfrog.Pagination` exist with the JSON tags above; `T` is the only per-read variable (`Page[Role]`, `Page[Action]`, …)
    - Decoding a body with no `meta.pagination` yields `Page[T]{Data:…, Meta:{Pagination:{HasNextPage:false}}}` (no error)
    - `internal/glassfrog` still imports nothing internal; `go build ./...` and `go vet ./...` clean
    - If `Pagination` already exists from a sibling spec, it is reused (not duplicated)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-2 (generic `Page[T]` envelope, paging read from the decoded body's `meta.pagination`, supersedes the per-resource envelope)
  - **Interface references**: interface-spec.md — Envelope contract (`glassfrog.Page[T]` + shared `glassfrog.Pagination`)
  - **Scenario references**: pagination.feature: "A non-paginated endpoint returns in one response"
  - **Risk**: ⚠️ Coordinate with 012–014's parallel sessions on the first-to-land creation of `glassfrog.Pagination` — grep before adding so a second envelope isn't forked.

- [x] **T002** [Shared] Implement the `internal/paging` walker — `All[T]` over an `Executor` seam, with `Result[T]`, `WithPageSize`, and `MalformedPageError` — RED-first unit tests — 13 unit tests via a fake Executor (single/multi/absent-meta/empty, mid-walk-429 + first-page-failure partial-retain, blank + repeated cursor no-loop tripwires, cancellation, query-preservation/no-mutation, WithPageSize, nil-query); import graph verified clean of internal/cli
  - **Scope**: Create `internal/paging`. Define the one-method seam `Executor interface { Execute(reqCtx context.Context, req apiclient.Request, out any) (*apiclient.Response, error) }` (satisfied by `*apiclient.Client`), the result `Result[T any]{ Records []T; Complete bool; Stop error; Pages int }`, the typed `MalformedPageError{ Page int }`, and the `Option`/`WithPageSize(n int) Option` functional option (default page size **500**). Implement `All[T any](reqCtx context.Context, ex Executor, req apiclient.Request, opts ...Option) Result[T]`: loop — **clone** `req.Query` (never mutate the caller's map), set `per_page` to the configured size and `cursor` to the prior page's `next_cursor` (omitted on the first request, preserving all other params), `ex.Execute(reqCtx, …, &glassfrog.Page[T]{})`, append `page.Data` in order, then branch on `page.Meta.Pagination`: `HasNextPage=false` (incl. absent meta) → `Result{Complete:true}`; `HasNextPage=true` + an **advancing** `next_cursor` (non-blank AND different from the cursor just sent) → continue; `HasNextPage=true` + a **non-advancing** cursor (blank/absent **or equal to the cursor just sent**) → `Result{Complete:false, Stop:&MalformedPageError{Page:n}}` (no loop); any `Execute` error → `Result{Records:<so far>, Complete:false, Stop:err}`. Track the prior cursor across iterations to make the advance comparison. `Pages` counts requests issued. No exit-code mapping, no printing, never reads the token. RED-first unit tests via a **fake `Executor`** decoding canned pages into `out`.
  - **Acceptance criteria**:
    - `All[T]` returns a complete `Result` for: a single page; multiple pages (cursor threaded, `Data` concatenated in API order, `Pages` counted); an absent `meta.pagination` (one page, complete); an empty `data:[]` (complete, zero records)
    - A mid-walk `Execute` error (e.g. a `429` `*apiclient.ResponseError`, or `*TransportError`) returns `Complete:false` with the **records gathered so far retained** and `Stop` set to that error; a first-page failure returns empty records, `Complete:false`, `Stop` set
    - A **non-advancing** cursor under `HasNextPage=true` returns `*MalformedPageError` with `Complete:false` and **does not loop** — covering BOTH a blank/absent `next_cursor` AND a `next_cursor` identical to the cursor just sent; each variant pinned by a call-counting fake `Executor` tripwire (the repeated-cursor case asserts the walker stops rather than re-requesting the same page)
    - Each page request preserves the caller's other query params (`q`/`include`) and sets only `per_page` (default 500, overridable via `WithPageSize`) + `cursor`; the caller's `req.Query` is not mutated
    - No `paging` output renders a token; the walk reads no flag/env/credentials file directly; `internal/paging` does not import `internal/cli`
    - RED-first unit tests cover every branch above; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-1 (library-only walker in `internal/paging` over an `Executor` seam), ADR-3 (explicit `Result[T]` partial-flagged-incomplete), ADR-4 (default 500 + `WithPageSize` + clone-query, no silent clamp), ADR-5 (unbounded walk + non-advancing-cursor guard)
  - **Interface references**: interface-spec.md — Entry points (`All`, `WithPageSize`), seam contract (`Executor`), Output contract (`Result[T]`), error type (`MalformedPageError`), Interactions, Error Communication
  - **Scenario references**: pagination.feature: "A single page completes the set", "Multiple pages are assembled into a complete set", "An empty result set is a complete answer", "The caller's query parameters are preserved across pages", "A mid-walk page failure yields a partial set flagged incomplete", "A first-page failure yields an empty set flagged incomplete", "A blank cursor under has_next_page does not loop", "A repeated cursor under has_next_page does not loop", "A cancelled request context stops the walk with the partial set", "Records are returned in API order without reordering or de-duplication", "The walker re-resolves nothing", "A partial result is never indistinguishable from a complete one"
  - **Risk**: ⚠️ Don't mutate the caller's shared `req.Query` map — clone per page. ⚠️ The non-advancing-cursor guard must fail loud for BOTH a blank/absent cursor AND a cursor equal to the one just sent (a wrong `cursor` param name or an ignoring API loops past a blank-only check — risk H-2/H-9) — track the prior cursor and pin both variants with a tripwire. ⚠️ Keep the partial set on `Stop` — `Result.Records` must be returned even when `Stop != nil` (the whole point of ADR-3); resist a `return Result{}` on error.

## Phase 2: Executable acceptance + 010 comment correction [Shared]

- [x] **T003** [Shared] Make the 016 driving scenarios pass via godog in a new `internal/paging` suite scoped to its own feature file — new `pagination_bdd_test.go` suite (Paths→pagination.feature only), 10 behavioral scenarios pass (10 scenarios, 42 steps); 3 `@validation` scenarios kept `@wip` for validate; cancellation scenario implemented; reuses the unit-test fake Executor
  - **Scope**: Add godog step definitions for `features/silent-truncation/pagination.feature` (all three Rule blocks) in a **new** `internal/paging` godog suite whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `paging.All[T]` over a **fake `Executor`** returning canned pages keyed by `cursor` (single-page / multi-page / non-paginated / empty / query-preservation / mid-walk-429 / first-page-failure / malformed-cursor / cancellation). Step helpers return errors, never panic (LEARNINGS). Remove `@wip` from the behavioral scenarios; keep the three `@validation` scenarios `@wip` (held out for validate). The proposed `@wip` "A cancelled request context stops the walk with the partial set" scenario is implemented unless the developer dropped it during preview.
  - **Acceptance criteria**:
    - Every non-`@validation` 016 scenario has an executable, passing path against the fake `Executor`
    - `@wip` removed from those scenarios; the three `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `pagination.feature`; the suite runs and reports its own `N scenarios (N passed)` count
    - No real network or filesystem touched (fake `Executor` only); `go build ./...`, `go vet ./...`, and the suite run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 2 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: pagination.feature: all 016 behavioral Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — point `Paths` at the single feature file, not the directory. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations in sibling suites and match phrasing before adding bindings; helpers return errors, never panic.

- [x] **T004** [Shared] [P] Correct 010's stale "Link/paging headers" comment to "decoded body `meta.pagination`" — `execute.go` `Response` doc now states 016 reads paging from the body's `meta.pagination` via `glassfrog.Page[T]`, not a header; comment-only, no behavior change
  - **Scope**: Amend the comment in `internal/apiclient/execute.go` (the `Response` doc and any inline note) that says Pagination (016) "reads Link/paging headers" — the v5 API carries paging in the **response body** at `meta.pagination`, which the walker decodes via an enveloped target; `Response` exposes only status+headers. A comment-only change; no behavior, no signature, no test changes. (Optionally note the correction where 010's plan/interface text is co-located, if a docs sweep is in scope — otherwise leave the spec artifacts as historical record.)
  - **Acceptance criteria**:
    - `execute.go`'s comment no longer claims 016 reads paging from a response header; it states paging is read from the decoded body's `meta.pagination`
    - No code/signature/behavior change; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 2; ADR-2 (corrects 010's header assumption); Risks ("Stale 'Link header' assumption spreading")
  - **Risk**: ⚠️ Comment-only — resist scope creep into refactoring `Response`; the header→body correction is the entire change.
