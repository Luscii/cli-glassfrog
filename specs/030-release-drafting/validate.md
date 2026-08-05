# Validate: Release Drafting

**Feature**: 030-release-drafting
**Round**: 1 of 3
**Date**: 2026-06-12
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (4 of 4 tasks complete), interface-spec.md, features/no-automated-pipeline/release-drafting.feature, PROJECT.md
**Implementation files**: 4 — `.github/settings.yml` + `.github/labeler.yml` (label contract), `.github/release-drafter.yml` (drafting config), `.github/workflows/release-drafting.yml` (workflow), `internal/build/labelcontract.go` + `labelcontract_test.go` (config-drift guard)

> **Tooling note**: `guardian-agent.md` and the context-engineering / self-verification references are not deployed in this Score install — applied the SKILL.md process and skill-specific self-checks alone (reduced character consistency, not a blocked skill). The few-shot dimension is covered by `validate-template.md`, which was loaded at compose time.

> **Evidence standard for a CI-config feature**: This feature ships no runtime CLI code (plan § Testing strategy: "there is no local harness for the GitHub-hosted draft"). Driving scenarios describe GitHub-hosted release-drafter behaviour and are traced to the committed declarative config that realizes them, plus the `internal/build` guard — the same level at which 022's release behaviour is verified. Two checks were executed for additional confidence: `go test ./internal/build/ -run TestLabelContract` (pass) and `actionlint` on the workflow (clean).

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

| Scenario | Status | Implementation |
|---|---|---|
| a feature merge bumps the draft to the next minor and files a note | ✓ Covered | `release-drafter.yml` `version-resolver` minor category (`when.labels: [features]`; positions per 071) + Features note category (`when.labels: [features]`); `release-drafting.yml` runs release-drafter on `push:main`; no publish step keeps it a draft |
| highest semver label wins across several merges | ✓ Covered | the three `version-resolver` categories `major:[breaking]`/`minor:[features]`/`patch:[fixes]` (positions per 071) — release-drafter resolves highest-wins natively; all three labels also mapped as note categories |
| a docs-only change drafts under Docs with a default patch bump | ✓ Covered | `categories` `docs`→"Documentation" section; the condition-less `version-resolver` category (`semver-increment: patch`, the declared fallback; position per 071) |
| the draft is never published automatically | ✓ Covered | `release-drafting.yml` has no publish/tag step; release-drafter creates a *draft*; the status PATCH sets `prerelease`/`make_latest` only, never `draft:false` |
| an excluded pull request affects neither notes nor version | ✓ Covered | `release-drafter.yml`'s `pre-exclude` category (`when.labels: [no-release-note]`; position per 071) — release-drafter drops excluded PRs from notes *and* bump resolution |
| a spec-only pull request is omitted without a label | ✓ Covered | `labeler.yml` `no-release-note` negate-over-noteworthy matcher applies the label to specs/.feature-confined PRs → dropped via the `pre-exclude` category (position per 071) |
| the first ever release proposes v0.1.0 as a pre-release | ✓ Covered | release-drafter from the `0.0.0` base + `features`→minor yields `v0.1.0`; post-step `major_version == 0` → `prerelease=true` |
| reconciliation converges rather than duplicating | ✓ Covered | release-drafter maintains one draft keyed to the unpublished release it owns (authoritative regenerate); `concurrency.cancel-in-progress` keeps the latest tip |

All driving scenarios trace to identifiable config/workflow paths. The "Docs" vs "Documentation" section-title wording (scenario says "Docs", config titles it "Documentation") is display text the interface explicitly marks `[ASSUMED]`/tunable — the binding contract is the `docs` *label*, which matches; not a gap.

---

## Acceptance Criteria

**Status**: Pass (all criteria met for all 4 checked tasks)

