# Analyze: Role Reads

**Feature**: 025-role-reads
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/governance-reads/role-reads.feature, tasks.md
**Checklist context**: checklist.md loaded (0 P0, 0 P1, 0 P2 — all 20 pass)
**Findings**: 16 cross-artifact checks (16 pass, 0 fail); interface checks evaluated across both interface files
**Generated**: 2026-06-08

---

## Summary

All 16 cross-artifact checks pass. Consistency: 6/6. Completeness: 6/6. Coherence: 4/4.

| Severity | Category | Count | Pass | Fail |
|---|---|---|---|---|
| P0 | Consistency | 6 | 6 | 0 |
| P1 | Completeness | 6 | 6 | 0 |
| P2 | Coherence | 4 | 4 | 0 |
| **Total** | | **16** | **16** | **0** |

---

## Consistency (P0): 6/6 passed

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — plan's components (`internal/cli` `roles`, `internal/glassfrog` growth, `internal/paging`, `internal/render`) align with the spec's named boundaries (Glassfrog API, Request Execution 010, Request Authentication 007, Pagination 016, Output Format Selection 020, Exit-Code 004). PASS.
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — the architecture serves every described behavior: list, single read, the four filters, `--include`, the default walk + opt-out + mid-walk-failure completeness, and the named-failure paths. PASS.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture — plan architects nothing the spec excludes: no standalone related-resource reads (ADR-2 keeps `--include` embed-only), no raw-JSON default (renders via 020), no write path. PASS.
- **C4** plan § Architecture Decisions ↔ interface-cli.md + interface-spec.md § Surface — both interface files reflect the plan's ADRs: one runnable command + optional id (ADR-1), grown `Role` + `RoleDetail` (ADR-2), default walk vs `--first-page` (ADR-3), validate-include / pass-id-through (ADR-4). PASS (evaluated across both interface files).
- **C5** plan § System Architecture ↔ tasks § Scope — every task builds something the plan describes (T001 schema, T002/T003 list, T004 single, T005 acceptance); no task introduces an unplanned component. PASS.
- **C6** interface-cli.md § Surface ↔ role-reads.feature steps — every scenario step references a defined surface (`glassfrog roles`, `roles <id>`, `--parent`, `--include`, exit codes 0/2/6/non-zero, the two stderr notes). No step uses an undefined command, flag, or field. PASS.

## Completeness (P1): 6/6 passed

- **K1** spec § Driving Scenarios → role-reads.feature — all 12 spec driving scenarios have a Gherkin equivalent (+3 `@validation`, +1 architecture-informed mid-walk-failure). PASS.
- **K2** spec § Integration Boundaries → interface files — the CLI surface has interface-cli.md; the Go package surface (the schema growth + command symbols + render keys downstream specs consume) has interface-spec.md; the internal consumed seams (009/010/016/017/018/019/020) are justified-absent (reused, not re-specified). PASS.
- **K3** plan § Phases → tasks — Phase 1 → T001, Phase 2 → T002+T003, Phase 3 → T004+T005. Every phase decomposes. PASS.
- **K4** plan § Components → tasks § Scope — every component has an implementing task: `glassfrog` growth (T001), the `roles` command list path (T002/T003) and single path (T004), the `render` keys (T002/T004), wiring + acceptance (T002/T005). PASS.
- **K5** interface-cli.md § Surface → role-reads.feature — every 025-specific surface and the principal exit-code paths have scenario coverage (list, single, each filter shape, `--include` valid + invalid, both completeness signals, auth/transport/API failures). The generic cross-cutting error rows (decode→1, base-URL→2, invalid `--output`→2) are reused behaviors covered by their owning specs' suites (011/020), not 025-specific surfaces — see Governance Notes. PASS.
- **K6** spec § User Scenarios → interface-cli.md § Surface — all four user flows have interface coverage: navigate/drill (list + single), filter (filter flags), embed (`--include`), completeness (walk + opt-out + signal). PASS.

## Coherence (P2): 4/4 passed

- **H1** Terminology — the key concepts (role / `RoleDetail`, org-wide vs token-scoped, walk / `--first-page` opt-out, embed `--include`, projection) are named consistently across spec, plan, both interface files, feature, and tasks. PASS.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; no artifact carries 3×+ the detail of its neighbour on a shared topic. PASS.
- **H3** Scope alignment — the capability set (spec: list + single + filters + include + completeness) equals the surfaces (interface-cli) equals the work (tasks). Nothing is added or dropped silently; `Q` search is explicitly out-of-scope in both spec and plan. PASS.
- **H4** Phase coverage — tasks reference only plan-defined phases (1/2/3), and every plan phase has corresponding tasks. PASS.

---

## Checklist Correlation

Checklist found zero issues (20/20 pass), so there are no vertical findings to correlate. The cross-artifact agreements analyze confirms here (C2/C4 on the walk + opt-out + failure exit semantics; C6 on the exit-code surface) are consistent with checklist's PASS on CONSTITUTION III/VI/X.

## Governance Notes

- All expected artifacts present; no checks skipped. Both interface files (cli + spec) were covered for the interface-related checks (C4, K2).
- **K5 cross-cutting-error coverage**: the decode / base-URL / invalid-`--output` failure rows in interface-cli.md's error table are reused, shared behaviors owned and scenario-tested by 011 (`classifyClientError`) and 020 (`--output` resolution). The role-reads feature file deliberately scenarios only the 025-specific surfaces plus the principal error classes (auth/transport/API). This is intentional reuse, not a coverage gap.
- **Feature file size** (coherence-adjacent, not a check): role-reads.feature carries 16 scenarios (12 spec-derived + 3 validation + 1 architecture-informed), above the ~12 soft cap. It is one coherent capability (list + single read), so it was kept in one file; flagged for awareness, not split.
- Done-criteria coverage is checklist's domain; see its note on absent `accords/governance/done-*.md`.
