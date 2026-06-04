# Validate: Credential Storage

**Feature**: 006-credential-storage
**Round**: 1 of 3
**Date**: 2026-06-04
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (5 of 5 tasks complete), interface-cli.md, interface-spec.md, features/unauthenticated-access/credential-storage.feature, PROJECT.md
**Implementation files**: `internal/auth/{credentials.go, write.go}` (shared format module + writer), `internal/cli/{authlogin.go, authlogin_seam.go, authcmd.go}` (resolution, seam, command), `internal/cli/dispatch.go` (commandUsageError seam), `internal/cli/app.go` (wiring); tests in `internal/auth/auth_test.go`, `internal/cli/{authlogin_test.go, authlogin_seam_test.go, authcmd_test.go, credstorage_bdd_test.go, dispatch_test.go}`

> Note: `agents/guardian-agent.md` is not present in this Score version — validated against SKILL.md alone (reduced character consistency, not a blocked skill).

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

**Status**: Pass (11 of 11 scenarios covered)

Every driving scenario has an identifiable code path; all are referenced by checked tasks (T004 references all command-level scenarios, T005 their executable steps).

| Scenario | Status | Implementation |
|---|---|---|
| Store a token argument to the home file | ✓ Covered | `authlogin.go:resolveTokenSource` (tokenFromArg) → `authcmd.go:runLogin` → `auth.WriteCredentials` |
| Store a piped token to the current directory | ✓ Covered | `resolveTokenSource` (tokenFromStdin) + `targetPath(cwd)` |
| Persist a token from the environment | ✓ Covered | `resolveTokenSource` (tokenFromEnv) |
| Interactive prompt for a missing token | ✓ Covered | `resolveTokenSource` (tokenNeedsPrompt) → `interactor.promptToken` (`ttyInteractor`, `term.ReadPassword`) |
| Target location not writable | ✓ Covered | `write.go:writeAtomic` CreateTemp failure → `*WriteError` → `runLogin`/`classifyAuthError` |
| Existing file cannot be parsed for a merge | ✓ Covered | `write.go:WriteCredentials` parse-validate → `*FormatError`, no write |
| Non-interactive session, no token | ✓ Covered | `resolveTokenSource` (tokenNone) → UsageError "no token to store" |
| Supplied token is blank | ✓ Covered | `authlogin.go:usableToken` → UsageError |
| Merge preserves other keys | ✓ Covered | `write.go:mergeTokenLine` (line-preserving replace/append) |
| Existing token, non-interactive, no overwrite | ✓ Covered | `authlogin.go:existingTokenGuard` (guardBlocked) → UsageError |
| Interactive confirmation chooses location | ✓ Covered | `existingTokenGuard` (guardInteractive) → `confirmReplace` + `chooseLocations` |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 writer + shared format module | ✓ Met | `internal/auth`; 12 unit tests (new-file create, 0600 perms, merge-preserves, malformed→no-write, failed-write-leaves-original/no-temp, round-trip) |
| T002 pure resolution / blank / target / guard | ✓ Met | `authlogin.go`; table-driven hermetic tests for precedence, blank rejection, target path, guard |
| T003 production input seam + non-echoing prompt | ✓ Met | `authlogin_seam.go`; `gatherInputsFrom` read-vs-skip tests; `term.ReadPassword` prompt |
| T004 auth group + login leaf, classify outcomes | ✓ Met | `authcmd.go`/`app.go`; runLogin + command tests; usage→code 2 / write→code 1 verified |
| T005 executable acceptance (godog) | ✓ Met | `credstorage_bdd_test.go`; 12 behavioral scenarios pass; 3 @validation held out |

`go build ./...`, `go vet ./...`, and `go test ./...` all clean.

---

## Interface Contract Conformance

**Status**: Pass (both accords conformant)

**interface-cli.md** (invocation surface):

| Surface element | Status | Evidence |
|---|---|---|
| `auth` group (non-runnable, non-empty Short) | ✓ Conformant | `authcmd.go:newAuthCommand` ("Manage Glassfrog API credentials"); guard-registered |
| `auth login [TOKEN]` (`MaximumNArgs(1)`) | ✓ Conformant | `newAuthLoginCommand`; extra positional → UsageError (test) |
| `--cwd`, `--overwrite` flags | ✓ Conformant | `cmd.Flags().BoolVar` for both |
| Success output `Stored credentials in <path>` | ✓ Conformant | `runLogin` success line; smoke-confirmed, token absent |
| Token-source precedence (arg→stdin→env→prompt) | ✓ Conformant | `resolveTokenSource` |
| Error table → Outcome categories / codes | ✓ Conformant | no-token/blank/existing-no-overwrite → UsageError (2); write/format → RuntimeError (1); mapped via existing `ExitCode` |

