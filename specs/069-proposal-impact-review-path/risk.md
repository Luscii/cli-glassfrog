# Risk: Proposal Impact Review Path

**Feature**: 069-proposal-impact-review-path
**Round**: 1
**Date**: 2026-07-19
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Degradation flags**: none — full artifact set present. Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. No Regulatory Context in PROJECT.md — regulatory bridge omitted.

---

## Risk Register

| ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | The gated `respond` executes without human confirmation | spec § Crossing the guardrail; plan ADR-3; interface § Gated-write nuance | High | Low | RC-1, RC-2, RC-3 | **Yellow** (accepted with justification) |
| H-2 | The path decides the operator's response — fabricates an objection verdict or auto-fills `--response` from the review | spec § Reviewing + Non-Behaviors (never decide); plan ADR-3 + ADR-4b; interface § Never-decide nuance | High | Medium→Low | RC-4, RC-5, RC-6, RC-1 | **Yellow** (accepted with justification) |
| H-3 | A rejected or failed `respond` is misreported (fabricated state, or a non-2xx absorbed as success) | spec § Recording a consent response (rejected); interface § Error Communication | High | Low | RC-7 | **Yellow** (accepted with justification) |
| H-4 | A false "no impact on you" from a silently-incomplete `me roles` footprint | spec § Reviewing (footprint honesty); plan ADR-4a; interface § footprint_coverage | Medium | Medium | RC-8, RC-9 | **Yellow** (accepted with justification) |
| H-5 | Boundary bleed: the path advances/withdraws (068) or creates (067) beyond its one `respond` | spec § Non-Behaviors; plan Risk (fence erosion) | Medium | Low | RC-10, RC-11 | Green |
| H-6 | Artifact/CLI drift: a named leaf changes, or `respond` leaves / a read enters 063's gated set | spec § Staying within the operator layer; plan ADR-5 | Medium | Medium | RC-12 | Green |
| H-7 | The host does not honour subagents, tool grants, or hooks | plan Risk (divergent locus); interface § Error Communication | High | Low | RC-13, RC-3 | **Yellow** (accepted with justification) |
| H-8 | Skill and agent workflow prose diverge | plan Risk (family pattern) | Low | Medium | RC-14 | Green |
| H-9 | Split-locus violation: the reviewer subagent runs the `respond` itself | spec § Handoff; plan ADR-3; interface § Identity & scope | Medium | Low | RC-11, RC-15, RC-9 | Green |

**Unacceptable (Red) residual risks: none.**

---

## Hazard Detail

### H-1 — The gated `respond` executes without human confirmation

