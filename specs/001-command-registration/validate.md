# Validate: Command Registration

**Feature**: 001-command-registration
**Round**: 1 of 3
**Date**: 2026-06-03
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, interface-cli.md, features/no-runnable-cli.feature, PROJECT.md
**Implementation files**: 6 (`main.go`; `internal/cli/{doc,root,app,registry,version,roles}.go`) + 2 test files (`registry_test.go`, `bdd_test.go`)

> Independence note: this run is pipeline-mode — the same agent implemented and validated. Structural creator/evaluator separation (Principle 4) is therefore weaker than role-based mode. The @validation scenarios were traced by fresh inspection and are flagged for the developer to weigh accordingly.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass (with advisories) | 3 advisory (non-blocking) |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed; 3 advisory observations (expected framework/entrypoint overlaps, not conformance failures). All 12 behavioral scenarios execute and pass (godog, 45 steps); all 3 @validation scenarios satisfied by inspection.

---

## Driving Scenario Coverage

**Status**: Pass (12 of 12 covered — all executable and passing via godog)

| Scenario | Status | Implementation |
|---|---|---|
| Registering a leaf command makes it known | ✓ Covered | `registry.go:Register` + cobra `Find` |
| Registering a command leaves existing commands untouched | ✓ Covered | `registry.go:Register` (sibling-scoped attach) |
| Registering a group exposes its subcommands by path | ✓ Covered | `registry.go` + `roles.go:newRolesCommand` |
| A bare group name resolves to the group itself | ✓ Covered | cobra `Find([group])` → group node (root.go) |
| A name is unique only within its own group | ✓ Covered | `registry.go` sibling-only collision loop (L70) |
| Groups nest to arbitrary depth | ✓ Covered | `Register` recursion (group-into-group) |
| Duplicate sibling name is rejected | ✓ Covered | `registry.go` collision rule |
| Empty command name is rejected | ✓ Covered | `registry.go` name rule |
| Missing summary is rejected | ✓ Covered | `registry.go` summary rule |
| Leaf command without an action is rejected | ✓ Covered | `registry.go` "neither" rule |
| Group without children is rejected | ✓ Covered | `registry.go` "neither" rule |
| One failed registration prevents the whole CLI from running | ✓ Covered | `MustRegister` panic in `app.go:Assemble`, before `Execute` in `main.go` |

## Acceptance Criteria

**Status**: Pass (5 of 5 checked tasks satisfied)

- **T001** — `go.mod` + cobra v1.10.2 pinned (`go.sum`), `go build ./...` clean, `internal/cli` home present. ✓
- **T002** — `CGO_ENABLED=0 go build` yields a single executable; `glassfrog` prints help, exit 0; root exposed via `NewRootCommand`/`Assemble`. ✓
- **T003** — `Register` attaches valid commands; all five rules reject with a named `*RegistrationError`; `MustRegister` panics before `Execute`; ≥1-child enforced at attach; 10 unit tests. ✓
- **T004** — `glassfrog version`/`roles`/`roles list`/`roles get` resolve and run; explicit wiring in `Assemble`, no package `init`. ✓
- **T005** — 12 non-@validation scenarios executable and passing; failed-registration-no-dispatch covered; 2 doc-check validations confirmed. ✓

## Interface Contract Conformance

**Status**: Pass (both touchpoints conformant)

- **interface-spec.md** — `Register(parent, child) -> error` and `MustRegister(parent, child)` match (`registry.go`). Command definition (name=Use / summary=Short / action=RunE / children) and the "exactly one of action or children" rule are implemented. Every row of the Error Communication table maps to a `*RegistrationError` naming the offending command (duplicate names the conflict + parent; empty name/summary, leaf-without-action, group-without-children all identified). ✓
- **interface-cli.md** — binary is `glassfrog`; registered leaves reachable by path; bare group resolves to the group node and lists children; arbitrary-depth paths resolve. Output format / exit codes are correctly *absent* (deferred to sibling capabilities, as the accord states). ✓

## Non-Behavior Absence

