# Interface Accord: Role Fillers — CLI

**Feature**: 047-role-fillers
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (single top-level `fillers <role-id>`, `cobra.ExactArgs(1)`, no plural/singular pair), ADR-2 (reuse `glassfrog.Assignment` as-is + new `fillers` render key, full+compact), ADR-3 (no filters, no `--include`, validate nothing locally, pass the role id through). Completeness: Cross-cutting (reuses 025 ADR-3, as 038/048).

---

This accord pins the operator-facing "who fills this role?" surface: the single `fillers` command, its required positional role id, the rendered output, completeness signalling, and exit codes. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). It is the **role-scoped assignment read** of the Actor Reads slice — distinct from Actor Directory (048, `actors --role-id` returns bare actor records for discovery) and Actor Assignments (050, the same `Assignment` resource read by actor); Role Fillers reads `GET /roles/{role_id}/assignments` only and embeds the default `actor`.

---

## Surface

### `glassfrog fillers <role-id>` — list who fills a role

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /roles/{role_id}/assignments` (`listRoleAssignments`) and produces the role's assignments — the actors who fill it, with each filling's focus and election context — as a list result. Walks to completion by default (see Interactions). It is the **role-scoped list half of the 034/038 read pattern standing alone** — there is no singular `filler <asgn-id>` sibling, because the API exposes no `GET /assignments/{id}` (plan ADR-1).

**Synopsis**:
```
glassfrog fillers <role-id> [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `<role-id>` | string | yes | The role whose fillers to read (`role_…`), sent into the path. **Not** validated locally — a free identifier the API resolves; a malformed or unknown value surfaces as the API's clean `404` (025 ADR-4). Omitting it, or passing more than one positional, is a usage error before any request — no API call (`cobra.ExactArgs(1)`). |

**Filter flags**: **none.** `listRoleAssignments` accepts no query filter beyond `include` and pagination, and `--include`'s only non-default value (`role`) is redundant when the caller already named the role — so the command exposes neither (plan ADR-3). The request always carries the endpoint's default `include=actor`, so each filler's `{id, name, kind}` arrives without a flag.

**Walk flags** (optional):

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--first-page` | — | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | — | int | *(016 default)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*Human (`full`/`compact`)* — the **new `fillers` render key** (plan ADR-2), rendering the reused `glassfrog.Assignment` projection. Every row is the same `Assignment` shape (homogeneous, like 048's `actors` / 043's `tensions`). The row leads with the **filling actor** (the answer to "whom do I contact?") and adds the assignment's governance context — `focus` and `elected_until`:
- `full` — one block per filler:
  ```
  <per_… | agt_…>  [<kind>]
    Name:           <actor name>
    Focus:          <focus | (none)>
    Elected until:  <elected_until | (not an elected seat)>
  ```
- `compact` — one line per filler: `<per_… | agt_…>  [<kind>]  <actor name>  — focus: <focus | —>; elected until: <elected_until | —>`

The actor `id` prefix (`per_`/`agt_`) and the `kind` badge let the operator tell a person from an agent and carry the id forward into the Actor Read (049) drill-in. `focus` and `elected_until` are **nullable** in the spec, so each renders an **explicit-absence marker** when unset — never an empty field that hides the distinction "no focus" / "not an elected seat" (spec Output accord). The assignment's own id (`asgn_…`) is not rendered in the human projection (it is not part of the spec's named row fields) but is present in the structured output. For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each assignment's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed roles/domains/policies/projects/actors walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope and **not** a decode-and-re-encode of `[]Assignment`. The structured object carries the full assignment (`id`, `actor_id`, `role_id`, `focus`, `elected_until`, embedded `actor`) regardless of the human projection.

**Empty list** (the role has no fillers): under `full`/`compact`, stdout is exactly the `fillers` template's empty line (`no fillers`) and the command exits `0` — an unfilled role is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: the command has a single `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (a missing or extra positional fails here as an `ExactArgs(1)` violation; an unknown flag fails here too); (2) `--output` resolution (020). There is **no local input validation** beyond cobra's own arg/flag checks — the command exposes no closed-enum flag (plan ADR-3), so unlike 038 (`--status`) / 048 (`--kind`) there is no reject-unknown validator. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/048).

**List completeness** (reuses 025 ADR-3 verbatim, as 038/048):
- **Default** — the command walks every page via `paging.All` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more fillers exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the assignments gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the fillers shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario).

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 047 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Fillers listed (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown or malformed `<role-id>`, or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step (an unknown role id is typically `404`) |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text — "malformed paging: the cursor did not advance at page N" — surfaced inside the incompleteness note (the partial set is already on stdout) |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Missing or extra positional, or unknown flag | — (cobra) | `UsageError` | 2 | cobra's `ExactArgs(1)` / unknown-flag message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Single command, no plural/singular pair** (plan ADR-1): the per-role-read precedent (034 ADR-1, followed by 033/038) pairs a plural list with a singular standalone read; Role Fillers ships only the plural-style list because `/assignments/{id}` carries no `GET` (only the administrative `PATCH`/`DELETE`, out of scope per PROJECT). A `filler <asgn-id>` command would have no endpoint to call (would fabricate API surface — "Spec is the contract"). The required positional makes a missing/extra arg a cobra `ExactArgs(1)` usage error — the structural counterpart of how 038's split rejects list-only flags on the single read, and the inverse of 048's `NoArgs` flag-only surface.
- **No filters, no `--include`, no local validator** (plan ADR-3): the smallest input surface in the read family. The endpoint offers no query filter; the default `include=actor` already embeds what the read is about; the only other include (`role`) re-fetches the named role. So the command sends only the default request + pagination and validates nothing locally — silent conformance to 034 ADR-3's "validate nothing" (and a deliberate divergence from 038's `--status` / 048's `--kind`, which *had* closed-enum inputs to guard). The `role_` id passes through to a clean `404` (025 ADR-4).
- **No schema growth, reuses landed model + generics** (plan ADR-2): reuses `internal/glassfrog.Assignment` (grown by Role Reads, 025, and reserved there for this standalone read) unchanged — `focus`/`elected_until` and the embedded `actor` are already present; the projection simply renders fields 025's embedded view omitted. The list walks `Page[json.RawMessage]` (structured) / `Page[Assignment]` (human). No model edit; the actor-footprint `?include` embeds belong to 049/050. Actor Assignments (050) reuses `Assignment` likewise.
- **Render key** (plan ADR-2): the only new render artifact is the **`fillers`** key — a `FillersView` (mirroring the existing list views), the `ResourceFillers` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and the `fillers.full`/`fillers.compact` templates. Nullable `focus`/`elected_until` get explicit-absence markers in the templates. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes).
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), the walked-list render pattern — `aggregateRawData` for structured + `renderFn` projection for human (018/019/020; `renderResult[T]` is the single-page `/me*` dispatch and is **not** used by a walked list) — the `projectsSeam`-shaped list seam, and the landed failure chain (031's `Diagnose` rendered format-aware by 032's `reportFailure`). Adds only the one command and the `fillers` render path.
- **Command/flag spellings**: `fillers` resolves the spec's confirmed user-facing surface (the spec flagged the spelling `[ASSUMED]`; the command speaks "who fills this role?" while the model/endpoint stay the `Assignment` resource). The walk flags (`--first-page`/`--per-page`) reuse the list-read convention (016/025/038/048). Spellings are conventional, adjustable at build time without changing behavior.
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator (`ExactArgs(1)`) + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
