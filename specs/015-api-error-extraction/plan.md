# Plan: API Error Extraction

**Feature**: 015-api-error-extraction
**Role**: Shaper
**Inputs**: spec.md (015-api-error-extraction); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: `internal/apiclient` is the home for client/transport/response types — 008/009/010; 010 produces the generic code-free `ResponseError{StatusCode, Header, Body}` and explicitly never interprets it — 010 ADR-3; the producer-classifies-a-code-free-outcome / consumer-maps split runs through 002/004/005/007/008/009/010/011; 011 introduced the ONE shared `classifyClientError(err) Outcome` chain + `formatClientErrorMessage` + `reportClientError` chokepoint in `internal/cli`, reused verbatim by 012–017, mapping `*ResponseError → APIError(3)` and forecasting "015/017 SPLIT 401/403→permission(4) and 429→rate-limit(5) at this one registry without renumbering"; 004 publishes frozen codes 0–6 with **4 (permission) reserved** and a fail-safe `default→1`; `internal/glassfrog` is schema-only with no behaviour; `apiclient` never imports `internal/cli`); `.score/memory/LEARNINGS.md` (relevant: a change-detector table test needs a `len`+comma-ok exhaustiveness guard so a dropped mapping fails loud — PR #10; a godog suite points at its OWN feature file, never the whole `features/` dir; step helpers return errors, never panic; grep the package's existing `sc.Step(` registrations before adding phrasings). `.score/memory/DEPRECATION.md` has no entry bearing on 015. No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord with When/Then grouped by concern, three happy-path + two error + three edge scenarios, eight non-behaviors with reasoning, integration boundaries naming every neighbour (010/017/Unsignalled-Plan-Limits/004/the API), user scenarios, four assumptions, no remaining `[NEEDS CLARIFICATION]`. The three behavioral forks were resolved during `/score:define` (graceful degradation on an unreadable body; HTTP status authoritative over the body's `status` member; stay narrow — carry status + detail, leave 429-backoff to 017 and 403 plan-limit messaging to Unsignalled Plan Limits). **Two architectural decisions were resolved during the `/score:shape` resolve conversation**: (1) 015 is a **full slice** — it produces the `apiclient` extraction capability AND lands the consumer wiring (surface the API's `detail` in the message; split 401/403→permission(4) in `classifyClientError`), filling the reserved code 4 with its forecast producer while the `apiclient` capability itself decides no exit code; (2) the extracted detail reaches the operator through the shared `formatClientErrorMessage`, falling back to the current "status N" wording when the body wasn't parseable. The remaining `[ASSUMED]` items (the typed-error type/field names, the function signature, the fallback-detail wording) are interface/detail-level, not behavioral gaps.

---

## System Architecture

API Error Extraction is the capability that makes a non-2xx Glassfrog response **legible** — it pairs with the Next-tier *Opaque Failures* problem ("when a call fails, the caller can't tell what went wrong or what to do next"). Request Execution (010) deliberately stops at a **generic, uncategorized** `*apiclient.ResponseError{StatusCode, Header, Body}` and never interprets it (010 ADR-3). 015 takes exactly that value and refines it into a typed, code-free `*apiclient.APIError` carrying the authoritative HTTP status, the API's RFC 9457 Problem Details (`type`/`title`/`detail`), and the raw body/headers — so the operator sees the API's own cause instead of a bare status number, and the consuming command can branch on the status to assign the right exit code.

The work splits across the two packages the read surface already uses, with no new package:

- **`internal/apiclient` — the extraction capability (produces, decides no exit code).** A new typed error `APIError` (working name; interface pins it) that **wraps** the originating `*ResponseError` (so `errors.As(err, &ResponseError)` still matches and the raw carrier stays reachable), plus a pure `ExtractAPIError(*ResponseError) *APIError`. The function best-effort-parses the body as RFC 9457 Problem Details (`application/problem+json`: `type`, `title`, `status`, `detail`), **degrades gracefully** when the body isn't readable (empty, non-JSON, HTML from a gateway, or JSON missing the required members) by deriving a fallback detail from the HTTP status and keeping the raw body, and treats the **HTTP status as authoritative** — the body's own `status` member is captured as metadata only and never overrides. It never returns an error of its own and never reads the request token (it touches only the response-side `ResponseError`).
- **`internal/cli` — the consumer wiring (maps + presents).** The three shared helpers 011 introduced grow to understand the refined error:
  - `reportClientError` (the read-surface chokepoint that 011–014 all funnel through) **refines** a `*ResponseError` into a `*APIError` via `ExtractAPIError` once, so the typed error travels up the chain and both the message and the category are computed from it — no per-command edits.
  - `formatClientErrorMessage` surfaces the API's `detail` (and `title`) when present, falling back to the existing "the API returned a non-2xx response: status N" wording when extraction yielded no detail.
  - `classifyClientError` branches the API error on its status: **401/403 → `PermissionError`** (taking the reserved code 4), everything else (including 429, until 017) → `APIError` (code 3). This adds `PermissionError` to the `Outcome` enum + its `ExitCode` case.

```
internal/apiclient                       (the extraction capability — produces, no exit code)

  ExtractAPIError(re *ResponseError) *APIError          ── pure; never errors; response-side only
    │  parse re.Body as RFC 9457 problem+json (best-effort, NOT Content-Type-gated)
    │    ├─ valid          → APIError{Status: re.StatusCode (authoritative), Type, Title, Detail, *ResponseError}
    │    └─ unreadable     → APIError{Status: re.StatusCode, Detail: fallback(re.StatusCode), *ResponseError}
    └─ body `status` member, if it disagrees with re.StatusCode → carried as metadata, NEVER overrides

  APIError ── wraps *ResponseError (Unwrap); carries Status + Type/Title/Detail; Body/Header via the wrapped value

         consumed by ──►

internal/cli                              (the consumer — maps + presents; 011's shared helpers, grown)

  reportClientError(stderr, err)
    │  if errors.As(err, &re) && not already *APIError → err = apiclient.ExtractAPIError(re)   [ADR-4]
    │  Fprintln(stderr, formatClientErrorMessage(err))    ── surfaces .Detail / .Title, fallback "status N"  [ADR-4]
    └─ return classifyClientError(err), err               ── the typed *APIError travels up the chain

  classifyClientError(err) Outcome                        ── the ONE shared chain (011), grown  [ADR-3]
    ├─ *AuthError{NoCredentials}   → UsageError            (unchanged)
    ├─ *AuthError{CredentialError} → RuntimeError          (unchanged)
    ├─ *TransportError             → NetworkUnavailable    (unchanged)
    ├─ *APIError / *ResponseError  → status 401|403 → PermissionError(4)   [the split — reserved code 4]
    │                                else            → APIError(3)         (429 stays 3 until 017)
    ├─ *DecodeError                → RuntimeError          (unchanged)
    └─ base-URL / rcfile errors    → UsageError            (unchanged)
```

The split and the message enrichment are **purely additive on the existing seams**: a non-2xx that is neither 401 nor 403 still maps to `APIError(3)` exactly as before, so 011–014's landed behaviour is unchanged except that 401/403 now exits 4 (the reserved code) and every non-2xx message now carries the API's detail. `apiclient` still never imports `internal/cli`; the capability produces the typed error, the command maps and presents it.

---

## Architecture Decisions

### ADR-1: API Error Extraction is a pure `internal/apiclient` capability that refines 010's `*ResponseError` into a typed code-free `*APIError` wrapping it

**Context**: 010 produces the generic `*ResponseError{StatusCode, Header, Body}` and explicitly "does not classify it by failure kind and never decodes the body" (010 ADR-3, spec Non-Behaviors). 015's spec says the system "consumes the generic non-2xx outcome produced by Request Execution" and "produces one structured API error carrying the authoritative HTTP status code, the extracted detail/title/type, the raw body, and the response headers." `internal/apiclient` is the established home for client/transport/response types (008/009/010); `internal/glassfrog` is schema-only with no behaviour; `internal/cli` only classifies/formats/maps.

**Options considered**:
1. **A new typed `apiclient.APIError` + a pure `ExtractAPIError(*ResponseError) *APIError`** — the error **wraps** the `*ResponseError` (via `Unwrap`) so the raw carrier stays reachable and `errors.As(err, &ResponseError)` still matches; extraction is a standalone pure function. Keeps the parse in the package that owns the response shape; reusable by any caller (not just `cli`).
2. **Enrich `ResponseError` in place / parse inside 010's `Execute`** — rejected: directly contradicts 010's "generic, uncategorized, never interprets" non-behavior and its ADR-3; 010 already landed and is a separate capability; folding the parse there blurs the 010↔015 seam the project deliberately drew.
3. **Put the extraction in `internal/cli`** — rejected: parsing the API's response body into the RFC 9457 shape is a client/response-domain concern, not a CLI presentation concern; a future non-`cli` consumer (a programmatic caller, a test) would want the typed error too, and `cli` is meant to map/present, not parse wire bodies.

**Decision**: Option 1. A pure `ExtractAPIError(*ResponseError) *APIError` in `internal/apiclient`; `APIError` wraps the `*ResponseError`. Because it wraps, the consumer's classifier must check `*APIError` (or branch on status) **before** the bare `*ResponseError` arm — the same discrimination-order discipline 010/011 already apply for `*AuthError` before `*TransportError`. The exact type name (the `cli.APIError` *Outcome* shares the spelling; the package qualifier disambiguates, but interface may prefer `ProblemError`/`APIProblem`), field names, and signature are interface-level.

**Consequences**: The non-2xx carrier and its interpretation stay cleanly split across 010 and 015, each in `apiclient`. The typed error is `errors.As`-able and carries everything 016 (headers) and a programmatic caller (detail) need. `apiclient` still imports nothing from `cli`. *Precedent-setting: a non-2xx is refined by a pure `apiclient` function into a typed error that wraps 010's `ResponseError`; later response-interpretation capabilities (017) follow the same refine-the-carrier shape.*

### ADR-2: `ExtractAPIError` degrades gracefully and treats the HTTP status as authoritative (resolves spec forks 1 & 2)

**Context**: The Glassfrog API serves every 4xx/5xx as RFC 9457 Problem Details (`type`/`title`/`status`/`detail`, `application/problem+json`) — but gateways and proxies return empty, HTML, or non-conformant bodies, and RFC 9457's in-body `status` member can disagree with the actual HTTP status. The spec resolved (forks 1 & 2): the system must **never fail to produce a typed error because the body couldn't be parsed**, must **not gate parsing on `Content-Type`**, and the **HTTP status is authoritative** while the body's `status` is carried as metadata only.

**Options considered**:
1. **Best-effort parse + status-derived fallback; HTTP (ResponseError) status authoritative.** Attempt to decode the body as Problem Details regardless of `Content-Type`; on empty/non-JSON/missing-members, set a fallback detail from the status (Go's `http.StatusText`) and keep the raw body; ignore a disagreeing body `status` for the authoritative field (capture it as metadata). `ExtractAPIError` returns no error.
2. **Require `Content-Type: application/problem+json` before parsing** — rejected: spec Non-Behavior ("must not condition extraction on the response Content-Type"); gateways send problem bodies with varying/absent types, and gating would discard readable detail.
3. **Trust the body's `status` member** — rejected: spec fork 2; the transport status is what actually happened, and a self-described status that disagrees would mislead callers branching on the code.

**Decision**: Option 1. `ExtractAPIError` parses best-effort, degrades to a `http.StatusText`-derived fallback detail (the exact wording `[ASSUMED]`, interface-level), preserves the raw body on every path, and sets the authoritative `Status` from the `ResponseError` — the body's `status` is captured separately (or simply not promoted) and never overrides.

**Consequences**: A failed call is never opaque and never panics on a junk body (the empty/HTML/missing-members edges all yield a usable typed error). The function is total (returns `*APIError`, no error), which keeps the consumer flow branch-free. The fallback wording is a tunable detail, not a behavioral contract. *Feature-local; the totality + fail-soft-on-parse discipline is the notable property.*

### ADR-3: The exit-code split lives in the consumer (`classifyClientError`), reusing the status; the `apiclient` capability decides no exit code (resolves spec fork 3; reconciles the 004/011 forecast)

**Context**: 004 reserved code **4 (permission)** and 011's `classifyClientError`/`exitcode.go`/`dispatch.go` forecast that "API Error Extraction (015) splits 401/403→permission(4) ... at this one registry, without renumbering." But 015's spec says the capability "must not decide the process exit code" and must "stay narrow." The reconciliation (resolved in the shape conversation): the **split is consumer-side** — `classifyClientError` (the shared `internal/cli` chain, the "callers branch on the code" the spec describes) maps the status to the reserved code; 015's `apiclient` capability only carries the status.

**Options considered**:
1. **Split in `classifyClientError`, reusing the status.** Add `PermissionError` to the `Outcome` enum + its `ExitCode` case (taking the already-reserved constant `codePermissionError = 4`); the classifier branches the API error on status (401/403 → `PermissionError`, else → `APIError`). The `apiclient` type carries the raw status; `cli` maps it. Continues producer-classifies/consumer-maps (002/004/005/007/008/009/010/011).
2. **Bucket statuses into named categories inside the `apiclient` `APIError` type** — rejected: spec fork 3 chose "stay narrow"; and exit-code mapping is `cli`'s concern per 004/011, not `apiclient`'s (which would mean `apiclient` reaching toward exit codes).
3. **Leave code 4 unfilled / defer the split entirely** — rejected: the forecast assigns 015 as code 4's producer, and the *Opaque Failures* problem wants 401/403 (auth/membership rejection) distinguishable from a generic API error; deferring leaves a reserved code with no producer and the problem half-solved.

**Decision**: Option 1. `classifyClientError` gains the 401/403→`PermissionError`(4) split (checked on the `*APIError`/`*ResponseError` status, **before** the generic `*ResponseError`→`APIError` arm given the wrapping in ADR-1); `PermissionError` joins the `Outcome` enum (`dispatch.go`, with its `String()` arm) and `ExitCode` (`exitcode.go`). **429 stays `APIError`(3)** — Rate-Limit Handling (017) owns the 429→rate-limit(5) split, and 403 plan-limit *messaging* stays with Unsignalled Plan Limits (spec Non-Behaviors). No code is renumbered. The classifier's table test grows with the new mapping and keeps its `len`+comma-ok exhaustiveness guard (PR #10 LEARNINGS).

**Consequences**: The reserved code 4 is filled by its forecast producer; the `apiclient` capability stays exit-code-free (honoring the spec). 011–014 are unchanged except 401/403 now exits 4 (previously 3) — a published, reserved code, not a renumber. `outcomeToDispatchError`'s existing `default` arm already routes a non-Success/Usage/Runtime `Outcome` through `*outcomeError`, so `PermissionError` reaches `ExitCode` with no dispatch edit. *Precedent-setting: an API-response-interpretation capability fills its reserved exit code by splitting the generic `APIError`(3) at the shared `classifyClientError` registry on the status its `apiclient` type carries — 017 mirrors this for 429→rate-limit(5).*

### ADR-4: The extracted detail reaches the operator via `formatClientErrorMessage`, enriched once at the `reportClientError` chokepoint

**Context**: 015's spec says the system carries the detail so "the operator can read a human-meaningful cause," but it "must not print, or compose the user-facing message" itself — presentation is the command's. `formatClientErrorMessage` currently renders only "the API returned a non-2xx response: status N" for a `*ResponseError`, dropping the API's detail. The shape conversation resolved the detail should surface in that message (fallback to "status N" when the body wasn't parseable). The read surface funnels every client error through `reportClientError` (011), which calls both `formatClientErrorMessage` and `classifyClientError`.

**Options considered**:
1. **Enrich once at `reportClientError`.** Before format+classify, refine a `*ResponseError` into a `*APIError` via `ExtractAPIError`; pass the typed error to both helpers, so the message surfaces `.Detail`/`.Title` (fallback "status N") and the returned error that travels up the chain IS the typed `*APIError`. One shared edit; all reads (011–014) benefit; the category and the message are computed from the same typed value (they can never disagree — the `reportClientError` contract).
2. **Enrich in each command's `RunE`** — rejected: edits 011–014 (and every future read) and invites drift; the chokepoint exists precisely to avoid this.
3. **Parse independently inside `classifyClientError` and `formatClientErrorMessage`** — rejected: classification needs only the status (which `ResponseError` already carries — no parse needed), so this double-parses; and the error returned to the caller would stay the raw `*ResponseError`, not the typed `*APIError` the spec says the system "reports ... to the calling command."

**Decision**: Option 1. `reportClientError` refines `*ResponseError → *APIError` once (guarding against double-refinement when the error is already an `*APIError`); `formatClientErrorMessage` gains an `*APIError` arm that surfaces the detail (and title) with the "status N" fallback; the returned error is the typed `*APIError`. `classifyClientError` reads the same typed value (ADR-3).

**Consequences**: The operator sees the API's own cause (the *Opaque Failures* payoff) through one shared edit, and the typed error reaches the calling command as the spec requires. The message stays token-free (it renders only response-side fields — `detail`/`title`/`status`, never the request `X-Auth-Token`). The fallback path keeps 011–014's existing wording for unparseable bodies, so no read regresses. *Precedent-setting: response-interpretation refinement is applied once at the shared `reportClientError` chokepoint, not per command.*

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: 015 touches only the **response** side — `ExtractAPIError` reads `ResponseError`'s status, headers, and body (the API's reply); the `X-Auth-Token` is a *request* header and is never present. No 015 output (`APIError.Error()`, the surfaced detail/title, the fallback) can carry the token. Pinned by a test asserting the typed error and the rendered message never contain a token-shaped value, mirroring 010's token-never-in-output test.

