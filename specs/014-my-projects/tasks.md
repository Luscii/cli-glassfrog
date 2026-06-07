# Tasks: My Projects

**Feature**: 014-my-projects
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/self-service-reads/my-projects.feature

---

## Dependency Graph

Phase 1: `glassfrog.Project` (+ reuse shared `Pagination`/list envelope) (1 task, no phase dependencies) [Shared]
Phase 2: The `my projects` command — `newMyProjectsCommand` + `runMyProjects` + `formatMyProjects` + seam + wiring + godog acceptance (2 tasks; T002 depends on T001 **and 010+011+012+013 implemented**; T003 depends on T002) [Shared]

3 tasks total | T001 startable immediately (leaf schema); T002/T003 build on the reused 011 + 012 + 013 foundations | Builder: pipeline

> Every task is `[Shared]`: `my projects` is a single command serving all three user scenarios (list-my-projects, filter-by-status, signal-more-results).
>
> **014 is the twin of My Actions (013) and is almost entirely reuse.** It adds only `glassfrog.Project`, the `my projects` leaf, and the pure `runMyProjects`/`formatMyProjects`. It reuses: from Identity Read (011) the `internal/glassfrog` package, `classifyClientError`, the `Outcome`/`ExitCode` categories (codes `3`/`6`), and the persistent `--base-url` flag; from My Roles (012) the `my` parent, `glassfrog.Pagination`, the list envelope, and the "more results available" signal; from My Actions (013) the shared `validateStatus` + status set and the list-projection/seam shape. It introduces **no validator, no exit code, no flag registration** — so it has no Phase 2 "shared validator" task (unlike 013).
>
> **Cross-spec dependencies — sequencing matters.** 010 (Request Execution) is **landed on main**. **011, 012, and 013 are shaped/in-progress but NOT yet on main**; T002/T003 reuse their code (`classifyClientError`/`--base-url`/`Outcome` from 011; `my` parent/`Pagination`/envelope/signal from 012; `validateStatus` from 013). Cut the 014 base from a main that carries 011, 012, and 013. If 014 must lead 013, it owns the shared `validateStatus` (first status-filtered read to land creates it) and 013 reuses — coordinate so only one copy exists. Preferred order: 010 → 011 → 012 → 013 → 014.

---

## Branching Guidance

