# Analyze: Output Format Selection

**Feature**: 020-output-format-selection
**Artifacts analyzed**: spec.md, plan.md, interface-spec.md, interface-cli.md, features/unconsumable-output/output-format-selection.feature, tasks.md
**Checklist context**: checklist.md present (12/12 pass, 0 findings)
**Findings**: 0 P0, 0 P1, 0 P2 (2 cross-spec / governance notes — advisory, do not block 020)
**Generated**: 2026-06-08

---

## Summary

| Severity | Category | Count |
|---|---|---|
| P0 (contradiction) | Consistency | 0 |
| P1 (gap) | Completeness | 0 |
| P2 (drift) | Coherence | 0 |
| **Total** | | **0** |

Consistency: 6/6 pass. Completeness: 6/6 pass. Coherence: 4/4 pass. (Interface checks ran across both `interface-spec.md` and `interface-cli.md`.)

---

## Consistency (P0) — 6/6 pass

- **C1** spec Integration Boundaries ↔ plan System Architecture: the plan's components (`internal/output` selection vocabulary + resolver, `internal/render` consumed, the `internal/cli` dispatch, `internal/rcfile`, 004 exit codes, 032 deferred) align with the spec's named boundaries (018/019/005/008/004/032/reads/inputs). Pass.
- **C2** spec Behavioral Accord ↔ plan System Architecture: the plan serves every accord group — resolution (chain), valid formats (case-insensitive parse), dispatch (route to renderer), invalid selector (fail-fast) — with no behavior contradicted. Pass.
- **C3** spec Non-Behaviors ↔ plan System Architecture: the plan architects nothing the spec excludes — it implements no encoder/template (routes to existing renderers), writes no config (reads only), owns no rcfile mechanics (reuses `internal/rcfile`), renders no command failures (defers to 032), and changes no fetched fields. Pass.
- **C4** plan ADRs ↔ interface-cli.md Surface: the persistent `--output`/`-o` flag, the four values, and the precedence chain reflect plan ADR-1/ADR-2. Pass.
- **C4** plan ADRs ↔ interface-spec.md Surface: `OutputFormat`/`ParseFormat`/`IsStructured`/`MachineFormat`, `ResolveFormat`/`FormatError`, and the `cli` dispatch reflect plan ADR-1/ADR-3/ADR-4. Pass.
- **C5** plan System Architecture ↔ tasks Task Scope: every task builds a plan-named component (vocabulary T001, resolver T002, flag T003, classifier arm T004, dispatch T005, read routing T006); no task builds anything the plan omits. Pass.
- **C6** interface ↔ feature steps: every scenario step references a defined surface — `--output` values, `GLASSFROG_OUTPUT`, `.glassfrogrc output`, exit code 2 (interface-cli.md); resolution + dispatch routing (interface-spec.md). No step uses an undefined flag or symbol. Pass.

---

## Completeness (P1) — 6/6 pass

- **K1** spec Driving Scenarios ↔ feature: all 8 driving scenarios (3 happy, 2 error, 3 edge) and all 3 validation scenarios have Gherkin equivalents, each carrying a `# Source:` comment. Pass.
- **K2** spec Integration Boundaries ↔ interface files: 020's own surfaces have interface files — the CLI boundary → interface-cli.md, the package/specification boundary → interface-spec.md. The remaining boundaries (018/019/005/008/004/032) are dependencies the spec marks consumed-unchanged or downstream — justified absences, not 020's surfaces. Pass.
- **K3** plan Phases ↔ tasks: Phase 1 → T001/T002; Phase 2 → T003–T006. Both phases decomposed. Pass.
- **K4** plan Components ↔ tasks Scope: every component has implementing tasks (vocabulary, resolver, flag, classifier arm, dispatch, read routing). Pass.
- **K5** interface surfaces ↔ feature coverage: the flag/formats/precedence/errors (interface-cli) and the resolver/dispatch (interface-spec) are each exercised — json selection, compact reachability, default full, invalid-selector at flag and env, precedence ordering, and per-renderer routing. Pass.
- **K6** spec User Scenarios ↔ interface coverage: all three user scenarios (machine format per invocation; set-once via env/config; reach compact) map to interface surfaces (flag + precedence + the four formats; resolver chain). Pass.

