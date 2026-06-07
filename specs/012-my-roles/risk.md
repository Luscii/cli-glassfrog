# Risk: My Roles

**Feature**: 012-my-roles
**Round**: 1
**Date**: 2026-06-07
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial role list is presented as the complete set (silent truncation) | spec Non-Behaviors (no page-following); CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting / Risks | High | Low | RC-2 | Yellow |
| H-3 | An error the agent can't recover from — no next step given | interface § Error Communication; checklist P0 (II) | Medium | Medium | RC-3 | Yellow |
| H-4 | Contributing to org-wide throttling on `429` | interface/plan ADR-3; CONSTITUTION X | Medium | Low | RC-4 | Green |
| H-5 | Roles shown that aren't the caller's (wrong endpoint / over-exposure) | spec Non-Behaviors (token-scoped, no selector); CONSTITUTION I | Medium | Low | RC-5 | Green |
| H-6 | Absence indicators misread as real data | spec § Output; CONSTITUTION VIII | Low | Low | RC-6 | Green |
| H-7 | Shared-scaffolding divergence with Identity Read (011) — *reconciled* | plan ADR-2 / Risks | Medium | Medium | RC-7 | Green |
| H-8 | Built against 011's unbuilt scaffolding (010 now implemented) | plan Phase 0 / Risks | Low | Medium | RC-8 | Green |

No residual risk is Red.

---

## Hazard Detail

### H-1 — Silent truncation
A practitioner with more roles than one API page receives a list that looks complete but isn't — a false picture of their governance footprint, the exact harm CONSTITUTION VI guards against. **Severity High**: governance decisions made on an incomplete role set. **Probability Low**: the plan and interface mandate an incompleteness signal derived from `meta.pagination.has_next_page`.
- **RC-1**: When more roles exist than the response carried, the command emits an explicit incomplete-result note to stderr and never presents the partial list as complete (spec accord; interface § Interactions; validation scenario "A partial list cannot be read as complete"). Full paging arrives with Pagination (016).
- **Residual: Yellow** (High×Low) — acceptable: the signal converts a silent truncation into a visible, exit-0-with-note boundary; complete paging is a known later capability.

### H-2 — Token leakage
My Roles is the first command on the live request path, where the `X-Auth-Token` is most exposed to tracing/logging. **Severity High**: secret disclosure (CONSTITUTION II). **Probability Low**: the command never reads the token — 010's replay thunk and 007 own its only path.
- **RC-2**: The command never reads `ctx.Cred.Token`; errors carry only the response side and a network-level cause; a token-never-in-output test covers stdout and stderr across every branch (plan § Cross-cutting; scenario discipline inherited from 010).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path control and the explicit output test.

### H-3 — Unrecoverable error (missing next step)
Three of six error conditions (`APIError`, `NetworkUnavailable`, decode `RuntimeError`) currently name a cause but no next step, so an AI-agent operator may not know how to recover — contradicting Action Transparency (II). **Severity Medium**: the agent acts on incomplete failure information. **Probability Medium**: half the error surface is affected.
- **RC-3**: Every error message names both the cause and a concrete next step (e.g. 403 → "you may lack permission"; transport → "check network/base URL"; decode → "API response shape changed — report it"). This is the checklist P0 fix.
- **Residual: Yellow** pre-fix (Medium×Medium); **Green** once RC-3 is applied. Flagged for resolution before implementation.

### H-4 — Org-wide throttling
A `429` is surfaced generically with no `Retry-After`/backoff (deferred to 017). **Severity Medium**: throttling affects the whole org. **Probability Low**: exactly one attempt per invocation — no retry loop, so no request storm originates here (the specific anti-pattern X names is absent).
- **RC-4**: One bounded attempt, no retry; the `429` is surfaced (status named) rather than hidden; backoff is Rate-Limit Handling's (017) concern.
- **Residual: Green** (Medium×Low) — accepted for the first read slice; revisit when 017 lands.

