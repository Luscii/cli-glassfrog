# Tasks: Homebrew Tap

**Feature**: 036-homebrew-tap
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/runtime-dependent-distribution/homebrew-tap.feature

---

> **Cross-spec gate**: every task here builds on **022 Automated Release Pipeline** — its `.goreleaser.yaml` (`archives`/`checksum`/`release`) and `release.yml` (`build → verify → publish` spine), plus 021's `builds` and the `internal/build` config-guard. 022 has **landed** (release pipeline on main), so the config and CI tasks are ready now. Two **new external prerequisites** this feature introduces are ops, not code: a dedicated tap repository and a cross-repo write token (Phase 1).

## Dependency Graph

Phase 1: Provisioning Prerequisites (2 tasks, no dependencies) [Shared]
Phase 2: GoReleaser Formula Configuration (1 task, no dependencies — parallel with Phase 1) [Shared]
Phase 3: Release-Pipeline Wiring (1 task, depends on Phases 1, 2) [Shared]

4 tasks total | Phases 1 and 2 parallelizable | Builder: pipeline (single spec)

## Branching Guidance

**Pipeline mode**: `spec/036-homebrew-tap/base` → `spec/036-homebrew-tap/task-1`, `task-2`, `task-3`, `task-4`.

The `base` branch is cut from a main that contains 021/022/023 (the complete `.goreleaser.yaml` and the release workflow). T001 (ops provisioning) and T002 (LICENSE decision) are not code branches in the usual sense — T001 is repository/secret administration; T002 carries a decision the developer owns. Other distribution specs (027 landed, 037 npm) build on their own base branches; this feature only hard-depends on 022.

---

## Phase 1: Provisioning Prerequisites [Shared]

- [ ] **T001** [Shared] [P] Provision the dedicated tap repository and its cross-repo write token
  - **Scope**: Create the empty public repository `Luscii/homebrew-cli-glassfrog` (default branch `main`, left unprotected so the publisher can push directly). Mint a **least-privilege** credential scoped to *only* that repository with `contents: write` — baseline a fine-grained PAT; a GitHub App installation token is the no-expiry upgrade — and store it as a repository secret named `HOMEBREW_TAP_TOKEN` in `cli-glassfrog`. No application code; no change to this repo's `main`.
  - **Acceptance criteria**:
    - `Luscii/homebrew-cli-glassfrog` exists, is reachable, and its `main` is writable by the minted token.
    - The token's scope is limited to the tap repository with `contents: write` and nothing broader.
    - The secret `HOMEBREW_TAP_TOKEN` is present in `cli-glassfrog`'s Actions secrets.
    - Nothing is committed to `cli-glassfrog`'s protected `main` as part of this step.
  - **Dependencies**: None
  - **Plan reference**: Phase 1: Provision the tap repository and credential; ADR-2, ADR-4
  - **Interface references**: interface-spec.md: `brews` section (`repository.*`, `repository.token`); `tap` job (`env: HOMEBREW_TAP_TOKEN`)
  - **Risk**: ⚠️ First credential beyond `GITHUB_TOKEN` in the pipeline — keep it scoped to the tap repo only; track expiry (prefer an App token if rotation is unwanted). A leak's blast radius is one repo, never this repo's `main`.

