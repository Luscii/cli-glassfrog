# Tasks: Cross-Model Search

**Feature**: 041-cross-model-search
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/undiscoverable-governance/cross-model-search.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` schema (1 task, no phase dependencies) [Shared]
Phase 2: The `search` command + acceptance (2 tasks, depends on Phase 1 · T001) [Shared]

3 tasks total | T001 startable immediately | Builder: pipeline (with role-based awareness — see Branching Guidance)

> Every task is `[Shared]`: the single `search` command serves all four user scenarios (ranked discovery US1, type scoping US2, drill-in bridge US3, completeness US4). The plan's two phases map to: schema (T001 `SearchResult`) → the `search` command (T002) + executable acceptance (T003).
>
> **All hard dependencies are landed on main.** Per STATUS.md, 007/009/010/015/016/017/018/020 are Complete and 019's `internal/render` package is landed (the `*.full/compact.tmpl` templates ship from it). So the 041 base is cut from current main. The BACKLOG declares only 007/010.
>
> **No cross-spec schema coordination.** Unlike 025/026 (which share `RoleDetail`), `SearchResult` is a **wholly new** type shared with nothing — the heterogeneous result row (`type`/`id`/`title`/`excerpt`/`rank`/`role_id`) has no sibling that defines it. T001 creates it free of any first-to-land coordination.
>
> **Existing main state 041 builds on, not around**: 041 adds **one new sibling command** (`search`) — org-wide and cross-type, child of no resource group (plan ADR-1) — and **one new** render key (`search`), distinct from every shipped/planned key. 041 reuses the landed `paging.All`/`Page[T]`/`renderResult`/`classifyClientError`/`validateIncludeSet` machinery and adds **no** new `Outcome` category, `ExitCode` case, or root flag. The one helper note: `validateIncludeSet` hard-codes `--include` in its message, so T002 either parameterizes the flag name or adds a thin `validateTypes` sibling (plan ADR-3).

---

## Branching Guidance

**Pipeline mode**: `spec/041-cross-model-search/base` → `spec/041-cross-model-search/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all hard dependencies landed).

**Role-based awareness**: parallel Conductor workspaces carry other governance reads, but 041 shares **no** schema or command surface with them — `SearchResult` and the `search` command/render key are collision-free. 041 touches `internal/glassfrog`, `internal/cli`, and `internal/render`, but adds only new files/keys there (no growth of a shared type), so a parallel spec cannot collide on its definitions.

---

## Phase 1: `internal/glassfrog` schema [Shared]

- [ ] **T001** [Shared] Add the flat `SearchResult` type decoded via `Page[SearchResult]` — RED-first decode tests; new `internal/glassfrog/search.go` + `search_test.go`
  - **Scope**: In `internal/glassfrog`, add a `SearchResult` struct decoding a `data` row of the `GET /search` body: `Type string` (the `role`/`note`/`project`/`action`/`skill`/`actor`/`policy`/`domain` enum — decoded as a plain string, not a constrained Go type), `ID string`, `Title string`, `Excerpt *string` (nullable), `Rank float64`, `RoleID *string` (nullable). The list decodes the **existing** generic `Page[SearchResult]` (016) — do **not** define a new envelope. It is its **own** flat type — not a reuse/growth of `Role`/`RoleDetail` (a `SearchResult` carries `rank`/`excerpt` no resource has — plan ADR-2). Decoding tolerates unknown/extra fields; no transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A `GET /search` page fixture decodes into `Page[SearchResult]` with `Data` populated (mixed `type` values across rows) and `Meta.Pagination` read (`has_next_page`, `next_cursor`)
    - `Excerpt` and `RoleID` decode as nullable — a row with `excerpt: null` / no `role_id` decodes with those fields nil, and a row with both present populates them
    - `Rank` decodes as a float; an all-types row set preserves the order the fixture lists (decode order = response order)
    - Unknown/extra fields are ignored; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Reuses `Page[T]`/`Pagination` (016).
  - **Plan reference**: Phase 1 (Schema); ADR-2 (new flat `SearchResult`, decoded via `Page[SearchResult]`, not a resource projection); System Architecture (`internal/glassfrog`)
  - **Interface references**: interface-cli.md — Surface (search output shape, nullable fields)
  - **Scenario references**: cross-model-search.feature: "A query searches across all resource types", "Each result carries the bridge into a read command"
  - **Risk**: ⚠️ Keep `SearchResult` separate from `Role`/`RoleDetail` (different shape: `rank`/`excerpt`/heterogeneous `type`). ⚠️ Reuse the generic `Page[T]`; never define a 041-local envelope. ⚠️ `Excerpt`/`RoleID` are nullable (pointers) — a null must decode cleanly, not error. ⚠️ Token is never a field.

