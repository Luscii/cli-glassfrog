# Source: 038-role-projects — Scenario: List the projects on a role

Feature: Governance Reads — Role Projects
  A project is a role's tracked outcome — the most operational governance
  element. My Projects reads the practitioner's own projects; this is the
  role-addressable surface — `glassfrog projects <role-id>` lists the projects
  owned by a role (walking pages to completion by default, optionally narrowed
  by a free-text search, a status filter, and/or a tag), and
  `glassfrog project <proj-id>` reads one project with full detail. Both render
  through the shared output seam or fail with a named error and the right exit
  code.
  (affects: Practitioner)

  Rule: See what work a role is responsible for
    # In order to see what work a role is responsible for before I act inside it,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list the projects owned by any role by its id with one command.

    # Source: 038-role-projects — Scenario: List the projects on a role
    Scenario: A role's projects are listed
      Given a complete connection context with a stored token
      And the role "role_0123" owns several projects
      When an agent runs "glassfrog projects role_0123"
      Then the request will read the role's projects endpoint
      And each project will be printed as a projection
      And the command will exit with code 0

    # Source: 038-role-projects — Scenario: Role owns no projects
    Scenario: A role with no projects is a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" owns no projects
      When an agent runs "glassfrog projects role_0123"
      Then "no projects" will be printed to stdout
      And the command will exit with code 0

    # Source: 038-role-projects — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog projects role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no project data will be printed
      And the command will exit with code 2

    # Source: 038-role-projects — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog projects role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Read a single project with full detail
    # In order to read the full detail of a specific project I found referenced elsewhere,
    # as a practitioner reviewing the work around a role,
    # I want to fetch a single project by its id.

    # Source: 038-role-projects — Scenario: Read a single project with full detail
    Scenario: A single project is read with full detail
      Given a complete connection context with a stored token
      And a project "proj_0123" exists
      When a practitioner runs "glassfrog project proj_0123"
      Then the project's status, description, and owning role will be printed
      And the command will exit with code 0

    # Source: 038-role-projects — Scenario: Project id does not exist
    Scenario: An unknown project id fails with the API status
      Given a complete connection context with a stored token
      And no project "proj_ffff" exists
      When a practitioner runs "glassfrog project proj_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 038-role-projects — Scenario: Filter flag on the single read is rejected
    Scenario: A list filter is rejected on the single read
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog project proj_0123 --status current"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 038-role-projects — Scenario: The two commands never collide on id kind
    @validation @wip
    Scenario: The plural and singular commands do not collide on id kind
      Given a complete connection context with a stored token
      When a role id is passed to "glassfrog project" or a project id to "glassfrog projects"
      Then the wrong-kind id will surface the API's not-found or be rejected
      And the wrong resource will never be silently read

  Rule: Narrow a role's projects by status, search, or tag
    # In order to focus on just the live work in a role with many projects,
    # as an AI agent assembling context,
    # I want to narrow the role's project list by status, free-text, or tag.

    # Source: 038-role-projects — Scenario: Narrow a role's projects by status
    Scenario: The project list is narrowed by a supported status
      Given a complete connection context with a stored token
      And the role "role_0123" owns projects in several statuses
      When an agent runs "glassfrog projects role_0123 --status current"
      Then the request will carry the "status" parameter set to "current"
      And only the current projects will be printed
      And the command will exit with code 0

    # Source: 038-role-projects — Scenario: Unsupported status value is rejected before any request
    Scenario: An unsupported status is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog projects role_0123 --status active"
      Then stderr will report the unsupported value and list the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 038-role-projects — Scenario: An unsupported status costs no request
    @validation @wip
    Scenario: A rejected status issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog projects role_0123 --status active"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Trust the list is whole, or be told it is incomplete
    # In order to trust I am seeing every project on a role,
    # as a practitioner in a busy circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 038-role-projects — Scenario: Paginated list with first-page opt-out
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has projects spanning more than one page
      When a practitioner runs "glassfrog projects role_0123 --first-page"
      Then only the first page of projects will be printed
      And stderr will note that more projects exist
      And the command will exit with code 0

    # Source: 038-role-projects — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the project list walk fails after retrieving the first page
      When a practitioner runs "glassfrog projects role_0123"
      Then the projects retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code
