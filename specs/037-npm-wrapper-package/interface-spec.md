# Interface Accord: NPM Wrapper Package — Specification

**Feature**: 037-npm-wrapper-package
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (umbrella + 4 platform packages, the in-repo sources vs CI-generated layout); ADR-1 (optionalDependencies topology), ADR-2 (CI-generated, not committed, scaffolding), ADR-3 (027-conformant deterministic-URL + sha256 fallback, base-URL seam), ADR-4 (zero-dependency launcher + passthrough + backstop), ADR-5 (unsupported-platform refusal), ADR-6 (version coupling), ADR-7 (release.yml publish job, OIDC trusted publishing)

---

## Surface

This feature is **one declarative artifact set**: a small body of in-repo npm sources that a CI job assembles into **five published npm packages** per release. It adds no command to the Go CLI — the published `glassfrog` command is a launcher that transparently execs the existing binary.

### Published packages

| Package | Role | Platform-gated | Bundles binary | Has `bin` |
|---|---|---|---|---|
| `@luscii-healthtech/glassfrog` | Umbrella — exposes the `glassfrog` command, lists the four platform packages as `optionalDependencies` | no | no | yes (the launcher) |
| `@luscii-healthtech/glassfrog-darwin-arm64` | Platform package | `os: darwin`, `cpu: arm64` | yes | no |
| `@luscii-healthtech/glassfrog-darwin-x64` | Platform package | `os: darwin`, `cpu: x64` | yes | no |
| `@luscii-healthtech/glassfrog-linux-arm64` | Platform package | `os: linux`, `cpu: arm64` | yes | no |
| `@luscii-healthtech/glassfrog-linux-x64` | Platform package | `os: linux`, `cpu: x64` | yes | no |

The npm `os`/`cpu` vocabulary maps from GoReleaser's: `darwin`→`darwin`, `linux`→`linux`; `amd64`→`x64`, `arm64`→`arm64`. Only the umbrella declares `bin`, so exactly one `glassfrog` command is linked; platform packages ship the binary at a known path with **no** `bin` (avoids a conflicting second link).

### Consumer invocation

| Form | Command |
|---|---|
| One-off | `npx @luscii-healthtech/glassfrog <args…>` |
| Global | `npm i -g @luscii-healthtech/glassfrog` then `glassfrog <args…>` |
| Pinned | `npm i -g @luscii-healthtech/glassfrog@1.3.0` (or `@1.4.0-rc.1` for a pre-release) |
| Project dependency | `npm i @luscii-healthtech/glassfrog` (runs via `npx glassfrog` / package scripts) |

### umbrella `package.json` — required contract

| Field | Contract |
|---|---|
| `name` | `@luscii-healthtech/glassfrog` |
| `version` | the release version (tag without leading `v`; see Version normalisation) |
| `bin` | `{ "glassfrog": "bin/glassfrog" }` — the launcher |
| `optionalDependencies` | each platform package pinned to the **exact** version: `"@luscii-healthtech/glassfrog-darwin-arm64": "=1.3.0"`, … (all four) |
| `scripts.postinstall` | runs `postinstall.js` |
| `files` | includes `bin/glassfrog` and `postinstall.js` |
| `publishConfig.access` | `public` (scoped package, public registry) |

### platform `package.json` — required contract

| Field | Contract |
|---|---|
| `name` | `@luscii-healthtech/glassfrog-<os>-<cpu>` |
| `version` | the same release version as the umbrella |
| `os` | `["darwin"]` or `["linux"]` |
| `cpu` | `["arm64"]` or `["x64"]` |
| `files` | includes the bundled binary (e.g. `bin/glassfrog`) |
| `bin` | **absent** |
| `publishConfig.access` | `public` |

### In-repo sources (committed — the single source of truth)

Under a dedicated directory (e.g. `npm/`):

| Element | Contract |
|---|---|
| Launcher (`bin/glassfrog`) | Zero-dependency CommonJS. Resolves the platform binary, execs it, propagates exit code/signal; runtime backstop on unresolved binary (ADR-4). |
| `postinstall.js` | Zero-dependency CommonJS. Detects platform; if a bundled platform binary is resolvable → no-op; else fallback download+verify+place; refuse on unsupported platform / mismatch / download failure (ADR-3, ADR-5). |
| umbrella `package.json` template | The umbrella fields above, with version + `optionalDependencies` versions stamped at build time. |
| Generator (e.g. `npm/build.mjs`) | Emits the umbrella + four platform package directories into the output dir from the version + the verified `dist/` binaries, applying the os/cpu map and `=version` pinning (ADR-2). |
| Tests (`node --test`) | Hermetic unit tests for launcher, postinstall, and generator (see Interactions → Test invocation). |

