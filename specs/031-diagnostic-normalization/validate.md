# Validate: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Round**: 1 of 3
**Date**: 2026-06-10
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/opaque-failures/diagnostic-normalization.feature, PROJECT.md
**Implementation files**: `internal/cli/diagnostic.go` (new — `Diagnostic`, `Diagnose`, `renderDiagnostic`, `categoryForStatus`, `nextStepForStatus`, `problemCause`), `internal/cli/clienterror.go` (`classifyClientError` delegate), `internal/cli/me.go` (`reportClientError`), `internal/cli/exitcode.go` (category→code map, unchanged); ripple across 011–014 tests + 4 feature files

> Note: `agents/guardian-agent.md` and the context-engineering references are not deployed in this Score install — applied the skill's own self-checks and the validate-template only; reduced character consistency, not a blocked validation.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass (all task-referenced scenarios covered) | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. One driving scenario (*Usage error normalized from dispatch*) is **deferred to 032** by plan ADR-3 — documented below, not a defect.

---

## Driving Scenario Coverage

**Status**: Pass — every driving scenario referenced by a checked task has an identifiable code path. One spec scenario is an intentional, documented deferral (see the note below).

| Scenario | Status | Implementation |
|---|---|---|
| Transport failure → network-unavailable | ✓ Covered | `diagnostic.go:79-86` — `*TransportError` → `NetworkUnavailable`, cause `transportErr.Error()` ("request failed: …"), next step "check connectivity; the API may be unreachable" |
| Permission failure carries the API's own detail (403 + detail) | ✓ Covered | `diagnostic.go:94-101` + `problemCause:206` — `categoryForStatus(403)=PermissionError`, cause surfaces the API `detail`, `nextStepForStatus(403)` = membership/permission |
| API error with no readable detail (500) → general API error, cause from status | ✓ Covered | `diagnostic.go:94-101` — synthesized detail → `problemCause` emits "status 500"; `categoryForStatus(500)=APIError`; generic (non-fabricated) next step |
| Undecodable 2xx → general API error | ✓ Covered | `diagnostic.go:112-126` — `*DecodeError` → `APIError`, cause "the API response did not match the expected shape" |
| Rate-limited (429 after retries) → rate-limited, reset-window next step, no extra retry | ✓ Covered | `diagnostic.go:94-109` + `nextStepForStatus:193` — `RateLimited`, reset-window hint; `Diagnose` is pure (no clock/transport) |
| A success is never normalized | ✓ Covered | `diagnostic.go:57-59` — `Diagnose(nil)` → `Diagnostic{Category: Success}`, empty cause/next step |
| Most-specific category wins (429 ≠ general API error) | ✓ Covered | `diagnostic.go:165-174` — `categoryForStatus(429)=RateLimited`; the switch default is the generic `APIError` bucket, so 429 can't fall through |
| Unrecognized failure → internal-error diagnostic (cause = own message, no next step, no stack trace) | ✓ Covered | `diagnostic.go:154-158` — fail-safe → `RuntimeError`, cause `err.Error()`, no next step, no trace |
| Usage error normalized from dispatch (unknown command "rolez") | ⚠ Deferred to 032 | No `Diagnose` arm for the cobra unknown-command/flag surface; plan ADR-3 leaves that path untouched in 031 and routes it through 032 when 032 owns rendering. Scenario correctly remains `@wip` and is referenced by **no** checked task. |

