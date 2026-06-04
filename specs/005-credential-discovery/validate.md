# Validate: Credential Discovery

**Feature**: 005-credential-discovery
**Round**: 1 of 3
**Date**: 2026-06-04
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-spec.md, features/unauthenticated-access/credential-discovery.feature, PROJECT.md
**Implementation files**: 2 production files in `internal/auth/` (`credentials.go`, `resolve.go`) + 3 test files (`credentials_test.go`, `resolve_test.go`, `bdd_test.go`)

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

Every driving scenario has an identifiable code path. The behavioral scenarios are also executable: `internal/auth`'s godog suite runs all 10 behavioral scenarios green.

| Scenario | Status | Implementation |
|---|---|---|
| Environment variable overrides any stored file | ✓ Covered | `resolve.go:75-79` — non-empty `getenv(GLASSFROG_TOKEN)` → `SourceEnvironment`, returns before any file read (Path stays empty) |
| Nearest credentials file wins over the home file | ✓ Covered | `resolve.go:81-93` over `candidateDirs` (startDir first); first usable token wins |
| Walk-up finds an ancestor's credentials file | ✓ Covered | `candidateDirs` ascends via `filepath.Dir` (`resolve.go:115-125`) |
| Home-directory file as the final fallback | ✓ Covered | `candidateDirs` appends `homeDir` last (`resolve.go:126-128`) |
| A credentials file exists but cannot be read | ✓ Covered | `resolve.go:84-88` — non-`ErrNotExist` read error fails loud, no fall-through |
| A credentials file cannot be parsed | ✓ Covered | `FormatError` (`credentials.go:90-94`) propagated by `resolve.go:88`, not reported as absence |
| No credentials anywhere | ✓ Covered | `resolve.go:96` → `Source: None`, nil error |
| Environment variable set but empty | ✓ Covered | `resolve.go:75` `strings.TrimSpace(token) != ""` — an empty, unset, or whitespace-only value falls through to the file search |
| A file is present but holds no token | ✓ Covered | `readCredentialsFile` `found=false` (`credentials.go:105`) → `resolve.go:90-91` skip, continue |

## Acceptance Criteria

**Status**: Pass (all criteria of all 3 checked tasks met)

- **T001** — `readCredentialsFile` returns `(token, true)` for `token=gf_x` (`credentials.go:98-105`); tokenless / whitespace-only value → `found=false`, no error (line 105); `#` comments + blanks + unknown keys ignored (lines 86, 96-100); non-blank/non-comment line without `=` → `FormatError` naming the path with no token in the message (lines 90-94); 7 RED-first unit tests; `go build`/`go vet` clean. ✓
- **T002** — env-first short-circuit on a value non-empty after trimming, with empty/unset/whitespace-only falling through (`resolve.go:75-79`, `strings.TrimSpace(token) != ""`); nearest-wins / walk-up / home fallback (`candidateDirs`); home-on-ascent deduped to a single read and walk-up stops at root (`seen` map + `parent == dir` break, lines 107-124); tokenless-skip + fail-loud read/format errors naming the path + `None`-on-absence (`resolve.go:81-96`); injected `startDir`/`homeDir` + stubbed env seam — no test reads the real `~/.glassfrogrc`; 13 unit tests; build/vet clean. ✓
- **T003** — all 10 non-`@validation` scenarios executable and passing; `@wip` removed from them, the 3 `@validation` scenarios kept `@wip`; steps set/unset the real `GLASSFROG_TOKEN` and build temp `.glassfrogrc` trees while binding `getwd`/`userHomeDir` to temp dirs (never the real home); suite + build/vet clean. ✓

