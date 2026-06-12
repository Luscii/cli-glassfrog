# Specification: Homebrew Tap

**Feature**: 036-homebrew-tap
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The Homebrew Tap is an acquisition channel for **macOS and Linux** users in the Self-Contained Distribution cluster. It is a Homebrew **formula** published to a **dedicated tap repository** (`Luscii/homebrew-cli-glassfrog`, distinct from this CLI source repository). A user taps that repository and runs `brew install`, and Homebrew downloads the pre-built `glassfrog` release binary matching their platform and places it on PATH; later `brew upgrade` moves them to the newest stable release. No Go toolchain, no source compilation, no manual download — the formula installs the already-built release archive.

It sits directly downstream of the Automated Release Pipeline (022): that pipeline attaches one archive per supported platform (macOS amd64/arm64, Linux amd64/arm64) plus one checksums file to each published release, and the formula points at those archives, each pinned to its published sha256. It is one of several acquisition channels alongside the Install Script (027) and the NPM Wrapper (037); each channel installs the same released binary by a different route. This channel owns *the formula definition and keeping the tap repository current with stable releases* — it owns none of building, signing, version embedding, or release authoring, which live upstream. The tap is a **separate repository** so the formula can be published to it directly by the release process without committing anything to this repository's protected `main`.

---

## Behavioral Accord

### Tapping and installation

- When a user taps the dedicated tap repository and installs the formula, Homebrew downloads the release archive matching the user's operating system and CPU architecture, verifies it against the recorded checksum, and places a working `glassfrog` binary on PATH.
- When the user is on any supported platform — macOS amd64, macOS arm64, Linux amd64, or Linux arm64 — the formula resolves the archive matching that platform.
- The formula is installable both by tapping then installing, and by the one-shot tap-qualified install form (a single `brew install <tap>/glassfrog`).
- After a successful install, running `glassfrog version` (equivalently `glassfrog --version`) reports the version of the release the formula currently points at.
- The formula installs the pre-built release binary as-is; it does not compile from source.

### Currency with releases

- When a new **stable** release is published, the formula in the tap repository is updated to point at that release's archives and their checksums, so a subsequent `brew install`/`brew upgrade` lands that version.
- When a **pre-release** is published, the tap repository is left unchanged — `brew install`/`brew upgrade` continues to resolve the latest stable release, never a pre-release.

### Upgrade and integrity

- When an already-installed user runs `brew upgrade`, Homebrew moves them to the newest stable release the formula points at; when they are already on it, nothing changes.
- When a downloaded archive does not match the checksum recorded in the formula, Homebrew refuses the install and reports the integrity failure — no binary is placed.

---

## User Scenarios

**In order to** install the CLI with the package manager I already use, on either my Mac or my Linux box,
**as a** practitioner or AI agent on macOS or Linux,
**I want to** tap the repository and `brew install glassfrog`, and get a working binary on PATH without a runtime or manual download.

**In order to** stay current with releases the way I update everything else,
**as a** user who already installed via brew,
**I want to** run `brew upgrade` and move to the latest stable `glassfrog` automatically.

**In order to** trust that brew installed the authentic released artifact,
**as an** operator,
**I want to** have Homebrew verify the download against the checksum the release published before it lands on PATH.

---

## Non-Behaviors

- The channel must not build the binary or require a Go toolchain or source checkout. **Why**: it is an acquisition channel that installs the pre-built release binary; a formula that compiled on the host would defeat self-contained distribution (CONSTITUTION XII) and reintroduce a build dependency.
- The channel must not track pre-releases. **Why**: `brew install`/`brew upgrade` should always land a stable version, matching the Install Script's latest-stable default; a tap that floated to pre-releases would surprise users who expect brew to give them production builds.
- The release process must not commit the formula to *this* (the CLI source) repository or push to its `main`. **Why**: `main` is branch-protected and the release pipeline deliberately keeps the publishing/git side out of GoReleaser (it attaches assets with `gh release upload`, not `goreleaser publish`). The formula lives in a separate tap repository precisely so updating it never mutates this repository's protected branch.
- The channel must not author release notes, sign or notarize artifacts, or bump versions. **Why**: those belong to Release Drafting (030), the (out-of-scope) signing concern, and Version Embedding (023); this channel only points Homebrew at what the pipeline already produced.
- The channel must not support platforms outside the four release targets (e.g. Windows). **Why**: the Automated Release Pipeline attaches archives only for macOS and Linux on amd64/arm64; the formula can only install what the release provides.

---

## Integration Boundaries

