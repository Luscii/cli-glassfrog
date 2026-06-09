# Validate: Role Reads

**Feature**: 025-role-reads
**Round**: 1 of 3
**Date**: 2026-06-09
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (5 of 5 tasks complete), interface-cli.md, interface-spec.md, features/governance-reads/role-reads.feature, PROJECT.md
**Implementation files**: `internal/cli/roles.go` (command + run/validate functions), `internal/glassfrog/roles.go` (schema), `internal/render/render.go` + `templates/{org-roles,role}.{full,compact}.tmpl` (rendering), `internal/cli/app.go` (wiring); tests in `internal/cli/{roles_test.go,role_reads_bdd_test.go}` and `internal/glassfrog/roles_test.go`

> Note: `agents/guardian-agent.md`, `references/context-engineering-review.md`, and `references/self-verification-checklist.md` are not deployed in this Score cache — applied the SKILL.md process and the self-checks inline.

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (12 of 12 behavioral scenarios + 1 architecture-informed covered)

Every driving/behavioral scenario has an identifiable code path, and each is exercised by a unit test and/or the `TestRoleReadsFeatures` godog suite (13 scenarios passing).

| Scenario | Status | Implementation |
|---|---|---|
| List the organization's roles | ✓ Covered | `runRolesList` walk via `paging.All` + `org-roles` render (`roles.go:151`) |
| Read a single role by id (name/purpose/accountabilities/domains/fillers) | ✓ Covered | `runRoleGet` + `role.full.tmpl` (`roles.go:335`) |
| Filter the list by parent circle | ✓ Covered | `rolesListQuery` → `parent_role_id` (`roles.go:283`) |
| Read a single role with related resources embedded | ✓ Covered | `runRoleGet` `?include=` + guarded `role` sections (`roles.go:336`) |
| No usable token | ✓ Covered | `reportClientError` → `UsageError` on first-page `Stop` (`roles.go:187`) |
| The API cannot be reached | ✓ Covered | `reportClientError` → `NetworkUnavailable` (`roles.go:165/187`) |
| A single read for an unknown id (404) | ✓ Covered | id passed through; `reportClientError` → `APIError` (`roles.go:357`) |
| Unsupported `--include` rejected without an API call | ✓ Covered | `validateRolesInclude` before assembly (`roles.go:113`) |
| The organization has no roles | ✓ Covered | empty walk → `org-roles` "No roles." (`org-roles.full.tmpl`) |
| Roles span more than one page (default walk) | ✓ Covered | `paging.All` walk to completion (`roles.go:182`) |
| First-page opt-out stops + signals more | ✓ Covered | `runRolesFirstPage` + `moreRolesNote` (`roles.go:239`) |
| A list filter rejected on the single read | ✓ Covered | `validateRolesFlags` (`roles.go:96`) |
| Mid-walk failure → partial set flagged incomplete (arch-informed) | ✓ Covered | `reportIncompleteWalk` (`roles.go:206`) |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete; criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — schema growth | ✓ Met | `internal/glassfrog/roles.go`: grown `Role`, `RoleDetail`, leaf models, `RoleDocument`; decode tests pin grown fields, null `parent_role_id` → nil, omitted includes empty, unknown fields tolerated |
| T002 — list command + scaffold | ✓ Met | walk + render + `validateRolesFlags` + `Assemble` wiring; error mappings via shared classifier; offline tests; no new `Outcome`/`ExitCode`/root flag |
| T003 — filters/--first-page/--per-page/completeness | ✓ Met | `rolesListQuery` (incl. tri-state `has_subroles`), `runRolesFirstPage`, `WithPageSize`, `reportIncompleteWalk`; transport tripwire on filter+id |
| T004 — single read + `--include` + `role` templates | ✓ Met | `runRoleGet`, `validateRolesInclude`, `role.{full,compact}.tmpl` with guarded sections via `RoleView` |
| T005 — godog acceptance | ✓ Met | `TestRoleReadsFeatures` (Paths → `role-reads.feature` only): 13 scenarios pass, 3 `@validation` held |

**Observation (not a finding)**: T002/T004's acceptance text said `-o json`/`yaml` "emit the raw payload". The implementation deliberately emits an aggregated `{"data":[…]}` document for the **list** (each role's bytes preserved verbatim; only the envelope synthesized) so structured and human output fetch the *same* roles — a post-implementation correction the developer requested for predictability, with the interface-cli/interface-spec Output sections updated to match. This conforms to the **spec**, which (§ Output / § Completeness) requires only delegation to Output Format Selection (020), walk-by-default, and never-silently-truncated — all satisfied. The single read still emits the raw `{data: RoleDetail}` verbatim.

