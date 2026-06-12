# Specification: Tension Update

**Feature**: 044-tension-update
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Tension Update is the **edit operation for an existing tension** — the third member of the `tension` command family, after Tension Capture (042) wrote one and Tension Reads (043) surfaced them. It changes the fields of a tension already on the record: `glassfrog tension update <ten-id>` → `PATCH /tensions/{id}` (`updateTension`), producing the updated tension. Where capture creates the proposal seed and reads expose it, update lets an operator correct the body, retitle it, reroute it, or move it through its lifecycle — including the explicit `archived` transition that capture deliberately left to "the future `tension update`."

It continues the verb grammar 042 opened and 043 extended: `tension create`, `tension list`, `tension get`, and now `tension update <ten-id>` — a sub-verb under `tension`, not a new top-level noun. It sits on the proven chain rather than rebuilding it: it hands the request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render the updated tension, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the `glassfrog.Tension` model 042 established and the `ContentType` request field 042 added for a body-bearing write. The one field update can set that capture could not is **status**: on a `PATCH` the server allows an explicit transition (notably to `archived`), so update validates and forwards `--status` where capture refused it.

It is deliberately scoped to *editing fields alone*: it does not soft-delete (Tension Discard, 045), and it does not guard against concurrent edits — optimistic concurrency (`If-Match`/`ETag`) is **Clobbered Changes**, a separate Client-Foundation capability that becomes relevant once writes exist. Update writes last-write-wins for now, exactly as the API does when `If-Match` is omitted.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog tension update <ten-id>` with at least one editable field flag, the system applies those field changes to the named tension and produces the updated tension as its result.
- When the user omits the required `<ten-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- When the user supplies no editable field flag at all (none of `--body`, `--label`, `--status`, `--meeting-type`), the system rejects the invocation as a usage error — naming that at least one field is required — and sends no request. An update that changes nothing is meaningless.

### Input

- When the user supplies `--body "<text>"`, the system sends it as the tension's new body. When the supplied `--body` is empty or only whitespace, the system rejects the invocation as a usage error and sends no request — a tension cannot be blanked out (mirroring how capture (042) rejects a blank body).
- When the user supplies `--label "<text>"`, the system sends it as the tension's new label.
- When the user supplies `--status <value>`, the system validates the value against the tension status vocabulary (`unprocessed`, `processed`, `archived`) before issuing any request; a supported value is sent as the tension's `status`. Unlike capture, status is editable here — the server allows the transition (e.g. to `archived`) on a `PATCH`, then re-runs its own auto-computation on save.
- When the user supplies a `--status` outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how `--status` is validated in My Projects (014), Role Projects (038), and Tension Reads (043)).
- When the user supplies `--meeting-type <value>`, the system validates it against the meeting-type vocabulary (`tactical`, `governance`) before issuing any request; a supported value is sent as the tension's `meeting_type`. An unsupported value is rejected as a usage error before any request, naming the value and the supported set (mirroring capture, 042).
- Whatever subset of fields the user supplies, the system sends only those fields (partial update) and leaves every unsupplied field untouched on the server.

### Output

- When the update succeeds, the system produces the updated tension — including its `ten_` id, body, label, status, meeting-type, and sensing role / person — as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the update succeeds, the system exits successfully.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a tension id that does not exist (typically `404`) and a rejected field or value (typically `422`) — the system reports that the update failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** fix a tension I worded poorly or mis-titled,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** edit its body or label by id with one command.

**In order to** retire a tension I have finished working,
**as a** practitioner managing my governance backlog,
**I want to** move it to `archived` through the update command.

**In order to** reroute a tension to the right forum after I have reconsidered it,
**as a** practitioner,
**I want to** change its meeting-type without recreating it.

**In order to** avoid issuing a request that changes nothing,
**as an** AI agent driving the CLI,
**I want** an update with no field flags rejected before any request is made.

---

## Non-Behaviors

- The system must not create, list, get, or soft-delete tensions. **Why**: Tension Update owns the field-edit alone, matching the project's one-capability-per-spec pattern. Capture is Tension Capture (042); the reads are Tension Reads (043); soft-delete is Tension Discard (045). A command that strayed into those would fork their contracts.
- The system must not send an `If-Match` precondition or otherwise guard against concurrent edits. **Why**: optimistic concurrency is **Clobbered Changes**, a separate Client-Foundation capability (it is relevant across every write, not this one command). Until it lands, update writes unconditionally — last-write-wins — exactly as the API behaves when `If-Match` is omitted. When Clobbered Changes lands, this command opts into the shared guard rather than growing its own.
- The system must not offer a way to *clear* a field to empty/null (e.g. removing a label or unsetting `meeting_type`). **Why**: `--meeting-type` (and the others) only *set* a validated value; a clear-to-null affordance is speculative scope with no requested consumer, and the repo's anti-speculation idiom adds a knob only when a real consumer needs it. A blank `--body` is a usage error, not a clear.
- The system must not perform its own status auto-computation, nor reconcile the value it sends against the server's recompute. **Why**: status is server-owned — the API accepts an explicit transition on `PATCH` and then re-runs its own computation on save; the command forwards the validated value and renders whatever the server returns, claiming no authority over the final state.
- The system must not infer, set, or override the sensing person (`sensed_by`) or re-target the sensing role. **Why**: the server derives the person from the token's identity, and the sensing role is fixed at capture; update edits the tension's own fields, not its provenance.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`PATCH /tensions/{id}`, `updateTension`)**: the system edits an existing tension. The path `id` is the tension's `ten_` id; the request body carries `tension.{body, label, status, meeting_type}` — only the fields the user supplied. Data flows outbound (the changed fields in) and inbound (the updated tension, with the server's re-computed status, back). A `404` means the tension id is unknown; a `422` means the server rejected a field or value. The endpoint accepts an optional `If-Match` for optimistic concurrency, which this command does not send (Clobbered Changes). When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or update failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the `PATCH` (with its `ContentType`), hands it to these seams to authenticate and execute it, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: the command hands the updated tension to the output seam, which renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **Tension Capture (042) / Tension Reads (043) — siblings**: 042 established the `tension` command, the `create`/`list`/`get` verbs, the `Tension` model shape, and the `ContentType` request field this write reuses; update continues that grammar and forwards the `--status` capture withheld.

