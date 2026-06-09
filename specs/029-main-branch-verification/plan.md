# Plan: Main-Branch Verification

**Feature**: 029-main-branch-verification
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (esp. the 024 entry that mirrors its `test` job onto `push: main` and explicitly leaves the shared-reusable-workflow call to 029, and 022's GitHub Actions precedent), LEARNINGS.md / DEPRECATION.md (no conflicting precedent), sibling 024's shipped `.github/workflows/ci.yml` + plan.md

---

## System Architecture

Main-Branch Verification ships **no application code** — like 022/024 it is CI infrastructure. It introduces no Go package, registers no cobra command, and adds no test: it re-runs the project's *existing* `go test ./...` suite on every push to `main`.

Its defining structural choice is that 024 and 029 must run the **same** test matrix so "green on `main`" carries the same meaning as "green on the PR" (spec's load-bearing contract). Rather than duplicate the matrix in two files and rely on humans keeping them in sync (the drift risk the spec names), the test matrix becomes a **single source of truth**: a reusable workflow that both the pre-merge gate (024) and the post-merge net (029) call.

```
.github/workflows/test.yml         ← NEW reusable workflow (on: workflow_call)
   strategy.matrix.os = {ubuntu-latest, macos-latest}
   setup-go (go-version-file: go.mod) → go test ./...        ── the shared suite
        ▲                                   ▲
        │ uses:                             │ uses:
why ci.yml (024, pre-merge)            main-verify.yml (029, NEW, post-merge)
  on: pull_request {branches:[main]}     on: push {branches:[main]}
  lint job (ONCE) + test (uses) →        test (uses)
  ci-success needs:[lint,test]           │   no aggregation job, no gate
  → branch protection requires           ▼
    ci-success (merge-blocking)         failing run + failing commit status on main
                                          (loud signal only — nothing blocks/reverts)
```

The post-merge workflow has **no enforcement layer**: unlike 024 there is no `ci-success` aggregation job and no branch-protection coupling, because the code is already on `main` and cannot be blocked. The whole observable outcome is the workflow run plus the commit status GitHub Actions attaches automatically (spec §Result; the developer chose "red run + commit status, nothing more"). Lint is *not* re-run here — it is OS-independent and 024 already enforced it pre-merge.

Tag pushes are excluded **structurally**: `on: push: { branches: [main] }` matches branch refs only, so a tag push (which the release pipeline 022 owns) never triggers this workflow — no `tags-ignore` needed.

---

## Architecture Decisions

### ADR-1: The shared test matrix becomes a reusable workflow (single source of truth) called by both `ci.yml` and `main-verify.yml`

**Context**: The spec requires 029's post-merge run to mirror 024's pre-merge test matrix so the two carry the same meaning, and flags matrix drift between the two as a risk. 024's plan/DECISIONS entry kept `go test ./...` a plain invocation specifically so 029 could reuse it, and explicitly left "whether via a shared reusable workflow" to 029.

**Options considered**:
1. **Extract the test-matrix job into a reusable workflow** (`.github/workflows/test.yml` with `on: workflow_call`) that both `ci.yml` (024) and a new `main-verify.yml` (029) invoke via `uses:`. The matrix and `go test ./...` invocation live in exactly one place, so parity is structural — a future matrix change applies to both gates at once, and "same suite, same environments" cannot silently drift.
2. **Duplicate the ~10 lines of test-job YAML** in a standalone `main-verify.yml`, leaving `ci.yml` untouched. Simpler and fully independent, but the two matrices must be hand-synced; a one-sided edit silently breaks the "same meaning" guarantee the spec rests on.

**Decision**: Option 1 (developer-confirmed). Add a reusable workflow holding the OS matrix + `setup-go` + `go test ./...`; refactor 024's `ci.yml` so its `test` job becomes a `uses: ./.github/workflows/test.yml` call; add `main-verify.yml` that calls the same reusable workflow. This makes the spec's matrix-parity contract a property of the code, not a convention. It is the "import the source of truth, don't mirror it with a comment" principle applied to CI.

**Consequences**: 029's PR touches a sibling's shipped artifact (`ci.yml`), but does not change *what* runs — the same matrix and the same `go test ./...`. 024's required check is unaffected: branch protection requires only the stable `ci-success` context (024 ADR-4), and `ci-success`'s `needs: [lint, test]` still aggregates the called workflow's result correctly. The per-cell check *names* nest under the reusable workflow (e.g. `test / test (ubuntu-latest)`), but nothing requires those by name, so the gate survives. The exact reusable-workflow filename, inputs, and job/step names are interface-level.

### ADR-2: A new `main-verify.yml` triggered by `push: { branches: [main] }`; tags excluded structurally

**Context**: The spec fixes the trigger as every change landing on `main` (a merge = a push to `main`), excluding tag pushes (022 owns tags). The provider is settled precedent (021/022/024 all run GitHub Actions).

**Options considered**:
1. **A dedicated `main-verify.yml` with `on: push: { branches: [main] }`** — a separate workflow whose sole job is the post-merge net, cleanly mirroring 024's "one workflow = one responsibility" shape (024 is the `pull_request` half; 029 is the `push` half).
2. **Add a `push: { branches: [main] }` trigger to the existing `ci.yml`** — one fewer file, but `ci.yml` is "the PR gate" with a lint job, a `ci-success` aggregation, and PR-ref concurrency; bolting a push trigger on would conditionally run/skip jobs by event and blur the pre-/post-merge boundary the specs deliberately separate.

**Decision**: Option 1. Add `.github/workflows/main-verify.yml` with `on: push: { branches: [main] }` and least-privilege `permissions: contents: read`. `branches: [main]` matches only the `main` branch ref, so tag pushes do not trigger it — the "tag push does not verify" edge is realized structurally, with no `tags-ignore` list. The workflow's single job calls the ADR-1 reusable workflow.

**Consequences**: A focused workflow that reads as "the post-merge net." It runs the test matrix only — no lint job, mirroring the spec's tests-only scope.

### ADR-3: No enforcement/aggregation layer — the failure surface is the run + commit status

**Context**: 024 builds a `ci-success` aggregation job and a branch-protection requirement because its verdict must *block a merge*. 029 runs after the merge, so there is nothing to block; the developer chose "red workflow run + failing commit status, nothing more" (no issue creation, no external notification).

**Options considered**:
1. **No aggregation job, no branch protection, no notification step** — rely on the workflow run's own pass/fail and the commit status GitHub Actions attaches automatically. A failing matrix cell turns the run red and marks the commit red in history; that *is* the loud signal.
2. **Add an aggregation job / open-an-issue / notify step** — extra machinery for a signal GitHub already surfaces, and beyond the scope the developer explicitly drew. Active alerting, if ever wanted, is a separate additive feature.

**Decision**: Option 1. 029 owns only running the matrix on `push: main`; the observable outcome is the run status and commit status. No `ci-success`-style job (there is no required-check name to stabilize because nothing requires it), no branch-protection change, no issue/notification.

**Consequences**: Minimal surface. A regression on `main` is visibly red in Actions and on the commit, and a human decides remediation — the spec's non-behaviors (no block, no revert, no auto-fix) hold by construction. If maintainers later want active alerting or issue-tracking, that is a clean future addition that does not disturb this workflow.

### ADR-4: Each `main` commit is verified independently — no `cancel-in-progress`

**Context**: 024 uses a `concurrency` group keyed on the PR ref with `cancel-in-progress: true`, because only the PR's *latest* head matters (a superseded run is worthless). 029's spec says the opposite: "each merge is verified against its own commit" — every commit that lands on `main` must get its own verdict.

**Options considered**:
1. **No cancelling** — either no `concurrency` group, or a group keyed on the commit SHA (so distinct commits never collide). Every merge commit runs to completion and gets its own status.
2. **Reuse 024's cancel-in-progress on a branch-level group** — would cancel the in-flight verification of commit A when commit B lands moments later, leaving A unverified and silently undermining the "each commit verified" contract.

**Decision**: Option 1. Do not cancel in-progress post-merge runs. This is the deliberate inverse of 024's latest-commit semantics: pre-merge, only the tip matters; post-merge, every commit matters. The exact mechanism (omit `concurrency` entirely vs. a SHA-keyed group with `cancel-in-progress: false`) is interface-level; the contract is "no merge commit's verification is cancelled by a later one."

**Consequences**: A burst of rapid merges produces one verification run per commit — more runner minutes than a cancel-latest scheme, accepted because pinpointing *which* commit regressed is the whole point of a post-merge net.

---

## Cross-cutting Concerns

- **Shared suite / single source of truth (relationship to 024)** — ADR-1's reusable workflow is the one definition of the test matrix; 024 and 029 both call it, so they cannot diverge. Lint stays in `ci.yml` only (not extracted) — 029 runs tests only.
- **024's required check is preserved** — branch protection requires only `ci-success`; refactoring `test` into a `uses:` call keeps `needs: [lint, test]` valid and the cell-name nesting irrelevant to the rule. The 029 PR must not alter the `ci-success` contract.
- **Permissions & secrets** — `permissions: contents: read` only, no secrets; the net builds and tests, it never writes. (The commit status GitHub attaches to a workflow run needs no extra token grant.)
- **Failure legibility** — `fail-fast: false` in the shared matrix (inherited from 024) lets each OS cell report independently, so a post-merge failure names the environment that broke (spec §Tests, error scenarios).
- **Testing strategy** — a workflow is hard to unit-test; confidence comes from (a) the existing `go test ./...` suite (unit + godog + `internal/build`) being what runs, unchanged; (b) the reusable-workflow refactor being exercised by 024's own PR gate on the 029 branch (the refactored `ci.yml` runs on the PR that introduces it); and (c) a one-time observation that `main-verify.yml` runs green on the first merge after it lands. No new Go tests are added.

---

## Implementation Strategy

Small CI infrastructure; phases are sequencing aids and may collapse into one PR at the tasks stage.

**Phase 1 — Extract the reusable test workflow.** Add `.github/workflows/test.yml` (`on: workflow_call`) holding the OS matrix (`{ubuntu-latest, macos-latest}`, `fail-fast: false`), `actions/setup-go` (`go-version-file: go.mod`), and `go test ./...`. Refactor `ci.yml`'s `test` job to `uses: ./.github/workflows/test.yml`; confirm `ci-success`'s `needs: [lint, test]` still resolves and the PR gate stays green (this PR is its own proof).

**Phase 2 — Add the post-merge workflow.** Add `.github/workflows/main-verify.yml` with `on: push: { branches: [main] }`, `permissions: contents: read`, no cancelling concurrency, and a single job that calls the reusable workflow. Verify it triggers on merge to `main` and not on tag pushes.

Phases are ordered 1 → 2 (Phase 2 depends on the reusable workflow from Phase 1).

---

## Risks

- **Refactoring 024's shipped `ci.yml` disturbs the required gate** — extracting `test` into a reusable workflow changes job nesting. *Low–medium.* Mitigation: branch protection requires only the stable `ci-success` name (024 ADR-4), and `needs: [lint, test]` aggregates the called workflow's result; the 029 PR runs through 024's own gate, so a break shows up before merge.
- **macOS runner minutes on every merge** — the post-merge net runs the OS matrix on each commit to `main`, and macOS minutes bill at a higher multiplier; ADR-4 deliberately does not cancel. *Low–medium.* Mitigation: two cells, amd64, single Go version; merges to `main` are far less frequent than PR pushes, so volume is bounded.
- **Reusable-workflow check-name nesting confuses observers** — cell checks appear as `test / test (ubuntu-latest)`. *Low.* Mitigation: documented in ADR-1; affects display only, not the gate.
- **A flaky test that passed on the PR reddens `main`** — the net surfaces flakes that slipped the gate. *Low — by design.* That is exactly what 029 exists to catch; the outcome is a visible red status, not a block or revert.

---

## What This Plan Does Not Cover

- **Protocol/structural contracts** — the exact YAML (reusable-workflow filename and `workflow_call` inputs, runner labels, job/step names, the `main-verify.yml` `concurrency` expression or its omission, action versions) is the interface skill's concern (`/score:interface`).
- **024's lint job and `.golangci.yml`** — owned by 024; 029 does not run lint and does not touch the linter config.
- **Branch protection / required checks** — 024 owns the only required-check (`ci-success`); 029 adds no enforcement and changes no protection rule.
- **Automated Release Pipeline (022)** — owns tag/release-triggered build-package-attach; 029 verifies `main` commits and is structurally disjoint (tags excluded).
- **The test suite's contents** — owned by the specs that produced each package and its tests; 029 runs the existing `go test ./...`, it adds and modifies nothing.
- **Active alerting / issue tracking on failure** — out of scope per ADR-3; a clean future addition if maintainers want it.
