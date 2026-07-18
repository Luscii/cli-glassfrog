# Checklist: Proposal Circulation Path

**Feature**: 068-proposal-circulation-path
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/proposal-circulation-path.feature, tasks.md
**Checks**: 12 (12 pass, 0 fail)
**Generated**: 2026-07-18

---

## Summary

All 12 checks pass. Constitution: 12/12. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Three observations are recorded below the check results. Observation 1 (record form vs Action Transparency) and Observation 2 (the 063 confirmation vs the constitution's no-interactive-prompts conflict row) are carried forward from 067 with the same disposition. Observation 3 is new to this path: the *reads-inform-never-gate* discipline is a prompt-level guarantee, not a tool-enforced one — recorded for traceability because 068 is the first path that reads `available_transitions` while also invoking the transitions it advertises.

---

## Constitution Checks: 12/12 passed

### Passed (12/12)

- **I. Spec Fidelity** (P0) — The path invents no endpoint, parameter, or behavior: it composes only shipped commands (`proposal propose` 057, `proposal withdraw` 059, `proposal get`/`proposal list` 056), plan ADR-4 forbids a new `proposal circulate` command, monitoring claims only the filters `proposal list` actually offers (no tension filter — same acknowledged limit as 067), and T002's drift guard pins the composed leaves to the CLI registry. The reads-inform-never-gate rule keeps the server the sole transition authority — the path adds no client-side transition logic the spec doesn't define. spec.md Non-Behaviors + interface-spec Surface + validation scenario "The path names no command the CLI lacks".
- **II. Action Transparency** (P0) — The circulation record carries each element's id (`proposal.id`, situating `prp_` ids, the `handoff` id); the underlying commands keep the CLI's machine-parseable output; the transition-rejected and monitoring-failure scenarios surface cause + a concrete next step; and the 063 confirmation displays the exact bodyless transition command before it runs (plan ADR-3). (See Observation 1 on the record's *form*.)
- **III. Fail Safe, Not Silent** (P0) — Each transition is a single atomic API call, so no multi-step partial-write state is possible; a rejected transition surfaces the failure by name and transitions nothing; a `422` is a real refusal, **never** absorbed as success (the 057/059 invariant, relayed); a declined confirmation is an explicit outcome (`action: declined`), never a silent skip or a failure-as-success; a mid-walk monitoring failure is flagged incomplete with its cause. spec Error scenarios + interface Error Communication.
- **IV. Test-Driven Development** (P0) — 17 acceptance scenarios exist `@wip` in the feature file before implementation (11 non-`@validation` + 6 `@validation`); tasks T001 (16) and T002 (1, the no-invented-surface drift check) reference the scenarios they un-`@wip`.
- **V. Composition over Monolith** (P0) — Additive: a new `plugin/skills/proposal-circulation/` + `plugin/agents/proposal-circulator.md` sibling pair and a new `internal/build` test; plan ADR-2 explicitly declines to modify 067's `proposal-drafter` (fence preservation); T001's acceptance criteria pin every sibling artifact untouched.
- **VI. Size-Aware by Design** (P0) — The monitoring step, when it surfaces the circle's in-flight proposals, pages through the **full** result set (spec Monitoring accord + interface Interactions + T001 acceptance criterion + the scenario "A failed monitoring walk yields a partial picture"); a mid-walk failure yields a partial picture *flagged incomplete* (056's flag, relayed) — never a silent cap. Underlying reads inherit the CLI's pagination via the orientation dependency.
- **VII. Working Software** (P0) — Neither task is a code-only/test-only increment: T001 un-`@wip`s its 16 scenarios (ships the step definitions that verify the artifacts); T002 ships the drift-guard test with its 1 scenario.
- **VIII. No Fabricated Data** (P0) — The transition-rejected scenario "will fabricate no state the record does not contain"; the failed-walk scenario "will not invent the missing data"; the returned proposal is surfaced **exactly as the server last returned it** (interface output shape) with acceptance never computed client-side; the `proposal` element is absent (not defaulted) when the grounding read failed. spec Advancing/Withdrawing accords.
- **IX. Writes Require Explicit Intent** (P0) — The path's two writes are the direct result of the explicit `proposal propose` / `proposal withdraw` commands, each run deliberately and *additionally* gated behind 063's human confirmation, each confirmed independently (never batched or pre-authorized); the reads (`proposal get`/`list`) never mutate; a declined confirmation means no write occurs. validation scenarios "The path routes both writes through the guardrail" + "Two transitions in one session confirm twice". (See Observation 2.)
- **X. Respect API Limits** (P0) — Neither transition carries `If-Match` because a transition has no prior `ETag` to guard (057/059 non-behavior — concurrency applies to field edits, which this path performs none of); a `429` mid-circulation is the CLI's rate-limit concern (017), reached via orientation. interface-spec Error Communication marks optimistic-concurrency N/A here (owned by the CLI).
- **XI. Governance via Proposals** (P0) — The path is the proposal path working as designed: it moves a proposal through its `/proposals` lifecycle (`propose`/`withdraw` transitions) and exposes no direct governance mutation and no bypass; both writes are gated (063) on top of the proposal discipline the constitution requires. It creates no governance change directly (that is 067) and records no consent (that is 069).
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the skill/agent are host-consumed plugin artifacts and the drift guard is a build-time Go test.

---

## Observations (not failing checks)

1. **Circulation-record form vs. Action Transparency (II)** — *open, accepted*: the output contract is field-shaped (proposal/situating/action/handoff/notes with ids) but interface-spec states it is "a contract shape, not a serialization format." Traceability is preserved by the ids, so II passes; whether the record should *also* be offered machine-parseable is a design choice, accepted for now (the consumer is an LLM agent; the ids keep every element actionable). Same disposition as 064/065/066/067's Observation 1.
2. **Interactive confirmation vs. the "no interactive prompts" conflict row (IX)** — *resolved upstream, recorded for traceability*: the constitution's conflict-resolution table says intent is expressed by the explicit write command, "not by an interactive prompt" — yet 068's transitions are confirmed interactively. That tension was adjudicated by 063 itself: the confirmation lives at the **operator layer** (a host `PreToolUse` hook), not in the CLI, whose commands remain non-interactive and automation-friendly. 068 introduces no new prompt of its own and the CLI surface is unchanged, so IX passes as adjudicated. Same disposition as 067's Observation 2.
3. **Reads-inform-never-gate is prompt-level, not tool-enforced (I / IX)** — *accepted, novel to this path*: 068 is the first path that reads `available_transitions` and then invokes the transitions it advertises, so the discipline "issue the transition and let the server authorize; never pre-gate client-side" is a **prompt-level** guarantee (plan ADR-4) — it cannot be tool-enforced, because reading the proposal is legitimate and required. It is verified only by the held-out validation scenario "The path never pre-gates a transition client-side", not by the drift guard. This mirrors how 064's surface-not-judge and 065's no-local-verdict are prompt-level guarantees verified by validation scenarios rather than structurally. Accepted; flagged so analyze/validate keep the scenario as the enforcement point.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
- **guardian-agent.md not bundled** in this checklist skill deployment — checklist ran on SKILL.md process alone (reduced character consistency, not a blocked skill), as the SKILL.md fallback permits.
