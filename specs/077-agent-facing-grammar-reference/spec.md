# Specification: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Building a proposal means assembling a `changes[]` array of governance commands, and the knowledge of what is sayable in that array is split three ways: the published v5 contract carries the change-type vocabulary and the rule that accountability/domain edits ride nested inside a role operation; the residual shapes the contract does not carry live in the recorded grammar facts; and per-type field guidance exists nowhere, because the contract defines no per-type field schema. Today none of this is served at the point of assembly — the record ships as a file inside one skill of the operating surface, is consulted by nothing, and cites a contract file that exists only in the development repository. An assembler on an operator's machine still supplies the grammar from prior knowledge.

Agent-Facing Grammar Reference closes that gap by putting the grammar in the artifact every consumer already has: a **dedicated informational read command on the CLI** that renders the change-set grammar — every contract-enumerated change type with its placement class, drift-guarded against the vendored contract, plus the empirical residue with the symptom each wrong shape produces, every shape marked by its provenance. The command **informs and never validates**: it makes no API request, accepts nothing to judge, and leaves the server the sole judge of validity (VISION Exclusion 2). It is consulted before a change set is built; pointing the drafting workflow at it is the separate consultation capability's job.

---

## Behavioral Accord

### Content

- When the command runs, it renders every change type the published contract enumerates, each classified by where it may appear in a change set: top-level, or nested-only as the child of a role operation.
- When the command renders the empirical residue, each fact carries the shape to use, its explicit disposition, and the observable symptom of getting it wrong — the symptom being where the wrong shape is named, as the recorded grammar facts state it. The disposition is drawn from the record's own closed vocabulary and reproduced verbatim, so "the server took it" is never conflated with "the server considers it valid"; the accord does not fix which values that vocabulary holds.
- When any shape renders, it carries its provenance — **published contract** or **empirical observation** — so a consumer can always tell which standing a shape has.
- When the record gains or retires a fact, the rendered reference follows it; the command renders what is known and never carries per-type field guidance beyond what the record verifies.

### Conduct

- When the command runs, it makes no API request; the reference renders identically with or without network access.
- When no credential is configured, the command still renders in full — consulting the grammar precedes, and never requires, auth setup.
- When an output format is selected, the reference renders in it the way other reads do, so an agent parses it with the same machinery it parses every read with.
- When the command is invoked, it returns the whole reference in one invocation — no state, no partial views required to see the full picture.

### Sync

- When the vendored contract's change-type vocabulary changes — a type added, removed, or renamed, or the nested-only membership shifting — and the rendered reference no longer matches it, the repository's merge-gating verification run fails, naming the divergence; the rendered vocabulary cannot silently drift from the contract.
- When the empirical residue renders, it renders from the single recorded source of grammar facts; no second copy of a fact's text exists to drift from the record.

---

## User Scenarios

**In order to** assemble a valid `changes[]` without prior knowledge of each command's shape or refused round-trips against the server,
**as an** AI agent drafting a proposal,
**I want to** run one read that gives me the sayable change types, where each may appear, and the known dead shapes with their symptoms — before I build.

**In order to** know how far to trust each shape when the server changes underneath it,
**as an** AI agent,
**I want** every rendered shape to carry its provenance, so a contract-published shape and a verified observation are never confused.

**In order to** consult the grammar on any machine that has the CLI — before credentials are set up, with or without the operating surface installed,
**as a** practitioner (or the agent acting for one),
**I want** the reference served by the binary itself rather than by a file only one install layout carries.

---

## Non-Behaviors

- The command must not accept a change set to judge — no argument or flag of its own takes one, and an attempt fails as a usage error, never as a validity verdict. The output-template source every read inherits is not an exception: it renders a caller's template and evaluates nothing, so it is not a change-set input path. **Why**: local governance judgment is VISION Exclusion 2; the server is the single judge, and the step from "renders the shapes" to "checks your shape" is exactly the erosion this boundary exists to stop.
- The command must not present an empirical fact as contract-authoritative. **Why**: VISION Principle 1 — nothing outside the published spec may be presented as spec-authoritative; an unmarked fact invites trust past the point the server changes beneath it.
- The command must not invent per-type field guidance the record does not verify. **Why**: the contract defines no per-type field schema, so a field list beyond recorded observation would be a fabricated authority claim; content grows only as the record grows.
- The command must not carry routing or identifier facts. **Why**: inherited from the record's own boundary — where a proposal lands belongs to the circle-routing record, identifier resolution to the identifier capabilities; folding them in blurs what "grammar" means.
- The command must not replace or retire the recorded grammar facts. **Why**: the record stays the single retirement-disciplined source the command renders; a command owning its own fact text would be the second source the record exists to prevent.
- The command must not rewire the drafting path's consultation step. **Why**: pointing the drafting workflow at this command — and surfacing a recognized dead shape before the write — is Pre-Assembly Grammar Consultation's charter; landing it here would ship that gate without its own specification.

