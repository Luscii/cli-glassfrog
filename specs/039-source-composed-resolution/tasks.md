# Tasks: Source-Composed Resolution

**Feature**: 039-source-composed-resolution
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/duplicated-setting-resolution/source-composed-resolution.feature

---

## Dependency Graph

Phase 1: Core walk + value-only sources (3 tasks; T001 → {T002, T003}) [Shared/US1]
Phase 2: I/O sources + OS binding (3 tasks, depends on Phase 1) [US2/US3]

6 tasks total | within-phase parallelism only | Builder: pipeline

Story labels: US1 = compose a resolver by listing reusable source constructors; US2 = receive the provenance of the winning value; US3 = keep OS access behind an injectable seam.

---

## Branching Guidance

**Pipeline mode**: `spec/039-source-composed-resolution/base` → `spec/039-source-composed-resolution/task-1`, `spec/039-source-composed-resolution/task-2`, …

The `internal/resolve` package is purely additive — no existing file is edited (the three current resolvers are untouched until 040), so task branches rebase cleanly onto `base`.

---

## Phase 1: Core walk + value-only sources [Shared/US1]

- [x] **T001** [Shared] Define the package's core types — `types.go` (SourceKind/Provenance/Resolution/Source/Flag); stdlib-only; unit tests for String/Found/Kind
  - **Scope**: New `internal/resolve` package; declare `SourceKind` (`KindNone`/`KindFlag`/`KindEnv`/`KindFile`/`KindStdin`/`KindDefault`) with `String()`, `Provenance{Kind, Origin}`, `Resolution{Value, Provenance}` with `Found()`, and the opaque `Source` (kind tag + lazy eval) with `Kind()`. No domain imports.
  - **Acceptance criteria**:
    - `SourceKind.String()` returns the lowercase token for each member; `KindNone` is the zero value
    - `Resolution{}.Found()` is false (zero value = nothing found); a non-`KindNone` provenance reports true
    - `Source.Kind()` returns the constructed kind without evaluating the source
    - Package imports only the standard library (no `auth`/`apiclient`/`output`; `rcfile` arrives in Phase 2)
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — Core walk + value-only sources, ADR-2
  - **Scenario references**: source-composed-resolution.feature: "The resolver is setting-agnostic"
  - **Interface references**: interface-spec.md: Types (SourceKind, Provenance, Resolution, Source)

- [x] **T002** [Shared] [P] Implement the `Resolve` walk — `resolve.go`; 6 scenarios (5 BDD + stdin-guard panic), lazy tripwire unit test
  - **Scope**: `Resolve(sources ...Source) (Resolution, error)` — ordered walk, first-yield-wins, lazy (no lower source evaluated after a yield), none-found outcome (`KindNone`, nil error), abort-and-surface on a source error (no fall-through), and the at-most-one-`Stdin`-source guard.
  - **Acceptance criteria**:
    - Sources evaluate in argument order; the first that yields returns its value + provenance
    - A source that yields nothing is skipped; once a source yields, no lower source's `eval` runs (assert via a tripwire source that fails the test if evaluated)
    - No source and no `Default` → `Resolution{Provenance:{Kind:KindNone}}` with nil error
    - A source that returns an error aborts the walk and returns that error verbatim (no fall-through)
    - More than one source with `Kind: KindStdin` panics with a message naming the misuse; the panic fires before any source is evaluated (stream not drained)
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 — Core walk + value-only sources, ADR-2, ADR-5
  - **Scenario references**: source-composed-resolution.feature: "The highest-precedence source wins", "Resolution falls through empty sources to the environment", "The trailing default backstops an empty chain", "An empty chain with no default yields a valid empty result", "Composing more than one stdin source fails as a wiring error"
  - **Interface references**: interface-spec.md: Functions (Resolve), Error Communication

- [x] **T003** [US1] [P] Implement the value-only source constructors — `sources.go` (FromFlags/FromEnv/Default); alias-walk scenario + unit tests
  - **Scope**: `FromFlags(...Flag)` (presence-based yield per `Flag.Present` — yields even when `Value` is empty; walks aliases in order; `Provenance.Origin` = winning flag `Name`), `FromEnv(lookup, names...)` (first non-empty after trim; walks names; `Origin` = winning name), `Default(value)` (always yields, `Kind: KindDefault`).
  - **Acceptance criteria**:
    - A `Present` flag with an empty `Value` yields (presence-based), and reports its `Name` as `Origin`
    - `FromFlags` over `["--output","-o"]` with only `-o` present yields `-o`'s value with `Origin` `-o`
    - `FromEnv` skips empty/whitespace-only values and yields the first non-empty name, reporting that name as `Origin`
    - `Default(v)` yields `v` with `Provenance{Kind:KindDefault}` and empty `Origin`
  - **Dependencies**: T001
  - **Plan reference**: Phase 1 — Core walk + value-only sources, ADR-2, ADR-4
  - **Scenario references**: source-composed-resolution.feature: "A list-valued flag source yields from its first present alias"
  - **Interface references**: interface-spec.md: Functions (FromFlags, FromEnv, Default), Flag type

