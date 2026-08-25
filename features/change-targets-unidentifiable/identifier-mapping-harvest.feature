# Source: 080-identifier-mapping-harvest — Scenario: An accountability's number is recovered from inside a role change

Feature: Identifier Mapping Harvest
  A proposal's change can only name an existing governance element by a legacy
  numeric identifier, and for accountabilities, domains, and policies no read
  exposes that number — the legacy-identifier request reaches roles, actors,
  the tree, and the identity read only. What does carry the numbers is the
  organization's own proposal history: `glassfrog proposal identifiers` walks
  it and projects every numeric identifier the change payloads carry into a
  typed mapping with labels and provenance — a derived document, assembled by
  the CLI because it exists in no API response, live and stateless on every
  run. This file covers what the harvest recovers and what an entry carries;
  its sibling identifier-mapping-completeness.feature covers how the walk
  bounds the mapping. The 075 siblings in this directory cover the read-route
  complement.
  (affects: Practitioner, AI agent)

  Rule: A change target's number is recoverable from the organization's own proposal history
    # In order to name an accountability or domain as a change target without
    # a number that no read will give me,
    # as an AI agent drafting a proposal on a practitioner's behalf,
    # I want to obtain the numeric identifiers the organization's own proposal
    # history has already recorded.

    # Source: 080-identifier-mapping-harvest — Scenario: An accountability's number is recovered from inside a role change
    @wip
    Scenario: A nested accountability change yields a typed accountability entry
      Given a complete connection context with a stored token
      And the proposal history carries proposal "prp_0123abcd" whose UpdateRole change nests an UpdateAccountability change with databaseId 31763476 described as "Delivering the platform roadmap"
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry an entry with element_type "accountability" and number 31763476
      And that entry will carry the label "Delivering the platform roadmap" from key "description" marked recognized
      And no entry will type 31763476 as a role

    # Source: 080-identifier-mapping-harvest — Scenario: An identifier that cannot be typed is kept, not guessed at
    @wip
    Scenario: An untypeable identifier is kept with its element type reported as unknown
      Given a complete connection context with a stored token
      And the proposal history carries a change of type "MoveItems" with a key "widgetDatabaseId" holding 55512345
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry an entry with element_type "unknown" and number 55512345
      And that entry's origins will name key "widgetDatabaseId" and change type "MoveItems"
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: One number seen as two element types stays two entries
    @wip
    Scenario: One number seen as two element types yields two entries
      Given a complete connection context with a stored token
      And the proposal history carries the number 97233 once under "circleDatabaseId" and once under "roleDatabaseId"
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry an entry with element_type "circle" and number 97233
      And the mapping will carry an entry with element_type "role" and number 97233
      And neither entry will carry the other's labels

    # Source: 080-identifier-mapping-harvest — Scenario: A number written as text and as a number is one identifier
    @wip
    Scenario: String and integer spellings of one identifier merge into one entry
      Given a complete connection context with a stored token
      And the proposal history carries an UpdateRole change with databaseId "11079492" as text and a later UpdateRole change with databaseId 11079492 as a number
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry exactly one entry with element_type "role" and number 11079492
      And that entry will carry the labels from both changes

    # Source: 080-identifier-mapping-harvest — Scenario: A value that is not a whole number is excluded and counted, never truncated
    @wip
    Scenario: A non-whole-number value is excluded and counted, never truncated
      Given a complete connection context with a stored token
      And the proposal history carries an UpdateRole change with databaseId 12345.5
      When an agent runs "glassfrog proposal identifiers -o json"
      Then no entry will carry the number 12345
      And the mapping will carry no entry for that change's identifier
      And the document will report "complete" as false with "unrepresentable_identifiers" as 1
      And the command will exit with code 0

    # Source: 080-identifier-mapping-harvest — Scenario: No harvested number is ever sent back to the API
    @validation @wip
    Scenario: No harvested number is ever sent back to the API
      Given a complete connection context with a stored token
      And the proposal history carries an UpdateRole change with databaseId 14067864
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the only requests sent will read the proposals endpoint
      And no request path or query will carry "14067864"

    # Source: 080-identifier-mapping-harvest — Scenario: The proposal list read is unchanged
    @validation @wip
    Scenario: The proposal list read is unchanged by the harvest
      Given a complete connection context with a stored token
      And several proposals are visible to the caller
      When an agent runs "glassfrog proposal list"
      Then the output will carry no identifier-mapping fields
      And "glassfrog proposal list --help" will offer no harvest option

  Rule: Every name an identifier ever carried stays visible
    # In order to recognise an element whose name has changed since it was
    # last proposed on,
    # as an AI agent resolving a change target,
    # I want to see every name an identifier ever carried, not only its
    # current one.

    # Source: 080-identifier-mapping-harvest — Scenario: An identifier carries every name it ever had
    @wip
    Scenario: A renamed identifier carries every name it ever had
      Given a complete connection context with a stored token
      And the proposal history named the number 14035882 as "Cloud Platform" in one proposal and as "Cloud Estate" in a later one
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the entry for number 14035882 will carry both the labels "Cloud Platform" and "Cloud Estate"
      And neither label will be marked as the current name

    # Source: 080-identifier-mapping-harvest — Scenario: Text under an unrecognised key is kept as a tagged label
    @wip
    Scenario: Text under an unrecognised key survives as a label tagged with its key
      Given a complete connection context with a stored token
      And the proposal history carries an UpdateDomain change with databaseId 31775269 whose only text sits under the key "rationale"
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the entry for number 31775269 will carry that text as a label from key "rationale" not marked recognized
      And the entry will not be unlabelled

    # Source: 080-identifier-mapping-harvest — Scenario: An identifier no change ever named is still in the mapping
    @wip
    Scenario: An identifier no change ever named appears without a label
      Given a complete connection context with a stored token
      And the proposal history carries an ElectRoleFiller change holding "roleDatabaseId" 14011248 and no text beside it
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the mapping will carry an entry for number 14011248 with an empty labels array
      And no label will be synthesized for it

    # Source: 080-identifier-mapping-harvest — Scenario: A reader can tell an unlabelled identifier from an untypeable one
    @validation @wip
    Scenario: An unlabelled identifier and an untypeable identifier are distinguishable
      Given a complete connection context with a stored token
      And the proposal history carries a role identifier no change ever named and an identifier under an unrecognised key
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the unlabelled entry will carry its element type and an empty labels array with no origins field
      And the untypeable entry will carry element_type "unknown" with its origins named
      And neither entry will be reported as the other

  Rule: A name can be traced to the proposal that used it
    # In order to check for myself that a number is the element I think it is,
    # as a practitioner whose governance work the CLI serves,
    # I want to see the most recent proposal that used each name, so I can
    # read the change that used it.

    # Source: 080-identifier-mapping-harvest — Scenario: An identifier carries every name it ever had
    @wip
    Scenario: Each label names the most recent proposal that carried it and how many did
      Given a complete connection context with a stored token
      And three proposals named the number 14029793 as "Cloud Services", the most recently created being "prp_4567cdef"
      When an agent runs "glassfrog proposal identifiers -o json"
      Then the label "Cloud Services" on the entry for 14029793 will carry last_proposal_id "prp_4567cdef"
      And that label will carry a proposal_count of 3
