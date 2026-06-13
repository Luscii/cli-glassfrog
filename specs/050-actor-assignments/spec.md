# Specification: Actor Assignments

**Feature**: 050-actor-assignments
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Actor Assignments is the **"what does this actor fill?" read** in the Actor Reads slice. It offers one read: list the assignments of an actor (`glassfrog assignments <actor-id>` → `GET /actors/{actor_id}/assignments` → `listActorAssignments`). An assignment maps an actor — a person or an AI agent — to a role they fill, so this capability turns an actor id into the addressable answer a practitioner most often needs about a person or agent: *which roles does this actor fill, and in what capacity?* It surfaces each filled role's identity (name, the `role_` id, purpose, parent role) alongside the assignment's own governance context — its `focus` and `elected_until`.

It is the actor-addressable counterpart to Role Fillers (047), which reads the same `Assignment` resource from the other end (`GET /roles/{role_id}/assignments` — the actors who fill a role); Actor Assignments reads it by actor. It is distinct from Actor Directory (048): 048's `actors --role-id` filter returns bare **actor** records (discovery — turn a role into actor ids), while Actor Assignments returns the **assignment** — the filling relationship itself, carrying the role reference, focus, and election. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Assignment` model — grown by Role Reads (025) and projected by Role Fillers (047) — rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog assignments <actor-id>`, the system reads the assignments of that actor and produces them as a list result.
- When the user omits the required actor-id, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Output

- When a read succeeds, the system produces the assignment data as its result — each row carrying the filled role (the `role_` id, name, and the role context the default include provides: purpose and parent role) and the assignment's `focus` and `elected_until` — and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the actor fills no roles, the system produces an empty list result and exits successfully — an actor with no assignments is a valid answer, not an error.
- When a row's `focus` or `elected_until` is absent (the assignment has no focus, or is not an elected seat), the system renders an explicit-absence marker for that field rather than omitting it silently or inventing a value.

### Completeness of the list

- When an actor's assignments span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the assignments gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a read whose actor id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** understand a person's or agent's whole governance footprint at a glance,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list every role an actor fills by its id with one command.

**In order to** understand not just which roles an actor fills but in what capacity,
**as a** practitioner reviewing who is doing what,
**I want** each assignment shown with its focus and, for elected seats, its election expiry.

