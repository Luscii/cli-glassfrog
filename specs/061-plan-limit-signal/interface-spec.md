# Interface Accord: Plan-Limit Signal — Specification

**Feature**: 061-plan-limit-signal
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (`Method`/`Path` on `ResponseError`, set in `Execute`), ADR-2 (central `Diagnose` refinement), ADR-3 (`Feature` on `Diagnostic`; gate→display-name mapping), ADR-4 (`Feature` on `output.ErrorDetail`).

---

This accord pins the Go contracts 061 adds along the existing failure chain: the **request-identity fields** on 010's `ResponseError`, the **`Feature` field** on 031's `Diagnostic`, the **additive `feature` field** on 018's `output.ErrorDetail`, the **central recognition branch** in `Diagnose`, and the **gate→display-name mapping**. 061 consumes 060's `RecognizeFeatureGate` and the rest of the 015/031/032/018 chain unchanged. Concrete symbol spellings are a build detail; the **fields/tags, the recognition branch's placement and contract, the category-unchanged invariant, and the mapping's totality** are the contract.

---

## Surface

### `internal/apiclient` — request identity on the typed error (NEW fields)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `ResponseError.Method` | `string` | NEW. The failed request's HTTP method, set in `Execute` where `req.Method` is in scope (ADR-1). Zero-values to `""`. |
| `ResponseError.Path` | `string` | NEW. The failed request's path (`req.Path`, as given — the **concrete** path, e.g. `/proposals/prp_0123/propose`, which 060's recognizer matches against its `{…}` path templates; never store a template here). Zero-values to `""`. |

`Execute` sets both when it constructs the non-2xx `*ResponseError` (the one site, `execute.go`). `ResponseError.Error()` is **unchanged** (status only), so every existing message and golden test is byte-stable. The fields ride the error through `refineClientError`'s `*ProblemError` wrap (which `Unwrap`s to the `ResponseError`), so they reach `Diagnose` with no call-site changes.

### `internal/cli` — `Diagnostic` carries the recognized gate (NEW field)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `Diagnostic.Feature` | `string` | NEW field on 031's `Diagnostic`. The recognized gating feature's display name (e.g. `Premium async proposals`), or `""` when the failure is not a recognized plan-limit `403`. Set once by `Diagnose`; read by both renderers (the human path via the cause prose, the structured path via `errorEnvelopeFor`). |

### `internal/output` — additive field on 018's envelope (NEW field)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `ErrorDetail.Feature` | `string` with tag `json:"feature,omitempty"` | NEW field on 018's `ErrorDetail`. Carries the gating feature's display name as its own parseable key, distinct from `message` (ADR-4). `omitempty` so every non-plan-limit failure renders no key. Field declaration order: `Message`, `NextStep`, `Feature`, `Kind`, `Status`, `Body`. Declared here (018's home) but populated in `internal/cli`, so `internal/output` stays transport-free and classification-free (018 invariant). |

