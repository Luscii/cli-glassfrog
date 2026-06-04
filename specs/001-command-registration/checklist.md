# Checklist: Command Registration

**Feature**: 001-command-registration
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/no-runnable-cli/command-registration.feature, tasks.md
**Checks**: 11 (11 pass, 0 fail)
**Generated**: 2026-06-03

---

## Summary

All 11 checks pass. Constitution: 11/11. Done-criteria: not run (no accords). Cross-references: not run (no accords).

6 of 12 constitution principles produced applicable checks; 6 are N/A for this internal command-framework feature (see Governance Notes).

---

## Constitution Checks: 11/11 passed

### Failures

None.

### Passed (11/11)

**P0** | CONSTITUTION.md II (Action Transparency): "every error MUST explain what went wrong"
→ **interface-spec.md § Error Communication**: every registration violation is reported with an error naming the offending command and the violated rule. Cause is traceable. *(See Advisory Notes on the "next step" clause.)*

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "MUST validate a write before sending"
→ **plan.md § ADR-3 / interface-spec.md**: the guard validates each command (name, summary, action/children, collision) before attaching it to the tree.

**P0** | CONSTITUTION.md III: "MUST NOT leave governance in a partially-applied state"
→ **plan.md § Cross-cutting Concerns / no-runnable-cli/command-registration.feature**: a failed registration aborts startup before dispatch; scenario "One failed registration prevents the whole CLI from running" asserts no command runs and no partial command tree is exposed.

**P0** | CONSTITUTION.md III (anti-pattern: swallowing errors)
→ **interface-spec.md § Error Communication**: violations surface as a returned error (`Register`) or panic (`MustRegister`) — never swallowed.

**P0** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/no-runnable-cli/command-registration.feature**: 15 `@wip` scenarios exist before implementation, covering the registration behaviors.

**P0** | CONSTITUTION.md IV: "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T003**: acceptance criteria mandate RED-first unit tests for the guard's happy path and each rule; T005 provides executable acceptance.

**P0** | CONSTITUTION.md V (Composition over Monolith): "Adding a new command MUST NOT require changing unrelated ones"
→ **spec.md § Behavioral Accord / plan.md § ADR-4 / no-runnable-cli/command-registration.feature**: registering one command leaves existing commands unchanged; explicit wiring adds one line plus the command's own package; scenario "Registering a command leaves existing commands untouched" verifies it.

**P0** | CONSTITUTION.md V: "modular, independently-testable parts"
→ **plan.md § System Architecture**: each command lives in its own package; the guard is a pure function over (parent, child), unit-testable in isolation.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests
→ **tasks.md § T003, T005**: implementation tasks carry verifiable acceptance criteria with tests. *(See Advisory Notes on the T004/T005 split.)*

**P0** | CONSTITUTION.md XII (Standalone Executable): "self-contained executable … no language runtime"
→ **plan.md § ADR-1**: Go self-contained executable chosen specifically to satisfy XII; no runtime assumed.

**P0** | CONSTITUTION.md XII: build emits a standalone artifact
→ **tasks.md § T002**: `go build` produces a single self-contained `glassfrog` executable (`CGO_ENABLED=0` to avoid cgo where supported); acceptance criterion confirms a runnable binary on a clean build.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria checks did not. Consider creating, to enable vertical quality checks for later specs:
  - `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`
- **Principle I (Spec Fidelity)**: no applicable checks — this feature defines the command framework, not any API-backed command. The sample commands (`version`, `roles list/get`) are registration exercises, not spec-operation bindings. Deferred to the endpoint-command specs.
- **Principle VI (Size-Aware)**: N/A — no result sets or pagination in this feature.
- **Principle VIII (No Fabricated Data)**: N/A — no API data is presented.
- **Principle IX (Writes Require Explicit Intent)**: N/A — no writes/mutations; registration is in-process setup, not an API mutation.
- **Principle X (Respect API Limits)**: N/A — no network calls.
- **Principle XI (Governance via Proposals)**: N/A — no governance-mutating command path exists in this feature.

## Advisory Notes (not severity findings)

These passed their binary checks but are worth the developer's attention:

- **II — "next step" in errors**: registration errors specify the cause (offending command + rule). The constitution also asks errors to state a *next step*. For developer-facing startup errors the remediation is implicit (fix that command), so this passes — but making the remediation explicit in the guard's error text would fully honor Principle II.
- **VII — T004/T005 split**: T004 (wiring) defers its tests to T005 (acceptance). To avoid a code-only increment under Principle VII, T004's PR should include tests for its own acceptance criteria rather than relying entirely on T005.
- **V — central wiring file**: explicit wiring in `main` (ADR-4) is a shared touch-point every new command edits. This satisfies "no unrelated *command* edits" and was a deliberate choice over `init()`, but the wiring file concentrates change — worth watching as the command set grows.
