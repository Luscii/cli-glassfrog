# Checklist: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Checked against**: CONSTITUTION.md v1.1 (13 principles)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, tasks.md, features/unguided-change-construction/agent-facing-grammar-reference.feature
**Checks**: 23 (23 pass, 0 fail)
**Generated**: 2026-08-09 (run 3 — re-derived on the rebased base after 076 merged)

---

## Summary

All 23 checks pass. Constitution: 23/23. Done-criteria: not run (no accords). Cross-references: not generated (see Governance Notes).

| Severity | Pass | Fail |
|---|---|---|
| P0 | 17 | 0 |
| P1 | 6 | 0 |
| P2 | 0 | 0 |

---

## Changes Since Previous Run

**Previous (run 1)**: 2 P0, 0 P1, 0 P2 (2 failures)
**Current (run 2)**: 0 P0, 0 P1, 0 P2 (0 failures)

**Resolved**:

- ~~P0 | CONSTITUTION.md IV: *"User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it."* → **tasks.md § Phase 2, T004**: the human-format content requirements (placements, the nesting rule, per-fact fields, provenance separation, `compact`'s one-line summaries) had no acceptance scenario — `compact` was uncovered entirely and `full` was exercised only on the empty-residue path.~~ → **fixed**. Two scenarios were added to the feature file: *"The full format presents the vocabulary, the nesting rule, and every fact"* (residue present, asserting placements, the nesting rule stated once, each fact's four fields, and the visible contract-vs-observation separation) and *"The compact format condenses each type and fact to one line"*. Both are mapped to T004 in the Scenario Disposition table, and T004 gained a `Scenario references` field. The scenario count moved 13 → 15, with the disposition table covering all 15.

- ~~P0 | CONSTITUTION.md IV: *"Features MUST be built test-first: a failing test (RED) before implementation (GREEN)."* → **tasks.md § Phase 2, T004 and T005**: acceptance required only that tests pass, so both tasks were satisfiable by writing the implementation first.~~ → **fixed**. Both tasks gained an explicit **Test-first** acceptance criterion in the form 076 adopted: T004's three human-format scenarios and its golden tests are written and observed red before the templates that satisfy them; T005's scenarios are written and observed red before the command. T003 already carried the equivalent requirement and is unchanged.

**Also addressed in run 2** (an analyze finding, recorded here because its remedy touched checked artifacts): spec.md § Integration Boundaries gained the operating-surface write gate, and tasks.md records the exit-1 decode path as a deliberate no-scenario decision covered by T002's unit test.

**Run 3 — re-derived after 076 merged to main.** The rebase changed two things this checklist depends on. (1) CONSTITUTION.md gained **Principle XIII**, so the check set grew 21 → 23 and the run-1/2 governance note about XIII being non-constitutional is retired. (2) The landed write-safety guardrail was read directly rather than from memory, correcting a claim these artifacts had carried since the plan stage: the CI forcing function for a new `proposal` leaf is `expectedProposalSurface` in `internal/build/writesafetyguardrail.go`, **not** the shell script's classification. `plan.md` (§ Integration Design, Risk R4, Phase 2), `interface-cli.md` (§ Interactions), and `tasks.md` (T005) were corrected to name both anchors and to state that only the Go-side edit is CI-enforced. No check outcome changed as a result — the correction sharpened the instruction the Builder receives.

---

## Constitution Checks

### I. Spec Fidelity

- ✅ **P0** — No invented endpoint, parameter, or API behavior. `plan.md` § System Architecture and `interface-cli.md` § The command define a command with no client, no request, and no parameters sent anywhere; the vendored contract is a generation-time input only.
- ✅ **P0** — Empirical content is never presented as contract-defined. `spec.md` § Non-Behaviors, `interface-cli.md` provenance tokens on every entry, guard invariant in `tasks.md` T003, and the human-format separation now asserted by the added `full` scenario.

### II. Action Transparency

- ✅ **P0** — Machine-parseable output exists. `interface-cli.md` § The rendered structure pins the key and token vocabulary; `tasks.md` T005 acceptance requires it.
- ✅ **P0** — Failures carry cause and next step. `interface-cli.md` § Error Communication maps each condition to an outcome and defers to the CLI's existing usage-error conduct; the drift guard's failure messages name the remedy (`tasks.md` T003).

### III. Fail Safe, Not Silent

- ✅ **P0** — No swallowed errors. `tasks.md` T002 requires the accessor to return a decode error rather than panic or render empty; `interface-cli.md` classifies it as exit 1; the Scenario Disposition note records why that path is unit-tested rather than scenario-covered.
- ✅ **P0** — No partially-applied state. The feature performs no writes; the client-less design makes a partial write structurally impossible.

