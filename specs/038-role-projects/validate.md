# Validate: Role Projects

**Feature**: 038-role-projects
**Round**: 1 of 3
**Date**: 2026-06-11
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/governance-reads/role-projects.feature, PROJECT.md
**Implementation files**: `internal/cli/projects.go` (commands + run funcs), `internal/cli/app.go` (wiring), `internal/render/render.go` + `internal/render/templates/project.{full,compact}.tmpl` (singular render key); tests: `internal/cli/projects_test.go`, `internal/cli/role_projects_bdd_test.go`, `internal/render/projects_test.go`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 @validation scenarios satisfied.

All 4 tasks in tasks.md are checked (`- [x]`) — full validation. Optional test execution: `go build ./...`, `go vet ./...`, `go test ./...` all pass; the `TestRoleProjectsFeatures` godog suite reports 10 scenarios / 54 steps passing.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 driving scenarios covered; the architecture-informed mid-walk scenario also covered)

| Scenario | Status | Implementation |
|---|---|---|
| List the projects on a role | ✓ Covered | `projects.go:runProjectsListWalk` (walks `GET /roles/{id}/projects` via `paging.All[Project]`, renders `ProjectsView`) |
| Read a single project with full detail | ✓ Covered | `projects.go:runProjectGet` (`GET /projects/{id}` → `Document[Project]` → `project` render key) |
| Narrow a role's projects by status | ✓ Covered | `projects.go:projectsQuery` sends `status=`; `validateStatus` accepts a supported value |
| Project id does not exist | ✓ Covered | `runProjectGet` → `reportFailure` → shared `classifyClientError` (404 → APIError/3) |
| No usable credential | ✓ Covered | `newClient`/send path surfaces `*AuthError{NoCredentials}` → `reportFailure` → UsageError/2 |
| Role owns no projects | ✓ Covered | empty walk renders the `projects` template's `no projects` line, exit 0 |
| Unsupported status rejected before any request | ✓ Covered | `runProjectsList` step 1 `validateStatus` → UsageError/2, no request |
| Filter flag on the single read is rejected | ✓ Covered | `newProjectCommand` declares no list flags → cobra unknown-flag UsageError |
| Paginated list with first-page opt-out | ✓ Covered | `runProjectsFirstPage` (single page + `moreRoleProjectsNote`, exit 0) |
| (Mid-walk failure flagged incomplete) | ✓ Covered | `runProjectsListWalk` renders partial set + `reportIncompleteProjectsWalk` → non-zero |

Each path is exercised by a passing BDD scenario and unit tests over a fake transport.

## Acceptance Criteria

