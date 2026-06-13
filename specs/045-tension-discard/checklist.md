# Checklist: Tension Discard

**Feature**: 045-tension-discard
**Checked against**: CONSTITUTION.md (12 principles). No `accords/governance/done-*.md` accords found — done-criteria checks skipped.
**Artifacts checked**: spec.md, plan.md, interface-cli.md, tasks.md, features/tension-capture/tension-discard.feature (13 scenarios)
**Checks**: 23 (23 pass, 0 fail)
**Generated**: 2026-06-13 (re-run — confirms readiness after one architecture-informed feature-file scenario was added)

> Pre-implementation assessment. No code exists yet. Each check evaluates whether the artifact set commits the build to conformant behavior — readiness to implement against the principle, not the implementation itself. Constitution principles requiring runtime evidence (e.g. a passing test, a clean-environment run) are calibrated to their artifact-level proxy: "does the artifact set obligate and design for this?"

---

## Summary

All 23 checks pass. Constitution: 23/23. Done-criteria: 0/0 (no accords). Cross-references: included in constitution category below.

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 19 | 19 | 0 |
| P1 (should fix) | 4 | 4 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **23** | **23** | **0** |

---

## Changes Since Previous Run

**Previous**: 0 P0, 0 P1, 0 P2 (0 failures) — 23/23 pass.
**Current**: 0 P0, 0 P1, 0 P2 (0 failures) — 23/23 pass.

**Trigger for re-run**: one architecture-informed behavioral scenario — "An invalid output format is rejected before any request" (`-o xml` → usage error naming the unsupported value and the valid formats, no request, exit 2) — was added to `tension-discard.feature` (now 13 scenarios, up from 12). It closes part of a prior analyze/risk finding about the bad-`--output` fail-fast path lacking an executable scenario.

**Net effect**: No check regressed and no check newly failed. The new scenario *strengthens* the evidence under three checks that were already passing:
- **C9** (TDD — acceptance scenarios for user-facing behavior): the bad-`--output` fail-fast path now has its own `@wip` behavioral scenario, and T003 explicitly un-`@wip`s the "spec-derived **+ architecture-informed** behavioral scenarios", so the architecture-derived path (resolve `--output` first) is now covered by an executable acceptance scenario rather than only by a task acceptance criterion.
- **C3 / C5** (Action Transparency — output channel + structured result): the new scenario pins that an invalid format selector fails as a usage error before any network call, naming the bad value and the valid formats — consistent with the structured-output and channel-discipline obligations.

No checks were resolved-from-failing (the previous run had none failing), and no previously-passing check moved to failing.

---

## Calibration

Five principles were calibrated to feature-specific, binary assertions because they are broad imperatives without measurable thresholds:

- **II Action Transparency** → "Does the artifact set obligate (a) machine-parseable structured output for the success result, (b) traceable endpoint + resource id, and (c) every error to name cause + next step?"
- **IV Test-Driven Development** → "Does the artifact set obligate a failing acceptance scenario before implementation (RED-first), with `@wip` scenarios for every user-facing behavior?"
- **V Composition over Monolith** → "Does the design add the feature without forcing edits to unrelated commands or sibling tension leaves?"
- **VII Working Software** → "Do the tasks bundle implementation with its tests and require build/vet to pass per unit?"
- **X Respect API Limits** → "Does the artifact set honor `429` handling, and address the `If-Match`/optimistic-concurrency obligation (apply it or record a justified deferral against the conflict-resolution table)?"

Three principles produced applicability findings (see Governance Notes): VI Size-Aware, XI Governance via Proposals, XII Standalone Executable.

---

## Constitution Checks: 23/23 passed

### Failures

None.

### Passed (23/23)

**C1 — P0** | CONSTITUTION.md I (Spec Fidelity): "Every command MUST map to an operation defined in the Glassfrog API v5 spec."
→ **spec.md § System Overview / Integration Boundaries**, **plan.md Inputs/ADR-1**: `tension discard` maps to `DELETE /tensions/{id}` (`deleteTension`), the vendored `spec/glassfrog-api-v5.yaml` operation. Path id is the documented `ten_` path parameter; request carries no body. PASS.

