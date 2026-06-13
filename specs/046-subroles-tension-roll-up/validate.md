# Validate: Subroles Tension Roll-up

**Feature**: 046-subroles-tension-roll-up
**Round**: 1 of 3
**Date**: 2026-06-13
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/tension-capture/subroles-tension-roll-up.feature, PROJECT.md
**Implementation files**: 2 production (`internal/cli/tension_reads.go` — `tensionsConfig.subroles`, `tensionsListPath`, `newTensionSubrolesCommand`; `internal/cli/tension.go` — group wiring), 2 test (`internal/cli/tension_subroles_test.go`, `internal/cli/tension_subroles_bdd_test.go`)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 4 of 4 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 behavioral scenarios covered)

Each scenario from spec.md § Driving Scenarios (and its `.feature` concretization) traces to a code path. The 9 behavioral scenarios run executably in `TestSubrolesTensionRollUpFeatures` (51 steps, all passing).

| Scenario | Status | Implementation |
|---|---|---|
| Roll up tensions across a circle's direct sub-roles | ✓ Covered | `newTensionSubrolesCommand` → `runTensionList{subroles:true}` → `tensionsListPath` → `GET /roles/{id}/subroles/tensions`; `paging.All` walk + `render.TensionsView` |
| Narrow the roll-up by status | ✓ Covered | `tensionsQuery` sends `status` when non-empty (tension_reads.go:231) after `validateTensionStatus` |
| Roll-up walks every page to completion | ✓ Covered | default branch walks `paging.All[Tension]` / `[json.RawMessage]` (tension_reads.go:128/146) |
| Anchor is a leaf role (404) | ✓ Covered | walk first-page error with zero records → `reportFailure`/`classifyClientError` (tension_reads.go:133/153); no special case |
| No usable credential | ✓ Covered | shared `classifyClientError` → `UsageError(2)`, not-authenticated message (reused chain) |
| Sub-roles exist but carry no tensions | ✓ Covered | empty-200 → `res.Stop==nil`, empty records → `tensions` empty line `no tensions`, exit 0 |
| Unsupported status rejected before any request | ✓ Covered | `validateTensionStatus` fail-fast (tension_reads.go:95) before `seam.assemble` (102) |
| Paginated roll-up with first-page opt-out | ✓ Covered | `runTensionListFirstPage` + `moreTensionsNote` (tension_reads.go:166-211) |
| Mid-walk failure yields a partial set flagged incomplete | ✓ Covered | partial render + `reportIncompleteTensionsWalk` (tension_reads.go:160/168) |

## Acceptance Criteria

**Status**: Pass (both checked tasks, all criteria met)

- **T001** — every criterion traced: page walk + projection + exit 0; empty-line success; leaf-404 → `APIError(3)`/`PermissionError(4)` with no "no sub-roles" message (unit test `TestRunTensionSubroles_LeafAnchor404IsFailure`); `--status` sent only when supplied; unsupported `--status` → `UsageError(2)` naming value + set, no request (`...UnsupportedStatusIsUsageErrorNoRequest`); `--first-page` one-page + note; mid-walk partial + non-zero; `*AuthError` → 2, `*TransportError` → 6, base-URL/`--output` → 2; structured `{data:[…]}` vs human projection vs template; no new `Outcome`/`ExitCode`/root flag/render key/validator; token never printed; `go build`/`go vet` clean.
- **T002** — 9 behavioral scenarios un-`@wip` and passing; 4 `@validation` kept `@wip`; suite `Paths` names only `subroles-tension-roll-up.feature`; leaf-404 (`cannedTransport{status:404}`) and empty-200 (`tensionsPageEmpty`) exercise distinct fakes with distinct outcomes; offline only; suites clean.

## Interface Contract Conformance

**Status**: Pass (single surface conformant)

