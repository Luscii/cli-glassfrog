# Checklist: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/unguided-change-construction/change-set-grammar-facts.feature, features/unguided-change-construction/change-set-grammar-guard.feature
**Checks**: 9 (8 pass, 1 fail) + 1 P2 consideration
**Generated**: 2026-08-07 (round 2 — re-derived after the manifest enrichment)

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** — the same standing as sibling specs 024/028/029/030/071. Done-criteria checks are skipped, not failed.

> Calibration note: this feature ships **no runtime code** — one committed knowledge artifact plus one `internal/build` guard. Nine principles were calibrated to that shape; three produced zero applicable checks (see Governance Notes). The round-1 calibrations are carried forward unchanged, so round 2 is measured against the same bar rather than a re-drawn one.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 9 | 8 | 1 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 1 | — | — |
| **Total** | **9 checks** | **8** | **1** |

---

## Changes Since Previous Run

**Previous** (round 1): 0 P0 fail, 3 P2 considerations
**Current** (round 2): 1 P0 fail, 1 P2 consideration

**Resolved**:
- ~~P2-1: plan.md § System Architecture — hard-coded "21-value" enum count would go stale silently~~ → fixed; the prose now names the enumerated change types open-endedly and hedges the nested-only set. Verified by grep: no count survives anywhere in plan.md.
- ~~P2-2: interface-spec.md § Error Communication — failures name the fault but not the resolution path~~ → fixed; the table gained a **Resolution path named** column across all eight conditions, and plan ADR-3 states the requirement so the interface isn't the only place it lives.
- ~~P2-3: tasks.md T002 — acceptance criteria said "two scenarios" while three were referenced~~ → fixed; scenario ownership was reassigned by *what turns a scenario green*, and both T001 and T002 now claim six and list six. Verified mechanically, not by eye.

**New failure** (introduced by the round-1 fix, not present before):
- P0 | III — the manifest design added a designed end-state (the record is deleted when its last fact retires) whose guard behavior is undefined. See below.

---

## Constitution Checks: 8/9 passed

### Failures

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): *"Errors MUST be obvious and recoverable, never hidden"* — calibrated in round 1 as *"the guard fails loudly on every condition, with no defaulting or partial-success mode, and its coverage limits are stated, not silent."*

→ **interface-spec.md § Error Communication ↔ § Interactions (Maintenance flow), and plan.md ADR-4**: the artifact set now instructs deleting the record itself when its last fact retires — *"an empty record is deleted, not kept as a shell"* (ADR-4), *"the record itself is deleted rather than kept as an empty shell"* (Interactions). **The guard's behavior when the record file is absent is specified nowhere.** Condition 3 covers a record that *exists with zero fact sections*; a record that does not exist is a different input and is not in the eight conditions, and the Degradation note (*"none — the record has no optional inputs"*) does not reach it.

So the Builder implementing T002 must decide, unguided, what `os.ReadFile` failure means — and both answers are defensible and wrong in opposite directions: **fail** and CI breaks the moment a maintainer correctly follows ADR-4's terminal instruction; **pass** and the guard silently green-lights a missing record, which is exactly the swallowed-failure mode III exists to prevent. Neither behavior is stated, so whichever gets baked in is an accident.

Related and unaddressed in the same seam: ADR-4 says the record "goes with it" but never says whether the **guard** retires alongside it. A deleted record with a live guard is the state that produces the failure above.

*Severity is inherited mechanically from the source (III is a MUST → P0), not assessed by impact. The state is not reachable today — both facts exist — but the guard code that decides it is written by T002.*

### Passed (8/9)

- **P0 | I (Spec Fidelity)** — the record still cites rather than restates; the manifest is a statement about the record's own membership and touches no contract claim. The enum and nested-only anchors remain schema/property names with no pinned count (P2-1's fix strengthened this). **PASS.**
- **P0 | II (Action Transparency)** — no CLI action added. The guard's diagnostics now name the invariant, the offending element, **and the applicable resolution path** across all eight conditions — the round-1 P2-2 gap is closed, making this the strongest it has been. **PASS.**
- **P0 | IV (Test-Driven Development / BDD)** — 16 scenarios across two feature files, all `@wip`, written before implementation. tasks.md dispositions all 16 by reason, and the hold set stays split — three `@validation`, one process-inexecutable with a pointer to the guard scenarios that mechanize its checkable half. Task-to-scenario counts verified mechanically (T001 six/six, T002 six/six). **PASS.**
- **P0 | V (Composition over Monolith)** — still a sibling guard, no widening of an existing one; the second feature file is additive. No existing skill, agent, command, or guard is edited. **PASS.**
- **P0 | VII (Working Software)** — each task pairs its change with verification: T001 record + six step definitions, T002 guard + six step definitions + `gofmt -l .` and `go test ./...` gates, T003 verified by a held-out validation scenario. **PASS.**
- **P0 | VIII (No Fabricated Data)** — evidence and provenance unchanged and previously verified live against the vendored contract (enum 21 values, nested-only exactly the six, `CreatePolicy` absent). The manifest introduces no claim about external data. **PASS.**
- **P0 | IX (Writes Require Explicit Intent)** — no write path, no command; the record ships inert with no SKILL.md, agent, or symlink change. **PASS.**
- **P0 | XI (Governance via Proposals)** — no local pre-rejection; the manifest constrains the record's own membership, never a change set. The held-out validation scenario still verifies this independently. **PASS.**

---

## P2 Considerations

- **P2-1 | interface-spec.md § Surface (anatomy row 5)** — the rule that fact IDs are *"allocated monotonically and never reused"* is stated as contract but **nothing enforces it and nothing discloses that it is unenforced**. The guard cannot know which IDs were previously used (that history lives in git and DEPRECATION.md), so this is process discipline wearing a contract's clothing. Consequence if broken: a reused `CSG-1` gives two different meanings to one permanent handle, and DEPRECATION.md entries become ambiguous. Low likelihood, contained impact. Fix: state the rule as unenforced-by-design in the same place it is declared — the same "no silent caps" discipline the guard's partial-coverage note already follows.

---

## Governance Notes

Informational — what is not being checked, and why. Not findings.

- **Constitution VI (Size-Aware by Design)**: no applicable checks — the feature performs no API reads and handles no result sets. Its "no silent caps" analogue is checked under III rather than double-counted here.
- **Constitution X (Respect API Limits)**: no applicable checks — no API requests, so no `429` handling or `If-Match` surface.
- **Constitution XII (Standalone Executable)**: no applicable checks — the record is a plugin file, not part of the compiled artifact. The symlink host-environment caveat remains plan risk R3 and transfers to the first spec shipping a second skill-consumer.
- **done-* accords**: none deployed. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks across the pipeline.
