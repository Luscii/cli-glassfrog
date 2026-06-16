# Checklist: Plan-Limit Signal

**Feature**: 061-plan-limit-signal
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, interface-spec.md, tasks.md (scenarios live at `features/unsignalled-plan-limits/plan-limit-signal.feature`)
**Checks**: 13 (13 pass, 0 fail) — 2 principles produced zero applicable checks (N/A)
**Generated**: 2026-06-15

---

## Summary

All 13 checks pass. Constitution: 13/13. (No `done-*` accords present — done-criteria and cross-reference checks not run; see Governance Notes.)

This feature is the **legibility half** of Plan-Limit Signalling — it shapes a recognized plan-gate `403` into an actionable diagnostic and renders it across formats. So unlike its recognizer sibling (060), the principles that govern **output legibility** bite hardest and are **positively upheld**: **II (Action Transparency)** — the whole point is to make a plan-gate `403` explain what went wrong and the next step, with a machine-parseable `feature` element — and **VIII (No Fabricated Data)** — the possibility-not-certainty wording and the no-invented-remedy decision (ADR-3/ADR-4). The write/limit-facing principles have a narrow or empty trigger: **VI (Size-Aware)** and **X (Respect API Limits)** produced **zero applicable checks** (no list/tree surface; no request issued — it interprets an already-arrived `403`); **III / IX / XI** were calibrated to the failure-rendering scope (no write, no mutation, no governance command). **The exit code is unchanged (still `PermissionError`/4)**, so II's no-regression edge holds by construction.

---

## Constitution Checks: 13/13 passed

### Calibration notes (applied before evaluation)

- **Principle II (Action Transparency)**: directly triggered — 061 renders a failure. Calibrated to: *a recognized plan-gate `403` now explains what went wrong (may not be available on the plan, naming the gating feature) and the next step (verify the plan), in machine-parseable form (the distinct `feature` envelope element); and 061 regresses no existing legibility — the category and exit code are unchanged, and non-recognized failures render byte-identically.* Evaluated and passes.
- **Principle III (Fail Safe, Not Silent)**: the "validate a write / no partial governance state" clauses are **N/A** (061 issues no write). Calibrated to: *the failure is surfaced loudly and clearly in every format; a `403` 061 does not recognize stays its existing surfaced failure — refinement only adds information, never hides one.* Evaluated and passes.
- **Principle VI (Size-Aware by Design)**: 061 renders one diagnostic for one failure; it issues no list, walks no tree, pages nothing. Per Step 5, no applicable surface → **zero checks**. Recorded as a Governance Note.
- **Principle IX (Writes Require Explicit Intent)**: calibrated — 061 operates on the **failure path** (reading a typed error and shaping a diagnostic); it performs no write or mutation and has no side effect. The `Method`/`Path` enrichment is read-only request metadata captured at the source. Evaluated and passes.
- **Principle X (Respect API Limits)**: 061 **issues no request** — it interprets a `403` that already arrived, sends no `If-Match`, triggers no retry (spec Non-Behavior: "must not call the API to probe plan status"). Both the `429`-backoff and `If-Match` clauses have no trigger → **zero applicable checks**. Recorded as a Governance Note.
- **Principle XI (Governance via Proposals)**: calibrated to "introduces no governance-mutating command path." 061 adds no command or flag at all — only diagnostic-chain edits — so it trivially upholds it. Evaluated and passes.

### Passed (13/13)

**P0** | CONSTITUTION.md I (Spec Fidelity): 061 invents no operation, endpoint, parameter, or behavior.
→ **plan.md System Architecture / ADR-2, interface-cli.md**: 061 adds no command and no flag; it refines an existing `403` failure's wording and adds one envelope key. The gating feature it names is the recognized `Gate`'s display name (from 060, traced to the spec's Premium/`x-feature-gate` metadata); no endpoint or parameter is introduced.

**P0** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): a plan-gate `403` now explains what went wrong and the next step.
→ **spec.md § Behavioral Accord, interface-cli.md (Surface / Error Communication)**: the diagnostic states the operation may not be available on the plan, names the gating feature, and gives a verify-the-plan next step — fulfilling "every error MUST explain what went wrong and the next step." The `feature` element is its own parseable key (machine-parseable form), and the exit code is unchanged, so no existing legibility regresses.

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): the failure is surfaced clearly, never hidden.
→ **spec.md § Behavioral Accord ("Staying in its lane"), interface-cli.md (Error Communication)**: a recognized plan-limit `403` is rendered loudly in every format with its exit code; a `403` 061 does not recognize stays today's surfaced failure. Refinement only adds information. Write/partial-state clauses N/A (no write).

