# Tasks: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/npm-wrapper-package.feature

---

## Dependency Graph

Phase 1: Package sources + hermetic tests (4 tasks, no phase dependencies) [Shared]
Phase 2: Release integration (2 tasks, depends on Phase 1) [Shared]

6 tasks total | within Phase 1: T001 ∥ T002, then T003, then T004; within Phase 2: T005 ∥ T006 | Builder: single Builder (pipeline mode)

---

## Branching Guidance

**Pipeline mode**: `spec/037-npm-wrapper-package/base` → `spec/037-npm-wrapper-package/task-1`, `spec/037-npm-wrapper-package/task-2`, …

**Role-based mode**: `spec/037-npm-wrapper-package/base` as the integration point; task branches `spec/037-npm-wrapper-package/task-N`. The two phases are sequential (Phase 2 needs the generated packages from Phase 1); inside Phase 1, the launcher (T001) and postinstall (T002) are independent files that can be built in parallel before the generator (T003) bundles them and the test suite (T004) covers them.

---

## Phase 1: Package sources + hermetic tests [Shared]

- [x] **T001** [Shared] [P] Write the umbrella launcher (`bin/glassfrog`) **and its unit tests** — zero-dependency CommonJS exec shim — launcher + shared `lib/platform.js` + 4 launcher scenarios (argv/exit/signal/backstop) + 8 lib unit tests; lib is shared infra reused by T002/T003
  - **Scope**: One new file (e.g. `npm/bin/glassfrog`), CommonJS, no runtime npm dependencies (Node built-ins only), **together with its `node --test` unit tests in the same PR**. Resolves the platform binary — the installed platform package via `require.resolve('@luscii-healthtech/glassfrog-<os>-<cpu>/bin/glassfrog')`, else the postinstall-placed path — and `spawn`s it with `stdio: 'inherit'` and `process.argv.slice(2)`. Propagates the child's exit code and re-raises its terminating signal. Runtime backstop: when no binary resolves, prints the detected platform + supported set and an advice to reinstall without `--ignore-scripts`, then exits non-zero.
  - **Acceptance criteria**:
    - Running the launcher forwards all arguments and stdin/stdout/stderr to the binary unchanged.
    - The launcher exits with the binary's exact exit code; a signal-terminated child re-raises the signal.
    - With no resolvable binary, the launcher writes a clear refusal to stderr (naming detected platform + the four supported targets) and exits non-zero — never an uncaught crash.
    - Zero runtime npm dependencies; uses only Node built-ins.
    - **Ships `node --test` unit tests** (against a stub binary) asserting argv/exit-code/signal passthrough and the no-binary backstop — implementation and its tests land together (CONSTITUTION VII).
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Package sources; ADR-4 (zero-dependency launcher + passthrough + backstop)
  - **Scenario references**: npm-wrapper-package.feature: "Exit code and arguments pass through unchanged", "Launcher refuses clearly when no binary is installed", "npx resolves and runs the matching platform binary" (exec half)
  - **Interface references**: interface-spec.md: Surface (In-repo sources — Launcher); Interactions (Runtime flow); Error Communication (Runtime/launcher table)

