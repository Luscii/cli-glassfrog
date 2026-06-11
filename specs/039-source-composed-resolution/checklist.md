# Checklist: Source-Composed Resolution

**Feature**: 039-source-composed-resolution
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/duplicated-setting-resolution/source-composed-resolution.feature
**Checks**: 8 (8 pass, 0 fail) — 4 principles produced no applicable checks (see Governance Notes)
**Generated**: 2026-06-11

---

## Summary

All 8 applicable checks pass. Constitution: 8/8.

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 5 | 5 | 0 |
| P1 (should fix) | 3 | 3 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **8** | **8** | **0** |

No done-* governance accords exist, so done-criteria and cross-reference categories generated no checks (see Governance Notes).

---

## Constitution Checks: 8/8 passed

### Calibration notes

Spec 039 is an internal, in-process mechanism (the `internal/resolve` package) with no command, no API call, no writes, no governance mutation, and no operator-facing output of its own (the 040 call sites consume it). Four runtime-behaviour principles therefore produce no applicable checks for this feature; the remaining principles were calibrated to the mechanism's surface.

### Passed (8/8)

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts … adding a new command MUST NOT require changing unrelated ones."
→ **plan.md § ADR-1 / tasks.md**: PASS — the feature is a new `internal/resolve` leaf importing only `internal/rcfile`, purely additive ("no existing file is edited"; the three current resolvers are untouched until 040). The composable-source design is itself modular composition.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "a read-shaped command MUST NEVER mutate as a side effect."
→ **spec.md § Non-Behaviors**: PASS — explicit non-behaviour: "must not write any file or environment, and must not make a network call." The resolver is read-only by contract.

**P0** | CONSTITUTION.md XII (Standalone Executable): "no language runtime or … software that must be installed first."
→ **interface-spec.md § Package / OS binding**: PASS — the package uses only the standard library, `internal/rcfile`, and `golang.org/x/term` (already compiled into the binary, used by `authlogin_seam.go`). No new runtime dependency.

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable, never hidden."
→ **spec.md § Behavioral Accord (Resolution errors) / plan.md ADR-3, ADR-5**: PASS — a source read error aborts the walk and is surfaced verbatim with no fall-through; a present-but-invalid high-precedence value still wins (not silently superseded by a lower source); a multi-stdin composition panics loudly. No swallowed errors.

**P0** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): "every action … report … in machine-parseable form; every error MUST explain what went wrong and the next step."
→ **interface-spec.md § Types (Provenance) / Error Communication**: PASS — the resolver returns `Provenance{Kind, Origin}` so the caller can report which source supplied a value, and resolution errors name the offending origin. The mechanism carries (does not obstruct) the transparency the operator-facing 040 sites will surface.

**P1** | CONSTITUTION.md IV (Test-Driven Development): "user-facing behavior MUST have an executable acceptance scenario before the code."
→ **features/…/source-composed-resolution.feature / tasks.md**: PASS — acceptance scenarios (11, `@wip`) exist before any implementation; every task carries concrete, verifiable acceptance criteria and scenario references. The pre-implementation state is correctly test-first.

**P1** | CONSTITUTION.md VII (Working Software): "no code-only or test-only increments … MUST validate and build."
→ **tasks.md**: PASS (at this stage) — each task pairs a behavioural concern with verifiable acceptance criteria, setting up code+test increments. Enforced concretely at implement/PR time; the decomposition does not invite code-only increments.

**P1** | CONSTITUTION.md VIII (No Fabricated Data): "MUST NOT invent, guess, or fill placeholder values."
→ **spec.md § Non-Behaviors / plan.md ADR-3**: PASS — the resolver returns the resolved value verbatim and synthesizes nothing. (The `Default(value)` backstop is a caller-supplied built-in constant, not fabricated API data — distinct from VIII's concern.)

---

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords exist in this repository.

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent.

---

## Governance Notes

- **Principles with no applicable checks for this feature** (an internal mechanism, not a command):
  - **I (Spec Fidelity)** — 039 adds no command and makes no API call; nothing to diff against `spec.yaml`.
  - **VI (Size-Aware)** — no API result sets or pagination in a resolver. *(Observation for analyze: the interface notes a bounded stdin read (`maxStdinBytes`) but neither spec nor interface specifies behaviour when piped input exceeds the cap — a consistency gap better suited to analyze than a constitution check.)*
  - **X (Respect API Limits)** — no API calls, no updates, no `If-Match`/`ETag`.
  - **XI (Governance via Proposals)** — no governance-structure mutation.
- **Missing done-* accords**: `accords/governance/` contains no `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, or `done-tasks.md`. Consider creating them to enable done-criteria and cross-reference checks. This gap affects every spec in the repo, not just 039.
