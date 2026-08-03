# Checklist: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/drafter-config-contract.feature
**Checks**: 7 (7 pass, 0 fail) + 3 P2 considerations
**Generated**: 2026-08-03 (round 2 — re-derived after F1 was resolved)

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** — the same standing as sibling CI specs 024/028/029/030. Done-criteria checks are skipped, not failed.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 7 | 7 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 3 | — | — |
| **Total** | **7 checks** | **7** | **0** |

**Improvement summary** — round 1: 1 P0 fail (F1), 1 P1 (F2). Round 2: 0 P0, 1 P1. **Resolved**: F1 — `tasks.md` T004 now carries a per-scenario execution table naming five scenarios to execute and eight to hold, with the two hold reasons kept distinct, and `plan.md` ADR-8 grounds the runner it wires. **Remaining**: F2, unchanged and now sharper — the scenario that would have observed the deprecation surface is explicitly in the held set.

---

## Constitution Checks: 7/7 passed

- **P0 | III (Fail Safe, Not Silent)** — *calibrated*: this feature's "failure surface" is the CI guard, not a governance write. Calibrated assertions: (a) the guard does not pass on a condition it exists to catch — ADR-4 rejects the superseded config shape by name rather than letting it degrade into seven confusing missing-label messages, and ADR-5 forbids an underivable pinned ref from defaulting to a passing value; (b) no swallowed errors — every condition in interface-spec.md's Error Communication table fails, and the table states explicitly that there is no partial-success or defaulting mode; (c) no partially-applied state — the config, guard, and docs ship as one PR (tasks.md preamble), so the repository never sits between shapes. The guard's deliberate one-directional coverage is disclosed rather than hidden (plan Risks; see P2-1). **PASS.**

- **P0 | IV (Test-Driven Development / BDD)** — *calibrated*, two assertions:
  - (a) *User-facing behavior carries acceptance scenarios written before the code.* 13 scenarios in `drafter-config-contract.feature`, each `# Source:`-traced, all preceding the config and guard they describe; tasks.md T001–T004 reference them as verifying conditions. **Passes.**
  - (b) *No task requires executing a scenario the spec forecloses verifying.* T004's per-scenario table marks all four drafter-output scenarios **hold**, and its criteria forbid rewording any scenario to make it executable. The four `@validation` scenarios are held separately for `/score:validate` — Score's own convention, and the same disposition `release_bdd_test.go` applies. **Passes** (round 2; failed in round 1 as F1).

  **PASS** — both assertions hold. Note the tags are now load-bearing: with `Tags: "~@wip"`, clearing a tag is what puts a scenario under test, so the table in T004 is the executable record rather than commentary.

- **P0 | V (Composition over Monolith)** — ADR-3 places the coupling verdict in a new `internal/build/drafterschema.go` rather than widening `CheckLabelContract`'s signature, so the two invariants stay independently reviewable and neither existing guard (021 `.goreleaser`, 022 release-workflow, 030 label contract, 036 brews-targets-the-tap) is edited. The new file reuses `RepoRoot()`, `loadYAML`, `ReleaseDrafterConfig`, and `workflow.go`'s `Workflow`/`Step` types by reference, adding no duplicate parse. Adding this guard forces no change to an unrelated one. **PASS.**

- **P0 | VI (Size-Aware by Design / never silently truncate)** — *calibrated*: the analogue of truncation here is a contract element silently disappearing during the position change. The realignment moves the seven category labels, the exclusion label, the three semver buckets, and the patch fallback without dropping any; T002's drift cases fail loudly on each individual loss, and the validation scenario "No label is invented or dropped by the realignment" pins the set equality against 028's managed labels. Notably, ADR-2 keeps the patch fallback **declared** precisely because its silent removal would change no observable output — the one case where only the guard can see the loss. **PASS.**

