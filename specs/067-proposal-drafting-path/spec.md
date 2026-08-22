# Specification: Proposal Drafting Path

**Feature**: 067-proposal-drafting-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Proposal Drafting Path is the **fourth operator path** on the Agent Operating Surface, and the first one to cross into a *gated* governance write. It rides on the root *Operator Orientation* (062), alongside the read-only *Governance Navigation Path* (064), the evaluative *Constraint Discovery Path* (065), and the ungated *Tension Processing Path* (066). Where 066 works a tension through its operational lifecycle and stops at the moment that tension is ready to become a governance change, this path picks up that `ten_` id and carries it the rest of the way: it situates the anchor tension against the proposals already in flight in the relevant circle, helps assemble the set of **governance changes**, and creates the **draft proposal** through the CLI's existing write command — turning a well-formed tension into a submittable draft the circulation path can advance.

The path is knowledge, not capability. It composes already-shipped commands — Proposal Creation (055) for the `proposal create` write, Proposal Reads (056) for situating against in-flight proposals, and Tension Reads (043) to read the anchor tension back — and adds no command, flag, or governance logic of its own. Its write is a **gated proposal write**: unlike 066's operational tension edits, `proposal create` is exactly the boundary the Write-Safety Guardrail (063) exists to guard, so the path surfaces the assembled anchor and change set for confirmation and runs the create through the guardrail's confirmed write flow rather than smuggling a gated write ungated. It stops at the `draft` — it does not advance the proposal to circulation (068), record responses (069), or judge whether the tension needs a proposal at all (065). The exact delivery form — a plugin skill, a proposal-drafting agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a tension is ready to become a governance change, the path takes its `ten_` id as the starting point — the id handed off by the Tension Processing Path (066), or one the practitioner identifies from an existing tension.
- The path requires an already-existing tension: it does not capture, refine, or discard the tension itself — that is the Tension Processing Path (066). It reads the anchor tension back through the existing read command to ground the draft, but the tension must already be on the record.

> **Superseded in part — 2026-08-22, by 079-pre-assembly-grammar-consultation.** The entry widened: the path now takes the intended change with the anchor `ten_` id *optionally* in hand, and its first workflow step is a routing determination that names the target circle and every eligible anchor before an anchor is settled on — the anchor choice stays the practitioner's, and a settled anchor must still be an already-existing tension. In the same change the composed fence grew from seven leaves to eight (`proposal grammar` joined as the consultation read), the draft record grew from six elements to seven (the `consultation` element joined), and the `action` vocabulary grew from four values to seven (`surfaced-routing-mismatch`, `named-anchors`, `surfaced-dead-shape` joined). The text above stays legible as what was true before that change; 079's interface accord is the current statement of these contracts.

### Situating

- When the anchor tension's circle is known, the path surfaces the proposals already in flight there — narrowing the proposal list by circle and, where relevant, by `draft` status — so a new draft is a deliberate addition rather than a blind duplicate.
- When the proposals in flight span more than one page, the path pages through the full result set before judging whether a matching draft already exists — the duplicate check is made over the complete set, never a silent single-page cap.
- When a draft already in flight matches the change the practitioner intends, the path surfaces that existing proposal with its `prp_` id rather than silently creating a second one.
- The path situates by circle and status because that is what the proposal list exposes; it does not claim to filter proposals by the anchor tension, a narrowing the CLI's list does not offer.

### Assembling the change set

- When the change is ready to record, the path helps shape the `changes` as a JSON array of governance-command objects — each element an object carrying a non-empty `type` — sourced inline, from a file, or from piped standard input, mirroring how the create command resolves its `--changes` source.
- The path passes the assembled change set through **verbatim** above that `type` floor; it does not recognize individual change types or validate their command-specific keys, and it does not build a typed per-change constructor — that typed construction is the deferred *Unguided Change Construction* capability, and the server validates each change.

