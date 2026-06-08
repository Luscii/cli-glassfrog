# Interface Accord: Structured Serialization — Specification

**Feature**: 018-structured-serialization
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (new `internal/output` leaf package), ADR-2 (serialize raw bytes via a `json.RawMessage` target), ADR-3 (`sigs.k8s.io/yaml` `JSONToYAML`), ADR-4 (unified error envelope; `kind` from the `classifyClientError` taxonomy).

---

This accord pins two contracts: the **Go API surface** of the new `internal/output` package, and the **external document shapes** a consumer (an AI agent) parses — the verbatim success document and the unified error envelope. 018 owns the encoders and the envelope shape only; the `--output` flag, format selection, decode-target wiring, and the typed-error→envelope mapping are the consuming surface's (Output Format Selection, 020) and are *not* defined here. There is no CLI touchpoint for 018 — it adds no command and no flag. Concrete Go type/field names are a build detail; the **shapes, signatures, format identifiers, and document structure** are the contract.

---

## Surface

### `internal/output` (NEW leaf package — pure; no cobra, no transport, no domain types)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `Format` | enum: `JSON`, `YAML` | The machine formats 018 supports. 020 extends the *selection* vocabulary (`full`/`compact` human formats, 019); these two identifiers are 018's. |
| `RenderSuccess` | `(f Format, payload json.RawMessage) ([]byte, error)` | Renders the raw 2xx body verbatim in `f`. JSON → the bytes normalized (valid, consistently indented). YAML → `yaml.JSONToYAML(payload)`. An empty/whitespace-only `payload` renders as a valid empty document (never an empty channel). Returns a render error if the bytes are not valid JSON (a contract violation), never a partial document. |
| `ErrorEnvelope` | struct wrapping one `ErrorDetail` under an `error` key | The unified failure document 018 defines (ADR-4). One shape for every failure origin. |
| `ErrorDetail` | `Message string` (always); `Kind string` (always); `Status int` (omitempty); `Body json.RawMessage` (omitempty) | `Message`: human-readable, token-free. `Kind`: taxonomy term (see Error Communication). `Status`: the HTTP status, present only for a non-2xx response. `Body`: the raw API error body verbatim, present only when the API returned one. |
| `RenderError` | `(f Format, env ErrorEnvelope) ([]byte, error)` | Renders the envelope in `f` (JSON: marshal; YAML: equivalent). Deterministic field order; complete document or a render error, never a fragment. |

### External document shapes (what a consumer parses)

**Success** — the raw API payload, verbatim, encoded in the active format. All fields the API returned are present (no reshaping, no field loss, no number-precision loss). The JSON form and the YAML form carry identical data.

**Error envelope** — JSON form (YAML is the same data):
```json
{ "error": { "message": "the API returned a non-2xx response: status 404", "kind": "api", "status": 404, "body": { "...": "the raw API error body, verbatim" } } }
```
For a bodiless failure (transport, auth fail-safe, decode), `status` and `body` are absent:
```json
{ "error": { "message": "request failed: <network cause>", "kind": "network" } }
```

### Consumed unchanged (not defined here)

- **From `010/interface-spec.md`**: `(*Client).Execute(reqCtx, req, out any)` accepts a `*json.RawMessage` as `out` (raw 2xx capture — no 010 change); `*ResponseError{StatusCode, Header, Body []byte}` already carries the raw error body; the typed errors `*AuthError` / `*TransportError` / `*ResponseError` / `*DecodeError`.
- **From `011/interface-spec.md`**: `classifyClientError(err) Outcome` and the `Outcome` taxonomy — reused by the (020-owned) typed-error→envelope mapping to populate `Kind`, so the discrimination chain lives in exactly one place.
- **From the Go toolchain**: `encoding/json` (JSON encode/normalize); `sigs.k8s.io/yaml` (`JSONToYAML`).

**Example (shapes, not literal values — the producing side lives in 020)**:
```
// success, structured format active:
var raw json.RawMessage
_, err := client.Execute(reqCtx, req, &raw)          // raw 2xx body captured verbatim (ADR-2)
doc, rerr := output.RenderSuccess(format, raw)       // JSON normalize | YAML JSONToYAML

// failure, structured format active:
env := buildEnvelope(err)                            // 020-owned: kind via classifyClientError; status/body from *ResponseError
doc, rerr := output.RenderError(format, env)
```

---

## Interactions

