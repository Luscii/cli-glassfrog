# Plan: Role Reads

**Feature**: 025-role-reads
**Role**: Shaper
**Inputs**: spec.md (025), PROJECT.md, `.score/memory/DECISIONS.md` (40+ entries), DEPRECATION.md (1 entry), LEARNINGS.md (passive), spec/glassfrog-api-v5.yaml (`listRoles`/`getRole`/`Role`/`RoleDetail`)

---

## System Architecture

Role Reads is the first **org-wide** read command and the dependency root of Governance Reads. It adds **one cobra command, `roles`**, to the existing tree and otherwise composes the landed read stack — it builds no new transport, identity, pagination, error-classification, or output machinery.

**Components and how they connect**:

- **`internal/cli` — the `roles` command** (new `roles.go` + godog suite). A guard-registered (001), explicitly-wired (001) runnable leaf that takes an *optional* positional id. A thin cobra `RunE` over an injected seam (011 ADR-5 pattern) delegates to two pure run paths:
  - **list** (`glassfrog roles`, no id) — validates filter flags, assembles the connection context, and walks `GET /roles` to completion via the shared walker, producing `[]Role` + a completeness flag.
  - **single** (`glassfrog roles <id>`) — validates `--include`, fetches `GET /roles/{id}`, producing one `RoleDetail`.
  Both paths route 010's typed client errors through the shared `classifyClientError` (011/015) — **no new `Outcome` category, no `ExitCode` edit** — and render through the generic `renderResult[T]` dispatch (020), which picks `internal/output` (json/yaml, raw bytes) or `internal/render` (full/compact templates) by the resolved `--output` format.
