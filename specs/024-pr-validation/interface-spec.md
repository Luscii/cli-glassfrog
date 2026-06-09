# Interface Accord: PR Validation — Specification

**Feature**: 024-pr-validation
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the `.github/workflows/ci.yml` lint/test/gate jobs + `.golangci.yml`); ADR-1 (`pull_request`→`main` trigger), ADR-2 (lint-once job), ADR-3 (OS test matrix), ADR-4 (`ci-success` aggregation gate), ADR-5 (branch-protection enforcement)

---

## Surface

This feature is two declarative artifacts at the repo root plus one repository-settings contract. There is no runtime command added to the CLI.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/workflows/ci.yml` | GitHub event `pull_request` with the default activity types (`opened`, `synchronize`, `reopened`), filtered to `branches: [main]` | The single automated trigger. `synchronize` covers every push to the PR head; the `branches` filter scopes to PRs whose **base** is `main`, so a PR targeting any other base does not run. |
| Local lint reproduction | `gofmt -l .` + `go vet ./...` + `golangci-lint run` | Reproduces the `lint` job before pushing. |
| Local test reproduction | `go test ./...` | Reproduces a `test` matrix cell on the developer's host OS. |
| Branch-protection setup | a repository ruleset (`POST …/rulesets`), or `gh api --method POST …/required_status_checks/contexts` on existing protection | A one-time maintainer (admin) step that adds `ci-success` as a required check **additively** — never the full-document `PUT …/protection` (full replace) nor a `PATCH …/required_status_checks` sending only `ci-success` (replaces the contexts list). Not auto-applied by any committed file. |

### `.github/workflows/ci.yml` structure

| Element | Contract |
|---|---|
| `name` | `CI` |
| `on` | `pull_request: { branches: [main] }` — no explicit `types:` (the defaults `opened`/`synchronize`/`reopened` are exactly the spec's three triggers). |
| `permissions` | `contents: read` — the only privilege; no secrets, no write. Fork PRs run identically under the default read-only token. |
| `concurrency` | `group: ${{ github.workflow }}-${{ github.event.pull_request.number }}`, `cancel-in-progress: true` — a new push cancels the superseded in-flight run, so the gate reflects the latest head commit. |
| Job `lint` | `runs-on: ubuntu-latest`. `actions/checkout@v4`; `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`); a **gofmt check** step (`gofmt -l .` — fail if the output is non-empty, listing the unformatted files); `go vet ./...`; `golangci/golangci-lint-action@v6` (`version:` pinned, see `.golangci.yml`). Runs **once** — not in the matrix. |
| Job `test` (matrix) | `strategy: { fail-fast: false, matrix: { os: [ubuntu-latest, macos-latest] } }`, `runs-on: ${{ matrix.os }}`. `actions/checkout@v4`; `actions/setup-go@v5` (`go-version-file: go.mod`, `cache: true`); `go test ./...`. `fail-fast: false` so a failure in one OS still reports the other cell's result. Every cell must pass. |
| Job `ci-success` | `runs-on: ubuntu-latest`, `needs: [lint, test]`, `if: always()`. A single guard step asserts every dependency succeeded — green **iff** `lint` and **every** `test` cell passed. This is the one stable status context branch protection requires. |

**`ci-success` guard step** (the explicit success assertion — `if: always()` makes the job run even when a dependency failed, so it can fail loudly rather than be skipped):

```yaml
ci-success:
  needs: [lint, test]
  if: always()
  runs-on: ubuntu-latest
  steps:
    - name: Verify required jobs succeeded
      run: |
        if [ "${{ needs.lint.result }}" != "success" ] || [ "${{ needs.test.result }}" != "success" ]; then
          echo "One or more required jobs did not succeed (lint=${{ needs.lint.result }}, test=${{ needs.test.result }})."
          exit 1
        fi