**Description**: `proposal respond` is the one write the Write-Safety Guardrail exists to gate on this path. If it executed unconfirmed — because the hook did not fire, because the leaf left 063's gated registry, or because the write was routed around the gate — a consent response would be recorded without the explicit human confirmation the design promises.
**Severity: High** — an unconfirmed consent response is a real governance event: a `no_objection` moves the proposal toward auto-acceptance and, when it completes the expected set, **triggers server-side auto-acceptance of the governance change** — the most consequential terminal state in the whole proposal family. This path's worst case reaches an *accepted* change, which 067 (a withdrawable draft) and 068 (a reversible transition, save the destructive withdraw) do not.
**Probability: Low** — unlike 067/068 the write runs in the **caller** context (plan ADR-3), the ordinary case for a `PreToolUse` hook (no in-subagent dependency needed for the gate); `proposal respond` is in the shipped `gated-commands.txt` today; and the drift guard makes registry-side regression a build failure.
**Controls**:
- **RC-1**: 063's `PreToolUse` write gate interposes human confirmation on every `proposal respond` Bash call in the caller context — the value rides inline so the human sees the exact `--response` payload.
- **RC-2**: The drift guard asserts `proposal respond` is a member of 063's gated set — the leaf silently leaving the registry turns CI red (plan ADR-5; per-leaf contract-fact).
- **RC-3**: The skill issues the write as a plain `glassfrog` Bash command with the value inline (no indirection, no file/stdin source the hook can't display).
**Residual: High × Low = Yellow — accepted.** Justification: the caller-context locus makes the gate the standard hook case (more robust than 067/068 on the in-subagent axis), two structural layers (hook + guard) cover the credible causes, and what remains is host-contract risk (H-7). Held at a carefully-justified Yellow rather than Green because the terminal impact (an accepted governance change) is the family's highest.

### H-2 — The path decides the operator's response

**Description**: This path's signature hazard, novel to 069. The review draws the impact; the operator must choose. The natural drift for an LLM is to collapse the two — "the picture shows no objections, so answer `no_objection`" — fabricating an objection verdict inside the picture or auto-filling `--response` from the review's content. That forks the consent judgment the operator owns.
**Severity: High** — an auto-chosen `no_objection` the operator did not decide can, if confirmed, trigger auto-acceptance of a change the operator never actually consented to — the same terminal impact as H-1, reached by a different route (a decision the operator didn't make rather than a write they didn't confirm).
**Probability: Medium → Low after controls** — genuinely tempting for an LLM that just synthesized the impact, and the prompt-level half cannot be fully tool-enforced; but the structural half (below) and the fact that 063 still surfaces the exact chosen value for human confirmation before any write drop the realized probability to Low.
**Controls**:
- **RC-4**: The impact-picture output shape has **no verdict field** — there is structurally nowhere to emit a "must object" / "may consent" ruling (interface § Output contract; plan ADR-4b).
- **RC-5**: The reviewer agent's zero-write fence means the thing that draws the picture cannot record any answer — the decision input enters only from the operator, in the caller context (plan ADR-3, the split locus). "Reviews inform, never decide" is thus partly *structural*, not only prompt-level.
- **RC-6**: The skill's no-inferred-value rule forbids deriving `--response` from the review ("no objections found" is not an instruction to answer `no_objection`), verified by the held-out validation scenario "The picture carries no verdict and chooses no response" (checklist Observation 3).
- **RC-1**: (shared) even if a value were auto-chosen, 063 surfaces the exact `--response` payload for human confirmation before it executes — the last-line human check.
**Residual: High × Low = Yellow — accepted.** Justification: this is the path's strongest-controlled hazard — a structural layer (verdict-free shape + write-incapable reviewer) beneath the prompt-level rule and the scenario, with 063's confirmation as the final backstop. Held at Yellow (not Green) honestly because the terminal impact is auto-acceptance and the prompt half of the never-decide rule is not tool-enforceable.

### H-3 — A rejected or failed `respond` is misreported

**Description**: A `respond` can fail as the `403` Premium refusal, a `404` unknown/invisible proposal, or a `422` the server does not allow (e.g. the operator already responded). If the agent fabricated a resulting state, reported failure as success, or absorbed a `422` as a success end-state, the caller would believe a response was recorded when it was not.
**Severity: High** — a mis-reported response corrupts the caller's model of the consent flow: a phantom "recorded / accepted" could lead a human or downstream step to believe the proposal was answered when it was not, or to stop watching a proposal still awaiting them.
**Probability: Low** — the CLI's exit codes and error envelope surface non-2xx failures unambiguously (015/004, via orientation); the no-fabrication contract is pinned by the record shape (nothing recorded, the failure named in `notes`); no retry follows (a retry is itself a gated write needing fresh confirmation, 063 ADR-5).
**Controls**:
- **RC-7**: The recorded-response contract makes failure explicit — nothing recorded, the failure named, no fabricated `prr_`/status; the scenario "A rejected response fabricates no state" pins it, and the plan-gate `403` renders through 061's possibility-framed wording.
**Residual: High × Low = Yellow — accepted.** Justification: a prompt-level contract backed by an executable scenario, sitting on the shipped, tested CLI error layer; same disposition as 067 H-3 / 068 H-3.

### H-4 — A false "no impact on you" from a silently-incomplete footprint

**Description**: The one composed read that does not paginate is `me roles` (012: silently-incomplete list + stderr note, exit 0). If the reviewer treats a first-page footprint as the whole, a change that actually touches a role on an unshown page would be reported as "does not touch your current governance" — the operator consents to a change that in fact affects them.
**Severity: Medium** — a misleading impact picture that could steer the operator toward `no_objection` on a change that touches an unshown role; the harm is a wrong impression feeding a real decision, though 063 still surfaces the eventual write for confirmation.
**Probability: Medium** — this is the subtlest instruction in the reviewer prompt (the plan flagged it medium/medium), the control is prompt-level, and `me roles`'s silent incompleteness is exactly the trap the 065/#155 memory records.
**Controls**:
- **RC-8**: The tri-state `footprint_coverage` (`complete`/`incomplete`/`unknown`, never a silent `complete`) plus the qualified no-impact wording ("not in the roles visible to this read — list incomplete") — the reviewer must read the incompleteness signal (stderr in human formats, in-band pagination metadata in `json`/`yaml`) and carry it forward (plan ADR-4a; interface footprint honesty).
- **RC-9**: (shared) the "fail toward surfacing" default — when unsure whether a change touches the operator, show it — biases the picture away from a confident negative.
**Residual: Medium × Medium = Yellow — accepted.** Justification: held honestly at Yellow (not smoothed to Green) because the control is prompt-level and this is the path's subtlest correctness instruction; the validation scenario "An incomplete footprint qualifies the no-impact conclusion" is its enforcement point, and 063's confirmation of the eventual response bounds the downstream damage. This is the footprint analogue of 068's carefully-justified destructive-branch Yellow.

### H-5 — Boundary bleed into circulation or drafting

**Description**: While reviewing it is tempting to advance/withdraw the proposal (`proposal propose`/`withdraw`, 068's writes) or, after judging, to re-open drafting (`proposal create`, 067's write) — beyond this path's single `respond`.
**Severity: Medium** — an unwanted advance/withdraw/create is a real governance event, but each is itself gated by 063, so a human sees it before it executes.
**Probability: Low** — the reviewer fence forbids all four proposal writes; the skill's only write is `respond`; the validation scenario "The path performs no circulation or creation step" asserts absence; advancing/withdrawing are named handoffs to 068.
**Controls**:
- **RC-10**: Prompt fence — the reviewer forbids `create`/`propose`/`withdraw` (and its own `respond`); the skill's sole write is `respond` (interface § Identity & scope).
- **RC-11**: (shared) 063 gates every proposal-write leaf regardless — the backstop confirmation would surface any bleed to the human.
**Residual: Medium × Low = Green.**

### H-6 — Artifact/CLI drift

**Description**: The artifacts name ten command leaves and depend on a one-in-nine-out registry invariant. A CLI rename/removal would falsify the guidance; a gate-registry edit that pulls `respond` **out** of the gated set silently loses the confirmation property (escalates to H-1), while pulling a read **into** it would make the review start prompting.
**Severity: Medium** — an agent following stale guidance mis-drives the CLI; the registry-side branch escalates to H-1 (lost confirmation) or degrades the review UX.
**Probability: Medium** — surfaces change across releases; this is the failure mode every sibling guarded.
**Controls**:
- **RC-12**: The best-effort `internal/build` drift guard — the ten composed leaves exist in the registry; `proposal respond` ∈ the gated set; the nine reads ∉ it; the write anchored as a per-leaf contract-fact so a read/write swap is caught rather than count-satisfied; all sides source-derived (plan ADR-5).
**Residual: Medium × Medium → Green after control** — the guard converts silent drift into a build failure on both membership directions; what it deliberately does not cover (flags, prose, the footprint-honesty and never-decide prompt rules, parser) is stated in the test, not silent.

### H-7 — Host does not honour subagents, grants, or hooks

**Description**: The reviewer's isolation, its read-only grant, and the caller-context write gate all assume a capable host. On a host without hook support, RC-1 is absent and the `respond` could execute unconfirmed (H-1's worst branch, reaching auto-acceptance); without subagent support, the review degrades to caller-context guidance (losing synthesis-not-raw and the read fence).
**Severity: High** — same failure class as H-1 when hooks are absent while the write still executes.
**Probability: Low** — the target host is Claude Code, where hook coverage is confirmed; and this path's write gate is *less* host-fragile than 067/068 because it needs only ordinary caller-context hook firing, not in-subagent coverage. Distribution/host targeting is owned by Operating-Surface Packaging (#70).
**Controls**:
- **RC-13**: Documented degradation — without the agent, the skill remains guidance only; the decision-and-respond note tells any operator that the response requires the confirmed flow.
- **RC-3**: (shared) the inline-value narration keeps the presentation layer even where the structural layer is missing.
**Residual: High × Low = Yellow — accepted.** Justification: the host contract is external to this feature; #70 owns targeting capable hosts, and the prompt layer plus the CLI's own non-interactive safety (the server authorizes who may respond, and a second response is a `422`) bound the damage. Slightly more robust than 068's H-7 on the write axis: the caller-context locus removes the in-subagent-hook dependency for the gate itself.

