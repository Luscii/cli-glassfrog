# Tasks: My Roles

**Feature**: 012-my-roles
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/self-service-reads/my-roles.feature

---

## Dependency Graph

**Prerequisites (external)**:
- **010 Request Execution — implemented ✓** on `main` (`apiclient.NewClientFromOS`, `Execute`, `Request`, typed errors).
- **011 Identity Read scaffolding — implemented on `main` ✓**: the runnable `me` command, the persistent root `--base-url`, the shared `classifyClientError(err) Outcome` helper, and the `internal/glassfrog` schema package (`Role{ID,Name}`) all exist (`internal/cli/me.go`, `internal/cli/clienterror.go`, `internal/glassfrog/`; 011 Complete/validate). T002/T003 build on them directly — no creation-vs-reuse contingency.

```
Phase 1: Schema growth      (1 task, depends on internal/glassfrog existing) [US2, US3]
Phase 2: me roles command   (1 task, depends on Phase 1 + 010 + 011 scaffolding) [US1]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | T001 startable as soon as internal/glassfrog exists | Builder: glassfrog-cli (after/with the 011 session)
```

## Branching Guidance

**Pipeline mode**: `spec/012-my-roles/base` → `spec/012-my-roles/task-1`, `spec/012-my-roles/task-2`, `spec/012-my-roles/task-3`

**Role-based mode**: `spec/012-my-roles/base` is the integration point. **Parallel-spec awareness**: 012 **conforms to** Identity Read (011) — it reuses 011's `me` command, root `--base-url`, `classifyClientError`, and `internal/glassfrog`. Sequence 012's command/acceptance work after 011's scaffolding lands; the shared `Role` type and the classifier are extension points, not 012-owned. No registry edit and no new `Outcome` category in 012.

---

## Phase 1: Schema growth [US2, US3]

