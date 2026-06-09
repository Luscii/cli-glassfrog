# Risk: Role Reads

**Feature**: 025-role-reads
**Round**: 1
**Date**: 2026-06-08
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial role list (mid-walk failure or opt-out) is presented as the complete set | spec § Completeness; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting | High | Low | RC-2 | Yellow |
| H-3 | An unbounded walk on a very large org → excessive round-trips / memory / runtime | plan ADR-3; interface-spec § paging; CONSTITUTION VI | Medium | Low | RC-3 | Green |
| H-4 | The multi-page walk contributes to org-wide `429` throttling | plan § System Architecture (RetryExecutor); CONSTITUTION X | Medium | Low | RC-4 | Green |
| H-5 | Roles surfaced beyond the caller's membership (over-exposure) | spec § Non-Behaviors (org-wide); CONSTITUTION I | Medium | Low | RC-5 | Green |
| H-6 | Absence / omitted-section indicators misread as real data | spec § Output; CONSTITUTION VIII | Low | Low | RC-6 | Green |
| H-7 | Embedded `--include` view treated as the authoritative standalone read / downstream model duplication | spec § Non-Behaviors; plan ADR-2 | Medium | Low | RC-7 | Green |
| H-8 | Positional-id command shape forecloses per-role subcommands downstream | plan ADR-1 / Risks | Low | Medium | RC-8 | Green |
| H-9 | `?include=skills` summary misread as full skill content | spec § Behavioral Accord; plan ADR-2 | Low | Low | RC-9 | Green |

No residual risk is Red. Two Yellows (H-1, H-2), each acceptable with its documented control.

---

## Hazard Detail

### H-1 — Partial list presented as complete
The org-wide list can span many pages. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete set, an agent would act on a false picture of the org's governance — the exact harm CONSTITUTION VI guards. **Severity High**: org-wide governance decisions on an incomplete role set. **Probability Low**: plan ADR-3 + interface mandate distinct, explicit signals for both incompleteness causes.
- **RC-1**: The default path walks every page to completion (`paging.All`). The `--first-page` opt-out prints one page with a "more exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits **non-zero** (`classifyClientError(Stop)`). The two causes are distinguished so a partial set is never read as the whole (spec § Completeness; interface-cli § Interactions; validation scenario "List incompleteness is never silent").
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a visible, signalled boundary, never silent truncation.

### H-2 — Token leakage
The command sits on the live request path where `X-Auth-Token` is most exposed. **Severity High**: secret disclosure (CONSTITUTION II). **Probability Low**: the command never reads the token — 010's replay thunk and 007 own its only path.
- **RC-2**: The reads never read `ctx.Cred.Token`; errors carry only the response side and a network-level cause; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T002/T004 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path control and the explicit output test.

### H-3 — Unbounded walk on a very large org
`paging.All` is unbounded (016 left `WithMaxPages` open), so a pathological org could drive many round-trips, large memory, and long runtime. **Severity Medium**: degraded responsiveness / resource use, not data corruption. **Probability Low** after controls.
- **RC-3**: Page size defaults to the API maximum (016), minimizing round-trips; `--per-page` and the `--first-page` opt-out give the operator explicit brakes; 016's non-advancing-cursor guard (`MalformedPageError`) prevents an infinite loop. Streaming-vs-accumulate memory shape is an implementation detail for tasks.
- **Residual: Green** (Medium×Low) — accepted; the brakes and the loop guard bound the worst case.

### H-4 — Multi-page walk contributes to org-wide throttling
Unlike the single-request `/me*` reads, the list issues one request per page, raising `429` exposure on large orgs. **Severity Medium**: throttling affects the whole org. **Probability Low**: 017's `RetryExecutor` (landed) backs off on `429`/`Retry-After` for GET, and page-size-at-max minimizes the request count.
- **RC-4**: Each page goes through 017's `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5). The walk does not spin a tight retry loop of its own (CONSTITUTION X). This is a stronger control than the `/me*` reads had — 017 is now landed, not deferred.
- **Residual: Green** (Medium×Low).

### H-5 — Over-exposure beyond membership
The command intentionally queries the org-wide `GET /roles`; the hazard is surfacing roles the caller's membership shouldn't see. **Severity Medium**. **Probability Low**: the API enforces permissions per the token's membership server-side (PROJECT.md Constraints: "permissions follow that person's membership"); the CLI adds no client-side filtering and invents nothing.
- **RC-5**: The command issues only `GET /roles` / `GET /roles/{id}` (Spec Fidelity I); the server is the single authority on what the membership may read; a contract/acceptance test pins the request shape.
- **Residual: Green** (Medium×Low).

