# Validate: Organization Tree

**Feature**: 026-organization-tree
**Round**: 1 of 3
**Date**: 2026-06-09
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (5 of 5 tasks complete), interface-cli.md, features/governance-reads/organization-tree.feature, PROJECT.md
**Implementation files**: `internal/glassfrog/tree.go` (+ test), `internal/cli/tree.go` (+ test), `internal/cli/subroles.go` (+ test), `internal/cli/include.go`, `internal/cli/organization_tree_bdd_test.go`, `internal/render/render.go` + `templates/{tree,subroles}.{full,compact}.tmpl`; reuse-only edits to `internal/glassfrog/roles.go` (via 025) confirmed in `roles_test.go`; wiring in `internal/cli/app.go`.

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

Supplementary test confidence (inspection is the baseline): `go build ./...`, `go vet ./...`, `go test ./...` all clean; the held-out `TestOrganizationTreeFeatures` godog suite reports 16/16 behavioral scenarios passing (88 steps), with the 4 `@validation` scenarios held `@wip` for this pass.

---

## Driving Scenario Coverage

**Status**: Pass (15 of 15 behavioral scenarios covered)

Every driving scenario (spec.md § Driving Scenarios, realized in the feature file) traces to an identifiable code path.

| Scenario | Status | Implementation |
|---|---|---|
| Read the whole organization tree | ✓ Covered | `tree.go:runTree` (0 args) → `treeRequest` → `GET /tree` → `runTreeRead` |
| Read the subtree rooted at a role | ✓ Covered | `tree.go:treeRequest` (1 arg) → `GET /roles/{id}/tree` (PathEscape) |
| List a role's immediate subroles | ✓ Covered | `subroles.go:runSubrolesList` → `paging.All[RoleDetail]` over `GET /roles/{id}/subroles` |
| Bound the tree depth | ✓ Covered | `tree.go:treeQuery` sends `depth` only when `depthSet` (`Changed`) |
| Read a tree with related resources per node | ✓ Covered | `treeQuery` comma-joins `include`; `render.NewTreeView` + guarded template sections |
| List subroles with related resources embedded | ✓ Covered | `subrolesQuery` sends `include`; `SubrolesView` + per-child guarded sections |
| No usable token | ✓ Covered | `runTree`/`runSubroles` → `newClient` fail-safe → `reportClientError` → UsageError(2) |
| A tree read for an unknown role id | ✓ Covered | `runTreeRead` → `reportClientError` → shared classifier (404 → APIError(3)) |
| Unsupported `--include` rejected without an API call | ✓ Covered | `validateIncludeSet` before assembly; transport tripwire (0 calls) |
| A leaf role's tree is a single node | ✓ Covered | `NewTreeView` flattens a childless root to one row; no nested output |
| A leaf role has no subroles | ✓ Covered | `subroles.{full,compact}.tmpl` renders `No subroles.` on empty `Children` |
| Subroles span more than one page (default walk) | ✓ Covered | `paging.All[RoleDetail]` walks to completion |
| `--depth` rejected on subroles | ✓ Covered | `validateSubrolesFlags` rejects `depthSet`; tripwire (0 calls) |
| A depth-capped node is marked as having more below | ✓ Covered | `TreeRow.BelowDepth` (`HasSubroles && len(Children)==0`) → `(+ subroles below depth)` / `has_subroles=yes` |
| First-page opt-out stops at one page and signals more | ✓ Covered | `runSubrolesFirstPage` → single `Execute` + `moreSubrolesNote` (exit 0) |

