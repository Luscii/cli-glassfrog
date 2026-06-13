# Interface Accord: Tension Update — CLI

**Feature**: 044-tension-update
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (new `TensionUpdateInput` — all fields `omitempty`, incl. `status`), ADR-2 (`update <ten-id>` leaf on 042's `tension` group; reuse landed `validateTensionStatus` (043) + `validateMeetingType` (042); id passed through), ADR-3 (at-least-one editable field fail-fast; blank `--body` rejected only when supplied; both pure pre-assembly). Output: Cross-cutting (single-resource `Document[Tension]`, reuses 018/020/035). Retry: Cross-cutting (silent conformance to §133 `isSafeMethod`).

---

This accord pins the operator-facing tension-update surface: the new `update` leaf under the `tension` group, its editable-field flags, the rendered updated-tension output, and exit codes. The `tension` group, the `glassfrog.Tension` model, the `Document[Tension]` envelope, and the singular `tension` render key are landed by Tension Capture (042); this feature adds one leaf and one request-input shape (`044/interface-spec.md`) and reuses everything else. The request seam it calls through is pinned in `010/interface-spec.md` (the write-body `ContentType` was added by 042 and is reused unchanged here); format selection/rendering in `018`/`019`/`020`/`035`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the **edit verb of the `tension` family** — it changes the fields of an existing tension by id, including the explicit `archived` transition capture deliberately withheld.

---

## Surface

### `glassfrog tension` — command group (no action)

The non-runnable group landed by 042 (plan ADR-2): it carries no action and parents its children (`create` from 042; `list`/`get` from 043; `update` added here), so `glassfrog tension` with no subcommand prints help. This feature attaches one new leaf to the existing group through the registration guard (001); it does not redefine the group. The group `Short` widens to name the edit alongside capture and the reads (a write-only or read-only `Short` would make `glassfrog tension --help` misleading).

### `glassfrog tension update <ten-id>` — edit an existing tension

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `PATCH /tensions/{id}` (`updateTension`) and produces the updated tension as a single-resource result.

**Synopsis**:
```
glassfrog tension update <TEN_ID> [--body TEXT] [--label TEXT] [--status unprocessed|processed|archived] [--meeting-type tactical|governance] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `TEN_ID` | string | yes | The tension to edit (`ten_…`), used as the `PATCH /tensions/{id}` path id. Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**Editable-field flags** (all optional; declared only on `update`; at least one must result in a sent field — see Interactions):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--body` | string | — | New tension text, sent as `tension.body`. Optional here (unlike `create`), but when **supplied** a value that is empty or only whitespace is a usage error naming `--body`, **before any request** — a tension cannot be blanked out (mirrors capture's blank-body rejection; this is not a clear-to-null). Free text, passed through verbatim — never truncated, reflowed, or parsed (CONSTITUTION I/VI). |
| `--label` | string | — | New short label, sent as `tension.label`. Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); an omitted or empty `--label` is omitted from the body (no clear-to-null). Free text, passed through. |
| `--status` | string | — | New status, sent as `tension.status`. **Validated locally** before any request against the tension status set (`unprocessed`, `processed`, `archived`); an unsupported value is a usage error naming the value and the supported set, no API call (plan ADR-2, reuses 043's `validateTensionStatus`). Unlike `create`, status **is** editable here — the API allows the transition (e.g. to `archived`) on `PATCH`, then re-runs its own auto-computation on save. Sent only when present and non-empty. |
| `--meeting-type` | string | — | New routing hint, sent as `tension.meeting_type`. **Validated locally** before any request against the closed set (`tactical`, `governance`); an unsupported value is a usage error naming the value and the supported set, no API call (plan ADR-2, reuses 042's `validateMeetingType`). Sent only when present and non-empty. |

The leaf exposes **no** `sensed_by`/person flag and **no** flag to clear a field to null (spec non-behaviors): the API derives the sensing person from the token, the sensing role is fixed at capture, and an editable flag only *sets* a value — an empty value is omitted, never sent as `null`.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 (widened by 035) | `full` (default) \| `compact` \| `json` \| `yaml`, or a user-template ref (`@file`/`-`). |

**Output** (success, stdout): the updated tension is rendered by Output Format Selection (020, widened by 035) in the effective format — `json`/`yaml` emit the structured `{data: …}` document verbatim (018), `full`/`compact` (or a selected user template) render the human projection (019) over the **landed `tension` render key** (042). The raw API envelope is never emitted under a human format. The rendered `status` is whatever the server returns after its recompute — the command claims no authority over the final value (spec non-behavior).

*`full`* / *`compact`* — identical to the `tension` key landed by 042 (it is reused, not redefined):
```
<ten_…>  [<status>]
  Body:          <body>
  Label:         <label | (none)>
  Sensing role:  <role_… | (none)>
  Sensed by:     <per_… | (none)>
  Meeting type:  <tactical | governance | (none)>
  Parent role:   <role_… | (none)>
  Created:       <created_at | (unknown)>
  Updated:       <updated_at | (unknown)>
```
`compact` renders `<ten_…>  [<status>]  <body>`.

## Interactions

**Dispatch**: the `update` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here); (2) `--output` resolution (020/035); (3) if `--body` was supplied (`cmd.Flags().Changed`), reject a whitespace-only value as a usage error naming `--body` — the specific message runs first so `--body "   "` is not swallowed by the generic precondition; (4) `--status` validation against the tension status set; (5) `--meeting-type` validation against its set; (6) the **at-least-one-editable-field** precondition — the resolved send-set (presence `Changed` AND non-empty value, per the capture rule extended to body and status) must be non-empty, else a usage error naming the four flags. Resolving `--output` first keeps error precedence consistent with the siblings; the input checks are pure and pre-assembly, so the no-request guarantee holds regardless of their relative order (a transport tripwire asserts it, per 042/043).

**Request**: on valid input, the leaf marshals the partial body `{"tension": { … only the supplied fields … }}` (each of `body`/`label`/`status`/`meeting_type` rides only when supplied and non-empty; all use `omitempty`), and sends **one** `Execute` to `PATCH /tensions/{id}` with `Content-Type: application/json` (plan ADR-1; the `ContentType` seam 042 added is reused). There is no walk and no pagination — a single request. **No `If-Match` header is sent** (optimistic concurrency is Clobbered Changes — deferred; last-write-wins, exactly as the API behaves when `If-Match` is omitted). The `ten_` id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (plan ADR-2). The `200` body `{data: Tension}` decodes into `glassfrog.Document[Tension]`; the structured path serializes the raw payload verbatim.

**Non-idempotent retry**: `update` is built on the same `RetryExecutor` as the reads, but 017's `isSafeMethod` restricts 429 auto-retry to `GET`/`HEAD`, so a `PATCH` surfaces the `429` on first occurrence and is **never silently re-sent** (silent conformance to §133, no command-side special-casing).

**Piping / scripting**: on success, stdout carries only the rendered updated tension. On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 044 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Tension updated | — | `Success` | 0 | — (updated tension on stdout) |
| No editable field supplied (send-set empty: no flags, or only empty-valued flags) | — (local) | `UsageError` | 2 | "at least one of --body, --label, --status, --meeting-type is required"; **no request sent** |
| `--body` supplied but empty / whitespace-only | — (local) | `UsageError` | 2 | "`--body` must not be empty" (a body cannot be blanked); **no request sent** |
| Unsupported `--status` value | — (`validateTensionStatus`) | `UsageError` | 2 | "unsupported --status value \"…\" — supported: archived, processed, unprocessed"; **no request sent** |
| Unsupported `--meeting-type` value | — (`validateMeetingType`) | `UsageError` | 2 | "unsupported --meeting-type value \"…\" — supported: governance, tactical"; **no request sent** |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown tension id (`404`), rejected field/value (`422`), or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted RFC 9457 detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector / unparseable user template | `*output.FormatError` / template error (020/035) | `UsageError` | 2 | names the bad format value + the valid names (or the template parse failure) |
| Unknown flag, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. A `422` validation rejection classifies as `APIError`(3) with the server's detail surfaced — the command adds no interpretation. The token value never appears in any message.

## Consistency Notes

- **Edit verb on the reserved family** (plan ADR-2): `update` is the third addition under 042's `tension` group, after 043's `list`/`get`. The editable flags live only on `update`, so passing `--body`/`--status` to `get` remains a cobra unknown-flag usage error for free (the structural guard 034/038/042/043 rely on). No `tension`-group redefinition; no `role` group.
- **Reuses the landed enum validators** (plan ADR-2): `--status` is validated by 043's `validateTensionStatus`/`supportedTensionStatuses` (the tension status set, distinct from the action/project `validateStatus` — 043's recorded decision already names 044 as the next consumer) and `--meeting-type` by 042's `validateMeetingType`/`supportedMeetingTypes`. Both are reused as new *consumers*, not copied — each set stays single-sourced from the vendored `spec.yaml` enum (no drift across `list`/`create`/`update`).
- **Status is editable here, withheld at capture** (spec System Overview): capture refused `--status` because the server auto-computes it at creation; update forwards a validated `--status` because the API allows an explicit transition on `PATCH` (notably `archived`), then recomputes on save. The command renders whatever the server returns — it performs no client-side recompute and claims no authority over the final state (spec non-behavior).
- **Partial update, last-write-wins** (plan ADR-1/ADR-3): only the supplied fields are sent (every field `omitempty`), unsupplied fields are left untouched server-side, and no `If-Match` is sent — optimistic concurrency is Clobbered Changes, deferred. The at-least-one-field precondition keys on the resolved send-set (presence + non-empty), so a no-op edit (no flags, or `--label ""` alone) costs no request; a supplied blank `--body` is rejected specifically and early.
- **No new machinery** (plan System Architecture): unlike 042 (which added the `ContentType` seam) and 043 (which added the plural `tensions` render), update adds **no** transport, model-response, or render surface — it reuses `apiclient.Request`/`Execute` incl. `ContentType` (042), `Document[Tension]` (034), the singular `tension` render key (042), the `tensionSeam`, `resolveRenderTarget`/`writeHuman`/`output.RenderSuccess` (020/035), and `reportFailure`/`classifyClientError`/the frozen `Outcome`/`ExitCode` registry (011/015/032). The only net-new surface is the `update` leaf and the `TensionUpdateInput` request shape (`044/interface-spec.md`).
- **Flag/command spellings** (`tension`, `update`, `--body`, `--label`, `--status`, `--meeting-type`) resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. No `accords/` directory exists, so there are no cross-spec accord patterns to align against — this conforms to the in-repo CLI precedent set by 042/043.
