# Analyze: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/unguided-change-construction/change-set-grammar-facts.feature, features/unguided-change-construction/change-set-grammar-guard.feature
**Checklist context**: loaded — 8/9 constitution checks pass, 1 P0 failure, 1 P2 consideration
**Checks**: 16 (15 pass, 1 fail)
**Generated**: 2026-08-07 (round 2 — re-derived after the manifest enrichment and the scenario split)

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 6 | 6 | 0 |
| Completeness (P1) | 6 | 5 | 1 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **16** | **15** | **1** |

One interface file and two feature files. Scenario-scaled checks (C6, K1, K5) evaluate across both feature files as one set, per the matrix's scaling rule.

---

## Changes Since Previous Run

**Previous** (round 1): 1 P0, 1 P1, 1 P2 (3 findings)
**Current** (round 2): 0 P0, 1 P1, 0 P2 (1 finding)

**Resolved**:
- ~~P0 | C4: plan.md § ADR-4 ↔ interface-spec.md § Error Communication — retirement deletes a fact while the guard requires both facts by name, so the retirement flow tripped the guard it was meant to satisfy~~ → fixed. The record now declares a `Live facts` manifest; the guard derives the expected set from it and hard-codes no fact IDs. Conditions 1 and 2 are directional, so a complete retirement passes and a partial one fails. plan ADR-3 and ADR-4 now describe one flow, and interface-spec.md states plainly what the old contract would have done.
- ~~P2 | H3: spec.md § Ambiguity Warnings ↔ plan.md — the spec deferred drift detection to a separate capability while the plan built one~~ → fixed. plan.md § What This Plan Does Not Cover now carries an explicit narrowing that supersedes the spec's deferral sentence to a named extent (general vendored-spec drift stays separate; citation integrity of this record does not) and flags that the distinction lives in the plan, not the spec.

**Carried forward, narrowed**:
- P1 | K5 — round 1 found three of six guard conditions and two of five per-fact fields uncovered. The enrichment took the guard to eight conditions and the scenario work covered seven of them plus both previously-missing fields. **One condition remains uncovered** — see below. The gap is materially smaller but not closed.

---

## Consistency: 6/6 passed

All consistency checks pass. C1 spec integration boundaries → plan components; C2 spec behavioral accord ↔ plan architecture (the manifest serves the Maintenance accord rather than contradicting any behavior); **C4 plan ADRs ↔ interface contracts — the round-1 contradiction is resolved**, and ADR-3's "hard-codes no fact IDs" now agrees with the interface's manifest-derived expected set; C5 plan components ↔ task scope (record → T001, guard → T002, supersession → T003); C6 interface surfaces ↔ scenario steps (every step references a defined element — the `Live facts` manifest, fact IDs, the five field labels, the closed disposition vocabulary, the resolution path).

**C3 evaluation note** (pass, reasoning shown because the call is close, as in round 1): spec § Non-Behaviors 3 forbids restating contract-carried change-type *shapes*, and the record carries the six nested-only type names as a citation list. Judged compatible on the same grounds as round 1 — a bare list of names carries no shape information, the drift the non-behavior guards against is structurally prevented by the guard's set-equality check, and interface-spec.md discloses the choice rather than letting it pass silently. The manifest does not change this analysis: it declares fact IDs, not contract content.

---

## Completeness: 5/6 passed

### Findings

