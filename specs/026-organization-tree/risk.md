# Risk: Organization Tree

**Feature**: 026-organization-tree
**Round**: 2 (re-run after clarify + propagation)
**Date**: 2026-06-09
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md, features/governance-reads/organization-tree.feature (test-gap analysis)
**Degradation flags**: No Regulatory Context in PROJECT.md — IEC 14971 bridge omitted. No project risk-acceptability matrix — using the default 3×3 traffic-light matrix.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | A partial subroles list (mid-walk failure or `--first-page` opt-out) is presented as the complete set | spec § Completeness of the subroles list; CONSTITUTION VI | High | Low | RC-1 | Yellow |
| H-2 | The API token leaks into stdout or an error line | plan § Cross-cutting | High | Low | RC-2 | Yellow |
| H-3 | A depth-capped tree node is indistinguishable from a true leaf → silent truncation at the `--depth` boundary | interface § Surface (tree render); plan ADR-2/ADR-4; CONSTITUTION VI; spec Clarifications | Medium | Low | RC-3 | Green |
| H-4 | `--depth 0` vs. omitted is conflated → a whole-tree read silently returns only the root | plan ADR-4 / Risks; interface § Surface | Medium | Low | RC-4 | Green |
| H-5 | An unknown `--include` value is silently ignored by the tree endpoint → a tree returned without the requested embed (silent-wrong-results) | spec § Behavioral Accord (include); plan ADR-4; CONSTITUTION I/VIII | Medium | Low | RC-5 | Green |
| H-6 | A large unpaginated tree response → excessive memory / runtime on a big org | plan § Risks; CONSTITUTION VI | Medium | Low | RC-6 | Green |
| H-7 | No ETag/304 caching → repeated full-tree fetches add API load / rate-budget consumption | spec § Non-Behaviors (no caching); plan § Cross-cutting; CONSTITUTION X | Low | Low | RC-7 | Green |
| H-8 | The subroles multi-page walk contributes to org-wide `429` throttling | plan § System Architecture (RetryExecutor); CONSTITUTION X | Medium | Low | RC-8 | Green |
| H-9 | Roles/subroles surfaced beyond the caller's membership (over-exposure) | spec § Integration Boundaries; PROJECT.md Constraints; CONSTITUTION I | Medium | Low | RC-9 | Green |
| H-10 | Absence / omitted-section indicators (`No subroles.`, leaf node, unrequested include) misread as real data | spec § Output; CONSTITUTION VIII | Low | Low | RC-10 | Green |
| H-11 | The shared `RoleDetail` schema drifts or double-defines if 025 and 026 land in parallel | plan ADR-3 / Risks; tasks T002 | Low | Medium | RC-11 | Green |

No residual risk is Red. Two Yellows (H-1, H-2) — the inherent read-surface pair, each acceptable with its documented control. H-3 dropped from Yellow to **Green** in round 2 once its control RC-3 landed in the contract (see detail + improvement summary).

---

## Hazard Detail

### H-1 — Partial subroles list presented as complete
The subroles list can span many pages. If a mid-walk failure or the `--first-page` opt-out produced output indistinguishable from a complete child set, an agent navigating the hierarchy would act on a false picture — the harm CONSTITUTION VI guards. **Severity High**; **Probability Low** (plan ADR-3 + interface mandate distinct explicit signals, reused verbatim from 025).
- **RC-1**: Default walks every page (`paging.All`); `--first-page` prints one page with a "more subroles exist" stderr note (exit 0); a mid-walk failure prints the partial set with an "incomplete — cause" stderr note and exits non-zero (`classifyClientError(Stop)`). The two causes are distinguished (interface § Interactions; validation scenario "Subroles incompleteness is never silent").
- **Residual: Yellow** (High×Low) — acceptable: every incomplete path is a signalled boundary, never silent.