### H-5 — Over-exposure / wrong roles
If the command queried the org-wide `/roles` instead of the token-scoped `/me/roles`, it could surface roles the caller shouldn't see. **Severity Medium**. **Probability Low**: the spec, interface, and plan all pin `GET /me/roles`.
- **RC-5**: The command issues only `GET /me/roles` (Spec Fidelity I); a contract/acceptance test pins the path. The endpoint is token-scoped server-side.
- **Residual: Green** (Medium×Low).

### H-6 — Misread absence indicators
`(none)` / `(no purpose set)` / `No roles.` could be misread as API-returned values. **Severity Low**. **Probability Low**.
- **RC-6**: These are explicit absence indicators, never synthesized values; the decode DTO holds only API fields, so nothing is fabricated (CONSTITUTION VIII).
- **Residual: Green**.

### H-7 — Shared-scaffolding divergence with 011  *(RECONCILED 2026-06-07)*
Parallel sessions (011/012) could create divergent versions of the `me` command, `--base-url`, and the `Outcome`/`ExitCode` mapping. **This materialized and was resolved**: 011 landed first with a different exit-code mapping (auth fail-safe→2/1, no `PermissionError`) and a runnable `me` command. 012's plan/interface/tasks were **conformed** to 011's landed contract (developer decision 2026-06-07) — 012 now reuses `classifyClientError`, the runnable `me` command, the root `--base-url`, and `internal/glassfrog`, adding no new category and no registry edit.
- **RC-7**: Conformance to 011's recorded DECISIONS contract; 012 introduces no divergent mapping, so the value-pinning exit-code test (011) has nothing to reject. The one residual coordination item is the registration guard permitting `me` runnable-with-children (tracked in the plan).
- **Residual: Green** — the divergence is resolved, not merely controlled.

### H-8 — Built against an unbuilt dependency
010 (Request Execution) is now **implemented** on `main` ✓. The remaining unbuilt dependency is **011's scaffolding** (the runnable `me` command, `classifyClientError`, the root `--base-url`, `internal/glassfrog`) — recorded but not yet implemented; building T002/T003 before it lands fails. **Severity Low** (schedule, not runtime safety). **Probability Medium** (011 is ahead on `main` and will likely implement first).
- **RC-8**: Phase 0 lists both prerequisites; T002/T003 are gated on 011's scaffolding; T001 (schema growth) proceeds as soon as `internal/glassfrog` exists. If 012 reaches implementation first, it creates the shared pieces to 011's recorded contract.
- **Residual: Green** — sequencing is explicit; no runtime safety impact.

---

## Residual Risk Summary

8 hazards, 8 controls, 0 Red. **H-3 (error next-step) is resolved** — the cause+next-step rule was applied to interface-cli § Error Communication and the spec failure accord. **H-7 (divergence with 011) is reconciled** — 012 conforms to 011's landed mapping/scaffolding. The remaining Yellows (H-1 silent truncation, H-2 token leak) are acceptable with their documented controls; H-8 (depends on 011's scaffolding) is a sequencing item, not a runtime risk.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Non-Behaviors (no page-following), § Incomplete results; CONSTITUTION VI |
| H-2 | plan § Cross-cutting Concerns (Secret hygiene), § Risks |
| H-3 | interface-cli § Error Communication; checklist.md P0 (II) |
| H-4 | plan ADR-3; interface-cli § Error Communication; CONSTITUTION X |
| H-5 | spec § Non-Behaviors (token-scoped, no selector); CONSTITUTION I |
| H-6 | spec § Output / Behavioral Accord; CONSTITUTION VIII |
| H-7 | plan ADR-2, § Risks |
| H-8 | plan § Implementation Strategy (Phase 0), § Risks |
| RC-1 | spec accord + validation scenario; interface § Interactions (incompleteness signal) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test) |
| RC-3 | interface § Error Communication (cause + next step per condition) |
| RC-4 | plan ADR-3 / ADR-4 (one bounded attempt, no retry) |
| RC-5 | plan ADR-1 (GET /me/roles); CONSTITUTION I contract test |
| RC-6 | plan ADR-4 (decode DTO holds only API fields) |
| RC-7 | plan ADR-2/3 (jointly-held contract; guard + value-pinning test) |
| RC-8 | plan § Implementation Strategy (Phase 0 gate) |
