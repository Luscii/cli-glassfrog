# Checklist: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Checked against**: CONSTITUTION.md (I–XII)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/operating-surface-packaging.feature, tasks.md
**Checks**: 9 applicable (9 pass, 0 fail) · 3 not applicable (VI, VIII, X)
**Generated**: 2026-07-21

---

## Summary

All 9 applicable constitution checks pass. Three principles (VI Size-Aware, VIII No Fabricated Data, X Respect API Limits) produce no applicable checks for this feature — packaging renders no API result set, fabricates no data, and opens no new API-call path — and are recorded with their reason rather than forced into a pass. Done-criteria: not run (no `accords/` directory). Cross-references: folded into done-criteria (not run).

Two observations are recorded below the check results. Observation 1 (setup-skill output form vs Action Transparency) carries the prior paths' disposition. Observation 2 is new and the one finding worth a developer's attention before build: the missing-CLI fix lists three install channels, one of which (npm) reintroduces a Node dependency — recommend framing the zero-dependency channels as the default so the skill's own guidance stays aligned with XII.

---

## Constitution Checks: 9/9 applicable passed

### Passed (9/9 applicable)

- **I. Spec Fidelity** (P0) — Packaging invents no command, endpoint, parameter, or API behavior: it is distribution only. spec.md Non-Behaviors ("must not add any API capability, command, or flag"), plan ADR-4 (checks are instructed knowledge — explicitly *not* a `glassfrog doctor` command or a shipped script), interface-spec Surface (no runtime entry point; no `glassfrog` subcommand added), and the validation scenario "Packaging adds no operating surface of its own". The setup skill's checks reuse commands the CLI already exposes (`--version`, `me`).
- **II. Action Transparency** (P0) — The setup skill's two failure classes each carry a cause and an actionable next step (missing binary → the named install channels; failing auth → the CLI's `X-Auth-Token` setup), kept distinct end-to-end; the underlying `--version`/`me` commands retain the CLI's machine-parseable output; the consistency guard reports *which* identity field drifted on failure. spec Behavioral Accord (Setup skill) + interface Required sections + Error Communication. (See Observation 1 on the *form* of the ready/not-ready report.)
- **III. Fail Safe, Not Silent** (P0) — Marketplace↔plugin drift is surfaced loudly (guard test red in CI), never silently tolerated (ADR-5, interface Guard contract, spec "defect, not difference" accord); the setup journey re-checks after a fix and never reports ready off a still-failing check (scenario "Setup re-checks after a fix instead of assuming success"); there is no multi-step write, so no partial-apply state is reachable.
- **IV. Test-Driven Development** (P0) — 13 acceptance scenarios exist `@wip` in the feature file before implementation (9 non-`@validation` + 4 `@validation`); tasks T001 and T003 reference the scenarios they un-`@wip`, and each ships its guard test alongside the artifact it guards.
- **V. Composition over Monolith** (P0) — Additive: a new repo-root `.claude-plugin/marketplace.json`, a new `plugin/skills/glassfrog-setup/` skill, and a new `internal/build/operatingsurfacepackaging.go`; plan "What This Plan Does Not Cover" pins nothing in 062–069's skills, agents, or hook modified, and ADR-3 keeps the closed operator-path agent family closed. Adding a future sibling plugin is one appended entry (scenario "A sibling plugin is one appended entry"), not a restructure.
- **VII. Working Software** (P0) — No code-only/test-only increment: T001 ships the manifest with its consistency guard; T003 ships the skill with its enumerable-facts guard and BDD coverage. T002 is documentation only (prose, no code) — outside the code+tests pairing by nature, not a violation of it.
- **IX. Writes Require Explicit Intent** (P0) — The setup skill's presence and auth checks are read-only diagnostics that issue no write or mutation as a side effect (ADR-4; T003 acceptance criterion "never installs a binary or stores a credential of its own"); the marketplace manifest and guard touch no live governance at all. Packaging adds no write path.
- **XI. Governance via Proposals** (P0) — Packaging exposes no governance-structure mutation and no bypass: it distributes a plugin and provisions an environment, both entirely outside the `/proposals` flow. No default write path, no opt-in escape hatch, nothing to gate.
- **XII. Standalone Executable** (P0) — Adds no runtime dependency to the CLI binary; the CLI stays self-contained and is *not* installed or bundled by packaging (spec Non-Behaviors, plan "does not cover CLI distribution"). The marketplace manifest and setup skill are host-consumed plugin artifacts; the guard is a build-time Go test. (See Observation 2 — the setup skill's *guidance* about install channels touches XII's spirit even though the CLI's standalone-ness is unchanged.)

### Not applicable (3 — zero checks, with reason)

- **VI. Size-Aware by Design** — Packaging renders no API result set and drives no paginated read of its own; the setup auth check is a single low-cost identity read whose paging is the CLI's concern (016). No truncation surface exists here. No applicable checks.
- **VIII. No Fabricated Data** — Packaging presents no API-returned data to fabricate or default; the auth check reads through the CLI, which owns data fidelity. No applicable checks.
- **X. Respect API Limits** — Packaging opens no new API-call path; the auth check rides the existing client's rate-limit/backoff handling (017). Optimistic concurrency is N/A (no field edits). No applicable checks.

---

## Observations (not failing checks)

1. **Setup-skill output form vs. Action Transparency (II)** — *open, accepted*: the ready/not-ready report and the fix guidance are instructional prose an LLM agent consults, not a machine-parseable serialization. Traceability is preserved by naming the exact cause and the exact next command, so II passes; whether the report should *also* be structured is a design choice, accepted for now (the consumer is an agent reading a skill). Same disposition as 064–069's Observation 1.
2. **Missing-CLI fix lists a Node-dependent channel among the defaults (XII)** — *new, advisory (P2)*: the interface's missing-CLI fix names three install channels — install script, Homebrew tap, npm wrapper — "sourced from README," with no stated ordering. The npm wrapper (037) reintroduces a Node runtime, which is exactly the dependency XII exists to avoid assuming. XII is not violated (the CLI binary stays standalone regardless of channel, and offering npm to those who have Node is legitimate), but the *skill's own guidance* would align better with XII if it presented the zero-dependency channels (install.sh / Homebrew) as the default and the npm wrapper as the Node-environment option. Recommend T003/interface encode that ordering rather than a flat list. Verified against: interface-spec "Missing-CLI fix" + spec Behavioral Accord. This is the intended enforcement point for the observation — carry it to analyze/validate.

---

## Governance Notes

- **No `accords/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not run; there is no source for them in this repo. This matches every prior spec in the project (constitution-only checking). Consider creating `accords/governance/done-*.md` to enable done-criteria checks across the pipeline.
- **guardian-agent.md not resolvable at the skill-relative path** — checklist ran on SKILL.md process alone (reduced character consistency, not a blocked skill), as the SKILL.md fallback permits.
