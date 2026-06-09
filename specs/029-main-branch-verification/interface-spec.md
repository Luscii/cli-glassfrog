# Interface Accord: Main-Branch Verification — Specification

**Feature**: 029-main-branch-verification
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the reusable `test.yml` + refactored `ci.yml` + new `main-verify.yml`); ADR-1 (reusable-workflow single-source-of-truth test matrix), ADR-2 (`push: main` trigger, tags excluded), ADR-3 (no enforcement layer — run + commit status only), ADR-4 (each commit verified independently — no cancel)

---

## Surface

This feature is **three** declarative artifacts under `.github/workflows/` — one new reusable workflow, a refactor of 024's shipped `ci.yml`, and one new post-merge workflow. No runtime command is added to the CLI, no Go package, no test.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/workflows/main-verify.yml` | GitHub event `push` filtered to `branches: [main]` | The single automated trigger for the post-merge net. `branches: [main]` matches the `main` branch ref only — a **tag** push does not match, so 022 (release) keeps tags. |
| `.github/workflows/test.yml` | `workflow_call` only | The shared test matrix. Not independently triggerable; invoked via `uses:` by both `ci.yml` (024) and `main-verify.yml` (029). |
| `.github/workflows/ci.yml` | unchanged: `pull_request: { branches: [main] }` | 024's pre-merge gate; its `test` job is refactored to call `test.yml`. Trigger, lint job, and `ci-success` gate are unchanged. |
| Local reproduction | `go test ./...` | Reproduces a `test` matrix cell on the developer's host OS — same invocation pre- and post-merge. |

### `.github/workflows/test.yml` structure (NEW — reusable, ADR-1)

| Element | Contract |
|---|---|
| `name` | `Test` (display name for the called workflow). |
| `on` | `workflow_call` — invocable only by other workflows in this repo. `[ASSUMED]` no `inputs:` initially (the Go version comes from `go-version-file: go.mod`, not a caller input); a `go-version` input is an additive knob if a version axis is later wanted. |
| `permissions` | `contents: read` — a called workflow runs under permissions no broader than the caller grants; both callers grant exactly this. |
| Job `test` (matrix) | `strategy: { fail-fast: false, matrix: { os: [ubuntu-latest, macos-latest] } }`, `runs-on: ${{ matrix.os }}`. Steps: `actions/checkout@v4`; `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`); `go test ./...`. Identical to 024's shipped `test` job — moved here verbatim. Every cell must pass; `fail-fast: false` lets each OS report independently so a failing cell is identifiable by name. |

```yaml
# .github/workflows/test.yml
name: Test
on:
  workflow_call:
permissions:
  contents: read
jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - name: go test
        run: go test ./...
```

### `.github/workflows/ci.yml` refactor (MODIFIED — 024's artifact, ADR-1)

The inline `test` job (currently `runs-on: ${{ matrix.os }}` + the checkout/setup-go/`go test` steps) is replaced by a call to the reusable workflow. Everything else in `ci.yml` — `name`, the `pull_request` trigger, `permissions`, the PR-ref `concurrency` block, the `lint` job, and the `ci-success` job — is **unchanged**.

```yaml
# .github/workflows/ci.yml — the test job becomes:
  test:
    uses: ./.github/workflows/test.yml
```

| Invariant the refactor must preserve | Why it still holds |
|---|---|
| `ci-success` (`needs: [lint, test]`) stays the one required check | `needs.test.result` aggregates a **called workflow's** result the same way it aggregated the inline matrix — `success` iff every cell of `test.yml` passed. The `ci-success` guard step is untouched. |
| Branch protection on `main` is unaffected | It requires only the stable context `ci-success` (024 ADR-4), never the cell contexts. Cell checks now display nested as `test / test (ubuntu-latest)` (vs `test (ubuntu-latest)`), but nothing requires those by name. |
| Lint still runs once, pre-merge only | The `lint` job is **not** moved into `test.yml`; it stays in `ci.yml`. 029 runs tests only. |

### `.github/workflows/main-verify.yml` structure (NEW — ADR-2/3/4)

