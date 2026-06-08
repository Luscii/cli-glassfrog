# Plan: Version Embedding

**Feature**: 023-version-embedding
**Role**: Shaper
**Inputs**: spec.md (023), PROJECT.md, `.score/memory/DECISIONS.md` (relevant precedent: 001 Go/cgo, 003 cobra version-unify, 021 GoReleaser build-only + ldflags seam, 022 release completion), `.score/memory/DEPRECATION.md` (no relevant entries), `.score/memory/LEARNINGS.md` (pure-function + injected-seam test pattern). No CONSTITUTION load (Tier 2). No SOUL.md present.

---

## System Architecture

Version Embedding adds two small, co-located pieces to the existing CLI, and changes nothing about how the binary is built or how `--version` is printed:

1. **A runtime version resolver** in `internal/cli` — a pure function that, given the build-time-injected value and Go's recorded module build info, returns the single version string the CLI reports. It implements the spec's precedence: injected → build-info → placeholder. It is co-located with the existing version handling (`version.go`, `helpversion.go`) because there is no import-cycle reason to split it out (`runtime/debug` is stdlib) and because 003's version-unify property — `--version` flag and `version` command emit the *same* value — is preserved by having both wiring sites read the same resolved value.

2. **A build-time injection seam** in `.goreleaser.yaml` — the `builds.ldflags` entry that 021 deliberately left empty and 022 deliberately did not touch. 023 fills it with a `-X` linker flag that stamps the version into the package-level `version` variable. GoReleaser derives the version from git (the published tag for a `goreleaser release`, a snapshot version for `goreleaser build --snapshot`), so the same seam serves both the release pipeline (022) and local/CI snapshot builds (021).

**Flow** — when the CLI is asked for its version (via the `--version` flag or the `version` command, both routed by 003):
```
            ┌─────────────────────────────────────────────┐
            │ resolveVersion(injected, buildInfo, ok)      │
            │                                              │
 injected ──┤ 1. injected != ""        → return injected   │
 (ldflags)  │ 2. ok && Main.Version!="" → return verbatim   │──→ root.Version
            │ 3. otherwise              → return placeholder │     & version cmd
 build info ┤                                              │     (identical value)
 (toolchain)└─────────────────────────────────────────────┘
```
The injected value comes from the `-X` ldflags at build time (empty when built by plain `go build`/`go install`). The build info comes from `runtime/debug.ReadBuildInfo()` — the module version the Go toolchain recorded into the binary at build time, surfaced verbatim (`vX.Y.Z`, a pseudo-version, or `(devel)`). No runtime network or VCS lookup occurs; both inputs are baked into the binary.

The two existing wiring sites in `configureHelpAndVersion` (sets `root.Version`) and `newVersionCommand` (prints the value) change from reading the raw `version` var to reading the resolved value.

---

## Architecture Decisions

### ADR-1: Resolve the reported version with a pure three-tier function in `internal/cli`

