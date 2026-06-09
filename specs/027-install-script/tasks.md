# Tasks: Install Script

**Feature**: 027-install-script
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/install-script.feature

---

## Dependency Graph

Phase 1: Core installer (1 task, no dependencies) [Shared]
Phase 2: Test harness (2 tasks, depends on Phase 1) [Shared]
Phase 3: Documentation surface (1 task, depends on Phase 1 — parallel with Phase 2) [Shared]

4 tasks total | Phase 2 and Phase 3 parallelizable | Builder: single Builder (pipeline mode)

---

## Branching Guidance

**Pipeline mode**: `spec/027-install-script/base` → `spec/027-install-script/task-1`, `spec/027-install-script/task-2`, …

**Role-based mode**: `spec/027-install-script/base` as the integration point; task branches `spec/027-install-script/task-N`. Low parallel pressure here — Phase 1 is a single artifact every later task depends on, so the practical order is T001, then T002/T003/T004.

---

## Phase 1: Core installer [Shared]

- [ ] **T001** [Shared] Write `install.sh` — the complete POSIX install script
  - **Scope**: A single new file `install.sh` at the repo root (`#!/bin/sh`, `set -eu`). One reviewable PR. Implements the full pipeline: platform detection + mapping, tooling detection, tag resolution (latest-via-redirect default + pinned), deterministic asset-name construction, temp-dir download, sha256 verification, atomic `mv` install, PATH check + guidance, success reporting, and the exit-code scheme. Reads only `GLASSFROG_VERSION`, `GLASSFROG_INSTALL_DIR`, `GLASSFROG_DOWNLOAD_BASE_URL`.
  - **Acceptance criteria**:
    - `uname -s`/`uname -m` map to `{darwin,linux}`×`{amd64,arm64}`; any other platform exits 2 with a message naming the detected platform and supported set, installing nothing.
    - Default resolution follows the `${GLASSFROG_DOWNLOAD_BASE_URL}/Luscii/cli-glassfrog/releases/latest` redirect to a tag; a set `GLASSFROG_VERSION` (with or without leading `v`) is used verbatim.
    - Asset URLs are constructed as `glassfrog_<ver>_<os>_<arch>.tar.gz` and `glassfrog_<ver>_checksums.txt` (`<ver>` = tag without `v`; download path keeps the published `v`).
    - All download/verify/extract happen in `mktemp -d` with a `trap … EXIT` cleanup; the archive's sha256 is checked against its checksums line before anything is placed; a mismatch exits 3 with nothing written to the install dir.
    - A verified binary is `mv`'d into `${GLASSFROG_INSTALL_DIR:-${XDG_BIN_HOME:-$HOME/.local/bin}}` (created if absent), overwriting any existing binary; no `sudo`/escalation is ever invoked.
    - On success, stdout reports the install path and installed version; when the install dir is not on `$PATH`, the exact `export PATH=…` line is printed and no shell profile is edited.
    - Missing downloader or sha256 utility exits 2 before any download, naming the missing category; a missing/404 release asset exits 1.
    - Exit codes follow the interface scheme: 0 success, 1 runtime, 2 usage/environment, 3 integrity; all diagnostics go to stderr.
    - Functions are factored (`detect_platform`, `resolve_tag`, `asset_names`, `download`, `verify_checksum`, `install_binary`, `check_path`, `main`) so pure steps are unit-testable.
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Core installer; ADR-1 (POSIX `sh`), ADR-2 (redirect resolution + deterministic URLs), ADR-3 (atomic temp-dir install)
  - **Scenario references**: install-script.feature: "Fresh install on a supported platform", "Unsupported platform is refused", "Install directory not on PATH still installs with guidance", "Install a pinned version", "Install into a custom directory", "Checksum mismatch aborts the install", "Re-running upgrades in place", "A host missing required tooling fails before any download", "A pinned version that does not exist fails clearly"
  - **Interface references**: interface-spec.md: Surface (Invocation, Configuration schema, Structural contract); Error Communication (exit-code scheme)
  - **Risk**: ⚠️ Hard coupling to 022's archive/checksum name template — a template change 404s downloads (pin via the test fixtures in T002)

