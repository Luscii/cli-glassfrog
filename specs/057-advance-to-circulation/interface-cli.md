# Interface Accord: Advance to Circulation — CLI

**Feature**: 057-advance-to-circulation
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (`propose <prp-id>` flagless leaf on the `proposal` group; bodyless `POST` to the `/propose` sub-path; id passed through), ADR-2 (decode the returned `{data: Proposal}` and render through the shared singular `proposal` path — decode, NOT synthesize; no new `glassfrog` model, no new render surface), ADR-3 (server-authorized transition — no client pre-check of `available_transitions`; `404`/`422` are REAL FAILURES, the inverse of discard's `404`-as-success; the Premium `403` routes through the shared classifier with no plan-gate special-casing). Cross-cutting: sibling coordination (the `proposal` group/model/render are 055/056's — first-to-land-creates, follower-reuses); retry (silent conformance to §133 `isSafeMethod` — `POST` not auto-retried on `429`).

---

This accord pins the operator-facing advance-to-circulation surface: the new flagless `propose` leaf under the `proposal` group, the bodyless `POST /proposals/{id}/propose`, the decode-and-render of the advanced proposal, and exit codes. The `proposal` group, the `glassfrog.Proposal` model + `Document[Proposal]` decode, the singular `proposal` render path, the registration guard, format selection/rendering (`018`/`019`/`020`/`035`), the request seam (`010`, incl. the optional `ContentType` 042 added — here left empty), the shared failure chokepoint (`reportFailure`/`classifyClientError`, `015`/`031`/`032`), and the frozen `Outcome`/`ExitCode` registry (`004`/`011`/`054`) are all landed (the proposal-specific ones by Proposal Reads 056 / Proposal Creation 055); this feature adds **one leaf** and reuses everything else — it is the first proposal *write* to be a pure consumer of the model and render path. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the **`propose` transition verb of the `proposal` family** — it advances one `draft` proposal into circulation by id, and is the first command to issue a bodyless `POST` that also decodes a response body.

---

## Surface

### `glassfrog proposal` — command group (no action)

The non-runnable group landed by Proposal Reads (056) / Proposal Creation (055) (plan 056 ADR-1): it carries no action and parents its children (`list`/`get` from 056; `create` from 055; `propose` added here), so `glassfrog proposal` with no subcommand prints help. This feature attaches one new leaf to the existing group through the registration guard (001); it does not redefine the group. The group `Short` widens to name `propose` alongside the rest (a `Short` that omitted it would make `glassfrog proposal --help` misleading). **Sibling coordination** (plan Cross-cutting): the group, the `Proposal` model, and the singular `proposal` render path are created by whichever of 055/056 lands first; 057 reuses them. If 057 somehow lands first, it creates the minimal subset it needs (the group + seam, the model + `Document[Proposal]`, the singular render only).

### `glassfrog proposal propose <prp-id>` — advance a draft proposal into circulation

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `POST /proposals/{proposal_id}/propose` (`proposeProposal`) and produces the advanced proposal the server returns.

