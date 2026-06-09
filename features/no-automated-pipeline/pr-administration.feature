# Source: 028-pr-administration — Scenario: a feature pull request is labelled from its title

Feature: No Automated Pipeline — PR Administration
  On every pull request — including those opened from forks — administrative
  labels are applied from two signals: the title/branch (the semver-bearing
  categories breaking/features/fixes) and the changed files (docs,
  infrastructure, dependencies, internal). Labelling is authoritative: it
  reconciles its managed labels to the pull request's current state and never
  touches labels outside the managed set. It is not a merge gate — a labelling
  failure never blocks a merge (PR Validation owns the gate). The labels feed
  Release Drafting, which reads them for the semver bump and release-note
  categories. Fork pull requests are labelled by reading metadata and changed
  files only, never by executing pull-request code.
  (affects: Maintainer)

  Rule: Label every pull request by what it changes and what it is
    # In order to triage a pull request and know what kind of change it is at a glance,
    # as a maintainer,
    # I want every pull request automatically labelled by what it changes and what it is.

    # Source: 028-pr-administration — Scenario: a feature pull request is labelled from its title
    @wip
    Scenario: A feature pull request is labelled from its title
      Given a pull request whose title declared a feature-type change
      When labelling runs
      Then the pull request will carry the "features" label
      And that label will be available for Release Drafting to read

    # Source: 028-pr-administration — Scenario: a docs-only change is labelled from its changed files
    @wip
    Scenario: A docs-only change is labelled from its changed files
      Given a pull request that changed only documentation files
      When labelling runs
      Then the pull request will carry the "docs" label
      And no semver-bearing label will be forced onto it

    # Source: 028-pr-administration — Scenario: a pull request carries multiple matching labels
    @wip
    Scenario: A pull request carries every matching label
      Given a pull request whose title declared a feature and which also changed a dependency manifest
      When labelling runs
      Then the pull request will carry both the "features" and "dependencies" labels
      And resolving them into a single semver bump will be left to Release Drafting

    # Source: 028-pr-administration — Scenario: an unrecognized change is not mislabelled
    @wip
    Scenario: An unrecognized change is not mislabelled
      Given a pull request whose title, branch, and changed files matched no managed signal
      When labelling runs
      Then no managed label will be applied
      And Release Drafting's default bump will apply at release time

  Rule: Keep labels accurate and current for release drafting
    # In order to produce a correct semver bump and well-categorized release notes without hand-labelling every merge,
    # as a maintainer running Release Drafting,
    # I want each pull request to already carry accurate, current category and version labels.

    # Source: 028-pr-administration — Scenario: editing the title reconciles the labels (sync)
    @wip
    Scenario: Editing the title reconciles the labels
      Given a pull request previously labelled "features" from a feature-type title
      When the title is edited to a fix-type change and labelling re-runs
      Then the "features" label will be removed
      And the "fixes" label will be applied

    # Source: 028-pr-administration — Scenario: a labelling failure does not block the merge
    @wip
    Scenario: A labelling failure does not block the merge
      Given an otherwise-mergeable pull request
      When the labelling run fails to complete
      Then the merge will not be blocked or reddened by PR Administration
      And the merge gate will remain solely PR Validation's to decide

    # Architecture-informed (proposed by skill — plan concurrency/cancel-in-progress, latest-state sync)
    @wip
    Scenario: Rapid successive title edits reconcile to the latest title
      Given a pull request whose title is edited several times in quick succession
      When labelling runs
      Then the applied labels will reflect the latest title
      And labels from a superseded edit will not remain

    # Source: 028-pr-administration — Scenario: reconciliation never touches labels outside the managed set
    @validation @wip
    Scenario: Reconciliation never touches labels outside the managed set
      Given a pull request a maintainer had hand-labelled with a triage label outside the managed set
      When labelling re-runs and reconciles its managed labels
      Then the hand-applied triage label will be left untouched
      And only labels within the managed set will be added or removed

    # Source: 028-pr-administration — Scenario: labelling is not a required check
    @validation @wip
    Scenario: Labelling is not a required check
      Given the repository's merge protection
      When it is inspected
      Then PR Administration's labelling will not be a required, merge-blocking status
      And only the verification gate will block merge

  Rule: Label fork contributions on the same terms
    # In order to have my contribution categorized correctly even though I work from a fork,
    # as an external contributor or AI agent submitting a change,
    # I want my pull request labelled on the same terms as an internal one.

    # Source: 028-pr-administration — Scenario: a fork pull request is labelled on the same terms
    @wip
    Scenario: A fork pull request is labelled on the same terms
      Given a pull request opened from a fork by an external contributor
      When labelling runs
      Then the pull request will be classified and labelled like an internal one
      And its code will never be checked out or executed to do so

    # Source: 028-pr-administration — Scenario: fork labelling reads metadata only
    @validation @wip
    Scenario: Fork labelling reads metadata only
      Given a labelling run on a fork-originated pull request
      When the run is inspected
      Then it will derive labels from the pull request's title, branch, and changed-file list
      And it will not check out or execute the pull request's head code
