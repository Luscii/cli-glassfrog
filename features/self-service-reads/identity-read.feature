# Source: 011-identity-read — Scenario: me prints the resolved identity

Feature: Self-Service Reads — Identity Read
  Reading "what's mine" starts with knowing who the token resolves to.
  Identity Read is the `me` command: the smallest end-to-end read, proving the
  whole chain (connection, authentication, request execution) by printing the
  authenticated actor with its organization and membership, and optionally the
  roles it fills. It is the first call an agent makes to orient itself.
  (affects: AI agent, Practitioner)

  Rule: Confirm my token works and learn who I am and where I am
    # In order to confirm my token works and learn which actor and organization it resolves to,
    # as an AI agent making its first call,
    # I want to run one command that prints who I am and where I am.

    # Source: 011-identity-read — Scenario: me prints the resolved identity
    Scenario: The identity projection prints actor, organization, and access
      Given a complete connection context with a present, valid token
      And the API would return the actor "Alice Smith", organization "Acme", and access level "admin"
      When the operator runs the me command
      Then the projection will name the actor "Alice Smith" with its id and kind
      And it will name the organization "Acme" with its id
      And it will report the access level "admin"
      And the command will exit successfully

    # Source: 011-identity-read — Scenario: me distinguishes a human from an agent
    Scenario: An agent token is reported as an agent
      Given a complete connection context whose token resolves to an agent actor
      And the agent actor has an "agt_" id
      When the operator runs the me command
      Then the projection will report the actor kind as agent
      And it will surface the "agt_" id as the actionable handle

    # Source: 011-identity-read — Scenario: me resolves nothing itself
    @validation @wip
    Scenario: The command resolves nothing itself
      Given a complete connection context
      When the operator runs the me command
      Then it will use only the assembled context and the request-execution seam
      And it will read no flag, environment variable, or credentials file to build the request

    # Source: 011-identity-read — Scenario: no structured-JSON output leaks in
    @validation @wip
    Scenario: Output is the reshaped projection, not structured JSON
      Given a complete connection context with a present, valid token
      And the API would return a successful identity response
      When the operator runs the me command
      Then the output will be the reshaped identity projection
      And it will not be raw or structured JSON

    # Source: 011-identity-read — Scenario: the token value never appears in produced output
    @validation @wip
    Scenario: The token never appears in any output
      Given a complete connection context carrying the token "gf_live_secret123"
      When the me command produces any outcome, success or failure
      Then the token value "gf_live_secret123" will never appear in plaintext

  Rule: Get identity and my roles in a single read
    # In order to get identity and the roles I fill in a single round-trip,
    # as an AI agent about to act on a practitioner's behalf,
    # I want to opt into embedding my roles in the same `me` read.

    # Source: 011-identity-read — Scenario: me embeds roles on request
    Scenario: Requested roles are embedded in the projection
      Given a complete connection context with a present, valid token
      And the actor fills the roles "Marketing Lead" and "Treasurer"
      When the operator runs the me command with the roles embed requested
      Then the request will carry the include-roles query parameter
      And the projection will list each role's id and name alongside the identity facts

    # Source: 011-identity-read — Scenario: roles embed requested but the actor fills none
    Scenario: An empty roles embed omits the roles section
      Given a complete connection context whose actor fills no roles
      When the operator runs the me command with the roles embed requested
      Then the projection will print the identity facts
      And it will omit the roles section rather than printing an empty list

    # Source: 011-identity-read — Scenario: an unsupported include target is rejected before any request
    Scenario: An unsupported include target is rejected before any request
      Given a complete connection context with a present, valid token
      When the operator runs the me command requesting an include target the spec does not define
      Then the command will reject the input as a usage error naming the unsupported target
      And no request will reach the API

  Rule: Tell a bad token apart from a network failure
    # In order to tell "my token is bad" apart from "the network failed" when a command fails,
    # as an operator diagnosing a failed run,
    # I want the `me` command to surface the transport-versus-response distinction Request Execution already draws.

    # Source: 011-identity-read — Scenario: an unusable token surfaces a non-2xx outcome
    Scenario: An unusable token surfaces a non-2xx outcome
      Given a complete connection context whose token is expired or wrong
      And the API would answer with a 401 response
      When the operator runs the me command
      Then the command will surface the non-2xx outcome with its status code
      And it will print no identity projection
      And it will exit with a non-success result

    # Source: 011-identity-read — Scenario: a transport failure is surfaced as transport, not as a response
    Scenario: A network failure is surfaced as a transport outcome
      Given a complete connection context with a usable base URL
      And the API could not be reached
      When the operator runs the me command
      Then the command will surface a transport failure naming the cause
      And it will exit with a non-success result
      And it will not retry

    # Source: 011-identity-read — Scenario: no usable token — the fail-safe is propagated
    Scenario: A missing token is refused before sending
      Given a connection context with a usable base URL but no usable token
      When the operator runs the me command
      Then the authenticated transport's fail-safe will refuse the call
      And the command will exit with a non-success result
      And no unauthenticated request will be sent

    # Source: 011-identity-read — Scenario: a non-2xx status is surfaced, not classified
    @validation @wip
    Scenario: A non-2xx status is surfaced, not classified
      Given a complete connection context
      And the API would answer the me read with a 404 response
      When the operator runs the me command
      Then the command will surface the generic non-2xx outcome carrying the status
      And it will not turn it into a specific, interpreted API error message

    # Proposed (architecture-informed): plan ADR-4 — an undecodable 2xx is a distinct exit-code outcome the spec did not enumerate
    Scenario: An undecodable response is surfaced as an internal error
      Given a complete connection context with a present, valid token
      And the API would return a 200 response whose body does not match the identity shape
      When the operator runs the me command
      Then the command will surface a decode failure
      And it will exit with the internal-error result rather than a success

    # Proposed (architecture-informed): plan ADR-4 — a malformed base URL is a distinct exit-code outcome observable at the CLI surface
    Scenario: A malformed base URL is refused before sending
      Given a connection context carrying a base-URL error from a malformed configured value
      When the operator runs the me command
      Then the command will surface the base-URL problem as a usage error
      And the message will explain the next step to correct the configured base URL
      And no request will reach the API

    # Proposed (guard-derived): interface error table — the CredentialError outcome (exit 1) had no acceptance coverage (analyze K5)
    Scenario: A malformed credentials file fails the read loudly
      Given a connection context whose credentials file is malformed
      When the operator runs the me command
      Then the command will surface a credential-file error naming the file
      And the message will explain the next step to fix or re-create the file
      And it will exit with the internal-error result
