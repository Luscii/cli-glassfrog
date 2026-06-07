# Interface Accord: My Roles — CLI

**Feature**: 012-my-roles
**Role**: Crafter
**Plan reference**: ADR-1 (the `roles` subcommand under 011's runnable `me`), ADR-2 (conform to 011's landed scaffolding — reuse, don't recreate), ADR-3 (reuse 011's `classifyClientError` + exit-code mapping; auth→2/1), ADR-4 (grow the shared `internal/glassfrog.Role` + pure projection/incompleteness).

---

This accord pins the operator-facing command that lists the roles the authenticated practitioner fills (`GET /me/roles`, token-scoped). It pins the **invocation surface** — the command, its flag, the default projection output, the incompleteness signal, and the exit codes. The request seam it calls through is pinned in 010's `interface-spec.md`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009).

---

## Surface

### `glassfrog me` (command, owned by Identity Read 011)

`me` is the **runnable** identity command Identity Read (011) landed (`glassfrog me [--include roles]` prints the authenticated actor). My Roles does not create it — it attaches a `roles` subcommand under it (a command that both runs and parents children; confirm the 001 guard permits this at implementation). `me actions` (013) / `me projects` (014) attach the same way later. Distinct from the org-wide `roles` group (Governance Reads).

### `glassfrog me roles`

List the roles the authenticated practitioner fills via a primary, non-discarded assignment.

**Synopsis**: `glassfrog me roles [--base-url URL]`

| Argument | Type | Required | Description |
|---|---|---|---|
| *(none)* | — | — | The command takes no positional arguments. The caller is identified by the stored token, not named on the command line. Stray positionals are rejected (`Args: NoArgs`). |

| Flag | Type | Default | Description |
|---|---|---|---|
| `--base-url` | string | *(008 precedence)* | Override the API base URL. **Registered by 011 as a persistent flag on the root** (name from `apiclient.FlagBaseURL`) and inherited by `me roles` — this command does not declare it. Highest rung of the 008 precedence chain (flag → `GLASSFROG_BASE_URL` → `.glassfrogrc base_url` → built-in default). |

**Output** (success, stdout) — a reshaped projection, one block per role, blocks separated by a blank line. The raw API JSON envelope is never emitted (a raw `--output json` mode is the deferred Unconsumable Output capability):

```
<Role Name> (role_0123456789abcdef0123456789abcdef)
  Purpose: <purpose>
  Domains:
    - <domain description>
    - <domain description>
  Accountabilities:
    - <accountability description>
    - <accountability description>
```

- **Name line**: the localized role name, followed by the full `role_…` id in parentheses — present so an agent can make follow-up calls, kept unobtrusive ("minimal").
- **Purpose**: the role's purpose; `(no purpose set)` when the API returns it null/empty.
- **Domains** then **Accountabilities**: each header always renders (uniform, agent-parseable structure); ordered Domains-before-Accountabilities to match the Glassfrog web UI. Each lists the items' `description` text; when the role has none, the header is followed by `    (none)`.
- The role's `fillers`, `tags`, and classification `flags` are **not** shown (spec Non-Behaviors).

**Empty result** (the practitioner fills no roles): stdout is exactly `No roles.` and the command exits `0` — an empty list is a valid answer, not an error.

## Interactions

