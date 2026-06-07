# Interface Accord: Request Execution — Specification

**Feature**: 010-request-execution
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4 — the `internal/apiclient` `Client` built once from the `ConnectionContext` (009), its `NewClient`/`NewClientFromOS` constructors and `(*Client).Execute` send seam, and the typed code-free `Response` / `TransportError` / `ResponseError` / `DecodeError` outcomes consumed by the read commands (011–014), API Error Extraction (015), Pagination (016), and Rate-Limit Handling (017).

---

This accord pins the Go API surface of the API-client request seam: the **`Client`** built once from a `ConnectionContext`, the **`NewClient`** pure constructor and its **`NewClientFromOS`** production seam, the **`Execute`** send method, the request descriptor it consumes, and the **`Response`** plus the three typed error outcomes it returns. There is **no command and no entry point** in this slice — execution is a function call, and the cobra command + `--base-url` flag *registration* that triggers a real API call belong to the first consuming command (011) — so the *invocation* and *instructional* surfaces are **N/A**. The capability introduces **no configuration of its own**: the base URL and token are owned upstream (008 `GLASSFROG_BASE_URL` / `.glassfrogrc base_url`; 005 `GLASSFROG_TOKEN` / `.glassfrogrc token`) and arrive pre-resolved inside the `ConnectionContext`; the only tunable, the **request timeout**, is a fixed `[ASSUMED]` constant whose value and eventual configurability are deferred (Consistency Notes). This accord defines the **client built from the context value** — the mirror of 009's accord, which defined the context value itself.

---

## Surface

### Entry points

