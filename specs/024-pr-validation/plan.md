# Plan: PR Validation

**Feature**: 024-pr-validation
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (28 entries — esp. 021/022's GitHub Actions + GoReleaser precedent, the `internal/build` config-guard, and the `go test ./...` suite shape), LEARNINGS.md / DEPRECATION.md (no conflicting precedent for CI), sibling 022's plan.md + interface-spec.md (the release workflow that names PR Validation as its drift gate)

---

## System Architecture

PR Validation ships **no application code** — like 022, it is CI infrastructure: a single new declarative artifact (`.github/workflows/ci.yml`) plus a linter config (`.golangci.yml`) at the repo root, running the project's *existing* Go test suite and lint tooling on every pull request that targets `main`. It introduces no Go package and registers no cobra command.

The workflow has three logical pieces, mapping one-to-one onto the spec's Behavioral Accord:

```
pull_request targeting main  (opened | synchronize/push | reopened)
        │   concurrency group keyed on the PR ref, cancel-in-progress
        ▼
[lint job]   (runs ONCE, ubuntu-latest)              ── spec §Lint
   gofmt-check  →  go vet ./...  →  golangci-lint run   (.golangci.yml)
        │   any problem ⇒ job fails, naming what failed
        │
[test job]   (matrix: OS = {ubuntu-latest, macos-latest})  ── spec §Tests
   setup-go (go-version-file: go.mod)  →  go test ./...
   (unit + godog BDD suites + internal/build host self-containment
    + the .goreleaser.yaml config-guard test 022 relies on)
        │   every matrix cell must pass; a failing cell names its OS
        ▼
[ci-success]  needs: [lint, test]                    ── spec §Result and gate
   the SINGLE stable status check; green iff lint + every test cell passed
        │
        ▼
branch protection on main requires "ci-success" ⇒ merge blocked until green
        (fail-closed: a missing/failed ci-success blocks the merge)
```

The **test job is the gate spec 022 already depends on**: 022's interface-spec states that `.goreleaser.yaml` drift ("lost target, `CGO_ENABLED=1`, missing `archives`/`checksum`/`release`") "fails in PR Validation (#24) before a release is ever cut." `go test ./...` runs the `internal/build` `CheckConfigGuard` test plus all godog suites and unit tests, so no test logic is invented here — PR Validation orchestrates the suite the repo already has. **Main-Branch Verification (029)** later runs the same `go test ./...` on push to `main`; PR Validation is the pre-merge half, 029 the post-merge net.

The build/CI host needs the Go toolchain and golangci-lint; **neither is a dependency of the produced binary** — CONSTITUTION XII governs the artifact's runtime, not the CI host (021's load-bearing distinction, carried forward, exactly as 022 treats GoReleaser).

---

## Architecture Decisions

### ADR-1: A new `.github/workflows/ci.yml` on GitHub Actions, triggered by `pull_request` against `main`