**interface-spec.md** (written artifact):

| Contract | Status | Evidence |
|---|---|---|
| Write outcomes by prior state (absent→create / no-token→append / has-token→replace / unparseable→no write) | ✓ Conformant | `WriteCredentials` + `mergeTokenLine` |
| At-rest `0600`, secret never in a more-permissive intermediate | ✓ Conformant | `writeAtomic` chmod 0600 before bytes; temp-in-same-dir + rename |
| Round-trip with the reader | ✓ Conformant | `ReadCredentialsFile`; round-trip unit tests |
| Errors name paths only | ✓ Conformant | `FormatError`/`WriteError` carry Path; no token |

---

## Non-Behavior Absence

**Status**: Pass (0 of 6 excluded behaviors present)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not resolve/read/use a stored token for a request | ✓ Absent | `ReadCredentialsFile` is used only for the existing-token guard/merge-validation, never to attach a token to a request; no request path exists |
| Must not make an API call to validate the token | ✓ Absent | no `net/http`/network imports in auth or command code |
| Must not print/log/echo the token (input or success) | ✓ Absent | prompt via `term.ReadPassword` (no echo); success names path only; errors carry path only; unit + BDD assert token absent from output |
| Must not support multiple tokens/profiles/per-host | ✓ Absent | single `token` key; last-token-wins parse; no profile structure |
| Must not remove/clear credentials this slice | ✓ Absent | no logout/remove/delete/clear command |
| Must not decide the process exit code | ✓ Absent | `runLogin` returns a code-free `Outcome`; no `os.Exit` in command/auth; mapping stays in 004's `ExitCode` |

---

## @wip Lifecycle Completion

**Status**: Pass

The 12 behavioral scenarios (referenced by checked tasks T004/T005) have had `@wip` removed and execute under `TestFeatures`. The 3 `@validation` scenarios correctly retain `@validation @wip` — held out for this validation pass, not Builder work.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation; corroborated by passing unit tests)

These scenarios are `@wip` (no step definitions) by design — traced by inspection, with existing unit tests as independent corroboration.

| Scenario | Status | Trace |
|---|---|---|
| Stored token round-trips through Discovery | ✓ Satisfied | `auth.WriteCredentials` → `auth.ReadCredentialsFile` (the shared format module Discovery will resolve through — the binding contract, ADR-1); `TestWriteCredentials_RoundTripsThroughReader` + `_RoundTripsAfterMerge`. See note below. |
| Token value never appears in produced output | ✓ Satisfied | success line + every error (`FormatError`/`WriteError`) carry paths only; non-echoing prompt; asserted by `TestRunLogin_ArgToHome_Success` (no token in stdout+stderr) and BDD `credNoTokenInOutput` |
| Stored file is owner-only readable | ✓ Satisfied | `writeAtomic` chmod `0600` before any bytes; `TestWriteCredentials_AbsentPath_OwnerOnlyPermissions`; smoke run showed `.rw-------` |

**Note (round-trip)**: Credential Discovery (005) is not yet implemented, so the scenario's literal "Discovery resolves it back" cannot execute end-to-end. Per plan ADR-1, 006 created the shared `internal/auth` format module that Discovery *will* read through; the round-trip is verified against that exact reader. The contract is pinned; when 005 lands it consumes this module rather than a parallel one. This is a planned, documented sequencing condition (tasks.md T001 / DECISIONS / LEARNINGS), not a conformance gap.

---

## Verdict: Ready

All five conformance dimensions pass with zero findings, and all three held-out validation scenarios trace to clear code paths corroborated by passing tests. All 5 tasks are checked; `go build`/`go vet`/`go test ./...` are clean; an end-to-end smoke run confirmed the production seam (0600 file, token never echoed, usage errors exit 2, existing-credential guard).

The implementation conforms to its specification. The specification loop is closed.

One observation carried forward (not a finding): 006 added a `commandUsageError` seam to `internal/cli/dispatch.go` (002's code) so a resolved command can emit `UsageError` — the mechanism ADR-5/DECISIONS assume but that did not previously exist. It is additive, pinned by `TestRun_CommandUsageError_IsUsageErrorCategory`, and recorded in LEARNINGS for 007. The other carried-forward item is the 005 sequencing note above.

---

## Handoff

Implementation conforms to the specification — suggest **PR review and merge** against `main`.

- The 3 `@validation` scenarios remain `@wip` with no step definitions; if you want them executable (not just inspection-traced), that is a small follow-up — but their properties are already covered by passing unit tests.
- Coordinate the merge order with 005-credential-discovery: whichever lands first owns the shared `internal/auth` format module; the other must consume it (the round-trip test guards the contract).