### H-8 — Skill/agent prose drift

**Description**: Two artifacts describe one workflow; unsynchronized edits could diverge them.
**Severity: Low** — inconsistent guidance, caught at review or by confused output; no write-safety impact.
**Probability: Medium** — plain prose, no automated guard covers it.
**Controls**:
- **RC-14**: The workflow is single-sourced in the skill and referenced by the agent (plan ADR-1); review carries what the guard cannot.
**Residual: Low × Medium = Green.**

### H-9 — Split-locus violation: the reviewer runs the `respond`

**Description**: 069 is the only path whose gated write is deliberately kept *out* of the subagent (plan ADR-3). The drift risk is the reviewer running `proposal respond` itself, collapsing the split locus and recording a response from the isolated context that has no channel to carry the operator's decision.
**Severity: Medium** — even if the reviewer ran it, 063 still gates `respond` in any context (caller or subagent), so a human still confirms the exact value; the harm is the loss of the intended decision-in-the-caller flow, not an unconfirmed write.
**Probability: Low** — the reviewer's tool grant withholds `Write`/`Edit` and its prompt fence forbids all proposal writes including `respond`; the validation scenario "The reviewer hands the respond step back to the caller" asserts the refusal-and-handoff.
**Controls**:
- **RC-11**: (shared) 063 gates `respond` regardless of context — a human confirms even a mislocated write.
- **RC-15**: The reviewer's all-proposal-writes fence (the family's strictest) + the "hands the respond step back to the caller" scenario (interface § Identity & scope).
- **RC-9**: (shared) the reviewer's read-posture identity ("returns only the impact picture") keeps the write out of its output contract.
**Residual: Medium × Low = Green.** Accepted — the split locus is enforced by fence + grant + scenario, and 063 is the safety backstop even if the locus is violated.

