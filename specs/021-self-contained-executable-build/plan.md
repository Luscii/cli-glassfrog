# Plan: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Role**: Shaper
**Inputs**: spec.md (021), PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/LEARNINGS.md`, `.score/memory/DEPRECATION.md`

---

## System Architecture

This feature introduces the repository's first build tooling: a single declarative build configuration that compiles the `glassfrog` source tree into self-contained binaries. There is no runtime component — the "system" is the build itself, and its observable outputs are the produced binaries and a verification that they satisfy CONSTITUTION XII.

Three parts:

- **GoReleaser build configuration** (`.goreleaser.*`, repo root) — a declarative `builds` block describing how to compile `glassfrog`: the source entry point (the module-root main package, `main: .`), `CGO_ENABLED=0`, and the GOOS/GOARCH target matrix (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64). This is the single source of build truth that both a maintainer and (later) 022's CI invoke. It contains **only** the build concern — no archives, checksums, release, or Homebrew sections (those are 022).
- **Build invocations** — the canonical entry point is `goreleaser build --snapshot --clean`. The full matrix is `goreleaser build --snapshot --clean` (snapshot mode, no tag required); the local development build adds `--single-target` (host GOOS/GOARCH only). Both apply the same config, so the local binary and a matrix binary for the same target are produced identically — one mechanism, no second path. Output lands under `dist/`.
- **Self-containment verification** — a Go test that produces the host-target binary, executes it, and inspects its dynamic-library linkage to confirm it depends only on OS-provided libraries. A companion config-guard test asserts the `.goreleaser` build matrix and `CGO_ENABLED=0` have not drifted (the exact four targets, no Windows). This is the CONSTITUTION XII detection made executable, applied per target and proven on at least the host target.

Data/control flow: source tree → `goreleaser build` → per-target binary in `dist/` → self-containment verification runs a produced binary on a clean host of its target and asserts it executes with OS-only dependencies. The build host needs the Go toolchain and the `goreleaser` binary; **neither is a dependency of the produced artifact** — CONSTITUTION XII governs the artifact's runtime, not the build host.

This conforms to the foundational language decision (DECISIONS, 001-command-registration): Go, `CGO_ENABLED=0`, GOOS/GOARCH cross-compilation, with self-containment — not "fully static" linking — as the criterion.

---

## Architecture Decisions

### ADR-1: Adopt GoReleaser in build-only mode as the canonical build mechanism

**Context**: The repo has no build tooling yet. The spec requires one repeatable build entry point that a maintainer and an automated pipeline both invoke (so the binaries can't diverge), producing the four-target matrix and a host-target local build. The spec's assumptions deferred the tool shape to plan. Downstream, 022 (Automated Release Pipeline) and the Homebrew Tap already require GoReleaser for archiving and publishing (FEATURE-MODEL).

**Options considered**:
1. **Makefile wrapping `go build`** — a `build`/`build-all` target looping the matrix with `CGO_ENABLED=0`. Dependency-free and transparent, but 022 would introduce GoReleaser for release, creating a *second* build path (Makefile build vs GoReleaser build) that can drift.
2. **GoReleaser in build-only mode** — a `.goreleaser` `builds` block invoked via `goreleaser build`. One tool spans build through release: 022 extends the same config with `archives`/`release` rather than introducing a parallel build. Cost: adds a build-host tool dependency now and brings a release-tool config file forward ahead of 022.
3. **Shell script over the matrix** — `scripts/build.sh` looping GOOS/GOARCH. No `make` dependency, but reinvents matrix/templating that GoReleaser provides and still leaves 022 to introduce a second mechanism.

**Decision**: Option 2 — GoReleaser, build-only. The decisive factor is the spec's one-build-path requirement read forward: because 022 and Homebrew will use GoReleaser regardless, adopting it now means the release pipeline *extends* one config instead of reconciling two build mechanisms.

In practice: a `.goreleaser` file at the repo root carries a single `builds` entry for `glassfrog` (`main: .`, `CGO_ENABLED=0`, the four-target matrix). The matrix build is `goreleaser build --snapshot --clean` (snapshot mode — no git tag needed); the local dev build is `goreleaser build --snapshot --clean --single-target`. The file carries **no** `archives`, `checksum`, `release`, or `brew` sections — those are 022. The `builds.ldflags` field is the seam 023 will use to inject the version; 021 leaves it free of version logic.

**Consequences**: One tool, one config, build→release with no drift — 022 adds sections rather than a parallel path. The build host must have `goreleaser` installed; this does not affect the artifact's self-containment (XII is about runtime, not build). A `.goreleaser` config exists before its release sections do — the file is explicitly partial, and 022 owns completing it. The self-containment verification must not assume `goreleaser` is installed in a plain `go test` run (see ADR-2).

### ADR-2: Verify self-containment by running and inspecting the host binary, guarded by a config-drift test

**Context**: The spec requires a per-target check — run a produced binary on a clean host of its own target and confirm it executes with no separately-installed dependency — provided here and proven on at least the host target, with cross-target breadth deferred to 022. "Self-containment, not fully-static" is the criterion (DECISIONS 001): a Linux `CGO_ENABLED=0` binary is fully static, but a macOS Go binary always dynamically links the system `libSystem` — which is OS-provided and therefore self-contained. The check must encode "depends only on OS-provided libraries," not "zero dynamic links." A pure config check can't prove the binary runs; a pure run can't prove the absence of a hidden library dependency.

**Options considered**:
1. **Run-only** — execute the binary (e.g. `glassfrog version`) and assert exit 0. Proves it runs on the build host, but the build host isn't clean — an accidentally-introduced library dependency that happens to be present locally would pass.
2. **Run plus linkage inspection plus config guard** — execute the binary and assert exit 0, *and* inspect its dynamic-library linkage against an OS-only allowlist (per platform), *and* a separate test asserting the `.goreleaser` matrix and `CGO_ENABLED=0` have not drifted.
3. **Run the GoReleaser artifact directly** — point the check at `dist/`'s output. Most faithful to the shipped artifact, but couples the check to a prior `goreleaser build` and to `goreleaser` being installed wherever the check runs.

**Decision**: Option 2, with a fallback that absorbs Option 3's fidelity where available. The verification is a Go test (reusing the existing subprocess pattern in `internal/cli/smoke_test.go`): it obtains a host-target binary — preferring a GoReleaser-produced `dist/` binary when present (via a discovery/env hook), else building the host target itself with `CGO_ENABLED=0` so plain `go test` still runs — executes it and asserts exit 0, then inspects linkage for OS-provided libraries only. A companion guard test reads the `.goreleaser` config and asserts the exact four-target matrix and `CGO_ENABLED=0`, failing loudly on drift (e.g. a stray `windows` target or `CGO_ENABLED=1`).

The linkage inspection is per-platform: on Linux, a self-contained binary reports no dynamic dependencies (statically linked / only the dynamic loader); on macOS, only `/usr/lib` + `/System` system libraries are permitted. The allowlist *is* the OS-only criterion.

**Consequences**: Self-containment is proven on the host target in `go test` without requiring `goreleaser` locally, while CI (022) can point the same check at real `dist/` artifacts and run it on additional target hosts. The config guard catches the two highest-value drifts (lost `CGO_ENABLED=0`, an unsupported target) at config level, independent of which arch the runner can execute. The per-platform linkage allowlist is the one piece that must be maintained as supported targets change.

---

## Self-Containment Verification Design

The verification answers one question per target: *does this binary run on a clean host of its target with only OS-provided libraries?* It decomposes into three observable checks:

- **Executes** — the binary returns success on a trivial, no-network invocation (a `version`-class command), confirming the loader and runtime start with nothing extra installed.
- **OS-only linkage** — the binary's dynamic dependencies fall entirely within a per-platform OS allowlist (Linux: none / loader only; macOS: system libraries under `/usr/lib`, `/System`). A dependency outside the allowlist is a XII violation and fails the check.
- **No config drift** — the build config still declares exactly the four supported targets (no Windows, nothing extra) with `CGO_ENABLED=0`.

The first two run against an actual binary (preferring the `dist/` artifact, else a host build); the third runs against the config text. Together they make the spec's "self-containment verification" accord and its error scenario ("the check catches a runtime dependency") concrete. Cross-target *execution* (running the linux/arm64 binary, etc.) is explicitly out of this feature — 022 decides where multi-arch hosts or emulation run the same check.

---

## Cross-cutting Concerns

- **Build-host vs artifact dependency** — the single most important distinction this feature must keep straight. The build host requires Go and `goreleaser`; the *artifact* requires only its OS plus network to the API. The verification asserts the latter; nothing about the former weakens XII. This is called out so 022 and reviewers don't mistake the build-host `goreleaser` requirement for a XII regression.
- **Testing strategy** — Go tests only, consistent with the project's `go test` + godog suites and the subprocess-exec pattern already in `internal/cli/smoke_test.go`. The self-containment test and the config-guard test are unit/integration tests in the Go suite; no new test framework. The config-guard follows the LEARNINGS-noted change-detector rigor (assert presence *and* absence — fail on a missing target as loudly as on an extra one).
- **Configuration** — the `.goreleaser` build matrix, binary name, and `CGO_ENABLED=0` are fixed by this feature. `builds.ldflags` is left open as 023's version-injection seam. Archive/checksum/release/Homebrew configuration is intentionally absent (022).
- **Reproducibility** — `goreleaser build` is invoked in snapshot mode for local/matrix builds so no git tag is required; `-trimpath` (GoReleaser default) keeps builds path-independent across hosts, supporting the cross-host-build requirement.

---

## Implementation Strategy

Two small phases; they could land as one PR, but they are distinct testable units.

**Phase 1 — Build configuration.** Add the `.goreleaser` file with the single `glassfrog` `builds` entry: `main: .`, `CGO_ENABLED=0`, the four-target matrix, `builds.ldflags` left as 023's seam, and no release/archive sections. Confirm `goreleaser build --snapshot --clean` (matrix) and `goreleaser build --snapshot --clean --single-target` (local) produce binaries under `dist/`. Document both invocations.

**Phase 2 — Self-containment verification.** Add the Go test that obtains a host-target binary (dist-artifact-preferred, host-build fallback), executes it, and asserts OS-only linkage per platform; add the config-guard test asserting the four-target matrix and `CGO_ENABLED=0`. Phase 2 depends on Phase 1's config existing.

---

## Risks

- **`CGO_ENABLED=0` drift** — a future edit (or a copied GoReleaser example) re-enables cgo, silently reintroducing a C-library dependency. *Likely-ish over time, high impact.* Mitigation: the config-guard test fails on `CGO_ENABLED != 0`.
- **Unsupported target creep** — a Windows or other target is added to the matrix, implying support the project hasn't committed to (spec non-behavior). *Medium.* Mitigation: the config-guard asserts the exact four-target set and fails on any extra (or missing) target.
- **macOS cross-build fidelity** — darwin binaries cross-compiled on a Linux CI host build fine with `CGO_ENABLED=0` (pure Go, no macOS SDK), but are unsigned/unnotarized. *Low for this feature.* Signing/notarization is a distribution concern (022/channels), not self-containment; noted so it isn't mistaken for a build defect here.
- **Verification fidelity gap** — the host-build fallback compiles with `CGO_ENABLED=0` independently, so it proves the *property* but not that GoReleaser's output matches. *Low.* Mitigation: the dist-artifact-preferred path closes the gap whenever `dist/` exists (local `goreleaser build` first, or CI), and the config-guard pins the GoReleaser flag.
- **Pulling GoReleaser config forward** — 022 might re-decide the build instead of extending the existing `builds` block. *Process risk, medium.* Mitigation: ADR-1 fixes the boundary — 021 owns `builds`; 022 adds `archives`/`release`/`brew` to the same file.

---

## What This Plan Does Not Cover

- **Protocol/structural contracts** — the exact `.goreleaser` field schema, the `dist/` path/filename format, the precise build-invocation flags, and the OS-only linkage allowlist's exact entries are the interface skill's concern (`/score:interface`).
- **Executable scenarios** — Gherkin for the driving scenarios is `/score:scenarios`.
- **Task decomposition** — PR-sized units are `/score:tasks`.
- **Version embedding** — `builds.ldflags` version injection and the `go install` build-info fallback are 023 (Version Embedding); 021 only leaves the seam.
- **Packaging & publishing** — archives, checksums, GitHub Releases, and the Homebrew cask are 022 (Automated Release Pipeline) and later channels; this plan deliberately omits those GoReleaser sections.
- **Cross-target execution of the check** — where/how the self-containment check runs against non-host targets (multi-arch runners, emulation) is 022's CI concern.
