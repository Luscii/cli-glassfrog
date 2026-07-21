# Risk: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Round**: 1
**Date**: 2026-07-21
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md declares no Regulatory Context (this is a developer CLI, not a regulated device)

---

## Summary

8 hazards, 8 controls. No unacceptable (Red) residual risks. Two Yellow residuals worth the developer's attention: **H-3** (the setup skill's missing-CLI guidance could nudge a runtime-less environment toward the Node-dependent npm channel — the checklist's P2, now grounded as a hazard) and **H-5** (the Claude marketplace host's manifest schema could evolve out from under the committed file — external, plan-acknowledged, only partly controllable). Everything else reduces to Green, mostly via the consistency guard and the setup skill's re-check-before-ready discipline.

The feature's blast radius is intrinsically small: it distributes knowledge and provisions an environment, does no governance write, and touches no live record. The worst realistic outcome is "the operating surface won't install / won't come up," never "governance was changed wrongly."

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Level | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Marketplace entry drifts from the plugin (name/source/description mismatch) → install fails or resolves wrong | spec Consistency accord; interface Error Comm | Med | Low | Green | RC-1 | Green |
| H-2 | Setup reports "ready" while the CLI is absent or unauthenticated (false-ready) → agent runs commands that fail | spec Setup accord; scenario "re-checks after a fix" | Med | Low | Green | RC-2 | Green |
| H-3 | Missing-CLI guidance steers a runtime-less environment to the npm (Node) channel → CLI can't run, defeating XII | checklist Obs 2; interface Missing-CLI fix | Med | Med | Yellow | RC-3 | Yellow |
| H-4 | Auth check misjudges credential validity (false pos/neg) → operator proceeds without working creds | interface Auth check; plan ADR-4 | Low | Low | Green | RC-4 | Green |
| H-5 | Claude host's marketplace manifest schema evolves → committed manifest no longer accepted, install breaks | plan Risk 1; integration boundary (host) | Med | Med | Yellow | RC-5 | Yellow |
| H-6 | Setup skill's enumerable facts (channel names, auth leaf) drift from README/CLI → operator sent to a dead channel/command | plan Risk 2; interface Guard contract | Med | Low | Green | RC-6 | Green |
| H-7 | Setup and orientation give contradictory credential guidance → operator confused | plan Risk 3 | Low | Med | Green | RC-7 | Green |
| H-8 | A later sibling-plugin entry is added malformed → whole marketplace fails to load, breaking the shipped glassfrog install too | spec generality edge case | Med | Low | Green | RC-8 | Green |

---

## Hazard Detail

**H-1 — Marketplace/plugin drift.** *Severity Med*: a broken entry defeats the whole capability (nothing installs), but no governance data is touched. *Probability Low*: the entry and plugin ship in one repo, so drift only appears via an editing mistake. **RC-1**: the ADR-5 consistency guard resolves the entry's `source`, `name`, and `description` against `plugin.json` at test time (both sides derived) and fails CI on any mismatch or a stray `version` key. Residual **Green**.

**H-2 — False-ready.** *Severity Med*: the agent would run commands that then fail — recoverable but wasteful. *Probability Low*: the journey re-checks after each fix and gates the ready report on both checks passing. **RC-2**: presence check → fix → re-check loop, with "ready" reachable only when both presence and auth checks pass (scenario "Setup re-checks after a fix instead of assuming success"). Residual **Green**.

**H-3 — Node channel in a runtime-less environment.** *Severity Med*: the operator installs a channel that can't run where XII says the CLI must (no runtime), then has to backtrack. *Probability Med*: the interface currently lists the three channels flat, "sourced from README," with no default ordering — an agent may pick the first or the familiar one. **RC-3**: present the zero-dependency channels (install script, Homebrew) as the default and the npm wrapper as the Node-environment option. *This control is recommended, not yet encoded* — it lands when T003 writes the skill. Residual **Yellow** (acceptable with the recommendation tracked; drops to Green once the ordering is encoded and the enumerable-facts guard covers the channel list). This is the checklist's P2 Observation 2 promoted to a tracked hazard.