**Context**: The spec fixes the trigger (PRs targeting `main`, on open / update / reopen; not other base branches, not pushes to `main`, not release publishes). The repo already runs GitHub Actions (021/022's `release.yml`), so the provider is settled precedent; the open decision is the workflow's trigger surface and that it is a *separate* workflow from `release.yml`.

**Options considered**:
1. **A dedicated `ci.yml` with `on: pull_request: { branches: [main] }`** — one workflow whose sole job is the pre-merge gate. GitHub's `pull_request` event defaults to the `opened`, `synchronize`, and `reopened` activity types — exactly the spec's three triggers — and `branches: [main]` scopes it to PRs whose base is `main`. Clean separation from release concerns.
2. **Fold PR checks into the existing `release.yml`** — one file, but `release.yml` triggers on `release: published`; merging two unrelated triggers/permission sets into one workflow muddies both and couples the pre-merge gate to release machinery.

**Decision**: Option 1. Add `.github/workflows/ci.yml` with `on: pull_request: { branches: [main] }` and least-privilege `permissions: contents: read`. The default activity types (`opened`/`synchronize`/`reopened`) satisfy the spec's three triggers with no explicit `types:` list needed; a PR whose base is not `main` does not match `branches: [main]` and so does not run — realizing the "non-main base does not trigger" edge structurally. Fork PRs get a read-only token, which is all the gate needs (it builds and tests, it does not write).

**Consequences**: A focused workflow that reads as "the PR gate." The exact action versions (`actions/checkout`, `actions/setup-go`, `golangci-lint-action`), runner labels, and job names are interface-level. Adding `synchronize` coverage is automatic, so the "push a fix re-runs against the new head" behavior needs no extra wiring.

### ADR-2: Lint is a single job — gofmt-check + `go vet` + golangci-lint — run once, not per matrix cell

**Context**: The spec's accord defines "lint" as a formatting check + `go vet` + a configured aggregate linter, and explicitly says the lint pass runs **once** per validation run (not repeated per test-matrix cell). golangci-lint is a new CI dev tool; the repo has no linter config yet.

**Options considered**:
1. **A standalone `lint` job running gofmt-check, `go vet ./...`, then `golangci-lint run`** — golangci-lint already bundles `gofmt` and `govet` as sub-linters, but running gofmt-check and `go vet` explicitly keeps each failure category legible and gives a fast, dependency-light signal even if golangci-lint setup changes. One job, one run.
2. **Fold lint into each test matrix cell** — would run lint N times across OSes, contradicting the spec's "lint runs once" and wasting CI minutes for an OS-independent check.

**Decision**: Option 1. A single `lint` job on `ubuntu-latest`: a gofmt check (fail if `gofmt -l` lists any file), `go vet ./...`, then `golangci-lint run` driven by a committed `.golangci.yml`. golangci-lint is pinned to a specific version via its official action (no floating "latest"). The linter is a CI-host tool, not an artifact dependency, so it does not touch CONSTITUTION XII — the same standing this project gives GoReleaser and `sigs.k8s.io/yaml`.

**Consequences**: One authoritative lint signal feeding the gate. The exact linter set enabled in `.golangci.yml` and the pinned golangci-lint version are interface-level (pinning prevents a new upstream linter from spontaneously reddening unrelated PRs — see Risks). Running gofmt/vet explicitly alongside golangci-lint is mild redundancy accepted for legibility and resilience.

### ADR-3: Tests run on an OS matrix (ubuntu + macos); Go version tracks go.mod; arch breadth stays at release time

**Context**: The spec requires the test suite to run across a matrix of multiple Go versions and/or OSes, every cell required to pass, and leaves the exact axes as a plan detail. The project ships on macOS and Linux (021's four targets) and has genuinely OS-divergent code: `internal/build` inspects linkage with `otool` on darwin vs `ldd` on linux, and `golang.org/x/term` has per-OS paths — so OS coverage catches real bugs.

**Options considered**:
1. **OS matrix `{ubuntu-latest, macos-latest}`, single Go version from `go.mod`, amd64 only** — covers both shipped OS families (exercising the `otool`/`ldd` and TTY branches) at two cells; cheap and fast per PR. Go version is pinned via `setup-go`'s `go-version-file: go.mod` so CI and the module agree by construction.
2. **OS × Go-version × arch matrix** — maximal coverage, but arm64 runners and extra Go versions multiply per-PR cost and minutes; arch-specific surprises are already caught at release time by 022's four-runner cross-target self-containment gate, making per-PR arch coverage low marginal value.

**Decision**: Option 1. The `test` job declares `strategy.matrix.os: [ubuntu-latest, macos-latest]` and runs `go test ./...` on each, with `setup-go` reading the version from `go.mod`. This satisfies the spec's "multiple ... OSes" with the two families the tool actually ships to. A Go-version axis is supported and left as an explicit, easily-extended knob (`[ASSUMED]` default: the single go.mod-pinned version) so per-PR cost stays bounded; cross-**arch** verification deliberately stays at release time (022), not per PR. The behavioral contract — every configured cell must pass — holds regardless of the exact axes.

**Consequences**: Two cells catch the OS-divergent paths that matter most for this codebase. macOS runner minutes cost more than Linux (accepted, two cells only). If a future need arises (a Go upgrade window, an arch-specific report), the matrix extends by adding axis values without restructuring — and the `ci-success` gate (ADR-4) keeps the required check name stable when it does.

### ADR-4: A single `ci-success` aggregation job is the one required status check

**Context**: The spec makes the validation result a **required gate**. A matrix `test` job publishes one status check *per cell* (e.g. `test (ubuntu-latest)`, `test (macos-latest)`), and those context names change whenever the matrix changes. GitHub branch protection requires checks **by name**, so requiring matrix-cell checks directly is brittle: edit the matrix and the required-check names silently drift, leaving the gate misconfigured.

**Options considered**:
1. **A terminal `ci-success` job with `needs: [lint, test]`** — a tiny job that exists only to succeed when its dependencies all succeeded (and to fail/!skip when any did not). Branch protection requires the single stable context `ci-success`; the matrix can grow or shrink underneath it without touching the protection rule. Mirrors 022's `publish needs: [build, verify]` gating shape.
2. **Require each matrix-cell check directly in branch protection** — no extra job, but the required-check list must be hand-edited in lockstep with every matrix change, and a forgotten edit yields a gate that passes while a cell is unconfigured.

**Decision**: Option 1. Add a `ci-success` job with `needs: [lint, test]`. Because a skipped/failed dependency must **block** (not pass-by-skip), the job asserts success explicitly rather than relying on default `needs` semantics — the exact guard (e.g. checking `needs.*.result`, or a community "all-checks-passed" gate) is interface-level, but the contract is: `ci-success` is green **iff** lint and every test cell passed. Branch protection requires only `ci-success`.

**Consequences**: One stable required-status name decouples the protection rule from the matrix shape — the gate survives matrix edits, directly serving the spec's "the gate reflects the latest result across all cells." The cost is one trivial extra job per run. This is the structural realization of "report + enforce": the aggregation job is what the enforcement (ADR-5) points at.

### ADR-5: Enforcement is GitHub branch protection requiring `ci-success`; the capability owns providing the stable check + documenting/scripting the rule

**Context**: The developer chose "report **+ enforce**" — PR Validation owns making the check a required, merge-blocking gate, not an advisory signal. But GitHub branch protection / rulesets are **repository settings**, not a file in the repo that GitHub auto-applies (unlike the workflow YAML). So "the capability owns the gate" cannot mean "commit a file that turns it on."

**Options considered**:
1. **Own the check + document and script the branch-protection rule** — the workflow guarantees a single stable status context (`ci-success`, ADR-4); the capability additionally provides the operator step to mark it required on `main` (a documented `gh api`/repo-settings procedure, runnable once by a maintainer with admin rights). The repo provides everything that *can* live in-repo; the one-time settings act is explicit and scripted, not hand-wavy.
2. **Treat enforcement as wholly out of scope (advisory only)** — simplest, but contradicts the developer's explicit "enforce" choice; an advisory check that maintainers can ignore is exactly the "merely reports" outcome the spec rejects.

**Decision**: Option 1. PR Validation owns: (a) emitting the stable `ci-success` status, and (b) the branch-protection requirement that makes it merge-blocking on `main`, captured as a documented and scripted maintainer step that adds `ci-success` through the **narrow** required-status-checks sub-resource (`gh api --method PATCH …/branches/main/protection/required_status_checks`) or a repository ruleset — **not** the full-document `PUT …/protection`, which would replace the whole config and clobber existing review/restriction settings. Fail-closed follows from GitHub's required-check semantics: a missing or failed `ci-success` blocks the merge. The exact API payload / ruleset JSON is interface-level.

**Consequences**: The "enforce" half is real and reproducible, not tribal knowledge. Because branch protection lives in repo settings, it is the one part of this capability not enforced by a committed file — so it carries a drift risk (Risks) mitigated by scripting it and by the stable `ci-success` name. A maintainer with admin rights must apply it once; CI cannot self-grant branch protection from a read-only PR token.

---

## Cross-cutting Concerns

- **Atomicity / gating** — gating is structural (the 022 pattern): `ci-success` depends on `lint` + the whole `test` matrix, so any failure simply never lets `ci-success` go green, and branch protection blocks the merge. No partial-pass path exists.
- **Latest-commit semantics & superseded runs** — a `concurrency` group keyed on the PR ref with `cancel-in-progress: true` cancels an in-flight run when a new commit is pushed, so the gate reflects the **latest** head commit and a superseded run cannot satisfy it — directly realizing the spec's rapid-successive-push edge.
- **Permissions & secrets** — `permissions: contents: read` only; no secrets. The gate builds and tests, it never writes, so fork PRs run under the default read-only token without exception.
- **Configuration** — the trigger, the OS matrix, the pinned golangci-lint version, and `.golangci.yml`'s linter set are the only knobs; all declarative. No application configuration.
- **Relationship to 029 (shared suite)** — both 024 and 029 run `go test ./...`; keeping the test invocation a plain `go test ./...` (no bespoke flags) lets 029 mirror it directly. Whether the two share a reusable workflow / composite action is 029's decision — not pre-built here (YAGNI; 029 owns that call).
- **Testing strategy** — a workflow is hard to unit-test; confidence comes from (a) the workflow running on its **own** introducing pull request (the gate validates itself before merge), (b) the `internal/build` config-guard + self-containment tests and all godog suites already in `go test ./...`, and (c) a local `golangci-lint run` + `go vet ./...` + gofmt check reproducing the lint job before push. No new Go tests are added by this capability.

---

## Implementation Strategy

This is small CI infrastructure; the phases are sequencing aids and may collapse into one or two PRs at the tasks stage.

**Phase 1 — Lint config + lint job.** Add `.golangci.yml` (enabled linter set, `[ASSUMED]` starting set) and the `lint` job in a new `.github/workflows/ci.yml` (`on: pull_request: { branches: [main] }`, `permissions: contents: read`): gofmt-check, `go vet ./...`, `golangci-lint run` via the pinned official action. Verify locally with `golangci-lint run` + `go vet ./...` + `gofmt -l`.

**Phase 2 — Test matrix job.** Add the `test` job to `ci.yml`: `strategy.matrix.os: [ubuntu-latest, macos-latest]`, `actions/setup-go` with `go-version-file: go.mod`, `go test ./...`. Add the `concurrency` block (PR-ref group, `cancel-in-progress`). Confirm the suite (unit + godog + `internal/build`) is green on both OSes.

**Phase 3 — Gate + enforcement.** Add the `ci-success` job (`needs: [lint, test]`, explicit success assertion). Document and script the branch-protection rule on `main` requiring the `ci-success` context (the maintainer-run `gh api`/ruleset step).

Phases are ordered 1 → 2 → 3.

---

## Risks

- **golangci-lint version drift reddening unrelated PRs** — a floating linter version can introduce a new check overnight and fail PRs that touched nothing related. *Medium.* Mitigation: pin golangci-lint to an explicit version in the action and a committed `.golangci.yml`; bumping the version is then a deliberate, reviewable PR (which itself runs through the gate).
- **Branch protection is repo settings, not an in-repo file (drift / not-applied)** — the "enforce" half lives outside version control, so it can be forgotten on setup or silently changed. *Medium.* Mitigation: ADR-4's stable `ci-success` name + ADR-5's documented, scripted `gh api` step; the gate's "report" half still runs and is visible even if the "required" flag lapses, surfacing the misconfiguration.
- **macOS runner minutes / cost** — the OS matrix doubles per-PR test runs and macOS minutes are billed at a higher multiplier. *Low–medium.* Mitigation: two cells only, amd64 only, single Go version, `cancel-in-progress` to kill superseded runs; arch breadth stays at release time (022).
- **`internal/build` self-containment test behavior in CI** — `TestSelfContainment_HostBinary` builds a host binary (go-build fallback) and inspects linkage with `otool`/`ldd`; a runner missing the expected toolchain layout could misfire. *Low.* Mitigation: GitHub-hosted ubuntu/macos runners ship `ldd`/`otool` and the Go toolchain; the test already runs under plain `go test` locally.
- **Fork-PR token limitations** — a contributor's fork PR runs with a read-only token. *Low (by design).* The gate needs only `contents: read`; no step requires write, so fork PRs validate identically to branch PRs.

---

## What This Plan Does Not Cover

- **Protocol/structural contracts** — the exact `ci.yml` YAML (action versions, runner labels, job/step names, the `ci-success` success-assertion expression, the `concurrency` group expression), the `.golangci.yml` linter set and pinned golangci-lint version, and the `gh api` branch-protection payload / ruleset JSON are the interface skill's concern (`/score:interface`).
- **Main-Branch Verification (029)** — the post-merge `go test ./...` workflow on push to `main`, and any decision to share a reusable workflow with 024, live in 029's spec.
- **PR Administration (028) / Release Drafting (030)** — label application and release-note/draft maintenance are separate capabilities; PR Validation neither reads nor writes labels and authors no notes.
- **Automated Release Pipeline (022)** — already landed/shaped; 024 is the upstream guarantee 022 assumes, not a consumer of it.
- **The test suite's contents** — owned by the specs that produced each package and its tests; 024 runs `go test ./...`, it does not add or modify tests.
