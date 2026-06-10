# Tasks: Role Policies

**Feature**: 034-role-policies
**Concretization**: Full context (plan + spec + interface-cli + interface-spec + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/governance-reads/role-policies.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` schema growth + `Document[T]` (1 task, no phase dependencies) [Shared]
Phase 2: `internal/render` policy templates (1 task, no phase dependencies — parallel with Phase 1) [Shared]
Phase 3: The two commands (2 tasks, depends on Phases 1, 2) [US1/US3/US4 + US2]
Phase 4: Executable acceptance (1 task, depends on Phase 3) [Shared]

5 tasks total | T001 and T002 startable immediately (parallel) | Builder: pipeline

> The plan's four phases map 1:1 to phases here: schema (T001) and render (T002) are independent and parallel; the two commands (T003 list, T004 single) depend on both; executable acceptance (T005) depends on the commands. Story labels follow the spec's four user scenarios — US1 (list a role's policies), US2 (read one policy's full body), US3 (narrow with search), US4 (trust the list is whole). T003 (the `policies` list) serves US1/US3/US4 together → `[Shared]`; T004 (the `policy` single read) is cleanly `[US2]`.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/015/016/017/018/020 are Complete, 019/025 are Complete/Ready with their packages landed — `internal/glassfrog.Policy` (minimal `ID`/`Title`/`Body`), `RoleDocument`, `Page[T]`/`Pagination`, `paging.All`, `RetryExecutor`, `classifyClientError`/`Outcome`/`ExitCode`, the persistent `--base-url`/`--output`/`-o`, and `renderResult[T]` over `internal/output`/`internal/render` all ship from main. So the 034 base cuts from current main with no sequencing caveat. **Existing main state 034 builds on**: `internal/glassfrog.Policy` already exists in its minimal `ID`/`Title`/`Body` shape (T001 grows it, not duplicating — 011 ADR-1); `RoleDocument` already exists as a concrete wrapper (T001 generalizes it to `Document[T]`); there is **no** `policies`/`policy` command and **no** `policy`/`policies` render key (both new). 034 adds **no** new `Outcome` category, `ExitCode` case, validator, or root flag.
>
> **The addressable policy surface Role Reads (025) deferred here.** Distinct from `roles <id> --include=policies` (embedded view). Sets the per-role-read surface precedent (plural `policies <role-id>` + singular `policy <pol-id>`) that #33 (Role Domains) and #38 (Role Projects) follow — see plan ADR-1.

---

## Branching Guidance

**Pipeline mode**: `spec/034-role-policies/base` → `spec/034-role-policies/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling per-role reads (#33 Role Domains, #38 Role Projects). 034 touches `internal/glassfrog` (grows `Policy`, generalizes `RoleDocument`→`Document[T]`), `internal/cli` (new `policies.go`), and `internal/render` (new keys). Coordinate the `Document[T]` generalization with any concurrent single-object read so only one definition exists.

---

## Phase 1: `internal/glassfrog` schema growth + `Document[T]` [Shared]

- [x] **T001** [Shared] [P] Grow `Policy` with scope + timestamps and generalize the single-object envelope to `Document[T]` (refactor `RoleDocument`) — RED-first decode tests; `internal/glassfrog/roles.go` (+ a `Document` location) + tests — 4 decode tests; RoleDocument kept byte-stable as a type alias
  - **Scope**: In `internal/glassfrog`, grow the shared `Policy` (currently `ID`/`Title`/`Body`) with `RoleID string`, `DomainID string`, `CreatedAt string`, `UpdatedAt string` — nullable `role_id`/`domain_id` modeled as plain strings (empty = null), mirroring the existing nullable `Body`; never a second policy type (011 ADR-1). Introduce a generic single-object envelope `Document[T any]{ Data T \`json:"data"\` }` (the single-read counterpart to `Page[T]`, 016), and refactor the landed `RoleDocument` to `Document[RoleDetail]` — keep `RoleDocument` as a **type alias** (`type RoleDocument = Document[RoleDetail]`) so 025's decode call site and BDD stay byte-stable (plan Risk 1). The single policy read will decode `Document[Policy]`; the list decodes the **existing** `Page[Policy]` (016) — define no new list envelope. Decoding tolerates unknown/extra fields; no transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A `GET /policies/{id}` fixture decodes into `Document[Policy]`; `Data` exposes `ID`/`Title`/`Body`/`RoleID`/`DomainID`/`CreatedAt`/`UpdatedAt`
    - A policy with null `role_id` and null `domain_id` decodes without error (empty strings)
    - A `GET /roles/{id}/policies` page fixture decodes into `Page[Policy]` with `Data` populated and `Meta.Pagination` read
    - `RoleDocument` remains usable at 025's existing call site (alias) — 025's decode tests still pass
    - Unknown/extra fields are ignored; no new internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Reuses `glassfrog.Page[T]`/`Pagination` (016).
  - **Plan reference**: Phase 1 (Schema); ADR-2 (grow `Policy`, generalize `Document[T]`); System Architecture (`internal/glassfrog`)
  - **Interface references**: interface-spec.md — `internal/glassfrog` (Surface): `Policy` (grown), `Document[T]`, `RoleDocument` (refactored)
  - **Scenario references**: role-policies.feature: "A single policy is read with its full body", "A role's policies are listed"
  - **Risk**: ⚠️ Grow the shared `Policy`; do **not** fork a list/detail type (011 ADR-1). ⚠️ Keep `RoleDocument` as an alias of `Document[RoleDetail]` so 025 stays byte-stable. ⚠️ Reuse `Page[T]`; never define a 034-local list envelope. ⚠️ `role_id`/`domain_id` are nullable — empty string on null, never a panic.

## Phase 2: `internal/render` policy templates [Shared]

- [x] **T002** [Shared] [P] Add the `policy` and `policies` render keys + view structs + templates — golden + registry-guard tests; `internal/render` (new `policy.{full,compact}.tmpl`, `policies.{full,compact}.tmpl`) + tests — 4 templates, 9 golden/guard tests; verbatim-body check
  - **Scope**: In `internal/render`, add two **new** render keys: `policies` (list, data `[]glassfrog.Policy` via a `PoliciesView`) and `policy` (single, data `glassfrog.Policy` via a `PolicyView`). Add four `//go:embed` templates: `policies.full` (one block per policy — title, id, role/domain scope with explicit-absence markers), `policies.compact` (one line per policy — `pol_…  <title>  role=…  domain=…`), `policy.full` (the policy block plus the **full body rendered verbatim — never truncated or reflowed**, CONSTITUTION VI, and `created_at`/`updated_at` with explicit-absence guards), `policy.compact` (title + id + scope, body omitted). Use 019's `Option("missingkey=error")` + `{{if}}` absence guards; render the inherited empty-set line `No policies.` for the list. Register both keys in the same set the registry-exhaustiveness guard checks (PR #10 `len`+comma-ok). Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `policies` key renders both `full` and `compact`; an empty list renders exactly `No policies.`
    - The `policy` key renders both `full` and `compact`; `full` shows the complete body verbatim (a long/HTML body is neither truncated nor reflowed)
    - A policy with null `role_id`/`domain_id` renders its explicit-absence markers, never `<no value>` or an invented value
    - The registry exhaustiveness guard passes with both new keys carrying both formats; golden tests pin each template
    - `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: None for the engine; the view structs reference `glassfrog.Policy` (grown by T001) — golden fixtures can use the grown shape, so T002 can land in parallel with T001 and only the fixtures need the new fields.
  - **Plan reference**: Phase 2 (Render); ADR-4 (`policy`/`policies` render keys)
  - **Interface references**: interface-spec.md — `internal/render` additions
  - **Scenario references**: role-policies.feature: "A role with no policies is a clean success", "A single policy is read with its full body"
  - **Risk**: ⚠️ Never truncate or reflow the policy `Body` — faithfulness over brevity (CONSTITUTION VI). ⚠️ Use explicit-absence guards for null `role_id`/`domain_id`; never invent a value. ⚠️ New keys only — touch no existing template/key.

## Phase 3: The two commands [US1/US3/US4 + US2]

- [x] **T003** [Shared] Add the `policies <role-id>` list command + the shared `policiesSeam` + `--query`/`--first-page`/`--per-page` + completeness + wiring — RED-first unit tests for every branch; new `internal/cli/policies.go` + tests + `Assemble()` wiring — every branch + classification covered
  - **Scope**: New `internal/cli/policies.go`. `newPoliciesCommand(seam policiesSeam) *cobra.Command`: a guard-registered leaf (`Use:"policies <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), declares `--query`/`-q`, `--first-page`, `--per-page`; reads the persistent `--base-url`/`--output`; delegates to pure `runPoliciesList`. Define the shared `policiesSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client` executor — prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Executor`). `runPoliciesList(cfg) (Outcome, error)`: resolve `--output` (020) **before** assembly; build the `q` query only when `--query` is `Changed()` and non-empty; default path walks `GET /roles/{id}/policies` to completion via `paging.All[Policy]` and dispatches through `renderResult("policies", format, records)`; `--first-page` does one `Execute` into `Page[Policy]` and writes a "more policies exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The role id is passed through (`url.PathEscape`, no local validation — plan ADR-3). Wire `MustRegister(root, newPoliciesCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog policies role_0123` walks every page (`paging.All`) and prints the projection of all the role's policies; exits 0
    - A role with no policies prints `No policies.` and exits 0
    - `--query approvals` sends `q=approvals` and prints only matching policies; `--query ""` (or omitted) sends no `q`
    - `--first-page` against a multi-page list prints one page, writes a "more policies exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the raw payload, `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`Page[Policy]` decode target), T002 (`policies` render key)
  - **Plan reference**: Phase 3 (Commands); ADR-1 (sibling command, `ExactArgs(1)`), ADR-3 (`--query` free-text pass-through, id pass-through); Cross-cutting (list completeness reuses 025 ADR-3, error handling, output)
  - **Interface references**: interface-cli.md — `policies` Surface, list flags, Output, Interactions (completeness), Error Communication; interface-spec.md — `newPoliciesCommand`/`runPoliciesList`/`policiesSeam`
  - **Scenario references**: role-policies.feature: "A role's policies are listed", "A role with no policies is a clean success", "A missing token fails as a not-authenticated usage error", "The policy list is narrowed by a search query", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse `classifyClientError`/`renderResult`/`paging.All`/`RetryExecutor` — inline no second `errors.As` chain, render branch, or page loop. ⚠️ `--query` is sent only when `Changed()` and non-empty (026 `--depth` optional-flag discipline); no enum validation. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

- [x] **T004** [US2] Add the `policy <pol-id>` single-read command — `runPolicyGet` + `Document[Policy]` decode + the `policy` render dispatch — RED-first unit tests; extends `internal/cli/policies.go` + `Assemble()` wiring — list-only-flag tripwire across all 3 flags
  - **Scope**: In `internal/cli/policies.go`, add `newPolicyCommand(seam policiesSeam) *cobra.Command`: a guard-registered leaf (`Use:"policy <pol-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`), declares **no** list flags (so `--query`/`--first-page`/`--per-page` are cobra unknown-flag usage errors — the structural list-only guard, plan ADR-1); reads the persistent `--base-url`/`--output`; delegates to pure `runPolicyGet`. `runPolicyGet(cfg, id) (Outcome, error)`: resolve `--output` before assembly; one `Execute` into `Document[Policy]` (`{data: Policy}`) over the shared seam, with the id `url.PathEscape`-d and passed through unvalidated (plan ADR-3) so an unknown id surfaces the API's non-2xx via the shared classifier; dispatch through `renderResult("policy", format, doc.Data)`. Wire `MustRegister(root, newPolicyCommand(...))` in `Assemble()`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog policy pol_0123` reads the policy and prints its title + full body; exits 0
    - An unknown `pol_ffff` surfaces the API status (APIError(3)/PermissionError(4)) — the id is not validated locally
    - A list-only flag on `policy` (`--query`/`--first-page`/`--per-page`) is a UsageError(2) via cobra's unknown-flag handling, **no request sent** (transport tripwire)
    - `-o json`/`yaml` emit the raw single-policy payload; `full`/`compact` render the `policy` human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`Document[Policy]`), T002 (`policy` render key), T003 (the shared `policiesSeam` + file scaffold)
  - **Plan reference**: Phase 3 (Commands); ADR-1 (singular command, list-only-ness structural), ADR-3 (id pass-through), ADR-2 (`Document[Policy]`)
  - **Interface references**: interface-cli.md — `policy` Surface, Output (single), Error Communication; interface-spec.md — `newPolicyCommand`/`runPolicyGet`
  - **Scenario references**: role-policies.feature: "A single policy is read with its full body", "An unknown policy id fails with the API status", "The search flag is rejected on the single read"
  - **Risk**: ⚠️ Declare no list flags on `policy` — that's how list-only-ness is enforced (no hand-rolled cross-combo guard; plan ADR-1). ⚠️ Pass the id through (no regex gate); the API 404s cleanly (plan ADR-3). ⚠️ Render the full body verbatim. ⚠️ Never read `ctx.Cred.Token`.

## Phase 4: Executable acceptance [Shared]

- [x] **T005** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `role-policies.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held — 9 scenarios pass; 2 @validation held; noted dispatch flagFailed-stderr gap in LEARNINGS
  - **Scope**: Add godog step definitions for `features/governance-reads/role-policies.feature` in a **new** `internal/cli` godog suite (e.g. `TestRolePoliciesFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `policies`/`policy` through the shared seam over a fake base `http.RoundTripper` returning canned `GET /roles/{id}/policies` (single-page, multi-page, mid-walk error, empty) and `GET /policies/{id}` (found, 404) responses. Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 2 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection phrasings from the `me*`/`roles` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` role-policies scenario has an executable, passing path; `@wip` removed from them
    - The 2 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `role-policies.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (`policies` list), T004 (`policy` single) — all behavioral scenarios must be implementable
  - **Plan reference**: Phase 4 (BDD); System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: role-policies.feature: all behavioral Rule-block scenarios (the 2 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `role-policies.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mid-walk-error, multi-page, and empty-list fakes so the completeness + empty scenarios genuinely exercise the walk.
