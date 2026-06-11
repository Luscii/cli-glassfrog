# Interface Accord: Tension Capture — Specification

**Feature**: 042-tension-capture
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-1 (narrow `ContentType` field on `apiclient.Request`), ADR-2 (`tension` group + `create` leaf), ADR-3 (`validateMeetingType`, required-non-empty `--body`, id pass-through). Cross-cutting: single-resource `Document[Tension]`, no new `Outcome`/`ExitCode`, POST never auto-retried (§133).

---

This accord pins the **Go API surface** Tension Capture introduces: the `internal/apiclient` request-descriptor growth that lets a request carry a body's content type (the first write needs it), the `internal/glassfrog` schema it sends/decodes, the `internal/cli` command symbols, and the one new `internal/render` template key. The CLI-facing surface — command, flags, output, exit codes — is in `042/interface-cli.md`. Field names and concrete Go types are a build detail; the **shapes, signatures, and projected fields** are the contract. Everything in `internal/paging`, `internal/output`, and `internal/render`'s engine is consumed **unchanged**; `internal/apiclient` is consumed unchanged except for the additive `ContentType` field below.

---

## Surface

### `internal/apiclient` — write-body content type (ADR-1)

| Symbol | Change | Notes |
|---|---|---|
| `apiclient.Request` | **additive** field `ContentType string` | The per-call content type for a request body. Empty for the bodyless reads (every landed GET) — byte-identical behavior. Set to `application/json` by the write. Sits beside the existing `Method`/`Path`/`Query`/`Body io.Reader`. |
| `(*Client).Execute` | sets the header when `req.ContentType != ""` | After `http.NewRequestWithContext`, before `Do`: `if req.ContentType != "" { httpReq.Header.Set("Content-Type", req.ContentType) }`. No other behavior changes; the response path, error taxonomy, and body-close discipline are unchanged. |

Chosen over a general `Header http.Header` bag (plan ADR-1): the only second consumer in sight is `If-Match` (Clobbered Changes), which is deferred and forbidden by 042's non-behaviors; generalize when a real second header lands. `RetryExecutor` (017) forwards the `Request` unchanged, so the `ContentType` rides through with no edit there; the `isSafeMethod` gate already prevents a `POST` from being auto-retried on a 429.

### `internal/glassfrog` — tension schema + request input (conforms to 011 ADR-1)

Plain JSON-tagged structs, tolerant of unknown/extra fields, explicit snake_case tags (encoding/json does not bridge underscores). Leaf package — no transport, no cobra. The token is never a field (it is an `X-Auth-Token` request header, not a response field — CONSTITUTION II).

| Type | Shape | Notes |
|---|---|---|
| `Tension` (new) | `ID string` (`ten_…`), `Type string` (enum `tension`), `Body string` (nullable→empty), `Status string` (enum `unprocessed`/`processed`/`archived`), `RoleID string` (`role_…`, nullable), `SensedByID string` (`per_…`, nullable), `CreatedAt string`, `UpdatedAt string`, `Label string` (nullable), `MeetingType string` (enum `tactical`/`governance`/null), `ParentRoleID string` (`role_…`, nullable) | The v5 `Tension` response schema. Nullable fields modeled as plain strings (empty = null), mirroring the existing nullable-string convention (`Policy.Body`, `Project.RoleID`). The `201` create response decodes into `Document[Tension]` (`{data: Tension}`). |
| tension request input (new) | nested envelope `{ "tension": { "body": string, "label"?: string, "meeting_type"?: string } }` | The `createTension` request body (`TensionInput`). `body` always present; `label`/`meeting_type` use `omitempty` so an absent flag sends no field. **`status` is not sent** (server auto-computes) and **no `sensed_by`** field exists (server derives it from the token). Exact Go type name/exported-vs-private and whether it is a struct or a small builder is a build detail; the **wire shape** is the contract. |

