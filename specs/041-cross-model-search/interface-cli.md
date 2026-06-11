# Interface Accord: Cross-Model Search — CLI

**Feature**: 041-cross-model-search
**Role**: Crafter
**Plan reference**: ADR-1 (`search <query>`, single-positional sibling leaf, `ExactArgs(1)`, query forwarded verbatim as `query`), ADR-2 (new flat `SearchResult` type, one `search` render key, relevance order preserved, nullable fields rendered as absent), ADR-3 (`--types` reject-unknown local validation over the closed 8-value set, `validateIncludeSet` shape), ADR-4 (walk-by-default + `--first-page` opt-out, 025/026 model parameterized on `SearchResult`).

---

This accord pins the operator-facing cross-model discovery surface: the `search` command, its flags, the rendered output (a relevance-ordered **heterogeneous** result list with a per-row `type` badge), the completeness signalling, and the exit codes. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). Distinct from every per-resource read (`roles`/`role`, `subroles`/`tree`, `domains`/`domain`, `policies`/`policy`) — `search` is org-wide and crosses all eight resource types, returning a ranked summary the operator drills into via each result's `type` + `id`.

---

## Surface

### `glassfrog search` — required positional query (cross-model full-text search)

`search` is a guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. The single positional is the full-text query (`GET /search` → `search`), forwarded **byte-for-byte** as the `query` parameter. A missing query (zero positionals) and more than one positional are both rejected by the `Args` validator (usage error, no API call). This read **is paginated**.

**Synopsis**:
```
glassfrog search <QUERY> [--types a,b,…] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `QUERY` | string | **required** | The full-text query, in the API's websearch syntax (quoted phrases, `or`, `-` exclusions). Forwarded verbatim as `query` — the CLI never parses, escapes, normalizes, or splits it (plan ADR-1). A **multi-word query must be quoted** at the shell (`search "strategy review"`); an unquoted multi-word query is a `>1`-positional usage error. The query content is **not** validated locally; the API owns websearch interpretation and rejects a malformed query with `400`. |

**Flags**:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--types` | string (comma-separated) | *(omitted = all types)* | Scope the search to one or more resource types (`types` query, `style:form explode:false`). Valid values: `role`, `note`, `project`, `action`, `skill`, `actor`, `policy`, `domain`. Validated locally before any request; an unsupported value is a fail-fast usage error naming the bad value + the supported set (plan ADR-3). Sent **only when set** — omitting it requests all types (the API default), it is not spelled out as the full list. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(default: 100)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range (`1`–`100`). |

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the raw API payload verbatim (018), `full`/`compact` render the human projection (019) under the new `search` resource key. The raw API envelope is never emitted under a human format. **Result order is the API's relevance order, preserved exactly** — the CLI never re-sorts, de-dups, or filters (plan ADR-2).

