# Specification: Identity Read

**Feature**: 011-identity-read
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Identity Read is the CLI's first end-to-end read — a `me` command that prints the authenticated actor (the person or agent the token resolves to) together with its organization and membership. It is the smallest call that proves the whole chain end-to-end: connection context (009) → authenticated transport (007) → Request Execution (010) → a parsed governance read surfaced to the operator. It maps one-to-one onto the spec's `GET /me` → `getMe` operation, the call the API itself documents as "typically the first an API consumer makes to orient itself."

It is deliberately the thinnest possible read on top of the API Client. It owns *one* request shape (`GET /me`, with an opt-in roles embed), the projection of that response into agent-legible output, and the mapping of Request Execution's typed outcome onto a process result. It does not own transport, identity, the base URL, paging, or error classification — those belong to the siblings it composes. Its value is to make the proven chain visible and to set the shape every later read (My Roles, My Actions, My Projects) follows.

---

## Behavioral Accord

### Entry

- When the operator runs the `me` command with no arguments, the system issues the authenticated `GET /me` read and prints the resolved identity.
- When the operator adds the opt-in roles flag (e.g. `me --include roles`), the system requests the API embed the requester's roles in the same response, so an agent orienting itself gets identity and role context in one call.
- When the operator supplies an `--include` target the spec does not define, the system rejects it as a usage error before issuing any request, naming the unsupported target and the supported set — it does not send a request it knows the contract will refuse.

### Reading

- When the command runs, it sends the request through Request Execution (010), supplying `GET`, the `/me` path, the `include` query parameter only when the flag was given, and a decode target for the `MeResponse` shape — it does not attach identity, resolve the base URL, or open a connection itself.

### Output

- When the read succeeds, the system prints a reshaped, predictable projection of the actor's identity: the actor's id, name, and kind (human or agent); the organization's id and name; and the membership access level. The id values are always surfaced because they are the machine-actionable handles an agent uses in follow-up calls.
- When the roles embed was requested and the response carries roles, the projection additionally lists each embedded role's identifying facts (id and name); when no roles are present, the projection omits the roles section rather than printing an empty one.
- When the read succeeds, the command completes with the success result, so the operator (or a wrapping agent) can tell the call worked without inspecting output.

### Failure

- When Request Execution reports a transport failure (the call could not reach or complete at the wire), the system surfaces that typed failure, naming it, and completes with a non-success result; it prints no identity projection.
- When Request Execution reports a non-2xx response (e.g. an expired or wrong token answered with `401`), the system surfaces the generic outcome carrying the status code, and completes with a non-success result; it does not interpret which kind of failure the status represents.
- When Request Execution reports the no-token fail-safe or a base-URL problem, the system propagates that outcome unchanged and completes with a non-success result; it does not re-resolve or re-decide either concern.
- The system reports its outcome through the standard exit-code mapping (004); it does not invent its own exit codes.

---

## User Scenarios

**In order to** confirm my token works and learn which actor and organization it resolves to,
**as an** AI agent making its first call,
**I want to** run one command that prints who I am and where I am.

**In order to** get identity and the roles I fill in a single round-trip,
**as an** AI agent about to act on a practitioner's behalf,
**I want to** opt into embedding my roles in the same `me` read.

**In order to** tell "my token is bad" apart from "the network failed" when a command fails,
**as an** operator diagnosing a failed run,
**I want** the `me` command to surface the transport-versus-response distinction Request Execution already draws.

---

## Non-Behaviors

- The system must not emit raw or structured JSON output. **Why**: the default surface is a reshaped, summarized projection; a verbatim/structured `--output json` mode is a planned future capability, and shipping it implicitly here would pre-empt that decision and freeze a format the project has not yet committed to.
- The system must not attach the `X-Auth-Token` header, resolve the base URL, open the connection, or apply a timeout itself. **Why**: Request Authentication (007), Base URL Resolution (008), Connection Context Assembly (009), and Request Execution (010) own those; duplicating any would drift from their contracts.
- The system must not interpret a non-2xx status into a specific, meaningful API error or extract the API's error detail. **Why**: API Error Extraction (015) owns turning a raw status and body into a typed error; `me` surfaces the generic outcome so 015 can later enrich it without changing this command's contract.
- The system must not own the full roles-reading surface — filtering, status, paging, or the comprehensive role rendering. **Why**: My Roles (012) owns the dedicated roles read; `me --include roles` only passes the API's optional embed through its identity projection.
- The system must not follow pagination, retry, or back off on `429`. **Why**: `/me` is a single resource; paging (016) and rate-limit handling (017) own those concerns for the list reads that need them.
- The system must not print, log, or expose the token value in output or diagnostics. **Why**: the token is a secret and the read path is exactly where it leaks; the same secret-never-emitted rule that governs Discovery, Storage, Authentication, and Request Execution applies here.
- The system must not prompt interactively for anything. **Why**: the primary operator is a non-interactive AI agent; a missing or broken part is surfaced as a typed outcome, not solicited.
- The system must not span or switch organizations. **Why**: a token is scoped to one org and one person (PROJECT constraint); `me` reports that single resolved identity and never tries to read another.

---

## Integration Boundaries

