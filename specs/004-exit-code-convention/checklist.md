# Checklist: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/no-runnable-cli.feature, tasks.md
**Checks**: 10 (10 pass, 0 fail)
**Generated**: 2026-06-03 (re-derived after the crash-diagnostic clarification)

---

## Summary

All 10 checks pass. Constitution: 10/10. Done-criteria: not run (no accords). Cross-references: not run (no accords).

6 of 12 constitution principles produced applicable checks; 6 are N/A for an in-process exit-code mapper with no API/data/governance surface yet (see Governance Notes).

**Improvement since last run**: previous run 10/10 pass, 0 fail — unchanged in count. The clarification strengthened the Action Transparency (II) coverage: the crash diagnostic on the panic path is now an explicit spec behavior (safety-net accord) rather than only a plan/interface detail, so the II check below no longer leans on an advisory. The prior advisory about the panic-text boundary is resolved.

---

## Constitution Checks: 10/10 passed

### Failures

None.

### Passed (10/10)

**P0** | CONSTITUTION.md II (Action Transparency): "the operator MUST always be able to tell what the CLI did … in machine-parseable form"
→ **spec.md § Behavioral Accord + interface-cli.md § Surface**: each failure class carries a distinct, stable exit code (0/1/2/3/4/5/6) — the single machine signal an agent reads from `$?` to tell what kind of outcome occurred without parsing text.

**P0** | CONSTITUTION.md II: a failure stays diagnosable even on the unexpected path
→ **spec.md § Behavioral Accord (Unexpected failure) + plan.md § ADR-4**: the safety net writes the crash (failure value and any trace) to stderr before exiting `1`, so an unanticipated crash is not silently reduced to a bare code. Now an explicit spec behavior with a matching Non-Behavior carve-out and relaxed Process/shell boundary (clarification 2026-06-03).

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious … never hidden"
→ **spec.md § Behavioral Accord (Unexpected failure) + plan.md § ADR-1**: any termination matching no known category exits non-zero `1` via the registry's Fail-Safe default — a failure is never reported as `0`.

**P0** | CONSTITUTION.md III (anti-pattern: swallowing errors / failure reported as success)
→ **spec.md § Non-Behaviors**: "The system must not catch and suppress a failure to force a `0` exit" — the never-zero-on-failure rule is stated as an explicit boundary.

**P0** | CONSTITUTION.md III: an internal failure is never aliased onto a different failure class
→ **plan.md § ADR-4 + features § "An internal panic exits one and never collides with the usage code"**: a panic exits `1`, not Go's default status `2`, so a crash is never mistaken for a usage error.

**P0** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/no-runnable-cli.feature**: the 004 driving scenarios exist as `@wip` (three Rule blocks) before implementation.

**P0** | CONSTITUTION.md IV: "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001–T002**: the registry is specified tests-RED-first (uniqueness / no-shell-reserved / exact-values), and the category reclassification flips the two deferral tests RED before GREEN.

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts"
→ **plan.md § ADR-1**: `ExitCode(Outcome) int` is a pure, independently-testable function in its own file. The dispatch reclassification (T002) is the planned fulfillment of 002's deferred `RuntimeError`, not entanglement with an unrelated command.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests, validates and builds
→ **tasks.md § T001–T004**: each task carries verifiable acceptance criteria with tests and `go build`/`go vet`/`go test` clean conditions.

**P0** | CONSTITUTION.md XII (Standalone Executable): no new runtime or dependency
→ **plan.md § System Architecture**: the feature is three small parts inside the existing `internal/cli` package over the existing cobra tree; it introduces no new external or runtime dependency.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria and cross-reference checks did not. (Same as 001–003 — consider creating `accords/governance/done-*.md` to enable vertical quality checks.)
- **Principle I (Spec Fidelity)**: N/A — exit-code mapping defines no API operation. (API conformance arrives with the endpoint-command specs.)
- **Principle VI (Size-Aware)**: N/A — no result sets or pagination.
- **Principle VIII (No Fabricated Data)**: N/A — the feature emits a numeric code, presents no API data.
- **Principle IX (Writes Require Explicit Intent)**: N/A — no mutation.
- **Principle X (Respect API Limits)**: N/A for 004 directly — it makes no API calls. Note: 004 *enables* X by reserving a distinct rate-limit code (`5`) so the future API client/agent can back off; the spec explicitly defers retry/backoff to that layer (a Non-Behavior).
- **Principle XI (Governance via Proposals)**: N/A — no governance mutation.

## Advisory Notes (not severity findings)

These passed their checks but are worth the developer's attention going into implementation:

- **`main`'s placeholder behavior changes** (carries forward 002's advisory): the old entrypoint mapped *all* errors to exit `1`; usage errors now exit `2`. Intended, but call it out in the PR so no consumer is surprised.
- **Action Transparency for a runtime failure depends on the command's own error text**: 004 emits only the code; the "what went wrong + next step" message for a `RuntimeError` is the resolved command's `RunE` error rendered by cobra — out of 004's scope, but the system-level guarantee depends on it. (The panic-path diagnostic, by contrast, *is* now 004's explicit responsibility — see the II check.)
- **Forward-looking codes (3–6) have no producer yet**: their scenarios assert the registry mapping directly. Keep the operational constants framed as reserved (plan ADR-2) so the future API client adds categories at the one registry site without renumbering.
