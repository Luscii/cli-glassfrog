# Specification: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Connection Context Assembly is the consuming half of **Connection Configuration** (problem: *Undefined Connection Settings* — the CLI doesn't know which token or base URL to use, or where to read them from). Its sibling **Base URL Resolution** (008) resolves the effective endpoint; **Credential Discovery** (005) resolves the token. This capability combines those two outputs into the single **connection context** every request hangs off — the one bundle that answers "which endpoint, as which identity, right now?" It gates **Request Execution** (010), the API-client seam, so it must exist before the client can issue a call.

It is a **transparent aggregator**, deliberately in the same "report, don't decide" lane as its siblings: it consumes the base URL outcome and the credential outcome, pairs them, and reports an aggregate readiness — but it re-resolves nothing, makes no API call, writes no file, decides no exit code, and never refuses a request (that fail-safe is **Request Authentication**'s). When either input is missing or errored, the context still assembles and carries the problem forward rather than short-circuiting. Note: this CLI-internal "connection context" is distinct from the Glassfrog domain term **Context** (the hypermedia document for filling a role, per PROJECT.md vocabulary); the two are kept separate.

---

## Behavioral Accord

### Assembling the context

- When a connection context is requested, the system obtains the base URL outcome from Base URL Resolution and the credential outcome from Credential Discovery and combines them into a single connection context, without resolving either value itself.
- When the inputs are obtained, the context carries the base URL together with its source, and the credential outcome together with its source — each exactly as its resolver reported it, with no transformation.

### Carry-forward & transparency

- When Credential Discovery reports no credentials, the system still produces a context — carrying the resolved base URL and a credential outcome of "absent" — and does not refuse, fabricate a token, or treat the absence as an error of its own.
- When a resolver reports an error (a base-URL format or read error, or a credential read or format error), the system carries that error into the context naming its source, rather than dropping it or falling back.
- When both inputs report a problem at once, the system surfaces both in the context rather than stopping at the first.

### Readiness reporting

- When the context carries a usable base URL and a present token, the system reports the context as complete — every part needed for an authenticated request is present.
- When the base URL, the token, or both is missing or errored, the system reports the context as incomplete and names which part is missing or errored, without deciding whether the request proceeds or what exit code accompanies it.

### Lifecycle

- When a context has been assembled for an invocation, the same context applies to every API request in that invocation; the system does not re-assemble or re-resolve per request.

### Secret handling

- When the context is reported, inspected, or rendered in diagnostics, the token value never appears — only its presence and its source.

---

## User Scenarios

**In order to** point a command at the right endpoint as the right identity without wiring the two together by hand,
**as an** AI agent driving the CLI,
**I want** one connection context that already pairs the resolved base URL with the resolved token.

**In order to** understand exactly what is and isn't ready before a call is attempted,
**as an** operator diagnosing a misconfigured setup,
**I want** the context to tell me whether it is complete and, if not, which part is missing or broken — all in one look.

**In order to** keep the same identity and endpoint across every call a command makes,
**as a** practitioner running a multi-request command,
**I want** the context assembled once and reused, not re-derived per request.

---

## Non-Behaviors

- The system must not resolve, read, or choose the base URL or the token itself. **Why**: Base URL Resolution (008) and Credential Discovery (005) own resolution; a second path here would drift from their precedence contracts and split resolution across capabilities.
- The system must not decide whether the request proceeds or refuse a request when the token is absent or errored. **Why**: that fail-safe decision is Request Authentication's (007); deciding it here would split the fail-safe contract across two capabilities.
- The system must not decide the process exit code or the user-facing message for an incomplete context. **Why**: Exit-Code Convention (004) and the consuming command own classification; assembly reports readiness and lets that mapping happen downstream.
- The system must not make any API call or check that the base URL is reachable or the token valid. **Why**: assembly is offline and deterministic; reachability and credential validity are the request's concern, not assembly's.
- The system must not transform the carried values — no normalizing the URL, no building or encoding the auth header, no trimming the token. **Why**: 008 passes the URL through as given and 007 owns the header; composing or rewriting here would duplicate or contradict those owners and could mask a misconfiguration.
- The system must not write, create, or modify any file. **Why**: it is a pure assembler; Credential Storage owns writing, and a second writer would split the file contract.
- The system must not print, log, or expose the token value. **Why**: the token is a secret; the same secret-never-emitted rule that governs Discovery and Request Authentication applies here.
- The system must not prompt interactively for a base URL or token. **Why**: the operator is usually a non-interactive AI agent; a missing part is reported in the context, not solicited.
- The system must not support multiple connection contexts, profiles, or per-host entries. **Why**: an API key is scoped to a single org + person (PROJECT constraint), so one context is the whole need.

---

## Integration Boundaries

- **Base URL Resolution (008 — upstream dependency)**: provides the resolved base URL and its source, or a format/read error. Assembly consumes this and never re-resolves; an error is carried into the context, not retried.
- **Credential Discovery (005 — upstream dependency)**: provides the resolved token and its source, or a no-credentials / read-error / format-error outcome. Assembly consumes this and never re-resolves.
- **Request Authentication (007 — sibling consumer)**: reads the credential portion to attach the `X-Auth-Token` header and owns the fail-safe refusal when no usable token is present. Whether 007 reads the token from this context or directly from Discovery is a planning-time seam to reconcile (see Assumptions) — the mirror of 007's own "Connection Configuration seam" item.
- **Request Execution / API Client (010 — downstream consumer)**: reads the base URL as the request root and uses the assembled context for every call. This capability produces the context; Request Execution consumes it.
- **Exit-Code Convention (004 — downstream)**: an incomplete context ultimately informs the exit code that accompanies a command, but the classification and mapping belong to that capability, not this one.

---

## Driving Scenarios

### Happy path

**Scenario: Complete context from a usable base URL and a present token**
Given Base URL Resolution reports a usable base URL with its source
And Credential Discovery reports a present token with its source
When a connection context is requested
Then the system produces a context carrying the base URL and its source and the token and its source
And reports the context as complete.

**Scenario: Built-in default base URL paired with a token still completes**
Given Base URL Resolution reports the built-in default base URL as its source
And Credential Discovery reports a present token
When a connection context is requested
Then the system produces a complete context
And the base URL source is reported as the built-in default.

**Scenario: One context applies across multiple calls in an invocation**
Given a connection context was assembled for the invocation
When a command makes more than one API request
Then every request uses the same assembled context
And the system does not re-assemble or re-resolve between requests.

### Error scenarios

**Scenario: No credentials — context still assembles, carrying the absence**
Given Base URL Resolution reports a usable base URL
And Credential Discovery reports that no credentials were found
When a connection context is requested
Then the system produces a context carrying the base URL and a credential outcome of "absent"
And reports the context as incomplete, naming the missing credential
And does not refuse the request, fabricate a token, or decide an exit code.

**Scenario: Both inputs report a problem — both are surfaced**
Given Base URL Resolution reports a format error naming its source
And Credential Discovery reports that no credentials were found
When a connection context is requested
Then the system produces a context surfacing both the base-URL error and the absent credential
And reports the context as incomplete
And does not stop at the first problem.

### Edge cases

**Scenario: Base URL error while a token is present**
Given Base URL Resolution reports a read error naming a config file
And Credential Discovery reports a present token
When a connection context is requested
Then the system carries the base-URL error into the context naming that file
And carries the present token and its source intact
And reports the context as incomplete, naming the base-URL part.

**Scenario: Token is redacted from the rendered context**
Given a complete context has been assembled
When the context is rendered or written to diagnostics
Then the token's source and presence appear
And the token value itself does not appear anywhere in the output.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Assembly re-resolves nothing**
Given Base URL Resolution and Credential Discovery are the only resolvers
When a context is assembled
Then the only base URL and token in the context are those its resolvers reported
And the capability reads no flag, environment variable, or file directly.

**Scenario: Assembly performs no writes and no network call**
Given any starting filesystem and network state
When a context is assembled
Then the filesystem is unchanged afterward
And no outbound connection or API call is made during assembly.

**Scenario: The token value never appears in produced output**
Given any outcome (complete, incomplete, or error-carrying)
When the context's output and diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: Assembly is deterministic**
Given an unchanged set of resolver outputs
When a context is assembled twice
Then both produce the same context with the same readiness and the same carried sources.

---

## Assumptions

- **Credential outcome set** `[ASSUMED]`: the context distinguishes the credential portion as present (with source), absent, read error, or format error — mirroring exactly the outcomes Credential Discovery (005) reports. (Pinned to 005's reported outcome set so the two do not drift.)
- **Readiness terminology** `[ASSUMED]`: the aggregate signal is named "complete" / "incomplete". (Adjustable without changing any behavior.)
- **Request Authentication seam** `[ASSUMED]`: whether Request Authentication (007) reads the token from this connection context or directly from Credential Discovery is a coordination item deferred to planning. This is the same seam 007 records from its side; the two must agree on one boundary. (Cross-capability boundary detail, not a behavioral gap.)
- **Single context per invocation**: one context is assembled per CLI invocation and reused. (Follows the single-org-+-person-per-key PROJECT constraint and the deterministic resolution of both inputs.)
- **Term disambiguation**: "connection context" is a CLI-internal aggregate of endpoint + identity, distinct from the Glassfrog domain term "Context" (the role-filling hypermedia document in PROJECT.md vocabulary). (Recorded so the term is not overloaded downstream.)

---

## Ambiguity Warnings

_None remaining — the three behavioral forks were resolved during the defining conversation: the capability is a transparent aggregator that reports readiness without deciding to refuse (the fail-safe stays with Request Authentication 007); it carries both sub-outcomes forward rather than short-circuiting on the first problem; and the context carries the base URL + source, the credential outcome + source, and the aggregate readiness, with nothing composed or derived. The Request Authentication seam is recorded as an `[ASSUMED]` coordination item to reconcile during planning — a cross-capability boundary detail, not a behavioral gap in this spec._
