# Validate: Feature-Gate Recognition

**Feature**: 060-feature-gate-recognition
**Round**: 1 of 3
**Date**: 2026-06-15
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, features/unsignalled-plan-limits/feature-gate-recognition.feature, PROJECT.md
**Implementation files**: 3 in `internal/apiclient/` — `featuregate.go` (recognizer, 127 LOC), `featuregate_test.go` (unit tests), `feature_gate_recognition_bdd_test.go` (godog suite)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | — Skipped (no interface files) | — |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 4 dimensions checked (1 skipped), 4 passed, 0 findings. Both held-out validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

Every driving scenario (referenced by checked tasks T001/T002) has an identifiable code path. The godog suite `TestFeatureGateRecognitionFeatures` exercises all eight behavioral scenarios live (8 scenarios / 32 steps passing).

| Scenario | Status | Implementation |
|---|---|---|
| Advancing a draft → possible plan limit (Premium) | ✓ Covered | `featuregate.go:RecognizeFeatureGate` + registry row `POST /proposals/{proposal_id}/propose` |
| Creating a proposal → possible plan limit (Premium) | ✓ Covered | registry row `POST /proposals` |
| Recording a response → possible plan limit (Premium) | ✓ Covered | registry row `POST /proposals/{proposal_id}/responses` |
| 403 from a non-gated read → not recognized | ✓ Covered | unregistered-op fall-through → `GateNone` (`GET /roles/...`) |
| Non-403 from gated op → not recognized | ✓ Covered | `status != http.StatusForbidden` guard → `GateNone` |
| Genuine permission denial on gated op → flagged possible, not confirmed | ✓ Covered | recognizer is body/cause-agnostic; `Gate` documented as suspicion |
| Recognition ignores body content | ✓ Covered | `RecognizeFeatureGate(method, path, status)` — no body parameter |
| Modeled `ai_integration` gate, no reachable command today | ✓ Covered | `GateAIIntegration` in type; no registry row carries it |

## Acceptance Criteria

**Status**: Pass (T001, T002 — both checked, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — `Gate` type + static registry + path matcher + `RecognizeFeatureGate`, RED-first unit tests | ✓ Met | `featuregate.go`; `featuregate_test.go` covers the four gated 403s, non-403 (422/412/200) → `GateNone`, non-gated 403 → `GateNone`, segment-count/literal/wildcard/query/trailing-slash edges, purity/totality, the `ai_integration` modeled-but-unregistered guard, and the registry change-detector (length + comma-ok, avoiding the map-zero-value trap) |
| T002 — godog suite over the feature file driving the recognizer directly; un-`@wip` behavioral, hold `@validation` | ✓ Met | `feature_gate_recognition_bdd_test.go`; `Paths` scoped to only `feature-gate-recognition.feature`; suite reports its own count (8 scenarios / 32 steps); helpers return errors, never panic |

## Interface Contract Conformance

**Status**: Skipped — no interface-*.md files present (the feature has no external-facing or specification boundary, per plan § "What This Plan Does Not Cover")

## Non-Behavior Absence

**Status**: Pass (5 of 5 non-behaviors respected)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not render a user-facing "not available on your plan" / diagnostic text | ✓ Respected | `RecognizeFeatureGate` returns a bare `Gate`; grep for "not available on your plan"/"upgrade" in `featuregate.go` → none |
| Must not assert certainty a 403 is a plan limit | ✓ Respected | `Gate` type and recognizer doc frame a non-`None` result as a *suspicion* ("consistent with … never a confirmed plan limit"); no certainty-bearing field/return |
| Must not inspect the response body to identify a gate | ✓ Respected | recognizer signature is `(method, path string, status int)` — no body argument exists |
| Must not call the API to probe plan status | ✓ Respected | pure/total function; no I/O, no client use; registry is static in-code data |
| Must not register deferred `ai_integration` / out-of-scope ops, must not change exit-code mapping or alter the error chain | ✓ Respected | registry holds only the four Premium rows; `GateAIIntegration` unregistered; `git diff origin/main` shows only 3 new additive files in `internal/`, no existing type/call site touched |

## @wip Lifecycle Completion

**Status**: Pass

The 8 behavioral scenarios (referenced by checked tasks) have had `@wip` removed and run live. The 2 `@validation` scenarios retain `@validation @wip` — correct: they are referenced by no checked task and are held for this validate pass.

---

## Validation Scenario Results

**Status**: Satisfied (2 of 2 traced to implementation by independent inspection)

| Scenario | Status | Trace |
|---|---|---|
| Recognition produces only a classification, not a rendered message ("names no user-facing diagnostic wording") | ✓ Satisfied | `RecognizeFeatureGate` returns only a `Gate` enum value; no rendered message string anywhere in `featuregate.go` (grep confirmed). The "not available on your plan" wording is left to #61. |
| A recognized rejection is always expressed as a possibility ("possibility expressed everywhere recognition is described") | ✓ Satisfied | Both the `Gate` type doc and the `RecognizeFeatureGate` doc frame a non-`None` result as a suspicion ("a SUSPICION, never a verdict"; "*consistent with* the named gate, never a confirmed plan limit"); no surface asserts certainty. |

---

## Verdict: Ready

All checked conformance dimensions pass (interface skipped — no such boundary exists for this feature). All 8 driving scenarios trace to live code paths exercised by the godog suite; both T001/T002 acceptance criteria are met; all 5 non-behaviors are respected by construction (the recognizer is a pure, unwired leaf adding only additive surface in `internal/apiclient`); the `@wip` lifecycle is correct; and both held-out validation scenarios trace independently to the implementation. The implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The downstream consumer is Plan-Limit Signal (#61), which will thread operation identity to the central failure site and render the diagnostic — out of scope here by design.