**Status**: Pass (4 of 4 tasks checked, criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — singular `project` render key | ✓ Met | `ResourceProject` in `builtinResources` (exhaustiveness guard passes), `ProjectView`, two templates; explicit-absence + verbatim-note goldens pass |
| T002 — `projects <role-id>` list | ✓ Met | walk-to-completion, three combinable filters (`q`/`status`/`tag`), `--first-page`/`--per-page`, completeness notes, classification; 19 unit tests + wiring |
| T003 — `project <proj-id>` single read | ✓ Met | `Document[Project]` decode, `project` render dispatch, no list flags; 5 unit tests + wiring |
| T004 — executable acceptance | ✓ Met | `TestRoleProjectsFeatures` scoped to its own feature file; 10 behavioral scenarios un-`@wip` and passing |

No new `Outcome`/`ExitCode`/validator/root flag introduced (confirmed by inspection — the run funcs reuse `validateStatus`, `classifyClientError`, `reportFailure`, `aggregateRawData`, `renderFn`, `paging.All`, `RetryExecutor`).

## Interface Contract Conformance

**Status**: Pass (both surfaces conformant)

| Surface | Status | Notes |
|---|---|---|
| `projects <role-id>` (`ExactArgs(1)`, `--query`/`-q`, `--status`, `--tag`, `--first-page`, `--per-page`) | ✓ Conformant | flags declared only on `projects`; `--status` validated locally, `--query`/`--tag` `Changed()`+non-empty gated |
| `project <proj-id>` (`ExactArgs(1)`, no list flags) | ✓ Conformant | list-only-ness enforced structurally via cobra unknown-flag handling |
| Single `project` `full` output layout | ✓ Conformant | template field set/order/markers match interface-cli.md § Output exactly (golden `TestRender_ProjectFull_Golden`) |
| Single `project` `compact` output | ✓ Conformant | `<proj_…>  [<status>]  <description\|—>` |
| List structured output (`{data:[…]}` aggregated, per-page meta dropped) | ✓ Conformant | `aggregateRawData` over `Page[json.RawMessage]`; test asserts no `"pagination"` envelope |
| Error-communication table (auth/transport/non-2xx/base-url/output) | ✓ Conformant | routed through shared `classifyClientError` + `reportFailure`; exit codes 2/3/4/5/6 unchanged |
| Completeness notes (exact strings) | ✓ Conformant | `moreRoleProjectsNote` / `incompleteRoleProjectsWalkNote` match interface-cli.md byte-for-byte |
| Inherited `--base-url`, `-o`/`--output` | ✓ Conformant | read via inheritance, not redeclared |

## Non-Behavior Absence

**Status**: Pass (0 violations)

| Non-behavior | Status | Evidence |
|---|---|---|
| No single command with optional positional id | ✓ Absent | two distinct commands `projects`/`project`, each `ExactArgs(1)` |
| No routing through `GET /me/projects` | ✓ Absent | only `/roles/{id}/projects` and `/projects/{id}` paths; the lone `/me/projects` hit is a comment distinguishing the note constant |
| No sub-project/action model or `--include` | ✓ Absent | no `include` token anywhere in `projects.go`; presence flags rendered as signals only |
| No standalone reads of other resources | ✓ Absent | only project endpoints touched |
| No raw-JSON default / private format flag | ✓ Absent | no private output flag; format resolved via 020 |
| No base-URL/token/header/exit-code re-implementation | ✓ Absent | reuses assemble/newClient/classifyClientError/Outcome registry |
| No project mutation | ✓ Absent | `GET`-only; no POST/PATCH/PUT/DELETE |

## @wip Lifecycle Completion

**Status**: Pass

The 10 behavioral scenarios referenced by checked task T004 have had `@wip` removed and pass under the suite's `~@wip` filter. The only 3 remaining `@wip` tags are on the `@validation` scenarios (lines 45, 82, 112), correctly held for this validate pass — not referenced by any checked task's implementation obligation.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| The two commands never collide on id kind | ✓ Satisfied | command path is fixed by command name, not id kind: `projects <id>` → `/roles/{id}/projects`, `project <id>` → `/projects/{id}`; the id is `url.PathEscape`-d and passed through unvalidated (ADR-3), so a wrong-kind id surfaces the API's `404`/`400` — never silently reads the other resource |
| An unsupported status costs no request | ✓ Satisfied | `runProjectsList` calls `validateStatus(cfg.status)` as step 1, before `resolveFormat`/`assemble`/`newClient`; returns `UsageError` with no executor built. Transport tripwire (`tr.calls == 0`) confirmed in `TestRunProjects_UnsupportedStatusIsUsageErrorNoRequest` and `TestProjectsCommand_UnsupportedStatusNoRequest` |
| Output is structured, not pre-rendered | ✓ Satisfied | no private format flag (verified by inspection); the dispatch routes all four formats from one read — structured via `aggregateRawData`/`output.RenderSuccess`, human via `renderFn` over `ProjectsView`/`ProjectView` |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 held-out @validation scenarios are independently traceable to clear code paths. Every task is checked, the build/vet/test suite is green, the singular `project` template matches the interface layout byte-for-byte, the completeness-note strings match the interface contract exactly, and no non-behavior is present in the implementation. The implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The specification loop for 038 is closed. The 3 `@validation` scenarios remain `@wip` by design — they document the held-out checks this validate pass exercised by inspection; leave them as-is unless a future runnable-validation convention is adopted across the governance-read specs.
