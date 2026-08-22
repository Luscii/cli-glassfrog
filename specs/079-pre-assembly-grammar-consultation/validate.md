# Validate: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Round**: 1 of 3
**Date**: 2026-08-22
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-spec.md, `features/unguided-change-construction/pre-assembly-grammar-consultation.feature`, `features/proposal-circle-not-choosable/pre-assembly-routing-application.feature`, PROJECT.md
**Implementation files**: 6 — three edited operating-surface artifacts (`plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt`), one new production helper (`internal/build/preassemblygate.go`), two new BDD suites (`internal/build/pre_assembly_grammar_consultation_bdd_test.go`, `internal/build/pre_assembly_routing_application_bdd_test.go`). Plus ADR-4 sweep edits in `features/proposal-circle-not-choosable/circle-routing-guard.feature`, `specs/067-proposal-drafting-path/{spec,validate}.md`, `specs/073-circle-routing-rule/validate.md`. No Go CLI code added, consistent with the plan's declarative-change architecture.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (5 of 5) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. One advisory observation (non-blocking) recorded below.

---

## Driving Scenario Coverage

**Status**: Pass (10 of 10 driving scenarios covered)

Every driving scenario is concretized as a runnable scenario in one of the two feature files and passes under the `~@wip` filter. Because this is a declarative operating-surface feature, the "code path" for each scenario is the instruction text the drafting agent executes, held truthful by the content-inspection suites — the same tracing convention 067 and 073 established.

| Spec driving scenario | Status | Implementation |
|---|---|---|
| The gate runs in order before the write | ✓ Covered | SKILL.md § The workflow, nine numbered steps + "run in order on every draft" preamble; pinned by `whenStepsCompared`/`thenGateOrderHolds` against the `DraftingGateOrder` anchor |
| The target circle and its eligible anchors are established first | ✓ Covered | SKILL.md step 1 (Route); agent § Workflow routing clause; pinned by `whenRoutingReadsInOrder`/`thenNamedAnchorsChoosingNone` |
| A recognized dead shape is surfaced before the write | ✓ Covered | SKILL.md step 7 (Match); agent `consultation.match` + `surfaced-dead-shape`; pinned by `thenSurfacesHandleShapeSymptom` |
| The grammar read fails | ✓ Covered | SKILL.md step 5 continuation clause; agent defensive entry "The grammar read fails"; pinned by `thenRecordsNotConsulted`/`thenDraftingNotWithheld` |
| A routing read fails part-way | ✓ Covered | Agent defensive entry "A routing read fails part-way"; `consultation.routing` incomplete-walk note; pinned by `thenIncompleteInConsultation` |
| A handed-in anchor routes the change elsewhere | ✓ Covered | SKILL.md step 1 mismatch clause + § The relay; agent `surfaced-routing-mismatch`; pinned by `thenMismatchReportedNotEnforced` |
| An anchor-dependent dead shape is recognized | ✓ Covered | SKILL.md step 7 "with the routing answer in hand … rests on both the change's target and the circle the proposal would be anchored in"; pinned by `thenRecognitionRestsOnBoth` |
| The practitioner proceeds past a surfaced dead shape | ✓ Covered | Agent defensive entry "Direction is present" (create runs unchanged, change set not altered); pinned by `thenActsAndCreatesUnchanged` |
| No eligible anchor exists yet | ✓ Covered | Agent `consultation.routing` capture-gap note; `named-anchors` with empty set; pinned by `thenCaptureGapHandedOnward` |
| The change set matches nothing recorded | ✓ Covered | Agent `consultation.match` explicit no-match clause; pinned by `thenStatesNoShapeMatched`/`thenNothingImplied` |

Two architecture-informed additions beyond the spec's driving set also pass (the root-circle decline and the widened-description assertion, closing the analyze K5 gap), plus two contract-derived additions (re-delegation direction, ungated consultation read) — 14 runnable scenarios total, all green.

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete, all criteria evidenced)

