# Tasks: Organization Tree

**Feature**: 026-organization-tree
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/governance-reads/organization-tree.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` schema (2 tasks, no phase dependencies) [Shared]
Phase 2: The `tree` reads (1 task, depends on Phase 1 · T001) [Shared]
Phase 3: The `subroles` read + acceptance (2 tasks, depends on Phase 1 · T002 and Phase 2) [Shared]

5 tasks total | T001 and T002 startable immediately (parallel) | Builder: pipeline (with role-based awareness — see Branching Guidance)

> Every task is `[Shared]`: the two commands serve all four user scenarios (whole-org tree US1, rooted subtree US2, subroles US3, depth-capping US4). The plan's three phases map to: schema (T001 `TreeNode`, T002 shared `RoleDetail`) → `tree` reads (T003) → `subroles` read (T004) + executable acceptance (T005).
>
> **Hard dependencies are landed on main.** Per STATUS.md, 007/009/010/015/016/017/018/020 are Complete and 019's `internal/render` package is landed (the `*.full/compact.tmpl` templates ship from it). So the 026 base is cut from current main. The BACKLOG declares only 007/010; the **tree reads (T001 → T003) are fully independent of 025** — they use `TreeNode`, no role-detail schema, no role id required for the whole-org read.
>
> **Shared-schema coordination with 025 (not a hard dependency).** T002 (the `RoleDetail` + `Assignment`/`Policy`/`Note`/`SkillSummary` leaf models + `Role` growth) is the SAME schema 025 designs (025 ADR-2 / its T001). **025 is *Analyzed*, not implemented** — none of it exists in `internal/glassfrog` yet (today it holds only the minimal `Role`/`Accountability`/`Domain`/`Actor`). So whichever of 025/026 implements first **creates** these types; the other **reuses** them verbatim (the 005/006/016 first-to-land-creates rule). T002 must be written create-if-absent-else-reuse and coordinated with any concurrent 025 work so only one definition exists.
>
> **Existing main state 026 builds on, not around**: `internal/cli/roles.go` is a wired (`Assemble()`) `roles` *group* with `list`/`get` stubs (025's concern, untouched here). 026 adds **two new sibling commands** (`tree`, `subroles`) — never children of `roles` (plan ADR-1 / 025 ADR-1 foreclosure) — and **two new** render keys (`tree`, `subroles`), distinct from the shipped `roles` key and from 025's planned `org-roles`/`role`. 026 adds **no** new `Outcome` category, `ExitCode` case, validator-beyond-its-own, or root flag.

---

## Branching Guidance

**Pipeline mode**: `spec/026-organization-tree/base` → `spec/026-organization-tree/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all hard dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry 025 (Role Reads), which defines the **same** `RoleDetail`/leaf-model/`Role`-growth schema T002 needs. Coordinate so only one definition of those types lands in `internal/glassfrog` — if 025 lands first, T002 reduces to "reuse, add nothing." 026 touches `internal/glassfrog`, `internal/cli`, and `internal/render`; the `tree` half (T001/T003) is collision-free.

---

## Phase 1: `internal/glassfrog` schema [Shared]

