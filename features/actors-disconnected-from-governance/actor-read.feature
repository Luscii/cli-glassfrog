# Source: 049-actor-read — Scenario: Read a single actor by id

Feature: Actors Disconnected from Governance — Actor Read
  Once the directory has turned a name or a role into a per_/agt_ id, the
  operator still can't see what that actor does. Actor Read is the single-actor
  drill-in — `glassfrog actors <id>` reads one person or agent by id and, on
  request, embeds their governance footprint: the roles they fill with the
  accountabilities, domains, and purposes those carry, and their assignments.
  It issues one request and never walks pages, reads agents through the ungated
  unified endpoint, validates the footprint flag locally while passing the id
  through to a clean not-found, and renders through the shared output seam or
  fails with a named error and the right exit code.
  (affects: Practitioner, AI agent)

  Rule: See what an actor does — their governance footprint
    # In order to see what an actor actually does — the roles they fill and the
    # accountabilities, domains, and purposes those carry — once I know their id,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to read one actor with their roles embedded.

    # Source: 049-actor-read — Scenario: Read an actor with their governance footprint embedded
    Scenario: A roles include embeds the actor's governance footprint
      Given a complete connection context with a stored token
      And an actor that exists in the organization
      When an agent runs "glassfrog actors per_abc --include roles"
      Then the request will carry "include" set to "roles"
      And the actor will be printed with each role's name, purpose, accountabilities, and domains
      And the command will exit with code 0

    # Source: 049-actor-read — Scenario: Read an actor with their assignments embedded
    Scenario: An assignments include embeds the actor's assignments
      Given a complete connection context with a stored token
      And an actor who fills several roles
      When an agent runs "glassfrog actors per_abc --include assignments"
      Then the request will carry "include" set to "assignments"
      And the actor will be printed with its assignments embedded
      And the command will exit with code 0

    # Source: 049-actor-read — Scenario: Unsupported --include value is rejected before any request
    Scenario: An unsupported include is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog actors per_abc --include nonsense"
      Then stderr will report the unsupported value and list the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 049-actor-read — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog actors per_abc" run under the default human format
      When the output is inspected
      Then it will show the reshaped actor projection only
      And it will not contain the raw "data" JSON envelope

    # Source: 049-actor-read — Scenario: The single read issues no page walk
    @validation @wip
    Scenario: A single read issues exactly one request and no page walk
      Given a transport tripwire that records every request that is sent
      And an actor who fills many roles is returned with assignments embedded
      When an agent runs "glassfrog actors per_abc --include assignments"
      Then exactly one request will be issued to "/actors/per_abc"
      And no pagination cursor will be followed

    # Source: 049-actor-read — Proposed: interface CLI (mode separation — --include is single-only)
    Scenario: A footprint include with no id is rejected
      Given a complete connection context with a stored token
      When an agent runs "glassfrog actors --include roles"
      Then stderr will report that --include applies only to a single actor read
      And no API request will be sent
      And the command will exit with code 2

  Rule: Drill into an actor by id
    # In order to drill into an id I found in the directory or as a role's filler,
    # as an AI agent assembling context before acting,
    # I want to read a single actor — person or agent — by their per_/agt_ id with one command.

    # Source: 049-actor-read — Scenario: Read a single actor by id
    Scenario: An id reads a single actor
      Given a complete connection context with a stored token
      And an actor that exists in the organization
      When an agent runs "glassfrog actors per_abc"
      Then the request will read "/actors/per_abc"
      And the actor's id, name, and kind will be printed
      And the command will exit with code 0

    # Source: 049-actor-read — Scenario: Read an agent by its agt_ id
    Scenario: An agt_ id reads an agent
      Given a complete connection context with a stored token
      And an agent that exists in the organization
      When an agent runs "glassfrog actors agt_def"
      Then the request will read "/actors/agt_def"
      And the agent will be printed as a single actor
      And the command will exit with code 0

    # Source: 049-actor-read — Scenario: An agent read does not require the gated alias
    Scenario: An agent read reaches the ungated unified endpoint
      Given a token without the ai_integration feature
      When an agent runs "glassfrog actors agt_def"
      Then the request will read the unified "/actors/agt_def" endpoint
      And it will not route through the ai_integration-gated "/agents" alias

    # Source: 049-actor-read — Scenario: Agent drill-in does not route through the gated alias
    @validation @wip
    Scenario: An agent drill-in never touches the gated alias
      Given a token without the ai_integration feature
      And a transport tripwire that records every request that is sent
      When an agent runs "glassfrog actors agt_def"
      Then the tripwire will confirm the only request read "/actors/agt_def"
      And no request reached the ai_integration-gated "/agents" alias

    # Source: 049-actor-read — Proposed: interface CLI (mode separation — filters are list-only)
    Scenario: A list filter combined with an id is rejected
      Given a complete connection context with a stored token
      When an agent runs "glassfrog actors per_abc --kind human"
      Then stderr will report that the filter applies only to the directory list
      And no API request will be sent
      And the command will exit with code 2

    # Source: 049-actor-read — Proposed: plan ADR-1 (the grown command still lists with no id)
    Scenario: The command with no id still lists the directory
      Given a complete connection context with a stored token
      And the organization has several actors
      When an agent runs "glassfrog actors"
      Then every actor will be printed as a list
      And the command will exit with code 0

  Rule: Tell a missing actor apart from a failed network
    # In order to tell "no such actor" apart from "the network failed" when a read fails,
    # as an operator diagnosing a failed run,
    # I want the command to surface the status-versus-transport distinction the shared seams already draw.

    # Source: 049-actor-read — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog actors per_abc"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no actor will be printed
      And the command will exit with code 2

    # Source: 049-actor-read — Scenario: A single read for an unknown id
    Scenario: An unknown id fails with the API status
      Given a complete connection context with a stored token
      And the API answers the actor read with a 404
      When an agent runs "glassfrog actors per_missing"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 049-actor-read — Scenario: A non-2xx status is surfaced, not classified
    @validation @wip
    Scenario: A non-2xx status is surfaced, not reinterpreted
      Given a complete connection context with a stored token
      And the API answers the actor read with a 404
      When an agent runs "glassfrog actors per_missing"
      Then the failure will carry the HTTP status as the shared error handling classifies it
      And the command will not turn it into a message of its own
