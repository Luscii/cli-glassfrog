# Interface Accord: Proposal Reads — CLI

**Feature**: 056-proposal-reads
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (create the `proposal` group via `newProposalCommand`; two verb leaves `proposal list` (global, `NoArgs`) + `proposal get <prp-id>` (`ExactArgs(1)`); list-only flags structurally enforced), ADR-2 (establish `glassfrog.Proposal` + free-form `ProposalChange` + `ResponseSummary`; add both the plural `proposals` and singular `proposal` render paths), ADR-3 (`--status` validated locally via a new `validateProposalStatus`; the four other filters and the id passed through). Completeness: Cross-cutting (reuses 025 ADR-3). Coordination: shares the group/model/singular-render with the concurrent Proposal Creation (055) — first-to-land creates, follower reuses.

---

This accord pins the operator-facing proposal read surface: the two leaves `proposal list` (global list) and `proposal get` (standalone read) under a new `proposal` group, their flags, the rendered output, completeness signalling, and exit codes. Because no proposal code exists yet, this feature (as cut from current `main`) creates the `proposal` group, the `glassfrog.Proposal` model, and both the plural `proposals` and singular `proposal` render keys — all shared with the concurrently-specified Proposal Creation (055). The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the read half of the Governance Proposals write flow — it surfaces the proposals in flight and reads one by its `prp_` id with its changes, response summary, and available transitions.

---

## Surface

### `glassfrog proposal` — command group (no action)

A new non-runnable group created by this feature (plan ADR-1) via `newProposalCommand(seam)` (the `newTensionCommand` shape): it carries no action and parents its children (`list` and `get` added here; `create` added by 055; later `propose`/`withdraw`/`respond`), so `glassfrog proposal` with no subcommand prints help. Leaves attach through the registration guard (001). **Coordination**: whichever of 055/056 lands first defines this group; the follower attaches its leaves to the existing group (043's relationship to 042's group).

### `glassfrog proposal list` — list the proposals visible to the caller

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.NoArgs`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage`. Reads `GET /proposals` (`listProposals`) and produces the proposals visible to the caller as a list result. **Takes no positional** — unlike `tension list <role-id>`, the list is global; the circle is the optional `--role-id` filter. Walks to completion by default (see Interactions).

**Synopsis**:
```
glassfrog proposal list [--status STATUS] [--role-id ROLE_ID] [--proposer-id PERSON_ID]
                        [--proposed-after TIMESTAMP] [--accepted-after TIMESTAMP]
                        [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

`proposal list` accepts **no positional argument**; any positional is a usage error (cobra `NoArgs`), no API call.

**List flags** (declared only on `list`):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--status` | string | — | Filter by proposal status, sent as the `status` parameter. **Validated locally** before any request against the proposal status set (`draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts`); an unsupported value is a usage error naming the value and the supported set, no API call (plan ADR-3, a new `validateProposalStatus` distinct from the action/project `validateStatus` and the tension `validateTensionStatus` sets). Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); `--status ""` behaves as no filter. |
| `--role-id` | string | — | Filter to proposals in the circle supported by this role, sent as `role_id`. **Passed through unvalidated** (free identifier); a malformed value surfaces as the API's clean `400`. Sent only when present and non-empty. |
| `--proposer-id` | string | — | Filter to proposals created by this person, sent as `proposer_id`. Passed through unvalidated; sent only when present and non-empty. |
| `--proposed-after` | string | — | Only proposals with `proposed_at` on or after this timestamp, sent as `proposed_after`. Passed through unvalidated (the API validates the timestamp shape); sent only when present and non-empty. |
| `--accepted-after` | string | — | Only proposals with `accepted_at` on or after this timestamp, sent as `accepted_after`. Passed through unvalidated; sent only when present and non-empty. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

Supplied filters **combine** (each sent as its own query parameter). With no filter, every proposal visible to the caller is requested.

### `glassfrog proposal get <prp-id>` — read a single proposal

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /proposals/{id}` (`getProposal`) and produces the single proposal with full detail — its `changes`, aggregate `response_summary`, and `available_transitions`.

**Synopsis**:
```
glassfrog proposal get <PRP_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `PRP_ID` | string | yes | The proposal to read (`prp_…`). Exactly one required (cobra `ExactArgs(1)`). Passed through unvalidated; an unknown or invisible id → `404`. |

