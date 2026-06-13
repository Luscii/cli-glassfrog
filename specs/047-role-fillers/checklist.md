# Checklist: Role Fillers

**Feature**: 047-role-fillers
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/who-to-contact-for-a-role/role-fillers.feature, tasks.md
**Checks**: 14 (14 pass, 0 fail)
**Generated**: 2026-06-13

---

## Summary

All 14 checks pass. Constitution: 14/14. (No `done-*` accords found — done-criteria and cross-reference categories not generated; see Governance Notes.)

---

## Constitution Checks: 14/14 passed

### Passed (14/14)

- **P0** | CONSTITUTION.md I (Spec Fidelity): "Every command MUST map to an operation defined in the spec; MUST NOT invent endpoints, parameters, or behaviors" → **spec.md/plan.md/interface-cli.md**: `fillers <role-id>` maps to `listRoleAssignments` (`GET /roles/{role_id}/assignments`, `spec/glassfrog-api-v5.yaml:1644`); sends only the spec's default `include=actor` + pagination params; invents no parameter. ADR-1 explicitly *refuses* to invent a `filler <asgn-id>` read because the spec exposes no `GET /assignments/{id}`. **PASS.**
- **P0** | CONSTITUTION.md II (Action Transparency): "Every action MUST report the operation + target resource in machine-parseable form; every error MUST explain cause + next step" → **interface-cli.md § Output / Error Communication**: success emits structured `json`/`yaml` (018) carrying `role_id`/`actor_id`; the error table maps every failure to a cause + next step and never prints the token. **PASS.**
- **P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable; failure MUST NOT be reported as success" → **interface-cli.md § Error Communication / Completeness**: every failure exits non-zero with a named diagnostic; no swallowed errors. (Write-validation / partial-state clause is N/A — read-only feature; see Governance Notes.) **PASS.**
- **P0** | CONSTITUTION.md IV (TDD / BDD): "User-facing behavior MUST have an executable acceptance scenario before the code" → **tasks.md T001–T003 + role-fillers.feature**: the feature file exists with behavioral scenarios; T001/T002 are RED-first ("RED-first unit tests for every branch"); T003 makes the driving scenarios executable and un-`@wip`s them. **PASS.**
- **P0** | CONSTITUTION.md V (Composition over Monolith): "Modular per-resource command modules; adding a command MUST NOT change unrelated ones" → **plan.md § System Architecture / tasks.md**: a new `internal/cli/fillers.go` + one new `internal/render` key; reuses shared seams; touches no unrelated command (model reused as-is, no `internal/glassfrog` edit). **PASS.**
- **P0** | CONSTITUTION.md VI (Size-Aware by Design): "MUST handle pagination and NEVER silently truncate; page through or signal the boundary" → **spec.md § Completeness / interface-cli.md § Interactions / role-fillers.feature**: walks all pages via `paging.All` by default; `--first-page` opt-out writes a "more fillers exist" note (exit 0); mid-walk failure renders the partial set + explicit "incomplete" note + non-zero exit. **PASS.**
- **P0** | CONSTITUTION.md VII (Working Software): "Every commit/PR includes implementation + tests, and validates/builds" → **tasks.md T001–T003**: each task's acceptance criteria require its own tests (golden/unit/BDD) and `go build`/`go vet` clean — code and tests land together. **PASS.**
- **P0** | CONSTITUTION.md VIII (No Fabricated Data): "Present only data the API returned; MUST NOT invent or fill placeholder values" → **spec.md § Output / plan.md ADR-2 / interface-cli.md § Output / T001**: nullable `focus`/`elected_until` render an *explicit-absence marker* (`(none)` / `(not an elected seat)`), never an invented value; structured output carries raw API bytes. The absence markers represent genuine null, satisfying — not violating — this principle. **PASS.**
- **P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "A read-shaped command MUST NEVER mutate as a side effect" → **spec.md § Non-Behaviors / plan.md ADR-1**: `fillers` issues only `GET`; the endpoint's `POST`/`PATCH`/`DELETE` are explicitly out of scope. **PASS.**
- **P0** | CONSTITUTION.md X (Respect API Limits): "Back off on 429" → **interface-cli.md § Error Communication / plan.md**: requests go through the landed `RetryExecutor` (017); `429` maps to `RateLimited(5)`. (If-Match/ETag clause is N/A — no updates; see Governance Notes.) **PASS.**
- **P0** | CONSTITUTION.md XI (Governance via Proposals): "No default command path mutates governance structure directly" → **spec.md / plan.md**: a read of assignments mutates nothing; no governance-structure write path is exposed. **PASS.**
- **P0** | CONSTITUTION.md XII (Standalone Executable): "Self-contained binary, no pre-installed runtime/deps" → **plan.md / tasks.md**: adds pure Go to the existing binary (`internal/cli` + `internal/render`); introduces no new runtime or external dependency. **PASS.**
- **P0** | CONSTITUTION.md II + VIII (token safety): "never includes the token" / no fabricated values in errors → **interface-cli.md § Error Communication + tasks.md T002 risk**: "Never read `ctx.Cred.Token`"; the token appears in no message. **PASS.**
- **P0** | CONSTITUTION.md I (no invented filter surface): MUST NOT send undefined parameters → **plan.md ADR-3 / interface-cli.md**: the command exposes **no** filter flags and **no** `--include` because `listRoleAssignments` accepts none beyond `include`+pagination — it does not fabricate a query surface the spec lacks. **PASS.**

---

## Governance Notes

- **done-specify.md / done-plan.md / done-scenarios.md / done-tasks.md / done-interface.md**: Not found (no `accords/governance/` directory exists). Done-criteria and cross-reference check categories were not generated. Consider creating these accords to enable vertical done-criteria checks; until then, checklist coverage for this project is constitution-only. (Consistent with prior specs — `interface-cli.md` notes "No accords/ directory exists.")
- **CONSTITUTION.md III — write-validation / partial-state clause**: No applicable checks for this feature. Role Fillers is read-only; there is no write to validate or partial state to roll back.
- **CONSTITUTION.md X — optimistic concurrency (If-Match/ETag) clause**: No applicable checks for this feature. The `If-Match` requirement governs updates; Role Fillers issues no updates. The `429` back-off half of the principle is checked and passes.

---

_Constitution-only checklist (no done-* accords). All 12 principles evaluated; 14 binary checks generated (some principles produced multiple checks, two principles contributed N/A sub-clauses noted above). No P0/P1 failures._
