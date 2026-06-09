# Checklist: Version Embedding

**Feature**: 023-version-embedding
**Checked against**: CONSTITUTION.md (12 principles)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, tasks.md, features/runtime-dependent-distribution/version-embedding.feature
**Checks**: 8 (8 pass, 0 fail)
**Generated**: 2026-06-08

---

## Summary

All 8 applicable constitution checks pass. Constitution: 8/8 (4 principles N/A — see Governance Notes). No done-criteria or cross-reference checks (no `accords/governance/done-*.md` present).

---

## Constitution Checks: 8/8 passed

### Failures

None.

### Passed (8/8)

- **P0** | I. Spec Fidelity ("MUST NOT invent endpoints, parameters, or behaviors"): 023 adds no command and makes no API call — it changes only the *value* reported behind the existing `--version`/`version` surface (spec.md § System Overview, Non-Behaviors; interface-spec.md "no new CLI accord"). No endpoint, parameter, or behavior is invented. **Pass.**
- **P0** | II. Action Transparency ("output … in machine-parseable form … traceable"): the reported version is a single deterministic string traceable to a known source (injected → build-info → placeholder), never free-form or fabricated; resolution is deterministic across runs (plan.md § Cross-cutting Concerns; interface-spec.md Output-value contract). Formatting/printing remains 003's surface, unchanged. **Pass.**
- **P0** | III. Fail Safe, Not Silent ("errors MUST be obvious … never hidden"): the one silent-wrong-version hazard — a blanked or stale `ldflags` injection seam reporting a wrong version — is caught loudly by a dedicated config-regression guard, not shipped silently (tasks.md T002 acceptance; interface-spec.md § Error Communication; plan.md § Risks). The resolver itself never throws and never returns empty. **Pass.**
- **P0** | IV. Test-Driven Development ("user-facing behavior MUST have an executable acceptance scenario"): user-facing version behavior has executable acceptance scenarios (features/runtime-dependent-distribution/version-embedding.feature — 12 scenarios under 3 Rule blocks; at guard time all were `@wip`, and the 4 `@validation` scenarios remain held out for independent verification), and tasks specify resolver unit tests + the config-regression test before/with implementation. **Pass.**
- **P0** | V. Composition over Monolith ("adding … MUST NOT require changing unrelated ones"): changes are confined to version handling in `internal/cli` (the resolver + the two existing 003 wiring sites) and the `builds.ldflags` line of `.goreleaser.yaml`; no unrelated command module (roles, me, auth) is touched (plan.md § System Architecture; tasks.md T001/T002 scope). **Pass.**
- **P0** | VII. Working Software ("every commit and PR MUST include implementation together with its tests"): both tasks require implementation and its tests to ship in the same PR (tasks.md T001/T002 acceptance criteria). **Pass.** *(See Governance Notes re: the principle-number cited.)*
- **P0** | VIII. No Fabricated Data ("MUST NOT invent, guess, or fill placeholder values … presented as real"): the development placeholder `0.0.0-dev` is a recognizable non-release marker, never presented as a real release; build-info versions (real tags, pseudo-versions, `(devel)`) are passed through verbatim with no invented value (spec.md § Behavioral Accord/Assumptions; interface-spec.md Output-value contract; ADR-3). **Pass.**
- **P0** | XII. Standalone Executable ("only assumed external dependency is network access to the API"): version resolution requires no runtime network, VCS checkout, or external dependency — both inputs are embedded at build time and read via stdlib `runtime/debug`; no new dependency is introduced (spec.md Non-Behavior "no runtime lookup"; plan.md § Cross-cutting Concerns; scenario "Version determination needs no network or VCS at runtime"). **Pass.**

---

## Governance Notes

- **No `accords/governance/done-*.md` present**: done-criteria and cross-reference checks were not generated. Consider creating `done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable artifact-level quality checks (this gap applies to every spec in the repo, not just 023).
- **Principle VI (Size-Aware by Design)**: no applicable checks — Version Embedding handles no result sets or pagination.
- **Principle IX (Writes Require Explicit Intent)**: no applicable checks — the feature performs no writes or mutations.
- **Principle X (Respect API Limits)**: no applicable checks — the feature makes no API calls.
- **Principle XI (Governance via Proposals)**: no applicable checks — the feature mutates no governance structure.
- **Observation (informational, not a finding)**: tasks.md T001/T002 cite "(CONSTITUTION I)" for the impl-and-tests-in-one-PR requirement, but that requirement is Principle VII (Working Software) / IV (TDD); Principle I is Spec Fidelity. The substance is satisfied — only the principle number is imprecise. This is a pre-existing repo convention (022's tasks.md carries the same citation), so it is noted, not raised as a 023 finding.
