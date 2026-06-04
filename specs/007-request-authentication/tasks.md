# Tasks: Request Authentication

**Feature**: 007-request-authentication
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unauthenticated-access/request-authentication.feature

---

## Dependency Graph

Phase 1: Auth outcome model (1 task, no phase dependencies) [Shared]
Phase 2: Auth round-tripper (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: request authentication is infrastructure serving all three user scenarios (attach-identity / refuse-when-no-credential / report-active-source) rather than any single one.
>
> **Cross-spec note**: 007 consumes `internal/auth.Resolution` from Credential Discovery (005) — now implemented and validated Ready on main (PR #11), so the consumed contract is real — and shares the API-client package with Connection Configuration (still modelled in a parallel session). The package name and the composition seam are `[ASSUMED]` (see plan ADR-1/ADR-2) — reconcile with Connection Configuration before integration.

---

## Branching Guidance

**Pipeline mode**: `spec/007-request-authentication/base` → `spec/007-request-authentication/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: Connection Configuration is being developed concurrently and will share the API-client package. Whichever lands first creates the package; the second joins it. Coordinate the package name and the transport-composition seam at integration rather than assuming a fixed shape.

---

## Phase 1: Auth outcome model [Shared]

- [x] **T001** [Shared] Create the API-client package and the typed `AuthError` + pure resolution-to-outcome mapping, with RED-first unit tests — 7 unit tests (3 branches + secret-hygiene + Kind-distinct + header constant); created `internal/apiclient` (007 lands first, ADR-2)
  - **Scope**: New API-client package (`internal/apiclient` proposed — `[ASSUMED]`, reconcile with Connection Configuration). Define `AuthError` carrying a `Kind` (`NoCredentials` or `CredentialError`); `CredentialError` wraps Discovery's underlying read/format error (unwrappable via `errors.As`/`Unwrap`). Add the pure mapping `authorize(res auth.Resolution, err error) → (token string, authErr *AuthError)`: `err != nil` → `CredentialError` wrapping it; `Source == None` → `NoCredentials`; `Source ∈ {Environment, File}` → return `Token`. Centralize the `X-Auth-Token` header name as a constant. No `net/http`, no command, no `os.Exit` yet. Consumes `internal/auth.Resolution` (005).
  - **Acceptance criteria**:
    - `authorize` returns the token with no `AuthError` for `Source` `Environment` and `File`
    - `Source: None` → `AuthError{Kind: NoCredentials}`, no token
    - A resolver error → `AuthError{Kind: CredentialError}` that unwraps to the original error via `errors.As`; the wrapped error names only the path, never the token
    - `NoCredentials` and `CredentialError` are distinguishable by `Kind`
    - The token value never appears in any `AuthError` string or other output; the `X-Auth-Token` constant is defined
    - RED-first unit tests cover all three branches plus secret-hygiene; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Auth outcome model; ADR-2 (API-client package), ADR-4 (typed `AuthError`)
  - **Interface references**: interface-spec.md — `AuthError` (produced output), Configuration (header name)
  - **Scenario references**: unauthenticated-access/request-authentication.feature: "A missing credential refuses the call", "A broken credential fails loudly without sending"
  - **Risk**: ⚠️ Depends on `internal/auth.Resolution` — now implemented on main (PR #11), so import the real `auth.Resolution` and bind `auth.Resolve` directly; the mapping is still tested with a fake `Resolution` for hermeticity.

## Phase 2: Auth round-tripper [Shared]

- [x] **T002** [Shared] Implement the `http.RoundTripper` that authenticates outgoing requests (resolve-once, attach-or-refuse), with RED-first unit tests over a fake base transport — 8 unit tests; `AuthTransport` clones the request before setting the header (net/http convention), resolve-once via `sync.Once`, `ActiveIdentity` exposes Source/Path
  - **Scope**: Add the auth round-tripper wrapping a base `http.RoundTripper` with an injected `func() (auth.Resolution, error)`, plus its constructor. Resolve once per invocation (cache the outcome). Per `RoundTrip`: resolver error → return `CredentialError` (base **not** called); `Source: None` → return `NoCredentials` (base **not** called); usable token → `req.Header.Set("X-Auth-Token", token)` verbatim, then delegate to the base transport and return its result unchanged. Expose `Source`/`Path` as the reportable active identity (never the token). No exit code, no `os.Exit`, no command surface. The composition seam with Connection Configuration is `[ASSUMED]`.
  - **Acceptance criteria**:
    - On a resolved token, `RoundTrip` sets `X-Auth-Token` to the token verbatim and delegates to the base transport, returning its response/error unchanged
    - On `Source: None`, `RoundTrip` returns `AuthError{NoCredentials}` and the base transport is never called
    - On a resolver error, `RoundTrip` returns `AuthError{CredentialError}` naming the file and the base transport is never called
    - Multiple `RoundTrip`s in one invocation resolve the credential once (resolver invoked once) and carry the same identity
    - The token value is never logged or placed in diagnostics; `Source`/`Path` are reportable
    - RED-first unit tests use a fake base `RoundTripper` (records the header and call count) and a fake resolver; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Auth round-tripper; ADR-1 (RoundTripper seam), ADR-3 (injected resolver)
  - **Interface references**: interface-spec.md — Auth round-tripper (Surface), per-request authenticate flow (Interactions), Error Communication
  - **Scenario references**: unauthenticated-access/request-authentication.feature: "The resolved token is attached to the request", "The token is attached verbatim", "Every request in an invocation carries the same identity", "The credential is resolved once per invocation", "A missing credential refuses the call", "A broken credential fails loudly without sending", "The active identity source is reported without the secret"
  - **Risk**: ⚠️ Seam mismatch with Connection Configuration (parallel session) — keep the round-tripper composable over any base transport so adapting the wiring is small; reconcile before integration.

## Phase 3: Executable acceptance [Shared]

- [x] **T003** [Shared] Make the 007 driving scenarios pass as executable acceptance via godog, driving the round-tripper with a fake base transport and a fake resolver — 8 behavioral scenarios pass (33 steps); 3 @validation scenarios kept @wip; own godog suite scoped to request-authentication.feature only (per LEARNINGS godog-suite-scoping)
  - **Scope**: Add godog step definitions for `features/unauthenticated-access/request-authentication.feature` (all three Rule blocks), driving the auth round-tripper with a fake base transport (capturing the attached header, whether it was called, and the call count) and a fake resolver supplying canned `Resolution`s and errors. Assert header attachment, base-not-called-on-failure, same identity across calls, resolve-once, active-source reporting, and diagnostic redaction. Remove `@wip` from the passing behavioral scenarios; keep the three `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 007 scenario (token attached / verbatim / same-identity / resolved-once / missing-credential refusal / broken-credential refusal / active-source reported / diagnostic redaction) has an executable, passing path
    - `@wip` removed from those scenarios; the three `@validation` scenarios (no-resolution-of-its-own / no-unauthenticated-send / token-never-in-output) keep `@wip`
    - No real network and no real home directory are touched — fakes only; the suite asserts no token value appears in captured output
    - `go build ./...`, `go vet ./...`, and the feature suite run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: unauthenticated-access/request-authentication.feature: all 007 Rule-block scenarios
  - **Risk**: ⚠️ Test isolation — fakes must not leak between scenarios; assert no token value appears in captured diagnostics.
