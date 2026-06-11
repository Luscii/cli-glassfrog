# Risk: Cross-Model Search

**Feature**: 041-cross-model-search
**Round**: 1
**Date**: 2026-06-11
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial result set (mid-walk failure or `--first-page` opt-out) is presented as the complete set | spec § Completeness; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting | High | Low | RC-2 | Yellow |
| H-3 | The query is rewritten/escaped/split client-side → the API searches for something other than what the operator typed (wrong results) | spec § Non-Behaviors (verbatim); plan ADR-1; CONSTITUTION I | Medium | Low | RC-3 | Green |
| H-4 | Results are re-sorted/de-duped/filtered client-side → the displayed ranking is not the API's relevance order (misleading "best match") | spec § Non-Behaviors (no re-sort); plan ADR-2; CONSTITUTION VIII | Medium | Low | RC-4 | Green |
| H-5 | A null `excerpt`/`role_id` is rendered as placeholder/invented text → fabricated governance data | spec § Output / Non-Behaviors; plan ADR-2; CONSTITUTION VIII | Medium | Low | RC-5 | Green |
| H-6 | An unsupported `--types` value reaches the API or is silently narrowed → wrong scope / generic error instead of a discoverable local rejection | spec § Behavioral Accord (type scoping); plan ADR-3; CONSTITUTION I | Low | Low | RC-6 | Green |
| H-7 | `query` (and `--types`) is dropped on page 2+ of the walk → later pages search differently/unscoped → a corrupted result set | plan § Risks (params on every page); interface § Interactions | Medium | Low | RC-7 | Green |
| H-8 | The walk-by-default over a broad, unbounded relevance query contributes to org-wide `429` throttling (long-tail walk) | plan ADR-4 / Risks; CONSTITUTION X | Medium | Medium | RC-8 | Yellow |
| H-9 | Results surfaced beyond the caller's membership (over-exposure) | spec § Cross-cutting (permission scoping); PROJECT.md Constraints; CONSTITUTION I | Medium | Low | RC-9 | Green |
| H-10 | A malformed query the API rejects (`400`) is reported as success or an empty result rather than a failure | spec § Failure; interface § Error Communication; CONSTITUTION III | Medium | Low | RC-10 | Green |
| H-11 | A multi-word query passed unquoted is split into `>1` positional → usage error (operator footgun) | plan ADR-1 / Risks; interface § Surface | Low | Medium | RC-11 | Green |
| H-12 | Eight result types in one ranked stream → the `type` badge fails to distinguish kinds → the operator drills into the wrong record | plan ADR-2 / Risks (render legibility); interface § Surface | Low | Low | RC-12 | Green |

No residual risk is Red. Three Yellows (H-1, H-2, H-8) — each acceptable with its documented control (see detail).

---

## Hazard Detail

### H-1 — Partial result set presented as complete
The result list can span many pages. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete set, an agent assembling context would act on a false picture — the harm CONSTITUTION VI guards. **Severity High**; **Probability Low** (plan ADR-4 + interface mandate distinct explicit signals, reused verbatim from 025/026).
- **RC-1**: Default walks every page (`paging.All[SearchResult]`); `--first-page` prints one page with a "more results exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits non-zero (`classifyClientError(Stop)`). The two causes are distinguished (interface § Interactions; validation scenario "A partial result set cannot be read as complete").
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a signalled boundary, never silent.

