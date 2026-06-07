# Specification: API Error Extraction

**Feature**: 015-api-error-extraction
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

API Error Extraction is the capability that answers *what went wrong* when the Glassfrog API answers but not with success. It pairs with the Next-tier **Opaque Failures** problem — when a call fails, the caller can't tell what went wrong or what to do next. Request Execution (010) deliberately stops at a **generic, uncategorized non-2xx error** carrying the status code, the response headers, and the raw response body; it never interprets which kind of failure the status represents. API Error Extraction takes exactly that outcome and turns it into a single **typed API error** carrying the API's status and human-meaningful detail, so an AI agent or operator receives a structured cause instead of a raw response.

It is deliberately narrow — the *interpretation* of one non-2xx outcome, nothing more. The Glassfrog API serves every 4xx and 5xx as an **RFC 9457 Problem Details** document (`type`, `title`, `status`, `detail`, served as `application/problem+json`, with optional extension members). This capability reads that documented shape, extracts the standard members, and degrades gracefully when the body isn't a readable Problem Details document — so a failed call is never opaque. It does not send requests (Request Execution, 010, owns transport), does not back off on `429` (Rate-Limit Handling, 017, owns that), does not translate a `403` into plan-availability guidance (the Unsignalled Plan Limits problem owns that), and does not decide the process exit code or user-facing message (Exit-Code Convention, 004, and the consuming command own that). Its job is to make a non-2xx legible.

---

## Behavioral Accord

### Consuming a non-2xx outcome

- When given the generic non-2xx outcome produced by Request Execution (a status code, the response headers, and the raw response body), the system interprets it into a single typed API error carrying the API's status and error detail; it reads only what was handed to it and sends nothing.
- When the outcome handed in is a success, a transport error, or a decode error rather than a non-2xx response, the system does not act on it — those are outside its concern and are not its to interpret.

### Extracting the error detail

- When the non-2xx body is a valid RFC 9457 Problem Details document, the system extracts the `detail`, `title`, and `type` members and carries them on the typed error alongside the HTTP status code, so the caller can read a human-meaningful cause.
- When the body carries additional members beyond the standard four (RFC 9457 "extensions"), the system preserves the raw body so extensions remain available, while surfacing only the standard members as named fields.

### Degrading when the body can't be read

- When the body is absent, empty, not JSON, or JSON that lacks the required Problem Details members, the system still produces the typed error from the HTTP status, supplying a fallback detail derived from the status and preserving the raw body for diagnostics. It never fails to produce a typed error because the body could not be parsed.
- The system does not condition extraction on the response `Content-Type`: it attempts to read the documented shape regardless of the declared type and degrades when the shape isn't there.

### Status authority

- When the body carries a `status` member that disagrees with the HTTP response status, the HTTP response status is authoritative on the typed error; the body's `status` is carried as extracted metadata and never overrides what actually came back at the wire.

### Surfacing the typed error

- The system produces one structured API error carrying the authoritative HTTP status code, the extracted `detail`/`title`/`type` when available, the raw body, and the response headers it received, so callers can branch on the status code and read the cause.
- The system reports its typed error to the calling command; it does not decide the process exit code, print, or compose the user-facing message for the failure.

---

## User Scenarios

**In order to** know *why* a Glassfrog call failed instead of staring at a raw status and body,
**as an** AI agent driving a command,
**I want** a non-2xx response turned into a typed error carrying the API's status and human-readable detail.

**In order to** decide what to do next after a failure (re-fetch, fix input, give up),
**as an** operator diagnosing a failed command,
**I want** the API's own `detail` and `title` surfaced as named fields, with the raw body still available when I need more.

**In order to** still get a usable error when a gateway returns junk instead of a Problem Details body,
**as an** AI agent,
**I want** the system to fall back to the HTTP status rather than failing to produce any error at all.

---

## Non-Behaviors

