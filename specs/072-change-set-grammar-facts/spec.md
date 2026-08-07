# Specification: Change-Set Grammar Facts

**Feature**: 072-change-set-grammar-facts
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Building a proposal means assembling a `changes[]` array of governance commands, and the CLI passes that array to the server as supplied — the server is the sole judge of what is valid. Most of the grammar for those commands is now published: the v5 contract enumerates the change types and states that accountability/domain edits must ride as children of a role operation. But the write path's first live exercise surfaced two shapes the published contract still does **not** carry — and one of them lets the CLI report success for a proposal the server has already marked dead. An assembler that knows only the published contract will get these two shapes wrong.

Change-Set Grammar Facts is the **single recorded source** for exactly those residual shapes: what the server accepts, what it rejects, and — for the shape that is accepted-but-invalid — the symptom each produces. It is a curated knowledge artifact, not a command: it carries no runtime behavior, adds no API capability, and reaches an agent only through the two solutions that consume it (a rendered grammar reference the assembler loads, and typed change builders scoped to what it verifies). It replaces the provisional home these facts occupy in `LEARNINGS.md` and becomes the source of truth they are maintained against. Two founding boundaries hold it in place: it **informs** an assembler but never pre-rejects a change set locally (the server stays the judge), and it records **empirical observations**, never anything presented as v5-contract-authoritative.

---

## Behavioral Accord

### Content

- When the record is consulted for the shape of an own-circle policy change, it carries the fact that such a policy is a **top-level `CreatePolicy` part with no `UpdateRole` wrapper**, together with the symptom of getting it wrong.
- When the record is consulted for a role update that targets the circle from inside its own governance, it carries the fact that the server **accepts this at create but returns it invalid** — `valid: false`, a blocking alert, and no available transitions — together with the symptom the caller observes.
- When the record names a change-type shape the published contract already carries — the enumerated change types, the rule that accountability/domain edits are nested-only — it **cites the contract** rather than restating the shape.

### Structure

- When any shape is recorded, it is paired with the observable symptom of using it wrongly, so a consumer learns not just the shape but how a mistake presents.
- When a shape is recorded, its disposition is stated explicitly — **accepted**, **rejected**, or **accepted-but-invalid** — so "the server took it" is never conflated with "the server considers it valid."
- When a fact is recorded, its status as an **empirical observation** (not published contract) is marked on the fact itself.

### Consumption

- When a consumer reads the record before assembling a change set, it can find the accepted shape to use and the dead shape to avoid without a refused round-trip against the server.
- When the record is consumed, it supplies knowledge only; nothing in it causes a change set to be rejected locally before it reaches the server.

### Maintenance

- When the published contract later absorbs a recorded fact, that fact **retires** from the record and its supersession is recorded, so a shape is never carried in two places at once.

---

## User Scenarios

**In order to** assemble a valid change set without discovering the two undocumented shapes by refused round-trips,
**as an** AI agent drafting a proposal,
**I want to** read the accepted shape and the dead shape — each with its symptom — before I build the `changes[]` array.

**In order to** build the downstream grammar reference and typed change builders against one authority instead of scattered notes,
**as a** developer implementing the write-path fidelity solutions,
**I want to** a single recorded source that says exactly which residual shapes are verified and which are merely published contract.

**In order to** keep the record honest as the API moves,
**as the** maintainer,
**I want to** retire a recorded fact the moment the published contract absorbs it, rather than letting two copies drift.

---

## Non-Behaviors

- The record must not cause a change set to be rejected locally before it reaches the server. **Why**: local governance judgment is VISION Exclusion 2; the server is the single judge of validity, and a local pre-reject would diverge from the server's own verdict and block shapes the server would accept.
- The record must not present its facts as v5-contract-authoritative. **Why**: these are empirical observations of undocumented behavior (VISION Principle 1 — nothing outside the published spec may be published as spec-authoritative); marking them as contract would invite a consumer to trust them past the point the real contract changes underneath them.
- The record must not restate change-type shapes the published contract already carries. **Why**: a second copy of the enum or the nested-only rule drifts from the contract on the next refresh; citing the contract keeps the residue — the only part that needs recording — distinct from the part that maintains itself.
- The record must not perform runtime detection of the accepted-but-invalid proposal. **Why**: detecting a dead proposal after create is the separate read-back capability's job; this record only states that the shape is a shape to avoid, so the two capabilities do not duplicate the detection.
- The record must not carry change-target identifier facts (legacy/numeric id resolution). **Why**: those belong to the identifier-surfacing capabilities; folding them in would blur what "grammar" means and couple this record to a capability with its own retirement clock.

---

## Integration Boundaries

