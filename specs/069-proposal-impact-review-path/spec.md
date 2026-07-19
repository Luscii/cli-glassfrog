# Specification: Proposal Impact Review Path

**Feature**: 069-proposal-impact-review-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Proposal Impact Review Path is the **sixth operator path** on the Agent Operating Surface, and the responder's counterpart to the proposer-owned Proposal Circulation Path (068). It rides on the root *Operator Orientation* (062), alongside the read-only *Governance Navigation Path* (064), the evaluative *Constraint Discovery Path* (065), the ungated *Tension Processing Path* (066), the gated *Proposal Drafting Path* (067), and the gated *Proposal Circulation Path* (068). Where 068 is the proposer moving their own draft through circulation, this path picks up a proposal that is *already circulating* and works it from the other side of the consent window: it shows the operator **what the proposal would change once it lands, and how that change lands on the operator's own governance** — the roles they fill, the actions they hold, the projects they carry — and then, when the operator has decided, records their consent response. It is the responder's review-and-respond path, turning a circulating proposal from a wall of change into a clear picture of the operator's new world.

The path is knowledge, not capability. It composes already-shipped commands — Proposal Reads (056) to read the circulating proposal and its change set back, the `me` reads (Identity Read 011, My Roles 012, My Actions 013, My Projects 014) to draw the operator's current governance footprint, Governance Navigation (064) / Role Reads (025) to read a specifically-affected role back when the change touches one the operator fills, and Response Recording (058) for the `respond` write — and adds no command, flag, or governance logic of its own. It performs **one gated governance write**: recording a consent response with `proposal respond <prp-id> --response <value>` is exactly the boundary the Write-Safety Guardrail (063) exists to guard, so the path surfaces the proposal and the operator's chosen response for confirmation and runs the write through the guardrail's confirmed flow rather than smuggling a gated write ungated; every review read stays ungated. It stops before **advancing or withdrawing** the proposal (that is the proposer's Proposal Circulation Path, 068), before **drafting or creating** it (067), and before **judging whether the change is within the proposer's authority** (065) — the responder's impact review is the operator assessing the change against *their own* work, a distinct act from the proposer-side authority test. The exact delivery form — a plugin skill, a proposal-impact-review agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a proposal is circulating for consent, the path takes its `prp_` id as the starting point — the id of a proposal the operator has been asked to respond to, identified from the circle's proposals in flight or handed to the operator directly.
- The path requires an **already-circulating** proposal: it does not create, draft, advance, or withdraw the proposal — those are the Proposal Drafting Path (067) and the Proposal Circulation Path (068). It reads the proposal back through the existing read command to ground the review, but the proposal must already be on the record and open for response.
- When the operator is deciding across several proposals awaiting their response, the path may surface the circle's proposals in flight — narrowing the proposal list by circle and status — so the operator sees what is pending on them; it then reviews one proposal at a time.

### Reviewing the proposal's impact

