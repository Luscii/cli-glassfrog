# Interface Accord: Diagnostic Normalization — Specification

**Feature**: 031-diagnostic-normalization
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3 — the single cli-side `Diagnostic{Category, Cause, NextStep}` value produced by a total `Diagnose(err)` normalizer that consolidates the previously-split `classifyClientError` (category) + `formatClientErrorMessage`/`clientErrorNextStep` (cause + next step) in `internal/cli`, with `reportClientError` delegating to it; plus the decode→APIError(3) divergence and the 401/403 + 429 next-step refinements.

---

This accord pins the Go API surface of Diagnostic Normalization: the new **`Diagnostic`** value type and the total **`Diagnose`** function in `internal/cli`, and the consolidation of the shared read-surface helpers (`classifyClientError`, `formatClientErrorMessage`, `clientErrorNextStep`) into `Diagnose`'s one `errors.As` chain, with `reportClientError` refactored to delegate. There is **no command and no entry point** in this slice — `Diagnose` refines the same error the read commands (011–014) already hand to `reportClientError`, so the *invocation* and *instructional* surfaces are **N/A**. The capability introduces **no configuration of its own** — no new `.glassfrogrc` key, env var, or flag — so the *configuration* surface is **N/A**. This accord is the consumer-side consolidation of the diagnostic surface 015 grew: it produces the structured value that Exit-Code Convention (004) reads (`.Category`) and that Output-Aware Failure Rendering (032) will render per `--output`.

**Naming note**: the value is `Diagnostic` and the function is `Diagnose` (not `Normalize`) — the pairing reads as a unit and ties the symbol to the spec name. `Diagnose` is the single classification+composition chain; `classifyClientError` survives only as a thin category-only delegate (below).

---

## Surface

### Entry point — `internal/cli` (the produced capability)

