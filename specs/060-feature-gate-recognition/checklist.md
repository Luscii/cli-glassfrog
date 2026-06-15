# Checklist: Feature-Gate Recognition

**Feature**: 060-feature-gate-recognition
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, tasks.md (no interface-*.md or .feature in the spec dir — the scenarios live at `features/unsignalled-plan-limits/feature-gate-recognition.feature`)
**Checks**: 13 (13 pass, 0 fail) — 2 principles produced zero applicable checks (N/A)
**Generated**: 2026-06-15

---

## Summary

All 13 checks pass. Constitution: 13/13. (No `done-*` accords present — done-criteria and cross-reference checks not run; see Governance Notes.)

This feature is a **pure internal recognizer** — no command, no request, no output, no write, no wiring. Several principles that govern action/output/write surfaces therefore have a narrow or empty trigger here: **VI (Size-Aware)** and **X (Respect API Limits)** produced **zero applicable checks** (no list/tree surface; no request issued), and **II / III / IX / XI** were calibrated to the recognition-only scope (the action-reporting, partial-write, mutation, and governance-command clauses have no trigger; the non-regression and read-only clauses are checked and pass). The principle that bites hardest — **VIII (No Fabricated Data)** — is squarely upheld by the possibility-not-certainty decision (ADR-4).

---

## Constitution Checks: 13/13 passed

### Calibration notes (applied before evaluation)

- **Principle II (Action Transparency)**: #60 performs no action and renders nothing, so the "report the operation + target" clause has **no trigger**. Calibrated to the binary assertion: *#60 alters no rendered message and no exit code (a non-behavior), so it regresses no existing failure legibility, and the classification it produces is the enabling input for #61's clearer next-step.* Evaluated and passes.
- **Principle III (Fail Safe, Not Silent)**: the "validate a write / no partial governance state" clauses are **N/A** (#60 issues no write). Calibrated to the assertion that recognition **never hides a failure** — a 403 it declines to recognize stays a generic surfaced failure. Evaluated and passes.
- **Principle VI (Size-Aware by Design)**: the recognizer reads only a single (method, path, status); it issues no list, walks no tree, and pages nothing. Per Step 5, a principle with no applicable surface produces **zero checks**. Recorded as a Governance Note, not a finding.
- **Principle IX (Writes Require Explicit Intent)**: calibrated — #60 does no I/O at all (a pure function over an already-received error), so it neither writes nor mutates as a side effect. Evaluated and passes.
- **Principle X (Respect API Limits)**: the recognizer **issues no request** — it interprets a 403 that already arrived, sends no `If-Match`, and triggers no retry (reinforced by the spec Non-Behavior "must not call the API to probe plan status"). Both the `429`-backoff and `If-Match` clauses have **no trigger**: **zero applicable checks**. Recorded as a Governance Note.
- **Principle XI (Governance via Proposals)**: calibrated to "introduces no governance-mutating command path." #60 adds no command at all, so it trivially upholds it. Evaluated and passes.

### Passed (13/13)

**P0** | CONSTITUTION.md I (Spec Fidelity): the recognizer invents no operation or gate — every registered row and gate kind traces to the spec.
→ **plan.md ADR-2/ADR-3, spec.md § Integration Boundaries / System Overview**: the static registry holds exactly the four Premium async-proposal write operations documented in `spec/glassfrog-api-v5.yaml` ("requires async proposals … Returns 403"); `GateAIIntegration` traces to the `x-feature-gate: ai_integration` vendor extension. No endpoint, parameter, gate, or behavior is invented.

**P0** | CONSTITUTION.md II (Action Transparency, NON-NEGOTIABLE): #60 regresses no failure legibility and changes no exit code.
→ **spec.md § Non-Behaviors, plan.md ADR-1**: recognition "renders no message, changes no exit code, alters no error chain"; it produces only a classification that #61 consumes to make a plan-gated 403's next-step legible. No transparency regression; the action-reporting clause has no trigger (no action).

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): recognition never hides or swallows a failure.
→ **spec.md § Behavioral Accord ("Declining to recognize"), plan.md ADR-1**: a 403 from a non-gated operation, or a non-403 from a gated one, is **not** recognized and **stays a generic surfaced failure** — recognition only adds information, never suppresses it. Write/partial-state clauses N/A (no write).

