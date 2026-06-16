# Specification: Withdraw Proposal

**Feature**: 059-withdraw-proposal
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Withdraw Proposal is the **`withdraw` transition for a governance proposal** — the command that pulls a circulating proposal back off the consent window and returns it to the bench for re-editing: `glassfrog proposal withdraw <prp-id>` → `POST /proposals/{proposal_id}/withdraw` → `withdrawProposal`. Where Advance to Circulation (057) moves a `draft` into circulation, this command is its mirror: it invokes the `withdraw` transition the proposal's `available_transitions` advertises, returning a `proposed_outside_meeting` or `escalated` proposal to `draft`. It is the recovery path off the happy path to `accepted` — the way a proposer re-opens a proposal that is circulating so it can be amended and re-proposed.

It attaches one more leaf — `withdraw` — to the `proposal` command family the proposal read/creation/advance siblings open, continuing the `proposal <verb>` grammar (`proposal propose` from 057, `proposal respond` from 058) rather than introducing a new top-level noun. It sits on the proven write chain rather than rebuilding it: it hands the bodyless request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, renders its result through **Output Format Selection (020)**, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**.

Three facts of the endpoint shape its behavior, and they are the same three that shape its `propose` mirror. First, the transition is **Premium-gated** — the API returns `403` when async proposals are not enabled; the command surfaces that `403` through the shared error handling like any other refusal, because distinguishing a plan-gate `403` from a permission `403` is a separate capability (Plan-Limit Signalling), not this command's job. Second, the transition is **server-authorized** — it is allowed only when `withdraw` appears in the proposal's `available_transitions` (for the proposer and admins), and the server enforces that with a `422` when it does not; the command issues the `POST` and lets the server decide rather than pre-reading the proposal to gate the call itself. Third, a successful withdraw returns **`200` with the withdrawn `Proposal`** — now back in `draft`, with its `proposed_at` and `response_deadline` cleared and its prior `proposal_responses` deleted server-side — so the command renders the proposal the server returned and synthesizes nothing.

It is deliberately scoped to *withdrawing one proposal*: it sends no request body, it never treats a `404` or `422` as a success end-state (a withdraw has no idempotent re-run — a proposal already in `draft` no longer offers `withdraw`), it sends no `If-Match` concurrency guard (withdraw is a transition, not a field edit), it does not interpret or warn about the destructive side effects the withdraw triggers (responses deleted, timestamps cleared), and it does not re-edit, re-propose, or otherwise chain past the single transition.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog proposal withdraw <prp-id>`, the system returns the named circulating proposal to `draft` and produces the resulting proposal as its result.
- When the user omits the required `<prp-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- The withdraw takes no body or field flags — it is a pure transition, with nothing to edit. Only the root persistent flags (`--base-url`, `--output`/`-o`) inherited from the command tree apply.

### Output

- When the withdraw succeeds (the API answers `200`), the server returns the withdrawn `Proposal` — now back in `draft`, with its `proposed_at` and `response_deadline` cleared and its prior responses deleted, and with updated `available_transitions`. The system produces that returned proposal as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- The system surfaces the proposal exactly as the server returned it — it neither synthesizes fields the response did not carry nor narrates the side effects the withdraw triggered (responses being deleted, timestamps being cleared, responders needing to re-respond if the proposal is later re-proposed); those consequences are server-owned and visible only through the returned data.
- When the withdraw succeeds, the system exits successfully.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers `422` — the transition is not allowed (the proposal is already in `draft`, or `withdraw` is not in its `available_transitions`) — the system reports that the withdraw failed, naming the HTTP status, and exits non-zero. A `422` is a genuine failure, never a success: a withdraw has no idempotent end-state, so withdrawing an already-draft proposal is a real refusal the operator must see.
- When the API answers `404` — the proposal id does not exist or is not visible to the caller — the system reports that the withdraw failed, naming the HTTP status, and exits non-zero. The withdraw does not treat `404` as success.
- When the API answers any other non-2xx response — including the Premium `403` returned when async proposals are not enabled, and the permission (`401`) and rate-limit (`429`) statuses — the system reports that the withdraw failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own: the shared error handling (API Error Extraction, 015) classifies the status, and the command surfaces whichever outcome results. It does not give the Premium `403` any special "not available on your plan" treatment — that is a separate capability (Plan-Limit Signalling).
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** re-open a proposal I am circulating so I can amend it before it is accepted,
**as a** practitioner operating the governance write flow,
**I want to** withdraw it back to draft by id with one command.

**In order to** roll a circulating proposal back from an automated pipeline,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want** the withdraw to produce the resulting draft proposal as structured data I can parse, including its now-cleared deadline and updated transitions.

**In order to** trust that a withdraw only happens when the server allows it,
**as a** practitioner,
**I want** the command to issue the transition and surface the server's refusal plainly rather than guessing client-side whether it is permitted.