**In order to** trust I am seeing every assignment an actor holds,
**as a** practitioner reviewing a busy organization,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not create, update, or remove an assignment. **Why**: assigning or un-assigning an actor is **actor administration**, which PROJECT.md places out of scope; the `/roles/{role_id}/assignments` `POST` and the `/assignments/{id}` `PATCH`/`DELETE` operations are deliberately not surfaced. Actor Assignments is a read.
- The system must not offer a single-assignment read command. **Why**: the API exposes no `GET /assignments/{id}` — that path carries only the administrative `PATCH`/`DELETE` — so there is no standalone read to surface, and an actor's assignments are already complete in the list.
- The system must not expose an `--include` flag. **Why**: the endpoint's default include (`role`) already embeds the filled role's name and context, which is the whole point of the read; the only other include value (`actor`) is redundant when the caller already supplied the actor id. A flag here would add a knob with no useful setting.
- The system must not invent client-side filters (by role type, name, or focus). **Why**: `listActorAssignments` accepts no query filter beyond `include` and pagination, so any narrowing would be the CLI second-guessing the API. Actor Assignments returns the actor's assignments whole.
- The system must not duplicate the Role Fillers (047) read. **Why**: 047 reads the same `Assignment` resource from the role end (`GET /roles/{role_id}/assignments`, default-embedding the actor); Actor Assignments reads it from the actor end (`GET /actors/{actor_id}/assignments`, default-embedding the role). Conflating the two would fork the two read directions of one resource.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /actors/{actor_id}/assignments`)**: the system reads an actor's assignments through this endpoint. It is paginated and embeds the full role object by default (`include=role`); each assignment also carries `focus` and `elected_until`. Data flows inbound (assignments in, nothing written). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020)**: receives the assignment data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Role Reads (025) — sibling**: established the `glassfrog.Assignment` model. The model is shared; this read projects the embedded `role` reference alongside `focus` and `elected_until`.
- **Role Fillers (047) / Actor Directory (048) — slice siblings**: 047 reads the same `Assignment` resource by role (`GET /roles/{role_id}/assignments`); 048 lists bare actor records. Actor Assignments owns the actor-scoped assignment read; the surfaces are distinct.

---

## Driving Scenarios

### Happy path

**Scenario: List the roles an actor fills**
Given a valid stored credential and an actor id who fills two roles
When the user runs `glassfrog assignments <actor-id>`
Then the system reads `GET /actors/{actor_id}/assignments`
And produces both assignments as a list result, each carrying its filled role's name and id
And exits successfully.

**Scenario: An assignment row shows its focus and election expiry**
Given an actor with an assignment that has a focus and an election date
When the user runs `glassfrog assignments <actor-id>`
Then the result for that assignment carries both its `focus` and its `elected_until`
And exits successfully.

**Scenario: Assignments read for either a person or an agent**
Given a `per_` actor and an `agt_` actor, each filling at least one role
When the user runs `glassfrog assignments <actor-id>` for each
Then the system produces each actor's assignments from the same endpoint
And exits successfully.

### Error scenarios

**Scenario: Actor id does not exist**
Given a valid stored credential and an actor id that no actor has
When the user runs `glassfrog assignments <actor-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog assignments <actor-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Actor fills no roles**
Given a valid stored credential and an actor that fills no role
When the user runs `glassfrog assignments <actor-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Missing actor-id is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog assignments` with no positional id
Then the system rejects it as a usage error
And issues no request
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given an actor whose assignments span more than one page
When the user runs `glassfrog assignments <actor-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The filled role's name appears without an include flag**
Given a successful read of an actor with assignments
When the result is rendered
Then each assignment's filled role name and id are present, sourced from the endpoint's default `role` include
And the command exposed no `--include` flag to obtain them.

**Scenario: Focus and election are projected, not dropped**
Given an assignment that carries a `focus` and an `elected_until`
When the result is rendered in a human format
Then both fields are visible (and an absent one shows an explicit-absence marker), confirming the read surfaces the assignment's governance context, not just the role reference.

**Scenario: A missing token costs no request**
Given no usable credential
When the command runs
Then the system surfaces the not-authenticated refusal before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful assignments read
When the result reaches Output Format Selection (020)
Then the command supplied structured assignment data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **`Assignment` model is reused, not redefined**: the `glassfrog.Assignment` type grown in Role Reads (025) and projected by Role Fillers (047) already carries `id`, `actor_id`, `role_id`, `focus`, `elected_until`, and the embedded related objects. Role Fillers (047) projects the embedded `actor`; this read projects the embedded `role` (`id`, `type`, `name`, and the `purpose`/`parent_role_id` context the include carries). Whether the `role` projection adds fields to the shared type is a planning detail; no new leaf model is needed. (Reflects the 025 reservation and the 047 projection pattern.)
- **Default `role` include is relied upon**: the endpoint embeds the full role by default when accessed via `/actors/{actor_id}/assignments`, so the command gets each filled role's name and context without a flag. Whether the command sends `include=role` explicitly or trusts the server default is a planning detail. (Reflects the endpoint's documented default — the mirror of 047's default `actor` include.)
- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038 / 047 / 048), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **Command is named `assignments`, the resource stays `Assignment`**: the user-facing command speaks the practitioner's question ("what does this actor fill?"); the model, endpoint, and API tag remain "assignment". ([ASSUMED] — the exact command spelling is pinned at the interface stage, paired with the sibling `fillers <role-id>` (047); the behavior — one actor-scoped, read-only list of the roles an actor fills with their focus and election — is fixed.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/033/034/038/047/048 read pattern and is the direct mirror of Role Fillers (047): it reads the one `Assignment` resource from the actor end (`GET /actors/{actor_id}/assignments`) where 047 reads it from the role end. The two boundary questions specific to the assignments endpoint were resolved during specification: (1) the command is named `assignments <actor-id>` (practitioner-facing, paired with 047's `fillers <role-id>`) while the underlying resource stays the `Assignment` model; and (2) the projection surfaces the assignment's filled `role` (the default include) alongside its `focus` and `elected_until`. There is no single-assignment read (the API exposes none) and no `--include` or client-side filters (the endpoint offers neither beyond the default role include and pagination)._
