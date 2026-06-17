# Specification: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Write-Safety Guardrail is the **governance-integrity gate of the Agent Operating Surface** — the enforcing counterpart to the write-safety *guidance* that Operator Orientation (062) only describes. Today an AI agent driving the CLI can run a command that changes governance through the proposal write path (create or advance a proposal, record a response, withdraw a circulating proposal) directly, with nothing standing between the agent's decision and the write. VISION principle 2 says a change to the governance record must be a *deliberate, explicit choice* — "never something an agent reaches without asking." This capability makes that real: it gates every governance write behind an explicit human confirmation from the practitioner, and when a write is refused as stale it re-reads and re-confirms rather than blindly retrying.

It rides on the operator layer, not the CLI binary: like its sibling capabilities it arrives as an additional capability in the plugin Operator Orientation (062) defines, and it adds **no API capability of its own** — it sequences and gates invocations of commands the CLI already exposes. It builds on two things: Operator Orientation (062) as the plugin root that already states the write-safety expectation as guidance, and Stale-Write Surfacing (054) as the distinct `412` signal (its own failure category and exit code) the guardrail reacts to on a refused write. It deliberately stops at gating: it does **not** add or change any command, does **not** re-validate the change locally (the API stays the source of truth), and does **not** judge whether the change is governance-sound (that is coaching, VISION Exclusion 1).

---

## Behavioral Accord

### Recognizing a governance write

- When the agent is about to invoke a command on the proposal write path — creating a proposal, advancing it into circulation, withdrawing it, or recording a response — the guardrail treats it as a governance write that requires confirmation before it runs.
- When the agent invokes a read-only command, or an operational tension edit (capturing, updating, or discarding a tension), the guardrail does not gate it — those flow through unguarded.

### Confirming before writing

- When the agent is about to perform a governance write, it first surfaces what it is about to do — the command, the target proposal, and the change it carries — to the practitioner in terms they can review before anything is sent.
- When the practitioner has not given explicit, deliberate confirmation for that specific write, the agent does not execute it — a governance write is never reached without the practitioner's explicit go-ahead, and the agent cannot self-authorize one.
- When the practitioner confirms, the agent performs exactly the write that was confirmed — it does not broaden the change, substitute a different target, or bundle additional writes into the confirmed action.

### Handling a stale-write refusal

- When a confirmed write is refused as a stale write — the distinct stale-write outcome that Stale-Write Surfacing (054) surfaces, recognized by its own failure category and exit code — the guardrail does not retry with the version it already held.
- Instead it re-reads the resource to obtain its current version and surfaces the now-current state, so the practitioner sees what changed underneath before deciding.
- When the practitioner re-confirms against the current state, the agent retries the write with the refreshed version; without re-confirmation, it does not retry.

### Staying within the operator layer

- The guardrail adds no command, flag, or capability to the CLI — it only sequences and gates invocations of commands the CLI already exposes.
- The guardrail does not re-validate or reimplement the change locally — the API remains the source of truth for whether a change is valid; confirmation establishes operator *intent*, not correctness.

---

## User Scenarios

**In order to** keep a deliberate human decision in the loop on every change to the governance record,
**as a** practitioner whose AI agent acts on my behalf,
**I want to** personally confirm each proposal-path write before the agent sends it, so the agent never alters governance without my explicit go-ahead.

**In order to** avoid clobbering a concurrent change when a write is refused as stale,
**as a** practitioner,
**I want to** have the agent re-read and show me the current state and ask again before retrying, rather than blindly re-sending the stale write.

**In order to** keep reads and operational tension edits fast and frictionless,
**as an** AI agent,
**I want to** have read-only commands and tension edits pass through ungated, so only governance writes through the proposal path carry the confirmation cost.

---

## Non-Behaviors

