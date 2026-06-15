# Interface Accord: Proposal Creation — CLI

**Feature**: 055-proposal-creation
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (`proposal` group + `create` leaf, reserves the namespace), ADR-2 (`--changes` resolved from reserved `stdin` / existing file / inline JSON behind a seam), ADR-3 (`type`-floor fail-fast, verbatim `[]json.RawMessage` pass-through), ADR-4 (`Proposal` response model + render). Output: Cross-cutting (single-resource `Document[Proposal]`, reuses 018/020). Transport: Cross-cutting (silent conformance to 042 `ContentType`; no `If-Match`). Retry: Cross-cutting (silent conformance to §133 `isSafeMethod`).

---

This accord pins the operator-facing proposal-creation surface: the `proposal` command group, its `create` leaf, the leaf's positional anchor and `--changes` flag, the rendered created-proposal output, and exit codes. The request seam it calls through is pinned in `010/interface-spec.md` (the write-body `Content-Type` from `042/interface-spec.md`); format selection/rendering in `018`/`019`/`020`/`035`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the CLI's **second write** and the anchor of the governance write path — it creates a `draft` proposal against a tension and produces it (with its `prp_` id) so a later step can advance it to circulation.

---

## Surface

### `glassfrog proposal` — command group (no action)

A guard-registered (001), non-runnable group: it carries **no action** and parents at least one child (`create`), so `glassfrog proposal` with no subcommand prints help. Mirrors the `tension` / `tension create` group/leaf shape (plan ADR-1, 042 ADR-2). The group is assembled with its child **before** being registered under root, so the registration guard's ">=1 child" rule holds at attach time. The `proposal` namespace hosts the rest of the write-flow and the reads (`list`/`get`/`propose`/`withdraw`/`respond`) — out of scope here (spec non-behaviors). The group itself is **shared with the concurrently-specified Proposal Reads (056)**: whichever of 055/056 lands first creates the group, and the follower attaches its leaf(s) to it (plan ADR-1; 043→042 relationship).

### `glassfrog proposal create <tension-id>` — create a draft proposal

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)` (zero or more than one positional is a usage error, no API call), a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `POST /proposals` (`createProposal`) and produces the created proposal as a single-resource result.

**Synopsis**:
```
glassfrog proposal create <TENSION_ID> --changes <SOURCE> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `TENSION_ID` | string | yes | The **anchor** tension the proposal is raised against (`ten_…`), sent as `proposal.tension_id`. Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`/`422`). |

**Write flags** (declared only on `create`):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--changes` | string | — | **Required.** The governance change set, a JSON array, resolved from one of three sources (plan ADR-2): the reserved keyword `stdin` (any casing) reads the array from piped standard input; a value that resolves to an **existing regular file** reads the array from that file; any other value is treated as the **inline JSON** array itself. A file literally named `stdin` is reachable as `./stdin`. A missing `--changes` is a usage error naming `--changes` as required, **before any request**. The array's elements are passed through **verbatim** above a `type` floor (see Interactions); the CLI reads only each element's `type`, never its other keys. |

The leaf does **not** expose any per-change-type flags or builders (free-form pass-through; typed builders are the deferred *Unguided Change Construction* problem), any proposer flag (the API derives the proposer from the token), any status flag (the API creates the proposal in `draft`), or any `--if-match`/concurrency flag (a create has no prior `ETag`).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020/035 | `full` (default) \| `compact` \| `json` \| `yaml` \| a template file \| `stdin`. |

**Output** (success, stdout): the created proposal is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured `{data: …}` document verbatim (018), `full`/`compact` render the human projection (019) through the **shared singular `proposal` render key**. The raw API envelope is never emitted under a human format. The singular `proposal` view is shared with Proposal Reads (056) under first-to-land-creates / follower-reuses (plan ADR-4): if 056's view has landed, the created proposal renders through it (changes by type, response summary, available transitions); if 055 lands first, it adds the thinner view sketched below, which 056 then grows. Either way the load-bearing `prp_` id and `draft` status are surfaced.

