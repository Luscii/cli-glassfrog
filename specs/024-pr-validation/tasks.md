# Tasks: PR Validation

**Feature**: 024-pr-validation
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-automated-pipeline/pr-validation.feature

---

> **No cross-spec gate**: PR Validation has no upstream dependencies (backlog: "no dependencies"). It runs the project's *existing* `go test ./...` suite and lint tooling — it adds no Go package and no tests. All tasks are ready to implement now. The three tasks all edit the single new `.github/workflows/ci.yml`, so they form a sequential chain and may reasonably collapse into one or two PRs (the plan notes the phases "may collapse"); they are listed separately to mirror the plan's phase structure.

## Dependency Graph

Phase 1: Lint Config + Lint Job (1 task, no dependencies) [Shared]
Phase 2: Test Matrix Job (1 task, depends on Phase 1) [Shared]
Phase 3: Aggregation Gate + Enforcement (1 task, depends on Phase 2) [US3]

3 tasks total | 0 phases parallelizable | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/024-pr-validation/base` → `spec/024-pr-validation/task-1`, `task-2`, `task-3`.

The `base` branch is cut from current `main`. Because all three tasks edit the same `ci.yml`, run them in order on the same base (or as one combined PR). Other pipeline specs (023, 025, 026, 028, 029, 030, …) may build in parallel on their own base branches; this feature hard-depends on none of them. Note: the PR that introduces `ci.yml` is itself the gate's first subject — once merged, every subsequent PR (including the later tasks here) runs through it.

---

## Phase 1: Lint Config + Lint Job [Shared]

- [ ] **T001** [Shared] Create `.github/workflows/ci.yml` (PR→main trigger) with the lint job, and add `.golangci.yml`
  - **Scope**: Create `.github/workflows/ci.yml` with `name: CI`, `on: pull_request: { branches: [main] }` (no explicit `types:` — the defaults `opened`/`synchronize`/`reopened` are the spec's three triggers), and least-privilege `permissions: contents: read`. Add a `lint` job (`runs-on: ubuntu-latest`): `actions/checkout@v4`; `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`); a gofmt-check step (fail when `gofmt -l .` lists any file); `go vet ./...`; `golangci/golangci-lint-action@v6` with a **pinned** `version:` (not `latest`). Add `.golangci.yml` enabling a conservative linter set (`[ASSUMED]` starting set: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gofmt`, `misspell`). The lint job runs **once** — it is not part of the matrix. Do **not** add the test job or the gate yet (T002/T003).
  - **Acceptance criteria**:
    - A pull request whose base is `main` triggers the workflow; a pull request whose base is not `main` does not.
    - The `lint` job fails (non-zero) and names the offending files/diagnostics when code is unformatted, when `go vet ./...` reports a problem, or when `golangci-lint run` reports a finding; it passes on clean code.
    - golangci-lint is pinned to a concrete version compatible with the `go.mod` Go version — not `latest`.
    - `permissions: contents: read` only; the job needs no secrets and works under a fork PR's read-only token.
  - **Dependencies**: none
  - **Plan reference**: Phase 1 (Lint config + lint job); ADR-1 (trigger/scope), ADR-2 (lint-once)
  - **Scenario references**: pr-validation.feature: "A lint problem blocks the merge"; "Lint runs once while tests run per matrix cell" (lint-once half); "A pull request targeting a non-main branch does not trigger the gate"
  - **Interface references**: interface-spec.md: `.github/workflows/ci.yml` structure (Job `lint`, `on`, `permissions`); `.golangci.yml` structure

## Phase 2: Test Matrix Job [Shared]

