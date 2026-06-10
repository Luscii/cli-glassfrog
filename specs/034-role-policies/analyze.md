# Analyze: Role Policies

**Feature**: 034-role-policies
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/governance-reads/role-policies.feature, tasks.md
**Checklist context**: checklist.md loaded (0 fail — 20/20 pass, 16 P0 + 4 P1)
**Findings**: 16 cross-artifact checks (16 pass, 0 fail); interface checks evaluated across both interface files
**Generated**: 2026-06-10

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

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — plan's components (`internal/cli` `policies`/`policy`, `internal/glassfrog` `Policy` growth + `Document[T]`, `internal/render` keys) align with the spec's named boundaries (Glassfrog API `GET /roles/{id}/policies` + `GET /policies/{id}`, Request Execution 010, Request Authentication 007, Pagination 016, Output Format Selection 020, Exit-Code 004). PASS.
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — the architecture serves every described behavior: per-role list, single read, the `--query` search, the default walk + opt-out + mid-walk-failure completeness, and the named-failure paths. PASS.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture — plan architects nothing the spec excludes: no embedded-on-role view (that stays 025's `--include`), no standalone domain/project/note reads, no raw-JSON default (renders via 020), no write path. ADR-1's two-command split realizes the "list filter is list-only" non-behavior structurally. PASS.
- **C4** plan § Architecture Decisions ↔ interface-cli.md + interface-spec.md § Surface — both interface files reflect the plan's ADRs: two sibling commands `ExactArgs(1)` (ADR-1), grown `Policy` + `Document[T]` (ADR-2), `--query`/id pass-through with no local validator (ADR-3), `policies`/`policy` render keys (ADR-4). PASS (evaluated across both interface files).
- **C5** plan § System Architecture ↔ tasks § Scope — every task builds something the plan describes (T001 schema + `Document[T]`, T002 render, T003 list command, T004 single command, T005 acceptance); no task introduces an unplanned component. PASS.
- **C6** interface-cli.md § Surface ↔ role-policies.feature steps — every scenario step references a defined surface (`glassfrog policies <role-id>`, `glassfrog policy <pol-id>`, `--query`, the `q` parameter, `--first-page`, exit codes 0/2/non-zero, the "more exist" / "incomplete" stderr notes). No step uses an undefined command, flag, or field. PASS.

## Completeness (P1): 6/6 passed

- **K1** spec § Driving Scenarios → role-policies.feature — all 8 spec driving scenarios have a Gherkin equivalent (list / empty / no-token / search / first-page-opt-out / single-read / unknown-id / search-on-single-rejected), plus both `@validation` scenarios (id-kind collision, structured-output) and 1 architecture-informed mid-walk-failure. PASS.
- **K2** spec § Integration Boundaries → interface files — the CLI surface has interface-cli.md; the Go package surface (the `Policy` growth + `Document[T]` + command symbols + render keys) has interface-spec.md; the internal consumed seams (009/010/016/017/018/019/020) are justified-absent (reused, not re-specified). The upstream Glassfrog API boundary is consumed, not produced — no interface file is owed for it. PASS.
- **K3** plan § Phases → tasks — Phase 1 → T001, Phase 2 → T002, Phase 3 → T003+T004, Phase 4 → T005. Every phase decomposes. PASS.
- **K4** plan § Components → tasks § Scope — every component has an implementing task: `Policy` growth + `Document[T]` (T001), the `policies` list command + seam (T003), the `policy` single command (T004), the `render` keys (T002), wiring + acceptance (T003/T004/T005). PASS.
- **K5** interface-cli.md § Surface → role-policies.feature — every 034-specific surface and the principal exit-code paths have scenario coverage (list, single, `--query`, both completeness signals, auth/API failures, the list-only-flag-on-`policy` usage error). The generic cross-cutting error rows (decode→1, base-URL→2, invalid `--output`→2) are reused behaviors covered by their owning specs' suites (011/020), not 034-specific surfaces — see Governance Notes. PASS.
- **K6** spec § User Scenarios → interface-cli.md § Surface — all four user flows have interface coverage: list a role's policies (US1 → `policies <role-id>`), read one policy's full body (US2 → `policy <pol-id>`), narrow with search (US3 → `--query`), trust the list is whole (US4 → walk + opt-out + signal). PASS.

## Coherence (P2): 4/4 passed

- **H1** Terminology — the key concepts (policy / `Policy`, addressable vs embedded, role-id vs pol-id, `Document[T]` / `Page[T]`, walk / `--first-page` opt-out, `--query`/`q`, projection) are named consistently across spec, plan, both interface files, feature, and tasks. PASS.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; no artifact carries 3×+ the detail of its neighbour on a shared topic. PASS.
- **H3** Scope alignment — the capability set (spec: per-role list + single read + search + completeness) equals the surfaces (interface-cli: `policies` + `policy`) equals the work (tasks T001–T005). Nothing is added or dropped silently; the embedded-on-role view is explicitly out-of-scope (deferred to 025) in both spec and plan. PASS.
- **H4** Phase coverage — tasks reference only plan-defined phases (1/2/3/4), and every plan phase has corresponding tasks; the plan's "Phases 1–2 independent, 3 depends on both, 4 depends on 3" structure is mirrored in the tasks dependency graph. PASS.

---

## Checklist Correlation

Checklist found zero failures (20/20 pass). Its one elevated note — the P1 under CONSTITUTION V (the `Document[T]` generalization touching landed 025 `RoleDocument`) — is a vertical observation about a shared-schema change, not a cross-artifact contradiction; analyze confirms the change is consistent across plan ADR-2, interface-spec.md (`Document[T]` / `RoleDocument` alias), and tasks T001 (byte-stability acceptance) — C4/C5 PASS on exactly those sections. No horizontal finding contradicts the vertical note. The agreements analyze confirms (C2/C4 on the walk + opt-out + failure semantics; C6 on the surface) are consistent with checklist's PASS on CONSTITUTION III/VI/X.

## Governance Notes

- All expected artifacts present; no checks skipped. Both interface files (cli + spec) were covered for the interface-related checks (C4, K2, K5, K6).
- **K5 cross-cutting-error coverage**: the decode / base-URL / invalid-`--output` failure rows in interface-cli.md's error table are reused, shared behaviors owned and scenario-tested by 011 (`classifyClientError`) and 020 (`--output` resolution). The role-policies feature file deliberately scenarios only the 034-specific surfaces plus the principal error classes (auth/API/usage). This is intentional reuse, not a coverage gap.
- **Feature file size** (coherence-adjacent, not a check): role-policies.feature carries 11 scenarios (8 spec-derived + 2 validation + 1 architecture-informed), within the ~12 soft cap. One coherent capability, one file.
- Done-criteria coverage is checklist's domain; see its note on absent `accords/governance/done-*.md`.
