# Plan: Organization Tree

**Feature**: 026-organization-tree
**Role**: Shaper
**Inputs**: spec.md (026), PROJECT.md, `.score/memory/DECISIONS.md` (50+ entries), DEPRECATION.md, LEARNINGS.md (passive), spec/glassfrog-api-v5.yaml (`getOrgTree`/`getRoleTree`/`listSubroles`/`TreeNode`/`Role`), and the **current codebase state** (025 is *Analyzed*, not implemented — `RoleDetail` and the leaf models do not yet exist; `internal/cli/roles.go` is still the registration stub).

---

## System Architecture

Organization Tree reads the circle hierarchy three ways and otherwise **composes the landed read stack** — it builds no new transport, identity, pagination, error-classification, or output machinery. It adds **two new cobra commands** (`tree`, `subroles`) and **one new schema type** (`TreeNode`); the subroles read shares the role-detail schema with Role Reads (025) under the first-to-land-creates rule.

The defining shape is that the API draws **two completeness models** the design must honor distinctly:
- The two **tree reads** (`GET /tree`, `GET /roles/{id}/tree`) return one **recursive `TreeNode`** in a single, **unpaginated** response — bounded only by an optional `--depth`. They do not touch Pagination (016).
- The **subroles read** (`GET /roles/{id}/subroles`) returns a **paginated flat list** of roles and reuses 025's walk-by-default + first-page-opt-out machinery verbatim.

**Components and how they connect**:

- **`internal/cli` — the `tree` command** (new `tree.go` + godog suite). A guard-registered (001), explicitly-wired (001) runnable leaf taking an *optional* positional id (`MaximumNArgs(1)`, the 025 ADR-1 shape). A thin cobra `RunE` over an injected seam (011 ADR-5) branches on `len(args)`: 0 args → `GET /tree` (`getOrgTree`, whole org), 1 arg → `GET /roles/{id}/tree` (`getRoleTree`, rooted subtree). Both validate `--depth` and `--include` (tree set) first, assemble the connection context, issue **one** `Execute` into a `{data: TreeNode}` document wrapper, and render through `renderResult` (020).
- **`internal/cli` — the `subroles` command** (new `subroles.go`). A guard-registered runnable leaf taking a **required** id (`ExactArgs(1)`), `GET /roles/{id}/subroles`. Validates `--include` (subroles set), walks the list to completion via the shared walker (016), and renders. It is a *sibling* of `tree`, not a child of `roles` — DECISIONS 2026-06-08 (025 ADR-1) records that the positional-id `roles` shape forecloses `roles <id> <subcommand>`.
- **`internal/glassfrog` — `TreeNode`** (new). A recursive struct decoded from the `{data: TreeNode}` wrapper, reusing the existing `Accountability` / `Domain` / `Actor` projections for the optional `?include` fields. Distinct from `Role`/`RoleDetail` (it carries `children`, a constrained `flags` enum, and `type: role`). **Not paginated.**
- **`internal/glassfrog` — shared role-detail schema** (025-coordinated). The subroles read decodes `Page[RoleDetail]` so per-child `?include` embeds land inline. `RoleDetail` (embeds the grown `Role` + `Assignment`/`Policy`/`Note`/`SkillSummary`) is the **same** schema 025 designs; whichever of 025/026 implements first creates it in `glassfrog`, the other reuses (005/006/016 first-to-land pattern). The tree reads need none of it.
- **`internal/paging` — the walker** (016). The subroles default path calls `paging.All[RoleDetail]` over the retrying `Executor` (017). The tree reads do not use it.
- **`internal/render` — templates** (019). Adds **two new resource keys** — `tree` (nested) and `subroles` (list) — each with full+compact templates, registered into the embedded set and covered by the exhaustiveness guard (PR #10 shape). The `tree` template renders the recursive node; it is distinct from the shipped `roles` key and from 025's planned `org-roles`/`role` keys.

**Data flow** (whole-org tree): `tree` → validate depth+include → `AssembleFromOS(--base-url)` (009) → `NewClientFromOS` (010) → `RetryExecutor` (017) → one `Execute` into `{data: TreeNode}` → `renderResult` → stdout. Subroles is the 025 list flow with `RoleDetail` items: `paging.All[RoleDetail]` → `Result[RoleDetail]` → `renderResult` (+ incompleteness note to stderr).

---

## Architecture Decisions

### ADR-1: `tree` (optional positional id) and `subroles <id>` are two new sibling commands, not children of `roles`

**Context**: The capability needs a CLI surface for three reads. DECISIONS 2026-06-08 (025 ADR-1) establishes that the `roles` command takes an optional positional id, which makes `glassfrog roles <id> <subcommand>` impossible — cobra can't distinguish a role id from a subcommand name. The spec assumed `glassfrog tree` / `glassfrog tree <id>` / `glassfrog subroles <id>`.

**Options considered**:
1. **Two sibling leaves: `tree` (`MaximumNArgs(1)`) + `subroles` (`ExactArgs(1)`)** — `tree` mirrors 025's optional-positional-id pattern (0 → whole org, 1 → rooted subtree); `subroles` requires the parent id the endpoint requires. Two top-level commands, two help entries.
2. **A singular `role <id> tree` / `role <id> subroles` group** — collects the per-role reads under one `role` parent. But it splits the whole-org tree (`GET /tree`, no id) awkwardly from the rooted tree, and pre-commits the `role` group that the downstream per-role specs (#33/#34/#38) should decide; that group is out of this spec's scope.
3. **A `tree` command with a `--subroles` flag** — one command, mode flag. Conflates a recursive tree read with a flat paginated list read whose flags (`--depth` vs pagination) don't overlap — two behaviors wearing one command.

**Decision**: Option 1. `tree` registers once (001 guard + explicit `main` wiring) with `MaximumNArgs(1)` and branches in `RunE` on `len(args)`; `subroles` registers as `ExactArgs(1)`. `--depth` is valid only on `tree`; `--include` is validated against the running read's own set. Each cross-combination (`--depth` on subroles, `>1` positional, an unknown `--include`) is a fail-fast `UsageError(2)` before assembly, with a transport tripwire asserting no request was sent (011/013/025 precedent).

**Consequences**: Two more top-level commands join the tree. The exact command spelling stays interface-level (the spec flagged it `[ASSUMED]`), but the component split — a tree leaf and a subroles leaf, both siblings of `roles` — is fixed here. This does **not** create the `role` group; the downstream per-role specs remain free to choose their own surface.

### ADR-2: `TreeNode` is a new recursive type in `internal/glassfrog`, decoded from a `{data: TreeNode}` wrapper; it is not `Role`

**Context**: `getOrgTree`/`getRoleTree` return a single `TreeNode` (spec §5571): `id`, `type`, `name`, `purpose`, `parent_role_id`, `has_subroles`, `flags` (enum `structural`/`elected`/`linked`), `children []TreeNode`, plus optional `accountabilities`/`domains`/`fillers` gated behind `?include`. This is a different shape from `Role`/`RoleDetail` — recursion via `children`, a constrained flags enum, no `tags`. 011 ADR-1 mandates shared schema types in `glassfrog`, grown not duplicated — but only when the *shape* is the same.

**Options considered**:
1. **A dedicated recursive `TreeNode` struct** — `children []TreeNode` (the recursion is the endpoint's whole point); optional `Accountabilities`/`Domains`/`Fillers` reusing the existing leaf projections, nil unless `?include`d. Decoded from a `{data: TreeNode}` document wrapper (the 025 single-read shape), tolerant of unknown fields (011).
2. **Reuse/grow `RoleDetail` with a `Children` field** — one role type. But `RoleDetail.Subroles` is `[]Role` (flat, no recursion by 025 ADR-2's explicit no-recursion decision), the tree's `flags` enum and `type` field aren't on `Role`, and forcing a recursive `Children` onto `RoleDetail` would muddy both models.

**Decision**: Option 1. `TreeNode` is its own type in `glassfrog` (schema-only, no transport/cobra). `Children []TreeNode` is the recursion. The optional include fields reuse the existing `Accountability`/`Domain`/`Actor` projections (extend them only if the tree node carries fields the current minimal projections drop and the render wants — an interface-level call). The tree reads decode `{data: TreeNode}`; **no pagination envelope**.

**Consequences**: The recursive type drives a recursive (or depth-indented) human render template — new ground for `internal/render` (every prior template renders a flat or one-level-nested record). Structured output (json/yaml) is unaffected: it decodes `json.RawMessage` and serializes the raw bytes verbatim (018 ADR-2), so the nesting needs no special handling on the machine path. `parent_role_id` is retained on every node so a consumer that flattens the tree keeps the edge.

### ADR-3: The subroles read reuses 025's pagination + first-page-opt-out verbatim and shares the `RoleDetail` schema under first-to-land-creates

**Context**: `listSubroles` returns a paginated `{data: [Role], meta: {pagination}}` with per-child `?include` (`assignments`/`subroles`/`parent_role`/`policies`/`notes`/`skills` — exactly `getRole`'s set). CONSTITUTION VI requires walk-to-completion-or-signal. 025 (ADR-3) already designed `paging.All` + a `--first-page` opt-out for the role list, and (ADR-2) the `RoleDetail` schema + `Assignment`/`Policy`/`Note`/`SkillSummary` leaf models — but 025 is *Analyzed*, not landed, so none of it exists in code yet.

**Options considered**:
1. **Decode `Page[RoleDetail]`, reuse the 025 walk + opt-out shape; create the shared schema if 026 lands first** — subroles embeds land inline on each child; the walk, the opt-out exit-code split, and the `RoleDetail`/leaf models are identical to 025's, so they are created once (whichever spec implements first) and reused.
2. **Decode `Page[Role]` and scope subroles `--include` down to nothing (or only `subroles`)** — avoids the `RoleDetail` dependency, but contradicts the spec's behavioral accord (which committed to the full include set) and would silently drop a confirmed capability.
3. **Define a 026-local subroles-detail type** — avoids coupling to 025, but forks the role-detail model and violates 011 ADR-1 / 025 ADR-2 ("grow the SAME type, never a second").

**Decision**: Option 1. The subroles list decodes `Page[RoleDetail]`. Completeness reuses 025 ADR-3 exactly: default `paging.All[RoleDetail]` walk → `(records, complete)`; a `--first-page` opt-out does one `Execute` into `Page[RoleDetail]` (`Data`/`!HasNextPage`); a deliberate opt-out with more pages exits **0** with a "more available" stderr note, a mid-walk failure renders the partial set + an "incomplete — <cause>" stderr note and exits **non-zero** via `classifyClientError(Stop)`. `RoleDetail` + the leaf models + the `Role` growth are the **shared 025 schema**: first-to-land creates them in `glassfrog`, the other reuses (005/006/007/016 pattern).

**Consequences**: A real coordination point with 025 if both run in parallel sessions — the `RoleDetail`/leaf-model/`Role`-growth definitions must exist exactly once. This is a shared-schema coordination, **not** a hard dependency edge (the BACKLOG declares only 007/010; the tree reads are fully independent). The opt-out flag name (`--first-page`, provisional) and `--per-page` exposure are interface-level, shared with 025.

### ADR-4: Validate `--include` (per-read closed set) and `--depth` locally; pass the role id straight through

**Context**: The tree endpoints **silently ignore** unknown `?include` values (spec §647) — the exact silent-wrong-results hazard 013/025 validate against. The tree and subroles reads expose **different** include sets. `--depth` is a bounded integer (`minimum: 0`, where `0` = root node alone, a meaningful distinct value from "omitted = full tree"); the tree endpoints document a `400` for a bad request. A role id is a free identifier the endpoints answer with a clean `404`.

**Options considered**:
1. **Reject-unknown `--include` per-read-set + validate `--depth >= 0` locally; pass the id through** — `validateTreeInclude` checks `{accountabilities, domains, members}`, `validateSubrolesInclude` checks `{assignments, subroles, parent_role, policies, notes, skills}`, both reusing 011's pure fail-fast `validateInclude` shape; `--depth` is an *optional* int (sent only when set, so `0` ≠ absent), rejected locally if negative; the id is left to the API's `404`.
2. **Pass `--include` through (mirror the API's silent-ignore)** — one fewer validator, but a typo'd include returns a tree *without* the embed and no error (silent-wrong-results) — exactly what DECISIONS 2026-06-08 (025 ADR-4) rejects.
3. **One shared include validator across both reads** — simpler, but it would accept tree-only values on subroles (and vice versa), failing the API silently for the values that don't apply.

**Decision**: Option 1. Two include validators, one per read, each reject-unknown as `UsageError(2)` naming the bad value and that read's supported set, with a transport tripwire. `--depth` is an optional integer flag — un-set means "omit `depth`" (full tree), set means send `depth=<n>`; a negative value is rejected locally (cheap, avoids a wasted call) while the API's own bounds back-stop anything else. The role id is passed through; an unknown id surfaces as the API's non-2xx → `APIError(3)` / `PermissionError(4)` via the shared classifier.

**Consequences**: The CLI is deliberately stricter than the tree endpoints (rejecting includes they would silently ignore) — friendlier and consistent with 011/025. The `0`-vs-absent depth distinction makes the flag optional (a pointer/`Changed()` check), a small but load-bearing interface detail recorded here. If the API later documents a `400` for malformed ids, local id validation can be added without disturbing this design (025 ADR-4's same note).

---

## Cross-cutting Concerns

**Error handling**: All failures flow through the single landed path — `classifyClientError` (011, widened by 015) maps 010's typed errors to the frozen `Outcome`/`ExitCode` registry (auth fail-safe → `UsageError(2)`/`RuntimeError(1)`; transport → `NetworkUnavailable(6)`; non-2xx → `APIError(3)`/`PermissionError(4)`/`RateLimited(5)`; `*output.FormatError` → `UsageError(2)`; render error → `RuntimeError(1)`). Organization Tree adds **no** new category or code. The token never appears in output or errors (secret hygiene).

**No conditional-request caching (spec non-behavior)**: the tree endpoints offer `ETag`/`If-None-Match` → `304`. The CLI issues a **plain GET** each time and sends no `If-None-Match`, so a `304` is never elicited and needs no handling. Caching is deliberately out of scope (latency optimization, not a faithful first read surface).

**Output**: `renderResult[T]` (020) owns format dispatch; structured formats decode `json.RawMessage` and serialize raw bytes verbatim (018 ADR-2 — the recursive tree needs no special machine-path handling), human formats render the typed struct via `internal/render`. The `tree` and `subroles` resource keys are new, distinct from the shipped `roles` key (`me roles`) and from 025's planned `org-roles`/`role` keys.

**Configuration**: `--base-url` (011) and `--output`/`-o` (020) are inherited persistent root flags. `--depth`, `--include`, and the subroles first-page opt-out (+ optionally `--per-page`) are local to their commands. Subroles page size defaults to the API max (016).

**Testing**: Pure `run`/`validate` functions are unit-tested offline behind the injected seam (fake `Executor` returning canned trees and multi-page subrole sets, including a mid-walk error and a `--depth 0` single node); a transport tripwire asserts no request on any fail-fast rejection. Command behavior is a godog suite against a feature file. Render output is golden/unit-tested with the registry exhaustiveness guard (PR #10 `len`+comma-ok shape), including a recursive `tree` render and a leaf (empty `children`) node.

---

## Implementation Strategy

Buildable now for the tree half — all hard dependencies are landed (007, 009, 010, 015, 016, 017, 018, 019, 020). Three phases by dependency:

1. **Schema** — add the recursive `TreeNode` + the `{data: TreeNode}` document wrapper to `internal/glassfrog`. Ensure the subroles-shared `RoleDetail` + `Assignment`/`Policy`/`Note`/`SkillSummary` leaf models + the `Role` growth exist (create them here if 025 has not landed; reuse if it has). No command yet. (Foundation for both reads.)
2. **Tree reads** — the `tree` command (optional-id branch): depth + include validation, `GET /tree` and `GET /roles/{id}/tree`, decode `TreeNode`, the new recursive `tree` render templates, render dispatch, exit-code routing. Registration + `main` wiring. godog + unit tests.
3. **Subroles read** — the `subroles <id>` command: include validation, `paging.All[RoleDetail]` walk + first-page opt-out + completeness signalling, the `subroles` render templates, render dispatch. godog + unit tests.

Phases 2 and 3 both depend on phase 1; they are otherwise independent (different commands, different completeness models) and could be split across PRs in either order. The tasks skill decomposes these into PR-sized units.

---

## Risks

- **Shared-schema coordination with 025** (ADR-3). `RoleDetail`, the leaf models, and the `Role` growth must exist exactly once. *Likelihood: high if 025 and 026 run in parallel. Impact: moderate (a compile-time collision or a drifted second type). Mitigation: first-to-land-creates; the later spec reuses verbatim and adds nothing. The tree reads carry no such dependency.*
- **Recursive human render is new** (ADR-2). Every existing `internal/render` template is flat or one-level-nested; a `TreeNode` needs recursion or depth-indentation with absence markers and a sensible compact form. *Likelihood: certain. Impact: moderate (template design effort). Mitigation: flag it for interface/scenarios; structured output is unaffected.*
- **`--depth 0` vs. omitted** (ADR-4). `0` (root only) is a valid distinct value, so the flag must be optional (not a plain int defaulting to 0). *Likelihood: certain to surface at interface time. Impact: low if modeled as optional. Mitigation: recorded here; the run path sends `depth` only when the flag was set.*
- **Large unpaginated trees, no caching** (cross-cutting). A big org's full tree is one large response, re-fetched on every call with no `304`. *Likelihood: low-moderate for large orgs. Impact: low (latency, not correctness). Mitigation: `--depth` bounds size; ETag/304 is a deliberate deferred non-behavior, revisitable when there is demand.*
- **`Q` free-text search on subroles not exposed**. `listSubroles` also accepts a `q` parameter, not in this spec. *Likelihood: a user may expect search. Impact: low. Mitigation: deferred deliberately, same as 025's `Q` note; not a behavioral gap.*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — exact command spellings (`tree` / `subroles`), flag names/spellings (`--depth`, `--include` form, the first-page opt-out), help strings, the `{data: TreeNode}` wrapper symbol names, and the recursive render template's field layout / indentation. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **The downstream per-role reads** (#33/#34/#38) and **the `role` group surface** — separate specs; this plan only consumes the 025 schema/boundary precedent and adds `tree`/`subroles` as siblings of `roles`.