**Invocation flow** (from the operator's view): the command resolves the connection context once (the stored token + the effective base URL), builds the request client once, and sends one `GET /me/roles`. The operator supplies nothing but the optional `--base-url`; identity comes from the stored credential (005/007).

**Piping / scripting**: stdout carries only the role projection (or `No roles.`), so it pipes cleanly into other tools. All diagnostics — failure messages and the incompleteness note — go to **stderr**, never interleaved into the stdout projection.

**Incompleteness signal**: `GET /me/roles` is paginated and this slice fetches only the first response (full paging is the deferred Pagination capability, 016). When the API reports more roles than the response carried (`meta.pagination.has_next_page`), the command prints the roles it received to stdout and writes a single explicit line to **stderr**, still exiting `0`:

```
note: more roles exist than shown; pagination is not yet supported, so this list may be incomplete
```

A partial list is therefore never silently presented as complete (spec accord; validation scenario).

**Configuration precedence**: `--base-url` is the top rung of 008's chain (flag → `GLASSFROG_BASE_URL` → `.glassfrogrc base_url` → default); the token is resolved by 005's precedence (`GLASSFROG_TOKEN` → `.glassfrogrc token`). No other configuration this slice.

## Error Communication

Errors are written to **stderr**; the process exit code is the category mapped by Exit-Code Convention (004). This command routes the 010 outcome through **011's shared `classifyClientError(err) Outcome`** helper — it does not inline its own `errors.As` chain and emits no code directly. Every message names the cause **and** a next step, and never includes the token.

The `Outcome` column uses the canonical `internal/cli` enum names. This command **introduces no `Outcome` category and no `ExitCode` case** — it reuses the categories 011 added (`APIError`/3, `NetworkUnavailable`/6) and the pre-existing ones (`UsageError`/2, `RuntimeError`/1, `Success`/0), per the conformance reconciliation (plan ADR-2/3; DECISIONS 2026-06-07).

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit code | stderr message (cause + next step; token never included) |
|---|---|---|---|---|
| Roles listed (incl. empty list) | — | `Success` | 0 | — (projection or `No roles.` on stdout; incompleteness note on stderr when applicable) |
| No usable token (not authenticated) | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" (mirrors `runLogin`'s no-token outcome) |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the unreadable credential path — "fix or remove the malformed `.glassfrogrc`" |
| API returned a non-2xx response (generic; incl. 401/403/429) | `*ResponseError` | `APIError` | 3 | names the HTTP status, plus a status-keyed next step — `403` → "you may lack permission for this resource; check the role/membership the token grants"; `429` → "the API rate limit was hit; wait before retrying" |
| Could not reach / complete at the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure and the host, plus "check network access and that the base URL is correct (`--base-url` / `GLASSFROG_BASE_URL` / `.glassfrogrc base_url`)" |
| 2xx body could not be decoded into the projection | `*DecodeError` | `RuntimeError` | 1 | "the API response did not match the expected shape — this may be an API change; report it" |
| Base-URL configuration error (malformed endpoint) | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL and its source — "fix the `--base-url` flag / `GLASSFROG_BASE_URL` / `.glassfrogrc base_url`" |
| Stray positional argument | — (cobra) | `UsageError` | 2 | cobra's standard unknown-argument/usage error (002) |

The auth fail-safe maps to **2/1** (mirroring the shipped `glassfrog auth login`: no-token→2, bad-file→1), not a `PermissionError`(4): codes `4` (`PermissionError`) and `5` (`RateLimited`) stay reserved with no producer here — API Error Extraction (015) later splits `401/403`→4 and Rate-Limit Handling (017) splits `429`→5, without renumbering `3`. The token value never appears in any message.

## Consistency Notes

- **Conforms to Identity Read (011)'s landed scaffolding**: 011 owns the runnable `me` command, the persistent root `--base-url`, the shared `classifyClientError` helper, the operational `Outcome` categories (`APIError`/3, `NetworkUnavailable`/6), and the `internal/glassfrog` schema package. My Roles reuses all of them and adds only the `roles` leaf, the grown `Role` fields, and its projection (plan ADR-2; DECISIONS 2026-06-07). It introduces no new category and no registry edit.
- **Projection labelling syncs with 011**: the *field selection* (name, purpose, domains, accountabilities, minimal id) is fixed by the spec; the rendered wording aligns with 011's `formatMe` projection convention, and both render the shared `internal/glassfrog.Role` fields.
- **Incompleteness signal** resolves the spec's one remaining `[NEEDS CLARIFICATION]` (the signal's *form*) as a single stderr line at exit 0; the *behavior* (must signal, never silent) was already locked in the spec accord.
- **Exit-code mapping** is owned by 004's single registry and produced through 011's `classifyClientError`; this command adds no case. The auth fail-safe→2/1 mirrors the shipped `glassfrog auth login` (006).
- **Command conventions** follow 001/003/006 precedent: the `roles` leaf registers through the fail-loud guard under the runnable `me` command, is wired in `Assemble`, declares `Args: cobra.NoArgs`, and carries a non-empty `Short`. No package-global cobra toggles change. (Open coordination: confirm the guard permits `me` to be both runnable and a parent.)
- **No `--output json` and no filter flags** this slice (spec Non-Behaviors): the default reshaped projection is the only output mode, and `/me/roles` offers no `?status=` filter (unlike `me actions` / `me projects`).
