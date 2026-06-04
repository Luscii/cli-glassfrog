# Validate: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Round**: 1 of 3
**Date**: 2026-06-03
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/no-runnable-cli/exit-code-convention.feature, PROJECT.md
**Implementation files**: 4 source (`internal/cli/exitcode.go`, `internal/cli/dispatch.go`, `internal/cli/entrypoint.go`, `main.go`) + 6 test files (`exitcode_test.go`, `entrypoint_test.go`, `smoke_test.go`, `exitcode_bdd_test.go`, plus updates to `dispatch_test.go` and `bdd_test.go`)

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. All 4 tasks complete; full (non-partial) validation. Supplementary evidence: `go test ./...` clean; BDD suite 36 scenarios / 145 steps green; subprocess smoke test confirms the real `os.Exit` status including panic→1.

A designed deferral (operational categories, codes 3–6) is documented under Driving Scenario Coverage and Validation Scenario Results. It is **not a finding** — the spec's own Assumptions and plan ADR-2 explicitly scope it out of this iteration.

---

## Driving Scenario Coverage

**Status**: Pass (4 of 4 producer-backed scenarios covered; 3 operational scenarios deferred by spec design)

The producer-backed driving scenarios trace to clear code paths through the entrypoint (`cli.Main` → `runToExitCode` → `cli.Run` → `cli.ExitCode`):

| Scenario | Status | Implementation |
|---|---|---|
| A successful command exits zero | ✓ Covered | `dispatch.go:Run` (Success) → `exitcode.go:ExitCode` (→0); BDD `no-runnable-cli/exit-code-convention.feature:13` |
| A help/listing outcome exits zero | ✓ Covered | bare group → Success → 0; BDD `:19` |
| An unknown command exits the usage code | ✓ Covered | `dispatch.go:Run` (UsageError) → `ExitCode` (→2); BDD `:28` |
| An unexpected internal failure never exits zero | ✓ Covered | `dispatch.go:Run` default arm (RuntimeError) → `ExitCode` (→1); BDD `:35` |
| An internal panic exits one (never code 2) | ✓ Covered | `entrypoint.go:recoverToCode` (→1); `smoke_test.go` subprocess; BDD `:71` |

**Deferred by design (not a gap)**: the operational driving scenarios — *Different failure classes carry different codes* (2 vs 5), *A rate-limited request exits the rate-limit code* (5 not 3), *The most specific category wins* (4 not 3) — describe codes 3–6. spec.md § Assumptions states "no domain command exists yet in the skeleton, so the API/permission/rate-limit/network categories are defined as convention and registry now, and first exercised when domain commands arrive," and plan ADR-2 codifies "publish the codes as constants now, grow the live `Outcome` enum only as producers arrive." The codes exist as the frozen, test-pinned constants (`codeAPIError=3 … codeNetworkUnavailable=6` in `exitcode.go`); the category→code *mapping arms* and producers arrive with the future API client. The scenarios are correspondingly held out as `@validation @wip`. The implementation delivers exactly what this iteration's artifacts agreed to build.

---

## Acceptance Criteria

**Status**: Pass (all criteria met across 4 checked tasks)

| Task | Criteria status | Evidence |
|---|---|---|
| T001 — registry | ✓ Met | `ExitCode(Success)==0`, `ExitCode(UsageError)==2`, default→1, 7 distinct constants, no shell-reserved, exact values pinned — `exitcode.go` + `exitcode_test.go` (5 tests pass) |
| T002 — RuntimeError | ✓ Met | `RuntimeError` in `Outcome` enum + `String()`; `ExitCode(RuntimeError)==1`; `dispatch.go:Run` default arm reclassified; two deferral tests flipped to assert `RuntimeError`; Success/UsageError arms unchanged |
| T003 — entrypoint | ✓ Met | `cli.Main()` returns `ExitCode(outcome)` (0/2/1 verified); panic recovered to 1 (not 2) with value to stderr (`entrypoint_test.go`); `main()` is `os.Exit(cli.Main())`; placeholder doc replaced |
| T004 — executable acceptance | ✓ Met | 5 producer-backed scenarios de-`@wip`'d and passing via the entrypoint; subprocess smoke test covers `os.Exit` incl. panic→1; 7 `@validation` scenarios held out; full feature suite green |

---

## Interface Contract Conformance

**Status**: Pass (the published category↔code registry and its communication rules are realized)

