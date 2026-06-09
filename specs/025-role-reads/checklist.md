# Checklist: Role Reads

**Feature**: 025-role-reads
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` present — done-criteria and cross-reference checks not generated.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/governance-reads/role-reads.feature, tasks.md
**Checks**: 20 (20 pass, 0 fail)
**Generated**: 2026-06-08

---

## Summary

All 20 checks pass. Constitution: 20/20. (No done-criteria or cross-reference checks — no `done-*` accords found.)

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 17 | 17 | 0 |
| P1 (should fix) | 3 | 3 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **20** | **20** | **0** |

One constitution principle (XI Governance via Proposals) produced no applicable checks for this read-only command — see Governance Notes. Unlike the `/me*` reads (012–014), **X (Respect API Limits) is now applicable** because the list walk routes through 017's `RetryExecutor` (429 backoff is live), and **VI (Size-Aware) is exercised fully** (the list walks to completion by default, not just signalling).

---

## Constitution Checks: 20/20 passed

### Calibration

Six principles were calibrated to this feature (broad MUST / decision-framing language). Each calibrated assertion is binary:

- **I Spec Fidelity** → "Both commands target real spec operations (`listRoles` `GET /roles`, `getRole` `GET /roles/{id}`); every parameter sent (`parent_role_id`/`person_id`/`has_subroles`/`tag`, `include`) is documented on that operation; the `include` enum validated against matches the spec's enum; no invented parameter is offered."
- **II Action Transparency** → "Output is a structured/parseable projection that always carries the machine-actionable role id; every error path states a next step; no free-form-only output; the token never appears."
- **V Composition over Monolith** → "The feature adds a new command module (`roles.go`) + two render keys without editing sibling command modules; registration is a single additive wiring line; reused symbols are declared, not hidden."
- **VI Size-Aware** → "The list walks every page to completion by default (`paging.All`); the `--first-page` opt-out and a mid-walk failure each surface an explicit incomplete boundary signal and never silently drop records."
- **X Respect API Limits** → "The list walk sends each page through 017's `RetryExecutor`, so a `429` is backed off rather than ignored; reads send no `If-Match` (no write exists to clobber)."
- **XII Standalone Executable** → "The artifact remains a self-contained Go binary; no new language runtime or external software dependency is introduced; only network access to the API is assumed."

### Passed (20/20)

**P0** | CONSTITUTION I (Spec Fidelity): "Every command MUST map to a spec operation; MUST NOT invent endpoints, parameters, or behaviors."
→ **interface-cli.md § Surface + plan.md ADR-1/ADR-2 + spec.md**: Verified against `spec/glassfrog-api-v5.yaml` — `listRoles` (`GET /roles`, L134) documents `per_page`/`cursor`/`q`/`parent_role_id`/`person_id`/`has_subroles`/`tag`; `getRole` (`GET /roles/{id}`, L201) documents `id` + an `include` enum of exactly `assignments,subroles,parent_role,policies,notes,skills`. The command sends only `parent_role_id`/`person_id`/`has_subroles`/`tag` (+ paging's `per_page`/`cursor`) and the validated `include`; no invented parameter. PASS.

**P0** | CONSTITUTION I corollary (no invented behavior — id pass-through): "MUST NOT rely on behavior the spec doesn't define."
→ **plan.md ADR-4 + interface-cli.md Error Communication**: The role id is passed through unvalidated; an unknown/malformed id surfaces the API's documented `404` (getRole documents `401`/`404`). The CLI invents no local "not found" behavior. PASS.

**P0** | CONSTITUTION I (Detection — contract test, request shape matches a spec operation): "Each command's request shape matches a spec operation."
→ **interface-spec.md Example + role-reads.feature "The list is filtered by parent circle" / "Requested related resources are embedded inline" + tasks.md T002/T003/T004 acceptance**: Pinned requests are `GET /roles` (Query: filters) and `GET /roles/{id}` (Query: `include=…`); scenarios assert the request carries `parent_role_id` and `include=policies,subroles`; T003/T004 acceptance criteria assert no out-of-spec parameter is sent. PASS.

**P0** | CONSTITUTION II (Action Transparency, NON-NEGOTIABLE): "Output MUST be machine-parseable, report the operation and target; every error MUST state a cause and next step."
→ **interface-cli.md § Output + Error Communication**: Each rendered role carries its `role_…` id (the machine-actionable handle); `-o json`/`yaml` emit the raw payload; the error table maps every outcome to an exit code and a next step. PASS.

**P0** | CONSTITUTION II corollary (token hygiene): "Output must not leak the token."
→ **interface-cli.md Error Communication + interface-spec.md Error Communication + plan.md Cross-cutting + tasks.md T002/T004 ("never read ctx.Cred.Token")**: The reads never read `ctx.Cred.Token`; projection renders response-side fields only; pinned by acceptance criteria across success and every error branch. PASS.

**P0** | CONSTITUTION II corollary (Detection — traceable to endpoint + resource): negative check.
→ **interface-cli.md § Output**: Every rendered role surfaces its `role_…` id; the operations are `GET /roles` / `GET /roles/{id}`. Output is traceable to operation + resource. PASS.

**P0** | CONSTITUTION III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable; never leave partially-applied state."
→ **interface-cli.md Error Communication + plan.md Cross-cutting + ADR-3**: An unsupported `--include`/`--output` and the filter/id misuse are refused before any request; base-URL error refuses at `NewClient`; non-2xx is never treated as success; an undecodable 2xx is a loud `RuntimeError`; a mid-walk failure renders the partial set, signals the cause, and exits non-zero (never reported as success). No write path, so partial-application cannot arise. PASS.

**P0** | CONSTITUTION III corollary (Detection — no swallowed errors / no failure-as-success): negative check.
→ **interface-cli.md Error Communication + interface-spec.md Error Communication**: Every typed client error maps to a non-success `Outcome`; the deliberate `--first-page` boundary exits 0 with a signal, but a mid-walk *error* exits non-zero with its cause — failure is never reported as success. PASS.

**P0** | CONSTITUTION IV (TDD, Red→Green): "Built test-first; user-facing behavior MUST have an executable acceptance scenario before the code."
→ **plan.md Implementation Strategy + tasks.md T001–T005 + role-reads.feature**: Every task is RED-first; the driving scenarios exist as Gherkin carrying `@wip`; T005 removes `@wip` from the behavioral scenarios when their executable path passes (the 3 `@validation` scenarios stay `@wip`). PASS.

**P0** | CONSTITUTION V (Composition over Monolith): "Modular, independently-testable parts; adding a command MUST NOT require changing unrelated ones."
→ **plan.md System Architecture + tasks.md T002**: 025 adds `internal/cli/roles.go` (new), grows the shared `glassfrog.Role`, adds two `internal/render` keys, and a single additive `MustRegister(root, newRolesCommand(...))` line in `Assemble()`. No edit to sibling command modules; the pure `runRoles`/`runRolesList`/`runRoleGet` are independently unit-testable over the injected seam. PASS.

**P0** | CONSTITUTION VI (Size-Aware by Design): "Handle large result sets within pagination limits; MUST NEVER silently truncate; page through OR signal the boundary."
→ **spec.md Completeness + interface-cli.md Interactions + plan.md ADR-3 + tasks.md T002/T003**: The list walks every page to completion via `paging.All` (the "page through" half); the `--first-page` opt-out prints one page with a "more exist" stderr signal, and a mid-walk failure prints the partial set with an "incomplete — cause" signal (the "signal the boundary" half). No path silently drops records. PASS.

**P0** | CONSTITUTION VI corollary (Detection — no fetch that assumes one page is complete): negative check.
→ **plan.md ADR-3 + interface-spec.md `runRolesList`**: The default path never assumes a single page is the whole set — it walks until `has_next_page` is false; the only single-page path is the *explicit, signalled* `--first-page` opt-out. PASS.

**P0** | CONSTITUTION VII (Working Software): "Every commit/PR includes implementation with its tests and MUST validate and build."
→ **tasks.md T001–T005 acceptance**: Each task pairs implementation with RED-first tests and asserts `go build ./...` / `go vet ./...` clean; no code-only or test-only increment is specified. PASS.

**P0** | CONSTITUTION VIII (No Fabricated Data): "Present only data the API returned; MUST NOT invent or fill placeholder values."
→ **plan.md ADR-2 + interface-cli.md § Output + interface-spec.md `render` keys**: The projection renders only decoded fields; absent optionals render an explicit-absence marker (`(none)` / `(no purpose set)` / `(none — anchor role)`) via 019's `{{if}}`/`missingkey=error`, never a synthesized value; an unrequested `--include` section is omitted entirely (not faked). PASS.

**P0** | CONSTITUTION VIII corollary (Detection — no value not traceable to an API response): negative check.
→ **interface-cli.md § Output + plan.md ADR-2**: Every projected field maps to a decoded response field; `SkillSummary` carries no `Content` (it is not invented), and the no-parent marker renders an actual null, not a guessed id. PASS.

**P0** | CONSTITUTION IX (Writes Require Explicit Intent): "No mutation except via an explicit write command; a read MUST NEVER mutate as a side effect."
→ **spec.md Non-Behaviors + plan.md (GET-only)**: `roles` is a read-shaped command issuing only `GET /roles` / `GET /roles/{id}`; a Non-Behavior explicitly forbids writing/mutating any role. No POST/PATCH/DELETE path exists. PASS.

**P0** | CONSTITUTION X (Respect API Limits): "Honor rate limits — back off on 429; use If-Match/ETag for updates."
→ **plan.md System Architecture + interface-spec.md `rolesSeam` (RetryExecutor) + DECISIONS (017)**: The list walk sends each page through 017's landed `RetryExecutor`, so a `429` is backed off (with `Retry-After`/`X-RateLimit-*`), not ignored — and 015 classifies a capped-out `429` to rate-limit(5). No `If-Match` applies (read, no write to clobber). PASS.

**P0** | CONSTITUTION XII (Standalone Executable): "Self-contained executable; no pre-installed runtime or software beyond network access."
→ **plan.md System Architecture + tasks.md T001 ("no new internal imports")**: 025 is Go command/schema/template code in the existing binary; it introduces no new language runtime or external dependency. The artifact remains the same self-contained executable. PASS.

**P1** | CONSTITUTION III (SHOULD — anti-pattern: auto-fixing without explicit intent): non-interactive, no auto-fix.
→ **interface-cli.md Error Communication + spec.md**: The command surfaces problems as typed outcomes (usage errors, named failures) rather than prompting or auto-correcting; consistent with the agent-operator model. PASS.

**P1** | CONSTITUTION V (SHOULD — no hidden cross-module dependency): the reuse surface is declared, not hidden.
→ **interface-spec.md "Consumed unchanged" + plan.md System Architecture**: Every reused symbol (`AssembleFromOS`/`NewClientFromOS`/`Execute`/`RetryExecutor`, `paging.All`/`Page[T]`, `classifyClientError`/`Outcome`/`ExitCode`, `--base-url`/`--output`, `renderResult[T]`) is named explicitly as a consumed, unchanged dependency — declared, not hidden. PASS.

**P1** | CONSTITUTION VI (SHOULD — boundary signal is explicit and unambiguous): the two incompleteness causes are distinguished.
→ **plan.md ADR-3 + interface-cli.md Interactions**: A deliberate `--first-page` opt-out exits 0 with a "more available" note; a mid-walk *failure* exits non-zero with an "incomplete — cause" note — so a partial set from an error is never mistaken for a chosen boundary, and neither is mistaken for the whole. PASS.

---

## Governance Notes

- **No `accords/governance/` directory**: No `done-*.md` accords exist (`done-specify`, `done-plan`, `done-interface`, `done-scenarios`, `done-tasks` all absent). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-<skill>.md` files to enable per-skill quality checks beyond the constitution.
- **Constitution X (Respect API Limits)** — now applicable (was N/A for 012–014): the list walk routes each page through 017's `RetryExecutor`, so `429` back-off is honored on the walk path. The `If-Match`/`ETag` half remains N/A (read-only, no write to clobber).
- **Constitution XI (Governance via Proposals)**: No applicable checks. The principle governs governance-structure *mutation*; `roles` mutates nothing and exposes no write path — it is the read half of the read/propose split (PROJECT.md Domain).
- **Severity calibration note**: All constitution MUST/MUST NOT principles inherited P0; the three SHOULD-flavored checks inherited P1. Severity was inherited mechanically from the source, not assessed by impact.
