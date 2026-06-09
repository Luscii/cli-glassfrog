# Source: 025-role-reads — Scenario: List the organization's roles

Feature: Governance Reads — Role Reads
  Reading governance structure starts with the org-wide role surface: list
  every role in the organization and drill into one by id. Unlike the
  token-scoped My Roles, this answers "what roles exist here, and what is this
  one" — and it is the source of the role ids the per-role reads consume.
  `glassfrog roles` lists (walking pages to completion by default), and
  `glassfrog roles <id>` reads one role, optionally embedding related
  resources, rendered through the shared output seam or failing with a named
  error and the right exit code.
  (affects: Practitioner)

  Rule: Navigate the organization and drill into any role by id
    # In order to navigate the organization's governance structure and drill into any role I find,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list every role and read one by its id with one command each.

    # Source: 025-role-reads — Scenario: List the organization's roles
    @wip
    Scenario: The organization's roles are listed
      Given a complete connection context with a stored token
      And the API would return several roles in the organization
      When an agent runs "glassfrog roles"
      Then each role will be printed as a projection
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: Read a single role by id
    @wip
    Scenario: A single role is read by id
      Given a complete connection context with a stored token
      And a role "role_0123" exists in the organization
      When an agent runs "glassfrog roles role_0123"
      Then the role's name, purpose, accountabilities, domains, and fillers will be printed
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: No usable token
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog roles"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no role data will be printed
      And the command will exit with code 2

    # Source: 025-role-reads — Scenario: The API cannot be reached
    @wip
    Scenario: An unreachable API fails as network-unavailable
      Given a complete connection context with a stored token
      And the API is unreachable at the wire
      When an agent runs "glassfrog roles"
      Then stderr will name the transport failure
      And the command will exit with code 6

    # Source: 025-role-reads — Scenario: A single read for an unknown id
    @wip
    Scenario: An unknown role id fails with the API status
      Given a complete connection context with a stored token
      And no role "role_ffff" exists in the organization
      When an agent runs "glassfrog roles role_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 025-role-reads — Scenario: The organization has no roles
    @wip
    Scenario: An empty organization is a clean success
      Given a complete connection context with a stored token
      And the API would return no roles
      When an agent runs "glassfrog roles"
      Then "No roles." will be printed to stdout
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: Default output contains no raw API envelope
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog roles" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Filter the role list by parent, person, tag, or sub-role presence
    # In order to find the roles under a particular circle, held by a particular person, or carrying a tag,
    # as a practitioner exploring the org,
    # I want to filter the role list by parent, person, tag, or whether a role has subroles.

    # Source: 025-role-reads — Scenario: Filter the list by parent circle
    @wip
    Scenario: The list is filtered by parent circle
      Given a complete connection context with a stored token
      And the parent role "role_aaaa" contains several roles
      When the practitioner runs "glassfrog roles --parent role_aaaa"
      Then the request will carry the "parent_role_id" filter
      And only roles under that parent will be printed
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: A list filter is rejected on the single read
    @wip
    Scenario: A filter passed with a role id is a usage error
      Given a complete connection context with a stored token
      When the practitioner runs "glassfrog roles role_0123 --tag marketing"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

  Rule: Embed a single role's related resources in one call
    # In order to see a role together with its related resources in a single call,
    # as an AI agent assembling context before acting,
    # I want to request a single role with its assignments, subroles, parent, policies, notes, or skills embedded inline.

    # Source: 025-role-reads — Scenario: Read a single role with related resources embedded
    @wip
    Scenario: Requested related resources are embedded inline
      Given a complete connection context with a stored token
      And a role "role_0123" exists in the organization
      When an agent runs "glassfrog roles role_0123 --include policies,subroles"
      Then the request will carry "include=policies,subroles"
      And the policies and subroles will be printed inline within the role
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: An unsupported `--include` value is rejected without an API call
    @wip
    Scenario: An unsupported include value is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog roles role_0123 --include nonsense"
      Then stderr will name the unsupported value and the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 025-role-reads — Scenario: Embedded-include view does not substitute for the standalone reads
    @validation @wip
    Scenario: An embedded include is not a standalone read
      Given a successful "glassfrog roles role_0123 --include policies" run
      When the result is inspected
      Then the policies will appear embedded inline on the role
      And no standalone per-policy projection will be produced

  Rule: Trust the list is whole, or be told it is incomplete
    # In order to trust that I am seeing every role in the org,
    # as a practitioner in a large organization,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 025-role-reads — Scenario: Roles span more than one page (default walk to completion)
    @wip
    Scenario: A multi-page role list is walked to completion
      Given a complete connection context with a stored token
      And the organization's roles span three pages of API responses
      When the practitioner runs "glassfrog roles"
      Then the command will walk every page to completion
      And all roles across the pages will be printed
      And the command will exit with code 0

    # Source: 025-role-reads — Scenario: First-page opt-out stops at one page and signals more exist
    @wip
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the organization's roles span more than one page
      When the practitioner runs "glassfrog roles --first-page"
      Then only the first page of roles will be printed
      And stderr will note that more roles exist
      And the command will exit with code 0

    # Source: 025-role-reads — Proposed: plan ADR-3 mid-walk failure exit semantics
    @wip
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the role list walk fails after retrieving the first page
      When the practitioner runs "glassfrog roles"
      Then the roles retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 025-role-reads — Scenario: List incompleteness is never silent
    @validation @wip
    Scenario: List incompleteness is never silent
      Given a role list run that did not retrieve every page
      When the output is inspected
      Then an explicit incomplete signal with its cause will be present
      And the partial list cannot be read as the complete set
