# Plan: Request Authentication

**Feature**: 007-request-authentication
**Role**: Shaper
**Inputs**: spec.md (007-request-authentication), PROJECT.md, `.score/memory/DECISIONS.md` (relevant precedent: Go self-contained binary, cobra tree, fail-loud guard, explicit `main` wiring, outcome-category split, `internal/auth` package, `Resolution{Token,Source,Path}` shape, injected resolver roots, "007 lives with the API client not in `internal/auth`", "auth stays command- and network-free"; this plan appends the request-attachment precedent), `.score/memory/LEARNINGS.md` (cobra findings — not relevant; 007 registers no command), 005 interface-spec.md (the `Resolution` contract consumed here). No SOUL.md.

---

## System Architecture

Request Authentication is the point where the resolved identity meets the wire: it ensures every outgoing Glassfrog API request carries the `X-Auth-Token` header for the org + person the token is scoped to. It is deliberately narrow — it decorates the *outgoing request* with the auth header and short-circuits when no usable credential exists. It does **not** own the HTTP client (base URL, timeouts, retries, response parsing, pagination); that is **Connection Configuration**, a sibling capability being modelled in parallel.

Per DECISIONS precedent, token *resolution* lives in `internal/auth` (Credential Discovery 005) and `internal/auth` stays command- and network-free; request-attachment "lives with the API client, not in `internal/auth`." So 007's code lives in the API-client/transport package alongside Connection Configuration — not in `internal/auth`, which it only *consumes*.

The parts:

