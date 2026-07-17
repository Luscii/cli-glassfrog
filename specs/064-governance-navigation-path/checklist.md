# Checklist: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/governance-navigation-path.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-07-17

---

## Summary

All 12 checks pass. Constitution: 12/12. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Two observations are recorded below the check results and carried into analyze as completeness questions — neither is a binary violation of an artifact, so neither is a failing check.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **I. Spec Fidelity** (P0) — The path invents no endpoint, parameter, or behavior: it composes only shipped read commands (`search`, `roles`, `tree`, `fillers`, `subrole-actors`, `domains`, `policies`), plan ADR-3 forbids a new `navigate` command, and T002's drift guard pins those leaves to the CLI registry. spec.md Non-Behaviors + interface-spec Surface + validation scenario "The path names no read the CLI lacks".
- **II. Action Transparency** (P0) — The synthesized picture carries each element's id (traceable, actionable); the underlying reads keep the CLI's machine-parseable output; error/empty/partial-failure scenarios surface cause + a refine next-step. (See Observation 1 on the picture's *form*.)
- **III. Fail Safe, Not Silent** (P0) — Read-only, so no partial-write state is possible; the partial-failure scenario surfaces what failed and returns the reads that succeeded; the empty-search scenario reports "nothing found" rather than an empty-looking success.
- **IV. Test-Driven Development** (P0) — 14 acceptance scenarios exist `@wip` in the feature file before implementation (10 non-`@validation` + 4 `@validation`); tasks T001 (13) and T002 (1) reference the scenarios they un-`@wip`.
- **V. Composition over Monolith** (P0) — Additive: a new `plugin/skills/governance-navigation/` + `plugin/agents/` sibling and a new `internal/build` test; plan ADR-2 forbids restructuring; the orientation skill and unrelated commands are untouched.
- **VI. Size-Aware by Design** (P0) — The over-broad-concern scenario **signals** the boundary ("note that the picture was narrowed so the practitioner can refine") rather than silently truncating; underlying reads inherit the CLI's pagination via the orientation dependency. (See Observation 2 on paging-before-narrowing.)
- **VII. Working Software** (P0) — Neither task is a code-only/test-only increment: T001 un-`@wip`s its 13 scenarios (ships the step definitions that verify the artifacts); T002 ships the drift-guard test for its 1 scenario.
- **VIII. No Fabricated Data** (P0) — The empty-search scenario "will fabricate no roles or governance"; the partial-failure scenario "will not invent the missing piece"; spec Non-Behaviors bar local governance logic.
- **IX. Writes Require Explicit Intent** (P0) — Read-only by construction: ADR-5's tool grant withholds Write/Edit, the prompt scopes to read leaves, and 063's write hook backstops; validation scenario "The path only reads, never writes".
- **X. Respect API Limits** (P0) — No updates, so `If-Match`/`ETag` is N/A; a `429` mid-traversal is a read failure handled by the partial-picture scenario, and rate-limit/retry behavior is the CLI's (017), reached via orientation.
- **XI. Governance via Proposals** (P0) — Trivially satisfied: the path performs no governance-structure mutation at all; it defers even the authority *question* to 065 and capture to 066.
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the skill/agent are host-consumed plugin artifacts and the drift guard is a build-time Go test.

---

## Observations (not failing checks)

1. **Synthesized-picture form vs. Action Transparency (II)** — *open, accepted*: the output contract is field-shaped (roles/fillers/domains/policies with ids) but interface-spec states it is "presented as a readable picture, not a serialization format." Traceability is preserved by the ids, so II passes; whether the picture should *also* be offered in a machine-parseable form is a design choice, accepted for now (the consumer is an LLM agent that reads prose well, and the ids keep every element actionable).
2. **Paging-before-narrowing vs. Size-Aware (VI)** — *resolved (post-guard fix)*: the spec Traversal accord and interface Surface/Interactions now state the navigator pages through the full result set before narrowing (narrowing over the complete set, never a silent single-page cap), and the over-broad scenario asserts the page-through step. VI's silent-truncation risk is now pinned in the artifacts, not left to the orientation dependency alone.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
