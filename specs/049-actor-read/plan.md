# Plan: Actor Read

**Feature**: 049-actor-read
**Role**: Shaper
**Inputs**: spec.md (049), PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 011 ADR-1/ADR-5, 015, 016, 019, 020, **025 ADR-1/ADR-2/ADR-4**, 026, **048 ADR-1/ADR-2/ADR-4**), DEPRECATION.md, LEARNINGS.md (passive), spec/glassfrog-api-v5.yaml (`getActor` §855, `Actor` §6226), and the **current codebase state** — the landed read stack: `glassfrog.Actor` (`internal/glassfrog/me.go:30`), `glassfrog.Role`/`RoleDetail`/`Assignment` (`internal/glassfrog/roles.go:21/52/76`), the 048 `actors` command (`internal/cli/actors.go`, `Args: cobra.NoArgs`), `ResourceActors` + the `ResourceRole`/`ResourceDomain`/`ResourcePolicy` singular-detail render keys (`internal/render/render.go`), `classifyClientError` (`internal/cli/clienterror.go`), the `include.go` reject-unknown validator, `AssembleFromOS`/`NewClientFromOS`/`Execute` (`internal/apiclient`), and the render `Resource` registry with its exhaustiveness guard.

---

## System Architecture

Actor Read adds the **single-actor drill-in** to the `actors` command and otherwise **composes the landed read stack** — it builds no new transport, identity, error-classification, or output machinery, and (unlike the 048 directory) needs **no pagination** because `GET /actors/{id}` returns one resource. It is the actor analogue of Role Reads' single-read half (025): the same noun command serves both an org-wide list (the directory, 048) and a by-id read (this spec), branching on `len(args)`. What is genuinely new is small and bounded: **the `actors` command grows an optional positional id** (the 025 `roles <id>` shape, superseding 048's `cobra.NoArgs`), **one new schema type** (`ActorDetail`, embedding the landed `Actor` plus the optional `roles`/`assignments` embeds — the 025 `RoleDetail` shape), **one new render key** (`ResourceActor`, singular — the single-actor + footprint projection), and **one new local `--include` validator** over the `{roles, assignments}` set.

