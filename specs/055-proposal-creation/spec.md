# Specification: Proposal Creation

**Feature**: 055-proposal-creation
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Proposal Creation is the **anchor of the governance write path** — the command that turns a captured tension into a draft governance proposal, fulfilling VISION success criterion #2 (submit a valid proposal). Where Tension Capture (042) records the *seed* of a change, this command assembles the change itself: it creates a `draft` proposal anchored to an existing tension and carrying a set of **governance changes** (create role, add accountability, update policy, …). The command is `glassfrog proposal create <tension-id>`: the positional `<tension-id>` is the **anchor tension** (the `ten_` id from a prior capture or read), and the change set comes from a required `--changes` flag carrying a JSON array — supplied inline, from a file, or from piped standard input. It maps to `POST /proposals` (`createProposal`) and produces the created proposal — including its `prp_` id and `draft` status — which an agent later advances to circulation.

The whole proposal write surface is **Premium-gated** (async proposals): every write returns `403` when the feature is not enabled. Each entry in `changes` is a free-form governance command — `type` plus open, command-specific keys, with no per-type schema in the API — so the CLI **passes the changes through as supplied** and lets the server validate them. Typed, per-change builders (the *Unguided Change Construction* problem) are deliberately deferred to a separate future capability; this command owns only the create, with the change set as a verbatim pass-through.

It sits on the proven chain rather than rebuilding it: it hands the request to **Request Execution (010)**, authenticates through **Request Authentication (007)**, lets **Output Format Selection (020)** render the created proposal, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)** — the `403` Premium refusal surfaces as the permission outcome 015 already distinguishes. It is the write counterpart of the (future) proposal reads and the first link in the multi-step write flow (advance / respond / withdraw), all of which are distinct siblings under the `proposal` noun. The `create` sub-verb leaves room for them without claiming them here.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog proposal create <tension-id> --changes <source>`, the system creates a draft proposal anchored to that tension and carrying the supplied change set, and produces the created proposal as its result.
- When the user omits the required `<tension-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- When the user omits `--changes`, the system rejects the invocation as a usage error — naming `--changes` as required — and sends no request. A proposal with no change set is meaningless.

### Input — the change set

- The change set comes from the required `--changes` flag, resolved to one of three sources (the source convention mirrors how Output Format Selection (035) sources a template): when the value is the reserved keyword `stdin`, the system reads the JSON from piped standard input; otherwise, when the value resolves to an existing file path, the system reads the JSON from that file; otherwise the value is treated as an **inline JSON** array. A file literally named `stdin` is read only via a path such as `./stdin`.
- When the supplied change set cannot be parsed as JSON, or parses to something other than an array, the system reports a usage error naming the source, makes no API request, and runs no command — the malformed change set is caught before any write.
- When the change set parses to an **empty array**, the system rejects the invocation as a usage error — at least one change is required — and sends no request.
- When any element of the change set is not a JSON object, or is an object lacking a non-empty `type`, the system rejects the invocation as a usage error — every change must carry a `type` — and sends no request. This is the only per-change check the client makes.
- When the change set is a non-empty array of typed change objects, the system sends it through **verbatim** as the proposal's `changes` — preserving every command-specific key untouched beyond the `type` floor; the server validates each change's `type` value and command-specific keys.

### Output

- When the create succeeds, the system produces the created proposal — including its `prp_` id, its `draft` status, the anchor `tension_id`, and the `changes` it carries — as its result, and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the create succeeds, the system exits successfully (the created proposal, with its `prp_` id, is the load-bearing output a later advance-to-circulation step references).

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a Premium-disabled refusal (`403`), an unknown anchor tension (typically `404`), and a rejected change set or field (typically `422`) — the system reports that the create failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** turn a tension I captured into an actionable governance change,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** create a draft proposal anchored to that tension in one command.

**In order to** advance the proposal to circulation later,
**as an** AI agent assembling the write flow,
**I want** the create to return the proposal's `prp_` id and `draft` status.

**In order to** submit a large or complex change set without cramming JSON into the command line,
**as an** AI agent driving the CLI,
**I want to** read the `--changes` array from a file or from piped stdin.

**In order to** avoid submitting a meaningless empty proposal,
**as an** AI agent driving the CLI,
**I want** a missing or empty change set rejected before any request is made.

---

## Non-Behaviors