---

## Non-Behaviors

- The system must not create, list, get, advance (`propose`), or respond to proposals. **Why**: Withdraw Proposal owns the `withdraw` transition alone, matching the project's one-capability-per-spec pattern. Creation is Proposal Creation; the reads are Proposal Reads (056); advancing a draft into circulation is Advance to Circulation (057); recording a consent response is Response Recording (058). A command that strayed into those would fork their contracts.
- The system must not re-edit the proposal's `changes[]` or re-propose it after withdrawing. **Why**: withdraw is a single transition that returns the proposal to `draft`; editing the draft and re-circulating it are separate acts (Proposal Creation's edit surface and Advance to Circulation, 057). Chaining them here would couple distinct capabilities and hide the intermediate `draft` state from the operator.
- The system must not pre-read the proposal to inspect `available_transitions` or otherwise pre-validate the transition client-side before issuing the `POST`. **Why**: `available_transitions` is server-owned and time-sensitive; the server enforces the rule with a `422`. A client-side gate would fork that authority, add a read round-trip, and risk acting on a stale snapshot. The command issues the transition and lets the server decide.
- The system must not treat a `404` or a `422` as a success outcome. **Why**: unlike a soft-delete, a withdraw has no idempotent end-state — withdrawing an already-draft proposal is a genuine `422` (the transition is no longer offered), and a `404` is a real missing-or-invisible proposal. Both are failures the operator must see, not success end-states to absorb.
- The system must not give the Premium `403` any plan-aware "not available on your plan" messaging. **Why**: distinguishing a plan-gate `403` from a permission `403` is Plan-Limit Signalling, a separate capability; here the `403` routes through the shared error handling like any other refusal.
- The system must not send an `If-Match` precondition or otherwise guard against concurrent edits. **Why**: `withdraw` is a transition with no `If-Match` parameter; optimistic concurrency (Guarded Writes, 053) applies to field edits, not transitions. The withdraw sends exactly the bodyless `POST` the endpoint defines.
- The system must not interpret, re-describe, warn about, or act on the destructive side effects the withdraw triggers (the deletion of existing `proposal_responses`, the clearing of `proposed_at` and `response_deadline`). **Why**: those are server-owned consequences reflected in the returned proposal; the CLI is a faithful surface of what the API returns, not a Holacracy facilitator (VISION Exclusion 1).
- The system must not prompt for confirmation or require a `--force`/`--yes` flag before withdrawing, even though the withdraw discards existing responses. **Why**: the CLI is built for non-interactive, agent-driven use; an interactive guard would break that contract. An operator-layer confirmation for destructive writes is a separate capability (Write-Safety Guardrail), not this command's job.
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`POST /proposals/{proposal_id}/withdraw`, `withdrawProposal`)**: the system returns a circulating proposal to `draft`. The path `id` is the proposal's `prp_` id; the request carries no body. A `200` returns the withdrawn `Proposal` (now `draft`, with `proposed_at` and `response_deadline` cleared, prior `proposal_responses` deleted, and updated `available_transitions`). The transition is **Premium-gated** — `403` when async proposals are not enabled. A `404` means the proposal does not exist or is not visible; a `422` means the transition is not allowed (already `draft`, or `withdraw` not in `available_transitions`). Data flows: nothing written in the body, the withdrawn proposal returned. When the endpoint is unreachable, the command surfaces a transport failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the bodyless `POST`, hands it to these seams to authenticate and execute, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the returned proposal as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success (`200`), the non-2xx (`403`/`404`/`422`/`401`/`429`), transport, and not-authenticated outcomes to exit codes and messages through these seams.
- **Advance to Circulation (057) / Proposal Reads (056) / Proposal Creation (sibling)**: establish the `proposal` command family and the `glassfrog.Proposal` model this command attaches to and renders. The `available_transitions` the reads surface is exactly what gates this transition server-side — a read shows `withdraw` is offered; this command invokes it. Withdraw is the explicit mirror of 057's `propose`, returning to `draft` what `propose` advanced into circulation.
- **Plan-Limit Signalling (separate problem)**: the Premium `403` this command can receive is where dedicated plan-gate signalling will eventually attach. Until that capability lands, the `403` routes through API Error Extraction (015) like any other refusal; this command adds no plan-aware behavior of its own.

---

## Driving Scenarios

### Happy path

**Scenario: Withdraw a circulating proposal back to draft**
Given a stored credential and an existing `proposed_outside_meeting` proposal whose `available_transitions` include `withdraw`
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the system issues `POST /proposals/prp_<id>/withdraw` with no body
And the API answers `200` with the proposal now back in `draft`
And the system produces that returned proposal as its result
And exits successfully.

**Scenario: Withdrawn proposal rendered as JSON**
Given a stored credential and a circulating proposal that can be withdrawn
When the user runs `glassfrog proposal withdraw prp_<id> -o json`
Then the system withdraws the proposal
And Output Format Selection renders the returned proposal as JSON
And exits successfully.

