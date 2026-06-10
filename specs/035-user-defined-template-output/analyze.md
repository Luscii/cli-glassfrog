# Analyze: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/unconsumable-output/user-defined-template-output.feature, tasks.md
**Checklist context**: loaded — 12/12 constitution checks pass
**Checks**: 21 (21 pass, 0 open) — scaled for 2 interface files. One coherence finding (H3) was raised by this run and **resolved within the PR**; see Post-Analysis Update.
**Generated**: 2026-06-10

---

> **Post-Analysis Update (commit 5bead74).** This analysis originally raised one P2 coherence finding, **H3**: the spec's Behavioral Accord did not mention the built-in-composition capability that plan ADR-2 and the interfaces expose. It was resolved within this PR — spec.md now states a user template "may also compose a built-in view by name." No open findings remain. The finding is retained below (marked resolved) as an audit record rather than deleted.

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 8 | 8 | 0 |
| Completeness (P1) | 9 | 9 | 0 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **21** | **21** | **0** |

*H3 (the one coherence finding) was raised by this run and resolved within the PR by commit 5bead74 — counted as resolved above; see the Coherence section and the Post-Analysis Update.*

---

## Consistency: 8/8 passed

### Findings

None — no contradictions between artifacts.

### Passed (8/8)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture: the plan's three-layer extension (`internal/output`, `internal/render`, `internal/cli`) maps onto spec's named boundaries (020 selection, 019 seam, read commands, filesystem/stdin, exit codes).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture: the flag-only recognition / read+parse-fail-fast / render-through-seam flow serves the spec's behaviors with no contradiction.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture: the plan architects nothing the spec excludes — flag-only (no env/config templates), clone-not-reimplement, success-only, anti-fabrication, data-only sandbox, no fetched-field change all hold.
- **C4** (×2) plan § Architecture Decisions ↔ interface-cli / interface-spec § Surface: the widened flag value set + error table (cli) and the discriminated selection / engine clone / typed error / seam I/O (spec) reflect ADR-1–4.
- **C5** plan § System Architecture ↔ tasks § Task Scope: every task builds a component the plan names (engine T001, selection T002, cli wiring T003–T005); no task invents scope.
- **C6** (×2) interface-cli / interface-spec § Surface ↔ feature § Given/When/Then: every scenario step references a defined surface (`-o <file>`, `-o stdin`, reserved tokens, `GLASSFROG_OUTPUT`/`.glassfrogrc`, usage exit 2, render-through-template, absence marker).

## Completeness: 9/9 passed

### Findings

None — every upstream promise has a downstream realization.

### Passed (9/9)

- **K1** spec § Driving Scenarios → feature: all 10 spec scenarios (3 happy / 2 error / 2 edge / 3 validation) have Gherkin equivalents; one additional architecture-informed scenario is marked `Proposed`.
- **K2** (×2) spec § Integration Boundaries → interface files: the feature's only touchpoints are CLI and the Go package surface — both covered (interface-cli, interface-spec); no API/UI/events boundary is implied or missing.
- **K3** plan § Implementation Strategy → tasks: every phase has tasks (P1→T001, P2→T002, P3→T003–T005).
- **K4** plan § System Architecture → tasks § Scope: every component has implementing tasks (render engine, output selection, classifier arm, dispatch arm, seam + read wiring, root usage string).
- **K5** (×2) interface § Surface → feature: every CLI surface (widened values, error conditions) and Go surface (parse/render/error, selection, seam, classifier) has scenario coverage.
- **K6** (×2) spec § User Scenarios → interface: US1 (file), US2 (stdin), US3 (own view) each have interface coverage in the widened value set and error contract.

## Coherence: 4/4 passed

### Resolved findings

**P2 — RESOLVED within PR (commit 5bead74)** | H3: spec.md § Behavioral Accord ↔ plan.md § ADR-2 ↔ interface-cli.md § Interactions / interface-spec.md § Surface
> ~~At analysis time, the plan (ADR-2) and both interface files exposed a capability the spec's behavioral accord did not mention: a user template may **compose a built-in template** by name (e.g. `{{template "roles.full.tmpl" .}}`), because it is parsed into a clone of the built-in set — a benign, documented side-effect present downstream and absent upstream.~~
> **Resolved by 5bead74:** spec.md's Behavioral Accord now states "Because a user template is rendered by the same mechanism as the built-in views, it may also compose a built-in view by name rather than re-describing it." Spec and downstream artifacts now describe the same surface — no open coherence finding remains.

### Passed (H1, H2, H4)

- **H1** Terminology: behavioral terms in spec (`template file`, `stdin`, `reserved names`) and Go-symbol terms in plan/interface/tasks (`UserTemplate`, `TemplateRef`, `Selection`, `UserTemplateError`) are used consistently and map cleanly across the layer boundary — no concept is renamed without an alias.
- **H2** Detail symmetry: spec (behavioral), plan (architectural), tasks (granular) are proportionate on shared topics; no artifact is 3×+ deeper than its neighbour.
- **H4** Phase coverage: tasks' three phases match the plan's three phases in ordering, grouping, and dependencies; no task references a non-existent phase, and no plan phase lacks tasks.

---

## Checklist Correlation

- Checklist recorded **no failures**, so there are no failing-check overlaps. One correlation of note: checklist's **calibration of Principle II (Action Transparency)** flagged that a user template can render *success* output that omits the operation/resource trace the built-ins guarantee. This is operator-directed (the agent supplies the template; `json`/`yaml`/`full` remain available), so checklist passed II on capability-retention. It is **not** a matrix analyze finding (no contradiction, gap, or drift between artifacts — the artifacts agree that the output is operator-defined). Surfaced here only so the link between the II calibration and user-template traceability is visible to the developer. It originally paired with the H3 finding, but H3 is now resolved (5bead74 added the built-in-composition mention to the spec); the Principle-II traceability point remains an operator-directed design choice, not a finding.

## Governance Notes

- **All 16 base relationship checks were evaluable** — the full artifact set is present (spec, plan, 2 interface files, 1 feature file, tasks, checklist). Interface- and scenario-scaled checks (C4, C6, K2, K5, K6) ran ×2 for the two interface files.
- **Checklist context**: loaded — 12/12 constitution checks pass; no overlapping failures to correlate.
