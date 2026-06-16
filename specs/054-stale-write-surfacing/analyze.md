# Analyze: Stale-Write Surfacing

**Feature**: 054-stale-write-surfacing
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, features/clobbered-changes/stale-write-surfacing.feature, tasks.md
**Checklist context**: checklist.md present (12/12 pass, 0 P0/P1/P2)
**Checks**: 16 (16 pass, 0 findings)
**Generated**: 2026-06-14

---

## Summary

All 16 cross-artifact checks pass. Consistency: 6/6 · Completeness: 6/6 · Coherence: 4/4.

P0: 0 · P1: 0 · P2: 0.

The artifact set tells one story end to end: a `412 Precondition Failed` is branched out of the generic API-error bucket into a distinct `StaleWrite` category mapped to a new exit code `7`, with a `412`-specific cause and a re-read/retry next step, classified by status alone — additively, leaving every other status untouched. Spec → plan → interface → scenarios → tasks agree on scope, terminology, and the three classification sites.

---

## Consistency (P0 — contradiction): 6/6 passed

- **C1 — spec Integration Boundaries ↔ plan System Architecture**: PASS. Spec names 053 (producer of the `412`), 015 (types it), 031 (classifies), 004 (registry), 032 (renders); the plan's System Architecture describes the same producer-classifies/registry-maps chain across the same siblings. Compatible.
- **C2 — spec Behavioral Accord ↔ plan System Architecture**: PASS. The accord's four requirements (distinct category, distinct exit code, `412`-cause/next-step, status-driven + additive) are each realized by the plan's `StaleWrite` Outcome, `codeStaleWrite = 7`, the `categoryForStatus`/`nextStepForStatus`/cause arms, and the status-only classification. No behavior is contradicted.
- **C3 — spec Non-Behaviors ↔ plan System Architecture**: PASS. The plan architects nothing the non-behaviors exclude — it does not re-read/retry (recovery left to the caller), does not render (032 owns that), does not emit the code itself (004's registry), does not change other statuses, and does not renumber. The plan's "What This Plan Does Not Cover" mirrors the spec's non-behaviors.
- **C4 — plan Architecture Decisions ↔ interface-cli Surface**: PASS. interface-cli's registry row (`7` stale-write) reflects ADR-1; its Error Communication table (cause provenance + re-read/retry next step) reflects ADR-2. No technology or pattern choice in the interface contradicts the plan.
- **C5 — plan System Architecture ↔ tasks Task Scope**: PASS. T001's scope is exactly the three sites the plan names — `dispatch.go` (Outcome), `exitcode.go` (constant + `ExitCode` case), `diagnostic.go` (classification + cause). No task builds anything the plan does not describe.
- **C6 — interface-cli Surface ↔ feature Given/When/Then**: PASS. Every value the scenarios assert — exit code `7`, the generic code `3`, status `412`, and the statuses `401/403/404/429/500` held unchanged — is defined in interface-cli's registry table and Error Communication section. No step references a surface the interface does not define.

---

## Completeness (P1 — gap): 6/6 passed

- **K1 — spec Driving Scenarios → feature**: PASS. All 7 driving scenarios have a Gherkin equivalent (3 happy, 2 error, 2 edge), plus the 3 Validation Scenarios as `@validation @wip`. Each `# Source:` comment maps to a spec scenario title verbatim.
- **K2 — spec Integration Boundaries → interface file presence**: PASS. The one external (operator-facing) touchpoint — the CLI exit code + stderr diagnostic — has interface-cli.md. The remaining boundaries (053/015/031/004/032) are in-process seams, not external surfaces, and the plan + interface explicitly note no other touchpoint type exists (no cross-package code-API surface).
- **K3 — plan Phases → tasks**: PASS. The plan's single phase (Stale-Write Classification) has a corresponding tasks Phase 1.
- **K4 — plan Components → tasks Scope**: PASS. Each plan component (the three classification/registry sites) is covered by T001's scope and acceptance criteria.
- **K5 — interface-cli Surface → feature coverage**: PASS. Both interface surfaces — the exit code `7` and the `412` diagnostic (cause present / synthesized, and the re-read/retry next step) — are exercised by scenarios.
- **K6 — spec User Scenarios → interface coverage**: PASS. The three user scenarios (agent branches on `$?`; practitioner understands *why*; Maintainer's shared-pipeline integrity) each have interface coverage — the `$? == 7` branching example, the cause/next-step diagnostic, and the registry extension row.

---

## Coherence (P2 — drift): 4/4 passed

- **H1 — Terminology**: PASS. The key concepts — *stale-write / precondition failed*, *exit code 7*, *412*, *category/Outcome*, *re-read and retry* — are used consistently. The category appears as the prose name "stale-write" (spec, interface) and the symbol `StaleWrite` (plan, tasks); the spec's `[ASSUMED]` note explicitly aliases the two (the identifier is a planning detail), so this is an explicit alias, not drift.
- **H2 — Detail symmetry**: PASS. spec↔plan and plan↔tasks are proportionate; no artifact carries 3x+ more detail on a shared topic. The `412`-cause synthesis detail is present and consistent across spec accord, plan ADR-2, interface, and the T001 acceptance criteria.
- **H3 — Scope alignment (spec + interface + tasks)**: PASS. All three describe the same capability — distinct `412` surfacing via a new code `7` plus a `412`-specific cause/next-step, additive to the existing pipeline. No artifact adds or drops a capability the others omit.
- **H4 — Phase coverage (plan ↔ tasks)**: PASS. The plan's single phase maps 1:1 to tasks Phase 1; tasks reference no phase absent from the plan, and the plan's one phase has corresponding tasks.

---

## Checklist Correlation

checklist.md was loaded (12/12 constitution checks pass, 0 failures). No analyze finding overlaps a checklist finding because neither produced failures. One checklist *observation* (C-VIII No Fabricated Data — the synthesized `412` cause must state only what the status defines, not assert who/what changed the resource) sits in the same region as the consistent C2/H3 scope claims; analyze confirms the spec, plan ADR-2, and interface all describe the same no-fabrication boundary, so there is no horizontal contradiction to add — the observation remains an implementation-time watch item, not a cross-artifact gap.

---

## Governance Notes

- No checks were skipped — all five artifact types (spec, plan, interface, scenarios, tasks) plus checklist.md are present, so the full 16-check matrix ran at 1 interface file × 1 feature file scale.
