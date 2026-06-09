# Tasks: PR Administration

**Feature**: 028-pr-administration
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-automated-pipeline/pr-administration.feature

---

> **No cross-spec gate**: PR Administration has no upstream dependencies (backlog: "no dependencies"). It adds no Go package and no test — three committed declarative artifacts. All tasks are ready to implement now. The three tasks each create/extend a **different** file (`.github/settings.yml`, `.github/labeler.yml`, `.github/workflows/pr-administration.yml`), so they have no file conflicts, but they share a contract dependency on the seven label **names** fixed in Phase 1 — so they form a logical chain and may reasonably collapse into one PR (the plan notes the phases "may collapse"). They are listed separately to mirror the plan's phase structure.
>
> **Prerequisite**: the label catalog (T001) is reconciled by the Probot "Settings" GitHub App — a one-time repo/org install (already established for the team via ailign). Document it alongside the feature.
>
> **Downstream contract**: the seven label names defined here are the contract Release Drafting (030) consumes. **Self-trigger caveat**: under `pull_request_target`, GitHub runs the workflow definition from the **base** branch — so the PR that introduces `pr-administration.yml` does **not** label itself; labelling begins on the next PR after merge to `main`.

## Dependency Graph

Phase 1: Label Catalog (1 task, no dependencies) [Shared]
Phase 2: Signal Mapping (1 task, depends on Phase 1) [Shared]
Phase 3: Labelling Workflow (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (shared name-contract chain) | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/028-pr-administration/base` → `spec/028-pr-administration/task-1`, `task-2`, `task-3`.

The `base` branch is cut from current `main`. The three tasks touch different files, so they can be implemented on the same base and merged together or as one combined PR; keep the label names identical across all three. Other pipeline specs (025, 026, 029, 030, …) may build in parallel on their own base branches; this feature hard-depends on none of them. Because `pull_request_target` runs the **base-branch** workflow, the introducing PR will not exercise the new workflow on itself — verify on a follow-up PR (or a fork PR) after merge.

---

## Phase 1: Label Catalog [Shared]

- [ ] **T001** [Shared] Create `.github/settings.yml` with a `labels:` block defining the seven managed PR labels
  - **Scope**: Add `.github/settings.yml` (Probot "Settings" app schema) with a `labels:` block defining exactly the seven managed labels — `breaking`, `features`, `fixes`, `docs`, `infrastructure`, `dependencies`, `internal` — each with `name`/`color`/`description` per interface-spec.md's catalog table (colors `[ASSUMED]`, tunable). Keep it **labels-only** (no `repository:`/`branches:` blocks) so the app manages only these labels and leaves human triage labels in place (no prune). Add a header comment that the names are the contract Release Drafting (030) reads. This is the single source of truth for the label strings — do **not** create label definitions from the workflow (spec §Non-Behaviors: PR Administration does not own the label catalog). Document the one-time prerequisite that the Settings app must be installed on the repo/org.
  - **Acceptance criteria**:
    - `.github/settings.yml` defines all seven labels with the intended names, colors, and descriptions, in valid Probot Settings-app YAML; the file is labels-only.
    - The seven names are exactly those consumed by `.github/labeler.yml` (T002) and by Release Drafting (030) — no extras, none missing.
    - Once the Settings app is installed, a push to the default branch reconciles the seven labels (and a hand-deleted/renamed managed label is restored); labels not listed are left untouched.
    - YAML lints clean (e.g. `yamllint`/`actionlint`-adjacent check); the prerequisite app install is documented.
  - **Dependencies**: none
  - **Plan reference**: Phase 1 (Label catalog); ADR-4 (`settings.yml` via Probot Settings app)
  - **Scenario references**: pr-administration.feature — precondition for all labelling scenarios (the labels must exist); no direct behavioral scenario of its own.
  - **Interface references**: interface-spec.md: "Label catalog contract (`.github/settings.yml` `labels:` block)" and the Invocation table.

## Phase 2: Signal Mapping [Shared]

- [ ] **T002** [Shared] Create `.github/labeler.yml` mapping title/branch + changed-file signals to the seven labels, with authoritative sync
  - **Scope**: Add `.github/labeler.yml` (`srvaroa/labeler` `version: 1`) with one entry per managed label, each carrying its `title` (conventional-commit) and/or `branch` and/or `files` matchers per interface-spec.md's mapping table: `breaking` (`!`-marked type / `major|breaking/` branch), `features` (`feat:` / `feat|feature/`), `fixes` (`fix:` / `fix|bug|hotfix/`), `docs` (`docs:` / `*.md`, `docs/`), `infrastructure` (`ci|build:` / `.github/workflows/`, `.goreleaser.yaml`, `.golangci.yml`, `scripts/`, `Makefile`, `Dockerfile*`, pre-commit), `dependencies` (`*(deps)` / `dependabot/` branch / `go.mod`, `go.sum`), `internal` (`chore|refactor|perf|test|style:` / `chore|refactor|cleanup/`). Rely on (or, if the pinned version requires it, explicitly enable) the per-label evaluate→add-if-match / remove-if-not-match behavior so stale managed labels are removed (B1 sync) — scoped to labels named in this file only. Regexes are `[ASSUMED]` and tunable; the names must match T001 exactly.
  - **Acceptance criteria**:
    - A `feat:`-titled PR maps to `features`; a docs-only diff maps to `docs`; a PR matching several signals carries all matching labels (e.g. `features` + `dependencies`); a PR matching no signal gets no managed label.
    - When a previously-matching managed label no longer matches (e.g. title edited `feat:`→`fix:`), it is removed and the now-correct one applied; labels **not** named in this file are never added or removed.
    - The config validates against the pinned action's schema (no syntax errors); label names are exactly the seven from T001.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 (Signal mapping); ADR-2 (`srvaroa/labeler` + committed config), ADR-3 (seven-category family, semver-bearing)
  - **Scenario references**: pr-administration.feature: "A feature pull request is labelled from its title"; "A docs-only change is labelled from its changed files"; "A pull request carries every matching label"; "An unrecognized change is not mislabelled"; "Editing the title reconciles the labels"; "Reconciliation never touches labels outside the managed set"
  - **Interface references**: interface-spec.md: "`.github/labeler.yml` structure" (mapping table + example)

## Phase 3: Labelling Workflow [Shared]

- [ ] **T003** [Shared] Create `.github/workflows/pr-administration.yml` on `pull_request_target` running the pinned labeler, fork-safe and non-blocking
  - **Scope**: Add `.github/workflows/pr-administration.yml` with `name: PR Administration`; `on: pull_request_target: { types: [opened, reopened, synchronize, edited] }` (no `branches:` filter); least-privilege `permissions: { contents: read, pull-requests: write }`; a `concurrency` block keyed on `${{ github.workflow }}-${{ github.event.pull_request.number }}` (the same group shape as 024's `ci.yml`, so the workflow name namespaces the group and PRs can't cross-cancel between workflows) with `cancel-in-progress: true`; a single `label` job (`runs-on: ubuntu-latest`) whose only step runs `srvaroa/labeler` pinned to a concrete tag (`[ASSUMED]` `v1.14.0` — pin at impl time) with `GITHUB_TOKEN: ${{ github.token }}` and `continue-on-error: true`. **No `actions/checkout`** and no PR-supplied script (ADR-5 — the labeller reads PR title, head branch, and changed-file list via the API, and loads the config from the base branch). Do **not** add this workflow to branch protection (spec §Non-Blocking). Remove `@wip` from the behavioral scenarios in `pr-administration.feature` as they pass; leave the three `@validation` scenarios for `/score:validate`.
  - **Acceptance criteria**:
    - The workflow triggers on open/reopen/synchronize/edit of any PR including fork PRs, and applies labels per `.github/labeler.yml`; a fork PR is labelled without checking out or executing PR head code.
    - Permissions are exactly `contents: read` + `pull-requests: write`; no secrets are referenced; there is no checkout step.
    - A new edit/push cancels the superseded in-flight run (concurrency group), so labels reflect the latest PR state; rapid successive title edits resolve to the latest title.
    - A labelling-step failure does not block or redden the merge (the workflow is not a required check; `continue-on-error` keeps a flake from surfacing as a blocking mark).
    - `actionlint` is clean.
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 (Workflow); ADR-1 (`pull_request_target`, separate from `ci.yml`), ADR-5 (no checkout)
  - **Scenario references**: pr-administration.feature: "A fork pull request is labelled on the same terms"; "Fork labelling reads metadata only"; "A labelling failure does not block the merge"; "Labelling is not a required check"; "Rapid successive title edits reconcile to the latest title" — and end-to-end, every T002 scenario runs through this workflow.
  - **Interface references**: interface-spec.md: "`.github/workflows/pr-administration.yml` structure" (YAML block); Interactions (fork safety, re-reconcile); Error Communication
  - **Risk**: ⚠️ `pull_request_target` carries a writable token — the no-checkout invariant (ADR-5) is load-bearing; any future step that checks out PR head code is a security change. ⚠️ The seven label names are an un-guarded cross-feature contract until 030 ships (plan Risk) — keep them stable.
