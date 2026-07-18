# Risk: Proposal Circulation Path

**Feature**: 068-proposal-circulation-path
**Round**: 1
**Date**: 2026-07-18
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Degradation flags**: none — full artifact set present. Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. No Regulatory Context in PROJECT.md — regulatory bridge omitted.

---

## Risk Register

| ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A gated transition (`propose` or `withdraw`) executes without human confirmation | spec § Crossing the guardrail; plan ADR-3; interface § Gated-write nuance | High | Low | RC-1, RC-2, RC-3 | **Yellow** (accepted with justification) |
| H-2 | The path pre-gates a transition client-side on a stale `available_transitions` snapshot (forks the server's authority) | spec § Monitoring + Non-Behaviors; plan ADR-4; interface § No-pre-gate nuance | Medium | Medium | RC-4, RC-5 | Green |
| H-3 | A rejected or failed transition is misreported (fabricated state, or a `422` absorbed as success) | spec § Error scenarios; interface § Error Communication | High | Low | RC-6 | **Yellow** (accepted with justification) |
| H-4 | An incomplete monitoring walk is treated as the complete picture | spec § Error scenarios; interface § Error Communication | Medium | Low | RC-7 | Green |
| H-5 | Boundary bleed: the path creates (067) or records a response (069) beyond its two transitions | spec § Non-Behaviors; plan Risk 4 | Medium | Low | RC-8, RC-9 | Green |
| H-6 | Artifact/CLI drift: a named command changes, or a transition leaves / a read enters 063's gated set | spec § Staying within the operator layer; plan ADR-5 + Risk 2 | Medium | Medium | RC-10 | Green |
| H-7 | The host does not honour subagents, tool grants, or hooks | plan Risk 6; interface § Error Communication | High | Low | RC-11, RC-3 | **Yellow** (accepted with justification) |
| H-8 | Skill and agent workflow prose diverge | plan Risk 1 | Low | Medium | RC-12 | Green |
| H-9 | Confirmation fatigue: repeated advance/withdraw cycles lead to a rubber-stamped destructive transition | spec § Crossing the guardrail; plan Risk 5 | Medium | Low | RC-13, RC-3 | Green |

**Unacceptable (Red) residual risks: none.**

---

## Hazard Detail

### H-1 — A gated transition executes without human confirmation

**Description**: `proposal propose` and `proposal withdraw` are the two writes the Write-Safety Guardrail exists to gate. If either executed unconfirmed — because the hook did not fire inside the subagent, because the leaf left 063's gated registry, or because the agent routed the write around the gate — a governance transition would run without the explicit human confirmation the design promises. This is the first path to cross the gate twice, so there are two leaves to keep gated.
**Severity: High** — an unconfirmed governance transition is exactly the failure class 063 was built to prevent. Unlike 067 (whose worst case is a *visible, withdrawable draft*), this path's worst case is more consequential: an unconfirmed `propose` starts the consent window (records the proposer's implicit `no_objection` and sets the deadline), and an unconfirmed `withdraw` **irreversibly deletes the prior `proposal_responses`** server-side. Neither yet produces an *accepted* change (acceptance still requires the full consent window), but the destructive `withdraw` branch means the bounding property is weaker than 067's — response data can be lost.
**Probability: Low** — 066 empirically confirmed Claude Code `PreToolUse` hooks fire inside subagent Bash calls (hook input carries `agent_id`); both `proposal propose` and `proposal withdraw` are in the shipped `gated-commands.txt` today (lines 19, 21); and the drift guard makes registry-side regression a build failure for **both** leaves.
**Controls**:
- **RC-1**: 063's `PreToolUse` write gate interposes human confirmation on every `proposal propose` and `proposal withdraw` Bash call, including the subagent's (empirically confirmed coverage).
- **RC-2**: The drift guard asserts `proposal propose` **and** `proposal withdraw` are each members of 063's gated set — either leaf silently leaving the registry turns CI red (plan ADR-5; per-leaf contract-facts, not count-satisfiable).
- **RC-3**: The agent prompt scopes the write surface to the two bodyless transitions, run as plain `glassfrog` Bash commands the hook can see (no indirection).
**Residual: High × Low = Yellow — accepted.** Justification: the two structural layers (hook + guard) cover the two credible causes, and the guard now pins two leaves. What remains is host-contract risk (H-7). The destructive-`withdraw` branch is the reason this stays a carefully-justified Yellow rather than a Green: the probability is genuinely low, but the impact (lost response data) is acknowledged rather than dismissed.

