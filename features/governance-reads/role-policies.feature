# Source: 034-role-policies — Scenario: List the policies on a role

Feature: Governance Reads — Role Policies
  A policy is a governance rule on a role's interior. Role Reads embeds a
  role's policies inline as a convenience; this is the addressable surface it
  deferred — `glassfrog policies <role-id>` lists the policies governing a
  role (walking pages to completion by default, optionally narrowed by a
  free-text search), and `glassfrog policy <pol-id>` reads one policy with its
  full body. Both render through the shared output seam or fail with a named
  error and the right exit code.
  (affects: Practitioner)

  Rule: See every governance rule on a role
    # In order to see every governance rule on a role before I act inside it,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list the policies governing any role by its id with one command.

    # Source: 034-role-policies — Scenario: List the policies on a role
    @wip
    Scenario: A role's policies are listed
      Given a complete connection context with a stored token
      And the role "role_0123" is governed by several policies
      When an agent runs "glassfrog policies role_0123"
      Then the request will read the role's policies endpoint
      And each policy will be printed as a projection
      And the command will exit with code 0

    # Source: 034-role-policies — Scenario: Role has no policies
    @wip
    Scenario: A role with no policies is a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" is governed by no policies
      When an agent runs "glassfrog policies role_0123"
      Then "No policies." will be printed to stdout
      And the command will exit with code 0

    # Source: 034-role-policies — Scenario: No usable credential
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog policies role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no policy data will be printed
      And the command will exit with code 2

    # Source: 034-role-policies — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog policies role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Read a single policy with its full body
    # In order to read the full text of a specific policy I found referenced elsewhere,
    # as a practitioner reviewing the governance around my work,
    # I want to fetch a single policy by its id and see its full body.

    # Source: 034-role-policies — Scenario: Read a single policy with its full body
    @wip
    Scenario: A single policy is read with its full body
      Given a complete connection context with a stored token
      And a policy "pol_0123" exists
      When a practitioner runs "glassfrog policy pol_0123"
      Then the policy's title and full body will be printed
      And the command will exit with code 0

    # Source: 034-role-policies — Scenario: Policy id does not exist
    @wip
    Scenario: An unknown policy id fails with the API status
      Given a complete connection context with a stored token
      And no policy "pol_ffff" exists
      When a practitioner runs "glassfrog policy pol_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 034-role-policies — Scenario: Search flag on the single read is rejected
    @wip
    Scenario: The search flag is rejected on the single read
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog policy pol_0123 --query approvals"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 034-role-policies — Scenario: The two commands never collide on id kind
    @validation @wip
    Scenario: The plural and singular commands do not collide on id kind
      Given a complete connection context with a stored token
      When a role id is passed to "glassfrog policy" or a policy id to "glassfrog policies"
      Then the wrong-kind id will surface the API's not-found or be rejected
      And the wrong resource will never be silently read

  Rule: Narrow a role's policies with a free-text search
    # In order to find the relevant policy in a circle that has many,
    # as an AI agent assembling context,
    # I want to narrow the role's policy list with a free-text search.

    # Source: 034-role-policies — Scenario: Narrow a role's policies with a search
    @wip
    Scenario: The policy list is narrowed by a search query
      Given a complete connection context with a stored token
      And the role "role_0123" is governed by a policy titled "All PRs require two approvals"
      When an agent runs "glassfrog policies role_0123 --query approvals"
      Then the request will carry the "q" search parameter
      And only the matching policies will be printed
      And the command will exit with code 0

  Rule: Trust the list is whole, or be told it is incomplete
    # In order to trust I am seeing every policy on a role,
    # as a practitioner in a large circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 034-role-policies — Scenario: Paginated list with first-page opt-out
    @wip
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" has policies spanning more than one page
      When a practitioner runs "glassfrog policies role_0123 --first-page"
      Then only the first page of policies will be printed
      And stderr will note that more policies exist
      And the command will exit with code 0

    # Source: 034-role-policies — Proposed: plan Cross-cutting mid-walk failure exit semantics (025 ADR-3)
    @wip
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the policy list walk fails after retrieving the first page
      When a practitioner runs "glassfrog policies role_0123"
      Then the policies retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code
