# Checklist: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/clobbered-changes/stale-write-surfacing.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-06-14

---

## Summary

All 12 checks pass. Constitution: 12/12. (Done-criteria + cross-reference: skipped — no `done-*` accords; see Governance Notes.)

P0: 12/12 passed · P1: 0 · P2: 0.

Three principles pass by **correctly-asserted structural inapplicability** (VI Size-Aware, IX Writes-Require-Intent, XI Governance-via-Proposals). Two principles are the ones this feature most directly advances: **II Action Transparency** (it makes a refused write *more* legible — distinct category, cause, next step) and **X Respect API Limits** (it completes the optimistic-concurrency capture→send→surface chain X requires). One principle (VIII No Fabricated Data) is the one to watch during implementation — see the observation under its check.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **C-I — Spec Fidelity (P0)**: `412 Precondition Failed` is the spec-defined response for a failed `If-Match` precondition (v5 "Optimistic Concurrency (ETags)"; FEATURE-MODEL §200; spec System Overview). 054 classifies a status the API actually returns — it invents no endpoint, parameter, or API behavior, and adds no request surface. The new exit code `7` is a CLI-side *outcome* contract (interface-cli.md registry), not an API surface; classification is status-driven (plan ADR-1). Conformant.

- **C-II — Action Transparency (P0, NON-NEGOTIABLE)**: 054 *advances* II — it is the capability that makes a refused guarded write legible. A `412` gains a distinct, machine-parseable signal (exit code `7`), a cause naming what went wrong (the resource changed since it was read), and a next step (re-read for the current version, then retry), replacing the misleading generic "check that the token has access" step (spec Behavioral Accord; interface-cli Error Communication; plan ADR-2). The three-field Diagnostic is machine-parseable per II. Strongly conformant.

- **C-III — Fail Safe, Not Silent (P0)**: 054 makes a refused write *more* obvious, not less — a `412` exits non-zero (`7`), never `0` (spec non-behavior; interface-cli Error Communication; CONSTITUTION III). It swallows no error and reports no failure as success; its entire purpose is to surface the clobber distinctly so the operator does not silently overwrite a concurrent edit. Advances Fail Safe.

- **C-IV — Test-Driven Development (P0)**: T001 ships the classification change (enum value + code + `categoryForStatus`/`nextStepForStatus`/cause arms) together with unit tests, and the `stale-write-surfacing.feature` scenarios are authored now (`@wip`) as executable acceptance before the code that satisfies them (tasks T001; plan Cross-cutting Testing; spec Driving/Validation Scenarios). Conformant.

- **C-V — Composition over Monolith (P0)**: The change is additive `case` arms on the existing `categoryForStatus`/`nextStepForStatus` switches, one new `Outcome` value, and one new `ExitCode` constant + case — localized to `internal/cli`'s classification and registry. It wires no command and forces no edit to unrelated modules; adding this category does not require changing unrelated commands (plan ADR-1; System Architecture). Mirrors 015's `APIError`-split precedent exactly. Conformant.

- **C-VI — Size-Aware, no silent truncation (P0)**: Inapplicable by structure, correctly asserted. 054 concerns the classification of a single failure outcome, not result-set paging or truncation — no collection walk, no `per_page` handling, nothing dropped (plan System Architecture; spec scope).

- **C-VII — Working Software (P0)**: T001 pairs implementation with its tests as one reviewable unit ("the code must not merge without the tests that pin its uniqueness and the no-drift guarantee") — no code-only or test-only increment (tasks T001 Scope/Acceptance).

- **C-VIII — No Fabricated Data (P0)**: 054 surfaces the API's own `detail`/`title` as the cause when present; when the API supplied none, the cause is derived from the `412` status's well-defined precondition-failed semantics and is explicitly **never invented** (spec Behavioral Accord + non-behavior; plan ADR-2). The next step is the actionable recovery, not a guess. *Observation for implementation*: the synthesized cause must state only what the `412` status defines (a precondition failure / changed-since-read), not assert *who* or *what* changed the resource — the plan's "Cause over-reach" risk names exactly this boundary. Conformant as specified; the implementation must hold the no-fabrication line in the synthesized-cause wording.

- **C-IX — Writes Require Explicit Intent (P0)**: Inapplicable by structure, correctly asserted. 054 is read-only failure classification — it issues no request and mutates nothing. It adds no read/list/get path that writes; it only reclassifies a failure the request layer already surfaced (plan System Architecture; spec non-behaviors).

- **C-X — Respect API Limits (P0)**: 054 *advances* X — X mandates optimistic concurrency (`If-Match`/`ETag`) "rather than last-write-wins clobbering." 054 completes that capability's capture (052) → send (053) → surface (054) chain by making the clobber-refusal (`412`) legible so the operator re-reads before retrying (spec System Overview; FEATURE-MODEL §207). It does not touch the `429`/`Retry-After` backoff path (031/017 own that, unchanged). Conformant — and the realization of X's "rather than clobbering" clause.

- **C-XI — Governance via Proposals (P0)**: Inapplicable by structure, correctly asserted. 054 exposes no governance-structure mutation path — it is a failure-classification change with no command, flag, or write. No opt-in flag is needed because there is no governance mutation to gate (plan System Architecture; spec non-behaviors).

- **C-XII — Standalone Executable (P0)**: Inapplicable / conformant. 054 adds Go code compiled into the existing binary — no language runtime, no external software, no new dependency beyond network access to the API (plan Implementation Strategy: changes within `internal/cli` only). The distributed artifact still runs on host OS + network alone.

---

## Governance Notes

- **No `done-*` accords found**: `accords/governance/` does not exist, so done-criteria checks and cross-reference (link-presence) checks were not generated. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable per-artifact done-criteria checks in future runs. This is a project-wide infrastructure gap, not a 054 finding — consistent with the same note on 053's checklist.
- **No principles produced zero checks** for inapplicability reasons that would warrant suppression: VI, IX, and XI pass as *correctly-asserted structural inapplicability* (the feature genuinely has no paging surface, no mutation, and no governance-structure path), not as skipped checks.