| Element | Contract |
|---|---|
| `name` | `Main-Branch Verification`. |
| `on` | `push: { branches: [main] }` — every commit landing on `main` (a merge, or a direct push). Tag pushes do not match `branches:`, so they do not trigger it (no `tags-ignore` needed). |
| `permissions` | `contents: read` — the net builds and tests, never writes. No secrets. |
| `concurrency` | **No cancellation** (ADR-4). `[ASSUMED]` realization: either omit `concurrency` entirely, or use a **SHA-keyed** group with `cancel-in-progress: false` (e.g. `group: main-verify-${{ github.sha }}`). The load-bearing contract: a later merge must **not** cancel an in-flight verification of an earlier commit — each commit gets its own verdict. (Deliberate inverse of `ci.yml`'s PR-ref group with `cancel-in-progress: true`.) |
| Job `test` | `uses: ./.github/workflows/test.yml` — the same reusable workflow `ci.yml` calls. No lint job, no `ci-success`-style aggregation job (ADR-3: nothing requires a stable check name here), no branch-protection coupling. |

```yaml
# .github/workflows/main-verify.yml
name: Main-Branch Verification
on:
  push:
    branches: [main]
permissions:
  contents: read
jobs:
  test:
    uses: ./.github/workflows/test.yml
```

---

## Interactions

**Post-merge flow (end-to-end):** A pull request merges into `main` (or any commit is pushed to `main`) → the `push` event starts `main-verify.yml` → its single job calls `test.yml` → the matrix fans out across `ubuntu-latest` + `macos-latest`, each running `go test ./...` → the workflow run is green iff every cell passed, and GitHub attaches the run's pass/fail as a commit status on the merge commit. There is **no gate** — nothing blocks, merges, or reverts on the result; a red run is the signal.

**What `go test ./...` covers:** the same suite 024 runs — the project's existing unit tests, every godog BDD suite, the `internal/build` host self-containment test, and the `internal/build` `.goreleaser.yaml` config-guard test. 029 invents no test logic and adds no flags; it re-runs, post-merge, exactly what the PR gate ran pre-merge.

**Single source of truth:** because both `ci.yml` and `main-verify.yml` call the same `test.yml`, the matrix and the `go test ./...` invocation exist in exactly one place — "green on `main`" is the same computation as "green on the PR" by construction, not by two files agreeing.

**Independent per-commit verdicts (ADR-4):** a burst of rapid merges produces one run per commit; none is cancelled by a later one, so a regression is pinned to the specific commit that introduced it.

---

## Error Communication

| Condition | Behavior |
|---|---|
| Test failure in one OS cell | that `test` cell exits non-zero; `fail-fast: false` lets the other cell finish and report → the `main-verify.yml` run is red → GitHub marks the commit status failed, the failing OS identified by the nested cell name (`test / test (macos-latest)`). No merge is blocked (the code is already on `main`); the red status is the loud signal. |
| Whole suite fails on both cells | both cells red → run red → failing commit status; the run log names the failures per OS. |
| A flaky test that passed on the PR | reddens the `main` run/commit status (this is exactly what the net exists to surface). Remediation (re-run, fix, revert) is a human decision — 029 does not auto-retry or auto-revert. |
| Tag push (release) | `branches: [main]` does not match a tag ref → `main-verify.yml` does not run (022 owns tag-triggered work). |
| Pull request event | `main-verify.yml` has no `pull_request` trigger → it does not run on PRs (024 owns the pre-merge run). |
| GitHub Actions unavailable / run never reported | no commit status appears; because 029 is a **net, not a gate**, there is nothing to fail-closed — the absence is itself visible in the Actions history. (Contrast 024, where a missing `ci-success` blocks the merge.) |

---

## Consistency Notes

- **Sibling boundary (024 PR Validation):** ADR-1 realizes 024's explicit deferral ("029 mirrors the `test` job on `push: main`; whether via a shared reusable workflow is 029's call"). The refactor extracts 024's `test` job into `test.yml` and points `ci.yml` at it — 024's `ci-success` required check and branch protection are preserved (they bind the stable `ci-success` name, not cell names). 029 runs tests only; lint stays 024's pre-merge concern.
- **Sibling boundary (022 Automated Release Pipeline):** 022 triggers on `release: published` (tags); 029's `branches: [main]` filter structurally excludes tags, so the two never overlap. 029 verifies `main` commits; 022 builds/attaches release binaries (a distinct artifact-verification concern).
- **No enforcement, by design (ADR-3):** unlike 024, 029 adds **no** aggregation job, **no** required-check name, and **no** branch-protection change. The whole observable surface is the workflow run + the commit status GitHub attaches. Active alerting / issue-tracking on failure is explicitly out of scope and a clean future addition.
- **Convention — GitHub Actions + pinned action versions:** no `accords/` directory exists; conventions come from PROJECT.md (Go CLI on GitHub) and the existing `ci.yml`/`release.yml`, which pin `actions/checkout@v4` and `actions/setup-go@v5` with `go-version-file: go.mod`. `test.yml` reuses those exact pins verbatim (it is 024's job moved, not rewritten).
- **Assumption — `concurrency` realization (`[ASSUMED]`):** omit-or-SHA-keyed-no-cancel; the contract is "no merge commit's verification is cancelled by a later one." A SHA-keyed group is the explicit form if a `concurrency` block is wanted for housekeeping.
- **Assumption — `workflow_call` inputs (`[ASSUMED]`):** none initially. If 024 later adds a Go-version axis, it becomes a `workflow_call` input that both callers pass, keeping the SoT property.
- **Assumption — `go test` flags:** plain `go test ./...`, matching 024 exactly (a shared reusable workflow makes divergence impossible). Adding `-race`/`-count=1` is an additive knob applied **in `test.yml`**, so it affects both gates at once — never a 029-only flag, which would break the "same meaning" contract.
- **CONSTITUTION XII note:** the Go toolchain is a CI-host tool, not an artifact dependency — XII governs the produced binary's runtime, the same standing this project gives GoReleaser and golangci-lint.
