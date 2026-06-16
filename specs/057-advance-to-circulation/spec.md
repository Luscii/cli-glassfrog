# Specification: Advance to Circulation

**Feature**: 057-advance-to-circulation
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Advance to Circulation is the **`propose` transition for a governance proposal** — the command that moves a draft proposal off the bench and into the consent window: `glassfrog proposal propose <prp-id>` → `POST /proposals/{proposal_id}/propose` → `proposeProposal`. Where Proposal Creation drafts a proposal and Proposal Reads (056) surface it with its `available_transitions`, this command invokes the `propose` transition those reads advertise — advancing a `draft` to `proposed_outside_meeting`, the shortcut-path circulation state. The advance is the first server-driven step of the lifecycle (`draft → proposed_outside_meeting → accepted`): it is the action that actually starts the clock the rest of the write flow runs against.

It attaches one more leaf — `propose` — to the `proposal` command family the proposal read/creation siblings open, continuing the `proposal <verb>` grammar (`proposal list` / `proposal get` from 056) rather than introducing a new top-level noun. It sits on the proven write chain rather than rebuilding it: it hands the bodyless request to **Request Execution (010)**, reads identity through **Request Authentication (007)**, renders its result through **Output Format Selection (020)**, and maps outcomes through **Exit-Code Convention (004)** and **API Error Extraction (015)**.

