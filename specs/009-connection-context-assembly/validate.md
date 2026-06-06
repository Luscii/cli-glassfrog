# Validate: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Round**: 1 of 3
**Date**: 2026-06-06
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (3 of 3 tasks complete), interface-spec.md, connection-context-assembly.feature, PROJECT.md
**Implementation files**: 5 files in `internal/apiclient/` — `context.go`, `assemble.go` (production); `context_test.go`, `assemble_test.go`, `context_bdd_test.go` (tests)

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 4 of 4 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

Every driving scenario (spec.md § Driving Scenarios, plus the interface-proposed credential-error scenario) has an identifiable, passing code path. All eight are executable in the `TestConnectionContextFeatures` godog suite (8 scenarios, 8 passed).

| Scenario | Status | Implementation |
|---|---|---|
| Complete context from a usable base URL and a present token | ✓ Covered | `assemble.go:Assemble` packs both outcomes; `context.go:Complete` returns true |
| Built-in default base URL paired with a token still completes | ✓ Covered | `context.go:Complete` (source-agnostic); `BaseURL.Source == SourceDefault` carried verbatim |
| One context applies across multiple calls in an invocation | ✓ Covered | `ConnectionContext` is a reusable value; `AssembleFromOS` documented once-per-invocation. Pinned by the suite's reuse/no-re-resolve assertion (request-layer wiring is 010, per plan) |
| No credentials — context still assembles, carrying the absence | ✓ Covered | `Cred.Source == auth.SourceNone` carried; `context.go:Problems` names the credential; no fabrication |
| Both inputs report a problem — both are surfaced | ✓ Covered | `assemble.go:Assemble` carry-both (no short-circuit); `Problems()` lists both parts |
| Base URL error while a token is present | ✓ Covered | `BaseURLErr` carried; `Cred` kept intact; `Problems()` names the base-URL part |
| A credential error is carried into the context naming the file | ✓ Covered | `CredErr` carries `*rcfile.ReadError`; `Problems()` names the credential part |
| Token is redacted from the rendered context | ✓ Covered | `context.go:String` renders `<redacted>`/`<none>`, never the value |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 checked tasks conform)

| Task | Status | Evidence |
|---|---|---|
| T001 — `ConnectionContext` + readiness + redacting `String()` | ✓ Met | `Complete()` true only on usable base URL + present token + no errors; `Problems()` empty-when-complete, stable order (base URL → credential), secret-free; value-receiver `String()` redacts. `TestConnectionContextComplete/Problems/Redaction` green |
| T002 — `Assemble` + `AssembleFromOS` | ✓ Met | Both resolvers called exactly once even when the first errors (carry-both tripwire); every outcome combination packed; no error return; nil-resolver panic; `AssembleFromOS` binds `ResolveBaseURLFromOS`/`auth.Resolve`. `TestAssemble_*` green |
| T003 — executable acceptance via godog | ✓ Met | 8 behavioral scenarios un-wipped & passing; 4 `@validation` kept `@wip`; new `TestConnectionContextFeatures` suite names only this feature file; three `apiclient` suites report independent counts (008: 10, 007: 8, 009: 8); no real network/home/filesystem |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-spec.md) | Status | Implementation |
|---|---|---|
| `Assemble(resolveBaseURL func() (BaseURL, error), resolveCred func() (auth.Resolution, error)) ConnectionContext` | ✓ Conformant | `assemble.go:21` — exact signature; no `error` return; nil-resolver fail-fast panic |
| `AssembleFromOS(flagValue string) ConnectionContext` | ✓ Conformant | `assemble.go:52` — binds `ResolveBaseURLFromOS(flagValue)` + `auth.Resolve`, delegates |
| `ConnectionContext{ BaseURL, BaseURLErr, Cred, CredErr }` | ✓ Conformant | `context.go:23` — field names/types match (`BaseURL`, `error`, `auth.Resolution`, `error`) |
| `Complete() () bool` | ✓ Conformant | `context.go:41` — `BaseURLErr==nil && CredErr==nil && Cred.Source != auth.SourceNone` |
| `Problems() () []string` | ✓ Conformant | `context.go:51` — empty when complete; stable order; safe labels only |
| `String() () string` (value-receiver, redacting) | ✓ Conformant | `context.go:73` — value receiver; token reported present/absent |
| Error communication (never returns error; nil panics) | ✓ Conformant | `Assemble` has no error return; nil resolvers panic |

