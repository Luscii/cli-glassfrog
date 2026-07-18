# Specification: Proposal Circulation Path

**Feature**: 068-proposal-circulation-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Proposal Circulation Path is the **fifth operator path** on the Agent Operating Surface, and the direct downstream sibling of the Proposal Drafting Path (067). It rides on the root *Operator Orientation* (062), alongside the read-only *Governance Navigation Path* (064), the evaluative *Constraint Discovery Path* (065), the ungated *Tension Processing Path* (066), and the gated *Proposal Drafting Path* (067). Where 067 stops at the created `draft` — a submittable proposal carrying its `prp_` id — this path picks up that id and carries it the rest of the way through the consent lifecycle: it advances the draft into circulation, lets the proposer watch it circulate, and — when the proposal needs amending — pulls it back to `draft` so it can re-enter drafting. It is the proposer's circulation-management path, turning a prepared draft into a proposal moving toward acceptance.

The path is knowledge, not capability. It composes already-shipped commands — Advance to Circulation (057) for the `propose` transition, Withdraw Proposal (059) for the `withdraw` transition, and Proposal Reads (056) to read the proposal back and monitor where it stands — and adds no command, flag, or governance logic of its own. It performs **two gated governance writes**: both `propose` and `withdraw` are exactly the boundary the Write-Safety Guardrail (063) exists to guard, so the path surfaces the proposal for confirmation and runs each transition through the guardrail's confirmed write flow, rather than smuggling a gated write ungated; the monitoring read stays ungated. It stops before **recording consent responses** — that is the response side (069), a responder's act on a circulating proposal, not the proposer's circulation management this path owns — and it does not draft or create the proposal (067) or judge whether the change is within authority (065). The exact delivery form — a plugin skill, a proposal-circulation agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a draft proposal is ready to circulate, the path takes its `prp_` id as the starting point — the id handed off by the Proposal Drafting Path (067), or one the practitioner identifies from an existing proposal.
- The path requires an already-existing proposal: it does not draft, create, or assemble the change set — that is the Proposal Drafting Path (067). It reads the proposal back through the existing read command to ground the circulation, but the proposal must already be on the record.
- The entry proposal may be a `draft` the practitioner wants to advance, or an already-circulating proposal the practitioner wants to monitor or withdraw; the path takes the `prp_` id whichever state it is in.

### Monitoring

- When the target proposal's `prp_` id is known, the path reads it back — surfacing its status, `response_summary`, `response_deadline`, and `available_transitions` drawn together — so the proposer can see where the proposal stands and whether it is progressing toward auto-acceptance.
- When the circle's proposals in flight are relevant, the path may surface them — narrowing the proposal list by circle and status — so the proposer situates the target proposal against what else is circulating.
- When the proposals in flight span more than one page, the path pages through the full result set before drawing its picture — never a silent single-page cap.
- The path reads to **inform**, never to gate: it surfaces `available_transitions` so the proposer knows what the server currently offers, but it does not turn that snapshot into a client-side gate — it issues the transition and lets the server authorize, consistent with how Advance to Circulation (057) and Withdraw Proposal (059) issue the transition and surface the server's `422`.

### Advancing into circulation

- When a `draft` is ready to circulate, the path surfaces it for confirmation and — because advancing a proposal is a governance write — runs `proposal propose <prp-id>` through the Write-Safety Guardrail (063) confirmed write flow, rather than issuing the gated transition directly.
- When the advance succeeds, the path returns the advanced proposal exactly as the server returned it — now `proposed_outside_meeting`, carrying its server-set `response_deadline`, the proposer's auto-recorded implicit `no_objection` in `response_summary`, and updated `available_transitions` — so the proposer can monitor it. It synthesizes no fields and narrates none of the side effects the advance triggered.
- When the advance is rejected — a Premium-disabled refusal (`403`), an unknown proposal (`404`), or a transition the server does not allow (`422`) — the path surfaces the failure by name and advances nothing; it does not treat any non-2xx as success or fabricate a state the record does not contain.

### Withdrawing back to draft

