# Validate: Argument Dispatch

**Feature**: 002-argument-dispatch
**Round**: 1 of 3
**Date**: 2026-06-03
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, interface-spec.md, features/no-runnable-cli.feature, PROJECT.md
**Implementation files**: 4 — `internal/cli/dispatch.go` (Run + Outcome), `main.go` (entrypoint dispatch), `internal/cli/dispatch_test.go` (10 unit tests), `internal/cli/dispatch_bdd_test.go` + `internal/cli/bdd_test.go` (godog steps)

> Note: context-engineering / self-verification / three-tier-boundaries references and `guardian-agent.md` are not deployed in this Score install — applied skill-specific checks only; reduced Guardian-character consistency, not a blocked validation.

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All eight driving scenarios are referenced by the checked T003 and have identifiable, executable code paths (each is an unskipped godog scenario that passes, plus a corresponding unit test). Traced against `internal/cli/dispatch.go:Run`:

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| Route a nested leaf command | ✓ Covered | `Run` default case → cobra executes `roles list`; `dispatch_test.go:TestRun_NestedLeafRoutes_Success` |
| Route a top-level leaf command | ✓ Covered | `Run` default case → `version`; `TestRun_TopLevelLeafRoutes_Success` |
| Bare group surfaces its subcommands | ✓ Covered | `Run` `err==nil && !Runnable() && len(leftover)==0` → Success; help listing via cobra; `TestRun_BareGroupResolvesToHelp_Success` |
| Unknown top-level command | ✓ Covered | `Run` `err!=nil && !Runnable()` → UsageError; cobra emits `unknown command "rolez"` + `Run 'glassfrog --help' …`; `TestRun_UnknownTopLevelCommand_UsageError` |
| Unknown subcommand under a known group | ✓ Covered | `Run` nested-swallow branch synthesizes `unknown command "lst" for "glassfrog roles"` → UsageError; `TestRun_UnknownSubcommandUnderGroup_UsageError` |
| Unexpected flag is rejected | ✓ Covered | `Run` `flagFailed` (SetFlagErrorFunc) → UsageError, action never runs; `TestRun_UnknownFlag_UsageError_CommandDoesNotRun` |
| A prefix does not resolve to a longer command | ✓ Covered | `EnablePrefixMatching` left false → `ro` is unknown command; `TestRun_PrefixDoesNotResolve_UsageError` + `TestExactMatch_PrefixMatchingDisabled` |
| Empty invocation resolves to root help | ✓ Covered | `Run` `err==nil && !Runnable() && len(leftover)==0` on root → Success; `TestRun_EmptyInvocationResolvesToRoot_Success` |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks checked, all criteria met)

| Task | Criteria | Evidence |
|---|---|---|
| T001 | `Run` with the interface signature returning two-value `Outcome`; `main.go` dispatches through `Run`; existing commands unregressed; build+vet clean | `dispatch.go:74` `Run(root, args) (Outcome, error)`; `dispatch.go:21-31` `Outcome{Success,UsageError}`; `main.go:18` `cli.Run(cli.Assemble(), os.Args[1:])`; suite green, build/vet clean |
| T002 | Each category derived (RED-first per category); prefix no-resolve regression; unknown flag → UsageError + no run; unknown command names token, resolved → Success, action error uncategorized | `dispatch.go:88-117` four-arm classification; 10 tests in `dispatch_test.go` incl. `TestRun_RuntimeActionError_IsSuccessCategory` (action error → Success + err) and `TestExactMatch_PrefixMatchingDisabled` |
| T003 | Every non-`@validation` 002 scenario passing; `@wip` removed from them; 3 `@validation` keep `@wip`; validation scenarios confirmed against spec.md | godog: 20/20 scenarios, 80 steps pass; 8 behavioral 002 scenarios de-`@wip`'d; 3 `@validation` 002 scenarios retain `@wip` (verified in feature file); spec-text confirmation below |

---

## Interface Contract Conformance

**Status**: Pass (both accords conformant)

