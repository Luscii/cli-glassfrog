# Plan: Release Drafting

**Feature**: 030-release-drafting
**Role**: Shaper
**Inputs**: spec.md (030), PROJECT.md, `.score/memory/DECISIONS.md` (precedent grep: 021/022/023/024/028/029), `.score/memory/DEPRECATION.md`; existing `.github/` (labeler.yml, settings.yml, workflows/release.yml, ci.yml, main-verify.yml, test.yml) and `internal/build/` config-guard package; verified `srvaroa/labeler` matcher semantics

---

## System Architecture

Release Drafting is a CI-only feature: it ships **no application code**. It is a GitHub Actions workflow plus committed declarative config, guarded by a Go config-drift test. Three artifacts, plus one coordinated extension to 028:

- **`.github/workflows/release-drafting.yml`** — a new workflow on `push: { branches: [main] }` with `permissions: { contents: write, pull-requests: read }`. On every merge to main it runs the pinned `release-drafter` action, which regenerates the single draft release, then a follow-on step sets the draft's pre-release/latest status from the resolved version.
- **`.github/release-drafter.yml`** — release-drafter's config: the `version-resolver` (label→bump mapping + default), the `categories` (label→note-section mapping), `exclude-labels`, and the name/tag/version templates.
- **`internal/build/` release-drafter config guard** — a Go test (joining the existing `.goreleaser.yaml` config-guard and `release.yml` workflow guard) that parses `.github/release-drafter.yml`, `.github/labeler.yml`, and `.github/settings.yml` and asserts the label contract is consistent across all three with change-detector rigor.
- **Coordinated 028 change** — add an eighth managed label (the exclusion label) to `.github/settings.yml` (catalog) and `.github/labeler.yml` (a matcher that applies it to PRs touching nothing noteworthy), so release-drafter can exclude spec/feature-only PRs purely by label.

**Data flow**: merge to `main` → `release-drafting.yml` fires → release-drafter reads the merged PRs since the last published release and their 028-applied labels → computes the next version (highest-wins resolver, default patch) → groups PR titles into note categories, dropping `exclude-labels` PRs → writes/updates the draft release → the prerelease step reads release-drafter's `major_version` output and marks the draft pre-release (0.x) or latest (≥1.0.0). The draft stays unpublished; a maintainer publishing it is what triggers 022's `release.yml`.

This sits cleanly inside the pipeline cluster already on disk: 024 (`ci.yml`, pre-merge gate), 028 (`pr-administration.yml`, labels), 029 (`main-verify.yml`, post-merge tests), 022 (`release.yml`, build+attach on publish). 030 is the only stage that writes the draft's notes/version/status — 022 explicitly honors them (DECISIONS, 022 ADR).

---

## Architecture Decisions

### ADR-1: Realize drafting with the pinned `release-drafter` action on a dedicated `release-drafting.yml`, not by extending an existing workflow

**Context**: The spec triggers on merge to `main` and maintains a draft GitHub Release. FEATURE-MODEL names release-drafter; 028's DECISIONS records that both developer-reference repos (ailign-cli, terraform-vitals-template) use the `srvaroa/labeler` + release-drafter shape. `main-verify.yml` (029) already runs on `push: main` but with `contents: read`, tests-only, no enforcement.

**Options considered**:
1. **Extend `main-verify.yml`** — add a drafting job to the existing push:main workflow. Fewer files, but forces `contents: write` onto a workflow 029 deliberately scoped `contents: read`, and couples the post-merge test net to release bookkeeping.
2. **New dedicated `release-drafting.yml`** — separate file, separate trigger config, least-privilege `contents: write` + `pull-requests: read`. One more workflow file; clean concern/permission separation.

**Decision**: Option 2. This is silent conformance to 028's precedent (a separate `pr-administration.yml` rather than extending 024's `ci.yml`, justified there by divergent permissions/trigger). Release Drafting needs write access to releases that the test workflows must not have; a separate file keeps each workflow's permission surface minimal and its purpose single. The action is pinned to a concrete version (024/028 action-pinning discipline) so an upstream tag move can't silently change drafting behaviour.

**Consequences**: A fourth pipeline workflow joins the cluster. Per-`main`-ref `concurrency` + `cancel-in-progress` keeps the draft reflecting the latest tip. The workflow is not a required status check (it is post-merge bookkeeping; `ci-success` from 024 stays the sole gate), so a drafting failure never blocks anything — consistent with the spec's "administrative, runs after merge."

### ADR-2: Resolve the semver bump with release-drafter's `version-resolver` (highest-wins; default patch); first release floors at v0.1.0 via the 0.0.0 base

