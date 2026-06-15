# Plan: Response Recording

**Feature**: 058-response-recording
**Role**: Shaper
**Inputs**: `specs/058-response-recording/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md` (background); the sibling plans/accords 056-proposal-reads (the `proposal` group + `glassfrog.Proposal` model + render coordination), 042-tension-capture (the write-body `ContentType` seam, namespace group + leaf, closed-enum fail-fast validation, `Document[T]` single-write render), 045-tension-discard (the synthesized-result render for a write whose response the family renders), 053-guarded-writes (the landed `Request.IfMatch` field this plan deliberately leaves empty); existing code in `internal/cli` (`tension.go`, `status.go`), `internal/glassfrog`, `internal/apiclient` (`client.go` `Request.ContentType`/`IfMatch`, `execute.go`), `internal/render`, `internal/output`; the vendored `spec/glassfrog-api-v5.yaml` (`createProposalResponse`, `CreateProposalResponseRequest`, `ProposalVote`)

---

## System Architecture

Response Recording adds the **consent-window write** of the proposal flow — architecturally the single-write shape Tension Capture (042) established: resolve the output format, validate the closed-enum input fail-fast, assemble the connection, build the retrying executor, send exactly one `POST` carrying a small JSON body, then render the created resource or classify the failure. It introduces **no new transport machinery** — the write-body `Content-Type` seam landed in 042 (`Request.ContentType`), so this command merely *reuses* it; it sends **no `If-Match`** (recording a response is an append-create with no prior `ETag` — the `Request.IfMatch` field from 053 stays empty). It adds **no new `Outcome` or `ExitCode`** — every failure (incl. the Premium `403` and the one-per-person `422`) routes through the landed `classifyClientError`/`refineClientError` chain (011/015).

What it introduces is one verb leaf, a tiny request input, a new response model, a render path for the recorded response, and a closed-enum validator:

- **`glassfrog proposal respond <prp-id> --response <value>`** → `POST /proposals/{proposal_id}/responses` (`createProposalResponse`) — records the caller's consent-window response on one circulating proposal. `cobra.ExactArgs(1)` (the `prp_` id); the required `--response` carries `no_objection` | `bring_to_meeting`, validated locally before any request. The server derives the responding person from the token (007); the body carries no person. The `201` returns `{data: ProposalVote}` — the recorded vote with the **parent proposal's status at the time of response** (`accepted` when this response triggered auto-acceptance).

Because **no proposal code exists on the base cut from current `main`** (056 Proposal Reads and 055 Proposal Creation are concurrent in parallel workspaces), this feature is one of three that may be first to create the `proposal` group via `newProposalCommand`. The first-to-land-creates / follower-reuses contract (056 ADR-1, 042→043 lineage) governs: whichever lands first builds the group; the followers attach their leaves.

