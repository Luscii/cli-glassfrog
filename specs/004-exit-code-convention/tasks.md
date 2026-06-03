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

- [x] **T001** [Shared] Add the canonical exit-code registry — `internal/cli/exitcode.go` with the frozen 0–6 code constants and `ExitCode(Outcome) int` (Fail-Safe default 1), plus RED-first `exitcode_test.go` — 0 scenarios (constant-level pins for the held-out @validation scenarios), RED→GREEN
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

- [x] **T002** [Shared] Resolve the deferred `RuntimeError` category — extend `Outcome`, reclassify dispatch's runtime-error arm, map it to code 1, update the two deferral tests — 0 scenarios; flipped the two deferral tests RED→GREEN (renamed `…_IsSuccessCategory` → `…_IsRuntimeErrorCategory` to match its new expectation), added a `RuntimeError.String()` pin
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

- [x] **T003** [Shared] Rewire the entrypoint — `main.go` exits via `ExitCode`, recovers panics to exit 1, and drops the placeholder doc — 0 scenarios (scenario bindings land in T004); recover+map extracted to `runToExitCode` so in-process tests/BDD drive it without mirroring the recover. Behavior change: usage errors now exit 2 (was 1 under the placeholder)
  - **Scope**: Extract a testable entrypoint `cli.Main() int` that dispatches via `cli.Run`, maps the outcome through `cli.ExitCode`, and recovers a panic to return `1` (writing the panic value/stack to stderr for Action Transparency) — guaranteeing an unrecovered panic yields `1`, not Go's default status `2` (which collides with `UsageError`). `main.go` reduces to `os.Exit(cli.Main())` and drops the placeholder doc comment. Keeping the logic in `Main()` rather than inline in `main` lets T004 exercise the exit-code and panic paths in-process, since `os.Exit` would otherwise terminate the test binary.
  - **Acceptance criteria**:
    - `cli.Main()` returns `cli.ExitCode(outcome)` for every invocation; a successful command yields `0`, an unknown command `2`, a resolved command whose action fails `1`
    - A panic is recovered inside `Main()` and yields `1`, not `2`, with the panic value written to stderr
    - `main()` is a thin `os.Exit(cli.Main())`; the placeholder doc comment is replaced; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 1 step 3 (Entrypoint); ADR-1, ADR-4 (panic-recover → 1)
  - **Interface references**: interface-cli.md — Error Communication (never zero on failure, panic safety net)
  - **Scenario references**: no-runnable-cli.feature: "A successful command exits zero", "A help or listing outcome exits zero", "An unknown command exits the usage code", "An internal panic exits one and never collides with the usage code"
  - **Risk**: ⚠️ Go's default panic exit status `2` collides with `UsageError = 2` — the recover is load-bearing. ⚠️ `main`'s old placeholder mapped *all* errors to `1`; usage errors now exit `2` (intended behavior change — call it out in the PR).

- [ ] **T004** [Shared] Make the 004 behavioral scenarios pass as executable acceptance — godog steps for the exit-code Rule blocks, de-`@wip` the passing scenarios
  - **Scope**: Add godog step definitions for the producer-backed 004 behavioral scenarios in `features/no-runnable-cli.feature`, exercising the extracted `cli.Main()` (and `cli.Run` / `cli.ExitCode`) in-process and asserting the resulting exit code; add a subprocess smoke test that runs the compiled CLI to validate the real `main()` → `os.Exit(cli.Main())` wiring and the panic→1 path end-to-end. Remove `@wip` from the producer-backed behavioral scenarios. The operational-category scenarios stay `@validation @wip`: their `Outcome` values have no producer yet (ADR-2), so they are verified against the published code constants (T001) and exercised behaviorally only when the API-client producer lands.
  - **Acceptance criteria**:
    - Every producer-backed behavioral scenario passes via `cli.Main()`: success→0, help→0, unknown→2, internal-failure→1, panic→1-not-2
    - A subprocess smoke test confirms the built CLI exits with the mapped code (including panic→1) — validating the `os.Exit(cli.Main())` wiring that in-process tests cannot observe
    - `@wip` removed from the producer-backed scenarios; the operational-category scenarios (different-classes, rate-limit, most-specific) plus the existing four `@validation` scenarios keep `@wip`
    - The full feature suite still passes (no regression to 001/002/003 scenarios); `go test ./...` clean
  - **Dependencies**: T003
  - **Plan reference**: Phase 1 step 4 (Scenarios wiring); ADR-2 (operational categories deferred until a producer), ADR-4 (panic→1); Cross-cutting Concerns (testing strategy)
  - **Scenario references**: no-runnable-cli.feature: the producer-backed 004 scenarios (de-`@wip`'d) plus the `@validation` operational-category and convention scenarios (held out)
  - **Risk**: ⚠️ The operational categories (API/permission/rate-limit/network) have no producer yet — don't call `ExitCode` with `Outcome` values that don't exist; those scenarios stay `@validation` and are pinned at the constant level by T001. ⚠️ The panic→1 and process-exit paths can't be observed through `cli.Run`/`cli.ExitCode` alone (they live behind `os.Exit`); cover them via the extracted `cli.Main()` plus a subprocess smoke test.
