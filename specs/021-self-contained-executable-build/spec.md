# Specification: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Self-Contained Executable Build is the mechanism that turns the repository's source tree into runnable `glassfrog` executables that carry their own everything — the satisfaction of CONSTITUTION XII (Standalone Executable). The CLI's operators are practitioners and the AI agents acting for them, running in environments we don't control and can't assume carry any particular runtime. If running the tool first required installing a language runtime or supporting software, it wouldn't run where it's needed and adoption would stall. This feature guarantees the produced binary's only assumed external dependency is network access to the Glassfrog API.

The feature owns two build modes from one source tree: a **local development build** that produces a single runnable binary for the maintainer's own platform, and a **cross-platform release build** that produces one binary for each supported target — macOS amd64, macOS arm64, Linux amd64, Linux arm64. It is the dependency root of the distribution work: the Automated Release Pipeline (022) consumes its binaries to archive, checksum, and publish, and Version Embedding (023) stamps the release version through the same build entry point. This spec fixes only the *observable outputs* and the self-containment guarantee; the build's tool shape (Makefile target, shell script, or a release tool) is an architecture decision for `/score:plan`.

---

## Behavioral Accord

### Local development build

- When a maintainer runs the local build, the system produces a single executable for the host platform (the maintainer's own OS and architecture) that runs without any separately-installed runtime.

### Cross-platform release build

- When the release build runs, the system produces one executable for each supported target — macOS amd64, macOS arm64, Linux amd64, Linux arm64 — four binaries from one source tree.
- Each target binary is produced for its declared OS and architecture regardless of the host the build runs on, so a maintainer (or CI host) on one platform can produce the binaries for the others.
- The release build is invokable from a single repeatable entry point in this repository, so an automated pipeline (022) and a maintainer on a laptop invoke the same build rather than two paths that could drift.

### Self-containment (CONSTITUTION XII)

- Every produced binary — local or release — depends on nothing beyond the host operating system and network access to the Glassfrog API: no language runtime, no interpreter, no separately-installed library or service.
- When a produced binary is run on a clean environment — host OS plus network only, with nothing else installed — it executes successfully.

### Self-containment verification

- The feature defines a per-target verification: run a produced binary on a clean host of *its own* target and confirm it executes with no separately-installed dependency — the CONSTITUTION XII detection made executable, applied one target at a time.
- The feature provides this check and proves it on at least the host-platform target. How many of the four targets are exercised automatically — and where in a pipeline — is 022's concern; this feature does not require multi-architecture execution (e.g. emulation) of foreign-target binaries.
- When the verification runs against a binary that needs some separately-installed dependency, it fails and surfaces the violation.

---

## User Scenarios

**In order to** run the CLI in environments we don't control without first installing a runtime,
**as an** AI agent or practitioner,
**I want** a binary that runs on a bare host with only network access to the API.

**In order to** ship the CLI to every supported operator platform from one source tree,
**as a** maintainer,
**I want** a single release build that cross-compiles all four target binaries.

**In order to** check a change quickly on my own machine,
**as a** maintainer,
**I want** a local build that produces a runnable binary for my own platform.

---

## Non-Behaviors

- The system must not package, archive, checksum, or publish the binaries (no `.tar.gz`, no checksums file, no GitHub Release upload). **Why**: that is 022 Automated Release Pipeline; folding distribution into the build couples it to a publishing surface 022 owns and would force 022's redesign.
- The system must not inject or compute the release version into the binary. **Why**: that is 023 Version Embedding; the build produces the artifact and 023 stamps its version through the same entry point — merging them double-commits the version contract.
- The system must not produce binaries for Windows or any platform beyond the four supported targets. **Why**: the supported operator surface is macOS and Linux (FEATURE-MODEL); emitting an untested binary for an unsupported platform implies a support commitment the project has not made.
- The system must not require a particular host OS or architecture to produce the full matrix. **Why**: cross-compilation must work from any maintainer or CI host; pinning the build to one host platform would make the pipeline fragile and block local release builds.
- A produced binary must not require any software, service, or library to be installed before it runs, beyond the host OS and network to the API. **Why**: this is CONSTITUTION XII restated as a hard boundary — any such requirement is the exact failure the feature exists to prevent.

---

## Integration Boundaries

- **Glassfrog API (runtime dependency of the artifact)**: the only assumed external dependency of a produced binary is network access to the API. The build emits an artifact; the artifact reaches the API at runtime — the build itself needs no API access.
- **Repository source tree (upstream)**: the single source tree the build compiles; all four target binaries and the local binary come from it, with no per-platform source variation.
- **Automated Release Pipeline (022, downstream — not yet built)**: consumes the release build's binaries to archive, checksum, and publish, and owns where the self-containment verification runs automatically in CI. This feature exposes the binaries and the check it consumes.
- **Version Embedding (023, downstream — not yet built)**: stamps the release version into the binary at build time through this feature's build entry point; this feature leaves the version unset.

---

## Driving Scenarios

### Happy path

**Scenario: local build produces a runnable host binary**
Given a maintainer on a supported host platform
When they run the local development build
Then a single `glassfrog` executable for their own OS and architecture is produced
And it runs on their machine without any separately-installed runtime.

**Scenario: release build produces the full matrix from one source tree**
Given the repository source tree
When the cross-platform release build runs
Then one executable is produced for each of macOS amd64, macOS arm64, Linux amd64, and Linux arm64
And all four come from the same single source tree.

**Scenario: a produced binary runs on a clean environment**
Given a binary produced for a target platform
When it is run on a clean environment of that platform — host OS plus network only, nothing else installed
Then it executes successfully and can reach the Glassfrog API.

### Error scenarios

**Scenario: the self-containment check catches a runtime dependency**
Given a produced binary that requires some separately-installed dependency
When the self-containment verification runs against it on a clean host of its own target
Then the verification fails and surfaces the missing-dependency violation
And the binary is not treated as a valid self-contained artifact.

**Scenario: a failed target fails the whole release build**
Given the cross-platform release build is producing the four target binaries
When one target fails to build
Then the release build fails as a whole
And it does not emit a partial or inconsistent set of binaries.

### Edge cases

**Scenario: cross-compilation from a foreign host**
Given a maintainer working on a macOS arm64 host
When they run the release build
Then the Linux amd64 binary (and the other foreign-target binaries) are produced
And producing them does not require running on a Linux or amd64 host.

**Scenario: self-containment is per-target, not universal**
Given a binary produced for macOS arm64
When it is taken to a Linux host or a macOS amd64 host
Then it is not expected to run there — the guarantee is that each binary runs on a clean host of *its own* target, not that one binary runs everywhere.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the matrix is exactly the four declared targets**
Given the output of a release build
When the produced binaries are enumerated
Then there is exactly one binary for each of the four declared targets — none missing, and no binary for an unsupported platform (no Windows).

**Scenario: the artifact's only external need is the API**
Given any produced binary run on a clean host of its target
When it executes
Then its only unmet external dependency is network access to the Glassfrog API — no runtime, interpreter, or installed library is required.

**Scenario: one build entry point, no second path**
Given the build a maintainer invokes locally and the build the automated pipeline invokes
When both are traced
Then they resolve to the same release-build entry point, so the binaries a maintainer produces and the binaries the pipeline produces cannot diverge.

---

## Assumptions

- **Go cross-compilation with cgo disabled** *(technical)*: built via the Go toolchain's `GOOS`/`GOARCH` cross-compilation with cgo disabled for the supported targets, per the foundational language decision (DECISIONS, 2026-06-03). Self-containment — *not* "fully static" linking, which is platform-specific — is the criterion.
- **Binary name `glassfrog`** *(technical)*: the produced executable is named `glassfrog` (the root command's `Use`). Per-platform artifact *file* naming (e.g. an os/arch suffix) is 022's archiving concern, not this feature's.
- **Build tool shape deferred to plan** *(technical)*: whether the build is a Makefile target, a shell script, or a release tool (e.g. GoReleaser) is an architecture decision for `/score:plan`; this spec fixes only the observable outputs and the self-containment guarantee.
- **"Clean environment" per CONSTITUTION XII** *(technical)*: a clean environment means host OS plus network only, with no language runtime or extra software installed — the same condition CONSTITUTION XII names as its detection.

---

## Ambiguity Warnings

None — the warning raised during specification was resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-08

- **Self-containment check coverage across targets**: the verification is defined as a *per-target contract* — run a produced binary on a clean host of its own target and confirm it executes with no separately-installed dependency. This feature provides the check and proves it on at least the host-platform target; how many of the four targets are exercised automatically, and where in a pipeline, is 022's concern. This feature does not require multi-architecture execution (e.g. emulation) of foreign-target binaries. Sharpened the "Self-containment verification" accord group and the matching error scenario; removed the corresponding `[NEEDS CLARIFICATION]` marker.
