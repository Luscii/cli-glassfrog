# Validate: Actor Directory

**Feature**: 048-actor-directory
**Round**: 1 of 3
**Date**: 2026-06-12
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/actors-disconnected-from-governance/actor-directory.feature, PROJECT.md
**Implementation files**: `internal/cli/actors.go` (command + `actorsSeam` + `validateKind`), `internal/cli/app.go` (`Assemble()` wiring), `internal/render/render.go` (`ActorsView` + `ResourceActors`), `internal/render/templates/actors.{full,compact}.tmpl`; tests: `internal/cli/actors_test.go`, `internal/cli/actor_directory_bdd_test.go`, `internal/render/actors_test.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 3 of 3 @validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 scenarios covered)

All driving scenarios from spec.md § Driving Scenarios have identifiable code paths; the 10 behavioral feature scenarios also pass executably (`TestActorDirectoryFeatures`, 56 steps).

| Scenario | Status | Implementation |
|---|---|---|
| List every actor in the organization | ✓ Covered | `actors.go:runActorsListWalk` walks `GET /actors` via `paging.All`; no filter sent (`actorsQuery` returns nil) |
| Find the actors filling a role | ✓ Covered | `actorsQuery` sets `role_id` when `roleSet && roleID != ""` (`actors.go:256`) |
| Narrow the directory to agents | ✓ Covered | `validateKind` accepts `agent`; `actorsQuery` sets `kind` (`actors.go:253`) |
| No usable credential | ✓ Covered | `reportFailure` over `*AuthError{NoCredentials}` → `UsageError(2)` via shared `classifyClientError` |
| Malformed role-id rejected by the API | ✓ Covered | `role_id` passed through unvalidated; non-2xx surfaces via `reportFailure`/`classifyClientError` (`actors_test.go:TestRunActors_MalformedRoleIDSurfacesAPIStatus`) |
| No actor matches the filters | ✓ Covered | empty walk renders the `no actors` line, exits 0 (`actors_test.go:TestRunActors_EmptyIsCleanSuccess`) |
| Paginated directory with first-page opt-out | ✓ Covered | `runActorsFirstPage` does one `Execute`, writes `moreActorsNote` on `HasNextPage`, exits 0 |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — `actors` render key + `ActorsView` + templates | ✓ Met | `ResourceActors` in `builtinResources`; full/compact templates render id+kind+name with `no actors` empty line; registry-exhaustiveness guard passes at 15 keys; `internal/render` imports neither `cli` nor `apiclient` |
| T002 — `actors` command + `--kind`/`--role-id`/`--query`/`--first-page`/`--per-page` + completeness + wiring | ✓ Met | `cobra.NoArgs`; output-first then `--kind` validation pre-assembly; walk-to-completion + first-page + mid-walk partial; filters `Changed()`-gated and carried every page; `MustRegister` in `Assemble()`; no new `Outcome`/`ExitCode`/root flag; token never read |
| T003 — executable acceptance | ✓ Met | `TestActorDirectoryFeatures` Paths names only `actor-directory.feature`; 10 behavioral scenarios pass; `@validation` held `@wip`; reuses shared fakes/phrasings |

---

## Interface Contract Conformance

**Status**: Pass (surface, filter flags, output shapes, error table all conformant)

| Contract element | Status | Evidence |
|---|---|---|
| `glassfrog actors`, `cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage` | ✓ Conformant | `newActorsCommand` (`actors.go`) |
| `--kind` (local reject-unknown), `--role-id`, `--query`/`-q`, `--first-page`, `--per-page` | ✓ Conformant | flag declarations `actors.go:361-365`; `-q` short alias pinned (`TestActorsCommand_ShortQueryAlias`) |
| Inherited `--base-url` / `-o`/`--output` (not redeclared) | ✓ Conformant | reads `apiclient.FlagBaseURL` / `output.FlagOutput`; declares no own format flag |
| `full` shape `<id>  [<kind>]` + indented `Name:  <name>` | ✓ Conformant | `actors.full.tmpl`; golden `TestRender_ActorsFull_MixedKinds_Golden` |
| `compact` shape `<id>  [<kind>]  <name>` | ✓ Conformant | `actors.compact.tmpl`; golden `TestRender_ActorsCompact_MixedKinds_Golden` |
| Empty list → `no actors`; structured `{data:[]}` | ✓ Conformant | template empty line; structured via `aggregateRawData` over `Page[json.RawMessage]` |
| Error table (no-token→2, transport→6, non-2xx→3/4/5, bad output→2, unsupported kind→2) | ✓ Conformant | routed through shared `classifyClientError`/`reportFailure`; `TestRunActors_Non2xxClassifies`, `..._NoCredentialsIsUsageError`, `..._TransportErrorIsNetworkUnavailable`, `..._BadOutputIsUsageErrorNoRequest` |
| Unsupported `--kind` message names value + `agent, human` | ✓ Conformant | `validateKind`→`validateClosedFlagSet`; `TestValidateKind` |
| Output resolved before kind (error precedence) | ✓ Conformant | `runActorsList` order; `TestRunActors_OutputResolvedBeforeKind` |

---

## Non-Behavior Absence

**Status**: Pass (all 5 exclusions absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No single-actor read / `?include=` embed | ✓ Absent | only `Path: "/actors"`; no `/actors/{id}` path, no `include` query |
| No separate `people`/`agents` commands | ✓ Absent | `app.go` registers only `newActorsCommand`; no people/agents command exists |
| No create/invite/update/delete | ✓ Absent | only `http.MethodGet` issued |
| No raw JSON fixed default / own format flag | ✓ Absent | reads inherited `output.FlagOutput`; default human (`full`); structured only under explicit `-o json`/`yaml` |
| No base-URL/token resolution, header attach, non-2xx typing, exit-code choice | ✓ Absent | delegates to `actorsSeam`/`RetryExecutor`/`classifyClientError`; never reads `ctx.Cred.Token`; adds no `Outcome`/`ExitCode` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 10 behavioral scenarios referenced by checked task T003 have `@wip` removed and pass executably. The 4 `@validation @wip` scenarios correctly retain `@wip` — held for this validate step (their step definitions were not written into the Builder's suite, per the create/evaluate separation).

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

> The spec defines 3 held-out @validation scenarios; the feature file expresses them as 4 `@validation` scenario blocks ("A rejected kind issues no request" + "Default output carries no raw API envelope" + "Agent discovery reaches the ungated unified endpoint" + "The filters are carried on every page of the walk"). All trace to clear code paths through inspection. They carry no step definitions in the Builder's suite (correctly held), so verification is inspection-based, not executed.

| Scenario | Status | Trace |
|---|---|---|
| An unsupported kind costs no request | ✓ Satisfied | `runActorsList` calls `validateKind` **before** `seam.assemble`/`newClient`; rejection returns `UsageError` with no `Execute`. Transport tripwire confirms 0 calls (`TestRunActors_UnsupportedKindIsUsageErrorNoRequest`, `TestActorsCommand_UnsupportedKindNoRequest`) |
| Agent discovery does not require the gated alias | ✓ Satisfied | request `Path` is the literal `"/actors"` with `kind=agent`; no `/agents` string anywhere in `actors.go`. The endpoint is ungated; an org lacking `ai_integration` simply yields no agent rows (no special-casing) |
| Output is structured, not pre-rendered (all four formats from one result) | ✓ Satisfied | the command declares no format flag and produces structured data: human path → `renderFn(ResourceActors, …)`; `json`/`yaml` → `aggregateRawData` over `Page[json.RawMessage]` (no human projection, per-page `meta` dropped — `TestRunActors_StructuredJSONEmitsAggregatedRawPayload`) |
| The filters are carried on every page of the walk | ✓ Satisfied | filters live on the base `Request.Query` that `paging.All` clones across pages; `TestRunActors_FiltersRetainedOnEveryPage` asserts `kind=agent` on every recorded page request of a multi-page walk |

---

## Verdict: Ready

All 5 conformance dimensions pass and all held-out @validation scenarios are satisfied through inspection. The implementation conforms to its specification: the `actors` command reads the unified `GET /actors` with `cobra.NoArgs` and three combinable filters (`--kind` validated locally, `--role-id`/`--query` passed through), walks to completion with explicit incompleteness signalling, renders through the new `actors` key / shared structured path, routes failures through the shared classifier with no new exit code, and creates no `people`/`agents` command and no write surface. No findings.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 4 `@validation` scenarios remain `@wip` in the feature file by design (held-out, no step definitions); leaving them tagged is correct — they document the independent checks performed here rather than the Builder's executable suite.
