# Source: 065-constraint-discovery-path — Scenario: A wanted action falls under another role's domain

Feature: Constraint Discovery Path
  The AI agent driving the CLI can read the governance record, but when a
  practitioner wants to *do* something it has no guided way to learn whether the
  action is theirs to take, falls under another role's domain, is shaped by a
  policy, or needs a proposal. Constraint Discovery Path is the second operator
  path on the Agent Operating Surface and the mirror of Governance Navigation:
  a thin, discoverable "constraint-discovery" skill (when to reach for it + the
  workflow, including a clarify step when the action is too vague) that delegates
  to a read-only "constraint-navigator" agent under `plugin/agents/`. The agent
  runs the traversal in its own isolated context — cross-model search on the
  wanted action, then the owning role's domains and policies, plus the caller's
  own roles — and returns a synthesized picture that *characterizes* the authority
  situation drawn from the record, with every element carrying the id that reads
  it again. It composes only reads the CLI already exposes, never writes, and
  never computes a permission verdict from local rules: when the record does not
  clearly answer, it says so rather than guessing. A best-effort drift guard in
  `internal/build` keeps the named read leaves truthful to the shipped CLI.
  (affects: AI agent, Practitioner)

  Rule: Follow a guided path from a wanted action to the governing domains and policies
    # In order to know whether an action I want to take is mine to take, needs
    # someone else's permission, or needs a proposal — without hand-assembling a
    # set of domain and policy reads,
    # as an AI agent,
    # I want to follow a guided path from a wanted action to the governing domains
    # and policies that constrain it.

    # Source: 065-constraint-discovery-path — Scenario: A wanted action falls under another role's domain
    @wip
    Scenario: A wanted action under another role's domain is surfaced with its owner
      Given a practitioner's free-form wanted action with no role or domain in hand
      When the constraint-navigator traverses the action
      Then it will search the governance record for what the action touches
      And it will surface the domain that governs the action and the role that holds it
      And it will characterize the action as falling under that role's authority, needing its permission or a proposal
      And each domain and role will carry the id needed to read it again

    # Source: 065-constraint-discovery-path — Proposed: own-role authority branch (interface Surface `owned_by_caller` + spec Characterization) — the own-vs-other determination drawn from `me roles`
    @wip
    Scenario: An action under the caller's own role's domain is within their authority
      Given a wanted action governed by a domain that a role the caller fills holds
      When the constraint-navigator reads the caller's own roles and the governing domain
      Then it will find the governing domain belongs to a role the caller fills
      And it will characterize the action as within the caller's own authority
      And it will not frame the action as needing another role's permission

    # Source: 065-constraint-discovery-path — Scenario: The wanted action is too vague to locate its governance
    @wip
    Scenario: A too-vague action is clarified by the skill before any traversal
      Given a wanted action described too vaguely to search for its governing governance
      When the constraint-discovery skill begins
      Then the skill will ask the operator to sharpen the action before delegating
      And it will not guess a meaning and traverse on the guess
      And the constraint-navigator will not be invoked until the action is well-formed

    # Source: 065-constraint-discovery-path — Scenario: An over-broad action matches many models
    @wip
    Scenario: An over-broad action is narrowed, not dumped
      Given an action so broad that the search matched many roles, domains, and policies across several pages
      When the constraint-navigator assembles the picture
      Then it will page through the full result set before choosing what is most relevant
      And it will present the most relevant governing constraints rather than every match
      And it will note that the picture was narrowed so the practitioner can refine

    # Source: 065-constraint-discovery-path — Scenario: A read in the discovery fails
    @wip
    Scenario: A failed read yields a partial picture
      Given a traversal in which one read failed while the others succeeded
      When the constraint-navigator assembles the picture
      Then it will surface what the failure was
      And it will return the picture built from the reads that succeeded
      And it will not invent the missing piece

    # Source: 065-constraint-discovery-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    @wip
    Scenario: The navigator is reachable once the plugin registers it
      Given the plugin was present with the constraint-navigator agent registered
      When the constraint-discovery skill delegates a well-formed action for traversal
      Then the navigator will run the traversal in its own context
      And it will return only the synthesized picture to the caller

    # Source: 065-constraint-discovery-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    @wip
    Scenario: A missing navigator degrades the path to guidance
      Given the plugin was present but the constraint-navigator agent was absent or unregistered
      When the constraint-discovery skill is consulted for an action
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

  Rule: See what governs the action, drawn from the record not the tool's opinion
    # In order to decide how to proceed before I act,
    # as a practitioner (served through the agent),
    # I want to see which domain governs my action, which policies shape it, and
    # whether it needs a proposal — drawn from the record, not from the tool's own
    # opinion.

    # Source: 065-constraint-discovery-path — Scenario: A wanted action is shaped by a policy
    @wip
    Scenario: A policy that shapes the action is surfaced as a constraint to observe
      Given an action the constraint-navigator had located in the governance record
      When it traverses the policies bearing on the action
      Then it will surface the policy that grants or limits the action
      And it will present it as the constraint to observe, drawn together with any governing domain
      And it will not present it as a concatenation of separate dumps

    # Source: 065-constraint-discovery-path — Scenario: A wanted action that nothing in the record constrains
    @wip
    Scenario: An unconstrained action surfaces the absence without asserting permission
      Given an action for which no domain in view governs it and no policy in view limits it
      When the constraint-navigator completes the traversal
      Then it will surface that the record shows nothing constraining the action
      And it will report that absence plainly
      And it will not assert that the operator is permitted

    # Source: 065-constraint-discovery-path — Scenario: The record does not clearly answer
    @wip
    Scenario: An ambiguous record is reported as unclear, not resolved by a guess
      Given an action for which the match is ambiguous and no domain plainly owns it
      When the constraint-navigator completes the traversal
      Then it will characterize the situation as one the record does not clearly answer
      And it will surface what it found
      And it will not fabricate an authority ruling to resolve the ambiguity

    # Source: 065-constraint-discovery-path — Scenario: Surfacing, not ruling
    @validation @wip
    Scenario: The path surfaces and characterizes without ruling from local logic
      Given the constraint-navigator's treatment of the wanted action
      When it is inspected for a permission verdict computed from local logic
      Then it will contain none
      And it will only surface the governing domains and policies drawn from the record
      And it will nowhere reimplement permission rules or rule on whether the action is allowed

    # Source: 065-constraint-discovery-path — Scenario: No fabricated ruling under uncertainty
    @validation @wip
    Scenario: Under uncertainty the path says so rather than fabricating a verdict
      Given the path's handling of an action the record does not clearly constrain
      When its result is inspected
      Then it will state what is unclear and surface what it found
      And it will nowhere assert a permitted or forbidden verdict it cannot ground in the record

    # Source: 065-constraint-discovery-path — Scenario: Synthesized, not raw
    @validation @wip
    Scenario: The result is a synthesized picture, not raw output
      Given the picture the constraint-navigator returned
      When it is compared against the raw command output
      Then it will be a drawn-together picture of the governing domains and policies
      And it will not be a concatenation of unsynthesized dumps

  Rule: Move from a surfaced constraint to action with every element carrying its id
    # In order to move from understanding a constraint to acting on it without
    # losing my place,
    # as an AI agent,
    # I want to have each surfaced domain, policy, and role carry the id that reads
    # it again or feeds the next path.

    # Source: 065-constraint-discovery-path — Proposed: synthesized-picture output contract (interface Surface) — every element carries its id
    @wip
    Scenario: Every element of the picture carries its actionable id
      Given a picture the constraint-navigator had returned for a wanted action
      When the caller inspects each domain, policy, and role in it
      Then every element will carry the id needed to read it again
      And the caller will be able to act on any element without re-running the search

    # Source: 065-constraint-discovery-path — Scenario: Read-only throughout
    @validation @wip
    Scenario: The path only reads, never writes
      Given the constraint-discovery skill and agent content
      When it is inspected for any write, confirm, or gate step
      Then it will contain none
      And the path will only read

    # Source: 065-constraint-discovery-path — Scenario: No invented surface
    @validation @wip
    Scenario: The path names no read the CLI lacks
      Given the produced constraint-discovery-path content
      When every command and read it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no read the CLI does not expose
