# Source: 062-operator-orientation — Scenario: Getting parseable output by consulting orientation

Feature: Operator Orientation
  The AI agent driving the CLI has no packaged operating knowledge — command
  surface, output parsing, pagination, exit-code handling, credential setup,
  write-safety — so it rediscovers how to drive the CLI each session and can
  mis-drive it or run ungated writes. Operator Orientation is the root of the
  Agent Operating Surface: a repo-shipped Claude plugin (a manifest at
  `plugin/.claude-plugin/plugin.json` plus one skill at
  `plugin/skills/orientation/SKILL.md`) that packages cross-cutting
  operating knowledge the agent consults on demand. It adds no API capability
  and points at the CLI's own `--help` flag (`glassfrog <command> --help`) for per-command detail. A
  best-effort drift guard in `internal/build` keeps the skill's enumerable
  facts (output formats, exit codes, the credential command) truthful to the
  shipped CLI. Distribution (its marketplace) and write-safety enforcement are
  separate, later capabilities.
  (affects: AI agent, Practitioner)

  Rule: Operate the CLI correctly from the first session without rediscovery
    # In order to operate the CLI correctly from the first session without
    # rediscovering its surface,
    # as an AI agent,
    # I want to consult packaged orientation covering output formats,
    # pagination, exit codes, credentials, and where to find per-command detail.

    # Source: 062-operator-orientation — Scenario: Getting parseable output by consulting orientation
    Scenario: Select a parseable output format
      Given the orientation skill was available to the agent
      And the agent needed to read a practitioner's roles for downstream parsing
      When the agent consults the orientation for machine-parseable output
      Then the orientation will name "json" and "yaml" as the parseable formats
      And it will instruct the agent to pass "--output json" rather than parse human-rendered text

    # Source: 062-operator-orientation — Scenario: Paging through a large result set
    Scenario: Page through a multi-page result set
      Given a list command returned more results than one response held
      When the agent consults the orientation on pagination
      Then the orientation will explain how to detect that more pages exist
      And it will explain how to fetch the subsequent pages

    # Source: 062-operator-orientation — Scenario: Reacting to a non-zero exit code
    Scenario: React to a non-zero exit code
      Given a glassfrog command had just exited with a non-zero code
      When the agent consults the orientation for that exit code
      Then the orientation will state the meaning of each code in the 0–7 convention
      And it will state the appropriate reaction for the code received

    # Source: 062-operator-orientation — Scenario: Missing credentials
    Scenario: Set up missing credentials
      Given the agent had supplied no credential
      And a command failed for lack of authentication
      When the agent consults the orientation
      Then the orientation will direct the agent to "glassfrog auth login" for the X-Auth-Token key
      And it will introduce no credential mechanism beyond the CLI's own

    # Source: 062-operator-orientation — Scenario: Per-command detail comes from the CLI, not the orientation
    Scenario: Find per-command detail in the CLI's own help
      Given the agent needed the exact flags for one specific command
      When the agent consults the orientation
      Then the orientation will direct the agent to "glassfrog <command> --help" for that command
      And it will not itself enumerate the command's flags

    # Source: 062-operator-orientation — Scenario: Cross-cutting knowledge drifts from the shipped CLI
    Scenario: Detect orientation drifted from the shipped CLI
      Given the CLI's exit-code or output-format behavior had changed
      When the orientation is checked against the shipped CLI
      Then the mismatch will be treated as a defect to fix
      And it will not be accepted as a tolerable difference

    # Source: 062-operator-orientation — Scenario: No invented surface
    @validation @wip
    Scenario: Orientation names no surface the CLI lacks
      Given the produced orientation content
      When every command, flag, and format it names is checked against the shipped CLI
      Then each one will exist in the CLI
      And the orientation will name no surface the CLI does not expose

    # Source: 062-operator-orientation — Scenario: No Holacracy coaching
    @validation @wip
    Scenario: Orientation carries no Holacracy coaching
      Given the produced orientation content
      When it is inspected for governance-practice instruction
      Then it will describe only how to drive the CLI
      And it will contain no Holacracy coaching or tension interpretation

    # Source: 062-operator-orientation — architecture-informed (interface Error Communication; proposed by skill)
    Scenario: Malformed manifest leaves the plugin unloadable
      Given the plugin manifest at "plugin/.claude-plugin/plugin.json" was malformed
      When the plugin host attempts to load the plugin
      Then the host will not register the orientation skill
      And the agent will fall back to rediscovery with no command in the CLI broken

  Rule: Have one installable unit that carries the operating knowledge
    # In order to have a single installable unit that carries the operating knowledge,
    # as a practitioner (or whoever provisions the agent),
    # I want the operating surface defined as a Claude plugin I can later install
    # into the agent's environment.

    # Source: 062-operator-orientation — Scenario: The plugin makes orientation consultable
    Scenario: Orientation is consultable once the plugin is present
      Given the plugin was present in an agent's environment
      When the agent looks for operating knowledge
      Then the orientation knowledge will be available to consult
      And no configuration beyond the CLI's existing credential setup will be required

    # Source: 062-operator-orientation — Scenario: No distribution defined
    @validation @wip
    Scenario: Plugin defines no distribution machinery
      Given the plugin definition produced by this feature
      When it is inspected for a marketplace, publishing workflow, or install flow
      Then none will be present
      And distribution will be left to the Operating-Surface Packaging capability

  Rule: Be warned of write-safety before the enforcing guardrail exists
    # In order to avoid mis-driving governance writes before the enforcing
    # guardrail exists,
    # as an AI agent,
    # I want to be told the write-safety expectations as part of orientation.

    # Source: 062-operator-orientation — Scenario: Guidance precedes enforcement on a governance write
    Scenario: Surface the write-safety expectation without gating
      Given the Write-Safety Guardrail did not yet exist
      And the agent was about to run a command that writes to the governance record
      When the agent consults the orientation
      Then the orientation will state the expectation to confirm before writing
      And it will state that a 412 stale-write refusal means re-read and re-confirm, not blind retry
      And it will not block or gate the write itself

    # Source: 062-operator-orientation — Scenario: Guidance, not gating
    @validation @wip
    Scenario: Orientation describes but never enforces gating
      Given the orientation's treatment of governance writes
      When it is inspected for enforcement behavior
      Then it will only describe the write-safety expectation
      And it will nowhere implement confirmation, gating, or blocking

    # Source: 062-operator-orientation — architecture-informed (plan ADR-4; proposed by skill)
    Scenario: Drift guard fails when a documented anchor leaves the CLI
      Given the orientation documented an output-format token that the CLI no longer supported
      When the internal/build drift guard runs
      Then the guard will fail
      And it will report which documented anchor no longer matches the shipped CLI
