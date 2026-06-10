# Plan: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/DEPRECATION.md`, and the shipped error surface in `internal/cli` (`clienterror.go`, `me.go`, `exitcode.go`, `dispatch.go`) and `internal/apiclient` / `internal/output`.

---

## System Architecture

Diagnostic Normalization is **not greenfield** — the CLI already classifies failures and composes a cause-plus-next-step message, but across three *separate* functions in `internal/cli`, tied together only at a print site:

- `classifyClientError(err) Outcome` (`clienterror.go`) — the single `errors.As` chain mapping an API-client error to a code-free `Outcome` category (transport→NetworkUnavailable, non-2xx→APIError/PermissionError/RateLimited by status, decode→RuntimeError, base-URL/rcfile/format→UsageError, fallback→RuntimeError).
- `formatClientErrorMessage(err) string` + `clientErrorNextStep(status) string` (`me.go`) — compose the operator-facing **cause** and per-class **next step**.
- `reportClientError(stderr, err)` (`me.go`) — the chokepoint: refines a generic `*ResponseError` into a `*ProblemError` once, prints the message to stderr, and returns the category.

This feature **consolidates those into one value**. It introduces a single pure type `Diagnostic` carrying the spec's three fields — **category**, **cause**, **next step** — and one total normalizer `Diagnose(err) Diagnostic` that computes all three from the same refined error in one `errors.As` chain. `reportClientError` is refactored to delegate to `Diagnose`, so the category and message can never be computed from divergent sources, and — critically — the **structured value is now separable from the human print**, which is what Output-Aware Failure Rendering (032) needs to render a failure in any `--output` format.

Data flow (unchanged in shape, consolidated in substance):

```
command action → reportClientError(stderr, err)
                   └─ refineClientError(err)          // *ResponseError → *ProblemError, once (015)
                   └─ Diagnose(refined) → Diagnostic{Category, Cause, NextStep}
                        ├─ human stderr render (today's surface; 032 takes over per --output)
                        └─ return Diagnostic.Category → dispatch → ExitCode(category) → os.Exit  (004)
```

`Diagnostic` lives in `internal/cli` because it carries an `Outcome` and consumes both `internal/apiclient` errors and dispatch's usage errors — and `internal/apiclient` must never import `internal/cli` (010/015 layering). The category vocabulary is exactly `exitcode.go`'s taxonomy; `Diagnose` classifies, `ExitCode` maps — the producer-classifies / consumer-maps split (002/004/011/015) is preserved, now expressed as one value instead of two function calls.

---

## Architecture Decisions

### ADR-1: Consolidate classification + cause + next-step into a single `Diagnostic` value produced by one `Diagnose` normalizer

**Context**: The spec demands "exactly one normalized diagnostic … carrying a cause, a category, and a next step, in the same shape regardless of which family the failure came from." Today those three are produced by `classifyClientError` (category) and `formatClientErrorMessage`/`clientErrorNextStep` (cause + next step) — two independent chains tied at `reportClientError`. The next feature (032, Output-Aware Failure Rendering) must render the diagnostic per `--output`, which requires a structured value, not a pre-printed string.

**Options considered**:
1. **Leave the two chains split; have 032 call both** — minimal change now. But it pushes the "one consistent shape" obligation onto 032 and keeps two chains that can drift in arm order (the exact hazard the comments in `clienterror.go`/`me.go` warn about).
2. **Introduce a `Diagnostic` value + total `Diagnose` normalizer in `internal/cli`** — one `errors.As` chain computes category, cause, and next step together per arm; `reportClientError` delegates. 004 reads `Diagnostic.Category`; 032 reads the whole value.

