# Source: 056-proposal-reads — Scenario: List the proposals visible to the caller

Feature: Proposal Reads
  A proposal is a governance change request carrying `changes`, anchored to a
  tension — the only sanctioned path to alter governance structure. This is the
  read half of the write flow: `glassfrog proposal list` lists the proposals
  visible to the caller (walking pages to completion by default, optionally
  narrowed by status, circle, proposer, or date), and `glassfrog proposal get
  <prp-id>` reads one proposal with its changes, aggregate response summary, and
  available transitions. They are verb leaves under a `proposal` group (shared
  with `proposal create`). Reads are not Premium-gated. Both render through the
  shared output seam or fail with a named error and the right exit code; a
  partial list is never silently presented as whole; and no per-person response
  attribution is ever surfaced.
  (affects: Practitioner)

  Rule: See which proposals are in flight before acting
    # In order to see which proposals are in flight in a circle before I act,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to list proposals, narrowed to a circle or status, with one command.

    # Source: 056-proposal-reads — Scenario: List the proposals visible to the caller
    Scenario: The visible proposals are listed
      Given a complete connection context with a stored token
      And several proposals are visible to the caller
      When an agent runs "glassfrog proposal list"
      Then the request will read the proposals endpoint
      And each proposal will be printed as a projection
      And the command will exit with code 0

    # Source: 056-proposal-reads — Scenario: Narrow the list to a circle and a status
    Scenario: A circle-and-status narrowed list is requested
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal list --role-id role_0123 --status proposed_outside_meeting"
      Then the request will send "role_id" as "role_0123" and "status" as "proposed_outside_meeting"
      And only the matching proposals will be printed
      And the command will exit with code 0

    # Source: 056-proposal-reads — Scenario: No proposals are visible
    Scenario: An empty visible set is a clean success
      Given a complete connection context with a stored token
      And no proposals are visible to the caller
      When an agent runs "glassfrog proposal list"
      Then "no proposals" will be printed to stdout
      And the command will exit with code 0

    # Source: 056-proposal-reads — Scenario: No usable credential
    Scenario: A missing token fails as a not-authenticated usage error
      Given no usable token is available to the CLI
      When an agent runs "glassfrog proposal list"
      Then stderr will report "not authenticated" and point to "glassfrog auth login"
      And no proposal data will be printed
      And the command will exit with code 2

    # Source: 056-proposal-reads — Scenario: Output is structured, not pre-rendered
    @validation @wip
    Scenario: Default output carries no raw API envelope
      Given a successful "glassfrog proposal list" run under the default human format
      When the output is inspected
      Then it will show the reshaped projection only
      And it will not contain the raw "data" or "meta" JSON envelope

    # Source: 056-proposal-reads — Proposed: an invalid --output is rejected before any request (resolve-first ordering)
    Scenario: An invalid output format is rejected before any list request
      Given a complete connection context with a stored token
      When an agent runs "glassfrog proposal list -o xml"
      Then stderr will report a usage error and name the rejected output value "xml"
      And no request will be sent
      And the command will exit with code 2

  Rule: Read a proposal's full detail by its id
    # In order to read a proposal's full detail — its changes, how the circle has
    # responded, and what I can do with it next,
    # as a practitioner operating the governance write flow,
    # I want to fetch a single proposal by its prp_ id.

    # Source: 056-proposal-reads — Scenario: Read a single proposal with full detail
    Scenario: A single proposal is read with full detail
      Given a complete connection context with a stored token
      And a proposal "prp_0123" exists
      When a practitioner runs "glassfrog proposal get prp_0123"
      Then the proposal's status, changes, response summary, and available transitions will be printed
      And the command will exit with code 0

    # Source: 056-proposal-reads — Scenario: Proposal id does not exist
    Scenario: An unknown proposal id fails with the API status
      Given a complete connection context with a stored token
      And no proposal "prp_ffff" exists
      When a practitioner runs "glassfrog proposal get prp_ffff"
      Then stderr will report that the read failed and name the HTTP status
      And the command will exit with a non-zero API-error code

    # Source: 056-proposal-reads — Scenario: List filter on the single read is rejected
    Scenario: A list filter on the single read is a usage error
      Given a complete connection context with a stored token
      When a practitioner runs "glassfrog proposal get prp_0123 --status draft"
      Then stderr will report a usage error
      And no request will be sent
      And the command will exit with code 2

    # Source: 056-proposal-reads — Scenario: No per-person response attribution is reconstructed
    @validation @wip
    Scenario: A read proposal shows only aggregate response counts
      Given a successful "glassfrog proposal get prp_0123" run
      When the rendered proposal is inspected in every format
      Then only the aggregate response counts will be shown
      And no field will attribute a response to an individual

    # Source: 056-proposal-reads — Scenario: The read surface never reaches into the write verbs
    @validation @wip
    Scenario: The read surface surfaces transitions without invoking them
      Given a read proposal that lists "propose" among its available transitions
      When the read commands are exercised for any create-, propose-, withdraw-, or respond-style behavior
      Then no such behavior will be present under "list" or "get"
      And the available transitions will be printed but never invoked

  Rule: Track my own proposals through the response window
    # In order to check whether my own proposal has cleared the response window,
    # as a proposer,
    # I want to filter the list to proposals I created, or to a status, or to recently-proposed ones.

    # Source: 056-proposal-reads — Proposed: proposer-id filter (Behavioral Accord: Filters (list))
    Scenario: The list is narrowed to the caller's own proposals
      Given a complete connection context with a stored token
      When a proposer runs "glassfrog proposal list --proposer-id per_0123"
      Then the request will send "proposer_id" as "per_0123"
      And only that proposer's proposals will be printed
      And the command will exit with code 0

    # Source: 056-proposal-reads — Scenario: Unsupported status value is rejected before any request
    Scenario: An unsupported status value is rejected before any request
      Given a complete connection context with a stored token
      When a proposer runs "glassfrog proposal list --status open"
      Then stderr will report a usage error naming the value "open" and the supported set
      And no request will be sent
      And the command will exit with code 2

    # Source: 056-proposal-reads — Scenario: An unsupported status costs no request
    @validation @wip
    Scenario: A rejected status filter sends nothing over the wire
      Given a "--status" value outside the proposal status vocabulary
      When the list command runs
      Then the value will be rejected before the connection is assembled
      And a transport tripwire will confirm no request was issued

  Rule: Trust the list is complete
    # In order to trust I am seeing every proposal that matters,
    # as a practitioner in a busy circle,
    # I want the list to walk to completion, or to tell me plainly when it is incomplete.

    # Source: 056-proposal-reads — Scenario: Paginated list with first-page opt-out
    Scenario: The first-page opt-out signals more proposals exist
      Given a complete connection context with a stored token
      And the visible proposals span more than one page
      When an agent runs "glassfrog proposal list --first-page"
      Then only the first page of proposals will be printed
      And stderr will note that more proposals exist than shown
      And the command will exit with code 0