- [x] **T002** [Shared] [P] Write the postinstall fallback (`postinstall.js`) **and its fixture-server tests** — platform detection, verified download, unsupported refusal — postinstall.js (no-op-when-bundled, verified fallback download/extract/atomic-place, refusals) + 5 fixture-server tests (happy/bundled-no-network/mismatch/unsupported/missing-tar); fixtures pin 022's asset names
  - **Scope**: One new file (e.g. `npm/postinstall.js`), CommonJS, zero runtime npm dependencies, **together with its `node --test` tests (driven against a local fixture server via the `GLASSFROG_DOWNLOAD_BASE_URL` seam) in the same PR**. If a bundled platform binary resolves → no-op success (no network). Else fallback: map `process.platform`/`process.arch` to a supported target; on no supported target → refuse (message naming detected platform + supported set), place nothing, exit non-zero. On a supported target: construct the release asset URLs from 022's `name_template` against `GLASSFROG_DOWNLOAD_BASE_URL` (default `https://github.com`), download archive + checksums, verify the archive's sha256 against its checksums entry (refuse + non-zero on mismatch), probe for `tar` (clear failure if absent), extract, and place the binary atomically (temp → verify → move; nothing placed on any failure). Each error names a cause and a next step (CONSTITUTION II — see interface-spec.md § Error Communication).
  - **Acceptance criteria**:
    - With the matching platform package present, the postinstall is a no-op and makes no network call.
    - On an unsupported platform, it places nothing, names the detected platform + the four supported targets and a next step, and exits non-zero (failing the install).
    - On the fallback path, it downloads the `glassfrog_<ver>_<os>_<arch>.tar.gz` archive and `glassfrog_<ver>_checksums.txt`, and places the binary only after the sha256 matches.
    - A checksum mismatch leaves no runnable binary and exits non-zero naming the integrity failure and the retry next step.
    - A missing `tar` extractor fails before any placement, naming the missing tool and the next step.
    - `GLASSFROG_DOWNLOAD_BASE_URL` redirects all downloads (the test seam); zero runtime npm dependencies.
    - **Ships `node --test` tests** against a local fixture server (no network): happy fallback places the stub binary; mismatch / unsupported / missing-`tar` each fail with nothing placed — implementation and its tests land together (CONSTITUTION VII). Fixtures encode 022's exact asset names so template drift breaks a test.
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Package sources; ADR-3 (027-conformant deterministic-URL + sha256 verification, base-URL seam), ADR-5 (unsupported-platform refusal)
  - **Scenario references**: npm-wrapper-package.feature: "Fallback download verifies before placing the binary", "Checksum mismatch aborts the fallback install", "Unsupported platform is refused at install", "Missing extractor fails before any binary is placed", "Offline install uses the bundled platform package" (no-op half), "A corrupted fallback download never becomes runnable" (@validation)
  - **Interface references**: interface-spec.md: Surface (In-repo sources — postinstall; Configuration schema); Interactions (Install-time flow); Error Communication (Install-time table)
  - **Risk**: ⚠️ Hard coupling to 022's archive/checksum name template — a template change 404s the fallback (pin via this task's fixtures; see risk.md H-2 / RC-2)

- [x] **T003** [Shared] Write the package generator + umbrella `package.json` template **and its package-shape tests** — npm/package.json template (umbrella + dev manifest) + build.mjs generator (os/cpu map, =version pinning, extract-from-022-archive, copies sources) + 5 shape/integration tests
  - **Scope**: The umbrella `package.json` template and a generator (e.g. `npm/build.mjs`) that, given the release version and the verified `dist/` binaries, emits the umbrella (`@luscii-healthtech/glassfrog`) and the four platform packages (`@luscii-healthtech/glassfrog-<os>-<cpu>`) into a gitignored output dir — **together with its `node --test` package-shape tests in the same PR**. Applies the GoReleaser→npm arch map (`amd64`→`x64`, `arm64`→`arm64`; `darwin`/`linux` unchanged), stamps every package at the version (tag minus leading `v`), pins the umbrella's `optionalDependencies` to `=<version>`, sets `bin` only on the umbrella (the launcher) and `os`/`cpu` on each platform package, copies the launcher + postinstall into the umbrella, and bundles each platform binary.
  - **Acceptance criteria**:
    - The generator emits one umbrella + four platform package directories from a version + a `dist/` location.
    - Umbrella `package.json`: `name` `@luscii-healthtech/glassfrog`, `bin` `{glassfrog: bin/glassfrog}`, `scripts.postinstall`, `optionalDependencies` listing all four platform packages pinned `=<version>`, `publishConfig.access: public`.
    - Each platform `package.json`: scoped name with `<os>-<cpu>`, the matching `os`/`cpu` values, the bundled binary in `files`, **no** `bin`.
    - Version is the tag without the leading `v` (`v1.4.0-rc.1`→`1.4.0-rc.1`); all five packages share it.
    - **Ships `node --test` tests** asserting the emitted umbrella + four platform `package.json` shapes — the os/cpu map, `=<version>` pinning, and bin placement — from a fixture version + `dist/` layout; implementation and its tests land together (CONSTITUTION VII).
  - **Dependencies**: T001, T002
  - **Plan reference**: Phase 1 — Package sources; ADR-1 (optionalDependencies topology), ADR-2 (CI-generated scaffolding), ADR-6 (version coupling)
  - **Scenario references**: npm-wrapper-package.feature: "Pinned global install places the matching binary", "npx resolves and runs the matching platform binary" (optional-dep resolution half), "Each supported platform resolves exactly its own binary" (@validation), "The placed binary version matches the package and the release tag" (@validation)
  - **Interface references**: interface-spec.md: Surface (Published packages, umbrella/platform package.json contracts, Generated output); Interactions (Generator invocation, Version normalisation)

