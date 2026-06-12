# Checklist: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, interface-cli.md, tasks.md, features/duplicated-setting-resolution/resolution-call-site-retrofit.feature
**Checks**: 8 (8 pass, 0 fail) — 4 principles produced no applicable checks (see Governance Notes)
**Generated**: 2026-06-12

---

## Summary

All 8 applicable checks pass. Constitution: 8/8.

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 5 | 5 | 0 |
| P1 (should fix) | 3 | 3 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **8** | **8** | **0** |

No done-* governance accords exist, so done-criteria and cross-reference categories generated no checks (see Governance Notes). Two advisory observations (non-blocking) are noted inline under II and V.

---

## Constitution Checks: 8/8 passed

### Calibration notes

Spec 040 is a behaviour-preserving refactor that migrates three existing resolvers (token/base URL/output format) onto 039's `internal/resolve`, plus one deliberate behaviour change: an explicitly-supplied empty/whitespace `--base-url`/`--output` now fails loud (presence semantics). It is read-only, makes no API call, no write, no governance mutation, and handles no result sets. Four principles concerned with writes, API limits, governance proposals, and pagination therefore produce no applicable checks. The remaining principles were calibrated to the retrofit's surface — the resolver internals, the preserved public types, and the operator-facing flag behaviour.

### Passed (8/8)

**P0** | CONSTITUTION.md I (Spec Fidelity): "Every command MUST map to a spec operation. The CLI MUST NOT invent endpoints, parameters, or behaviors the spec does not define."
→ **spec.md § System Overview, Non-Behaviors / interface-cli.md**: PASS — 040 adds no command, endpoint, or parameter. The `--base-url`/`--output` flags, env vars, and `.glassfrogrc` keys are pre-existing CLI conventions (008/020), unchanged. The flag-presence change is a CLI-layer resolution semantic, not an API-surface change; no operation, request shape, or parameter is altered.

**P0** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): "every error MUST explain what went wrong and the next step."
→ **interface-cli.md § Error Communication / interface-spec.md § Error Communication**: PASS — the new empty-flag failures reuse the existing typed errors verbatim (`*BaseURLError` names the source; `*FormatError` names the source, the offending value, and the supported set), routed through the same `classifyClientError` → `UsageError(2)` path that supplies the operator-facing cause-plus-next-step. Provenance `Origin` preserves the exact source labels. *(Observation, non-blocking: the `*BaseURLError` text names cause + source but no explicit remediation clause; this message is inherited unchanged from 008 and was accepted there — not a 040 regression. Any next-step enrichment is an upstream 008/031 concern.)*

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable, never hidden."
→ **spec.md § Behavioral Accord / plan.md ADR-2, ADR-3**: PASS — 040 actively strengthens fail-safe: an explicitly-supplied empty/whitespace flag is converted from a silent fall-through into a loud `UsageError(2)` (the presence change). Resolution errors (unreadable/malformed `.glassfrogrc`) surface verbatim with no fall-through; a present-but-invalid winner fails loud and is never silently superseded by a lower source. No swallowed errors, no partial state.

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts … adding a new command MUST NOT require changing unrelated ones."
→ **plan.md § ADR-1, ADR-4 / tasks.md**: PASS — the retrofit collapses three duplicated precedence chains onto one composable resolver (the canonical composition move), and each resolver stays an independently-testable adapter. Adding a future read command continues to read the shared persistent flags with no special work. *(Observation, non-blocking: the presence-bit threading touches every read-command `RunE` (13 today) + all the `resolveSelection` seam declarations (11 today). This is a one-time migration of a shared-flag contract — the established per-command seam / persistent-flag boundary, not new per-resource coupling — and the overload-free signature change is the deliberate compiler-enforced safeguard, T002/T003.)*

**P0** | CONSTITUTION.md XII (Standalone Executable): "no language runtime or … software that must be installed first."
→ **interface-spec.md § (resolver composition) / plan.md Cross-cutting (Layering)**: PASS — 040 adds only an import of `internal/resolve` (already compiled into the binary via 039) and changes no build or distribution surface. No new external dependency.

**P1** | CONSTITUTION.md IV (Test-Driven Development): "user-facing behavior MUST have an executable acceptance scenario before the code."
→ **features/…/resolution-call-site-retrofit.feature / tasks.md**: PASS — 13 acceptance scenarios (`@wip`) exist before implementation, including the user-facing flag-semantics change ("An explicitly empty base-URL flag is honoured by presence and fails loud"). Each task carries verifiable acceptance criteria, scenario references, and the "carry existing suite green" + new-test instruction. Correctly test-first.

**P1** | CONSTITUTION.md VII (Working Software): "no code-only or test-only increments … MUST validate and build."
→ **tasks.md**: PASS (at this stage) — each task pairs the retrofit change with acceptance criteria that require the existing suite green plus new edge-case/mapping tests; the atomic, overload-free signature change ensures each task compiles as a unit. Enforced concretely at implement/PR time; the decomposition does not invite code-only increments.

**P1** | CONSTITUTION.md VIII (No Fabricated Data): "MUST NOT invent, guess, or fill placeholder values."
→ **spec.md § Non-Behaviors / interface-spec.md § Provenance mapping**: PASS — the resolvers return the resolved value verbatim and map provenance from the real winning source; nothing is synthesized. The built-in default base URL / format is a caller-supplied constant (008/020), not fabricated API data — distinct from VIII's concern.

---

## Done-Criteria Checks

No `accords/governance/done-*.md` accords exist — done-criteria and cross-reference checks generated none.

---

## Cross-Reference Checks

No done-* accords — no cross-reference checks generated.

---

## Governance Notes

- **No done-* governance accords found.** Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria and cross-reference checks. (Carried over from prior specs — the repo has run constitution-only checklists throughout.)
- **Principles with no applicable checks for this feature** (read-only refactor, no result sets): VI (Size-Aware — no pagination/result sets), IX (Writes Require Explicit Intent — no writes), X (Respect API Limits — no API calls), XI (Governance via Proposals — no governance mutation).
- **Advisory observations** (non-blocking, recorded inline above): II — `*BaseURLError` carries no explicit remediation clause (inherited from 008, not a 040 regression); V — the presence-threading edit spans every read-command `RunE` (13 today) + the `resolveSelection` seam declarations (11 today) (one-time shared-contract migration, compiler-guarded).
