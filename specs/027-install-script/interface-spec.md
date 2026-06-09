# Interface Accord: Install Script — Specification

**Feature**: 027-install-script
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the single `install.sh` POSIX script); ADR-1 (POSIX `sh`), ADR-2 (redirect resolution + deterministic asset URLs), ADR-3 (atomic temp-dir install), ADR-4 (download-base-URL test seam); Script Structure & Configuration (platform mapping, config inputs, tooling detection)

---

## Surface

This feature is **one declarative artifact**: a POSIX shell script at the repo root, `install.sh` (`#!/bin/sh`, `set -eu`), invoked over the network as a one-liner. It adds no command to the CLI.

### Invocation

The canonical entry points (the exact hosted URL is fixed here as `main`-branch raw content, so the newest script always resolves the newest stable release — making the one-liner double as the upgrade path):

| Form | Command |
|---|---|
| Default (curl) | `curl -fsSL https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh \| sh` |
| Default (wget) | `wget -qO- https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh \| sh` |
| With configuration | `curl -fsSL <url> \| GLASSFROG_VERSION=v1.3.0 GLASSFROG_INSTALL_DIR="$HOME/bin" sh` |

Configuration is supplied through **environment variables only** (not flags): a piped `… | sh` cannot receive positional flags cleanly, and env vars keep the one-liner readable. The env must be set on the `sh` invocation (the right side of the pipe), as shown — not before `curl`.

### Configuration schema

| Variable | Required | Default | Description |
|---|---|---|---|
| `GLASSFROG_VERSION` | no | _(latest stable)_ | Release to install. A tag, with or without the leading `v` (`v1.3.0` or `1.3.0`). Unset → the latest **stable** release (pre-releases excluded). A pinned tag installs that exact release, **including** a pre-release. |
| `GLASSFROG_INSTALL_DIR` | no | `${XDG_BIN_HOME:-$HOME/.local/bin}` | Directory the binary is installed into. Created if absent. Must be writable by the user — the script never escalates privileges. |
| `GLASSFROG_DOWNLOAD_BASE_URL` | no | `https://github.com` | Base URL for resolution and downloads. The test/mirror seam (ADR-4). **Deliberately distinct from the CLI's `GLASSFROG_BASE_URL`** (the API endpoint, spec 008) — see Consistency Notes. |

No other input is read. Platform, asset names, and the checksum algorithm are derived or fixed, never configured.

### Structural contract (script internals the Builder must realise)

| Element | Contract |
|---|---|
| Shebang / mode | `#!/bin/sh`; `set -eu` (no `pipefail` — not POSIX). Runs identically when executed directly or piped to `sh`. |
| Platform detection | `uname -s`: `Darwin`→`darwin`, `Linux`→`linux`, else reject. `uname -m`: `x86_64`/`amd64`→`amd64`, `arm64`/`aarch64`→`arm64`, else reject. |
| Tooling detection | Downloader: `curl` preferred, else `wget`. Checksum: `sha256sum`, else `shasum -a 256`, else `openssl dgst -sha256`. Plus `tar`. Probe before downloading; if a category has none, fail (exit 2) with an actionable message naming the missing tool. |
| Tag resolution | Default: request `${GLASSFROG_DOWNLOAD_BASE_URL}/Luscii/cli-glassfrog/releases/latest` and read the resolved tag from the redirect (`Location`/effective URL `…/releases/tag/<tag>`). Pinned: use `GLASSFROG_VERSION` as the tag verbatim. |
| Asset naming | Derived from the resolved tag. `<ver>` = tag **without** leading `v`; download path uses the tag **as published** (carries `v`). Archive: `glassfrog_<ver>_<os>_<arch>.tar.gz`. Checksums: `glassfrog_<ver>_checksums.txt`. Download base: `${GLASSFROG_DOWNLOAD_BASE_URL}/Luscii/cli-glassfrog/releases/download/<tag>/<name>`. (Names owned by spec 022 — see Consistency Notes.) |
| Atomic install | All download/verify/extract happen in `mktemp -d` with `trap 'rm -rf "$tmp"' EXIT`. Verify sha256 of the archive against its line in the checksums file. Only a verified binary is `mv`'d into `GLASSFROG_INSTALL_DIR` (overwrite = upgrade). Nothing touches the install dir before verification passes. |
| Success output | To stdout: the install path and the installed version, e.g. `Installed glassfrog v1.3.0 to /home/u/.local/bin/glassfrog`. |
| PATH guidance | If `GLASSFROG_INSTALL_DIR` ∉ `$PATH`, after install print the exact line to add it (e.g. `export PATH="$HOME/.local/bin:$PATH"`). Never edit shell profiles. |

### Function decomposition (for testability — ADR-4)

`detect_platform`, `resolve_tag`, `asset_names`, `download`, `verify_checksum`, `install_binary`, `check_path`, `main`. Pure functions (`detect_platform`, `asset_names`, and checksum-line extraction) are unit-testable without network or filesystem effects.

---

## Interactions

**End-to-end flow:** operator runs the one-liner → `main` detects platform (reject → exit 2) → detects tooling (none → exit 2) → resolves the tag (default redirect, or the pinned value) → constructs archive + checksums names → downloads both into a temp dir → verifies the archive's sha256 against the checksums file (mismatch → exit 3, nothing installed) → extracts and `mv`s the binary into the install dir (overwriting any existing binary) → prints the install path + version → if the dir isn't on `$PATH`, prints the export line.