**Synopsis**:
```
glassfrog proposal propose <PRP_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `PRP_ID` | string | yes | The proposal to advance (`prp_…`), used as the `POST /proposals/{proposal_id}/propose` path id. Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it — and an unknown or invisible id's `404` is a **failure** here (plan ADR-3), unlike discard's `404`-as-success. |

**Flags**: **none.** A transition advances a proposal whole — there is nothing to edit and the endpoint takes no request body. `propose` declares no flags of its own, so a stray `--status`/`--changes`/`--body` is a cobra unknown-flag usage error for free (the structural guard 045/056 rely on). The leaf exposes no confirmation/`--force`/`--yes` flag (spec non-behavior — the CLI is non-interactive and agent-driven; the advance is reversible via Withdraw) and no `--force-transition`/pre-check affordance (the server authorizes the transition; the command does not pre-read `available_transitions`, plan ADR-3).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 (widened by 035) | `full` (default) \| `compact` \| `json` \| `yaml`, or a user-template ref (`@file`/`-`). |

**Output** (success, stdout): the **advanced proposal the server returned** on the `200` — the full `Proposal`, now `status: proposed_outside_meeting`, carrying the server-set `response_deadline`, the proposer's auto-recorded implicit `no_objection` reflected in `response_summary`/`received_response_count`, and the updated `available_transitions` (typically `propose` is gone, `withdraw` now present). It is decoded into the shared `glassfrog.Proposal` (`Document[Proposal]`) and rendered by Output Format Selection (020, widened by 035) in the effective format, through the **same singular `proposal` render path `proposal get` uses** — no new render key. The command **synthesizes nothing** (plan ADR-2) and **narrates none of the side effects** (notifications, the deadline computation, the implicit response) — they are visible only through the returned data.

*`json`* / *`yaml`* — the raw `{data: Proposal}` document, serialized verbatim (018, faithful regardless of model field coverage):
```json
{
  "data": {
    "id": "prp_0123456789abcdef0123456789abcdef",
    "status": "proposed_outside_meeting",
    "tension_id": "ten_0123456789abcdef0123456789abcdef",
    "circle_id": "role_0123456789abcdef0123456789abcdef",
    "proposer_id": "per_0123456789abcdef0123456789abcdef",
    "proposed_at": "2026-06-15T12:00:00Z",
    "response_deadline": "2026-06-22T12:00:00Z",
    "changes": [ { "type": "CreateRole", "name": "Scheduling" } ],
    "response_summary": { "total": 1, "no_objection": 1, "bring_to_meeting": 0 },
    "expected_response_count": 5,
    "received_response_count": 1,
    "available_transitions": [ "withdraw" ]
  }
}
```

*`full`* / *`compact`* (or a selected user template, 035) — the human projection (019) over the existing singular `proposal` render key (056/055): `full` shows id, the new `proposed_outside_meeting` status, the anchoring `tension_id`/`circle_id`/`proposer_id`, the lifecycle timestamps incl. the `response_deadline`, `changes` by type (body verbatim — CONSTITUTION VI), the `response_summary` counts, expected/received counts, and `available_transitions`; `compact` is the one-line projection (id + status + change-count + a one-line response summary). The exact field selection and template text are 056/055's interface contract, reused unchanged.

The command introduces **no new field names, no new render key, and no new template** — it renders the same `Proposal` shape the reads surface.

## Interactions

**Dispatch**: the `propose` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here); (2) `--output` resolution (020/035) — a present-but-invalid selector, or an unparseable/empty user-template source, fails fast as a usage error with **no request**. There are no input-field checks (the leaf has no flags). Resolving `--output` first keeps error precedence consistent with the siblings; a transport tripwire asserts the no-request guarantee on the bad-`--output` path (per 045/056).

**Request**: on valid input, the leaf assembles the connection and sends **one** `Execute` to `POST /proposals/{proposal_id}/propose` with **no body and no `Content-Type`** (the optional `ContentType` seam 042 added is left empty — a bodyless `POST`, the discard `DELETE` shape, not the create body shape) and **`out == &Document[Proposal]`** (the `200` carries the advanced proposal to decode — the create `POST`+decode shape, combined here with discard's bodyless request). There is **no prior `GET`** — the command does not pre-read the proposal to check `available_transitions` (plan ADR-3); a validation tripwire asserts exactly one request and no read. **No `If-Match` header is sent** (`propose` is a transition with no `If-Match` parameter; Guarded Writes' `Request.IfMatch` stays unused — 053). The `prp_` id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (plan ADR-1).

**Outcome interpretation** (plan ADR-2/ADR-3): the leaf branches on the `Execute` result —
- **`err == nil`** (a `2xx`, i.e. `200`): success. Decode the returned `Document[Proposal]` and render it on stdout (machine: `output.RenderSuccess` over the raw bytes; human/`-o template`: `writeHuman` over the decoded `ProposalView`, buffer-then-write) and exit `Success` (0).
- **any error** — including a `404` (no such/invisible proposal) and a `422` (transition not allowed): route to the shared `reportFailure` **with no status interception** (the exact inverse of discard's `errors.As`-for-`404`-as-success divert). See Error Communication.

No status is folded into success; an advance has no idempotent re-run, so `404`/`422` are real failures the operator must see. There is **no stderr advisory** (unlike discard's `204`-vs-`404` note — there is a single success outcome here).

**Non-idempotent retry**: `propose` is built on the same `RetryExecutor` as the rest, but 017's `isSafeMethod` restricts `429` auto-retry to `GET`/`HEAD`, so a `POST` surfaces a `429` on first occurrence and is **never silently re-sent** (silent conformance to §133) — which matters, since re-firing a succeeded advance would itself be a `422`, not a no-op.

**Piping / scripting**: on success, **stdout** carries the rendered advanced proposal; an agent can parse `-o json` to read the new `status`, `response_deadline`, and `available_transitions` and drive the next step (respond / withdraw). On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 057 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic / output |
|---|---|---|---|---|
| Proposal advanced (`200`) | — | `Success` | 0 | the advanced `{data: Proposal}` rendered on stdout (now `proposed_outside_meeting`) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Transition not allowed (`422` — not a `draft`, or `propose` not in `available_transitions`) | `*ResponseError{422}` (→ `*ProblemError`, 015) | `APIError` | 3 | names `422` + extracted RFC 9457 detail; **not** folded into success (plan ADR-3) |
| Proposal not found / not visible (`404`) | `*ResponseError{404}` (→ `*ProblemError`, 015) | `APIError` | 3 | names `404` + detail; **not** folded into success (the inverse of discard) |
| Premium not enabled / not permitted (`401`/`403`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + detail; **no** plan-gate special-casing — the Premium `403` is a generic permission refusal (plan ADR-3; Plan-Limit Signalling is separate) |
| Rate limited (`429`) | `*ResponseError` (→ `*ProblemError`, 015) | `RateLimited` | 5 | names `429` + detail; `POST` is not auto-retried (§133) |
| Other non-2xx (`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + detail |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector / unparseable user template | `*output.FormatError` / template error (020/035) | `UsageError` | 2 | names the bad format value + the valid names (or the template parse failure); **no request sent** |
| Render of the returned proposal fails | render error (018/019) | `RuntimeError` | 1 | buffer-then-write leaves stdout empty; token-free |
| Unknown flag, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

