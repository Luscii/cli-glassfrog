# Analyze: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/unguided-change-construction/pre-assembly-grammar-consultation.feature, features/proposal-circle-not-choosable/pre-assembly-routing-application.feature
**Checklist context**: loaded — 24/24 constitution checks pass, 0 findings
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-08-22 (round 2 — re-derived after the K5 remedy landed: two scenarios added to the routing feature file, tasks.md T002 inventory and assertion minimums refreshed)

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 6 | 6 | 0 |
| Completeness (P1) | 6 | 6 | 0 |
| Coherence (P2) | 4 | 4 | 0 |

**No findings.** The round-1 K5 gap is closed; no contradictions, no drift.

---

## Consistency (P0) — 6/6 pass

- **C1 PASS** (spec § Integration Boundaries ↔ plan § System Architecture): the plan's component map covers every boundary the spec names — the three upstream artifacts consumed, the host path's in-place widening, the tension-processing handoff, the write-gate non-change, and self-containment. The two spec boundaries with no plan component (Identifier Prompt downstream, grammar command's no-tracking rule) are consume-side constraints the plan restates rather than builds — compatible.
- **C2 PASS** (spec § Behavioral Accord ↔ plan § System Architecture): the nine-step flow realizes every accord group in the accord's own order (route → consult → assemble → match, all before the write); the two-phase returns realize the practitioner-decides bullets. Scrutinized specifically: the plan's stop-and-return on a decision point is not a contradiction of the spec's never-delay non-behavior — the spec's own accord makes those the practitioner's decisions ("the practitioner decides whether to proceed", "may still direct the draft onto the anchor they chose"), and the plan's fence ("direction given is always acted on") matches the spec's ("withholds nothing"). The interface records this as a deliberate deviation ("awaiting direction" is neither success nor failure), so a reviewer cannot mistake it for error semantics.
- **C3 PASS** (spec § Non-Behaviors ↔ plan § System Architecture): the plan architects none of the eight exclusions — no identifier ask, no typed validator (recognition is prompt-level named-match), no content restatement (no-copy is a task acceptance criterion and a held validation scenario), no tension write (the capture gap is handed onward), no record/rendering edits, no new proposal transitions, no acceptance guarantee.
- **C4 PASS** (plan § Architecture Decisions ↔ interface § Surface): the three `action` spellings are byte-identical across plan, interface, and both feature files (verified by extraction: `surfaced-routing-mismatch` ×4, `named-anchors` ×7, `surfaced-dead-shape` ×5); the eight-leaf fence, the consultation element's three parts, and the nine-step order match; tasks T001 restates the same order (dispatch-order sites agree).
- **C5 PASS** (plan § Implementation Strategy ↔ tasks § Task Scope): T001+T002 = plan Phase 1 (artifact wiring + BDD suite), T003 = plan Phase 2 (the three sweep edits, item-for-item with ADR-4). No task builds anything the plan does not describe; the same-PR/separable-commit guidance matches the plan's Phase 2 note. The T002 refresh (14 runnable, two added assertion minimums) stays within the plan's testing strategy — no new component entered tasks.
- **C6 PASS** (interface § Surface ↔ feature Given/When/Then): every surface a scenario step references exists in the accord — the action values, the `consultation` element and its parts, the eight leaves, the gated-set posture, the registry annotation contract. The dead-shape scenarios' concrete instances (CSG-1's refused wrapper form, CSG-2's self-target) match the accord's match-part definition (the fact's handle with shape and symptom as the rendering states them) and the records' actual content.

## Completeness (P1) — 5/6 pass

- **K1 PASS** (spec § Driving Scenarios → features): all 15 spec scenarios (10 driving + 5 validation) have Gherkin equivalents with verbatim `# Source:` titles (title-diff empty, re-verified mechanically this round).
- **K2 PASS** (spec § Integration Boundaries → interface files): one specification touchpoint, one interface-spec.md; the boundaries that are consumed rather than designed (grammar command, write gate, routing record) are explicitly assigned to their owning accords in Consistency Notes — justified absence stated.
- **K3 PASS** (plan phases → tasks): both phases decomposed, dependencies expressed (T002/T003 depend on T001), same-PR guidance carried.
- **K4 PASS** (plan components → tasks): each of the three plugin artifacts, the BDD suite, and each sweep edit has an implementing task; the zero-guard-code claim is pinned as a T001 acceptance criterion rather than left implicit.
- **K5 PASS** (interface § Surface → features): every accord surface now has scenario coverage. The two round-1 gaps are closed: the root-circle decline is covered by "A root circle's missing parent is declined, not resolved" (asserting the routing part carries the record's decline and no target is invented — both step surfaces exist verbatim in the interface's routing part and Error Communication row), and the widened frontmatter descriptions by "Both artifact descriptions state the routed entry" (asserting the routed entry is stated and every prior boundary sentence retained, matching the Invocation table). Both are `@wip` runnable and carried in T002's step-assertion minimums, so the Builder holds them executable.
- **K6 PASS** (spec § User Scenarios → interface): US1 → the workflow/match/action contracts; US2 → the routing determination and its returns; US3 → the consultation element on every action path.

## Coherence (P2) — 4/4 pass

- **H1 PASS** (terminology): the load-bearing terms — target circle, eligible anchor, handed-in vs settled anchor, recorded dead shape, consultation element, two-phase return, awaiting direction — are used with one meaning across all six artifacts; "gate" consistently names the whole pre-assembly step set.
- **H2 PASS** (detail symmetry): spec (~230 lines) ↔ plan (~150) ↔ interface (~160) ↔ tasks (~90) carry proportionate depth; no shared topic is 3x+ lopsided.
- **H3 PASS** (scope alignment): spec, interface, and tasks describe the same capability set — wiring, two consultations, recognition, reporting, sweep. The sweep appears in all three (spec Assumption → plan ADR-4 → T003); nothing is added or dropped silently.
- **H4 PASS** (phase coverage): tasks' two phases mirror the plan's structurally — same names, same ordering, same dependency (Phase 2 on Phase 1), and the same-PR rider carried in both.

## Checklist Correlation

Checklist passed 24/24 in the same round — no failing sections on either side to correlate.

## Governance Notes

None — all six artifact types present; no checks skipped.

## Improvement Summary

Previous: 0 P0, 1 P1, 0 P2. Current: 0 P0, 0 P1, 0 P2. Resolved: K5 (interface→scenario coverage) — closed by the two architecture-informed scenarios added to pre-assembly-routing-application.feature and the tasks.md T002 refresh that counts and asserts them.
