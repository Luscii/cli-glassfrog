# Source: 072-change-set-grammar-facts — Proposed: plan ADR-3 citation-integrity tripwire on spec refresh

Feature: Change-Set Grammar Facts — guard enforcement
  The change-set grammar record only helps an assembler for as long as it stays
  true: its citations must still match the published contract, its facts must
  keep their required shape, and a fact absorbed by the contract must actually
  leave. None of that is self-maintaining, and the vendored spec gives no usable
  drift signal — `info.version` did not move across a refresh that added fifteen
  operations. A best-effort `internal/build` guard therefore holds the record to
  its contract, deriving every side from its file: the record's declared
  `Live facts` manifest against its actual fact sections, each fact's required
  fields and closed disposition vocabulary, the empirical marker, and every
  contract citation against `spec/glassfrog-api-v5.yaml`. It hard-codes no enum
  values, no type names and no fact ids — which is what lets a deliberate
  retirement pass while a partial one fails.

  This file covers what the guard enforces. Its sibling
  `change-set-grammar-facts.feature` covers what the record says and how a
  consumer reads it.
  (affects: Practitioner, AI agent)

  Rule: Build the downstream consumers against one authority, not scattered notes
    # In order to build the downstream grammar reference and typed change
    # builders against one authority instead of scattered notes,
    # as a developer implementing the write-path fidelity solutions,
    # I want a single recorded source that says exactly which residual shapes
    # are verified and which are merely published contract.

    # Source: 072-change-set-grammar-facts — Proposed: plan ADR-3 citation-integrity tripwire on spec refresh
    @wip
    Scenario: A spec refresh that moves a cited anchor fails the build
      Given the record's nested-only citation list matched the vendored spec's nested-only set
      When a refreshed spec no longer carries the same nested-only set
      Then the guard will fail the build naming both sets
      And the failure will demand re-derivation or retirement before the change lands

    # Source: 072-change-set-grammar-facts — Proposed: interface-spec.md Error Communication conditions 4–5
    @wip
    Scenario Outline: A structurally invalid fact fails the guard
      Given a record carrying the fact "CSG-1"
      When <defect> is introduced
      Then the guard will fail naming the fact "CSG-1"
      And it will name <detail>

      Examples:
        | defect                            | detail                    |
        | an empty Evidence field           | the offending field label |
        | a Disposition of "probably-fine"  | the unrecognized value    |

  Rule: Retire a fact the moment the published contract absorbs it
    # In order to keep the record honest as the API moves,
    # as the maintainer,
    # I want to retire a recorded fact the moment the published contract
    # absorbs it, rather than letting two copies drift.

    # Source: 072-change-set-grammar-facts — Proposed: plan ADR-3/ADR-4 manifest invariant admits a complete retirement
    @wip
    Scenario: A complete retirement passes the guard
      Given a record whose manifest declared "CSG-1, CSG-2"
      When a contract-absorbed fact is retired by deleting its section and dropping its id from the manifest
      Then the guard will pass
      And the surviving fact will still be recorded

    # Source: 072-change-set-grammar-facts — Proposed: interface-spec.md Error Communication conditions 1–2
    @wip
    Scenario Outline: A partial retirement fails the guard
      Given a record carrying the fact "CSG-1" in both its manifest and its fact sections
      When only <removed half> is removed
      Then the guard will fail naming "CSG-1"
      And the failure will name the resolution path

      Examples:
        | removed half       |
        | the fact section   |
        | the manifest entry |

    # Source: 072-change-set-grammar-facts — Proposed: interface-spec.md Error Communication condition 3
    @wip
    Scenario: A record with no facts left fails as an empty shell
      Given every recorded fact had been absorbed by the published contract
      When the record is left in place with an empty manifest and no fact sections
      Then the guard will fail reporting the record as empty
      And the failure will direct the maintainer to delete the record rather than keep a shell
