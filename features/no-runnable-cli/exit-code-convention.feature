Feature: Exit-Code Convention
  Part of the No Runnable CLI problem: no project skeleton or command framework
  exists to build any command on. The Exit-Code Convention gives every command
  outcome a standardized process exit code (success / usage error / runtime
  error) so CI and agents can react without parsing text. (affects: CI pipeline, AI agent, Maintainer)

  Rule: A failed run is distinguishable from success by exit code
    # In order to fail a build whenever the CLI did not fully succeed,
    # as a CI pipeline,
    # I want every non-success outcome to exit non-zero.

    # Source: 004-exit-code-convention — Scenario: A successful command exits zero
    Scenario: A successful command exits zero
      Given a "version" command is registered at the top level
      When the caller invokes "glassfrog version"
      Then the process will exit with code 0

    # Source: 004-exit-code-convention — Scenario: A help/listing outcome exits zero
    Scenario: A help or listing outcome exits zero
      Given a "roles" group with "list" and "get" subcommands is registered
      When the caller invokes "glassfrog roles" with no further token
      Then dispatch will resolve to the "roles" group
      And route to a help outcome listing "list" and "get"
      And the outcome will be a success
      And the process will exit with code 0

    # Source: 004-exit-code-convention — Scenario: An unknown command exits the usage code
    Scenario: An unknown command exits the usage code
      Given no command named "rolez" is registered
      When the caller invokes "glassfrog rolez"
      Then it will classify the outcome as a usage error
      And the process will exit with code 2

    # Source: 004-exit-code-convention — Scenario: An unexpected internal failure never exits zero
    Scenario: An unexpected internal failure never exits zero
      Given a "boom" command whose action fails for a reason matching no known category is registered
      When the caller invokes "glassfrog boom"
      Then the process will exit with code 1
      And it will not exit with code 0

  Rule: Each failure class carries a distinct exit code
    # In order to react correctly to a failure without parsing text,
    # as an AI agent acting for a practitioner,
    # I want each failure class to carry a distinct exit code.

    # Source: 004-exit-code-convention — Scenario: Different failure classes carry different codes
    @validation @wip
    Scenario: Different failure classes carry different codes
      Given a usage-error outcome and a rate-limited outcome
      When each outcome is mapped to its exit code
      Then the usage-error code 2 and the rate-limited code 5 will differ
      And an agent will tell the two failures apart from the exit status alone

    # Source: 004-exit-code-convention — Scenario: A rate-limited request exits the rate-limit code
    @validation @wip
    Scenario: A rate-limited outcome maps to the rate-limit code
      Given an outcome the producer classifies as rate-limited
      When its exit code is determined
      Then the registry will map it to code 5
      And not to the general API-error code 3

    # Source: 004-exit-code-convention — Scenario: The most specific category wins
    @validation @wip
    Scenario: The most specific category determines the code
      Given an outcome classified as a permission failure that is also a general API error
      When its exit code is determined
      Then the registry will map it to the permission code 4
      And not to the general API-error code 3

    # Source: 004-exit-code-convention — Proposed: Panic exits 1, not code 2 (plan ADR-4)
    Scenario: An internal panic exits one and never collides with the usage code
      Given a "boom" command whose action panics is registered
      When the caller invokes "glassfrog boom"
      Then the process will exit with code 1
      And it will not exit with code 2

  Rule: Command outcomes map onto a stable, published code registry
    # In order to add a new command without inventing my own error signalling,
    # as a Maintainer,
    # I want my command's outcomes to map onto the existing code registry.

    # Source: 004-exit-code-convention — Scenario: Codes and categories are one-to-one
    @validation @wip
    Scenario: Codes and categories are one-to-one
      Given the registry of categories and codes
      When each category is matched to its code and back
      Then no two categories will share a code
      And no category will have two codes

    # Source: 004-exit-code-convention — Scenario: No shell-reserved code is assigned
    @validation @wip
    Scenario: No shell-reserved code is assigned
      Given the assigned set of exit codes
      When each is checked against the shell-reserved values 126, 127, and 128 plus N
      Then none of the assigned codes will fall in that reserved range

    # Source: 004-exit-code-convention — Scenario: Adding a category never renumbers existing codes
    @validation @wip
    Scenario: Adding a category never renumbers existing codes
      Given a published set of assigned codes
      When a new outcome category is introduced in the registry
      Then it will take a previously-unused code
      And every existing category will keep the code it had before

    # Source: 004-exit-code-convention — Scenario: Specification names no implementation technology
    @validation @wip
    Scenario: Exit-Code Convention specification names no implementation technology
      Given the Exit-Code Convention specification text
      When it is scanned for technology names
      Then no programming language, framework, or data-structure choice will appear
