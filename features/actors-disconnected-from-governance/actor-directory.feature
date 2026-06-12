# Source: 048-actor-directory — Scenario: List every actor in the organization

Feature: Actors Disconnected from Governance — Actor Directory
  When working a tension, the operator can't bridge people and the governance
  record in either direction: from a role to the actor to contact about it, or
  from an actor to the governance they hold. Actor Directory is the discovery
  entry — `glassfrog actors` lists the people and agents in the organization,
  narrowed by kind, by the role an actor fills, or by free-text, so the operator
  can turn a name or a role into the per_/agt_ id the rest of the slice drills
  into. The list walks every page to completion by default, or is plainly
  flagged incomplete. It renders through the shared output seam or fails with a
  named error and the right exit code.
  (affects: Practitioner, AI agent)

  Rule: Find whom to contact for a role
    # In order to find whom to contact about a tension once I know the role,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list the actors filling a given role by its id.

    # Source: 048-actor-directory — Scenario: Find the actors filling a role
    Scenario: A role-id filter lists the actors filling that role
      Given a complete connection context with a stored token
      And a role that two actors fill
      When an agent runs "glassfrog actors --role-id role_abc"
      Then the request will carry "role_id" set to "role_abc"
      And both actors will be printed as a list
      And the command will exit with code 0

    # Source: 048-actor-directory — Scenario: Malformed role-id filter is rejected by the API
    Scenario: A malformed role-id filter fails with the API status
      Given a complete connection context with a stored token
      And the API cannot parse the submitted role-id filter
      When an agent runs "glassfrog actors --role-id not-a-role"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 048-actor-directory — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog actors"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no actors will be printed
      And the command will exit with code 2

  Rule: Turn a name into an id to drill into
    # In order to turn a half-remembered name into the id I can drill into,
    # as a practitioner bridging people and the governance record,
    # I want to find an actor by free-text search across the directory.

    # Source: 048-actor-directory — Scenario: List every actor in the organization
    Scenario: A directory lists every actor in the organization
      Given a complete connection context with a stored token
      And the organization has several actors
      When an agent runs "glassfrog actors"
      Then the request will carry no kind, role, or query filter
      And every actor will be printed as a list
      And the command will exit with code 0

    # Source: 048-actor-directory — Scenario: No actor matches the filters
    Scenario: A free-text query matching no actor is a clean success
      Given a complete connection context with a stored token
      And no actor's name matches "zzzznomatch"
      When an agent runs "glassfrog actors --query zzzznomatch"
      Then the request will carry "q" set to "zzzznomatch"
      And "no actors" will be printed to stdout
      And the command will exit with code 0

    # Source: 048-actor-directory — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog actors" run under the default human format
      When the output is inspected
      Then it will show the reshaped actor projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Tell people apart from agents
    # In order to tell automation apart from people before I act,
    # as an AI agent assembling context,
    # I want to narrow the directory to just humans or just agents.

    # Source: 048-actor-directory — Scenario: Narrow the directory to agents
    Scenario: A kind filter narrows the directory to agents
      Given a complete connection context with a stored token
      And the organization has people and agents
      When an agent runs "glassfrog actors --kind agent"
      Then the request will carry "kind" set to "agent"
      And only the agents will be printed as a list
      And the command will exit with code 0

    # Source: 048-actor-directory — Scenario: Unsupported kind value is rejected before any request
    Scenario: An unsupported kind is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog actors --kind robot"
      Then stderr will report the unsupported value and list the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 048-actor-directory — Scenario: An unsupported kind costs no request
    @validation @wip
    Scenario: A rejected kind issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog actors --kind robot"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

    # Source: 048-actor-directory — Scenario: Agent discovery does not require the gated alias
    @validation @wip
    Scenario: Agent discovery reaches the ungated unified endpoint
      Given a token without the ai_integration feature
      When an agent runs "glassfrog actors --kind agent"
      Then the request will read the unified "/actors" endpoint with "kind" set to "agent"
      And it will not route through the ai_integration-gated "/agents" alias

  Rule: Trust the directory is whole, or be told it is incomplete
    # In order to trust I'm acting on the whole directory, not a silently truncated slice,
    # as an AI agent with a bounded context,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 048-actor-directory — Scenario: Paginated directory with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the organization's actors span more than one page
      When an agent runs "glassfrog actors --first-page"
      Then only the first page of actors will be printed
      And stderr will note that more actors exist
      And the command will exit with code 0

    # Source: 048-actor-directory — Proposed: plan Cross-cutting (025 ADR-3 walk-by-default)
    Scenario: A multi-page directory walks to completion by default
      Given a complete connection context with a stored token
      And the organization's actors span more than one page
      When an agent runs "glassfrog actors"
      Then every page will be walked and the complete set of actors will be printed
      And the command will exit with code 0

    # Source: 048-actor-directory — Proposed: plan Cross-cutting (mid-walk failure, 025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the page walk fails after retrieving the first page
      When an agent runs "glassfrog actors"
      Then the actors retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 048-actor-directory — Proposed: plan Risk (filters carried on every page of the walk)
    @validation @wip
    Scenario: The filters are carried on every page of the walk
      Given a complete connection context with a stored token
      And actors filtered by "--kind human" span more than one page
      When an agent runs "glassfrog actors --kind human"
      Then every page request of the walk will retain "kind" set to "human"
      And the command will exit with code 0
