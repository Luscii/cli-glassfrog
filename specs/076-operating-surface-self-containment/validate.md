# Validate: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Round**: 1 of 3
**Date**: 2026-08-09
**Verdict**: Issues
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, PROJECT.md, `features/unequipped-agent-operators/operating-surface-self-containment.feature`
**Implementation files**: 26 files under `plugin/` (21 swept), `internal/build/surfaceselfcontainment.go`, `internal/build/surface_self_containment_guard_test.go`, `internal/build/surface_self_containment_bdd_test.go`, 5 re-derived BDD suites, `CONSTITUTION.md`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✗ Unsatisfied (3 of 5) | 2 |

**Total**: 5 dimensions checked, 5 passed, 2 findings (both from held-out validation scenarios)

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

| Scenario | Status | Implementation |
|---|---|---|
| A conforming surface passes verification | ✓ Covered | `ScanOperatingSurface`; BDD `givenConformingSurface`, `TestScanOperatingSurfaceCleanFixture` |
| A handoff reads by name where the operator stands | ✓ Covered | BDD surface-content steps read the real swept surface; RED→GREEN observed pre/post sweep |
| A future file is covered without registration | ✓ Covered | `filepath.WalkDir` derives the set; `TestScanOperatingSurfaceDerivedWalk` |
| A spec-number reference turns the run red | ✓ Covered | Family A `\b0\d{2}\b`; report grammar with remedy |
| A repository mention fails even without a path | ✓ Covered | Family B patterns; verified empirically to match "a drift guard in the source repository" |
| Non-reference tokens do not false-positive | ✓ Covered | `knownSafeTokens` corpus, `TestSurfaceLexiconKnownSafeTokens` |
| An empty surface is a failure, not a pass | ✓ Covered | Zero-file and missing-directory both error "missing or empty"; `TestScanOperatingSurfaceEmptyOrMissing` |

Note: the "repository mention" scenario names three forms in its Given (test path, plan ADR citation, the phrase "the source repository's drift guard"). All three are matched. A fourth phrasing found live in the surface is not — see F-2; that gap is against the Behavioral Accord rule and the held-out scenarios, not against this driving scenario as written.

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks checked, all criteria verified independently)

| Task | Status | Verification |
|---|---|---|
| T001 registry/gate headers | ✓ Met | Deny-lexicon grep zero over the 8 files; non-comment leaves byte-identical (diff-verified per file); consumer lists, editing rules, fail-closed consequences present |
| T002 skill/agent sweep + re-derivation + suite | ✓ Met | Lexicon grep zero over the 11 files; 13 assertions re-derived; `normalizeWS` unchanged; suite `Paths` names only this feature file |
| T003 reference-record provenance | ✓ Met | Lexicon grep zero over both records; `Live facts` manifest and fact/rule bodies unchanged; record guards green |
| T004 walker, lexicon, guard, scenarios | ✓ Met | All violations reported per run (≥4 in the seeded fixture); walk-derived set; loud empty failure; known-safe fixtures; live tripwire green |
| T005 Principle XIII + version marker | ✓ Met | Three parts in sibling form; marker reads 1.1 with adjacent dated justification; diff shows additions only; `pre-commit run` passes on the amended document |

Full gate re-run at validation time: `go test ./...` 2332 passed, `gofmt -l .` clean, `golangci-lint run` 0 issues, `pre-commit run --from-ref origin/main --to-ref HEAD` all hooks passed.

---

## Interface Contract Conformance

**Status**: Pass (4 of 4 surfaces conformant)

| Surface | Status | Notes |
|---|---|---|
| §1 Deny lexicon (Family A + B) | ✓ Conformant | All 12 patterns present with the accord's exact RE2 syntax, case-sensitivity, and anchoring; each entry carries property + example; known-safe corpus pinned as fixtures |
| §2 Guard surface (3 files) | ✓ Conformant | Production source shared by guard and suite; live tripwire in the guard test; the suite's detection steps run only against `t.TempDir()` fixtures and never invoke the walker on the real `plugin/` |
| §3 Registry-header instructional contract | ✓ Conformant | All 7 artifacts keep what-the-file-is, in-plugin consumer list by `plugin/`-relative path, editing rule, and fail-closed consequence in surface terms; no guard paths, ADR citations, or "turns CI red" |
| §4 Constitution Principle XIII | ✓ Conformant | Heading after XII; bold statement, *Rationale*, *Detection* naming the observable; version marker 1.0→1.1 with adjacent justification |

