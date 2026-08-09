# Analyze: Operating-Surface Self-Containment

**Feature**: 076-operating-surface-self-containment
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, tasks.md, features/unequipped-agent-operators/operating-surface-self-containment.feature
**Checklist context**: loaded — run 2, 19/19 pass (run 1's 2 P0 findings resolved)
**Checks**: 16 (16 pass, 0 fail — 1 P0 found and resolved during this run; see below)
**Generated**: 2026-08-09

---

## Summary

All 16 checks pass at time of writing. Consistency: 6/6. Completeness: 6/6. Coherence: 4/4.

One P0 consistency finding (C4) was detected during this run and resolved before the file was written. It is recorded below rather than omitted — the artifact set that existed when analyze started did contain the contradiction.

---

## Consistency: 6/6 passed

### Finding detected and resolved during this run

**P0** | C4: plan.md § Cross-cutting Concerns ↔ interface-spec.md § 2 Guard surface
> plan.md (as revised after the checklist findings) stated that the surface-content scenario reads the real swept surface and belongs to Phase 1, and tasks.md T002 bound that scenario in `surface_self_containment_bdd_test.go`. interface-spec.md §2 described that same suite as driving production functions "against `t.TempDir()` fixture surfaces only". A Builder reading both got contradictory instructions about what the suite may read.
> **Origin**: introduced by the checklist-remediation edit, which updated plan.md and tasks.md but not the interface accord — the same propagation gap the repo's process memory records as a recurring cause of review rounds.
> **Resolution applied**: interface-spec.md §2 now defines two step families — *detection steps* (fixtures only, so a seeded violation never enters the checkout) and *surface-content steps* (read-only over the real surface) — with the surviving invariant stated explicitly: the suite never writes to `plugin/` and never runs the walker against it, which remains the guard test's job.

### Passed (6/6)

- **C1** spec.md § Integration Boundaries ↔ plan.md § System Architecture: all four spec boundaries (the surface, the merge-gating run, the existing per-path guards, CONSTITUTION.md) map to the plan's four parts.
- **C2** spec.md § Behavioral Accord ↔ plan.md § System Architecture: every accord group (the rule, detection, the conforming surface, re-derived expectations, the standing accord) has an architectural home; none is contradicted.
- **C3** spec.md § Non-Behaviors ↔ plan.md § System Architecture: the plan architects none of the six excluded capabilities — it states wording-only change, leaves merged specs untouched (ADR-3), keeps repository→surface pointers legal, permits operating-world references (ADR-2 known-safe tokens), derives the file set by walking, and forbids satisfying an assertion by deletion.
- **C4** plan.md § Architecture Decisions ↔ interface-spec.md § Surface: after the resolution above, the interface reflects ADR-1 (guard as `internal/build` test), ADR-2 (two-family lexicon), ADR-3 (name-pins), and ADR-4 (Principle XIII + version marker).
- **C5** plan.md § System Architecture ↔ tasks.md § Task Scope: every task builds something the plan names; no task introduces an unmentioned component.
- **C6** interface-spec.md § Surface ↔ feature file § Given/When/Then: every scenario step references a surface the accord defines — the deny-lexicon families, the violation-report grammar, the condition table's dangling-path and empty-surface rows, and the known-safe token classes.

---

## Completeness: 6/6 passed

- **K1** spec.md § Driving Scenarios → feature file: all 7 driving scenarios have Gherkin equivalents, and all 5 validation scenarios are carried with `@validation`. Verified by title comparison — every `# Source:` title matches a spec scenario title verbatim (12 of 12, plus the file-level source comment and one `Proposed:` architecture-informed scenario).
- **K2** spec.md § Integration Boundaries → interface files: the single specification touchpoint covers all four boundaries (§2 the guard/verification run, §3 the registry artifacts, §4 CONSTITUTION.md, §1 the surface itself); the absence of sibling interface files is stated explicitly in Consistency Notes.
- **K3** plan.md § Implementation Strategy → tasks.md: Phase 1 → T001–T003, Phase 2 → T004–T005. Both phases decomposed.
- **K4** plan.md § System Architecture → tasks.md § Scope: each of the plan's four parts has implementing tasks — swept surface (T001–T003), re-derived assertions (T002), guard (T004), constitution (T005).
- **K5** interface-spec.md § Surface → feature file: each of the four surface groups has scenario coverage, including §3's registry-header contract (covered by the "lightweight and workflow-oriented" validation scenario) and §4's principle contract (covered by the "house form" validation scenario).
- **K6** spec.md § User Scenarios → interface-spec.md: all three user scenarios have interface coverage — operator reading the surface (§1 permitted classes + Interactions), the merge gate failing on residue (§2 + Error Communication), and the recorded principle (§4).

---

## Coherence: 4/4 passed

- **H1** Terminology across all artifacts: the load-bearing concepts hold their names — "operating surface", "development repository", "deny lexicon", "Family A/B", "in-plugin name", "live tripwire". Where a concept carries two names, the alias is introduced explicitly (plan § Guard Design ties "the live pass over the real `plugin/`" to "the drift tripwire"; plan § System Architecture names "the self-containment guard" for what the spec calls the detection check).
- **H2** Detail symmetry: spec → plan → interface → tasks scale proportionately; no artifact carries 3x the detail of its neighbour on a shared topic. The lexicon is the deepest shared topic and lives at contract depth in the interface, summarized in the plan ADR — the expected gradient.
- **H3** Scope alignment across spec.md, interface-spec.md, tasks.md: the same capability set appears in all three — sweep, detection, re-derived assertions, constitutional principle. No artifact introduces or drops a capability.
- **H4** Phase coverage plan.md ↔ tasks.md: phase names, ordering, and dependency structure match, including the refined dependency ("Phase 2 depends on Phase 1 only for the live tripwire") which both artifacts now state the same way.

---

## Checklist Correlation

- Checklist run 1 flagged **2 P0** in tasks.md § Phase 2 (CONSTITUTION IV and VII — acceptance scenarios bound after the code, and a test-only increment). Analyze's C4 finding is in the artifact set produced by remediating those findings: the same edit that resolved them introduced the plan↔interface contradiction. The vertical and horizontal findings share a root cause — a decomposition change propagated to two of the three artifacts that state it.
- No other overlap: checklist run 2's passing checks and analyze's passing checks reference distinct sections.

---

## Governance Notes

- **All artifact types present** — no checks skipped for missing artifacts. Interface checks ran against 1 file (`interface-spec.md`), scenario checks against 1 feature file; per the matrix's scaling rules, that yields 16 evaluations, matching the base check count.
- **Checklist context**: loaded and parsed — run 2 header reports 19/19 pass, with run 1's two P0 findings recorded as resolved.
- **Finding recorded rather than erased**: per the Guardian's append-only principle, the C4 finding stays in this record with its resolution, so a reviewer sees that the artifact set was contradictory when analyze began, not only that it is consistent now.