**Error handling (CONSTITUTION III)**: the capability is **total and fail-soft on the parse** but the overall flow still **fails loud** — `ExtractAPIError` never returns an error and never panics on a junk body (it degrades), yet the result is always a typed error that the consumer maps to a non-zero exit code (401/403→4, else→3) and a non-empty message. A malformed body never becomes a silent success (the input is already a non-2xx). The body `status` mismatch is resolved deterministically (HTTP status wins, ADR-2).

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network. `ExtractAPIError` is unit-tested table-driven over crafted `ResponseError` values: a valid Problem Details body (detail/title/type extracted); extension members (standard four surfaced, raw body retained); an empty body (status fallback); a non-JSON/HTML gateway body (status fallback, no Content-Type gate); JSON missing required members (fallback); a body `status` that disagrees with the HTTP status (HTTP authoritative). The consumer wiring is tested at `internal/cli`: the `classifyClientError` table test grows with 401/403→`PermissionError` and a 429/500→`APIError` row, keeping the `len`+comma-ok exhaustiveness guard (PR #10 LEARNINGS); `exitcode_test.go` adds the now-live `PermissionError`→4 mapping (code 4 was already a pinned constant); `formatClientErrorMessage` tests assert detail-surfaced and the "status N" fallback. The driving scenarios become a **new `internal/cli` godog suite** pointed at its **own** feature file `features/opaque-failures/api-error-extraction.feature` (LEARNINGS — never the whole `features/` dir); step helpers return errors, never panic; reuse the existing read-surface step vocabulary where an assertion already exists (grep the package's `sc.Step(` registrations first).

**No new command surface**: 015 registers no cobra command and prints nothing itself — it grows the `apiclient` capability and the shared `cli` helpers the existing read commands already call. The cobra/help LEARNINGS do not apply.

---

## Implementation Strategy

Three phases, linear. Depends only on landed code: 010 (`ResponseError{StatusCode, Header, Body}`), 011 (`classifyClientError`, `formatClientErrorMessage`, `reportClientError`, the `Outcome` enum + `ExitCode`), 004 (the reserved `codePermissionError = 4`). Additive in `internal/apiclient` (new file) and a focused grow of 011's shared `internal/cli` helpers.

- **Phase 1 — the `apiclient` extraction capability**: define `APIError` (wraps `*ResponseError`; carries the authoritative `Status` + `Type`/`Title`/`Detail`; exposes `Body`/`Header` via the wrapped value) and the pure `ExtractAPIError(*ResponseError) *APIError` (best-effort RFC 9457 parse, no Content-Type gate, status-derived fallback, HTTP-status-authoritative, total/never-errors). RED-first table-driven unit tests over crafted bodies (valid / extensions / empty / HTML / missing-members / status-mismatch). Purely additive; no existing `apiclient` file changed. *Depends on: 010 (landed).*
- **Phase 2 — the consumer wiring (`internal/cli`)**: add `PermissionError` to the `Outcome` enum (`dispatch.go`) + its `String()` arm and its `ExitCode` case (`exitcode.go`, taking code 4); split `classifyClientError` (401/403→`PermissionError`, checked before the generic `*ResponseError` arm; 429/others→`APIError`); enrich `reportClientError` to refine `*ResponseError`→`*APIError` once; grow `formatClientErrorMessage` to surface `.Detail`/`.Title` with the "status N" fallback. Update the classifier table test (+ exhaustiveness guard), `exitcode_test.go` (code 4 now live), and the message tests; confirm 011–014's existing tests still pass (only 401/403 changes exit code, 3→4). *Depends on: Phase 1.*
- **Phase 3 — executable acceptance**: godog step definitions for the driving scenarios (valid extraction; 404 detail; extension preservation; empty-body degradation; non-JSON degradation; status-mismatch authority; 429-extracted-not-backed-off; 403-carried-generically) in a new `internal/cli` suite pointed at `features/opaque-failures/api-error-extraction.feature`. *Depends on: Phase 2.*

---

## Risks

- **Wrapping breaks the classifier discrimination order** (medium likelihood, medium impact): `APIError` wraps `*ResponseError`, so `errors.As(err, &responseErr)` matches an `*APIError` too; if the classifier checks the bare `*ResponseError` arm first, every 401/403 falls through to `APIError`(3) and the split silently never fires. *Mitigation*: branch on status in the `*APIError`/`*ResponseError` arm itself (one arm, status-driven) rather than two ordered arms — or check `*APIError` first; pin a test asserting a 401 maps to `PermissionError`(4), not `APIError`(3).
- **A read silently changes exit code 3→4** (medium likelihood, low impact): 401/403 responses that exited 3 now exit 4. This is intended (filling the reserved code) but is an observable change for any 011–014 caller scripting on exit 3. *Mitigation*: it is a published reserved code, not a renumber (004's convention); call it out in the handoff and the PR description; the `exitcode`/classifier tests pin the new mapping so it's deliberate, not incidental.
- **429 expected to split here** (low likelihood, low impact): a reader of the 004/011 forecast ("015/017 split ... 429→rate-limit(5)") might expect 015 to also map 429. *Mitigation*: 429→`APIError`(3) is explicitly retained (017 owns the 5 split); a test pins 429→`APIError`(3) for now, and the plan's "Does Not Cover" names it.
- **Malformed/junk body crashes extraction** (low likelihood, high impact): an empty, HTML, or truncated body must not panic. *Mitigation*: `ExtractAPIError` is total (ADR-2) — every parse failure degrades to the status fallback; the empty/HTML/missing-members edges are explicit table-test rows.
- **Token leak through the surfaced detail** (low likelihood, high impact): the message now renders API-supplied text. *Mitigation*: the detail is response-side (the API's `detail`, never the request token); a token-never-in-output test covers the typed error and the rendered message.

---

## What This Plan Does Not Cover

- **429 backoff/retry and the 429→rate-limit(5) exit-code split** — Rate-Limit Handling (017); 015 extracts a 429 into a typed error like any other non-2xx and leaves it at `APIError`(3).
- **403 plan-limit "not available on your plan" messaging** — the Unsignalled Plan Limits problem; 015 carries the 403 status + detail generically and maps it to `PermissionError`(4), not to plan-availability guidance.
- **Following pagination** off the response headers — Pagination (016), which reads the same `Header` the carrier exposes.
- **Structured `--output json` error rendering** — a deferred cross-cutting output flag (a Non-Behavior on the read surface since 011).
- **The exact type/field names and the `ExtractAPIError` signature** — `/score:interface` pins them (including whether the type is named `APIError`, `ProblemError`, or `APIProblem` to avoid the `cli.APIError` *Outcome* spelling).
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/opaque-failures/api-error-extraction.feature`.
