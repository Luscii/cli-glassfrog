# Checklist: Homebrew Tap

**Feature**: 036-homebrew-tap
**Checked against**: CONSTITUTION.md (12 principles; done-* accords not present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/homebrew-tap.feature
**Checks**: 9 (9 pass, 0 fail)
**Generated**: 2026-06-12

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 9 | 9 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **9** | **9** | **0** |

All 9 checks derive from constitution MUST / MUST NOT / NON-NEGOTIABLE principles, so every check is P0. Seven principles (I, II, V, VI, IX, X, XI) produced zero applicable checks — they govern the CLI's Glassfrog-API command surface and operator-facing output, which this distribution channel does not add (the user surface is Homebrew's). See Governance Notes.

---

## Constitution Checks: 9/9 passed

> **Resolution trace**: the initial pre-implementation guard pass flagged a P0 against **VII Working Software** — T004 added the `tap` job to `release.yml` with no test. That gap was closed in the same PR by extending T004's scope to ship the `internal/build` workflow-structural guard alongside the workflow change (see VII below). This artifact reflects the resolved decomposition.

### Passed (9/9)

- **VII. Working Software** (1 check): every code/config-changing task ships its tests in the same PR — T003 ships the config-guard extension and offline render test; **T004 now extends the `internal/build` workflow guard** (the `CheckReleaseWorkflow`/`CheckVerifyGate` family) to assert the `tap` job's contract (`needs: [publish]`, the `if: !prerelease` gate, the `HOMEBREW_TAP_TOKEN` env, brew-publisher-only), mirroring 022's guard-as-proxy. No task is planned as a code-only increment. **Pass.**
- **III. Fail Safe, Not Silent** (3 checks): (a) integrity-before-install — spec.md § Behavioral Accord ("verifies it against the recorded checksum … no binary is placed" on mismatch) and the driving + validation integrity scenarios; (b) no partial/cross-contaminated state — interface-spec.md § Error Communication: a failed `tap` job leaves the already-published release unaffected, and a missing asset fails the install clearly rather than placing a wrong binary; (c) no failure-as-success — interface Error Communication maps a missing/expired token to a non-zero (loud red) `tap` run, never a silent skip. **Pass.**
- **IV. Test-Driven Development** (2 checks): (a) user-facing behavior has executable acceptance scenarios authored before the code — homebrew-tap.feature, 12 scenarios, all `@wip` (the pre-implementation RED layer); (b) the decomposition includes test work — T003 ships the config-guard extension and the offline render-and-inspect test in the same PR. **Pass.**
- **VIII. No Fabricated Data**: the formula records only real release data — interface-spec.md § "Generated Formula structural contract" pins each `url` to an attached release asset and each `sha256` to its `checksums.txt` entry (hard contract); the reproducibility requirement (T003, pinned `archives.mtime`) ensures the recorded checksums equal the published bytes rather than a divergent rebuild. No guessed/placeholder values. **Pass.**
- **XII. Standalone Executable** (2 checks): (a) the installed artifact remains the self-contained binary — spec.md § Behavioral Accord and plan ADR-1 install the pre-built release binary with no Go toolchain or source compile (`bin.install`), the non-behavior explicitly forbidding a source build; (b) Homebrew's own presence as the acquisition tool is scoped to the channel and is not a XII concern — XII governs the distributed binary's runtime, and the 021/022 build-host-vs-artifact distinction (DECISIONS) treats acquisition channels as host tooling. **Pass.**

---

## Done-Criteria Checks: not run

No `accords/governance/done-*.md` accords are present in the repository, so no done-criteria checks were generated. See Governance Notes.

---

## Cross-Reference Checks: not run

Cross-reference checks derive from done-* accords that require inter-artifact references; with none present, none were generated. The artifacts do carry traceability (tasks.md references the feature file, interface, and plan; interface-spec.md cites spec 022's contract) — horizontal consistency of those links is `/score:analyze`'s domain.

---

## Governance Notes

- **No `accords/governance/` directory.** done-specify.md, done-plan.md, done-interface.md, done-scenarios.md, done-tasks.md are all absent. Consider creating `accords/governance/done-<skill>.md` for each to enable done-criteria and cross-reference quality checks. Until then, checklist coverage for this project is constitution-only. (Same gap as 027.)
- **Zero-applicable constitution principles** (surface this distribution channel doesn't touch):
  - **I. Spec Fidelity** — no applicable checks: the channel invokes no Glassfrog API v5 operation; it publishes a Homebrew formula referencing GitHub release artifacts, which the spec contract does not govern.
  - **II. Action Transparency** — no applicable checks: the channel adds no glassfrog CLI action or operator-facing output of our own. The user interacts with Homebrew's commands (`brew install`/`upgrade`), whose output and error transparency Homebrew owns; the `version` output a `brew install` yields is governed by 003, not here. (This is where 036 differs from 027, whose installer output is authored in-repo.)
  - **V. Composition over Monolith** — no applicable checks: the feature adds no per-resource command module; it is config (`.goreleaser.yaml`) plus a CI job. The two refinements to 022's `release:`/`archives` sections are within the same distribution-tooling concern, not edits forced onto unrelated command modules.
  - **VI. Size-Aware by Design** — no applicable checks: no API result-set pagination or org-tree traversal is involved.
  - **IX. Writes Require Explicit Intent** — no applicable checks: no CLI read/write command is added. The `tap` job's push to the tap repo is CI publishing triggered by an explicit release publish, not a read-shaped CLI command mutating as a side effect.
  - **X. Respect API Limits** — no applicable checks against the Glassfrog API: the channel never calls it.
  - **XI. Governance via Proposals** — no applicable checks: the channel performs no governance-structure mutation.
- **Open input (not a constitution check)**: interface-spec.md carries an unresolved `[NEEDS INPUT]` — the repo has no `LICENSE`, which `brew audit` needs. Tracked as tasks.md T002; surfaced here for visibility, not as a check failure.
