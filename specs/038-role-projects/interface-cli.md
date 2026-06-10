# Interface Accord: Role Projects — CLI

**Feature**: 038-role-projects
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (two sibling commands `projects <role-id>` + `project <proj-id>`, list-only flags structurally enforced), ADR-2 (reuse `Project` + `Document[T]`, no schema growth), ADR-3 (`--status` validated locally via the shared `validateStatus`; `--query`/`--tag`/id passed through), ADR-4 (reuse the `projects` list render key, add a singular `project` key). Completeness: Cross-cutting (reuses 025 ADR-3).

---

This accord pins the operator-facing project read surface: the two commands `projects` (per-role list) and `project` (standalone read), their flags, the rendered output, completeness signalling, and exit codes. The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the `--status` validator and its set in `013`/`014` (`internal/cli/status.go`); the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). Distinct from My Projects' token-scoped view (`me projects` → `GET /me/projects`, 014) — this is the *role-addressable* project surface, sharing the `internal/glassfrog.Project` model but not a command.

---

## Surface

### `glassfrog projects <role-id>` — list a role's projects

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /roles/{role_id}/projects` (`listRoleProjects`) and produces the projects owned by that role as a list result. Walks to completion by default (see Interactions).

**Synopsis**:
```
glassfrog projects <ROLE_ID> [--query TEXT] [--status STATUS] [--tag NAME] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The role whose projects to list (`role_…`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `400`/`404`). |

**List flags** (declared only on `projects`):

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--query` | `-q` | string | — | Free-text search, sent verbatim as the endpoint's `q` parameter. Sent only when the flag is **present and non-empty** (`cmd.Flags().Changed`); `--query ""` behaves as no filter. Not validated locally — it is a search string, not a closed enum. |
| `--status` | — | string | — | Filter by project status, sent as the `status` parameter. **Validated locally** before any request against the shared status set (`archived`, `cancelled`, `completed`, `current`, `scheduled`, `someday`, `waiting`); an unsupported value is a usage error naming the value and the supported set, no API call. Sent only when present and non-empty. |
| `--tag` | — | string | — | Filter by tag name (case-insensitive, matched by the API), sent as the `tag` parameter. Free text, passed through; sent only when present and non-empty. |
| `--first-page` | — | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | — | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

The three filters **combine** — each present, non-empty filter is sent as its own query parameter, and the API applies them together (narrowing the list further). With no filter, every project owned by the role is requested.

### `glassfrog project <proj-id>` — read a single project

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /projects/{id}` (`getProject`) and produces the single project with full detail.

**Synopsis**:
```
glassfrog project <PROJ_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `PROJ_ID` | string | yes | The project to read (`proj_…`). Exactly one required (cobra `ExactArgs(1)`). Passed through unvalidated; an unknown id → `404`. |

`project` declares **no list flags**. Passing `--query`/`-q`, `--status`, `--tag`, `--first-page`, or `--per-page` to it is rejected by cobra's unknown-flag handling as a usage error before any request — this is how the spec's "filters apply only to the list" is enforced (no hand-rolled cross-combo guard; plan ADR-1).

**Inherited (persistent root) flags**, read by cobra inheritance on both commands, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*List (`projects`)* — reuses the **landed `projects` render key** (014), unchanged. `full` renders one block per project (`<proj_…>  [<status>]  <description|—>` then an indented `role: … sub-projects: yes/no actions: yes/no tags: …` line); `compact` renders one line per project (`<proj_…>  [<status>]  <description|—>`).

*Single (`project`), `full`* — the full single-project detail (the new `project` render key, plan ADR-4):
```
<proj_…>  [<status>]
  Description:   <description | —>
  Role:          <role_… | (individual initiative — no role)>
  Parent:        <proj_… | (top-level — no parent)>
  Sub-projects:  yes | no
  Actions:       yes | no
  Tags:          <tag, tag | (none)>
  Created:       <created_at | (unknown)>
  Updated:       <updated_at | (unknown)>
  Link:          <link | (none)>
  Note:          <note | (none)>
