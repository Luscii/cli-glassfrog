# Interface Accord: My Projects — CLI

**Feature**: 014-my-projects
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (the `my projects` leaf under the `my` parent); ADR-1 (reuse `Pagination`/envelope + `validateStatus`); ADR-2 (no `?include` embedding); ADR-3 (guard-registered leaf + injected seam + pure `runMyProjects`/`formatMyProjects`). Reuses 011's `--base-url`, `classifyClientError`, and error→exit-code mapping; 013's `validateStatus` and projection shape.

---

## Surface

This accord defines the **`my projects` command** — the dedicated self-service read that lists the projects owned by roles the authenticated practitioner fills. It is the structural twin of `my actions` (013).

### Command

```
glassfrog my projects [--status <status>]
```

- A **leaf under the `my` parent command** (introduced by My Roles, 012), sibling to `my actions` (013) and `my roles` (012). Takes **no positional arguments** — an unexpected positional is a usage error (002).
- **Output goes to stdout**; diagnostics and error messages go to stderr; the outcome class is also signalled by the exit code (see Error Communication).
- **No `--include` flag**: the `/me/projects` operation offers no `include` parameter, so sub-project/action embedding is not exposed (ADR-2).

### Flags

| Flag | Scope | Value | Description |
|---|---|---|---|
| `--status` | local to `my projects` | one of `archived` `cancelled` `completed` `current` `scheduled` `someday` `waiting` | Optional filter mapped to the API's `?status=`. **Validated before any request** by the shared `validateStatus` (013): an unsupported value is rejected as a usage error naming the value and the supported set, with no request issued. Absent → no status constraint. |
| `--base-url` | **persistent, on the root** (011) | a URL | The Glassfrog API base URL — highest-precedence rung of base-URL resolution (008). Registered once on the root by Identity Read (011) and inherited here; `my projects` reads its value, registers nothing. |

### Output — the projects projection (success, stdout)

On a `200`, `my projects` prints a **reshaped, predictable projection** (not raw JSON — a structured `--output json` mode is a deferred cross-cutting capability, see Consistency Notes). It renders **one entry per project** in the order the API returned them; each entry surfaces a stable set of fields, and the **id is always present** (the machine-actionable handle):

- **id** (`proj_…`), **status** (one of the status enum), **description** (the project text), **role** (the owning `role_…` id, or an explicit "—" / "no role" marker when null — non-role-owned projects), **children** (whether sub-projects / actions exist, from `has_sub_projects` / `has_actions`), **tags** (when present).

When the response reports a further page (`meta.pagination.has_next_page` true), the projection appends the **"more results available" signal** in My Roles (012)'s convention. When the practitioner owns no matching projects, the projection prints an explicit **empty-result line** rather than nothing.

