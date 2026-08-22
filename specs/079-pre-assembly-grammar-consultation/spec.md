# Specification: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Three artifacts now exist for building a proposal's change set correctly, and nothing consults any of them. The change-set grammar facts record and the circle-routing rule record both ship inside the drafting path's own `references/` directory, deliberately inert — the routing record's three reads were even added to the path's composed surface ahead of a workflow that uses them, under an explicit note that their presence implies no routing step. Agent-Facing Grammar Reference then put the same grammar in the binary as a client-less read command. The drafting workflow still assembles a change set and picks an anchor tension without consulting either. The knowledge is shipped; it is not applied. Both shapes that the records exist to prevent — a policy change wrapped in a role operation, and a self-targeting role update that comes back dead — are still reachable by an assembler that has the answer sitting unread in its own directory.

Pre-Assembly Grammar Consultation is the gate that wires them in: before a change set is assembled, the drafting path determines where the change must land through the recorded routing procedure and consults the grammar through the CLI's read — and where the assembled set matches a shape the grammar records as dead, it surfaces that fact **before** the gated write rather than leaving the operator to discover it after. It is one gate serving both questions, which is how the portfolio models it: shape and routing are separate content with separate owners sharing a single point of application. The gate **informs and never withholds** — it adds no command, expresses no validity verdict, and blocks no write; the server stays the sole judge (VISION Exclusion 2). What changes is that the drafting path arrives at its one gated write already knowing what the record knows.

---

## Behavioral Accord

### The order of the gate

- When the path drafts, consultation runs in one order: the routing determination establishes the target circle and the eligible anchors, the grammar is consulted, the change set is assembled, and the assembled set is then matched against the recorded dead shapes — all of it before the gated write.
- When the matching step runs, the routing determination's answer is available to it, because a recorded dead shape may be defined partly by where the proposal would be anchored — the self-targeting shape is recognizable only when both the change's target and the circle it would be proposed from are known.

### Consulting the grammar

- When the path reaches the point of assembling a change set, it consults the change-set grammar first — the sayable change types, where each may appear, and the recorded dead shapes with their symptoms are in view while the set is built, not after it is built.
- When the grammar is consulted, it is consulted through the CLI's grammar read rather than a file shipped beside the path, so the reference resolves on any machine that has the CLI.
- Every draft consults the grammar, whatever the change targets; there is no condition under which assembly proceeds unconsulted by design.
- When the grammar read fails, the path surfaces the failure, records that the grammar was not consulted, and continues — drafting is not withheld, and the result carries the gap rather than presenting the assembly as consulted.

### Determining where the change lands

- When a change is to be drafted, the path determines the **target circle** and the **anchor tensions eligible to route it there** before an anchor is settled on — the create carries no circle of its own, so the anchor choice is the routing choice.
- When the determination runs, it follows the recorded routing procedure — its named reads in the order the record names them — and reports its answer as the target circle together with **every** eligible anchor tension, naming all of them and choosing none.
- When an anchor tension has already been handed in, the determination runs against it: where it lands the change in the target circle, drafting proceeds on it; where it does not, the path surfaces the mismatch and names the eligible anchors that would reach the target. The practitioner may still direct the draft onto the anchor they chose — the path reports the mismatch and withholds nothing.
- When the operator fills a role in the target circle but no tension is sensed on it, the path reports that no eligible anchor exists yet and names capture on that specific role in that specific circle as the step that closes the gap — handing that step onward rather than performing it.
- When the target circle has no containing circle, the path reports that the recorded rule resolves no target for that case rather than choosing one.
- When the determination rests on the operator's own-roles read, the path names the read the search rested on and marks the conclusion's completeness uncertain — an absence there is an absence in what was read, never a settled absence.
- When a routing read fails part-way, the path names what failed, presents the determination as incomplete, and continues on what was established — it neither invents the unread part nor abandons what it has.
- Every draft is routed, whatever the change targets: the classification test is what tells which case applies, so its answer cannot also be the condition for running it.

### Surfacing a recognized dead shape

