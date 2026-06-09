# Tasks: Role Reads

**Feature**: 025-role-reads
**Concretization**: Full context (plan + spec + interface-cli + interface-spec + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/governance-reads/role-reads.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` schema growth (1 task, no phase dependencies) [Shared]
Phase 2: The `roles` list read (2 tasks, depends on Phase 1) [Shared]
Phase 3: The single role read + acceptance (2 tasks, depends on Phases 1, 2) [Shared]

5 tasks total | T001 startable immediately | Builder: pipeline

> Every task is `[Shared]`: `roles` is one command serving all four user scenarios (navigate/drill US1, filter US2, embed US3, completeness US4). The plan's three phases map to: schema (T001) → list read core + refinements (T002, T003) → single read (T004) + executable acceptance (T005).
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/015/016/017/018/019/020 are Complete. So the 025 base is cut from current main with no sequencing caveat. T001 reuses `internal/glassfrog` (011/012) + `Page[T]`/`Pagination` (016); T002–T004 reuse `AssembleFromOS`/`NewClientFromOS`/`Execute` (009/010), `NewRetryExecutor` (017), `paging.All` (016), `classifyClientError`/`Outcome`/`ExitCode` (011/015), the persistent `--base-url` (011) + `--output`/`-o` (020), and `renderResult[T]` over `internal/output`/`internal/render` (018/019/020). 025 adds **no** new `Outcome` category, `ExitCode` case, validator-beyond-its-own, or root flag.
>
> **First org-wide read.** Distinct from `me roles` (012, token-scoped). Sets the schema + boundary precedent the downstream per-role reads (#33/#34/#38) and Organization Tree (#26) consume — see plan ADR-1/ADR-2.

---

## Branching Guidance

**Pipeline mode**: `spec/025-role-reads/base` → `spec/025-role-reads/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry other specs (e.g. 022 release pipeline). 025 touches `internal/glassfrog`, `internal/cli`, and `internal/render` — coordinate `internal/glassfrog.Role` growth with any concurrent read spec so only one definition of the grown fields exists.

---

## Phase 1: `internal/glassfrog` schema growth [Shared]

- [ ] **T001** [Shared] [P] Grow `Role` to the full spec shape and add `RoleDetail` + the `Assignment`/`Policy`/`Note`/`SkillSummary` leaf models and the `{data: RoleDetail}` document wrapper — RED-first decode tests; `internal/glassfrog/roles.go` (+ related model files) + tests
  - **Scope**: In `internal/glassfrog`, grow the shared `Role` with the remaining spec fields (`Type`, `ParentRoleID *string`, `HasSubroles bool`, `Flags []string`, `Fillers []Actor`, `Tags []string`) — never a second role type (011 ADR-1). Add `RoleDetail` embedding `Role` plus the optional related-resource fields `Assignments []Assignment`, `Subroles []Role`, `ParentRole *Role`, `Policies []Policy`, `Notes []Note`, `Skills []SkillSummary` (nested `Subroles`/`ParentRole` are plain `Role` — no recursion). Add minimal leaf models `Assignment`, `Policy` (incl. `Title`), `Note`, `SkillSummary` (no `Content`). Add the single-object document wrapper (`RoleDocument{ Data RoleDetail }`, or a generic `Document[T]` if preferred — first single-object read creates it). The list decodes the **existing** generic `Page[Role]` (016) — do not define a new envelope. Decoding tolerates unknown/extra fields; no transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A `GET /roles` page fixture decodes into `Page[Role]` with `Data` populated and `Meta.Pagination` read
    - A `GET /roles/{id}` fixture decodes into the document wrapper; `RoleDetail` exposes the embedded `Role` fields
    - With each `?include` value present, the matching `RoleDetail` field populates; when absent it stays nil/empty
    - A null `parent_role_id` and an empty `subroles`/`policies`/etc. decode without error; `SkillSummary` has no `Content` field
    - Unknown/extra fields are ignored; no new internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Reuses `glassfrog.Page[T]`/`Pagination` (016) and `Actor` (011).
  - **Plan reference**: Phase 1 (Schema); ADR-2 (grow `Role`, add `RoleDetail` + leaf models); System Architecture (`internal/glassfrog`)
  - **Interface references**: interface-spec.md — `internal/glassfrog` (Surface)
  - **Scenario references**: role-reads.feature: "A single role is read by id", "Requested related resources are embedded inline"
  - **Risk**: ⚠️ Grow the shared `Role`; do **not** fork a list/detail type (011 ADR-1). ⚠️ Nested `Subroles`/`ParentRole` are plain `Role`, not `RoleDetail` — avoid recursion. ⚠️ `SkillSummary` carries no `Content` (summary-only embed). ⚠️ Reuse `Page[T]`; never define a 025-local envelope.

## Phase 2: The `roles` list read [Shared]

- [ ] **T002** [Shared] Add the `roles` command scaffold + the bare org-wide list (walk to completion) + render + wiring — RED-first unit tests for every branch; `internal/cli/roles.go` + `roles_test.go`, new `roles` render templates
  - **Scope**: Add `internal/cli/roles.go`. `newRolesCommand(seam rolesSeam) *cobra.Command`: a guard-ready leaf (`Use:"roles [id]"`, `Args: cobra.MaximumNArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), reads the persistent `--base-url` (011) + `--output`/`-o` (020), delegates to pure `runRoles`. `runRoles(cfg) (Outcome, error)`: resolve `--output` (020) and run `validateRolesFlags` (id/flag-combo guard) **before** assembly; with no id → `runRolesList`. `runRolesList`: default path walks `GET /roles` to completion via `paging.All[Role]` over the seam's executor (`RetryExecutor` wrapping `NewClientFromOS`, 016/017), then dispatches the records through `renderResult("roles", format, records)` (020). On a `Result.Stop` error, map via `classifyClientError`. Register the `roles` `full`/`compact` templates in `internal/render` (019) — one block per role (name, id, purpose, domains, accountabilities) with explicit-absence markers; registry exhaustiveness guard. Wire `MustRegister(root, newRolesCommand(productionSeam{}))` once in `Assemble()`/`main`. `roles` never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog roles` on a complete context walks every page (`paging.All`) and prints the projection of all roles; exits 0
    - An empty org prints `No roles.` (human format) and exits 0
    - `*AuthError{NoCredentials}` → UsageError(2, "run `glassfrog auth login`"); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5) via the shared classifier; `*DecodeError` → RuntimeError(1); base-URL error → UsageError(2); invalid `--output` → UsageError(2)
    - `-o json`/`yaml` emit the raw payload (018), `full`/`compact` render the human projection (019); selection changes only presentation
    - The `roles` render key has both `full` and `compact`; the registry guard passes; no `Outcome`/`ExitCode`/root-flag is added
    - No output renders the token; all branches run offline over the fake seam; `go build`/`go vet` clean
  - **Dependencies**: T001 (decode target). Reuses 009/010/011/015/016/017/018/019/020 (all landed).
  - **Plan reference**: Phase 2 (List read); ADR-1 (one command, optional id), ADR-3 (default walk via `paging.All`); Cross-cutting (error handling, output, testing)
  - **Interface references**: interface-cli.md — Surface, Output, Error Communication; interface-spec.md — `newRolesCommand`/`runRoles`/`runRolesList`/`rolesSeam`, `internal/render` keys
  - **Scenario references**: role-reads.feature: "The organization's roles are listed", "A missing token fails as a not-authenticated usage error", "An unreachable API fails as network-unavailable", "An empty organization is a clean success", "A multi-page role list is walked to completion"
  - **Risk**: ⚠️ Reuse `classifyClientError`/`renderResult`/`paging.All`/`RetryExecutor` — inline no second `errors.As` chain, render branch, or page loop. ⚠️ Add no new `Outcome`/`ExitCode`/root flag. ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Secret hygiene — never read `ctx.Cred.Token`.

- [ ] **T003** [Shared] Add the list filters, the `--first-page` opt-out, the completeness signalling, and `--per-page` — RED-first unit tests; extends `internal/cli/roles.go`
  - **Scope**: Extend `roles` (list branch). Add the four filter flags — `--parent` (`parent_role_id`), `--person` (`person_id`), `--tag` (`tag`), `--has-subroles` (tri-state via `cmd.Flags().Changed`: sent only when present) — passed through as `GET /roles` query parameters; `validateRolesFlags` rejects any filter when an id is present (UsageError, no request). Add `--first-page`: a single `Execute` into `Page[Role]` (no walk), rendering the first page and writing one stderr note when `HasNextPage` (exit 0). Add the mid-walk incomplete signal: when `paging.All` returns `Result.Complete == false` with a non-nil `Stop`, render the partial `Records`, write one stderr "incomplete — <cause>" note, and exit non-zero via `classifyClientError(Stop)`. Add `--per-page` (016 `WithPageSize`) for the walk; out-of-range surfaces the API's rejection. Diagnostics/notes go to stderr only.
  - **Acceptance criteria**:
    - `glassfrog roles --parent role_aaaa` sends `parent_role_id` and prints only that parent's roles; `--person`/`--tag` send their params; `--has-subroles` is sent only when present (`--has-subroles=false` sends `false`; omitted sends nothing)
    - A filter combined with a role id is a UsageError(2) with **no request sent** (transport tripwire)
    - `--first-page` against a multi-page org prints only the first page, writes a "more roles exist" stderr note, and exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero (classified from `Stop`)
    - A no-match filter prints `No roles.` and exits 0; `--per-page` sets the walk page size and an out-of-range value surfaces the API error
    - stdout carries only the projection; all notes/diagnostics go to stderr; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T002 (the list branch it extends)
  - **Plan reference**: Phase 2 (List read); ADR-3 (default walk vs `--first-page`; opt-out→0+signal, mid-walk-fail→non-zero+partial), ADR-1 (filter/id combination guard)
  - **Interface references**: interface-cli.md — list flags, Interactions (completeness), Error Communication; interface-spec.md — `runRolesList`, `validateRolesFlags`
  - **Scenario references**: role-reads.feature: "The list is filtered by parent circle", "A filter passed with a role id is a usage error", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ `--has-subroles` is tri-state — use `Changed`, never a plain bool default (omitted ≠ false). ⚠️ Opt-out reuses the single-page-signal shape (012–014), not a `paging.WithMaxPages` cap — preserve 016's `Result.Complete == (Stop==nil)` invariant (plan ADR-3). ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI).

## Phase 3: The single role read + acceptance [Shared]

- [ ] **T004** [Shared] Add the single-role read branch — `runRoleGet` + `--include` validation + `RoleDetail` decode + the `role` render templates — RED-first unit tests; extends `internal/cli/roles.go`, new `role` templates
  - **Scope**: Extend `roles` (single branch, `len(args)==1`). `validateRolesInclude([]string)` rejects an unsupported `--include` value against `{assignments,subroles,parent_role,policies,notes,skills}` **before** any request (UsageError, transport tripwire) — the 011 `validateInclude` shape; `validateRolesFlags` rejects `--include` without an id and list filters with an id. `runRoleGet(cfg, id)`: one `Execute` into the `RoleDocument` (`{data: RoleDetail}`), with `?include=` from the validated comma-joined values; the role **id is passed through** (not locally validated — plan ADR-4), so an unknown/malformed id surfaces as the API's non-2xx via the shared classifier. Dispatch through `renderResult("role", format, doc.Data)`. Register the `role` `full`/`compact` templates in `internal/render`: the role block plus one guarded section per **requested** include resource (omitted when not requested; explicit-absence marker when requested-but-empty); skills render as summaries.
  - **Acceptance criteria**:
    - `glassfrog roles role_0123` reads the role and prints name/purpose/accountabilities/domains/fillers; exits 0
    - `--include policies,subroles` sends `include=policies,subroles` and renders those embedded inline; an unrequested include section is omitted; a requested-but-empty section renders its absence marker
    - An unsupported `--include` value is a UsageError(2) naming the value + supported set, **no request sent** (tripwire); `--include` without an id is a UsageError(2)
    - An unknown role id surfaces the API status (APIError(3)/PermissionError(4)) — the id is not validated locally
    - `-o json`/`yaml` emit the raw single-role payload; the `role` render key has both `full`/`compact` and passes the registry guard; no `Outcome`/`ExitCode` added
    - No output renders the token; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`RoleDetail`/document wrapper), T002 (command scaffold + `runRoles` dispatch + `validateRolesFlags`)
  - **Plan reference**: Phase 3 (Single read); ADR-4 (validate `--include`, pass id through), ADR-2 (`RoleDetail` related fields), ADR-1 (single branch)
  - **Interface references**: interface-cli.md — single-read flag, Output (single), Error Communication; interface-spec.md — `runRoleGet`/`validateRolesInclude`, `role` render key
  - **Scenario references**: role-reads.feature: "A single role is read by id", "Requested related resources are embedded inline", "An unsupported include value is rejected before any request", "An unknown role id fails with the API status"
  - **Risk**: ⚠️ Validate `--include` locally but pass the id through (plan ADR-4) — do not regex-gate the id. ⚠️ Render guarded include sections (omit-when-unrequested, marker-when-empty) via 019's `{{if}}`/`missingkey=error`; never invent a value. ⚠️ `?include` is comma-joined (`style:form explode:false`). ⚠️ Secret hygiene — never read `ctx.Cred.Token`.

- [ ] **T005** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `role-reads.feature`; un-@wip the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/governance-reads/role-reads.feature` in a **new** `internal/cli` godog suite (e.g. `TestRoleReadsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `roles` through its seam over a fake base `http.RoundTripper` returning canned `GET /roles`/`GET /roles/{id}` single- and multi-page responses (incl. a mid-walk error). Remove `@wip` from the 12 spec-derived + 1 architecture-informed behavioral scenarios; keep the 3 `@validation` scenarios `@wip` (held for validate). Reuse existing `internal/cli` step phrasings where assertions already exist (grep `sc.Step(` registrations first — exit-code, stderr-substring, and projection phrasings are shared with the `me*` reads); step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` role-reads scenario has an executable, passing path; `@wip` removed from them
    - The 3 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `role-reads.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (list refinements), T004 (single read) — all behavioral scenarios must be implementable
  - **Plan reference**: Phase 2 + Phase 3; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: role-reads.feature: all behavioral Rule-block scenarios (the 3 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `role-reads.feature` only (not the directory); verify it reports its own count. ⚠️ Grep existing `sc.Step(` registrations and reuse shared `me*`-read phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mid-walk-error and multi-page fakes so the completeness scenarios genuinely exercise the walk.
