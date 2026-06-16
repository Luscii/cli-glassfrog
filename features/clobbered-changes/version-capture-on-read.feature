# Source: 052-version-capture-on-read — Scenario: Version captured from a single tension read

Feature: Clobbered Changes — Version Capture on Read
  Concurrent governance edits overwrite each other when writes skip version
  checks. This capability is the foundation: it captures the resource version
  (the ETag a single-resource read carries) and makes it available in-process so
  a later guarded write can send it back as If-Match. It captures and exposes
  only — sending the precondition is Guarded Writes (053). (affects: Practitioner, Maintainer)

  Rule: Retain the resource version at read time so the eventual write can detect a change
    # In order to avoid silently clobbering a concurrent governance edit on my behalf,
    # as a practitioner whose work the CLI serves,
    # I want to have the resource version retained at read time, so the eventual write can detect that the resource changed under me.

    # Source: 052-version-capture-on-read — Scenario: Version captured from a single tension read
    Scenario: A single-resource read retains its version
      Given a single-resource read of a tension whose response carried an ETag of "a1b2c3"
      When the read completes
      Then the captured version will be "a1b2c3"

    # Source: 052-version-capture-on-read — Scenario: No version present on the response
    Scenario: A response without an ETag retains no version
      Given a single-resource read whose response carried no ETag header
      When the read completes
      Then no version will be captured
      And the read will still succeed and render normally

    # Source: 052-version-capture-on-read — Scenario: Failed read captures nothing
    Scenario: A rejected read retains no version
      Given a single-resource read that the server rejected with status 404
      When the failure is handled
      Then no version will be captured
      And the existing diagnostic message and exit code will be unchanged

  Rule: Capture a forwardable version once, so each write command need not re-derive it
    # In order to keep a foundation in place that lets later edits be guarded against intervening changes,
    # as a Maintainer building the Optimistic Concurrency capability,
    # I want to capture the version a read already carries, so a guarded write has a value to send without each write command re-deriving it.

    # Source: 052-version-capture-on-read — Scenario: Mechanism is resource-agnostic
    Scenario: Version capture is resource-agnostic
      Given a single-resource read of a role whose response carried an ETag of "r9s8t7"
      When the read completes
      Then the captured version will be "r9s8t7"
      And the capture will behave identically to a tension read

    # Source: 052-version-capture-on-read — Scenario: Captured version does not change read output
    Scenario: Capturing a version leaves rendered output unchanged
      Given a single-resource read whose response carried an ETag of "a1b2c3"
      When the read is rendered in any output format
      Then the rendered output will be byte-for-byte what it was before version capture existed
      And the captured version will be present only on the in-process result

    # Source: 052-version-capture-on-read — Scenario: Collection read yields no per-resource version
    Scenario: A list read retains no per-resource version
      Given a read that returned a list of tensions with a collection-level ETag
      When the read completes
      Then no per-resource version will be captured for any item in the list

    # Source: 052-version-capture-on-read — Scenario: Version token is captured verbatim
    Scenario: A weak-validator version is captured verbatim
      Given a single-resource read whose ETag was a quoted weak validator
      When the read completes
      Then the captured version will preserve the weak-validator prefix and the surrounding quotes
      And no part of the token will be stripped or normalized

    # Source: 052-version-capture-on-read — Scenario: Read contract is provably unchanged
    @validation @wip
    Scenario: Adding version capture changes no read contract
      Given the read commands that existed before version capture
      When version capture is added
      Then no user-facing output, exit code, or diagnostic will change

    # Source: 052-version-capture-on-read — Scenario: No precondition leaks into a request
    @validation @wip
    Scenario: No If-Match header is sent by version capture
      Given the version-capture mechanism
      When any read or any existing write executes
      Then no If-Match header will be sent by anything this capability introduces
