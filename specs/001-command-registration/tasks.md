# Tasks: Command Registration

**Feature**: 001-command-registration
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, interface-cli.md, features/no-runnable-cli.feature

---

## Dependency Graph

Phase 1: Go module + root command skeleton (2 tasks, no dependencies) [Shared]
Phase 2: Registration guard (1 task, depends on Phase 1) [Shared]
Phase 3: Exercise nested registration (2 tasks, depends on Phase 2) [Shared]

5 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Note: this is a foundation feature — every task is `[Shared]` because the registry it builds serves all three user scenarios (modular add / nested groups / fail-loud) rather than any single one.

---

## Branching Guidance

**Pipeline mode**: `spec/001-command-registration/base` → `spec/001-command-registration/task-1`, `spec/001-command-registration/task-2`, … (one task branch per T-id, merged back into the spec base).

---

## Phase 1: Go module + root command skeleton [Shared]

- [ ] **T001** [Shared] Initialize the Go module, project layout, and cobra dependency
  - **Scope**: Create `go.mod` (`go mod init`), add the cobra dependency, and establish the package layout that will host the command tree and guard. No command behavior yet.
  - **Acceptance criteria**:
    - `go.mod` exists with a module path and the cobra dependency pinned in `go.sum`
    - `go build ./...` succeeds on a clean checkout
    - Package layout has a clear home for the entrypoint, the root command, and the registration guard
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Go module + root command skeleton; ADR-1 (Go standalone binary), ADR-2 (cobra)

- [ ] **T002** [Shared] Create the root command and `main` entrypoint that builds a runnable binary
  - **Scope**: Define the cobra root command (the top of the known command set) and the `main` entrypoint that executes it. With no subcommands yet, invoking the binary prints root help.
  - **Acceptance criteria**:
    - `go build -o glassfrog` (with `CGO_ENABLED=0`) produces a single static binary
    - Running `glassfrog` with no arguments prints root help and exits without error
    - The root command is exposed for later wiring (subcommands can be attached to it)
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 — Go module + root command skeleton; ADR-1
  - **Interface references**: interface-cli.md: the `glassfrog` invocation surface (root)

## Phase 2: Registration guard [Shared]

- [ ] **T003** [Shared] Implement the command model and the fail-loud `Register`/`MustRegister` guard
  - **Scope**: Define the command definition (name, summary, action for leaves, children for groups) and implement `Register(parent, child) -> error` plus `MustRegister`, enforcing all five registration rules before attaching to the cobra tree. This is the spec's core. Test-first (RED→GREEN) per the constitution.
  - **Acceptance criteria**:
    - `Register` attaches a valid leaf/group under its parent and returns no error
    - Each rule rejects with a descriptive error naming the offending command: duplicate sibling name, empty/whitespace name, empty/whitespace summary, leaf without action, group with no children
    - `MustRegister` panics on violation; both surface before `cobra.Execute()` is called
    - The "group must have ≥1 child" rule is enforced at the time the group is attached to its parent
    - Unit tests cover the happy path and each of the five rules independently (RED-first)
  - **Dependencies**: T001, T002
  - **Plan reference**: Phase 2 — Registration guard; ADR-3 (fail-loud guard over cobra defaults)
  - **Interface references**: interface-spec.md: command definition + `Register`/`MustRegister` entry points + Error Communication table
  - **Scenario references**: no-runnable-cli.feature: "Registering a leaf command makes it known", "Duplicate sibling name is rejected", "Empty command name is rejected", "Missing summary is rejected", "Leaf command without an action is rejected", "Group without children is rejected"
  - **Risk**: ⚠️ Technology familiarity — cobra's `AddCommand` does not enforce these rules; the guard must wrap it and be the only attach path (guard-bypass risk from plan).

## Phase 3: Exercise nested registration [Shared]

- [ ] **T004** [Shared] Wire a `version` leaf and a nested sample group through the guard in `main`
  - **Scope**: Explicit registration assembly (ADR-4): register a `version` leaf and a sample `roles` group with `list`/`get` subcommands via `MustRegister`, proving arbitrary-depth registration and the "add without touching unrelated commands" property. A child group is assembled before being attached.
  - **Acceptance criteria**:
    - `glassfrog version` runs the version leaf's action
    - `glassfrog roles` resolves to the group and lists its `list`/`get` subcommands
    - `glassfrog roles list` and `glassfrog roles get` resolve to their leaves
    - Registration is wired explicitly in `main` (no package `init()` auto-registration); adding a command is one wiring line plus its own package
  - **Dependencies**: T003
  - **Plan reference**: Phase 3 — Exercise nested registration; ADR-4 (explicit wiring)
  - **Interface references**: interface-cli.md: invocation table (bare group, nested paths); interface-spec.md: composition example
  - **Scenario references**: no-runnable-cli.feature: "Registering a group exposes its subcommands by path", "A bare group name resolves to the group itself", "Groups nest to arbitrary depth", "A name is unique only within its own group", "Registering a command leaves existing commands untouched"

- [ ] **T005** [Shared] Make the driving scenarios pass as executable acceptance
  - **Scope**: Provide the executable acceptance coverage for `features/no-runnable-cli.feature` against the assembled CLI — registration, nested lookup, bare-group resolution, collision/validation rejection, and the startup-abort behavior. (Step definitions / `@wip` removal are the implement skill's BDD outer loop; this task ensures the scenarios are coverable end-to-end.)
  - **Acceptance criteria**:
    - Every non-`@validation` scenario in no-runnable-cli.feature has an executable path against the built CLI / registry
    - A single failed registration prevents any command from being dispatched and exposes no partial command tree
    - The two `@validation` scenarios (no implementation leakage, non-behaviors name their owners) are confirmed against the spec text (documentation checks)
  - **Dependencies**: T004
  - **Plan reference**: Phase 3 — Exercise nested registration; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: no-runnable-cli.feature: "One failed registration prevents the whole CLI from running", "Lookup is predictable from registration alone", and all Rule-block scenarios
  - **Risk**: ⚠️ Edge semantics — confirm cobra's bare-group-shows-help and unknown-path behavior match the spec; the precise unknown-path contract belongs to the later Argument Dispatch spec.
