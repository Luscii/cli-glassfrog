# Validate: Output Format Selection

**Feature**: 020-output-format-selection
**Round**: 1 of 3
**Date**: 2026-06-08
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture + ADRs), tasks.md (6 of 6 tasks complete), interface-spec.md, interface-cli.md, features/unconsumable-output/output-format-selection.feature, PROJECT.md
**Implementation files**: `internal/output/format.go` (selection vocabulary + resolver), `internal/cli/root.go` (flag), `internal/cli/clienterror.go` (classify arm), `internal/cli/render.go` (dispatch), `internal/cli/{me,me_roles,me_actions,me_projects}.go` (rewired reads); tests in `internal/output/format_test.go`, `internal/cli/render_test.go`, `internal/cli/output_format_selection_bdd_test.go`

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

**Status**: Pass (9 of 9 implemented scenarios covered)

Every driving scenario is concretized in the feature file and executable (godog, `TestOutputFormatSelectionFeatures` — 9 scenarios / 47 steps green). The three `@validation` scenarios are held out (see below).

| Scenario (spec.md / feature) | Status | Implementation |
|---|---|---|
| `--output json` selects the JSON encoder | ✓ Covered | `internal/cli/render.go:renderResult` structured branch → `output.RenderSuccess` |
| Omitting `--output` selects the default `full` template | ✓ Covered | `internal/output/format.go:ResolveFormat` rung 4 (`DefaultFormat`) → human branch |
| `--output compact` makes the compact rendering reachable | ✓ Covered | `renderResult` human branch → `render.Render(..., FormatCompact, ...)` via `humanFormat` |
| Format matching is case-insensitive | ✓ Covered | `format.go:ParseFormat` (`strings.ToLower(strings.TrimSpace(s))`) |
| An unknown format value fails fast | ✓ Covered | `ResolveFormat` → `*FormatError`; `me.go:reportFormatResolutionError` → UsageError, before assembly/request |
| An invalid lower-precedence value surfaces loudly | ✓ Covered | `ResolveFormat` rung-2 env arm returns `*FormatError{Source: EnvVarOutput}`, no fall-through |
| Flag overrides environment and config file | ✓ Covered | `ResolveFormat` rung-1 short-circuit |
| Config file supplies the format when flag and env absent | ✓ Covered | `ResolveFormat` rung-3 via `rcfile.Resolve(... outputKey)` |
| Unreadable config fails resolution as a usage error | ✓ Covered | `ResolveFormat` surfaces `*rcfile.ReadError`; `classifyClientError` → UsageError |

---

## Acceptance Criteria

