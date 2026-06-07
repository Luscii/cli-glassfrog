# Analyze: API Error Extraction

**Feature**: 015-api-error-extraction
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/opaque-failures/api-error-extraction.feature, tasks.md
**Checklist context**: loaded (13/13 pass, 0 fail — after the round-2 triage)
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-06-07 (round 1); **refreshed 2026-06-07 (PR #37 triage — after round-2/3 fixes + the 429→`RateLimited`(5) fold-in)**

---

## Summary

16 checks, **16 pass, 0 fail**. Consistency: 6/6. Completeness: 6/6. Coherence: 4/4.

> **Refresh note (PR #37):** this analysis was re-run after the triage rounds. The two originally-flagged findings are now resolved: **K5** (the default exit-3 path had no 015 scenario) — a `404→3` scenario was added; **H3** (the spec framed 015 narrowly while the slice scoped in the consumer exit-code split) — the spec's 429 non-behavior now states the consumer classifies `429→rate-limit(5)` and `401/403→permission(4)`, closing the framing gap. The artifact set also now reflects the **429→`RateLimited`(5) fold-in** (015 owns both consumer-side splits) and the `DetailSynthesized`/`BodyStatus` fields.

No contradictions and no gaps — the artifacts tell the same story: the `apiclient` capability (`ExtractProblem`/`ProblemError`) decides no exit code; the consumer (`classifyClientError`) maps **401/403→`PermissionError`(4)**, **429→`RateLimited`(5)**, else→`APIError`(3). Single interface file + single feature file — no check scaling.

---

## Consistency: 6/6 passed

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — every named boundary (010 upstream; 017 landed sibling; Unsignalled Plan Limits future; 004 downstream; the Glassfrog API) appears in the plan with the same role.
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — `ExtractProblem` + the consumer wiring realizes every accord clause (consume the non-2xx outcome, extract detail/title/type, degrade gracefully, HTTP-status-authoritative, surface one typed error) with no contradiction.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture — the plan architects none of the excluded capabilities: it does not send requests (010), retry/back off/sleep on 429 (017 owns the *handling*), translate 403 into plan-limit guidance (Unsignalled Plan Limits), gate on `Content-Type`, or decode the error body into the success target. The non-behavior "must not decide the process exit code … the consuming command owns classification" is honored: plan ADR-3 keeps the `apiclient` capability exit-code-free and places the splits in the **consumer** (`internal/cli`). The 429→`RateLimited`(5) *classification* is the consumer's (the spec's amended non-behavior assigns it there, distinct from the 017-owned backoff); 403→`PermissionError`(4) is a permission exit code, distinct from the "not available on your plan" *messaging* the spec excludes. No contradiction.
- **C4** plan § Architecture Decisions ↔ interface-spec § Surface — the interface reflects ADR-1 (`ProblemError` wraps `ResponseError`, with the `DetailSynthesized` + `BodyStatus` fields; pure `ExtractProblem`), ADR-2 (graceful degradation with the `DetailSynthesized` provenance marker, HTTP-status-authoritative with `BodyStatus` as metadata), ADR-3 (consumer-side split — 401/403→`PermissionError`(4) and 429→`RateLimited`(5); `apiclient` decides no exit code), ADR-4 (detail + next-step hint surfaced in `formatClientErrorMessage`, refined once at `reportClientError`). The pinned name `ProblemError`/`ExtractProblem` matches plan + DECISIONS throughout (the round-3 rename swept the working names).
- **C5** plan § Phases/System Architecture ↔ tasks § Scope — T001/T002/T003 build exactly the plan's components (apiclient `ProblemError`+`ExtractProblem`+the two provenance/metadata fields; the consumer wiring incl. both `PermissionError` and `RateLimited`; godog acceptance); no task builds anything the plan doesn't name.
- **C6** interface-spec § Surface ↔ feature Given/When/Then — every scenario step references a defined surface: `ExtractProblem`/the typed error, the `Detail`/`Title`/`Type`/`DetailSynthesized`/`BodyStatus` fields, the status authority, the `detail`-in-message, and the consumer exit-code mappings (`exit with code 4` / `code 5` / `code 3` — the status→exit-code table's three rows). No step uses a surface the interface doesn't define.

## Completeness: 6/6 passed

### Passed (6/6)

- **K1** spec § Driving Scenarios → feature — all 12 spec scenarios (8 driving: 3 happy / 2 error / 3 edge, + 4 validation) have Gherkin equivalents (verified one-to-one via `# Source:` comments). The feature adds 4 architecture-informed scenarios (`Proposed:` — the detail-in-message and the three exit-code mappings 403→4, 429→5, 404→3), correctly marked.
- **K2** spec § Integration Boundaries → interface files — the one specification boundary (the `internal/apiclient` + `internal/cli` package API) has interface-spec.md; the other boundaries (010/017/004 Go packages, the future Unsignalled Plan Limits, the API system actor) are consuming/consumed code — a justified absence consistent with "no command in this slice."
- **K3** plan § Phases → tasks — each of the three linear plan phases has a decomposing task (Phase 1→T001, Phase 2→T002, Phase 3→T003).
- **K4** plan § Components → tasks § Scope — every plan component has an implementing task: `ProblemError`+`ExtractProblem`+`DetailSynthesized`/`BodyStatus` (T001); the consumer wiring — `PermissionError`+`RateLimited` enum/`ExitCode`(4 and 5), the `classifyClientError` 401/403→4 + 429→5 split, the `reportClientError` refine, the `formatClientErrorMessage` detail + next-step hints (all T002); godog acceptance (T003).
- **K5** interface-spec § Surface (status→Outcome→exit-code table) → feature § Given/When/Then — **pass** *(resolved this round)*. The interface's three exit-code outcomes — 401/403→`PermissionError`(4), 429→`RateLimited`(5), else→`APIError`(3) — each have a dedicated feature scenario ("An authorization failure exits with the permission code", "A rate-limited response exits with the rate-limit code", "A non-permission API error exits with the general API code"). The default (3) row, originally uncovered (the round-1 K5 gap), now has the 404→3 scenario.
- **K6** spec § User Scenarios → interface-spec § Surface — each User Scenario maps to interface coverage: US1 (know-why → `ExtractProblem`+`ProblemError{Status,Detail}`), US2 (decide-next → the named fields + raw body via the wrapped `ResponseError` + the exit-code mappings + next-step hints), US3 (usable-on-junk → the graceful-degradation contract with `DetailSynthesized`).

## Coherence: 4/4 passed

### Passed (4/4)

- **H1** Terminology across all artifacts — the domain concepts (`ResponseError`, the typed `ProblemError`, RFC 9457 `type`/`title`/`detail`, the authoritative status, `DetailSynthesized`, `BodyStatus`, `PermissionError`, `RateLimited`) are used consistently. The spec↔code spelling bridge ("typed API error" vs `ProblemError`/`ExtractProblem`) is explicitly aliased in interface-spec.md's Naming note, and the round-3 rename removed the stale `APIError`/`ExtractAPIError` working names from plan + DECISIONS.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; the spec stays behavioral, the plan/interface add concrete shapes, no shared topic is 3x+ lopsided.
- **H3** Scope alignment across spec.md + interface-spec.md + tasks.md — **pass** *(resolved this round)*. The round-1 drift (spec framed 015 as extraction-only while plan/interface/tasks scoped in the consumer exit-code split on landed reads 011–014) is closed: the spec's amended 429 non-behavior now states the consumer classifies `429→rate-limit(5)` and `401/403→permission(4)` (mirroring each other), so a spec-only reader anticipates the consumer-side split. The slice scope and the spec now visibly agree (the cross-spec impact on 011–014 remains recorded in the checklist Governance Notes).
- **H4** Phase coverage plan ↔ tasks — the tasks mirror the plan's three-phase linear structure exactly, with the dependency chain T001→T002→T003 matching Phase 1→2→3.

---

## Checklist Correlation

- **Checklist is now 13/13 pass** (round-2 triage resolved the II next-step and VIII fallback-provenance P1s). Analyze finds no contradiction or gap that re-opens them — the artifacts are internally consistent about the `DetailSynthesized` provenance, the `BodyStatus` metadata, and the next-step hints.
- **Checklist's cross-spec exit-code observation** (401/403→4 and 429→5 change the landed reads 011–014's exit codes from 3) is the only remaining advisory — analyze confirms it is the **planned** split (C3 pass: the interface, the 011/004 forecast, and landed 017's deferral all agree), not a contradiction. Both surface it for a deliberate confirmation, not as a defect.
- **The 429→`rate-limit`(5) ownership** (once flagged as an unresolved 015↔017 contradiction) is resolved: 015 owns the classification (exit 5), 017 owns the retry/backoff — consistent across spec/plan/interface/tasks/DECISIONS.

---

## Governance Notes

- **All matrix relationships were evaluable** — no checks skipped; the full pipeline artifact set (spec, plan, interface, scenarios, tasks) is present.
- **Single interface + single feature file** — no scaling multiplier applied; the 16 base checks ran once each.
