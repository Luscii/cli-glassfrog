# Source: 077-agent-facing-grammar-reference — Scenario: An assembler reads the grammar before building

Feature: Agent-Facing Grammar Reference
  Building a proposal means assembling a changes[] array, and the knowledge of
  what is sayable in it is split three ways: the published contract carries the
  type vocabulary and the nested-only rule, the residual shapes live in the
  change-set grammar record, and per-type field guidance exists nowhere. None
  of it is served at the point of assembly. The Agent-Facing Grammar Reference
  closes that gap with a dedicated informational read command on the CLI —
  `glassfrog proposal grammar` — rendering every contract-enumerated change
  type with its placement class plus the empirical residue with its symptoms,
  every shape marked by provenance. The command informs and never validates:
  no API request, no credential, no verdict.

  This file covers the command and its rendering. Its siblings
  `change-set-grammar-facts.feature` (what the record says) and
  `change-set-grammar-guard.feature` (what the record's guard enforces) cover
  the upstream record this command renders.
  (affects: Practitioner, AI agent)

  Rule: Assemble a valid change set without prior knowledge or refused round-trips
    # In order to assemble a valid changes[] without prior knowledge of each
    # command's shape or refused round-trips against the server,
    # as an AI agent drafting a proposal,
    # I want to run one read that gives me the sayable change types, where each
    # may appear, and the known dead shapes with their symptoms — before I build.

    # Source: 077-agent-facing-grammar-reference — Scenario: An assembler reads the grammar before building
    @wip
    Scenario: An assembler reads the full grammar in one invocation
      Given an AI agent was about to assemble a change set
      When it runs "glassfrog proposal grammar --output json"
      Then the "change_types" array will carry every change type the vendored contract enumerates, each with its placement
      And the "facts" array will carry "CSG-1" and "CSG-2" with the symptom each produces
      And no API request will have been made to learn either

    # Source: 077-agent-facing-grammar-reference — Scenario: A change set offered for judgment is refused as usage, not judged
    @wip
    Scenario: A change set offered for checking is refused as usage, not judged
      Given the implemented "glassfrog proposal grammar" command
      When a practitioner runs "glassfrog proposal grammar changes.json"
      Then the run will fail as a usage error with exit code 2
      And the failure will express no verdict on the change set's validity

    # Source: 077-agent-facing-grammar-reference — Scenario: "Accepted" is not "valid" survives the rendering
    @wip
    Scenario: Accepted-but-invalid stays distinct from accepted in the rendering
      Given a successful "glassfrog proposal grammar --output json" run
      When a consumer reads the rendered fact "CSG-2"
      Then its disposition will read "accepted-but-invalid" rather than "accepted"
      And its symptom will state that a returned prp_ id is a dead draft, not a successful change

    # Source: 077-agent-facing-grammar-reference — Proposed: interface-cli.md § The human formats — full content with residue present
    @wip
    Scenario: The full format presents the vocabulary, the nesting rule, and every fact
      Given the record carried the facts "CSG-1" and "CSG-2"
      When a practitioner runs "glassfrog proposal grammar --output full"
      Then every change type will appear with its placement class
      And the nesting rule will be stated once with its wrapper types
      And each fact will carry its title, shape, disposition, and symptom
      And the contract-derived content will be visibly separated from the empirical observations

    # Source: 077-agent-facing-grammar-reference — Proposed: interface-cli.md § The human formats — compact form
    @wip
    Scenario: The compact format condenses each type and fact to one line
      Given the record carried the facts "CSG-1" and "CSG-2"
      When a practitioner runs "glassfrog proposal grammar --output compact"
      Then every change type will appear with its placement class in condensed form
      And each fact will appear as one line carrying its id, disposition, and title

    # Source: 077-agent-facing-grammar-reference — Scenario: No live residue
    @wip
    Scenario: The contract vocabulary renders even with no live residue
      Given the grammar record carried no live facts
      When the reference is rendered in the default human format
      Then the contract-derived change-type vocabulary will still render in full
      And the output will state that no empirical residue is currently recorded

    # Source: 077-agent-facing-grammar-reference — Scenario: No judgment path exists
    @validation @wip
    Scenario: No invocation judges a change set
      Given the landed "glassfrog proposal grammar" command
      When its invocation surface is inspected for what it can be made to do
      Then no invocation will evaluate, filter, or score a change set
      And its only effect will be rendering knowledge

  Rule: Know how far to trust each shape when the server changes
    # In order to know how far to trust each shape when the server changes
    # underneath it,
    # as an AI agent,
    # I want every rendered shape to carry its provenance, so a contract-published
    # shape and a verified observation are never confused.

    # Source: 077-agent-facing-grammar-reference — Scenario: Provenance is visible on every shape
    @wip
    Scenario: Every rendered shape carries its provenance token
      Given a successful "glassfrog proposal grammar --output json" run
      When a consumer reads the rendered structure
      Then every "change_types" entry will carry the provenance token "published-contract"
      And every "facts" entry will carry the provenance token "empirical-observation"

    # Source: 077-agent-facing-grammar-reference — Scenario: The contract vocabulary drifts from the rendering
    @wip
    Scenario: A contract refresh that outruns the rendering fails the build
      Given a vendored-contract refresh changed the change-type enum
      And the committed grammar artifact was not regenerated
      When the repository's merge-gating verification runs
      Then it will fail naming the divergence between the contract and the rendered vocabulary
      And the failure will name regeneration as the remedy

    # Source: 077-agent-facing-grammar-reference — Scenario: A recorded fact retires
    @wip
    Scenario: A retired fact leaves the rendering with the record
      Given the fact "CSG-1" retired from the record together with its manifest entry
      And the grammar artifact was regenerated
      When the reference next renders
      Then "CSG-1" will no longer appear among the rendered facts
      And the "CreatePolicy" type will remain in the contract-derived vocabulary

    # Source: 077-agent-facing-grammar-reference — Scenario: The rendered vocabulary equals the contract's
    @validation @wip
    Scenario: The rendered vocabulary matches the contract exactly
      Given the landed command and the vendored contract
      When the rendered change types and nested-only membership are set-compared against the contract's enum and rule
      Then the sets will match exactly with no missing, extra, or renamed type on either side

    # Source: 077-agent-facing-grammar-reference — Scenario: One source for the residue
    @validation @wip
    Scenario: Each rendered fact renders from the record alone
      Given the landed command and the grammar-facts record
      When the origin of each rendered empirical fact is traced
      Then each will render from the record through the generated artifact
      And no fact's text will be hand-maintained outside the record

  Rule: Consult the grammar on any machine that has the CLI
    # In order to consult the grammar on any machine that has the CLI — before
    # credentials are set up, with or without the operating surface installed,
    # as a practitioner (or the agent acting for one),
    # I want the reference served by the binary itself rather than by a file
    # only one install layout carries.

    # Source: 077-agent-facing-grammar-reference — Scenario: The grammar renders without credentials
    @wip
    Scenario: The grammar renders with no credentials and no network
      Given a machine with the CLI installed and no credential configured
      When a practitioner runs "glassfrog proposal grammar"
      Then the full reference will render successfully
      And no API request will be attempted

    # Source: 077-agent-facing-grammar-reference — Proposed: plan Integration Design — the write gate's recognized-read set gains "grammar"
    @wip
    Scenario: The write gate passes the grammar read ungated
      Given the operating surface's write gate was installed
      When an agent runs "glassfrog proposal grammar"
      Then the gate will pass the command ungated as a recognized proposal read
      And no confirmation will be asked

    # Source: 077-agent-facing-grammar-reference — Proposed: interface determinism contract — output ordering is pinned
    @wip
    Scenario: Repeated invocations render identical structured output
      Given a successful "glassfrog proposal grammar --output json" run
      When the same binary runs the same command again
      Then the structured output will be identical across the two runs
