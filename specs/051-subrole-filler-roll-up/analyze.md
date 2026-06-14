# Analyze: Subrole Filler Roll-up

**Feature**: 051-subrole-filler-roll-up
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, features/who-to-contact-for-a-role/subrole-filler-roll-up.feature, tasks.md
**Checklist context**: loaded — 14/14 pass, 0 failures
**Checks**: 16 (16 pass, 0 fail — original P2 finding H1 resolved in this PR; see note)
**Generated**: 2026-06-14 (updated after PR #120 triage)

---

## Summary

All 16 checks pass. Consistency: 6/6. Completeness: 6/6. Coherence: 4/4.

> **Update (PR #120 triage):** the original run found one P2 coherence finding (H1 — command-spelling drift: spec used `actors subroles`, downstream artifacts used `subrole-actors`). It was resolved during triage by standardizing spec.md on the pinned `subrole-actors` spelling and aligning the Assumptions/Ambiguity sections, so H1 now passes. The original finding text is retained below for traceability.

(1 interface file + 1 feature file — no check scaling beyond the 16 base types.)

---

## Consistency: 6/6 passed

### Findings

None.

### Passed (6/6)

- **C1** spec § Integration Boundaries ↔ plan § System Architecture: the spec's named boundary (`GET /roles/{role_id}/subroles/actors` + the shared 010/007/016/020/004/015 seams) matches the plan's architecture (attach a distinct role-keyed leaf, walk the endpoint through the reused seams).
- **C2** spec § Behavioral Accord ↔ plan § System Architecture: the plan's design serves every accord behavior (invocation, `--kind` filter, output, completeness, failure); none is contradicted.
- **C3** spec § Non-Behaviors ↔ plan § System Architecture: the plan architects no excluded capability — no transitive roll-up, no listing the anchor's own fillers, no assignment-shaped projection, no separate `people` command, no leaf-`404` special-case (ADR-3 explicitly forbids it), no writes.
- **C4** plan § Architecture Decisions ↔ interface-cli § Surface: the interface reflects ADR-1 (a distinct role-keyed leaf, not a subcommand of `actors`), ADR-2 (reuse the `actors` render + `validateKind`, `--kind` only), and ADR-3 (path swap; leaf-`404` surfaced verbatim).
- **C5** plan § System Architecture ↔ tasks § Task Scope: T001 builds the leaf the plan describes; T002 the BDD suite. No task builds anything the plan doesn't mention.
- **C6** interface-cli § Surface ↔ feature § steps: every scenario step references a surface the interface defines (`subrole-actors`, the subroles actors endpoint, `--kind`, `--first-page`); no step invents an endpoint or flag.

## Completeness: 6/6 passed

### Findings

None.

### Passed (6/6)

- **K1** spec § Driving Scenarios → feature: all 8 driving scenarios (3 happy, 2 error, 3 edge) and all 5 validation scenarios have Gherkin equivalents; the mid-walk partial and the kind-carried-on-every-page assertions are added as architecture-informed scenarios.
- **K2** spec § Integration Boundaries → interface files: the single Glassfrog-API CLI boundary has interface-cli.md.
- **K3** plan § Implementation Strategy → tasks: both plan phases (Command, BDD) have task decomposition (T001, T002).
- **K4** plan § System Architecture → tasks § Task Scope: the one new component (the roll-up leaf) has an implementing task (T001); the BDD suite is T002.
- **K5** interface-cli § Surface → feature: the `subrole-actors` surface (and its flags, completeness, error paths) has scenario coverage.
- **K6** spec § User Scenarios → interface: all four user scenarios (see who staffs the circles, roll up in one read, tell people from agents, trust the whole) map to the single command surface the interface defines.

## Coherence: 4/4 (original P2 finding H1 resolved in PR #120)

### Findings

- **H1 (P2 — RESOLVED in PR #120)** — **Command-spelling drift: spec vs. downstream artifacts.** The spec's driving/validation scenario prose uses `glassfrog actors subroles <role-id>` (e.g. spec.md § Driving Scenarios), while plan ADR-1 superseded that *form* (rejecting a `subroles` subcommand under the positional-bearing `actors`) and interface-cli.md pinned the command as `glassfrog subrole-actors <role-id>` — the spelling the feature file and tasks.md use throughout. **Mitigation already present**: the spec's Assumptions section flagged the exact command spelling as `[ASSUMED]`, explicitly deferring it to the interface stage, and the spec's scenario *titles* (the `# Source:` keys) are spelling-agnostic, so the feature file maps cleanly. The behavior is fully consistent across all artifacts — only the illustrative command token differs. **Recommendation (developer's call, non-blocking)**: optionally backfill the spec's scenario prose to `subrole-actors` (or to a neutral phrasing) for surface symmetry, or leave it — the spec's `[ASSUMED]` deferral makes the interface the authoritative source for the spelling. *(See also the open naming question carried from shape: `subrole-actors` vs `subrole-fillers`; if that is revisited, update spec + interface + feature + tasks together.)*

### Passed (4/4)

- **H1** (resolved) Command spelling is now consistent — spec.md, plan.md, interface-cli.md, the feature file, and tasks.md all use `subrole-actors <role-id>` (the rejected `actors subroles` form appears only where artifacts explain *why* it was rejected).
- **H2** Detail symmetry (spec↔plan, plan↔tasks): proportionate — no artifact carries 3x+ the detail of its neighbor on a shared topic (the thin 2-task decomposition matches the thin single-phase plan).
- **H3** Scope alignment (spec + interface + tasks): the same capability — a one-level roll-up, one command, the `--kind` filter, full-walk completeness, the leaf-`404`/empty-`200` distinction, the actor-not-assignment shape — appears in all three; nothing is added or dropped.
- **H4** Phase coverage (plan + tasks): tasks' two phases (Command, BDD) map exactly to the plan's two ordered steps; no task references a phase the plan doesn't define, and no plan phase lacks tasks.

---

## Checklist Correlation

Checklist correlation: no overlapping findings — checklist reported 0 failures, so there were no vertical findings to correlate with the original horizontal P2 finding (H1, now resolved in PR #120). H1 was a coherence/terminology drift, a category checklist (vertical, per-artifact) does not evaluate.

---

## Governance Notes

- **All 16 base relationship checks ran** — the full artifact set (spec, plan, 1 interface, 1 feature, tasks) was present; no checks were skipped for missing artifacts.
- **Checklist context**: loaded — 14/14 constitution checks pass (no done-* accords deployed project-wide).
