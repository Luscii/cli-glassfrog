# Validate: Guarded Writes

**Feature**: 053-guarded-writes
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/clobbered-changes/guarded-writes.feature, PROJECT.md
**Implementation files**: 2 production (`internal/apiclient/client.go`, `internal/apiclient/execute.go`); 2 test (`internal/apiclient/execute_test.go`, `internal/apiclient/guarded_writes_bdd_test.go`)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 2 of 2 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

All seven driving scenarios have identifiable implementation code paths, each pinned by both an executable BDD scenario and a direct unit test.

| Scenario | Status | Implementation |
|---|---|---|
| Captured version guards the write | ✓ Covered | `execute.go:137-146` (conditional `Header.Set("If-Match", …)`); `execute_test.go:TestExecuteSetsIfMatchWhenPresent` |
| No version falls through to an unconditional write | ✓ Covered | `execute.go:137` (`if req.IfMatch != ""` guard); `execute_test.go:TestExecuteOmitsIfMatchWhenEmpty` |
| A delete is guarded the same way | ✓ Covered | `execute.go:137-146` (method-agnostic — keyed only on the field); `execute_test.go:TestExecuteSetsIfMatchOnDelete` |
| Server refuses a stale write (412) | ✓ Covered | `execute.go:151-155` (existing generic `*ResponseError` path, unchanged); BDD "A stale guarded write is refused untouched" |
| Empty captured version is not sent | ✓ Covered | `execute.go:137` (empty fails the guard); BDD "An empty captured version sends no precondition" |
| Weak-validator version forwarded verbatim | ✓ Covered | `execute.go:145` (`Header.Set` with no transformation); `execute_test.go:TestExecuteForwardsWeakValidatorIfMatchVerbatim` |
| Precondition composes with a content type | ✓ Covered | `execute.go:128-146` (separate `Content-Type` and `If-Match` blocks); `execute_test.go:TestExecuteIfMatchAndContentTypeAreIndependent` |

---

## Acceptance Criteria

**Status**: Pass (T001 checked — all 6 criteria met)

| Criterion (T001) | Status | Evidence |
|---|---|---|
| Non-empty `IfMatch` → `If-Match` header verbatim | ✓ Met | `execute.go:145`; `TestExecuteSetsIfMatchWhenPresent` asserts value `a1b2c3` |
| Empty/unset `IfMatch` → no header, unconditional, not an error | ✓ Met | `execute.go:137` guard; `TestExecuteOmitsIfMatchWhenEmpty` |
| Method-agnostic (DELETE guarded like PUT/PATCH) | ✓ Met | keyed on field only; `TestExecuteSetsIfMatchOnDelete` + BDD method-agnostic replay |
| `If-Match` and `Content-Type` independent | ✓ Met | distinct blocks; `TestExecuteIfMatchAndContentTypeAreIndependent` |
| Every landed read/write byte-identical (no new `Outcome`/exit code; `412` rides existing path) | ✓ Met | field zero-values to `""`; grep confirms no production call site sets `IfMatch`; no `412`/`Outcome` handling added |
| Unit tests cover the 5 listed cases | ✓ Met | 5 send unit tests present (`TestExecuteSetsIfMatchWhenPresent`, `…OmitsIfMatchWhenEmpty`, `…ForwardsWeakValidatorIfMatchVerbatim`, `…SetsIfMatchOnDelete`, `…IfMatchAndContentTypeAreIndependent`) |

---

## Interface Contract Conformance

**Status**: Pass (1 of 1 surface conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| `Request.IfMatch string` field on `Request` in `client.go`, beside `ContentType`; no new type, no `Response`/`executor`/`NewClient`/`buildURL` change | ✓ Conformant | `client.go:45-61` — narrow field with the doc-comment contract; no other type touched |
| Conditional `If-Match` set in `Execute`, immediately after the `ContentType` block, verbatim and only when non-empty | ✓ Conformant | `execute.go:137-146` — placed directly after the `Content-Type` block (`execute.go:128-135`), no new imports |

The contract latitude the interface left open (field name `IfMatch`, doc wording, exact block placement) is honored within the verbatim / empty-as-absent / method-agnostic shape.

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions upheld)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not perform the read that obtains the version | ✓ Upheld | No read added; `Execute` makes exactly one `Do` (`execute.go:147`) for the caller's own request |
| Must not interpret/relabel/specially-handle the `412` | ✓ Upheld | No `412`/`Precondition` branch in production code; refusal rides the unchanged generic `*ResponseError` (`execute.go:151-155`) |
| Must not send `If-Match` when no version is present | ✓ Upheld | `if req.IfMatch != ""` guard (`execute.go:137`) |
| Must not interpret/unquote/normalize/validate the token | ✓ Upheld | `Header.Set("If-Match", req.IfMatch)` forwards the value with zero transformation |
| Must not wire any specific production write command | ✓ Upheld | Production `IfMatch` references confined to `client.go`/`execute.go`; CLI tension update/discard tests still assert **no** `If-Match` is sent (`tension_update_test.go:57`, `tension_discard_test.go:57`) |
| Must not change the outbound request when no version is supplied | ✓ Upheld | Field zero-values to `""`; no landed call site sets it, so every existing request stays byte-identical |

---

## @wip Lifecycle Completion

**Status**: Pass

The 7 non-validation scenarios referenced by checked task T001 have their `@wip` tags removed and pass executably (7 scenarios / 27 steps). The 2 `@validation @wip` scenarios remain tagged — correctly held out from the Builder for independent verification (lines 71, 79 of the feature file).

---

## Validation Scenario Results

**Status**: Satisfied (2 of 2 scenarios traced independently)

| Scenario | Status | Trace |
|---|---|---|
| Guarded Writes wires no production write command | ✓ Satisfied | Production `IfMatch` appears only in the shared mechanism (`client.go`, `execute.go`); no write command populates it, and the CLI tension commands' tests assert the field stays empty (last-write-wins) |
| The 412 refusal is left for downstream surfacing | ✓ Satisfied | The mechanism produces the `412` condition (a guarded write can be refused) but adds no interpretation — the refusal surfaces as the existing generic `*ResponseError{StatusCode: 412, …}`; distinct surfacing remains for Stale-Write Surfacing (054) |

Both validation scenarios are mechanism/artifact-level (`Given the produced artifact` / `Given the guarded-write send mechanism`), verified by inspection — there is no separate runtime path to execute.

---

## Verdict: Ready

All 5 conformance dimensions pass and both validation scenarios are satisfied through inspection. The implementation delivers exactly the shaped mechanism — a narrow `Request.IfMatch` field sent verbatim as `If-Match` only when non-empty, method-agnostic, independent of `Content-Type` — and nothing more: no read, no `412` interpretation, no production command wired, every existing request byte-identical. The full test suite (`go test ./...`) and `go vet ./...` are green.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The capture (052) → **send (053)** → surface (054) split is preserved; the named downstream consumers (each write command's read-then-write retrofit, and Stale-Write Surfacing 054) build on this seam.
