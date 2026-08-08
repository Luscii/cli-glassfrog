# Source: 073-circle-routing-rule — Proposed: plan ADR-3 composed-surface widening and ADR-4 premise tripwire

Feature: Circle Routing Rule — guard enforcement
  The routing record only helps an assembler for as long as its premise holds.
  That premise is an absence: `proposal create` carries no circle parameter, so
  the anchor determines the circle. If a refresh ever adds one, the rule is not
  merely stale but wrong — a consumer following it would route to a circle that
  cannot decide the change. The vendored spec gives no usable drift signal, so a
  best-effort `internal/build` guard holds the record to its contract, deriving
  every side from its file: the create request's whole property set against the
  premise, the cited `Role` fields against the schema, the record's required
  sections and field labels against the record, and the reads the record names
  against both the shipped CLI and the drafting path's composed-leaf registry.
  The guard hard-codes no read names, no property sets, and no field values.

  Because the composed surface widens in this spec rather than in the
  consultation that uses it, the registry and the drafter agent's fence gain the
  three routing reads here — and this file is where that agreement is pinned.
  (affects: AI agent, Maintainer)

  Rule: Build the pre-assembly gate against one routing authority
    # In order to build the pre-assembly gate against one routing authority
    # instead of re-deriving the rule,
    # as a developer implementing the gate that consumes it,
    # I want to work from a single recorded source that states the rule, its
    # own-circle consequence, the Circle Lead exception, and the reads that
    # answer it.

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 8 (premise tripwire)
    Scenario: A circle parameter on the create request fails the build
      Given the record's premise cited "CreateProposalRequest.properties.proposal" as carrying no circle property
      When a spec refresh adds any property to that object beyond "tension_id" and "changes"
      Then the guard will fail naming both property sets so the addition is readable from the failure
      And the message will name re-deriving the rule or retiring the record as the two resolution paths

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 9 (classification anchors)
    Scenario: A dropped Role field fails at the section citing it
      Given the record cited "Role.has_subroles" as the circle indicator and "Role.parent_role_id" as the root signal
      When a spec refresh drops either field from the Role schema
      Then the guard will fail naming the missing field and the section citing it
      And the message will name re-deriving the citation or retiring the record as the two resolution paths

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 5 (named-read integrity)
    Scenario: A named read the CLI no longer exposes fails the build
      Given the record named "tension list" in its named-reads block
      When the CLI drops or renames that subcommand
      Then the guard will fail naming the leaf and which surface was searched
      And the record will not be able to name a read the CLI no longer exposes

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 6 (unanchorable command path)
    Scenario: An unanchorable command path is reported rather than skipped
      Given the record named a read carrying a command path of three tokens
      When the guard resolves the named-reads block
      Then it will fail naming the leaf and the supported forms
      And it will not silently skip the leaf it cannot anchor

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 7 (record ↔ registry agreement)
    @wip
    Scenario: A named read missing from the composed registry fails the build
      Given the record named "me roles" in its named-reads block
      When that leaf is absent from the drafting path's composed-leaf registry
      Then the guard will fail naming the leaf and the registry path
      And the record will not be able to name a read the path is forbidden to run

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md conditions 1, 2, 4 (structural invariants)
    Scenario Outline: A structurally incomplete record fails the guard
      Given a record missing <element>
      When the guard evaluates the record
      Then it will fail naming <named>
      And the message will name supplying the missing element as the resolution path

      Examples:
        | element                | named                           |
        | a required section     | the missing section             |
        | a required field label | the section and the field label |
        | the named-reads block  | that the record declares no reads |

    # Source: 073-circle-routing-rule — Proposed: interface-spec.md condition 3 (empirical marker)
    Scenario: A record without its empirical marker fails the build
      Given the record's landing behaviour was observed rather than published
      When the record is checked for its leading marker
      Then the marker will state that the absent circle parameter is contract while the landing is observed behaviour
      And a record missing that marker will fail the guard

    # Source: 073-circle-routing-rule — Proposed: plan ADR-3 gated-membership posture preserved
    @wip
    Scenario: Widening the composed surface leaves the gate posture unchanged
      Given the three routing reads joined the drafting path's composed surface
      When the gated-membership invariant is checked
      Then "proposal create" will remain the only composed leaf in the write-safety gated registry
      And every other composed leaf will remain absent from it

    # Source: 073-circle-routing-rule — Proposed: plan ADR-5 retirement is whole-record
    @wip
    Scenario: Premise dissolution retires the whole record, not one field
      Given the record's premise was dissolved by a circle parameter appearing on the create request
      When the retirement is carried out
      Then the record will be deleted rather than edited to keep its consequences
      And the supersession will be recorded in the deprecation log naming the spec revision that dissolved it

  # --- Validation scenarios ---

  # Source: 073-circle-routing-rule — Scenario: The composed surface matches what the content names
  @validation @wip
  Scenario: The three routing reads appear in both the registry and the agent fence
    Given the reads the recorded procedure names
    When the composed-leaf registry and the drafter agent artifact are read
    Then every named read will appear in both, and each will still resolve as a command the CLI exposes
    And no write leaf will have entered the composed surface alongside them

  # Source: 073-circle-routing-rule — Scenario: The content ships unconsulted
  @validation @wip
  Scenario: No workflow step consults the record or runs its reads to route
    Given this capability landed on its own
    When the surfaces around the record are inspected
    Then no workflow step will consult the routing content or run its named reads to route
    And the drafting path's workflow will be unchanged

  # Source: 073-circle-routing-rule — Scenario: Nothing prescribed refuses a write
  @validation @wip
  Scenario: Nothing the feature ships can refuse a change set locally
    Given the landed record and its guard were inspected for what they cause
    When a proposal is created against the record's knowledge
    Then nothing shipped by this feature will reject, filter, or pre-validate that write locally
    And the record's only effect will be to inform what an assembler anchors on

