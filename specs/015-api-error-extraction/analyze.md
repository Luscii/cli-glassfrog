# Analyze: API Error Extraction

**Feature**: 015-api-error-extraction
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, features/opaque-failures/api-error-extraction.feature, tasks.md
**Checklist context**: loaded (11/13 pass, 2 P1 failures)
**Checks**: 16 (14 pass, 2 fail — 1 P1, 1 P2)
**Generated**: 2026-06-07

---

## Summary

16 checks, **14 pass, 2 fail (1 P1, 1 P2, no P0)**. Consistency: 6/6. Completeness: 5/6. Coherence: 3/4.

No contradictions (consistency clean) — the artifacts tell the same story, including the carefully-resolved spec-vs-forecast tension (the `apiclient` capability decides no exit code; the consumer maps). Two findings:
1. **K5 (P1, completeness)** — the interface's status→exit-code table has two outcomes (401/403 → permission(4); everything else → API error(3)), but only the **(4) row** has a dedicated 015 scenario; no 015 scenario asserts the default **(3)** path. *Mitigated:* the (3) baseline is unchanged from 011 and covered by 011's identity-read suite.
2. **H3 (P2, coherence)** — the spec frames 015 narrowly (the extraction capability, with a non-behavior pushing exit-code decisions to the consumer), while plan/interface/tasks scope in the **consumer-side exit-code split (401/403→4) and message change across the landed reads 011–014**. Deliberate (plan ADR-3 resolves the tension) and not silent downstream — but a spec-only reader wouldn't anticipate that 015's PRs edit shipped reads' exit codes.

Single interface file + single feature file — no check scaling.

---

