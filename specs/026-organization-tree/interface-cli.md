# Interface Accord: Organization Tree — CLI

**Feature**: 026-organization-tree
**Role**: Crafter
**Plan reference**: ADR-1 (`tree` optional id + `subroles <id>`, two sibling leaves, not children of `roles`), ADR-2 (new recursive `TreeNode`, `{data: TreeNode}` wrapper, not paginated), ADR-3 (subroles reuses 025's walk + `--first-page`; shares `RoleDetail` under first-to-land), ADR-4 (per-read `--include` reject-unknown; `--depth` optional int, 0≠omitted, validated locally; id passed through).

---

This accord pins the operator-facing circle-hierarchy surface: the `tree` and `subroles` commands, their flags, the rendered output (including the **recursive tree** rendering), the subroles completeness signalling, and the exit codes. The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). Distinct from `roles` (025, the org-wide list + single read) and from the token-scoped `me roles` (012) — these reads expose the *nesting* between roles.

---

## Surface

### `glassfrog tree` — one command, optional positional id (the two tree reads)

`tree` is a guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.MaximumNArgs(1)`. With no positional it reads the **whole organization** (`GET /tree` → `getOrgTree`); with one positional it reads the **subtree rooted at that role** (`GET /roles/{id}/tree` → `getRoleTree`). More than one positional is rejected by the `Args` validator (usage error, no API call). Both reads return one nested `TreeNode` document; **neither is paginated**.

**Synopsis**:
```
glassfrog tree [ROLE_ID] [--depth N] [--include a,b,…] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | optional | Omitted → whole-org tree (anchor role as root). Present → subtree rooted at that role (`role_…`). The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**Flags**:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--depth` | int (optional) | *(omitted = full subtree)* | Maximum descendant depth (`depth` query). `0` → the root node alone; `1` → root plus direct children; and so on. Sent **only when present** (`cmd.Flags().Changed`), so `--depth 0` (root only) is distinct from omitting it (full tree). A **negative** value is a fail-fast usage error before any request. |
| `--include` | string (comma-separated) | — | Per-node related resources to embed (`?include=`, `style:form explode:false`). Valid values: `accountabilities`, `domains`, `members`. Validated locally before any request; an unsupported value is a fail-fast usage error (the API would otherwise *silently ignore* it — plan ADR-4). |

### `glassfrog subroles` — required positional id (immediate children)

`subroles` is a guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. It lists the **immediate child roles** of the given role (`GET /roles/{id}/subroles` → `listSubroles`), one level only. Missing or extra positionals are rejected by the `Args` validator (usage error, no API call). This read **is paginated**.

**Synopsis**:
```
glassfrog subroles <ROLE_ID> [--include a,b,…] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | **required** | The parent role whose direct children are listed (`role_…`). Not validated locally; the API resolves it (unknown → `404`). |

**Flags**:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--include` | string (comma-separated) | — | Per-child related resources to embed (`?include=`). Valid values: `assignments`, `subroles`, `parent_role`, `policies`, `notes`, `skills` (the `getRole` set). Validated locally before any request; an unsupported value is a fail-fast usage error. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

> `--depth` is **rejected on `subroles`** (usage error, no request) — depth bounds a recursive tree; subroles returns only one level. `--first-page`/`--per-page` are **rejected on `tree`** — the tree reads are unpaginated.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the raw API payload verbatim (018), `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format.

*Tree, `full`* — a depth-indented tree (two spaces per level), one node per block, the root flush-left:
```
<Role Name> (role_0123…) [structural,elected]
  Purpose: <purpose | (no purpose set)>
  <Child Name> (role_0456…)
    Purpose: <purpose | (no purpose set)>
    <Grandchild Name> (role_0789…)
      Purpose: …
```
When `--include` is set, each node also renders one guarded section per **requested** resource, indented under that node (omitted when not requested; explicit-absence marker when requested-but-empty):
```
  Accountabilities:
    - <accountability description> | (none)
  Domains:
    - <domain description> | (none)
  Members:
    - <actor name> (per_… | agt_…) | (none)
```
**Depth-boundary signal** (spec Clarifications 2026-06-09; risk RC-3): a node whose API `has_subroles` is `true` but whose `children` are not in the result — because `--depth` capped beneath it, or the API withheld them — is rendered with an explicit marker so it is **not** mistaken for a true leaf:
```
  <Branch Name> (role_0456…)  (+ subroles below depth)
```
A node with no children **and** `has_subroles: false` is a true leaf — it renders with nothing indented and no marker. The marker is driven solely by the API's `has_subroles` boolean; it never invents a count of the omitted descendants (CONSTITUTION VI/VIII).

