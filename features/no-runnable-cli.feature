# Source: 001-command-registration — Scenario: Register a top-level leaf command

Feature: No Runnable CLI
  No project skeleton or command framework exists to build any command on.
  Before any command can be built, the CLI needs a way to know which commands
  it offers and how they are organized, and to reject malformed registrations
  before users hit them. (affects: Maintainer)

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

  Rule: Discover the available commands
    # In order to discover what the CLI can do without prior knowledge of its commands,
    # as an AI agent operating the CLI,
    # I want to see a listing of every command with its summary.

    # Source: 003-help-and-version — Scenario: Root listing shows all top-level commands
    Scenario: Root help lists all top-level commands with summaries
      Given a "version" command and a "roles" group were registered, each with a summary
      When the caller invokes "glassfrog --help"
      Then the CLI will list "version" and "roles"
      And each will appear with its one-line summary

    # Source: 003-help-and-version — Scenario: Group help lists its subcommands
    Scenario: Group help lists its subcommands with summaries
      Given a "roles" group containing "list" and "get" was registered, each with a summary
      When the caller invokes "glassfrog roles --help"
      Then the CLI will show the "roles" summary
      And it will list "list" and "get", each with its summary

    # Source: 003-help-and-version — Scenario: Root help on an empty command set
    Scenario: Root help on an empty command set lists no commands
      Given no commands were registered
      When the caller invokes "glassfrog --help"
      Then the CLI will produce a listing naming no commands
      And it will not fail

    # Source: 003-help-and-version — Scenario: Group help lists only immediate children
    Scenario: Group help lists only immediate children
      Given a "proposals" group whose "changes" subgroup contains "add" was registered
      When the caller invokes "glassfrog proposals --help"
      Then the CLI will list "changes" with its summary
      And it will not list the nested "add" command

    # Source: 003-help-and-version — Proposed: Alphabetical listing order (plan ADR-1, framework sorting)
    Scenario: Commands are listed in alphabetical order
      Given a "version" command and a "roles" group were registered
      When the caller invokes "glassfrog --help"
      Then "roles" will be listed before "version"

    # Source: 003-help-and-version — Proposed: Built-in commands hidden (plan ADR-2, framework built-ins)
    Scenario: Framework built-in commands are absent from the listing
      Given the command set was assembled
      When the caller invokes "glassfrog --help"
      Then the listing will not include a "help" command
      And it will not include a "completion" command

    # Source: 003-help-and-version — Scenario: Output is deterministic across runs
    @validation @wip
    Scenario: Listing output is identical across repeated runs
      Given an unchanged command set
      When the caller invokes "glassfrog --help" twice
      Then the two listings will be identical

  Rule: Read a command's usage before invoking it
    # In order to invoke a command correctly,
    # as an AI agent or practitioner,
    # I want to request a command's help and read its path and purpose.

    # Source: 003-help-and-version — Scenario: Leaf command usage
    Scenario: Leaf command help shows its path and summary
      Given a "roles get" command with summary "Show one role" was registered
      When the caller invokes "glassfrog roles get --help"
      Then the CLI will show the "roles get" usage line
      And it will show the summary "Show one role"

    # Source: 003-help-and-version — Scenario: Help requested for an unregistered command
    Scenario: Help for an unregistered command renders no usage
      Given no command named "bogus" was registered
      When the caller invokes "glassfrog bogus --help"
      Then the CLI will not render usage for "bogus"
      And the unknown-command outcome will be left to dispatch

    # Source: 003-help-and-version — Scenario: Both help and version requested
    Scenario: Help takes precedence over version
      Given the CLI was built with a version string
      When the caller invokes "glassfrog --help --version"
      Then the CLI will produce help output
      And it will not produce version output

    # Source: 003-help-and-version — Scenario: Help introduces no new required registration data
    @validation @wip
    Scenario: Help shows no description beyond declared data
      Given commands that declare only the data registration requires
      When help or usage is produced
      Then every command-specific description shown will come from data the command already declares
      And no new mandatory registration field will be required

  Rule: Confirm which build of the CLI is running
    # In order to confirm I am operating the expected build of the CLI,
    # as an operator,
    # I want to request the version and read a single version string.

    # Source: 003-help-and-version — Scenario: Version via flag and via command match
    Scenario: Version flag and version command produce identical output
      Given the CLI was built with version "1.2.0"
      When the caller invokes "glassfrog --version"
      And the caller invokes "glassfrog version"
      Then both invocations will print the same output
      And the output will contain "1.2.0"

    # Source: 003-help-and-version — Scenario: Version flag on a non-root command is not a version request
    Scenario: Version flag on a subcommand is not a version request
      Given a "roles" group without its own version flag was registered
      When the caller invokes "glassfrog roles --version"
      Then the CLI will not print version output
      And the unrecognized flag will be left to dispatch as a usage error

    # Source: 003-help-and-version — Scenario: No exit-code or routing logic leaks into rendering
    @validation @wip
    Scenario: Rendering neither selects exit codes nor routes invocations
      Given the Help & Version capability
      When its help, listing, and version rendering is inspected
      Then it will not select a process exit code
      And it will not decide which command an invocation names
