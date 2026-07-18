# Checklist: Proposal Drafting Path

**Feature**: 067-proposal-drafting-path
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/proposal-drafting-path.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-07-18

---

## Summary

All 12 checks pass. Constitution: 12/12. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Two observations are recorded below the check results. Observation 1 (record form vs Action Transparency) is accepted, mirroring 064/065/066. Observation 2 (the 063 confirmation vs the constitution's no-interactive-prompts conflict row) is resolved by 063's own adjudication — recorded here for traceability because 067 is the first path that *exercises* that gate.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **I. Spec Fidelity** (P0) — The path invents no endpoint, parameter, or behavior: it composes only shipped commands (`proposal create` 055, `proposal list`/`proposal get` 056, `tension get` 043), plan ADR-4 forbids a new `proposal draft` command, situating claims only the filters `proposal list` actually offers (circle + status, no tension filter — spec Assumption 5), and T002's drift guard pins the composed leaves to the CLI registry. spec.md Non-Behaviors + interface-spec Surface + validation scenario "The path names no command the CLI lacks".
- **II. Action Transparency** (P0) — The draft record carries each element's id (`draft.id`/`tension_id`, `anchor.id`, situating `prp_` ids, the `handoff` id); the underlying commands keep the CLI's machine-parseable output; the create-rejected and situating-failure scenarios surface cause + a concrete next step; and the 063 confirmation displays the exact create command — payload inline — before it runs (plan ADR-3). (See Observation 1 on the record's *form*.)
- **III. Fail Safe, Not Silent** (P0) — The create is a single atomic API call, so no multi-step partial-write state is possible; the change set is validated client-side before any request (055's floor: valid JSON array, non-empty, `type` per element); a rejected create surfaces the failure by name and creates nothing; a declined confirmation is an explicit outcome (`action: declined`), never a silent skip or a failure-as-success; a mid-walk situating failure is flagged incomplete with its cause.
- **IV. Test-Driven Development** (P0) — 16 acceptance scenarios exist `@wip` in the feature file before implementation (10 non-`@validation` + 6 `@validation`); tasks T001 (15) and T002 (1) reference the scenarios they un-`@wip`.
- **V. Composition over Monolith** (P0) — Additive: a new `plugin/skills/proposal-drafting/` + `plugin/agents/proposal-drafter.md` sibling pair and a new `internal/build` test; plan ADR-2 explicitly declines to modify 066's `tension-processor` (fence preservation); T001's acceptance criteria pin every sibling artifact untouched.
- **VI. Size-Aware by Design** (P0) — The situating step pages through the **full** in-flight result set before judging duplicates (spec Situating accord + interface Interactions + T001 acceptance criterion + the situating scenario "will page through the full result set before judging duplicates"); a mid-walk failure yields a partial picture *flagged incomplete* (056's flag, relayed) — never a silent cap. Underlying reads inherit the CLI's pagination via the orientation dependency.
- **VII. Working Software** (P0) — Neither task is a code-only/test-only increment: T001 un-`@wip`s its 15 scenarios (ships the step definitions that verify the artifacts); T002 ships the drift-guard test with its 1 scenario.
- **VIII. No Fabricated Data** (P0) — The create-rejected scenario "will fabricate no prp_ id the record does not contain"; the failed-walk scenario "will not invent the missing proposals"; spec Non-Behaviors bar local governance/validation logic; the record's `draft` element is absent (not defaulted) when nothing was created.
- **IX. Writes Require Explicit Intent** (P0) — The path's one write is the direct result of the explicit `proposal create` command, run deliberately and *additionally* gated behind 063's human confirmation; the reads (`proposal list`/`get`, `tension get`) never mutate; a declined confirmation means no write occurs. validation scenarios "The path routes its one write through the guardrail" + "The path stops at the created draft". (See Observation 2.)
- **X. Respect API Limits** (P0) — The create carries no `If-Match` because a create has no prior `ETag` (055's non-behavior — concurrency applies to edits, which this path performs none of); a `429` mid-drafting is the CLI's rate-limit concern (017), reached via orientation. interface-spec Error Communication marks optimistic-concurrency N/A here (owned by the CLI).
- **XI. Governance via Proposals** (P0) — The path is the proposal path working as designed: it alters governance **only** by creating a draft through `/proposals` (`proposal create`), exposes no direct governance mutation and no bypass, and hands circulation to 068. Its single write is gated (063) on top of the proposal discipline the constitution requires.
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the skill/agent are host-consumed plugin artifacts and the drift guard is a build-time Go test.

---

## Observations (not failing checks)

1. **Draft-record form vs. Action Transparency (II)** — *open, accepted*: the output contract is field-shaped (draft/anchor/situating/action/handoff/notes with ids) but interface-spec states it is "a contract shape, not a serialization format." Traceability is preserved by the ids, so II passes; whether the record should *also* be offered machine-parseable is a design choice, accepted for now (the consumer is an LLM agent; the ids keep every element actionable). Same disposition as 064/065/066's Observation 1.
2. **Interactive confirmation vs. the "no interactive prompts" conflict row (IX)** — *resolved upstream, recorded for traceability*: the constitution's conflict-resolution table says intent is expressed by the explicit write command, "not by an interactive prompt" — yet 067's create is confirmed interactively. That tension was adjudicated by 063 itself: the confirmation lives at the **operator layer** (a host `PreToolUse` hook), not in the CLI, whose commands remain non-interactive and automation-friendly. 067 is the first path to *exercise* that gate; it introduces no new prompt of its own and the CLI surface is unchanged, so IX passes as adjudicated.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