### Crossing the guardrail and creating the draft

- When the anchor tension and change set are assembled, the path surfaces them for confirmation and — because creating a proposal is a governance write — runs `proposal create <tension-id> --changes <source>` through the Write-Safety Guardrail (063) confirmed write flow, rather than issuing the gated write directly.
- When the create succeeds, the path returns the created proposal including its `prp_` id and its `draft` status, so a later step can reference it.
- When the create is rejected — a Premium-disabled refusal (`403`), an unknown anchor tension (`404`), or a rejected change set (`422`) — the path surfaces the failure by name and creates nothing; it does not fabricate a `prp_` id the record does not contain.

### Handoff

- When a draft proposal is ready to enter circulation, the path hands its `prp_` id to the Proposal Circulation Path (068); it does not advance, withdraw, respond to, or circulate the proposal itself.
- When the path returns a result, the draft it surfaces carries the `prp_` id needed to read it again, advance it, or withdraw it — so drafting bridges back into the CLI's own commands rather than being a dead end.

### Staying within the operator layer

- The path only composes proposal and tension commands the CLI already exposes; it invents no command, flag, or capability, and reimplements no governance, permission, or change-validation logic locally.
- When the CLI's proposal command surface changes, the path is expected to stay consistent with it; guidance that names a command the CLI no longer offers is a defect, not a difference of opinion.
- The path performs exactly one gated write — the proposal create — and always through the guardrail's confirmed flow; it neither treats that write as ungated nor performs any further governance write beyond the draft.

---

## User Scenarios

**In order to** turn a well-formed tension into a submittable draft proposal without hand-assembling the create command,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** follow a guided path from the anchor tension to a created draft proposal carrying its `prp_` id.

**In order to** avoid opening a second draft for a change already in flight,
**as a** practitioner (served through the agent),
**I want** the path to show me the proposals already circulating in the circle before it creates a new one.

**In order to** move a draft into circulation without losing my place,
**as an** AI agent assembling the write flow,
**I want** the created draft to carry the `prp_` id that feeds the Proposal Circulation Path (068).

---

## Non-Behaviors

- The path must not advance a draft to circulation, withdraw it, record a response, or otherwise transition a proposal. **Why**: those are the Proposal Circulation Path (068) and the response side (069); folding them in here would fork the staged write flow and claim work these siblings own.
- The path must not build a typed per-change constructor, nor validate a change's `type` value or its command-specific keys. **Why**: a change is a free-form governance command with no per-type schema, so the CLI passes the array through verbatim above a `type` floor and the server validates; typed construction is the deferred *Unguided Change Construction* capability, not this path.
- The path must not capture, refine, or discard the anchor tension. **Why**: the tension operational lifecycle is the Tension Processing Path (066); this path begins from a tension already on the record and only reads it back to ground the draft.
- The path must not perform the proposal create as an ungated write. **Why**: creating a proposal is exactly the boundary the Write-Safety Guardrail (063) guards; issuing it without the confirmed write flow would smuggle a gated write past the guardrail.
- The path must not judge whether the tension needs a proposal, or whether the intended change is within the practitioner's authority. **Why**: that evaluative read is the Constraint Discovery Path (065); ruling on it here would duplicate the judgment in two places.
- The path must not add any command, flag, or API capability of its own, nor reimplement governance, permission, or validation logic locally. **Why**: it is a guided composition of existing commands (knowledge + guardrails, never capability); the API is the source of truth (VISION Exclusion 2), and local logic drifts from the record the CLI faithfully surfaces.
- The path must not teach or coach Holacracy practice, nor dump raw, unsynthesized command output as its result. **Why**: it drafts the proposal through the record, it does not facilitate the governance craft (VISION Exclusion 1); and raw dumps are the rediscovery burden the operating surface exists to remove — the value is the drawn-together draft carrying its id.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing commands — Proposal Creation (055) for the `proposal create` write, Proposal Reads (056) for situating against in-flight proposals, and Tension Reads (043) to read the anchor tension — and defers to the CLI's built-in help for their exact flags. If a command changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Write-Safety Guardrail (063)**: the guardrail gates the proposal write path, and this path crosses that boundary exactly once — the proposal create. The path runs that create through the guardrail's confirmed write flow; it performs no other governance write.
- **Tension Processing Path (066)**: the upstream handoff. This path picks up the `ten_` id 066 produces when a tension is ready to become a governance change; it does not itself capture or process the tension.
- **Proposal Circulation Path (068)**: the downstream handoff. When a draft is ready to circulate, this path hands the `prp_` id to 068; it does not advance the proposal.
- **Glassfrog API**: never touched directly — the CLI mediates every command, and the API enforces who may draft a proposal and validates the change set. The proposal write surface is Premium-gated: a `403` means async proposals are not enabled. The path only ever does what the caller is permitted to do.

