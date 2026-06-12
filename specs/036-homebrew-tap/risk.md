# Risk: Homebrew Tap

**Feature**: 036-homebrew-tap
**Round**: 1
**Generated**: 2026-06-12
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md declares no Regulatory Context (not a regulated domain)

---

## Risk Register

| ID | Hazard | Source | Sev | Prob | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Cross-repo token leaks or is over-scoped → attacker pushes a poisoned formula users `brew install` (supply chain) | plan ADR-4, Risk #1; interface Error Comm. | High | Low | Yellow | RC-1, RC-2 | **Yellow** |
| H-2 | Tap-job rebuild not byte-reproducible → formula sha256 ≠ published assets → every `brew install` fails integrity | plan Risk #3; interface Refinements; tasks T003 | Medium | Medium | Yellow | RC-3, RC-4, RC-5 | **Green** |
| H-3 | Stable-only gate mis-wired → a pre-release formula lands in the tap; `brew upgrade` moves users to an unstable build | spec non-behavior; plan Risk #2; tasks T004 | Medium | Low | Yellow | RC-6 | **Green** |
| H-4 | Tap job (running `goreleaser release`) creates/modifies the GitHub release → clobbers #30's notes/status or taints the release | spec non-behavior; plan ADR-3; interface Refinements | High | Low | Yellow | RC-7, RC-8 | **Yellow** |
| H-5 | Formula fails `brew audit` on a real tap (no `LICENSE` in repo) → acquisition blocked/warned | interface `[NEEDS INPUT]`; tasks T002 | Medium | Medium | Yellow | RC-9, RC-10 | **Green** |
| H-6 | The two refinements to 022's config (`release.disable`, `archives.mtime`) regress the landed build→verify→publish pipeline | interface Refinements; plan Consistency | High | Low | Yellow | RC-11, RC-12 | **Yellow** |

No residual risk is Red. Three Yellow residuals (H-1, H-4, H-6) are acceptable with the documented controls below.

---

## Hazard Detail

### H-1 — Token leak / over-scope → poisoned formula (supply chain)
- **Severity: High** — the tap is what users `brew install`; a formula pushed by a leaked token could point `glassfrog` at attacker-controlled bytes, executed on users' machines. Blast radius is every brew user.
- **Probability: Low** — mitigated by design: a fine-grained credential scoped to only the tap repo, used only in CI.
- **Controls**: **RC-1** — least-privilege token scoped to *only* `homebrew-cli-glassfrog` with `contents: write` (plan ADR-4; tasks T001). **RC-2** — token referenced only in the `tap` job's `env`, never in build/verify/publish (interface § `tap` job; tasks T004).
- **Residual: Yellow** — accepted with justification: least privilege bounds the blast radius to the tap repo, and CI-only exposure limits the attack window. An App token (no user-owned PAT, no expiry) would further reduce probability; recommended if the org prefers it.

### H-2 — Non-reproducible rebuild → formula checksums diverge
- **Severity: Medium** — fails **safe**: brew refuses the install on a checksum mismatch (no wrong binary lands), so this is an availability outage of the channel, not a security/data hazard.
- **Probability: Medium (pre-control)** — non-deterministic archive `mtime` is a common GoReleaser default and the `tap` job rebuilds in a separate job from the one that produced the published archives.
- **Controls**: **RC-3** — pin `archives.mtime` so archives are byte-reproducible across jobs (tasks T003; interface Refinements). **RC-4** — offline render-and-inspect test asserting each formula `sha256` equals the snapshot `checksums.txt` entry (tasks T003). **RC-5** — documented fallback: derive the formula's `sha256` from the published `checksums.txt` instead of a rebuild (interface Refinements).
- **Residual: Green** — after RC-3/RC-4 the checksums match by construction and a regression is caught pre-release.

### H-3 — Pre-release leaks into the tap
- **Severity: Medium** — users get an unstable but real build; recoverable by pinning/downgrading.
- **Probability: Low (after control)** — the authoritative gate is unambiguous.
- **Controls**: **RC-6** — job-level `if: ${{ !github.event.release.prerelease }}` as the authoritative stable gate, explicitly *not* GoReleaser's `skip_upload: auto` (which keys off a semver `-rc` suffix that can diverge from the GitHub pre-release flag) (plan Cross-cutting; interface § `tap` job; tasks T004).
- **Residual: Green**.