## Consistency: 6/6 passed

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — every named boundary (010 upstream; 017 downstream sibling; Unsignalled Plan Limits future; 004 downstream; the Glassfrog API) appears in the plan with the same role.
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — `ExtractProblem` + the consumer wiring realizes every accord clause (consume the non-2xx outcome, extract detail/title/type, degrade gracefully, HTTP-status-authoritative, surface one typed error) with no contradiction.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture — the plan architects none of the excluded capabilities: it does not send requests (010), back off/retry on 429 (017), translate 403 into plan-limit guidance (Unsignalled Plan Limits), gate on `Content-Type`, or decode the error body into the success target. Critically, the non-behavior "must not decide the process exit code … the consuming command owns classification" is honored: plan ADR-3 keeps the `apiclient` capability exit-code-free and places the split in the **consumer** (`internal/cli`), exactly where the spec assigns it. The 403→`PermissionError`(4) mapping is a generic permission exit code, distinct from the "not available on your plan" *messaging* the spec excludes. No contradiction.
- **C4** plan § Architecture Decisions ↔ interface-spec § Surface — the interface reflects ADR-1 (`ProblemError` wraps `ResponseError`; pure `ExtractProblem`), ADR-2 (graceful degradation, HTTP-status-authoritative), ADR-3 (consumer-side split, `PermissionError`, `apiclient` decides no exit code), ADR-4 (detail surfaced in `formatClientErrorMessage`, refined once at `reportClientError`). The interface's decision to pin the name `ProblemError`/`ExtractProblem` realizes the plan's "interface pins it (may be `ProblemError`)" — consistent, not a conflict.
- **C5** plan § Phases/System Architecture ↔ tasks § Scope — T001/T002/T003 build exactly the plan's components (apiclient `ProblemError`+`ExtractProblem`; the consumer wiring; godog acceptance); no task builds anything the plan doesn't name.
- **C6** interface-spec § Surface ↔ feature Given/When/Then — every scenario step references a defined surface: `ExtractProblem`/the typed error (extraction scenarios), the `Detail`/`Title`/`Type` fields, the status authority, the `detail`-in-message (formatClientErrorMessage), and the `exit with code 4` (the status→exit-code table's 401/403 row). No step uses a surface the interface doesn't define.

## Completeness: 5/6 passed

### Failed (1 — P1)

- **P1** | **K5** interface-spec § Surface (status→Outcome→exit-code table) → feature § Given/When/Then — **fail**. The interface defines two exit-code outcomes for a non-2xx — **401/403 → `PermissionError`(4)** and **everything else (incl. 429) → `APIError`(3)**. The feature covers the (4) row ("An authorization failure exits with the permission code") but has **no scenario asserting the default (3) exit code** for a non-401/403 non-2xx (e.g. a 404 or 500 exiting 3). The "404 surfaces the API's own detail" and "429 extracted without backoff" scenarios assert the *detail/headers*, not the exit code. **Mitigation/context:** the →(3) mapping is **unchanged from 011** (which shipped `*ResponseError`→`APIError`(3)) and is covered by 011's identity-read suite — so this is a coverage gap *within 015's own feature file*, not an untested behavior project-wide. **Recommendation (for the Verifier, not applied here):** add one scenario asserting a non-401/403 non-2xx exits 3, or note in the feature that the (3) baseline is covered by 011.

### Passed (5/6)

- **K1** spec § Driving Scenarios → feature — all 12 spec scenarios (8 driving: 3 happy / 2 error / 3 edge, + 4 validation) have Gherkin equivalents (verified one-to-one via `# Source:` comments). The feature adds 2 architecture-informed scenarios (`Proposed:`), correctly marked.
- **K2** spec § Integration Boundaries → interface files — the one specification boundary (the `internal/apiclient` + `internal/cli` package API) has interface-spec.md; the other boundaries (010/017/004 Go packages, the future Unsignalled Plan Limits, the API system actor) are consuming/consumed code, not external surfaces needing their own interface file — a justified absence consistent with "no command in this slice."
- **K3** plan § Phases → tasks — each of the three linear plan phases has a decomposing task (Phase 1→T001, Phase 2→T002, Phase 3→T003).
- **K4** plan § Components → tasks § Scope — every plan component has an implementing task: `ProblemError`+`ExtractProblem` (T001); the consumer wiring — `PermissionError`+`ExitCode` case, the `classifyClientError` split, the `reportClientError` refine, the `formatClientErrorMessage` detail-surfacing (all T002); godog acceptance (T003).
- **K6** spec § User Scenarios → interface-spec § Surface — each User Scenario maps to interface coverage: US1 (know-why → `ExtractProblem`+`ProblemError{Status,Detail}`), US2 (decide-next → the named `Type`/`Title`/`Detail` fields + raw body via the wrapped `ResponseError` + the exit-code mapping), US3 (usable-on-junk → the graceful-degradation contract).

## Coherence: 3/4 passed

### Failed (1 — P2)

- **P2** | **H3** Scope alignment across spec.md + interface-spec.md + tasks.md — **fail (drift)**. The spec scopes 015 as the **extraction capability** (the behavioral accord is entirely about interpreting a non-2xx into a typed error; a non-behavior states 015 "must not decide the process exit code"). The plan, interface, and tasks scope the slice **wider** — T002 lands the consumer-side **401/403→`PermissionError`(4)** exit-code split and the detail-surfacing message **on the already-landed reads 011–014**. This is **deliberate and explicit downstream** (plan ADR-3 reconciles the spec's "stay narrow" with 004/011's forecast that 015 fills the reserved permission code; the consumer — not the `apiclient` capability — does the mapping, which the spec's own non-behavior assigns to "the consuming command"). So it is *coherent on close reading*, not a contradiction (see C3, pass). The drift is one of **framing**: a reader who reads only spec.md would not anticipate that 015's PRs touch shipped reads' exit codes. **Recommendation (for the Definer/Shaper, not applied here):** optionally add a one-line note to the spec's System Overview or Non-Behaviors that the *consuming command's* exit-code split (filling code 4) lands with this capability, so the spec's scope and the slice's scope visibly agree.

### Passed (3/4)

- **H1** Terminology across all artifacts — the domain concepts (the non-2xx outcome / `ResponseError`, the typed `ProblemError`, RFC 9457 `type`/`title`/`detail`, the authoritative status, the fallback, `PermissionError`) are used consistently. The one spelling bridge — the spec says "typed API error" while the code symbols are `ProblemError`/`ExtractProblem` — is **explicitly aliased** in interface-spec.md's Naming note ("the capability/spec name stays 'API Error Extraction'; only the symbols change"), so it is a documented alias, not an unlabeled drift.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; the spec stays behavioral, the plan/interface add concrete shapes, and no shared topic is 3x+ lopsided.
- **H4** Phase coverage plan ↔ tasks — the tasks mirror the plan's three-phase linear structure exactly, with the dependency chain T001→T002→T003 matching Phase 1→2→3.

---

## Checklist Correlation

- **Checklist's cross-spec exit-code observation correlates with analyze H3.** Checklist recorded (Governance Notes) that 401/403 move 3→4 for the landed reads 011–014; analyze H3 finds the same change as a spec-vs-slice **scope-framing** drift. Shared root: the consumer-side exit-code split is scoped into 015 but not surfaced in the spec. Both are advisory; resolving H3's recommendation (a spec note) also addresses the checklist observation's "confirm this is the planned split, not a contradiction" — analyze confirms it **is** planned (C3 pass: the interface and the 011/004 forecast agree).
- **Checklist's two P1 findings (II next-step, VIII fallback-provenance) do not correlate to an analyze finding** — both are *vertical* (an artifact vs a constitution principle), with no cross-artifact contradiction or gap. The artifacts are internally consistent *about* the fallback detail and the message; whether they fully satisfy II/VIII is checklist's domain.
- **Checklist's 429-backoff sequencing observation** is a cross-spec/roadmap concern outside analyze's single-spec horizontal scope (015's artifacts agree: they defer backoff to 017 and carry the headers). Carried to risk.

---

## Governance Notes

- **All matrix relationships were evaluable** — no checks skipped; the full pipeline artifact set (spec, plan, interface, scenarios, tasks) is present.
- **Single interface + single feature file** — no scaling multiplier applied; the 16 base checks ran once each.
