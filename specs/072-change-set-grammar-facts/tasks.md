# Tasks: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/unguided-change-construction/change-set-grammar-facts.feature, features/unguided-change-construction/change-set-grammar-guard.feature

---

## Dependency Graph

Phase 1: The record + guard + supersession (3 tasks, no phase dependencies) — single-phase build

3 tasks total | 1 phase | Builder: implement skill (BDD outer loop)

T001 is unblocked. T002 and T003 each depend on T001 and are independent of each other.

---

## Branching Guidance

**Pipeline mode**: `spec/072-change-set-grammar-facts/base` → `spec/072-change-set-grammar-facts/task-1`, `…/task-2`, `…/task-3`. The three tasks are one coherent PR-sized change and T002/T003 both depend on T001, so a single branch in T-order is the natural shape; separate branches are available if T003's bookkeeping is worth isolating.

---

## Scenario Disposition

Per-scenario execution disposition across both feature files (16 scenarios). Tasks execute only the scenarios listed against them. The hold set splits by reason — `@validation`-held versus process-inexecutable — rather than being claimed as one group.

**`change-set-grammar-facts.feature`** (11 — what the record says):

| Scenario | Disposition |
|---|---|
| Own-circle policy shape is recorded with its symptom | executed by T001 |
| Self-targeting role update carries the accepted-but-invalid disposition | executed by T001 |
| An assembler finds both the shape to use and the shape to avoid | executed by T001 |
| A returned proposal id is not read as a valid governance change | executed by T001 |
| Contract-carried shapes appear as citations, never restatements | executed by T001 |
| Every recorded fact carries its full five-field contract | executed by T001 |
| Every fact is marked empirical, never contract | executed by **T002** — T001 supplies the marker, but the closing `And a record missing that marker will fail the guard` cannot pass until the guard exists, so the task that turns it green owns it |
| A contract-absorbed fact retires from the record | **inexecutable by automation** — asserts a future maintenance event against a real contract absorption that has not happened. Not unmechanized, though: the guard file's complete/partial retirement scenarios exercise the same flow against a fixture, and the supersession half is carried by the `@validation` scenario below |
| Only the two residual shapes are recorded as empirical facts | `@validation` — held for `/score:validate` |
| Nothing in the feature rejects a change set before the server sees it | `@validation` — held for `/score:validate` |
| The LEARNINGS copy is superseded, not left as a second source | `@validation` — held for `/score:validate` |

**`change-set-grammar-guard.feature`** (5 — what the guard enforces):

| Scenario | Disposition |
|---|---|
| A spec refresh that moves a cited anchor fails the build | executed by T002 |
| A structurally invalid fact fails the guard *(Outline, 2 examples)* | executed by T002 |
| A complete retirement passes the guard | executed by T002 |
| A partial retirement fails the guard *(Outline, 2 examples)* | executed by T002 |
| A record with no facts left fails as an empty shell | executed by T002 |

Totals: **12 executed** (T001 6, T002 6), **3 held for validate**, **1 process-inexecutable**.

---

## Phase 1: The record + guard + supersession