### H-4 — Tap job corrupts the GitHub release
- **Severity: High** — clobbering #30's notes or flipping pre-release/latest status damages the release record relied on by *all* channels (install-script, npm), not just brew.
- **Probability: Low** — the design forecloses it.
- **Controls**: **RC-7** — `release.disable: true` so the brew-publishing `goreleaser release` never creates/modifies the GitHub release (interface Refinements; tasks T003). **RC-8** — the `tap` job is `needs: [publish]` and independent; it runs after the release is fully published, and its failure does not roll back or alter the release (interface § Error Comm.; tasks T004).
- **Residual: Yellow** — accepted: the `gh release upload` boundary (DECISIONS 178) plus `release.disable` keep GoReleaser out of the release record; verify against a draft/pre-release before the first real run.

### H-5 — `brew audit` failure from a missing license
- **Severity: Medium** — blocks/warns acquisition on a fresh tap; no data or safety impact.
- **Probability: Medium (pre-control)** — the repo has no `LICENSE` today, so this *will* occur on the first real tap unless resolved.
- **Controls**: **RC-9** — resolve the licensing decision / add a `LICENSE` before the first stable release relies on the tap (tasks T002; the open interface `[NEEDS INPUT]`). **RC-10** — an optional `brew audit`/`brew style` CI step against the rendered formula surfaces audit failures before a release depends on them (interface testing strategy).
- **Residual: Green** — once RC-9 lands; until then this is the feature's most concrete open thread (correlates with checklist's open-input note and analyze H3).

### H-6 — 022 config refinements regress the release pipeline
- **Severity: High** — a broken `.goreleaser.yaml` would fail the landed build/verify/publish flow, blocking *all* distribution.
- **Probability: Low** — `release.disable` is inert in the build job (which uses `--skip=publish`), and `archives.mtime` only fixes timestamps; the config-guard catches drift.
- **Controls**: **RC-11** — extend the `internal/build` config-guard to assert `release.disable` is true and `archives.mtime` is pinned, alongside the existing matrix/sections checks (tasks T003). **RC-12** — the existing build→verify→publish jobs are unchanged and remain covered by PR validation (#24) and the landed 022 guards.
- **Residual: Yellow** — accepted: guarded by the config-drift test and exercised on the next release; a snapshot dry-run before merge confirms archives still build and verify.

---

## Residual Risk Summary

- **Red (unacceptable)**: none.
- **Yellow (accepted with justification)**: H-1 (token/supply-chain), H-4 (release-record corruption), H-6 (pipeline regression) — each bounded by least-privilege, the `gh`-publish boundary + `release.disable`, and the config-guard respectively.
- **Green (accepted)**: H-2, H-3, H-5 — fail-safe behavior plus pre-release controls.

The dominant theme is **supply-chain and pipeline integrity**: the new cross-repo write path (H-1) and the refinements to a landed, shared pipeline (H-4, H-6) carry the highest severity. All are reduced to acceptable by least-privilege scoping, keeping GoReleaser out of the release record, and the config-drift guard. The most actionable open item is RC-9 (the license, H-5).

---

## Traceability Index

**Hazards → source**
- H-1 → plan ADR-4 + Risk #1; interface Error Communication / Consistency Notes
- H-2 → plan Risk #3; interface "Refinements to 022's sections"; tasks T003
- H-3 → spec Non-Behaviors (no pre-release tracking); plan Risk #2; tasks T004
- H-4 → spec Non-Behaviors (no commit to this repo / no note authoring); plan ADR-3; interface Refinements
- H-5 → interface `[NEEDS INPUT]` (license); tasks T002
- H-6 → interface "Refinements to 022's sections" (supersedes 022 `keep-existing`); plan Consistency Notes

**Controls → grounding / implementing task**
- RC-1, RC-2 → ADR-4; T001 (token), T004 (env scoping)
- RC-3, RC-4, RC-5 → interface Refinements; T003
- RC-6 → plan Cross-cutting (stable-only gate); T004
- RC-7 → interface Refinements (`release.disable`); T003
- RC-8 → interface § `tap` job (`needs: [publish]`, independent failure); T004
- RC-9 → T002 (license decision); RC-10 → interface testing strategy (`brew audit` CI step)
- RC-11 → T003 (config-guard extension); RC-12 → existing 022 guards + #24

---

## Notes

- **Default acceptability matrix** used — PROJECT.md defines no project-level matrix. If distribution-channel risk warrants a stricter bar, define one in PROJECT.md.
- **First run** — no test-gap analysis (that is a re-run step). The `.feature` scenarios already cover H-2 (checksums-match), H-3 (pre-release-doesn't-move-tap), and the integrity hazards; H-1/H-4/H-6 are structural/CI hazards whose controls are best verified by the T004 workflow guard (the gap checklist VII and analyze K5 both flag) rather than by Gherkin.
