# Risk: Response Recording

**Feature**: 058-response-recording
**Round**: 1
**Date**: 2026-06-15
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix found in PROJECT.md)
**Degradation flags**: none — spec, plan, and one interface file all present. No Regulatory Context in PROJECT.md → no IEC 14971 regulatory bridge.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | A valid-but-unintended consent value is recorded on live governance (e.g. `no_objection` recorded where the practitioner meant `bring_to_meeting`), letting a proposal auto-accept that should have been discussed | spec §Behavioral Accord (Input), consent model | High | Medium | Red→ | RC-1, RC-2, RC-3 | **Yellow** |
| H-2 | The Premium plan-gate `403` is surfaced as a generic permission error, leaving the operator unable to tell "not on your plan" from "no access to this proposal" | spec §Failure; interface §Error Communication; plan §Cross-cutting (Premium gating) | Low | Medium | Green | RC-4 | **Green** |
| H-3 | A second response by the same person is mishandled — retried, or the `422` swallowed/folded into success — misreporting consent state | spec §Non-Behaviors, §Edge (second response); plan §Cross-cutting (one-per-person) | Medium | Low | Green | RC-5, RC-6 | **Green** |
| H-4 | The token leaks into output, or the client spoofs the responding person | spec §Non-Behaviors (no person; no token); CONSTITUTION II | High | Low | Yellow | RC-7, RC-8 | **Yellow** |
| H-5 | A response is recorded against an unintended proposal (the `prp_` id is passed through unvalidated) | spec §Assumptions ([ASSUMED] id not validated); interface §Surface (id pass-through) | Medium | Low | Green | RC-3, RC-9 | **Green** |
| H-6 | A recording that did not happen is presented as success (silent failure / render error) | spec §Failure; plan §Cross-cutting (buffer-then-write); CONSTITUTION III | Medium | Low | Green | RC-6, RC-10 | **Green** |

**Unacceptable (Red) residual risks: none.** Two Yellow residuals (H-1, H-4) are acceptable with the documented justifications below.

---

## Hazard Detail

### H-1 — Unintended consent value on live governance
- **Description**: Recording a response moves a circulating proposal toward (or away from) auto-acceptance. A wrong-but-valid value — especially `no_objection` when discussion was intended — can let a governance change pass that should have been escalated. This is hard to reverse once `accepted`.
- **Severity — High**: the blast radius is the org's governance structure; auto-acceptance alters roles/accountabilities/policies and is not trivially undone (plan §System Architecture; PROJECT.md domain — "the only way to change governance is the proposal flow").
- **Probability — Medium**: the operator is usually an AI agent; the two values are semantically close, and validation confirms only that the value is *in the set*, not that it matches intent. Lowered by the required-no-default flag (the agent cannot drift into a value by omission).
- **Controls**: **RC-1**, **RC-2**, **RC-3**.
- **Residual — Yellow** (High × Low after controls): severity is irreducible (the act is intrinsically governance-altering), but probability drops to Low because the value must be a deliberate, valid, explicitly-chosen token and the result's `proposal_status` is surfaced for the agent to detect an unexpected `accepted`. **Justification**: intent-correctness is ultimately the operator's responsibility; the CLI's duty is to guarantee a valid, deliberate value and to make the outcome legible — both met. Operator-layer confirmation is **Write-Safety Guardrail**, a separate capability (spec non-behavior), which would further reduce probability when it lands.

### H-2 — Premium plan-gate surfaced generically
- **Description**: On a non-Premium org every attempt returns `403`; rendered as a generic permission error, the agent may retry uselessly or misdiagnose access.
- **Severity — Low**: no data harm; a less-actionable message only.
- **Probability — Medium**: certain to occur on any non-Premium org.
- **Controls**: **RC-4**.
- **Residual — Green**: the shared chain surfaces the HTTP status and the RFC 9457 `detail` (015), which typically names the plan gate. **Plan-Limit Signal** (a later spec) sharpens this into an explicit "not available on your plan" diagnostic — noted, deliberately out of scope (spec non-behavior).

### H-3 — Double-response / `422` mishandling
- **Description**: A re-run or retry could record (or appear to record) a second response, or the `422` could be swallowed as success.
- **Severity — Medium**: a false "recorded" misleads the agent about consent state.
- **Probability — Low**: the architecture prevents it — `POST` is not auto-retried on `429` (017 `isSafeMethod`), and the `422` routes through the shared classifier as `APIError(3)`, never folded into success.
- **Controls**: **RC-5**, **RC-6**.
- **Residual — Green**: accepted.

### H-4 — Token leak or person spoofing
- **Description**: A write command could expose the token in output, or let the client claim a response on another person's behalf.
- **Severity — High**: credential exposure / governance-attribution integrity.
- **Probability — Low**: the token rides the transport, never a model field or render value; the request body carries no person field (the server derives identity).
- **Controls**: **RC-7**, **RC-8**.
- **Residual — Yellow** (High × Low): severity is intrinsic to any authenticated write; probability is Low given landed seams plus an explicit BDD tripwire. **Justification**: both controls are enforced by landed mechanisms (007 transport / 032 rendering) and pinned by a validation scenario + transport tripwire (no person in body; token never printed).