| Task | Status | Evidence |
|---|---|---|
| T001 — `no-release-note` label | ✓ Met | `settings.yml` adds the 8th labels-only entry (7 unchanged, parses clean); `labeler.yml` negate block carries only `files:`; label name identical across both files + the drafter exclusion (`exclude-labels` then; a `pre-exclude` category since 071) + guard |
| T002 — `release-drafter.yml` | ✓ Met | seven `categories` labels exact; resolver buckets exactly breaking→major/features→minor/fixes→patch + default patch; exclusion lists `no-release-note`; v-prefixed templates; YAML valid (validated against 030's schema; 071 moved the same invariants to `when.labels` note categories, `version-resolver` categories with a condition-less patch fallback, and a `pre-exclude` category) |
| T003 — `release-drafting.yml` | ✓ Met | `on: push:{branches:[main]}` (no tags/PR); `permissions: contents:write + pull-requests:read` exactly; action pinned at an exact patch (`@v6.4.0` at this validation; `@v7.7.0` since 071's coordinated schema/major migration); never publishes; 0.x→pre-release / ≥1.0.0→latest; not a required check; actionlint clean |
| T004 — label-contract guard | ✓ Met | `CheckLabelContract` fails on renamed/dropped/added category label, drifted resolver bucket, missing `no-release-note` in any of three places, or a managed set ≠ 8; `TestLabelContract_RealConfig` passes against shipped files; runs under `go test ./...` |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Notes |
|---|---|---|
| `release-drafting.yml` structure | ✓ Conformant | name, `push:main` trigger, exact permissions, concurrency group + cancel-in-progress, `draft` job on ubuntu-latest with pinned release-drafter step (id `draft`) + status post-step, no checkout — all match |
| `release-drafter.yml` structure | ✓ Conformant | tag/name/version templates, the note/exclusion/version-resolver categories (Breaking→Internal order; positions per 071), change-template, template, `prerelease` left default — all match the interface block |
| Eighth label `no-release-note` | ✓ Conformant | `labeler.yml` negate block matches the interface block verbatim; `settings.yml` entry uses color `EDEDED` + the specified description |
| `internal/build` label-contract guard | ✓ Conformant | all four interface assertions implemented (categories-agree, resolver buckets, exclusion-label-present-in-three, managed-set-of-eight) |

**Sanctioned `[ASSUMED]` resolutions (conformant, not deviations):** The interface marked the status-step `gh` invocation and the action pin as `[ASSUMED]`, instructing verify-at-implement with a documented fallback. Implementation resolved both: (1) status via `gh api -X PATCH /releases/{id}` rather than the example `gh release edit <tag>` — the latter 404s on an unpublished draft (no real git tag), so the by-id form is the reliable realization of the same ADR-5 intent; (2) pinned `release-drafter@v6.4.0` because the example `v6.1.0` does not exist (071 later moved the pin to `v7.7.0` with the schema migration). Both honor the load-bearing contract (status follows version via a pinned post-step) and are recorded in `.score/memory/LEARNINGS.md`.

---

## Non-Behavior Absence

**Status**: Pass (all 6 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not publish / create the git tag | ✓ Absent | No publish or tag step; release-drafter produces a draft; the PATCH never sets `draft:false` |
| Must not build/package/attach/upload binaries | ✓ Absent | Workflow has no goreleaser/upload steps — only draft + status |
| Must not run lint/tests, nor gate/block the merge | ✓ Absent | Runs on `push:main` (post-merge), drafts only; not wired as a required check (`settings.yml` is labels-only, no branch-protection block) |
| Must not apply/remove/manage PR labels | ✓ Absent | `release-drafting.yml` has `pull-requests: read` (not write) and no labeler step — it only *reads* the 028 labels. The `no-release-note` label is *applied* by 028's separate `pr-administration.yml`, not by this stage |
| Must not run on PR events / compute at PR time | ✓ Absent | Trigger is `push:{branches:[main]}` only — no `pull_request`/`pull_request_target` |
| Must not decide status at publish time or re-mark a published release | ✓ Absent | Status is set on the *draft* (pre-publish) by the post-step targeting the draft by id; 022 honors it downstream |

---

## @wip Lifecycle Completion

**Status**: Pass

The only `@wip` tags remaining in `release-drafting.feature` are on the two `@validation`-tagged scenarios (lines 90, 110). This matches the project-wide convention — merged specs 022 and 028 retain `@validation @wip` after implementation — and the validate skill's own Step 5, which treats `@validation` scenarios as held-out independent verification (evaluated in the section below). No non-`@validation` scenario carries a lingering `@wip`. The other behavioral scenarios were authored without `@wip` (inspection-level, no local harness). Implement correctly did not de-wip the held-out `@validation` scenarios.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced independently)

| Scenario | Source | Status | Trace |
|---|---|---|---|
| the spec names no specific drafting tool or workflow syntax in its behaviors | spec.md § Validation Scenarios #1 | ✓ Satisfied | Behavioral Accord + Non-Behaviors describe outcomes (draft state, version, categories, status) with no GitHub Action / workflow / YAML named; release-drafter appears only under Assumptions, marked `[ASSUMED]` |
| every category the notes can produce traces to an 028 managed label | spec.md § Validation Scenarios #2 = feature line 91 | ✓ Satisfied | `internal/build/labelcontract.go` `CheckLabelContract` asserts `categories` labels == the seven canonical 028 labels, cross-checked against `labeler.yml`/`settings.yml`; `TestLabelContract_RealConfig` passes against the shipped files — executable, independent code path |
| a drafting failure blocks nothing | feature line 111 (architecture-informed) | ✓ Satisfied | `release-drafting.yml` runs on `push:main` (post-merge, the merge already happened) and is not wired into any required-status/branch-protection config; release-drafter regenerates its own draft, so a failed run leaves the prior draft intact |

---

## Verdict: Ready

All 4 tasks are checked. All 5 conformance dimensions pass with zero findings, and all 3 validation scenarios are satisfied through independent inspection (one of them executable, via the config-drift guard). The implementation conforms to its specification: the committed declarative config + workflow realize every behavioral guarantee (highest-wins bump with patch default, v0.1.0 first-release floor, category↔label fidelity, label-and-path exclusion, authoritative reconciliation, version-derived pre-release/latest status, stops-at-the-draft), and every non-behavior is absent. The two `[ASSUMED]` interface items were resolved as the interface instructed (verify-at-implement) and are conformant, not deviations.

**Observation (not a finding — outside spec-conformance scope):** `.score/memory/LEARNINGS.md` records a design tension where a maintainer hand-applying `no-release-note` to a *noteworthy* PR would have it stripped by 028's sync labeler on the next PR event. The implementation faithfully realizes ADR-4's specified exclusion behavior; spec.md § Exclusion only promises that a PR *marked* with the label is omitted, which holds. The "maintainer-flagged" framing that previously appeared in the `settings.yml`/`release-drafter.yml` comments and the interface accord was **corrected during PR #102 triage** (Copilot independently flagged it) — those now describe the durable behavior (auto-exclude spec/feature-only PRs). The residual question — whether durable maintainer suppression of *code* PRs is intended — is a spec-level matter worth the authors' attention for a future cycle (candidate for `/score:clarify`), but it does not block this verdict.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.

- When opening the PR, surface the announced cross-spec divergence: this widens 028's recorded "EXACTLY seven managed labels" decision to eight — a candidate for `/score:deprecate` (already flagged in tasks.md and DECISIONS.md).
- The two `[ASSUMED]` resolutions and the maintainer-flagged tension are recorded in `.score/memory/LEARNINGS.md` for future specification cycles; the interface accord could be updated to the by-id `gh api` form and a real action pin if revised.
