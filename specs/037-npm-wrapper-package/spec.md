# Specification: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The NPM Wrapper Package is an **acquisition channel** in the Self-Contained Distribution cluster, aimed at Node-based agent environments. It is an npm package that, when installed, makes the released `glassfrog` binary runnable through the Node toolchain: an operator can `npx @luscii-healthtech/glassfrog ...` for a one-off, `npm i -g @luscii-healthtech/glassfrog` for a global command, or add it as a project dependency. The package does not contain the CLI's logic — it resolves and places the same self-contained binary the Automated Release Pipeline (022) already builds, exposing it as an npm-installed command. It is one of several acquisition channels alongside the Install Script (027) and the Homebrew Tap (036); each installs the same released binary by a different route, and none reimplements the tool.

It works by the conventional npm native-binary pattern: a thin wrapper package declares one **platform-specific optional dependency** per supported target, and npm installs only the package matching the host OS and CPU. A small launcher exposed by the wrapper execs whichever platform binary was installed, passing arguments and exit codes straight through. When no matching optional package is available (for example when optional dependencies are omitted, or the registry lacks the platform package), a **postinstall fallback** downloads the matching archive and checksums file from the release's GitHub assets — the same artifacts 027 consumes — verifies the archive against the checksums file, and only then places the binary. The released binary remains the self-contained executable mandated by CONSTITUTION XII; this channel publishes and places it through npm, and owns the npm publishing step that the release pipeline (022) deliberately leaves to it.

---

## Behavioral Accord

### Platform resolution

- When the package is installed on a supported target — macOS amd64, macOS arm64, Linux amd64, Linux arm64 — npm resolves the matching platform-specific optional dependency, and the wrapper's command runs that target's binary.
- When the package is installed on a platform outside the four supported targets (for example Windows, or an unsupported architecture), the install stops with a clear message naming the detected platform and the supported set, installs no runnable command, and exits non-zero. A Node environment running on an unsupported platform never ends up with a `glassfrog` command that cannot work.

### Install and placement

- When the matching platform optional dependency is present, the install completes using its bundled binary and contacts no external download source.
- When no matching optional dependency is present (optional dependencies omitted, or the registry lacks the platform package), the postinstall fallback downloads the archive matching the detected platform and the release's checksums file from that release's GitHub assets, verifies the archive against its checksums entry, and installs the binary only when the checksum matches.
- When the fallback's checksum does not match, or the download fails, the install stops, leaves no runnable `glassfrog` command in place, and exits non-zero.
- The version of the binary placed corresponds to the version of the npm package installed: installing `@luscii-healthtech/glassfrog@X.Y.Z` yields the binary whose `--version` reports `vX.Y.Z` — the release tag, i.e. the npm package version with the leading `v` the build injects (Version Embedding, 023). The npm package version omits the `v` (npm semver) while the binary's embedded version restores it.

### Invocation and pass-through

- When the operator runs the installed command (via `npx`, a global install, or a local dependency), the wrapper execs the resolved platform binary, forwarding all arguments, standard input, standard output, and standard error unchanged.
- When the underlying binary exits, the wrapper exits with the binary's exit code unchanged, so callers scripting against the CLI's exit-code convention see the same codes whether they invoked the binary directly or through npm.

### Upgrade

- When an operator installs a different version of the package, npm replaces the resolved binary with that version's binary, so re-installing doubles as an upgrade or a pinned downgrade.

---

## User Scenarios

**In order to** run the CLI once in a Node-based agent environment without a separate install step,
**as an** AI agent or practitioner,
**I want to** invoke `npx @luscii-healthtech/glassfrog ...` and have the right platform binary resolved and executed.

**In order to** provision the CLI reproducibly in a Node-centric CI pipeline,
**as a** maintainer,
**I want to** `npm i -g @luscii-healthtech/glassfrog@<version>` and get the matching binary, with the install pinned to a known version.

**In order to** trust that the binary npm placed is the authentic released artifact even when it was downloaded rather than bundled,
**as an** operator,
**I want to** have the fallback download verified against the release's checksums file before it becomes runnable.

