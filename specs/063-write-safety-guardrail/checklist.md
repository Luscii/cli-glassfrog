# Checklist: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/unequipped-agent-operators/write-safety-guardrail.feature
**Checks**: 13 (11 pass, 2 fail)
**Generated**: 2026-06-16

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 8 | 8 | 0 |
| P1 (should fix) | 3 | 2 | 1 |
| P2 (consider) | 2 | 1 | 1 |
| **Total** | **13** | **11** | **2** |

---

## Post-Triage Resolution (PR #149)

Both failing checks were addressed during PR #149 review triage (Copilot comments `r3421226233`, `r3421226275`); the findings below are preserved as the original point-in-time assessment:

- **P1 (II/IX conflict-resolution)** → resolved: plan.md ADR-1 now carries an explicit **Constitution reconciliation** note — the gate is operator-layer, the CLI's command-is-intent / non-interactive contract is untouched, and an amendment is required only if the resolution is read as binding on the whole operating surface.
- **P2 (hook runtime)** → resolved: tasks.md T001 now pins the gate script's runtime to `bash` (the interpreter the `hooks.json` `command` invokes).

---

## Constitution Checks: 11/13 passed

### Failures

**P1** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE) + IX (Writes Require Explicit Intent), "When Principles Conflict" row 2: *"Intent is expressed by the explicit write command/flag itself, not by an interactive prompt … the command's existence is the intent, so an agent can act without a human in the loop."*
→ **spec.md § Behavioral Accord / plan.md § ADR-2 / interface-spec.md § Surface**: The guardrail introduces a mandatory **human-in-the-loop** interactive confirmation (`permissionDecision:"ask"`) for governance writes — the direct inverse of the constitution's documented resolution that intent is the command itself and an agent acts *without* a human in the loop. The artifacts reconcile this with VISION principle 2 and DECISIONS §399 (the **CLI** stays non-interactive; the gate is operator-layer), which is a sound substantive reconciliation — but **none of the artifacts cite or reconcile against this constitutional resolution**. Since the constitution "supersedes conflicting practices," the reconciliation must be explicit: state that the gate lives at the operator/host layer (not the CLI, whose non-interactive command-is-intent contract is preserved per §399), so the constitution's CLI-focused resolution is not breached. If the team instead reads that resolution as binding on the whole operating surface, this escalates to P0 and a constitution amendment (version bump) is required. *Recommend the Shaper add an explicit reconciliation note to plan ADR-1/ADR-2.*

**P2** | CONSTITUTION.md XII (Standalone Executable): *"no language runtime or interpreter … no other software that must be installed first."*
→ **plan.md § Cross-cutting / interface-spec.md § Surface**: XII binds the **CLI executable**, which 063 leaves untouched (no Go CLI code) — so XII is **not** violated. But the gate **hook script's runtime** (bash / python3 / node) is left open ("language chosen at interface/implementation time for portability"), and the project's no-runtime-dependency value extends in spirit to the operator layer. *Recommend: pick a runtime the plugin host reliably provides (installed examples use `bash` and `python3`) and document the assumption in tasks T001/T002, rather than leaving it unresolved.*

### Passed (11/13)

- **P0 | I (Spec Fidelity)** — spec.md non-behaviors + plan ADR-1/ADR-3: the guardrail adds no command, flag, or API capability; it gates only existing CLI commands. No invented surface.
- **P0 | II (Action Transparency, binding text)** — the hook emits structured JSON (`permissionDecision`) and a `systemMessage` naming command/target/effect; the CLI's own machine-parseable output and cause+next-step errors are unchanged. (The conflict-resolution *tension* is the P1 above; the binding MUST is met.)
- **P0 | III (Fail Safe, Not Silent)** — plan ADR-3/ADR-4 + interface Error Communication: fail-closed within the `proposal` namespace, fail-safe on parse error (never blocks unrelated Bash), and the drift tripwire prevents a silently-ungated write. No swallowed errors.
- **P0 | IV (TDD / BDD acceptance)** — 16 `@wip` acceptance scenarios exist (write-safety-guardrail.feature) before implementation; tasks T002/T003 carry concrete acceptance criteria and scenario references.
- **P0 | V (Composition over Monolith)** — plan: 063 is purely additive (new hook + registry + drift test); it modifies no existing command and does not edit 062's orientation skill.
- **P0 | VIII (No Fabricated Data)** — interface: the `systemMessage` is derived from the parsed command string (command, target id, effect); it invents no API data.
- **P0 | IX (Writes Require Explicit Intent, binding text)** — reads and tension edits pass ungated; the gate adds confirmation to explicit write commands and makes no read mutate. (Automation tension captured in the P1 above.)
- **P0 | XI (Governance via Proposals)** — the guardrail gates the sanctioned proposal write path and exposes no direct-mutation bypass; it is orthogonal to the `role-manage-without-proposal` escape hatch.
- **P1 | X (Respect API Limits)** — 063 *reinforces* optimistic concurrency: the no-blind-retry rule enforces the re-read-then-If-Match cycle (054) rather than clobbering on a `412`.
- **P1 | VII (Working Software)** — tasks bundle implementation with tests (T002 recognizer + its unit tests; T003 is itself the drift test); no code-only/test-only split implied.
- **P2 | III/VIII (systemMessage fidelity)** — interface constrains the confirmation message to the command's own data; no fabricated resource detail.

---

## Governance Notes

- **No `accords/governance/` directory found** — all done-* accords are absent. Done-criteria and cross-reference checks were **not run**; this checklist is constitution-only.
  - Consider creating `accords/governance/done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`, and `done-specify.md` to enable vertical done-criteria checks for future specs.
- **Constitution principle VI (Size-Aware)**: no applicable checks for this feature — the guardrail handles no result sets or pagination. Zero checks (not a gap).
- **Severity calibration note**: the headline P1 (II/IX conflict-resolution) would be **P0** if the team treats the "When Principles Conflict" resolution as binding on the operating surface rather than CLI-scoped. The Guardian assessed it P1 because the binding MUST text of II and IX is not breached and the CLI's non-interactive contract is preserved; the developer owns the escalation call.
