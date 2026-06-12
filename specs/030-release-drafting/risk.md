# Risk Assessment: Release Drafting

**Feature**: 030-release-drafting
**Round**: 1
**Date**: 2026-06-12
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Matrix**: Using default 3×3 traffic-light matrix — no project-level risk acceptability matrix or Regulatory Context found in PROJECT.md (no IEC 14971 bridge required).

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Level | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Drafting auto-publishes / creates a tag, shipping an un-reviewed release and triggering 022 | spec §Stops at the draft; Non-Behavior #1 | High | Low | Yellow | RC-1, RC-2 | Yellow (accepted) |
| H-2 | A *noteworthy* PR is silently excluded (negate-over-noteworthy mis-classifies real change as spec/feature-only) | spec §Exclusion; CONSTITUTION VI; plan ADR-4 | Medium | Low | Green | RC-3, RC-4, RC-5 | Green |
| H-3 | Draft carries the wrong pre-release/latest status (gh edit silently no-ops on a draft), and 022 honors it | plan ADR-5 `[ASSUMED]`; checklist P2-1 | Medium | Medium | Yellow | RC-6, RC-7 | Green |
| H-4 | First release computes `v0.0.1` instead of the `v0.1.0` floor (patch/docs-only first release) | plan ADR-2 `[ASSUMED]`; checklist P2-3 | Low | Medium | Green | RC-7, RC-5 | Green |
| H-5 | 028↔030 label contract desyncs (a category renamed in one file), mis-categorizing notes or mis-resolving the bump | plan ADR-6; spec Integration Boundaries | Medium | Low | Green | RC-8 | Green |
| H-6 | Wrong semver bump (multiple/zero semver labels mis-resolved) | spec §Version computation; plan ADR-2 | Medium | Low | Green | RC-8, RC-9, RC-5 | Green |
| H-7 | `contents: write` token abused (action alters/deletes releases) | plan Cross-cutting (permissions); interface workflow contract | High | Low | Yellow | RC-10, RC-11, RC-12 | Yellow (accepted) |
| H-8 | Supply-chain: the third-party release-drafter action is compromised at the pinned tag | plan ADR-1; interface Consistency Notes | High | Low | Yellow | RC-11, RC-12 | Yellow (accepted) |

No **Red** (unacceptable) residual risks. Three Yellow residuals are accepted with justification below; the rest reduce to Green.

---

## Hazard Detail

**H-1 — Auto-publish.** *Severity High*: a published release ships un-reviewed and triggers 022 to build/attach binaries — the whole point of the draft is to gate this on a human. *Probability Low*: the workflow has **no** publish/`gh release create` step, and release-drafter creates **drafts** by default. Materializing this would require a deliberate misconfiguration.
- **RC-1**: release-drafter operates in draft-only mode (its default; the workflow adds no publish step) — the draft never has a git tag until a maintainer publishes (spec §Stops at the draft).
- **RC-2**: the status post-step uses `--draft` so editing pre-release/latest never publishes (interface workflow contract).
- *Residual Yellow (accepted)*: structurally the workflow cannot publish; publishing is a separate maintainer act. Accepted — reducing further would require removing a capability the feature doesn't have.

**H-2 — Noteworthy PR silently excluded.** *Severity Medium*: a real change missing from the notes (and not counted for the bump) misleads consumers — but it is caught before harm because the draft is reviewed before publishing. *Probability Low*: the negate matcher reuses 028's already-enumerated noteworthy paths plus a `.*\.go$` code catch-all, so the common cases (code, docs, infra, deps) are covered.
- **RC-3**: noteworthy patterns reuse 028's labeler path set + a `.go` catch-all, so any code/doc/infra/deps change is noteworthy (plan ADR-4 / interface eighth-label section).
- **RC-4**: the exclusion is **label-driven and visible** — `no-release-note` appears on the PR, so a wrong exclusion is observable on the PR, not hidden.
- **RC-5**: the maintainer reviews the draft (contents + version) before publishing — a human gate on the final notes.
- *Residual Green*: a new noteworthy path category not yet in the pattern set would need adding; visible-label + human-review keep it from silently shipping.

**H-3 — Wrong pre-release/latest status.** *Severity Medium*: 022 honors whatever status the draft carries, so a 0.x release marked latest (or a 1.x marked pre-release) ships mis-flagged; recoverable by editing the published release. *Probability Medium pre-control*: `gh release edit` on an *unpublished* draft is `[ASSUMED]` (ADR-5).
- **RC-6**: verify `gh release edit` addresses a draft by tag at implement (tasks T003); make the step **fail loudly** if the edit does not take effect (so a mis-status is not silent — CONSTITUTION III).
- **RC-7**: documented fallback — release-drafter's static `prerelease` config with a 1.0.0 flip, correct for the project's current 0.x phase.
- *Residual Green*: fail-loud + a working fallback drop probability to Low.

