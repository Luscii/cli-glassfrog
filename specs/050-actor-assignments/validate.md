# Validate: Actor Assignments

**Feature**: 050-actor-assignments
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (4 of 4 tasks complete), interface-cli.md, features/an-actors-governance-footprint/actor-assignments.feature, PROJECT.md
**Implementation files**: 4 changed — `internal/glassfrog/roles.go` (embedded `role` block), `internal/render/render.go` + `internal/render/templates/assignments.{full,compact}.tmpl` (the `assignments` render key), `internal/cli/assignments.go` (the command), `internal/cli/app.go` (wiring); plus `internal/cli/actor_assignments_bdd_test.go` (godog suite)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 4 of 4 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 spec scenarios covered; +1 plan-derived mid-walk scenario covered)

Every driving scenario referenced by checked tasks T003/T004 has an identifiable code path and an executable, passing acceptance scenario (`TestActorAssignmentsFeatures` — 9 scenarios, 49 steps, all pass).

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| List the roles an actor fills | ✓ Covered | `assignments.go:runAssignmentsListWalk` → `paging.All[glassfrog.Assignment]` → `render.AssignmentsView`/`writeHuman` |
| An assignment row shows its focus and election expiry | ✓ Covered | `templates/assignments.full.tmpl` (Focus / Elected until lines) |
| Assignments read for either a person or an agent | ✓ Covered | `assignments.go:runAssignmentsListWalk` — id passed through via `url.PathEscape` (no kind branch) |
| Actor id does not exist (404) | ✓ Covered | `reportFailure` → shared `classifyClientError` (APIError/3) |
| No usable credential | ✓ Covered | `seam.newClient` fail-safe → `reportFailure` before the walk |
| Actor fills no roles | ✓ Covered | `assignments.{full,compact}.tmpl` `{{if not .Data}}no assignments` |
| Missing actor-id rejected before any request | ✓ Covered | `cobra.ExactArgs(1)` (no request — transport tripwire asserts 0 calls) |
| Paginated list with first-page opt-out | ✓ Covered | `assignments.go:runAssignmentsFirstPage` + `moreAssignmentsNote` |
| Mid-walk failure yields partial set flagged incomplete (plan cross-cutting) | ✓ Covered | `reportIncompleteAssignmentsWalk` → `classifyClientError(Stop)`, non-zero |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 checked tasks' criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — grow `Assignment` with embedded `role` | ✓ Met | `roles.go` `Role struct {…} json:"role"` (id/type/name + nullable purpose/parent_role_id as plain strings); `TestActorAssignmentsPageDecodesEmbeddedRole`, `TestAssignmentRoleEndLeavesEmbeddedRoleZero` pin populated + forward-compatible zero-valued paths |
| T002 — `assignments` render key + view + templates | ✓ Met | `ResourceAssignments` added to `builtinResources` (exhaustiveness guard passes), `AssignmentsView`, two templates; 7 golden/marker tests; explicit-absence markers for all four nullable fields; focus/purpose verbatim |
| T003 — `assignments <actor-id>` command + seam + wiring | ✓ Met | `newAssignmentsCommand` (`ExactArgs(1)`, no filters/no `--include`), `assignmentsSeam`, two-track walked-list `RunE`, wired in `Assemble()`; 17 unit tests across all branches |
| T004 — executable acceptance | ✓ Met | `TestActorAssignmentsFeatures`, `Paths` names only this feature file; 9 behavioral scenarios pass, 4 `@validation` held |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-cli.md) | Status | Implementation |
|---|---|---|
| `assignments <actor-id>` — `ExactArgs(1)`, guard-registered, `SilenceErrors`/`SilenceUsage`, non-empty `Short` | ✓ Conformant | `assignments.go:newAssignmentsCommand` |
| No filter flags, no `--include` | ✓ Conformant | only `--first-page`/`--per-page` declared; unknown-flag tests assert UsageError + 0 calls |
| Inherited `--base-url` / `-o`,`--output` | ✓ Conformant | reads `apiclient.FlagBaseURL` / `output.FlagOutput`, redeclares neither |
| Output: `full`/`compact` row shapes; aggregated `{data:[…]}` for `json`/`yaml` | ✓ Conformant | `writeHuman` over `AssignmentsView`; `aggregateRawData` over `Page[json.RawMessage]` (per-page meta dropped) |
| Empty list → `no assignments`, exit 0 | ✓ Conformant | template empty line + `Success` |
| Completeness signals (more / incomplete notes) | ✓ Conformant | `moreAssignmentsNote`, `incompleteAssignmentsWalkNote` on stderr |
| Error table (auth/transport/non-2xx/output/cobra) | ✓ Conformant | shared `classifyClientError` + `reportFailure`; no new `Outcome`/`ExitCode` |

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 exclusions honored)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No create/update/remove of an assignment | ✓ Absent | only `http.MethodGet` issued; no POST/PATCH/DELETE path |
| No single-assignment read command | ✓ Absent | only the plural `assignments` command exists; no `assignment <id>` constructor or wiring |
| No `--include` flag | ✓ Absent | no `include` flag declared; role arrives via the `json:"role"` default-include decode |
| No client-side filters | ✓ Absent | no `--query`/`--kind`/`--status`/`--role-id` flags; unknown-flag → UsageError |
| No duplication of Role Fillers (047) | ✓ Absent | reads `/actors/{id}/assignments` (vs 047's `/roles/{id}/assignments`); distinct `assignments` render key leading with the role, not the actor |
| No raw-JSON default / own format flag | ✓ Absent | format resolved through Output Format Selection (020); no private format flag |
| No re-implementation of base-url/token/header/exit-code | ✓ Absent | delegates to `assemble`/`newClient`/`RetryExecutor`/`classifyClientError`; never reads `ctx.Cred.Token` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios referenced by checked task T004 have had `@wip` removed and pass in `TestActorAssignmentsFeatures`. The 4 `@validation @wip` scenarios remain tagged — correct, as they are not referenced by any checked task and are held for this validation pass. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation, independent of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| The filled role's name appears without an include flag | ✓ Satisfied | Row leads with `.Role.ID` + `.Role.Name` (`assignments.full.tmpl`); role data decodes from `json:"role"` default include; no `--include` flag declared (`assignments.go`). Independently exercised by `TestRender_AssignmentsFull_LeadsWithRoleIDAndName` and `TestRunAssignments_RoleNameShownWithoutIncludeFlag` (asserts no include/filter query param sent). |
| Focus and election are projected, not dropped | ✓ Satisfied | `Focus`/`Elected until` lines render verbatim with explicit-absence markers (`(none)`/`(not an elected seat)`); `TestRunAssignments_FocusAndElectionProjected` + `TestRender_AssignmentsFull_AbsentFieldsShowMarkers`. |
| A missing token costs no request | ✓ Satisfied | `runAssignmentsList` order is resolve-output → `assemble` → `newClient` → walk; `newClient` (→ `apiclient.NewClient`) raises the no-credentials fail-safe before the walk, so `exec.Execute` is never reached → no request. `TestRunAssignments_NoCredentialsIsUsageError` confirms UsageError/2 with no data printed. (The dedicated transport-tripwire scenario stays held `@wip`; behavior traces cleanly by inspection.) |
| Output is structured, not pre-rendered | ✓ Satisfied | Command supplies structured data (`AssignmentsView` / `Page[json.RawMessage]`) and declares no format flag; all formats render from one result. `TestRunAssignments_StructuredJSONEmitsAggregatedRawPayload` confirms `json` carries the raw payload, omits the human block labels, and drops the per-page `pagination` meta. |

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios are satisfied through independent inspection (and corroborated by the package's golden/unit tests and the godog acceptance suite). The implementation conforms to its specification — additive model growth, one render key, one read-only command, no new outcome/exit-code/flag surface, and the spec's non-behaviors are all honored. The specification loop is closed.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 4 `@validation` scenarios remain `@wip` by design — they document the held-out checks this validation pass traced and are not intended to be un-tagged by implement.
