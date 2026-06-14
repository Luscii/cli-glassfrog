# Validate: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/clobbered-changes/stale-write-surfacing.feature, PROJECT.md
**Implementation files**: 3 changed + 3 test files in `internal/cli/` — `dispatch.go`, `exitcode.go`, `diagnostic.go`; `exitcode_test.go`, `diagnostic_test.go`, `stale_write_surfacing_bdd_test.go` (new)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 driving scenarios covered)

All driving scenarios are concretized in the feature file, referenced by the checked task T001, and run as live BDD (7 scenarios / 28 steps passing in `TestStaleWriteSurfacingFeatures`). Each traces to an identifiable code path.

| Scenario | Status | Implementation |
|---|---|---|
| A stale write surfaces under its own exit code | ✓ Covered | `diagnostic.go:categoryForStatus` (412→`StaleWrite`) + `exitcode.go:ExitCode` (`StaleWrite`→`codeStaleWrite=7`) |
| The next step points the operator to re-read and retry | ✓ Covered | `diagnostic.go:nextStepForStatus` (412 arm → "re-read the resource for its current version, then retry the write") |
| Classification ignores whether a precondition was sent | ✓ Covered | `diagnostic.go:categoryForStatus(status int)` — branches on status alone, no `If-Match`/command/resource input |
| The cause names the stale write | ✓ Covered | `diagnostic.go:problemCause` (non-synthesized path surfaces API detail) + `StaleWrite` category |
| A 412 without readable detail derives its cause from the status | ✓ Covered | `diagnostic.go:problemCause` (synthesized 412 branch → status-derived precondition-failure cause) |
| Another non-2xx status is unaffected | ✓ Covered | `diagnostic.go:categoryForStatus` default arm (404→`APIError`) unchanged |
| Adding the stale-write code renumbers no existing code | ✓ Covered | `exitcode.go:codeStaleWrite=7` (additive); `exitcode_test.go` pins no-renumber |

---

## Acceptance Criteria

**Status**: Pass (T001 — all criteria met)

| Criterion (tasks.md T001) | Status | Evidence |
|---|---|---|
| 412 → `Category: StaleWrite`, exit `codeStaleWrite=7`, distinct from generic `3` | ✓ Met | `categoryForStatus`/`ExitCode`; `TestExitCode_ProducerBackedCategories` (`StaleWrite→7`), `TestDiagnose_StaleWrite_412` |
| 412 next step = re-read/retry, not the generic token step | ✓ Met | `nextStepForStatus` 412 arm; BDD "next step" scenario + `TestDiagnose_StaleWrite_412` assert re-read/retry and reject "token has access" |
| Cause surfaces API `detail`/`title` when present; status-derived (never fabricated) when synthesized | ✓ Met | `problemCause` 412 synthesized branch; API-detail path unchanged; `TestDiagnose_StaleWrite_412` both sub-cases |
| Classification status-driven only (command/resource/`If-Match` independent) | ✓ Met | `categoryForStatus(status int)` takes only the status |
| Every other status unchanged (404/500→`APIError`(3); 401/403→`PermissionError`(4); 429→`RateLimited`(5)) | ✓ Met | `TestDiagnose_412_DoesNotDriftOtherStatuses`; `TestRenderDiagnostic_ByteEquivalence` (unchanged cause/next-step strings); default arms untouched |
| `7` new/unused, no renumber, outside shell-reserved (`126`/`127`/`128+N`) | ✓ Met | `exitcode.go`; `TestExitCodeConstants_ExactValues`/`_Distinct`/`_NoShellReserved` |
| Unit tests with `len` + comma-ok lookups (zero-value-trap guard) | ✓ Met | `exitcode_test.go` `outcomeCodes`/`publishedCodes` use `len` + comma-ok; `StaleWrite→7` added across all three pins |

---

## Interface Contract Conformance

**Status**: Pass (both published surfaces conformant)