*Observation (not a finding):* T003's criterion mentions "the suite asserts no token value appears in captured output." Discovery produces no output on the behavioral path, so the secret-hygiene assertion correctly lives (a) in the held-out `@validation` "token never in output" scenario and (b) as an explicit `!strings.Contains(err.Error(), secret)` assertion on the format-error path in the unit tests (`credentials_test.go:112`). The read-error path cannot leak a token by construction (`os.ReadFile` returns a path-only `*PathError` and the unreadable case is now exercised with a directory, which has no content), so the leak check belongs on the parse path. The behavior is delivered and tested; only the assertion's location differs from the criterion's literal wording.

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| `GLASSFROG_TOKEN` — value non-empty after trimming used as-is + short-circuits; empty/unset/whitespace-only ignored (falls through) | ✓ Conformant | `resolve.go:75-79` |
| `.glassfrogrc` format — split on first `=`, trim key/value, unknown keys ignored | ✓ Conformant | `credentials.go:89-100` |
| Empty/whitespace-only `token` value → no token present | ✓ Conformant | `credentials.go:97,105` |
| Blank lines ignored; first-non-whitespace-`#` ignored | ✓ Conformant | `credentials.go:85-86` (`TrimSpace` then `HasPrefix("#")`) |
| Non-blank/non-comment line without `=` → malformed | ✓ Conformant | `credentials.go:90-94` |
| Search precedence: env → CWD+ancestors (nearest) → home (read once) | ✓ Conformant | `candidateDirs` + `resolve` loop |
| `Resolution{Token, Source, Path}`; `Source ∈ {Environment, File, None}`; Path set only for File | ✓ Conformant | `resolve.go:14-38, 78, 93, 96` |
| Error Communication table (found / none-no-error / read-error / format-error) | ✓ Conformant | `resolve.go:84-96`; errors name only the path |

## Non-Behavior Absence

**Status**: Pass (all 7 exclusions absent)

Direct inspection of the production files (`credentials.go`, `resolve.go`) confirms no excluded capability is present:

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not write/create/modify any credentials file | ✓ Absent | only `os.ReadFile`; no `os.Create`/`WriteFile`/`Mkdir`/`OpenFile`/`.Write(` in production code |
| Must not attach the token to requests / make an API call | ✓ Absent | no `net/http`, no header attachment |
| Must not decide the exit code or no-token message | ✓ Absent | no `os.Exit`, no exit codes; returns a code-free `Resolution` |
| Must not print/log/expose the token in plaintext | ✓ Absent | no `fmt.Print`/`log.`; errors carry only the path; `Token` never rendered |
| Must not prompt interactively | ✓ Absent | no `os.Stdin`/`bufio`/`Scan` |
| Must not support multiple tokens/profiles/per-host | ✓ Absent | single `tokenKey = "token"`, single `Token` field |
| Must not accept a token via a CLI flag in this slice | ✓ Absent | no `flag`/`cobra`/`pflag` references in the package |

## @wip Lifecycle Completion

**Status**: Pass

The only remaining `@wip` tags are on the 3 `@validation` scenarios (`unauthenticated-access/credential-discovery.feature:57,64,71`). T003 (checked) explicitly keeps these held out and removed `@wip` from the 10 behavioral scenarios it implemented. No scenario referenced by a checked task for implementation still carries `@wip`.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 scenarios traced to implementation)

These were held out from the Builder. Traced independently against the code; the behavioral test suite skips them (`Tags: "~@wip"`) and they have no step definitions, so satisfaction is inspection-based, with unit tests as supplementary evidence.

| Scenario | Status | Trace |
|---|---|---|
| Resolution is deterministic | ✓ Satisfied | `resolve` has no randomness/time; `candidateDirs` returns an ordered slice (the `seen` map is membership-only, iteration is over the ordered slice). Same env + filesystem → same candidate order → same first-usable result from the same source. |
| The token value never appears in produced output | ✓ Satisfied | Discovery prints nothing. `FormatError` carries no file content; `ReadError` wraps an `os` error (path, not token). The only token-bearing field is `Resolution.Token`, documented as never rendered. Unit tests pin token absence in both error types (`credentials_test.go:112`, `resolve_test.go:212`). |
| Discovery performs no writes | ✓ Satisfied | Only `os.ReadFile` is used across both production files; no create/modify/mkdir call exists. |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 3 held-out validation scenarios are satisfied through inspection (with unit-test corroboration for secret hygiene). All 3 tasks are checked, all 9 driving scenarios have identifiable code paths (10 behavioral godog scenarios execute green), every interface surface conforms, and all 7 non-behaviors are confirmed absent in the production code. The implementation conforms to its specification.

The two `[ASSUMED]` markers (`.glassfrogrc` name/format and `GLASSFROG_TOKEN`) remain a forward coordination item with Credential Storage (006), as the spec and plan record — not a conformance gap in this slice.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. Before 006/007 ship, reconcile the `[ASSUMED]` file-format and env-var-name contract between Credential Discovery and Credential Storage as the artifacts flag.
