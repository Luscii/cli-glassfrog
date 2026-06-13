# Interface Accord: Tension Update — Specification

**Feature**: 044-tension-update
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-1 (new `TensionUpdateInput`, all fields `omitempty` incl. `status`; capture's `TensionInputBody` left byte-stable), ADR-2 (`update <ten-id>` leaf on 042's `tension` group; reuse landed validators; id pass-through), ADR-3 (at-least-one-field precondition; blank-`--body`-when-supplied check). Cross-cutting: single-resource `Document[Tension]`, no new `Outcome`/`ExitCode`, `PATCH` never auto-retried (§133), no `If-Match` (Clobbered Changes deferred).

---

This accord pins the **Go API surface** Tension Update introduces: the `internal/glassfrog` request-input shape it sends (a new partial-update type), the `internal/cli` command symbols, and the two new pure pre-assembly checks. The CLI-facing surface — command, flags, output, exit codes — is in `044/interface-cli.md`. Field names and concrete Go types are a build detail; the **shapes, signatures, and wire contract** are the contract. Unlike 042, this feature adds **no transport growth** (it reuses the `apiclient.Request.ContentType` field 042 added) and **no response-model or render growth** (it reuses `glassfrog.Tension`, `Document[Tension]`, and the singular `tension` render key). Everything in `internal/apiclient`, `internal/output`, `internal/render`, and `internal/paging` is consumed **unchanged**.

---

## Surface

### `internal/glassfrog` — partial-update request input (plan ADR-1; conforms to 011 ADR-1)

Plain JSON-tagged structs, explicit snake_case tags (encoding/json does not bridge underscores). Leaf package — no transport, no cobra. The token is never a field.

| Type | Shape | Notes |
|---|---|---|
| `TensionUpdateInput` (new) | nested envelope `{ "tension": { "body"?: string, "label"?: string, "status"?: string, "meeting_type"?: string } }` | The `updateTension` request body. **All four fields use `omitempty`** (including `status`) — a partial update sends only the supplied fields. The constructor receives the already-resolved (presence-filtered) values; because every supplied value is guaranteed non-empty by the command (blank `--body` rejected; `--status`/`--meeting-type` closed-enum; `--label` only-when-non-empty), `omitempty` + plain strings faithfully expresses "send only what was supplied" with no pointer fields. **A new type, NOT a reuse of capture's `TensionInputBody`** — that type is deliberately `{Body (no omitempty), Label, MeetingType}` with no `status` field (creation cannot claim a server-owned status); update's contract diverges (status editable, body optional, all-partial). Exact Go type name and whether it is a struct + constructor or a small builder is a build detail; the **wire shape** is the contract. |
| `NewTensionUpdateInput` (new) | `(body, label, status, meetingType string) TensionUpdateInput` (shape) | Builds the partial body from the validated, presence-filtered field values. Marshals to `{"tension":{ … only the non-empty supplied fields … }}`. Mirrors `NewTensionInput` (042) but over the four-field, all-`omitempty` body. |

The `200` update response decodes into the existing generic `glassfrog.Document[Tension]` (034) — **no new envelope or response type**; `glassfrog.Tension` is reused unchanged. Capture's landed `TensionInput`/`TensionInputBody`/`NewTensionInput` (042) are **untouched** — 042's create wire output stays byte-stable.

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newTensionUpdateCommand` | `(seam tensionSeam) *cobra.Command` | Guard-registered `update` leaf (`Use:"update <ten-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares `--body`, `--label`, `--status`, `--meeting-type`; reads inherited `--base-url`/`--output`; delegates to `runTensionUpdate`. Attached to the existing group in `newTensionCommand` beside `create`/`list`/`get`. |
| `newTensionCommand` | `(seam tensionSeam) *cobra.Command` (existing) | **Edited additively**: attaches the new `update` leaf; the group `Short` widens to name the edit alongside capture/reads. The `create`/`list`/`get` leaves and the seam are unchanged. |
| `runTensionUpdate` | `(cfg tensionUpdateConfig) (Outcome, error)` | Pure over injected values. Resolves `--output` **first**; then, in order: if `--body` was supplied, rejects a `strings.TrimSpace`-empty value (`UsageError(2)`, no request); validates `--status` (`validateTensionStatus`, 043) and `--meeting-type` (`validateMeetingType`, 042); requires the resolved send-set to be non-empty (`UsageError(2)`, no request). All checks pure + pre-assembly (transport tripwire). Then assembles the `ConnectionContext`, builds the `RetryExecutor`, marshals the partial `{tension:{…}}` body, sends one `Execute` (`PATCH /tensions/{id}`, `ContentType: "application/json"`, **no `If-Match`**) into a `Document[Tension]`. Structured `--output` emits the raw `{data}` verbatim (018); the human path renders the `tension` template (or a selected user template, 035) over a `render.TensionView`. Writes the result to `cfg.stdout`, diagnostics to `cfg.stderr`. Never reads `ctx.Cred.Token`. |
| `tensionUpdateConfig` | struct: `seam`, `baseURL`, `baseURLPresent`, `outputFlag`, `outputPresent`, `id`, `body`, `bodySet`, `label`, `labelSet`, `status`, `statusSet`, `meetingType`, `meetingTypeSet`, `reqCtx`, `stdout`, `stderr` | The config the leaf's `RunE` gathers (the `tensionCreateConfig` shape, plus `--status`/`--body` presence). Each `*Set` is `cmd.Flags().Changed(...)` — a field is sent only when set AND non-empty; the four `*Set` flags also feed the at-least-one-field precondition (presence + non-empty resolves the send-set). |

