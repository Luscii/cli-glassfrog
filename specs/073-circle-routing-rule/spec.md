# Specification: Circle Routing Rule

**Feature**: 073-circle-routing-rule
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

A proposal has no circle of its own. `proposal create` takes an anchor tension and nothing else, so the proposal lands in the circle of that tension's sensing role — the circle is chosen for the operator, by whichever tension they anchored on. The consequence is counter-intuitive and cost a wasted draft during the write path's first live use: a change to a **circle's own** governance cannot be proposed from inside that circle. It must be anchored in the **parent** circle, on a tension sensed by a role the operator fills there — with the circle-role itself working when the operator is Circle Lead.

Circle Routing Rule is the recorded routing content for that question: where a change must land, and which tension can anchor it there. **Terminology**: this spec says "the content" for what the artifact states and "the record" for the artifact itself; downstream artifacts favour "the record". The two name the same thing, and a constraint on one is a constraint on the other. It carries two things — the **rule**, and the **procedure** for applying it against reads the CLI already has, naming which reads to run and how to report what they find. It does not perform the determination. Nothing invokes this content when it lands; the invocation, and the scenarios that verify an applied determination, belong to the pre-assembly gate that consumes it — the same shape the change-set grammar facts take. Two boundaries hold it in place, matching that sibling: it **informs** an assembler but never refuses a write locally (the server stays the judge), and where its procedure rests on a read that cannot prove completeness, it prescribes saying so rather than asserting an absence.

---

## Behavioral Accord

### The rule

- When the content is consulted for where a proposal lands, it states that a proposal inherits the circle of its anchor tension's sensing role, and that the create carries no circle parameter of its own — so the anchor choice *is* the routing choice.
- When the content is consulted for a change to a circle's own governance, it states the change must be anchored in that circle's **parent**, on a tension sensed by a role the operator fills there.
- When the content is consulted for which of the two cases applies, it states the test that tells an own-circle change from a change to a role inside a circle, and names the read that resolves whether a given target is a circle — so a consumer can route from the change target alone rather than being told which case it is.
- When the operator fills the circle's own Circle Lead role, the content states the circle-role itself is a valid anchor site for that circle's own governance.
- When the content is consulted for an ordinary change — one targeting a role inside a circle rather than the circle itself — the same stated rule routes it, with no separate special case to look up.
- When the target circle has no containing circle, the content states that there is no parent to route to and that it does not resolve this case, rather than naming a default target.

### The procedure

- When the content is consulted for how to reach an answer, it names the reads to run and their order: the operator's own roles to find which roles they fill in the target circle, then the tensions sensed on each of those roles.
- When the content prescribes how an answer is stated, the target circle is named by its `role_` id and each eligible anchor tension by its `ten_` id, so the answer can be acted on without a further lookup.
- When several anchors are eligible, the content prescribes naming all of them and choosing none.

### Reporting a gap

- When the operator fills a role in the target circle and no tension is sensed on it, the content prescribes naming capture on that specific role as the step that would close the gap.
- When no role the operator fills is found in the target circle, the content prescribes reporting **none found** and naming the read the search rested on, rather than reporting that no such role exists.
- When an answer rests on the operator's own-roles read, the content prescribes marking that answer's completeness uncertain, because the read does not follow pagination and an absence in it is an absence in what was read.

### Consumption

- When the content is consumed, it informs what an assembler anchors on; nothing it prescribes refuses, blocks, or delays a proposal create.
- When this capability lands, no surface consults it to route — it is consultable content whose application, and the scenarios that verify an applied determination, belong to the pre-assembly gate that consumes it.

---

## User Scenarios

**In order to** not spend a gated write on a proposal that lands in a circle that cannot decide it,
**as an** AI agent about to assemble a change set,
**I want to** read the recorded rule and the procedure that names which circle to look for and which of the operator's tensions can anchor it there.

**In order to** know what to do when I have no usable anchor rather than being told only that I have none,
**as a** practitioner whose agent is drafting on my behalf,
**I want to** read a procedure that requires naming which role in which circle to capture a tension on, and how certain that conclusion is.

**In order to** build the pre-assembly gate against one routing authority instead of re-deriving the rule,
**as a** developer implementing the gate that consumes it,
**I want to** work from a single recorded source that states the rule, its own-circle consequence, the Circle Lead exception, and the reads that answer it.

---

## Non-Behaviors

