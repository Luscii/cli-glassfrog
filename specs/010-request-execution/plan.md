# Plan: Request Execution

**Feature**: 010-request-execution
**Role**: Shaper
**Inputs**: spec.md (010-request-execution); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: `internal/apiclient` is Connection Configuration's home for transport — base URL, **timeouts, retries** — 008 ADR-1; `internal/auth`/`internal/cli` are network-/command-free; 007's `AuthTransport` is an `http.RoundTripper` wrapping a base transport, holding an injected `func() (auth.Resolution, error)`, resolve-once via `sync.Once`, typed code-free `AuthError{NoCredentials|CredentialError}`, fail-safe refusal before reaching the base transport; 009 `ConnectionContext{BaseURL,BaseURLErr,Cred,CredErr}` carries the token and is the single resolution point, with 009 ADR-2 pinning that **010 replays it into `AuthTransport`** via `func(){ return ctx.Cred, ctx.CredErr }` — never a fresh walk; 004 publishes frozen codes 0–6 with 3–6 reserved for the API client and "the future API client adds APIError/PermissionError/RateLimited/NetworkUnavailable + their `ExitCode` cases when their producer exists"; the producer-classifies-a-code-free-outcome / consumer-maps split — 002/004/005/007/008/009; inject seams, fail-fast on nil — 005/008/PR #20); `.score/memory/LEARNINGS.md` (relevant: nil-default-vs-fail-loud for injected seams and "don't default `base` to `http.DefaultTransport` — it has no timeouts and would make this layer own transport" — PR #20; a godog suite must point at its own feature file; godog step helpers return errors, never panic; install a tripwire to assert a negative property — e.g. "exactly one attempt"). No SOUL.md. No DEPRECATION.md beyond the 007/009-seam note 009 ADR-2 already settled.

**Readiness**: Must met + Should substantial — behavioral accord with When/Then, three happy-path + three error + four edge scenarios, nine non-behaviors, integration boundaries naming every sibling (007/009/015/016/017/004), user scenarios, assumptions. Strong foundation. The five behavioral forks were resolved during `/score:define` (decode-on-2xx into an optional target; non-2xx short-circuits to a generic error; "fail fast" is base-URL-only with the token fail-safe staying in 007; no retry; non-2xx error carries status+headers+body). **No architectural unknowns required a resolve conversation**: package placement, the transport-wrapping shape, the replay-the-credential seam, the code-free typed-outcome discipline, and the reserved exit codes are all fixed by DECISIONS precedent; the remaining `[ASSUMED]` items (request-descriptor shape, type names, timeout duration, URL-join semantics) are interface/detail-level, not behavioral gaps.

---

## System Architecture

Request Execution is **the heart of the API Client** (problem: *No Shared API Client* — the CLI can assemble a connection context but has no shared way to issue a request, so every endpoint command would reinvent transport plumbing). It is the single seam Identity Read (011), My Roles (012), My Actions (013), My Projects (014), and the Should-tier client capabilities all call through. It turns the parts the prior slices produced — 009's `ConnectionContext` (endpoint + identity, resolved once) and 007's `AuthTransport` (attaches `X-Auth-Token`, owns the no-token fail-safe) — into one configured HTTP client and one send operation, then reports a typed, code-free outcome.

It is the slice that finally **builds the `http.Client`** — the work 008 forecast for "the connection-context half" and 009 explicitly deferred ("client assembly is NOT here — it moves to Request Execution (010)"). It is purely additive in `internal/apiclient` (new files; no existing file is modified) and registers **no cobra command** — like 005/006/007/008/009, it prints nothing and decides no exit code. Its consumers are sibling Go capabilities, so its boundary is a **specification boundary**: the package API (the client seam, the request descriptor, and the typed outcome/error types) is the structural contract 011–017 consume — the exact shape is the interface skill's concern.

The parts (all in `internal/apiclient`):

