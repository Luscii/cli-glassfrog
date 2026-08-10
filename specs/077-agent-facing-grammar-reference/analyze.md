# Analyze: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, tasks.md, features/unguided-change-construction/agent-facing-grammar-reference.feature
**Checklist context**: loaded (run 3 — 23 checks, 0 failures)
**Checks**: 16 (16 pass, 0 findings)
**Generated**: 2026-08-09 (run 3 — re-derived on the rebased base after 076 merged)

---

## Summary

| Category | Severity | Pass | Findings |
|---|---|---|---|
| Consistency | P0 | 6 | 0 |
| Completeness | P1 | 6 | 0 |
| Coherence | P2 | 4 | 0 |

All 16 checks pass. The artifacts tell the same story, every upstream promise has a downstream realization, and no scope or terminology drift remains.

---

## Changes Since Previous Run

**Previous (run 1)**: 0 P0, 1 P1, 1 P2 (2 findings)
**Current (run 2)**: 0 P0, 0 P1, 0 P2 (0 findings)

**Resolved**:

- ~~P1 | **K5** interface-cli.md § "The human formats (`full` / `compact`)" → feature file: three defined surfaces had no scenario coverage — `compact` entirely, `full` with residue present, and the exit-1 decode path.~~ → **fixed**. Two scenarios were added covering `full` with both facts present (placements, the nesting rule stated once, each fact's four fields, visible contract-vs-observation separation) and `compact`'s condensed form; both are mapped to T004. The exit-1 decode path is now recorded in tasks.md § Scenario Disposition as a deliberate no-scenario decision — reaching it requires a build the drift guard makes impossible — with T002's unit test named as its coverage. A justified absence stated where the Builder looks counts as a realization.

- ~~P2 | **H3** Scope alignment (spec.md + interface-cli.md + tasks.md): the write-gate hook edit appeared in plan, interface, feature file, and tasks, but spec.md named five integration boundaries and the operating surface was not among them.~~ → **fixed**. spec.md § Integration Boundaries gained the operating-surface write gate, stated at spec altitude: the gate must recognize this read as a read so consulting the grammar never asks for confirmation, and a placement inside the gate's fail-close scope requires teaching the gate in the same change. The downstream artifacts' treatment is unchanged — they now realize a boundary the spec names.

---

**Run 4 (review round 1)**: Copilot found a plan↔interface field-level contradiction that **C4 missed in run 3** — `plan.md` § System Architecture listed the artifact's fact entries as `(id, shape, wrong shape, disposition, symptom, provenance)` while `interface-cli.md` pins `(id, title, shape, disposition, symptom, provenance)`. C4 was evaluated at the level of "does the accord reflect the ADRs' technology and pattern choices" and passed on that reading; it did not compare the field lists the two artifacts each spell out, which is squarely within its assertion. Recorded rather than quietly corrected: the run-3 pass on C4 was optimistic, not earned. The same drift had a spec-side half (`spec.md` § Content promised a "wrong shape" element and enumerated two of the record's three disposition values); both were corrected in the review round, along with a FEATURE-MODEL phrasing that read as "never errors". Post-fix, C4 and C6 hold on a field-by-field comparison, and the disposition vocabulary is stated open-endedly in the spec and verbatim-closed in the interface — the two altitudes that are correct for each.

**Run 3 (rebase re-derivation)**: 076 merged to main mid-shape, changing files these artifacts cite. Re-verified against the landed tree: the drafting skill still references neither reference file (so the spec's untouched-consumption claim holds), the record's per-fact field structure is unchanged (so the interface's upstream-contract claim holds), and the write-safety guardrail's real anchors were read rather than recalled — correcting plan/interface/tasks to name `expectedProposalSurface` as the CI forcing function alongside the script's `PROPOSAL_READS`. All 16 checks re-evaluated on the corrected artifacts; no finding was introduced or resolved by the rebase.

---

## Consistency (P0) — 6/6 pass

- ✅ **C1** spec.md § Integration Boundaries ↔ plan.md § System Architecture — every named boundary has a corresponding component, including the operating-surface gate, which maps to plan § Integration Design ("Command registration") and T005's two guardrail edits.
- ✅ **C2** spec.md § Behavioral Accord ↔ plan.md § System Architecture — the architecture serves all three accord groups: Content via embed + accessor, Conduct via the client-less command, Sync via the regenerate-and-compare guard.
- ✅ **C3** spec.md § Non-Behaviors ↔ plan.md § System Architecture — the plan architects nothing the spec excludes: no input path for a change set, the 072 record left as the single hand-edited source, and the drafting skill's workflow untouched (the gate edit changes command *classification*, not the consultation step — a distinction the spec's new boundary makes explicit rather than implicit).
- ✅ **C4** plan.md § Architecture Decisions ↔ interface-cli.md § Surface — the accord reflects ADR-1, ADR-3, ADR-4, and ADR-5.
- ✅ **C5** plan.md § System Architecture ↔ tasks.md § Task Scope — every task builds a component the plan names; no task introduces an unmentioned one.
- ✅ **C6** interface-cli.md § Surface ↔ feature file Given/When/Then — every step references a defined surface. Re-verified for the two added scenarios: `--output full` and `--output compact` are both format tokens the accord defines, and the asserted content (placement class, nesting rule with wrapper types, per-fact fields, one-line summaries) matches § "The human formats" clause for clause.

---

## Completeness (P1) — 6/6 pass

- ✅ **K1** spec.md § Driving Scenarios → feature file — all 11 spec scenarios (8 driving + 3 validation) have Gherkin equivalents; the file adds 4 architecture-informed scenarios, each marked as proposed with its source.
- ✅ **K2** spec.md § Integration Boundaries → interface file presence — the external surfaces have `interface-cli.md`; the artifact contract is pinned there rather than in a sibling file (stated in § Consistency Notes), and 072 owns the record's own structural contract.
- ✅ **K5** interface-cli.md § Surface → feature file — every defined surface now has coverage: both structured formats, both human formats (with and without residue), the usage refusal, the interactions (credential-free, offline, gate, determinism), and the embedded artifact via the drift scenario. The one uncovered path carries a recorded justification naming its unit test.
- ✅ **K3** plan.md § Implementation Strategy → tasks.md — both phases decomposed.
- ✅ **K4** plan.md § System Architecture components → tasks.md § Task Scope — generator, artifact, grammar package, render integration, command, and drift guard each have an implementing task.
- ✅ **K6** spec.md § User Scenarios → interface-cli.md — US1 maps to § Surface, US2 to the provenance tokens, US3 to § Interactions.

---

## Coherence (P2) — 4/4 pass

- ✅ **H1** Terminology — *empirical residue*, *provenance*, *disposition*, *placement*, and *change-type vocabulary* hold their names across all five artifacts. Re-verified for the added scenarios: they use `placement class`, `nesting rule`, and the four per-fact field names as the interface spells them, introducing no synonym.
- ✅ **H3** Scope alignment — the capability set now matches across spec, interface, and tasks. The operating-surface gate appears in all three altitudes: named as a boundary in the spec, specified as conduct in the interface, and implemented in T005.
- ✅ **H2** Detail symmetry — spec↔plan and plan↔tasks remain proportionate; the plan's deeper ADR-1 treatment is justification depth for a one-line spec accord, not asymmetry on a shared topic.
- ✅ **H4** Phase coverage — tasks.md mirrors the plan's two phases by name, ordering, and dependency direction.

---

## Checklist Correlation

- Run 1's K5 correlated with checklist run 1's first P0 (the same coverage gap seen horizontally and vertically). Both are resolved by the same remedy, and both runs now report clean — the correlation held through the fix, which is the expected outcome for a single underlying defect.
- Checklist run 2 reports 0 failures; no analyze finding contradicts a checklist result.

---

## Governance Notes

- No checks were skipped — the full artifact set was present.
- Check count is 16 base evaluations: one interface file and one feature file mean no scaling multipliers applied.