---

## Non-Behavior Absence

**Status**: Pass (9 of 9 exclusions absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not resolve/read/choose the base URL or token itself | ✓ Absent | Pure `Assemble` calls only injected resolvers; no `os`/env/file access in `context.go`/`assemble.go` `Assemble` |
| Must not decide whether the request proceeds / refuse | ✓ Absent | No refusal path; `Complete()`/`Problems()` report only |
| Must not decide exit code or user message | ✓ Absent | No `os.Exit`, no message construction |
| Must not make an API call / check reachability | ✓ Absent | No `net/http`; no outbound calls |
| Must not transform the carried values | ✓ Absent | Outcomes packed verbatim into fields; no normalization/encoding/trimming |
| Must not write/create/modify any file | ✓ Absent | No `WriteFile`/file mutation |
| Must not print/log/expose the token | ✓ Absent | `String()` redacts; `Problems()` derives only from safe labels; pinned by redaction test |
| Must not prompt interactively | ✓ Absent | No `os.Stdin`/`bufio` reads |
| Must not support multiple contexts/profiles/per-host entries | ✓ Absent | Single `ConnectionContext` value; no profile/map structures |

---

## @wip Lifecycle Completion

**Status**: Pass

The 8 behavioral scenarios referenced by T003 have had their `@wip` tags removed and pass in the suite. The 4 `@validation` scenarios are not referenced by any checked task (held out for this validation step) and correctly retain `@validation @wip`.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation)

These scenarios were held out from the Builder (`@validation @wip`). Each is traced to a code path by inspection; supporting unit tests are noted where they independently pin the same property.

| Scenario | Status | Trace |
|---|---|---|
| Assembly re-resolves nothing | ✓ Satisfied | `assemble.go:Assemble` invokes only the two injected resolvers; reads no flag/env/file directly (no `os`/`getenv`/`getwd` references in the pure path). The only OS binding is the `AssembleFromOS` seam, which delegates to the resolvers |
| Assembly performs no writes and no network call | ✓ Satisfied | No `WriteFile`/`net/http`/outbound I/O anywhere in `context.go`/`assemble.go`; assembly is in-memory over resolver return values |
| The token value never appears in produced output | ✓ Satisfied | `context.go:String` renders the token only as `<redacted>`/`<none>`; `Problems()` builds labels from `BaseURLErr.Error()`, credential `Source`/`Path`, and a fixed "no credentials found" phrase. Independently pinned by `context_test.go:TestConnectionContextRedaction` across `%v`/`%+v`/`%s`/`String()`/`Problems()` |
| Assembly is deterministic | ✓ Satisfied | `assemble.go:Assemble` is pure over its resolver outcomes — same outcomes → identical context. Independently pinned by `assemble_test.go:TestAssemble_Deterministic` |

**Supplementary test execution**: `go test ./internal/apiclient/ -count=1` passes (all unit suites + all three godog suites). Inspection-based findings above are the baseline; the green run is corroborating confidence, not a substitute.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 4 held-out validation scenarios are satisfied through inspection and corroborated by passing tests. The implementation conforms to the specification: assembly is a transparent, pure aggregator that pairs the two resolved outcomes, carries both forward (including errors and absence) without short-circuiting, reports readiness without deciding, and never leaks the token. The interface accord's signatures and contracts are matched exactly, and every spec non-behavior is absent.

One scope note (not a finding): the spec's "one context reused across every request" lifecycle is realized at the value level here (a reusable `ConnectionContext` + a once-per-invocation `AssembleFromOS` contract); the request-time wiring that replays the context into 007's `AuthTransport` is deferred to Request Execution (010) by plan ADR-2. This matches the plan and tasks; no gap against this slice's scope.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 009 is closed.