- When the target proposal's `prp_` id is known, the path reads it back — surfacing the **change set it carries** (the governance changes it would apply), its `response_summary`, its `response_deadline`, and its `available_transitions` — so the operator sees what would change and how long the consent window stays open.
- The path then draws the change against the operator's **own current governance**: it reads the operator's footprint through the `me` reads — the roles they fill (012), the actions they hold (013), the projects they carry (014), grounded in their identity (011) — so the operator can see whether and how the proposed change touches their work once it lands.
- When the change set names a role, domain, or policy the operator's footprint touches, the path may read that specifically-affected governance back (Governance Navigation 064 / Role Reads 025) to show what it is today beside what the proposal would make it — a current-versus-proposed picture for the parts that land on the operator.
- The path draws these together into one **impact picture**: what the proposal changes, and where those changes intersect the operator's roles, actions, and projects — including the honest "this does not touch your current governance" when the change set falls outside the operator's footprint, which is itself a load-bearing review result.
- The path reads to **inform**, never to compute a verdict or gate: it surfaces the change set beside the operator's footprint so the impact is visible, but it does not fabricate a ruling that the operator *must* object or *may* consent, and it does not decide the response on the operator's behalf — the operator judges the impact and chooses.
- The review is **valuable on its own**: even when the operator finds no objection, the drawn-together picture shows them what the new world looks like for their work — the path's value is not conditional on an objection being found.
- When a list the path walks (the proposals in flight, or an affected role's members) spans more than one page, it pages through the full result set before drawing its picture — never a silent single-page cap. The path defers to each composed command's own read behavior and reimplements no walking of its own.

### Recording a consent response

- When the operator has reviewed the impact and decided, the path surfaces the proposal and the chosen response for confirmation and — because recording a consent response is a governance write — runs `proposal respond <prp-id> --response <value>` through the Write-Safety Guardrail (063) confirmed write flow, with `<value>` one of `no_objection` or `bring_to_meeting`, rather than issuing the gated write directly.
- The responding person is the **token's own identity** — the path supplies no person; the server records the response under the caller.
- When recording succeeds, the path returns the recorded response exactly as the server returned it — its `prr_` id, the value recorded, and the **parent proposal's status at the time of response** — so the operator sees the result of their answer. It synthesizes no fields.
- When the recorded response triggered auto-acceptance, the parent proposal's status in the result reads `accepted` rather than `proposed_outside_meeting`; the path surfaces that status as the load-bearing signal that the response closed the consent window, without computing acceptance itself.
- When the response is rejected — a Premium-disabled refusal (`403`), an unknown or invisible proposal (`404`), or a response the server does not allow (`422`, e.g. a second response by the same person) — the path surfaces the failure by name and records nothing; it does not treat any non-2xx as success or fabricate a state the record does not contain.

### Crossing the guardrail

- The path performs exactly **one gated governance write** — recording the consent response — and always through the guardrail's confirmed flow; it neither treats that write as ungated nor performs any further governance write.
- When the response is not confirmed, no response is recorded — the path issues no `respond`.
- Every review read — the proposal read-back, the `me` footprint reads, the affected-role reads — is ungated; only the response recording crosses the guardrail.

### Handoff

- When the operator's review surfaces a concern, recording `bring_to_meeting` persists it on the proposal and blocks auto-acceptance; the path records the response and stops — withdrawing or re-advancing the proposal is the proposer's Proposal Circulation Path (068), not this path.
- When the operator's review surfaces no concern, recording `no_objection` moves the proposal toward auto-acceptance; the operator may also complete the review without responding yet — the review stands on its own and the response is its optional culmination.
- When the path returns a result, the recorded response carries its `prr_` id and names the parent proposal's `prp_` id, so the review bridges back into the CLI's own commands rather than being a dead end.

### Staying within the operator layer

- The path only composes proposal, `me`, and role commands the CLI already exposes; it invents no command, flag, or capability, and reimplements no governance, permission, or response-authorization logic locally.
- When the CLI's proposal, `me`, or role command surface changes, the path is expected to stay consistent with it; guidance that names a command the CLI no longer offers is a defect, not a difference of opinion.

---

## User Scenarios

**In order to** understand what a circulating proposal would change for my own roles and work before I answer it,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** follow a guided path that reads the proposal's change set and draws it against my current governance.

**In order to** decide whether a proposal creates a tension for my work,
**as a** practitioner (served through the agent),
**I want** the path to show me where the change intersects the roles, actions, and projects I hold — rather than my cross-reading raw command output myself.

**In order to** answer a proposal once I have reviewed its impact without hand-assembling the response command,
**as an** AI agent operating the write flow,
**I want** the path to record my `no_objection` or `bring_to_meeting` through the confirmed write flow and return the recorded response.

---

## Non-Behaviors

- The path must not advance a proposal into circulation or withdraw it back to draft. **Why**: those transitions are the proposer's Proposal Circulation Path (068); this path is the responder acting on a proposal already circulating, and folding the proposer's circulation management in would claim work a sibling owns.
- The path must not draft, create, or assemble the change set of a proposal. **Why**: creation and drafting are the Proposal Drafting Path (067); this path begins from a proposal already on the record and open for response, and only reads its change set back to ground the review.
- The path must not record a consent response as an ungated write. **Why**: `proposal respond` is a governance write at exactly the boundary the Write-Safety Guardrail (063) guards; issuing it without the confirmed write flow would smuggle a gated write past the guardrail.
- The path must not decide the operator's response for them, nor manufacture a verdict that an objection is or is not required. **Why**: the path draws the impact so the operator can judge; ruling on the response would fork the consent judgment the operator owns and turn a review into an unrequested decision.
- The path must not judge whether the proposed change is within the proposer's authority. **Why**: that evaluative, proposer-side read is the Constraint Discovery Path (065); the responder's self-impact review is a distinct act, and ruling on authority here would duplicate the judgment in two places.
- The path must not compute acceptance, re-describe, or act on the server-owned consequences the response triggers (the auto-acceptance an all-`no_objection` set produces, the block a `bring_to_meeting` persists). **Why**: those are server-owned outcomes reflected in the returned response and the parent proposal's status; the CLI is a faithful surface, not a Holacracy facilitator (VISION Exclusion 1).
- The path must not add any command, flag, or API capability of its own, nor reimplement governance, permission, or response-authorization logic locally. **Why**: it is a guided composition of existing commands (knowledge + guardrails, never capability); the API is the source of truth (VISION Exclusion 2), and local logic drifts from the record the CLI faithfully surfaces.
- The path must not teach or coach Holacracy practice — how to weigh an objection, whether a tension is valid — nor dump raw, unsynthesized command output as its result. **Why**: it reviews the proposal against the record, it does not facilitate the governance craft (VISION Exclusion 1); and raw dumps are the rediscovery burden the operating surface exists to remove — the value is the drawn-together impact picture.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing commands — Proposal Reads (056) for the circulating proposal and its change set, the `me` reads (011/012/013/014) for the operator's footprint, Governance Navigation (064) / Role Reads (025) for affected-role read-backs, and Response Recording (058) for the `respond` write — and defers to the CLI's built-in help for their exact flags. If a command changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Write-Safety Guardrail (063)**: the guardrail gates the proposal write path, and this path crosses that boundary exactly once — recording the consent response. The path runs that write through the guardrail's confirmed write flow; every review read stays ungated.
- **Proposal Circulation Path (068)**: the proposer-side counterpart and upstream. 068 advances a proposal into circulation; this path is the responder acting on the circulating proposal. Advancing and withdrawing stay with 068.
- **Constraint Discovery Path (065)**: the proposer-side authority read. This path reviews a circulating proposal's impact on the responder; it does not judge whether the change is within the proposer's authority.
- **The `me` reads (011/012/013/014)**: the lens for personal impact. The path reads the operator's own roles, actions, and projects to draw the proposal's change against their current work; it defers to each command's own read behavior and does not reimplement it.
- **Glassfrog API**: never touched directly — the CLI mediates every command, and the API authorizes the response and enforces who may respond to a proposal. The proposal write surface is Premium-gated: a `403` means async proposals are not enabled. The path only ever does what the caller is permitted to do.

---

## Driving Scenarios

### Happy path

**Scenario: Review a circulating proposal's impact on the operator's roles**
Given the `prp_` id of a proposal circulating for consent
When the path reads the proposal's change set back and draws it against the operator's `me` roles, actions, and projects
Then it presents a drawn-together impact picture showing which of the operator's roles the change touches and how
And it records no response — the review stands on its own.

**Scenario: Record no-objection after review finds no concern**
Given a reviewed proposal whose change does not create a tension for the operator's work
When the path surfaces the proposal and `no_objection` for confirmation and runs `respond` through the confirmed write flow
Then it returns the recorded response with its `prr_` id and the parent proposal's status at the time of response
And when that response completed the expected set, the returned parent status reads `accepted`.

**Scenario: Record bring-to-meeting when review surfaces a concern**
Given a reviewed proposal whose change lands on a role the operator fills in a way they want discussed
When the path surfaces the proposal and `bring_to_meeting` for confirmation and runs `respond` through the confirmed write flow
Then it returns the recorded response, which persists on the proposal and blocks auto-acceptance
And the path stops — advancing or withdrawing the proposal is the Proposal Circulation Path (068).

### Error scenarios

**Scenario: A response is rejected**
Given a `respond` whose proposal is unknown (`404`), whose response the server does not allow (`422`, e.g. the operator already responded), or whose organization has async proposals disabled (`403`)
When the path runs the response through the confirmed write flow
Then it surfaces the API failure by name and records nothing
And it does not treat any non-2xx as success or fabricate a state the record does not contain.

**Scenario: A review read fails mid-picture**
Given a review where the proposal read, a `me` read, or an affected-role read fails
When the path continues
Then it surfaces what the failure was and presents what it gathered so far, flagged incomplete
And it does not invent the missing data or abandon the whole review.

### Edge cases

**Scenario: The response must be confirmed before it crosses the boundary**
Given a proposal the operator is ready to answer
When the path reaches the `respond` write
Then it routes the write through the Write-Safety Guardrail (063) confirmed flow, surfacing the proposal and the chosen response first
And if the write is not confirmed, no response is recorded.

**Scenario: The change does not touch the operator's governance**
Given a circulating proposal whose change set falls outside the operator's roles, actions, and projects
When the path draws the impact picture
Then it reports that the change does not touch the operator's current governance as a load-bearing review result
And it still shows what the proposal would change overall.

**Scenario: The review is complete without a response**
Given an operator who has reviewed a proposal's impact but is not ready to answer
When the path finishes the review
Then it presents the impact picture and records no response
And the review is a useful result on its own.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced proposal-impact-review-path content
When every command it composes is checked against the shipped CLI
Then each one exists — the path invents no command the CLI does not expose.

**Scenario: The single gated response is routed through the guardrail**
Given the path's treatment of the `respond` write
When it is inspected against the Write-Safety Guardrail (063)
Then it runs through the confirmed write flow — the path does not record the response as if it were an ungated write, and it performs no other governance write.

**Scenario: Reviews inform, never decide**
Given the path's use of the change set and the operator's footprint
When it is inspected for a fabricated objection verdict or an auto-chosen response
Then it draws the impact together for the operator to judge but leaves the response choice to the operator — it neither rules that an objection is required nor answers on the operator's behalf.

**Scenario: Review-and-respond only, no circulation transitions**
Given the path content
When it is inspected for any `propose`, `withdraw`, or `create` step
Then none is present — advancing and withdrawing are the Proposal Circulation Path (068) and creation is the Proposal Drafting Path (067).

**Scenario: Reviewing, not judging authority or coaching**
Given the path content
When it is inspected for a proposer-side authority verdict or Holacracy coaching
Then it neither rules on whether the change is within the proposer's authority (that is 065) nor advises on how to weigh an objection.

**Scenario: Synthesized, not raw**
Given the path's review result
When it is inspected against raw command output
Then it is a drawn-together impact picture relating the change set to the operator's footprint, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a proposal-impact-review agent, or both is decided during shaping. (The developer confirmed the form is deferred, mirroring how 062, 064, 065, 066, 067, and 068 deferred their decomposition.)
- **[ASSUMED] Impact review reaches current governance, centered on the operator**: the review reads the operator's own current governance — the `me` reads (011/012/013/014), extended to a specifically-affected role via 064/025 when the change touches one the operator fills — to show what the operator's world becomes if the proposal lands. (The developer confirmed the review must check current governance, especially the `me` paths.)
- **[ASSUMED] Review is more than an objection test**: the path's value is showing the operator what the new world looks like for their work; it is useful even when no objection surfaces, and it does not decide the response for the operator. (The developer confirmed the review stands on its own beyond finding objections.)
- **[ASSUMED] Entry is an already-circulating proposal id**: the path starts from a `prp_` id already circulating for consent and reads it back; it does not create, draft, advance, or withdraw the proposal. (The developer confirmed the responder acts on a proposal on the record.)
- **[ASSUMED] The response write crosses the guardrail**: recording the consent response runs through the Write-Safety Guardrail (063) confirmed write flow; every review read is ungated. (The developer confirmed the one guardrail crossing.)
- **Composed commands are already shipped** (technical): Proposal Reads (056), the `me` reads (011/012/013/014), Governance Navigation (064) / Role Reads (025), and Response Recording (058) all exist in the CLI today, so the path composes them rather than waiting on new commands. (Grounded in the shipped proposal, identity, and role command families.)
- **The response is server-authorized and Premium-gated** (technical): `respond` returns `403` when async proposals are disabled, `404` for an unknown proposal, and `422` for a response the server does not allow (e.g. a second response by the same person), so the path issues the write and lets the server decide. (Grounded in 058's Behavioral Accord.)

---

## Ambiguity Warnings

None remaining — the review depth (current governance, centered on the operator's `me` footprint), the review-is-more-than-objection framing, the single guardrail crossing, the entry point (an already-circulating proposal), and the scope boundaries (review-and-respond in; advancing/withdrawing to 068, drafting to 067, authority judgment to 065) were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-19

- **Review depth**: The review checks current governance, not just the proposal in isolation — centered on the operator's own footprint via the `me` reads (roles/actions/projects), extended to a specifically-affected role via Governance Navigation (064) / Role Reads (025) when the change touches one the operator fills. The goal is to make clear what will change once the proposal lands and how it lands on the operator's work. The developer confirmed current governance — especially the `me` paths — must be checked.
- **Review is more than an objection test**: The path shows the operator what the new world looks like for their work and is useful even when no objection is found; it does not decide the response. The developer confirmed the review stands on its own beyond the objection test.
- **One guardrail crossing**: Recording the consent response (`proposal respond`, 058) runs through the Write-Safety Guardrail (063) confirmed write flow; every review read stays ungated. The developer confirmed the crossing.
- **Scope boundary**: Review-and-respond is this path (the responder's side). Advancing and withdrawing are the proposer's Proposal Circulation Path (068), drafting and creation are the Proposal Drafting Path (067), and authority judgment is the Constraint Discovery Path (065). The developer confirmed the responder-side scope.