Data flow per invocation (the 042 single-write lineage): resolve `--output` (020) → validate `--response` is supplied and in the two-value set, fail-fast (`UsageError(2)`, no request) → resolve the connection context once (`AssembleFromOS`, 009) → build the `*apiclient.Client` (010/008/007) and `NewRetryExecutor` (017) → marshal `{response:{value}}` → one `Execute` `POST /proposals/{id}/responses` with `Content-Type: application/json`, `If-Match` empty (010; `X-Auth-Token` via 007's transport) → on `201` decode `Document[ProposalVote]`; machine formats emit the raw `{data}` bytes via `output.RenderSuccess` (018), the human path renders a recorded-response view (019) → success (0). On any failure, `reportFailure` + `classifyClientError`/`refineClientError` (015/017/032) map status to the existing outcomes.

---

## Architecture Decisions

### ADR-1: `proposal respond <prp-id> --response <value>` as a verb leaf under the (coordinated) `proposal` group; required closed-enum `--response` validated fail-fast

**Context**: The spec fixes the surface as `glassfrog proposal respond <prp-id>` with a required consent value from a two-value vocabulary. The `proposal` surface is verb-rich (create / list / get / propose / withdraw / respond), so it uses the **verb grammar** under a non-runnable `proposal` group — the same constraint 042 ADR-2 / 056 ADR-1 cite (a bare `proposal <prp-id>` would collide with subcommand dispatch). No `proposal` group exists in this base; 055/056 are concurrent. The value `--response` is a closed enum (`no_objection`, `bring_to_meeting`) where a wrong value the API would reject opaquely — exactly the input class 025 ADR-4 / 042 ADR-3 validate locally.

**Options considered**:
1. **Carry the value as a second positional** (`proposal respond <prp-id> <value>`). Rejected: a bare second positional reads as another id, gives no flag name to surface in the "required, supported values" usage error, and diverges from the `--body`/`--meeting-type`/`--status` flag-input precedent (042/043/056). The validated-flag shape is the family idiom.
2. **A required `--response` flag, validated locally via a new `validateProposalResponse`; the `prp_` id passed through.** Chosen — conformance to the 042 ADR-3 closed-enum fail-fast + the verb-grammar group precedent, adapted to the consent value.

**Decision**: Option 2. Add a `respond` leaf (`cobra.ExactArgs(1)` for the `prp_` id) under the `proposal` group, guard-registered (001) and explicitly wired. The leaf declares `--response` and delegates to a pure `runProposalRespond(cfg)` over the seam shape the family uses (`assemble`/`newClient`/`sleep`/`resolveFormat`), so every branch runs offline against a fake transport. A new pure `validateProposalResponse` (the `validateMeetingType`/`validateProposalStatus` shape — a `supportedProposalResponses` map + a sorted `supportedProposalResponseNames()` helper, in the proposal command file, **not** the shared `status.go`) rejects an **omitted** `--response` (the value is required — there is no default consent answer) and an **unsupported** value as `UsageError(2)`, naming the value/requirement and the supported set, **before any context assembly or request** (a transport tripwire asserts nothing was sent). Error precedence follows the siblings: resolve `--output` first, then validate `--response`. The `prp_` id is escaped as one path segment (`url.PathEscape`) and passed through to the API's clean `404` — no local id-shape regex (056 ADR-3 / 042 ADR-3).

**Concurrent-sibling coordination (055/056)**: first-to-land defines `newProposalCommand` + the group; the followers add their leaves to the existing group (043's relationship to 042's `newTensionCommand`). 058's `respond` leaf attaches alongside 056's `list`/`get` and 055's `create`. The `--response` flag lives only on `respond`, so passing it to another leaf is a cobra unknown-flag `UsageError(2)` for free (the structural guard 034/038/042/056 rely on).

**Consequences**: A new `respond` verb leaf on the shared `proposal` group; the `--base-url` / `--output` persistent root flags (011/020) are inherited. A new single-sourced proposal-response set in the proposal command code (adding a value tracks the vendored spec enum). The shared `status.go` and the tension/proposal-status validators are untouched. Adds no new `Outcome`/`ExitCode`. Cross-spec relevant (the namespace and the response vocabulary) → record in DECISIONS.md.

### ADR-2: Establish the `glassfrog.ProposalVote` response model and add a recorded-response render path; tiny per-operation request input

**Context**: The `201` returns `{data: ProposalVote}` — a **different schema** from `Proposal` (which 056 establishes): `ProposalVote` carries `id` (`prr_`), `type` (`proposal_response`), nullable `proposal_id`, **`proposal_status`** (the parent proposal's status at response time — `draft` / `proposed_outside_meeting` / `escalated` / `accepted` / `draft_with_conflicts`), `value` (`no_objection` | `bring_to_meeting`), and `created_at` / `updated_at`. The request body is the nested envelope `{response: {value}}` (`CreateProposalResponseRequest`). The family contract (042/045) is to **produce structured data rendered in the effective format**, so the recorded response must render under `full`/`compact`/`json`/`yaml`. No `ProposalVote` model and no recorded-response render path exist (056 adds `Proposal`/`ProposalChange`/`ResponseSummary` and the `proposals`/`proposal` render keys — not the vote).

**Options considered**:
1. **Reuse 056's `Proposal` model / `proposal` render path.** Rejected: the `201` body is a `ProposalVote`, not a `Proposal`; reusing the proposal render would mis-describe the result (it has no `changes`/`response_summary`). The response is its own resource.
2. **Grow the shared request/response types speculatively.** Rejected per the per-operation request-input rule (DECISIONS §322 — request inputs are constructed per operation; fork when contracts diverge) and the anti-speculation idiom.
3. **A new `ProposalVote` response model + a tiny `{response:{value}}` request input + a dedicated recorded-response render path.** Chosen — one new response type (011 ADR-1 grow-not-duplicate applies to *response* models; this is a genuinely new schema, not a grow of `Proposal`), a minimal request envelope, and a render path mirroring the `ResourceProject` singular shape.

**Decision**: Option 3. Add `glassfrog.ProposalVote` (forward-compatible decoding, the `Tension`/`Proposal` shape: `ID`, `Type`, `ProposalID` (nullable → empty string), `ProposalStatus`, `Value`, `CreatedAt`, `UpdatedAt`) plus a tiny request input encoding `{response: {value}}` (a `ProposalResponseInput`/`ProposalResponseBody` with a single non-`omitempty` `Value`, since the value is always supplied non-empty after validation — the 042 `TensionInputBody` shape). Place these in `internal/glassfrog/proposal.go` (the file 055/056 introduce) under grow-not-duplicate — adding the vote type, never a second proposal file. The `201` decodes into the existing generic `Document[ProposalVote]` (§143/§254 — no per-resource envelope). Add a recorded-response render path: a `ProposalVoteView` + `ResourceProposalResponse` + `proposal-response.full` / `.compact` templates, registered in `builtinResources` so the exhaustiveness guard covers them. `proposal-response.full` surfaces the `prr_` id, the recorded `value`, the anchoring `proposal_id`, and — **load-bearing** — the parent `proposal_status` (rendered so `accepted` is legible as "this response closed the consent window"); `.compact` is id + value + proposal status on one line. Structured `json`/`yaml` reuses the landed machinery: `output.RenderSuccess` over the raw `{data: ProposalVote}` bytes (018) — faithful regardless of model field coverage.

**Coordination**: `ProposalVote` is a **new type either landing order** (neither 055 nor 056 creates it), so the only shared-file touch is appending to `proposal.go` and `builtinResources`; no grow/shrink negotiation with 055/056's `Proposal`. If 055/056 land first, 058 appends; if 058 lands first, it adds `proposal.go` with the vote type and 055/056 grow it with `Proposal`.

**Consequences**: A new response model, a tiny request input, and one new render resource (two templates). No transport change beyond reusing `ContentType` (ADR-3). The `proposal_status` field is the agent-visible signal of auto-acceptance, surfaced without the CLI computing acceptance. Cross-spec relevant (the vote model + render key future response reads, if any, would reuse) → record in DECISIONS.md.

### ADR-3: Reuse the landed `ContentType` write-body seam; send no `If-Match` (append-create, not a guarded edit)

**Context**: Two request headers landed for the write path: `Request.ContentType` (042, set to `application/json` for a JSON body) and `Request.IfMatch` (053, the optimistic-concurrency precondition a guarded *edit* sends). Recording a response is a `POST` that **creates** a vote; the write commands that just landed (053/054 guarded writes + stale-write surfacing) might invite the reflex to guard this write too.

**Options considered**:
1. **Unguarded append-create: set `Content-Type: application/json`, leave `If-Match` empty.** Chosen — recording a response creates a new sub-resource; there is no prior `ETag` of *the vote* to guard against, and concurrency on the parent proposal is not what a response conditions on. The API enforces correctness itself (one-per-person `422`); `If-Match` would add a precondition the endpoint neither expects nor needs.
2. **Guard the write with `If-Match` (the 053 mechanism).** Rejected: 053's field is opt-in — "the caller populates `IfMatch` only on requests it intends to guard" — and a create has no captured version to send. Guarding here would either send an empty (no-op) precondition or require a spurious pre-read with no clobber semantics to protect. The double-response case is the server's `422`, not a `412`.

**Decision**: Option 1. `runProposalRespond` sets `Request.ContentType = "application/json"` (the 042 seam, reused unchanged) and leaves `Request.IfMatch` empty (the 053 field's documented empty-as-unguarded default). This is **silent conformance** to both landed precedents — reusing `ContentType` as 042 intended for "the rest of the write path (proposals, operational writes)", and honouring 053's opt-in guard contract by not opting in.

**Consequences**: No transport-layer change; the reads and guarded edits stay byte-identical. The non-behavior "must not send an `If-Match`" (spec) holds by construction. A `POST` is never auto-retried on `429` (017's `isSafeMethod` gate), so a rate-limit retry cannot double-record a response — the one-per-person rule is doubly safe (method gate + server `422`).

---

## Data Model Design

`internal/glassfrog/proposal.go` (created by whichever of 055/056/058 lands first; 058 appends), forward-compatible decoding throughout:

- **`ProposalVote`** (new, 058) — `ID` (`prr_`), `Type` (`proposal_response`), `ProposalID` (nullable → empty string), `ProposalStatus`, `Value`, `CreatedAt`, `UpdatedAt`. Decoded via the existing generic `Document[ProposalVote]` for the single `201` write result.
- **`ProposalResponseInput` / `ProposalResponseBody`** (new, 058) — the request envelope `{response: {value: <string>}}`; a single non-`omitempty` `Value` field (always non-empty after validation — the 042 `TensionInputBody` shape). A `NewProposalResponseInput(value)` constructor mirrors the 042 input-constructor precedent.

No change to `Proposal`/`ProposalChange`/`ResponseSummary` (056's types) — the vote is a distinct schema. The request body sets `ContentType`; `IfMatch` stays `""`.

---

## Cross-cutting Concerns

**Premium gating (no new outcome)**: `createProposalResponse` is Premium-gated — a `403` means async proposals are not enabled. It classifies through the landed shared chain as `PermissionError(4)` (015's `401`/`403` arm), with the RFC 9457 detail surfaced (015's `ExtractProblem`). The command adds **no plan-limit-specific interpretation** — turning a recognized plan-gate `403` into an actionable "not available on your plan" diagnostic is Plan-Limit Signal, a separate later capability (spec non-behavior, FEATURE-MODEL). No `x-feature-gate` inspection here.

**One-per-person `422` (no special handling)**: a second response by the same person returns `422`, classified as `APIError(3)` with the API detail surfaced — the command does **not** retry it or fold it into success (contrast 045's `404`-as-success for the idempotent delete; a `422` here is a genuine rejection, not an idempotent re-run). An unknown/invisible proposal is `404` → `APIError(3)`.

**Non-idempotent retry (silent conformance to §133)**: `respond` uses the same `NewRetryExecutor` as every command, but 017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD` — a `POST` surfaces the `429` (→ `RateLimited(5)`) on first occurrence and is never silently re-sent, so recording cannot double-fire on a rate-limit retry.

**Failure mapping (no new outcomes)**: every failure routes through the landed `reportFailure` → `refineClientError`/`classifyClientError` chain. Not-authenticated → the shared fail-safe (`UsageError(2)` NoCredentials / `RuntimeError(1)` CredentialError); transport → `NetworkUnavailable(6)`; non-2xx via 015's `ExtractProblem`; an undecodable `201` body → `DecodeError` → `APIError(3)`. Failures render format-aware via 032's `reportFailure` chokepoint (structured → 018 envelope on stdout; human → diagnostic on stderr). The command adds no interpretation and never prints the token.

**Output**: the `201` `{data: ProposalVote}` decodes into the existing generic `Document[ProposalVote]`. Machine formats emit the raw `{data}` via `output.RenderSuccess`; the human path renders the recorded-response view surfacing the `prr_` id, the recorded value, and the parent `proposal_status`. Buffer-then-write so a render failure leaves stdout empty and maps to `RuntimeError(1)`.

**Input validation order**: `--output` resolves first (020), then `validateProposalResponse` (pure, no I/O) runs ahead of any context assembly, so a transport tripwire can assert no request issued on either rejection path (missing or unsupported value) — the 011/013/014/038/043/056 fail-fast discipline.

**Testing**: the `ProposalVote` decode (nullable `proposal_id` → empty, `proposal_status` incl. `accepted`, `Document[ProposalVote]`) gets `internal/glassfrog` unit tests (the `Tension`/`Proposal` decode shape); a golden test for the two new templates (`proposal-response.full`/`.compact`, incl. an `accepted`-status case proving auto-acceptance is legible); `validateProposalResponse` unit tests (both supported values pass, empty/omitted is the required-error, an unsupported value names the value + set); a `internal/cli` godog suite over `features/response-recording/...` driven by a fake transport, with **transport tripwires** asserting no request when `--response` is omitted and when it is unsupported, and asserting the body carries `{response:{value}}` and **no person field** and that `Content-Type: application/json` is set while `If-Match` is absent. The seam injection keeps `runProposalRespond` pure over a fake transport for the happy/usage/`403`/`422`/`404` branches offline.

**Configuration**: none new. Reuses `--base-url` (008) and `--output`/`-o` (020/035) persistent root flags; the token via 005/007.

---

## Implementation Strategy

Single cohesive write leaf; every seam below it landed. Because the base has no proposal code, the first-to-land sibling builds the `proposal` group; this plan designs `respond` so it attaches whether the group pre-exists or not. Suggested ordering for task decomposition:

1. **Model + request input** — append `ProposalVote` and `ProposalResponseInput`/`ProposalResponseBody` (+ constructor) to `internal/glassfrog/proposal.go` (create the file if first to land); decode + marshal unit tests (nullable handling, `{response:{value}}` envelope shape). Foundational.
2. **Render** — `ProposalVoteView` + `ResourceProposalResponse` + `proposal-response.full`/`.compact`, registered in `builtinResources`; golden tests (incl. the `accepted` proposal-status case). Depends on Phase 1.
3. **Validator** — `validateProposalResponse` + the proposal-response set in the proposal command file (the `validateMeetingType` shape, required-value semantics); unit tests. Tiny; parallel to 1–2.
4. **Command** — `runProposalRespond` + the `respond` leaf attached to the `proposal` group (created via `newProposalCommand` if first to land, else reused), guard-registered and wired; resolve format → validate → assemble → marshal `{response:{value}}` → one `Execute` `POST` (`Content-Type: application/json`, no `If-Match`) → render `Document[ProposalVote]` / classify. Depends on Phases 1–3.
5. **BDD** — the `response-recording` feature suite covering the spec's driving + validation scenarios and the structural tripwires (omitted/unsupported `--response`; no person in body; `Content-Type` set, `If-Match` absent). Depends on Phase 4.

Phases 1→2 sequential; Phase 3 parallel to 1–2; Phase 4 needs 1–3; Phase 5 needs 4. If a sibling (055/056) has already landed the `proposal` group, Phase 4 attaches to it; otherwise Phase 4 also creates the group (the tasks skill flags this branch point).

---

## Risks

- **055/056 concurrency on the `proposal` group (medium likelihood, low impact)**: 055/056/058 all need `newProposalCommand` + the `proposal` group, and may land in any order (parallel workspaces). *Impact*: a merge conflict on the group constructor and the `proposal.go` model file. *Mitigation*: the first-to-land-creates / follower-reuses contract (ADR-1, 056 ADR-1, the 042→043 precedent). `ProposalVote` is a new type in every order (no grow/shrink negotiation), so 058's model touch is an append; the cobra touch is one leaf attach. Mirrors the 042→043 sequence, which resolved cleanly by rebase.
- **Treating the Premium `403` or one-per-person `422` specially (low likelihood, low impact)**: the reflex (after 053/054's write-path work) might be to add plan-limit or duplicate-response handling here. *Mitigation*: ADR cross-cutting fixes both as shared-chain classifications (`PermissionError(4)` / `APIError(3)`) with no command-side interpretation; Plan-Limit Signal is explicitly out of scope (spec non-behavior).
- **Reflexively guarding the write with `If-Match` (low likelihood, low impact)**: 053 just landed the guarded-write mechanism. *Mitigation*: ADR-3 — recording a response is an append-create with no prior version; `If-Match` stays empty (the field's documented opt-in default), pinned by a BDD tripwire asserting the header is absent.
- **`proposal_status` legibility (low likelihood, low impact)**: the auto-acceptance signal rides on the vote's `proposal_status`. *Mitigation*: the `proposal-response.full` template surfaces it explicitly; a golden test pins the `accepted` case. Structured output carries the raw field regardless.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — the exact `respond`/`--response` spellings, the `ProposalVote`/input field names, the request-descriptor shape, and the template text are the **interface** skill's concern. The names used here resolve the spec-confirmed surface (`proposal respond <prp-id> --response <no_objection|bring_to_meeting>`); treat them as the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving + Validation scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units within the phases are the **tasks** skill's output.
- **The rest of the proposal write flow** — `proposal create` (055), `list`/`get` (056), and `propose`/`withdraw` (advance/withdraw specs) are deferred to their own specs; this plan records only a response and surfaces the resulting `proposal_status` without driving any transition.
- **Plan-Limit Signal and Write-Safety confirmation** — the actionable plan-gate diagnostic and the operator-layer write confirmation are their own capabilities (spec non-behaviors); out of scope.
- **Optimistic concurrency (`If-Match`)** — deliberately not used (ADR-3); the 053 field stays empty.
