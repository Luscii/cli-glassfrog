# Validate: Tension Discard

**Feature**: 045-tension-discard
**Round**: 1 of 3
**Date**: 2026-06-13
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/tension-capture/tension-discard.feature, PROJECT.md
**Implementation files**: `internal/cli/tension.go` (`newTensionDiscardCommand`, `runTensionDiscard`, `tensionDiscardResult`, `isNotFound`, group wiring), `internal/render/render.go` (`TensionDiscardView`, `ResourceTensionDiscard`), `internal/render/templates/tension-discard.{full,compact}.tmpl`; tests in `internal/cli/tension_discard_test.go`, `internal/cli/tension_discard_bdd_test.go`, `internal/render/tension_discard_test.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (10 of 10 behavioral scenarios covered)

Every behavioral scenario in `tension-discard.feature` has an identifiable code path and an executable, passing test in `TestTensionDiscardFeatures` (10 scenarios / 57 steps green).

| Scenario | Status | Implementation |
|---|---|---|
| Discard a live tension (`204`) | ✓ Covered | `runTensionDiscard` `derr == nil` branch → "discarded" advisory + synthesized result (`tension.go`) |
| Re-discarding an already-gone tension stays safe (`404`) | ✓ Covered | `isNotFound(derr)` branch → "already gone" advisory, `Success` |
| Discard result rendered as JSON | ✓ Covered | `rt.format.MachineFormat()` branch → `output.RenderSuccess` over `{data:{id,discarded}}` |
| Missing tension id rejected before request | ✓ Covered | `cobra.ExactArgs(1)`; transport tripwire confirms no request |
| More than one positional id rejected | ✓ Covered | `cobra.ExactArgs(1)` |
| Missing token → not-authenticated usage error | ✓ Covered | `newClient`/`Execute` `*AuthError` → `reportFailure` → `UsageError` |
| Refused permission (`403`) fails with status | ✓ Covered | non-404 error → `reportFailure` → `PermissionError(4)` |
| Transport failure → network-unavailable | ✓ Covered | `*TransportError` → `reportFailure` → `NetworkUnavailable(6)` |
| Invalid `--output` rejected before request | ✓ Covered | `resolveRenderTarget` first → `UsageError(2)`, no request |
| Rate-limited discard surfaced, not retried | ✓ Covered | `429` not auto-retried (017 `isSafeMethod` gate); `RateLimited(5)`, one call |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked; all criteria met)

- **T001** — `ResourceTensionDiscard` resolves to both `tension-discard.{full,compact}.tmpl` (covered by `TestRegistry_AllBuiltinsResolve`, the key is in `builtinResources`); `TensionDiscardView{ID}` renders the `<ten_…>  [discarded]` line in both formats; view exposes only the id (no `discarded_at`). Verified by `internal/render/tension_discard_test.go`.
- **T002** — bodyless `DELETE /tensions/{id}` (`out == nil`, no body, no `Content-Type`); `204` → synthesized result + "discarded" advisory + exit 0; `404` → identical result + "already discarded" advisory + no not-found error + exit 0, keyed on exact `404`; `-o json` → `{"data":{"id":…,"discarded":true}}`; missing/multiple id and stray flag → `UsageError(2)` no request; bad `--output` → `UsageError(2)` no request; AuthError/transport/`401`-`403`/`429`/other classified via `reportFailure`; no new `Outcome`/`ExitCode`, siblings untouched, no token leak, offline, `go build`/`go vet` clean. Verified by 14 unit tests.
- **T003** — every non-`@validation` scenario passing with `@wip` removed; 3 `@validation` kept `@wip`; suite `Paths` names only the discard feature file; `404`/`429`/no-request paths asserted; no real network/home. Verified by `TestTensionDiscardFeatures`.

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface element | Status | Evidence |
|---|---|---|
| `tension discard <ten-id>` leaf (`ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) | ✓ Conformant | `newTensionDiscardCommand` |
| No editable-field flags (stray flag = cobra usage error) | ✓ Conformant | no `Flags()` declarations; `TestTensionDiscardCommand_RejectsFieldFlag` |
| Attached to existing `tension` group; group `Short` widened to name discard | ✓ Conformant | `newTensionCommand` `MustRegister(group, newTensionDiscardCommand(seam))`; group `Short` lists discard |
| One bodyless `DELETE /tensions/{id}` — `out == nil`, no `Content-Type`, no `If-Match`, id `url.PathEscape`d | ✓ Conformant | `runTensionDiscard` `apiclient.Request{Method: DELETE, Path: …}`; asserted by unit + BDD |
| Outcome interpretation: `204` advisory, `404`-as-success advisory, else `reportFailure` | ✓ Conformant | advisory text matches the accord ("discarded tension `<id>`" / "tension `<id>` was already discarded — nothing to do") |
| Output: machine `{data:{id,discarded}}`; human `<ten_…>  [discarded]`; identical for `204`/`404` | ✓ Conformant | machine via `RenderSuccess`, human via `writeHuman`/`TensionDiscardView`; `*_204And404StdoutIdentical` tests |
| Error Communication table (exit codes per status; `404` folded to success) | ✓ Conformant | all via shared `reportFailure`/`classifyClientError`; per-status unit tests (2/4/5/3/6) |

