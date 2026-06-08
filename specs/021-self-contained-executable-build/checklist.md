# Checklist: Self-Contained Executable Build

**Feature**: 021-self-contained-executable-build
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/self-contained-executable-build.feature
**Checks**: 7 (7 pass, 0 fail)
**Generated**: 2026-06-08 (VII finding resolved 2026-06-08 — see note)

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 6 | 6 | 0 |
| P1 (should fix) | 1 | 1 | 0 |
| P2 (consider) | 0 | 0 | 0 |
| **Total** | **7** | **7** | **0** |

---

## Constitution Checks: 7/7 passed

### Resolved

**P1** | CONSTITUTION.md VII (Working Software): "Every commit and PR MUST include implementation together with its tests … No code-only or test-only increments (except a RED test immediately followed by its GREEN implementation)."
→ **tasks.md** originally decomposed the work into three separate task branches (config T001 vs tests T002/T003), which as separate PRs would have been code-only/test-only increments. **Resolved**: tasks.md was restructured into a single RED→GREEN increment shipped in one PR — the config-guard test (T001, RED) precedes the `.goreleaser.yaml` config (T002, GREEN), and the self-containment verification (T003) closes the same increment. The config never ships without its tests, and tests are written first. Now passes VII (and IV ordering).

### Passed (6/6 originally)

- **P0 | III (Fail Safe, Not Silent)** — spec.md (driving scenario "a failed target fails the whole release build") + interface-spec.md (Error Communication: "build fails as a whole and emits no partial `dist/` set"; verification fails and names the violation). No swallowed errors, no partial state.
- **P0 | IV (Test-Driven Development)** — user-facing build behaviors carry executable acceptance scenarios written before implementation (12 `@wip` scenarios in the feature file); tasks T002/T003 are the verifying tests.
- **P0 | V (Composition over Monolith)** — plan ADR-1/ADR-2 keep the `.goreleaser` `builds` block and the verification tests as isolated, independently-testable parts; the verification reuses the `internal/cli/smoke_test.go` pattern without modifying unrelated command modules.
- **P0 | XII (Standalone Executable) — no pre-installed runtime dependency** — spec Behavioral Accord + plan (CGO_ENABLED=0) require the artifact to depend only on host OS + network to the API.
- **P0 | XII — clean-environment detection exists** — the self-containment verification (T002) runs a produced binary on a clean host of its target and asserts execution; scenarios pin it.
- **P0 | XII — build emits no dependency-requiring artifact** — config-guard (T003, `CGO_ENABLED=0`) + per-platform OS-only linkage allowlist (interface-spec.md) reject any binary linking a non-OS library.

## Done-Criteria Checks

Not run — no `accords/governance/done-*.md` accords exist in this repository (see Governance Notes).

## Cross-Reference Checks

Not run — cross-reference checks derive from done-* accords, which are absent. (Informally: tasks.md T001–T003 do carry plan, interface, and scenario references; the feature file carries `# Source:` comments.)

---

## Governance Notes

- **No `accords/governance/` directory.** Done-criteria and cross-reference checks could not run. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable vertical quality checks for those artifacts. (This gap applies project-wide, not to spec 021 specifically.)
- **Principles with no applicable checks for this feature** (build tooling has no runtime command/API/data/governance surface):
  - I (Spec Fidelity) — adds no command or API operation.
  - II (Action Transparency) — no runtime operator-action surface; build/verification failures do name their cause (interface Error Communication), but the principle targets the CLI's record actions.
  - VI (Size-Aware) — no result sets or pagination.
  - VIII (No Fabricated Data) — no data presentation surface.
  - IX (Writes Require Explicit Intent) — no commands or mutations.
  - X (Respect API Limits) — the build makes no API calls.
  - XI (Governance via Proposals) — no governance-structure mutation.
