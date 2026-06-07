# Checklist: My Projects

**Feature**: 014-my-projects
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` present — done-criteria and cross-reference checks not generated.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/self-service-reads/my-projects.feature, tasks.md
**Checks**: 18 (18 pass, 0 fail)
**Generated**: 2026-06-07

---

## Summary

All 18 checks pass. Constitution: 18/18. (No done-criteria or cross-reference checks — no `done-*` accords found.)

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 16 | 16 | 0 |
| P1 (should fix) | 2 | 2 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **18** | **18** | **0** |

Two constitution principles (X Respect API Limits, XI Governance via Proposals) produced no applicable checks for this read-only command — see Governance Notes.

---

## Constitution Checks: 18/18 passed

### Calibration

Five principles were calibrated to this feature (broad MUST / decision-framing language). Each calibrated assertion is binary:

- **I Spec Fidelity** → "The command targets a real spec operation; every parameter it sends is documented on that operation; the status enum it validates against matches the spec's enum; no invented parameter (notably no `include`) is offered."
- **II Action Transparency** → "Output is a structured/parseable projection that always carries the machine-actionable id; every error path states a next step; no free-form-only output; the token never appears."
- **V Composition over Monolith** → "The feature adds a new command module without editing sibling command modules; registration is a single additive wiring line; no hidden cross-module dependency is introduced."
- **VI Size-Aware** → "When more results exist, the command surfaces a 'more available' boundary signal and never silently drops records; the one-page fetch is an explicit, signalled boundary, not silent truncation."
- **XII Standalone Executable** → "The artifact remains a self-contained Go binary; no new language runtime or external software dependency is introduced; only network access to the API is assumed."

### Passed (18/18)

**P0** | CONSTITUTION I (Spec Fidelity): "Every command MUST map to a spec operation; MUST NOT invent endpoints, parameters, or behaviors."
→ **interface-cli.md § Command/Flags + plan.md ADR-1/ADR-2 + spec.md**: Verified against `spec/glassfrog-api-v5.yaml` L1040–1091 — `listMyProjects` (`GET /me/projects`) exists; its documented parameters are exactly `per_page`, `cursor`, `status`; the `status` enum is exactly `archived, cancelled, completed, current, scheduled, someday, waiting` — matching `spec.md` Assumptions L178 and `interface-cli.md` Flags L28. ADR-2 correctly declines an `include` flag because the operation documents none. PASS.

**P0** | CONSTITUTION II (Action Transparency, NON-NEGOTIABLE): "Output MUST be machine-parseable, report the operation and target; every error MUST state a cause and next step."
→ **interface-cli.md § Output projection + Error Communication**: The projection emits one entry per project with the id always present (`proj_…`, the machine-actionable handle, L33–35); the error table (L74–84) maps every outcome to an exit code, and L86 requires every message to state a next step. PASS.

**P0** | CONSTITUTION II (token hygiene corollary): "Output must not leak the token."
→ **interface-cli.md L56/L88 + interface-spec.md L85 + plan.md Cross-cutting L119**: The command never reads `ctx.Cred.Token`; projection renders response-side fields only; pinned by a token-never-in-output test across success and every error branch (feature scenario "The token never appears in any output", L109–113). PASS.

**P0** | CONSTITUTION III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable; validate a write before sending; never leave partially-applied state."
→ **plan.md Cross-cutting L121 + spec.md Error handling**: Unsupported `--status` is refused before any request; base-URL error refuses at `NewClientFromOS`; non-2xx is never treated as success (generic `APIError`/3); an undecodable 2xx is a loud `RuntimeError`/1; the `default→1` registry fail-safe backstops unmapped categories. (No write path exists, so partial-application cannot arise.) PASS.

**P0** | CONSTITUTION IV (TDD, Red→Green): "Built test-first; user-facing behavior MUST have an executable acceptance scenario before the code."
→ **plan.md Testing L123 + tasks.md T001/T002/T003 + my-projects.feature**: All tasks are RED-first; the driving scenarios exist as Gherkin in `features/self-service-reads/my-projects.feature` carrying `@wip`, and T003 removes `@wip` from behavioral scenarios when their executable path passes (held `@validation` scenarios stay `@wip`). PASS.

**P0** | CONSTITUTION V (Composition over Monolith): "Modular, independently-testable parts; adding a command MUST NOT require changing unrelated ones."
→ **plan.md System Architecture L20 + tasks.md T003**: 014 adds `internal/cli/my_projects.go` (new) and one additive `MustRegister(myParent, newMyProjectsCommand(...))` line in `Assemble()`. No edit to sibling command modules. The pure `runMyProjects`/`formatMyProjects` are independently unit-testable over the injected seam. PASS.

**P0** | CONSTITUTION VI (Size-Aware by Design): "Handle large result sets within pagination limits; MUST NEVER silently truncate; signal the boundary."
→ **spec.md Pagination boundary L31–33 + interface-cli.md L37/L66 + interface-spec.md L61**: When `meta.pagination.has_next_page` is true, the projection appends the "more results available" signal; the first-page-only fetch is an explicit, signalled boundary (one request, no second page). Multi-page walking is explicitly deferred to Pagination (016), not silently dropped. PASS.

**P0** | CONSTITUTION VII (Working Software): "Every commit/PR includes implementation with its tests and MUST validate and build."
→ **tasks.md T001/T002/T003 acceptance criteria**: Each task pairs implementation with RED-first tests and asserts `go build ./...` / `go vet ./...` clean; no code-only or test-only increment is specified. PASS.

**P0** | CONSTITUTION VIII (No Fabricated Data): "Present only data the API returned; MUST NOT invent or fill placeholder values."
→ **plan.md Data Model L97 + interface-cli.md L35 + my-projects.feature L82–89**: The projection renders only decoded response fields; a null `role_id` renders an explicit no-role marker ("—" / "no role") rather than a fabricated role id. Decoding tolerates unknown fields without synthesizing values. PASS.

**P0** | CONSTITUTION IX (Writes Require Explicit Intent): "No mutation except via an explicit write command; a read MUST NEVER mutate as a side effect."
→ **spec.md Non-Behaviors L72 + plan.md (single `GET /me/projects`)**: `my projects` is a read-shaped command issuing exactly one `GET`; a Non-Behavior explicitly forbids create/update/mutate. No POST/PATCH/DELETE path exists. PASS.

**P0** | CONSTITUTION XII (Standalone Executable): "Self-contained executable; no pre-installed runtime or software beyond network access."
→ **plan.md System Architecture / tasks.md T001 ("no new internal imports")**: 014 is a Go command leaf plus a schema struct in the existing binary; introduces no new language runtime or external dependency. The artifact remains the same self-contained executable. PASS.

**P0** | CONSTITUTION I corollary (Detection — contract test): "Each command's request shape matches a spec operation."
→ **interface-spec.md L42–52 + my-projects.feature "A supported status filters the request"**: The pinned request is `{Method:"GET", Path:"/me/projects", Query:{status?}}`; the status filter scenario asserts the request carries `status=current`; no parameter outside the spec's three is sent. PASS.

**P0** | CONSTITUTION II corollary (Detection — traceable to endpoint + resource): "An action whose output cannot be traced to an endpoint plus resource id is a violation."
→ **interface-cli.md L33–35**: The projection always surfaces the project `id` (the resource handle) for each rendered entry, and the command's single operation is `GET /me/projects`. Output is traceable to operation + resource. PASS.

**P0** | CONSTITUTION III corollary (Detection — no swallowed errors / no failure-as-success): negative check.
→ **interface-spec.md Error Communication L70–86**: Every typed error from the client maps to a non-success `Outcome`; "Never zero on failure" is asserted (interface-cli.md L89). No error is swallowed or reported as success. PASS.

**P0** | CONSTITUTION VI corollary (Detection — no fetch that ignores per_page and assumes one page is complete): negative check.
→ **spec.md L33 + interface-spec.md L61**: The command makes a single page request by design and signals incompleteness via `HasNextPage`; it does not assume one page is the complete set — it explicitly marks the boundary. PASS.

**P0** | CONSTITUTION VIII corollary (Detection — no value not traceable to an API response): negative check.
→ **interface-cli.md L35–37 + plan.md L97**: Every projected field maps to a decoded response field (`id`, `status`, `description`, `role_id`, `has_sub_projects`, `has_actions`, `tags`); the no-role marker is a rendering of an actual null, not a synthesized value. PASS.

**P1** | CONSTITUTION III (SHOULD — anti-pattern: auto-fixing without explicit intent): non-interactive, no auto-fix.
→ **spec.md Non-Behaviors L73**: The command does not prompt interactively and surfaces problems as typed outcomes rather than auto-correcting; consistent with the agent-operator model. PASS.

**P1** | CONSTITUTION V (SHOULD — no hidden cross-module dependency): the reuse surface is declared, not hidden.
→ **interface-spec.md "Consumed unchanged" L36–40 + plan.md L108–113**: Every reused symbol (011's `classifyClientError`/`Outcome`/`ExitCode`/`--base-url`; 012's `my` parent/`Pagination`/envelope/signal; 013's `validateStatus`) is named explicitly as a consumed, unchanged dependency — declared, not hidden. PASS.

---

## Governance Notes

- **No `accords/governance/` directory**: No `done-*.md` accords exist (`done-specify`, `done-plan`, `done-interface`, `done-scenarios`, `done-tasks` all absent). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-<skill>.md` files to enable per-skill quality checks beyond the constitution.
- **Constitution X (Respect API Limits)**: No applicable checks for this feature. The principle governs `If-Match`/`ETag` on writes and `429`/`Retry-After` back-off; `my projects` is a read (no `If-Match`), and `429` back-off is explicitly deferred to Rate-Limit Handling (017). Non-2xx (incl. `429`) maps to the generic `APIError`/3 bucket today, by design.
- **Constitution XI (Governance via Proposals)**: No applicable checks. The principle governs governance-structure mutation; `my projects` mutates no governance and exposes no write path.
- **Severity calibration note**: All constitution MUST/MUST NOT principles inherited P0; the two SHOULD-flavored anti-pattern checks inherited P1. Severity was inherited mechanically from the source, not assessed by impact.
