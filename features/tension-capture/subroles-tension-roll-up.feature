# Source: 046-subroles-tension-roll-up — Scenario: Roll up tensions across a circle's direct sub-roles

Feature: Subroles Tension Roll-up
  A tension is a gap a role senses — recorded by `glassfrog tension create`
  (042) and listed per-role by `glassfrog tension list` (043). This is the
  cross-role counterpart: `glassfrog tension subroles <role-id>` rolls up the
  tensions sensed across the anchor role's direct sub-roles (one level, not
  transitive), walking pages to completion by default and optionally narrowed
  to a status. It is a verb leaf under the same `tension` group as `create`,
  `list`, `get`, and `update`, reusing the tension model and list render
  unchanged. It renders through the shared output seam or fails with a named
  error and the right exit code; a partial roll-up is never silently presented
  as whole, and a leaf anchor's 404 is never disguised as an empty success.
  (affects: Practitioner)

  Rule: See what the circles inside this one are sensing
    # In order to see everything the circles inside this one are sensing before a governance meeting,
    # as a practitioner facilitating a circle,
    # I want to roll up the tensions across a circle's direct sub-roles with one command.

    # Source: 046-subroles-tension-roll-up — Scenario: Roll up tensions across a circle's direct sub-roles
    Scenario: A circle's direct sub-roles' tensions are rolled up
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles carrying several tensions
      When an agent runs "glassfrog tension subroles role_0123"
      Then the request will read the role's subroles tensions endpoint
      And each sub-role tension will be printed as a projection
      And the command will exit with code 0

    # Source: 046-subroles-tension-roll-up — Scenario: Anchor is a leaf role
    Scenario: A leaf anchor fails with the API status
      Given a complete connection context with a stored token
      And the role "role_0123" has no sub-roles
      When an agent runs "glassfrog tension subroles role_0123"
      Then stderr will report that the read failed and name the HTTP status
      And no "this role has no sub-roles" message will be added
      And the command will exit with a non-zero API-error code

    # Source: 046-subroles-tension-roll-up — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tension subroles role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no tension data will be printed
      And the command will exit with code 2

    # Source: 046-subroles-tension-roll-up — Scenario: Sub-roles exist but carry no tensions
    Scenario: Sub-roles with no tensions are a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles carrying no tensions
      When an agent runs "glassfrog tension subroles role_0123"
      Then "no tensions" will be printed to stdout
      And the command will exit with code 0

    # Source: 046-subroles-tension-roll-up — Scenario: A leaf-role 404 is a failure, not an empty success
    @validation @wip
    Scenario: A leaf 404 is distinct from an empty roll-up
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension subroles role_0123" against a leaf anchor that answers 404
      Then the command will exit with a non-zero API-error code
      And the outcome will differ from the zero-exit empty roll-up of a childless-but-tensionless circle

    # Source: 046-subroles-tension-roll-up — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog tension subroles role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Assemble the sub-roles' tensions without fetching each child separately
    # In order to assemble the tensions a circle is carrying below it without fetching each child role separately,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to read the sub-roles' tensions in a single roll-up by the anchor role's id.

    # Source: 046-subroles-tension-roll-up — Scenario: The roll-up is one level only
    @validation @wip
    Scenario: The roll-up reads only the direct sub-roles
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles whose own children also carry tensions
      When an agent runs "glassfrog tension subroles role_0123"
      Then only the direct sub-roles' tensions will be read through the subroles tensions endpoint
      And no request will be made for grand-child roles' tensions

  Rule: Focus the roll-up on just the unworked tensions
    # In order to focus on just the unworked tensions surfacing across a busy circle's children,
    # as an AI agent assembling context,
    # I want to narrow the roll-up by status.

    # Source: 046-subroles-tension-roll-up — Scenario: Narrow the roll-up by status
    Scenario: The roll-up is narrowed by a supported status
      Given a complete connection context with a stored token
      And the role "role_0123" has direct sub-roles carrying tensions in several statuses
      When an agent runs "glassfrog tension subroles role_0123 --status unprocessed"
      Then the request will carry the "status" parameter set to "unprocessed"
      And only the unprocessed tensions will be printed
      And the command will exit with code 0

    # Source: 046-subroles-tension-roll-up — Scenario: Unsupported status value is rejected before any request
    Scenario: An unsupported status is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension subroles role_0123 --status open"
      Then stderr will report the unsupported value and list the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 046-subroles-tension-roll-up — Scenario: An unsupported status costs no request
    @validation @wip
    Scenario: A rejected status issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog tension subroles role_0123 --status open"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Trust the roll-up is whole, or be told it is incomplete
    # In order to trust I am seeing every tension the sub-roles are carrying,
    # as a practitioner in a large circle,
    # I want the roll-up to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 046-subroles-tension-roll-up — Scenario: Roll-up walks every page to completion
    Scenario: The roll-up walks every page to completion
      Given a complete connection context with a stored token
      And the role "role_0123" has sub-role tensions spanning more than one page
      When an agent runs "glassfrog tension subroles role_0123"
      Then every page of sub-role tensions will be walked
      And the complete set will be printed
      And the command will exit with code 0

    # Source: 046-subroles-tension-roll-up — Scenario: Paginated roll-up with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has sub-role tensions spanning more than one page
      When a practitioner runs "glassfrog tension subroles role_0123 --first-page"
      Then only the first page of tensions will be printed
      And stderr will note that more tensions exist
      And the command will exit with code 0

    # Source: 046-subroles-tension-roll-up — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the subroles tension roll-up walk fails after retrieving the first page
      When a practitioner runs "glassfrog tension subroles role_0123"
      Then the tensions retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code
