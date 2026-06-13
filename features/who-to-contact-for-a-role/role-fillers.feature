# Source: 047-role-fillers — Scenario: List the fillers of a role

Feature: Who to Contact for a Role — Role Fillers
  When working a tension, the operator can't tell which actor fills a relevant
  role, so they don't know whom to reach out to. Role Fillers answers it
  directly — `glassfrog fillers <role-id>` lists the actors who fill a role,
  each shown with the focus and election of their assignment. The list walks
  every page to completion by default, or is plainly flagged incomplete, and
  renders through the shared output seam or fails with a named error and the
  right exit code.
  (affects: Practitioner, AI agent)

  Rule: Know whom to contact about a role
    # In order to know whom to contact about a role I do not fill,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list the actors who fill any role by its id with one command.

    # Source: 047-role-fillers — Scenario: List the fillers of a role
    Scenario: A role's fillers are listed
      Given a complete connection context with a stored token
      And the role "role_0123" is filled by two actors
      When an agent runs "glassfrog fillers role_0123"
      Then the request will read the role's assignments endpoint
      And each filler will be printed as a projection
      And the command will exit with code 0

    # Source: 047-role-fillers — Scenario: Fillers span both a person and an agent
    Scenario: A person and an agent filler are distinguished by kind
      Given a complete connection context with a stored token
      And the role "role_0123" is filled by one person and one agent
      When an agent runs "glassfrog fillers role_0123"
      Then both fillers will be printed
      And each filler will show whether it is a person or an agent
      And the command will exit with code 0

    # Source: 047-role-fillers — Scenario: Role id does not exist
    Scenario: An unknown role id fails with the API status
      Given a complete connection context with a stored token
      And no role "role_ffff" exists
      When an agent runs "glassfrog fillers role_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 047-role-fillers — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog fillers role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no filler data will be printed
      And the command will exit with code 2

    # Source: 047-role-fillers — Scenario: Role has no fillers
    Scenario: A role with no fillers is a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" is filled by no actors
      When an agent runs "glassfrog fillers role_0123"
      Then "no fillers" will be printed to stdout
      And the command will exit with code 0

    # Source: 047-role-fillers — Scenario: Missing role-id is rejected before any request
    Scenario: A missing role-id is a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog fillers"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 047-role-fillers — Scenario: The filler's name appears without an include flag
    @validation @wip
    Scenario: The actor name is shown without an include flag
      Given a complete connection context with a stored token
      And the role "role_0123" is filled by an actor
      When an agent runs "glassfrog fillers role_0123"
      Then each filler's name and kind will be printed
      And the filler's name will come from the endpoint's default actor include
      And the command will declare no "--include" flag to obtain them

    # Source: 047-role-fillers — Scenario: A missing token costs no request
    @validation @wip
    Scenario: A missing token issues no request
      Given a transport tripwire that records whether any request is sent
      And no usable token is available to the CLI
      When an agent runs "glassfrog fillers role_0123"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: See in what capacity each actor fills the role
    # In order to understand not just who fills a role but in what capacity,
    # as a practitioner reviewing the governance around a role,
    # I want each filler shown with its focus and, for elected seats, its election expiry.

    # Source: 047-role-fillers — Scenario: A filler row shows its focus and election expiry
    Scenario: A filler shows its focus and election expiry
      Given a complete connection context with a stored token
      And the role "role_0123" is filled by an actor whose assignment has a focus and an election date
      When an agent runs "glassfrog fillers role_0123"
      Then the filler's focus and election expiry will be printed
      And the command will exit with code 0

    # Source: 047-role-fillers — Scenario: Focus and election are projected, not dropped
    @validation @wip
    Scenario: Focus and election are projected, not dropped
      Given a complete connection context with a stored token
      And a filler whose assignment carries a focus and an election date
      When an agent runs "glassfrog fillers role_0123" under the default human format
      Then the filler's focus and election expiry will both be shown
      And an absent focus or election will show an explicit-absence marker

  Rule: Trust the filler list is whole, or be told it is incomplete
    # In order to trust I am seeing every filler of a role,
    # as a practitioner in a busy circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 047-role-fillers — Scenario: Paginated list with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has fillers spanning more than one page
      When a practitioner runs "glassfrog fillers role_0123 --first-page"
      Then only the first page of fillers will be printed
      And stderr will note that more fillers exist
      And the command will exit with code 0

    # Source: 047-role-fillers — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the filler list walk fails after retrieving the first page
      When a practitioner runs "glassfrog fillers role_0123"
      Then the fillers retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 047-role-fillers — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog fillers role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope
