# Analyze: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md + interface-spec.md (2), features/change-targets-unidentifiable/*.feature (3), tasks.md
**Checklist context**: checklist.md present and parsed (round 2 — 17 checks, 0 fail)
**Checks**: 21 evaluations (21 pass, 0 fail) — 16 base types scaled by 2 interface files and 3 feature files
**Generated**: 2026-08-08 (round 2 — re-derived after the round-1 finding and the probe-driven amendments)

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 8 | 8 | 0 |
| Completeness | P1 | 9 | 9 | 0 |
| Coherence | P2 | 4 | 4 | 0 |
| **Total** | | **21** | **21** | **0** |

**Resolved since round 1**:

- ~~**K5 (P1)**: three interface-cli.md surfaces had no scenario exercising them — the `actors` list read, the user-template surface, and compact-format rendering~~ → fixed. The actor directory gained "The actor directory carries a legacy number for every actor" (compact); the template surface gained "An operator template sees the membership number the built-in render omits"; compact rendering gained a `Scenario Outline` over the role family. All three land in legacy-identifier-request.feature.

**Round-2 re-run was not optional.** Round 1 warned that the consistency and coherence passes were green *on the strength of the artifacts agreeing with each other*, and that a probe-driven edit would invalidate that. It did: eight artifacts changed, one accord bullet was falsified, and a third feature file was created. Every check below was re-evaluated against the amended set.

---

## Consistency (P0)

All 8 pass.

**C1 — spec Integration Boundaries ↔ plan System Architecture** — PASS
Five boundaries, five architectural homes, unchanged by the amendments.

**C2 — spec Behavioral Accord ↔ plan System Architecture** — PASS
Re-checked against the amended Absence group, which now has **two** embed bullets (one per direction) instead of one blanket rule. plan § Render Design matches bullet-for-bullet: `role.full`'s embed groups get the note, `me.full`'s embedded roles render the number and get none. The round-1 correlation note is discharged — spec and plan agreed with each other then and still do, but they now also agree with observed behavior.

**C3 — spec Non-Behaviors ↔ plan System Architecture** — PASS
No new conflict introduced by the decode tolerance. Non-behavior 8 forbids the CLI *validating, normalizing, range-checking, or judging* the number; accepting two JSON spellings of the same integer is a decode concern, not a judgment about the value — and the accepted value set is unchanged, which is what the non-behavior protects. Non-behavior 4 (no synthesis) likewise holds: nothing is produced that the API did not send.

**C4 — plan Architecture Decisions ↔ interface Surface** (2 evaluations) — PASS
interface-cli.md reflects ADR-1/2/3 including the amended decode tolerance and the `TreeNode` grounding note. interface-spec.md reflects ADR-4 **and** its new constraint that the guard anchors the parameter rather than the response schema — plan ADR-4 and interface-spec.md invariant 5 state the same reason in the same terms.

**C5 — plan System Architecture ↔ tasks Task Scope** (1 evaluation) — PASS
No task builds anything the plan does not name. T005's scope grew by the observed-exception register, which plan ADR-4 now describes.

**C6 — interface Surface ↔ feature Given/When/Then** (2 evaluations) — PASS
No step names an undefined surface, across all three feature files. Specifically re-checked: the embed-note step wording ("this read carries no legacy number for them") matches interface-cli.md's amended note text; the two invariant-5 guard scenarios match interface-spec.md's register semantics including the stale-entry rule; and the decode-spelling steps use concrete values (`14062695`, `"14062695"`, `"not-a-number"`) rather than restating the tolerance rule the code keys on.

---

## Completeness (P1)

All 9 pass.

**K1 — spec Driving Scenarios ↔ feature files** (1 evaluation) — PASS
All 16 spec scenarios (11 driving + 5 validation) map to Gherkin. The spec gained an eleventh driving scenario during round 2 — "A read whose embedded resources do carry the numeric identifier renders it" — so the probe finding (W2) is a spec-level behavior with a spec-level scenario, not a Gherkin-only addition. Two spec scenarios map to two Gherkin scenarios each (the structured/human splits), which satisfies the check: one upstream promise, two downstream realizations. The 5 remaining Gherkin scenarios beyond the spec's set carry `Proposed:` source comments per the standard: three from interface surfaces (actors list, user template, compact rendering) and two from plan ADR-3's decode tolerance and its rejection path. Verified mechanically: 28 Gherkin scenario blocks (12 + 10 + 6), expanding to 30 executable instances via the two Scenario Outlines.

**K2 — spec Integration Boundaries ↔ interface file presence** (2 evaluations) — PASS
Four boundaries covered by the two interface files; the fifth is labelled *(downstream consumer)* in spec.md and deferred in plan § What This Plan Does Not Cover.

**K3 — plan Phases ↔ tasks entries** (1 evaluation) — PASS
Phase 1 → T001, T002. Phase 2 → T003, T004. Phase 3 → T005.