**Decision**: Option 2. `Diagnose(err) Diagnostic` is the single source of truth. Each `errors.As` arm (auth, transport, non-2xx-by-status, decode, base-URL/rcfile, format, fallback) sets all three fields, so category and message are computed from the same matched value in the same place — they cannot disagree. `classifyClientError` is folded into `Diagnose` and either deleted or retained as a one-line `Diagnose(err).Category` shim for the three direct callers that need only the category and print their own message (`me.go`'s output-format path, `roles.go:215` and `subroles.go:237` partial-result paths). `formatClientErrorMessage`/`clientErrorNextStep` are absorbed into the arms.

**Consequences**: One chain to reason about; 032 has a clean structured value to render; the arm-order hazards documented today live in exactly one function. Direct category-only callers must switch to `Diagnose(err).Category` (or the shim) — an SoT-grep is required so no sibling keeps classifying independently. The exact symbol names, field names, and whether `NextStep` is `""` vs an `*string` for "no next step" are interface-level (`/score:interface`).

### ADR-2: DIVERGENCE — reclassify a decode error from `RuntimeError`(1) to `APIError`(3)

**Context**: Shipped precedent (`clienterror.go`, pinned by `clienterror_test.go:38` `{"decode-is-runtime", &DecodeError{200}, RuntimeError}`, and echoed by 011/019's "undecodable-body → RuntimeError(1)") maps a 2xx body that won't decode to the internal-error code 1. The clarified spec (Clarifications 2026-06-10, developer-confirmed during plan) classifies it as a **general API error (3)**: the call succeeded at the wire but the API returned an unreadable shape — an API-exchange problem, not a CLI bug.

**Options considered**:
1. **Keep decode → RuntimeError(1)** — no test churn; but conflates an API-shape mismatch with a genuine CLI/internal fault, and contradicts the clarified spec.
2. **Decode → APIError(3)** — matches the spec; yields a coherent split where exit 3 = "the API's fault" (non-2xx *or* unreadable 2xx) and exit 1 = "our fault" (panic, render-template failure, fallback).

**Decision**: Option 2. The decode arm sets `Category: APIError`. The cause wording stays as today's ("the API response did not match the expected shape — this may be an API change; report it (…)"), which already matches the spec's cause; only the category changes. This **supersedes** the shipped `classifyClientError` decode precedent and the 011/019 "undecodable-body → RuntimeError(1)" wording.

**Consequences**: `clienterror_test.go:38` flips to `APIError`; any other assertion of decode→1 must be grepped and updated (exitcode/integration tests). Render-template failures (019) stay at RuntimeError(1) — they are genuinely CLI-internal, so the split sharpens rather than blurs. This divergence is a candidate for `/score:deprecate` to formally record that the prior decode-classification precedent is retired.

### ADR-3: `Diagnostic` is a cli-side value; the envelope mapping and the cobra usage-error path stay 032's concern

**Context**: `internal/output.ErrorEnvelope{message, kind, status?, body?}` (018) is the wire shape for a failure, and 018 placed the typed-error→envelope mapping "at the cli/020 boundary." Separately, cobra-native usage errors (unknown command/flag) are produced *and rendered* by cobra+dispatch on their own path, not through `reportClientError`. The spec lists usage as a normalized family, but re-routing cobra's path is a distinct change (developer chose "stay focused").

**Options considered**:
1. **031 also maps `Diagnostic`→`ErrorEnvelope` and re-routes the cobra usage path now** — broadest, but duplicates 032's reason to exist and risks regressing cobra's established usage output.
2. **031 produces the cli-side `Diagnostic` only; defer envelope mapping and cobra-usage rendering to 032** — `Diagnose` is *able* to represent a usage error (category UsageError + cause + "see --help"), but the cobra unknown-command/flag surface (already cause+help+UsageError) is left untouched; 032 reconciles both into the unified envelope when it owns rendering.

**Decision**: Option 2. `Diagnose` covers the `reportClientError` error-value families (transport, decode, typed-API, base-URL, rcfile, format, auth, command-originated usage). The `Diagnostic`→`ErrorEnvelope` mapping (cause→`message`, category→`kind`; `status`/`body` read from the wrapped `*ResponseError` via `errors.As`, per 018) and the cobra usage-path rendering are 032's.

**Consequences**: 031 stays a focused consolidation in `internal/cli` and does not touch `dispatch.go`'s cobra path or `internal/output`. 032 plugs into `Diagnose` and reaches the `*ResponseError` for `status`/`body` exactly as 018 specified. The spec's "one consistent shape" is satisfied at the *value* level now; uniform *rendering* across the cobra surface lands with 032.

---

## Diagnostic Composition

How each `errors.As` arm fills the three fields (the substance `Diagnose` carries; refinements from the 2026-06-10 clarifications marked **[new]**):

| Matched error | Category | Cause | Next step |
|---|---|---|---|
| `*AuthError{NoCredentials}` | UsageError | "not authenticated" | run `glassfrog auth login` / set `GLASSFROG_TOKEN` |
| `*AuthError{CredentialError}` | RuntimeError | names the credentials file (never the token) | fix / re-create the credentials file |
| `*TransportError` | NetworkUnavailable | names the wire failure | check connectivity; the API may be unreachable |
| `*ProblemError`/`*ResponseError`, status 401 | PermissionError | API `detail` (or status fallback) | **[new]** verify the configured API token |
| …status 403 | PermissionError | API `detail` (or status fallback) | **[new]** your identity may lack the required role membership/permission |
| …status 429 | RateLimited | API `detail` (or status fallback) | **[new]** wait for the rate-limit window to reset (per the rate-limit headers) and retry |
| …any other non-2xx | APIError | API `detail` (or status fallback) | generic "check access and retry / consult the status code" |
| `*DecodeError` | **[changed → ADR-2]** APIError | "the API response did not match the expected shape — this may be an API change" | report it |
| `*BaseURLError` / `*rcfile.ReadError` / `*rcfile.FormatError` (base-URL path) | UsageError | names the source | correct `--base-url` / `GLASSFROG_BASE_URL` / `.glassfrogrc base_url` |
| `*output.FormatError` | UsageError | names the invalid selector | correct the `--output` / env / rcfile value |
| anything else | RuntimeError (fail-safe) | verbatim error string (token-free by apiclient contract) | — (none) |

**No-next-step case**: when no reliable recovery exists (the generic-API fallback retains today's generic line; the fail-safe arm has none), `Diagnose` leaves the next step empty rather than fabricating one (spec Non-Behaviors; CONSTITUTION VIII). The 401-vs-403 split refines today's single combined "check the token's access / membership" hint; the rate-limit refinement points at the reset window (`Retry-After` / `X-RateLimit-Reset` already on `*ResponseError.Header`) instead of the bare "retry later".

---

## Cross-cutting Concerns

**Token safety (load-bearing, tested)**: every cause and next step is path/status/detail only — never the `X-Auth-Token`. The consolidation must preserve this invariant arm-for-arm; the existing no-token-leak assertions are the guard and must cover every `Diagnose` arm.

**Totality / Fail Safe (CONSTITUTION III)**: `Diagnose` never panics and always returns a `Diagnostic`; an unrecognized error maps to the `RuntimeError` fallback (→ exit 1, never 0), matching today's `classifyClientError` fail-safe. Unrecognized/internal *crashes* that never reach `Diagnose` remain `main`'s panic-recover + `ExitCode`'s default-arm safety net (004) — 031 adds no competing internal-diagnostic path (spec Non-Behaviors).

**Testing strategy**: a table-driven `Diagnose` test asserting `{Category, Cause-substring, NextStep}` per family; reuse the `len`+comma-ok exhaustiveness guard so a dropped arm fails loudly (PR #10 learning). Flip `clienterror_test.go:38` decode→`APIError` and grep every other decode→RuntimeError assertion. Keep human-stderr **byte-equivalence** for unchanged families via golden capture before/after (capture stderr with a temp file, not `os.Pipe` — PR #10 deadlock learning). Assert no `Diagnose` arm emits the token.

**Configuration / observability**: none new. The human stderr message is the only rendered surface until 032; its wording changes only for the three refined arms (401, 403, 429) and the decode category.

---

## Implementation Strategy

Single phase, one package (`internal/cli`), TDD throughout (RED→GREEN per arm):

1. **Introduce the value, behavior-preserving** — add `Diagnostic` + `Diagnose` folding in `classifyClientError`/`formatClientErrorMessage`/`clientErrorNextStep` verbatim (decode still RuntimeError, permission hint still combined). Refactor `reportClientError` to delegate; switch the three category-only callers to `Diagnose(...).Category` (or the shim). Golden tests prove the human surface and all categories are unchanged. This is a pure refactor — green with the existing test suite plus the new `Diagnose` table.
2. **Apply the clarified behavior changes** — decode→APIError (ADR-2, flip the pinned test + grep siblings), the 401/403 next-step split, and the rate-limit reset-window wording. Each is a focused RED→GREEN with its own assertion.

Step 1 lands a safe consolidation; step 2 carries the only observable changes, each isolated and individually reversible. The tasks skill can split this into ~2 PR-sized units (consolidation; behavior refinements + deprecation note).

---

## Risks

- **Token leakage during arm consolidation** (low likelihood, high impact): merging the message arms could accidentally route an error string that includes the token. *Mitigation*: the existing no-token assertions must run against every `Diagnose` arm; add cases for any arm not already covered.
- **Human-output drift for unchanged families** (medium / medium): folding two functions into one risks subtle wording/spacing changes that break callers parsing stderr or golden tests. *Mitigation*: byte-equivalence golden capture before/after for every unchanged family; only the three refined arms (+ decode category) may differ.
- **Decode reclassification ripple** (medium / medium): tests or integration assertions beyond `clienterror_test.go:38` may assume decode→exit 1. *Mitigation*: grep all decode→RuntimeError / exit-1 assertions before changing; update together; record the `/score:deprecate` entry so the retired precedent is explicit.
- **032 coupling mismatch** (low / medium): if `Diagnostic` omits what the envelope needs, 032 stalls. *Mitigation*: ADR-3 keeps `status`/`body` sourced from the wrapped `*ResponseError` via `errors.As` (per 018), so `Diagnostic` need only carry cause/category/next-step; the refined error remains reachable up the chain.

---

## What This Plan Does Not Cover

- **Exact symbols and signatures** — `Diagnostic` field names/types, `Diagnose` vs an alternative name, `NextStep` empty-vs-optional encoding, and whether `classifyClientError` survives as a shim: `/score:interface`.
- **Executable Gherkin** — the spec's driving/validation scenarios become `.feature` files: `/score:scenarios`.
- **Failure rendering per `--output`** — mapping `Diagnostic` (+ `*ResponseError`) to `output.ErrorEnvelope`, and routing the cobra usage surface through the unified envelope: **032 (Output-Aware Failure Rendering)**.
- **Exit-code mapping** — `ExitCode(Outcome)` is unchanged (004); 031 only feeds it a category.
