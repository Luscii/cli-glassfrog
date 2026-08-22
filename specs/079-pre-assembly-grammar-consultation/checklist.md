# Checklist: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Checked against**: CONSTITUTION.md (v1.1 — 13 principles, I–XIII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/unguided-change-construction/pre-assembly-grammar-consultation.feature (12 scenarios), features/proposal-circle-not-choosable/pre-assembly-routing-application.feature (7 scenarios)
**Checks**: 24 (24 pass, 0 fail)
**Generated**: 2026-08-22 (round 2 — re-derived after the K5 remedy: two architecture-informed scenarios added to the routing feature file, tasks.md inventory refreshed)

> Source note: no `accords/governance/` directory is deployed in this repo, so this checklist runs **constitution checks only** — the same standing as the sibling specs' checklists. Done-criteria checks are skipped, not failed.

> Calibration note: this feature is a declarative operating-surface change — plugin-prose edits plus repo-side BDD, no CLI code. Eleven principles were calibrated to that shape; **XII (Standalone Executable)** produced zero applicable checks (no build, distribution, or dependency change), and **X (Respect API Limits)** produced one narrow check (no new retry loop; the added reads ride the CLI's existing 429 handling). Every principle is phrased as MUST/MUST NOT, so mechanical severity inheritance puts all 24 checks at P0.

> Verification note: mechanical claims were verified against the working tree and the built binary, not against the artifacts' own assertions — all eight composed leaves resolve on the shipped CLI (each `--help` exits 0, including `proposal grammar`); every one of the spec's 15 scenarios (10 driving + 5 validation) has a feature-file scenario whose `# Source:` title matches verbatim (title-diff empty, re-run this round); the 4 architecture-informed additions are marked `Proposed:` with their sources; `PROPOSAL_READS=" list get grammar "` and `gated-commands.txt` were re-read to confirm the gate posture the accord claims.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 24 | 24 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | 0 | 0 |

**Verdict: all 24 checks pass.** Constitution: 24/24. Done-criteria: not run (no accords). Cross-references: folded into constitution checks IV.2–IV.3 (no accord source exists to generate a separate category).

---

## Constitution Checks

### I. Spec Fidelity — 3/3 pass

- **I.1 PASS** — Every command the artifacts compose exists on the shipped CLI. All eight leaves the interface pins (`tension get`, `proposal list`, `proposal get`, `proposal create`, `me roles`, `tension list`, `roles`, `proposal grammar`) resolve (`--help` exit 0, verified against the built binary).
- **I.2 PASS** — No artifact claims an API capability the spec does not define: no tension-id filter on `proposal list` is implied (grep clean), no circle parameter on the create is implied (the routing premise is the parameter's *absence*), and the grammar command is consumed as landed with no new flags claimed.
- **I.3 PASS** — The feature adds no command, flag, or endpoint: spec non-behavior 6-adjacent boundary ("adds no command, flag, or capability of its own"), plan System Architecture ("no Go CLI code"), tasks T001 scope (three plugin files "and no other file") agree.

### II. Action Transparency (NON-NEGOTIABLE) — 3/3 pass

*Calibrated: for this feature, transparency means the consultation is legible from the returned record — the gate must be falsifiable from the operator's side.*

- **II.1 PASS** — The result carries what was consulted and what it surfaced: spec Reporting accord group; interface pins the `consultation` element **on every action path** with three named parts (grammar / routing / match).
- **II.2 PASS** — Incomplete consultation is named, never presented whole: spec Reporting bullet 2; interface Error Communication rows for the failed grammar read and the incomplete routing walk; runnable scenarios assert both.
- **II.3 PASS** — Every awaiting-direction return names its cause and substance: the three new `action` values each carry the decision's content (mismatch + eligible anchors; named anchors or capture gap; fact handle + shape + symptom) — interface action table and defensive contract.

### III. Fail Safe, Not Silent — 3/3 pass

*Calibrated: this feature deliberately performs no local validation (the server stays the sole judge — settled repo-wide by the 072→078 line), so Fail Safe here means no failure state reads as success and nothing degrades silently.*

- **III.1 PASS** — A failed grammar read never reads as a consulted assembly: spec accord ("rather than presenting the assembly as consulted"), scenario "A failed grammar read is recorded and drafting continues".
- **III.2 PASS** — An incomplete routing walk is flagged, inventing nothing and abandoning nothing: spec accord bullet, scenario "An incomplete routing walk continues flagged, inventing nothing".
- **III.3 PASS** — "No recorded shape matched" is stated, not implied by silence, and implies no validity endorsement: spec accord bullets 3–4 of the surfacing group; scenario "A change set matching nothing recorded implies nothing about validity".

### IV. Test-Driven Development — 3/3 pass

- **IV.1 PASS** — Executable acceptance scenarios exist ahead of implementation: 17 scenarios across two feature files, all `@wip`; 12 runnable, 5 held `@validation` for independent verification.
- **IV.2 PASS** — Every spec scenario is covered: title-diff between the spec's 15 scenario titles and the feature files' `# Source:` titles is empty; the 4 architecture-informed additions (2 from shaping, 2 closing the round-1 analyze K5 gap) are marked `Proposed:` with their plan/interface/accord sources.
- **IV.3 PASS** — Tasks bind tests to the work: T002 carries the full scenario inventory with per-file counts and step-assertion minimums for every conduct, including the two K5-closing scenarios; tasks' Scenario inventory section matches the files on disk (19 scenarios: 14 runnable + 5 held; grammar 12, routing 7).

### V. Composition over Monolith — 2/2 pass

- **V.1 PASS** — The change is confined to the module that owns it: T001 scope names the three drafting-path files "and no other file"; the plan verifies zero guard-code changes and zero edits to sibling paths.
- **V.2 PASS** — No hidden cross-module coupling is introduced: the routing-record and grammar-command contracts stay owned by their features (spec non-behavior 6; interface Consistency Notes "what stays owned elsewhere").

### VI. Size-Aware by Design — 2/2 pass

- **VI.1 PASS** — The situating full-walk survives the workflow rewrite: interface workflow table step 3 pins "full walk" unchanged; T001 acceptance requires retaining every 067-pinned phrase (which include the page-before-judging conduct).
- **VI.2 PASS** — The own-roles read's non-pagination is hedged, not hidden: spec accord ("an absence there is an absence in what was read, never a settled absence"); interface routing part carries the hedge; the boundary is signalled rather than silently truncated.

### VII. Working Software — 2/2 pass

- **VII.1 PASS** — Every task requires green build/lint/tests in its acceptance criteria (`gofmt -l .` clean and `go test` green appear in T001, T002, T003).
- **VII.2 PASS** — Implementation and tests land together: T002 (the suite) is Phase 1 beside T001, and Phase 2 rides in the same PR — no code-only or test-only increment is scheduled.

### VIII. No Fabricated Data — 2/2 pass

- **VIII.1 PASS** — The routing answer presents only what the reads returned: eligible anchors come from the record's own tensions; an incomplete walk "invents nothing"; the empty set is reported as empty with the gap named.
- **VIII.2 PASS** — The unchanged create-failure conduct is preserved: no `prp_` id is fabricated (interface Error Communication row; inherited 067 conduct explicitly restated).

### IX. Writes Require Explicit Intent — 3/3 pass

- **IX.1 PASS** — Consultation is composed of reads only, and reads never mutate: the four added-to-workflow leaves (`me roles`, `tension list`, `roles`, `proposal grammar`) are all reads; the two-phase returns write nothing.
- **IX.2 PASS** — The gate posture holds by construction: `proposal grammar` is in the write gate's recognized-read set and absent from `gated-commands.txt` (re-read from the working tree); `proposal create` remains the only gated composed leaf; interface pins both directions and an architecture-informed scenario asserts them.
- **IX.3 PASS** — Nothing turns consultation into a confirmation: spec accord ("consultation itself asks the operator to confirm nothing"); the one confirmed write remains the create.

### X. Respect API Limits — 1/1 pass

- **X.1 PASS** — No new retry loop or limit-ignoring behavior is introduced: the added reads ride the CLI's existing rate-limit handling; the repeat-from-top re-delegation is bounded by practitioner direction (one return per decision, not a loop); no artifact prescribes automatic retries.

### XI. Governance via Proposals — 1/1 pass

- **XI.1 PASS** — No direct governance mutation is added or made reachable: the feature's only write remains the proposal create through the confirmed flow; the never-withhold fence removes nothing from the proposal path.

### XII. Standalone Executable — 0 checks

*No applicable checks: the feature changes no build, distribution, or dependency surface (see Governance Notes).*

### XIII. Self-Contained Operating Surface — 3/3 pass

- **XIII.1 PASS** — The rule is carried into every artifact that directs plugin edits: spec Integration Boundaries (in-surface resolution), interface Consistency Notes (the ban restated for new prose), T001 acceptance criterion (self-containment check green).
- **XIII.2 PASS** — The planned annotation rewrites stay in-surface: the interface's registry contract references only plugin paths and the CLI; the existing registry text it amends was already swept by the self-containment feature, and no new repo-side reference is prescribed.
- **XIII.3 PASS** — Detection is inherited, not re-invented: the walking self-containment guard covers edited files with no list update (plan Integration Design), so a violation in T001's edits fails the build.

---

## Governance Infrastructure Notes

- **No `accords/governance/` directory**: done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) could not run. Consider creating these accords to enable done-criteria checking; consistent with every sibling spec to date, so not a finding against this feature.
- **Principle XII produced zero checks**: the feature ships no executable change — plugin prose and repo-side tests only. Nothing to assert.
- **Calibration record**: II (transparency → consultation legibility in the returned record), III (fail-safe → no failure reads as success; deliberately *not* read as "validate locally", which the repo's settled server-sole-judge posture supersedes and spec non-behavior 1 forbids), X (limits → no new retry loop) were calibrated; all other principles applied concretely.

---

## Improvement Summary

Previous run: 24 checks, 24 pass, 0 fail. Current run: 24 checks, 24 pass, 0 fail. No regressions; IV.2/IV.3 re-verified against the grown scenario set (17 → 19) and the refreshed tasks inventory.
