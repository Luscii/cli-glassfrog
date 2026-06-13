# Tasks: Role Fillers

**Feature**: 047-role-fillers
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/who-to-contact-for-a-role/role-fillers.feature

---

## Dependency Graph

Phase 1: `internal/render` `fillers` list key (1 task, no phase dependencies) [Shared]
Phase 2: The `fillers` command (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | T001 startable immediately | single-phase-deep linear chain | Builder: pipeline

> Plan-faithful: the plan's single implementation phase has three concerns — render path (T001), the one command (T002), and BDD acceptance (T003). **Unlike 038 there is no second command and no schema phase**: plan ADR-1 ships a single `fillers <role-id>` list (the API exposes no `GET /assignments/{id}`), and plan ADR-2 reuses `internal/glassfrog.Assignment` (grown by Role Reads 025, with `focus`/`elected_until` and the embedded `actor` already present) unchanged. The command (T002) references the **new** `fillers` render key, so — unlike 038's list command, which reused the landed `projects` key — T002 **depends on T001** (no parallelism). Every task serves all three of the spec's user scenarios (US1 list who fills a role, US2 show focus/election, US3 trust completeness) at once, so all carry `[Shared]`.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/015/016/017/018/020/025/031/032/035 are Complete with their packages shipped — `internal/glassfrog.Assignment` (the model, grown by 025, carrying `id`/`actor_id`/`role_id`/`focus`/`elected_until`/embedded `actor`), `Page[T]`/`Pagination` + `paging.All`, `RetryExecutor`, `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the `reportFailure` format-aware failure chokepoint (032), the persistent `--base-url`/`--output`/`-o` (011/020) + `-o <template-ref>` (035), and the walked-list render machinery (`aggregateRawData` + `renderFn`/`writeHuman`, the roles/domains/policies/projects pattern) over `internal/output`/`internal/render` (018/019/020) all ship from main. The 047 base cuts from current main with no sequencing caveat. **Existing main state 047 builds on**: there is **no** `fillers` render key (T001 adds it) and **no** `fillers` command (T002 adds it). 047 adds **no** new `Outcome` category, `ExitCode` case, validator, generic type, or root flag.
>
> **Not a dependency on 048.** Actor Directory (048) is a sibling in the Actor Reads world but 047 does **not** depend on it: the embedded `actor{id,name,kind}` lives inside `glassfrog.Assignment` (025), not on 048's `glassfrog.Actor`. The new `fillers` render key is independent of 048's `actors` key. Actor Assignments (050) will reuse `glassfrog.Assignment` likewise — the role-scoped read here is its mirror.

---

## Branching Guidance

**Pipeline mode**: `spec/047-role-fillers/base` → `spec/047-role-fillers/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling reads. 047 touches `internal/cli` (new `fillers.go`) and `internal/render` (one new list key); it makes **no** change to `internal/glassfrog` (model reused as-is — plan ADR-2), so it carries no cross-package contract change to coordinate.

---

## Phase 1: `internal/render` `fillers` list key [Shared]

- [x] **T001** [Shared] Add the `fillers` list render key + `FillersView` + templates — golden + registry-guard tests — full/compact goldens + empty + absent/verbatim-focus + blank-name marker; registry guard covers `fillers`; no findings
  - **Scope**: In `internal/render`, add **one new** list render key `fillers` (data `[]glassfrog.Assignment` via a `FillersView`, mirroring the landed list views `ProjectsView`/`ActorsView`). Add the `ResourceFillers` constant (`"fillers"`) to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates rendering, per filler, the **filling actor first** (the "whom to contact" answer) plus the assignment's governance context: `fillers.full` — one block per filler: the actor id (`per_`/`agt_` prefix) + a `[kind]` badge, then `Name`, `Focus` (with `(none)` when absent), `Elected until` (with `(not an elected seat)` when absent); `fillers.compact` — one line per filler: `<per_…|agt_…>  [<kind>]  <name>  — focus: <focus|—>; elected until: <date|—>`. Use 019's absence-guard discipline (`{{if eq (trimSpace …) ""}}…`) for the nullable `focus`/`elected_until`, and render `focus` verbatim (it is user text — never truncated/reflowed, CONSTITUTION VI). The assignment id (`asgn_…`) is **not** rendered in the human projection (not a spec row field) — it remains in the structured output. Add a `fillers` empty line (`no fillers`). Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `fillers` key renders both `full` and `compact`; each row leads with the actor id + `[kind]` and shows the actor name, focus, and election expiry
    - A filler with no `focus` renders `(none)`; a non-elected assignment renders `(not an elected seat)` for `elected_until` — never `<no value>` or an invented value
    - A person (`per_`) and an agent (`agt_`) filler are visually distinguishable by id prefix and `[kind]` badge
    - The empty-list render is the `no fillers` line
    - The registry-exhaustiveness guard passes with the new `fillers` key carrying both formats; golden tests pin each template (present and absent focus/election)
    - No existing render key/template is touched; `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: None (the view struct references the landed `glassfrog.Assignment`).
  - **Plan reference**: Phase 1 (Implementation Strategy step 1); ADR-2 (reuse `Assignment` as-is, add `fillers` render key)
  - **Interface references**: interface-cli.md — Output (`full`/`compact` row shapes, explicit-absence markers), Consistency Notes (render key)
  - **Scenario references**: role-fillers.feature: "A filler shows its focus and election expiry", "A person and an agent filler are distinguished by kind"
  - **Risk**: ⚠️ Add the `fillers` list key only — touch no existing key/template. ⚠️ Explicit-absence guards for both nullable fields (`focus`, `elected_until`); never invent a value. ⚠️ Render `focus` verbatim (no truncation/reflow, CONSTITUTION VI). ⚠️ Add `ResourceFillers` to the guarded set so exhaustiveness still holds.

## Phase 2: The `fillers` command [Shared]

- [ ] **T002** [Shared] Add the `fillers <role-id>` command + `fillersSeam` + completeness + wiring — RED-first unit tests for every branch
  - **Scope**: New `internal/cli/fillers.go`. `newFillersCommand(seam fillersSeam) *cobra.Command`: a guard-registered leaf (`Use:"fillers <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares **no filter flags and no `--include`** (plan ADR-3) — only the shared `--first-page`/`--per-page` walk flags; reads the persistent `--base-url`/`--output`; delegates to a pure `runFillersList`. Define the `fillersSeam` of the same shape as `projectsSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client` executor, sleep, resolve selection, read template source; prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Executor`). `runFillersList(cfg) (Outcome, error)`: resolve `--output` (020) **before** assembly (the only pre-assembly check — there is no closed-enum input to validate, plan ADR-3); default path walks `GET /roles/{role_id}/assignments` to completion using the landed two-track walked-list pattern (NOT `renderResult[T]`): for `json`/`yaml`, walk `paging.All[json.RawMessage]` then `aggregateRawData` into the `{data:[…]}` document (per-record raw bytes preserved, per-page `meta` dropped); for `full`/`compact`, walk `paging.All[glassfrog.Assignment]` then `writeHuman(…, render.ResourceFillers, …)` over a `FillersView`. `--first-page` does one `Execute` into the corresponding `Page[json.RawMessage]`/`Page[Assignment]` and writes a "more fillers exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The role id is passed through (`url.PathEscape`, no local validation — plan ADR-3) so an unknown id surfaces the API's clean `404`. Failures render format-aware through the landed `reportFailure` (032). Wire `MustRegister(root, newFillersCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog fillers role_0123` walks every page (`paging.All`) and prints the projection of all the role's fillers; exits 0
    - A role with no fillers prints the `fillers` empty line (`no fillers`) and exits 0
    - The request carries the endpoint's default `actor` include so each row has a name/kind; the command declares no `--include` and no filter flag (an unknown flag is a cobra `UsageError(2)`)
    - A missing or extra positional is a cobra `ExactArgs(1)` `UsageError(2)`, **no request sent** (transport tripwire)
    - `--first-page` against a multi-page list prints one page, writes a "more fillers exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*AuthError{CredentialError}` → RuntimeError(1); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); unknown role id → APIError(3) (typically 404); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the aggregated `{data:[…]}` document (per-record raw bytes, per-page `meta` dropped — never a single page's envelope), `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/validator/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (the `fillers` render key + `FillersView` the command renders through).
  - **Plan reference**: Phase 1 (Implementation Strategy step 2); ADR-1 (single command, `ExactArgs(1)`, no plural/singular pair), ADR-3 (no filters/no `--include`, validate nothing locally, id passed through); Cross-cutting (completeness reuses 025 ADR-3, failure via 032)
  - **Interface references**: interface-cli.md — Surface (`fillers`, walk flags, Output), Interactions (validation order + completeness), Error Communication
  - **Scenario references**: role-fillers.feature: "A role's fillers are listed", "A person and an agent filler are distinguished by kind", "An unknown role id fails with the API status", "A missing token fails as a not-authenticated usage error", "A role with no fillers is a clean success", "A missing role-id is a usage error", "The actor name is shown without an include flag", "A missing token issues no request", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse `classifyClientError`/`aggregateRawData`+`writeHuman`/`paging.All`/`RetryExecutor`/`reportFailure` — inline no second copy of the `errors.As` chain, render branch, or page loop. ⚠️ Declare **no** filter flags and **no** `--include` — the endpoint offers none beyond the default include + pagination (plan ADR-3); do not add a `validateInclude`-style validator (there is no closed-enum input). ⚠️ Structured output is the aggregated `{data:[…]}` over `Page[json.RawMessage]`, NOT a decode-and-re-encode of `[]Assignment` and NOT a single page's raw envelope. ⚠️ Resolve `--output` BEFORE assembly so the tripwire confirms no request on a usage error. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Pass the role id through (no regex gate); the API 404s cleanly. ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `role-fillers.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/who-to-contact-for-a-role/role-fillers.feature` in a **new** `internal/cli` godog suite (e.g. `TestRoleFillersFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `fillers` through the `fillersSeam` over a fake base `http.RoundTripper` returning canned `GET /roles/{role_id}/assignments` responses (single-page with a person + an agent, a row with focus + election, multi-page, mid-walk error, empty, unknown-id 404), plus a transport tripwire for the no-request paths (missing positional; no usable token). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection / "complete connection context with a stored token" / "no usable token" phrasings from the `roles`/`projects`/`actors` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` role-fillers scenario has an executable, passing path; `@wip` removed from them
    - The 4 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `role-fillers.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002 (the `fillers` command — all behavioral scenarios must be implementable).
  - **Plan reference**: Phase 1 (Implementation Strategy step 3); System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: role-fillers.feature: all behavioral Rule-block scenarios (the 4 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `role-fillers.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the person+agent, focus+election, multi-page, mid-walk-error, empty-list, unknown-id, and no-request (tripwire) fakes so the kind-distinction, capacity, completeness, empty, failure, and rejection scenarios genuinely exercise their paths.
