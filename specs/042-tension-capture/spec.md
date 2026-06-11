# Specification: Tension Capture

**Feature**: 042-tension-capture
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Tension Capture is the **first write into the governance system** — the entry point to the proposal write path. Where the read specs surface governance state, this command records a new **tension** sensed by one of the practitioner's roles. A tension is the required seed of a proposal (its `tension_id`), so capturing one is where the write half of the CLI's purpose begins (VISION success #2). The command is `glassfrog tension create <role-id>`: the positional `<role-id>` is the **sensing role**, the token's person is recorded by the server as the **sensing person** (`sensed_by`), and the tension's body and optional label / meeting-type come from flags. It maps to `POST /roles/{role_id}/tensions` (`createTension`) and produces the created tension — including its `ten_` id — which an agent can later feed to a proposal.

It sits on the proven chain rather than rebuilding it: it hands the request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render the created tension, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It is the write counterpart to the read surface — deliberately scoped to *capture alone*, distinct from the future tension reads/edits (`listRoleTensions`, `getTension`, `updateTension`, `deleteTension`) and from the **Proposal Write-Flow** it feeds. The `create` sub-verb under `tension` leaves room for those siblings without claiming them here.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog tension create <role-id> --body "<text>"`, the system captures a new tension sensed by that role and produces the created tension as its result.
- When the user omits the required `<role-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- When the user omits `--body`, or supplies a `--body` that is empty or only whitespace, the system rejects the invocation as a usage error — naming `--body` as required — and sends no request. A bodyless tension is meaningless.

### Input

- When the user supplies `--label "<text>"`, the system sends it as the tension's label (a short title) alongside the body.
- When the user supplies `--meeting-type <value>`, the system validates the value against the meeting-type vocabulary (`tactical`, `governance`) before issuing any request; a supported value is sent as the tension's `meeting_type` — a routing-intent hint, not a meeting binding.
- When the user supplies a `--meeting-type` outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how `--status` is validated in My Projects (014) and Role Projects (038)).
- When the user supplies neither `--label` nor `--meeting-type`, the system captures the tension with body alone and sends no value for those fields; the server auto-computes the tension's status.

### Output

- When the capture succeeds, the system produces the created tension — including its `ten_` id, body, status, and sensing role / person — as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the capture succeeds, the system exits successfully (the created tension, with its id, is the load-bearing output a later proposal references).

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including an unknown sensing role (typically `404`) and a rejected body or field (typically `422`) — the system reports that the capture failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** begin a governance change from a gap I've noticed,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** record a tension against the sensing role with one command.

**In order to** hand the new tension to a proposal later,
**as an** AI agent assembling the write flow,
**I want** the capture to return the tension's `ten_` id.

**In order to** give the tension a short handle and signal where it should be worked,
**as a** practitioner sensing an issue,
**I want to** attach a label and a meeting-type at capture.

**In order to** avoid recording an empty, meaningless tension,
**as an** AI agent driving the CLI,
**I want** a missing or blank body rejected before any request is made.

---

## Non-Behaviors

- The system must not list, get, update, or delete tensions. **Why**: the tension API offers `listRoleTensions`, `getTension`, `updateTension`, and `deleteTension`, but Tension Capture owns the create alone — matching the project's one-capability-per-spec pattern (each read is its own spec). Tension reads and edits are their own future capabilities.
- The system must not list tensions across a role's subroles (`listSubrolesTensions`). **Why**: that is a read, and belongs with the future tension reads, not the capture write.
- The system must not create a proposal, nor attach the captured tension to one. **Why**: that is Proposal Write-Flow — the multi-step, Premium-gated governance write path. Capture only produces the seed (`tension_id`) that flow later consumes; coupling them would fork the staged write path.
- The system must not set or override the tension's status on capture. **Why**: the API auto-computes status (`unprocessed`) from associated work, and `archived` is set only via `PATCH` — which is the future update capability, not capture. Exposing a `--status` here would let the CLI claim a state the server owns.
- The system must not bind the tension to a meeting or manage a meeting agenda. **Why**: tensions captured this way are explicitly *not* bound to a meeting; `--meeting-type` is a stored routing hint, not a meeting attachment, and meeting/agenda management is outside this surface entirely.
- The system must not infer, set, or override the sensing person (`sensed_by`). **Why**: the server derives it from the token's identity; the CLI operates within that single identity (PROJECT constraint) and must not claim to sense a tension on another person's behalf.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not send an `If-Match` precondition or otherwise handle optimistic concurrency. **Why**: a create has no prior `ETag` to guard against; concurrency on *edits* is Clobbered Changes, a separate later capability that only becomes relevant once edits exist.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`POST /roles/{role_id}/tensions`, `createTension`)**: the system writes a new tension. The path `role_id` is the sensing role; the request body carries `tension.{body, label, meeting_type}` (only the fields the user supplied). Data flows outbound (the tension fields in) and inbound (the created tension, with its `ten_` id and server-computed status, back). A `404` means the sensing role is unknown; a `422` means the server rejected the body or a field. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or capture failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the `POST`, hands it to these seams to authenticate and execute it, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: the command hands the created tension to the output seam, which renders it in the effective format.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.