---

## Residual Risk Summary

No Red residuals. Five Yellow residuals — H-1 (unconfirmed `respond`), H-2 (decides for the operator), H-3 (misreported `respond`), H-4 (false no-impact from an incomplete footprint), H-7 (host contract) — each accepted with documented justification. This path carries two more Yellows than 068 (three), both from its distinctive shape: H-2, the never-decide hazard, and H-4, the footprint-honesty hazard, neither of which the proposer-side paths faced. The bounding property is the family's *strongest* terminal impact — an accepted governance change via auto-acceptance — so H-1/H-2/H-3 are held honestly at Yellow rather than Green even though their probabilities are Low; the controls are correspondingly the strongest in the family (H-2 uniquely gains a *structural* never-decide layer: a verdict-free output shape and a write-incapable reviewer, beneath the prompt rule and the 063 confirmation). Four Green. Two axes are actually *more* robust than 068: the caller-context write locus removes the in-subagent-hook dependency for the gate (H-1, H-7), and the single write eliminates 068's repeated-cycle confirmation-fatigue hazard.

---

## Traceability Index

| Hazard | Source section |
|---|---|
| H-1 | spec § Crossing the guardrail; plan ADR-3; interface § Gated-write nuance |
| H-2 | spec § Reviewing + Non-Behaviors (never decide); plan ADR-3 + ADR-4b; interface § Never-decide nuance |
| H-3 | spec § Driving Scenarios (A response is rejected); interface § Error Communication |
| H-4 | spec § Reviewing (footprint honesty); plan ADR-4a; interface § footprint_coverage / Error Communication |
| H-5 | spec § Non-Behaviors (no advance/withdraw/create); plan Risks (fence erosion) |
| H-6 | spec § Staying within the operator layer; plan ADR-5 |
| H-7 | plan Risks (divergent locus / host); interface § Error Communication (degradation rows) |
| H-8 | plan Risks (family pattern / workflow drift) |
| H-9 | spec § Handoff; plan ADR-3 (split locus); interface § Identity & scope |