### H-2 — The path pre-gates a transition client-side on a stale snapshot

**Description**: 068 is the first path that reads `available_transitions` and then invokes the transitions it advertises. The natural drift is "only issue `propose` if the read showed it available" — a client-side gate that forks the server's authority onto a possibly-stale snapshot. The failure is a *refusal*: the agent declines a legitimate transition the server would have accepted, or (mirror) acts on a snapshot that no longer holds.
**Severity: Medium** — no wrong write occurs (the server is never mis-written; a refusal blocks work rather than corrupting data), but a legitimate governance action is silently blocked, and the operator is told the wrong thing about what they can do.
**Probability: Medium** — this is a genuinely tempting shortcut for an LLM agent that just read the transitions, and it cannot be tool-enforced (reading the proposal is legitimate and required — plan ADR-4).
**Controls**:
- **RC-4**: The prompt fence states the reads-inform-never-gate rule explicitly — narrate the snapshot, issue the transition regardless, surface the server's `422` plainly (plan ADR-4; interface § No-pre-gate nuance).
- **RC-5**: The held-out validation scenario "The path never pre-gates a transition client-side" is the enforcement point checklist/analyze both identified (checklist Observation 3).
**Residual: Medium × Medium → Green after control.** The discipline is prompt-level and scenario-verified — the same posture 064's surface-not-judge and 065's no-local-verdict rely on. Accepted as the established operator-layer pattern.

### H-3 — A rejected or failed transition is misreported

**Description**: A transition can fail as the `403` Premium refusal, a `404` unknown proposal, or a `422` transition-not-allowed. If the agent fabricated a resulting state, reported failure as success, or absorbed a `422` as a success end-state, the caller would believe the proposal advanced/withdrew when it did not.
**Severity: High** — a mis-reported transition corrupts the caller's model of the write flow: a phantom "advanced" status could lead a downstream step (or a human) to expect circulation that never started, or to stop watching a proposal that is still `draft`.
**Probability: Low** — the CLI's exit codes and error envelope surface non-2xx failures unambiguously (015/004, via orientation); the no-fabrication contract is pinned by the record shape (`action: none`, no fabricated `proposal`); and the "a `422` is never absorbed as success" invariant is inherited directly from 057/059 and relayed.
**Controls**:
- **RC-6**: The record contract makes failure explicit — `action: none`, the failure named in `notes`, no fabricated `proposal` state; the scenario "A rejected transition fabricates no state" pins it, and the `422`-never-absorbed rule is the 057/059 invariant.
**Residual: High × Low = Yellow — accepted.** Justification: a prompt-level contract backed by an executable scenario, sitting on the shipped, tested CLI error layer (exit codes, error envelopes); the same disposition as 067 H-3.

### H-4 — An incomplete monitoring walk treated as complete

**Description**: If the in-flight `proposal list` fails mid-walk and the partial result were treated as the whole, the monitoring picture (what else is circulating in the circle) would be built on a false view.
**Severity: Medium** — a misleading circulation picture, though it drives no write directly (monitoring is read-only); the harm is a wrong impression, not a wrong action.
**Probability: Low** — 056 already flags a partial walk as incomplete with its cause; the agent's contract is to relay the flag, pinned by the scenario "A failed monitoring walk yields a partial picture."
**Controls**:
- **RC-7**: The relayed incomplete flag + the partial-picture scenario — partial is always labeled partial (Constitution VI; spec Monitoring accord).
**Residual: Medium × Low = Green.**

### H-5 — Boundary bleed into drafting or the response side