**H-4 — Wrong first-release floor.** *Severity Low*: `v0.0.1` vs `v0.1.0` on the first release is cosmetic and recoverable. *Probability Medium*: only if the first release is patch/docs-only (a first CLI release almost always carries a feature → `v0.1.0` natively).
- **RC-7** (fallback config / seed) and **RC-5** (maintainer reviews the proposed version before publishing). *Residual Green*.

**H-5 — Label-contract desync.** *Severity Medium*: a category renamed in `settings.yml`/`labeler.yml` but not `release-drafter.yml` (or vice versa) mis-files notes or mis-resolves the bump. *Probability Low*: the config guard catches it.
- **RC-8**: the `internal/build` config guard (ADR-6 / T004) asserts all eight labels agree across the three files, running in the existing `go test ./...` matrix (024 pre-merge, 029 post-merge) — desync reddens CI. *Residual Green*.

**H-6 — Wrong semver bump.** *Severity Medium*, *Probability Low*: release-drafter resolves highest-wins natively and the guard pins the resolver buckets to breaking/features/fixes.
- **RC-8** (guard pins buckets), **RC-9** (release-drafter native highest-wins + `default: patch`), **RC-5** (maintainer reviews the proposed version). *Residual Green*.

**H-7 — `contents: write` token abuse.** *Severity High*: write scope on releases could alter/delete them. *Probability Low*: scope is the minimum to maintain a draft release.
- **RC-10**: least-privilege — `contents: write` + `pull-requests: read` only; no `id-token`, no extra secrets (plan Cross-cutting; interface workflow contract).
- **RC-11**: the action is pinned to a concrete version (no floating tag).
- **RC-12**: the workflow runs on trusted `main` and performs **no PR-head checkout**, so it never executes untrusted contributor code with the write token (unlike 028's `pull_request_target`).
- *Residual Yellow (accepted)*: `contents: write` is the irreducible minimum to write a draft release; accepted with least-privilege + pinned + no-untrusted-code controls.

**H-8 — Supply-chain (third-party action).** *Severity High*, *Probability Low*: a compromised release-drafter at the pinned tag could misuse the token.
- **RC-11** (pinned to a concrete version, so an upstream tag move can't silently change behavior) and **RC-12** (no untrusted-code execution; scoped token). *Residual Yellow (accepted)*: the same standing the project gives every pinned CI action (GoReleaser, srvaroa/labeler, golangci-lint).

---

## Residual Risk Summary

8 hazards, 12 controls. No Red residuals. Three Yellow (accepted): H-1 (cannot-publish-by-construction), H-7 and H-8 (irreducible `contents: write` + pinned third-party action, scoped by least-privilege and no-untrusted-code). The remaining five reduce to Green, primarily via the config guard (RC-8), the human publish gate (RC-5), and fail-loud status handling (RC-6).

The two `[ASSUMED]` items (RC-6 gh-on-draft, RC-7 first-release floor) are the only implementation-time verifications that affect residual risk; both have documented fallbacks, so neither can escalate to Red.

---

## Traceability Index

**Hazard → source**
- H-1 → spec §Stops at the draft, Non-Behavior #1 | H-2 → spec §Exclusion, CONSTITUTION VI, plan ADR-4 | H-3 → plan ADR-5, checklist P2-1 | H-4 → plan ADR-2, checklist P2-3 | H-5 → plan ADR-6, spec Integration Boundaries | H-6 → spec §Version computation, plan ADR-2 | H-7 → plan Cross-cutting (permissions), interface workflow contract | H-8 → plan ADR-1, interface Consistency Notes

**Control → grounding** (downstream tasks/scenarios reference these RC-N IDs)
- RC-1/RC-2 → interface workflow contract (draft-only, `--draft`) | RC-3 → plan ADR-4 / interface eighth-label (T001) | RC-4 → label visible on PR | RC-5 → maintainer publish gate (spec §Stops at the draft) | RC-6/RC-7 → plan ADR-5/ADR-2 (T003) | RC-8 → plan ADR-6 config guard (T004) | RC-9 → release-drafter version-resolver (T002) | RC-10/RC-11/RC-12 → plan Cross-cutting + ADR-1 (T003)