**P0** | CONSTITUTION.md IV (TDD Red→Green): the recognizer is built test-first with executable acceptance.
→ **tasks.md T001 (RED-first table-driven unit tests), T002 (godog acceptance over the .feature, `@wip`→un-`@wip`), plan.md Cross-cutting**: a failing test precedes implementation, and the driving scenarios have an executable suite before/with the code that satisfies them.

**P0** | CONSTITUTION.md V (Composition over Monolith): the recognizer is a modular, independently-testable part added without entangling unrelated code.
→ **plan.md ADR-1, tasks.md Branching Guidance / T001 Scope**: a **new file** `internal/apiclient/featuregate.go` (sibling to `problem.go`) that touches no existing type, no sibling spec, and no call site. Adding it requires changing nothing unrelated.

**P0** | CONSTITUTION.md VII (Working Software): implementation and tests ship together and build/validate.
→ **tasks.md T001/T002 acceptance criteria**: T001 bundles the recognizer with its unit tests; T002 bundles step definitions with passing scenarios; both require `go build ./...` and `go vet ./...` clean. No code-only or test-only increment.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): recognition expresses possibility, never a fabricated certainty.
→ **spec.md § Non-Behaviors + edge scenario, plan.md ADR-4**: a non-`None` `Gate` means the 403 is *consistent with* a known plan gate — a **suspicion**, never "the 403 *is* a plan limit" (a cause the API never confirmed). It never claims certainty for a genuine permission-denial 403 on a gated op.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): the gate metadata is transcribed from the spec, not invented.
→ **plan.md ADR-2/ADR-3**: registry rows and gate kinds are a literal transcription of the spec's documented Premium/`x-feature-gate` metadata; no gate, operation, or status meaning is guessed or placeholder-filled.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): the recognizer performs no write or mutation.
→ **plan.md ADR-1, spec.md § Non-Behaviors**: a pure, total function over an already-received error — no I/O, no API call, no side effect. It cannot mutate anything.

**P0** | CONSTITUTION.md XI (Governance via Proposals): #60 introduces no governance-mutating command path.
→ **tasks.md (adds no command, no flag, no Outcome/ExitCode)**: the feature adds only recognizer code in `internal/apiclient`; it exposes no command path that could mutate governance, directly or otherwise.

**P0** | CONSTITUTION.md XII (Standalone Executable): the recognizer adds no external dependency.
→ **tasks.md T001 ("needs only the standard library")**: the `Gate` type, registry, matcher, and `RecognizeFeatureGate` use only the Go standard library — no new runtime or third-party dependency.

**P0** | CONSTITUTION.md II/VIII (token & body hygiene): recognition reads only method, path, and status.
→ **spec.md § Non-Behaviors (body-independence), plan.md Cross-cutting (secret hygiene), tasks.md T001 Risk**: the recognizer never reads the `X-Auth-Token` or the response body — it keys solely on the operation and HTTP status, so no token can leak and no body content can sway recognition (CONSTITUTION II).

**P0** | CONSTITUTION.md I/VIII (boundary fidelity): the modeled-but-unregistered `ai_integration` gate matches the spec's deferred scope.
→ **plan.md ADR-3 + guard test, spec.md edge scenario**: `GateAIIntegration` exists in the type but no registry row carries it (the deferred agent/skill endpoints have no command), pinned by an assertion test — recognition's active scope matches PROJECT scope, neither under- nor over-claiming.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords present** (`accords/governance/` does not exist). Done-criteria and cross-reference checks were not run. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-scenarios.md`, and `done-tasks.md` to enable done-criteria quality checks across the pipeline. (Same gap noted on sibling specs — project-wide, not specific to 060.)
- **No guardian-agent.md** at the checklist skill's `agents/` path — proceeded with SKILL.md alone (reduced character consistency, not a blocked skill).
- **Principle VI (Size-Aware by Design)** produced **zero applicable checks** — the recognizer has no list/tree/pagination surface.
- **Principle X (Respect API Limits)** produced **zero applicable checks** — the recognizer issues no request (it interprets an already-received 403), so neither the `429`-backoff nor the `If-Match` clause has a trigger.