The per-platform package **directories are not committed** — they are generated. Only the four elements above live in the repo (one copy each).

### Generated output (gitignored)

A build directory (e.g. `dist/npm/`, added to `.gitignore` alongside `/dist/`) holds the assembled `@luscii-healthtech/glassfrog/` umbrella and four `…/glassfrog-<os>-<cpu>/` package directories, each ready to `npm publish`.

### Configuration schema

| Variable | Scope | Required | Default | Description |
|---|---|---|---|---|
| `GLASSFROG_DOWNLOAD_BASE_URL` | install-time (postinstall) | no | `https://github.com` | Base URL for the fallback download. The test/mirror seam (ADR-3). The **same** variable name 027 defines — both acquisition channels fetch the same release assets (see Consistency Notes). |

The release version is supplied by the publish job from the git tag, not read at install time. Package names, the os/cpu map, and the checksum algorithm (sha256) are fixed, never configured. The launcher reads no configuration of its own — every argument and the environment pass through to the binary unchanged.

---

## Interactions

**Install-time flow (consumer):** `npm`/`npx` resolves the umbrella and — by `os`/`cpu` — the one matching platform package (its binary is bundled, so the **primary path needs no network**). The umbrella's `postinstall.js` then runs: if a bundled platform binary resolves → done; otherwise (optional deps omitted, or the registry lacked the platform package) → **fallback**: map host → supported target, construct the release asset URLs from 022's template against `GLASSFROG_DOWNLOAD_BASE_URL`, download archive + checksums, verify the archive's sha256 against its checksums entry, extract with system `tar`, and place the binary atomically inside the umbrella (temp → verify → move; nothing placed on failure).

**Runtime flow:** `glassfrog <args>` → the launcher resolves the binary (the platform package via `require.resolve('@luscii-healthtech/glassfrog-<os>-<cpu>/bin/glassfrog')`, else the postinstall-placed path), `spawn`s it with `stdio: 'inherit'` and `process.argv.slice(2)`, and exits with the child's exact exit code (re-raising a terminating signal). All output and the exit code are the binary's — the launcher adds nothing.

**Publish flow (release time):** the `npm-publish` job in `release.yml`, `needs: [build, verify]`, downloads the verified `dist/` artifact, runs the generator at the release version, then publishes — **platform packages first, umbrella last** — with `npm publish --access public --provenance`, authenticated by GitHub OIDC (`id-token: write`; no stored token). Ordering is load-bearing: the umbrella's pinned `optionalDependencies` must already exist on the registry.

**Generator invocation (local/CI):** `node npm/build.mjs` with the version and the `dist/`/output locations (exact flag names are a Builder detail); produces the gitignored package directories. Runnable locally for testing without publishing.

**Version normalisation:** the npm version is the git tag **without** the leading `v` (`v1.3.0`→`1.3.0`, `v1.4.0-rc.1`→`1.4.0-rc.1`) — the same `<ver>` transform 022/027 use. The umbrella and all four platform packages publish at that exact version; `optionalDependencies` pin `=<ver>`. The bundled/downloaded binary reports `v<ver>` via 023, so `glassfrog --version` equals the installed package version with the `v` restored. Pre-release tags publish as npm pre-release versions (installed only when explicitly requested).

**Test invocation (ADR-3 seam):** `node --test` drives the launcher against a stub binary (asserting argv/exit/signal passthrough and the no-binary backstop) and the postinstall against a local fixture server via `GLASSFROG_DOWNLOAD_BASE_URL` (fake archive + checksums) — happy path, checksum mismatch, unsupported platform, missing `tar`. No network, no real registry, mirroring 027's hermetic Go exec-test.

---

## Error Communication

Two distinct surfaces fail differently:

**Install-time (postinstall)** — a non-zero exit fails `npm install` with the script's stderr message; nothing runnable is left behind (027's verified-or-nothing atomicity).

Every error names both its **cause** and a **next step** (CONSTITUTION II, Action Transparency — non-negotiable).

