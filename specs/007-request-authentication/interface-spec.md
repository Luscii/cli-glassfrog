# Interface Accord: Request Authentication — Specification

**Feature**: 007-request-authentication
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture (the auth round-tripper) + ADR-1 (RoundTripper seam), ADR-3 (injected resolver), ADR-4 (typed `AuthError`), and the Integration Design `X-Auth-Token` contract toward the Glassfrog API.

---

This accord pins the contracts the rest of the request path depends on: the **auth round-tripper** Connection Configuration composes into its HTTP client, the **resolver seam** through which 007 consumes Credential Discovery (005), the **`AuthError`** the consuming command discriminates, and the **`X-Auth-Token`** header attached to outgoing calls. There is **no command or shell entry point** — 007 is consumed by composing its `RoundTripper`, not invoked — so the *invocation* and *instructional* surfaces are N/A. The composition seam and the package name are `[ASSUMED]`, to be reconciled with the Connection Configuration spec (modelled in parallel) before either ships; the header name (`X-Auth-Token`) is already pinned by PROJECT.md, not provisional.

---

## Surface

### Auth round-tripper (the seam) `[ASSUMED]`

The unit of consumption. An `http.RoundTripper` that wraps a base transport and authenticates each outgoing request. Connection Configuration supplies the base transport and assembles the `http.Client`; this round-tripper is layered in front of it.

| Construction input | Type | Description |
|---|---|---|
| base transport | `http.RoundTripper` | The next transport in the chain (Connection Configuration's), reached only on the authenticated branch. |
| resolver | `func() (auth.Resolution, error)` | The injected credential source (ADR-3). Production binds Credential Discovery's resolver (005); tests bind a fake. |

Returns an `http.RoundTripper`. The constructor name (e.g. `NewAuthTransport`) and the package it lives in (`internal/apiclient` proposed) are `[ASSUMED]` — reconcile with Connection Configuration.

### Resolver seam — consumed input

007 consumes Credential Discovery's `Resolution` (defined in 005's interface-spec); it never resolves itself.

| Field (from `auth.Resolution`) | Type | Use here |
|---|---|---|
| `Token` | string | Attached as the `X-Auth-Token` value when `Source` is `Environment` or `File`. Never rendered, logged, or placed in an error. |
| `Source` | enum (`Environment`, `File`, `None`) | `None` → no credential (refuse); otherwise → attach. |
| `Path` | string | Reportable as the active identity (with `Source`) for the operator-facing "acting as" surface. Never accompanies the token in output. |

### `AuthError` — produced output `[ASSUMED]` (name)

The typed, code-free failure 007 returns when it cannot authenticate. Carries a discriminable `Kind`; the consuming command maps it to an exit code (007 decides none).

| Field | Type | Description |
|---|---|---|
| `Kind` | enum | `NoCredentials` or `CredentialError`. |
| (wrapped cause) | error | For `CredentialError`, the Discovery read/format error — which names only the **path**, never the token. Unwrappable via `errors.As`/`Unwrap`. Empty for `NoCredentials`. |

| Kind | Means | Maps from |
|---|---|---|
| `NoCredentials` | No usable credential exists anywhere | `Resolution{Source: None}`, no error |
| `CredentialError` | A credentials file exists but is unreadable/unparseable | Discovery returned a typed read/format error |

### Configuration

| Item | Value | Notes |
|---|---|---|
| Header name | `X-Auth-Token` | Centralized constant; pinned by the Glassfrog v5 API scheme (PROJECT constraint) — fixed, not provisional. |
| Identity lifetime | once per invocation | The resolver is consulted once and the result cached for the invocation; resolution is deterministic (005), so every request in one invocation carries the same identity. |

---

## Interactions

**Composition**: Connection Configuration builds its base transport, wraps it with the auth round-tripper, and assembles the `http.Client`. Every request issued through that client passes through 007 first. (Whether the wrapping is done exactly this way is the `[ASSUMED]` seam.)

**Per-request authenticate flow** (`RoundTrip`):
1. Obtain the credential via the injected resolver (once per invocation; cached thereafter).
2. **Resolver returned an error** → return `AuthError{Kind: CredentialError}` wrapping it; **do not call the base transport**.
3. **`Source == None`** → return `AuthError{Kind: NoCredentials}`; **do not call the base transport**.
4. **`Source ∈ {Environment, File}`** → set request header `X-Auth-Token: <Token>` (verbatim — no trimming or re-encoding), then delegate to the base transport and return its result unchanged.

**Active-identity reporting**: `Source` and `Path` are the only parts safe to surface as the operator-facing "acting as" line. The `Token` is never rendered.

**Resolution precedence / configuration surface**: N/A here — env-vs-file precedence and the walk-up live entirely in Credential Discovery (005). 007 consumes the already-resolved outcome.

---

## Error Communication

007 emits **no exit code and prints nothing itself**; it returns outcomes for the consuming command and Exit-Code Convention (004) to map.

| Condition | Outcome |
|---|---|
| Usable token resolved | `X-Auth-Token` set; request delegated to the base transport; the transport's response/error is returned unchanged |
| No credential anywhere (`Source: None`) | `AuthError{Kind: NoCredentials}`; request **unsent** |
| Credentials file exists but unreadable/unparseable | `AuthError{Kind: CredentialError}` wrapping Discovery's path-only error; request **unsent**; distinct from `NoCredentials` |

**No unauthenticated send** (constraint, structural): there is no branch on which the base transport is reached without `X-Auth-Token` set — absence or error ends in an `AuthError` with the request unsent.

**Secret hygiene** (constraint, enforced): the token appears only as the header value (set via the request header API). It never appears in an `AuthError`, a log line, or any diagnostic/verbose output — diagnostics redact or omit the `X-Auth-Token` value.

**Response-side rejections out of scope**: a `401` (rejected token) or `403` (permission/premium) in the API *response* is **not** interpreted here — that belongs to the response-handling / Exit-Code path. 007 owns the outgoing direction only.

**Open downstream gap**: which exit code a cannot-authenticate outcome ultimately receives is unresolved — 004's frozen convention has no local-precondition code, and code 4 is reserved for API-side rejection. The consuming-command spec decides; flagged for `/score:clarify`. 007 is unaffected (it stays code-free).

---

## Consistency Notes

- **Consumes Credential Discovery (005)**: the `auth.Resolution{Token, Source, Path}` shape and the typed read/format errors are defined in 005's interface-spec; this accord refines only what 007 *does* with them. The token-never-rendered rule and the `Source`/`Path`-only reporting rule are carried from 005, not re-litigated.
- **Reconcile with Connection Configuration (parallel session)**: the composition seam (`RoundTripper` wrapping), the package name (`internal/apiclient` proposed), and the `net/http` substrate are `[ASSUMED]`. Mirrors the 005↔006 shared-contract reconciliation — settle before either ships; whichever capability lands first creates the package.
- **No CLI surface**: unlike Help & Version (003) and Credential Storage (006), 007 registers no command and has no `interface-cli.md` — it is a consumed library contract, hence a Specification accord only (like 005).
- **No `accords/` directory** exists in the project, so there are no established cross-spec accord patterns to align against; this accord stands on its own.
