# Validate: My Projects

**Feature**: 014-my-projects
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-cli.md, interface-spec.md, features/self-service-reads/my-projects.feature, PROJECT.md
**Implementation files**: 5 — `internal/glassfrog/projects.go` (+ `projects_test.go`), `internal/cli/my_projects.go` (+ `my_projects_test.go`, `my_projects_bdd_test.go`), `internal/cli/app.go` (wiring)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 5 of 5 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 spec driving scenarios covered; all 12 behavioral feature scenarios traced)

Every driving scenario in spec.md § Driving Scenarios has an identifiable code path, and the behavioral feature scenarios pass as executable acceptance (`TestMyProjectsFeatures`, 12/12 green).

| Scenario (spec.md) | Status | Implementation |
|---|---|---|
| list the practitioner's projects | ✓ Covered | `my_projects.go:runMyProjects` (Execute `GET /me/projects`) → `formatMyProjects` |
| filter by a supported status | ✓ Covered | `runMyProjects` step 3 adds `?status=` when set; `validateStatus` reused (013) |
| more results than one page | ✓ Covered | `incompleteProjects` reads `Meta.Pagination.HasNextPage`; signal on stderr, one Execute |
| no usable token | ✓ Covered | `reportClientError` on `*AuthError{NoCredentials}` → UsageError, no projection |
| API responds with a non-2xx | ✓ Covered | `client.Execute` error → `classifyClientError` → `*ResponseError`/APIError (generic) |
| no matching projects | ✓ Covered | `formatMyProjects` empty branch → `no projects`, Success |
| invalid status rejected before any request | ✓ Covered | `validateStatus` fail-fast before assemble/send |

Architecture-informed scenarios (network failure, undecodable body, malformed base URL, malformed credentials, no-role marker) are likewise covered by `runMyProjects`' reused `classifyClientError` branches and the `formatMyProjects` no-role path.

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked; all criteria evidenced)

| Task | Status | Evidence |
|---|---|---|
| T001 — `glassfrog.Project` + envelope | ✓ Met | `projects.go`; `projects_test.go` pins single/multi-page/empty decode, null `role_id`, `has_*` booleans, unknown-field tolerance; leaf package, no new imports |
| T002 — `me projects` command + pure trio | ✓ Met | `my_projects.go`; `my_projects_test.go` covers success/empty/no-role/has-next, status filter + rejection-before-request (tripwire), every client-error branch → Outcome/ExitCode (3/6), no `--include`, token-never-in-output, pure `formatMyProjects` |
| T003 — wiring + godog suite | ✓ Met | one `MustRegister(meCmd, newMyProjectsCommand(...))` in `app.go`; `TestMyProjectsFeatures` (`Paths` → only `my-projects.feature`); 12 behavioral scenarios un-`@wip`'d and passing; `TestAssemble_WiresMeProjectsWithoutPanic` |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md / interface-cli.md) | Status | Implementation |
|---|---|---|
| `Project` shape — `ID/Status/Description/RoleID/Tags/HasSubProjects/HasActions` projected; `IndividualInitiative/ParentProjectID/CreatedAt/UpdatedAt/Link/Note` decoded-not-projected; embeds not modelled | ✓ Conformant | `internal/glassfrog/projects.go` |
| `Pagination` reused (not redefined); `{Data []Project, Meta{Pagination}}` envelope | ✓ Conformant | `projects.go:MyProjectsResponse` references shared `glassfrog.Pagination` |
| `newMyProjectsCommand(seam) *cobra.Command` — `Use:"projects"`, `NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`, local `--status`, reads persistent `--base-url`, wired once | ✓ Conformant | `my_projects.go:newMyProjectsCommand`; `app.go` |
| `runMyProjects(cfg) (Outcome, error)` — validate → assemble → newClient → Execute → format/classify | ✓ Conformant | `my_projects.go:runMyProjects` |
| `formatMyProjects(list) string` — pure, fields/order, empty-result line, signal driven by `HasNextPage` | ✓ Conformant | `my_projects.go:formatMyProjects` |
| seam — assemble + build client; production real resolvers, tests fake transport | ✓ Conformant | reuses `meSeam` / `productionSeam` |
| Error Communication mapping (8 rows: validate→2, success→0, AuthError→2/1, base-URL→2, Decode→1, Response→3, Transport→6) | ✓ Conformant | `runMyProjects` + `classifyClientError` + `ExitCode`; pinned by `TestMyProjectsCommand_ExitCodesAcrossOutcomes` |