- The content must not itself perform the routing determination. **Why**: the invocation belongs to the pre-assembly gate that consumes it; a capability that both records the procedure and runs it would duplicate the gate and leave two places to change when routing changes. The reads the procedure names *do* enter the drafting path's composed surface in this spec, so that surface matches what the content names — but no workflow step consults the content or runs those reads to route.
- The content must not refuse, block, or gate a proposal create through anything it prescribes. **Why**: local governance judgment is VISION Exclusion 2 — the server is the sole judge — and the procedure rests on a read that cannot prove completeness, so a local block would sometimes stop a write the server would have accepted.
- The content must not prescribe reporting "the operator fills no role in that circle" as a settled fact. **Why**: the own-roles read does not follow pagination, so an absence there is an absence in what was read; presenting it as definitive would send an operator to capture a tension on a new role when a usable anchor already existed on one that went unread.
- The content must not present its routing conclusion as v5-contract-authoritative. **Why**: the landing behaviour is an empirical observation of a live server, not published contract (VISION Principle 1); marking it as contract would invite trust past the point the real contract moves underneath it.
- The content must not capture a tension, create a proposal, or perform any other write. **Why**: it is consultable knowledge; capturing the tension it names as the closing step belongs to the Tension Processing Path, and folding a write in would make a knowledge artifact a write surface.
- The content must not teach Holacracy practice beyond the routing question. **Why**: governance coaching is VISION Exclusion 1 — the content describes where the server puts a proposal and what anchors are available, not how to practise governance.
- The content must not restate the change-set shape facts its sibling record owns. **Why**: routing and shape are separate questions sharing one gate; a second copy of either drifts from its owner on the next correction.

---

## Integration Boundaries

- **My Roles (`me roles`)**: the read the procedure names first — the roles the operator fills and the circles those sit in, the basis for anchor eligibility. Does not follow pagination, so an absence in its result is not proof of absence; every conclusion the procedure draws from it inherits that limit.
- **Tension Reads (`tension list <role-id>`, `tension get <ten-id>`)**: the read the procedure names second — the tensions already sensed on a role the operator fills, and confirmation of a chosen anchor's sensing role. Read-only.
- **Role Reads (`roles [id]`)**: the read that resolves whether a target is a circle, and its containing role (`parent_role_id`) when the target circle is not among the operator's own roles. A declared dependency of this capability alongside the two above.
- **Pre-Assembly Grammar Consultation (downstream consumer)**: the gate that invokes the routing procedure before a change set is assembled. It owns the wiring, the applied determination, and the scenarios that verify an application; this capability owns the content.
- **Proposal Drafting Path (context)**: the path whose anchor tension the routing governs. Unchanged by this capability on its own — it still takes the anchor tension it is given.
- **Change-Set Grammar Facts (sibling)**: shares the same gate. It owns the *shape* of a change; this owns *where the change lands*.
- **`LEARNINGS.md` recorded fact F7 (provisional source)**: currently holds the routing observation. On landing, the routing content lives here and the supersession is recorded via `/score:deprecate` rather than leaving two copies to drift.

---

## Driving Scenarios

### Happy path

**Scenario: A change to a circle's own governance routes to the parent**
Given the recorded routing content
When it is consulted for a change to a circle's own domain or policy
Then it states the change must be anchored in that circle's parent
And it states the reason: the proposal lands in the circle of whichever tension anchors it.

**Scenario: The content states how to tell the two cases apart**
Given the recorded routing content
When it is consulted for which case a change falls under
Then it states the test that distinguishes a change to a circle's own governance from a change to a role inside a circle
And it names the read that resolves whether a given target is a circle.

**Scenario: The procedure names the reads and how to state the answer**
Given the recorded routing content
When it is consulted for how to reach an answer
Then it names the operator's own-roles read and the tension read on the roles that read finds, in that order
And it prescribes naming the target circle by its `role_` id and each eligible anchor tension by its `ten_` id.

**Scenario: The Circle Lead exception is stated**
Given the recorded routing content
When it is consulted for a circle's own governance where the operator fills that circle's Circle Lead role
Then it states the circle-role itself is a valid anchor site in that case
And it does not send the operator to the parent circle to find one.

### Error scenarios

**Scenario: The procedure prescribes what to say when no tension is sensed**
Given the recorded routing content
When it is consulted for the case where the operator fills a role in the target circle and no tension is sensed on it
Then it prescribes reporting that no eligible anchor exists yet
And it prescribes naming capture on that specific role in that specific circle as the step that would close the gap.

**Scenario: Nothing the content prescribes stops a write**
Given the recorded routing content read end to end
When it is inspected for what it would have a consumer refuse
Then nothing it prescribes refuses, blocks, or delays a proposal create
And the server remains the judge of what it accepts.

### Edge cases

**Scenario: An ordinary change needs no parent hop**
Given the recorded routing content
When it is consulted for a change to a role inside a circle rather than to the circle itself
Then the same stated rule routes it to the circle containing that role
And no separate case has to be looked up for it.

**Scenario: An absence that cannot be proven must be reported as unproven**
Given the recorded routing content
When it is consulted for how to report finding none of the operator's roles in the target circle
Then it prescribes reporting "none found" and naming the read the search rested on
And it prescribes marking the conclusion uncertain rather than stating the operator fills no role there.

