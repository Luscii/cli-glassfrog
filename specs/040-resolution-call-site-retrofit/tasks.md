# Tasks: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, interface-cli.md, features/duplicated-setting-resolution/resolution-call-site-retrofit.feature

---

## Dependency Graph

Phase 1: Token retrofit (1 task, depends on 039 only) [US1/US3]
Phase 2: Base-URL retrofit + presence threading (1 task, depends on 039 only — parallel with Phases 1 and 3) [US1/US2/US3]
Phase 3: Output-selection retrofit + presence threading (1 task, depends on 039 only — parallel with Phases 1 and 2) [US1/US2/US3]

3 tasks total | all three phases parallelizable | Builder: pipeline

All three depend only on 039 (landed). Phases 2 and 3 both edit the read-command `RunE` files (base-URL vs. output plumbing — different lines); that shared-edit surface is a merge-order coordination point, not a blocking dependency.

Story labels: US1 = express each setting's resolution as a composition of shared resolve sources (Maintainer); US2 = the supplied flag honoured by presence (Practitioner); US3 = leave every public output surface unchanged (downstream consumer).

---

## Branching Guidance

**Pipeline mode**: `spec/040-resolution-call-site-retrofit/base` → `spec/040-resolution-call-site-retrofit/task-1`, `task-2`, `task-3`.

Each task carries its existing resolver test suite forward green (the behaviour-preservation contract). Phase 1 is internal to `internal/auth` and rebases cleanly. Phases 2 and 3 both touch `internal/cli` `RunE` files; land one, then rebase the other onto it to reconcile the shared `RunE` edits before merge.

---

## Phase 1: Token retrofit [US1/US3]

- [ ] **T001** [US1] Migrate the token resolver onto `internal/resolve` (env → file), mapping provenance back onto `auth.Resolution`
  - **Scope**: In `internal/auth/resolve.go`, replace the hand-rolled env-short-circuit-then-rcfile chain in `resolve(startDir, homeDir)` with `resolve.Resolve(resolve.FromEnv(getenv, envTokenVar), resolve.FromFile(startDir, homeDir, tokenKey))` (no flag rung, no default). Map `KindEnv→SourceEnvironment`, `KindFile→SourceFile` (Path = `Origin`), `KindNone→SourceNone`; `Token = res.Value`. `auth.Resolve()`'s signature is unchanged; `auth`'s own `getenv`/`getwd`/`userHomeDir` seam feeds the constructors (ADR-4). No cli plumbing changes.
  - **Acceptance criteria**:
    - `GLASSFROG_TOKEN` set → `auth.Resolution{Token, Source: SourceEnvironment}`, no `.glassfrogrc` read
    - A nearest-`.glassfrogrc` `token` (env unset) → `Source: SourceFile` with `Path` set to the resolved path
    - Nothing anywhere → `auth.Resolution{Source: SourceNone}` with a nil error
    - An unreadable/unparseable `.glassfrogrc` returns rcfile's typed error verbatim, no fall-through
    - The `resolve.Resolution` is never formatted/logged before mapping; the token reaches `auth.Resolution.Token` only (secret hygiene preserved)
    - `auth.Resolution` / `auth.Source` fields and members are unchanged; the existing `internal/auth` suite passes green
  - **Dependencies**: None (039 landed)
  - **Plan reference**: Phase 1 — Token retrofit, ADR-1, ADR-4
  - **Scenario references**: resolution-call-site-retrofit.feature: "The token resolver returns the environment value through the composed walk", "No token anywhere remains a normal empty outcome"
  - **Interface references**: interface-spec.md: Token resolver, Provenance → per-domain-type mapping

## Phase 2: Base-URL retrofit + presence threading [US1/US2/US3]

- [ ] **T002** [Shared] Migrate the base-URL resolver onto `internal/resolve` with presence-based flag semantics, threading `Changed()` through assembly and every read command
  - **Scope**: In `internal/apiclient/baseurl.go`, rewrite `ResolveBaseURL` to compose `resolve.Resolve(FromFlags(Flag{Name:"--"+FlagBaseURL, Present: flagPresent, Value: flagValue}), FromEnv(getenv, EnvVarBaseURL), FromFile(startDir, homeDir, baseURLKey), Default(DefaultBaseURL))`; relocate `isUsableURL` to validate the winner when `Kind != KindDefault` (ADR-3), returning `&BaseURLError{Source: res.Provenance.Origin}`; map `KindFlag/KindEnv/KindFile/KindDefault` → the existing `BaseURLSource` members (Path = `Origin` for file). Change signatures (no default-value overload): `ResolveBaseURL(flagValue string, flagPresent bool, startDir, homeDir string)`, `ResolveBaseURLFromOS(flagValue string, flagPresent bool)`, `AssembleFromOS(flagValue string, flagPresent bool)`. Thread the presence bit: every read-command `RunE` passes `cmd.Flags().Changed(apiclient.FlagBaseURL)` alongside its `GetString` value.
  - **Acceptance criteria**:
    - Precedence unchanged: supplied `--base-url` (valid) wins and is used verbatim; unset flag → env → file → default
    - Presence change: `--base-url ""` / `--base-url "  "` (supplied) wins its rung by presence and fails loud with `*BaseURLError{Source:"--base-url"}`, no fall-through to env
    - Whitespace-only `GLASSFROG_BASE_URL` / file value (flag absent) still falls through (unchanged)
    - Present-but-malformed value at any rung fails loud naming that source via `Provenance.Origin`, no fall-through; an unreadable `.glassfrogrc` returns rcfile's typed error before any validation
    - The default backstops the chain and is never re-validated
    - `BaseURL` / `BaseURLSource` / `*BaseURLError` shapes and message text unchanged; `ConnectionContext.Complete()`/`Ready()`/status rendering untouched; the existing `internal/apiclient` suite passes green
    - The signature change forces every caller to compile; a `RunE`-level test asserts presence for supplied / unsupplied / `--flag=` and for the flag on either side of the subcommand
  - **Dependencies**: None (039 landed); coordinate `RunE`-file edits with T003
  - **Plan reference**: Phase 2 — Base-URL retrofit + presence threading, ADR-1, ADR-2, ADR-3, ADR-4
  - **Scenario references**: resolution-call-site-retrofit.feature: "A supplied base-URL flag wins its rung", "A malformed base-URL flag fails loud without consulting lower sources", "An explicitly empty base-URL flag is honoured by presence and fails loud", "The base URL falls through an unsupplied flag and unset environment to the file", "An empty base-URL flag fails loud regardless of its position on the command path"
  - **Interface references**: interface-spec.md: Base-URL resolver, RunE plumbing, Validation at the winner; interface-cli.md: the presence change, Error Communication

