# Validate: My Roles

**Feature**: 012-my-roles
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Issues
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-cli.md, features/self-service-reads/my-roles.feature, PROJECT.md
**Implementation files**: 3 production files — `internal/glassfrog/me.go` (grown `Role`, `Pagination`, `MyRolesResponse`), `internal/cli/myroles.go` (`formatMyRoles`, `incomplete`, `runMyRoles`, `newMyRolesCommand`), `internal/cli/app.go` (wiring); reuses `internal/cli/clienterror.go` + `internal/cli/me.go` error helpers (011). Tests: `internal/glassfrog/roles_test.go`, `internal/cli/myroles_pure_test.go`, `internal/cli/myroles_test.go`, `internal/cli/my_roles_bdd_test.go`.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✗ Fail | 1 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 4 passed, 1 finding (F-1). 2 of 2 validation scenarios satisfied by inspection.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All driving scenarios (and the edge cases) trace to identifiable code paths in `runMyRoles` / `formatMyRoles`, and are exercised by the godog suite (`my-roles.feature`, 10 scenarios / 51 steps passing).

| Scenario | Status | Implementation |
|---|---|---|
| List the roles I fill | ✓ Covered | `myroles.go:runMyRoles` (Execute `GET /me/roles` → `formatMyRoles` → stdout, exit 0) |
| A projected role carries its essentials, not the raw payload | ✓ Covered | `myroles.go:formatMyRoles` (name + `role_…`, Purpose, Domains, Accountabilities; no fillers/tags/flags — they are not fields on `Role`) |
| The practitioner fills no roles | ✓ Covered | `myroles.go:formatMyRoles` (`len(Data)==0` → `No roles.`, exit 0) |
| No usable token | ✓ Covered | `myroles.go:runMyRoles` → `reportClientError` → `classifyClientError` `*AuthError{NoCredentials}` → UsageError/2, "auth login" pointer; no request sent |
| The API cannot be reached | ✓ Covered | `clienterror.go` `*TransportError` → NetworkUnavailable/6; no retry |
| The API answers with a non-2xx status | ✓ Covered (message gap → F-1) | `clienterror.go` `*ResponseError` → APIError/3, status named |
| More roles exist than one response carried | ✓ Covered | `myroles.go:runMyRoles` + `incomplete` (stderr note, exit 0) |
| Extra arguments are rejected without an API call | ✓ Covered | `newMyRolesCommand` `Args: cobra.NoArgs` → UsageError/2, no request |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — grow `Role` + `/me/roles` schema + pure renderer | ✓ Met | `me.go` (snake_case-tagged `Role`/`Pagination`/`MyRolesResponse`), `myroles.go` (`formatMyRoles`/`incomplete`); `roles_test.go` pins snake_case tags (`has_next_page`→true, `next_cursor`→"abc", purpose/description populated), `myroles_pure_test.go` covers multi-role/empty/null-purpose/no-sections/field-absence |
| T002 — `roles` subcommand under runnable `me` | ✓ Met | `newMyRolesCommand`; wired in `app.go` (leaf-first then child-attach); `myroles_test.go` covers every branch + token-never-in-output + registration + Assemble; declares no own `--base-url` |
| T003 — godog acceptance suite | ✓ Met | `my_roles_bdd_test.go` points only at `my-roles.feature`; 10 scenarios / 51 steps pass; 2 `@validation` held `@wip` |

---

## Interface Contract Conformance

**Status**: Fail (surface and exit-code mapping conformant; one error-message gap, F-1)

| Surface element (interface-cli.md) | Status | Evidence / Finding |
|---|---|---|
| `glassfrog me roles` command, `Args: NoArgs`, non-empty `Short` | ✓ Conformant | `newMyRolesCommand` |
| `--base-url` inherited from root, not re-declared | ✓ Conformant | `myroles.go` reads `apiclient.FlagBaseURL`; `TestMyRolesCommand_DeclaresNoOwnBaseURLFlag` |
| Output: block per role (`<Name> (role_…)`, `Purpose:`, `Domains:` before `Accountabilities:`, `    - item` / `    (none)`, blocks blank-line separated) | ✓ Conformant | `formatMyRoles` + `writeRoleSection` |
| Empty result → exactly `No roles.`, exit 0 | ✓ Conformant | `formatMyRoles` |
| Incompleteness note (exact text) on stderr, exit 0 | ✓ Conformant | `incompleteRolesNote` matches interface-cli verbatim; written only when `incomplete(resp)` |
| Error→Outcome→exit-code mapping (2/1/3/6/1/2/2) via `classifyClientError`; no new category/case | ✓ Conformant | reuses `classifyClientError`; `TestMyRolesCommand_ExitCodesAcrossOutcomes` |
| Error messages: cause **+ concrete next step**, token-free | ✗ Non-conformant (2 of 7 arms) | F-1 |

---