**Context**: The spec fixes a precedence — an explicitly embedded version wins; absent that, the Go-recorded module version is used verbatim; absent that, a development placeholder. This logic must be deterministic, offline, and unit-testable, and must feed both `--version` and the `version` command identically (003's version-unify property). PROJECT.md stack is Go + cobra; `runtime/debug.ReadBuildInfo()` is the toolchain-native source of recorded module version.

**Options considered**:
1. **Pure function `resolveVersion(injected string, info *debug.BuildInfo, ok bool) string` in `internal/cli`, with a thin production caller reading `debug.ReadBuildInfo()`** — precedence is unit-tested offline with crafted inputs; the production wiring passes the real build info. Co-located with 003's version code. Matches the LEARNINGS pure-function-over-injected-seam pattern (cf. `formatMe`/`validateInclude`).
2. **Resolve inline in the cobra wiring** (compute in `configureHelpAndVersion`) — fewer moving parts, but the precedence logic becomes untestable without assembling cobra and is duplicated across the two wiring sites, risking drift from the byte-identical-output requirement.
3. **New `internal/version` package** — clean separation, but there is no import-cycle or distinct-test-shape reason to justify it (unlike `internal/build`/`internal/output`/`internal/render`); it would scatter 003's version handling across two packages.

**Decision**: Option 1. A pure `resolveVersion(injected string, info *debug.BuildInfo, ok bool) string` plus a thin `resolvedVersion()` that calls `debug.ReadBuildInfo()` and delegates. Both live in `internal/cli` alongside the existing version handling.

`resolveVersion` returns `injected` when it is non-empty; otherwise, when build info is available and `info.Main.Version` is non-empty, it returns that value **verbatim** (no trimming, no normalization — `(devel)`, pseudo-versions, and real tags all pass through unchanged, per the spec's confirmed decisions); otherwise it returns the placeholder constant. `configureHelpAndVersion` sets `root.Version = resolvedVersion()` and `newVersionCommand` prints `resolvedVersion()`, so the two request forms resolve to one value.

**Consequences**: Precedence and the verbatim-passthrough are testable offline against crafted `*debug.BuildInfo` values without building a binary. The version-unify property holds because both sites read the same deterministic resolver. `resolveVersion` is the single point that decides *which value* — it deliberately contains no formatting (that stays in 003 / cobra's template). Negative: a test that wants to exercise the build-info branch must construct a `*debug.BuildInfo`; the injected and placeholder branches need no build info.

### ADR-2: Inject the version through GoReleaser's `builds.ldflags` into `internal/cli.version`

**Context**: 021 established GoReleaser as the single canonical build mechanism and left `builds.ldflags` empty *as 023's seam* (DECISIONS 2026-06-08); 022 completed `archives`/`checksum`/`release` and explicitly did not touch `builds`/`ldflags`. The package var to stamp already exists: `internal/cli.version`, with the `-X` target spelled out in its doc comment. GoReleaser computes the version from git, so it can supply the value on every invocation.

**Options considered**:
1. **Fill `builds.ldflags` with `-X github.com/Luscii/cli-glassfrog/internal/cli.version=<version-template>`** — uses the reserved seam; one build path for local, snapshot, and release; the value flows from git via GoReleaser with no extra pipeline wiring. 022 already triggers `goreleaser release` on `release: published`, so the published tag reaches the binary automatically.
2. **A separate `Makefile`/script that calls `go build -ldflags`** — would create the second build path 021 explicitly rejected, and 022's release job would have to diverge from the local build.

**Decision**: Option 1. 023 sets the single `builds.ldflags` entry to inject `internal/cli.version`. This is purely additive to the shared `.goreleaser.yaml` — it touches only the line 021/022 reserved, leaving the matrix, `flags: [-trimpath]`, and 022's `archives`/`checksum`/`release` untouched. The `internal/build` config-guard asserts only the matrix + cgo (it parses but does not assert `ldflags`), so filling the seam does not trip the guard.

**Consequences**: A `goreleaser release` (022) and a `goreleaser build --snapshot` (021) both produce version-carrying binaries; the release path needs no new code, only the seam being filled. Plain `go build`/`go install` (no ldflags) leave `version` empty and fall to ADR-1's build-info branch — the desired source-build behavior. The exact GoReleaser template token (`{{ .Tag }}` vs `v{{ .Version }}` vs `-s -w` size flags) is an interface-level detail (see ADR-4 and the interface skill). Negative: the version contract now spans a config file and Go code; the resolver and the ldflags target symbol must name the same package var, so a rename of `internal/cli.version` must update both (mitigated by the doc comment on the var already naming the `-X` path).

### ADR-3: Default the injected `version` var to empty; the placeholder is the resolver's return value

**Context**: Today `internal/cli.version` defaults to `"0.0.0-dev"` (003). For ADR-1's precedence to detect "no version was injected," the default must be distinguishable from an injected value. A non-empty default would make the resolver treat *every* build as "injected" and never consult build info — defeating the `go install` fallback.

**Options considered**:
1. **Default `version = ""`; treat empty as "not injected"; the placeholder (`"0.0.0-dev"`) becomes a constant `resolveVersion` returns at tier 3** — a clean, unambiguous sentinel (ldflags `-X` only ever sets a non-empty value), and the "clear placeholder, never empty" guarantee moves into the resolver where the spec's last-resort tier lives.
2. **Keep a non-empty sentinel default and compare against it** (`if version != "0.0.0-dev"`) — fragile: a release legitimately tagged with that string (however unlikely) would be misread as "not injected," and the sentinel value would be duplicated between the default and the comparison.

**Decision**: Option 1. `var version string` (empty default). `resolveVersion` returns a placeholder constant (`"0.0.0-dev"`, semver-shaped and clearly non-release) when neither an injected value nor usable build info exists.

**Consequences**: 003's `TestVersionDefaultPlaceholder` — which currently asserts the *var default* equals `"0.0.0-dev"` — relocates to assert that `resolveVersion("", nil, false)` returns the non-empty placeholder. This is a legitimate 023 update: 003's own assumption ("version-unset fallback shows a clear placeholder rather than an empty string") is now *satisfied by* this feature, so the property moves from the var to the resolver. The 003 parity test and the help-listing tests that call `runAssembled(t, "0.0.0-dev", …)` keep working — a non-empty injected value resolves to itself. The test harness's `version = ver` pin continues to drive the injected branch end-to-end.

### ADR-4: Preserve the `vX.Y.Z` shape across the injected and build-info sources

**Context**: The build-info fallback yields `v`-prefixed versions (Go records `Main.Version` as `v1.3.2`, `v0.0.0-<ts>-<commit>`, or `(devel)`). The spec's driving scenarios report the injected release version as `v1.4.0` (with the `v`). The feature's whole purpose is a *consistent, meaningful* version across install methods, so the two sources should not differ in prefix shape (a release reporting `1.4.0` while a source install reports `v1.3.2` would be a gratuitous inconsistency).

**Options considered**:
1. **Inject the tag in its `v`-prefixed form so injected and build-info versions share one shape** — consistent `--version` output regardless of how the CLI was obtained; matches the spec scenarios verbatim.
2. **Inject GoReleaser's de-prefixed `.Version` (`1.4.0`)** — diverges from the build-info shape and from the spec scenarios; would require either documenting the inconsistency or stripping the `v` from build-info too (extra logic the spec's "verbatim" decision forbids).

**Decision**: Option 1 — inject the version in `vX.Y.Z` shape. The resolver still passes build-info through verbatim (ADR-1); this decision only constrains the *injection* shape so it aligns. The exact GoReleaser template token that yields the `v` prefix (`{{ .Tag }}`, `v{{ .Version }}`) is pinned by the interface skill against GoReleaser's template semantics; the architectural requirement is "injected version carries the same `vX.Y.Z` shape the build-info fallback produces."

**Consequences**: `--version` reads consistently across release builds, tagged source installs, and `@latest` pseudo-version installs. Snapshot builds report whatever shape GoReleaser's snapshot template yields (a dev-flavored version) — acceptable, as snapshots are non-release. Negative: ties the injection token choice to GoReleaser's `.Tag`/`.Version` distinction, which the interface skill must get right (a wrong token would silently drop or double the `v`); a scenario asserting the `v` prefix guards against that.

---

## Cross-cutting Concerns

**Determinism & offline operation**: Version resolution must never touch the network or a VCS at runtime — both inputs (injected ldflags value, recorded build info) are embedded at build time. This is a spec non-behavior and a CONSTITUTION XII (self-containment) corollary; the resolver uses only `runtime/debug` (stdlib) and a package var.

**Output byte-identity (003 boundary)**: 023 supplies the version *value*; 003/cobra owns formatting (bare string, `{{.Version}}\n` template, no `glassfrog version X` prefix). Both `--version` and `version` must remain byte-identical — preserved by routing both through the one resolver. 023 must not add commit/date metadata to the output (spec non-behavior; 003's bare-string scope).

**Testing strategy** (mirrors the LEARNINGS pure-function pattern):
- *Pure resolver unit tests*: precedence across all branches with crafted inputs — injected non-empty wins over build info; empty injected + build-info `v1.3.2` → `v1.3.2`; empty + pseudo-version → verbatim; empty + `(devel)` → `(devel)` verbatim; empty + no build info → non-empty placeholder; output never empty in any branch.
- *Wiring tests* (extend 003's `runAssembled`): a non-empty injected version is reported by both `--version` and `version`, byte-identical (003 parity preserved); these assert the resolved value flows to `root.Version` and the command.
- *Config-guard*: the existing `internal/build` `TestConfigGuard_RealConfig` continues to pass after the ldflags edit (guard ignores `ldflags`); optionally a focused assertion that the real config's `ldflags` injects `internal/cli.version` keeps the seam from silently regressing to empty.
- *003 test migration*: `TestVersionDefaultPlaceholder` is rewritten to target the resolver's placeholder (ADR-3).

**Configuration**: The only configurable input is the build-time version (via ldflags/git tag). The placeholder string is a hardcoded constant — not a knob.

---

## Implementation Strategy

A single phase — the feature is small and the two pieces are interdependent (the resolver's empty-injected branch is only meaningful once the var default is empty, and the ldflags seam only matters once the resolver reads the injected value). Suggested ordering within the phase:

1. Introduce the pure `resolveVersion` + `resolvedVersion()` and the placeholder constant; change `var version` to an empty default (ADR-1, ADR-3).
2. Re-point the two wiring sites (`configureHelpAndVersion`, `newVersionCommand`) at the resolved value; migrate `TestVersionDefaultPlaceholder` and add the resolver unit tests (ADR-1, ADR-3).
3. Fill `.goreleaser.yaml` `builds.ldflags` to inject `internal/cli.version` in `vX.Y.Z` shape (ADR-2, ADR-4); confirm the config-guard still passes.

No phase dependencies beyond this internal ordering. The tasks skill may keep this as one PR-sized unit or split the code change from the build-config change; both are small.

---

## Risks

- **Build-info absent in some build modes** (low likelihood, low impact): a binary built in a way that yields no `ReadBuildInfo()` (e.g. unusual GOPATH-mode or stripped-of-buildinfo builds) falls to the placeholder. This is the designed last-resort tier, not a failure — but it means a misconfigured build could ship reporting the placeholder. Mitigation: the release path always injects via ldflags (tier 1), so a real release never relies on build info; the placeholder only appears for non-release builds that also lack build info.
- **GoReleaser template token mismatch for the `v` prefix** (medium likelihood, low impact): choosing `{{ .Version }}` (de-prefixed) instead of `{{ .Tag }}` would report `1.4.0` while source installs report `v1.4.0`. Mitigation: a driving/validation scenario asserts the `v`-prefixed shape; the interface skill pins the token against GoReleaser's documented semantics.
- **Silent ldflags regression** (low likelihood, medium impact): a future edit to `.goreleaser.yaml` could blank the `ldflags` entry (back to 021's empty seam), and the config-guard would not catch it (it ignores `ldflags`). Releases would then fall to build-info/placeholder and report a wrong version. Mitigation: a focused config assertion that the real config injects `internal/cli.version` (noted in testing strategy).
- **`-X` symbol path drift** (low likelihood, medium impact): renaming `internal/cli.version` without updating the ldflags target makes injection a silent no-op (Go ignores `-X` for an unknown symbol), and the binary would report build-info/placeholder. Mitigation: the var's doc comment names the `-X` path; the focused config assertion above also catches a stale symbol if paired with a release-shape build test.

---

## What This Plan Does Not Cover

- **Exact GoReleaser template token and ldflags string** (`{{ .Tag }}` vs `v{{ .Version }}`, whether to add `-s -w`): the interface skill pins the literal `builds.ldflags` value against GoReleaser's template semantics. This plan fixes the *target symbol*, the *injected shape* (`vX.Y.Z`), and that the seam is the reserved `builds.ldflags`.
- **The exact resolver signature and placeholder literal as a published contract**: the interface skill specifies the `resolveVersion` signature and the placeholder constant value. This plan fixes the behavior (three-tier precedence, verbatim passthrough, non-empty placeholder).
- **Executable scenarios**: the scenarios skill concretizes the spec's driving scenarios into Gherkin.
- **Task decomposition**: the tasks skill breaks the single phase into PR-sized unit(s).
- **The release pipeline trigger and packaging** (022) and **the build matrix/self-containment** (021): owned by their specs; 023 only fills the version seam between them.
