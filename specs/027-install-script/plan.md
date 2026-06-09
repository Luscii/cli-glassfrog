# Plan: Install Script

**Feature**: 027-install-script
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, .score/memory/DECISIONS.md (021 GoReleaser build, 022 archive/checksum naming, 006 atomic-rename storage, 008 inject-seam precedent), 022's interface-spec.md (the consumed archive/checksum contract)

---

## System Architecture

This feature ships **one declarative artifact**: a single POSIX shell script (recommended path `install.sh` at the repo root, for a clean `curl … | sh` URL — exact path/URL is interface-level). It adds no Go code and no command to the CLI. It is a host-side installer that consumes the artifacts the Automated Release Pipeline (022) attaches to a GitHub Release and places the `glassfrog` binary on the user's PATH.

The script is a linear pipeline of small functions, each mapping to a spec accord:

```
detect            uname -s / uname -m  →  {darwin,linux} × {amd64,arm64}        ⇒ Platform detection
   │                  unsupported (Windows, other arch) ⇒ clear error, exit ≠0
   ▼
resolve           default: GET https://github.com/Luscii/cli-glassfrog/releases/latest
   │                       follow the redirect, read the tag from the Location header
   │              pinned:  caller-supplied version → use that tag verbatim          ⇒ Release resolution
   │                  no release / bad version ⇒ clear error, exit ≠0
   ▼
download          to a private temp dir (mktemp -d, trap-cleaned):                  ⇒ Download and verify
   │                  archive   = glassfrog_<ver>_<os>_<arch>.tar.gz
   │                  checksums = glassfrog_<ver>_checksums.txt
   │                  from https://github.com/.../releases/download/<tag>/<name>
   │                  download failure ⇒ clear error, exit ≠0
   ▼
verify            sha256(archive) vs its line in checksums.txt                       ⇒ Download and verify
   │                  mismatch ⇒ error, nothing installed, exit ≠0
   ▼
install           extract binary in temp dir → mv into <install-dir>                 ⇒ Install and placement
   │                  (mv over any existing binary = upgrade/overwrite)
   │              PATH check: if <install-dir> ∉ $PATH, print the exact export line
   │              report install location + installed version
```

`<ver>` is the release tag **without** the leading `v` (GoReleaser's `{{ .Version }}` convention — 022's interface); the script strips a leading `v` from the resolved/pinned tag when building asset names, and uses the `v`-prefixed tag in the download path. `<os>` ∈ {`darwin`,`linux`}, `<arch>` ∈ {`amd64`,`arm64`}. Because the archive and checksums names are a **deterministic template** (pinned by 022), the script needs nothing from GitHub except the latest tag — it never has to list or parse a release's asset JSON.

The binary is `glassfrog` (`project_name`/`builds.binary` in `.goreleaser.yaml`); the repo is `Luscii/cli-glassfrog`. The installer is a shell script and legitimately uses host tooling (a downloader, `tar`, a sha256 utility) — this is **not** a CONSTITUTION XII concern, which governs the *distributed binary's* runtime, not the installer that places it (the same build-host-vs-artifact distinction 021/022 carried).

---

## Architecture Decisions

### ADR-1: Write the installer in POSIX `sh`, not `bash`

**Context**: The FEATURE-MODEL calls this "a POSIX one-liner," and the canonical invocation is `curl -fsSL <url> | sh`. The script runs on whatever shell the host's `sh` resolves to (dash, busybox ash, bash-as-sh) across macOS and Linux laptops and CI images. The repo's existing operational scripts (`scripts/setup-branch-protection.sh`) use `#!/usr/bin/env bash` with `set -euo pipefail`, but those are maintainer-run, not piped into an arbitrary host's `sh`.

**Options considered**:
1. **POSIX `sh`** — portable to any host without assuming bash is installed or is `/bin/sh`. Cost: no arrays, no `[[ ]]`, no `local` guarantees in the strict sense, no `pipefail` — more disciplined shell.
2. **`bash`** — richer features and the repo's existing convention, but bash is not guaranteed on minimal Linux/CI images, and shebang-vs-`| sh` mismatch is a classic install-script footgun.

**Decision**: Option 1 — POSIX `sh`. The script targets `#!/bin/sh`, uses only POSIX-portable constructs, and sets `set -eu` (no `pipefail`, which is not POSIX). It must run identically whether executed directly or piped to `sh`.

**Consequences**: Maximum reach with no shell dependency — aligns with the self-contained-distribution goal. The script is held to POSIX shell discipline, enforced by `shellcheck` with a POSIX/`sh` dialect target (Cross-cutting Concerns). This diverges from the repo's bash convention for operational scripts, deliberately, because the audience differs (arbitrary host vs. maintainer's machine).

