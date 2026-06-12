# Specification: Tension Reads

**Feature**: 043-tension-reads
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Tension Reads is the **read surface for tensions** — the counterpart to Tension Capture (042), which writes a tension as the seed of a proposal. Where 042 records a tension, this capability surfaces existing ones: list the tensions on a role (`glassfrog tension list <role-id>` → `GET /roles/{role_id}/tensions` → `listRoleTensions`) and read a single tension by its `ten_` id (`glassfrog tension get <ten-id>` → `GET /tensions/{id}` → `getTension`). It lets an agent or practitioner find what tensions a role is carrying, optionally narrowed to a status, and fetch the full detail of any one — the read step before working a tension toward a proposal.

It continues the `tension` command grammar 042 opened: 042 deliberately used the `create` sub-verb under `tension` to leave room for these siblings, so the reads are `tension list` and `tension get` rather than a new top-level noun. It is the tension-domain analogue of Role Projects (038) — a list-by-role read plus a read-by-id — and sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the `glassfrog.Tension` model shape 042 established (its `ten_` id, body, label, status, meeting-type, and sensing role / person) rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog tension list <role-id>`, the system reads the tensions on that role and produces them as a list result.
- When the user runs `glassfrog tension get <ten-id>`, the system reads the single tension with that id and produces it as a single-tension result.
- When the user omits the required id on either command, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Filters (list)

- When the user supplies `--status <value>` on `tension list <role-id>`, the system validates the value against the tension status vocabulary (`unprocessed`, `processed`, `archived`) before issuing any request; a supported value is sent as the `status` query parameter, narrowing the list to tensions in that status.
- When the user supplies a `--status` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how My Projects (014) and Role Projects (038) validate `--status`).
- When the user supplies no filter, the system requests every tension on the role.
- When the user supplies `--status` on the single read (`tension get <ten-id>`), the system rejects the invocation as a usage error — the filter applies only to the per-role list.

### Output

- When a read succeeds, the system produces the tension data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the role carries no tensions, or none match the supplied status, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When a role's tensions span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the tensions gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a read whose role id or tension id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see what tensions a role is carrying before I act inside it,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list the tensions on any role by its id with one command.

**In order to** read the full detail of a tension I captured or found referenced elsewhere,
**as a** practitioner working a governance issue,
**I want to** fetch a single tension by its `ten_` id.

**In order to** focus on just the unworked tensions in a role with a long history,
**as an** AI agent assembling context,
**I want to** narrow the role's tension list by status.

**In order to** trust I am seeing every tension on a role,
**as a** practitioner in a busy circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not create, update, or discard tensions. **Why**: Tension Reads owns the read pair alone, matching the project's one-capability-per-spec pattern. Capture is Tension Capture (042); editing body/label/status/meeting-type is Tension Update (044); soft-delete is Tension Discard (045). A read surface that mutated would fork those write contracts.
- The system must not list tensions across a role's subroles (`listSubrolesTensions`). **Why**: that one-level roll-up is its own capability — Subroles Tension Roll-up (046) — with its own endpoint (`GET /roles/{role_id}/subroles/tensions`) and its own "anchor must have subroles" behavior. Tension Reads is scoped to a single role's own tensions and the by-id read.
- The system must not expose `tension list`/`tension get` as a plural/singular noun pair (`tensions <role-id>` / `tension <ten-id>`) the way Role Projects (038) does. **Why**: 042 already established `tension` as a command with the `create` sub-verb, and cobra cannot tell a bare `tension <ten-id>` from a subcommand name. Continuing the verb grammar (`tension list`, `tension get`, alongside the existing `tension create`) keeps every invocation unambiguous and consistent with the shipped write verb.
- The system must not set, override, or recompute a tension's status, nor treat `--status` as anything but a list filter. **Why**: status is server-owned (auto-computed except explicit `archived` via PATCH, per 042); on a read it is only a narrowing query parameter, never a write.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not interpret, summarize, or advise on the tensions it reads. **Why**: the CLI is a faithful API surface, not a Holacracy facilitator (VISION Exclusion 1); it produces the tension data and lets the operator reason about it.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{role_id}/tensions`, `GET /tensions/{id}`)**: the system reads tensions through these endpoints. The list endpoint accepts an optional `status` (enum) query filter and is paginated; the single read returns full tension detail by `ten_` id. Data flows inbound (tensions in, nothing written). A `404` means the role id or tension id does not exist. When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and (for the list) walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the tension data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **Tension Capture (042) — sibling**: established the `tension` command, its `create` sub-verb, and the tension model shape (`ten_` id, body, label, status, meeting-type, sensing role / person) that these reads surface and reuse.

