# Validate: Actor Read

**Feature**: 049-actor-read
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (4 of 4 tasks complete), interface-cli.md, features/actors-disconnected-from-governance/actor-read.feature, PROJECT.md
**Implementation files**: 4 — `internal/glassfrog/actors.go` (ActorDetail/ActorDocument), `internal/render/render.go` + `templates/actor.{full,compact}.tmpl` (ResourceActor/ActorDetailView), `internal/cli/actors.go` (grown command: runActors/runActorRead, validateActorInclude/validateActorsFlags)

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

Every driving scenario in spec.md (and its concretized counterpart in the feature file) traces to an identifiable code path, and the 11 behavioral feature scenarios pass as executable acceptance (`TestActorReadFeatures`).

| Scenario | Status | Implementation |
|---|---|---|
| Read a single actor by id | ✓ Covered | `runActors` (hasID branch) → `runActorRead` issues `GET /actors/{id}`, decodes `ActorDocument`, renders `ResourceActor` (`actors.go`) |
| Read an agent by its `agt_` id | ✓ Covered | `runActorRead` PathEscapes the id verbatim — `per_`/`agt_` both reach `/actors/{id}` |
| Read an actor with governance footprint (`--include roles`) | ✓ Covered | `?include=roles` query built from validated includes; `actor.full` renders each role's name/purpose/accountabilities/domains |
| Read an actor with assignments (`--include assignments`) | ✓ Covered | `?include=assignments`; `actor.full` Assignments section |
| No usable credential | ✓ Covered | `reportFailure` → `classifyClientError` → `UsageError(2)`, not-authenticated message |
| Unknown id → 404 | ✓ Covered | id passed through; `reportFailure` → `APIError(3)` naming the status |
| Unsupported `--include` rejected before any request | ✓ Covered | `validateActorInclude` (pre-assembly), transport tripwire asserts no request |
| Agent read does not require the gated alias | ✓ Covered | path is `/actors/{id}`; no `/agents` reference exists in `internal/cli` |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 tasks complete; all criteria evidenced)

| Task | Status | Evidence |
|---|---|---|
| T001 — `ActorDetail` decode type | ✓ Met | `ActorDetail{Actor; Roles; Assignments}` + `ActorDocument` alias; decode tests cover bare / roles / assignments; `Actor`/`Role`/`Assignment` unchanged; package imports neither `cli` nor `apiclient` |
| T002 — singular `actor` render key | ✓ Met | `ResourceActor` in `builtinResources` (registry-exhaustiveness guard passes); `actor.full`/`actor.compact` with explicit-absence markers; 9 golden tests; `ResourceMe`/`ResourceActors` untouched |
| T003 — grow `actors` to `MaximumNArgs(1)` | ✓ Met | one `Execute` per `GET /actors/{id}`; `agt_` reaches ungated endpoint; `--include` variants send `roles`/`assignments`/`roles,assignments`; unsupported `--include` and mode-separation misuse are `UsageError(2)` with no request; no-id still lists (048 preserved); `json`/`yaml` emit the single `{data:…}` raw document; 404→APIError(3); no new `Outcome`/`ExitCode`/root flag |
| T004 — executable acceptance | ✓ Met | `TestActorReadFeatures` Paths names only `actor-read.feature`; 11 behavioral scenarios pass; 4 `@validation` scenarios held `@wip`; fake transport / no real network |

---

## Interface Contract Conformance

**Status**: Pass (single-read surface conformant)

| Surface element | Status | Evidence |
|---|---|---|
| `glassfrog actors <id>` — `MaximumNArgs(1)`, branch on `len(args)` | ✓ Conformant | `newActorsCommand` Args + `runActors` dispatch |
| `--include LIST` (comma-separated, validated `{roles, assignments}`) | ✓ Conformant | `StringSliceVar`; `validateActorInclude` reject-unknown; sent as one `include=` parameter |
| Mode separation (list filter + id → usage 2; `--include` no id → usage 2; second positional → usage 2) | ✓ Conformant | `validateActorsFlags` + cobra `MaximumNArgs(1)`, all pre-assembly with transport tripwire |
| Output: `json`/`yaml` → single `{data:…}` raw bytes; `full`/`compact` → human projection | ✓ Conformant | `runActorRead` machine-format branch uses `output.RenderSuccess` on raw bytes (not `aggregateRawData`); human branch renders `ResourceActor` |
| Error communication (404→3, 401/403→4, 429→5, transport→6, no-token→2, bad `--output`→2, unsupported `--include`→2) | ✓ Conformant | shared `reportFailure`/`classifyClientError`; no new `Outcome`/`ExitCode` |
| Validation order: `--output` → mode-separation → `--include` | ✓ Conformant | `runActors` resolves render target first, then guards, then enum (`TestRunActorRead_OutputResolvedBeforeInclude`) |