| Function | Signature (shape) | Description |
|---|---|---|
| `NewClient` | `(ctx ConnectionContext, base http.RoundTripper) (*Client, error)` | The pure constructor. Returns `ctx.BaseURLErr` **verbatim** (and a nil `*Client`) when the base-URL part is errored — the base-URL fail-fast (010's own concern); it builds nothing on that branch and never inspects the token. Otherwise it builds the configured client: `&http.Client{Timeout: requestTimeout, Transport: NewAuthTransport(base, func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr })}` — wrapping the **injected** base transport in 007's `AuthTransport` over the **replay thunk** (009 ADR-2), never a fresh Discovery walk. `base` is required and must be non-nil; a nil base is a wiring bug and panics (fail-fast, no nil-default — DECISIONS/PR #20). |
| `NewClientFromOS` | `(ctx ConnectionContext) (*Client, error)` | The thin production seam: binds `base` to the real base `http.RoundTripper` (a configured `*http.Transport`) and delegates to `NewClient`. Intended to be called **once per invocation**, paired with the single `AssembleFromOS` call (resolve-once; see Interactions). |
| `(*Client).Execute` | `(reqCtx context.Context, req Request, out any) (*Response, error)` | The single send seam. Joins `ctx.BaseURL.Value` with `req.Path`, builds the `*http.Request` (method, query, body, `reqCtx` for cancellation), makes **exactly one** `Do` call (no retry), and maps the result to a `*Response` (2xx) or one of the typed errors below. `out` is the optional decode target: non-nil → the 2xx body is decoded into it; nil → no decode is attempted (bodyless / caller-doesn't-need). |

### Input contract — `Request`

The code-free request descriptor the caller hands to `Execute`. (Field shape is the contract; concrete Go types are a build detail.)

| Field | Type | Required | Description |
|---|---|---|---|
| `Method` | `string` | yes | HTTP method (`GET`, `POST`, …). |
| `Path` | `string` | yes | Request path joined onto the context's base URL (008 pass-through-as-given join; see Interactions). |
| `Query` | `map[string]string` (or `url.Values`) | no | Optional query parameters appended to the URL. |
| `Body` | `io.Reader` (or `[]byte`) | no | Optional request body. Nil/empty for bodyless requests (`GET`, `DELETE`). |

### Output contract — `Response` (success, 2xx)

| Field | Type | Description |
|---|---|---|
| `StatusCode` | `int` | The 2xx status the API returned. |
| `Header` | `http.Header` | The response headers — exposed so Pagination (016) can read Link/paging headers. |

The decoded body is written into the caller's `out` target, not stored on `Response`. When `out` is nil, the body is drained and closed without decoding.

### Error outcomes (typed, code-free)

All are returned as `error` from `Execute` (or `NewClient`), discriminable via `errors.As`. None carries the token. Each maps to a reserved exit code **at the consuming command** (004), not here.

| Type | Shape | Returned when |
|---|---|---|
| `*AuthError` (007, propagated) | `{ Kind: NoCredentials \| CredentialError }` | The `AuthTransport` refused — no usable token or a credential-file error. Propagated **unchanged** from `Do` (discriminated before wrapping); 010 neither owns nor re-decides it. |
| `*TransportError` | `{ cause error }` (Unwrap → cause) | The request could not reach the API or complete at the wire — connection refused, DNS failure, TLS failure, or the request **timeout** elapsed. `Error()` names the failure; the cause is host/port/network-level, never the token. |
| `*ResponseError` | `{ StatusCode int; Header http.Header; Body []byte }` | The API returned a **non-2xx** response. **Generic and uncategorized** — carries the status, headers, and raw body so 015 can extract API detail and 017 can read a `429`'s rate-limit headers. 010 does not classify it. |
| `*DecodeError` | `{ StatusCode int; cause error }` (Unwrap → cause) | The API returned **2xx** and `out` was non-nil, but the body could not be decoded into `out`. Surfaced loud rather than returning a zero-valued target. |
| base-URL error (from `NewClient`) | 008's `*BaseURLError` / `internal/rcfile` error, verbatim | The `ConnectionContext` carried `BaseURLErr`; `NewClient` returns it and builds no client. |

**Example (shapes, not literal values)**:
```
// ctx = the assembled ConnectionContext (009); reqCtx = the per-request context.Context.
build:           client, err := NewClient(ctx, base)                                → err is ctx.BaseURLErr when the endpoint is unusable (built nothing)
2xx + target:    client.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)      → &Response{StatusCode:200, Header:{…}}, nil   // me populated
2xx no target:   client.Execute(reqCtx, Request{Method:"GET", Path:"/health"}, nil)  → &Response{StatusCode:204, Header:{…}}, nil   // no decode
non-2xx:         client.Execute(reqCtx, Request{Method:"GET", Path:"/me"}, &me)      → nil, &ResponseError{StatusCode:403, Header:{…}, Body:[…]}
transport:       client.Execute(reqCtx, …)                                          → nil, &TransportError{cause: dial tcp …: connection refused}
no token:        client.Execute(reqCtx, …)                                          → nil, &AuthError{Kind: NoCredentials}            // from 007
```

---

## Interactions

**Build-once / send flow**: the command layer calls `AssembleFromOS` once (009) to get the `ConnectionContext`, then `NewClientFromOS(ctx)` once to get the `*Client`, then `(*Client).Execute` per request. The client — its base transport, `AuthTransport`, and timeout — is constructed once; resolution already happened once at assembly, so the same identity and endpoint apply to every call by construction.

**URL join**: `Execute` joins `ctx.BaseURL.Value` (008's resolved root, e.g. `https://glassfrog.com/api/v5`) with `req.Path` and appends `req.Query`. The base URL is used as 008 resolved it (pass-through-as-given); the exact join rule is a build detail consistent with 008's contract.

**Decode-or-skip**: on a 2xx, `Execute` decodes the body into `out` when `out != nil`, and skips decoding (draining and closing the body) when `out == nil`. A decode failure on a non-nil `out` is a `DecodeError`, never a silent zero value.

**Single attempt, bounded**: `Execute` makes **exactly one** `Do` call. The request timeout (a fixed constant) bounds it so a hung connection fails loud as a `TransportError`; there is **no retry** — Rate-Limit Handling (017) layers backoff above this seam.

**Replay seam (resolve-once)**: `NewClient` wires `AuthTransport` with `func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr }` — replaying the context's cached credential, not re-walking Discovery (009 ADR-2). 007's `authorize` then yields the token on the present branch and the typed `AuthError` on the absent/errored branch, so the fail-safe refusal fires at request time, unchanged. 010 never reads `ctx.Cred.Token` itself.

**Body lifecycle**: the response body is always closed on every branch (2xx-decoded, 2xx-skipped, non-2xx, decode-error) — no fd/connection leak.

---

## Error Communication

`NewClient` returns an error only for the base-URL fail-fast (it never inspects the token). `Execute` returns exactly one outcome per call, fail-loud at every fork:

| Condition | Outcome |
|---|---|
| `ctx.BaseURLErr` set (at `NewClient`) | Returns that error verbatim; `*Client` is nil; nothing is built or sent. |
| 2xx, `out` non-nil, body decodes | `*Response{StatusCode, Header}`, nil error; `out` populated. |
| 2xx, `out` nil | `*Response{StatusCode, Header}`, nil error; body drained, not decoded. |
| 2xx, `out` non-nil, body does **not** decode | `nil, *DecodeError{StatusCode, cause}`. |
| Non-2xx (any status, incl. `429`) | `nil, *ResponseError{StatusCode, Header, Body}` — generic, carrying the status, headers, and raw body. Not classified, not retried. |
| Wire failure / timeout | `nil, *TransportError{cause}`. |
| No usable token / credential-file error | `nil, *AuthError{…}` — propagated unchanged from 007 (discriminated via `errors.As` before any wrapping, so it is never mislabeled as a `TransportError`). |

**No secret anywhere**: 010 never holds the token (the replay thunk is its only path). `TransportError` carries a network-level cause; `ResponseError` carries the **response** status/headers/body (the API's reply — the `X-Auth-Token` *request* header is not echoed); `DecodeError` names a parse failure. Pinned by a token-never-in-output test across all three error types.

**Code-free, consumer-maps**: none of these outcomes carries an exit code. The first consuming command (011) does `errors.As` and adds the `Outcome` categories + `ExitCode` cases at `internal/cli`'s single registry — transport → code 6 (network-unavailable); non-2xx → codes 3/4/5 once 015/017 classify (004 reserved them). The same open gap 007/008/009 each flagged, now assigned to the first read command.

---

## Consistency Notes

- **Mirrors the established seam patterns**: `NewClient(ctx, base)` + `NewClientFromOS(ctx)` follows the pure-constructor-plus-OS-seam shape of 009's `Assemble` / `AssembleFromOS` and 007's injected-base `NewAuthTransport`; `base` is injected (prod binds the real transport, tests a fake), fail-fast on nil (DECISIONS/PR #20). The typed errors follow 007's `AuthError` precedent — code-free, `errors.As`-able, `Unwrap`-able, token-free.
- **Consumes 009's `ConnectionContext`, builds 008's forecast client**: this accord defines the *client* built from the *context value* 009 defined — realizing the `http.Client` 008 forecast and 009 deferred. It wires 007's `AuthTransport` unchanged (the replay thunk satisfies its existing injected-resolver contract).
- **Generic non-2xx carrier, by design**: `ResponseError` is deliberately uncategorized — API Error Extraction (015) refines it into typed API/permission/rate-limit errors, and Rate-Limit Handling (017) reads the `429` rate-limit headers off the same value. Pagination (016) reads `Response.Header` on the 2xx path. Keeping 010 generic is what lets all three build on one seam (spec Non-Behaviors).
- **No command surface, no new configuration**: like 005/006/007/008/009, this slice registers no cobra command and prints nothing; invocation and instructional surfaces are N/A. The request timeout is the only tunable — an `[ASSUMED]` constant, configurability deferred (as 008 deferred the default URL). No new `.glassfrogrc` key, env var, or flag.
- **Fourth specification touchpoint in this project**: like 005/007/008/009, a specification accord, not a CLI one. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