| Condition | Behavior (cause → next step) |
|---|---|
| Unsupported platform (Windows, unsupported arch) | stderr message naming the detected `platform`/`arch` and the supported set (`darwin`/`linux` × `x64`/`arm64`); **next step**: "use a supported platform, or install via the install-script (027) or Homebrew (036) channel". Nothing placed; non-zero exit (fails the install). |
| Checksum mismatch (fallback) | stderr message naming the integrity failure; **next step**: "re-run the install to retry the download; if it persists, the release asset may be corrupt — report it". **No binary placed**; temp cleaned; non-zero exit. |
| Download failure / missing asset (fallback) | stderr message naming the failing URL/asset; **next step**: "check network access to the release host and re-run the install; verify the requested version exists". Nothing placed; non-zero exit. |
| Missing `tar` (fallback) | stderr message naming the missing extractor; **next step**: "install `tar` (preinstalled on macOS/Linux) and re-run, or use the install-script (027) channel". Before any placement; non-zero exit. |
| Bundled platform binary present | postinstall is a no-op success; no network. |

**Runtime (launcher)** — exit codes are the **binary's own**, propagated verbatim, so the npm channel preserves 004's convention (0 success / 1 internal / 2 usage / 3 API / 4 permission / 5 rate-limit / 6 network) identically to a direct call.

| Condition | Behavior |
|---|---|
| Binary runs | The launcher exits with the binary's exit code; signals re-raised. Output is the binary's, unmodified. |
| No binary resolves (e.g. postinstall skipped via `--ignore-scripts` on an otherwise-supported host with no optional dep) | The launcher emits the same clear refusal (detected platform + supported set + "reinstall without `--ignore-scripts`") to stderr and exits non-zero — never a confusing crash (ADR-4 backstop). |

All diagnostics go to **stderr**; the launcher writes nothing to stdout of its own.

---

## Consistency Notes

- **Hard dependency on spec 022's asset-name template.** The fallback constructs `glassfrog_<ver>_<os>_<arch>.tar.gz` + `glassfrog_<ver>_checksums.txt` (sha256, `<ver>` = tag without `v`) from 022's `.goreleaser.yaml` `name_template` (`022/interface-spec.md`). A change to 022's template would 404 the fallback. Mitigation (mirrors 027): the test fixtures encode these exact names so drift breaks a test; a cross-reference comment belongs in the generator/postinstall and `.goreleaser.yaml`. The **publish path bundles `dist/` binaries directly** (not URL-constructed), so only the install-time fallback depends on the URL shape.
- **Shared download-base-URL variable with 027, deliberately.** `GLASSFROG_DOWNLOAD_BASE_URL` (default `https://github.com`) is the **same** name the install script (027) defines — both are acquisition channels fetching the same release assets, so one override mirrors both to a test server. It stays distinct from the CLI runtime's `GLASSFROG_BASE_URL` (the API endpoint, 008); the npm channel reads neither `GLASSFROG_BASE_URL`, `GLASSFROG_TOKEN`, nor `GLASSFROG_OUTPUT` (runtime-CLI concerns).
- **No `interface-cli.md`.** The launcher's `glassfrog` command is pure pass-through to the existing CLI surface (001–048); it defines no new commands or flags. Its only contract — transparent argv/output/exit-code passthrough — is captured here as a structural contract of the launcher, so a separate CLI accord would duplicate the existing surface.
- **Exit-code split mirrors 027's reasoning.** Runtime exit codes are 004's (propagated verbatim by the launcher). Install-time failures fail the npm install with a non-zero postinstall and a clear message; the postinstall does not adopt 004's API-side codes (3–6), which have no meaning during installation — same separation 027 documents between its installer scheme and the CLI's.
- **Sibling acquisition channels.** Install Script (027) and Homebrew Tap (036) install the same released binary by other routes from the same upstream artifacts; all three are independent and unaware of each other. There is no `accords/` directory; conventions follow PROJECT.md and the sibling declarative-artifact accords (022, 027), which set the single-`interface-spec.md` precedent this accord follows.
- **OIDC trusted publishing — one-time setup.** `npm publish --provenance` over GitHub OIDC needs each of the five package names registered as a trusted publisher on npmjs.com before the first publish (a maintainer ops step, like 024's branch-protection bootstrap). Documented as a release prerequisite, not a code contract.
