# Source: 055-proposal-creation — Scenario: Create a proposal with an inline change set

Feature: Proposal Creation
  A proposal is the only sanctioned path to alter governance structure, and the
  anchor of the write path. This is the CLI's second write: `glassfrog proposal
  create <tension-id> --changes <source>` creates a draft proposal against an
  existing tension, carrying a caller-supplied governance change set (a JSON
  array sourced inline, from a file, or from piped stdin). The change set is
  passed through verbatim above a minimal floor — every element must be an
  object with a non-empty `type`. The created proposal, including its `prp_` id
  and `draft` status, is produced so a later step can advance it to circulation.
  The whole write surface is Premium-gated; every failure exits with a named
  cause and the right exit code.
  (affects: Practitioner)

  Rule: Create a draft proposal anchored to a tension
    # In order to turn a tension I captured into an actionable governance change,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to create a draft proposal anchored to that tension in one command.

    # Source: 055-proposal-creation — Scenario: Create a proposal with an inline change set
    Scenario: A draft proposal is created from an inline change set
      Given a complete connection context with a stored token
      And the tension "ten_0123" exists
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\",\"name\":\"Scribe\"}]'"
      Then the request will post the proposal to the proposals endpoint
      And the request body will carry the anchor "tension_id" and the changes array verbatim
      And the created proposal will be printed with its "prp_" id and "draft" status
      And the command will exit with code 0

    # Source: 055-proposal-creation — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]'"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 055-proposal-creation — Scenario: Unknown anchor tension
    Scenario: An unknown anchor tension fails with the API status
      Given a complete connection context with a stored token
      And no tension "ten_ffff" exists
      When an agent runs "glassfrog proposal create ten_ffff --changes '[{\"type\":\"CreateRole\"}]'"
      Then stderr will report that the create failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 055-proposal-creation — Scenario: Premium async proposals not enabled
    Scenario: A create on an organization without async proposals is refused
      Given a complete connection context with a stored token
      And the proposals endpoint answers the create with a permission-denied response
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]'"
      Then stderr will report that the create failed and name the HTTP status
      And the command will exit with the permission code

    # Source: 055-proposal-creation — Proposed: plan Risk + §133 (POST is non-idempotent, never auto-retried on 429)
    Scenario: A rate-limited create is surfaced, not silently re-sent
      Given a complete connection context with a stored token
      And the proposals endpoint answers the create with a rate-limit response
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]'"
      Then the rate-limit will be surfaced on the first occurrence
      And the create will not be retried, so no duplicate proposal is created
      And the command will exit with the rate-limit code

    # Source: 055-proposal-creation — Scenario: The change set is sent through verbatim beyond the type floor
    @validation @wip
    Scenario: The change set is sent through verbatim above the type floor
      Given a complete connection context with a stored token
      When an agent creates a proposal with a change array of typed objects carrying extra command-specific keys
      Then the request body's "changes" will match the supplied array exactly
      And the command will read only each element's "type", leaving every other key untouched

    # Source: 055-proposal-creation — Scenario: No client-side feature-gate pre-check
    @validation @wip
    Scenario: The command issues the request rather than pre-checking the Premium gate
      Given a complete connection context with a stored token on an organization without async proposals
      When an agent creates a proposal
      Then the request will be sent and the server's permission-denied response surfaced
      And the command will not refuse locally on any client-side Premium check

    # Source: 055-proposal-creation — Scenario: Create does not reach into the rest of the write flow or the reads
    @validation @wip
    Scenario: Create exposes no other write-flow transitions or reads
      Given the implemented "glassfrog proposal" command group
      When its subcommands and help are inspected
      Then only "create" will be available
      And no advance, respond, withdraw, list, or get behavior will be advertised or implemented

  Rule: Get back the proposal's id and draft status
    # In order to advance the proposal to circulation later,
    # as an AI agent assembling the write flow,
    # I want the create to return the proposal's prp_ id and draft status.

    # Source: 055-proposal-creation — Scenario: The created proposal's id and status are visible in JSON output
    Scenario: The created proposal's id and status are present in structured output
      Given a complete connection context with a stored token
      And the tension "ten_0123" exists
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]' --output json"
      Then the structured result will contain the created proposal's "prp_" id and a "draft" status
      And the command will exit with code 0

    # Source: 055-proposal-creation — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    Scenario: An invalid output format is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"type\":\"CreateRole\"}]' -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Source the change set from a file or stdin
    # In order to submit a large or complex change set without cramming JSON into the command line,
    # as an AI agent driving the CLI,
    # I want to read the --changes array from a file or from piped stdin.

    # Source: 055-proposal-creation — Scenario: Read the change set from a file
    Scenario: The change set is read from a file
      Given a complete connection context with a stored token
      And a file "changes.json" holding a JSON array of changes
      When an agent runs "glassfrog proposal create ten_0123 --changes changes.json"
      Then the request body will carry the changes read from the file verbatim
      And the created proposal will be printed
      And the command will exit with code 0

    # Source: 055-proposal-creation — Scenario: Read the change set from piped stdin
    Scenario: The change set is read from piped stdin
      Given a complete connection context with a stored token
      And a JSON array of changes piped on standard input
      When an agent runs "glassfrog proposal create ten_0123 --changes stdin"
      Then the request body will carry the changes read from stdin verbatim
      And the command will exit with code 0

  Rule: Refuse to create a changeless or malformed proposal
    # In order to avoid submitting a meaningless empty proposal,
    # as an AI agent driving the CLI,
    # I want a missing or empty change set rejected before any request is made.

    # Source: 055-proposal-creation — Scenario: Missing change set is rejected before any request
    Scenario: A missing change set is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal create ten_0123"
      Then stderr will report that "--changes" is required
      And no request will be sent
      And the command will exit with code 2

    # Source: 055-proposal-creation — Scenario: Empty change set is rejected before any request
    Scenario: An empty change set is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal create ten_0123 --changes '[]'"
      Then stderr will report that at least one change is required
      And no request will be sent
      And the command will exit with code 2

    # Source: 055-proposal-creation — Scenario: Unparseable change set is rejected before any request
    Scenario: An unparseable change set is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal create ten_0123 --changes 'not json'"
      Then stderr will report a usage error naming the change source
      And no request will be sent
      And the command will exit with code 2

    # Source: 055-proposal-creation — Scenario: A change without a type is rejected before any request
    Scenario: A change lacking a type is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal create ten_0123 --changes '[{\"name\":\"Scribe\"}]'"
      Then stderr will report that every change must carry a "type"
      And no request will be sent
      And the command will exit with code 2