---

## Driving Scenarios

### Happy path

**Scenario: From a tension to a draft proposal**
Given a well-formed anchor tension's `ten_` id and an assembled set of governance changes
When the path surfaces them for confirmation and runs the create through the confirmed write flow
Then it returns the created proposal including its `prp_` id and `draft` status
And carries that id so the draft can be advanced or withdrawn.

**Scenario: Situating against proposals already in flight**
Given the circle the anchor tension belongs to
When the path surfaces the proposals already in flight there, paging through the full set
Then it presents them drawn together so the practitioner can see what is already circulating
And treats the new draft as a deliberate addition rather than a blind duplicate.

**Scenario: Sourcing the change set from a file**
Given a set of governance changes held in a JSON file
When the path assembles the change set from that source
Then it passes the array through verbatim above the `type` floor as the proposal's changes
And does not interpret or validate any change's command-specific keys.

### Error scenarios

**Scenario: A create is rejected**
Given a create attempt whose anchor tension is unknown, whose change set the server rejects, or whose organization has async proposals disabled
When the path runs the create through the confirmed write flow
Then it surfaces the API failure by name (for example `404`, `422`, or the `403` Premium refusal) and creates nothing
And it does not fabricate a `prp_` id the record does not contain.

**Scenario: A situating read fails**
Given a situating step where the proposal list read fails mid-walk
When the path continues
Then it surfaces what the failure was and presents the proposals the read gathered so far, flagged incomplete
And it does not invent the missing proposals or abandon the whole result.

### Edge cases

**Scenario: The create must be confirmed before it crosses the boundary**
Given an assembled anchor and change set ready to create
When the path reaches the proposal create
Then it routes the write through the Write-Safety Guardrail (063) confirmed flow, surfacing the change set first
And if the write is not confirmed, no proposal is created.

**Scenario: A matching draft is already in flight**
Given a change that matches a draft already circulating in the circle
When the path situates before creating
Then it surfaces the existing proposal with its `prp_` id and lets the practitioner decide
And it does not silently create a duplicate draft.

**Scenario: The draft is ready to circulate**
Given a created draft proposal the practitioner wants to advance
When the path completes its drafting
Then it hands the `prp_` id to the Proposal Circulation Path (068)
And it does not advance, withdraw, or circulate the proposal itself.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced proposal-drafting-path content
When every command it composes is checked against the shipped CLI
Then each one exists — the path invents no command the CLI does not expose.

**Scenario: The gated create is routed through the guardrail**
Given the path's treatment of the proposal create
When it is inspected against the Write-Safety Guardrail (063)
Then the create runs through the confirmed write flow — the path does not issue the gated proposal write as if it were ungated.

**Scenario: Assembly, not typed construction**
Given the path's handling of the change set
When it is inspected for per-change interpretation
Then it assembles and passes the array through verbatim above a `type` floor — it validates no change's `type` value or command-specific keys and builds no typed constructor.

