# Source: 044-tension-update — Scenario: Edit a tension's body

Feature: Tension Update
  A tension is a gap a role senses — recorded by `glassfrog tension create`
  (042) and surfaced by `glassfrog tension list`/`get` (043). This is the edit
  counterpart: `glassfrog tension update <ten-id>` changes the fields of an
  existing tension — its body, label, status, or meeting-type — through a single
  partial `PATCH`, sending only the fields supplied and leaving the rest
  untouched. It is a verb leaf under the same `tension` group as `create`. Unlike
  capture it can set status (including the `archived` transition), it requires at
  least one editable field, and it refuses a blank body. It renders the updated
  tension through the shared output seam or fails with a named error and the
  right exit code, and it never sends an `If-Match` precondition (last-write-wins).
  (affects: Practitioner)

  Rule: Fix a poorly-worded or mis-titled tension
    # In order to fix a tension I worded poorly or mis-titled,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to edit its body or label by id with one command.

    # Source: 044-tension-update — Scenario: Edit a tension's body
    Scenario: A tension's body is edited
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When an agent runs "glassfrog tension update ten_0123 --body \"Roadmap updates lag behind shipped work.\""
      Then the request will PATCH the tension with the new body only
      And the updated tension will be printed as the result
      And the command will exit with code 0

    # Source: 044-tension-update — Scenario: Unknown tension id
    Scenario: An unknown tension id fails with the API status
      Given a complete connection context with a stored token
      And no tension "ten_ffff" exists
      When an agent runs "glassfrog tension update ten_ffff --body \"New wording.\""
      Then stderr will report that the update failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 044-tension-update — Scenario: Whitespace-only body is treated as empty
    Scenario: A whitespace-only body is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension update ten_0123 --body \"   \""
      Then stderr will report a usage error that a body cannot be blanked
      And no request will be sent
      And the command will exit with code 2

    # Source: 044-tension-update — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog tension update ten_0123 --body \"New wording.\""
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no request will be sent
      And the command will exit with code 2

    # Source: 044-tension-update — Scenario: Update edits fields and nothing else
    @validation @wip
    Scenario: The update verb exposes only field edits
      Given the implemented "glassfrog tension update" verb
      When its behavior and help are inspected
      Then only field edits will be available
      And no create, list, get, or discard behavior will be advertised or implemented

    # Source: 044-tension-update — Proposed: plan Cross-cutting non-idempotent retry (§133 isSafeMethod)
    @wip
    Scenario: A rate-limited update is surfaced, not silently retried
      Given a complete connection context with a stored token
      And the update request is answered with a rate-limit response
      When an agent runs "glassfrog tension update ten_0123 --body \"New wording.\""
      Then the rate-limit will be surfaced on its first occurrence
      And the PATCH will not be automatically retried
      And the command will exit with the rate-limit code

  Rule: Retire a finished tension
    # In order to retire a tension I have finished working,
    # as a practitioner managing my governance backlog,
    # I want to move it to archived through the update command.

    # Source: 044-tension-update — Scenario: Archive a tension via status transition
    Scenario: A tension is archived through a status transition
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When a practitioner runs "glassfrog tension update ten_0123 --status archived"
      Then the value will be accepted as a supported status
      And the request will carry "status" set to "archived"
      And the updated tension will be printed as the result
      And the command will exit with code 0

    # Source: 044-tension-update — Scenario: Unsupported status is rejected before any request
    Scenario: An unsupported status is rejected as a usage error
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog tension update ten_0123 --status open"
      Then stderr will report "open" and list the supported set "unprocessed, processed, archived"
      And no request will be sent
      And the command will exit with code 2

    # Source: 044-tension-update — Scenario: An unsupported status or meeting-type costs no request
    @validation @wip
    Scenario: A rejected enum value issues no request
      Given a transport tripwire that records whether any request is sent
      When a practitioner runs "glassfrog tension update ten_0123 --status open"
      Then the command will be rejected before any context assembly
      And the tripwire will confirm no request was issued

  Rule: Reroute a tension to the right forum
    # In order to reroute a tension to the right forum after I have reconsidered it,
    # as a practitioner,
    # I want to change its meeting-type without recreating it.

    # Source: 044-tension-update — Scenario: Change label and meeting-type together
    Scenario: A label and meeting-type are changed together
      Given a complete connection context with a stored token
      And a tension "ten_0123" exists
      When a practitioner runs "glassfrog tension update ten_0123 --label \"Roadmap drift\" --meeting-type governance"
      Then the request will carry "label" and "meeting_type" set to "governance" and no other fields
      And the updated tension will be printed as the result
      And the command will exit with code 0

    # Source: 044-tension-update — Scenario: Only supplied fields are sent
    @validation @wip
    Scenario: A partial update sends only the supplied fields and no precondition
      Given a complete connection context with a stored token
      And an update that supplies a subset of the editable fields
      When the request built for the update is inspected
      Then its body will carry only the supplied fields
      And no "If-Match" header will be sent

  Rule: Reject an update that changes nothing
    # In order to avoid issuing a request that changes nothing,
    # as an AI agent driving the CLI,
    # I want an update with no field flags rejected before any request is made.

    # Source: 044-tension-update — Scenario: No editable field is rejected before any request
    Scenario: An update with no editable field is rejected before any request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension update ten_0123"
      Then stderr will report a usage error naming that at least one field is required
      And no request will be sent
      And the command will exit with code 2

    # Source: 044-tension-update — Proposed: plan ADR-3 send-set precondition (presence + non-empty)
    @wip
    Scenario: An update whose only flag is empty-valued is rejected as changing nothing
      Given a complete connection context with a stored token
      When an agent runs "glassfrog tension update ten_0123 --label \"\""
      Then stderr will report a usage error naming that at least one field is required
      And no request will be sent
      And the command will exit with code 2
