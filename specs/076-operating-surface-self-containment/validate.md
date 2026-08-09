# Validate: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Round**: 2 of 3
**Date**: 2026-08-09
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, PROJECT.md, `features/unequipped-agent-operators/operating-surface-self-containment.feature`, validate.md (round 1)
**Implementation files**: 26 files under `plugin/` (22 swept), `internal/build/surfaceselfcontainment.go`, `internal/build/surface_self_containment_guard_test.go`, `internal/build/surface_self_containment_bdd_test.go`, 5 re-derived BDD suites, `CONSTITUTION.md`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings

All dimensions were re-run from source rather than carried forward from round 1 — a fix round can introduce gaps as readily as it closes them.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

| Scenario | Status | Implementation |
|---|---|---|
| A conforming surface passes verification | ✓ Covered | `ScanOperatingSurface`; BDD `givenConformingSurface`, `TestScanOperatingSurfaceCleanFixture` |
| A handoff reads by name where the operator stands | ✓ Covered | BDD surface-content steps read the real swept surface |
| A future file is covered without registration | ✓ Covered | `filepath.WalkDir` derives the set; `TestScanOperatingSurfaceDerivedWalk` |
| A spec-number reference turns the run red | ✓ Covered | Family A `\b0\d{2}\b` with report grammar and remedy |
| A repository mention fails even without a path | ✓ Covered | Family B patterns; all three forms named in the scenario's Given are matched |
| Non-reference tokens do not false-positive | ✓ Covered | `knownSafeTokens` corpus, `TestSurfaceLexiconKnownSafeTokens` |
| An empty surface is a failure, not a pass | ✓ Covered | Zero-file and missing-directory both error "missing or empty" |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks checked, criteria re-verified independently)

Re-verified this round from source, not carried forward: deny-lexicon grep returns zero across all 26 surface files; all seven registries' non-comment leaves remain byte-identical to `origin/main`; the gate script's diff is still comment-only (zero executable lines); no number-pinned assertion against surface text remains anywhere in `internal/build`; every `plugin/…` path mentioned in the surface resolves.

Gate at validation time: `go test ./...` 2332 passed, `gofmt -l .` clean, `golangci-lint run` 0 issues, `pre-commit run --from-ref origin/main --to-ref HEAD` — 8 hooks passed, 0 failed.

T002's scope paragraph was corrected during the fix round to record that its "already carries zero development-repository references" premise was false for one file, and why a clean lexicon grep is not evidence of conformance. The task record now matches what was true.

---

## Interface Contract Conformance

**Status**: Pass (4 of 4 surfaces conformant)

| Surface | Status | Notes |
|---|---|---|
| §1 Deny lexicon (Family A + B) | ✓ Conformant | All 12 patterns present with the accord's RE2 syntax, case-sensitivity, and anchoring; property comment + example on each entry; known-safe corpus pinned as fixtures |
| §2 Guard surface (3 files) | ✓ Conformant | Production source shared by guard and suite; live tripwire in the guard test; detection steps run only against `t.TempDir()` fixtures and never invoke the walker on the real `plugin/` |
| §3 Registry-header instructional contract | ✓ Conformant | All 7 artifacts keep what-the-file-is, in-plugin consumer list by `plugin/`-relative path, editing rule, and fail-closed consequence in surface terms |
| §4 Constitution Principle XIII | ✓ Conformant | Heading after XII; bold statement, *Rationale*, *Detection*; version marker 1.0→1.1 with adjacent dated justification |

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions held)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not change what the surface does | ✓ Held | Gate script diff comment-only; all seven registries' leaves byte-identical; the fix round touched prose only |
| Must not rewrite any already-merged spec | ✓ Held | `git diff --name-only origin/main -- specs/` returns only 076's own files and STATUS.md |
| Must not restrict repo→surface references | ✓ Held | Every `plugin/…` constant in `internal/build` resolves; the only two non-resolving strings are deliberate fixture paths inside the guard test |
| Must not ban operating-world references | ✓ Held | Known-safe corpus green; the setup skill's three install channels (raw URL, brew tap, npm package) survived the F-2 fix untouched |
| Must not enumerate the checked files | ✓ Held | `filepath.WalkDir` over `plugin/`; no file list in the guard |
| Must not satisfy a failing assertion by deletion | ✓ Held | All 13 re-derived assertions retain a component-name pin |

---

## @wip Lifecycle Completion

**Status**: Pass

8 scenarios executed by checked tasks have their `@wip` removed. 5 `@wip` tags remain, all on `@validation`-tagged scenarios — matching tasks.md's disposition table (8 executed, 5 held, 0 inexecutable). Driven by tag, not count.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 scenarios traced)

| Scenario | Status | Trace |
|---|---|---|
| The sweep preserved every handoff | ✓ Satisfied | All three compound `(068/069)` sites now name both receiving components; every other replacement names the component its number pointed at |
| Re-derived assertions kept the property | ✓ Satisfied | All 13 sites pin a component name; feature-file narrative and the `071` guard-output assertion correctly left untouched (repo-side, not surface content) |
| The surface reads lightweight and workflow-oriented | ✓ Satisfied | Wide residual sweep (beyond the lexicon) returns no repository mechanics, design history, or enforcement machinery — see Observation 2 for one recorded ambiguity |
| The constitutional principle is in house form | ✓ Satisfied | Statement + *Rationale* + *Detection* mirror siblings I–XII; version bump and adjacent justification satisfy the Governance clause |
| Pointer direction is intact | ✓ Satisfied | Surface→repo: zero references in any form. Repo→surface: every guard path constant resolves |

