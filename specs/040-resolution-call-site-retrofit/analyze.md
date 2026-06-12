# Analyze: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/duplicated-setting-resolution/resolution-call-site-retrofit.feature, tasks.md
**Checklist context**: checklist.md present (8/8 pass, 0 fail)
**Findings**: 0 (0 P0, 0 P1, 0 P2) — all relationship checks pass
**Generated**: 2026-06-12

---

## Summary

All cross-artifact relationship checks pass. The artifact set tells one consistent story: a surface-stable retrofit of three resolvers onto 039's `internal/resolve`, with one deliberate presence-based flag-semantics change.

| Category | Severity | Evaluations | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 7 | 7 | 0 |
| Completeness | P1 | 7 | 7 | 0 |
| Coherence | P2 | 4 | 4 | 0 |
| **Total** | — | **18** | **18** | **0** |

Two interface files (interface-spec.md, interface-cli.md) and one feature file scale C4/C6/K5 accordingly. One advisory governance note (non-blocking) is recorded below.

---

## Consistency (P0): 7/7 passed

- **C1** spec Integration Boundaries ↔ plan System Architecture — PASS. The boundaries spec names (`internal/resolve`, `internal/rcfile`, 007/`assemble.go`, the cli read commands) are exactly the components plan's architecture describes.
- **C2** spec Behavioral Accord ↔ plan System Architecture — PASS. Compose-and-map-back, presence threading, and validate-the-winner serve every behaviour the accord describes; none is contradicted.
- **C3** spec Non-Behaviors ↔ plan System Architecture — PASS. Plan architects nothing the spec excludes: ADR-1 keeps the public types (not removed), ADR-4 keeps the per-site seam, no STDIN source, token precedence unchanged, default not re-validated.
- **C4** plan Architecture Decisions ↔ interface-spec.md Surface — PASS. Signatures, the provenance→type mapping, validate-at-winner, and the per-site seam reflect ADR-1/2/3/4.
- **C4** plan Architecture Decisions ↔ interface-cli.md Surface — PASS. The presence-based flag rung reflects ADR-2; no new flag/command (consistent with "adds no surface").
- **C5** plan System Architecture ↔ tasks Task Scope — PASS. T001/T002/T003 build exactly the three resolvers + their threading; no task introduces a component plan doesn't name.
- **C6** interface-*.md Surface ↔ feature Given/When/Then — PASS. Every scenario step references a defined surface (`--base-url`/`--output`/`GLASSFROG_*`/`.glassfrogrc`, the typed-error labels, the `SourceNone`/`SourceFile`/default outcomes). The position-independence step ("either side of the subcommand") references the inherited-persistent-flag behaviour interface-cli carries forward from 011/020.

## Completeness (P1): 7/7 passed

- **K1** spec Driving Scenarios ↔ feature — PASS. All 9 driving scenarios (4 happy / 2 error / 3 edge) have a Gherkin equivalent; the 3 validation scenarios are carried with `@validation`.
- **K2** spec Integration Boundaries ↔ interface file presence — PASS. The changed surfaces (code-API → interface-spec; CLI flag behaviour → interface-cli) have files; the "007/assemble consume unchanged" boundary is an explicit no-change (justified absence — no new contract).
- **K3** plan Phases ↔ tasks — PASS. Each of the 3 plan phases maps 1:1 to T001/T002/T003.
- **K4** plan Components ↔ tasks Task Scope — PASS. Token, base-URL, and output resolvers plus the cli plumbing each have an implementing task.
- **K5** interface-spec.md Surface ↔ feature — PASS. The resolver surfaces, the seam contract, and the validation/mapping behaviour are exercised by the composition, presence, and surface-stability scenarios.
- **K5** interface-cli.md Surface ↔ feature — PASS. The empty/whitespace-flag-fails-loud surface and the env-still-falls-through contrast are both covered.
- **K6** spec User Scenarios ↔ interface coverage — PASS. The three user scenarios (Maintainer composition, Practitioner presence, consumer surface-stability) each map to interface content.

## Coherence (P2): 4/4 passed

- **H1** Terminology — PASS. "provenance", "presence" (`Changed()`), "winner", "surface-stable", and the per-domain `Source` enum names are used consistently across all artifacts.
- **H2** Detail symmetry — PASS. spec↔plan and plan↔tasks are proportionate; no shared topic is 3x more detailed on one side.
- **H3** Scope alignment (spec + interface + tasks) — PASS. One feature scope throughout (three settings, presence change, surface-stable). The architecture-informed flag-position scenario tests existing inherited-flag behaviour; it adds a verification angle, not a capability.
- **H4** Phase coverage (plan + tasks) — PASS. Tasks reference exactly plan's three phases, including the "phases 2 & 3 parallel, share `RunE` edits" structure (reflected in the dependency graph and branching guidance).

---

## Checklist Correlation

Checklist (8/8 pass, 0 fail) raised two non-blocking observations (II — inherited `*BaseURLError` remediation text; V — presence-threading breadth). Neither overlaps an analyze finding — analyze found no cross-artifact contradictions or gaps in those areas. No correlated findings.

---

## Governance Notes

- **Advisory (non-blocking) — token-from-file scenario asymmetry**: interface-spec.md's provenance mapping includes the token `KindFile → SourceFile` (with `Path`) row, but no `.feature` scenario exercises a token resolved from `.glassfrogrc` (the token scenarios cover env-yield and nothing-found). This is asymmetric with the base-URL file rung, which has a scenario (H4: "falls through … to the file"). It traces to the spec's own driving scenarios, which omit a token-from-file case, and is covered at the unit level by the carried-forward 005 suite (tasks T001's "existing `internal/auth` suite passes green"). K5 passes at surface granularity (the token resolver is covered); recorded here as an optional sharpening, not a gap that blocks implementation.
- No artifacts were missing; no checks were skipped.
