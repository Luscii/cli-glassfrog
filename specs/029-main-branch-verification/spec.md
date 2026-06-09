# Specification: Main-Branch Verification

**Feature**: 029-main-branch-verification
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Main-Branch Verification is the **post-merge safety net** of the glassfrog-cli project. Whenever a change lands on `main` (a merge, i.e. a push to `main`), it re-runs the project's full test suite against that exact commit. PR Validation (024) already runs lint + tests on every pull request and blocks the merge until they are green, so 029 is not the primary gate — it is the backstop that catches the rare regression that slips through pre-merge: a flaky test that passed on the PR, a semantic conflict between two independently-green branches that only manifests once both are on `main`, or a base that drifted under an un-rebased PR.

Because the code is *already* on `main` by the time 029 runs, it cannot block a merge — that ship has sailed. Its job is to make a post-merge regression **loud and unmissable**: it reports a failing run and a failing commit status on `main`, so the broken commit is visibly red in the Actions history and on the commit itself. It mirrors the same test suite and the same test matrix PR Validation runs, so a green PR and a green post-merge run mean the same thing — verified across the same environments. It deliberately owns nothing else: not lint (pre-merge owns it), not building or releasing (022 owns that), not reverting or alerting beyond the standard CI status surface.

---

## Behavioral Accord

### Trigger

- When a commit is pushed to `main` (the result of merging a pull request, or any direct push to `main`), verification runs automatically against that commit.
- Verification does not run on pull requests — PR Validation (024) owns the pre-merge run; 029 runs only after a change has landed on `main`.
- Verification does not run on tag pushes — the Automated Release Pipeline (022) owns tag-triggered work; 029 is about the `main` branch ref, not release tags.

### Tests

- Verification runs the project's full test suite across the **same matrix of environments** that PR Validation runs (the post-merge run mirrors the pre-merge run so the two carry the same meaning).
- Every cell of the matrix must pass for the verification to pass.
- When the test suite fails in any matrix cell, verification fails and identifies the failing environment so the regression can be reproduced.
- Verification runs the test suite only — it does not run the lint pass.

### Result

- Verification reports a single, aggregate pass/fail status on the `main` commit: it passes only when every test-matrix cell passes.
- When verification fails, the failing run and a failing commit status are visible in the project's CI history and on the offending commit — the regression is surfaced loudly, not silently swallowed.
- Verification does not block, gate, revert, or otherwise alter `main` — the commit is already merged. A failing run is a signal for a human to act on, not an automated rollback.
- Each push to `main` is verified independently against its own commit; the status reflects the commit it ran against.

---

## User Scenarios

**In order to** find out quickly when a regression reaches `main` despite passing pre-merge,
**as a** maintainer,
**I want** the full test suite re-run automatically on every merge to `main`, with a loud red status when it fails.

**In order to** trust that "green on `main`" means the same thing as "green on the PR",
**as a** maintainer or AI agent,
**I want** the post-merge run to mirror the same test suite and environment matrix the pull-request gate ran.

**In order to** reproduce a post-merge failure without guesswork,
**as a** contributor investigating a red `main`,
**I want** the failing run to name the environment and commit it failed on.

---

## Non-Behaviors

- Main-Branch Verification must not block, gate, or prevent any merge. **Why**: it runs *after* the code is already on `main`; PR Validation (024) owns the pre-merge merge-blocking gate. Pretending to block here would be a false guarantee — the change has already landed.
- Main-Branch Verification must not run the lint pass. **Why**: lint (formatting, `go vet`, the aggregate linter) is OS-independent and was already enforced as a required check on the pull request by 024; re-running it post-merge spends CI on a check that cannot have changed since merge and blurs the pre/post-merge split.
- Main-Branch Verification must not build, package, or attach release binaries, nor compute or bump a version. **Why**: the Automated Release Pipeline (022) owns build-package-attach on a published release; 029 verifies a merged commit, it does not ship it.
- Main-Branch Verification must not run on pull requests. **Why**: that is exactly PR Validation's (024) job; duplicating it would run the same suite twice on the same commit and fork the pre-merge contract.
- Main-Branch Verification must not run on tag pushes. **Why**: tags trigger the release pipeline (022); 029 verifies the `main` branch ref, and a tag push is not a merge to `main`.
- Main-Branch Verification must not auto-revert, auto-fix, or modify the failing commit. **Why**: it is an observer that reports a status; silently rewriting `main` history would surprise maintainers and could compound the problem. Remediation is a deliberate human decision.
- Main-Branch Verification must not add any new test, package, or command. **Why**: it re-runs the suite the repository already has (the suite 024 runs); introducing post-merge-only tests would mean `main` is verified against checks the PR never saw, breaking the "same meaning" guarantee.

