# Checklist: Drafter Config Migration

**Feature**: 071-drafter-config-migration
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/drafter-config-contract.feature
**Checks**: 7 (6 pass, 1 fail) + 3 P2 considerations
**Generated**: 2026-08-03

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** — the same standing as sibling CI specs 024/028/029/030. Done-criteria checks are skipped, not failed.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 7 | 6 | 1 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 3 | — | — |
| **Total** | **7 checks** | **6** | **1** |

---

## Constitution Checks: 6/7 passed

- **P0 | III (Fail Safe, Not Silent)** — *calibrated*: this feature's "failure surface" is the CI guard, not a governance write. Calibrated assertions: (a) the guard does not pass on a condition it exists to catch — ADR-4 rejects the superseded config shape by name rather than letting it degrade into seven confusing missing-label messages, and ADR-5 forbids an underivable pinned ref from defaulting to a passing value; (b) no swallowed errors — every condition in interface-spec.md's Error Communication table fails, and the table states explicitly that there is no partial-success or defaulting mode; (c) no partially-applied state — the config, guard, and docs ship as one PR (tasks.md preamble), so the repository never sits between shapes. The guard's deliberate one-directional coverage is disclosed rather than hidden (plan Risks; see P2-1). **PASS.**

- **P0 | IV (Test-Driven Development / BDD)** — *calibrated*, two assertions:
  - (a) *User-facing behavior carries acceptance scenarios written before the code.* 13 scenarios in `drafter-config-contract.feature`, each `# Source:`-traced, all preceding the config and guard they describe; tasks.md T001–T004 reference them as verifying conditions. **Passes.**
  - (b) *No task requires executing a scenario the spec forecloses verifying.* **FAILS.** See finding F1 below.

  **FAIL** — the check is binary and (b) does not hold.

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

### F1 | P0 | IV — tasks.md T004 requires executing scenarios that cannot be executed

**Source**: CONSTITUTION.md IV (*"User-facing behavior MUST have an executable acceptance scenario"*), calibrated assertion (b).
**Artifacts**: `tasks.md` T004 acceptance criteria; `features/no-automated-pipeline/drafter-config-contract.feature`; `spec.md` Non-Behaviors.

T004 instructs the Builder to execute every scenario under two named Rule blocks and clear their `@wip` tags. At least three of the scenarios so designated assert things no test in this feature can observe, because `spec.md`'s Non-Behaviors forecloses runtime verification of drafter output and no "before" state exists at test time:

| Scenario | Why it is not executable |
|---|---|
| "A drafting run reports no schema deprecations" (Rule 1 — explicitly in scope per T004) | Asserts release-drafter's runtime output. Same class as the three scenarios T004 correctly excludes. |
| "The four label-contract assertions survive in number and strictness" (Rule 2, `@validation`) | Compares the guard *before* and *after* the change. No runtime has access to the prior guard. |
| "The change claims no fix for the untagged-release failure" (Rule 2, `@validation`) | Asserts about the pull-request description, which does not exist when tests run. The spec.md half could be grepped; the PR half cannot. |

This is not a wording slip in one criterion — it recurs across both Rule blocks T004 names, so the criterion is systematically over-broad. Left as written it pushes the Builder toward the one thing the same task's final criterion forbids: rewording a behavioral or review-time assertion into a config-shape assertion to make it green.

**Recommended resolution** (the Shaper's call, not the Guardian's): replace T004's "every scenario under [Rule]" phrasing with an explicit per-scenario execution table, and extend the deliberately-unexecuted set from three scenarios to six with the reason recorded per scenario. Then close the consequence in F2.

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

- **P2-2 | VII — T002 is a test-only unit.** Constitution VII forbids test-only increments outside the RED→GREEN pair. T002 (new drift cases) carries no implementation. It is benign — additive coverage over implementation T001 just landed, inside a single PR — but if the Builder follows tasks.md's `task-1 … task-5` branch guidance literally, it becomes a test-only commit. Recommend T001 and T002 land as one commit, mirroring 030's P2-2.

- **P2-3 | V — `DrafterWhen` deliberately declines the package's `StringOrSlice` idiom.** ADR-7 rejects a tolerant unmarshaller so no untested arm ships, with the consequence (a list-form config fails at parse rather than as a violation) recorded in interface-spec.md's Error Communication rather than smoothed over. Sound, and the reasoning is written down. Flagged only so a future reader encountering the parse-level failure finds it was anticipated.

---

## Governance Infrastructure Notes

Separate from the feature findings above.

- No `accords/governance/` directory exists. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria checks — currently every Score checklist in this repo (024, 028, 029, 030, and now 071) runs constitution-only and reports the same gap.
- Five of twelve constitution principles produce no applicable checks for this feature (I, II, IX, X, XI). This is expected for a pipeline/build feature with no CLI surface and matches 030's profile, where six were inapplicable. The difference is VIII, calibrated here rather than skipped — reasoning recorded inline above.
