# Specification: Actor Directory

**Feature**: 048-actor-directory
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Actor Directory is the **discovery entry** of the Actor Reads slice. It offers one read: list the actors — people and agents — in the organization (`glassfrog actors` → `GET /actors` → `listActors`), filterable by kind, by the role an actor fills, and by free-text. It is the find-then-read primitive: before an operator can read an actor's governance footprint or learn whom to contact about a tension, they have to identify *which* actor they mean. Actor Directory answers that — turn a name or a role into the addressable `per_`/`agt_` id the rest of the slice drills into.

It is the first capability in the Actor Reads slice, ahead of Actor Read (049, one actor + footprint) and Actor Assignments (050, the roles an actor fills). It reads the unified `/actors` endpoint, which carries no feature gate — so agents are reachable through it via `--kind agent`, while the dedicated `ai_integration`-gated `/agents` alias stays deferred. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Actor` model grown in Identity Read (011) rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog actors`, the system reads the actors in the organization and produces them as a list result.
- When the user passes a positional argument, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API — the directory takes filters, not positional ids.

### Filters

- When the user supplies `--kind <value>`, the system validates the value against the actor kind vocabulary (`human`, `agent`) before issuing any request; a supported value is sent as the `kind` query parameter, narrowing the list to actors of that kind.
- When the user supplies a `--kind` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how Role Projects, 038, validates `--status`).
- When the user supplies `--role-id <id>`, the system sends it as the `role_id` query parameter, narrowing the list to the actors who fill that role; the id is passed through and the API resolves it.
- When the user supplies `--query <text>` (or `-q <text>`), the system sends it as the `q` query parameter, narrowing the list server-side by free-text search over the actors' names.
- When the user supplies more than one filter, the system sends each as its own query parameter — they combine, narrowing the list further; the API applies them together.
- When the user supplies no filter, the system requests every actor in the organization.

### Output