---

## Driving Scenarios

### Happy path

**Scenario: Edit a tension's body**
Given a stored credential and an existing `ten_` id
When the user runs `glassfrog tension update ten_<id> --body "Roadmap updates lag behind shipped work."`
Then the system PATCHes the tension with the new body only
And produces the updated tension as the result
And exits successfully.

**Scenario: Archive a tension via status transition**
Given a stored credential and an existing `ten_` id
When the user runs `glassfrog tension update ten_<id> --status archived`
Then the value is accepted as a supported status
And the system sends `status=archived` on the `PATCH`
And produces the updated tension as the result
And exits successfully.

**Scenario: Change label and meeting-type together**
Given a stored credential and an existing `ten_` id
When the user runs `glassfrog tension update ten_<id> --label "Roadmap drift" --meeting-type governance`
Then the system sends `label` and `meeting_type=governance` (and no other fields)
And produces the updated tension as the result
And exits successfully.

### Error scenarios

**Scenario: No editable field is rejected before any request**
Given a stored credential
When the user runs `glassfrog tension update ten_<id>` with no field flags
Then the system rejects the invocation as a usage error, naming that at least one field is required
And makes no API call
And exits non-zero.

**Scenario: Unsupported status is rejected before any request**
Given a stored credential
When the user runs `glassfrog tension update ten_<id> --status open`
Then the system rejects the invocation as a usage error, naming `open` and the supported set (`unprocessed`, `processed`, `archived`)
And makes no API call
And exits non-zero.

**Scenario: Unknown tension id**
Given a stored credential
When the user runs `glassfrog tension update ten_<id> --body "<text>"` against an id no tension has
Then the API answers `404`
And the system reports the update failed, naming the HTTP status, and exits non-zero.

### Edge cases

**Scenario: Whitespace-only body is treated as empty**
Given a stored credential
When the user runs `glassfrog tension update ten_<id> --body "   "`
Then the system rejects the invocation as a usage error — a body cannot be blanked
And makes no API call
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and no token in the environment
When the user runs `glassfrog tension update ten_<id> --body "<text>"`
Then the system surfaces the not-authenticated refusal pointing at how to store a credential
And makes no API call
And exits non-zero.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Update edits fields and nothing else**
Given the implemented command
When a reviewer exercises `glassfrog tension update` for any create-, list-, get-, or delete-style behavior
Then no such behavior is present — only field edits are implemented
And the help text does not advertise creation, reads, or deletion.

**Scenario: Only supplied fields are sent**
Given the implemented command
When a reviewer inspects the request built for an update that supplies a subset of fields
Then the request body carries only the supplied fields, and no `If-Match` header is sent.

**Scenario: An unsupported status or meeting-type costs no request**
Given a `--status` or `--meeting-type` value outside its vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

---

## Assumptions

- **Partial update sends only supplied fields**: a `PATCH` carries just the flags the user gave; unsupplied fields are omitted from the body and left untouched server-side. (Standard `PATCH` semantics, and the `updateTension` body fields are all optional in the spec.)
- **Status is editable on update, unlike capture**: the API description states status transitions (including to `archived`) are allowed on `PATCH`, with auto-computation still running on save — so the command forwards a validated `--status` and renders the server's result rather than predicting it. (Reflects the `updateTension` spec note; resolves capture's deferral of `--status` to "the future update".)
- **Status and meeting-type vocabularies track the spec enums**: `--status` is validated against (`unprocessed`, `processed`, `archived`) and `--meeting-type` against (`tactical`, `governance`), the values `TensionInput` accepts. The accepted sets track the vendored spec (`spec/glassfrog-api-v5.yaml`).
- **[ASSUMED] Tension-id format is not validated client-side**: the command requires exactly one positional id but lets the API reject an unknown or malformed `ten_` id (typically `404`), rather than enforcing the `^ten_[0-9a-f]{32}$` pattern locally. (Mirrors how Role Projects (038), Tension Capture (042), and Tension Reads (043) leave id-shape validation to the server.)

---

## Ambiguity Warnings

_None — the edit boundary is fully determined: a single partial `PATCH` over body/label/status/meeting-type, at least one field required, status now editable (including `archived`), the updated tension as output, and delete / optimistic concurrency (Clobbered Changes) / field-clearing explicitly deferred to their own capabilities._
