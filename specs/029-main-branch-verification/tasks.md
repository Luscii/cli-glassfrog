# Tasks: Main-Branch Verification

**Feature**: 029-main-branch-verification
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-automated-pipeline/main-branch-verification.feature

---

> **No cross-spec gate**: Main-Branch Verification has no hard upstream dependencies (backlog: "Should Have", no `requires:`). It runs the project's *existing* `go test ./...` suite — it adds no Go package and no tests. Both tasks edit only `.github/workflows/` files. **T001 refactors 024's shipped `ci.yml`** (its `test` job becomes a `uses:` call) — this is sanctioned: 024's plan/DECISIONS explicitly left the shared-reusable-workflow call to 029. The refactor preserves 024's required `ci-success` check (it binds the stable name, not matrix-cell names). The two tasks form a sequential chain (T002 calls the workflow T001 creates) and may reasonably collapse into one PR; they are listed separately to mirror the plan's two phases.

## Dependency Graph

Phase 1: Extract the reusable test workflow (1 task, no dependencies) [Shared]
Phase 2: Add the post-merge workflow (1 task, depends on Phase 1) [US1]

2 tasks total | 0 phases parallelizable | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/029-main-branch-verification/base` → `spec/029-main-branch-verification/task-1`, `task-2`.

The `base` branch is cut from current `main`. Because T002 calls the reusable workflow T001 introduces, run them in order on the same base (or as one combined PR). The PR that introduces this change runs through 024's own pre-merge gate (`ci.yml`) — so the `ci.yml` refactor in T001 is validated by the very gate it edits before it merges, and `main-verify.yml` (T002) first fires on the merge of this PR itself. Other pipeline specs may build in parallel on their own base branches; this feature hard-depends on none of them.

---

## Phase 1: Extract the reusable test workflow [Shared]

- [x] **T001** [Shared] Add `.github/workflows/test.yml` (reusable, `on: workflow_call`) holding the OS test matrix, and refactor `.github/workflows/ci.yml`'s `test` job into a `uses:` call to it — actionlint clean; ci-success `needs: [lint, test]` preserved; removed @wip from "mirrors the pre-merge matrix" (the 2 @validation refs held for /score:validate) — the test matrix becomes a single source of truth shared by the pre-merge gate (024) and the post-merge net (029)
  - **Scope**: Create `.github/workflows/test.yml` with `name: Test`, `on: workflow_call` (no `inputs:` — the Go version comes from `go-version-file: go.mod`), and `permissions: contents: read`. Move 024's existing `test` job into it verbatim: a single `test` job with `strategy: { fail-fast: false, matrix: { os: [ubuntu-latest, macos-latest] } }`, `runs-on: ${{ matrix.os }}`, steps `actions/checkout@v4` → `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`) → `go test ./...` (plain, flag-free). In `ci.yml`, replace the inline `test` job with `test: { uses: ./.github/workflows/test.yml }`. Do **not** touch `ci.yml`'s `name`, `pull_request` trigger, `permissions`, PR-ref `concurrency`, `lint` job, or `ci-success` job. Do **not** move `lint` into the reusable workflow — it stays a pre-merge-only concern in `ci.yml`.
  - **Acceptance criteria**:
    - `test.yml` is callable only via `workflow_call`; it runs `go test ./...` on `ubuntu-latest` and `macos-latest`, both cells required, `fail-fast: false` so a failing cell still lets the other report and names the failing OS.
    - `ci.yml`'s `ci-success` job (`needs: [lint, test]`) still resolves: `needs.test.result` aggregates the called workflow's matrix result (`success` iff every cell passed), so 024's pre-merge gate behaves identically — verifiable because this PR runs through that gate.
    - Branch protection on `main` is unchanged: it requires only the stable `ci-success` context; per-cell checks may now display nested (`test / test (ubuntu-latest)`) but nothing requires them by name.
    - No Go package, command, or test is added; `go test ./...` stays a plain invocation.
    - `actionlint` is clean on both `test.yml` and the refactored `ci.yml`.
  - **Dependencies**: none
  - **Plan reference**: Phase 1 (Extract the reusable test workflow); ADR-1 (reusable-workflow single source of truth)
  - **Scenario references**: main-branch-verification.feature: "The post-merge run mirrors the pre-merge matrix"; "The post-merge suite is the same suite, not a superset"; "Lint does not run post-merge"
  - **Interface references**: interface-spec.md: `.github/workflows/test.yml` structure; `.github/workflows/ci.yml` refactor (invariants table)
  - **Risk**: ⚠️ Refactoring 024's shipped `ci.yml` changes job nesting. Mitigated by branch protection binding only the stable `ci-success` name and by this PR running through 024's own gate, which fails loudly if the refactor breaks the aggregation.

## Phase 2: Add the post-merge workflow [US1]

- [ ] **T002** [US1] Add `.github/workflows/main-verify.yml` triggered by `push` to `main`, calling the reusable test workflow — the post-merge net that re-runs the suite on every merge and surfaces a regression as a red run + failing commit status, with no enforcement and no cancelling of in-flight runs
  - **Scope**: Create `.github/workflows/main-verify.yml` with `name: Main-Branch Verification`, `on: push: { branches: [main] }`, and `permissions: contents: read`. Add a single job `test: { uses: ./.github/workflows/test.yml }`. Configure **no cancellation** of in-flight runs (ADR-4): either omit `concurrency` entirely, or use a SHA-keyed group (`group: main-verify-${{ github.sha }}`, `cancel-in-progress: false`) so a later merge never cancels an earlier commit's run. Add **no** lint job, **no** `ci-success`-style aggregation job, and **no** branch-protection change (ADR-3: the failure surface is the workflow run + the commit status GitHub attaches; nothing requires a stable check name post-merge).
  - **Acceptance criteria**:
    - A push to `main` (merge or direct) runs the matrix via the reusable workflow; a green run reports success and a passing commit status, a failing cell turns the run red and attaches a failing commit status — without blocking or reverting anything.
    - A **tag** push does not trigger the workflow (`branches: [main]` matches branch refs only — no `tags-ignore` needed); a **pull request** does not trigger it (no `pull_request` trigger).
    - Each commit landing on `main` gets its own run and its own verdict; a later commit's run does **not** cancel an earlier commit's in-flight run.
    - The run executes tests only (no lint) and adds no new test/package/command — it re-runs exactly what the PR gate ran, via the same `test.yml`.
    - A failing run names the failing environment cell so the failure can be reproduced.
    - `actionlint` is clean on `main-verify.yml`.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 (Add the post-merge workflow); ADR-2 (`push: main` trigger, tags excluded), ADR-3 (no enforcement layer), ADR-4 (independent per-commit, no cancel)
  - **Scenario references**: main-branch-verification.feature: "A clean merge to main verifies green"; "Each merge is verified against its own commit"; "A regression that reaches main is surfaced loudly"; "A tag push does not trigger verification"; "A pull request does not trigger verification"; "An environment-specific failure names its cell"; "Post-merge verification cannot block a merge"
  - **Interface references**: interface-spec.md: `.github/workflows/main-verify.yml` structure; Error Communication (tag/PR no-trigger, net-not-gate, failing cell named)
