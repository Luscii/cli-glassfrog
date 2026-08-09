# Tasks: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/operating-surface-self-containment.feature

---

## Dependency Graph

Phase 1: Conformance sweep + assertion re-derivation (3 tasks, no dependencies) [US1]
Phase 2: Guard + constitution (2 tasks, T004 depends on Phase 1 for its live tripwire; T005 independent) [US2/US3]

5 tasks total | Phase 1's three tasks parallelizable; T005 parallel with everything | Builder: implement skill (BDD outer loop)

T001, T002, T003, and T005 are all unblocked. T004 is the only gated task, and only because its live pass over the real surface would be red until the sweep lands — its fixture-driven scenarios depend on nothing Phase 1 does.

**Phase 1 completeness aid** (plan Risk 4): running the interface accord's Family A/B patterns as a local grep over `plugin/` is a fast way to spot a straggler before T004 lands. It is an aid, not the phase's evidence — T002 binds an executable scenario against the real swept surface, and T004's live tripwire is the standing check.

---

## Scenario Disposition

Every scenario in `features/unequipped-agent-operators/operating-surface-self-containment.feature`, and which task executes it. Driven by tag, never by count.

| Scenario | Disposition |
|---|---|
| Handoffs name in-plugin components | executed by T002 (reads the real swept surface) |
| A conforming surface passes verification | executed by T004 (fixture) |
| A future surface file is checked without registration | executed by T004 (fixture) |
| A spec-number reference fails the run | executed by T004 (fixture) |
| A pathless repository mention fails the run | executed by T004 (fixture) |
| An empty surface fails rather than passes | executed by T004 (fixture) |
| A dangling in-surface path fails the run | executed by T004 (fixture) |
| Known-safe tokens do not trip the check | executed by T004 (fixture) |
| The sweep preserved every handoff | **held** — `@validation`, for `/score:validate` |
| The surface reads lightweight and workflow-oriented | **held** — `@validation`, for `/score:validate` |
| Re-derived assertions kept the property | **held** — `@validation`, for `/score:validate` |
| Pointer direction is intact | **held** — `@validation`, for `/score:validate` |
| The constitutional principle is in house form | **held** — `@validation`, for `/score:validate` |

Totals: **8 executed** (T002 1, T004 7), **5 held for validate**, **0 inexecutable**.

---

## Branching Guidance

**Role-based mode** (074 and 075-legacy-identifier-request are in flight in parallel sessions): `spec/076-operating-surface-self-containment/base` as integration point, cut from a freshly fetched `origin/main` (main has moved since this workspace branched: the 074 implementation and the pre-commit config landed). Task branches: `spec/076-operating-surface-self-containment/task-N`. On any rebase, change only this spec's STATUS.md row and take main's side for the rest.

**Note for every Phase 1 task**: commits run under the newly landed pre-commit config — if a whitespace/EOF hook rewrites a swept plugin file, confirm the content-pinning guards still pass before judging the change unrelated.

---

## Phase 1: Conformance sweep + assertion re-derivation [US1]

- [x] **T001** [US1] [P] Sweep the write-gate artifacts and single-source registry headers — 0 scenarios (header-only sweep); no guard expectations pinned the removed text, so no internal/build changes were needed
  - **Scope**: `plugin/hooks/gated-commands.txt`, `plugin/hooks/glassfrog-write-gate.sh`, and the six `plugin/agents/*-commands.txt` / `*-reads.txt` registries — rewrite headers to the interface accord's instructional contract: keep what the file is, the in-plugin consumer list by `plugin/`-relative path, the editing rule, and the fail-closed consequence stated through the gate's own behavior; delete guard-test pointer lines, plan-ADR citations, spec-number ids, and repo-machinery consequences. Adjust any `internal/build` guard expectations that pin the removed header text, in the same commit.
  - **Acceptance criteria**:
    - The deny-lexicon grep (interface Family A + B) over these eight files returns zero matches
    - Each header still names every in-plugin consumer and states the editing rule and fail-closed consequence
    - Non-comment registry lines (the command leaves) are byte-identical to before
    - `go test ./...` green; `gofmt -l .` clean if Go files were touched
  - **Dependencies**: None
  - **Plan reference**: Phase 1, Sweep Design (registry headers, hook script)
  - **Interface references**: interface-spec.md §3 Registry-header instructional contract, §1 deny lexicon

- [x] **T002** [US1] Sweep the skill and agent artifacts; re-derive the number-pinned assertions; open the godog suite — 1 scenario (RED observed pre-sweep, GREEN post-sweep, @wip removed); 13 assertions re-derived across five suites; the six sibling feature files carried no number-pinned step text
  - **Scope**: eleven surface files — six of the eight `plugin/skills/*/SKILL.md` (constraint-discovery, governance-navigation, tension-processing, proposal-drafting, proposal-circulation, proposal-impact-review) and five of the six `plugin/agents/*.md` (governance-navigator, tension-processor, proposal-drafter, proposal-circulator, proposal-impact-reviewer). The three files the globs also match — `plugin/skills/orientation/SKILL.md`, `plugin/skills/glassfrog-setup/SKILL.md`, and `plugin/agents/constraint-navigator.md` — are deliberately out of scope: each already carries zero development-repository references, so the sweep has nothing to change in them. They are still checked, by T004's guard, which walks the whole surface. Within the eleven: drop spec-number parentheticals where the prose name already stands, substitute the plan's Sweep Design name where a number stands alone, and reword any contrast sentence the drop strands (word-boundary matching). In the same commits, re-derive every assertion in the `internal/build` BDD suites and the six existing feature files under `features/unequipped-agent-operators/` that pinned a spec number in surface text, to pin the in-plugin component name instead. Create `internal/build/surface_self_containment_bdd_test.go` — this spec's own godog suite, pointed at only this spec's feature file — and bind the surface-content scenario, written first so it is red against the unswept artifacts and green when the sweep lands.
  - **Acceptance criteria**:
    - The step definitions for "Handoffs name in-plugin components" are written before the sweep of the file they read, observed red, then green (RED→GREEN within the task)
    - `@wip` is removed from that scenario once green; the suite's `Paths` names only this spec's feature file
    - The deny-lexicon grep over the eleven surface files returns zero matches
    - Every re-derived assertion pins an in-plugin name from the Sweep Design mapping; none is deleted or weakened to presence-of-any-text (spec non-behavior 6)
    - Whitespace-collapsed matching in the BDD helpers is unchanged — only needles change
    - `go test ./...` green at every commit; `gofmt -l .` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1, Sweep Design (number classes), ADR-3: name-pins, Cross-cutting Concerns (scenario execution is test-first within each phase)
  - **Scenario references**: operating-surface-self-containment.feature: "Scenario: Handoffs name in-plugin components"
  - **Risk**: ⚠️ Sequencing — the suite and the scenario it binds must precede the sweep of the artifact it reads, or the RED half is unobservable

