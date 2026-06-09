# Source: 024-pr-validation — Scenario: a clean pull request passes and becomes mergeable

Feature: No Automated Pipeline — PR Validation
  On every pull request targeting main, lint (a formatting check, `go vet`,
  and a configured aggregate linter) runs once and the test suite runs across
  an OS matrix, every cell required to pass. A single aggregate status
  (`ci-success`) is green only when lint and every test cell pass, and branch
  protection requires it — so a pull request cannot merge until it is green.
  The gate reflects the latest head commit, names what failed when it fails,
  and fails closed when no result is reported. It is the pre-merge half of the
  project's verification story; Main-Branch Verification re-runs the suite
  post-merge.
  (affects: Maintainer)

  Rule: Verify every change to main before it merges
    # In order to trust that no unverified change ever reaches main,
    # as a maintainer,
    # I want lint and the full test suite to run automatically on every pull request and block the merge until they pass.

    # Source: 024-pr-validation — Scenario: a clean pull request passes and becomes mergeable
    @wip
    Scenario: A clean pull request passes and becomes mergeable
      Given a pull request had been opened against "main"
      When the lint job and the test matrix run
      And lint reports no problems and every matrix cell passes
      Then the "ci-success" check will report success
      And the pull request will be mergeable with respect to the gate

    # Source: 024-pr-validation — Scenario: pushing a fix re-runs validation against the new head
    @wip
    Scenario: Pushing a fix re-runs validation against the new head
      Given a pull request whose validation had previously failed
      When the contributor pushes a new commit to the pull request
      Then validation will re-run against the updated head commit
      And the reported status will reflect the latest commit rather than the superseded one

    # Source: 024-pr-validation — Scenario: rapid successive pushes resolve to the latest commit
    @wip
    Scenario: Rapid successive pushes resolve to the latest commit
      Given a pull request that receives several pushes in quick succession
      When validation runs
      Then the gate will reflect the result for the latest head commit
      And a result for an earlier superseded commit will not satisfy the gate

    # Source: 024-pr-validation — Scenario: a pull request targeting a non-main branch does not trigger the gate
    @wip
    Scenario: A pull request targeting a non-main branch does not trigger the gate
      Given a pull request whose base branch is not "main"
      When the pull request is opened or updated
      Then PR Validation will not run as the required gate for that pull request

    # Source: 024-pr-validation — Scenario: lint runs once while tests run per matrix cell
    @validation @wip
    Scenario: Lint runs once while tests run per matrix cell
      Given a validation run for a pull request
      When the run is inspected
      Then the lint job will have executed exactly once
      And the test suite will have executed once per matrix cell, with every cell required to pass

  Rule: Surface exactly which check failed
    # In order to fix a problem before asking for a merge,
    # as a contributor or AI agent submitting a change,
    # I want to see exactly which check failed — the lint problem or the failing test environment — on the pull request.

    # Source: 024-pr-validation — Scenario: a failing test in one matrix cell blocks the merge
    @wip
    Scenario: A failing test in one matrix cell blocks the merge
      Given a pull request targeting "main"
      When the test suite fails in one matrix cell while the other cells pass
      Then the "ci-success" check will report failure
      And the failing environment will be named
      And the merge will be blocked until the failure is fixed

    # Source: 024-pr-validation — Scenario: a lint problem blocks the merge
    @wip
    Scenario: A lint problem blocks the merge
      Given a pull request targeting "main"
      When the lint job reports a formatting, "go vet", or configured-linter problem
      Then the "ci-success" check will report failure
      And the problem will be named
      And the merge will be blocked until it is resolved

  Rule: Enforce the gate rather than merely reporting
    # In order to keep a red pull request from being merged at all,
    # as a maintainer,
    # I want the validation status to be a required gate, not just an advisory signal.

    # Source: 024-pr-validation — Scenario: a green pull request is allowed to merge
    @wip
    Scenario: A green pull request is allowed to merge
      Given a pull request targeting "main" with a passing "ci-success" status
      When a maintainer attempts to merge it
      Then the required gate will not block the merge

    # Source: 024-pr-validation — Scenario: the gate actually blocks, it does not merely report
    @validation @wip
    Scenario: The gate blocks rather than merely reporting
      Given a pull request targeting "main" with a failing "ci-success" status
      When a merge is attempted
      Then the merge will be blocked by the required check
      And the failing status will not be advisory

    # Source: 024-pr-validation — Scenario: a missing validation result fails closed
    @validation @wip
    Scenario: A missing validation result fails closed
      Given a pull request targeting "main" for which no "ci-success" status had been reported
      When a merge is attempted
      Then the required gate will block the merge
      And an unverified change will not be allowed through
