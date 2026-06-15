# Checklist: Withdraw Proposal

**Feature**: 059-withdraw-proposal
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/proposal-write-flow/withdraw-proposal.feature, tasks.md
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-06-15

---

## Summary

All 16 checks pass. Constitution: 16/16. (No `done-*` accords present — done-criteria and cross-reference checks not run; see Governance Notes.)

Two principles required calibration for a single-resource governance *transition*: **VI (Size-Aware)** produced no applicable checks (no list/tree surface), and **X (Respect API Limits)** — its `If-Match` clause is N/A for a transition endpoint that defines no `If-Match` parameter; its `429` clause is checked and passes. Two principles carry a withdraw-specific note: **III (Fail Safe)** because the withdraw is destructive (the server deletes existing responses), and **IX (Writes Require Explicit Intent)** because that destructive write deliberately carries no confirmation/`--force` — a stance the constitution's own conflict-resolution rule endorses.

---

## Constitution Checks: 16/16 passed

### Calibration notes (applied before evaluation)

- **Principle VI (Size-Aware by Design)**: Withdraw Proposal is a single-resource transition (`POST …/withdraw` → one `Proposal`); it issues no list and walks no tree. Per Step 5, a principle with no applicable surface produces **zero checks** for this feature. Recorded as a Governance Note, not a finding.
- **Principle X (Respect API Limits)**: split into two binary assertions for this feature — (X-a) `429` is honored via the shared retry/rate-limit handling, and (X-b) optimistic concurrency. (X-b) `If-Match` is **N/A**: `withdrawProposal` is a state transition that defines no `If-Match` parameter and involves no read-modify-write of mutable fields — there is no `ETag` to send back. The "update that omits `If-Match` when an `ETag` is available" detection has no trigger here. (X-a) is evaluated and passes; (X-b) is recorded as N/A with reasoning rather than passed silently.

### Passed (16/16)

**P0** | CONSTITUTION.md I (Spec Fidelity): the command maps to a defined spec operation and invents nothing.
→ **spec.md § Behavioral Accord / Integration Boundaries, plan.md ADR-1, interface-cli.md § Surface**: `glassfrog proposal withdraw <prp-id>` → `POST /proposals/{proposal_id}/withdraw` (`withdrawProposal`), bodyless, path id only — no invented endpoint, parameter, or behavior. The `circulating → draft` direction, transition semantics, destructive side effects, and Premium gate trace to `spec/glassfrog-api-v5.yaml` (`withdrawProposal`, ~line 3829).

**P0** | CONSTITUTION.md II (Action Transparency): the action reports the operation + target in machine-parseable form.
→ **interface-cli.md § Surface / Interactions**: success renders the returned `{data: Proposal}` (the resource acted on, by `prp_` id, now `draft`) in `json`/`yaml`/`full`/`compact`; the command produces structured data, never free-form-only text.

**P0** | CONSTITUTION.md II (Action Transparency): every error explains cause + next step.
→ **spec.md § Failure, interface-cli.md § Error Communication**: the error table names the HTTP status (404/422/403/401/429) and a next step per class, routed through the shared `reportFailure`; spec mandates "names both what went wrong and a concrete next step … never includes the token."

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): errors are obvious and not swallowed; no partial-write state — even though the withdraw is destructive.
→ **plan.md ADR-1/ADR-2, spec.md § Failure / Non-Behaviors**: the withdraw is a single atomic `POST` — the destructive side effects (responses deleted, `proposed_at`/`response_deadline` cleared) are applied **server-side, atomically, within the one transition**, so the client never leaves governance in a partially-applied state. `404`/`422` are surfaced as real failures (exit non-zero), explicitly **not** folded into success — the inverse of discard's `404`-as-success. No failure-reported-as-success path.

**P0** | CONSTITUTION.md IV (TDD Red→Green): test-first, with an executable acceptance scenario before the code.
→ **tasks.md T001/T002, withdraw-proposal.feature**: T001 is "RED-first unit tests for every branch"; T002 makes the `.feature` scenarios executable acceptance; the feature file exists with `@wip` behavioral scenarios.

**P0** | CONSTITUTION.md V (Composition over Monolith): adding the command does not change unrelated commands.
→ **plan.md § System Architecture / Cross-cutting, tasks.md (branching/role-based awareness)**: 059 adds a new `proposal_withdraw.go`, attaches one leaf to the shared `proposal` group, and adds its own `run` function; it modifies no other command (the five landed leaves are untouched), adds no shared-state coupling.