## Phase 2: Test harness [Shared]

- [ ] **T002** [Shared] Go exec-test driving `install.sh` against an `httptest` server
  - **Scope**: A Go test (e.g. `internal/install/install_test.go`, or a test beside the script) that starts an `httptest.Server` serving a fake `releases/latest` redirect, a real `tar.gz` of a stub `glassfrog`, and a matching checksums file; execs `install.sh` with `GLASSFROG_DOWNLOAD_BASE_URL`, `GLASSFROG_INSTALL_DIR`, and (per case) `GLASSFROG_VERSION`. Encodes the exact 022 asset names as fixtures so template drift breaks the test. Hermetic — no network.
  - **Acceptance criteria**:
    - Happy path: the stub binary lands in the temp install dir and the script exits 0.
    - Checksum mismatch: no binary is written to the install dir and the script exits 3.
    - Pinned version: the requested tag's asset is fetched (not the latest) and installed.
    - Unsupported platform: the rejection path exits 2 (via a detection seam or a `uname` shim) with nothing installed.
    - Pure-function unit tests cover platform mapping and asset-name construction directly.
    - The test runs under the existing `go test ./...` invocation (so PR Validation 024 / Main-Branch Verification 029 pick it up).
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Test harness; ADR-4 (download-base-URL test seam)
  - **Scenario references**: install-script.feature: "A corrupted download never reaches the install directory" (@validation), "Latest resolution installs the newest stable, not a pre-release" (@validation), "The installed binary reports the resolved version" (@validation), "Checksum mismatch aborts the install", "Install a pinned version", "Unsupported platform is refused"
  - **Interface references**: interface-spec.md: Interactions (Test invocation); Surface (Function decomposition)

- [ ] **T003** [Shared] [P] Add `shellcheck` (sh dialect) static analysis to CI
  - **Scope**: Wire `shellcheck` over `install.sh` into CI — either as a step in PR Validation (024)'s lint job or a dedicated lint step — targeting the POSIX `sh` dialect. CI-host tool only, not an artifact dependency.
  - **Acceptance criteria**:
    - `shellcheck` runs against `install.sh` in CI with the `sh` dialect and fails the build on a finding.
    - The chosen wiring (extend 024's lint job vs. dedicated step) is consistent with the existing lint surface and does not change `golangci-lint`'s scope.
    - A deliberate bashism in `install.sh` is reported (verified once locally).
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Test harness; ADR-1 (POSIX `sh`, `shellcheck` gate)
  - **Interface references**: interface-spec.md: Consistency Notes (shell-dialect divergence from operational scripts)
  - **Risk**: ⚠️ Touches 024's lint surface — confirm the `shellcheck` install/step is fork-safe and matches 024's `permissions: contents: read`

## Phase 3: Documentation surface [Shared]

- [ ] **T004** [Shared] [P] Document the canonical one-liner and configuration
  - **Scope**: Add install instructions to the README (and usage docs as appropriate): the canonical `curl … | sh` and `wget … | sh` one-liners, the `main`-branch raw URL, and the three env vars (`GLASSFROG_VERSION`, `GLASSFROG_INSTALL_DIR`, `GLASSFROG_DOWNLOAD_BASE_URL`) with defaults and the env-on-the-`sh`-side caveat.
  - **Acceptance criteria**:
    - The README shows the default curl one-liner, the wget alternative, and a configured example with env vars set on the `sh` invocation.
    - The three env vars are documented with their defaults; `GLASSFROG_DOWNLOAD_BASE_URL` is noted as distinct from the CLI's `GLASSFROG_BASE_URL`.
    - The supported platforms (macOS/Linux × amd64/arm64) and the per-user/no-sudo default are stated.
  - **Dependencies**: T001
  - **Plan reference**: Phase 3 — Documentation surface
  - **Interface references**: interface-spec.md: Surface (Invocation, Configuration schema)
