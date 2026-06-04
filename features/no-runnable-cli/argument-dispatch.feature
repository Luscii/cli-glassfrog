Feature: Argument Dispatch
  Part of the No Runnable CLI problem: no project skeleton or command framework
  exists to build any command on. Argument Dispatch parses the invocation
  arguments and routes them to the correct registered command, failing unknown
  or invalid invocations with guidance. (affects: AI agent, operator)

  Rule: Routing a typed invocation to its command
    # In order to run a registered command,
    # as an AI agent acting for a practitioner,
    # I want to type its full path and have it routed and executed.

    # Source: 002-argument-dispatch — Scenario: Route a nested leaf command
    Scenario: A nested path routes to its leaf command
      Given a "roles" group with a "list" subcommand is registered
      When the caller invokes "glassfrog roles list"
      Then dispatch will route to the "list" command
      And the "list" command's action will run

    # Source: 002-argument-dispatch — Scenario: Route a top-level leaf command
    Scenario: A top-level name routes to its command
      Given a "version" command is registered at the top level
      When the caller invokes "glassfrog version"
      Then dispatch will route to "version"
      And its action will run

    # Source: 002-argument-dispatch — Scenario: A prefix does not resolve to a longer command
    Scenario: A prefix does not resolve to a longer command
      Given "roles" is the only registered command beginning with "ro"
      When the caller invokes "glassfrog ro list"
      Then dispatch will not route to "roles"
      And it will report an unknown-command error naming "ro"

    # Source: 002-argument-dispatch — Scenario: No routing depends on abbreviation
    @validation @wip
    Scenario: No routing depends on abbreviation
      Given the Argument Dispatch specification
      When every routing scenario is read
      Then each will name a full, exact command path with none relying on a prefix

    # Source: 002-argument-dispatch — Scenario: Specification names no implementation technology
    @validation @wip
    Scenario: Dispatch specification names no implementation technology
      Given the Argument Dispatch specification text
      When it is scanned for technology names
      Then no programming language, framework, or data-structure choice will appear

  Rule: Unknown or invalid invocations fail with guidance
    # In order to recover quickly from a typo,
    # as an operator,
    # I want an unknown command to tell me which token wasn't recognized and how to get help.

    # Source: 002-argument-dispatch — Scenario: Unknown top-level command
    Scenario: An unknown top-level command fails with guidance
      Given no command named "rolez" is registered
      When the caller invokes "glassfrog rolez"
      Then dispatch will report an unknown-command error naming "rolez"
      And it will point the caller to help
      And it will classify the outcome as a usage error

    # Source: 002-argument-dispatch — Scenario: Unknown subcommand under a known group
    Scenario: An unknown subcommand under a known group fails with guidance
      Given a "roles" group with a "list" subcommand is registered
      When the caller invokes "glassfrog roles lst"
      Then dispatch will report an unknown-command error naming "lst"
      And it will classify the outcome as a usage error

    # Source: 002-argument-dispatch — Scenario: Unexpected flag is rejected
    Scenario: An unexpected flag is rejected as a usage error
      Given a "roles list" command is registered
      When the caller invokes "glassfrog roles list --bogus"
      Then dispatch will report a usage error naming the unexpected "--bogus"
      And the "list" command will not run

    # Source: 002-argument-dispatch — Scenario: Each excluded concern names its owner
    @validation @wip
    Scenario: Each dispatch non-behavior names its owning capability
      Given the Non-Behaviors section of the Argument Dispatch specification
      When each non-behavior is read
      Then it will name the capability that owns the excluded concern

  Rule: A group or root invocation surfaces its subcommands
    # In order to discover what a group offers,
    # as an operator,
    # I want to type the group name alone to surface its available subcommands.

    # Source: 002-argument-dispatch — Scenario: Bare group surfaces its subcommands
    Scenario: A bare group invocation surfaces its subcommands
      Given a "roles" group with "list" and "get" subcommands is registered
      When the caller invokes "glassfrog roles" with no further token
      Then dispatch will resolve to the "roles" group
      And route to a help outcome listing "list" and "get"
      And the outcome will be a success

    # Source: 002-argument-dispatch — Scenario: Empty invocation resolves to root help
    Scenario: An empty invocation resolves to root help
      Given any registered command set
      When the caller invokes "glassfrog" with no tokens
      Then dispatch will resolve to the root
      And route to a help outcome
      And the outcome will be a success
