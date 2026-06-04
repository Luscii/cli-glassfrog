# Checklist: Help & Version

**Feature**: 003-help-and-version
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, features/no-runnable-cli/help-and-version.feature, tasks.md
**Checks**: 6 (6 pass, 0 fail)
**Generated**: 2026-06-03

---

## Summary

All 6 checks pass. Constitution: 6/6. Done-criteria: not run (no accords). Cross-references: not run (no accords).

4 of 12 constitution principles produced applicable checks; 8 are N/A for this internal, read-only CLI-presentation feature (see Governance Notes).

---

## Constitution Checks: 6/6 passed

### Failures

None.

### Passed (6/6)

**P0** | CONSTITUTION.md IV (TDD): "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/no-runnable-cli/help-and-version.feature**: 14 `@wip` 003 scenarios exist before implementation, covering listing, usage, version, built-ins-hidden, precedence, and the boundary cases.

**P0** | CONSTITUTION.md IV: "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001, T002**: T001 acceptance criteria mandate RED-first unit tests (version parity, built-ins absent, `--help` renders, sorting pinned, precedence); T002 provides executable acceptance.

**P0** | CONSTITUTION.md V (Composition over Monolith): "Adding a new command MUST NOT require changing unrelated ones"
→ **plan.md § ADR-1 / ADR-2 / System Architecture**: help/version is a single `configureHelpAndVersion(root)` pass; no existing command package is edited, and later commands get standard help for free by setting a non-empty `Short`.

**P0** | CONSTITUTION.md V: "modular, independently-testable parts"
→ **plan.md § System Architecture**: the feature is root configuration applied at one site; command modules stay in their own packages, and the config pass is exercised by calling the assembled root in tests.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests
→ **tasks.md § T001, T002**: both tasks carry verifiable acceptance criteria with tests; `go build ./...` and `go vet ./...` clean are explicit T001 criteria.

**P0** | CONSTITUTION.md XII (Standalone Executable): "self-contained executable … no language runtime"
→ **plan.md § System Architecture / go.mod**: the feature adds only cobra configuration (cobra is already a dependency); no new runtime or external software is introduced, so the artifact remains a single self-contained Go binary.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria and cross-reference checks did not. Consider creating, to enable vertical quality checks for later specs:
  - `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`
- **Principle I (Spec Fidelity)**: N/A — Help & Version invokes no API operation; `--help`/`--version`/`version` are framework surfaces, not spec-operation bindings. Deferred to endpoint-command specs.
- **Principle II (Action Transparency)**: N/A — this capability performs no API-backed action and targets no record, so the "report operation + resource" clause does not apply, and it surfaces no errors of its own (unknown-command/flag errors belong to Argument Dispatch). See Advisory Notes — the determinism property aligns with II's machine-legibility spirit.
- **Principle III (Fail Safe, Not Silent)**: N/A — no writes, no governance state, no validate-before-send. The empty-set "does not fail" behavior is graceful degradation, not a write-safety concern.
- **Principle VI (Size-Aware)**: N/A — no API result sets or pagination. The faithful command listing (no command silently omitted) echoes VI's no-silent-truncation spirit but is an in-process set, not an API page.
- **Principle VIII (No Fabricated Data)**: N/A — no API data is presented; listings show only registered names/summaries.
- **Principle IX (Writes Require Explicit Intent)**: N/A — no writes or mutations.
- **Principle X (Respect API Limits)**: N/A — no network calls.
- **Principle XI (Governance via Proposals)**: N/A — no governance-mutating command path.

## Advisory Notes (not severity findings)

These passed (or are out of scope) but are worth the developer's attention:

- **II — legible output, not structured output**: decision B adopts cobra's standard *human-text* help; `--json`/structured output is explicitly out of scope (spec non-behavior). Agent-parseability therefore rests on the **determinism** the spec guarantees (alphabetical, identical version output) rather than a machine format. The validation scenario "Listing output is identical across repeated runs" pins this. If agents later need structured help, that is a new spec, not a checklist gap.
- **VII — T001 bundles three concerns**: T001 combines version-unify, built-in-hiding, and the sorting pin in one PR. Its acceptance criteria carry tests for each, so it satisfies VII, but the reviewer should confirm all three test groups land in T001's PR rather than deferring any to T002.
