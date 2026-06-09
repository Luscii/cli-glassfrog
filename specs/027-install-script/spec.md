# Specification: Install Script

**Feature**: 027-install-script
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The Install Script is the **primary acquisition path** for Linux and macOS laptops and CI runners in the Self-Contained Distribution cluster. It is a POSIX shell script, hosted in this repository, that an operator runs as a one-liner: it detects the host operating system and architecture, resolves a release, downloads the matching archive and its checksums file from the project's GitHub Releases, verifies the archive's integrity against that checksums file, and installs the extracted binary onto the user's PATH. The result is a single command that turns a clean machine into one with a working `glassfrog` binary, with no language runtime, package manager, or manual download required.

It sits directly downstream of the Automated Release Pipeline (022): that pipeline attaches one archive per supported platform (macOS amd64/arm64, Linux amd64/arm64) plus one checksums file to each published release, and this script consumes exactly that output. It is one of several acquisition channels alongside the Homebrew Tap (036) and the NPM Wrapper (037); each channel installs the same released binary by a different route. The script owns *acquisition and placement* on the host — detection, selection, download, verification, and install — and owns none of *building*, *signing*, or *version embedding*, which live upstream. The installed binary itself remains the self-contained executable mandated by CONSTITUTION XII; the script is the host-side installer that places it, and is the only piece in this cluster that touches the user's machine directly.

---

## Behavioral Accord

### Platform detection

- When the script runs, it detects the host operating system and CPU architecture and maps them to one of the supported targets: macOS amd64, macOS arm64, Linux amd64, Linux arm64.
- When the host operating system or architecture is not one of the supported targets (for example Windows, or an unsupported architecture), the script stops with a clear message naming the detected platform and the supported set, installs nothing, and exits non-zero.

### Release resolution

- By default the script resolves the **latest stable release** — the newest release that is not marked as a pre-release.
- When the caller requests a specific version, the script resolves that exact release instead of the latest, and installs it even if it is a pre-release.
- When no release can be resolved (the requested version does not exist, or no stable release is published yet), the script stops with a clear message, installs nothing, and exits non-zero.

### Download and verify

- When a release and target are resolved, the script downloads the archive matching the detected platform and the release's checksums file from that release's attached assets.
- Before installing, the script verifies the downloaded archive against its entry in the checksums file.
- When the checksum matches, the script proceeds to install. When the checksum does not match, the script stops, installs nothing, leaves no partially-written binary on PATH, and exits non-zero.
- When a download fails (asset missing, network error), the script stops with a clear message, installs nothing, and exits non-zero.

### Install and placement

- When verification succeeds, the script extracts the binary and installs it into a per-user directory on the PATH. It does not require root and does not invoke privilege escalation.
- When the caller specifies an install directory, the script installs there instead of the default.
- When the binary is already installed at the target location, the script overwrites it with the freshly downloaded version — so re-running the script doubles as an upgrade (and as a downgrade when an older version is pinned).
- When the chosen install directory is not on the user's PATH, the script still installs the binary and prints the exact line the user can add to make it discoverable. It does not edit the user's shell profiles or environment files.
- On success, the script reports where the binary was installed and which version was placed.

---

## User Scenarios

**In order to** get a working CLI on a fresh Linux or macOS machine without installing a runtime or package manager,
**as an** AI agent or practitioner,
**I want to** run a single command that detects my platform, downloads the right binary, verifies it, and puts it on my PATH.

**In order to** provision the CLI reproducibly in CI without interactive privilege prompts,
**as a** maintainer writing a pipeline,
**I want to** run the install script unattended, optionally pinning a version and choosing an install directory, with no sudo required.

**In order to** trust that the binary I just installed is the authentic released artifact,
**as an** operator,
**I want to** have the download verified against the release's checksums file before it lands on my PATH.

**In order to** move to a new version (or roll back to a known-good one),
**as an** operator who already has the CLI installed,
**I want to** re-run the script and have it replace the existing binary in place.

---

## Non-Behaviors

- The script must not install on Windows or any platform outside the four supported targets. **Why**: the Automated Release Pipeline (022) attaches archives only for macOS and Linux on amd64/arm64; installing where no matching archive exists would either fail opaquely or place an unusable binary.
- The script must not build the binary or require a Go toolchain or source checkout. **Why**: it is an acquisition channel that consumes pre-built release archives; building on the host would defeat the self-contained distribution goal (CONSTITUTION XII) and reintroduce a build dependency.
- The script must not require root or run privilege escalation as part of the normal path. **Why**: agents and CI runners often have no sudo; defaulting to a per-user directory keeps the install unattended and friction-free. A system-wide install remains possible only by the caller pointing the install directory at a system location they can already write.
- The script must not install a binary whose checksum does not match the release's checksums file. **Why**: skipping verification would let a corrupted or tampered download reach the PATH; integrity is the whole reason the pipeline attaches a checksums file.
- The script must not edit the user's shell profiles or environment files. **Why**: silently rewriting `.bashrc`/`.zshrc`/`.profile` is surprising and hard to undo; printing the exact PATH line leaves the user in control of their own configuration.
- The script must not author release notes, sign or notarize artifacts, or bump versions. **Why**: those belong to Release Drafting (030), the (out-of-scope) signing concern, and Version Embedding (023); the script only places what the pipeline already produced.
- The script must not manage uninstalling or list/manage multiple installed versions. **Why**: the acquisition surface is install-and-upgrade-in-place; version management beyond overwrite is not part of this channel and would add lifecycle complexity no scenario requires.

---

## Integration Boundaries