---

## Integration Boundaries

- **GitHub push events on `main` (upstream / trigger source)**: a push to the `main` branch ref (a merge landing, or a direct push) triggers a verification run against that commit. Tag pushes are excluded.
- **GitHub Actions / commit status (destination)**: verification publishes one aggregate pass/fail status on the `main` commit and a run in the Actions history. A failing run is the loud signal; no further action (issue creation, external notification) is in scope.
- **Go toolchain (system actor)**: the test suite runs against the repository's Go module across the environment matrix. Exact matrix axes and tool versions mirror PR Validation and are plan/interface details.
- **PR Validation (024) — sibling / upstream**: runs lint + the same test matrix pre-merge and blocks the merge until green. 029 re-runs the *test* half post-merge as a net. The two share the same suite and matrix but trigger on different events (pull request vs. push to `main`).
- **Automated Release Pipeline (022) — adjacent**: triggered by published releases / tags, owns build-package-attach, and assumes already-validated code. 029 does not overlap — it verifies `main` commits, not release tags.

---

## Driving Scenarios

### Happy path

**Scenario: a clean merge to main verifies green**
Given a pull request has merged into `main`
When verification runs the full test suite across every matrix cell against the merge commit
And every matrix cell passes
Then verification reports a passing status on the `main` commit.

**Scenario: each merge is verified against its own commit**
Given two pull requests merge into `main` one after the other
When verification runs for each merge
Then each run executes against its own merge commit
And each commit gets its own pass/fail status.

**Scenario: the post-merge run mirrors the pre-merge matrix**
Given a commit that passed PR Validation across its environment matrix
When verification runs post-merge
Then it runs the same test suite across the same environment matrix
So that a green post-merge result carries the same meaning as the green pull-request result.

### Error scenarios

**Scenario: a regression that reaches main is surfaced loudly**
Given a commit on `main` whose test suite fails in at least one matrix cell
When verification runs
Then verification reports a failing status on that commit
And names the failing environment
And the failing run is visible in the CI history — without blocking or reverting anything.

**Scenario: a flaky or environment-specific failure names its cell**
Given the test suite fails in one matrix cell while other cells pass
When verification runs on the `main` commit
Then verification reports a failing status
And identifies which environment cell failed so it can be reproduced.

### Edge cases

**Scenario: a tag push does not trigger verification**
Given a release tag is pushed
When the push event fires
Then Main-Branch Verification does not run (the release pipeline (022) owns tag-triggered work).

**Scenario: a pull request does not trigger verification**
Given a pull request is opened or updated against `main`
When its events fire
Then Main-Branch Verification does not run (PR Validation (024) owns the pre-merge run).

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: post-merge verification cannot block a merge**
Given a failing verification run on a `main` commit
When the repository is inspected
Then the failing status is informational on the already-merged commit — there is no mechanism by which 029 prevents, reverts, or gates a merge.

**Scenario: the post-merge suite is the same suite, not a superset**
Given a verification run on `main`
When the run is inspected
Then it executes the test suite the repository already has (the one 024 runs) with no post-merge-only test, package, or command added.

**Scenario: lint does not run post-merge**
Given a verification run on `main`
When the run is inspected
Then no lint pass (formatting, `go vet`, aggregate linter) executed — only the test matrix ran.

---

## Assumptions

- **Matrix parity** `[ASSUMED]`: the *behavior* is fixed — the post-merge run mirrors PR Validation's test matrix so the two carry the same meaning. The concrete axes (which OSes, which Go version) are a plan detail and should track 024's matrix; if 024's matrix changes, 029's tracks it.
- **CI provider is GitHub Actions** (technical default): the repository lives on GitHub and the pre-merge gate (024) and release pipeline (022) already run there, so the push-to-`main` trigger and commit status are assumed to run on GitHub Actions. An implementation default, not a behavioral requirement.
- **Failure surface is the standard CI status** (per the defining conversation): a failing run + failing commit status visible in Actions and on the commit is the whole observable outcome — no issue creation and no external notification (channel/mention) are in scope for this spec. If maintainers later want active alerting, that is a separate, additive feature.
- **Triggered on the `main` branch ref**: "merge to `main`" is realized as a push to the `main` branch; the behavioral requirement is "every change that lands on `main` is re-verified", with tag pushes excluded.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) on failure the observable outcome is a **red run + failing commit status** and nothing more (no issue, no external notification); (2) the post-merge run executes the **test suite only**, mirroring 024's environment matrix (lint stays a pre-merge-only concern); and (3) the trigger is **every push to `main`** (each merge), **excluding tag pushes** (022 owns those). The remaining `[ASSUMED]` items (concrete matrix axes, CI provider) are plan/interface-level details, not behavioral gaps._
