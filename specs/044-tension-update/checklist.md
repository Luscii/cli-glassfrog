# Checklist: Tension Update

**Feature**: 044-tension-update
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md, features/tension-capture/tension-update.feature
**Checks**: 22 (22 pass, 0 fail)
**Generated**: 2026-06-13

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 18 | 18 | 0 |
| P1 (should fix) | 1 | 1 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| Cross-reference (P1) | 3 | 3 | 0 |
| **Total** | **22** | **22** | **0** |

> **Correction (guard pre-mode review)**: the initial pass recorded three failures (Constitution IV, VII, and the tasks→scenarios cross-reference) on the premise that no `.feature` artifact existed. That premise was wrong — it looked only in the spec directory. This project keeps feature files under `features/<problem>/` (the convention 043 set with `features/tension-capture/tension-reads.feature`), and the scenarios artifact is present at `features/tension-capture/tension-update.feature` (13 scenarios), exactly where every tasks.md reference points. All three findings are withdrawn; the items pass.

---

## Constitution Checks: 18/18 passed

### Passed (18/18)

- **IV (Test-Driven Development)** — User-facing behavior has an executable acceptance scenario ahead of the GREEN code: `features/tension-capture/tension-update.feature` exists (13 scenarios under four `Rule:` blocks from the spec's User Scenarios), all behavioral scenarios `@wip` pending T003, the three held-out validation scenarios `@validation @wip`. T003 ("Make the driving scenarios pass as executable acceptance") consumes this file and removes `@wip` as each scenario goes GREEN — the RED-first acceptance source precedes implementation. (tasks.md § Phase 1, tension-update.feature)
- **VII (Working Software)** — The decomposition pairs implementation with its tests: T002 is RED-first unit tests for every branch, T003 the godog acceptance suite over the present `tension-update.feature`; no test-task is gated on an absent input (the scenarios artifact exists where tasks.md references it). (tasks.md § Phase 1)

Constitution checks passed:

- **I (Spec Fidelity)** — Every command maps to a defined operation: `tension update <ten-id>` → `PATCH /tensions/{id}` (`updateTension`), verified present in `spec/glassfrog-api-v5.yaml:2684`. The request body `{tension:{body,label,status,meeting_type}}` matches the vendored `TensionInput` schema (`spec.yaml:6737`); the `status` enum (`unprocessed`/`processed`/`archived`) and `meeting_type` enum (`tactical`/`governance`) match the spec exactly (spec.md § Input, interface-spec.md § Surface). No invented endpoints, parameters, or behaviors. (spec.md, plan.md, interface-cli.md, interface-spec.md)
- **II (Action Transparency, NON-NEGOTIABLE)** — Output traces to endpoint + resource id: the rendered tension carries its `ten_` id and the action is `PATCH /tensions/{id}`; structured `-o json/yaml` emits machine-parseable `{data}`; every error names cause + next step and exits with a typed code (spec.md § Output/Failure, interface-cli.md § Error Communication table). (spec.md, interface-cli.md)
- **III (Fail Safe, Not Silent)** — Writes are validated before sending: blank-`--body`-when-supplied, `--status`/`--meeting-type` enum, and at-least-one-field preconditions all run pre-assembly with a transport tripwire (plan ADR-3, interface-cli.md § Interactions). Single `PATCH`, no multi-step partial-apply path; failures surface non-zero. (spec.md, plan.md, interface-cli.md)
- **V (Composition over Monolith)** — Adds one leaf to the existing `tension` group via the registration guard, one request-input type, two pure checks; reuses `tensionSeam`, validators, render key, `Document[Tension]`. No edits to unrelated commands; `create`/`list`/`get` left unchanged (plan § System Architecture, interface-spec.md § Consistency Notes). (plan.md, interface-cli.md, interface-spec.md)
- **VI (Size-Aware by Design)** — Single-resource read of one tension; no list, no pagination. The "never silently truncate" surface is not engaged (no result set); interface-spec.md explicitly states `Page[T]`/`Pagination` are not used. No violation. (spec.md, interface-spec.md)
- **VIII (No Fabricated Data)** — The command forwards the validated `--status` and renders whatever the server returns after its recompute, claiming no authority over the final value; no defaulted/guessed fields (spec.md Non-Behaviors, plan.md ADR-1, interface-cli.md § Consistency Notes). (spec.md, plan.md, interface-cli.md)
- **IX (Writes Require Explicit Intent)** — The mutation occurs only as the direct result of the explicit `tension update` write command; reads (`get`/`list`) are untouched and the editable flags live only on `update`, so a read path cannot reach the `PATCH` (plan ADR-2, interface-cli.md § Consistency Notes). (plan.md, interface-cli.md)
- **X (Respect API Limits)** — Honors the rate-limit/concurrency model: a `PATCH` `429` is surfaced (never silently re-sent) because `isSafeMethod` gates auto-retry to GET/HEAD (verified `retry.go:65`). Note: this is a SHOULD-adjacent point on `If-Match` — see P1 pass below. (plan § Cross-cutting, interface-cli.md § Interactions)
- **XI (Governance via Proposals)** — A tension is an operational item, not governance structure (roles/accountabilities/domains/policies); PROJECT.md § Domain confirms tensions are directly editable. No governance-structure mutation, so no proposal-gating obligation applies. No violation. (spec.md, PROJECT.md)
- **XII (Standalone Executable)** — Adds only Go code to existing packages; no new runtime/interpreter/external dependency introduced (plan § Implementation Strategy reuses landed transport/render). No violation. (plan.md)

(Constitution X carries both a P0 MUST on rate-limit back-off and the `If-Match` optimistic-concurrency point; the back-off MUST is the P0 pass above. The `If-Match` decision is recorded as the P1 check below.)

---

## Cross-Reference Checks: 3/3 passed

### Passed (3/3)

- **tasks.md → scenarios artifact**: the Inputs line, T001/T002/T003 "Scenario references" all cite `features/tension-capture/tension-update.feature` (T002 lists eleven named scenarios). The target file exists at that exact path with all 13 scenarios; every named scenario title resolves to a `Scenario:` line in the file (spot-checked against the eleven T002 names). No dangling links.
- **tasks.md → plan.md**: T001/T002/T003 "Plan reference" fields cite Phase 1, ADR-1, ADR-2, ADR-3, and Cross-cutting — all present in plan.md § Architecture Decisions / Cross-cutting Concerns. Links verified present.
- **tasks.md → interface artifacts**: T001/T002/T003 "Interface references" cite interface-spec.md (`internal/glassfrog` partial-update input; `internal/cli` additions) and interface-cli.md (Surface, Interactions, Error Communication) — all sections present in both files. Links verified present.

---

## Done-Criteria Checks

No `done-*.md` accords exist (no `accords/` directory in the repository). Done-criteria checks were not generated. See Governance Notes.

---

## Governance Notes

- **No `accords/governance/` directory**: No done-* accords are present anywhere in the repository, so done-criteria checks were skipped entirely. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable per-artifact done-criteria quality checks. This matches the repo's established state (interface-cli.md § Consistency Notes itself notes "No `accords/` directory exists"). Checklist ran in constitution-only mode.
- **Constitution VI (Size-Aware) and XI (Governance via Proposals)**: produced no applicable failure-mode checks for this feature — VI because update is a single-resource write with no result set to truncate, XI because a tension is an operational item, not governance structure. Both are recorded as passing/non-applicable, not skipped.
- **Constitution principle classification**: All twelve principles carry concrete detection mechanisms (MUST/MUST NOT + observable pattern), so no calibration step was required; checks were generated directly from each principle's detection clause.
