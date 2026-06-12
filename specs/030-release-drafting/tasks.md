# Tasks: Release Drafting

**Feature**: 030-release-drafting
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-automated-pipeline/release-drafting.feature

---

> **Cross-spec coordination (028)**: Phase 1 extends PR Administration's managed label set from **seven to eight** by adding the `no-release-note` exclusion label to `.github/settings.yml` and `.github/labeler.yml`. This is an announced divergence from 028's recorded "EXACTLY seven managed labels" decision (DECISIONS, candidate for `/score:deprecate`); the Phase 2 config guard re-pins the new eight-label invariant. 028 has shipped, so both files exist to edit now.
>
> **Prerequisite**: the new label is reconciled by the same Probot "Settings" GitHub App 028 already relies on (one-time repo/org install, established for the team) — no new prerequisite.
>
> **No app code**: like 028/022/024/029, this feature ships **no runtime CLI code**. Its only Go is the `internal/build` config-drift guard (T004), which runs in the existing `go test ./...` suite. The behavioral scenarios are verified by inspection + the guard; there is no local harness for the GitHub-hosted draft.
>
> **Self-trigger note**: `release-drafting.yml` runs on `push: main`, so the merge that introduces it **does** exercise it (unlike 028's `pull_request_target` base-branch caveat) — the first draft appears on that merge. The seam to 022 is unchanged: the draft is published by hand, and publishing triggers `release.yml`.

## Dependency Graph

Phase 1: Label Contract Extension (1 task, no dependencies) [US2]
Phase 2: Drafting Config, Workflow, and Guard (3 tasks, depends on Phase 1) [Shared]

4 tasks total | 0 phases parallelizable (Phase 2 depends on Phase 1's label) | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/030-release-drafting/base` (cut from current `main`) → `spec/030-release-drafting/task-1` … `task-4`.

Phase 1 (T001) lands the label contract first because Phase 2's `exclude-labels` reference and the config guard both depend on the `no-release-note` name existing. Within Phase 2, T003 (workflow) and T004 (guard) touch different files and can run in parallel once T002 (config) lands. The four tasks may reasonably collapse into one or two PRs (the label + its config consumers are a single logical contract), but are listed separately to mirror the plan's phase structure. Other pipeline specs build on their own base branches; this feature hard-depends only on 028 (already merged).

---

## Phase 1: Label Contract Extension [US2]

- [x] **T001** [US2] Add the `no-release-note` exclusion label to `.github/settings.yml` and `.github/labeler.yml` (eighth managed label; negate-over-noteworthy matcher) — 2 scenarios (auto half), header comments updated seven→eight; noted maintainer-flagged-vs-sync-removal tension in LEARNINGS.md
  - **Scope**: Two coordinated edits, one contract. (1) In `.github/settings.yml`, add an eighth entry to the `labels:` block — `name: no-release-note`, a color (`[ASSUMED]`, tunable), and a description (e.g. "Excluded from release notes (spec/feature-only or maintainer-flagged)"). (2) In `.github/labeler.yml`, add an eighth label entry with `negate: true` and a `files:` list of the **noteworthy** path patterns (`.*\.go$`, `go\.mod$`/`go\.sum$`, `docs/.*`, `README\.md$`, `\.github/.*`, `\.goreleaser\.ya?ml$`, `\.golangci\.ya?ml$`, `scripts/.*`, `Makefile`, `Dockerfile.*`, `\.pre-commit-config\.yaml`, `spec/.*`) so the label is applied **iff no changed file is noteworthy** — i.e. the PR is confined to `specs/` and `.feature` files. **Do not** use a broad `.*\.md$` pattern: Score spec markdown lives under `specs/` and must remain non-noteworthy. Leave the existing seven labeler entries untouched. Update the in-file header comments in both files that assert "seven managed labels" to reflect eight.
  - **Acceptance criteria**:
    - `.github/settings.yml` defines `no-release-note` as an eighth labels-only entry; valid Probot Settings-app YAML; the existing seven entries are unchanged.
    - `.github/labeler.yml` applies `no-release-note` to a PR whose changed files are confined to `specs/` and/or `.feature`, and does **not** apply it to a PR that touches any `.go`, `docs/`, `README.md`, infra, deps, or `spec/` path; the `negate` block carries only `files:` (so the inversion is over file-matching only).
    - The label name is identical across both files (and will match the Phase 2 `exclude-labels` and config guard).
    - Once reconciled by the Settings app, the label exists in the repo; YAML lints clean.
  - **Dependencies**: None (028's `settings.yml`/`labeler.yml` already exist)
  - **Plan reference**: Phase 1 (Label contract extension); ADR-4 (eighth exclusion label via negate-over-noteworthy)
  - **Scenario references**: release-drafting.feature: "A pull request confined to spec and feature files is omitted"; "An excluded pull request affects neither notes nor version" (the explicit-label half is maintainer-applied; this task realizes the auto spec/feature-only half)
  - **Interface references**: interface-spec.md: "Eighth managed label — `no-release-note`"

## Phase 2: Drafting Config, Workflow, and Guard [Shared]

- [x] **T002** [Shared] Add `.github/release-drafter.yml` — version-resolver (highest-wins, default patch), seven categories, exclude-labels, templates — 4 scenarios, config matches interface-spec verbatim
  - **Scope**: Create `.github/release-drafter.yml` per interface-spec.md: `tag-template`/`name-template` = `v$RESOLVED_VERSION`; `version-template` = `$MAJOR.$MINOR.$PATCH`; `version-resolver` mapping `major.labels: [breaking]`, `minor.labels: [features]`, `patch.labels: [fixes]`, `default: patch`; a `categories` block mapping the seven 028 labels to titled sections (titles `[ASSUMED]`, tunable; label strings fixed); `exclude-labels: [no-release-note]`; `change-template: "- $TITLE (#$NUMBER) @$AUTHOR"`; a `template` with `$CHANGES`. Leave `prerelease` at the default (status is set by T003's post-step). Do not invent category labels — reference the exact 028 strings.
  - **Acceptance criteria**:
    - The seven `categories` labels are exactly `breaking`/`features`/`fixes`/`docs`/`infrastructure`/`dependencies`/`internal`; `version-resolver` buckets are exactly breaking→major / features→minor / fixes→patch with `default: patch`.
    - `exclude-labels` contains `no-release-note` (the T001 label).
    - Tag/name/version templates produce a `v`-prefixed semver; YAML validates against release-drafter's schema.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-2 (version-resolver), ADR-3 (seven categories), ADR-4 (exclude-labels)
  - **Scenario references**: release-drafting.feature: "A feature merge bumps the draft to the next minor"; "The highest semver label wins across several merges"; "A change with no semver label takes the default patch bump"; "An excluded pull request affects neither notes nor version"
  - **Interface references**: interface-spec.md: "`.github/release-drafter.yml` structure"

- [x] **T003** [Shared] [P] Add `.github/workflows/release-drafting.yml` — push:main, least-privilege, pinned release-drafter, auto pre-release/latest post-step — actionlint clean; corrected [ASSUMED] gh invocation (PATCH-by-id, not edit-by-tag — drafts 404 by tag) and pin (v6.4.0; v6.1.0 doesn't exist), both noted in LEARNINGS.md
  - **Scope**: Create `.github/workflows/release-drafting.yml` per interface-spec.md: `name: Release Drafting`; `on: push: { branches: [main] }` (no tags); `permissions: { contents: write, pull-requests: read }`; `concurrency` group `${{ github.workflow }}-${{ github.ref }}` with `cancel-in-progress: true`; one `draft` job on `ubuntu-latest` with (step 1) the **pinned** `release-drafter/release-drafter` action (id `draft`, `config-name: release-drafter.yml`, `GITHUB_TOKEN`), and (step 2) a status step that reads `steps.draft.outputs.major_version` and marks the draft pre-release when `== 0`, else latest, via `gh release edit` (exact invocation `[ASSUMED]` — verify a draft is addressable by its tag at implement; fallback is release-drafter's static `prerelease` with a documented 1.0.0 flip). No `actions/checkout`.
  - **Acceptance criteria**:
    - Workflow triggers only on `push` to `main` (a tag push or PR does not trigger it); permissions are exactly `contents: write` + `pull-requests: read`; the action is pinned to a concrete version.
    - The draft is created/updated but never published (no tag created); a 0.x version yields a pre-release draft, a ≥1.0.0 version yields a latest draft.
    - The workflow is not added as a required status check (it blocks no merge); `actionlint` clean.
  - **Dependencies**: T002
  - **Plan reference**: Phase 2; ADR-1 (dedicated workflow, pinned action, push:main), ADR-5 (auto pre-release/latest)
  - **Scenario references**: release-drafting.feature: "The first ever release proposes v0.1.0 as a pre-release"; "The draft is marked latest once the version reaches 1.0.0"; "The draft is never published automatically"; "Reconciliation converges rather than duplicating"; "A drafting failure blocks nothing"
  - **Interface references**: interface-spec.md: "`.github/workflows/release-drafting.yml` structure"; "Interactions"
  - **Risk**: ⚠️ External behaviour — `gh release edit` on an unpublished draft (ADR-5) and the v0.1.0 first-release floor (ADR-2) are verified at implement; both have documented fallbacks.

- [ ] **T004** [Shared] [P] Add the label-contract config guard to `internal/build`
  - **Scope**: Add a Go config-drift test to `internal/build` (joining the existing `.goreleaser`/`release.yml` guards; parse YAML via `sigs.k8s.io/yaml`, change-detector rigor — a missing entry fails as loudly as an extra). Assert across `.github/release-drafter.yml`, `.github/labeler.yml`, and `.github/settings.yml`: (a) release-drafter `categories` labels == the seven category labels in labeler.yml and settings.yml; (b) `version-resolver` major/minor/patch buckets == exactly breaking/features/fixes; (c) `no-release-note` present in settings.yml, labeler.yml, and release-drafter `exclude-labels`; (d) the managed set is exactly eight across labeler.yml/settings.yml. Exact Go symbol/file names are implementation-level.
  - **Acceptance criteria**:
    - The test fails if any category label is renamed/dropped/added in one of the three files but not the others.
    - The test fails if the resolver buckets drift from breaking/features/fixes, or if `no-release-note` is missing from any of the three places, or if the managed set is not exactly eight.
    - The test passes against the T001/T002 configs and runs under `go test ./...` (so 024 pre-merge and 029 post-merge execute it). No runtime CLI package is added.
  - **Dependencies**: T001, T002
  - **Plan reference**: Phase 2; ADR-6 (internal/build label-contract guard)
  - **Scenario references**: release-drafting.feature: "Every release-note category traces to a managed PR Administration label" (@validation)
  - **Interface references**: interface-spec.md: "Structural contract — `internal/build` label-contract config guard"
