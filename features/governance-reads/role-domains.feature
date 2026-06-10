# Source: 033-role-domains — Scenario: List a role's domains

Feature: Governance Reads — Role Domains
  A domain is an area of control a role holds. Beyond the description embedded
  inline on a role, Role Domains is the addressable surface: `glassfrog domains
  <role-id>` lists the domains a role controls (walking pages to completion by
  default, searchable with --query), and `glassfrog domain <dom-id>` reads one
  domain by its own id, optionally embedding the policies scoped to it. Both
  render through the shared output seam or fail with a named error and the right
  exit code. The role ids the list consumes come from Role Reads.
  (affects: Practitioner)

  Rule: List a role's domains by id
    # In order to see exactly what a role is accountable to control,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list a role's domains by its id.

    # Source: 033-role-domains — Scenario: List a role's domains
    Scenario: A role's domains are listed
      Given a complete connection context with a stored token
      And the role "role_0123" controls several domains
      When an agent runs "glassfrog domains role_0123"
      Then each domain will be printed as a projection
      And the command will exit with code 0

    # Source: 033-role-domains — Scenario: No usable token
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog domains role_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no domain data will be printed
      And the command will exit with code 2

    # Source: 033-role-domains — Scenario: The API cannot be reached
    Scenario: An unreachable API fails as network-unavailable
      Given a complete connection context with a stored token
      And the API is unreachable at the wire
      When an agent runs "glassfrog domains role_0123"
      Then stderr will name the transport failure
      And the command will exit with code 6

    # Source: 033-role-domains — Scenario: The role controls no domains
    Scenario: A role with no domains is a clean success
      Given a complete connection context with a stored token
      And the role "role_0123" controls no domains
      When an agent runs "glassfrog domains role_0123"
      Then "No domains." will be printed to stdout
      And the command will exit with code 0

    # Source: 033-role-domains — Scenario: Default output carries no raw API envelope
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog domains role_0123" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

  Rule: Read a single domain with its governing policies
    # In order to understand a single area of control together with the policies that govern it,
    # as an AI agent assembling context before acting,
    # I want to read one domain by its id with its policies embedded inline.

    # Source: 033-role-domains — Scenario: Read a single domain by id
    Scenario: A single domain is read by id
      Given a complete connection context with a stored token
      And a domain "dom_0123" exists in the organization
      When an agent runs "glassfrog domain dom_0123"
      Then the domain's description and controlling role will be printed
      And the command will exit with code 0

    # Source: 033-role-domains — Scenario: Read a single domain with its policies embedded
    Scenario: Requested policies are embedded inline on the domain
      Given a complete connection context with a stored token
      And a domain "dom_0123" exists in the organization
      When an agent runs "glassfrog domain dom_0123 --include policies"
      Then the request will carry "include=policies"
      And the domain's policies will be printed inline within the domain
      And the command will exit with code 0

    # Source: 033-role-domains — Scenario: A single read for an unknown domain id
    Scenario: An unknown domain id fails with the API status
      Given a complete connection context with a stored token
      And no domain "dom_ffff" exists in the organization
      When an agent runs "glassfrog domain dom_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 033-role-domains — Scenario: An unsupported `--include` value is rejected without an API call
    Scenario: An unsupported include value is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog domain dom_0123 --include nonsense"
      Then stderr will name the unsupported value and the supported set
      And no API request will be sent
      And the command will exit with code 2

    # Source: 033-role-domains — Scenario: `--include` is rejected on the role-scoped list
    Scenario: An include passed to the list is a usage error
      Given a complete connection context with a stored token
      When an agent runs "glassfrog domains role_0123 --include policies"
      Then stderr will report a usage error
      And no API request will be sent
      And the command will exit with code 2

    # Source: 033-role-domains — Scenario: Embedded-policies view does not substitute for the standalone policy read
    @validation @wip
    Scenario: An embedded policies view is not a standalone read
      Given a successful "glassfrog domain dom_0123 --include policies" run
      When the result is inspected
      Then the policies will appear embedded inline on the domain
      And no standalone per-policy projection will be produced

  Rule: Search a role's domains by full-text term
    # In order to find a particular area of control on a role with many domains,
    # as a practitioner exploring the org,
    # I want to search a role's domains by a full-text term.

    # Source: 033-role-domains — Scenario: Search a role's domains
    Scenario: A role's domains are searched by a full-text term
      Given a complete connection context with a stored token
      And the role "role_0123" controls a domain matching "review"
      When the practitioner runs "glassfrog domains role_0123 --query review"
      Then the request will carry the "q" search term
      And only the matching domains will be printed
      And the command will exit with code 0

  Rule: Trust the list is whole, or be told it is incomplete
    # In order to trust that I am seeing every domain a role controls,
    # as a practitioner reviewing a large circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 033-role-domains — Scenario: A role's domains span more than one page (default walk to completion)
    Scenario: A multi-page domains list is walked to completion
      Given a complete connection context with a stored token
      And the role "role_0123" controls domains spanning three pages of API responses
      When the practitioner runs "glassfrog domains role_0123"
      Then the command will walk every page to completion
      And all domains across the pages will be printed
      And the command will exit with code 0

    # Source: 033-role-domains — Scenario: First-page opt-out stops at one page and signals more exist
    Scenario: The first-page opt-out stops at one page and signals more
      Given a complete connection context with a stored token
      And the role "role_0123" controls domains spanning more than one page
      When the practitioner runs "glassfrog domains role_0123 --first-page"
      Then only the first page of domains will be printed
      And stderr will note that more domains exist
      And the command will exit with code 0

    # Source: 033-role-domains — Proposed: plan ADR-3 mid-walk failure exit semantics
    Scenario: A mid-walk failure yields a partial set flagged incomplete
      Given a complete connection context with a stored token
      And the domains list walk fails after retrieving the first page
      When the practitioner runs "glassfrog domains role_0123"
      Then the domains retrieved so far will be printed
      And stderr will note the result is incomplete and name the cause
      And the command will exit with a non-zero code

    # Source: 033-role-domains — Scenario: List incompleteness is never silent
    @validation @wip
    Scenario: List incompleteness is never silent
      Given a domains list run that did not retrieve every page
      When the output is inspected
      Then an explicit incomplete signal with its cause will be present
      And the partial list cannot be read as the complete set