| Function | Signature (shape) | Description |
|---|---|---|
| `Diagnose` | `(err error) Diagnostic` | The single, **total** normalizer. One `errors.As` chain matches the failure family and sets **all three** `Diagnostic` fields from the same matched value, so category and message are computed together and cannot drift. Never panics; an unrecognized error returns the `RuntimeError` fail-safe `Diagnostic` (matching today's `classifyClientError` default). Token-free in every arm (response/path/status only — never the `X-Auth-Token`). Expects the error already refined by `refineClientError` for the API-detail arms (as `reportClientError` does today); an unrefined `*ResponseError` still classifies correctly via the status arm with the status-fallback cause. |

### Output contract — `Diagnostic` (the normalized value)

The one consistent shape for every failure family (spec: cause + category + next step).

| Field | Type | Description |
|---|---|---|
| `Category` | `Outcome` | The code-free category, drawn from the existing `exitcode.go` taxonomy: `UsageError`, `APIError`, `PermissionError`, `RateLimited`, `NetworkUnavailable`, `RuntimeError`. 004's `ExitCode(Category)` maps it to the process code; `Diagnose` never emits the code. |
| `Cause` | `string` | The human-meaningful, token-free explanation of what went wrong — the API's `detail`/`title` for a refined non-2xx, a status-derived fallback when the body wasn't parseable, the named wire failure for a transport error, the shape-mismatch message for a decode error, or the verbatim (token-free) error string for the fail-safe arm. Always non-empty. |
| `NextStep` | `string` | The recovery the caller can take, **or `""` when no reliable next step exists** (the generic-API fallback keeps its existing generic line; the fail-safe arm is empty). `""` is the single, unambiguous "no next step" signal — there is no separate presence flag. |

### Consumer contract changes — `internal/cli` (the shared helpers, consolidated)

| Symbol | Change | Description |
|---|---|---|
| `Diagnostic` (new) | **+ type** | `struct{ Category Outcome; Cause string; NextStep string }` in `internal/cli` (e.g. a new `diagnostic.go` or absorbed into `clienterror.go` — file placement is implementation-level). |
| `Diagnose` (new) | **+ function** | The total normalizer above. Absorbs the arm logic of `classifyClientError` + `formatClientErrorMessage` + `clientErrorNextStep`. |
| `classifyClientError` (`clienterror.go`) | **→ thin delegate** | Becomes `func classifyClientError(err error) Outcome { return Diagnose(err).Category }`, so the three category-only callers that render their own message — the `me.go` output-format path (`:175`), and the `roles.go` (`:215`) / `subroles.go` (`:237`) partial-result paths — keep working unchanged. (Alternatively those call sites use `Diagnose(err).Category` directly; the delegate is the zero-diff choice.) |
| `formatClientErrorMessage`, `clientErrorNextStep` (`me.go`) | **removed / absorbed** | Their per-arm cause and next-step text moves into `Diagnose`'s arms. The human-render composition (below) replaces them at the print site. |
| `reportClientError` (`me.go`) | **delegates** | `err = refineClientError(err); d := Diagnose(err); fmt.Fprintln(stderr, renderDiagnostic(d)); return d.Category, err`. Same refine-once flow; the printed string is byte-equivalent to today for every unchanged family. |
| `renderDiagnostic` (new, unexported) | **+ helper** | Composes the human line: `Cause` when `NextStep == ""`, else `Cause + " — " + NextStep`. This reproduces today's `formatClientErrorMessage` output exactly (every current message is `"<cause> — <next step>"`), so the stderr surface does not drift for unchanged arms. (032 will render the structured `Diagnostic` per `--output` instead of calling this; `renderDiagnostic` is the human-format fallback until then.) |

### Status → Category → exit-code mapping (with the decode change)

| Failure | `Category` | Exit code | Note |
|---|---|---|---|
| `*ResponseError`/`*ProblemError` status 401, 403 | `PermissionError` | 4 | unchanged category; next-step now **split** by status (below) |
| …status 429 | `RateLimited` | 5 | unchanged category; next-step **refined** to the reset window |
| …any other non-2xx | `APIError` | 3 | unchanged |
| **`*DecodeError`** | **`APIError`** | **3** | **CHANGED (ADR-2)** — was `RuntimeError`(1). Cause/next-step wording unchanged. |
| `*TransportError` | `NetworkUnavailable` | 6 | unchanged |
| `*AuthError{NoCredentials}` / base-URL / rcfile / `*output.FormatError` / command-originated usage | `UsageError` | 2 | unchanged |
| `*AuthError{CredentialError}` / anything else (fail-safe) | `RuntimeError` | 1 | unchanged — render-template failures (019) also stay here |

### Next-step contract (the refinements)

| Status / family | Next step | Change |
|---|---|---|
| 401 | "verify the configured API token" | **NEW — split from the combined hint** |
| 403 | "check that the configured identity has the required role membership / permission" | **NEW — split from the combined hint** |
| 429 | "wait for the rate-limit window to reset (per the `Retry-After` / `X-RateLimit-Reset` headers) and retry" | **REFINED — was "the API is rate-limiting; retry later"** |
| other non-2xx | (existing generic line) | unchanged |
| `*DecodeError` | "this may be an API change; report it" | unchanged wording |
| transport / auth / base-URL / format | (existing per-arm lines) | unchanged |
| fail-safe (`RuntimeError` catch-all) | `""` (none) | unchanged |

Representative copy — exact wording is implementation-level, but the 401≠403 distinction, the 429 reset-window reference, and the decode→3 category are contractual.

---

## Interactions

**Single normalize flow**: the read command calls `client.Execute` once; on a failure it hands the error to the shared `reportClientError`, which refines a `*ResponseError` to a `*ProblemError` once, then calls `Diagnose` **once** to get `{Category, Cause, NextStep}`, prints `renderDiagnostic(d)` to stderr, and returns `d.Category`. Category and message come from the same value — they cannot disagree. No per-command (011–014) edit is needed; the chokepoint already exists.

**Totality / fail-safe**: `Diagnose` always returns a `Diagnostic`; an unrecognized error maps to `RuntimeError` with the verbatim (token-free) cause and no next step — never `Success`, so a failure can never exit 0 (CONSTITUTION III). Unrecognized/internal *crashes* that never reach `Diagnose` remain `main`'s panic-recover + `ExitCode`'s default-1 safety net (004); `Diagnose` adds no competing internal-diagnostic path.

**Decode reclassification**: a `*DecodeError` (a 2xx body that would not decode) now yields `Category: APIError` → exit 3 — the API answered but in an unreadable shape (the API's fault), distinct from a CLI-internal fault (exit 1). The cause ("the API response did not match the expected shape") and next step ("…report it") are unchanged.

**Structured value for 032**: `Diagnose` returns the structured `Diagnostic`; Output-Aware Failure Rendering (032) consumes it to render a failure in the selected `--output` format, sourcing `status`/`body` for the JSON/YAML envelope from the wrapped `*ResponseError` via `errors.As` (per 018's `ErrorEnvelope` mapping at the cli/020 boundary). The cobra-native unknown-command/flag usage path is untouched by this slice — 032 reconciles it into the unified envelope when it owns rendering.

---

## Error Communication

`Diagnose` itself communicates no failure — it is total and always returns a usable `Diagnostic`. The operator-facing communication happens at `reportClientError` via `renderDiagnostic`:

| Condition | Message (stderr) | Category → exit code |
|---|---|---|
| Refined non-2xx, API detail parsed | the API's `detail` (and `title` where useful) — **+ the per-status next step** | 401/403 → `PermissionError`(4); 429 → `RateLimited`(5); else → `APIError`(3) |
| Refined non-2xx, body unparseable | `"the API returned a non-2xx response: status N"` — **+ the per-status next step** | same as above |
| `*DecodeError` | `"the API response did not match the expected shape — this may be an API change; report it (…)"` | **`APIError`(3)** (changed) |
| `*TransportError` | `"<wire error> — check connectivity; the API may be unreachable"` | `NetworkUnavailable`(6) |
| `*AuthError`, base-URL, rcfile, `*output.FormatError` | existing per-arm cause + correction step | `UsageError`(2) / `RuntimeError`(1) for `CredentialError` |
| fail-safe | verbatim (token-free) error string, no next step | `RuntimeError`(1) |

**Token-free invariant (load-bearing)**: every `Cause` and `NextStep` is response/path/status only — never the `X-Auth-Token` (a request header). The consolidation must preserve this arm-for-arm; pinned by a token-never-in-output test over `Diagnose(...).Cause`, `.NextStep`, and the rendered line (mirrors 010/015's tests).

**Byte-equivalence for unchanged families**: `renderDiagnostic(Diagnose(err))` reproduces today's `formatClientErrorMessage(err)` output exactly for every family except the three refined arms (401, 403, 429) and the decode category — pinned by golden capture before/after.

---

## Consistency Notes

- **Consolidates 015's read-surface helpers, doesn't fork them**: `classifyClientError`, `formatClientErrorMessage`, and `clientErrorNextStep` were the single chain 011 introduced and 012–015 grew; 031 folds them into one `Diagnose` returning a `Diagnostic`, with `classifyClientError` retained as a thin `.Category` delegate. This is the value-level realization of 015's "message+category computed from the same typed value" — now literally one value.
- **DIVERGENCE — decode → APIError(3)** (supersedes 015's `clienterror.go` decode arm and the 011/019 "undecodable-body → RuntimeError(1)" precedent): a decode failure is an API-exchange problem (exit 3), not a CLI-internal fault (exit 1). Render-template failures (019) stay `RuntimeError`(1). Candidate for `/score:deprecate`.
- **Refines 015's permission + rate-limit next steps**: 015 gave one combined hint for 401/403 ("check the token's access / membership") and "retry later" for 429; 031 splits 401 (verify token) from 403 (membership/permission) and points 429 at the reset window via the `Retry-After`/`X-RateLimit-Reset` headers already on `*ResponseError`. Category mapping (4/4/5) is unchanged.
- **Produces the value 004 and 032 consume**: `ExitCode(Diagnostic.Category)` is unchanged (004); `Diagnostic` is the structured input 032 renders. `status`/`body` for the JSON/YAML envelope come from the wrapped `*ResponseError` (018), so `Diagnostic` need only carry cause/category/next-step.
- **`NextStep string` with `""` = none** (Crafter assumption, plan-deferred): chosen over `*string` to match the codebase's string-message idiom (`formatClientErrorMessage`/`clientErrorNextStep` already return `string`) — the renderer checks emptiness. `Diagnose`/`renderDiagnostic` naming likewise a Crafter choice.
- **No command, no new configuration**: like 005–010/015, this slice registers no cobra command and prints nothing of its own; invocation and instructional surfaces are N/A, and there is no new `.glassfrogrc` key, env var, or flag. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