- [ ] **T001** [US1] Author the grammar-facts record under the proposal-drafting skill
  - **Scope**: Create `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` (the shipped plugin's first `references/` directory) with the full anatomy interface-spec.md pins, in order — empirical-marker blockquote, document header (Owner, Contract citations, and the `Live facts` manifest), contract-citations section, and the two fact sections. No other plugin file changes: drafting's SKILL.md and the drafter agent stay untouched, so the record ships inert and 067's validate-pinned prompt fences are unaffected.
  - **Acceptance criteria**:
    - The `Live facts` manifest declares `CSG-1, CSG-2` and set-equals the actual fact sections; fact sections are headed `## CSG-<n> — <title>`
    - CSG-1 carries all five fields; Disposition reads `accepted`; Shape states the top-level `CreatePolicy` part with no `UpdateRole` wrapper; Symptom carries the wrapped-shape refusal and the web-UI corroboration; Evidence cites `prp_ebe2815f…`; Provenance cites LEARNINGS 2026-08-05 F5
    - CSG-2 carries all five fields; Disposition reads `accepted-but-invalid`; Shape states the self-targeting `UpdateRole`; Symptom states `valid: false`, the blocking alert, empty `available_transitions`, and that a returned `prp_` id is not a successful governance change; Evidence cites `prp_c76cd6bf…`; Provenance cites LEARNINGS 2026-08-05 F6
    - Contract citations are by schema and property name only — no line numbers, no restated enum values beyond the citation lists, no pinned count
    - Step definitions land for this task's six scenarios (godog; content assertions against a whitespace-collapsed copy, structural checks against the raw file, per the operator-path BDD convention); `@wip` removed as each passes
  - **Dependencies**: None
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-1: proposal-drafting ownership, ADR-2: structured markdown form
  - **Scenario references**: change-set-grammar-facts.feature: "Own-circle policy shape is recorded with its symptom", "Self-targeting role update carries the accepted-but-invalid disposition", "An assembler finds both the shape to use and the shape to avoid", "A returned proposal id is not read as a valid governance change", "Contract-carried shapes appear as citations, never restatements", "Every recorded fact carries its full five-field contract"
  - **Interface references**: interface-spec.md § Surface — The record file (anatomy rows 1–5, per-fact contract, fact content at landing)

- [ ] **T002** [US2] Add the citation-integrity guard with its manifest and retirement invariants
  - **Scope**: `internal/build/grammarfacts.go` (record path constant; parsers deriving every side — the `Live facts` manifest, fact section IDs and titles, field labels, disposition values, cited type names, nested-only citation list, marker presence; and the spec side — enum values and the nested-only set from the `ProposalChange` description) plus `internal/build/grammarfacts_guard_test.go` asserting the eight conditions from interface-spec.md. Derivation helpers in production source, assertions in the test (family convention).
  - **Acceptance criteria**:
    - All **eight** violation conditions fail loudly, each naming the invariant, the offending element, **and which resolution path applies**: (1) manifest declares an ID with no section, (2) a section's ID absent from the manifest, (3) zero fact sections, (4) missing or empty required field, (5) Disposition outside the closed vocabulary, (6) cited change type absent from the spec enum, (7) nested-only citation not set-equal to the spec's set, (8) empirical marker absent or degraded
    - Conditions 1 and 2 are distinct directional checks, so a **partial** retirement fails while a **complete** one passes — verified by the complete/partial retirement scenarios, not by inspection
    - Every side is derived from its file at test time — no hard-coded enum values, type names, **or fact IDs** anywhere in the guard
    - The explicitly-partial residue (semantic absorption of CSG-2 is undetectable) is stated in the guard's comment, naming the refresh-diff review as its owner
    - Step definitions land for this task's six scenarios; `@wip` removed as each passes
    - `gofmt -l .` clean and `go test ./...` green before push
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-3: sibling guard, everything derived, manifest invariant
  - **Scenario references**: change-set-grammar-guard.feature: "A spec refresh that moves a cited anchor fails the build", "A structurally invalid fact fails the guard", "A complete retirement passes the guard", "A partial retirement fails the guard", "A record with no facts left fails as an empty shell"; change-set-grammar-facts.feature: "Every fact is marked empirical, never contract"
  - **Interface references**: interface-spec.md § Surface — Guard coupling; § Error Communication (conditions 1–8 with resolution paths)
  - **Risk**: ⚠️ Largest task in the decomposition — eight conditions, two parser sides, six scenarios. Deliberately not split: the natural seam (internal-consistency vs citation-integrity invariants) runs through the same two files, so splitting would produce two PRs touching the same pair.

- [ ] **T003** [US3] Record the LEARNINGS supersession via /score:deprecate
  - **Scope**: One `/score:deprecate` entry in `.score/memory/DEPRECATION.md` recording that LEARNINGS 2026-08-05 F5/F6 are superseded by the record at `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`. No edit to the LEARNINGS entry itself — it already carries the forward pointer, and git history is the changelog.
  - **Acceptance criteria**:
    - The entry names both superseded facts (F5, F6), the superseding record path and fact IDs (CSG-1, CSG-2), and the origin (072-change-set-grammar-facts)
    - The entry states the go-forward retirement mechanics as the deliberate three-part act — delete the fact section, drop its ID from the `Live facts` manifest, add the deprecation entry — plus the terminal case: when the last fact retires the record itself is deleted, not kept as an empty shell
    - Retired fact IDs are recorded as never reused, so the entry stands as the permanent handle for the shape
    - No second copy of the fact content is introduced anywhere
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 (Implementation Strategy), ADR-4: remove-and-deprecate retirement
  - **Scenario references**: change-set-grammar-facts.feature: "The LEARNINGS copy is superseded, not left as a second source" (`@validation` — held for validate, not executed by this task)
  - **Interface references**: interface-spec.md § Interactions — Maintenance flow