| Surface (interface-spec.md / interface-cli.md) | Status | Implementation |
|---|---|---|
| `Run(root, args) -> (Outcome, error)` entry point | ✓ Conformant | `dispatch.go:74` exact signature; single dispatch boundary called from `main.go` |
| `Outcome` = `{Success, UsageError}`, code-free | ✓ Conformant | `dispatch.go:21-32`; no exit code, no rendered message in the type |
| Resolved-action error via returned `error`; RuntimeError deferred | ✓ Conformant | `Run` default arm returns `Success, err`; `TestRun_RuntimeActionError_IsSuccessCategory` pins it |
| Exact leaf / bare group / no-tokens → Success; unknown token / unknown flag → UsageError, no run | ✓ Conformant | classification arms in `Run`; covered by driving-scenario tests above |
| Exact matching, no prefix/abbreviation | ✓ Conformant | `EnablePrefixMatching` false, pinned by test |
| Error to stderr; best-effort "did you mean" suggestion | ✓ Conformant | cobra writes unknown-command error + `Did you mean …` + help pointer to stderr (observed in probe + godog `points the caller to help`) |

---

## Non-Behavior Absence

**Status**: Pass (5 of 5 exclusions absent from the implementation)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not render help text / listing itself | ✓ Absent | `Run` delegates help to cobra; it only returns a category and (nested case) an error string naming a token — no usage/listing rendering of its own |
| Must not define or emit process exit codes | ✓ Absent | No `os.Exit`/exit codes in `dispatch.go`. `main.go`'s `os.Exit(1)` is the entrypoint's documented placeholder mapping (plan § Cross-cutting / interface-spec.md), explicitly reserved for Exit-Code Convention (004) — outside the dispatch capability |
| Must not register, add, or modify commands | ✓ Absent | `Run` only reads the assembled tree (`SetArgs`, `SetFlagErrorFunc`, `Execute`); it adds/removes no commands |
| Must not match by prefix or abbreviation | ✓ Absent | `cobra.EnablePrefixMatching` left false; `TestExactMatch_PrefixMatchingDisabled` pins the package-global |
| Must not perform a command's actual work | ✓ Absent | `Run` hands control to the resolved command via cobra; no API/governance calls |

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3)

These three `@validation` scenarios were held out from the Builder (kept `@wip`) and assert properties of the **specification text**, so each is traced against `spec.md` (the artifact under test) rather than a code path.

| Scenario (spec.md § Validation Scenarios) | Status | Trace |
|---|---|---|
| No routing depends on abbreviation | ✓ Satisfied | Every routing scenario names a full, exact path (`roles list`, `version`, `glassfrog roles`, `glassfrog rolez`, `roles lst`, `roles list --bogus`, `glassfrog`); the sole prefix mention (`ro list`) is the negative edge case asserting `ro` does **not** resolve. Implementation reinforces this (`EnablePrefixMatching` false). |
| Each non-behavior names its owning capability | ✓ Satisfied | NB1→Help & Version, NB2→Exit-Code Convention, NB3→Command Registration name a sibling explicitly; NB5 names "its own spec's concern" (the resolved command's capability); NB4 (no prefix matching) is a self-constraint whose concern dispatch itself owns (stated in § Matching). No excluded concern is orphaned. (Transparency note: NB4 is the only exclusion not delegating to a *sibling* — it is owned by dispatch itself, so it satisfies the scenario's intent that every excluded concern has a clear owner.) |
| Specification names no implementation technology | ✓ Satisfied | Scan of spec.md found no language/framework/library/data-structure names (no `Go`, `cobra`, `godog`, `struct`, `map[]`, etc.); the closest term, "command framework," is a generic capability reference, not a concrete technology. |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 held-out validation scenarios are satisfied. The implementation conforms to the specification: every driving scenario has an executable, passing code path; the `Run`/`Outcome` accords match the code exactly; every spec non-behavior is absent from the implementation; the `@wip` lifecycle is complete (8 behavioral 002 scenarios live, 3 `@validation` held out); and exact matching is pinned against the cobra package-global.

One forward-looking note (not a finding): `main.go` maps the `Outcome` to a process exit code minimally (`success → 0, error → non-zero`). This is the documented placeholder the spec and plan reserve for **Exit-Code Convention (004)** — which is also where the deferred `RuntimeError` category lands. 004 should replace this mapping rather than discover it. The nested-group unknown-subcommand path produces `UsageError` via a *synthesized* (non-cobra) error, recorded in `.score/memory/LEARNINGS.md` for 003/004 to account for.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 002-argument-dispatch is closed.
