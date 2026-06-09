# Checklist: PR Validation

**Feature**: 024-pr-validation
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/pr-validation.feature
**Checks**: 5 (5 pass, 0 fail)
**Generated**: 2026-06-08

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 5 | 5 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **5** | **5** | **0** |

---

## Constitution Checks: 5/5 passed

- **P0 | III (Fail Safe, Not Silent)** — the gate fails closed and never hides a failure: spec §Result and gate + interface Error Communication require a missing or failed `ci-success` to **block** the merge ("a missing validation result fails closed"), the `ci-success` aggregation job (`needs:[lint,test]`, `if: always()`) is green **iff** lint and every test cell pass — so there is no partial-pass path — and a failing check names what failed (the lint problem or the failing OS cell). No swallowed errors, no silently-truncated verdict.
- **P0 | IV (Test-Driven Development / BDD)** — the user-facing gate behaviors carry executable-style acceptance scenarios written before implementation: 10 `@wip` scenarios in `pr-validation.feature` (3 `@validation`), each `# Source:`-traced to a spec scenario, precede the workflow they describe. tasks.md T001–T003 reference those scenarios as their verifying conditions.
- **P0 | V (Composition over Monolith)** — plan ADR-1 keeps PR Validation as an isolated declarative artifact (`.github/workflows/ci.yml` + `.golangci.yml`); adding it changes no command module, and the three jobs (lint / test / `ci-success`) are independently legible parts. The stable `ci-success` aggregation (ADR-4) decouples enforcement from the matrix shape, so the matrix can evolve without touching the protection rule. No entanglement with sibling capabilities.
- **P0 | VII (Working Software)** — every task is a working, self-validating increment: T001 creates a functional lint workflow, T002 adds the test matrix, T003 adds the gate — each leaves `ci.yml` valid and the workflow runnable (tasks.md notes they may collapse into one PR). There is no code-only/test-only split: the capability adds **no** Go code and **no** new tests — it runs the project's existing `go test ./...` suite — and the workflow validates itself by running on its own introducing PR. (Unlike 021's original three-branch split, 024's tasks were decomposed as working increments from the start.)
- **P0 | XII (Standalone Executable) — no new runtime dependency; actively guards XII** — PR Validation introduces golangci-lint and the Go toolchain as **CI-host** tools only; plan ADR-2, interface Consistency Notes, and DECISIONS all record that XII governs the produced binary's runtime, not the CI host (same standing as GoReleaser / `sigs.k8s.io/yaml`). It adds nothing to the distributed artifact. In fact this gate *enforces* XII: its `go test ./...` runs the `internal/build` host self-containment test and the `.goreleaser.yaml` config-guard test that 021/022 rely on PR Validation to run before merge.

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords exist in this repository (see Governance Notes).

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent. (Informally: tasks.md T001–T003 each carry plan, interface, and scenario references; `pr-validation.feature` carries `# Source:` comments traced to spec scenarios.)

---

## Governance Notes

- **No `accords/governance/` directory.** Done-criteria and cross-reference checks could not run. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable vertical quality checks for those artifacts. (This gap applies project-wide, not to spec 024 specifically — same note as 021.)
- **Principles with no applicable checks for this feature** (CI infrastructure has no runtime command / API / data / governance surface):
  - I (Spec Fidelity) — adds no CLI command or Glassfrog API operation.
  - II (Action Transparency) — no runtime operator-action surface; the gate's failures *do* name their cause (spec §Result, interface Error Communication), but the principle targets the CLI's record actions, not CI status.
  - VI (Size-Aware) — no result sets or pagination.
  - VIII (No Fabricated Data) — no data-presentation surface.
  - IX (Writes Require Explicit Intent) — adds no CLI command or mutation (the one-time `gh api` branch-protection setup is maintainer tooling, not a CLI command path).
  - X (Respect API Limits) — the workflow makes no Glassfrog API calls (its `gh api` use targets GitHub, not the rate-limited Glassfrog API the principle governs).
  - XI (Governance via Proposals) — no governance-structure mutation.
