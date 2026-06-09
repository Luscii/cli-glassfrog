Feature: No Automated Pipeline — Main-Branch Verification
  On every push to main (a merge landing, or a direct push), the project's
  full test suite re-runs across the same OS matrix the pull-request gate
  ran — the matrix lives in one shared reusable workflow, so "green on main"
  is the same computation as "green on the PR". The post-merge run is a net,
  not a gate: it never blocks, reverts, or gates a merge (the code is already
  on main), it runs tests only (lint stays the pre-merge concern), and tag
  pushes do not trigger it. A failure turns the run red and attaches a failing
  status to the offending commit — the loud signal a human acts on. It is the
  post-merge half of the project's verification story; PR Validation is the
  pre-merge gate.
  (affects: Maintainer)

  Rule: Re-run the full suite on every merge to main
    # In order to find out quickly when a regression reaches main despite passing pre-merge,
    # as a maintainer,
    # I want the full test suite re-run automatically on every merge to main, with a loud red status when it fails.

    # Source: 029-main-branch-verification — Scenario: a clean merge to main verifies green
    @wip
    Scenario: A clean merge to main verifies green
      Given a pull request had merged into "main"
      When the post-merge workflow runs the test matrix against the merge commit
      And every matrix cell passes
      Then the run will report success
      And a passing status will be attached to the merge commit

    # Source: 029-main-branch-verification — Scenario: each merge is verified against its own commit
    @wip
    Scenario: Each merge is verified against its own commit
      Given two commits land on "main" one after the other in quick succession
      When the post-merge workflow runs for each
      Then each run will execute against its own commit
      And a later commit's run will not cancel an earlier commit's in-flight run
      And each commit will receive its own pass or fail status

    # Source: 029-main-branch-verification — Scenario: a regression that reaches main is surfaced loudly
    @wip
    Scenario: A regression that reaches main is surfaced loudly
      Given a commit on "main" whose test suite fails in at least one matrix cell
      When the post-merge workflow runs
      Then the run will report failure
      And a failing status will be attached to that commit
      And neither the commit nor the merge will be blocked or reverted

    # Source: 029-main-branch-verification — Scenario: a tag push does not trigger verification
    @wip
    Scenario: A tag push does not trigger verification
      Given a release tag is pushed
      When the push event fires
      Then the post-merge workflow will not run

    # Source: 029-main-branch-verification — Scenario: a pull request does not trigger verification
    @wip
    Scenario: A pull request does not trigger verification
      Given a pull request is opened or updated against "main"
      When its events fire
      Then the post-merge workflow will not run

    # Source: 029-main-branch-verification — Scenario: post-merge verification cannot block a merge
    @validation @wip
    Scenario: Post-merge verification cannot block a merge
      Given a failing post-merge run on a "main" commit
      When the repository configuration is inspected
      Then the failing status will be informational on the already-merged commit
      And no mechanism will prevent, revert, or gate a merge on that result

  Rule: Mirror the pull-request test suite and matrix
    # In order to trust that "green on main" means the same thing as "green on the PR",
    # as a maintainer or AI agent,
    # I want the post-merge run to mirror the same test suite and environment matrix the pull-request gate ran.

    # Source: 029-main-branch-verification — Scenario: the post-merge run mirrors the pre-merge matrix
    @wip
    Scenario: The post-merge run mirrors the pre-merge matrix
      Given a commit that passed PR Validation across its environment matrix
      When the post-merge workflow runs
      Then it will run the same test suite across the same environment matrix
      And a green post-merge result will carry the same meaning as the green pull-request result

    # Source: 029-main-branch-verification — Scenario: the post-merge suite is the same suite, not a superset
    @validation @wip
    Scenario: The post-merge suite is the same suite, not a superset
      Given a post-merge run on "main"
      When the run is inspected
      Then it will execute the existing test suite invoked from the shared reusable workflow
      And no post-merge-only test, package, or command will have been added

    # Source: 029-main-branch-verification — Scenario: lint does not run post-merge
    @validation @wip
    Scenario: Lint does not run post-merge
      Given a post-merge run on "main"
      When the run is inspected
      Then no lint pass will have executed
      And only the test matrix will have run

  Rule: Name the failing environment and commit
    # In order to reproduce a post-merge failure without guesswork,
    # as a contributor investigating a red main,
    # I want the failing run to name the environment and commit it failed on.

    # Source: 029-main-branch-verification — Scenario: a flaky or environment-specific failure names its cell
    @wip
    Scenario: An environment-specific failure names its cell
      Given the test suite fails in one matrix cell while the other cells pass
      When the post-merge workflow runs on the "main" commit
      Then the run will report failure
      And the failing environment cell will be named so the failure can be reproduced
