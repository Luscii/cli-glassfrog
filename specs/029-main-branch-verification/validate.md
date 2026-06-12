# Validate: Main-Branch Verification

**Feature**: 029-main-branch-verification
**Round**: 1 of 3
**Date**: 2026-06-09
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/no-automated-pipeline/main-branch-verification.feature, PROJECT.md
**Implementation files**: 3 GitHub Actions workflows under `.github/workflows/` — `test.yml` (new, reusable), `main-verify.yml` (new, post-merge), `ci.yml` (024's artifact, `test` job refactored to a `uses:` call)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

All driving scenarios referenced by the two checked tasks trace to identifiable code paths. This feature is declarative CI infrastructure (no runnable Go harness — confirmed by the absence of godog bindings for `features/no-automated-pipeline/`); coverage is traced against the workflow YAML, consistent with the plan's testing strategy (§ Cross-cutting Concerns) and sibling 024's precedent.

| Scenario | Status | Implementation |
|---|---|---|
| A clean merge to main verifies green | ✓ Covered | `main-verify.yml` `on: push: { branches: [main] }` → job `test: uses: ./.github/workflows/test.yml`; run is green iff every matrix cell passes; GitHub attaches the run status to the commit |
| Each merge is verified against its own commit | ✓ Covered | `main-verify.yml` `concurrency: { group: main-verify-${{ github.sha }}, cancel-in-progress: false }` — SHA-keyed, no cancel, so distinct commits never collide and no in-flight run is superseded |
| A regression that reaches main is surfaced loudly | ✓ Covered | `test.yml` `fail-fast: false` matrix; a failing cell exits non-zero → run red; `main-verify.yml` has no aggregation/branch-protection/revert step, so nothing blocks or reverts |
| A tag push does not trigger verification | ✓ Covered | `main-verify.yml` `branches: [main]` matches branch refs only — a tag ref does not match (no `tags-ignore` needed) |
| A pull request does not trigger verification | ✓ Covered | `main-verify.yml` has only a `push` trigger; no `pull_request` trigger present |
| The post-merge run mirrors the pre-merge matrix | ✓ Covered | both `ci.yml` and `main-verify.yml` resolve `test` via `uses: ./.github/workflows/test.yml` — one matrix definition, so "green on main" is the same computation as "green on the PR" |
| An environment-specific failure names its cell | ✓ Covered | `test.yml` `fail-fast: false` lets each OS report independently; the failing cell is named (e.g. `test / test (macos-latest)`) |

## Acceptance Criteria

**Status**: Pass (2 of 2 checked tasks, all criteria met)

**T001 — reusable `test.yml` + `ci.yml` refactor**:
- `test.yml` invocable only via `workflow_call` (`on: workflow_call`, no other trigger) — met.
- Runs `go test ./...` on `ubuntu-latest` + `macos-latest`, both required, `fail-fast: false` names the failing OS — met (`test.yml` job `test`).
- `ci.yml`'s `ci-success` (`needs: [lint, test]`) still resolves — met: `test` and `lint` jobs both present; `test` now a `uses:` call whose result `needs.test.result` aggregates the same way. The `ci-success` guard step is untouched.
- Branch protection on `main` unchanged — met: no branch-protection artifact touched; only the stable `ci-success` context is required (cell checks merely nest as `test / test (...)`).
- No Go package/command/test added; `go test ./...` stays plain — met.
- `actionlint` clean on `test.yml` and refactored `ci.yml` — verified during implementation.

**T002 — `main-verify.yml` post-merge net**:
- Push to `main` runs the matrix; green → success + passing commit status; failing cell → red + failing commit status; no block/revert — met (trigger + `uses:` + no enforcement layer).
- Tag push does not trigger (`branches: [main]`); PR does not trigger (no `pull_request`) — met.
- Each commit gets its own run/verdict; a later commit does not cancel an earlier in-flight run — met (SHA-keyed `concurrency`, `cancel-in-progress: false`).
- Tests only, no new test/package/command — met (single `test` job via shared `test.yml`).
- Failing run names the failing cell — met (`fail-fast: false`).
- `actionlint` clean on `main-verify.yml` — verified during implementation.

## Interface Contract Conformance

**Status**: Pass (3 of 3 surfaces conformant)

| Surface | Status | Conformance |
|---|---|---|
| `.github/workflows/test.yml` (new, reusable) | ✓ Conformant | `name: Test`, `on: workflow_call` (no `inputs:`), `permissions: contents: read`, job `test` with `fail-fast: false` matrix `os: [ubuntu-latest, macos-latest]`, `runs-on: ${{ matrix.os }}`, steps `checkout@v4` → `setup-go@v5` (`go-version-file: go.mod`, `cache: true`) → `go test ./...` — matches the interface YAML verbatim |
| `.github/workflows/ci.yml` (024, refactored) | ✓ Conformant | inline `test` job replaced by `test: { uses: ./.github/workflows/test.yml }`; `name`, `pull_request` trigger, `permissions`, PR-ref `concurrency`, `lint` job, and `ci-success` job all unchanged — all three invariants in the interface table hold |
| `.github/workflows/main-verify.yml` (new) | ✓ Conformant | `name: Main-Branch Verification`, `on: push: { branches: [main] }`, `permissions: contents: read`, `concurrency` realized as the SHA-keyed no-cancel form (`group: main-verify-${{ github.sha }}`, `cancel-in-progress: false`) — one of the two `[ASSUMED]` realizations the interface sanctions — and job `test: { uses: ./.github/workflows/test.yml }` with no lint and no aggregation job |

## Non-Behavior Absence

**Status**: Pass (7 of 7 exclusions absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not block/gate/prevent a merge | ✓ Absent | `main-verify.yml` has no `ci-success`-style aggregation job, no required-check name, no branch-protection coupling |
| Must not run the lint pass | ✓ Absent | neither `main-verify.yml` nor `test.yml` runs gofmt/`go vet`/golangci-lint; lint stays solely in `ci.yml`'s `lint` job |
| Must not build/package/attach release binaries or bump a version | ✓ Absent | no goreleaser/build/release steps; only `go test ./...` |
| Must not run on pull requests | ✓ Absent | no `pull_request` trigger in `main-verify.yml` |
| Must not run on tag pushes | ✓ Absent | `branches: [main]` matches branch refs only |
| Must not auto-revert/auto-fix/modify the failing commit | ✓ Absent | no revert/commit/push step; `permissions: contents: read` cannot write the repo |
| Must not add any new test/package/command | ✓ Absent | plain `go test ./...`; no new files under the Go module |

## @wip Lifecycle Completion

**Status**: Pass

The 7 behavioral scenarios referenced by the checked tasks (T001: "mirrors the pre-merge matrix"; T002: the other six) have had `@wip` removed. The 3 remaining `@wip` tags are all on `@validation`-tagged scenarios — held out from the Builder and excluded from implement's removal scope, correctly retained for this validation pass.

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| Post-merge verification cannot block a merge | ✓ Satisfied | `main-verify.yml` has no aggregation job, no required-check name, and no branch-protection coupling; `permissions: contents: read` cannot revert/push. The only required check on `main` is `ci.yml`'s pre-merge `ci-success`, which `main-verify.yml` neither defines nor touches — a failing post-merge run is informational only |
| The post-merge suite is the same suite, not a superset | ✓ Satisfied | `main-verify.yml`'s `test` job is `uses: ./.github/workflows/test.yml` — the very workflow `ci.yml` calls; `test.yml` runs plain `go test ./...` with no post-merge-only test, package, or command added anywhere |
| Lint does not run post-merge | ✓ Satisfied | `main-verify.yml` has a single job calling `test.yml`; `test.yml` contains only the `test` job (checkout → setup-go → `go test`) — no gofmt/`go vet`/golangci-lint. Lint exists only in `ci.yml`'s `lint` job, which was deliberately not extracted into `test.yml` |

---

## Verdict: Ready

All 5 conformance dimensions pass. All 3 validation scenarios are satisfied through independent inspection. Both tasks are complete. The implementation conforms to the specification: the post-merge net re-runs the existing test suite on every push to `main` via the same reusable matrix the PR gate uses, surfaces a regression as a red run + failing commit status with no enforcement layer, excludes tags and PRs structurally, and verifies each commit independently with no cancellation — matching the Behavioral Accord, the Non-Behaviors, and the interface contracts.

Note (not a finding): full runtime confirmation of the GitHub-side behavior (the `ci.yml` refactor passing 024's own gate on this PR, and `main-verify.yml` first firing green on the merge of this PR) is observable only once the PR runs and merges, per the plan's testing strategy. The static/structural conformance is complete.

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.