- When the assembled change set matches a shape the grammar records as dead, the path surfaces that fact **before** the write — naming the fact's handle, the shape, and the symptom the wrong shape produces — so the recognition arrives ahead of the gated write rather than after it.
- When a match is surfaced, the practitioner decides whether to proceed. The path does not withhold the write, does not alter the change set, and expresses no verdict on the set's validity — it reports what the record says about a shape it recognized.
- When no recorded shape matches, the path says so; silence is not the signal that the set was looked at.
- When no recorded shape matches, nothing about validity is implied — a change set the record does not name is not thereby endorsed.

### Reporting the consultation

- When the path returns its result, the result carries what was consulted and what consultation surfaced: that the grammar was read, the routing determination and its answer, and any recognized dead shape.
- When consultation was incomplete — a failed grammar read, an incomplete routing read — the result names which part was incomplete and why, rather than presenting a partial consultation as a whole one.

### Staying within the operator layer

- The path adds no command, flag, or capability of its own; consultation composes reads the CLI already exposes.
- The write the path crosses is unchanged: exactly one gated proposal create, always through the confirmed write flow. Consultation itself asks the operator to confirm nothing.

---

## User Scenarios

**In order to** assemble a change set the server will take on the first attempt, instead of learning each shape from a refused round-trip inside a gated write,
**as an** AI agent drafting a proposal,
**I want** the grammar consulted before I build, and a recorded dead shape named before the write rather than after it.

**In order to** not spend a confirmed governance write on a proposal that lands in a circle which cannot decide it,
**as a** practitioner served through the agent,
**I want** the target circle and the anchors eligible to reach it established before an anchor is settled on.

**In order to** trust that the shipped grammar and routing records are actually being applied,
**as a** practitioner reviewing what the agent did,
**I want** the returned draft record to say what was consulted and what it surfaced.

---

## Non-Behaviors

- The path must not refuse, block, or delay the proposal create on anything consultation surfaces, nor withhold a draft on the anchor the practitioner has chosen. **Why**: local governance judgment is VISION Exclusion 2 — the server is the sole judge — and the routing determination rests on a read that cannot prove completeness, so a local withholding would sometimes stop a write the server would have accepted. Every finding this gate produces is a report the practitioner acts on.
- The path must not build a typed per-change validator, nor check a change's `type` value or its command-specific keys. **Why**: recognizing a named match against a recorded shape is not schema validation; the drafting path's verbatim pass-through above the `type` floor stands, and typed construction is the separate Typed Change Builders capability.
- The path must not restate the grammar content or the routing rule inside its own workflow text. **Why**: the grammar read and the routing record are the single sources; a copy in the workflow drifts from its owner on the next correction, which is exactly the failure the single-source discipline exists to prevent.
- The path must not capture, refine, or retire a tension — including the capture the routing gap names as the closing step. **Why**: tension writes are the Tension Processing Path; naming the closing step is guidance, performing it would make this gate a second write surface.
- The path must not ask the practitioner for a change target's numeric identifier. **Why**: that ask is Identifier Prompt Before Assembly, a separate capability that rides on this gate; landing it here would ship it without its own specification.
- The path must not change what the grammar read renders or what the routing record states. **Why**: those contracts belong to Agent-Facing Grammar Reference and Circle Routing Rule; this capability is the wiring, and a content edit here would put the same fact under two owners.
- The path must not advance, withdraw, or respond to a proposal, nor judge whether the change needs a proposal or is within the practitioner's authority. **Why**: unchanged from the drafting path's existing boundaries — circulation, the response side, and the Constraint Discovery Path own those.
- The path must not present consultation as a guarantee of acceptance. **Why**: the grammar carries what is recorded, not everything the server checks; "consulted" reading as "will be accepted" would restore the false confidence the post-create validity read exists to remove.

---

## Integration Boundaries

