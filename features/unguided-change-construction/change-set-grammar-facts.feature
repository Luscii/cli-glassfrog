# Source: 072-change-set-grammar-facts — Scenario: The own-circle policy shape is recorded with its symptom

Feature: Change-Set Grammar Facts
  Building a proposal means hand-writing each governance change command as
  free-form JSON, and the write path's first live exercise surfaced two shapes
  the published v5 contract still does not carry — one of which lets the CLI
  report success for a proposal the server has already marked dead. Change-Set
  Grammar Facts is the single recorded source for exactly that residue: a
  hand-authored record owned by the proposal-drafting skill at
  `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`,
  carrying the own-circle policy shape (CSG-1) and the self-targeting
  accepted-but-invalid role update (CSG-2), each with its disposition and the
  symptom of misuse, marked empirical throughout, and citing the published
  contract where it already speaks.

  This file covers what the record says and how a consumer reads it. Its sibling
  `change-set-grammar-guard.feature` covers what the `internal/build` guard
  enforces about it.
  (affects: Practitioner, AI agent)

  Rule: Read the accepted shape and the dead shape before assembling, not by refused round-trips
    # In order to assemble a valid change set without discovering the two
    # undocumented shapes by refused round-trips,
    # as an AI agent drafting a proposal,
    # I want to read the accepted shape and the dead shape — each with its
    # symptom — before I build the changes[] array.

    # Source: 072-change-set-grammar-facts — Scenario: The own-circle policy shape is recorded with its symptom
    @wip
    Scenario: Own-circle policy shape is recorded with its symptom
      Given the record carried the fact "CSG-1"
      When the record is consulted for changing a circle's own policy
      Then it will state the change as a top-level "CreatePolicy" part with no "UpdateRole" wrapper
      And its Symptom field will state that a wrapped shape is refused while the web UI generates exactly the top-level form

    # Source: 072-change-set-grammar-facts — Scenario: The accepted-but-invalid shape is recorded with its disposition
    @wip
    Scenario: Self-targeting role update carries the accepted-but-invalid disposition
      Given the record carried the fact "CSG-2"
      When the record is consulted for a role update that self-targets the circle from inside its own governance
      Then it will state that the server accepts the create but returns the proposal invalid with a blocking alert and no available transitions
      And its Disposition field will read "accepted-but-invalid" rather than "accepted"

    # Source: 072-change-set-grammar-facts — Scenario: A consumer reads a dead shape to avoid before assembling
    @wip
    Scenario: An assembler finds both the shape to use and the shape to avoid
      Given an assembler was about to build a change set touching a circle's own governance
      When it consults the record before assembling
      Then it will find the accepted shape to use in "CSG-1"
      And it will find the dead shape to avoid in "CSG-2"
      And no round-trip against the server will be needed to learn either

    # Source: 072-change-set-grammar-facts — Scenario: "Created" is not "valid"
    @wip
    Scenario: A returned proposal id is not read as a valid governance change
      Given the record carried the fact "CSG-2"
      When a consumer reads the fact's disposition
      Then the record will distinguish the server returning a created prp_ id from the proposal being valid
      And it will not let a returned id read as a successful governance change

  Rule: Build the downstream consumers against one authority, not scattered notes
    # In order to build the downstream grammar reference and typed change
    # builders against one authority instead of scattered notes,
    # as a developer implementing the write-path fidelity solutions,
    # I want to work from a single recorded source that says exactly which
    # residual shapes are verified and which are merely published contract.

    # Source: 072-change-set-grammar-facts — Scenario: A recorded fact would duplicate the published contract
    @wip
    Scenario: Contract-carried shapes appear as citations, never restatements
      Given the published contract carried the change-type enum at "ProposalChange.properties.type.enum"
      When the record names the enumerated change types or the nested-only rule
      Then it will cite the contract anchor by schema and property name
      And it will restate no enum values beyond its citation lists

    # Source: 072-change-set-grammar-facts — Scenario: A recorded fact would read as spec-authoritative
    @wip
    Scenario: Every fact is marked empirical, never contract
      Given the record's facts were observed only from live server behavior
      When the record is checked for its empirical marker
      Then a leading marker will state that every fact is observed behavior and none is part of the published contract
      And a record missing that marker will fail the guard

    # Source: 072-change-set-grammar-facts — Proposed: interface-spec.md per-fact contract (five required fields)
    @wip
    Scenario: Every recorded fact carries its full five-field contract
      Given the landed record
      When each fact section is read
      Then every fact will carry all five required fields
      And its Evidence will name the live proposal the fact was verified against
      And its Provenance will name the LEARNINGS entry it supersedes

  Rule: Retire a fact the moment the published contract absorbs it
    # In order to keep the record honest as the API moves,
    # as the maintainer,
    # I want to retire a recorded fact the moment the published contract
    # absorbs it, rather than letting two copies drift.

    # Source: 072-change-set-grammar-facts — Scenario: The published contract absorbs a recorded fact
    @wip
    Scenario: A contract-absorbed fact retires from the record
      Given a shape was recorded as an empirical fact
      When a spec refresh publishes that shape in the contract
      Then the fact will be deleted from the record rather than marked retired in place
      And its supersession will be recorded so the shape is carried only by the contract

  # --- Validation scenarios ---

  # Source: 072-change-set-grammar-facts — Scenario: The record carries only the residual shapes
  @validation @wip
  Scenario: Only the two residual shapes are recorded as empirical facts
    Given the landed record was read end to end
    When its fact sections are enumerated
    Then exactly "CSG-1" and "CSG-2" will be recorded as empirical facts
    And every mention of the enumerated change types or the nested-only rule will be a citation of the contract

  # Source: 072-change-set-grammar-facts — Scenario: No local judgment leaks into the record
  @validation @wip
  Scenario: Nothing in the feature rejects a change set before the server sees it
    Given the landed record and its guard were inspected for what they cause
    When a change set is assembled against the record's knowledge
    Then nothing shipped by this feature will reject, filter, or pre-validate that change set locally
    And the record's only effect will be to inform what a consumer assembles

  # Source: 072-change-set-grammar-facts — Scenario: The provisional source is retired, not copied
  @validation @wip
  Scenario: The LEARNINGS copy is superseded, not left as a second source
    Given the facts previously held in the LEARNINGS provisional entry
    When this capability lands
    Then the facts will live in the record
    And the supersession of the LEARNINGS copy will be recorded in the deprecation log