---

## Driving Scenarios

### Happy path

**Scenario: List the tensions on a role**
Given a valid stored credential and a role id that carries two tensions
When the user runs `glassfrog tension list <role-id>`
Then the system reads `GET /roles/{role_id}/tensions`
And produces both tensions as a list result
And exits successfully.

**Scenario: Read a single tension with full detail**
Given a valid stored credential and an existing `ten_` id
When the user runs `glassfrog tension get <ten-id>`
Then the system reads `GET /tensions/{id}`
And produces the single tension including its full detail
And exits successfully.

**Scenario: Narrow a role's tensions by status**
Given a role that carries tensions in several statuses
When the user runs `glassfrog tension list <role-id> --status unprocessed`
Then the value is accepted as a supported status
And the system sends `status=unprocessed` on `GET /roles/{role_id}/tensions`
And produces only the unprocessed tensions as a list result.

### Error scenarios

**Scenario: Tension id does not exist**
Given a valid stored credential and a `ten_` id that no tension has
When the user runs `glassfrog tension get <ten-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog tension list <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Role carries no tensions**
Given a valid stored credential and a role that carries no tensions
When the user runs `glassfrog tension list <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported status value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog tension list <role-id> --status open`
Then the system rejects it as a usage error, naming the unsupported value and the supported set (`unprocessed`, `processed`, `archived`)
And issues no request
And exits with the usage-error code.

**Scenario: Status filter on the single read is rejected**
Given a valid stored credential
When the user runs `glassfrog tension get <ten-id> --status unprocessed`
Then the system rejects the invocation as a usage error before calling the API
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given a role whose tensions span more than one page
When the user runs `glassfrog tension list <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The read surface never reaches into the write verbs**
Given the implemented commands
When a reviewer exercises `glassfrog tension` for any create-, update-, or discard-style behavior under `list` or `get`
Then no such behavior is present — `list` and `get` only read
And the help text for the read verbs does not advertise mutation.

**Scenario: An unsupported status costs no request**
Given a `--status` value outside the tension status vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful tension read
When the result reaches Output Format Selection (020)
Then the command supplied structured tension data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **`Tension` model is reused, not redefined**: the tension shape grown in 042 (carrying `ten_` id, body, label, status, meeting-type, and sensing role / person) is the same schema both endpoints return — the list returns many, `getTension` returns full detail of one — so no new leaf model is needed; the renderers (019) decide how much to show per format. (Mirrors the per-role reuse pattern of 033/034/038.)
- **Status vocabulary tracks the spec enum**: `--status` is validated against the tension status set (`unprocessed`, `processed`, `archived`) before any request, the values the `listRoleTensions` endpoint accepts. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`). (Reflects the spec's documented `?status=` enum.)
- **[ASSUMED] Tension-id and role-id formats are not validated client-side**: each read requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing the `^ten_…` / `^role_…` patterns locally. (Mirrors how Role Projects (038) and Tension Capture (042) leave id-shape validation to the server.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/026/033/034/038 read pattern (list-by-role + read-by-id, status-validated list filter, full-walk pagination with first-page opt-out), and the one tension-specific shape question was settled by precedent: 042 already chose the `tension <verb>` grammar, so the reads are `tension list` / `tension get` (not a plural/singular noun pair), and the status filter applies only to the list._
