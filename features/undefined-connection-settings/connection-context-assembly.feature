Feature: Undefined Connection Settings — Connection Context Assembly
  Once the base URL and the token have each been resolved, the CLI still needs
  one bundle that pairs them — the single connection context every request
  hangs off. Assembly combines the resolved base URL with the discovered token,
  carries both outcomes forward (including absence or errors), and reports
  whether the context is complete — without deciding to refuse a call or pick
  an exit code. (affects: Practitioner)

  Rule: Pair endpoint and identity in one context
    # In order to point a command at the right endpoint as the right identity without wiring the two together by hand,
    # as an AI agent driving the CLI,
    # I want one connection context that already pairs the resolved base URL with the resolved token.

    # Source: 009-connection-context-assembly — Scenario: Complete context from a usable base URL and a present token
    @wip
    Scenario: A complete context pairs the resolved base URL and token
      Given the base URL resolved to "https://glassfrog.com/api/v5" from a config file
      And a token resolved from that config file
      When the CLI assembles the connection context
      Then it will carry the resolved base URL and its source
      And it will carry the resolved token and its source
      And it will report the context as complete

    # Source: 009-connection-context-assembly — Scenario: Built-in default base URL paired with a token still completes
    @wip
    Scenario: A built-in-default base URL with a token still completes
      Given the base URL resolved to the built-in default
      And a token resolved from the environment
      When the CLI assembles the connection context
      Then it will report the context as complete
      And it will report the base URL source as the built-in default

    # Source: 009-connection-context-assembly — Scenario: Assembly re-resolves nothing
    @validation @wip
    Scenario: Assembly re-resolves nothing
      Given base URL resolution and credential discovery are the only resolvers
      When the CLI assembles the connection context
      Then it will use only the outcomes those resolvers reported
      And it will read no flag, environment variable, or file itself

    # Source: 009-connection-context-assembly — Scenario: Assembly performs no writes and no network call
    @validation @wip
    Scenario: Assembly performs no writes and no network call
      Given any starting filesystem and network state
      When the CLI assembles the connection context
      Then the filesystem will be unchanged afterward
      And no outbound connection or API call will be made

  Rule: See what's ready and what's missing at a glance
    # In order to understand exactly what is and isn't ready before a call is attempted,
    # as an operator diagnosing a misconfigured setup,
    # I want the context to tell me whether it is complete and, if not, which part is missing or broken — all in one look.

    # Source: 009-connection-context-assembly — Scenario: No credentials — context still assembles, carrying the absence
    @wip
    Scenario: No credentials still assembles a context carrying the absence
      Given the base URL resolved to "https://glassfrog.com/api/v5"
      And no credentials were found
      When the CLI assembles the connection context
      Then it will carry the resolved base URL
      And it will carry a credential outcome of absent
      And it will report the context as incomplete naming the missing credential
      And it will not refuse the request or fabricate a token

    # Source: 009-connection-context-assembly — Scenario: Both inputs report a problem — both are surfaced
    @wip
    Scenario: A base-URL error and an absent credential are surfaced together
      Given base URL resolution reported a format error naming the flag
      And no credentials were found
      When the CLI assembles the connection context
      Then it will surface both the base-URL error and the absent credential
      And it will report the context as incomplete
      And it will not stop at the first problem

    # Source: 009-connection-context-assembly — Proposed: A credential error is carried into the context naming the file (interface: credential read/format error outcome)
    @wip
    Scenario: A credential error is carried into the context naming the file
      Given the base URL resolved to "https://glassfrog.com/api/v5"
      And credential discovery reported a read error naming a config file
      When the CLI assembles the connection context
      Then it will carry the credential error naming that file
      And it will report the context as incomplete naming the credential part

    # Source: 009-connection-context-assembly — Scenario: Base URL error while a token is present
    @wip
    Scenario: A base-URL error is carried while a present token is kept intact
      Given base URL resolution reported a read error naming a config file
      And a token resolved from the environment
      When the CLI assembles the connection context
      Then it will carry the base-URL error naming that file
      And it will keep the resolved token and its source
      And it will report the context as incomplete naming the base-URL part

    # Source: 009-connection-context-assembly — Scenario: Token is redacted from the rendered context
    @wip
    Scenario: The token is redacted when the context is rendered
      Given a complete context assembled with the token "gf_live_secret123"
      When the connection context is rendered for diagnostics
      Then it will show the credential source and path
      And the token value "gf_live_secret123" will not appear in the output

    # Source: 009-connection-context-assembly — Scenario: The token value never appears in produced output
    @validation @wip
    Scenario: The token value never appears in produced output
      Given any assembly outcome
      When the connection context's output and diagnostics are inspected
      Then the token value will never appear in plaintext

  Rule: Assemble once, reuse across calls
    # In order to keep the same identity and endpoint across every call a command makes,
    # as a practitioner running a multi-request command,
    # I want the context assembled once and reused, not re-derived per request.

    # Source: 009-connection-context-assembly — Scenario: One context applies across multiple calls in an invocation
    @wip
    Scenario: One context applies across every call in an invocation
      Given a connection context assembled for the invocation
      When a command makes more than one API request
      Then every request will use the same assembled context
      And the context will not be reassembled or re-resolved between requests

    # Source: 009-connection-context-assembly — Scenario: Assembly is deterministic
    @validation @wip
    Scenario: Assembly is identical across repeated runs
      Given an unchanged set of resolver outcomes
      When the CLI assembles the connection context twice
      Then both runs will produce the same context with the same readiness and sources
