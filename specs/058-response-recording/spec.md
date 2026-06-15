# Specification: Response Recording

**Feature**: 058-response-recording
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Response Recording is the **consent-window write** of the Governance Proposals flow — the step that lets a circle member answer a proposal that is circulating for acceptance. Where Proposal Reads (056) surfaces the proposals in flight and Advance to Circulation (057) opens the consent window, this command records one member's response to a single circulating proposal. The two answers are `no_objection` (willing to let the proposal pass as written — when every expected responder has no-objection, the server auto-accepts) and `bring_to_meeting` (wants live discussion; persists on the proposal and blocks auto-acceptance). Recording a response is the act that completes the path to auto-acceptance, so it is core to VISION success #2.

The command is `glassfrog proposal respond <prp-id> --response <value>`: the positional `<prp-id>` names the circulating proposal, and the required `--response` carries one of the two consent values, validated locally before any request. The **responding person is the token's own identity** — the server derives it; the CLI supplies no person. It maps to `POST /proposals/{proposal_id}/responses` (`createProposalResponse`) and produces the recorded response — including the **parent proposal's status at the time of response**, which reads `accepted` when this very response triggered auto-acceptance.

Like the other governance writes it sits on the proven chain rather than rebuilding it: it hands the request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, lets **Output Format Selection (020)** render the recorded response, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**. It is the **Premium-gated** consume/respond verb of the verb-rich `proposal` group established by 055/056 — a sibling of `create`, `propose`, and `withdraw`, deliberately scoped to *recording a response alone*, distinct from reading the aggregate `response_summary` (that is Proposal Reads, 056) and from the operator-layer confirmation that Write-Safety Guardrail will add.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog proposal respond <prp-id> --response no_objection`, the system records that consent response against the named proposal and produces the recorded response as its result.
- When the user omits the required `<prp-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- When the user omits `--response`, the system rejects the invocation as a usage error — naming `--response` as required and listing the supported values — and sends no request. A consent response has no sensible default; the member must choose explicitly.

### Input

- When the user supplies `--response <value>` with a supported value (`no_objection` or `bring_to_meeting`), the system validates it against that vocabulary before issuing any request and sends it as the response value.
- When the user supplies a `--response` outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how `--meeting-type` is validated in Tension Capture (042) and `--status` in Proposal Reads (056)).
- The system supplies no responding person; the server records the response under the token's own identity.

### Output

- When recording succeeds, the system produces the recorded response — including its `prr_` id, the value recorded, and the **parent proposal's status at the time of response** — as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the recorded response triggered auto-acceptance, the parent proposal's status in the result reads `accepted` (rather than `proposed_outside_meeting`); the system surfaces that status as the load-bearing signal that the response closed the consent window, without computing acceptance itself.
- When recording succeeds, the system exits successfully.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response, the system reports that recording failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results. In particular: an unknown or invisible proposal is typically `404`; a Premium plan-gate (async proposals not enabled) is `403`, surfaced as the shared permission outcome with no response-specific plan-limit interpretation; and a **second response by the same person** is `422`, surfaced as a generic API error — the command does not retry it or treat it specially.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** let a circulating proposal pass to auto-acceptance,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** record a `no_objection` response with one command.

**In order to** keep a proposal from auto-accepting so it can be discussed live,
**as a** practitioner with a concern,
**I want to** record a `bring_to_meeting` response that blocks auto-acceptance.

**In order to** know whether my response closed the consent window,
**as an** AI agent tracking the write flow,
**I want** the recorded response to carry the parent proposal's status (which reads `accepted` when my response triggered auto-acceptance).

**In order to** avoid an accidental or empty consent answer,
**as an** AI agent driving the CLI,
**I want** a missing or unsupported response value rejected before any request is made.

---

## Non-Behaviors

- The system must not list, read, or aggregate responses, and must not expose per-person response data. **Why**: the API offers no per-person response read — the proposal carries an aggregate `response_summary` only (`ProposalVote` is explicitly "no per-person attribution"), and reading that summary belongs to Proposal Reads (056), not the recording write. Response Recording owns the create alone, matching the project's one-capability-per-spec pattern.
- The system must not record more than one response, nor retry or specially handle the `422` returned for a second response by the same person. **Why**: the API enforces one response per person per proposal; a retry would either be a no-op or mask the server's own rule. The `422` surfaces through the shared error chain like any other rejected write.
- The system must not create, advance (`propose`), or withdraw a proposal. **Why**: those are sibling verbs of the `proposal` group with their own specs (Proposal Creation, Advance to Circulation (057), Withdraw Proposal); recording a response is its own consent-window act and coupling them would fork the staged write path.
- The system must not interpret the Premium plan-gate `403` as anything other than the shared permission outcome. **Why**: turning a recognized plan-limit `403` into an actionable "not available on your plan" diagnostic is Plan-Limit Signal, a separate later capability; this command surfaces the `403` through API Error Extraction (015) like any permission denial.
- The system must not prompt for confirmation before recording. **Why**: operator-layer confirmation of governance writes is Write-Safety Guardrail, a separate capability at the operator layer; Response Recording is scoped to the write itself, matching how Tension Capture (042) records without a prompt.
- The system must not send an `If-Match` precondition or otherwise handle optimistic concurrency. **Why**: recording a response is a create (`POST`) with no prior `ETag` to guard against; guarded writes (053) and stale-write surfacing (054) apply to conditional *edits*, not to this append-only create.
- The system must not infer, set, or override the responding person. **Why**: the server derives it from the token's identity; the CLI operates within that single identity (PROJECT constraint) and must not claim to respond on another person's behalf.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`POST /proposals/{proposal_id}/responses`, `createProposalResponse`)**: the system records a response on a circulating proposal. The path `proposal_id` is the target proposal; the request body carries `response.value` (`no_objection` | `bring_to_meeting`). Data flows outbound (the response value in) and inbound (the recorded `ProposalVote` — its `prr_` id, recorded value, and the parent `proposal_status` — back). This endpoint is **Premium-gated**: a `403` means async proposals are not enabled on the org's plan. A `404` means the proposal is unknown or invisible; a `422` means the body was rejected or this person already responded. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or recording failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the `POST`, hands it to these seams to authenticate and execute it, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: the command hands the recorded response to the output seam, which renders it in the effective format.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **`proposal` command group (055/056)**: `respond` attaches as a verb leaf to the existing `proposal` group; it does not create the group or re-declare the shared model.

