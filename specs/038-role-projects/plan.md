# Plan: Role Projects

**Feature**: 038-role-projects
**Role**: Shaper
**Inputs**: spec.md (038), PROJECT.md, DECISIONS.md, LEARNINGS.md (background), DEPRECATION.md

---

## System Architecture

Role Projects adds the **addressable project read surface** to the Governance Reads slice. Architecturally it is a near-exact sibling of Role Policies (034) and Role Domains (033): two thin cobra commands in `internal/cli` that build a request, hand it to the proven read chain, and render the result. It introduces **no new package** and **no new transport, pagination, error, or output machinery** — every seam it needs is landed, and the per-role-read surface precedent was set by 034 ADR-1, which named this feature as a verbatim follower (`projects <role-id>`/`project <proj-id>`).

Two commands, each `ExactArgs(1)`, guard-registered (001) and explicitly wired in `main`:

- **`glassfrog projects <role-id>`** → `GET /roles/{role_id}/projects` (`listRoleProjects`) — a **paginated list** of the projects owned by a role. Walks to completion through `paging.All[Project]` (016) by default, with a `--first-page` opt-out and three optional, combinable server-side filters: `--query`/`-q` (sent as `q`), `--status` (validated locally, sent as `status`), and `--tag` (sent as `tag`).
- **`glassfrog project <proj-id>`** → `GET /projects/{id}` (`getProject`) — a **single project** with full detail, decoded from a `{data: Project}` document.

Data flow per invocation (identical to 033/034): the command validates its inputs locally where the API would otherwise silently mislead (only `--status`, the one closed-enum input), resolves the connection context once (`AssembleFromOS`), builds the `*apiclient.Client` (010), issues the request (list → `paging.All`; single → one `Execute`), and renders by the resolved `--output` format (020): the **single** read emits the raw `{data: Project}` bytes (018) or the human `project` template (019); the **list** walks pages — for `json`/`yaml` aggregating each project's raw bytes into a `{data:[…]}` document via `aggregateRawData` (per-page `meta` dropped), for `full`/`compact` rendering the `projects` projection (019) — the landed roles/domains/policies walked-list pattern (`renderResult[T]` is the single-page `/me*` dispatch, not used here). Typed client errors route through the one shared `classifyClientError` chain (011/015) — no new `Outcome` category, no new exit code.

The only genuinely new artifacts are: the **`projects`/`project` commands** and a **singular `project` render path** (`ProjectView`, the `ResourceProject` registry entry, and the `project.full`/`project.compact` templates). The `glassfrog.Project` model and the `projects` *list* render path are reused as-is from My Projects (014); the single-object `Document[T]` envelope is reused as-is from Role Policies (034).

---

## Architecture Decisions

### ADR-1: Expose two sibling commands `projects <role-id>` and `project <proj-id>` (conformance to 034 ADR-1)

**Context**: 034 ADR-1 established the per-role-read surface — two sibling top-level commands, plural `<noun>s <role-id>` for the role-scoped list and singular `<noun> <id>` for the standalone read, each `ExactArgs(1)` — and explicitly set it as the precedent "#33 (Role Domains) and #38 (Role Projects) can follow verbatim — `projects <role-id>`/`project <proj-id>`." 033 already followed it. Role Projects has two reads keyed on **different id kinds**: a `role_` id selects a per-role list, a `proj_` id selects one project. The spec confirmed the surface with the developer.

**Options considered**:
1. **One command, optional positional** (`projects [id]`). Rejected: the two reads take different id kinds and hit different endpoints; an optional positional cannot carry that distinction, and `me projects` (014) already occupies the self-service reading of "projects" — a top-level `projects <role-id>` keeps the role-addressable surface cleanly separate.
2. **A `role <role-id> projects` group.** Rejected for the same reason 034 rejected it: it forces the sibling per-role reads to land a shared group in lockstep; 033/034 deliberately did not create it.
3. **Two sibling top-level commands** — plural `projects <role-id>` + singular `project <proj-id>`, both `ExactArgs(1)`. Chosen — silent conformance to 034 ADR-1.

**Decision**: Option 3. `projects` and `project` are two guard-registered, explicitly-wired sibling commands. As in 034, the **list-only flags (`--query`, `--status`, `--tag`, `--first-page`, `--per-page`) are registered only on `projects`** — passing any of them to `project` is rejected by cobra's own unknown-flag handling as a `UsageError(2)` before assembly, so the spec's "filters on the single read are a usage error" needs no hand-rolled guard.

**Consequences**: No `role` group is created. `projects` (role-scoped, top-level) coexists with `me projects` (token-scoped, 014) without collision — distinct command paths, the same `glassfrog.Project` projection. The `--base-url` and `--output`/`-o` persistent root flags (011/020) are inherited by both.

### ADR-2: Reuse `glassfrog.Project` and `Document[T]` as-is — no schema growth