- **Agent-Facing Grammar Reference (upstream)**: the read this gate consults, invoked as a CLI read that makes no request and needs no credential. It tracks nothing about whether consultation happened — that reporting lives here.
- **Circle Routing Rule (upstream)**: the recorded routing content, its classification test, and the procedure with its named reads. That capability owns the content; this one owns the invocation, the applied determination, and the scenarios that verify an application.
- **Change-Set Grammar Facts (upstream)**: the source of the recorded dead shapes the grammar read renders. A fact added or retired there changes what this gate can recognize, with no edit here.
- **Proposal Drafting Path (the host)**: the gate lands inside this path's workflow, ahead of assembly. The path's **entry contract widens** — from taking a handed-in anchor as its starting point to determining where the change lands before an anchor is settled on. Its single gated write, its situating step, and its handoff to circulation are unchanged. The path's composed surface must grow the grammar read, and the standing note that its three routing reads imply no routing step ceases to be true.
- **Tension Processing Path (onward handoff)**: receives the capture step the routing gap names when no eligible anchor exists yet, and remains the upstream that hands an anchor in. A handed-in anchor is still the ordinary entry; what changes is that the determination runs against it rather than around it.
- **Identifier Prompt Before Assembly (downstream)**: rides on this gate rather than introducing its own; this specification leaves room for it and does not perform it.
- **Write-Safety Guardrail**: the grammar read and the routing reads are reads, and the gate must not turn consultation into something the operator confirms. The one confirmed write remains the create.
- **Operating-surface self-containment**: every reference the wired workflow makes resolves within the surface or to the CLI it drives — the gate names in-surface components and CLI commands, not development-repository artifacts.

---

## Driving Scenarios

### Happy path

**Scenario: The gate runs in order before the write**
Given a change ready to be drafted
When the path reaches the drafting step
Then the routing determination runs first, the grammar is consulted next, the change set is assembled, and the assembled set is matched against the recorded dead shapes
And all of it happens before the gated create.

**Scenario: The target circle and its eligible anchors are established first**
Given an intended change and no anchor settled on
When the path runs the recorded routing procedure
Then it reports the target circle and every anchor tension eligible to route the change there
And it chooses none of them — the choice is the practitioner's.

**Scenario: A recognized dead shape is surfaced before the write**
Given an assembled change set matching a shape the grammar records as dead
When the path reaches the gated create
Then it surfaces the fact's handle, the shape, and the symptom before the write
And the practitioner decides whether to proceed — the path withholds nothing.

### Error scenarios

**Scenario: The grammar read fails**
Given a grammar read that fails
When the path continues
Then it surfaces the failure and records that the grammar was not consulted
And drafting continues with the gap carried in the result rather than being withheld.

**Scenario: A routing read fails part-way**
Given a routing determination whose reads fail before the procedure completes
When the path reports its answer
Then it names what failed and presents the determination as incomplete
And it continues on what was established, neither inventing the unread part nor abandoning what it has.

### Edge cases

**Scenario: A handed-in anchor routes the change elsewhere**
Given an anchor tension handed in and a determination showing it lands the change outside the target circle
When the path evaluates the handed-in anchor
Then it surfaces the mismatch and names the eligible anchors that would reach the target circle
And where the practitioner directs the draft onto the handed-in anchor anyway, drafting proceeds — the mismatch is reported, not enforced.

**Scenario: An anchor-dependent dead shape is recognized**
Given a change set whose role operation targets the circle the proposal would be anchored in
When the path matches the assembled set against the recorded dead shapes with the routing determination's answer in hand
Then it recognizes the self-targeting shape and names its symptom before the write
And the recognition rests on both the change's target and the circle the proposal would be anchored in.

**Scenario: The practitioner proceeds past a surfaced dead shape**
Given a recognized dead shape surfaced before the write
When the practitioner directs the path to proceed anyway
Then the create runs through the confirmed write flow unchanged
And the path neither withholds the write nor alters the change set.

**Scenario: No eligible anchor exists yet**
Given a target circle where the operator fills a role but no tension is sensed on it
When the routing determination reports
Then it states that no eligible anchor exists and names capture on that specific role in that specific circle as the step that closes the gap
And it does not capture the tension itself.

**Scenario: The change set matches nothing recorded**
Given an assembled change set matching no recorded dead shape
When the path reaches the write
Then it states that no recorded shape matched
And it implies nothing about the set's validity.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Consultation is unconditional and ordered**
Given the wired drafting workflow
When it is inspected for a path from intent to a created draft
Then no such path reaches assembly without consulting the grammar, none reaches a settled anchor without the routing determination, and none matches the assembled set against the recorded shapes before the routing determination has answered.

**Scenario: Nothing withholds a write locally**
Given the wired workflow and everything consultation can surface
When it is inspected for a refusal, block, filter, delay, or withheld draft applied before the server sees the create
Then none is present — every surfaced finding leaves the decision with the practitioner.

**Scenario: No content was copied into the wiring**
Given the wired workflow
When its text is compared against the grammar the CLI renders and the routing record
Then it names and invokes them without restating a change type, a placement rule, a recorded shape, or the routing rule.

