# Validate: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Round**: 1 of 3
**Date**: 2026-06-13
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/runtime-dependent-distribution/npm-wrapper-package.feature, PROJECT.md
**Implementation files**: 10 under `npm/` (lib/platform.js, bin/glassfrog, postinstall.js, build.mjs, package.json, 5 test files) + CI/release wiring (.github/workflows/test.yml, release.yml), .gitignore, README.md, scripts/npm-trusted-publishers.md

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 validation scenarios satisfied. Suite: 24 `node --test` tests pass.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 non-validation scenarios covered)

| Scenario | Status | Implementation |
|---|---|---|
| npx resolves and runs the matching platform binary | ✓ Covered | `npm/lib/platform.js:resolveBinary`/`resolveBundledBinary` (require.resolve of the platform package) → `npm/bin/glassfrog:run` spawnSync |
| Pinned global install places the matching binary | ✓ Covered | `npm/build.mjs:umbrellaPackageJson` (=version pinning) + `npmVersion`; placed binary corresponds to package version |
| Fallback download verifies before placing the binary | ✓ Covered | `npm/postinstall.js:postinstall` (download → sha256 verify at :153 → rename at :181) |
| Unsupported platform is refused at install | ✓ Covered | `npm/postinstall.js:103` throws `unsupportedMessage`; nothing placed |
| Checksum mismatch aborts the fallback install | ✓ Covered | `npm/postinstall.js:154` `actual !== expected` → throw; temp removed in `finally`, nothing placed |
| Offline install uses the bundled platform package | ✓ Covered | `npm/postinstall.js:107` no-op when `resolveBundledBinary` succeeds — no network (test asserts 0 server hits) |
| Exit code and arguments pass through unchanged | ✓ Covered | `npm/bin/glassfrog:40` spawnSync `stdio: 'inherit'`, argv forwarded; `:59` propagates `child.status` |

Note: the "reported version is the release tag (`vX.Y.Z`)" clause depends on Version Embedding (023), an integration boundary. The npm channel's responsibility — placing the binary that matches the installed package version — is met; the `v`-prefixed `--version` output is produced by the Go binary's ldflags seam (023), which this channel consumes, not implements.

---

## Acceptance Criteria