**Context**: Unlike 034 (which had to grow the minimal `Policy` type), the `glassfrog.Project` model grown in 014 (`internal/glassfrog/projects.go`) already carries every field both endpoints return — `id`, `status`, `description`, `role_id`, `tags`, `has_sub_projects`, `has_actions`, plus the detail fields `individual_initiative`, `parent_project_id`, `created_at`, `updated_at`, `link`, `note` — and decodes tolerantly of extra fields. `getProject` returns "full detail" of the same `Project` schema. The generic single-object envelope `Document[T]` was landed by 034 (`internal/glassfrog/document.go`) and is already used as `Document[Policy]`, `Document[RoleDetail]`, `Document[Domain]`.

**Options considered**:
1. **Add a fuller single-read type beside `Project`.** Rejected: violates 011 ADR-1 (one shared schema type, grown not duplicated) and forks the projection; the model is already complete.
2. **Reuse `Project` unchanged; the list walks `Page[json.RawMessage]` (structured) / `Page[Project]` (human) per the walked-list pattern, and the single read decodes `Document[Project]`.** Chosen.

**Decision**: Option 2. No model change. The list walk reads `glassfrog.Page[json.RawMessage]` for structured output (aggregated via `aggregateRawData`, the roles/domains/policies pattern) and `glassfrog.Page[Project]` for human rendering; the single read decodes `glassfrog.Document[Project]`. All already exist generically — this feature only instantiates them.

**Consequences**: Smaller than 034 — no schema phase at all. The single read surfaces detail fields (`created_at`/`updated_at`/`parent_project_id`/`link`/`note`) the existing `projects` list projection does not render; those become render targets for the new singular `project` template (ADR-4), with explicit-absence guards for the nullable ones (`role_id` null for individual-initiative projects; `description`/`parent_project_id`/`link`/`note` nullable).

### ADR-3: Validate the one closed-enum input (`--status`) locally; pass `--query`, `--tag`, and the id through

**Context**: 025 ADR-4 set the input-handling principle — *validate closed-enum inputs locally* (where a wrong value makes the API silently return wrong results), but *pass free identifiers and free text through* (where the API reports cleanly). This is where Role Projects diverges from 034 ADR-3 ("validate nothing"): `listRoleProjects` offers a **closed `status` enum** (`archived`, `cancelled`, `completed`, `current`, `scheduled`, `someday`, `waiting`) that 034's policy endpoint lacked. `--query` and `--tag` are free text; the `role_`/`proj_` id is a free identifier the endpoints answer with a clean `400`/`404`.

**Options considered**:
1. **Pass everything through, including `--status`.** Rejected: an unsupported status is silently ignored or mis-handled by the API, returning wrong results indistinguishable from "no matches" — exactly the closed-enum hazard 025 ADR-4 / 013 / 014 validate against locally.
2. **Validate `--status` locally via the shared `validateStatus`; pass `--query`, `--tag`, and the id through.** Chosen.

**Decision**: Option 2 — silent conformance to 025 ADR-4 and reuse of the landed validator. `--status` is checked by the existing `validateStatus` (`internal/cli/status.go`, the `supportedActionStatuses` set shared by 013/014) **before any context assembly or request**; an unsupported value is a `UsageError(2)` naming the value and the supported set, with no request issued. `--query` → `q` and `--tag` → `tag` are sent verbatim (no enum check); the positional id is sent into the path unvalidated and a bad id surfaces as the API's clean non-2xx, classified by the shared chain. Each filter is sent only when its flag is `Changed()` and non-empty (the 026/034 optional-flag discipline), so the absent and empty-value cases both mean "no constraint".

**Consequences**: No new validator — `validateStatus` is reused verbatim, extending its consumers from `me actions`/`me projects` to `projects <role-id>`. The status set stays single-sourced in `status.go` (adding a status remains a one-line change tracking the spec enum). The three filters combine (each its own query parameter); the API applies them together.

### ADR-4: Add a singular `project` render path; reuse the `projects` list path and structured output unchanged

**Context**: 019 renders human output from `//go:embed` templates per resource key in `internal/render`, dispatched by 020, with explicit-absence guards and golden tests; the registry-exhaustiveness guard (`builtinResources`) requires every resource to have both `full` and `compact`. The **list** path already exists: `ResourceProjects` ("projects") with `projects.full`/`projects.compact` templates, landed by My Projects (014) and rendering the shared `Project` projection. There is **no singular `project` resource** — `builtinResources` has `ResourceDomain`/`ResourcePolicy` but no `ResourceProject`, and there are no `project.full`/`project.compact` templates.

**Decision**: Add the singular `project` render path: a `ProjectView` struct in `internal/render`, the `ResourceProject` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and two embedded templates — `project.full` (the full single-project detail: status, description, owning role with the null-role marker, the `has_sub_projects`/`has_actions` presence flags, tags, and the timestamps/link/note with explicit-absence guards) and `project.compact` (id + status + description summary). The **list** read reuses the landed `projects` templates unchanged (same `Project` type, same projection). Structured `json`/`yaml` output reuses the landed machinery with no new code, but the two reads differ: the **single** read serializes the raw `{data: Project}` body verbatim (018 ADR-2, via `output.RenderSuccess`); the **list** walks each page as `json.RawMessage` and emits the aggregated `{data:[…]}` document via `aggregateRawData` — per-record raw-byte fidelity preserved, per-page `meta` dropped — the roles/domains/policies walked-list pattern, never a single page's envelope.

