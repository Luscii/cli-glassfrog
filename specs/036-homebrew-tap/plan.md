# Plan: Homebrew Tap

**Feature**: 036-homebrew-tap
**Role**: Shaper
**Inputs**: spec.md (036), PROJECT.md, FEATURE-MODEL.md (Self-Contained Distribution cluster), `.goreleaser.yaml`, `.github/workflows/release.yml`, DECISIONS.md (021/022/023/027 distribution precedent), DEPRECATION.md (no distribution entries)

---

## System Architecture

The Homebrew channel adds a **third leg** to the existing release pipeline without disturbing its `build → verify → publish` spine. Two things change, both in *this* repository:

1. **`.goreleaser.yaml` gains a `brews` block.** GoReleaser becomes the single author of the formula's content — its version, the per-OS/per-arch archive URLs, the sha256 sums (the same ones 022 already computes), and the install/test stanzas. This conforms to the standing decision (DECISIONS 173/179/185) that the brew section lives in the same config file as `builds`/`archives`/`checksum`/`ldflags`.

2. **`release.yml` gains a `tap` job.** It hangs off `needs: [publish]`, runs only when the triggering release is **not** a pre-release, and invokes GoReleaser's Homebrew publisher — and *only* that publisher — to write the rendered formula into a **separate tap repository**.

```
  release: published
        │
   ┌────┴─────────────────────────────────────────┐
   │ build (goreleaser release --skip=publish)     │  ← unchanged (021/022/023)
   │   └─ verify (4-leg self-containment matrix)   │  ← unchanged (022)
   │        └─ publish (gh release upload)         │  ← unchanged (022): attaches archives+checksums
   │             └─ tap  (NEW)                     │
   │                 if: !release.prerelease       │  ← stable-only gate (job level)
   │                 goreleaser → brew publisher   │  ← pushes Formula/glassfrog.rb …
   └────────────────────────────────────────────────┘
                                  │
                                  ▼
              Luscii/homebrew-cli-glassfrog   (separate repo = the tap)
                  Formula/glassfrog.rb   ← users `brew tap` / `brew install` this
```

The flow that reaches a user: a stable release publishes → the `tap` job pushes an updated `Formula/glassfrog.rb` into `homebrew-cli-glassfrog` → a user who has tapped that repo gets the new version on `brew upgrade`. New machines run `brew install luscii/cli-glassfrog/glassfrog` (one-shot, auto-taps `homebrew-cli-glassfrog`) or `brew tap luscii/cli-glassfrog && brew install glassfrog`.

The crucial structural property: **nothing in this flow writes to *this* repository's `main`.** The formula is published to a different repository, so the protected-branch friction never arises. The cost is one new piece of state outside this repo (the tap repository) and one new credential (a cross-repo write token).

---

## Architecture Decisions

### ADR-1: Distribute as a GoReleaser Homebrew formula (not a cask)

**Context**: The spec requires `brew install`/`brew upgrade` on **macOS and Linux**, installing the pre-built release binary with no source compilation. Homebrew Cask is a macOS-only mechanism (Linuxbrew has no cask support); a formula works on both platforms and, for a Go binary, installs the pre-built archive without compiling.

**Options considered**:
1. **Cask (macOS-only)** — installs the pre-built binary, but cannot serve Linux brew; would force Linux users onto other channels. Narrower than the spec's reach.
2. **Formula** — GoReleaser's `brews` block generates a formula that downloads the platform archive and `bin.install`s the binary (no `go build` on the host). Works on macOS and Linux. Matches the spec exactly.

**Decision**: Option 2 — a GoReleaser-authored formula. The `brews` block keys off the existing `archives`, so the formula's per-platform `url`/`sha256` are derived from the same artifacts 022 publishes. The install stanza places the `glassfrog` binary; a `test` stanza runs a no-network version command.

**Consequences**: One formula serves all four release targets. The "formula compiles from source" worry does not apply to this binary-install style. Homebrew's own `brew audit`/`brew style` rules now apply to the rendered formula (license, test block, desc) — the formula must pass them or `brew install` from a fresh tap warns/fails. The dropped cask path is simply not built.

### ADR-2: Publish to a dedicated tap repository, not in-repo

**Context**: The formula has to live somewhere users can `brew tap`. This repository's `main` is branch-protected (024 requires `ci-success`), and the release pipeline deliberately keeps git/publishing out of GoReleaser (it attaches assets with `gh release upload`, never `goreleaser publish` — DECISIONS 178/179). An in-repo tap would force CI to mutate protected `main` on every release (a PR-merge dance or a privileged bypass token). The Feature Model's original "no separate tap repo" intent was reconsidered against this friction and updated.

**Options considered**:
1. **In-repo tap** — `Formula/` in this repo; CI opens a PR or pushes to protected `main` each release. Either a manual merge per release, or a token that bypasses branch protection. Friction on every release, on the protected branch.
2. **Dedicated tap repo** (`Luscii/homebrew-cli-glassfrog`) — the conventional GoReleaser setup; the brew publisher pushes the formula straight to that repo. No interaction with this repo's `main`. Also unlocks the one-shot `brew install owner/repo/name` UX. Cost: a second repository to own.

