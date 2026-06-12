# Checklist: Release Drafting

**Feature**: 030-release-drafting
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/release-drafting.feature
**Checks**: 6 (6 pass, 0 fail) + 3 P2 considerations
**Generated**: 2026-06-12

> Source note: no `accords/governance/done-*.md` accords are deployed in this repo, so this checklist runs **constitution checks only** (same as sibling CI specs 024/028/029). Done-criteria checks are skipped — not failed.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 6 | 6 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 3 | — | — |
| **Total** | **6 checks** | **6** | **0** |

---

## Constitution Checks: 6/6 passed

- **P0 | III (Fail Safe, Not Silent)** — *calibrated*: Release Drafting's only "write" is a **draft** GitHub Release (not governance data, and not a publish). Calibrated assertions: (a) drafting never publishes or tags — publishing is a deliberate maintainer act (spec §Stops at the draft; ADR-1); (b) the draft is regenerated **authoritatively** each run, so there is no partial-apply that misleads a consumer (spec §Draft maintenance; plan Cross-cutting); (c) a drafting failure is **non-blocking** — the workflow is not a required check, the prior draft is left intact, and the next merge re-reconciles (interface Error Communication). No silent success-on-failure and no partial governance state. **PASS.** (See P2-1 on the `gh release edit` status observability nuance.)
- **P0 | IV (Test-Driven Development / BDD)** — the user-facing behaviours carry acceptance scenarios written before the config: 11 scenarios in `release-drafting.feature` (2 `@validation`, 2 architecture-informed), each `# Source:`-traced to a spec scenario and preceding the workflow/config they describe; tasks.md T001–T004 reference them as verifying conditions. As with siblings 024/028/029, a GitHub-Actions config feature is verified by inspection + a live-merge exercise rather than a godog suite, and the **config-drift guard (T004)** is an executable RED→GREEN test that pins the label contract. **PASS.**
- **P0 | V (Composition over Monolith)** — plan ADR-1 keeps Release Drafting an isolated declarative set (`.github/release-drafter.yml` + `.github/workflows/release-drafting.yml`) on a **separate** workflow from 029, plus one `internal/build` guard test that joins the existing config-guard package without touching any command module. It adds no Go command and couples no resource commands. The coordinated 028 edit (the eighth label) is additive to declarative config, and the guard makes the 028↔030 contract explicit rather than hidden. **PASS.**
- **P0 | VI (Size-Aware by Design / never silently truncate)** — *calibrated*: release-drafter accumulates **every** PR merged since the last published release into the draft (no per-page cap of its own), and the two omissions are **deliberate and legible**, not silent truncation: an explicit `no-release-note` exclusion label (ADR-4) and the spec/feature-only confinement that the labeler applies that same label for. The config guard (ADR-6/T004) makes the category↔label contract fail **loudly** on desync. No noteworthy change is dropped without a labelled, contract-pinned reason. **PASS.**
- **P0 | VII (Working Software)** — the capability adds **no** CLI Go code; its only Go is the T004 guard test, which lands with the config it guards. Each task leaves a valid state (a complete `release-drafter.yml`, a valid workflow, a passing guard). No code-only/test-only split outside that. **PASS.** (See P2-2: T004 should ship in the same PR as the config it guards.)
- **P0 | XII (Standalone Executable) — no new runtime dependency** — Release Drafting introduces the `release-drafter` action and `gh` as **CI-host** tools only; plan Cross-cutting (Security/permissions) and interface Consistency Notes (the "CONSTITUTION XII note") record that XII governs the produced binary's runtime, not the CI host — the same standing the project gives GoReleaser, golangci-lint, srvaroa/labeler, and `sigs.k8s.io/yaml`. It adds nothing to the distributed artifact. **PASS.**

### Principles with no applicable checks for this feature

- **I (Spec Fidelity)**, **II (Action Transparency)**, **VIII (No Fabricated Data)**, **IX (Writes Require Explicit Intent)** — these govern the CLI's command/request/output surface against the Glassfrog API v5 spec. Release Drafting adds no CLI command, issues no Glassfrog API call, and renders no API data — its inputs are merged-PR metadata and its output is a GitHub draft release. No applicable checks. *(Thematic alignment worth noting, not a check: IX's "writes require explicit intent" is mirrored by the spec's "never auto-publish" — the actual release mutation requires a deliberate maintainer publish.)*
- **X (Respect API Limits)**, **XI (Governance via Proposals)** — concern the Glassfrog API's rate limits/concurrency and governance-structure mutation. Release Drafting touches neither. No applicable checks.

---

## P2 Considerations (advisory — not blocking)

- **P2-1 | III — auto-status observability** [ASSUMED, ADR-5]: the pre-release/latest status is set by a post-step running `gh release edit` against the **unpublished draft** keyed by its tag. If `gh` cannot address a draft by tag (or silently no-ops), the draft could carry the wrong status without surfacing an error — a quiet Fail-Safe gap. Verify at implement (tasks.md T003 already flags it); the documented fallback (static `prerelease` with a 1.0.0 flip) is acceptable for the current 0.x phase. Recommend the step fail loudly if the edit does not take effect.
- **P2-2 | VII — guard ships with its config**: T004 (the `internal/build` label-contract guard) should land in the **same PR** as T001/T002 (the label + `release-drafter.yml` it guards). The tasks already note the four may collapse into one or two PRs; merging the config without its guard would leave the 028↔030 label contract unguarded between PRs — the exact gap T004 exists to close.
- **P2-3 | VI — first-release v0.1.0 floor** [ASSUMED, ADR-2]: a patch/docs-only first release computes `v0.0.1` under release-drafter's native 0.0.0 base, below the spec's `v0.1.0` floor. Not silent truncation, but a quiet contract deviation; verify at implement (tasks.md T003 risk) and pin the floor if needed.