**P0** | CONSTITUTION.md IV (TDD Red→Green): the signal is built test-first with executable acceptance.
→ **tasks.md T001/T002 (unit tests with the code), T003 (godog acceptance over the .feature, `@wip`→un-`@wip`), plan.md Cross-cutting (testing)**: failing tests precede implementation, and the driving scenarios have an executable suite before/with the code that satisfies them.

**P0** | CONSTITUTION.md V (Composition over Monolith): the refinement is a central, additive edit that entangles no unrelated code.
→ **plan.md ADR-1/ADR-2, interface-spec.md (no call-site threading)**: 061 refines the single `Diagnose` chokepoint and adds additive fields; the ~100 `reportFailure` call sites and every unrelated command are **untouched** (operation identity is threaded at the source, not via a signature change). Adding the signal requires changing no unrelated command — the central-site choice is the composition-respecting one (vs per-command interception).

**P0** | CONSTITUTION.md VII (Working Software): implementation and tests ship together and build/validate.
→ **tasks.md T001/T002/T003 acceptance criteria**: each task bundles code with its tests (T001/T002) or step definitions with passing scenarios (T003); all require `go build ./...` and `go vet ./...` clean. No code-only or test-only increment.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): the wording expresses possibility, never a fabricated certainty.
→ **spec.md § Non-Behaviors, plan.md ADR-3 (wording) / consuming 060 ADR-4**: the cause frames the plan limit as *possible* (notes it may instead be a permission issue) and the next step is to *verify*, never to *upgrade* — it never asserts a plan the API never confirmed is insufficient. A validation scenario holds this boundary.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): no invented remedy detail beyond the recognized gate name.
→ **spec.md § Non-Behaviors, plan.md ADR-3, interface-cli.md**: the diagnostic names no plan price, no upgrade URL, and no plan name beyond the gating feature's display name (which comes from the recognized `Gate`). No placeholder or guessed value is filled.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): 061 performs no write or mutation.
→ **plan.md System Architecture / Cross-cutting, spec.md § Non-Behaviors**: 061 acts on the failure path — reading a typed error and shaping a diagnostic. It issues no API call and has no side effect; a read-shaped command's failure rendering mutates nothing.

**P0** | CONSTITUTION.md XI (Governance via Proposals): 061 introduces no governance-mutating command path.
→ **tasks.md (adds no command, no flag, no Outcome/ExitCode)**: the feature adds only diagnostic-chain code; it exposes no command path that could mutate governance, directly or otherwise.

**P0** | CONSTITUTION.md XII (Standalone Executable): 061 adds no external dependency.
→ **tasks.md T001–T003, plan.md**: the new fields, the `Diagnose` branch, the display-name mapping, and the envelope key use only the Go standard library and existing internal packages (`apiclient`/`cli`/`output`) — no new runtime or third-party dependency.

**P0** | CONSTITUTION.md II/VIII (token & body hygiene): the diagnostic and envelope never emit the token, and 061 does not inspect the body to decide a plan-gate.
→ **plan.md Cross-cutting (secret hygiene), spec.md § Non-Behaviors, interface-cli.md / interface-spec.md**: the wording is static literals plus the gate display name; the inputs are the request method/path and response-side status/body; the gate decision comes from 060's recognizer (operation + status), not from inspecting the body. No `X-Auth-Token` can appear in any rendered failure.

**P0** | CONSTITUTION.md I/VIII (boundary fidelity): the modeled-but-dormant `ai_integration` gate matches the spec's deferred scope.
→ **spec.md § edge scenario / Non-Behaviors, plan.md ADR-3 ("What This Plan Does Not Cover")**: 061 maps `GateAIIntegration`'s display name for readiness, but no command reaches it today, so it produces no message — active scope matches PROJECT scope, neither under- nor over-claiming.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords present** (`accords/governance/` does not exist). Done-criteria and cross-reference checks were not run. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria quality checks across the pipeline. (Same gap noted on sibling specs — project-wide, not specific to 061.)
- **Principle VI (Size-Aware by Design)** produced **zero applicable checks** — 061 renders one diagnostic per failure; it has no list/tree/pagination surface.
- **Principle X (Respect API Limits)** produced **zero applicable checks** — 061 issues no request (it interprets an already-arrived `403`), so neither the `429`-backoff nor the `If-Match` clause has a trigger.
- **Cross-spec dependency (not a quality finding)**: 061's T002/T003 consume 060's `RecognizeFeatureGate`, now **landed on `main`** (#142) — the dependency is satisfied. Recorded for traceability, not a constitution violation.
