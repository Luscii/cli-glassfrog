# Validate: Proposal Circulation Path

**Feature**: 068-proposal-circulation-path
**Round**: 1 of 3
**Date**: 2026-07-19
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, `features/unequipped-agent-operators/proposal-circulation-path.feature`, PROJECT.md
**Implementation files**: 6 — `plugin/skills/proposal-circulation/SKILL.md`, `plugin/agents/proposal-circulator.md`, `plugin/agents/proposal-circulation-commands.txt`, `internal/build/proposalcirculation.go`, `internal/build/proposal_circulation_guard_test.go`, `internal/build/proposal_circulation_bdd_test.go`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. 6 of 6 validation scenarios satisfied.

The deliverable is two declarative plugin artifacts (a thin skill delegating to a fenced subagent) plus a single-source leaf list and a best-effort Go drift guard — there is no runtime Go path of its own. Conformance is therefore traced against artifact *content* and the guard's structural assertions, exercised by the build-side BDD suite (17/17 scenarios) run against 063's real gate script. Nothing under `plugin/` compiles into the CLI (`OrientationPluginHasNoGoCode` holds), so the artifacts add knowledge + guardrails, never capability.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 driving scenarios covered)

Every driving scenario in spec.md § Driving Scenarios maps to a `@wip`-cleared feature-file scenario with a passing trace in `proposal_circulation_bdd_test.go`.

| Driving scenario (spec.md) | Status | Trace |
|---|---|---|
| Advance a draft into circulation | ✓ Covered | SKILL.md workflow step 3 (advance) + agent `proposal propose`; BDD `A draft proposal advances into circulation` |
| Monitor a circulating proposal | ✓ Covered | SKILL.md workflow step 3 (monitor) drawing `response_summary`/`response_deadline`/`available_transitions`; BDD `A circulating proposal is monitored as one picture` |
| Withdraw a circulating proposal back to draft | ✓ Covered | SKILL.md workflow step 3 (withdraw) + step 4 handoff to 067; BDD `A circulating proposal is withdrawn back to draft` |
| A transition is rejected | ✓ Covered | agent "Circulating defensively" (403/404/422 by name, `action: none`, fabricate no state); BDD `A rejected transition fabricates no state` |
| A monitoring read fails | ✓ Covered | agent "A monitoring read fails mid-walk" (gathered-so-far, flagged incomplete); BDD `A failed monitoring walk yields a partial picture` |
| The transition must be confirmed before it crosses the boundary | ✓ Covered | gated-writes note + confirmation contract; declined = no transition; BDD `An unconfirmed transition leaves the record untouched` |
| The path reads to inform, not to gate | ✓ Covered | reads-inform-never-gate fence, server-authorizes + `422` plainly; BDD `A stale snapshot does not stop a transition` |
| A response belongs to the response side | ✓ Covered | handoff to response side (069); BDD `A consent response is handed to the response side` |

---

## Acceptance Criteria