*Tree, `compact`* — one line per node, depth shown by leading indentation, id first then double-space-separated fragments (the repo's compact convention, see `internal/render/templates/*.compact.tmpl`). `has_subroles` is rendered so a depth-capped node (`children=0 has_subroles=yes`) is distinct from a true leaf (`children=0 has_subroles=no`):
```
role_0123…  <Role Name>  children=<N>  has_subroles=<yes|no>  flags=<structural,elected | —>
  role_0456…  <Child Name>  children=<N>  has_subroles=<yes|no>  flags=<—>
```

*Subroles, `full`* — one block per child role (the 025 role-block shape), blocks separated by a blank line; with `--include`, one guarded section per requested resource per child. *Subroles, `compact`* — one line per child: `role_0456…  <Child Name>  has_subroles=<yes|no>`.

**Empty result**:
- `tree` on a leaf root (or `--depth 0`): the root node renders alone with no children indented, exit `0` — an empty `children` set is a valid answer, not an error. If that root's `has_subroles` is `true` (a `--depth 0` cut over a non-empty circle), it carries the depth-boundary marker so the cut is visible.
- `subroles` on a leaf role: under `full`/`compact`, stdout is exactly `No subroles.` and the command exits `0`. (Structured formats emit the empty payload as the API returned it.)

## Interactions

**Dispatch** (`tree`): `RunE` branches on `len(args)` — 0 → whole-org, 1 → rooted. Before any network call, in order: (1) cobra `Args`/flag parsing; (2) flag-applicability checks (`--first-page`/`--per-page` forbidden on `tree`; `--depth` forbidden on `subroles`); (3) `--depth >= 0` check, `--include` value validation against the running read's set, and `--output` resolution (020). Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/025).

**Tree reads are unpaginated** (plan ADR-2): each issues exactly **one** request and renders the full nested document (bounded by `--depth` when given). There is no walk, no first-page opt-out, and no incompleteness note — the response *is* the complete (depth-bounded) tree.

**No conditional-request caching** (spec Non-Behavior): the command issues a plain `GET` and sends **no** `If-None-Match`, so the endpoints' `ETag`/`304` path is never exercised.

**Subroles completeness** (plan ADR-3, reusing 025 verbatim):
- **Default** — walks every page via `paging.All[RoleDetail]` (016) and renders the complete child set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more subroles exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** — renders the children gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the subroles shown are a partial set
  ```
A partial subroles list is therefore never silently presented as complete (CONSTITUTION VI).

**Piping / scripting**: stdout carries only the rendered result (or `No subroles.`); all diagnostics — failure messages and the incompleteness notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`**. These commands **introduce no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Tree read / subroles listed (incl. leaf root, empty subroles) | — | `Success` | 0 | — (result on stdout; subroles incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the path — "fix or remove the malformed `.glassfrogrc`" |
| Unknown/forbidden role id, bad `depth` (API `400`), or other non-2xx | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure + host — "check network access and the base URL" |
| 2xx body did not match the expected shape | `*DecodeError` | `RuntimeError` | 1 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk (subroles) | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Unsupported `--include` value | *(local validation)* | `UsageError` | 2 | names the unsupported value + **the running read's** supported set; no request sent |
| Negative `--depth` | *(local validation)* | `UsageError` | 2 | "--depth must be 0 or greater"; no request sent |
| `--depth` on `subroles`, `--first-page`/`--per-page` on `tree`, missing/extra positional | — (local / cobra) | `UsageError` | 2 | names the misuse (e.g. "--depth applies to a tree read; subroles returns one level") |

Codes `4`/`5` arrive via 015's split of `APIError`(3) at the shared classifier (already landed); these commands benefit with no edit. The token value never appears in any message.

## Consistency Notes

- **Sibling of `roles`, not a child** (plan ADR-1, 025 ADR-1 foreclosure): `tree`/`subroles` are top-level commands because the positional-id `roles <id>` shape can't host `roles <id> <subcommand>`. This accord does **not** create a `role` group; the downstream per-role reads (#33/#34/#38) remain free to choose their own surface.
- **Two completeness models, deliberately**: the tree reads are unpaginated single documents (`--depth` bounds size, no `--first-page`/`--per-page`); `subroles` is paginated and reuses 025's walk + `--first-page` opt-out verbatim. The flag-applicability rejections (`--depth` ⊥ subroles; pagination flags ⊥ tree) keep each command's surface honest.
- **Per-read `--include` sets**: `tree` validates against `{accountabilities, domains, members}`; `subroles` against the `getRole` set `{assignments, subroles, parent_role, policies, notes, skills}`. Two validators, never one shared set (a shared set would accept cross-read values the API silently drops). Both follow 011's `validateInclude` fail-fast shape; the comma-separated form (`style:form explode:false`) matches the API and 025.
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` for subroles (016/017), and the `renderResult[T]` dispatch over `internal/output`/`internal/render` (018/019/020). Adds the `tree` and `subroles` commands, the recursive `TreeNode` schema, and two **new** render keys — `tree` (recursive) and `subroles` (list) — distinct from the shipped `roles` key and from 025's planned `org-roles`/`role` keys.
- **Recursive render is new ground** (plan ADR-2): the `tree` `full`/`compact` templates render depth via indentation — the first non-flat templates in `internal/render`. Structured `json`/`yaml` output is unaffected (raw-bytes path, 018). The indentation width and exact compact fragments are conventional and may be tuned at build time without changing behavior.
- **Shared `RoleDetail` schema** (plan ADR-3): `subroles` decodes `Page[RoleDetail]`; `RoleDetail` + the leaf models + the `Role` growth are shared with 025 under first-to-land-creates (neither landed yet). The subroles per-child include rendering reuses the same role-block + guarded-section shape 025 designs.
- **Flag spellings** (`--depth`, `--include`, `--first-page`, `--per-page`) resolve the spec's `[ASSUMED]` notes; they are conventional, not behavioral, and may be adjusted at build time. The role **id** is intentionally *not* validated locally (plan ADR-4); `--include` and `--depth>=0` are (closed enum / bounded int the API would otherwise mishandle silently or with a wasted call).
- **`Q` free-text search on subroles not exposed**: `listSubroles` also accepts a `q` parameter; this command exposes none (spec scope). `q` is a candidate for a later spec, as in 025.
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
