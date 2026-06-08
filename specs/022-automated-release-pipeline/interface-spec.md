# Interface Accord: Automated Release Pipeline — Specification

**Feature**: 022-automated-release-pipeline
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the `.goreleaser` release sections + `.github/workflows/release.yml`); ADR-1 (extend `.goreleaser`), ADR-2 (`release: published` trigger, attach to existing release), ADR-3 (cross-target verify gate)

---

## Surface

This feature is two declarative artifacts at the repo root, consumed by GitHub Actions and GoReleaser. There is no runtime command added to the CLI.

### Invocation

| Entry point | Trigger | Notes |
|---|---|---|
| `.github/workflows/release.yml` | GitHub event `release` with `types: [published]` | The single automated trigger. `github.event.release` carries the tag, the `prerelease` flag, and the existing body. |
| Local dry-run | `goreleaser release --snapshot --clean --skip=publish` | Produces `dist/` without a tag or upload — for verifying the archive/checksum layout locally. |
| CI build invocation | `goreleaser release --clean --skip=publish` | Run at the release tag (checkout ref = the tag); builds + archives + checksums, no upload. |

### `.goreleaser` sections added by 022

022 **adds** these top-level sections to the file 021 created. It does **not** modify `builds` or `builds.ldflags` (021 owns the matrix; 023 owns the ldflags version seam).