- The guardrail must not gate read-only commands. **Why**: reads change nothing, so gating them adds friction with no integrity benefit and slows the agent's primary mode (legible reads), for no protection in return.
- The guardrail must not gate operational tension edits — capturing, updating, or discarding a tension. **Why**: a tension is an *operational* item, not a governance change; PROJECT.md's domain splits operational items (editable directly) from governance structure (changed only through proposals), and VISION principle 2's integrity concern targets governance changes — the proposal path. Gating tension edits would add confirmation friction to work that does not alter the governance record.
- The guardrail must not add, remove, or modify any CLI command, flag, or capability. **Why**: it is knowledge + guardrail at the operator layer; adding capability breaks "Bounded by the API surface" (PROJECT constraint) and VISION Exclusion 2, turning a guardrail into a second, drifting surface.
- The guardrail must not re-validate or reimplement governance/validation logic locally. **Why**: the API is the source of truth (VISION Exclusion 2); local checks inevitably drift from the spec. Confirmation is about operator intent, not re-deciding correctness.
- The guardrail must not blindly retry a stale-write refusal. **Why**: re-sending the stale version either fails again or clobbers the very concurrent change the `412` exists to protect, defeating the Optimistic Concurrency the surfacing was built for.
- The guardrail must not auto-confirm or let the agent self-authorize a governance write. **Why**: VISION principle 2 — a governance change must be a deliberate, explicit choice, never something an agent reaches without asking; silent self-confirmation removes the human deliberation the guardrail exists to guarantee.
- The guardrail must not coach Holacracy or judge the governance merits of the change. **Why**: VISION Exclusion 1 — the surface orients on driving the CLI safely, not facilitating governance; assessing whether a change is *wise* is coaching, a different surface's concern.
- This spec must not define how the guardrail is distributed or packaged. **Why**: distribution is *Operating-Surface Packaging* (#70) and the plugin it lives in is defined by Operator Orientation (062); defining delivery here couples the guardrail to a mechanism that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The guardrail sequences invocations of the CLI's existing write and read commands and observes their outcomes (including exit codes); it invokes nothing beyond what the CLI exposes.
- **Operator Orientation (062) / the Claude plugin**: the guardrail ships as an additional capability in the same plugin. Orientation *describes* the write-safety expectation as guidance; this capability *enforces* it. If the guardrail is absent, writes fall back to the unguarded, orientation-only behavior.
- **Stale-Write Surfacing (054)**: supplies the distinct stale-write signal (its own failure category and exit code) the guardrail reacts to on a refused write. A refusal not classified as stale-write is left to the normal failure paths.
- **Practitioner (human in the loop)**: the actor whose explicit confirmation gates each governance write and whose re-confirmation is required after a stale-write re-read.

---

## Driving Scenarios

### Happy path

**Scenario: Confirming a proposal write before it runs**
Given an agent about to advance a proposal into circulation on the practitioner's behalf
When the guardrail intercepts the write
Then it surfaces the command, the target proposal, and the change to be made
And the write is not sent until the practitioner explicitly confirms it.

**Scenario: A read passes through ungated**
Given an agent about to run a read-only command (for example, listing the practitioner's roles)
When the guardrail evaluates the command
Then it does not require confirmation
And the read proceeds immediately.

**Scenario: Performing exactly the confirmed write**
Given a proposal write the practitioner has explicitly confirmed
When the agent executes it
Then it sends exactly the confirmed change to the confirmed proposal
And it does not broaden, substitute, or bundle any additional write into the action.

### Error scenarios

**Scenario: Confirmation withheld**
Given an agent about to create a proposal
When the practitioner does not give explicit confirmation
Then the write is not executed
And the governance record is left unchanged.

**Scenario: Stale-write refusal triggers re-read and re-confirm**
Given a confirmed write that the server refuses as a stale write (the distinct stale-write outcome surfaced by 054)
When the guardrail handles the refusal
Then it does not retry with the version it already held
And it re-reads the resource for its current version, surfaces the now-current state, and asks the practitioner to re-confirm before any retry.

### Edge cases

**Scenario: Re-confirmation withheld after a stale-write re-read**
Given a stale-write refusal that has prompted a re-read of the current state
When the practitioner does not re-confirm against that current state
Then the agent does not retry the write
And the resource is left as the concurrent change last set it.

**Scenario: An operational tension edit passes through ungated**
Given an agent about to capture a tension (an operational edit, not a governance change)
When the guardrail evaluates the command
Then it does not require confirmation
And the tension is captured immediately.

**Scenario: A non-stale-write failure is not treated as a clobber**
Given a confirmed write that fails with an outcome other than the stale-write category (for example, a permission or rate-limit failure)
When the guardrail observes the outcome
Then it does not invoke the re-read/re-confirm recovery
And the failure flows through the CLI's normal failure handling unchanged.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No ungated governance write**
Given the produced guardrail
When every proposal-path write it covers is traced
Then each is reachable only after explicit human confirmation — none has a path that writes without asking.

**Scenario: Operational edits are not gated**
Given the guardrail's command coverage
When the tension edits (capture, update, discard) are traced
Then none of them requires confirmation — the gate covers the proposal path only.

**Scenario: No invented surface**
Given the guardrail's gated command set
When every command it names is checked against the shipped CLI
Then each one exists — the guardrail invents no command, flag, or capability the CLI does not already expose.

**Scenario: No blind retry on a clobber**
Given the guardrail's stale-write path
When it is inspected for a retry
Then no retry occurs without an interposed re-read and a fresh confirmation.

**Scenario: Guardrail, not coach**
Given the confirmation the guardrail surfaces
When its content is inspected
Then it states what will change (command, target, change) and nowhere advises whether the change is governance-sound.

---

## Assumptions

- **Delivered as an added capability in the existing plugin**: the guardrail is an additional capability in the plugin Operator Orientation (062) defines. The delivery mechanism is a shaping decision — the plan resolves it to an operator-layer `PreToolUse` hook (plan ADR-1), not new skill content. (Structural; this spec stays mechanism-agnostic.)
- **Scope narrowed from FEATURE-MODEL**: FEATURE-MODEL enumerates the gated set as "tension capture, proposals, responses"; this spec deliberately narrows it to the proposal write path only (create, advance, withdraw, record response), treating tension capture/update/discard as ungated operational edits. (Resolved during clarification — grounded in PROJECT.md's operational/governance split; FEATURE-MODEL may warrant a reconciling update.)
- **Stale-write signal reuse** (technical): the guardrail keys off the distinct stale-write category and exit code from Stale-Write Surfacing (054) rather than re-classifying `412` itself, mirroring 054's own decoupling from the write command. (Keeps the guardrail independent of the classification mechanism.)

---

## Ambiguity Warnings

None remaining — both ambiguities raised during specification (the confirmation model and the gated command membership) were resolved during clarification. See Clarifications.

---

## Clarifications

### Session 2026-06-16

- **Confirmation model**: Each governance write is gated behind a mandatory human-in-the-loop checkpoint — the practitioner explicitly confirms every proposal-path write before the agent sends it, and the agent cannot self-authorize one. (Anchored in VISION principle 2: "never something an agent reaches without asking.")
- **Gated command membership**: The guardrail gates only governance writes through the proposal write path — creating a proposal, advancing it into circulation, withdrawing it, and recording a response. Operational tension edits (capture, update, discard) are left ungated, consistent with PROJECT.md's split between operational items (editable directly) and governance structure (changed only through proposals). This narrows FEATURE-MODEL's broader "tension capture, proposals, responses" enumeration; FEATURE-MODEL may warrant a reconciling update.
