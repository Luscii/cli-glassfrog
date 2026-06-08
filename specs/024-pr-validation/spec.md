# Specification: PR Validation

**Feature**: 024-pr-validation
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

PR Validation is the **core CI quality gate** of the glassfrog-cli project. On every pull request targeting `main`, it runs the lint suite and the test suite, and surfaces a single pass/fail status on the pull request. That status is a **required gate**: a pull request cannot merge into `main` until it is green. It is the mechanism that makes "every change is verified before it reaches main" a hard rule rather than a hope — the faithful-CLI discipline (PROJECT constraint "Spec is the contract") extends to the build itself.

It is the **pre-merge** half of the project's verification story, and it is deliberately narrow. Main-Branch Verification (029) re-runs the test suite *after* a merge lands on `main`, as a post-merge safety net. Automated Release Pipeline (022) builds and attaches binaries when a release is published and explicitly assumes the code it ships was already validated here. PR Administration (028) applies triage/release-note labels to pull requests. PR Validation owns none of those — it owns running lint + tests on a pull request and blocking merge until they pass.

---

## Behavioral Accord

### Trigger

- When a pull request targeting `main` is opened, validation runs automatically.
- When new commits are pushed to an open pull request targeting `main` (the head is updated), validation re-runs against the updated head commit.
- When a closed pull request targeting `main` is reopened, validation runs against its current head.
- Validation does not run for pull requests whose base branch is not `main`, nor on a direct push or merge to `main` (Main-Branch Verification (029) owns post-merge), nor on a published release (Automated Release Pipeline (022) owns that).

### Lint

- Validation runs a lint pass over the changed code consisting of a **formatting check**, **`go vet`**, and a **configured aggregate linter** (golangci-lint-style). These are collectively reported as "lint".
- When any part of the lint pass reports a problem, validation fails and names what failed so the contributor can locate it.
- The lint pass runs once per validation run — it is not repeated per environment in the test matrix.

### Tests

- Validation runs the project's full test suite across a **matrix of environments** spanning multiple Go versions and/or operating systems.
- Every cell of the matrix must pass for the test portion to pass.
- When the test suite fails in any matrix cell, validation fails and identifies the failing environment so the contributor can reproduce it.

### Result and gate

- Validation reports a single, aggregate pass/fail status on the pull request: it passes only when the lint pass and every test-matrix cell pass.
- The validation status is a **required gate** — a pull request targeting `main` cannot be merged until validation is green. A failing or missing validation result blocks the merge.
- When all checks pass, the pull request is mergeable with respect to this gate; when any check fails, merge stays blocked until the contributor pushes a fix and validation re-runs green.
- The gate always reflects the **latest** head commit of the pull request — a result for a superseded commit does not satisfy the gate.

---

## User Scenarios

**In order to** trust that no unverified change ever reaches `main`,
**as a** maintainer,
**I want to** have lint and the full test suite run automatically on every pull request and block the merge until they pass.

**In order to** fix a problem before asking for a merge,
**as a** contributor or AI agent submitting a change,
**I want to** see exactly which check failed — the lint problem or the failing test environment — directly on the pull request.

**In order to** keep a red pull request from being merged at all,
**as a** maintainer,
**I want to** the validation status to be a required gate, not just an advisory signal.

---

## Non-Behaviors

- PR Validation must not run on a direct push or merge to `main`. **Why**: Main-Branch Verification (029) owns the post-merge test re-run; duplicating it here would fork the post-merge contract and run the same suite twice on the same commit.
- PR Validation must not build, package, or attach release binaries. **Why**: Automated Release Pipeline (022) owns build-package-attach and assumes already-validated code; mixing artifact production into the pre-merge gate would slow every pull request and blur the two responsibilities.
- PR Validation must not apply, read, or react to pull-request labels. **Why**: PR Administration (028) owns labelling for triage and release-note categorization; the quality gate's verdict depends on code, not on labels.
- PR Validation must not author release notes, compute, or bump a version. **Why**: Release Drafting (030) owns notes and the semver bump; the gate verifies a change, it does not describe or version it.
- PR Validation must not modify the pull request's code — it must not auto-format, auto-fix, or commit changes. **Why**: the gate is an observer that reports pass/fail; silently rewriting a contributor's branch would make the verdict non-reproducible and surprise the author.
- PR Validation must not run for pull requests whose base branch is not `main`. **Why**: the gate's purpose is protecting `main`; running it elsewhere would spend CI on branches it is not responsible for and imply a guarantee it does not own.
- PR Validation must not deploy, publish, or release anything. **Why**: it is a verification gate only; shipping is gated separately on publishing a release (022).

---

## Integration Boundaries

