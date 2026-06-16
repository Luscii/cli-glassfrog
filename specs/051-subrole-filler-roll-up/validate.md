# Validate: Subrole Filler Roll-up

**Feature**: 051-subrole-filler-roll-up
**Round**: 1 of 3
**Date**: 2026-06-14
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/who-to-contact-for-a-role/subrole-filler-roll-up.feature, PROJECT.md
**Implementation files**: `internal/cli/subrole_actors.go` (new leaf + runner), `internal/cli/actors.go` (parameterized list-walk reused), `internal/cli/app.go` (wiring); tests `internal/cli/subrole_actors_test.go` (19 unit), `internal/cli/subrole_filler_roll_up_bdd_test.go` (godog suite)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 6 of 6 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 spec driving scenarios covered; the feature's 9 behavioral scenarios all execute and pass)

Each spec driving scenario traces to an identifiable code path in `runSubroleActors` (`internal/cli/subrole_actors.go`) and is executed by the passing godog suite `TestSubroleFillerRollUpFeatures`.

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| Roll up the actors filling a circle's direct sub-roles | ✓ Covered | `runSubroleActors` → `req` path `/roles/{id}/subroles/actors` → `runActorsListWalk`; BDD "A circle's direct sub-roles' fillers are rolled up" |
| Narrow the roll-up to agents | ✓ Covered | `subroleActorsQuery` sets `kind` when Changed+non-empty; BDD "A kind filter narrows the roll-up to agents" |
| Roll-up walks every page to completion | ✓ Covered | default `paging.All` walk via `runActorsListWalk`; BDD "The roll-up walks every page to completion" |
| Anchor is a leaf role (404) | ✓ Covered | leaf-404 → `reportFailure`/`classifyClientError`, no special-case; BDD "A leaf anchor fails with the API status" |
| No usable credential | ✓ Covered | `*AuthError{NoCredentials}` → `reportFailure` → UsageError(2); BDD "A missing token fails as a not-authenticated usage error" |
| Sub-roles exist but carry no fillers | ✓ Covered | empty `200` → `writeHuman` renders `no actors`, exit 0; BDD "Sub-roles with no fillers are a clean success" |
| Unsupported kind value rejected before any request | ✓ Covered | `validateKind` pre-assembly → UsageError(2), no request; BDD "An unsupported kind is rejected as a usage error" |
| Paginated roll-up with first-page opt-out | ✓ Covered | `cfg.firstPage` → `runActorsFirstPage` + `moreActorsNote`; BDD "The first-page opt-out stops at one page and signals more" |

The feature file also concretizes a plan-derived mid-walk-failure scenario ("A mid-walk failure yields a partial set flagged incomplete"), covered by `reportIncompleteActorsWalk` and passing in the suite.

---

## Acceptance Criteria

**Status**: Pass (T001 and T002 both checked; all criteria evidenced)

| Task | Status | Evidence |
|---|---|---|
| T001 — `subrole-actors <role-id>` leaf | ✓ Met | `subrole_actors.go`: `ExactArgs(1)`, `--kind` only (no `--role-id`/`--query`), path swap to `/roles/{id}/subroles/actors`, leaf-404 surfaced verbatim, reuses `glassfrog.Actor` + `render.ResourceActors` + `validateKind`, no new `Outcome`/`ExitCode`/render key; 19 unit tests cover walk/empty/kind/first-page/per-page/mid-walk/no-cred/transport/404/non-2xx/output-first/structured-JSON/escaping/surface |
| T002 — executable acceptance | ✓ Met | `TestSubroleFillerRollUpFeatures` Paths names only this feature file; 9 behavioral scenarios pass (51 steps), 6 `@validation` kept `@wip`; leaf-404 (`cannedTransport` 404) and empty-200 (`cannedTransport` empty body) use distinct fakes with distinct outcomes; offline (fake transport), `go build`/`go vet` clean |

ADR-3's single-sourcing requirement is met by extracting `actorsWalkConfig` and adding a `req` parameter to `runActorsListWalk`/`runActorsFirstPage`/`actorsWalkOptions` — both the `actors` directory and `subrole-actors` drive one page loop / render branch / `errors.As` chain (no copied runner). Note: the artifacts name a `runActorsList` symbol that no longer exists (049 landed and grew it to `runActors`/`runActorsListWalk`); this is recorded as a benign drift in LEARNINGS and is **not** a conformance gap — the reuse target is the post-049 walk.

---

## Interface Contract Conformance

**Status**: Pass (the single `subrole-actors` surface conforms)