- When a read succeeds, the system produces the actor data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When no actor matches the supplied filters, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the actors span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the actors gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a malformed `--role-id` rejected as a bad request (typically `400`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** find whom to contact about a tension once I know the role,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list the actors filling a given role by its id.

**In order to** turn a half-remembered name into the id I can drill into,
**as a** practitioner bridging people and the governance record,
**I want to** find an actor by free-text search across the directory.

**In order to** tell automation apart from people before I act,
**as an** AI agent assembling context,
**I want to** narrow the directory to just humans or just agents.

**In order to** trust I'm acting on the whole directory, not a silently truncated slice,
**as an** AI agent with a bounded context,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not read a single actor by id (`GET /actors/{id}`), nor embed an actor's roles or assignments via `?include=`. **Why**: that is Actor Read (049) — the entry to an actor's governance footprint — and Actor Assignments (050). Actor Directory is the flat *discovery* read; folding the per-id read or the footprint embed into it would blur the find-then-read split the slice is built on.
- The system must not expose separate `people` or `agents` commands. **Why**: `/people` and `/agents` are convenience aliases over the unified `/actors` endpoint, and `--kind human|agent` selects either through that one endpoint with no capability lost (`/people` is just `GET /actors?kind=human`). A second command would fork the discovery surface; and the dedicated `/agents` alias is `ai_integration`-gated and deferred, while `/actors?kind=agent` reaches agents without the gate.
- The system must not create, invite, update, or delete actors. **Why**: Actor Directory is a read surface; the endpoint's `POST` (invite) and the actor-admin `PATCH`/`DELETE` operations are out of scope, reflecting the read-only stance of the Actor Reads slice. Actor changes belong to a separate write capability.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /actors`)**: the system reads actors through this endpoint. It accepts `kind` (enum), `role_id`, and `q` (free-text) query filters and is paginated (`{data: [Actor], meta: {pagination}}`). Data flows inbound (actors in, nothing written). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020)**: receives the actor data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Identity Read (011) — sibling**: established the `glassfrog.Actor` model (`id`, `name`, `kind`, timestamps) this command reuses. 011 reads the token-scoped `GET /me` actor; this command reads the org-wide directory. The model is shared; the surfaces are distinct.

---

## Driving Scenarios

### Happy path

**Scenario: List every actor in the organization**
Given a valid stored credential and an organization with several actors
When the user runs `glassfrog actors`
Then the system reads `GET /actors`
And produces the actors as a list result
And exits successfully.

**Scenario: Find the actors filling a role**
Given a valid stored credential and a role id that two actors fill
When the user runs `glassfrog actors --role-id <role-id>`
Then the system sends `role_id=<role-id>` on `GET /actors`
And produces both actors as a list result
And exits successfully.

**Scenario: Narrow the directory to agents**
Given a valid stored credential and an organization with people and agents
When the user runs `glassfrog actors --kind agent`
Then the value is accepted as a supported kind
And the system sends `kind=agent` on `GET /actors`
And produces only the agents as a list result.

### Error scenarios

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog actors`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

**Scenario: Malformed role-id filter is rejected by the API**
Given a valid stored credential and a `--role-id` value the API cannot parse
When the user runs `glassfrog actors --role-id not-a-role`
Then the API answers `400`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

### Edge cases

**Scenario: No actor matches the filters**
Given a valid stored credential and a free-text query no actor's name matches
When the user runs `glassfrog actors --query zzzznomatch`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported kind value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog actors --kind robot`
Then the system rejects it as a usage error, naming the unsupported value and the supported set
And issues no request
And exits with the usage-error code.

**Scenario: Paginated directory with first-page opt-out**
Given an organization whose actors span more than one page
When the user runs `glassfrog actors` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: An unsupported kind costs no request**
Given a `--kind` value outside the actor kind vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Agent discovery does not require the gated alias**
Given a token without the `ai_integration` feature
When the user runs `glassfrog actors --kind agent`
Then the system reaches agents through `GET /actors?kind=agent` (the ungated unified endpoint)
And never routes through the deferred, `ai_integration`-gated `/agents` alias.

**Scenario: Output is structured, not pre-rendered**
Given any successful directory read
When the result reaches Output Format Selection (020)
Then the command supplied structured actor data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **`Actor` model is reused, not redefined**: the `glassfrog.Actor` type grown in Identity Read (011) already carries `id`, `name`, `kind`, `created_at`, and `updated_at` — the same schema this endpoint returns in each `data` element — so no new leaf model is needed. The list returns the shared `{data: [...], meta: {pagination}}` envelope (the one My Roles, 012, established and 016/038 reuse); whether a named list-response type is added is a planning detail. (Reflects the 011 decision and the per-list reuse pattern.)
- **Kind vocabulary tracks the spec enum, validated locally**: `--kind` is validated against the actor kind set (`human`, `agent`) before any request, the same shape Role Projects (038) validates `--status` against. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`); whether validation shares a helper is a planning detail.
- **`--role-id` and `--query` are pass-through filters**: unlike `--kind`, the role-id and free-text `--query` (`q`) carry no fixed local vocabulary, so the command sends them as-is and lets the API match (the API ignores empty/whitespace `q` and returns no rows for a malformed query rather than erroring); an empty or no-match result is a valid empty list, not an error. ([ASSUMED] — the exact flag spellings (`-q` short form per 038, `--role-id` vs `--role`) are pinned at the interface stage; the behavior — three combinable server-side filters, `--kind` locally validated and the other two pass-through — is fixed.)
- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038), not a new per-command flag. (Consistency across every list surface in the CLI.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/026/033/034/038 read pattern, and the boundary question specific to actors was resolved during specification: the directory is a single `glassfrog actors` command with a locally-validated `--kind human|agent` filter over the unified `/actors` endpoint, rather than separate `people`/`agents` commands — `/people` is just `?kind=human` and the `/agents` alias is `ai_integration`-gated and deferred, so one command loses no capability and avoids forking the discovery surface._

---

## Clarifications

### Session 2026-06-11

- **Completeness as a user need**: Added a fourth User Scenario expressing the walk-to-completion / never-silently-truncated benefit in benefit-first voice ("trust I'm acting on the whole directory"), restoring parity with the sibling read 038 (which carried the same completeness user scenario). No behavior changed — the completeness behavior was already fixed in the Behavioral Accord (§ Completeness of the list) and the first-page-opt-out driving scenario; this surfaces it as an explicit first-class user need rather than an accord-only behavior, and gives the scenarios' "Trust the directory is whole" Rule block a Connextra anchor.
