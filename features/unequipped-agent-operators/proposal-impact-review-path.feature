# Source: 069-proposal-impact-review-path — Scenario: Review a circulating proposal's impact on the operator's roles

Feature: Proposal Impact Review Path
  The AI agent driving the CLI has no packaged operating knowledge, so when a
  proposal circulates for the practitioner's consent it must hand-assemble the
  cross-reads — the proposal's change set against the operator's own roles,
  actions, and projects — and the response write, without guidance. Proposal
  Impact Review Path is the sixth and last operator path on the Agent Operating
  Surface, the responder's side of the consent window: a thin, discoverable
  "proposal-impact-review" skill (when to reach for it + the workflow) that
  delegates the review to a "proposal-impact-reviewer" agent under
  `plugin/agents/`. The agent runs the read traversal in its own isolated
  context — the circulating proposal and its change set, the operator's
  footprint through the me reads, current-vs-proposed read-backs for the
  governance the change touches — and returns a drawn-together impact picture
  that shows the operator what the new world looks like for their work, useful
  even when no objection surfaces. The picture informs but never decides: it
  carries no verdict, and the one gated write — recording the operator's
  explicitly chosen no-objection or bring-to-meeting through the Write-Safety
  Guardrail's confirmed flow — runs in the caller's context, after the operator
  decides, never inside the reviewer. It composes only commands the CLI already
  exposes — never propose or withdraw (circulation), never create (drafting),
  never an authority verdict. A best-effort drift guard in `internal/build`
  keeps the named leaves truthful to the shipped CLI and pins the respond
  leaf's membership in the guardrail's gated set.
  (affects: AI agent, Practitioner)

  Rule: See what a circulating proposal would change for my own work
    # In order to understand what a circulating proposal would change for my
    # own roles and work before I answer it,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to follow a guided path that reads the proposal's change set and
    # draws it against my current governance.

    # Source: 069-proposal-impact-review-path — Scenario: Review a circulating proposal's impact on the operator's roles
    Scenario: A circulating proposal is reviewed against the operator's footprint
      Given the prp_ id of a proposal circulating for consent
      When the proposal-impact-reviewer reads the change set back and draws it against the operator's me roles, actions, and projects
      Then it will present a drawn-together impact picture showing which of the operator's roles the change touches and how
      And it will record no response

    # Source: 069-proposal-impact-review-path — Scenario: A review read fails mid-picture
    Scenario: A failed review read yields a partial picture
      Given a review in which an affected-role read failed
      When the proposal-impact-reviewer continues
      Then it will surface what the failure was
      And it will present what it gathered so far, flagged incomplete
      And it will not invent the missing data or abandon the whole review

    # Source: 069-proposal-impact-review-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    Scenario: The reviewer is reachable once the plugin registers it
      Given the plugin was present with the proposal-impact-reviewer agent registered
      When the proposal-impact-review skill delegates a review
      Then the reviewer will run the read traversal in its own context
      And it will return only the impact picture to the caller

    # Source: 069-proposal-impact-review-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    Scenario: A missing reviewer degrades the path to guidance
      Given the plugin was present but the proposal-impact-reviewer agent was absent or unregistered
      When the proposal-impact-review skill is consulted for a review
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

    # Source: 069-proposal-impact-review-path — Scenario: No invented surface
    @validation
    Scenario: The path names no command the CLI lacks
      Given the produced proposal-impact-review-path content
      When every command it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no command the CLI does not expose

    # Source: 069-proposal-impact-review-path — Scenario: Synthesized, not raw
    @validation
    Scenario: The result is a synthesized impact picture, not raw output
      Given the picture the proposal-impact-reviewer returned
      When it is compared against the raw command output
      Then it will be a drawn-together impact picture relating the change set to the operator's footprint
      And it will not be a concatenation of unsynthesized dumps

  Rule: Judge for myself whether the change creates a tension for my work
    # In order to decide whether a proposal creates a tension for my work,
    # as a practitioner (served through the agent),
    # I want the path to show me where the change intersects the roles,
    # actions, and projects I hold — rather than my cross-reading raw command
    # output myself.

    # Source: 069-proposal-impact-review-path — Scenario: The change does not touch the operator's governance
    Scenario: A no-impact review is a load-bearing result
      Given a circulating proposal whose change set fell outside the operator's roles, actions, and projects
      When the proposal-impact-reviewer draws the impact picture
      Then it will report that the change does not touch the operator's current governance
      And it will still show what the proposal would change overall

    # Source: 069-proposal-impact-review-path — Scenario: The review is complete without a response
    Scenario: A review stands on its own without a response
      Given an operator who had reviewed a proposal's impact but was not ready to answer
      When the path finishes the review
      Then it will present the impact picture and record no response
      And the review will be a useful result on its own

    # Source: 069-proposal-impact-review-path — Proposed: footprint honesty (plan ADR-4a / interface footprint_coverage) — an incomplete me roles read qualifies the picture
    Scenario: An incomplete footprint qualifies the no-impact conclusion
      Given a footprint read in which me roles signalled that more roles exist than shown
      When the proposal-impact-reviewer draws the impact picture
      Then it will carry the incompleteness forward as footprint coverage
      And a no-impact conclusion will read as not found in the roles visible to this read
      And it will never state an unqualified negative over a known-incomplete list

    # Source: 069-proposal-impact-review-path — Scenario: Reviews inform, never decide
    @validation
    Scenario: The picture carries no verdict and chooses no response
      Given the path's use of the change set and the operator's footprint
      When it is inspected for a fabricated objection verdict or an auto-chosen response
      Then it will draw the impact together for the operator to judge
      And it will neither rule that an objection is required nor answer on the operator's behalf

    # Source: 069-proposal-impact-review-path — Scenario: Reviewing, not judging authority or coaching
    @validation
    Scenario: The path reviews without judging authority or coaching
      Given the proposal-impact-review skill and agent content
      When it is inspected for a proposer-side authority verdict or Holacracy coaching
      Then it will not rule on whether the change is within the proposer's authority
      And it will not advise on how to weigh an objection

  Rule: Answer a reviewed proposal without hand-assembling the response
    # In order to answer a proposal once I have reviewed its impact without
    # hand-assembling the response command,
    # as an AI agent operating the write flow,
    # I want the path to record my no-objection or bring-to-meeting through the
    # confirmed write flow and return the recorded response.

    # Source: 069-proposal-impact-review-path — Scenario: Record no-objection after review finds no concern
    Scenario: A no-objection is recorded through the confirmed write flow
      Given a reviewed proposal whose change did not create a tension for the operator's work
      When the path surfaces the proposal and no-objection for confirmation and runs the respond through the confirmed write flow
      Then it will return the recorded response with its prr_ id and the parent proposal's status at the time of response
      And the returned parent status will read accepted when that response completed the expected set

    # Source: 069-proposal-impact-review-path — Scenario: Record bring-to-meeting when review surfaces a concern
    Scenario: A bring-to-meeting is recorded and blocks auto-acceptance
      Given a reviewed proposal whose change landed on a role the operator fills in a way they wanted discussed
      When the path surfaces the proposal and bring-to-meeting for confirmation and runs the respond through the confirmed write flow
      Then it will return the recorded response, which persists on the proposal and blocks auto-acceptance
      And the path will stop, leaving advancing or withdrawing to the Proposal Circulation Path

    # Source: 069-proposal-impact-review-path — Scenario: The response must be confirmed before it crosses the boundary
    Scenario: An unconfirmed response leaves the record untouched
      Given a proposal the operator was ready to answer
      When the path reaches the respond write
      Then it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the proposal and the chosen response first
      And no response will be recorded when the write is not confirmed

    # Source: 069-proposal-impact-review-path — Scenario: A response is rejected
    Scenario: A rejected response fabricates no state
      Given a respond attempt the server did not allow
      When the path runs the response through the confirmed write flow
      Then it will surface the API failure by name
      And it will record nothing
      And it will not treat any non-2xx as success or fabricate a state the record does not contain

    # Source: 069-proposal-impact-review-path — Proposed: split write locus (plan ADR-3 / interface Error Communication) — the reviewer hands the respond step back to the caller
    Scenario: The reviewer hands the respond step back to the caller
      Given a caller who asked the proposal-impact-reviewer to record the response itself
      When the reviewer answers
      Then it will refuse and name the respond as the skill's caller-context step, taken after the operator decides
      And no respond will be run by the reviewer

    # Source: 069-proposal-impact-review-path — Scenario: The single gated response is routed through the guardrail
    @validation
    Scenario: The path routes its one write through the guardrail
      Given the path's treatment of the respond write
      When it is inspected against the Write-Safety Guardrail
      Then it will run through the confirmed write flow
      And the path will not record the response as if it were an ungated write
      And the path will perform no other governance write

    # Source: 069-proposal-impact-review-path — Scenario: Review-and-respond only, no circulation transitions
    @validation
    Scenario: The path performs no circulation or creation step
      Given the proposal-impact-review skill and agent content
      When it is inspected for any propose, withdraw, or create step
      Then it will contain none
      And advancing and withdrawing will remain the Proposal Circulation Path's acts
      And creation will remain the Proposal Drafting Path's act