**Status**: Pass (both tasks checked; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — skill + circulator agent + single-source leaf list, auto-discovered | ✓ Met | SKILL.md frontmatter (`name`/`description`) + required sections (when / workflow / delegation / gated-writes note); agent frontmatter (`name`/`description`/`tools: Bash, Read, Grep, Glob`/`model`) excludes Write/Edit; five required agent sections present; circulation-record shape (`proposal`/`situating`/`action`/`handoff`/`notes`); `proposal-circulation-commands.txt` holds exactly the four leaves; `plugin.json` unchanged (`TestProposalCirculationKeepsManifestAutoDiscovered`); siblings untouched (git diff shows no non-068 plugin files changed) |
| T002 — drift guard: composed leaves + two-write gated-membership | ✓ Met | `CheckProposalCirculationDrift` asserts each composed leaf resolves on the CLI's proposal command; `proposal propose` & `proposal withdraw` each anchored (`ProposalCirculationGatedWrites`) as members of 063's gated set and cross-checked against the composed list; the two reads asserted absent; agent names every leaf; partial coverage documented in-test; verified red-on-drift (removing `proposal withdraw` from `gated-commands.txt` fails the build naming the leaf) |

---

## Interface Contract Conformance

**Status**: Pass (all specified surfaces present with the specified shapes)

interface-spec.md fixes the two files' frontmatter, required sections, the confirmation contract, the circulation-record shape, and the single-source leaf list. Each is present and structurally asserted by the BDD suite.

| Surface (interface-spec.md) | Status | Evidence |
|---|---|---|
| `SKILL.md` frontmatter (`name` + `description`, non-firing on 064–067/069 needs) | ✓ Conformant | `name: proposal-circulation`; description enumerates the three circulation needs and the five sibling exclusions |
| `SKILL.md` required sections (when / workflow / delegation / gated-writes note) | ✓ Conformant | All four headed sections present; workflow single-sourced, agent references it |
| `proposal-circulator.md` frontmatter (`name`/`description`/`tools`/`model`) | ✓ Conformant | Write-capable-but-fenced grant `Bash, Read, Grep, Glob`; Write/Edit withheld |
| Required agent sections (Identity & scope / Workflow / Confirmation contract / Composed commands / Output contract) | ✓ Conformant | All five present; workflow by reference, not a divergent copy |
| Circulation-record output shape | ✓ Conformant | `proposal`/`situating`/`action`(`advanced`\|`monitored`\|`withdrawn`\|`declined`\|`none`)/`handoff`/`notes`, each element id-carrying |
| Single-source leaf list | ✓ Conformant | `proposal-circulation-commands.txt` consumed by both agent prose and guard |

---

## Non-Behavior Absence

**Status**: Pass (all 8 exclusions upheld)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not draft/create/assemble | ✓ Absent | `proposal create` appears only in negated prose ("never runs", "no other command"); not in the leaf registry |
| Must not record a consent response | ✓ Absent | `proposal respond` only negated; `no_objection`/`bring_to_meeting` appear as server-returned state or response-side handoff, never as a write step |
| Must not perform propose/withdraw ungated | ✓ Absent | Both transitions gated at 063's real gate script (`ask`); grounding read passes ungated; guard pins both memberships |
| Must not pre-read to gate client-side | ✓ Absent | reads-inform-never-gate fence in both artifacts; `@validation` scenario `never pre-gates ... client-side` passes |
| Must not judge authority | ✓ Absent | No verdict phrasing found; explicit deferral to Constraint Discovery Path (065) |
| Must not interpret/re-describe server-owned side effects | ✓ Absent | Cleared timestamps / deleted responses stated as "reflected in the returned record", not narrated as side-effect commentary |
| Must not add capability / reimplement logic | ✓ Absent | Composition only; plugin tree carries no Go code (`OrientationPluginHasNoGoCode`); no `plugin.json` capability keys |
| Must not coach / dump raw output | ✓ Absent | Coaching disclaimer present; output framed "drawn-together ... never a concatenation of raw, unsynthesized command output" |

---

## @wip Lifecycle Completion

**Status**: Pass

Zero `@wip` tags remain in `proposal-circulation-path.feature`. All 17 scenarios (16 owned by T001, the no-invented-surface drift check owned by T002) are live and pass under the suite's `~@wip` filter. No scenario referenced by a checked task retains its marker.

---

## Validation Scenario Results

**Status**: Satisfied (6 of 6 traced to implementation)

These were held out from the Builder's task references and independently traced; each executes green in the BDD suite.

| Validation scenario | Status | Trace |
|---|---|---|
| No invented surface | ✓ Satisfied | Guard checks all four leaves against the live CLI proposal surface; BDD `The path names no command the CLI lacks` |
| Both gated transitions routed through the guardrail | ✓ Satisfied | Both bodyless commands return `ask` at 063's real gate; per-leaf membership pinned; BDD `The path routes both writes through the guardrail` |
| Reads inform, never gate | ✓ Satisfied | Fence + "issue the intended transition / let the server authorize / never a client-side precondition"; BDD `The path never pre-gates a transition client-side` |
| Circulation only, no response recording | ✓ Satisfied | No gated composed leaf beyond the two transitions; response recording deferred to 069; BDD `The path records no consent response` |
| Circulating, not judging or coaching | ✓ Satisfied | No authority verdict, no coaching; deferrals to 065; BDD `The path circulates without judging authority or coaching` |
| Synthesized, not raw | ✓ Satisfied | "drawn-together circulation picture carrying the prp_ id", never a concatenation; BDD `The result is a synthesized circulation picture, not raw output` |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 6 held-out validation scenarios are satisfied. The implementation conforms to the specification: the path composes exactly the four already-shipped proposal leaves, routes both `propose` and `withdraw` through 063's confirmed write flow (verified against the real gate script) with the two reads left ungated, reads to inform but never to pre-gate, records no response and creates no draft, narrates none of the server-owned side effects, and returns a drawn-together circulation record carrying the `prp_` id. The drift guard pins the composed surface and the two-in-two-out gate posture, and was confirmed red-on-drift. No `plugin.json` change; siblings 062–067 and `plugin/hooks/` untouched. Full `internal/build` package green (283 tests); no regressions.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 068 is closed.

Note for a future cycle: 069 (the response side) will face the agent-reuse question ADR-2 deliberately left open — whether it mints its own agent or shares one; `proposal-circulator` was intentionally not pre-generalized into a shared transition executor.
