# Interface Accord: Withdraw Proposal — CLI

**Feature**: 059-withdraw-proposal
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (`withdraw <prp-id>` as the structural twin of the **landed** `propose` leaf — flagless leaf on the `proposal` group; bodyless `POST` to the `/withdraw` sub-path; id passed through; decode-and-render the returned `{data: Proposal}` — NOT synthesize; server-authorized — no client pre-check of `available_transitions`; `404`/`422` are REAL FAILURES, the inverse of discard's `404`-as-success; the Premium `403` routes through the shared classifier with no plan-gate special-casing — all silent conformance to 057 §392/§393/§395/§396), ADR-2 (the withdraw is DESTRUCTIVE — the server deletes existing responses and clears `proposed_at`/`response_deadline`; the command surfaces the returned `draft` proposal faithfully, narrates none of the deletion, emits no stderr advisory, and adds NO confirmation/`--force`/`--yes` guard — destructive-write confirmation is deferred to the separate Write-Safety Guardrail). Cross-cutting: no sibling-coordination concern (055/056/057/058 all landed — the `proposal` group/model/render/`propose`-template are on `main`); retry (silent conformance to §133 `isSafeMethod` — `POST` not auto-retried on `429`).

---

This accord pins the operator-facing withdraw surface: the new flagless `withdraw` leaf under the `proposal` group, the bodyless `POST /proposals/{id}/withdraw`, the decode-and-render of the returned `draft` proposal, and exit codes. Every shared seam it rides is **landed**: the registration guard (001), format selection/rendering (`018`/`019`/`020`/`035`), the request seam (`010`, incl. the optional `ContentType` left empty here), the shared failure chokepoint (`reportFailure`/`classifyClientError`, `015`/`031`/`032`), and the frozen `Outcome`/`ExitCode` registry (`004`/`011`/`054`). Unlike Advance to Circulation (057), the proposal-specific machinery — the `proposal` group, the `glassfrog.Proposal` model + `Document[Proposal]` decode, and the singular `proposal` render path — is **already on `main`** (landed by Proposal Creation 055 / Proposal Reads 056), and the near-identical `propose` leaf (057) is the **structural template** this command mirrors. This feature adds **one leaf** and reuses everything else — no new model, no new render key, no new transport. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the **`withdraw` transition verb of the `proposal` family** — it returns one circulating (`proposed_outside_meeting`/`escalated`) proposal to `draft` by id, the mirror of `propose`.

---

## Surface

### `glassfrog proposal` — command group (no action)

The non-runnable group landed by Proposal Creation (055) / Proposal Reads (056): it carries no action and parents its children (`create` from 055; `propose` from 057; `respond` from 058; `list`/`get` from 056; `withdraw` added here), so `glassfrog proposal` with no subcommand prints help. This feature attaches one new leaf to the existing group through the registration guard (001); it does not redefine the group. The group `Short` widens to name `withdraw` alongside the rest (a `Short` that omitted it would make `glassfrog proposal --help` misleading). **No sibling-coordination caveat** (plan Cross-cutting): the group, the `Proposal` model, the singular `proposal` render path, and the `propose` leaf template are all landed — 059 reuses them directly, with no first-to-land-creates contract.

### `glassfrog proposal withdraw <prp-id>` — return a circulating proposal to draft

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `POST /proposals/{proposal_id}/withdraw` (`withdrawProposal`) and produces the `draft` proposal the server returns.

**Synopsis**:
```
glassfrog proposal withdraw <PRP_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `PRP_ID` | string | yes | The proposal to withdraw (`prp_…`), used as the `POST /proposals/{proposal_id}/withdraw` path id. Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it — and an unknown or invisible id's `404` is a **failure** here (plan ADR-1), unlike discard's `404`-as-success. |

**Flags**: **none.** A transition returns a proposal to draft whole — there is nothing to edit and the endpoint takes no request body. `withdraw` declares no flags of its own, so a stray `--status`/`--changes`/`--body` is a cobra unknown-flag usage error for free (the structural guard 045/056/057 rely on). The leaf exposes **no confirmation/`--force`/`--yes` flag** even though the withdraw is destructive (it deletes existing responses and clears the proposed timestamps): the CLI is non-interactive and agent-driven, and operator-layer destructive-write confirmation is the separate Write-Safety Guardrail capability (plan ADR-2). It exposes no `--force-transition`/pre-check affordance either (the server authorizes the transition; the command does not pre-read `available_transitions`, plan ADR-1).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 (widened by 035) | `full` (default) \| `compact` \| `json` \| `yaml`, or a user-template ref (`@file`/`-`). |

**Output** (success, stdout): the **`draft` proposal the server returned** on the `200` — the full `Proposal`, now `status: draft`, with its `proposed_at` and `response_deadline` **cleared**, its prior `proposal_responses` **deleted** (so `response_summary` is zeroed and `received_response_count` is `0`), and the updated `available_transitions` (typically `withdraw` is gone and `propose` is offered again). It is decoded into the shared `glassfrog.Proposal` (`Document[Proposal]`) and rendered by Output Format Selection (020, widened by 035) in the effective format, through the **same singular `proposal` render path `proposal get`/`proposal propose` use** — no new render key. The command **synthesizes nothing** (plan ADR-1) and **narrates none of the side effects** (the response deletion, the timestamp clearing, the fact that responders must re-respond if the proposal is re-proposed) — they are visible only through the returned data (plan ADR-2).

*`json`* / *`yaml`* — the raw `{data: Proposal}` document, serialized verbatim (018, faithful regardless of model field coverage). The example below is **abbreviated** — standard `Proposal` fields (`type`, `created_at`, `updated_at`) are omitted for brevity; the real response carries the full schema (per 056's `Proposal` model). Note `proposed_at`/`response_deadline` cleared (`null`), `response_summary` zeroed, and `available_transitions` showing `propose` offered again:
```json
{
  "data": {
    "id": "prp_0123456789abcdef0123456789abcdef",
    "status": "draft",
    "tension_id": "ten_0123456789abcdef0123456789abcdef",
    "circle_id": "role_0123456789abcdef0123456789abcdef",
    "proposer_id": "per_0123456789abcdef0123456789abcdef",
    "proposed_at": null,
    "response_deadline": null,
    "changes": [ { "type": "CreateRole", "name": "Scheduling" } ],
    "response_summary": { "total": 0, "no_objection": 0, "bring_to_meeting": 0 },
    "expected_response_count": 5,
    "received_response_count": 0,
    "available_transitions": [ "propose" ]
  }
}
```

*`full`* / *`compact`* (or a selected user template, 035) — the human projection (019) over the existing singular `proposal` render key (056/055): `full` shows id, the new `draft` status, the anchoring `tension_id`/`circle_id`/`proposer_id`, the lifecycle timestamps (now with the deadline cleared), `changes` by type (body verbatim — CONSTITUTION VI), the zeroed `response_summary` counts, expected/received counts, and `available_transitions`; `compact` is the one-line projection (id + status + change-count + a one-line response summary). The exact field selection and template text are 056/055's interface contract, reused unchanged.

The command introduces **no new field names, no new render key, and no new template** — it renders the same `Proposal` shape the reads, creation, and advance surface.

## Interactions

**Dispatch**: the `withdraw` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here); (2) `--output` resolution (020/035) — a present-but-invalid selector, or an unparseable/empty user-template source, fails fast as a usage error with **no request**. There are no input-field checks (the leaf has no flags). Resolving `--output` first keeps error precedence consistent with the siblings; a transport tripwire asserts the no-request guarantee on the bad-`--output` path (per 045/056/057).

**Request**: on valid input, the leaf assembles the connection and sends **one** `Execute` to `POST /proposals/{proposal_id}/withdraw` with **no body and no `Content-Type`** (the optional `ContentType` seam 042 added is left empty — a bodyless `POST`, the discard `DELETE` shape, not the create body shape) and **`out == &Document[Proposal]`** (the `200` carries the withdrawn proposal to decode — the `propose` shape exactly). There is **no prior `GET`** — the command does not pre-read the proposal to check `available_transitions` (plan ADR-1); a validation tripwire asserts exactly one request and no read. **No `If-Match` header is sent** (`withdraw` is a transition with no `If-Match` parameter; Guarded Writes' `Request.IfMatch` stays unused — 053). The `prp_` id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (plan ADR-1).

**Outcome interpretation** (plan ADR-1): the leaf branches on the `Execute` result —
- **`err == nil`** (a `2xx`, i.e. `200`): success. Decode the returned `Document[Proposal]` and render it on stdout (machine: `output.RenderSuccess` over the raw bytes; human/`-o template`: `writeHuman` over the decoded `ProposalView`, buffer-then-write) and exit `Success` (0).
- **any error** — including a `404` (no such/invisible proposal) and a `422` (transition not allowed — already `draft`, or `withdraw` not offered): route to the shared `reportFailure` **with no status interception** (the inverse of discard's `errors.As`-for-`404`-as-success divert). See Error Communication.

No status is folded into success; a withdraw has no idempotent re-run (re-withdrawing a `draft` proposal is a genuine `422`), so `404`/`422` are real failures the operator must see. There is **no stderr advisory** (plan ADR-2 — there is a single success outcome, and the response deletion is already legible in the returned proposal's zeroed `response_summary` and cleared deadline).

**Destructive, but unconfirmed** (plan ADR-2): the withdraw deletes the proposal's existing responses and clears its proposed timestamps server-side. The command does **not** prompt, require `--force`/`--yes`, or print a "this will discard N responses" warning — it issues the transition and surfaces the resulting `draft` proposal as returned. An agent can withdraw and immediately re-read the now-`draft` proposal without handling a prompt. Destructive-write gating is the operator-layer Write-Safety Guardrail's job, applied across the write path centrally, not per-command here.

**Non-idempotent retry**: `withdraw` is built on the same `RetryExecutor` as the rest, but 017's `isSafeMethod` restricts `429` auto-retry to `GET`/`HEAD`, so a `POST` surfaces a `429` on first occurrence and is **never silently re-sent** (silent conformance to §133) — which matters, since re-firing a succeeded withdraw would itself be a `422`, not a no-op.

**Piping / scripting**: on success, **stdout** carries the rendered `draft` proposal; an agent can parse `-o json` to read the reset `status`, the cleared `response_deadline`, and the updated `available_transitions` (now offering `propose`) and drive the next step (edit the changes / re-propose). On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 059 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic / output |
|---|---|---|---|---|
| Proposal withdrawn (`200`) | — | `Success` | 0 | the `draft` proposal rendered on stdout (status `draft`, deadline/responses cleared) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Transition not allowed (`422` — already a `draft`, or `withdraw` not in `available_transitions`) | `*ResponseError{422}` (→ `*ProblemError`, 015) | `APIError` | 3 | names `422` + extracted RFC 9457 detail; **not** folded into success (plan ADR-1) |
| Proposal not found / not visible (`404`) | `*ResponseError{404}` (→ `*ProblemError`, 015) | `APIError` | 3 | names `404` + detail; **not** folded into success (the inverse of discard) |
| Premium not enabled / not permitted (`401`/`403`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + detail; **no** plan-gate special-casing — the Premium `403` is a generic permission refusal (plan ADR-1; Plan-Limit Signalling is separate) |
| Rate limited (`429`) | `*ResponseError` (→ `*ProblemError`, 015) | `RateLimited` | 5 | names `429` + detail; `POST` is not auto-retried (§133) |
| Other non-2xx (`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + detail |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector / unparseable user template | `*output.FormatError` / template error (020/035) | `UsageError` | 2 | names the bad format value + the valid names (or the template parse failure); **no request sent** |
| Render of the returned proposal fails | render error (018/019) | `RuntimeError` | 1 | buffer-then-write leaves stdout empty; token-free |
| Unknown flag, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

**No** non-2xx status is folded into success — `404` and `422` are present as failure rows (plan ADR-1), the inverse of discard, where `404` is the one status diverted to success. Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Structural twin of the landed `propose` leaf** (plan ADR-1): `withdraw` is the second *transition* leaf under the `proposal` group (after 057's `propose`), built by mirroring `internal/cli/proposal_propose.go` onto the `/withdraw` sub-path. Like `propose` it is a **flagless** leaf keyed off the `prp_` id, posting to a **sub-resource path** (`/{id}/withdraw`), decoding `Document[Proposal]`, and rendering through the existing singular `proposal` path. The only behavioral differences from `propose` are the path segment, the result status (`draft`, not `proposed_outside_meeting`), and the destructive side-effect profile (ADR-2). No `proposal`-group redefinition.
- **Decode-and-render, not synthesize** (plan ADR-1, silent conformance to 057 §393): `withdraw` receives the full `draft` `Proposal` on the `200` and renders it through the **existing** singular `proposal` path (`Document[Proposal]` → `output.RenderSuccess` / `ProposalView`) — no synthesized result, **no new `glassfrog` model, no new render key**. It is the second proposal write to be a pure consumer of the model + render.
- **`404`/`422` as real failures — the inverse of discard** (plan ADR-1, silent conformance to 057 §396): `withdraw` intercepts nothing, so `404` (no such proposal) and `422` (transition not allowed — already `draft`, or `withdraw` not offered) both fail loudly with their status. A withdraw has no idempotent end-state (re-withdrawing a `draft` proposal is a genuine `422`), so neither is a success.
- **Server-authorized transition, no client pre-check** (plan ADR-1): the command does not pre-read `available_transitions` to gate the call — it issues the `POST` and lets the server's `422` enforce. `available_transitions` is surfaced by the reads (056) for the operator to inspect; invoking the transition is this command's job. A validation tripwire asserts exactly one request and no prior `GET`.
- **Destructive but unconfirmed** (plan ADR-2): unlike `propose` (which *sets* the deadline and records an implicit response), `withdraw` *deletes* the existing responses and *clears* the proposed timestamps. The command surfaces the returned `draft` proposal faithfully and adds **no** confirmation prompt, **no** `--force`/`--yes` flag, and **no** stderr advisory naming the deletion — the destructiveness is server-owned and legible in the returned data (zeroed `response_summary`, cleared deadline). This is the first destructive-but-unconfirmed governance write; operator-layer destructive-write confirmation is deferred to the separate **Write-Safety Guardrail** capability, which gates across the write path centrally (the same central-refinement shape as 054's `412` and Plan-Limit Signalling's `403`).
- **Premium `403` stays generic** (plan ADR-1): the transition is Premium-gated, but the `403` is a plain `PermissionError(4)` like any refusal — no bespoke "not available on your plan" message. Distinguishing a plan-gate `403` is the separate Plan-Limit Signalling capability.
- **No new machinery** (plan System Architecture): `withdraw` adds **no** transport surface (bodyless `POST`, `ContentType` empty), **no** `glassfrog` model (the `Proposal` is 055/056's), **no** render surface (the singular `proposal` path is 055/056's), and **no** new `Outcome`/`ExitCode` — it reuses `apiclient.Request`/`Execute` (010), the `proposalSeam`, `resolveRenderTarget`/`writeHuman`/`output.RenderSuccess` (020/035), `Document[Proposal]`, and `reportFailure`/`classifyClientError`/the frozen registry (011/015/032/054). The only net-new surface is the `withdraw` leaf and its pure `run` function.
- **Flag/command spellings** (`proposal`, `withdraw`) resolve the spec's confirmed surface; `withdraw` matches the API/lifecycle transition name and the exact string in `available_transitions`, continuing the 056/057 verb grammar. The `proposal withdraw` repetition is acknowledged (spec `[ASSUMED]`); it is adjustable at build time without changing behavior. No `accords/` directory exists, so there are no cross-spec accord patterns to align against — this conforms to the in-repo CLI precedent set by 042/043/044/045/056/057/058.