**No `Outcome`/`ExitCode` edit** — reuses the categories 011/015 landed. `resolveRenderTarget`/`writeHuman`/`output.RenderSuccess` (020/035/018) are reused for output dispatch; `reportFailure`/`refineClientError`/`classifyClientError` (032/015/011) for failures. **No new validator set** — `validateTensionStatus`/`supportedTensionStatuses` (043) and `validateMeetingType`/`supportedMeetingTypes` (042) are reused.

### Consumed unchanged (not defined here)

- `apiclient.Request` (incl. the `ContentType` field added by 042), `(*Client).Execute`, the typed errors (`*AuthError`/`*TransportError`/`*ResponseError`/`*DecodeError`), and `NewRetryExecutor`/`RetryExecutor` incl. its `isSafeMethod` 429 gate (which makes `PATCH` non-retryable — `retry.go:65`). No transport growth.
- `glassfrog.Tension` + `glassfrog.Document[T]` (042/034); `glassfrog.Page[T]`/`Pagination` are **not** used (no pagination on update). Capture's `TensionInput`/`TensionInputBody`/`NewTensionInput` (042) are untouched.
- `output.ResolveFormat`/`OutputFormat`/`RenderSuccess`/`ErrorEnvelope` (018/020), the user-template resolver (035); `render.Render` + the `tension` key + `TensionView` (019/042); the `cli` `resolveRenderTarget`/`writeHuman` dispatch (020/035).
- `classifyClientError`, `refineClientError`, `reportFailure`, `Outcome`, `ExitCode` (011/015/032); `validateTensionStatus` (043) + `validateMeetingType` (042); the `tensionSeam` (042).

**Example (shapes, not literal values)**:
```
// after --output resolved; if --body supplied, blank rejected; --status/--meeting-type validated;
// send-set non-empty required — else UsageError, no request:
in := NewTensionUpdateInput(suppliedBody, suppliedLabel, suppliedStatus, suppliedMeetingType) // {"tension":{…only supplied…}}
body := mustMarshal(in)
ctx := seam.assemble(cfg.baseURL, cfg.baseURLPresent)
client, err := seam.newClient(ctx)
ex := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, seam.sleep(), cfg.stderr)
req := apiclient.Request{
    Method:      "PATCH",
    Path:        "/tensions/" + url.PathEscape(cfg.id),
    Body:        bytes.NewReader(body),
    ContentType: "application/json",                                 // 042's seam, reused; NO If-Match
}
if machineFmt, ok := rt.format.MachineFormat(); ok {                 // structured
    var raw json.RawMessage
    if _, err := ex.Execute(cfg.reqCtx, req, &raw); err != nil { return reportFailure(...) }
    doc, _ := output.RenderSuccess(machineFmt, raw); stdout.Write(doc)   // raw {data: Tension} verbatim
} else {                                                             // human / user template
    var doc glassfrog.Document[glassfrog.Tension]
    if _, err := ex.Execute(cfg.reqCtx, req, &doc); err != nil { return reportFailure(...) }
    return writeHuman(stdout, stderr, rt.tmpl, render.ResourceTension, rt.format, render.TensionView{Tension: doc.Data})
}
// 200 → doc.Data (server's recomputed status); 404/422/401/403/429/5xx → *ResponseError; never auto-retried (PATCH, §133)
```

---

## Interactions