**P1** | K5: interface-spec.md § Error Communication → features/unguided-change-construction/*.feature

> **Condition 6 has no scenario.** The interface defines eight guard failure conditions; the scenarios now reach seven:
>
> | Condition | Covered by |
> |---|---|
> | 1 — manifest declares an ID with no section | "A partial retirement fails the guard" (row: the fact section) |
> | 2 — a section's ID absent from the manifest | "A partial retirement fails the guard" (row: the manifest entry) |
> | 3 — zero fact sections | "A record with no facts left fails as an empty shell" |
> | 4 — missing or empty required field | "A structurally invalid fact fails the guard" (row: empty Evidence) |
> | 5 — Disposition outside the closed vocabulary | "A structurally invalid fact fails the guard" (row: probably-fine) |
> | **6 — a cited change type absent from the spec enum** | **— nothing** |
> | 7 — nested-only citation not set-equal to the spec's set | "A spec refresh that moves a cited anchor fails the build" |
> | 8 — empirical marker absent or degraded | "Every fact is marked empirical, never contract" |
>
> Condition 7's scenario is specifically about the *nested-only set*; it does not exercise the enum-membership check. The nearest record-side scenario, "Contract-carried shapes appear as citations, never restatements", asserts citation *form* — that anchors are schema/property names and no enum values are restated — not that every cited type actually exists in the enum.
>
> This matters more than a generic coverage gap: condition 6 is the half of the citation tripwire that catches a fact naming a change type the contract *removed* or *renamed*, which is the likelier refresh outcome for a record whose facts each name specific types (`CreatePolicy`, `UpdateRole`). The nested-only check would not fire on it.
>
> Mitigation on record: tasks.md T002 acceptance criteria enumerate all eight conditions and require each to fail loudly, so condition 6 will be covered by a Go guard test. The gap is in Gherkin coverage, not verification altogether — the same shape as round 1's finding.
>
> **Correction to a prior claim**: the `/score:scenarios` handoff stated that all eight conditions had scenario coverage. That was wrong — seven do.

### Passed (5/6)

K1 spec driving scenarios → Gherkin (all 10 spec scenarios survive the split intact — 7 driving and 3 validation, all in the record file; verified by title against spec.md); K2 spec integration boundaries → interface coverage (all four boundaries covered by the single specification touchpoint; sibling-file absence justified in Consistency Notes); K3 plan phases → tasks (one phase, one section); K4 plan components → implementing tasks (record, guard pair, supersession); K6 spec user scenarios → interface coverage (US1 → pre-assembly consultation, US2 → structural contract + guard coupling, US3 → maintenance flow).

---

## Coherence: 4/4 passed

All coherence checks pass. H1 terminology (the new vocabulary — *manifest*, *Live facts*, *resolution path* — is used identically across plan, interface, both feature files and tasks; the `CSG-1`/`CSG-2` ↔ LEARNINGS `F5`/`F6` alias remains explicit via the required Provenance field); H2 detail symmetry (spec↔plan and plan↔tasks proportionate; the interface's greater depth on the guard is the expected refinement direction); **H3 scope alignment — resolved**, the spec's deferral is now explicitly narrowed in the plan rather than silently overridden; H4 phase coverage (one plan phase, one task phase, no task references a phase the plan does not define).

**Split evaluation note** (H3, pass): the scenario set moved from one file to two during round 2. Judged not to be scope drift — both files sit in the same problem directory (`unguided-change-construction/`), each Feature narrative names the other so neither reads as the whole story, the moved scenario kept its `# Source:` comment, and tasks.md lists both files as inputs and dispositions all 16 scenarios across them. Multi-file problem directories are established precedent in this repo (`unequipped-agent-operators/` carries nine).

---

## Checklist Correlation

- Checklist's **P0 under III** (the guard's behavior when the record file is *absent* is unspecified) is a vertical finding with no horizontal twin — analyze's C4 confirms plan and interface now *agree* about retirement, and they agree about a flow that neither one carries to its terminal state. The two findings are complementary, not duplicates: consistency between artifacts does not imply completeness of what they jointly specify. Worth noting that C4's fix is what created the end-state the checklist finding is about.
- Checklist's **P2** (fact IDs "never reused" is stated but unenforced and not disclosed as unenforced) sits in the same seam as this round's **K5**: both are places where the interface asserts something the guard does not check. The difference is that K5's condition 6 *is* meant to be checked and lacks a scenario, while the never-reuse rule cannot be checked and lacks a disclosure.
- Round 1's correlation between checklist P2-2 and analyze C4 is retired — both are resolved, and by the same edit, as predicted.

---

## Governance Notes

- **All 16 base checks ran.** No checks skipped — spec.md, plan.md, one interface file, two feature files and tasks.md are all present.
- **Checklist context**: loaded and parsed — 8/9 constitution checks pass, 1 P0, 1 P2, no done-criteria checks (no `accords/governance/` deployed).
- **Provenance discipline**: findings report gaps without deciding which artifact should change. K5's condition 6 could be closed by a scenario in the guard file, or the developer may accept Go-test-only coverage for it as they may for the others — that call belongs to the Verifier, not this analysis.
