# Analyze: Organization Tree

**Feature**: 026-organization-tree
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, features/governance-reads/organization-tree.feature, tasks.md
**Checklist context**: checklist.md present (round 2 — 0 findings) — correlated below
**Findings**: 16 checks (16 pass, 0 fail) — 0 P0, 0 P1, 0 P2
**Generated**: 2026-06-09 (round 2 — re-run after clarify + propagation)

---

## Summary

| Severity | Category | Count | Pass | Fail |
|---|---|---|---|---|
| P0 | Consistency | 6 | 6 | 0 |
| P1 | Completeness | 6 | 6 | 0 |
| P2 | Coherence | 4 | 4 | 0 |
| **Total** | | **16** | **16** | **0** |

## Changes Since Previous Run

**Previous (round 1)**: 0 P0, 1 P1 fail (1 failure) | **Current (round 2)**: 0 P0, 0 P1 (0 failures)

**Resolved**:
- ~~P1: K5 — subroles `--include` and `--first-page` surfaces had no scenario coverage~~ → fixed. The clarify session added both as driving scenarios in spec.md, and propagation added them to organization-tree.feature ("Requested related resources are embedded on each child", "The subroles first-page opt-out stops at one page and signals more"). Every interface surface now has scenario coverage.

---

## Consistency (P0): 6/6 passed

All consistency checks pass — no contradiction between artifacts:
- **C1** spec Integration Boundaries ↔ plan System Architecture: the plan names every boundary the spec lists (Glassfrog API's three endpoints, Request Execution 010, Request Authentication 007, Pagination 016, Output Format Selection 020, Exit-Code 004).
- **C2** spec Behavioral Accord ↔ plan architecture: the plan serves every behavior — the two tree reads, the subroles read, depth, per-read include, completeness, and failure routing.
- **C3** spec Non-Behaviors ↔ plan: the plan architects none of the excluded capabilities — it explicitly makes ETag/304 caching a non-behavior, fakes no tree pagination, defines no private output flag, and adds no write path.
- **C4** plan Architecture Decisions ↔ interface Surface: the interface reflects every plan choice (ADR-1 `tree` optional-id / `subroles` ExactArgs; ADR-2 unpaginated TreeNode; ADR-3 reuse of 025's walk + `--first-page`; ADR-4 per-read include + optional depth + id-passthrough).
- **C5** plan System Architecture ↔ tasks Task Scope: every task builds a component the plan names (TreeNode T001, shared RoleDetail T002, `tree` T003, `subroles` T004, acceptance T005); no task introduces an un-planned component.
- **C6** interface Surface ↔ feature steps: every Gherkin step references a surface the interface defines (`glassfrog tree`, `tree <id>`, `--depth 1`→`depth=1`, `--include accountabilities,domains`, `glassfrog subroles <id>`, `--depth 2` rejected).

## Completeness (P1): 6/6 passed

### Passed (6/6)

- **K5** interface → feature *(newly resolved)*: every interface surface now has scenario coverage — `subroles --include` ("Requested related resources are embedded on each child") and the `subroles --first-page` opt-out ("The subroles first-page opt-out stops at one page and signals more") were added to organization-tree.feature; the depth-boundary marker is covered by "A depth-capped node is marked as having subroles below" + its validation scenario. *(Round 1 flagged these two subroles surfaces as uncovered; the clarify + propagation closed the gap.)*

- **K1** spec Driving Scenarios → feature: all 12 driving scenarios (5 happy / 3 error / 4 edge) and 3 validation scenarios have Gherkin equivalents.
- **K2** spec Integration Boundaries → interface: the one external surface (the CLI) has interface-cli.md; the internal seams (010/007/016/020/004) are reused infrastructure, not surfaces needing their own file.
- **K3** plan phases → tasks: all three plan phases decompose into tasks (Schema→T001/T002, Tree reads→T003, Subroles+acceptance→T004/T005).
- **K4** plan components → tasks: every component (TreeNode, shared RoleDetail, the two commands, the two render keys, the walk) has an implementing task.
- **K6** spec User Scenarios → interface: all four user flows (whole-org tree, rooted subtree, subroles, depth-cap) have interface coverage.

## Coherence (P2): 4/4 passed

- **H1** Terminology: `tree`/`TreeNode`, `subroles`, `depth`, `include`, `role` are used consistently across all artifacts. The `members` (include keyword) ↔ `fillers` (struct field) duality is the API's own naming, applied consistently (interface renders a `Members:` section from the `Fillers` field) — an explicit alias, not drift.
- **H2** Detail symmetry: spec↔plan and plan↔tasks are proportionate; no artifact is dramatically thinner or thicker than its neighbor on a shared topic.
- **H3** Scope alignment: spec, interface, and tasks describe the same scope — three reads, two commands, no capability silently added or dropped. (The subroles `--include`/`--first-page` surfaces are within the subroles capability, not a new capability — the gap is scenario coverage, K5, not scope drift.)
- **H4** Phase coverage: tasks cover all plan phases structurally — phase ordering, grouping, and the T001→T003 / T002→T004 / {T003,T004}→T005 dependencies match the plan's structure; no task references a non-existent phase.

---

## Checklist Correlation

- Round 1's two findings shared a root (the spec under-specified the depth boundary and the subroles flags) and were resolved together by the clarify + propagation: checklist's CONSTITUTION VI P1 (tree-render boundary signal) and analyze's K5 P1 (subroles scenario coverage). Round 2: checklist.md reports 0 findings and analyze reports 0 findings — the contract, the render shape, and the scenarios are now mutually consistent.

## Governance Notes

- No checks were skipped — all six artifact types are present, so all 16 base checks ran (no interface/scenario/task scaling beyond 1 file each).