```

`needs.test.result` aggregates the whole matrix: it is `success` only when every cell succeeded, and `failure`/`cancelled` otherwise — so a cancelled (superseded) run also fails the gate, never silently passes.

### `.golangci.yml` structure

| Element | Contract |
|---|---|
| `version` of golangci-lint | Pinned via the `version:` input of `golangci-lint-action@v6` (a concrete tag, not `latest`) so a new upstream release cannot redden unrelated PRs. `[ASSUMED]` exact tag — set to a release compatible with the `go.mod` Go version at implementation time. |
| `linters` | An explicit enabled set over golangci-lint's defaults. `[ASSUMED]` starting set: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gofmt`, `misspell`. Conservative and low-false-positive; tunable in a follow-up. |
| `run` | Lints the whole module (`./...`); no per-directory excludes beyond generated files, if any. |

### Branch-protection contract (repository settings — ADR-5)

The enforcement half lives in repo settings, applied once by a maintainer with admin rights. The committed artifacts guarantee the stable `ci-success` context; this step adds it as a required check on `main` **additively** — without dropping other already-required checks or relaxing unrelated protections. The classic required-check lists are *replace-on-write*, so endpoint choice matters.

**Preferred — a repository ruleset** (`POST …/rulesets`): additive by construction — it layers a `required_status_checks` rule for `ci-success` on top of whatever already exists, works whether or not classic branch protection is configured (no precondition, no 404 on a fresh repo), and never owns the rest of the protection config. The exact ruleset JSON is a setup detail.

**Classic branch protection** — only when a protection rule with required status checks **already exists**, add the context additively (leaves existing required contexts in place):

```bash
gh api --method POST \
  repos/{owner}/{repo}/branches/main/protection/required_status_checks/contexts \
  -f 'contexts[]=ci-success'
```

Pitfalls to avoid:
- **Do not** `PATCH …/protection/required_status_checks` with only `contexts[]=ci-success` — the `contexts` list is **replaced**, silently dropping any already-required checks (e.g. code/secret scanning). If a PATCH is unavoidable, send the **union** of the existing contexts plus `ci-success`.
- **Do not** use the full-document `PUT …/branches/main/protection` — it **replaces the entire** protection configuration, so any field omitted (or set to `null` — e.g. `required_pull_request_reviews`, `restrictions`) is disabled, relaxing protections unrelated to this gate.
- The additive `…/required_status_checks/contexts` POST **404s when no protection / required-status-checks config exists yet** — for initial setup, create protection first (a ruleset or the repo UI), then add the context.

Only `ci-success` is required (never the matrix-cell contexts, whose names drift with the matrix — ADR-4). `strict` (require branches up to date before merge) and any admin-enforcement policy (`enforce_admins`) are separate `[ASSUMED]` maintainer-policy decisions, not part of requiring this gate. The load-bearing contract is that `ci-success` is a required check that blocks merge until green.

---

## Interactions

**Pre-merge flow (end-to-end):** A contributor opens (or pushes to, or reopens) a pull request whose base is `main` → the `pull_request` event starts `ci.yml` → `lint` runs once (gofmt-check → `go vet` → `golangci-lint`) and `test` fans out across `ubuntu-latest` + `macos-latest`, each running `go test ./...` → `ci-success` runs after both and is green only if `lint` and every `test` cell passed → branch protection, requiring `ci-success`, permits the merge only when it is green.

**What `go test ./...` covers:** the project's existing unit tests, every godog BDD suite (`internal/cli`, `internal/auth`, …), the `internal/build` host self-containment test, **and** the `internal/build` `.goreleaser.yaml` config-guard test that 022's release pipeline relies on PR Validation to run (catching a lost build target, `CGO_ENABLED=1`, or a missing `archives`/`checksum`/`release` section before a release is ever cut). PR Validation invents no test logic — it orchestrates the suite the repo already has.

**Re-run on update:** because `synchronize` is a default trigger, each push to the PR head re-runs the workflow against the new head commit, and `cancel-in-progress` cancels the superseded run — so the reported `ci-success` always corresponds to the latest commit.

**Latest-commit / superseded semantics:** a cancelled run leaves `needs.test.result`/`needs.lint.result` non-`success`, so the `ci-success` guard fails — a superseded result can never satisfy the gate.

