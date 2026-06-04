Feature: Help & Version
  Part of the No Runnable CLI problem: no project skeleton or command framework
  exists to build any command on. Help & Version renders --help / usage output,
  a discoverable listing of registered commands, and --version. (affects: AI agent, operator)

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
