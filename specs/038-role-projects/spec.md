# Specification: Role Projects

**Feature**: 038-role-projects
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Role Projects is the **project read surface** in the Governance Reads slice. It offers two reads: list the projects owned by a role (`glassfrog projects <role-id>` → `GET /roles/{role_id}/projects` → `listRoleProjects`) and read a single project by its id (`glassfrog project <proj-id>` → `GET /projects/{id}` → `getProject`). A project is a role's tracked outcome — the most operational of the governance elements — so this capability gives an agent or practitioner an addressable way to find the projects on any role and to fetch the full detail of any one of them.

It is the operational counterpart to Role Domains (033) and Role Policies (034), and it completes the per-role read trio they belong to. It is distinct from My Projects (014): 014 reads the **token-scoped** "what's mine" surface (`GET /me/projects`); Role Projects reads the **role-addressable** surface — any role's projects by id, and any project by id. It sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the existing `glassfrog.Project` model grown in 014 rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog projects <role-id>`, the system reads the projects owned by that role and produces them as a list result.
- When the user runs `glassfrog project <proj-id>`, the system reads the single project with that id and produces it as a single-project result.
- When the user omits the required id on either command, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Filters (list)

- When the user supplies `--query <text>` (or `-q <text>`) on `projects <role-id>`, the system sends it as the `q` query parameter, narrowing the list server-side by free-text search; the role-id path scope still applies.
- When the user supplies `--status <value>` on `projects <role-id>`, the system validates the value against the project status vocabulary (`archived`, `cancelled`, `completed`, `current`, `scheduled`, `someday`, `waiting`) before issuing any request; a supported value is sent as the `status` query parameter, narrowing the list to projects in that status.
- When the user supplies a `--status` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how My Projects, 014, validates `--status`).
- When the user supplies `--tag <name>` on `projects <role-id>`, the system sends it as the `tag` query parameter, narrowing the list to projects carrying that tag.
- When the user supplies more than one filter, the system sends each as its own query parameter — they combine, narrowing the list further; the API applies them together.
- When the user supplies no filter, the system requests every project owned by the role.
- When the user supplies `--query`, `--status`, or `--tag` on the single read (`project <proj-id>`), the system rejects the invocation as a usage error — the filters apply only to the per-role list.

### Output

- When a read succeeds, the system produces the project data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the role owns no projects, or no project matches the supplied filters, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When a role's projects span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the projects gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a read whose role id or project id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see what work a role is responsible for before I act inside it,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list the projects owned by any role by its id with one command.

**In order to** read the full detail of a specific project I found referenced elsewhere,
**as a** practitioner reviewing the work around a role,
**I want to** fetch a single project by its id.

**In order to** focus on just the live work in a role with many projects,
**as an** AI agent assembling context,
**I want to** narrow the role's project list by status, free-text, or tag.

**In order to** trust I am seeing every project on a role,
**as a** practitioner in a busy circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not split the two reads into one command with an optional positional id the way `roles` does. **Why**: the two reads take *different* id kinds (a `role_` id selects a per-role list; a `proj_` id selects one project) with different meanings, and cobra cannot tell a role id from a subcommand name (the constraint that already kept the per-role reads off `roles <id> …`). Two distinct commands — plural `projects <role-id>` for the role-scoped list, singular `project <proj-id>` for the standalone read — keep each invocation unambiguous.
- The system must not read the token-scoped `GET /me/projects` surface, nor route the per-role list through it. **Why**: that is My Projects (014) — the *self-service* "what's mine" read. Role Projects owns the *addressable* role-scoped reads; conflating them would fork the two surfaces.
- The system must not model or embed sub-projects or actions, nor expose an `--include=sub_projects,actions` option, even though both endpoints offer it. **Why**: Role Projects is a flat project read, consistent with its governance-read siblings Role Domains (033) and Role Policies (034); the `Project` model's `has_sub_projects` / `has_actions` flags already signal that children exist without fetching them, and standalone sub-project / action reads are their own future capability.
- The system must not provide standalone reads of other related resources (domains, policies, notes, skills). **Why**: each is its own capability — Role Domains (033), Role Policies (034), Organization Tree subroles (026). Role Projects owns the project surface alone.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not create, update, or otherwise mutate any project. **Why**: Role Projects is a read surface; the endpoints' `POST`/`PATCH` operations are out of scope, reflecting the read-only stance of the governance reads. Project changes belong to a separate write capability.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{role_id}/projects`, `GET /projects/{id}`)**: the system reads projects through these endpoints. The list endpoint accepts `q` (free-text), `status` (enum), and `tag` query filters and is paginated; the single read returns full project detail. Data flows inbound (projects in, nothing written). When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and (for the list) walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020)**: receives the project data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **My Projects (014) — sibling**: established the `glassfrog.Project` model and the `--status` vocabulary this command reuses. 014 reads `GET /me/projects` (token-scoped); this command reads the role-addressable endpoints. The model and status enum are shared; the surfaces are distinct.

