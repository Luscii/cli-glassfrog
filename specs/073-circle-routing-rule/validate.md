# Validate: Circle Routing Rule

**Feature**: 073-circle-routing-rule
**Round**: 1 of 3
**Date**: 2026-08-08
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (6 of 6 tasks complete), interface-spec.md, circle-routing-rule.feature (12 scenarios), circle-routing-guard.feature (12 scenarios), PROJECT.md
**Implementation files**: 9 — `plugin/skills/proposal-drafting/references/circle-routing-rule.md` (the record), `internal/build/circleroutingrule.go` (parsers + guard), `internal/build/circle_routing_rule_bdd_test.go`, `internal/build/circle_routing_rule_guard_test.go`, `plugin/agents/proposal-drafting-commands.txt`, `plugin/agents/proposal-drafter.md`, `plugin/skills/proposal-drafting/SKILL.md` (count sweep), `internal/build/proposaldrafting.go` (widened drift check), `.score/memory/DEPRECATION.md` (F7 supersession); plus the 067 re-validation addendum in `specs/067-proposal-drafting-path/validate.md`

*(guardian-agent.md not deployed — proceeded per SKILL.md fallback. Context-engineering references not found — applied skill-specific checks only.)*

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. Full test suite green (2216 tests, 12 packages), `gofmt -l` clean.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 spec driving scenarios covered; 18 executable scenarios pass)

The deliverable is a committed knowledge artifact, so coverage means the record *states* what each scenario asserts, pinned by passing godog scenarios that parse the committed file (operator-path convention: whitespace-collapsed content assertions, raw-file structural checks).

| Spec driving scenario | Status | Implementation |
|---|---|---|
| A change to a circle's own governance routes to the parent | ✓ Covered | record § Rule (Own-circle consequence, Mechanism); BDD "An own-circle change routes to the parent circle" |
| The content states how to tell the two cases apart | ✓ Covered | record § Classification test (Test, Resolved by naming `Role.has_subroles`); BDD "The classification test distinguishes the two cases" |
| The procedure names the reads and how to state the answer | ✓ Covered | record § Procedure named-reads block (`me roles`, `tension list`, `roles`, in order) + Answer shape (`role_`/`ten_` ids); BDD "The procedure names its reads in the order they run" |
| The Circle Lead exception is stated | ✓ Covered | record § Rule (Circle Lead exception: circle-role itself a valid anchor site, "need not go to the parent circle"); BDD scenario |
| The procedure prescribes what to say when no tension is sensed | ✓ Covered | record § Procedure (Gap reporting: "no eligible anchor exists yet", capture on that specific role in that specific circle); BDD scenario |
| Nothing the content prescribes stops a write | ✓ Covered | record closing statement ("reported, never enforced… nothing in this record refuses, blocks, or delays a `proposal create` — the server remains the judge"); BDD scenario |
| An ordinary change needs no parent hop | ✓ Covered | record § Rule (Mechanism: "lands in the circle containing that role… no separate case exists for it"); BDD scenario |
| An absence that cannot be proven must be reported as unproven | ✓ Covered | record § Procedure (Uncertainty: "none found" in `me roles`, completeness uncertain, "does not follow pagination… never a settled absence"); BDD scenario |
| The target circle has no parent | ✓ Covered | record § Rule (Root circle: "no parent circle to route to… not resolved… neither the circle itself nor any other circle"); BDD scenario |

The interface-added header scenario ("The document header names its owner and the consumption rule") also passes — Owner line names the skill + symlink rule, Contract citations line names the vendored spec.

The guard-side scenarios (8 executable) all pass: premise tripwire, dropped Role field, unresolvable named read, unanchorable path, the 3-example structural Outline, missing marker, condition-7 registry cross-check, and the gate-posture-unchanged widening scenario.

---

## Acceptance Criteria

**Status**: Pass (6 of 6 tasks checked, all criteria met)