---

## Non-Behavior Absence

**Status**: Pass (8 of 8 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No create/list/get/update | ✓ Absent | leaf issues only `DELETE`; help describes removal alone |
| No hard-delete / proposal cascade | ✓ Absent | single `DELETE /tensions/{id}`, no cascade logic |
| No treating `404` as not-found failure | ✓ Absent | `isNotFound` folds `404` into success |
| No confirmation / `--force` / `--yes` | ✓ Absent | no such flags declared |
| No restore / un-discard | ✓ Absent | no such command |
| No `If-Match` precondition | ✓ Absent | request sets no `If-Match`; tests assert `lastIfMatch == ""` |
| No raw-JSON default / private format flag | ✓ Absent | renders via Output Format Selection only |
| No re-resolving base URL/token/header/exit codes | ✓ Absent | delegates to `assemble`/`newClient`/`Execute`/`reportFailure` |

---

## @wip Lifecycle Completion

**Status**: Pass

`@wip` remains only on the 3 `@validation` scenarios, which are deliberately held for this skill and are referenced by no checked task. All 10 behavioral scenarios referenced by T003 had their `@wip` removed by implement.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation, independently)

| Scenario | Status | Trace |
|---|---|---|
| Discard exposes no read or write verb of its own | ✓ Satisfied | `glassfrog tension discard --help` advertises removal only — no create/list/get/update, no field flags; `runTensionDiscard` issues a single `DELETE` and renders only the discard result |
| The synthesized result claims nothing the server did not return | ✓ Satisfied | `tensionDiscardResult{ID, Discarded}` and `TensionDiscardView{ID}` carry only the id + marker; no `discarded_at`. `TestRender_TensionDiscard_ClaimsOnlyTheID` + `TestRunTensionDiscard_StructuredJSON` assert no server-owned field |
| The `404` path leaks no not-found error | ✓ Satisfied | `isNotFound` branch writes an informational "already discarded" advisory and returns `Success`/exit 0; `TestRunTensionDiscard_AlreadyGoneIsSuccess` and the BDD `stderrNotesAlreadyGone` step assert no `404`/"not found" reaches stderr |

---

## Verdict: Ready

All 5 conformance dimensions pass. All 3 validation scenarios are satisfied through independent inspection, corroborated by the green unit, render, and godog suites (`go build`/`go vet`/`go test` clean). The implementation conforms to its specification — the synthesized client-side result, the `404`-as-success interception keyed on the exact status, the `204`-vs-`404` stderr advisory with identical stdout, and the no-new-machinery constraint (no `internal/glassfrog` model, no transport field, no new `Outcome`/`ExitCode`) are all present and faithful to plan ADR-1 through ADR-4.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop is closed for 045-tension-discard.
