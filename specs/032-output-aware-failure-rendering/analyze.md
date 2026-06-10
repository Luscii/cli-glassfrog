# Analyze: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, interface-spec.md, features/opaque-failures/output-aware-failure-rendering.feature, tasks.md
**Checklist context**: checklist.md present (15/15 pass, 0 fail) — correlated, not re-evaluated
**Findings**: 21 checks (21 pass, 0 fail) | 0 P0, 0 P1, 0 P2 | 1 coherence note
**Generated**: 2026-06-10

---

## Summary

All 21 cross-artifact checks pass. Consistency: 8/8. Completeness: 9/9. Coherence: 4/4. (Interface checks scaled ×2 for two interface files: interface-cli.md, interface-spec.md.) No contradictions, no gaps, no drift. One non-blocking coherence observation recorded under H3.

---

## Consistency (P0 — contradiction): 8/8 passed

- **C1** spec § Integration Boundaries ↔ plan § System Architecture — the plan's components (`Diagnose`, `OutputFormat`/`MachineFormat`, `ErrorEnvelope`/`RenderError`, the `reportFailure` chokepoint) align with the spec's named boundaries (031, 020, 018, 019, 015, 004). **Pass.**
- **C2** spec § Behavioral Accord ↔ plan § System Architecture — the `reportFailure` routing (structured→stdout, human→stderr, next-step preserved, exit-code pairing) serves every accord behavior; none is contradicted. **Pass.**
- **C3** spec § Non-Behaviors ↔ plan § Architecture — the plan architects nothing the spec excludes: it delegates classification to 031, format selection to 020, the exit code to 004, and keeps usage/invalid-selector rendering out. **Pass.**
- **C4** plan § ADRs ↔ interface-cli.md § Surface — the CLI accord reflects ADR-1/2/3/4 (the envelope on stdout, the `next_step` field, the channel split, body-when-valid). **Pass.**
- **C4** plan § ADRs ↔ interface-spec.md § Surface — the package accord reflects ADR-1/2 (`reportFailure` signature, `ErrorDetail.NextStep`, `errorEnvelopeFor`, `kind`). **Pass.**
- **C5** plan § System Architecture ↔ tasks § Scope — T001 (mapping) and T002 (chokepoint + threading) build exactly the plan's two phases; no task builds anything the plan does not name. **Pass.**
- **C6** interface-cli.md § Surface ↔ feature Given/When/Then — every scenario step references a surface the CLI accord defines (envelope fields, channels, exit codes); no step uses an undefined field or channel. **Pass.**
- **C6** interface-spec.md § Surface ↔ feature — the package symbols (`reportFailure`, `errorEnvelopeFor`, `kind`) are behavioral targets the scenarios exercise indirectly; no Gherkin step contradicts the package contract. **Pass.**

## Completeness (P1 — gap): 9/9 passed

- **K1** spec § Driving Scenarios → feature — all 9 spec driving scenarios (3 happy, 2 error, 4 edge) have Gherkin equivalents, each with a `# Source:` comment; the 3 validation scenarios are present with `@validation @wip`. **Pass.**
- **K2** spec § Integration Boundaries → interface files — 032's external surface is the CLI failure output, covered by interface-cli.md (+ interface-spec.md for the package contract). The other boundaries are internal capability seams (031/020/018/004), not external systems each needing a file. **Pass.**
- **K3** plan § Implementation Strategy → tasks — Phase 1 → T001, Phase 2 → T002; both phases decomposed. **Pass.**
- **K4** plan § components → tasks § Scope — every plan component (`next_step` field, `kind`, `errorEnvelopeFor`, `reportFailure`, call-site threading) has an implementing task. **Pass.**
- **K5** interface-cli.md § Surface → feature — the envelope/channel/exit-code surfaces have scenario coverage (permission-on-stdout, transport-no-body, human-stderr, exit-code-parity, body-omitted-when-invalid). **Pass.**
- **K5** interface-spec.md § Surface → feature/tasks — the package symbols are realized through behavioral scenarios plus T001/T002 acceptance criteria (table-driven `kind`, `errorEnvelopeFor` unit tests). **Pass.**
- **K6** spec § User Scenarios → interface — all 3 user scenarios (parse failure like success; next step preserved; human stays stderr) have interface coverage in interface-cli's channel split, `next_step` field, and human-stderr contract. **Pass.**

## Coherence (P2 — drift): 4/4 passed

- **H1** Terminology — "diagnostic", "envelope", "cause / next step", "kind", "structured / human format", "command-execution failure", "chokepoint" are used consistently across all artifacts; the prose "next step" and the code field `next_step` are an explicit alias (named in interface-spec). **Pass.**
- **H2** Detail symmetry — spec↔plan and plan↔tasks pairs are proportionate; no artifact carries 3x+ more detail on a shared topic than its neighbor. **Pass.**
- **H3** Scope alignment (spec + interface + tasks) — all describe the same capability: render command-execution failures per format, with the `next_step` field, on the correct channel, paired with the unchanged exit code. No capability is silently added or dropped. **Pass** — see coherence note below.
- **H4** Phase coverage (plan + tasks) — tasks reference exactly plan's two phases with matching dependency direction (T002 depends on T001 ↔ Phase 2 depends on Phase 1). No orphan or missing phase. **Pass.**

### Coherence note (non-blocking, H3)

The **partial-walk-with-partial-data** handling (a mid-walk paginated failure keeps its incompleteness note on stderr even under `json`/`yaml`, while the partial `{data:[…]}` document stays on stdout) appears in plan ADR-3, both interface files, the feature file, and tasks T002 — but it is **not** a spec driving scenario. This is intentional, not drift: it is an architecture-level *realization* of the spec's "command-execution failures only" scope combined with 018's one-document-per-channel contract, and the feature file correctly marks it `# Proposed` (architecture-informed). Recorded so a reviewer knows the behavior originates at the plan layer, not the spec. No action required; if the team wants it spec-visible, a one-line spec edge scenario would close the gap.

## Checklist Correlation

checklist.md (15/15 pass, 0 fail) was loaded and correlated, not re-evaluated:
- Checklist **CONSTITUTION VI** (Size-Aware / never silently truncate) and analyze **H3 coherence note** reference the same partial-walk behavior (plan ADR-3) — checklist confirms it satisfies VI; analyze confirms it is a coherent, properly-flagged plan-layer realization. Mutually reinforcing, no conflict.
- Checklist **CONSTITUTION II/III** passes (Action Transparency / Fail Safe) overlap with analyze **C2** (spec behavior ↔ plan architecture) on the same accord sections — both clean.
- No checklist failure overlaps any analyze finding (both artifact sets are clean).

## Governance Notes

- No checks were skipped — the full artifact set (spec, plan, two interface files, one feature file, tasks) was present.
- No `accords/governance/` directory exists; this does not affect analyze (horizontal checks need no done-* accords — that is checklist's vertical domain).