### ADR-2: Resolve "latest stable" via the `releases/latest` redirect; construct asset URLs from the deterministic name template — no GitHub API, no `jq`

**Context**: The spec resolves the latest **stable** release by default (pre-releases only when a version is pinned) and downloads the matching archive + checksums. Two mechanisms exist to learn which release and where its assets live. The script must run on a bare host (no `jq`, possibly no API token) without tripping GitHub's unauthenticated API rate limit (60/hr).

**Options considered**:
1. **GitHub REST API** — `GET /repos/Luscii/cli-glassfrog/releases/latest` (and `/releases/tags/<tag>` for a pin) returns JSON with asset download URLs. Authoritative, but requires parsing JSON in POSIX `sh` without `jq` (fragile hand-rolled parsing or an added dependency) and is rate-limited unauthenticated.
2. **Redirect + deterministic URLs** — `https://github.com/Luscii/cli-glassfrog/releases/latest` 302-redirects to `…/releases/tag/<tag>`; read `<tag>` from the `Location` header (`curl -sI` / `curl -fsSL -o /dev/null -w '%{url_effective}'`). GitHub's `releases/latest` **excludes pre-releases by definition** — which is exactly the spec's "latest stable." Then build asset URLs from 022's pinned template: `…/releases/download/<tag>/glassfrog_<ver>_<os>_<arch>.tar.gz`. A pinned version skips the redirect and uses the tag directly. No JSON, no token, no `jq`.

**Decision**: Option 2. Default path: follow the `releases/latest` redirect to learn the tag, then construct the archive and checksums URLs deterministically. Pinned path: take the caller's version as the tag (normalise to ensure the download path's `v` prefix and the asset name's non-`v` `<ver>`), skip resolution, download directly — so a pinned **pre-release** works because the user named its exact tag, while the default never selects a pre-release.

**Consequences**: The installer depends on **two contracts owned elsewhere**: GitHub's `releases/latest`-excludes-prereleases semantics, and 022's archive/checksum **name template** (`glassfrog_<ver>_<os>_<arch>.tar.gz`, `glassfrog_<ver>_checksums.txt`). If 022 ever changes the template, the installer breaks — a coupling to pin in the interface and guard against (Risks). No `jq`/JSON-parse fragility and no API-rate-limit exposure. A pinned version that doesn't exist surfaces as a download 404 → clear error.

### ADR-3: Install atomically through a trap-cleaned temp directory; verify before anything reaches the install dir

**Context**: The spec is absolute that a checksum mismatch, a download failure, or any abort leaves **no** binary on PATH and **no** partial artifact — and that a re-run overwrites in place. CONSTITUTION I (no partial writes / no failure-as-success) governs. The repo already set this discipline for secrets: 006 writes credentials via temp-file-then-`rename` so a mid-write failure leaves the original intact (DECISIONS).

**Options considered**:
1. **Temp dir + verify + `mv` into place, with a cleanup trap** — download archive and checksums into `mktemp -d`, verify the checksum there, extract there, then `mv` the binary onto the final path (atomic within a filesystem; overwrites an existing binary = the upgrade path). `trap 'rm -rf "$tmp"' EXIT` guarantees cleanup on success, failure, or interrupt. Nothing touches the install dir until verification passes.
2. **Download straight to the install dir, verify in place** — fewer moves, but a failed/mismatched download leaves a corrupt or half-written binary on PATH, violating the spec and CONSTITUTION I.

**Decision**: Option 1. All fetching, verification, and extraction happen in a private temp dir; only a verified binary is `mv`'d to the install dir. A single `EXIT` trap removes the temp dir on every exit path. The final `mv` is the only mutation of the user's environment and happens last.

**Consequences**: Strong atomicity — the only observable outcomes are "the verified new binary is installed" or "nothing changed." Re-runs overwrite cleanly (upgrade/downgrade). If the install dir and `TMPDIR` are on different filesystems, `mv` degrades to copy-then-unlink, still effectively atomic for the consumer (a brief window, acceptable for a single binary). Cleanup is interrupt-safe.

### ADR-4: Expose a download-base-URL override as the test seam; cover the script with `shellcheck` + a Go exec-test against a local HTTP server

**Context**: The project has a deep testing culture (golden tests, config-guards, BDD), but a network-fetching shell installer is normally untested — it hits real GitHub. PR Validation (024) lints Go only (`golangci-lint`) and runs `go test ./...`; it does not lint shell. The installer's correctness (correct platform mapping, checksum gating, atomic install, unsupported-platform rejection) is exactly the kind of thing that regresses silently. 008 established an inject-seam pattern (a resolvable base with an OS production seam) precisely so network behavior is testable hermetically.

