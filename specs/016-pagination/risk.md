# Risk: Pagination

**Feature**: 016-pagination
**Round**: 1
**Date**: 2026-06-07
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial result is rendered as the complete set (a consumer ignores `Complete`/`Stop`) | spec Non-Behaviors; plan ADR-3/Risks; CONSTITUTION VI | High | Low | RC-1, RC-2 | Yellow |
| H-2 | Infinite loop / duplicate explosion on a **non-advancing** cursor (blank **or repeated**) | spec edge scenarios; plan ADR-5 | High | Low | RC-3, RC-4 | Green |
| H-3 | Rate-limit exhaustion mid-walk on a large result set (`429`) | plan Cross-cutting/Risks; checklist X note; CONSTITUTION X | Medium | Medium | RC-5, RC-6 | Yellow |
| H-4 | The caller's other query params (`q`/`include`) dropped when re-issuing a page | spec Non-Behaviors; plan ADR-4 | Medium | Low | RC-7 | Green |
| H-5 | Records reordered, de-duplicated, or transformed across pages | spec Non-Behaviors; plan ADR-2; CONSTITUTION I/VIII | Medium | Low | RC-8 | Green |
| H-6 | The API token leaks into walker output or an error/log line | plan Cross-cutting; checklist II note; CONSTITUTION II | High | Low | RC-9 | Yellow |
| H-7 | A non-paginated endpoint (no `meta.pagination`) is mishandled | spec edge scenario; plan ADR-5 | Low | Low | RC-10 | Green |
| H-8 | The generic `Page[T]` divergence forks a second envelope / `Pagination` across 012–014 | plan ADR-2/Risks; DECISIONS §109 | Low | Medium | RC-11 | Green |
| H-9 | `cursor` is the wrong param name (spec prose says `after`) — paging silently fails to advance | spec Assumption; plan Risks; checklist I note | High | Low | RC-12, RC-4 | Yellow |

No residual risk is Red. Four Yellows (H-1, H-3, H-6, H-9) are acceptable with the documented controls below.