- [x] **T003** [US1] [P] Sweep the drafting reference records' provenance lines — 0 scenarios; record guards pin only field presence, so the adjusted expectations were the two BDD step defs (vendored-spec path, portfolio-memory name) with their step text
  - **Scope**: `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` and `plugin/skills/proposal-drafting/references/circle-routing-rule.md` — remove the spec-number and portfolio-document provenance citations; adjust the two `internal/build` record guards' pinned expectations in the same commit. Record content is otherwise byte-identical.
  - **Acceptance criteria**:
    - The deny-lexicon grep over both records returns zero matches
    - The grammar-facts `Live facts` manifest and both records' fact/rule content are unchanged
    - Both record guards and the grammar/routing BDD suites are green
  - **Dependencies**: None
  - **Plan reference**: Phase 1, Sweep Design (reference-file provenance), Risk 3
  - **Risk**: ⚠️ Content-pinning collision — the record guards pin surface text the sweep edits; update guard expectations in-phase, never relax the records

## Phase 2: Guard + constitution [US2/US3]

- [x] **T004** [US2] Build the self-containment walker, lexicon, guard test, and its detection scenarios — 7 scenarios (each observed red against a stubbed scan, then green; @wip removed); live tripwire green over the real plugin/
  - **Scope**: new `internal/build/surfaceselfcontainment.go` (walker over `plugin/` from the repo root, two-family deny lexicon with per-entry property comment + example, scan producing violations, `plugin/…`-path resolution check) and `internal/build/surface_self_containment_guard_test.go` (fixture unit tests + the live tripwire), plus the seven detection scenarios' step definitions added to the suite T002 opened — driving these production functions against `t.TempDir()` fixture surfaces only, never mutating the real plugin. TDD inside the task: fixture tests and step definitions written first (red on seeded Family A/B violations, a dangling path, an empty surface, and each known-safe token class), implementation second, live tripwire last.
  - **Acceptance criteria**:
    - Every detection scenario's step definitions are observed red before the implementation that satisfies them, then green; `@wip` removed from each as it passes; the five `@validation` scenarios keep their tags (driven by tag, never by count)
    - Violation report matches the interface grammar: `plugin/<path>:<line>: forbidden reference "<text>" (family …: <property>). Remedy: …` — all violations in one run, not first-only
    - The walked set is derived (no file enumeration); a file added to a fixture surface is caught with no registration
    - Zero files or missing surface fails loudly; no skip path, no warning tier
    - Every known-safe token fixture passes (`prp_0123`, `0.34.1`, exit codes, `--per-page 500`, "the v5 spec", "CLI")
    - The live tripwire over the real `plugin/` is green; `go test ./...`, `gofmt -l .`, `golangci-lint run` clean
  - **Dependencies**: T001, T002, T003 — for the live tripwire only; the fixture-driven work needs none of them. T002 owns the suite file this task extends.
  - **Plan reference**: Phase 2, Guard Design, ADR-1, ADR-2, Cross-cutting Concerns (testing strategy)
  - **Interface references**: interface-spec.md §1 deny lexicon, §2 guard surface, Error Communication condition table
  - **Scenario references**: operating-surface-self-containment.feature: "A conforming surface passes verification", "A future surface file is checked without registration", "A spec-number reference fails the run", "A pathless repository mention fails the run", "An empty surface fails rather than passes", "A dangling in-surface path fails the run", "Known-safe tokens do not trip the check"

- [x] **T005** [US3] [P] Amend CONSTITUTION.md: Principle XIII + version marker — 0 executed scenarios (its scenario is @validation-held); statement/Rationale/Detection in sibling form, version marker established at 1.1 with adjacent justification
  - **Scope**: add `### XIII. Self-Contained Operating Surface` after XII in house form (bold statement per interface §4, *Rationale*, *Detection* naming the verification-run observable); establish the version marker in the Governance section (initial `1.0` acknowledging the pre-existing twelve principles, bumped to `1.1` by this amendment) with the justification recorded adjacent.
  - **Acceptance criteria**:
    - Principle XIII carries all three parts in sibling form
    - The version marker exists, reads `1.1`, and the adjacent justification names this amendment and date
    - No other principle's text changed
    - The repository's pre-commit hooks pass on the amended document
  - **Dependencies**: None (parallel with every other task — different file)
  - **Plan reference**: Phase 2, ADR-4
  - **Interface references**: interface-spec.md §4 Constitution Principle XIII structural contract
  - **Scenario references**: operating-surface-self-containment.feature: "The constitutional principle is in house form" (held for `/score:validate`)
