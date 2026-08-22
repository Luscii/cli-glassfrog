# Tasks: Pre-Assembly Grammar Consultation

**Feature**: 079-pre-assembly-grammar-consultation
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/unguided-change-construction/pre-assembly-grammar-consultation.feature, features/proposal-circle-not-choosable/pre-assembly-routing-application.feature

---

## Dependency Graph

Phase 1: The wired gate (2 tasks, no phase dependencies; within the phase T002 depends on T001) [Shared]
Phase 2: The truth sweep (1 task, depends on Phase 1) [Shared]

3 tasks total | T002 and T003 parallelizable once T001 lands | Builder: implement (BDD outer loop)

## Branching Guidance

**Role-based mode**: `spec/079-pre-assembly-grammar-consultation/base` as integration point; task branches `spec/079-pre-assembly-grammar-consultation/task-1` … `…/task-3` in T-order. Two sibling specs (075-legacy-identifier-request, 078-invalid-create-outcome) have complete artifact sets awaiting implementation — 079 touches `plugin/` and `internal/build` BDD files neither sibling edits, but coordinate on `internal/build` if landing in the same window (different files, no shared state). Phase 2 rides in the same PR as Phase 1 (plan: the sweep is only honest alongside the change that forces it) as a separable commit.

## Scenario inventory

**`pre-assembly-grammar-consultation.feature`** (12 — the gate and its legibility): 8 runnable (`@wip`), 4 held (`@validation @wip`).
**`pre-assembly-routing-application.feature`** (7 — the applied routing): 6 runnable (`@wip`), 1 held (`@validation @wip`). Two are architecture-informed additions closing the analyze K5 gap: the root-circle decline and the widened-description assertion.
Held scenarios stay `@wip` for /score:validate; the runners' `~@wip` filter never executes them.

---

## Phase 1: The wired gate [Shared]

- [ ] **T001** [Shared] Wire the gate into the drafting path's three plugin artifacts, in one coherent edit
  - **Scope**: `plugin/skills/proposal-drafting/SKILL.md`, `plugin/agents/proposal-drafter.md`, `plugin/agents/proposal-drafting-commands.txt` — and no other file. The skill's workflow section becomes the nine-step gate order (route → ground → situate → duplicate check → consult → assemble → match → confirm & create → hand off) with the two-phase relay documented; both frontmatter descriptions widen to the routed entry while keeping every existing boundary sentence; the agent's fence grows to the eight leaves, its output contract gains the `consultation` element (grammar / routing / match parts), its `action` vocabulary grows to seven values, and three defensive entries are added; the registry gains the `proposal grammar` line with its consultation-read comment and the routing block's inert annotation is rewritten (matching fence note rewritten in the same pass). All per interface-spec.md's Surface tables — the accord's enumerations are canonical.
  - **Acceptance criteria**:
    - The nine steps appear in the workflow in the accord's order, each naming its composed leaves; the relay loop (return → direction → re-delegate, repeat from the top, direction-present-means-act) is documented
    - The registry lists exactly the eight leaves; no sentence in registry or fence claims the routing reads are ahead of their use or imply no routing step
    - No new sentence restates a change type, placement rule, recorded shape, or the routing rule; every new sentence resolves in-surface or to the CLI (self-containment check green)
    - Every phrase the existing 067/073 executable expectations pin is retained: full `go test ./internal/build/...` green unchanged, including `CheckProposalDraftingDrift` (which must pass with zero guard-code edits, per plan ADR-3) and the 067 BDD suite
    - `gofmt -l .` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (in-place widening), ADR-2 (two-phase decision points), ADR-3 (ordinary read, no guard code)
  - **Interface references**: interface-spec.md: The workflow contract; The fence contract; Draft-record output shape; Defensive-drafting contract; Invocation
  - **Scenario references**: these artifact edits are what T002's suite asserts — see T002 for the scenario list

- [ ] **T002** [Shared] [P] BDD content-inspection suite binding the two new feature files
  - **Scope**: New `internal/build` godog test file(s) (sibling shape to `proposal_drafting_bdd_test.go` / `circle_routing_rule_bdd_test.go`) binding `features/unguided-change-construction/pre-assembly-grammar-consultation.feature` and `features/proposal-circle-not-choosable/pre-assembly-routing-application.feature` with the `~@wip` filter, plus step definitions asserting the T001 artifacts' load-bearing content. Any parsing/normalization helpers go in production source (`internal/build/*.go`), not test files, per the established operator-path discipline; comparisons are whitespace-normalized.
  - **Acceptance criteria**:
    - All 14 runnable scenarios pass once their `@wip` tags are removed by the implement loop; the 5 `@validation` scenarios stay held
    - Step definitions assert, at minimum: the workflow's gate order; the surfaced-dead-shape conduct (handle + shape + symptom, no verdict); the grammar-failure continuation; the routing-answer shape (target circle, all anchors, none chosen); the mismatch/empty-set/incomplete-walk conducts; the three new `action` values and the `consultation` element in the output contract; the eight-leaf registry with `proposal grammar` outside the gated set and `proposal create` its only gated member; the annotation rewrite (no inert claim survives); the root-circle decline (the routing part carries the record's decline, no target invented); and the widened frontmatter descriptions (each states the routed entry with every prior boundary sentence retained)
    - No hard-coded copy of a registry or record fact a source-derived read can supply (drift-guard-must-not-hardcode discipline)
    - `gofmt -l .` clean; full `go test ./...` green
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: pre-assembly-grammar-consultation.feature — all 8 non-`@validation` scenarios; pre-assembly-routing-application.feature — all 6 non-`@validation` scenarios

## Phase 2: The truth sweep [Shared]

- [ ] **T003** [Shared] [P] Sweep the statements 079 falsifies, in the same PR
  - **Scope**: Three bookkeeping edits, no plugin or production-code changes: (1) `features/proposal-circle-not-choosable/circle-routing-guard.feature` — delete the `@validation @wip` scenario "No workflow step consults the record or runs its reads to route", leaving a source comment noting the consultation landed with this feature (runner-safe: `~@wip` filter, no Go step binds it); (2) `specs/067-proposal-drafting-path/spec.md` — dated superseding note on the Entry accord group and the "Entry is an existing tension id" assumption, pointing at 079's widened entry (original text stays legible; the note is self-contained); (3) re-validation addenda appended to `specs/067-proposal-drafting-path/validate.md` and `specs/073-circle-routing-rule/validate.md` naming the validate-pinned surfaces this feature edited and the disposition of each (the 073→067 addendum is the precedent shape).
  - **Acceptance criteria**:
    - No repository text still claims the drafting path's workflow is unchanged by the routing reads or that no workflow step consults the records — grep for the retired scenario's phrases returns only historical spec/validate records, never a live artifact
    - The 067 note names what changed (entry, element list, fence count, action vocabulary) and which feature changed it; both addenda name the edited surfaces
    - Portfolio files (FEATURE-MODEL.md, BACKLOG.md, ROADMAP.md) are untouched (plan ADR-4: status reconciliation is the prioritize stage's job)
    - Full `go test ./...` green (the deletion removes a never-executed scenario)
  - **Dependencies**: T001 (the sweep is only honest once the wiring exists)
  - **Plan reference**: Phase 2; ADR-4 (upstream truth sweep)
  - **Scenario references**: none executable — this task retires a held scenario; 079's own held validation scenario "The registry no longer claims the routing reads are ahead of their use" verifies the adjacent annotation flip at /score:validate
