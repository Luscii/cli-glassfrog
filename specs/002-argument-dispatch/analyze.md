# Analyze: Argument Dispatch

**Feature**: 002-argument-dispatch
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/no-runnable-cli/argument-dispatch.feature, tasks.md
**Checklist context**: loaded — 9/9 pass, 0 failures
**Checks**: 16 (16 pass, 0 fail)
**Generated**: 2026-06-03 (H3 updated after the RuntimeError-deferral decision)

---

## Summary

| Category | Checks | Pass | Fail |
|---|---|---|---|
| Consistency (P0) | 6 | 6 | 0 |
| Completeness (P1) | 6 | 6 | 0 |
| Coherence (P2) | 4 | 4 | 0 |
| **Total** | **16** | **16** | **0** |

No contradictions, gaps, or coherence drift. The one coherence drift originally found (H3) was resolved by the decision to defer the `RuntimeError` category — see the Coherence section.

---

## Consistency: 6/6 passed

### Findings

None.

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — plan's components align with the spec's named boundaries (001 upstream, Help & Version, Exit-Code Convention, no external systems).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — cobra resolution + the classification layer serve the resolution/unknown/invalid behaviors without contradiction.
- **C3** spec § Non-Behaviors ↔ plan — plan architects none of the excluded concerns (no help rendering, no exit codes, no registration, no prefix matching, no command work); "What This Plan Does Not Cover" mirrors them.
- **C4** plan § ADRs ↔ interface — interface-cli reflects cobra resolution + exact match (ADR-1); interface-spec reflects the `Outcome` category (ADR-2).
- **C5** plan § System Architecture ↔ tasks § Scope — T001–T003 map to plan components/phases; no task builds anything the plan omits.
- **C6** interface ↔ no-runnable-cli/argument-dispatch.feature steps — 002 scenario steps reference only surfaces the dispatch accords define (resolution, outcome category, usage error).

## Completeness: 6/6 passed

### Findings

None.

### Passed (6/6)

- **K1** spec § Driving Scenarios → features — all 8 spec driving scenarios (3 happy / 3 error / 2 edge) have Gherkin equivalents in the 002 Rule blocks.
- **K2** spec § Integration Boundaries → interface — "no external systems" needs no interface; the sibling boundaries (Help & Version, Exit-Code Convention) are future specs, cross-referenced in the accords (justified absence).
- **K3** plan § Phases → tasks — Phase 1→T001, Phase 2→T002, Phase 3→T003.
- **K4** plan § Components → tasks — `Run` entry + `Outcome` (T001), classification (T002).
- **K5** interface § Surface → features — resolution, unknown-command, unexpected-flag, bare-group, and outcome-category surfaces all have scenario coverage.
- **K6** spec § User Scenarios → interface — all three user flows (route / recover-from-typo / discover-subcommands) are covered by the dispatch contract.

## Coherence: 4/4 passed

### Findings

None.

> **Resolved (H3)**: this analysis originally flagged a P2 spec↔plan asymmetry — the plan and interface named a third outcome category, `RuntimeError`, that spec.md did not. The developer chose to **defer `RuntimeError`** to Exit-Code Convention (004); the plan/interface/tasks now use a two-value category (`Success`/`UsageError`) matching what the spec names, and the architecture-informed runtime-error scenario was removed from the feature file. The asymmetry is gone.

### Passed (4/4)

- **H1** Terminology — "dispatch", "outcome category", "usage error", "command set" are used consistently across artifacts.
- **H2** Detail symmetry — spec↔plan and plan↔tasks are proportionate; no artifact dominates a shared topic.
- **H3** Scope alignment — spec, plan, interface, and tasks now describe the same two-category outcome model; no capability is added or dropped silently (runtime-failure categorization is explicitly deferred, not silently introduced).
- **H4** Phase coverage — tasks' three phases match plan's three phases and their linear dependencies.

---

## Checklist Correlation

No overlapping **severity** findings — checklist had 0 failures. The H3 drift (RuntimeError not named in spec) that originally surfaced here — adjacent to the checklist advisory "outcome category has no consumer yet (004)" — has since been resolved by deferring `RuntimeError` to 004, which simplifies the classification to the two values the spec names.

## Governance Notes

- **All 16 base checks ran** — full artifact set present (spec, plan, 2 interface, the shared feature file, tasks). No checks skipped for missing artifacts.
- **Interface/scenario checks scaled** across both interface files and the 002 scenarios in the shared feature file.
- **Checklist context**: loaded and parsed (9/9 pass).
