# Source: 079-pre-assembly-grammar-consultation — Scenario: The gate runs in order before the write

Feature: Pre-Assembly Grammar Consultation
  Three artifacts exist for building a proposal's change set correctly — the
  grammar the CLI renders, the recorded dead shapes it carries, and the routing
  record — and nothing consults any of them: the drafting path still assembles
  a change set from prior knowledge, so both recorded dead shapes stay
  reachable by an assembler that has the answer sitting unread beside it. The
  gate wires the consultation into the drafting path ahead of its one gated
  write, informing and never withholding.

  Rule: Consult the grammar before building, and see a dead shape before the write
    # In order to assemble a change set the server will take on the first
    # attempt, instead of learning each shape from a refused round-trip inside
    # a gated write,
    # as an AI agent drafting a proposal,
    # I want the grammar consulted before I build, and a recorded dead shape
    # named before the write rather than after it.

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The gate runs in order before the write
    Scenario: The workflow orders the gate ahead of the gated create
      Given the proposal-drafting skill's workflow was read end to end
      When its steps are compared against the gate's order
      Then the routing determination will precede the grammar consult, the consult will precede assembly, and the match against recorded dead shapes will follow assembly with the routing answer in hand
      And every gate step will precede the confirmed create

    # Source: 079-pre-assembly-grammar-consultation — Scenario: A recognized dead shape is surfaced before the write
    Scenario: A recognized dead shape is named before the write
      Given an assembled change set that matched the recorded dead shape CSG-1's refused wrapper form
      When the drafter reaches the gated create
      Then it will surface the fact's handle, shape, and symptom before the write
      And it will return action surfaced-dead-shape awaiting the practitioner's direction
      And it will express no verdict on the change set's validity

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The grammar read fails
    Scenario: A failed grammar read is recorded and drafting continues
      Given a grammar read that failed
      When the drafter continues
      Then the consultation element will record that the grammar was not consulted, naming the failure
      And drafting will continue rather than being withheld
      And assembly will not be presented as consulted

    # Source: 079-pre-assembly-grammar-consultation — Scenario: An anchor-dependent dead shape is recognized
    Scenario: The self-targeting shape is recognized with the routing answer in hand
      Given a change set whose role operation targeted the circle the proposal would be anchored in
      When the drafter matches the assembled set against the recorded dead shapes
      Then it will recognize the self-targeting shape recorded as CSG-2 and name its symptom before the write
      And the recognition will rest on both the change's target and the circle the proposal would be anchored in

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The practitioner proceeds past a surfaced dead shape
    Scenario: Proceeding past a surfaced dead shape runs the create unchanged
      Given a recognized dead shape surfaced with action surfaced-dead-shape
      When the practitioner directs the drafter to proceed past that fact
      Then the re-delegated run will act on the direction and run the create through the confirmed write flow unchanged
      And the change set will not be altered

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The change set matches nothing recorded
    Scenario: A change set matching nothing recorded implies nothing about validity
      Given an assembled change set matching no recorded dead shape
      When the drafter reaches the write
      Then the consultation element will state that no recorded shape matched
      And nothing about the set's validity will be implied

    # Source: 079-pre-assembly-grammar-consultation — Proposed: interface-spec.md defensive contract — direction present means act
    Scenario: A re-delegation carrying direction does not re-surface the same decision
      Given a re-delegation whose input carried the settled anchor and the proceed-past instruction naming the surfaced fact
      When the drafter runs the gate from the top
      Then it will act on the direction rather than returning the same decision again

    # Source: 079-pre-assembly-grammar-consultation — Proposed: plan ADR-3 — the consultation read joins the composed surface ungated
    Scenario: The consultation read joins the composed surface as an ungated read
      Given the composed-leaf registry and the write gate's registries
      When the eight composed leaves are checked against the gate's membership
      Then proposal grammar will be listed as a composed read and absent from the gated set
      And proposal create will remain the only composed leaf in the gated set

    # Source: 079-pre-assembly-grammar-consultation — Scenario: Consultation is unconditional and ordered
    @validation @wip
    Scenario: No drafting path reaches assembly unconsulted or a settled anchor unrouted
      Given the wired drafting workflow
      When every path from intent to a created draft is inspected
      Then none will reach assembly without consulting the grammar, none will reach a settled anchor without the routing determination, and none will match the assembled set before routing has answered

    # Source: 079-pre-assembly-grammar-consultation — Scenario: No content was copied into the wiring
    @validation @wip
    Scenario: The wiring names its sources without restating them
      Given the wired workflow's text
      When it is compared against the grammar the CLI renders and the routing record
      Then it will name and invoke them without restating a change type, a placement rule, a recorded shape, or the routing rule

  Rule: Trust that the shipped records are actually applied
    # In order to trust that the shipped grammar and routing records are
    # actually being applied,
    # as a practitioner reviewing what the agent did,
    # I want the returned draft record to say what was consulted and what it
    # surfaced.

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The consultation is legible in the result
    @validation @wip
    Scenario: A reader can tell from the record what was consulted and surfaced
      Given a completed drafting run's returned record
      When someone who did not watch the run reads it
      Then they will be able to tell that the grammar was read, what the routing determination answered, and whether a dead shape was surfaced

    # Source: 079-pre-assembly-grammar-consultation — Scenario: The inert-reads note no longer misdescribes the path
    @validation @wip
    Scenario: The registry no longer claims the routing reads are ahead of their use
      Given the drafting path's composed-leaf registry and agent fence after the gate landed
      When their annotations are read
      Then no sentence will claim the routing reads imply no routing step
      And the surface will describe a path that routes
