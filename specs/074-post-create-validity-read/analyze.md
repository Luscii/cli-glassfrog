# Analyze: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md, features/success-reported-for-a-dead-proposal/post-create-validity-read.feature
**Checklist context**: `checklist.md` present and parsed (round 2 — 18 checks, 0 P0 fail, 1 P2 consideration)
**Checks**: 19 (19 pass, 0 fail)
**Generated**: 2026-08-08 (round 2 — re-derived after the machine-format, coverage, example-fidelity, and citation fixes)

Full artifact set present — every relationship in the matrix is checkable. C4 and C6 scale across the two interface files.

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 8 | 8 | 0 |
| Completeness | P1 | 7 | 7 | 0 |
| Coherence | P2 | 4 | 4 | 0 |
| **Total** | — | **19** | **19** | **0** |

---

## Changes Since Previous Run

**Previous** (round 1): 2 P0, 1 P1, 1 P2
**Current** (round 2): 0 P0, 0 P1, 0 P2

**Resolved**:

- ~~**C2** | spec.md § Behavioral Accord ↔ plan.md ADR-5 — the accord promised the verdict was readable from the emitted document without human text; ADR-5 put the provenance and the unavailability reason outside that document as prose, and in the unavailable case the document carried no verdict at all.~~ → fixed by **widening the design, not narrowing the accord**. ADR-5 gained an amendment making the advisory format-aware (032's precedent, previously applied only to the failure path); the accord now claims that everything the command reports about the verdict is machine-readable, and the design delivers exactly that. Re-checked: the accord, ADR-5, `interface-cli.md`'s four-case table, `interface-spec.md`'s `verdictSource` type, and the validation scenario now make one claim between them. Correlates with checklist CH-4, also resolved.
- ~~**C4** | plan.md ADR-4 ↔ interface-cli.md § compact format — four worked examples showed an elided id the delegated shared template cannot produce, and T003 requires that output to be asserted.~~ → fixed; all four examples now carry the full id, with an explicit note that they are not eliding it. The JSON example keeps its existing "abbreviated" marker, so both example families now state which they are.
- ~~**K5** | interface-cli.md § Surface ↔ feature file — the `compact` format and the user-template path were defined surfaces with no scenario.~~ → fixed; *"The compact line carries the validity token"* and *"A user template written before the verdict still renders"* added, plus *"An unobtainable verdict is machine-readable in a machine format"* for the newly-widened machine-format contract. T005/T006 criteria extended to match.
- ~~**H1** | Terminology — spec.md cited `LEARNINGS F6` for a fact whose canonical home became `CSG-2` when 072 landed, while plan.md cited `CSG-2`.~~ → fixed; all citations of the accepted-but-invalid shape now read `CSG-2` in both artifacts (six sites — three in each, one of which is plan.md's Inputs line). The `LEARNINGS` references that remain point at **S5** and **F8**, neither of which moved — verified by grep: zero `F6` mentions survive, and every `CSG-2` mention resolves to the landed record.

**Scenario count**: 15 → 19 (15 executed, 4 held). The held set is unchanged in size and composition.

---

## Consistency (P0 — contradiction)

### All consistency checks pass (8/8)

- **C1** | spec.md § Integration Boundaries ↔ plan.md § System Architecture — PASS. Every boundary the spec names has an architectural counterpart: `getProposal` as the read-back target, `createProposal` unchanged, `listProposals` named specifically as the surface that does *not* carry the verdict, rate limits as a doubled-cost consequence, and 055/056/063/072/Invalid-Create Outcome each placed. The plan adds `internal/apiclient` as an explicitly untouched component, which is a narrowing, not a contradiction.
- **C3** | spec.md § Non-Behaviors ↔ plan.md § System Architecture — PASS. All eleven non-behaviors are honored by the design: no local derivation (ADR-3 disproves the proxies), no outcome or exit-code change (ADR-2), no contract claim (ADR-3 field comments), no missing-verdict-as-favourable (tri-state), no id withholding (ADR-2), no read-back after a failed create (the flow diagram), no opt-out (no flag added), no polling (ADR-1 single attempt), no extension to sibling writes (§ What This Plan Does Not Cover), no verdict in the list, no suppression of what `get` already returns.
- **C2** | spec.md § Behavioral Accord ↔ plan.md § Architecture Decisions — PASS (round 2). Every accord bullet has an architectural counterpart that serves rather than contradicts it, including the two machine-format bullets, which ADR-5's amendment now satisfies through the format-aware advisory.
- **C4** (interface-cli.md) | plan.md § Architecture Decisions ↔ interface-cli.md § Surface — PASS (round 2). The compact examples now show what the delegated template emits, and the stderr section carries both renderings of the advisory with the four-case reading table. ADR-5's amendment and the interface's structured-advisory shape state the same contract.
- **C4** (interface-spec.md) | plan.md § Architecture Decisions ↔ interface-spec.md § Surface — PASS. Every ADR has its Go-surface counterpart: ADR-3's tri-state as `Valid *bool` with the reasoning in the field comment, ADR-4's delegation as the embedded view plus the `{{template …}}` invocation, ADR-6's local decode as the id-lift contract, ADR-2's isolation as a helper signature that cannot return an error, and ADR-5's amendment as the `verdictSource` type whose `omitempty` tags encode "absent means not applicable."
- **C5** | plan.md § System Architecture ↔ tasks.md § Task Scope — PASS. No task builds something the plan doesn't name. T001↔model, T002/T003↔render, T004/T005/T006↔orchestration, T007↔the plan's testing strategy. `internal/apiclient` correctly has no task.
- **C6** (both interface files) | interface-*.md § Surface ↔ feature file Given/When/Then — PASS. Every step references a defined surface: the four validity labels, the three structured field names, the `draft_with_conflicts` status (a declared enum value), the exhausted-budget reason phrasing, and the id-undeterminable reason all appear in `interface-cli.md`'s own tables. No step invents a field or an endpoint.

---

## Completeness (P1 — gap)

### All completeness checks pass (7/7)

- **K5** (interface-cli.md) | interface-cli.md § Surface → feature file — PASS (round 2). Every user-visible surface now has a scenario: the four verdict states, the `full` block, the `compact` line, both machine formats including the structured advisory, the user-template compatibility guarantee, the stderr advisory in both renderings, and the exit-code invariant.
- **K1** | spec.md § Driving Scenarios → feature file — PASS. All nine driving scenarios have Gherkin equivalents: 3 happy (valid, invalid, machine), 2 error (unreachable read-back, rejected create), 4 edge (no verdict, valid-without-transitions, status disagreement, exhausted budget). Each carries a `# Source:` comment naming its spec scenario.
- **K2** | spec.md § Integration Boundaries → interface file presence — PASS. The CLI boundary has `interface-cli.md`; the Go-surface boundary has `interface-spec.md`. The Glassfrog API boundaries are *consumed*, not designed, and both files document the operations invoked — consistent with siblings 056 and 057, which shipped `interface-cli.md` alone against the same consumed boundaries.
- **K3** | plan.md § Implementation Strategy → tasks.md — PASS. All three phases decompose: Phase 1→T001, Phase 2→T002/T003, Phase 3→T004–T007.
- **K4** | plan.md § System Architecture components → tasks.md § Task Scope — PASS. Three of four components have implementing tasks; the fourth (`internal/apiclient`) carries an explicit "untouched" justification in the plan, which K4 counts as a realization.
- **K5** (interface-spec.md) | Go surface → feature file — PASS with note. The Go surface is internal, so scenario coverage is the wrong instrument; its coverage lives in T001–T004's acceptance criteria, which name every symbol the accord pins. Recorded as a pass rather than a skip because the coverage exists — it is simply at the unit layer by design.
- **K6** | spec.md § User Scenarios → interface-*.md — PASS. US1 maps to the human render, US2 to the machine format, US3 to the unavailable state and its advisory. Each of the three has a defined surface.

---

## Coherence (P2 — drift)

### All coherence checks pass (4/4)

- **H1** | Terminology — PASS (round 2). The accepted-but-invalid shape is cited as `CSG-2` everywhere it appears in spec.md and plan.md; zero `F6` references survive. The `LEARNINGS` citations that remain point at S5 (the change-type enum landing) and F8 (the lagging aggregate) — neither of which moved when 072 landed, so citing LEARNINGS for them is correct rather than stale. The verdict vocabulary (`valid` / `not valid` / `not reported` / `unavailable`) is identical across the spec, the plan, both interface accords, the feature file, and tasks.
- **H2** | Detail symmetry across adjacent pairs — PASS. spec (189 lines) → plan (six ADRs plus a design section) → interface (two accords) → tasks (seven tasks with per-task criteria) escalates in specificity without any artifact being dramatically thinner or thicker than its neighbour warrants.
- **H3** | Scope alignment across spec + interface + tasks — PASS. Nothing is silently added or dropped. Every "deliberately absent" item in `interface-cli.md` § Interactions traces to a spec non-behavior; T003's byte-identical guard traces to plan Risk 3; the exchange-count contract traces to ADR-1's consequences.
- **H4** | Phase coverage — PASS. The task graph reproduces the plan's structure, not just its phase names: strict sequencing, Phase 3 depending on both predecessors, and the `[P]` marking on T005/T006 reflecting the plan's own observation that the two dispatch arms are independent. T004's finer dependency (T001 only) is a narrowing within the phase, not a reordering of it.

---

## Checklist Correlation

Round 2 has no open findings on either side. The round-1 correlations are recorded here because they explain why the fixes landed where they did:

| Round-1 analyze finding | Round-1 checklist finding | Relationship | Round-2 state |
|---|---|---|---|
| C2 (spec accord ↔ ADR-5) | CH-4 (Principle II) | **Same defect, two angles** — a constitution violation and a spec↔plan contradiction, found independently. | Both resolved by one change: the format-aware advisory. The fix was chosen by asking which artifact held the *intent* — the accord did, so the design moved. |
| K5 (compact + user-template surfaces) | CH-9 (Principle IV) | **Adjacent, not identical** — coverage gaps against different sources (an accord bullet vs two interface surfaces), landing in the same feature file. | Both resolved; four scenarios added, one of which also covers the widened machine-format contract. |
| H1 (F6 vs CSG-2) | — | No checklist counterpart — citation provenance has no constitution principle behind it. | Resolved across both artifacts. |

One round-1 note is worth preserving: checklist's third resolution option for CH-4 (a composed key in the machine path) would have created a *new* C4-class inconsistency with ADR-5's verbatim commitment. The chosen fix avoided that entirely by reshaping only the CLI's own diagnostic, which no artifact claims is verbatim.

---

## Governance Notes

**Skipped checks**: none. The full artifact set is present, so all 16 base check types ran, with C4, C6, and K5 scaled across the two interface files.

**Round-2 verification method**: the four round-1 findings were re-checked against the edited artifacts rather than assumed fixed — the citation sweep by grep (zero `F6`, six `CSG-2`), the scenario count by grep (19 scenarios, 4 `@validation`), and the machine-format claim by reading the accord, ADR-5's amendment, the interface's four-case table, and the validation scenario as one set. The remaining 15 checks were re-evaluated because five artifacts changed; none regressed.
