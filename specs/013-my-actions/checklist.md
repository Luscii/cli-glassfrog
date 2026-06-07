# Checklist: My Actions

**Feature**: 013-my-actions
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/self-service-reads/my-actions.feature, tasks.md
**Checks**: 18 (18 pass, 0 fail)
**Generated**: 2026-06-07

---

## Summary

All 18 checks pass. Constitution: 18/18.

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 15 | 15 | 0 |
| P1 (should fix) | 3 | 3 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **18** | **18** | **0** |

No done-* governance accords exist, so this run generated constitution checks only (see Governance Notes).

---

## Constitution Checks: 18/18 passed

### Failures

None.

### Passed (18/18)

**Principle I — Spec Fidelity (MUST / MUST NOT, P0)** — 2/2
- **C-I.1** | "Every command MUST map to an operation defined in the Glassfrog API v5 spec." → **interface-cli.md § Surface / interface-spec.md § Surface / plan.md § Integration Design**: `my actions` maps to `GET /me/actions` (`listMyActions`), confirmed present in `spec/glassfrog-api-v5.yaml:1092-1094`. PASS.
- **C-I.2** | "The CLI MUST NOT invent endpoints, parameters, or behaviors the spec does not define." → **interface-cli.md § Flags / spec.md § Assumptions**: the only request parameter is `?status=`, whose 7-value set (`archived, cancelled, completed, current, scheduled, someday, waiting`) matches the spec enum verbatim (`spec/glassfrog-api-v5.yaml:1102-1114`). No invented endpoint or parameter. PASS.

**Principle II — Action Transparency (NON-NEGOTIABLE, MUST, P0)** — 3/3
- **C-II.1** | "Every action MUST report the spec operation it invoked and the target resource, in machine-parseable form." → **interface-cli.md § Output**: the projection renders one entry per action with the `actn_…` id always present (the machine-actionable handle), and `interface-spec.md` pins the read to the `GET /me/actions` operation. PASS.
- **C-II.2** | "Every error MUST explain what went wrong and the next step." → **interface-cli.md § Error Communication**: each error class states a next step (no-token → "run `glassfrog auth login` or set `GLASSFROG_TOKEN`"; bad base URL → name source + correction; transport → name cause + "check connectivity"; unsupported `--status` → name value + supported set). PASS.
- **C-II.3** | "The operator MUST always be able to tell what the CLI did." → **interface-cli.md § Error Communication / interface-spec.md § Error Communication**: the outcome class is signalled via the exit code on every path (Success 0 / UsageError 2 / RuntimeError 1 / APIError 3 / NetworkUnavailable 6); stdout/stderr separation pinned. PASS.

**Principle III — Fail Safe, Not Silent (MUST / MUST NOT + anti-patterns, P0)** — 3/3
- **C-III.1** | "The CLI MUST validate a write before sending it." (read-analogue: validate input before issuing the request) → **spec.md § Filtering / plan.md ADR-2 / interface-spec.md § Interactions**: an unsupported `--status` is rejected by `validateStatus` *before* context assembly or any request; a transport tripwire asserts nothing was sent. PASS (no writes; the validate-before-I/O discipline is applied to the one request-shaping input).
- **C-III.2** | "Errors MUST be obvious and recoverable, never hidden" / anti-pattern "swallowing errors" → **plan.md § Cross-cutting (Error handling) / interface-spec.md § Error Communication**: every fork fails loud — base-URL error refuses at `NewClientFromOS`, non-2xx is never treated as success (APIError 3), an undecodable 2xx is a loud RuntimeError 1 (never a zero-valued projection), transport failure is NetworkUnavailable 6. PASS.
- **C-III.3** | Anti-pattern "failure condition reported as success" → **interface-cli.md § Error Communication ("Never zero on failure") / interface-spec.md ("Fail-safe: default→1")**: any non-success outcome exits non-zero; an unmapped Outcome falls through to 1. PASS.

**Principle IV — Test-Driven Development (MUST, P0)** — 2/2
- **C-IV.1** | "User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it." → **features/self-service-reads/my-actions.feature**: every behavioral form in the accord has a Gherkin scenario under a Rule block (list, empty, no-token, non-2xx, transport, decode-fail, bad-base-URL, supported-status, unsupported-status, more-available). PASS.
- **C-IV.2** | "Features MUST be built test-first: RED before GREEN." → **plan.md § Cross-cutting (Testing) / tasks.md T001/T002/T003/T004**: every task is specified "RED-first"; T004 makes scenarios pass as executable acceptance and removes `@wip` only from behavioral scenarios. PASS.

**Principle V — Composition over Monolith (MUST / MUST NOT + anti-patterns, P0)** — 2/2
- **C-V.1** | "The CLI MUST be built from modular, independently-testable parts (per-resource command modules over a shared API client)." → **plan.md § System Architecture / ADR-4 / interface-spec.md § Surface**: `my actions` is a self-contained leaf in `internal/cli/my_actions.go` with a pure `runMyActions`/`formatMyActions`/`validateStatus` trio behind an injected seam; `Action` is a schema-only addition to `internal/glassfrog`. PASS.
- **C-V.2** | "Adding a new command MUST NOT require changing unrelated ones." → **plan.md § System Architecture ("013 introduces no new infrastructure") / tasks.md T004**: 013 adds one resource model, one leaf, one validator, and a single `MustRegister` wiring line under the `my` parent; it reuses 011/012 surfaces unchanged and adds no Outcome/ExitCode case. No unrelated command is edited. PASS.

