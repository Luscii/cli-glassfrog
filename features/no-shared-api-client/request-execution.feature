# Source: 010-request-execution — Scenario: 2xx response decoded into a supplied target

Feature: No Shared API Client — Request Execution
  The CLI can assemble a connection context but has no shared way to issue a
  request — so every endpoint command would reinvent transport plumbing.
  Request Execution is the single seam each command calls through: it sends an
  authenticated request and returns a parsed response or a typed transport
  error, leaving classification, paging, and backoff to its siblings.
  (affects: AI agent, Practitioner, Maintainer)

  Rule: One seam returns a parsed response or a typed error
    # In order to issue a Glassfrog API call without re-wiring base URL, identity, timeouts, and response handling for every command,
    # as an AI agent building an endpoint command,
    # I want one seam I can hand a request to and get back a parsed response or a typed error.

    # Source: 010-request-execution — Scenario: 2xx response decoded into a supplied target
    @wip
    Scenario: A successful response is decoded into the caller's target
      Given a client had been built from a complete connection context
      And the API would return a 200 response with a JSON body
      When an endpoint command executes a request with a decode target
      Then the response status and headers will be returned
      And the body will be decoded into the target

    # Source: 010-request-execution — Scenario: 2xx response with no decode target (bodyless)
    @wip
    Scenario: A bodyless success returns status and headers without decoding
      Given a client had been built from a complete connection context
      And the API would return a 204 response with no body
      When an endpoint command executes a request with no decode target
      Then the response status and headers will be returned
      And no body will be decoded

    # Source: 010-request-execution — Scenario: identity is carried by the authenticated transport
    @wip
    Scenario: The request is authenticated by the transport
      Given a client had been built from a connection context carrying a present token
      When an endpoint command executes a request
      Then the outgoing request will carry the X-Auth-Token header attached by the transport
      And the client will not attach the header itself

    # Source: 010-request-execution — Scenario: Request Execution re-resolves nothing
    @validation @wip
    Scenario: Execution uses only the connection context and transport
      Given a client had been built from a connection context
      When an endpoint command executes a request
      Then it will use only the context's base URL and the transport's identity
      And it will read no flag, environment variable, or credentials file itself

  Rule: Failures are surfaced as distinct, typed outcomes
    # In order to distinguish "the network failed" from "the API answered, but not with success,"
    # as an operator diagnosing a failed command,
    # I want transport failures, non-2xx responses, and decode failures surfaced as distinct, typed outcomes.

    # Source: 010-request-execution — Scenario: transport failure at the wire
    @wip
    Scenario: A wire failure is surfaced as a transport error
      Given a client had been built from a complete connection context
      And the API could not be reached
      When an endpoint command executes a request
      Then a transport error naming the failure will be returned
      And no response or decoded body will be returned

    # Source: 010-request-execution — Scenario: non-2xx response is short-circuited
    @wip
    Scenario: A non-2xx response is surfaced as a generic response error
      Given a client had been built from a complete connection context
      And the API would return a 403 response with a body
      When an endpoint command executes a request with a decode target
      Then a response error carrying the status, headers, and raw body will be returned
      And the body will not be decoded into the target
      And the error will not be classified by failure kind

    # Source: 010-request-execution — Scenario: base-URL problem refuses before sending
    @wip
    Scenario: A base-URL problem is refused before sending
      Given a connection context had carried a base-URL error
      When an endpoint command builds a client for the request
      Then the base-URL error will be returned
      And no client will be built and no request will reach the API

    # Source: 010-request-execution — Scenario: 2xx body cannot be decoded into the supplied target
    @wip
    Scenario: An undecodable success body is surfaced as a decode error
      Given a client had been built from a complete connection context
      And the API would return a 200 response whose body does not match the target
      When an endpoint command executes a request with a decode target
      Then a decode error will be returned rather than a success

    # Source: 010-request-execution — Scenario: hung connection fails on the request timeout
    @wip
    Scenario: A hung connection fails on the request timeout
      Given a client had been built from a complete connection context
      And the API accepts the connection but never responds
      When an endpoint command executes a request
      Then the request timeout will elapse
      And a transport error will be returned without a retry

    # Source: 010-request-execution — Scenario: no usable token — the transport's fail-safe is propagated
    @wip
    Scenario: A missing token refuses the request without sending
      Given a client had been built from a connection context with a usable base URL but no token
      When an endpoint command executes a request
      Then the transport's authentication failure will be propagated
      And no unauthenticated request will be sent

    # Source: 010-request-execution — Scenario: exactly one send attempt per request
    @validation @wip
    Scenario: Execution makes exactly one attempt
      Given a client had been built from a complete connection context
      When an endpoint command executes a request
      Then exactly one outbound attempt will be made
      And it will not retry, back off, or follow paging

    # Source: 010-request-execution — Scenario: the token value never appears in produced output
    @validation @wip
    Scenario: The token never appears in any output
      Given a client had been built from a connection context carrying the token "gf_live_secret123"
      When a request produces any outcome or error
      Then the token value "gf_live_secret123" will never appear in plaintext

    # Source: 010-request-execution — Scenario: a non-2xx body is never decoded into the success target
    @validation @wip
    Scenario: A non-2xx body is never decoded into the success target
      Given a decode target had been supplied
      When the API returns a non-2xx response
      Then the body will be carried raw on the response error
      And it will never be decoded into the success target

  Rule: Status and headers are exposed for the sibling capabilities
    # In order to build paging and rate-limit handling on top without reinventing transport,
    # as a maintainer adding the sibling capabilities,
    # I want the status code and response headers exposed on both the success and the non-2xx outcome.

    # Source: 010-request-execution — Scenario: 429 is surfaced as a non-2xx carrying its rate-limit headers
    @wip
    Scenario: A 429 carries its rate-limit headers
      Given a client had been built from a complete connection context
      And the API would return a 429 rate-limit response
      When an endpoint command executes a request
      Then a response error carrying the 429 status and rate-limit headers will be returned
      And the client will not sleep, back off, or retry