- **Resolve-then-validate-before-call**: `resolveRenderTarget` (020/035) runs first, then the pure input checks — (if supplied) blank `--body`, `--status` in set, `--meeting-type` in set, and the at-least-one-field send-set — all **before** `seam.assemble`, so a bad `--output`, a blank supplied `--body`, an unknown `--status`/`--meeting-type`, a no-op edit (no fields), or (at cobra parse) an unknown flag / wrong arg count costs no network call (a tripwire fake asserts the executor is never invoked on rejection).
- **One request, no walk**: update is a single `Execute`; no pagination, no completeness signalling, no `--first-page`/`--per-page`. Resolution happened at assembly (009); the write re-resolves nothing and never reads `ctx.Cred.Token`.
- **Partial body, presence-faithful**: the send-set is computed from `Changed()` + non-empty value (the capture `labelSet`/`meetingTypeSet` rule extended to `body` and `status`); `NewTensionUpdateInput` + `omitempty` then drop unsupplied fields. `--label ""`/`--status ""`/`--meeting-type ""` resolve to "no field sent" and do not satisfy the precondition on their own. No field is ever sent as JSON `null` (no clear-to-null affordance — spec non-behavior).
- **Decode target**: the `200` `{data: Tension}` → `Document[Tension]` for the human path; raw `json.RawMessage` verbatim for the structured path (018 ADR-2). The server's recomputed `status` is rendered as returned.
- **Content type rides the descriptor**: the body's `Content-Type: application/json` is carried on `apiclient.Request.ContentType` (042's seam, reused) and set by `Execute` — not assembled in the command or the auth layer (007 owns only `X-Auth-Token`). **No `If-Match`** is attached (Clobbered Changes deferred; last-write-wins).

## Error Communication

`runTensionUpdate` returns exactly one code-free `Outcome`; `classifyClientError` maps any typed client error to it. The full mapping (incl. local/cobra rows) is the table in `044/interface-cli.md`. Salient points:

- **Discrimination order**: `*AuthError` matched before `*TransportError` (007's fail-safe must not be mislabelled transport — 010's discipline).
- **No new codes**: the command adds no `Outcome`/`ExitCode` case; 015's landed split supplies `permission`(4)/`rate-limit`(5) for 401/403/429, and a `422` validation rejection classifies as `APIError`(3) with the server's RFC 9457 detail surfaced. `404` (unknown tension id) classifies as `APIError`(3).
- **Two new local usage errors** (no API call): the at-least-one-field precondition and the blank-`--body`-when-supplied check, both `UsageError(2)` with a transport tripwire. The unsupported-`--status`/`--meeting-type` rejections reuse the landed validators' messages.
- **Exhaustiveness guard**: `classifyClientError`'s table test keeps its `len`+comma-ok completeness check.
- **No secret anywhere**: no message or projection renders the token; the write never reads `ctx.Cred.Token`, and the request body carries no credential.

## Consistency Notes

- **Request-input fork, not a growth** (plan ADR-1): a **new** `TensionUpdateInput` is added rather than growing capture's `TensionInputBody`, because the two operations impose divergent contracts on the shared `TensionInput` wire schema (status editable vs. server-owned; body optional vs. required; all-partial vs. body-always). The 011 ADR-1 grow-not-duplicate precedent governs *response* models (Policy/Domain grown additively); request inputs are constructed per-operation, and growing the shared type would erase capture's type-level "cannot send server-owned status" guarantee. Capture's input stays byte-stable.
- **Schema package** (`internal/glassfrog`, 011 ADR-1): only the new partial-update input is added; `Tension` + `Document[T]` are reused unchanged. The all-`omitempty` plain-string shape is sound here because every supplied value is non-empty by construction (the command's validation), so there is no "empty vs. absent" ambiguity to model with pointers.
- **Command package** (`internal/cli`): the `update` leaf is the fourth member of the `tension` family (after `create`/`list`/`get`); it reuses the `tensionSeam` (042) and the `runTension*`/`*Config` shape; it introduces **no** new validator (it reuses 043's `validateTensionStatus` and 042's `validateMeetingType`) and **no** `Outcome`/`ExitCode` edit. The two new pure checks (at-least-one-field, blank-body-when-supplied) live with the tension command code (next to the landed tension validators), not in the shared `status.go`.
- **No transport / response-model / render growth** (plan System Architecture): update is the lightest member of the family — it reuses 042's `ContentType` seam, `Document[Tension]`, and the singular `tension` render key, and 043's status validator. The only net-new Go surface is `TensionUpdateInput`/`NewTensionUpdateInput` and the `update` command symbols.
- **Retry** (`017`): `update` is a new consumer of `RetryExecutor`; the `isSafeMethod` gate (GET/HEAD only) makes a `PATCH` non-retryable on 429 with no edit — the write benefits from the landed safety, so a rate-limited update cannot be silently re-sent.
- **Specification touchpoint** in a project with no `accords/` directory: no cross-spec accord patterns to align against; conforms to the in-repo precedent set by 010/011/014/025/034/042's `interface-spec.md`.