Three facts of the endpoint shape its behavior. First, the transition is **Premium-gated** — the API returns `403` when async proposals are not enabled; the command surfaces that `403` through the shared error handling like any other refusal, because distinguishing a plan-gate `403` from a permission `403` is a separate capability (Plan-Limit Signalling), not this command's job. Second, the transition is **server-authorized** — it is allowed only when `propose` appears in the proposal's `available_transitions`, and the server enforces that with a `422` when it does not; the command issues the `POST` and lets the server decide rather than pre-reading the proposal to gate the call itself. Third, a successful advance returns **`200` with the advanced `Proposal`** — unlike a bodyless soft-delete, there is a real payload (now in `proposed_outside_meeting`, carrying the server-set `response_deadline`, the proposer's auto-recorded implicit `no_objection`, and updated `available_transitions`), so the command renders the proposal the server returned and synthesizes nothing.

It is deliberately scoped to *advancing one proposal*: it sends no request body, it does not choose the backend dispatch path (shortcut / meeting-queue / CL-forward — the org's config decides), it never treats a `404` or `422` as a success end-state (an advance has no idempotent re-run), it sends no `If-Match` concurrency guard (propose is a transition, not a field edit), and it does not interpret the side effects the advance triggers server-side.

---

## Behavioral Accord

### Invocation

- When the user runs `glassfrog proposal propose <prp-id>`, the system advances the named draft proposal into circulation and produces the advanced proposal as its result.
- When the user omits the required `<prp-id>` positional, passes more than one positional id, or passes an unknown flag, the system rejects the invocation as a usage error and calls no API.
- The advance takes no body or field flags — it is a pure transition, with nothing to edit. Only the root persistent flags (`--base-url`, `--output`/`-o`) inherited from the command tree apply.

### Output

- When the advance succeeds (the API answers `200`), the server returns the advanced `Proposal` — now in `proposed_outside_meeting`, carrying its server-set `response_deadline`, the auto-recorded implicit `no_objection` reflected in `response_summary`, and updated `available_transitions`. The system produces that returned proposal as its result and lets Output Format Selection (020) render it in the effective format (`full` / `compact` / `json` / `yaml`); it neither fixes raw API JSON as its default nor defines its own format flag.
- The system surfaces the proposal exactly as the server returned it — it neither synthesizes fields the response did not carry nor narrates the side effects the advance triggered (notifications firing, the deadline being computed, the implicit response being recorded); those consequences are server-owned and visible only through the returned data.
- When the advance succeeds, the system exits successfully.

### Failure

- When no usable token is available, the system surfaces the authentication fail-safe's refusal and exits non-zero with a not-authenticated outcome, reusing the shared not-authenticated message and pointing the operator at how to store a credential.
- When the request cannot reach or complete at the wire (connection, DNS, TLS, timeout), the system surfaces the transport failure by name and exits non-zero with the network-unavailable outcome.
- When the API answers `422` — the transition is not allowed (the proposal is not in `draft`, or `propose` is not in its `available_transitions`) — the system reports that the advance failed, naming the HTTP status, and exits non-zero. A `422` is a genuine failure, never a success: an advance has no idempotent end-state, so re-proposing an already-circulating proposal is a real refusal the operator must see.
- When the API answers `404` — the proposal id does not exist or is not visible to the caller — the system reports that the advance failed, naming the HTTP status, and exits non-zero. Unlike a soft-delete, the advance does not treat `404` as success.
- When the API answers any other non-2xx response — including the Premium `403` returned when async proposals are not enabled, and the permission (`401`) and rate-limit (`429`) statuses — the system reports that the advance failed, naming the HTTP status, and exits non-zero. The command adds no interpretation of its own: the shared error handling (API Error Extraction, 015) classifies the status, and the command surfaces whichever outcome results. It does not give the Premium `403` any special "not available on your plan" treatment — that is a separate capability (Plan-Limit Signalling).
- Whatever the failure, the message names both what went wrong and a concrete next step (Action Transparency), and never includes the token.

---

## User Scenarios

**In order to** start the consent window on a proposal I have drafted,
**as a** practitioner operating the governance write flow,
**I want to** advance it into circulation by id with one command.

**In order to** move a created proposal toward acceptance from an automated pipeline,
**as an** AI agent operating the CLI on a practitioner's behalf,
**I want** the advance to produce the resulting proposal as structured data I can parse, including its new status and response deadline.

**In order to** trust that an advance only happens when the server allows it,
**as a** practitioner,
**I want** the command to issue the transition and surface the server's refusal plainly rather than guessing client-side whether it is permitted.

---

## Non-Behaviors

- The system must not create, list, get, withdraw, or respond to proposals. **Why**: Advance to Circulation owns the `propose` transition alone, matching the project's one-capability-per-spec pattern. Creation is Proposal Creation; the reads are Proposal Reads (056); returning a circulating proposal to draft is Withdraw Proposal; recording a consent response is Response Recording. A command that strayed into those would fork their contracts.
- The system must not pre-read the proposal to inspect `available_transitions` or otherwise pre-validate the transition client-side before issuing the `POST`. **Why**: `available_transitions` is server-owned and time-sensitive; the server enforces the rule with a `422`. A client-side gate would fork that authority, add a read round-trip, and risk acting on a stale snapshot. The command issues the transition and lets the server decide.
- The system must not choose or influence the backend dispatch path (shortcut / meeting-queue / CL-forward / etc.). **Why**: the API routes `propose` to the appropriate internal path based on circle and org config — callers do not choose the path. The command sends `propose` and accepts whatever path the backend selects.
- The system must not treat a `404` or a `422` as a success outcome. **Why**: unlike a soft-delete, an advance has no idempotent end-state — re-proposing a circulating proposal is a genuine `422` (the transition is no longer offered), and a `404` is a real missing-or-invisible proposal. Both are failures the operator must see, not success end-states to absorb.
- The system must not give the Premium `403` any plan-aware "not available on your plan" messaging. **Why**: distinguishing a plan-gate `403` from a permission `403` is Plan-Limit Signalling, a separate capability; here the `403` routes through the shared error handling like any other refusal.
- The system must not send an `If-Match` precondition or otherwise guard against concurrent edits. **Why**: `propose` is a transition with no `If-Match` parameter; optimistic concurrency (Guarded Writes, 053) applies to field edits, not transitions. The advance sends exactly the bodyless `POST` the endpoint defines.
- The system must not interpret, re-describe, or act on the side effects the advance triggers (notifications, the computed `response_deadline`, the implicit `no_objection`). **Why**: those are server-owned consequences reflected in the returned proposal; the CLI is a faithful surface of what the API returns, not a Holacracy facilitator (VISION Exclusion 1).
- The system must not prompt for confirmation or require a `--force`/`--yes` flag before advancing. **Why**: the CLI is built for non-interactive, agent-driven use; an interactive guard would break that contract, and the advance is reversible in spirit (Withdraw Proposal returns a circulating proposal to draft).
- The system must not emit raw API JSON as a fixed default, nor define its own output-format flag. **Why**: Output Format Selection (020) resolves the format and dispatches to the 018/019 renderers; a private flag here would fork that contract.
- The system must not resolve the base URL or token, attach the `X-Auth-Token` header, decide the no-token fail-safe, type a non-2xx response, or choose its own exit codes. **Why**: Base URL Resolution (008), Credential Discovery (005), Request Authentication (007), API Error Extraction (015), and Exit-Code Convention (004) own those; a second path here would drift from their contracts.

---

## Integration Boundaries

- **Glassfrog API v5 (`POST /proposals/{proposal_id}/propose`, `proposeProposal`)**: the system advances a `draft` proposal into circulation. The path `id` is the proposal's `prp_` id; the request carries no body. A `200` returns the advanced `Proposal` (now `proposed_outside_meeting`, with `response_deadline` set, the proposer's implicit `no_objection` recorded, and updated `available_transitions`). The transition is **Premium-gated** — `403` when async proposals are not enabled. A `404` means the proposal does not exist or is not visible; a `422` means the transition is not allowed (not a `draft`, or `propose` not in `available_transitions`). Data flows: nothing written in the body, the advanced proposal returned. When the endpoint is unreachable, the command surfaces a transport failure and exits non-zero.
- **Request Execution (010) / Request Authentication (007)**: the command builds the bodyless `POST`, hands it to these seams to authenticate and execute, and does not re-implement transport or auth.
- **Output Format Selection (020) / Structured Serialization (018) / Templated Human Rendering (019)**: receives the returned proposal as the command's result and renders it in the effective format. The command produces structured data, never pre-rendered output.
- **Exit-Code Convention (004) / API Error Extraction (015)**: the command maps the success (`200`), the non-2xx (`403`/`404`/`422`/`401`/`429`), transport, and not-authenticated outcomes to exit codes and messages through these seams.
- **Proposal Reads (056) / Proposal Creation (sibling)**: establish the `proposal` command family and the `glassfrog.Proposal` model this command attaches to and renders. The `available_transitions` the reads surface is exactly what gates this transition server-side — a read shows `propose` is offered; this command invokes it.
- **Plan-Limit Signalling (separate problem)**: the Premium `403` this command can receive is where dedicated plan-gate signalling will eventually attach. Until that capability lands, the `403` routes through API Error Extraction (015) like any other refusal; this command adds no plan-aware behavior of its own.

---

## Driving Scenarios

### Happy path

**Scenario: Advance a draft proposal into circulation**
Given a stored credential and an existing `draft` proposal whose `available_transitions` include `propose`
When the user runs `glassfrog proposal propose prp_<id>`
Then the system issues `POST /proposals/prp_<id>/propose` with no body
And the API answers `200` with the advanced proposal now in `proposed_outside_meeting`
And the system produces that returned proposal as its result
And exits successfully.

**Scenario: Advanced proposal rendered as JSON**
Given a stored credential and a `draft` proposal that can be proposed
When the user runs `glassfrog proposal propose prp_<id> -o json`
Then the system advances the proposal
And Output Format Selection renders the returned proposal as JSON
And exits successfully.

**Scenario: The result reflects the server-set deadline and implicit response**
Given a stored credential and a `draft` proposal that can be proposed
When the user runs `glassfrog proposal propose prp_<id>`
Then the rendered result carries the `response_deadline`, the `response_summary` reflecting the proposer's implicit `no_objection`, and the updated `available_transitions` exactly as the server returned them
And the command neither synthesizes those fields nor narrates the notifications the advance triggered.

### Error scenarios

**Scenario: Transition not allowed is a failure**
Given a stored credential and a proposal for which `propose` is not currently allowed (already circulating, or not a `draft`)
When the user runs `glassfrog proposal propose prp_<id>`
Then the API answers `422`
And the system reports that the advance failed, naming the HTTP status
And exits non-zero.

**Scenario: Proposal id does not exist**
Given a stored credential and a `prp_` id that no visible proposal has
When the user runs `glassfrog proposal propose prp_<id>`
Then the API answers `404`
And the system reports that the advance failed, naming the HTTP status
And exits non-zero.

**Scenario: No credential surfaces the not-authenticated outcome**
Given no usable token is available
When the user runs `glassfrog proposal propose prp_<id>`
Then the system surfaces the not-authenticated refusal without calling the API
And exits non-zero
And the message points the operator at how to store a credential.

### Edge cases

**Scenario: Missing proposal id is rejected before any request**
Given a stored credential
When the user runs `glassfrog proposal propose` with no positional id
Then the system rejects the invocation as a usage error naming the required `<prp-id>`
And sends no request.

**Scenario: Premium not enabled surfaces a plain refusal**
Given a stored credential on a plan without async proposals enabled
When the user runs `glassfrog proposal propose prp_<id>`
Then the API answers `403`
And the system reports that the advance failed, naming the HTTP status, through the shared error handling — with no special "not available on your plan" treatment
And exits non-zero.

**Scenario: A transport failure surfaces network-unavailable**
Given a stored credential and an unreachable API endpoint
When the user runs `glassfrog proposal propose prp_<id>`
Then the system surfaces the transport failure by name
And exits non-zero with the network-unavailable outcome.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The advance issues exactly one transition request and reads nothing first**
Given the `proposal propose` command
When a reviewer traces a single invocation
Then the command issues exactly one `POST /proposals/{id}/propose` and no prior `GET` to inspect `available_transitions` — it does not pre-gate the transition client-side.

**Scenario: A `404` and a `422` are real failures**
Given a `prp_` id the API answers `404` for, and a proposal the API answers `422` for
When the command runs against each
Then neither produces a success result, both exit non-zero, and the command treats no non-2xx status as a success end-state.

**Scenario: The result is the server's proposal, unembellished**
Given a successful advance (`200`)
When a reviewer inspects the rendered result in every format
Then it carries only fields the server returned (status, `response_deadline`, `response_summary`, `available_transitions`, `changes`) and fabricates no side-effect narration.

**Scenario: Output is structured, not pre-rendered**
Given any successful advance
When the result reaches Output Format Selection (020)
Then the command supplied structured proposal data and defined no format flag of its own, so all four formats (`full` / `compact` / `json` / `yaml`) render from the same result.

---

## Assumptions

- **[ASSUMED] Command grammar `proposal propose <prp-id>`**: the transition uses the API/lifecycle verb name `propose` directly — the exact string the proposal's `available_transitions` lists — continuing the `proposal <verb>` grammar Proposal Reads (056) opened. The repetition (`proposal propose`) is acknowledged; an alternative spelling (e.g. `proposal advance`) is open and will be pinned at the interface stage alongside the proposal siblings, which this spec then follows. Default: `propose`, to mirror `available_transitions` exactly. (Grounded in the 056 verb-grammar precedent and the API transition name.)
- **`prp_` id form not validated client-side**: `proposal propose` requires exactly one positional id but lets the API reject an unknown or malformed id (typically `404`), rather than enforcing `^prp_[0-9a-f]{32}$` locally. (Mirrors 056/045.)
- **`Proposal` model is shared, not re-defined**: the `200` returns the same `Proposal` shape Proposal Reads (056) surface; the command renders it through the shared `internal/glassfrog` model, adding no second model. (Mirrors the grow-not-duplicate model reuse across the project.)
- **Bodyless `POST`**: `propose` carries no request body — the endpoint defines none — only the path id. (Reflects the `proposeProposal` operation having no `requestBody`.)
- **`200` carries the advanced proposal**: the success renders the server payload, with no client-side synthesis, distinguishing it from the bodyless-delete pattern (045) where the command had to synthesize a result. (Reflects the documented `200` returning `data: Proposal`.)

---

## Ambiguity Warnings

_None — the feature follows the established write-action pattern (045 Tension Discard as the closest analogue, minus the soft-delete idempotency twist and the body synthesis, since `propose` returns the advanced `Proposal`). The distinctive decisions — `404`/`422` as real failures, server-side transition authorization with no client pre-check, and the Premium `403` routed through the shared error handling rather than given bespoke plan messaging — are settled by the API spec and the deliberate separation of Plan-Limit Signalling. The one proposal-specific shape question, the command-verb spelling, is settled by the 056 verb-grammar precedent and captured as an Assumption with its interface-stage dependency made explicit rather than left open._
