# Checklist: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/constraint-discovery-path.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-07-18

---

## Summary

All 12 checks pass. Constitution: 12/12. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Two observations are recorded below the check results and carried into analyze as completeness/consistency questions — neither is a binary violation of an artifact, so neither is a failing check.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **I. Spec Fidelity** (P0) — The path invents no endpoint, parameter, or behavior: it composes only shipped read commands (`search`, `roles`, `tree`, `domains`, `policies`, `policy`, `me roles`), plan ADR-2 forbids a new `constraints`/`authority` command, and T002's drift guard pins those leaves to the CLI registry. spec.md Non-Behaviors + interface-spec Surface + validation scenario "The path names no read the CLI lacks".
- **II. Action Transparency** (P0) — The synthesized picture carries each element's id (traceable, actionable) and a `characterization` drawn from the record; the underlying reads keep the CLI's machine-parseable output; the ambiguous/empty/partial-failure outcomes surface cause + a refine next-step. (See Observation 1 on the picture's *form*.)
- **III. Fail Safe, Not Silent** (P0) — Read-only, so no partial-write state is possible; the failed-read scenario surfaces what failed and returns the reads that succeeded; the record-does-not-clearly-answer scenario reports the ambiguity rather than a confident-looking success (write-validation clauses are N/A — the path drives no write).
- **IV. Test-Driven Development** (P0) — 15 acceptance scenarios exist `@wip` in the feature file before implementation (11 non-`@validation` + 4 `@validation`); tasks T001 (14) and T002 (1) reference the scenarios they un-`@wip`. (RED→GREEN itself is exercised at implement time.)
- **V. Composition over Monolith** (P0) — Additive: a new `plugin/skills/constraint-discovery/` + `plugin/agents/constraint-navigator.md` sibling and a new `internal/build` test; plan ADR-1 forbids restructuring; the orientation (062) and governance-navigation (064) artifacts and unrelated commands are untouched.
- **VI. Size-Aware by Design** (P0) — The over-broad-action scenario **pages through the full result set before narrowing** and signals the boundary ("note that the picture was narrowed so the practitioner can refine") rather than silently truncating; underlying reads inherit the CLI's pagination via the orientation dependency. Pinned in the spec Discovery accord, interface Surface/Interactions, and T001 AC.
- **VII. Working Software** (P0) — Neither task is a code-only/test-only increment: T001 un-`@wip`s its 14 scenarios (ships the step definitions that verify the artifacts); T002 ships the drift-guard test for its 1 scenario. (Fully evaluable at PR/implement time.)
- **VIII. No Fabricated Data** (P0) — The unconstrained-action scenario surfaces the *absence* without asserting permission; the ambiguous-record scenario "will not fabricate an authority ruling"; the validation scenarios pin "no permission verdict from local logic" and "no fabricated ruling under uncertainty"; spec Non-Behaviors bar local permission logic. This is the path's central discipline.
- **IX. Writes Require Explicit Intent** (P0) — Read-only by construction: ADR-4's tool grant withholds Write/Edit, the prompt scopes to read leaves, and the agent is non-interactive; validation scenario "The path only reads, never writes".
- **X. Respect API Limits** (P0) — No updates, so `If-Match`/`ETag` is N/A; a `429` mid-traversal is a read failure handled by the partial-picture scenario, and rate-limit/retry behavior is the CLI's (017/031), reached via orientation — not reimplemented locally.
- **XI. Governance via Proposals** (P0) — Trivially satisfied and, in fact, *reinforced*: the path performs no governance-structure mutation, and its `characterization` explicitly names "a governance change → needs a proposal," pointing at the proposal norm rather than any bypass.
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the skill/agent are host-consumed plugin artifacts and the drift guard is a build-time Go test. (The plugin host is an external contract of the operating surface, not a dependency of the CLI executable.)

---

## Observations (not failing checks)

1. **Synthesized-picture form vs. Action Transparency (II)** — *open, accepted*: the output contract is field-shaped (`domains`/`policies`/owning-role with ids + `characterization`) but interface-spec states it is "presented as a readable picture, not a serialization format." Traceability is preserved by the ids, so II passes; whether the picture should *also* be offered in a machine-parseable form is a design choice, accepted for now (the consumer is an LLM agent, and the ids keep every element actionable). Identical to 064 Observation 1.
2. **Composed-reads set vs. FEATURE-MODEL dependency list** — *resolved (post-guard fix)*: the interface composes `roles`/`tree` (Role Reads) and `me roles` in addition to the FEATURE-MODEL's listed deps (Cross-Model Search + Role Domains + Role Policies). This is not a Spec-Fidelity violation — all are shipped, v5-backed reads. Analyze picked this up as H3 (plan↔interface enumeration drift); it is now closed by reconciling plan ADR-2 to name `me roles`, so plan, interface, tasks, and the drift-guard list agree. Justified in interface Consistency Notes and DECISIONS.md.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
