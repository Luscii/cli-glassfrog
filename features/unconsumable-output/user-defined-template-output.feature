# Source: 035-user-defined-template-output — Scenario: a template file renders the result

Feature: Unconsumable Output — User-Defined Template Output
  Results aren't shaped for an AI agent to parse reliably or for a human to
  read. User-Defined Template Output lets an operator render a read's successful
  result through their own template instead of a built-in format, reusing 020's
  -o / --output flag: at the flag rung only, a reserved token (full | compact |
  json | yaml) keeps its meaning, "stdin" reads a template from a pipe, and any
  other value is a path to a template file. Reserved names win; env var and
  config file keep their four-token contract. The template renders the invoked
  command's result through a clone of 019's engine (data-only, no fabrication);
  a missing file, unparseable template, or empty stdin fails fast as a usage
  error before any request, and every template failure exits with the usage
  code. (affects: AI agent, Practitioner)

  Rule: Render a read through my own template file
    # In order to transform a read's result into the exact shape my downstream pipeline expects,
    # as an AI agent operating the CLI,
    # I want to pass -o <templateFile> and receive the result rendered through my own template.

    # Source: 035-user-defined-template-output — Scenario: a template file renders the result
    Scenario: A template file renders the result
      Given the read "glassfrog me roles" had produced several roles
      And the invocation passed "-o ./roles.tmpl" naming a readable, parseable template
      When the result is rendered
      Then the roles' data will be rendered through that template
      And stdout will carry the template's output

    # Source: 035-user-defined-template-output — Scenario: a missing template file fails fast
    Scenario: A missing template file fails fast
      Given the invocation passed "-o ./nope.tmpl" naming a file that does not exist
      When the command is run
      Then the command will report a usage error naming the file
      And it will make no API request
      And it will exit with the usage exit code 2

    # Source: 035-user-defined-template-output — Scenario: a malformed template fails fast before the read
    Scenario: A malformed template fails before any request
      Given the invocation passed "-o ./broken.tmpl" naming a file whose template cannot be parsed
      When the command is run
      Then the command will report a usage error naming the source
      And it will make no API request
      And it will exit with the usage exit code 2

    # Source: 035-user-defined-template-output — Scenario: a guarded template renders an absence marker for a missing field
    Scenario: A guarded template renders an absence marker for a missing field
      Given the read had produced a result that omitted an embedded collection
      And a template that guards a reference to that collection
      When the result is rendered
      Then an explicit absence marker will appear where the field would be
      And no fabricated data value will stand in for data the API did not return

    # Proposed (architecture-informed): plan ADR-3 + interface-cli error table — a post-parse execution failure is buffer-then-write and maps to UsageError(2)
    Scenario: An execution failure writes nothing to stdout
      Given the read had produced a successful result
      And a template that references an absent field without guarding it
      When the result is rendered
      Then stdout will carry nothing
      And the command will report a usage error naming the source
      And it will exit with the usage exit code 2

    # Source: 035-user-defined-template-output — Scenario: a user template introduces no value absent from the source
    @validation @wip
    Scenario: A user template introduces no value absent from the source
      Given any successful result and any user template
      When the result is rendered
      Then every data value shown will trace to a field the result carried
      And no data value will be synthesized

    # Source: 035-user-defined-template-output — Scenario: a malformed template is caught before any API call
    @validation @wip
    Scenario: A malformed template is caught before any API call
      Given a user template that cannot be parsed, from a file or from stdin
      When the command is run
      Then the failure will be reported before any read is attempted
      And no API request will be made

  Rule: Pipe a one-off template without writing a file
    # In order to render a result with a one-off template without writing a file to disk,
    # as an AI agent composing a command on the fly,
    # I want to pipe a template into the command and select it with -o stdin.

    # Source: 035-user-defined-template-output — Scenario: a template is read from piped stdin
    Scenario: A template piped on stdin renders the result
      Given a template had been piped to the command on standard input
      And the invocation passed "-o stdin" to a successful "glassfrog me" read
      When the result is rendered
      Then the template will be read from standard input
      And the "me" result's data will be rendered through it

    # Source: 035-user-defined-template-output — Scenario: stdin selected with nothing piped
    Scenario: Selecting stdin with nothing piped fails fast
      Given the invocation passed "-o stdin"
      And no template had been piped to standard input
      When the command is run
      Then the command will report a usage error
      And it will make no API request
      And it will exit with the usage exit code 2

  Rule: Supply my own view instead of the built-in formats
    # In order to produce a bespoke human-readable view of governance data,
    # as a practitioner with a specific reporting format in mind,
    # I want to supply my own template instead of the built-in full / compact views.

    # Source: 035-user-defined-template-output — Scenario: a reserved name wins over a same-named file
    Scenario: A reserved name wins over a same-named file
      Given a file named "full" existed in the current working directory
      And the invocation passed "-o full"
      When the result is rendered
      Then the built-in full template will be selected
      And the file named "full" will not be read

    # Source: 035-user-defined-template-output — Scenario: a template source is never honored from env or config
    @validation @wip
    Scenario: A template source is never honored from env or config
      Given no -o flag was present
      And GLASSFROG_OUTPUT or the .glassfrogrc output value held a file path
      When the effective format is resolved
      Then the invalid-selector usage error will be raised for that source
      And no template file will be read or rendered
