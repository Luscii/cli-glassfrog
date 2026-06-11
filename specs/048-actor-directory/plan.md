# Plan: Actor Directory

**Feature**: 048-actor-directory
**Role**: Shaper
**Inputs**: spec.md (048), PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 011/015/016/020/025/026/033/034/038/041 read-stack ADRs), DEPRECATION.md, LEARNINGS.md (passive), spec/glassfrog-api-v5.yaml (`listActors` §771, `Actor` §6226, `Q` §5389, `Pagination` §5405), and the **current codebase state** — the landed read stack: `glassfrog.Actor` (`internal/glassfrog/me.go`), `glassfrog.Page[T]` (`internal/glassfrog/page.go`), `paging.All[T]` (`internal/paging`), `renderResult[T]` + `aggregateRawData` (`internal/cli/render.go`, `roles.go`), `classifyClientError` (`internal/cli/clienterror.go`), the `include.go`/`status.go` reject-unknown validators, `AssembleFromOS` (`internal/apiclient`), and the render `Resource` registry with its exhaustiveness guard (`internal/render/render.go`).

---

## System Architecture

Actor Directory adds the `actors` command and otherwise **composes the landed read stack** — it builds no new transport, identity, pagination, error-classification, or output machinery. It is the **simplest member of the 025/026/038 paginated-list family**, and its closest sibling is Cross-Model Search (041): one endpoint (`GET /actors`), a flat paginated list, the same walk-to-completion completeness model, minus the recursion (026) and the per-resource `?include` sets (025/026). What is genuinely new is small and bounded: **one new cobra command** (`actors`), **one new render key** (`actors`, a homogeneous actor list), and **one new local validator** (`--kind`, reusing the established reject-unknown shape). The `glassfrog.Actor` model is **reused as-is** — no schema growth.

The defining shape is that this is the **first read keyed purely on flags, with no positional**. Unlike `roles <id>`/`subroles <id>`/`projects <role-id>` (a role-scoped list whose subject is a path id) and `search <query>` (a required positional subject), the directory's whole subject is the *organization*, narrowed only by optional filters — so `actors` takes **`cobra.NoArgs`** and three optional, combinable filter flags. The other defining decision is the **single-command-with-`--kind`** resolution of the people/agents question: `/people` and `/agents` are aliases over the unified `/actors`, and `--kind human|agent` selects either through that one endpoint, so no separate `people`/`agents` commands are created (and the `ai_integration`-gated `/agents` alias stays deferred).

**Components and how they connect**:

- **`internal/cli` — the `actors` command** (new `actors.go` + godog suite). A guard-registered (001), explicitly-wired (001) runnable leaf taking **no positional** (`cobra.NoArgs` — a positional is a fail-fast `UsageError(2)` via cobra's own arg validator, no hand-rolled guard). A thin cobra `RunE` over the injected `Executor` seam (011 ADR-5) resolves `--output` first (020 — a present-but-invalid selector fails fast before anything else), then validates `--kind` (reject-unknown, transport tripwire), assembles the connection context, then walks `GET /actors?kind=&role_id=&q=` to completion via the shared walker (016) and renders through the walked-list path (020). Each filter is attached only when its flag is `Changed()` and non-empty (the 033/038 optional-flag discipline).
- **`internal/glassfrog` — `Actor`** (reused as-is, no growth). The list decodes the `Page[Actor]` envelope (`{data: [Actor], meta: {pagination}}`) for human rendering and `Page[json.RawMessage]` for structured output. `Actor` already carries `id`, `name`, `kind`, `created_at`, `updated_at` — every field the directory rows project — and decodes tolerantly of extra fields (011).
- **`internal/paging` — the walker** (016). The default path calls `paging.All[Actor]` (human) / `paging.All[json.RawMessage]` (structured) over the retrying `Executor` (017); the `--first-page` opt-out does one `Execute` into the corresponding `Page[…]`. Same machinery as the projects/subroles reads, parameterized on `Actor`.
- **`internal/cli` — `--kind` validator** (new, `validateIncludeSet`/`validateStatus` shape). Reject-unknown against the closed 2-value set (`human`, `agent`), as `UsageError(2)` before any request, with a transport tripwire. `--role-id` and `--query`/`-q` are pass-through (no local validation).
- **`internal/render` — templates** (019). Adds **one new resource key** — `actors` (a flat homogeneous list) — with full+compact templates, registered into `builtinResources` and covered by the exhaustiveness guard (PR #10 `len`+comma-ok shape). Renders per row the `kind` badge + `name` + the `per_`/`agt_` `id` the operator drills into. There is no existing actor-list render — `ResourceMe` renders one actor inside the `me` document, not a directory.

**Data flow**: `actors [--kind] [--role-id] [--query]` → `resolveFormat(--output)` (020) → `validateKind(--kind)` → `AssembleFromOS(--base-url)` (009) → `NewClientFromOS` (010) → `RetryExecutor` (017) → `paging.All[Actor]` over `GET /actors` with the filters carried on every page (016) → `Result[Actor]` → walked-list render (`aggregateRawData` for `json`/`yaml`; the `actors` projection for `full`/`compact`) → stdout (+ an "incomplete — <cause>" or "more available" note to stderr when applicable). The `--first-page` opt-out replaces the walk with one `Execute`.

---

## Architecture Decisions

### ADR-1: `actors` is a new flag-only top-level list command (`cobra.NoArgs`) — not separate `people`/`agents` commands

**Context**: `GET /actors` lists actors across the org with three optional filters (`kind`, `role_id`, `q`) and no required input. `/people` (§1146) and `/agents` are convenience aliases over the same unified endpoint (`/people` = `?kind=human`); the `/agents` alias is `ai_integration`-gated (PROJECT Deferred) while `/actors?kind=agent` reaches agents ungated. The spec confirmed a single discovery command with a `--kind` filter. Every prior read either takes a path id positional (025/026/033/034/038) or a required query positional (041) — the directory has neither.

**Options considered**:
1. **One `actors` command, `cobra.NoArgs`, three optional filter flags including `--kind human|agent`.** The whole subject is the org; filters narrow it. Humans and agents are both reachable through the one ungated endpoint.
2. **Separate `people` and `agents` commands** (mirroring the API aliases). Rejected: `--kind` selects either through one endpoint with no capability lost (`/people` is just `?kind=human`), so a second command forks the discovery surface for nothing; and `/agents` is `ai_integration`-gated and deferred (PROJECT), so an `agents` command would ship a gated path the org may not have — while `actors --kind agent` already reaches agents.
3. **An optional positional name/query** (`actors [name]`). Rejected: the API exposes free-text as the `q` *filter*, not a path subject; a positional would imply a primary-subject read (the `search <query>` shape) and collide with the "filters narrow a directory" model. Free text belongs on `--query`/`-q` (the 033/034 list-filter precedent), not a positional.

**Decision**: Option 1. `actors` registers once (001 guard + explicit `main` wiring) as a runnable leaf with `cobra.NoArgs`; any positional is a fail-fast `UsageError(2)` (cobra's arg validator, no hand-rolled guard). It inherits the `--base-url`/`--output`/`-o` persistent root flags (011/020). No `people` or `agents` command is created.

**Consequences**: First read whose argument surface is **flags only** — a new shape in the read family, but mechanically the simplest (no positional to validate). The exact command spelling (`actors`) and flag spellings (`--kind`, `--role-id`, `--query`/`-q`, the first-page opt-out) stay interface-level (the spec flagged spellings `[ASSUMED]`); the component shape — a flag-only sibling leaf over `/actors` — is fixed here. The people/agents boundary is recorded as cross-spec precedent: Actor Read (049) / Actor Assignments (050) likewise read `/actors*` rather than the gated aliases.

### ADR-2: Reuse `glassfrog.Actor` as-is — no schema growth

**Context**: The `glassfrog.Actor` type, grown by Identity Read (011, `internal/glassfrog/me.go`), already carries `id`, `name`, `kind`, `created_at`, `updated_at` — exactly the `Actor` schema (§6226) each `data` element of `listActors` returns — and decodes tolerantly of extra fields. The `?include=roles,assignments` embeds the endpoint also offers (`roles []Role`, `assignments []Assignment`) belong to Actor Read (049) and Actor Assignments (050), not the directory (spec Non-Behavior). The generic `Page[T]` envelope (016) already exists.

**Options considered**:
1. **Add a directory-specific actor type.** Rejected: violates 011 ADR-1 (one shared schema type, grown not duplicated) and forks the projection; the model is already complete for a directory row.
2. **Reuse `Actor` unchanged; the list walks `Page[json.RawMessage]` (structured) / `Page[Actor]` (human), the walked-list pattern.** Chosen — the 038 ADR-2 "reuse as-is, no schema phase" shape.

**Decision**: Option 2. No model change. The list walk reads `glassfrog.Page[json.RawMessage]` for structured output (aggregated via `aggregateRawData`, per-record raw-byte fidelity, per-page `meta` dropped — the roles/domains/policies/projects pattern) and `glassfrog.Page[Actor]` for human rendering. Both already exist generically — this feature only instantiates them.

**Consequences**: Smaller than 034/041 — no schema phase at all (unlike 041's new `SearchResult`). The directory deliberately does not surface or embed an actor's roles/assignments (`?include` is not sent) — that is the 049/050 footprint surface. If a future need arises to grow `Actor` (e.g. an email field the directory should show), it grows in place per 011 ADR-1; the directory does not need it today.

### ADR-3: Validate the one closed-enum input (`--kind`) locally; pass `--role-id` and `--query` through

**Context**: 025 ADR-4 set the input-handling principle — *validate closed-enum inputs locally* (where a wrong value makes the API silently mislead), but *pass free identifiers and free text through* (where the API reports cleanly). `listActors` offers a **closed `kind` enum** (`human`, `agent`); `role_id` is a free identifier the endpoint answers with a clean `400` on a malformed value; `q` is free text the API matches (and ignores when empty/whitespace).

**Options considered**:
1. **Pass everything through, including `--kind`.** Rejected: an unsupported kind is silently ignored or mishandled by the API, returning results indistinguishable from "no matches" — the closed-enum hazard 025 ADR-4 / 013 / 014 / 041 ADR-3 validate against locally.
2. **Validate `--kind` locally (reject-unknown against `{human, agent}`); pass `--role-id` and `--query` through.** Chosen — silent conformance to 025 ADR-4 and the reject-unknown precedent (`validateIncludeSet`/`validateStatus`).

**Decision**: Option 2. `--kind` is checked **before any context assembly or request** against the closed 2-value set; an unsupported value is a `UsageError(2)` naming the value and the supported set, with no request issued (transport tripwire). `--role-id` → `role_id` and `--query` → `q` are sent verbatim (no enum check); a malformed `role_id` surfaces as the API's clean non-2xx, classified by the shared chain; an empty/whitespace `q` matches the API's own ignore semantics (sent only when `Changed()` and non-empty, so absent and empty both mean "no constraint"). The three filters combine, each its own query parameter.

**Consequences**: A small new validator. Whether it is a thin `validateKind` sibling or a parameterization of the landed `validateIncludeSet` (which hard-codes `--include` in its message) is an interface-level factoring detail — the same open factoring 041 ADR-3 noted for `--types`; the behavior (reject-unknown, named set, fail-fast) is fixed. The CLI is deliberately stricter than the raw endpoint on `--kind`. No new error category or exit code.

### ADR-4: Add a new `actors` render key (full+compact); reuse structured output unchanged

**Context**: 019 renders human output from `//go:embed` templates per resource key in `internal/render`, dispatched by 020, with explicit-absence guards and golden tests; the registry-exhaustiveness guard (`builtinResources`) requires every resource to have both `full` and `compact`. There is **no actor-list render key** — `ResourceMe` renders a single actor *inside* the `me` document (actor + org + membership + roles), not a directory of actors.

**Options considered**:
1. **Reuse the `me` render path.** Rejected: `me` projects one actor with org/membership context; a directory is a flat homogeneous list of actor rows — a different projection, and forcing it onto `me` would fork that template.
2. **Add a new `actors` resource key (full+compact) over `[]Actor`.** Chosen — the 041 ADR-2 / 038 ADR-4 "new list render key" shape, but homogeneous (one `Actor` type, not 041's eight-type heterogeneous row).

**Decision**: Option 2. Add an `ActorsView` (or equivalent) in `internal/render`, the `ResourceActors` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and two embedded templates — `actors.full` (per row: the `kind` badge, `name`, and the `per_`/`agt_` `id`; the empty-set line "no actors") and `actors.compact` (id + kind + name summary). Structured `json`/`yaml` reuses the landed machinery with no new code: the walk reads `Page[json.RawMessage]` and emits the aggregated `{data:[…]}` document via `aggregateRawData` — per-record raw-byte fidelity (018 ADR-2), per-page `meta` dropped — never a single page's envelope.

**Consequences**: One new resource key and two new templates, mirroring 041's `search` addition (and following the PR #10 registry-exhaustiveness shape) — but simpler, because every row is the same `Actor` shape. The exact row layout, the `kind`-badge presentation, and the compact form are interface-level. The incomplete-walk stderr note lives on the command, not the template (025 ADR-3 / 032).

---

## Cross-cutting Concerns

**List completeness** (silent conformance to 025 ADR-3, exactly as 038/041): `actors` defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Actor]` for human; the `--first-page` opt-out does a single `Execute` into the corresponding `Page[…]`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial records, writes "incomplete — <cause>" to stderr, and exits non-zero via `classifyClientError(Stop)`. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`. The filters (`kind`/`role_id`/`q`) are carried on **every** page request, not just the first (the walker threads the base request).

**Error handling**: identical to every read since 011 — typed client errors route through the single shared `classifyClientError` chain (011, widened by 015): auth fail-safe (007) refuses at send time → not-authenticated; transport → `NetworkUnavailable(6)`; non-2xx, including a malformed `--role-id` `400` → `APIError(3)`; `403` → `PermissionError(4)`; `429` → `RateLimited(5)`; `*output.FormatError` → `UsageError(2)`. The landed failure surface is 031's `Diagnose` rendered format-aware by 032's `reportFailure` chokepoint (structured failures emit the 018 error envelope on stdout, human failures write the diagnostic on stderr; partial-walk notes stay on stderr in every format). Messages name the failure + a next step and never include the token. No new exit codes.

**Permission scoping**: the API returns only the actors the caller's membership permits (PROJECT single-org-+-person constraint); the CLI does not second-guess this — it renders what the API returns. An org without `ai_integration` simply yields no agent rows for `--kind agent` (not an error), because `/actors` itself is ungated.

**Input validation order**: `--output` resolves first (020), then `--kind` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/014/041 fail-fast discipline.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025). Page size uses the `paging.All` default (no endpoint-specific maximum to override, unlike 041's `/search` cap of 100).

**Testing**: pure-unit coverage is mostly inherited (the `Actor` decode and `Page[T]` generic are already tested). New tests: golden tests for the two `actors` templates (incl. an empty directory and a mixed human/agent page); a `internal/cli` godog suite over a new `features/actor-reads/actor-directory.feature` driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when a positional is passed (cobra `NoArgs` `UsageError`), (b) no request when `--kind` is unsupported, and (c) the `kind`/`role_id`/`q` filters are present on every page request of a multi-page walk.

---

## Implementation Strategy

Buildable now — all hard dependencies are landed (007, 009, 010, 015, 016, 017, 018, 019, 020). No cross-spec schema coordination (the `Actor` model is reused as-is, ADR-2). Two phases by dependency:

1. **Render** — add the `actors` render path: the `ActorsView` (or equivalent) struct, the `ResourceActors` registry entry (in `builtinResources`, so the exhaustiveness guard covers it), and the `actors.full`/`actors.compact` templates; golden tests (incl. empty set, mixed kinds). No command yet. (Foundation.)
2. **Command** — the `actors` command: `cobra.NoArgs`, `--kind` reject-unknown validation, `--role-id`/`--query`/`-q` pass-through filters carried on every page, `paging.All[Actor]` walk + `--first-page` opt-out + completeness signalling, the walked-list render dispatch (`aggregateRawData` for structured / the `actors` projection for human) + exit-code routing. Guard-registration + `main` wiring. godog + unit tests.

Phase 2 depends on phase 1's `actors` render key; they are small enough to land together or split across two PRs. No schema phase (ADR-2). The tasks skill decomposes these into PR-sized units.

---

## Risks

- **`--kind` validator factoring** (ADR-3). The landed `validateIncludeSet` hard-codes `--include` in its message; reusing it for `--kind` needs a flag-name parameter or a thin sibling. *Likelihood: certain. Impact: low. Mitigation: interface decides the factoring; behavior (reject-unknown, named 2-value set) is fixed.*
- **Filters dropped on page 2+** (cross-cutting). The walker must carry `kind`/`role_id`/`q` across every page request, not just the first. *Likelihood: low (the walker threads the base request). Impact: moderate if dropped (later pages lose the filter, silently wrong results). Mitigation: a unit test asserts every page request retains the filters.*
- **`Actor` model proves too thin for a useful directory row** (low likelihood, low impact). The directory renders `id`/`name`/`kind`; if operators need more (e.g. email) the model grows in place per 011 ADR-1. *Mitigation: the spec scopes the directory to discovery (find the id, then drill in via 049); a richer row is a deliberate later growth, not a gap.*
- **Large org fully walked by default** (low likelihood, low-moderate impact). A filterless `actors` in a large org walks every page (more API calls, more agent context). *Mitigation: `--kind`/`--role-id`/`--query` narrow it and `--first-page` caps it; identical to the 038/041 walk-by-default tradeoff, resolved the same way for cross-command symmetry (CONSTITUTION VI).*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — exact command spelling (`actors`), flag names/spellings (`--kind`, `--role-id`, `--query`/`-q`, the first-page opt-out, `--per-page`), help strings, the `ActorsView` field-symbol names, the request-descriptor shape, and the `actors` render template's row layout / compact form / `kind`-badge presentation. → `/score:interface`.
- **The `--kind` validator factoring** — `validateKind` sibling vs. parameterizing `validateIncludeSet`. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **The actor drill-in reads** — the directory produces a `per_`/`agt_` `id` bridge; Actor Read (049, single actor + roles) and Actor Assignments (050) are their own specs. This plan reads no single actor and embeds no `?include` set (spec Non-Behavior).
- **A `people`/`agents` command and the `ai_integration`-gated `/agents` alias** — deliberately not created (ADR-1); `--kind` covers actor-kind selection through the ungated `/actors`.
