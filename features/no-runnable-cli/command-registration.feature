Feature: Command Registration
  Part of the No Runnable CLI problem: no project skeleton or command framework
  exists to build any command on. Command Registration lets the CLI know which
  commands it offers and how they are organized, and rejects malformed
  registrations before users hit them. (affects: Maintainer)

  Rule: Adding a command never disturbs the ones already there
    # In order to add a new command without risking the ones already there,
    # as a Maintainer,
    # I want to register my command in isolation.

    # Source: 001-command-registration — Scenario: Register a top-level leaf command
    Scenario: Registering a leaf command makes it known
      Given the command set was empty
      When a Maintainer registers a "version" leaf with summary "Print the version" and an action
      Then querying the command set for "version" will return that command
      And enumerating the command set will list "version" with its summary

    # Source: 001-command-registration — Scenario: Register unrelated commands independently
    Scenario: Registering a command leaves existing commands untouched
      Given the command set already contained a "roles" group
      When a Maintainer registers a "proposals" group
      Then both "roles" and "proposals" will be present in the command set
      And the "roles" group will be unchanged

    # Source: 001-command-registration — Scenario: No implementation leakage
    @validation @wip
    Scenario: Specification names no implementation technology
      Given the Command Registration specification
      When its text is scanned for technology names
      Then no programming language, framework, or data-structure choice will appear

    # Source: 001-command-registration — Scenario: Each excluded concern names its owner
    @validation @wip
    Scenario: Each non-behavior names its owning capability
      Given the Non-Behaviors section of the specification
      When each non-behavior is read
      Then it will name the sibling capability that owns the excluded concern

  Rule: Commands organize into nested groups
    # In order to mirror the API's resource-and-verb structure,
    # as a Maintainer,
    # I want to register commands inside named groups.

    # Source: 001-command-registration — Scenario: Register a group with subcommands
    Scenario: Registering a group exposes its subcommands by path
      Given the command set was empty
      When a Maintainer registers a "roles" group containing "list" and "get" subcommands
      Then querying the path "roles list" will return the list command
      And querying the path "roles get" will return the get command
      And enumerating the command set will list the "roles" group and both subcommands

    # Source: 001-command-registration — Scenario: Look up a group on its own
    Scenario: A bare group name resolves to the group itself
      Given a "roles" group with "list" and "get" subcommands was registered
      When the command set is queried for "roles" with no further path
      Then the "roles" group node will be returned
      And its "list" and "get" subcommands will be reachable through it

    # Source: 001-command-registration — Scenario: Same subcommand name under different groups
    Scenario: A name is unique only within its own group
      Given a "roles" group and a "proposals" group were registered
      When a Maintainer registers a "get" subcommand under each group
      Then both registrations will succeed
      And "roles get" and "proposals get" will resolve independently

    # Source: 001-command-registration — Scenario: A group nested within a group
    Scenario: Groups nest to arbitrary depth
      Given a "proposals" group was registered
      When a Maintainer registers a "changes" subgroup containing an "add" subcommand under it
      Then querying the path "proposals changes add" will return the add command

    # Source: 001-command-registration — Scenario: Lookup is predictable from registration alone
    @validation @wip
    Scenario: Lookup is predictable from registration alone
      Given only the Command Registration specification
      When a reader registers a "roles" group with a "list" subcommand
      Then they will be able to state that "roles list" resolves to the list command
      And that "roles" alone resolves to the group
      And that enumerating the set lists the "roles" group

  Rule: Malformed registration fails at startup
    # In order to catch mistakes before users hit them,
    # as a Maintainer,
    # I want a duplicate or malformed registration to fail at startup.

    # Source: 001-command-registration — Scenario: Duplicate top-level name
    Scenario: Duplicate sibling name is rejected
      Given the name "roles" was already registered at the top level
      When a Maintainer registers another command named "roles" at the top level
      Then registration will fail with an error naming "roles"
      And the failure will occur before any user command runs

    # Source: 001-command-registration — Scenario: Empty command name
    Scenario: Empty command name is rejected
      Given an otherwise valid registration state
      When a Maintainer registers a command whose name is empty or only whitespace
      Then registration will fail with an error identifying the command

    # Source: 001-command-registration — Scenario: Command without a summary
    Scenario: Missing summary is rejected
      Given an otherwise valid registration state
      When a Maintainer registers a command whose summary is empty or only whitespace
      Then registration will fail with an error identifying the command

    # Source: 001-command-registration — Scenario: Leaf command without an action
    Scenario: Leaf command without an action is rejected
      Given an otherwise valid registration state
      When a Maintainer registers a leaf command that has no action
      Then registration will fail with an error identifying the command

    # Source: 001-command-registration — Scenario: Group with no children
    Scenario: Group without children is rejected
      Given an otherwise valid registration state
      When a Maintainer registers a group that has no children
      Then registration will fail with an error identifying the group

    # Source: 001-command-registration — Proposed: Startup aborts on any failed registration (plan: no partial command tree)
    Scenario: One failed registration prevents the whole CLI from running
      Given several commands were registered successfully
      And one further command fails registration
      When the CLI starts
      Then no command will be dispatched
      And the CLI will not expose a partial command tree