- **Format is chosen upstream**: the consumer (020) determines the active `Format` before the request and passes it in; 018 never reads a flag.
- **Decode-target selection (ADR-2)**: when a structured format is active, the consumer decodes the 2xx body into `json.RawMessage` (verbatim) rather than the typed `glassfrog` struct; the typed-struct path remains for the human projection (019). The choice is the consumer's; `RenderSuccess` only encodes the bytes it is handed.
- **One document per outcome**: a single `RenderSuccess` *or* `RenderError` call produces the whole document; the consumer writes it to stdout as the sole content of the channel (diagnostics stay on stderr).
- **JSON ≡ YAML**: both forms derive from the same JSON bytes (`JSONToYAML` transforms the JSON), so parsing either yields structurally equivalent data — by construction, not by parallel encoders.
- **Number & field fidelity**: both paths operate on bytes, so no JSON number is coerced to float64 and no field is dropped.
- **Envelope population (consumer-side, 020)**: `Kind` is set from the `classifyClientError` taxonomy; `Status`/`Body` from `*ResponseError`; `Message` from the typed error's token-free string. 018 supplies the shape and the encoder; it performs no classification.

---

## Error Communication

Two distinct failure surfaces:

**1. The serialized error envelope** (the external contract for a *command's* failure). `Kind` is the lowercased taxonomy term, derived from `classifyClientError`'s `Outcome` — which **API Error Extraction (015, landed in #44)** widened from the original five categories by splitting non-2xx on status:

| Originating typed error | `Outcome` | Envelope `kind` | `status` / `body` |
|---|---|---|---|
| `*ResponseError`/`*ProblemError` 401, 403 | `PermissionError` | `permission` | `status` = HTTP status; `body` = raw API error body |
| `*ResponseError`/`*ProblemError` 429 | `RateLimited` | `rate-limit` | `status` = HTTP status; `body` = raw API error body |
| `*ResponseError`/`*ProblemError` other non-2xx | `APIError` | `api` | `status` = HTTP status; `body` = raw API error body |
| `*TransportError` | `NetworkUnavailable` | `network` | absent |
| `*AuthError{NoCredentials}` / base-URL error | `UsageError` | `usage` | absent |
| `*AuthError{CredentialError}` / `*DecodeError` / fail-safe | `RuntimeError` | `runtime` | absent |

- The envelope is emitted in the **active format**, so a consumer that requested JSON always parses JSON on the error path (clarify resolution).
- 015's `*ProblemError` (which wraps `*ResponseError`) carries extracted problem detail; the (020-owned) mapping can carry that detail into `body`/`message` — the envelope *shape* defined here is unchanged and already absorbs the widened `kind` set.
- The token never appears in `message` or anywhere in the envelope (pinned by test, success and every error branch).

**2. A render failure** (018's own failure to encode). If `RenderSuccess` is handed bytes that are not valid JSON (a 2xx contract violation) or encoding otherwise fails, it returns a render `error` and **no document** — never a partial or invalid fragment. The consumer (020) maps that to `RuntimeError` (`ExitCode` 1) via the existing classifier's fail-safe; 018 itself emits no exit code.

---

## Consistency Notes

- **New leaf package, established pattern**: `internal/output` is a pure leaf like `internal/glassfrog` (schema) and `internal/apiclient` (transport), imported one-directionally by `internal/cli` (ADR-1, CONSTITUTION VI). It is the package-API accord style 014/interface-spec.md uses for the read surface, applied to the rendering surface.
- **Reuses, does not redefine**: `Execute`'s `out any` contract and `*ResponseError.Body` (010); `classifyClientError` and the `Outcome` taxonomy (011). 018 adds no `Outcome` case, no `ExitCode` edit, and no classifier branch.
- **Downstream siblings extend this package**: Templated Human Rendering (019) adds template rendering; Output Format Selection (020) adds the `--output` flag, the full format vocabulary (`json|yaml|full|compact`), the success/error routing, the decode-target wiring, and the invalid-selector bootstrap. Their interface files do not exist yet — these boundaries are named for forward reference.
- **Composition with 015**: API Error Extraction (015, landed) supplies the classified `Outcome` (incl. the `permission`/`rate-limit` split) and the `*ProblemError` detail that the (020-owned) mapping carries into the envelope's `body`/`message`; the envelope *shape* defined here is the stable contract. Same handling-vs-classification split as 017↔015.
- **No `accords/` directory** exists, so there are no cross-spec accord patterns to align against beyond the DECISIONS precedent recorded for 018.