**Scenario: Drafting only, no further transition**
Given the path content
When it is inspected for any advance, withdraw, response, or circulate step
Then none is present — the path stops at the created `draft` and hands the `prp_` id to 068.

**Scenario: Drafting, not judging or coaching**
Given the path content
When it is inspected for an authority verdict or Holacracy coaching
Then it neither rules on whether the tension needs a proposal (that is 065) nor advises on governance craft.

**Scenario: Synthesized, not raw**
Given the path's result
When it is inspected against raw command output
Then it is a drawn-together draft carrying its `prp_` id, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a proposal-drafting agent, or both is decided during shaping. (The developer confirmed the form is deferred, mirroring how 062, 064, 065, and 066 deferred their decomposition.)
- **[ASSUMED] Entry is an existing tension id**: the path starts from a `ten_` id already on the record — handed off by 066 or identified by the practitioner — and reads it back; it does not capture the tension. (The developer confirmed drafting begins from a ready tension.)
  *Superseded in part — 2026-08-22, by 079-pre-assembly-grammar-consultation: the anchor `ten_` id is optional at entry — when none is handed in (or the handed-in one routes the change elsewhere), the path's routing determination names the target circle and the eligible anchors and returns awaiting the practitioner's choice. What survives unchanged: a settled anchor must be a tension already on the record, and the path still captures nothing.*
- **[ASSUMED] Scope is drafting only**: assemble the change set and create the draft proposal, then hand off. Circulation is the Proposal Circulation Path (068), the response side is 069, and authority judgment is the Constraint Discovery Path (065). (The developer set these boundaries explicitly.)
- **[ASSUMED] Change-set help is assembly, not typed construction**: the path helps source and shape the JSON array (each element carrying a non-empty `type`) and passes it through verbatim; it does not reimplement per-type schemas — that is the deferred *Unguided Change Construction* capability. (The developer confirmed the pass-through boundary.)
- **[ASSUMED] Situating is by circle and status, not by tension** (technical): the proposal list (056) filters by status, circle, proposer, and date but offers no tension-id filter, so the path situates by circle and `draft` status rather than by the anchor `ten_` id. (Grounded in 056's Behavioral Accord.)
- **Composed commands are already shipped** (technical): Proposal Creation (055), Proposal Reads (056), and Tension Reads (043) all exist in the CLI today, so the path composes them rather than waiting on new commands. (Grounded in the shipped proposal and tension command families.)
- **The proposal create is a gated, Premium-gated write** (technical): the create returns `403` when async proposals are disabled and is the boundary the Write-Safety Guardrail (063) guards, so the path routes it through the confirmed write flow. (Grounded in 055's and 063's Behavioral Accords.)

---

## Ambiguity Warnings

None remaining — the entry point, the guardrail crossing, the change-set assembly boundary (assembly vs. typed construction), the drafting-only scope with 065 / 068, and the situating narrowing available from 056 were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-18

- **Guardrail boundary**: The proposal create is a gated governance write. The path surfaces the assembled anchor and change set for confirmation and runs the create through the Write-Safety Guardrail (063) confirmed write flow — it is the first operator path to cross that boundary, and it does not treat the write as ungated.
- **Change-set assembly**: The path helps source and shape the `changes` JSON array (each element an object carrying a non-empty `type`) and passes it through verbatim; it does not build a typed per-change constructor or validate command-specific keys — that is the deferred *Unguided Change Construction* capability, and the server validates.
- **Scope boundary**: Drafting covers assembling the change set and creating the `draft` proposal only. The path stops at the created draft and hands the `prp_` id to Proposal Circulation Path (068); it does not advance, withdraw, respond, or circulate.
- **Situating**: Before drafting, the path surfaces proposals already in flight in the relevant circle — narrowing the proposal list (056) by circle and `draft` status — so a new draft is a deliberate addition. It situates by circle and status because that is what the list exposes; it does not claim a tension-id filter the CLI does not offer.
