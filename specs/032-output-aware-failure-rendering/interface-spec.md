# Interface Accord: Output-Aware Failure Rendering — Specification

**Feature**: 032-output-aware-failure-rendering
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (format-aware chokepoint), ADR-2 (`next_step` field on `output.ErrorDetail`; cli-side mapping), ADR-3 (channel/scope), ADR-4 (`kind` map; body-when-valid).

---

This accord pins two Go contracts: the **additive `next_step` field** 032 adds to 018's `output.ErrorDetail`, and the **format-aware failure-render seam** in `internal/cli` that evolves the single failure chokepoint and owns the Diagnostic→envelope mapping. 032 consumes 031's `Diagnose`, 020's `OutputFormat`, and 018's `RenderError` unchanged. Concrete symbol spellings are a build detail; the **field/tag, signatures, the kind mapping, the channel/scope semantics, and the body-validity gate** are the contract.

---

## Surface

### `internal/output` — additive field on 018's envelope

| Symbol | Signature (shape) | Description |
|---|---|---|
| `ErrorDetail.NextStep` | `string` with tag `json:"next_step,omitempty"` | NEW field on 018's existing `ErrorDetail`. Carries the diagnostic's recovery action as its own parseable key, distinct from `message`. `omitempty` so a failure with no next step (internal-error fallback, bare general-API) renders no key. Field **declaration order**: `Message`, `NextStep`, `Kind`, `Status`, `Body` — so the JSON/YAML document reads message → next_step → kind → status → body. |

Everything else in `internal/output` is unchanged: `ErrorEnvelope{Error ErrorDetail}`, `ErrorDetail{Message, Kind, Status, Body}`, and `RenderError(f Format, env ErrorEnvelope) ([]byte, error)` keep their shapes. The new field is *declared* here (018's home) but *populated* in `internal/cli` (below), so `internal/output` imports no transport and performs no classification (018 invariant).

### `internal/cli` — format-aware failure seam (NEW / evolved)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `reportFailure` | `(stdout, stderr io.Writer, format output.OutputFormat, err error) (Outcome, error)` | The evolved single chokepoint (was `reportClientError(stderr, err)`). Refines `err` once, calls `Diagnose(err)` → `Diagnostic`. If `format.MachineFormat()` is structured: writes `output.RenderError(machineFmt, errorEnvelopeFor(d, err))` to **stdout** — passing the single `Diagnostic` it just computed, so `Diagnose` runs once (buffer-then-write — on a render error, nothing reaches stdout and it returns `RuntimeError`). Otherwise writes `renderDiagnostic(d)` to **stderr** (today's behavior, unchanged). Returns `d.Category` and the (refined) error, exactly as the prior chokepoint did. |
| `errorEnvelopeFor` | `(d Diagnostic, err error) output.ErrorEnvelope` | Pure mapping from a failure to the envelope. The caller passes the SAME `Diagnostic` it derives the exit-code category from (the chokepoint computes `Diagnose` once and hands it here — so the rendered facts and the outcome are structurally one value, and `Diagnose` is never run twice on the failure path): `Message ← d.Cause`; `NextStep ← d.NextStep`; `Kind ← kind(d.Category)`. `err` carries the typed-error chain for `Status`/`Body` ← extract the wrapped `*apiclient.ResponseError` (`var re *apiclient.ResponseError; if errors.As(err, &re) { … }`) — `Status` from the response, `Body` set only when `json.Valid(raw)` (else omitted). Lives in `cli`, the only importer of both `internal/output` and `internal/apiclient`. |
| `kind` | `(Outcome) string` | Total 1:1 map from the diagnostic category to the envelope kind token: `UsageError→"usage"`, `RuntimeError→"runtime"`, `NetworkUnavailable→"network"`, `APIError→"api"`, `PermissionError→"permission"`, `RateLimited→"rate-limit"`; a defensive `default` returns `"runtime"` (the internal-error token). Guarded by an exhaustiveness test (LEARNINGS PR #10 — a new `Outcome` must be added here or the test fails). |

### Consumed unchanged (not defined here)

| Symbol | Owner | Use |
|---|---|---|
| `Diagnose(err) → Diagnostic{Category Outcome, Cause, NextStep string}` | 031 | the one normalized diagnostic 032 renders |
| `renderDiagnostic(Diagnostic) string` | 031 | the human cause+next-step line (full/compact path) |
| `refineClientError(err) error` | 015/031 | one-shot refinement of `*ResponseError`→`*ProblemError` before mapping |
| `(OutputFormat).MachineFormat() (output.Format, bool)` | 020 | structured-vs-human branch + the encoder format |
| `output.RenderError(Format, ErrorEnvelope) ([]byte, error)` | 018 | encodes the complete JSON/YAML envelope or returns an error (no fragment) |
| `apiclient.ResponseError{StatusCode int, Body []byte, …}` | 010/015 | source of `status` + raw `body` (reached via `errors.As`; survives refinement because `*ProblemError` unwraps to it) |

## Interactions

**Call-site threading**: every command-execution failure site that today calls `reportClientError(cfg.stderr, err)` changes to `reportFailure(cfg.stdout, cfg.stderr, format, err)`. `format` is already a parameter of each read's run function and `cfg` already carries `stdout`/`stderr`, so the change is mechanical. The signature change is the completeness backstop — an un-threaded site is a **compile error**, so no site can silently keep the plain-text path.

**Out of scope (unchanged symbols)**: `reportIncompleteWalk(stderr, stop)` / `reportIncompleteSubrolesWalk(stderr, stop)` keep their `stderr`-only signatures and plain-text note in every format (the partial structured document already occupies stdout — ADR-3). The `dispatch.go` usage-error path and 020's invalid-selector path are untouched. The three category-only callers (`Diagnose(err).Category`) are untouched — they read the category, they do not render.

**Buffer-then-write**: the structured path builds the whole document in memory (`RenderError`) before writing; a render error leaves stdout empty and maps to `RuntimeError(1)` — the established 018/019/020 contract, extended to the failure path.

## Error Communication

- `reportFailure` returns the same `Outcome` the prior chokepoint did; `ExitCode` (004) maps it. **No new exit code.**
- A structured render that cannot complete returns `(RuntimeError, renderErr)` with nothing on stdout and a token-free message on stderr — the failure-render path is never itself a silent failure (LEARNINGS: no-silent-failure in helpers).
- `errorEnvelopeFor` and `kind` never read or emit the token; combined with `Diagnose`'s token-free output and `RenderError` adding nothing, the envelope is secret-free by construction.

## Consistency Notes

- **Pairs with `interface-cli.md`**: that file pins the operator/agent-facing channels, envelope JSON example, and the exit-code table; this file pins the Go symbols, the field/tag, and the mapping.
- **Conforms to 018's leaf invariant**: the `next_step` field is declared in `internal/output` but populated in `internal/cli`; `internal/output` still imports no transport and does no classification. This mirrors 018's own `ErrorDetail` doc ("the typed-error→envelope mapping … lives at the cli/020 boundary").
- **Conforms to the single-chokepoint discipline** (031 `Diagnose`, 011 `classifyClientError`, render.go single-importer-of-both): 032 extends the one chokepoint rather than forking a parallel renderer; `reportFailure` lives in `cli`, the only importer of both `internal/output` and `internal/apiclient`.
- **Additive to 018's tests**: `next_step` is `omitempty`, so existing structured-success and structured-error envelope tests are byte-stable unless a next step is present.
- **No `accords/` directory** exists, so there are no cross-spec specification accord patterns to align against.
