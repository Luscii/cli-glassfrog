# Analyze: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/opaque-failures/diagnostic-normalization.feature, tasks.md
**Checklist context**: checklist.md loaded (14/14 pass, 0 failures)
**Findings**: 16 checks (16 pass, 0 findings)
**Generated**: 2026-06-10

---

## Summary

All 16 cross-artifact checks pass. Consistency: 6/6. Completeness: 6/6. Coherence: 4/4. No contradictions, gaps, or drift found.

---

## Consistency Checks (P0): 6/6 passed

All consistency checks pass:
- **C1** spec § Integration Boundaries ↔ plan § System Architecture: the plan references every named boundary (Request Execution 010, API Error Extraction 015, Argument Dispatch 002, Exit-Code Convention 004, Output-Aware Failure Rendering 032).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture: the `Diagnostic`/`Diagnose` design serves the "collapse into one diagnostic" behavior; no accord behavior is contradicted.
- **C3** spec § Non-Behaviors ↔ plan: the plan architects none of the excluded capabilities — ADR-3 explicitly keeps rendering, exit-code emission, and the cobra usage path out (deferred to 032/004), and adds no retry or body re-parse.
- **C4** plan § Architecture Decisions ↔ interface-spec § Surface: the accord's `Diagnostic`, `Diagnose`, `classifyClientError` delegate, decode→APIError, and next-step splits reflect plan ADR-1/2/3.
- **C5** plan § System Architecture ↔ tasks § Scope: T001/T002 build exactly the plan's components; no task introduces unmentioned work.
- **C6** interface-spec § Surface ↔ feature § Given/When/Then: scenario steps reference only categories, exit codes (3/4/5), and next-steps defined in the interface accord.

## Completeness Checks (P1): 6/6 passed

All completeness checks pass:
- **K1** spec § Driving Scenarios ↔ feature: all 9 driving + 3 validation spec scenarios have Gherkin equivalents (plus 3 architecture-informed additions).
- **K2** spec § Integration Boundaries ↔ interface presence: the boundaries are sibling-capability seams, not external systems needing their own interface file; the single `interface-spec.md` covers the Specification touchpoint (same pattern as 010/015). Justified.
- **K3** plan § Implementation Strategy ↔ tasks: the plan's single phase (two ordered steps) maps to T001 (step 1) and T002 (step 2).
- **K4** plan § System Architecture ↔ tasks § Scope: every plan component (`Diagnostic`, `Diagnose`, `renderDiagnostic`, the `reportClientError` refactor, the `classifyClientError` delegate) has an implementing task.
- **K5** interface-spec § Surface ↔ feature: every surface element (category assignment, exit-code mapping, next-step contract, token-free invariant) has scenario coverage.
- **K6** spec § User Scenarios ↔ interface-spec § Surface: the three user scenarios (cause+category+next-step; one shape; fixed vocabulary) are all served by `Diagnostic{Category, Cause, NextStep}`.

## Coherence Checks (P2): 4/4 passed

All coherence checks pass:
- **H1** Terminology: `Diagnostic`/`Diagnose`/`Category`/`Cause`/`NextStep`/`Outcome` are used consistently across all artifacts; "category" ↔ "`Outcome` taxonomy" is an explicit alias.
- **H2** Detail symmetry: spec↔plan and plan↔tasks are proportionate; the concise 2-task decomposition matches the plan's single-phase, two-step strategy.
- **H3** Scope alignment (spec + interface + tasks): all three describe the same capability; the envelope-mapping and cobra-usage deferral to 032 is explicitly stated (spec Non-Behaviors + plan ADR-3), not a silent drop.
- **H4** Phase coverage (plan + tasks): the single plan phase has corresponding tasks; tasks reference no phantom phase.

---

## Checklist Correlation

No overlapping findings (both checklist and analyze are clean). One corroboration: checklist's governance note that machine-parseable *emission* is deferred to 032 is confirmed by analyze C3 and H3 — the deferral is stated consistently across spec, plan, and interface, so it is a deliberate, coherent boundary rather than a gap or contradiction.

---

## Governance Notes

- No checks skipped — all artifact relationships were evaluable (full artifact set present).