**H-4 — Auth-check misjudgment.** *Severity Low*: a wrong verdict is self-correcting — the next real command fails legibly via the CLI's own exit codes. *Probability Low*: the check is a real authenticated identity read (`me`), not a heuristic. **RC-4**: verify auth by observing the CLI's own identity-read outcome; introduce no separate credential validation that could disagree with the CLI (plan ADR-4). Residual **Green**.

**H-5 — Host schema evolution.** *Severity Med*: install breaks for new adopters until the manifest is updated. *Probability Med*: external and outside our control; Claude Code's marketplace format can change. **RC-5**: keep the manifest minimal (name, owner, one entry) to shrink the exposed surface; verify the exact field set against current host docs at implementation; the guard checks internal consistency so a host-schema change surfaces as a doc-check, not a silent break. Residual **Yellow** (external dependency, plan-acknowledged Risk 1 — cannot be driven to Green by anything in this repo).

**H-6 — Enumerable-fact drift.** *Severity Med*: the operator is sent to a channel or command that no longer exists. *Probability Low*: the facts change rarely and defer detail to CLI help. **RC-6**: the best-effort guard anchors the named channels and the auth-check leaf to README/the CLI registry (interface Guard contract). Residual **Green**.

**H-7 — Setup/orientation contradiction.** *Severity Low*: conflicting credential guidance is confusing but not harmful. *Probability Med* at authoring time. **RC-7**: the explicit journey-vs-reference boundary — setup owns check→fix→verify, orientation owns the credential/exit-code reference; setup links and never restates (plan Risk 3, interface Boundary note). Residual **Green**.

**H-8 — Malformed sibling entry.** *Severity Med*: a bad future entry could make the host reject the whole manifest, taking the working glassfrog install down with it. *Probability Low*: adding an entry is a deliberate, reviewed change. **RC-8**: the list-shaped manifest plus the guard's existence-based lookup (asserts the glassfrog entry exists and resolves — not "is the only entry") means the guard still validates the glassfrog entry when siblings are added, and re-runs on any manifest change. Residual **Green**.

---

## Residual Risk Summary

- **Red (unacceptable)**: none.
- **Yellow (acceptable, tracked)**: H-3 (encode channel ordering in T003 — then re-evaluate to Green), H-5 (external host-schema dependency — verify field set at implementation, monitor host docs).
- **Green (accepted)**: H-1, H-2, H-4, H-6, H-7, H-8.

---

## Traceability Index

| ID | Grounding |
|---|---|
| H-1 | spec § Behavioral Accord (Consistency); interface § Error Communication; plan ADR-5 |
| H-2 | spec § Behavioral Accord (Setup skill); feature "Setup re-checks after a fix" |
| H-3 | checklist Observation 2; interface § Required sections (Missing-CLI fix); CONSTITUTION XII |
| H-4 | interface § Required sections (Auth check); plan ADR-4 |
| H-5 | plan § Risks (Risk 1); spec § Integration Boundaries (plugin host) |
| H-6 | plan § Risks (Risk 2); interface § Guard contract |
| H-7 | plan § Risks (Risk 3); interface § Required sections (Boundary note) |
| H-8 | spec § Driving Scenarios (edge: second plugin); interface § Interactions (second-plugin flow) |
| RC-1 | plan ADR-5 (consistency guard, both sides derived) |
| RC-2 | interface Setup journey (re-check-before-ready) |
| RC-3 | recommended — T003 / interface Missing-CLI fix ordering |
| RC-4 | plan ADR-4 (CLI-owned auth verification) |
| RC-5 | plan Risks Risk 1 mitigation (minimal manifest + doc-check) |
| RC-6 | plan ADR-5 (enumerable-fact anchoring) |
| RC-7 | plan Risks Risk 3 mitigation (journey/reference boundary) |
| RC-8 | plan ADR-2 (list shape) + ADR-5 (existence-based guard) |

---

## Governance Notes

- Using the default risk acceptability matrix — no project-level matrix found in PROJECT.md.
- **guardian-agent.md not resolvable at the skill-relative path** — risk ran on SKILL.md process alone (reduced character consistency, not a blocked skill).