---

## Driving Scenarios

### Happy path

**Scenario: List the projects on a role**
Given a valid stored credential and a role id that owns two projects
When the user runs `glassfrog projects <role-id>`
Then the system reads `GET /roles/{role_id}/projects`
And produces both projects as a list result
And exits successfully.

**Scenario: Read a single project with full detail**
Given a valid stored credential and an existing project id
When the user runs `glassfrog project <proj-id>`
Then the system reads `GET /projects/{id}`
And produces the single project including its full detail
And exits successfully.

**Scenario: Narrow a role's projects by status**
Given a role that owns projects in several statuses
When the user runs `glassfrog projects <role-id> --status current`
Then the value is accepted as a supported status
And the system sends `status=current` on `GET /roles/{role_id}/projects`
And produces only the current projects as a list result.

### Error scenarios

**Scenario: Project id does not exist**
Given a valid stored credential and a `proj_` id that no project has
When the user runs `glassfrog project <proj-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog projects <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Role owns no projects**
Given a valid stored credential and a role that owns no projects
When the user runs `glassfrog projects <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported status value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog projects <role-id> --status active`
Then the system rejects it as a usage error, naming the unsupported value and the supported set
And issues no request
And exits with the usage-error code.

**Scenario: Filter flag on the single read is rejected**
Given a valid stored credential
When the user runs `glassfrog project <proj-id> --status current`
Then the system rejects the invocation as a usage error before calling the API
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given a role whose projects span more than one page
When the user runs `glassfrog projects <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The two commands never collide on id kind**
Given the project reads are exposed as `projects <role-id>` and `project <proj-id>`
When a `role_` id is passed to `project` (or a `proj_` id to `projects`)
Then the system either reports the API's `404`/`400` for the wrong id kind, or rejects it — never silently reads the wrong resource — so the plural/singular split stays unambiguous in practice.

**Scenario: An unsupported status costs no request**
Given a `--status` value outside the project status vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful project read
When the result reaches Output Format Selection (020)
Then the command supplied structured project data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **`Project` model is reused, not redefined**: the `glassfrog.Project` type grown in 014 already carries `id`, `status`, `description`, `role_id`, `tags`, `has_sub_projects`, `has_actions`, and the remaining detail fields — the same schema both endpoints return — so no new leaf model is needed. The single read (`getProject`) returns "full detail" of the same shape; the renderers (019) decide how much to show per format. (Reflects the 014 decision and the per-role reuse pattern of 033/034.)
- **Status vocabulary tracks the spec enum, shared with 014**: `--status` is validated against the project status set (`archived`, `cancelled`, `completed`, `current`, `scheduled`, `someday`, `waiting`) before any request, the same enum My Projects (014) validates against. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`); whether validation shares a helper with 014 is a planning detail.
- **`--query` and `--tag` are pass-through filters**: unlike `--status`, free-text `--query` (`q`) and `--tag` carry no fixed vocabulary, so the command sends them as-is and lets the API match; an empty or no-match result is a valid empty list, not an error. ([ASSUMED] — the exact flag spellings (`-q` short form per 034) are pinned at the interface stage; the behavior — three combinable server-side filters on the list, none on the single read — is fixed.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/026/033/034 read pattern, and the two boundary questions specific to the richer projects endpoint were resolved during specification: (1) the list exposes all three server-side filters the endpoint offers (`--query`/`-q`, `--status` with local enum validation per 014, and `--tag`), combinable, and rejected on the single read; and (2) Role Projects is a flat read with no `--include=sub_projects,actions` option — the `has_sub_projects` / `has_actions` presence flags signal children, consistent with its flat governance-read siblings 033/034._