**Scenario: The target circle has no parent**
Given the recorded routing content
When it is consulted for a change to the governance of a circle that has no containing circle
Then it states there is no parent circle to route to and that it does not resolve this case
And it does not name the circle itself, or any other circle, as a default target.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The content carries routing only**
Given the landed routing content
When it is read end to end
Then everything it records answers where a change lands and what can anchor it there
And no change-set shape fact appears that its sibling record owns.

**Scenario: Nothing prescribed refuses a write**
Given the landed content and the reads it names
When it is inspected for what it causes
Then nothing in it has a consumer reject, filter, or gate a proposal create before the server sees it
And its only effect is to inform what an assembler anchors on.

**Scenario: Every unprovable absence is prescribed as hedged**
Given every statement the content makes about finding none of the operator's roles somewhere
When those statements are read
Then each prescribes phrasing as none found in a named read, with completeness marked uncertain
And none prescribes asserting a settled absence.

**Scenario: The content ships unconsulted**
Given this capability landed on its own
When the surfaces around it are inspected
Then no workflow step consults the routing content or runs its named reads to route — the drafting path's workflow is unchanged
And the content asserts no outcome of an applied determination — the application and its verification are left to the pre-assembly gate that consumes it.

**Scenario: The composed surface matches what the content names**
Given the reads the recorded procedure names
When the drafting path's composed-leaf registry and its agent artifact are read
Then every read the content names appears in both, and each still resolves as a command the CLI exposes
And no write leaf entered the composed surface alongside them.

---

## Assumptions

- **Content and procedure, not application**: The capability delivers the routing rule and the procedure for applying it as something a consumer reads; the invocation and every scenario asserting an applied determination belong to the pre-assembly gate. (Confirmed with the developer during specification; matches the FEATURE-MODEL note that this "contributes the routing content, not a second gate.")
- **The rule is stated generally**: The mechanism is recorded once, with the own-circle case as its consequence and the Circle Lead case as its exception, rather than recording only the counter-intuitive case. (Confirmed with the developer during specification — stating only the exception would leave the ordinary case unrouted.)
- **Absence is hedged, never blocking**: A gap is reported with the step that would close it and with its certainty marked, and nothing prescribed stops a write. (Confirmed with the developer; grounded in `internal/glassfrog/me.go` — the own-roles read decodes the next-page cursor but does not follow it.)
- **Circle Lead exception carried as recorded**: The exception is taken from the recorded fact's own parenthetical and is not independently re-verified by this capability. (Informed by `LEARNINGS.md` 2026-08-05, F7.)
- **[ASSUMED] Physical home deferred**: The content's file location and format are assumed to be a shaping decision, not fixed here — the behavioral requirement is a single routing source, however it is materialized. (Mirrors the sibling record's deferral.)

---

## Ambiguity Warnings

_None. The root-circle case was resolved during clarification (the content states the limit and declines to name a target). The remaining open decision — the content's physical home and format — is technical, deferred to `/score:plan`._

---

## Clarifications

### Session 2026-08-08

- **What the capability delivers**: Narrowed to the recorded rule and the application procedure. Every scenario that asserted the outcome of an applied determination moved out; the application, and the scenarios verifying it, belong to the pre-assembly gate that consumes this content. The behavioral accord's application and gap-reporting groups now describe what the content *prescribes* rather than what it emits, and a non-behavior forbids the content performing the determination itself.
- **Which case a change falls under**: The content states the test that distinguishes a change to a circle's own governance from a change to a role inside a circle, and names the read that resolves whether a target is a circle — so a consumer routes from the change target alone rather than being told which case applies.
- **The root circle**: For a circle with no containing circle, the content states the limit and declines to name a target, rather than defaulting to the circle itself. A wrong default would cost a proposal that cannot be decided anywhere. The `[NEEDS CLARIFICATION]` marker is resolved and removed.
- **Third read dependency**: `roles [id]` is a declared dependency alongside My Roles and Tension Reads — it resolves both whether a target is a circle and the containing role of a target circle the operator fills no role in. The `BACKLOG.md` entry for this item was corrected to match.

### Session 2026-08-08 (amendment during `/score:shape`)

- **The composed surface widens in this spec, not in #77**: The developer directed that the three reads the procedure names (`me roles`, `tension list`, `roles`) enter the drafting path's composed-leaf registry and its agent artifact now, rather than when the pre-assembly gate wires them. Two statements were amended to stay true: the non-behavior forbidding the content from invoking the reads it names now forbids only the content performing the determination (the composed surface widening is stated explicitly beside it), and the "ships unwired and unapplied" validation scenario became "ships unconsulted" — the invariant that survives is that no *workflow step* consults the content or runs the reads to route. A new validation scenario checks the composed surface agrees with what the content names. Recorded here because the amendment came from planning, not from a specification gap: the spec as written was accurate for the original plan and became untrue only under this decision.