- **T001** — Record anatomy complete and in order: marker blockquote with the cite-versus-observe split (`RoutingMarkerIsWellFormed` asserts both halves); Rule 4 fields with the Root circle declining a target; Classification 3 fields naming `Role.has_subroles`/`Role.parent_role_id` and the own-roles shortcut; Procedure 4 fields + fenced named-reads block in `*-commands.txt` token grammar; header Owner + Contract citations lines; citations by schema/property name only (no line numbers, no pinned counts — verified by grep); no refusal prescribed. Ten step definitions landed, @wip removed.
- **T002** — Conditions 1–6 and 8–9 fire loudly with invariant + element + resolution path (`TestCircleRoutingRuleGuardConditions` mutation table, one case per condition). Premise tripwire is set-equality of the whole `proposal` property set against the record's own cited premise set — both derived, both named in the failure with the two resolution paths. Four-way named-read resolution with a reporting unanchorable-default arm. No hard-coded read names, property sets, or schema values (the required field *labels* are pinned as checked-in contract facts, the 072 precedent). Three residues stated in `CheckCircleRoutingRule`'s comment. Six scenarios, gofmt clean, suite green.
- **T003** — DEPRECATION.md entry names F7, the record path, and the origin (073); states whole-record retirement triggered by premise dissolution vs consequence-level ordinary edits; states the no-manifest contrast with the sibling's three-part flow; no second copy of routing content introduced (the LEARNINGS entry is untouched).
- **T004** — Registry, fence, and `CheckProposalDraftingDrift` widened in one commit (e1ce01d); `TestProposalDraftingDriftGuard` green; unanchorable-default arm reports; gated membership re-asserted over seven leaves (create sole gated member, six reads absent — pinned by the widening scenario); annotations in both registry and fence state the reads are present ahead of #77's consultation; gofmt clean.
- **T005** — Condition 7 one-direction: a named read absent from the registry fails naming the leaf, the registry path, and both resolution paths; the reverse direction's rationale is written into the guard comment; both sides parsed into the same token grammar; no read name hard-coded.
- **T006** — 067's four verbatim-pinned assertions re-derived by property (fence = registry exactly; every leaf resolves; tension leaves are reads only) with in-place "re-validated 2026-08-08" markers; must-not-change invariants confirmed (no propose/respond/withdraw or tension write in registry or as runnable fence commands; create sole gated member); re-validation addendum records trigger, re-run, and outcome; 067 STATUS row stays Complete.

---

## Interface Contract Conformance

**Status**: Pass — all specification-touchpoint surfaces present as contracted (interface-spec.md)

| Interface surface | Status | Evidence |
|---|---|---|
| Record path + sibling placement (`references/` second member) | ✓ Conformant | File at the contracted path beside `change-set-grammar-facts.md` |
| Anatomy rows 1–6 in order (marker, header, citations, rule, classification, procedure) | ✓ Conformant | Parsed and asserted by the BDD suite and guard conditions 1–3 |
| Rule/Classification/Procedure field tables (4/3/4 bold-labelled fields) | ✓ Conformant | `routing*RequiredFields` pin the labels; condition 2 enforces non-emptiness |
| Named-reads block (fenced, `*-commands.txt` token grammar, procedure order) | ✓ Conformant | `parseNamedReadsBlock`; order asserted by BDD; grammar shared with the registry parse (condition 7 compares like with like) |
| Composed-surface additions (registry + fence + widened drift check, one commit) | ✓ Conformant | Commit e1ce01d; annotations present in both artifacts; "these and no others" wording preserved |
| Guard coupling (`circleroutingrule.go` + guard test, every side derived) | ✓ Conformant | Record side, spec side (`LoadSpecRoutingAnchors`), live surfaces, and registry all parsed at test time |
| Error communication — conditions 1–9 with named resolution paths | ✓ Conformant | Mutation table exercises every condition; messages name invariant, element, and resolution path |
| Three stated residues (semantic drift; condition 7 reverse; unguarded flags) | ✓ Conformant | `CheckCircleRoutingRule` doc comment, EXPLICITLY PARTIAL block |
| Maintenance flow (whole-record retirement; F7 supersession via /score:deprecate) | ✓ Conformant | DEPRECATION.md entry (T003) |

---

## Non-Behavior Absence