**Description**: After a withdraw it is tempting to re-edit the change set and re-create (`proposal create`, 067's write); while monitoring, to record a `no_objection`/`bring_to_meeting` (`proposal respond`, 069's write).
**Severity: Medium** — an unwanted create or response is a real governance event, but each is itself gated by 063, so a human sees it before it executes.
**Probability: Low** — the prompt fence names the forbidden leaves; the validation scenarios "The path records no consent response" and "circulates without judging" assert absence, and the withdraw-handoff is a named workflow step (back to 067, not a self-create).
**Controls**:
- **RC-8**: Prompt fence — the agent's two writes are `proposal propose`/`withdraw`; `create`/`respond` are named as forbidden (interface § Identity & scope).
- **RC-9**: 063 gates every proposal-write leaf regardless — the backstop confirmation would surface any bleed to the human.
**Residual: Medium × Low = Green.**

### H-6 — Artifact/CLI drift

**Description**: The artifacts name four command leaves and depend on a two-in-two-out registry invariant. A CLI rename/removal would falsify the guidance; a gate-registry edit that pulls a transition **out** of the gated set silently loses the confirmation property (escalates to H-1), while pulling a read **into** it would make monitoring start prompting.
**Severity: Medium** — an agent following stale guidance mis-drives the CLI; the registry-side branch escalates to H-1 (lost confirmation) or degrades monitoring UX.
**Probability: Medium** — surfaces change across releases; this is the failure mode every sibling guarded.
**Controls**:
- **RC-10**: The best-effort `internal/build` drift guard — composed leaves exist in the registry; `proposal propose` ∈ **and** `proposal withdraw` ∈ the gated set; the two reads ∉ it; both writes anchored as per-leaf contract-facts so a read/write swap is caught rather than count-satisfied; all sides source-derived (plan ADR-5).
**Residual: Medium × Medium → Green after control** — the guard converts silent drift into a build failure on both membership directions; what it deliberately does not cover (flags, prose, the prompt-level no-pre-gate rule, parser) is stated in the test, not silent.

### H-7 — Host does not honour subagents, grants, or hooks

