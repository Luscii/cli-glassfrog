# Plan: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Role**: Shaper
**Inputs**: `specs/037-npm-wrapper-package/spec.md`, `PROJECT.md`, `.score/memory/DECISIONS.md` (50+ entries; distribution cluster 021/022/023/027 actively consulted), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md` (background); grounded against the live `.goreleaser.yaml`, `.github/workflows/release.yml`, and `install.sh` (027). Reference studied (and deliberately diverged from on one point): `ailign-cli/cli` release workflow.

---

## System Architecture

The NPM Wrapper is an **acquisition channel** that makes the released `glassfrog` binary runnable through the Node toolchain. It adds no CLI logic — it packages and places the same self-contained binary that `021` builds, `022` packages, and `023` versions. It conforms to the acquisition-channel pattern `027` established (deterministic asset URLs from `022`'s `name_template`, sha256 verification against the checksums file, a download-base-URL seam for hermetic tests).

The channel has two halves: **package sources** (committed, testable) and a **release-time publisher** (CI, generates and publishes).

**Package topology** (the esbuild/ailign model):

```
@luscii-healthtech/glassfrog       (main "umbrella" package — published to npm)
├── bin/glassfrog                 launcher: a zero-dependency CommonJS shim (the `glassfrog` command)
├── postinstall.js                fallback: detect platform → download+verify → place, or refuse
├── package.json                  declares the 4 platform packages as optionalDependencies (pinned =X.Y.Z)
└── optionalDependencies:
    ├── @luscii-healthtech/glassfrog-darwin-arm64   (each: package.json with os/cpu gating + the bundled binary)
    ├── @luscii-healthtech/glassfrog-darwin-x64
    ├── @luscii-healthtech/glassfrog-linux-arm64
    └── @luscii-healthtech/glassfrog-linux-x64
