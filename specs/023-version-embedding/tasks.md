# Tasks: Version Embedding

**Feature**: 023-version-embedding
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/version-embedding.feature

---

> **Cross-spec gate (satisfied)**: every task here builds on **021 Self-Contained Executable Build** — its `.goreleaser.yaml` `builds.ldflags` entry (left empty *as this feature's seam*) and the existing `internal/build` config-guard (extended by T002). 021 has **landed** (#54 on main), so these tasks are ready now. **022 Automated Release Pipeline is not a dependency**: it edits disjoint sections of the same `.goreleaser.yaml` (`archives`/`checksum`/`release`), not `builds`/`ldflags` — see the merge note in Branching Guidance.

## Dependency Graph

Phase 1: Version Embedding (2 tasks, depends on 021)

2 tasks total | 0 phases parallelizable | Builder: pipeline (single spec)

T001 (resolution) → T002 (injection seam). T002 depends on T001 because the build-time-injected value is only meaningful once the resolver reads it with the injected-wins precedence and the `version` var carries the empty "not-injected" default.

## Branching Guidance

**Pipeline mode**: `spec/023-version-embedding/base` → `spec/023-version-embedding/task-1`, `task-2`.

The `base` branch is cut from a main that contains 021 (the empty `builds.ldflags` seam and the `internal/build` config-guard — landed in #54). **Merge note**: both 022 and 023 extend the single `.goreleaser.yaml`, in *disjoint* sections — 022 adds `archives`/`checksum`/`release`; 023 fills `builds[0].ldflags`. Whichever lands second rebases onto the other; there is no logical conflict, only a possible textual one in the same file. Other distribution specs (027, 030, 036, 037) may build in parallel on their own base branches.

---

## Phase 1: Version Embedding

- [ ] **T001** [Shared] Resolve the reported version with a pure three-tier function, re-point the 003 wiring, default the injected var to empty
  - **Scope**: In `internal/cli` (alongside the existing `version.go`/`helpversion.go`): add a pure `resolveVersion(injected string, info *debug.BuildInfo, ok bool) string` implementing the precedence — `injected` when non-empty; else `info.Main.Version` **verbatim** when `ok && info.Main.Version != ""` (no trim/normalize — `(devel)`, pseudo-versions, real `vX.Y.Z` tags all pass through); else an unexported placeholder constant `"0.0.0-dev"`. Add a thin `resolvedVersion()` that calls `runtime/debug.ReadBuildInfo()` and delegates. Change `var version` from `"0.0.0-dev"` to `""` (the "not-injected" sentinel). Re-point both 003 sites to the resolved value: `configureHelpAndVersion` → `root.Version = resolvedVersion()`; `newVersionCommand`'s `RunE` prints `resolvedVersion()`. Migrate 003's `TestVersionDefaultPlaceholder` to assert `resolveVersion("", nil, false)` returns the non-empty placeholder (not the var default). Add resolver unit tests covering every precedence branch and the never-empty invariant. Do **not** add any formatting, commit/date metadata, or runtime network/VCS lookup.
  - **Acceptance criteria**:
    - `resolveVersion` returns: the injected value when non-empty; `Main.Version` verbatim when injected is empty and build info is present (asserted for a real tag, a pseudo-version, and `(devel)`); the `"0.0.0-dev"` placeholder when both are absent — and never returns an empty string.
    - `--version` and the `version` command produce byte-identical output, both reading `resolvedVersion()` (003 version-unify parity preserved); a non-empty injected value is reported by both.
    - `var version` defaults to `""`.
    - All existing 003 tests pass, with `TestVersionDefaultPlaceholder` migrated to target the resolver's placeholder rather than the var default.
    - Implementation and its tests ship in the same PR (CONSTITUTION VII).
  - **Dependencies**: 021 (Self-Contained Executable Build) — landed
  - **Plan reference**: Implementation Strategy steps 1–2; ADR-1 (pure resolver), ADR-3 (empty default + placeholder)
  - **Scenario references**: version-embedding.feature: "The version is resolved by a fixed precedence"; "Version output is never empty for any build or install path"; "Version resolution produces a value and leaves formatting to Help & Version"; "A tagged source install reports the recorded module version"; "An untagged source install reports the pseudo-version verbatim"; "A plain local build reports Go's development marker"; "A build with no embedded version and no build info reports the placeholder"; "An embedded version wins over recorded build info"; "Version determination needs no network or VCS at runtime"
  - **Interface references**: interface-spec.md: Module structural contract (`version`, `resolveVersion`, `resolvedVersion`, placeholder); Output-value contract

- [ ] **T002** [US3] Fill the `.goreleaser.yaml` `builds.ldflags` injection seam and guard it against regression
  - **Scope**: Set `.goreleaser.yaml` `builds[0].ldflags` to `["-X github.com/Luscii/cli-glassfrog/internal/cli.version=v{{ .Version }}"]` — the `v{{ .Version }}` token restores the `vX.Y.Z` shape (GoReleaser strips the `v` from `.Version`) and keeps snapshots honest (a `v`-prefixed snapshot version, not a stale `.Tag`). Do **not** touch the `builds` matrix, `env` (`CGO_ENABLED=0`), `flags` (`-trimpath`), or 022's `archives`/`checksum`/`release` sections. Adding `-s -w` is not required and out of this seam's contract. Add a focused config-regression test in `internal/build` (alongside the config-guard) asserting the **real** config's `builds[0].ldflags` injects `internal/cli.version` — the matrix config-guard ignores `ldflags`, so this is the guard against a blanked seam or a stale `-X` symbol path.
  - **Acceptance criteria**:
    - `goreleaser build --snapshot --clean --single-target` produces a host binary whose `--version` reports the GoReleaser-derived, `v`-prefixed version (e.g. `v…`) — **not** the `0.0.0-dev` placeholder.
    - The `builds` matrix, `env`, and `flags` are byte-unchanged from 021, and the existing `internal/build` `TestConfigGuard_RealConfig` still passes.
    - The new config-regression test fails loudly if `builds[0].ldflags` is blanked or no longer injects `internal/cli.version`.
    - Implementation and its test ship in the same PR (CONSTITUTION I).
  - **Dependencies**: T001
  - **Plan reference**: Implementation Strategy step 3; ADR-2 (ldflags injection), ADR-4 (`vX.Y.Z` shape)
  - **Scenario references**: version-embedding.feature: "A release build reports the stamped version"; "A pre-release version is reported verbatim"; "A blanked version-injection seam is caught before release"
  - **Interface references**: interface-spec.md: Declarative artifact (`builds[0].ldflags`); Error Communication (regression guard)
  - **Risk**: ⚠️ Template-token mismatch would drop or double the `v` prefix — use `v{{ .Version }}` per interface-spec and verify the produced binary reports the `v`-prefixed shape before relying on it.
