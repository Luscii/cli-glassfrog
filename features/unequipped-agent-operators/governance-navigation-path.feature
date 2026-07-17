# Source: 064-governance-navigation-path — Scenario: From a concern to the roles that touch it

Feature: Governance Navigation Path
  The AI agent driving the CLI has no packaged operating knowledge, so to work a
  tension it must hand-assemble a dozen reads and untangle the raw output itself.
  Governance Navigation Path is the first operator path on the Agent Operating
  Surface: a thin, discoverable "governance-navigation" skill (when to reach for
  it + the traversal workflow) that delegates to a read-only "governance-navigator"
  agent under `plugin/agents/`. The agent runs the traversal in its own isolated
  context — cross-model search on a free-form concern, then role reads, fillers,
  domains, and policies — and returns a synthesized picture (roles, who fills them,
  and the governing domains and policies) rather than raw dumps, with every element
  carrying the id that reads it again. It composes only reads the CLI already
  exposes, never writes (deferring capture to the Tension Processing Path), and
  only surfaces governing governance — the authority verdict is the Constraint
  Discovery Path's job. A best-effort drift guard in `internal/build` keeps the
  named read leaves truthful to the shipped CLI.
  (affects: AI agent, Practitioner)

  Rule: Work a tension through a guided path, not hand-assembled raw reads
    # In order to work a tension without hand-assembling a dozen reads and
    # untangling raw output,
    # as an AI agent,
    # I want to follow a guided path from the practitioner's concern to a
    # synthesized picture of the governance around it.

    # Source: 064-governance-navigation-path — Scenario: From a concern to the roles that touch it
    Scenario: Search a concern surfaces the relevant roles
      Given a practitioner had voiced a free-form concern with no role in hand
      When the governance-navigator traverses the concern
      Then it will search the governance record for what the concern touches
      And it will return the relevant roles and who fills them
      And each role and filler will carry the id needed to read it again

    # Source: 064-governance-navigation-path — Proposed: registration/discovery surface (interface Surface) — the agent is reachable once registered
    Scenario: The navigator is reachable once the plugin registers it
      Given the plugin was present with the governance-navigator agent registered
      When the governance-navigation skill delegates a concern for traversal
      Then the navigator will run the traversal in its own context
      And it will return only the synthesized picture to the caller

    # Source: 064-governance-navigation-path — Proposed: missing-agent degradation (interface Error Communication) — path degrades to guidance
    Scenario: A missing navigator degrades the path to guidance
      Given the plugin was present but the governance-navigator agent was absent or unregistered
      When the governance-navigation skill is consulted for a concern
      Then its workflow will remain readable as guidance the caller can follow by hand
      And no command in the CLI will be broken by the agent's absence

    # Source: 064-governance-navigation-path — Scenario: An over-broad concern matches many models
    Scenario: An over-broad concern is narrowed, not dumped
      Given a concern so broad that the search matched many roles, domains, and policies across several pages
      When the governance-navigator assembles the picture
      Then it will page through the full result set before choosing what is most relevant
      And it will present the most relevant results rather than every match
      And it will note that the picture was narrowed so the practitioner can refine

    # Source: 064-governance-navigation-path — Scenario: The concern matches nothing
    Scenario: An empty search reports nothing found without fabricating
      Given a concern for which the search returned no results
      When the governance-navigator completes the traversal
      Then it will report that nothing relevant was found
      And it will suggest refining the concern
      And it will fabricate no roles or governance

    # Source: 064-governance-navigation-path — Scenario: A read in the traversal fails
    Scenario: A failed read yields a partial picture
      Given a traversal in which one read failed while the others succeeded
      When the governance-navigator assembles the picture
      Then it will surface what the failure was
      And it will return the picture built from the reads that succeeded
      And it will not invent the missing piece

    # Source: 064-governance-navigation-path — Scenario: Synthesized, not raw
    @validation
    Scenario: The result is a synthesized picture, not raw output
      Given the picture the governance-navigator returned
      When it is compared against the raw command output
      Then it will be a drawn-together picture of roles, fillers, domains, and policies
      And it will not be a concatenation of unsynthesized dumps

  Rule: See the roles, fillers, and governing domains and policies drawn together
    # In order to understand who and what my concern touches before I decide what
    # to do,
    # as a practitioner (served through the agent),
    # I want to see the relevant roles, their fillers, and the domains and policies
    # that shape them, drawn together.

    # Source: 064-governance-navigation-path — Scenario: Drawing in the governing domains and policies
    Scenario: A relevant role's domains and policies are drawn in
      Given a role the governance-navigator had identified as relevant to the concern
      When it traverses that role's governance
      Then it will draw in the domains the role controls that bear on the concern
      And it will draw in the policies on the role's interior that bear on the concern
      And it will present them as part of one picture

    # Source: 064-governance-navigation-path — Scenario: A circle concern follows into its sub-roles
    Scenario: A circle concern follows into its sub-roles
      Given the concern touched a role that is a circle
      When the governance-navigator judges the sub-roles relevant
      Then it will follow into those sub-roles and their fillers as far as the concern warrants
      And it will stop short of walking the whole tree

    # Source: 064-governance-navigation-path — Scenario: The concern is really an authority question
    Scenario: An authority question surfaces governance but defers the verdict
      Given a concern phrased as whether the practitioner may take an action
      When the governance-navigator surfaces the domains and policies that govern it
      Then it will present that governing governance
      And it will defer the authority verdict to the Constraint Discovery Path
      And it will not rule on whether the action is permitted

    # Source: 064-governance-navigation-path — Scenario: Surfacing, not judging
    @validation
    Scenario: The path surfaces governance without judging authority
      Given the governance-navigator's treatment of domains and policies
      When it is inspected for an authority or permission verdict
      Then it will only surface the governing governance
      And it will nowhere rule on whether an action is allowed

  Rule: Move from understanding to action with every element carrying its id
    # In order to move from understanding to action without losing my place,
    # as an AI agent,
    # I want to have each element of the picture carry the id that reads it again
    # or feeds the next path.

    # Source: 064-governance-navigation-path — Proposed: synthesized-picture output contract (interface Surface) — every element carries its id
    Scenario: Every element of the picture carries its actionable id
      Given a picture the governance-navigator had returned for a concern
      When the caller inspects each role, filler, domain, and policy in it
      Then every element will carry the id needed to read it again
      And the caller will be able to act on any element without re-running the search

    # Source: 064-governance-navigation-path — Scenario: Read-only throughout
    @validation
    Scenario: The path only reads, never writes
      Given the governance-navigation skill and agent content
      When it is inspected for any write, confirm, or gate step
      Then it will contain none
      And the path will only read

    # Source: 064-governance-navigation-path — Scenario: No invented surface
    @validation
    Scenario: The path names no read the CLI lacks
      Given the produced navigation-path content
      When every command and read it composes is checked against the shipped CLI
      Then each one will exist
      And the path will name no read the CLI does not expose