*`full`* (the thin 055-first view; superseded by 056's richer view if that lands first) — the created proposal's detail:
```
<prp_…>  [<status>]
  Tension:        <ten_… | (none)>
  Circle:         <role_… | (none)>
  Proposer:       <per_…/agt_… | (none)>
  Changes:        <N>
  Responses:      <no_objection>/<total> no-objection, <bring_to_meeting> bring-to-meeting
  Transitions:    <propose, withdraw | (none)>
  Created:        <created_at | (unknown)>
  Updated:        <updated_at | (unknown)>
```
Each nullable field (`tension_id`, `circle_id`, `proposer_id`) renders through an explicit-absence guard (`{{if .X}}…{{else}}(none){{end}}`) rather than a blank. `Changes` is the element count (the command does not project individual change bodies — 056's view renders them by `type`). On a successful create the server-set `status` is `draft` and `available_transitions` is typically `propose`.

*`compact`* — `<prp_…>  [<status>]  <N> change(s)` (the detail block omitted; the `prp_` id is always present — it is the load-bearing handle a later advance step references).

## Interactions

**Dispatch**: the `create` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here — `cobra.ExactArgs(1)` rejects both a missing and an extra positional before `RunE`); (2) `--output` resolution (020/035); (3) `--changes` presence (missing → usage error naming `--changes`); (4) change-source read (stdin/file/inline) and JSON parse; (5) the `type` floor (array, non-empty, every element an object with a non-empty `type`). Resolving `--output` first keeps error precedence consistent with the reads; the source read and floor are pure over the injected seam and pre-assembly, so any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/038/042).

**Request**: on valid input, the leaf marshals the body `{"proposal": {"tension_id": "<ten_…>", "changes": [ …verbatim… ]}}` (the change array carried byte-for-byte as `[]json.RawMessage`), and sends **one** `Execute` to `POST /proposals` with `Content-Type: application/json` (042 ADR-1; no command/transport change). There is no walk and no pagination — a single request. No `If-Match` header is sent. The `201` body `{data: Proposal}` decodes into `glassfrog.Document[Proposal]`; the structured path serializes the raw payload verbatim.

**Premium gate**: `POST /proposals` requires the async-proposals Premium feature. The command performs **no client-side feature check** — it always issues the request and surfaces a server `403` through the shared classifier as a permission outcome (spec non-behavior; distinct plan-limit signalling is deferred to 060/061).

**Non-idempotent retry**: `create` is built on the same `RetryExecutor` as the reads, but 017's `isSafeMethod` restricts 429 auto-retry to `GET`/`HEAD`, so a `POST` surfaces the `429` on first occurrence and is **never silently re-sent** — creation cannot double-submit a proposal on a rate-limit retry (silent conformance to §133, no command-side special-casing).

**Change-set sourcing**: `--changes stdin` requires a piped array and rejects a terminal stdin or an empty pipe (the 035 TTY/empty fail-fast, reusing the 006 bounded reader); a file source is read cwd-relative and rejected if unreadable or not a regular file; an inline value is the array bytes themselves. All three feed the same JSON parse + `type` floor.

