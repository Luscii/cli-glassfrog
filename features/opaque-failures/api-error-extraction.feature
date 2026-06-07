# Source: 015-api-error-extraction — Scenario: a valid Problem Details body is extracted

Feature: Opaque Failures — API Error Extraction
  When a Glassfrog call fails, the caller can't tell what went wrong or what to
  do next — Request Execution surfaces a non-2xx only as a raw status, headers,
  and body. API Error Extraction turns that into a typed error carrying the
  API's status and RFC 9457 detail, so a failed call is legible rather than
  opaque.
  (affects: AI agent, Practitioner)

  Rule: A non-2xx becomes a typed error carrying the API's status and detail
    # In order to know why a Glassfrog call failed instead of staring at a raw status and body,
    # as an AI agent driving a command,
    # I want a non-2xx response turned into a typed error carrying the API's status and human-readable detail.

    # Source: 015-api-error-extraction — Scenario: a valid Problem Details body is extracted
    Scenario: A valid Problem Details body is extracted
      Given a non-2xx response had a valid RFC 9457 Problem Details body
      When the system interprets the non-2xx outcome
      Then a typed API error carrying the HTTP status will be returned
      And the error will carry the extracted detail, title, and type

    # Source: 015-api-error-extraction — Scenario: a 404 surfaces the API's own detail
    Scenario: A 404 surfaces the API's own detail
      Given a non-2xx response had status 404 and a detail of "Not Found"
      When the system interprets the non-2xx outcome
      Then a typed API error carrying status 404 will be returned
      And the error will carry the detail "Not Found"

    # Source: 015-api-error-extraction — Scenario: a 429 is extracted but not backed off
    Scenario: A 429 is extracted without backoff
      Given a non-2xx response had status 429 with rate-limit headers
      When the system interprets the non-2xx outcome
      Then a typed API error carrying status 429 and the response headers will be returned
      And the system will not sleep, back off, or retry

    # Source: 015-api-error-extraction — Scenario: a 403 is carried generically
    Scenario: A 403 is carried generically
      Given a non-2xx response had status 403 from a plan-gated endpoint
      When the system interprets the non-2xx outcome
      Then a typed API error carrying status 403 and the API's detail will be returned
      And the system will not translate it into plan-availability guidance

    # Source: 015-api-error-extraction — Scenario: extraction reads only the outcome handed in
    @validation @wip
    Scenario: Extraction reads only the outcome handed in
      Given a non-2xx outcome was the only input
      When the system interprets it
      Then it will send no request
      And it will read no flag, environment variable, or credentials file

    # Source: 015-api-error-extraction — Scenario: no backoff is observable for a 429
    @validation @wip
    Scenario: No backoff occurs while interpreting a 429
      Given a non-2xx response had status 429
      When the system interprets the non-2xx outcome
      Then no sleep, delay, or retry will occur
      And the rate-limit headers will be carried through unchanged

  Rule: The API's detail and title are surfaced as named fields, raw body retained
    # In order to decide what to do next after a failure,
    # as an operator diagnosing a failed command,
    # I want the API's own detail and title surfaced as named fields, with the raw body still available when I need more.

    # Source: 015-api-error-extraction — Scenario: extension members are preserved without being promoted
    Scenario: Extension members are preserved without being promoted
      Given a non-2xx Problem Details body had extension members beyond the standard four
      When the system interprets the non-2xx outcome
      Then only the detail, title, and type will be surfaced as named fields
      And the raw body will be preserved so the extension members remain available

    # Source: 015-api-error-extraction — Scenario: body status disagrees with the HTTP status
    Scenario: The HTTP status wins over a disagreeing body status
      Given a non-2xx response had HTTP status 403 and a body status of 401
      When the system interprets the non-2xx outcome
      Then the typed error's authoritative status will be 403
      And the body status 401 will be carried as metadata only

    # Source: 015-api-error-extraction — Scenario: the body status never overrides the produced status
    @validation @wip
    Scenario: The produced status always equals the HTTP status
      Given a non-2xx response body status disagreed with the HTTP status
      When the produced typed error is inspected
      Then its authoritative status will equal the HTTP response status
      And the body status will appear only as carried metadata

    # Source: 015-api-error-extraction — Proposed: the API detail reaches the operator message (plan ADR-4)
    Scenario: The API detail appears in the failure message
      Given a command received a non-2xx response with a detail of "Token lacks access to this circle"
      When the command reports the failure to the operator
      Then the failure message will contain "Token lacks access to this circle"

    # Source: 015-api-error-extraction — Proposed: 401 and 403 exit with the permission code (plan ADR-3)
    Scenario: An authorization failure exits with the permission code
      Given a command received a non-2xx response with status 403
      When the command maps the failure to an exit code
      Then the command will exit with code 4

    # Source: 015-api-error-extraction — Proposed: 429 exits with the rate-limit code (plan ADR-3)
    Scenario: A rate-limited response exits with the rate-limit code
      Given a command received a non-2xx response with status 429
      When the command maps the failure to an exit code
      Then the command will exit with code 5

    # Source: 015-api-error-extraction — Proposed: non-permission non-2xx keeps the general API exit code (plan ADR-3)
    Scenario: A non-permission API error exits with the general API code
      Given a command received a non-2xx response with status 404
      When the command maps the failure to an exit code
      Then the command will exit with code 3

  Rule: A usable error is produced even when the body is unreadable
    # In order to still get a usable error when a gateway returns junk instead of a Problem Details body,
    # as an AI agent,
    # I want the system to fall back to the HTTP status rather than failing to produce any error at all.

    # Source: 015-api-error-extraction — Scenario: an empty body degrades to the HTTP status
    Scenario: An empty body degrades to the HTTP status
      Given a non-2xx response had status 500 and no body
      When the system interprets the non-2xx outcome
      Then a typed API error carrying status 500 will be returned
      And a fallback detail derived from the status will be supplied
      And the raw body will be preserved

    # Source: 015-api-error-extraction — Scenario: a non-JSON gateway body degrades gracefully
    Scenario: A non-JSON gateway body degrades gracefully
      Given a non-2xx response had status 502 with an HTML body
      When the system interprets the non-2xx outcome
      Then a typed API error carrying status 502 with a fallback detail will be returned
      And the parsing will not be conditioned on the response content type

    # Source: 015-api-error-extraction — Scenario: every non-2xx yields a typed error
    @validation @wip
    Scenario: Every non-2xx yields a typed error
      Given a non-2xx outcome whose body could not be parsed
      When the system interprets it
      Then a typed API error will always be returned
      And no path will return success or a nil error