**Options considered**:
1. **Base-URL override env var + Go exec-test + shellcheck** — the script reads an optional base-URL override (defaulting to `https://github.com`); a Go test (`internal/install` or a `*_test.go` near the script) starts an `httptest.Server` that serves a fake `releases/latest` redirect, a fake archive, and a checksums file, then execs the script with the override and a temp install dir, asserting: happy-path install, checksum-mismatch abort (no binary written), and pinned-version selection. `shellcheck` (sh dialect) runs in CI for static correctness. The unsupported-platform path is covered by factoring detection so it's unit-testable (or a focused test that shims `uname`).
2. **No automated test (manual/CI-smoke only)** — cheapest, but leaves the installer as the one untested distribution artifact; checksum-gating and atomicity regressions would ship unnoticed.

**Decision**: Option 1. The script gains exactly one testability seam — an overridable download base URL (same shape as 008's base-URL resolution) — plus the already-required version and install-dir inputs. A Go exec-test drives the script end-to-end against `httptest` with hermetic fixtures; `shellcheck` provides static analysis. Whether `shellcheck` is wired into 024's lint job or run as a dedicated step is an interface/tasks call (it touches 024's lint surface — noted as an integration point).

**Consequences**: The installer earns the same regression protection as the rest of the pipeline; checksum-gating, platform mapping, and atomic-install behavior are pinned by tests that never touch the network. Cost: a small Go test harness and fixtures, and a `shellcheck` dependency on CI hosts (a CI-host tool, not an artifact dependency — same standing as GoReleaser/`golangci-lint`). The base-URL seam is a documented power-user/enterprise-mirror affordance as a side benefit.

---

## Script Structure & Configuration

The script is organised as discrete functions (`detect_platform`, `resolve_tag`, `download`, `verify_checksum`, `install_binary`, `check_path`, `main`) so individual steps are reasoned about and, where pure, unit-testable.

**Platform mapping** (the only nontrivial normalisation):

| `uname -s` | → os | | `uname -m` | → arch |
|---|---|---|---|---|
| `Darwin` | `darwin` | | `x86_64`, `amd64` | `amd64` |
| `Linux` | `linux` | | `arm64`, `aarch64` | `arm64` |
| anything else | **reject** | | anything else | **reject** |

**Configuration inputs** (behavior fixed by the spec; the *mechanism* and exact names are interface-level — likely environment variables to keep the `curl | sh` one-liner clean, e.g. `VERSION`, an install-dir override, and a download base-URL override):
- **Version** — unset → latest stable; set → that exact tag (may be a pre-release).
- **Install directory** — unset → default per-user dir, recommended `${XDG_BIN_HOME:-$HOME/.local/bin}` (`[ASSUMED]`, interface-pinned); set → install there.
- **Download base URL** — unset → `https://github.com`; set → the test/mirror seam (ADR-4).

**Tooling detection**: a downloader (`curl` preferred, `wget` fallback) and a sha256 utility (`sha256sum` on Linux/coreutils, `shasum -a 256` on macOS, `openssl dgst -sha256` as a last resort) and `tar`. The script probes for an available tool in each category and fails with a clear, actionable message if none is present, before any download. No tool is assumed beyond a POSIX base + one downloader + one sha256 utility.

---

## Cross-cutting Concerns

- **Error handling / atomicity** — `set -eu` plus explicit checks at each step; every failure (unsupported platform, unresolved/absent release, download failure, checksum mismatch, missing tooling) prints a clear stderr message naming the cause and exits non-zero. Per ADR-3, nothing reaches the install dir until the checksum verifies, and an `EXIT` trap removes the temp dir on every path — honoring CONSTITUTION I (no partial install, no failure-reported-as-success).
- **Privilege** — never invokes `sudo` or escalation; a per-user default dir keeps it unattended (spec). A system-wide install is reachable only by the caller pointing the install dir at a writable system location.
- **PATH guidance** — when the install dir is not on `$PATH`, the script still installs and prints the exact `export PATH=…` line; it never edits `.bashrc`/`.zshrc`/`.profile` (spec non-behavior).
- **Configuration** — three inputs only (version, install dir, base URL); everything else (platform, asset names, checksum algorithm) is derived or fixed.
- **Testing strategy** — `shellcheck` (sh dialect) for static analysis; a Go exec-test against an `httptest` server for happy-path install, checksum-mismatch abort, and pinned-version selection; pure functions (platform mapping, name construction, checksum-line extraction) unit-tested directly (ADR-4). No test touches the network.
- **Idempotence / upgrade** — re-running overwrites in place (ADR-3); the script is safe to run repeatedly and doubles as the upgrade/downgrade path.

---

## Implementation Strategy

This is a single artifact with a natural internal order; it can ship as one PR-sized unit, but decomposes cleanly into three concerns the tasks skill can sequence:

**Phase 1 — Core installer.** Write `install.sh` (`#!/bin/sh`, `set -eu`): platform detect + mapping, tooling detection, tag resolution (redirect default + pinned path), deterministic asset-name construction, temp-dir download, sha256 verification, atomic `mv` install, PATH check + guidance, success/error reporting. Honor the three config inputs.

**Phase 2 — Test harness.** Add the Go exec-test driving `install.sh` against an `httptest` server with fixtures (fake `releases/latest` redirect, a real tar.gz of a stub `glassfrog`, a matching checksums file): assert happy-path install into a temp dir, checksum-mismatch abort (no binary written), pinned-version selection, and the unsupported-platform rejection (via a detection seam or `uname` shim). Wire `shellcheck` into CI (a dedicated step, or extend 024's lint job — interface decides).

**Phase 3 — Documentation surface.** The canonical one-liner and the config env-var names in the README/usage. (Doc/interface concern — listed so it isn't lost; not application logic.)

Phases are ordered 1 → 2 → 3; all are unblocked because 022 has landed (the archive/checksum naming the script consumes exists).

**Dependency gate**: the Automated Release Pipeline (022) — which produces the consumed archives + checksums — is `Implemented` on main (STATUS), so this feature is buildable now. The script can be tested hermetically (ADR-4) without waiting for a real published release.

---

## Risks

- **Coupling to 022's asset-name template** — the script constructs `glassfrog_<ver>_<os>_<arch>.tar.gz` / `glassfrog_<ver>_checksums.txt` by convention (ADR-2). If 022 changes the GoReleaser `name_template`, downloads 404 silently against the new layout. *Medium.* Mitigation: pin the shared template as a named contract in the interface (cite 022's interface-spec); consider a cross-reference comment in both `.goreleaser.yaml` and `install.sh`; the exec-test fixtures encode the expected names so a drift breaks a test.
- **`releases/latest` redirect / version-prefix handling** — the redirect target format or the `v`-prefix normalisation (download path needs `v`, asset name needs non-`v`) could be mishandled, picking the wrong tag or building a wrong URL. *Medium.* Mitigation: unit-test the tag-parse and name-construction pure functions; exec-test the resolution path against the fake redirect.
- **POSIX-portability regressions** — a bashism (arrays, `[[ ]]`, `local`, `pipefail`) creeps in and breaks on dash/busybox. *Medium.* Mitigation: `shellcheck` with the `sh` dialect in CI; `set -eu` (not `pipefail`).
- **sha256 / downloader tool absence or variance** — a minimal host lacks `curl`/`wget` or a sha256 utility, or `shasum` vs `sha256sum` output formats differ. *Low–medium.* Mitigation: probe-and-fallback across the known tools, normalise the hash comparison, fail clearly before download if none is available.
- **Non-atomic `mv` across filesystems** — if `TMPDIR` and the install dir are on different mounts, `mv` is copy+unlink, not a rename. *Low.* Mitigation: acceptable for a single binary; optionally `mktemp` inside the install dir's parent when writable to keep the rename on one filesystem (interface/tasks detail).

---

## What This Plan Does Not Cover

- **Protocol/structural contracts** — the exact config env-var names, the canonical hosted URL and one-liner form, the precise default install directory, the stderr message wording, and the exit-code mapping are the interface skill's concern (`/score:interface`). The exit-code convention itself is 004's (`internal/cli/exitcode.go`) — the script reports its own non-zero codes, but is not bound to the CLI's internal `Outcome` enum (it is not Go).
- **The archive/checksum production** — owned by 022 (GoReleaser `archives`/`checksum`); 027 consumes the pinned names, never re-decides them.
- **Version embedding / `--version` output** — owned by 023; the script reports the version it installed and may run `glassfrog --version` to confirm (validation scenario), but does not own the embedding.
- **Sibling acquisition channels** — Homebrew Tap (036) and NPM Wrapper (037) install the same binary by other routes; out of scope here.
- **Signing / notarization** — explicitly out of scope (spec); integrity is the checksums file.
- **Uninstall / multi-version management** — non-behaviors in the spec; not designed here.
