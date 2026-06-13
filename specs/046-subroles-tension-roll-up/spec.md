# Specification: Subroles Tension Roll-up

**Feature**: 046-subroles-tension-roll-up
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Subroles Tension Roll-up is the **one-level cross-role read for tensions** — the capability Tension Reads (043) deliberately carved out and forward-declared. Where 043 lists the tensions a single role senses (`tension list <role-id>`), this command rolls up the tensions sensed across an anchor role's **direct sub-roles** in one read: `glassfrog tension subroles <role-id>` → `GET /roles/{role_id}/subroles/tensions` → `listSubrolesTensions`. It answers "what are the circles inside this one sensing?" — the view a circle lead or facilitator wants before a governance meeting, without fetching each child role's tensions one at a time.

The roll-up is **one level only** — the anchor's direct children, not a transitive closure of the whole subtree. The anchor must be an expanded role (a circle, `has_subroles: true`); the API returns `404` for a leaf role, which the command surfaces as a plain read failure rather than interpreting. It continues the `tension <verb>` grammar 042/043 grew (`create` / `list` / `get`, now `subroles`) so each verb maps to exactly one endpoint, and it sits on the proven read chain rather than rebuilding it: it hands requests to **Request Execution (010)**, reads identity through **Request Authentication (007)**, walks the list through **Pagination (016)**, lets **Output Format Selection (020)** render the result, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It reuses the `glassfrog.Tension` model shape (042/043) — the same schema the endpoint returns — rather than defining its own.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog tension subroles <role-id>`, the system reads the tensions sensed across that anchor role's direct sub-roles and produces them as a list result.
- When the user omits the required `<role-id>`, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.

### Filter

- When the user supplies `--status <value>`, the system validates the value against the tension status vocabulary (`unprocessed`, `processed`, `archived`) before issuing any request; a supported value is sent as the `status` query parameter, narrowing the roll-up to tensions in that status.
- When the user supplies a `--status` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how Tension Reads (043), My Projects (014), and Role Projects (038) validate `--status`).
- When the user supplies no filter, the system requests every tension across the direct sub-roles.

### Output

- When the read succeeds, the system produces the tension data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the anchor's sub-roles carry no tensions, or none match the supplied status, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the rolled-up tensions span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the tensions gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including an anchor that is a leaf role (no sub-roles) or an unknown role id, both typically `404` — the system reports that the read failed, naming the HTTP status, and exits non-zero. It adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see everything the circles inside this one are sensing before a governance meeting,
**as a** practitioner facilitating a circle,
**I want to** roll up the tensions across a circle's direct sub-roles with one command.

**In order to** assemble the tensions a circle is carrying below it without fetching each child role separately,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** read the sub-roles' tensions in a single roll-up by the anchor role's id.

**In order to** focus on just the unworked tensions surfacing across a busy circle's children,
**as an** AI agent assembling context,
**I want to** narrow the roll-up by status.

**In order to** trust I am seeing every tension the sub-roles are carrying,
**as a** practitioner in a large circle,
**I want** the roll-up to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not roll up tensions beyond the anchor's **direct** sub-roles. **Why**: the endpoint (`listSubrolesTensions`) is a one-level roll-up, not a transitive closure of the whole subtree; recursing into grand-children would invent a behavior the API does not offer and silently change what "subroles" means.
- The system must not list the anchor role's *own* tensions. **Why**: that is Tension Reads' `tension list <role-id>` (043), backed by a different endpoint (`listRoleTensions`). This command surfaces only the children's tensions; merging the two would fork 043's contract and blur which role sensed what.
- The system must not special-case the leaf-role `404` into a "this role has no sub-roles" message or an empty-list success. **Why**: the API itself answers `404` for a leaf anchor; the CLI is a faithful surface (VISION Exclusion 1) and passes the status through the shared error handling, distinct from the genuine empty-list success returned when sub-roles exist but carry no tensions.
- The system must not expose the roll-up as a `--subroles` flag on `tension list`. **Why**: 043 deliberately scoped `tension list <role-id>` to a role's own tensions and declared the roll-up its own capability; overloading one verb across two endpoints with different "anchor must have sub-roles" semantics would fork that contract. Each `tension` verb maps to exactly one endpoint.
- The system must not create, update, or discard tensions. **Why**: this is a read surface alone, matching the one-capability-per-spec pattern; capture is 042, editing is Tension Update (044), and a read that mutated would fork those write contracts.
- The system must not set, override, or recompute a tension's status, nor treat `--status` as anything but a list filter. **Why**: status is server-owned; on a read it is only a narrowing query parameter, never a write.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not interpret, summarize, or advise on the tensions it rolls up. **Why**: the CLI is a faithful API surface, not a Holacracy facilitator (VISION Exclusion 1); it produces the tension data and lets the operator reason about it.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /roles/{role_id}/subroles/tensions`)**: the system reads the direct sub-roles' tensions through this endpoint. It accepts an optional `status` (enum) query filter and is paginated (`per_page` / cursor). Data flows inbound (tensions in, nothing written). A `404` means the anchor is a leaf role (no sub-roles) or the role id does not exist. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the rolled-up tension data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **Tension Reads (043) — sibling**: established the `tension list` / `tension get` reads and the tension model shape this roll-up surfaces and reuses; 043's non-behavior #1 forward-declared this capability and its endpoint.

