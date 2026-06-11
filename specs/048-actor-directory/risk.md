# Risk: Actor Directory

**Feature**: 048-actor-directory
**Round**: 1
**Date**: 2026-06-11
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial result set (mid-walk failure or `--first-page` opt-out) is presented as the complete directory | spec § Completeness; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting; CONSTITUTION II | High | Low | RC-2 | Yellow |
| H-3 | An unsupported `--kind` value reaches the API or is silently narrowed → wrong scope / generic error instead of a discoverable local rejection | spec § Filters; plan ADR-3; CONSTITUTION I | Low | Low | RC-3 | Green |
| H-4 | A filter (`kind`/`role_id`/`q`) is dropped on page 2+ of the walk → later pages are unfiltered → a corrupted/over-broad result set | plan § Risks (filters on every page); interface § Interactions | Medium | Low | RC-4 | Green |
| H-5 | An empty/missing actor field (e.g. name) is rendered as a placeholder/invented value → fabricated governance data | plan ADR-4; spec § Output; CONSTITUTION VIII | Medium | Low | RC-5 | Green |
| H-6 | The org-wide directory surfaces actors beyond the caller's membership (over-exposure of people) | spec § Failure (permission scoping via shared chain); PROJECT.md Constraints; CONSTITUTION I | Medium | Low | RC-6 | Green |
| H-7 | A malformed `--role-id` the API rejects (`400`) is reported as success or an empty result rather than a failure | spec § Failure; interface § Error Communication; CONSTITUTION III | Medium | Low | RC-7 | Green |
| H-8 | The walk-by-default over a large organization contributes to org-wide `429` throttling | plan § Risks (large org walked by default); CONSTITUTION X | Medium | Low | RC-8 | Green |
| H-9 | Agent discovery is routed through the `ai_integration`-gated `/agents` alias → `--kind agent` fails (403) for orgs without the feature, instead of reading the ungated `/actors` | spec § System Overview / Non-Behaviors; plan ADR-1 | Medium | Low | RC-9 | Green |
| H-10 | The read-shaped `actors` command mutates as a side effect (a stray non-GET, or an unintended `?include`/write) | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX | High | Low | RC-10 | Yellow |

No residual risk is Red. Three Yellows (H-1, H-2, H-10) — each acceptable with its documented control (see detail).

---

## Hazard Detail

### H-1 — Partial directory presented as complete
The actor list can span many pages in a large org. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete set, an agent assembling context would act on a false picture of who is in the org — the harm CONSTITUTION VI guards. **Severity High**; **Probability Low** (plan Cross-cutting + interface mandate distinct explicit signals, reused verbatim from 025/026/038/041).
- **RC-1**: Default walks every page (`paging.All[Actor]`); `--first-page` prints one page with a "more actors exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits non-zero (`classifyClientError(Stop)`). The two causes are distinguished (interface § Interactions; scenario "A mid-walk failure yields a partial set flagged incomplete").
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a signalled boundary, never silent.

### H-2 — Token leakage
The `actors` command sits on the live request path where `X-Auth-Token` is most exposed. **Severity High** (secret disclosure, CONSTITUTION II); **Probability Low** (the command never reads the token — 010's replay thunk and 007 own its only path).
- **RC-2**: The read never reads `ctx.Cred.Token`; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T002 acceptance "no token in any output").
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path discipline + output test.

### H-3 — Unsupported `--kind` reaches the API
An out-of-set `--kind` value, if passed through, returns the API's generic error (or silently narrows) instead of a discoverable client-side error naming the supported set. **Severity Low** (a discoverable usage error, no wrong governance data); **Probability Low** (the CLI validates locally).
- **RC-3**: The `--kind` validator rejects any value outside `{human, agent}` as `UsageError(2)` before any request, with a transport tripwire; omitting `--kind` sends no param (plan ADR-3; interface § Surface; scenario "An unsupported kind is rejected as a usage error" + validation "A rejected kind issues no request").
- **Residual: Green** (Low×Low).

### H-4 — Filters dropped on later pages of the walk
The walker must carry `kind`/`role_id`/`q` on every page request, not just the first. If page 2+ dropped them, later pages would be unfiltered, silently widening the assembled set beyond what the operator asked for. **Severity Medium**; **Probability Low** (the walker threads the base request; the plan calls this out explicitly as a risk to test).
- **RC-4**: The filters are attached to the base request the walker reuses for every page; a unit test (tasks T002 acceptance) and the validation scenario "The filters are carried on every page of the walk" assert the page-2 request retains them (plan § Risks; interface § Interactions).
- **Residual: Green** (Medium×Low).

### H-5 — Fabricated absent field
An actor field rendered for the directory could be empty. If an empty value rendered as a placeholder or invented string, the operator would read fabricated governance data. **Severity Medium**; **Probability Low** (the render uses absence guards + `missingkey=error`).
- **RC-5**: The `actors` render shows only the `id`/`name`/`kind` the API returned, with an absence guard for any empty field; no synthesized value is presented as real, and the render preserves the API's returned order (no client re-sort/filter) (plan ADR-4; interface § Output; CONSTITUTION VIII; the 019 `{{if}}`/`missingkey=error` guards).
- **Residual: Green** (Medium×Low).

### H-6 — Over-exposure beyond membership
The directory is org-wide and lists people; the hazard is surfacing actors the caller's membership shouldn't see. **Severity Medium**; **Probability Low** (the API enforces permissions per the token's membership server-side — PROJECT.md single-org-+-person constraint; the spec's permission-scoping stance).
- **RC-6**: The command issues only the defined `GET /actors` operation (Spec Fidelity I); the server is the single authority on visibility; the CLI adds no client-side filtering and invents nothing; a contract/acceptance test pins the request shape (spec § Failure; PROJECT.md Constraints).
- **Residual: Green** (Medium×Low).

