# Interface Accord: Identity Read — CLI

**Feature**: 011-identity-read
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (the `me` command surface); ADR-2 (persistent root `--base-url` flag), ADR-4 (error→exit-code mapping), ADR-5 (reshaped projection, request-time `--include` validation).

---

## Surface

This accord defines the **`me` command** — the first `glassfrog` command that calls the live API. It prints the authenticated actor the token resolves to, with its organization and membership.

### Command

```
glassfrog me [--include <target>]
```

- A top-level leaf command. Takes **no positional arguments** — an unexpected positional is a usage error (the dispatch invalid-input convention, 002).
- **Output goes to stdout**; diagnostics and error messages go to stderr; the outcome class is also signalled by the exit code (see Error Communication).

### Flags

| Flag | Scope | Value | Description |
|---|---|---|---|
| `--include` | local to `me` | `roles` | Opt-in: embed the requester's roles in the same read (maps to the API's `?include=roles`). Default absent → identity only. An unsupported target is rejected before any request (see Error Communication). |
| `--base-url` | **persistent, on the root** | a URL | The Glassfrog API base URL — the highest-precedence rung of base-URL resolution (008: flag → `GLASSFROG_BASE_URL` → `.glassfrogrc base_url` → built-in default). Registered once on the root and inherited by every API command (ADR-2); its name/usage come from the `apiclient.FlagBaseURL` constant. |

### Output — the identity projection (success, stdout)

On a `200`, `me` prints a **reshaped, predictable projection** (not raw JSON — a structured `--output json` mode is a deferred future capability, see Consistency Notes). The projection surfaces these facts, each on its own labelled line, in a stable order; the **id values are always present** (they are the machine-actionable handles an agent uses in follow-up calls):

- **actor** — name, kind (`human` | `agent`), and id (`per_…` / `agt_…`)
- **organization** — name and id (`org_…`)
- **access** — the membership access level (`admin` | `normal`)
- **roles** *(only when `--include roles` was given and the response carries roles)* — one line per role: name and id (`role_…`). When roles were requested but the actor fills none, the roles section is **omitted** (no empty list).

Illustrative shape (exact labels/glyphs are a build detail; the fields, their presence, and stable order are the contract):

```
actor:        Alice Smith (human) per_0123456789abcdef0123456789abcdef
organization: Acme (org_0123456789abcdef0123456789abcdef)
access:       admin
```

With `--include roles` (roles present):

```
actor:        Claude (agent) agt_0123456789abcdef0123456789abcdef
organization: Acme (org_0123456789abcdef0123456789abcdef)
access:       normal
roles:
  - Marketing Lead (role_0123456789abcdef0123456789abcdef)
  - Treasurer (role_00000000000000000000000000000001)
```

The token value never appears in the projection (it is a request header, not a response field).

---

## Interactions

- **Bare read**: `glassfrog me` resolves the connection (base URL + token, resolved once), issues `GET /me` through the API client, and prints the projection. Exit `0`.
- **Roles embed**: `glassfrog me --include roles` adds `?include=roles` to the request; the projection gains the roles section. The comprehensive roles surface is **My Roles (012)** — this embed is the API's convenience affordance, not a second roles command.
- **Base URL override**: `glassfrog me --base-url https://example.test/api/v5` uses the flag value as the endpoint root (highest precedence); otherwise resolution falls through env → file → default (008).
- **Reading the result**: a caller (CI / AI agent) reads stdout for the projection and `$?` for the outcome class; the two are orthogonal (004).
- **Single attempt**: the read makes exactly one request (no retry, no paging — `/me` is a single resource); rate-limit backoff and pagination are later capabilities (016/017).

---

## Error Communication

