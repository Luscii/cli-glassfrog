# Validate: Role Policies

**Feature**: 034-role-policies
**Round**: 1 of 3
**Date**: 2026-06-10
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, interface-spec.md, features/governance-reads/role-policies.feature, PROJECT.md
**Implementation files**: `internal/glassfrog/{roles.go,document.go}`, `internal/render/render.go` + 4 templates (`policies.{full,compact}.tmpl`, `policy.{full,compact}.tmpl`), `internal/cli/{policies.go,app.go,dispatch.go}` (+ unit/golden/BDD tests)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 2 of 2 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 covered)

Every driving scenario (spec.md § Driving Scenarios + the completeness accord) has an identifiable code path and a passing executable scenario in the godog suite `TestRolePoliciesFeatures` (9 scenarios / 49 steps, all pass).

| Scenario (spec) | Status | Implementation |
|---|---|---|
| List the policies on a role | ✓ Covered | `policies.go:runPoliciesListWalk` → `GET /roles/{id}/policies` via `paging.All[Policy]` |
| Read a single policy with its full body | ✓ Covered | `policies.go:runPolicyGet` → `GET /policies/{id}` → `Document[Policy]` → `policy.full` template (full body verbatim) |
| Narrow a role's policies with a search | ✓ Covered | `policies.go:policiesQuery` sends `q` when `Changed()` && non-empty |
| Policy id does not exist (404) | ✓ Covered | id passed through unvalidated → `reportClientError`/`classifyClientError` → APIError(3) |
| No usable credential | ✓ Covered | 007 fail-safe at send → `*AuthError{NoCredentials}` → UsageError(2), no API data printed |
| Role has no policies | ✓ Covered | empty `Page[Policy]` → `policies` template renders `No policies.`, exit 0 |
| Search flag on the single read is rejected | ✓ Covered | `policy` declares no list flags → cobra unknown-flag → UsageError(2), no request |
| Paginated list with first-page opt-out | ✓ Covered | `runPoliciesFirstPage` → one page + `morePoliciesNote` on stderr, exit 0 |
| Mid-walk failure flagged incomplete | ✓ Covered | `reportIncompletePoliciesWalk` → partial set on stdout, `incompletePoliciesNote` on stderr, non-zero via `classifyClientError(Stop)` |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete; all criteria evidenced)

All tasks in tasks.md are checked (`- [x]`). Each task's acceptance criteria trace to implementation and a passing test.

| Task | Status | Evidence |
|---|---|---|
| T001 — grow `Policy` + generic `Document[T]` | ✓ Met | `Document[Policy]`/`Page[Policy]` decode + null scope + `RoleDocument` alias byte-stable (`internal/glassfrog/policies_test.go`, 4 tests) |
| T002 — `policy`/`policies` render keys + templates | ✓ Met | both formats, empty→`No policies.`, full body verbatim, null/timestamp absence markers, registry guard now 20 templates; `render` imports only `glassfrog`+stdlib (`internal/render/policies_test.go`) |
| T003 — `policies` list command | ✓ Met | walk/empty/`--query` set·empty·unset/`--first-page`/`--per-page`/mid-walk/error classification/structured-raw + command wiring (`internal/cli/policies_test.go`) |
| T004 — `policy` single read | ✓ Met | title+full body, unknown-id→APIError(3), list-only-flag tripwire across all 3 flags, structured-raw (`internal/cli/policies_test.go`) |
| T005 — executable acceptance | ✓ Met | `TestRolePoliciesFeatures` 9/9 pass; 2 `@validation` held; suite scoped to its own feature file |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-cli / interface-spec) | Status | Implementation |
|---|---|---|
| `policies <role-id>` — `ExactArgs(1)`, `--query`/`-q`, `--first-page`, `--per-page`, inherited `--base-url`/`-o` | ✓ Conformant | `newPoliciesCommand` (`policies.go:392`) |
| `policy <pol-id>` — `ExactArgs(1)`, **no** list flags, inherited flags | ✓ Conformant | `newPolicyCommand` (`policies.go:347`) |
| Output: list/single × full/compact + empty `No policies.` + structured `{data:[…]}` | ✓ Conformant | 4 templates + `runPoliciesListWalk`/`runPolicyGet` machine paths |
| Completeness: default walk, `--first-page` "more exist" (exit 0), mid-walk "incomplete" (non-zero) | ✓ Conformant | `morePoliciesNote` / `incompletePoliciesNote`, exact interface wording |
| Error table — all via `classifyClientError`, no new `Outcome`/`ExitCode` | ✓ Conformant | reuses shared classifier; no registry edit |
| Go surface — `Policy` grown, `Document[T]`, `RoleDocument` alias, `newPoliciesCommand`/`newPolicyCommand`/`runPoliciesList`/`runPolicyGet`/`policiesSeam`, render keys `policies`/`policy` | ✓ Conformant | present with the specified shapes |

