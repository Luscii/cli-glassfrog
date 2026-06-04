# Source: 007-request-authentication — Scenario: Resolved token is attached to the outgoing call

Feature: Unauthenticated Access — Request Authentication
  The CLI has no way to prove it's acting as a specific org + person, so
  Glassfrog can't authorize its calls. Once the caller's token is resolved,
  the CLI must attach it as the X-Auth-Token header on every outgoing API
  call — and refuse to call the API at all when no usable credential exists,
  rather than sending an anonymous request. (affects: AI agent, Practitioner)

  Rule: Attach my identity to every API call automatically
    # In order to have my CLI commands authorized as me without managing headers by hand,
    # as a practitioner who stored credentials once,
    # I want the CLI to attach my resolved identity to every API call automatically.

    # Source: 007-request-authentication — Scenario: Resolved token is attached to the outgoing call
    Scenario: The resolved token is attached to the request
      Given Credential Discovery resolved the token "gf_resolved_token" from a source
      When the CLI sends an API request
      Then the request will carry an "X-Auth-Token" header set to "gf_resolved_token"
      And the request will be sent to the API

    # Source: 007-request-authentication — Scenario: Token is attached verbatim
    Scenario: The token is attached verbatim
      Given Credential Discovery resolved a token containing unusual characters
      When the CLI sends an API request
      Then the "X-Auth-Token" header value will exactly equal the resolved token
      And no characters will be added, removed, or re-encoded

    # Source: 007-request-authentication — Scenario: The same identity applies across multiple calls in one invocation
    Scenario: Every request in an invocation carries the same identity
      Given Credential Discovery resolved the token "gf_resolved_token"
      When the CLI sends more than one API request in a single invocation
      Then every request will carry the same "X-Auth-Token" identity

    # Source: 007-request-authentication — Proposed: The credential is resolved once per invocation (plan: resolve-once cache)
    Scenario: The credential is resolved once per invocation
      Given Credential Discovery was available as the credential source
      When the CLI sends more than one API request in a single invocation
      Then the credential will be resolved only once
      And every request will reuse the resolved identity

    # Source: 007-request-authentication — Scenario: Authentication performs no resolution of its own
    @validation @wip
    Scenario: Authentication performs no resolution of its own
      Given Credential Discovery was the only credential source
      When the CLI authenticates an API request
      Then the attached token will come only from Credential Discovery's output
      And the CLI will not read the environment or any file directly

  Rule: Refuse to call the API when no credential is available
    # In order to fail safely in automation instead of issuing anonymous calls,
    # as an AI agent,
    # I want the CLI to refuse to reach the API when no credential is available, and tell me why.

    # Source: 007-request-authentication — Scenario: No credentials — refuse to call
    Scenario: A missing credential refuses the call
      Given Credential Discovery reported that no credentials were found
      When the CLI prepares an API request
      Then the request will not be sent
      And the CLI will report that it cannot authenticate because no credentials were found

    # Source: 007-request-authentication — Scenario: Credential error — refuse to call and name the cause
    Scenario: A broken credential fails loudly without sending
      Given a ".glassfrogrc" existed but could not be read or parsed
      And Credential Discovery reported a credential error naming that file
      When the CLI prepares an API request
      Then the request will not be sent
      And the CLI will report a credential error naming that file
      And the outcome will be distinct from a no-credentials outcome

    # Source: 007-request-authentication — Scenario: No request is ever sent unauthenticated
    @validation @wip
    Scenario: No request is ever sent unauthenticated
      Given any credential state
      When the CLI runs to completion
      Then no API request will have been sent without an "X-Auth-Token" header
      And an absent credential will end in a cannot-authenticate outcome

  Rule: Report which identity a command ran as
    # In order to confirm which identity a command ran as,
    # as an operator who moves between projects and tokens,
    # I want the CLI to report the credential source used — never the secret itself.

    # Source: 007-request-authentication — Scenario: Active identity source is reported without exposing the secret
    Scenario: The active identity source is reported without the secret
      Given Credential Discovery resolved a token from the file "/home/dev/.glassfrogrc"
      When the CLI authenticates an API request
      Then the CLI will report the active identity source as that file path
      And the token value will not appear in the output

    # Source: 007-request-authentication — Scenario: Token is redacted from request diagnostics
    Scenario: The token is redacted from request diagnostics
      Given request diagnostics were enabled
      And Credential Discovery resolved a token
      When the CLI sends an API request carrying the "X-Auth-Token" header
      Then the token value will be redacted from the diagnostic output

    # Source: 007-request-authentication — Scenario: The token value never appears in produced output
    @validation @wip
    Scenario: The token value never appears in output
      Given any authentication outcome
      When the produced output and error messages are inspected
      Then the token value will not appear in them
