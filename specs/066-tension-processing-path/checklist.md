# Checklist: Tension Processing Path

**Feature**: 066-tension-processing-path
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/tension-processing-path.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-07-18

---

## Summary

All 12 checks pass. Constitution: 12/12. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Two observations are recorded below the check results. Observation 1 (record form vs Action Transparency) is accepted, mirroring 064. Observation 2 (paging-before-duplicate-judgment vs Size-Aware) was resolved by a post-guard fix that pinned page-through in the spec Situating accord and the situating scenario.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **I. Spec Fidelity** (P0) — The path invents no endpoint, parameter, or behavior: it composes only shipped operational tension commands (`tension create`/`list`/`get`/`update`/`discard`/`subroles`), plan ADR-4 forbids a new `tension process` command, and T002's drift guard pins those leaves to the CLI registry. spec.md Non-Behaviors + interface-spec Surface + validation scenario "The path names no command the CLI lacks".
- **II. Action Transparency** (P0) — The tension record carries each element's id (`tension.id`/`sensing_role_id`, the `handoff` `ten_` id); the underlying commands keep the CLI's machine-parseable output; the capture-rejected and situating-failure scenarios surface cause + a concrete next step. (See Observation 1 on the record's *form*.)
- **III. Fail Safe, Not Silent** (P0) — Each tension write is a single atomic API call (create/update/discard), so no multi-step partial-write state is possible; the capture-rejected scenario surfaces the failure by name and records nothing; the situating-failure scenario surfaces what failed and returns the reads that succeeded rather than a failure-as-success.
- **IV. Test-Driven Development** (P0) — 15 acceptance scenarios exist `@wip` in the feature file before implementation (10 non-`@validation` + 5 `@validation`); tasks T001 (14) and T002 (1) reference the scenarios they un-`@wip`.
- **V. Composition over Monolith** (P0) — Additive: a new `plugin/skills/tension-processing/` + `plugin/agents/tension-processor.md` sibling and a new `internal/build` test; plan ADR-2 forbids restructuring; the orientation skill, the 064 navigator agent, and the 063 hooks are untouched.
- **VI. Size-Aware by Design** (P0) — The situating step pages through the full result set **before** judging duplicates (spec Situating accord + interface Interactions + T001 acceptance criterion + the situating scenario "will page through the full result set before judging duplicates") — the duplicate check is over the complete set, never a silent single-page cap. Underlying reads inherit the CLI's pagination via the orientation dependency. (See Observation 2 — resolved.)
- **VII. Working Software** (P0) — Neither task is a code-only/test-only increment: T001 un-`@wip`s its 14 scenarios (ships the step definitions that verify the artifacts); T002 ships the drift-guard test for its 1 scenario.
- **VIII. No Fabricated Data** (P0) — The capture-rejected scenario "will fabricate no ten_ id"; the situating-failure scenario "will not invent the missing tensions"; spec Non-Behaviors bar local governance logic.
- **IX. Writes Require Explicit Intent** (P0) — The path's writes are the direct result of explicit tension write commands (`tension create`/`update`/`discard`), each run deliberately by the processor as the practitioner directs; the situating reads (`list`/`get`/`subroles`) never mutate. Intent is expressed by the explicit write command itself (constitution conflict-resolution row), so no read-shaped command mutates as a side effect. validation scenario "The path performs only operational tension writes".
- **X. Respect API Limits** (P0) — The path composes the CLI's `tension update`, which carries the CLI's optimistic-concurrency (`If-Match`/`ETag`, 052/054) — the path adds no update logic and does not bypass it; a `429` mid-processing is the CLI's rate-limit concern (017), reached via orientation. interface-spec Error Communication marks `If-Match`/`412` N/A here (owned by the CLI).
- **XI. Governance via Proposals** (P0) — The path performs **no** governance-structure mutation (roles/accountabilities/domains/policies): its writes are operational tension edits, and it explicitly never crosses into a proposal write, deferring proposal drafting to 067 and the authority question to 065. It exposes no default path that bypasses `/proposals`.
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the skill/agent are host-consumed plugin artifacts and the drift guard is a build-time Go test.

---

## Observations (not failing checks)

1. **Tension-record form vs. Action Transparency (II)** — *open, accepted*: the output contract is field-shaped (tension/situating/action/handoff/notes with ids) but interface-spec states it is "presented as a readable record, not a serialization format." Traceability is preserved by the ids, so II passes; whether the record should *also* be offered in a machine-parseable form is a design choice, accepted for now (the consumer is an LLM agent that reads prose well, and the ids keep every element actionable). Same disposition as 064's Observation 1.
2. **Paging-before-duplicate-judgment vs. Size-Aware (VI)** — *resolved (post-guard fix)*: the interface and tasks already stated page-through-before-narrowing, but the spec Situating accord and the situating driving scenario did not pin it — a latent silent-truncation gap (a duplicate on an unfetched page would be missed, so a duplicate could be recorded). Fixed during this guard run: the spec Situating accord now states the path pages through the full set before judging duplicates, and the situating scenario asserts "it will page through the full result set before judging duplicates." VI's silent-truncation risk is now pinned in the artifacts, not left to the interface/tasks alone.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
