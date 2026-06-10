# Plan: Role Domains

**Feature**: 033-role-domains
**Role**: Shaper
**Inputs**: spec.md (033), PROJECT.md, `.score/memory/DECISIONS.md` (precedent 011/025/026 active; no deprecations relevant), DEPRECATION.md (1 entry, unrelated), LEARNINGS.md (passive), landed code (`internal/cli/subroles.go`, `internal/cli/roles.go`, `internal/glassfrog/roles.go`), spec/glassfrog-api-v5.yaml (`listRoleDomains`/`getDomain`/`Domain`)

---

## System Architecture

Role Domains is a **per-role read** in the Governance Reads slice and the second consumer (after Organization Tree's `subroles`) of 025's foreclosure: cobra cannot tell a role id from a subcommand name, so these reads cannot live under `roles <id>` (025 ADR-1) — they get their own command surface. It composes the landed read stack and builds no new transport, identity, pagination, error-classification, or output machinery; it is the `subroles` shape applied to domains, plus a single-object read keyed by a domain id.

**Components and how they connect**:

- **`internal/cli` — two new runnable leaves** (001 guard-registered + explicitly wired in `Assemble()`), each a thin cobra `RunE` over an injected seam (the `subrolesSeam`/`rolesSeam` shape: `assemble` + `newClient` + `sleep` + `resolveFormat`) delegating to a pure run path:
  - **list** (`GET /roles/{id}/domains` → `listRoleDomains`, **required role id**, `ExactArgs(1)`) — resolves the output format, validates flags fail-fast (transport tripwire), assembles the connection, and walks to completion via the shared walker, producing `[]Domain` + a completeness flag. Carries the optional `q` full-text search. This is `runSubroles` with the domains path and a search flag instead of `--include`/`--depth`.
  - **single** (`GET /domains/{id}` → `getDomain`, **required domain id**, `ExactArgs(1)`) — validates `--include` against the closed `{policies}` set, fetches one `Domain` from the `{data: Domain}` envelope, producing one domain with its policies embedded when requested. This is 025's single-read shape narrowed to one optional embed.
  Both route 010's typed client errors through the shared `classifyClientError` (011/015) — **no new `Outcome` category, no `ExitCode` edit** — and render through the format-dispatch path (020): structured formats emit raw bytes (018 ADR-2, via the existing raw aggregator for the list, raw `{data}` for the single), human formats render new `internal/render` views.
- **`internal/glassfrog` — schema growth** (011 ADR-1, 025 precedent). The shared `Domain` (today `{ID, Description}`, the inline-on-Role projection) grows the remaining `getDomain` fields (`type`, `role_id`, `created_at`, `updated_at`) and an optional `Policies []Policy` that stays nil unless `?include=policies` was requested — **reusing the landed `Policy` leaf model**, never a second policy type (025 set that precedent). A `DomainDocument{Data Domain}` wrapper is added for the single read (the `RoleDocument` sibling). The list decodes the existing generic `Page[Domain]` (016).
- **`internal/paging` — the walker** (016). The list's default path calls `paging.All[Domain]` over the retrying `Executor` (017) for the complete set; `--first-page` opts out to a single `Page[Domain]`. Reused verbatim from 025 ADR-3 / 026.
- **`internal/render` — templates** (019). Adds a `domains` list view and a `domain` single view (full + compact) to the embedded registry, guarded by the registry exhaustiveness check (PR #10 shape). The `domain` view guards its policies section (omit when unrequested, explicit-absence marker when requested-but-empty — 019 `missingkey=error` + `{{if}}`, the `RoleDetail` related-field pattern).

**Data flow** (list): `domains <role-id>` → resolve format → validate flags → `assemble(--base-url)` (009) → `newClient` (010) → `RetryExecutor` (017) → `paging.All[Domain]` (carrying `q`) → records + completeness → render/aggregate → stdout (+ incompleteness note to stderr). Single read: `domain <dom-id>` → resolve format → validate `--include` → one `Execute` into `DomainDocument` → render → stdout.

---

## Architecture Decisions

### ADR-1: Two sibling leaf commands keyed by different id types — list by role id, single by domain id; not children of `roles`

**Context**: The two reads key off **different identifiers**: the list is `GET /roles/{id}/domains` (a `role_` id path param) and the single is `GET /domains/{id}` (a `dom_` id path param). 025 ADR-1's optional-positional `roles <id>` shape is foreclosed for hosting subcommands (DECISIONS, 025), and 026 chose two sibling commands (`tree` + `subroles <id>`) rather than create a `role` group, explicitly leaving #33/#34/#38 free to choose their own surface. The spec defers exact command/flag *spelling* to interface but fixes the *behavior* (a role-scoped list + a single-domain read).

**Options considered**:
1. **Two sibling leaves, each `ExactArgs(1)`** — a plural list command taking a role id (the `subroles <id>` shape) and a singular single-read command taking a domain id. Explicit, mirrors 026's two-sibling precedent, each is its own guard-registered runnable leaf; the singular/plural pairing maps cleanly to single/list. Cost: two registrations and help entries.
2. **One command, prefix-routed positional** — `domains <id>` where a `role_` id lists and a `dom_` id reads one. One registration, but routes on id-prefix "magic", needs its own usage error for an unrecognized prefix, and diverges from the codebase's established explicit-command style (025/026 never overload one positional across two endpoints/id-spaces).
3. **One command, flag for the list** — `domains <dom-id>` single + `domains --role <role-id>` list. Mixes positional and flag for the same conceptual "what to read"; the two reads stop being symmetric and the required-role-id list becomes a flag rather than a positional, unlike every other per-role read.

**Decision**: Option 1. Two sibling runnable leaves, each `cobra.ExactArgs(1)`, each guard-registered (001) and explicitly wired in `Assemble()`, neither a child of `roles`. The list command takes the **required role id**; the single command takes the **required domain id**. More than one positional, or a missing positional, is rejected by the `Args` validator (usage error, no API call); the search flag is list-only and `--include` is single-only, each cross-combination a fail-fast `UsageError(2)` before assembly with a transport tripwire (011/013/025/026). The exact command names (plural-list vs singular-single, e.g. `domains`/`domain`) and flag spellings are interface-level — this ADR fixes the *structure* (two required-positional leaves, separate id spaces, off `roles`), not the spelling.

**Consequences**: This does **not** create a shared `role` group — #34 Role Policies (`GET /roles/{id}/policies` + `GET /policies/{id}`) has the identical shape and SHOULD follow this two-sibling precedent. The singular/plural pairing must be disambiguated in help text (interface) so the two leaves aren't mistaken for each other; the prefix-typed ids mean a wrong id still 404s cleanly (ADR-4). No `role`-group decision is taken here, leaving #38 free as before.

### ADR-2: Grow the shared `glassfrog.Domain` (add the standalone fields + optional `Policies`); add a `DomainDocument` wrapper — reuse the landed `Policy`

**Context**: The landed `Domain` is the inline-on-Role projection (`{ID, Description}` — 025/026 surface only the description). The standalone `getDomain`/`listRoleDomains` return the full `Domain`: `id`, `type`, `description`, `role_id`, `created_at`, `updated_at`, plus `policies` only when `?include=policies` (single read). 011 ADR-1 mandates one shared schema type, grown not duplicated; 025 grew `Role` exactly this way and set the precedent that the per-role specs reuse `Policy` rather than redefine it.

**Options considered**:
1. **Grow the shared `Domain` + reuse `Policy` + add `DomainDocument`** — `Domain` gains `type`/`role_id`/`created_at`/`updated_at` and an optional `Policies []Policy` that stays nil unless included; one canonical domain type serves the inline embeds (role/tree) and both standalone reads. `DomainDocument{Data Domain}` is the single-object envelope (the `RoleDocument` sibling).
2. **Separate `DomainDetail` embedding `Domain`** (the `RoleDetail` shape) — keeps the list/inline type lean. But `RoleDetail` earned its separate type by carrying *six* include-gated related-resource fields; `Domain` has exactly one optional embed (`policies`) and the list and single reads return the **same** shape otherwise, so a second type would fork the model for a single nilable slice — the cost 011 ADR-1 / 025 ADR-2 warn against.

**Decision**: Option 1. Grow `Domain` directly: the always-present standalone fields are added unconditionally; `Policies []Policy` is nil/empty unless `?include=policies` was requested, exactly as `Role` carries its optional Accountabilities/Domains. The single read decodes `DomainDocument`; the list decodes the generic `Page[Domain]` (016). Decoding stays tolerant of unknown/extra fields (011).

**Consequences**: Growing the shared `Domain` touches the inline embeds on Role (025) and TreeNode (026), but additively — those render templates surface only `Description` and stay unchanged; the new fields decode and sit unused there (forward-compatible, the 012→025 growth pattern). The `domain` render view guards the policies section so an absent embed never invents a value (019). #34 Role Policies reuses `Policy` and this `DomainDocument`/single-read precedent rather than redefining either.

### ADR-3: Expose the API's `q` as a list-only search flag, sent only when non-blank; it composes with the default walk

**Context**: `listRoleDomains` accepts a `q` full-text parameter (Postgres FTS; empty/whitespace-only ignored by the API; malformed queries return no rows rather than an error). 025 deliberately did **not** expose `q` on `roles` (scoped to structural filters; recorded as a deferred future filter); the 033 defining conversation decided to surface it here. This is the first read in the CLI to expose a search filter.

**Options considered**:
1. **List-only search flag, sent only when the trimmed value is non-blank, carried on every page of the walk** — the flag is local to the list command; a blank/whitespace value is treated as "no search" client-side (so no `q` is sent), matching the API's own ignore semantics; the walker carries `q` on each page request so search composes with walk-to-completion.
2. **Always send `q` (even blank), let the API ignore it** — fewer client rules, but sends an empty parameter the API discards and muddies the request; diverges from the spec's "empty/whitespace ignored" phrasing being observable as no-filter.

**Decision**: Option 1. The search term is a list-only flag (a `UsageError(2)` if combined with the single read). When the trimmed term is non-empty it is set as the `q` query parameter and included in `subrolesQuery`-style query construction so it rides every page request in the walk and the `--first-page` opt-out alike; a blank/whitespace term sends no `q`. A malformed query needs no special handling — the API returns an empty list, which flows through the existing empty-list success path (exit 0). The exact flag spelling (e.g. `--query`/`-q`/`--search`) is interface-level; the behavior is fixed here.

**Consequences**: Search + pagination compose for free because `q` is just another query value on the walked request. This sets the precedent for any future searchable list read (it is the first). An empty result from a matching-nothing search is a valid answer, not an error — the same boundary as "the role controls no domains."

### ADR-4: Validate `--include` locally against the closed `{policies}` set; pass both ids through to a clean 404

**Context**: `getDomain`'s `?include` is a closed enum whose only value is `policies`; by the API's pattern (getRole/tree silently *ignore* unknown include values) a bad value would return the domain **without** the embed — the silent-wrong-results hazard 025 ADR-4 / 026 reject locally. The role id (list) and domain id (single) are free identifiers; the endpoints 404 an unknown id cleanly (`listRoleDomains` also documents a `400` for a malformed request, which the shared classifier already handles).

**Options considered**:
1. **Validate `--include` locally (reject-unknown `UsageError(2)`), pass both ids through** — reuses 011's pure `validateInclude`/`validateIncludeSet` shape against `{policies}`; an unknown include value is named with the supported set before any request (transport tripwire); an unknown/malformed id surfaces as the API's non-2xx via `classifyClientError`.
2. **Also validate the ids locally** (regex `^role_…$` / `^dom_…$`) — symmetric, but adds a second failure shape and maintenance burden for cases the API already reports cleanly; diverges from "validate only where the API would otherwise silently mislead" (025 ADR-4).

**Decision**: Option 1. `--include` is validated against the closed `{policies}` set, fail-fast, only on the single read (it is a `UsageError(2)` on the list — ADR-1). Both ids are passed through, path-escaped as a single segment (`url.PathEscape`, the `subroles` safeguard so a raw `/`/`..` cannot traverse), and an unknown id becomes the API's `404` → `APIError(3)` (or `PermissionError(4)` on 401/403, `RateLimited(5)` on 429) through the shared classifier — matching the spec's "names the HTTP status" scenarios.

**Consequences**: Asymmetry is deliberate and matches the established precedent (closed-enum-with-silent-failure → validate; clean-404 identifier → pass through). The `{policies}` set is the single-read's own validator (not the role/subroles include set), so it can't accept cross-read values the API would drop.

---

## Cross-cutting Concerns

**Error handling**: All failures flow through the single landed path — `classifyClientError` (011, widened by 015) maps 010's typed errors to the frozen `Outcome`/`ExitCode` registry (auth fail-safe → `UsageError(2)`/`RuntimeError(1)`; transport → `NetworkUnavailable(6)`; non-2xx → `APIError(3)`/`PermissionError(4)`/`RateLimited(5)`). Role Domains adds **no** new category or code. List completeness reuses 025 ADR-3 / 026 verbatim: a deliberate `--first-page` opt-out with more pages exits **0** with a "more exist" stderr note; a mid-walk failure renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)` (the `reportIncompleteSubrolesWalk` shape) — a partial set is never silently presented as complete (CONSTITUTION VI). The token never appears in output or errors.

**Output**: format dispatch (020) is reused — structured `json`/`yaml` emit raw bytes (018 ADR-2): the list aggregates per-domain raw bytes into a `{data:[…]}` document (the existing `aggregateRaw*` shape over `Page[json.RawMessage]`), the single emits the raw `{data: Domain}` verbatim. Human formats render the new `domains`/`domain` views via `internal/render`. The two views are new resource keys in the render registry, guarded by the exhaustiveness check.

**Configuration**: `--base-url` (011) and `--output`/`-o` (020) are inherited persistent root flags. The search flag, `--first-page`, and `--per-page` are local to the list command; `--include` is local to the single command. Page size defaults to the API max (016).

**Testing**: pure `run`/`validate` functions are unit-tested offline behind the injected seam (fake `Executor` returning canned pages, including a multi-page walk, a mid-walk error, a `q`-search request assertion, and an empty result); a transport tripwire asserts no request on any fail-fast rejection (unknown `--include`, cross-flag misuse, wrong arg count). Command behavior is a godog suite against a feature file. Render output is golden/unit-tested with the registry exhaustiveness guard.

---

## Implementation Strategy

Buildable now — every dependency is landed (007, 009, 010, 015, 016, 017, 018, 019, 020, 025). One feature, three phases by dependency:

1. **Schema** — grow `glassfrog.Domain` (add `type`/`role_id`/`created_at`/`updated_at` + optional `Policies []Policy` reusing the landed `Policy`); add the `DomainDocument{Data Domain}` wrapper; confirm `Page[Domain]` decodes. No command yet. Foundation for both reads.
2. **List read** — the role-scoped list leaf (`ExactArgs(1)` role id): search flag, `paging.All[Domain]` walk + `--first-page` opt-out + completeness signalling, structured raw aggregation, human `domains` view, exit-code routing, 001 registration + `Assemble()` wiring. godog + unit tests (incl. the `q`-search and empty-result paths).
3. **Single read** — the domain-scoped single leaf (`ExactArgs(1)` domain id): `--include {policies}` validation, `GET /domains/{id}` → `DomainDocument`, human `domain` view with a guarded policies section, structured raw `{data}`. 001 registration + wiring. Extends the test suite.

Phases 2 and 3 both depend on phase 1 and are otherwise independent (could land in either order); 3 reuses 2's render/test conventions. The tasks skill decomposes these into PR-sized units.

---

## Risks

- **Growing the shared `Domain` perturbs the inline embeds** (ADR-2). `Role` (025) and `TreeNode` (026) embed `[]Domain`. *Likelihood: certain the type changes. Impact: low — additive fields, those views render only `Description`. Mitigation: keep growth additive (no field renames/removals); a render-registry/golden test on the role and tree views back-stops any drift.*
- **Singular/plural two-command surface confusion** (ADR-1). Two leaves whose names differ only by number could be mistyped for each other. *Likelihood: moderate. Impact: low — a wrong-id-type request 404s cleanly (ADR-4). Mitigation: interface writes disambiguating Short/Long help; consider distinct, non-near-homograph spellings at interface time.*
- **`--include=policies` vs standalone Role Policies (#34)** (spec non-behavior). The embed is the inline convenience view; #34 owns the standalone `GET /policies/{id}`. *Likelihood: design-time confusion for #34. Impact: low if the boundary is honored. Mitigation: the spec records the two-views-coexist boundary; #34's Shaper reads it.*
- **`q` search is a new surface** (ADR-3). First read to expose full-text search; future searchable reads will look here for the pattern. *Likelihood: certain to be reused. Impact: low. Mitigation: keep the flag behavior (non-blank-only, walk-composed) clean and document it as the precedent.*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — exact command names (plural-list vs singular-single), the search/`--first-page`/`--per-page`/`--include` flag spellings, help strings, the `DomainDocument` symbol name, and the render template field layout. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **Role Policies (#34)** and the other per-role reads — separate specs; this plan sets the two-sibling command-surface and `Domain`/`Policy` reuse precedent they consume, but designs only Role Domains.