`me` writes a controlled, **token-free** message to stderr on failure and signals the outcome class via the exit code. It is the **first producer of reserved codes `3` and `6`** (004 published them; this command's typed-error classification gives them a producer). The mapping (plan ADR-4):

| Condition | Source outcome (from the API client, 010) | Exit code | Category (004) |
|---|---|---|---|
| Read succeeded, projection printed | `*Response` 2xx, body decoded | `0` | Success |
| Unsupported `--include` target | rejected locally, **no request issued** | `2` | UsageError |
| Unexpected positional argument | rejected by the arg validator, nothing ran | `2` | UsageError |
| No usable token (not authenticated) | `*AuthError{NoCredentials}` | `2` | UsageError |
| Bad base-URL configuration (flag/env/file) | base-URL error from `NewClientFromOS` | `2` | UsageError |
| Malformed `.glassfrogrc` credential file | `*AuthError{CredentialError}` | `1` | Internal error |
| 2xx body could not be decoded | `*DecodeError` | `1` | Internal error |
| API answered non-2xx (any status, incl. `401`/`403`/`429`) | `*ResponseError` (generic) | `3` | API error |
| API could not be reached (connection/DNS/TLS/timeout) | `*TransportError` | `6` | Network-unavailable |

- **Every error message states a next step (II)** — for each error class `me` owns, the stderr message names the cause *and* what to do:
  - no usable token → "not authenticated — run `glassfrog auth login` or set `GLASSFROG_TOKEN`" (exit `2`)
  - malformed credentials file (`CredentialError`) → names the file, then "fix or re-create it with `glassfrog auth login`" (exit `1`)
  - malformed base URL → names the source, then "correct `--base-url` / `GLASSFROG_BASE_URL` / the `.glassfrogrc` `base_url`" (exit `2`)
  - transport failure → names the cause, then "check connectivity — the API may be unreachable" (exit `6`)
- **Generic non-2xx today**: every non-2xx maps to `3` (the residual "general API error" bucket); `me` surfaces the **status code** but not a tailored next step. The *meaningful* per-status message and next step (e.g. `401` → re-authenticate, `429` → back off) are **API Error Extraction (015)'s** concern — 015 turns the non-2xx into a typed error carrying the API's status and detail, and Rate-Limit Handling (017) adds the `429` next step. They later **split** `401`/`403` → permission (`4`) and `429` → rate-limited (`5`) **without renumbering** `3` (Consistency Notes). 011 stays generic by design (spec fork 3).
- **No secret in any message**: transport messages name the network-level cause; the non-2xx path reports the response status (never the `X-Auth-Token` request header); decode errors name a parse failure. Pinned by a token-never-in-output test across every branch.
- **Never zero on failure**: any non-success outcome exits non-zero; an unmapped category falls through to `1` (004's fail-safe).

---

## Consistency Notes

- **Exit-Code Convention (`004/interface-cli.md`)**: this command realises codes `3` (API error) and `6` (network-unavailable) that 004 reserved "for the future API client," and reuses `0`/`1`/`2`. It adds the categories at the single `ExitCode` registry (see `011/interface-spec.md`); it never invents codes outside the published 0–6 set, and never renumbers.
- **Request Execution (`010/interface-spec.md`)**: `me` consumes 010's `Client`/`Execute`/`Request` and its typed errors unchanged; the projection decodes the 2xx body into `glassfrog.MeResponse`. The exact decode types and the seam are in `011/interface-spec.md`.
- **Credential Storage (`006/interface-cli.md`)**: `me`'s "not authenticated" path points the operator at `glassfrog auth login` (006's command). The `NoCredentials`→`UsageError` and `CredentialError`→`RuntimeError` mappings mirror 006's `runLogin` outcome shape.
- **My Roles (012)**: `--include roles` is the API embed; 012 owns the dedicated `my roles` read and grows the shared `glassfrog.Role` type.
- **Deferred `--output json`**: the spec defers a structured/verbatim output mode to a future capability; this accord pins only the reshaped projection. When `--output json` lands it will be a cross-cutting flag (likely persistent on the root, like `--base-url`), not a `me`-local concern.
- **PROJECT.md / VISION**: the projection is machine-readable and predictable (agent-legible output, VISION principle 3); the per-failure exit codes realise Action Transparency (the operator always learns the outcome class from `$?`).