| Surface (interface-cli.md) | Status | Evidence |
|---|---|---|
| New `7` (stale-write) row in the published exit-code registry | ✓ Conformant | `exitcode.go:codeStaleWrite=7` + `ExitCode` arm; registry table value matches |
| `412` diagnostic — cause + re-read/retry next step — in the Error Communication table | ✓ Conformant | `nextStepForStatus`/`problemCause` 412 arms; cause prefers API words, otherwise status-derived |
| "Never zero on failure" | ✓ Conformant | `StaleWrite`→`7` (non-zero); `ExitCode` default arm returns `1`, never `0` |
| "No new shell-reserved code" | ✓ Conformant | `7 < 126`; pinned by `TestExitCodeConstants_NoShellReserved` |
| "Renders no text itself / no new stderr writer" | ✓ Conformant | `diagnostic.go` produces the `Diagnostic` value only; `renderDiagnostic`/032 unchanged |
| No new command/flag/output format | ✓ Conformant | No command-tree, flag, or output-format change in the diff |

---

## Non-Behavior Absence

**Status**: Pass (all 7 exclusions respected)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No re-read/retry/back-off/recovery | ✓ Absent | `Diagnose`/`ExitCode` are pure classification — no transport, clock, sleep, or retry introduced |
| No rendering/printing/formatting in any `--output` | ✓ Absent | Change supplies category/cause/next-step into the existing `Diagnostic`; no new writer; 032 untouched |
| Does not emit/decide the exit code itself | ✓ Absent | `ExitCode` remains the single mapper; the change registers a code, does not emit one |
| No change to any non-412 status's surfacing | ✓ Absent | Additive `case` labels; default arms unchanged; `TestDiagnose_412_DoesNotDriftOtherStatuses` + byte-equivalence pins |
| No renumber/reassign/reuse of existing codes | ✓ Absent | `7` is a fresh constant; `0`–`6` untouched; `TestExitCodeConstants_ExactValues` pins |
| Classification not conditioned on `If-Match`/command/resource | ✓ Absent | `categoryForStatus` signature is `(status int)` only |
| No fabricated cause/next step | ✓ Absent | API detail preferred; synthesized fallback is `412`-status-derived (precondition-failed / changed-since-read), never an invented specific reason |

---

## @wip Lifecycle Completion

**Status**: Pass

The 7 driving scenarios referenced by checked task T001 have had `@wip` removed and run live (`TestStaleWriteSurfacingFeatures`: 7 passed). The 3 `@validation` scenarios correctly retain `@wip` — they are held out for this validation pass per T001 ("3 @validation held @wip") and were not to be un-wipped by the Builder. No stray `@wip` remains on a Builder-owned scenario.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation independently)

These were held out from the Builder. Each is traced to a code path by inspection and corroborated by a mirroring unit test the Builder did not need to satisfy the driving scenarios.

| Scenario | Status | Trace |
|---|---|---|
| A stale write is distinct from the generic bucket (412 vs 500) | ✓ Satisfied | 412→`StaleWrite`/`7` via `categoryForStatus`+`ExitCode`; 500→`APIError`/`3` (default arm). `TestDiagnose_StaleWrite_412` + `TestDiagnose_412_DoesNotDriftOtherStatuses` (500 row) confirm both. |
| The capability surfaces without recovering | ✓ Satisfied | `Diagnose`/`ExitCode` are pure — no transport/clock/retry exists in the path. BDD `thenNoRecovery` asserts `usedTransport` stays false; only category/cause/next-step/code are produced. |
| No existing surfacing drifts (401/403/404/429/500) | ✓ Satisfied | Additive `case` labels leave every other arm untouched. `TestDiagnose_412_DoesNotDriftOtherStatuses` pins category+code+next-step for all five; `TestRenderDiagnostic_ByteEquivalence` pins the unchanged cause/next-step strings. |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 validation scenarios are satisfied through independent inspection (corroborated by passing unit and BDD tests). The implementation delivers exactly what the spec, plan, and interface promised: the `412` is branched out of the generic API-error bucket into a distinct `StaleWrite` category mapped to a new, previously-unused exit code `7`, with a cause that names the precondition failure and a re-read/retry next step — classified by status alone, renumbering no existing code, rendering nothing itself, and changing no other status's surfacing. The implementation conforms to the specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The Optimistic Concurrency capture → send → surface chain (052 → 053 → 054) is complete.
