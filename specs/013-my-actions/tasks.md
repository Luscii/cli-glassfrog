# Tasks: My Actions

**Feature**: 013-my-actions
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/self-service-reads/my-actions.feature

---

## Dependency Graph

Phase 1: `glassfrog.Action` (+ shared `Pagination`/list envelope if not yet present) (1 task, no phase dependencies) [Shared]
Phase 2: `validateStatus` + the spec-sourced status set (1 task, no phase dependencies — parallel with Phase 1) [Shared]
Phase 3: The `my actions` command — `newMyActionsCommand` + `runMyActions` + `formatMyActions` + seam + wiring + godog acceptance (2 tasks; T003 depends on T001/T002 **and 010+011+012 implemented**; T004 depends on T003) [Shared]

4 tasks total | T001/T002 startable immediately (pure/leaf); T003/T004 build on the reused 011 + 012 foundations | Builder: pipeline

> Every task is `[Shared]`: `my actions` is a single command serving all three user scenarios (list-my-actions, filter-by-status, signal-more-results); the resource model, the validator, and the command are shared infrastructure, not per-scenario slices.
>
> **013 is a thin additive read — it reuses two foundations rather than building them.** From **Identity Read (011)**: the `internal/glassfrog` package, the shared `classifyClientError`, the `Outcome`/`ExitCode` categories (codes `3`/`6`), and the persistent `--base-url` root flag — 013 **adds no exit code, no classifier branch, and no flag registration**. From **My Roles (012)**: the `my` parent command, the `glassfrog.Pagination` type, the list envelope, and the "more results available" signal convention. 013 adds only `glassfrog.Action`, the `validateStatus` validator (shared with 014), and the `my actions` leaf with its pure trio.
>
> **Cross-spec dependencies — sequencing matters.** 010 (Request Execution) is **landed on main**. **011 and 012 are shaped/in-progress but NOT yet on main**; T003/T004 reuse their code. Cut the 013 base from a main that carries 011 and 012. If 013 must lead a sibling, it creates the shared types it needs (the `glassfrog.Pagination`/envelope, or the `my` parent) **idempotently in place** and the later sibling reuses them (the 005/006/007 first-to-land-creates pattern) — but the preferred order is 010 → 011 → 012 → 013.

---

## Branching Guidance

