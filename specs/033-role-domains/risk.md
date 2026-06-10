# Risk: Role Domains

**Feature**: 033-role-domains
**Round**: 1 (first invocation)
**Date**: 2026-06-10
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial domains list (mid-walk failure or `--first-page` opt-out) is presented as the complete set | spec § Completeness of the list; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting | High | Low | RC-2 | Yellow |
| H-3 | An unknown `--include` value is silently ignored by `getDomain` → a domain returned without the requested policies embed (silent-wrong-results) | spec § Behavioral Accord (include); plan ADR-4; CONSTITUTION I/VIII | Medium | Low | RC-3 | Green |
| H-4 | A null `role_id` renders as an empty/ambiguous controlling-role line → misread as real data or as a missing field | interface § Surface (`domain` render); spec Domain (`role_id` nullable); CONSTITUTION VIII; checklist observation | Low | Low | RC-4 | Green |
| H-5 | A malformed/empty `--query` silently returns an empty list → "No domains." misread as "no matches" when the search was malformed | spec § List search; interface § Interactions; CONSTITUTION VIII | Low | Low | RC-5 | Green |
| H-6 | The domains multi-page walk contributes to org-wide `429` throttling | plan § System Architecture (RetryExecutor); CONSTITUTION X | Medium | Low | RC-6 | Green |
| H-7 | Domains surfaced beyond the caller's membership (over-exposure) | spec § Integration Boundaries; PROJECT.md Constraints; CONSTITUTION I | Medium | Low | RC-7 | Green |
| H-8 | Absence / omitted-section indicators (`No domains.`, unrequested policies section) misread as real data | spec § Output; CONSTITUTION VIII | Low | Low | RC-8 | Green |
| H-9 | Growing the shared `Domain` breaks the inline embeds on Role (025) / TreeNode (026) → regression in landed reads | plan ADR-2 / Risks; tasks T001 | Medium | Low | RC-9 | Green |
| H-10 | No ETag/304 caching → repeated full-list/single fetches add API load / rate-budget consumption | interface § Interactions (no caching); CONSTITUTION X | Low | Low | RC-10 | Green |

No residual risk is Red. Two Yellows (H-1, H-2) — the inherent read-surface pair, each acceptable with its documented control. All other hazards are Green.

---

## Hazard Detail

### H-1 — Partial domains list presented as complete
A role's domains can span many pages. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete set, an agent reviewing a role's areas of control would act on a false picture — the harm CONSTITUTION VI guards. **Severity High**; **Probability Low** (plan ADR-3 + interface mandate distinct explicit signals, reused verbatim from 025/026).
- **RC-1**: Default walks every page (`paging.All[Domain]`); `--first-page` prints one page with a "more domains exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits non-zero (`classifyClientError(Stop)`). The two causes are distinguished (interface § Interactions; validation scenario "List incompleteness is never silent").
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a signalled boundary, never silent.