**No** non-2xx status is folded into success — `404` and `422` are present as failure rows (plan ADR-3), the exact inverse of discard, where `404` is the one status diverted to success. Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Transition verb on the `proposal` family** (plan ADR-1): `propose` is the first *transition* leaf under the `proposal` group (056 ADR-1), after 056's `list`/`get` and 055's `create`. Like discard it is a **flagless** leaf, so any flag is a cobra unknown-flag usage error for free (the structural guard 034/038/042/043/045/056 rely on). It is keyed off the `prp_` id and posts to a **sub-resource path** (`/{id}/propose`), not the collection path the reads/creation use. No `proposal`-group redefinition.
- **Decode-and-render, not synthesize** (plan ADR-2): unlike discard (045), whose bodyless `204` forced a synthesized `{data:{id,discarded}}` over a new render key, `propose` receives the full advanced `Proposal` on the `200` and renders it through the **existing** singular `proposal` path (`Document[Proposal]` → `output.RenderSuccess` / `ProposalView`). The two writes bracket the write-output design space: echo-a-resource decodes-and-renders (057); bodyless-success synthesizes (045). The first proposal write that is a pure consumer of the model + render — **no new `glassfrog` model, no new render key**.
- **`404`/`422` as real failures — the inverse of discard** (plan ADR-3): every prior command except discard routes all non-2xx through the shared classifier; discard alone intercepts `404` as success. `propose` restores the default — it intercepts nothing, so `404` (no such proposal) and `422` (transition not allowed) both fail loudly with their status. An advance has no idempotent end-state (re-proposing a circulating proposal is a genuine `422`), so neither is a success.
- **Server-authorized transition, no client pre-check** (plan ADR-3): the command does not pre-read `available_transitions` to gate the call — it issues the `POST` and lets the server's `422` enforce. `available_transitions` is surfaced by the reads (056) for the operator to inspect; invoking the transition is this command's job. A validation tripwire asserts exactly one request and no prior `GET`.
- **Premium `403` stays generic** (plan ADR-3): the transition is Premium-gated, but the `403` is a plain `PermissionError(4)` like any refusal — no bespoke "not available on your plan" message. Distinguishing a plan-gate `403` is the separate Plan-Limit Signalling capability, which will refine the `403` centrally (as 054 refined the `412` for all writes), not per-command here.
- **No new machinery** (plan System Architecture): `propose` adds **no** transport surface (bodyless `POST`, `ContentType` empty; the bodyless-`POST`+decode combination needs no new field — `Method` is a free string, `ContentType` optional, `Execute` decodes `out`), **no** `glassfrog` model (the `Proposal` is 055/056's), **no** render surface (the singular `proposal` path is 055/056's), and **no** new `Outcome`/`ExitCode` — it reuses `apiclient.Request`/`Execute` (010), the `proposalSeam`, `resolveRenderTarget`/`writeHuman`/`output.RenderSuccess` (020/035), `Document[Proposal]`, and `reportFailure`/`classifyClientError`/the frozen registry (011/015/032/054). The only net-new surface is the `propose` leaf and its pure `run` function.
- **Flag/command spellings** (`proposal`, `propose`) resolve the spec's confirmed surface; `propose` matches the API/lifecycle transition name and the exact string in `available_transitions`, continuing the 056 verb grammar. The `proposal propose` repetition is acknowledged (spec `[ASSUMED]`); it is adjustable at build time without changing behavior. No `accords/` directory exists, so there are no cross-spec accord patterns to align against — this conforms to the in-repo CLI precedent set by 042/043/044/045/056.