**C2 — P0** | CONSTITUTION.md I (Spec Fidelity): "The CLI MUST NOT invent endpoints, parameters, or behaviors the spec does not define."
→ **spec.md § Non-Behaviors / Assumptions**, **interface-cli.md § Surface**: discard sends no `If-Match`, no body, no `Content-Type`, and no editable flags; the `404`-as-success behavior is grounded in the endpoint's documented not-REST-strict-idempotent semantics ("treat 404-following-204 as success") rather than invented. The synthesized result carries no server-owned field. The newly-added bad-`--output` scenario (`-o xml`) exercises only the landed Output Format Selection (020) validation — it introduces no new parameter or behavior (`-o` is the inherited persistent flag). No undefined parameter is introduced. PASS.

**C3 — P0** | CONSTITUTION.md II (Action Transparency): "Every action MUST report the spec operation it invoked and the target resource, in machine-parseable form."
→ **spec.md § Output**, **interface-cli.md § Surface/Output**: the success result is the synthesized `{data:{id,discarded}}` document carrying the target `ten_` id, rendered through Output Format Selection (020) in `json`/`yaml`/`full`/`compact` — machine-parseable, traceable to the discarded resource. PASS.

**C4 — P0** | CONSTITUTION.md II (Action Transparency): "Every error MUST explain what went wrong and the next step."
→ **spec.md § Failure**, **interface-cli.md § Error Communication** (table): each failure row names a cause and a concrete next step (e.g. not-authenticated → "run `glassfrog auth login` or set GLASSFROG_TOKEN"; transport → "check connectivity"; invalid `--output` → names the bad value + the valid names). All route through the shared `reportFailure`/`classifyClientError` chokepoint (031/032). PASS.

**C5 — P1** | CONSTITUTION.md II (Action Transparency): structured-output channel discipline.
→ **interface-cli.md § Interactions (Piping/scripting)**: the `204`-vs-`404` distinction is an advisory on stderr; stdout carries the stable machine result for both. The advisory is explicitly "informational, not an error," does not change the exit code, and never includes the token. The new bad-`--output` scenario confirms an invalid selector is rejected on the diagnostic channel before any request (no stdout result, exit 2), preserving channel discipline. Channel separation preserves machine parseability. PASS.

**C6 — P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable, never hidden" / no "failure condition reported as success."
→ **plan.md ADR-2**, **interface-cli.md § Error Communication**: only the exact `StatusCode == 404` is folded into success — keyed via `errors.As` on the exact status; `401`/`403`/`429`/other non-2xx/transport/not-authenticated all route to `reportFailure` unchanged. `404`-as-success is a documented, justified end-state (the tension is gone), not a swallowed failure; the accepted cost (mistyped id) is recorded as a spec non-behavior and softened by the stderr advisory. No genuine error is hidden. PASS.

**C7 — P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "MUST NOT leave governance in a partially-applied state."
→ **spec.md § System Overview / Non-Behaviors**: discard issues exactly one bodyless `DELETE` of one tension; it does not cascade to proposals and performs no multi-step write, so there is no partial-application surface. PASS.

**C8 — P0** | CONSTITUTION.md III (Fail Safe, Not Silent): render-failure handling does not silently succeed.
→ **plan.md § Output / ADR-3**, **interface-cli.md § Error Communication** (render row): the stdout render is buffer-then-write, so a render failure leaves stdout empty and maps to `RuntimeError(1)` rather than a partial/false-success output. PASS.

**C9 — P0** | CONSTITUTION.md IV (TDD): "User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it."
→ **tension-discard.feature**, **tasks.md T003**: the feature file carries `@wip` scenarios for every behavioral user scenario (live discard, re-discard-safe, JSON render, missing id, multiple id, no token, permission, transport, rate-limit, **and the newly-added invalid-output-format fail-fast**); T003 makes them executable acceptance and un-`@wip`s the "spec-derived **+ architecture-informed** behavioral set" while holding the `@validation` scenarios. The added architecture-informed scenario is itself an executable acceptance for the resolve-`--output`-first path. PASS.

**C10 — P0** | CONSTITUTION.md IV (TDD): "Features MUST be built test-first (RED before GREEN)."
→ **tasks.md T002 (title + Risk) / T003**: T002 is "RED-first unit tests for every branch"; T003 makes scenarios pass as executable acceptance. Plan § Cross-cutting (Testing) enumerates the offline branch coverage including the bad-`--output` fail-fast with a transport tripwire. PASS.

