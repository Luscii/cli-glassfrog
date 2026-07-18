# Source: 067-proposal-drafting-path — Scenario: From a tension to a draft proposal

Feature: Proposal Drafting Path
  The AI agent driving the CLI has no packaged operating knowledge, so to turn a
  ready tension into a governance change it must hand-assemble the proposal-create
  command — and the one write that most needs care, the gated governance write —
  without guidance. Proposal Drafting Path is the fourth operator path on the
  Agent Operating Surface and the first to cross the Write-Safety Guardrail: a
  thin, discoverable "proposal-drafting" skill (when to reach for it + the
  workflow) that delegates to a "proposal-drafter" agent under `plugin/agents/`.
  The agent runs the workflow in its own isolated context — grounding the draft in
  the anchor tension, situating against the proposals already in flight in the
  circle, assembling the change set — and then creates the draft through the
  guardrail-confirmed write, passing the change set inline so the confirmation
  shows the exact payload. It returns a drawn-together draft record carrying the
  prp_ id that feeds the Proposal Circulation Path. It composes only commands the
  CLI already exposes; its single write is the gated create — never advance,
  respond, or withdraw (circulation), never a tension write (processing), never an
  authority verdict (constraint discovery). A best-effort drift guard in
  `internal/build` keeps the named leaves truthful to the shipped CLI and pins the
  create's membership in the guardrail's gated set.
  (affects: AI agent, Practitioner)

  Rule: Turn a ready tension into a submittable draft, not a hand-assembled create
    # In order to turn a well-formed tension into a submittable draft proposal
    # without hand-assembling the create command,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to follow a guided path from the anchor tension to a created draft
    # proposal carrying its prp_ id.

    # Source: 067-proposal-drafting-path — Scenario: From a tension to a draft proposal
    
    Scenario: A ready tension becomes a created draft proposal
      Given a well-formed anchor tension's ten_ id and an assembled set of governance changes
      When the proposal-drafter surfaces them for confirmation and runs the create through the confirmed write flow
      Then it will return the created proposal including its prp_ id and draft status
      And the record will carry that id so the draft can be advanced or withdrawn

    # Source: 067-proposal-drafting-path — Scenario: Sourcing the change set from a file
    
    Scenario: A file-held change set is passed through verbatim
      Given a set of governance changes held in a JSON file
      When the proposal-drafter assembles the change set from that source
      Then it will pass the array through verbatim above the type floor as the proposal's changes
      And it will not interpret or validate any change's command-specific keys

    # Source: 067-proposal-drafting-path — Scenario: A create is rejected
    
    Scenario: A rejected create fabricates no id
      Given a create attempt whose anchor tension was unknown to the record
      When the proposal-drafter runs the create through the confirmed write flow
      Then it will surface the API failure by name
      And it will create nothing
      And it will fabricate no prp_ id the record does not contain

    # Source: 067-proposal-drafting-path — Scenario: The create must be confirmed before it crosses the boundary
    
    Scenario: An unconfirmed create leaves the record untouched
      Given an assembled anchor and change set ready to create
      When the proposal-drafter reaches the proposal create
      Then it will route the write through the Write-Safety Guardrail's confirmed flow, surfacing the change set first
      And no proposal will be created when the write is not confirmed

    # Source: 067-proposal-drafting-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    
    Scenario: The drafter is reachable once the plugin registers it
      Given the plugin was present with the proposal-drafter agent registered
      When the proposal-drafting skill delegates a ready tension for drafting
      Then the drafter will run the workflow in its own context
      And it will return only the draft record to the caller

    # Source: 067-proposal-drafting-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    
    Scenario: A missing drafter degrades the path to guidance
      Given the plugin was present but the proposal-drafter agent was absent or unregistered
      When the proposal-drafting skill is consulted for a ready tension
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

    # Source: 067-proposal-drafting-path — Scenario: No invented surface
    @validation
    Scenario: The path names no command the CLI lacks
      Given the produced proposal-drafting-path content
      When every command it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no command the CLI does not expose

    # Source: 067-proposal-drafting-path — Scenario: Assembly, not typed construction
    @validation
    Scenario: The path assembles the change set without typed construction
      Given the proposal-drafting skill and agent content
      When it is inspected for per-change interpretation
      Then it will assemble and pass the array through verbatim above a type floor
      And it will validate no change's type value or command-specific keys
      And it will build no typed constructor

    # Source: 067-proposal-drafting-path — Scenario: Synthesized, not raw
    @validation
    Scenario: The result is a synthesized draft record, not raw output
      Given the record the proposal-drafter returned
      When it is compared against the raw command output
      Then it will be a drawn-together draft carrying its prp_ id
      And it will not be a concatenation of unsynthesized dumps

  Rule: Show what's already in flight before opening a duplicate draft
    # In order to avoid opening a second draft for a change already in flight,
    # as a practitioner (served through the agent),
    # I want the path to show me the proposals already circulating in the circle
    # before it creates a new one.

    # Source: 067-proposal-drafting-path — Scenario: Situating against proposals already in flight
    
    Scenario: A draft is situated against the proposals already in flight
      Given a circle whose in-flight proposals span several pages
      When the proposal-drafter surfaces the proposals already in flight there
      Then it will page through the full result set before judging duplicates
      And it will present them drawn together so the practitioner can see what is already circulating
      And it will treat the new draft as a deliberate addition rather than a blind duplicate

    # Source: 067-proposal-drafting-path — Scenario: A matching draft is already in flight
    
    Scenario: A matching in-flight draft is surfaced instead of duplicated
      Given a change that matched a draft already circulating in the circle
      When the proposal-drafter situates before creating
      Then it will surface the existing proposal with its prp_ id
      And it will let the practitioner decide
      And it will not silently create a duplicate draft

    # Source: 067-proposal-drafting-path — Scenario: A situating read fails
    
    Scenario: A failed situating walk yields a partial picture
      Given a situating step in which the proposal list read failed mid-walk
      When the proposal-drafter continues
      Then it will surface what the failure was
      And it will present the proposals the read gathered so far, flagged incomplete
      And it will not invent the missing proposals or abandon the whole result

  Rule: Move a created draft toward circulation carrying its id
    # In order to move a draft into circulation without losing my place,
    # as an AI agent assembling the write flow,
    # I want the created draft to carry the prp_ id that feeds the Proposal
    # Circulation Path.

    # Source: 067-proposal-drafting-path — Scenario: The draft is ready to circulate
    
    Scenario: A created draft is handed off without being advanced
      Given a created draft proposal the practitioner wanted to advance
      When the proposal-drafter completes its drafting
      Then it will hand the prp_ id to the Proposal Circulation Path
      And it will not advance, withdraw, or circulate the proposal itself

    # Source: 067-proposal-drafting-path — Scenario: The gated create is routed through the guardrail
    @validation
    Scenario: The path routes its one write through the guardrail
      Given the path's treatment of the proposal create
      When it is inspected against the Write-Safety Guardrail
      Then the create will run through the confirmed write flow
      And the path will not issue the gated proposal write as if it were ungated

    # Source: 067-proposal-drafting-path — Scenario: Drafting only, no further transition
    @validation
    Scenario: The path stops at the created draft
      Given the proposal-drafting skill and agent content
      When it is inspected for any advance, withdraw, response, or circulate step
      Then it will contain none
      And it will hand the prp_ id to the Proposal Circulation Path

    # Source: 067-proposal-drafting-path — Scenario: Drafting, not judging or coaching
    @validation
    Scenario: The path drafts without judging authority or coaching
      Given the proposal-drafting skill and agent content
      When it is inspected for an authority verdict or Holacracy coaching
      Then it will neither rule on whether the tension needs a proposal
      And it will not advise on governance craft
