# Tasks: Actor Assignments

**Feature**: 050-actor-assignments
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/an-actors-governance-footprint/actor-assignments.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog.Assignment` embedded-`role` growth (1 task, no phase dependencies) [Shared]
Phase 2: `internal/render` `assignments` list key (1 task, depends on Phase 1) [Shared]
Phase 3: The `assignments` command (1 task, depends on Phase 2) [Shared]
Phase 4: Executable acceptance (1 task, depends on Phase 3) [Shared]

4 tasks total | T001 startable immediately | single-deep linear chain | Builder: pipeline

> Plan-faithful: the plan's implementation strategy has four naturally-ordered concerns — the additive model growth (T001), the render path (T002), the one command (T003), and BDD acceptance (T004). **The one difference from 047 (Role Fillers) is the schema phase**: 047 reused `glassfrog.Assignment` as-is because the role-end default include (`actor`) was already on the model; the actor-end default include (`role`) is **not**, so plan ADR-2 grows `Assignment` with an embedded `Role` before the render key projects it (T002 depends on T001; the command T003 references the new render key, so it depends on T002 — no parallelism). Every task serves all three of the spec's user scenarios (US1 list the roles an actor fills, US2 show focus/election, US3 trust completeness) at once, so all carry `[Shared]`.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/015/016/017/018/020/025/031/032/035 are Complete with their packages shipped — `internal/glassfrog.Assignment` (the model, grown by 025, carrying `id`/`actor_id`/`role_id`/`focus`/`elected_until`/embedded `actor`), `Page[T]`/`Pagination` + `paging.All`, `RetryExecutor`, `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the `reportFailure` format-aware failure chokepoint (032), the persistent `--base-url`/`--output`/`-o` (011/020) + `-o <template-ref>` (035), and the walked-list render machinery (`aggregateRawData` + `renderFn`/`writeHuman`, the roles/domains/policies/projects/actors pattern) over `internal/output`/`internal/render` (018/019/020) all ship from main. The 050 base cuts from current main with no sequencing caveat. **Existing main state 050 builds on**: `glassfrog.Assignment` carries the embedded `actor` but **no** embedded `role` (T001 adds it), there is **no** `assignments` render key (T002 adds it), and **no** `assignments` command (T003 adds it). 050 adds **no** new `Outcome` category, `ExitCode` case, validator, generic type, or root flag.
>
> **Not a dependency on 047 or 048.** Role Fillers (047, role-end `fillers <role-id>`) and Actor Directory (048, `actors`) are siblings in the Actor Reads world, but 050 does **not** depend on either landing first. The embedded-`role` growth (T001) is additive and forward-compatible: 025's `?include=assignments` embed and 047's `fillers` projection read only the `actor` block, so the new `role` field decodes unused there — whichever of 047/050 lands first creates no conflict, and a model test pins that the role-end / 025 bodies leave `role` zero-valued. The `assignments` render key is independent of 047's `fillers` and 048's `actors` keys.

---

## Branching Guidance

