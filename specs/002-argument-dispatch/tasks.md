# Tasks: Argument Dispatch

**Feature**: 002-argument-dispatch
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/no-runnable-cli.feature

---

## Dependency Graph

Phase 1: Dispatch entry + outcome category (1 task, no phase dependencies) [Shared]
Phase 2: Outcome classification (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: dispatch is infrastructure serving all three user scenarios (route / fail-with-guidance / surface-subcommands) rather than any single one.

---

## Branching Guidance

**Pipeline mode**: `spec/002-argument-dispatch/base` → `spec/002-argument-dispatch/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

---

## Phase 1: Dispatch entry + outcome category [Shared]

- [x] **T001** [Shared] Introduce the `Run` dispatch entry and the `Outcome` category, and route `main` through it — seam + 2-value `Outcome` in `internal/cli/dispatch.go`, `main.go` dispatches via `cli.Run(cli.Assemble(), os.Args[1:])`; existing commands unregressed
  - **Scope**: Add `cli.Run(root, args) (Outcome, error)` that executes the assembled cobra tree and returns a code-free `Outcome` category (`Success` / `UsageError`). Rewire `main.go` to dispatch via `cli.Run(cli.Assemble(), …)` instead of calling `Execute` directly. Classification logic itself is T002 — this task establishes the seam and the type.
  - **Acceptance criteria**:
    - `Run` exists with the signature in interface-spec.md and returns the two-value `Outcome` (`Success` / `UsageError`)
    - `main.go` dispatches through `Run`; the existing commands still work (`glassfrog version`, `glassfrog roles`, `glassfrog roles list`, `glassfrog roles get`) — no regression
    - `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Dispatch entry + outcome category; ADR-2
  - **Interface references**: interface-spec.md — `Run` entry point + `Outcome` category

## Phase 2: Outcome classification [Shared]

- [ ] **T002** [Shared] Derive the outcome category and confirm exact-match + unknown-flag rejection
  - **Scope**: Implement the category derivation — unknown command / unknown flag / unexpected arg → `UsageError`; clean run or group/root help → `Success`. A resolved command's own action error is returned uncategorized (RuntimeError deferred to 004). Confirm cobra's `EnablePrefixMatching` is left `false` (exact match) and unknown-flag rejection is on. Test-first per the constitution.
  - **Acceptance criteria**:
    - Each category is derived correctly from a representative invocation (RED-first unit tests per category)
    - A prefix does not resolve: `ro` does not route to `roles` (regression test pinning the exact-match non-behavior)
    - An unknown flag yields `UsageError` and the command does not run
    - An unknown command yields `UsageError` and names the unrecognized token; a resolved command that runs yields `Success` (its own action error, if any, is returned uncategorized — RuntimeError deferred to 004)
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — Outcome classification; ADR-1 (exact match), ADR-2 (category)
  - **Interface references**: interface-cli.md — resolution + error contract; interface-spec.md — category derivation table
  - **Scenario references**: no-runnable-cli.feature: "A prefix does not resolve to a longer command", "An unknown top-level command fails with guidance", "An unexpected flag is rejected as a usage error"
  - **Risk**: ⚠️ cobra exposes no typed "command not found" — classify `UsageError` at the seams (resolution check + flag-error hook). Pin each category with a test.

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the 002 driving scenarios pass as executable acceptance
  - **Scope**: Add godog step definitions for the Argument Dispatch scenarios in `features/no-runnable-cli.feature` (the three 002 Rule blocks), exercising `Run` against an assembled tree and asserting `(outcome category, output)`. Remove `@wip` from the passing behavioral scenarios.
  - **Acceptance criteria**:
    - Every non-`@validation` 002 scenario (route nested / route top-level / prefix-no-resolve / unknown command / unknown subcommand / unexpected-flag rejected / bare group / empty invocation) has an executable, passing path
    - `@wip` removed from those scenarios; the three 002 `@validation` scenarios keep `@wip` (held out for validate)
    - The three `@validation` scenarios (no routing by abbreviation; no implementation tech; non-behaviors name owners) confirmed against spec.md text
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: no-runnable-cli.feature: all 002 Rule-block scenarios
  - **Risk**: ⚠️ cobra's built-in `help`/`completion` are resolvable commands; if a scenario probes unknown-command behavior near them, account for them. The keep/hide decision is Help & Version's (003).