**Consequences**: One new resource key and two new templates, mirroring 034's `policy` addition exactly (and following the PR #10 registry-exhaustiveness shape). The `projects` list templates already handle the empty-set line (`no projects`); the incomplete-walk stderr note lives on the command, not the template (025 ADR-3 / 032). The presence flags (`has_sub_projects`/`has_actions`) are rendered as signals — the single read deliberately does not embed children (spec Non-Behavior: no `--include`).

---

## Cross-cutting Concerns

**List completeness** (silent conformance to 025 ADR-3): `projects <role-id>` defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Project]` for human; the `--first-page` opt-out does a single `Execute` into the corresponding `Page[…]`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial `Records`, writes "incomplete — <cause>" to stderr, and exits non-zero. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`.

**Error handling**: identical to every read since 011 — typed client errors route through the single shared `classifyClientError` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. The landed failure surface is 031's `Diagnose` (normalized diagnostic) rendered format-aware by 032's `reportFailure` chokepoint — structured (`json`/`yaml`) failures emit the 018 error envelope on stdout, human (`full`/`compact`) failures write the diagnostic on stderr; the partial-walk incompleteness notes stay on stderr in every format (a partial document already occupies stdout). 038 calls `reportFailure` exactly as the landed reads do. No new exit codes.

**Input validation order**: `--status` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/014 fail-fast discipline.

**Testing**: pure-unit coverage is mostly inherited (the `Project` decode and `Document[T]`/`Page[T]` generics are already tested). New tests: golden tests for the two `project` templates (incl. a null-`role_id` individual-initiative project and the nullable detail fields); a `internal/cli` godog suite over a new `features/governance-reads/role-projects.feature` driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when `project` is given a list-only flag (unknown-flag `UsageError`) and (b) no request when `--status` is unsupported.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025), plus the existing `validateStatus` set.

---

## Implementation Strategy

Single phase — one cohesive read pair with no internal dependencies, every seam below it landed. Suggested ordering for task decomposition:

1. **Render** — add the singular `project` path: `ProjectView` struct, `ResourceProject` registry entry (in `builtinResources`), and the `project.full`/`project.compact` templates; golden tests. (The `projects` list path is reused unchanged.)
2. **Commands** — `projects <role-id>` (list: `paging.All[Project]` + `--query`/`-q`, `--status` via `validateStatus`, `--tag`, `--first-page`/`--per-page`, the 025 completeness logic) and `project <proj-id>` (single: `Document[Project]`), guard-registered and wired in `main`; route through the walked-list render pattern (`aggregateRawData` for structured / `renderFn` projection for human; the single read via `output.RenderSuccess` / `renderFn`) and `classifyClientError`.
3. **BDD** — the `role-projects.feature` suite covering the spec's driving scenarios + the two structural tripwires (list-only flag on `project`; unsupported `--status`).

Phase 1 (render) and the `projects` list command are independent and can run in parallel (the list reuses the landed `projects` render key, so it needs nothing from Phase 1); only the `project` single command depends on Phase 1's new `project` render key; Phase 3 (BDD) depends on both commands. No schema phase (ADR-2: model reused as-is). (Matches the tasks.md dependency graph: T001 ∥ T002; T003 needs T001+T002; T004 needs T002+T003.)

---

## Risks

- **`projects` list template assumes the `/me*` projection shape** (low likelihood, low impact): the landed `projects` templates were authored for My Projects (014). The role-scoped list renders the same `[]Project`, so they should apply directly. Mitigation: a golden test over the role-scoped list result confirms parity; if 014's projection diverges (e.g. omits a field the role list should show), pin the difference at the interface stage rather than forking the template.
- **`status` enum drift** (low likelihood, low impact): the `status` set is single-sourced in `status.go` against the spec enum. Mitigation: it already matches `listRoleProjects`' enum verbatim; a spec change is a one-line update tracking the vendored `spec/glassfrog-api-v5.yaml`.
- **Project `note`/`description` may contain long free text** (low likelihood, low impact): the single-`project` `full` template renders them faithfully (`text/template`, no truncation — CONSTITUTION VI); `compact` summarizes. No reflow.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — exact command/flag spellings, the `ProjectView` field names, the request-descriptor shape, and the template text are the **interface** skill's concern. The names used here (`projects`/`project`, `--query`/`-q`, `--status`, `--tag`) are the developer-confirmed surface; treat them as the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units are the **tasks** skill's output; the Implementation Strategy above is the input.
- **A `role` command group** — deliberately not created (ADR-1), consistent with 033/034.
- **`--include=sub_projects,actions` and standalone sub-project/action reads** — out of scope by spec Non-Behavior (flat read); a future capability if needed.
