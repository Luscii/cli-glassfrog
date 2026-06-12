# Validate: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Round**: 1 of 3
**Date**: 2026-06-12
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-spec.md, interface-cli.md, features/duplicated-setting-resolution/resolution-call-site-retrofit.feature, PROJECT.md
**Implementation files**: internal/auth/resolve.go; internal/apiclient/baseurl.go, assemble.go; internal/output/selection.go; internal/cli/* (16 read-command + seam files threading presence) — all 3 tasks checked complete

> Note: `agents/guardian-agent.md` was empty/absent — validated against SKILL.md alone (reduced character consistency, not a blocked skill).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 3) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings. One non-blocking observation (O-1) recorded below.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 scenarios covered)

Every driving scenario maps to an identifiable code path, each pinned by a passing BDD or unit test (suites cached green at validation time).

| Scenario | Status | Implementation |
|---|---|---|
| Token resolved from the environment | ✓ Covered | `internal/auth/resolve.go:resolve` → `KindEnv→SourceEnvironment`; `auth/resolution_retrofit_bdd_test.go` (@token) |
| Base URL resolved from a supplied flag | ✓ Covered | `internal/apiclient/baseurl.go:ResolveBaseURL` `KindFlag` arm; `apiclient/resolution_retrofit_bdd_test.go` (@base-url) |
| Output selection falls through to the built-in default | ✓ Covered | `internal/output/selection.go:ResolveSelectionFromOS` default arm; `output/resolution_retrofit_bdd_test.go` (@output) |
| Base URL falls through flag→env→file→default | ✓ Covered | `ResolveBaseURL` `KindFile` arm (Path = Origin); @base-url suite |
| Malformed base URL from the flag fails loud | ✓ Covered | winner-validation `!isUsableURL` → `*BaseURLError`; @base-url suite |
| Unparseable `.glassfrogrc` on the output walk fails loud | ✓ Covered | `ResolveSelectionFromOS` returns resolve's rcfile error before parse; @output suite |
| Explicitly empty `--base-url` honoured by presence, fails loud | ✓ Covered | `FromFlags{Present}` yield + winner validation; `baseurl_test.go` empty/whitespace tests + @base-url suite |
| Whitespace-only `GLASSFROG_OUTPUT` treated as absent, falls through | ✓ Covered | `FromEnv` trim-yield → default; @output suite |
| No token anywhere is a normal empty outcome | ✓ Covered | `resolve` `KindNone`→`SourceNone`, nil error; @token suite |

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete; all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 Token retrofit | ✓ Met | env→file composition, provenance mapping, signature unchanged, secret hygiene (no format of `resolve.Resolution`), `internal/auth` suite green |
| T002 Base-URL retrofit + presence threading | ✓ Met | precedence preserved, winner-validation, presence threaded through `ResolveBaseURL`/`ResolveBaseURLFromOS`/`AssembleFromOS` + every RunE; compiler-enforced (no overload); RunE-level presence test (both flag positions, `--flag=`) |
| T003 Output retrofit + presence threading | ✓ Met | composing `ResolveSelectionFromOS`, 6-arg core folded+removed, 035 flag classification preserved, seam contract updated across all declarations + production impl, `readTemplateSource` untouched, output suite reworked onto temp-dir+getenv harness |

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| `Resolve() (Resolution, error)` — unchanged | ✓ Conformant | `internal/auth/resolve.go:71` |
| `ResolveBaseURL(flagValue string, flagPresent bool, startDir, homeDir string)` | ✓ Conformant | `baseurl.go` |
| `ResolveBaseURLFromOS(flagValue string, flagPresent bool)` | ✓ Conformant | `baseurl.go` |
| `AssembleFromOS(flagValue string, flagPresent bool)` | ✓ Conformant | `assemble.go:52` |
| `ResolveSelectionFromOS(flagValue string, flagPresent bool, startDir, homeDir string)` | ✓ Conformant | `selection.go` |
| 6-arg `ResolveSelection` folded in and **removed** | ✓ Conformant | absent from `internal/output` |
| `resolveSelection(flagValue string, flagPresent bool)` seam (all decls + productionSeam) | ✓ Conformant | `usertemplate.go:22,181` + 11 command seams |
| `readTemplateSource` seam — unchanged | ✓ Conformant | `usertemplate.go:23,198` |
| Provenance → per-domain mapping (Kind→enum, Origin→Path/label) | ✓ Conformant | switch arms in all three resolvers |

## Non-Behavior Absence

**Status**: Pass (8 of 8 exclusions honoured)

| Non-behavior | Status | Evidence |
|---|---|---|
| No public output type/enum/error shape change | ✓ Absent | `git diff origin/main` shows no type/enum/error-definition lines changed in auth/apiclient/output — only function signatures + bodies |
| No regression of 035 (non-token `--output` = template, `stdin` = piped, seam works) | ✓ Absent | `classifyFlagSelection` intact; @output template tests + `readTemplateSource` untouched |
| No setting constants moved into `internal/resolve` | ✓ Absent | no `Flag*`/`GLASSFROG_*`/`*Key` constant definitions in `internal/resolve` (matches are doc-comment examples only) |
| No value validation in `internal/resolve` | ✓ Absent | no `isUsableURL`/`ParseFormat` in `internal/resolve`; validation stays at call sites |
| No STDIN source in the three precedence walks | ✓ Absent | the three resolvers compose only `FromFlags`/`FromEnv`/`FromFile`/`Default` (token: `FromEnv`/`FromFile`); `FromStdin` is unused by them |
| Token resolver: no flag rung, no default, env→file preserved | ✓ Absent | `resolve()` composes `FromEnv, FromFile` only |
| No re-validation/normalization of the built-in default | ✓ Absent | base URL skips validation when `Kind == KindDefault`; output default → `Selection{DefaultFormat}` directly |
| No change to `.glassfrogrc` format / nearest-wins walk / rcfile errors | ✓ Absent | file rung delegates to `resolve.FromFile` → `rcfile.Resolve`; typed errors propagate verbatim |

## @wip Lifecycle Completion

**Status**: Pass

All 10 implemented scenarios (2 @token + 5 @base-url + 3 @output) have had `@wip` removed and pass in their per-package suites. The only remaining `@wip` tags are on the 3 `@validation` scenarios — correctly held out for this step (not referenced by any task).

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation, independent of the dimension passes)

| Scenario | Status | Trace |
|---|---|---|
| No precedence skeleton remains at a call site | ✓ Satisfied | The three files the scenario names — `internal/auth/resolve.go`, `internal/apiclient/baseurl.go`, `internal/output/selection.go` — each express precedence as a `resolve.Resolve(...)` source composition; none re-implements the env-trim / file-walk / source-ordering skeleton by hand (the residual `TrimSpace` in `selection.go` is 035 template-path interpretation, not a precedence walk). See O-1 for a scoped observation. |
| Public surfaces are byte-for-byte stable for consumers | ✓ Satisfied | `git diff origin/main` confirms no field, provenance member, or error-message-shape change to `auth.Resolution`/`Source`, `apiclient.BaseURL`/`BaseURLSource`/`*BaseURLError`, `output.Selection`/`OutputFormat`/`TemplateRef`/`*FormatError` — only function signatures gained the presence input |
| The flag-semantics change is the only observable behaviour difference | ✓ Satisfied | The single behaviour change (explicitly-supplied empty/whitespace `--base-url`/`--output` now fails loud) is pinned by the rewritten `TestResolveBaseURL_WhitespaceOnlyFlagSuppliedFailsLoud`, the new explicit-empty test, and `TestResolveSelection_EmptyFlagSuppliedIsDegenerateTemplate`; every other resolver/cli suite carries forward green, evidencing byte-identical behaviour on all other paths |

---

## Observations (non-blocking)

### O-1: 020's `ResolveFormat` pure core remains a hand-rolled skeleton (out of this slice's scope)

- **Dimension**: Validation scenario "No precedence skeleton remains" (scoped note, not a finding)
- **Source**: spec.md § System Overview ("deletes the duplicated skeleton"); tasks.md T003 scope (folds+removes only the 6-arg `ResolveSelection`)
- **Implementation**: `internal/output/format.go:ResolveFormat` / `ResolveFormatFromOS`
- **Why not a finding**: The validation scenario inspects exactly three files (`auth/resolve.go`, `apiclient/baseurl.go`, `output/selection.go`), and all three are now compositions. `ResolveFormat` lives in a fourth file (`format.go`), is 020's pre-fetched-source format-only core, is no longer on the production output path (`ResolveSelectionFromOS` composes `resolve` directly), and is retained deliberately because the 020 cli BDD suite still drives it. It is a candidate for the deferred enum-unification / skeleton cleanup spec (see DEPRECATION.md 2026-06-12 and 040 ADR-1). Surfaced here so a future reader inspecting the whole output domain isn't surprised by a residual hand-rolled walk.

---

## Verdict: Ready

All 5 conformance dimensions pass and all 3 held-out validation scenarios are satisfied through independent inspection. The retrofit is behaviour-preserving at every public surface (confirmed by diff: no type/enum/error-shape change), carries exactly the one intended behaviour change (presence-based empty-flag fail-loud, pinned by tests), preserves 035's template branch, and honours all eight non-behaviors. The one observation (O-1) is a scoped, deliberately-deferred residual that falls outside both the task scope and the validation scenario's literal inspection set. The implementation conforms to the specification.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge — PR #101 (`feat/040-resolution-call-site-retrofit`) is open against `main`, rebased clean on the latest main (043 + 030), full suite green. The deferred enum-unification cleanup (O-1 / 039's retired forecast) can be picked up as a later spec when judged worth the consumer blast radius.