### IV. Test-Driven Development

- ✅ **P0** — Every user-facing behavior has an executable acceptance scenario. The 15 scenarios cover both structured formats, both human formats (with and without residue), the usage refusal, credential-free and offline conduct, the gate pass, determinism, drift, and retirement. The single uncovered surface (exit-1 corrupt-embed) is recorded as a deliberate decision with its unit-test coverage named.
- ✅ **P0** — Tasks require RED before GREEN. T003, T004, and T005 each require their tests or scenarios observed failing before the code that satisfies them.
- ✅ **P0** — Guard behavior is specified test-first. `tasks.md` T003 requires a red-case test per divergence class.

### V. Composition over Monolith

- ✅ **P0** — Modular, independently-testable parts. Generator, `internal/grammar`, render resource, command, and guard are separate units with one-directional dependencies.
- ✅ **P1** — Adding this command forces no change to unrelated commands. T005 touches its own command file, the render resource, and one word in the gate registry; 072's path constant, guards, and record are untouched (`plan.md` ADR-1).

### VI. Size-Aware by Design

- ✅ **P0** — No silent omission. `facts` is present as `[]` when empty and both human formats state that no residue is recorded; pinned by the empty-residue scenario.

### VII. Working Software

- ✅ **P0** — Each PR pairs implementation with its tests. Phase 1 (T001–T003) lands the generator and artifact with the guard that pins them; Phase 2 pairs each unit with golden tests and scenarios, now with explicit red-first ordering.
- ✅ **P1** — No code-only or test-only increments. No task in either phase is test-only or binding-only.

### VIII. No Fabricated Data

- ✅ **P0** — Every rendered value traces to a source. `plan.md` ADR-2 derives both halves mechanically; T003's byte-comparison makes an untraceable value a build failure.
- ✅ **P0** — No invented per-type field guidance. `spec.md` § Non-Behaviors forbids it; the contract defines no per-type field schema and the rendering carries only what the record verifies.

### IX. Writes Require Explicit Intent

- ✅ **P0** — No mutation path exists. The command is client-less; no POST/PATCH/DELETE is reachable from this read.
- ✅ **P1** — The read cannot be mistaken for a write by the operator layer. T005 adds `grammar` to the gate's recognized-read set in the same change; 063's tripwire fails CI otherwise; `spec.md` § Integration Boundaries now carries the gate as a named boundary.

### X. Respect API Limits

*No applicable checks* — the feature issues no requests, so there is no `429` to back off from and no `ETag`/`If-Match` exchange to honor.

### XI. Governance via Proposals

- ✅ **P1** — No governance-structure mutation is exposed. The command renders knowledge about proposal changes; `spec.md` § Non-Behaviors forbids it from judging or accepting a change set.

### XII. Standalone Executable

- ✅ **P0** — No runtime dependency outside the binary. `plan.md` ADR-1/ADR-3 embed the artifact at build time; the command works with no credential, no network, and no operating-surface install.
- ✅ **P1** — Development-repository inputs stay development-time. The vendored contract and the grammar record are read by the generator only.

### XIII. Self-Contained Operating Surface

- ✅ **P0** — The feature's one surface edit introduces no development-repository reference. `tasks.md` T005 adds a single word (`grammar`) to `PROPOSAL_READS` in `plugin/hooks/glassfrog-write-gate.sh`; no spec number, repo path, or pipeline citation enters the surface, so the walk-derived guard stays green.
- ✅ **P0** — Pointers flow repository → surface only. `plan.md` ADR-1 rejects the record-relocation-plus-symlink option precisely because a symlink from `plugin/` into `internal/` would be a surface → repository pointer; the generated artifact lives entirely on the repository side, and the shipped surface gains no reference to it.

---

## Governance Notes

*Infrastructure observations, separate from the feature quality findings above.*

- **No `accords/` directory exists in this repository.** Done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) and cross-reference checks were not generated. Consider creating `accords/governance/done-<skill>.md` to enable them; until then, checklist runs on constitution principles alone — the standing posture for every prior spec here.
- **Principle XIII became constitutional mid-run.** 076 merged to main while this spec was being shaped, adding Principle XIII (Self-Contained Operating Surface) and the constitution's version marker (1.1). Run 1 and run 2 checked against twelve principles and carried a note that the rule was recorded-but-not-constitutional; that note is now obsolete and the two XIII checks above were generated in its place. `plan.md` ADR-1's reliance on the rule was correct before the merge and is now a constitutional citation.
- **Principle X produced zero checks** for this feature, by the nature of a command that issues no requests. Recorded per the zero-checks convention rather than omitted.
