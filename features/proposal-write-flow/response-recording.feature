# Source: 058-response-recording — Scenario: Record a no-objection response

Feature: Response Recording
  A proposal circulating for acceptance is decided by its circle members'
  consent-window responses. This is the consume/respond write of the governance
  write flow: `glassfrog proposal respond <prp-id> --response
  <no_objection|bring_to_meeting>` records one member's response to a circulating
  proposal and produces the recorded response, including the parent proposal's
  status — which reads `accepted` when this very response triggered
  auto-acceptance. The responding person is the token's own identity (never sent
  by the client); a missing or unsupported response value is refused before any
  request; recording is Premium-gated; and every failure exits with a named
  cause and the right exit code. It is a verb leaf under the `proposal` group
  (shared with `proposal create`/`list`/`get`).
  (affects: Practitioner)

  Rule: Let a circulating proposal pass to auto-acceptance
    # In order to let a circulating proposal pass to auto-acceptance,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to record a no_objection response with one command.

    # Source: 058-response-recording — Scenario: Record a no-objection response
    @wip
    Scenario: A no-objection response is recorded against the proposal
      Given a complete connection context with a stored token
      And the proposal "prp_0123" is circulating for acceptance
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection"
      Then the request will post the response to the proposal's responses endpoint
      And the request body will carry "value" set to "no_objection"
      And the recorded response will be printed with its "prr_" id and the proposal status
      And the command will exit with code 0

    # Source: 058-response-recording — Scenario: No usable credential
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 058-response-recording — Scenario: A second response by the same person is rejected by the server
    @wip
    Scenario: A second response by the same person is surfaced, not retried
      Given a complete connection context with a stored token
      And the responses endpoint answers that this person has already responded
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection"
      Then stderr will report that the recording failed and name the HTTP status
      And the recording will not be retried
      And the command will exit with a non-zero API-error code

    # Source: 058-response-recording — Scenario: Premium plan-gate is surfaced as a permission failure
    @wip
    Scenario: A Premium plan-gate is surfaced as a permission failure
      Given a complete connection context with a stored token
      And the responses endpoint answers that async proposals are not enabled
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection"
      Then stderr will report that the recording failed and name the HTTP status
      And no plan-limit-specific interpretation will be added
      And the command will exit with the permission code

    # Source: 058-response-recording — Scenario: Unknown or invisible proposal
    @wip
    Scenario: An unknown proposal fails with the API status
      Given a complete connection context with a stored token
      And no proposal "prp_ffff" is visible to the caller
      When an agent runs "glassfrog proposal respond prp_ffff --response no_objection"
      Then stderr will report that the recording failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 058-response-recording — Proposed: §133 (POST is non-idempotent, never auto-retried on 429)
    @wip
    Scenario: A rate-limited recording is surfaced, not silently re-sent
      Given a complete connection context with a stored token
      And the responses endpoint answers the recording with a rate-limit response
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection"
      Then the rate-limit will be surfaced on the first occurrence
      And the recording will not be retried, so no duplicate response is recorded
      And the command will exit with the rate-limit code

    # Source: 058-response-recording — Proposed: plan ADR-3 (reuse ContentType seam; send no If-Match)
    @wip
    Scenario: The write carries a JSON content type and no precondition
      Given a complete connection context with a stored token
      When an agent records a response with "glassfrog proposal respond prp_0123 --response no_objection"
      Then the request will be sent with a JSON content type
      And the request will carry no If-Match precondition

    # Source: 058-response-recording — Scenario: The response value is validated before any request
    @validation @wip
    Scenario: An unsupported response value sends no request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal respond prp_0123 --response abstain"
      Then stderr will report the unsupported value and list the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 058-response-recording — Scenario: The responding person is never supplied by the client
    @validation @wip
    Scenario: The responding person is derived from the token, never sent
      Given a complete connection context with a stored token
      When an agent records a response with "glassfrog proposal respond prp_0123 --response no_objection"
      Then the request body will carry no person field
      And the responding person on the recorded response will be the token's own identity

    # Source: 058-response-recording — Scenario: Recording does not reach into the read surface
    @validation @wip
    Scenario: Recording exposes no response reads or aggregation
      Given the implemented "glassfrog proposal respond" command
      When its behavior and help are inspected
      Then only recording a single response will be available
      And no reading or aggregating of responses will be advertised or implemented

  Rule: Keep a proposal from auto-accepting for live discussion
    # In order to keep a proposal from auto-accepting so it can be discussed live,
    # as a practitioner with a concern,
    # I want to record a bring_to_meeting response that blocks auto-acceptance.

    # Source: 058-response-recording — Scenario: Record a bring-to-meeting response
    @wip
    Scenario: A bring-to-meeting response is recorded against the proposal
      Given a complete connection context with a stored token
      And the proposal "prp_0123" is circulating for acceptance
      When an agent runs "glassfrog proposal respond prp_0123 --response bring_to_meeting"
      Then the request body will carry "value" set to "bring_to_meeting"
      And the recorded response will be printed
      And the command will exit with code 0

  Rule: Know whether the response closed the consent window
    # In order to know whether my response closed the consent window,
    # as an AI agent tracking the write flow,
    # I want the recorded response to carry the parent proposal's status.

    # Source: 058-response-recording — Scenario: A response that triggers auto-acceptance shows the accepted status
    @wip
    Scenario: A response that triggers auto-acceptance shows the accepted status
      Given a complete connection context with a stored token
      And the proposal "prp_0123" is awaiting only this member's response
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection --output json"
      Then the structured result will carry the parent proposal status as "accepted"
      And the command will exit with code 0

  Rule: Refuse an empty or invalid consent answer
    # In order to avoid an accidental or empty consent answer,
    # as an AI agent driving the CLI,
    # I want a missing or unsupported response value rejected before any request is made.

    # Source: 058-response-recording — Scenario: Missing response value is rejected before any request
    @wip
    Scenario: A missing response value is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal respond prp_0123"
      Then stderr will report that "--response" is required and list the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 058-response-recording — Scenario: Unsupported response value is rejected before any request
    @wip
    Scenario: An unsupported response value is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal respond prp_0123 --response maybe"
      Then stderr will report the unsupported value and list the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 058-response-recording — Proposed: interface dispatch (resolve --output before validating --response)
    @wip
    Scenario: An invalid output format is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal respond prp_0123 --response no_objection -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2
