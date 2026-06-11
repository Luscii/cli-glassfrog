# Tasks: Actor Directory

**Feature**: 048-actor-directory
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/actors-disconnected-from-governance/actor-directory.feature

---

## Dependency Graph

Phase 1: `internal/render` `actors` list key (1 task, no phase dependencies) [Shared]
Phase 2: The `actors` command (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | T001 startable immediately | linear chain (T001 → T002 → T003) | Builder: pipeline

> Plan-faithful: the plan's two phases map here as render (T001) and command (T002), plus the executable-acceptance task (T003) that the plan's Cross-cutting/testing section calls for. **Unlike 038, the command depends on the render task** — 038's list reused the landed `projects` render key (014), so its list command ran parallel with the render task; 048's `actors` key is **new** (plan ADR-4 — `ResourceMe` renders one actor inside the `me` document, not a directory list), so T002 needs T001. The result is a clean linear chain, with only T001 startable immediately. **There is no schema phase** — plan ADR-2 reuses `internal/glassfrog.Actor` (grown by Identity Read, 011) unchanged and instantiates the landed generic `Page[Actor]` (016). Story labels: the spec's four user scenarios are US1 (find whom to contact for a role, `--role-id`), US2 (turn a name into an id, `--query`), US3 (tell people apart from agents, `--kind`), and US4 (trust the directory is whole, or be told it is incomplete — the completeness benefit, added in the 2026-06-11 clarify session). The single `actors` command serves all four (the completeness behavior is realized in T002's walk + `--first-page` signalling), so T002 is `[Shared]`; T001/T003 are `[Shared]` infrastructure.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/015/016/017/018/019/020/025 are Complete with their packages shipped — `internal/glassfrog.Actor` (id/name/kind/timestamps, 011), `Page[T]`/`Pagination` + `paging.All` (016), `RetryExecutor` (017), `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the persistent `--base-url`/`--output`/`-o`, the walked-list render machinery (`aggregateRawData` + `renderFn`, the roles/domains/policies/projects pattern) over `internal/output`/`internal/render` (018/019/020), and the reject-unknown local-validator shape (`internal/cli/include.go`, `status.go`). The 048 base cuts from current main with no sequencing caveat. **Existing main state 048 builds on**: there is **no** `actors` render key (T001 adds it) and **no** `actors` command (T002 adds it). 048 adds a **new `--kind` validator** but **no** new `Outcome` category, `ExitCode` case, generic type, or root flag.
>
> **The discovery entry of the Actor Reads slice.** `actors` reads `GET /actors` (the unified, ungated endpoint); Actor Read (049) and Actor Assignments (050) — the per-id read and the footprint — are their own specs. This is the first read keyed purely on flags (`cobra.NoArgs`), with no `people`/`agents` command (plan ADR-1).

---

## Branching Guidance

**Pipeline mode**: `spec/048-actor-directory/base` → `spec/048-actor-directory/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base). Cut the base from current main (all cross-spec dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling reads (the actor-reads wave: 049 Actor Read, 050 Actor Assignments, 047 Role Fillers). 048 touches `internal/cli` (new `actors.go`, a new `validateKind` or `validateIncludeSet` parameterization) and `internal/render` (one new `actors` key); it makes **no** change to `internal/glassfrog` (model reused as-is — plan ADR-2), so it carries no cross-package contract change to coordinate. If 049/050 land first and grow `glassfrog.Actor` (011 ADR-1) for the `?include` embeds, 048's directory rows are unaffected (additive growth; 048 renders only id/name/kind).

---

## Phase 1: `internal/render` `actors` list key [Shared]

- [ ] **T001** [Shared] Add the `actors` list render key + `ActorsView` + templates — golden + registry-guard tests; `internal/render` (new `actors.{full,compact}.tmpl`, `ActorsView`, `ResourceActors`) + tests
  - **Scope**: In `internal/render`, add **one new** render key `actors` (a flat homogeneous list of `glassfrog.Actor`, via an `ActorsView` mirroring the landed list views). Add the `ResourceActors` constant to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates: `actors.full` — one block per actor (`<per_…|agt_…>  [<kind>]` then an indented `Name: <name>` line) with the empty-set line `no actors`; `actors.compact` — one line per actor (`<per_…|agt_…>  [<kind>]  <name>`) with the same empty line. Render `name` verbatim — never truncated or reflowed (CONSTITUTION VI). Use 019's absence-guard discipline for any empty field. `ResourceMe` (the single actor inside the `me` document) is **not** reused — touch no existing key or template. Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `actors` key renders both `full` and `compact`; each row shows the `per_`/`agt_` id, the `kind` badge, and the name
    - A mixed page of humans and agents renders each row with its correct `kind` badge; an empty list renders exactly `no actors`
    - The registry-exhaustiveness guard passes with the new `actors` key carrying both formats; golden tests pin each template (mixed kinds + empty set)
    - `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: None (the view struct references the landed `glassfrog.Actor`).
  - **Plan reference**: Phase 1 (Render); ADR-4 (new `actors` render key; `ResourceMe` not reused)
  - **Interface references**: interface-cli.md — Output (human `full`/`compact` shapes, empty line), Consistency Notes (render key)
  - **Scenario references**: actor-directory.feature: "A directory lists every actor in the organization", "Default output carries no raw API envelope"
  - **Risk**: ⚠️ Add the new `actors` key only — do not reuse or fork `ResourceMe`. ⚠️ Explicit-absence/empty handling; never invent a value. ⚠️ Add `ResourceActors` to the guarded set so exhaustiveness still holds. ⚠️ Never truncate/reflow `name` (CONSTITUTION VI).

## Phase 2: The `actors` command [Shared]

- [ ] **T002** [Shared] Add the `actors` command + `actorsSeam` + `--kind`/`--role-id`/`--query`/`--first-page`/`--per-page` + completeness + wiring — RED-first unit tests for every branch; new `internal/cli/actors.go` (+ the `--kind` validator) + tests + `Assemble()` wiring
  - **Scope**: New `internal/cli/actors.go`. `newActorsCommand(seam actorsSeam) *cobra.Command`: a guard-registered leaf (`Use:"actors"`, **`Args: cobra.NoArgs`**, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), declares `--kind`, `--role-id`, `--query`/`-q`, `--first-page`, `--per-page`; reads the persistent `--base-url`/`--output`; delegates to pure `runActorsList`. Define the shared `actorsSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client` executor; prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Executor`). Add the **`--kind` reject-unknown validator** against `{human, agent}` (a thin `validateKind` or a parameterization of `validateIncludeSet` — plan ADR-3; the existing helper hard-codes `--include` in its message, so do not regress that message). `runActorsList(cfg) (Outcome, error)`: **resolve `--output` first (020), then validate `--kind`** (the one closed-enum input) — both pure and **before** assembly, so a bad format / bad kind is a fail-fast `UsageError(2)` with no request; output-first preserves error precedence when both are invalid (interface § Interactions); build the `kind`/`role_id`/`q` query parameters each only when its flag is `Changed()` and non-empty (the combinable filters; the 033/038 optional-flag discipline); default path walks `GET /actors` to completion using the landed roles/domains/policies/projects two-track list pattern (NOT `renderResult[T]`, the single-page `/me*` dispatch): for `json`/`yaml`, walk `paging.All[json.RawMessage]` then `aggregateRawData` into the `{data:[…]}` document (per-record raw bytes preserved, per-page `meta` dropped); for `full`/`compact`, walk `paging.All[Actor]` then `renderFn(ResourceActors, …)` over an `ActorsView`. The filters must be carried on **every** page request of the walk, not just the first (plan Risk). `--first-page` does one `Execute` into the corresponding `Page[json.RawMessage]`/`Page[Actor]` and writes a "more actors exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The `--role-id`/`--query` values are passed through unvalidated (plan ADR-3) so a malformed `role_id` surfaces the API's clean non-2xx. Wire `MustRegister(root, newActorsCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog actors` walks every page (`paging.All`) and prints the projection of every actor with no filter sent; exits 0
    - An org/filter combination with no matching actor prints the `actors` empty line (`no actors`) and exits 0
    - `--kind agent` sends `kind=agent`; `--role-id role_x` sends `role_id=role_x`; `--query q` sends `q=q`; the three combine when all present; an omitted/empty filter sends nothing
    - `--kind robot` (unsupported) is a `UsageError(2)` naming the value + supported set (`agent, human`), **no request sent** (transport tripwire)
    - A positional argument is a `UsageError(2)` via cobra's `NoArgs`, **no request sent**
    - `--first-page` against a multi-page directory prints one page, writes a "more actors exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - The filters are retained on every page request of a multi-page walk (a malformed `--role-id` surfaces the API status, not a local rejection)
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the aggregated `{data:[…]}` document, `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (the new `actors` render key). Reuses the walked-list machinery `Page[json.RawMessage]`/`Page[Actor]` + `aggregateRawData` + `renderFn` (016/033/034/038), `classifyClientError` (011/015), and the reject-unknown validator shape (`include.go`/`status.go`).
  - **Plan reference**: Phase 2 (Command); ADR-1 (flag-only `actors`, `cobra.NoArgs`, no `people`/`agents`), ADR-3 (`--kind` validated locally; `--role-id`/`--query` passed through), ADR-2 (`Page[Actor]` reused, no schema growth); Cross-cutting (completeness reuses 025 ADR-3; filters on every page)
  - **Interface references**: interface-cli.md — `actors` Surface, filter flags, Output, Interactions (validation order + completeness), Error Communication
  - **Scenario references**: actor-directory.feature: "A role-id filter lists the actors filling that role", "A malformed role-id filter fails with the API status", "A missing token fails as a not-authenticated usage error", "A directory lists every actor in the organization", "A free-text query matching no actor is a clean success", "A kind filter narrows the directory to agents", "An unsupported kind is rejected as a usage error", "A rejected kind issues no request", "The first-page opt-out stops at one page and signals more", "A multi-page directory walks to completion by default", "A mid-walk failure yields a partial set flagged incomplete", "The filters are carried on every page of the walk"
  - **Risk**: ⚠️ `cobra.NoArgs` (not `ExactArgs`) — the directory takes no positional; a positional is the structural usage error (no hand-rolled guard). ⚠️ Reuse `classifyClientError`/`aggregateRawData`+`renderFn`/`paging.All`/`RetryExecutor` — inline no second copy of the `errors.As` chain, render branch, or page loop. ⚠️ Structured output is the aggregated `{data:[…]}` over `Page[json.RawMessage]` (the roles/domains/policies/projects pattern), NOT a decode-and-re-encode of `[]Actor` and NOT a single page's raw envelope. ⚠️ Each filter sent only when `Changed()` and non-empty, and carried on EVERY page of the walk; only `--kind` is validated (`--role-id`/`--query` are free, passed through). ⚠️ Resolve `--output` first, then validate `--kind`, BEFORE assembly (output-first preserves error precedence per interface § Interactions) so the tripwire confirms no request on rejection. ⚠️ If parameterizing `validateIncludeSet`, keep its existing `--include` message intact for current callers. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `actor-directory.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/actors-disconnected-from-governance/actor-directory.feature` in a **new** `internal/cli` godog suite (e.g. `TestActorDirectoryFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `actors` through the shared seam over a fake base `http.RoundTripper` returning canned `GET /actors` responses (single-page, multi-page, mid-walk error, empty, kind-filtered, role-filtered), plus a transport tripwire for the no-request paths (unsupported `--kind`; positional argument). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection / param-assertion phrasings from the `me*`/`roles`/`projects` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` actor-directory scenario has an executable, passing path; `@wip` removed from them
    - The `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `actor-directory.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002 (the `actors` command — all behavioral scenarios must be implementable)
  - **Plan reference**: Phase 2 (Command, the single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: actor-directory.feature: all behavioral Rule-block scenarios (the `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `actor-directory.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mid-walk-error, multi-page, empty-list, kind/role-filtered, every-page-retains-filter, and no-request (tripwire) fakes so the completeness, empty, filter, and rejection scenarios genuinely exercise their paths.