| Section | Contract |
|---|---|
| `archives` | One archive per build target, format `tar.gz`. Name template (GoReleaser default) `glassfrog_{{.Version}}_{{.Os}}_{{.Arch}}` → e.g. `glassfrog_1.4.0_darwin_arm64.tar.gz`. Each archive contains the `glassfrog` binary (and may include `LICENSE`/`README`). `{{.Version}}` is the tag without the leading `v`; `{{.Os}}` ∈ {`darwin`,`linux`}; `{{.Arch}}` ∈ {`amd64`,`arm64`}. |
| `checksum` | A single checksums file, algorithm `sha256`, default name `glassfrog_{{.Version}}_checksums.txt`, with one line per published archive. |
| `release` | Defensive default for any direct `goreleaser release`: `mode: keep-existing` (never replace an existing release's body), `draft: false`, no `prerelease`/`make_latest` override. In the canonical workflow this section is not the publisher (see Interactions — publish uses `gh release upload`). |

### `.github/workflows/release.yml` structure

| Element | Contract |
|---|---|
| `on` | `release: { types: [published] }` |
| `permissions` | `contents: write` (the only privilege; no external secrets) |
| Job `build` | `ubuntu-latest`: `actions/checkout@v4` (`fetch-depth: 0`), `actions/setup-go@v5` (`go-version-file: go.mod`), install GoReleaser via `goreleaser/goreleaser-action@v6` (`version: "~> v2"`), run `goreleaser release --clean --skip=publish`, then `actions/upload-artifact` of `dist/` (the four `*.tar.gz` + the checksums file). |
| Job `verify` (matrix) | `needs: build`. Matrix of the four targets mapped to native-arch runners: linux/amd64 → `ubuntu-latest`, linux/arm64 → `ubuntu-24.04-arm`, darwin/amd64 → `macos-13`, darwin/arm64 → `macos-14`. Each leg downloads `dist/`, selects its target archive, and runs 021's self-containment check (execute the binary → assert exit 0 → inspect dynamic-library linkage against the per-platform OS-only allowlist). |
| Job `publish` | `needs: [build, verify]` (runs only if build and **every** verify leg succeed). Downloads `dist/` and uploads the archives + checksums file to the triggering release via `gh release upload "${{ github.event.release.tag_name }}" dist/* --clobber`. |

### Config-guard extension (021's drift test)

021's config-guard test (asserts the `builds` matrix is exactly the four targets and `CGO_ENABLED=0`) is **extended** to also assert the `archives` (tar.gz, four targets), `checksum` (sha256, single file), and `release` (keep-existing) sections are present and unchanged — fail loudly on drift, same change-detector rigor (a missing section fails as loudly as an extra one).

---

## Interactions

**Release flow (end-to-end):** A maintainer publishes a GitHub Release — drafted by Release Drafting (#30) or created by hand → the `release: published` event starts the workflow at the release tag → `build` produces `dist/` (four `tar.gz` + checksums, version derived from the tag) → `verify` fans out across the four native-arch runners and runs the self-containment check on each target binary → `publish` runs only if all pass and uploads the verified `dist/` to the same release.

**Honoring notes and status:** the publish step uses `gh release upload … --clobber`, which adds/overwrites *assets only* and never touches the release body or the `prerelease`/`latest` flags. The status and notes the publisher (or #30) set are therefore preserved unchanged — 022 adds artifacts, not prose, and does not decide status.

**Verified-bytes = published-bytes:** the artifacts uploaded by `publish` are the exact `dist/` produced by `build` and checked by `verify` (carried as a CI artifact), not a rebuild.

**Re-publish convergence:** re-running for an already-published release re-uploads with `--clobber`, converging on one asset set (no duplicates).

**Version source:** GoReleaser derives `{{.Version}}` from the git tag the published release points at; 022 injects no version logic (023 owns embedding via the untouched `ldflags`).

---

## Error Communication

| Condition | Behavior |
|---|---|
| Any target fails to build/archive in `build` | `goreleaser release` exits non-zero → `build` fails → `verify` and `publish` never run → nothing uploaded (atomic, no partial release). |
| Any `verify` matrix leg fails the self-containment check | that leg exits non-zero → `publish` (which `needs` the whole matrix) is skipped → nothing uploaded. Extends spec atomicity to "build-or-verification failure aborts." |
| Trigger fires but no release object exists | not reachable for `release: published` (the event implies a published release). A raw tag push does **not** trigger the workflow. |
| `.goreleaser` drift (lost target, `CGO_ENABLED=1`, missing `archives`/`checksum`/`release`) | the extended config-guard test fails in PR Validation (#24) before a release is ever cut. |
| Re-run for an existing release | converges via `--clobber`; assets replaced, release body/status untouched. |

---

## Consistency Notes

- **Sibling boundary (021 Self-Contained Executable Build):** this accord extends the `.goreleaser` file 021 defines and reuses 021's self-containment check verbatim (pointed at `dist/`). It must not redefine `builds`/`ldflags`. 022's implementation is gated on 021 landing.
- **Sibling boundary (023 Version Embedding):** `builds.ldflags` is left untouched as 023's seam; archive/version naming flows from the tag GoReleaser reads.
- **Sibling boundary (#30 Release Drafting):** #30 owns the release body, draft, and pre-release/latest status; this accord's `gh release upload --clobber` publish step is specifically chosen to preserve all three.
- **Downstream consumers (#27 Install Script, #36 Homebrew Tap, #37 NPM Wrapper):** depend on the archive name template (`glassfrog_<version>_<os>_<arch>.tar.gz`) and the sha256 checksums file pinned here. Homebrew later adds a `brew` section to the same `.goreleaser`.
- **Assumption — publish via `gh release upload` rather than `goreleaser release`:** GoReleaser OSS cannot resume-publish a pre-built, externally-verified `dist/`, and the spec requires verification *before* publish. `gh release upload --clobber` publishes the exact verified bytes and inherently preserves notes/status. The reference pipeline mixes `goreleaser release` (build) and `gh release upload` (extra assets) the same way. The `release` section is retained with `mode: keep-existing` as a defensive default for direct GoReleaser runs.
- **Assumption — runner labels** (`ubuntu-24.04-arm`, `macos-13`, `macos-14`): the chosen GitHub-hosted native-arch runners. Where a native runner is unavailable, QEMU emulation on a linux runner is the documented fallback (plan Risk).
- **Conventions:** there is no `accords/` directory in this project; conventions are taken from PROJECT.md (Go CLI, GoReleaser per DECISIONS) and the maintainer's reference pipelines. No deviation from an established accord because none exists for release tooling yet — this accord sets the pattern.
