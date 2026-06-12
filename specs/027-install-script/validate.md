# Validate: Install Script

**Feature**: 027-install-script
**Round**: 1 of 3
**Date**: 2026-06-12
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/runtime-dependent-distribution/install-script.feature, PROJECT.md
**Implementation files**: `install.sh` (repo root); `internal/install/install_test.go` + `internal/install/install_bdd_test.go` (test harness); `.github/workflows/ci.yml` (shellcheck step); `README.md` (install docs)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. All 3 validation scenarios satisfied through inspection (2 with noted boundary/confidence caveats that are by-design, not gaps).

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

Every driving scenario has an identifiable code path, and all 7 are additionally exercised as green godog scenarios in `internal/install` (9 scenarios / 45 steps pass, executed via `go test ./...`).

| Scenario | Status | Implementation |
|---|---|---|
| Fresh install on a supported platform | ✓ Covered | `install.sh:main` → `detect_platform`, `download`, `verify_checksum`, `install_binary`, success `printf` |
| Install a pinned version | ✓ Covered | `install.sh:resolve_tag` (pinned branch uses `GLASSFROG_VERSION` verbatim) |
| Re-running upgrades in place | ✓ Covered | `install.sh:install_binary` (`mv -f` overwrites the existing binary) |
| Checksum mismatch aborts the install | ✓ Covered | `install.sh:verify_checksum` → `main` `exit 3` before extract/install (lines 314–315) |
| Unsupported platform is refused | ✓ Covered | `install.sh:detect_platform` returns 2 → `main` `exit 2` |
| Install directory not on PATH | ✓ Covered | `install.sh:check_path` prints the exact `export PATH=…` line |
| Custom install directory | ✓ Covered | `install.sh:main` honours `GLASSFROG_INSTALL_DIR` |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — write `install.sh` | ✓ Met | All nine criteria present: `uname` mapping + exit 2; redirect-default/pinned resolution; `glassfrog_<ver>_<os>_<arch>.tar.gz` + `_checksums.txt` URL construction; `mktemp -d` + `trap … EXIT` + verify-before-place + exit 3; `mv` into `${GLASSFROG_INSTALL_DIR:-${XDG_BIN_HOME:-$HOME/.local/bin}}`, no sudo; success report + PATH export line, no profile edit; missing tool → exit 2 before download / 404 → exit 1; 0/1/2/3 scheme, diagnostics to stderr; factored functions. shellcheck-clean. |
| T002 — Go exec-test | ✓ Met | `internal/install`: godog suite (happy/mismatch→3/pinned/unsupported→2/custom/PATH/missing-tooling→2/not-found→1/upgrade) + pure-function unit tests (platform map, asset names, v-prefix). Fixtures encode the exact 022 names. Runs under `go test ./...`. |
| T003 — shellcheck in CI | ✓ Met | `ci.yml` `lint` job runs `shellcheck --shell=sh install.sh`; pre-installed on ubuntu (fork-safe under `contents: read`); golangci-lint scope untouched; bashism detection verified locally; actionlint OK. |
| T004 — documentation | ✓ Met | `README.md`: curl + wget one-liners, configured example with env on the `sh` side, three env vars + defaults, `GLASSFROG_DOWNLOAD_BASE_URL` vs `GLASSFROG_BASE_URL` distinction, supported platforms + per-user/no-sudo default. |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Element (interface-spec.md) | Status | Implementation |
|---|---|---|
| Shebang / mode (`#!/bin/sh`, `set -eu`, no `pipefail`) | ✓ Conformant | `install.sh:1`, `install.sh:48` |
| Platform detection mapping | ✓ Conformant | `detect_platform` |
| Tooling detection (curl→wget; sha256sum→shasum→openssl; tar; probe-before-download, exit 2) | ✓ Conformant | `detect_tooling` |
| Tag resolution (redirect default / pinned verbatim) | ✓ Conformant | `resolve_tag`, `resolve_redirect`, `extract_location` |
| Asset naming (`<ver>` without `v`, download path with `v`) | ✓ Conformant | `asset_names`, `normalize_tag`, `strip_v` |
| Atomic install (mktemp + trap + verify + `mv` overwrite) | ✓ Conformant | `main` (tmp + `trap … EXIT INT TERM HUP`), `install_binary` |
| Success output (`Installed glassfrog <tag> to <path>`) | ✓ Conformant | `main` success `printf` |
| PATH guidance (export line, never edit profiles) | ✓ Conformant | `check_path` |
| Configuration schema (3 env vars + defaults) | ✓ Conformant | `main` (`GLASSFROG_DOWNLOAD_BASE_URL` default `https://github.com`, `GLASSFROG_INSTALL_DIR` default `${XDG_BIN_HOME:-$HOME/.local/bin}`, `GLASSFROG_VERSION`) |
| Exit-code scheme (0/1/2/3, stderr diagnostics) | ✓ Conformant | `die_runtime`/`die_usage`/`die_integrity`, `err` |
| Function decomposition | ✓ Conformant | all named functions present |