### H-2 — Token leakage
The two commands sit on the live request path where `X-Auth-Token` is most exposed. **Severity High** (secret disclosure, CONSTITUTION II); **Probability Low** (the commands never read the token — 010's replay thunk and 007 own its only path).
- **RC-2**: The reads never read `ctx.Cred.Token`; a token-never-in-output assertion covers stdout and stderr across every branch (plan § Cross-cutting; tasks T003/T004 acceptance).
- **Residual: Yellow** (High×Low) — acceptable with the no-token-path discipline + output test.

### H-3 — Depth-capped node indistinguishable from a leaf
At a `--depth` boundary the API returns nodes with `has_subroles: true` and an empty `children`. If the human render shows nothing to distinguish such a node from a true leaf (`has_subroles: false`, empty `children`), the operator reads a capped tree as complete — silent truncation at the boundary that CONSTITUTION VI's signalling clause exists to prevent. **Severity Medium** (governance structure misread as smaller/complete); **Probability Low** (round 2) — RC-3 is now in the contract, so a capped node is explicitly marked.
- **RC-3** *(landed in the contract, round 2)*: The `tree` render surfaces a boundary marker for any node with `has_subroles: true` and absent `children` (`(+ subroles below depth)` in `full`, `has_subroles=yes` in `compact`), driven by the API boolean with no invented descendant count. Specified in spec.md (Behavioral Accord § Output + Clarifications), interface-cli.md § Surface, organization-tree.feature (driving "A depth-capped node is marked as having subroles below" + validation "A depth-capped node is distinguishable from a leaf"), and tasks T003 acceptance.
- **Residual: Green** (Medium×Low) — the depth boundary is now an explicit, tested signal; a capped branch is never read as empty.

### H-4 — `--depth 0` vs. omitted conflated
`0` (root node alone) is a meaningful value distinct from "omitted = full tree." If `--depth` were implemented as a plain int defaulting to 0, every whole-tree read would silently collapse to root-only — a wrong-results bug. **Severity Medium**; **Probability Low** (plan ADR-4, interface, and tasks T003 all mandate the optional / `Changed()` treatment).
- **RC-4**: `--depth` is an optional flag sent only when `cmd.Flags().Changed` is true; tasks T003 acceptance pins "omitting `--depth` sends no `depth`; `--depth 0` sends `depth=0`." A negative value is rejected locally.
- **Residual: Green** (Medium×Low).

### H-5 — Unknown tree include silently ignored
The tree endpoints silently ignore an unknown `?include` value, returning a tree *without* the requested embed and no error — the silent-wrong-results hazard (the operator asked for accountabilities and got none, with no signal). **Severity Medium**; **Probability Low** (the CLI validates locally per read).
- **RC-5**: `validateTreeInclude` rejects any value outside `{accountabilities,domains,members}` as `UsageError(2)` before any request, with a transport tripwire; `subroles` validates against its own set (plan ADR-4; interface § Surface; validation scenario "A typo'd tree include is caught locally").
- **Residual: Green** (Medium×Low).

### H-6 — Large unpaginated tree
The tree reads return the whole (depth-bounded) tree in one response; a pathological org could drive a large payload, memory, and decode time. **Severity Medium** (degraded responsiveness, not corruption); **Probability Low**.
- **RC-6**: `--depth` is the operator's brake on response size; the API is designed for the org sizes it serves (spec/endpoint note: tree returned whole by design). Streaming-vs-accumulate is a tasks-level implementation detail.
- **Residual: Green** (Medium×Low).

### H-7 — No conditional-request caching
The CLI deliberately sends no `If-None-Match`, so repeated full-tree reads re-fetch the whole tree, adding API load and consuming the rolling rate budget. **Severity Low** (latency/budget, not correctness); **Probability Low**.
- **RC-7**: A deliberate, documented non-behavior (spec) — revisitable when there's demand; the subroles walk still benefits from 017's `429` backoff, and the tree reads are single GETs. No correctness impact.
- **Residual: Green** (Low×Low).

### H-8 — Subroles walk contributes to throttling
The subroles list issues one request per page, raising `429` exposure on a circle with many children. **Severity Medium** (throttling affects the whole org); **Probability Low**.
- **RC-8**: Each page goes through 017's landed `RetryExecutor` (honors `Retry-After`/`X-RateLimit-*`, GET-only safe retry); 015 classifies a capped-out `429` to rate-limit(5); page-size-at-max minimizes request count (CONSTITUTION X). The tree reads add no extra request pressure (single GET each).
- **Residual: Green** (Medium×Low).

### H-9 — Over-exposure beyond membership
The reads query the org-wide tree / a role's subroles; the hazard is surfacing roles the caller's membership shouldn't see. **Severity Medium**; **Probability Low** (the API enforces permissions per the token's membership server-side — PROJECT.md Constraints; the spec's TreeNode notes children "hidden by authorization").
- **RC-9**: The commands issue only the three defined `GET` operations (Spec Fidelity I); the server is the single authority on visibility; a contract/acceptance test pins the request shape. The CLI adds no client-side filtering and invents nothing.
- **Residual: Green** (Medium×Low).

### H-10 — Misread absence / omitted indicators
`No subroles.`, a leaf node with nothing indented, and an unrequested include section could be misread as API-returned data. **Severity Low**; **Probability Low**.
- **RC-10**: Explicit absence indicators, never synthesized values (019 `{{if}}`/`missingkey=error`); an unrequested include section is omitted while a requested-but-empty one renders its marker; decode DTOs hold only API fields (CONSTITUTION VIII; spec Non-Behavior "never a 'no children' line invented as data"). *(Note: this control is what H-3 extends — the leaf/depth-cap distinction is the one absence case the current render does not yet make.)*
- **Residual: Green** (Low×Low).

### H-11 — Shared `RoleDetail` schema drift with 025
The subroles read decodes `Page[RoleDetail]`; `RoleDetail` + leaf models + `Role` growth are the same schema 025 designs, and neither has landed. Parallel sessions could double-define or drift them. **Severity Low** (compile collision / drift, caught at build); **Probability Medium** (parallel Conductor workspaces).
- **RC-11**: First-to-land-creates — tasks T002 is written create-if-absent-else-reuse and the branching guidance flags role-based awareness so only one definition lands (the 005/006/016 pattern). The `tree` half carries no such dependency.
- **Residual: Green** (Low×Medium).

---

## Residual Risk Summary

11 hazards, 11 controls, **0 Red** (round 2). Two Yellows: H-1 (subroles truncation) and H-2 (token leak) are the inherent read-surface pair, each fully controlled (dual completeness signals + non-zero exit; no-token-path discipline + output test). **H-3 (depth-boundary truncation) dropped to Green** in round 2 — its control RC-3 (the `has_subroles` boundary marker + the depth-cap driving/validation scenarios) is now pinned across spec, interface, feature file, and tasks. The remaining tree-specific Greens (H-4 depth-0/absent, H-5 silent-ignored include, H-6 large tree, H-7 no caching) are controlled by the plan's existing decisions. No hazard is unacceptable; nothing requires resolution before implementation.

## Improvement Summary (Round 2)

**Round 1**: 11 hazards — 0 Red, 3 Yellow (H-1, H-2, H-3), 8 Green.
**Round 2**: 11 hazards — 0 Red, **2 Yellow (H-1, H-2)**, 9 Green.

- **Changed**: H-3 Yellow → **Green**. The clarify session + propagation landed RC-3 in the contract (spec Behavioral Accord/Clarifications, interface render marker, feature scenarios, tasks T003 acceptance), so the depth boundary is now an explicit, tested signal rather than a pending fix. Probability Medium → Low.
- No new hazards; no hazards removed; H-1/H-2 unchanged (inherent, controlled).

## Test Gap Analysis (Round 2)

Cross-referencing hazards against organization-tree.feature scenarios:

- **H-1** (subroles partial as complete) — covered: "A multi-page subroles list is walked to completion", "The subroles first-page opt-out stops at one page and signals more", "A mid-walk subroles failure yields a partial set flagged incomplete", validation "Subroles incompleteness is never silent".
- **H-3** (depth boundary) — now covered: "A depth-capped node is marked as having subroles below" + validation "A depth-capped node is distinguishable from a leaf".
- **H-5** (silent-ignored include) — covered: validation "A typo'd tree include is caught locally" (the per-read reject mechanism; subroles shares it).
- **H-2** (token leak) — covered by a cross-cutting token-never-in-output assertion (tasks T003/T004 acceptance), not a feature scenario — appropriate, as it is an output-hygiene invariant rather than a behavioral flow.
- **H-4 / H-6 / H-7 / H-8 / H-9 / H-10 / H-11** — accepted Greens; H-4 (depth-0≠absent) is pinned by tasks T003 acceptance; the rest are controlled by architecture/non-behaviors rather than scenarios. No new gap.

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | spec § Completeness of the subroles list; CONSTITUTION VI |
| H-2 | plan § Cross-cutting (secret hygiene) |
| H-3 | interface § Surface (tree render); plan ADR-2/ADR-4; CONSTITUTION VI; spec Clarifications 2026-06-09 |
| H-4 | plan ADR-4 / Risks; interface § Surface (`--depth`) |
| H-5 | spec § Behavioral Accord (include); plan ADR-4; CONSTITUTION I/VIII |
| H-6 | plan § Risks (large unpaginated tree); CONSTITUTION VI |
| H-7 | spec § Non-Behaviors (no caching); plan § Cross-cutting; CONSTITUTION X |
| H-8 | plan § System Architecture (RetryExecutor); CONSTITUTION X |
| H-9 | spec § Integration Boundaries; PROJECT.md Constraints; CONSTITUTION I |
| H-10 | spec § Output / Behavioral Accord; CONSTITUTION VIII |
| H-11 | plan ADR-3 / Risks; tasks T002 |
| RC-1 | interface-cli § Interactions (subroles completeness); validation scenario (incompleteness never silent) |
| RC-2 | plan § Cross-cutting (replay thunk, token-never-in-output test); tasks T003/T004 |
| RC-3 | interface § Surface (tree render boundary marker); spec Behavioral Accord/Clarifications; organization-tree.feature (depth-cap driving + validation); tasks T003 acceptance |
| RC-4 | plan ADR-4; interface § Surface; tasks T003 acceptance (`--depth 0` ≠ omitted) |
| RC-5 | plan ADR-4; interface § Surface (per-read validation); validation scenario (typo'd tree include caught locally) |
| RC-6 | plan § Risks; interface § Surface (`--depth` brake) |
| RC-7 | spec § Non-Behaviors (deliberate, revisitable); plan § Cross-cutting |
| RC-8 | plan § System Architecture (017 RetryExecutor); CONSTITUTION X |
| RC-9 | spec § Integration Boundaries; PROJECT.md membership enforcement; CONSTITUTION I contract test |
| RC-10 | interface-cli § Surface (omit-vs-marker, leaf renders nothing); CONSTITUTION VIII |
| RC-11 | plan ADR-3; tasks T002 (create-if-absent-else-reuse); branching guidance (role-based awareness) |
