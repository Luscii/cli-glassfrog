# Plan: Subrole Filler Roll-up

**Feature**: 051-subrole-filler-roll-up
**Role**: Shaper
**Inputs**: `specs/051-subrole-filler-roll-up/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent: §279 048 `actors`/`--kind`, §333/§334 046 subroles roll-up, §205/§211 026 `subroles`, §342 050 separate command, §345/§346 049 `actors` optional-positional + foreclosure), `.score/memory/DEPRECATION.md` (no relevant entries), `.score/memory/LEARNINGS.md` (background); the landed code on `main` — `internal/cli/actors.go` (`runActorsListWalk`, `validateKind`, `aggregateRawData` dispatch), `internal/render/render.go` (`ResourceActors`/`ActorsView`), `internal/glassfrog/me.go` (`Actor`), `internal/glassfrog/page.go` (`Page[T]`), `internal/paging` — and the sibling plans 046 (the twin roll-up) and 048 (the actor read stack)

---

## System Architecture

Subrole Filler Roll-up adds **one read leaf** that rolls up the actors filling an anchor role's **direct sub-roles**: `GET /roles/{role_id}/subroles/actors` (`listSubrolesActors`) — a **paginated roll-up** (one level, not transitive) returning the **same** `{data: [Actor], meta}` shape `GET /actors` returns (the spec notes "Same response shape as `GET /actors`"; cross-linked assignments excluded). It is to **Actor Directory (048)** exactly what **Subroles Tension Roll-up (046)** is to **Tension Reads (043)**: a pure-additive command leaf that **reuses every seam already landed** — model, render path, `--kind` validator, transport, pagination, completeness, error classification, output — and introduces **no new package, model, render resource, validator, transport, pagination, error, or output machinery**. The only genuinely new artifact is the leaf itself (and its BDD feature); its request differs from the `actors` directory walk only in the path segment, the role-keyed positional, and the leaf-role `404`.

The defining decision is **placement**: where a role-keyed actor roll-up lives now that `actors` is a positional-bearing command (048 ships `cobra.NoArgs`; 049 — pending — grows it to `MaximumNArgs(1)`). It does **not** become a `subroles` subcommand under `actors` (which would force the runnable-parent-with-children shape the codebase's optional-positional discipline avoids, 025 ADR-1) — it is its **own** role-keyed top-level read leaf (`ExactArgs(1)`), exactly as `subroles <role-id>` (026) stands beside `roles` and `assignments <actor-id>` (050) stands beside `actors`. The endpoint offers only a `kind` filter (no `role_id`, no `q`), so the leaf surfaces **just `--kind`** — the landed `validateKind` over `{human, agent}` — plus the shared `--first-page`/`--per-page` list flags and the root `--base-url`/`--output`/`-o` persistent flags.

Data flow per invocation (the landed `runActorsListWalk` shape with the path swapped and a required role id): resolve `--output`/render target first (020/035), validate the one closed-enum input (`--kind`) fail-fast (transport tripwire), resolve the connection context once (`AssembleFromOS`, 009), build the `*apiclient.Client` (010/008/007), walk `GET /roles/{role_id}/subroles/actors?kind=` to completion via `paging.All` (016/017) by default, and render by the resolved format (020): `json`/`yaml` aggregate each page's raw bytes into a `{data:[…]}` document via `aggregateRawData`; `full`/`compact` render the **landed** `actors` projection (`render.ResourceActors` + `render.ActorsView`, 048). Typed client errors route through the one shared `classifyClientError`/`Diagnose` chain (011/015/031/032) — no new `Outcome`, no new exit code. The leaf-role `404` (anchor has no sub-roles, or unknown role id) is surfaced verbatim by that chain.

---

## Architecture Decisions

### ADR-1: A distinct role-keyed read leaf (`ExactArgs(1)`) — not a `subroles` subcommand of the positional-bearing `actors` command

**Context**: The roll-up keys on a `role_` anchor id and hits `GET /roles/{role_id}/subroles/actors`. The natural rhyme with 046 would be `actors subroles <role-id>` (a `subroles` verb under the actor surface). But `actors` is **not** a non-runnable group like 042's `tension` (which hosts `tension subroles` cleanly): 048 ships `actors` as a runnable leaf with `cobra.NoArgs`, and 049 (pending, §345) grows it to `MaximumNArgs(1)` (0 args → directory, 1 arg → single actor read). 025 ADR-1's foreclosure (cobra cannot host a subcommand under an optional-positional command without the runnable-parent-with-children ambiguity) is the discipline the codebase has held to: 026 made `subroles <role-id>` its own command rather than a child of `roles`, and 050 made `assignments <actor-id>` its own command rather than `actors <id> assignments` (§342/§346).

**Options considered**:
1. **`actors subroles <role-id>` as a subcommand of `actors`** — rhymes with 046. Rejected: it turns `actors` into a runnable command that *also* parents a subcommand while *also* (post-049) carrying an optional positional — the exact mixed shape 025 ADR-1's foreclosure avoids, and it creates a hard ordering coordination with the pending 049/050 (the `actors` arg surface is in flux). 046's pattern doesn't transfer because `tension` is a group and `actors` is a runnable read.
2. **A distinct role-keyed top-level read leaf** (`ExactArgs(1)` on the anchor role id), registered beside the other reads — the `subroles <role-id>` (026) / `assignments <actor-id>` (050) "own command, not a subcommand" shape. Chosen — silent conformance to 026 ADR-1 and 050; keeps the positional-bearing `actors` command free of children, and carries no coordination dependency on 049/050.

**Decision**: Option 2. The roll-up is a guard-registered (001), explicitly-wired runnable leaf taking exactly one positional role id (`cobra.ExactArgs(1)` — an omitted or second positional is a fail-fast `UsageError(2)` via cobra's own arg validator). It carries the list-only `--kind`/`--first-page`/`--per-page` flags and inherits the root `--base-url`/`--output`/`-o` persistent flags (011/020/035). The **exact command spelling** (leading candidates `subrole-fillers <role-id>` — naming the capability, parallel to 047's `fillers` — or `subrole-actors <role-id>` — naming the endpoint resource) is an interface-level call; the component shape (a distinct role-keyed `ExactArgs(1)` read leaf over `/roles/{role_id}/subroles/actors`, not a child of `actors`) is fixed here.

**Consequences**: A new top-level read command, mechanically a clone of the `actors` directory runner with the path swapped and the org-subject replaced by a required role id. No churn to the landed `actors` command, and no first-to-land race with 049/050. If a future consolidation groups the actor reads, that is a coordinated cross-spec change (the same note 033/034/038 carry).

### ADR-2: Reuse the landed `Actor` model, `actors` render path, and `validateKind` unchanged — add no new model, render resource, or validator; surface only `--kind`

**Context**: `listSubrolesActors` returns the **same** paginated `{data: [Actor], meta}` shape as `listActors` (verified `spec/glassfrog-api-v5.yaml:319-375` — `data: [Actor]`, `meta.pagination`). 048 (landed, #90) already shipped the `actors` render path — `render.ResourceActors` + `render.ActorsView` (`render.go`) — and the closed-enum validator `validateKind` over `{human, agent}` (`actors.go:288`, built on the shared `validateClosedFlagSet`, message names `--kind`). The `glassfrog.Actor` model (`id`/`name`/`kind`/timestamps, 011) and the generic `Page[Actor]`/`aggregateRawData` path are landed. The subroles endpoint's **only** query filter is `kind` — it offers no `role_id` and no `q` (those are 048's directory filters).

**Options considered**:
1. **Add a roll-up-specific render or model** (e.g. grouping actors by sub-role). Rejected: the endpoint returns a flat `[]Actor`, the same shape the directory renders; inventing a grouped projection would fabricate structure the API does not provide (019 anti-fabrication, CONSTITUTION VI) and duplicate a landed render path (011 ADR-1, grow-not-duplicate).
2. **Reuse `ResourceActors`/`ActorsView`, the `Actor` model, and `validateKind` exactly as landed** — the roll-up is a different *source* of the same row shape, narrowed by the same closed-enum filter. Chosen — silent conformance to 011 ADR-1, 048 ADR-2/ADR-3/ADR-4, and 046's "new consumer of a landed validator, not a new validator" pattern.

**Decision**: Option 2. No new render resource, no new model, no new validator. `--kind` reuses the landed `validateKind` (a new *consumer* of 048's set), rejecting an unsupported value as a `UsageError(2)` naming the value and the supported set **before any context assembly or request** (transport tripwire). `--kind` rides (as `kind`) only when `Changed()` and non-empty (the 048 optional-flag discipline). The `role_` anchor id passes through unvalidated to the API (046/048 — clean `404`/`400`). The endpoint's missing `role_id`/`q` filters are simply **not surfaced** (spec non-behavior: no client-side filters beyond `--kind`).

**Consequences**: 051 is purely additive at the command layer — zero churn in `internal/render`, `internal/glassfrog`, or the validator set. If the actor kind enum ever drifts, the one-line change lives in 048's landed set and 051 inherits it for free. No schema phase, no render phase, no validator phase.

### ADR-3: Reuse the `runActorsListWalk` runner shape with the path swapped and a required role id; the leaf-role `404` is surfaced verbatim, distinct from the empty-list `200`

**Context**: The roll-up differs from the `actors` directory walk only in (a) the request path (`/roles/{role_id}/subroles/actors` vs `/actors`), (b) the subject (a required anchor role id vs the whole-org no-positional directory) and the filter set (`--kind` only), and (c) the API's leaf-role behavior: a leaf anchor (no sub-roles) returns **`404`** rather than an empty `200` (spec §319-328 "Only available on expanded roles … Leaf roles return 404"). The spec settled that the CLI surfaces that `404` as the shared read failure with **no** "this role has no sub-roles" interpretation — and that it stays distinct from the genuine empty-list success (sub-roles exist but carry no fillers → `200` with `{data: []}`). This is 046 ADR-3 applied to the actor shape.

**Options considered**:
1. **Special-case the leaf `404`** into a friendly "no sub-roles" empty-list success. Rejected: the CLI is a faithful API surface (VISION Exclusion 1); turning a server `404` into a `0`-exit empty success would hide the wire truth and conflate two genuinely different outcomes (leaf vs childless-but-unfilled).
2. **Surface the `404` through the shared `classifyClientError`/`Diagnose` chain unchanged**, and let the empty `200` render as the landed `no actors` empty-set line. Chosen — silent conformance to 015/031/032 and 046 ADR-3; the two empty-ish outcomes stay distinguishable (non-zero "read failed, status 404" vs `0`-exit empty list).

**Decision**: Option 2. The leaf reuses the landed `runActorsListWalk` shape (resolve `--output` → validate `--kind` → walk → render → completeness note) with the request path set to `/roles/{role_id}/subroles/actors` and the role id supplied from the positional. Whether that is expressed by **parameterizing** `runActorsListWalk` with the path + base query or by a **thin sibling runner** is an interface/tasks-level call; the recommended shape is to parameterize, since the kind filter, paging, completeness, render, and error handling are byte-identical. A `404` (leaf anchor or unknown role id) routes through the shared chain → generic API-error outcome naming the status, non-zero exit; `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)` (a `GET`, so 017 may auto-retry). An empty `200` renders the landed empty-set line and exits `0`. The one-level boundary (direct sub-roles only) is the endpoint's own contract — the command makes no recursive call.

**Consequences**: No new `Outcome`/`ExitCode`. The Builder must not add a leaf-`404` special case nor any recursion into grand-children (a Guardian flag if either appears). A transport tripwire and BDD scenarios pin both the leaf-`404`-is-a-failure and the empty-`200`-is-a-success outcomes so they cannot be conflated.

---

## Cross-cutting Concerns

**Dependency on 048 (landed) and the 046 pattern**: verified against `main` — 048 landed at #90 (the `actors` command, `validateKind`, the `ResourceActors`/`ActorsView` render path, the `aggregateRawData` walked-list dispatch), reusing the 011 `Actor` model and the 016/017 walk. The roll-up reuses all of these unchanged and only adds its own leaf. 035 (User-Defined Template Output) is landed, so the leaf inherits `-o <template-ref>` support through the shared `resolveRenderTarget`/`writeHuman`/`aggregateRawData`/`output.RenderSuccess` flow for free. 046 (landed) is the structural template for the roll-up (one-level, leaf-`404`, reuse-everything).

**List completeness** (silent conformance to 025 ADR-3, exactly as 046/048): the leaf defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Actor]` for human; the `--first-page` opt-out does a single `Execute` into the corresponding `Page[…]`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial records, writes "incomplete — <cause>" to stderr, and exits non-zero via `classifyClientError(Stop)`. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`. The `kind` filter is carried on **every** page request, not just the first (the walker threads the base request).

**Error handling** (conformance to 011/015/031/032, see ADR-3): typed client errors route through the single shared `classifyClientError`/`Diagnose` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. Failures render format-aware through 032's `reportFailure` chokepoint; partial-walk notes stay on stderr in every format.

**Input validation order**: `--output` resolves first (020), then `--kind` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/048 fail-fast discipline. The `role_` id passes through to the API's clean `404`.

**Testing**: pure-unit coverage is inherited (the `Actor` decode, `Page[T]` generics, the `actors` golden templates, and `validateKind`'s set are all tested by 011/048). New tests: a `internal/cli` godog suite over a new `subrole-filler-roll-up.feature` (alongside the existing actor features) driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when `--kind` is unsupported, (b) the leaf-`404` is surfaced as a non-zero failure naming the status, distinct from the empty-`200` success, and (c) the `kind` filter rides every page of a multi-page walk. Reuse the sibling-suite step phrasing (godog matches by text) and never hard-code the validator's supported-set order — it is alphabetically sorted via the shared `validateClosedFlagSet` names helper.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025).

---

## Implementation Strategy

Single phase — one cohesive read leaf with no internal dependencies, every seam below it landed (007, 009, 010, 015, 016, 017, 018, 019, 020, 035; the `Actor` model, the `actors` render path, and `validateKind` from 048):

1. **Command** — the role-keyed roll-up leaf (`ExactArgs(1)`), modeled on the landed `runActorsListWalk` (`--kind` via `validateKind`, `--first-page`/`--per-page`, the 025 completeness logic, the walked-list render via `aggregateRawData` / the `actors` render key) with the request path set to `/roles/{role_id}/subroles/actors` and the role id taken from the positional. Surfaces `--kind` only (no `--role-id`/`--query`). Route failures through `classifyClientError`/`Diagnose` with no leaf-`404` special case and no recursion. Guard-registration + `main` wiring.
2. **BDD** — the `subrole-filler-roll-up.feature` suite covering the spec's driving scenarios (roll-up, `--kind` filter, full-walk, leaf-`404` failure, no-credential, empty-`200`, unsupported-`--kind`, first-page opt-out) plus the structural tripwires.

Phase 2 depends on Phase 1. No render phase, no validator phase, no schema phase (ADR-2: all reused from 048). The tasks skill decomposes these into PR-sized units.

---

## Risks

- **`actors` command surface in flux (049 pending; 050 landed)** (medium likelihood, low impact): 049 grows `actors` to `MaximumNArgs(1)` (still `Analyzed`, not yet merged); 050 added `assignments <actor-id>` and has landed on `main` (#118). ADR-1's distinct-command decision **removes** the coupling — 051 adds its own leaf and touches neither `actors.go` nor the `assignments` command, so it has no ordering dependency on 049/050. Mitigation: 051's base is cut from current `main`; whichever lands next, the roll-up only *appends* a guard-registered leaf and its `main` wiring.
- **Leaf-`404` mistaken for empty result** (low likelihood, medium impact): a Builder might "helpfully" turn the leaf anchor's `404` into an empty-list success, conflating two distinct outcomes (ADR-3). Mitigation: an explicit BDD scenario + the spec's validation scenario pin the `404`-is-a-failure and empty-`200`-is-a-success outcomes; the Guardian flags any special-casing.
- **Path-swap implementation drift** (low likelihood, low impact): if the leaf clones `runActorsListWalk` instead of parameterizing the path, the two runners could drift (e.g. a future completeness fix lands in one only). Mitigation: ADR-3 recommends parameterizing the path so the kind/paging/render/error logic is single-sourced; if a sibling runner is chosen, a test asserts both emit the same render path.
- **Result shape mistaken for assignment-shaped** (low likelihood, low impact): a Builder might project `focus`/`elected_until` (the 047 assignment fields) onto rows. Mitigation: the endpoint returns bare `Actor` records (no such fields) and ADR-2 reuses the actor render path; the spec's validation scenario pins the actor-not-assignment shape.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — the exact command spelling (`subrole-fillers` / `subrole-actors` / …), flag names/spellings (`--kind`, `--first-page`, `--per-page`), help strings, the request-descriptor shape, and whether the runner parameterizes `runActorsListWalk` or forks a sibling are the **interface** skill's concern. The shapes used here are the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units within the single phase are the **tasks** skill's output; the Implementation Strategy above is the input.
- **The `actors` command, the `Actor` model, the `actors` render path, and `validateKind`** — owned by 048 (landed); this plan reuses them unchanged (ADR-2) and only adds the role-keyed roll-up leaf.
- **Transitive/recursive roll-up, the actors filling the anchor role itself, the assignment-shaped read, the `/subroles/people` alias as a separate command, and actor administration** — out of scope per the spec non-behaviors (048 owns the directory and `--role-id` actor filter; 047 owns the assignment-shaped `fillers`; `--kind human` covers the people alias).