## Phase 2: I/O sources + OS binding [US2/US3]

- [x] **T004** [US2] [P] Implement the file source — `file.go` (delegates to `rcfile.Resolve`); malformed-file scenario + unit tests
  - **Scope**: `FromFile(startDir, homeDir, key string) Source` — delegates the nearest-wins walk to `rcfile.Resolve(startDir, homeDir, key)`; yields the value with `Provenance{Kind:KindFile, Origin: <resolved path>}`; surfaces `rcfile`'s typed `*ReadError`/`*FormatError` verbatim; a missing or key-less file does not yield. Adds the `internal/rcfile` import.
  - **Acceptance criteria**:
    - A present `.glassfrogrc` key yields its value with `Origin` set to the resolved file path
    - A missing or key-less file yields nothing (the walk continues in `Resolve`)
    - An unreadable/unparseable file returns the verbatim `rcfile` typed error; under `Resolve` this aborts the walk with no fall-through
    - No second parser is introduced — the walk goes through `rcfile.Resolve`
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — I/O sources + OS binding, ADR-3 (Origin reproduces the file-path label)
  - **Scenario references**: source-composed-resolution.feature: "A malformed config file fails loud naming the file", "Provenance reproduces the existing source labels"
  - **Interface references**: interface-spec.md: Functions (FromFile), Consistency Notes (rcfile delegation)

- [x] **T005** [US3] [P] Implement the stdin source — `stdin.go` (FromStdin + bounded reader/maxStdinBytes); read-failure scenario + TTY/empty/bound unit tests
  - **Scope**: `FromStdin(read func() (string, error), isTTY bool) Source` — yields trimmed piped input when `isTTY` is false and the content is non-empty; never reads on a TTY; bounds the read (`maxStdinBytes` constant); a read failure returns the error; `Provenance{Kind:KindStdin}` with empty `Origin`.
  - **Acceptance criteria**:
    - With `isTTY` false and non-empty piped content, yields the trimmed content with `Kind: KindStdin`
    - With `isTTY` true, never invokes `read` and yields nothing
    - Empty/whitespace-only piped content yields nothing
    - A failing `read` returns the error verbatim; under `Resolve` it aborts resolution the same way a config-file failure does (uniform handling, not a uniform type)
    - The read is bounded by `maxStdinBytes`
    - Piped input that exceeds `maxStdinBytes` returns an error (no silent truncation) — never a truncated value (Constitution VI)
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — I/O sources + OS binding, ADR-4
  - **Scenario references**: source-composed-resolution.feature: "A stdin read failure is surfaced uniformly"
  - **Interface references**: interface-spec.md: Functions (FromStdin)

- [x] **T006** [US3] Implement the thin OS-binding helpers — `osbinding.go` (OSRoots/EnvFromOS/StdinFromOS); only file importing `os`/`term`; parity unit tests
  - **Scope**: `OSRoots() (startDir, homeDir string, err error)` (real `os.Getwd` — error if undeterminable; `os.UserHomeDir` — `""` on failure, drop the home fallback), `EnvFromOS(names...)` = `FromEnv(os.Getenv, names...)`, `StdinFromOS()` = `FromStdin` bound to a bounded `os.Stdin` reader + `term.IsTerminal(int(os.Stdin.Fd()))`. Only this file imports `os`/`term`.
  - **Acceptance criteria**:
    - `OSRoots` returns the working directory, returns an error when it cannot be determined, and yields `homeDir == ""` (not an error) when the home directory cannot be determined
    - `EnvFromOS` and `StdinFromOS` produce sources behaviorally identical to their pure constructors over real OS access
    - The pure constructors (T003/T005) carry no `os`/`term` dependency — only the binding helpers do
  - **Dependencies**: T003, T005
  - **Plan reference**: Phase 2 — I/O sources + OS binding, ADR-4
  - **Scenario references**: source-composed-resolution.feature: "A stdin read failure is surfaced uniformly" (seam exercised via the injected reader)
  - **Interface references**: interface-spec.md: OS binding (OSRoots, EnvFromOS, StdinFromOS)
