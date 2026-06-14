# Source: 051-subrole-filler-roll-up — Scenario: Roll up the actors filling a circle's direct sub-roles

Feature: Who to Contact for a Role — Subrole Filler Roll-up
  When a role relevant to a tension is vacant or shared, the operator needs to
  reach the surrounding circle. Subrole Filler Roll-up answers it —
  `glassfrog subrole-actors <role-id>` rolls up the actors filling the anchor
  role's direct sub-roles (one level, not transitive), each shown with its kind,
  optionally narrowed to people or agents. It is the actor-shaped counterpart of
  Role Fillers (047): where `fillers` lists who fills one role, `subrole-actors`
  lists who staffs the circles inside it. The roll-up walks every page to
  completion by default, or is plainly flagged incomplete, and renders through
  the shared output seam or fails with a named error and the right exit code; a
  leaf anchor's 404 is never disguised as an empty success.
  (affects: Practitioner, AI agent)

  Rule: See who is staffing the circles inside this one
    # In order to see who is staffing the circles inside this one when the role itself is vacant or shared,
    # as a practitioner facilitating a circle,
    # I want to roll up the actors filling a circle's direct sub-roles with one command.

    # Source: 051-subrole-filler-roll-up — Scenario: Roll up the actors filling a circle's direct sub-roles
    @wip
    Scenario: A circle's direct sub-roles' fillers are rolled up
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles filled by several actors
      When an agent runs "glassfrog subrole-actors role_0123"
      Then the request will read the role's subroles actors endpoint
      And each sub-role filler will be printed as a projection
      And the command will exit with code 0

    # Source: 051-subrole-filler-roll-up — Scenario: Anchor is a leaf role
    @wip
    Scenario: A leaf anchor fails with the API status
      Given a complete connection context with a stored token
      And the role "role_0123" has no sub-roles
      When an agent runs "glassfrog subrole-actors role_0123"
      Then stderr will report that the read failed and name the HTTP status
      And no "this role has no sub-roles" message will be added
      And the command will exit with a non-zero API-error code

    # Source: 051-subrole-filler-roll-up — Scenario: No usable credential
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog subrole-actors role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no actor data will be printed
      And the command will exit with code 2

    # Source: 051-subrole-filler-roll-up — Scenario: Sub-roles exist but carry no fillers
    @wip
    Scenario: Sub-roles with no fillers are a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles filled by no actors
      When an agent runs "glassfrog subrole-actors role_0123"
      Then "no actors" will be printed to stdout
      And the command will exit with code 0

    # Source: 051-subrole-filler-roll-up — Scenario: A leaf-role 404 is a failure, not an empty success
    @validation @wip
    Scenario: A leaf 404 is distinct from an empty roll-up
      Given a complete connection context with a stored token
      When an agent runs "glassfrog subrole-actors role_0123" against a leaf anchor that answers 404
      Then the command will exit with a non-zero API-error code
      And the outcome will differ from the zero-exit empty roll-up of a childless-but-unfilled circle

    # Source: 051-subrole-filler-roll-up — Scenario: The result is actor-shaped, not assignment-shaped
    @validation @wip
    Scenario: The rolled-up rows are actor-shaped, not assignment-shaped
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles whose fillers hold focuses and elected seats
      When an agent runs "glassfrog subrole-actors role_0123"
      Then each row will carry the actor's id, name, and kind
      And no "focus" or "elected_until" field will be projected

    # Source: 051-subrole-filler-roll-up — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog subrole-actors role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped actor projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Roll up the sub-roles' fillers without fetching each child separately
    # In order to reach the surrounding circle without fetching each child role's fillers one at a time,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to read the sub-roles' fillers in a single roll-up by the anchor role's id.

    # Source: 051-subrole-filler-roll-up — Scenario: The roll-up is one level only
    @validation @wip
    Scenario: The roll-up reads only the direct sub-roles
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles whose own children also have fillers
      When an agent runs "glassfrog subrole-actors role_0123"
      Then only the direct sub-roles' fillers will be read through the subroles actors endpoint
      And no request will be made for grand-child roles' fillers

  Rule: Tell people apart from agents
    # In order to tell automation apart from people before I decide whom to contact,
    # as an AI agent assembling context,
    # I want to narrow the roll-up to just humans or just agents.

    # Source: 051-subrole-filler-roll-up — Scenario: Narrow the roll-up to agents
    @wip
    Scenario: A kind filter narrows the roll-up to agents
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles filled by both people and agents
      When an agent runs "glassfrog subrole-actors role_0123 --kind agent"
      Then the request will carry "kind" set to "agent"
      And only the agents will be printed as a list
      And the command will exit with code 0

    # Source: 051-subrole-filler-roll-up — Scenario: Unsupported kind value is rejected before any request
    @wip
    Scenario: An unsupported kind is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog subrole-actors role_0123 --kind robot"
      Then stderr will report the unsupported value and list the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 051-subrole-filler-roll-up — Scenario: An unsupported kind costs no request
    @validation @wip
    Scenario: A rejected kind issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog subrole-actors role_0123 --kind robot"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Trust the roll-up is whole, or be told it is incomplete
    # In order to trust I am seeing every actor staffing the sub-roles,
    # as a practitioner in a large circle,
    # I want the roll-up to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 051-subrole-filler-roll-up — Scenario: Roll-up walks every page to completion
    @wip
    Scenario: The roll-up walks every page to completion
      Given a complete connection context with a stored token
      And the role "role_0123" has sub-role fillers spanning more than one page
      When an agent runs "glassfrog subrole-actors role_0123"
      Then every page of sub-role fillers will be walked
      And the complete set will be printed
      And the command will exit with code 0

    # Source: 051-subrole-filler-roll-up — Scenario: Paginated roll-up with first-page opt-out
    @wip
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has sub-role fillers spanning more than one page
      When a practitioner runs "glassfrog subrole-actors role_0123 --first-page"
      Then only the first page of actors will be printed
      And stderr will note that more actors exist
      And the command will exit with code 0

    # Source: 051-subrole-filler-roll-up — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    @wip
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the subrole filler roll-up walk fails after retrieving the first page
      When a practitioner runs "glassfrog subrole-actors role_0123"
      Then the actors retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 051-subrole-filler-roll-up — Proposed: plan Cross-cutting (kind filter carried on every page of the walk)
    @validation @wip
    Scenario: The kind filter is carried on every page of the walk
      Given a complete connection context with a stored token
      And sub-role fillers filtered by "--kind human" span more than one page
      When an agent runs "glassfrog subrole-actors role_0123 --kind human"
      Then every page request of the walk will retain "kind" set to "human"
      And the command will exit with code 0