**P0** | CONSTITUTION.md VII (Working Software): implementation ships with tests, validates and builds.
→ **tasks.md T001/T002 acceptance criteria**: each task requires `go build`/`go vet` clean and passing tests alongside the code; no code-only/test-only increment.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): only data the API returned is presented.
→ **plan.md ADR-1/ADR-2, spec.md § Output / Non-Behaviors, withdraw-proposal.feature (validation)**: the success decodes-and-renders the server's `draft` `Proposal` (machine format emits raw bytes verbatim), synthesizes nothing, and narrates nothing about the deleted responses; validation scenario "The withdrawn result is the server's proposal unembellished" pins it.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): mutation only via an explicit write command; no read-shaped mutation.
→ **spec.md § Invocation, plan.md ADR-1, interface-cli.md § Interactions**: `withdraw` is itself the explicit write verb; it is not a read-shaped command. It issues no prior `GET` (no read that mutates), and a validation scenario asserts exactly one request and no prior read.

**P0** | CONSTITUTION.md X-a (Respect API Limits — rate limits): `429` is honored, not ignored.
→ **plan.md § Cross-cutting (non-idempotent retry), interface-cli.md § Error Communication**: `429` → `RateLimited(5)` via the shared classifier; the `POST` is not auto-retried (017 `isSafeMethod` GET/HEAD-only), so it surfaces once rather than hammering — and a succeeded withdraw cannot double-fire.

**P0** | CONSTITUTION.md XI (Governance via Proposals): no default path mutates governance structure directly.
→ **spec.md § System Overview / Non-Behaviors, PROJECT.md Domain**: withdrawing a proposal *is* the sanctioned proposal flow — it returns a circulating proposal to `draft` for re-editing; it does not mutate roles/accountabilities/domains/policies directly and exposes no governance-bypass path. Fully aligned with the proposal-gated model.

**P0** | CONSTITUTION.md XII (Standalone Executable): no new runtime/dependency assumption introduced.
→ **plan.md § System Architecture / Implementation Strategy**: 059 adds one cobra leaf to the existing Go binary; it introduces no new transport, library, or runtime dependency — self-containment is preserved.

**P0** | CONSTITUTION.md I (Spec Fidelity — request shape): the request sends only spec-defined fields.
→ **plan.md ADR-1, interface-cli.md § Interactions**: bodyless `POST` (the endpoint defines no `requestBody`), no `If-Match` (the endpoint defines no such parameter), `out == Document[Proposal]` — the request carries nothing the spec doesn't define.

**P0** | CONSTITUTION.md VIII (No Fabricated Data — failure path): no fabricated detail on failure.
→ **interface-cli.md § Error Communication**: failures surface the API's extracted RFC 9457 `detail`; the Premium `403` is a generic refusal with **no** invented "not available on your plan" message (deferred to Plan-Limit Signalling) — no fabricated explanation.

**P0** | CONSTITUTION.md III (Fail Safe — pre-request fail-fast): invalid invocation costs no request.
→ **spec.md § Invocation, interface-cli.md § Interactions, withdraw-proposal.feature**: a missing/extra positional, an unknown flag, or an invalid `--output` is a `UsageError(2)` with **no request** (transport tripwire); resolve-first ordering pins it.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent — non-interactive intent for a destructive write): intent is the command itself, no hidden mutation, no interactive prompt.
→ **spec.md § Non-Behaviors, plan.md ADR-2, interface-cli.md § Surface / Interactions**: although the withdraw is destructive (it deletes existing responses), the command adds **no** confirmation prompt and **no** `--force`/`--yes` flag — the command's existence is the intent, exactly per the constitution's conflict-resolution rule ("Intent is expressed by the explicit write command/flag itself, not by an interactive prompt"). No side effect beyond the named transition; operator-layer confirmation is deferred to the separate Write-Safety Guardrail. Constitution-aligned.

---

## Governance Notes

- **`accords/governance/done-*.md`**: Not found (no `accords/` directory). Done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were **not run** — only constitution checks. Consider creating these accords to enable per-artifact done-criteria checking. This conforms to the in-repo precedent (045/056/057/058 ran constitution-only as well).
- **Constitution principle VI (Size-Aware by Design)**: No applicable checks for this feature — the withdraw is a single-resource transition with no list or tree surface to paginate. (Pagination/size-awareness is exercised by the read specs, e.g. 056.)
- **Constitution principle X-b (optimistic concurrency / `If-Match`)**: N/A — `withdrawProposal` is a transition endpoint with no `If-Match` parameter and no read-modify-write cycle; there is no `ETag` to echo. The principle's update-clobbering detection has no trigger here. (Guarded Writes' `Request.IfMatch` field stays unused, per plan/053.)