The create read decodes the generic `glassfrog.Document[Tension]` (034) — no new envelope type.

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newTensionCommand` | `(seam tensionSeam) *cobra.Command` | Guard-registered **non-runnable group** (`Use:"tension"`, non-empty `Short`, **no** `RunE`); built with its `create` child attached **before** it is registered under root (so the guard's ">=1 child" rule holds). Explicitly wired in `Assemble()`. |
| `newTensionCreateCommand` | `(seam tensionSeam) *cobra.Command` | Guard-registered `create` leaf (`Use:"create <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares `--body`, `--label`, `--meeting-type`; reads inherited `--base-url`/`--output`; delegates to `runTensionCreate`. |
| `runTensionCreate` | `(cfg tensionCreateConfig) (Outcome, error)` | Pure over injected values. Resolves `--output` **before** assembly; validates `--body` non-empty (after `strings.TrimSpace`) and `--meeting-type` against the closed set — both fail-fast `UsageError(2)` with **no request** (transport tripwire). Then assembles the `ConnectionContext`, builds the `RetryExecutor`, marshals the `{tension:{…}}` body, sends one `Execute` (`POST /roles/{id}/tensions`, `ContentType: "application/json"`) into a `Document[Tension]`. Structured `--output` emits the raw `{data}` payload verbatim (018); the human path renders the `tension` template over a `render.TensionView`. Writes the result to `cfg.stdout`, diagnostics to `cfg.stderr`. Never reads `ctx.Cred.Token`. |
| `tensionCreateConfig` | struct: `seam`, `baseURL`, `outputFlag`, `id`, `body`, `label`, `labelSet`, `meetingType`, `meetingTypeSet`, `reqCtx`, `stdout`, `stderr` | The config the leaf's `RunE` gathers (the `projectConfig` shape, plus the write inputs). `labelSet`/`meetingTypeSet` are `cmd.Flags().Changed(...)` — a field is sent only when set AND non-empty. |
| `validateMeetingType` | `(value string) error` | New pure validator mirroring `validateStatus` (`internal/cli/status.go`): empty is the "no constraint" case (caller omits the field); a non-empty value outside `{tactical, governance}` returns a `UsageError`-bound error naming the value and the supported set. The set is sourced from the `spec.yaml` `meeting_type` enum. |
| `tensionSeam` | interface: `assemble(baseURL) ConnectionContext`, `newClient(ctx) (*Client, error)`, `sleep() func(time.Duration)`, `resolveFormat(flag) (OutputFormat, error)` | Same shape as `projectsSeam` minus nothing (the write needs assemble + client + sleep + format; no paging). Production binds `apiclient.AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`; tests bind a fake whose `Execute` returns a canned `201`/non-2xx and a tripwire that asserts no call on a rejected input. |

**No `Outcome`/`ExitCode` edit** — reuses the categories 011/015 landed. `renderResult[T]`/`output.RenderSuccess` (020/018) are reused for output dispatch; `reportFailure`/`refineClientError`/`classifyClientError` (032/015/011) for failures.

### `internal/render` additions

| Key | Formats | Data | Notes |
|---|---|---|---|
| `tension` | `full`, `compact` | `glassfrog.Tension` (a `TensionView`) | Single created-tension projection (042 interface-cli Output). **New** key. Renders the `ten_` id, status badge, and the verbatim free-text `body` as primary content (must not truncate/reflow — the `Policy.Body` precedent), with explicit-absence guards (`{{if}}…{{else}}(none){{end}}`) on the nullable `label`/`role_id`/`sensed_by_id`/`meeting_type`/`parent_role_id`. |

