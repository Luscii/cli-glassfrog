# Interface Accord: Self-Contained Executable Build — Specification

**Feature**: 021-self-contained-executable-build
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (GoReleaser build configuration, build invocations, self-containment verification); ADR-1, ADR-2

---

This feature's interface is a **declarative build configuration** (`.goreleaser.yaml`) plus its **invocation surface** (`goreleaser build`) and the **structural output contract** (`dist/`) that downstream specs (022, 023) and the self-containment verification consume. There is no runtime/API/CLI surface on the `glassfrog` binary itself — the artifact *is* the config and its outputs.

---

## Surface

### Build configuration file — `.goreleaser.yaml` (repository root)

A single GoReleaser v2 config carrying **only** a `builds` block in this feature.

Top-level keys present in 021:

| Key | Value | Notes |
|---|---|---|
| `version` | `2` | GoReleaser v2 config schema |
| `project_name` | `glassfrog` | names the project/build id prefix |
| `builds` | one entry (below) | the sole build concern |

The single `builds` entry:

| Field | Value (021) | Purpose |
|---|---|---|
| `id` | `glassfrog` | build id; becomes the `dist/` subdir prefix |
| `main` | `.` | the module root package (`main.go` at repo root) |
| `binary` | `glassfrog` | output executable name (matches the root command's `Use`) |
| `env` | `["CGO_ENABLED=0"]` | the self-containment lever |
| `goos` | `["darwin", "linux"]` | the two supported operating systems |
| `goarch` | `["amd64", "arm64"]` | the two supported architectures |
| `flags` | `["-trimpath"]` | path-independent, reproducible builds across hosts |
| `ldflags` | empty in 021 (the 023 seam) | **left free of version logic**; 023 injects `-X` here |

`goos × goarch` yields exactly the four supported targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64. **No `windows`.**

Keys intentionally **absent** in 021 (owned by 022 Automated Release Pipeline): `archives`, `checksum`, `release`, `brews`, `snapshot` (name template), `changelog`. The file is a deliberately partial contract that 022 extends additively.

### Build invocations (maintainer / CI surface)

| Command | Produces |
|---|---|
| `goreleaser build --snapshot --clean` | all four target binaries under `dist/` (snapshot = no git tag required) |
| `goreleaser build --snapshot --clean --single-target` | the host-platform binary only — the **local development build** |
| `goreleaser build … --id glassfrog` | restricts to the `glassfrog` build (forward-compat once 022 adds more) |

### `dist/` output contract

- Per-target binaries land at `dist/<id>_<goos>_<goarch>[_<microarch>]/glassfrog` — e.g. `dist/glassfrog_linux_amd64_v1/glassfrog`, `dist/glassfrog_darwin_arm64/glassfrog`. (GoReleaser appends a GOAMD64 microarch suffix such as `_v1` for amd64 targets.)
- `dist/artifacts.json` — the **canonical, stable discovery mechanism**: a structured list of every produced artifact with its `path`, `goos`, `goarch`, and `name`. Consumers (the verification, 022) read this rather than parsing directory names, so the microarch suffix never has to be hardcoded.

### Self-containment verification (test surface)

Invoked through the project's existing test command (`go test ./...`); the verification lives as Go tests, not a separate tool.

| Item | Contract |
|---|---|
| Binary under test | a host-target `glassfrog` — discovered from `dist/artifacts.json` when present, else built on the fly for the host target with `CGO_ENABLED=0` |
| Execute check | runs `glassfrog version` (a no-network, `version`-class command) and asserts exit code `0` |
| Linkage check | inspects the binary's dynamic-library dependencies against a per-platform OS-only allowlist (below) |
| Config-guard check | reads `.goreleaser.yaml` and asserts the exact four-target matrix and `CGO_ENABLED=0` |

---

## Interactions

### Configuration extension model (additive, shared file)

The `.goreleaser.yaml` is a single contract extended across specs, never forked:

- **021** owns `version`, `project_name`, and the `builds` block.
- **023 Version Embedding** sets `builds[0].ldflags` (e.g. `-X main.version={{.Version}}`) — the only field 021 leaves open.
- **022 Automated Release Pipeline** appends `archives`, `checksum`, `release`, `brews` (and snapshot/changelog naming) to the same file. 022 must not alter or re-decide the `builds` block.

### Invocation-to-output flow

`goreleaser build --snapshot --clean` → compiles each target with `CGO_ENABLED=0` → writes binaries under `dist/` and a `dist/artifacts.json` manifest. `--single-target` narrows the matrix to the host's GOOS/GOARCH for the local dev build. Both paths apply the identical `builds` config, so a local host binary and the matrix binary for that same target are byte-for-byte equivalent — the spec's "one build entry point, no second path."

### Verification flow

`go test ./...` → the verification locates a host-target binary (manifest-first, host-build fallback) → executes `glassfrog version` (assert exit 0) → inspects linkage against the allowlist → reports pass/fail. The config-guard test runs independently of any built binary, so matrix/cgo drift is caught even on a runner that can't execute a foreign target.

### OS-only linkage allowlist (the self-containment criterion, per platform)

| Platform | Permitted dynamic dependencies | Anything else |
|---|---|---|
| Linux | none — statically linked (or only the dynamic loader `ld-linux*`) | a XII violation → fail |
| macOS | only system libraries under `/usr/lib/**` and `/System/Library/**` (e.g. `libSystem.B.dylib`) | a XII violation → fail |

This encodes "self-contained, not fully-static": a macOS Go binary always links the OS-provided `libSystem`, which is permitted; a dependency outside the allowlist is the failure the check exists to catch.

---

## Error Communication

Constraint violations and failure behaviors (the consumer-visible contract):

| Condition | Behavior |
|---|---|
| `.goreleaser.yaml` declares a target outside the four (e.g. `windows`) | config-guard test **fails**, naming the unexpected target |
| `.goreleaser.yaml` omits `CGO_ENABLED=0` (or sets it `1`) | config-guard test **fails** |
| `.goreleaser.yaml` is missing one of the four required targets | config-guard test **fails** (change-detector: a missing target fails as loudly as an extra one) |
| Produced binary exits non-zero on `glassfrog version` | verification **fails** — the binary does not run on a clean host of its target |
| Produced binary links a library outside the OS allowlist | verification **fails**, naming the offending dependency |
| Any target fails to compile during `goreleaser build` | the build **fails as a whole** and emits no partial `dist/` set (GoReleaser default; matches the spec's "a failed target fails the whole release build") |
| `goreleaser` not installed when running the verification locally | the verification **falls back** to a host `go build` (`CGO_ENABLED=0`) — it does **not** error; `go test` stays runnable without GoReleaser |

---

## Consistency Notes

- **No sibling interface files** — this feature has only a specification touchpoint; there is no API/CLI/UI/events accord. The `glassfrog` binary's own CLI surface is owned by the command specs (001–004), not this feature.
- **No existing `accords/` patterns** — `accords/` is absent in this repository, so there are no established interface conventions to conform to or deviate from.
- **Alignment with project conventions** — Go toolchain and `CGO_ENABLED=0` follow PROJECT.md's stack and the foundational build decision (DECISIONS, 001). The verification reuses the existing subprocess-exec pattern in `internal/cli/smoke_test.go` (re-execute a binary, assert process exit status) and the project's `go test` suite — no new test framework.
- **Cross-references** — the `builds.ldflags` seam is consumed by 023; the additive `archives`/`release`/`brews` sections are owned by 022. Both are pinned in DECISIONS (021 entries) so the boundary survives.

---

## Assumptions

- **`.goreleaser.yaml` (not `.yml`), GoReleaser v2** — the modern default extension and config schema (`version: 2`). If the project later standardizes on `.yml`, the contract is unchanged but the path differs.
- **`version`-class execute probe** — the execute check runs `glassfrog version`, which already exists and makes no network call (used by `internal/cli/smoke_test.go`). It is the lightest no-network proof that the binary's loader and runtime start.
- **Manifest-first discovery** — consumers locate binaries via `dist/artifacts.json` rather than directory-name parsing, so GoReleaser's amd64 microarch suffix (`_v1`) and future naming changes don't break discovery.