**C11 — P1** | CONSTITUTION.md IV (TDD): acceptance scenarios cover the held-out validation behaviors.
→ **tension-discard.feature**: `@validation @wip` scenarios cover the no-read/write-verb claim, the no-fabricated-field claim, and the `404`-leaks-no-error claim — held for independent verification per the spec's Validation Scenarios. The newly-added scenario is tagged `@wip` only (behavioral), so the `@validation` held-out set is unchanged (3 scenarios). PASS.

**C12 — P0** | CONSTITUTION.md V (Composition over Monolith): "Adding a new command MUST NOT require changing unrelated ones."
→ **plan.md § Components / ADR-1**, **tasks.md T002 / Branching Guidance**: discard attaches one leaf to the existing `tension` group via `MustRegister` and adds one render key; it touches no `internal/glassfrog` model, no transport field, no shared `status.go`, and does not modify the landed `create`/`list`/`get`/`update` leaves. The only edits to existing code are the additive `newTensionCommand` registration and the group `Short` widening. PASS.

**C13 — P1** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts."
→ **plan.md § Cross-cutting (Testing)**, **tasks.md T002**: `runTensionDiscard` is pure over the landed `tensionSeam`, so every branch runs offline against a fake transport — independently testable without entangling siblings. PASS.

**C14 — P0** | CONSTITUTION.md VII (Working Software): "Every commit/PR MUST include implementation together with its tests."
→ **tasks.md T001/T002/T003**: each task bundles its tests — T001 ships render unit tests with the view/key/templates; T002 ships RED-first branch unit tests with the command; T003 is the executable acceptance suite. No code-only or test-only unit. PASS.

**C15 — P0** | CONSTITUTION.md VII (Working Software): "MUST validate and build."
→ **tasks.md T001/T002/T003 acceptance criteria**: every task requires `go build`/`go vet` clean (and the feature suites run clean in T003). PASS.

**C16 — P0** | CONSTITUTION.md VIII (No Fabricated Data): "MUST NOT invent, guess, or fill placeholder values for fields the API did not provide."
→ **spec.md § Output / Validation Scenarios**, **plan.md ADR-3**, **interface-cli.md § Output / Consistency Notes**, **tasks.md T001 Risk**: the synthesized result carries only the caller-supplied `ten_` id and a `discarded` marker — explicitly never a server-owned field such as `discarded_at`, which the bodyless response never provided. A validation scenario pins this. The `discarded:true` marker is the command's own confirmation of the action it took (a `204`/`404` is the server's confirmation the tension is gone), not a fabricated API field. PASS.

**C17 — P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "No write/mutation MUST occur except as the direct result of an explicit write command."
→ **spec.md § Invocation**, **interface-cli.md § Surface**: the `DELETE` fires only on the explicit `tension discard <ten-id>` command. The command exposes no read/list/get behavior (a `@validation` scenario asserts it), so there is no read-shaped path that mutates. PASS.

**C18 — P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): no interactive prompt substitutes for the explicit-command intent.
→ **spec.md § Non-Behaviors**, **interface-cli.md § Surface**: discard requires no `--force`/`--yes` and no confirmation prompt — intent is the explicit command itself, consistent with the constitution's conflict-resolution row (agent automation: the command's existence is the intent). PASS.

**C19 — P0** | CONSTITUTION.md X (Respect API Limits): "backing off on `429` responses."
→ **plan.md § Cross-cutting (Non-idempotent retry) / ADR (Cross-cutting)**, **interface-cli.md § Interactions / Error Communication**: discard reuses `NewRetryExecutor`; 017's `isSafeMethod` restricts `429` auto-retry to `GET`/`HEAD`, so a `DELETE` `429` surfaces once (mapped to `RateLimited`, exit 5) and is never silently re-sent. A scenario pins "surfaced, not silently retried." PASS.

**C20 — P0** | CONSTITUTION.md X (Respect API Limits): optimistic concurrency (`If-Match`) obligation.
→ **spec.md § Non-Behaviors**, **plan.md § What This Plan Does Not Cover / ADR notes**, **interface-cli.md § Interactions**: discard sends no `If-Match`. The constitution's conflict-resolution table requires `If-Match` "when an `ETag` is available." The artifacts record a justified deferral: `deleteTension` is an unconditional soft-delete (the server takes the tension off the active record whole; there is no field-level lost-update to clobber), and optimistic concurrency is the separate **Clobbered Changes** capability the write family opts into when it lands (precedent: 042/044 deferred the same guard). The detection rule keys on omission "when an `ETag` is available" — a whole-resource `DELETE` has no per-field ETag precondition to honor, so the deferral is conformant rather than a violation. PASS (deferral documented and traceable; see Governance Notes for the cross-spec dependency).

