# Checklist: Main-Branch Verification

**Feature**: 029-main-branch-verification
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/no-automated-pipeline/main-branch-verification.feature
**Checks**: 5 (5 pass, 0 fail)
**Generated**: 2026-06-09

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

- **P0 | III (Fail Safe, Not Silent)** — a post-merge regression is made obvious and named, never hidden: spec §Result + interface Error Communication require a failing run to turn red and attach a **failing commit status** to the offending commit, and `fail-fast: false` (shared `test.yml`) names the failing OS cell so the failure is reproducible. No swallowed errors, no silent verdict. **Calibration note (distinct from 024):** 029 deliberately does **not** "fail closed / block" — ADR-3 — because the code is *already merged*; there is nothing to gate. Fail Safe here means "loud and recoverable," which is satisfied by the red run + commit status + named cell. This is the correct reading for a post-merge net, not a gap relative to 024's merge-blocking gate.
- **P0 | IV (Test-Driven Development / BDD)** — the user-facing behaviors carry executable-style acceptance scenarios written before implementation: 10 scenarios in `main-branch-verification.feature` (7 `@wip` + 3 `@validation @wip`), each `# Source:`-traced to a spec scenario, precede the workflow YAML they describe. tasks.md T001–T002 reference those scenarios as their verifying conditions. (For CI infra the scenarios are validated by the workflow running on its own introducing PR / first merge — see VII — rather than godog step definitions, the same standing 024 carries.)
- **P0 | V (Composition over Monolith)** — 029 *reduces* entanglement: plan ADR-1 extracts the shared test matrix into a single reusable workflow (`test.yml`) so the pre- and post-merge gates stop duplicating it, and adds the post-merge net as an isolated declarative artifact (`main-verify.yml`). It changes no command module. **Explicit cross-capability note:** T001 edits a sibling's shipped artifact (024's `ci.yml`, whose `test` job becomes a `uses:` call). This is *not* a Composition violation — the edit is sanctioned (024's plan/DECISIONS explicitly deferred the shared-workflow call to 029), it is the opposite of monolith (one source of truth replaces two hand-synced copies), and it preserves 024's required `ci-success` check (which binds the stable name, not matrix-cell names). The introducing PR runs through 024's own gate, so a broken refactor fails loudly before merge. Surfaced so a reviewer sees the cross-artifact edit deliberately, not by surprise.
- **P0 | VII (Working Software)** — every task is a working, self-validating increment: T001 leaves both `test.yml` and the refactored `ci.yml` valid and runnable, T002 adds a functional `main-verify.yml` (tasks.md notes the two may collapse into one PR). No code-only/test-only split: the capability adds **no** Go code and **no** new tests — it re-runs the project's existing `go test ./...` suite via the shared workflow — and each workflow validates itself (actionlint clean per task acceptance; `main-verify.yml` first fires on the merge of its own PR).
- **P0 | XII (Standalone Executable) — no new runtime dependency; actively exercises XII** — 029 introduces the Go toolchain as a **CI-host** tool only; plan Cross-cutting Concerns and interface Consistency Notes record that XII governs the produced binary's runtime, not the CI host (same standing as GoReleaser / golangci-lint). It adds nothing to the distributed artifact. The shared `go test ./...` it re-runs includes the `internal/build` host self-containment test and the `.goreleaser.yaml` config-guard test — so the post-merge net also re-confirms XII's guards on every merge to `main`.

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords exist in this repository (see Governance Notes).

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent. (Informally: tasks.md T001–T002 each carry plan, interface, and scenario references; `main-branch-verification.feature` carries `# Source:` comments traced to spec scenarios.)

---

## Governance Notes

- **No `accords/governance/` directory.** Done-criteria and cross-reference checks could not run. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable vertical quality checks for those artifacts. (This gap applies project-wide, not to spec 029 specifically — same note as 021/024.)
- **Principles with no applicable checks for this feature** (CI infrastructure has no runtime command / API / data / governance surface):
  - I (Spec Fidelity) — adds no CLI command or Glassfrog API operation.
  - II (Action Transparency) — no runtime operator-action surface; the workflow's failures *do* name their cause (spec §Result, interface Error Communication), but the principle targets the CLI's record actions, not CI status.
  - VI (Size-Aware) — no result sets or pagination.
  - VIII (No Fabricated Data) — no data-presentation surface.
  - IX (Writes Require Explicit Intent) — adds no CLI command or mutation. (029 adds **no** `gh api` usage at all — unlike 024 it carries no branch-protection setup, since ADR-3 adds no enforcement layer.)
  - X (Respect API Limits) — the workflow makes no Glassfrog API calls.
  - XI (Governance via Proposals) — no governance-structure mutation.
