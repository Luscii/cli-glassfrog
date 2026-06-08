# Analyze: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/unconsumable-output/templated-human-rendering.feature, tasks.md
**Checklist context**: checklist.md present (13/13 pass, 0 findings)
**Findings**: 0 (initial run found 1 P0; resolved this session — see below)
**Generated**: 2026-06-08

---

## Summary

| Severity | Category | Count |
|---|---|---|
| P0 (contradiction) | Consistency | 0 |
| P1 (gap) | Completeness | 0 |
| P2 (drift) | Coherence | 0 |
| **Total** | | **0** |

Consistency: 6/6 pass. Completeness: 6/6 pass. Coherence: 4/4 pass. (Interface checks ran across both `interface-spec.md` and `interface-cli.md`.)

---

## Consistency (P0)

### Resolved this session

**P0 (RESOLVED)** | C2/C4 | **plan.md § ADR-3 (+ Cross-cutting "Data fidelity")** vs **spec.md § Empty and absent data** and **interface-cli.md § full format**
- The initial analyze run found that plan.md ADR-3 still said *"omit an absent field"* — stale wording from before the 2026-06-08 clarification, which reframed the rule so `full` **preserves** the landed explicit-absence markers (`(none)`, `—`, `(no purpose set)`, `(no role)`; `me` omits its empty roles section). spec.md and interface-cli.md were already consistent; only plan.md's wording contradicted them.
- **Resolution**: plan.md ADR-3 (Context, Option 1, Decision), the Cross-cutting "Data fidelity" note, Phase 1, and the Risk mitigation were all updated to *"render the landed explicit-absence marker via `{{if .X}}…{{else}}<marker>{{end}}`; never fabricate a data value"*, with `me`'s roles-omission called out. Architecture unchanged — wording only. Re-evaluated C2/C4: now consistent.

### Passed (6/6)

- **C1** spec Integration Boundaries ↔ plan System Architecture: plan components (`internal/render` + the four reads) align with the spec's named boundaries (011–014, 020, 029). Pass.
- **C3** spec Non-Behaviors ↔ plan System Architecture: plan architects no excluded capability (no `--output` flag, no JSON/YAML, no caller-supplied templates). Pass.
- **C5** plan System Architecture ↔ tasks Task Scope: every task builds a plan-named component (`internal/render` → T001; the four read rewires → T002–T005); no task builds anything plan omits. Pass.
- **C6** interface ↔ feature steps: every scenario step references a surface the interface defines (`full`/`compact` templates, `roles=3`, `no projects`, render failure). Pass.
- **C4** plan ADRs ↔ interface files: the `internal/render` API contract (`Render`, `Resource`/`Format`, `RenderError`, `//go:embed` set) reflects plan ADR-1/2/4; after this session's plan ADR-3 reconciliation, the marker-vs-omit wording also agrees with interface-cli.md. Pass.

---

## Completeness (P1)

### Passed (6/6)

- **K1** spec Driving Scenarios → feature file: all 7 driving scenarios (3 happy / 2 error / 2 edge) have Gherkin equivalents; the 3 validation scenarios and 1 architecture-informed scenario are also present. Pass.
- **K2** spec Integration Boundaries → interface files: the CLI surface → interface-cli.md; the `internal/render` package surface → interface-spec.md; 020/029 are downstream (not surfaces to build now). Pass.
- **K3** plan phases → tasks: Phase 1 → T001; Phase 2 → T002–T005. Pass.
- **K4** plan components → tasks: `internal/render` → T001; each of the four reads → T002–T005. Pass.
- **K5** interface surfaces → feature coverage: `full`, `compact`, the empty-result line, and the render-failure/`RenderError` path each have scenario coverage. Pass.
- **K6** spec User Scenarios → interface: US1 (human text) / US2 (compact) / US3 (full detail) each map to interface-cli.md surfaces. Pass.

---

## Coherence (P2)

### Passed (4/4)

- **H1** Terminology: `full`/`compact`, resource/format, render seam, and the per-result-type templating are used consistently; spec's generic "template mechanism" is refined to plan/interface's concrete `text/template`/`Render` without conflicting aliases. Pass.
- **H2** Detail symmetry: spec↔plan and plan↔tasks are proportionate — no artifact is dramatically over/under-detailed on a shared topic. Pass.
- **H3** Scope alignment: spec capabilities, interface surfaces, and tasks work describe the same scope (two human templates, four reads rewired, no flag, no JSON). Pass.
- **H4** Phase coverage: tasks cover plan's phase structure including dependencies (T002–T005 depend on T001, mirroring Phase 2 depends on Phase 1). Pass.

---

## Checklist Correlation

Checklist (vertical) reported 13/13 pass with no findings. The analyze P0 is a **horizontal-only** finding: plan.md ADR-3 is internally fine (so it passes every vertical check), but it contradicts the clarified spec.md and interface-cli.md — a contradiction only visible when comparing artifacts against each other. No checklist finding overlaps.

---

## Governance Notes

- All expected artifacts are present; no checks were skipped for missing artifacts.
- Guardian agent definition not found at the expected path — ran the process without the character layer.
