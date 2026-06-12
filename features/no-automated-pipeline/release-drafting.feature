# Source: 030-release-drafting — Scenario: a feature merge bumps the draft to the next minor and files a note

Feature: No Automated Pipeline — Release Drafting
  On every merge to main, a single draft GitHub Release is kept current with
  everything merged since the last published release: the next semantic version
  (resolved highest-wins from the PR labels PR Administration applied, defaulting
  to patch) and the merged-PR titles grouped into the seven release-note
  categories. Pull requests carrying the exclusion label — and those whose
  changes are confined to specification and feature files — are left out of both
  the notes and the bump. The draft's pre-release/latest status follows the
  version (pre-release while 0.x, latest at 1.0.0+). Drafting never publishes:
  the draft waits for a maintainer to publish it, and that act triggers the
  Automated Release Pipeline. A drafting failure blocks nothing.
  (affects: Maintainer)

  Rule: Maintain an always-current draft with the computed version and categorized notes
    # In order to ship without hand-assembling release notes or guessing the next version,
    # as a maintainer,
    # I want an always-current draft release that already carries the computed version and categorized notes for everything merged since the last release.

    # Source: 030-release-drafting — Scenario: a feature merge bumps the draft to the next minor and files a note
    Scenario: A feature merge bumps the draft to the next minor
      Given the last published release was "v1.2.0"
      And a pull request labelled "features" merged to main
      When release drafting runs
      Then the draft's proposed version will be "v1.3.0"
      And the pull request's title will appear under the "Features" category
      And the release will remain a draft

    # Source: 030-release-drafting — Scenario: highest semver label wins across several merges
    Scenario: The highest semver label wins across several merges
      Given the last published release was "v1.2.0"
      And pull requests labelled "fixes", "features", and "breaking" merged since then
      When release drafting runs
      Then the draft's proposed version will be "v2.0.0"
      And each pull request's title will appear under its own category

    # Source: 030-release-drafting — Scenario: a docs-only change drafts under Docs with a default patch bump
    Scenario: A change with no semver label takes the default patch bump
      Given the last published release was "v1.2.0"
      And a pull request changing only "README.md", labelled "docs", merged
      When release drafting runs
      Then the pull request's title will appear under the "Documentation" category
      And the draft's proposed version will be "v1.2.1"

    # Source: 030-release-drafting — Scenario: the first ever release proposes v0.1.0 as a pre-release
    Scenario: The first ever release proposes v0.1.0 as a pre-release
      Given no GitHub Release had been published yet
      And a pull request labelled "features" merged to main
      When release drafting runs
      Then the draft's proposed version will be "v0.1.0"
      And the draft will be marked a pre-release

    # Source: 030-release-drafting — Scenario: reconciliation converges rather than duplicating
    Scenario: Reconciliation converges rather than duplicating
      Given a pull request was already reflected in the draft from an earlier run
      When a later merge triggers release drafting again
      Then the draft will reflect the full set of pull requests merged since the last release
      And the earlier pull request's note line will appear exactly once

    # Architecture-informed (proposed by skill — plan ADR-5, status crossover at 1.0.0)
    Scenario: The draft is marked latest once the version reaches 1.0.0
      Given the last published release was "v0.9.0"
      And a pull request labelled "breaking" merged to main
      When release drafting runs
      Then the draft's proposed version will be "v1.0.0"
      And the draft will be marked latest rather than a pre-release

  Rule: Keep release notes about real, user-facing change
    # In order to keep release notes about real, user-facing change,
    # as a maintainer,
    # I want spec/feature-file-only and explicitly-excluded pull requests left out of the notes and the version bump.

    # Source: 030-release-drafting — Scenario: an excluded pull request affects neither notes nor version
    Scenario: An excluded pull request affects neither notes nor version
      Given the last published release was "v1.2.0"
      And a pull request labelled "features" but also carrying the exclusion label merged
      When release drafting runs
      Then the pull request's title will not appear anywhere in the draft notes
      And it will not contribute to the version bump

    # Source: 030-release-drafting — Scenario: a spec-only pull request is omitted without a label
    Scenario: A pull request confined to spec and feature files is omitted
      Given a pull request whose changed files were confined to "specs/" and ".feature" files merged
      When release drafting runs
      Then the pull request's title will not appear in the draft notes
      And it will not drive the version bump, even if it carries a semver-bearing label

    # Source: 030-release-drafting — Scenario: every category the notes can produce traces to an 028 managed label
    @validation @wip
    Scenario: Every release-note category traces to a managed PR Administration label
      Given the release-drafting configuration and PR Administration's managed label set
      When the label-contract guard compares them
      Then the seven note categories will map exactly to "breaking", "features", "fixes", "docs", "infrastructure", "dependencies", and "internal"
      And no note category will reference a label PR Administration does not manage

  Rule: Stop at a draft the maintainer publishes deliberately
    # In order to decide when a version actually ships,
    # as a maintainer,
    # I want Release Drafting to stop at a draft I publish deliberately, rather than releasing automatically on merge.

    # Source: 030-release-drafting — Scenario: the draft is never published automatically
    Scenario: The draft is never published automatically
      Given a pull request merged to main and the draft was updated
      When release drafting completes
      Then the release will remain an unpublished draft with no tag created
      And the Automated Release Pipeline will not run until a maintainer publishes it

    # Architecture-informed (proposed by skill — plan §Non-Blocking, not a required check)
    @validation @wip
    Scenario: A drafting failure blocks nothing
      Given a merge to main that the drafting run fails to complete
      When the repository's merge protection is inspected
      Then release drafting will not be a required, merge-blocking status
      And the previous draft will remain intact for the next merge to reconcile