### H-6 — Misread absence / omitted-section indicators
`(none)` / `(no purpose set)` / `(none — anchor role)` / `No roles.`, and an omitted (unrequested) `--include` section, could be misread as API-returned data. **Severity Low**. **Probability Low**.
- **RC-6**: These are explicit absence indicators, never synthesized values (019 `{{if}}`/`missingkey=error`); an *unrequested* include section is omitted entirely while a *requested-but-empty* one renders its marker, so "not asked for" and "asked for, none exist" are distinguishable; the decode DTOs hold only API fields (CONSTITUTION VIII).
- **Residual: Green**.

### H-7 — Embedded-include treated as the standalone read / downstream duplication
`--include=policies` embeds policies inline; a consumer could treat that as the authoritative standalone policy read, or the downstream per-role specs (#33/#34/#38) could redefine `Policy`/`Assignment` rather than reuse them. **Severity Medium** (design/consistency). **Probability Low** after controls.
- **RC-7**: The spec Non-Behavior and plan ADR-2 pin the two-views-coexist boundary (embedded-on-the-role here; addressable standalone reads in #33/#34/#38); interface-spec.md sets the model-reuse precedent and DECISIONS records it. The validation scenario "An embedded include is not a standalone read" pins the distinction.
- **Residual: Green** (Medium×Low).

### H-8 — Positional-id forecloses per-role subcommands
`roles <id>` with `MaximumNArgs(1)` means cobra cannot distinguish a role id from a subcommand, so the downstream per-role reads cannot be children of `roles`. **Severity Low** (a design constraint, not runtime safety). **Probability Medium** (it will surface when #33/#34/#38/#26 are specified).
- **RC-8**: Plan ADR-1 and DECISIONS record the constraint explicitly, directing those specs to a singular `role <id> …` surface or flags. Surfaced now, before the dependent specs are built.
- **Residual: Green** (Low×Medium).

### H-9 — Skills summary misread as full content
`?include=skills` returns `SkillSummary` (no `content`); a consumer might assume full skill content is present. **Severity Low**. **Probability Low**.
- **RC-9**: The schema models `SkillSummary` with no `Content` field (plan ADR-2), and the render template labels it a summary — full content is fetched on demand via a future skills read. Nothing is fabricated (CONSTITUTION VIII).
- **Residual: Green**.

---

## Residual Risk Summary

9 hazards, 9 controls, **0 Red**. The two Yellows are the same High×Low pair the read surface carries throughout — H-1 (truncation, controlled by the dual completeness signals + non-zero exit on mid-walk failure) and H-2 (token leak, controlled by the no-token-path discipline + output test). H-4 (throttling) is **better controlled here than in the `/me*` reads** because 017's backoff is landed and applies to the walk. The org-wide-specific hazards (H-3 unbounded walk, H-5 over-exposure, H-7 include boundary, H-8 subcommand foreclosure) are all Green with documented controls. No hazard requires resolution before implementation.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene) |
| H-3 | plan ADR-3; interface-spec § paging (`WithPageSize`/`WithMaxPages`); CONSTITUTION VI |
| H-4 | plan § System Architecture (RetryExecutor); CONSTITUTION X |
| H-5 | spec § Non-Behaviors (org-wide, server-enforced); PROJECT.md Constraints; CONSTITUTION I |
| H-6 | spec § Output / Behavioral Accord; CONSTITUTION VIII |
| H-7 | spec § Non-Behaviors; plan ADR-2 |
| H-8 | plan ADR-1 / Risks |
| H-9 | spec § Behavioral Accord; plan ADR-2 |
| RC-1 | spec § Completeness; interface-cli § Interactions; validation scenario (incompleteness never silent) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T002/T004 |
| RC-3 | interface-spec § paging (page-size max, `--per-page`, `--first-page`, non-advancing guard) |
| RC-4 | plan § System Architecture (017 RetryExecutor); CONSTITUTION X |
| RC-5 | plan ADR-1 (GET /roles); PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-6 | plan ADR-2 (decode DTOs hold only API fields); interface-cli § Output (omit-vs-marker) |
| RC-7 | spec § Non-Behaviors; plan ADR-2; interface-spec (model reuse); validation scenario (embedded ≠ standalone) |
| RC-8 | plan ADR-1; DECISIONS (subcommand foreclosure recorded) |
| RC-9 | plan ADR-2 (`SkillSummary`, no content) |
