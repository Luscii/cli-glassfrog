# Source: 050-actor-assignments — Scenario: List the roles an actor fills

Feature: An Actor's Governance Footprint — Actor Assignments
  Given an actor, the operator can't see what they do — the roles they fill and
  the context those carry. Actor Assignments answers it from the actor end —
  `glassfrog assignments <actor-id>` lists the roles an actor fills, each shown
  with the role it names and the focus and election of the assignment. It is the
  mirror of Role Fillers, reading the same assignment relationship by actor
  instead of by role. The list walks every page to completion by default, or is
  plainly flagged incomplete, and renders through the shared output seam or fails
  with a named error and the right exit code.
  (affects: Practitioner, AI agent)

  Rule: See the roles an actor fills
    # In order to understand a person's or agent's whole governance footprint at a glance,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list every role an actor fills by its id with one command.

    # Source: 050-actor-assignments — Scenario: List the roles an actor fills
    Scenario: An actor's assignments are listed
      Given a complete connection context with a stored token
      And the actor "per_0123" fills two roles
      When an agent runs "glassfrog assignments per_0123"
      Then the request will read the actor's assignments endpoint
      And each assignment will be printed as a projection naming its filled role
      And the command will exit with code 0

    # Source: 050-actor-assignments — Scenario: Assignments read for either a person or an agent
    Scenario: A person and an agent are read from the same endpoint
      Given a complete connection context with a stored token
      And the person "per_0123" and the agent "agt_0456" each fill a role
      When an agent runs "glassfrog assignments agt_0456"
      Then the request will read the actor's assignments endpoint
      And the agent's assignments will be printed
      And the command will exit with code 0

    # Source: 050-actor-assignments — Scenario: Actor id does not exist
    Scenario: An unknown actor id fails with the API status
      Given a complete connection context with a stored token
      And no actor "per_ffff" exists
      When an agent runs "glassfrog assignments per_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 050-actor-assignments — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog assignments per_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no assignment data will be printed
      And the command will exit with code 2

    # Source: 050-actor-assignments — Scenario: Actor fills no roles
    Scenario: An actor with no assignments is a clean success
      Given a complete connection context with a stored token
      And the actor "per_0123" fills no roles
      When an agent runs "glassfrog assignments per_0123"
      Then "no assignments" will be printed to stdout
      And the command will exit with code 0

    # Source: 050-actor-assignments — Scenario: Missing actor-id is rejected before any request
    Scenario: A missing actor-id is a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog assignments"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 050-actor-assignments — Scenario: The filled role's name appears without an include flag
    @validation @wip
    Scenario: The filled role name is shown without an include flag
      Given a complete connection context with a stored token
      And the actor "per_0123" fills a role
      When an agent runs "glassfrog assignments per_0123"
      Then each assignment's filled role name and id will be printed
      And the role will come from the endpoint's default role include
      And the command will declare no "--include" flag to obtain them

    # Source: 050-actor-assignments — Scenario: A missing token costs no request
    @validation @wip
    Scenario: A missing token issues no request
      Given a transport tripwire that records whether any request is sent
      And no usable token is available to the CLI
      When an agent runs "glassfrog assignments per_0123"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: See in what capacity each role is filled
    # In order to understand not just which roles an actor fills but in what capacity,
    # as a practitioner reviewing who is doing what,
    # I want each assignment shown with its focus and, for elected seats, its election expiry.

    # Source: 050-actor-assignments — Scenario: An assignment row shows its focus and election expiry
    Scenario: An assignment shows its focus and election expiry
      Given a complete connection context with a stored token
      And the actor "per_0123" fills a role through an assignment with a focus and an election date
      When an agent runs "glassfrog assignments per_0123"
      Then the assignment's focus and election expiry will be printed
      And the command will exit with code 0

    # Source: 050-actor-assignments — Scenario: Focus and election are projected, not dropped
    @validation @wip
    Scenario: Focus and election are projected, not dropped
      Given a complete connection context with a stored token
      And an assignment that carries a focus and an election date
      When an agent runs "glassfrog assignments per_0123" under the default human format
      Then the assignment's focus and election expiry will both be shown
      And an absent focus or election will show an explicit-absence marker

  Rule: Trust the assignment list is whole, or be told it is incomplete
    # In order to trust I am seeing every assignment an actor holds,
    # as a practitioner reviewing a busy organization,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 050-actor-assignments — Scenario: Paginated list with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the actor "per_0123" has assignments spanning more than one page
      When a practitioner runs "glassfrog assignments per_0123 --first-page"
      Then only the first page of assignments will be printed
      And stderr will note that more assignments exist
      And the command will exit with code 0

    # Source: 050-actor-assignments — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the assignment list walk fails after retrieving the first page
      When a practitioner runs "glassfrog assignments per_0123"
      Then the assignments retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 050-actor-assignments — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog assignments per_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope
