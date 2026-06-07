# Tasks: Identity Read

**Feature**: 011-identity-read
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/self-service-reads/identity-read.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` schema package (1 task, no phase dependencies) [Shared]
Phase 2: Exit-code surface — `Outcome`/`ExitCode` extension + `classifyClientError`, and the persistent `--base-url` root flag (2 tasks; T003 no deps, T002 needs 010's error types — parallel with Phase 1) [Shared]
Phase 3: The `me` command — `newMeCommand` + `runMe` + `formatMe` + `validateInclude` + seam (1 task, depends on T001, T002, T003 **and 010 implemented**) [Shared]
Phase 4: Wiring + executable acceptance via godog (1 task, depends on Phase 3) [Shared]

5 tasks total | T001/T003 startable immediately; T002/T004/T005 build on 010 (landed on main, #30) | Builder: pipeline

> Every task is `[Shared]`: `me` is a single command serving all three user scenarios (confirm-token-and-learn-who/where, identity-plus-roles-in-one-read, tell-bad-token-from-network-failure) rather than decomposing per scenario; the schema, exit-code surface, and flag are shared infrastructure.
>
> **Cross-spec dependency — 010 (Request Execution), landed on main (#30).** T002's `classifyClientError` references 010's typed error types (`TransportError`/`ResponseError`/`DecodeError`), and T004 calls 010's `NewClientFromOS`/`Execute`/`Request` — both now compile against the `internal/apiclient` code on main, so cut the 011 base from current main and the dependency is satisfied. **T001** (`internal/glassfrog`, a new leaf package) and **T003** (the `--base-url` root flag, which uses `apiclient.FlagBaseURL`) have **no** 010 dependency in any case.
>
> **Not purely additive**: T002 and T003 modify existing files — `internal/cli/dispatch.go` (the `Outcome` enum) and `internal/cli/exitcode.go` (the `ExitCode` registry) for T002; the root wiring (`app.go`/`root.go`) and the 003 help-regression tests for T003. This is the forecast extension: 011 is the first consuming command, so it populates the `Outcome` categories + codes 004 reserved and registers the connection flag 008/010 deferred.

---

## Branching Guidance

**Pipeline mode**: `spec/011-identity-read/base` → `spec/011-identity-read/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). The spec base should be cut from a point that includes 010's implementation (see the cross-spec note above).

**Parallel-spec awareness**: 010 (Request Execution) is the active upstream dependency — its `internal/apiclient` seam must be on the base branch before T002/T004/T005 compile. Specs 001–009 are Complete. The sibling reads (012–014) and the Should-tier client capabilities (015–017) are later specs that depend on this command's `glassfrog` models and `classifyClientError`, not concurrent ones.

---

## Phase 1: `internal/glassfrog` schema package [Shared]

- [x] **T001** [Shared] [P] Add the `internal/glassfrog` API-schema package with the `GET /me` decode targets — RED-first unit tests — 4 unit tests (identity, roles embed, no-embed, unknown-field tolerance); leaf package, no internal imports
  - **Scope**: Create a new leaf package `internal/glassfrog` holding plain JSON-tagged structs decoded from API responses: `MeResponse{ Actor; Organization; Membership; Roles []Role }`, `Actor{ ID; Name; Kind }` (`Kind` is `human`|`agent`; `CreatedAt`/`UpdatedAt` decoded but unused), `Organization{ ID; Name }`, `Membership{ AccessLevel }` (plus `ID`/`ActorID`/`OrganizationID` decoded), and a **minimal** `Role{ ID; Name }`. Decoding must tolerate unknown/extra fields (forward-compatible). No transport, no cobra, no exit codes — the package imports nothing internal (so `cli` and `apiclient` can both import it without a cycle). The token is never a field here.
  - **Acceptance criteria**:
    - A `GET /me` JSON fixture decodes into `MeResponse` with `Actor`/`Organization`/`Membership` populated
    - A fixture with `?include=roles` populates `Roles`; a fixture without it leaves `Roles` empty/nil
    - Unknown/extra JSON fields are ignored (decode does not fail)
    - `Role` carries `ID`+`Name` and decodes from an embedded role entry
    - The package has no internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (new leaf package; no 010 dependency)
  - **Plan reference**: Phase 1; ADR-1 (API response models live in `internal/glassfrog`); Data Model Design
  - **Interface references**: interface-spec.md — `internal/glassfrog` (Surface)
  - **Scenario references**: identity-read.feature: "The identity projection prints actor, organization, and access", "An agent token is reported as an agent", "Requested roles are embedded in the projection"
  - **Risk**: ⚠️ Keep `Role` minimal (id+name) — My Roles (012) grows this **same** type to the full role shape; do not fork a second role type, and do not over-model fields the projection doesn't surface.

