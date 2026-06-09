# Specification: Version Embedding

**Feature**: 023-version-embedding
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Version Embedding is what makes `--version` tell the truth. The CLI already has a place to *report* a version string — Help & Version (003) renders `glassfrog --version` and `glassfrog version` — but on its own the binary has no real version to report: an un-stamped build falls back to a development placeholder, so a shipped release would announce itself as "dev". This feature owns getting a *meaningful* version into the binary so that, whichever way an operator obtained the CLI, `--version` names the build they are actually running. That matters because the CLI's operator is usually an AI agent acting for a practitioner: it confirms which build it is talking to before relying on its behavior, and a wrong or absent version misleads that judgement.

The feature sits inside the Self-Contained Distribution cluster as the version contract shared by the build and the release pipeline. Self-Contained Executable Build (021) produces the binary and deliberately leaves the version unset — it exposes a single build entry point with an empty version-injection seam. Automated Release Pipeline (022) reads the version from a published release's tag and supplies it to that build, but does not own the embedding. Version Embedding owns the rest: the mechanism that stamps a supplied version into the binary at build time, the runtime *resolution* of which version to report when no version was stamped (a fallback to the version Go records for source builds), and the development placeholder of last resort. It decides *what value* `--version` resolves to; it does not decide how that value is printed (003) or where a release's version number originates (022).

---

## Behavioral Accord

### Build-time embedding

- When the build is invoked with a release version supplied (the path used by the release pipeline), the system stamps that exact version into the produced binary, so `--version` and the `version` command report it verbatim — including a pre-release suffix such as `-rc.1`.
- The version is embedded through the build's single existing entry point (021), not a separate build path, so the binary a maintainer builds locally and the binary the pipeline builds resolve their version the same way.
- The supplied version is embedded as given; the system does not normalize, re-derive, or bump it.

### Version resolution at runtime

- When the binary is asked for its version, the system resolves a single version string by precedence: a build-time-embedded version takes priority; if none was embedded, the version Go recorded in the binary's module build info is used; if no build info is available at all, a clear development placeholder is used.
- An embedded release version always wins over recorded build info — the explicitly stamped version is authoritative whenever it is present.
- When no version was embedded, the system reports the module version Go recorded for the binary verbatim — a real release tag (`vX.Y.Z`) for a tagged source install, a pseudo-version (e.g. `v0.0.0-<timestamp>-<commit>`) for an untagged source install, and Go's `(devel)` marker for a plain local build — each surfaced as-is rather than reshaped.

### Fallback of last resort

- When neither an embedded version nor any module build info is available, the system reports a clear, non-empty development placeholder rather than an empty string, so an operator never sees blank version output.
- The placeholder is recognizable as a non-release marker, so it cannot be mistaken for a published version.

---

## User Scenarios

**In order to** confirm I am operating the exact published build before I trust its behavior,
**as an** AI agent operating the CLI,
**I want** `--version` to report the real release version that was shipped, not a development placeholder.

**In order to** get a meaningful version even when I installed the CLI from source,
**as a** practitioner or maintainer who ran `go install`,
**I want** the version to reflect the module version Go recorded for that install.

**In order to** ship releases whose binaries self-report their version with no extra manual step,
**as a** maintainer,
**I want** the release version to be stamped into every binary through the one build entry point the pipeline already invokes.

---

## Non-Behaviors

- The system must not decide how the version string is formatted or printed. **Why**: Help & Version (003) owns `--version`/`version` rendering (bare string, identical output for both forms, `--help` precedence). This feature resolves *which value* is reported; coupling formatting here would create two owners of the output surface.
- The system must not compute, bump, increment, or derive a version number. **Why**: the version's origin is a release tag read by Automated Release Pipeline (022); inventing or incrementing a version here would fork the source of truth for what version a release is.
- The system must not emit build metadata — commit hash, build date, builder — as part of the version output. **Why**: 003 fixes the scope as a bare version string and explicitly excludes build metadata; enriched version output is a separate deferred concern. (A pseudo-version that Go itself embeds in its recorded module version is reported as-is — that is the recorded *version*, not added metadata.)
- The system must not change how the binary is built or cross-compiled. **Why**: Self-Contained Executable Build (021) owns the build and self-containment (CONSTITUTION XII); this feature fills the version seam 021 left empty, it does not alter the build matrix or add a parallel build path.
- The system must not require network access, a git checkout, or any runtime lookup to determine the version. **Why**: the version is fixed when the binary is produced (stamped or recorded in build info); a runtime lookup would break the self-contained guarantee and make `--version` fail offline.
- The system must not fail the build when no version is supplied. **Why**: local development builds and snapshot builds run with no release tag; a missing supplied version is the normal local case and must resolve to the build-info or placeholder path, not abort the build.

---

## Integration Boundaries

