# Source: 054-stale-write-surfacing — Scenario: A stale write surfaces under its own category and exit code

Feature: Clobbered Changes — Stale-Write Surfacing
  When a guarded write is refused because the resource changed since it was
  read, the refusal (412 Precondition Failed) lands in the generic API-error
  bucket — indistinguishable from a 404 or 500, with a misleading "check that
  the token has access" next step. This capability is the surface half: it
  branches the 412 out of that bucket into a distinct stale-write category with
  its own process exit code (7), a cause that names the precondition failure,
  and a next step that tells the operator to re-read and retry. It classifies by
  the 412 status alone (decoupled from Guarded Writes, 053, and every write
  command) and changes the surfacing of no other status. (affects: Practitioner, Maintainer)

  Rule: Detect a clobber from $? and react without parsing text
    # In order to detect that my guarded write was clobbered and react automatically — re-read the resource and retry,
    # as an AI agent driving a write on a practitioner's behalf,
    # I want a 412 Precondition Failed surfaced under its own exit code and with a next step that tells me to re-read and retry, so I can branch on $? without parsing text.

    # Source: 054-stale-write-surfacing — Scenario: A stale write surfaces under its own category and exit code
    Scenario: A stale write surfaces under its own exit code
      Given a guarded write the server refused with status 412
      When the failure is surfaced
      Then the process will exit with the stale-write code 7
      And the code will be distinct from the generic API-error code 3

    # Source: 054-stale-write-surfacing — Scenario: The next step points to re-read and retry
    Scenario: The next step points the operator to re-read and retry
      Given a guarded write the server refused with status 412
      When the failure is surfaced
      Then the next step will tell the operator to re-read the resource for its current version and retry the write
      And it will not be the generic "check that the token has access" step

    # Source: 054-stale-write-surfacing — Scenario: Classification ignores whether this CLI sent a precondition
    Scenario: Classification ignores whether a precondition was sent
      Given a status 412 surfaced on a request that carried no If-Match header
      When the failure is surfaced
      Then it will still be classified as a stale write from the 412 status alone
      And the classification will not depend on the command or the resource

    # Source: 054-stale-write-surfacing — Scenario: 412 is distinct from the generic bucket
    @validation @wip
    Scenario: A stale write is distinct from the generic bucket
      Given a status 412 outcome and a status 500 outcome
      When each is surfaced
      Then the 412 will carry the stale-write category and exit code 7
      And the 500 will keep the generic API-error category and exit code 3

    # Source: 054-stale-write-surfacing — Scenario: The capability surfaces but does not recover
    @validation @wip
    Scenario: The capability surfaces without recovering
      Given a guarded write the server refused with status 412
      When the failure is surfaced
      Then only a category, a cause, a next step, and an exit code will be produced
      And no re-read, retry, or back-off will be performed

  Rule: Understand why my edit was refused
    # In order to understand why an edit I made was refused,
    # as a practitioner whose work the CLI serves,
    # I want the refusal explained as "the resource changed since you read it — re-read and retry" rather than a generic API error, so I know my change was protected, not broken.

    # Source: 054-stale-write-surfacing — Scenario: The cause names the stale write
    Scenario: The cause names the stale write
      Given a status 412 whose response body carried an error detail
      When the failure is surfaced
      Then the cause will surface the API's own detail
      And the failure will be identified as a precondition failure from the resource changing since it was read

    # Source: 054-stale-write-surfacing — Scenario: A 412 with no readable detail derives its cause from the status
    Scenario: A 412 without readable detail derives its cause from the status
      Given a status 412 whose response carried no readable detail
      When the failure is surfaced
      Then the cause will be derived from the 412 status rather than invented
      And the stale-write category, exit code, and re-read next step will still be assigned

  Rule: Surface the refusal distinctly through the one shared diagnostic pipeline
    # In order to complete the Optimistic Concurrency capability so a refused write is finally legible end to end,
    # as a Maintainer building the Optimistic Concurrency solution,
    # I want the 412 that Guarded Writes (053) can produce surfaced distinctly through the one shared diagnostic pipeline, so capture → send → surface is whole.

    # Source: 054-stale-write-surfacing — Scenario: Another non-2xx is unaffected
    Scenario: Another non-2xx status is unaffected
      Given a status 404 outcome
      When the failure is surfaced
      Then it will keep the generic API-error category and exit code 3
      And only the 412 status will be branched out of the generic bucket

    # Source: 054-stale-write-surfacing — Scenario: Adding the stale-write code renumbers no existing code
    Scenario: Adding the stale-write code renumbers no existing code
      Given the published exit codes 0 through 6
      When the stale-write category is registered
      Then it will take the previously-unused code 7
      And every existing code will keep the meaning it had before

    # Source: 054-stale-write-surfacing — Scenario: No existing surfacing drifts
    @validation @wip
    Scenario: No existing surfacing drifts
      Given the surfacing of statuses 401, 403, 404, 429, and 500 before this capability
      When the 412 branch is added
      Then each of those statuses will keep its prior category, exit code, cause, and next step
