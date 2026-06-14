# Specification: Proposal Reads

**Feature**: 056-proposal-reads
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Proposal Reads is the **read surface for governance proposals** — the way an agent or practitioner sees the proposals in flight before, during, and after the write flow that creates and advances them. It is two reads: list the proposals visible to the caller (`glassfrog proposal list` → `GET /proposals` → `listProposals`), optionally narrowed by status, circle, proposer, or date; and read a single proposal by its `prp_` id (`glassfrog proposal get <prp-id>` → `GET /proposals/{id}` → `getProposal`), which carries the proposal's free-form `changes[]`, its aggregate `response_summary`, and the `available_transitions` the caller may invoke on it right now. It is the proposal-domain analogue of Tension Reads (043) — a list plus a read-by-id — and the read half of the **Governance Proposals** solution, pairing with Proposal Creation (the concurrently-specified sibling that opens `proposal create`) as the core read/write pair, with no dependency in either direction.

It continues the verb-grammar pattern Tension Reads established for a verb-rich noun: because the proposal surface carries several verbs (create, propose, withdraw, respond), the reads are `proposal list` and `proposal get` rather than a plural/singular noun pair, leaving the noun unambiguous for its siblings. Unlike Tension Reads, the list is **global, not by-role** — `GET /proposals` takes no path id; the circle is an optional `role_id` filter, not a required positional. The reads sit on the proven read chain rather than rebuilding it: they hand requests to **Request Execution (010)**, read identity through **Request Authentication (007)**, walk the list through **Pagination (016)**, let **Output Format Selection (020)** render the result, and map outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**.

