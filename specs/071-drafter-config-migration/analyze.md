# Analyze: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/no-automated-pipeline/drafter-config-contract.feature, tasks.md
**Checklist context**: checklist.md present, round 2 (7 P0 pass, 1 P1 finding, 3 P2 considerations) — correlated, not re-evaluated
**Findings**: 16 checks (15 pass, 1 fail)
**Generated**: 2026-08-03 (round 2 — re-derived after A1/A3 were resolved)

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 6 | 6 | 0 |
| Completeness | P1 | 6 | 5 | 1 |
| Coherence | P2 | 4 | 4 | 0 |
| **Total** | | **16** | **15** | **1** |

**Improvement summary** — round 1: 1 P0 (C5), 1 P1 (K5), 1 P2 (H3). Round 2: 1 P1. **Resolved**: C5 and H3, which shared one root — `tasks.md` T004 added a godog runner no upstream artifact described. `plan.md` now grounds it in ADR-8, describes it in Phase 2 and the Testing strategy, and `interface-spec.md`'s Consistency Notes acknowledges the layer. **Remaining**: K5, untouched by that change.

The one remaining failure is independent of the runner: the scenario set is thinner than the error contract the interface defines.

---

## Consistency: 6/6 passed (P0)

- **C1 | spec Integration Boundaries ↔ plan System Architecture** — the plan's components (`.github/release-drafter.yml`, `labelcontract.go`, `drafterschema.go`, the reused `workflow.go` types, `labeler.yml`/`settings.yml`) map onto the spec's seven named boundaries, including the two the spec adds over 030: the drafting workflow definition as a new guard input, and the post-merge drafting run as an explicitly unguarded observation point. **PASS.**
- **C2 | spec Behavioral Accord ↔ plan System Architecture** — all five accord groups (drafting outcomes preserved, deprecation surface, contract guard, schema/version coupling, documentation) have architectural realization. The accord's "reads the real config files from the repository, not only a fixture" is served by plan Cross-cutting > Testing strategy. **PASS.**
- **C3 | spec Non-Behaviors ↔ plan System Architecture** — the plan architects nothing the spec excludes. It states "**What is deliberately not built.** Nothing evaluates drafter output," which is the load-bearing one; the dependabot major-bump policy and the `dependabot.yml` comment correction are both parked in "What This Plan Does Not Cover" rather than designed. **PASS.**
- **C4 | plan Architecture Decisions ↔ interface-spec Surface** — all seven ADRs are reflected: hard-switch (no dual-shape fields read for contract), the condition-less fallback entry, `drafterschema.go` as a sibling, the rejection-detector fields, the derived major plus named floor, `exclusive` omitted everywhere, and `DrafterWhen` in mapping form only. **PASS.**
- **C5 | plan System Architecture ↔ tasks Task Scope** — **PASS** (round 2; failed in round 1 as A1). Every task maps to a described component: T001 to the config, the pinned version, and `labelcontract.go`; T002 to its fixtures; T003 to `drafterschema.go`; T004 to the godog suite the plan now describes in ADR-8, Phase 2, and the Testing strategy; T005 to the Phase 3 sweep.
- **C6 | interface-spec Surface ↔ feature Given/When/Then** — scenario steps reference surfaces the interface defines: the superseded top-level keys, the condition-less `version-resolver` category, `no-release-note`, the pinned ref and its SHA form, the schema floor. **PASS.** *Nuance, not a finding*: the `@validation` scenario "The change claims no fix for the untagged-release failure" asserts about the pull-request description, which is a review artifact rather than an interface surface. This matches the meta-level shape of `@validation` scenarios elsewhere in the repo (030's "Every release-note category traces to a managed PR Administration label" likewise reaches outside its interface), so it is not read as an undefined-surface reference. It is, however, one of the scenarios T004 now holds — for `/score:validate` primarily, and because the PR half is unobservable at test time secondarily.

---

## Completeness: 5/6 passed (P1)

- **K1 | spec Driving Scenarios → feature file** — all 7 driving scenarios and all 4 validation scenarios have Gherkin equivalents, each `# Source:`-traced. 11 traced, plus 2 architecture-informed. **PASS.**
- **K2 | spec Integration Boundaries → interface file presence** — all seven boundaries fold into the single Specification touchpoint and each has coverage in `interface-spec.md` (the action and the drafting run in the Invocation table, the workflow and both guard files in Surface, the 028 labels in the config structure, `labeler.yml`/`settings.yml` in the exported contract, and the 030 artifacts in Consistency Notes). **PASS.**
- **K3 | plan phases → tasks entries** — Phase 1 → T001/T002, Phase 2 → T003/T004, Phase 3 → T005. Every phase decomposed. **PASS.**
- **K4 | plan components → tasks Scope fields** — every component has an implementing task; `labeler.yml`/`settings.yml` correctly have none, since the plan states they do not change. **PASS.**
- **K5 | interface-spec Surface → feature scenario coverage** — **FAIL.** See finding A2.
- **K6 | spec User Scenarios → interface Surface** — all three user scenarios have interface coverage: schema currency (the target structure plus the forbidden-field lists), positions-only movement (the position table and the byte-identical display-text rule), and the coupling verdict (the `drafterschema.go` contract). **PASS.**

---

## Coherence: 4/4 passed (P2)

- **H1 | Terminology** — the load-bearing concepts hold their names across the set: "superseded schema" / "current schema", "the label-contract guard" / "the coupling guard", "the condition-less version-resolver category", "the schema floor". **PASS.** *Nuance, not a finding*: `spec.md` deliberately avoids version numerals (it says "release-drafter's new major"), while `plan.md`, `tasks.md`, and the feature file use `v6.4.0`/`v7.7.0` directly. The two vocabularies are bridged in exactly one place — `DrafterSchemaMinMajor`'s definition in `interface-spec.md`, which states that `7` is "the lowest action major that understands the config schema this feature adopts." That single bridge holds, but it is the only one; if that constant's comment is ever thinned, the spec's version-agnostic prose loses its tie to the numerals downstream.
- **H2 | Detail symmetry** — no artifact carries 3x+ the detail of its neighbour on a shared topic. Topics absent from `spec.md` (the `when` list-form handling, the `dependabot.yml` comment defect) are absent correctly — they are not behavior. **PASS.**
- **H3 | Scope alignment (spec + interface + tasks)** — **PASS** (round 2; failed in round 1 as A3). `interface-spec.md`'s Consistency Notes now records the acceptance suite as a layer above both guards that defines no surface of its own, so the capability T004 introduces is no longer present in only one artifact of three. `spec.md` still does not mention it, correctly — a test-execution mechanism is not feature scope, and what the spec does own on this subject (the limit on runtime verification, in Non-Behaviors) is exactly what the suite's hold set respects.
- **H4 | Phase coverage (plan ↔ tasks)** — the three phase names match the plan's verbatim, and the dependency structure matches (Phase 2 depends on Phase 1 for its precondition; Phase 3 depends on both). No task references a phase the plan doesn't define. **PASS.**

---

## Findings

### A1 | P0 | Consistency C5 — RESOLVED in round 2

**Was**: `tasks.md` T004 added a godog runner over `drafter-config-contract.feature`, while `plan.md` described this feature's verification entirely as Go table tests and its Phase 2 covered only `drafterschema.go`. The task disclosed the divergence; disclosure is not consistency, and a Builder reading the plan for architecture and the tasks for work got two different answers about what verifies this feature.

**Resolution applied**: `plan.md` gained ADR-8 (wire the runner, with two distinct hold-out classes), its Phase 2 was renamed and extended to cover the suite, its System Architecture and Cross-cutting > Testing strategy describe the layer, and `tasks.md` Phase 2's heading and dependency-graph line were updated to match. The ADR records that this is conformance to `release_bdd_test.go`'s existing pattern rather than a new mechanism.

**Verified**: C5 passes — every task maps to a component the plan describes, and H4 still holds because both phase headings moved together.

### A2 | P1 | Completeness K5 — four interface-defined failure conditions have no scenario coverage

**Artifacts**: `interface-spec.md` Error Communication ↔ `features/no-automated-pipeline/drafter-config-contract.feature`.

`interface-spec.md` defines ten failure conditions. Six have Gherkin coverage. Four do not:

| Interface condition | Scenario? | Task coverage? |
|---|---|---|
| A `version-resolver` bucket drifts off its single semver label | none | T002 drift case |
| `no-release-note` absent from the `pre-exclude` category | none | T002 drift case |
| No step in the workflow uses the drafter action | none | T003 drift case |
| `when` written in the list form → parse error, not a violation | none | none |

The first three are the notable ones: `tasks.md` explicitly requires a drift case for each, so the Gherkin layer is thinner than both the interface above it and the tasks below it. A Builder tracing coverage top-down finds the contract, finds no scenario, and has to reach the tasks to learn the case is in fact covered.

The fourth has no coverage anywhere. That may be deliberate — ADR-7 treats a list-form config as a parse-level failure whose message shape differs from the others, and `interface-spec.md` says so — but nothing records the decision not to cover it.

*Note on the origin*: `spec.md` has no driving scenario for any of the four, so the gap opened between spec and interface rather than being introduced by the scenarios step. Adding scenarios without a spec source would be scope expansion; the cleaner resolution is either a spec addition or an explicit "covered by unit drift cases, not scenarios" note in the interface's Error Communication table.

### A3 | P2 | Coherence H3 — RESOLVED in round 2

**Was**: same root as A1, evaluated across spec + interface + tasks. The BDD execution layer appeared in `tasks.md` alone; `interface-spec.md`'s Consistency Notes described the package's verification idiom as "the same real-file-plus-fixture test pairing" with no third mode.

**Resolution applied**: `interface-spec.md`'s Consistency Notes now records the suite as a layer above both guards that defines no surface of its own, states that it adds no assurance beyond the guards, and gives the five-execute / eight-hold split. `spec.md` deliberately still does not mention it — a test-execution mechanism is not feature scope, and the spec's Non-Behaviors already own the part that is (the limit on runtime verification), which is precisely what the hold set respects.

**Verified**: H3 passes.

---

## Checklist Correlation

| Analyze finding | Checklist finding | Relationship |
|---|---|---|
| A1 (resolved) | F1 (resolved) | Same artifact (`tasks.md` T004), different defects, resolved by the same round of edits — the plan gained ADR-8 (closing A1) and T004 gained a per-scenario table (closing F1). Neither fix alone would have closed both. |
| A3 (resolved) | F1 (resolved) | Same root as A1. |
| A2 (P1, K5) | — | No checklist overlap. Horizontal gap only, and untouched by the round-2 edits. |
| — | F2 (P1, III) | No analyze counterpart. F2 concerns a spec accord with no detection mechanism — the matrix has no check for "accord bullet → verification", so the vertical pass is the only one that could catch it. Round 2 sharpened rather than resolved it: the scenario that would have observed the deprecation surface is now explicitly in T004's hold set, which makes the absence of detection visible rather than implicit. |

The three P2 considerations in checklist.md (one-directional guard, T002 and now T004 as test-only units, `DrafterWhen` declining `StringOrSlice`) are vertical judgments with no cross-artifact dimension; none is contradicted by any check above. Note that checklist P2-2 widened in round 2 — adding the suite added a second test-only task — which is a consequence of the A1 fix, not a new finding.

---

## Governance Notes

- No checks were skipped. The full artifact set is present: spec.md, plan.md, one interface file, one feature file, tasks.md, checklist.md.
- Check count is 16 — the base matrix, unscaled, since there is exactly one interface file and one feature file.
