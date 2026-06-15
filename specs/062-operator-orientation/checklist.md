# Checklist: Operator Orientation

**Feature**: 062-operator-orientation
**Inputs**: CONSTITUTION.md (12 principles); spec.md, plan.md, interface-spec.md, features/unequipped-agent-operators/operator-orientation.feature, tasks.md
**Check sources**: Constitution only — no `accords/governance/done-*.md` present
**Checks**: 11 generated, 11 pass, 0 fail (2 principles N/A — no applicable checks)
**Generated**: 2026-06-15

---

## Summary

| Severity | Pass | Fail | Total |
|---|---|---|---|
| P0 | 8 | 0 | 8 |
| P1 | 3 | 0 | 3 |
| P2 | 0 | 0 | 0 |
| **Total** | **11** | **0** | **11** |

Calibration note: this feature adds **no CLI code** — it is a Claude plugin packaging operating knowledge. Runtime-behavior principles (II, VI, VIII, IX, X) were calibrated to their *translated* form: the orientation must not **describe** a violation, or it **teaches** the safe behavior the principle protects. Principles VIII and IX produced no directly-applicable checks and are noted under Governance Infrastructure.

---

## Constitution Checks

### C-I — Spec Fidelity → no invented surface (P0) — PASS
**Principle I** (MUST NOT invent endpoints/parameters/behaviors). Calibrated: the orientation names no CLI command, flag, or output format the CLI does not expose.
**Evidence**: spec Non-Behaviors #1 ("must not add any API capability, command, or flag"); interface-spec "names no surface the CLI lacks"; feature scenario "Orientation names no surface the CLI lacks" (@validation); tasks T002 acceptance + T003 drift anchors (`supportedFormats`, `ExitCode`, `auth login`).

### C-II — Action Transparency → teaches parseable output + traceable reactions (P1) — PASS
**Principle II** (NON-NEGOTIABLE). The principle's direct subject — the CLI's own action output — is **N/A** (the plugin performs no action). Calibrated supporting check: the orientation directs the agent to structured output and exit-code reactions so the agent can act traceably.
**Evidence**: interface-spec required sections (output-for-parsing names `json`/`yaml`; exit-code reactions, 0–6); feature scenarios "Select a parseable output format", "React to a non-zero exit code".

### C-III — Fail Safe, Not Silent → drift guard fails loudly (P0) — PASS
**Principle III** (errors obvious, never hidden). Calibrated: the drift guard fails loudly naming the offending anchor; any reduced coverage is stated, never silent.
**Evidence**: plan ADR-4 ("a reduction must be stated, not silent — LEARNINGS: no silent caps"); tasks T003 acceptance ("fails loudly and names the offending anchor; any anchor deliberately left uncovered is documented in the test, not omitted silently"). Recoverable-error guidance: spec write-safety (412 → re-read + re-confirm).

### C-IV — Test-Driven (Red→Green / BDD) → acceptance scenarios precede code (P0) — PASS
**Principle IV**. Executable acceptance scenarios exist before implementation; tasks reference them.
**Evidence**: feature file with 14 `@wip` scenarios written (STATUS: "Scenarios written" precedes "Tasks ready"); every task (T001/T002/T003) carries Scenario references. The drift guard (T003) is itself a test.

### C-V — Composition over Monolith → additive, modular (P0) — PASS
**Principle V** (MUST be modular; adding a command MUST NOT change unrelated ones). 
**Evidence**: plan ADR-1 (plugin is a separate top-level `plugin/`, distinct from the Go CLI) and ADR-2 (#63/#64–#69 land as additive sibling skills, never a restructure); recorded in DECISIONS. Adding the plugin requires no edit to any existing CLI command.

### C-VI — Size-Aware → teaches pagination (P0) — PASS
**Principle VI** (never silently truncate). Calibrated: the orientation teaches detecting and fetching additional pages.
**Evidence**: interface-spec pagination section; feature scenario "Page through a multi-page result set".

### C-VII — Working Software → content paired with verification (P1) — PASS
**Principle VII** (no code-only/test-only increments). 
**Evidence**: the content (T001/T002) is accompanied by the BDD acceptance scenarios (feature file) and the drift-guard test (T003). **Note**: for markdown content the testable surface is the scenarios' step definitions + the drift guard — implement's BDD outer loop must write those step definitions; this is a coherence point for analyze, not a checklist failure.

### C-VIII — No Fabricated Data (—) — N/A
**Principle VIII**. The plugin renders no API data, so there is no fabrication surface. The adjacent "no invented behavior" concern is covered by C-I. No applicable checks. (See Governance Infrastructure.)

### C-IX — Writes Require Explicit Intent → no mutating path introduced (P0) — PASS
**Principle IX**. The plugin exposes no command and performs no reads or writes; it explicitly does not enforce or perform writes (guidance only), so no read-shaped path can mutate.
**Evidence**: plan System Architecture ("adds no code to the CLI"); spec Non-Behavior ("must not enforce, gate, or block writes").

### C-X — Respect API Limits → teaches concurrency + rate-limit reaction (P0) — PASS
**Principle X** (`If-Match`/`ETag` concurrency; `429` backoff). Calibrated: the orientation describes the 412 stale-write re-read+re-confirm (optimistic concurrency) and the rate-limit exit-code reaction.
**Evidence**: spec/interface write-safety (412 → re-read, not blind retry); exit-code section covers the 0–6 convention including rate-limit; feature scenario "Surface the write-safety expectation without gating".

### C-XI — Governance via Proposals → no default mutating path (P0) — PASS
**Principle XI**. The plugin introduces no governance-mutating command path and does not bypass the proposal flow; write-safety is described as guidance only.
**Evidence**: spec Non-Behaviors; plan (no command added). The orientation reinforces, rather than bypasses, the proposal-gated discipline.

### C-XII — Standalone Executable → CLI self-containment unaffected (P0) — PASS
**Principle XII**. The plugin adds no Go code or runtime dependency to the CLI binary; it is separate content consumed by the Claude plugin host. The drift guard is test-only (`internal/build`, not shipped in the binary).
**Evidence**: plan System Architecture ("adds no code to the CLI"); ADR-4 (test in `internal/build`).

---

## Governance Infrastructure Notes

*(Separate from feature quality findings.)*

- **No `accords/governance/done-*.md` accords exist in this project.** Checklist ran constitution checks only; done-criteria checks (per-artifact bars for specify/plan/interface/scenarios/tasks) were not available. Consider creating `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, and `done-tasks.md` to enable per-artifact done-criteria checking on future runs.
- **Principle VIII (No Fabricated Data)** produced zero checks — this feature renders no API data; the boundary it protects is covered for this feature by C-I (Spec Fidelity / no invented surface).
- **Principle II (Action Transparency)** — its literal subject (the CLI's own action output) is N/A for a knowledge artifact; only the supporting "teaches parseable output" check (C-II, P1) applies.