**Context**: The spec requires highest-wins resolution (breaking→major, else features→minor, else fixes→patch), a patch default when no semver-bearing label is present, and a v0.1.0 floor for the first-ever release. 028 deliberately deferred bump resolution to 030 (DECISIONS: "030 owns resolution — which semver label wins, default bump").

**Options considered**:
1. **release-drafter `version-resolver`** — declarative `major.labels`/`minor.labels`/`patch.labels` + `default: patch`. release-drafter computes the bump highest-wins natively against the last release tag. Matches the spec almost exactly; the first-release floor needs care.
2. **A hand-rolled bump step** (script reads labels via `gh`, computes the tag) — full control over the v0.1.0 floor, but reinvents what release-drafter does and forks the mechanism FEATURE-MODEL committed to.

**Decision**: Option 1. Map the three semver-bearing labels to the resolver buckets and set `default: patch`; release-drafter applies highest-wins across the included PRs. With no prior release, release-drafter computes from a `0.0.0` base, so a first release carrying a `features` (or `breaking`) label naturally yields `v0.1.0` — the spec's first-release scenario. The exact label strings come from 028's catalog; the resolver config references them.

**Consequences**: The patch-default and highest-wins behaviour are configuration, not code — cheap and legible. The v0.1.0 floor holds natively for any first release that includes a minor-or-higher change, which a first CLI release will; a *patch/docs-only* first release would compute `v0.0.1` under the native 0.0.0 base, which does not meet the literal floor (see Risks — pinned as an interface/config detail to verify, e.g. an explicit initial-version setting or a seed). The label→bucket mapping is part of the cross-feature contract the config guard pins (ADR-6).

### ADR-3: Map the seven 028 categories to note sections in `release-drafter.yml`, reusing the exact label strings

**Context**: The spec files each PR title under one of seven categories whose names must match 028's managed set exactly (validation scenario; DECISIONS: "this label SET is the cross-feature contract 028 PRODUCES and 030 CONSUMES"). 028 chose exactly seven labels with no parallel `version:*` namespace.

