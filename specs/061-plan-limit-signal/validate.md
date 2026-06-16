# Validate: Plan-Limit Signal

**Feature**: 061-plan-limit-signal
**Round**: 1 of 3
**Date**: 2026-06-16
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-spec.md, interface-cli.md, features/unsignalled-plan-limits/plan-limit-signal.feature, PROJECT.md
**Implementation files**: 4 source files — `internal/apiclient/execute.go` (`ResponseError.Method/Path`), `internal/cli/diagnostic.go` (`Diagnose` 403 branch, `featureGateDisplayName`, `planLimitCause`/`planLimitNextStep`, `Diagnostic.Feature`), `internal/cli/errorenvelope.go` (`errorEnvelopeFor` mapping), `internal/output/error.go` (`ErrorDetail.Feature`); plus tests in `execute_test.go`, `diagnostic_test.go`, `errorenvelope_test.go`, `internal/output/error_test.go`, and the BDD suite `internal/cli/plan_limit_signal_bdd_test.go`

> **Tooling note**: `guardian-agent.md` and the context-engineering references are not deployed in this Score cache — applied the SKILL.md process and skill-specific self-checks only (reduced character consistency, not a blocked validation).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All driving scenarios are concretized in `plan-limit-signal.feature` and pass as executable acceptance (`TestPlanLimitSignalFeatures`: 8 scenarios, 38 steps, all pass). Each traces to a code path through the `reportFailure → refineClientError → Diagnose → renderDiagnostic/errorEnvelopeFor` chain.

| Scenario | Status | Implementation |
|---|---|---|
| Advancing a draft names the gating feature + next step | ✓ Covered | `diagnostic.go:118` 403 branch → `planLimitCause`/`planLimitNextStep` (282/289) |
| Creating a proposal names the gate as a possibility | ✓ Covered | same branch; `RecognizeFeatureGate("POST","/proposals",403)` → `GatePremiumAsyncProposals` |
| Distinct `feature` element under json | ✓ Covered | `errorenvelope.go:81` `Feature ← d.Feature`; `output/error.go:45` `json:"feature,omitempty"` |
| Non-recognized 403 keeps generic diagnostic | ✓ Covered | `GateNone` fall-through leaves `categoryForStatus`/`nextStepForStatus` output unchanged |
| Non-403 on a gated op gets no plan-limit wording | ✓ Covered | 403-only guard (`StatusCode == http.StatusForbidden`) |
| Recognized 403 keeps exit code 4 across formats | ✓ Covered | `Category` stays `PermissionError`; `exitcode.go:54` → `codePermissionError = 4` |
| `ai_integration` gate produces no message today | ✓ Covered | gate absent from `gatedOperations` registry (060) → `GateNone` for any reached op |
| Genuine permission denial is hedged, never asserted | ✓ Covered | `planLimitCause` notes "a 403 may instead mean your identity lacks permission" |

---

## Acceptance Criteria

**Status**: Pass (all criteria for T001–T003 met)

- **T001** — `ResponseError.Method/Path` added and set in `Execute` (`execute.go`); `Error()` unchanged (status only — verified `TestExecuteNon2xxCarriesRequestIdentity`); reachable from a refined `*ProblemError` via the unwrap path; build/vet clean.
- **T002** — `Diagnostic.Feature` set once in `Diagnose`'s `*ProblemError` arm on a recognized 403; `featureGateDisplayName` total with exhaustiveness guard (`TestFeatureGateDisplayName_Exhaustive`); possibility-framed wording; `Category` stays `PermissionError`; `ErrorDetail.Feature` `omitempty` in order `message, next_step, feature, kind, status, body`; `errorEnvelopeFor` reads the one `Diagnostic` (no re-recognition); non-gated/non-403 cases unchanged.
- **T003** — every non-`@validation` scenario executable and passing; `@wip` removed from the 8 behavioral scenarios; the 2 `@validation` scenarios retain `@wip`; suite `Paths` names only `plan-limit-signal.feature`; step helpers return errors, reuse the sibling `reportFailure`/`renderCapture` chokepoint.

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Evidence |
|---|---|---|
| `ResponseError.Method` / `.Path` (NEW, set in `Execute`) | ✓ Conformant | `execute.go` struct + non-2xx construction; `Error()` byte-stable |
| `Diagnostic.Feature` (NEW, set once by `Diagnose`) | ✓ Conformant | `diagnostic.go:123` |
| `ErrorDetail.Feature` `json:"feature,omitempty"`, declared in `internal/output`, populated in `internal/cli` | ✓ Conformant | `output/error.go:45` + `errorenvelope.go:81`; declaration order matches |
| `Diagnose` 403 recognition branch (category unchanged) | ✓ Conformant | `diagnostic.go:118–127`; `categoryForStatus`/`nextStepForStatus` untouched |
| `featureGateDisplayName` total, human-prose, guarded | ✓ Conformant | maps `GatePremiumAsyncProposals → "Premium async proposals"`, `GateAIIntegration → "AI Integration"`, `GateNone → ""` |
| `errorEnvelopeFor` adds `detail.Feature = d.Feature` (no re-recognition) | ✓ Conformant | reads the single `Diagnostic` |
| `renderDiagnostic` unchanged | ✓ Conformant | gate rides in `Cause` prose |
| CLI: rendered cause/next-step strings | ✓ Conformant | `diagnostic.go:282/289` match the `interface-cli.md` example verbatim |
| CLI: `feature` key presence/source + exit code 4 | ✓ Conformant | `omitempty`; `kind: permission`; exit 4 unchanged |