**Scenario: The result reflects the cleared deadline and updated transitions**
Given a stored credential and a circulating proposal that can be withdrawn
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the rendered result shows the proposal in `draft` with `proposed_at` and `response_deadline` cleared and the `available_transitions` updated exactly as the server returned them
And the command neither synthesizes those fields nor narrates the responses the withdraw deleted.

### Error scenarios

**Scenario: Transition not allowed is a failure**
Given a stored credential and a proposal for which `withdraw` is not currently allowed (already a `draft`, or `withdraw` not in `available_transitions`)
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the API answers `422`
And the system reports that the withdraw failed, naming the HTTP status
And exits non-zero.

**Scenario: Proposal id does not exist**
Given a stored credential and a `prp_` id that no visible proposal has
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the API answers `404`
And the system reports that the withdraw failed, naming the HTTP status
And exits non-zero.

**Scenario: No credential surfaces the not-authenticated outcome**
Given no usable token is available
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Missing proposal id is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal withdraw` with no positional id
Then the system rejects the invocation as a usage error naming the required `<prp-id>`
And sends no request.

**Scenario: Premium not enabled surfaces a plain refusal**
Given a stored credential on a plan without async proposals enabled
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the API answers `403`
And the system reports that the withdraw failed, naming the HTTP status, through the shared error handling — with no special "not available on your plan" treatment
And exits non-zero.

**Scenario: A transport failure surfaces network-unavailable**
Given a stored credential and an unreachable API endpoint
When the user runs `glassfrog proposal withdraw prp_<id>`
Then the system surfaces the transport failure by name
And exits non-zero with the network-unavailable outcome.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The withdraw issues exactly one transition request and reads nothing first**
Given the `proposal withdraw` command
When a reviewer traces a single invocation
Then the command issues exactly one `POST /proposals/{id}/withdraw` and no prior `GET` to inspect `available_transitions` — it does not pre-gate the transition client-side.

**Scenario: A `404` and a `422` are real failures**
Given a `prp_` id the API answers `404` for, and a proposal the API answers `422` for
When the command runs against each
Then neither produces a success result, both exit non-zero, and the command treats no non-2xx status as a success end-state.

**Scenario: The result is the server's proposal, unembellished**
Given a successful withdraw (`200`)
When a reviewer inspects the rendered result in every format
Then it carries only fields the server returned (now-`draft` status, cleared `proposed_at`/`response_deadline`, updated `available_transitions`, `changes`) and fabricates no side-effect narration about deleted responses.

**Scenario: Output is structured, not pre-rendered**
Given any successful withdraw
When the result reaches Output Format Selection (020)
Then the command supplied structured proposal data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **[ASSUMED] Command grammar `proposal withdraw <prp-id>`**: the transition uses the API/lifecycle verb name `withdraw` directly — the exact string the proposal's `available_transitions` lists — continuing the `proposal <verb>` grammar Proposal Reads (056) opened and Advance to Circulation (057) extended. Default: `withdraw`, to mirror `available_transitions` exactly; any alternative spelling is open and will be pinned at the interface stage alongside the proposal siblings, which this spec then follows. (Grounded in the 057 verb-grammar precedent and the API transition name.)
- **`prp_` id form not validated client-side**: `proposal withdraw` requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing `^prp_[0-9a-f]{32}$` locally. (Mirrors 057/056/045.)
- **`Proposal` model is shared, not re-defined**: the `200` returns the same `Proposal` shape Proposal Reads (056) surface; the command renders it through the shared `internal/glassfrog` model, adding no second model. (Mirrors the grow-not-duplicate model reuse across the project.)
- **Bodyless `POST`**: `withdraw` carries no request body — the endpoint defines none — only the path id. (Reflects the `withdrawProposal` operation having no `requestBody`.)
- **`200` carries the withdrawn proposal**: the success renders the server payload, with no client-side synthesis, mirroring 057's advance result. (Reflects the documented `200` returning `data: Proposal`.)

---

## Ambiguity Warnings

_None — Withdraw Proposal is the direct mirror of Advance to Circulation (057), and inherits its settled decisions: `404`/`422` as real failures, server-side transition authorization with no client pre-check, the Premium `403` routed through the shared error handling rather than given bespoke plan messaging, and a bodyless transition with no `If-Match` guard. The one withdraw-specific concern — that the transition is destructive (it deletes existing responses and clears timestamps) — is settled by treating those as server-owned side effects faithfully surfaced in the returned proposal, with the non-interactive no-confirmation stance held consistent with 057 and the operator-layer guardrail explicitly deferred. The command-verb spelling is settled by the 057 verb-grammar precedent and captured as an Assumption with its interface-stage dependency made explicit rather than left open._
