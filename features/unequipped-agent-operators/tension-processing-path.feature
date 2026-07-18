# Source: 066-tension-processing-path — Scenario: From a voiced tension to a captured record

Feature: Tension Processing Path
  The AI agent driving the CLI has no packaged operating knowledge, so to act on
  a tension it must hand-assemble the tension commands and untangle the raw output
  itself. Tension Processing Path is the second operator path on the Agent Operating
  Surface and the write-side counterpart to Governance Navigation: a thin,
  discoverable "tension-processing" skill (when to reach for it + the workflow) that
  delegates to a "tension-processor" agent under `plugin/agents/`. The agent runs
  the workflow in its own isolated context — situating the tension against what the
  role and its sub-roles already sense, capturing it on the sensing role, then
  refining or retiring it — and returns a drawn-together tension record rather than
  raw dumps, with every element carrying the id that reads it again or feeds the
  next path. It composes only the operational tension commands the CLI already
  exposes, whose writes 063 leaves ungated; it never crosses into a proposal write
  (that is the Proposal Drafting and Circulation Paths) and never judges authority
  (that is the Constraint Discovery Path). A best-effort drift guard in
  `internal/build` keeps the named tension leaves truthful to the shipped CLI and
  disjoint from 063's gated proposal-write set.
  (affects: AI agent, Practitioner)

  Rule: Turn a voiced tension into a well-formed record, not hand-assembled commands
    # In order to turn a gap I've noticed into a well-formed record without
    # hand-assembling the tension commands,
    # as an AI agent operating the CLI on a practitioner's behalf,
    # I want to follow a guided path from the voiced tension to a captured,
    # refined tension carrying its id.

    # Source: 066-tension-processing-path — Scenario: From a voiced tension to a captured record
    Scenario: A voiced tension is captured on the sensing role
      Given a practitioner had voiced a tension and the sensing role it belongs to
      When the tension-processor captures the tension against that role
      Then it will return the created tension including its ten_ id
      And the record will carry that id so the tension can be refined or fed onward

    # Source: 066-tension-processing-path — Scenario: Refining a captured tension
    Scenario: A captured tension is refined without recapturing
      Given a tension already on the record that needed a clearer body or better routing
      When the tension-processor refines it through the update command
      Then it will return the updated tension
      And it will not recapture the tension as a second entry

    # Source: 066-tension-processing-path — Scenario: The tension is no longer worth pursuing
    Scenario: A moot tension is retired rather than pushed to a proposal
      Given a tension the practitioner decided was moot
      When the tension-processor processes that decision
      Then it will retire the tension through the discard command
      And it will not push the tension toward a proposal

    # Source: 066-tension-processing-path — Scenario: A capture is rejected
    Scenario: A rejected capture fabricates no id
      Given a capture attempt with an unknown sensing role or a blank body
      When the tension-processor runs the capture command
      Then it will surface the usage or API failure by name
      And it will record nothing
      And it will fabricate no ten_ id the record does not contain

    # Source: 066-tension-processing-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    Scenario: The processor is reachable once the plugin registers it
      Given the plugin was present with the tension-processor agent registered
      When the tension-processing skill delegates a voiced tension for processing
      Then the processor will run the workflow in its own context
      And it will return only the tension record to the caller

    # Source: 066-tension-processing-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    Scenario: A missing processor degrades the path to guidance
      Given the plugin was present but the tension-processor agent was absent or unregistered
      When the tension-processing skill is consulted for a tension
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

    # Source: 066-tension-processing-path — Scenario: No invented surface
    @validation
    Scenario: The path names no command the CLI lacks
      Given the produced tension-processing-path content
      When every command it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no command the CLI does not expose

    # Source: 066-tension-processing-path — Scenario: Synthesized, not raw
    @validation
    Scenario: The result is a synthesized tension record, not raw output
      Given the record the tension-processor returned
      When it is compared against the raw command output
      Then it will be a drawn-together tension record carrying ids
      And it will not be a concatenation of unsynthesized dumps

  Rule: Show what's already sensed before capturing a duplicate
    # In order to avoid cluttering the record with a tension that is already sensed,
    # as a practitioner (served through the agent),
    # I want the path to show me what's already sensed on the role and its
    # sub-roles before it captures a new one.

    # Source: 066-tension-processing-path — Scenario: Situating a tension against what's already sensed
    Scenario: A tension is situated against existing and rolled-up tensions
      Given a sensing role whose already-sensed tensions and rolled-up sub-role tensions span several pages
      When the tension-processor situates the voiced tension against them
      Then it will page through the full result set before judging duplicates
      And it will present the tensions drawn together so the practitioner can see what is already on the record
      And it will treat the new capture as a deliberate addition rather than a blind duplicate

    # Source: 066-tension-processing-path — Scenario: The tension is already sensed
    Scenario: A duplicate surfaces the existing tension instead of recording a second
      Given a voiced tension that matched one already on the record
      When the tension-processor situates it before capturing
      Then it will surface the existing tension with its id
      And it will let the practitioner refine that one
      And it will not silently record a duplicate

    # Source: 066-tension-processing-path — Scenario: A situating read fails
    Scenario: A failed situating read yields a partial record
      Given a situating step in which the sub-role roll-up failed while the other reads succeeded
      When the tension-processor continues
      Then it will surface what the failure was
      And it will present the tensions the reads that succeeded returned
      And it will not invent the missing tensions or abandon the whole record

  Rule: Move a processed tension toward a governance change carrying its id
    # In order to move from a captured tension into a governance change without
    # losing my place,
    # as an AI agent assembling the write flow,
    # I want the processed tension to carry the ten_ id that feeds the Proposal
    # Drafting Path.

    # Source: 066-tension-processing-path — Scenario: The tension is ready to become a governance change
    Scenario: A ready tension is handed off without drafting a proposal
      Given a captured, well-formed tension the practitioner wanted to turn into a proposal
      When the tension-processor completes its processing
      Then it will hand the ten_ id to the Proposal Drafting Path
      And it will not draft, create, or circulate the proposal itself

    # Source: 066-tension-processing-path — Scenario: Operational writes only, no governance write
    @validation
    Scenario: The path performs only operational tension writes
      Given the tension-processing skill and agent content
      When it is inspected for any proposal create, advance, withdraw, or response step
      Then it will contain none
      And it will hand the proposal work to the Proposal Drafting Path

    # Source: 066-tension-processing-path — Scenario: Correct guardrail boundary
    @validation
    Scenario: The path stays on the ungated operational side of the guardrail
      Given the path's treatment of its writes
      When it is inspected against the Write-Safety Guardrail
      Then it will neither gate the operational tension edits behind the governance-write confirmation
      And it will perform no governance write that would require it

    # Source: 066-tension-processing-path — Scenario: Processing, not judging or coaching
    @validation
    Scenario: The path processes the tension without judging authority or coaching
      Given the tension-processing skill and agent content
      When it is inspected for an authority verdict or Holacracy coaching
      Then it will neither rule on whether a tension needs a proposal
      And it will not advise on governance craft
