# Source: 071-drafter-config-migration — Scenario: a drafting run reports no schema deprecations

Feature: No Automated Pipeline — Drafter Config Contract
  The release-drafting configuration, the in-repo guard that parses it, and the
  drafting workflow's pinned action version must all agree. The configuration is
  expressed in the schema the pinned action actually reads, with the seven
  category labels, the exclusion label, the three semver buckets and the patch
  fallback each declared at their current positions. The guard asserts the
  three-file label contract from those positions and additionally holds the
  pinned action major at or above the floor the schema requires. This last
  verdict is the point: an action major that predates the configuration's schema
  does not reject it — it accepts it and silently ignores what it does not
  recognise, so the drafting run reports success while producing miscategorised
  notes and a wrong bump. Nothing at runtime reveals that, and the drafting
  workflow blocks nothing, so the guard is the only place it can be caught.
  (affects: Maintainer)

  Rule: Express the configuration in the schema the running action reads
    # In order to stop every drafting run reporting that the release configuration is written against a superseded schema,
    # as a maintainer,
    # I want the configuration expressed in the schema the running action actually reads.

    # Source: 071-drafter-config-migration — Scenario: a drafting run reports no schema deprecations
    @wip
    Scenario: A drafting run reports no schema deprecations
      Given the drafter configuration was realigned to the current schema
      When release drafting runs
      Then the run will complete without emitting a configuration-schema deprecation warning

    # Architecture-informed (proposed by skill — plan ADR-4, reject the superseded shape by name)
    Scenario: A configuration left on the superseded schema is rejected by name
      Given a drafter configuration still carrying top-level "version-resolver" and "exclude-labels" keys
      When the label-contract guard runs
      Then the guard will fail
      And the violation will name the superseded schema and point at the migration
      And it will not report the seven category labels as merely missing

  Rule: Move labels between configuration positions and nothing else
    # In order to keep trusting the draft release without re-reading it,
    # as a maintainer,
    # I want the realignment to move labels between configuration positions and nothing else, with the three-file label contract still asserted in CI as the standing check.

    # Source: 071-drafter-config-migration — Scenario: a feature merge still bumps minor and files under Features
    @wip
    Scenario: A feature merge still bumps the draft to the next minor
      Given the last published release was "v1.2.0"
      And the drafter configuration was realigned to the current schema
      And a pull request labelled "features" merged to main
      When release drafting runs
      Then the draft's proposed version will be "v1.3.0"
      And the pull request's title will appear under the "Features" category

    # Source: 071-drafter-config-migration — Scenario: the declared fallback supplies the patch bump
    @wip
    Scenario: The declared fallback supplies the patch bump
      Given the last published release was "v1.2.0"
      And the drafter configuration was realigned to the current schema
      And no included merged pull request carried a semver-bearing label
      When release drafting runs
      Then the draft's proposed version will be "v1.2.1"
      And the bump will come from the fallback declared in the configuration

    # Source: 071-drafter-config-migration — Scenario: exclusion survives the realignment
    @wip
    Scenario: The exclusion survives the realignment
      Given the drafter configuration was realigned to the current schema
      And a merged pull request carried the "no-release-note" label
      When release drafting runs
      Then the pull request's title will appear nowhere in the draft notes
      And it will not contribute to the version bump

    # Source: 071-drafter-config-migration — Scenario: a category losing its label predicate fails the guard
    Scenario: A category losing its label predicate fails the guard
      Given a changelog category in the drafter configuration named no label in its condition
      When the label-contract guard runs
      Then the guard will fail
      And the violation will name the configuration file and the label missing from the contract

    # Source: 071-drafter-config-migration — Scenario: removing the declared fallback fails the guard
    Scenario: Removing the declared fallback fails the guard
      Given the condition-less version-resolver category was deleted from the drafter configuration
      When the label-contract guard runs
      Then the guard will fail
      And the violation will name the expected fallback increment "patch" as an absent declaration
      And the guard will not pass on the grounds that the action would have fallen back to patch anyway

    # Source: 071-drafter-config-migration — Scenario: no label is invented or dropped by the realignment
    @validation @wip
    Scenario: No label is invented or dropped by the realignment
      Given every label named in the realigned drafter configuration
      When they are compared against PR Administration's managed label set
      Then the two sets will match exactly
      And the realignment will have introduced no label of its own

    # Source: 071-drafter-config-migration — Scenario: the pre-existing assertions survive in number and strictness
    @validation @wip
    Scenario: The four label-contract assertions survive in number and strictness
      Given the guard's four label-contract verdicts before the realignment
      When they are compared against the guard after the realignment
      Then all four will remain
      And each will still fail on a missing value as loudly as on an extra one
      And the coupling verdict will be additional to them rather than a replacement for any

    # Source: 071-drafter-config-migration — Scenario: the artifact makes no claim about the untagged-release failure
    @validation @wip
    Scenario: The change claims no fix for the untagged-release failure
      Given the specification and the pull-request description for the realignment
      When they are read end to end
      Then neither will assert that the change fixes the placeholder-tag publish failure
      And neither will imply it

  Rule: Fail CI when the pinned action version and the configuration schema disagree
    # In order to never again discover a schema mismatch only by reading a wrong release draft,
    # as a maintainer,
    # I want CI to fail when the pinned action version and the configuration's schema disagree.

    # Source: 071-drafter-config-migration — Scenario: the pinned action version falling behind the config schema fails the guard
    Scenario: A pinned action version behind the configuration schema fails the guard
      Given the drafter configuration was written in the current schema
      And the drafting workflow pinned the drafter action at "v6.4.0"
      When the coupling guard runs
      Then the guard will fail
      And the violation will name the pinned version and the schema floor the configuration requires
      And the mismatch will be caught before merge rather than by a drafting run

    # Architecture-informed (proposed by skill — plan ADR-5, an underivable ref is a finding not a pass)
    Scenario: A pinned reference with no derivable major fails rather than passes
      Given the drafting workflow pinned the drafter action at a commit SHA
      When the coupling guard runs
      Then the guard will fail
      And the violation will state that the pinned major could not be determined
      And it will name the workflow file and the reference it read

    # Source: 071-drafter-config-migration — Scenario: neither side of the coupling verdict is a hard-coded literal
    @validation @wip
    Scenario: Neither side of the coupling verdict is a hard-coded literal
      Given the coupling guard's two inputs
      When they are traced to their sources
      Then the pinned major will have been read from the drafting workflow
      And the schema generation will have been derived from the configuration's own shape
      And neither will be a fixed value written into the guard