Mid-walk-failure partial-set behavior (spec.md § Completeness; ADR-3) also traces: `runSubrolesList` → `reportIncompleteSubrolesWalk` writes the incomplete note and exits non-zero via `classifyClientError(Stop)`.

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 `TreeNode` + `{data: TreeNode}` wrapper | ✓ Met | `glassfrog/tree.go`: recursive `Children []TreeNode`, nullable `*string` name/purpose/parent, `Flags`, per-`include` fields reusing `Accountability`/`Domain`/`Actor`; 4 decode tests (recursion, nullability, include population, leaf + unknown-field tolerance) |
| T002 shared role-detail schema (reuse-only) | ✓ Met | 025 landed the schema first; no duplicate type introduced. `Page[RoleDetail]` decode confirmed by `TestSubrolesPageDecodesRoleDetail` (the paginated shape `subroles` decodes) |
| T003 `tree` command | ✓ Met | depth + include validation before assembly, both endpoints, `TreeDocument` decode, recursive depth-indented render, registry guard passes, wired in `Assemble`; 14 unit tests cover every branch |
| T004 `subroles` command | ✓ Met | include validation, `paging.All` walk + `--first-page` opt-out + `--per-page` + mid-walk completeness note, `subroles` render, wired; 12 unit tests |
| T005 executable acceptance | ✓ Met | `TestOrganizationTreeFeatures` `Paths` names only this feature file; 16 behavioral scenarios un-`@wip`'d and passing; 4 `@validation` held |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Conformance |
|---|---|---|
| `glassfrog tree [id]` | ✓ Conformant | `Use: "tree [id]"`, `Args: MaximumNArgs(1)`, `--depth`/`--include`, inherited `--base-url`/`-o`, `SilenceErrors`/`SilenceUsage` |
| `glassfrog subroles <id>` | ✓ Conformant | `Use: "subroles <id>"`, `Args: ExactArgs(1)`, `--include`/`--first-page`/`--per-page` |
| Flag applicability rejections | ✓ Conformant | pagination flags rejected on `tree`, `--depth` rejected on `subroles` — each with an operator-facing message + tripwire |
| Output dispatch | ✓ Conformant | `json`/`yaml` emit raw payload verbatim (raw-bytes path); `full`/`compact` render the human projection via two **new** render keys (`tree`, `subroles`) |
| Depth-boundary signal | ✓ Conformant | `full`: `(+ subroles below depth)`; `compact`: `has_subroles=yes`; no invented count |
| Subroles completeness notes | ✓ Conformant | `moreSubrolesNote` and `incompleteSubrolesNote` match interface §Interactions wording verbatim |
| Error communication | ✓ Conformant | all conditions routed through the shared `classifyClientError`; no new `Outcome`/`ExitCode`/root flag |

---

## Non-Behavior Absence

**Status**: Pass (all 6 exclusions absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No tree pagination / fabricated depth-walk | ✓ Absent | `runTreeRead` issues exactly one `Execute`; no `paging.All` on the tree path |
| No ETag cache / `If-None-Match` | ✓ Absent | grep of `tree.go`/`subroles.go` finds no `If-None-Match`/`ETag`/`304` |
| No pass-through unknown include; no shared include set | ✓ Absent | two validators (`supportedTreeIncludes` vs `supportedSubrolesIncludes`), each reject-unknown before any request |
| No raw-JSON default / no own format flag | ✓ Absent | human path renders typed views; format owned by `output`/`render` (020), no local `--output` |
| No base-URL/token/header/exit-code reimplementation | ✓ Absent | reuses `assemble`/`newClient`/`RetryExecutor`/`classifyClientError`; token never referenced |
| No write/mutate of governance | ✓ Absent | both requests are `http.MethodGet`; no POST/PUT/PATCH/DELETE |

---

## @wip Lifecycle Completion

**Status**: Pass

All 16 behavioral scenarios referenced by the (checked) T005 have had `@wip` removed and pass under `TestOrganizationTreeFeatures`. The 4 `@validation @wip` scenarios remain held (lines 35, 89, 162, 194) — correct: they are reserved for this validate pass and not owned by any implement task. No stray `@wip` remains on an implemented scenario.

> Note: tasks.md T005 says "the 3 `@validation` scenarios"; the feature file actually carries **4**. Implemented and validated to the file (4 held). Documentary drift only — recorded in LEARNINGS during implement; not a conformance gap.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| Tree default output carries no raw API envelope | ✓ Satisfied | Human path (`tree.go:176`) decodes `TreeDocument` and renders `render.NewTreeView` — a typed projection; the raw `{data:…}` envelope is emitted **only** on the `json`/`yaml` machine path. The `tree.{full,compact}.tmpl` templates contain no `data` wrapper. |
| CLI rejects unknown includes the API would silently ignore | ✓ Satisfied | `validateIncludeSet` runs **before** assembly; `TestRunTree_UnsupportedIncludeRejectedBeforeRequest` asserts `transport.calls == 0`. Reject-unknown is local, not deferred to the API's silent-ignore. |
| Subroles incompleteness is never silent | ✓ Satisfied | `reportIncompleteSubrolesWalk` writes an explicit `note: result is incomplete — <cause>…` to stderr and returns a non-zero classified outcome; `--first-page` writes `moreSubrolesNote`. Partial sets are always flagged. |
| A depth-capped node is distinguishable from a leaf | ✓ Satisfied | `TreeRow.BelowDepth` marks a `has_subroles && len(children)==0` node distinctly from a true leaf; `render` test `TestRender_TreeFull_DepthBoundaryMarker` asserts the marker is present on the capped node, absent on the leaf, and carries no invented descendant count. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 4 held-out validation scenarios are independently traceable to code paths. The implementation conforms to the specification: both completeness models (unpaginated tree vs. walked subroles list) are honored distinctly, the depth-boundary signal closes the silent-truncation gap (CONSTITUTION VI), the two per-read include sets are validated locally, and every non-behavior is absent. No new `Outcome`/`ExitCode`/root flag was introduced; the read reuses the landed transport, pagination, classification, and output stack. The specification loop is closed.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main` from `boitewitte/implement-spec-026`). After merge, mark 026 Complete in STATUS.md (validate has set it to Ready).