**Shared suite with 029:** Main-Branch Verification (029) runs the same `go test ./...` on `push: main`. The invocation is kept plain (no bespoke flags) so 029 mirrors the `test` job directly; whether the two share a reusable workflow is 029's decision.

---

## Error Communication

| Condition | Behavior |
|---|---|
| Unformatted code | `gofmt -l .` lists the files and the `lint` step exits non-zero → `lint` fails → `ci-success` fails → merge blocked, the unformatted files named in the step log. |
| `go vet` problem | `go vet ./...` exits non-zero → `lint` fails → `ci-success` fails → merge blocked, vet diagnostics in the log. |
| Linter finding | `golangci-lint run` exits non-zero → `lint` fails → `ci-success` fails → merge blocked, findings annotated on the PR (the action surfaces them as inline annotations). |
| Test failure in one OS cell | that `test` cell exits non-zero; `fail-fast: false` lets the other cell finish and report → `ci-success` fails (the matrix result is not `success`) → merge blocked, the failing OS identified by the cell name (`test (macos-latest)`). |
| PR base is not `main` | the `branches: [main]` filter excludes it → the workflow does not run → `ci-success` is not produced for that PR (it is not the required gate there). |
| New push mid-run | `concurrency` cancels the in-flight run; the cancelled jobs report `cancelled`, so `ci-success` for the superseded commit fails, and a fresh run starts for the new head. |
| `ci-success` missing/never reported | branch protection treats a required check that has not reported success as unmet → merge blocked (fail-closed). |
| Branch-protection rule not applied | the workflow still runs and reports `ci-success` (the "report" half is intact and visible), but the merge is not *blocked* until a maintainer applies the rule — the one failure mode not caught by a committed file (plan Risk). |

---

## Consistency Notes

- **Sibling boundary (022 Automated Release Pipeline):** 022's interface-spec names PR Validation (#24) as the gate where `.goreleaser.yaml` drift "fails … before a release is ever cut." This accord honors that: `go test ./...` runs the `internal/build` config-guard test, so the drift gate 022 depends on is exactly this workflow's `test` job. PR Validation is the upstream guarantee 022 assumes; it does not consume 022's output.
- **Sibling boundary (029 Main-Branch Verification):** 029 re-runs `go test ./...` on merge to `main`. The plain test invocation here is chosen so 029 can mirror it; PR Validation owns the pre-merge gate, 029 the post-merge net.
- **Sibling boundary (028 PR Administration / 030 Release Drafting):** PR Validation neither reads nor writes PR labels (028) and authors no release notes (030); its verdict depends only on code.
- **Convention — GitHub Actions + pinned action versions:** there is no `accords/` directory in this project; conventions are taken from PROJECT.md (Go CLI on GitHub) and the existing `release.yml` (022), which pins `actions/checkout@v4` and `actions/setup-go@v5` with `go-version-file: go.mod`. This accord reuses those exact choices for consistency and adds `golangci/golangci-lint-action@v6` (pinned linter version) — the same "pin the toolchain, don't float `latest`" discipline GoReleaser is pinned with.
- **Assumption — `.golangci.yml` linter set and golangci-lint version (`[ASSUMED]`):** a conservative starting set and a concrete pinned tag, both tunable in a follow-up without changing the contract (the lint job's *existence* and its three categories are fixed; the enabled-linter list is policy).
- **Assumption — branch-protection policy fields (`strict`, `enforce_admins`) (`[ASSUMED]`):** maintainer policy defaults; the load-bearing contract is only that `ci-success` is required and merge-blocking on `main`.
- **Assumption — `go test` flags:** the contract is plain `go test ./...`. Adding `-race` (valuable given the project's concurrency seams) or `-count=1` is an open, additive knob the implementation may adopt; it does not change the behavioral contract that every cell must pass.
- **CONSTITUTION XII note:** golangci-lint and the Go toolchain are CI-host tools, not artifact dependencies — XII governs the produced binary's runtime, the same standing this project gives GoReleaser and `sigs.k8s.io/yaml`.
