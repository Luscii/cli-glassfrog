# Validate: Base URL Resolution

**Feature**: 008-base-url-resolution
**Round**: 1 of 3
**Date**: 2026-06-06
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-spec.md, features/undefined-connection-settings/base-url-resolution.feature, PROJECT.md
**Implementation files**: `internal/apiclient/baseurl.go` (resolver, BaseURL/BaseURLSource/BaseURLError, constants), `internal/rcfile/{rcfile,resolve}.go` (the shared `.glassfrogrc` read/parse/walk the file rung delegates to), plus the godog suite `internal/apiclient/baseurl_bdd_test.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 4 of 4 validation scenarios satisfied by inspection.

---

## Driving Scenario Coverage

**Status**: Pass (10 of 10 behavioral scenarios covered)

Every behavioral scenario in the feature file traces to an identifiable code path in `ResolveBaseURL` (`internal/apiclient/baseurl.go`) and the shared walk `rcfile.Resolve` (`internal/rcfile/resolve.go`).

| Scenario | Status | Implementation |
|---|---|---|
| The flag overrides every other source | ✓ Covered | baseurl.go:157-162 (flag rung short-circuits; no other source read) |
| A malformed flag value fails loudly | ✓ Covered | baseurl.go:158-159 (`!isUsableURL` → `BaseURLError{Source:"--base-url"}`, no fall-through) |
| The built-in default is used when nothing is configured | ✓ Covered | baseurl.go:186-188 (`SourceDefault`, `DefaultBaseURL`) |
| The nearest config file wins over the home file | ✓ Covered | baseurl.go:175 → rcfile.Resolve nearest-first via `candidateDirs` (resolve.go) |
| The environment variable wins over a config file | ✓ Covered | baseurl.go:165-169 (env checked before the file rung; no file read) |
| A config file with no base URL is skipped | ✓ Covered | rcfile resolve.go (`if !ok { continue }` skips a key-less file) |
| An empty environment variable is ignored | ✓ Covered | baseurl.go:165 (`strings.TrimSpace(envValue) != ""` → falls through) |
| An unreadable config file fails loudly | ✓ Covered | baseurl.go:175-178 + rcfile.Resolve (non-`ErrNotExist` `*ReadError`, no default fall-through) |
| A malformed config-file value fails loudly naming the file | ✓ Covered | baseurl.go:180-181 (`BaseURLError{Source: filePath}`) |
| A malformed environment value names the environment variable | ✓ Covered | baseurl.go:166-167 (`BaseURLError{Source: EnvVarBaseURL}`) |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete; all criteria evidenced)

| Task | Status | Evidence |
|---|---|---|
| T001 — secret-safe `base_url` file read over the shared parser/walk | ✓ Met | Realized in `internal/rcfile` (`Read`/`ReadValue`/`Resolve` over `candidateDirs`); nearest-wins, key-absent-skips, missing-skips, unreadable→`*ReadError`, unparseable→`*FormatError`. Token-never-returned is now structural: `ReadValue(key)` returns only the requested key's value. (Location note below.) |
| T002 — precedence resolver + `http(s)` validation + code-free `BaseURL` | ✓ Met | `BaseURL{Value,Source,Path}`, `BaseURLSource` (Flag/Environment/File/Default, no None), `BaseURLError`; constants `GLASSFROG_BASE_URL`/`base-url`/`DefaultBaseURL`; `ResolveBaseURL` precedence; `isUsableURL` (absolute http(s)); whitespace falls through; malformed names source, no fall-through; default not re-validated; verbatim pass-through; `ResolveBaseURLFromOS` seam; no exit code / `os.Exit` / command surface. |
| T003 — executable acceptance via godog | ✓ Met | `TestBaseURLFeatures` scoped to `base-url-resolution.feature`; 10 behavioral scenarios pass; 4 `@validation` kept `@wip`; temp dirs + injected roots + stubbed env (no real network/home); both apiclient suites report independent counts (10 + 8). |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| Configuration inputs `--base-url` / `GLASSFROG_BASE_URL` / `.glassfrogrc base_url` / built-in default | ✓ Conformant | `FlagBaseURL="base-url"`, `EnvVarBaseURL="GLASSFROG_BASE_URL"`, `baseURLKey="base_url"`, `DefaultBaseURL="https://glassfrog.com/api/v5"` |
| Precedence (flag → env → file → default) | ✓ Conformant | `ResolveBaseURL` rung order |
| "Usable" URL contract (absolute http(s); whitespace absent; non-http(s) malformed) | ✓ Conformant | `isUsableURL` + the trim/fall-through rules |
| `BaseURL{Value, Source, Path}` output (no None member) | ✓ Conformant | struct + `BaseURLSource` enum |
| Error Communication — `BaseURLError` naming source; shared `ReadError`/`FormatError` for the file | ✓ Conformant | `BaseURLError.Source` is `--base-url` / `GLASSFROG_BASE_URL` / file path; file errors are `rcfile.ReadError`/`rcfile.FormatError` |

---

## Non-Behavior Absence

**Status**: Pass (all 7 excluded behaviors confirmed absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not resolve/read/attach the token | ✓ Absent | File rung is `rcfile.Resolve(…, baseURLKey)` — only the base_url value is returned; the token is structurally out of scope on this path |
| Must not make an API call / reachability check | ✓ Absent | No HTTP client or network dial anywhere on the path; `url.Parse` is offline |
| Must not write/create/modify a config file | ✓ Absent | Read-only (`os.ReadFile` in rcfile); no write calls |
| Must not decide exit code or error message | ✓ Absent | `BaseURLError` is code-free; no `os.Exit` |
| Must not normalize/rewrite/canonicalize the URL | ✓ Absent | Value passed verbatim (no trim/scheme/slash rewrite); pinned by the trailing-slash test |
| Must not prompt interactively | ✓ Absent | No prompt; deterministic, default-backed |
| Must not support multiple endpoints/profiles | ✓ Absent | Single `base_url` key; one effective value |

---

## @wip Lifecycle Completion

**Status**: Pass

The 10 behavioral scenarios carry no `@wip` and are served by `TestBaseURLFeatures`. The 4 `@validation` scenarios retain `@wip` — they are held out for this validation pass and are not referenced by any implementing task, so the tags are correctly still present.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation by inspection)

These held-out scenarios were never implemented as godog steps (correctly kept `@wip`); each is verified here independently against the code.

| Scenario | Status | Trace |
|---|---|---|
| Resolution is deterministic | ✓ Satisfied | `ResolveBaseURL` is a pure function of (flag, env, fs); no randomness or time; same inputs → same `Value`+`Source` (also pinned by `TestResolveBaseURL_Deterministic`) |
| Resolution always yields a value | ✓ Satisfied | baseurl.go:186-188 — the default backstops the chain; `BaseURLSource` has no None member, so no "absent" outcome is representable |
| Resolution performs no writes | ✓ Satisfied | Only `os.ReadFile` is reached (in rcfile); no create/write/rename on the resolution path |
| Resolution makes no network call | ✓ Satisfied | No HTTP/socket use in `ResolveBaseURL`, `rcfile.Resolve`, or `isUsableURL` (`url.Parse` is local) |

---

## Verdict: Ready

All 5 conformance dimensions pass and all 4 validation scenarios are satisfied by inspection. The implementation conforms to the Behavioral Accord, the driving and edge/error scenarios, the interface accord, and every Non-Behavior. The full test suite (rcfile, auth, apiclient incl. both godog suites, cli) was observed green, supplementing the inspection.

### Observations (transparency — not behavioral findings)

These do not affect behavioral conformance and carry no severity. They are recorded because the Guardian traces everything:

- **Implementation location diverged from the task/plan text — deliberately.** T001, plan ADR-3, and interface-spec.md describe the `base_url` file read living in `internal/auth` (beside the token reader). A post-implementation refactor moved the generic `.glassfrogrc` read/parse/walk into a new `internal/rcfile` package, with `auth` and `apiclient` as consumers and the `base_url` key owned by `apiclient`. Behavior is unchanged and fully conformant. The move is recorded in `.score/memory/DECISIONS.md` as an explicit supersession of 005 ADR-1 / 008 ADR-3. At the time of this validation pass the 008 spec artifacts still described the `auth` placement and lagged the code — a doc-vs-code drift on the artifacts, not a behavioral gap; cross-artifact consistency is analyze's domain, not validate's.
  - **Update (resolved in this PR, commit `ce076bc`)**: the 008 artifacts (plan ADR-3, tasks.md, interface-spec.md, risk.md) were reconciled to the `internal/rcfile` situation with dated amendment notes; the Guardian assessment artifacts (checklist.md, analyze.md) were intentionally left as historical pre-implementation records. The drift described above no longer exists.
- **Two stale code comments** (code-review scope, outside validate's behavioral lane): at the time of this pass, `baseurl.go:14` still said the `base_url` key "lives in internal/auth beside the token key" (it now lives in `apiclient`), and `baseurl.go:147` said a broken file "surfaces internal/auth's typed read/format error" (it now surfaces `rcfile`'s). Comment accuracy only — no behavioral effect.
  - **Update (resolved in this PR, commit `ce076bc`)**: both comments were corrected. `baseurl.go` now documents the `base_url` key as owned by `internal/apiclient` and the read/format errors as `internal/rcfile`'s.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. If you want the 008 artifacts (plan ADR-3, tasks.md, interface wording) re-pointed at `internal/rcfile` to match the refactor, run `/score:analyze 008` to surface the cross-artifact drift, or update them as a docs follow-up — neither blocks the merge.