---

## Coherence (P2) — 4/4 pass

- **H1** Terminology: the load-bearing concepts — the four format tokens (`full`/`compact`/`json`/`yaml`), `--output`/`-o`, `GLASSFROG_OUTPUT`, the `.glassfrogrc output` key, `OutputFormat`/`ResolveFormat`/`FormatError`, and the "selection + success here; failures → 032" split — are named consistently across spec, plan, both interface files, scenarios, and tasks. Pass.
- **H2** Detail symmetry: spec↔plan and plan↔tasks are proportionate; no shared topic is 3×+ deeper in one artifact than its neighbour. Pass.
- **H3** Scope alignment (spec + interface + tasks): all describe the same capability set — selection + success dispatch + invalid-selector — and all consistently exclude command-failure rendering (→ 032). Nothing is added or dropped silently. Pass.
- **H4** Phase coverage (plan + tasks): tasks reference exactly Phase 1 and Phase 2; no phantom phase, and each plan phase has tasks. Pass.

---

## Checklist Correlation

- Checklist (12/12 pass) recorded two forward notes; analyze corroborates and locates both:
  - **Deferred structured-error path** — checklist noted (under II) that command failures stay plain-text on stderr under `json`/`yaml` until 032. Within 020's artifact set this is internally **coherent** (H3 pass — spec/plan/interface/scenarios all exclude failure rendering uniformly). The only tension is cross-spec; see Cross-Spec Note 1 below.
  - **Size-boundary signal under structured formats** — checklist noted (under VI) that 020's artifacts don't explicitly state how the 016 incompleteness signal appears under `json`/`yaml`. See Cross-Spec / Observation Note 2.

---

## Cross-Spec / Governance Notes

> Analyze evaluates a single spec's artifact set; the following are outside the single-spec matrix but surfaced because they are actionable. Neither blocks 020's implementation — 020's own artifacts are internally consistent and complete.

1. **018's wording attributes the error→envelope path to 020, but 020 defers it to 032 (advisory).** 018's `spec.md` Integration Boundaries says *"Output Format Selection (020) … routes the command's output — success **and error** — through these encoders,"* and 018's `interface-spec.md` refers to *"the (020-owned) typed-error→envelope mapping."* Per the 020 defining conversation (Decision B), 020 owns selection + success dispatch + the invalid-selector usage error only; command-failure rendering moved to **032 Output-Aware Failure Rendering**. 020's artifacts state this consistently. The discrepancy is in **018's** published wording, now stale relative to the agreed split — and 018's "every output is structured under a structured format" guarantee is therefore not fully met during the 020→032 window. *Recommended action* (on 018, not 020): update 018's Integration Boundaries + interface-spec wording to attribute the error path to 032, or add an interim-gap note. Track alongside the deferred-decision sweep discipline; no code change to 020.

2. **Size-boundary signal under `json`/`yaml` is inherited, not restated (observation).** The 016/VI partial-result incompleteness signal rides the stderr diagnostic channel and is orthogonal to the selected format (interface-cli.md says so); 020 introduces no truncation. 020's spec is silent on it because it is upstream behavior, not 020's to define. No gap in 020; noted so the Builder of the list-read routing (T006) preserves the existing stderr note under all formats.

---

## Governance Notes

- **No `accords/governance/done-*.md` accords exist** — analyze ran its full cross-artifact matrix (which does not depend on done-* accords); the gap affects checklist's done-criteria checks only (noted there).
- **Guardian agent definition not found** at the expected path — ran the process without the character layer (reduced style consistency, not a blocked check).
- All 16 base checks were evaluable (no coherence check degraded to non-binary); interface checks ran across both interface files.