**Description**: Isolation, the fenced grant, and the two in-subagent gates all assume a capable host. On a host without hook support, RC-1 is absent and a transition could execute unconfirmed (H-1's worst branch, including the destructive withdraw); without subagent support, the path degrades to caller-context guidance.
**Severity: High** — same failure class as H-1 when hooks are absent while writes still execute.
**Probability: Low** — the target host is Claude Code, where coverage is confirmed; distribution/host targeting is owned by Operating-Surface Packaging (#70).
**Controls**:
- **RC-11**: Documented degradation — without the agent, the skill remains guidance only; the gated-writes note tells any operator that both transitions require the confirmed flow.
- **RC-3**: (shared) the prompt-level narration keeps the presentation layer even where the structural layer is missing.
**Residual: High × Low = Yellow — accepted.** Justification: the host contract is external to this feature; #70 owns targeting capable hosts, and the prompt layer plus the CLI's own non-interactive safety (server permissions, the transition-authorization the API enforces) bound the damage. This shares H-1's honest caveat: on a hookless host, the destructive `withdraw` branch is real.

### H-8 — Skill/agent prose drift

**Description**: Two artifacts describe one workflow; unsynchronized edits could diverge them.
**Severity: Low** — inconsistent guidance, caught at review or by confused output; no write-safety impact.
**Probability: Medium** — plain prose, no automated guard covers it.
**Controls**:
- **RC-12**: The workflow is single-sourced in the skill and referenced by the agent (plan ADR-1); review carries what the guard cannot.
**Residual: Low × Medium = Green.**

### H-9 — Confirmation fatigue on a destructive transition

**Description**: 068 is the first path where a single session can cross the gate repeatedly (advance → withdraw → re-advance). A fatigued practitioner may rubber-stamp a confirmation — and one of the two transitions, `withdraw`, is destructive (deletes prior responses).
**Severity: Medium** — a rubber-stamped `withdraw` loses response data; a rubber-stamped `propose` starts an unwanted consent window. Both are governance events, both were at least surfaced.
**Probability: Low** — the friction is by design (spec: "exactly two gated governance writes… each always through the guardrail's confirmed flow"), and each confirmation is independent and specific.
**Controls**:
- **RC-13**: Each transition confirms **independently** — never batched or pre-authorized (plan ADR-3) — so no single confirmation waves through both.
- **RC-3**: (shared) the narration layer keeps each prompt informative (the specific bodyless command + what the transition will do, including that withdraw deletes responses), so the confirmation is a real decision, not a rote one.
**Residual: Medium × Low = Green.** Accepted — per-transition confirmation with informative narration is the spec's explicit design; eliminating the friction entirely would defeat the guardrail.

---

## Residual Risk Summary

No Red residuals. Three Yellow residuals — H-1 (unconfirmed transition), H-3 (misreported transition), H-7 (host contract) — each accepted with documented justification. Unlike 067, the bounding property is **not** uniformly "a visible, withdrawable draft": the destructive `withdraw` branch (irreversible deletion of prior responses) means H-1/H-7's worst case can lose response data on a hookless host or an unconfirmed execution. This is called out honestly rather than smoothed over; the probability stays Low because both transition leaves are gated in the shipped registry and pinned by the drift guard, and the target host has confirmed hook coverage. Six Green, including the two circulation-specific hazards (H-2 pre-gating, H-9 fatigue) that 067 did not face.

---

## Traceability Index

| Hazard | Source section |
|---|---|
| H-1 | spec § Crossing the guardrail; plan ADR-3; interface § Gated-write nuance |
| H-2 | spec § Monitoring + Non-Behaviors (no client pre-gate); plan ADR-4; interface § No-pre-gate nuance |
| H-3 | spec § Driving Scenarios (A transition is rejected); interface § Error Communication |
| H-4 | spec § Driving Scenarios (A monitoring read fails); interface § Error Communication |
| H-5 | spec § Non-Behaviors (no create/respond); plan Risks (boundary bleed) |
| H-6 | spec § Staying within the operator layer; plan ADR-5 + Risks (gate-registry drift) |
| H-7 | plan Risks (host support); interface § Error Communication (degradation rows) |
| H-8 | plan Risks (workflow drift) |
| H-9 | spec § Crossing the guardrail (independent confirmations); plan Risks (two-write confirmation fatigue) |

| Control | Architectural grounding |
|---|---|
| RC-1 | 063 `PreToolUse` gate (`plugin/hooks/glassfrog-write-gate.sh`), subagent coverage confirmed by 066, both transition leaves gated |
| RC-2 | Drift guard two-write gated-membership assertion (plan ADR-5; T002) |
| RC-3 | Agent prompt: two bodyless write leaves, no indirection, informative narration (plan ADR-3; T001) |
| RC-4 | Prompt fence: reads-inform-never-gate (plan ADR-4; T001) |
| RC-5 | Held-out validation scenario "never pre-gates a transition client-side" (feature file) |
| RC-6 | Circulation-record contract: `action: none`, no fabricated state; `422`-never-absorbed (interface § Output contract; 057/059 invariant) |
| RC-7 | 056's incomplete flag, relayed (interface § Error Communication) |
| RC-8 | Prompt fence naming the forbidden proposal-write leaves `create`/`respond` (T001) |
| RC-9 | 063 gates all proposal-write leaves as backstop (shipped) |
| RC-10 | `internal/build` drift guard, two-write membership both ways, all sides source-derived (T002) |
| RC-11 | Documented degradation-to-guidance (T001; interface § Error Communication) |
| RC-12 | Single-sourced workflow, agent references the skill (plan ADR-1; T001) |
| RC-13 | Independent per-transition confirmation, never batched (plan ADR-3; interface § Confirmation contract) |

---

## Scenario Coverage Note

Scenarios pre-exist this risk round (guard runs post-shape in this project). Spot-check: H-1 → "The path routes both writes through the guardrail" + "An unconfirmed transition leaves the record untouched"; H-2 → "A stale snapshot does not stop a transition" + "The path never pre-gates a transition client-side"; H-3 → "A rejected transition fabricates no state"; H-4 → "A failed monitoring walk yields a partial picture"; H-5 → "The path records no consent response" + "circulates without judging authority or coaching"; H-6 → "The path names no command the CLI lacks"; H-7 → "A missing circulator degrades the path to guidance"; H-9 → "Two transitions in one session confirm twice". H-8 (prose drift) has no dedicated scenario — its control is a review concern, not scenario-testable as declarative prose. No test gap requiring action.
