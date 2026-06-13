# Source: 045-tension-discard — Scenario: Discard a live tension

Feature: Tension Discard
  A tension is a gap a role senses — recorded by `glassfrog tension create`
  (042), surfaced by `glassfrog tension list`/`get` (043), and edited by
  `glassfrog tension update` (044). This is the soft-delete counterpart:
  `glassfrog tension discard <ten-id>` removes a tension from the active record
  through a single bodyless `DELETE`, setting `discarded_at` server-side. It is a
  flagless verb leaf under the same `tension` group. The successful delete carries
  no body, so the command synthesizes its own result (the discarded id and a
  discarded marker) and renders it through the shared output seam. Because the
  delete is not REST-strict idempotent, a `404` is treated as success — a re-run
  stays safe — and a one-line advisory on stderr says whether a live tension was
  discarded (`204`) or one was already gone (`404`). It sends no `If-Match`
  precondition (last-write-wins). (affects: Practitioner)

  Rule: Retire a tension from the active record
    # In order to retire a tension that no longer belongs on the active record,
    # as a practitioner managing my governance backlog,
    # I want to remove it by id with one command.

    # Source: 045-tension-discard — Scenario: Discard a live tension
    @wip
    Scenario: A live tension is discarded
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When an agent runs "glassfrog tension discard ten_0123"
      Then the request will DELETE the tension with no body
      And the API will answer 204
      And the discard result will be printed as the result
      And stderr will note that the tension was discarded
      And the command will exit with code 0

    # Source: 045-tension-discard — Scenario: Missing tension id is rejected before any request
    @wip
    Scenario: A missing tension id is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension discard"
      Then stderr will report a usage error naming the required "<ten-id>"
      And no request will be sent
      And the command will exit with code 2

    # Source: 045-tension-discard — Scenario: More than one positional id is a usage error
    @wip
    Scenario: More than one positional id is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension discard ten_0123 ten_0456"
      Then stderr will report a usage error
      And no request will be sent
      And the command will exit with code 2

    # Source: 045-tension-discard — Scenario: No credential surfaces the not-authenticated outcome
    @wip
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tension discard ten_0123"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 045-tension-discard — Scenario: A refused permission fails loudly
    @wip
    Scenario: A refused permission fails with the API status
      Given a complete connection context with a stored token
      And the caller may not delete the tension "ten_0123"
      When an agent runs "glassfrog tension discard ten_0123"
      Then stderr will report that the discard failed and name the HTTP status
      And the command will exit with a non-zero permission code

    # Source: 045-tension-discard — Scenario: A transport failure surfaces network-unavailable
    @wip
    Scenario: A transport failure surfaces the network-unavailable outcome
      Given a complete connection context with a stored token
      And the tensions endpoint cannot be reached
      When an agent runs "glassfrog tension discard ten_0123"
      Then stderr will report the transport failure by name
      And the command will exit with the network-unavailable code

    # Source: 045-tension-discard — Proposed: plan Interactions (--output resolved first; an invalid selector fails fast before any request)
    @wip
    Scenario: An invalid output format is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension discard ten_0123 -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

    # Source: 045-tension-discard — Scenario: Discard exposes no read or write verb of its own
    @validation @wip
    Scenario: The discard verb exposes only soft-delete
      Given the implemented "glassfrog tension discard" verb
      When its behavior and help are inspected
      Then only soft-delete behavior will be available
      And no create, list, get, or update behavior will be advertised or implemented

  Rule: Clean up safely with a retry-safe discard
    # In order to clean up after a tension I captured by mistake,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to discard it and have a re-run of the same command stay safe rather than fail.

    # Source: 045-tension-discard — Scenario: Re-discarding an already-gone tension stays safe
    @wip
    Scenario: Re-discarding an already-gone tension stays safe
      Given a complete connection context with a stored token
      And a tension "ten_0123" that has already been discarded
      When an agent runs "glassfrog tension discard ten_0123"
      Then the API will answer 404
      And the outcome will be treated as success
      And the discard result will be printed as the result
      And stderr will note that the tension was already gone
      And the command will exit with code 0

    # Source: 045-tension-discard — Scenario: The 404 path leaks no not-found error
    @validation @wip
    Scenario: The 404 path leaks no not-found error
      Given a complete connection context with a stored token
      And no tension "ten_ffff" exists
      When an agent runs "glassfrog tension discard ten_ffff"
      Then no not-found error will reach the user
      And stderr will note the tension was already discarded as advisory information
      And the command will exit with code 0

    # Source: 045-tension-discard — Proposed: plan Cross-cutting non-idempotent retry (§133 isSafeMethod)
    @wip
    Scenario: A rate-limited discard is surfaced, not silently retried
      Given a complete connection context with a stored token
      And the tensions endpoint answers the discard with a rate-limit response
      When an agent runs "glassfrog tension discard ten_0123"
      Then the rate-limit will be surfaced on the first occurrence
      And the discard will not be retried
      And the command will exit with the rate-limit code

  Rule: Parse the discard outcome downstream
    # In order to feed the discard outcome into a downstream automation,
    # as an AI agent driving the CLI,
    # I want the discard to produce a structured result I can parse as JSON, even though the API returns no body.

    # Source: 045-tension-discard — Scenario: Discard result rendered as JSON
    @wip
    Scenario: The discard result is rendered as JSON
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When an agent runs "glassfrog tension discard ten_0123 -o json"
      Then the synthesized discard result will be rendered as JSON
      And the command will exit with code 0

    # Source: 045-tension-discard — Scenario: The synthesized result claims nothing the server did not return
    @validation @wip
    Scenario: The synthesized result claims nothing the server did not return
      Given a successful discard that returns no response body
      When the rendered discard result is inspected
      Then it will carry the discarded tension id and a discarded marker
      And it will not include any server-owned field such as a discarded-at timestamp
