# Tasks: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/self-contained-executable-build.feature

---

## Dependency Graph

Increment 1: Build configuration and verification (3 tasks, spans plan Phases 1–2) — single RED→GREEN increment

3 tasks total | 1 increment (plan Phases 1–2) | Builder: pipeline

> **One increment, one PR (CONSTITUTION VII — Working Software).** The `.goreleaser.yaml` config is the only implementation in this feature, and its verifying tests are the config-guard and the self-containment check. They land together in a single PR — never as a config-only or a test-only increment. **Order is RED→GREEN (CONSTITUTION IV):** the config-guard test (T001) is written first and fails with no config, then the config (T002) makes it pass, and the self-containment verification (T003) closes the increment.

---

## Branching Guidance

**Pipeline mode**: `spec/021-self-contained-executable-build/base` → a single task branch `spec/021-self-contained-executable-build/task-1` carrying the whole increment (config + both tests) in one PR. The T-numbers below are RED→GREEN steps *within* that one increment, not separate PRs.

---

## Increment 1: Build configuration and verification (spans plan Phases 1–2) [Shared]

Single RED→GREEN increment — all three tasks ship in one PR. (Plan-reference fields below cite plan.md's Phase 1/Phase 2; this single increment deliberately spans both.)

- [x] **T001** [Shared] Config-guard test for the build matrix (RED) — `internal/build/config.go` (CheckConfigGuard) + `config_guard_test.go`; change-detector rigor (missing target fails as loudly as an extra), cgo + 4-target drift cases
  - **Scope**: A Go test that reads `.goreleaser.yaml` and asserts the build matrix is exactly the four supported targets and `CGO_ENABLED=0`. Change-detector rigor: a missing target fails as loudly as an extra one. Written first — it fails until T002 adds the config.
  - **Acceptance criteria**:
    - Fails when a target outside the four is declared (e.g. a Windows target), naming it.
    - Fails when `CGO_ENABLED=0` is absent or set to a non-zero value.
    - Fails when any one of the four required targets is missing.
    - Passes on exactly the four supported targets with cgo disabled.
  - **Dependencies**: None (RED — authored before the config exists)
  - **Plan reference**: Phase 2: Self-containment verification, ADR-2 (config-guard)
  - **Scenario references**: self-contained-executable-build.feature: "The matrix is exactly the four supported targets", "Config drift to enabled cgo is rejected", "An unsupported target in the build config is rejected"
  - **Interface references**: interface-spec.md: Error Communication (config-guard constraint violations)

- [x] **T002** [Shared] Add the GoReleaser build-only configuration (GREEN) — `.goreleaser.yaml` (build-only, 4-target matrix, CGO_ENABLED=0, -trimpath, empty ldflags 023 seam); verified `goreleaser build --snapshot --clean` → 4 binaries + dist/artifacts.json, `--single-target` → host only; `/dist/` gitignored
  - **Scope**: Add `.goreleaser.yaml` at the repo root carrying a single `builds` entry for `glassfrog` (`main: .`, `binary: glassfrog`, `env: [CGO_ENABLED=0]`, `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `flags: [-trimpath]`, `ldflags` left empty as the 023 seam) and `version: 2` + `project_name: glassfrog`. No `archives`, `checksum`, `release`, or `brews` sections. Document the two invocations (`goreleaser build --snapshot --clean` and `--single-target`). Turns T001 GREEN.
  - **Acceptance criteria**:
    - T001 (config-guard) passes once this config lands.
    - `goreleaser build --snapshot --clean` produces one `glassfrog` binary for each of darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 under `dist/`, plus `dist/artifacts.json`.
    - `goreleaser build --snapshot --clean --single-target` produces only the host-platform binary.
    - The config contains no release/archive/checksum/brews sections (build-only).
    - `ldflags` carries no version logic (left as the 023 injection point).
  - **Dependencies**: T001 (GREEN follows the RED config-guard test)
  - **Plan reference**: Phase 1: Build configuration, ADR-1
  - **Scenario references**: self-contained-executable-build.feature: "The release build produces all four target binaries", "The local build produces a runnable host binary", "Foreign-target binaries build from any host", "A failed target fails the whole release build"
  - **Interface references**: interface-spec.md: Build configuration file `.goreleaser.yaml`; Build invocations; `dist/` output contract

- [x] **T003** [US1] Add the self-containment verification test (run + OS-only linkage) — `internal/build/{linkage,hostbinary}.go` + `selfcontainment_test.go`; dist-artifact-preferred with host-build fallback (runs without goreleaser), `version` execute probe (exit 0, no network), per-platform OS-only allowlist; 9 BDD scenarios un-@wip'd in `selfcontained_bdd_test.go` (3 @validation held)
  - **Scope**: A Go test that obtains a host-target `glassfrog` binary — preferring a `dist/artifacts.json`-listed artifact, else building the host target on the fly with `CGO_ENABLED=0` — executes `glassfrog version` asserting exit 0, then inspects the binary's dynamic-library linkage against a per-platform OS-only allowlist (Linux: statically linked / loader only; macOS: only `/usr/lib/**` + `/System/Library/**`). Reuses the subprocess-exec pattern from `internal/cli/smoke_test.go`. Closes the increment alongside the config it verifies.
  - **Acceptance criteria**:
    - Passes for a self-contained host binary (executes, OS-only linkage).
    - Fails and names the offending dependency when a binary links a library outside the OS allowlist.
    - Runs under `go test ./...` without `goreleaser` installed (host-build fallback engages).
    - The execute probe makes no network call.
  - **Dependencies**: T002 (verifies the build's output; ships in the same PR)
  - **Plan reference**: Phase 2: Self-containment verification, ADR-2; Self-Containment Verification Design
  - **Scenario references**: self-contained-executable-build.feature: "A produced binary runs on a clean host", "The self-containment check rejects a binary with a runtime dependency", "A binary runs only on its own target's host", "A produced binary needs only the API at runtime"
  - **Interface references**: interface-spec.md: Self-containment verification (test surface); OS-only linkage allowlist
