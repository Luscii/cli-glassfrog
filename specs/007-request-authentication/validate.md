# Validate: Request Authentication

**Feature**: 007-request-authentication
**Round**: 1 of 3
**Date**: 2026-06-04
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/unauthenticated-access/request-authentication.feature, PROJECT.md
**Implementation files**: 2 production files in `internal/apiclient/` (`auth.go`, `transport.go`) + 3 test files (`auth_test.go`, `transport_test.go`, `bdd_test.go`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (7 of 7 spec driving scenarios covered; the executable suite runs 8 — the 7 plus the plan-added "resolved once per invocation")

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| Resolved token is attached to the outgoing call | ✓ Covered | `transport.go:54` `RoundTrip` → `Header.Set(AuthHeaderName, token)` then `base.RoundTrip` (`:64–65`) |
| Active identity source is reported without exposing the secret | ✓ Covered | `transport.go:87` `ActiveIdentity` returns `Identity{Source, Path}`; `Identity` (`:71`) has no token field |
| The same identity applies across multiple calls in one invocation | ✓ Covered | `transport.go:43` `ensureResolved` caches via `sync.Once`; every `RoundTrip` reuses the cached resolution |
| No credentials — refuse to call | ✓ Covered | `auth.go:97` `authorize` → `NoCredentials`; `transport.go:56` returns before reaching base |
| Credential error — refuse to call and name the cause | ✓ Covered | `auth.go:94` `authorize` → `CredentialError` wrapping cause; `Error()` (`:73`) renders the path-only cause; distinct `Kind` |
| Token is attached verbatim | ✓ Covered | `transport.go:64` `Header.Set` with the raw token — no trim/re-encode |
| Token is redacted from request diagnostics | ✓ Covered (contextual — see note) | `transport.go:76` `Identity.String()` renders Source/Path only; 007 emits no diagnostics itself |

**Note (lower-confidence, not a finding)**: the "redacted from request diagnostics" scenario is satisfied *structurally*. 007 produces no verbose/trace output of its own (no `log`/`fmt.Print` — verified by grep), so there is no diagnostic emitter here that could leak the header. 007's contribution to safe diagnostics — a reportable surface (`Identity`) that carries Source/Path and never the token — is present and correct. The actual rendering of request diagnostics belongs to the consuming command / Connection Configuration (downstream), consistent with spec § Non-Behaviors (007 "must not own the HTTP transport") and the Integration Boundaries. This is conformant to what 007 owns; flagged transparently because the end-to-end diagnostic surface lives outside this capability.

---

## Acceptance Criteria

**Status**: Pass (all criteria for the 3 checked tasks met)

| Task | Status | Evidence |
|---|---|---|
| T001 — typed `AuthError` + pure mapping | ✓ Met | `authorize` returns token for Environment/File (`auth.go:100`), `NoCredentials` for None (`:97`), `CredentialError` wrapping the cause for errors (`:94`); `Unwrap` (`:82`) enables `errors.As`; Kinds distinct (`:37–44`); `AuthHeaderName` constant (`:29`); token absent from `Error()` (`:69–78`) |
| T002 — auth round-tripper | ✓ Met | Verbatim set + delegate, response unchanged (`transport.go:63–65`); base not called on None/error (`:55–57`); resolve-once via `sync.Once` (`:43–48`); `Source`/`Path` reportable via `ActiveIdentity` (`:87`); fake-base + fake-resolver unit tests present |
| T003 — executable acceptance | ✓ Met | 8 behavioral scenarios pass / 33 steps (`go test ./internal/apiclient -run TestFeatures`); `@wip` removed from behavioral, kept on 3 `@validation`; suite scoped to `request-authentication.feature` only; fakes only — no real network/home |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| Auth round-tripper (base `http.RoundTripper` + injected resolver → `http.RoundTripper`) | ✓ Conformant | `NewAuthTransport(base, resolve) *AuthTransport`; compile-time `var _ http.RoundTripper = (*AuthTransport)(nil)` (`transport.go:97`) |
| Resolver seam — consumed `auth.Resolution{Token, Source, Path}` | ✓ Conformant | `authorize` consumes `Resolution`; Token attached on Environment/File, None refuses, Path surfaced via `Identity` |
| `AuthError` — produced output (`Kind` enum; wrapped cause unwrappable; empty for NoCredentials) | ✓ Conformant | `auth.go:64–82` — `Kind` + unexported `cause`, `Unwrap` returns nil for NoCredentials |
| Configuration — `X-Auth-Token` constant; identity once per invocation | ✓ Conformant | `AuthHeaderName` (`auth.go:29`); `sync.Once` cache (`transport.go:27,44`) |
| Per-request authenticate flow (steps 1–4) | ✓ Conformant | `RoundTrip` order matches: resolve-once → error → None → set+delegate (`transport.go:54–65`) |
| Error Communication — emits no exit code, prints nothing | ✓ Conformant | No `os.Exit`, no `fmt.Print`, no `log` in production files (verified by grep) |

---

## Non-Behavior Absence

**Status**: Pass (all 7 non-behaviors absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not resolve/read/search/choose credential sources | ✓ Absent | Production code imports only `fmt`, `net/http`, `sync`, `internal/auth` (the type). No `os`/env/file reads; `auth.Resolve` never called (only named in a comment). Resolver is injected. |
| Must not own HTTP transport (base URL, retries, timeouts, parsing) | ✓ Absent | `AuthTransport` wraps a base and returns its result unchanged; owns no client config |
| Must not interpret `401`/`403` responses | ✓ Absent | `transport.go:65` returns `base.RoundTrip` result unchanged; no status inspection |
| Must not decide exit code or message | ✓ Absent | No `os.Exit`, no code; returns typed `AuthError` for the consumer to map |
| Must not send an unauthenticated fallback | ✓ Absent | Base reached only after `authorize` succeeds (`transport.go:55–58, 65`); no path sends without the header |
| Must not print/log/expose the token | ✓ Absent | No `log`/`fmt.Print`; token appears only in `Header.Set`; `Identity`/`AuthError` redact |
| Must not prompt for a token | ✓ Absent | No stdin/prompt code |

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation, independently of the driving-scenario pass)

| Scenario (held out from the Builder) | Status | Trace |
|---|---|---|
| The token value never appears in produced output | ✓ Satisfied | Token surfaces only as the wire header (`transport.go:64`). `AuthError.Error()` for NoCredentials carries no token (`auth.go:72`); for CredentialError it renders Discovery's path-only cause (`:74`; `auth.ReadError`/`FormatError` name only the path). `Identity.String()` (`:76`) has no token. Unit test `TestAuthError_NeverContainsTheToken` + BDD `thenTokenNotInOutput` assert it. |
| No request is ever sent unauthenticated | ✓ Satisfied | `RoundTrip` is the only call site of `base.RoundTrip`, guarded by `authorize`: base is reached only when `authErr == nil` (a token is present). None/error return before the base (`transport.go:55–58`). Unit tests assert `base.calls == 0` on None and on error. Structural, not disciplinary. |
| Authentication performs no resolution of its own | ✓ Satisfied | Grep confirms production code reads no environment or file and never calls `auth.Resolve`; the injected resolver is the sole source of the token. The `Resolution` type is the only thing imported from `internal/auth`. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 3 held-out validation scenarios trace to clear code paths through independent inspection. The implementation conforms to its specification: the resolved token is attached as `X-Auth-Token` verbatim and the request delegated; absence and credential errors refuse the call with distinct typed outcomes and the base transport is never reached unauthenticated; the active identity is reportable as Source/Path without the secret; and 007 owns neither resolution, transport, response interpretation, nor exit-code classification.

One contextual note (not a finding): the request-diagnostic redaction scenario is satisfied structurally — 007 emits no diagnostics itself and exposes only a redacted reportable surface; the end-to-end verbose-output rendering belongs to the downstream consuming command / Connection Configuration. This is conformant to 007's owned scope.

Two `[ASSUMED]` items carried from the plan remain open as cross-capability coordination (outside conformance): the package name (`internal/apiclient`) and the transport-composition seam, both to reconcile with Connection Configuration. These are noted for integration, not conformance gaps.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (PR #20 is open). At integration with Connection Configuration, reconcile the `[ASSUMED]` package name and composition seam; the three `@validation` scenarios remain `@wip` by design and need no further action here — they were the held-out set this validation traced.
