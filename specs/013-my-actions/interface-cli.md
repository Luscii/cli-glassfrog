# Interface Accord: My Actions — CLI

**Feature**: 013-my-actions
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (the `my actions` leaf under the `my` parent); ADR-2 (request-time `--status` validation via shared `validateStatus`); ADR-3 (first page + "more available" signal, reusing 012's convention); ADR-4 (guard-registered leaf + injected seam + pure `runMyActions`/`formatMyActions`). Reuses 011's `--base-url` (ADR-2), `classifyClientError`, and error→exit-code mapping (ADR-3/4).

---

## Surface

This accord defines the **`my actions` command** — the dedicated self-service read that lists the actions owned by roles the authenticated practitioner fills.

### Command

```
glassfrog my actions [--status <status>]
```

- A **leaf under the `my` parent command** (introduced by My Roles, 012). Takes **no positional arguments** — an unexpected positional is a usage error (the dispatch invalid-input convention, 002).
- **Output goes to stdout**; diagnostics and error messages go to stderr; the outcome class is also signalled by the exit code (see Error Communication).

### Flags

| Flag | Scope | Value | Description |
|---|---|---|---|
| `--status` | local to `my actions` | one of `archived` `cancelled` `completed` `current` `scheduled` `someday` `waiting` | Optional filter mapped to the API's `?status=`. **Validated before any request** (ADR-2): an unsupported value is rejected as a usage error naming the value and the supported set, with no request issued. Absent → no status constraint. |
| `--base-url` | **persistent, on the root** (011) | a URL | The Glassfrog API base URL — highest-precedence rung of base-URL resolution (008). Registered once on the root by Identity Read (011) and inherited here; `my actions` reads its value, registers nothing. |

### Output — the actions projection (success, stdout)

On a `200`, `my actions` prints a **reshaped, predictable projection** (not raw JSON — a structured `--output json` mode is a deferred cross-cutting capability, see Consistency Notes). It renders **one entry per action** in the order the API returned them; each entry surfaces a stable set of fields, and the **id is always present** (the machine-actionable handle an agent uses in follow-up calls):

- **id** (`actn_…`), **status** (one of the status enum), **description** (the action text; `—` when null), **role** (the owning `role_…` id), **tags** (when present).

When the response reports a further page (`meta.pagination.has_next_page` true), the projection appends a **"more results available" signal** in the form My Roles (012) establishes for paginated reads (it surfaces that the list is the first page only; the multi-page walk is Pagination, 016). When the practitioner owns no matching actions, the projection prints an explicit **empty-result line** rather than nothing.

Illustrative shape (exact labels/glyphs/layout are a build detail aligned with 012's list convention; the fields, their presence, stable order, and the signal are the contract):

```
actn_0123456789abcdef0123456789abcdef  [current]   Review PR #6818
  role: role_0123456789abcdef0123456789abcdef   tags: marketing, q2
actn_00000000000000000000000000000001  [waiting]   Draft Q2 plan
  role: role_0123456789abcdef0123456789abcdef

… more results available — showing the first page (pagination lands with a later capability)
```

Empty result:

```
no actions
```

The token value never appears in the projection (it is a request header, not a response field).

---

## Interactions

- **Bare list**: `glassfrog my actions` resolves the connection (base URL + token, resolved once), issues `GET /me/actions` through the API client, and prints the first-page projection. Exit `0`.
- **Filtered list**: `glassfrog my actions --status current` validates `current` against the status set, adds `?status=current`, and prints the matching first page. An unsupported `--status` value never reaches the network (ADR-2).
- **Base URL override**: `glassfrog my actions --base-url https://example.test/api/v5` uses the flag value as the endpoint root (highest precedence); otherwise resolution falls through env → file → default (008).
- **Reading the result**: a caller (CI / AI agent) reads stdout for the projection and `$?` for the outcome class; the two are orthogonal (004).
- **Single attempt, first page**: the read makes exactly one request (no retry, no paging walk); a further page is *signalled*, not fetched — Pagination (016) and rate-limit backoff (017) are later capabilities.

---

## Error Communication

`my actions` writes a controlled, **token-free** message to stderr on failure and signals the outcome class via the exit code. It **reuses Identity Read (011)'s `classifyClientError` and the error→exit-code mapping unchanged — it adds no new codes** (011 already gave codes `3` and `6` their producer). The mapping:

| Condition | Source outcome (from the API client, 010) | Exit code | Category (004) |
|---|---|---|---|
| Read succeeded, projection printed (incl. empty list) | `*Response` 2xx, body decoded | `0` | Success |
| Unsupported `--status` value | rejected locally by `validateStatus`, **no request issued** | `2` | UsageError |
| Unexpected positional argument | rejected by the arg validator, nothing ran | `2` | UsageError |
| No usable token (not authenticated) | `*AuthError{NoCredentials}` | `2` | UsageError |
| Bad base-URL configuration (flag/env/file) | base-URL error from `NewClientFromOS` | `2` | UsageError |
| Malformed `.glassfrogrc` credential file | `*AuthError{CredentialError}` | `1` | Internal error |
| 2xx body could not be decoded | `*DecodeError` | `1` | Internal error |
| API answered non-2xx (any status, incl. `401`/`429`) | `*ResponseError` (generic) | `3` | API error |
| API could not be reached (connection/DNS/TLS/timeout) | `*TransportError` | `6` | Network-unavailable |

- **Every error message states a next step (II)** — `my actions` reuses 011's message shapes for the shared client errors (no usable token → "not authenticated — run `glassfrog auth login` or set `GLASSFROG_TOKEN`"; malformed base URL → name the source and how to correct it; transport failure → name the cause and "check connectivity"). For the one error class it owns — an unsupported `--status` — the stderr message names the unsupported value and lists the supported set (exit `2`).
- **Generic non-2xx today**: every non-2xx maps to `3`; `my actions` surfaces the status code but not a tailored next step. Per-status meaning (e.g. `401` → re-authenticate, `429` → back off) is **API Error Extraction (015)** / **Rate-Limit Handling (017)**, which later split `401`/`403` → permission (`4`) and `429` → rate-limited (`5`) **without renumbering** `3`.
- **No secret in any message**: transport messages name the network-level cause; the non-2xx path reports the response status (never the `X-Auth-Token` request header); decode errors name a parse failure. Pinned by a token-never-in-output test across every branch.
- **Never zero on failure**: any non-success outcome exits non-zero; an unmapped category falls through to `1` (004's fail-safe).

---

## Consistency Notes

- **Identity Read (`011/interface-cli.md` + `011/interface-spec.md`)**: `my actions` reuses 011's persistent `--base-url` root flag, the shared `classifyClientError`, and the `Outcome`/`ExitCode` categories (codes `3`/`6`) **unchanged** — it is a *consumer* of that surface, not an extender. Its projection follows the same reshaped-not-JSON, id-always-present convention as `formatMe`.
- **My Roles (`012`)**: `my actions` is a leaf under the `my` parent command 012 introduces, and reuses 012's `Pagination` type, list envelope, and "more results available" signal convention. The `actions` leaf and the `roles` leaf are siblings under `my`.
- **Request Execution (`010/interface-spec.md`)**: `my actions` consumes 010's `Client`/`Execute`/`Request` and typed errors unchanged; the projection decodes the 2xx body into the `glassfrog` list-of-`Action` envelope. The exact decode types and seam are in `013/interface-spec.md`.
- **My Projects (014 — twin)**: reuses `validateStatus` + the status set (013 introduces them) and the same projection/seam shape over `glassfrog.Project`.
- **Deferred `--output json`**: the spec defers a structured/verbatim output mode to a future cross-cutting capability (a persistent root flag per 011); this accord pins only the reshaped projection.
- **Deferred pagination flags**: `--per-page` / `--cursor` and the multi-page walk are Pagination (016); `my actions` exposes neither and fetches one page.
- **PROJECT.md / VISION**: the projection is machine-readable and predictable (agent-legible output, VISION principle 3); the per-failure exit codes realise Action Transparency (the operator always learns the outcome class from `$?`).