- **`internal/glassfrog` — schema growth** (011 ADR-1). The shared `Role` struct gains the spec fields it still lacks (`type`, `parent_role_id`, `has_subroles`, `flags`, `fillers`, `tags`); a new `RoleDetail` embeds `Role` and adds the optional related-resource fields. New leaf resource models (`Assignment`, `Policy`, `Note`, `SkillSummary`) are added here as needed — never command-local duplicates. The list decodes the existing generic `Page[Role]` (016); the single decodes a `{data: RoleDetail}` document wrapper.
- **`internal/paging` — the walker** (016). The list's default path calls `paging.All[Role]` over the retrying `Executor` (017) for the complete set.
- **`internal/render` — templates** (019). Adds `roles`/`role` template sets (full + compact) to the embedded registry; the registry exhaustiveness guard (PR #10 shape) keeps both formats present.

**Data flow** (list): `roles` → validate filters → `AssembleFromOS(--base-url)` (009) → `NewClientFromOS` (010) → `RetryExecutor` (017) → `paging.All[Role]` → `Result[Role]` → `renderResult` → stdout (+ incompleteness note to stderr). Single read is the same minus the walker: one `Execute` into `RoleDetail`.

---

## Architecture Decisions

### ADR-1: `roles` is one runnable command with an optional positional id; 0 args lists, 1 arg reads one

**Context**: The spec (clarified choice A) wants `glassfrog roles` (list) and `glassfrog roles <id>` (single, positional id) — not an explicit `get` verb. The 001 guard permits a runnable command, and 012 confirmed a command may both run and (in its case) parent children.

**Options considered**:
1. **One `roles` command, `Args: cobra.MaximumNArgs(1)`, branch in RunE** — 0 args → list, 1 arg → single. Matches the chosen positional UX exactly; one registration, one help entry.
2. **Two commands: `roles` (list) + `roles get <id>`** — explicit verb. Cleaner cobra semantics, but contradicts the spec's positional-id choice and adds a surface the user declined.

**Decision**: Option 1. `roles` registers once (guard + explicit `main` wiring) with `MaximumNArgs(1)`; RunE dispatches on `len(args)`. More than one positional id is rejected by the `Args` validator (usage error, no API call). Filters are valid only on the list branch; `--include` only on the single branch — each cross-combination (`filter + id`, `--include` without id) is rejected as a fail-fast usage error in RunE before assembly, with a transport tripwire asserting nothing was sent (011/013 precedent).

**Consequences**: The positional-id shape forecloses future `glassfrog roles <id> <subcommand>` (e.g. a child `domains`) — cobra can't tell a role id from a subcommand name. The downstream per-role specs (#33/#34/#38) must therefore hang their reads off a different surface (e.g. a singular `role` group, or flags), not as children of `roles`. Recorded as a risk, not solved here.

### ADR-2: Grow the shared `Role`; add `RoleDetail` (embeds `Role` + optional related resources) in `internal/glassfrog`

**Context**: `GET /roles` returns `Role`; `GET /roles/{id}` returns `RoleDetail` = `Role` + optional `assignments`/`subroles`/`parent_role`/`policies`/`notes`/`skills` (spec `allOf`). 011 ADR-1 mandates shared schema types, grown not duplicated; 012 already grew `Role` with purpose/accountabilities/domains.

**Options considered**:
1. **Grow `Role` to full spec shape + add `RoleDetail` embedding it** — one canonical role type; the detail view is a superset struct with pointer/slice fields that stay nil/empty when not `?include`d.
2. **Separate `ListRole` and `DetailRole` types** — avoids optional fields, but forks the role model and violates 011 ADR-1 ("grow the SAME type, never a second").

**Decision**: Option 1. `Role` gains the remaining spec fields; `RoleDetail struct { Role; Assignments []Assignment; Subroles []Role; ParentRole *Role; Policies []Policy; Notes []Note; Skills []SkillSummary }`. Nested `subroles`/`parent_role` are plain `Role` (not `RoleDetail`), so there is no recursion. Decoding stays tolerant of unknown/extra fields (011). New related models (`Assignment`, `Policy`, `Note`, `SkillSummary`) are minimal and live in `glassfrog` for reuse by the downstream per-role specs.

**Consequences**: Related-resource fields are absent unless requested — the render templates guard each with an explicit-absence marker / omission (019 `missingkey=error` + `{{if}}` pattern), never invent a value. The downstream specs (#33/#34/#38) reuse `Policy`/`Assignment`/etc. rather than redefining them — Role Reads sets that precedent.

### ADR-3: Default list walks to completion via `paging.All`; the `--first-page` opt-out does a single-page read — both reduce to `(records, complete)`

**Context**: Spec clarification C: walk by default, expose an opt-out flag that limits to the first page and signals "more exist". 016 ships `paging.All[T]` (complete walk, `Result[T]{Records,Complete,Stop}`, invariant `Complete == (Stop==nil)`) and left a `WithMaxPages` option open. The landed `/me*` reads (012–014) already implement first-page + `has_next_page` signal.

**Options considered**:
1. **Two branches: default `paging.All`, opt-out single `Execute`** — opt-out decodes `Page[Role]`, takes `Data` + `!HasNextPage`; default takes `Result.Records` + `Result.Complete`. Reuses both landed patterns; leaves 016's `Result` invariant untouched.
2. **`paging.WithMaxPages(1)`** — one code path. But a cap-stop would need `Complete=false` with `Stop=nil`, breaking 016's `Complete == (Stop==nil)` invariant (a cap is not an error). Rejected to avoid mutating the walker's contract for a consumer convenience.

**Decision**: Option 1. Both branches yield `(records []Role, complete bool)`, which `renderResult` renders identically; an incomplete result writes one explicit signal to **stderr**. Exit code distinguishes the two ways a list can be incomplete:
- **Deliberate opt-out** (`--first-page`, more pages exist) → exit **0** with a "more available — re-run without --first-page to fetch all" note. Not an error; the operator chose the boundary.
- **Mid-walk failure** (default walk, `Result.Stop != nil` — a 010 transport/API error or `MalformedPageError`) → render the partial `Result.Records`, write an explicit "incomplete — <cause>" note to stderr, and exit **non-zero** via `classifyClientError(Stop)`. Honors the spec ("produces the roles gathered so far, flagged incomplete with the cause") while failing loudly because an error stopped the walk.

**Consequences**: A partial set is never silently presented as whole (CONSTITUTION VI) in either case. The opt-out flag name (`--first-page` provisional) is interface-level. `--per-page` (016) is also available for the walk; whether `roles` surfaces it is an interface detail.

### ADR-4: Validate `--include` locally (closed enum); pass a role id straight through to the API

**Context**: `--include` is a closed enum (`assignments,subroles,parent_role,policies,notes,skills`); a bad value would be silently ignored by the API, returning the role *without* the requested embed (silent-wrong-results — exactly 013's `validateStatus` rationale). A role id is a free identifier; `getRole` documents only `401`/`404` (no `400`), so a malformed/unknown id yields a clean not-found, not silent-wrong-results.

**Options considered**:
1. **Validate include locally, pass id through** — `validateRolesInclude([]string)` rejects unknown include values as a fail-fast `UsageError(2)` (naming the value + supported set, transport tripwire); a bad id is left to the API's `404`.
2. **Validate both locally** (regex the `^role_[0-9a-f]{32}$` id) — symmetric, but adds a maintenance burden and a second failure shape for a case the API already reports cleanly; diverges from the "validate where the API would otherwise silently mislead" principle.

**Decision**: Option 1. Include validation reuses 011's `validateInclude` shape (pure, pre-request). The id is passed through; an unknown id surfaces as the API's non-2xx → `APIError(3)` (or `PermissionError(4)` on 401/403) via the shared classifier, matching the spec's "unknown id → names the HTTP status" scenario.

**Consequences**: Asymmetry is deliberate and reasoned (closed-enum-with-silent-failure → validate; clean-404 identifier → pass through). If the API later documents a `400` for malformed ids, local id validation can be added without disturbing this design.

---

## Cross-cutting Concerns

**Error handling**: All failures flow through the single landed path — `classifyClientError` (011, widened by 015) maps 010's typed errors to the frozen `Outcome`/`ExitCode` registry (auth fail-safe → `UsageError(2)`/`RuntimeError(1)`; transport → `NetworkUnavailable(6)`; non-2xx → `APIError(3)`/`PermissionError(4)`/`RateLimited(5)`; `*output.FormatError` → `UsageError(2)`). Role Reads adds **no** new category or code. The token never appears in output or errors (secret hygiene).

**Output**: `renderResult[T]` (020) owns format dispatch; structured formats decode `json.RawMessage` (018 ADR-2, raw bytes verbatim), human formats render the typed struct via `internal/render`. The list and single read are two new resource keys in the render registry.

**Configuration**: `--base-url` (011) and `--output`/`-o` (020) are inherited persistent root flags. Filter flags, `--include`, the first-page opt-out, and (optionally) `--per-page` are local to `roles`. Page size defaults to the API max (016).

**Testing**: Pure `run`/`validate` functions are unit-tested offline behind the injected seam (fake `Executor` returning canned pages, including a multi-page walk and a mid-walk error); a transport tripwire asserts no request on any fail-fast rejection. Command behavior is a godog suite against a feature file. Render output is golden/unit-tested with the registry exhaustiveness guard (PR #10 `len`+comma-ok shape).

---

## Implementation Strategy

Buildable now — all dependencies are landed (007, 009, 010, 015, 016, 017, 018, 019, 020). One feature, naturally three phases by dependency:

1. **Schema** — grow `glassfrog.Role` to the full spec shape; add `RoleDetail` + the `Assignment`/`Policy`/`Note`/`SkillSummary` leaf models and the `{data: RoleDetail}` document wrapper. No command yet. (Foundation for both reads.)
2. **List read** — `roles` command (no-id branch): filter validation, `paging.All[Role]` walk, first-page opt-out, completeness signalling, render dispatch, exit-code routing. Registration + `main` wiring. godog + unit tests.
3. **Single read** — the `<id>` branch: `--include` validation, `GET /roles/{id}` → `RoleDetail`, related-resource rendering. Extends the same command and test suite.

Phases 2 and 3 both depend on phase 1; 3 builds on 2's command scaffold. The tasks skill decomposes these into PR-sized units.

---

## Risks

- **Positional-id forecloses role subcommands** (ADR-1). The downstream per-role specs (#33/#34/#38) cannot be children of `roles`. *Likelihood: certain to surface. Impact: moderate (a naming decision for those specs). Mitigation: flag now so they choose a singular `role <id> domains` group or a flag-based surface; Role Reads stays the org-wide list+single only.*
- **Embedded `--include` vs standalone reads overlap** (spec non-behavior). `--include=policies` embeds policies inline; Role Policies (#34) will own the standalone `GET /policies/{id}` read. *Likelihood: design-time confusion. Impact: low if the boundary is honored. Mitigation: the spec records the two-views-coexist boundary; the downstream Shaper reads it before adding standalone reads.*
- **`Q` free-text search not exposed**. `GET /roles` also accepts a `q` search parameter not in this spec's filter set. *Likelihood: a user may expect search. Impact: low. Mitigation: deferred deliberately — note it as a future filter; not a behavioral gap (the spec scoped to the four structural filters).*
- **Skills embed is summary-only**. `?include=skills` returns `SkillSummary` (no `content`); full content needs `GET /skills/{id}`. *Likelihood: certain. Impact: low. Mitigation: model it as `SkillSummary` and let the render template reflect "summary"; do not imply full content is present.*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — exact flag names/spellings (`--first-page`, filter flags, `--include` form), command help strings, the `{data: RoleDetail}` wrapper symbol names, and the render template field layout. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **Downstream per-role reads** (#33/#34/#38) and **Organization Tree** (#26) — separate specs; this plan only sets the schema and boundary precedent they consume.