**Decision**: Option 2. A dedicated `homebrew-cli-glassfrog` repository is the tap. GoReleaser's `brews.repository` targets it (`owner: Luscii`, `name: homebrew-cli-glassfrog`, `directory: Formula`). The release process pushes `Formula/glassfrog.rb` directly to that (unprotected) repo — no PR needed there.

**Consequences**: Removes the protected-`main` friction entirely and gives the better install UX. This conforms to DECISIONS 179's "brew section in the same `.goreleaser.yaml`" (the *config* still lives here) while diverging from its "entirely within the repo (only `GITHUB_TOKEN`)" property — see ADR-4. The tap repo must exist and be writable before the `tap` job can run (Phase 1). The tap repo holds only the formula; its history is machine-generated commits.

### ADR-3: Wire publishing as a post-publish `tap` job that runs only the brew publisher

**Context**: The existing pipeline runs `goreleaser release --skip=publish` (build/package only) and then attaches assets with `gh release upload`. To publish the formula, GoReleaser's Homebrew publisher must run — but it must **not** also publish/modify the GitHub release (that boundary belongs to `gh release upload` and Release Drafting #30). The formula must reference assets that already exist and have passed the self-containment gate.

**Options considered**:
1. **Enable GoReleaser publish wholesale** — drop `--skip=publish`. GoReleaser would then also create/modify the GitHub release, colliding with the `gh release upload` + #30 boundary and the verify-before-publish atomicity. Rejected.
2. **Dedicated `tap` job after publish** — `needs: [publish]`, runs GoReleaser with the GitHub-release publisher disabled and only the brew publisher active, against the tap repo. Keeps the spine untouched; the formula references the just-published, just-verified assets.
3. **GoReleaser opens a PR on the tap repo** — unnecessary, since the tap repo is unprotected; a direct push is simpler.

**Decision**: Option 2. A `tap` job, `needs: [publish]`, `if: ${{ !github.event.release.prerelease }}`, runs GoReleaser's brew publisher only. The exact skip-flags / config toggles (e.g. disabling the `release` section for this invocation, or a scoped `--skip`) are interface/tasks-level; the architectural commitment is "brew publisher yes, GitHub-release publisher no, gated on a verified non-prerelease publish."

**Consequences**: The release spine (build/verify/publish) is unchanged and still owns asset attachment. The formula is only published for releases that built, verified, and published successfully — partial releases attach nothing and tap nothing. The `tap` job re-running on a re-published release re-renders the formula deterministically (idempotent).

### ADR-4: Authenticate the tap push with a cross-repo, least-privilege token

**Context**: The default `GITHUB_TOKEN` is scoped to the workflow's own repository and **cannot push to another repository**. Publishing to `homebrew-cli-glassfrog` needs a credential with `contents: write` on that repo. This is the **first** credential in the release pipeline beyond `GITHUB_TOKEN` (DECISIONS 179 recorded the pipeline as "entirely within the repo (only `GITHUB_TOKEN`)").

**Options considered**:
1. **Classic PAT** — broad account scope (`repo`), works everywhere but over-privileged; a leak exposes every repo the owner can write.
2. **Fine-grained PAT scoped to only the tap repo** — `contents: write` on `homebrew-cli-glassfrog` and nothing else. Least privilege; but PATs expire and are user-owned.
3. **GitHub App installation token scoped to the tap repo** — no expiry, org-owned, least privilege; more setup.

**Decision**: Option 2 as the baseline — a fine-grained PAT scoped to `homebrew-cli-glassfrog` with `contents: write`, stored as a repository secret and injected **only** into the `tap` job's environment. Option 3 (App token) is the preferred upgrade for an org that wants no user-owned PAT or no expiry; swapping is just changing how the secret is minted.

**Consequences**: Blast radius of a leak is one repository (the tap), not this repo's `main` and not the whole org. The credential touches only the `tap` job. Negative: a fine-grained PAT expires — a lapsed token fails the `tap` job loudly on a future release (red run, not a silent-wrong publish); rotation must be tracked, which is the main argument for the App-token upgrade.

---

## Cross-cutting Concerns

**Stable-only gate (correctness-critical)**: The authoritative gate is the **job-level** `if: ${{ !github.event.release.prerelease }}` on the `tap` job — it keys off GitHub's release pre-release flag, which is what #30/the publisher actually set. Do **not** rely solely on GoReleaser's `brews.skip_upload: auto`: that keys off a *semver* pre-release suffix in the tag (e.g. `-rc1`), which can diverge from the GitHub pre-release flag (a release can be flagged pre-release with a clean semver tag). The two may both be present, but the job-level `if` is the contract.