`get` declares **no list flags**. Passing `--status`, `--role-id`, `--proposer-id`, `--proposed-after`, `--accepted-after`, `--first-page`, or `--per-page` to it is rejected by cobra's unknown-flag handling as a usage error before any request — this is how the spec's "filters apply only to the list" is enforced (no hand-rolled cross-combo guard; plan ADR-1).

**Inherited (persistent root) flags**, read by cobra inheritance on both leaves, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml` (widened by 035 to also accept a user-template ref). |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*List (`proposal list`)* — the **new plural `proposals` render key** (plan ADR-2): `full` renders one block per proposal (`<prp_…>  [<status>]`, then indented `proposer: <per_…|agt_…|—>`, `changes: <N>`, and a one-line `responses: <total> total (<no_objection> no-objection, <bring_to_meeting> bring-to-meeting)` summary); `compact` renders one line per proposal (`<prp_…>  [<status>]  <N changes>`). For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each proposal's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope and **not** a decode-and-re-encode of `[]Proposal`.

*Single (`proposal get`)* — the **new singular `proposal` render key** (plan ADR-2). `full` renders the full single-proposal detail:
```
<prp_…>  [<status>]
  Tension:        <ten_… | (none)>
  Circle:         <role_… | (none)>
  Proposer:       <per_… | agt_… | (none)>
  Proposed:       <proposed_at | (none)>
  Deadline:       <response_deadline | (none)>
  Accepted:       <accepted_at | (none)>
  Responses:      <total> total — <no_objection> no-objection, <bring_to_meeting> bring-to-meeting
  Expected/recv:  <expected_response_count> / <received_response_count>
  Transitions:    <propose, withdraw | (none)>
  Changes (<N>):
    - [<change type>] <change body / properties, rendered verbatim>
    …
```
Each nullable field renders through an explicit-absence guard rather than a blank; each change's free-form properties are rendered **verbatim**, never truncated or reflowed (CONSTITUTION VI). `compact` — `<prp_…>  [<status>]  <N changes>  <total> responses` (the detail block omitted).

**Empty list** (no proposals visible, or none matches the filters): under `full`/`compact`, stdout is exactly the `proposals` template's empty line (`no proposals`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: each leaf has its own `RunE` (no `len(args)` branching). Before any network call, in order: (1) cobra `Args`/flag parsing (a list-only flag on `get`, or any positional on `list`, fails here); (2) `--output` resolution (020); (3) `--status` validation on `list` (the one closed-enum input — `validateProposalStatus`). Output resolution precedes status validation to keep error precedence consistent with the sibling reads — both are pure, pre-assembly checks, so either order keeps the no-request guarantee, but resolving `--output` first means an invalid `--output` is reported even when `--status` is also invalid. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/014/025/038/043).

**List completeness** (`proposal list`; reuses 025 ADR-3 verbatim):
- **Default** — the command walks every page via `paging.All` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more proposals exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the proposals gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero**:
  ```
  note: result is incomplete — <cause>; the proposals shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario). The single `proposal get` read is unpaginated — no completeness signalling.

**Request**: `list` walks `GET /proposals` (sending each filter only when supplied); `get` issues **one** `Execute` to `GET /proposals/{id}` and decodes `glassfrog.Document[Proposal]`. The `prp_` id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (ADR-3). Filter values are sent as query parameters, server-validated. All reads are bodyless `GET`s — no write-body `Content-Type`, no `If-Match` (proposal reads are not guarded writes).