---

## Non-Behavior Absence

**Status**: Pass (no excluded behavior present)

| Non-behavior | Status | Evidence |
|---|---|---|
| No certainty / no "upgrade" as sole remedy | ✓ Absent | wording hedges with "may not"; tests assert no `upgrade` across all command paths and the BDD `neverAssertsCertainInsufficiency` |
| Does not recognize gates itself | ✓ Absent | consumes `apiclient.RecognizeFeatureGate`; no gate metadata duplicated in `internal/cli` |
| No fabricated plan name/price/URL | ✓ Absent | `planLimitCause`/`planLimitNextStep` interpolate only the gate display name |
| Does not change category or exit code | ✓ Absent | `Category = categoryForStatus(403) = PermissionError`; `categoryForStatus`/`exitcode.go` untouched |
| Does not implement format rendering / redefine envelope shape | ✓ Absent | additive `omitempty` field declared in 018's home, populated in `cli`, no encoder change — the sanctioned extension pattern mirroring 032's `next_step`; rendering still owned by 032/018 |
| Does not inspect body or call API | ✓ Absent | recognition keys on method/path/status only; no body read, no request issued |
| No message for the unreached `ai_integration` gate | ✓ Absent | gate is mapped for readiness but absent from the gated registry, so no reached operation yields it today |

---

## @wip Lifecycle Completion

**Status**: Pass

`plan-limit-signal.feature` retains `@wip` on exactly the two `@validation` scenarios (lines 92, 100); all 8 behavioral scenarios — every one referenced by the checked T003 — have had `@wip` removed and pass under the suite's `~@wip` filter. No stray `@wip` remains on implemented scenarios.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation by inspection)

These were held out from the Builder. Their `.feature` forms (2 scenarios) remain `@wip` (not executed); traced independently against the code.

| Scenario | Status | Trace |
|---|---|---|
| Every rendered plan-limit failure frames the limit as a possibility | ✓ Satisfied | Wording produced once in `Diagnose` and read by both renderers (single-classification-site invariant), so framing is format-independent. `planLimitCause` hedges ("may not include"; "may instead mean your identity lacks permission"); neither helper contains "upgrade" or a certainty claim. |
| A rendered plan-limit failure invents no remedy detail | ✓ Satisfied | The only interpolated specific is the gate display name from `featureGateDisplayName`; no price, URL, or plan name literal exists; the next step is a verify action (`diagnostic.go:289`). |
| No implementation leakage in the artifact (spec-only) | ✓ Satisfied | `spec.md` names observable behavior (diagnostic facts, the gate element, which format renders where) and prescribes no language/type/data layout. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 validation scenarios are satisfied through independent inspection (the 8 behavioral driving scenarios additionally pass as executed BDD). The implementation conforms to the specification: a recognized plan-gate `403` surfaces a possibility-framed diagnostic naming the gating feature, with a distinct `omitempty` `feature` envelope element under structured formats, while the permission category and exit code 4 are provably unchanged. The full module suite (`go build`, `go vet`, `go test ./...`) is clean offline.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The two `@validation` scenarios stay `@wip` by design (held-out verification); they need no further code. The specification loop is closed.