| Surface element (interface-cli.md) | Status | Evidence |
|---|---|---|
| Code table 0–6 | ✓ Conformant | `exitcode.go` constants `codeSuccess=0 … codeNetworkUnavailable=6` |
| Producer-backed rows (0 Success, 1 internal/RuntimeError, 2 Usage) | ✓ Conformant | `ExitCode` maps `Success→0`, `RuntimeError→1`, `UsageError→2`; default→1 |
| Codes 3–6 "published now but not yet produced" | ✓ Conformant | present as constants, no `Outcome`/`ExitCode` arm yet — matches the stated reservation |
| Producer-classifies model (no error re-derivation) | ✓ Conformant | `ExitCode(Outcome) int` takes only the category; `runToExitCode` discards `Run`'s error |
| Error Communication — never zero on failure | ✓ Conformant | RuntimeError→1, default→1, panic→1 |
| Error Communication — panic safety net (1 not 2, stderr diagnostic) | ✓ Conformant | `recoverToCode` writes value+stack to stderr, returns `codeInternalError` |
| Error Communication — no shell-reserved codes | ✓ Conformant | pinned by `TestExitCodeConstants_NoShellReserved` |
| Extension — single registry site, never renumber | ✓ Conformant | one `ExitCode` switch + one constant block; exact-value change-detector test guards renumbering |

---

## Non-Behavior Absence

**Status**: Pass (no excluded capability present; the one carve-out is the permitted one)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not render/format an outcome message (carve-out: panic crash diagnostic to stderr) | ✓ Absent | `exitcode.go` emits only an `int`; `entrypoint.go` writes to stderr only in `recoverToCode` (the permitted crash diagnostic); no category-message rendering |
| Must not decide which category an outcome belongs to | ✓ Absent | classification lives in `dispatch.go:Run` (the producer, 002's role); `ExitCode` only maps a given `Outcome` |
| Must not retry / back off / recover | ✓ Absent | no retry or backoff logic in any 004 file |
| Must not catch & suppress a failure to force 0 | ✓ Absent | `recoverToCode` maps a panic to 1 (never 0); default arm → 1 |
| Must not reuse a shell-reserved code (126/127/128+N) | ✓ Absent | all codes 0–6; pinned by test |

---

## @wip Lifecycle Completion

**Status**: Pass

- The 5 producer-backed 004 scenarios referenced by checked task T004 had their `@wip` tags removed and now run live (verified in the BDD pass).
- The 7 `@validation` scenarios (operational-category + convention + spec-text invariants) retain `@validation @wip`. These are not referenced by any checked task for behavioral implementation — they are correctly excluded from the lifecycle requirement (held out for independent verification / future producer), consistent with ADR-2.

---

## Validation Scenario Results

**Status**: Satisfied (7 of 7 traced; operational scenarios satisfied at the constant level with behavioral exercise deferred by spec design)

| Scenario (@validation) | Status | Trace |
|---|---|---|
| Codes and categories are one-to-one | ✓ Satisfied | `ExitCode` is a total function (one code per category); `TestExitCodeConstants_Distinct` pins no two codes share a value |
| No shell-reserved code is assigned | ✓ Satisfied | `TestExitCodeConstants_NoShellReserved` (all codes < 126) |
| Adding a category never renumbers existing codes | ✓ Satisfied | constants are fixed; `TestExitCodeConstants_ExactValues` is a change-detector that breaks loudly on any renumber; reserved codes 3–6 demonstrate the "add a category to a pre-reserved code" path |
| Specification names no implementation technology | ✓ Satisfied | spec.md scanned — no language/framework/data-structure names; numeric codes are the external contract |
| Different failure classes carry different codes | ✓ Satisfied (constant level) | `codeUsageError=2` ≠ `codeRateLimited=5`, distinct & pinned; UsageError→2 is live, rate-limit→5 deferred (no producer yet, ADR-2) |
| A rate-limited outcome maps to code 5 not 3 | ✓ Satisfied (constant level) | `codeRateLimited=5` ≠ `codeAPIError=3`, reserved & distinct; mapping arm + producer arrive with the API client |
| The most specific category determines the code | ✓ Satisfied (by design) | `codePermissionError=4` ≠ `codeAPIError=3`; "most specific" selection is the *producer's* responsibility per spec (004 must not decide category), so it correctly lives with the future API client |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 7 `@validation` scenarios are satisfied through inspection (corroborated by a green test suite and a subprocess smoke test). The implementation conforms to the specification as scoped by its own Assumptions and the plan's ADR-2: the producer-backed convention (Success→0, UsageError→2, RuntimeError→1, panic→1) is fully realized and behaviorally exercised, the full 0–6 code convention is published as frozen, test-pinned constants, and the operational categories (3–6) are intentionally deferred until their producer — the future API client — lands. No gaps between what the spec said to build in this iteration and what the implementation delivers.

The single deferral (operational categories) is recorded for transparency, not as a finding: it is authorized by spec.md § Assumptions, plan ADR-2, interface-cli.md ("reserved — future API client"), and the held-out `@validation @wip` scenarios — a consistent, conscious boundary across every artifact.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for the CLI Skeleton problem (001–004) is closed: command registration, argument dispatch, help & version, and the exit-code convention are all implemented and validated. When the future API client lands, it adds the `APIError`/`PermissionError`/`RateLimited`/`NetworkUnavailable` categories and their `ExitCode` arms at the single registry site, taking the already-reserved codes 3–6, and de-`@wip`s the operational `@validation` scenarios.