- [ ] **T001** [US2] Grow the shared `internal/glassfrog.Role` and add the `/me/roles` response type + pure renderer
  - **Scope**: In `internal/glassfrog`, grow `Role{ID,Name}` (011) with `Purpose string`, `Accountabilities []struct{ Description string }`, `Domains []struct{ Description string }`. Define a **reusable named `Pagination struct{ PerPage int; HasNextPage bool; NextCursor string }`** (the shared list-pagination model 013/014 reuse — created here, per the 013-my-actions DECISIONS contract, matching the API's `Pagination` schema) and a `MyRolesResponse{ Data []Role; Meta struct{ Pagination Pagination } }` that **references** the named type (not an anonymous struct). **Every decoded field carries an explicit JSON tag matching the API's snake_case name** — `json:"data"`, `json:"meta"`, `json:"pagination"`, `json:"per_page"`, `json:"has_next_page"`, `json:"next_cursor"`, `json:"purpose"`, `json:"accountabilities"`, `json:"domains"`, `json:"description"` — because Go's `encoding/json` is case-insensitive but does **not** bridge underscores, so an untagged `HasNextPage` would never bind to `has_next_page` and `incomplete()` would silently stay false. `NextCursor` is present when `has_next_page` is true (the `?cursor=` value Pagination (016) will follow; 012 decodes but does not yet use it). Decoding is tolerant of unknown/extra fields (one shared type; never a 012-local duplicate). Add a pure `formatMyRoles(MyRolesResponse) string` (name + `role_…` id; `Purpose:` / `(no purpose set)`; `Domains:` then `Accountabilities:`, headers always present, `(none)` when empty; `No roles.` for an empty list) and a pure `incomplete(meta) bool` from `HasNextPage`. No HTTP, no cobra.
  - **Acceptance criteria**:
    - The grown `Role` decodes `me --include roles` (id+name only) AND `me roles` (full shape) from one type; extra fields are ignored
    - `formatMyRoles` renders Domains before Accountabilities, always emits both headers (`(none)` when empty), `(no purpose set)` for null purpose, `No roles.` for an empty list, and never renders fillers/tags/flags
    - `incomplete` is true iff `Meta.Pagination.HasNextPage`
    - A decode test feeds a real `{"data":[…],"meta":{"pagination":{"per_page":N,"has_next_page":true,"next_cursor":"abc"}}}` payload and asserts `HasNextPage` decoded `true` (so `incomplete()` returns true), `NextCursor` decoded `"abc"`, and the role `description`/`purpose` fields populated — pinning the snake_case JSON tags
    - Unit tests: multi-role, empty list, null purpose, a role with neither domains nor accountabilities, field-absence guarantee
  - **Dependencies**: `internal/glassfrog` (011, implemented on `main`)
  - **Plan reference**: Phase 1, ADR-4
  - **Interface references**: interface-cli.md: Surface (output format)
  - **Scenario references**: my-roles.feature: "A projected role shows its essentials only", "The default output contains no raw API envelope"

## Phase 2: me roles command [US1]

- [ ] **T002** [US1] The `roles` subcommand under the runnable `me` command
  - **Scope**: Register a `roles` `NoArgs` leaf under 011's runnable `me` command (guard-registered, non-empty `Short`; confirm the guard permits `me` runnable-with-children). A thin `RunE` over an injected seam delegates to a pure `runMyRoles(cfg)`: read the inherited root `--base-url`, `AssembleFromOS(flag)` → `NewClientFromOS(ctx)` → `Execute(reqCtx, Request{Method:"GET", Path:"/me/roles"}, &resp)`; on success `formatMyRoles` → stdout and, when `incomplete`, the note → stderr; on error route through **011's `classifyClientError`** → `Outcome` → exit code (no inline `errors.As` chain, no new category). Never reads the token.
  - **Acceptance criteria**:
    - `glassfrog me roles` on a multi-role context prints the projection and exits 0; an empty context prints `No roles.` and exits 0
    - Failures map via `classifyClientError`: no token→2 ("not authenticated — run `glassfrog auth login`"), unreadable credential→1, wire failure→6, non-2xx→3 (status named), undecodable 2xx→1, malformed base URL→2 (nothing sent), stray arg→2 (nothing sent)
    - When `has_next_page` is true, the incompleteness note is on stderr, the projection on stdout, exit 0
    - Hermetic tests over a fake base `RoundTripper` for every branch; a token-never-in-output test across stdout and stderr; the command registers no `--base-url` flag of its own and adds no `Outcome`/`ExitCode` case
  - **Dependencies**: T001; 010 (implemented); 011 scaffolding (`me` command, `classifyClientError`, root `--base-url`)
  - **Plan reference**: Phase 2, ADR-1, ADR-2, ADR-3
  - **Interface references**: interface-cli.md: Surface, Interactions, Error Communication
  - **Scenario references**: my-roles.feature: "The roles the practitioner fills are listed", "An empty role list is a clean success", "A missing token fails as a not-authenticated usage error", "A wire failure is reported as a transport failure", "A non-2xx response reports the read failed with its status", "A stray argument is rejected before any API call", "A malformed base URL is refused before any call", "An unparseable response body fails loudly", "An incomplete list is signalled, not silently truncated"
  - **Risk**: ⚠️ Depends on 011's scaffolding being implemented (or created to contract)

- [ ] **T003** [Shared] godog acceptance suite for My Roles
  - **Scope**: A godog suite pointed at **its own** `features/self-service-reads/my-roles.feature`, step definitions backed by a fake base transport returning canned `/me/roles` payloads (multi-role, empty, has-next-page) and contexts for the failure branches. Step helpers return errors, never panic; reuse existing `internal/cli` step vocabulary where present. Remove `@wip` as scenarios pass.
  - **Acceptance criteria**:
    - The suite runs only `my-roles.feature` (never the whole `features/` tree)
    - Every spec-derived and validation scenario has a passing step definition; per-scenario exit codes (0/1/2/3/6) and the projected-field absences are asserted
    - No real network or real `~/.glassfrogrc` access; `@wip` removed from passing scenarios
  - **Dependencies**: T002
  - **Plan reference**: Phase 3
  - **Scenario references**: my-roles.feature (all scenarios)
