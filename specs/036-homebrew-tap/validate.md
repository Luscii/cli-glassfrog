# Validate: Homebrew Tap

**Feature**: 036-homebrew-tap
**Round**: 2 of 3
**Date**: 2026-06-13
**Verdict**: Issues
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, homebrew-tap.feature, PROJECT.md, validate.md (round 1)
**Implementation files**: `.goreleaser.yaml` (brews + refinements), `LICENSE`, `.github/workflows/release.yml` (tap job + GoReleaser pin), `internal/build/config.go` + `workflow.go` (guards), `internal/build/{config_guard,workflow,formula_render,homebrew_tap_bdd}_test.go`; tap repo `Luscii/homebrew-cli-glassfrog` (created)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass (lower-confidence; brew-side is manual) | 0 |
| Acceptance criteria | ✗ Fail (T001 deviates from its criteria) | F-1, F-2 |
| Interface contract conformance | ✓ Pass (3 intent-conformant adaptations) | F-4, F-5 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✗ Fail (T004 scenarios still @wip) | F-3 |
| **Validation scenarios** | ◐ Partially satisfied (1 structural; 2 need a live tap) | — |

**Total**: 5 dimensions checked, 3 pass / 2 fail; 5 findings (2 medium-ish operational, 3 low intent-conformant). **All 4 tasks complete.** Verdict **Issues** — incremental/operational, not fundamental gaps.

---

## Driving Scenario Coverage

**Status**: Pass (lower-confidence). The code paths that *produce* each scenario's outcome now exist and are guarded; the Homebrew-side install/verify behavior is the package manager's and needs a live, public tap to exercise end-to-end (manual).

| Scenario | Status | Implementation / proxy |
|---|---|---|
| Fresh install on macOS / Linux | ◑ Covered (manual e2e) | Tap job publishes a correct `Formula/glassfrog.rb` (render test verifies shape + 4 url/sha256); `bin.install` places the binary. Actual `brew install` is manual. |
| Upgrade to latest stable / no-op when current | ◑ Covered (manual e2e) | Each stable release re-renders the formula at the new version (reproducible); `brew upgrade` is Homebrew's behavior. |
| Checksum mismatch refuses install / missing asset fails | ◑ Covered (manual e2e) | Formula sha256 == release `checksums.txt` (render test asserts); URL points at the published asset. Homebrew enforces the integrity check. |
| Pre-release does not move the tap | ✓ Covered | `tap` job `if: ${{ !github.event.release.prerelease }}`; `CheckTapJob` asserts the gate. |

---

## Acceptance Criteria

**Status**: Fail — T002/T003/T004 criteria met; **T001's credential and visibility criteria diverge** from what was implemented (developer-approved, documented), and one criterion is unverified.