| Task | Status | Evidence |
|---|---|---|
| T001 — wire the gate into the three plugin artifacts | ✓ Met | Nine steps in the accord's order each naming their leaves (see Observation O-1 on step 4); relay loop documented in § The relay (return → direction → re-delegate, repeats from the top, direction-present-means-act); registry lists exactly eight leaves; no sentence in registry or fence claims the reads precede their use; no added sentence restates a change type, placement rule, recorded shape, or the routing rule (verified by added-line grep); `go test ./internal/build/...` green including `CheckProposalDraftingDrift` with zero guard-code edits; `gofmt -l .` clean |
| T002 — BDD content-inspection suites | ✓ Met | 14 runnable scenarios pass (8 + 6), 5 `@validation` held; parsing helpers in production source (`preassemblygate.go`), not test files; comparisons whitespace-normalized via `grammarNorm`; premises derived from the grammar-facts and circle-routing records (fact ids, named-read order, root-circle decline) rather than hard-coded; the two pinned literals (`DraftingGateOrder`, the eight-leaf count) are independent contract anchors whose derivation from the registry would render the check vacuous — the documented exception to the no-hard-coding discipline, matching the `ProposalDraftingGatedWrite` precedent; `gofmt` clean, `go test ./...` green |
| T003 — the truth sweep | ✓ Met | Ships-unconsulted scenario deleted from `circle-routing-guard.feature` with a retirement comment that does not reproduce the retired claim; the claim's phrases now survive only inside the guard's negative-assertion list (the retired-claim set in `givenRegistries`) — no live artifact asserts it; 067 spec carries dated superseding notes on § Entry and the entry assumption naming entry, element list, fence count, and action vocabulary and pointing at 079; both re-validation addenda name the edited surfaces and each one's disposition; portfolio files untouched (`git diff` on FEATURE-MODEL/BACKLOG/ROADMAP is empty) |

---

## Interface Contract Conformance

**Status**: Pass (7 of 7 surface contracts conformant)

| Accord surface | Status | Evidence |
|---|---|---|
| Invocation — both descriptions state the widened entry, keeping every prior boundary sentence | ✓ Conformant | Both frontmatter descriptions state "determines where the change lands … before an anchor is settled on"; all six skill boundary sentences and all four agent fences retained verbatim (pinned by `thenBoundarySentencesRetained`) |
| Structural layout — edits in place, no new plugin files | ✓ Conformant | `git diff --stat` shows three edited files under `plugin/`, zero additions; `hooks/` and both `references/` records untouched |
| The workflow contract — nine ordered steps with named reads | ✓ Conformant | Steps parse in the accord's exact order; prose names the record and the grammar command and restates neither; the six original step activities remain present (067's presence-based pins stay green). See O-1 |
| The fence contract — exactly eight leaves, consultation-read comment, annotation rewritten | ✓ Conformant | Registry lists eight leaves; `proposal grammar` carries its consultation-read annotation in the registry's existing voice; routing block rewritten as the routing step's named reads; the agent fence's matching note rewritten in the same pass |
| Gate posture — `proposal create` the sole gated member | ✓ Conformant | `proposal create` is the only composed leaf in `gated-commands.txt`; the seven reads are absent; the real gate script returns a non-ask decision for `glassfrog proposal grammar` (executed, not assumed) |
| Draft-record output shape — seven elements, `consultation` with three parts, seven `action` values | ✓ Conformant | Six original elements intact; `consultation` present with grammar/routing/match parts and stated as present on every action path; all seven `action` values named with awaiting-direction semantics explicit |
| Defensive-drafting contract — three new entries | ✓ Conformant | "The grammar read fails", "A routing read fails part-way", "Direction is present" added in the section's existing voice |

Error-communication table: all ten rows are honored in the artifacts — the six report-and-continue rows in `consultation`/`notes`, the three awaiting-direction rows in the `action` vocabulary, and the two unchanged write-failure rows preserved from 067.

---

## Non-Behavior Absence