- [ ] **T001** [Shared] [P] Add the recursive `TreeNode` type and the `{data: TreeNode}` document wrapper — RED-first decode tests; `internal/glassfrog/tree.go` + tests
  - **Scope**: In `internal/glassfrog`, add a `TreeNode` struct decoding the `getOrgTree`/`getRoleTree` body: `ID string`, `Type string`, `Name *string`, `Purpose *string`, `ParentRoleID *string`, `HasSubroles bool`, `Flags []string`, `Children []TreeNode` (the recursion — children are `TreeNode`, not `Role`), and the optional `?include` fields `Accountabilities []Accountability`, `Domains []Domain`, `Fillers []Actor` (reuse the existing leaf projections; extend them only if the tree node carries a field the render needs that the current minimal projection drops). Add the single-object document wrapper (`TreeDocument{ Data TreeNode }`, or reuse a generic `Document[T]` if one exists / is introduced by 025's wrapper — first single-object read creates it). **Not paginated** — no `Page`/`Meta` here. Decoding tolerates unknown/extra fields; no transport, no cobra, no exit codes; the token is never a field.
  - **Acceptance criteria**:
    - A `GET /tree` fixture decodes into `TreeDocument`; `Children` nests recursively to multiple levels
    - `Name`/`Purpose`/`ParentRoleID` decode as nullable (a null `parent_role_id` on the anchor node is fine); `Flags` decodes the `structural`/`elected`/`linked` values
    - With each `?include` value present, the matching field (`Accountabilities`/`Domains`/`Fillers`) populates; when absent it stays nil/empty
    - A leaf node decodes with an empty (non-nil-required) `Children`; unknown/extra fields are ignored
    - No new internal imports; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (leaf schema). Reuses `Accountability`/`Domain`/`Actor` (011/012).
  - **Plan reference**: Phase 1 (Schema); ADR-2 (recursive `TreeNode`, `{data: TreeNode}` wrapper, not paginated); System Architecture (`internal/glassfrog`)
  - **Interface references**: interface-cli.md — Surface (tree output shape)
  - **Scenario references**: organization-tree.feature: "The whole organization tree is read", "A subtree rooted at a role is read", "A leaf root renders as a single node"
  - **Risk**: ⚠️ `Children` is `[]TreeNode` (recursion is the point) — do **not** reuse `RoleDetail.Subroles` (flat `[]Role`, no recursion — 025 ADR-2). ⚠️ Keep `TreeNode` separate from `Role`/`RoleDetail` (different shape: children/flags/type). ⚠️ Single-object wrapper, **no** pagination envelope. ⚠️ Token is never a field.

- [ ] **T002** [Shared] [P] Add the shared role-detail schema (`RoleDetail` + leaf models + `Role` growth) for the subroles read — create-if-absent-else-reuse, coordinate with 025 — RED-first decode tests; `internal/glassfrog/roles.go` (+ related model files) + tests
  - **Scope**: The schema the `subroles` read decodes (`Page[RoleDetail]`). This is the **same** schema 025 ADR-2 designs. If 025 has not landed it, **create** it here: grow the shared `Role` with the remaining spec fields (`Type`, `ParentRoleID *string`, `HasSubroles bool`, `Flags []string`, `Fillers []Actor`, `Tags []string`) — never a second role type (011 ADR-1); add `RoleDetail` embedding `Role` plus `Assignments []Assignment`, `Subroles []Role`, `ParentRole *Role`, `Policies []Policy`, `Notes []Note`, `Skills []SkillSummary` (nested `Subroles`/`ParentRole` are plain `Role` — no recursion); add minimal leaf models `Assignment`, `Policy` (incl. `Title`), `Note`, `SkillSummary` (no `Content`). The subroles list decodes the **existing** generic `Page[RoleDetail]` (016) — do not define a new envelope. If 025 has already landed these, this task is **reuse only** — add nothing, just confirm the types exist and `Page[RoleDetail]` decodes. Decoding tolerates unknown/extra fields; no transport, cobra, or exit codes; token never a field.
  - **Acceptance criteria**:
    - A `GET /roles/{id}/subroles` page fixture decodes into `Page[RoleDetail]` with `Data` populated and `Meta.Pagination` read
    - With each `?include` value present on a child, the matching `RoleDetail` field populates; when absent it stays nil/empty; `SkillSummary` has no `Content`
    - A null `parent_role_id` and empty related slices decode without error; nested `Subroles`/`ParentRole` are plain `Role`
    - If the types already exist (025 landed), no duplicate definition is introduced; unknown/extra fields ignored; `go build`/`go vet` clean
  - **Dependencies**: None (leaf schema). Reuses `Page[T]`/`Pagination` (016) and `Actor` (011). **Coordinate with 025** (same types).
  - **Plan reference**: Phase 1 (Schema); ADR-3 (subroles decodes `Page[RoleDetail]`, shared schema under first-to-land)
  - **Interface references**: interface-cli.md — Surface (subroles output, per-child includes)
  - **Scenario references**: organization-tree.feature: "A role's immediate subroles are listed"
  - **Risk**: ⚠️ This is 025's schema — create exactly once; if 025 lands first, REUSE and add nothing (first-to-land). ⚠️ Grow the shared `Role`; never fork a list/detail type (011 ADR-1). ⚠️ Nested `Subroles`/`ParentRole` are plain `Role` (no recursion); `SkillSummary` is summary-only. ⚠️ Reuse `Page[T]`; never define a 026-local envelope.

## Phase 2: The `tree` reads [Shared]

- [ ] **T003** [Shared] Add the `tree` command (whole-org + rooted subtree): depth + include validation, both endpoints, `TreeNode` decode, the recursive `tree` render, wiring — RED-first unit tests for every branch; new `internal/cli/tree.go` + `tree_test.go`, new `tree` render templates
  - **Scope**: New guard-registered, explicitly-wired leaf. `newTreeCommand(seam treeSeam) *cobra.Command`: `Use:"tree [id]"`, `Args: cobra.MaximumNArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`; reads the persistent `--base-url` (011) + `--output`/`-o` (020); declares local `--depth` (optional int) and `--include`. Delegates to pure `runTree`. `runTree(cfg) (Outcome, error)`: resolve `--output` (020); run `validateTreeFlags` **before** assembly — reject `--first-page`/`--per-page` (tree is unpaginated), reject negative `--depth`, and `validateTreeInclude` rejects any `--include` value outside `{accountabilities,domains,members}` (the 011 `validateInclude` shape; UsageError(2), transport tripwire) — even though the API would silently ignore it (plan ADR-4). Branch on `len(args)`: 0 → `GET /tree`, 1 → `GET /roles/{id}/tree`. Send `depth` **only when the flag was set** (`cmd.Flags().Changed` — so `--depth 0` ≠ omitted); send `?include=` from the validated values. The role **id is passed through** (not validated locally — plan ADR-4). Issue **one** `Execute` into `TreeDocument` (no walk, no `If-None-Match`), then dispatch through `renderResult("tree", format, doc.Data)`. Register **new** `tree` `full`/`compact` templates in `internal/render` (019): depth-indented recursive rendering (two spaces per level), id + name + flags per node, `Purpose` with explicit-absence marker, guarded per-node include sections (omit-when-unrequested, marker-when-empty), a leaf renders nothing indented; registry exhaustiveness guard. Wire `MustRegister(root, newTreeCommand(...))` in `Assemble()`. `tree` never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog tree` walks no pages — one request — and prints the nested projection rooted at the anchor role; exits 0
    - `glassfrog tree role_0123` reads `GET /roles/{id}/tree` and prints the subtree rooted at `role_0123`; a leaf root prints a single node with nothing indented; exits 0
    - A node whose `has_subroles` is true but whose `children` are absent (depth cap / API-withheld) renders the depth-boundary marker (`(+ subroles below depth)` in `full`, `has_subroles=yes` in `compact`), distinct from a true leaf (`has_subroles: false`); the marker is driven by the API boolean and carries no invented descendant count (spec Clarifications; risk RC-3; CONSTITUTION VI/VIII)
    - `--depth 1` sends `depth=1`; omitting `--depth` sends no `depth`; `--depth 0` sends `depth=0` (root only); a negative `--depth` is UsageError(2) with no request
    - `--include accountabilities,domains` sends `include=accountabilities,domains` and renders those per node; an unsupported `--include` value is UsageError(2) naming the value + the tree set, **no request sent** (tripwire); `--first-page`/`--per-page` on `tree` are UsageError(2)
    - An unknown role id surfaces the API status via the shared classifier (APIError(3)/PermissionError(4)); `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*DecodeError` → RuntimeError(1); base-URL error / invalid `--output` → UsageError(2)
    - `-o json`/`yaml` emit the raw nested payload verbatim (018); `full`/`compact` render the human tree (019); the new `tree` render key has both formats and passes the registry guard; no `Outcome`/`ExitCode`/root flag added
    - No output renders the token; all branches run offline over the fake seam; `go build`/`go vet` clean
  - **Dependencies**: T001 (`TreeNode`/document wrapper). Reuses 009/010/011/015/017/018/019/020 (all landed).
  - **Plan reference**: Phase 2 (Tree reads); ADR-1 (`tree` optional id, sibling of `roles`), ADR-2 (`TreeNode`, unpaginated), ADR-4 (per-read include validation, optional depth); Cross-cutting (error handling, output, no caching, testing)
  - **Interface references**: interface-cli.md — `glassfrog tree` Surface, Interactions (unpaginated, no caching), Error Communication
  - **Scenario references**: organization-tree.feature: "The whole organization tree is read", "A missing token fails as a not-authenticated usage error", "A subtree rooted at a role is read", "Requested per-node resources are embedded in the tree", "An unknown role id fails with the API status", "An unsupported tree include value is rejected before any request", "A leaf root renders as a single node", "A depth flag bounds the tree to direct children", "A depth-capped node is marked as having subroles below"
  - **Risk**: ⚠️ `--depth` is optional — use `Changed`, never a plain int default (0 = root-only ≠ omitted = full tree). ⚠️ Reject unknown `--include` locally even though the API silently ignores it (plan ADR-4); validate against the **tree** set only. ⚠️ Tree reads are unpaginated — one `Execute`, no `paging.All`, no `--first-page`/`--per-page`. ⚠️ Recursive render is new ground — depth-indent via the template; structured output uses the raw-bytes path (018), no special nesting handling. ⚠️ Reuse `classifyClientError`/`renderResult` — no second `errors.As` chain or render branch. ⚠️ Send no `If-None-Match` (no caching — spec Non-Behavior). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

## Phase 3: The `subroles` read + acceptance [Shared]

- [ ] **T004** [Shared] Add the `subroles <id>` command: include validation, the page walk + `--first-page` opt-out + completeness signalling + `--per-page`, the `subroles` render, wiring — RED-first unit tests for every branch; new `internal/cli/subroles.go` + `subroles_test.go`, new `subroles` render templates
  - **Scope**: New guard-registered, explicitly-wired leaf. `newSubrolesCommand(seam subrolesSeam) *cobra.Command`: `Use:"subroles <id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`; reads the persistent `--base-url` + `--output`/`-o`; declares local `--include`, `--first-page`, `--per-page`. Delegates to pure `runSubroles(cfg, id)`. Validate **before** assembly: `validateSubrolesFlags` rejects `--depth` (subroles is one level — UsageError(2), tripwire); `validateSubrolesInclude` rejects any `--include` value outside `{assignments,subroles,parent_role,policies,notes,skills}` (the `getRole` set; UsageError(2), tripwire). Default path walks `GET /roles/{id}/subroles` to completion via `paging.All[RoleDetail]` over the seam's executor (`RetryExecutor` wrapping `NewClientFromOS`, 016/017), sending `?include=` from the validated values; `--first-page` does a single `Execute` into `Page[RoleDetail]` (no walk), rendering the first page and writing one stderr note when `HasNextPage` (exit 0); a mid-walk failure (`Result.Complete == false`, non-nil `Stop`) renders the partial `Records`, writes one stderr "incomplete — <cause>" note, and exits non-zero via `classifyClientError(Stop)`. `--per-page` sets the walk page size (016 `WithPageSize`). The role **id is passed through** (not validated locally). Dispatch through `renderResult("subroles", format, records)`. Register **new** `subroles` `full`/`compact` templates in `internal/render`: a child role block (the 025 role-block shape) with guarded per-child include sections; empty set prints `No subroles.`; registry guard. Wire `MustRegister(root, newSubrolesCommand(...))` in `Assemble()`. Never reads `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog subroles role_0123` walks every page (`paging.All`) and prints each direct child role; exits 0
    - A leaf role prints `No subroles.` (human format) and exits 0
    - `--include subroles,assignments` sends `include=subroles,assignments` and renders them inline per child; an unsupported `--include` value is UsageError(2) naming the value + the subroles set, **no request sent** (tripwire)
    - `--depth 2` on `subroles` is UsageError(2) with **no request sent** (tripwire)
    - `--first-page` against a multi-page role prints only the first page, writes a "more subroles exist" stderr note, and exits 0; a mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); `*MalformedPageError` → RuntimeError(1); an unknown id surfaces the API status
    - `-o json`/`yaml` emit the raw payload; the new `subroles` render key has both formats and passes the registry guard; no `Outcome`/`ExitCode`/root flag added
    - No output renders the token; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T002 (`RoleDetail` decode target / `Page[RoleDetail]`). Reuses 009/010/011/015/016/017/018/019/020 (all landed).
  - **Plan reference**: Phase 3 (Subroles read); ADR-1 (`subroles <id>` sibling, `--depth` rejected), ADR-3 (walk + `--first-page` opt-out, reuse 025 verbatim), ADR-4 (validate `--include` per set, pass id through); Cross-cutting (error handling, output, testing)
  - **Interface references**: interface-cli.md — `glassfrog subroles` Surface, Interactions (subroles completeness), Error Communication
  - **Scenario references**: organization-tree.feature: "A role's immediate subroles are listed", "Requested related resources are embedded on each child", "The subroles first-page opt-out stops at one page and signals more", "A leaf role's subroles are an empty success", "A multi-page subroles list is walked to completion", "A depth flag on subroles is a usage error", "A mid-walk subroles failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse 025's pagination shape verbatim — `paging.All` default + single-page `--first-page` signal (not `paging.WithMaxPages`), preserve 016's `Result.Complete == (Stop==nil)` invariant. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Reject `--depth` on subroles (one level only). ⚠️ Validate `--include` against the **subroles** set; pass the id through. ⚠️ Reuse `classifyClientError`/`renderResult`/`paging.All`/`RetryExecutor` — inline no second chain, render branch, or page loop. ⚠️ Temp-file capture in tests, not `os.Pipe` (PR #10). ⚠️ Never read `ctx.Cred.Token`.

- [ ] **T005** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `organization-tree.feature`; un-@wip the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/governance-reads/organization-tree.feature` in a **new** `internal/cli` godog suite (e.g. `TestOrganizationTreeFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `tree` and `subroles` through their seams over a fake base `http.RoundTripper` returning canned `GET /tree`, `GET /roles/{id}/tree`, and single-/multi-page `GET /roles/{id}/subroles` responses (incl. a mid-walk error and a nested tree with depth). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Reuse existing `internal/cli` step phrasings where assertions already exist (grep `sc.Step(` registrations first — exit-code, stderr-substring, request-query, and projection phrasings are shared with the `me*`/roles reads); step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` organization-tree scenario has an executable, passing path; `@wip` removed from them
    - The 3 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `organization-tree.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (`tree` reads), T004 (`subroles` read) — all behavioral scenarios must be implementable
  - **Plan reference**: Phase 2 + Phase 3; System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: organization-tree.feature: all behavioral Rule-block scenarios (the 3 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `organization-tree.feature` only (not the directory); verify it reports its own count. ⚠️ Grep existing `sc.Step(` registrations and reuse shared read phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the nested-tree, depth-bounded, multi-page, and mid-walk-error fakes so the recursion and completeness scenarios genuinely exercise the behavior.