**K4 — plan Components ↔ tasks Scope** (1 evaluation) — PASS
Request side → T002; Model side → T001; structured Output side → T002; human Output side → T003, T004; Guard side → T005.

**K5 — interface Surface ↔ feature scenario coverage** (2 evaluations) — PASS *(was the round-1 failure)*
interface-cli.md: every surface now has at least one scenario. The flag, The request, and Structured output → multiple; The field's decode tolerance → the two-spelling outline plus the non-numeric rejection scenario; The field's `TreeNode` note → the every-depth tree scenario; the twelve per-template rows → the role-family compact outline, the actor directory compact scenario, the `actor.full` agent-backed scenario, the `me.full` actor/organization scenario, and the `me.full` embedded-roles scenario; User templates → the membership-number scenario.
interface-spec.md: all five invariants covered, including invariant 5 in both directions (a new mapped operation with an incapable schema fails; a stale register entry fails).
*Coverage thinness noted, not failed*: `actors.full` and `me.compact` have no scenario naming them specifically. Both are single templates inside a read and a format family that are otherwise covered, and both are owned by tasks whose criteria require render tests across the requested × present/absent matrix — a different situation in kind from round 1, where a whole read, a whole feature surface, and a whole format family had zero coverage.

**K6 — spec User Scenarios ↔ interface Surface** (2 evaluations) — PASS
All four user scenarios have interface coverage; US4's coverage now spans both files (help-text constant in interface-cli.md, its mechanical anchor in interface-spec.md invariant 4).

---

## Coherence (P2)

All 4 pass.

**H1 — Terminology** — PASS
The round-1 alias chain holds. Three terms introduced by the amendments are used consistently wherever they appear: *observed-exception register* (plan ADR-4 → interface-spec.md invariant 5 and its design note → tasks T005 → guard.feature), *decode tolerance* (plan ADR-3 → interface-cli.md § The field and § Error Communication → tasks T001/T003 → absence.feature), and *per-read observed fact* (spec Absence accord → plan Render Design → interface-cli.md § shared idioms). No concept acquired a second name.

**H2 — Detail symmetry** — PASS
No pair shows 3x asymmetry. The invariant-5 design note is the deepest new passage and sits in interface-spec.md, which is where a structural contract's reasoning belongs; plan ADR-4 states the same constraint in two sentences and points at it rather than repeating it.

**H3 — Scope alignment (spec + interface + tasks)** — PASS
Two behaviors were *added* during round 2 and both are present in all three artifacts, so neither is a silent introduction: the per-read embed distinction (spec Absence accord ↔ interface-cli.md `me.full` row and § shared idioms ↔ tasks T004) and the decode tolerance (spec § Clarifications and § Assumptions ↔ interface-cli.md § The field ↔ tasks T001/T003). The round-1 embed-note exemption remains disclosed inline. Nothing was dropped: the tree read stayed in scope and its grounding is stated in all three.

**H4 — Phase coverage (plan + tasks)** — PASS
Structure matches by ordering, grouping, and dependencies. tasks.md now additionally asserts **phase locality of scenario ownership**, which the plan's phase boundaries imply but did not previously state anywhere checkable.

---

## Checklist Correlation

Round 2 has no overlaps to report, because checklist round 2 has no findings. The round-1 correlations resolved as follows:

| Round-1 checklist finding | Outcome |
|---|---|
| **C2 (P0)** — tree read not contract-supported | Resolved in the keep direction by live probe (W1). The remediation touched eight artifacts as round 1 predicted, and this re-run confirms consistency and coherence survived it. |
| **C8 (P0)** — no lint gate on T002–T005 | Resolved. Never had an analyze overlap — K3/K4 check that phases and components have tasks, not what those tasks require. |
| **X1 (P1)** — scenario owned by two tasks | Resolved. Its remedy and analyze's K5 remedy shared an edit pass over tasks.md and the feature files, as round 1 anticipated. |

---

## Governance Notes

**Skipped checks**: none. All artifact types present; the full 16-type matrix ran at 21 scaled evaluations.

**Scaling applied**: C4, C6, K2, K5, K6 ran once per interface file (2 each); C6, K1, K5 evaluated scenarios across all three feature files. Base 16 → 21 evaluations. The feature-file count rose from 2 to 3 without changing the evaluation count, since the matrix scales those checks by interface file and evaluates scenarios in aggregate.

**File size**: legacy-identifier-request.feature is at 12 scenarios and legacy-identifier-absence.feature at 10 — both within the ~12 guideline. The behavior was split across two files at authoring time rather than shipping one 19-scenario file; the split is by concern (surfacing versus absence and degradation), so each file's Feature narrative states a coherent problem.

**Durability of this round's green**: unlike round 1, these results do not rest on unobserved contract claims. Five of the six reads' runtime behavior is now observed (W1–W5). The one contract-only claim remaining — agent-backed nullability (W6) — is marked `[ASSUMED]` in spec.md and affects one scenario's reason text, not any cross-artifact relationship.