**Status**: Pass (authored code honors all five non-behaviors) — 3 advisory observations

The registration mechanism (`registry.go`) authors none of the excluded concerns: it does not parse arguments, render help, execute actions, or emit exit codes, and there is no dynamic/plugin discovery or post-dispatch registration (static `Assemble` wiring only). The five non-behaviors hold for the code this feature wrote.

Three behaviors *are* exhibited by the assembled binary via the cobra framework and the minimal entrypoint. These are **expected and anticipated** (plan ADR-2 chose cobra precisely because dispatch/help come bundled; the non-behaviors' own "Why" clauses assign them to sibling capabilities) and were necessary to exercise registration end-to-end per T004/T005. They are recorded as advisories, not violations:

| ID | Non-behavior | Observation | Disposition |
|---|---|---|---|
| V-1 | "must not parse arguments / decide which command" | The binary dispatches typed invocations via cobra `Execute`. No dispatch logic was authored here. | Owned by **Argument Dispatch** (sibling). Expected; framework-provided. |
| V-2 | "must not render help / execute a command's action" | `glassfrog roles` prints cobra help; `glassfrog version` runs its `RunE`. Needed to satisfy T004 acceptance. No help-rendering or dispatch-execution logic was authored here. | Owned by **Help & Version** / **Argument Dispatch**. Expected. |
| V-3 | "must not define or emit process exit codes" | `main.go:15` emits `os.Exit(1)` on error (implicit 0 on success). | Minimal entrypoint status only; the standardized scheme is deferred to **Exit-Code Convention** (noted in `main.go` and the plan). Worth confirming the later spec replaces this default. |

These do not block the verdict: the feature's authored surface conforms, and the overlaps fall to specs the architecture already names.

## @wip Lifecycle Completion

**Status**: Pass

The 12 behavioral scenarios referenced by checked tasks (T003–T005) have had `@wip` removed. The 3 `@validation` scenarios retain `@wip` correctly — they are held out for independent verification and are not the Builder's to remove, even where a checked task lists one in its scenario references.

---

## Validation Scenarios (held out from the Builder)

**Status**: Satisfied (3 of 3) — traced by inspection (see independence note above)

| Scenario | Status | Evidence |
|---|---|---|
| Lookup is predictable from registration alone | ✓ Satisfied | `registry.go` + cobra support all three claims: `roles list` resolves by path, `roles` alone resolves to the group node, and enumeration (`root.Commands()`) lists the group. |
| Specification names no implementation technology | ✓ Satisfied | `spec.md` scanned: no language/framework/data-structure choice appears. The only "node" occurrences are "group node" (tree vocabulary), not Node.js. |
| Each non-behavior names its owning capability | ✓ Satisfied | Every entry in `spec.md` § Non-Behaviors names its owner (Argument Dispatch / Help & Version / Exit-Code Convention). |

---

## Verdict: Ready

All five conformance dimensions pass and all three held-out validation scenarios are satisfied. Every driving scenario has an executable, passing code path; every checked task's acceptance criteria are met; both interface contracts conform; the authored registration code honors all non-behaviors; and the `@wip` lifecycle is correct.

The three Non-Behavior advisories (V-1–V-3) are expected framework/entrypoint overlaps that the plan explicitly anticipated and assigned to sibling capabilities — they are not conformance failures and require no fix in this feature. They are surfaced so the **Argument Dispatch**, **Help & Version**, and **Exit-Code Convention** specs formally claim them.

The specification loop is closed: the Builder delivered what the spec promised.

---

## Handoff

Implementation conforms to the specification → **suggest PR review and merge.**

- Advisory for downstream specs: V-1/V-2/V-3 mark where cobra's bundled dispatch/help and the minimal `main` exit currently stand in for not-yet-specified sibling capabilities. The `.score/memory/LEARNINGS.md` entry about cobra's built-in `completion`/`help` commands is related.
- Independence caveat: this was a pipeline-mode self-validation. For higher assurance, a second reviewer (or a role-based re-run) could re-trace the @validation scenarios.