### H-7 — Malformed `--role-id` misreported
A `--role-id` the API rejects with `400` must surface as a failure, not a success or a misleading empty result. **Severity Medium**; **Probability Low** (the shared classifier routes non-2xx to a non-zero exit).
- **RC-7**: A `400` flows through `classifyClientError` → `APIError(3)`, naming the HTTP status on stderr and exiting non-zero; an empty (2xx) result is a distinct genuine-success path (`no actors`, exit 0) — the two are never conflated (interface § Error Communication; spec § Failure; CONSTITUTION III; scenarios "A malformed role-id filter fails with the API status" + "A free-text query matching no actor is a clean success").
- **Residual: Green** (Medium×Low).

### H-8 — Walk-by-default contributes to throttling
A large organization can span many actor pages; walking to completion by default issues one request per page and raises `429` exposure for the whole org. **Severity Medium** (throttling affects every caller in the org); **Probability Low** — a directory is bounded by org size, not by the long unbounded relevance tail that made 041's search walk a Medium×Medium Yellow; most orgs page out quickly.
- **RC-8**: Each page goes through 017's landed `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5); the operator narrows with `--kind`/`--role-id`/`--query` or caps with `--first-page` (plan § Cross-cutting/Risks; CONSTITUTION X).
- **Residual: Green** (Medium×Low) — bounded by org size with three narrowing brakes plus 017's backoff; less acute than the unbounded-search case (041 H-8).

### H-9 — Agent discovery routed through the gated alias
`/agents` is `ai_integration`-gated and deferred; the unified `/actors` is ungated. If `--kind agent` were implemented against `/agents` (or `/people`) rather than `/actors?kind=agent`, agent discovery would 403 for orgs without the feature — a capability the spec deliberately keeps reachable. **Severity Medium** (a working read becomes a hard failure for some orgs); **Probability Low** (plan ADR-1 fixes the single-`/actors`-endpoint shape).
- **RC-9**: The command reads only the unified `GET /actors`, selecting kind via the `kind` param; it never routes through `/agents` or `/people`. The validation scenario "Agent discovery reaches the ungated unified endpoint" pins that `--kind agent` reads `/actors` and not the gated alias (plan ADR-1; interface § Consistency Notes).
- **Residual: Green** (Medium×Low).

### H-10 — Read command mutates as a side effect
`actors` is a discovery read; a read path that issued a POST/PATCH/DELETE — or sent an unintended `?include` side-fetch — would violate Writes-Require-Explicit-Intent and could touch live governance. **Severity High** (an accidental mutation of governance/people data); **Probability Low** (the command is GET-only by construction and the spec forbids any write or embed).
- **RC-10**: The command issues only `GET /actors` and sends no `?include`; the spec Non-Behavior forbids creating/inviting/updating/deleting actors, and no POST/PATCH/DELETE path exists on the command; an acceptance test pins the request method/shape (spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX).
- **Residual: Yellow** (High×Low) — acceptable: read-only by construction, GET-only request shape pinned by test; the high severity reflects the consequence *were* it to occur, not its likelihood.

---

## Residual Risk Summary

10 hazards, 10 controls, **0 Red**. Three Yellows: H-1 (partial-as-complete) and H-2 (token leak) are the inherent read-surface pair shared with every sibling read, each fully controlled (dual completeness signals + non-zero exit; no-token-path discipline + output test); H-10 (read-mutates) is the read-only-surface Yellow — its High severity is the consequence-if-it-occurred, controlled to Low probability by the GET-only construction and a request-shape test. The data-integrity and scope Greens (H-3 local `--kind` validation, H-4 filters-on-every-page, H-5 no fabrication, H-6 server-side membership, H-9 ungated `/actors` only) are controlled by the plan's ADR-1/ADR-3/ADR-4 decisions and pinned by scenarios. Unlike 041, there is no Yellow for the page walk — a directory is bounded by org size, so H-8 is Green. No hazard is unacceptable; nothing requires resolution before implementation.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene); CONSTITUTION II |
| H-3 | spec § Filters; plan ADR-3; CONSTITUTION I |
| H-4 | plan § Risks (filters on every page); interface § Interactions |
| H-5 | plan ADR-4; spec § Output; CONSTITUTION VIII |
| H-6 | spec § Failure (permission scoping); PROJECT.md Constraints; CONSTITUTION I |
| H-7 | spec § Failure; interface § Error Communication; CONSTITUTION III |
| H-8 | plan § Risks (large org walked by default); CONSTITUTION X |
| H-9 | spec § System Overview / Non-Behaviors; plan ADR-1 |
| H-10 | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX |
| RC-1 | interface § Interactions (completeness signals); scenario (mid-walk partial flagged); CONSTITUTION VI |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T002 |
| RC-3 | plan ADR-3; interface § Surface (local `--kind` validation); scenarios (unsupported kind rejected / no request) |
| RC-4 | plan § Risks; interface § Interactions; validation scenario (filters carried on every page); tasks T002 |
| RC-5 | plan ADR-4; interface § Output (absence guard); CONSTITUTION VIII; 019 `missingkey=error` |
| RC-6 | spec § Failure; PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-7 | interface § Error Communication; spec § Failure; scenarios (malformed role-id failure / empty-result success) |
| RC-8 | plan § Cross-cutting/Risks (017 RetryExecutor); CONSTITUTION X; `--kind`/`--role-id`/`--query`/`--first-page` brakes |
| RC-9 | plan ADR-1; interface § Consistency Notes; validation scenario (ungated `/actors` only) |
| RC-10 | spec § Non-Behaviors; plan § What This Plan Does Not Cover; CONSTITUTION IX; request-shape acceptance test |