### `internal/cli` — central recognition branch + mapping (NEW / evolved)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `Diagnose` | `(err error) Diagnostic` | EVOLVED. In the existing `*ProblemError` arm, when `problemErr.StatusCode == 403`, reach the wrapped `*apiclient.ResponseError` (`var re *apiclient.ResponseError; errors.As(err, &re)`) and call `apiclient.RecognizeFeatureGate(re.Method, re.Path, problemErr.StatusCode)`. On a non-`GateNone` gate: set `Cause`/`NextStep` to the gate-aware possibility-framed wording and `Feature` to the gate's display name, **keeping `Category = categoryForStatus(403) = PermissionError`** (ADR-2). On `GateNone` (and the `re`-absent fallback): the arm behaves exactly as today. No other arm changes; `categoryForStatus`/`nextStepForStatus` are untouched. |
| `featureGateDisplayName` (working name) | `(apiclient.Gate) string` | NEW. Total map from a recognized `Gate` kind to its **human-prose** display name: `GatePremiumAsyncProposals → "Premium async proposals"`, `GateAIIntegration → "AI Integration"` (modeled for readiness — no command reaches it today, 060 ADR-3); `GateNone → ""`. **Distinct from 060's landed `Gate.String()`**, which returns kebab-case (`"premium-async-proposals"`) for logs/`%v` — 061's operator-facing diagnostic uses the human-prose form, not `String()`. Guarded by an exhaustiveness test (LEARNINGS PR #10 `len`/comma-ok shape) so a new `Gate` kind without a display name fails loud. The plan-limit cause/next-step wording is composed from this name (061 owns the wording; 060 stays code-free, ADR-3). |
| `errorEnvelopeFor` | `(d Diagnostic, err error) output.ErrorEnvelope` | EVOLVED. Adds `detail.Feature = d.Feature` to the existing mapping (`Message ← d.Cause`, `NextStep ← d.NextStep`, `Kind ← kind(d.Category)`, `Status`/`Body` from the wrapped `*ResponseError`). With `omitempty`, an empty `d.Feature` renders no key. No re-recognition here — it reads the single `Diagnostic` (ADR-3). |
| `renderDiagnostic` | `(Diagnostic) string` | UNCHANGED. Still `"<cause> — <next step>"`; the gate name reaches the human line through `d.Cause`, so no signature or body change is required. |

### Consumed unchanged (not defined here)

| Symbol | Owner | Use |
|---|---|---|
| `RecognizeFeatureGate(method, path string, status int) apiclient.Gate` | 060 | the recognizer — supplies the gate (or `GateNone`) for the failed operation |
| `apiclient.Gate` (`GateNone`, `GatePremiumAsyncProposals`, `GateAIIntegration`) + `Gate.String()` | 060 | the gate-kind enum the display-name map keys on; `String()` gives kebab-case for logs (not used in the operator diagnostic) |
| `refineClientError(err) error` | 015/031 | one-shot `*ResponseError`→`*ProblemError` refinement before `Diagnose` |
| `reportFailure(stdout, stderr, format, err)` | 032 | the single failure chokepoint — **unchanged signature**; it already calls `Diagnose` and `errorEnvelopeFor` |
| `categoryForStatus(403) → PermissionError` | 031 | the category mapping 061 does **not** touch |
| `output.RenderError(Format, ErrorEnvelope)` | 018 | encodes the complete envelope (now possibly carrying `feature`) |

## Interactions

**No call-site threading**: because `Method`/`Path` are set at the source (`Execute`) and flow through the error chain, **none of the ~100 `reportFailure` call sites change**, and `reportFailure`/`refineClientError` keep their signatures. The recognition branch lives entirely inside `Diagnose` — the one place the refined error and its wrapped `ResponseError` are both in hand. This is the completeness property the source-enrichment choice buys (ADR-1): every gated operation's `403`, present and future, routes through the one branch with no per-command work.

**Single classification site**: `Diagnose` sets `Cause`, `NextStep`, `Category`, and `Feature` from the one matched value, so the rendered facts, the exit-code category, and the gate name cannot drift — the invariant `Diagnose`'s doc-comment already protects, extended to the gate.

**Additive, byte-stable**: all four new struct members (`ResponseError.Method`, `ResponseError.Path`, `Diagnostic.Feature`, `ErrorDetail.Feature`) zero-value to `""` / omit. Every non-plan-limit failure — and every existing test, golden projection, and envelope snapshot — is unchanged.

## Error Communication

- `Diagnose` returns the same `Category` (`PermissionError`) for a recognized plan-limit `403` as for any `403`; `ExitCode` (004) maps it to `4`. **No new `Outcome`, no new exit code.**
- `RecognizeFeatureGate` is total and never errors (060); a `GateNone` result leaves the existing diagnostic untouched, so the change can never turn a non-plan-limit `403` into a plan-limit message.
- `featureGateDisplayName`, the wording, and `errorEnvelopeFor` never read or emit the token; the inputs are the request method/path and response-side status/body, so the envelope stays secret-free by construction.

## Consistency Notes

- **Extends `032/interface-spec.md`**: 061 adds the `feature` field to `ErrorDetail` exactly as 032 added `next_step` (declared in `internal/output`, populated in `internal/cli`'s `errorEnvelopeFor`), and evolves `Diagnose` in place rather than forking a renderer.
- **Pairs with `interface-cli.md`**: that file pins the operator/agent channels, the envelope example, and the exit-code table; this file pins the Go symbols, fields/tags, and the recognition branch.
- **Conforms to 060** (`RecognizeFeatureGate`, `Gate`): consumed unchanged; 060 stays a pure code-free classifier, 061 owns the display-name/wording layer (060 plan ADR-1, this plan ADR-3).
- **Conforms to 015's single-refinement discipline**: 061 does **not** wrap the error at call sites (which `refineClientError` would discard); it enriches the `ResponseError` at its source so the existing one-shot refinement carries the request identity through.
- **Conforms to 018's leaf invariant**: `feature` is declared in `internal/output` but populated in `internal/cli`; `internal/output` still imports no transport and does no classification.
- **No `accords/` directory** exists, so there are no cross-spec specification accord patterns to align against.