- [ ] **T002** [Shared] [P] Decide and add the project LICENSE, then wire the formula's `license` metadata
  - **Scope**: The repository has no `LICENSE` file, but the formula needs a `license` field (the interface's open `[NEEDS INPUT]`). Resolve the licensing decision (developer-owned), add the chosen `LICENSE` at the repo root so GoReleaser auto-detects it for `brews.license`, or — if the tap stays private/non-strict — explicitly decide the formula carries no SPDX license and document that. This task is the home for the open decision; it unblocks a clean `brew audit`.
  - **Acceptance criteria**:
    - The licensing decision is recorded (a `LICENSE` file is present, or a documented decision that the formula omits a license).
    - If a `LICENSE` is added, GoReleaser fills `brews.license` with the matching SPDX id and `brew audit` no longer fails on a missing license.
  - **Dependencies**: None
  - **Plan reference**: Phase 1 (prerequisite); interface `[NEEDS INPUT]` (license)
  - **Interface references**: interface-spec.md: `brews` section (`license`); Consistency Notes (license gap)
  - **Risk**: ⚠️ Open decision the Builder cannot make alone — surface to the developer. Until resolved, `brew install` from a fresh tap may warn/fail audit; this gates the *first real* stable release relying on the tap, not the config/CI tasks.

## Phase 2: GoReleaser Formula Configuration [Shared]

- [x] **T003** [Shared] Add the `brews` block to `.goreleaser.yaml` with the 022-config refinements, a config-guard assertion, and an offline render-and-inspect test — 2 scenarios (config-guard + checksum-match) un-@wip'd; drift-adapted `archives.mtime`→`archives.builds_info.mtime` and added `brews.url_template` (release.disable breaks the default URL); flagged the `brews` deprecation (conflicts with ADR-1) — both in LEARNINGS.md
  - **Scope**: Add a `brews` entry to `.goreleaser.yaml` (do **not** touch `builds`/`ldflags`): `name: glassfrog`, `ids: [glassfrog]`, `repository: { owner: Luscii, name: homebrew-cli-glassfrog, branch: main, token: "{{ .Env.HOMEBREW_TAP_TOKEN }}" }`, `directory: Formula`, `homepage`, `description`, `license` (from T002), `install` (`bin.install "glassfrog"`), `test` (`system "#{bin}/glassfrog", "version"`), `skip_upload: false`. Apply the two refinements to 022's sections required for the separate-job publish: change `release:` to `disable: true` (strict form of `keep-existing`, so the brew-publishing `goreleaser release` never touches the GitHub release) and pin `archives.mtime` (to the commit date) so the `tap` job's rebuild is byte-reproducible and its checksums equal the published archives'. Extend the `internal/build` config-guard (`CheckConfigGuard`) to assert a `brews` entry exists targeting `owner: Luscii` / `name: homebrew-cli-glassfrog` / `name: glassfrog`, that `release.disable` is true, and that `archives.mtime` is pinned — same change-detector rigor as the siblings. Add an offline render-and-inspect test: `goreleaser release --snapshot --clean --skip=publish` renders `dist/.../glassfrog.rb`; assert the formula's shape (class, per-platform `on_macos`/`on_linux` × `on_arm`/`on_intel` url+sha256 blocks, `bin.install`, `test`) and that each rendered sha256 equals the snapshot's `checksums.txt` entry.
  - **Acceptance criteria**:
    - `goreleaser release --snapshot --clean --skip=publish` renders a `Formula/glassfrog.rb` with all four platform `url`+`sha256` blocks, the install and test stanzas, and a populated `version` field.
    - Each `sha256` in the rendered formula equals the matching line in the snapshot `checksums.txt`.
    - `builds`/`ldflags` are byte-unchanged; `release.disable: true` and a pinned `archives.mtime` are in effect.
    - The config-guard test fails loudly if the `brews` block is missing/retargeted, if `release.disable` is not true, or if `archives.mtime` is unpinned.
    - Implementation and its tests ship in the same PR (CONSTITUTION VII).
  - **Dependencies**: None (parallel with Phase 1; the `brew audit`-clean property additionally needs T002's license)
  - **Plan reference**: Phase 2: Add the `brews` block + config-guard; ADR-1, ADR-3; Cross-cutting Concerns (config-drift guard, reproducibility)
  - **Scenario references**: homebrew-tap.feature: "Config-guard fails when the brews block is blanked or retargeted"; "The published formula's checksums match the release's checksums file"
  - **Interface references**: interface-spec.md: `.goreleaser.yaml` — `brews` section; Refinements to 022's sections; Config-guard extension; Generated `Formula/glassfrog.rb` structural contract
  - **Risk**: ⚠️ Reproducibility — without a pinned `archives.mtime` the `tap` job's rebuilt archives differ from the published ones and every `brew install` fails the integrity check. If reproducibility proves brittle, the documented fallback is to derive the formula's sha256 from the published `checksums.txt` instead of a rebuild.

## Phase 3: Release-Pipeline Wiring [Shared]

- [ ] **T004** [Shared] Add the post-publish `tap` job to `release.yml` (brew-publisher-only, stable-only)
  - **Scope**: Add a `tap` job to `.github/workflows/release.yml`: `needs: [publish]`, `if: ${{ !github.event.release.prerelease }}` (the **authoritative** stable-only gate — do not rely on GoReleaser's `skip_upload: auto`), `runs-on: ubuntu-latest`, `env: { HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }} }`. Steps: `actions/checkout@v4` (`fetch-depth: 0`, ref = the release tag), `actions/setup-go@v5` (`go-version-file: go.mod`), `goreleaser/goreleaser-action@v6` (`version: "~> v2"`), then a GoReleaser invocation that runs the **brew publisher only** — it pushes `Formula/glassfrog.rb` to `homebrew-cli-glassfrog` and must **not** create or modify the GitHub release (that stays with the `publish` job's `gh release upload`). The leg's success/failure is independent of the already-completed `publish` — a tap failure must not alter the published release. **Ship the workflow change with its test (CONSTITUTION VII)**: extend the `internal/build` workflow structural guard (the `CheckReleaseWorkflow`/`CheckVerifyGate` family established by 022, the landed Automated Release Pipeline guard) with a `tap`-job assertion — the guard parses `release.yml` and fails loudly unless the `tap` job exists with `needs: [publish]`, the `if: ${{ !github.event.release.prerelease }}` gate, the `HOMEBREW_TAP_TOKEN` env wiring, and a brew-publisher-only (no GitHub-release-publishing) invocation. This is the CI-testable proxy for the workflow behavior, exactly as 022 guards its jobs; it covers the structural contract that the held-`@wip` end-to-end scenarios cannot exercise without a live tap.
  - **Acceptance criteria**:
    - Publishing a **stable** release runs the `tap` job after `publish`; it pushes an updated `Formula/glassfrog.rb` to `Luscii/homebrew-cli-glassfrog` whose `url`s point at that release's assets and whose `sha256`s match them.
    - Publishing a **pre-release** skips the `tap` job (the `if`); the tap repo's formula is left untouched.
    - The `tap` job neither creates nor modifies the GitHub release; the assets/notes/status from `publish` and #30 are unchanged.
    - A missing/unauthorized/expired `HOMEBREW_TAP_TOKEN` fails the `tap` job non-zero (loud red run), and the published release is unaffected.
    - Re-running for an already-published stable release re-pushes the formula idempotently (reproducible build → identical formula).
    - The `internal/build` workflow guard asserts the `tap` job's contract (`needs: [publish]`, the pre-release `if`-gate, the token env, brew-publisher-only) and **fails loudly** if any is missing or drifts — including a case that fails as loudly on a *removed* assertion as an added one (change-detector rigor, per the sibling guards).
    - Implementation and its guard test ship in the same PR (CONSTITUTION VII).
  - **Dependencies**: T001 (tap repo + secret), T003 (`brews` config). Clean `brew audit` on the pushed formula additionally needs T002.
  - **Plan reference**: Phase 3: Add the `tap` job; ADR-3, ADR-4; Cross-cutting Concerns (stable-only gate, config-drift guard)
  - **Scenario references**: homebrew-tap.feature: "Fresh install on macOS"; "Fresh install on Linux"; "Upgrade moves to the latest stable"; "Upgrading when already current is a no-op"; "A pre-release does not move the tap"; "A pre-release leaves the tap repository untouched"; "Checksum mismatch refuses the install"; "A missing release asset fails the install clearly"; "A mismatched checksum never lets a binary reach PATH"; "The installed binary matches the release the formula points at"
  - **Interface references**: interface-spec.md: `release.yml` — `tap` job structure; Interactions (publish flow, pre-release flow); Error Communication
  - **Risk**: ⚠️ Stable-only gating mismatch — the gate must be the job-level `if`, not `skip_upload: auto` (which keys off a semver `-rc` suffix, divergent from the GitHub pre-release flag); the workflow guard pins this. End-to-end `brew install` from the live tap remains a manual post-first-release validation; the structural guard covers everything CI can assert without a live tap.