- When a circulating proposal needs amending, the path surfaces it for confirmation and runs `proposal withdraw <prp-id>` through the guardrail's confirmed write flow, returning the proposal to `draft`.
- When the withdraw succeeds, the path returns the proposal exactly as the server returned it — now back in `draft`, with `proposed_at` and `response_deadline` cleared and its prior responses deleted server-side, and with updated `available_transitions`. It does not narrate the destructive side effects the withdraw triggered.
- When the withdraw is rejected (`403`, `404`, or `422`), the path surfaces the failure by name and withdraws nothing.

### Crossing the guardrail

- The path performs exactly two gated governance writes — the `propose` and the `withdraw` transitions — and each always through the guardrail's confirmed flow; it neither treats those writes as ungated nor performs any further governance write.
- When a transition is not confirmed, no transition happens — the path issues neither the `propose` nor the `withdraw`.
- The monitoring read is ungated; only the two transitions cross the guardrail.

### Handoff

- When a proposal is circulating and awaiting consent, recording a `no_objection` or `bring_to_meeting` response is the response side (069) — a responder's act, a sibling path; this path does not record responses.
- When a withdraw returns a proposal to `draft`, the path hands the `prp_` id back to the Proposal Drafting Path (067) for re-editing, then a later circulation re-advances it; it does not re-assemble the change set itself.
- When the path returns a result, the proposal it surfaces carries the `prp_` id needed to read it again, advance it, or withdraw it — so circulation bridges back into the CLI's own commands rather than being a dead end.

### Staying within the operator layer

- The path only composes proposal commands the CLI already exposes; it invents no command, flag, or capability, and reimplements no governance, permission, or transition-authorization logic locally.
- When the CLI's proposal command surface changes, the path is expected to stay consistent with it; guidance that names a command the CLI no longer offers is a defect, not a difference of opinion.

---

## User Scenarios

**In order to** open the consent window on a draft I have prepared without hand-assembling the transition command,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** follow a guided path that advances the draft into circulation and shows me where it stands.

**In order to** know whether my circulating proposal is progressing toward acceptance,
**as a** practitioner (served through the agent),
**I want** the path to surface its `response_summary` and `response_deadline` drawn together rather than my reading raw command output.

**In order to** amend a proposal that is already circulating without losing my place,
**as an** AI agent operating the write flow,
**I want** the path to withdraw it back to `draft` and carry the `prp_` id back to the Proposal Drafting Path (067).

---

## Non-Behaviors

