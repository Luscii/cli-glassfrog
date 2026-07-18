# Specification: Tension Processing Path

**Feature**: 066-tension-processing-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Tension Processing Path is the second **operator path** on the Agent Operating Surface — a guided composition that rides on the root *Operator Orientation* (062) established, alongside the read-only *Governance Navigation Path* (064). Where 064 reads the governance record to *understand* the concern around a tension, this path *works the tension itself* through its operational lifecycle: it captures a voiced tension onto the sensing role, situates it against what is already sensed so a duplicate is a deliberate choice rather than a blind add, refines or retires it, and — when the tension is ready to become a governance change — hands its `ten_` id onward to the Proposal Drafting Path (067). It turns a felt gap into a well-formed record the write path can consume, without the agent hand-assembling the tension commands and untangling raw output.

The path is knowledge, not capability. It composes already-shipped operational commands — Tension Capture (042), Tension Reads (043), Subroles Tension Roll-up (046), Tension Update (044), and Tension Discard (045) — and adds no command, flag, or governance logic of its own. Its writes are **operational tension edits**, which the Write-Safety Guardrail (063) deliberately leaves **ungated**; the guardrail engages only when work crosses into a *proposal*, which is the territory of Proposal Drafting (067) and Proposal Circulation (068), not this path. It does not judge whether a tension needs a proposal or whether an action is within authority — that evaluative read is the Constraint Discovery Path (065). The exact delivery form — a plugin skill, a tension-processing agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a practitioner voices a tension they have decided to act on, the path takes that voiced concern as its starting point — no pre-existing `ten_` id is required, because the path captures the tension itself.
- When a synthesized picture from the Governance Navigation Path (064) is already in hand, the path may consume it to identify the sensing role, but it does not require 064 to have run first.

### Situating

- When the sensing role is known, the path can surface the tensions that role already senses — and, where relevant, roll up the tensions its direct sub-roles sense — so a new capture is a deliberate addition rather than a blind duplicate.
- When the tensions already sensed span more than one page, the path pages through the full result set before judging whether the voiced tension is a duplicate — the duplicate check is made over the complete set, never a silent single-page cap.
- When the voiced tension matches one already on the record, the path surfaces that existing tension with its id rather than silently recording a second one.

### Capture and refine

- When the tension is ready to record, the path captures it against the sensing role through the CLI's existing capture command, and returns the created tension including its `ten_` id.
- When a captured tension needs correcting — a clearer body, a better label, a different routing, or a move through its lifecycle — the path refines it through the existing update command rather than recapturing.
- When a tension is no longer worth pursuing, the path retires it through the existing discard command rather than leaving a stale entry on the active record.

### Handoff

- When a tension is ready to become a governance change, the path hands its `ten_` id to the Proposal Drafting Path (067); it does not draft, create, or circulate the proposal itself.
- When the path returns a result, each tension it surfaces carries the id needed to read it again, refine it, or feed it to the drafting path — so processing bridges back into the CLI's own commands rather than being a dead end.

### Staying within the operator layer

- The path only composes operational tension commands the CLI already exposes; it invents no command, flag, or capability, and reimplements no governance or permission logic locally.
- When the CLI's tension command surface changes, the path is expected to stay consistent with it; guidance that names a command the CLI no longer offers is a defect, not a difference of opinion.
- The path performs only operational tension writes, which flow ungated; it neither gates those writes behind the governance-write confirmation nor performs any proposal write that would require it — the Write-Safety Guardrail (063) governs the proposal boundary, which this path does not cross.

---

## User Scenarios

**In order to** turn a gap I've noticed into a well-formed record without hand-assembling the tension commands,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** follow a guided path from the voiced tension to a captured, refined tension carrying its id.

**In order to** avoid cluttering the record with a tension that is already sensed,
**as a** practitioner (served through the agent),
**I want** the path to show me what's already sensed on the role and its sub-roles before it captures a new one.

**In order to** move from a captured tension into a governance change without losing my place,
**as an** AI agent assembling the write flow,
**I want** the processed tension to carry the `ten_` id that feeds the Proposal Drafting Path.

---

## Non-Behaviors

- The path must not create, advance, withdraw, or circulate a proposal, nor attach a tension to one. **Why**: those are governance writes owned by Proposal Drafting (067) and Proposal Circulation (068) and gated by the Write-Safety Guardrail (063); folding them in here would fork the staged write path and smuggle a gated write into an ungated path.
- The path must not judge whether a tension needs a proposal, or whether a wanted action is within the practitioner's authority. **Why**: that evaluative read is the Constraint Discovery Path (065); ruling on it here would duplicate the judgment in two places.
- The path must not add any command, flag, or API capability of its own. **Why**: it is a guided composition of existing operational commands (knowledge + guardrails, never capability); inventing a surface would break "Bounded by the API surface" and VISION Exclusion 2.
- The path must not reimplement or duplicate governance, permission, or validation logic locally. **Why**: the API is the source of truth (VISION Exclusion 2); local logic drifts from the record the CLI faithfully surfaces and second-guesses who may sense or edit a tension.
- The path must not teach or coach Holacracy practice — interpreting the tension or advising whether it is well-formed governance craft. **Why**: it works the tension through the record, it does not facilitate the sensing (VISION Exclusion 1).
- The path must not dump raw, unsynthesized command output as its result. **Why**: raw dumps are exactly the rediscovery burden the operating surface exists to remove; the value is the drawn-together tension record, not a concatenation of command dumps.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing operational tension commands (capture, list, get, subroles roll-up, update, discard) and defers to the CLI's built-in help for their exact flags. If a command changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Write-Safety Guardrail (063)**: the guardrail gates the proposal write path, not operational tension edits. This path stays wholly on the ungated operational side; the boundary it must not cross is the point where a tension becomes a proposal.
- **Proposal Drafting Path (067)**: the downstream handoff. When a tension is ready to become a governance change, this path hands the `ten_` id to 067; it does not draft the proposal.
- **Glassfrog API**: never touched directly — the CLI mediates every command, and the API enforces who may sense or edit a tension. The path only ever does what the caller is permitted to do.