Notably, and unlike the proposal *write* flow, these reads are **not Premium-gated** — `listProposals` and `getProposal` answer on any plan; only the write transitions return `403` when async proposals are disabled. Per-person response attribution is intentionally not exposed by the API, so the reads surface aggregate response counts only.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog proposal list`, the system reads the proposals visible to the caller and produces them as a list result.
- When the user runs `glassfrog proposal get <prp-id>`, the system reads the single proposal with that id and produces it as a single-proposal result, including its `changes[]`, aggregate `response_summary`, and `available_transitions`.
- When the user omits the required id on `proposal get`, passes more than one positional id, passes a positional to `proposal list` (which takes none), or passes an unknown flag on either command, the system rejects the invocation as a usage error and calls no API.

### Filters (list)

- When the user supplies `--status <value>` on `proposal list`, the system validates the value against the proposal status vocabulary (`draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts`) before issuing any request; a supported value is sent as the `status` query parameter, narrowing the list to proposals in that status.
- When the user supplies a `--status` value outside that vocabulary, the system rejects it as a usage error before issuing any request — naming the unsupported value and the supported set — and sends no request (mirroring how My Projects (014), Role Projects (038), and Tension Reads (043) validate `--status`).
- When the user supplies any of `--role-id`, `--proposer-id`, `--proposed-after`, or `--accepted-after`, the system sends each as its corresponding query parameter (`role_id`, `proposer_id`, `proposed_after`, `accepted_after`), narrowing the list; supplied filters combine. The system leaves the shape of these values (id pattern, timestamp format) for the server to validate rather than enforcing it client-side.
- When the user supplies no filter, the system requests every proposal visible to the caller.
- When the user supplies any list filter on the single read (`proposal get <prp-id>`), the system rejects the invocation as a usage error — the filters apply only to the list.

### Output

- When a read succeeds, the system produces the proposal data as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- When the single read renders, the `changes[]` entries are surfaced as supplied by the server — each change is `type` plus its free-form properties, passed through without per-type interpretation — alongside the aggregate `response_summary` (total, no-objection, bring-to-meeting counts) and the `available_transitions` list.
- When no proposals are visible, or none match the supplied filters, the system produces an empty list result and exits successfully — an empty list is a valid answer, not an error.

### Completeness of the list

- When the visible proposals span more than one page, the system walks every page through Pagination (016) by default and produces the complete set.
- When the user supplies the first-page opt-out flag, the system makes a single page request and, if more pages exist, produces the first page flagged incomplete with a clear "more exist" signal — so even the opted-out result is never silently truncated.
- When the walk cannot complete (a page fails mid-walk), the system produces the proposals gathered so far, flagged incomplete with the cause, so a partial list is never mistaken for the whole.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers with a non-2xx response — including a single read whose proposal id does not exist (typically `404`) — the system reports that the read failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own; the shared error handling (API Error Extraction, 015) classifies the status — a generic API-error outcome, or the permission (`401`/`403`) and rate-limit (`429`) outcomes it already distinguishes — and the command surfaces whichever results.
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** see which proposals are in flight in a circle before I act,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want to** list proposals, narrowed to a circle or status, with one command.

**In order to** read a proposal's full detail — its changes, how the circle has responded, and what I can do with it next,
**as a** practitioner operating the governance write flow,
**I want to** fetch a single proposal by its `prp_` id.

**In order to** check whether my own proposal has cleared the response window,
**as a** proposer,
**I want to** filter the list to proposals I created, or to a status, or to recently-proposed ones.

**In order to** trust I am seeing every proposal that matters,
**as a** practitioner in a busy circle,
**I want** the list to walk to completion, or to tell me plainly when it is incomplete.

---

## Non-Behaviors

- The system must not create, advance (`propose`), withdraw, or respond to proposals. **Why**: Proposal Reads owns the read pair alone, matching the project's one-capability-per-spec pattern. Creation is Proposal Creation (the sibling write spec); advancing is Advance to Circulation; withdrawing is Withdraw Proposal; responding is Response Recording. A read surface that mutated would fork those write contracts — all of which are Premium-gated, while the reads are not.
- The system must not act on `available_transitions` — it surfaces the list the server returned but never auto-invokes `propose`/`withdraw` or constructs a transition request. **Why**: `available_transitions` is descriptive data about what the caller *may* do; invoking a transition is a write owned by the advance/withdraw specs. The read produces the names; the operator (or a future write command) decides.
- The system must not expose per-person response attribution, nor reconstruct who responded which way from any other field. **Why**: the API exposes `response_summary` as aggregate counts only — an intentional anti-political-pressure design decision. The CLI is a faithful surface of what the API returns; it must not manufacture attribution the API withholds.
- The system must not interpret the `changes[]` payload, validate change types, or render them through any per-type schema. **Why**: `ProposalChange` is `type` plus free-form properties with no per-type schema in the API; the CLI passes changes through as the server supplied them. Typed change construction is a deferred, separate problem (Unguided Change Construction).
- The system must not treat `--status` (or any other filter) as anything but a list narrowing parameter, nor set/recompute a proposal's status. **Why**: status is server-owned AASM state; on a read it is only a query filter, never a write.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.
- The system must not interpret, summarize, or advise on the proposals it reads. **Why**: the CLI is a faithful API surface, not a Holacracy facilitator (VISION Exclusion 1); it produces the proposal data and lets the operator reason about it.

---

## Integration Boundaries

- **Glassfrog API v5 (`GET /proposals`, `GET /proposals/{id}`)**: the system reads proposals through these endpoints. The list accepts optional `status` (enum), `role_id`, `proposer_id`, `proposed_after`, and `accepted_after` query filters and is paginated; the single read returns full proposal detail by `prp_` id, including `changes[]`, `response_summary`, and `available_transitions`. Data flows inbound (proposals in, nothing written). Neither read is Premium-gated. A `404` means the proposal id does not exist or is not visible to the caller. When an endpoint is unreachable or answers non-2xx, the command surfaces a transport or read failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007) / Pagination (016)**: the command builds the request and hands it to these seams to authenticate, execute, and (for the list) walk pages; it does not re-implement transport, auth, or paging.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the proposal data as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success and the non-2xx / transport / not-authenticated outcomes to exit codes and messages through these seams.
- **Proposal Creation (sibling write spec) — concurrent**: opens the `proposal create` verb and the `glassfrog.Proposal` model shape these reads surface and reuse. The two have no runtime dependency; they share the `proposal` command grammar and the response model.

---

## Driving Scenarios

### Happy path

**Scenario: List the proposals visible to the caller**
Given a valid stored credential and two proposals visible to the caller
When the user runs `glassfrog proposal list`
Then the system reads `GET /proposals`
And produces both proposals as a list result
And exits successfully.

**Scenario: Read a single proposal with full detail**
Given a valid stored credential and an existing `prp_` id
When the user runs `glassfrog proposal get <prp-id>`
Then the system reads `GET /proposals/{id}`
And produces the single proposal including its `changes[]`, aggregate `response_summary`, and `available_transitions`
And exits successfully.

**Scenario: Narrow the list to a circle and a status**
Given proposals across several circles and statuses
When the user runs `glassfrog proposal list --role-id <role-id> --status proposed_outside_meeting`
Then `proposed_outside_meeting` is accepted as a supported status
And the system sends `role_id` and `status=proposed_outside_meeting` on `GET /proposals`
And produces only the matching proposals as a list result.

### Error scenarios

**Scenario: Proposal id does not exist**
Given a valid stored credential and a `prp_` id that no visible proposal has
When the user runs `glassfrog proposal get <prp-id>`
Then the API answers `404`
And the system reports the read failed, naming the HTTP status
And exits non-zero.

**Scenario: No usable credential**
Given no stored credential and none in the environment
When the user runs `glassfrog proposal list`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: No proposals are visible**
Given a valid stored credential and no proposals visible to the caller
When the user runs `glassfrog proposal list`
Then the system produces an empty list result
And exits successfully.

**Scenario: Unsupported status value is rejected before any request**
Given a valid stored credential
When the user runs `glassfrog proposal list --status open`
Then the system rejects it as a usage error, naming the unsupported value and the supported set (`draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts`)
And issues no request
And exits with the usage-error code.

**Scenario: List filter on the single read is rejected**
Given a valid stored credential
When the user runs `glassfrog proposal get <prp-id> --status draft`
Then the system rejects the invocation as a usage error before calling the API
And exits with the usage-error code.

**Scenario: Paginated list with first-page opt-out**
Given more visible proposals than fit on one page
When the user runs `glassfrog proposal list` with the first-page opt-out flag
Then the system makes a single page request
And produces the first page flagged incomplete with a clear "more exist" signal.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The read surface never reaches into the write verbs**
Given the implemented commands
When a reviewer exercises `glassfrog proposal` for any create-, propose-, withdraw-, or respond-style behavior under `list` or `get`
Then no such behavior is present — `list` and `get` only read, and `available_transitions` is surfaced but never invoked
And the help text for the read verbs does not advertise mutation.

**Scenario: No per-person response attribution is reconstructed**
Given a single proposal read
When the rendered result is inspected in every format
Then only the aggregate `response_summary` counts appear — no field names or attributes a response to an individual.

**Scenario: An unsupported status costs no request**
Given a `--status` value outside the proposal status vocabulary
When the command runs
Then the system rejects it as a usage error before assembling the connection or sending a request
And a transport tripwire confirms no request was issued.

**Scenario: Output is structured, not pre-rendered**
Given any successful proposal read
When the result reaches Output Format Selection (020)
Then the command supplied structured proposal data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **`proposal list` / `proposal get` verb grammar**: because the proposal surface is verb-rich (create / propose / withdraw / respond), the reads use the `proposal <verb>` grammar rather than a plural/singular noun pair, mirroring how Tension Capture (042) / Tension Reads (043) handled a verb-rich noun and keeping the noun unambiguous for its sibling write verbs. The concurrently-specified Proposal Creation opens `proposal create`; should that sibling settle on a different spelling, this spec follows it. (Grounded in the 042/043 precedent and the verb-rich proposal domain.)
- **`Proposal` model is shared, not re-defined**: the proposal shape (id, status, `tension_id`, `circle_id`, `proposer_id`, timestamps, `changes[]`, `response_summary`, `expected`/`received_response_count`, `available_transitions`) is established once in `internal/glassfrog` and reused by both the reads and Proposal Creation — the list returns many, `getProposal` returns full detail of one — so no second model is needed. (Mirrors the grow-not-duplicate model reuse of 011/043 and the DECISIONS.md "resource models live in `internal/glassfrog`" rule.)
- **Status vocabulary tracks the spec enum**: `--status` is validated against the proposal status set (`draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts`) before any request, the values `listProposals` accepts. The accepted set tracks the vendored spec (`spec/glassfrog-api-v5.yaml`). (Reflects the spec's documented `?status=` enum, including `draft_with_conflicts`.)
- **First-page opt-out flag is the shared one**: the list reuses the same first-page opt-out flag and "more exist" signal established by the earlier list reads (016 / 025 / 026 / 033 / 034 / 038 / 043), not a new per-command flag. (Consistency across every list surface in the CLI.)
- **[ASSUMED] Filter values other than status are not validated client-side**: `--role-id`, `--proposer-id`, `--proposed-after`, and `--accepted-after` are passed through as supplied and the server rejects a malformed id or timestamp, rather than the CLI enforcing the `^role_…` / `^per_…` patterns or an RFC 3339 timestamp shape locally. (Mirrors how Tension Reads (043) leaves id-shape validation to the server; only the closed `--status` enum is checked client-side.)
- **[ASSUMED] Proposal-id format is not validated client-side**: `proposal get` requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing `^prp_…` locally. (Mirrors 038/042/043.)

---

## Ambiguity Warnings

_None — the feature follows the established 025/026/033/034/038/043 read pattern (list + read-by-id, status-validated list filter, full-walk pagination with first-page opt-out, structured output). The one proposal-specific shape question — command grammar — is settled by the 042/043 verb-grammar precedent and the verb-rich proposal domain, captured as an Assumption with its dependency on the concurrent Proposal Creation spec made explicit rather than left open._