**Pipeline mode**: `spec/014-my-projects/base` → `spec/014-my-projects/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). The spec base should be cut from a point that includes 010, 011, 012, and 013's implementations (see the cross-spec note above).

**Parallel-spec awareness**: 011, 012, and 013 are the active upstream dependencies — their foundations (the `glassfrog` package, `classifyClientError`, `--base-url`, the `my` parent, `Pagination`/envelope/signal, and `validateStatus`) must be on the base branch before T002/T003 compile and pass. 013 (My Actions) is this command's twin; land 013's `validateStatus` (or coordinate the shared validator) before this command's task. Specs 001–010 are Complete/landed.

---

## Phase 1: `glassfrog.Project` schema [Shared]

- [x] **T001** [Shared] [P] Add `Project` to `internal/glassfrog` and decode it through the shared list envelope — RED-first unit tests — 4 scenarios referenced; `internal/glassfrog/projects.go` + `projects_test.go`, reused shared `Pagination`/envelope
  - **Scope**: Add a `Project` struct to `internal/glassfrog`: `ID` (`proj_…`), `Status` (the status enum), `Description`, `RoleID` (`role_…`, **nullable** — null for non-role-owned projects), `Tags []string`, `HasSubProjects`, `HasActions`, plus `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, `Link` (nullable), `Note` (nullable) decoded but not projected. Decode `GET /me/projects` through the **shared** list envelope `{ Data []Project; Meta{ Pagination } }` and the shared `Pagination` struct (reuse 012/013's types — do not redefine). The `sub_projects`/`actions` embed arrays are **not** modelled (no `?include` on this operation — ADR-2). Decoding tolerates unknown/extra fields. No transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A single-page `GET /me/projects` fixture decodes into the envelope with `Data` populated and `Meta.Pagination.HasNextPage == false`
    - A multi-page fixture decodes with `HasNextPage == true` and `NextCursor` set
    - An empty-`data` fixture decodes into an empty `Data` slice (not an error)
    - A null `role_id` decodes without failing (the nullable owning role); `has_sub_projects`/`has_actions` decode as booleans
    - Unknown/extra fields are ignored; the package has no new internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Reuses 012/013's shared `Pagination`/envelope.
  - **Plan reference**: Phase 1; ADR-1 (`Project` joins `internal/glassfrog`; reuse `Pagination`/envelope), ADR-2 (no `?include`); Data Model Design
  - **Interface references**: interface-spec.md — `internal/glassfrog` (Surface)
  - **Scenario references**: my-projects.feature: "The projects projection lists the practitioner's projects", "A further page is signalled, not fetched", "No matching projects reports an empty list, not a failure", "A project with no owning role renders an explicit no-role marker"
  - **Risk**: ⚠️ Reuse the shared `Pagination`/envelope — do **not** define a 014-local copy (DECISIONS: one envelope across all list reads). ⚠️ Do **not** model `sub_projects`/`actions` embeds — the operation offers no `include` (ADR-2); project `has_*` booleans instead. ⚠️ `role_id` is nullable — handle the no-role case in decode and projection.

## Phase 2: The `my projects` command [Shared]

- [x] **T002** [Shared] Add the `my projects` command — `newMyProjectsCommand(seam)` + pure `runMyProjects`/`formatMyProjects` over an injected seam — RED-first unit tests for every branch — implemented as `me projects` under `me` (drift: artifacts say `my`, codebase convention is `me`, mirroring `me roles`/`me actions` — see LEARNINGS); reuses `meSeam`/`validateStatus`/`classifyClientError`/012 signal; `internal/cli/my_projects.go` + `my_projects_test.go`
  - **Scope**: Add `internal/cli/my_projects.go`. `newMyProjectsCommand(seam myProjectsSeam) *cobra.Command`: a guard-ready leaf (`Use:"projects"`, non-empty `Short`, `Args: cobra.NoArgs`, `SilenceErrors`/`SilenceUsage`) **attached to the `my` parent (012)**, with a local `--status` flag (**no `--include`** — ADR-2); its `RunE` reads the persistent `--base-url` value (011), calls pure `runMyProjects(cfg)`, and maps the returned `Outcome` onto dispatch's error channel (the `runMyActions` pattern). `runMyProjects(cfg) (Outcome, error)`: **reused** `validateStatus(statusFlag)` (reject → UsageError, **no request**) → `seam.assemble(baseURL)` → `seam.newClient(ctx)` (base-URL error → `classifyClientError` → UsageError) → `client.Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me/projects", Query: statusQuery}, &list)` → on success `formatMyProjects(list)` to stdout (Success); on a typed error `classifyClientError(err)` (**reused from 011**) + a token-free stderr message. `formatMyProjects(list) string`: the reshaped projection (one entry per project: id, status, description, role-or-no-role-marker, has-sub-projects/has-actions, tags), an explicit empty-list line, and the "more results available" signal when `list.Meta.Pagination.HasNextPage` (per 012's convention). `myProjectsSeam`: production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS`; tests bind a fake base `http.RoundTripper` returning canned `GET /me/projects` responses. `my projects` never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog my projects` on a complete context prints the projection (id/status/description/role/has-children per project) and returns Success
    - `my projects --status current` validates `current` (reused `validateStatus`), adds `?status=current`, and renders the matching page
    - An unsupported `--status` value returns a usage error naming the value and the supported set **with no request issued** (a tripwire fake asserts the transport is never called)
    - A first page reporting `HasNextPage` renders only that page **and** appends the "more results available" signal; exactly one request is made (no second page)
    - An empty `data` response prints the empty-list line and returns Success
    - A project with a null `role_id` renders an explicit no-role marker (still surfacing id/status/description)
    - `*ResponseError` → no projection, non-success (APIError→3); `*TransportError` → non-success (NetworkUnavailable→6), no retry; `*AuthError{NoCredentials}` → UsageError(2); `*AuthError{CredentialError}`/`*DecodeError` → RuntimeError(1); base-URL error → UsageError(2)
    - No `--include` flag exists; no `?include` is ever added to the request
    - No output or error renders the token; the command never references `ctx.Cred.Token`; pinned across success + every error branch
    - `formatMyProjects` is unit-tested pure; all branches run offline over the fake seam; `go build`/`go vet` clean
  - **Dependencies**: T001 (decode target), and 010 + 011 + 012 + 013 implemented (`NewClientFromOS`/`Execute`/`Request`; `classifyClientError`/`Outcome`/`ExitCode`/`--base-url`; the `my` parent + `Pagination`/envelope/signal; `validateStatus`)
  - **Plan reference**: Phase 2; ADR-3 (injected seam + pure trio), ADR-1 (reuse `validateStatus`), ADR-2 (no `?include`); Cross-cutting (secret hygiene, error handling, testing)
  - **Interface references**: interface-cli.md — Command, Flags, Output projection, Error Communication; interface-spec.md — `newMyProjectsCommand`/`runMyProjects`/`formatMyProjects`/`myProjectsSeam`, Error Communication
  - **Scenario references**: my-projects.feature: "The projects projection lists the practitioner's projects", "No matching projects reports an empty list, not a failure", "A project with no owning role renders an explicit no-role marker", "A missing token is refused before sending", "A non-2xx response is surfaced, not classified", "A network failure is surfaced as a transport outcome", "An undecodable response is surfaced as an internal error", "A malformed base URL is refused before sending", "A supported status filters the request", "An unsupported status value is rejected before any request", "A further page is signalled, not fetched"
  - **Risk**: ⚠️ Reuse 013's `validateStatus`, 011's `classifyClientError`, and 012's signal renderer — do **not** inline a second validator, `errors.As` chain, or signal format. ⚠️ No `--include` flag — `/me/projects` offers no `include` (ADR-2). ⚠️ Secret hygiene — never read `ctx.Cred.Token`; render only response-side fields; capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Fetch one page only — read `HasNextPage` to signal, never to loop.