**In order to** script the CLI from Node tooling the same way I would from a shell,
**as an** AI agent,
**I want to** have arguments and exit codes pass straight through the wrapper to the underlying binary.

---

## Non-Behaviors

- The package must not build the binary or require a Go toolchain or source checkout. **Why**: it is an acquisition channel that consumes pre-built release artifacts (and npm-published platform packages built from them); building on the host would defeat the self-contained distribution goal (CONSTITUTION XII) and reintroduce a build dependency.
- The package must not publish, or claim to install, a Windows or any other unsupported-platform artifact. **Why**: the Automated Release Pipeline (022) builds only macOS and Linux on amd64/arm64; offering a Node install path where no matching binary exists would either fail opaquely at first run or place an unusable command.
- The package must not place a binary whose checksum does not match the release's checksums file on the fallback path. **Why**: the download path must carry the same integrity guarantee as the Install Script (027); skipping verification would let a corrupted or tampered download become a runnable command.
- The package must not require network access at install when the matching platform optional dependency is available. **Why**: the bundled-binary path is the primary one and must work in offline or air-gapped Node environments; the download is only a fallback for when the bundled package is absent.
- The package must not modify, re-parse, or reinterpret the binary's arguments, output, or exit code. **Why**: the wrapper is a transparent launcher; altering exit codes or output would fork the CLI's behavior (exit-code convention, output rendering) between the npm channel and every other channel.
- The package must not produce the release archives or checksums file, author release notes, sign or notarize artifacts, or bump versions. **Why**: those belong to the Automated Release Pipeline (022), Release Drafting (030), the (out-of-scope) signing concern, and Version Embedding (023); this channel publishes and places what those produce.
- The package must not edit the user's shell profiles or PATH directly. **Why**: linking the command onto PATH is npm's job (its `bin` mechanism and global prefix); duplicating it would surprise operators and conflict with npm's own management.

---

## Integration Boundaries

- **Automated Release Pipeline (022 — upstream)**: attaches one archive per supported platform plus one checksums file to each published GitHub Release. The fallback path downloads the platform archive and the checksums file from that release and relies on the archive-naming convention to pick the right asset. 022 explicitly leaves npm publishing to this channel.
- **npm registry (destination and source)**: this channel publishes the wrapper package and the per-platform binary packages, versioned to match the release. Installs resolve the wrapper and its matching optional dependency from the registry. A registry or package unavailable for the host platform drops the install to the fallback path; a total registry failure surfaces as a normal npm install failure.
- **GitHub Releases (source — fallback)**: when the matching platform package is unavailable, the postinstall downloads the archive and checksums from the resolved release. A missing release or asset surfaces as a clear install failure with a non-zero exit.
- **Version Embedding (023 — informational)**: the installed binary reports its version via `--version` as `vX.Y.Z` (the release tag); the npm package version is that tag without the leading `v` (`X.Y.Z`, npm semver), and the placed binary's embedded version is what an operator verifies after install.
- **Node / npm environment (host)**: requires Node and npm; npm performs optional-dependency resolution by OS/architecture and links the command onto PATH. The wrapper reads the detected platform and, on the fallback path, standard download tooling available to a Node process.
- **Install Script (027) / Homebrew Tap (036) — sibling channels**: independent acquisition routes that place the same released binary. This channel is unaware of them; they do not interact.

---

## Driving Scenarios

### Happy path

**Scenario: npx on a supported platform resolves and runs the binary**
Given a published release with the npm wrapper and its per-platform packages published, on a Linux amd64 host with Node and npm
When the operator runs `npx @luscii-healthtech/glassfrog --version`
Then npm resolves the Linux amd64 optional dependency
And the wrapper execs that binary
And the reported version is the release tag — the installed package version with the leading `v` (e.g. package `1.4.0` → `v1.4.0`).

**Scenario: pinned global install places the matching binary**
Given a `@luscii-healthtech/glassfrog@1.3.0` package is published alongside a newer `1.4.0`
When the operator runs `npm i -g @luscii-healthtech/glassfrog@1.3.0` on a supported platform
Then the install resolves the `1.3.0` platform binary
And running `glassfrog --version` reports `v1.3.0`.