- The path must not draft, create, or assemble the change set of a proposal. **Why**: creation and drafting are the Proposal Drafting Path (067); this path begins from a proposal already on the record and only reads it back to ground the circulation.
- The path must not record a consent response (`no_objection` or `bring_to_meeting`). **Why**: recording a response is the response side (069), a responder's act on a circulating proposal; folding it into the proposer's circulation management would fork the staged write flow and claim work that sibling owns.
- The path must not perform the `propose` or `withdraw` transition as an ungated write. **Why**: both are governance writes at exactly the boundary the Write-Safety Guardrail (063) guards; issuing either without the confirmed write flow would smuggle a gated write past the guardrail.
- The path must not pre-read the proposal to gate a transition client-side. **Why**: `available_transitions` is server-owned and time-sensitive, and Advance to Circulation (057) and Withdraw Proposal (059) issue the transition and let the server enforce the rule with a `422`. The path reads to inform the proposer, never to fork that authority or act on a stale snapshot.
- The path must not judge whether the proposal should circulate, or whether the change it carries is within the practitioner's authority. **Why**: that evaluative read is the Constraint Discovery Path (065); ruling on it here would duplicate the judgment in two places.
- The path must not interpret, re-describe, or act on the server-owned side effects the transitions trigger (the computed `response_deadline`, the implicit `no_objection`, the responses a withdraw deletes). **Why**: those are server-owned consequences reflected in the returned proposal; the CLI is a faithful surface, not a Holacracy facilitator (VISION Exclusion 1).
- The path must not add any command, flag, or API capability of its own, nor reimplement governance, permission, or transition-authorization logic locally. **Why**: it is a guided composition of existing commands (knowledge + guardrails, never capability); the API is the source of truth (VISION Exclusion 2), and local logic drifts from the record the CLI faithfully surfaces.
- The path must not teach or coach Holacracy practice, nor dump raw, unsynthesized command output as its result. **Why**: it circulates the proposal through the record, it does not facilitate the governance craft (VISION Exclusion 1); and raw dumps are the rediscovery burden the operating surface exists to remove — the value is the drawn-together circulation picture carrying its id.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing commands — Advance to Circulation (057) for the `propose` transition, Withdraw Proposal (059) for the `withdraw` transition, and Proposal Reads (056) to read the proposal back and monitor it — and defers to the CLI's built-in help for their exact flags. If a command changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Write-Safety Guardrail (063)**: the guardrail gates the proposal write path, and this path crosses that boundary exactly twice — the `propose` and the `withdraw`. The path runs each transition through the guardrail's confirmed write flow; the monitoring read stays ungated.
- **Proposal Drafting Path (067)**: the upstream handoff and the downstream loop. This path picks up the `prp_` draft id 067 produces; and when a withdraw returns a proposal to `draft`, it hands the `prp_` id back to 067 for re-editing.
- **Response side (069)**: the sibling that records consent responses (`no_objection` / `bring_to_meeting`) on a circulating proposal. This path circulates the proposal; it does not record responses.
- **Constraint Discovery Path (065)**: the evaluative read. This path circulates the proposal; it does not judge whether the change is within authority.
- **Glassfrog API**: never touched directly — the CLI mediates every command, and the API authorizes each transition and enforces who may advance or withdraw a proposal. The proposal write surface is Premium-gated: a `403` means async proposals are not enabled. The path only ever does what the caller is permitted to do.

---

## Driving Scenarios

### Happy path

**Scenario: Advance a draft into circulation**
Given a `draft` proposal's `prp_` id whose `available_transitions` include `propose`
When the path surfaces it for confirmation and runs `propose` through the confirmed write flow
Then it returns the proposal now `proposed_outside_meeting`, carrying its `response_deadline` and implicit `no_objection`
And carries the `prp_` id so the proposal can be monitored or withdrawn.

**Scenario: Monitor a circulating proposal**
Given the `prp_` id of a proposal already circulating
When the path reads it back
Then it surfaces the `response_summary`, `response_deadline`, and `available_transitions` drawn together so the proposer sees progress toward acceptance
And it does not compute acceptance itself.

**Scenario: Withdraw a circulating proposal back to draft**
Given a circulating proposal the proposer wants to amend
When the path surfaces it for confirmation and runs `withdraw` through the confirmed write flow
Then it returns the proposal now back in `draft`
And hands the `prp_` id back to the Proposal Drafting Path (067) for re-editing.

### Error scenarios

**Scenario: A transition is rejected**
Given a `propose` or `withdraw` whose proposal is unknown (`404`), whose transition the server does not allow (`422`), or whose organization has async proposals disabled (`403`)
When the path runs the transition through the confirmed write flow
Then it surfaces the API failure by name and transitions nothing
And it does not treat any non-2xx as success or fabricate a state the record does not contain.

