# Tasks: Help & Version

**Feature**: 003-help-and-version
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/no-runnable-cli.feature

---

## Dependency Graph

Phase 1: Root configuration pass (1 task, no phase dependencies) [Shared]
Phase 2: Executable acceptance (1 task, depends on Phase 1) [Shared]

2 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Both tasks are `[Shared]`: Help & Version is root configuration serving all three user scenarios (discover commands / read usage / confirm build) rather than any single one.

---

## Branching Guidance

**Pipeline mode**: `spec/003-help-and-version/base` → `spec/003-help-and-version/task-1`, `…/task-2` (one task branch per T-id, merged back into the spec base). Sibling specs (002 at Analyzed) are progressing in parallel; this spec only requires 001 (implemented), so it does not wait on 002.

---

## Phase 1: Root configuration pass [Shared]

- [ ] **T001** [Shared] Configure the assembled root for help and version
  - **Scope**: Add a configuration step applied to the assembled root (e.g. `configureHelpAndVersion(root)` called from `Assemble()` after wiring), realizing three cohesive root-level concerns in one reviewable change:
    - **Version unify (ADR-3)**: set `root.Version` and a version template that prints the **bare** version value — the exact line the `version` command prints — overriding cobra's default `Name version X` template; both read the single `version` var in `internal/cli/version.go`.
    - **Hide built-ins (ADR-2)**: replace cobra's auto `help` command with a hidden command under a **non-`help` name** (e.g. `SetHelpCommand(&cobra.Command{Use: "__help_disabled", Hidden: true})`) so `glassfrog help` no longer resolves — `Hidden:true` alone only hides from listings — and disable the `completion` command (`CompletionOptions.DisableDefaultCmd = true`), while keeping the `--help` flag.
    - **Standard rendering + sorting (ADR-1)**: keep cobra's default help/usage rendering (no custom template); keep alphabetical listing (cobra's default) and pin it.
    Test-first per CONSTITUTION IV. No existing command package is edited.
  - **Acceptance criteria**:
    - `glassfrog --version` and `glassfrog version` produce byte-identical output (parity test)
    - the version-unset default is a clear non-empty placeholder (`0.0.0-dev`), never empty
    - `glassfrog --help` lists neither `help` nor `completion`; `glassfrog help` / `glassfrog completion` do not resolve as built-in commands; the `--help` flag still renders the listing/usage
    - a regression test asserts `cobra.EnableCommandSorting == true` and that a listing shows commands alphabetically (e.g. `roles` before `version`)
    - `glassfrog --help --version` produces help output, not version output (precedence)
    - `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (builds on 001, already implemented)
  - **Plan reference**: Phase 1 — Root configuration pass; ADR-1, ADR-2, ADR-3
  - **Interface references**: interface-cli.md — Surface (flags/commands, built-ins hidden), Interactions (version parity, precedence)
  - **Scenario references**: no-runnable-cli.feature (003 Rule blocks): "Root help lists all top-level commands with summaries", "Group help lists its subcommands with summaries", "Commands are listed in alphabetical order", "Framework built-in commands are absent from the listing", "Version flag and version command produce identical output", "Help takes precedence over version"
  - **Risk**: ⚠️ alphabetical order and help-command hiding depend on a cobra package-global (`EnableCommandSorting`) and a cobra idiom (`SetHelpCommand` with a hidden command); pin both with regression tests (plan Risks).

## Phase 2: Executable acceptance [Shared]

- [ ] **T002** [Shared] Make the 003 driving scenarios pass as executable acceptance
  - **Scope**: Add godog step definitions for the three 003 Rule blocks in `features/no-runnable-cli.feature` (Discover the available commands / Read a command's usage before invoking it / Confirm which build of the CLI is running), exercising the assembled root and asserting the rendered output. Remove `@wip` from the passing behavioral scenarios; leave the `@validation` scenarios tagged for validate.
  - **Acceptance criteria**:
    - every non-`@validation` 003 scenario has an executable, passing path: root listing, group listing, empty-set root help, group-immediate-children, alphabetical order, built-ins absent, leaf usage, help-on-unregistered-renders-nothing, help precedence, version parity, version-flag-on-subcommand
    - the empty-command-set and unregistered-path scenarios run without panics
    - `@wip` removed from those behavioral scenarios; the three 003 `@validation` scenarios (deterministic output; no new required registration data; no exit-code/routing leak) keep `@validation @wip`, held out for validate
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: no-runnable-cli.feature — all 003 Rule-block scenarios