| Surface element (interface-cli.md) | Status | Evidence |
|---|---|---|
| `subrole-actors <role-id>` — `ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`, guard-registered + wired | ✓ Conformant | `newSubroleActorsCommand`; `app.go` `MustRegister`; `TestNewSubroleActorsCommand_SurfaceOnlyKind` |
| Flags: `--kind`, `--first-page`, `--per-page`; inherits `--base-url`/`-o`; NO `--role-id`/`--query` | ✓ Conformant | flag declarations; surface test asserts presence of the three and absence of `role-id`/`query` |
| Validation order: `--output` resolved first, then `--kind` (both pre-assembly, no request on failure) | ✓ Conformant | `runSubroleActors` step 1 `resolveRenderTarget`, step 2 `validateKind`; `TestRunSubroleActors_OutputResolvedBeforeKind`, `…_BadOutputIsUsageErrorNoRequest` |
| Output: `json`/`yaml` aggregated `{data:[…]}` via `aggregateRawData`; `full`/`compact` human projection; no raw envelope under human | ✓ Conformant | `runActorsListWalk` machine vs human branches; `…_StructuredJSONEmitsAggregatedRawPayload` |
| Empty list → `no actors`, exit 0, distinct from leaf-404 | ✓ Conformant | `…_EmptyIsCleanSuccess` vs `…_LeafAnchor404SurfacesStatus` |
| Error table: leaf-404 surfaced through shared classifier; no new `Outcome`/`ExitCode` | ✓ Conformant | `reportFailure`/`classifyClientError`; `…_Non2xxClassifies` (403/4, 429/5, 500/3), `…_NoCredentialsIsUsageError`, `…_TransportErrorIsNetworkUnavailable` |
| One-level read, anchor id `url.PathEscape`-d and passed through unvalidated | ✓ Conformant | `Path: "/roles/" + url.PathEscape(cfg.roleID) + "/subroles/actors"`; `…_RoleIDEscapedAsOneSegment` |

---

## Non-Behavior Absence

**Status**: Pass (all 9 non-behaviors absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No roll-up beyond direct sub-roles (no transitive recursion) | ✓ Absent | exactly one paginated read; no grand-child request constructed (`@validation` "reads only the direct sub-roles") |
| Does not list the anchor role's own fillers | ✓ Absent | reads `/roles/{id}/subroles/actors`, not `/actors?role_id=` or `/roles/{id}/assignments` |
| No assignment relationship (`focus`/`elected_until`) | ✓ Absent | renders `glassfrog.Actor` (id/name/kind only); grep finds `focus`/`elected_until` only in comments asserting their absence |
| No separate `people`/`agents` subroles command | ✓ Absent | only `subrole-actors` registered; `--kind human` covers the alias |
| No leaf-404 special-casing into "no sub-roles" / empty success | ✓ Absent | 404 surfaced verbatim; BDD "no \"this role has no sub-roles\" message will be added" passes |
| No actor/assignment writes | ✓ Absent | bodyless `GET` only |
| No raw-JSON default, no own format flag | ✓ Absent | reuses `-o`/`--output` (020); human default via `resolveRenderTarget` |
| Does not resolve base URL/token, attach header, type non-2xx, or pick exit codes | ✓ Absent | delegates to `assemble`/`newClient`/`classifyClientError`; comment + code confirm `ctx.Cred.Token` never read |
| No interpretation/advice on rolled-up actors | ✓ Absent | produces actor data only |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 behavioral scenarios referenced by checked task T002 have had `@wip` removed and execute. The 6 `@validation @wip` scenarios remain held — T002 explicitly keeps them for validate, so their `@wip` is correct, not a leftover. No behavioral scenario referenced by a checked task still carries `@wip`.

---

## Validation Scenario Results

**Status**: Satisfied (6 of 6 scenarios traced to implementation)

These were held from the Builder. Each traces to a clear code path; 5 of the 6 are additionally exercised by passing unit tests (the suite's `@validation` Gherkin steps remain unbound, so verification is inspection-based per the skill's baseline, with unit tests as supplementary evidence).

| Scenario | Status | Trace |
|---|---|---|
| The roll-up reads only the direct sub-roles | ✓ Satisfied | `runSubroleActors` issues exactly one paginated read of `/roles/{id}/subroles/actors`; no recursive/child request exists. `TestRunSubroleActors_KindRetainedOnEveryPage` asserts every page path is the subroles endpoint |
| A leaf-role 404 is a failure, not an empty success | ✓ Satisfied | 404 → `reportFailure`→`classifyClientError` (APIError/3, non-zero); empty-200 → `no actors`/exit 0. `…_LeafAnchor404SurfacesStatus` vs `…_EmptyIsCleanSuccess` prove distinct outcomes |
| An unsupported kind costs no request | ✓ Satisfied | `validateKind` runs before `assemble`/`newClient`; `…_UnsupportedKindIsUsageErrorNoRequest` asserts `tr.calls == 0` |
| The result is actor-shaped, not assignment-shaped | ✓ Satisfied | `render.ActorsView{Data: []glassfrog.Actor}` (id/name/kind); `…_RollsUpAndProjectsAtSubrolesPath` asserts id/name/kind present and `focus`/`elected_until` absent |
| Output is structured, not pre-rendered | ✓ Satisfied | command supplies structured data to `writeHuman`/`aggregateRawData` and declares no format flag; all four formats render from one walk result. Human branch renders the `actors` template (no `data`/`meta` envelope); `…_StructuredJSONEmitsAggregatedRawPayload` confirms the json envelope path |
| The kind filter is carried on every page of the walk | ✓ Satisfied | `subroleActorsQuery` builds `kind` into the base `req`; `paging.All` preserves it across pages. `…_KindRetainedOnEveryPage` asserts `kind=human` on every recorded page query |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 6 held-out validation scenarios are satisfied through inspection (5 additionally backed by passing unit tests). Both tasks are checked; `go build ./...`, `go vet ./...`, and the feature + unit suites run clean. The implementation conforms to the specification — including the load-bearing distinctions the spec emphasizes: leaf-404-as-failure vs empty-200-as-success, actor-shape vs assignment-shape, one-level-only, and reject-kind-before-request.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The `@validation` scenarios remain `@wip` by design (held-out verification); they may be un-`@wip`-ped and given step bindings in a follow-up if executable regression coverage of the validation set is wanted, but that is optional — it is not a conformance gap.