**Pipeline mode**: `spec/050-actor-assignments/base` → `spec/050-actor-assignments/task-1`, `…/task-2`, `…/task-3`, `…/task-4` (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling reads (047 role-fillers, 048 actor-directory, 049 actor-read). 050 touches `internal/glassfrog` (an **additive** embedded `role` field on the shared `Assignment` — coordinate only if 047 lands a conflicting edit to the same struct; the growth is additive so a merge is mechanical), `internal/render` (one new list key), and `internal/cli` (new `assignments.go`). The only cross-package contract change is the additive model field.

---

## Phase 1: `internal/glassfrog.Assignment` embedded-`role` growth [Shared]

- [x] **T001** [Shared] Grow `glassfrog.Assignment` with an embedded `role` object — additive, forward-compatible decode tests — 2 decode tests (actor-end populated + role-end/025 zero-valued); no scenarios at this layer
  - **Scope**: In `internal/glassfrog/roles.go`, add an embedded `Role` struct field to `Assignment` tagged `json:"role"`, mirroring the existing embedded `actor` block — fields `id`, `type`, `name`, and the nullable `purpose`/`parent_role_id` modeled as plain strings (the existing nullable `focus`/`elected_until` pattern). This is the actor-end default-include shape (`{id, type, name, purpose, parent_role_id}`) `listActorAssignments` returns. Growth is **additive** — change no existing field, tag, or the embedded `actor` block (plan ADR-2; 011 ADR-1 / 025 ADR-2 grow-not-duplicate, **not** a new `AssignmentDetail` type). Schema only — no transport, cobra, or exit codes; the package stays a leaf imported by `cli`/`apiclient` without a cycle.
  - **Acceptance criteria**:
    - A `GET /actors/{actor_id}/assignments` body with `?include=role` decodes into `Assignment` with the embedded `role` (`id`, `type`, `name`, `purpose`, `parent_role_id`) populated
    - A role-end (`/roles/{id}/assignments`, `?include=actor`) body and a 025 `?include=assignments` body decode with the new `role` field left zero-valued and **no error** (forward-compatible — the 012→025 pattern)
    - A `role` with absent `purpose`/`parent_role_id` decodes them as empty strings (the existing nullable-as-empty-string convention)
    - No existing field/tag is changed; the embedded `actor` block is untouched; `go build`/`go vet` clean
  - **Dependencies**: None.
  - **Plan reference**: Implementation Strategy step 1; ADR-2 (grow `Assignment` with embedded `role`)
  - **Interface references**: interface-cli.md — Output (structured object carries embedded `role`), Consistency Notes (additive schema growth)
  - **Scenario references**: actor-assignments.feature: "The filled role name is shown without an include flag" (the role data the growth carries)
  - **Risk**: ⚠️ Additive only — do not touch the embedded `actor` block, existing fields, or tags. ⚠️ Model the nullable `purpose`/`parent_role_id` as plain strings (no `*string`), matching the landed nullable-field convention. ⚠️ Add a decode test asserting the role-end / 025 paths leave `role` zero-valued (forward-compatibility is the risk this task exists to pin).

## Phase 2: `internal/render` `assignments` list key [Shared]

- [x] **T002** [Shared] Add the `assignments` list render key + `AssignmentsView` + templates — golden + registry-guard tests — 7 golden/marker tests; ResourceAssignments added to builtinResources (exhaustiveness guard passes)
  - **Scope**: In `internal/render`, add **one new** list render key `assignments` (data `[]glassfrog.Assignment` via an `AssignmentsView`, mirroring the landed list views `ProjectsView`/`ActorsView`). Add the `ResourceAssignments` constant (`"assignments"`) to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates rendering, per assignment, the **filled role first** (the "which roles does this actor fill" answer) plus the assignment's governance context: `assignments.full` — one block per assignment: the role id (`role_…`), then `Role` (name), `Purpose` (with `(none)` when absent), `Parent role` (with `(top-level)` when absent), `Focus` (with `(none)` when absent), `Elected until` (with `(not an elected seat)` when absent); `assignments.compact` — one line per assignment: `<role_…>  <role name>  — focus: <focus|—>; elected until: <date|—>`. Use 019's absence-guard discipline (`{{if eq (trimSpace …) ""}}…`) for the nullable `purpose`/`parent_role_id`/`focus`/`elected_until`, and render `focus`/`purpose` verbatim (user text — never truncated/reflowed, CONSTITUTION VI). The assignment id (`asgn_…`) and `actor_id` are **not** rendered in the human projection (not spec row fields) — they remain in the structured output. Add an `assignments` empty line (`no assignments`). Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `assignments` key renders both `full` and `compact`; each row leads with the role id (`role_…`) and shows the role name, focus, and election expiry
    - An assignment with no `focus` renders `(none)`; a non-elected assignment renders `(not an elected seat)` for `elected_until`; a role with no `purpose` renders `(none)` and a top-level role renders `(top-level)` for the parent — never `<no value>` or an invented value
    - The empty-list render is the `no assignments` line
    - The registry-exhaustiveness guard passes with the new `assignments` key carrying both formats; golden tests pin each template (present and absent focus/election; present and absent purpose/parent)
    - No existing render key/template is touched; `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: T001 (the `AssignmentsView` projects the embedded `role` the growth adds).
  - **Plan reference**: Implementation Strategy step 2; ADR-2 (add `assignments` render key)
  - **Interface references**: interface-cli.md — Output (`full`/`compact` row shapes, explicit-absence markers), Consistency Notes (render key)
  - **Scenario references**: actor-assignments.feature: "An assignment shows its focus and election expiry", "The filled role name is shown without an include flag"
  - **Risk**: ⚠️ Add the `assignments` list key only — touch no existing key/template. ⚠️ Explicit-absence guards for all four nullable fields (`focus`, `elected_until`, role `purpose`, role `parent_role_id`); never invent a value. ⚠️ Render `focus`/`purpose` verbatim (no truncation/reflow, CONSTITUTION VI). ⚠️ Add `ResourceAssignments` to the guarded set so exhaustiveness still holds.

## Phase 3: The `assignments` command [Shared]

- [x] **T003** [Shared] Add the `assignments <actor-id>` command + `assignmentsSeam` + completeness + wiring — RED-first unit tests for every branch — assignments.go (mirrors fillers.go), wired in Assemble(); 17 unit tests over walk/first-page/per-page/mid-walk/classification/bad-output/structured/ExactArgs/unknown-flag branches
  - **Scope**: New `internal/cli/assignments.go`. `newAssignmentsCommand(seam assignmentsSeam) *cobra.Command`: a guard-registered leaf (`Use:"assignments <actor-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares **no filter flags and no `--include`** (plan ADR-3) — only the shared `--first-page`/`--per-page` walk flags; reads the persistent `--base-url`/`--output`; delegates to a pure `runAssignmentsList`. Define the `assignmentsSeam` of the same shape as `fillersSeam`/`projectsSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client` executor, sleep, resolve selection, read template source; prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Executor`). `runAssignmentsList(cfg) (Outcome, error)`: resolve `--output` (020) **before** assembly (the only pre-assembly check — no closed-enum input to validate, plan ADR-3); default path walks `GET /actors/{actor_id}/assignments` to completion using the landed two-track walked-list pattern (NOT `renderResult[T]`): for `json`/`yaml`, walk `paging.All[json.RawMessage]` then `aggregateRawData` into the `{data:[…]}` document (per-record raw bytes preserved, per-page `meta` dropped); for `full`/`compact`, walk `paging.All[glassfrog.Assignment]` then `writeHuman(…, render.ResourceAssignments, …)` over an `AssignmentsView`. `--first-page` does one `Execute` into the corresponding `Page[json.RawMessage]`/`Page[Assignment]` and writes a "more assignments exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The actor id is passed through (`url.PathEscape`, no local validation — plan ADR-3) so an unknown id surfaces the API's clean `404`. Failures render format-aware through the landed `reportFailure` (032). Wire `MustRegister(root, newAssignmentsCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog assignments per_0123` walks every page (`paging.All`) and prints the projection of all the actor's assignments; exits 0
    - An actor with no assignments prints the `assignments` empty line (`no assignments`) and exits 0
    - The request carries the endpoint's default `role` include so each row names its filled role; the command declares no `--include` and no filter flag (an unknown flag is a cobra `UsageError(2)`)
    - A missing or extra positional is a cobra `ExactArgs(1)` `UsageError(2)`, **no request sent** (transport tripwire)
    - `--first-page` against a multi-page list prints one page, writes a "more assignments exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*AuthError{CredentialError}` → RuntimeError(1); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); unknown actor id → APIError(3) (typically 404); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the aggregated `{data:[…]}` document (per-record raw bytes, per-page `meta` dropped — never a single page's envelope), `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/validator/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T002 (the `assignments` render key + `AssignmentsView` the command renders through).
  - **Plan reference**: Implementation Strategy step 3; ADR-1 (single command, `ExactArgs(1)`, no plural/singular pair), ADR-3 (no filters/no `--include`, validate nothing locally, id passed through); Cross-cutting (completeness reuses 025 ADR-3, failure via 032)
  - **Interface references**: interface-cli.md — Surface (`assignments`, walk flags, Output), Interactions (validation order + completeness), Error Communication
  - **Scenario references**: actor-assignments.feature: "An actor's assignments are listed", "A person and an agent are read from the same endpoint", "An unknown actor id fails with the API status", "A missing token fails as a not-authenticated usage error", "An actor with no assignments is a clean success", "A missing actor-id is a usage error", "The filled role name is shown without an include flag", "A missing token issues no request", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse `classifyClientError`/`aggregateRawData`+`writeHuman`/`paging.All`/`RetryExecutor`/`reportFailure` — inline no second copy of the `errors.As` chain, render branch, or page loop. ⚠️ Declare **no** filter flags and **no** `--include` — the endpoint offers none beyond the default include + pagination (plan ADR-3); do not add a `validateInclude`-style validator (there is no closed-enum input). ⚠️ Structured output is the aggregated `{data:[…]}` over `Page[json.RawMessage]`, NOT a decode-and-re-encode of `[]Assignment` and NOT a single page's raw envelope. ⚠️ Resolve `--output` BEFORE assembly so the tripwire confirms no request on a usage error. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Pass the actor id through (no regex gate); the API 404s cleanly. ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

## Phase 4: Executable acceptance [Shared]

- [x] **T004** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `actor-assignments.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held — TestActorAssignmentsFeatures (Paths names only this feature file): 9 behavioral scenarios pass (49 steps), 4 @validation kept @wip
  - **Scope**: Add godog step definitions for `features/an-actors-governance-footprint/actor-assignments.feature` in a **new** `internal/cli` godog suite (e.g. `TestActorAssignmentsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `assignments` through the `assignmentsSeam` over a fake base `http.RoundTripper` returning canned `GET /actors/{actor_id}/assignments` responses (single-page for a person and for an agent, a row with focus + election, multi-page, mid-walk error, empty, unknown-id 404), plus a transport tripwire for the no-request paths (missing positional; no usable token). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection / "complete connection context with a stored token" / "no usable token" phrasings from the `roles`/`projects`/`actors` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` actor-assignments scenario has an executable, passing path; `@wip` removed from them
    - The 4 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `actor-assignments.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (the `assignments` command — all behavioral scenarios must be implementable).
  - **Plan reference**: Implementation Strategy step 4; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: actor-assignments.feature: all behavioral Rule-block scenarios (the 4 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `actor-assignments.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the person, agent, focus+election, multi-page, mid-walk-error, empty-list, unknown-id, and no-request (tripwire) fakes so the person/agent, capacity, completeness, empty, failure, and rejection scenarios genuinely exercise their paths.
