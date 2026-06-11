# Checklist: Tension Capture

**Feature**: 042-tension-capture
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/tension-capture/tension-capture.feature, tasks.md
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-06-11

---

## Summary

All 16 checks pass. Constitution: 16/16. Done-criteria: not run (no `done-*` accords). Cross-references: not run (no `done-*` accords).

One principle (VI Size-Aware) produced zero applicable checks for this feature; two principles (X, XI) pass on applicability grounds with calibration notes below.

---

## Constitution Checks: 16/16 passed

### Passed

**P0** | I. Spec Fidelity — "Every command MUST map to an operation defined in the spec"
→ **interface-cli.md / interface-spec.md**: `tension create` maps to `createTension` (`POST /roles/{role_id}/tensions`); the request body uses only spec-defined `TensionInput.tension` fields (`body`/`label`/`meeting_type`). `status` and `sensed_by` are deliberately **not** sent (server-owned) — no invented endpoint, parameter, or behavior. Pass (×2: operation mapping + no-invented-params).

**P0** | II. Action Transparency (NON-NEGOTIABLE) — "report the spec operation and target resource, machine-parseable; every error explains cause + next step"
→ **interface-cli.md § Output / Error Communication**: success produces the created tension incl. its `ten_` id, machine-parseable under `json`/`yaml`; the Error Communication table gives every failure a named cause **and** a concrete next step, and never prints the token. Pass (×2: success traceability + error cause/next-step).

**P0** | III. Fail Safe, Not Silent — "validate a write before sending; never partial state"
→ **spec.md § Behavioral Accord / interface-cli.md § Interactions / tasks.md T004**: `--body` non-empty and `--meeting-type` are validated fail-fast **before any request** (transport tripwire asserts no call on rejection); capture is a single `POST` with no multi-step partial-application surface; failures are surfaced, never swallowed. Pass (×2: pre-send validation + no-partial-state).

**P0** | IV. Test-Driven Development — "test-first; user-facing behavior has an executable acceptance scenario"
→ **tasks.md (T001–T005) / tension-capture.feature**: every task mandates RED-first unit tests; T005 makes the driving scenarios executable acceptance; the feature file exists with `@wip` scenarios held for that step. Pass (×2: test-first mandate + acceptance scenarios present).

**P0** | V. Composition over Monolith — "modular per-resource modules; adding a command MUST NOT require changing unrelated ones"
→ **plan.md § System Architecture / tasks.md**: a new `internal/cli/tension.go` module over the shared client; no edits to any unrelated command. The one shared touch (`apiclient.Request.ContentType`, T001) is **additive and backward-compatible** — existing reads pass `""` and are byte-identical — so it forces no change to unrelated commands. Pass.

**P0** | VII. Working Software — "implementation together with its tests; validates and builds"
→ **tasks.md (T001–T005)**: each task bundles implementation with its tests and lists `go build`/`go vet` clean in acceptance — no code-only or test-only increment outside the RED→GREEN pair. Pass.

**P0** | VIII. No Fabricated Data — "present only data the API returned; no invented/placeholder values"
→ **interface-cli.md § Output / tasks.md T003**: the `tension` render key uses explicit-absence guards (`(none)`) for every nullable field and renders only the API-returned tension; a long `body` is rendered verbatim, never truncated or invented. Pass.

**P0** | IX. Writes Require Explicit Intent — "no mutation except via an explicit write command; a read MUST NEVER mutate"
→ **spec.md § Non-Behaviors / interface-cli.md**: `tension create` is an explicit write command; the feature adds no read path, and the spec forbids reads mutating. Intent is the command itself (constitution's conflict-resolution: no interactive prompt needed). Pass.

**P0** | X. Respect API Limits — "back off on 429; use If-Match/ETag for updates"
→ **plan.md § Cross-cutting / interface-spec.md**: the capture reuses the landed `RetryExecutor` (017), which honors `429`/`Retry-After`; for the non-idempotent `POST` the `isSafeMethod` gate surfaces the `429` on first occurrence rather than retrying — there is **no retry loop that ignores 429** (the principle's stated anti-pattern). The If-Match clause targets **updates**; capture is a **create** with no prior `ETag` to guard, and the plan scopes `If-Match` out (Clobbered Changes, deferred). Pass — see Calibration note 1.

**P0** | XI. Governance via Proposals — "governance-structure mutations go through a Proposal; no default path mutating governance structure directly"
→ **spec.md § System Overview / PROJECT.md**: a tension is an **operational** resource and the *seed* of a proposal — not governance structure (roles/accountabilities/domains/policies). Capturing a tension mutates no governance structure, so the opt-in-flag requirement does not apply; the spec positions capture as the entry point **to** the proposal flow, not a bypass. Pass — see Calibration note 2.

**P0** | XII. Standalone Executable — "self-contained; no pre-installed dependencies beyond network"
→ **plan.md / tasks.md**: 042 adds only Go code (the additive `ContentType` field, stdlib JSON marshalling, a new model/render key) — it introduces no language runtime, service, or library that must be installed first. Pass.

---

## Governance Infrastructure Notes

*(Separate from the feature quality findings above.)*

- **No `done-*` accords found** — `accords/governance/` does not exist, so **done-criteria checks and cross-reference checks were not generated**. This is a **project-wide** condition (no spec in the repo has these accords), not specific to 042. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to unlock done-criteria checks for every spec; until then, checklist evaluates constitution principles only.
- **`agents/guardian-agent.md` not deployed** — the checklist ran on its SKILL.md process alone (reduced character consistency, not a blocked skill).

## Calibration Notes

1. **Principle X applied to a create.** X reads naturally for *updates* (back-off + `If-Match`). For 042's `POST` create, the binding parts are: (a) 429 must not be ignored — satisfied, the 429 is surfaced (and a non-idempotent retry is deliberately avoided to prevent double-create, which the constitution's Fail-Safe-wins conflict rule supports); (b) `If-Match` — not applicable, a create has no prior `ETag`. Recorded as a pass rather than a forced severity. The architecture-informed scenario "A rate-limited capture is surfaced, not silently re-sent" makes this behavior executable.
2. **Principle XI scope.** XI governs *governance structure*. A tension is operational (the proposal seed), so capture is outside XI's mutation set — no opt-in flag is required or expected. If a future spec routed a governance-structure change through this path, XI's opt-in-flag requirement would re-engage.
3. **Principle VI (Size-Aware) produced zero applicable checks** — capture is a single-resource create with no list or pagination, so there is nothing to truncate.
