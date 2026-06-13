# Validate: Role Fillers

**Feature**: 047-role-fillers
**Round**: 1 of 3
**Date**: 2026-06-13
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-cli.md, features/who-to-contact-for-a-role/role-fillers.feature, PROJECT.md
**Implementation files**: 3 areas — `internal/render` (FillersView + ResourceFillers + `fillers.full`/`fillers.compact` templates), `internal/cli/fillers.go` (the command), `internal/cli/app.go` (wiring); tests in `internal/render/fillers_test.go`, `internal/cli/fillers_test.go`, `internal/cli/role_fillers_bdd_test.go`

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

**Status**: Pass (8 of 8 driving scenarios covered)

All driving scenarios from spec.md (concretized in role-fillers.feature, all behavioral scenarios un-`@wip`'d) trace to identifiable code paths, each exercised by a passing godog scenario in `TestRoleFillersFeatures` (9 scenarios / 49 steps).

| Scenario (spec.md) | Status | Implementation |
|---|---|---|
| List the fillers of a role | ✓ Covered | `runFillersListWalk` walks `GET /roles/{id}/assignments` (fillers.go:113-114); `TestRunFillers_ListSuccessWalksAndProjects` |
| A filler row shows its focus and election expiry | ✓ Covered | `FillersView` + `fillers.full`/`compact` project `.Focus`/`.ElectedUntil`; `TestRunFillers_FocusAndElectionProjected`, render goldens |
| Fillers span both a person and an agent | ✓ Covered | row leads with `.Actor.ID` (`per_`/`agt_`) + `[.Actor.Kind]`; `TestRunFillers_PersonAndAgentDistinguished` |
| Role id does not exist (404) | ✓ Covered | id passed through `url.PathEscape` (fillers.go:113), classified by shared chain; `TestRunFillers_UnknownRoleSurfacesAPIStatus` → APIError/3 |
| No usable credential | ✓ Covered | `newClient`/Execute surfaces `*AuthError{NoCredentials}` via `reportFailure`; `TestRunFillers_NoCredentialsIsUsageError` → UsageError/2, no data |
| Role has no fillers | ✓ Covered | empty walk → `FillersView{}` renders `no fillers`; `TestRunFillers_EmptyIsCleanSuccess` → exit 0 |
| Missing role-id rejected before any request | ✓ Covered | `cobra.ExactArgs(1)` (fillers.go:253); `TestFillersCommand_RequiresExactlyOneArg` → UsageError, 0 calls |
| Paginated list with first-page opt-out | ✓ Covered | `runFillersFirstPage` single Execute + `moreFillersNote`; `TestRunFillers_FirstPageStopsAndSignals` |

The completeness accord's mid-walk-failure path (spec § Completeness; feature "A mid-walk failure yields a partial set flagged incomplete") is covered by `reportIncompleteFillersWalk` (fillers.go:200) and `TestRunFillers_MidWalkFailurePartialAndIncomplete` — partial set printed, incomplete note on stderr, non-zero via `classifyClientError(Stop)`.

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — `fillers` render key + FillersView + templates | ✓ Met | `ResourceFillers` in `builtinResources` (registry guard passes), both formats pinned by goldens, explicit-absence markers `(none)`/`(not an elected seat)`/`—`, focus verbatim, `no fillers` empty line; `internal/render` imports neither `cli` nor `apiclient`; build/vet clean |
| T002 — `fillers <role-id>` command + seam + wiring | ✓ Met | walks every page, empty→`no fillers`/exit 0, default `actor` include (no `--include` declared), `ExactArgs(1)` no-request tripwire, `--first-page` note, mid-walk partial+incomplete, full error-class table, structured `{data:[…]}` aggregation; wired in `Assemble()`; no new Outcome/ExitCode/validator/root flag; never reads `ctx.Cred.Token` |
| T003 — godog suite over role-fillers.feature | ✓ Met | `TestRoleFillersFeatures` Paths names only `role-fillers.feature`; 9 behavioral scenarios pass, 4 `@validation` held; reuses sibling step phrasings; helpers return errors, never panic; full module `go test ./...` clean |

## Interface Contract Conformance

**Status**: Pass (the single `fillers <role-id>` surface conforms to interface-cli.md)

| Surface element (interface-cli.md) | Status | Evidence |
|---|---|---|
| `glassfrog fillers <role-id>`, `ExactArgs(1)`, guard-registered, `SilenceErrors`/`SilenceUsage` | ✓ Conformant | fillers.go:243-254; wired via `MustRegister` in app.go |
| No filter flags, no `--include` — only `--first-page`/`--per-page` + inherited `--base-url`/`-o` | ✓ Conformant | only two local flags declared (fillers.go:285-286); `TestFillersCommand_UnknownFlagRejectedNoRequest` rejects `--include`/`--kind`/`--status`/`--query` before any request |
| Reads `GET /roles/{role_id}/assignments`, id via `url.PathEscape`, no local validation | ✓ Conformant | fillers.go:113-114 |
| Output: `full`/`compact` human projection; `json`/`yaml` aggregated `{data:[…]}` (per-page meta dropped), never raw envelope | ✓ Conformant | two-track walk (fillers.go:121-159); `TestRunFillers_StructuredJSONEmitsAggregatedRawPayload` confirms raw payload, no human labels, no `pagination` |
| Validation order: `--output` resolved before assembly; usage error sends no request | ✓ Conformant | `resolveRenderTarget` first (fillers.go:84); `TestRunFillers_BadOutputIsUsageErrorNoRequest` → 0 calls |
| Completeness: walk-by-default / `--first-page` note exit 0 / mid-walk partial+incomplete non-zero | ✓ Conformant | `moreFillersNote`, `incompleteFillersWalkNote` |
| Error table → exit codes via shared `classifyClientError`; no new Outcome/ExitCode | ✓ Conformant | `TestRunFillers_NoCredentialsIsUsageError`/`_UnknownRoleSurfacesAPIStatus`/`_TransportErrorIsNetworkUnavailable`/`_Non2xxClassifies` (403→4, 429→5, 500→3); `git diff` shows no exitcode.go/dispatch.go change |

## Non-Behavior Absence

**Status**: Pass (all 7 non-behaviors honored)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| No create/update/remove of an assignment | ✓ Absent | fillers.go issues `http.MethodGet` only — no POST/PATCH/DELETE |
| No single-assignment read command | ✓ Absent | only `newFillersCommand` registered; no `filler <asgn-id>` command exists |
| No `--include` flag | ✓ Absent | only `--first-page`/`--per-page` declared; `--include` is a rejected unknown flag |
| No client-side filters (kind/name/focus) | ✓ Absent | no `--kind`/`--query`/`--status`; `runFillersListWalk` builds a bare request (no `Query`) |
| No duplication of Actor Directory (048) discovery | ✓ Absent | reads `/roles/{id}/assignments` and renders `glassfrog.Assignment` (FillersView), distinct from 048's `/actors` + ActorsView |
| No raw API JSON fixed default, no own format flag | ✓ Absent | format resolved through 020 (`resolveRenderTarget`); no private format flag |
| No re-implementation of base-URL/token/header/non-2xx typing/exit codes | ✓ Absent | reuses `assemble`/`newClient`/`classifyClientError`/`reportFailure`; no new Outcome/ExitCode (git diff confirms); never reads `ctx.Cred.Token` |

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios referenced by the checked tasks have had `@wip` removed and pass under `TestRoleFillersFeatures`. The 4 `@validation`-tagged scenarios retain `@wip` — correctly held out for this validation pass (not referenced as implementable by any checked task), so they are excluded from the lifecycle check. No stray `@wip` remains on a behavioral scenario.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation, independent of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| The filler's name appears without an include flag | ✓ Satisfied | `fillers.full`/`compact` render `.Actor.Name` + `[.Actor.Kind]` from the default `actor` include; the command declares no `--include` flag and sends no `include` param (`TestRunFillers_NameAndKindShownWithoutIncludeFlag` asserts name+kind shown and no `include`/`kind`/`q`/`role_id` param) |
| Focus and election are projected, not dropped | ✓ Satisfied | both fields rendered under `full`/`compact`; nullable `Focus`→`(none)`/`—`, `ElectedUntil`→`(not an elected seat)`/`—` (render goldens `TestRender_FillersFull_FocusAndElectionProjected` / `_AbsentFocusAndElectionShowMarkers`; CLI `TestRunFillers_FocusAndElectionProjected`) |
| A missing token costs no request | ✓ Satisfied | the 007 `AuthTransport` refuses to send when `Cred.Source == None`, short-circuiting before the wire — the base (tripwire) transport is never called, so no request is issued (`TestRunFillers_NoCredentialsIsUsageError` → UsageError/2, no data printed). *Note on phrasing*: the connection is assembled and the client built before the auth-transport refusal fires; the spec's "before assembling the connection" is the established 007 fail-safe ordering (refuse before the wire), and the load-bearing guarantee — "no request issued" — holds exactly as written. |
| Output is structured, not pre-rendered | ✓ Satisfied | the command supplies structured data (a `FillersView` for human, raw per-record bytes aggregated via `aggregateRawData` for `json`/`yaml`) into the 020 selection and defines no format flag; all four formats render from the same walked result (`TestRunFillers_StructuredJSONEmitsAggregatedRawPayload` confirms the aggregated `{data:[…]}` with no human projection and no `pagination` meta) |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 4 validation scenarios are satisfied through independent inspection (and corroborated by the existing unit + render tests). The implementation conforms to its specification: the single role-scoped `fillers <role-id>` read, the reused `glassfrog.Assignment` model with the new `fillers` render key, the no-filters/no-`--include` input surface, the walk-to-completion completeness contract with explicit incompleteness signalling, and the shared error/exit-code chain — each is present and behaves as the spec, plan, and interface promised. The non-behaviors (no writes, no singular read, no `--include`, no client-side filters, no forked output/error machinery) are all honored.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 4 `@validation` scenarios remain `@wip` by design (held-out verification); they were traced by inspection here rather than executed, since their step phrasings are unique to the held set — leaving them `@wip` is correct and they can stay as a permanent independent-verification record.
