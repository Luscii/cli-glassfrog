# Interface Accord: API Error Extraction — Specification

**Feature**: 015-api-error-extraction
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4 — the `internal/apiclient` extraction capability (the `ProblemError` typed error wrapping 010's `ResponseError`, and the pure `ExtractProblem` function) and the grown shared `internal/cli` helpers (`classifyClientError`, `formatClientErrorMessage`, `reportClientError`) that turn the refined error into the operator-facing message and the reserved exit code 4.

---

This accord pins the Go API surface of API Error Extraction: the **`ProblemError`** typed error and the pure **`ExtractProblem`** function added to `internal/apiclient`, and the consumer-side contract changes to the shared `internal/cli` read-surface helpers (the `Outcome` enum gains `PermissionError`; `classifyClientError` splits **401/403 → permission(4)**; `formatClientErrorMessage` surfaces the API's `detail`; `reportClientError` refines once). There is **no command and no entry point** in this slice — extraction is a function call refining the error the read commands (011–014) already receive from `Execute`, so the *invocation* and *instructional* surfaces are **N/A**. The capability introduces **no configuration of its own** — no new `.glassfrogrc` key, env var, or flag (the fallback-detail wording is a fixed `[ASSUMED]` constant) — so the *configuration* surface is **N/A**. This accord defines the **refinement of 010's generic non-2xx carrier** — the consuming side of the seam 010's accord forecast ("API Error Extraction (015) refines `ResponseError` into typed API/permission errors").

**Naming note**: the Go symbols use the RFC 9457 term **`Problem`** (`ProblemError`, `ExtractProblem`), not `APIError`, deliberately — `internal/cli` already owns an `APIError` *Outcome* category, and a producer named `apiclient.APIError` that the consumer maps to `PermissionError`(4) for 401/403 would read as a contradiction at the mapping site. `ProblemError` ties the type to the wire format it parses (RFC 9457 Problem Details) and keeps `classifyClientError` legible (`*apiclient.ProblemError` → `PermissionError`).

---

## Surface

### Entry points — `internal/apiclient` (the produced capability)

| Function | Signature (shape) | Description |
|---|---|---|
| `ExtractProblem` | `(re *ResponseError) *ProblemError` | The pure extraction. Best-effort-parses `re.Body` as an RFC 9457 Problem Details document, **never gated on `Content-Type`**, and returns a `*ProblemError` carrying the **authoritative** status (`re.StatusCode`), the extracted `Type`/`Title`/`Detail` and `BodyStatus` when present, and the wrapped `*ResponseError`. **Total**: it never returns an error and never panics — an empty / non-JSON / HTML / member-missing body degrades to a status-derived fallback `Detail` (with `DetailSynthesized = true`) and the raw body preserved (ADR-2). On a parseable detail it sets `DetailSynthesized = false`; on a parseable body `status` it sets `BodyStatus`. Reads only the response-side `ResponseError`; never the request token. |

### Output contract — `ProblemError` (typed, code-free)

The refined non-2xx error. Wraps the originating `*ResponseError` so `errors.As(err, &ResponseError)` still matches and the raw carrier stays reachable; consumers branch on the status (or match `*ProblemError`) **before** the bare `*ResponseError` arm.

| Field / accessor | Type | Description |
|---|---|---|
| `StatusCode` | `int` | The **authoritative** HTTP status (from the wrapped `ResponseError`). The body's own `status` member never overrides this (ADR-2). |
| `Type` | `string` | RFC 9457 `type` (a URI, default `about:blank`). Empty when the body wasn't parseable. |
| `Title` | `string` | RFC 9457 `title` (short summary). Empty when not parseable. |
| `Detail` | `string` | RFC 9457 `detail` (occurrence-specific explanation) when the body parsed and carried one; otherwise a **status-derived fallback** (e.g. `http.StatusText(StatusCode)`). Always non-empty for a known status. **Read `DetailSynthesized` to tell which it is** — the field alone does not distinguish API-provided detail from the fallback. |
| `DetailSynthesized` | `bool` | **Provenance marker**: `false` when `Detail` is the API's own parsed `detail`; `true` when `Detail` is the status-derived fallback (no parseable API detail). The consumer keys its message on this — `true` ⇒ render the `"status N"` fallback wording, `false` ⇒ render the API's detail. (Resolves the No-Fabricated-Data ambiguity: a synthesized detail is never presented as the API's own words.) |
| `BodyStatus` | `int` (optional, e.g. `*int`) | The body's RFC 9457 `status` member when the body parsed and carried one; **unset/nil otherwise**. Carried as **metadata only** — never authoritative, never overrides `StatusCode` (ADR-2). Present so a consumer can observe a body-vs-HTTP status disagreement (the spec/feature "carried as metadata only" assertion). |
| `Body` / `Header` | `[]byte` / `http.Header` | The raw body and response headers, via the wrapped `*ResponseError` (so 016 still reads paging headers, and the raw body/extension members stay available). |
| `Error() string` | — | A token-free message naming the status and the `Detail`. `Unwrap()` returns the wrapped `*ResponseError`. |

Input is 010's `*ResponseError{StatusCode, Header, Body}` — unchanged; 015 adds no field to it.

### Consumer contract changes — `internal/cli` (the shared read-surface helpers, grown)

| Symbol | Change | Description |
|---|---|---|
| `Outcome` enum (`dispatch.go`) | **+ `PermissionError`** | A new category (plus its `String()` arm) for an API auth/membership rejection (401/403). The producer 004 reserved code 4 for now exists. |
| `ExitCode` (`exitcode.go`) | **+ `case PermissionError: codePermissionError`** | Maps the new category to the already-pinned constant `codePermissionError = 4`. Stays a pure mapper; `default → 1` fail-safe unchanged. No code renumbered. |
| `classifyClientError` (`clienterror.go`) | **status split** | For a `*ProblemError`/`*ResponseError`: **401 or 403 → `PermissionError`**(4); everything else (incl. **429, until 017**) → `APIError`(3). Checked before the generic `*ResponseError` arm given the wrapping. Keeps its `len`+comma-ok table-test exhaustiveness guard. |
| `formatClientErrorMessage` (`me.go`) | **detail surfacing + next step** | For the refined error: when `DetailSynthesized == false`, render the API's `Detail` (and `Title` when useful); when `DetailSynthesized == true`, render the existing `"the API returned a non-2xx response: status N"` fallback wording. **Append a next-step hint** so the arm matches its siblings and CONSTITUTION II ("…and the next step") — at minimum for `PermissionError` (401/403): "check the token's access / membership". Token-free (response-side fields only). |
| `reportClientError` (`me.go`) | **refine once** | Before format+classify, refines a `*ResponseError` into a `*ProblemError` via `ExtractProblem` (guarding against double-refinement), so the typed error travels up the chain and message+category are computed from the same value. |

### Status → Outcome → exit-code mapping (consumer side)

| API status | `Outcome` | Exit code | Note |
|---|---|---|---|
| 401, 403 | `PermissionError` | **4** | The split this slice adds — fills the reserved permission code. |
| 429 | `APIError` | 3 | Unchanged here; Rate-Limit Handling (017) later splits → rate-limit(5). |
| Any other non-2xx (400, 404, 5xx, …) | `APIError` | 3 | The residual generic "API error" bucket, now carrying the API's detail in its message. |

**Example (shapes, not literal values)**:
```
// re = the *ResponseError 010 returned for a non-2xx.
valid problem+json:  ExtractProblem(&ResponseError{404, hdr, `{"type":"about:blank","title":"Not Found","status":404,"detail":"Not Found"}`})
                       → &ProblemError{StatusCode:404, Type:"about:blank", Title:"Not Found", Detail:"Not Found", DetailSynthesized:false, BodyStatus:&404, <wraps re>}
empty body:          ExtractProblem(&ResponseError{500, hdr, ``})
                       → &ProblemError{StatusCode:500, Detail:"Internal Server Error", DetailSynthesized:true /*fallback*/, BodyStatus:nil, <wraps re>}
HTML gateway body:   ExtractProblem(&ResponseError{502, hdr, `<html>…</html>`})
                       → &ProblemError{StatusCode:502, Detail:"Bad Gateway", DetailSynthesized:true /*fallback*/, BodyStatus:nil, <wraps re>}
body status mismatch: ExtractProblem(&ResponseError{403, hdr, `{"status":401,"detail":"Forbidden"}`})
                       → &ProblemError{StatusCode:403 /*authoritative*/, Detail:"Forbidden", DetailSynthesized:false, BodyStatus:&401, <wraps re>}   // 401 carried as metadata only
// consumer:
classifyClientError(&ProblemError{StatusCode:403, …})  → PermissionError   → ExitCode → 4   (message: detail + "check the token's access / membership")
classifyClientError(&ProblemError{StatusCode:404, …})  → APIError          → ExitCode → 3   (message: API detail, or "status 404" fallback when DetailSynthesized)
```

---

## Interactions

**Refine-once flow**: the read command calls `client.Execute` once; on a non-2xx it gets a `*ResponseError` and hands it to the shared `reportClientError`, which refines it to a `*ProblemError` **once** (`ExtractProblem`), writes the operator message from `formatClientErrorMessage`, and returns `classifyClientError`'s category — all computed from the same typed value, so the message and the exit code can never disagree. No per-command (011–014) edit is needed; the chokepoint already exists.

**Best-effort parse, graceful degradation**: `ExtractProblem` attempts to decode the body as RFC 9457 Problem Details regardless of the declared `Content-Type`. A valid body yields `Type`/`Title`/`Detail`; an empty, non-JSON, HTML, or member-missing body yields a status-derived fallback `Detail` with the raw body preserved. It never fails to produce a `*ProblemError`.

**Status authority**: the `StatusCode` on the result is always the HTTP status the wire returned (the wrapped `ResponseError`'s). A disagreeing in-body `status` member is captured as metadata only and never promoted to the authoritative field.

**Builds on 010's seam**: `ProblemError` wraps `010`'s `ResponseError` rather than replacing it — the raw body (RFC 9457 extension members included) and the response headers remain reachable through the wrapped value, so Pagination (016) and any programmatic caller still read what they need off the same chain.

---

## Error Communication

`ExtractProblem` itself communicates no failure — it is total and always returns a usable `*ProblemError`. The operator-facing communication happens at the shared `internal/cli` helpers:

| Condition | Message (stderr) | Outcome → exit code |
|---|---|---|
| `DetailSynthesized == false` (API detail parsed) | the API's `detail` (and `title` where useful) — the API's own cause — **+ a next-step hint** (≥ for 401/403: "check the token's access / membership") | 401/403 → `PermissionError`(4); else → `APIError`(3) |
| `DetailSynthesized == true` (body unparseable) | fallback: `"the API returned a non-2xx response: status N"` — **+ the same next-step hint** | same as above (status drives the code regardless of body parseability) |

The consumer keys the message on `DetailSynthesized` (the provenance marker), **not** on whether `Detail` is empty — the fallback always fills `Detail`, so emptiness can't distinguish the two. The next-step hint satisfies CONSTITUTION II ("…and the next step") and matches the sibling `formatClientErrorMessage` arms (auth → "run `glassfrog auth login`", transport → "check connectivity", base-URL → "correct --base-url").

**Detail is response-side only**: the surfaced `detail`/`title` come from the API's reply; the `X-Auth-Token` is a *request* header and is never echoed, so no 015 output can carry the token. Pinned by a token-never-in-output test over `ProblemError.Error()` and the rendered message (mirrors 010's test).

**Code-free at the producer; consumer maps**: `ExtractProblem`/`ProblemError` carry no exit code (continuing the producer-classifies/consumer-maps split — 002/004/005/007/008/009/010/011). The reserved code 4 is taken at `internal/cli`'s single `ExitCode` registry, not in `apiclient`. 429 stays `APIError`(3) until 017; 403 maps to `PermissionError`(4), **not** to plan-availability guidance (the Unsignalled Plan Limits problem owns that messaging).

---

## Consistency Notes

- **Joins 010's `apiclient` error family**: `ProblemError` follows the `<noun>Error`, `errors.As`-able, `Unwrap`-able, token-free shape of `ResponseError`/`TransportError`/`DecodeError`/`AuthError`/`BaseURLError`. It wraps `010`'s `ResponseError` (the value 010's accord built), so this accord is the refinement 010's Consistency Notes forecast ("015 refines it into typed API/permission errors").
- **Deliberate `Problem` naming** (deviation from the spec's "API error" wording): the Go symbols use `ProblemError`/`ExtractProblem` rather than `APIError`/`ExtractAPIError` to avoid colliding with the `cli.APIError` *Outcome* and to keep the `classifyClientError` mapping legible (a `*apiclient.APIError → PermissionError` mapping would read as a contradiction). The capability/spec name stays "API Error Extraction"; only the symbols change.
- **Reuses 011's shared read-surface helpers, doesn't fork them**: `classifyClientError`, `formatClientErrorMessage`, and `reportClientError` are the single chain 011 introduced and 012–014 reuse verbatim; 015 grows them in place (additive — only 401/403 change category, 3→4) rather than adding a parallel path. The exit-code split mirrors how 013's `validateStatus` and 011's `validateInclude` extended the shared surface.
- **Fills the reserved code 4, no renumber**: `PermissionError`(4) takes the constant 004 published and 011's comments forecast; 429→rate-limit(5) and the full 015/017 split note stay accurate. No existing code is renumbered (004's "Extension" rule).
- **No command, no new configuration**: like 005/006/007/008/009/010, this slice registers no cobra command and prints nothing of its own; invocation and instructional surfaces are N/A, and there is no new `.glassfrogrc` key, env var, or flag (the fallback-detail wording is a fixed `[ASSUMED]` constant). The fifth specification touchpoint in this project; no `accords/` directory exists, so there are no cross-spec accord patterns to align against.