- [ ] **T002** [Shared] Add the OS-matrix `test` job and the `concurrency` cancel-in-progress block
  - **Scope**: Add a `test` job to `ci.yml` with `strategy: { fail-fast: false, matrix: { os: [ubuntu-latest, macos-latest] } }`, `runs-on: ${{ matrix.os }}`: `actions/checkout@v4`; `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`); `go test ./...` (plain invocation, kept flag-free so Main-Branch Verification (029) can mirror it; `-race`/`-count=1` is an optional additive knob). Add a top-level `concurrency` block keyed on the PR (`group: ${{ github.workflow }}-${{ github.event.pull_request.number }}`, `cancel-in-progress: true`) so a new push cancels the superseded run.
  - **Acceptance criteria**:
    - `go test ./...` runs on both `ubuntu-latest` and `macos-latest`; both cells must pass for the test portion to pass, and a failure in one cell still lets the other report (`fail-fast: false`), with the failing OS identifiable from the cell name.
    - The suite exercised includes the existing unit tests, every godog BDD suite, the `internal/build` host self-containment test, and the `internal/build` `.goreleaser.yaml` config-guard test (the drift gate 022 relies on) — no new tests are added by this task.
    - Pushing a new commit to an open PR re-runs validation against the new head, and the prior in-flight run is cancelled (verifiable via the concurrency group); a superseded run does not report success.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 (Test matrix job); ADR-3 (OS matrix)
  - **Scenario references**: pr-validation.feature: "A clean pull request passes and becomes mergeable"; "Pushing a fix re-runs validation against the new head"; "Rapid successive pushes resolve to the latest commit"; "A failing test in one matrix cell blocks the merge"; "Lint runs once while tests run per matrix cell" (per-cell half)
  - **Interface references**: interface-spec.md: `.github/workflows/ci.yml` structure (Job `test` (matrix), `concurrency`)

## Phase 3: Aggregation Gate + Enforcement [US3]

- [ ] **T003** [US3] Add the `ci-success` aggregation job and document/script the branch-protection rule
  - **Scope**: Add a `ci-success` job (`runs-on: ubuntu-latest`, `needs: [lint, test]`, `if: always()`) whose single guard step exits non-zero unless `needs.lint.result == "success"` and `needs.test.result == "success"` (the `needs.test.result` aggregates the whole matrix, so a failed or cancelled cell fails the gate). Document and provide the maintainer-run branch-protection step using the **narrow** required-status-checks sub-resource — a `gh api --method PATCH repos/{owner}/{repo}/branches/main/protection/required_status_checks` call adding the `ci-success` context (or a repository ruleset) — **never** the full-document `PUT …/protection` endpoint, which replaces the whole protection config and would clobber existing review/restriction settings. Capture it in a short `CONTRIBUTING`/CI note or a committed setup script. Only `ci-success` is required (never the drifting matrix-cell contexts).
  - **Acceptance criteria**:
    - `ci-success` reports success **iff** the `lint` job and **every** `test` matrix cell succeeded; it fails when any did not, and (via `if: always()`) it runs and fails loudly rather than being skipped when a dependency fails.
    - A cancelled (superseded) run leaves `ci-success` non-success, so it never satisfies the gate.
    - The branch-protection step requires the single `ci-success` context on `main`; after it is applied, a PR with a failing or missing `ci-success` cannot be merged (the gate blocks, not merely reports), and a green `ci-success` permits merge.
    - Applying the step adds `ci-success` to the required checks **without** disabling or overwriting any pre-existing review requirements or restrictions (it touches only the required-status-checks sub-resource, not the full protection document).
    - The documented/scripted enforcement step is reproducible by a maintainer with admin rights; the limitation that branch protection is repo settings (not an auto-applied committed file) is stated.
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 (Gate + enforcement); ADR-4 (`ci-success` aggregation), ADR-5 (branch-protection enforcement)
  - **Scenario references**: pr-validation.feature: "A green pull request is allowed to merge"; "The gate blocks rather than merely reporting"; "A missing validation result fails closed"; "A clean pull request passes and becomes mergeable" (ci-success success)
  - **Interface references**: interface-spec.md: `ci-success` guard step; Branch-protection contract; Error Communication (missing result fails closed; rule-not-applied limitation)
  - **Risk**: ⚠️ Branch protection lives in repo settings, not a committed file — it can be forgotten or drift. The stable `ci-success` name + the scripted `gh api` step mitigate; the "report" half stays visible even if the "required" flag lapses.