**Pipeline mode**: `spec/013-my-actions/base` → `spec/013-my-actions/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). The spec base should be cut from a point that includes 010, 011, and 012's implementations (see the cross-spec note above).

**Parallel-spec awareness**: 011 (Identity Read) and 012 (My Roles) are the active upstream dependencies — their `internal/glassfrog` foundation, `classifyClientError`, `--base-url` flag, `my` parent, `Pagination` type, list envelope, and signal must be on the base branch before T003/T004 compile and pass. 014 (My Projects) is this command's twin: it reuses `validateStatus` and the status set introduced here, so land 013's T002 (or coordinate the shared validator) before 014's command task. Specs 001–010 are Complete/landed.

---

## Phase 1: `glassfrog.Action` schema [Shared]

- [x] **T001** [Shared] [P] Add `Action` to `internal/glassfrog` and decode it through the shared list envelope — RED-first unit tests — 3 decode tests; reused 012's `Pagination`; envelope named `MyActionsResponse` mirroring `MyRolesResponse`
  - **Scope**: Add an `Action` struct to `internal/glassfrog`: `ID` (`actn_…`), `Status` (the status enum), `Description` (nullable), `RoleID` (`role_…`), `Tags []string`, plus `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, and optional `Permissions`/`TriggerEvent`/`Note` decoded but not projected. Decode `GET /me/actions` through the shared list envelope `{ Data []Action; Meta{ Pagination } }` and the `Pagination` struct (`PerPage`, `HasNextPage`, `NextCursor`). **Reuse** `glassfrog.Pagination` and the envelope from My Roles (012) if present; if 012 has not landed, introduce them here as shared `glassfrog` types (not 013-local). Decoding tolerates unknown/extra fields. No transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A single-page `GET /me/actions` fixture decodes into the envelope with `Data` populated and `Meta.Pagination.HasNextPage == false`
    - A multi-page fixture decodes with `HasNextPage == true` and `NextCursor` set
    - An empty-`data` fixture decodes into an empty `Data` slice (not an error)
    - A nullable `description` decodes without failing; unknown/extra fields are ignored
    - The package has no new internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Coordinates with 012 on the shared `Pagination`/envelope types (reuse if present, create-in-place if not).
  - **Plan reference**: Phase 1; ADR-1 (`Action` joins `internal/glassfrog`; reuse 012's `Pagination`/envelope); Data Model Design
  - **Interface references**: interface-spec.md — `internal/glassfrog` (Surface)
  - **Scenario references**: my-actions.feature: "The actions projection lists the practitioner's actions", "A further page is signalled, not fetched", "No matching actions reports an empty list, not a failure"
  - **Risk**: ⚠️ Do **not** define a 013-local `Pagination` or envelope — there is exactly one of each across all list reads (DECISIONS: paginated reads share one envelope). If 012 owns them, reuse; if not yet present, create them as shared `glassfrog` types so 012/014 reuse. ⚠️ Keep `Action` to the spec's fields; project only the surfaced subset.

## Phase 2: `--status` validation [Shared]

- [ ] **T002** [Shared] [P] Add the shared `validateStatus` + spec-sourced status set — RED-first unit tests
  - **Scope**: Add a pure `validateStatus(status string) error` (in `internal/cli`, reusable by 014) that checks a non-empty `--status` value against the spec's status set `{archived, cancelled, completed, current, scheduled, someday, waiting}` (sourced from the `spec/glassfrog-api-v5.yaml` `status` enum). An unsupported value returns a usage error naming the value and listing the supported set; an empty/absent value passes (no constraint). The validator performs no I/O — it runs before any context assembly or request.
  - **Acceptance criteria**:
    - Each of the seven supported statuses passes
    - An unsupported value returns a usage error whose message names the unsupported value and lists the supported set
    - An empty string passes (no filter)
    - The validator is pure (no network, no filesystem); `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (pure)
  - **Plan reference**: Phase 2; ADR-2 (shared `validateStatus`, mirrors 011 `validateInclude`)
  - **Interface references**: interface-spec.md — `validateStatus` (Surface)
  - **Scenario references**: my-actions.feature: "A supported status filters the request", "An unsupported status value is rejected before any request", "An unsupported status costs no request"
  - **Risk**: ⚠️ Source the set from the spec enum and keep it shared (014 reuses it) — do not inline a second copy in 014. ⚠️ Validate before any I/O (the `validateInclude` fail-fast timing); the wasted-call tripwire lives in T003.

## Phase 3: The `my actions` command [Shared]

- [ ] **T003** [Shared] Add the `my actions` command — `newMyActionsCommand(seam)` + pure `runMyActions`/`formatMyActions` over an injected seam — RED-first unit tests for every branch
  - **Scope**: Add `internal/cli/my_actions.go`. `newMyActionsCommand(seam myActionsSeam) *cobra.Command`: a guard-ready leaf (`Use:"actions"`, non-empty `Short`, `Args: cobra.NoArgs`, `SilenceErrors`/`SilenceUsage`) **attached to the `my` parent (012)**, with a local `--status` flag; its `RunE` reads the persistent `--base-url` value (011), calls pure `runMyActions(cfg)`, and maps the returned `Outcome` onto dispatch's error channel (the `runMe` pattern). `runMyActions(cfg) (Outcome, error)`: `validateStatus(statusFlag)` (reject → UsageError, **no request**) → `seam.assemble(baseURL)` → `seam.newClient(ctx)` (base-URL error → `classifyClientError` → UsageError) → `client.Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me/actions", Query: statusQuery}, &list)` → on success `formatMyActions(list)` to stdout (Success); on a typed error `classifyClientError(err)` (**reused from 011, not re-implemented**) + a token-free stderr message. `formatMyActions(list) string`: the reshaped projection (one entry per action: id, status, description, role, tags), an explicit empty-list line, and the "more results available" signal when `list.Meta.Pagination.HasNextPage` (per 012's signal convention). `myActionsSeam`: production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS`; tests bind a fake base `http.RoundTripper` returning canned `GET /me/actions` responses. `my actions` never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog my actions` on a complete context prints the projection (id/status/description/role per action) and returns Success
    - `my actions --status current` validates `current`, adds `?status=current`, and renders the matching page
    - An unsupported `--status` value returns a usage error naming the value and the supported set **with no request issued** (a tripwire fake asserts the transport is never called)
    - A first page reporting `HasNextPage` renders only that page **and** appends the "more results available" signal; exactly one request is made (no second page)
    - An empty `data` response prints the empty-list line and returns Success
    - `*ResponseError` → no projection, non-success (APIError→3); `*TransportError` → non-success (NetworkUnavailable→6), no retry; `*AuthError{NoCredentials}` → UsageError(2) with a "run `glassfrog auth login`" message; `*AuthError{CredentialError}`/`*DecodeError` → RuntimeError(1); base-URL error → UsageError(2)
    - No output or error renders the token; the command never references `ctx.Cred.Token`; pinned across success + every error branch
    - `formatMyActions` is unit-tested pure; all branches run offline over the fake seam (no real network, no real `~/.glassfrogrc`); `go build`/`go vet` clean
  - **Dependencies**: T001 (decode target), T002 (`validateStatus`), and 010 + 011 + 012 implemented (`NewClientFromOS`/`Execute`/`Request`; `classifyClientError`/`Outcome`/`ExitCode`/`--base-url`; the `my` parent + `Pagination`/envelope/signal)
  - **Plan reference**: Phase 3; ADR-4 (injected seam + pure trio), ADR-2 (validate-first), ADR-3 (first page + signal); Cross-cutting (secret hygiene, error handling, testing)
  - **Interface references**: interface-cli.md — Command, Flags, Output projection, Error Communication; interface-spec.md — `newMyActionsCommand`/`runMyActions`/`formatMyActions`/`myActionsSeam`, Error Communication
  - **Scenario references**: my-actions.feature: "The actions projection lists the practitioner's actions", "No matching actions reports an empty list, not a failure", "A missing token is refused before sending", "A non-2xx response is surfaced, not classified", "A network failure is surfaced as a transport outcome", "An undecodable response is surfaced as an internal error", "A malformed base URL is refused before sending", "A supported status filters the request", "An unsupported status value is rejected before any request", "A further page is signalled, not fetched"
  - **Risk**: ⚠️ Reuse 011's `classifyClientError` and 012's signal renderer — do **not** inline a second `errors.As` chain or a second signal format. ⚠️ Validate `--status` before assembling/sending (fail-fast) — pin with a tripwire transport. ⚠️ Secret hygiene — never read `ctx.Cred.Token`; render only response-side fields; capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Fetch one page only — read `HasNextPage` to signal, never to loop (Pagination 016 owns the walk).

- [ ] **T004** [Shared] Wire `my actions` under the `my` parent in `Assemble()`, and make the driving scenarios pass as executable acceptance via a new `internal/cli` godog suite
  - **Scope**: Add one `MustRegister(myParent, newMyActionsCommand(productionSeam{}))` line in `Assemble()` (`internal/cli/app.go`), attaching the `actions` leaf to the `my` parent (012). Add godog step definitions for `features/self-service-reads/my-actions.feature` in a **new** `internal/cli` godog suite (e.g. `TestMyActionsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive the command through its seam over a fake base `http.RoundTripper` for the behavioral scenarios. Remove `@wip` from the behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Step helpers return errors, never panic; reuse existing `internal/cli` step phrasing where an assertion already exists (grep the package's `sc.Step(` registrations first).
  - **Acceptance criteria**:
    - `glassfrog my actions` is registered under the `my` parent through the guard (one `MustRegister` line); the leaf appears in `my`'s help and dispatches
    - Every non-`@validation` my-actions scenario (the spec-derived behavioral + architecture-informed) has an executable, passing path
    - `@wip` removed from those behavioral scenarios; the `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `my-actions.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (and the `my` parent from 012 present on the base)
  - **Plan reference**: Phase 3; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Command; interface-spec.md — `newMyActionsCommand` wiring
  - **Scenario references**: my-actions.feature: all behavioral Rule-block scenarios (the `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — keep every `internal/cli` godog suite pointed at its specific feature file (not the directory), or un-wipping one spec's scenarios breaks another suite; verify each reports its own count. ⚠️ If the `my` parent is not yet on the base (012 not landed), this task must coordinate with 012 on parent ownership rather than registering a duplicate parent. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings; step helpers return errors, never panic (LEARNINGS).