**Status**: Pass (8 of 8 exclusions absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No refusing, blocking, delaying the create; no withholding a draft on the practitioner's chosen anchor | ✓ Absent | Every occurrence of refuse/block/delay/withhold in both artifacts is a negation ("nothing in the loop refuses, blocks, or delays the create"; "refuses, blocks, filters, delays, or withholds"; "reported, never enforced"). The two-phase return is the mechanism by which the practitioner decides — see the VS-2 trace below |
| No typed per-change validator; no checking a change's `type` value or command-specific keys | ✓ Absent | The verbatim-above-the-type-floor clause and "builds **no typed constructor**" / "no typed per-change validator" are retained unchanged; the match step compares against recorded shapes, adding no per-key check |
| No restating grammar content or the routing rule in the workflow text | ✓ Absent | No line 079 added contains a change-type token, placement rule, recorded shape, or the routing rule. The three grep hits for "top-level"/`parent_role_id` are all pre-existing 073 text present at `origin/main`, and both use the CLI-command sense (`roles` is a top-level command) or name a field a read returns — neither is a placement rule or the routing rule |
| No capturing, refining, or retiring a tension — including the capture the gap names | ✓ Absent | The capture-gap note states capture is "handed onward, never performed"; no tension write leaf in the registry; the agent's no-tension-write fence retained |
| No asking for a change target's numeric identifier | ✓ Absent | No identifier-ask language in either artifact; the downstream capability is not anticipated in prose |
| No changing what the grammar read renders or what the routing record states | ✓ Absent | Neither `references/` record nor any grammar-rendering source is in the branch diff |
| No advancing, withdrawing, or responding; no authority verdict | ✓ Absent | 067's fences retained verbatim; the registry adds a read, not a write; 067's authority-verdict scan stays green |
| No presenting consultation as a guarantee of acceptance | ✓ Absent | No acceptance-guarantee language; the artifacts state the opposite — a no-match "implies nothing about validity" and "the server stays the sole judge of what it accepts" |

---

## @wip Lifecycle Completion

**Status**: Pass — remaining tags match the declared hold set exactly

The 14 runnable scenarios across both feature files carry no `@wip`. The five remaining tags are all `@validation @wip`: four in `pre-assembly-grammar-consultation.feature` (unconditional-and-ordered, no-copy, result-legibility, annotation-flip) and one in `pre-assembly-routing-application.feature` (nothing-withholds) — the hold set tasks.md declares, held out by design for this stage. Not a lifecycle gap.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5). Traced independently against the implementation.

