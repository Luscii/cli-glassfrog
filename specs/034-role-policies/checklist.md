# Checklist: Role Policies

**Feature**: 034-role-policies
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` present — done-criteria and cross-reference checks not generated.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/governance-reads/role-policies.feature, tasks.md
**Checks**: 20 (20 pass, 0 fail)
**Generated**: 2026-06-10

---

## Summary

All 20 checks pass. Constitution: 20/20. (No done-criteria or cross-reference checks — no `done-*` accords found.)

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 16 | 16 | 0 |
| P1 (should fix) | 4 | 4 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **20** | **20** | **0** |

One constitution principle (XI Governance via Proposals) produced no applicable checks for this read-only feature — see Governance Notes. As with Role Reads (025), **X (Respect API Limits)** is applicable on the list path (the walk routes through 017's `RetryExecutor`) and **VI (Size-Aware)** is exercised fully (the `policies` list walks to completion by default; the single `policy` read is one unpaginated object). The one finding worth the developer's eye is a **P1** under V — the `Document[T]` generalization touches the landed 025 `RoleDocument` — which the plan mitigates with a type alias and a byte-stability acceptance criterion; see the V check below and Governance Notes.

---

## Constitution Checks: 20/20 passed

### Calibration

Six principles were calibrated to this feature (broad MUST / decision-framing language). Each calibrated assertion is binary:

- **I Spec Fidelity** → "Both commands target real spec operations (`listRolePolicies` `GET /roles/{id}/policies`, `getPolicy` `GET /policies/{id}`); the only query parameter sent on the list is `q` (+ paging's `per_page`/`cursor`), all documented; the single read sends no query parameter; no invented parameter or `include` is offered."
- **II Action Transparency** → "Output is a structured/parseable projection that always carries the machine-actionable `pol_…` id; every error path states a next step; no free-form-only output; the token never appears."
- **V Composition over Monolith** → "The feature adds a new `internal/cli/policies.go` (two commands) + two new render keys + grows the shared `Policy` + generalizes the single-object envelope, updating only the `Assemble()` wiring; no sibling command module is edited, and the one shared-schema change (`RoleDocument`→`Document[RoleDetail]`) is alias-preserving so 025's command code is untouched."
- **VI Size-Aware** → "The `policies` list walks every page to completion by default (`paging.All`); the `--first-page` opt-out and a mid-walk failure each surface an explicit incomplete boundary signal and never silently drop records; the single `policy` read returns one object (no pagination)."
- **X Respect API Limits** → "The list walk sends each page through 017's `RetryExecutor`, so a `429` is backed off rather than ignored; reads send no `If-Match` (no write exists to clobber)."
- **XII Standalone Executable** → "The artifact remains a self-contained Go binary; no new language runtime or external software dependency is introduced; only network access to the API is assumed."

### Passed (20/20)

**P0** | CONSTITUTION I (Spec Fidelity): "Every command MUST map to a spec operation; MUST NOT invent endpoints, parameters, or behaviors."
→ **interface-cli.md § Surface + interface-spec.md Example + spec.md**: Verified against `spec/glassfrog-api-v5.yaml` — `listRolePolicies` (`GET /roles/{id}/policies`, L2758) documents `id` + `per_page`/`cursor`/`q`; `getPolicy` (`GET /policies/{id}`, L2806) documents only `id`. `policies` sends `q` (from `--query`) + paging's `per_page`/`cursor`; `policy` sends no query parameter. No invented parameter; no `include` is offered (the policies endpoints define none). PASS.

**P0** | CONSTITUTION I corollary (no invented behavior — id pass-through): "MUST NOT rely on behavior the spec doesn't define."
→ **plan.md ADR-3 + interface-cli.md Error Communication**: Both ids are passed through unvalidated; an unknown/malformed id surfaces the API's documented response (`listRolePolicies` documents `400`/`401`/`404`; `getPolicy` documents `401`/`404`). The CLI invents no local "not found" or id-format behavior. PASS.

**P0** | CONSTITUTION I (Detection — request shape matches a spec operation): "Each command's request shape matches a spec operation."
→ **interface-spec.md Example + role-policies.feature "The policy list is narrowed by a search query" + tasks.md T003/T004 acceptance**: Pinned requests are `GET /roles/{id}/policies` (Query: `q` when set) and `GET /policies/{id}` (no query); the search scenario asserts the request carries the `q` parameter; T003/T004 acceptance assert no out-of-spec parameter is sent. PASS.

**P0** | CONSTITUTION II (Action Transparency, NON-NEGOTIABLE): "Output MUST be machine-parseable, report the operation and target; every error MUST state a cause and next step."
→ **interface-cli.md § Output + Error Communication**: Each rendered policy carries its `pol_…` id (the machine-actionable handle); `-o json`/`yaml` emit the raw payload; the error table maps every outcome to an exit code and a next step. PASS.

**P0** | CONSTITUTION II corollary (token hygiene): "Output must not leak the token."
→ **interface-cli.md/interface-spec.md Error Communication + plan.md Cross-cutting + tasks.md T003/T004 ("never read ctx.Cred.Token")**: The reads never read `ctx.Cred.Token`; projection renders response-side fields only; pinned by acceptance criteria across success and every error branch. PASS.

**P0** | CONSTITUTION II corollary (Detection — traceable to endpoint + resource): negative check.
→ **interface-cli.md § Output**: Every rendered policy surfaces its `pol_…` id; the operations are `GET /roles/{id}/policies` / `GET /policies/{id}`. Output is traceable to operation + resource. PASS.

**P0** | CONSTITUTION III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable; never leave partially-applied state."
→ **interface-cli.md Error Communication + plan.md Cross-cutting**: An invalid `--output` and a list-only flag on `policy` are refused before any request (cobra usage error, transport tripwire); base-URL error refuses at `NewClient`; non-2xx is never treated as success; an undecodable 2xx is a loud `RuntimeError`; a mid-walk failure renders the partial set, signals the cause, and exits non-zero. No write path, so partial-application cannot arise. PASS.

**P0** | CONSTITUTION III corollary (Detection — no swallowed errors / no failure-as-success): negative check.
→ **interface-cli.md Error Communication + interface-spec.md Error Communication**: Every typed client error maps to a non-success `Outcome`; the deliberate `--first-page` boundary exits 0 with a signal, but a mid-walk *error* exits non-zero with its cause — failure is never reported as success. PASS.

**P0** | CONSTITUTION IV (TDD, Red→Green): "Built test-first; user-facing behavior MUST have an executable acceptance scenario before the code."
→ **plan.md Implementation Strategy + tasks.md T001–T005 + role-policies.feature**: Every task is RED-first; the driving scenarios exist as Gherkin carrying `@wip`; T005 removes `@wip` from the behavioral scenarios when their executable path passes (the 2 `@validation` scenarios stay `@wip`). PASS.

**P0** | CONSTITUTION V (Composition over Monolith): "Modular, independently-testable parts; adding a command MUST NOT require changing unrelated ones."
→ **plan.md System Architecture + ADR-1 + tasks.md T003/T004**: 034 adds a new `internal/cli/policies.go` (two commands + shared seam), grows the shared `glassfrog.Policy`, adds two new `internal/render` keys (`policies`/`policy`), and updates the `Assemble()` wiring. No edit to sibling command modules; the pure `runPoliciesList`/`runPolicyGet` are independently unit-testable over the injected seam. PASS.

**P0** | CONSTITUTION VI (Size-Aware by Design): "Handle large result sets within pagination limits; MUST NEVER silently truncate; page through OR signal the boundary."
→ **spec.md Completeness + interface-cli.md Interactions + plan.md Cross-cutting + tasks.md T003**: The `policies` list walks every page to completion via `paging.All` (the "page through" half); the `--first-page` opt-out prints one page with a "more exist" stderr signal, and a mid-walk failure prints the partial set with an "incomplete — cause" signal (the "signal the boundary" half). No path silently drops records. PASS.

**P0** | CONSTITUTION VI corollary (Detection — no fetch that assumes one page is complete): negative check.
→ **plan.md Cross-cutting + interface-spec.md `runPoliciesList`**: The default path never assumes a single page is the whole set — it walks until `has_next_page` is false; the only single-page path is the *explicit, signalled* `--first-page` opt-out. The single `policy` read is a single object, not a truncated list. PASS.

**P0** | CONSTITUTION VII (Working Software): "Every commit/PR includes implementation with its tests and MUST validate and build."
→ **tasks.md T001–T005 acceptance**: Each task pairs implementation with RED-first tests and asserts `go build ./...` / `go vet ./...` clean; no code-only or test-only increment is specified. PASS.

**P0** | CONSTITUTION VIII (No Fabricated Data): "Present only data the API returned; MUST NOT invent or fill placeholder values."
→ **plan.md ADR-4 + interface-cli.md § Output + tasks.md T002**: The projection renders only decoded fields; a null `role_id`/`domain_id` renders an explicit-absence marker (`(org-level — no role)` / `(whole-role — no domain)`) via 019's `{{if}}`/`missingkey=error`, never a synthesized value; the full body is rendered verbatim, never truncated, reflowed, or filled. PASS.

**P0** | CONSTITUTION VIII corollary (Detection — no value not traceable to an API response): negative check.
→ **interface-cli.md § Output + interface-spec.md `Policy` (grown)**: Every projected field maps to a decoded response field (`id`/`title`/`body`/`role_id`/`domain_id`/`created_at`/`updated_at`); the no-role / no-domain markers render actual nulls, not guessed ids. PASS.

**P0** | CONSTITUTION IX (Writes Require Explicit Intent): "No mutation except via an explicit write command; a read MUST NEVER mutate as a side effect."
→ **spec.md Non-Behaviors + plan.md (GET-only)**: `policies`/`policy` are read-shaped commands issuing only `GET /roles/{id}/policies` / `GET /policies/{id}`; a Non-Behavior explicitly forbids writing/mutating any policy or role. No POST/PATCH/DELETE path exists. PASS.

**P0** | CONSTITUTION X (Respect API Limits): "Honor rate limits — back off on 429; use If-Match/ETag for updates."
→ **plan.md Cross-cutting + interface-spec.md `policiesSeam` (RetryExecutor) + DECISIONS (017)**: The list walk sends each page through 017's landed `RetryExecutor`, so a `429` is backed off (with `Retry-After`/`X-RateLimit-*`), not ignored — and 015 classifies a capped-out `429` to rate-limit(5). No `If-Match` applies (read, no write to clobber). PASS.

**P0** | CONSTITUTION XII (Standalone Executable): "Self-contained executable; no pre-installed runtime or software beyond network access."
→ **plan.md System Architecture + tasks.md T001 ("no new internal imports")**: 034 is Go command/schema/template code in the existing binary; it introduces no new language runtime or external dependency. The artifact remains the same self-contained executable. PASS.

**P1** | CONSTITUTION V (SHOULD — adding a command must not force changes to unrelated parts): the one shared-schema change is alias-preserving.
→ **plan.md ADR-2 + Risk 1 + tasks.md T001 acceptance ("RoleDocument remains usable at 025's existing call site (alias) — 025's decode tests still pass")**: The `Document[T]` generalization is the only change touching landed code outside this feature. It is realized as `type RoleDocument = Document[RoleDetail]`, so 025's command and decode call sites compile and behave unchanged, and T001's acceptance pins that 025's decode tests still pass. This is a shared-schema generalization (the established `Page[T]` pattern, 016), not a coupling of unrelated command modules. PASS — flagged so the Builder treats the byte-stability criterion as load-bearing.

**P1** | CONSTITUTION III (SHOULD — anti-pattern: auto-fixing without explicit intent): non-interactive, no auto-fix.
→ **interface-cli.md Error Communication + spec.md**: The commands surface problems as typed outcomes (usage errors, named failures) rather than prompting or auto-correcting; consistent with the agent-operator model. PASS.

**P1** | CONSTITUTION V (SHOULD — no hidden cross-module dependency): the reuse surface is declared, not hidden.
→ **interface-spec.md "Consumed unchanged" + plan.md System Architecture**: Every reused symbol (`AssembleFromOS`/`NewClientFromOS`/`Execute`/`RetryExecutor`, `paging.All`/`Page[T]`, `classifyClientError`/`Outcome`/`ExitCode`, `--base-url`/`--output`, `renderResult[T]`) is named explicitly as a consumed, unchanged dependency — declared, not hidden. PASS.

**P1** | CONSTITUTION VI (SHOULD — boundary signal is explicit and unambiguous): the two incompleteness causes are distinguished.
→ **plan.md Cross-cutting + interface-cli.md Interactions**: A deliberate `--first-page` opt-out exits 0 with a "more available" note; a mid-walk *failure* exits non-zero with an "incomplete — cause" note — so a partial set from an error is never mistaken for a chosen boundary, and neither is mistaken for the whole. PASS.

---

## Governance Notes

- **No `accords/governance/` directory**: No `done-*.md` accords exist (`done-specify`, `done-plan`, `done-interface`, `done-scenarios`, `done-tasks` all absent). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-<skill>.md` files to enable per-skill quality checks beyond the constitution. (Same gap noted for 025/026.)
- **Constitution V (shared-schema change)** — the `Document[T]` generalization is the only edit reaching landed 025 code. It is alias-preserving and acceptance-pinned (T001); not a sibling-command coupling. Recorded as a P1 so the byte-stability criterion stays load-bearing through implementation, not because the design violates V.
- **Constitution X (Respect API Limits)** — applicable on the list path: the `policies` walk routes each page through 017's `RetryExecutor`, so `429` back-off is honored. The `If-Match`/`ETag` half remains N/A (read-only, no write to clobber).
- **Constitution XI (Governance via Proposals)**: No applicable checks. The principle governs governance-structure *mutation*; `policies`/`policy` mutate nothing and expose no write path — they are the read half of the read/propose split (PROJECT.md Domain).
- **Severity calibration note**: All constitution MUST/MUST NOT principles inherited P0; the four SHOULD-flavored checks inherited P1. Severity was inherited mechanically from the source, not assessed by impact.