- **Automated Release Pipeline (022 — upstream)**: attaches the four platform archives and a checksums file to each published release, under a stable naming convention. The formula consumes those asset names and their checksums; if an expected asset is absent, the install fails clearly rather than placing the wrong file.
- **GitHub Releases (source)**: the formula references the release's attached archives by URL and pins each to the sha256 from the release's checksums file.
- **Tap repository (`Luscii/homebrew-cli-glassfrog` — destination)**: a dedicated repository, separate from this one, that holds the formula. On each stable release the release process publishes the updated formula into this repository directly. It is the repository users tap. Writing to it requires a credential scoped to that repository (the default in-repo release token cannot write to another repository).
- **GoReleaser configuration (this repository)**: the formula is generated and published from the same `.goreleaser.yaml` that owns `builds`/`archives`/`checksum` (a `brew` section is reserved there for this feature). The formula's content (version, per-platform archive URLs, checksums, install stanza) is GoReleaser-authored, keeping one source of build truth.
- **Homebrew (user-facing surface)**: users interact through `brew tap`, `brew install`, and `brew upgrade`. The dedicated repository is the tap; both the explicit tap-then-install and the one-shot tap-qualified install forms work.
- **Install Script (027) / NPM Wrapper (037) — sibling channels**: independent acquisition routes that install the same released binary. They do not interact with this channel.

---

## Driving Scenarios

### Happy path

**Scenario: fresh install on macOS**
Given a Mac (arm64) with no `glassfrog` installed
And a latest stable release with the four platform archives and a checksums file attached
When the user taps the repository and runs `brew install glassfrog`
Then Homebrew downloads the macOS arm64 archive and verifies it against the recorded checksum
And places a working `glassfrog` binary on PATH
And `glassfrog version` reports the installed release's version.

**Scenario: fresh install on Linux**
Given a Linux amd64 machine with Homebrew and no `glassfrog` installed
When the user installs the formula
Then Homebrew resolves the Linux amd64 archive, verifies it, and installs the binary on PATH.

**Scenario: upgrade to the latest stable**
Given `glassfrog` was installed via brew at an older stable release
And a newer stable release has since been published and the formula updated to it
When the user runs `brew upgrade`
Then Homebrew moves the install to the newer stable release
And `glassfrog version` reports the newer version.

### Error scenarios

**Scenario: checksum mismatch refuses the install**
Given an archive that does not match the checksum recorded in the formula
When the user runs `brew install glassfrog`
Then Homebrew refuses the install and reports the integrity failure
And no `glassfrog` binary is placed.

**Scenario: expected release asset missing**
Given the formula references a release archive that is not attached to the release
When the user runs `brew install glassfrog`
Then the install fails clearly rather than placing a partial or wrong binary.

### Edge cases

**Scenario: pre-release does not move the tap**
Given the newest published release is marked as a pre-release
And the newest stable release is an older one
When a user runs `brew install` or `brew upgrade`
Then Homebrew resolves the older stable release, not the pre-release
And the tap repository's formula still points at the stable release.

**Scenario: already on the latest stable**
Given the user is already installed at the newest stable release
When the user runs `brew upgrade`
Then nothing is reinstalled and the version is unchanged.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the installed binary matches the release the formula points at**
Given a fresh `brew install` from the tap
When the installed binary is run with `glassfrog version`
Then the reported version equals the stable release the formula currently references
And equals that release's tag.

**Scenario: integrity gates the install**
Given a formula whose recorded checksum does not match the referenced archive
When `brew install` runs to completion or failure
Then at no point does a `glassfrog` binary appear on PATH from this channel.

**Scenario: a pre-release leaves the tap repository untouched**
Given a sequence of releases where the most recent is a pre-release
When the tap repository's formula is inspected after that pre-release is published
Then it still references the latest stable release, not the pre-release.

---

## Assumptions

- **Supported platforms** (decision): the formula serves the four release targets — macOS amd64/arm64 and Linux amd64/arm64 — the build matrix (021). How the formula expresses per-platform selection is a plan/interface detail.
- **Tap repository name and exact commands** `[ASSUMED]`: the *behavior* is fixed — a dedicated tap repository holds the formula; users `brew tap` it (or use the one-shot form) and then `brew install`/`brew upgrade`. The repository's exact name (e.g. `Luscii/homebrew-cli-glassfrog`) and the literal tap/install strings are interface/documentation details.
- **Publish mechanism and credential** (decision): each published stable release must leave the tap repository's formula pointing at that release. The release process publishes the formula to the separate tap repository using a credential scoped to that repository; the precise wiring (which GoReleaser invocation/step, gated on a verified non-prerelease publish, and the exact credential form — PAT vs GitHub App token) is a plan decision.
- **Integrity model**: integrity is provided by the per-archive sha256 the formula records (sourced from 022's checksums file), matching 022's no-signing decision. Cryptographic signing or notarization is out of scope here.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) the channel is a Homebrew **formula** (not a cask), so it serves macOS **and** Linux brew with a pre-built binary; (2) the formula tracks **stable releases only**, never pre-releases; (3) the tap is a **dedicated, separate repository**, so the release process publishes the formula to it directly and never commits to this repository's protected `main`, and the one-shot tap-qualified install form is supported. The remaining `[ASSUMED]` items (tap repository name, literal commands, the exact publish wiring and credential form) are interface/plan-level details, not behavioral gaps._