| @validation scenario | Status | Trace |
|---|---|---|
| Consultation is unconditional and ordered | ✓ Satisfied | The workflow preamble states the nine steps "run **in order on every draft**", naming both non-skip conditions explicitly (routing's answer cannot gate its own run; the consult is client-less and request-free); the agent repeats it ("Run every step in order on every delegation"). Enumerating the paths: a run either completes through step 8, or returns at step 1 or step 7. A step-1 return never reaches assembly, so no path reaches assembly unconsulted; step 5 precedes step 6 unconditionally; step 7 follows both 1 and 5. Re-delegated runs "repeat the gate from the top", so no resumed path skips a step. No conditional or exception clause exists anywhere in either artifact's gate prose |
| Nothing withholds a write locally | ✓ Satisfied | The sharpest check in this spec, since a two-phase return does pause a run. It is not a local withholding: (a) the pause is never conditioned on what consultation found being *bad* — it is conditioned on the practitioner not yet having decided, and the agent is non-interactive by established precedent (plan ADR-2, developer-confirmed); (b) direction given is always acted on, stated in both artifacts and pinned by the proceed-past scenario (create runs through the confirmed flow unchanged, change set not altered); (c) the spec's own edge-case scenario distinguishes an anchor "handed in" from the practitioner "directing the draft onto the handed-in anchor anyway", and asserts drafting proceeds on the latter — exactly the relay's shape; (d) the only other stop is the pre-existing declined gate confirmation, which 067 already owned. No refusal, filter, or pre-validation sits between the assembler and the create |
| No content was copied into the wiring | ✓ Satisfied | Added-line grep over both prose artifacts returns zero change-type tokens, placement rules (top-level/nested/wrapper), `CSG-` handles, or routing-rule substance (parent-circle/inherits/has_subroles/sensing-role). The artifacts name their sources — the record by in-surface path, the grammar by its CLI command — and invoke them. The dead-shape scenarios' premises are checked against the record in the suite precisely because the prose cannot restate a shape |
| The registry no longer claims the routing reads are ahead of their use | ✓ Satisfied | Both the registry's routing block and the agent fence's matching note were rewritten to describe the reads as the routing step's reads run by workflow step 1. All five phrasings of the retired claim are absent from every live artifact, and their absence is pinned by `givenRegistries` so a regression fails the suite. Both surfaces affirmatively contain "routing step" |
| A reader can tell from the record what was consulted and surfaced | ✓ Satisfied | The `consultation` element is stated as present on **every** action path with three named parts, each specifying what it carries: grammar (consulted, or the named failure with assembly explicitly unconsulted), routing (target circle `role_` id, every eligible anchor `ten_` id, the completeness hedge, the incomplete-walk note, the record's decline, or the capture-gap note), and match (fact handle + shape + symptom, or the explicit no-match). A reader who did not watch the run can answer all three questions from the record alone |

---

## Observations (advisory — non-blocking, not a conformance gap)

**O-1: `proposal get` is named in step 3 rather than step 4.** The accord's workflow table maps `proposal get` to step 4 (Duplicate check); the implementation names it in step 3 (Situate), one sentence before the step whose prose already says "before judging duplicates", and step 4 names no leaf of its own. The accord marks **both** rows "Unchanged", and the pre-079 prose carried `proposal get` inside the situating step — so preserving that placement is the faithful reading of "unchanged", and moving it would edit prose 067's presence-based expectations pin. The behavior is fully present (the agent is instructed to inspect a candidate match with `proposal get`, and the duplicate check returns `surfaced-existing`), so this is a precision mismatch in the accord's own table rather than an implementation gap. Recorded so a future reader does not treat the table's leaf column as falsified; no action owed by this spec.

**O-2: two pinned literals in the new suite are deliberate contract anchors.** `DraftingGateOrder` (the nine step names) and the eight-leaf count are hard-coded rather than source-derived. This is the documented exception to the drift-guard discipline: both sides of these checks would otherwise come from the artifact under test, making the assertion vacuous. Each carries a property comment explaining what it encodes, following the `ProposalDraftingGatedWrite` precedent. A legitimate future fence or workflow change must update the anchor deliberately — which is the intent.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 5 held-out validation scenarios are satisfied by independent trace. All 3 tasks are checked, the 14 runnable scenarios pass, the remaining `@wip` tags match the declared hold set exactly, and the full suite is green (12 packages, `go test ./...` exit 0) with `gofmt -l .` clean. The operating-surface self-containment scan walks all three edited artifacts with zero violations. `CheckProposalDraftingDrift` and 067's and 073's suites stay green over the widened surface with zero guard-code edits, as plan ADR-3 predicted. The implementation conforms to its specification.

Two advisory observations are recorded above; neither is a conformance gap and neither blocks the change.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge — tasks.md branching guidance places Phase 1 and Phase 2 in the same PR as separable commits, which is how the branch is shaped (three commits: T001 wiring, T002 suites, T003 sweep). The specification loop is closed.

---
---

# Validate: Pre-Assembly Grammar Consultation — Round 2

**Feature**: 079-pre-assembly-grammar-consultation
**Round**: 2 of 3
**Date**: 2026-08-22
**Verdict**: Ready
**Trigger**: PR #209 review round 1 surfaced four contract inconsistencies that round 1 of this validation did not catch. Fixed in `136cf0f`; this round re-checks the five dimensions and re-traces the held-out scenarios against the corrected artifacts.
**Artifacts loaded**: unchanged from round 1
**Implementation files**: unchanged set; `plugin/agents/proposal-drafter.md`, `plugin/skills/proposal-drafting/SKILL.md`, and `internal/build/pre_assembly_grammar_consultation_bdd_test.go` edited since round 1

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass (one recorded divergence) | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (5 of 5) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 new findings. All four review findings resolved.

---

## Per-Dimension Results (round 2)

**Driving scenario coverage** — Pass, 10 of 10. Unchanged coverage; two scenarios are now traced through corrected prose rather than contradictory prose. *A handed-in anchor routes the change elsewhere* and *The practitioner proceeds past a surfaced dead shape* previously traced to step-local branches that contradicted the relay; both steps now carry an explicit direction-present exception, so the trace no longer depends on a reader preferring the global rule over the step's own instruction. 14 runnable scenarios pass.

**Acceptance criteria** — Pass, 3 of 3 tasks. T001's criterion "the relay loop … direction-present-means-act is documented" is now satisfied at the step level as well as in § The relay; T002's suite gained three assertions pinning the new phrases, each probed red before being trusted.

**Interface contract conformance** — Pass, 7 of 7 surfaces, with one **recorded divergence**: the agent's `consultation` element now defines a third *not reached* state for its `grammar` and `match` parts, which the accord's canonical part definitions do not carry. The developer decided (2026-08-22) to leave `interface-spec.md` unamended in this PR. The divergence is a **superset, not a conflict** — the artifact satisfies every state the accord names and adds one the accord's own principle demands ("the result names which part was incomplete and why, rather than presenting a partial consultation as a whole one", spec § Reporting the consultation). Recorded here so the next editor amends the accord deliberately rather than "correcting" the artifact back to an untruthful two-state contract. No divergence exists for the direction-present exceptions: the accord's Interactions section already states that a re-delegated run "with direction present, acts on it rather than re-asking" — the fix makes that rule locally explicit at the two steps that can return.

**Non-behavior absence** — Pass, 8 of 8. Re-checked the first exclusion (no refusing, blocking, delaying, or withholding) against the new prose: both added clauses are *continuations* — "the run continues to step 2 on the directed anchor", "the run continues to step 8 with the change set unaltered" — so the fix strengthens the exclusion rather than qualifying it. No refusal, filter, or withholding language was introduced.

**@wip lifecycle completion** — Pass. 19 scenarios total across the two files; 14 runnable carry no tag, 5 carry `@validation @wip`, 0 carry a bare `@wip`. Matches the declared hold set exactly.

---

## Validation Scenario Results (round 2)

**Status**: Satisfied (5 of 5). Re-traced against the corrected artifacts.

| @validation scenario | Status | Trace |
|---|---|---|
| Consultation is unconditional and ordered | ✓ Satisfied | Unchanged in substance, and now stronger: the two direction-present exceptions continue the run *forward* (to step 2, to step 8) rather than skipping a gate step, so no path reaches assembly unconsulted or matches before routing answered. The re-delegated run still repeats the gate from the top |
| Nothing withholds a write locally | ✓ Satisfied — **corrected trace** | Round 1 traced this to the relay's global "direction given is always acted on" and pronounced it satisfied. That trace was **incomplete**: steps 1 and 7 branched unconditionally to a return, so a re-delegation carrying direction re-surfaced the same decision indefinitely and the create was never reached — a withholding produced by contradiction rather than by intent. With the step-local exceptions in place the property now holds at every step that can return, not only in the section that summarizes them |
| No content was copied into the wiring | ✓ Satisfied | Re-grepped the added lines: no change type, placement rule, recorded shape, or routing-rule substance. The new clauses reference the practitioner's direction and step numbers only |
| The registry no longer claims the routing reads are ahead of their use | ✓ Satisfied | Unaffected by the fix; still pinned by `givenRegistries` |
| A reader can tell from the record what was consulted and surfaced | ✓ Satisfied — **strengthened** | This is the scenario the first review finding bore on directly. A reader of an early-return record previously could not distinguish "the grammar was read and nothing matched" from "neither step ran", because the parts had no not-reached state. Both parts now name the return that ended the run ahead of them, and each forbids reporting work that never happened |

---

## Changes Since Previous Run

**Round**: 2 (previous: Round 1 — verdict Ready, zero findings)

### Resolved (4 items, all from PR #209 review round 1 — none had been found by round 1 of this validation)

- **Consultation element had no not-reached state** — resolved in `136cf0f`. `grammar` and `match` each gained a third state naming the return that ended the run, plus an explicit ban on reporting work that did not happen; the element intro states that early returns carry it with unreached parts saying so.
- **Agent identity paragraph stated the narrow entry** — resolved in `136cf0f`. The body's input contract now matches the widened frontmatter description (intended change, anchor `ten_` id optionally in hand, plus explicit direction on a re-delegation).
- **Step 1 branched unconditionally, looping on re-delegation** — resolved in `136cf0f` with an explicit direction-present exception continuing to step 2 on the directed anchor.
- **Step 7 branched unconditionally, looping on proceed-past** — resolved in `136cf0f` with an exception continuing to step 8, change set unaltered.

### Remaining (0)

### New (0)

### Correction to round 1's record

Round 1's verdict of "Ready, zero findings" was **wrong on one dimension**, and the error is worth naming precisely because it is a repeatable inspection failure, not bad luck:

- Round 1 verified the **frontmatter description** carried the widened entry and never checked whether the artifact **body** still stated the narrow one. A widened-entry check must sweep every place the entry is stated, not only the discovery surface.
- Round 1's VS-2 trace credited a **globally stated rule** (§ The relay's direction-present promise) without checking whether the **step-local branches** that rule governs contradicted it. Where an executing agent reads one step at a time, a global rule and a contradicting local branch is a defect, and the local text is what gets followed.

Round 1's per-dimension results and verdict are left in place above as the record of what was checked and concluded at the time.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero new findings. All 5 held-out validation scenarios are satisfied, two of them (never-withhold, result-legibility) now on materially firmer ground than in round 1. The four review findings are resolved in `136cf0f`, each new load-bearing phrase pinned by the content-inspection suite and probed red before being trusted. Full suite green (`go test ./...` exit 0, 12 packages), `gofmt -l .` clean, and all 9 CI checks pass on the fix commit.

One divergence is recorded rather than fixed, by developer decision: the accord's `consultation` part definitions do not carry the not-reached state the artifact now defines. The artifact is a truthful superset; the accord should gain the state whenever it is next revised.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge of #209. If the accord divergence is to be closed in this PR after all, the edit is three lines in `interface-spec.md` (the two part definitions and a note); otherwise it carries forward as recorded above.
