# Validate: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Round**: 1 of 3
**Date**: 2026-06-08
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/runtime-dependent-distribution/self-contained-executable-build.feature, PROJECT.md
**Implementation files**: 4 in `internal/build/` (config.go, linkage.go, hostbinary.go + 3 test files), `.goreleaser.yaml` (repo root), `.gitignore` (dist/ exclusion)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 conformance dimensions checked, 5 passed, 0 findings. The sixth row, Validation scenarios, is counted separately (held-out from the Builder): 3 of 3 satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 driving scenarios covered)

All driving scenarios have identifiable implementation code paths. The build-behavior scenarios (matrix, atomic failure, cross-compile) are traced to the parsed `.goreleaser.yaml` contract — the plan's in-process proxy (ADR-2), since `go test` must run without `goreleaser` (T003 host-build fallback). Their behavior was additionally confirmed empirically during implementation: `goreleaser build --snapshot --clean` produced all four target binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) plus `dist/artifacts.json`.

| Scenario | Status | Implementation |
|---|---|---|
| local build produces a runnable host binary | ✓ Covered | `hostbinary.go:97` buildHostBinary + `selfcontainment_test.go` execute/linkage; BDD `selfcontained_bdd_test.go:whenLocalBuild` |
| release build produces the full matrix | ✓ Covered | `.goreleaser.yaml:30-35` (goos×goarch); guard `config.go:165` diffTargetSet; verified `dist/artifacts.json` = 4 Binary entries |
| a produced binary runs on a clean environment | ✓ Covered | `hostbinary.go:30` HostBinary + `selfcontainment_test.go:TestSelfContainment_HostBinary` (version exit 0 + OS-only linkage) |
| self-containment check catches a runtime dependency | ✓ Covered | `linkage.go:18` osOnlyViolations + `TestSelfContainment_RejectsForeignDependency` (names offending dep) |
| a failed target fails the whole release build | ✓ Covered | single `builds` entry (`.goreleaser.yaml:19-20`) → GoReleaser atomic; BDD `thenAtomicBuild` |
| cross-compilation from a foreign host | ✓ Covered | `CGO_ENABLED=0` (`.goreleaser.yaml:27`); empirically built all 4 targets from darwin/arm64 host |
| self-containment is per-target, not universal | ✓ Covered | `linkage.go:30` isOSProvided (per-platform allowlist); BDD `thenPerTargetAllowlist` |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 checked tasks conformant)

| Task | Criteria | Evidence |
|---|---|---|
| T001 (config-guard) | fails on extra/missing target naming it; fails on cgo≠0; passes on the four | `config_guard_test.go:TestConfigGuard_Drift` — all 7 subcases pass; `diffTargetSet`/`checkCgoDisabled` name the offender |
| T002 (config) | T001 passes; matrix build → 4 binaries + artifacts.json; single-target → host only; no release/archive/checksum/brews; ldflags no version logic | `.goreleaser.yaml` build-only (grep confirms no forbidden sections); empirically verified both invocations; ldflags is `[""]` |
| T003 (self-containment) | passes for self-contained binary; fails naming foreign dep; runs without goreleaser; execute probe no network | `TestSelfContainment_*` pass under sandbox (no goreleaser/network); probe is `version` (offline) |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Evidence |
|---|---|---|
| `.goreleaser.yaml` top-level keys (version 2, project_name, builds) | ✓ Conformant | `.goreleaser.yaml:16-20` |
| builds entry fields (id, main `.`, binary, env CGO_ENABLED=0, goos, goarch, flags -trimpath, ldflags empty) | ✓ Conformant | `.goreleaser.yaml:20-44` — matches the interface field table exactly |
| Keys intentionally absent (archives/checksum/release/brews) | ✓ Conformant | grep confirms none present (build-only) |
| `dist/artifacts.json` manifest-first discovery | ✓ Conformant | `hostbinary.go:46` discoverDistBinary parses the manifest (path/goos/goarch/type) |
| OS-only linkage allowlist (Linux loader-only; macOS /usr/lib + /System/Library) | ✓ Conformant | `linkage.go:30-44` isOSProvided; real host binary links only `/usr/lib/**` + `/System/Library/**` |
| Error communication (config-guard + verification failure modes) | ✓ Conformant | each row maps to a passing test assertion |

---

## Non-Behavior Absence

**Status**: Pass (all 5 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No package/archive/checksum/publish | ✓ Absent | `.goreleaser.yaml` has no archives/checksum/release/brews; no archiving code |
| No version injection | ✓ Absent | ldflags empty (`.goreleaser.yaml:43-44`); built binary prints `0.0.0-dev` (default), no `-X` |
| No Windows / beyond four targets | ✓ Absent | goos `[darwin, linux]` only; config-guard rejects windows (`TestConfigGuard_Drift`) |
| Must not require a particular host OS/arch | ✓ Absent | `CGO_ENABLED=0`; all four targets built from a single darwin/arm64 host |
| Binary requires nothing beyond OS + network | ✓ Absent | self-containment test asserts OS-only linkage; version runs offline |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 non-`@validation` scenarios referenced by checked tasks had their `@wip` tags removed and run green in the godog suite (`selfcontained_bdd_test.go`: 9 scenarios / 36 steps pass). The 3 `@validation` scenarios correctly **retain** `@wip` — they are held out from the Builder by design (spec.md § Validation Scenarios) and are traced independently in the section below, matching the established convention (the same stance the 020 suite takes).

**Note (transparency, not a finding)**: T001 and T003 list two of the held-out `@validation` scenarios in their *Scenario references* fields ("the matrix is exactly the four declared targets", "the artifact's only external need is the API"). A strict literal reading of this dimension ("@wip on scenarios referenced by checked tasks should have been removed") would flag them — but those references are informational (the task's verification *relates* to those scenarios), and the held-out `@validation` convention takes precedence. Keeping them `@wip` is correct, not a lifecycle gap.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

These were held out from the Builder. Each is traced independently to a code path; `@validation` BDD scenarios are not executed by the suite (held out), so the trace is by inspection of the supporting logic.

| Scenario | Status | Trace |
|---|---|---|
| the matrix is exactly the four declared targets | ✓ Satisfied | `config.go:115` CheckConfigGuard asserts the closed set (no windows, none missing); `TestConfigGuard_RealConfig` passes against the shipped config; empirical full build produced exactly 4 binaries |
| the artifact's only external need is the API | ✓ Satisfied | `linkage.go:18` osOnlyViolations confirms every dynamic dep is OS-provided (no installed library); `version` probe runs offline → only unmet need is network to the API |
| one build entry point, no second path | ✓ Satisfied | A single `.goreleaser.yaml` `builds` entry; both `--single-target` (local) and full matrix apply the identical config — no Makefile/script second path exists in the repo |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 3 held-out validation scenarios are satisfied through inspection. The implementation conforms to the specification: the build produces the four self-contained targets from one source tree via a single `.goreleaser.yaml` entry point, the config-guard enforces the closed matrix and `CGO_ENABLED=0` with change-detector rigor, and the self-containment verification runs (with a goreleaser-free host-build fallback), executes the binary, and inspects OS-only linkage. The deferred concerns (packaging/publishing → 022, version embedding → 023) are correctly absent, with the `ldflags` seam left open for 023.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 021 is closed. When 022 (Automated Release Pipeline) lands, it should *extend* this `.goreleaser.yaml` additively (archives/checksum/release/brews) and may promote the build-behavior scenarios from config-proxy assertions to artifact-level CI checks across target hosts.
