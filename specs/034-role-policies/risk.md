# Risk: Role Policies

**Feature**: 034-role-policies
**Round**: 1
**Date**: 2026-06-10
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial policy list (mid-walk failure or `--first-page` opt-out) is presented as the complete set | spec § Completeness; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting | High | Low | RC-2 | Yellow |
| H-3 | An unbounded walk on a role with very many policies → excessive round-trips / memory / runtime | plan § Cross-cutting (paging); CONSTITUTION VI | Medium | Low | RC-3 | Green |
| H-4 | The multi-page walk contributes to org-wide `429` throttling | plan § Cross-cutting (RetryExecutor); CONSTITUTION X | Medium | Low | RC-4 | Green |
| H-5 | Policies surfaced beyond the caller's membership (over-exposure) | spec § Non-Behaviors; PROJECT.md Constraints; CONSTITUTION I | Medium | Low | RC-5 | Green |
| H-6 | Absence indicators (null role/domain, `No policies.`) misread as real data | spec § Output; CONSTITUTION VIII | Low | Low | RC-6 | Green |
| H-7 | The full policy `body` (may contain HTML) is mis-rendered, or a very long body floods output | plan § Risks; interface-cli § Output; CONSTITUTION VI/VIII | Low | Medium | RC-7 | Green |
| H-8 | The `Document[T]` generalization regresses the landed 025 single-role read | plan ADR-2 / Risks; CONSTITUTION V/VII | Medium | Low | RC-8 | Green |
| H-9 | A wrong-kind id (`role_` to `policy`, or `pol_` to `policies`) silently reads the wrong resource | spec § Validation Scenarios; plan ADR-1/ADR-3 | Medium | Low | RC-9 | Green |
| H-10 | An empty `--query ""` issues a degenerate `q=` search | plan § Risks; ADR-3 | Low | Low | RC-10 | Green |

No residual risk is Red. Two Yellows (H-1, H-2), each acceptable with its documented control.

---

## Hazard Detail

### H-1 — Partial list presented as complete
A role's policies can span more than one page. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete set, an agent would act on a false picture of the governance on that role — the exact harm CONSTITUTION VI guards. **Severity High**: an agent acts inside a role on an incomplete policy set. **Probability Low**: plan Cross-cutting + interface mandate distinct, explicit signals for both incompleteness causes (reusing 025 ADR-3 verbatim).
- **RC-1**: The default path walks every page to completion (`paging.All[Policy]`). The `--first-page` opt-out prints one page with a "more policies exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits **non-zero** (`classifyClientError(Stop)`). The two causes are distinguished so a partial set is never read as the whole (spec § Completeness; interface-cli § Interactions; role-policies.feature mid-walk scenario).
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a visible, signalled boundary, never silent truncation.

### H-2 — Token leakage
Both commands sit on the live request path where `X-Auth-Token` is most exposed. **Severity High**: secret disclosure (CONSTITUTION II). **Probability Low**: the commands never read the token — 010's replay thunk and 007 own its only path.
- **RC-2**: The reads never read `ctx.Cred.Token`; errors carry only the response side and a network-level cause; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T003/T004 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path control and the explicit output test.

### H-3 — Unbounded walk on a role with very many policies
`paging.All` is unbounded (016 left `WithMaxPages` open), so a role with a pathological number of policies could drive many round-trips and large memory. **Severity Medium**: degraded responsiveness / resource use, not data corruption. **Probability Low** — and lower than the org-wide role list (025 H-3), since a single role's policy count is typically small.
- **RC-3**: Page size defaults to the API maximum (016), minimizing round-trips; `--per-page` and the `--first-page` opt-out give the operator explicit brakes; 016's non-advancing-cursor guard (`MalformedPageError`) prevents an infinite loop.
- **Residual: Green** (Medium×Low) — accepted; the brakes and the loop guard bound the worst case.

### H-4 — Multi-page walk contributes to org-wide throttling
The list issues one request per page, raising `429` exposure. **Severity Medium**: throttling affects the whole org. **Probability Low**: 017's `RetryExecutor` (landed) backs off on `429`/`Retry-After` for GET, and page-size-at-max minimizes the request count.
- **RC-4**: Each page goes through 017's `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5). The walk does not spin a tight retry loop of its own (CONSTITUTION X).
- **Residual: Green** (Medium×Low).

### H-5 — Over-exposure beyond membership
The hazard is surfacing policies the caller's membership shouldn't see. **Severity Medium**. **Probability Low**: the API enforces permissions per the token's membership server-side (PROJECT.md Constraints); the CLI adds no client-side filtering and invents nothing.
- **RC-5**: The commands issue only `GET /roles/{id}/policies` / `GET /policies/{id}` (Spec Fidelity I); the server is the single authority on what the membership may read; a contract/acceptance test pins the request shape.
- **Residual: Green** (Medium×Low).

### H-6 — Misread absence indicators
`(org-level — no role)` / `(whole-role — no domain)` / `(no body set)` / `No policies.` could be misread as API-returned data. **Severity Low**. **Probability Low**.
- **RC-6**: These are explicit absence indicators, never synthesized values (019 `{{if}}`/`missingkey=error`); the decode DTOs hold only API fields, with nullable `role_id`/`domain_id` rendered as their explicit-absence markers, never a guessed id (CONSTITUTION VIII; tasks T002).
- **Residual: Green**.