- **Automated Release Pipeline (022 — upstream)**: attaches one archive per supported platform plus one checksums file to each published GitHub Release. The script depends on that output existing and on its archive-naming convention to pick the asset matching the detected platform. If the expected asset is absent, the script fails clearly rather than installing the wrong file.
- **GitHub Releases (source)**: the script queries releases to resolve the latest stable release (or a pinned version) and downloads the archive and checksums assets from the resolved release. Unavailability or a missing release surfaces as a clear failure.
- **Host environment (destination)**: the script reads the OS, architecture, PATH, and a caller-provided install directory if any; it writes the binary into the chosen per-user directory. It relies on standard host tooling (a download tool, an archive extractor, a checksum utility) being present — the installer is a shell script, distinct from the self-contained binary it installs.
- **Version Embedding (023 — informational)**: the installed binary reports its version via `--version`; the script reports the version it placed, and the embedded version is what an operator verifies after install.
- **Homebrew Tap (036) / NPM Wrapper (037) — sibling channels**: independent acquisition routes that install the same released binary. The script is unaware of them; they do not interact.

---

## Driving Scenarios

### Happy path

**Scenario: fresh install on a supported platform**
Given a clean Linux amd64 machine with no `glassfrog` binary installed
And a latest stable release with the four platform archives and a checksums file attached
When the operator runs the install script with no options
Then the script detects Linux amd64 and downloads the matching archive and the checksums file
And verifies the archive against the checksums file
And installs the binary into a per-user directory on PATH without sudo
And reports the install location and the installed version.

**Scenario: install a pinned version**
Given a published release `v1.3.0` exists alongside a newer `v1.4.0`
When the operator runs the script requesting version `v1.3.0`
Then the script resolves the `v1.3.0` release rather than the latest
And downloads, verifies, and installs the `v1.3.0` binary.

**Scenario: re-running upgrades in place**
Given `glassfrog` `v1.3.0` is already installed at the target location
And `v1.4.0` is the latest stable release
When the operator re-runs the script with no version pinned
Then the script installs `v1.4.0` over the existing binary at the same location
And the installed binary reports `v1.4.0`.

### Error scenarios

**Scenario: checksum mismatch aborts the install**
Given the downloaded archive does not match its entry in the checksums file
When the script runs
Then it stops before installing
And leaves no `glassfrog` binary written to the target location
And exits non-zero with a message naming the integrity failure.

**Scenario: unsupported platform is refused**
Given the script runs on Windows (or an unsupported architecture)
When platform detection completes
Then the script stops with a message naming the detected platform and the supported targets
And installs nothing
And exits non-zero.

### Edge cases

**Scenario: install directory not on PATH**
Given the chosen install directory is not present in the user's PATH
When the script finishes installing the binary
Then it reports the install location
And prints the exact line the user can add to put the directory on PATH
And does not modify any shell profile or environment file.

**Scenario: custom install directory**
Given the operator sets a custom install directory the operator can write
When the script runs
Then it installs the binary into that directory instead of the default per-user directory
And reports that location.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: verification gates the install**
Given a deliberately corrupted archive whose checksum will not match
When the script runs to completion or failure
Then at no point does a binary appear at the target location
And the script's exit status is non-zero.

**Scenario: the installed binary reports the resolved version**
Given a fresh install resolving the latest stable release (or a pinned version)
When the installed binary is run with `--version`
Then the reported version equals the version the script said it installed
And equals the resolved release's tag.

**Scenario: latest resolution excludes pre-releases**
Given the newest release is marked as a pre-release and an older release is the newest stable one
When the script runs with no version pinned
Then it resolves and installs the older stable release, not the pre-release
And a pre-release is installed only when its version is explicitly pinned.

---

## Assumptions

- **Default install directory** `[ASSUMED]`: the *behavior* is fixed — a per-user directory on PATH, no sudo, overridable by the caller. The literal default path (e.g. `~/.local/bin`) and the precedence with any existing convention are interface/plan details, pinned downstream.
- **Configuration mechanism** `[ASSUMED]`: the *behavior* is fixed — the caller can pin a version and choose an install directory. Whether these are environment variables (e.g. `VERSION`, `INSTALL_DIR`), flags, or both, and their exact names, is an interface decision, not a behavioral requirement.
- **Host tooling** (decision): the installer assumes standard POSIX tooling is present — a download tool (curl/wget), an archive extractor (tar), and a checksum utility (sha256sum/shasum). Which specific tools and the fallback order are a plan decision. This does not relax CONSTITUTION XII, which governs the distributed *binary*, not the shell installer.
- **Archive naming consumed from 022** (decision): the script relies on the Automated Release Pipeline's archive-naming convention to select the asset matching the detected platform. The naming itself is owned by 022; the script reads it.
- **Hosting and invocation** `[ASSUMED]`: the script is hosted in this repository and intended to run as a piped one-liner. The exact hosted URL and the canonical one-liner form are documentation/interface details.
- **No signing/notarization**: integrity is provided solely by the checksums file, matching 022's decision. Cryptographic signing or macOS notarization is out of scope here and would be an additive concern if a channel later requires it.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) default release resolution is the latest **stable** release, with pre-releases installed only when a version is explicitly pinned; (2) the default install target is a **per-user** directory on PATH with **no sudo**, overridable by the caller; (3) when the install directory is not on PATH the script **installs and warns with the exact PATH line**, without editing shell profiles; and (4) re-running **overwrites in place**, so the script doubles as an upgrade/downgrade path. The remaining `[ASSUMED]` items (default path, configuration mechanism and names, hosted URL/one-liner form) are interface/plan-level details, not behavioral gaps._
