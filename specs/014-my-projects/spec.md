# Specification: My Projects

**Feature**: 014-my-projects
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

My Projects is one of the four **Self-Service Reads** (problem: *read "what's mine" — me, my roles, actions, and projects*). It is a read command that lists the **projects owned by roles the authenticated practitioner fills** — token-scoped through `GET /me/projects` (`listMyProjects`), not the org-wide project surface. It answers "what projects am I responsible for?" for the practitioner the API key resolves to.

It sits on the proven read chain: identity rides through Request Authentication (007), and the call goes out through Request Execution (010), the single transport seam. It is the structural twin of My Actions (013) and a sibling of Identity Read (011) and My Roles (012), sharing their `/me*` read-presentation convention. It is deliberately narrow: it fetches **one page**, applies an optional **status filter** validated against the spec's vocabulary, and renders the result through the shared projection. It does not walk pages (Pagination, 016), does not interpret non-2xx responses (API Error Extraction, 015), and does not back off on `429` (Rate-Limit Handling, 017) — those siblings own that work and build on what Request Execution surfaces.

---

## Behavioral Accord

### Listing

- When invoked, the system requests the authenticated actor's projects from `GET /me/projects` through Request Execution, so the call carries the resolved identity without this command attaching the header or re-resolving the connection.
- When the API returns a successful response, the system renders the projects it received through the shared `/me*` read-presentation projection — a reshaped, summarized view, not the raw API payload.

### Filtering

- When invoked with a status filter that the spec's status vocabulary defines, the system sends it as the request's `status` query parameter.
- When invoked with a status value outside the spec's vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how Identity Read, 011, validates its `--include` target).
- When invoked with no status filter, the system requests the actor's projects without a status constraint.

### Pagination boundary

- When the result set spans more than one page, the system renders only the first page and surfaces a clear "more results are available" signal, so the consumer knows the list is not complete.
- The system makes a single page request and never walks to subsequent pages itself; fetching the remainder is deferred to Pagination (016).

### Empty result

- When the practitioner owns no projects, or none match the supplied status filter, the system reports success with an empty list — zero matches is a valid answer, not a failure.

### Error handling

- When no usable token is available, the system does not front-run the decision: the request goes through the authenticated transport (007), whose fail-safe refuses the unauthenticated call, and the system propagates that authentication outcome unchanged; no list is produced.
- When the request fails at the wire or the API answers with a non-2xx response, the system surfaces the typed outcome Request Execution returns; it does not classify which kind of failure it is (deferred to API Error Extraction, 015) nor decide the process exit code (deferred to Exit-Code Convention, 004, and the command surface).

---

## User Scenarios

**In order to** see the projects I'm responsible for without opening the Glassfrog web app,
**as a** practitioner (usually via an AI agent),
**I want to** list the projects owned by the roles I fill.

**In order to** focus on just the work that is live,
**as a** practitioner triaging my projects,
**I want to** filter the list by status (e.g. only `current`).

**In order to** know when a result is incomplete rather than silently truncated,
**as an** AI agent consuming the output,
**I want** a clear signal when more results exist beyond the page I received.

---

## Non-Behaviors

- The system must not walk pagination or fetch beyond the first page. **Why**: Pagination (016) owns completing the result set across all `/me*` lists; duplicating it here would drift from that one contract and hide the boundary this command is meant to signal.
- The system must not send a `--status` value the spec's vocabulary does not define. **Why**: an unsupported filter is a usage error the contract will refuse; catching it locally (as Identity Read, 011, does for `--include`) gives the agent operator a usage-error exit and the supported set to self-correct, rather than an opaque API error indistinguishable from a real failure.
- The system must not embed sub-projects or actions, nor expose an `?include=` option. **Why**: the `/me/projects` operation does not offer an `include` parameter, so embedding is outside the endpoint's contract; the list view's `has_sub_projects` / `has_actions` flags already signal their presence.
- The system must not render the raw API payload or implement its own output shape. **Why**: the shared `/me*` read-presentation projection (established by Identity Read, 011) keeps every self-service read legible and consistent for the AI-agent operator; a one-off shape here would fork that convention.
- The system must not expose a `--output json` (raw/structured) mode. **Why**: a structured-output flag is a deferred, cross-cutting capability (011 anticipates it as a persistent root-level flag, not a per-command one); introducing it here would pre-empt that shared decision and fork the surface.
- The system must not resolve, read, or choose the token or base URL, nor attach the `X-Auth-Token` header. **Why**: Credential Discovery (005), Base URL Resolution (008), Connection Context Assembly (009), and Request Authentication (007) own those; a second path here would split their contracts.
- The system must not interpret a non-2xx response into a specific error, nor decide the process exit code or user-facing message. **Why**: API Error Extraction (015), Exit-Code Convention (004), and the command surface own classification and mapping.
- The system must not read the org-wide project surface. **Why**: this is the *self-service* "what's mine" read — it is token-scoped to `GET /me/projects`, and reaching past that would exceed the caller's self-service intent.
- The system must not create, update, or otherwise mutate projects. **Why**: standalone operational writes are out of scope for the project's read + propose core; this is a read.
- The system must not prompt interactively. **Why**: the operator is usually a non-interactive AI agent; a missing or broken part is surfaced as a typed outcome, not solicited.

---

## Integration Boundaries

