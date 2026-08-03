# Analyze: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/no-automated-pipeline/drafter-config-contract.feature, tasks.md
**Checklist context**: checklist.md present (1 P0 fail, 1 P1 fail, 6 P0 pass, 3 P2 considerations) — correlated, not re-evaluated
**Findings**: 16 checks (13 pass, 3 fail)
**Generated**: 2026-08-03

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 6 | 5 | 1 |
| Completeness | P1 | 6 | 5 | 1 |
| Coherence | P2 | 4 | 3 | 1 |
| **Total** | | **16** | **13** | **3** |

Two of the three failures (C5, H3) share one root: tasks.md T004 adds a godog runner that no upstream artifact describes. The third (K5) is independent — the scenario set is thinner than the error contract the interface defines.

---

## Consistency: 5/6 passed (P0)

- **C1 | spec Integration Boundaries ↔ plan System Architecture** — the plan's components (`.github/release-drafter.yml`, `labelcontract.go`, `drafterschema.go`, the reused `workflow.go` types, `labeler.yml`/`settings.yml`) map onto the spec's seven named boundaries, including the two the spec adds over 030: the drafting workflow definition as a new guard input, and the post-merge drafting run as an explicitly unguarded observation point. **PASS.**
- **C2 | spec Behavioral Accord ↔ plan System Architecture** — all five accord groups (drafting outcomes preserved, deprecation surface, contract guard, schema/version coupling, documentation) have architectural realization. The accord's "reads the real config files from the repository, not only a fixture" is served by plan Cross-cutting > Testing strategy. **PASS.**
- **C3 | spec Non-Behaviors ↔ plan System Architecture** — the plan architects nothing the spec excludes. It states "**What is deliberately not built.** Nothing evaluates drafter output," which is the load-bearing one; the dependabot major-bump policy and the `dependabot.yml` comment correction are both parked in "What This Plan Does Not Cover" rather than designed. **PASS.**
- **C4 | plan Architecture Decisions ↔ interface-spec Surface** — all seven ADRs are reflected: hard-switch (no dual-shape fields read for contract), the condition-less fallback entry, `drafterschema.go` as a sibling, the rejection-detector fields, the derived major plus named floor, `exclusive` omitted everywhere, and `DrafterWhen` in mapping form only. **PASS.**
- **C5 | plan System Architecture ↔ tasks Task Scope** — **FAIL.** See finding A1.
- **C6 | interface-spec Surface ↔ feature Given/When/Then** — scenario steps reference surfaces the interface defines: the superseded top-level keys, the condition-less `version-resolver` category, `no-release-note`, the pinned ref and its SHA form, the schema floor. **PASS.** *Nuance, not a finding*: the `@validation` scenario "The change claims no fix for the untagged-release failure" asserts about the pull-request description, which is a review artifact rather than an interface surface. This matches the meta-level shape of `@validation` scenarios elsewhere in the repo (030's "Every release-note category traces to a managed PR Administration label" likewise reaches outside its interface), so it is not read as an undefined-surface reference. It is, however, part of why K5 and checklist F1 both land.

---

## Completeness: 5/6 passed (P1)

- **K1 | spec Driving Scenarios → feature file** — all 7 driving scenarios and all 4 validation scenarios have Gherkin equivalents, each `# Source:`-traced. 11 traced, plus 2 architecture-informed. **PASS.**
- **K2 | spec Integration Boundaries → interface file presence** — all seven boundaries fold into the single Specification touchpoint and each has coverage in `interface-spec.md` (the action and the drafting run in the Invocation table, the workflow and both guard files in Surface, the 028 labels in the config structure, `labeler.yml`/`settings.yml` in the exported contract, and the 030 artifacts in Consistency Notes). **PASS.**
- **K3 | plan phases → tasks entries** — Phase 1 → T001/T002, Phase 2 → T003/T004, Phase 3 → T005. Every phase decomposed. **PASS.**
- **K4 | plan components → tasks Scope fields** — every component has an implementing task; `labeler.yml`/`settings.yml` correctly have none, since the plan states they do not change. **PASS.**
- **K5 | interface-spec Surface → feature scenario coverage** — **FAIL.** See finding A2.
- **K6 | spec User Scenarios → interface Surface** — all three user scenarios have interface coverage: schema currency (the target structure plus the forbidden-field lists), positions-only movement (the position table and the byte-identical display-text rule), and the coupling verdict (the `drafterschema.go` contract). **PASS.**

---

## Coherence: 3/4 passed (P2)

- **H1 | Terminology** — the load-bearing concepts hold their names across the set: "superseded schema" / "current schema", "the label-contract guard" / "the coupling guard", "the condition-less version-resolver category", "the schema floor". **PASS.** *Nuance, not a finding*: `spec.md` deliberately avoids version numerals (it says "release-drafter's new major"), while `plan.md`, `tasks.md`, and the feature file use `v6.4.0`/`v7.7.0` directly. The two vocabularies are bridged in exactly one place — `DrafterSchemaMinMajor`'s definition in `interface-spec.md`, which states that `7` is "the lowest action major that understands the config schema this feature adopts." That single bridge holds, but it is the only one; if that constant's comment is ever thinned, the spec's version-agnostic prose loses its tie to the numerals downstream.
- **H2 | Detail symmetry** — no artifact carries 3x+ the detail of its neighbour on a shared topic. Topics absent from `spec.md` (the `when` list-form handling, the `dependabot.yml` comment defect) are absent correctly — they are not behavior. **PASS.**
- **H3 | Scope alignment (spec + interface + tasks)** — **FAIL.** See finding A3.
- **H4 | Phase coverage (plan ↔ tasks)** — the three phase names match the plan's verbatim, and the dependency structure matches (Phase 2 depends on Phase 1 for its precondition; Phase 3 depends on both). No task references a phase the plan doesn't define. **PASS.**

---

## Findings

### A1 | P0 | Consistency C5 — tasks.md T004 builds a component plan.md does not describe

**Artifacts**: `tasks.md` Phase 2 > T004 ↔ `plan.md` System Architecture, Cross-cutting Concerns > Testing strategy, Implementation Strategy > Phase 2.

`tasks.md` T004 adds a godog runner over `drafter-config-contract.feature` in `internal/build`. `plan.md` describes the feature's verification entirely as Go table tests — "a real-file change-detector … plus a table-driven drift suite over in-memory fixtures" — and its Phase 2 covers only `drafterschema.go` and its tests. The plan mentions godog nowhere; neither does the System Architecture diagram, which shows two guard files and their inputs.

T004 states its own status plainly ("**This task is not named in plan.md**"), so the divergence is disclosed rather than hidden. Disclosure does not resolve the inconsistency: a Builder reading `plan.md` for the architecture and `tasks.md` for the work gets two different answers about what verifies this feature, and the plan is the artifact downstream skills treat as the architectural source of truth.

Analyze does not determine which artifact is right. The two coherent resolutions are: enrich `plan.md`'s Phase 2 and Testing strategy to cover the runner and its tag boundary, or drop T004 and record the feature file's unexecuted status as a known state. Either removes the contradiction.

*Correlation*: checklist F1 (P0) flags a different defect **inside** T004 — that its acceptance criteria require executing scenarios that cannot be executed. The two are independent: F1 would remain even if the plan were enriched to cover the runner, and A1 would remain even if T004's execution list were corrected.

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

### A3 | P2 | Coherence H3 — the artifact set does not agree on what verifies this feature

**Artifacts**: `spec.md` Non-Behaviors ↔ `interface-spec.md` (whole) ↔ `tasks.md` T004.

Same root as A1, evaluated at the three-artifact scope rather than the plan↔tasks pair. `tasks.md` introduces a BDD execution layer that neither `spec.md` nor `interface-spec.md` mentions. `interface-spec.md`'s Consistency Notes describes the package's verification idiom as "the same real-file-plus-fixture test pairing" with no third mode. `spec.md`'s Non-Behaviors forecloses runtime verification but says nothing about whether the Gherkin is executed.

The drift is mild — a Builder would not be blocked — but the capability appears in exactly one artifact of three, which is the shape H3 exists to catch. Resolving A1 resolves this too.

---

## Checklist Correlation

| Analyze finding | Checklist finding | Relationship |
|---|---|---|
| A1 (P0, C5) | F1 (P0, IV) | Same artifact (`tasks.md` T004), different defects. F1: the execution list names scenarios that cannot be executed. A1: the runner itself is absent from the plan. Both must be resolved; fixing either leaves the other. |
| A3 (P2, H3) | F1 (P0, IV) | Same root as A1. |
| A2 (P1, K5) | — | No checklist overlap. Horizontal gap only. |
| — | F2 (P1, III) | No analyze counterpart. F2 concerns a spec accord with no detection mechanism — the matrix has no check for "accord bullet → verification", so the vertical pass is the only one that could catch it. Recorded here so the absence is not read as a passing check. |

The three P2 considerations in checklist.md (one-directional guard, T002 as a test-only unit, `DrafterWhen` declining `StringOrSlice`) are vertical judgments with no cross-artifact dimension; none is contradicted by any check above.

---

## Governance Notes

- No checks were skipped. The full artifact set is present: spec.md, plan.md, one interface file, one feature file, tasks.md, checklist.md.
- Check count is 16 — the base matrix, unscaled, since there is exactly one interface file and one feature file.
