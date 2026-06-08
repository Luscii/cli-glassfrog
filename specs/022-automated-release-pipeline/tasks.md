# Tasks: Automated Release Pipeline

**Feature**: 022-automated-release-pipeline
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/automated-release-pipeline.feature

---

> **Cross-spec gate (satisfied)**: every task here builds on **021 Self-Contained Executable Build** — its `.goreleaser.yaml` `builds` block (extended here) and its `internal/build` self-containment check (`TestSelfContainment_HostBinary`, reused by Phase 3). 021 has **landed** (#54 on main), so these tasks are ready to implement now.

## Dependency Graph

Phase 1: Release Configuration (1 task, depends on 021) [Shared]
Phase 2: Release Workflow (1 task, depends on Phase 1) [Shared]
Phase 3: Cross-Target Verification Gate (1 task, depends on Phase 2) [US1]

3 tasks total | 0 phases parallelizable | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/022-automated-release-pipeline/base` → `spec/022-automated-release-pipeline/task-1`, `task-2`, `task-3`.

The `base` branch is cut from a main that now contains 021 (the `.goreleaser.yaml` `builds` block and the `internal/build` self-containment check — landed in #54). Other distribution specs (023, 027, 030, 036, 037) may build in parallel on their own base branches; this feature only hard-depends on 021.

---

## Phase 1: Release Configuration [Shared]

- [x] **T001** [Shared] Extend `.goreleaser.yaml` with `archives`, `checksum`, and `release` sections (plus config-guard test) — snapshot emits 4 tar.gz + checksums; `builds`/`ldflags` byte-identical; 8 new config-guard drift cases. Both scenario refs are `@validation`, held @wip for /score:validate.
  - **Scope**: Add to 021's `.goreleaser.yaml` (do **not** touch `builds` or `builds.ldflags`): an `archives` entry (one `tar.gz` per target, name template `glassfrog_{{.Version}}_{{.Os}}_{{.Arch}}`, containing the `glassfrog` binary); a `checksum` entry (single sha256 file, default name `glassfrog_{{.Version}}_checksums.txt`); and a `release` entry with `mode: keep-existing`, `draft: false`, no `prerelease`/`make_latest` override. Extend 021's config-guard (`internal/build` — `CheckConfigGuard`, in `internal/build/config_guard_test.go`; same change-detector rigor) to assert the three new sections are present and the build matrix is still exactly the four targets with `CGO_ENABLED=0`.
  - **Acceptance criteria**:
    - `goreleaser release --snapshot --clean --skip=publish` emits exactly four `tar.gz` archives (one per target) plus one sha256 checksums file under `dist/`, named per the templates above.
    - `builds` and `builds.ldflags` are byte-unchanged from 021.
    - The config-guard test fails loudly if any of `archives`/`checksum`/`release` is missing, if a target is added/removed, or if `CGO_ENABLED` ≠ 0.
    - Implementation and its test ship in the same PR (CONSTITUTION I).
  - **Dependencies**: 021 (Self-Contained Executable Build)
  - **Plan reference**: Phase 1: Release configuration; ADR-1
  - **Scenario references**: automated-release-pipeline.feature: "The attached matrix is exactly the four supported targets"; "Every attached archive has a matching checksum entry"
  - **Interface references**: interface-spec.md: `.goreleaser.yaml` sections added by 022; Config-guard extension

## Phase 2: Release Workflow [Shared]

- [x] **T002** [Shared] Add `.github/workflows/release.yml` with the `release: published` trigger, build job, and publish job — 6 scenarios un-@wip'd (proven via CheckReleaseWorkflow structural guard, mirroring 021's config-guard-as-proxy); workflow-guard drift suite added. Note: `on:` parses under JSON key `"true"` (YAML 1.1 coercion) — documented in workflow.go.
  - **Scope**: Create the workflow: `on: release: { types: [published] }`, `permissions: contents: write`. A `build` job (`ubuntu-latest`: `actions/checkout@v4` with `fetch-depth: 0`, `actions/setup-go@v5` with `go-version-file: go.mod`, `goreleaser/goreleaser-action@v6` `version: "~> v2"`, run `goreleaser release --clean --skip=publish`, then `actions/upload-artifact` of `dist/`). A `publish` job (`needs: build`) with `env: { GH_TOKEN: ${{ github.token }} }` (required — `gh` authenticates from the token env var, not from `permissions` alone) that downloads `dist/` and uploads **only the release assets** to the triggering release via `gh release upload "${{ github.event.release.tag_name }}" dist/*.tar.gz dist/*checksums.txt --clobber` — never a bare `dist/*`, which would also pick up GoReleaser's metadata (`artifacts.json`, `metadata.json`, `config.yaml`) and per-target build subdirectories. Upload the checksums file last (completeness signal); the step is re-runnable via `--clobber`. (Phase 3 inserts the verify gate between build and publish.)
  - **Acceptance criteria**:
    - Publishing a GitHub Release runs the workflow at the release tag; the build job produces `dist/` and the publish job attaches the four archives + checksums file to that release.
    - The release carries **only** the four archives and the checksums file — no GoReleaser metadata (`artifacts.json`, `metadata.json`, `config.yaml`) or build directories are uploaded as assets.
    - A routine push/merge or an unpublished tag does **not** trigger the workflow.
    - The publish step changes only assets — the release body and `prerelease`/`latest` status are untouched (verify against a draft release marked pre-release).
    - The publish step sets `GH_TOKEN` (= `${{ github.token }}`) so `gh release upload` authenticates; the step works with only `contents: write` + that env, no external secret.
    - Re-running for an already-published release re-uploads with `--clobber`, converging on one asset set — including recovery from a partial mid-upload failure.
  - **Dependencies**: T001
  - **Plan reference**: Phase 2: Release workflow; ADR-2
  - **Scenario references**: automated-release-pipeline.feature: "Publishing a release attaches all platform archives and a checksums file"; "Routine activity without a published release triggers no build"; "A hand-created published release is handled identically"; "Re-running for an already-published release converges on one artifact set"; "Publishing a pre-release attaches artifacts without changing its status"; "The release's pre-release and latest status is preserved"; "A build failure aborts the whole release"
  - **Interface references**: interface-spec.md: `.github/workflows/release.yml` structure; Interactions (honoring notes and status)
  - **Risk**: ⚠️ Misconfigured publish could overwrite #30's notes or flip status — test against a draft before first real publish.

## Phase 3: Cross-Target Verification Gate [US1]

- [x] **T003** [US1] Insert the cross-target self-containment verify matrix as a blocking gate before publish — verify job (4 targets → ubuntu-latest/ubuntu-24.04-arm/macos-13/macos-14) runs TestSelfContainment_HostBinary against dist; publish now needs [build, verify]; QEMU fallback documented in the workflow. 1 scenario un-@wip'd; CheckVerifyGate guard + 5 drift cases.
  - **Scope**: Add a `verify` job (`needs: build`) with a matrix over the four targets mapped to native-arch runners (linux/amd64 → `ubuntu-latest`, linux/arm64 → `ubuntu-24.04-arm`, darwin/amd64 → `macos-13`, darwin/arm64 → `macos-14`). Each leg downloads `dist/`, selects its target binary, and runs 021's `internal/build` self-containment check (`TestSelfContainment_HostBinary`, dist-artifact-preferred via `DiscoverDistBinary` reading `dist/artifacts.json`): execute → assert exit 0 → inspect dynamic-library linkage against the per-platform OS-only allowlist. Change `publish` to `needs: [build, verify]` so it runs only when build and every matrix leg pass. Document the QEMU-emulation fallback where a native runner is unavailable.
  - **Acceptance criteria**:
    - A self-containment failure on any target leg skips the publish job — nothing is attached (atomic; extends "build failure aborts" to "build-or-verification failure aborts").
    - When build and all four verify legs pass, publish runs and attaches the exact `dist/` bytes that were verified (no rebuild).
    - The emulation fallback path is documented for any unavailable native runner.
  - **Dependencies**: T002 (and 021's self-containment check)
  - **Plan reference**: Phase 3: Cross-target verification gate; ADR-3
  - **Scenario references**: automated-release-pipeline.feature: "A self-containment verification failure aborts the release"
  - **Interface references**: interface-spec.md: Job `verify` (matrix); Error Communication
  - **Risk**: ⚠️ Native arm64/macOS runner availability/cost — emulation fallback mitigates.