---

## Driving Scenarios

### Happy path

**Scenario: Roll up tensions across a circle's direct sub-roles**
Given a valid stored credential and an anchor role whose direct sub-roles carry tensions
When the user runs `glassfrog tension subroles <role-id>`
Then the system reads `GET /roles/{role_id}/subroles/tensions`
And produces the sub-roles' tensions as a list result
And exits successfully.

**Scenario: Narrow the roll-up by status**
Given an anchor role whose sub-roles carry tensions in several statuses
When the user runs `glassfrog tension subroles <role-id> --status unprocessed`
Then the value is accepted as a supported status
And the system sends `status=unprocessed` on `GET /roles/{role_id}/subroles/tensions`
And produces only the unprocessed tensions as a list result.

**Scenario: Roll-up walks every page to completion**
Given an anchor role whose sub-roles' tensions span more than one page
When the user runs `glassfrog tension subroles <role-id>`
Then the system walks every page through Pagination (016)
And produces the complete set of sub-role tensions
And exits successfully.

### Error scenarios

**Scenario: Anchor is a leaf role**
Given a valid stored credential and a role that has no sub-roles
When the user runs `glassfrog tension subroles <role-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero
And adds no "this role has no sub-roles" interpretation of its own.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog tension subroles <role-id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Sub-roles exist but carry no tensions**
Given a valid stored credential and an anchor whose sub-roles carry no tensions
When the user runs `glassfrog tension subroles <role-id>`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported status value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog tension subroles <role-id> --status open`
Then the system rejects it as a usage error, naming the unsupported value and the supported set (`archived`, `processed`, `unprocessed`)
And issues no request
And exits with the usage-error code.

**Scenario: Paginated roll-up with first-page opt-out**
Given an anchor whose sub-roles' tensions span more than one page
When the user runs `glassfrog tension subroles <role-id>` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The roll-up is one level only**
Given an anchor role with direct sub-roles that themselves contain grand-child roles carrying tensions
When the user runs `glassfrog tension subroles <role-id>`
Then only the direct sub-roles' tensions are read through `GET /roles/{role_id}/subroles/tensions`
And the command makes no attempt to recurse into grand-child roles.

**Scenario: A leaf-role 404 is a failure, not an empty success**
Given a leaf anchor role for which the API answers `404`
When the command runs
Then the outcome is the shared non-2xx read failure naming the status and a non-zero exit
And it is distinct from the empty-list success returned when sub-roles exist but carry no tensions.

**Scenario: An unsupported status costs no request**
Given a `--status` value outside the tension status vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful roll-up read
When the result reaches Output Format Selection (020)
Then the command supplied structured tension data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **First-page opt-out flag is the shared one**: the roll-up reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038 / 043), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **`Tension` model is reused, not redefined**: the tension shape grown in 042/043 (carrying `ten_` id, body, label, status, meeting-type, and sensing role / person) is the same schema this endpoint returns, so no new leaf model is needed; the renderers (019) decide how much to show per format. (Mirrors the per-role reuse pattern of 033/034/038/043.)
- **Status vocabulary tracks the spec enum**: `--status` is validated against the tension status set (`unprocessed`, `processed`, `archived`) before any request — the values `listSubrolesTensions` accepts. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`). (Reflects the spec's documented `?status=` enum.)
- **[ASSUMED] Role-id format is not validated client-side**: the read requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing the `^role_…` pattern locally. (Mirrors how Role Projects (038), Tension Capture (042), and Tension Reads (043) leave id-shape validation to the server.)

---

## Ambiguity Warnings

_None — the feature follows the established 043 read pattern (status-validated list filter, full-walk pagination with first-page opt-out, structured output via 020) over a single new endpoint. The two design calls were settled in conversation: the grammar is a distinct `tension subroles <role-id>` verb (not a `--subroles` flag on `tension list`, preserving 043's one-verb-one-endpoint boundary), and the leaf-role `404` is surfaced as the shared read failure with no special-cased interpretation._