> **Key risk-driven control, now APPLIED (H-2):** the non-advancing-cursor guard was broadened in the contract (PR #36 review #1). ADR-5/T002/interface/spec now stop the walk when `next_cursor` is blank/absent **or identical to the cursor just used**, not only when blank — so an API that ignores an unrecognized `cursor` param (the H-9 wrong-name case) or a buggy `has_next_page` cannot loop the walker on a repeated page. This drops H-2 to **Green** and caps H-9's worst case at a flagged-incomplete stop. The walker tracks the prior cursor and the repeated-cursor tripwire is pinned in T002.

---

## Hazard Detail

### H-1 — Partial result rendered as complete
A consumer that does `if res.Stop != nil { return }` (the Go reflex) discards `res.Records`, or one that prints `Records` without checking `Complete`, presents a partial governance set as the whole — the exact harm CONSTITUTION VI guards against. **Severity High**: decisions on an incomplete set. **Probability Low**: the design makes the partial set a first-class field and the completeness an explicit flag.
- **RC-1**: `Result[T]` exposes `Records`, `Complete`, and `Stop` as separate fields (plan ADR-3; interface § Output contract) — the partial set is never shadowed by an error return, so the truncation can't happen silently inside 016.
- **RC-2**: A consumer-side scenario (when 012–014 adopt the walker) asserts the partial records are printed *and* an incompleteness note is emitted to stderr (interface § Error Communication; mirrors 012's `has_next_page` signal).
- **Residual: Yellow** (High×Low) — acceptable: 016 owns the shape; the render/signal is the consumer's, documented and scenario-pinned. The risk lives at the boundary 016 can't fully control.

### H-2 — Non-advancing cursor loop
`has_next_page=true` with a cursor that never advances makes the walker re-request indefinitely. **Severity High**: an unbounded loop (hang) and/or duplicate-record explosion; `Pages` grows without limit. **Probability Low**: requires a malformed/ignoring API response — but H-9 (wrong param name) is a concrete path to exactly this.
- **RC-3**: The guard stops the walk with a typed `MalformedPageError` when the cursor does not advance — blank/absent **or identical to the cursor just used** — rather than re-issuing (plan ADR-5; spec edge scenarios "blank cursor does not loop" / "repeated cursor does not loop"; interface § error type). The walker tracks the prior cursor to make the comparison.
- **RC-4**: Call-counting tripwire tests pin "does not loop" for **both** the blank and the repeated-cursor variants (task T002).
- **Residual: Green** (High×Low) — the broadened guard makes the loop structurally impossible for blank and repeated cursors alike; applied in the contract (PR #36 review #1), not deferred.

### H-3 — Rate-limit exhaustion mid-walk
A multi-page walk issues N requests; on a Free-plan org (50 req/hr) or a very large set, the rolling budget can be exhausted partway, returning `429`. **Severity Medium**: an incomplete read — but a *flagged* one, not corruption. **Probability Medium**: large orgs / Free plan / 017 not yet built.
- **RC-5**: `per_page` defaults to the API max (500) to minimize round-trips (plan ADR-4) — the single largest lever on request count.
- **RC-6**: A `429` mid-walk stops and returns the partial set flagged incomplete with the `429` as `Stop` (plan ADR-3; spec error scenario), rather than failing wholesale; Rate-Limit Handling (017), layered on the `Executor`, will add backoff without changing 016.
- **Residual: Yellow** (Med×Med) — acceptable: the design's purpose is to degrade gracefully here; 017 will further reduce probability.

### H-4 — Caller's query params dropped
Re-issuing a page by mutating or rebuilding `req.Query` could drop the caller's `q`/`include`, silently changing the result set being paged. **Severity Medium**: wrong-result-set (fabrication-adjacent). **Probability Low**: a single clone-and-set is simple.
- **RC-7**: The walker **clones** `req.Query` per page and sets only `per_page`+`cursor`, preserving all other params and never mutating the caller's map (plan ADR-4; spec Non-Behavior; scenario "The caller's query parameters are preserved across pages").
- **Residual: Green** (Med×Low).

### H-5 — Records reordered / de-duplicated / transformed
Any reordering, dedup, or transform of the gathered records misrepresents what the API returned. **Severity Medium**: a distorted governance view. **Probability Low**: append-in-order is the simple path.
- **RC-8**: The walker concatenates each page's `Data` in API order with no transform (plan ADR-2; spec Non-Behavior); the `@validation` scenario "Records are returned in API order without reordering or de-duplication" pins it.
- **Residual: Green** (Med×Low).

### H-6 — Token leakage
The walker sits on the live request path where the `X-Auth-Token` is most exposed. **Severity High**: secret disclosure (CONSTITUTION II). **Probability Low**: the walker never reads the token — it rides `*apiclient.Client` → 007's transport.
- **RC-9**: The walker never reads/holds the token and logs nothing; `Result.Stop` carries only response-side / network causes or a page index (plan § Cross-cutting); a token-never-rendered test covers the walk's outputs (task T002).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path control and the explicit output test (consistent with 010/012).

### H-7 — Non-paginated endpoint mishandled
An endpoint that returns no `meta.pagination` (the org role tree) must complete in one page, not error or fabricate a cursor. **Severity Low**: one endpoint family fails. **Probability Low**: absent `meta` decodes to the zero value (`HasNextPage=false`).
- **RC-10**: Tolerant decode of an absent `meta.pagination` → `HasNextPage=false` → the single response is the complete result (plan ADR-5; spec edge scenario; task T001 acceptance); the "non-paginated endpoint" scenario pins it.
- **Residual: Green** (Low×Low).

### H-8 — Envelope divergence forks the siblings
016's generic `glassfrog.Page[T]` supersedes the per-resource `MyRolesResponse` named in 012/013's not-yet-landed plans; parallel sessions could fork a second envelope or a second `Pagination`. **Severity Low**: cross-spec rework, not a runtime fault. **Probability Medium**: three concurrent sibling specs touch the shared type.
- **RC-11**: First-to-land reuse-or-create rule for `glassfrog.Pagination` with grep-before-add (task T001); the divergence is announced (plan ADR-2 / DECISIONS) with a `/score:deprecate` suggestion to retire the per-resource note.
- **Residual: Green** (Low×Med) — coordination control in place; no runtime exposure.

### H-9 — Wrong cursor parameter name
The v5 spec's prose intro pages with `after`, but the defined `Cursor` parameter is `name: cursor` and the `Pagination` schema says "Pass as `?cursor=`". If `cursor` is wrong and the API ignores it, the walk fails to advance — either stopping at page 1 (looking complete = **silent truncation**, the VI harm) or looping on a repeated cursor (H-2). **Severity High**: silent under-fetch presented as complete. **Probability Low**: two of three spec sources say `cursor`.
- **RC-12**: Verify the param name against the live API before consumers ship (plan Risks; checklist I note) — a one-line change if wrong.
- **RC-4** (shared, applied): the broadened non-advancing-cursor guard (now in the contract) bounds the *loop* variant of this failure to a flagged-incomplete stop. The remaining exposure is the *single-page* variant — if a wrong `cursor` is silently ignored and the API returns one page with `has_next_page=false`, the walk completes under-fetched. Only live-API verification (RC-12) closes that.
- **Residual: Yellow** (High×Low) — acceptable: the loop variant is closed by the broadened guard; the single-page under-fetch variant is held by the verification control. Track until `cursor` is confirmed against the live API.

---

## Residual Risk Summary

| Residual | Count | Hazards |
|---|---|---|
| Red | 0 | — |
| Yellow | 4 | H-1 (partial-as-complete, consumer boundary), H-3 (rate-limit mid-walk), H-6 (token leak), H-9 (wrong param name — single-page under-fetch variant) |
| Green | 5 | H-2 (non-advancing-cursor loop — guard now broadened to blank OR repeated), H-4, H-5, H-7, H-8 |

Using the default 3×3 traffic-light matrix — no project-level matrix in PROJECT.md. No residual risk is unacceptable (Red). The Yellows are inherent to a feature that walks a remote, rate-limited API and hands results to a separate consumer — each is reduced by a documented control. H-2 is now **Green**: the non-advancing-cursor guard was broadened (PR #36 review #1) to stop on a blank **or repeated** cursor, closing the loop hazard and the loop variant of H-9.

---

## Traceability Index

**Hazards → source**
- H-1 → spec § Non-Behaviors ("must not report partial as complete"); plan ADR-3 / Risks; CONSTITUTION VI
- H-2 → spec § Driving Scenarios (edge: blank cursor / repeated cursor); plan ADR-5
- H-3 → plan § Cross-cutting / Risks (017 composition); checklist X note; CONSTITUTION X
- H-4 → spec § Non-Behaviors (query params); plan ADR-4
- H-5 → spec § Non-Behaviors (no reorder/dedup); plan ADR-2; CONSTITUTION I/VIII
- H-6 → plan § Cross-cutting (secret hygiene); checklist II note; CONSTITUTION II
- H-7 → spec § Driving Scenarios (edge: non-paginated); plan ADR-5
- H-8 → plan ADR-2 / Risks; DECISIONS §109
- H-9 → spec § Assumptions (cursor name); plan § Risks; checklist I note

**Controls → architectural grounding**
- RC-1, RC-6 → plan ADR-3 (`Result[T]` partial-flagged-incomplete)
- RC-2 → interface § Error Communication (consumer render + stderr signal)
- RC-3, RC-4 → plan ADR-5 (`MalformedPageError` guard; non-advancing = blank **or repeated** cursor); task T002 blank + repeated tripwires
- RC-5, RC-7 → plan ADR-4 (default 500; clone-query, set only `per_page`+`cursor`)
- RC-8 → plan ADR-2 (concatenate in API order, no transform); `@validation` scenario
- RC-9 → plan § Cross-cutting (no token path); task T002 token-never-rendered test
- RC-10 → plan ADR-5 (absent `meta` → zero-value → complete); task T001
- RC-11 → plan ADR-2 / DECISIONS §109 (first-to-land reuse-or-create; announced divergence)
- RC-12 → plan § Risks (live-API `cursor` verification)