**Scenario: fallback download verifies before placing the binary**
Given the matching platform package is not available from the registry for the host
And the resolved release has the platform archive and checksums file attached
When the package is installed
Then the postinstall downloads the matching archive and the checksums file
And verifies the archive against its checksums entry
And places the binary only after the checksum matches.

### Error scenarios

**Scenario: unsupported platform is refused at install**
Given the package is installed on Windows (or an unsupported architecture)
When the install runs
Then it stops with a message naming the detected platform and the supported targets
And leaves no runnable `glassfrog` command
And exits non-zero.

**Scenario: checksum mismatch aborts the fallback install**
Given the fallback downloads an archive that does not match its checksums entry
When the install runs
Then it stops before placing the binary
And leaves no runnable `glassfrog` command
And exits non-zero with a message naming the integrity failure.

### Edge cases

**Scenario: offline install using the bundled platform package**
Given the matching platform optional dependency is available in the install cache
And there is no network access to GitHub Releases
When the package is installed
Then the install completes using the bundled binary
And does not attempt a fallback download.

**Scenario: exit code and arguments pass through unchanged**
Given the CLI is installed through the npm wrapper
When the operator runs a command that the underlying binary exits non-zero on (for example an API error)
Then the wrapper forwards the arguments to the binary
And exits with the binary's own exit code unchanged.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the placed binary's version matches the package and the release tag**
Given the package is installed at a specific version (via the bundled package or the fallback)
When the installed binary is run with `--version`
Then the reported version equals the resolved release's tag (e.g. `v1.3.0`)
And stripping the leading `v` equals the installed npm package version (e.g. `1.3.0`).

**Scenario: each supported platform resolves exactly its own binary**
Given the wrapper and per-platform packages are published for the four supported targets
When the package is installed on each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64
Then each install resolves the binary for that exact platform
And no install resolves a binary for a different OS or architecture.

**Scenario: verification gates the fallback path**
Given a deliberately corrupted archive whose checksum will not match, on the fallback path
When the install runs to completion or failure
Then at no point does a runnable `glassfrog` command appear
And the install's exit status is non-zero.

---

## Assumptions

- **Install mechanism** (decision): platform-specific optional dependencies with a postinstall fallback, per the project's Feature Model. The exact launcher/shim mechanism and how the postinstall detects an absent optional dependency are interface/plan details.
- **Package naming and scope** `[ASSUMED]`: the published wrapper package name, any npm scope, and the per-platform package names are interface/plan-level decisions, not behavioral requirements. The behavior fixed here is `npx` / `npm i -g` / local-dependency installability and one package per supported target.
- **Node/npm version floor** (decision): a reasonable minimum supported Node/npm is a plan detail; the behavior fixed here is that npm performs OS/architecture-based optional-dependency resolution and bin linking.
- **Archive naming consumed from 022** (decision): the fallback relies on the Automated Release Pipeline's archive-naming convention to select the asset matching the detected platform. The naming itself is owned by 022; this channel reads it.
- **Publishing trigger** `[ASSUMED]`: the npm publish runs off a published release (in parallel with 022/036), versioned to the release tag. Whether it is a dedicated job or part of the release tooling (e.g. GoReleaser's npm support) is a plan decision; the behavior fixed here is that published npm versions mirror release tags.
- **No signing/notarization**: integrity on the fallback path is provided solely by the checksums file, matching 022 and 027. Cryptographic signing or notarization is out of scope and would be an additive concern if a channel later requires it.

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) on an unsupported platform (e.g. Windows) the install **fails clearly and places nothing**, mirroring the Install Script (027); (2) the postinstall fallback **verifies the download against the release's checksums file** and refuses on mismatch, at parity with 027; and (3) this spec **owns publishing** the wrapper and per-platform packages to the npm registry on each release, versioned to the release tag, since 022 explicitly leaves npm to this channel. The remaining `[ASSUMED]` items (package names/scope, publishing-trigger mechanics) are interface/plan-level details, not behavioral gaps._
