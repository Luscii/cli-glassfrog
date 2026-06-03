# Validate: Help & Version

**Feature**: 003-help-and-version
**Round**: 1 of 3
**Date**: 2026-06-03
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/no-runnable-cli.feature, PROJECT.md
**Implementation files**: `internal/cli/helpversion.go` (configuration pass), `internal/cli/app.go` (wiring), `internal/cli/version.go` (version source of truth); tests in `internal/cli/helpversion_test.go` and `internal/cli/helpversion_bdd_test.go`

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

**Status**: Pass (9 of 9 scenarios covered)

All driving scenarios trace to identifiable code paths in `configureHelpAndVersion` over cobra's standard rendering, and each has a passing executable acceptance scenario in `no-runnable-cli.feature`.

| Scenario | Status | Implementation |
|---|---|---|
| Root listing shows all top-level commands | ✓ Covered | `helpversion.go` (standard listing) + BDD "Root help lists all top-level commands with summaries" |
| Group help lists its subcommands | ✓ Covered | cobra group listing + BDD "Group help lists its subcommands with summaries" |
| Leaf command usage | ✓ Covered | cobra leaf usage + BDD "Leaf command help shows its path and summary" |
| Version via flag and via command match | ✓ Covered | `helpversion.go:18-20` (`root.Version`, `{{.Version}}\n` template) + `version.go` + unit `TestVersionFlagAndCommandParity` |
| Help requested for an unregistered command | ✓ Covered | `helpversion.go` renders nothing; `dispatch.go` owns the unknown-command error + BDD "Help for an unregistered command renders no usage" |
| Version flag on a non-root command | ✓ Covered | `--version` flag on root only; `dispatch.go` rejects unknown flag + BDD "Version flag on a subcommand is not a version request" |
| Root help on an empty command set | ✓ Covered | BDD "Root help on an empty command set lists no commands" (help output, no Available Commands section, no panic) |
| Group help lists only immediate children | ✓ Covered | cobra immediate-child listing + BDD "Group help lists only immediate children" |
| Both help and version requested | ✓ Covered | cobra help-flag short-circuit (relied on, pinned) + unit `TestHelpPrecedesVersion` + BDD "Help takes precedence over version" |

---

## Acceptance Criteria

**Status**: Pass (both checked tasks, all criteria met)

**T001 — Configure the assembled root for help and version**

| Criterion | Status | Evidence |
|---|---|---|
| `--version` and `version` byte-identical (parity) | ✓ | `TestVersionFlagAndCommandParity` — both read the one `version` var; template emits the bare value |
| Version-unset default is non-empty `0.0.0-dev` | ✓ | `version.go:11` + `TestVersionDefaultPlaceholder` |
| `--help` lists neither `help` nor `completion`; both don't resolve; `--help` flag still renders | ✓ | `TestListingExcludesBuiltinsAlphabetically`, `TestBuiltinCommandsDoNotResolve`, `TestHelpFlagStillRenders` |
| Regression test pins `EnableCommandSorting == true` + alphabetical listing | ✓ | `TestCommandSortingEnabled` + alphabetical assertion in the listing test |
| `--help --version` → help, not version (precedence) | ✓ | `TestHelpPrecedesVersion` |
| `go build ./...` and `go vet ./...` clean | ✓ | Verified clean during validation |

**T002 — Make the 003 driving scenarios pass as executable acceptance**

| Criterion | Status | Evidence |
|---|---|---|
| Every non-`@validation` 003 scenario has an executable, passing path | ✓ | 11 behavioral scenarios pass (`go test ./internal/cli` green) |
| Empty-command-set and unregistered-path scenarios run without panics | ✓ | Both pass; empty set produces help with no Available Commands section |
| `@wip` removed from behavioral scenarios; 3 `@validation` scenarios keep `@validation @wip` | ✓ | No bare `@wip` remains on any behavioral 003 scenario |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-cli.md) | Status | Implementation |
|---|---|---|
| `glassfrog --help` → root listing, alphabetical, summaries | ✓ Conformant | cobra default listing; `EnableCommandSorting` pinned |
| `glassfrog <group> --help` → group listing, immediate children only | ✓ Conformant | cobra group help (verified non-recursive in BDD) |
| `glassfrog <path> --help` → leaf usage | ✓ Conformant | cobra standard usage |
| `glassfrog --version` → bare version value (not `glassfrog version X`) | ✓ Conformant | `SetVersionTemplate("{{.Version}}\n")` — explicitly overrides cobra's default form |
| `glassfrog version` → same string | ✓ Conformant | `version.go` `Fprintln(version)`; parity test confirms byte-equality |
| Built-in `help`/`completion` hidden + non-resolving; `--help` flag kept | ✓ Conformant | `SetHelpCommand({Use:"__help_disabled", Hidden:true})` + `CompletionOptions.DisableDefaultCmd = true` |

---

## Non-Behavior Absence

**Status**: Pass (8 of 8 non-behaviors respected)

| Non-behavior | Status | Inspection |
|---|---|---|
| Must not resolve/route invocations | ✓ Absent | `helpversion.go` only configures the root; no `Find`/`Run`/routing |
| Must not decide/emit exit codes | ✓ Absent | grep confirms no `os.Exit`/exit logic in `helpversion.go` |
| Must not produce structured/JSON output | ✓ Absent | No JSON encoding; cobra text rendering only |
| Must not provide a standalone `help` command | ✓ Absent | The `help` command is replaced by a hidden, non-resolving placeholder |
| Must not introduce new *required* per-command doc data | ✓ Absent | Only `Short` is consumed; registry guard still requires only name + summary |
| Must not emit build metadata in version output | ✓ Absent | Bare version string only |
| Must not mutate the guard-registered/domain command set | ✓ Absent | Only framework-level config (Version, help command, completion); no guard-registered command added/reordered/removed — permitted per spec parenthetical |
| Must not fabricate usage for an unregistered path | ✓ Absent | `glassfrog bogus --help` renders no usage; dispatch owns the error |

---

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral 003 scenarios referenced by the checked tasks have had `@wip` removed and pass. The 3 held-out 003 scenarios correctly retain `@validation @wip`. No behavioral 003 scenario retains a stray `@wip`.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

These three scenarios were held out from the Builder and retain `@validation @wip` (no step definitions). They were verified by independent inspection.

| Scenario | Status | Trace |
|---|---|---|
| Help shows no description beyond declared data | ✓ Satisfied | `configureHelpAndVersion` adds no required fields; `registry.go` mandates only name + summary; help consumes only `Short` and cobra's auto-generated flag list — no new mandatory registration field |
| Listing output is identical across repeated runs | ✓ Satisfied | Alphabetical ordering via `EnableCommandSorting` (pinned); no randomness or time-dependence in the rendering path. Empirically stable: `TestListingExcludesBuiltinsAlphabetically -count=2` green |
| Rendering neither selects exit codes nor routes invocations | ✓ Satisfied | `helpversion.go` contains no exit-code selection and no routing; routing lives in `dispatch.go`, exit codes are deferred to Exit-Code Convention (004) |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 3 validation scenarios are satisfied through independent inspection. Both tasks are complete and checked. The implementation conforms to the specification: the command listing, per-command usage, version parity, built-in suppression, alphabetical ordering, and help-over-version precedence are all present and faithful to the Behavioral Accord, the interface accord, and the non-behaviors.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 003-help-and-version is closed.