**Premium gating**: proposal **reads are not Premium-gated** (only the write transitions are). No plan-gate `403` is expected; if one arrives it is classified generically as `PermissionError(4)` through the shared chain — there is no proposal-read-specific plan-limit handling (that signal belongs to the write path's Plan-Limit Signalling, out of scope).

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 056 reuses `reportFailure` unchanged. Both leaves **introduce no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Proposals listed / proposal read (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown/invisible proposal id (`get`), malformed filter (`list`, `400`), or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk (`list`) | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text — "malformed paging: the cursor did not advance at page N" — surfaced inside the incompleteness note (the partial set is already on stdout) |
| Unsupported `--status` value (`list`) | — (`validateProposalStatus`) | `UsageError` | 2 | "unsupported --status value \"…\" — supported: accepted, draft, draft_with_conflicts, escalated, proposed_outside_meeting"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| List-only flag on `get`, or a positional on `list`, or wrong positional count on `get` (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; these commands benefit with no edit. The token value never appears in any message.

## Consistency Notes

- **Read half of the proposal write flow** (plan System Architecture): `proposal list`/`proposal get` are the structural twin of Tension Reads' `tension list`/`tension get` (043), run as verb leaves under the `proposal` group rather than a plural/singular noun pair — because the proposal surface is verb-rich (create/propose/withdraw/respond) and a bare `proposal <prp-id>` would collide with subcommand dispatch (the cobra constraint 038 ADR-1 cites). They reuse the persistent `--base-url` (011) and `--output`/`-o` (020/035), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), the 035-widened render flow — `resolveRenderTarget` then `aggregateRawData` (structured) / `writeHuman` over the projection (human) for the list, and `output.RenderSuccess` (structured) / `writeHuman` (human) for the single read — and 032's `reportFailure` chokepoint, all unchanged (the exact pattern 043's `runTensionList`/`runTensionGet` use).
- **Global list, not role-scoped** (plan ADR-1): `proposal list` is the CLI's first global (non-positional) paginated list alongside the `me`-family self-service reads — `cobra.NoArgs`, the circle as an optional `--role-id` filter. `paging.All` over a query-only descriptor (no path id) is already supported; this is a usage difference from `tension list <role-id>`, not a walker change. `proposal get` is `ExactArgs(1)` (mirrors 043's `tension get`).
- **Verb leaves under the group enforce list-only-ness structurally** (plan ADR-1): the five filters + `--first-page`/`--per-page` simply don't exist on `get`, so cobra's unknown-flag handling does the rejecting — no hand-rolled cross-combo guard.
- **One local validator, a new set** (plan ADR-3): `--status` is the single closed-enum input, validated by a **new `validateProposalStatus`** over the proposal status set (`draft`, `proposed_outside_meeting`, `escalated`, `accepted`, `draft_with_conflicts` — note `draft_with_conflicts`, which the FEATURE-MODEL prose omits), placed **with the proposal command code** beside the `validateTensionStatus`/`validateMeetingType` precedent (NOT in the shared `internal/cli/status.go`) — and **not** reusing `validateStatus`/`validateTensionStatus`, whose vocabularies are wrong for proposals. The four other filters and the `prp_`/`role_`/`per_` ids are free identifiers passed through to a clean `400`/`404`. (025 ADR-4 principle: validate where the API would silently mislead; pass through where it reports cleanly.)
- **New model + both render keys** (plan ADR-2): this feature establishes `internal/glassfrog.Proposal` (with a free-form `ProposalChange` keeping `id`/`type` + a `map[string]any` remainder — the `actions.go` precedent — and an aggregate-only `ResponseSummary`), and instantiates the landed generics: the list walks `Page[json.RawMessage]` (structured) / `Page[Proposal]` (human), the single read decodes `Document[Proposal]`. It adds **both** render keys — the plural `proposals` (a `ProposalsView` mirroring `ProjectsView`) and the singular `proposal` (a `ProposalView`), each registered in `builtinResources` so the exhaustiveness guard covers them. Structured `json`/`yaml` output needs no machinery change (018 ADR-2 serializes raw bytes — faithful even for fields the model omits). The `ResponseSummary` exposes counts only — no per-person field exists, enforcing the anti-attribution non-behavior at the type level.
- **Coordination with Proposal Creation (055)** (plan Risks): the `proposal` group, the `Proposal` model, and the singular `proposal` render are shared with the concurrent 055. First-to-land creates them; the follower attaches its leaves and grows (never duplicates) the model/singular render. If 055 lands first with a thinner singular render, 056 grows it to surface `changes`/`response_summary`/`available_transitions` — a tasks-stage branch point, not a contract change. Structured output is faithful regardless (raw-bytes pass-through).
- **Flag/command spellings** (`proposal list`, `proposal get`, `--status`, `--role-id`, `--proposer-id`, `--proposed-after`, `--accepted-after`) resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. The plural-list/singular noun pattern of 038 does not apply — these are verb leaves under a noun group (the verb-rich-noun grammar 042/043 established).
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