**Piping / scripting**: on success, stdout carries only the rendered created proposal. On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020/035 chain) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 055 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Proposal created | — | `Success` | 0 | — (created proposal on stdout) |
| Missing or extra `<tension-id>` positional (0 or >1) | — (cobra `ExactArgs(1)`) | `UsageError` | 2 | cobra's arg-count message; **no request sent** |
| Missing `--changes` | — (local) | `UsageError` | 2 | "`--changes` is required"; **no request sent** |
| `--changes` not valid JSON, or not a JSON array | — (local parse) | `UsageError` | 2 | names the source (inline / file path / stdin) and that a JSON array is expected; **no request sent** |
| `--changes` is an empty array | — (local) | `UsageError` | 2 | "at least one change is required"; **no request sent** |
| A change element is not an object, or lacks a non-empty `type` | — (local floor) | `UsageError` | 2 | "every change must carry a `type`"; **no request sent** |
| `--changes stdin` with a terminal / empty pipe; or an unreadable / non-regular file | — (seam) | `UsageError` | 2 | names the source; for stdin, how to pipe; **no request sent** |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Async proposals not enabled (`403`), or other permission (`401`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + extracted RFC 9457 detail (e.g. "async proposals not enabled") |
| Unknown anchor tension (`404`), rejected change/field (`422`), or other `4xx`/`5xx` | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + extracted RFC 9457 detail (015) |
| Rate limited (`429`) | `*ResponseError` | `RateLimited` | 5 | names the status; next step to retry later (POST not auto-retried) |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector (env/file rung) | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the valid names |
| Unknown flag | — (cobra) | `UsageError` | 2 | cobra's unknown-flag message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. A `403` (Premium) classifies as `PermissionError`(4) with the server's detail surfaced — the command adds no interpretation. The token value never appears in any message.

## Consistency Notes

- **Second write, reuses the read+write chain** (plan System Architecture): `create` is the single-resource shape run with a body-carrying `POST`, exactly like Tension Capture (042). It reuses the persistent `--base-url` (011) and `--output`/`-o` (020/035), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `RetryExecutor` (017), the landed `apiclient.Request.ContentType` (042 — **no transport change**), `output.RenderSuccess` (structured) + `render.Render` (human), and 032's `reportFailure` chokepoint — all unchanged. 055's net-new machinery is the `--changes` source resolver, the `type` floor, and the write request-input; the `proposal` group, `Proposal` model, and singular `proposal` render are **shared with 056** (next note).
- **Concurrent sibling 056 — shared group/model/render** (plan Cross-cutting + ADR-1/ADR-4): the `proposal` group, the `glassfrog.Proposal` model, and the singular `proposal` render are shared with the concurrently-specified Proposal Reads (056) under **first-to-land-creates / follower-reuses-or-grows** (043→042). As cut from current `main` (no proposal code present), 055 creates them; if 056 lands first, 055 attaches its `create` leaf to the existing group and reuses 056's model + richer singular view unchanged.
- **Group namespace, not a flat command** (plan ADR-1): the `proposal` group hosts a `proposal <verb>` family — `create` (055), `list`/`get` (056), and later `propose`/`respond`/`withdraw`; `--changes` lives only on `create`, so a sibling leaf (e.g. `proposal get`) passing `--changes` is a cobra unknown-flag usage error for free.
- **`--changes` source mirrors the template source** (plan ADR-2): the reserved-`stdin`-keyword + file-or-inline resolution reuses 035's reserved-name idiom and the 006 bounded stdin reader, **diverging** by adding an existing-file check and an inline-JSON form (a template has no inline form). `<tension-id>` passes through to a clean `404`/`422` (§200), like 042/044's id handling.
- **`type` floor, verbatim pass-through** (plan ADR-3): the only client-side check on the change set is "valid JSON array, non-empty, each element an object with a non-empty `type`" — the one key the schema requires. Every command-specific key rides through untouched; the server validates values. This is deliberately *not* a typed input fork (contrast 044) because the API is schema-free for changes.
- **No `--status` / no proposer / no `--if-match`** (spec non-behaviors): the API owns the `draft` status and the proposer (derived from the token), and a create has no prior `ETag`, so the command exposes none of these.
- **Flag/command spellings** (`proposal`, `create`, `--changes`) resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. The plural-list/singular pattern of the reads does not apply — this is a verb leaf under a noun group.
- **Command conventions** follow 001/003: the group and leaf register through the fail-loud guard, are explicitly wired in `Assemble`, the leaf declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