- [ ] **T003** [Shared] Wire `my projects` under the `my` parent in `Assemble()`, and make the driving scenarios pass as executable acceptance via a new `internal/cli` godog suite
  - **Scope**: Add one `MustRegister(myParent, newMyProjectsCommand(productionSeam{}))` line in `Assemble()` (`internal/cli/app.go`), attaching the `projects` leaf to the `my` parent (012). Add godog step definitions for `features/self-service-reads/my-projects.feature` in a **new** `internal/cli` godog suite (e.g. `TestMyProjectsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive the command through its seam over a fake base `http.RoundTripper` for the behavioral scenarios. Remove `@wip` from the behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Step helpers return errors, never panic; reuse existing `internal/cli` step phrasing where an assertion already exists (grep the package's `sc.Step(` registrations first — many phrasings are shared with My Actions/Identity Read).
  - **Acceptance criteria**:
    - `glassfrog my projects` is registered under the `my` parent through the guard (one `MustRegister` line); the leaf appears in `my`'s help and dispatches
    - Every non-`@validation` my-projects scenario (the spec-derived behavioral + architecture-informed) has an executable, passing path
    - `@wip` removed from those behavioral scenarios; the `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `my-projects.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002 (and the `my` parent from 012 present on the base)
  - **Plan reference**: Phase 2; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Command; interface-spec.md — `newMyProjectsCommand` wiring
  - **Scenario references**: my-projects.feature: all behavioral Rule-block scenarios (the `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — keep every `internal/cli` godog suite pointed at its specific feature file (not the directory); verify each reports its own count. ⚠️ If the `my` parent is not yet on the base (012 not landed), coordinate with 012 on parent ownership rather than registering a duplicate parent. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and reuse the shared My Actions/Identity Read phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS).
