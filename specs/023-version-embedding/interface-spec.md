# Interface Accord: Version Embedding — Specification

**Feature**: 023-version-embedding
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (version resolver + injection seam); ADR-1 (pure resolver), ADR-2 (ldflags injection), ADR-3 (empty default + placeholder), ADR-4 (`vX.Y.Z` shape)

---

This feature's interface is two coupled contracts: a **declarative build-configuration edit** (the `builds.ldflags` line in `.goreleaser.yaml`) and a **module structural contract** in `internal/cli` (the version variable, the pure resolver, and the resolved value the two existing 003 wiring sites read). 023 adds **no new CLI surface** — the `--version` flag and the `version` command are owned, named, and formatted by Help & Version (003); 023 only changes the *value* those surfaces report. There is no runtime/API/event surface.

---

## Surface

### Declarative artifact — `.goreleaser.yaml` `builds[0].ldflags`

023 fills the single `ldflags` entry that 021 left empty and 022 left untouched. The whole rest of the file (`version`, `project_name`, the `builds` matrix, `flags`, and 022's `archives`/`checksum`/`release`) is unchanged.

| Field | Value (023) | Purpose |
|---|---|---|
| `builds[0].ldflags` | `["-X github.com/Luscii/cli-glassfrog/internal/cli.version=v{{ .Version }}"]` | stamps the version into the `internal/cli.version` variable at link time |

- **Target symbol**: `github.com/Luscii/cli-glassfrog/internal/cli.version` — the package-level var (not `main.version`; the var lives in `internal/cli` per 003's version-unify, ADR-3). 021's interface forecast `main.version`; 023 corrects it to where the var actually is (see Consistency Notes).
- **Version template token**: `v{{ .Version }}`. GoReleaser's `.Version` is the tag with the leading `v` stripped (`v1.4.0` → `1.4.0`); prefixing a literal `v` restores the `vX.Y.Z` shape and, in snapshot mode, yields a `v`-prefixed snapshot version (e.g. `v1.4.1-next-<commit>`) rather than the *previous* tag that `{{ .Tag }}` would return. A pre-release tag `v1.4.0-rc.1` → `.Version` `1.4.0-rc.1` → injected `v1.4.0-rc.1` (suffix preserved).
- **Out of the version contract**: whether to also pass `-s -w` (symbol/DWARF stripping for binary size) is the Builder's discretion — it does not affect version resolution or `runtime/debug` build info, and is not required by this accord. The required content of `ldflags` is exactly the one `-X` flag above.

### Module structural contract — `internal/cli`

| Symbol | Signature / value | Contract |
|---|---|---|
| `version` | `var version string` | the injection target; **empty by default** (empty = "not injected"). Set only by the `-X` ldflags flag at build time. |
| `resolveVersion` | `func resolveVersion(injected string, info *debug.BuildInfo, ok bool) string` | **pure**; implements the three-tier precedence (below); contains no formatting and no I/O. |
| `resolvedVersion` | `func resolvedVersion() string` | production wrapper: calls `runtime/debug.ReadBuildInfo()` and delegates to `resolveVersion(version, info, ok)`. |
| placeholder | unexported constant, value `"0.0.0-dev"` | the tier-3 last-resort value `resolveVersion` returns; non-empty, semver-shaped, clearly non-release. |

**`resolveVersion` precedence** (first match wins):
1. `injected != ""` → return `injected`.
2. `ok && info.Main.Version != ""` → return `info.Main.Version` **verbatim** (no trim, no normalization — `vX.Y.Z` tags, `v0.0.0-<ts>-<commit>` pseudo-versions, and Go's `(devel)` marker all pass through unchanged).
3. otherwise → return the placeholder constant.

**Wiring (003 sites, re-pointed — not new surface)**: `configureHelpAndVersion` sets `root.Version = resolvedVersion()`; `newVersionCommand`'s `RunE` prints `resolvedVersion()`. Both read the same resolved value, preserving 003's byte-identical `--version`/`version` output (version-unify, ADR-3).

### Output-value contract (what `--version` / `version` report)

| Build/install path | `version` var | Reported value |
|---|---|---|
| `goreleaser release` (tag `v1.4.0`) — via 022 | `v1.4.0` (injected) | `v1.4.0` |
| `goreleaser release` (pre-release tag `v1.4.0-rc.1`) | `v1.4.0-rc.1` (injected) | `v1.4.0-rc.1` |
| `goreleaser build --snapshot` — via 021 (local/CI) | `v…-next-<commit>` (injected) | the snapshot version |
| `go install …@v1.3.2` (tagged source) | `""` | `v1.3.2` (build info, verbatim) |
| `go install …@latest` (untagged) | `""` | `v0.0.0-<ts>-<commit>` pseudo-version (verbatim) |
| plain `go build` | `""` | `(devel)` (build info, verbatim) |
| no build info at all | `""` | `0.0.0-dev` (placeholder) |

The value is **never empty** in any path.

---

## Interactions

### Configuration extension model (additive, shared file)

The `.goreleaser.yaml` is one contract extended across specs, never forked. With 023, it is **complete**:
- **021** owns `version`, `project_name`, the `builds` matrix, and `flags`.
- **023** sets `builds[0].ldflags` (the only field 021 left open) — this accord.
- **022** owns `archives`, `checksum`, `release` and must not touch `builds`/`ldflags`.
- **Homebrew Tap (036)** later adds only a `brew` section.

### Injection-to-report flow

GoReleaser computes `.Version` from git on **every** invocation, so the `-X` flag injects on both the release path (`goreleaser release`, 022 — tag-derived) and the snapshot path (`goreleaser build --snapshot`, 021). Plain `go build`/`go install` apply no ldflags, leaving `version` empty so the resolver consults build info. At report time: `--version` flag or `version` command → cobra/003 → `resolvedVersion()` → `resolveVersion(version, debug.ReadBuildInfo())` → the single value both surfaces print. No runtime network or VCS access — both inputs are baked into the binary at build time.

### Testability seam

`resolveVersion` is pure and exercised offline with crafted `(injected, *debug.BuildInfo, ok)` triples (every precedence branch, plus the never-empty invariant). The production `resolvedVersion()` is the only place that reads `debug.ReadBuildInfo()`. End-to-end, 003's `runAssembled(t, ver, …)` helper drives the injected branch by setting the `version` var; both request forms must report `ver` byte-identically.

---

## Error Communication

This feature has no error states of its own — version resolution always returns a non-empty string and never fails. The consumer-visible contract is about *misconfiguration being caught*, not runtime errors:

| Condition | Behavior |
|---|---|
| Reported version is empty | **contract violation** — the never-empty invariant; a resolver unit test fails. |
| `.goreleaser.yaml` `ldflags` blanked back to empty / `-X` removed | not caught by the `internal/build` config-guard (it ignores `ldflags`); a **focused config assertion** that the real config injects `internal/cli.version` is the regression guard (see Consistency Notes). A blanked seam degrades releases to the build-info/placeholder value — a silent wrong-version, hence the dedicated guard. |
| `-X` symbol path stale (var renamed, flag not updated) | Go silently ignores `-X` for an unknown symbol → `version` stays empty → resolver falls to build info/placeholder. Same focused config assertion (paired with a release-shape check) is the guard. |
| Build info absent (unusual build mode) | not an error — the designed tier-3 placeholder path; reports `0.0.0-dev`. |
| `--version` and `version` disagree | **contract violation** — both must read `resolvedVersion()`; 003's parity test fails. |

---

## Consistency Notes

- **No new CLI accord** — the `--version` flag and `version` command are 003's surface (interface + implemented). 023 adds/renames/reorders no command or flag; it changes only the reported *value*. Hence no `interface-cli.md` — that would be scope expansion.
- **Correction to 021's forecast** — 021's `interface-spec.md` wrote the seam as `-X main.version={{.Version}}`. 023 pins `-X …/internal/cli.version=v{{ .Version }}` instead: the version var lives in `internal/cli` (003 ADR-3, the version-unify single source), not `main`, and the `v{{ .Version }}` token (vs `{{ .Version }}`/`{{ .Tag }}`) is chosen for `vX.Y.Z` shape consistency with Go build-info across release and snapshot builds. Documented divergence, not a silent change.
- **Pinned in DECISIONS** — the filled-seam decision and the three-tier resolution model are recorded as 023 entries; with this, `.goreleaser.yaml` is complete (builds=021, archives/checksum/release=022, ldflags=023).
- **No existing `accords/` patterns** — `accords/` is absent in this repository, so there are no established interface conventions to conform to or deviate from.
- **Alignment with project conventions** — `runtime/debug` is stdlib (no new dependency); the pure-function-over-injected-seam shape follows the established `formatMe`/`validateInclude` test pattern (LEARNINGS). The resolver lives in `internal/cli` alongside 003's version handling — no new package, no import cycle.

---

## Assumptions

- **GoReleaser `.Version` strips the leading `v`** — the `v{{ .Version }}` token assumes GoReleaser v2's documented behavior that `.Version` is the tag without its `v` prefix while `.Tag` keeps it. If a future GoReleaser changes this, the token is the single line to revisit; the `vX.Y.Z`-shape requirement (ADR-4) is unchanged.
- **`runtime/debug.ReadBuildInfo().Main.Version` is the source-build version** — Go records the module version here for `go install path@version` (real tag or pseudo-version) and `(devel)` for a plain `go build` of the main module. This is the toolchain-native, build-time-embedded source the fallback reads.
- **Placeholder literal `0.0.0-dev`** — retained from the current code as the recognizable non-release marker. The behavioral requirement is "non-empty and clearly not a release"; the exact literal is not a published knob.