**Observation (not a finding)**: the report grammar's family slot is elaborated. The accord writes `(family <A|B>: <property>)`; the implementation emits `(family A — resolvable reference: <property>)`. This is a strict superset — it still begins `family A` — and matches the accord's Interactions clause verbatim ("file, line, matched text, family, property, and remedy"). Recorded so a future reader does not read it as drift.

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions held)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not change what the surface does | ✓ Held | Gate script diff is 9 comment lines, zero executable lines; all 7 registries' non-comment leaves byte-identical |
| Must not rewrite any already-merged spec | ✓ Held | `git diff origin/main -- specs/` touches only 076's own tasks.md and STATUS.md |
| Must not restrict repo→surface references | ✓ Held | Every `plugin/…` constant in `internal/build` resolves; the only non-resolving strings are two deliberate fixture paths inside the guard test |
| Must not ban operating-world references | ✓ Held | Known-safe corpus (ids, versions, exit codes, `--per-page 500`, "the v5 spec", "CLI") pinned green; setup skill's install channels untouched |
| Must not enumerate the checked files | ✓ Held | `filepath.WalkDir` over `plugin/`; no file list anywhere in the guard |
| Must not satisfy a failing assertion by deletion | ✓ Held | All 13 re-derived assertions retain a component-name pin; none deleted, none reduced to presence-of-any-text |

**Observation (not a finding)** on the last row: the re-derivations took two shapes. Where the assertion already pinned the prose name beside the number (11 sites, e.g. `"Constraint Discovery Path"` AND `"065"`), the number conjunct was dropped, leaving one condition. Where the number was the only component pin (2 sites, `"response side"` AND `"069"`), the hyphenated in-plugin name `"proposal-impact-review"` was substituted. Both preserve the property the spec names ("the artifact defers by a name resolvable where the operator stands"); the first shape is a genuine reduction from two conditions to one, but the dropped conjunct pinned a repo id — precisely what the spec ordered removed.

---

## @wip Lifecycle Completion

**Status**: Pass

8 scenarios executed by checked tasks have their `@wip` removed. 5 `@wip` tags remain, all on `@validation`-tagged scenarios held out for this run — matching tasks.md's disposition table exactly (8 executed, 5 held, 0 inexecutable). Driven by tag, not by count.

---

## Validation Scenario Results

**Status**: Unsatisfied (3 of 5 scenarios traced)

| Scenario | Status | Trace |
|---|---|---|
| The sweep preserved every handoff | ✗ Unsatisfied | F-1 — two compound `(068/069)` references collapsed to a single component |
| Re-derived assertions kept the property | ✓ Satisfied | All 13 sites pin a component name; feature-file narrative and the `071` guard-output assertion correctly left untouched (repo-side, not surface content) |
| The surface reads lightweight and workflow-oriented | ✗ Unsatisfied | F-2 — one file still explains repository document ownership |
| The constitutional principle is in house form | ✓ Satisfied | Statement + *Rationale* + *Detection* mirror siblings I–XII; version bump and justification satisfy the Governance clause |
| Pointer direction is intact | ✗ Partially | Repo→surface direction verified intact (all guard path constants resolve). Surface→repo direction violated at one site — F-2 |

---

## Findings

### F-1: Two collapsed compound references misattribute the respond handoff

- **Dimension**: Validation scenario — "The sweep preserved every handoff"
- **Source**: spec.md § Behavioral Accord > The rule ("When a file hands work onward … it names the receiving in-surface component"); plan.md § Sweep Design (069 → the proposal-impact-review path, "includes every 'response side (069)' mention")
- **Implementation**: `plugin/skills/proposal-drafting/SKILL.md:31` and `:101`
- **Gap**: Both sites carried the compound `(068/069)` pre-sweep, where 068 pointed at the Proposal Circulation Path and 069 at the proposal-impact-review path. The sweep dropped the parenthetical without substituting the second component, so both sentences now attribute *responding* to the Proposal Circulation Path alone:
  - `:31` — "*advance, respond to, or withdraw a circulating proposal* — that is the **Proposal Circulation Path**"
  - `:101` — "It never runs `proposal propose`, `proposal respond`, or `proposal withdraw` — advancing, responding to, or withdrawing a proposal is the **Proposal Circulation Path**"

  This contradicts the surface's own fences: `plugin/skills/proposal-circulation/SKILL.md:105` states responding is "the **response side**, the proposal-impact-review path", and the sibling agent artifact `plugin/agents/proposal-drafter.md:29` handled the identical compound correctly ("the **Proposal Circulation Path** or the response side (the proposal-impact-review path)"). The handoff to the response side is dropped from the drafting skill entirely — the term appears nowhere else in that file, so there is no local binding to recover it from. Asymmetric treatment of the same construct across skill and agent.
