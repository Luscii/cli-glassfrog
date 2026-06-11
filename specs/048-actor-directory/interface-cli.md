# Interface Accord: Actor Directory — CLI

**Feature**: 048-actor-directory
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (one flag-only `actors` command, `cobra.NoArgs`; no separate `people`/`agents`), ADR-2 (reuse `glassfrog.Actor` as-is, no schema growth), ADR-3 (`--kind` validated locally; `--role-id`/`--query` passed through), ADR-4 (new `actors` render key, full+compact). Completeness: Cross-cutting (reuses 025 ADR-3, as 038/041).

---

This accord pins the operator-facing actor-discovery surface: the single `actors` command, its filter flags, the rendered output, completeness signalling, and exit codes. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). It is the **discovery entry** of the Actor Reads slice — distinct from Actor Read (049, a single actor + footprint) and Actor Assignments (050); the directory reads `GET /actors` only and embeds no `?include` set.

---

## Surface

### `glassfrog actors` — list and find actors

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.NoArgs`. Reads `GET /actors` (`listActors`) and produces the actors in the organization as a list result. Walks to completion by default (see Interactions). It is the **first read keyed purely on flags** — its subject is the whole organization, narrowed only by the optional filters; there is **no positional argument**.

**Synopsis**:
```
glassfrog actors [--kind KIND] [--role-id ROLE_ID] [--query TEXT] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| *(none)* | — | — | `actors` takes **no positional argument** (`cobra.NoArgs`). Any positional is a usage error before any request — no API call. |

**Filter flags** (all optional; each sent only when **present and non-empty** via `cmd.Flags().Changed`):

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--kind` | — | string | — | Filter by actor type, sent as the `kind` parameter. **Validated locally** before any request against the closed set (`human`, `agent`); an unsupported value is a usage error naming the value and the supported set, no API call. Sent only when present and non-empty. |
| `--role-id` | — | string | — | Filter to the actors filling a specific role (`role_…`), sent as the `role_id` parameter. **Not** validated locally — it is a free identifier the API resolves (a malformed value → `400`). Sent only when present and non-empty. |
| `--query` | `-q` | string | — | Free-text search over actor names, sent verbatim as the endpoint's `q` parameter. Not validated locally (a search string, not a closed enum); the API ignores an empty/whitespace value. `--query ""` behaves as no filter. Sent only when present and non-empty. |
| `--first-page` | — | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | — | int | *(016 default)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

The three filters **combine** — each present, non-empty filter is sent as its own query parameter, and the API applies them together (narrowing the directory further). With no filter, every actor in the organization is requested.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*Human (`full`/`compact`)* — the **new `actors` render key** (plan ADR-4), rendering the reused `glassfrog.Actor` projection. Every row is the same `Actor` shape (homogeneous, unlike 041's heterogeneous `search` rows):
- `full` — one block per actor:
  ```
  <per_… | agt_…>  [<kind>]
    Name:  <name>
  ```
- `compact` — one line per actor: `<per_… | agt_…>  [<kind>]  <name>`

The `id` prefix (`per_`/`agt_`) and the `kind` badge let the operator tell a person from an agent and carry the id forward into the 049 drill-in read. For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each actor's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed roles/domains/policies/projects walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope and **not** a decode-and-re-encode of `[]Actor`.

**Empty list** (no actor matches the supplied filters): under `full`/`compact`, stdout is exactly the `actors` template's empty line (`no actors`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: the command has a single `RunE` (no positional branching — `cobra.NoArgs`). Before any network call, in order: (1) cobra `Args`/flag parsing (a positional fails here as a `NoArgs` violation; an unknown flag fails here too); (2) `--output` resolution (020); (3) `--kind` validation (the one closed-enum input — the reject-unknown validator). Output resolution precedes kind validation to keep error precedence consistent with the sibling reads — both are pure, pre-assembly checks, so either order keeps the no-request guarantee, but resolving `--output` first means an invalid `--output` is reported even when `--kind` is also invalid. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/041).

**List completeness** (reuses 025 ADR-3 verbatim, as 038/041):
- **Default** — the command walks every page via `paging.All[Actor]` (016) and renders the complete set. The filters (`kind`/`role_id`/`q`) are carried on **every** page request, not just the first.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more actors exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the actors gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the actors shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario).

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes (more-exist, mid-walk-incomplete) stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 048 reuses `reportFailure` unchanged — every landed read calls it. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Actors listed (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Malformed `--role-id`, or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text — "malformed paging: the cursor did not advance at page N" — surfaced inside the incompleteness note (the partial set is already on stdout) |
| Unsupported `--kind` value | — (the reject-unknown validator) | `UsageError` | 2 | "unsupported --kind value \"…\" — supported: agent, human"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Positional argument, or unknown flag | — (cobra) | `UsageError` | 2 | cobra's `NoArgs` / unknown-flag message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Flag-only, no positional** (plan ADR-1): unlike every prior read — which takes a path id (`roles <id>`, `subroles <id>`, `projects <role-id>`) or a required query (`search <query>`) — `actors`' subject is the organization, so it takes `cobra.NoArgs` and three optional filter flags. A positional is rejected by cobra's `NoArgs` validator (no hand-rolled guard), the structural counterpart of how 038's two-command split rejects list-only flags on the single read.
- **One command, not `people`/`agents`** (plan ADR-1): `/people` (`?kind=human`) and `/agents` are aliases over the unified `/actors`; `--kind human|agent` selects either through that one ungated endpoint with no capability lost, and `/agents` is `ai_integration`-gated and deferred (PROJECT). A second command would fork the discovery surface (spec Non-Behavior).
- **One local validator** (plan ADR-3): `--kind` is the single closed-enum input, validated against `{human, agent}` (reject-unknown, fail-fast, transport tripwire) — the `validateStatus`/`validateIncludeSet` shape. Whether it lands as a thin `validateKind` or a parameterization of `validateIncludeSet` (which hard-codes `--include` in its message) is a build-time factoring detail — the behavior (reject-unknown, named set, no request on failure) is fixed. `--role-id` and `--query` are passed through (`role_id`/`q`) to a clean API response. (025 ADR-4 principle: validate where the API would silently mislead; pass through where it reports cleanly.)
- **No schema growth, reuses landed generics** (plan ADR-2): the directory reuses `internal/glassfrog.Actor` (grown by Identity Read, 011) unchanged and instantiates the landed generics — the list walks `Page[json.RawMessage]` (structured) / `Page[Actor]` (human). No model edit; the `?include=roles,assignments` embeds belong to 049/050.
- **Render key** (plan ADR-4): the only new render artifact is the **`actors`** key — an `ActorsView` (mirroring the existing list views), the `ResourceActors` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and the `actors.full`/`actors.compact` templates. `ResourceMe` renders one actor *inside* the `me` document — a different projection, not reused. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes).
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), the walked-list render pattern — `aggregateRawData` for structured + `renderFn` projection for human (018/019/020; `renderResult[T]` is the single-page `/me*` dispatch and is **not** used by a walked list) — and the landed failure chain: 031's `Diagnose` rendered format-aware by 032's `reportFailure` chokepoint, which every landed read calls and 048 reuses unchanged. Adds only the one command and the `actors` render path.
- **`-q` short alias**: reused from the per-role list convention (033/034/038), no collision (`-o` is output's). **Flag/command spellings** (`actors`, `--kind`, `--role-id`, `--query`/`-q`) resolve the spec's confirmed surface (the spec flagged spellings `[ASSUMED]`); they are conventional, adjustable at build time without changing behavior.
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator (`NoArgs`) + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