**Status**: Pass (6 of 6 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — four-format vocabulary + constants | ✓ Met | `format.go`: `OutputFormat{FormatFull,FormatCompact,FormatJSON,FormatYAML}`, `ParseFormat`, `IsStructured`, `MachineFormat`, `FlagOutput`/`EnvVarOutput`/`outputKey`/`DefaultFormat`; no `cli`/`render` import (build has no cycle); `format_test.go` |
| T002 — `ResolveFormat` + `FormatError` + OS seam | ✓ Met | `ResolveFormat` (pure precedence core), `ResolveFormatFromOS` (binds `os.Getenv` + `rcfile.Resolve`), `*FormatError{Source,Value}`; per-source invalid + rcfile-error + precedence tests |
| T003 — persistent `--output`/`-o` flag | ✓ Met | `root.go:NewRootCommand` `PersistentFlags().StringP(output.FlagOutput, "o", …)`; verified on the built binary (help lists it; parses before and after the subcommand) |
| T004 — `*output.FormatError` → `UsageError` arm | ✓ Met | `clienterror.go` `errors.As` arm; `TestClassifyClientError_FormatErrorIsUsage` (→ exit 2) |
| T005 — shared generic render-dispatch | ✓ Met | `render.go:renderResult[T]` + `humanFormat` + `executor`; `render_test.go` pins routing, decode-target switch, render-failure→RuntimeError(1) with empty stdout |
| T006 — route four reads + fail-fast resolve | ✓ Met | `meSeam.resolveFormat` (+ `productionSeam` impl), `outputFlag` on each config, `reportFormatResolutionError`, resolve-first-then-dispatch in all four reads; default `full` byte-equivalent (pre-020 suites green) |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

`interface-spec.md` (Go API) and `interface-cli.md` (operator surface) both present and checked. Symbol spellings are explicitly a build detail per the accord preamble; the **shapes, signatures, format identifiers, precedence semantics, and literal flag/env/key names** are the contract.

| Surface | Status | Evidence |
|---|---|---|
| `OutputFormat`, `DefaultFormat`, `ParseFormat`, `IsStructured`, `MachineFormat` | ✓ Conformant | `format.go` (members spelled `FormatXxx` to avoid colliding with 018's `Format{JSON,YAML}` — a build-detail choice the accord permits) |
| `ResolveFormat` (pure core), `ResolveFormatFromOS(flagValue, startDir, homeDir)` | ✓ Conformant | `format.go`; `FromOS` signature matches exactly; pure core takes injected sources |
| `FormatError{Source, Value}` | ✓ Conformant | `format.go`; token-free message naming source + value + supported set |
| Constants `FlagOutput`/`EnvVarOutput`/`outputKey` = `"output"`/`"GLASSFROG_OUTPUT"`/`"output"` | ✓ Conformant | `format.go` (literal names — the contract) |
| Render-dispatch (generic, selects decode target + renderer) | ✓ Conformant | `render.go:renderResult[T]` — structured→`json.RawMessage`+`RenderSuccess`, human→`*T`+`render.Render` |
| `classifyClientError` arm `*output.FormatError` → `UsageError` | ✓ Conformant | `clienterror.go` |
| CLI `--output`/`-o` persistent flag, four values, precedence, per-format stdout | ✓ Conformant | `root.go`; verified on the built binary |
| Error Communication (invalid → UsageError 2; rcfile unreadable → 2; render fail → 1; command failure unchanged) | ✓ Conformant | `reportFormatResolutionError` + `renderResult` + `reportClientError` |

**Observations (not findings — behavioral contract holds):**
- The interface example types the dispatch as a single-`io.Writer` signature; the implementation uses `(stdout, stderr)` plus an optional `note func(T) string`. This is a shape elaboration over the example (the preamble marks examples as "shapes, not literal values"); the behavioral contract — decode-target selection, route to the matching renderer, buffer-then-write → `RuntimeError(1)` — is met.
- ADR-3's "single site that imports both `internal/output` and `internal/render`" is realized as intended at the load-bearing level: the two renderer packages remain **non-importing siblings** (no import cycle), and the bridging branch + the `OutputFormat`→`render.Format` mapping live in exactly one site (`render.go:renderResult`/`humanFormat`). The four read files also import `output` (for the `output.FlagOutput` constant and the `output.OutputFormat` seam return type) and `render` (for the resource keys, as since 019) — incidental constant/type/key references, not duplicated bridging logic. Architectural code-organization is outside validate's behavioral scope; noted for transparency.

---

## Non-Behavior Absence

**Status**: Pass (all 8 exclusions respected)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not implement the encoders/templates | ✓ Absent | `render.go` calls `output.RenderSuccess`/`render.Render`; defines no encoder or template |
| Must not render command failures in the selected format | ✓ Absent | `output.RenderError` is never called in `internal/cli` (grep: none); send errors route through the unchanged `reportClientError` — 032's scope |
| Must not define any new process exit code | ✓ Absent | reuses `UsageError(2)`/`RuntimeError(1)`; no `exitcode.go`/`registry.go` change |
| Must not write/create/modify any config file | ✓ Absent | resolver only reads via `rcfile.Resolve`; no write path |
| Must not own `.glassfrogrc` location/walk/parse | ✓ Absent | reuses `rcfile.Resolve(startDir, homeDir, outputKey)` |
| Must not support multiple formats / per-command overrides / profiles | ✓ Absent | one `OutputFormat` per invocation |
| Must not change which fields a command fetches | ✓ Absent | the request is built independent of the format (see validation scenario V2) |
| Must not make an API call to resolve the format | ✓ Absent | `internal/output` imports no `net/http`; `ResolveFormat` is pure (grep: no http/.Do) |

---

## @wip Lifecycle Completion

**Status**: Pass

The 9 scenarios referenced by the (now-checked) implementing tasks have had `@wip` removed and execute green. The 3 `@validation @wip` scenarios correctly remain tagged — they are held out for independent verification and are not referenced by any implementing task. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation through independent inspection)

These scenarios were held out from the implementing pass (`@validation`); they have no registered step definitions and were traced by code inspection, corroborated by the targeted unit tests.

| Scenario | Status | Trace |
|---|---|---|
| Each token dispatches to exactly its renderer | ✓ Satisfied | `render.go:renderResult` branches on `format.MachineFormat()` `ok`: `ok` (json/yaml) → `output.RenderSuccess`; `!ok` (full/compact) → `render.Render` via `humanFormat`. The branch is mutually exclusive, so no format can reach the other renderer. Corroborated by `TestRenderResult_StructuredRoutesToOutputEncoder` (asserts structured output is NOT the human projection) and `TestRenderResult_HumanRoutesToRenderTemplates` (asserts human output is NOT valid JSON). |
| Selection changes rendering only, never the fetched data | ✓ Satisfied | In all four reads the `apiclient.Request` (method, path, `?include`/`?status`) is built **before** and independent of the resolved format; only `renderResult`'s decode target (`json.RawMessage` vs `*T`) and renderer vary, both decoding the same response body. The request — i.e. which fields are fetched — is format-invariant. |
| The precedence chain resolves the first available source | ✓ Satisfied | `format.go:ResolveFormat`: rung 1 flag → rung 2 env → rung 3 `.glassfrogrc` → rung 4 default; a whitespace-only value is skipped, a present-but-invalid value returns `*FormatError` with no fall-through, an rcfile error surfaces. Corroborated by `TestResolveFormat_Precedence`, `TestResolveFormat_PresentButInvalidNamesSource`, `TestResolveFormat_InvalidLowerSourceDoesNotFallThrough`, `TestResolveFormat_SurfacesRcfileError`. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 held-out validation scenarios are satisfied through independent inspection (corroborated by green unit + godog suites). The implementation conforms to its specification: `--output` selects one of four formats from the flag → env → `.glassfrogrc` → default chain, dispatches a successful result to the matching renderer without re-implementing it, fails fast on an invalid selector at any source with the conventional usage code, and respects every declared non-behavior (no failure-rendering, no new exit code, no config write, no field-set change, offline resolution). The interface observations noted above are shape elaborations and code-organization details outside validate's behavioral scope, with the load-bearing contract intact.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 020 is closed — Output Format Selection makes `compact`, `json`, and `yaml` reachable from the command line for the first time. Downstream, Output-Aware Failure Rendering (032) consumes the format 020 selects to render *failures* in that format (the documented interim gap until 032 lands); 029 (User-Defined Template Output) extends the same `internal/render` engine.