## Non-Behavior Absence

**Status**: Pass (5 of 5 exclusions absent)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| No roles beyond the practitioner's; no actor/person selector | ✓ Absent | hits `/me/roles` only; `NoArgs`, no selector flag |
| No pagination / multi-page assembly | ✓ Absent | exactly one `Execute`; `NextCursor` decoded but unused; incompleteness signalled instead |
| No raw JSON default output; no output-format flag | ✓ Absent | `formatMyRoles` reshapes; command declares no flags |
| No base-URL/token resolution, header attach, fail-safe, or own exit codes | ✓ Absent | delegated to 007/008/009/010/004/011; never reads `ctx.Cred.Token` |
| No interpretation of non-2xx into a specific API error | ✓ Absent | all non-2xx → generic `APIError`/3; message names status only, does not classify 401/403/429 |

---

## @wip Lifecycle Completion

**Status**: Pass

The 10 scenarios referenced by checked tasks T002/T003 have had `@wip` removed and pass. The 2 `@validation @wip` scenarios (lines 95, 117) are correctly retained — they are held-out verification, not implementables referenced by a checked task. No stray `@wip` remains on an implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (2 of 2 traced to implementation by inspection)

These scenarios were not implemented as step definitions (held out); traced independently against the code.

| Scenario | Status | Trace |
|---|---|---|
| Default output contains no raw API envelope | ✓ Satisfied | `formatMyRoles` builds only labelled lines from projected fields; it never serializes `MyRolesResponse`/`Data`/`Meta` and emits no `data`/`meta`/`pagination`/`{` tokens. Supplementary: `TestFormatMyRoles_*` assert reshaped content; `TestRunMyRoles_SuccessMultiRole` asserts no raw envelope on stdout. |
| Incompleteness is never silent | ✓ Satisfied | `runMyRoles` calls `incomplete(resp)`; when `Meta.Pagination.HasNextPage` it always writes `incompleteRolesNote` to stderr while still printing the partial list (exit 0). Supplementary: `TestRunMyRoles_HasNextPageSignalsIncompleteOnStderr`, `TestIncomplete`. |

---

## Findings

### F-1: Non-2xx and decode failure messages omit the concrete next step the spec and interface require

- **Dimension**: Interface contract conformance (cross-reference: spec.md § Behavioral Accord > Failure — "Whatever the failure, the message names both what went wrong **and a concrete next step** the operator can take")
- **Source**: interface-cli.md § Error Communication table — the *non-2xx* row prescribes "...plus a **generic** next step ('the API rejected the read; check that the token has access and retry, or consult the status code')"; the *decode* row prescribes "the API response did not match the expected shape — this may be an API change; **report it**".
- **Implementation**: `internal/cli/me.go:formatClientErrorMessage` (reused by `runMyRoles` via `reportClientError`):
  - `*ResponseError` arm → `"the API returned a non-2xx response: status %d"` — names the cause/status but carries **no next step**.
  - `*DecodeError` arm → `"could not decode the API response: %s"` — names the cause but carries **no next step**.
- **Gap**: For these two failure modes the operator-facing message states what went wrong but not a concrete next step, where the spec's Failure accord makes the next step mandatory for *every* failure and the interface table prescribes specific next-step wording. The other arms (no-token, credential, transport, base-URL) do carry a next step and are conformant.
- **Notes / ambiguity**: (1) The renderer is **011's shared `formatClientErrorMessage`**, reused by design (plan ADR-2/3, interface Consistency Notes), so a fix touches a helper shared with `glassfrog me` — the developer decides whether to address it here, against the shared helper, or accept it as already-shipped 011 behavior. (2) For the non-2xx case there is mild interpretive latitude (the status arguably points the operator); the decode case clearly lacks any next step. Both are surfaced for transparency, not enforcement.

---

## Verdict: Issues

1 finding in 1 dimension (interface contract conformance). The `me roles` surface, output projection, empty-list handling, incompleteness signal, exit-code mapping, all non-behavior exclusions, the @wip lifecycle, and both held-out validation scenarios conform. The single gap is incremental — two reused error-message arms omit the concrete next step the spec's Failure accord and the interface table require — and is fixable in an implement round (or consciously accepted, since it lives in shared 011 code). This is not a fundamental gap: no dimension is wholly unimplemented and no validation scenario lacks a code path.

---

## Next Steps

1 finding (F-1) to address. Suggested options, developer's choice:
- Fix via `/score:implement`: add a next-step clause to the `*ResponseError` and `*DecodeError` arms of `formatClientErrorMessage` (matching interface-cli's prescribed wording), then re-validate (`/score:validate 012`). Because the helper is shared with `me`, update/extend the corresponding `me`/`clienterror` assertions in the same pass.
- Or consciously accept F-1 as inherited 011 behavior and proceed to PR review — the finding stays visible in this artifact, making that a transparent decision.
