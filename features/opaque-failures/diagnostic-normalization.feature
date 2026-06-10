# Source: 031-diagnostic-normalization — Scenario: Transport failure normalized to network-unavailable

Feature: Opaque Failures — Diagnostic Normalization
  When a Glassfrog call fails, the caller can't tell what went wrong or what to
  do next. The CLI surfaces failures in family-specific shapes — transport and
  decode errors, typed API errors, and usage errors. Diagnostic Normalization
  collapses them into one diagnostic carrying a cause, a category, and (where one
  exists) a next step, so every failure is legible and an agent can branch on it.
  (affects: AI agent, Practitioner)

  Rule: A failure is handed back as one diagnostic with a cause, a category, and a next step
    # In order to decide what to do after a failure instead of decoding a status and body myself,
    # as an AI agent driving a command,
    # I want every failure handed back as one diagnostic carrying a cause, a category, and a next step.

    # Source: 031-diagnostic-normalization — Scenario: Transport failure normalized to network-unavailable
    Scenario: Transport failure is normalized to network-unavailable
      Given a command's request had failed with a connection-refused transport error
      When the failure is normalized
      Then the diagnostic's category will be network-unavailable
      And the cause will name that the API could not be reached
      And the next step will point the caller to check connectivity and the configured endpoint

    # Source: 031-diagnostic-normalization — Scenario: Permission failure carries the API's own detail
    Scenario: A permission failure carries the API's own detail
      Given a typed API error had HTTP status 403 with a detail of "You are not a member of this circle"
      When the failure is normalized
      Then the diagnostic's category will be permission/authorization
      And the cause will be the API's detail text
      And the next step will point the caller toward the required membership or permission

    # Source: 031-diagnostic-normalization — Scenario: Usage error normalized from dispatch
    @wip
    Scenario: A usage error is normalized from dispatch
      Given a usage error had reported the unknown command "rolez"
      When the failure is normalized
      Then the diagnostic's category will be usage error
      And the cause will name the unrecognized token
      And the next step will point the caller to the command's help

    # Source: 031-diagnostic-normalization — Scenario: API error with no readable detail derives its cause from the status
    Scenario: An API error without a readable detail derives its cause from the status
      Given a typed API error had HTTP status 500 with no detail or title
      When the failure is normalized
      Then the diagnostic's category will be general API error
      And the cause will be derived from the HTTP status rather than invented
      And no fabricated next step will be attached

    # Source: 031-diagnostic-normalization — Scenario: Rate-limited surfaced after retries are exhausted
    @wip
    Scenario: A rate-limited response surfaced after retries carries a reset-window next step
      Given a typed API error had HTTP status 429 surfaced after rate-limit retries were exhausted
      When the failure is normalized
      Then the diagnostic's category will be rate-limited
      And the next step will point the caller to wait for the rate-limit window to reset and retry
      And no additional wait or retry will be performed

    # Source: 031-diagnostic-normalization — Proposed: 401 and 403 carry distinct next steps (interface-spec next-step contract)
    @wip
    Scenario: A 401 and a 403 carry distinct next steps
      Given one typed API error had status 401 and another had status 403
      When each failure is normalized
      Then both diagnostics will share the permission/authorization category
      And the 401 next step will point the caller to verify the configured API token
      And the 403 next step will point the caller to the required role membership or permission

    # Source: 031-diagnostic-normalization — Scenario: The 403 boundary holds — no plan guidance leaks in
    @validation @wip
    Scenario: A 403 diagnostic carries no plan-availability guidance
      Given a typed API error had HTTP status 403
      When the failure is normalized
      Then the diagnostic will contain no plan- or Premium-availability language
      And the category will be permission/authorization

  Rule: A failure carries a category drawn from one fixed vocabulary
    # In order to branch reliably on the kind of failure (back off, fix input, escalate, give up),
    # as an operator,
    # I want every failure to carry a category drawn from one fixed vocabulary.

    # Source: 031-diagnostic-normalization — Scenario: Undecodable 2xx body normalized to general API error
    @wip
    Scenario: An undecodable 2xx body is normalized to a general API error
      Given a decode error had occurred on a 2xx response whose body would not decode
      When the failure is normalized
      Then the diagnostic's category will be general API error
      And the cause will name that the API responded but its body could not be read as expected

    # Source: 031-diagnostic-normalization — Scenario: Most-specific category wins on an overlapping status
    @wip
    Scenario: The most-specific category wins on an overlapping status
      Given a typed API error had status 429
      When the failure is normalized
      Then the diagnostic's category will be rate-limited
      And the category will not be general API error

    # Source: 031-diagnostic-normalization — Proposed: a decode error exits with the general API code (plan ADR-2)
    @wip
    Scenario: A decode error exits with the general API code
      Given a command had received a 2xx response whose body would not decode
      When the command maps the failure to an exit code
      Then the command will exit with code 3

  Rule: Every failure is collapsed into a single consistent diagnostic shape
    # In order to write one failure-handling path instead of one per failure source,
    # as a maintainer building endpoint commands,
    # I want transport, decode, API, and usage failures collapsed into a single consistent diagnostic shape.

    # Source: 031-diagnostic-normalization — Scenario: A success is never normalized
    Scenario: A successful outcome is never normalized
      Given a successful 2xx outcome had reached the normalizer
      When it is processed
      Then no diagnostic will be produced
      And the success will pass through untouched

    # Source: 031-diagnostic-normalization — Scenario: An unrecognized failure falls back to the internal-error diagnostic
    Scenario: An unrecognized failure falls back to the internal-error diagnostic
      Given a failure the normalizer cannot map to a transport, decode, typed-API, or usage family
      When the failure is normalized
      Then a diagnostic with the internal-error category will be produced
      And the cause will be the failure's own message
      And no stack trace will be written

    # Source: 031-diagnostic-normalization — Scenario: One consistent shape across every failure family
    @validation @wip
    Scenario: Every failure family produces the same diagnostic shape
      Given a transport error, a typed API error, and a usage error
      When each is normalized
      Then each diagnostic will expose the same cause, category, and next-step fields
      And each category will be drawn from the one fixed vocabulary

    # Source: 031-diagnostic-normalization — Scenario: No implementation leakage in the artifact
    @validation @wip
    Scenario: A diagnostic exposes only its observable fields
      Given any normalized failure
      When the diagnostic is inspected
      Then it will expose only a cause, a category, and a next step
      And it will expose no internal layout or implementation detail

    # Source: 031-diagnostic-normalization — Proposed: no diagnostic output carries the token (token-free invariant)
    @validation @wip
    Scenario: No diagnostic output carries the auth token
      Given any failure had been normalized
      When the diagnostic's cause, category, and next step are inspected
      Then none of them will contain the X-Auth-Token value