| Surface element | Status | Evidence |
|---|---|---|
| `tension subroles <role-id>` leaf — `ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`, guard-registered | ✓ Conformant | `newTensionSubrolesCommand`; `MustRegister` in `newTensionCommand` (tension.go) |
| Reads `GET /roles/{role_id}/subroles/tensions` | ✓ Conformant | `tensionsListPath` subroles branch |
| List flags `--status` (validated, sent on Changed+non-empty) / `--first-page` / `--per-page` (presence) | ✓ Conformant | flags declared tension_reads.go:321-323; `perPageSet` via `Changed` |
| Inherited `--base-url`, `-o/--output`; no own format flag | ✓ Conformant | reads `apiclient.FlagBaseURL` / `output.FlagOutput`; only the 3 list flags declared |
| Output: `json`/`yaml` aggregated `{data:[…]}`, `full`/`compact` human projection, empty → `no tensions` + exit 0 | ✓ Conformant | `aggregateRawData` / `writeHuman` over `render.ResourceTensions` |
| Validation order (`--output` then `--status`), no-request-on-rejection | ✓ Conformant | `resolveRenderTarget` then `validateTensionStatus`, both before assembly |
| Error Communication: leaf-404 surfaced verbatim via shared classifier, no new `Outcome`/`ExitCode` | ✓ Conformant | `reportFailure`/`classifyClientError`; no `isNotFound`/special-case in the runner |

## Non-Behavior Absence

**Status**: Pass (9 of 9 exclusions respected)

| Non-behavior | Status | Evidence |
|---|---|---|
| No roll-up beyond direct sub-roles | ✓ Absent | exactly one paginated read of `/subroles/tensions`; no grand-child fetch in code |
| No listing of the anchor's own tensions | ✓ Absent | `tensionsListPath` subroles branch is distinct from the `list` path |
| No leaf-404 special-casing | ✓ Absent | no `isNotFound`/empty-success fold in the runner; routes through `reportFailure` |
| No `--subroles` flag on `tension list` | ✓ Absent | implemented as a distinct verb leaf, not a flag |
| No create/update/discard | ✓ Absent | bodyless `GET` only |
| No status set/override; `--status` is filter-only | ✓ Absent | `--status` only feeds `tensionsQuery`; nothing written |
| No raw-JSON default / own format flag | ✓ Absent | dispatches through 020's `resolveRenderTarget` |
| No own base-URL/token/header/exit-code path | ✓ Absent | reuses `seam.assemble`/`newClient`/`classifyClientError` |
| No interpretation/summary/advice | ✓ Absent | renders tension data only |

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios referenced by checked tasks T001/T002 have their `@wip` removed and pass. The 4 `@validation` scenarios (not referenced by any checked task — held for this skill) correctly retain `@wip`. No stray `@wip` remains on implemented scenarios.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation)

These scenarios were held out from the Builder (kept `@wip`). Traced independently against the code; each behavior is additionally pinned by a passing unit test.

| Scenario | Status | Trace |
|---|---|---|
| The roll-up is one level only | ✓ Satisfied | `tensionsListPath` yields exactly `/roles/{id}/subroles/tensions`; the walk follows only that endpoint's cursor — no code reads grand-child roles or issues a secondary request |
| A leaf-role 404 is a failure, not an empty success | ✓ Satisfied | two distinct paths: first-page error + zero records → `reportFailure` → `APIError(3)`, non-zero; empty-200 → `no tensions`, exit 0. Pinned by `TestRunTensionSubroles_LeafAnchor404IsFailure` vs `_EmptyIsCleanSuccess` |
| An unsupported status costs no request | ✓ Satisfied | `validateTensionStatus` (line 95) precedes `seam.assemble` (line 102); `TestRunTensionSubroles_UnsupportedStatusIsUsageErrorNoRequest` asserts `tr.calls == 0` |
| Output is structured, not pre-rendered | ✓ Satisfied | command produces `render.TensionsView`/raw records via `aggregateRawData`/`writeHuman`; declares only `--status`/`--first-page`/`--per-page` (no format flag); `TestRunTensionSubroles_StructuredJSONEmitsAggregatedRawPayload` confirms the aggregated `{data:[…]}` document |

**Note**: the 4 `@validation` scenarios remain `@wip` (not executable in the godog suite), per the held-out contract. Their behaviors are nonetheless covered by passing unit tests as cited above, so verification is inspection-based plus incidental test coverage — not a gap.

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios satisfied through inspection (and corroborated by passing unit tests). The implementation conforms to the specification: a single new `tension subroles <role-id>` leaf that path-swaps the landed `tension list` runner, reuses 042's model + 043's `tensions` render and `validateTensionStatus` with no new model/render/validator/`Outcome`/`ExitCode`/flag, and surfaces the leaf-anchor 404 verbatim — distinct from the empty-list success. `go build ./...`, `go vet ./...`, and `go test ./...` are clean.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 4 `@validation` scenarios may be un-`@wip`'d in a follow-up if the team wants them executable in CI (their behaviors are already pinned by unit tests).