- The system must not advance a proposal to circulation, record a response, withdraw, list, or get proposals. **Why**: the proposal API offers `proposeProposal`, `createProposalResponse`, `withdrawProposal`, `listProposals`, and `getProposal`, but Proposal Creation owns the create alone — matching the project's one-capability-per-spec pattern. Each is its own future capability under the `proposal` noun.
- The system must not validate the *value* of a change's `type`, nor any command-specific key, of any individual change. **Why**: a change is a free-form governance command with no per-type schema in the API; beyond a minimal floor — every element is an object carrying a non-empty `type` — the CLI passes the array through verbatim and lets the server validate. Typed, per-change builders (recognizing each `type` and validating its keys) are the *Unguided Change Construction* problem — a separate, later capability. This command requires only that the change set be valid JSON, non-empty, and that each element carry a `type`.
- The system must not detect, pre-check, or interpret the Premium async-proposals feature-gate before sending. **Why**: the command always issues the request and surfaces the server's `403` through the shared permission outcome; recognizing the gate and signalling a plan limit distinctly are *Feature-Gate Recognition* and *Plan-Limit Signal* — separate future capabilities, not this one.
- The system must not set, override, or interpret the created proposal's status. **Why**: the API creates the proposal in `draft`; status transitions happen through the separate advance/withdraw actions. Exposing a status flag here would let the CLI claim a state the server owns.
- The system must not infer or supply the proposer. **Why**: the server derives the proposer from the token's identity; the CLI operates within that single identity (PROJECT constraint) and must not claim to propose on another person's behalf.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not send an `If-Match` precondition or otherwise handle optimistic concurrency. **Why**: a create has no prior `ETag` to guard against; concurrency on *edits* is the Clobbered Changes problem (052/053/054), already shipped for the edit path, not relevant to a create.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`POST /proposals`, `createProposal`)**: the system writes a new draft proposal. The request body carries `proposal.{tension_id, changes}` — the positional anchor and the verbatim change array. Data flows outbound (the anchor and changes in) and inbound (the created proposal, with its `prp_` id, `draft` status, and changes, back). The endpoint is **Premium-gated**: a `403` means async proposals are not enabled; a `404` means the anchor tension is unknown; a `422` means the server rejected the change set or a field. When the endpoint is unreachable or answers non-2xx, the command surfaces a transport or create failure and exits non-zero.
- **Filesystem / standard input (upstream inputs)**: when the `--changes` value resolves to an existing file path, the change set JSON is read from that file; when it is the reserved `stdin` keyword, the JSON is read from the piped stream; otherwise the value is inline JSON. Parse failures, an empty/non-array change set, and an element lacking `type` are caught here, before any request.
- **Request Execution (010) / Request Authentication (007)**: the command builds the `POST`, hands it to these seams to authenticate and execute it, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: the command hands the created proposal to the output seam, which renders it in the effective format.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx (including the `403` Premium refusal) / transport / not-authenticated outcomes to exit codes and messages through these seams.

---

## Driving Scenarios

### Happy path

**Scenario: Create a proposal with an inline change set**
Given a stored credential for an org and person
When the user runs `glassfrog proposal create ten_<id> --changes '[{"type":"CreateRole","name":"Scribe"}]'`
Then the system POSTs the proposal anchored to the tension, carrying the change array verbatim
And produces the created proposal — with its `prp_` id and `draft` status — as the result
And exits successfully.

**Scenario: Read the change set from a file**
Given a stored credential and a file `changes.json` holding a JSON array of changes
When the user runs `glassfrog proposal create ten_<id> --changes changes.json`
Then the system resolves the value to the existing file and parses the array from it
And sends it through verbatim as the proposal's changes
And produces the created proposal as the result
And exits successfully.

**Scenario: The created proposal's id and status are visible in JSON output**
Given a stored credential and `--output json`
When the user creates a proposal successfully
Then the rendered result contains the proposal's `prp_` id and a `draft` status
So a later step can reference it to advance the proposal to circulation.

### Error scenarios

**Scenario: Missing change set is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal create ten_<id>` with no `--changes`
Then the system rejects the invocation as a usage error naming `--changes` as required
And makes no API call
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and no token in the environment
When the user runs `glassfrog proposal create ten_<id> --changes '[{"type":"CreateRole"}]'`
Then the system surfaces the not-authenticated refusal pointing at how to store a credential
And makes no API call
And exits non-zero.

### Edge cases

**Scenario: Empty change set is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal create ten_<id> --changes '[]'`
Then the system rejects the invocation as a usage error — at least one change is required
And makes no API call
And exits non-zero.

