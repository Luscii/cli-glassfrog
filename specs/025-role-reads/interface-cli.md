# Interface Accord: Role Reads — CLI

**Feature**: 025-role-reads
**Role**: Crafter
**Plan reference**: ADR-1 (one runnable `roles`, optional positional id), ADR-2 (grow `Role`, add `RoleDetail`), ADR-3 (default walk + `--first-page` opt-out; opt-out→0+signal, mid-walk-fail→non-zero+partial), ADR-4 (validate `--include` locally, pass id through).

---

This accord pins the operator-facing org-wide role surface: the `roles` command (list and single read), its flags, the rendered output, the completeness signalling, and the exit codes. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). Distinct from the **token-scoped** `me roles` (012) — this is the whole organization's roles.

---

## Surface

### `glassfrog roles` — one command, optional positional id

`roles` is a guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.MaximumNArgs(1)`. With no positional it **lists** the organization's roles; with one positional it **reads that role by id**. More than one positional is rejected by the `Args` validator (usage error, no API call). Distinct from the org-wide read group hosting future per-role reads (see Consistency Notes).

**Synopsis**:
```
glassfrog roles [--parent ROLE_ID] [--person ACTOR_ID] [--has-subroles[=true|false]] [--tag TAG] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
glassfrog roles <ROLE_ID> [--include a,b,…] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | optional | Omitted → list all org roles. Present → read that one role (`role_…`). The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**List flags** (rejected with a usage error when a `ROLE_ID` is also given):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--parent` | string | — | Filter to roles within a parent role (`parent_role_id`, `role_…`). |
| `--person` | string | — | Filter to roles assigned to a person/agent (`person_id`, `per_`/`agt_…`). |
| `--has-subroles` | bool (tri-state) | *(omitted = all)* | Filter by whether a role has sub-roles. Sent only when the flag is **present** (`cmd.Flags().Changed`): `--has-subroles`/`=true` → roles with sub-roles, `=false` → leaf roles; omit for all. |
| `--tag` | string | — | Filter by tag name (`tag`, case-insensitive). |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

**Single-read flag** (rejected with a usage error when no `ROLE_ID` is given):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--include` | string (comma-separated) | — | Related resources to embed inline (`?include=`, `style:form explode:false`). Valid values: `assignments`, `subroles`, `parent_role`, `policies`, `notes`, `skills`. Validated locally before any request; an unsupported value is a fail-fast usage error. |

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the raw API payload verbatim (018), `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format.

*List, `full`* — one block per role, blocks separated by a blank line:
```
<Role Name> (role_0123…)
  Purpose: <purpose | (no purpose set)>
  Domains:
    - <domain description> | (none)
  Accountabilities:
    - <accountability description> | (none)
```
*List, `compact`* — one line per role: `<Role Name> (role_0123…) — fillers=<N>, subroles=<yes|no>`.

*Single, `full`* — the role block above, followed by one guarded section per **requested** `--include` resource (omitted entirely when not requested; rendered with an explicit-absence marker when requested-but-empty):
```
Assignments:   - <actor name> (per_… | agt_…) | (none)
Subroles:      - <subrole name> (role_…) | (none)
Parent role:   <parent name> (role_…) | (none — anchor role)
Policies:      - <policy title> | (none)
Notes:         - <note text> | (none)
Skills:        - <skill name> (summary; full content via skills read) | (none)
```

**Empty result** (the org has no roles, or no role matches the filters): under `full`/`compact`, stdout is exactly `No roles.` and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit the empty payload as the API returned it.)

## Interactions

**Dispatch**: `RunE` branches on `len(args)` — 0 → list, 1 → single. Before any network call, in order: (1) cobra `Args`/flag parsing; (2) flag-combination checks (`--include` requires an id; list filters/`--first-page`/`--per-page` forbid an id); (3) `--include` value validation and `--output` resolution (020). Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013).

**List completeness** (plan ADR-3):
- **Default** — the command walks every page via `paging.All[Role]` (016) and renders the complete set.
- **`--first-page`** — the command issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more roles exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — the command renders the roles gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the roles shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario).

**Piping / scripting**: stdout carries only the rendered result (or `No roles.`); all diagnostics — failure messages and both incompleteness notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`**. This command **introduces no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Roles listed / role read (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the path — "fix or remove the malformed `.glassfrogrc`" |
| Unknown/forbidden role id, or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step (permission: "check the token's access"; rate-limit: "retry later") |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure + host — "check network access and the base URL" |
| 2xx body did not match the expected shape | `*DecodeError` | `RuntimeError` | 1 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Unsupported `--include` value | *(local validation)* | `UsageError` | 2 | names the unsupported value + the supported set; no request sent |
| `--include` without an id, list filter with an id, `--has-subroles` with a non-bool value, or >1 positional | — (local / cobra) | `UsageError` | 2 | names the misuse (e.g. "--include applies to a single role; pass a role id") |

Codes `4`/`5` arrive via 015's split of `APIError`(3) at the shared classifier (already landed); `roles` benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Org-wide, not token-scoped**: `glassfrog roles` is the whole-organization surface; `glassfrog me roles` (012) lists only the caller's own roles. The two are deliberately separate commands (spec Non-Behavior). They share the grown `internal/glassfrog.Role` but not a command.
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), and the `renderResult[T]` dispatch over `internal/output`/`internal/render` (018/019/020). Adds only the `roles` command, the schema growth, and two render keys.
- **Fail-fast validation** follows 011's `validateInclude` and 013's `validateStatus` precedent (local check before any request, transport tripwire). The role **id** is intentionally *not* validated locally (plan ADR-4): a closed enum that the API silently ignores gets local validation; a free identifier the API 404s gets passed through.
- **`--include` form** is comma-separated (`style:form explode:false`), matching the API and 011's `--include`. `--has-subroles` uses cobra's `Changed` check for tri-state (unset = all). **Flag spellings** (`--parent`, `--person`, `--has-subroles`, `--tag`, `--first-page`, `--per-page`) resolve the spec's `[ASSUMED]` flag-name notes; they are conventional, not behavioral, and may be adjusted at build time without changing behavior.
- **`Q` free-text search not exposed**: `GET /roles` also accepts a `q` parameter; this command exposes only the four structural filters (spec scope). `q` is a candidate for a later spec.
- **Positional-id forecloses subcommands** (plan ADR-1 / risk): the downstream per-role reads (Role Domains #33, Policies #34, Projects #38) and Organization Tree (#26) cannot be children of `roles` — they need a singular `role <id> …` surface or flags. This accord owns only the org-wide list + single read.
- **Command conventions** follow 001/003: the `roles` leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
