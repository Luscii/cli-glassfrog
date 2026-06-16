# Plan: Proposal Reads

**Feature**: 056-proposal-reads
**Role**: Shaper
**Inputs**: `specs/056-proposal-reads/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md` (background); the sibling plans 043-tension-reads (the read-pair architecture: list + read-by-id, status validation, walked-list render), 042-tension-capture (the namespace-group + model + singular-render pattern), 038-role-projects (the plural/singular render pair); existing code in `internal/cli`, `internal/glassfrog`, `internal/render`, `internal/paging`, `internal/output`; the vendored `spec/glassfrog-api-v5.yaml` (`listProposals`, `getProposal`, the `Proposal` / `ProposalChange` schemas)

---

## System Architecture

Proposal Reads adds the **read pair for proposals** — architecturally the twin of Tension Reads (043): two thin cobra leaves that build a request, hand it to the proven read chain, and render the result. It introduces **no new transport, pagination, error, or output machinery** — every seam it needs is landed (010/008/007 client, 016 paging, 011/015 error chain, 020/018/019/035 output). What it *does* introduce, because **no proposal code exists yet**, is the **`proposal` command group**, the **`glassfrog.Proposal` model**, and **both** the plural (`proposals`) and singular (`proposal`) render paths — where 043 only added the plural because 042 had already landed the namespace, model, and singular render. Proposal Creation (the concurrently-specified sibling, BACKLOG #55) shares all three of these; whichever of the two lands first creates them and the other reuses them (the resolved-by-rebase coordination 043 documented for the 042→043 sequence — see Risks).

Two leaves, guard-registered (001), attached to a new `proposal` group command:

- **`glassfrog proposal list`** → `GET /proposals` (`listProposals`) — a **global, paginated list** of the proposals visible to the caller (no path id — unlike `tension list <role-id>`, the circle is an optional `--role-id` *filter*, not a positional). `cobra.NoArgs`. Walks to completion through `paging.All` (016) by default, with a `--first-page` opt-out and five optional server-side filters: `--status` (validated locally against the proposal status set, sent as `status`), `--role-id`, `--proposer-id`, `--proposed-after`, `--accepted-after` (all passed through, sent only when supplied).
- **`glassfrog proposal get <prp-id>`** → `GET /proposals/{id}` (`getProposal`) — a **single proposal** with full detail (`changes[]`, aggregate `response_summary`, `available_transitions`), decoded from a `{data: Proposal}` document. `cobra.ExactArgs(1)`.

Data flow per invocation (identical to 043's 038/033/034 lineage): resolve `--output` format first (020), validate the one closed-enum input (`--status`) fail-fast where the API would otherwise silently mislead, resolve the connection context once (`AssembleFromOS`, 009), build the `*apiclient.Client` (010/008/007), issue the request (list → `paging.All`; single → one `Execute`), and render by the resolved format (020): the **single** read emits the raw `{data: Proposal}` bytes (018) or the human `proposal` template (019); the **list** walks pages — for `json`/`yaml` aggregating each proposal's raw bytes into a `{data:[…]}` document via `aggregateRawData` (per-page `meta` dropped), for `full`/`compact` rendering the new `proposals` projection (019). Typed client errors route through the one shared `classifyClientError` chain (011/015) — **no new `Outcome` category, no new exit code** (notably, proposal *reads* are not Premium-gated, so unlike the write path they expect no plan-gate `403`).

---

## Architecture Decisions

### ADR-1: Create the `proposal` group command and add `list` / `get` as verb leaves (verb grammar; global list)

**Context**: The spec fixes the surface as `proposal list` / `proposal get` — the verb grammar (not a plural/singular noun pair) because the proposal surface is verb-rich (create / propose / withdraw / respond) and a bare `proposal <prp-id>` would collide with subcommand dispatch the moment `proposal create` exists, the exact cobra constraint 038 ADR-1 / 042 ADR-2 / 043 ADR-1 cite. Unlike Tension Reads, **no `proposal` group exists** — Proposal Creation (#55, concurrent) opens `proposal create`, but it has not landed in this base. And the list is **global** (`GET /proposals`, no path param), so `proposal list` takes **no positional**, where `tension list <role-id>` took one.

**Options considered**:
1. **Plural/singular noun pair** (`proposals` / `proposal <prp-id>`), as 038 did for projects. Rejected: the proposal surface is verb-rich; a bare `proposal <prp-id>` would collide with the `create`/`propose`/`withdraw`/`respond` subcommands the write specs add. The verb grammar is forced by the write verbs ahead.
2. **Two verb leaves under a new `proposal` group** — `proposal list` (`cobra.NoArgs`) + `proposal get <prp-id>` (`cobra.ExactArgs(1)`), with 056 creating the group via a `newProposalCommand(seam)` constructor (the `newTensionCommand` shape from 042). Chosen — conformance to the verb-grammar precedent (042/043), adapted to a group that does not yet exist.

**Decision**: Option 2. 056 introduces `newProposalCommand(seam)` building a non-runnable `proposal` group, and guard-registers two leaves under it: `list` (`NoArgs`) and `get` (`ExactArgs(1)`). The **list-only flags (`--status`, `--role-id`, `--proposer-id`, `--proposed-after`, `--accepted-after`, `--first-page`, `--per-page`) live only on `list`** — passing any to `get` is rejected by cobra's own unknown-flag handling as a `UsageError(2)` before assembly, so the spec's "filter on the single read is a usage error" needs no hand-rolled guard (the structural guard 034/038/042/043 rely on). Concurrent-sibling coordination: whichever of 055/056 lands first defines `newProposalCommand` and the group; the follower adds its leaves to the existing group (043's relationship to 042's `newTensionCommand`).

**Consequences**: A new top-level `proposal` group parents `list` and `get` (and, once 055 lands, `create`; later `propose`/`withdraw`/`respond`). The `--base-url` and `--output`/`-o` persistent root flags (011/020) are inherited by all leaves. `proposal list` is the CLI's **first global (non-role-scoped) paginated list** alongside the `me`-family self-service reads — `paging.All` over a path with no id is already supported (the request descriptor carries query, not path, params), so this is a usage difference, not a machinery change.

### ADR-2: Establish the `glassfrog.Proposal` model with a free-form `ProposalChange`; add the plural `proposals` and singular `proposal` render paths

**Context**: No `glassfrog.Proposal` exists. Both reads decode the **same** `Proposal` schema — the list returns many, `getProposal` returns full detail of one — and Proposal Creation (#55) returns the same shape on `201`. The schema (verified against `spec/glassfrog-api-v5.yaml`) carries `id`, `type`, `status`, nullable `tension_id` / `circle_id` / `proposer_id` / `proposed_at` / `response_deadline` / `accepted_at`, `changes[]`, an aggregate `response_summary` object (`total` / `no_objection` / `bring_to_meeting`), `expected_response_count`, `received_response_count`, `available_transitions[]` (`propose` / `withdraw`), and `created_at` / `updated_at`. The wrinkle is **`changes[]`**: `ProposalChange` is `type` plus **`additionalProperties: true`** — free-form per-type keys with no per-type schema. The codebase already has a free-form-decode precedent (`actions.go`: `Permissions map[string]any`).

**Options considered**:
1. **Model `changes[]` with a typed per-change struct hierarchy.** Rejected: the API has no per-type schema; the CLI passes changes through as supplied (spec non-behavior — no `changes[]` interpretation). Typed change construction is a deferred separate problem (Unguided Change Construction).
2. **`Proposal` with `Changes []ProposalChange`, where `ProposalChange` keeps `ID` + `Type` and captures the rest free-form (`map[string]any`, the `actions.go` precedent); add both render paths.** Chosen — one shared schema type (011 ADR-1 grow-not-duplicate), faithful pass-through, decodes enough for human render while structured output emits raw bytes verbatim.

**Decision**: Option 2. Add `internal/glassfrog/proposal.go` with the `Proposal` model (forward-compatible decoding, the `Tension`/`Project` shape) and a `ProposalChange` that decodes `ID`/`Type` plus a free-form remainder (the `map[string]any` precedent), and a small `ResponseSummary` struct (`Total`/`NoObjection`/`BringToMeeting`). Add **both** render paths to `internal/render` (the `ResourceProjects`/`ResourceProject` plural/singular shape): a `ProposalsView` + `ResourceProposals` + `proposals.full` / `proposals.compact`, and a `ProposalView` + `ResourceProposal` + `proposal.full` / `proposal.compact` — both registered in `builtinResources` so the exhaustiveness guard (the PR #10 registry shape) covers them. `proposal.full` shows id, status, the anchoring `tension_id` / `circle_id` / `proposer_id`, the lifecycle timestamps, `changes` **by type** (each change's `type`, the body rendered verbatim — CONSTITUTION VI, no truncation), the `response_summary` counts, `expected`/`received_response_count`, and `available_transitions`; `proposal.compact` is id + status + change-count + a one-line response summary. `proposals.full` is a per-proposal projection (id, status, proposer, change-count, response summary); `proposals.compact` is id + status + change-count, with an empty-set line (`no proposals`). Structured `json`/`yaml` reuses the landed machinery: the **single** read serializes the raw `{data: Proposal}` verbatim (018, via `output.RenderSuccess`); the **list** walks each page as `json.RawMessage` and emits the aggregated `{data:[…]}` document via `aggregateRawData`.

**Consequences**: A new model file and two new render resources (four templates). Proposal Creation reuses the model and the singular `proposal` render path unchanged — coordination: whichever lands first adds `proposal.go` and the singular render; if 055 lands first with a thinner singular render, 056 grows it to show `changes`/`response_summary`/`available_transitions` (grow-not-duplicate). Structured output is **faithful regardless of model completeness** — `aggregateRawData` / `RenderSuccess` pass the server bytes through, so a field the model omits still appears under `-o json`.

### ADR-3: Validate the closed-enum `--status` locally via a new `validateProposalStatus`; pass the four other filters and the id through

**Context**: 025 ADR-4 set the input-handling principle (applied by 038/042/043): validate closed-enum inputs locally where a wrong value makes the API silently return wrong results; pass free identifiers through where the API reports cleanly. `listProposals` offers a **closed `status` enum** — a **different set** from the tension and action/project vocabularies: `draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts`. Its other four filters — `role_id` / `proposer_id` (pattern-checked ids) and `proposed_after` / `accepted_after` (date-times) — are free values the server validates, and the `prp_` path id is a free identifier the endpoint answers with a clean `404`.

**Options considered**:
1. **Reuse `validateStatus` / `validateTensionStatus`.** Rejected: both cover different vocabularies (action/project; tension `unprocessed`/`processed`/`archived`); reusing either would accept invalid proposal statuses and reject valid ones — a correctness bug.
2. **Validate all five filters locally** (id patterns, RFC 3339 timestamps). Rejected: only `status` is a closed enum where a wrong value silently mis-filters; id/timestamp shape is exactly the free-identifier class the precedent passes through for clean server `400`/`404` (spec `[ASSUMED]`). Local regex/timestamp validation would duplicate server logic and risk drift.
3. **A new `validateProposalStatus` over the proposal status set, placed with the proposal command code; pass the other four filters and the ids through.** Chosen — silent conformance to the validate-closed-enum-locally pattern, a new validator instance over a new set, in the location 042/043 established (the command file, not the shared `status.go`).

**Decision**: Option 3. A pure `validateProposalStatus` (the `validateTensionStatus`/`validateMeetingType` shape — a `supportedProposalStatuses` map + a sorted `supportedProposalStatusNames()` helper) over the proposal status set, in the proposal command file, rejects an unsupported value as a `UsageError(2)` naming the value and the supported set **before any context assembly or request** (a transport tripwire asserts nothing was sent). Each filter (`--status` and the four pass-through filters) is sent as its query parameter only when `Changed()` and non-empty (the 026/034/038/043 optional-flag discipline); the `prp_` positional and the pass-through filter values surface a bad value as the API's clean `400`/`404`, classified by the shared chain.

**Consequences**: A new single-sourced proposal-status set with the proposal command code; adding a status is a one-line change tracking the vendored `spec/glassfrog-api-v5.yaml` (and includes `draft_with_conflicts`, easily missed against the FEATURE-MODEL prose). The shared `status.go` and `validateTensionStatus` are untouched. The four pass-through filters add no validation surface. No new `Outcome`/`ExitCode`.

---

## Data Model Design

`internal/glassfrog/proposal.go` (new), forward-compatible decoding throughout (unknown fields tolerated — the established decode posture):

- **`Proposal`** — `ID`, `Type`, `Status`, `TensionID`, `CircleID`, `ProposerID` (nullable → empty string), `ProposedAt`, `ResponseDeadline`, `AcceptedAt` (nullable timestamps), `Changes []ProposalChange`, `ResponseSummary`, `ExpectedResponseCount`, `ReceivedResponseCount`, `AvailableTransitions []string`, `CreatedAt`, `UpdatedAt`.
- **`ProposalChange`** — `ID`, `Type`, plus a free-form remainder (the `map[string]any` precedent from `actions.go`) so per-type keys decode without a per-type schema and structured output stays faithful.
- **`ResponseSummary`** — `Total`, `NoObjection`, `BringToMeeting` (ints; aggregate-only — no per-person field exists, enforcing the spec non-behavior at the type level).

Decoded via the existing generic envelopes: `Document[Proposal]` for the single read, `Page[json.RawMessage]` / `Page[Proposal]` for the walked list (the 043 pattern). No schema change to any other model; no transport change (both reads are bodyless `GET`s, `ContentType` stays `""`).

---

## Cross-cutting Concerns

**Concurrent-sibling coordination (055 Proposal Creation)** — the only non-trivial cross-cutting concern. 055 and 056 share the `proposal` group (`newProposalCommand`), the `glassfrog.Proposal` model, and the singular `proposal` render path. Design contract: **first-to-land creates, follower reuses** (043's resolved relationship to 042). 056's base is cut from current `main` where no proposal code exists, so as written 056 creates all three. If 055 merges first, 056 rebases and (a) adds its leaves to the existing group, (b) reuses the model — growing it if 055's was thinner — and (c) reuses/grows the singular render to surface `changes`/`response_summary`/`available_transitions`. Structured output is faithful regardless (raw-bytes pass-through), bounding the blast radius to human render + model field coverage. See Risks.

**List completeness** (silent conformance to 025 ADR-3): `proposal list` defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Proposal]` for human; the `--first-page` opt-out does a single `Execute`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial records, writes "incomplete — <cause>" to stderr, and exits non-zero. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`.

**Error handling**: identical to every read since 011 — typed client errors route through the single shared `classifyClientError`/`refineClientError` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. Failures render format-aware through 031's `Diagnose` and 032's `reportFailure` chokepoint — structured failures emit the 018 error envelope on stdout, human failures write the diagnostic on stderr; partial-walk notes stay on stderr in every format. No new exit codes. A `404` is the unknown/invisible proposal id (`get`); a `400` is a malformed pass-through filter (`list`); `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)` (a `GET`, so 017 may auto-retry). **Reads are not Premium-gated** — no plan-gate `403` is expected, so no special-casing of it here (that signal belongs to the write path's Plan-Limit Signalling, out of scope).

**Input validation order**: `--status` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/014/038/043 fail-fast discipline. `get` declares none of the list flags, so the cross-read rejection is cobra's unknown-flag handling (ADR-1).

**Testing**: the `Proposal`/`ProposalChange`/`ResponseSummary` decode (incl. nullable fields → empty, free-form change keys preserved, `Document[Proposal]`/`Page[Proposal]`) gets `internal/glassfrog` unit tests (the `Tension` decode shape); golden tests for the four new templates (`proposals.full`/`.compact` incl. the empty-set line and a multi-status list; `proposal.full`/`.compact` incl. a multi-change proposal and the response summary); `validateProposalStatus` unit tests (supported set passes, empty passes, unsupported names the value + set, `draft_with_conflicts` accepted); a `internal/cli` godog suite over `features/proposal-reads/...` driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when `get` is given a list flag (unknown-flag `UsageError`) and (b) no request when `--status` is unsupported.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025).

---

## Implementation Strategy

Single cohesive read pair; every seam below it landed. Because 056 (as cut from current `main`) is the first proposal command, it builds the group, model, and render paths bottom-up. Suggested ordering for task decomposition:

1. **Model** — `internal/glassfrog/proposal.go`: `Proposal`, `ProposalChange` (free-form remainder), `ResponseSummary`; decode unit tests (nullable handling, free-form change keys, envelope generics). Foundational — Phases 2–3 depend on it.
2. **Render** — both render paths: `ProposalsView` + `ResourceProposals` + `proposals.full`/`.compact`, and `ProposalView` + `ResourceProposal` + `proposal.full`/`.compact`, all in `builtinResources`; golden tests. Depends on Phase 1 (views project the model).
3. **Validator** — `validateProposalStatus` + the proposal status set in the proposal command file (the `validateTensionStatus` shape); unit tests. Tiny; independent of Phases 1–2, can run in parallel.
4. **Commands** — `newProposalCommand(seam)` group + `proposal list` (global: `paging.All` + the five filters, `--status` via `validateProposalStatus`, `--first-page`/`--per-page`, the 025 completeness logic) and `proposal get <prp-id>` (single: `Document[Proposal]`), guard-registered; route through the walked-list render pattern (`aggregateRawData` for structured / projection for human; the single read via `output.RenderSuccess` / the `proposal` render key) and `classifyClientError`. Depends on Phases 1–3.
5. **BDD** — the `proposal-reads` feature suite covering the spec's driving scenarios + the two structural tripwires (a list flag on `get`; unsupported `--status`). Depends on Phase 4.

Phases 1→2 are sequential; Phase 3 is parallel to 1–2; Phase 4 needs 1–3; Phase 5 needs 4.

---

## Risks

- **055 Proposal Creation concurrency (medium likelihood, low–medium impact)**: 055 and 056 both introduce the `proposal` group, `glassfrog.Proposal`, and the singular `proposal` render path; landing order is unknown (parallel workspaces). *Impact*: a merge conflict on `newProposalCommand`, `proposal.go`, and the singular render if both create them. *Mitigation*: the first-to-land-creates / follower-reuses contract (ADR-1, ADR-2, Cross-cutting) — the follower rebases, attaches its leaves, and grows (not duplicates) the model/render. Structured output is faithful regardless of model coverage (raw-bytes pass-through), bounding the conflict to human render + cobra wiring. Mirrors the 042→043 sequence, which resolved cleanly by rebase.
- **Free-form `changes[]` fidelity (low likelihood, low impact)**: `ProposalChange` is `additionalProperties: true` with no per-type schema. *Impact*: a typed-only model would drop per-type keys from human render. *Mitigation*: the `map[string]any` free-form remainder (the `actions.go` precedent) preserves all keys; structured `json`/`yaml` emits the raw server bytes verbatim, so no field is lost regardless of the human projection. The CLI never interprets change types (spec non-behavior).
- **Proposal status enum drift (low likelihood, low impact)**: the status set is single-sourced with the proposal command code against the spec enum (incl. `draft_with_conflicts`, which the FEATURE-MODEL prose omits). *Mitigation*: a spec change is a one-line update tracking the vendored `spec/glassfrog-api-v5.yaml`; the validator's unit test pins the set.
- **Proposal `changes`/`body` may contain long free text (low likelihood, low impact)**: `proposal.full` renders change bodies **verbatim** (`text/template`, no truncation — CONSTITUTION VI); only `*.compact` summarizes to a one-line projection. No reflow.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — exact command/flag spellings, the `ProposalsView`/`ProposalView` field names, the request-descriptor shape, and the template text are the **interface** skill's concern. The names used here (`proposal list`/`proposal get`, `--status`/`--role-id`/`--proposer-id`/`--proposed-after`/`--accepted-after`) are the spec-confirmed surface; treat them as the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units within the phases are the **tasks** skill's output; the Implementation Strategy above is the input.
- **Proposal Creation and the rest of the write flow** — `proposal create` (055) and `propose`/`withdraw`/`respond` (advance/withdraw/response specs) are deferred to their own specs, per the spec non-behaviors; this plan reads only, and surfaces `available_transitions` without invoking them.
- **Plan-Limit Signalling** — the Premium-gate `403` signal pairs with the proposal *write* path, not these reads (which are not gated); out of scope.
