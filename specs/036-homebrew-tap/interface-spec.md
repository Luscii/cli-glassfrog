# Interface Accord: Homebrew Tap — Specification

**Feature**: 036-homebrew-tap
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the `brews` block in `.goreleaser.yaml` + a new `tap` job in `.github/workflows/release.yml`, publishing `Formula/glassfrog.rb` to a separate tap repo); ADR-1 (formula not cask), ADR-2 (dedicated tap repo), ADR-3 (post-publish `tap` job, brew-publisher-only, stable-only), ADR-4 (cross-repo least-privilege token)

---

## Surface

This feature adds **two declarative artifacts in this repository** — a `brews` section in `.goreleaser.yaml` and a `tap` job in `.github/workflows/release.yml` — which together publish a **third artifact** to a separate repository: `Formula/glassfrog.rb` in `Luscii/homebrew-cli-glassfrog`. No runtime command is added to the `glassfrog` CLI itself; the user-facing surface is Homebrew's own commands, parameterized by the tap and formula names chosen here.

### User invocation surface (Homebrew commands)

| Action | Command | Observable result |
|---|---|---|
| Tap + install | `brew tap luscii/cli-glassfrog && brew install glassfrog` | Downloads the archive for the host OS/arch, verifies its sha256, places `glassfrog` on PATH. |
| One-shot install | `brew install luscii/cli-glassfrog/glassfrog` | Auto-taps `Luscii/homebrew-cli-glassfrog` then installs (same result). |
| Upgrade | `brew upgrade glassfrog` | Moves to the newest stable release the formula points at; no-op if already current. |
| Version check | `glassfrog version` | Reports the release version the formula installed (`v`-prefixed, per 023). |

The tap shorthand `luscii/cli-glassfrog` resolves to `github.com/Luscii/homebrew-cli-glassfrog` (Homebrew inserts the mandatory `homebrew-` prefix). The formula name is `glassfrog`; no `--cask` qualifier (this is a formula, ADR-1). Supported platforms: macOS amd64/arm64, Linux amd64/arm64 — the four release targets.

### Producer / CI invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `tap` job in `release.yml` | `release: published` **and** `!github.event.release.prerelease` | The only automated formula publish. Runs after `publish` succeeds. |
| Local dry-run | `goreleaser release --snapshot --clean --skip=publish` | Renders `dist/.../glassfrog.rb` without tagging, uploading, or pushing — for inspecting the formula shape locally. |

### `.goreleaser.yaml` — `brews` section (added by 036)

036 **adds** a `brews` entry. It does **not** modify `builds`/`ldflags` (021/023) or `archives`/`checksum` (022), except the two reproducibility/publish refinements noted under "Refinements to 022's sections" below.

| Field | Contract |
|---|---|
| `name` | `glassfrog` (the formula/installed name). |
| `ids` | `[glassfrog]` — selects 022's archive set as the formula's source artifacts. |
| `repository.owner` | `Luscii` |
| `repository.name` | `homebrew-cli-glassfrog` (the separate tap repo, ADR-2). |
| `repository.branch` | `main` |
| `repository.token` | `{{ .Env.HOMEBREW_TAP_TOKEN }}` — the cross-repo token (ADR-4), injected only in the `tap` job. |
| `directory` | `Formula` (the formula lands at `Formula/glassfrog.rb` in the tap repo). |
| `homepage` | `https://github.com/Luscii/cli-glassfrog` |
| `description` | One line describing the CLI (a faithful command-line surface over the Glassfrog v5 API). |
| `license` | **`[NEEDS INPUT]`** — the repo has no `LICENSE` file. `brew audit --strict` requires a valid SPDX id; a private tap tolerates its absence. Resolve by adding a `LICENSE` (then GoReleaser fills this) or by deciding the formula omits/sets a proprietary marker. See Consistency Notes. |
| `install` | `bin.install "glassfrog"` — install the pre-built binary, no compile (ADR-1). |
| `test` | `system "#{bin}/glassfrog", "version"` — a no-network smoke that asserts the binary runs (the `version` leaf exists, 003). |
| `skip_upload` | `false` (explicit). Pre-release gating is enforced at the **job** level (the `if`), not via `skip_upload: auto` — see Error Communication. |

### `release.yml` — `tap` job structure (added by 036)

