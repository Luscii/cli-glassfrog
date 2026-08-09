# Source: 075-legacy-identifier-request — Scenario: The option exists on exactly the supported reads and nowhere else

Feature: Legacy Identifier Coverage Guard
  The legacy number is a declared transition bridge that retires with the v3
  API, and neither the vendored contract's version field nor its changelog
  signals drift — only diffing the file does. This guard is the capability's
  retirement tripwire: an internal/build test that derives the set of
  operations referencing the IncludeLegacyId parameter from the vendored spec,
  derives the set of command leaves registering --legacy-id from the live
  command tree, and fails CI when either side disagrees with the declared
  operation-to-command mapping. Retirement and widening both fail at the
  refresh PR instead of going unnoticed.

  This file covers what the guard enforces. Its siblings
  legacy-identifier-request.feature and legacy-identifier-absence.feature cover
  the CLI behavior itself.
  (affects: Practitioner, AI agent)

  Rule: Avoid building a durable dependence on a facility that will be withdrawn

    # In order to avoid building a durable dependence on a facility that is
    # going to be withdrawn,
    # as a maintainer of the CLI,
    # I want to have the number's opt-in, nullable, and time-limited nature
    # stated where an operator meets it.

    # Source: 075-legacy-identifier-request — Scenario: The option exists on exactly the supported reads and nowhere else
    @validation @wip
    Scenario: The flag exists on exactly the supported reads and nowhere else
      Given the CLI's full command surface
      When every read command's options are enumerated
      Then exactly the roles, tree, actors and me leaves will offer --legacy-id
      And no other command will offer it, including the me subcommands, subroles and fillers

    # Source: 075-legacy-identifier-request — Scenario: The retirement clock is stated where an operator meets the option
    @validation @wip
    Scenario: The retirement clock is stated in the flag's own help text
      Given the help output of a read that supports --legacy-id
      When a reader consults it
      Then the flag's help will state the number is a temporary transition aid that will be withdrawn
      And it will not present the number as a stable or permanent identifier

    # Source: 075-legacy-identifier-request — Proposed: plan ADR-4 / interface-spec.md invariant 1 (spec-side set equality)
    @wip
    Scenario: The guard derives both sides and fails on a vendored-contract change
      Given the guard derived the IncludeLegacyId operation set from the vendored spec
      And it derived the registration set from the live command tree
      When either derived set disagrees with the declared operation-to-command mapping
      Then the guard will fail naming the disagreeing operation or leaf
      And its failure message will name a remedy that can pass its sibling invariants

    # Source: 075-legacy-identifier-request — Proposed: interface-spec.md invariant 4 (shared help constant)
    @wip
    Scenario: The guard anchors the shared help constant's retirement property
      Given the four registrations share one help-text constant
      When the guard checks each registration's usage string
      Then it will fail if any registration diverges from the shared constant
      And it will fail if the constant loses the transition or retirement property it must convey

    # Source: 075-legacy-identifier-request — Proposed: interface-spec.md invariant 5 (observed-exception register)
    @wip
    Scenario: A newly mapped operation whose schema omits the field fails until it is probed
      Given an operation was added to the mapping whose response schema declares no legacy_id
      And no observed exception was registered for it
      When the guard evaluates response-schema capability
      Then it will fail naming that operation
      And its message will require probing the operation before either correcting the mapping or registering the exception

    # Source: 075-legacy-identifier-request — Proposed: interface-spec.md invariant 5 (stale register entry)
    @wip
    Scenario: A registered exception for an unmapped operation fails as stale
      Given the tree read's schema exception was registered with its probe evidence
      When that operation is removed from the mapping
      Then the guard will fail naming the exception as stale
      And the guard will pass while the operation remains mapped with its evidence recorded