- **Request Execution (010 — upstream dependency)**: the seam this command sends its single `GET /me/projects` request through and reads the response or typed error from. This command re-resolves nothing and makes no transport decisions of its own.
- **Request Authentication (007 — upstream dependency)**: provides the authenticated transport that attaches the token and fail-safe-refuses when none is usable. This command composes its request through it and propagates its outcome.
- **Glassfrog API — `GET /me/projects` (`listMyProjects`)**: returns a paginated list of projects owned by roles the requester fills (primary assignments), with an optional `status` query filter. The API owns permission scoping; the status vocabulary is the spec's enum the command validates against. No `include` parameter is offered on this operation.
- **Identity Read (011) — sibling**: established the shared `/me*` read-presentation projection and the `classifyClientError` error→exit-code mapping; this command renders through that projection and reuses that classification rather than reinventing either.
- **My Roles (012) — sibling**: the first paginated `/me*` list; establishes the "more results available" signal convention this command's first-page boundary follows. My Actions (013) is this command's twin over a different resource.
- **Pagination (016 — downstream sibling)**: will own walking past the first page. This command surfaces the "more available" boundary (per 012's convention) that 016 builds on.
- **Exit-Code Convention (004 — downstream)**: the command's outcome ultimately informs its exit code, but the mapping belongs there and to the command surface, not this capability.

---

## Driving Scenarios

### Happy path

**Scenario: list the practitioner's projects**
Given a connection context with a usable base URL and a present token
When the my-projects command is invoked with no status filter
Then the system requests `GET /me/projects` through Request Execution
And renders the first page of returned projects through the shared `/me*` projection.

**Scenario: filter by a supported status**
Given a present token
When the command is invoked with a status filter of `current`
Then the value is accepted as a supported status
And the request carries `status=current`
And only the projects the API returns for that filter are rendered.

**Scenario: more results than one page**
Given the practitioner owns more projects than fit on one page
When the command is invoked
Then the system renders only the first page
And surfaces a clear "more results are available" signal
And does not request a second page.

### Error scenarios

**Scenario: no usable token**
Given a connection context with a usable base URL but no usable token
When the command is invoked
Then the authenticated transport's fail-safe refuses the unauthenticated call
And the system propagates that authentication outcome without producing a list.

**Scenario: API responds with a non-2xx**
Given a present token
When the command is invoked and the API answers with a non-2xx response (e.g. unauthorized or not found)
Then the system surfaces the typed outcome Request Execution returns
And does not classify the failure or decide the exit code itself.

### Edge cases

**Scenario: no matching projects**
Given the practitioner owns no projects, or none match the supplied status filter
When the command is invoked
Then the system reports success with an empty list.

**Scenario: invalid status value is rejected before any request**
Given the command is invoked with a status value outside the spec's vocabulary
When the command runs
Then the system rejects it as a usage error, naming the unsupported value and the supported set
And issues no request (mirroring Identity Read's `--include` validation).

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: the command re-resolves nothing**
Given the connection context and authenticated transport are the only inputs
When the command runs
Then the only transport used is Request Execution (010)
And the command reads no flag, environment variable, or credentials file directly for the token or base URL.

**Scenario: exactly one page request**
Given any size of result set
When the command runs
Then exactly one `GET /me/projects` request is made
And no subsequent page is fetched, regardless of whether more results exist.

**Scenario: an unsupported status costs no request**
Given a status value outside the spec's vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: output is the shared projection, not raw payload**
Given any successful response
When the output is inspected
Then it is the reshaped/summarized `/me*` projection
And the raw API JSON payload is not emitted (no `--output json` mode exists yet).

---

## Assumptions

- **Command surface** `[ASSUMED]`: the exact CLI invocation (e.g. a `my projects` command, per the `my <noun>` form Identity Read implies for the dedicated reads) is a planning/interface detail aligned with the shared `/me*` surface and pinned alongside My Roles (012); the behavior — what is requested and what is surfaced — is fixed regardless of the final spelling.
- **Shared output projection**: output defers to the `/me*` read-presentation convention established by Identity Read (011) — a reshaped, summarized projection rather than raw API JSON. This command renders through that convention rather than defining its own.
- **`--output json` is deferred**: a raw/structured machine-output flag is a planned, cross-cutting future capability (likely a persistent root flag per 011), out of scope here.
- **"More available" signal shape** `[ASSUMED]`: how the first-page-only boundary is surfaced follows the convention My Roles (012) establishes as the first paginated `/me*` list; the behavior — one page, signal that more exist, no walk — is fixed. If 012 lands a different signal, this tracks it.
- **Status validation set**: `--status` is validated against the spec's status vocabulary (`archived, cancelled, completed, current, scheduled, someday, waiting`) before any request, mirroring 011's `validateInclude`; a supported value maps to the `status` query parameter. The accepted set tracks the spec (vendored at `spec/glassfrog-api-v5.yaml`); the exact flag name and whether validation shares a helper with 011 are planning details.

---

## Ambiguity Warnings

_None remaining — the four behavioral forks were resolved during the defining conversation (fork 2 later aligned to Identity Read's precedent during verification against the shaped 011 artifacts): (1) the command fetches only the first page and surfaces a "more available" signal, deferring multi-page walking to Pagination (016) and following the signal convention My Roles (012) sets; (2) `--status` is validated locally against the spec's status set before any request — an unsupported value is a usage error, mirroring 011's `--include` — then sent to the API; (3) output defers to the shared `/me*` projection that Identity Read (011) establishes, with a structured `--output json` mode deferred as a cross-cutting capability; and (4) an empty or fully-filtered-out result is a success with an empty list, not an error. The remaining `[ASSUMED]` items are planning-time shape details, not behavioral gaps._