- [ ] **T004** [Shared] Wire the `node --test` suite into CI + add cross-unit integration coverage; clear `@wip`
  - **Scope**: Wire the per-unit `node --test` suites shipped by T001–T003 into CI as a Node test job (running under the existing gate surface, alongside 024's lint/test), and add the cross-unit/end-to-end coverage that no single unit's tests own: a full install-then-launch pass against the fixture server (postinstall places via fallback → launcher execs the placed binary). This task adds the integration layer and CI wiring; it introduces no new untested package source — each source unit already ships its own unit tests (T001–T003). On completion, remove the `@wip` tags from the now-covered scenarios; the three `@validation` scenarios stay `@wip` for independent verification.
  - **Acceptance criteria**:
    - The combined `node --test` suite runs in CI (a Node test job) on PRs and main, so PR Validation (024) / Main-Branch Verification (029) pick it up.
    - An end-to-end test exercises fallback-install → launcher-exec against the fixture server with no network.
    - The 022 asset-name template fixture (from T002) is asserted as the shared contract so drift breaks the suite.
    - Non-validation scenarios lose `@wip`; the three `@validation @wip` scenarios remain held out.
  - **Dependencies**: T001, T002, T003
  - **Plan reference**: Phase 1 — Package sources (Testing strategy); ADR-3 (download-base-URL seam)
  - **Scenario references**: npm-wrapper-package.feature: all `@wip` scenarios (made runnable); the three `@validation @wip` scenarios held out for independent verification
  - **Interface references**: interface-spec.md: Interactions (Test invocation)

## Phase 2: Release integration [Shared]

- [ ] **T005** [Shared] [P] Add the `npm-publish` job to `release.yml` and gitignore the generated output dir
  - **Scope**: Append an `npm-publish` job to `.github/workflows/release.yml`, `needs: [build, verify]`, that downloads the verified `dist/` artifact, runs the generator (T003) at the release version, and publishes — **platform packages first, then the umbrella** — with `npm publish --access public --provenance` authenticated by GitHub OIDC (`permissions: id-token: write`; no stored `NPM_TOKEN`). Add the generated output dir (e.g. `dist/npm/`) to `.gitignore`.
  - **Acceptance criteria**:
    - The job runs only on `release: published` and only after `build` and `verify` succeed (no npm release without a verified release).
    - It reuses the verified `dist/` binaries (byte-parity), not a re-download.
    - It publishes the four platform packages before the umbrella, all at the release version, via `--provenance` over OIDC with no long-lived token.
    - The generated output directory is gitignored; no per-platform package directories are committed.
  - **Dependencies**: T003 (and T004 green)
  - **Plan reference**: Phase 2 — Release integration; ADR-7 (release.yml publish job, OIDC trusted publishing)
  - **Scenario references**: npm-wrapper-package.feature: establishes the "published release with the umbrella and its four platform packages on the registry" precondition exercised by the install/run scenarios
  - **Interface references**: interface-spec.md: Interactions (Publish flow); Consistency Notes (OIDC trusted publishing)
  - **Risk**: ⚠️ Edits 022's `release.yml` (additive, demarcated as 037's job) — confirm OIDC `id-token: write` is scoped to the publish job and the build/verify gates still bound it

- [ ] **T006** [Shared] [P] Document the npm install path and the one-time trusted-publisher setup
  - **Scope**: Add an npm Installation section to the README (`npx @luscii-healthtech/glassfrog`, `npm i -g @luscii-healthtech/glassfrog`, pinned-version form, supported platforms) and document the one-time npmjs.com trusted-publisher registration for the five package names as a release prerequisite (a `scripts/` helper or README instructions, in the spirit of `setup-branch-protection.sh`).
  - **Acceptance criteria**:
    - The README shows the `npx` / `npm i -g` / pinned forms with the `@luscii-healthtech/glassfrog` name and the supported platforms (macOS/Linux × x64/arm64).
    - The one-time trusted-publisher setup (per package name, before first publish) is documented as a release prerequisite.
    - The npm channel is presented alongside the sibling channels (install script 027, Homebrew 036) as installing the same binary.
  - **Dependencies**: T003 (package names settled)
  - **Plan reference**: Phase 2 — Release integration; ADR-7 (one-time trusted-publisher setup)
  - **Interface references**: interface-spec.md: Surface (Consumer invocation); Consistency Notes (sibling acquisition channels, OIDC trusted publishing)