---

## Integration Boundaries

- **Change-Set Grammar Facts record**: the single source of the empirical residue. The command renders it; a fact added or retired there changes the rendering with no second edit. Its physical home may move so the binary can carry it — a plan decision; the behavioral requirement is one source, however materialized.
- **Vendored v5 contract (`spec/glassfrog-api-v5.yaml`)**: the source the contract-derived vocabulary and placement classes are derived from, and the drift guard's comparison target. Lives in the development repository — a consumer of the command never needs it.
- **Pre-Assembly Grammar Consultation (downstream)**: the drafting-path capability that runs this command before assembling. Until it lands, the drafting skill's existing reference-file consumption is untouched.
- **Typed Change Builders (downstream)**: scopes its per-type builders to the shapes this reference verifies; reads the same grammar, adds no facts of its own.
- **Output rendering**: the reference renders through the CLI's existing per-invocation format selection; failures follow the CLI's existing failure conventions.
- **Operating-surface write gate**: the gate that stands in front of governance writes must recognize this read as a read, so consulting the grammar never asks the operator to confirm anything. Where the command is placed in the command tree determines whether the gate already recognizes it — a placement inside the gate's fail-close scope requires teaching the gate about the read, in the same change that ships the command.

---

## Driving Scenarios

### Happy path

**Scenario: An assembler reads the grammar before building**
Given an AI agent about to assemble a change set
When it runs the grammar command
Then it receives every contract-enumerated change type with its placement class
And the recorded dead shapes with the symptom each produces
And it needed no API request and no refused round-trip to learn either.

**Scenario: Provenance is visible on every shape**
Given the rendered reference
When a consumer reads any shape in it
Then contract-derived content is marked as published contract
And the residual facts are marked as empirical observation.

**Scenario: The grammar renders without credentials**
Given a machine with the CLI installed and no credential configured
When the grammar command runs
Then the full reference renders
And no API request is attempted.

### Error scenarios

**Scenario: A change set offered for judgment is refused as usage, not judged**
Given an operator who invokes the grammar command with a change set to check
When the command rejects the input
Then the failure is a usage error in the CLI's existing convention
And no verdict on the change set's validity is expressed or implied.

**Scenario: The contract vocabulary drifts from the rendering**
Given a vendored-contract refresh that changes the change-type enum or the nested-only membership
When the repository's merge-gating verification runs
Then it fails, naming the divergence between the contract and the rendered vocabulary
And the build stays red until the rendering follows the contract.

### Edge cases

**Scenario: A recorded fact retires**
Given a shape the contract absorbs in a refresh, retiring its fact from the record
When the reference next renders
Then the retired fact no longer appears as empirical residue
And the shape is carried only by the contract-derived layer.

**Scenario: "Accepted" is not "valid" survives the rendering**
Given the recorded accepted-but-invalid shape
When a consumer reads its rendered disposition
Then accepted-but-invalid is distinct from accepted
And a returned proposal id for that shape reads as a dead draft, not a successful change.

**Scenario: No live residue**
Given a record whose live facts have all retired
When the grammar command runs
Then the contract-derived vocabulary still renders in full
And the reference states that no empirical residue is currently recorded, rather than omitting it silently.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The rendered vocabulary equals the contract's**
Given the landed command and the vendored contract
When the rendered change types and nested-only membership are set-compared against the contract's enum and rule
Then the sets match exactly — no missing, extra, or renamed type on either side.

**Scenario: No judgment path exists**
Given the landed command
When it is inspected for what it can be made to do
Then no invocation evaluates, filters, or scores a change set
And its only effect is rendering knowledge.

**Scenario: One source for the residue**
Given the landed command and the grammar-facts record
When the origin of each rendered empirical fact is traced
Then each renders from the record, through whatever generated projection the binary carries
And no fact's text is hand-maintained outside the record.

---

## Assumptions

- **Two live facts today**: the empirical residue is the own-circle policy shape and the self-target accepted-but-invalid shape, per the record; the reference's content scales as facts enter or retire, with no spec change needed.
- **[ASSUMED] Command name and placement**: deferred to interface design; this spec constrains the command's conduct (read-only, no request, format-aware), not its name. (Confirmed with the developer during specification.)
- **[ASSUMED] Embedding mechanism and record relocation**: how the binary comes to carry the record and the contract-derived vocabulary is a plan decision; the behavioral requirements are the single source and the drift guard. (Confirmed with the developer during specification.)
- **Consultation ordering**: the drafting path keeps reading its shipped reference file until Pre-Assembly Grammar Consultation rewires it to the command; both surfaces render the same record in the interim.

---

## Ambiguity Warnings

_None behavioral. The open decisions — the command's name and placement (interface design), and the record's embeddable home plus the drift guard's mechanism (plan) — are technical, deferred to their stages._
