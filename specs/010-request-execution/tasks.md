# Tasks: Request Execution

**Feature**: 010-request-execution
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/no-shared-api-client/request-execution.feature

---

## Dependency Graph

Phase 1: Request descriptor + `Client` + `NewClient`/`NewClientFromOS` in `internal/apiclient` (1 task, no phase dependencies) [Shared]
Phase 2: `(*Client).Execute` — send + outcome mapping + typed errors (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance via godog (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: the request seam is infrastructure serving all three user scenarios (one-seam-returns-parsed-response-or-typed-error / failures-surfaced-as-distinct-typed-outcomes / status-and-headers-exposed-for-siblings) rather than any single one.
>
> **Cross-spec note**: this slice is purely additive — it changes no existing file. It consumes landed code: 007's `NewAuthTransport`/`AuthError`/`AuthHeaderName`, 009's `ConnectionContext` (`BaseURL`, `Cred`, `CredErr`, `BaseURLErr`), and 008's `BaseURL.Value`, all in `internal/apiclient`/`internal/auth` on main. It realizes the `http.Client` 008 forecast and 009 deferred, and is the consuming side of 009 ADR-2 (replay the context's cached credential into 007's existing `AuthTransport`) — so **007's code is untouched** (the replay thunk satisfies its injected-resolver contract). It registers **no cobra command** and decides **no exit code**: the `Outcome` categories + `ExitCode` cases (network-unavailable, etc.) and the `--base-url` flag *registration* belong to the first consuming command (011), at `internal/cli`'s single registry; `apiclient` never imports `internal/cli`. No new `.glassfrogrc` key, env var, or flag is introduced.

---

## Branching Guidance

**Pipeline mode**: `spec/010-request-execution/base` → `spec/010-request-execution/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: none active — specs 001–009 are Complete; 010 is the only in-progress spec. The read commands (011–014) and the Should-tier client capabilities (015–017) are later specs that depend on this seam, not concurrent ones.

---

## Phase 1: Request descriptor + `Client` + `NewClient`/`NewClientFromOS` [Shared]

- [ ] **T001** [Shared] Add the request descriptor and the `Client` built once from the `ConnectionContext`, with the base-URL fail-fast and the `AuthTransport` replay-thunk wiring — RED-first unit tests
  - **Scope**: In `internal/apiclient` (a new file, e.g. `request.go`/`client.go`), define the code-free request descriptor `Request{ Method string; Path string; Query map[string]string (or url.Values); Body io.Reader (or []byte) }` and a `Client` type. Add `NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error)`: when `ctx.BaseURLErr != nil`, return that error verbatim and a nil `*Client` (the base-URL fail-fast — 010's own concern, build nothing, do **not** inspect the token); otherwise build `&http.Client{Timeout: requestTimeout, Transport: NewAuthTransport(base, func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr })}` — wrapping the **injected** base transport in 007's `AuthTransport` over the **replay thunk** (009 ADR-2). `base` is required and must be non-nil; a nil base panics (fail-fast, no nil-default — DECISIONS/PR #20; document the precondition). Add `NewClientFromOS(ctx ConnectionContext) (*Client, error)` binding the real base `http.RoundTripper` (a configured `*http.Transport`) and delegating, documented as **once per invocation**. Define `requestTimeout` as a named `[ASSUMED]` constant. 010 never reads `ctx.Cred.Token` itself.
  - **Acceptance criteria**:
    - `NewClient` returns `ctx.BaseURLErr` verbatim (and a nil `*Client`) when the base-URL part is errored; it builds nothing and never inspects the token
    - On a usable base URL, `NewClient` builds a client whose transport is an `AuthTransport` over the replay thunk; with a present credential the outgoing request carries `X-Auth-Token` attached by the transport (observed via a fake base `http.RoundTripper`), not by the client
    - A nil `base` panics (fail-fast); the precondition is documented on the constructor
    - `NewClientFromOS` binds the real base transport and delegates to `NewClient`
    - `requestTimeout` is set on the built `*http.Client`
    - RED-first unit tests: base-URL-error refusal; successful build + header-attached-by-transport on the fake base; nil-base panic; timeout set; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (builds on 007's `NewAuthTransport`/`AuthHeaderName` and 009's `ConnectionContext`, on main)
  - **Plan reference**: Phase 1; ADR-1 (`Client` built once is the seam), ADR-2 (base-URL fail-fast in `NewClient`; token fail-safe stays in 007); Cross-cutting (injected base, fail-fast on nil; secret hygiene — never reads the token)
  - **Interface references**: interface-spec.md — Entry points (`NewClient`/`NewClientFromOS`), Input contract `Request` (Surface), Interactions (build-once / replay seam)
  - **Scenario references**: request-execution.feature: "The request is authenticated by the transport", "A base-URL problem is refused before sending"
  - **Risk**: ⚠️ Resist the nil-default reflex on the injected `base` (keep fail-fast — don't default to `http.DefaultTransport`, which has no timeouts and would make 010 own transport — PR #20). The base-URL fail-fast must **not** also check the token (that would duplicate 007's fail-safe — fork A/ADR-2).

## Phase 2: `(*Client).Execute` — send + outcome mapping + typed errors [Shared]

- [ ] **T002** [Shared] Implement `Execute`: join URL, build the request, make exactly one `Do`, and map the result to a `Response` or a typed `TransportError` / `ResponseError` / `DecodeError` (propagating 007's `AuthError`) — RED-first unit tests
  - **Scope**: In `internal/apiclient`, add `(*Client).Execute(reqCtx context.Context, req Request, out any) (*Response, error)` and the typed code-free outcome types: `Response{ StatusCode int; Header http.Header }`, `TransportError{ cause error }` (with `Error()`/`Unwrap`), `ResponseError{ StatusCode int; Header http.Header; Body []byte }`, and `DecodeError{ StatusCode int; cause error }` (with `Unwrap`). `Execute` joins the `Client`'s captured base URL (from the `ConnectionContext` at `NewClient`) with `req.Path` (008 pass-through-as-given) and appends `req.Query`, builds the `*http.Request` (method, body, `reqCtx`), makes **exactly one** `client.Do` (no retry), and maps: on error, `errors.As(err, &authErr)` **first** → return it unchanged, else return `TransportError{cause}`; on a response, `defer resp.Body.Close()` immediately, then — **2xx**: return `&Response{StatusCode, Header}` and, when `out != nil`, decode the body into `out` (a decode failure → `DecodeError`); when `out == nil`, drain without decoding — **non-2xx**: read the body and return `&ResponseError{StatusCode, Header, Body}` (generic, **not** classified, body **not** decoded into `out`). No exit-code mapping, no printing.
  - **Acceptance criteria**:
    - 2xx with a non-nil `out` decodes the body into `out` and returns `Response{StatusCode, Header}`; 2xx with `out == nil` returns `Response` without decoding (body drained + closed)
    - 2xx with a non-nil `out` whose body does not decode returns a `DecodeError`, not a success
    - A non-2xx (any status, including `429`) returns a `ResponseError` carrying status + headers + raw body; the body is never decoded into `out`; the error is not classified by kind
    - A wire failure returns a `TransportError`; a no-/broken-credential outcome from the transport is returned as the propagated `*AuthError` (discriminated via `errors.As` **before** wrapping — never mislabeled as `TransportError`)
    - `Execute` makes **exactly one** outbound attempt and never retries — pinned by a call-counting tripwire base `RoundTripper`; the response body is closed on every branch
    - A hung connection fails on the request timeout as a `TransportError` (a base/server that blocks past a short injected deadline)
    - No 010 error renders the token; 010 reads only the response side; RED-first unit tests over a fake/tripwire base `RoundTripper` (or `httptest.Server`) for every branch; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-3 (typed code-free `TransportError`/`ResponseError`/`DecodeError`, generic non-2xx, consumer maps), ADR-4 (one bounded attempt + timeout, no retry; `AuthError`-vs-transport discrimination); Cross-cutting (error handling, body always closed, secret hygiene)
  - **Interface references**: interface-spec.md — `(*Client).Execute` (Entry points), Output contract `Response`, Error outcomes (Surface), Interactions (URL join, decode-or-skip, single attempt, body lifecycle), Error Communication
  - **Scenario references**: request-execution.feature: "A successful response is decoded into the caller's target", "A bodyless success returns status and headers without decoding", "A wire failure is surfaced as a transport error", "A non-2xx response is surfaced as a generic response error", "An undecodable success body is surfaced as a decode error", "A hung connection fails on the request timeout", "A missing token refuses the request without sending", "A 429 carries its rate-limit headers"
  - **Risk**: ⚠️ Mislabeling 007's `AuthError` as a `TransportError` — `errors.As(err, &authErr)` must run before wrapping any `Do` error. ⚠️ Body leak — a single `defer resp.Body.Close()` after a non-nil response, before branching, covering the non-2xx and decode-error paths. ⚠️ Keep the non-2xx error **generic** (status+headers+body) — classification is 015's and the 429 backoff is 017's (one attempt, no retry here).

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the 010 driving scenarios pass as executable acceptance via godog, driving `NewClient`/`Execute` over a fake base `RoundTripper` in a new suite scoped to this spec's own feature file
  - **Scope**: Add godog step definitions for `features/no-shared-api-client/request-execution.feature` (all three Rule blocks), in a **new** `internal/apiclient` godog suite (`TestRequestExecutionFeatures`) whose `Paths` names **only** that feature file — the package now has four suites (`TestFeatures` → 007, `TestBaseURLFeatures` → 008, `TestConnectionContextFeatures` → 009, this → 010), each pointed at its specific file (LEARNINGS: a suite must point at its own file, never the `features/` directory). Drive `NewClient`/`Execute` over a **fake** base `http.RoundTripper` (and an `httptest.Server` where a real loopback response is clearer) for the 2xx-decode / bodyless / header-attached / transport-error / non-2xx / base-URL-refusal / decode-error / timeout / no-token / 429-headers behaviors. Step helpers return errors, never panic (LEARNINGS). **Reuse existing step phrasing** where an assertion already exists — grep the package's `sc.Step(` registrations before adding new bindings. Remove `@wip` from the 10 behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 010 scenario (2xx-decoded / bodyless-no-decode / authenticated-by-transport / wire-failure / non-2xx-generic / base-URL-refused / undecodable-decode-error / timeout / missing-token-propagated / 429-rate-limit-headers) has an executable, passing path
    - `@wip` removed from those scenarios; the four `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `request-execution.feature`; all four `apiclient` suites run and report their own independent `N scenarios (N passed)` counts
    - No real network beyond loopback `httptest` and no real home/filesystem are touched (fake base / `httptest.Server` only); `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: request-execution.feature: all 010 behavioral Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — a fourth feature file in the `apiclient` package must keep every suite pointed at specific files (not the directory), or un-wipping one spec's scenarios breaks another suite; verify all four report their own counts. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings (LEARNINGS); step helpers return errors, never panic.