---

## Driving Scenarios

### Happy path

**Scenario: From a voiced tension to a captured record**
Given a practitioner's voiced tension and the sensing role it belongs to
When the path captures the tension against that role
Then it returns the created tension including its `ten_` id
And carries that id so the tension can be refined or fed onward.

**Scenario: Situating a tension against what's already sensed**
Given a sensing role the path is about to capture a tension on
When the path surfaces the tensions the role already senses and rolls up its direct sub-roles' tensions
Then it presents them drawn together so the practitioner can see what's already on the record
And treats the new capture as a deliberate addition rather than a blind duplicate.

**Scenario: Refining a captured tension**
Given a tension already on the record that needs a clearer body or better routing
When the path refines it through the update command
Then it returns the updated tension
And does not recapture it as a second entry.

### Error scenarios

**Scenario: A capture is rejected**
Given a capture attempt with an unknown sensing role or a blank body
When the path runs the capture command
Then it surfaces the usage or API failure by name and records nothing
And it does not fabricate a `ten_` id the record does not contain.

**Scenario: A situating read fails**
Given a situating step where one read fails (for example, the sub-role roll-up errors)
When the path continues
Then it surfaces what the failure was and presents the tensions the reads that succeeded returned
And it does not invent the missing tensions or abandon the whole result.

### Edge cases

**Scenario: The tension is already sensed**
Given a voiced tension that matches one already on the record
When the path situates it before capturing
Then it surfaces the existing tension with its id and lets the practitioner refine that one
And it does not silently record a duplicate.

**Scenario: The tension is ready to become a governance change**
Given a captured, well-formed tension the practitioner wants to turn into a proposal
When the path completes its processing
Then it hands the `ten_` id to the Proposal Drafting Path (067)
And it does not draft, create, or circulate the proposal itself.

**Scenario: The tension is no longer worth pursuing**
Given a tension the practitioner decides is moot
When the path processes that decision
Then it retires the tension through the discard command
And it does not push the tension toward a proposal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced tension-processing-path content
When every command it composes is checked against the shipped CLI
Then each one exists — the path invents no command the CLI does not expose.

**Scenario: Operational writes only, no governance write**
Given the path content
When it is inspected for any proposal create, advance, withdraw, or response step
Then none is present — the path performs only operational tension edits and hands the proposal work to 067.

**Scenario: Correct guardrail boundary**
Given the path's treatment of its writes
When it is inspected against the Write-Safety Guardrail (063)
Then it neither gates the operational tension edits behind the governance-write confirmation nor performs a governance write that would require it.

**Scenario: Processing, not judging or coaching**
Given the path content
When it is inspected for an authority verdict or Holacracy coaching
Then it neither rules on whether a tension needs a proposal (that is 065) nor advises on governance craft.

**Scenario: Synthesized, not raw**
Given the path's result
When it is inspected against raw command output
Then it is a drawn-together tension record carrying ids, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a tension-processing agent, or both is decided during shaping. (The developer confirmed the form is deferred, mirroring how 062 and 064 deferred their decomposition.)
- **[ASSUMED] Entry is a voiced tension**: the path starts from a tension the practitioner has decided to act on and captures it, producing the `ten_` id; it does not require an already-captured id, and it may consume a Governance Navigation Path (064) picture but does not require one. (The developer agreed with this entry model.)
- **[ASSUMED] Scope is the tension operational lifecycle only**: capture, situate, refine, and retire a tension, then hand off. Proposal drafting is the Proposal Drafting Path (067), circulation is the Proposal Circulation Path (068), and authority judgment is the Constraint Discovery Path (065). (The developer set these boundaries explicitly.)
- **Composed commands are already shipped** (technical): Tension Capture (042), Tension Reads (043), Subroles Tension Roll-up (046), Tension Update (044), and Tension Discard (045) all exist in the CLI today, so the path composes them rather than waiting on new commands. (Grounded in the shipped tension command family.)
- **Operational tension edits are ungated** (technical): the path's writes flow without the governance-write confirmation, consistent with the Write-Safety Guardrail (063), which explicitly carves capturing, updating, and discarding a tension out of the gated proposal write path. (Grounded in 063's Behavioral Accord.)

---

## Ambiguity Warnings

None remaining — the entry point, the scope boundaries with 065 / 067 / 068, and the ungated-operational-side boundary with the Write-Safety Guardrail (063) were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-18

- **Entry point**: The path starts from a voiced tension the practitioner has decided to act on, captures it, and returns the `ten_` id. It does not require a pre-existing tension id and may consume a 064 picture but does not require one.
- **Scope boundary**: Processing covers the tension operational lifecycle only — capture, situate, refine, retire, and handoff. Proposal drafting is 067; proposal circulation is 068. The path stops at handing the ready `ten_` id to 067.
- **Authority judgment stays out**: Whether a tension needs a proposal, or whether a wanted action is within authority, is the Constraint Discovery Path (065). This path processes the tension; it does not rule on it.
- **Guardrail boundary**: The path's writes are operational tension edits, which the Write-Safety Guardrail (063) leaves ungated; the guardrail engages only when work crosses into a proposal (067/068), which this path does not do.