**Configuration precedence:** there is no precedence chain — each variable is independent, read once, with the default applied when unset. (Unlike the CLI's flag→env→rcfile chain (008/020); the installer has no flags and no config file.)

**Version normalisation:** a pinned `GLASSFROG_VERSION` is accepted with or without the `v` prefix. The script normalises internally: the **download path** uses the tag exactly as the release publishes it (GoReleaser tags carry `v`), while the **asset name** uses `<ver>` without `v` (GoReleaser's `{{ .Version }}`). The resolved-from-redirect tag already carries the published form.

**Idempotence / upgrade:** re-running installs the resolved version over any existing binary at the target path. Running with no `GLASSFROG_VERSION` upgrades to the newest stable; pinning an older tag downgrades. The operation converges — no duplicate state, no partial leftovers (the temp dir is always cleaned).

**Test invocation (ADR-4):** a Go exec-test sets `GLASSFROG_DOWNLOAD_BASE_URL` to an `httptest` server, `GLASSFROG_INSTALL_DIR` to a temp dir, and (optionally) `GLASSFROG_VERSION`; the server serves a fake `releases/latest` redirect, a tarball of a stub `glassfrog`, and a matching checksums file. No network, no real GitHub.

---

## Error Communication

The installer defines **its own** exit-code scheme — deliberately small and distinct from the CLI's `Outcome`/`ExitCode` convention (004), whose API/permission/rate-limit codes (3–6) have no meaning for an installer. It is not Go and does not import that mapping. Where the two overlap they agree (0 = success, 2 = usage).

| Code | Meaning | Triggering conditions |
|---|---|---|
| 0 | Success | Binary verified and installed. |
| 1 | Runtime failure | Download/network error, missing release asset (404), extraction failure, filesystem error writing the install dir. |
| 2 | Usage / environment error | Unsupported OS or architecture; no usable downloader or checksum tool; malformed `GLASSFROG_VERSION`; install dir not creatable. |
| 3 | Integrity failure | The archive's sha256 does not match its entry in the checksums file. |

| Condition | Behavior |
|---|---|
| Unsupported platform (Windows, unsupported arch) | stderr message naming the detected `os`/`arch` and the supported set (`darwin`/`linux` × `amd64`/`arch64`); nothing installed; exit 2. |
| No downloader / no sha256 tool | stderr message naming which tool category is missing and what satisfies it; exit 2 (before any download). |
| Latest cannot be resolved | No stable release exists / redirect yields no tag → stderr message; exit 1. |
| Pinned version not found | Asset download returns 404 → stderr message naming the version and asset; nothing installed; exit 1. |
| Download failure (network/transient) | stderr message; temp dir cleaned by the `EXIT` trap; nothing installed; exit 1. |
| Checksum mismatch | stderr message naming the integrity failure; **no binary written** to the install dir; temp dir cleaned; exit 3. |
| Install dir not on `$PATH` | **Not an error** — install succeeds (exit 0); stdout reports success and prints the exact `export PATH=…` line. |

All diagnostics go to **stderr**; only the success report (path + version) goes to stdout. Every non-zero exit leaves the install dir unchanged (no partial binary) — CONSTITUTION I.

---

## Consistency Notes

- **Hard dependency on spec 022's asset-name template.** The archive name `glassfrog_<ver>_<os>_<arch>.tar.gz` and checksums name `glassfrog_<ver>_checksums.txt` (sha256, `<ver>` = tag without `v`) are **owned by 022** (`022/interface-spec.md` → `.goreleaser.yaml` `archives`/`checksum`). This accord consumes them by construction (ADR-2), so a change to 022's `name_template` would 404 the installer's downloads. Mitigation pinned here: the test fixtures (ADR-4) encode these exact names, so drift breaks a test; recommend a cross-reference comment in both `install.sh` and `.goreleaser.yaml`.
- **Distinct env namespace from the CLI runtime.** `GLASSFROG_DOWNLOAD_BASE_URL` is intentionally **not** `GLASSFROG_BASE_URL`. The CLI already defines `GLASSFROG_BASE_URL` as the **API** endpoint override (spec 008, `internal/apiclient/baseurl.go` → `EnvVarBaseURL`). A user with `GLASSFROG_BASE_URL` exported for the API must not have the installer silently fetch release archives from the API host. Likewise the installer does not read `GLASSFROG_TOKEN` (008/auth) or `GLASSFROG_OUTPUT` (020) — they are runtime-CLI concerns. The installer's variables (`GLASSFROG_VERSION`, `GLASSFROG_INSTALL_DIR`, `GLASSFROG_DOWNLOAD_BASE_URL`) keep the shared `GLASSFROG_` prefix for discoverability while staying semantically separate.
- **Exit-code scheme is the installer's own, not 004's.** 004 (`internal/cli/exitcode.go`) governs the Go CLI's category→code mapping (0/1/2 plus API-specific 3–6). The installer reuses the *shape* (0 success, 2 usage) but defines 1=runtime and 3=integrity for its own failure modes; it deliberately does not adopt 004's 3–6 (API/permission/rate/network), which don't apply. Documented so a reviewer doesn't expect the CLI mapping.
- **Shell dialect diverges from the repo's operational scripts, deliberately.** `scripts/*.sh` use `#!/usr/bin/env bash` + `set -euo pipefail` for maintainer machines; `install.sh` targets `#!/bin/sh` + `set -eu` for an arbitrary host piped via `curl | sh` (ADR-1). `shellcheck` is run with the `sh` dialect.
- **Sibling acquisition channels.** Homebrew Tap (036) and NPM Wrapper (037) install the same released binary by other routes and share the same upstream artifacts; they are independent of this script. There is no `accords/` directory in this project — conventions are taken from PROJECT.md and the sibling spec 022, which set the precedent for declarative-artifact specification accords. This accord follows that precedent (single `interface-spec.md`).
- **No runtime CLI surface.** This feature touches no Go command, so there is no `interface-cli.md` — the only consumer-facing surface is the script's invocation and its stdout/stderr/exit-code contract, all captured here.