---

## Driving Scenarios

### Happy path

**Scenario: Record a no-objection response**
Given a stored credential for an org and person
And a proposal `prp_<id>` circulating for acceptance
When the user runs `glassfrog proposal respond prp_<id> --response no_objection`
Then the system POSTs the response value to the proposal
And produces the recorded response — with its `prr_` id and the parent proposal's status — as the result
And exits successfully.

**Scenario: Record a bring-to-meeting response**
Given a stored credential
And a proposal `prp_<id>` circulating for acceptance
When the user runs `glassfrog proposal respond prp_<id> --response bring_to_meeting`
Then the system sends `bring_to_meeting` as the response value
And produces the recorded response as the result
And exits successfully.

**Scenario: A response that triggers auto-acceptance shows the accepted status**
Given a stored credential and `--output json`
And a proposal `prp_<id>` awaiting only this member's response
When the user records `no_objection` successfully
Then the rendered result carries the parent proposal's status as `accepted`
So an agent can tell the response closed the consent window.

### Error scenarios

**Scenario: Missing response value is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal respond prp_<id>` with no `--response`
Then the system rejects the invocation as a usage error naming `--response` as required and listing the supported values
And makes no API call
And exits non-zero.

**Scenario: Unsupported response value is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal respond prp_<id> --response abstain`
Then the system rejects the invocation as a usage error, naming `abstain` and the supported set (`no_objection`, `bring_to_meeting`)
And makes no API call
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and no token in the environment
When the user runs `glassfrog proposal respond prp_<id> --response no_objection`
Then the system surfaces the not-authenticated refusal pointing at how to store a credential
And makes no API call
And exits non-zero.

### Edge cases

**Scenario: A second response by the same person is rejected by the server**
Given a stored credential
And the same person has already responded to `prp_<id>`
When the user runs `glassfrog proposal respond prp_<id> --response no_objection` again
Then the API answers `422`
And the system reports the recording failed, naming the HTTP status, and exits non-zero
And does not retry.

**Scenario: Premium plan-gate is surfaced as a permission failure**
Given a stored credential on an org without async proposals enabled
When the user runs `glassfrog proposal respond prp_<id> --response no_objection`
Then the API answers `403`
And the system reports the recording failed as a permission outcome, naming the HTTP status, and exits non-zero
And adds no plan-limit-specific interpretation.

**Scenario: Unknown or invisible proposal**
Given a stored credential
When the user responds to a `prp_<id>` that does not exist or is not visible
Then the API answers `404`
And the system reports the recording failed, naming the HTTP status, and exits non-zero.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Recording does not reach into the read surface**
Given the implemented command
When a reviewer exercises `glassfrog proposal respond` for any list- or read-of-responses behavior
Then no such behavior is present — only recording a single response is implemented
And the help text does not advertise reading or aggregating responses.

**Scenario: The responding person is never supplied by the client**
Given the implemented command
When a reviewer inspects the request the command builds
Then the request body carries no person field
And the responding person on the recorded response is the token's own identity.

**Scenario: The response value is validated before any request**
Given the implemented command
When a reviewer supplies a `--response` value outside the vocabulary
Then the command exits as a usage error with no request sent (a transport tripwire confirms no call was made).

---

## Assumptions

- **[ASSUMED] Command spelling — `proposal respond <prp-id> --response <value>`**: the `respond` verb leaf is the surface the Proposal Reads (056) interface accord already names; whether the consent value rides on a `--response` flag or a second positional is an interface detail. This spec assumes a validated `--response` flag, mirroring Tension Capture (042)'s validated `--meeting-type` flag. The exact spelling is conventional and adjustable at the interface/build step without changing behavior.
- **[ASSUMED] Proposal-id format is not validated client-side**: the command requires exactly one positional id but lets the API reject an unknown or malformed `prp_` id (typically `404`), rather than enforcing the `^prp_[0-9a-f]{32}$` pattern locally. (Mirrors how Proposal Reads (056) passes its `prp-id` through to a clean `404`.)
- **Response value is required with no default**: there is no implicit consent answer; the member must choose `no_objection` or `bring_to_meeting` explicitly, so an omitted `--response` is a usage error. (Grounded in the consent model — silence is not a recordable response.)

---

## Ambiguity Warnings

None affecting behavior — the recording boundary is fully determined: a single create against one circulating proposal, a required response value from a two-value vocabulary validated locally, the recorded response (with the parent proposal's status) as output, Premium gating surfaced through the shared permission outcome, and reads/other-transitions/confirmation explicitly deferred to their own specs. The only open point is the conventional command spelling (flag vs positional for the value), recorded as an `[ASSUMED]` interface detail for the Crafter to pin.