**Note (not a finding)**: interface-spec's example sketched a `seam.executor(ctx)` shape; the implementation uses the established 4-method seam (`assemble`/`newClient`/`sleep`/`resolveFormat`) that the same accord names as "the same shape as 025's `rolesSeam`" and binds via `productionSeam`. The "shapes, not literal values" framing of that example makes this conformant — `productionSeam` builds the `RetryExecutor`-wrapped `*Client` exactly as specified.

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No single command with an optional positional id | ✓ Absent | two commands, each `cobra.ExactArgs(1)` |
| No inline policy embed; no `role-id` on the single read | ✓ Absent | `policy` takes `pol-id` only, no `--include`; `policies` returns a list |
| No standalone reads of other resources (domains/projects/notes/skills) | ✓ Absent | only `policy`/`policies` added |
| No raw-JSON default; no own format flag | ✓ Absent | default `full`; only `-o`/`--output` (inherited root, 020) selects format |
| No re-resolving base URL/token, header, response typing, or own exit codes | ✓ Absent | reuses `AssembleFromOS`/`AuthTransport`/`classifyClientError`; `Outcome`/`ExitCode` registry untouched |
| No writes/mutations to any policy or role | ✓ Absent | both are `GET` reads |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios (referenced by checked task T005) have had `@wip` removed and pass executably. The 2 `@validation` scenarios retain `@wip` correctly — they are not referenced by any checked task and are held out for this validation pass. No stale `@wip` remains on implemented scenarios.

---

## Validation Scenario Results

**Status**: Satisfied (2 of 2 traced to implementation, independently)

| Scenario | Status | Trace |
|---|---|---|
| The plural and singular commands do not collide on id kind | ✓ Satisfied | The split is structural: `policies` targets `/roles/{id}/policies` (`policies.go:105`) and `policy` targets `/policies/{id}` (`policies.go:273`); neither validates the id locally, so a wrong-kind id is `url.PathEscape`-d onto its own endpoint and surfaces the API's `404`/`400` via the shared classifier — never silently reads the other resource. |
| Output is structured, not pre-rendered (default carries no raw API envelope) | ✓ Satisfied | Neither command declares an output-format flag (only `--query`/`-q`/`--first-page`/`--per-page` on `policies`); both produce typed data dispatched by resolved format. Default `full` renders the `PoliciesView`/`PolicyView` projection — no `data`/`meta` envelope (golden tests + `TestRunPolicies_StructuredJSONEmitsRawPayload` asserts the projection markers are absent from `json` and the envelope is absent from human output). All four formats render from the same fetched result. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and both held-out `@validation` scenarios are satisfied through independent inspection. The implementation conforms to the specification: the two-command split, the `--query`/id pass-through, the completeness signalling, the grown `Policy`/`Document[T]` schema, the four render templates, and the reuse of the shared error/exit/output machinery all match the spec, plan, and interface accords.

**Observation (transparency, not a finding)**: T005 added a guarded stderr print to the shared `internal/cli/dispatch.go` `flagFailed` branch so that a `SilenceErrors` leaf rejecting a flag purely at cobra level (the `policy --query` case) surfaces the usage message — without it, the interface error-table promise ("cobra's unknown-flag message; no request sent") and the "search flag is rejected" scenario could not be met. The change is strictly additive, guarded by `executed.SilenceErrors` (symmetric with the existing Args-validator branch, so non-silenced commands never double-print), introduces no new `Outcome`/`ExitCode`, and broke no existing test. It is recorded in `.score/memory/LEARNINGS.md`. Flagged here for reviewer visibility because it touches a shared file beyond the 034 surface.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 2 `@validation` scenarios stay `@wip` in the feature file as a permanent held-out record (they were verified here by inspection, not by un-`@wip`-ping); leave them tagged so they remain independent of the Builder's suite.