- **P0 | VII (Working Software)** — the feature adds no CLI Go code; its only Go is the two `internal/build` guard files, which land with the config they guard. Every task leaves a valid state, and the tasks.md preamble names the full pre-push gate (`gofmt -l .`, `go vet`, `go test -count=1`, `golangci-lint run`) including the reason `go test` alone is insufficient here — a struct reshape re-aligns sibling gofmt columns. **PASS.** (See P2-2 on T002's commit-level shape.)

- **P0 | VIII (No Fabricated Data)** — *calibrated*: the guard must present only values it actually derived. ADR-5 requires the pinned major to be read from the workflow's `uses:` ref and forbids substituting a value when the ref yields none — T003's acceptance criteria state "must not default to zero and must not pass," and the drift case for an underivable ref pins it. The schema floor is the one hardcoded value, and it is a named constant carrying its property in its comment rather than a synthesized value presented as derived. The validation scenario "Neither side of the coupling verdict is a hard-coded literal" holds the line. **PASS.** *(Divergence from 030's classification, deliberately: 030 marked VIII not-applicable because it renders no API data. Here the principle has a real analogue — the guard reports a derived version — so it is calibrated rather than skipped.)*

- **P0 | XII (Standalone Executable) — no new runtime dependency** — the feature changes nothing in the distributed artifact. The release-drafter action is a CI-host tool, the same standing the project already gives GoReleaser, golangci-lint, srvaroa/labeler, and `sigs.k8s.io/yaml` (030's XII note). Raising the pinned action major touches the CI host only. **PASS.**

### Principles with no applicable checks for this feature

- **I (Spec Fidelity)**, **II (Action Transparency)**, **IX (Writes Require Explicit Intent)** — these govern the CLI's command, request, and output surface against the Glassfrog API v5 spec. This feature adds no CLI command, issues no Glassfrog API call, and renders no API data. No applicable checks. *(Thematic alignment worth noting, not a check: II's "every error must explain what went wrong and the next step" is mirrored by interface-spec.md's Error Communication table, which requires every violation to name the file, the section, and the offending value — and requires the superseded-shape message to name the schema rather than the symptom, so the next step is unambiguous.)*
- **X (Respect API Limits)**, **XI (Governance via Proposals)** — concern the Glassfrog API's rate limits/concurrency and governance-structure mutation. This feature touches neither. No applicable checks.

---

## Findings

### F1 | P0 | IV — RESOLVED in round 2

**Was**: `tasks.md` T004 instructed the Builder to execute every scenario under two named Rule blocks. At least three of those assert things no test in this feature can observe — release-drafter's runtime output, a comparison against the guard's prior state, and the content of a pull-request description absent at test time. The phrasing recurred across both Rule blocks, so it was systematically over-broad, and it pushed toward the one thing the same task's final criterion forbade: rewording an assertion into a config-shape check to make it green.

**Resolution applied**: T004 now carries a per-scenario execution table — five execute, eight hold — and `plan.md` ADR-8 grounds the runner the task wires. The hold set came out larger than this finding recommended (eight rather than six) because the four `@validation` scenarios are held for `/score:validate` under Score's own convention, which `release_bdd_test.go` also follows; two of those overlap with the inexecutable set, and the table records both reasons rather than collapsing them. T004's criteria require the suite's doc comment to keep the two reasons separate, so a later reader cannot "finish" the inexecutable four by rewording them.

**Verified**: assertion (b) of the IV check now holds. Retained here rather than deleted — the Guardian's record is additive, and the reasoning is what stops the same over-broad phrasing returning.

### F2 | P1 | III — the zero-deprecation-warnings accord has no detection mechanism

**Source**: CONSTITUTION.md preamble (*"if a violation can't be observed, the principle is aspirational, not constitutional"*) applied to the spec's own accord; III (Fail Safe, Not Silent).
**Artifacts**: `spec.md` Behavioral Accord > Deprecation surface; `plan.md` Risks; `tasks.md`.

`spec.md` states: *"When the drafter runs against the realigned config, it emits no configuration-schema deprecation warnings."* This is the feature's headline outcome — the reason it exists. Once F1 is resolved and "A drafting run reports no schema deprecations" is correctly marked unexecutable, nothing verifies it: CI cannot (the workflow is `push: main`-triggered), and the spec explicitly says *"no acceptance gate depends on"* the post-merge run.

The plan does mitigate indirectly — T001's acceptance criteria enumerate the forbidden fields whose presence would emit warnings, so the config is constructed to avoid them. But that is prevention, not detection: a warning from a form nobody anticipated would surface to no one.

This is P1 rather than P0 because the outcome degrades noisily (a warning in a run log) rather than silently, and the spec already disclaims the gate. It is a finding because the accord currently reads as a guarantee while resting on inspection alone.

**Recommended resolution**: name the observer and the moment in `spec.md` — "the first drafting run after merge is read once; a residual warning is a follow-up, not a rollback" would make the accord honest — or accept it explicitly as inspection-verified. Either closes the gap; leaving it implicit does not.

---

## P2 Considerations (advisory — not blocking)

- **P2-1 | III — the coupling guard is one-directional by design.** It catches config-newer-than-action and not action-newer-than-config. Disclosed in ADR-5 and plan Risks with the reasoning (the caught direction is silent and catastrophic; the uncaught one is noisy and degrades through compatibility mode). Recorded here so the asymmetry is visible in the Guardian record too, not only in the plan's own prose. No action recommended. F2's gap sits in the uncaught direction, so the two used to compound — the noisy signal being the only signal, with nobody assigned to read it. That is partly relieved now: the repository's recorded procedure for a hand-made major bump (bump, then watch a throwaway pre-release end to end) does assign an observer to the run this change produces. It is a procedure rather than a check, and it does not close F2, but it means the warning is no longer guaranteed to go unread.

- **P2-2 | VII — T002 and T004 are test-only units.** Constitution VII forbids test-only increments outside the RED→GREEN pair. T002 (new drift cases) and, since round 2, T004 (the godog suite) both carry no implementation. Both are benign — additive coverage over implementation that just landed, inside a single PR — but if the Builder follows tasks.md's `task-1 … task-5` branch guidance literally, each becomes a test-only commit. Recommend T001+T002 land as one commit and T003+T004 as another, mirroring 030's P2-2. Widened in round 2: adding the suite added a second instance of the same shape.

- **P2-3 | V — `DrafterWhen` deliberately declines the package's `StringOrSlice` idiom.** ADR-7 rejects a tolerant unmarshaller so no untested arm ships, with the consequence (a list-form config fails at parse rather than as a violation) recorded in interface-spec.md's Error Communication rather than smoothed over. Sound, and the reasoning is written down. Flagged only so a future reader encountering the parse-level failure finds it was anticipated.

---

## Governance Infrastructure Notes

Separate from the feature findings above.

- No `accords/governance/` directory exists. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks — currently every Score checklist in this repo (024, 028, 029, 030, and now 071) runs constitution-only and reports the same gap.
- Five of twelve constitution principles produce no applicable checks for this feature (I, II, IX, X, XI). This is expected for a pipeline/build feature with no CLI surface and matches 030's profile, where six were inapplicable. The difference is VIII, calibrated here rather than skipped — reasoning recorded inline above.