---

## Interface Contract Conformance

**Status**: Pass (command, flags, signatures, error table, render keys all conformant)

| Surface | Status | Evidence |
|---|---|---|
| `roles [id]` runnable leaf (`MaximumNArgs(1)`, `Short`, `SilenceErrors/Usage`, inherited `--base-url`/`--output`) | ✓ Conformant | `newRolesCommand` (`roles.go:524`) |
| List flags `--parent`/`--person`/`--tag`/tri-state `--has-subroles`/`--first-page`/`--per-page` | ✓ Conformant | flag decls `roles.go:581`; `--has-subroles` via `Changed` |
| Single-read flag `--include` (comma-separated closed enum) | ✓ Conformant | `StringSliceVar` + `validateRolesInclude` |
| `runRoles`/`runRolesList`/`runRoleGet`/`validateRolesInclude`/`validateRolesFlags`/`rolesSeam` | ✓ Conformant | signatures match interface-spec (updated to `runRolesList(cfg, exec, format)`) |
| Error communication table (auth→2, transport→6, non-2xx→3/4/5, decode→1, base-url→2, format→2, include/flag-combo→2) | ✓ Conformant | routed through shared `classifyClientError`; no new `Outcome`/`ExitCode` |
| Render keys `org-roles` + `role` (both `full`/`compact`); registry exhaustiveness guard | ✓ Conformant | `render.go` builtinResources; `TestRegistry_AllBuiltinsResolve` passes |
| Output: list structured `{data:[…]}` walked; single raw `{data: RoleDetail}`; compact id-first `key=value` | ✓ Conformant | `aggregateRawRoles`; `org-roles.compact.tmpl` matches the repo convention |

---

## Non-Behavior Absence

**Status**: Pass (no excluded behavior present)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not scope the list to the token's roles / drop the selector | ✓ Absent | list hits org-wide `GET /roles` with filters; not token-scoped (distinct from `me roles`) |
| Must not provide standalone reads of related resources | ✓ Absent | `--include` embeds inline on the role only; no `GET /policies/{id}` etc. |
| Must not emit raw API JSON as a fixed default nor define its own format flag | ✓ Absent | default is `full` via 020's `resolveFormat`; uses inherited `--output`, declares no format flag |
| Must not resolve base URL/token, attach header, type a non-2xx, or choose exit codes | ✓ Absent | delegated to 005/007/008/015; `runRoles` adds no `Outcome`/`ExitCode`; never reads `ctx.Cred.Token` |
| Must not write or mutate any role | ✓ Absent | only `GET` requests issued |

---

## @wip Lifecycle Completion

**Status**: Pass

All 13 behavioral scenarios have had `@wip` removed and pass under `TestRoleReadsFeatures`. The only remaining tags are `@validation @wip` on the 3 held-out validation scenarios (lines 68, 120, 160) — correctly retained for this skill's independent verification, not referenced as un-wip deliverables by any task.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation)

These are held out from the Builder (kept `@validation @wip`, skipped by the `~@wip` godog filter) and verified here by independent inspection; each behavior is additionally pinned by an existing unit test.

| Scenario | Status | Trace |
|---|---|---|
| Default output carries no raw API envelope | ✓ Satisfied | default format = `full` → `org-roles.full.tmpl` renders `Name (id) / Purpose / Domains / Accountabilities` — no `data`/`meta`, no raw role objects (`roles.go:189`; `TestRunRoles_ListSuccessWalksAndProjects`) |
| Embedded-include view does not substitute for the standalone reads | ✓ Satisfied | `--include policies` renders `Policies:` inline within the role block (`role.full.tmpl`); no standalone per-policy command/projection exists (Non-behavior absent) |
| List incompleteness is never silent | ✓ Satisfied | mid-walk `Stop` → `reportIncompleteWalk` writes the explicit "result is incomplete — <cause>; … partial set" note + non-zero exit, in both human and structured paths (`roles.go:206`; `TestRunRoles_MidWalkFailure…`, `…StructuredMidWalkFailure…`) |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 validation scenarios are satisfied through inspection (and corroborated by the unit + godog suites, all green). The implementation conforms to the specification: the org-wide list and single read are present with the contracted flags, error mapping, completeness signalling, and render keys; every spec non-behavior is absent; and the format-independent walk satisfies the spec's "walk by default / never silently truncated" accord. The single post-implementation change (structured list output aggregates the walked set rather than emitting one raw page) is conformant to the spec and reflected in the interface accords.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (PR #61 is open). The specification loop for 025-role-reads is closed.
