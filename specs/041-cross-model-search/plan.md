# Plan: Cross-Model Search

**Feature**: 041-cross-model-search
**Role**: Shaper
**Inputs**: spec.md (041), PROJECT.md, `.score/memory/DECISIONS.md` (precedent: 011/015/016/020/025/026 read-stack ADRs), DEPRECATION.md, LEARNINGS.md (passive), spec/glassfrog-api-v5.yaml (`search` §4156, `SearchResult` §6123, `Pagination` §5405), and the **current codebase state** — the landed read stack is in place: `glassfrog.Page[T]` (`internal/glassfrog/page.go`), `paging.All[T]` (`internal/paging/paging.go`), `renderResult[T]` (`internal/cli/render.go`), `classifyClientError` (`internal/cli/clienterror.go`), `validateIncludeSet` (`internal/cli/include.go`), `AssembleFromOS` (`internal/apiclient/assemble.go`), and the render `Resource` registry with its exhaustiveness guard (`internal/render/render.go`).

---

## System Architecture

Cross-Model Search adds the `search` command and otherwise **composes the landed read stack** — it builds no new transport, identity, pagination, error-classification, or output machinery. It is the **simplest member of the 025/026 paginated-list family**: one endpoint (`GET /search`), a flat paginated result, and the same walk-to-completion completeness model as the subroles read — minus the recursion (026) and minus the per-resource `?include` sets (025/026). What is genuinely new is small and bounded: **one new cobra command** (`search`), **one new schema type** (`SearchResult`, decoded from `Page[SearchResult]`), **one new render key** (`search`), and **one new local validator** (`--types`, reusing the `validateIncludeSet` reject-unknown shape).

The defining shape is that this is a **discovery read whose result is heterogeneous and relevance-ordered**. A single `SearchResult` row can be any of eight resource types (`role`/`note`/`project`/`action`/`skill`/`actor`/`policy`/`domain`), and the API returns them in a single ranked list. Two data-integrity properties fall out of that and drive the design: the CLI **forwards the operator's query string verbatim** (the API owns websearch interpretation — no client-side parsing/escaping) and **preserves the API's relevance ordering exactly** (no client re-sort, de-dup, or filter — the ranking is the answer). The result is rendered uniformly so each row's `type` + `id` is the bridge the operator uses to drill into the matching read command.

**Components and how they connect**:

- **`internal/cli` — the `search` command** (new `search.go` + godog suite). A guard-registered (001), explicitly-wired (001) runnable leaf taking a **required** positional query (`ExactArgs(1)` — the spec makes a missing query and a `>1` positional both `UsageError(2)`; multi-word queries are therefore quoted, surfaced at interface/scenarios). A thin cobra `RunE` over an injected `Executor` seam (011 ADR-5) validates `--types` first (reject-unknown, transport tripwire), assembles the connection context, then walks `GET /search?query=<q>&types=<csv>` to completion via the shared walker (016) and renders through `renderResult` (020). The query is attached as the `query` param byte-for-byte; `--types` (when set) as the comma-separated `types` param.
- **`internal/glassfrog` — `SearchResult`** (new flat struct). Decoded from the `Page[SearchResult]` envelope (`{data: [SearchResult], meta: {pagination}}`): `type` (constrained 8-value enum), `id`, `title`, `excerpt` (nullable), `rank` (float), `role_id` (nullable). It is **its own type**, not a reuse of `Role`/`RoleDetail` — the shape is genuinely different (a cross-type summary row, not a resource projection), so 011 ADR-1's "grow the same type" rule does not apply. Tolerant of unknown fields (011).
- **`internal/paging` — the walker** (016). The default path calls `paging.All[SearchResult]` over the retrying `Executor` (017); the `--first-page` opt-out does one `Execute` into `Page[SearchResult]`. Same machinery as the subroles read, parameterized on `SearchResult`.
- **`internal/cli` — `--types` validator** (new, `validateIncludeSet` shape). Reject-unknown against the closed 8-value type set, as `UsageError(2)` before any request, with a transport tripwire. Reuses the established fail-fast shape (the only delta is the flag name in the message — an interface-level factoring detail, see ADR-3).
- **`internal/render` — templates** (019). Adds **one new resource key** — `search` (a flat heterogeneous list) — with full+compact templates, registered into `builtinResources` and covered by the exhaustiveness guard (PR #10 `len`+comma-ok shape). Distinct from every existing key; renders the `type` badge + `title` + `rank` + optional `excerpt`, and a nullable `excerpt`/`role_id` as **absent**, never fabricated.

**Data flow**: `search "<query>"` → `validateTypes(--types)` → `AssembleFromOS(--base-url)` (009) → `NewClientFromOS` (010) → `RetryExecutor` (017) → `paging.All[SearchResult]` over `GET /search` (016) → `Result[SearchResult]` (relevance order preserved) → `renderResult` → stdout (+ an "incomplete — <cause>" or "more available" note to stderr when applicable). The `--first-page` opt-out replaces the walk with one `Execute` into `Page[SearchResult]` (`Data` / `!HasNextPage`).

---

## Architecture Decisions

### ADR-1: `search <query>` is a new sibling command (`ExactArgs(1)`) that forwards the query string verbatim

**Context**: `GET /search` requires a `query` parameter in websearch syntax (quoted phrases, `or`, `-` exclusions). The spec's behavioral accord forwards it "byte-for-byte" and makes both a missing query and more than one positional a `UsageError(2)`. The CLI must decide how the query reaches the command and whether it touches the string.

**Options considered**:
1. **A required positional, `ExactArgs(1)`, forwarded verbatim as `query`** — one positional carries the whole query; a multi-word query is quoted by the operator (shell convention). The CLI never parses, escapes, normalizes, or splits the string — it is attached to the request exactly as received.
2. **A `--query` flag** — explicit, but inconsistent with the read-command family (which takes its primary subject as a positional, e.g. `roles <id>`, `subroles <id>`) and more verbose for the common case.
3. **`MinimumNArgs(1)` joining multiple args with spaces** — lets the operator skip quoting, but contradicts the spec (which makes `>1` positional a usage error) and silently rewrites the query (a join is a client-side transformation of the very string the API must interpret).

**Decision**: Option 1. `search` registers once (001 guard + explicit `main` wiring) as a runnable leaf with `ExactArgs(1)`; zero or `>1` positionals are a fail-fast `UsageError(2)` (cobra's arg validator) before assembly. The single positional is attached to the request as the `query` param with no client-side processing of its contents. **Spec Fidelity (CONSTITUTION I)**: the API owns websearch interpretation; any client rewrite would change what the operator asked for.

**Consequences**: Multi-word queries must be quoted at the shell (`search "strategy review"`) — an operator-facing detail for interface/scenarios to surface in help text and examples. The exact command spelling stays interface-level (the spec flagged it `[ASSUMED]`), but the component shape — a single-positional sibling leaf that forwards verbatim — is fixed here.

### ADR-2: `SearchResult` is a new flat heterogeneous type; the render preserves API relevance order and never fabricates absent fields

**Context**: `SearchResult` (§6123) is a cross-type summary row: `type` (8-value enum), `id`, `title`, `excerpt` (nullable), `rank` (float), `role_id` (nullable). The API returns rows ranked by relevance in a paginated envelope. The spec commits to preserving that order exactly (no re-sort/de-dup/filter) and to rendering null `excerpt`/`role_id` as absent (No Fabricated Data, CONSTITUTION VIII).

**Options considered**:
1. **A dedicated flat `SearchResult` struct + one uniform `search` render key; preserve received order; render nullable fields as absent** — every row decodes into the same struct regardless of `type`; the renderer prints a `type` badge so the operator can tell a role hit from a policy hit and drill in via `type`+`id`. Slice order is the decode order is the API order — the walker appends pages in sequence, so relevance order is preserved for free. Nullable fields render as absent (e.g. no excerpt line) rather than an empty/placeholder string.
2. **Reuse/grow `Role`/`RoleDetail` (or a union of resource types)** — but a `SearchResult` is not a resource projection; it has `rank` and `excerpt` that no resource carries, and forcing it onto `Role` would fork both models (011 ADR-1 only mandates sharing when the *shape* is the same — it is not).
3. **Per-type render keys (a `search-role`, `search-policy`, …)** — richer per-type rendering, but the API returns a single mixed-relevance stream; splitting by type would either break the ranked order or require client-side regrouping (a re-sort the spec forbids).

**Decision**: Option 1. `SearchResult` is its own schema-only type in `glassfrog`, decoded from `Page[SearchResult]`. The render is **one** uniform `search` key over the mixed list, in received (relevance) order, with the `type` shown per row and absent nullable fields omitted. Structured output (json/yaml) decodes `json.RawMessage` and serializes raw bytes verbatim (018 ADR-2), so ordering and nullability need no special machine-path handling. **Data integrity (CONSTITUTION VIII)**: the CLI adds no ranking opinion and invents no content.

**Consequences**: A new flat render key whose distinguishing feature is the per-row `type` badge — interface/scenarios design the exact row layout and compact form. The "preserve server order, never client re-sort" and "render nullable as absent" stances are recorded as cross-spec precedent (any future relevance-ranked or nullable-field read inherits them).

### ADR-3: `--types` is validated locally as a reject-unknown closed enum, reusing the established fail-fast shape

**Context**: `GET /search` accepts an optional `types` parameter — a comma-separated subset of the 8-value enum, defaulting to all types. This is the same closed-enum-input hazard that 013/025/026 validate locally: an out-of-set value should fail fast and discoverably, not reach the API. DECISIONS (025 ADR-4 / 026 ADR-4) already establish reject-unknown local validation for `--include`; `--types` is the same shape over a different vocabulary.

**Options considered**:
1. **Reject-unknown `--types` against the closed 8-value set, locally, before any request** — reuse the `validateIncludeSet` fail-fast shape (sorted, individually-quoted offending values, the supported set named) as `UsageError(2)` with a transport tripwire; omit the param entirely when `--types` is unset (so "all types" is the API default, not a client-spelled list).
2. **Pass `--types` through to the API** — one fewer validator, but a typo returns the API's generic `400` (or, worse, silently narrows) instead of a discoverable client-side error naming the supported set — exactly what 025/026 ADR-4 reject.

**Decision**: Option 1 — silent conformance to the 025/026 reject-unknown precedent, applied to `--types`. The validator reuses the `validateIncludeSet` shape; the only delta is the flag name in the usage message. Whether that is a parameter added to `validateIncludeSet` or a thin sibling `validateTypes` is an interface-level factoring detail (the existing helper hard-codes `--include` in its message). Unset `--types` sends no `types` param.

**Consequences**: The CLI is deliberately stricter than the raw endpoint (rejecting locally what the API would `400` or narrow). One small validator-naming decision is deferred to interface. No new error category or exit code.

### ADR-4: Completeness reuses the walk-by-default + `--first-page` opt-out verbatim — even though search is relevance-ranked

**Context**: `GET /search` is paginated (`per_page`, `cursor`, `Pagination` meta). CONSTITUTION VI requires walk-to-completion-or-signal. Every sibling list read (025 org roles, 026 subroles) walks by default with a `--first-page` opt-out. But search is **relevance-ranked with an unbounded, long low-relevance tail**, and the operator is an agent with a bounded context — so a first-page-by-default (with a "more exist" signal) was a genuine alternative, weighed during clarify.

**Options considered**:
1. **Walk all pages by default; `--first-page` opt-out (verbatim 025/026 model)** — `paging.All[SearchResult]` to completion; the opt-out does one `Execute` and, if more pages exist, exits **0** with a "more available" stderr note; a mid-walk failure renders the partial set + an "incomplete — <cause>" stderr note and exits non-zero via `classifyClientError(Stop)`. Maximally consistent across all list reads; the strongest reading of CONSTITUTION VI (complete by default).
2. **First page by default with a "more exist" signal; opt-in `--all` to walk** — lighter on the per-org rate limit (CONSTITUTION X) and the agent's context for the common "top hits" case, and still VI-compliant via the signal — but it breaks the cross-command symmetry and makes search the one read that defaults to partial.

**Decision**: Option 1 — **resolved in the spec's clarify session (2026-06-11)**. `search` walks by default and offers `--first-page`, identical to the subroles read, parameterized on `SearchResult`. The relevance-ranked alternative (Option 2) was considered and rejected for cross-command symmetry and the strongest reading of VI. Recorded as an ADR (not silent conformance) precisely because search is the case where diverging was tempting — future relevance-ranked/unbounded reads inherit this resolution.

**Consequences**: A large, low-relevance result set is fully walked by default (multiple API calls, more context) — the operator narrows with `--types` or `--first-page`. The opt-out flag name (`--first-page`, provisional) and `--per-page` exposure are interface-level, shared with 025/026.

---

## Cross-cutting Concerns

**Error handling**: All failures flow through the single landed path — `classifyClientError` (011, widened by 015) maps 010's typed errors to the frozen `Outcome`/`ExitCode` registry (auth fail-safe → `UsageError(2)`/`RuntimeError(1)`; transport → `NetworkUnavailable(6)`; non-2xx, including a malformed-query `400` → `APIError(3)`; `403` → `PermissionError(4)`; `429` → `RateLimited(5)`; `*output.FormatError` → `UsageError(2)`; render error → `RuntimeError(1)`). Cross-Model Search adds **no** new category or code. The token never appears in output or errors (secret hygiene).

**Output**: `renderResult[T]` (020) owns format dispatch; structured formats decode `json.RawMessage` and serialize raw bytes verbatim (018 ADR-2 — the relevance order and nullable fields need no machine-path handling), human formats render the typed `SearchResult` slice via the new `search` `internal/render` key. Default human format is `full` (020's default); the command defines no private format flag.

**Permission scoping**: the API returns only results the caller's membership permits (PROJECT single-org-+-person constraint); types the org cannot access (e.g. skills/actors behind `ai_integration`) simply yield no hits, not an error. The CLI does not second-guess this — it renders what the API returns.

**Configuration**: `--base-url` (011) and `--output`/`-o` (020) are inherited persistent root flags. `--types` and the `--first-page` opt-out (+ optionally `--per-page`) are local to the `search` command. Page size defaults to 100 (the `/search` maximum), overriding `paging.All`’s generic default.

**Testing**: The pure `run`/`validate` functions are unit-tested offline behind the injected `Executor` seam (a fake returning canned multi-page result sets — including a heterogeneous-type page, an empty result, a mid-walk error, and a null-excerpt row); a transport tripwire asserts no request on a missing query, a `>1` positional, or an unknown `--types` value; a fixture asserts the outbound `query` param equals the input byte-for-byte and that decode order equals render order (relevance preserved). Command behavior is a godog suite against a feature file. Render output is golden/unit-tested with the registry exhaustiveness guard (PR #10 `len`+comma-ok shape), including a mixed-type list and a null-excerpt absence row.

---

## Implementation Strategy

Buildable now — all hard dependencies are landed (007, 009, 010, 015, 016, 017, 018, 019, 020). No cross-spec schema coordination (unlike 025/026, `SearchResult` is wholly new and shared with nothing). Two phases by dependency:

1. **Schema** — add the flat `SearchResult` struct to `internal/glassfrog`, decoded via the existing `Page[SearchResult]` envelope. Unit-test decode (including null `excerpt`/`role_id`, the 8-value `type` enum, unknown-field tolerance). No command yet. (Foundation.)
2. **Search command** — the `search <query>` command: `--types` reject-unknown validation, verbatim `query` forwarding, `paging.All[SearchResult]` walk + `--first-page` opt-out + completeness signalling, the new `search` render templates (full+compact) + registry entry, render dispatch, exit-code routing. Registration + `main` wiring. godog + unit tests.

Phase 2 depends on phase 1; they could split across PRs but are small enough to land together. The tasks skill decomposes these into PR-sized units.

---

## Risks

- **Large, fully-walked relevance result** (ADR-4). A broad query with the default walk pulls the entire low-relevance tail (many API calls, more agent context). *Likelihood: moderate for broad queries. Impact: low-moderate (latency/rate-limit/context, not correctness). Mitigation: `--types` narrows the search, `--first-page` caps it; both are first-class. Resolved deliberately in clarify in favor of completeness.*
- **Heterogeneous render legibility** (ADR-2). One flat key renders eight result types in a single ranked stream; the `type` badge must make each row's kind and drill-in obvious without regrouping (which would break relevance order). *Likelihood: certain to surface at interface. Impact: moderate (template design). Mitigation: flag for interface/scenarios; structured output is unaffected.*
- **`--types` validator factoring** (ADR-3). The landed `validateIncludeSet` hard-codes `--include` in its message; reusing it for `--types` needs a flag-name parameter or a sibling. *Likelihood: certain. Impact: low. Mitigation: interface decides the factoring; behavior (reject-unknown, named set) is fixed.*
- **Multi-word query quoting** (ADR-1). `ExactArgs(1)` means an unquoted multi-word query is a `>1`-positional usage error. *Likelihood: a real operator footgun. Impact: low (a clear usage error, no wrong results). Mitigation: help text + examples must show quoting; scenarios cover the usage-error path.*
- **`per_page`/`cursor` query-param coexistence with `query`/`types`**. The walker must carry `query` (and `types`) across every page request, not just the first. *Likelihood: low (the walker threads the base request). Impact: moderate if dropped (page 2 loses the query). Mitigation: a unit test asserts every page request retains `query`+`types`.*

---

## What This Plan Does Not Cover

- **Protocol-level surface** — exact command spelling (`search`), flag names/spellings (`--types`, the first-page opt-out, `--per-page`), help strings and quoting examples, the `SearchResult` field-symbol names, and the `search` render template's row layout / compact form / `type`-badge presentation. → `/score:interface`.
- **Executable scenarios** — Gherkin step definitions from the spec's driving scenarios. → `/score:scenarios`.
- **Task decomposition** — PR-sized units with acceptance criteria. → `/score:tasks`.
- **The per-type drill-in read commands** — search produces a `type`+`id` bridge; the matching read commands are their own specs. This plan does not wire search hits to those commands (the spec's non-behavior: no auto-fetch per result).