### H-2 — Token leakage
The `search` command sits on the live request path where `X-Auth-Token` is most exposed. **Severity High** (secret disclosure, CONSTITUTION II); **Probability Low** (the command never reads the token — 010's replay thunk and 007 own its only path).
- **RC-2**: The read never reads `ctx.Cred.Token`; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T002 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path discipline + output test.

### H-3 — Query rewritten client-side
The query is websearch syntax the API interprets. If the CLI parsed, escaped, normalized, or split it, the API would search for something other than what the operator typed — a silent wrong-results bug, and a Spec Fidelity violation (the API owns websearch interpretation). **Severity Medium**; **Probability Low** (plan ADR-1 and the interface mandate verbatim forwarding).
- **RC-3**: The positional is attached as `query` byte-for-byte with no client-side processing; a validation scenario ("The query reaches the API byte-for-byte") and tasks T002 acceptance pin that the outbound param equals the input exactly (plan ADR-1; interface § Interactions; CONSTITUTION I).
- **Residual: Green** (Medium×Low).

### H-4 — Results re-sorted client-side
Relevance ranking is the API's job and the ordering is the answer. A client-side re-sort, de-dup, or filter would present a different, unfaithful ranking — misleading the operator about what matched best. **Severity Medium**; **Probability Low** (plan ADR-2; the walker appends pages in sequence, so decode order = API order by construction).
- **RC-4**: The render preserves received order exactly — no sort/dedup/filter; a validation scenario ("The rendered order matches the API's relevance order") and tasks T002 acceptance pin decode-order = render-order (plan ADR-2; CONSTITUTION VIII).
- **Residual: Green** (Medium×Low).

### H-5 — Fabricated absent fields
`excerpt` and `role_id` are nullable. If a null rendered as a placeholder or invented string, the operator would read fabricated governance data. **Severity Medium**; **Probability Low** (the render uses absence markers + `missingkey=error`).
- **RC-5**: A null `excerpt` renders as the explicit marker `—`; a null `role_id` omits the `Role:` line; no synthesized value is ever presented as real (interface § Surface; CONSTITUTION VIII; the 019 `{{if}}`/`missingkey=error` guards).
- **Residual: Green** (Medium×Low).

### H-6 — Unsupported `--types` reaches the API
An out-of-set `--types` value, if passed through, returns the API's generic `400` (or silently narrows) instead of a discoverable client-side error naming the supported set. **Severity Low** (a discoverable usage error, no wrong governance data); **Probability Low** (the CLI validates locally).
- **RC-6**: `validateTypes` rejects any value outside `{role,note,project,action,skill,actor,policy,domain}` as `UsageError(2)` before any request, with a transport tripwire; omitting `--types` sends no param (plan ADR-3; interface § Surface; scenario "An unsupported type is rejected as a usage error" + validation "A rejected type issues no request").
- **Residual: Green** (Low×Low).

### H-7 — Query/scope dropped on later pages of the walk
The walker must carry `query` (and `types`) on every page request, not just the first. If page 2+ dropped them, later pages would search differently or unscoped, corrupting the assembled set. **Severity Medium**; **Probability Low** (the walker threads the base request; the plan calls this out explicitly as a risk to test).
- **RC-7**: `query` and `types` are attached to the base request the walker reuses for every page; a unit test (tasks T002 acceptance) and the scenario "The query and type scope are carried on every page of the walk" assert the page-2 request retains both (plan § Risks; interface § Interactions).
- **Residual: Green** (Medium×Low).

### H-8 — Walk-by-default contributes to throttling
Unlike a bounded list read, a broad relevance query has a long low-relevance tail; walking it to completion by default issues one request per page and raises `429` exposure for the whole org. **Severity Medium** (throttling affects every caller in the org); **Probability Medium** (a broad query genuinely can span many pages — this is the cost the walk-by-default clarify decision knowingly accepted).
- **RC-8**: Each page goes through 017's landed `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5); page size defaults to the API max to minimize request count; the operator narrows with `--types` or caps with `--first-page` (plan ADR-4 / Cross-cutting; CONSTITUTION X).
- **Residual: Yellow** (Medium×Medium) — **acceptable with documented justification**: walk-by-default was the deliberate clarify resolution (cross-command symmetry + strongest reading of VI), and the operator has two first-class brakes (`--types`, `--first-page`) plus 017's backoff. Revisitable if real-world usage shows broad-query throttling.

### H-9 — Over-exposure beyond membership
The search is org-wide and cross-type; the hazard is surfacing resources the caller's membership shouldn't see. **Severity Medium**; **Probability Low** (the API enforces permissions per the token's membership server-side — PROJECT.md Constraints; the spec's permission-scoping note).
- **RC-9**: The command issues only the defined `GET /search` operation (Spec Fidelity I); the server is the single authority on visibility; the CLI adds no client-side filtering and invents nothing; a contract/acceptance test pins the request shape.
- **Residual: Green** (Medium×Low).

### H-10 — Malformed query misreported
A query the API rejects with `400` must surface as a failure, not a success or a misleading empty result. **Severity Medium**; **Probability Low** (the shared classifier routes non-2xx to a non-zero exit).
- **RC-10**: A `400` flows through `classifyClientError` → `APIError(3)`, naming the HTTP status on stderr and exiting non-zero; an empty (2xx) result is a distinct genuine-success path (`No results.`, exit 0) — the two are never conflated (interface § Error Communication; spec § Failure; CONSTITUTION III; scenarios "A query the API rejects as malformed…" + "A search matching nothing is a clean success").
- **Residual: Green** (Medium×Low).

### H-11 — Unquoted multi-word query
`ExactArgs(1)` means an unquoted multi-word query is split by the shell into multiple positionals → a usage error. **Severity Low** (a clear usage error, no wrong results — fails safe); **Probability Medium** (an easy operator mistake).
- **RC-11**: `ExactArgs(1)` rejects a `>1` positional as `UsageError(2)` before any request, with a message that names the misuse and prompts quoting; help text and examples show the quoted form (plan ADR-1; interface § Surface/Consistency Notes; scenario "A missing query is a usage error").
- **Residual: Green** (Low×Medium) — fails safe and discoverable; the only cost is a retry with quotes.

### H-12 — Heterogeneous render misread
One flat key renders eight result types in a single ranked stream; if the `type` badge failed to distinguish kinds, the operator could drill into the wrong record. **Severity Low**; **Probability Low**.
- **RC-12**: Each row leads with its `type` badge and carries `type` + `id` (and `role_id` where present) so the kind and drill-in target are explicit; golden/unit render tests cover a mixed-type list (interface § Surface; plan ADR-2; tasks T002 acceptance).
- **Residual: Green** (Low×Low).

---

## Residual Risk Summary

12 hazards, 12 controls, **0 Red**. Three Yellows: H-1 (partial-as-complete) and H-2 (token leak) are the inherent read-surface pair shared with every sibling read, each fully controlled (dual completeness signals + non-zero exit; no-token-path discipline + output test). H-8 (walk-by-default throttling) is the one feature-specific Yellow — the deliberate cost of the clarify walk-by-default decision, controlled by 017's backoff and the operator's `--types`/`--first-page` brakes, and revisitable from real usage. The data-integrity Greens (H-3 verbatim query, H-4 preserve order, H-5 no fabrication, H-7 params-on-every-page) are controlled by the plan's ADR-1/ADR-2 decisions and pinned by validation scenarios. No hazard is unacceptable; nothing requires resolution before implementation.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene) |
| H-3 | spec § Non-Behaviors (verbatim); plan ADR-1; CONSTITUTION I |
| H-4 | spec § Non-Behaviors (no re-sort); plan ADR-2; CONSTITUTION VIII |
| H-5 | spec § Output / Non-Behaviors; plan ADR-2; CONSTITUTION VIII |
| H-6 | spec § Behavioral Accord (type scoping); plan ADR-3; CONSTITUTION I |
| H-7 | plan § Risks (params on every page); interface § Interactions |
| H-8 | plan ADR-4 / Risks; CONSTITUTION X |
| H-9 | spec § Cross-cutting (permission scoping); PROJECT.md Constraints; CONSTITUTION I |
| H-10 | spec § Failure; interface § Error Communication; CONSTITUTION III |
| H-11 | plan ADR-1 / Risks; interface § Surface |
| H-12 | plan ADR-2 / Risks; interface § Surface |
| RC-1 | interface § Interactions (completeness signals); validation scenario (incompleteness never silent) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T002 |
| RC-3 | plan ADR-1; interface § Interactions; validation scenario (query byte-for-byte); tasks T002 |
| RC-4 | plan ADR-2; interface § Surface; validation scenario (rendered order = API order); tasks T002 |
| RC-5 | interface § Surface (absence markers); CONSTITUTION VIII; 019 `missingkey=error` |
| RC-6 | plan ADR-3; interface § Surface (per-flag local validation); scenarios (unsupported type rejected / no request) |
| RC-7 | plan § Risks; interface § Interactions; scenario (query/types carried on every page); tasks T002 |
| RC-8 | plan ADR-4 / Cross-cutting (017 RetryExecutor); CONSTITUTION X; `--types`/`--first-page` brakes |
| RC-9 | spec § Cross-cutting; PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-10 | interface § Error Communication; spec § Failure; scenarios (malformed-query failure / empty-result success) |
| RC-11 | plan ADR-1; interface § Surface/Consistency Notes; scenario (missing query is a usage error) |
| RC-12 | interface § Surface (type badge + id); plan ADR-2; tasks T002 (mixed-type render test) |