**Status**: Pass (6 of 6 tasks complete; criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 launcher + lib + tests | ✓ Met | argv/exit/signal passthrough + no-binary backstop (`bin/glassfrog`); zero npm deps (Node built-ins); `launcher.test.js` + `platform.test.js` |
| T002 postinstall + tests | ✓ Met | no-op-when-bundled, verified fallback, refusals for unsupported/mismatch/missing-tar; `GLASSFROG_DOWNLOAD_BASE_URL` seam; `postinstall.test.js` |
| T003 generator + template + tests | ✓ Met | emits umbrella + 4 platform packages; os/cpu map; `=version` pinning; bin only on umbrella; `generator.test.mjs` (verified by end-to-end CLI smoke run) |
| T004 CI wiring + integration + @wip | ✓ Met | `node-test` job in shared `test.yml` (picked up by 024 & 029); `integration.test.js` install→launch; 9 `@wip` removed, 3 `@validation @wip` held |
| T005 npm-publish job + gitignore | ✓ Met | `release.yml` `npm-publish` `needs: [build, verify]`, reuses dist/, platform-first, OIDC `--provenance`, `id-token: write`, no NPM_TOKEN; dist/npm under `/dist/` |
| T006 docs | ✓ Met | README "Install via npm" (npx / `-g` / pinned, platforms, siblings) + `scripts/npm-trusted-publishers.md` |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Evidence |
|---|---|---|
| Umbrella `package.json` (name, `bin`, `optionalDependencies` =version ×4, `scripts.postinstall`, `files`, `publishConfig.access`) | ✓ Conformant | `npm/package.json` template + `build.mjs:umbrellaPackageJson` (generated shape verified) |
| Platform `package.json` (scoped `<os>-<cpu>` name, `os`/`cpu`, bundled binary in `files`, **no** `bin`, public) | ✓ Conformant | `build.mjs:platformPackageJson` (generated shape verified) |
| In-repo sources (launcher, postinstall, umbrella template, generator, tests) | ✓ Conformant | present under `npm/` |
| Configuration schema (`GLASSFROG_DOWNLOAD_BASE_URL`, default `https://github.com`, install-time) | ✓ Conformant | `postinstall.js:95` |
| Runtime flow (`require.resolve('@luscii-healthtech/glassfrog-<os>-<cpu>/bin/glassfrog')`, spawn inherit, exit/signal) | ✓ Conformant | `lib/platform.js:79`, `bin/glassfrog` |
| Error Communication (cause + next step on each install-time refusal; verbatim exit-code propagation at runtime) | ✓ Conformant | `lib/platform.js` messages + `postinstall.js` refusals; launcher propagation |

Observation (not a finding): the umbrella `files` is `["bin/glassfrog", "postinstall.js", "lib/"]` — a superset of the interface's stated `bin/glassfrog` + `postinstall.js`. `lib/` is required because both the launcher and postinstall `require('../lib/platform.js')`; the interface specifies the field "includes" those entries, so the superset conforms.

---

## Non-Behavior Absence

**Status**: Pass (no excluded capability present)

| Non-behavior | Status | Evidence |
|---|---|---|
| No binary build / Go toolchain | ✓ Absent | generator extracts the prebuilt binary from the 022 archive; no compilation |
| No Windows/unsupported artifact published | ✓ Absent | `SUPPORTED` is the four macOS/Linux × x64/arm64 targets only; generator emits exactly four |
| No unverified binary placed (fallback) | ✓ Absent | sha256 verified (`postinstall.js:153-159`) before the rename (`:181`) |
| No network when bundled dep present | ✓ Absent | `postinstall.js:107` returns before any download; test asserts 0 server hits |
| No re-parsing of args/output/exit code | ✓ Absent | launcher forwards argv + inherits stdio + propagates exit code; its own diagnostics fire only on the no-binary backstop |
| No producing archives/checksums/notes/signing/version-bump | ✓ Absent | channel consumes 022's artifacts; version is derived from the tag, never bumped |
| No editing shell profiles/PATH | ✓ Absent | linking is npm's `bin` mechanism; no PATH writes anywhere |

---

## @wip Lifecycle Completion

**Status**: Pass

The feature file carries 0 bare `@wip` tags and exactly 3 `@validation @wip` tags. The nine non-validation scenarios referenced by checked tasks had `@wip` removed (T004); the three `@validation` scenarios remain held out for this validation pass, as intended. No scenario referenced by a checked task retains an unexpected `@wip`.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| The placed binary's version matches the package and the release tag | ✓ Satisfied | `postinstall.js` builds `tag = v${version}` from the umbrella's stamped version and downloads `assetNames(version, …)`; `build.mjs:npmVersion` stamps every package and the archive lookup at the same tag-minus-`v`. The package-version → matching-release-binary coupling is structural; the `v`-prefixed `--version` is 023's embedding (integration boundary), relied on, not re-implemented. |
| Each supported platform resolves exactly its own binary | ✓ Satisfied | `build.mjs:platformPackageJson` sets `os:[os]`/`cpu:[cpu]` per package (npm gates resolution); the launcher resolves `require.resolve(platformPackageName(detectTarget(host).os, .cpu)/bin/glassfrog)` — host-detected only, no cross-OS/arch path. `platform.test.js` asserts the map and the null-on-unsupported. |
| A corrupted fallback download never becomes runnable (verification gates the fallback) | ✓ Satisfied | `postinstall.js` downloads into a temp dir inside the package, computes sha256, and only renames into `placedBinaryPath` after `actual === expected`; on mismatch it throws and the `finally` removes the temp — no runnable binary ever appears, non-zero exit. `postinstall.test.js` "checksum mismatch" asserts nothing placed. |

---

## Verdict: Ready

All 5 conformance dimensions pass with 0 findings, and all 3 held-out validation scenarios trace to clear implementation code paths. The implementation conforms to the specification: the four-target optional-dependency topology, the offline-primary / verified-fallback install behavior, transparent argv/exit-code passthrough, the unsupported-platform and checksum-mismatch refusals, the version coupling, and the OIDC publish path are all present and exercised by a hermetic 24-test suite. No excluded capability was found. The only spec clauses that rest on external seams (the `vX.Y.Z` `--version` output via 023, npm's own os/cpu resolution and bin-linking) are correctly consumed as integration boundaries rather than re-implemented.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. Before the first real npm release, complete the one-time per-package OIDC trusted-publisher registration documented in `scripts/npm-trusted-publishers.md` (a maintainer ops prerequisite, not a code gap).
