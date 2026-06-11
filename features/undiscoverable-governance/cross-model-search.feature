# Source: 041-cross-model-search — Scenario: Search across all resource types

Feature: Undiscoverable Governance — Cross-Model Search
  When working a tension, the operator can't find which roles, policies, or
  role-fillers are relevant without already knowing where to look. Cross-Model
  Search adds `glassfrog search <query>` — one relevance-ranked full-text query
  across every resource type (roles, notes, projects, actions, skills, actors,
  policies, domains). Results render uniformly in the API's relevance order, and
  each result's type and id is the bridge the operator drills into via the
  matching read command. The list walks every page to completion by default, or
  is plainly flagged incomplete. It renders through the shared output seam or
  fails with a named error and the right exit code.
  (affects: Practitioner, AI agent)

  Rule: Find relevant governance by topic in one ranked query
    # In order to find the roles, policies, and projects relevant to a tension without already knowing where they live,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to run one relevance-ranked query across every governance resource type and get a uniform ranked list.

    # Source: 041-cross-model-search — Scenario: Search across all resource types
    @wip
    Scenario: A query searches across all resource types
      Given a complete connection context with a stored token
      And several resources of different types match "onboarding"
      When an agent runs "glassfrog search onboarding"
      Then the request will carry "query" set to "onboarding" with no type scope
      And the matching results will be printed in relevance order
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: A multi-word websearch query is forwarded verbatim
    @wip
    Scenario: A multi-word websearch query is forwarded verbatim
      Given a complete connection context with a stored token
      When an agent runs "glassfrog search \"strategy review\" -archived"
      Then the request will carry "query" as the whole string unmodified
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: No usable token
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog search onboarding"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no results will be printed
      And the command will exit with code 2

    # Source: 041-cross-model-search — Scenario: A query the API rejects as malformed
    @wip
    Scenario: A query the API rejects as malformed fails with the API status
      Given a complete connection context with a stored token
      And the API cannot process the submitted query
      When an agent runs "glassfrog search onboarding"
      Then stderr will report that the search failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 041-cross-model-search — Scenario: A search that matches nothing
    @wip
    Scenario: A search matching nothing is a clean success
      Given a complete connection context with a stored token
      And no resource matches "zxqv"
      When an agent runs "glassfrog search zxqv"
      Then "no results" will be printed to stdout
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: A missing query is a usage error
    @wip
    Scenario: A missing query is a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog search" with no query argument
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 041-cross-model-search — Scenario: The query is forwarded byte-for-byte
    @validation @wip
    Scenario: The query reaches the API byte-for-byte
      Given a search invoked with a query containing websearch operators
      When the outbound request is inspected
      Then the "query" parameter will equal the operator's input exactly
      And no client-side rewriting or escaping will have been applied

    # Source: 041-cross-model-search — Scenario: Search default output carries no raw API envelope
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog search onboarding" run under the default human format
      When the output is inspected
      Then it will show the reshaped result projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Narrow a noisy search to the kinds I care about
    # In order to narrow a noisy search to just the kinds of records I care about,
    # as an AI agent assembling context for a decision,
    # I want to scope the query to specific resource types.

    # Source: 041-cross-model-search — Scenario: Scope a search to specific types
    @wip
    Scenario: A search is scoped to specific types
      Given a complete connection context with a stored token
      And the organization has roles and projects matching "budget"
      When an agent runs "glassfrog search budget --types role,project"
      Then the request will carry "types" set to "role,project"
      And only role and project results will be printed
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: An unsupported `--types` value is rejected without an API call
    @wip
    Scenario: An unsupported type is rejected as a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog search budget --types nonsense"
      Then stderr will report the unsupported value and list the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 041-cross-model-search — Scenario: An unsupported `--types` value is rejected without an API call
    @validation @wip
    Scenario: A rejected type issues no request
      Given a transport tripwire that records whether any request is sent
      When an agent runs "glassfrog search budget --types nonsense"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Go from a search hit to the full record
    # In order to go from a search hit to the full record,
    # as an AI agent navigating governance,
    # I want each result to carry the type and id I need to drill into the matching read command.

    # Source: 041-cross-model-search — Scenario: Each result carries the bridge into a read command
    @wip
    Scenario: Each result carries the bridge into a read command
      Given a complete connection context with a stored token
      And a query that matches at least one role
      When an agent runs "glassfrog search onboarding"
      Then each result will carry its type, id, title, excerpt, and rank
      And a role result will also carry its owning role id

    # Source: 041-cross-model-search — Scenario: The rendered order matches the API's relevance order
    @validation @wip
    Scenario: The rendered order matches the API's relevance order
      Given a successful multi-result search
      When the produced result order is compared against the API response order
      Then they will be identical
      And no client-side re-sort or de-duplication will have occurred

  Rule: Trust the results are whole, or be told they are incomplete
    # In order to trust that I have seen everything that matched,
    # as an AI agent with a bounded context,
    # I want the result set to be complete by default or plainly flagged incomplete.

    # Source: 041-cross-model-search — Scenario: Results span more than one page (default walk to completion)
    @wip
    Scenario: A multi-page result walks to completion by default
      Given a complete connection context with a stored token
      And the query "onboarding" matches results spanning more than one page
      When an agent runs "glassfrog search onboarding"
      Then every page will be walked and the complete relevance-ordered set will be printed
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: First-page opt-out stops at one page and signals more exist
    @wip
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the query "onboarding" matches results spanning more than one page
      When an agent runs "glassfrog search onboarding --first-page"
      Then only the first page of results will be printed
      And stderr will note that more results exist
      And the command will exit with code 0

    # Source: 041-cross-model-search — Scenario: Incompleteness is never silent
    @validation @wip
    Scenario: A partial result set cannot be read as complete
      Given a search where the page walk could not assemble every page
      When the result is inspected
      Then an explicit incomplete signal with its cause will be present
      And the partial set cannot be read as the complete set

    # Source: 041-cross-model-search — Proposed: plan ADR-4 mid-walk failure exit semantics (025 ADR-3)
    @wip
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the result walk fails after retrieving the first page
      When an agent runs "glassfrog search onboarding"
      Then the results retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 041-cross-model-search — Proposed: plan Risk query/types carried on every page of the walk
    @wip
    Scenario: The query and type scope are carried on every page of the walk
      Given a complete connection context with a stored token
      And the query "onboarding" scoped to "--types role" spans more than one page
      When an agent runs "glassfrog search onboarding --types role"
      Then every page request of the walk will retain "query" set to "onboarding" and "types" set to "role"
      And the command will exit with code 0
