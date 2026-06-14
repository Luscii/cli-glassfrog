# Interface Accord: Proposal Creation — Specification

**Feature**: 055-proposal-creation
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-1 (`proposal` group + `create` leaf), ADR-2 (`resolveChangesSource`: reserved `stdin` / existing file / inline), ADR-3 (`type`-floor fail-fast, verbatim `[]json.RawMessage`), ADR-4 (`glassfrog.Proposal` model + `proposal` render key). Cross-cutting: single-resource `Document[Proposal]`, no new `Outcome`/`ExitCode`, no transport change (reuses 042 `ContentType`, sends no `If-Match`), POST never auto-retried (§133).

---

This accord pins the **Go API surface** Proposal Creation introduces: the `internal/glassfrog` schema it sends/decodes, the `internal/cli` command + change-source-resolution symbols, and the one new `internal/render` template key. The CLI-facing surface — command, flags, output, exit codes — is in `055/interface-cli.md`. Field names and concrete Go types are a build detail; the **shapes, signatures, and projected fields** are the contract. Everything in `internal/apiclient` (including the landed `ContentType` field from 042), `internal/paging`, `internal/output`, and `internal/render`'s engine is consumed **unchanged** — this feature adds no transport mechanism.

---

## Surface

### `internal/glassfrog` — proposal schema + request input (conforms to 011 ADR-1)

Plain JSON-tagged structs, tolerant of unknown/extra fields, explicit snake_case tags. Leaf package — no transport, no cobra. The token is never a field (CONSTITUTION II). The `Proposal`/`ProposalChange`/`ResponseSummary` **response** model is **shared with the concurrent Proposal Reads (056)** — created by whichever lands first (056 ADR-2), reused by the follower. 055's exclusive addition is the **write request-input** type.

| Type | Shape | Notes |
|---|---|---|
| `Proposal` (shared with 056 — created by first-to-land) | `ID string` (`prp_…`), `Type string` (enum `proposal`), `Status string` (enum `draft`/`proposed_outside_meeting`/`escalated`/`accepted`/`draft_with_conflicts`), `TensionID string` (`ten_…`, nullable), `CircleID string` (`role_…`, nullable), `ProposerID string` (`per_`/`agt_…`, nullable), `Changes []ProposalChange`, `ResponseSummary ResponseSummary`, `ExpectedResponseCount int`, `ReceivedResponseCount int`, `AvailableTransitions []string` (enum `propose`/`withdraw`), `ProposedAt/ResponseDeadline/AcceptedAt string` (nullable), `CreatedAt string`, `UpdatedAt string` | The v5 `Proposal` response schema. Nullable fields modeled as plain strings (empty = null), the existing convention. The `201` create response decodes into `Document[Proposal]` (`{data: Proposal}`). 055 consumes this shape unchanged — it adds no field. |
| `ProposalChange` (shared with 056) | `ID string`, `Type string`, plus a free-form `map[string]any` remainder (056 ADR-2, the `actions.go` precedent) | The response decode of each change. Free-form — per-type keys are preserved, never interpreted. **Note**: this is the *response* shape; 055's *request* body carries changes as `[]json.RawMessage` (verbatim send — see the request-input row), a deliberately distinct write-side type. |
| `ResponseSummary` (shared with 056) | `Total int`, `NoObjection int`, `BringToMeeting int` | The aggregate `response_summary` object (no per-person attribution). On a fresh `draft` these are typically `0`. |
| proposal request input (**055-exclusive — new**) | nested envelope `{ "proposal": { "tension_id": string, "changes": [ <raw>, … ] } }` | The `createProposal` request body (`CreateProposalRequest`). `tension_id` always present; `changes` is the validated, non-empty `[]json.RawMessage` carried **byte-for-byte** as supplied above the `type` floor — distinct from the response decode shape, and 056 never needs it (it has no write). **No `proposer`** field (server derives from token) and **no `status`** (server sets `draft`). Exact Go type name/exported-vs-private is a build detail; the **wire shape** is the contract. |