```

npm installs only the platform package whose `os`/`cpu` match the host; its `package.json` carries the OS/CPU map (GoReleaser `amd64` → npm `x64`; `arm64` → `arm64`; `darwin`/`linux` unchanged). The umbrella's launcher resolves whichever platform binary is present and execs it.

**Install-time flow**:
1. npm resolves the umbrella + the one matching platform package (its binary is bundled in the tarball — the primary, offline-capable path).
2. The umbrella's `postinstall.js` runs. If a bundled platform binary is resolvable → done, no network. If not (optional deps omitted, or the registry lacked the platform package) → **fallback**: map the host to a supported target, construct the release asset URLs, download the archive + checksums, verify sha256, extract with system `tar`, place the binary inside the umbrella package. On an **unsupported** platform (e.g. Windows) or checksum mismatch/download failure → print a clear message, place nothing runnable, exit non-zero.

**Runtime flow**: `glassfrog <args>` → the launcher resolves the binary path (bundled platform package, else the postinstall-placed path), `spawn`s it with inherited stdio and the forwarded argv, and exits with the child's exit code (re-raising signals) — preserving `004`'s exit-code convention through the npm channel. The launcher is also the **runtime backstop**: if no binary resolves (e.g. postinstall was skipped via `--ignore-scripts`), it emits the same clear refusal and a non-zero exit rather than a confusing crash.

**Release-time flow**: a dedicated `npm-publish` job in `release.yml`, gated on the existing `build` + `verify` jobs, reuses the **verified** `dist/` binaries (byte-parity — the bytes the self-containment matrix already gated are the bytes npm publishes), generates the umbrella + 4 platform package directories from in-repo sources into a gitignored output dir at the release version, and publishes platform-packages-first then the umbrella via npm OIDC trusted publishing (`--provenance`, no long-lived token).

---

## Architecture Decisions

### ADR-1: Umbrella package + four per-platform packages via `optionalDependencies` (bundled-binary primary path)

**Context**: The spec requires `npx` / `npm i -g` / local-dependency install of the right platform binary, an offline-capable primary path ("must not require network at install when the matching optional dependency is available"), and a verified download fallback. PROJECT.md / FEATURE-MODEL already names "platform-specific optional dependencies with a postinstall fallback".

**Options considered**:
1. **Umbrella + per-platform packages, optional deps, bundled binary** — npm's native `os`/`cpu` resolution installs only the matching package; the binary ships inside that package's tarball. Offline-capable; the canonical native-binary pattern (esbuild, swc, the ailign reference). More published artifacts (5 per release).
2. **Single package, postinstall-download-only** — one package; postinstall always downloads from the release. Fewer artifacts, but every install needs network and a working postinstall — violates the offline-capable non-behavior.
3. **Single package bundling all four binaries** — one tarball with every platform's binary. Simple, but ~4× download size for every user and ships three unusable binaries.

**Decision**: Option 1. Four platform packages (`glassfrog-<os>-<cpu>`) each declare `os`/`cpu` and bundle their binary; the umbrella lists them as `optionalDependencies` and exposes the `glassfrog` command. The postinstall fallback (ADR-3) covers the no-optional-dep case.

**Consequences**: Offline installs work from the bundled package; network is only the fallback. Five packages publish per release (ordering matters — ADR-7). The npm `os`/`cpu` vocabulary differs from GoReleaser's (`amd64`→`x64`) — the generator owns that mapping (ADR-2). The package scope/names are pinned in interface-spec (`@luscii-healthtech/glassfrog` plus the four platform packages); only the package.json metadata remains interface-level.

### ADR-2: Generate the package layout in CI from a single in-repo source — do not commit per-platform package directories

**Context**: The ailign reference commits N near-identical `npm/<platform>/` directories and only bumps versions in CI. The developer explicitly rejected committing generated scaffolding. But the launcher and fallback are real code that must be reviewed and tested — they can't be invented inline in a workflow.

**Options considered**:
1. **Commit static per-platform scaffolding, bump versions in CI** (ailign) — simplest workflow, but N committed near-duplicate package dirs that drift and clutter the repo. Rejected by the developer.
2. **One in-repo source + a CI generator** — commit only the real code (launcher, `postinstall.js`, the umbrella `package.json` template, a generator script) and synthesize the umbrella + 4 platform package directories at release time into a gitignored output dir. No committed duplication; the logic stays testable.
3. **GoReleaser Pro `npms` pipe** — native, one config block, but paid (Pro-only since v2.8) and would migrate the whole release tooling off OSS. Rejected (cost/policy).

**Decision**: Option 2. In-repo, the npm channel owns: `bin/glassfrog` (launcher), `postinstall.js` (fallback), an umbrella `package.json` template, and a generator (e.g. `npm/build.mjs`) that — given the release version and the verified `dist/` binaries — emits the umbrella and the four platform packages (each with its `os`/`cpu`, pinned `optionalDependencies`, bin mapping, and bundled binary) into a gitignored build dir. The generator owns the GoReleaser→npm arch mapping.

**Consequences**: The repo holds exactly one copy of each concern; no per-platform `package.json` files committed. The generated output dir is added to `.gitignore` (alongside `/dist/`). The generator and emitted shapes are unit-tested (ADR-3 testing). Exact metadata fields (description, homepage, repository, keywords) remain interface-level; the package names/scope are pinned in interface-spec (`@luscii-healthtech/…`).

### ADR-3: Conform to 027's deterministic-URL + sha256-verification acquisition pattern, with a download-base-URL seam for hermetic tests

**Context**: `027` ADR-1/ADR-2 set the acquisition-channel precedent and named `037` as a follower: construct release-asset URLs from `022`'s pinned `name_template` (`glassfrog_<ver>_<os>_<arch>.tar.gz`, `glassfrog_<ver>_checksums.txt`, `<ver>` = tag without `v`), verify sha256 against the checksums file, install atomically (verified-binary-or-nothing), and expose an overridable download base URL (default `https://github.com`) so tests run against a local fixture server with no network.

**Options considered**:
1. **Follow 027's pattern in the postinstall fallback** — construct URLs deterministically (no GitHub API, no JSON parsing), verify sha256, refuse on mismatch, base-URL seam for tests. Conforms to established precedent; hard-coupled to `022`'s template.
2. **Parse the GitHub Releases API in the fallback** — query assets by name. More robust to template changes, but adds API dependency/auth/rate-limit concerns and diverges from 027 for no behavioral gain.