| Task | Status | Evidence / gap |
|---|---|---|
| T001 — tap repo + scoped token + secret | ✗ Partial | Repo created (✓), but: token is the **org-wide `CI_GITHUB_TOKEN`**, not the criterion's *tap-scoped least-privilege* token; **no dedicated `HOMEBREW_TAP_TOKEN` secret** (env-mapped instead); repo is **internal**, not the criterion's *public*; "main writable by the token" **unverified**. See F-1, F-2. |
| T002 — LICENSE + `brews.license` | ✓ Met | MIT `LICENSE` at root; `brews.license: "MIT"` (set explicitly — GoReleaser doesn't auto-fill offline); rendered formula carries `license "MIT"`; audit no longer fails on a missing license. |
| T003 — brews block + config-guard + render test | ✓ Met | (Round 1) + triage hardening: commit-anchored mtime guard, branch pin, CI-runnable parse test. |
| T004 — tap job + workflow guard | ✓ Met (structurally) | `tap` job (`needs: [publish]`, `if: !prerelease`, token env, brew-publisher-only); `CheckTapJob` asserts the contract with drift tests; impl+guard same PR. Runtime publish behavior is manual (no live release yet). |

---

## Interface Contract Conformance

**Status**: Pass. All surfaces (brews section incl. `license: MIT`, `release.disable`, archive mtime, config-guard, generated formula, the `release.yml` tap job) conform. Three low-severity, intent-conformant adaptations (F-4, F-5).

---

## Non-Behavior Absence

**Status**: Pass.

| Non-behavior | Status | Evidence |
|---|---|---|
| No build / Go toolchain on host | ✓ Absent | `bin.install "glassfrog"`; no compile stanza |
| No pre-release tracking | ✓ Absent | Authoritative job-level `if: !github.event.release.prerelease` (now implemented) |
| No commit to this repo's `main` | ✓ Absent | Tap job pushes to the separate `homebrew-cli-glassfrog`; `release.disable: true` |
| No release authoring / signing / version bumps | ✓ Absent | `release.disable: true`; no signing/version code |
| No platforms beyond the four targets | ✓ Absent | brews `ids` draws from the four-target build; rendered formula has exactly darwin/linux × amd64/arm64 |

---

## @wip Lifecycle Completion

**Status**: Fail — F-3. T003's two scenarios are un-`@wip`'d. But **T004 is now checked and references the install / upgrade / missing-asset / checksum-mismatch / pre-release scenarios, which remain `@wip`.** They are not CI-exercisable (they need a live, public tap + a real published release), so they were intentionally left `@wip` with the `CheckTapJob` workflow guard as the structural proxy. This is a real divergence from the "remove `@wip` when the referencing task completes" expectation — see F-3 for resolution options.

---

## Validation Scenario Results

**Status**: Partially satisfied — 1 of 3 structurally provable now; 2 need a live tap (manual).

| Scenario | Status | Trace |
|---|---|---|
| A pre-release leaves the tap repository untouched | ✓ Satisfied (structural) | `tap` job `if`-gate skips pre-releases; `CheckTapJob` asserts it |
| The installed binary matches the release the formula points at | ◐ Not traceable in CI | Needs a live `brew install` from the public tap (manual post-first-release) |
| A mismatched checksum never lets a binary reach PATH | ◐ Not traceable in CI | Homebrew install-time integrity behavior; manual |

---

## Findings

### F-1: T001 credential diverges from the acceptance criterion (org-wide token, no dedicated secret)

- **Dimension**: Acceptance criteria
- **Source**: tasks.md § T001 — "least-privilege credential scoped to *only* that repository … stored as a repository secret named `HOMEBREW_TAP_TOKEN`"
- **Implementation**: `.github/workflows/release.yml` `tap.env.HOMEBREW_TAP_TOKEN: ${{ secrets.CI_GITHUB_TOKEN }}`
- **Gap**: The implemented credential is the **org-wide `CI_GITHUB_TOKEN`** mapped into the `HOMEBREW_TAP_TOKEN` env var, not a tap-repo-scoped least-privilege token, and there is no dedicated `HOMEBREW_TAP_TOKEN` repo secret. Developer-approved and recorded in `.score/memory/DEPRECATION.md` (ADR-4 deviation: broader blast radius than tap-only). Low-medium: a deliberate, documented tradeoff — surfaced here because it diverges from the task's written criterion.

### F-2: Tap repo is internal and write-access unverified (operational preconditions)

- **Dimension**: Acceptance criteria
- **Source**: tasks.md § T001 — "Create the empty **public** repository … its `main` is **writable by the minted token**"
- **Implementation**: `Luscii/homebrew-cli-glassfrog` created **internal**; token write-access not yet exercised
- **Gap**: While `internal`, anonymous `brew tap`/`brew install` cannot resolve the tap, so the user-facing channel is not yet live (flip to **public** before relying on it). And `CI_GITHUB_TOKEN`'s `contents: write` on the new repo is unconfirmed — the first stable release's `tap` job is the first real exercise (it fails loud if the token can't push). Operational, not a code defect.

### F-3: @wip remains on T004-referenced scenarios (not CI-exercisable)

- **Dimension**: @wip lifecycle completion
- **Source**: features/runtime-dependent-distribution/homebrew-tap.feature — the install/upgrade/missing-asset/checksum-mismatch/pre-release scenarios; tasks.md § T004 scenario references
- **Implementation**: tap job + `CheckTapJob` guard (the structural proxy)
- **Gap**: T004 (checked) references these scenarios, but they remain `@wip` because they describe live Homebrew install/upgrade behavior that cannot run in CI without a public tap and a real release. Resolution: either reclassify them as manual/held (e.g., keep them `@wip` by intent and document, or tag `@validation`), or exercise them on the live tap as the post-first-release manual validation. Intentional — flagged so the divergence is explicit, not silent.

### F-4: Archive mtime realized as `builds_info.mtime` (carried from round 1)

- **Dimension**: Interface contract conformance
- **Source**: interface-spec.md § Refinements — "Pin `archives.mtime`"
- **Implementation**: `.goreleaser.yaml` `archives.builds_info.mtime: "{{ .CommitDate }}"`; guard `mtimeIsCommitAnchored`
- **Gap**: GoReleaser ~> v2 has no top-level `archives.mtime`; realized via `builds_info.mtime`. Intent (reproducible archives) met and tested; the guard now requires a commit-anchored value. Low.

### F-5: `url_template` added and GoReleaser pinned to `~> v2.16`

- **Dimension**: Interface contract conformance
- **Source**: interface-spec.md § brews fields (no `url_template`); § Producer/CI (`version: "~> v2"`)
- **Implementation**: `.goreleaser.yaml` `brews.url_template`; `release.yml` `version: "~> v2.16"`
- **Gap**: `url_template` is required because `release.disable: true` removes GoReleaser's default URL template (it reproduces the interface's exact URL hard-contract). The GoReleaser pin (vs the interface's `~> v2`) is the deliberate decision to keep the deprecated `brews` publisher (recorded in DEPRECATION.md). Both intent-conformant. Low.

---

## Changes Since Previous Run

**Round**: 2 (previous: Round 1 — Not Ready (partial), 1 of 4 tasks)

### Resolved
- The partial state — **all 4 tasks now complete** (T001 repo created, T002 MIT LICENSE, T003 merged + hardened, T004 tap job + guard). Round 1's "0 of 7 driving scenarios / 0 of 3 @validation traceable" is now "driving scenarios covered (manual e2e) / 1 of 3 @validation structurally provable."

### Carried / re-scoped
- F-1 (round 1, mtime field) → **F-4** here, unchanged (intent-conformant).
- F-2 (round 1, url_template) → folded into **F-5** here, plus the new GoReleaser pin.

### New
- **F-1** (T001 credential deviation), **F-2** (internal repo + unverified write), **F-3** (@wip on T004 scenarios). The first two are operational/approved deviations; the third is an intentional, now-explicit deferral.

---

## Verdict: Issues

All four tasks are implemented and the structural conformance is strong: the formula renders correctly (MIT license, four platforms, checksums matching the release file), every non-behavior is absent, the config-guard and the new `CheckTapJob` workflow guard pin the contract with change-detector rigor, and the full suite is green. The findings are incremental and mostly **operational or intentional-deferral**, not fundamental gaps requiring rethinking:

- **F-1/F-2** — T001's credential and visibility diverge from its written acceptance criteria (org-wide token vs least-privilege; internal vs public; write-access unverified). Developer-approved and documented, but real divergences, and the tap is not yet *live-functional* (internal repo → no anonymous install).
- **F-3** — the live-install scenarios referenced by T004 stay `@wip` because they can't run in CI; the workflow guard is the structural proxy and `brew install` is the manual post-first-release check.
- **F-4/F-5** — low-severity, intent-conformant tooling adaptations.

---

## Next Steps

Mostly operational + a manual validation pass — not an `/score:implement` code round:

1. **Flip the tap repo to `public`** and **confirm `CI_GITHUB_TOKEN` has `contents: write`** on it (F-1, F-2) before the first stable release relies on the tap.
2. **Cut the first stable release** and run the **manual `brew install`/`brew upgrade`** validation from the live tap — this exercises the held `@wip` driving scenarios and the two non-CI-traceable `@validation` scenarios (F-3). Reclassify or un-`@wip` them once exercised.
3. **Accept or revisit** the documented deviations (F-1 org token vs least-privilege; F-5 GoReleaser pin) — both are tracked in DEPRECATION.md.
4. Re-validate (or close out) after the live run. The remaining items are not code defects, so a re-`/score:implement` is not required; this PR's code is mergeable on its own merits.
