# Specification: Role Fillers

**Feature**: 047-role-fillers
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Role Fillers is the **"who fills this role?" read** in the Actor Reads slice. It offers one read: list the assignments of a role (`glassfrog fillers <role-id>` → `GET /roles/{role_id}/assignments` → `listRoleAssignments`). An assignment maps an actor — a person or an AI agent — to the role they fill, so this capability turns a role id into the addressable answer a practitioner most often needs about a role they did not author: *whom do I contact about this, and in what capacity do they fill it?* It surfaces each filler's identity (name, kind, actor id) alongside the assignment's own governance context — its `focus` and `elected_until`.

It is the role-addressable counterpart to Actor Assignments (050), which reads the same `Assignment` resource from the other end (`GET /actors/{actor_id}/assignments` — the roles one actor fills); Role Fillers reads it by role. It is distinct from Actor Directory (048): 048's `actors --role-id` filter returns bare **actor** records filtered to a role (discovery — turn a role into actor ids), while Role Fillers returns the **assignment** — the filling relationship itself, carrying focus and election. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Assignment` model — grown by Role Reads (025) for `?include=assignments` and reserved there for exactly this standalone read — rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog fillers <role-id>`, the system reads the assignments of that role and produces them as a list result.
- When the user omits the required role-id, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Output

- When a read succeeds, the system produces the assignment data as its result — each row carrying the filling actor (id, name, kind) and the assignment's `focus` and `elected_until` — and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the role has no fillers, the system produces an empty list result and exits successfully — an unfilled role is a valid answer, not an error.
- When a row's `focus` or `elected_until` is absent (the assignment has no focus, or is not an elected seat), the system renders an explicit-absence marker for that field rather than omitting it silently or inventing a value.

### Completeness of the list

- When a role's fillers span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the assignments gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a read whose role id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** know whom to contact about a role I do not fill,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list the actors who fill any role by its id with one command.

**In order to** understand not just who fills a role but in what capacity,
**as a** practitioner reviewing the governance around a role,
**I want** each filler shown with its focus and, for elected seats, its election expiry.