**Decision**: Option 1 (silent conformance to 027). `postinstall.js` maps the host to a supported target, constructs the archive + checksums URLs from `022`'s template against a configurable base host (default `https://github.com`, under which it builds `/Luscii/cli-glassfrog/releases/download/<tag>/<name>` — the same seam shape as 027), downloads, verifies the archive's sha256 against its checksums entry, extracts via system `tar`, and places the binary atomically (temp → verify → move; nothing placed on failure). Verification gates the fallback (spec fork #2).

**Consequences**: Hard coupling to `022`'s `name_template` — a change 404s the fallback (same coupling 027 carries). Mitigation: pin the template shape in test fixtures as a shared contract; and the *publish-time* path (ADR-7) bundles `dist/` binaries directly rather than URL-construction, so only the install-time fallback depends on the URL shape. The base-URL seam lets `node --test` drive the fallback against a local server (fake archive + checksums) for happy-path / mismatch / unsupported — hermetic, mirroring 027's Go exec-test. `tar` is probed and a missing extractor fails clearly before any placement (027's tooling-probe discipline); `tar` ships on all supported targets (macOS/Linux only).

### ADR-4: The launcher is a zero-dependency CommonJS exec shim with full argv/stdio + exit-code/signal passthrough, and is the runtime backstop

**Context**: The spec requires transparent pass-through ("must not modify the binary's arguments, output, or exit code") so the npm channel honors `004`'s exit-code convention identically to a direct binary call. The binary lives in a *separate* optional-dependency package, so npm's own `bin` field can't point straight at it from the umbrella.

**Options considered**:
1. **JS launcher that `spawn`s the resolved binary** — resolve the platform binary path, `spawn` with `stdio: 'inherit'`, forward argv, exit with the child's code and re-raise its terminating signal. Honors exit codes; needs Node's `child_process` (built-in, no dependency).
2. **npm `bin` pointing at the binary** — impossible: the binary is in an optional-dep package, not the umbrella; the umbrella's `bin` must be a file it ships.

**Decision**: Option 1. `bin/glassfrog` (CommonJS, zero runtime npm dependencies — Node built-ins only) resolves the binary (the installed platform package via `require.resolve`, else the postinstall-placed path), spawns it with inherited stdio and `process.argv.slice(2)`, and propagates the exact exit code (and re-raises the signal for signal-terminated children). If no binary resolves, it emits the same clear refusal as the postinstall (naming the detected platform + supported set, suggesting reinstall without `--ignore-scripts`) and exits non-zero — so a skipped postinstall degrades to a clear error, never a confusing crash.

**Consequences**: Exit codes and signals match a direct invocation, so agents scripting against `004` see identical behavior through npm. The launcher is the second enforcement point for the unsupported/missing-binary refusal (ADR-5), closing the `--ignore-scripts` gap (Risk R3). Zero runtime deps keeps the install lean and audit-free.

### ADR-5: The unsupported-platform refusal fires in the postinstall, with the launcher as backstop

**Context**: Node/npm run on Windows and other arches, but `022` ships binaries only for macOS/Linux × amd64/arm64. The spec requires: unsupported platform → fail at install, place nothing runnable, exit non-zero (fork #1, mirroring 027).

**Options considered**:
1. **Refuse in postinstall (primary), launcher (backstop)** — `os`/`cpu` gating means no platform package installs on an unsupported host; postinstall detects "no supported target" and refuses; the launcher refuses too if reached. Two enforcement points cover the `--ignore-scripts` case.
2. **Rely on `os`/`cpu` + `engines` only** — let npm's own platform fields fail the install. Insufficient: the umbrella itself is platform-agnostic, so npm would install it and only fail confusingly at first run with no clear message.

**Decision**: Option 1. `postinstall.js` maps the host (`process.platform`/`process.arch`) to a supported target; if none, it prints a message naming the detected platform and the supported set, places nothing, and exits non-zero. The launcher repeats the refusal if it runs without a resolvable binary.

**Consequences**: A Windows/unsupported install never yields a silent or crashing `glassfrog`. The message mirrors 027's unsupported-platform wording for cross-channel consistency.

### ADR-6: Version coupling — npm version is the tag minus the leading `v`; optionalDependencies pinned to `=X.Y.Z`

**Context**: The spec fixes that the installed binary's version equals the installed package version, and the package version mirrors the release tag. Git tags are `vX.Y.Z` (optionally `-rc.N`); npm semver forbids a leading `v`. `023` embeds `vX.Y.Z` into `--version`.

**Options considered**:
1. **Strip the `v`; pin optional deps to the exact version** — npm version `X.Y.Z` (prerelease `X.Y.Z-rc.1`); the umbrella pins each `optionalDependency` to `=X.Y.Z`. A wrapper version resolves only its matching-version platform package.
2. **Use a range (`^X.Y.Z`) for optional deps** — lets npm pick a newer compatible platform package. Risks an umbrella/binary version skew; the spec wants exact coupling.

**Decision**: Option 1. The generator derives `X.Y.Z` from the tag (`${tag#v}`, the same transform 027/022 use for `<ver>`), stamps every package at that version, and pins `optionalDependencies` to `=X.Y.Z`. The bundled/downloaded binary reports `vX.Y.Z` via `023`.

**Consequences**: `@luscii-healthtech/glassfrog@X.Y.Z` always runs binary `vX.Y.Z` (the `v` is the build's, 023); `--version` parity is structural. Pre-release tags publish as npm prerelease versions (installed only when explicitly requested, matching npm's dist-tag semantics). No skew between umbrella and platform package.

### ADR-7: Publish from a dedicated `release.yml` job gated on build+verify, reusing the verified `dist/`, via npm OIDC trusted publishing

**Context**: Publishing the npm artifacts is `037`'s concern (`022` explicitly leaves npm to this channel). The repo's release ethos is "verified bytes = published bytes" and "only `GITHUB_TOKEN`, no external secrets" (022). The project's convention is to *extend* shared build files additively rather than fork parallel paths (021→022→023 all extend one `.goreleaser.yaml`).

**Options considered**:
1. **A dedicated `npm-publish` job in `release.yml`, `needs: [build, verify]`, reusing the `dist/` artifact, OIDC trusted publishing** — npm gets the exact bytes the self-containment matrix verified; no re-download, no GitHub-API parsing; no long-lived token. Couples into 022's workflow file (additive, demarcated).
2. **A separate `npm-release.yml` on `release: published` that downloads the published assets** — decoupled from 022's workflow, but re-downloads, must wait for/repeat asset resolution, and can publish before the verify gate unless it re-runs it.
3. **Long-lived `NPM_TOKEN` repo secret instead of OIDC** — simpler npm setup, but a stored credential against the repo's no-external-secrets ethos and rotation burden.

**Decision**: Option 1 with OIDC. Append an `npm-publish` job to `release.yml`, `needs: [build, verify]` (so a build or self-containment failure aborts npm too — no npm release without a release). It downloads the `dist/` CI artifact, runs the generator at the release version, then `npm publish --access public --provenance` the four platform packages **first**, then the umbrella (so the umbrella's pinned optional deps already exist). Auth is GitHub OIDC (`permissions: id-token: write`) — npm trusted publishing, no stored token.

**Consequences**: Byte-parity with the GitHub release; the install-time fallback's URL construction (ADR-3) is exercised only by end-users, never at publish. The job edits 022's `release.yml` (additive, clearly owned by 037 — mirroring `.goreleaser.yaml`'s per-spec ownership comments). Trusted publishing requires a **one-time maintainer setup** per package name on npmjs.com before the first publish (Risk R2) — a documented step, like 024's branch-protection script. Platform-first ordering is load-bearing: publishing the umbrella before its pinned platform packages would leave optional deps unresolvable for a window.

---

## Cross-cutting Concerns

**Error handling**: Install-time failures (unsupported platform, checksum mismatch, download/extract failure, missing `tar`) → a clear stderr message + non-zero exit, with nothing runnable placed (027's verified-or-nothing atomicity). Runtime: the launcher propagates the child's exit code and signal verbatim; an unresolvable binary → the same refusal + non-zero. No failure mode leaves a half-working command.

**Configuration**: The download base URL is overridable (default `https://github.com`) for hermetic tests. The version comes from the tag. The package name/scope is pinned at interface (`@luscii-healthtech/glassfrog`); only the package.json metadata remains an interface detail. Nothing else is configurable — the channel is install-and-run.

**Testing strategy**: The launcher and `postinstall.js` are plain CommonJS with `node --test` unit tests run in a CI job: launcher argv/exit/signal passthrough against a fake binary; postinstall platform detection, URL construction (pinned to 022's template fixture), sha256 verify + refuse-on-mismatch, unsupported-platform refusal, and `--ignore-scripts`/missing-binary launcher backstop — all driven against a local fixture server via the base-URL seam (no network), mirroring 027's hermetic exec-test. The generator is tested by asserting the emitted package.json shapes (os/cpu values, `=X.Y.Z`-pinned optional deps, bin mapping). A JS lint/format gate (the shellcheck analog 027 used) is an optional interface-level call. The npm channel is JS-isolated under its own directory; the Go suite is untouched.

**Observability**: The postinstall reports what it placed and from where (mirrors 027's report-on-success). Trusted-publishing provenance gives consumers a verifiable supply-chain attestation for free.

---

## Implementation Strategy

**Phase 1 — Package sources + hermetic tests** (depends on 022's landed `name_template` contract):
The committed npm channel under its own directory — the umbrella `package.json` template, `bin/glassfrog` launcher (ADR-4), `postinstall.js` fallback + unsupported refusal (ADR-3/ADR-5), and the generator (ADR-2) with the GoReleaser→npm arch mapping and `=X.Y.Z` pinning (ADR-6). `node --test` unit tests against a local fixture server via the base-URL seam. This phase produces no release wiring — it is fully testable offline and is the reviewable surface.

**Phase 2 — Release integration** (depends on Phase 1):
Append the `npm-publish` job to `release.yml` (ADR-7): `needs: [build, verify]`, download `dist/`, run the generator at the release version, OIDC `--provenance` publish platform-first then umbrella. Add the generated output dir to `.gitignore`. Provide the one-time npm trusted-publisher setup as a documented maintainer step (a `scripts/` helper in the spirit of `setup-branch-protection.sh`, or README instructions).

The phase split is real: Phase 1 is pure package logic (unit-testable, no secrets, no release event); Phase 2 is CI/publish wiring (touches 022's workflow, needs trusted-publisher setup). They can land in separate PRs — Phase 1 carries all the behavior the spec's driving scenarios assert; Phase 2 carries the publish path.

---

## Risks

- **R1 — Hard coupling to 022's `name_template`** (medium likelihood, high impact): a template change 404s the install-time fallback and breaks the generator's expectations. *Mitigation*: pin the template shape in test fixtures as a shared contract (027's approach); the publish path bundles `dist/` binaries directly, so only the fallback depends on the URL shape; a focused test asserts the constructed URL matches 022's current template.
- **R2 — Trusted-publishing bootstrap** (high likelihood on first release, medium impact): npm OIDC publishing requires a one-time per-package trusted-publisher configuration on npmjs.com; until set, publish fails, and each new platform-package name needs registering. *Mitigation*: a documented setup step run before the first release (like 024's branch-protection bootstrap); the four platform names are fixed, so this is a one-time cost.
- **R3 — Postinstall disabled (`--ignore-scripts`)** (medium likelihood, medium impact): with no matching optional dep AND scripts disabled, the umbrella installs with no working command and (without a backstop) no error. *Mitigation*: ADR-4's launcher backstop — it refuses clearly and non-zero at run time when no binary resolves, so the failure is legible regardless of postinstall.
- **R4 — Missing `tar` in the fallback** (low likelihood on supported targets, medium impact): the fallback shells out to system `tar`. *Mitigation*: probe and fail clearly before any placement (027's tooling probe); `tar` ships on macOS/Linux, the only supported targets.
- **R5 — Second language/toolchain in a Go repo** (low likelihood, low impact): JS adds a test/lint story. *Mitigation*: zero runtime dependencies, Node's built-in test runner only, isolated under one directory; no impact on the Go build or CONSTITUTION XII (which governs the distributed Go binary's runtime, not the host-side packager — the 021/022 build-host-vs-artifact distinction).

---

## What This Plan Does Not Cover

- **Protocol/structural detail (→ `/score:interface`)**: the exact npm package name and scope, the four platform package names, the umbrella/platform `package.json` field set (description, homepage, repository, keywords, `engines`), the launcher's behavior on edge argv, and the precise text of refusal/report messages. The plan fixes the topology, the version mapping, the verification contract, and the publish path; interface pins the names and shapes.
- **Executable scenarios (→ `/score:scenarios`)**: the spec's driving scenarios become `.feature` files.
- **Task decomposition (→ `/score:tasks`)**: the two phases become PR-sized units.
- **One-time npm account/org setup**: the npmjs.com trusted-publisher registration (org membership, 2FA policy) is a maintainer ops task, surfaced as R2, not designed here.
- **Behavioral gaps**: none — the spec is sharp; naming/scope and the publish path are pinned in this PR's interface-spec and ADR-7, leaving only the package.json metadata as an interface detail.