```
Each nullable field (`description`, `role_id`, `parent_project_id`, `link`, `note`) renders through an explicit-absence guard rather than a blank; the free-text `note`/`description` are rendered verbatim, never truncated or reflowed (CONSTITUTION VI).
*Single, `compact`* — `<proj_…>  [<status>]  <description|—>` (the detail block omitted).

**Empty list** (the role owns no projects, or none matches the supplied filters): under `full`/`compact`, stdout is exactly the `projects` template's empty line (`no projects`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: each command has its own `RunE` (no `len(args)` branching). Before any network call, in order: (1) cobra `Args`/flag parsing (a list-only flag on `project` fails here as an unknown flag); (2) `--status` validation on `projects` (the one closed-enum input — `validateStatus`); (3) `--output` resolution (020). Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/014/025).

**List completeness** (`projects`; reuses 025 ADR-3 verbatim):
- **Default** — the command walks every page via `paging.All[Project]` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more projects exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the projects gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the projects shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario). The single `project` read is unpaginated — no completeness signalling.

**Piping / scripting**: stdout carries only the rendered result (or the empty line); all diagnostics — failure messages and both incompleteness notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`) and rendered per `--output` by **032**. Both commands **introduce no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Projects listed / project read (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown/forbidden role id or project id, or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk (`projects`) | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Unsupported `--status` value (`projects`) | — (`validateStatus`) | `UsageError` | 2 | "unsupported --status value \"…\" — supported: archived, cancelled, completed, current, scheduled, someday, waiting"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| List-only flag on `project`, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; these commands benefit with no edit. The token value never appears in any message.

## Consistency Notes

- **Addressable, not token-scoped**: `glassfrog projects`/`project` is the role-addressable project surface; `glassfrog me projects` (014) reads the authenticated practitioner's own projects via `GET /me/projects`. The two coexist deliberately (spec Non-Behavior) and share the `internal/glassfrog.Project` model, not a command.
- **Two-command split enforces list-only-ness structurally** (plan ADR-1): `--query`/`--status`/`--tag`/`--first-page`/`--per-page` simply don't exist on `project`, so cobra's unknown-flag handling does the rejecting. Both commands are `ExactArgs(1)` (mirrors 026's `subroles <id>`, 033/034's pairs).
- **One local validator** (plan ADR-3): `--status` is the single closed-enum input, validated by the **shared `validateStatus`** (`internal/cli/status.go`, the `supportedActionStatuses` set established by 013 and reused by 014) — this feature extends that validator's consumers, adding no second copy of the set. `--query` and `--tag` are free text passed through (`q`/`tag`); the id is a free identifier passed through to a clean `404`. (025 ADR-4 principle: validate where the API would silently mislead; pass through where it reports cleanly.)
- **No schema growth, reuses landed generics** (plan ADR-2): unlike 034 (which grew `Policy` and generalized `Document[T]`), Role Projects reuses `internal/glassfrog.Project` (grown by 014) unchanged and instantiates the landed `Page[Project]` (016, list) and `Document[Project]` (034, single). No model edit.
- **Render keys** (plan ADR-4): the **list reuses the landed `projects` key** (014) unchanged; the only new render artifacts are a singular **`project`** key — a `ProjectView{Project}` view struct (mirroring `PolicyView{Policy}`), the `ResourceProject` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and the `project.full`/`project.compact` templates. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes).
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), the `renderResult[T]` dispatch (018/019/020), and the diagnostic/failure-rendering chain (031/032). Adds only the two commands and the singular `project` render path.
- **`-q` short alias**: reused from the per-role list convention (033/034), no collision (`-o` is output's). **Flag/command spellings** (`projects`, `project`, `--query`/`-q`, `--status`, `--tag`) resolve the spec's confirmed surface; they are conventional, adjustable at build time without changing behavior.
- **Follows the per-role-read surface precedent** (034 ADR-1 / 033): `projects <role-id>`/`project <proj-id>` completes the trio after `domains`/`domain` (033) and `policies`/`policy` (034). No `role` group is created.
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
