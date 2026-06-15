# Validate: Proposal Creation

**Feature**: 055-proposal-creation
**Round**: 1 of 3
**Date**: 2026-06-15
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, interface-spec.md, features/proposal-write-flow/proposal-creation.feature, PROJECT.md
**Implementation files**: 5 — `internal/glassfrog/proposal.go`, `internal/cli/proposalchanges.go`, `internal/cli/proposal.go`, `internal/render/render.go` (+ `templates/proposal.{full,compact}.tmpl`), wiring in `internal/cli/app.go`; tests in `internal/glassfrog/proposal_test.go`, `internal/cli/proposalchanges_test.go`, `internal/cli/proposal_test.go`, `internal/cli/proposal_creation_bdd_test.go`, `internal/render/proposal_test.go`

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

**Status**: Pass (10 of 10 scenarios covered)

Every driving scenario has an identifiable code path, and each is exercised by the executable `TestProposalCreationFeatures` godog suite (13 behavioral scenarios green) plus targeted unit tests.

| Scenario | Status | Implementation |
|---|---|---|
| Create a proposal with an inline change set | ✓ Covered | `runProposalCreate` → POST `/proposals`, `proposal.go`; `TestRunProposalCreate_PostsProposalVerbatim` |
| Read the change set from a file | ✓ Covered | `resolveChangesSource` existing-regular-file arm, `proposalchanges.go`; `TestRunProposalCreate_FileSource` |
| The created proposal's id and status are visible in JSON output | ✓ Covered | machine branch → `output.RenderSuccess` raw `{data}`, `proposal.go`; `TestRunProposalCreate_StructuredJSONEmitsRawPayload` |
| Missing change set is rejected before any request | ✓ Covered | `--changes` presence check, `proposal.go`; tripwire `calls==0` |
| No usable credential | ✓ Covered | `reportFailure`/`classifyClientError` on `*AuthError{NoCredentials}` → UsageError(2) |
| Empty change set is rejected before any request | ✓ Covered | `validateChanges` empty-array arm |
| Unparseable change set is rejected before any request | ✓ Covered | `validateChanges` non-JSON/non-array arm |
| A change without a type is rejected before any request | ✓ Covered | `validateChanges` per-element `type` probe |
| Read the change set from piped stdin | ✓ Covered | `resolveChangesSource` reserved-`stdin` arm via `readBoundedStdinN` |
| Premium async proposals not enabled (403) / Unknown anchor tension (404) | ✓ Covered | shared classifier → PermissionError(4) / APIError(3); `TestRunProposalCreate_PremiumDeniedIsPermission`, `_UnknownTensionSurfacesAPIStatus` |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks checked, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — Proposal model + `CreateProposalRequest` | ✓ Met | snake-case binding, null→empty, unknown-field tolerance, free-form change-key preservation, verbatim request marshal; 6 tests; `internal/glassfrog` imports neither `cli` nor `apiclient` |
| T002 — `resolveChangesSource` + `validateChanges` | ✓ Met | stdin/file/inline classification + TTY/empty/regular-file guards; type floor; pure over injected sources; 15 tests |
| T003 — singular `proposal` render key | ✓ Met | `ProposalView`, `proposal.full/compact`, registered in `builtinResources`; absence guards on nullable fields; 3 goldens + exhaustiveness guard |
| T004 — `proposal create` command | ✓ Met | POST shape + verbatim changes, `application/json`, no `If-Match`, no new Outcome/ExitCode, classifier mapping, `ExactArgs(1)`, group guard, no token read; 18 tests; wired in `Assemble()` |
| T005 — executable acceptance | ✓ Met | new `TestProposalCreationFeatures` suite (Paths = the one feature file), 13 behavioral scenarios un-`@wip`'d, 3 `@validation` held; single-quote-aware `splitArgsPOSIX` + its unit test; 429 fake proves one POST |

`go build ./...`, `go vet ./...`, and `go test ./...` all pass.

---

## Interface Contract Conformance

**Status**: Pass (CLI surface and Go API surface both conformant)