- **The request descriptor** — a code-free value the caller hands in: HTTP method, a path joined onto the context's base URL, optional query parameters, an optional request body, and an **optional decode target**. (Field shape is interface-level.)
- **`NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error)`** (with the thin `NewClientFromOS(ctx)` production seam that binds the real base transport) — built **once per invocation** from the assembled context. This is where the **base-URL fail-fast** lives: if `ctx.BaseURLErr != nil` it returns that carried error verbatim and builds nothing (010's own concern — no usable endpoint). Otherwise it constructs the configured `*http.Client`: a base transport wrapped in `NewAuthTransport(base, func() (auth.Resolution, error){ return ctx.Cred, ctx.CredErr })` — the **replay thunk** 009 ADR-2 pinned, not a fresh Discovery walk — and a **request timeout**. The token fail-safe is *not* checked here; it stays in 007's `AuthTransport`, firing at send time.
- **`(*Client).Execute(...)`** — the send seam. Joins the base URL with the request path, builds the `*http.Request` (method, query, body, a caller `context.Context` for cancellation), makes **exactly one** `Do` call (no retry), and maps the result:
  - **transport failure** (connection refused, DNS, TLS, timeout) → a typed `TransportError`;
  - **007's `AuthError`** (no/broken credential) → propagated **unchanged** (discriminated from a wire error via `errors.As` *before* wrapping);
  - **2xx** → success carrying the **status code and response headers**, with the body decoded into the caller's target when one was supplied (and skipped when not); an undecodable body with a target supplied → a typed `DecodeError`;
  - **non-2xx** → a **generic, uncategorized** `ResponseError` carrying the **status code, response headers, and raw body** — never decoded into the success target, never classified.

```
internal/apiclient                      (Connection Configuration / API Client; 008 base URL, 009 context, this = the client)

  NewClient(ctx, base) (*Client, error)                    ── built ONCE per invocation (NewClientFromOS binds the real base)
    │  if ctx.BaseURLErr != nil → return that error (base-URL fail-fast; build nothing)   [ADR-2]
    └─ *http.Client{
         Timeout:   requestTimeout,                          ── one bounded attempt, no retry  [ADR-4]
         Transport: NewAuthTransport(                         ── 007, untouched
                      baseRoundTripper,
                      func(){ return ctx.Cred, ctx.CredErr }, ── replay thunk, NOT a fresh walk [009 ADR-2]
                    ),
       }
        │
  (*Client).Execute(reqCtx, descriptor, target?) (Response, error)         ── the single seam  [ADR-1]
    │  build URL = join(client.baseURL, path) + query; build *http.Request(method, body, reqCtx)   ── baseURL captured at NewClient
    │  resp, err := client.Do(req)                            ── AuthTransport attaches X-Auth-Token / refuses
    ├─ err is *AuthError      → return it unchanged           ── 007's fail-safe, propagated     [ADR-4]
    ├─ err (other)            → return TransportError{cause}  ── network-unavailable             [ADR-3]
    ├─ 2xx                    → Response{Status, Header}; decode body→target if target!=nil
    │                            (decode fails → DecodeError)                                    [ADR-3]
    └─ non-2xx                → ResponseError{Status, Header, Body}  (generic; raw)              [ADR-3]

  consumed by ──► 011–014 reads (errors.As → Outcome → ExitCode at internal/cli registry),
                  015 API Error Extraction (refines ResponseError → typed API error),
                  016 Pagination (reads Response.Header Link/paging),
                  017 Rate-Limit Handling (reads ResponseError 429 + rate-limit headers).
```

The client is built once and threaded; resolution already happened once at assembly (009), so identity and endpoint are stable across every call by construction. 010 itself **never touches the token** — it hands the replay thunk to `AuthTransport` and lets 007 attach the header — so the secret takes exactly one path, the one 007 already governs.

---

## Architecture Decisions

### ADR-1: A `Client` built once from the `ConnectionContext` is the request seam; `Execute` sends through it

**Context**: 010 is "the single seam every endpoint command calls through" (FEATURE-MODEL, spec System Overview). 009 produces the `ConnectionContext` resolved **once per invocation** and reused; 008 fixed `internal/apiclient` as the home for the HTTP client (base URL, timeouts, retries); 007's `AuthTransport` wraps a base transport. The lifecycle the spec states — "the same context applies to every API request in that invocation" — wants the `http.Client`/transport constructed once, not per call.

**Options considered**:
1. **A `Client` value built once via `NewClient(ctx, base)`** (with a `NewClientFromOS(ctx)` production seam), holding the configured `*http.Client` (base transport + `AuthTransport` + timeout) and the base URL; endpoint commands call `(*Client).Execute(...)`. Models "assembled once, reused"; one obvious seam object for 011–017 to depend on.
2. **A free `Execute(ctx, …)` function that builds the `http.Client` on every call** — rejected: rebuilds the transport per request and re-runs `AuthTransport`'s `sync.Once` each call (harmless but wasteful), and offers no single seam object; it doesn't model the once-per-invocation lifecycle.
3. **Each endpoint command builds its own `http.Client` from the context** — rejected: that *is* the "every command reinvents transport plumbing" problem this capability exists to eliminate.

**Decision**: Option 1. `NewClient(ctx ConnectionContext, base http.RoundTripper) (*Client, error)` (with the `NewClientFromOS(ctx)` production seam) builds the configured client once; `(*Client).Execute` is the send seam every command calls. The client is constructed at the same once-per-invocation point the context is, and threaded to every request.

**Consequences**: One legible seam for the read capabilities and the Should-tier client capabilities to build on. The transport is configured once. The exact `Client`/`Execute`/descriptor API (constructor signature, how query+body+target are passed, method/type names) is **interface-level** — deferred to `/score:interface`. *Precedent-setting: the API-client seam is a `Client` built once from the context; sibling capabilities depend on it, not on a hand-built `http.Client`.*

### ADR-2: The base-URL fail-fast lives in `NewClient`; the token fail-safe stays in 007's `AuthTransport` (resolves spec fork A)

**Context**: A complete context needs both a usable base URL and a present token (009 `Complete()`). The spec resolved (fork A → option b): 010 refuses-before-sending only on its **own** concern — a base-URL problem — and the **token** fail-safe stays with 007, whose `AuthTransport` already refuses at `RoundTrip` time with a typed `AuthError`. The context carries `BaseURLErr` (008) separately from `Cred`/`CredErr` (005); 009 ADR-2 pinned the replay-into-`AuthTransport` seam.

**Options considered**:
1. **`NewClient` returns `ctx.BaseURLErr` when it is set** (refuse to build a client without a usable endpoint) and does **not** inspect the token — it wires `NewAuthTransport(base, func(){ return ctx.Cred, ctx.CredErr })`, and 007's `authorize` refuses at send time on the absent/errored credential. Each capability owns one half of the precondition; nothing is duplicated.
2. **`NewClient`/`Execute` front-runs the whole `ctx.Complete()` check and also refuses on a missing token** — rejected: duplicates 007's fail-safe decision and the `NoCredentials`/`CredentialError` taxonomy at a second site (spec Non-Behavior; fork A explicitly chose b).
3. **Check neither — let an empty/invalid base URL fail at the wire** — rejected: a base-URL *error* is a malformed-configuration signal that must fail loud at its source as 008's typed error (CONSTITUTION III), not surface as a confusing transport error after a doomed send.

**Decision**: Option 1. `NewClient` returns the carried `BaseURLErr` verbatim (no transformation, no re-resolution) and builds nothing on that branch; on the usable-base-URL branch it wires the replay thunk into `AuthTransport`. The token fail-safe is delegated, firing in 007 at send time.

**Consequences**: This is the consuming realization of 009 ADR-2 — **007's code is untouched** (the replay thunk satisfies its injected-resolver contract; its `sync.Once` caches an already-final value). The two refusals stay cleanly split: no-endpoint is 010's (a build-time refusal), no-token is 007's (a send-time refusal). A caller that gets a `BaseURLErr` back from `NewClient` never reaches `Execute`. *Precedent-setting: base-URL refusal is the client constructor's; the credential fail-safe is never re-decided outside 007.*

### ADR-3: Outcomes are typed and code-free — `TransportError`, a generic non-2xx `ResponseError{status,headers,body}`, and `DecodeError`; classification and exit-code mapping are the consumer's

**Context**: The producer-classifies / consumer-maps split is the project's spine (002/004/005/007/008/009). 004 publishes frozen codes 0–6 with **3–6 reserved for the API client** and states the API client "adds APIError/PermissionError/RateLimited/NetworkUnavailable + their `ExitCode` cases when their producer exists." 010 produces transport failures (→ network-unavailable, code 6) and non-2xx responses (generic — 015 refines into API/permission, 017 reads the 429 + rate-limit headers). 010 has **no command** and never calls `ExitCode`.

**Options considered**:
1. **010 returns typed code-free errors only** — `TransportError` (wraps the net cause), `ResponseError{StatusCode, Header, Body}` (generic non-2xx), `DecodeError` (2xx body undecodable into the supplied target). The first consuming command (011) / 015 / 017 do `errors.As` and add the `Outcome` categories + `ExitCode` cases at the single `internal/cli` registry. 010 wires no exit code and prints nothing.
2. **010 adds `NetworkUnavailable` to the `Outcome` enum and its `ExitCode` case now** — rejected: the enum and registry live in `internal/cli`; 010 is a network/command-free library seam with no exit path to exercise a category, and adding a case 010 never maps would front-load `internal/cli` coupling the first *command* should own (it would also be the first `apiclient`→`cli` dependency — wrong direction). This **refines** 004's forecast: the API *client* produces the typed errors; the consuming *command* adds the `Outcome` cases — mirroring how 009 refined 008's "assemble the http.Client" forecast.
3. **010 interprets the non-2xx into specific API/permission/rate-limit errors itself** — rejected: that is API Error Extraction's (015) job and Rate-Limit Handling's (017) (spec Non-Behaviors); 010 stays generic so both can build on the same raw carrier.

**Decision**: Option 1. Three typed, code-free error types plus a `Response` success value (status + headers; body decoded into the caller's target). The non-2xx `ResponseError` carries **status, headers, and body** (the superset fork B+E fixed) so 015 reads the body/detail and 017 reads the 429 rate-limit headers off the same value. No `Outcome`/`ExitCode` wiring here.

**Consequences**: 010 stays the pure transport seam; the operational exit codes (3–6) land with their first real consumer at 004's single registry site — the same open gap 007/008/009 each flagged, now explicitly assigned to the first read command. `DecodeError` (a 2xx the API contract says should parse but didn't) is surfaced loud rather than swallowed; its eventual code is the consumer's call. *Precedent-setting: the API client produces typed code-free transport/response/decode errors; the consuming command adds the `Outcome` cases and maps them — `apiclient` never imports `internal/cli`.*

### ADR-4: One bounded attempt with a client-level timeout, no retry; discriminate 007's `AuthError` from a transport error after `Do`

**Context**: The spec requires a request timeout (a hung connection must fail loud, not block forever), **exactly one** send attempt, and **no** retry (Rate-Limit Handling 017 owns backoff). 008 fixed timeouts as `internal/apiclient`'s concern; PR #20 LEARNINGS warns `http.DefaultTransport` has no timeouts and that this layer must own the client. Because `AuthTransport` returns its `*AuthError` *before* reaching the base transport, a no-/broken-credential outcome and a genuine wire failure both surface as the `error` from `client.Do` — 010 must not mislabel the fail-safe as a transport error.

**Options considered**:
1. **Configure the timeout on the built client and make one `Do`; on error, `errors.As(err, &authErr)` (a `*AuthError` target) first → return it unchanged, else wrap as `TransportError`.** One attempt, no retry loop; the auth fail-safe is preserved as its own typed outcome.
2. **Apply no timeout** (rely on the OS/server) — rejected: a hung or slow connection blocks indefinitely; the spec wants fail-loud (and `DefaultTransport` has none — PR #20).
3. **Retry transient transport errors a few times** — rejected: retry/backoff is 017's capability (spec Non-Behavior); 010 makes exactly one attempt so 017 can layer backoff above the seam without 010 hiding failures.

**Decision**: Option 1. A request timeout via a named `requestTimeout` constant (an `[ASSUMED]` default — exact value and whether it becomes configurable deferred, as 008 did for the default URL); one `client.Do`; `errors.As` discriminates `*AuthError` (propagate) from any other error (`TransportError`). A tripwire base `RoundTripper` that counts calls pins "exactly one attempt."

**Consequences**: Hung connections fail loud and fast; 017 can add backoff above a seam that never silently retries. The `AuthError`-vs-transport discrimination keeps 007's `NoCredentials`/`CredentialError` taxonomy intact through the seam. The timeout constant is a tunable detail, not a behavioral contract. *Feature-local; not precedent-setting beyond the timeout-lives-in-apiclient point 008 already set.*

---

## Integration Design

- **Connection Context Assembly (009, `internal/apiclient` — upstream)**: `NewClient` consumes the `ConnectionContext`. It reads `BaseURL.Value` as the request root and `BaseURLErr` for the fail-fast (ADR-2); it passes `Cred`/`CredErr` into the replay thunk **without ever reading the token itself**. No re-resolution.
- **Request Authentication (007, `internal/apiclient` — upstream, untouched)**: `NewClient` wires `NewAuthTransport(base, replayThunk)`. 007 attaches `X-Auth-Token` on the authenticated branch and returns its `*AuthError` on the absent/errored branch — `Execute` propagates that error unchanged (ADR-4). 010 neither attaches the header nor re-decides the fail-safe.
- **API Error Extraction (015, `internal/apiclient` — downstream sibling)**: consumes `ResponseError`'s status + body to produce a typed, meaningful API error. 010 produces the raw generic carrier; 015 interprets it. The `errors.As`-able shape is the seam.
- **Pagination (016 — downstream sibling)**: reads `Response.Header` (Link/paging) on a 2xx to walk pages. 010 returns one response per call.
- **Rate-Limit Handling (017 — downstream sibling)**: reads the `429` status and rate-limit headers carried on `ResponseError` to back off. 010 surfaces them; it never sleeps or retries.
- **Exit-Code Convention (004, `internal/cli` — downstream)**: the typed errors map to reserved codes (transport → 6 network-unavailable; non-2xx → 3/4/5 once 015/017 classify). The `Outcome` categories + `ExitCode` cases are added by the **first consuming command (011)** at the single registry site (ADR-3) — not here. `apiclient` does not import `internal/cli`.

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: 010 **never holds the token**. It hands `AuthTransport` the replay thunk `func(){ return ctx.Cred, ctx.CredErr }` and lets 007 attach the header — the only path the secret takes, already governed by 007's secret-safe `AuthError` and the context's redacting `String()`. None of 010's outputs carries the token: `TransportError` wraps a net cause (host/port/DNS, never the header); `DecodeError` names a decode failure; `ResponseError` carries the **response** status/headers/body (the API's reply — the auth header is a *request* header and is not echoed). Pinned by a test asserting no 010 error renders the token, and that 010's own code never reads `ctx.Cred.Token`.

**Error handling (CONSTITUTION III)**: fail loud at every fork — a base-URL error is refused at `NewClient` (no doomed send); a wire failure becomes a typed `TransportError`; a non-2xx is never silently treated as success (short-circuited to `ResponseError`); a 2xx body that won't decode becomes a loud `DecodeError`, not a zero-valued target. The response body is **always closed** on every path (`defer resp.Body.Close()`), including the non-2xx and decode-error branches, to avoid an fd/connection leak. No retry hides a failure (ADR-4).

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network. The base transport is an **injected `http.RoundTripper`** (production binds the real one in `NewClient`; tests bind a fake), so every branch is exercised offline: header-attached-by-007 (assert the fake base sees `X-Auth-Token`), one-attempt (a call-counting **tripwire** base — LEARNINGS), 2xx-decode-into-target, 2xx-no-target-skip, non-2xx → `ResponseError{status,headers,body}`, 2xx-undecodable → `DecodeError`, `AuthError` propagation (a no-token context, asserting the returned error is `*AuthError` not `*TransportError`), transport error (fake base returns a net error), and the timeout (a base that blocks past a short injected deadline, kept fast). The driving scenarios become a **fourth `internal/apiclient` godog suite** (`TestRequestExecutionFeatures`) pointed at its **own** feature file `features/no-shared-api-client/request-execution.feature` (LEARNINGS — never the whole `features/` dir); step helpers return errors, never panic (LEARNINGS); reuse existing `apiclient` step vocabulary where an assertion already exists (grep the package's `sc.Step(` registrations before adding phrasings).

**No command surface (this slice)**: like 005/006/007/008/009, 010 registers no cobra command, prints nothing, and calls no `os.Exit`. The cobra LEARNINGS do not apply. The exit-code mapping and the `--base-url` flag *registration* belong to the first consuming command (011); 010 accepts the assembled context as its input.

---

## Implementation Strategy

Three phases, linear. Depends only on landed code: 007 (`NewAuthTransport`, `AuthError`, `AuthHeaderName`), 009 (`ConnectionContext`, `Cred`, `BaseURLErr`), 008 (`BaseURL.Value`). Purely additive in `internal/apiclient` — no existing file is modified.

- **Phase 1 — request descriptor + `Client` + `NewClient`/`NewClientFromOS`**: define the code-free request descriptor (method, path, query, body, optional target) and the `Client`. `NewClient(ctx, base)` implements the base-URL fail-fast (return `ctx.BaseURLErr` verbatim, build nothing) and, on the usable branch, builds the `*http.Client` with the timeout and `NewAuthTransport(base, replayThunk)` over the injected base transport; `NewClientFromOS(ctx)` binds the real base transport and delegates. Fail-fast on a nil injected base (no nil-default — PR #20/LEARNINGS). RED-first unit tests: refuses on `BaseURLErr`; builds on a usable base URL; the replay thunk drives the `AuthTransport` (header attached for a present credential, observed on the fake base). *Depends on: 007/008/009 (landed).*
- **Phase 2 — `Execute`**: URL join (the `Client`'s captured base URL + path + query), `*http.Request` build with the caller `context.Context` and body, exactly one `client.Do`, and the full outcome mapping — 2xx decode-into-target / no-target skip; non-2xx → `ResponseError{status,headers,body}` (no decode); 2xx-undecodable → `DecodeError`; `errors.As(err, &authErr)` propagation; other error → `TransportError`; body always closed. RED-first unit tests over the fake/tripwire base `RoundTripper` for every branch, plus the one-attempt tripwire and the timeout case. *Depends on: Phase 1.*
- **Phase 3 — executable acceptance**: godog step definitions for the driving scenarios (2xx decoded; 2xx bodyless no-target; identity carried by the transport; transport failure; non-2xx short-circuit; base-URL refusal; undecodable 2xx; timeout; 429 carrying rate-limit headers; no-token fail-safe propagated), in the new `internal/apiclient` suite pointed at this spec's own feature file. *Depends on: Phase 2.*

---

## Risks

- **007's `AuthError` mislabeled as a `TransportError`** (medium likelihood, medium impact): both surface as the `error` from `Do`; if `Execute` wraps indiscriminately, the no-credential fail-safe loses its taxonomy and maps to the wrong exit code. *Mitigation*: `errors.As(err, &authErr)` **before** wrapping (ADR-4); a no-token-context test asserts the returned error is `*AuthError`, not `*TransportError`.
- **Response body not closed → fd/connection leak** (medium likelihood, medium impact): the non-2xx and decode-error branches are the easy ones to forget. *Mitigation*: a single `defer resp.Body.Close()` immediately after a non-nil response, before branching; covered by the httptest/fake-base tests exercising every branch.
- **Token leak through an error or log line** (low likelihood, high impact): 010 sits on the request path where headers are most likely to be traced. *Mitigation*: 010 never reads `ctx.Cred.Token` (replay thunk only); errors carry only the **response** side and net cause; pinned by a token-never-in-output test across `TransportError`/`ResponseError`/`DecodeError`.
- **Default timeout wrong for the real API** (low likelihood, low impact): a too-short timeout fails healthy slow calls; too-long blocks. *Mitigation*: a named `[ASSUMED]` constant, tunable, with configurability deferred (as 008 deferred the default URL); does not block the slice.
- **Exit code for transport/non-2xx undefined until a consumer maps it** (medium likelihood, low impact): 004 reserved codes 3–6 but their `Outcome` categories don't exist yet. *Mitigation*: 010 stays code-free (ADR-3); the first read command (011) adds the categories + `ExitCode` cases at the single registry site. The same open gap 007/008/009 each flagged — does not block 010.

---

## What This Plan Does Not Cover

- **Interpreting the non-2xx `ResponseError`** into typed API / permission / rate-limit errors — API Error Extraction (015), which consumes the raw status+headers+body this slice carries.
- **Following pagination** across pages — Pagination (016), which reads the `Response.Header` Link/paging this slice exposes.
- **`429` backoff/retry** off the rate-limit headers — Rate-Limit Handling (017); this slice surfaces the 429 as a generic `ResponseError` and makes exactly one attempt.
- **The exact package API shape** — the `Client`/`Execute` signatures, the request-descriptor fields, how query+body+target are passed, the error/`Response` type names — `/score:interface` pins these (the specification boundary).
- **The `Outcome` enum categories + `ExitCode` cases** (network-unavailable, etc.) and the command that maps the typed errors — the first consuming command (011) at `internal/cli`'s single registry site; 004 reserved the codes.
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/no-shared-api-client/request-execution.feature`.
- **The cobra command + `--base-url` flag registration that triggers a real API call** — a future command spec (011+); this slice accepts the assembled `ConnectionContext` as its input seam.