**Deferral note (not a finding)**: The spec lists *Usage error normalized from dispatch* as a happy-path driving scenario, but plan ADR-3 deliberately scopes the cobra-native usage surface out of 031 (`Diagnose` covers `reportClientError`'s error-value families; cobra renders its own usage errors today, and 032 reconciles them into the unified envelope). The scenario is `@wip` and unreferenced by T001/T002, so the @wip-lifecycle rule treats it as "not yet implemented," not a failing check. `Diagnose` *does* classify the usage-family errors within its scope (`*AuthError{NoCredentials}`, base-URL, `*output.FormatError` → `UsageError`). Tracked for 032; the implement-phase LEARNINGS entry (2026-06-10) records the same.

---

## Acceptance Criteria

**Status**: Pass — all acceptance criteria for both checked tasks (T001, T002) are met.

| Task | Criterion | Status | Evidence |
|---|---|---|---|
| T001 | Table-driven `Diagnose` category test + `len`+comma-ok exhaustiveness guard | ✓ Met | `diagnostic_test.go:TestDiagnose_Category_Table` (asserts every family; guard names a missing category) |
| T001 | `renderDiagnostic(Diagnose(err))` byte-equivalent to pre-change `formatClientErrorMessage` per family | ✓ Met | `diagnostic_test.go:TestRenderDiagnostic_ByteEquivalence` (golden per family; the 3 refined arms updated in T002 by design) |
| T001 | `reportClientError` delegates; stderr + Outcome unchanged; existing tests pass | ✓ Met | `me.go:reportClientError` (`refine→Diagnose→render→.Category`); `TestReportClientError_Delegates`; suite green |
| T001 | No arm emits `X-Auth-Token`; token test over `.Cause`/`.NextStep`/rendered for every arm | ✓ Met | `diagnostic_test.go:TestDiagnose_NeverEmitsToken` (14 arms) |
| T001 | `go build` + `go vet` + `go test ./...` green | ✓ Met | confirmed green |
| T002 | `*DecodeError` → `APIError`, `ExitCode`=3; `clienterror_test.go` asserts `APIError`; no surviving decode→exit-1 assertion | ✓ Met | `diagnostic.go:121-125`; `clienterror_test.go:38` (`decode-is-api-error`); repo grep clean; 5 unit tests + 4 BDD harnesses + 4 feature files updated |
| T002 | 401 and 403 both `PermissionError` (4), distinct next steps | ✓ Met | `categoryForStatus` (both `PermissionError`); `nextStepForStatus:189-192` (token vs membership) |
| T002 | 429 → `RateLimited` (5), reset-window next step, no wait/retry | ✓ Met | `nextStepForStatus:193`; `Diagnose` is pure |
| T002 | Render-template failure (019) + fail-safe stay `RuntimeError` (1) — regression guard | ✓ Met | `diagnostic.go:158` (fail-safe `RuntimeError`); `render_test.go:130` still asserts exit 1 (untouched) |
| T002 | Token-free invariant holds for the 3 changed arms; build + test green | ✓ Met | `TestDiagnose_NeverEmitsToken` covers 401/403/429; suite green |

---

## Interface Contract Conformance

**Status**: Pass — all surface elements in interface-spec.md exist with the specified shapes.

| Surface element | Status | Implementation |
|---|---|---|
| `Diagnose(err error) Diagnostic` — total normalizer | ✓ Conformant | `diagnostic.go:56` |
| `Diagnostic{Category Outcome; Cause string; NextStep string}` | ✓ Conformant | `diagnostic.go:20-32` |
| `classifyClientError` → thin `Diagnose(err).Category` delegate | ✓ Conformant | `clienterror.go` |
| `formatClientErrorMessage` / `clientErrorNextStep` removed/absorbed | ✓ Conformant | removed from `me.go`; arm logic absorbed into `Diagnose` + `nextStepForStatus` + `problemCause` |
| `reportClientError` delegates (`refine→Diagnose→render→.Category`) | ✓ Conformant | `me.go:reportClientError` matches the contract line-for-line |
| `renderDiagnostic` (unexported): `Cause` or `Cause + " — " + NextStep` | ✓ Conformant | `diagnostic.go:224-229` |
| Status→Category→exit map incl. **decode→3** | ✓ Conformant | `categoryForStatus` + `exitcode.go` (`APIError`=3); decode arm sets `APIError` |
| Next-step contract: 401=token, 403=membership/permission, 429=reset-window | ✓ Conformant | `nextStepForStatus:189-194` |
| `NextStep string` with `""` = none | ✓ Conformant | fail-safe + `*output.FormatError` arms leave `NextStep` empty; `renderDiagnostic` checks emptiness |

---

## Non-Behavior Absence

**Status**: Pass — no excluded behavior is present in the implementation.

| Non-behavior (spec § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not print / render in any `--output` format | ✓ Absent | `Diagnose` returns a value; the only print is `reportClientError`'s existing human-stderr line via `renderDiagnostic` (032 takes over) |
| Must not emit / decide the process exit code | ✓ Absent | `Diagnose` returns `Category` only; `ExitCode` (exitcode.go) is the sole mapper |
| Must not print a stack trace for a normalized failure | ✓ Absent | fail-safe returns `cause = err.Error()` only; trace-writing stays in `recoverToCode` (panic safety net) |
| Must not retry / back off / sleep | ✓ Absent | `Diagnose` is a pure `errors.As` chain — no clock, transport, or retry |
| Must not re-parse the raw response body | ✓ Absent | `problemCause` reads `Detail`/`Title`/`StatusCode` off the already-typed `*ProblemError`; never touches `.Body` |
| Must not translate 403 into plan/Premium guidance | ✓ Absent | `nextStepForStatus(403)` = "required role membership / permission" — no plan/Premium language; cause is the API's own detail |
| Must not fabricate a cause or next step | ✓ Absent | synthesized detail keeps the generic "status N" wording; fail-safe attaches no next step; generic non-2xx keeps the existing generic line |

---

## @wip Lifecycle Completion

**Status**: Pass — every `@wip` on a scenario referenced by a checked task was removed.

- **Removed (10 scenarios)**: T001 → transport, permission-detail, api-error-no-detail, success, unrecognized; T002 → undecodable-2xx, decode-exit-3, 401/403-split, rate-limit-reset, most-specific. All pass in `TestDiagnosticNormalizationFeatures` (10 scenarios, 45 steps green).
- **Correctly still `@wip`**: 4 `@validation` scenarios (held for this skill — independent verification) and *Usage error normalized from dispatch* (referenced by no checked task; deferred to 032 per ADR-3). Both classes are excluded from the lifecycle check by rule.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 `@validation` scenarios traced to implementation by inspection). These were held out from the Builder.

| Scenario | Status | Trace |
|---|---|---|
| One consistent shape across every failure family | ✓ Satisfied | `Diagnose` always returns `Diagnostic{Category, Cause, NextStep}` (the struct guarantees the same three fields), and `Category` is always an `Outcome` from the fixed `exitcode.go` taxonomy. Holds for transport + typed-API + the usage-family errors in `Diagnose`'s scope (auth/base-URL/format). |
| The 403 boundary holds — no plan guidance leaks in | ✓ Satisfied | `nextStepForStatus(403)` and `problemCause` contain no "plan"/"Premium"/"upgrade" language; the 403 cause is the API's own detail. Inspection confirms the *Unsignalled Plan Limits* boundary is intact. |
| No implementation leakage in the artifact | ✓ Satisfied | spec.md names only the observable fields (cause, category, next step) and prescribes no language/type/layout — the artifact is implementation-agnostic. |
| No diagnostic output carries the auth token | ✓ Satisfied | `TestDiagnose_NeverEmitsToken` asserts `meSecretToken` is absent from `.Cause`, `.NextStep`, and the rendered line across all 14 arms; every arm sources response/path/status text only. |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 4 `@validation` scenarios are satisfied by inspection. The implementation is a faithful consolidation: `Diagnose` produces the one `{Category, Cause, NextStep}` value from a single `errors.As` chain; `reportClientError` delegates; `classifyClientError` is a thin `.Category` shim; and the three developer-confirmed refinements (decode→APIError/3, 401/403 next-step split, 429 reset-window) are present, tested, and rippled consistently across the four sibling features that pinned the old decode→exit-1 behavior. The token-free and totality invariants hold.

One spec driving scenario — *Usage error normalized from dispatch* — is **not delivered in 031** because plan ADR-3 deliberately defers the cobra-native usage surface to 032 (Output-Aware Failure Rendering). The scenario is correctly `@wip` and referenced by no task, so it does not fail any conformance check; it is surfaced here and tracked for 032 rather than treated as a defect. Within 031's delivered scope, the implementation conforms to its specification.

---

## Next Steps

Implementation conforms to the specification. Suggest **PR review and merge** (`gh pr create --base main`).

Two items to carry forward (not blockers):
- **032 (Output-Aware Failure Rendering)** must (a) implement the deferred *Usage error normalized from dispatch* scenario by routing the cobra usage surface through the unified envelope, and (b) render the decode `Diagnostic` under the `APIError` kind/envelope. Confirm the totality contract (every failure → a renderable diagnostic) still holds end-to-end when 032 lands.
- **Doc/prose reconciliation**: the decode→1 rows in 011/012/014/025/026/034 `interface-cli.md` and the user-facing `docs/reference/*.md` still describe the superseded exit-1 behavior. The DEPRECATION.md `[decision]` entry records the supersession; regenerate the affected docs via `/score:document` when convenient.