---

## Changes Since Previous Run

**Round**: 2 (previous: Round 1 — Issues, 2 findings)

### Resolved (2 findings)

- **F-1** (Round 1): two collapsed compound references misattributed the respond handoff — **resolved**. `plugin/skills/proposal-drafting/SKILL.md:31` now splits the verb list ("*advance or withdraw* … the **Proposal Circulation Path** — or to *record a response* … the **response side**, the proposal-impact-review path"), and `:102` does the same in the gated-write note. The term "response side" now appears twice in a file where it appeared zero times, so the handoff is locally bound rather than inferred.
- **F-2** (Round 1): the setup skill referenced the development repository — **resolved**. `plugin/skills/glassfrog-setup/SKILL.md:50`'s provenance clause is gone; the sentence now ends "they all deliver the same released binary:" and the three install channels are unchanged. A `repositor(y|ies)` / README / "install guide" sweep across `plugin/` returns nothing. The false scope premise in T002 that hid the file was corrected in the same round.

### Remaining (0 findings)

None.

### New (0 findings)

None. The fix round touched four files (two surface prose files, tasks.md, LEARNINGS.md) and introduced no new gap: the full gate is green, the registry leaves and gate logic are still byte-identical to `origin/main`, and a sibling sweep for the F-1 construct across both drafting artifacts — including frontmatter, which the finding did not name — surfaced no site the sweep had broken.

---

## Observations

Recorded because they are ambiguous or deliberately deferred, not because they are defects. None affects the verdict.

1. **Two pre-existing misattributions remain in the drafting skill, correctly out of scope.** The frontmatter `description` (line 3) and the workflow's hand-off step (line 67) both attribute *responding* to the Proposal Circulation Path. Neither was introduced by the sweep: the frontmatter is byte-identical to `origin/main` and never carried a spec number, and line 67 carried `(068)` with "responding" already in the sentence. 076 is a repository-reference sweep, and non-behavior 1 forbids changing what the surface does, so fixing them here would be scope creep. They are a candidate for a future prose-accuracy pass.

2. **The reference records' `Provenance` lines are repository-free in wording but still point outward.** `change-set-grammar-facts.md:52,66` now read "supersedes the provisional field note of 2026-08-05 (F5/F6)" where they named a portfolio document before. This satisfies the strict ban as specified — no repository, path, spec id, or artifact name is mentioned — but the referent remains unresolvable for the operator. The field itself is a required structural element of the record's own contract, which non-behavior 1 forbids 076 from changing, so removing it was never an option. Naming the ambiguity rather than silently ruling either way: a stricter future reading could ask for a purely self-contained phrasing, and that would be a change to the record's contract, not to 076.

3. **The lexicon's reach gap that produced F-2 is still open, by decision.** Family B matches `(?:source|development|parent)\s+repositor(?:y|ies)` and does not reach a bare or possessive "the repository's". The Builder fixed the violation but deliberately did not broaden the pattern, because the pattern table is a shaped contract in `interface-spec.md` and that accord explicitly frames the lexicon as an intentional under-approximation ("the strict ban's *reach*, not its *definition* … novel phrasings are a review obligation"). So this is not a conformance gap — nothing in the spec or interface requires the pattern to catch that phrasing. It is, however, a live demonstration that the accepted residual can be real rather than hypothetical. Broadening the entry to `(?i)\brepositor(?:y|ies)\b` would land clean today (the word now appears nowhere in `plugin/`, and the existing example still matches so no fixture goes red); the cost is foreclosing any future legitimate use of the word inside the surface. A contract decision for the Crafter, not a defect.

4. **The report grammar's family slot is elaborated** (carried from round 1). The accord writes `(family <A|B>: <property>)`; the implementation emits `(family A — resolvable reference: <property>)` — a strict superset that still begins `family A`, matching the accord's Interactions clause verbatim. Recorded so it is not mistaken for drift later.

---

## Verdict: Ready

All 5 conformance dimensions pass and all 5 held-out validation scenarios are satisfied. Both round-1 findings are resolved at the source rather than worked around, and the fix round introduced no new gap.

The capability delivers what the spec promised: the surface is swept clean of development-repository references in every form the deny lexicon reaches and in the one form it did not; the guard walks a derived file set so future files are covered without registration; the report grammar names file, line, text, family, property, and a remedy that is reachable for every condition; the number-pinned assertions were re-derived rather than deleted, preserving the property each protected; and Principle XIII records the rule with an observable detection mechanism and a version marker that makes the next amendment mechanical.

Worth noting for the record: both round-1 findings came from the held-out scenarios, not from the conformance dimensions, and both were wording judgments that lost information — precisely the failure mode a mechanical guard cannot see. That is the separation between creation and evaluation doing its job.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.

The branch `spec/076-operating-surface-self-containment/base` is 18 commits ahead of `origin/main` with a clean tree and a green gate. Before opening the PR, `git fetch origin main` and confirm no conflicts — main has not moved during this session, but it has moved during the workspace's lifetime.

Two items for the PR description, both carried from the Observations above rather than left implicit: the announced divergence the plan recorded (re-derived feature-scenario step text now diverges from the merged specs' prose it was copied from — git history is the changelog, merged specs stay untouched), and the open lexicon-reach decision in Observation 3, which a reviewer may want to settle in this PR or defer.
