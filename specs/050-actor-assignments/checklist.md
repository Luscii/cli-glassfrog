# Checklist: Actor Assignments

**Feature**: 050-actor-assignments
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/an-actors-governance-footprint/actor-assignments.feature, tasks.md
**Checks**: 14 (14 pass, 0 fail)
**Generated**: 2026-06-13

---

## Summary

All 14 checks pass. Constitution: 14/14. (No `done-*` accords found — done-criteria and cross-reference categories not generated; see Governance Notes.)

---

## Constitution Checks: 14/14 passed

### Passed (14/14)

- **P0** | CONSTITUTION.md I (Spec Fidelity): "Every command MUST map to an operation defined in the spec; MUST NOT invent endpoints, parameters, or behaviors" → **spec.md/plan.md/interface-cli.md**: `assignments <actor-id>` maps to `listActorAssignments` (`GET /actors/{actor_id}/assignments`, `spec/glassfrog-api-v5.yaml:1742`); sends only the spec's default `include=role` + pagination params; invents no parameter. ADR-1 explicitly *refuses* to invent an `assignment <asgn-id>` read because the spec exposes no `GET /assignments/{id}`. **PASS.**
- **P0** | CONSTITUTION.md I (schema growth traces to the spec): the embedded `role` added to `glassfrog.Assignment` (plan ADR-2 / T001) carries exactly the fields the spec's `role` include defines — `id`, `type`, `name`, nullable `purpose`, nullable `parent_role_id` (`spec/glassfrog-api-v5.yaml:5706`); no field is invented beyond the spec's documented include shape. **PASS.**
- **P0** | CONSTITUTION.md II (Action Transparency): "Every action MUST report the operation + target resource in machine-parseable form; every error MUST explain cause + next step" → **interface-cli.md § Output / Error Communication**: success emits structured `json`/`yaml` (018) carrying `actor_id`/`role_id`; the error table maps every failure to a cause + next step and never prints the token. **PASS.**
- **P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable; failure MUST NOT be reported as success" → **interface-cli.md § Error Communication / Completeness**: every failure exits non-zero with a named diagnostic; no swallowed errors. (Write-validation / partial-state clause is N/A — read-only feature; see Governance Notes.) **PASS.**
- **P0** | CONSTITUTION.md IV (TDD / BDD): "User-facing behavior MUST have an executable acceptance scenario before the code" → **tasks.md T001–T004 + actor-assignments.feature**: the feature file exists with behavioral scenarios; T001–T003 are RED-first ("RED-first unit tests for every branch"; decode tests for the model growth); T004 makes the driving scenarios executable and un-`@wip`s them. **PASS.**
- **P0** | CONSTITUTION.md V (Composition over Monolith): "Modular per-resource command modules; adding a command MUST NOT change unrelated ones" → **plan.md § System Architecture / tasks.md**: a new `internal/cli/assignments.go` + one new `internal/render` key + an **additive** field on the shared `glassfrog.Assignment`; reuses shared seams. The model growth is additive and forward-compatible — 025's `?include=assignments` embed and 047's `fillers` projection read only the `actor` block and decode the new `role` field unused (a T001 decode test pins this), so no unrelated command must change. **PASS.**
- **P0** | CONSTITUTION.md VI (Size-Aware by Design): "MUST handle pagination and NEVER silently truncate; page through or signal the boundary" → **spec.md § Completeness / interface-cli.md § Interactions / actor-assignments.feature**: walks all pages via `paging.All` by default; `--first-page` opt-out writes a "more assignments exist" note (exit 0); mid-walk failure renders the partial set + explicit "incomplete" note + non-zero exit. **PASS.**
- **P0** | CONSTITUTION.md VII (Working Software): "Every commit/PR includes implementation + tests, and validates/builds" → **tasks.md T001–T004**: each task's acceptance criteria require its own tests (decode/golden/unit/BDD) and `go build`/`go vet` clean — code and tests land together. **PASS.**
- **P0** | CONSTITUTION.md VIII (No Fabricated Data): "Present only data the API returned; MUST NOT invent or fill placeholder values" → **spec.md § Output / plan.md ADR-2 / interface-cli.md § Output / T002**: all four nullable fields — `focus`, `elected_until`, and the embedded role's `purpose`/`parent_role_id` — render an *explicit-absence marker* (`(none)` / `(not an elected seat)` / `(top-level)`), never an invented value; structured output carries raw API bytes. The absence markers represent genuine null, satisfying — not violating — this principle. **PASS.**
- **P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "A read-shaped command MUST NEVER mutate as a side effect" → **spec.md § Non-Behaviors / plan.md ADR-1**: `assignments` issues only `GET`; the `/assignments/{id}` `PATCH`/`DELETE` and the role-end `POST` are explicitly out of scope. **PASS.**
- **P0** | CONSTITUTION.md X (Respect API Limits): "Back off on 429" → **interface-cli.md § Error Communication / plan.md**: requests go through the landed `RetryExecutor` (017); `429` maps to `RateLimited(5)`. (If-Match/ETag clause is N/A — no updates; see Governance Notes.) **PASS.**
- **P0** | CONSTITUTION.md XI (Governance via Proposals): "No default command path mutates governance structure directly" → **spec.md / plan.md**: a read of an actor's assignments mutates nothing; no governance-structure write path is exposed. **PASS.**
- **P0** | CONSTITUTION.md XII (Standalone Executable): "Self-contained binary, no pre-installed runtime/deps" → **plan.md / tasks.md**: adds pure Go to the existing binary (`internal/glassfrog` + `internal/cli` + `internal/render`); introduces no new runtime or external dependency. **PASS.**
- **P0** | CONSTITUTION.md II + VIII (token safety): "never includes the token" / no fabricated values in errors → **interface-cli.md § Error Communication + tasks.md T003 risk**: "Never read `ctx.Cred.Token`"; the token appears in no message. **PASS.**
- **P0** | CONSTITUTION.md I (no invented filter surface): MUST NOT send undefined parameters → **plan.md ADR-3 / interface-cli.md**: the command exposes **no** filter flags and **no** `--include` because `listActorAssignments` accepts none beyond `include`+pagination — it does not fabricate a query surface the spec lacks. **PASS.**

---

## Governance Notes

- **done-specify.md / done-plan.md / done-scenarios.md / done-tasks.md / done-interface.md**: Not found (no `accords/governance/` directory exists). Done-criteria and cross-reference check categories were not generated. Consider creating these accords to enable vertical done-criteria checks; until then, checklist coverage for this project is constitution-only. (Consistent with prior specs — `interface-cli.md` notes "No accords/ directory exists.")
- **CONSTITUTION.md III — write-validation / partial-state clause**: No applicable checks for this feature. Actor Assignments is read-only; there is no write to validate or partial state to roll back.
- **CONSTITUTION.md X — optimistic concurrency (If-Match/ETag) clause**: No applicable checks for this feature. The `If-Match` requirement governs updates; Actor Assignments issues no updates. The `429` back-off half of the principle is checked and passes.

---

_Constitution-only checklist (no done-* accords). All 12 principles evaluated; 14 binary checks generated (some principles produced multiple checks — including a dedicated Spec-Fidelity check for the additive `role` model growth that distinguishes 050 from the 047 mirror; two principles contributed N/A sub-clauses noted above). No P0/P1 failures._
