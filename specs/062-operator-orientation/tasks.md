# Tasks: Operator Orientation

**Feature**: 062-operator-orientation
**Concretization**: Full context
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/operator-orientation.feature

---

## Dependency Graph

Phase 1: Plugin scaffold + orientation content (2 tasks, no phase dependencies; intra-phase: T002 depends on T001) [Shared]
Phase 2: Drift guard (1 task, depends on Phase 1 — specifically on T002) [Shared]

3 tasks total | 0 phases parallelizable | Builder: pipeline (single active spec)

---

## Branching Guidance

**Pipeline mode**: `spec/062-operator-orientation/base` → `spec/062-operator-orientation/task-1`, `spec/062-operator-orientation/task-2`, `spec/062-operator-orientation/task-3`

T002 depends on T001 and T003 depends on T002, so the three task branches land in sequence onto the spec base.

---

## Phase 1: Plugin scaffold + orientation content [Shared]

- [ ] **T001** [Shared] Create the plugin scaffold and manifest
  - **Scope**: Add the top-level `plugin/` directory with `plugin/.claude-plugin/plugin.json` and an empty-bodied `plugin/skills/glassfrog-operator/SKILL.md` (frontmatter only). One structural change — the installable shell, no orientation prose yet.
  - **Acceptance criteria**:
    - `plugin/.claude-plugin/plugin.json` parses as JSON and carries the required `name` (`glassfrog-operator`), `version` (`0.1.0`), and `description`, plus a recommended (optional) `author` `{name}`; no `skills`/`commands`/`hooks` keys (skills are directory-discovered)
    - `plugin/skills/glassfrog-operator/SKILL.md` exists with YAML frontmatter `name` + `description`, the `description` stating when to consult it and naming the CLI-driving topics so it triggers on the right need
    - No `marketplace.json`, publishing workflow, or install flow is added (distribution is #70)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (plugin home & layout), ADR-2 (one skill, additive growth)
  - **Scenario references**: operator-orientation.feature: "Orientation is consultable once the plugin is present", "Plugin defines no distribution machinery", "Malformed manifest leaves the plugin unloadable"
  - **Interface references**: interface-spec.md — Surface (structural layout, `plugin.json` schema, `SKILL.md` frontmatter)

- [ ] **T002** [Shared] Author the orientation skill content
  - **Scope**: Fill `SKILL.md` body with the cross-cutting operating knowledge — the required topic sections — pointing at `glassfrog help` for per-command detail. Authoring only; adds no CLI code.
  - **Acceptance criteria**:
    - Sections present: output-for-parsing (names `json`/`yaml` as the parseable formats from the supported set `full, compact, json, yaml`), pagination, exit-code reactions (the 0–7 convention, including `StaleWrite`=7 for the `412` the write-safety section covers), credentials (directs to `glassfrog auth login`, introduces no new mechanism), write-safety expectation (confirm-before-write; `412` → re-read + re-confirm) explicitly marked as guidance not enforcement, and a driving-the-command-surface section that routes to `glassfrog help`
    - Content names no command, flag, or format the CLI does not expose, and enumerates no per-command flag list
    - Content contains no Holacracy coaching / tension interpretation and no write-gating logic
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-3 (hand-authored, defer per-command detail to `--help`)
  - **Scenario references**: operator-orientation.feature: "Select a parseable output format", "Page through a multi-page result set", "React to a non-zero exit code", "Set up missing credentials", "Find per-command detail in the CLI's own help", "Orientation names no surface the CLI lacks", "Orientation carries no Holacracy coaching", "Surface the write-safety expectation without gating", "Orientation describes but never enforces gating"
  - **Interface references**: interface-spec.md — Surface (required sections in `SKILL.md`), Interactions (instructional model)

---

## Phase 2: Drift guard [Shared]

- [ ] **T003** [Shared] Add the best-effort drift-guard test in `internal/build`
  - **Scope**: A new `internal/build` test asserting the orientation's enumerable facts still match their CLI source. Best-effort and explicitly partial; if an anchor proves infeasible to assert, state the reduced coverage rather than dropping it silently.
  - **Acceptance criteria**:
    - Test asserts the orientation's output-format tokens match `internal/output` `supportedFormats` exactly (`full, compact, json, yaml`)
    - Test asserts the exit-code numbers/labels the skill cites match `internal/cli` `Outcome`→`ExitCode` (0–7, including `StaleWrite`=7 — the anchor behind the orientation's 412 guidance)
    - Test asserts the `auth login` command still exists
    - Test fails loudly and names the offending anchor when an asserted fact leaves the CLI; any anchor deliberately left uncovered is documented in the test, not omitted silently
  - **Dependencies**: T002
  - **Plan reference**: Phase 2; ADR-4 (best-effort config-drift guard)
  - **Scenario references**: operator-orientation.feature: "Detect orientation drifted from the shipped CLI", "Drift guard fails when a documented anchor leaves the CLI"
  - **Interface references**: interface-spec.md — Error Communication (drift guard fails → CI red)
