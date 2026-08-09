# Source: 076-operating-surface-self-containment — Scenario: A conforming surface passes verification

Feature: Operating-Surface Self-Containment
  The agent operating surface ships to machines that have the plugin and the
  glassfrog CLI — and nothing else — yet 21 of its 26 files reach into the
  development repository: spec numbers used as cross-reference ids, guard-test
  paths in the single-source registry headers, and plan-ADR citations. To an
  operator without the repository every one is a dangling pointer. Self-
  containment is the standing rule plus its enforcement: every reference in
  the surface resolves within the surface or to the CLI it drives, the ban is
  strict (the development repository is not acknowledged in any form), a
  walk-derived internal-build guard fails the merge gate on any violation
  present or future, and Constitution Principle XIII records the rule for
  future authoring.
  (affects: AI agent operator, developer)

  Rule: Follow the surface's handoffs with only the plugin and the CLI
    # In order to follow every handoff and boundary in the surface on a machine
    # that has only the plugin and the CLI,
    # as an AI agent operating the glassfrog CLI,
    # I want to find each cross-reference resolvable in place, with no
    # repository or spec catalog needed to decode it.

    # Source: 076-operating-surface-self-containment — Scenario: A handoff reads by name where the operator stands
    Scenario: Handoffs name in-plugin components
      Given the swept operating surface was read on a machine with only the plugin and the CLI
      When the proposal-drafting skill's authority-question deferral is read
      Then it will name the constraint-discovery path as the receiving component
      And no spec number will be needed to follow the handoff

    # Source: 076-operating-surface-self-containment — Scenario: Non-reference tokens do not false-positive
    Scenario: Known-safe tokens do not trip the check
      Given a surface file contained the example id "prp_0123", the version string "0.34.1", and the phrase "the v5 spec"
      When the self-containment check runs over that surface
      Then it will report zero violations
      And the operating-world references will remain intact

    # Source: 076-operating-surface-self-containment — Scenario: The sweep preserved every handoff
    @validation @wip
    Scenario: The sweep preserved every handoff
      Given the reworded surface and the pre-sweep artifacts
      When each replaced spec-number reference is compared with its replacement
      Then each replacement will name the same in-plugin component the number pointed at
      And no deferral, boundary, or handoff will have been dropped

    # Source: 076-operating-surface-self-containment — Scenario: The surface reads lightweight and workflow-oriented
    @validation @wip
    Scenario: The surface reads lightweight and workflow-oriented
      Given the swept surface read end to end
      When its files are inspected for development residue
      Then no file will explain repository mechanics, design history, or enforcement machinery
      And registry headers will state their editing rules through the surface's own consequences

  Rule: Development residue cannot merge into the shipped surface
    # In order to keep future path features from leaking development residue
    # into the shipped surface,
    # as the developer,
    # I want the merge-gating verification run to fail on any surface file that
    # references the development repository, including files that do not exist yet.

    # Source: 076-operating-surface-self-containment — Scenario: A conforming surface passes verification
    Scenario: A conforming surface passes verification
      Given every file under the operating surface referenced only in-surface components and the glassfrog CLI
      When the merge-gating verification run executes
      Then the self-containment check will pass
      And it will report zero violations

    # Source: 076-operating-surface-self-containment — Scenario: A future file is covered without registration
    Scenario: A future surface file is checked without registration
      Given a new file had been added under the operating surface
      When the merge-gating verification run executes
      Then the new file will be among the files checked
      And no list or configuration will have been updated to include it

    # Source: 076-operating-surface-self-containment — Scenario: A spec-number reference turns the run red
    Scenario: A spec-number reference fails the run
      Given a surface file contained the reference "(067)"
      When the self-containment check runs over that surface
      Then it will fail naming the file and the line
      And the report will carry the matched text "067" as a resolvable-reference violation
      And the report will state the remedy: replace with the in-plugin component name, or remove the reference

    # Source: 076-operating-surface-self-containment — Scenario: A repository mention fails even without a path
    Scenario: A pathless repository mention fails the run
      Given a surface file contained the phrase "a drift guard in the source repository"
      When the self-containment check runs over that surface
      Then it will fail on that line
      And the report will name the repo-machinery phrase family as the violated rule

    # Source: 076-operating-surface-self-containment — Scenario: An empty surface is a failure, not a pass
    Scenario: An empty surface fails rather than passes
      Given an operating surface whose walk found zero files
      When the self-containment check runs
      Then it will fail reporting the surface as missing or empty
      And it will not report success over a vacuously clean set

    # Source: 076-operating-surface-self-containment — Proposed: in-surface path resolution check from the interface accord (plan ADR-2)
    Scenario: A dangling in-surface path fails the run
      Given a surface file referenced "plugin/hooks/does-not-exist.txt"
      When the self-containment check runs over that surface
      Then it will fail reporting the dangling path
      And the report will state the remedy: correct the path to the existing in-surface file, or remove the reference

    # Source: 076-operating-surface-self-containment — Scenario: Re-derived assertions kept the property
    @validation @wip
    Scenario: Re-derived assertions kept the property
      Given the repo-side expectations that previously asserted spec numbers in surface text
      When the re-derived expectations are read
      Then each will assert the in-plugin name of the same component
      And none will have been deleted or weakened to an assertion that any text at all is present

    # Source: 076-operating-surface-self-containment — Scenario: Pointer direction is intact
    @validation @wip
    Scenario: Pointer direction is intact
      Given the repository artifacts that reference surface files
      When their references are checked after the sweep
      Then repository-to-surface references will still resolve
      And no surface file will reference the repository in any form

  Rule: Author future surface artifacts against a recorded rule
    # In order to author new surface artifacts against the rule instead of
    # rediscovering it in review,
    # as the developer running the specification pipeline,
    # I want the principle recorded in CONSTITUTION.md with its detection mechanism.

    # Source: 076-operating-surface-self-containment — Scenario: The constitutional principle is in house form
    @validation @wip
    Scenario: The constitutional principle is in house form
      Given CONSTITUTION.md after the amendment
      When the new principle is read
      Then it will carry a statement, a rationale, and a detection section like its siblings
      And the amendment will carry the documented justification and version bump the governance section requires