**Status**: Pass — every spec § Non-Behaviors exclusion is honored

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not itself perform the routing determination | ✓ Absent | No workflow step consults the record: drafting SKILL.md workflow has zero mentions of the record or its reads (grep); the agent's only reference is the fence note that explicitly disclaims a routing step; no Go code outside `internal/build` reads `CircleRoutingRulePath` |
| Must not refuse, block, or gate a proposal create | ✓ Absent | The record prescribes reporting, never enforcement (its own closing statement); the guard is a CI build check, not a runtime path; the feature ships no runtime code at all |
| Must not report "fills no role" as settled fact | ✓ Absent | The only absence phrasing is the hedged one — Uncertainty prescribes "none found" in a named read, "never a settled absence" |
| Must not present routing as v5-contract-authoritative | ✓ Absent | The marker's cite-versus-observe split marks the landing behaviour as observed, no part of the published contract; guard condition 3 pins the split |
| Must not capture, create, or write anything | ✓ Absent | The record is markdown; the three composed-surface additions are reads (gated-membership scenario confirms none entered 063's gated set) |
| Must not teach Holacracy practice beyond routing | ✓ Absent | Record content is the rule, the classification test, and the procedure — no practice guidance (read end to end) |
| Must not restate the sibling's shape facts | ✓ Absent | Grep for the sibling's vocabulary (`CreatePolicy`, `UpdateRole`, top-level, wrapper, nested-only) finds nothing in the record |

---

## @wip Lifecycle Completion

**Status**: Pass — the remaining tags exactly match the disposition table's hold set

18 executed scenarios carry no `@wip` (10 content, 8 guard-side including the 3-example Outline). Remaining tags: the five `@validation @wip` scenarios (held out by design; the sibling 072 feature retains its `@validation @wip` tags the same way) and one plain `@wip` on "Premise dissolution retires the whole record, not one field" — declared **process-inexecutable by automation** in the tasks.md disposition table (it asserts a future maintenance event; its trigger half is exercised by the premise-tripwire scenario against a fixture, and its supersession half is carried by T003's DEPRECATION.md entry). Not a lifecycle gap.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5). Traced independently against the implementation.

| @validation scenario | Status | Trace |
|---|---|---|
| Only routing is recorded, never change-set shape | ✓ Satisfied | Read end to end: every element answers where a change lands or what anchors it (rule/classification/procedure/citations). No sibling-owned shape vocabulary appears — grep for `CreatePolicy`/`UpdateRole`/`CreateRole`/`CreateDomain`/`CreateAccountability`/top-level/wrapper/nested is empty; `changes` appears only as the cited create-request property, which is the premise citation, not a shape fact |
| No statement about a missing role asserts a settled absence | ✓ Satisfied | The record's only missing-role statements are Gap reporting (a missing *tension*, reported with the closing capture step) and Uncertainty ("none found" in `me roles`, completeness uncertain, "never a settled absence"); Parent resolution's "not among them" is a branch condition, not an absence claim |
| The three routing reads appear in both the registry and the agent fence | ✓ Satisfied | `me roles`, `tension list`, `roles` present in `proposal-drafting-commands.txt` and the drafter's § Composed commands; each resolves on the CLI (four-way drift resolution green); no write leaf entered alongside them (gate-posture scenario: create remains the sole gated member, six reads absent from the gated set) |
| No workflow step consults the record or runs its reads to route | ✓ Satisfied | Drafting SKILL.md workflow unchanged (zero references to the record or its reads); the drafter agent's only mention is the fence annotation that states "no step in your workflow consults that record or runs these reads to route"; no runtime consumer of `CircleRoutingRulePath` exists outside `internal/build` |
| Nothing the feature ships can refuse a change set locally | ✓ Satisfied | The feature ships one markdown record (prescribes reporting, never enforcement — "the server remains the judge") and one CI guard (`internal/build` test — fails builds, touches no request path); there is no code path between an assembler and `proposal create` introduced by this feature |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 5 held-out validation scenarios are satisfied by independent trace. All 6 tasks are checked, the 18 executable scenarios pass, remaining `@wip` tags match the declared hold set exactly, and the full suite is green (2216 tests) with gofmt clean. The implementation conforms to its specification.

Advisory notes (non-blocking):
- The premise-dissolution scenario stays permanently inexecutable by automation; its mechanics are recorded in DEPRECATION.md (T003) and its trigger is mechanized by guard condition 8. Nothing further owed by this spec.
- Phase 2 (commits e1ce01d…c47da66) edits the validate-pinned 067 fence; tasks.md branching guidance asks for it as its own PR, and the commits are cleanly separable at that boundary.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (per tasks.md branching guidance: Phase 1 and Phase 2 as separate PRs if practical). The specification loop is closed.

---

## Re-validation addendum — 2026-08-22 (079 T003: the record's consultation wired in)

**Trigger**: 079 (Pre-Assembly Grammar Consultation) landed the consultation this validation recorded as pending: the drafting workflow's first step now consults the circle-routing record and runs its named reads (`me roles`, `tension list`, `roles`) in the record's order, and the registry/fence annotations that said no workflow step consults the record were rewritten as the routing step's named reads — the "later change to this path" those annotations promised.

**Disposition of the pinned surfaces**:

- *The record itself* (`circle-routing-rule.md`) — untouched by 079. Its content contracts stay owned by this spec; 079 consults the record and restates nothing from it (pinned by 079's no-copy validation scenario).
- *Validation scenario 4, "No workflow step consults the record or runs its reads to route"* — its premise ("this capability landed on its own") dissolved with 079's landing, so the scenario was retired from `circle-routing-guard.feature` with a source comment naming what landed and where the consulted state is asserted. Runner-safe by inspection: the guard suite runs with `~@wip` and no Go step bound the retired scenario. The consulted state is now asserted by 079's `pre-assembly-routing-application.feature` suite.
- *Validation scenario 3, "The three routing reads appear in both the registry and the agent fence"* — still satisfied: the reads remain in both surfaces, now described as the routing step's reads, and the four-way drift resolution stays green.
- *Gate posture* (the widening scenario) — re-ran green over the eight-leaf registry: `proposal create` remains the sole gated composed leaf, every read absent from the gated set, zero guard-code edits.

**Outcome**: 073's STATUS row stays `Complete`. The feature's only falsified statement was the ships-unconsulted scenario, retired in the same change that falsified it (079 plan ADR-4). This addendum is the record of the re-validation, performed by 079 T003.
