# Source: 043-tension-reads — Scenario: List the tensions on a role

Feature: Tension Reads
  A tension is a gap a role senses — recorded by `glassfrog tension create`
  (042). This is the read counterpart: `glassfrog tension list <role-id>` lists
  the tensions on a role (walking pages to completion by default, optionally
  narrowed to a status), and `glassfrog tension get <ten-id>` reads one tension
  with full detail. They are verb leaves under the same `tension` group as
  `create`. Both render through the shared output seam or fail with a named
  error and the right exit code; a partial list is never silently presented as
  whole.
  (affects: Practitioner)

  Rule: See what tensions a role is carrying
    # In order to see what tensions a role is carrying before I act inside it,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list the tensions on any role by its id with one command.

    # Source: 043-tension-reads — Scenario: List the tensions on a role
    Scenario: A role's tensions are listed
      Given a complete connection context with a stored token
      And the role "role_0123" carries several tensions
      When an agent runs "glassfrog tension list role_0123"
      Then the request will read the role's tensions endpoint
      And each tension will be printed as a projection
      And the command will exit with code 0

    # Source: 043-tension-reads — Scenario: Role carries no tensions
    Scenario: A role with no tensions is a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" carries no tensions
      When an agent runs "glassfrog tension list role_0123"
      Then "no tensions" will be printed to stdout
      And the command will exit with code 0

    # Source: 043-tension-reads — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tension list role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no tension data will be printed
      And the command will exit with code 2

    # Source: 043-tension-reads — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog tension list role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

    # Source: 043-tension-reads — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    Scenario: An invalid output format is rejected before any list request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension list role_0123 -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Read the full detail of a tension by its id
    # In order to read the full detail of a tension I captured or found referenced elsewhere,
    # as a practitioner working a governance issue,
    # I want to fetch a single tension by its ten_ id.

    # Source: 043-tension-reads — Scenario: Read a single tension with full detail
    Scenario: A single tension is read with full detail
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When a practitioner runs "glassfrog tension get ten_0123"
      Then the tension's status, body, and sensing role will be printed
      And the command will exit with code 0

    # Source: 043-tension-reads — Scenario: Tension id does not exist
    Scenario: An unknown tension id fails with the API status
      Given a complete connection context with a stored token
      And no tension "ten_ffff" exists
      When a practitioner runs "glassfrog tension get ten_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 043-tension-reads — Scenario: Status filter on the single read is rejected
    Scenario: The list filter is rejected on the single read
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog tension get ten_0123 --status unprocessed"
      Then stderr will report a usage error
      And no request will be sent
      And the command will exit with code 2

    # Source: 043-tension-reads — Scenario: The read surface never reaches into the write verbs
    @validation @wip
    Scenario: The read verbs expose no write behavior
      Given the implemented "glassfrog tension list" and "glassfrog tension get" verbs
      When their behavior and help are inspected
      Then only reading will be available under each
      And no create, update, or discard behavior will be advertised or implemented

    # Source: 043-tension-reads — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    Scenario: An invalid output format is rejected before any get request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension get ten_0123 -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Narrow a role's tensions by status
    # In order to focus on just the unworked tensions in a role with a long history,
    # as an AI agent assembling context,
    # I want to narrow the role's tension list by status.

    # Source: 043-tension-reads — Scenario: Narrow a role's tensions by status
    Scenario: The tension list is narrowed by a supported status
      Given a complete connection context with a stored token
      And the role "role_0123" carries tensions in several statuses
      When an agent runs "glassfrog tension list role_0123 --status unprocessed"
      Then the request will carry the "status" parameter set to "unprocessed"
      And only the unprocessed tensions will be printed
      And the command will exit with code 0

    # Source: 043-tension-reads — Scenario: Unsupported status value is rejected before any request
    Scenario: An unsupported status is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension list role_0123 --status open"
      Then stderr will report the unsupported value and list the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 043-tension-reads — Scenario: An unsupported status costs no request
    @validation @wip
    Scenario: A rejected status issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog tension list role_0123 --status open"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Trust the list is whole, or be told it is incomplete
    # In order to trust I am seeing every tension on a role,
    # as a practitioner in a busy circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 043-tension-reads — Scenario: Paginated list with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has tensions spanning more than one page
      When a practitioner runs "glassfrog tension list role_0123 --first-page"
      Then only the first page of tensions will be printed
      And stderr will note that more tensions exist
      And the command will exit with code 0

    # Source: 043-tension-reads — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the tension list walk fails after retrieving the first page
      When a practitioner runs "glassfrog tension list role_0123"
      Then the tensions retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code