- **Glassfrog API v5 spec (`spec/glassfrog-api-v5.yaml`)**: the citation target. The record reads from it to decide what it must *not* restate; the recorded facts are exactly the residue the spec does not carry. When the spec absorbs a recorded shape (as it absorbed the type enum and the nested-only rule), the corresponding fact retires.
- **Agent-Facing Grammar Reference (downstream consumer)**: renders the recorded facts as the reference an assembling agent loads. Reads this record; adds no facts of its own.
- **Typed Change Builders (downstream consumer)**: scopes its per-type builders to the shapes this record verifies. Reads this record to decide which types to build.
- **`LEARNINGS.md` (provisional source, superseded on landing)**: currently holds these facts as a provisional home. On landing, the facts move here and the supersession is recorded via `/score:deprecate`, leaving no duplicate to drift.

---

## Driving Scenarios

### Happy path

**Scenario: The own-circle policy shape is recorded with its symptom**
Given the record of change-set grammar facts
When it is consulted for how to change a circle's own policy
Then it states the change is a top-level `CreatePolicy` part with no `UpdateRole` wrapper
And it carries the symptom that the web UI generates exactly this shape and a wrapped shape is refused.

**Scenario: The accepted-but-invalid shape is recorded with its disposition**
Given the record of change-set grammar facts
When it is consulted for a role update that self-targets the circle from inside its own governance
Then it states the server accepts this at create but returns it `valid: false` with a blocking alert and no available transitions
And its disposition is recorded as accepted-but-invalid, distinct from accepted.

**Scenario: A consumer reads a dead shape to avoid before assembling**
Given an assembler about to build a change set that touches a circle's own governance
When it consults the record first
Then it finds the accepted shape to use and the dead shape to avoid
And it needs no refused round-trip against the server to learn either.

### Error scenarios

**Scenario: A recorded fact would duplicate the published contract**
Given a candidate fact describing a change-type shape the v5 contract already enumerates
When the record is maintained
Then the shape is expressed as a citation of the contract, not a restated copy
And a restated duplicate counts as a defect to remove, not an acceptable redundancy.

**Scenario: A recorded fact would read as spec-authoritative**
Given a recorded shape observed only empirically from live server behavior
When the record presents it
Then it is marked as an empirical observation, not as v5-contract reference
And an unmarked fact that reads as contract counts as a defect to correct.

### Edge cases

**Scenario: The published contract absorbs a recorded fact**
Given a shape currently recorded here as an empirical fact
When a spec refresh publishes that shape in the contract
Then the fact retires from the record
And its supersession is recorded, leaving the shape carried only by the contract.

**Scenario: "Created" is not "valid"**
Given the accepted-but-invalid self-target shape
When a consumer reads its disposition
Then the record distinguishes that the server returned a created proposal id from whether that proposal is valid
And it does not let a returned id be read as a successful governance change.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The record carries only the residual shapes**
Given the landed record
When it is read end to end
Then the only shapes it records as empirical facts are the own-circle policy shape and the self-target accepted-but-invalid shape
And every reference to the enumerated change types or the nested-only rule is a citation of the contract, not a recorded shape.

**Scenario: No local judgment leaks into the record**
Given the landed record and its two consumers
When the record is inspected for what it causes
Then nothing in it rejects, filters, or pre-validates a change set before the server sees it
And its only effect is to inform what a consumer assembles.

**Scenario: The provisional source is retired, not copied**
Given the facts previously held in `LEARNINGS.md`
When this capability lands
Then those facts live in the new record
And the `LEARNINGS.md` copy is superseded via `/score:deprecate` rather than left as a second source.

---

## Assumptions

- **Two shapes, not more**: The residual grammar the contract does not carry is assumed to be exactly the own-circle policy shape (F5) and the self-target accepted-but-invalid shape (F6), because the spec refresh (S5) published the type enum and the nested-only rule that had been the other recorded shapes. (Informed by `LEARNINGS.md` 2026-08-05, S5.)
- **Symptom is part of the fact**: Each shape is assumed to be recorded with the observable symptom of misusing it, because the value of the record is teaching a consumer how a mistake presents, not just the correct shape. (Informed by the FEATURE-MODEL entry: "each with the symptom it produces.")
- **[ASSUMED] Physical home deferred**: The record's file location and format are assumed to be a plan/shape decision, not fixed here — the behavioral requirement is a single retirement-disciplined source, however it is materialized. (Confirmed with the developer during specification.)

---

## Ambiguity Warnings

_None. The two open decisions — the record's physical home/format, and the drift-detection mechanism that triggers retirement — are technical, deferred to `/score:plan` and to a separate spec-drift capability respectively; neither is a behavioral gap._
