# Checklist: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/unequipped-agent-operators/operating-surface-self-containment.feature
**Checks**: 19 (19 pass, 0 fail)
**Generated**: 2026-08-09 (run 2 — re-derived after the run-1 findings were addressed)

---

## Summary

All 19 checks pass. Constitution: 19/19. Done-criteria: not run (no accords). Cross-references: not generated (see Governance Notes).

---

## Changes Since Previous Run

**Previous (run 1)**: 2 P0, 0 P1, 0 P2 (2 failures)
**Current (run 2)**: 0 P0, 0 P1, 0 P2 (0 failures)

**Resolved**:
- ~~P0 | CONSTITUTION.md IV: "User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it." → **tasks.md § Phase 2, T005**: the godog suite binding the feature file depended on the task that wrote the code it verifies, so every acceptance scenario became executable only afterwards; Phase 1 had no executable scenario at all, resting on a manual grep.~~ → **fixed**. The scenarios were re-partitioned by what they read: the surface-content scenario ("Handoffs name in-plugin components") moved into Phase 1 as part of T002, whose acceptance now requires the step definitions to be written before the sweep of the file they read, observed red, then green. The seven fixture-driven detection scenarios stay with the guard implementation in T004, whose acceptance likewise requires them red before the implementation that satisfies them. The manual grep is demoted in wording and in fact — it is now labelled a completeness *aid*, with T002's scenario and T004's live tripwire as the phase's actual evidence.
- ~~P0 | CONSTITUTION.md VII: "…No code-only or test-only increments (except a RED test immediately followed by its GREEN implementation)." → **tasks.md § Phase 2, T005**: step-definitions-only task landing after the implementation.~~ → **fixed**. The separate binding task is gone (6 tasks → 5). Each remaining task pairs its own implementation with the tests that prove it: T002 = sweep + the scenario that verifies it; T004 = walker/lexicon/guard + the detection scenarios and fixture tests. T005 (constitution amendment) is a governance-document change, to which the code/test pairing criterion does not apply.

**Companion changes made for artifact agreement** (not themselves checklist findings): plan.md gained a *Scenario execution is test-first within each phase* paragraph in Cross-cutting Concerns and revised Phase 1/Phase 2 descriptions, so the plan and tasks state the same decomposition rather than leaving tasks.md to introduce it alone. tasks.md also gained a top-level per-scenario disposition table (13 scenarios: 8 executed, 5 held for validate, 0 inexecutable), matching the sibling records' convention.

**New failures**: none. No check that passed in run 1 fails in run 2.

---

## Constitution Checks: 19/19 passed

- **I. Spec Fidelity** (2/2): No artifact introduces, removes, or alters a CLI command or API request shape (spec.md non-behavior 1; plan.md System Architecture "nothing here adds capability"). Every command leaf in the swept registries still resolves to a shipped CLI command — T001's acceptance pins non-comment registry lines byte-identical.
- **II. Action Transparency** (2/2): The guard's failure output carries cause and next step — `plugin/<path>:<line>: forbidden reference "<text>" (family …: <property>). Remedy: …` (interface-spec.md § Error Communication), with the remedy's reachability checked against sibling conditions. The sweep preserves operator-facing guidance: T001 keeps each header's editing rule and fail-closed consequence; T002 rewords references without dropping deferrals.
- **III. Fail Safe, Not Silent** (2/2): No skip path, no warning tier, no partial pass; a walk error is a failure, not a skip (interface-spec.md § Error Communication "Degradation: none"; plan.md § Cross-cutting Concerns). A missing or empty surface fails loudly rather than passing vacuously (spec.md Behavioral Accord; interface condition 4; T004 acceptance).
- **IV. Test-Driven Development** (2/2): Every user-facing behavior has an executable acceptance scenario before the code that satisfies it — T002's surface-content scenario is written red against the unswept artifacts and goes green when the sweep lands; T004's detection scenarios and fixture tests are observed red before the walker and lexicon exist, with the live tripwire last (plan.md § Cross-cutting Concerns, § Implementation Strategy; tasks.md T002/T004 acceptance).
- **V. Composition over Monolith** (2/2): Walker and lexicon live in one production source (`surfaceselfcontainment.go`) shared by the guard test and the BDD suite — no duplicate implementation (plan ADR-1; interface §2). The new guard forces no edits to unrelated guards; T001/T003 touch only guards that pin text the sweep itself changes.
- **VI. Size-Aware by Design** (1/1): The guard reports every violation in a run rather than truncating to the first (plan § Guard Design; interface § Error Communication "one line per violation; the test failure aggregates all of them plus a count"; T004 acceptance).
- **VII. Working Software** (2/2): No code-only or test-only increment — the separate step-definitions task is gone; T002 and T004 each pair implementation with the tests that prove it. Every code-bearing task's acceptance names a green gate: T001–T004 name `go test ./...`, T004 adds `gofmt -l .` and `golangci-lint run`, and T005 (document-only) names the pre-commit hooks.
- **VIII. No Fabricated Data** (2/2): Every replacement name the sweep introduces resolves to a shipped in-surface artifact — plan.md § Sweep Design maps each id to an existing path, and the guard's positive check requires `plugin/…` paths to resolve. The guard reports only text it actually matched (interface grammar quotes `<matched text>`).
- **IX. Writes Require Explicit Intent** (2/2): The guard never mutates the surface; error and edge scenarios run against `t.TempDir()` fixtures (T004 scope; plan § Guard Design). The sweep leaves the gated-command registry's membership byte-identical (T001 acceptance).
- **XI. Governance via Proposals** (1/1): The write-gate's posture is unchanged — no command enters or leaves the gated set (spec.md non-behavior 1; T001).
- **XII. Standalone Executable** (1/1): No new runtime dependency; the guard is a stdlib directory walk in `internal/build` (plan § Cross-cutting Concerns).

---

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` files exist in this repository. See Governance Notes.

---

## Cross-Reference Checks

Not generated. Cross-reference checks derive from done-* accords that require references between artifacts; with no accords present, link-presence verification for this feature is left to `/score:analyze` (horizontal consistency is analyze's domain, not checklist's).

---

## Governance Notes

- **No `accords/` directory**: `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` are all absent. Consider creating `accords/governance/done-<skill>.md` to enable per-artifact quality checks — without them, checklist coverage is constitution-only and the pipeline's own done-criteria (which several Score skills cite in their Quality Gates) are unenforced across every spec in this repo, not just this one.
- **Constitution has no SHOULD-tier principles**: all twelve are MUST/MUST NOT, so mechanical severity inheritance yields P0 for every check. The absence of P1/P2 findings reflects the source's structure, not an unusually clean artifact set — and it means run 1's two findings were necessarily P0, with no lower tier available to express "structural deviation, not a hazard."
- **Principle X (Respect API Limits)**: no applicable checks for this feature — it performs no API interaction.
- **Principle XII analogy not checked**: the feature's subject (a plugin that must work with only the CLI present) rhymes with XII's standalone-executable requirement, but XII governs the CLI binary's runtime dependencies. The plugin-side rule is this feature's own Principle XIII, which does not yet exist in CONSTITUTION.md — T005 adds it. Checks against XIII will be available to future specs once it lands.
