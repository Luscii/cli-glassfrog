# Source: 059-withdraw-proposal — Scenario: Withdraw a circulating proposal back to draft

Feature: Withdraw Proposal
  A circulating proposal can be pulled back off the consent window and returned
  to `draft` for re-editing — the mirror of the `propose` transition. This is the
  `withdraw` transition of the write flow: `glassfrog proposal withdraw <prp-id>`
  issues a bodyless POST to the proposal's withdraw transition and prints the
  resulting proposal — now `draft`, with its proposed timestamps cleared, its
  prior responses deleted server-side, and the updated available transitions
  (offering `propose` again). It is a flagless verb leaf under the `proposal`
  group (shared with `proposal list`/`get`/`create`/`propose`/`respond`). The
  transition is Premium-gated and server-authorized: the command issues the POST
  and lets the server enforce, surfacing a 404 (no such proposal) or a 422
  (transition not allowed) as a real failure — never as success — and routing the
  Premium 403 through the shared error handling with no plan-specific message.
  The withdraw is destructive (it deletes existing responses) but the command
  prompts for no confirmation and requires no --force — it is non-interactive and
  agent-driven.
  (affects: Practitioner)

  Rule: Return a circulating proposal to draft by id
    # In order to re-open a proposal I am circulating so I can amend it before it is accepted,
    # as a practitioner operating the governance write flow,
    # I want to withdraw it back to draft by id with one command.

    # Source: 059-withdraw-proposal — Scenario: Withdraw a circulating proposal back to draft
    Scenario: A circulating proposal is withdrawn back to draft
      Given a complete connection context with a stored token
      And a circulating proposal "prp_0123" whose available transitions include "withdraw"
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then the request will POST to the proposal's withdraw transition with no body
      And the returned proposal will be printed with status "draft"
      And the command will exit with code 0

    # Source: 059-withdraw-proposal — Scenario: Transition not allowed is a failure
    Scenario: A disallowed transition fails with the API status
      Given a complete connection context with a stored token
      And a proposal "prp_0123" for which "withdraw" is not currently allowed
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then the API will answer 422
      And stderr will report that the withdraw failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 059-withdraw-proposal — Scenario: Proposal id does not exist
    Scenario: An unknown proposal id fails with the API status
      Given a complete connection context with a stored token
      And no proposal "prp_ffff" exists
      When a practitioner runs "glassfrog proposal withdraw prp_ffff"
      Then the API will answer 404
      And stderr will report that the withdraw failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 059-withdraw-proposal — Scenario: No credential surfaces the not-authenticated outcome
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 059-withdraw-proposal — Scenario: Missing proposal id is rejected before any request
    Scenario: A missing proposal id is rejected before any request
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog proposal withdraw"
      Then stderr will report a usage error naming the required proposal id
      And no request will be sent
      And the command will exit with code 2

    # Source: 059-withdraw-proposal — Scenario: Premium not enabled surfaces a plain refusal
    Scenario: A Premium-gated refusal surfaces plainly
      Given a complete connection context with a stored token
      And the organization does not have async proposals enabled
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then the API will answer 403
      And stderr will report that the withdraw failed and name the HTTP status
      And no plan-specific message will be added
      And the command will exit with a non-zero permission code

    # Source: 059-withdraw-proposal — Scenario: A transport failure surfaces network-unavailable
    Scenario: A transport failure surfaces network-unavailable
      Given a complete connection context with a stored token
      And the API endpoint is unreachable
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then stderr will report the transport failure by name
      And the command will exit with a non-zero network-unavailable code

    # Source: 059-withdraw-proposal — Proposed: rate-limited withdraw is surfaced, not silently retried (plan Cross-cutting: non-idempotent retry)
    Scenario: A rate-limited withdraw is surfaced, not silently retried
      Given a complete connection context with a stored token
      And the API answers the withdraw request with 429
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then stderr will report that the withdraw failed and name the HTTP status
      And the POST will not be automatically retried
      And the command will exit with a non-zero rate-limit code

    # Source: 059-withdraw-proposal — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    Scenario: An invalid output format is rejected before any withdraw request
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog proposal withdraw prp_0123 -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Produce the resulting draft proposal as parseable data
    # In order to roll a circulating proposal back from an automated pipeline,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want the withdraw to produce the resulting draft proposal as structured data I can parse,
    # including its now-cleared deadline and updated transitions.

    # Source: 059-withdraw-proposal — Scenario: Withdrawn proposal rendered as JSON
    Scenario: The withdrawn proposal is rendered as JSON
      Given a complete connection context with a stored token
      And a circulating proposal "prp_0123" that can be withdrawn
      When an agent runs "glassfrog proposal withdraw prp_0123 -o json"
      Then the returned proposal will be printed as JSON
      And the command will exit with code 0

    # Source: 059-withdraw-proposal — Scenario: The result reflects the cleared deadline and updated transitions
    Scenario: The withdrawn proposal carries the cleared deadline and updated transitions
      Given a complete connection context with a stored token
      And a circulating proposal "prp_0123" that can be withdrawn
      When an agent runs "glassfrog proposal withdraw prp_0123"
      Then the printed proposal will show status "draft" with the response deadline cleared
      And its available transitions will be the ones the server returned
      And the command will not narrate the responses the withdraw deleted

    # Source: 059-withdraw-proposal — Scenario: The result is the server's proposal, unembellished
    @validation @wip
    Scenario: The withdrawn result is the server's proposal unembellished
      Given a successful "glassfrog proposal withdraw prp_0123" run
      When the rendered proposal is inspected in every format
      Then it will carry only fields the server returned
      And it will not fabricate any side-effect narration about deleted responses

    # Source: 059-withdraw-proposal — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: The withdraw output carries no raw API envelope under a human format
      Given a successful "glassfrog proposal withdraw prp_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" JSON envelope

  Rule: Trust the server to authorize the transition
    # In order to trust that a withdraw only happens when the server allows it,
    # as a practitioner,
    # I want the command to issue the transition and surface the server's refusal plainly
    # rather than guessing client-side whether it is permitted.

    # Source: 059-withdraw-proposal — Scenario: The withdraw issues exactly one transition request and reads nothing first
    @validation @wip
    Scenario: The withdraw issues one request and reads nothing first
      Given a complete connection context with a stored token
      And a circulating proposal "prp_0123" that can be withdrawn
      When a practitioner runs "glassfrog proposal withdraw prp_0123"
      Then exactly one request will be sent to the withdraw transition
      And no prior read of the proposal will be issued

    # Source: 059-withdraw-proposal — Scenario: A 404 and a 422 are real failures
    @validation @wip
    Scenario: A 404 and a 422 are surfaced as real failures
      Given the withdraw is run against a proposal the API answers 404 for, and one it answers 422 for
      When each run completes
      Then neither run will produce a success result
      And both runs will exit with a non-zero code
