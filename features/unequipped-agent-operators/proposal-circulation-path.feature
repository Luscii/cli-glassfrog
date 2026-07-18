# Source: 068-proposal-circulation-path — Scenario: Advance a draft into circulation

Feature: Proposal Circulation Path
  The AI agent driving the CLI has no packaged operating knowledge, so to move a
  prepared draft through the consent lifecycle it must hand-assemble the
  transition commands — and the two writes that most need care, the gated
  governance transitions — without guidance. Proposal Circulation Path is the
  fifth operator path on the Agent Operating Surface and the second to cross the
  Write-Safety Guardrail, this time twice: a thin, discoverable
  "proposal-circulation" skill (when to reach for it + the workflow) that
  delegates to a "proposal-circulator" agent under `plugin/agents/`. The agent
  runs the workflow in its own isolated context — grounding the act in the
  proposal as the server returns it, situating against the circle's in-flight
  proposals where relevant — and advances the draft into circulation or withdraws
  the circulating proposal back to draft through the guardrail-confirmed bodyless
  transitions, or monitors without writing. It returns a drawn-together
  circulation record carrying the prp_ id. It composes only commands the CLI
  already exposes; its reads inform but never gate — the server is the only
  transition authority. Its two writes are the gated propose and withdraw — never
  create (drafting), never respond (the response side), never a tension write,
  never an authority verdict. A best-effort drift guard in `internal/build` keeps
  the named leaves truthful to the shipped CLI and pins both transitions'
  membership in the guardrail's gated set.
  (affects: AI agent, Practitioner)

  Rule: Advance a prepared draft without hand-assembling the transition
    # In order to open the consent window on a draft I have prepared without
    # hand-assembling the transition command,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to follow a guided path that advances the draft into circulation
    # and shows me where it stands.

    # Source: 068-proposal-circulation-path — Scenario: Advance a draft into circulation
    Scenario: A draft proposal advances into circulation
      Given a draft proposal's prp_ id whose available transitions included propose
      When the proposal-circulator surfaces it for confirmation and runs the propose through the confirmed write flow
      Then it will return the proposal now in proposed_outside_meeting, carrying its response deadline and the implicit no-objection
      And the record will carry the prp_ id so the proposal can be monitored or withdrawn

    # Source: 068-proposal-circulation-path — Scenario: The transition must be confirmed before it crosses the boundary
    Scenario: An unconfirmed transition leaves the record untouched
      Given a proposal ready to propose or withdraw
      When the proposal-circulator reaches the transition
      Then it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the proposal first
      And no transition will happen when the write is not confirmed

    # Source: 068-proposal-circulation-path — Scenario: A transition is rejected
    Scenario: A rejected transition fabricates no state
      Given a propose attempt whose transition the server did not allow
      When the proposal-circulator runs the transition through the confirmed write flow
      Then it will surface the API failure by name
      And it will transition nothing
      And it will fabricate no state the record does not contain

    # Source: 068-proposal-circulation-path — Scenario: The path reads to inform, not to gate
    Scenario: A stale snapshot does not stop a transition
      Given a proposal whose available transitions the proposal-circulator had read
      When it advances or withdraws the proposal
      Then it will issue the transition and let the server authorize
      And it will surface a 422 refusal plainly when the server refuses
      And it will not pre-gate the call on the read snapshot

    # Source: 068-proposal-circulation-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    Scenario: The circulator is reachable once the plugin registers it
      Given the plugin was present with the proposal-circulator agent registered
      When the proposal-circulation skill delegates a circulation act
      Then the circulator will run the workflow in its own context
      And it will return only the circulation record to the caller

    # Source: 068-proposal-circulation-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    Scenario: A missing circulator degrades the path to guidance
      Given the plugin was present but the proposal-circulator agent was absent or unregistered
      When the proposal-circulation skill is consulted for a circulation act
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

    # Source: 068-proposal-circulation-path — Scenario: Both gated transitions are routed through the guardrail
    @validation
    Scenario: The path routes both writes through the guardrail
      Given the path's treatment of the propose and withdraw transitions
      When each is inspected against the Write-Safety Guardrail
      Then both will run through the confirmed write flow
      And the path will not issue either gated transition as if it were ungated

    # Source: 068-proposal-circulation-path — Scenario: No invented surface
    @validation @wip
    Scenario: The path names no command the CLI lacks
      Given the produced proposal-circulation-path content
      When every command it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no command the CLI does not expose

    # Source: 068-proposal-circulation-path — Scenario: Reads inform, never gate
    @validation
    Scenario: The path never pre-gates a transition client-side
      Given the path's use of available transitions
      When it is inspected for a client-side transition gate
      Then it will read to show the proposer where the proposal stands
      And it will issue the transition and let the server authorize
      And it will not pre-gate the call client-side

  Rule: See where a circulating proposal stands, drawn together
    # In order to know whether my circulating proposal is progressing toward
    # acceptance,
    # as a practitioner (served through the agent),
    # I want the path to surface its response summary and response deadline
    # drawn together rather than my reading raw command output.

    # Source: 068-proposal-circulation-path — Scenario: Monitor a circulating proposal
    Scenario: A circulating proposal is monitored as one picture
      Given the prp_ id of a proposal already circulating
      When the proposal-circulator reads it back
      Then it will surface the response summary, response deadline, and available transitions drawn together
      And it will not compute acceptance itself

    # Source: 068-proposal-circulation-path — Scenario: A monitoring read fails
    Scenario: A failed monitoring walk yields a partial picture
      Given a monitoring step in which the proposal list read failed mid-walk
      When the proposal-circulator continues
      Then it will surface what the failure was
      And it will present what it gathered so far, flagged incomplete
      And it will not invent the missing data or abandon the whole result

    # Source: 068-proposal-circulation-path — Scenario: A response belongs to the response side
    Scenario: A consent response is handed to the response side
      Given a circulating proposal awaiting consent
      When a member wants to record a no-objection or bring-to-meeting response
      Then the proposal-circulator will name the response side as where that act belongs
      And it will not record the response itself

    # Source: 068-proposal-circulation-path — Scenario: Circulation only, no response recording
    @validation
    Scenario: The path records no consent response
      Given the proposal-circulation skill and agent content
      When it is inspected for any no-objection or bring-to-meeting record step
      Then it will contain none
      And recording a response will remain the response side's act

    # Source: 068-proposal-circulation-path — Scenario: Synthesized, not raw
    @validation
    Scenario: The result is a synthesized circulation picture, not raw output
      Given the record the proposal-circulator returned
      When it is compared against the raw command output
      Then it will be a drawn-together circulation picture carrying the prp_ id
      And it will not be a concatenation of unsynthesized dumps

  Rule: Pull a circulating proposal back to draft without losing my place
    # In order to amend a proposal that is already circulating without losing
    # my place,
    # as an AI agent operating the write flow,
    # I want the path to withdraw it back to draft and carry the prp_ id back
    # to the Proposal Drafting Path.

    # Source: 068-proposal-circulation-path — Scenario: Withdraw a circulating proposal back to draft
    Scenario: A circulating proposal is withdrawn back to draft
      Given a circulating proposal the proposer wanted to amend
      When the proposal-circulator surfaces it for confirmation and runs the withdraw through the confirmed write flow
      Then it will return the proposal now back in draft
      And it will hand the prp_ id back to the Proposal Drafting Path for re-editing

    # Source: 068-proposal-circulation-path — Proposed: independent confirmations (plan ADR-3) — each transition is its own gated act
    Scenario: Two transitions in one session confirm twice
      Given a session that advanced a draft and later withdrew the circulating proposal
      When the proposal-circulator runs each transition
      Then each transition will pass through its own confirmed write flow
      And neither confirmation will batch or pre-authorize the other

    # Source: 068-proposal-circulation-path — Scenario: Circulating, not judging or coaching
    @validation
    Scenario: The path circulates without judging authority or coaching
      Given the proposal-circulation skill and agent content
      When it is inspected for an authority verdict or Holacracy coaching
      Then it will not rule on whether the change is within authority
      And it will not advise on governance craft