- **Scope note**: `plugin/skills/proposal-drafting/SKILL.md:66` carries the same imprecision ("Advancing, responding to, or withdrawing … is that path's job") but carried `(068)` pre-sweep — that inaccuracy is pre-existing, not sweep-introduced, and non-behavior 1 keeps it out of this capability's scope.

### F-2: The setup skill still references the development repository

- **Dimension**: Validation scenarios — "The surface reads lightweight and workflow-oriented" / "Pointer direction is intact"
- **Source**: spec.md § Behavioral Accord > The rule ("it does not mention the development repository or its machinery — … no pathless prose"); spec.md § Validation Scenarios ("no surface file references the repository in any form"); CONSTITUTION.md Principle XIII ("The development repository MUST NOT be referenced in any form")
- **Implementation**: `plugin/skills/glassfrog-setup/SKILL.md:50`
- **Gap**: The line reads "The canonical invocations live in the repository's README and install guide — quoted here, owned there:". This is a pathless prose reference to the development repository that also asserts document ownership there — unusable knowledge for an operator who has only the plugin and the CLI, and the same "quoted here, owned there" provenance pattern T003 removed from the two drafting reference records. The three install commands beneath it are operating-world (a public raw URL, a brew tap, an npm package) and are correctly legal; only the provenance clause is in scope.

  Two compounding facts:
  1. **The guard does not catch it.** Verified empirically: the lexicon matches "a drift guard in the source repository" (Family B) but returns no match for "the repository's README and install guide" — Family B's pattern requires `source|development|parent` before `repositor(y|ies)`. This is plan Risk 1 ("a novel prose mention could evade it") materialized as a live violation rather than a hypothetical, so the live tripwire is green over a surface that violates the accord's central rule.
  2. **tasks.md scoped the file out on a false premise.** T002's scope states `plugin/skills/glassfrog-setup/SKILL.md` is "deliberately out of scope: each already carries zero development-repository references." That claim is incorrect; the Builder relied on it, so the fix should correct the scope claim as well as the line.
- **Remedy is reachable**: drop the clause (the sentence stands as "Direct the operator to any one of the CLI's three existing install channels; they all deliver the same released binary:"). Whether to additionally broaden Family B to reach bare/possessive `repositor(y|ies)` is a deliberate lexicon edit under the interface's Interactions flow (property comment + example + fixture) and is the developer's decision — a scan of `plugin/` shows this is the only occurrence, so the broadened pattern would be clean once the line is fixed.

---

## Verdict: Issues

2 findings, both surfaced by held-out validation scenarios; all 5 conformance dimensions pass. Both findings are single-sentence wording defects with reachable remedies — incremental, fixable in one implement round. Neither spans an entire dimension and neither reveals a missing implementation, so the gap is not fundamental.

The build-side capability is sound: the walker, lexicon, guard, report grammar, and constitutional amendment all conform to their accords, and the full gate is green. What the held-out scenarios caught is residue in the *sweep* half — one place where a number carried information the replacement dropped (F-1), and one place the sweep never visited because an upstream scope claim said it was already clean (F-2).

---

## Next Steps

2 findings to address. Suggest fixing via `/score:implement`, then re-validating with `/score:validate 076`.

Suggested order, both in one round:

1. **F-2 first** — it violates the accord's central rule and Principle XIII, and it is the one a reader would hit before any other. Correct `plugin/skills/glassfrog-setup/SKILL.md:50`, and correct T002's scope claim in tasks.md so the record matches what was true. Decide separately whether to broaden Family B; if yes, it needs its property comment, example, and a known-safe fixture per the interface's lexicon-edit flow, with every existing fixture kept green.
2. **F-1** — restore the dropped component at `plugin/skills/proposal-drafting/SKILL.md:31` and `:101`, matching the phrasing the sibling agent artifact already uses. Leave `:66` alone (pre-existing, out of scope).

Neither fix touches the guard's behavior, so the existing fixtures and the live tripwire should stay green; re-run `go test ./...`, `gofmt -l .`, and `golangci-lint run` before re-validating.
