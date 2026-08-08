# Validation: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Role**: Guardian
**Round**: 1
**Date**: 2026-08-08
**Verdict**: **Ready**

**Artifacts loaded**: spec.md, plan.md (§ System Architecture + ADRs), tasks.md (3 of 3 tasks complete), interface-spec.md, PROJECT.md, 2 `.feature` files (16 scenarios), implementation (10-file change set vs `origin/main`).

> Guardian agent definition and validate template are not deployed in this environment — proceeded with the SKILL.md process alone (reduced character consistency, not a blocked skill). Context-engineering references not found — applied skill-specific checks only.

> **Environment note**: local `main` is stale (spec 051); the true change set was inspected against `origin/main` (#187) per the workspace's target-branch convention. A first-pass diff against stale local `main` falsely showed the drafting SKILL.md/agent as modified — those changes belong to 067, already merged. None of 072's three commits touch them.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| 1. Driving Scenario Coverage | ✅ Pass | 0 |
| 2. Acceptance Criteria | ✅ Pass | 0 |
| 3. Interface Contract Conformance | ✅ Pass | 0 |
| 4. Non-Behavior Absence | ✅ Pass | 0 |
| 5. @wip Lifecycle Completion | ✅ Pass | 0 |
| Validation Scenarios (held-out) | ✅ 3/3 satisfied | 0 |

No dimension skipped — every source artifact was present.

---

## Dimension 1 — Driving Scenario Coverage

All 7 driving scenarios have identifiable code paths (the record's content + the guard's enforcement + the executable BDD steps that assert them):

- **Own-circle policy shape recorded with its symptom** → `## CSG-1` (Shape: top-level `CreatePolicy`, no `UpdateRole` wrapper; Symptom: wrapped shape refused, web UI emits the top-level form). Asserted green by `TestGrammarFactsFeatures`.
- **Accepted-but-invalid shape recorded with its disposition** → `## CSG-2` (Disposition `accepted-but-invalid`; Symptom: `valid: false`, blocking alert, `available_transitions: []`).
- **Consumer reads a dead shape to avoid before assembling** → both facts fully readable (Shape + Symptom present); BDD "An assembler finds both the shape to use and the shape to avoid".
- **A recorded fact would duplicate the published contract** → citations by schema/property anchor only; guard conditions 6/7 + `RestatedTypesBeyondCitations` reject a restated enum. BDD "Contract-carried shapes appear as citations, never restatements".
- **A recorded fact would read as spec-authoritative** → leading empirical marker; guard condition 8 + BDD "Every fact is marked empirical, never contract".
- **The published contract absorbs a recorded fact** (edge) → mechanized against fixtures by the guard's complete/partial retirement scenarios; the real future maintenance event is process-inexecutable and held (see Dimension 5).
- **"Created" is not "valid"** (edge) → CSG-2 Symptom states a returned `prp_` id is *not* a successful governance change; BDD "A returned proposal id is not read as a valid governance change".

## Dimension 2 — Acceptance Criteria

All 3 tasks checked (`- [x]`); each task's criteria met and evidenced by passing tests:
- **T001**: manifest declares `CSG-1, CSG-2` set-equal to the fact sections; both facts carry all five fields with the pinned Dispositions/Evidence/Provenance; citations by name only. (6 BDD scenarios green.)
- **T002**: all 8 violation conditions fire naming invariant + element + resolution path (`TestGrammarFactsGuardConditions`, 8 subtests); complete retirement passes while partial fails; every side derived — no hard-coded enum values, type names, or fact ids. `gofmt -l` clean, `go test ./...` green.
- **T003**: one `DEPRECATION.md` entry names F5→CSG-1, F6→CSG-2, the record path, origin 072, the three-part retirement act, the empty-shell terminal case, and never-reuse-ids; no second copy of the fact content.

## Dimension 3 — Interface Contract Conformance

`interface-spec.md` is the only interface file. Conformant:
- **Record anatomy** (5 rows) present in order: empirical marker → header (Owner, Contract citations, `Live facts` manifest) → contract-citations section → two fact sections headed `## CSG-<n> — <title>`.
- **Per-fact contract**: five bold-labelled fields, closed Disposition vocabulary, `prp_` Evidence, LEARNINGS Provenance.
- **Guard coupling**: `internal/build/grammarfacts.go` (path constant + both-side parsers) and `grammarfacts_guard_test.go`; derivation helpers in production source, assertions in the test (family convention).
- **Error Communication**: all 8 conditions with their named resolution paths, verified by the condition table test.

## Dimension 4 — Non-Behavior Absence

All 5 exclusions hold:
1. **No local pre-reject** — the record is inert markdown; drafting's SKILL.md/agent untouched, so no consumer wiring pre-rejects. The only code (the guard) inspects the record file at CI, never a user change set. ("rejected" appears solely as a *Disposition label*, not a code path.)
2. **Not spec-authoritative** — the empirical marker is a guarded structural invariant (condition 8).
3. **No restatement** — guard conditions 6/7 + `RestatedTypesBeyondCitations`; citations are by schema/property anchor, never line number (`lineCitationRE` rejects that form).
4. **No runtime detection** — the record states a shape to avoid; grep finds no detection/read-back code.
5. **No change-target identifier facts** — grep finds no `databaseId`/numeric-id/URL-scraping content.

## Dimension 5 — @wip Lifecycle Completion

- **Guard feature**: 0 `@wip` remaining — all 5 scenarios executed by T002.
- **Facts feature**: `@wip` remains only on the 3 `@validation` scenarios (future validate work, unreferenced by any task) and on "A contract-absorbed fact retires from the record" — a real future maintenance event, process-inexecutable by automation and referenced by no task, so correctly held. Every scenario referenced by a checked task has had its `@wip` removed.

---

## Validation Scenarios (held-out)

| Scenario | Result | Evidence |
|---|---|---|
| The record carries only the residual shapes | ✅ Satisfied | Exactly two `## CSG-` fact sections (CSG-1, CSG-2); manifest set-equals them; enum/nested-only appear only as citations (guard conditions 6/7 + restatement check). |
| No local judgment leaks into the record | ✅ Satisfied | Record is inert markdown; no code path rejects/filters/pre-validates a change set; drafting skill/agent untouched — the record's only effect is to inform. |
| The provisional source is superseded, not copied | ✅ Satisfied | `DEPRECATION.md` records F5→CSG-1 / F6→CSG-2 supersession (origin 072); LEARNINGS keeps its forward pointer; no duplicate fact content introduced. |

---

## Verdict: Ready

All three tasks are checked, all five conformance dimensions pass with zero findings, and all three held-out `@validation` scenarios trace to clear evidence. The implementation conforms to its specification: the record carries exactly the two residual shapes with their symptoms and dispositions, marked empirical and citing (never restating) the published contract; the sibling guard turns citation drift and partial retirement into build failures with named resolution paths, deriving every side from source; and the LEARNINGS provisional copy is superseded rather than left to drift.

One boundary worth carrying forward (not a finding — it is the design, stated in the plan and the guard's comment): the guard cannot detect the spec *semantically* absorbing CSG-2, since `accepted-but-invalid` is behavior prose, not a schema key. That residue is owned by the vendored-spec refresh-diff review (LEARNINGS 2026-08-05 S7), not by this guard.

**Handoff**: The specification loop is closed. Suggest PR review and merge.
