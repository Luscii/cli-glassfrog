# Source: 020-output-format-selection — Scenario: --output json selects the JSON encoder

Feature: Unconsumable Output — Output Format Selection
  Results aren't shaped for an AI agent to parse reliably or for a human to
  read. Output Format Selection adds the --output flag (-o) that picks one of
  four formats per invocation — full | compact | json | yaml — resolving it from
  a precedence chain (flag > GLASSFROG_OUTPUT > .glassfrogrc output > default
  full) and dispatching a successful result to the matching renderer: the human
  templates (019) for full/compact, the structured encoders (018) for
  json/yaml. It is what first makes compact, json, and yaml reachable from the
  command line. Selection and success dispatch are owned here; rendering
  failures in the selected format is Output-Aware Failure Rendering (032).
  (affects: AI agent, Practitioner)

  Rule: Select a machine format per invocation
    # In order to parse a command's full result with my tooling,
    # as an AI agent operating the CLI,
    # I want to select --output json (or yaml) per invocation and receive that format.

    # Source: 020-output-format-selection — Scenario: --output json selects the JSON encoder
    Scenario: A json selector routes the result to the JSON encoder
      Given the authenticated read "glassfrog me" had produced a successful payload
      And the invocation passed "--output json"
      When the result is rendered
      Then the result will be routed to the structured JSON encoder
      And stdout will carry a single JSON document of the raw payload

    # Source: 020-output-format-selection — Scenario: format matching is case-insensitive
    Scenario: An uppercase selector selects the same format
      Given the read "glassfrog me" had produced a successful payload
      And the invocation passed "--output JSON"
      When the result is rendered
      Then the JSON format will be selected
      And the result will be routed to the JSON encoder exactly as lowercase "json" would

    # Source: 020-output-format-selection — Scenario: an unknown format value fails fast
    Scenario: An unknown selector value fails before any request
      Given the invocation passed "--output xml"
      When the command is run
      Then the command will report a usage error naming the value "xml"
      And it will make no API request
      And it will exit with the usage exit code 2

    # Source: 020-output-format-selection — Scenario: each token dispatches to exactly its renderer
    @validation @wip
    Scenario: Each format routes to exactly its renderer
      Given the same successful result
      When it is rendered once under each of full, compact, json, and yaml
      Then full and compact will route through the human templates
      And json and yaml will route through the structured encoders
      And no format will route to a renderer other than its own

    # Source: 020-output-format-selection — Scenario: selection changes rendering only, never the fetched data
    @validation @wip
    Scenario: Selection changes rendering only, not the fetched data
      Given a read whose result carried a fixed set of fields
      When the result is rendered under each of the four formats
      Then every format will reflect the same underlying result data
      And the selection will alter only the encoding or template applied

  Rule: Set the output format once for every command
    # In order to always get machine-readable output without passing a flag on every call,
    # as an AI agent with a fixed pipeline,
    # I want to set the output format once via an environment variable or config file and have every command honor it.

    # Source: 020-output-format-selection — Scenario: omitting --output selects the default full template
    Scenario: Omitting the selector renders the default full format
      Given the read "glassfrog me" had produced a successful payload
      And no --output flag, GLASSFROG_OUTPUT value, or .glassfrogrc output value was present
      When the result is rendered
      Then the default format full will be resolved
      And the result will be routed to the full human template

    # Source: 020-output-format-selection — Scenario: the flag overrides the environment variable and config file
    Scenario: The flag overrides the environment variable and config file
      Given GLASSFROG_OUTPUT held "yaml"
      And the .glassfrogrc output value held "compact"
      And the invocation passed "--output json"
      When the format is resolved
      Then json will be used
      And no lower-precedence source will be consulted

    # Source: 020-output-format-selection — Scenario: the config file supplies the format when flag and environment are absent
    Scenario: The config file supplies the format when flag and environment are absent
      Given no --output flag and no GLASSFROG_OUTPUT value were present
      And the nearest .glassfrogrc on the walk-up held output "compact"
      When the format is resolved
      Then compact will be selected from that file
      And the result will be routed to the compact human template

    # Source: 020-output-format-selection — Scenario: an invalid value in a lower-precedence source surfaces loudly
    Scenario: An invalid environment value fails, naming its source
      Given no --output flag was present
      And GLASSFROG_OUTPUT held "xml"
      When the command is run
      Then the command will report a usage error naming the GLASSFROG_OUTPUT source and the value "xml"
      And it will not fall through to the config file or the default
      And it will exit with the usage exit code 2

    # Proposed (architecture-informed): plan ADR-4 / interface-cli error table — an unreadable config surfaces loudly, mirroring --base-url
    Scenario: An unreadable config file fails resolution as a usage error
      Given no --output flag and no GLASSFROG_OUTPUT value were present
      And a .glassfrogrc on the walk-up could not be read or parsed
      When the format is resolved
      Then the command will report the config read error naming the file
      And it will exit with the usage exit code 2

    # Source: 020-output-format-selection — Scenario: the precedence chain resolves the first available source
    @validation @wip
    Scenario: Resolution takes the first available source
      Given formats present at the flag, the environment variable, and the config file in various combinations
      When the effective format is resolved
      Then the first source in flag, environment, config, default order that yields a value will win
      And an absent source will be skipped while a present-but-invalid source will raise the usage error

  Rule: Reach the compact human rendering
    # In order to scan a long list quickly when reading as a human,
    # as a practitioner triaging governance,
    # I want to select the compact rendering that 019 built but the CLI did not yet expose.

    # Source: 020-output-format-selection — Scenario: --output compact makes the compact rendering reachable
    Scenario: A compact selector renders one line per record
      Given the read "glassfrog me roles" had produced several roles
      And the invocation passed "--output compact"
      When the result is rendered
      Then the result will be routed to the compact human template
      And each role will appear on a single line — the rendering previously reachable from no command-line surface