### H-7 — Policy body mis-rendered or floods output
The single-policy `full` view is the first template to render a long free-text `body` as primary content, and the schema notes the body "may contain HTML". A naive render could mangle the body, or a very long/HTML body could overwhelm the terminal. **Severity Low**: presentation only — no data corruption, and faithful rendering is the correct behavior. **Probability Medium**: HTML bodies are plausible in real governance data.
- **RC-7**: `text/template` (not `html/template`) renders the body verbatim, faithfully (CONSTITUTION VIII — present what the API returned), and is **deliberately not truncated or reflowed** (CONSTITUTION VI — faithfulness over brevity); the operator who wants brevity uses `compact` (body omitted) or a structured format. The faithfulness-vs-truncation choice is pinned in plan ADR-4, interface-cli § Output, and tasks T002 acceptance ("a long/HTML body is neither truncated nor reflowed").
- **Residual: Green** (Low×Medium) — accepted; verbatim rendering is the intended, faithful behavior, with `compact` as the brevity escape.

### H-8 — `Document[T]` generalization regresses the landed 025 read
034 refactors the landed `RoleDocument` (a Complete spec's shipped code) into `Document[RoleDetail]`. A botched refactor could break 025's single-role decode — a regression in a shipped read. **Severity Medium**: a working feature breaks. **Probability Low**: mitigated structurally.
- **RC-8**: The refactor is realized as a type alias (`type RoleDocument = Document[RoleDetail]`), so 025's command and decode call sites compile and behave unchanged; tasks T001 acceptance pins "RoleDocument remains usable at 025's existing call site — 025's decode tests still pass"; 024's pre-merge CI runs 025's full suite, so a regression fails the gate loudly (CONSTITUTION VII). Checklist flagged the same change as a P1-tracked V note.
- **Residual: Green** (Medium×Low).

### H-9 — Wrong-id-kind collision
The two-command split keys on id kind (`role_` → `policies`, `pol_` → `policy`). The hazard is a wrong-kind id silently reading the wrong resource and an agent acting on it. **Severity Medium**: acting on the wrong governance object. **Probability Low**: the two commands hit *distinct* endpoints with distinct path templates, and ids are passed through, so a wrong-kind id resolves to a clean `404`/`400` from a different endpoint — never silently the wrong resource.
- **RC-9**: Distinct endpoints (`GET /roles/{id}/policies` vs `GET /policies/{id}`) + id pass-through (plan ADR-3) mean a wrong-kind id surfaces the API's not-found, not a wrong success; the validation scenario "The plural and singular commands do not collide on id kind" pins that the wrong resource is never silently read.
- **Residual: Green** (Medium×Low).

### H-10 — Degenerate empty `--query`
`--query ""` could issue a `q=` empty search. **Severity Low**: at worst a no-op or a clean `400`. **Probability Low**.
- **RC-10**: `q` is sent only when `--query` is `Changed()` **and** non-empty (the 026 `--depth` optional-flag discipline; plan ADR-3 / Risk 3), so `--query ""` behaves as no filter rather than a degenerate API call.
- **Residual: Green**.

---

## Residual Risk Summary

10 hazards, 10 controls, **0 Red**. The two Yellows are the same High×Low pair the read surface carries throughout — H-1 (truncation, controlled by the dual completeness signals + non-zero exit on mid-walk failure) and H-2 (token leak, controlled by the no-token-path discipline + output test). The transferable read-surface hazards (H-3 unbounded walk, H-4 throttling, H-5 over-exposure, H-6 absence markers) are all Green with the same landed controls as 025/026. The four **034-specific** hazards are all Green: H-7 (HTML/long body — faithful verbatim render is intended, `compact` is the escape), H-8 (`Document[T]` regression — alias + 025's CI-run suite), H-9 (id-kind collision — distinct endpoints + clean 404), H-10 (empty query — send-only-when-set). No hazard requires resolution before implementation.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene) |
| H-3 | plan § Cross-cutting (paging); interface-spec § paging; CONSTITUTION VI |
| H-4 | plan § Cross-cutting (RetryExecutor); CONSTITUTION X |
| H-5 | spec § Non-Behaviors; PROJECT.md Constraints; CONSTITUTION I |
| H-6 | spec § Output / Behavioral Accord; CONSTITUTION VIII |
| H-7 | plan § Risks; interface-cli § Output; CONSTITUTION VI/VIII |
| H-8 | plan ADR-2 / Risks; CONSTITUTION V/VII |
| H-9 | spec § Validation Scenarios; plan ADR-1/ADR-3 |
| H-10 | plan § Risks; ADR-3 |
| RC-1 | spec § Completeness; interface-cli § Interactions; role-policies.feature (mid-walk + first-page) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T003/T004 |
| RC-3 | interface-spec § paging (page-size max, `--per-page`, `--first-page`, non-advancing guard) |
| RC-4 | plan § Cross-cutting (017 RetryExecutor); CONSTITUTION X |
| RC-5 | plan ADR-1 (GET endpoints); PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-6 | plan ADR-4 (decode DTOs hold only API fields); interface-cli § Output (absence markers); tasks T002 |
| RC-7 | plan ADR-4; interface-cli § Output (verbatim, no truncation); tasks T002 acceptance |
| RC-8 | plan ADR-2 / Risk 1 (alias); tasks T001 acceptance (025 tests still pass); CONSTITUTION VII (CI) |
| RC-9 | plan ADR-1/ADR-3 (distinct endpoints, id pass-through); validation scenario (no id-kind collision) |
| RC-10 | plan ADR-3 / Risk 3 (`Changed()`-and-non-empty gate) |