**In order to** trust I am seeing every filler of a role,
**as a** practitioner in a busy circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not create, update, or remove an assignment. **Why**: assigning or un-assigning an actor is **actor administration**, which PROJECT.md places out of scope; the `/roles/{role_id}/assignments` `POST` and the `/assignments/{id}` `PATCH`/`DELETE` operations are deliberately not surfaced. Role Fillers is a read.
- The system must not offer a single-assignment read command. **Why**: the API exposes no `GET /assignments/{id}` — that path carries only the administrative `PATCH`/`DELETE` — so there is no standalone read to surface, and a role's fillers are already complete in the list. This is one list read, unlike the list+single pair of Role Projects (038).
- The system must not expose an `--include` flag. **Why**: the endpoint's default include (`actor`) already embeds the filler's name and kind, which is the whole point of the read; the only other include value (`role`) is redundant when the caller already supplied the role id. A flag here would add a knob with no useful setting.
- The system must not invent client-side filters (by actor kind, name, or focus). **Why**: `listRoleAssignments` accepts no query filter beyond `include` and pagination, so any narrowing would be the CLI second-guessing the API — and the Actor Directory (048) `--role-id` + `--kind`/`--query` surface already covers actor-shaped filtering. Role Fillers returns the role's assignments whole.
- The system must not duplicate the Actor Directory (048) discovery read. **Why**: 048 returns **actor** records (`GET /actors?role_id=`) for finding ids; Role Fillers returns the **assignment** (`GET /roles/{role_id}/assignments`) with its focus and election context. Conflating them would fork the two return shapes and blur discovery from governance context.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{role_id}/assignments`)**: the system reads a role's assignments through this endpoint. It is paginated and embeds the full actor object by default (`include=actor`); each assignment also carries `focus` and `elected_until`. Data flows inbound (assignments in, nothing written). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020)**: receives the assignment data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Role Reads (025) — sibling**: established the `glassfrog.Assignment` model (via `?include=assignments`) and reserved it for this standalone read. The model is shared; 025 surfaces only the actor reference in its embedded view, while Role Fillers additionally projects `focus` and `elected_until`.
- **Actor Assignments (050) / Actor Directory (048) — slice siblings**: 050 reads the same `Assignment` resource by actor (`GET /actors/{actor_id}/assignments`); 048 lists actors filtered by role. Role Fillers owns the role-scoped assignment read; the surfaces are distinct.

---

## Driving Scenarios

### Happy path

**Scenario: List the fillers of a role**
Given a valid stored credential and a role id filled by two actors
When the user runs `glassfrog fillers <role-id>`
Then the system reads `GET /roles/{role_id}/assignments`
And produces both assignments as a list result, each carrying its filling actor's name and kind
And exits successfully.

**Scenario: A filler row shows its focus and election expiry**
Given a role filled by an actor whose assignment has a focus and an election date
When the user runs `glassfrog fillers <role-id>`
Then the result for that filler carries both its `focus` and its `elected_until`
And exits successfully.

**Scenario: Fillers span both a person and an agent**
Given a role filled by one `per_` actor and one `agt_` actor
When the user runs `glassfrog fillers <role-id>`
Then the system produces both assignments
And each row's actor distinguishes its kind (`human` vs `agent`)
And exits successfully.

### Error scenarios

**Scenario: Role id does not exist**
Given a valid stored credential and a `role_` id that no role has
When the user runs `glassfrog fillers <role-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog fillers <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Role has no fillers**
Given a valid stored credential and a role that no actor fills
When the user runs `glassfrog fillers <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Missing role-id is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog fillers` with no positional id
Then the system rejects it as a usage error
And issues no request
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given a role whose fillers span more than one page
When the user runs `glassfrog fillers <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The filler's name appears without an include flag**
Given a successful read of a role with fillers
When the result is rendered
Then each filler's name and kind are present, sourced from the endpoint's default `actor` include
And the command exposed no `--include` flag to obtain them.

**Scenario: Focus and election are projected, not dropped**
Given an assignment that carries a `focus` and an `elected_until`
When the result is rendered in a human format
Then both fields are visible (and an absent one shows an explicit-absence marker), confirming the read surfaces the assignment's governance context, not just the actor reference.

**Scenario: A missing token costs no request**
Given no usable credential
When the command runs
Then the system surfaces the not-authenticated refusal before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful fillers read
When the result reaches Output Format Selection (020)
Then the command supplied structured assignment data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **`Assignment` model is reused, not redefined**: the `glassfrog.Assignment` type grown in Role Reads (025) already carries `id`, `actor_id`, `role_id`, `focus`, `elected_until`, and the embedded `actor` (`id`/`name`/`kind`) — the same shape this endpoint returns. 025's embedded view surfaces only the actor reference; this read additionally projects `focus` and `elected_until`. No new leaf model is needed. (Reflects the 025 comment reserving the type for "the future standalone assignment reads" and the per-role reuse pattern of 033/034/038.)
- **Default `actor` include is relied upon**: the endpoint embeds the full actor by default when accessed via `/roles/{role_id}/assignments`, so the command gets each filler's name and kind without a flag. Whether the command sends `include=actor` explicitly or trusts the server default is a planning detail. (Reflects the endpoint's documented default.)
- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038 / 048), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **Command is named `fillers`, the resource stays `Assignment`**: the user-facing command speaks the practitioner's question ("who fills this role?"); the model, endpoint, and API tag remain "assignment". ([ASSUMED] — the exact command spelling is pinned at the interface stage; the behavior — one role-scoped, read-only list of a role's fillers with their focus and election — is fixed.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/033/034/038/048 read pattern, and the two boundary questions specific to the assignments endpoint were resolved during specification: (1) the command is named `fillers <role-id>` (practitioner-facing) while the underlying resource stays the `Assignment` model; and (2) the projection surfaces the assignment's `focus` and `elected_until` alongside the filling actor, not the actor reference alone. There is no single-assignment read (the API exposes none) and no `--include` or client-side filters (the endpoint offers neither beyond the default actor include and pagination)._