### H-5 — Recording against an unintended proposal
- **Description**: The `prp_` id is passed through unvalidated, so a wrong id could target the wrong proposal.
- **Severity — Medium**: consent recorded on the wrong proposal.
- **Probability — Low**: a malformed/unknown id `404`s cleanly (no silent mis-target); a valid-but-wrong id is operator input the CLI cannot distinguish — the same id-pass-through class as every other command (025 ADR-4 / 056 ADR-3).
- **Controls**: **RC-3**, **RC-9**.
- **Residual — Green**: accepted — consistent with the project-wide id-pass-through precedent; the recorded result echoes the target for confirmation.

### H-6 — Silent failure presented as success
- **Description**: A render error or swallowed failure could present a non-recorded response as success (CONSTITUTION III).
- **Severity — Medium**: the agent acts on a false belief that consent was recorded.
- **Probability — Low**: buffer-then-write means a render failure leaves stdout empty and maps to `RuntimeError(1)`; every non-2xx routes through `reportFailure`.
- **Controls**: **RC-6**, **RC-10**.
- **Residual — Green**: accepted.

---

## Risk Controls

| RC-ID | Control (assessment level) | Mitigates |
|---|---|---|
| RC-1 | Closed-enum validation accepts only `no_objection` / `bring_to_meeting` before any request | H-1 |
| RC-2 | `--response` is required with no default, forcing a deliberate explicit choice (an omitted value is a usage error) | H-1 |
| RC-3 | The recorded result surfaces the `proposal_id` and parent `proposal_status`, letting the operator confirm the target and detect unexpected auto-acceptance | H-1, H-5 |
| RC-4 | Non-2xx failures surface the HTTP status and the API's RFC 9457 `detail` through the shared error chain (015) | H-2 |
| RC-5 | The non-idempotent `POST` is never auto-retried on `429` (017 `isSafeMethod`), so no rate-limit retry can double-record | H-3 |
| RC-6 | Every non-2xx routes through the shared classifier and `reportFailure`; the `422` is surfaced as a real failure, never folded into success | H-3, H-6 |
| RC-7 | The token rides the transport layer only — never a model field, never rendered | H-4 |
| RC-8 | The request body carries no person field; the server derives the responding identity (pinned by a validation scenario + transport tripwire) | H-4 |
| RC-9 | An unknown/invisible `prp_` id surfaces a clean `404` through the shared chain — no silent mis-target for a non-existent proposal | H-5 |
| RC-10 | Buffer-then-write: a render failure leaves stdout empty and maps to `RuntimeError(1)` — no partial or false-success output | H-6 |

---

## Residual Risk Summary

| Residual | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (acceptable, justified) | 2 | H-1 (governance auto-acceptance — severity intrinsic), H-4 (token/attribution — severity intrinsic) |
| Green (accepted) | 4 | H-2, H-3, H-5, H-6 |

No control is deferred or unimplemented within this feature's scope. RC-1/RC-2/RC-3 map to tasks T003 (validator) and T004 (command render + `proposal_status`); RC-5/RC-6/RC-7/RC-8/RC-9/RC-10 are landed shared seams that T004 reuses and T005's BDD tripwires pin. The two Yellow residuals are bounded by severity that the feature cannot reduce (any consent write is governance-altering; any authenticated write handles a credential) and probability already driven to Low by the listed controls.

---

## Traceability Index

| ID | Grounding |
|---|---|
| H-1 | spec §Behavioral Accord (Input), §User Scenarios; PROJECT.md domain (proposal flow) |
| H-2 | spec §Failure; interface §Error Communication; plan §Cross-cutting (Premium gating) |
| H-3 | spec §Non-Behaviors (no retry/special-handling of 422), §Edge (second response); plan §Cross-cutting (one-per-person) |
| H-4 | spec §Non-Behaviors (no person, no token); CONSTITUTION II |
| H-5 | spec §Assumptions ([ASSUMED] id not validated client-side); interface §Surface (id pass-through) |
| H-6 | spec §Failure; plan §Cross-cutting (buffer-then-write); CONSTITUTION III |
| RC-1, RC-2 | plan ADR-1 (required closed-enum `--response` validated fail-fast); tasks T003 |
| RC-3 | plan ADR-2 (`proposal_status` surfaced); interface §Output; tasks T002/T004 |
| RC-4, RC-6 | API Error Extraction (015); Output-Aware Failure Rendering (032); tasks T004 |
| RC-5 | Rate-Limit Handling (017) `isSafeMethod`; plan §Cross-cutting |
| RC-7, RC-8 | Request Authentication (007) transport; spec validation scenarios; tasks T004/T005 |
| RC-9 | API Error Extraction (015); interface §Surface (id pass-through); tasks T004 |
| RC-10 | plan §Cross-cutting (buffer-then-write → RuntimeError(1)); tasks T004 |
