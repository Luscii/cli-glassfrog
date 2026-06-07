# Source: 013-my-actions — Scenario: list the practitioner's actions

Feature: Self-Service Reads — My Actions
  Reading "what's mine" includes the next-actions on my plate.
  My Actions is the `my actions` command: a token-scoped read that lists the
  actions owned by the roles the practitioner fills, optionally filtered by
  status. It fetches the first page, renders the shared reshaped projection,
  and signals when more results exist.
  (affects: Practitioner, AI agent)

  Rule: See the next-actions on my plate
    # In order to see the next-actions on my plate without opening the Glassfrog web app,
    # as a practitioner (usually via an AI agent),
    # I want to list the actions owned by the roles I fill.

    # Source: 013-my-actions — Scenario: list the practitioner's actions
    @wip
    Scenario: The actions projection lists the practitioner's actions
      Given a complete connection context with a present, valid token
      And the API would return one page of actions the practitioner owns
      When the operator runs the my-actions command with no status filter
      Then the request will go to the my-actions endpoint
      And the projection will list each action with its id, status, description, and owning role
      And the command will exit successfully

    # Source: 013-my-actions — Scenario: no matching actions
    @wip
    Scenario: No matching actions reports an empty list, not a failure
      Given a complete connection context with a present, valid token
      And the practitioner owns no actions matching the request
      When the operator runs the my-actions command
      Then the projection will report an empty list
      And the command will exit successfully

    # Source: 013-my-actions — Scenario: no usable token
    @wip
    Scenario: A missing token is refused before sending
      Given a connection context with a usable base URL but no usable token
      When the operator runs the my-actions command
      Then the authenticated transport's fail-safe will refuse the call
      And the command will exit with a non-success result
      And no unauthenticated request will be sent

    # Source: 013-my-actions — Scenario: API responds with a non-2xx
    @wip
    Scenario: A non-2xx response is surfaced, not classified
      Given a complete connection context with a present, valid token
      And the API would answer the my-actions read with a non-2xx response
      When the operator runs the my-actions command
      Then the command will surface the generic non-2xx outcome carrying the status
      And it will not turn it into a specific, interpreted API error message
      And it will exit with a non-success result

    # Proposed (architecture-informed): plan ADR/Cross-cutting — a wire failure is a distinct transport outcome
    @wip
    Scenario: A network failure is surfaced as a transport outcome
      Given a complete connection context with a usable base URL
      And the API could not be reached
      When the operator runs the my-actions command
      Then the command will surface a transport failure naming the cause
      And it will exit with a non-success result
      And it will not retry

    # Proposed (architecture-informed): reused 011 error mapping — an undecodable 2xx is a distinct internal-error outcome
    @wip
    Scenario: An undecodable response is surfaced as an internal error
      Given a complete connection context with a present, valid token
      And the API would return a 200 response whose body does not match the actions shape
      When the operator runs the my-actions command
      Then the command will surface a decode failure
      And it will exit with the internal-error result rather than a success

    # Proposed (architecture-informed): reused 011 error mapping — a malformed base URL is observable at the CLI surface
    @wip
    Scenario: A malformed base URL is refused before sending
      Given a connection context carrying a base-URL error from a malformed configured value
      When the operator runs the my-actions command
      Then the command will surface the base-URL problem as a usage error
      And the message will explain the next step to correct the configured base URL
      And no request will reach the API

    # Source: 013-my-actions — Scenario: the command re-resolves nothing
    @validation @wip
    Scenario: The command resolves nothing itself
      Given a complete connection context
      When the operator runs the my-actions command
      Then it will use only the assembled context and the request-execution seam
      And it will read no flag, environment variable, or credentials file to build the request

    # Source: 013-my-actions — Scenario: output is the shared projection, not raw payload
    @validation @wip
    Scenario: Output is the reshaped projection, not structured JSON
      Given a complete connection context with a present, valid token
      And the API would return a successful page of actions
      When the operator runs the my-actions command
      Then the output will be the reshaped actions projection
      And it will not be raw or structured JSON

    # Source: 013-my-actions — (cross-read invariant) the token never appears in output
    @validation @wip
    Scenario: The token never appears in any output
      Given a complete connection context carrying the token "gf_live_secret123"
      When the my-actions command produces any outcome, success or failure
      Then the token value "gf_live_secret123" will never appear in plaintext

  Rule: Focus on the work that is live
    # In order to focus on just the work that is live,
    # as a practitioner triaging my actions,
    # I want to filter the list by status (e.g. only current).

    # Source: 013-my-actions — Scenario: filter by a supported status
    @wip
    Scenario: A supported status filters the request
      Given a complete connection context with a present, valid token
      When the operator runs the my-actions command with the status filter "current"
      Then the value "current" will be accepted as a supported status
      And the request will carry the status filter "current"
      And only the actions the API returns for that filter will be rendered

    # Source: 013-my-actions — Scenario: invalid status value is rejected before any request
    @wip
    Scenario: An unsupported status value is rejected before any request
      Given a complete connection context with a present, valid token
      When the operator runs the my-actions command with a status value outside the spec's vocabulary
      Then the command will reject the input as a usage error naming the unsupported value and the supported set
      And no request will reach the API

    # Source: 013-my-actions — Scenario: an unsupported status costs no request
    @validation @wip
    Scenario: An unsupported status costs no request
      Given a complete connection context
      When the operator runs the my-actions command with an unsupported status value
      Then the command will reject it before assembling the connection or sending a request
      And a transport tripwire will confirm no request was issued

  Rule: Know when more results exist
    # In order to know when a result is incomplete rather than silently truncated,
    # as an AI agent consuming the output,
    # I want a clear signal when more results exist beyond the page I received.

    # Source: 013-my-actions — Scenario: more results than one page
    @wip
    Scenario: A further page is signalled, not fetched
      Given a complete connection context with a present, valid token
      And the API would return a first page reporting that more results are available
      When the operator runs the my-actions command
      Then the projection will render only the first page
      And it will surface a clear "more results are available" signal
      And it will not request a second page

    # Source: 013-my-actions — Scenario: exactly one page request
    @validation @wip
    Scenario: Exactly one page request is made
      Given a complete connection context with a present, valid token
      And the API would return a page reporting that more results are available
      When the operator runs the my-actions command
      Then exactly one request will be made to the my-actions endpoint
      And no subsequent page will be fetched
