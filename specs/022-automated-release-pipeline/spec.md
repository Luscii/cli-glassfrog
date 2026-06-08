# Specification: Automated Release Pipeline

**Feature**: 022-automated-release-pipeline
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The Automated Release Pipeline is the **ship mechanism** of the Self-Contained Distribution cluster. It is the automation that, when a GitHub Release is published, builds the supported platform binaries, packages each as a downloadable archive, generates a checksums file covering them, and attaches the whole set to that release. It runs entirely within this repository, with no manual build-and-upload step and no separate release service or repo.

It is the seam between *building* a binary and *acquiring* one. Self-Contained Executable Build (021) owns how a single dependency-free binary is cross-compiled for one platform; the pipeline orchestrates that build across every supported target, packages the results, and attaches them. Version Embedding (023) owns how the release version reaches `--version`; the pipeline reads the version from the published release's tag and supplies it to the build, but does not own the embedding mechanism. The acquisition channels — Install Script (027), Homebrew Tap (036), NPM Wrapper (037) — all consume the archives and checksums this pipeline attaches, so it gates every download path: nothing can be installed until the pipeline has attached it.

The trigger is the act of **publishing** a GitHub Release. In the normal flow, Release Drafting (030) maintains a draft release with label-driven notes and a semver bump as commits land on main; a maintainer publishes that draft (which creates the tag), and publishing is what triggers this pipeline. A release created and published by hand triggers it identically. So the pipeline depends on a published release existing — it owns the binaries and checksums, while Release Drafting (030) owns the notes, the draft, and the release's pre-release/latest status.

It deliberately does **not** run the lint/test quality gates (PR Validation (024) and Main-Branch Verification (029) own those — the pipeline assumes code that reached a published release is already validated), does **not** author or curate the release notes (Release Drafting (030) owns the notes the published release already carries), does **not** decide whether a release is a pre-release or the latest (that is set when the release is published), and does **not** run on routine branch pushes, merges, or an un-published tag — only publishing a release triggers it.

---

## Behavioral Accord

### Trigger

- When a GitHub Release is published in this repository — whether drafted by Release Drafting (030) or created by hand — the pipeline runs automatically with no further manual action.
- The pipeline does not run on routine branch pushes, on a merge to main, or on a tag push that has not been turned into a published release. Publishing a release is the single trigger.
- The pipeline reads the version from the published release's tag (a semver `vMAJOR.MINOR.PATCH`), and passes that version to the build and uses it to name the archives.
- The pipeline honors the published release's status: a release published as a pre-release stays a pre-release, and a normal (latest) release stays normal. The pipeline attaches artifacts; it does not promote, demote, or re-mark the release — the pre-release/latest decision is made when the release is published, not here.

### Build and package

- For each supported target platform — **macOS amd64, macOS arm64, Linux amd64, Linux arm64** — the pipeline produces a self-contained executable (cross-compiled via Self-Contained Executable Build (021)) carrying the version named by the release's tag.
- Each platform binary is packaged into a single downloadable archive named so a consumer can identify the tool, the version, the operating system, and the architecture from the file name alone.
- The pipeline generates one checksums file covering every attached archive, so a consumer can verify the integrity of any download.

### Attach

- When all target builds succeed, the pipeline attaches the archives and the checksums file to the published GitHub Release that triggered it.
- The pipeline does not author or edit the release notes; the published release already carries its notes (from Release Drafting (030) or whoever created it). The pipeline contributes the downloadable artifacts, not the prose.
- The attached set is the same for every release: one archive per supported platform plus one checksums file covering them.

### Atomicity and failure

- When the build for any target platform fails, the pipeline aborts and attaches nothing — there is no partial release with some platforms missing.
- The pipeline does not re-run the lint or test suite before attaching; a build that fails to *compile* still aborts, but verifying behavior is the job of PR Validation (024) and Main-Branch Verification (029), not this pipeline.
- The pipeline is the single artifact publisher for a release: running it again for a release that already has attached artifacts converges on one artifact set rather than producing duplicate or conflicting archives.

---

## User Scenarios

**In order to** ship a new version without manually building and uploading artifacts,
**as a** maintainer,
**I want to** publish a drafted release and have every platform archive and a checksums file built and attached automatically.

**In order to** verify that a downloaded binary is authentic and intact,
**as an** AI agent or practitioner acquiring the CLI,
**I want to** find a checksums file attached alongside the archives in the release.

**In order to** circulate a release candidate before promoting it to stable,
**as a** maintainer,
**I want to** publish a release marked as a pre-release and have the pipeline attach the same artifacts without changing that pre-release status.

---

## Non-Behaviors