- **GitHub pull-request events (upstream / trigger source)**: opening, updating (push to head), and reopening a pull request targeting `main` triggers a validation run. The gate is unaware of who or what opened the pull request.
- **GitHub status checks and branch protection (destination / enforcement)**: validation publishes one aggregate status on the pull request, and that status is configured as a required check so a failing or absent result blocks the merge. When the CI provider is unavailable and no status is reported, the merge stays blocked (fail-closed) rather than merging unverified.
- **Go toolchain and lint tooling (system actor)**: the test suite and the lint pass (formatting, `go vet`, configured aggregate linter) run against the repository's Go module. Exact tool versions and linter configuration are plan/interface details.
- **Main-Branch Verification (029) — sibling**: re-runs the test suite on merge to `main`. PR Validation is the pre-merge gate; 029 is the post-merge net. The two share the same suite but trigger on different events.
- **Automated Release Pipeline (022) — downstream**: assumes the code reaching a published release was already validated here and does not re-run lint/tests. PR Validation is the upstream guarantee 022 relies on.

---

## Driving Scenarios

### Happy path

**Scenario: a clean pull request passes and becomes mergeable**
Given a pull request targeting `main`
When validation runs the lint pass and the test suite across every matrix cell
And lint reports no problems and every matrix cell passes
Then validation reports a passing status on the pull request
And the required gate is satisfied so the pull request can be merged.

**Scenario: pushing a fix re-runs validation against the new head**
Given a pull request whose validation previously failed
When the contributor pushes a new commit to the pull request
Then validation re-runs against the updated head commit
And the reported status reflects the latest commit, not the superseded one.

**Scenario: a green pull request is allowed to merge**
Given a pull request targeting `main` with a passing validation status
When a maintainer attempts to merge it
Then the required gate does not block the merge.

### Error scenarios

**Scenario: a failing test in one matrix cell blocks the merge**
Given a pull request targeting `main`
When the test suite fails in one matrix cell while other cells pass
Then validation reports a failing status
And names the failing environment
And the required gate blocks the merge until the failure is fixed.

**Scenario: a lint problem blocks the merge**
Given a pull request targeting `main`
When the lint pass reports a formatting, `go vet`, or configured-linter problem
Then validation reports a failing status
And names what failed
And the required gate blocks the merge until it is resolved.

### Edge cases

**Scenario: a pull request targeting a non-main branch does not trigger the gate**
Given a pull request whose base branch is not `main`
When the pull request is opened or updated
Then PR Validation does not run as the required gate for that pull request.

**Scenario: rapid successive pushes resolve to the latest commit**
Given a pull request that receives several pushes in quick succession
When validation runs
Then the gate reflects the result for the latest head commit
And a result for an earlier, superseded commit does not satisfy the gate.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the gate actually blocks, it does not merely report**
Given a pull request targeting `main` with a failing validation status
When a merge is attempted
Then the merge is blocked by the required check — the failing status is not advisory.

**Scenario: lint runs once while tests run per matrix cell**
Given a validation run for a pull request
When the run is inspected
Then the lint pass executed exactly once
And the test suite executed once per matrix cell, with every cell required to pass.

**Scenario: a missing validation result fails closed**
Given a pull request targeting `main` for which no validation status has been reported
When a merge is attempted
Then the required gate blocks the merge rather than allowing an unverified change through.

---

## Assumptions

- **Matrix cells** `[ASSUMED]`: the *behavior* is fixed — the test suite runs across a matrix of multiple Go versions and/or operating systems, and every cell must pass. The exact axes (which Go minor versions, which OSes) are a plan detail; a sensible default mirrors the release OS targets (macOS, Linux) and the project's current plus previous Go minor.
- **Lint composition** `[ASSUMED]`: the lint pass is a formatting check + `go vet` + a configured aggregate linter (golangci-lint-style). The exact linter set and configuration file are plan/interface details; the behavioral requirement is that all three categories run and any problem fails the gate.
- **Required-gate mechanism**: the "required gate" is realized through GitHub branch protection / repository rulesets marking the validation status as required on `main`. The behavioral requirement is "merge blocked until green"; the configuration mechanism is a plan detail.
- **CI provider is GitHub Actions** (technical default): the repository lives on GitHub and ships releases through GitHub (022), so the pull-request trigger and status checks are assumed to run on GitHub Actions. This is an implementation default, not a behavioral requirement.
- **Validated-before-release**: Automated Release Pipeline (022) and its assumptions already record that a published release rests on code validated by this gate; recorded here for traceability of the upstream guarantee.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) PR Validation both runs the checks and owns the **required, merge-blocking gate** (report + enforce), not an advisory status; (2) the trigger is **pull requests targeting `main`** on open, update/push, and reopen — not other base branches, not pushes/merges to `main`, not release publishes; (3) tests run across a **matrix** of Go versions and/or OSes with every cell required to pass, while the **lint pass runs once**; and (4) "lint" means a formatting check + `go vet` + a configured aggregate linter. The remaining `[ASSUMED]` items (exact matrix axes, linter composition, branch-protection mechanism, CI provider) are plan/interface-level details, not behavioral gaps._