The registry exhaustiveness guard (PR #10 `len`+comma-ok) asserts the new key carries both formats. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes).

### Consumed unchanged (not defined here)

- `apiclient.AssembleFromOS`, `NewClientFromOS`, `(*Client).Execute`, the typed errors (`*AuthError`/`*TransportError`/`*ResponseError`/`*DecodeError`), and `NewRetryExecutor`/`RetryExecutor` incl. its `isSafeMethod` 429 gate (010/017). `apiclient.Request` is consumed unchanged **except** the additive `ContentType` field above.
- `glassfrog.Document[T]` (034); `glassfrog.Page[T]`/`Pagination` are **not** used (no pagination on create).
- `output.ResolveFormat`/`OutputFormat`/`RenderSuccess`/`ErrorEnvelope` (018/020); `render.Render(key, format, v)` (019); the `cli` `renderResult[T]` dispatch (020).
- `classifyClientError`, `refineClientError`, `reportFailure`, `Outcome`, `ExitCode` (011/015/032); the shared `validateStatus` shape this feature's `validateMeetingType` mirrors (013/014).

**Example (shapes, not literal values)**:
```
// after --output resolved, --body non-empty + --meeting-type validated (else UsageError, no request):
body := mustMarshalTensionInput(cfg.body, cfg.label, cfg.meetingType)            // {"tension":{"body":…,"label"?:…,"meeting_type"?:…}}
ctx := seam.assemble(cfg.baseURL)
client, err := seam.newClient(ctx)
ex := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, seam.sleep(), cfg.stderr)
req := apiclient.Request{
    Method:      "POST",
    Path:        "/roles/" + url.PathEscape(cfg.id) + "/tensions",
    Body:        bytes.NewReader(body),
    ContentType: "application/json",                                             // ADR-1: header set by Execute
}
if machineFmt, ok := format.MachineFormat(); ok {                                // structured
    var raw json.RawMessage
    if _, err := ex.Execute(cfg.reqCtx, req, &raw); err != nil { return reportFailure(...) }
    doc, _ := output.RenderSuccess(machineFmt, raw); stdout.Write(doc)           // raw {data: Tension} verbatim
} else {                                                                         // human
    var doc glassfrog.Document[glassfrog.Tension]
    if _, err := ex.Execute(cfg.reqCtx, req, &doc); err != nil { return reportFailure(...) }
    text, _ := render.Render("tension", humanFmt, render.TensionView{Tension: doc.Data}); stdout.Write(text)
}
// 201 → doc.Data (incl. ten_ id); 404/422/401/403/429/5xx → *ResponseError; never auto-retried (POST, §133)
```

---

## Interactions

- **Resolve-then-validate-before-call**: `output.ResolveFormat` runs first, then the two pure input checks (`--body` non-empty, `--meeting-type` in set), all **before** `seam.assemble` — so a bad `--output`, an empty `--body`, an unknown `--meeting-type`, or (at cobra parse) an unknown flag / wrong arg count costs no network call (a tripwire fake asserts the executor is never invoked on rejection).
- **One request, no walk**: capture is a single `Execute`; there is no pagination, no completeness signalling, no `--first-page`/`--per-page`. Resolution happened at assembly (009); the write re-resolves nothing and never reads `ctx.Cred.Token`.
- **Decode target**: the `201` `{data: Tension}` → `Document[Tension]` for the human path; raw `json.RawMessage` verbatim for the structured path (018 ADR-2). The created tension's `ten_` id is the load-bearing output.
- **Content type rides the descriptor**: the body's `Content-Type: application/json` is carried on `apiclient.Request.ContentType` and set by `Execute` (ADR-1) — not assembled in the command or the transport's auth layer (007 owns only `X-Auth-Token`).

## Error Communication

`runTensionCreate` returns exactly one code-free `Outcome`; `classifyClientError` maps any typed client error to it. The full mapping (incl. local/cobra rows) is the table in `042/interface-cli.md`. Salient points:

- **Discrimination order**: `*AuthError` matched before `*TransportError` (007's fail-safe must not be mislabelled transport — 010's discipline).
- **No new codes**: the command adds no `Outcome`/`ExitCode` case; 015's landed split already supplies `permission`(4)/`rate-limit`(5) for 401/403/429, and a `422` validation rejection classifies as `APIError`(3) with the server's RFC 9457 detail surfaced.
- **Exhaustiveness guard**: `classifyClientError`'s table test keeps its `len`+comma-ok completeness check.
- **No secret anywhere**: no message or projection renders the token; the write never reads `ctx.Cred.Token`, and the request body carries no credential.

## Consistency Notes

- **First write extends the transport descriptor additively** (plan ADR-1): the `ContentType` field is the smallest growth of `apiclient.Request` that lets a body be parsed by the API; the reads pass `""` and are unperturbed (pinned by a header-present/absent test). Generalizing to a `Header http.Header` bag is deferred to its first real consumer (`If-Match`, Clobbered Changes) — consistent with 008/010/017 deferring knobs until grounded.
- **Schema package** (`internal/glassfrog`, 011 ADR-1): a **new** `Tension` model is added (not a growth of an existing type — no read shares the tension shape), alongside the small request-input shape; the create decodes the landed generic `Document[T]` (034). The nullable-string-for-null convention follows `Policy.Body`/`Project.RoleID`.
- **Command package** (`internal/cli`): the `tension` group + `create` leaf are the first non-runnable-group/verb-leaf pair after `auth`/`auth login`; the seam mirrors `projectsSeam`; `validateMeetingType` is a new consumer of the `validateStatus` fail-fast shape (a sibling validator, not a second copy of any set). No `Outcome`/`ExitCode` edit.
- **Render** (`019`/`020`): adds one **new** `tension` key (a `TensionView`); reuses the two-package (`output`/`render`) split and `renderResult[T]` dispatch; imports neither package into the other. Structured json/yaml needs no change (raw-bytes path, 018 ADR-2).
- **Retry** (`017`): `create` is a new consumer of `RetryExecutor`; the `isSafeMethod` gate makes a `POST` non-retryable on 429 with no edit — the write benefits from the landed safety.
- **Specification touchpoint** in a project with no `accords/` directory: no cross-spec accord patterns to align against; conforms to the in-repo precedent set by 010/011/014/025/034's `interface-spec.md`.
