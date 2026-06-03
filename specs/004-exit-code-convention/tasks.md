# Tasks: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/no-runnable-cli.feature

---

## Dependency Graph

Phase 1: Exit-Code Convention (4 tasks, no phase dependencies) — single-phase build

4 tasks total | 1 phase | Builder: pipeline

> Every task is `[Shared]`: exit codes are infrastructure serving all three user scenarios (agent branches on `$?` / CI fails on non-zero / Maintainer inherits the registry) rather than any single one. The four tasks form a linear chain — registry → category → entrypoint → executable acceptance.

---

## Branching Guidance

**Pipeline mode**: `spec/004-exit-code-convention/base` → `spec/004-exit-code-convention/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

---

## Phase 1: Exit-Code Convention [Shared]

- [ ] **T001** [Shared] Add the canonical exit-code registry — `internal/cli/exitcode.go` with the frozen 0–6 code constants and `ExitCode(Outcome) int` (Fail-Safe default 1), plus RED-first `exitcode_test.go`
  - **Scope**: Create `internal/cli/exitcode.go`: named constants pinning the convention (`0` success, `1` internal, `2` usage, `3` API, `4` permission, `5` rate-limit, `6` network-unavailable) and a pure `ExitCode(Outcome) int` mapping the categories that have producers today (Success→0, UsageError→2) with a **default arm returning 1** so any unmapped/future category never yields 0. Operational constants 3–6 are documented as reserved for the future API client. No error inspection. (RuntimeError mapping arrives with its producer in T002.)
  - **Acceptance criteria**:
    - `ExitCode(Success) == 0` and `ExitCode(UsageError) == 2`
    - The default arm returns `1` for any category without an explicit case
    - The seven published code constants have distinct integer values (no two share a code) — pinned by test. (Category↔code one-to-one over the full `Outcome` enum is a later concern: categories for codes 3–6 don't exist until their producer lands — see the `@validation` "Codes and categories are one-to-one" scenario.)
    - No assigned code falls in the shell-reserved range (126, 127, 128+N) — pinned by test
    - The exact values `0/1/2/3/4/5/6` are pinned by a change-detector test (a future renumber breaks loudly)
    - `go build ./...` and `go vet ./...` clean; tests written RED-first then GREEN
  - **Dependencies**: None
  - **Plan reference**: Phase 1 step 1 (Registry); ADR-1 (pure mapper), ADR-2 (published constants, grow enum with producers)
  - **Interface references**: interface-cli.md — Surface (the category↔code table)
  - **Scenario references**: no-runnable-cli.feature: "Codes and categories are one-to-one", "No shell-reserved code is assigned", "Adding a category never renumbers existing codes", "A rate-limited outcome maps to the rate-limit code", "The most specific category determines the code", "Different failure classes carry different codes"

- [ ] **T002** [Shared] Resolve the deferred `RuntimeError` category — extend `Outcome`, reclassify dispatch's runtime-error arm, map it to code 1, update the two deferral tests
  - **Scope**: Add `RuntimeError` to the `Outcome` enum in `dispatch.go` (update `String()`), change `Run`'s default arm so a resolved command whose own action errored returns `RuntimeError, err` instead of `Success, err`, and add the explicit `ExitCode(RuntimeError) == codeInternalError` (1) case to `exitcode.go`. Update the two deferral tests (`TestRun_RuntimeActionError_IsSuccessCategory`, `TestRun_RuntimeError_NotMisclassifiedAsArgError`) to assert `RuntimeError` (error still travels via the return). The arg-rejection (UsageError) and flag-failure arms are untouched.
  - **Acceptance criteria**:
    - `Outcome` includes `RuntimeError`; `String()` renders it; `ExitCode(RuntimeError) == 1`
    - A resolved command whose action returns an error is classified `RuntimeError`, with the error still returned (RED-first: flip the two deferral tests, then GREEN)
    - `UsageError` (unknown command / unknown flag / unexpected arg) and `Success` (clean run, group/root help) classifications are unchanged — existing dispatch unit + BDD assertions still pass
    - `go build ./...`, `go vet ./...`, and `go test ./...` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 step 2 (Category); ADR-3 (RuntimeError → catch-all 1, reclassify at the producer)
  - **Interface references**: interface-cli.md — Surface (code `1` row), Consistency Notes (fulfils 002's deferral)
  - **Scenario references**: no-runnable-cli.feature: "An unexpected internal failure never exits zero"
  - **Risk**: ⚠️ Reclassifying the default arm can ripple — per LEARNINGS, audit *both* `dispatch_test.go` and the BDD harness for `Success` assertions on the runtime-error path before landing.

- [ ] **T003** [Shared] Rewire the entrypoint — `main.go` exits via `ExitCode`, recovers panics to exit 1, and drops the placeholder doc
  - **Scope**: Replace `main.go`'s placeholder mapping with `os.Exit(cli.ExitCode(outcome))`. Add a deferred `recover()` that writes the panic value and stack to stderr (Action Transparency) and `os.Exit(1)` — guaranteeing an unrecovered panic exits `1`, not Go's default status `2` (which collides with `UsageError`). Replace the placeholder doc comment with the real convention reference.
  - **Acceptance criteria**:
    - `main` exits with `cli.ExitCode(outcome)` for every invocation; a successful command exits `0`, an unknown command exits `2`, a resolved command whose action fails exits `1`
    - A panicking command exits `1`, not `2`, and the panic value is written to stderr
    - The placeholder doc comment is replaced; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 1 step 3 (Entrypoint); ADR-1, ADR-4 (panic-recover → 1)
  - **Interface references**: interface-cli.md — Error Communication (never zero on failure, panic safety net)
  - **Scenario references**: no-runnable-cli.feature: "A successful command exits zero", "A help or listing outcome exits zero", "An unknown command exits the usage code", "An internal panic exits one and never collides with the usage code"
  - **Risk**: ⚠️ Go's default panic exit status `2` collides with `UsageError = 2` — the recover is load-bearing. ⚠️ `main`'s old placeholder mapped *all* errors to `1`; usage errors now exit `2` (intended behavior change — call it out in the PR).

- [ ] **T004** [Shared] Make the 004 behavioral scenarios pass as executable acceptance — godog steps for the exit-code Rule blocks, de-`@wip` the passing scenarios
  - **Scope**: Add godog step definitions for the three 004 Rule blocks in `features/no-runnable-cli.feature`, exercising `cli.Run` / `cli.ExitCode` (and the registry directly for the forward-looking category mappings) and asserting the resulting exit code. Remove `@wip` from the passing behavioral scenarios; keep the four `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 004 scenario has an executable, passing path: success→0, help→0, unknown→2, internal-failure→1, panic→1-not-2, different-classes-differ, rate-limit→5, most-specific→4
    - `@wip` removed from those behavioral scenarios; the four 004 `@validation` scenarios (one-to-one, no shell-reserved, never-renumber, no implementation tech) keep `@wip`
    - The full feature suite still passes (no regression to 001/002/003 scenarios); `go test ./...` clean
  - **Dependencies**: T003
  - **Plan reference**: Phase 1 step 4 (Scenarios wiring); Cross-cutting Concerns (testing strategy)
  - **Scenario references**: no-runnable-cli.feature: all 004 Rule-block scenarios
  - **Risk**: ⚠️ The forward-looking categories (rate-limit, permission) have no command producer yet — their scenarios assert the registry mapping directly, not a full `glassfrog` invocation.