- **Auth round-tripper** — an `http.RoundTripper` that wraps a *base* transport (provided by Connection Configuration). Before delegating, it obtains the credential and either sets `X-Auth-Token` and delegates to the base transport, or returns a typed `AuthError` **without delegating** — so there is no code path on which an API request leaves the process unauthenticated (the spec's load-bearing Fail-Safe guarantee).
- **Resolver seam** — an injected `func() (auth.Resolution, error)` bound in production to Credential Discovery's resolver (005). 007 never reads the environment or filesystem itself (validation scenario: "performs no resolution of its own"); it only consumes Discovery's output.
- **Outcome mapping** — a pure function over the `(Resolution, error)` Discovery returns:
  - `error != nil` → `AuthError{Kind: CredentialError}` wrapping Discovery's path-only error (a broken/unreadable file) — do not send.
  - `Source == None` → `AuthError{Kind: NoCredentials}` — do not send.
  - `Source ∈ {Environment, File}` → set the header to `Token`, delegate. The active identity (`Source`/`Path`, never `Token`) is reportable.

```
API-client package  (e.g. internal/apiclient — [ASSUMED]; shared with Connection Configuration)

  Connection Configuration → base http.RoundTripper (base URL, timeouts, retries)   [other session]
                                       ▲
                                       │ wraps
  007 authRoundTripper{ base, resolve } │
    RoundTrip(req):
      res, err := resolve()                         ── injected; prod = auth.Resolve (005)
      switch:
        err != nil          → return AuthError{CredentialError, cause: err}   (base NOT called)
        res.Source == None  → return AuthError{NoCredentials}                 (base NOT called)
        else                → req.Header.Set("X-Auth-Token", res.Token); return base.RoundTrip(req)

  consumes  ──►  internal/auth.Resolution{Token, Source, Path}   (005, command/network-free)
```

---

## Architecture Decisions

### ADR-1: Request Authentication is an `http.RoundTripper` that wraps Connection Configuration's base transport

**Context**: The spec scopes 007 to "attach the resolved token as the `X-Auth-Token` header on outgoing calls" while Connection Configuration owns the transport. The load-bearing requirement is Fail-Safe: *no request may ever be sent unauthenticated* (spec validation scenario). The seam between the two capabilities is the `[ASSUMED]` coordination item the developer deferred during define. The project is stdlib-first (CONSTITUTION XII, no third-party deps) over a REST API, so `net/http` is the transport substrate.

**Options considered**:
1. **RoundTripper middleware** — 007 provides a `RoundTripper` that wraps the base transport; Connection Configuration composes it into its `http.Client`. The auth layer sits *in front of* the base transport and only delegates when authenticated, so an unauthenticated send is structurally impossible. Idiomatic Go for per-request concerns.
2. **Request-decorator function** — 007 exposes `Authorize(*http.Request) error`; Connection Configuration calls it before each send. Simpler signature, but the Fail-Safe guarantee moves to the *caller* (Connection Configuration must remember to call it and to abort on error) — easy to bypass.
3. **Client pulls the header value** — 007 exposes "give me the header"; the client sets it. Same bypass risk as option 2, and it leaks the token across one more boundary.

**Decision**: Option 1 — a `RoundTripper` wrapper. 007 owns only the auth-decorating transport; Connection Configuration supplies the base `RoundTripper` and builds the `http.Client` (base URL, timeouts, retries). Because the base transport is reached *only* on the authenticated branch, "no unauthenticated request is ever sent" is a structural property, not a discipline the caller must maintain.

**The composition seam is `[ASSUMED]`** pending reconciliation with the Connection Configuration spec: whether Connection Configuration wraps its base transport with 007's `RoundTripper`, or some other wiring. Flagged in the handoff (mirrors the 005↔006 shared-contract reconciliation).

**Consequences**: 007 stays out of transport ownership while still guaranteeing Fail-Safe. It presumes `net/http` — grounded by the no-deps constitution + REST, but recorded as the seam assumption. If Connection Configuration lands on a different seam, this reconciles through plan enrichment.

### ADR-2: 007 lives in the API-client package, not `internal/auth`; whichever of 007/Connection-Configuration lands first creates it

**Context**: DECISIONS pins it twice — "Request Authentication consumes the resolved credential (it lives with the API client, not in `internal/auth`)" and "auth stays command- and network-free." Connection Configuration is unbuilt (parallel session). 005 (the dependency) is now implemented and validated Ready (PR #11), so `internal/auth` already exists with the `Resolution`/`Resolve()` contract 007 consumes; the API-client package is the one still to be created.

**Options considered**:
1. **API-client package (e.g. `internal/apiclient`), created by whichever of 007 / Connection Configuration lands first** — exactly the 005↔006 "whichever lands first creates the shared module" precedent. Keeps `internal/auth` network-free.
2. **Put 007 in `internal/auth`** — rejected: directly violates the established precedent and would pull the network boundary into the credential-file package.

**Decision**: Option 1 — 007's round-tripper lives in the API-client package next to Connection Configuration. The exact package name is `[ASSUMED]` (`internal/apiclient` proposed); whichever capability lands first creates the package, the other joins it.

**Consequences**: `internal/auth` stays a pure file/resolution concern. The package name reconciles with Connection Configuration. 007 imports `internal/auth` for the `Resolution` type; the dependency direction (client → auth) matches the layering.

### ADR-3: Consume Discovery through an injected resolver seam; never re-resolve

**Context**: The spec forbids 007 from resolving credentials itself (non-behavior + validation scenario). 005 already exposes a pure resolver plus a thin production wrapper binding `os.Getwd`/`os.UserHomeDir`/`os.Getenv` (DECISIONS: "filesystem/env-dependent resolvers inject their roots"). CONSTITUTION IV wants hermetic, RED-first tests.

**Options considered**:
1. **Inject a `func() (auth.Resolution, error)`** into the round-tripper; production binds 005's resolver wrapper, tests bind a fake returning canned `Resolution`s and errors. Hermetic; 007's logic is testable before 005 is even built; structurally guarantees "no resolution of its own."
2. **Call 005's package-level resolver directly** inside `RoundTrip` — rejected: couples 007's tests to the real filesystem/env (and risks reading the developer's real `~/.glassfrogrc`), defeating the injected-roots precedent.

**Decision**: Option 1 — an injected resolver function. The only source of the token is this seam's output; 007 reads no environment variable and no file directly.

**Consequences**: 007's branching (token / none / error) is unit-tested against fake resolutions with no I/O. Extends 005's injected-seam precedent to the consumer side. Production wiring binds the real 005 resolver where the `http.Client` is assembled.

### ADR-4: Map the resolution to a typed, code-free `AuthError` (NoCredentials vs CredentialError); the consuming command classifies the exit code

**Context**: The spec says 007 reports its outcome but decides no exit code or message — Exit-Code Convention (004) + the consuming command own that. 004's frozen convention has *no* dedicated code for a *local* "cannot authenticate" precondition (its code 4 is for an *API*-side auth rejection, which 007 explicitly does not interpret). The caller will consume failures through a `switch`/`errors.As`, so it must be able to discriminate a validation/precondition failure from an unexpected bug (per the named-error precedent the project already follows).

**Options considered**:
1. **A typed `AuthError` with a `Kind` discriminator** (`NoCredentials`, `CredentialError`), `CredentialError` wrapping Discovery's path-only error. 007 emits no `Outcome` category and no code; the command does `errors.As` and maps to a category for `ExitCode`. Keeps 007 code-free (continues the 002/005 "producer returns typed outcome, consumer maps" split) and lets the command give different guidance per kind ("no token — run login" vs "credentials file is broken at <path>").
2. **Reuse 005's error directly** — rejected: conflates "file is broken" with "no credential configured" (005 already models `Source: None` as a *non*-error), and gives the command nothing to discriminate the no-credential case on.
3. **007 picks the `Outcome` category itself** — rejected: violates the spec non-behavior; 007 must not decide the code.

**Decision**: Option 1 — a typed `AuthError{Kind}`. `NoCredentials` and `CredentialError` are distinct; `CredentialError` wraps 005's typed read/format error (which already names only the path, never the token). 007 returns this from `RoundTrip` without sending; the consuming command classifies it downstream.

**Consequences**: 007 stays exit-code-free and secret-clean. **Open downstream question (behavioral gap, not 007's to resolve):** which exit code a "cannot authenticate" outcome ultimately receives — 004's convention has no dedicated precondition code, and code 4 is reserved for *API*-side rejection. Flagged for the consuming-command spec / `/score:clarify`; 007 is unaffected because it only surfaces the typed outcome.

---

## Integration Design

- **Credential Discovery (005 — upstream dependency, internal contract)**: 007 imports `internal/auth.Resolution` and consumes the injected resolver. Branches: token (`Environment`/`File`) → attach; `None` → `NoCredentials`; resolver error → `CredentialError`. 007 never re-resolves.
- **Connection Configuration (transport sibling — `[ASSUMED]` seam, modelled in parallel)**: provides the base `http.RoundTripper` and assembles the `http.Client`. 007's `RoundTripper` wraps the base; the precise composition is the reconciliation item. Connection Configuration owns base URL, timeouts, retries, response handling.
- **Glassfrog API (downstream system)**: receives `X-Auth-Token` and authorizes per the scoped org + person. 007 acts only on the *outgoing* request; it does not interpret the response (`401`/`403` belong to the response/Exit-Code path).
- **Exit-Code Convention (004 — downstream)**: a cannot-authenticate outcome maps to a non-zero code, but the classification belongs to the consuming command, not 007 (ADR-4).

---

## Cross-cutting Concerns

**Fail-Safe (CONSTITUTION III)**: the base transport is reached only on the authenticated branch — an absent or broken credential ends in a typed `AuthError` with the request unsent. A broken credential (`CredentialError`) is kept distinct from absence (`NoCredentials`); neither is masked as the other.

**Secret hygiene (CONSTITUTION II; spec non-behavior)**: the token appears only as the `X-Auth-Token` header value, set via `Header.Set`. It is never logged and never placed in an `AuthError` (`CredentialError` carries 005's path-only wrapped error). Any request diagnostics/verbose output must redact or omit the header value — asserted by a validation scenario.

**Testing (CONSTITUTION IV)**: RED-first. A fake base `RoundTripper` records whether it was called and with what header; an injected fake resolver supplies canned `Resolution`s and errors. Tests assert: header set verbatim on success; base **never** called on `None`/error; the same identity is applied across multiple requests in one invocation; the token never appears in output or error strings. Hermetic — no real network, and (via the injected resolver) no real home directory.

**Configuration**: the header name `X-Auth-Token` is a centralized constant (pinned by the Glassfrog v5 spec — a fixed PROJECT constraint, not provisional). Identity is resolved **once per invocation** (cached on first use): 005 resolution is deterministic, so caching both honours the "single identity per invocation" assumption and avoids repeating the filesystem walk on every call.

**No command surface**: 007 registers no cobra command and prints nothing itself; the command that actually triggers API calls is a future spec. The LEARNINGS cobra findings do not apply.

---

## Implementation Strategy

Three phases, linear. 007 depends on `internal/auth.Resolution` from 005; the build order is **005 → 007** (the FEATURE-MODEL depends-on edge) and 005 has now landed on main, so the consumed `Resolution`/`Resolve()` contract is real and importable rather than a pinned-but-unbuilt assumption.

- **Phase 1 — Auth outcome model (pure)**: define `AuthError{Kind}` (`NoCredentials`, `CredentialError`) and the pure mapping `authorize(res auth.Resolution, err error) → (token string, authErr *AuthError)` over Discovery's three outcomes. RED-first unit tests for each branch plus secret-hygiene (token never in the error). *Depends on: `internal/auth.Resolution` existing (005, or the pinned type).*
- **Phase 2 — Auth round-tripper**: the `http.RoundTripper` wrapping a base transport with an injected resolver; resolve-once cache; header attach + delegate on success; return `AuthError` without delegating on failure. RED-first unit tests with a fake base transport and fake resolver (header set / base-not-called-on-failure / same-identity-across-calls / token-redacted). *Depends on: Phase 1.*
- **Phase 3 — Executable acceptance**: godog step definitions for the 007 driving scenarios (token attached, source reported without the secret, same identity across calls, no-credentials refusal, credential-error refusal, verbatim attachment, diagnostic redaction). *Depends on: Phase 2.*

---

## Risks

- **Seam mismatch with Connection Configuration** (medium likelihood, medium impact): the parallel session may choose a different composition than the `RoundTripper` wrapper (ADR-1). Mitigation: the seam is `[ASSUMED]` and flagged for reconciliation; a `RoundTripper` composes into any `http.Client`, so the adaptation is small if the wiring differs. Reconcile during interface / via plan enrichment.
- **Build-order coupling with 005** (low likelihood, medium impact — substantially resolved): 007 needs `internal/auth.Resolution`, which is now implemented and validated Ready (PR #11) with a contract matching 007's assumptions. Mitigation: consume the real `auth.Resolution`/`auth.Resolve`; 007's logic stays independently testable via the injected fake resolver (ADR-3). If 005's `Resolution` shape shifts, 007's mapping updates at one site.
- **Exit-code category for "cannot authenticate" is undefined** (medium likelihood, low impact): 004's frozen convention has no local-precondition code, and code 4 is reserved for API-side rejection. Mitigation: 007 stays code-free (typed `AuthError`); the classification is a downstream decision flagged for the consuming-command spec / clarify. Does not block 007.
- **Secret leakage on the request side** (low likelihood, high impact): request logging/tracing could dump the `X-Auth-Token` header. Mitigation: header set only via `Header.Set`, never logged; redaction asserted by a validation scenario; `AuthError` carries no token.
- **`net/http` assumption** (low likelihood, low impact): the `RoundTripper` seam presumes the stdlib client. Mitigation: stdlib + REST makes `net/http` near-certain under CONSTITUTION XII; recorded as part of the ADR-1 seam assumption to confirm with Connection Configuration.

---

## What This Plan Does Not Cover

- **The HTTP client itself** — base URL, timeouts, retries, response parsing, pagination — Connection Configuration owns these.
- **Interpreting `401`/`403` responses** — the response-handling / Exit-Code path, not 007 (request-side only).
- **The exit code for a cannot-authenticate outcome** — the consuming command + Exit-Code Convention (004); 007 surfaces a typed outcome only (ADR-4, flagged behavioral gap).
- **The command that makes API calls** — a future spec; 007 has no command surface.
- **Package name and the composition seam** — `[ASSUMED]`; reconcile during interface and with the Connection Configuration spec. (The header name `X-Auth-Token` is already pinned by PROJECT.md, not provisional.)