**Options considered**:
1. **`categories:` block keyed on the seven existing labels** — each label maps to a titled note section (Breaking / Features / Fixes / Docs / Infrastructure / Dependencies / Internal). Single source of label truth (028's catalog); zero new label vocabulary.
2. **Introduce drafting-specific category labels** — a separate note taxonomy. Rejected outright: forks the label contract, desync risk, contradicts 028's deliberate single-family decision.

**Decision**: Option 1 — silent conformance to 028's seven-label contract. release-drafter's `categories` reference the same strings 028's `settings.yml`/`labeler.yml` define; section titles (display text) are the only new strings and are interface-level.

**Consequences**: Renaming any category label is a coordinated three-file change (settings.yml, labeler.yml, release-drafter.yml) — exactly what the config guard (ADR-6) exists to make fail loudly. Categories with no contributing PR are omitted by release-drafter natively (spec's "omitted from the notes").

### ADR-4: Exclusion is purely label-driven; add an eighth managed label that 028's labeler applies to PRs touching nothing noteworthy

**Context**: The spec excludes two PR classes from notes and the bump: those carrying an exclusion label, and those whose changes are confined to `specs/` and `.feature` files. release-drafter filters by **label only** — it has no path-based exclusion. The developer chose (define + this resolve) to ship the full behaviour now via a coordinated 028 change. Verified constraint: `srvaroa/labeler`'s `files:` matcher is *any-file-matches*; it has no native *all-files-confined-to* mode, and `negate` inverts an entire label block.

**Options considered**:
1. **028 labeler applies the exclusion label; 030 lists it in `exclude-labels`** — single exclusion mechanism (a label), conforms to the 028-produces/030-consumes contract. Requires extending 028's managed set from seven to eight and expressing "nothing noteworthy changed" within the labeler's matcher limits.
2. **030 post-filters release-drafter's draft on changed paths** — no 028 change, but 030 would own a second exclusion mechanism and fight release-drafter's label-driven model (it computes the PR set itself). Rejected (developer chose Option 1).

**Decision**: Option 1. Add one managed label (working name `no-release-note`; exact string is interface-level) to `settings.yml` (catalog) and `labeler.yml`. Because the labeler cannot say "all files are spec/feature," express the inverse with a `negate` block over the **noteworthy** path patterns — i.e. apply the exclusion label when *no* changed file matches code/docs/infra/deps paths (the patterns the existing seven labels already enumerate). That is a faithful realization of the developer's rule ("only when the change contains code or real docs… does it appear"): the complement of noteworthy is spec/feature-only churn. 030's `release-drafter.yml` lists this label under `exclude-labels`, so excluded PRs leave the notes and, because they carry no *counted* semver label in the draft set, do not drive the bump.

**Consequences**: This widens 028's "EXACTLY seven managed labels" invariant (DECISIONS, 028) to eight — a deliberate, announced divergence from that precedent, recorded here and surfaced for `/score:deprecate` consideration in the handoff. The negate-over-noteworthy expression reuses 028's existing path patterns, so the two stay maintainable together; its edge cases (a PR changing only a non-spec, non-noteworthy file like `.gitignore`; an empty PR) are accepted as also-excluded and flagged (Risks). The config guard (ADR-6) pins all eight labels across the three files.

### ADR-5: Set pre-release/latest automatically from the resolved version via a post-draft step

**Context**: The spec (clarified) derives status from the version — pre-release while `0.x`, latest at `≥1.0.0` — and 022 honors whatever status the draft carries. release-drafter's `prerelease` config is a *static* boolean; it has no version-conditional mode, but it exposes a `major_version` (and `resolved_version`) output.

**Options considered**:
1. **Static `prerelease: true`, flipped to `false` by hand at the 1.0.0 cut** — trivial config, but not the automatic rule the spec commits to; relies on a maintainer remembering at exactly one moment.
2. **Post-draft step reads release-drafter's `major_version` output and edits the draft's flag** — `major_version == 0` → mark pre-release, else mark latest, via `gh release edit` on the draft. Honors the automatic rule with no manual step.

**Decision**: Option 2. The workflow's drafting step exposes the resolver outputs; a following step sets the draft pre-release when the major version is `0`, latest otherwise. The exact `gh` invocation (targeting the draft by id/tag, the `--prerelease`/`--latest` flags) is interface-level.

**Consequences**: The status follows the version with zero maintenance and crosses over correctly the first time a `breaking` change pushes the project to `1.0.0`. It depends on `gh release edit` being able to set the flag on an unpublished draft (verified at implement; flagged in Risks). Status lives in exactly one place (the draft), which is the seam 022 reads.

### ADR-6: Add the label-contract config guard to `internal/build`, asserting all eight labels agree across the three config files

**Context**: DECISIONS (028) states plainly: the label strings are a cross-feature contract "currently un-guarded until 030 ships (a config-guard test belongs to 030)." `internal/build` is the established home for build/release config-drift guards (021 `.goreleaser` matrix guard, 022 release-workflow guard), parsing YAML via `sigs.k8s.io/yaml` with change-detector rigor (a missing entry fails as loudly as an extra one).

**Options considered**:
1. **New guard in `internal/build`** — joins the existing `config.go`/`workflow.go` guards; reuses the package's YAML-parse + exact-set-assertion idiom. Conforms to precedent; one home for all pipeline-config guards.
2. **A separate package / a YAML-lint-only CI step** — isolates the new guard, but forks the established config-guard home and loses the Go test-suite integration (the guard would not run under 024/029's `go test ./...`).

**Decision**: Option 1 — silent conformance to the `internal/build` precedent. The guard parses `.github/release-drafter.yml`, `.github/labeler.yml`, and `.github/settings.yml` and asserts: (a) the seven category labels in release-drafter's `categories` exactly equal the seven semver/category labels in labeler.yml and settings.yml; (b) the eighth exclusion label appears in settings.yml + labeler.yml and in release-drafter's `exclude-labels`; (c) the resolver's `major`/`minor`/`patch` label buckets are exactly `breaking`/`features`/`fixes`. Any rename, drop, or addition in one file without the others fails the test.

**Consequences**: The 028↔030 label contract becomes structurally enforced — it runs in the existing `go test ./...` suite that 024 (pre-merge) and 029 (post-merge) already execute, so a desync reddens CI rather than silently mis-drafting a release. This is the one piece of Go in the feature; it adds no runtime package, matching `internal/build`'s "the system under test is the pipeline itself" charter.

---

## Cross-cutting Concerns

**Failure handling**: Drafting is non-blocking by construction — `release-drafting.yml` is not a required check, so a failed run leaves the previous draft intact and blocks nothing (spec's administrative guarantee). The draft is regenerated authoritatively each run (release-drafter overwrites its own draft, keyed by the draft release it manages), so reconciliation converges rather than duplicating (spec edge-case scenario) with no idempotency code of our own.

**Configuration vs hardcoded**: Everything behavioural lives in committed config — the resolver buckets, default bump, categories, exclude-labels, and templates in `release-drafter.yml`; the trigger/permissions/concurrency in the workflow; the labels in `settings.yml`/`labeler.yml`. Only the prerelease threshold (`major_version == 0`) is expressed in the workflow step. Nothing is computed in application code.

**Testing strategy**: One config-drift guard (ADR-6) in `internal/build`, run by the existing `go test ./...` matrix (no new CI wiring). The spec's behavioural scenarios (bump resolution, categorization, exclusion, first-release, prerelease) are validated by the config guard at the contract level plus the scenarios skill's feature file; end-to-end drafting is exercised by the action itself on real merges — there is no local harness for the GitHub-hosted draft, mirroring how 022's release behaviour is guarded at config level rather than executed locally.

**Security/permissions**: Least privilege — `contents: write` (to write the draft release) + `pull-requests: read` (to read merged-PR labels/titles); no secrets beyond the default `GITHUB_TOKEN`, no `id-token`, no checkout of PR head code (the workflow runs on `main` post-merge, not on untrusted PR refs, so it does not inherit 028's `pull_request_target` hazard).

---

## Implementation Strategy

The feature is small (config + one workflow + one guard test + a coordinated 028 edit). Two phases, sequenced so the contract is pinned before it is relied upon.

**Phase 1 — Label contract extension (028 coordination)**: Add the eighth exclusion label to `.github/settings.yml` (catalog entry: name, color, description) and `.github/labeler.yml` (the `negate`-over-noteworthy matcher). This is the upstream half of ADR-4; it must land first so the label exists before 030 references it. Includes updating 028's in-file header comments that assert "seven managed labels."

**Phase 2 — Drafting workflow, config, and guard**: Add `.github/release-drafter.yml` (resolver, categories, exclude-labels, templates — ADR-2/3/4), `.github/workflows/release-drafting.yml` (trigger, permissions, concurrency, pinned action, prerelease post-step — ADR-1/5), and the `internal/build` config guard (ADR-6) asserting all eight labels agree across the three files. Phase 2 depends on Phase 1 (the guard and the `exclude-labels` reference the label Phase 1 introduces).

Each phase is independently reviewable and PR-sized; the tasks skill decomposes accordingly.

---

## Risks

- **First-release v0.1.0 floor under release-drafter's native 0.0.0 base** *(medium likelihood, low impact)*: a *patch/docs-only* first release would compute `v0.0.1`, not the spec's `v0.1.0` floor. Mitigation: verify release-drafter's first-release behaviour during interface/implement and, if needed, pin the floor via an explicit initial-version config or a one-time seed tag. A first CLI release almost certainly carries a feature, which yields `v0.1.0` natively, so the gap is a corner case.
- **`negate`-over-noteworthy over-excludes** *(low likelihood, low impact)*: the ADR-4 matcher excludes any PR that touches *nothing* noteworthy, which also catches a PR changing only a stray non-spec, non-noteworthy file (e.g. `.gitignore`, an editor config) or an empty PR. Mitigation: the noteworthy pattern set is the same one 028 already maintains; widen it if a real category of change is wrongly excluded. The config guard makes the pattern set visible and reviewable.
- **`gh release edit` on an unpublished draft** *(low likelihood, medium impact)*: ADR-5's auto-status step assumes the draft's pre-release flag can be set before publish. Mitigation: confirm at implement; fallback is release-drafter's static `prerelease` config with a documented 1.0.0 flip (ADR-5 Option 1), which still satisfies the spec for the project's current 0.x phase.
- **Widening 028's seven-label invariant** *(certain, low impact — a coordination risk)*: ADR-4 makes 028 manage eight labels, diverging from the recorded "EXACTLY seven" decision. Mitigation: announced here as a divergence ADR; surfaced for `/score:deprecate`; the config guard re-pins the new (eight-label) invariant so it cannot silently drift again.

---

## What This Plan Does Not Cover

- **Exact strings and YAML** — the concrete exclusion-label name, release-drafter section titles, template tokens (`$RESOLVED_VERSION`, tag prefix), the labeler regex set, the workflow's step/job names, the pinned action version, and the precise `gh release edit` invocation are **interface-level** (the interface skill produces the contract; these are stable contracts once chosen).
- **Executable scenarios** — the Gherkin realization of the spec's driving scenarios (and the config-guard assertions phrased as scenarios) belongs to the scenarios skill.
- **Task decomposition** — the PR-sized breakdown of the two phases belongs to the tasks skill.
- **028's broader labeler tuning** — only the single exclusion-label addition is in scope; the existing seven labels' regexes are untouched.
- **Branch-protection / publish flow** — publishing the draft is a maintainer act and 022's trigger; this plan stops at maintaining the draft.
