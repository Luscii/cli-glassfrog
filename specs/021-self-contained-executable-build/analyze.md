# Analyze: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/self-contained-executable-build.feature
**Checklist context**: loaded
**Checks**: 16 (15 pass, 1 fail)
**Generated**: 2026-06-08

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 6 | 6 | 0 |
| Completeness (P1) | 6 | 5 | 1 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **16** | **15** | **1** |

---

## Consistency: 6/6 passed

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — plan's parts (GoReleaser config, build invocations, verification) align with spec's boundaries (API runtime, source tree, 022, 023).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — the architecture serves every described behavior (local build, matrix build, self-containment, verification); none contradicted.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture — plan architects none of the excluded capabilities; it explicitly defers packaging to 022, version injection to 023, and pins the four-target matrix with no Windows.
- **C4** plan § Architecture Decisions ↔ interface-spec § Surface — the `.goreleaser.yaml` contract reflects ADR-1 (GoReleaser build-only) and ADR-2 (`CGO_ENABLED=0`, four-target matrix, verification).
- **C5** plan § System Architecture ↔ tasks § Scope — T001/T002/T003 map to plan components; no task builds anything the plan doesn't describe.
- **C6** interface-spec § Surface ↔ feature steps — every scenario step references a surface the interface defines (build invocations, `dist/`, the self-containment check, the config-guard, the linkage allowlist).

## Completeness: 5/6 passed

### Findings

**P1** | K5: interface-spec.md § Error Communication → features/…/self-contained-executable-build.feature
> The interface defines a `goreleaser`-not-installed fallback ("the verification falls back to a host `go build` … does not error") as an Error-Communication surface, but no scenario covers it. Every other interface surface has scenario coverage; this robustness behavior of the verification does not. Low stakes — it is a test-infrastructure behavior rather than a product behavior, and T002's acceptance criteria already pin it ("runs under `go test` without `goreleaser` installed"). Consider adding a scenario, or accept the gap as deliberately out of product-scenario scope.

### Passed (5/6)

- **K1** spec § Driving Scenarios → feature — all 7 driving scenarios (and all 3 validation scenarios) have Gherkin equivalents.
- **K2** spec § Integration Boundaries → interface — the build's own surface is covered by interface-spec.md; the API (runtime), source tree (input), and 022/023 (downstream, future) boundaries are not build-time surfaces, a justified absence.
- **K3** plan § Phases → tasks — Phase 1 → T001; Phase 2 → T002, T003.
- **K4** plan § Components → tasks § Scope — every component (config, invocations, run+linkage verification, config-guard) has an implementing task.
- **K6** spec § User Scenarios → interface — all three user flows (bare-host run, ship-all-platforms, local build) have interface coverage.

## Coherence: 4/4 passed

### Passed (4/4)

- **H1** Terminology — "self-contained", target/matrix, `CGO_ENABLED=0`, GoReleaser, `dist/`, self-containment verification / config-guard are used consistently across all artifacts.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; no shared topic is 3×+ deeper on one side.
- **H3** Scope alignment — spec capabilities, interface surfaces, and tasks describe the same scope (build + verify; no packaging, no version embedding).
- **H4** Phase coverage — tasks cover both plan phases structurally; the dependency (Phase 2 → Phase 1; T002/T003 → T001) matches the plan.

---

## Checklist Correlation

- Checklist flagged **P1** in tasks.md (CONSTITUTION VII — config/test increment split), since **resolved** by restructuring tasks.md into a single RED→GREEN increment. Analyze's tasks-facing checks (C5, K3, K4, H4) all **passed** — the decomposition is consistent with and complete against the plan, and remains so after the restructure (the same three work-items now ship as one increment). The two skills were complementary, not overlapping: checklist's finding was about PR/increment *grouping*, while analyze confirms the decomposition itself is correct.

---

## Governance Notes

- **All 16 base check types ran** — full artifact set present (spec, plan, 1 interface file, 1 feature file, tasks), so no checks were skipped for missing artifacts.
- **Checklist context**: loaded — 6/7 pass, 1 P1 (CONSTITUTION VII).