## Phase 3: Output-selection retrofit + presence threading [US1/US2/US3]

- [ ] **T003** [Shared] Migrate the output-selection resolver onto `internal/resolve` with presence-based flag semantics, threading `Changed()` through the `resolveSelection` seam and every read command — preserving 035's template branch
  - **Scope**: In `internal/output/selection.go`, make `ResolveSelectionFromOS(flagValue string, flagPresent bool, startDir, homeDir string)` the single composing entry: its precedence walk composes `resolve.Resolve(FromFlags(Flag{Name:"--"+FlagOutput, Present: flagPresent, Value: flagValue}), FromEnv(getenv, EnvVarOutput), FromFile(startDir, homeDir, outputKey), Default(FormatFull.String()))`, returning `output.Selection`. **Fold the existing 6-arg pre-fetched-source pure core `ResolveSelection(flagValue, envValue, fileValue, filePath, fileFound, fileErr)` into `ResolveSelectionFromOS` and remove it** — `resolve.Resolve` fetches env/file via `FromEnv`/`FromFile`, so the pre-fetched-values signature no longer fits (mirrors base URL's composing core, which takes dirs and has no pre-fetched variant). Interpret the winner when `Kind != KindDefault`: a **flag** winner via `classifyFlagSelection` (token → format, `"stdin"` → stdin template, else → template file path — a non-token flag value is NOT an error); an **env/file** winner via `ParseFormat`, returning `&FormatError{Source: res.Provenance.Origin, Value: res.Value}` for a non-token (templates are flag-only); default → `Selection{DefaultFormat}`. Update the `resolveSelection` seam contract to `resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)` across all declarations + the `productionSeam` impl; the `readTemplateSource` seam is untouched. Each read-command `RunE` passes `cmd.Flags().Changed(output.FlagOutput)`. Single `--output` Flag (cobra merges `-o`, so `Origin` is `--output`, matching today's label). Rework `internal/output`'s package tests onto the temp-dir + `getenv`-seam harness (ADR-4).
  - **Acceptance criteria**:
    - Precedence unchanged: supplied `--output` (token) wins; unset flag → `GLASSFROG_OUTPUT` → file → default `full`
    - **035 template branch preserved**: a non-token `--output` value resolves to a `TemplateRef{TemplateFile, Path}`, `"stdin"` to `TemplateRef{TemplateStdin}` — NOT a `*FormatError`; the env/file rungs still accept only the four tokens
    - Presence change: `--output ""` / `-o "  "` (supplied) wins by presence and fails loud (a degenerate empty template selection via 035's classification → `UsageError`), no fall-through
    - Whitespace-only `GLASSFROG_OUTPUT` / file value (flag absent) still falls through to the default `full` (unchanged)
    - A non-token at the **env/file** rung fails loud with `*FormatError` naming that source via `Provenance.Origin`; an unreadable/unparseable `.glassfrogrc` returns rcfile's typed error before any parse, no fall-through to the default
    - Case-insensitive token parse preserved; `output.Selection` / `OutputFormat` / `TemplateRef` / `*FormatError` shapes and message text unchanged; the `readTemplateSource` seam untouched; the existing `internal/output` + `internal/cli` suites pass green
    - The `resolveSelection` seam signature change forces every seam declaration and `RunE` to compile; a `RunE`-level test asserts presence for supplied / unsupplied / `--flag=`
  - **Dependencies**: None (039 landed); coordinate `RunE`-file edits with T002
  - **Plan reference**: Phase 3 — Output-selection retrofit + presence threading, ADR-1, ADR-2, ADR-3, ADR-4
  - **Scenario references**: resolution-call-site-retrofit.feature: "The output format falls through the composed chain to the built-in default", "An unparseable config file on the output walk fails loud without using the default", "A whitespace-only output environment value is treated as absent and falls through"
  - **Interface references**: interface-spec.md: Output-selection resolver, seam contract, RunE plumbing; interface-cli.md: the presence change, `--output` value space, Error Communication

---

## Notes

- **Task breadth is intentional, not a decomposition gap**: Phases 2 and 3 each touch one resolver package plus the `internal/cli` `RunE` sites because the presence-bearing signature change is deliberately overload-free (plan Risks) — every call site must update in the same compiling change. Splitting the resolver internals from the call-site threading would leave a non-compiling intermediate state, so each setting's retrofit is one atomic PR.
- **Validation scenarios** ("No call site re-implements the precedence skeleton", "The public output types and typed errors are unchanged", "The presence change is the only observable behaviour difference") are cross-cutting checks satisfied by the three tasks together rather than by any single task; they are held out for independent verification.
