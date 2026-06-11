# Validate: Source-Composed Resolution

**Feature**: 039-source-composed-resolution
**Round**: 1 of 3
**Date**: 2026-06-11
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, features/duplicated-setting-resolution/source-composed-resolution.feature, PROJECT.md
**Implementation files**: 6 source + 2 test in `internal/resolve/` (types.go, resolve.go, sources.go, file.go, stdin.go, osbinding.go; resolve_test.go, resolve_bdd_test.go)

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

**Total**: 5 dimensions checked, 5 passed, 3 validation scenarios satisfied, 0 findings.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All driving scenarios are concretized in the feature file, had `@wip` removed by the Builder, and pass under `go test ./internal/resolve/` (8 scenarios / 41 steps, all green).

| Scenario | Status | Implementation |
|---|---|---|
| The highest-precedence source wins | ✓ Covered | resolve.go:Resolve walk + sources.go:FromFlags |
| Resolution falls through empty sources to the environment | ✓ Covered | resolve.go:Resolve (empty source → continue) |
| The trailing default backstops an empty chain | ✓ Covered | sources.go:Default (always yields) |
| An empty chain with no default yields a valid empty result | ✓ Covered | resolve.go:Resolve returns `Provenance{Kind:KindNone}`, nil error |
| A list-valued flag source yields from its first present alias | ✓ Covered | sources.go:FromFlags (in-order alias walk, Origin = winning Name) |
| Composing more than one stdin source fails as a wiring error | ✓ Covered | resolve.go:Resolve Stdin-count guard panic (fires before any eval) |
| A malformed config file fails loud naming the file | ✓ Covered | file.go:FromFile → rcfile.Resolve; typed error surfaced, no fall-through |
| A stdin read failure is surfaced uniformly | ✓ Covered | stdin.go:FromStdin returns read error verbatim; resolve.go aborts walk |

---

## Acceptance Criteria

**Status**: Pass (all 6 tasks checked; every criterion traced)

| Task | Status | Evidence |
|---|---|---|
| T001 — core types | ✓ Met | types.go: `SourceKind.String()` lowercase tokens; `KindNone` is zero value (`iota`); `Resolution{}.Found()` false / non-`KindNone` true; `Source.Kind()` reads tag without eval; file imports no stdlib or domain pkg. Unit tests: TestSourceKindString, TestResolutionFound, TestSourceKindWithoutEvaluating |
| T002 — Resolve walk | ✓ Met | resolve.go: argument-order walk, first-yield-wins; skip-empty + lazy short-circuit (TestResolveLazyShortCircuit tripwire); `KindNone`/nil on none-found; verbatim error abort no fall-through; >1 Stdin panics before eval (TestResolvePanicsOnMultipleStdin asserts message + non-drain) |
| T003 — value-only sources | ✓ Met | sources.go: present empty-value flag yields w/ Origin=Name; alias walk; FromEnv trim-skip + verbatim value + Origin=name; Default yields w/ `KindDefault`, empty Origin. Unit tests cover each |
| T004 — file source | ✓ Met | file.go delegates to `rcfile.Resolve` (no second parser); yields w/ Origin=resolved path; missing/key-less no yield; unreadable/unparseable surfaces verbatim `*rcfile.FormatError`/`*ReadError`, aborts. Tests: TestFromFile* |
| T005 — stdin source | ✓ Met | stdin.go: trimmed piped yield on `isTTY` false; never reads on TTY; empty/whitespace no yield; read failure verbatim; `readBoundedStdin`/`maxStdinBytes` bound; over-cap errs (no silent truncation). Tests: TestFromStdin*, TestReadBoundedStdinUnderAndOverCap |
| T006 — OS binding | ✓ Met | osbinding.go: `OSRoots` returns getwd (error if undeterminable) + home (`""` on failure, no hard fail); `EnvFromOS`/`StdinFromOS` wrap pure constructors; only this file imports `os`/`term` (confirmed by per-file import scan). Tests: TestEnvFromOSMatchesPureConstructor, TestOSRootsReturnsWorkingDirectory |

---

## Interface Contract Conformance

**Status**: Pass (entire exported surface conformant)

`go doc ./internal/resolve` confirms every identifier in interface-spec.md § Surface exists with the specified shape:

