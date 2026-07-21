# Tasks: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/operating-surface-packaging.feature

---

## Dependency Graph

Phase 1: Distribution vehicle (2 tasks, no dependencies) [Shared]
Phase 2: Setup skill (1 task, no functional dependency — parallel with Phase 1) [US2]

3 tasks total | 2 phases parallelizable | Builder: implementing agent (BDD outer loop)

---

## Branching Guidance

**Pipeline mode**: `spec/070-operating-surface-packaging/base` → `spec/070-operating-surface-packaging/task-1`, `spec/070-operating-surface-packaging/task-2`, `spec/070-operating-surface-packaging/task-3`

T001 and T003 both touch `internal/build/operatingsurfacepackaging.go` (T003 extends the guard file T001 creates). If built on parallel branches, sequence T003's guard additions after T001 merges, or expect a merge in that one file.

---

## Phase 1: Distribution vehicle [Shared]

- [ ] **T001** [Shared] Ship the repo-root marketplace manifest and its consistency guard
  - **Scope**: Create `.claude-plugin/marketplace.json` at the repo root (ADR-1/-2) and the `internal/build` consistency guard (ADR-5). One reviewable change: the distribution manifest plus the test that keeps it honest.
  - **Acceptance criteria**:
    - `.claude-plugin/marketplace.json` exists at the repo root with `name: "glassfrog"`, `owner: {name: "Luscii"}`, `$schema` set to the Claude marketplace schema URL, and a `plugins` array
    - The single `plugins` entry has `name: "glassfrog"`, `source: "./plugin"`, and a `description` equal to `plugin/.claude-plugin/plugin.json`'s `description`; it carries **no** `version` key
    - `internal/build/operatingsurfacepackaging.go` exports the path constants and parse helpers (production source, not `_test.go`)
    - A guard test asserts, with both sides read from disk: a `plugins` entry named `glassfrog` exists (not "is the only entry"); its `source` resolves to a directory containing a plugin manifest; entry `name` and `description` equal the plugin manifest's; the entry has no `version` key
    - The guard's partial coverage is stated in-test (it checks in-repo consistency, not the host's marketplace schema)
    - `go test ./...` and `gofmt -l .` are clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Distribution vehicle; ADR-1, ADR-2, ADR-5
  - **Scenario references**: operating-surface-packaging.feature: "Marketplace add lists the glassfrog plugin", "Install brings the plugin's surface into the environment", "Marketplace entry drift is a defect", "Guard fails when a version pin appears on the marketplace entry", "Marketplace entry matches the plugin it ships" (@validation), "A sibling plugin is one appended entry", "Marketplace shape admits additional entries" (@validation)
  - **Interface references**: interface-spec.md: `marketplace.json` schema, Guard contract

- [ ] **T002** [Shared] Document the install flow for the operating surface
  - **Scope**: Add an "agent operating surface" install section to README and `docs/guides/agent-operators/`, beside the existing CLI install section — the marketplace-add then plugin-install steps. Prose only; no code.
  - **Acceptance criteria**:
    - README gains a section showing `/plugin marketplace add Luscii/cli-glassfrog` followed by `/plugin install glassfrog@glassfrog`
    - The section states the CLI is a prerequisite and points at the existing CLI install channels rather than repeating them
    - The agent-operators guide carries the same flow in guide form
    - Command names in the prose match the shipped surface (reviewed, not guarded — prose is not enumerable)
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 — Distribution vehicle; Cross-cutting Concerns (Documentation)
  - **Scenario references**: operating-surface-packaging.feature: "Marketplace add lists the glassfrog plugin"
  - **Interface references**: interface-spec.md: Invocation

---

## Phase 2: Setup skill [US2]

- [ ] **T003** [US2] [P] Add the `glassfrog-setup` skill and anchor its enumerable facts
  - **Scope**: Create `plugin/skills/glassfrog-setup/SKILL.md` (ADR-3/-4) and extend the Phase 1 guard file with the setup-skill anchors. The skill is instructed knowledge — a presence check, an auth check, and directed fixes — no shipped check script or new CLI command.
  - **Acceptance criteria**:
    - `plugin/skills/glassfrog-setup/SKILL.md` exists with frontmatter `name: glassfrog-setup` and a `description` that triggers on provisioning needs (fresh environment, post-install, command-not-found, auth failure) and stays clear of orientation's operating-question territory
    - The skill instructs a presence check (an existing innocuous command) and an auth check (a low-cost authenticated identity read), with the two failure classes kept distinct: missing binary → the three install channels; failing credential → the CLI's existing `X-Auth-Token` setup
    - The skill re-checks after a fix before reporting ready, and never installs a binary or stores a credential of its own
    - The skill carries a boundary note: setup owns the journey (check → fix → verify); orientation owns the reference — setup links, never restates
    - The guard extension anchors the enumerable facts: the three channel names appear in the skill, and the auth-check command leaf resolves in the CLI command registry; frontmatter `name`/`description` are present and non-empty
    - `go test ./...` and `gofmt -l .` are clean
  - **Dependencies**: None (functionally independent of Phase 1; shares the guard file with T001 — see Branching Guidance)
  - **Plan reference**: Phase 2 — Setup skill; ADR-3, ADR-4, ADR-5
  - **Scenario references**: operating-surface-packaging.feature: "Ready environment is reported ready", "Missing CLI routes to the install channels", "Failing credential routes to the CLI's own setup", "Setup re-checks after a fix instead of assuming success", "Setup leaves the CLI self-contained" (@validation)
  - **Interface references**: interface-spec.md: `SKILL.md` frontmatter, Required sections in `SKILL.md`, Guard contract

---

_The four `@validation` scenarios stay held out for independent verification and are referenced (not owned) by the tasks whose surface they check._
