# Source: 026-organization-tree — Scenario: Read the whole organization tree

Feature: Governance Reads — Organization Tree
  Reading governance structure includes the nesting between roles: the circle
  hierarchy. Unlike the flat role list, this answers "what contains what".
  `glassfrog tree` reads the whole organization (or a subtree rooted at a role)
  as a single nested document, bounded by an optional depth; `glassfrog
  subroles <id>` lists a role's immediate children, walking pages to completion
  by default. Both render through the shared output seam or fail with a named
  error and the right exit code.
  (affects: Practitioner)

  Rule: Read the whole organization tree in one call
    # In order to orient myself to an organization's whole governance structure in one call,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to read the entire role tree as a single nested document.

    # Source: 026-organization-tree — Scenario: Read the whole organization tree
    @wip
    Scenario: The whole organization tree is read
      Given a complete connection context with a stored token
      And the API would return a nested role tree for the organization
      When an agent runs "glassfrog tree"
      Then the tree will be printed as a nested projection rooted at the anchor role
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: No usable token
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tree"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no tree data will be printed
      And the command will exit with code 2

    # Source: 026-organization-tree — Scenario: Tree default output carries no raw API envelope
    @validation @wip
    Scenario: Default tree output carries no raw API envelope
      Given a successful "glassfrog tree" run under the default human format
      When the output is inspected
      Then it will show the reshaped nested projection only
      And it will not contain the raw "data" JSON envelope

  Rule: Focus on one branch with a rooted subtree
    # In order to focus on one branch of the org without pulling the whole tree,
    # as a practitioner exploring a particular circle,
    # I want to read the subtree rooted at a specific role, optionally bounding how deep it goes.

    # Source: 026-organization-tree — Scenario: Read the subtree rooted at a role
    @wip
    Scenario: A subtree rooted at a role is read
      Given a complete connection context with a stored token
      And a role "role_0123" exists in the organization
      When an agent runs "glassfrog tree role_0123"
      Then the tree will be printed rooted at and including "role_0123"
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: Read a tree with related resources on each node
    @wip
    Scenario: Requested per-node resources are embedded in the tree
      Given a complete connection context with a stored token
      And a role "role_0123" exists in the organization
      When an agent runs "glassfrog tree role_0123 --include accountabilities,domains"
      Then the request will carry "include=accountabilities,domains"
      And each node will carry its accountabilities and domains inline
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: A tree read for an unknown role id
    @wip
    Scenario: An unknown role id fails with the API status
      Given a complete connection context with a stored token
      And no role "role_ffff" exists in the organization
      When an agent runs "glassfrog tree role_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 026-organization-tree — Scenario: An unsupported `--include` value is rejected without an API call
    @wip
    Scenario: An unsupported tree include value is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tree role_0123 --include nonsense"
      Then stderr will name the unsupported value and the tree read's supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 026-organization-tree — Scenario: A leaf role's tree is a single node
    @wip
    Scenario: A leaf root renders as a single node
      Given a complete connection context with a stored token
      And a role "role_0123" exists with no child roles
      When an agent runs "glassfrog tree role_0123"
      Then a single-node tree with no children will be printed
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: The CLI rejects unknown includes the API would have silently ignored
    @validation @wip
    Scenario: A typo'd tree include is caught locally, not silently dropped
      Given a tree read invoked with an "--include" value outside accountabilities, domains, members
      When the invocation is processed
      Then it will be rejected as a usage error before any request is issued
      And no request carrying a silently-ignored include will be sent

  Rule: See exactly what sits directly inside a circle
    # In order to see exactly what sits directly inside a circle,
    # as an AI agent navigating the hierarchy step by step,
    # I want to list a role's immediate subroles and trust the list is complete or plainly flagged incomplete.

    # Source: 026-organization-tree — Scenario: List a role's immediate subroles
    @wip
    Scenario: A role's immediate subroles are listed
      Given a complete connection context with a stored token
      And the circle role "role_0123" has several child roles
      When an agent runs "glassfrog subroles role_0123"
      Then each direct child role will be printed as a projection
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: List subroles with related resources embedded
    @wip
    Scenario: Requested related resources are embedded on each child
      Given a complete connection context with a stored token
      And the circle role "role_0123" has several child roles
      When an agent runs "glassfrog subroles role_0123 --include assignments,policies"
      Then the request will carry "include=assignments,policies"
      And the assignments and policies will be printed inline on each child role
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: First-page opt-out on subroles stops at one page and signals more exist
    @wip
    Scenario: The subroles first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the subroles of "role_0123" span more than one page
      When an agent runs "glassfrog subroles role_0123 --first-page"
      Then only the first page of child roles will be printed
      And stderr will note that more subroles exist
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: A leaf role has no subroles
    @wip
    Scenario: A leaf role's subroles are an empty success
      Given a complete connection context with a stored token
      And the role "role_0123" has no child roles
      When an agent runs "glassfrog subroles role_0123"
      Then "No subroles." will be printed to stdout
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: Subroles span more than one page (default walk to completion)
    @wip
    Scenario: A multi-page subroles list is walked to completion
      Given a complete connection context with a stored token
      And the subroles of "role_0123" span three pages of API responses
      When an agent runs "glassfrog subroles role_0123"
      Then the command will walk every page to completion
      And all child roles across the pages will be printed
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: `--depth` is rejected on the subroles read
    @wip
    Scenario: A depth flag on subroles is a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog subroles role_0123 --depth 2"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 026-organization-tree — Proposed: plan ADR-3 mid-walk failure exit semantics
    @wip
    Scenario: A mid-walk subroles failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the subroles walk for "role_0123" fails after retrieving the first page
      When an agent runs "glassfrog subroles role_0123"
      Then the child roles retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 026-organization-tree — Scenario: Subroles incompleteness is never silent
    @validation @wip
    Scenario: Subroles incompleteness is never silent
      Given a subroles run that did not retrieve every page
      When the output is inspected
      Then an explicit incomplete signal with its cause will be present
      And the partial list cannot be read as the complete set

  Rule: Keep a large tree manageable by capping depth
    # In order to keep a large tree response manageable,
    # as an AI agent with a bounded context,
    # I want to cap the tree depth so I read only as far down as I need.

    # Source: 026-organization-tree — Scenario: Bound the tree depth
    @wip
    Scenario: A depth flag bounds the tree to direct children
      Given a complete connection context with a stored token
      And the organization has a deep role hierarchy
      When an agent runs "glassfrog tree --depth 1"
      Then the request will carry "depth=1"
      And the anchor role and only its direct children will be printed
      And the command will exit with code 0

    # Source: 026-organization-tree — Scenario: A depth-capped node is marked as having more below
    @wip
    Scenario: A depth-capped node is marked as having subroles below
      Given a complete connection context with a stored token
      And the organization is deeper than one level
      And a direct child "role_0456" itself contains subroles
      When an agent runs "glassfrog tree --depth 1"
      Then "role_0456" will be marked as having subroles below the returned tree
      And a true leaf child will be marked as having none
      And no invented count of omitted descendants will be printed

    # Source: 026-organization-tree — Scenario: A depth-capped node is distinguishable from a leaf
    @validation @wip
    Scenario: A depth-capped node is distinguishable from a leaf
      Given a depth-bounded tree where one boundary node has subroles below the cut and another is a true leaf
      When the result is inspected
      Then the boundary node will be marked as having subroles below the returned tree
      And the leaf will be marked as having none
      And neither will carry an invented count of omitted descendants
