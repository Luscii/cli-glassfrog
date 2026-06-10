# Source: 012-my-roles — Scenario: List the roles I fill

Feature: Self-Service Reads — My Roles
  The practitioner — usually via an AI agent — wants to read "what's mine"
  without naming themselves: the stored token already scopes the call to one
  person in one organization. My Roles is a self-service read on the proven
  transport chain: `glassfrog me roles` sends GET /me/roles and prints a
  reshaped projection of the roles the practitioner fills, or fails with a
  named error and the right exit code.
  (affects: Practitioner)

  Rule: List my roles with one command, without identifying myself
    # In order to see which roles I hold so I can decide where to sense a tension or raise a proposal,
    # as a practitioner whose governance work the CLI serves,
    # I want to list my roles with one command, without having to identify myself.

    # Source: 012-my-roles — Scenario: List the roles I fill
    Scenario: The roles the practitioner fills are listed
      Given a complete connection context with a stored token
      And the API would return the roles the practitioner fills
      When the practitioner runs "glassfrog me roles"
      Then each role will be printed as a projection
      And the command will exit with code 0

    # Source: 012-my-roles — Scenario: The practitioner fills no roles
    Scenario: An empty role list is a clean success
      Given a complete connection context with a stored token
      And the API would return no roles for the practitioner
      When the practitioner runs "glassfrog me roles"
      Then "No roles." will be printed to stdout
      And the command will exit with code 0

    # Source: 012-my-roles — Scenario: No usable token
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When the practitioner runs "glassfrog me roles"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no role data will be printed
      And the command will exit with code 2

    # Source: 012-my-roles — Scenario: The API cannot be reached
    Scenario: A wire failure is reported as a transport failure
      Given a complete connection context with a stored token
      And the API could not be reached
      When the practitioner runs "glassfrog me roles"
      Then the transport failure will be named on stderr
      And the command will exit with code 6

    # Source: 012-my-roles — Scenario: The API answers with a non-2xx status
    # Uses a 500 (generic server error) as the representative non-2xx: API Error
    # Extraction (015) split 401/403→permission(4) and 429→rate-limit(5), so a
    # 5xx is the faithful sample for the residual generic APIError(3) this
    # scenario pins.
    Scenario: A non-2xx response reports the read failed with its status
      Given a complete connection context with a stored token
      And the API would return a 500 response
      When the practitioner runs "glassfrog me roles"
      Then stderr will report that the read failed and name the 500 status
      And the command will exit with code 3

    # Source: 012-my-roles — Scenario: Extra arguments are rejected without an API call
    Scenario: A stray argument is rejected before any API call
      Given a complete connection context with a stored token
      When the practitioner runs "glassfrog me roles extra-argument"
      Then the invocation will be rejected as a usage error
      And no request will reach the API
      And the command will exit with code 2

    # Source: 012-my-roles — Scenario: A malformed base URL is refused before any call (architecture-informed: plan ADR-3 / interface base-URL error path)
    Scenario: A malformed base URL is refused before any call
      Given a connection context carrying a base-URL configuration error
      When the practitioner runs "glassfrog me roles"
      Then stderr will name the malformed base URL and its source
      And no request will reach the API
      And the command will exit with code 2

    # Source: 012-my-roles — Scenario: An unparseable response body fails loudly (architecture-informed: plan ADR-3 / interface decode-error path)
    # Superseded by 031 (Diagnostic Normalization) ADR-2: an unparseable 2xx body is a general API error (exit 3), not an internal error (exit 1)
    Scenario: An unparseable response body fails loudly
      Given a complete connection context with a stored token
      And the API would return a 200 response whose body cannot be parsed
      When the practitioner runs "glassfrog me roles"
      Then stderr will report that the response could not be parsed
      And the command will exit with code 3

  Rule: Each role is a concise, parseable projection
    # In order to orient on a practitioner's responsibilities before acting on their behalf,
    # as an AI agent operating the CLI,
    # I want each role returned as a concise, parseable projection carrying its name, purpose, and identifier.

    # Source: 012-my-roles — Scenario: A projected role carries its essentials, not the raw payload
    Scenario: A projected role shows its essentials only
      Given a complete connection context with a stored token
      And the API would return a role named "Marketing Lead" with a purpose, two domains, and three accountabilities
      When the practitioner runs "glassfrog me roles"
      Then the projection will show the role name, its identifier, its purpose, its domains, then its accountabilities
      And the role's fillers, tags, and classification flags will not be shown

    # Source: 012-my-roles — Scenario: Default output contains no raw API envelope
    @validation @wip
    Scenario: The default output contains no raw API envelope
      Given a successful "glassfrog me roles" run
      When the stdout output is inspected
      Then it will contain the reshaped projection only
      And it will not contain the raw data or meta JSON envelope

  Rule: Incompleteness is signalled, never silent
    # In order to trust that I am seeing every role I hold,
    # as a practitioner with many roles,
    # I want the command to tell me when the list it printed is incomplete rather than silently truncating it.

    # Source: 012-my-roles — Scenario: More roles exist than one response carried
    Scenario: An incomplete list is signalled, not silently truncated
      Given a complete connection context with a stored token
      And the API would return a first page reporting that more roles exist
      When the practitioner runs "glassfrog me roles"
      Then the roles from the response will be printed to stdout
      And an incomplete-result note will be written to stderr
      And the command will exit with code 0

    # Source: 012-my-roles — Scenario: Incompleteness is never silent
    @validation @wip
    Scenario: A partial list cannot be read as complete
      Given a run whose response did not carry every role the practitioner fills
      When the output is inspected
      Then an explicit incomplete-result signal will be present
      And the partial list will not be presentable as the complete set