| Control | Architectural grounding |
|---|---|
| RC-1 | 063 `PreToolUse` gate (`plugin/hooks/glassfrog-write-gate.sh`), caller-context firing, inline value displayed |
| RC-2 | Drift guard one-write gated-membership assertion (plan ADR-5; T002) |
| RC-3 | Skill: `respond` as a plain Bash command, value inline, informative narration (plan ADR-3; T001) |
| RC-4 | Verdict-free impact-picture output shape (interface § Output contract; plan ADR-4b) |
| RC-5 | Reviewer zero-write fence — the reviewer cannot record any answer (plan ADR-3; T001) |
| RC-6 | Skill no-inferred-value rule + held-out validation scenario "carries no verdict and chooses no response" (T001; feature file) |
| RC-7 | Recorded-response contract: nothing recorded on failure, no fabricated state; non-2xx never absorbed (interface § Error Communication) |
| RC-8 | Tri-state `footprint_coverage` + qualified no-impact wording; reviewer reads the `me roles` incompleteness signal (plan ADR-4a; T001) |
| RC-9 | Reviewer read-posture identity + fail-toward-surfacing default (interface § Identity & scope, Footprint honesty; T001) |
| RC-10 | Prompt fence naming the forbidden proposal-write leaves `create`/`propose`/`withdraw` (T001) |
| RC-11 | 063 gates all proposal-write leaves as backstop, any context (shipped) |
| RC-12 | `internal/build` drift guard, one-in-nine-out membership, all sides source-derived (T002) |
| RC-13 | Documented degradation-to-guidance (T001; interface § Error Communication) |
| RC-14 | Single-sourced workflow, agent references the skill (plan ADR-1; T001) |
| RC-15 | Reviewer all-proposal-writes fence + "hands the respond step back to the caller" scenario (T001; feature file) |

---

## Scenario Coverage Note

Scenarios pre-exist this risk round (guard runs post-shape in this project). Spot-check: H-1 → "The path routes its one write through the guardrail" + "An unconfirmed response leaves the record untouched"; H-2 → "The picture carries no verdict and chooses no response"; H-3 → "A rejected response fabricates no state"; H-4 → "An incomplete footprint qualifies the no-impact conclusion" + "A no-impact review is a load-bearing result"; H-5 → "The path performs no circulation or creation step"; H-6 → "The path names no command the CLI lacks"; H-7 → "A missing reviewer degrades the path to guidance"; H-9 → "The reviewer hands the respond step back to the caller". H-8 (prose drift) has no dedicated scenario — its control is a review concern, not scenario-testable as declarative prose. No test gap requiring action.