- **Self-Contained Executable Build (021, upstream)**: provides the single build entry point with an empty version-injection seam. Version Embedding fills that seam so a supplied version is stamped into the binary; it does not add a second build path.
- **Automated Release Pipeline (022, upstream — version source)**: reads the version from a published release's semver tag and supplies it to the build. 022 owns *where the number comes from*; this feature owns *embedding it and resolving it at runtime*. The pipeline expects that a version it supplies is exactly what `--version` reports.
- **Help & Version (003, downstream — render surface)**: renders the resolved version via `--version` and the `version` command, both producing identical bare-string output. This feature supplies the value; 003 prints it. 003's `[ASSUMED]` "clear placeholder when version unset" is satisfied here.
- **Go toolchain / module build info (system)**: for source installs (`go install`), the Go toolchain records the module version in the binary's build info; this feature reads that as the fallback when no version was stamped. No network or VCS access at runtime — the value is already embedded by the toolchain at build time.

---

## Driving Scenarios

### Happy path

**Scenario: a release build reports the stamped version**
Given the build is invoked with the release version `v1.4.0` supplied
When the produced binary is asked for its version via `glassfrog --version`
Then it reports `v1.4.0`
And `glassfrog version` reports the same `v1.4.0`.

**Scenario: a pre-release version is embedded verbatim**
Given the build is invoked with the release version `v1.4.0-rc.1` supplied
When the produced binary is asked for its version
Then it reports `v1.4.0-rc.1`, with the pre-release suffix preserved exactly.

**Scenario: a tagged source install reports the recorded module version**
Given the CLI was installed with `go install` from a tagged module version `v1.3.2`, with no version supplied to the build
When the installed binary is asked for its version
Then it reports `v1.3.2` from the module build info Go recorded.

### Error scenarios

**Scenario: no embedded version and no build info reports the placeholder**
Given a binary built with no version supplied and with no module build info available
When it is asked for its version
Then it reports a clear, non-empty development placeholder
And it does not report an empty string.

**Scenario: the version determination never reaches out at runtime**
Given a produced binary running on a clean host with no network and no git checkout
When it is asked for its version
Then it resolves and reports a version without any runtime lookup
And it does not fail for lack of network or VCS access.

### Edge cases

**Scenario: an embedded version overrides recorded build info**
Given a binary that was both built with version `v1.4.0` supplied and carries module build info recording a different version
When it is asked for its version
Then it reports `v1.4.0` — the explicitly embedded version wins over build info.

**Scenario: an untagged source install reports the pseudo-version verbatim**
Given the CLI was installed with `go install ...@latest` resolving to an untagged commit, so Go recorded a pseudo-version like `v0.0.0-20260101120000-abc123def456`
When the installed binary is asked for its version
Then it reports that pseudo-version verbatim, identifying the exact commit.

**Scenario: a plain local build reports Go's development marker**
Given a binary produced by a plain local `go build` with no version supplied, where Go records the module version as `(devel)`
When it is asked for its version
Then it reports `(devel)` verbatim, surfaced as Go recorded it.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the resolution precedence holds in order**
Given the three version sources — embedded version, recorded build info, placeholder
When the version is resolved for a binary that has any combination of them
Then an embedded version is reported if present; otherwise recorded build info if present; otherwise the placeholder — in that exact order, with no source skipped or reordered.

**Scenario: no formatting or render logic leaks into version resolution**
Given the produced spec and any later implementation
When the version-resolution surface is inspected
Then it produces a version *value* and does not decide how `--version` is printed, nor add commit/date build metadata to it — rendering remains Help & Version's (003) concern.

**Scenario: version output is never empty**
Given any way the CLI can be built or installed — release build, tagged source install, untagged source install, plain local build, or a build with no module info
When `--version` is invoked
Then the output is a non-empty string in every case.

---

## Assumptions

- **Resolution precedence is embedded → build-info → placeholder** (decision): confirmed during defining — an explicitly stamped version is authoritative; the Go-recorded module version is the source-build fallback; the placeholder is the last resort. Recorded here for traceability.
- **Recorded versions are surfaced verbatim** (decision): confirmed during defining — `(devel)`, pseudo-versions, and real tags from build info are each reported as-is rather than normalized, because each is the most accurate available identifier of the running build.
- **Embedding via Go build-time injection** *(technical)*: the supplied version is stamped using the Go toolchain's link-time variable injection (the `-X` ldflags seam 021 left empty in the shared build config), and the fallback reads `runtime/debug` module build info. This is the conventional Go mechanism and matches the foundational Go + cobra stack; the spec fixes the observable behavior, not the exact symbol path.
- **Placeholder shape** *(technical)*: a clear non-release development marker (the codebase currently uses a `0.0.0-dev`-style sentinel). The exact literal is an implementation detail; the behavioral requirement is "non-empty and recognizably not a release."

---

## Ambiguity Warnings

_None remaining — the three behavioral forks (resolution precedence, the `(devel)` build-info case, and pseudo-version handling) were resolved during the defining conversation and recorded under Assumptions._