**Scenario: A monitoring read fails**
Given a monitoring step where the proposal read (or the circle's in-flight list) fails mid-walk
When the path continues
Then it surfaces what the failure was and presents what it gathered so far, flagged incomplete
And it does not invent the missing data or abandon the whole result.

### Edge cases

**Scenario: The transition must be confirmed before it crosses the boundary**
Given a proposal ready to `propose` or `withdraw`
When the path reaches the transition
Then it routes the write through the Write-Safety Guardrail (063) confirmed flow, surfacing the proposal first
And if the write is not confirmed, no transition happens.

**Scenario: The path reads to inform, not to gate**
Given a proposal whose `available_transitions` the path has read
When it advances or withdraws
Then it issues the transition and lets the server authorize, surfacing a `422` if the server refuses
And it does not pre-gate the call on the read snapshot.

**Scenario: A response belongs to the response side**
Given a circulating proposal awaiting consent
When a member wants to record `no_objection` or `bring_to_meeting`
Then that is the response side (069)
And this path does not record the response itself.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced proposal-circulation-path content
When every command it composes is checked against the shipped CLI
Then each one exists — the path invents no command the CLI does not expose.

**Scenario: Both gated transitions are routed through the guardrail**
Given the path's treatment of the `propose` and `withdraw` transitions
When each is inspected against the Write-Safety Guardrail (063)
Then both run through the confirmed write flow — the path does not issue either gated transition as if it were ungated.

**Scenario: Reads inform, never gate**
Given the path's use of `available_transitions`
When it is inspected for a client-side transition gate
Then it reads to show the proposer where the proposal stands but issues the transition and lets the server authorize — it does not pre-gate the call client-side.

**Scenario: Circulation only, no response recording**
Given the path content
When it is inspected for any `no_objection` / `bring_to_meeting` record step
Then none is present — recording a response is the response side (069).

**Scenario: Circulating, not judging or coaching**
Given the path content
When it is inspected for an authority verdict or Holacracy coaching
Then it neither rules on whether the change is within authority (that is 065) nor advises on governance craft.

**Scenario: Synthesized, not raw**
Given the path's result
When it is inspected against raw command output
Then it is a drawn-together circulation picture carrying the `prp_` id, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a proposal-circulation agent, or both is decided during shaping. (The developer confirmed the form is deferred, mirroring how 062, 064, 065, 066, and 067 deferred their decomposition.)
- **[ASSUMED] Entry is an existing proposal id**: the path starts from a `prp_` id already on the record — a `draft` handed off by 067 (or identified by the practitioner) to advance, or an already-circulating proposal to monitor or withdraw — and reads it back; it does not create the proposal. (The developer confirmed circulation begins from a proposal on the record.)
- **[ASSUMED] Scope is circulation management only**: advance, monitor, and withdraw the proposer's own proposal, then hand off. Response recording is the response side (069), drafting/creation is the Proposal Drafting Path (067), and authority judgment is the Constraint Discovery Path (065). (The developer confirmed withdraw belongs here and that monitoring via the read command is included.)
- **[ASSUMED] Both `propose` and `withdraw` cross the guardrail**: each governance transition runs through the Write-Safety Guardrail (063) confirmed write flow; the monitoring read is ungated. (The developer confirmed the two guardrail crossings.)
- **Composed commands are already shipped** (technical): Advance to Circulation (057), Withdraw Proposal (059), and Proposal Reads (056) all exist in the CLI today, so the path composes them rather than waiting on new commands. (Grounded in the shipped proposal command family.)
- **The transitions are server-authorized and Premium-gated** (technical): each returns `403` when async proposals are disabled and `422` when the transition is not in `available_transitions`, so the path issues the transition and lets the server decide rather than pre-gating. (Grounded in 057's and 059's Behavioral Accords.)

---

## Ambiguity Warnings

None remaining — the entry point, the two guardrail crossings, the read-informs-never-gates boundary, the circulation-only scope (withdraw in, response recording out to 069), and the inclusion of monitoring were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-18

- **Scope boundary**: Circulation covers advancing into circulation (057), monitoring (056), and withdrawing back to draft (059) — the proposer's circulation management. Withdraw belongs here (066 and 067 both explicitly exclude it). Recording consent responses is the response side (069). The developer confirmed both boundaries.
- **Monitoring included**: Reading the circulating proposal back through Proposal Reads (056) — its `response_summary`, `response_deadline`, and `available_transitions` — is in scope, not deferred to the Governance Navigation Path (064). The developer confirmed monitoring is included.
- **Two guardrail crossings**: Both `propose` and `withdraw` run through the Write-Safety Guardrail (063) confirmed write flow; the monitoring read stays ungated.
- **Reads inform, never gate**: The path reads `available_transitions` to show the proposer where the proposal stands, but issues the transition and lets the server authorize (surfacing a `422` if refused), never pre-gating client-side — consistent with 057 and 059.
