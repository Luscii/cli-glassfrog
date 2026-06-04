# Checklist: Argument Dispatch

**Feature**: 002-argument-dispatch
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/no-runnable-cli/argument-dispatch.feature, tasks.md
**Checks**: 9 (9 pass, 0 fail)
**Generated**: 2026-06-03

---

## Summary

All 9 checks pass. Constitution: 9/9. Done-criteria: not run (no accords). Cross-references: not run (no accords).

6 of 12 constitution principles produced applicable checks; 6 are N/A for an in-process routing capability with no API/data/governance surface (see Governance Notes).

---

## Constitution Checks: 9/9 passed

### Failures

None.

### Passed (9/9)

**P0** | CONSTITUTION.md II (Action Transparency): "every error MUST explain what went wrong and the next step"
→ **spec.md § Unknown command / interface-cli.md § Error Communication**: an unknown command is reported with the unrecognized token named and a pointer to help — cause plus next step. A best-effort "did you mean" may accompany it.

**P0** | CONSTITUTION.md II: action outcome is traceable
→ **plan.md § Outcome Classification / interface-spec.md**: every dispatch outcome is classified (`Success` / `UsageError`), so the operator can tell what kind of outcome occurred. (A resolved command's own runtime failure is returned uncategorized; a distinct `RuntimeError` is deferred to Exit-Code Convention / 004.)

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): errors obvious, never hidden
→ **spec.md § Invalid input**: an unknown flag or unexpected argument is a usage error and the command does not run — unexpected input is never silently ignored (the clarified decision).

**P0** | CONSTITUTION.md III (anti-pattern: swallowing errors)
→ **plan.md § Cross-cutting Concerns**: dispatch returns the outcome category alongside any error and never swallows; it does not terminate the process itself.

**P0** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/no-runnable-cli/argument-dispatch.feature**: the 002 driving scenarios exist as `@wip` before implementation.

**P0** | CONSTITUTION.md IV: "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T002**: classification is specified test-first (RED-first unit tests per category).

**P0** | CONSTITUTION.md V (Composition over Monolith): modular, independently-testable parts
→ **plan.md § System Architecture**: dispatch is a thin `Run` layer over the registry, exercisable in isolation by passing an assembled tree and argument slices — it does not entangle command modules.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests
→ **tasks.md § T001–T003**: each task carries verifiable acceptance criteria with tests (unit per category, godog acceptance).

**P0** | CONSTITUTION.md XII (Standalone Executable): no new runtime/dependency
→ **plan.md § ADR-1**: dispatch is in-process over the existing cobra tree; it introduces no new external or runtime dependency.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria checks did not. (Same as 001 — consider creating `accords/governance/done-*.md` to enable vertical quality checks.)
- **Principle I (Spec Fidelity)**: N/A — dispatch routes to commands; it defines no API operation. (Real API conformance arrives with the endpoint-command specs.)
- **Principle VI (Size-Aware)**: N/A — no result sets or pagination.
- **Principle VIII (No Fabricated Data)**: N/A — dispatch presents no API data.
- **Principle IX (Writes Require Explicit Intent)**: N/A — dispatch performs no mutation; whether a read command mutates is the command's own concern.
- **Principle X (Respect API Limits)**: N/A — no network calls.
- **Principle XI (Governance via Proposals)**: N/A — no governance mutation.

## Advisory Notes (not severity findings)

These passed their checks but are worth the developer's attention going into implementation:

- **Outcome category has no consumer yet**: Exit-Code Convention (004) doesn't exist, so the category's only use today is the entrypoint's minimal mapping. Keep it minimal (plan ADR-2) so 004 adapts cleanly.
- **Exact-match depends on a cobra package-global**: `EnablePrefixMatching` must stay `false`. T002's regression test (a prefix not resolving) is the guard against silent regression — make sure it lands.
- **`main.go` exit-code placeholder carries forward 001's advisory (V-3)**: dispatch classifies but must not emit codes; the entrypoint's minimal 0/non-zero mapping remains a placeholder until Exit-Code Convention (004).
- **cobra built-in `help`/`completion` are resolvable**: their interaction with the unknown-command contract should be settled by Help & Version (003); noted in `.score/memory/LEARNINGS.md`.