*Note (not a finding):* the interface § Output illustrates an assignment row as `- <role name> (<role_…>)  <focus | —>`. The landed `glassfrog.Assignment` carries no role *name* on an actor's `?include=assignments` embed (only `role_id`), so `actor.full` renders `- <role_id>  <focus | —>` — the available facts, with the focus absence guard. This is the interface's explicitly build-time layout detail; rendering a name the response does not carry would fabricate a value (CONSTITUTION VIII). The contracted *behavior* (assignments embedded inline, present only when requested, focus with explicit-absence marker) is conformant.

---

## Non-Behavior Absence

**Status**: Pass (no excluded capability present)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No list/search on the single read | ✓ Absent | the 1-arg branch issues one `GET /actors/{id}`; the 0-arg list is 048, preserved verbatim |
| No standalone assignments listing | ✓ Absent | only `--include assignments` inline embed; no separate assignments command/endpoint added (050's surface untouched) |
| No `agt_` read via the gated `/agents/{id}` alias | ✓ Absent | grep confirms no `/agents` routing in `internal/cli`; path is `/actors/{id}` |
| No pagination | ✓ Absent | `runActorRead` makes exactly one `Execute`; no `paging.All`, no `Page[T]`, no cursor |
| No create/update/delete | ✓ Absent | `runActorRead` issues only `http.MethodGet`; no PATCH/DELETE |
| No raw-JSON fixed default / own format flag | ✓ Absent | renders through Output Format Selection (`rt.format`); only `--include` is added, `-o` is inherited |
| No re-resolving base URL/token/auth/error-typing/exit-codes | ✓ Absent | reuses `assemble`/`newClient`/`classifyClientError`/frozen `Outcome`/`ExitCode`; never reads `ctx.Cred.Token` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral scenarios referenced by the checked tasks have had `@wip` removed and pass under `TestActorReadFeatures`. The 4 `@validation @wip` scenarios correctly retain `@wip` — they are held-out for this skill and are not referenced for un-wip by any task. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation)

These held-out scenarios were traced independently against the code (they remain `@wip`, so no godog step exercises them; each also has a supporting unit test noted below).

| Scenario | Status | Trace |
|---|---|---|
| Agent drill-in does not route through the gated alias | ✓ Satisfied | `runActorRead` targets `/actors/{PathEscape(id)}`; no `/agents` reference exists in `internal/cli`. Supported by `TestRunActorRead_AgentReadsUngatedUnifiedEndpoint` (asserts path has no `/agents`). |
| Output is structured, not pre-rendered | ✓ Satisfied | the command produces an `ActorDetailView` (human) or the raw `{data:…}` bytes (machine) through Output Format Selection; it declares no format flag of its own (only `--include`), so all four formats render from one result. Supported by `TestRunActorRead_StructuredEmitsSingleRawDocument`. |
| The single read issues no page walk | ✓ Satisfied | exactly one `exec.Execute` per format branch; no `paging.All`/`Page[T]`/cursor in `runActorRead`. `TestRunActorRead_StructuredEmitsSingleRawDocument` asserts `tr.calls == 1` and emits a `{data: object}` (not the aggregated `{data:[…]}` list). |
| A non-2xx status is surfaced, not classified | ✓ Satisfied | both failure branches call `reportFailure` → shared `classifyClientError` (015); the command adds no bespoke status message. Supported by `TestRunActorRead_UnknownIdSurfacesAPIStatus` (404 → APIError/3, status named). |

---

## Verdict: Ready

All 4 tasks are complete. All 5 conformance dimensions pass with zero findings, and all 4 held-out validation scenarios trace to clear code paths. The implementation conforms to the specification: the single-actor drill-in reads `GET /actors/{id}` with exactly one request, embeds the `roles`/`assignments` footprint on `--include`, validates the closed include set and mode separation fail-fast before any request, reaches agents through the ungated unified endpoint, renders through the shared output seam, and routes failures through the shared classifier — adding no new error category, exit code, pagination, or write capability. The full suite (`go build`, `go vet`, `go test ./...`) is green.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 049-actor-read is closed.