### H-2 — Token leakage
Both commands sit on the live request path where `X-Auth-Token` is most exposed. **Severity High** (secret disclosure, CONSTITUTION II); **Probability Low** (the commands never read the token — 010's replay thunk and 007 own its only path).
- **RC-2**: The reads never read `ctx.Cred.Token`; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T002/T003 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path discipline + output test.

### H-3 — Unknown `--include` silently ignored
`getDomain` silently ignores an unknown `?include` value, returning the domain *without* the requested policies and no error — the silent-wrong-results hazard (the operator asked for policies and got none, with no signal). **Severity Medium**; **Probability Low** (the CLI validates locally).
- **RC-3**: `validateIncludeSet(cfg.include, {policies})` rejects any value outside `{policies}` as `UsageError(2)` before any request, with a transport tripwire (plan ADR-4; interface § Surface; scenario "An unsupported include value is rejected before any request"). The list rejects `--include` entirely (single-read concern).
- **Residual: Green** (Medium×Low).

### H-4 — Null `role_id` render ambiguity
The spec's `Domain.role_id` is nullable; the plan models it `*string` (T001). The `domain` single-read `full` render shows `Role: role_0456…`, but the contract does not yet pin how a **null** controlling role renders — risking an empty `Role:` line read as real data, or as a field the API failed to provide. No fabrication mechanism exists (a null renders absent, not invented — CONSTITUTION VIII holds), so the harm is low. **Severity Low**; **Probability Low**.
- **RC-4** *(control recommended, not yet pinned)*: Render a null `role_id` with an explicit-absence marker (the guarded-section / `{{if}}` pattern already used for the policies section and across 025/026), never a bare empty `Role:` line. **Recommendation**: add this to T003 acceptance and interface-cli § Surface so the absence treatment is contractual — this closes the lone checklist observation. Until then, the residual rests on the Low×Low rating, not on a landed control.
- **Residual: Green** (Low×Low) — acceptable; pinning RC-4 makes the absence explicit and tested.

### H-5 — Search false-empty on a malformed query
The API ignores an empty/whitespace `q` and returns no rows (rather than an error) for a malformed query. An operator who searches and sees `No domains.` cannot tell "no matches" from "malformed query." **Severity Low** (a re-runnable empty result, not corruption); **Probability Low**.
- **RC-5**: The CLI sends `q` only when the trimmed `--query` value is non-blank (plan ADR-3), so a blank term is observably no-filter rather than a silent empty search; a malformed-query empty result flows through the same explicit `No domains.` empty-success path (exit 0), which is a signalled valid answer, not a silent drop. The API's "malformed → empty, never error" behavior is spec-documented and faithfully surfaced (CONSTITUTION I/VIII).
- **Residual: Green** (Low×Low).

### H-6 — Domains walk contributes to throttling
The domains list issues one request per page, raising `429` exposure on a role with many domains. **Severity Medium** (throttling affects the whole org); **Probability Low**.
- **RC-6**: Each page goes through 017's landed `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5); page-size-at-max minimizes request count (CONSTITUTION X). The single read adds no extra request pressure (one GET).
- **Residual: Green** (Medium×Low).

### H-7 — Over-exposure beyond membership
The reads query a role's domains / a single domain; the hazard is surfacing domains the caller's membership shouldn't see. **Severity Medium**; **Probability Low** (the API enforces permissions per the token's membership server-side — PROJECT.md Constraints; the endpoints `404`/`401` what the caller can't reach).
- **RC-7**: The commands issue only the two defined `GET` operations (Spec Fidelity I); the server is the single authority on visibility; a contract/acceptance test pins the request shape. The CLI adds no client-side filtering and invents nothing.
- **Residual: Green** (Medium×Low).

### H-8 — Misread absence / omitted indicators
`No domains.` and an unrequested policies section could be misread as API-returned data. **Severity Low**; **Probability Low**.
- **RC-8**: Explicit absence indicators, never synthesized values (019 `{{if}}`/`missingkey=error`); an unrequested include section is omitted while a requested-but-empty one renders its marker; decode DTOs hold only API fields (CONSTITUTION VIII). Validation scenario "Default output carries no raw API envelope" backs the projection-only output.
- **Residual: Green** (Low×Low).

### H-9 — Shared `Domain` growth regresses the inline embeds
Role Domains grows the shared `glassfrog.Domain`, which `Role` (025) and `TreeNode` (026) embed as `[]Domain`. A non-additive change (a rename/removal/retag) would break those landed reads' decode or render. **Severity Medium** (regression in shipped reads); **Probability Low** (growth is additive by design and back-stopped by tests).
- **RC-9**: The growth is **additive only** — new fields (`Type`, `RoleID *string`, `CreatedAt`, `UpdatedAt`, `Policies`), no rename/removal of `ID`/`Description`; tasks T001 acceptance pins "the existing inline-on-Role and inline-on-TreeNode decode tests still pass" and `Policy` is reused, not redefined (plan ADR-2). The render registry exhaustiveness guard + the role/tree golden tests catch any drift.
- **Residual: Green** (Medium×Low).

### H-10 — No conditional-request caching
The CLI deliberately sends no `If-None-Match`, so repeated reads re-fetch and consume the rolling rate budget. **Severity Low** (latency/budget, not correctness); **Probability Low**.
- **RC-10**: Consistent with the sibling reads' posture (interface § Interactions states no `If-None-Match` / `ETag`/`304` path); the domains walk still benefits from 017's `429` backoff, and the single read is one GET. No correctness impact; revisitable when there's demand.
- **Residual: Green** (Low×Low).

---

## Residual Risk Summary

10 hazards, 10 controls, **0 Red**. Two Yellows: H-1 (list truncation) and H-2 (token leak) are the inherent read-surface pair, each fully controlled (dual completeness signals + non-zero exit; no-token-path discipline + output test). The eight Greens are controlled by the plan's existing decisions and the landed read stack. **One control, RC-4 (null `role_id` explicit-absence render), is recommended but not yet pinned in the contract** — it corresponds to the lone checklist observation; the hazard is Green on its own Low×Low rating, and pinning RC-4 in T003 acceptance + interface § Surface would make the absence treatment contractual. No hazard is unacceptable; nothing blocks implementation.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness of the list; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene) |
| H-3 | spec § Behavioral Accord (include); plan ADR-4; CONSTITUTION I/VIII |
| H-4 | interface § Surface (`domain` render); spec Domain (`role_id` nullable); CONSTITUTION VIII; checklist.md observation |
| H-5 | spec § List search; interface § Interactions; CONSTITUTION VIII |
| H-6 | plan § System Architecture (RetryExecutor); CONSTITUTION X |
| H-7 | spec § Integration Boundaries; PROJECT.md Constraints; CONSTITUTION I |
| H-8 | spec § Output / Behavioral Accord; CONSTITUTION VIII |
| H-9 | plan ADR-2 / Risks; tasks T001 |
| H-10 | interface § Interactions (no caching); CONSTITUTION X |
| RC-1 | interface-cli § Interactions (list completeness); validation scenario (incompleteness never silent) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T002/T003 |
| RC-3 | plan ADR-4; interface § Surface (`{policies}` validation); scenario (unsupported include rejected) |
| RC-4 | interface § Surface (`domain` render) — recommended T003 acceptance addition (null-`role_id` explicit-absence marker) |
| RC-5 | plan ADR-3 (non-blank-only `q`); interface § Interactions; spec § List search (malformed → empty) |
| RC-6 | plan § System Architecture (017 RetryExecutor); CONSTITUTION X |
| RC-7 | spec § Integration Boundaries; PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-8 | interface-cli § Surface (omit-vs-marker, `No domains.`); CONSTITUTION VIII |
| RC-9 | plan ADR-2 (additive growth, reuse `Policy`); tasks T001 acceptance (inline decode tests still pass) |
| RC-10 | interface § Interactions (deliberate, revisitable) |
