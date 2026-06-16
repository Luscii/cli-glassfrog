# Interface Accord: Response Recording — CLI

**Feature**: 058-response-recording
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (a `respond` verb leaf under the coordinated `proposal` group; `ExactArgs(1)` for the `prp-id`; a **required** `--response` validated locally via a new `validateProposalResponse`), ADR-2 (a new `glassfrog.ProposalVote` response model + `{response:{value}}` request input; a recorded-response render path surfacing the parent `proposal_status`), ADR-3 (reuse 042's `Request.ContentType` write-body seam; send no `If-Match`). Coordination: shares the `proposal` group/model file with the concurrent Proposal Creation (055) and Proposal Reads (056) — first-to-land creates the group, followers attach leaves.

---

This accord pins the operator-facing surface for recording a consent-window response: the single leaf `glassfrog proposal respond <prp-id>` under the `proposal` group, its required `--response` flag, the rendered recorded-response output, and the exit codes. Recording a response is the Premium-gated consume/respond write of the Governance Proposals flow — it posts one of two consent values (`no_objection` | `bring_to_meeting`) to a circulating proposal and surfaces the recorded vote, including the parent proposal's status (which reads `accepted` when this response triggered auto-acceptance). The request seam it calls through is pinned in `010/interface-spec.md` (`Request.ContentType` from 042; `Request.IfMatch` from 053, left empty here); format selection/rendering in `018`/`019`/`020`/`035`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009); failure rendering through the landed `reportFailure` chokepoint (032). Because no proposal code exists on the base, this feature may be the first to create the `proposal` group (`newProposalCommand`) — see Consistency Notes.

## Surface

### `glassfrog proposal` — command group (no action)

The non-runnable group that parents the proposal verb leaves (`create` from 055; `list`/`get` from 056; `respond` added here; later `propose`/`withdraw`). Created via `newProposalCommand(seam)` (the `newTensionCommand` shape) by whichever of 055/056/058 lands first; the followers attach their leaves to the existing group. `glassfrog proposal` with no subcommand prints help. Leaves attach through the registration guard (001).

### `glassfrog proposal respond <prp-id> --response <value>` — record a consent-window response

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage`. Writes `POST /proposals/{proposal_id}/responses` (`createProposalResponse`) and produces the recorded response (a `ProposalVote`) as a single result. **Premium-gated** — a `403` means async proposals are not enabled on the org's plan.

**Synopsis**:
```
glassfrog proposal respond <PRP_ID> --response <no_objection|bring_to_meeting>
                           [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `PRP_ID` | string | yes | The circulating proposal to respond to (`prp_…`). Exactly one required (cobra `ExactArgs(1)`). Escaped as a single path segment (`url.PathEscape`) and **passed through unvalidated** — an unknown or invisible id → `404`. |

**Leaf flag** (declared only on `respond`):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--response` | string | — (**required**) | The consent value to record. **Validated locally** before any request against the response vocabulary (`no_objection`, `bring_to_meeting`) via a new `validateProposalResponse`. **Required with no default** — consent has no implicit answer, so an omitted `--response` is a usage error naming the flag as required and listing the supported values; an unsupported value is a usage error naming the value and the supported set. Either rejection sends **no API call**. Sent as the request body's `response.value`. |

There is no `--person`/`--responder` flag: the server derives the responding person from the token's identity (007). The request body carries no person field.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml` (widened by 035 to also accept a user-template ref). |

**Output** (success, stdout): on `201` the recorded response is rendered by Output Format Selection (020) in the resolved format. `json`/`yaml` emit the raw `{data: ProposalVote}` document (018, via `output.RenderSuccess`); `full`/`compact` render the human projection (019) via the new `proposal-response` render key. The raw API envelope is never emitted under a human format.

*`proposal-response.full`* surfaces the recorded vote with the parent proposal's status legible as the auto-acceptance signal:
```
<prr_…>  recorded
  Response:        <no_objection | bring_to_meeting>
  Proposal:        <prp_… | (none)>
  Proposal status: <draft | proposed_outside_meeting | escalated | accepted | draft_with_conflicts>
  Recorded:        <created_at>
```
When the response closed the consent window, `Proposal status:` reads `accepted`. Each nullable field renders through an explicit-absence guard rather than a blank.

*`proposal-response.compact`* — one line: `<prr_…>  <response value>  [<proposal status>]`.

There is no empty/list case — `respond` always produces exactly one recorded response on success.

## Interactions

**Dispatch**: the leaf has its own `RunE` (no `len(args)` branching). Before any network call, in order: (1) cobra `Args`/flag parsing (any extra positional, a missing `--response`'s required-ness if enforced by cobra, or an unknown flag fails here); (2) `--output` resolution (020); (3) `--response` validation (presence + closed-enum via `validateProposalResponse`). Output resolution precedes response validation so an invalid `--output` is reported even when `--response` is also invalid — both are pure, pre-assembly checks, so either order preserves the no-request guarantee. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/014/042/056).

**Request**: one `Execute` to `POST /proposals/{prp-id}/responses` with body `{"response": {"value": "<no_objection|bring_to_meeting>"}}`, `Content-Type: application/json` (the 042 `Request.ContentType` seam, reused), and **no `If-Match`** (the 053 `Request.IfMatch` field left empty — recording a response is an append-create, not a guarded edit; ADR-3). The `201` decodes `glassfrog.Document[ProposalVote]`. The `POST` is **never auto-retried on `429`** (017's `isSafeMethod` gate restricts auto-retry to `GET`/`HEAD`), so a rate-limit retry cannot double-record.

**Premium gating**: this write **is Premium-gated**. A `403` (async proposals not enabled) is classified generically as `PermissionError(4)` through the shared chain — there is **no response-recording-specific plan-limit handling** (the actionable plan-limit diagnostic is the write path's Plan-Limit Signal, out of scope). No `x-feature-gate` inspection here.

**One response per person**: a second response by the same person returns `422`, surfaced as `APIError(3)` through the shared chain — **not retried** and **not folded into success** (contrast Tension Discard's idempotent `404`-as-success; a `422` here is a genuine rejection).

**Piping / scripting**: on success, stdout carries only the rendered result. On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005/007. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed `reportFailure` chokepoint (032): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 058 reuses `reportFailure` unchanged and **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Response recorded | — | `Success` | 0 | — (recorded response on stdout) |
| Missing `--response`, or unsupported value | — (`validateProposalResponse`) | `UsageError` | 2 | missing: "`--response` is required — supported: bring_to_meeting, no_objection"; unsupported: "unsupported --response value \"…\" — supported: bring_to_meeting, no_objection"; no request sent |
| Wrong positional count (zero / >1), or unknown flag | — (cobra) | `UsageError` | 2 | cobra's arg-count / unknown-flag message; no request sent |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Premium plan-gate — async proposals not enabled (`403`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + extracted detail (015); per-class next step (no plan-limit-specific text — out of scope) |
| Already responded (`422`), or other rejected body | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + extracted detail (015) |
| Unknown / invisible proposal id (`404`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + extracted detail (015) |
| Permission denied other than the plan-gate (`401`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + detail; per-class next step |
| Rate limited (`429`) | `*ResponseError` (→ `*ProblemError`, 015) | `RateLimited` | 5 | names the HTTP status; next step to retry later (POST not auto-retried) |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| `201` body did not match the expected shape | `*DecodeError` | `APIError` | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Render failure on a buffered result | `*RenderError` (019) | `RuntimeError` | 1 | buffer-then-write leaves stdout empty; cause names the render failure |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Consume/respond write of the proposal flow** (plan System Architecture): `proposal respond` is the single-write structural twin of Tension Capture (042)'s `tension create` — resolve output, validate the closed-enum input fail-fast, assemble, one `POST` with a JSON body, render the created resource or classify the failure — run as a verb leaf under the `proposal` group (the verb-rich-noun grammar 042/056 established) rather than a noun pair. It reuses the persistent `--base-url` (011) and `--output`/`-o` (020/035), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `RetryExecutor` (017), the 035-widened render flow (`output.RenderSuccess` for structured / `writeHuman` over the projection for human), and 032's `reportFailure` chokepoint, all unchanged.
- **Reuses the landed write-body seam; sends no `If-Match`** (plan ADR-3): the body rides `Request.ContentType = "application/json"` (042's narrow field, intended for "the rest of the write path"), and `Request.IfMatch` (053) is left **empty** — recording a response is an append-create with no prior `ETag` of the vote to guard. A BDD tripwire pins `Content-Type` present and `If-Match` absent.
- **One local validator, a new required set** (plan ADR-1): `--response` is the single closed-enum input, validated by a **new `validateProposalResponse`** over the response vocabulary (`no_objection`, `bring_to_meeting`) — placed **with the proposal command code** beside the `validateProposalStatus` (056) / `validateMeetingType` (042) precedent (NOT in shared `internal/cli/status.go`), and **not** reusing `validateStatus`/`validateTensionStatus`/`validateProposalStatus`, whose vocabularies are wrong here. Unlike those optional filters, `--response` is **required** — an omitted value is a usage error, not a "send no filter" no-op. The `prp_` id is a free identifier passed through to a clean `404`.
- **New response model + render key** (plan ADR-2): this feature establishes `internal/glassfrog.ProposalVote` (id `prr_`, type, nullable `proposal_id`, `proposal_status`, `value`, timestamps; forward-compatible decoding) — a **distinct** schema from 056's `Proposal` (not a grow of it), plus a tiny `ProposalResponseInput`/`ProposalResponseBody` encoding `{response:{value}}`. It adds **one** render key — the singular `proposal-response` (a `ProposalVoteView`), registered in `builtinResources` so the exhaustiveness guard covers it. The `proposal_status` field is surfaced explicitly in `proposal-response.full` (the auto-acceptance signal); structured `json`/`yaml` carries it via raw-bytes pass-through regardless of model coverage. No per-person attribution exists on the recorded vote (`ProposalVote` is summary-stats-only at the proposal level — enforced at the type level, matching 056's anti-attribution non-behavior).
- **Coordination with Proposal Creation (055) and Proposal Reads (056)** (plan Risks): the `proposal` group (`newProposalCommand`) and the `internal/glassfrog/proposal.go` model file are shared with the concurrent 055/056. First-to-land creates the group and the file; followers attach their leaves and append their types. `ProposalVote` is a **new type in every landing order** (neither 055 nor 056 defines it), so 058's model touch is an append with no grow/shrink negotiation; its cobra touch is one leaf attach. Mirrors the 042→043 sequence, which resolved cleanly by rebase.
- **Flag/command spellings** (`proposal respond`, `--response`) resolve the spec's confirmed surface; the verb and flag are conventional and adjustable at build time without changing behavior. The `[ASSUMED]` spec choice of a `--response` flag (over a second positional) is pinned here per plan ADR-1.
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
