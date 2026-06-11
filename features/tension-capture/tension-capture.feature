# Source: 042-tension-capture — Scenario: Capture a tension with body only

Feature: Tension Capture
  A tension is a gap a role-filler senses — and the required seed of a
  governance proposal. This is the CLI's first write: `glassfrog tension create
  <role-id> --body "..."` records a tension sensed by that role (the path role
  is the sensing role; the token's person is the sensing person) and produces
  the created tension, including its `ten_` id, so a later proposal can
  reference it. An optional `--label` and a validated `--meeting-type` may be
  attached. A missing or blank body is refused before any request, and every
  failure exits with a named cause and the right exit code.
  (affects: Practitioner)

  Rule: Record a tension as the entry point to a proposal
    # In order to begin a governance change from a gap I've noticed,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to record a tension against the sensing role with one command.

    # Source: 042-tension-capture — Scenario: Capture a tension with body only
    Scenario: A tension is captured against the sensing role
      Given a complete connection context with a stored token
      And the role "role_0123" exists
      When an agent runs "glassfrog tension create role_0123 --body \"We ship faster than we update the roadmap.\""
      Then the request will post the tension to the role's tensions endpoint
      And the created tension will be printed with its "ten_" id and computed status
      And the command will exit with code 0

    # Source: 042-tension-capture — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tension create role_0123 --body \"a tension\""
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 042-tension-capture — Scenario: Unknown sensing role
    Scenario: An unknown sensing role fails with the API status
      Given a complete connection context with a stored token
      And no role "role_ffff" exists
      When an agent runs "glassfrog tension create role_ffff --body \"a tension\""
      Then stderr will report that the capture failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 042-tension-capture — Proposed: plan Risk + §133 (POST is non-idempotent, never auto-retried on 429)
    Scenario: A rate-limited capture is surfaced, not silently re-sent
      Given a complete connection context with a stored token
      And the tensions endpoint answers the capture with a rate-limit response
      When an agent runs "glassfrog tension create role_0123 --body \"a tension\""
      Then the rate-limit will be surfaced on the first occurrence
      And the capture will not be retried, so no duplicate tension is created
      And the command will exit with the rate-limit code

    # Source: 042-tension-capture — Scenario: Capture stays on the write path only
    @validation @wip
    Scenario: Capture exposes no tension reads, edits, or deletes
      Given the implemented "glassfrog tension" command group
      When its subcommands and help are inspected
      Then only "create" will be available
      And no list, get, update, or delete behavior will be advertised or implemented

    # Source: 042-tension-capture — Scenario: The sensing person is never supplied by the client
    @validation @wip
    Scenario: The sensing person is derived from the token, never sent
      Given a complete connection context with a stored token
      When an agent captures a tension with "glassfrog tension create role_0123 --body \"a tension\""
      Then the request body will carry no sensing-person field
      And the sensing person on the created tension will be the token's own identity

  Rule: Get back the tension's id to seed a proposal
    # In order to hand the new tension to a proposal later,
    # as an AI agent assembling the write flow,
    # I want the capture to return the tension's ten_ id.

    # Source: 042-tension-capture — Scenario: The created tension's id is visible in JSON output
    Scenario: The created tension's id is present in structured output
      Given a complete connection context with a stored token
      And the role "role_0123" exists
      When an agent runs "glassfrog tension create role_0123 --body \"a tension\" --output json"
      Then the structured result will contain the created tension's "ten_" id
      And the command will exit with code 0

  Rule: Attach a label and meeting-type at capture
    # In order to give the tension a short handle and signal where it should be worked,
    # as a practitioner sensing an issue,
    # I want to attach a label and a meeting-type at capture.

    # Source: 042-tension-capture — Scenario: Capture a tension with body, label, and meeting-type
    Scenario: A tension is captured with a label and a meeting-type
      Given a complete connection context with a stored token
      And the role "role_0123" exists
      When an agent runs "glassfrog tension create role_0123 --body \"a tension\" --label \"Roadmap drift\" --meeting-type governance"
      Then the request body will carry the body, the label, and "meeting_type" set to "governance"
      And the created tension will be printed
      And the command will exit with code 0

    # Source: 042-tension-capture — Scenario: Unsupported meeting-type is rejected before any request
    Scenario: An unsupported meeting-type is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension create role_0123 --body \"a tension\" --meeting-type weekly"
      Then stderr will report the unsupported value and list the supported set
      And no request will be sent
      And the command will exit with code 2

  Rule: Refuse to record an empty tension
    # In order to avoid recording an empty, meaningless tension,
    # as an AI agent driving the CLI,
    # I want a missing or blank body rejected before any request is made.

    # Source: 042-tension-capture — Scenario: Missing body is rejected before any request
    Scenario: A missing body is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension create role_0123"
      Then stderr will report that "--body" is required
      And no request will be sent
      And the command will exit with code 2

    # Source: 042-tension-capture — Scenario: Whitespace-only body is treated as empty
    Scenario: A whitespace-only body is rejected as empty
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension create role_0123 --body \"   \""
      Then stderr will report that "--body" is required
      And no request will be sent
      And the command will exit with code 2