## Phase 2: Exit-code surface — `Outcome`/`ExitCode` extension + classifier, and the `--base-url` root flag [Shared]

- [x] **T002** [Shared] Extend the `Outcome` enum + `ExitCode` registry with `NetworkUnavailable`(6)/`APIError`(3) and add the shared `classifyClientError` mapping 010's typed errors — RED-first unit tests — classifier table (10 rows) + exhaustiveness guard + AuthError-before-rcfile ordering pin; ExitCode pins extended with a len+comma-ok guard; rcfile base-URL errors also mapped to UsageError per the interface table
  - **Scope**: In `internal/cli/dispatch.go`, add `NetworkUnavailable` and `APIError` to the `Outcome` enum (and its `String()`); in `internal/cli/exitcode.go`, add cases `NetworkUnavailable → codeNetworkUnavailable(6)` and `APIError → codeAPIError(3)` (the constants already exist as reserved values; `ExitCode` stays a pure mapper, `default → codeInternalError(1)` unchanged). Add a shared `classifyClientError(err error) Outcome` (a new small `internal/cli` file, e.g. `clienterror.go`) — the single `errors.As` chain over the API client's typed errors, mapping: `*apiclient.AuthError{NoCredentials}` → `UsageError`; `*apiclient.AuthError{CredentialError}` → `RuntimeError`; the base-URL error type (008's `*BaseURLError` / `rcfile` error) → `UsageError`; `*apiclient.DecodeError` → `RuntimeError`; `*apiclient.ResponseError` → `APIError`; `*apiclient.TransportError` → `NetworkUnavailable`. `*AuthError` must be matched before `*TransportError`. Reused verbatim by 012–017.
  - **Acceptance criteria**:
    - `ExitCode(NetworkUnavailable) == 6` and `ExitCode(APIError) == 3`; existing mappings (0/1/2) unchanged; the exit-code pin test covers the two new values with a `len`+comma-ok-style exhaustiveness guard so a dropped/added mapping fails loud (PR #10 LEARNINGS)
    - `classifyClientError` returns the mapped `Outcome` for each typed error per the table; `*AuthError` is discriminated before `*TransportError`
    - A table test asserts every typed-error → `Outcome` mapping with an exhaustiveness guard
    - `Outcome.String()` renders the two new names; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: 010 implemented (the `apiclient.TransportError`/`ResponseError`/`DecodeError` types must exist to reference and test against; `AuthError` is 007's, already on main)
  - **Plan reference**: Phase 2; ADR-3 (first consuming command adds categories + shared classifier at the single registry), ADR-4 (the error→category mapping)
  - **Interface references**: interface-spec.md — `internal/cli` additions (`Outcome`, `ExitCode`, `classifyClientError`), Error Communication; interface-cli.md — exit-code table
  - **Scenario references**: identity-read.feature: "An unusable token surfaces a non-2xx outcome", "A network failure is surfaced as a transport outcome", "A non-2xx status is surfaced, not classified", "An undecodable response is surfaced as an internal error"
  - **Risk**: ⚠️ `errors.As` order — match `*AuthError` before `*TransportError` so 007's fail-safe is never mislabeled (010's discipline, preserved). ⚠️ Keep `ExitCode` a pure mapper — the classifier produces the category; `ExitCode` never inspects an error. ⚠️ Generic non-2xx → `APIError`(3) is the residual bucket; do **not** interpret 401/403/429 here (that is 015/017, which split 3 later without renumbering).

- [ ] **T003** [Shared] [P] Register `--base-url` as a persistent flag on the root command; update the 003 help-regression tests — RED-first
  - **Scope**: In the root wiring (`internal/cli/root.go`/`app.go`), register `--base-url` as a **persistent** flag on the root command, using `apiclient.FlagBaseURL` for the flag name and a clear usage string (so the precedence rung and the registered flag can't drift). The flag value is read by API commands (T004) and passed to `apiclient.AssembleFromOS(flagValue)`. Update the 003 help-and-version tests that assert root help content to reflect the new global-flags section (the flag is optional documentation data — 003's narrowed non-behavior permits it).
  - **Acceptance criteria**:
    - `--base-url` is a persistent flag on the root, inherited by subcommands; its name is `apiclient.FlagBaseURL`
    - The flag value is retrievable by a subcommand's `RunE` (verified via a small test command or the `me` command in T004)
    - The 003 help-regression tests pass with the new global-flags section; dispatch/exact-match behavior is unaffected (existing dispatch tests still pass)
    - `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (uses `apiclient.FlagBaseURL`, landed via 008; no 010 dependency)
  - **Plan reference**: Phase 2; ADR-2 (persistent root `--base-url` flag)
  - **Interface references**: interface-cli.md — Flags (`--base-url`, persistent on root)
  - **Scenario references**: identity-read.feature: "A malformed base URL is refused before sending" (the flag is the highest-precedence base-URL source)
  - **Risk**: ⚠️ The persistent flag changes root help output — update the 003 tests in the same change rather than leaving them red. ⚠️ Do not gate the flag to API commands only (it is intentionally global); the inert appearance on `version`/`auth` is accepted (ADR-2).

## Phase 3: The `me` command [Shared]

- [ ] **T004** [Shared] Add the `me` command — `newMeCommand(seam)` + pure `runMe`/`formatMe`/`validateInclude` over an injected seam — RED-first unit tests for every branch
  - **Scope**: Add `internal/cli/me.go`. `newMeCommand(seam meSeam) *cobra.Command`: a guard-ready leaf (`Use:"me"`, non-empty `Short`, `Args: cobra.NoArgs`, `SilenceErrors`/`SilenceUsage`) with a local `--include` flag; its `RunE` reads the persistent `--base-url` value (T003), calls the pure `runMe(cfg)`, and maps the returned `Outcome` onto dispatch's error channel (UsageError → `&commandUsageError{}`, RuntimeError → the error as-is, Success → nil) — the `runLogin` pattern. `runMe(cfg meConfig) (Outcome, error)`: `validateInclude` → `seam.assemble(baseURL)` → `seam.newClient(ctx)` (base-URL error → `classifyClientError` → UsageError) → `client.Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me", Query: includeQuery}, &me)` → on success `formatMe(me, includeRoles)` to stdout (Success); on a typed error `classifyClientError(err)` + a token-free stderr message. `formatMe(me glassfrog.MeResponse, includeRoles bool) string`: the reshaped projection (actor name/kind/id, org name/id, access level; a roles section of id+name when `includeRoles` and roles present, omitted when none). `validateInclude(targets []string) error`: reject an unsupported target (spec `include` set = `{roles}` today) as a usage error **before** any assembly/request. `meSeam`: production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS` (real base transport); tests bind a fake base `http.RoundTripper` returning canned responses. `me` never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog me` on a complete context prints the projection (actor id/name/kind, org id/name, access level) and returns Success
    - An agent token renders kind `agent` and surfaces the `agt_` id
    - `me --include roles` adds the `include=roles` query and lists each role's id+name; with no roles, the roles section is omitted
    - An unsupported `--include` target returns a usage error naming the target **with no request issued** (a tripwire fake asserts the transport is never called); `validateInclude` runs before assembly
    - `*ResponseError` (e.g. 401/404) → no projection, non-success (APIError→3); `*TransportError` → non-success (NetworkUnavailable→6), no retry; `*AuthError{NoCredentials}` → UsageError(2) with a "run `glassfrog auth login`" message; `*AuthError{CredentialError}`/`*DecodeError` → RuntimeError(1); base-URL error → UsageError(2)
    - No output or error renders the token; `me` never references `ctx.Cred.Token`; pinned across success + every error branch
    - `formatMe` and `validateInclude` are unit-tested pure; all branches run offline over the fake seam (no real network, no real `~/.glassfrogrc`); `go build`/`go vet` clean
  - **Dependencies**: T001 (decode target), T002 (`classifyClientError` + `Outcome`), T003 (`--base-url` flag), and 010 implemented (`NewClientFromOS`/`Execute`/`Request`)
  - **Plan reference**: Phase 3; ADR-5 (injected seam + pure `runMe`/`formatMe`/`validateInclude`), ADR-4 (outcome mapping); Cross-cutting (secret hygiene, error handling, testing)
  - **Interface references**: interface-cli.md — Command, Flags, Output projection, Error Communication; interface-spec.md — `newMeCommand`/`runMe`/`formatMe`/`validateInclude`/`meSeam`
  - **Scenario references**: identity-read.feature: "The identity projection prints actor, organization, and access", "An agent token is reported as an agent", "Requested roles are embedded in the projection", "An empty roles embed omits the roles section", "An unsupported include target is rejected before any request", "An unusable token surfaces a non-2xx outcome", "A network failure is surfaced as a transport outcome", "A missing token is refused before sending", "An undecodable response is surfaced as an internal error", "A malformed base URL is refused before sending"
  - **Risk**: ⚠️ Validate `--include` **before** assembling/sending (fail-fast, no wasted call) — pin with a tripwire transport. ⚠️ Reuse `classifyClientError` (T002) rather than inlining a second `errors.As` chain. ⚠️ Secret hygiene — never read `ctx.Cred.Token`; render only response-side fields; capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS).