## Phase 2: The `search` command + acceptance [Shared]

- [ ] **T002** [Shared] Add the `search <query>` command: `--types` validation, verbatim query forwarding, the page walk + `--first-page` opt-out + completeness signalling + `--per-page`, the `search` render, wiring — RED-first unit tests for every branch; new `internal/cli/search.go` + `search_test.go`, new `search` render templates
  - **Scope**: New guard-registered, explicitly-wired leaf. `newSearchCommand(seam searchSeam) *cobra.Command`: `Use:"search <query>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`; reads the persistent `--base-url` (011) + `--output`/`-o` (020); declares local `--types`, `--first-page`, `--per-page`. Delegates to pure `runSearch(cfg, query)`. Validate **before** assembly: resolve `--output` (020); `validateTypes` (the `validateIncludeSet` reject-unknown shape — either pass the flag name as a parameter or add a thin sibling so the message names `--types`, not `--include`) rejects any `--types` value outside `{role,note,project,action,skill,actor,policy,domain}` as UsageError(2) with a transport tripwire. Attach the positional **verbatim** as `query` (no parse/escape/normalize/split — plan ADR-1); when `--types` is set, attach its comma-separated value as `types` (omit the param entirely when unset). Default path walks `GET /search` to completion via `paging.All[SearchResult]` over the seam's executor (`RetryExecutor` wrapping `NewClientFromOS`, 016/017), **carrying `query` (and `types`) on every page request**; `--first-page` does a single `Execute` into `Page[SearchResult]` (no walk), rendering the first page and writing one stderr note when `HasNextPage` (exit 0); a mid-walk failure (`Result.Complete == false`, non-nil `Stop`) renders the partial `Records`, writes one stderr "incomplete — <cause>" note, and exits non-zero via `classifyClientError(Stop)`. The default walk MUST pass `WithPageSize(100)` — the `/search` `per_page` maximum — because `paging.All`'s generic default is 500, which `/search` rejects with `400`; `--per-page` overrides it (the API owns the 1–100 range). Dispatch through `renderResult("search", format, records)` — **preserving the API relevance order** (no client re-sort/de-dup/filter — plan ADR-2). Register **new** `search` `full`/`compact` templates in `internal/render`: one block/line per result in received order, a per-row `type` badge, `Excerpt:` with the `—` absence marker when null, the `Role:` line only when `role_id` is present, an empty set prints `No results.`; registry exhaustiveness guard (PR #10 `len`+comma-ok). Wire `MustRegister(root, newSearchCommand(...))` in `Assemble()`. Never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog search onboarding` walks every page (`paging.All`) and prints the complete relevance-ordered result set; exits 0; an empty result prints `No results.` and exits 0
    - The outbound `query` parameter equals the positional **byte-for-byte** (a query with websearch operators is unmodified — no escaping/splitting); a multi-word query is supplied quoted (`search "strategy review"`); a missing or extra positional is UsageError(2) before any request (cobra `Args`)
    - `--types role,project` sends `types=role,project`; omitting `--types` sends no `types` param; an unsupported `--types` value is UsageError(2) naming the value + the 8-value set, **no request sent** (tripwire)
    - the default walk requests `per_page=100` (the `/search` max), **not** paging's generic 500 — a fixture asserts the first page request carries `per_page=100`; `--per-page N` overrides it
    - The rendered result order equals the API response order (no client re-sort/de-dup); a null `excerpt` renders as `—`, a null `role_id` omits the `Role:` line — no fabricated text (CONSTITUTION VIII)
    - `--first-page` against a multi-page result prints only the first page, writes a "more results exist" stderr note, exits 0; a mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`); `query`+`types` are retained on every page request of the walk
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); a malformed-query `*ResponseError` (`400`) → APIError(3) (and `403`/`429` → PermissionError(4)/RateLimited(5)); `*DecodeError` → APIError(3) (031); `*MalformedPageError` → RuntimeError(1); base-URL error / invalid `--output` → UsageError(2)
    - `-o json`/`yaml` emit the raw payload verbatim (018); the new `search` render key has both `full`/`compact` formats and passes the registry guard; no `Outcome`/`ExitCode`/root flag added
    - No output renders the token; all branches run offline over the fake seam; `go build`/`go vet` clean
  - **Dependencies**: T001 (`SearchResult` decode target / `Page[SearchResult]`). Reuses 009/010/011/015/016/017/018/019/020 (all landed).
  - **Plan reference**: Phase 2 (Search command); ADR-1 (`search <query>` sibling, `ExactArgs(1)`, verbatim forwarding), ADR-2 (one `search` render key, preserve order, nullable-as-absent), ADR-3 (`--types` reject-unknown), ADR-4 (walk + `--first-page` opt-out, 025/026 verbatim); Cross-cutting (error handling, output, testing)
  - **Interface references**: interface-cli.md — `glassfrog search` Surface, Interactions (query forwarding, completeness), Error Communication
  - **Scenario references**: cross-model-search.feature: "A query searches across all resource types", "A multi-word websearch query is forwarded verbatim", "A missing token fails as a not-authenticated usage error", "A query the API rejects as malformed fails with the API status", "A search matching nothing is a clean success", "A missing query is a usage error", "A search is scoped to specific types", "An unsupported type is rejected as a usage error", "Each result carries the bridge into a read command", "A multi-page result walks to completion by default", "The first-page opt-out stops at one page and signals more"
  - **Risk**: ⚠️ Forward the query **verbatim** — no parse/escape/normalize/split (plan ADR-1); a join would silently rewrite it. ⚠️ Preserve the API relevance order — never re-sort/de-dup/filter (plan ADR-2). ⚠️ Carry `query`+`types` on **every** page of the walk, not just the first (plan Risk). ⚠️ `validateTypes` reuses the `validateIncludeSet` shape but must name `--types` in the message (the landed helper hard-codes `--include`). ⚠️ Reuse 025/026's pagination shape verbatim — `paging.All` default + single-page `--first-page` signal; never silently truncate (CONSTITUTION VI). ⚠️ Render a null `excerpt`/`role_id` as absent, never fabricated (CONSTITUTION VIII; `missingkey=error` guards). ⚠️ Reuse `classifyClientError`/`renderResult`/`paging.All`/`RetryExecutor` — inline no second chain, render branch, or page loop. ⚠️ Temp-file capture in tests, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

- [ ] **T003** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `cross-model-search.feature`; un-@wip the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/undiscoverable-governance/cross-model-search.feature` in a **new** `internal/cli` godog suite (e.g. `TestCrossModelSearchFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `search` through its seam over a fake base `http.RoundTripper` returning canned `GET /search` responses — a single-page mixed-type set, an empty set, a multi-page set (to exercise the walk + `--first-page` + the query/types-on-every-page assertion), a mid-walk error, a malformed-query `400`, and a row with a null `excerpt`/`role_id`. Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Reuse existing `internal/cli` step phrasings where assertions already exist (grep `sc.Step(` registrations first — exit-code, stderr-substring, request-query, "no request sent" tripwire, and projection phrasings are shared with the `me*`/roles/subroles reads); step helpers return errors, never panic. **One reuse caveat**: the existing invocation step splits with `strings.Fields(invocation)` (whitespace), which cannot pass a quoted multi-word `<query>` as a single argument — exactly what this feature requires. This suite needs a quote-aware splitter (or an explicit arg-list step) so a multi-word query reaches cobra as one positional; and the stdout-literal step is case-sensitive (`strings.Contains`), so assert the exact `No results.` literal (capital N, trailing period), not a lowercased variant.
  - **Acceptance criteria**:
    - Every non-`@validation` cross-model-search scenario has an executable, passing path; `@wip` removed from them (the 11 spec-derived + 2 architecture-informed behavioral scenarios)
    - The 4 `@validation` scenarios keep `@wip` (held for validate)
    - The new suite's `Paths` names only `cross-model-search.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002 (the `search` command — all behavioral scenarios must be implementable)
  - **Plan reference**: Phase 2 (Search command — "godog + unit tests"); System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: cross-model-search.feature: all behavioral Rule-block scenarios (the 4 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `cross-model-search.feature` only (not the directory); verify it reports its own count. ⚠️ Grep existing `sc.Step(` registrations and reuse shared read phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mixed-type, empty, multi-page, mid-walk-error, malformed-query, and null-field fakes so the relevance-order, completeness, and absence-marker scenarios genuinely exercise the behavior. ⚠️ The "query carried on every page" scenario needs the fake to assert the param on the page-2 request, not only page 1.