The create decodes the generic `glassfrog.Document[Proposal]` (034) — no new envelope type.

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newProposalCommand` (**shared with 056** — created by first-to-land) | `(seam proposalSeam) *cobra.Command` | Guard-registered **non-runnable group** (`Use:"proposal"`, non-empty `Short`, **no** `RunE`); built with its leaves attached **before** registration under root, explicitly wired in `Assemble()`. If 055 lands first it builds the group with `create`; if 056 lands first, 055 **attaches its `create` leaf to the existing group** rather than redefining the constructor (043's relationship to 042's `newTensionCommand`). |
| `newProposalCreateCommand` | `(seam proposalSeam) *cobra.Command` | Guard-registered `create` leaf (`Use:"create <tension-id>"`, `Args: cobra.MaximumNArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares `--changes`; reads inherited `--base-url`/`--output`; delegates to `runProposalCreate`. |
| `runProposalCreate` | `(cfg proposalCreateConfig) (Outcome, error)` | Pure over injected values. Resolves `--output` **before** assembly; checks the required positional and `--changes` presence; reads the change source via the seam; parses JSON and applies the `type` floor — all fail-fast `UsageError(2)` with **no request** (transport tripwire). Then assembles the `ConnectionContext`, builds the `RetryExecutor`, marshals `{proposal:{tension_id, changes}}`, sends one `Execute` (`POST /proposals`, `ContentType: "application/json"`, no `IfMatch`) into a `Document[Proposal]`. Structured `--output` emits the raw `{data}` verbatim (018); the human path renders the `proposal` template over a `render.ProposalView`. Writes the result to `cfg.stdout`, diagnostics to `cfg.stderr`. Never reads `ctx.Cred.Token`. |
| `proposalCreateConfig` | struct: `seam`, `baseURL`, `baseURLPresent`, `outputFlag`, `outputFlagPresent`, `tensionID`, `changesValue`, `reqCtx`, `stdout`, `stderr` | The config the leaf's `RunE` gathers (the `tensionCreateConfig` shape, with the anchor id + the raw `--changes` flag value). `changesValue` is the unresolved flag string; the seam classifies/reads it. |
| `resolveChangesSource` | `(value string, stat func(string)(os.FileInfo,error), readFile func(string)([]byte,error), isTTY bool, stdin io.Reader) ([]byte, error)` | **New pure** source classifier (plan ADR-2): a trimmed value equal (case-insensitive) to `stdin` → the bounded stdin reader (reusing `readBoundedStdinN` with a changes cap + the TTY/empty fail-fast); else a value whose `stat` reports an **existing regular file** → `readFile`; else the value's own bytes (inline JSON). Returns a `UsageError`-bound error naming the source on a read/empty-pipe/non-regular failure. Pure over injected sources — tested with no real pipe or filesystem. |
| `validateChanges` | `([]byte) ([]json.RawMessage, error)` | **New pure** validator (plan ADR-3): unmarshals the bytes into `[]json.RawMessage`; returns a `UsageError`-bound error if not valid JSON, not an array, or empty; then probes each element by decoding into a minimal `{Type string}` and rejects (`UsageError`) an element that is not an object or has a blank `type`. Returns the verbatim slice on success — keys other than `type` are never read. |
| `proposalSeam` | interface: `assemble(baseURL string, baseURLPresent bool) ConnectionContext`, `newClient(ctx) (*Client, error)`, `sleep() func(time.Duration)`, `resolveSelection(flagValue string, flagPresent bool) (Selection, error)`, `readTemplateSource(ref TemplateRef) (string, error)`, **`readChangesSource(value string) ([]byte, error)`** | The 042 `tensionSeam` shape **plus** `readChangesSource`. Production binds `apiclient.AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, the 035 selection+template binders, and a `readChangesSource` over real `os.Stat`/`os.ReadFile`/`os.Stdin`; tests bind a fake whose `Execute` returns a canned `201`/non-2xx, a `readChangesSource` over injected bytes, and a tripwire asserting no `Execute` call on a rejected input. |

**No `Outcome`/`ExitCode` edit** — reuses the categories 011/015 landed. `renderResult[T]`/`output.RenderSuccess` (020/018) for output dispatch; `reportFailure`/`refineClientError`/`classifyClientError` (032/015/011) for failures.

### `internal/render` — singular `proposal` key (shared with 056)

| Key | Formats | Data | Notes |
|---|---|---|---|
| `proposal` (**shared with 056** — created by first-to-land) | `full`, `compact` | `glassfrog.Proposal` (a `ProposalView`) | Single-proposal projection. If 056 lands first, 055 **reuses its richer view unchanged** (changes by type, response summary, available transitions) — the created proposal renders through it, surfacing the load-bearing `prp_` id and `draft` status. If 055 lands first, it adds a thinner view (`prp_` id, status badge, anchor `tension_id`, change **count** = `len(Changes)`, aggregate response counts, `available_transitions`) with explicit-absence guards on the nullable `tension_id`/`circle_id`/`proposer_id`, which 056 then grows. Registered in `builtinResources` for the exhaustiveness guard. |

The registry exhaustiveness guard (PR #10 `len`+comma-ok) asserts the key carries both formats. Structured `json`/`yaml` output needs no change and is faithful regardless of model field coverage (018 ADR-2 serializes raw bytes).

### Consumed unchanged (not defined here)

- `apiclient.Request` (including the **landed `ContentType` field**, 042), `apiclient.AssembleFromOS`, `NewClientFromOS`, `(*Client).Execute`, the typed errors (`*AuthError`/`*TransportError`/`*ResponseError`/`*DecodeError`), and `NewRetryExecutor`/`RetryExecutor` incl. its `isSafeMethod` 429 gate (010/017). **No `apiclient` change** — the write body's content type rides the existing field; no `If-Match` is sent (the `IfMatch` field, 053, zero-values to `""`).
- `glassfrog.Document[T]` (034); `glassfrog.Page[T]`/`Pagination` are **not** used (no pagination on create).
- `output.ResolveSelectionFromOS`/`Selection`/`OutputFormat`/`RenderSuccess` (018/020/035); `render.Render(key, format, v)` (019); the `cli` `renderResult[T]` dispatch (020); `readBoundedStdinN` (006, reused by the changes-source stdin arm).
- `classifyClientError`, `refineClientError`, `reportFailure`, `Outcome`, `ExitCode` (011/015/032).

**Example (shapes, not literal values)**:
```
// after --output resolved, positional + --changes present, source read, type floor passed (else UsageError, no request):
raw := seam.readChangesSource(cfg.changesValue)          // stdin | existing file | inline → bytes
changes := validateChanges(raw)                          // []json.RawMessage; UsageError on bad JSON / empty / typeless
body := mustMarshalProposalInput(cfg.tensionID, changes) // {"proposal":{"tension_id":…,"changes":[…verbatim…]}}
ctx := seam.assemble(cfg.baseURL, cfg.baseURLPresent)
client, err := seam.newClient(ctx)
ex := apiclient.NewRetryExecutor(client, apiclient.DefaultRetryPolicy, seam.sleep(), cfg.stderr)
req := apiclient.Request{
    Method:      "POST",
    Path:        "/proposals",
    Body:        bytes.NewReader(body),
    ContentType: "application/json",                      // 042: header set by Execute; no IfMatch
}
if machineFmt, ok := selection.MachineFormat(); ok {     // structured
    var rawResp json.RawMessage
    if _, err := ex.Execute(cfg.reqCtx, req, &rawResp); err != nil { return reportFailure(...) }
    doc, _ := output.RenderSuccess(machineFmt, rawResp); stdout.Write(doc)   // raw {data: Proposal} verbatim
} else {                                                 // human / template
    var doc glassfrog.Document[glassfrog.Proposal]
    if _, err := ex.Execute(cfg.reqCtx, req, &doc); err != nil { return reportFailure(...) }
    text, _ := render.Render("proposal", humanFmt, render.ProposalView{Proposal: doc.Data}); stdout.Write(text)
}
// 201 → doc.Data (incl. prp_ id, draft status); 403 → permission(4); 404/422 → APIError(3); never auto-retried (POST, §133)
```

---

## Interactions

- **Resolve-then-validate-before-call**: `output.ResolveSelectionFromOS` runs first, then the positional + `--changes` presence checks, then the source read and the `type` floor — all **before** `seam.assemble`, so a bad `--output`, a missing positional/`--changes`, an unparseable/empty/non-array/typeless change set, a bad stdin/file source, or (at cobra parse) an unknown flag / >1 positional costs no network call (a tripwire fake asserts the executor is never invoked on rejection).
- **One request, no walk**: creation is a single `Execute`; no pagination, no completeness signalling. Resolution happened at assembly (009); the write re-resolves nothing and never reads `ctx.Cred.Token`.
- **Decode target**: the `201` `{data: Proposal}` → `Document[Proposal]` for the human path; raw `json.RawMessage` verbatim for the structured path (018 ADR-2). The created proposal's `prp_` id is the load-bearing output.
- **Verbatim change set**: the request `changes` is the validated `[]json.RawMessage` marshalled byte-for-byte (only the `type` of each element was ever decoded); the CLI neither reshapes nor drops command-specific keys.
- **Content type rides the descriptor; no concurrency header**: the body's `Content-Type: application/json` is carried on the landed `apiclient.Request.ContentType` (042) and set by `Execute`; `IfMatch` is left empty (a create has no prior `ETag`).

## Error Communication

`runProposalCreate` returns exactly one code-free `Outcome`; `classifyClientError` maps any typed client error to it. The full mapping (incl. local/cobra rows) is the table in `055/interface-cli.md`. Salient points:

- **Discrimination order**: `*AuthError` matched before `*TransportError` (007's fail-safe must not be mislabelled transport — 010's discipline).
- **No new codes**: the command adds no `Outcome`/`ExitCode` case; 015's landed split supplies `permission`(4)/`rate-limit`(5) for 401/403/429. A `403` (Premium async-proposals disabled) classifies as `PermissionError`(4) with the server's RFC 9457 detail surfaced; a `404`/`422` classifies as `APIError`(3).
- **Local-floor errors**: missing positional/`--changes`, unparseable/non-array/empty/typeless change set, and bad stdin/file source are all `UsageError(2)`, raised pre-assembly with no request.
- **Exhaustiveness guard**: `classifyClientError`'s table test keeps its `len`+comma-ok completeness check.
- **No secret anywhere**: no message or projection renders the token; the write never reads `ctx.Cred.Token`, and the request body carries no credential.

## Consistency Notes

- **No transport change** (plan Cross-cutting, 042 ADR-1): the body's `Content-Type` rides the landed `apiclient.Request.ContentType`; this is the second consumer of that field and adds nothing to `apiclient`. No `If-Match` is sent (a create has no prior `ETag`; the 053 `IfMatch` field stays `""`), so the 052/053/054 optimistic-concurrency path is untouched.
- **Concurrent sibling 056 — shared group/model/render** (plan Cross-cutting + ADR-1/ADR-4): the `proposal` group (`newProposalCommand`), the `glassfrog.Proposal`/`ProposalChange`/`ResponseSummary` response model, and the singular `proposal` render key are **shared with the concurrently-specified Proposal Reads (056)**. Contract: **first-to-land creates, follower reuses/grows** (043→042). As cut from current `main` (no proposal code present), 055 creates them; if 056 lands first, 055 attaches its `create` leaf to the existing group and reuses 056's model + richer singular render unchanged. 056 additionally owns the plural `proposals` render, `list`/`get`, and `validateProposalStatus` (none of 055's concern). 055's exclusive surface is the **write request-input**, `resolveChangesSource`, and the `type` floor.
- **Schema package** (`internal/glassfrog`, 011 ADR-1): the **shared** `Proposal` model + `ProposalChange` + `ResponseSummary` (response side, `Changes []ProposalChange` per 056 ADR-2) are reused-or-created per the 056 contract; the create decodes the generic `Document[T]` (034). 055 **adds** only the write request-input shape, whose `changes` are `[]json.RawMessage` carried byte-for-byte — deliberately distinct from the response decode shape.
- **Command package** (`internal/cli`): the `proposal` group + `create` leaf are the second non-runnable-group/verb-leaf pair after `tension`/`tension create`; the write seam is the `tensionSeam` shape plus `readChangesSource`. `resolveChangesSource`/`validateChanges` are **new** pure helpers exclusive to 055; `resolveChangesSource` reuses the 006 `readBoundedStdinN` and the 035 reserved-`stdin` idiom (diverging with an existing-file check + inline form). No `Outcome`/`ExitCode` edit.
- **Render** (`019`/`020`): the singular `proposal` key (a `ProposalView`) is shared with 056 (created by first-to-land); reuses the two-package (`output`/`render`) split and `renderResult[T]` dispatch; imports neither package into the other. Structured json/yaml needs no change (raw-bytes path, 018 ADR-2).
- **Retry** (`017`): `create` is a new consumer of `RetryExecutor`; the `isSafeMethod` gate makes a `POST` non-retryable on 429 with no edit.
- **Specification touchpoint** in a project with no `accords/` directory: no cross-spec accord patterns to align against; conforms to the in-repo precedent set by 010/011/014/025/034/042's `interface-spec.md`.
