# Source: 057-advance-to-circulation — Scenario: Advance a draft proposal into circulation

Feature: Advance to Circulation
  A proposal is the only sanctioned path to alter governance structure, and it
  must be advanced from `draft` into circulation before the circle can respond.
  This is the `propose` transition of the write flow: `glassfrog proposal
  propose <prp-id>` issues a bodyless POST to the proposal's propose transition
  and prints the advanced proposal — now `proposed_outside_meeting`, carrying the
  server-set response deadline, the proposer's implicit no-objection, and the
  updated available transitions. It is a flagless verb leaf under the `proposal`
  group (shared with `proposal list`/`get`/`create`). The transition is
  Premium-gated and server-authorized: the command issues the POST and lets the
  server enforce, surfacing a 404 (no such proposal) or a 422 (transition not
  allowed) as a real failure — never as success — and routing the Premium 403
  through the shared error handling with no plan-specific message.
  (affects: Practitioner)

  Rule: Advance a draft into circulation by id
    # In order to start the consent window on a proposal I have drafted,
    # as a practitioner operating the governance write flow,
    # I want to advance it into circulation by id with one command.

    # Source: 057-advance-to-circulation — Scenario: Advance a draft proposal into circulation
    @wip
    Scenario: A draft proposal is advanced into circulation
      Given a complete connection context with a stored token
      And a draft proposal "prp_0123" whose available transitions include "propose"
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then the request will POST to the proposal's propose transition with no body
      And the advanced proposal will be printed with status "proposed_outside_meeting"
      And the command will exit with code 0

    # Source: 057-advance-to-circulation — Scenario: Transition not allowed is a failure
    @wip
    Scenario: A disallowed transition fails with the API status
      Given a complete connection context with a stored token
      And a proposal "prp_0123" for which "propose" is not currently allowed
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then the API will answer 422
      And stderr will report that the advance failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 057-advance-to-circulation — Scenario: Proposal id does not exist
    @wip
    Scenario: An unknown proposal id fails with the API status
      Given a complete connection context with a stored token
      And no proposal "prp_ffff" exists
      When a practitioner runs "glassfrog proposal propose prp_ffff"
      Then the API will answer 404
      And stderr will report that the advance failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 057-advance-to-circulation — Scenario: No credential surfaces the not-authenticated outcome
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 057-advance-to-circulation — Scenario: Missing proposal id is rejected before any request
    @wip
    Scenario: A missing proposal id is rejected before any request
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog proposal propose"
      Then stderr will report a usage error naming the required proposal id
      And no request will be sent
      And the command will exit with code 2

    # Source: 057-advance-to-circulation — Scenario: Premium not enabled surfaces a plain refusal
    @wip
    Scenario: A Premium-gated refusal surfaces plainly
      Given a complete connection context with a stored token
      And the organization does not have async proposals enabled
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then the API will answer 403
      And stderr will report that the advance failed and name the HTTP status
      And no plan-specific message will be added
      And the command will exit with a non-zero permission code

    # Source: 057-advance-to-circulation — Scenario: A transport failure surfaces network-unavailable
    @wip
    Scenario: A transport failure surfaces network-unavailable
      Given a complete connection context with a stored token
      And the API endpoint is unreachable
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then stderr will report the transport failure by name
      And the command will exit with a non-zero network-unavailable code

    # Source: 057-advance-to-circulation — Proposed: rate-limited advance is surfaced, not silently retried (plan Cross-cutting: non-idempotent retry)
    @wip
    Scenario: A rate-limited advance is surfaced, not silently retried
      Given a complete connection context with a stored token
      And the API answers the propose request with 429
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then stderr will report that the advance failed and name the HTTP status
      And the POST will not be automatically retried
      And the command will exit with a non-zero rate-limit code

    # Source: 057-advance-to-circulation — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    @wip
    Scenario: An invalid output format is rejected before any advance request
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog proposal propose prp_0123 -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Produce the advanced proposal as parseable data
    # In order to move a created proposal toward acceptance from an automated pipeline,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want the advance to produce the resulting proposal as structured data I can parse,
    # including its new status and response deadline.

    # Source: 057-advance-to-circulation — Scenario: Advanced proposal rendered as JSON
    @wip
    Scenario: The advanced proposal is rendered as JSON
      Given a complete connection context with a stored token
      And a draft proposal "prp_0123" that can be proposed
      When an agent runs "glassfrog proposal propose prp_0123 -o json"
      Then the returned proposal will be printed as JSON
      And the command will exit with code 0

    # Source: 057-advance-to-circulation — Scenario: The result reflects the server-set deadline and implicit response
    @wip
    Scenario: The advanced proposal carries the deadline and implicit response
      Given a complete connection context with a stored token
      And a draft proposal "prp_0123" that can be proposed
      When an agent runs "glassfrog proposal propose prp_0123"
      Then the printed proposal will carry the server-set response deadline
      And its response summary will reflect the proposer's implicit no-objection
      And the command will not narrate the notification side effects

    # Source: 057-advance-to-circulation — Scenario: The result is the server's proposal, unembellished
    @validation @wip
    Scenario: The advanced result is the server's proposal unembellished
      Given a successful "glassfrog proposal propose prp_0123" run
      When the rendered proposal is inspected in every format
      Then it will carry only fields the server returned
      And it will not fabricate any side-effect narration

    # Source: 057-advance-to-circulation — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: The advance output carries no raw API envelope under a human format
      Given a successful "glassfrog proposal propose prp_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" JSON envelope

  Rule: Trust the server to authorize the transition
    # In order to trust that an advance only happens when the server allows it,
    # as a practitioner,
    # I want the command to issue the transition and surface the server's refusal plainly
    # rather than guessing client-side whether it is permitted.

    # Source: 057-advance-to-circulation — Scenario: The advance issues exactly one transition request and reads nothing first
    @validation @wip
    Scenario: The advance issues one request and reads nothing first
      Given a complete connection context with a stored token
      And a draft proposal "prp_0123" that can be proposed
      When a practitioner runs "glassfrog proposal propose prp_0123"
      Then exactly one request will be sent to the propose transition
      And no prior read of the proposal will be issued

    # Source: 057-advance-to-circulation — Scenario: A 404 and a 422 are real failures
    @validation @wip
    Scenario: A 404 and a 422 are surfaced as real failures
      Given the advance is run against a proposal the API answers 404 for, and one it answers 422 for
      When each run completes
      Then neither run will produce a success result
      And both runs will exit with a non-zero code