**Scenario: The inert-reads note no longer misdescribes the path**
Given the drafting path's composed surface after this capability lands
When the annotation stating that the routing reads imply no routing step is checked
Then it no longer stands — the surface describes a path that does route.

**Scenario: The consultation is legible in the result**
Given a completed drafting run
When its returned record is read by someone who did not watch the run
Then they can tell that the grammar was consulted, what the routing determination answered, and whether a dead shape was surfaced.

---

## Assumptions

- **Entry widens rather than stays as it was** (developer decision, taken during specification): the drafting path's entry moves from "takes a handed-in `ten_` id" to "determines the target circle and names the eligible anchors before an anchor is settled on". The alternative — verifying a handed-in anchor only — was considered and declined. The upstream path's own entry accord therefore needs a matching amendment; this specification states the widening, and the amendment sweep belongs to shaping.
- **The anchor must still already exist on the record**: the widened entry determines and names eligible anchors from what the record already holds. Where none exists, the gate reports the gap and hands the capture onward; it never creates the anchor. (Follows from the tension-write boundary, which this capability does not move.)
- **Recognition is a named match, not a schema check**: "a recognized dead shape" means the assembled set matches a shape the record names, reported with that fact's handle. It is not a per-type field check, and it carries no claim of completeness over shapes the record does not name. (Confirmed with the developer during specification.)
- **Both consultations run unconditionally** (developer decision): the grammar read is client-less and costs no request; the routing procedure's reads run on every draft, because the classification test is what tells which routing case applies and so cannot also gate whether it runs. The cheaper conditional variant was considered and declined.
- **Consultation is reported in the returned record** (developer decision): the result gains a consultation element, so the gate is falsifiable from the operator's side rather than being an unverifiable claim about a workflow step.
- **The grammar read joins the path's composed surface** (technical): the drafting path may invoke only the leaves its composed-surface registry names, and the grammar read is not among them today, so consultation requires that surface to grow. (Grounded in the registry's stated single-source role.)
- **The routing reads are already on the composed surface** (technical): they were added ahead of this capability, so the wiring needs no further widening for them — only the removal of the note declaring them inert. (Grounded in the registry's annotated routing block.)

---

## Ambiguity Warnings

_None behavioral. The open behavioral questions — recognition strength, what the routing determination does with a handed-in anchor, the order of the gate's steps, whether consultation is conditional, its conduct when a read fails, and whether it is reported — were all settled with the developer and are recorded under Assumptions and Clarifications. The remaining open decisions are technical and belong to their stages: how the widened entry and the ordering are expressed across the path's skill, agent, and registry artifacts, and where the amendment sweep of the upstream entry accord is sequenced (shaping)._

---

## Clarifications

### Session 2026-08-21

- **Informing versus withholding**: The accord as first drafted said a handed-in anchor "is accepted only once the determination shows it lands the change in the target circle" and that the path surfaces a mismatch "instead of drafting on it" — a local withholding that contradicted the non-behavior forbidding the gate to refuse, block, or delay the create. Resolved in favour of the non-behavior, which now also names withholding a draft explicitly: the mismatch and the eligible anchors are reported, and the practitioner may still direct the draft onto the anchor they chose. The affected edge-case scenario now asserts that drafting proceeds on the practitioner's direction.
- **The order of the gate, and why recognition depends on routing**: The two consultations were presented as parallel and unordered, leaving open whether dead-shape matching can see the routing answer. It must — the recorded self-targeting shape is defined partly by the circle the proposal would be anchored in, so it is unrecognizable unless routing has already answered. A new accord group pins one order (route → consult → assemble → match, all before the write), a new edge-case scenario exercises the anchor-dependent recognition, and the ordering joins the unconditional-consultation validation hold-out.
- **Conduct when a read fails**: A failed grammar read does not stop drafting. The path records that the grammar was not consulted and continues, carrying the gap in its result; the same continuation now applies to a routing read that fails part-way. This keeps the never-withhold fence whole rather than creating one case where the gate does stop the flow.
- **Proving the never-withhold boundary**: A scenario was added for the practitioner proceeding past a surfaced dead shape, so the boundary is exercised rather than only asserted in a non-behavior — the create runs through the confirmed write flow unchanged and the change set is not altered.
