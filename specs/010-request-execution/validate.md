# Validate: Request Execution

**Feature**: 010-request-execution
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/no-shared-api-client/request-execution.feature, PROJECT.md
**Implementation files**: 2 production + 3 test files in `internal/apiclient/` — `client.go` (Request, Client, NewClient, NewClientFromOS, requestTimeout), `execute.go` (Execute, Response, TransportError, ResponseError, DecodeError); `client_test.go`, `execute_test.go`, `request_execution_bdd_test.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. All 4 held-out validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (10 of 10 behavioral scenarios covered)

Every driving scenario referenced by the checked tasks (T001–T003) has an identifiable implementation code path. Each is also pinned by an executable godog scenario in `TestRequestExecutionFeatures` and supporting unit tests.

| Scenario | Status | Implementation |
|---|---|---|
| 2xx response decoded into a supplied target | ✓ Covered | `execute.go:Execute` — 2xx branch, `json.NewDecoder(resp.Body).Decode(out)` when `out != nil` |
| 2xx response with no decode target (bodyless) | ✓ Covered | `execute.go:Execute` — `out == nil` drains via `io.Copy(io.Discard, …)`, returns `*Response` |
| Identity carried by the authenticated transport | ✓ Covered | `client.go:NewClient` wires `NewAuthTransport(base, replay)`; `execute.go` sends via `httpClient.Do` — header attached by 007, not by Execute |
| Transport failure at the wire | ✓ Covered | `execute.go:Execute` — non-`AuthError` `Do` error → `&TransportError{cause}` |
| Non-2xx response short-circuited | ✓ Covered | `execute.go:Execute` — status `< 200 || >= 300` → `&ResponseError{StatusCode, Header, Body}`, no decode |
| Base-URL problem refuses before sending | ✓ Covered | `client.go:NewClient` — `ctx.BaseURLErr != nil` returns it verbatim, nil `*Client`, builds nothing |
| 2xx body cannot be decoded | ✓ Covered | `execute.go:Execute` — decode error on non-nil `out` → `&DecodeError{StatusCode, cause}` |
| Hung connection fails on the request timeout | ✓ Covered | `client.go` `requestTimeout` on `*http.Client`; `execute.go` maps the resulting `Do` error → `*TransportError`, one attempt |
| 429 carries its rate-limit headers | ✓ Covered | `execute.go:Execute` — 429 is generic `*ResponseError` carrying status + headers + body; no retry |
| No usable token — fail-safe propagated | ✓ Covered | `execute.go:Execute` — `errors.As(err, &authErr)` checked **before** wrapping; `*AuthError` returned unchanged |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — Request descriptor + Client + constructors | ✓ Met | `NewClient` returns `ctx.BaseURLErr` verbatim + nil client without inspecting the token (`client.go`; `TestNewClientRefusesOnBaseURLError` supplies a present credential to prove non-inspection); AuthTransport over replay thunk (`TestNewClientBuildsAuthenticatedTransport`); nil base panics, documented (`TestNewClientNilBasePanics`); `requestTimeout` set (`TestNewClientSetsRequestTimeout`); `NewClientFromOS` binds real base + delegates (`TestNewClientFromOSDelegates`); build/vet clean |
| T002 — Execute + typed outcomes | ✓ Met | 2xx-decode / 2xx-no-target / DecodeError / non-2xx ResponseError (incl. 429) generic / TransportError / AuthError-via-`errors.As`-first / exactly-one-`Do` tripwire / body-closed-every-branch / timeout / token-never-rendered — all in `execute.go` + 12 tests in `execute_test.go` |
| T003 — Executable acceptance via godog | ✓ Met | New `TestRequestExecutionFeatures` suite scoped to its own feature file; 10 behavioral scenarios pass; `@wip` removed from those, 4 `@validation` kept `@wip`; all four `apiclient` suites report independent counts (007→8, 008→10, 009→8, 010→10); fakes/`httptest`-only, no real network |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Implementation |
|---|---|---|
| `NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error)` | ✓ Conformant | `client.go` — exact signature; BaseURLErr-verbatim branch; replay-thunk wiring; base URL captured on `*Client`; nil-base panic |
| `NewClientFromOS(ctx ConnectionContext) (*Client, error)` | ✓ Conformant | `client.go` — binds a cloned real `*http.Transport`, delegates |
| `(*Client).Execute(reqCtx context.Context, req Request, out any) (*Response, error)` | ✓ Conformant | `execute.go` — exact signature; URL join + query; one `Do`; outcome mapping |
| `Request{ Method, Path, Query, Body }` | ✓ Conformant | `client.go` — `Method string` (req), `Path string` (req), `Query url.Values`, `Body io.Reader` (interface permitted either listed type) |
| `Response{ StatusCode int, Header http.Header }` | ✓ Conformant | `execute.go` |
| `*TransportError{ cause }` (Unwrap→cause) | ✓ Conformant | `execute.go` — `Error()` + `Unwrap()` |
| `*ResponseError{ StatusCode, Header, Body }` | ✓ Conformant | `execute.go` — generic, uncategorized; no `Kind` field |
| `*DecodeError{ StatusCode, cause }` (Unwrap→cause) | ✓ Conformant | `execute.go` — `Error()` + `Unwrap()` |
| `*AuthError` (007) propagated unchanged | ✓ Conformant | `execute.go` — `errors.As` before wrapping |
| base-URL error from `NewClient`, verbatim | ✓ Conformant | `client.go` — returns the carried error untransformed |

---

## Non-Behavior Absence

**Status**: Pass (all 9 exclusions confirmed absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not resolve/read/choose base URL or token, nor re-assemble context | ✓ Absent | `execute.go` reads only the captured `c.baseURL`; `client.go` reads `ctx.BaseURL.Value`/`ctx.Cred` via the replay thunk — no `os`/`getenv`/`rcfile`/`auth.Resolve` in either file |
| Must not attach `X-Auth-Token` itself or decide the fail-safe | ✓ Absent | `NewClient` wires 007's `AuthTransport`; `Execute` never sets the header; `errors.As` propagates `*AuthError` rather than re-deciding |
| Must not interpret/classify a non-2xx response | ✓ Absent | `*ResponseError` is generic — no `Kind`, no body interpretation |
| Must not follow pagination | ✓ Absent | One `Do`, one `*Response` returned; no page-walking |
| Must not retry, back off, sleep, or treat 429 specially | ✓ Absent | Single `c.httpClient.Do`; no `for`/retry/sleep/backoff; 429 handled by the same generic non-2xx branch |
| Must not decide exit code or user-facing message | ✓ Absent | No `os.Exit`, no printing; all outcomes code-free |
| Must not decode a non-2xx body into the target, nor force decode without a target | ✓ Absent | Non-2xx returns before any decode; `out == nil` drains without decoding |
| Must not print/log/expose the token | ✓ Absent | `ctx.Cred.Token` never read (only a doc-comment mention); errors carry net cause / response side / parse cause — `TestExecuteErrorsNeverRenderTheToken` pins it |
| Must not prompt interactively | ✓ Absent | No prompting anywhere on the path |

---

## @wip Lifecycle Completion

**Status**: Pass

The 10 behavioral scenarios referenced by checked task T003 have had `@wip` removed and pass in `TestRequestExecutionFeatures`. The 4 `@validation @wip` scenarios remain tagged — correctly held out for this validate pass, not referenced as behavioral work by any checked task.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation independently)

These scenarios were held out from the Builder. Each was traced to a code path by inspection; supporting unit tests (written for the behavioral pass) exercise the same paths and corroborate the trace.

| Scenario | Status | Trace |
|---|---|---|
| Request Execution re-resolves nothing | ✓ Satisfied | `client.go`/`execute.go` read no flag/env/credentials file — grep confirms no `os.`/`getenv`/`rcfile`/`auth.Resolve`; base URL comes only from the captured `c.baseURL`, identity only from the AuthTransport replay thunk |
| Exactly one send attempt per request | ✓ Satisfied | Single `c.httpClient.Do(httpReq)` in `execute.go`; no loop/retry/backoff. Corroborated by the call-counting tripwire (`TestExecuteMakesExactlyOneAttempt`, and the no-retry assertion on the timeout and 429 tests) |
| The token value never appears in produced output | ✓ Satisfied | 010 never reads `ctx.Cred.Token`; `TransportError` wraps a net cause, `ResponseError` carries the response side, `DecodeError` names a parse cause. Corroborated by `TestExecuteErrorsNeverRenderTheToken` across all three error types |
| A non-2xx body is never decoded into the success target | ✓ Satisfied | The non-2xx branch reads the body into `ResponseError.Body` and returns before reaching any decode; `out` is never touched. Corroborated by `TestExecuteNon2xxIsGenericResponseError` asserting the target stays empty |

> Note: the 4 `@validation` scenarios carry no godog step definitions (held out by design), so they are verified by inspection, not execution — consistent with validate's inspection-based baseline.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 4 held-out validation scenarios are satisfied through independent inspection. All 3 tasks are checked. The implementation conforms to the specification: it sends one authenticated request through 007's transport, applies a bounded single-attempt timeout, refuses on its own base-URL concern while delegating the token fail-safe to 007, and surfaces typed, code-free `Response`/`TransportError`/`ResponseError`/`DecodeError` outcomes — never classifying the non-2xx, never retrying, never touching the token, and never deciding an exit code. The interface surface matches the accord exactly.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 010 is closed.

The one open item is not a conformance gap but a forecast the spec itself records: the typed errors are code-free by design (ADR-3) — the `Outcome` categories + `ExitCode` cases (network-unavailable, etc.) land with the first consuming command (011) at `internal/cli`'s single registry. Carry that forward into 011's scope, not as a 010 finding.
