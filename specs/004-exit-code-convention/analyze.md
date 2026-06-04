# Analyze: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, features/no-runnable-cli/exit-code-convention.feature, tasks.md
**Checklist context**: loaded — 10/10 pass, 0 failures
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-06-03 (re-derived after the crash-diagnostic clarification)

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 6 | 6 | 0 |
| Completeness (P1) | 6 | 6 | 0 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **16** | **16** | **0** |

No contradictions, gaps, or coherence drift.

**Improvement since last run**: previous run 15/16 (1 P2 — H3). The H3 coherence drift is now **resolved**: the clarification carved out the crash diagnostic in the spec (safety-net accord behavior + Non-Behavior carve-out + relaxed Process/shell boundary), so spec, plan ADR-4, and interface-cli.md now state the same text boundary. Current run: 0 findings.

---

## Consistency: 6/6 passed

### Findings

None.

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — plan's parts (main/process boundary, dispatch as `Outcome` producer, `exitcode.go` registry, future API client) align with the spec's named boundaries (002 upstream, 003 sibling, API client downstream, process/shell external).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — the `ExitCode` registry, `RuntimeError`, and panic-recover serve the success / usage / operational / unexpected-failure behaviors without contradicting any.
- **C3** spec § Non-Behaviors ↔ plan — plan architects none of the excluded *capabilities*: no category classification (producers classify), no retry/backoff (deferred to the API-client per X), no suppress-to-zero, no shell-reserved codes. (The one text-emission nuance is a coherence concern, not a capability the plan wrongly architects — see H3.)
- **C4** plan § ADRs ↔ interface-cli § Surface — the accord reflects ADR-1 (pure mapper), ADR-2 (published 0–6 with reserved operational codes), ADR-3 (`RuntimeError`→1), and ADR-4 (panic→1) faithfully.
- **C5** plan § System Architecture ↔ tasks § Scope — T001–T004 map to the registry / category / entrypoint / acceptance parts; no task builds anything the plan omits.
- **C6** interface-cli ↔ no-runnable-cli/exit-code-convention.feature steps — every 004 scenario step references only surfaces the accord defines (codes 0–6, the category→code mapping, the never-zero rule).

## Completeness: 6/6 passed

### Findings

None.

### Passed (6/6)

- **K1** spec § Driving Scenarios → features — all 7 spec driving scenarios (3 happy / 2 error / 2 edge) have Gherkin equivalents in the three 004 Rule blocks.
- **K2** spec § Integration Boundaries → interface — the external process/shell boundary has interface-cli.md; the 002/003 boundaries are internal siblings (cross-referenced, no new interface); the API-client boundary is a forward-looking deferral the spec/plan state explicitly (justified absence).
- **K3** plan § Phases → tasks — the single plan phase's four steps map to T001–T004.
- **K4** plan § Components → tasks — registry→T001, `Outcome`/`RuntimeError`→T002, entrypoint→T003, scenario wiring→T004.
- **K5** interface-cli § Surface → features — codes 0/1/2/4/5 have positive behavioral scenarios; the full 0–6 set is pinned by the "one-to-one" and "exact values" validation scenarios + T001's exact-values test. Codes 3 (general API) and 6 (network) have registry-level rather than behavioral coverage — appropriate, since they have no producer yet (see Governance Notes).
- **K6** spec § User Scenarios → interface — all three user flows (agent distinct-codes / CI non-zero / Maintainer registry) are covered by the accord's Surface, Error Communication, and extension model.

## Coherence: 4/4 passed

### Findings

None.

> **Resolved (H3)**: the prior run flagged a P2 scope-alignment drift — the spec's blanket "render no text" non-behavior did not carve out the panic-diagnostic stderr write that plan ADR-4 and interface-cli define. The clarification (2026-06-03) resolved it: the spec now states the crash diagnostic as a positive safety-net behavior, carves it out of the rendering Non-Behavior, and relaxes the Process/shell boundary's "sole signal" wording. Spec, plan, and interface now state the same boundary.

### Passed (4/4)

- **H1** Terminology — "Outcome category", "exit code", "registry", "RuntimeError", "Fail-Safe default", "usage error" are used consistently across all artifacts.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; no artifact dominates a shared topic by 3x+.
- **H3** Scope alignment — spec, plan, interface, and tasks describe the same scope; the crash-diagnostic stderr write is now stated identically across spec (safety-net accord + Non-Behavior carve-out + Process/shell boundary), plan ADR-4, and interface-cli — no capability is added or dropped silently.
- **H4** Phase coverage — tasks' single phase (4 tasks, linear deps) matches plan's single phase (4 ordered steps); no task references a phase the plan lacks.

---

## Checklist Correlation

No overlapping severity findings — checklist 10/10 pass, analyze 0 findings. The text-emission boundary that the prior H3 drift and the checklist's Action-Transparency advisory both touched is now reconciled in the spec, so the thematic tension is closed (the panic-diagnostic carve-out is explicit in all three artifacts).

## Governance Notes

- **All 16 base checks ran** — full artifact set present (spec, plan, 1 interface, the shared feature file, tasks). No checks skipped for missing artifacts.
- **Codes 3 and 6 have no behavioral scenario** — by design: they have no producer in the skeleton (no API client). Their coverage is registry-level (one-to-one + exact-values validation scenarios, pinned by T001). The future API-client spec should add behavioral scenarios when it produces these categories.
- **Checklist context**: loaded and parsed (10/10 pass).