| Element | Contract |
|---|---|
| `needs` | `[publish]` — runs only after archives+checksums are attached and the 4-leg verify gate passed. |
| `if` | `${{ !github.event.release.prerelease }}` — **authoritative** stable-only gate. |
| `runs-on` | `ubuntu-latest` (the formula is platform-agnostic to render/push). |
| `env` | `HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}` — the only place the cross-repo token is referenced. |
| steps | `actions/checkout@v4` (`fetch-depth: 0`, ref = the release tag), `actions/setup-go@v5` (`go-version-file: go.mod`), `goreleaser/goreleaser-action@v6` (`version: "~> v2"`), then a GoReleaser invocation that runs the **brew publisher only** — it must push `Formula/glassfrog.rb` to the tap repo and must **not** create or modify the GitHub release (that boundary stays with 022's `gh release upload`). |

### Generated `Formula/glassfrog.rb` — structural contract

GoReleaser authors the formula (single build-truth, ADR-1). The committed file has this shape (illustrative — exact whitespace/order is GoReleaser's):

```ruby
class Glassfrog < Formula
  desc "<one-line description>"
  homepage "https://github.com/Luscii/cli-glassfrog"
  version "X.Y.Z"                       # the release tag without the leading v
  license "<spdx-or-resolved>"          # see [NEEDS INPUT]

  on_macos do
    on_arm   { url "https://github.com/Luscii/cli-glassfrog/releases/download/vX.Y.Z/glassfrog_X.Y.Z_darwin_arm64.tar.gz";  sha256 "<darwin_arm64>" }
    on_intel { url ".../glassfrog_X.Y.Z_darwin_amd64.tar.gz"; sha256 "<darwin_amd64>" }
  end
  on_linux do
    on_arm   { url ".../glassfrog_X.Y.Z_linux_arm64.tar.gz";  sha256 "<linux_arm64>" }
    on_intel { url ".../glassfrog_X.Y.Z_linux_amd64.tar.gz";  sha256 "<linux_amd64>" }
  end

  def install
    bin.install "glassfrog"
  end

  test do
    system "#{bin}/glassfrog", "version"
  end
end
```

**Hard contract**: each `url` points at an asset 022 attached to the release (archive names are 022's pinned `glassfrog_<ver>_<os>_<arch>.tar.gz`), and each `sha256` **equals** that asset's line in `glassfrog_<ver>_checksums.txt`. The formula must reference the exact published bytes.

### Refinements to 022's sections (coordination)

Publishing the formula from a *separate* `tap` job — after the build job already ran — requires two refinements so the formula's checksums match the published assets and the job touches only the tap:

| Refinement | Contract |
|---|---|
| `release.disable: true` | Supersedes 022's `release: { mode: keep-existing }`. With the brew publisher running in the `tap` job, GoReleaser must **not** create/modify the GitHub release (assets + status remain `gh release upload`'s / #30's domain). `disable` is the strict form of 022's defensive `keep-existing`. |
| Reproducible archives | The `tap` job rebuilds at the tag, so its archives' sha256 must be byte-identical to the build job's published archives. Pin `archives.mtime` (to the commit date) so `tar.gz` bytes are reproducible across jobs. Without this, the formula's sha256 would not match the published asset. (Alternative, if reproducibility proves brittle: derive sha256 from the published `checksums.txt` instead of a rebuild — a tasks-level fallback.) |

### Config-guard extension

021/022's `internal/build` config-guard (`CheckConfigGuard` over `LoadConfig`/`ParseConfig`, asserting the `builds` matrix and 022's `archives`/`checksum`/`release` sections) is **extended** with a focused assertion that a `brews` entry exists and targets the expected tap (`repository.owner: Luscii`, `repository.name: homebrew-cli-glassfrog`, `name: glassfrog`). Fail loudly on drift (blanked or retargeted brew block), same change-detector rigor as the siblings. Do not over-assert the formula DSL the publisher owns.

---

## Interactions

**Publish flow (release-time):** a maintainer publishes a stable GitHub Release → `build`/`verify`/`publish` run unchanged (022) and attach the verified archives + checksums → the `tap` job runs (`needs: [publish]`, `if: !prerelease`), rebuilds at the tag, and runs GoReleaser's brew publisher → `Formula/glassfrog.rb` is pushed to `Luscii/homebrew-cli-glassfrog` with `url`s pointing at the just-published assets and matching `sha256`s.

**Pre-release flow:** when the published release is flagged pre-release, the `tap` job is skipped by its `if` → the tap repo's formula is left untouched → `brew install`/`brew upgrade` continue to resolve the prior stable release.

**User install flow:** `brew tap luscii/cli-glassfrog` (or the one-shot form) → `brew install glassfrog` → Homebrew picks the `on_macos`/`on_linux` + `on_arm`/`on_intel` branch for the host, downloads that archive, verifies its `sha256` against the formula, and installs the binary. `brew upgrade glassfrog` re-evaluates the formula's `version` and replaces in place.

**Idempotency:** re-running the `tap` job for an already-published release re-renders and re-pushes the formula deterministically (reproducible archives → identical sha256 → identical formula → no-op or clean overwrite commit).

**Version source:** GoReleaser derives `version` from the release tag (022/023); the formula's `version` is the tag without the leading `v`; the download URL path keeps the `v` prefix.

---

## Error Communication

| Condition | Behavior |
|---|---|
| `brew install` and the archive's sha256 ≠ the formula's recorded `sha256` | Homebrew refuses the install and reports the checksum mismatch; no binary is placed (spec error scenario). |
| Formula references an asset not attached to the release | `brew install` fails on the download (404) rather than installing a partial/wrong binary (spec error scenario). |
| Triggering release is a pre-release | `tap` job skipped via `if`; tap repo formula unchanged; no user-visible change (spec edge case). |
| `HOMEBREW_TAP_TOKEN` missing, unauthorized, or expired | GoReleaser's push to the tap repo fails → the `tap` job exits non-zero (a loud red release run), never a silent wrong/partial publish. The release itself (assets + notes) is already published by `publish` and is unaffected. |
| Rebuilt archive sha256 ≠ published asset sha256 (reproducibility not pinned) | The formula would record a non-matching sha256 → every `brew install` fails the integrity check. Prevented by the `archives.mtime` reproducibility contract; covered by the local render-and-inspect + a checksum cross-check test. |
| `brew audit`/`brew style` failure (missing `license`/`desc`/bad `test`) | Surfaces at install/audit time on a fresh tap. Caught earlier by a CI `brew audit` step against the rendered formula (testing strategy) before a release relies on it. The `[NEEDS INPUT]` license is the most likely trigger. |
| `.goreleaser.yaml` `brews` drift (blanked/retargeted) | The extended config-guard test fails in PR Validation (#24) before a release is cut. |

---

## Consistency Notes

- **Sibling boundary (022 Automated Release Pipeline):** 036 extends the same `.goreleaser.yaml` and reuses 022's archive name template + `checksums.txt` as the formula's source of truth. It **refines** two of 022's contracts — `release:` becomes `disable: true` (strict form of `keep-existing`, since the brew publisher now runs `goreleaser release`), and `archives` gains a pinned `mtime` for cross-job reproducibility. These supersede 022's `release: { mode: keep-existing }` defensive default; flag for `/score:deprecate` if 022's keep-existing is treated as a formal decision. 022 has landed (release pipeline on main).
- **Sibling boundary (021/023):** must not touch `builds`/`ldflags`; the formula's `version` flows from the tag GoReleaser reads (023's embedding makes `glassfrog version` report the matching value — the formula `test` exercises it).
- **Sibling channel (027 Install Script):** independent acquisition route; both default to **latest stable** (027 via `releases/latest`; 036 via the pre-release `if`-gate). Parity intended; no interaction.
- **Cross-repo token (ADR-4):** the **first** credential beyond `GITHUB_TOKEN` in the release pipeline (022 was "only `GITHUB_TOKEN`"). Scoped to only the tap repo, referenced only in the `tap` job. Blast radius is the tap repo, never this repo's protected `main`.
- **User surface is Homebrew's, not the glassfrog CLI's:** the `brew ...` commands are the invocation surface; this feature adds **no** command to 001's cobra tree. That is why this is a Specification accord (config + declarative artifact), not a CLI accord.
- **License gap `[NEEDS INPUT]`:** the repo has no `LICENSE` file. Decide before the first stable release relies on the tap: add a `LICENSE` (preferred — GoReleaser then fills `brews.license` and `brew audit` passes) or accept a non-strict private tap. Tracked as the one open input from this accord.
- **Conventions:** there is no `accords/` directory; conventions are taken from PROJECT.md (Go CLI, GoReleaser per DECISIONS) and the sibling release specs. No deviation from an established accord — none exists for distribution tooling beyond the sibling specs this one aligns with.