---

## Non-Behavior Absence

**Status**: Pass (7 of 7 exclusions absent)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| No install on Windows / unsupported targets | ✓ Absent | `detect_platform` rejects with exit 2; no Windows branch |
| No build / Go toolchain / source checkout | ✓ Absent | grep: no `go build`/`make`/compiler invocation |
| No root / privilege escalation | ✓ Absent | grep: no `sudo`/`doas`/`pkexec` (only a comment documenting the absence) |
| No install on checksum mismatch | ✓ Absent | `verify_checksum` failure → `exit 3` before `tar`/`install_binary` |
| No shell-profile edits | ✓ Absent | grep: no `.bashrc`/`.zshrc`/`.profile` writes; `check_path` only prints |
| No release notes / signing / version bump | ✓ Absent | grep: no `gpg`/`cosign`/`notar`/`bump` |
| No uninstall / multi-version management | ✓ Absent | only install + in-place overwrite; no uninstall/list code |

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| verification gates the install (corrupted download never reaches the install dir) | ✓ Satisfied | Downloads land in `mktemp -d` (`$tmp`), not the install dir; `verify_checksum` runs there and `main` `exit 3`s *before* `tar`/`install_binary` (the only writer of the install dir, line 325). No binary can appear at the target on mismatch. Strongly corroborated by the executable non-@validation "Checksum mismatch" scenario (exit 3, nothing written). |
| the installed binary reports the resolved version | ✓ Satisfied (boundary note) | The script reports the resolved tag (`printf 'Installed glassfrog %s …'`) and downloads/installs the archive for that exact tag (download path uses `$tag`, asset uses `strip_v "$tag"`). The binary's own `--version` output is owned by Version Embedding (023) — spec Integration Boundaries marks this "informational/023-owned", so it is not a 027 gap. The hermetic test uses a stub binary, so the real `--version` is verifiable only against a published release. |
| latest resolution excludes pre-releases | ✓ Satisfied (lower confidence) | `resolve_tag` default branch follows `${BASE}/Luscii/cli-glassfrog/releases/latest`, which by GitHub's documented semantics redirects to the newest *stable* tag (ADR-2); the pinned branch installs a pre-release only when explicitly named. Correct by construction; the pre-release exclusion is delegated to GitHub and is not independently exercisable in the hermetic fixture (the test redirect returns whatever tag it is configured with). |

**Test execution note**: the three @validation scenarios remain `@wip` (held-out) and partly depend on a real release (binary `--version`) or external GitHub semantics (pre-release exclusion), so they are not hermetically executable with the current step set; they were verified by inspection. The corrupted-download case has a green hermetic twin among the executed driving scenarios.

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 implementable scenarios referenced by the checked tasks (T001/T002) had their `@wip` tags removed and now execute green. The 3 remaining `@wip` tags are all on `@validation` scenarios — the held-out set that validate consumes by design (per the feature file's own note and project convention). No non-`@validation` `@wip` scenario remains.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 validation scenarios are satisfied through inspection. The implementation conforms to the specification: platform detection, stable-by-default/pinnable resolution, deterministic asset naming, verify-before-place atomicity, no-sudo per-user install with PATH guidance, the installer's own exit-code scheme, and every non-behavior held. The two caveats (the binary's `--version` is 023's concern; pre-release exclusion is delegated to GitHub's `releases/latest`) are explicitly acknowledged design boundaries in the spec and ADR-2, not gaps in 027.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. (The 3 `@validation` scenarios stay `@wip` by design; a future end-to-end check against a real published release would close the binary-`--version` and pre-release-exclusion verifications that the hermetic suite cannot.)