| Surface element | Status |
|---|---|
| `SourceKind` + members (`KindNone`…`KindDefault`) + `String()` | ✓ Conformant |
| `Provenance{Kind, Origin}` | ✓ Conformant |
| `Resolution{Value, Provenance}` + `Found()` | ✓ Conformant |
| `Source` (opaque) + `Kind()` | ✓ Conformant |
| `Flag{Name, Present, Value}` | ✓ Conformant |
| `Resolve(...Source) (Resolution, error)` (panics on >1 Stdin) | ✓ Conformant |
| `FromFlags` / `FromEnv` / `FromFile` / `FromStdin` / `Default` | ✓ Conformant (signatures match) |
| `OSRoots` / `EnvFromOS` / `StdinFromOS` (thin OS binding) | ✓ Conformant |
| Package imports `internal/rcfile` + `golang.org/x/term` + stdlib only; no domain package | ✓ Conformant |
| `maxStdinBytes` package constant | ✓ Conformant (unexported, per the "Builder finalizes within the shape" note) |

Error communication (interface § Error Communication) matches: none-found → nil error; `FromFile` errors surface verbatim rcfile typed errors; stdin read/over-cap errs; >1 Stdin panics.

---

## Non-Behavior Absence

**Status**: Pass (all 5 exclusions confirmed absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not validate the resolved value | ✓ Absent | No `url.Parse`/`ParseFormat`/`regexp`/membership check in package source; `Resolve` returns the raw winner |
| Must not modify/migrate/wrap the 3 existing resolution sites | ✓ Absent | HEAD touches only new `internal/resolve/` files + feature/tasks/STATUS; no edit under `internal/auth`, `internal/apiclient`, `internal/output` |
| Must not write a file/env var or make a network call | ✓ Absent | No `WriteFile`/`os.Setenv`/`os.Create`/`net/http`/`net.Dial` in package source |
| Must not prompt or solicit input interactively | ✓ Absent | No `fmt.Print*`/`ReadString`/`ReadPassword`/`bufio` prompting; `FromStdin` reads piped only and is guarded by `isTTY` |
| Must not emit any resolved value into its own output | ✓ Absent | No print/log in package; the only `fmt` uses are the static Stdin-guard panic message, the byte-limit error (no value), and `OSRoots`' wrapped getwd error (no resolved value) |

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced independently to the implementation)

| Scenario | Status | Trace |
|---|---|---|
| The resolver names no concrete setting | ✓ Satisfied | `go doc` exported surface (types, constructors, the `Kind*` consts) carries no `token`/`base_url`/`output`/`format` identifier; flag names, env names, the `.glassfrogrc` key and default are all caller-supplied parameters. Setting names appear only in explanatory doc-comment prose, which the scenario explicitly permits |
| Provenance reproduces the existing source labels | ✓ Satisfied | `Provenance.Origin` carries the winning flag `Name` (caller passes `--base-url`), the winning env `name` (caller passes `GLASSFROG_BASE_URL`), and the resolved file path — a caller can phrase all three current labels with no setting knowledge in the resolver. Unit tests assert Origin for flag/env/file kinds |
| No resolved value leaks into diagnostics | ✓ Satisfied | The resolver emits no diagnostics; it returns values by field and never formats `Value` into a message (secret hygiene for the token setting). Confirmed by the print/log grep above |

These three scenarios retain `@wip` in the feature file by design — they are the held-out independent-verification set (Principle 4), not Builder work. Traced here by inspection.

---

## @wip Lifecycle Completion

**Status**: Pass

8 driving/error/edge scenarios had `@wip` removed and pass the suite. The 3 remaining `@wip` tags all carry `@validation` and are correctly held out (matching the 008/base-url precedent). No non-`@validation` scenario referenced by a checked task still carries `@wip`.

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 validation scenarios are satisfied through independent inspection. The implementation delivers exactly the setting-agnostic composable resolver the spec promised: ordered first-yield-wins walk, lazy evaluation, code-free `Resolution`/`Provenance`, the five source constructors, the at-most-one-Stdin panic guard, the injected OS seam, and verbatim resolution-error surfacing — with value validation correctly deferred to the caller and no setting-specific knowledge baked in. It is behavior-neutral toward the three existing resolution sites (040's retrofit remains untouched).

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 039 is closed; the call-site retrofit (relocating `isUsableURL`/`ParseFormat` to their sites and validating the resolved value there) is **040 (Resolution Call-Site Retrofit)**.