Illustrative shape (exact labels/glyphs/layout are a build detail aligned with 012's list convention; the fields, their presence, stable order, and the signal are the contract):

```
proj_0123456789abcdef0123456789abcdef  [current]   Rebuild onboarding flow
  role: role_0123456789abcdef0123456789abcdef   sub-projects: yes   actions: yes   tags: q2
proj_00000000000000000000000000000001  [scheduled] Audit vendor list
  role: —   sub-projects: no   actions: no

… more results available — showing the first page (pagination lands with a later capability)
```

Empty result:

```
no projects
```

The token value never appears in the projection (it is a request header, not a response field).

---

## Interactions

- **Bare list**: `glassfrog my projects` resolves the connection (resolved once), issues `GET /me/projects` through the API client, and prints the first-page projection. Exit `0`.
- **Filtered list**: `glassfrog my projects --status current` validates `current` (shared `validateStatus`), adds `?status=current`, and prints the matching first page. An unsupported value never reaches the network.
- **Base URL override**: `glassfrog my projects --base-url https://example.test/api/v5` uses the flag value as the endpoint root; otherwise resolution falls through env → file → default (008).
- **Reading the result**: a caller reads stdout for the projection and `$?` for the outcome class; the two are orthogonal (004).
- **Single attempt, first page**: exactly one request (no retry, no paging walk); a further page is *signalled*, not fetched — Pagination (016) and rate-limit backoff (017) are later capabilities.

---

## Error Communication

`my projects` writes a controlled, **token-free** message to stderr on failure and signals the outcome class via the exit code. It **reuses Identity Read (011)'s `classifyClientError` and the error→exit-code mapping unchanged — it adds no new codes**. The mapping:

| Condition | Source outcome (from the API client, 010) | Exit code | Category (004) |
|---|---|---|---|
| Read succeeded, projection printed (incl. empty list) | `*Response` 2xx, body decoded | `0` | Success |
| Unsupported `--status` value | rejected locally by `validateStatus`, **no request issued** | `2` | UsageError |
| Unexpected positional argument | rejected by the arg validator, nothing ran | `2` | UsageError |
| No usable token (not authenticated) | `*AuthError{NoCredentials}` | `2` | UsageError |
| Bad base-URL configuration (flag/env/file) | base-URL error from `NewClientFromOS` | `2` | UsageError |
| Malformed `.glassfrogrc` credential file | `*AuthError{CredentialError}` | `1` | Internal error |
| 2xx body could not be decoded | `*DecodeError` | `1` | Internal error |
| API answered non-2xx (any status, incl. `400`/`401`/`429`) | `*ResponseError` (generic) | `3` | API error |
| API could not be reached (connection/DNS/TLS/timeout) | `*TransportError` | `6` | Network-unavailable |

- **Every error message states a next step (II)** — reuses 011's message shapes for the shared client errors; for the one class it owns — an unsupported `--status` — the message names the value and lists the supported set (exit `2`).
- **Generic non-2xx today**: every non-2xx maps to `3` (note `/me/projects` documents `400` for a bad request); per-status meaning is **API Error Extraction (015)** / **Rate-Limit Handling (017)**, which later split `401`/`403` → permission (`4`) and `429` → rate-limited (`5`) **without renumbering** `3`.
- **No secret in any message**: transport messages name the network-level cause; the non-2xx path reports the response status (never the `X-Auth-Token` request header); decode errors name a parse failure. Pinned by a token-never-in-output test across every branch.
- **Never zero on failure**: any non-success outcome exits non-zero; an unmapped category falls through to `1` (004's fail-safe).

---

## Consistency Notes

- **My Actions (`013/interface-cli.md` + `013/interface-spec.md`)**: `my projects` is the twin of `my actions` — same command shape, flags, projection convention, error mapping, and the **same shared `validateStatus` + status set**. The only differences: the resource is `Project`, the projection surfaces `has_sub_projects`/`has_actions`, and there is no `--include` flag (ADR-2).
- **Identity Read (`011`)**: reuses the persistent `--base-url` root flag, `classifyClientError`, and the `Outcome`/`ExitCode` categories (codes `3`/`6`) unchanged — a consumer, not an extender.
- **My Roles (`012`)**: a leaf under the `my` parent; reuses 012's `Pagination`, list envelope, and "more results available" signal convention.
- **Request Execution (`010/interface-spec.md`)**: consumes 010's `Client`/`Execute`/`Request` and typed errors unchanged; the projection decodes the 2xx body into the `glassfrog` list-of-`Project` envelope. Decode types and seam are in `014/interface-spec.md`.
- **No `--include`**: unlike `me --include roles` (011), `/me/projects` offers no `include` parameter, so embedding is not exposed; the `has_*` booleans signal child presence (ADR-2).
- **Deferred `--output json` / pagination flags**: a structured output mode and `--per-page`/`--cursor` are deferred (a future root flag / Pagination 016); this accord pins only the reshaped first-page projection.
- **PROJECT.md / VISION**: the projection is machine-readable and predictable (agent-legible output, VISION principle 3); the per-failure exit codes realise Action Transparency.