The defining shape is that this read **drills into an id** rather than walking a list: it issues exactly one `Execute` into a `{data: ActorDetail}` document, and the `?include=roles,assignments` embeds (the actor's governance footprint — the roles they fill with their accountabilities/domains/purpose, and the assignments) arrive inline in that single response. The directory (048) finds the `per_`/`agt_` id; Actor Read is what that id feeds. The second defining decision is that the footprint embed (`--include`) is the convenience view; the standalone, paginated list of the roles an actor fills is Actor Assignments (050) — and because the optional-positional command shape forecloses `actors <id> <subcommand>` (025 ADR-1), 050 takes a flag/separate surface, not `actors <id> assignments`.

**Components and how they connect**:

- **`internal/cli` — the `actors` command, grown** (`actors.go` + its godog suite, extended). The command's `Args` changes from `cobra.NoArgs` (048) to `cobra.MaximumNArgs(1)`, and its `RunE` branches on `len(args)`: **0 args → the directory list** (048's behavior, unchanged), **1 arg → the single read** (this spec). The single-read branch resolves `--output` first (020), validates `--include` (reject-unknown over `{roles, assignments}`, transport tripwire), assembles the connection context, issues one `Execute` of `GET /actors/{id}?include=…` into a `{data: ActorDetail}` document, and renders through the single-resource path (020). Filters (`--kind`/`--role-id`/`--query`) stay **list-only** and `--include` stays **single-only**; each cross-combo (a filter with an id, `--include` without an id, more than one positional) is a fail-fast `UsageError(2)` before assembly with a transport tripwire (the 025 ADR-1 discipline).
- **`internal/glassfrog` — `ActorDetail`** (new, the 025 `RoleDetail` shape). `ActorDetail struct { Actor; Roles []Role; Assignments []Assignment }` embeds the landed `Actor` and adds the two optional embed slices, nil/empty unless `?include`d. The single read decodes the `{data: ActorDetail}` document wrapper; the embedded `Role`/`Assignment` are the **full landed types** (025), not new leaf models. `Actor` itself is unchanged (011 ADR-1 — grow the schema, don't duplicate; the embed fields live on the detail type exactly as `RoleDetail` carries Role's related resources).
- **`internal/cli` — `--include` validator** (new per-read set, the `include.go` reject-unknown shape). Reject-unknown against the closed 2-value set (`roles`, `assignments`) as `UsageError(2)` before any request, with a transport tripwire. This is a **per-read set distinct from the role `--include` set** (026's two-validator precedent) — never a shared set that would accept cross-read values the API silently drops.
- **`internal/render` — templates** (019). Adds **one new resource key** — `ResourceActor` (singular, the single-actor + footprint detail) — with `actor.full`/`actor.compact` templates, registered into `builtinResources` and covered by the exhaustiveness guard (PR #10 `len`+comma-ok shape). It renders the actor's `kind` badge + `name` + `id`, and — when embedded — the footprint: each role's name/purpose/accountabilities/domains, and the assignments. Each embed is guarded by an explicit-absence marker (019) — present only when `?include`d. This is distinct from `ResourceMe` (the `me` document: actor + org + membership) and `ResourceActors` (048's flat directory list).
- **`internal/apiclient` — `Execute`** (reused). One `Execute` into the `{data: ActorDetail}` target; no walker, no `Page[T]` (this is a single resource).

**Data flow**: `actors <id> [--include]` → `resolveFormat(--output)` (020) → `validateActorInclude(--include)` → `AssembleFromOS(--base-url)` (009) → `NewClientFromOS` (010) → `Execute(GET /actors/{id}?include=…)` into `{data: ActorDetail}` (010) → `Result[ActorDetail]` → single-resource render (`aggregateRawData`/raw-bytes for `json`/`yaml`; the `actor` projection for `full`/`compact`) → stdout. Failures route through the shared `classifyClientError` chain (011/015) → 004 exit code. A `404` for an unknown id is a clean non-2xx (no local id validation — 025 ADR-4).

---

## Architecture Decisions

### ADR-1: Grow `actors` to an optional positional id (`MaximumNArgs(1)`) — the single-command 025 shape — superseding 048 ADR-1's `cobra.NoArgs`

**Context**: `GET /actors/{id}` reads one actor by a `per_`/`agt_` id — the by-id drill-in the directory (048) defers to this spec. 048 ADR-1 registered `actors` with `cobra.NoArgs` (the directory's subject is the whole org, narrowed only by filters) and rejected an optional positional — but the positional it considered and rejected was a **free-text name/query** (which belongs on `--query`/`-q`), *not* an id subject; 048 explicitly left the single by-id read to 049. Role Reads (025 ADR-1) already set the precedent for exactly this situation: one runnable command with an optional positional id (0 args → org-wide list, 1 arg → single read).

**Options considered**:
1. **Grow `actors` to `cobra.MaximumNArgs(1)`; `RunE` branches on `len(args)` (0 → directory, 1 → single read).** Silent conformance to 025 ADR-1; one noun, one command, the directory and drill-in on the same surface. Supersedes the `NoArgs` half of 048 ADR-1.
2. **A separate singular `actor <id>` command** alongside the `actors` directory. Rejected: forks the noun surface (`actor` vs `actors`) and breaks the `roles`/`roles <id>` symmetry 025 deliberately established; operators would have to remember a singular/plural split that no other read in the CLI carries.
3. **Leave `actors` at `NoArgs`; host the single read elsewhere** (e.g. a `--id` flag on the directory). Rejected: a by-id read whose subject is a path id is a positional-subject read (the `roles <id>`/`subroles <id>` shape), not a filter; modelling it as a flag would contradict every other single-resource read.

**Decision**: Option 1. `actors` registers once (001 guard + explicit `main` wiring) with `Args: cobra.MaximumNArgs(1)`; `RunE` branches on `len(args)`. The 0-arg directory branch (048) is unchanged; the 1-arg branch is the single read. Filters are list-only, `--include` is single-only; each cross-combo is a fail-fast `UsageError(2)` before assembly with a transport tripwire. This is an **announced divergence** from 048 ADR-1's `cobra.NoArgs` sub-decision (the directory was specified before its by-id sibling; the two were always destined for one command per 025). The directory's *behavior* on 0 args is preserved exactly — only the arg validator widens.

**Consequences**: The `actors.go` arg validator and `RunE` are edited (not rewritten) — a bounded growth of an **already-landed** command. 048 is merged on `main` (`feat(048): Actor Directory — glassfrog actors` #90; `internal/cli/actors.go` ships with `Args: cobra.NoArgs` + `runActorsList`), so 049 simply grows that landed command — no first-to-land race, just an edit against current `main`. **Foreclosure (025 ADR-1)**: cobra cannot distinguish an actor id from a subcommand name under an optional positional, so `actors <id> <subcommand>` is impossible — Actor Assignments (050, `GET /actors/{id}/assignments`) therefore takes a flag-based or separate-command surface, **not** `actors <id> assignments`. 050's standalone read is distinct from this spec's `--include assignments` embed. Candidate for `/score:deprecate` to retire 048 ADR-1's `cobra.NoArgs` sub-decision.

### ADR-2: Add `ActorDetail` (embeds `Actor` + optional `Roles`/`Assignments`) — the 025 `RoleDetail` shape; reuse the full `Role`/`Assignment`

**Context**: `getActor` returns the `Actor` schema (§6226) with optional `roles []Role` and `assignments []Assignment` arrays, present only when `?include`d. The landed `glassfrog.Actor` (011, reused as-is by 048) carries `id`/`name`/`kind`/timestamps but no embed fields (048 ADR-2 deliberately left them to this spec). The full `Role` shape and the `Assignment` leaf model are already landed (025, `roles.go`). 011 ADR-1: one shared schema type, grown not duplicated. 025 ADR-2: `RoleDetail embeds Role + optional related-resource fields`; the single read decodes a `{data: RoleDetail}` document wrapper while the list decodes `Page[Role]`.

**Options considered**:
1. **Add the embed fields directly onto `Actor`.** Rejected: the directory list (048) and `me` (011) decode the bare `Actor`; hanging optional embed slices on the base type spreads single-read concerns onto every actor consumer — the exact split 025 ADR-2 avoided by introducing `RoleDetail` over `Role`.
2. **`ActorDetail struct { Actor; Roles []Role; Assignments []Assignment }`**, decoded from `{data: ActorDetail}` on the single read; the list keeps decoding `Page[Actor]`. Chosen — silent conformance to 025 ADR-2 (the `Role`/`RoleDetail` precedent) and 011 ADR-1.

**Decision**: Option 2. Add `ActorDetail` to `internal/glassfrog`, embedding the unchanged `Actor` and adding `Roles []Role` and `Assignments []Assignment` (the full landed types — no new leaf models). The single read decodes the `{data: ActorDetail}` document wrapper; the embed slices stay nil/empty unless `?include`d (the render guards each with an explicit-absence marker — 019). `Actor`, `Role`, and `Assignment` are unchanged.

**Consequences**: A small additive type, exactly parallel to `RoleDetail`. The footprint is assembled entirely from already-landed types — the embedded roles carry their purpose/accountabilities/domains because `Role` is already the full shape (025). No schema growth of `Actor`/`Role`/`Assignment`. The exact Go field/symbol names are interface-level.

### ADR-3: Validate `--include` locally against `{roles, assignments}`; pass the actor id through to a clean `404`

**Context**: 025 ADR-4 set the input-handling principle — *validate closed-enum inputs locally* (where a wrong value makes the API silently mislead), but *pass free identifiers through* (where the API reports cleanly). `getActor`'s `include` is a **closed enum** (`roles`, `assignments`) the API silently ignores when wrong (returns the actor without the embed — the silent-wrong-results hazard). The actor `id` is a free identifier; `getActor` documents only `401`/`404` (no `400`), so a malformed/unknown id yields a clean not-found.

**Options considered**:
1. **Pass `--include` through to the API.** Rejected: an unsupported include value is silently dropped, returning an actor with no embed — indistinguishable from "this actor has no roles." The closed-enum hazard 025 ADR-4 / 011 / 048 validate against locally.
2. **Validate `--include` locally (reject-unknown over `{roles, assignments}`); pass the id through.** Chosen — silent conformance to 025 ADR-4 and the `include.go` reject-unknown precedent.
3. **Also validate the id format locally** (the `^(per|agt)_…$` pattern). Rejected: `getActor` answers a bad id with a clean `404` (no silent-wrong-results), so 025 ADR-4 says pass it through; local pattern-matching would duplicate the API's own check and risk drifting from it.

**Decision**: Option 2. `--include` is checked **before any context assembly or request** against the closed 2-value set; an unsupported value is a `UsageError(2)` naming the value and the supported set, no request issued (transport tripwire). This is a **per-read validator with the actor include set** — distinct from the role `--include` set (026's two-validator precedent), never a shared set that would accept `roles`/`subroles`/`policies`/etc. the actor endpoint drops. The actor id is sent verbatim on the path; a malformed/unknown id surfaces as the API's clean `404`, classified by the shared chain (`APIError(3)`, or `PermissionError(4)` on `401`/`403`).

**Consequences**: A small new validator (a `validateActorInclude` sibling or a parameterization of the landed `include.go` validator — an interface-level factoring detail, the same open factoring 048 ADR-3 noted for `--kind`). The CLI is deliberately stricter than the raw endpoint on `--include`. No new error category or exit code.

### ADR-4: Add a singular `ResourceActor` render key (full+compact) for the actor + footprint; reuse structured output unchanged

**Context**: 019 renders human output from `//go:embed` templates per resource key in `internal/render`, dispatched by 020, with explicit-absence guards and golden tests; the registry-exhaustiveness guard requires every resource to have both `full` and `compact`. The registry already carries the singular/plural split for single-vs-list reads — `ResourceRole`/`ResourceOrgRoles`, `ResourceDomain`/`ResourceDomains`, `ResourcePolicy`/`ResourcePolicies` — and `ResourceActors` (048) renders the **flat directory list**, while `ResourceMe` renders one actor *inside the `me` document* (actor + org + membership).

**Options considered**:
1. **Reuse `ResourceActors` or `ResourceMe`.** Rejected: `ResourceActors` is a homogeneous list of directory rows (no footprint); `ResourceMe` is the org/membership-framed identity document. The single actor + governance footprint is a different projection — forcing it onto either forks that template.
2. **Add a singular `ResourceActor` key (full+compact) over `ActorDetail`.** Chosen — the established singular-detail shape (`ResourceRole`/`ResourceDomain`/`ResourcePolicy`), the natural sibling of `ResourceActors`.

**Decision**: Option 2. Add `ResourceActor Resource = "actor"` to the registry (covered by the exhaustiveness guard) with `actor.full`/`actor.compact` embedded templates. `actor.full` renders the `kind` badge + `name` + `per_`/`agt_` `id`, and — when embedded — the footprint: each role's name/purpose/accountabilities/domains and the assignments, each behind an explicit-absence guard (rendered only when `?include`d, never invented). `actor.compact` is the id + kind + name summary. Structured `json`/`yaml` reuses the landed raw-bytes path with no new code — the single read decodes the 2xx body into `json.RawMessage` for structured output and `ActorDetail` for human rendering (018 ADR-2; the single-document, not the `Page` envelope).

**Consequences**: One new resource key and two new templates, mirroring the `ResourceRole` single-detail addition (and following the PR #10 registry-exhaustiveness shape). The embedded-role footprint render can reuse the `role` template's accountability/domain fragments or define its own actor-framed layout — an interface-level detail. The exact row layout, `kind`-badge presentation, and compact form are interface-level.

---

## Cross-cutting Concerns

**No pagination** (distinct from the 048 directory): `GET /actors/{id}` is a single resource; the `roles`/`assignments` embeds arrive inline in the one response, so the read issues exactly one `Execute` and follows no cursor — even when the embedded arrays are large (a held-out validation scenario asserts a single request). This is the 011/025-single-read shape (`getMe`/`getRole`), not the `paging.All` walk; `Page[T]` is not instantiated here. No `--first-page`/`--per-page` flags apply to the single read.

**Error handling**: identical to every read since 011 — typed client errors route through the single shared `classifyClientError` chain (011, widened by 015): auth fail-safe (007) refuses at send time → not-authenticated (`UsageError(2)`); transport → `NetworkUnavailable(6)`; non-2xx, including a `404` for an unknown id → `APIError(3)`; `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)` (017 retries, 015 classifies); `*output.FormatError` → `UsageError(2)`. The landed failure surface is 031's `Diagnose` rendered format-aware by 032's `reportFailure` chokepoint. Messages name the failure + a next step and never include the token. No new exit codes.

**Permission scoping**: the API returns the actor only if the caller's membership permits (PROJECT single-org-+-person constraint); the CLI renders what the API returns and second-guesses nothing. An `agt_` read goes through the ungated `/actors/{id}` — no `ai_integration` gate, never the deferred `/agents/{id}` alias (048 ADR-1 cross-spec precedent; a held-out validation scenario asserts the alias is never routed through).

**Input validation order**: `--output` resolves first (020), then `--include` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/048 fail-fast discipline. The list-only/single-only cross-combo guards (ADR-1) also fire pre-assembly.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent flags, 011/020). No list flags on the single read.

**Testing**: pure-unit coverage is largely inherited (`Actor`/`Role`/`Assignment` decode and the shared error chain are already tested). New tests: golden tests for the two `actor` templates (bare actor; actor with roles embedded showing the footprint; actor with assignments embedded; an actor with no roles, exercising the absence guard); a `internal/cli` godog suite extending the actors feature with the single-read scenarios driven by a fake transport returning a canned `{data: ActorDetail}`, with **transport tripwires** asserting (a) no request when `--include` is unsupported, (b) no request when a filter is combined with an id or `--include` with no id, and (c) exactly one request issued for a single read (no page walk) even with a large embedded `assignments` array.

---

## Implementation Strategy

Hard dependencies are all landed: 007, 009, 010, 015, 018, 019, 020, and **048's `actors` command** (merged as #90 — `internal/cli/actors.go` ships with `cobra.NoArgs`; this spec grows it). The schema types it reuses (`Actor`, full `Role`, `Assignment`, `RoleDetail`) are landed (025 *Complete*, 011). There is no cross-spec race — 048 is already on `main`, so T003 grows a stable, landed command. Three phases by dependency:

1. **Schema** — add `ActorDetail{ Actor; Roles []Role; Assignments []Assignment }` to `internal/glassfrog`, decoded from the `{data: ActorDetail}` document. Reuses `Role`/`Assignment` as-is. Unit-test the decode (bare, roles-embedded, assignments-embedded). (Foundation.)
2. **Render** — add the `ResourceActor` key, register it in `builtinResources` (exhaustiveness guard), and the `actor.full`/`actor.compact` templates with explicit-absence guards for the embeds; golden tests (bare actor; footprint; absent embed). Depends on phase 1's `ActorDetail`.
3. **Command** — grow `actors` to `cobra.MaximumNArgs(1)`, branch `RunE` on `len(args)` (0 → existing directory, 1 → single read), add the `--include` reject-unknown validator over `{roles, assignments}`, the list-only/single-only cross-combo guards, one `Execute` into `{data: ActorDetail}` (no walk), and the single-resource render + exit-code routing. godog single-read scenarios + tripwire tests. Depends on phases 1–2 and on 048's `actors` command existing.

Phases 1–2 are independent of 048 and can land first; phase 3 is the growth of 048's command. The tasks skill decomposes these into PR-sized units.

---

## Risks

- **The `NoArgs` → `MaximumNArgs(1)` growth of 048's landed command** (ADR-1). 048 is merged on `main` (#90) and registers `actors` with `cobra.NoArgs`; this spec widens that validator and adds the `len(args)` branch without regressing the 0-arg directory behavior. *Likelihood: certain. Impact: low-moderate. Mitigation: the directory branch (`runActorsList`) is preserved verbatim; a godog scenario asserts `actors` (0 args) still lists, and the single-read scenarios cover the new branch. No cross-spec race — 048 is already landed.*
- **`--include` validator factoring** (ADR-3). The landed `include.go` validator may hard-code the role include set / `--include` message; reusing it for the actor set needs a set parameter or a thin sibling. *Likelihood: certain. Impact: low. Mitigation: interface decides the factoring; behavior (reject-unknown, named 2-value set, fail-fast) is fixed — and it must be a per-read set, not a shared one (026 two-validator precedent).*
- **Footprint render layout** (ADR-4). The embedded-role footprint (purpose/accountabilities/domains per role, plus assignments) is the richest single-actor projection so far; deciding reuse of the `role` template fragments vs an actor-framed layout is open. *Likelihood: certain. Impact: low. Mitigation: interface-level; golden tests pin whatever layout is chosen; explicit-absence guards prevent invented values (019).*
- **050 surface foreclosure** (ADR-1 consequence). The optional-positional shape forecloses `actors <id> assignments`, so Actor Assignments (050) must take a flag/separate surface. *Likelihood: certain. Impact: none on this spec (it is 050's design input). Mitigation: recorded as cross-spec precedent here; 050 resolves its own surface, exactly as #33/#34/#38 did under 025 ADR-1.*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — the exact `--include` flag shape (repeatable vs comma-separated), help strings, the `ActorDetail` field/symbol names, the request-descriptor shape, and the `actor` render template's footprint layout / compact form / `kind`-badge presentation. → `/score:interface`.
- **The `--include` validator factoring** — `validateActorInclude` sibling vs. parameterizing the landed `include.go` validator. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **Actor Assignments (050)** — the standalone, paginated list of the roles an actor fills (`GET /actors/{id}/assignments`); this plan's `--include assignments` is the inline embed only, and the optional-positional shape forecloses `actors <id> assignments` for 050 (ADR-1).
- **Actor administration** — `updateActor` (`PATCH`) / `deleteActor` (`DELETE`) are out of scope (spec Non-Behavior); this plan reads only.