**Scenario: Unparseable change set is rejected before any request**
Given a stored credential
When the user supplies an inline `--changes` value that is not valid JSON, or is not a JSON array
Then the system reports a usage error naming the source
And makes no API call
And exits non-zero.

**Scenario: A change without a type is rejected before any request**
Given a stored credential
When the user supplies a `--changes` array in which an element is not an object, or is an object lacking a non-empty `type`
Then the system rejects the invocation as a usage error — every change must carry a `type`
And makes no API call
And exits non-zero.

**Scenario: Read the change set from piped stdin**
Given a stored credential and a JSON array of changes piped on standard input
When the user runs `glassfrog proposal create ten_<id> --changes stdin`
Then the system reads and parses the array from the piped stream
And sends it through verbatim as the proposal's changes
And exits successfully.

**Scenario: Premium async proposals not enabled**
Given a stored credential on an organization without async proposals enabled
When the user creates a proposal
Then the API answers `403`
And the system reports the create failed, naming the HTTP status as the permission outcome, and exits non-zero.

**Scenario: Unknown anchor tension**
Given a stored credential
When the user creates a proposal against a `ten_<id>` that does not exist
Then the API answers `404`
And the system reports the create failed, naming the HTTP status, and exits non-zero.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Create does not reach into the rest of the write flow or the reads**
Given the implemented command
When a reviewer exercises `glassfrog proposal` for any advance-, respond-, withdraw-, list-, or get-style behavior
Then no such behavior is present — only `create` is implemented
And the help text does not advertise the other transitions or reads as available.

**Scenario: The change set is sent through verbatim beyond the type floor**
Given the implemented command and a non-empty array of objects each carrying a `type`
When a reviewer inspects the request the command builds
Then the `proposal.changes` body matches the supplied array exactly
And the command validated only the presence of a non-empty `type` on each element — every command-specific key is passed through untouched and unread.

**Scenario: No client-side feature-gate pre-check**
Given the implemented command
When a reviewer inspects the create against an org without async proposals
Then the command issues the request and surfaces the server's `403`
And does not refuse locally based on any client-side Premium check.

---

## Assumptions

- **Changes are a verbatim JSON pass-through above a `type` floor**: `--changes` is assumed to carry a JSON array of governance-command objects sent unchanged; the client requires valid JSON, at least one element, and a non-empty `type` on every element, but validates no `type` value and no command-specific key. (Follows the FEATURE-MODEL decision that `ProposalChange` has no per-type schema in the spec — so the CLI lets the server validate values — while the schema marks `type` itself required, so a typeless element is caught locally; typed builders are deferred.)
- **The `--changes` source is resolved like a template source (035)**: `stdin` is a reserved keyword; a value resolving to an existing file is read from disk; anything else is inline JSON. (Mirrors 035's reserved-name-wins precedent; a file literally named `stdin` is reachable via `./stdin`.)
- **[ASSUMED] Tension-id format is not validated client-side**: the command requires exactly one positional id but lets the API reject an unknown or malformed `ten_` id (typically `404`/`422`), rather than enforcing the `^ten_[0-9a-f]{32}$` pattern locally. (Mirrors how Tension Capture (042) and Role Projects (038) treat their ids — id-shape validation is the server's job.)
- **[ASSUMED] The created proposal is always `draft`**: the command does not assert the returned status; the API documents creation in `draft` and the command renders whatever the server returns. (Informed by the `createProposal` description.)

---

## Ambiguity Warnings

None — both open questions (the `--changes` source-selection convention and the client-side validation floor) were resolved in the clarification session below. The create boundary is fully determined: a positional anchor tension, a required `--changes` array sourced inline / from a file / from `stdin`, a `type` floor over an otherwise verbatim pass-through, the created draft proposal (with its `prp_` id) as output, and the Premium `403`, reads, advance, respond, and withdraw all explicitly deferred to their own specs.

---

## Clarifications

### Session 2026-06-14

- **`--changes` source selection**: The flag resolves its source the way Output Format Selection (035) resolves a template source — `stdin` is a reserved keyword that reads piped standard input; a value resolving to an existing file path is read from that file; any other value is treated as inline JSON. A file literally named `stdin` is reached via `./stdin`.
- **Client-side validation floor**: The CLI enforces a minimal floor before sending — the change set must be valid JSON, a non-empty array, and every element must be an object carrying a non-empty `type` (the one key the `ProposalChange` schema marks required). Beyond that floor, every command-specific key is passed through verbatim and left for the server to validate; typed per-change builders remain deferred.