| Surface (interface-cli.md / interface-spec.md) | Status | Implementation |
|---|---|---|
| `glassfrog proposal` non-runnable group (≥1 child, no action) | ✓ Conformant | `newProposalCommand` — no `RunE`, parents `create`; `TestProposalCommand_GroupRegistersUnderGuard` |
| `glassfrog proposal create <tension-id>` leaf, `ExactArgs(1)`, `SilenceErrors/Usage` | ✓ Conformant | `newProposalCreateCommand` |
| `--changes` flag (required; inline / file / reserved `stdin`) | ✓ Conformant | declared only on `create`; `resolveChangesSource` |
| Validation order: `--output` → `--changes` presence → source read → type floor (all pre-assembly) | ✓ Conformant | `runProposalCreate` steps 1–3; tripwire confirms no request on rejection |
| Request: one POST `/proposals`, `{proposal:{tension_id, changes verbatim}}`, `Content-Type: application/json`, no `If-Match` | ✓ Conformant | `proposal.go`; `glassfrog.NewCreateProposalRequest` |
| Output: machine raw `{data: Proposal}` via 018; human via shared singular `proposal` view | ✓ Conformant | `MachineFormat` branch + `writeHuman` over `ProposalView` |
| Error table: NoCredentials→2, CredentialError→1, 403/401→4, 404/422→3, 429→5, transport→6, bad `--output`/base-URL→2 | ✓ Conformant | `reportFailure`/`classifyClientError` reused; unit tests for each arm; no new Outcome/ExitCode |
| Go API: `Proposal`/`ProposalChange`/`ResponseSummary` + `CreateProposalRequest`, `proposalSeam` (= tensionSeam + `readChangesSource`) | ✓ Conformant | `internal/glassfrog/proposal.go`, `internal/cli/proposal.go` |

---

## Non-Behavior Absence

**Status**: Pass (8 of 8 non-behaviors absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No advance/respond/withdraw/list/get | ✓ Absent | group attaches only `create` (`proposal.go:192`) — confirmed by help output |
| No `type`-value or command-key validation | ✓ Absent | `validateChanges` decodes only `{type string}`; changes carried as `[]json.RawMessage` verbatim |
| No client-side Premium pre-check | ✓ Absent | `runProposalCreate` issues the request unconditionally; 403 classified server-side |
| No status set/override | ✓ Absent | `CreateProposalBody` has no `status` field; no `--status` flag |
| No proposer inference | ✓ Absent | `CreateProposalBody` has no `proposer` field; no proposer flag; token never read |
| No private output flag / no fixed raw-JSON default | ✓ Absent | uses inherited `-o`/`--output` (020); dispatches via `RenderSuccess`/`writeHuman` |
| No `If-Match` / optimistic concurrency | ✓ Absent | `apiclient.Request.IfMatch` left `""`; tests assert `lastIfMatch == ""` |
| No base-URL/token/header/exit-code re-implementation | ✓ Absent | reuses `assemble`/`newClient`/`AuthTransport`/`classifyClientError`/`ExitCode` seams |

---

## @wip Lifecycle Completion

**Status**: Pass

`features/proposal-write-flow/proposal-creation.feature` carries `@wip` on exactly the 3 `@validation` scenarios (held for this skill). Every behavioral scenario referenced by checked task T005 has had its `@wip` removed and runs green in `TestProposalCreationFeatures` under the `~@wip` filter. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| Create does not reach into the rest of the write flow or the reads | ✓ Satisfied | `newProposalCommand` attaches only `create`; `glassfrog proposal --help` advertises only `create`; no propose/respond/withdraw/list/get code path exists in `internal/cli`/`internal/glassfrog` for proposals |
| The change set is sent through verbatim beyond the type floor | ✓ Satisfied | `validateChanges` reads only each element's `type` and returns the `[]json.RawMessage` slice unmodified; the body marshals it byte-for-byte. `TestRunProposalCreate_PreservesExtraChangeKeys` and `TestValidateChanges_AcceptsTypedArrayVerbatim` pin that command-specific keys ride through untouched |
| No client-side feature-gate pre-check | ✓ Satisfied | `runProposalCreate` contains no Premium/feature-gate branch; it always reaches `Execute`. `TestRunProposalCreate_PremiumDeniedIsPermission` asserts the request is issued (`calls == 1`) and the server `403` surfaces as PermissionError(4) |

These 3 scenarios remain `@wip`-tagged in the feature file (held out from the Builder); they are verified here by inspection, not by the godog suite.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 held-out validation scenarios trace to clear code paths. The implementation conforms to the specification: the second write lands as a `proposal` group + `create` leaf, sends one POST `/proposals` with a verbatim change set above a `type` floor, reuses the landed transport/retry/render/classifier seams without change, and honors every non-behavior (no status/proposer/If-Match, no Premium pre-check, no reach into the rest of the write flow). `go build`/`go vet`/`go test ./...` are green.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The specification loop is closed for 055.

Note for the future specification cycle: the `proposal` group, the `Proposal`/`ProposalChange`/`ResponseSummary` model, and the singular `proposal` render key are now landed by 055. **Proposal Reads (056)** is the follower under the first-to-land-creates contract — it must attach its `list`/`get` leaves to the existing group, reuse the model, and grow (not duplicate) the singular `proposal` view to render changes by type.