**Principle VI — Size-Aware by Design (MUST / MUST NEVER, P0)** — 2/2
- **C-VI.1** | "MUST NEVER silently truncate." → **spec.md § Pagination boundary / interface-cli.md § Output / plan.md ADR-3**: when `meta.pagination.has_next_page` is true the projection appends a "more results available" signal — the rejected option "render the page silently with no signal" is explicitly the silent-truncation anti-pattern this guards against. PASS.
- **C-VI.2** | "When results are paged or capped, it MUST page through them OR clearly signal the boundary." → **spec.md § Pagination boundary / interface-spec.md § Interactions**: the command makes exactly one request and clearly signals the boundary (the page-walk itself is deferred to Pagination 016 — the principle's disjunctive "OR clearly signal" branch is satisfied). PASS.

**Principle VIII — No Fabricated Data (MUST / MUST NOT, P0)** — 1/1
- **C-VIII.1** | "MUST present only data the API actually returned; MUST NOT invent, guess, or fill placeholder values." → **interface-cli.md § Output / interface-spec.md § Surface (Action)**: the projection renders response-side fields only; a nullable `description` renders an explicit `—` placeholder for *absence* (not a fabricated value), and `formatMyActions` is a pure renderer over the decoded `Action`. An empty result prints an explicit empty line, not a synthesized row. PASS.

**Principle IX — Writes Require Explicit Intent (MUST NEVER, P0)** — 1/1
- **C-IX.1** | "A read-shaped command (get/list/show) MUST NEVER mutate as a side effect." → **spec.md § Non-Behaviors / plan.md § System Architecture / interface-spec.md**: `my actions` issues a single `GET /me/actions` and nothing else; spec Non-Behavior explicitly forbids create/update/mutate. No POST/PATCH/DELETE on any path. PASS.

**Principle VII — Working Software (MUST, P1)** — 1/1
- **C-VII.1** | "Every commit/PR MUST include implementation together with its tests, and MUST validate and build." → **tasks.md T001-T004 acceptance criteria**: every task pairs implementation with RED-first tests and asserts `go build ./...` / `go vet ./...` clean; no code-only or test-only increment is specified. PASS (artifact-level; full enforcement is commit-time).

**Principle X — Respect API Limits (SHOULD-grade for this read, P1)** — 1/1
- **C-X.1** | "MUST honor rate limits (back off on 429); use If-Match/ETag for updates." → **spec.md § System Overview + Non-Behaviors / plan.md § Risks**: `If-Match`/`ETag` is inapplicable (read, no update). 429 back-off is explicitly deferred to Rate-Limit Handling (017); 013 surfaces a non-2xx (incl. 429) as a generic APIError(3) without claiming to handle it, and the deferral is stated as an intentional Non-Behavior. No violation — the principle's write/back-off obligations are correctly out of this read's scope and the boundary is declared. PASS. *(See Governance Notes: the active-handling clause has no applicable check here.)*

**Principle XII — Standalone Executable (MUST, P1)** — 1/1
- **C-XII.1** | "MUST run as a self-contained executable; only assumed external dependency is network access to the Glassfrog API." → **plan.md § System Architecture / interface-spec.md § Surface**: 013 adds only Go source to existing `internal/cli` + `internal/glassfrog` packages, introduces no new runtime/interpreter/service dependency, and its sole external dependency is the `GET /me/actions` call. The package "has no new internal imports" (tasks.md T001). PASS (artifact-level; build-time artifact check is post-implementation).

---

## Cross-Reference Checks

Not generated as a separate category: cross-reference checks derive from done-* accords that require inter-artifact links, and no done-* accords are present. Link presence between tasks and their sources was nonetheless observed during evaluation and is sound — every task in tasks.md carries Plan / Interface / Scenario reference fields, and T001/T003 cite specific my-actions.feature scenarios. (Full inter-artifact consistency is analyze's domain.)

---

## Governance Notes

- **done-specify.md / done-plan.md / done-interface.md / done-scenarios.md / done-tasks.md**: Not found — no `accords/governance/` directory exists in this repository. Consider creating `accords/governance/done-<skill>.md` accords to enable done-criteria quality checks (currently this checklist is constitution-only). This is the dominant coverage gap: the vertical "does each artifact meet its own bar" dimension is checked only against the constitution, not against per-skill done-criteria.
- **Principle XI — Governance via Proposals**: No applicable checks for this feature. `my actions` is a read over operational `/me/actions`; it mutates no governance structure and exposes no proposal path, so the proposal-gating detection mechanism has nothing to evaluate.
- **Principle X — Respect API Limits (active-handling clause)**: The 429-back-off and `If-Match`/`ETag` obligations produce no applicable *failure* check here — both are inapplicable or intentionally deferred (017) for this first-page read. Recorded as a pass with the deferral declared, not a gap.
- **Principles VII (Working Software) and XII (Standalone Executable)** are graded at the artifact level only. Their detection mechanisms are fundamentally commit-time / build-artifact checks (CI build, clean-environment run) and cannot be fully discharged before implementation.
- **Severity note**: Principle X is treated as P1 for this read because its binding obligations (back-off, optimistic concurrency) are write/limit-oriented and out of this capability's scope; principles VII and XII are P1 because their pre-implementation artifact evidence is necessary-but-not-sufficient. All MUST/MUST-NOT principles with in-scope behavioral obligations (I, II, III, IV, V, VI, VIII, IX) are P0.
