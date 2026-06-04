# Specification: Request Authentication

**Feature**: 007-request-authentication
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Request Authentication is the consuming capability of Token Authentication (problem: *Unauthenticated Access* — the CLI has no way to prove it is acting as a specific org + person, so Glassfrog can't authorize its calls). Its two siblings prepare the credential: Credential Storage (006) writes the token, Credential Discovery (005) resolves it. This capability is where that resolved identity meets the wire — it ensures every outgoing Glassfrog API call carries the `X-Auth-Token` header for the resolved org + person, so the API authorizes the call as that identity.

It is deliberately narrow: it owns the *authentication* of a request, not the request itself. Credential Discovery is its upstream dependency and the only resolver — Request Authentication consumes Discovery's output (a resolved token and its source, or a no-credentials / error outcome) and never re-resolves. Connection Configuration owns the HTTP transport — constructing and sending the call; Request Authentication only decorates the outgoing request with the auth header. Like its siblings, its operator is usually a non-interactive AI agent, so it never prompts, never exposes the secret, and — honouring Fail Safe — never falls back to an anonymous call when no token is available.

---

## Behavioral Accord

### Attaching identity

- When a token has been resolved for the current invocation, the system attaches it as the `X-Auth-Token` header on every outgoing Glassfrog API request, so the API authorizes the call as the org + person the token is scoped to.
- When the header is attached, the system uses the resolved token value exactly as Discovery reported it — it does not trim, re-encode, or otherwise transform the value.

### Fail safe — no usable credential

- When Discovery reports that no credentials were found, the system does not send the request and surfaces a "cannot authenticate — no credentials" outcome.
- When Discovery reports a read or format error on a credentials file, the system does not send the request and surfaces a "cannot authenticate — credential error" outcome that names the underlying cause, distinct from the no-credentials outcome.
- In no case does the system fabricate a token, prompt for one, or send the request without the `X-Auth-Token` header.

### Reporting the outcome

- When a request is authenticated, the system reports the active identity's source — the environment variable, or the credentials file path, as Discovery reported it — so the operator can tell which credentials are in use, without exposing the token value.
- The system reports its outcome (authenticated, cannot-authenticate-no-credentials, or cannot-authenticate-credential-error) to the consuming command; it does not decide the process exit code or the user-facing message for that outcome.

---

## User Scenarios

**In order to** have my CLI commands authorized as me without managing headers by hand,
**as a** practitioner who stored credentials once,
**I want** the CLI to attach my resolved identity to every API call automatically.

**In order to** fail safely in automation instead of issuing anonymous calls,
**as an** AI agent,
**I want** the CLI to refuse to reach the API when no credential is available, and tell me why.

**In order to** confirm which identity a command ran as,
**as an** operator who moves between projects and tokens,
**I want** the CLI to report the credential source used — never the secret itself.

---

## Non-Behaviors

- The system must not resolve, read, search for, or choose between credential sources. **Why**: resolution is Credential Discovery's (005) sole job; a second resolution path here would drift from the first and split the precedence contract across two capabilities.
- The system must not own the HTTP transport — base URL, connection, retries, timeouts, response parsing, or pagination. **Why**: that is Connection Configuration's capability; entangling transport here would couple the auth seam to the whole client and blur which capability owns the network.
- The system must not interpret the API's authentication response — a `401` (rejected token) or `403` (permission / premium-gated rejection). **Why**: this capability owns only the outgoing direction; classifying a response belongs to the response-handling and Exit-Code path, which maps such a rejection to its code.
- The system must not decide the process exit code or the message shown when authentication can't proceed. **Why**: Exit-Code Convention (004) and the consuming command own classification; this capability reports the outcome and lets that mapping happen downstream.
- The system must not send an unauthenticated request as a fallback when no token is available. **Why**: Fail Safe — an anonymous call could silently act as a different (or no) identity and return misleading results; a missing credential must surface loudly, not degrade quietly.
- The system must not print, log, or otherwise expose the token value. **Why**: the token is a secret, and the request side is exactly where it tends to leak (request logs, verbose tracing); the same secret-never-emitted rule that governs Discovery applies here.
- The system must not prompt for or solicit a token when none is found. **Why**: the operator is usually a non-interactive AI agent; blocking on a prompt would hang automation. (Carried from Discovery.)

---

## Integration Boundaries

- **Credential Discovery (005 — upstream dependency)**: provides the resolved token and its source, or the no-credentials / read-error / format-error outcome. Request Authentication consumes this output and never re-resolves. If Discovery reports absence or error, Request Authentication does not reach the API.
- **Connection Configuration (transport sibling — modeled in parallel)**: owns constructing and sending the HTTP request to the Glassfrog API. Request Authentication decorates the outgoing request with the `X-Auth-Token` header; Connection Configuration sends it. The precise seam — whether auth sets the header on a request object that Connection Configuration sends, or Connection Configuration asks auth for the header — is a coordination item to reconcile during planning (see Assumptions).
- **Glassfrog API (downstream system)**: receives the `X-Auth-Token` header and authorizes the call as the org + person the token is scoped to, enforcing permissions per that identity (PROJECT constraint). Request Authentication produces the authenticated request; it does not act on the response.
- **Exit-Code Convention (004 — downstream)**: a cannot-authenticate outcome ultimately accompanies a non-zero exit code, but the classification and mapping belong to that capability and the consuming command, not to this one.

---

## Driving Scenarios

### Happy path

**Scenario: Resolved token is attached to the outgoing call**
Given Credential Discovery resolved a token from a source
When an API request is about to be sent
Then the system sets the `X-Auth-Token` header to the resolved token
And the request proceeds to be sent.

**Scenario: Active identity source is reported without exposing the secret**
Given Discovery resolved a token from a home-directory credentials file
When the request is authenticated
Then the system reports the source as that file path
And the token value does not appear anywhere in the output.

**Scenario: The same identity applies across multiple calls in one invocation**
Given a token was resolved for the invocation
When a command makes more than one API request
Then every outgoing request carries the same `X-Auth-Token` identity.

### Error scenarios

**Scenario: No credentials — refuse to call**
Given Discovery reports that no credentials were found
When an API request would be sent
Then the system does not send the request
And surfaces a "cannot authenticate — no credentials" outcome
And does not fabricate a token or send an anonymous request.

**Scenario: Credential error — refuse to call and name the cause**
Given Discovery reports a format error on a credentials file
When an API request would be sent
Then the system does not send the request
And surfaces a "cannot authenticate — credential error" outcome naming that file
And does not report it as "no credentials".

### Edge cases

**Scenario: Token is attached verbatim**
Given Discovery resolved a token whose value contains characters that look unusual
When the header is attached
Then the header value equals the resolved token exactly, with no characters added, removed, or re-encoded.

**Scenario: Token is redacted from request diagnostics**
Given diagnostic or verbose request output is produced
When a request carries the `X-Auth-Token` header
Then the token value is omitted or redacted from that diagnostic output.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The token value never appears in produced output**
Given any outcome (authenticated, no-credentials, or credential-error)
When the capability's output and diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: No request is ever sent unauthenticated**
Given any starting credential state
When the capability runs to completion
Then there is no path on which an API request is sent without the `X-Auth-Token` header — absence of a token ends in a cannot-authenticate outcome, never an anonymous call.

**Scenario: Authentication performs no resolution of its own**
Given Discovery is the only resolver
When authentication runs
Then the only source of the attached token is Discovery's output — the capability reads no environment variable and no credentials file directly.

---

## Assumptions

- **Header name** (pinned, not assumed): the authentication header is `X-Auth-Token`, the Glassfrog v5 scheme fixed by the PROJECT constraints and Discovery's hand-off. Recorded here for context — it is a fixed contract, not a provisional choice.
- **Connection Configuration seam** `[ASSUMED]`: Request Authentication attaches the header to the outgoing request that Connection Configuration sends; the precise mechanism of the seam is deferred to planning and must be reconciled with the Connection Configuration spec being modeled in parallel. (Coordination item — flagged so both capabilities agree on one boundary, mirroring the Discovery/Storage file-contract reconciliation.)
- **Identity established once per invocation**: a single resolved token applies to every API call within one CLI invocation. (Follows the single-org-+-person-per-key PROJECT constraint and Discovery's deterministic resolution.)
- **Secret handling carried from Discovery**: the token-never-emitted rule and the no-interactive-prompt rule are inherited as shared constraints across the Token Authentication capabilities, not re-derived here.

---

## Ambiguity Warnings

_None remaining — the capability boundary (header attachment only, transport owned by Connection Configuration), the fail-safe behaviour on missing or broken credentials (refuse to call, surface a distinct outcome, decide no exit code), the response-side boundary (no `401`/`403` interpretation), and the dependency on Discovery for resolution were all resolved during the defining conversation. The Connection Configuration seam is recorded as an `[ASSUMED]` coordination item to reconcile during planning; it is a cross-capability boundary detail, not a behavioral gap in this spec._
