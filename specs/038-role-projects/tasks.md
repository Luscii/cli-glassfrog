# Tasks: Role Projects

**Feature**: 038-role-projects
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/governance-reads/role-projects.feature

---

## Dependency Graph

Phase 1: `internal/render` singular `project` key (1 task, no phase dependencies) [Shared]
Phase 2: The two commands (2 tasks: T002 depends on nothing new — parallel with Phase 1; T003 depends on Phases 1, 2) [US1/US3/US4 + US2]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

4 tasks total | T001 and T002 startable immediately (parallel) | Builder: pipeline

> Plan-faithful: the plan's three phases map here as render (T001), commands (T002 list / T003 single), and BDD (T004). **Unlike 034, there is no schema phase** — plan ADR-2 reuses `internal/glassfrog.Project` (grown by 014) unchanged and instantiates the landed generics `Page[Project]` (016) and `Document[Project]` (034). The **list** command (T002) reuses the landed `projects` render key (014) and the shared `validateStatus` (013/014), so it depends on **nothing new** and runs parallel with T001; the **single** command (T003) needs the new singular `project` render key (T001) and the shared command scaffold (T002). Story labels follow the spec's four user scenarios — US1 (list a role's projects), US2 (read one project's detail), US3 (narrow by status/search/tag), US4 (trust the list is whole). T002 serves US1/US3/US4 → `[Shared]`; T003 is cleanly `[US2]`.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/014/015/016/017/018/020/025/033/034 are Complete with their packages shipped — `internal/glassfrog.Project` (the full model, grown by 014), `Page[T]`/`Pagination` + `paging.All`, the generic `Document[T]` (034), `RetryExecutor`, `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the shared `validateStatus` + `supportedActionStatuses` set (`internal/cli/status.go`, 013/014), the persistent `--base-url`/`--output`/`-o`, and the walked-list render machinery (`aggregateRawData` + `renderFn`, the roles/domains/policies pattern) over `internal/output`/`internal/render` (018/019/020) all ship from main. The 038 base cuts from current main with no sequencing caveat. **Existing main state 038 builds on**: the `projects` render key and `projects.{full,compact}.tmpl` already exist (014); there is **no** singular `project` render key (T001 adds it) and **no** `projects`/`project` command (T002/T003 add them). 038 adds **no** new `Outcome` category, `ExitCode` case, validator, generic type, or root flag.
>
> **The role-addressable project surface, distinct from My Projects (014).** `me projects` reads `GET /me/projects` (token-scoped); 038 reads `GET /roles/{role_id}/projects` + `GET /projects/{id}`. Completes the per-role-read trio after #33 (Role Domains) and #34 (Role Policies) — see plan ADR-1.

---

## Branching Guidance

**Pipeline mode**: `spec/038-role-projects/base` → `spec/038-role-projects/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling reads. 038 touches `internal/cli` (new `projects.go`) and `internal/render` (one new singular key); it makes **no** change to `internal/glassfrog` (model reused as-is — plan ADR-2), so it carries no cross-package contract change to coordinate (contrast 034's `Document[T]` generalization, now landed).

---

## Phase 1: `internal/render` singular `project` key [Shared]

- [ ] **T001** [Shared] [P] Add the singular `project` render key + `ProjectView` + templates — golden + registry-guard tests; `internal/render` (new `project.{full,compact}.tmpl`, `ProjectView`, `ResourceProject`) + tests
  - **Scope**: In `internal/render`, add **one new** render key `project` (single, data `glassfrog.Project` via a `ProjectView`, mirroring the landed `PolicyView{Policy}`). Add the `ResourceProject` constant to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates: `project.full` — the full single-project detail (status, description, owning role with the `(individual initiative — no role)` marker, parent with the `(top-level — no parent)` marker, the `sub-projects`/`actions` presence flags, tags with `(none)`, `created_at`/`updated_at` with `(unknown)`, link/note with `(none)`) rendering the free-text `description`/`note` **verbatim — never truncated or reflowed** (CONSTITUTION VI); `project.compact` — `<proj_…>  [<status>]  <description|—>` (detail omitted). Use 019's absence-guard discipline (`{{if eq (trimSpace …) ""}}…`). The **list** `projects` key (014) is reused unchanged — touch no existing key or template. Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `project` key renders both `full` and `compact`; `full` shows status, description, owning role, parent, presence flags, tags, timestamps, link, note
    - A project with null `role_id` renders `(individual initiative — no role)`; null `parent_project_id`/`link`/`note`/empty `description` render their explicit-absence markers, never `<no value>` or an invented value
    - A long free-text `note`/`description` is rendered verbatim (neither truncated nor reflowed)
    - The registry-exhaustiveness guard passes with the new `project` key carrying both formats; golden tests pin each template
    - The existing `projects` list key/templates are unchanged; `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: None (the view struct references the landed `glassfrog.Project`).
  - **Plan reference**: Phase 1 (Render); ADR-4 (reuse `projects` list key, add singular `project` key)
  - **Interface references**: interface-cli.md — Output (single `project`, `full`/`compact` shapes), Consistency Notes (render keys)
  - **Scenario references**: role-projects.feature: "A single project is read with full detail"
  - **Risk**: ⚠️ Add the singular key only — reuse the landed `projects` list key, touch no existing template. ⚠️ Never truncate/reflow `note`/`description` (CONSTITUTION VI). ⚠️ Explicit-absence guards for every nullable field; never invent a value. ⚠️ Add `ResourceProject` to the guarded set so exhaustiveness still holds.

## Phase 2: The two commands [US1/US3/US4 + US2]

- [ ] **T002** [Shared] [P] Add the `projects <role-id>` list command + `projectsSeam` + `--query`/`--status`/`--tag`/`--first-page`/`--per-page` + completeness + wiring — RED-first unit tests for every branch; new `internal/cli/projects.go` + tests + `Assemble()` wiring
  - **Scope**: New `internal/cli/projects.go`. `newProjectsCommand(seam projectsSeam) *cobra.Command`: a guard-registered leaf (`Use:"projects <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), declares `--query`/`-q`, `--status`, `--tag`, `--first-page`, `--per-page`; reads the persistent `--base-url`/`--output`; delegates to pure `runProjectsList`. Define the shared `projectsSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client` executor; prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Executor`). `runProjectsList(cfg) (Outcome, error)`: **validate `--status` via the shared `validateStatus`** (the one closed-enum input — plan ADR-3) and resolve `--output` (020) **before** assembly, so a bad status / bad format is a fail-fast `UsageError(2)` with no request; build the `q`/`status`/`tag` query parameters each only when its flag is `Changed()` and non-empty (the combinable filters; 026 `--depth` optional-flag discipline); default path walks `GET /roles/{role_id}/projects` to completion using the landed roles/domains/policies two-track list pattern (NOT `renderResult[T]`, which is the single-page `/me*` dispatch): for `json`/`yaml`, walk `paging.All[json.RawMessage]` then `aggregateRawData` into the `{data:[…]}` document (per-record raw bytes preserved, per-page `meta` dropped); for `full`/`compact`, walk `paging.All[Project]` then `renderFn(ResourceProjects, …)` over a `ProjectsView`. `--first-page` does one `Execute` into the corresponding `Page[json.RawMessage]`/`Page[Project]` and writes a "more projects exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The role id is passed through (`url.PathEscape`, no local validation — plan ADR-3). Wire `MustRegister(root, newProjectsCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog projects role_0123` walks every page (`paging.All`) and prints the projection of all the role's projects; exits 0
    - A role with no projects prints the `projects` empty line (`no projects`) and exits 0
    - `--status current` sends `status=current`; `--query x` sends `q=x`; `--tag t` sends `tag=t`; the three combine when all present; an omitted/empty filter sends nothing
    - `--status active` (unsupported) is a `UsageError(2)` naming the value + supported set, **no request sent** (transport tripwire)
    - `--first-page` against a multi-page list prints one page, writes a "more projects exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the aggregated `{data:[…]}` document (per-record raw bytes, per-page `meta` dropped — never a single page's envelope), `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/validator/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: None new — reuses the landed `projects` render key (014), the walked-list machinery `Page[json.RawMessage]`/`Page[Project]` + `aggregateRawData` + `renderFn` (016/033/034), and `validateStatus` (013/014). Runs parallel with T001.
  - **Plan reference**: Phase 2 (Commands); ADR-1 (sibling command, `ExactArgs(1)`, list-only flags), ADR-3 (`--status` validated locally; `--query`/`--tag`/id passed through); Cross-cutting (completeness reuses 025 ADR-3)
  - **Interface references**: interface-cli.md — `projects` Surface, list flags, Output, Interactions (validation order + completeness), Error Communication
  - **Scenario references**: role-projects.feature: "A role's projects are listed", "A role with no projects is a clean success", "A missing token fails as a not-authenticated usage error", "The project list is narrowed by a supported status", "An unsupported status is rejected as a usage error", "A rejected status issues no request", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse `validateStatus`/`classifyClientError`/`aggregateRawData`+`renderFn`/`paging.All`/`RetryExecutor` — inline no second copy of the status set, `errors.As` chain, render branch, or page loop. ⚠️ Structured output is the aggregated `{data:[…]}` over `Page[json.RawMessage]` (the roles/domains/policies pattern), NOT a decode-and-re-encode of `[]Project` and NOT a single page's raw envelope. ⚠️ Each filter sent only when `Changed()` and non-empty; only `--status` is validated (`--query`/`--tag` are free text). ⚠️ Validate `--status` and resolve `--output` BEFORE assembly so the tripwire confirms no request on rejection. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

- [ ] **T003** [US2] Add the `project <proj-id>` single-read command — `runProjectGet` + `Document[Project]` decode + the `project` render dispatch — RED-first unit tests; extends `internal/cli/projects.go` + `Assemble()` wiring
  - **Scope**: In `internal/cli/projects.go`, add `newProjectCommand(seam projectsSeam) *cobra.Command`: a guard-registered leaf (`Use:"project <proj-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`), declares **no** list flags (so `--query`/`--status`/`--tag`/`--first-page`/`--per-page` are cobra unknown-flag usage errors — the structural list-only guard, plan ADR-1); reads the persistent `--base-url`/`--output`; delegates to pure `runProjectGet`. `runProjectGet(cfg, id) (Outcome, error)`: resolve `--output` before assembly; with the id `url.PathEscape`-d and passed through unvalidated (plan ADR-3) so an unknown id surfaces the API's non-2xx via the shared classifier. For `json`/`yaml`, `Execute` into a raw `json.RawMessage` and emit the `{data: Project}` body verbatim via `output.RenderSuccess` (018); for `full`/`compact`, `Execute` into `Document[Project]` and render via `renderFn(ResourceProject, ProjectView{doc.Data})` (the single-read shape, mirroring `runPolicyGet` — NOT `renderResult[T]`). Wire `MustRegister(root, newProjectCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog project proj_0123` reads the project and prints its status, description, and owning role; exits 0
    - An unknown `proj_ffff` surfaces the API status (APIError(3)/PermissionError(4)) — the id is not validated locally
    - A list filter on `project` (`--status`/`--query`/`--tag`/`--first-page`/`--per-page`) is a `UsageError(2)` via cobra's unknown-flag handling, **no request sent** (transport tripwire across the flags)
    - `-o json`/`yaml` emit the raw single-project payload; `full`/`compact` render the `project` human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`project` render key), T002 (the shared `projectsSeam` + file scaffold)
  - **Plan reference**: Phase 2 (Commands); ADR-1 (singular command, list-only-ness structural), ADR-3 (id pass-through), ADR-2 (`Document[Project]`)
  - **Interface references**: interface-cli.md — `project` Surface, Output (single), Error Communication
  - **Scenario references**: role-projects.feature: "A single project is read with full detail", "An unknown project id fails with the API status", "A list filter is rejected on the single read"
  - **Risk**: ⚠️ Declare no list flags on `project` — that's how list-only-ness is enforced (no hand-rolled cross-combo guard; plan ADR-1). ⚠️ Pass the id through (no regex gate); the API 404s cleanly (plan ADR-3). ⚠️ Render the detail faithfully (no truncation). ⚠️ Never read `ctx.Cred.Token`.

## Phase 3: Executable acceptance [Shared]

- [ ] **T004** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `role-projects.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/governance-reads/role-projects.feature` in a **new** `internal/cli` godog suite (e.g. `TestRoleProjectsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `projects`/`project` through the shared seam over a fake base `http.RoundTripper` returning canned `GET /roles/{role_id}/projects` (single-page, multi-page, mid-walk error, empty, status-filtered) and `GET /projects/{id}` (found, 404) responses, plus a transport tripwire for the no-request paths (unsupported `--status`; list flag on `project`). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 3 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection phrasings from the `me*`/`roles`/`policies` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` role-projects scenario has an executable, passing path; `@wip` removed from them
    - The 3 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `role-projects.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002 (`projects` list), T003 (`project` single) — all behavioral scenarios must be implementable
  - **Plan reference**: Phase 3 (BDD); System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: role-projects.feature: all behavioral Rule-block scenarios (the 3 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `role-projects.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mid-walk-error, multi-page, empty-list, status-filtered, and no-request (tripwire) fakes so the completeness, empty, filter, and rejection scenarios genuinely exercise their paths.