---

## Driving Scenarios

### Happy path

**Scenario: Capture a tension with body only**
Given a stored credential for an org and person
When the user runs `glassfrog tension create role_<id> --body "We ship work faster than we update the roadmap."`
Then the system POSTs the tension to the sensing role
And produces the created tension — with its `ten_` id and an auto-computed status — as the result
And exits successfully.

**Scenario: Capture a tension with body, label, and meeting-type**
Given a stored credential
When the user runs `glassfrog tension create role_<id> --body "<text>" --label "Roadmap drift" --meeting-type governance`
Then the system sends `body`, `label`, and `meeting_type=governance` in the tension
And produces the created tension as the result
And exits successfully.

**Scenario: The created tension's id is visible in JSON output**
Given a stored credential and `--output json`
When the user captures a tension successfully
Then the rendered result contains the tension's `ten_` id
So a later proposal can reference it as `tension_id`.

### Error scenarios

**Scenario: Missing body is rejected before any request**
Given a stored credential
When the user runs `glassfrog tension create role_<id>` with no `--body`
Then the system rejects the invocation as a usage error naming `--body` as required
And makes no API call
And exits non-zero.

**Scenario: Unsupported meeting-type is rejected before any request**
Given a stored credential
When the user runs `glassfrog tension create role_<id> --body "<text>" --meeting-type weekly`
Then the system rejects the invocation as a usage error, naming `weekly` and the supported set (`tactical`, `governance`)
And makes no API call
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and no token in the environment
When the user runs `glassfrog tension create role_<id> --body "<text>"`
Then the system surfaces the not-authenticated refusal pointing at how to store a credential
And makes no API call
And exits non-zero.

### Edge cases

**Scenario: Whitespace-only body is treated as empty**
Given a stored credential
When the user runs `glassfrog tension create role_<id> --body "   "`
Then the system rejects the invocation as a usage error naming `--body` as required
And makes no API call
And exits non-zero.

**Scenario: Unknown sensing role**
Given a stored credential
When the user captures a tension against a `role_<id>` that does not exist
Then the API answers `404`
And the system reports the capture failed, naming the HTTP status, and exits non-zero.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Capture does not reach into the read surface**
Given the implemented command
When a reviewer exercises `glassfrog tension` for any list-, get-, update-, or delete-style behavior
Then no such behavior is present — only `create` is implemented
And the help text does not advertise reads or edits as available.

**Scenario: The sensing person is never supplied by the client**
Given the implemented command
When a reviewer inspects the request the command builds
Then the request body carries no `sensed_by` / person field
And the sensing person on the created tension is the token's own identity.

---

## Assumptions

- **Meeting-type is a routing hint, not a meeting binding**: `--meeting-type` is assumed to be stored as a categorization on the tension. (Informed by the `createTension` note that tensions captured this way are "not bound to a meeting" — the field signals intended forum, not an agenda attachment.)
- **Whitespace-only body is empty**: a body that trims to nothing is treated as missing. (Mirrors the trim-empty convention used elsewhere in the project; avoids recording a blank tension the server would likely reject.)
- **[ASSUMED] Role-id format is not validated client-side**: the command requires exactly one positional id but lets the API reject an unknown or malformed `role_` id (typically `404`/`422`), rather than enforcing the `^role_[0-9a-f]{32}$` pattern locally. (Mirrors how Role Projects (038) handles its `proj-id` — id-shape validation is the server's job.)

---

## Ambiguity Warnings

None — the capture boundary is fully determined: a single create operation, a required body, optional label and validated meeting-type, the created tension (with id) as output, and reads/edits/proposals explicitly deferred to their own future specs.