**C21 — P1 (cross-reference)** | Cross-artifact link presence (Action Transparency / TDD traceability chain).
→ **tasks.md**: T001/T002/T003 each carry Plan reference (ADR-1/2/3/4 + Phase 1), Interface references (interface-cli.md sections), and Scenario references (named tension-discard.feature scenarios). T003's scenario references resolve to scenarios that exist in the feature file. Links present. PASS.

**C22 — P0** | CONSTITUTION.md I (Spec Fidelity) — cross-reference: plan/interface decisions trace to the spec's defining behavior.
→ **plan.md ADR-2 / ADR-3 / ADR-4** ↔ **spec.md § Clarifications**: the `404`-as-success, bodyless-synthesis, and stderr-advisory decisions trace directly to the spec's three recorded clarifications (Session 2026-06-13). No plan decision contradicts or invents beyond the spec. PASS.

**C23 — P0** | CONSTITUTION.md II (Action Transparency) — cross-reference: the feature file's narration matches the spec's operation.
→ **tension-discard.feature header** ↔ **spec.md § System Overview**: the feature describes the same bodyless `DELETE`, `discarded_at` soft-delete, `404`-as-success, and stderr advisory — consistent operation framing. PASS.

---

## Done-Criteria Checks

No `accords/governance/done-*.md` accords exist in the repository — done-criteria checks were not generated. See Governance Notes.

---

## Governance Notes

These are infrastructure observations, separate from the feature-quality findings above. They are not counted in the severity summary.

- **No `accords/` directory exists.** Done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) could not be generated — checklist ran constitution-only. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable per-artifact quality checks for this and future specs. (Note: 045's artifacts conform to the in-repo precedent set by 042/043/044, which interface-cli.md § Consistency Notes explicitly references in lieu of accords.)
- **Principle VI (Size-Aware by Design): no applicable checks.** Discard operates on a single tension by id and returns a bodyless response; there is no result set to page or truncate. The pagination obligation does not apply to this feature.
- **Principle XI (Governance via Proposals): no applicable checks.** A tension is a captured gap, not a governance-structure artifact (role/accountability/domain/policy). `DELETE /tensions/{id}` is not a governance-structure mutation, so the proposal-default obligation does not apply. The tension family (042/043/044) sits outside the proposals-gated surface.
- **Principle XII (Standalone Executable): no applicable checks at artifact level.** This principle is verified by running the distributed artifact on a clean environment — a build/release concern (021/022), not a per-feature artifact concern. 045 adds no runtime dependency (no new package, no `internal/glassfrog`/`internal/apiclient` change), so it introduces no risk to this principle, but there is no artifact-level binary assertion to evaluate pre-implementation.
- **Principle X (Respect API Limits) — cross-spec dependency (C20):** the `If-Match` obligation is deferred to the **Clobbered Changes** capability. When that capability lands, the write family — including discard — is expected to opt into the shared concurrency guard. This deferral is recorded in spec.md § Non-Behaviors and plan.md, and matches the 042/044 precedent. Tracked here so the deferral is not lost.

---

## Notes for Analyze

- Checklist is vertical (each artifact against its own bar). The cross-reference checks (C21–C23) verify link/narration *presence and non-contradiction*, not full consistency — horizontal consistency (e.g. exact scenario-name matching between tasks.md T003 references and the feature file, sibling-decision drift, count accuracy) is analyze's domain.
- The feature file now carries **13 scenarios** (10 behavioral `@wip` + 3 `@validation @wip`). The newly-added "An invalid output format is rejected before any request" scenario is sourced as "Proposed: plan Interactions (--output resolved first; an invalid selector fails fast before any request)" — i.e. architecture-informed rather than spec-driven. Analyze should weigh whether the spec's Driving/Error scenario set should grow to mention this path, or whether the architecture-informed sourcing is sufficient (it is grounded in plan § Data flow and interface-cli.md § Interactions/Error Communication).
- One traceable wording detail for analyze to weigh: tasks.md T002/T003 scenario references cite scenario titles (e.g. "A live tension is discarded", "Re-discarding an already-gone tension stays safe") that match the feature file; the spec's Driving-Scenario titles differ slightly in wording ("Discard a live tension"). This is a presence-pass here; analyze owns exact cross-artifact title fidelity.