## Phase 4: Wiring + executable acceptance [Shared]

- [ ] **T005** [Shared] Wire `me` into the root in `Assemble()`, and make the driving scenarios pass as executable acceptance via a new `internal/cli` godog suite
  - **Scope**: Add one `MustRegister(root, newMeCommand(productionSeam{}))` line to `Assemble()` (`internal/cli/app.go`) — wiring the production seam, no existing command edited. Add godog step definitions for `features/self-service-reads/identity-read.feature` in a **new** `internal/cli` godog suite (e.g. `TestIdentityReadFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive the command through its seam over a fake base `http.RoundTripper` (and the real assembled tree where clearer) for the behavioral scenarios. Remove `@wip` from the 11 behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held for validate). Step helpers return errors, never panic; reuse existing `internal/cli` step phrasing where an assertion already exists (grep the package's `sc.Step(` registrations first).
  - **Acceptance criteria**:
    - `glassfrog me` is registered under the root through the guard (one `MustRegister` line); the command appears in help and dispatches
    - Every non-`@validation` identity-read scenario (the 8 spec-derived behavioral + 2 architecture-informed + 1 guard-derived `CredentialError`) has an executable, passing path
    - `@wip` removed from those 11 scenarios; the 4 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `identity-read.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T004
  - **Plan reference**: Phase 4; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Command; interface-spec.md — `newMeCommand` wiring
  - **Scenario references**: identity-read.feature: all 11 behavioral Rule-block scenarios (the 4 `@validation` stay held for validate)
  - **Risk**: ⚠️ Suite scoping — a new feature file in the `internal/cli` package must keep every suite pointed at specific files (not the directory), or un-wipping one spec's scenarios breaks another suite; verify each reports its own count. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings; step helpers return errors, never panic (LEARNINGS).