- The system must not send requests, open connections, or read transport. **Why**: Request Execution (010) owns the single send and surfaces the non-2xx outcome; a second transport path here would drift from that seam.
- The system must not retry, back off, or sleep on a `429`. **Why**: Rate-Limit Handling (017) owns the backoff/handling; 015 extracts the `429` into a typed error (carrying its rate-limit headers) so 017 can read and act on it, and the consumer *classifies* the `429` to `rate-limit`(5) by status — that is classification, not handling, mirroring how 401/403 classify to `permission`(4). (Landed 017's spec defers the `429`→`rate-limit`(5) classification to 015 as "the producer that types non-2xx responses.")
- The system must not translate a `403` into "not available on your plan" or other plan-availability guidance. **Why**: the Unsignalled Plan Limits problem owns plan/flag-gated interpretation; 015 carries the `403` status and detail generically so that capability can build on it without overlap.
- The system must not condition extraction on the response `Content-Type` header. **Why**: proxies and gateways return problem bodies (or none) with varying or absent content types; gating on the header would discard readable detail the API did send.
- The system must not interpret success, transport-error, or decode-error outcomes. **Why**: those are not failures of the API answering; treating them here would duplicate Request Execution's (010) typed outcomes and blur the boundary.
- The system must not decide the process exit code, print, or compose the user-facing message. **Why**: Exit-Code Convention (004) and the consuming command own classification and presentation; 015 reports the typed error and lets that mapping happen downstream.
- The system must not re-decode the error body into a command's success target. **Why**: the non-2xx body is an error document, not the resource the caller asked for; decoding it as success would corrupt the caller's result.
- The system must not let the body's `status` member override the HTTP response status. **Why**: the transport status is what actually happened; trusting a self-described status that disagrees would mislead callers branching on the code.

---

## Integration Boundaries

- **Request Execution (010 — upstream dependency)**: produces the generic, uncategorized non-2xx outcome (status code, response headers, raw body). API Error Extraction consumes exactly that and interprets it; this is the consuming side of the seam 010 recorded for 015. It never re-sends or re-reads transport.
- **Rate-Limit Handling (017 — landed sibling)**: reads the `429` status and rate-limit headers to retry/back off. API Error Extraction extracts the `429` and the consumer classifies it to `rate-limit`(5) — the classification 017's spec explicitly defers to 015 — while 015 itself never backs off. Composition: 010 sends → 017 retries/backs off → 015 classifies the final outcome.
- **Unsignalled Plan Limits (Next-tier problem / future capability)**: owns turning a `403`/feature-flag rejection into a clear plan-availability signal. API Error Extraction carries the `403` status and detail generically; it does not pre-empt that interpretation.
- **Exit-Code Convention (004 — downstream)**: the typed API error informs the command's exit code, but the classification and mapping belong to that capability and the consuming command, not this one.
- **Glassfrog API (system actor)**: the source of the RFC 9457 Problem Details body and the HTTP status. Per the project's "spec is the contract" constraint, this capability reads the documented Problem Details shape faithfully and treats divergence (an unreadable body) as something to degrade around, not to reject.

---

## Driving Scenarios

### Happy path

**Scenario: a valid Problem Details body is extracted**
Given a non-2xx outcome with a body that is a valid RFC 9457 Problem Details document
When the outcome is interpreted
Then the system produces a typed API error carrying the HTTP status code and the extracted `detail`, `title`, and `type`.

**Scenario: a 404 surfaces the API's own detail**
Given a non-2xx outcome with status `404` and a Problem Details body whose `detail` is "Not Found"
When the outcome is interpreted
Then the typed error carries status `404` and detail "Not Found"
And the calling command can read that cause without parsing the raw body itself.

**Scenario: extension members are preserved without being promoted**
Given a non-2xx outcome whose Problem Details body carries extension members beyond the standard four
When the outcome is interpreted
Then the typed error surfaces only `detail`, `title`, and `type` as named fields
And preserves the raw body so the extension members remain available.

### Error scenarios

**Scenario: an empty body degrades to the HTTP status**
Given a non-2xx outcome with status `500` and no body
When the outcome is interpreted
Then the system still produces a typed error carrying status `500`
And supplies a fallback detail derived from the status
And preserves the (empty) raw body for diagnostics.

**Scenario: a non-JSON gateway body degrades gracefully**
Given a non-2xx outcome with status `502` whose body is HTML returned by a gateway
When the outcome is interpreted
Then the system produces a typed error carrying status `502` with a fallback detail
And does not fail to produce an error because the body was not Problem Details
And does not condition this on the response `Content-Type`.

### Edge cases

**Scenario: body status disagrees with the HTTP status**
Given a non-2xx outcome with HTTP status `403` whose Problem Details body carries `status: 401`
When the outcome is interpreted
Then the typed error's authoritative status is `403`
And the body's `status` (`401`) is carried as extracted metadata only and does not override.

**Scenario: a 429 is extracted but not backed off**
Given a non-2xx outcome with status `429` and its rate-limit headers
When the outcome is interpreted
Then the system produces a typed error carrying status `429`, the extracted detail, and the response headers
And does not sleep, back off, or retry.

**Scenario: a 403 is carried generically**
Given a non-2xx outcome with status `403` from a plan- or flag-gated endpoint
When the outcome is interpreted
Then the typed error carries status `403` and the API's detail as-is
And the system does not translate it into plan-availability guidance.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: extraction reads only the outcome handed in**
Given the non-2xx outcome is the only input
When it is interpreted
Then the system sends no request and reads no flag, environment variable, or credentials file
And the only status, headers, and body used are those carried on the outcome.

**Scenario: every non-2xx yields a typed error**
Given any non-2xx outcome, including one whose body cannot be parsed
When it is interpreted
Then the system always produces a typed API error
And no path returns success or a nil error for a non-2xx outcome.

**Scenario: the body status never overrides the produced status**
Given a non-2xx outcome whose body `status` disagrees with the HTTP status
When the produced typed error is inspected
Then its authoritative status equals the HTTP response status
And the body's `status` appears only as carried metadata.

**Scenario: no backoff is observable for a 429**
Given a `429` non-2xx outcome
When it is interpreted
Then no sleep, delay, or retry occurs during interpretation
And the rate-limit headers are carried through unchanged.

---

## Assumptions

- **Input shape** `[ASSUMED]`: the system consumes Request Execution's (010) non-2xx outcome — a status code, response headers, and raw body. The exact type is 010's; its name and accessors are a planning detail. The behavior — what is extracted and surfaced — is fixed.
- **Typed error shape and name** `[ASSUMED]`: the name of the typed API error and its field names are adjustable at planning. The behavior is fixed: it carries the authoritative HTTP status, the extracted `detail`/`title`/`type` when available, the raw body, and the response headers.
- **Fallback detail wording** `[ASSUMED]`: when the body cannot be parsed, the fallback detail is derived from the HTTP status (e.g. the standard reason phrase for that code). The exact wording is a planning detail; the behavior — a non-empty detail always present — is fixed.
- **Problem Details contract**: the API serves every 4xx/5xx as RFC 9457 Problem Details (`type`, `title`, `status`, `detail` always present on conformant bodies, `application/problem+json`). This capability relies on that documented contract per the project's "spec is the contract" constraint, while degrading when a non-conformant body arrives.

---

## Ambiguity Warnings

_None remaining — the three behavioral forks were resolved during the defining conversation: (1) when the body is not a readable Problem Details document, the system degrades gracefully — it still produces a typed error from the HTTP status with a fallback detail and the raw body preserved, never failing to produce an error; (2) when the body's `status` member disagrees with the HTTP status, the HTTP status is authoritative and the body's status is carried as metadata only; and (3) the capability stays narrow — it carries the numeric status plus the extracted detail/title/type as one structured error and leaves `429`-backoff (017) and `403` plan-limit messaging (Unsignalled Plan Limits) to their own capabilities. The remaining `[ASSUMED]` items are planning-time shape details, not behavioral gaps._
