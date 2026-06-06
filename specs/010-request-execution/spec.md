# Specification: Request Execution

**Feature**: 010-request-execution
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Request Execution is the heart of the **API Client** solution (problem: *No Shared API Client* — the CLI can resolve a connection context but has no shared way to issue a request, so every endpoint command would reinvent transport plumbing). It is **the single seam every endpoint command calls through**: hand it a request and it sends an authenticated call to the Glassfrog API, then returns either a successful response or a typed transport error. Identity Read (011), My Roles (012), and every other read sits on top of it.

It is deliberately narrow — the *transport* of one request, nothing more. It consumes the assembled **connection context** (009) for the base URL and sends through the **authenticated transport** (007), which attaches `X-Auth-Token` and owns the no-token fail-safe. It does not interpret *what* a non-2xx response means (API Error Extraction, 015), does not follow paging (Pagination, 016), and does not back off on `429` (Rate-Limit Handling, 017). Those three siblings all depend on this seam and read from what it surfaces. Request Execution's job is to send one authenticated request, fail fast and loudly when it cannot, and hand back a structured outcome the siblings can build on.

---

## Behavioral Accord

### Sending an authenticated request

- When given a request (an HTTP method, a path joined onto the connection context's base URL, optional query parameters, and an optional body), the system sends it to the Glassfrog API through the authenticated transport, so the call carries the resolved identity without Request Execution attaching the header itself.
- When the request is sent, the system applies a request timeout so a hung or unresponsive connection fails loudly rather than blocking indefinitely; it makes exactly one send attempt and never retries.

### Fail fast on its own concern — the base URL

- When the connection context carries a base-URL problem (missing or errored endpoint), the system refuses before constructing or sending the request and surfaces that base-URL problem, naming it; no request is attempted.
- When the context carries a usable base URL but no usable token, the system does not front-run that decision: it sends through the authenticated transport, whose fail-safe refuses the unauthenticated call. The system propagates that authentication outcome unchanged rather than owning or duplicating it.

### Surfacing the outcome

- When the API returns a 2xx response, the system reports success carrying the status code, the response headers, and — only when the caller supplied a decode target — the response body decoded into that target. When no target is supplied, the system does not attempt to decode the body.
- When the API returns a 2xx response and a decode target was supplied but the body cannot be decoded into it, the system surfaces a typed decode error rather than a success.
- When the API returns a non-2xx response, the system short-circuits into a generic, uncategorized non-2xx error carrying the status code, the response headers, and the raw response body — without decoding the body into the success target and without interpreting which kind of failure the status represents.
- When the request cannot reach the API or complete at the wire (connection refused, DNS failure, TLS failure, timeout), the system returns a typed transport error naming the failure, with no response and no decode.
- The system reports its outcome to the calling command; it does not decide the process exit code or the user-facing message for any outcome.

---

## User Scenarios

**In order to** issue a Glassfrog API call without re-wiring base URL, identity, timeouts, and response handling for every command,
**as an** AI agent building an endpoint command,
**I want** one seam I can hand a request to and get back a parsed response or a typed error.

**In order to** distinguish "the network failed" from "the API answered, but not with success,"
**as an** operator diagnosing a failed command,
**I want** transport failures, non-2xx responses, and decode failures surfaced as distinct, typed outcomes.

**In order to** build paging and rate-limit handling on top without reinventing transport,
**as a** maintainer adding the sibling capabilities,
**I want** the status code and response headers exposed on both the success and the non-2xx outcome.

---

## Non-Behaviors

- The system must not resolve, read, or choose the base URL or token, nor re-assemble the connection context. **Why**: Base URL Resolution (008), Credential Discovery (005), and Connection Context Assembly (009) own those; a second path here would drift from their contracts.
- The system must not attach the `X-Auth-Token` header itself or decide the no-token fail-safe. **Why**: Request Authentication (007) owns the authenticated transport and the fail-safe refusal; Request Execution sends *through* it and would split that contract if it duplicated either.
- The system must not interpret or classify a non-2xx response into a specific API error, nor extract the API's error detail. **Why**: API Error Extraction (015) owns turning the raw status and body into a typed, meaningful error; Request Execution only surfaces them.
- The system must not follow pagination or fetch additional pages. **Why**: Pagination (016) owns walking the API's paging; Request Execution returns the single response it received.
- The system must not retry, back off, sleep, or treat `429` specially. **Why**: Rate-Limit Handling (017) owns backoff; a `429` is surfaced as just another non-2xx (carrying its rate-limit headers) so 017 can read and act on it.
- The system must not decide the process exit code or the user-facing message for any outcome. **Why**: Exit-Code Convention (004) and the consuming command own classification; Request Execution reports the outcome and lets that mapping happen downstream.
- The system must not decode a non-2xx response body into the success target, nor force a decode when no target was supplied. **Why**: a decode target is the caller's opt-in for a successful body; forcing it would corrupt error handling and break bodyless responses (e.g. `204`).
- The system must not print, log, or expose the token value. **Why**: the token is a secret, and the request path — request logs, verbose tracing — is exactly where it leaks; the same secret-never-emitted rule that governs Discovery, Assembly, and Request Authentication applies here.
- The system must not prompt interactively for anything. **Why**: the operator is usually a non-interactive AI agent; a missing or broken part is surfaced as a typed outcome, not solicited.

---

## Integration Boundaries

- **Connection Context Assembly (009 — upstream dependency)**: provides the base URL (and its source) for the request root, or a carried base-URL error. Request Execution reads the base-URL portion and refuses before sending when it is errored; it never re-resolves.
- **Request Authentication (007 — upstream dependency)**: provides the authenticated transport that attaches `X-Auth-Token` and fail-safe-refuses when no usable token exists. Request Execution composes its request through this transport and propagates its authentication outcome; it never attaches the header or owns the fail-safe itself. This is the consuming side of the seam 009 recorded.
- **API Error Extraction (015 — downstream sibling)**: consumes the non-2xx error's status, headers, and raw body to produce a typed, meaningful API error. Request Execution produces the raw non-2xx outcome; 015 interprets it.
- **Pagination (016 — downstream sibling)**: reads the response headers (paging / `Link`) Request Execution exposes to walk additional pages. Request Execution returns one response per call.
- **Rate-Limit Handling (017 — downstream sibling)**: reads the `429` status and rate-limit headers carried on the non-2xx error to back off. Request Execution does not back off or retry.
- **Exit-Code Convention (004 — downstream)**: the outcome ultimately informs the command's exit code, but the classification and mapping belong to that capability and the consuming command, not this one.

---

## Driving Scenarios

### Happy path

**Scenario: 2xx response decoded into a supplied target**
Given a connection context with a usable base URL and a present token
And a request with a decode target supplied
When the request is sent and the API returns a 2xx response with a JSON body
Then the system reports success carrying the status code, the response headers, and the body decoded into the target.

**Scenario: 2xx response with no decode target (bodyless)**
Given a connection context with a usable base URL and a present token
And a request with no decode target supplied
When the request is sent and the API returns a 2xx response with no body
Then the system reports success carrying the status code and the response headers
And does not attempt to decode a body.

**Scenario: identity is carried by the authenticated transport**
Given a connection context with a present token
When a request is sent
Then the call goes out through the authenticated transport with the `X-Auth-Token` header attached by that transport
And Request Execution does not attach the header itself.

### Error scenarios

**Scenario: transport failure at the wire**
Given a connection context with a usable base URL
When the request cannot reach the API (connection refused, DNS failure, or TLS failure)
Then the system returns a typed transport error naming the failure
And returns no response and attempts no decode
And does not retry.

**Scenario: non-2xx response is short-circuited**
Given a connection context with a usable base URL and a present token
When the request is sent and the API returns a non-2xx response
Then the system returns a generic, uncategorized non-2xx error carrying the status code, the response headers, and the raw body
And does not decode the body into the success target
And does not classify which kind of failure the status represents.

**Scenario: base-URL problem refuses before sending**
Given a connection context carrying a base-URL error
When a request is to be sent
Then the system refuses before constructing or sending the request
And surfaces the base-URL problem, naming it
And no request reaches the API.

### Edge cases

**Scenario: 2xx body cannot be decoded into the supplied target**
Given a request with a decode target supplied
When the API returns a 2xx response whose body is not valid for that target
Then the system surfaces a typed decode error rather than a success.

**Scenario: hung connection fails on the request timeout**
Given a connection that accepts but never responds
When a request is sent
Then the request timeout elapses and the system returns a typed transport (timeout) error
And does not retry.

**Scenario: 429 is surfaced as a non-2xx carrying its rate-limit headers**
Given the API returns a `429` rate-limit response
When the request is sent
Then the system returns the non-2xx error carrying the `429` status, the rate-limit headers, and the body
And does not sleep, back off, or retry.

**Scenario: no usable token — the transport's fail-safe is propagated**
Given a connection context with a usable base URL but no usable token
When a request is to be sent
Then the authenticated transport's fail-safe refuses the unauthenticated call
And the system propagates that authentication outcome without sending an unauthenticated request and without owning the decision.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Request Execution re-resolves nothing**
Given the connection context and the authenticated transport are the only inputs
When a request is sent
Then the only base URL used is the context's and the only identity used is the transport's
And the capability reads no flag, environment variable, or credentials file directly.

**Scenario: exactly one send attempt per request**
Given any response or failure
When a request is sent
Then the system makes exactly one outbound attempt
And does not retry, back off, or follow paging.

**Scenario: the token value never appears in produced output**
Given any outcome (success, transport error, non-2xx error, or decode error)
When the outcome and any diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: a non-2xx body is never decoded into the success target**
Given a decode target was supplied
When the API returns a non-2xx response
Then the body is carried raw on the non-2xx error
And is never decoded into the success target.

---

## Assumptions

- **Request shape** `[ASSUMED]`: the caller supplies an HTTP method, a path joined onto the context's base URL, optional query parameters, an optional body, and an optional decode target. (The exact parameter shape is a planning detail; the behavior — what is sent and what is surfaced — is fixed.)
- **Authenticated-transport seam**: Request Execution sends through 007's authenticated transport (composing it into the HTTP client); header attachment and the no-token fail-safe stay with 007. This is the consuming side of the seam 009 recorded between Assembly, Request Authentication, and Request Execution.
- **Request timeout** `[ASSUMED]`: a default request timeout exists so a hung connection fails loudly. The exact duration and whether it is configurable are planning details; the behavior (one bounded attempt, no retry) is fixed.
- **Outcome and error type names** `[ASSUMED]`: the names of the success result and the transport / non-2xx / decode error types are adjustable at planning without changing behavior.
- **Base-URL join semantics** `[ASSUMED]`: how a request path is joined onto the resolved base URL is a planning detail consistent with 008's pass-through-as-given contract.

---

## Ambiguity Warnings

_None remaining — the five behavioral forks were resolved during the defining conversation: (1) a supplied decode target is decoded on a 2xx body, with a typed decode error on failure; (2) a non-2xx response is short-circuited into a generic error, leaving specific classification to API Error Extraction (015); (3) "fail fast" is owned by Request Execution only for the base URL — the token fail-safe stays with Request Authentication (007), whose transport refuses at send time; (4) no retries (Rate-Limit Handling, 017, owns backoff); and (5) the non-2xx error carries status, headers, and body so both 015 and 017 can build on it. The remaining `[ASSUMED]` items are planning-time shape details, not behavioral gaps._