**Observation (not a finding)**: The interface artifacts use placeholder symbol names — `myProjectsSeam`, `glassfrog.ProjectList`, and "the `my` parent" — whereas the implementation reuses `meSeam`, names the envelope `glassfrog.MyProjectsResponse`, and attaches the leaf to the runnable `me` command (`me projects`). This is in-bounds: (a) interface-spec.md § Surface explicitly scopes names out — *"Field names and concrete Go types are a build detail; the shapes, signatures, and which fields are projected are the contract"* — and the shapes/signatures all conform; (b) the `my`→`me` command-path divergence is the documented spec-prose drift (`.score/memory/LEARNINGS.md`, 2026-06-07: "013's artifacts say 'my actions'… the implemented convention is `me actions` under the runnable `me` command"; the same entry prescribes `me projects` for 014). spec.md § Assumptions marks the command surface `[ASSUMED]` and states "the behavior … is fixed regardless of the final spelling." No behavioral or contract gap.

---

## Non-Behavior Absence

**Status**: Pass (all 9 non-behaviors absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not walk pagination / fetch beyond first page | ✓ Absent | exactly one `client.Execute`; `incompleteProjects` only reads `HasNextPage` to signal (`tr.calls == 1` pinned) |
| Must not send an unsupported `--status` value | ✓ Absent | reused `validateStatus` rejects before any I/O |
| Must not embed sub-projects/actions nor expose `?include` | ✓ Absent | no `--include` flag (`TestMyProjectsCommand_DeclaresNoIncludeFlag`); no `?include` ever sent (`TestRunMyProjects_NeverSendsInclude`); embed arrays not modelled |
| Must not render raw payload / own output shape | ✓ Absent | `formatMyProjects` reshapes; no payload marshal |
| Must not expose `--output json` | ✓ Absent | no such flag (only a comment documenting the deferral) |
| Must not resolve/read token or base URL, nor attach `X-Auth-Token` | ✓ Absent | `ctx.Cred.Token` never referenced; token rides 007's AuthTransport; only the `--base-url` flag *value* is read and passed to assemble |
| Must not interpret non-2xx / decide exit code or message | ✓ Absent | `classifyClientError` generic; message names status + a generic next step (no "forbidden"/"permission"/"rate limit" — pinned by test); exit-code mapping lives in `ExitCode` (004) |
| Must not read the org-wide project surface | ✓ Absent | path is `/me/projects` only |
| Must not create/update/mutate projects | ✓ Absent | `Method: GET` only |
| Must not prompt interactively | ✓ Absent | no prompts; outcomes are typed |

---

## @wip Lifecycle Completion

**Status**: Pass

`features/self-service-reads/my-projects.feature`: the 12 behavioral scenarios (referenced by checked task T003) have had `@wip` removed and pass under `TestMyProjectsFeatures`. The 5 `@validation`-tagged scenarios correctly retain `@wip` — they are held-out and not referenced by any checked task that should remove them.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 traced to implementation)

These were held out from the Builder. Each traces to a clear code path; several also have direct test pins.

| Scenario | Status | Trace |
|---|---|---|
| The command resolves nothing itself | ✓ Satisfied | `runMyProjects` uses only `seam.assemble`/`seam.newClient` + `Execute`; never reads `ctx.Cred.Token`; the only flag read is `--base-url`'s value (inherited, owned upstream). Token-never-in-output asserted across every branch |
| Exactly one page request is made | ✓ Satisfied | single `Execute`; `TestRunMyProjects_HasNextPageSignalsMoreOnStderr` asserts `tr.calls == 1` even when `HasNextPage` is true |
| An unsupported status costs no request | ✓ Satisfied | `validateStatus` precedes `assemble`; `TestRunMyProjects_UnsupportedStatusRejectedBeforeAnyRequest` asserts `tr.calls == 0` and `assembleCalled/newClientCalled == false` (tripwire) |
| Output is the reshaped projection, not structured JSON | ✓ Satisfied | `formatMyProjects` emits labelled projection lines; no raw/structured JSON path; `TestFormatMyProjects_*` |
| The token never appears in any output | ✓ Satisfied | `runMyProjectsOver` / `runMyProjectsCommand` / the godog `run` helper all fail if `meSecretToken` appears in `stdout+stderr`, across success and every error branch |

> Note: spec.md lists 4 validation scenarios; the feature file carries 5 (it adds the cross-read "token never appears" invariant). All 5 are traced — superset coverage, no gap.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 5 held-out validation scenarios trace to clear code paths. All 3 tasks are checked. The implementation conforms to the specification: it reads `GET /me/projects` through the single transport seam, validates `--status` locally before any request, fetches exactly one page and signals when more exist, renders the shared reshaped projection (with an explicit no-role marker and `has_*` presence signals), reuses 011's error→exit-code mapping unchanged, and exposes no `--include`/`--output json`. The spec-prose `my projects` / `my`-parent naming is stale label only (documented in LEARNINGS, `[ASSUMED]` in spec.md); the shipped `me projects` surface fulfils the fixed behavior.

The optional test run is green: `go test ./internal/glassfrog/ ./internal/cli/` passes, including the 12-scenario `TestMyProjectsFeatures` suite.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 014-my-projects is closed.

A natural follow-up before opening the PR: lift the 5 `@validation` scenarios out of `@wip` is **not** required (they are deliberately held), but the implementer may wish to wire step definitions for them in a future hardening pass. Not a blocker.
