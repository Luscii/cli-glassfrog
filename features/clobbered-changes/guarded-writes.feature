# Source: 053-guarded-writes — Scenario: Captured version guards the write

Feature: Clobbered Changes — Guarded Writes
  Concurrent governance edits overwrite each other when writes skip version
  checks. This capability is the send half: given a version a read captured
  (Version Capture on Read, 052), it sends that version on a write as the
  If-Match precondition so the server refuses a stale write (412 Precondition
  Failed) instead of overwriting it last-write-wins. It is a setting-agnostic,
  resource-agnostic mechanism — it wires no specific write command, and it does
  not interpret the refusal (that is Stale-Write Surfacing, 054). (affects: Practitioner, Maintainer)

  Rule: A write is refused by the server when the resource changed since it was read
    # In order to avoid silently clobbering a concurrent governance edit made on my behalf,
    # as a practitioner whose work the CLI serves,
    # I want to have a write I make refused by the server when the resource changed since it was read, so my edit cannot quietly overwrite someone else's.

    # Source: 053-guarded-writes — Scenario: Captured version guards the write
    Scenario: A captured version guards the write
      Given a write request carrying a captured version of "a1b2c3"
      When the request is sent
      Then the outbound request will carry an If-Match header of "a1b2c3"

    # Source: 053-guarded-writes — Scenario: Server refuses a stale write
    Scenario: A stale guarded write is refused untouched
      Given a write request carrying a version that no longer matches the resource
      When the request is sent and the server responds with status 412
      Then the refusal will not be interpreted or relabeled
      And the outcome will flow through the existing diagnostic message and exit code unchanged

  Rule: Send a captured version as If-Match through one shared mechanism, without each write command re-deriving how
    # In order to complete the optimistic-concurrency foundation so an edit can actually be refused when the resource changed underneath it,
    # as a Maintainer building the Optimistic Concurrency capability,
    # I want to send a captured version as an If-Match precondition on a write, so the one shared mechanism turns a retained version into an enforced guard without each write command re-deriving how.

    # Source: 053-guarded-writes — Scenario: No version falls through to an unconditional write
    Scenario: A write without a version is sent unconditionally
      Given a write request that carries no captured version
      When the request is sent
      Then the outbound request will carry no If-Match header
      And the write will proceed unconditionally, exactly as before this capability existed

    # Source: 053-guarded-writes — Scenario: A delete is guarded the same way
    Scenario: A delete is guarded the same way as an update
      Given a delete request carrying a captured version of "a1b2c3"
      When the request is sent
      Then the outbound request will carry an If-Match header of "a1b2c3"
      And the precondition will be attached identically regardless of the request method

    # Source: 053-guarded-writes — Scenario: Empty captured version is not sent as a precondition
    Scenario: An empty captured version sends no precondition
      Given a write request whose captured version is empty
      When the request is sent
      Then the outbound request will carry no If-Match header
      And the write will not be refused for a malformed precondition

    # Source: 053-guarded-writes — Scenario: Weak-validator version is forwarded verbatim
    Scenario: A weak-validator version is forwarded verbatim
      Given a write request carrying a captured version that is a quoted weak validator
      When the request is sent
      Then the If-Match header will preserve the weak-validator prefix and the surrounding quotes
      And no part of the token will be stripped or normalized

    # Source: 053-guarded-writes — Scenario: Precondition composes with an existing content type
    Scenario: An If-Match precondition composes with a content type
      Given a write request that carries both a body media type and a captured version
      When the request is sent
      Then the outbound request will carry both its Content-Type header and the If-Match header
      And neither header will displace the other

    # Source: 053-guarded-writes — Scenario: The spec names no command it secretly retrofits
    @validation @wip
    Scenario: Guarded Writes wires no production write command
      Given the guarded-write send mechanism
      When it is delivered
      Then only the shared send mechanism will be introduced
      And no production write command will be wired onto it

    # Source: 053-guarded-writes — Scenario: The 412 is owned downstream, not here
    @validation @wip
    Scenario: The 412 refusal is left for downstream surfacing
      Given a guarded write the server refuses with status 412
      When the refusal is handled by this capability
      Then the refusal will be produced but not interpreted or surfaced distinctly
      And distinct surfacing will be left to Stale-Write Surfacing (054)
