# Checklist: NPM Wrapper Package

**Feature**: 037-npm-wrapper-package
**Checked against**: CONSTITUTION.md (12 principles; done-* accords not present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/npm-wrapper-package.feature
**Checks**: 11 (11 pass, 0 fail)
**Generated**: 2026-06-12

---

## Summary

All 11 checks pass. Constitution: 11/11. (Done-criteria and cross-reference checks not run — no accords present.)

All 11 checks derive from constitution MUST / MUST NOT / NON-NEGOTIABLE principles, so every check is P0. Six principles (I, V, VI, IX, X, XI) produced zero applicable checks — they govern the CLI's Glassfrog-API command surface, which this npm packaging channel has none of (see Governance Notes).

> **Resolved in-session (2026-06-12, post-guard).** Two P0 findings were raised on first evaluation and fixed before implementation — they mirrored the install-script (027) sibling exactly:
> - ~~**P0 | II. Action Transparency** — interface-spec.md § Error Communication committed some errors to a cause but no next step.~~ → **Fixed**: the Error Communication table now names a next step for every install-time error (unsupported → use a supported platform / 027 / 036; mismatch / download failure → re-run to retry; missing `tar` → install it or use 027).
> - ~~**P0 | VII. Working Software** — tasks.md T001/T002/T003 were source-only PRs with tests deferred to T004.~~ → **Fixed**: T001–T003 each now ship their own `node --test` units (launcher passthrough; postinstall fixture-server; generator package-shape), so no PR is test-free; T004 is now CI wiring + cross-unit integration only.

---

## Constitution Checks: 11/11 passed

### Passed (11/11)

- **II. Action Transparency** (error next-step, resolved): every install-time error in interface-spec.md § Error Communication now names both a cause and a next step (non-negotiable II satisfied). **Pass.**
- **VII. Working Software** (per-PR tests, resolved): each Phase-1 source task (T001–T003) ships its own unit tests in the same PR; T004 adds only CI wiring + integration coverage. No code-only increment remains. **Pass.**

- **II. Action Transparency** (success/action reporting): the postinstall reports what it placed and from where (interface-spec.md § Observability via plan; plan § Cross-cutting Concerns), and the launcher is a transparent pass-through that forwards the binary's own output unmodified (interface-spec.md § Interactions — Runtime flow). **Pass.**
- **III. Fail Safe, Not Silent** (3 checks): (a) validate-before-place — the fallback verifies sha256 before placing any binary (plan ADR-3; interface-spec.md § Interactions); (b) no partial state — atomic temp→verify→move, nothing placed on any failure (plan ADR-3; spec § Behavioral Accord "leaves no runnable command"); (c) no failure-as-success — every install-time failure exits non-zero and fails the install (interface-spec.md § Error Communication). **Pass.**
- **IV. Test-Driven Development** (2 checks): (a) user-facing behavior has executable acceptance scenarios authored before the code — npm-wrapper-package.feature, 12 scenarios, all `@wip` (the pre-implementation RED layer); (b) the decomposition includes test work (T004 `node --test`, plus the `@validation` hold-outs). **Pass.** (The PR-granularity concern is captured under VII above.)
- **VIII. No Fabricated Data**: the launcher fabricates nothing (pure pass-through of the binary's output), and the reported version is the real package/tag version, not a synthesized value — spec § Behavioral Accord (version coupling), interface-spec.md § Interactions (Version normalisation). **Pass.**
- **XII. Standalone Executable** (2 checks): (a) the self-contained Go binary remains self-contained — the npm channel bundles/downloads it verbatim and adds no dependency to the binary itself (plan § System Architecture; § What This Plan Does Not Cover); (b) the channel's Node reliance is scoped to this opt-in route for Node-based environments — the sibling channels (install script 027, Homebrew 036, direct download) install and run the same binary with no language runtime, so XII's "runs on host-OS-plus-network alone" holds for the canonical artifact (plan ADR-2 / Risk R5; spec § System Overview). **Pass — see Governance Notes for the runtime-Node nuance.**

---

## Done-Criteria Checks: not run

No `accords/governance/done-*.md` accords are present in the repository, so no done-criteria checks were generated. See Governance Notes.

---

## Cross-Reference Checks: not run

Cross-reference checks derive from done-* accords that require inter-artifact references; with none present, none were generated. The artifacts do carry traceability (tasks.md references the feature file, interface, and plan; interface-spec.md cites 022's name-template contract and the 027 precedent) — horizontal consistency of those links is `/score:analyze`'s domain.

---

## Governance Notes

- **No `accords/governance/` directory.** done-specify.md, done-plan.md, done-interface.md, done-scenarios.md, done-tasks.md are all absent. Consider creating `accords/governance/done-<skill>.md` for each to enable done-criteria and cross-reference quality checks. Until then, checklist coverage for this project is constitution-only (consistent with 027).
- **XII runtime-Node nuance (observation, not a finding).** The npm-installed `glassfrog` command is a Node `bin` shim, so invoking it re-enters Node at runtime (not only at install) — unlike the install-script channel, where the bare binary runs Node-free after install. This is acceptable under XII because: (1) the npm channel exists *specifically* to serve Node-based environments that already have Node (spec § System Overview); (2) it adds no dependency to the binary, which stays self-contained and is installable+runnable with no runtime via 027/036/direct download; and (3) XII's rationale is that the CLI must run where operators need it without forced installs — the npm channel forces Node on no one. Surfaced for transparency; the project already sanctioned this channel (BACKLOG #37, FEATURE-MODEL Self-Contained Distribution).
- **Zero-applicable constitution principles** (governance surface this feature doesn't touch):
  - **I. Spec Fidelity** — no applicable checks: the npm channel invokes no Glassfrog API v5 operation; it consumes release artifacts (GitHub Releases) and publishes to the npm registry, which the spec contract does not govern.
  - **V. Composition over Monolith** — no applicable checks: the feature adds no per-resource command module; it is a packaging artifact set. (Its internal launcher/postinstall/generator decomposition is a plan concern, not the principle's command-module scope.)
  - **VI. Size-Aware by Design** — no applicable checks: no API result-set pagination or org-tree traversal is involved.
  - **IX. Writes Require Explicit Intent** — no applicable checks: the channel is not a read-shaped Glassfrog command; installing onto disk is its explicit, sole purpose.
  - **X. Respect API Limits** — no applicable checks against the Glassfrog API: the channel never calls it. (Related but non-constitutional: the fallback uses deterministic release-asset URLs, avoiding the GitHub API, mirroring 027.)
  - **XI. Governance via Proposals** — no applicable checks: the channel performs no governance-structure mutation.