- The pipeline must not run the lint or test quality gates before attaching artifacts. **Why**: PR Validation (024) and Main-Branch Verification (029) own behavioral verification; duplicating it in the release job would fork the quality contract and slow shipping. The pipeline assumes a published release rests on already-validated code. A failure to *compile* a target still aborts (see Atomicity).
- The pipeline must not implement the per-platform cross-compilation mechanics. **Why**: Self-Contained Executable Build (021) owns producing one dependency-free binary per platform (CONSTITUTION XII); the pipeline orchestrates that build across targets, packages, and attaches — it does not reinvent the build.
- The pipeline must not define how the release version reaches `--version`. **Why**: Version Embedding (023) owns build-time version injection and the `go install` fallback; the pipeline reads the tag and passes the version to the build, but the embedding mechanism lives in 023.
- The pipeline must not author or curate the release notes, nor maintain the draft. **Why**: Release Drafting (030) owns label-driven notes and the running draft; the published release already carries its notes when the pipeline runs, and the pipeline adds artifacts, not prose.
- The pipeline must not decide or change whether a release is a pre-release or the latest. **Why**: that status is set when the release is published (by Release Drafting (030), the publisher, or the tag's semver-suffix convention); the pipeline honors it so the publishing decision stays in one place.
- The pipeline must not publish Windows artifacts. **Why**: the supported matrix is macOS and Linux (amd64/arm64); Windows is not a target for this tool, so adding it would build artifacts no consumer channel installs.
- The pipeline must not provide any install or acquisition mechanics. **Why**: Install Script (027), Homebrew Tap (036), and NPM Wrapper (037) own acquisition; the pipeline only attaches the artifacts those channels consume.
- The pipeline must not run on routine commits, merges, branch pushes, or an un-published tag. **Why**: a release is an intentional, versioned act gated on publishing a GitHub Release; building on routine activity would flood releases with non-versioned artifacts.
- The pipeline must not produce a partial release. **Why**: a release missing one platform's archive is a broken download path for that platform's consumers; all-or-nothing keeps every release complete and every checksum entry backed by a real archive.

---

## Integration Boundaries

- **Release Drafting (030 — upstream / trigger source)**: as commits land on main, maintains a draft GitHub Release with label-driven notes and a semver-bumped tag. A maintainer publishes that draft, and publishing triggers this pipeline. The pipeline depends on a published release existing — normally produced by 030, but a release created and published by hand triggers it identically. 030 owns the notes, the draft, and the pre-release/latest status; the pipeline owns the binaries and the checksums file.
- **Self-Contained Executable Build (021 — upstream)**: provides the cross-compilation of one dependency-free binary per platform. The pipeline invokes it for each of the four targets, then packages and attaches the results. 021 owns the build; 022 owns the orchestrate-package-attach flow around it.
- **Version Embedding (023 — upstream)**: owns injecting the release version at build time (with the `go install` build-info fallback). The pipeline reads the version from the published release's tag and passes it into the build; 023 ensures `--version` reports it.
- **Published GitHub Release (trigger and destination)**: the publish event triggers the pipeline, and the same release is where the archives and checksums file are attached. The release's tag supplies the version; its status (pre-release/latest) is preserved.
- **Install Script (027) / Homebrew Tap (036) / NPM Wrapper (037) — downstream consumers**: each acquisition channel downloads the archive matching the host platform and verifies it against the attached checksums file. They consume the pipeline's output; the pipeline is unaware of them.
- **PR Validation (024) / Main-Branch Verification (029)**: own the test/lint gates that run before a release-worthy commit is published. The pipeline relies on them having already validated the code and does not re-run them.

---

## Driving Scenarios

### Happy path

**Scenario: publishing a release attaches all platform archives and a checksums file**
Given a draft release for tag `v1.4.0` exists (drafted by Release Drafting (030) or by hand)
When a maintainer publishes the release
Then the pipeline builds the macOS amd64, macOS arm64, Linux amd64, and Linux arm64 binaries
And packages each as a named archive
And attaches the four archives and one checksums file covering them to the `v1.4.0` release.

**Scenario: publishing a pre-release attaches artifacts without changing its status**
Given a release for tag `v1.4.0-rc.1` is published and marked as a pre-release
When the pipeline runs
Then it builds and attaches the same artifact set
And the release remains marked as a pre-release — the pipeline does not promote it to latest.

**Scenario: a consumer verifies a download against the checksums file**
Given a published release with four archives and a checksums file attached
When a consumer downloads one archive and the checksums file
Then the archive's checksum matches its entry in the checksums file
And the consumer can confirm the download is intact before installing.

### Error scenarios

**Scenario: a build failure aborts the whole release**
Given a release for tag `v1.4.0` is published
And the build for one target platform fails
When the pipeline runs
Then it aborts
And attaches no archives and no checksums file — there is no partial release.

**Scenario: routine activity without a published release triggers no build**
Given commits are merged to main and a draft release is updated, or a tag is pushed without publishing a release
When no release is published
Then the pipeline does not run
And no archives or checksums file are built or attached.

### Edge cases

**Scenario: re-running for an already-published release converges on one artifact set**
Given a `v1.4.0` release already has the artifact set attached
When the pipeline runs again for the `v1.4.0` release
Then it converges on a single attached artifact set
And does not create duplicate or conflicting archives.

**Scenario: a hand-created published release is handled identically**
Given a maintainer creates and publishes a `v1.4.0` release by hand, without Release Drafting (030)
When the pipeline runs
Then it builds and attaches the full artifact set exactly as it would for a drafted release
And does not require the draft-release flow to have produced the release.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: every attached archive has a matching checksum entry**
Given a published release with artifacts attached
When the checksums file is compared against the attached archives
Then every archive has exactly one matching entry
And every entry corresponds to an attached archive — no orphan entries, no unlisted archives.

**Scenario: the attached matrix is exactly the four supported targets**
Given a published release with artifacts attached
When the archives are enumerated
Then there is exactly one archive for each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64
And no archive for any other platform (including Windows).

**Scenario: the release's pre-release/latest status is preserved**
Given a release published as a pre-release and a release published as the latest
When the pipeline attaches artifacts to each
Then the pre-release stays a pre-release
And the latest stays the latest — the pipeline changes neither.

---

## Assumptions

- **Archive format and naming** `[ASSUMED]`: the *behavior* is fixed — one downloadable archive per target, named to identify tool, version, OS, and architecture, plus one checksums file covering all of them. The literal archive format (e.g. `tar.gz`) and exact file-name template are interface/plan details, pinned downstream alongside the build tooling.
- **Release tooling is a plan decision**: whether the pipeline is driven by a release tool (e.g. GoReleaser, used in the reference pipelines and named in the Homebrew Tap (036) item) or assembled by hand is an implementation choice for the plan, not a behavioral requirement. The spec fixes *what* is attached and *when*, not *how*.
- **Version source is the release's tag** (decision): the pipeline derives the version from the published release's semver tag (`vMAJOR.MINOR.PATCH`) and passes it to the build; it does not compute or bump a version itself.
- **Re-publish convergence** (decision): the pipeline is the single artifact publisher for a release; running it again for the same release converges on one artifact set rather than duplicating. Recorded here for traceability.
- **No artifact signing/notarization**: integrity is provided via the checksums file; cryptographic signing or macOS notarization is not in scope and surfaced no requirement. Could be revisited as an additive concern if a consumer channel needs it.
- **Validated-before-publish**: the pipeline assumes the published release rests on code already gated by PR Validation (024) / Main-Branch Verification (029). It does not re-gate, beyond the implicit requirement that every target compiles.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation, including a trigger-model revision after reviewing the maintainer's reference pipelines: (1) the trigger is **publishing a GitHub Release** (not a raw tag push) — Release Drafting (030) drafts the release, a maintainer publishes it, and that publish event runs the pipeline; a hand-published release works identically; (2) the pipeline depends on a published release existing and owns the binaries and checksums, while Release Drafting (030) owns the notes, draft, and pre-release/latest status, which the pipeline honors rather than decides; (3) the published matrix is macOS and Linux on amd64/arm64 (four archives) plus one checksums file, with Windows out of scope; and (4) the release is atomic — any target build failure aborts the whole release, and re-running the full test suite belongs to PR Validation (024) / Main-Branch Verification (029), not this pipeline. The remaining `[ASSUMED]` items (archive format/naming, release tooling) are interface/plan-level details, not behavioral gaps._

---

## Clarifications

### Session 2026-06-08

- **Trigger and pre-releases (initial)**: a release is triggered by a semver version tag; a pre-release-suffixed tag publishes a marked pre-release. *(Superseded below after reviewing the reference pipelines.)*
- **Platform matrix**: the pipeline publishes four archives — macOS amd64, macOS arm64, Linux amd64, Linux arm64 — plus a single checksums file covering them. Windows is out of scope.
- **Relationship to Release Drafting (030)**: see the revised trigger model below.
- **Pre-publish gating and atomicity**: the pipeline does not re-run the lint/test suite (PR Validation (024) and Main-Branch Verification (029) own that). Every target must build; any build failure aborts the whole release, so there is never a partial publish.

### Session 2026-06-08 — trigger model revised (reference pipelines)

After reviewing the maintainer's reference pipelines (an analogous Go CLI release workflow and a Luscii repo template), the trigger model was changed from raw tag-push to **`release: published`**, matching how the maintainer actually ships:

- **Trigger is publishing a release**: Release Drafting (030) maintains a draft release with label-driven notes and a semver bump as commits land on main; a maintainer publishes the draft (creating the tag), and the publish event triggers this pipeline. A release created and published by hand triggers it identically. The FEATURE-MODEL's own cross-reference agrees — it calls 022 the pipeline that "consumes the published tag."
- **Dependency reframed**: the pipeline depends on a published release existing (normally from 030, optionally hand-created). It owns the binaries and the checksums file; Release Drafting (030) owns the notes, the draft, and the pre-release/latest status. The pipeline honors that status rather than deciding it, and adds artifacts rather than authoring notes.
- **Deliberate differences from the reference**: the reference re-runs the test suite inside its release job, builds Windows + zip, and publishes NPM/Docker artifacts in the same workflow. In this project those concerns are decomposed into separate capabilities — PR Validation (024) / Main-Branch Verification (029) own testing, NPM Wrapper (037) owns npm, and Windows/Docker are out of scope — so 022 stays narrowly the build-package-attach step.
