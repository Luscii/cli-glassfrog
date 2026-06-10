# Interface Accord: Role Domains — CLI

**Feature**: 033-role-domains
**Role**: Crafter
**Plan reference**: ADR-1 (two sibling leaves keyed by different id types — `domains <role-id>` list + `domain <dom-id>` single, not children of `roles`), ADR-2 (grow shared `Domain` + optional `Policies`, `DomainDocument` wrapper, reuse `Policy`, no `DomainDetail`), ADR-3 (`q` exposed as a list-only search flag, sent only when non-blank, composes with the walk; completeness reuses 025 ADR-3 / 026 verbatim), ADR-4 (`--include` reject-unknown against `{policies}`; both ids passed through to a clean `404`).

---

This accord pins the operator-facing per-role domains surface: the `domains` (list a role's domains) and `domain` (read one domain) commands, their flags, the rendered output, the list completeness signalling, and the exit codes. The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). These are the second consumer of 025's foreclosure (after 026's `tree`/`subroles`) — they expose the *areas of control* a role holds, the addressable counterpart to the domains embedded inline on a role (025) or a tree node (026).

---

## Surface

### `glassfrog domains` — required positional role id (a role's domains, paginated)

`domains` is a guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. It lists the domains controlled by the given role (`GET /roles/{id}/domains` → `listRoleDomains`). Missing or extra positionals are rejected by the `Args` validator (usage error, no API call). This read **is paginated**.

**Synopsis**:
```
glassfrog domains <ROLE_ID> [--query TERM] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | **required** | The role whose domains are listed (`role_…`). **Not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**Flags**:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--query`, `-q` | string | — | Full-text search over the role's domains (the API `q` param; Postgres FTS, Google-style syntax: `marketing -intern`, `"exact phrase"`, `foo OR bar`). Sent **only when the trimmed value is non-blank** (a blank/whitespace term is treated as no search — plan ADR-3); composes with the walk (carried on every page request). A malformed query returns an empty list, not an error. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

### `glassfrog domain` — required positional domain id (one domain)

`domain` (singular) is a guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. It reads a single domain by its own id (`GET /domains/{id}` → `getDomain`), optionally embedding the policies scoped to it. Missing or extra positionals are rejected by the `Args` validator (usage error, no API call). This read returns one `{data: Domain}` document; it is **not** paginated.

**Synopsis**:
```
glassfrog domain <DOMAIN_ID> [--include policies] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `DOMAIN_ID` | string | **required** | The domain to read (`dom_…`). **Not** validated locally; the API resolves it (unknown → `404`). |

**Flags**:

| Flag | Type | Default | Description |
|---|---|---|---|
| `--include` | string (comma-separated) | — | Related resources to embed (`?include=`, `style:form explode:false`). Valid value: `policies` (the only member of the `getDomain` set). Validated locally before any request; an unsupported value is a fail-fast usage error (the API would otherwise *silently ignore* it — plan ADR-4). |

> `--query`/`--first-page`/`--per-page` are **rejected on `domain`** (usage error, no request) — the single read is unpaginated and unsearchable. `--include` is **rejected on `domains`** — related-resource embedding is a single-read concern.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the raw API payload verbatim (018; the list aggregates the walked pages into one `{data:[…]}` document, the single emits the `{data: Domain}` envelope), `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format.

*Domains list, `full`* — one block per domain, blocks separated by a blank line; the description is the headline, the id trailing:
```
<domain description> (dom_0123…)
```

*Domains list, `compact`* — one line per domain, id first then the description (the repo's compact convention — id, double-space separator):
```
dom_0123…  <domain description>
```

*Domain single, `full`* — the domain's description, id, and controlling role; when `--include policies` is requested, a guarded `Policies:` section (omitted when not requested; explicit-absence marker when requested-but-empty — 019 `{{if}}` + `missingkey=error`):
```
<domain description> (dom_0123…)
  Role: role_0456…
  Policies:
    - <policy title> | (none)
```

*Domain single, `compact`* — one line: `dom_0123…  <domain description>  role=role_0456…` (the policies embed is a `full`-only elaboration; `compact` stays one line).

**Empty result**:
- `domains` on a role that controls no domains, or a `--query` that matches none: under `full`/`compact`, stdout is exactly `No domains.` and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit the empty `{data:[]}` payload.)

## Interactions

**Dispatch**: each command is a single runnable leaf; there is no sub-branching on arg count (both take exactly one positional). Before any network call, in order: (1) cobra `Args`/flag parsing; (2) flag-applicability checks (`--query`/`--first-page`/`--per-page` forbidden on `domain`; `--include` forbidden on `domains`); (3) `--include` value validation against `{policies}` (on `domain`) and `--output` resolution (020). Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/025/026).

**Search composes with the walk** (plan ADR-3): when `--query` carries a non-blank term, `q` rides every page request of the default walk and the `--first-page` opt-out alike — search narrows the set server-side, completeness behaves identically. A blank/whitespace term sends no `q`.

**The single read is unpaginated**: `domain` issues exactly **one** request and renders the `{data: Domain}` document. There is no walk, no first-page opt-out, and no incompleteness note.

**No conditional-request caching**: the commands issue plain `GET`s and send **no** `If-None-Match`, so the `ETag`/`304` path is never exercised.

**Domains-list completeness** (plan ADR-3, reusing 025 verbatim):
- **Default** — walks every page via `paging.All[Domain]` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more domains exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** — renders the domains gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the domains shown are a partial set
  ```
A partial domains list is therefore never silently presented as complete (CONSTITUTION VI).

**Piping / scripting**: stdout carries only the rendered result (or `No domains.`); all diagnostics — failure messages and the incompleteness/more-exist notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`**. These commands **introduce no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Domains listed / domain read (incl. empty list, no-match search) | — | `Success` | 0 | — (result on stdout; list incompleteness/more-exist note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the path — "fix or remove the malformed `.glassfrogrc`" |
| Unknown/forbidden role or domain id, malformed list request (API `400`), or other non-2xx | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure + host — "check network access and the base URL" |
| 2xx body did not match the expected shape | `*DecodeError` | `RuntimeError` | 1 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk (domains) | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Unsupported `--include` value (`domain`) | *(local validation)* | `UsageError` | 2 | names the unsupported value + the supported set `{policies}`; no request sent |
| `--include` on `domains`, `--query`/`--first-page`/`--per-page` on `domain`, missing/extra positional | — (local / cobra) | `UsageError` | 2 | names the misuse (e.g. "--include applies to the single `domain` read; use `glassfrog domain <id> --include policies`") |

Codes `4`/`5` arrive via 015's split of `APIError`(3) at the shared classifier (already landed); these commands benefit with no edit. The token value never appears in any message.

## Consistency Notes

- **Two sibling leaves, not children of `roles`** (plan ADR-1, 025 ADR-1 foreclosure): the positional-id `roles <id>` shape can't host `roles <id> <subcommand>`, so the per-role reads get their own commands. Unlike 026's `tree` (`MaximumNArgs(1)` over one id space), the list and single key off **different** id types (`role_` vs `dom_`), so they are two **required-positional** leaves rather than one optional-positional command — the list mirrors `subroles <id>` (paginated, role id), the single mirrors 025's `roles <id>` read (one document, narrowed to one optional embed). This accord does **not** create a `role` group; #34 Role Policies (identical `GET /roles/{id}/policies` + `GET /policies/{id}` shape) should follow this two-sibling precedent.
- **Plural = list, singular = single**: `domains` lists a role's domains; `domain` reads one. The pairing is disambiguated in each command's `Short`/`Long` (which cross-reference the sibling) so the two leaves aren't mistaken for each other; a wrong-id-type request (e.g. a `dom_` id given to `domains`) 404s cleanly (ADR-4), never silently mis-reads.
- **`--query`/`-q` exposes the API search** (plan ADR-3): this is the **first** CLI read to surface `q` (025/026 deferred it deliberately). The flag is list-only; non-blank-only; walk-composed. The spelling resolves the spec's `[ASSUMED]` search-flag note — conventional, not behavioral, and may be tuned at build time. Sets the precedent for future searchable list reads.
- **Single `--include` set `{policies}`** (plan ADR-4): validated on `domain` only, following 011's `validateInclude` fail-fast shape and the comma-separated form (`style:form explode:false`) the API and 025/026 use. It is the single read's own validator — never shared with the role/subroles include sets, which the API would silently drop here.
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` for the list (016/017), the `--first-page` opt-out + completeness signalling verbatim from 025/026, and the format dispatch over `internal/output`/`internal/render` (018/019/020). Adds the `domains`/`domain` commands, the grown shared `Domain` + `DomainDocument`, and two **new** render keys — `domains` (list) and `domain` (single) — distinct from every shipped key.
- **Grown shared `Domain` is additive** (plan ADR-2): the inline-on-Role (025) and inline-on-TreeNode (026) embeds render only `Description` and are unperturbed by the new `type`/`role_id`/timestamp/`policies` fields. The `domain` view guards its policies section so an absent embed never invents a value (019). `Policy` is reused, not redefined.
- **`--include=policies` vs standalone Role Policies (#34)** (spec Non-Behavior): the embed is the inline convenience view on the domain; the addressable per-policy read belongs to #34. This accord renders the embed only; it adds no standalone policy surface.
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, and changes no package-global cobra toggles. The role/domain **ids** are intentionally *not* validated locally (plan ADR-4); only the closed-enum `--include` is. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