- **Request Execution (010 — upstream dependency)**: `me` hands it the `GET /me` request and a `MeResponse` decode target, and reads back either a decoded success or a typed transport / non-2xx / decode outcome. `me` adds no transport behavior of its own.
- **Request Authentication (007 — transitive)**: identity is attached by the authenticated transport that 010 sends through; the no-token fail-safe is propagated, not owned here.
- **Connection Context Assembly (009) / Base URL Resolution (008 — transitive)**: the base URL and its source come from the assembled context that 010 consumes; `me` never re-resolves them.
- **Glassfrog API `GET /me` (system actor)**: returns `MeResponse` (actor + organization + membership, plus an optional embedded `roles` array when `?include=roles` is requested) on `200`; answers `401` for an unusable token and `404` per the spec. `me` reads the response 010 surfaces.
- **Exit-Code Convention (004 — downstream)**: the success-versus-failure outcome maps to the process exit code through 004; `me` reports the outcome and lets that mapping happen.
- **My Roles (012 — sibling)**: owns the standalone roles read; `me`'s roles embed is the API's convenience affordance, not a second roles surface.

---

## Driving Scenarios

### Happy path

**Scenario: me prints the resolved identity**
Given a connection context with a usable base URL and a present, valid token
When the operator runs `me`
Then the system issues `GET /me` through Request Execution
And prints a projection carrying the actor's id, name, and kind, the organization's id and name, and the membership access level
And completes with a success result.

**Scenario: me distinguishes a human from an agent**
Given the token resolves to an agent actor (an `agt_` id with kind `agent`)
When the operator runs `me`
Then the projection reports the actor's kind as agent
And surfaces the `agt_` id as the actionable handle.

**Scenario: me embeds roles on request**
Given a connection context with a usable base URL and a present, valid token
When the operator runs `me --include roles`
Then the request carries the `include=roles` query parameter
And the projection lists each embedded role's id and name in addition to the identity facts.

### Error scenarios

**Scenario: an unusable token surfaces a non-2xx outcome**
Given a present but expired or wrong token
When the operator runs `me`
Then Request Execution returns a non-2xx outcome carrying the `401` status
And the system surfaces that outcome with its status code
And prints no identity projection
And completes with a non-success result.

**Scenario: a transport failure is surfaced as transport, not as a response**
Given the API cannot be reached (connection refused, DNS failure, or timeout)
When the operator runs `me`
Then the system surfaces the typed transport failure, naming it
And completes with a non-success result
And does not retry.

### Edge cases

**Scenario: no usable token — the fail-safe is propagated**
Given a connection context with a usable base URL but no usable token
When the operator runs `me`
Then the authenticated transport's no-token fail-safe refuses the call
And the system propagates that outcome unchanged and completes with a non-success result
And no unauthenticated request is sent.

**Scenario: an unsupported include target is rejected before any request**
Given a connection context with a usable base URL and a present, valid token
When the operator runs `me --include actions` (a target the spec's `include` set does not define)
Then the system rejects the input as a usage error, naming the unsupported target and the supported set
And issues no request.

**Scenario: roles embed requested but the actor fills none**
Given a valid token whose actor fills no roles
When the operator runs `me --include roles`
Then the projection prints the identity facts
And omits the roles section rather than printing an empty list.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: me resolves nothing itself**
Given the `me` command runs
When the request is issued
Then the only transport used is Request Execution (010), the only identity the authenticated transport (007)
And `me` reads no flag, environment variable, or credentials file directly to build the request.

**Scenario: the token value never appears in produced output**
Given any outcome (success projection, transport error, non-2xx error, or fail-safe refusal)
When the output and any diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: no structured-JSON output leaks in**
Given a successful read
When the output is inspected
Then it is the reshaped identity projection
And it is not raw or structured JSON (the `--output json` mode remains a future capability).

**Scenario: a non-2xx status is surfaced, not classified**
Given the API answers `GET /me` with a `404`
When the command runs
Then the system surfaces the generic non-2xx outcome carrying the status
And does not turn it into a specific, interpreted API error message.

---

## Assumptions

- **Command surface** `[ASSUMED]`: the command is invoked as `me` (a top-level command per the FEATURE-MODEL). The exact name registration and whether it lives at the root are command-registration (001) details; the behavior — one identity read — is fixed.
- **Roles-embed flag shape** `[ASSUMED]`: the opt-in embed is exposed as an `--include` flag accepting the spec's `include` targets (today only `roles`). Whether it is `--include roles`, a repeatable flag, or a comma-separated value is a planning detail; the behavior (opt-in embed, default pure identity) is fixed.
- **Projection field set**: the projection surfaces actor id/name/kind, organization id/name, and membership access level. These are the fields `MeResponse` marks required; the visual layout (key-value lines, table, etc.) is a planning detail consistent with the agent-legible-output principle.
- **Include validation set**: rejecting an unsupported `--include` target follows the dispatch invalid-input convention (002) and validates against the spec's `include` enum. If the spec later adds include targets, the accepted set tracks the spec; this is a planning-adjustable detail, not a behavioral change.
- **Decode target**: the read decodes into the `MeResponse` shape (actor + organization + membership + optional roles). The concrete type name is a planning detail; the surfaced facts are fixed.

---

## Ambiguity Warnings

_None remaining — the three behavioral forks were resolved during the defining conversation: (1) `me` is pure identity by default with an opt-in roles embed (not a second roles surface — that is My Roles, 012); (2) the default output is a reshaped, predictable projection, with structured `--output json` deferred to a future capability; and (3) failures surface Request Execution's generic typed outcome mapped through the exit-code convention (004), with meaningful per-status interpretation left to API Error Extraction (015). The remaining `[ASSUMED]` items are planning-time shape details, not behavioral gaps._
