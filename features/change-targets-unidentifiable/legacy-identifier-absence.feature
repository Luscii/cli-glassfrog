# Source: 075-legacy-identifier-request — Scenario: An agent-backed actor has no legacy number

Feature: Legacy Identifier Absence
  The legacy number is a transition-only bridge: opt-in, nullable, and dated —
  it retires when the v3 API retires. So absence is ordinary, not exceptional,
  and it arrives for three unrelated reasons: an actor backed by an agent has no
  legacy record at all, a read's embedded resources may not carry the number
  even though its own subject does, and one day the parameter will simply stop
  being answered. In none of those cases is a read a failure. What matters is
  that the CLI never lets an absence read as "this resource has no number in the
  system of record" — a confident false absence is worse than the gap this
  feature exists to close, because it argues against going to look.

  Whether a given read's embeds carry the number is a per-read observed fact,
  never a rule: the single-role read's embeds do not carry it, while the identity
  read's embedded roles do.

  This file covers absence and degradation. Its siblings:
  legacy-identifier-request.feature covers requesting and surfacing the number,
  and legacy-identifier-guard.feature covers the internal/build guard.
  (affects: Practitioner, AI agent)

  Rule: Avoid building a durable dependence on a facility that will be withdrawn

    # In order to avoid building a durable dependence on a facility that is
    # going to be withdrawn,
    # as a maintainer of the CLI,
    # I want to have the number's opt-in, nullable, and time-limited nature
    # stated where an operator meets it.

    # Source: 075-legacy-identifier-request — Scenario: An agent-backed actor has no legacy number
    @wip
    Scenario: An agent-backed actor renders its absence with the agent backing as reason
      Given an actor backed by an agent
      When the operator runs the actors read for that actor with --legacy-id and human full output
      Then the human render will show the legacy number as absent
      And it will name the actor's agent backing as the reason no number exists
      And the read will exit with code 0

    # Source: 075-legacy-identifier-request — Scenario: An agent-backed actor has no legacy number
    @wip
    Scenario: An agent-backed actor's structured output carries a bare null
      Given an actor backed by an agent
      When the operator runs the actors read for that actor with --legacy-id and structured output
      Then the structured document will carry an explicit null for the legacy number
      And it will carry no reason field alongside that null

    # Source: 075-legacy-identifier-request — Scenario: Embedded resources carry no numeric identifier
    @wip
    Scenario: A read whose embeds omit the number states it once per group
      Given a role with sub-roles and fillers
      When the operator runs the roles read for it with --legacy-id and the subroles include
      Then the role itself will carry its legacy number
      And the structured document will carry no legacy_id key on any embedded sub-role or filler
      And the human render will state once per embedded group that this read carries no legacy number for them
      And no absence marker will repeat per embedded member

    # Source: 075-legacy-identifier-request — Proposed: LEARNINGS W2 (per-read embed behaviour, found by probe during guard)
    @wip
    Scenario: A read whose embeds carry the number renders it and states nothing
      Given the caller filled several roles
      When the operator runs the me read with --legacy-id and the roles include
      Then each embedded role will carry its own legacy number
      And the human render will state nothing about embedded resources lacking the number

    # Source: 075-legacy-identifier-request — Scenario: Nothing comes back when the bridge no longer answers
    @wip
    Scenario: A retired bridge leaves the structured output without the key
      Given an API that no longer honours the include_legacy_id parameter
      When the operator runs the roles read with --legacy-id and structured output
      Then the structured document will carry no legacy_id key for any resource
      And the read will exit with code 0

    # Source: 075-legacy-identifier-request — Scenario: Nothing comes back when the bridge no longer answers
    @wip
    Scenario: A retired bridge leaves the human render showing explicit absence
      Given an API that no longer honours the include_legacy_id parameter
      When the operator runs the roles read with --legacy-id and human output
      Then every role will show its legacy number as explicitly absent
      And no diagnostic will be emitted
      And nothing in the output will claim the facility has retired

    # Source: 075-legacy-identifier-request — Proposed: plan ADR-3 decode tolerance (developer decision, LEARNINGS W5)
    @wip
    Scenario Outline: An integer-bearing legacy number is accepted in either spelling
      Given a role whose legacy number arrived as <spelling>
      When the operator runs the roles read for it with --legacy-id
      Then the number 14062695 will be rendered
      And the read will exit with code 0

      Examples:
        | spelling                     |
        | the JSON integer 14062695    |
        | the JSON string "14062695"   |

    # Source: 075-legacy-identifier-request — Proposed: plan ADR-3 decode tolerance, rejection path
    @wip
    Scenario: A non-numeric legacy number fails loudly rather than reading as absent
      Given a role whose legacy number arrived as the JSON string "not-a-number"
      When the operator runs the roles read for it with --legacy-id
      Then the CLI will report a decode failure
      And it will not render the number as explicitly absent

    # Source: 075-legacy-identifier-request — Scenario: A machine consumer can tell "not asked for" from "asked for and absent"
    @validation @wip
    Scenario: Structured output distinguishes not-requested from requested-and-absent
      Given the same role was read twice with structured output, once with --legacy-id and once without
      When a parser compares the two documents
      Then the document read without the flag will carry no legacy_id key at all
      And the document read with the flag will carry a legacy_id key holding a number or an explicit null

    # Source: 075-legacy-identifier-request — Scenario: The structured output's keys match what the API returned
    @validation @wip
    Scenario: Structured output carries the legacy_id key exactly where the response did
      Given a read that was run with --legacy-id and the raw response the API returned for it
      When each resource's legacy_id key in the structured document is compared against that response
      Then the document will carry the key exactly where the response did and nowhere else
      And no resource will carry a synthesized number, a filtered-away number, or an added reason field