**Config-drift guard**: 021/022/023 are protected by the `internal/build` config-guard (`LoadConfig`/`ParseConfig` over `.goreleaser.yaml`). Extend it with a focused assertion that a `brews` entry exists and targets the expected tap (`owner: Luscii`, `name: homebrew-cli-glassfrog`) — so a future blanking or retarget of the brew block fails a test rather than silently shipping no formula (same spirit as 023's ldflags regression-guard note). Keep it focused; do not over-assert formula DSL details the interface owns.

**Testing strategy**: (1) **Render-and-inspect without network** — `goreleaser release --snapshot --skip=publish` renders the formula into `dist/`; a test asserts its shape (binary name, per-platform url/sha256 presence, version), mirroring 027's "drive the script offline" approach. (2) **Homebrew lint** — run `brew style`/`brew audit` against the rendered formula in CI if a Homebrew runner is available, so audit failures surface before a release. (3) **End-to-end `brew install`** is a manual post-first-release validation (a live tap + real release); it cannot be fully exercised in unit tests and is called out as such.

**Idempotency & secrets**: The `tap` job is idempotent (deterministic, `-trimpath` builds → stable sha256 → same formula). The cross-repo token lives only in the `tap` job's env, never in build/verify/publish.

---

## Implementation Strategy

**Phase 1 — Provision the tap repository and credential (prerequisite, partly ops).**
Create `Luscii/homebrew-cli-glassfrog` (empty, default branch, unprotected). Mint the scoped token (ADR-4) and store it as a repository secret in this repo. Also resolve the repository **LICENSE decision** — the repo has no `LICENSE` today, and `brews.license`/`brew audit` need one (interface-spec.md's `[NEEDS INPUT]`, tasks T002): add a `LICENSE` so GoReleaser populates the field, or record a decision that the private tap runs non-strict. This phase is the only one with a manual/ops step; the CI wiring in Phase 3 cannot succeed until it is done. No app code.

**Phase 2 — Add the `brews` block to `.goreleaser.yaml` + config-guard assertion.**
Add the GoReleaser `brews` entry (formula name, tap repository target, `directory: Formula`, install + test + metadata stanzas) keyed off the existing `glassfrog` archives. Extend the `internal/build` config-guard with the focused brews assertion. Add the offline render-and-inspect test. Depends on nothing from Phase 1 (config + tests only); can proceed in parallel with Phase 1.

**Phase 3 — Add the `tap` job to `release.yml`.**
Add the job: `needs: [publish]`, `if: ${{ !github.event.release.prerelease }}`, checkout + setup-go + setup GoReleaser, run the brew-publisher-only invocation with the scoped token in env. Depends on Phase 1 (repo + secret must exist) and Phase 2 (the `brews` config must exist). This is the wiring that produces the first published formula.

Ordering: Phase 1 and Phase 2 in parallel; Phase 3 last (needs both). First real validation is the next stable release after Phase 3 merges, followed by a manual `brew install` from the live tap.

---

## Risks

1. **Cross-repo token misuse or leak** *(low likelihood, high impact — risk.md H-1)* — a new privileged credential in CI. Mitigation: fine-grained PAT scoped to *only* the tap repo with `contents: write` (or an App token), injected only into the `tap` job; rotate/track expiry. Blast radius is one repo, never this repo's `main`.
2. **Pre-release gating mismatch** *(low likelihood given the job-level gate, medium impact — risk.md H-3)* — relying on GoReleaser's `skip_upload: auto` instead of the GitHub pre-release flag could publish a formula for a release a maintainer flagged pre-release. Mitigation: authoritative job-level `if: !github.event.release.prerelease` (Cross-cutting Concerns).
3. **Formula references absent/unverified assets** *(low likelihood, high impact)* — a formula pointing at assets that aren't attached, or weren't verified, would break `brew install`. Mitigation: `needs: [publish]` (assets attached + 4-leg verify passed before tap runs); formula sha256 are the same checksums 022 published, from deterministic builds. Spec error scenario "expected release asset missing" covers the user-visible failure.
4. **`brew audit`/style failure on first real tap** *(medium likelihood, medium impact)* — a rendered formula missing `desc`/`license`/a valid `test` block fails Homebrew's audit, surfacing only when a user installs. Mitigation: offline render-and-inspect + `brew audit` in CI (Phase 2 testing) before a release relies on it.
5. **PAT expiry silently deferring a tap update** *(medium likelihood, low impact)* — a lapsed fine-grained PAT fails the `tap` job. Mitigation: the failure is a loud red release run (not a wrong publish); prefer the App-token upgrade (no expiry) if rotation overhead is unwanted.

---

## What This Plan Does Not Cover

- **Protocol/DSL detail** — the exact `brews` YAML fields, the formula Ruby DSL (`bin.install`, `test do`, `caveats`), the literal tap name and `brew install`/`tap` command strings, and the GoReleaser skip-flags/job YAML. These are the **interface** skill's concern (interface-spec for the channel's command surface and the config contract) and **tasks**' decomposition.
- **Branch protection of the tap repo** — assumed unprotected (default); if the org applies protection to it, the direct-push assumption in ADR-3 must be revisited.
- **Sibling channels** — the Install Script (027) and NPM Wrapper (037) are independent; the dropped cask path is out of scope (the formula subsumes it).
- **Signing/notarization** — out of scope here (integrity = 022's checksums), consistent with the cluster.