*Search, `full`* — one block per result, blocks separated by a blank line, in relevance order; the `type` badge leads each block so the operator can tell a role hit from a policy hit and drill in via `type` + `id`:
```
[<type>] <title> (<id>)  rank <rank>
  Excerpt: <excerpt | —>
  Role: <role_id>
```
- The **`Excerpt:`** line always renders; when the API returns a null `excerpt`, it shows the explicit-absence marker `—` (the repo's `{{if .X}}…{{else}}—{{end}}` guard convention) — never invented text (plan ADR-2; CONSTITUTION VIII).
- The **`Role:`** line renders **only when** the result carries a non-null `role_id` (it applies to a subset of types); it is omitted entirely otherwise. (`role_id` is the owning role's `role_…` id — a navigation aid into the role read.)

*Search, `compact`* — one line per result, in relevance order, `type` badge first then double-space-separated fragments (the repo's compact convention, see `internal/render/templates/*.compact.tmpl`):
```
[<type>]  <id>  <title>  rank=<rank>
```

**Empty result**: when the query matches nothing, under `full`/`compact` stdout is exactly `No results.` and the command exits `0` — zero matches is a valid answer, not an error. (Structured formats emit the empty payload as the API returned it.)

## Interactions

**Dispatch**: cobra `Args` (`ExactArgs(1)`) rejects a missing or extra positional at parse time. Then `search`'s `RunE`, before any network call, in order: (1) `--output` resolution (020) — a present-but-invalid selector fails fast **first**, matching the established orchestration order (`internal/cli/roles.go:86-107`); (2) `--types` value validation against the closed 8-value set. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/025/026).

**Query forwarding** (plan ADR-1): the positional is attached to the request as `query` with no client-side processing of its contents. When `--types` is set, its comma-separated value is attached as `types`. **Both `query` and `types` are carried on every page of the walk** — page 2+ requests retain them (a unit test pins this), so scoping and the query survive pagination.

**Completeness** (plan ADR-4, reusing 025/026 verbatim, parameterized on `SearchResult`):
- **Default** — walks every page via `paging.All[SearchResult]` (016) and renders the complete relevance-ordered set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more results exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** — renders the results gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the results shown are a partial set
  ```
A partial result set is therefore never silently presented as complete (CONSTITUTION VI).

**Piping / scripting**: stdout carries only the rendered result (or `No results.`); all diagnostics — failure messages and the incompleteness notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`**. This command **introduces no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Results listed (incl. empty result) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the path — "fix or remove the malformed `.glassfrogrc`" |
| Malformed query (API `400`) or other non-2xx | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure + host — "check network access and the base URL" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` | 3 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Unsupported `--types` value | *(local validation)* | `UsageError` | 2 | names the unsupported value + the supported 8-value set; no request sent |
| Missing / extra positional | — (cobra `Args`) | `UsageError` | 2 | names the misuse ("search takes exactly one query argument — quote a multi-word query") |

Codes `4`/`5` arrive via 015's split of `APIError`(3) at the shared classifier (already landed); this command benefits with no edit. The `*DecodeError` → `APIError`(3) row reflects the 031 reclassification (DEPRECATION 2026-06-10: a 2xx in an unreadable shape is the API's fault, exit 3). The token value never appears in any message.

## Consistency Notes

- **Top-level sibling command, no parent** (plan ADR-1): `search` is org-wide and crosses all eight resource types, so unlike the per-role reads it has no natural parent and is not a child of any resource command. It does not create or join a group.
- **Query is a required positional, forwarded verbatim** (plan ADR-1): the first read whose primary subject is free text rather than an id. `ExactArgs(1)` makes both a missing query and a `>1` positional a fail-fast usage error with **no** hand-rolled guard (cobra's `Args` validator). The consequence — a multi-word query must be quoted — is an operator-facing detail; help text and examples MUST show quoting, with the whole query — operators included — inside one quoted argument (`search "strategy review -archived"`), never a bare `-archived` token that cobra would read as a second positional/flag. Contrast with 033/034's optional list-filter `--query`/`-q` (passed through as `q`): there the query is an optional narrowing flag; here it is the required subject.
- **One heterogeneous `search` render key** (plan ADR-2): a single flat key renders all eight result types in one relevance-ordered stream, distinguished by the per-row `type` badge — **not** per-type keys (which would force a re-sort or regrouping the spec forbids). New key, distinct from every shipped/planned key. The first render key over a deliberately mixed-type list.
- **Relevance order preserved; nullable fields rendered as absent** (plan ADR-2; CONSTITUTION VIII): slice order = decode order = API relevance order (the walker appends pages in sequence), so the CLI never re-sorts/de-dups/filters. A null `excerpt` renders as `—` (marker, not fabricated text); a null `role_id` omits the `Role:` line. Structured `json`/`yaml` output is unaffected (raw-bytes path, 018).
- **`--types` reject-unknown local validation** (plan ADR-3): validated against the closed set `{role, note, project, action, skill, actor, policy, domain}` before any request; an unsupported value is a fail-fast usage error. Reuses the landed `validateIncludeSet` fail-fast shape (sorted, individually-quoted offenders, named supported set) — but that helper hard-codes `--include` in its message, so this command either passes the flag name as a parameter (a small generalization of `validateIncludeSet`) or adds a thin `validateTypes` sibling. That factoring is a build-time detail; the behavior (reject-unknown, named set, transport tripwire) is fixed here. Omitting `--types` sends no `types` param (the API default is all types).
- **Walk-by-default, even for a relevance-ranked read** (plan ADR-4, spec Clarifications 2026-06-11): `search` walks every page by default and offers `--first-page`, identical to the subroles read, over `paging.All[SearchResult]`. The first-page-by-default alternative (tempting for a relevance-ranked, long-tailed result) was weighed in clarify and rejected for cross-command symmetry + the strongest reading of CONSTITUTION VI. The operator narrows a broad query with `--types` or caps it with `--first-page`.
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015/031), the `paging.All` walker + `RetryExecutor` (016/017), and the `renderResult[T]` dispatch over `internal/output`/`internal/render` (018/019/020). Adds the `search` command, the flat `SearchResult` schema, and one **new** render key (`search`).
- **Flag spellings** (`--types`, `--first-page`, `--per-page`) and the render row layout / `type`-badge presentation / compact fragments resolve the spec's `[ASSUMED]` notes; they are conventional, not behavioral, and may be tuned at build time. The query content and the role **id** in `role_id` are intentionally *not* validated locally; only the closed-enum `--types` is (the API would otherwise `400` or silently narrow).
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
