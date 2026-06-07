# Validate: Identity Read

**Feature**: 011-identity-read
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, interface-spec.md, features/self-service-reads/identity-read.feature, PROJECT.md
**Implementation files**: 4 — `internal/glassfrog/me.go` (schema), `internal/cli/me.go` (command + pure runMe/formatMe/validateInclude + seam), `internal/cli/clienterror.go` (classifyClientError), and the extensions to `internal/cli/dispatch.go`/`exitcode.go`/`root.go`/`app.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings. 4 of 4 validation scenarios satisfied.

All 5 tasks (T001–T005) are checked complete in tasks.md — full validation, not partial.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

Every driving scenario has an identifiable code path AND an executable, passing godog path (the 11 un-@wip behavioral scenarios in `identity-read.feature` cover all 8 driving scenarios plus 3 architecture/guard-derived ones).

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| me prints the resolved identity | ✓ Covered | `me.go:runMe` (success path) → `formatMe` |
| me distinguishes a human from an agent | ✓ Covered | `formatMe` renders `(kind)` + `agt_` id |
| me embeds roles on request | ✓ Covered | `runMe` adds `Query{include:roles}`; `formatMe` lists roles |
| an unusable token surfaces a non-2xx outcome | ✓ Covered | `*ResponseError` → `classifyClientError` → APIError(3) |
| a transport failure is surfaced as transport | ✓ Covered | `*TransportError` → NetworkUnavailable(6); single `Execute`, no retry |
| no usable token — the fail-safe is propagated | ✓ Covered | `*AuthError{NoCredentials}` → UsageError(2), no request sent |
| an unsupported include target is rejected before any request | ✓ Covered | `validateInclude` runs before assemble/send (tripwire-pinned) |
| roles embed requested but the actor fills none | ✓ Covered | `formatMe` omits the roles section when `len(Roles)==0` |

---

## Acceptance Criteria

**Status**: Pass (all checked tasks' criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — `internal/glassfrog` schema | ✓ Met | Decode fixtures (identity / roles embed / no-embed / unknown-field tolerance) pass; leaf package, `go build`/`go vet` clean |
| T002 — Outcome/ExitCode + classifyClientError | ✓ Met | `ExitCode(APIError)==3`, `ExitCode(NetworkUnavailable)==6`; classifier table + len+comma-ok exhaustiveness guard; `*AuthError` discriminated before `*TransportError`/rcfile |
| T003 — persistent `--base-url` root flag | ✓ Met | Registered under `apiclient.FlagBaseURL`; inheritance/retrievability pinned; 003 help regression updated and passing |
| T004 — `me` command + pure trio + seam | ✓ Met | 19 unit tests cover every branch incl. tripwire (no request on usage), token-never-in-output, and the outcome→exit-code table (0/2/3/6) |
| T005 — wiring + godog acceptance | ✓ Met | One `MustRegister` line in `Assemble()`; `TestIdentityReadFeatures` scoped to only `identity-read.feature`; 11 behavioral scenarios pass (57 steps), 4 `@validation` kept `@wip` |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

**CLI (`interface-cli.md`)**:

| Surface | Status | Implementation |
|---|---|---|
| `glassfrog me [--include <target>]`, NoArgs | ✓ Conformant | `newMeCommand` (`Use:"me"`, `cobra.NoArgs`) |
| `--include` local flag, value `roles` | ✓ Conformant | `cmd.Flags().StringSliceVar(&include, "include", …)` |
| `--base-url` persistent root flag, name from `FlagBaseURL` | ✓ Conformant | `root.PersistentFlags().String(apiclient.FlagBaseURL, …)` |
| Output projection (actor name/kind/id, org name/id, access; roles section present/omitted) | ✓ Conformant | `formatMe` |
| Error/exit-code table (2/2/2/1/1/3/6) | ✓ Conformant | `classifyClientError` + `outcomeToDispatchError` + `ExitCode`; pinned by the exit-code table test |

**Specification (`interface-spec.md`)**:

| Symbol | Status | Implementation |
|---|---|---|
| `glassfrog.{MeResponse,Actor,Organization,Membership,Role}` | ✓ Conformant | `internal/glassfrog/me.go` (projected fields + tolerant decode) |
| `Outcome` += `NetworkUnavailable`, `APIError`; `ExitCode` cases | ✓ Conformant | `dispatch.go`, `exitcode.go` (ExitCode stays a pure mapper) |
| `classifyClientError(err) Outcome` | ✓ Conformant | `clienterror.go` (single `errors.As` chain) |
| `newMeCommand(seam) *cobra.Command` | ✓ Conformant | `me.go` |
| `runMe(cfg) (Outcome, error)` / `formatMe(me, bool) string` / `validateInclude([]string) error` | ✓ Conformant | `me.go` (signatures match) |
| `meSeam` (assemble + build client over a base transport) | ✓ Conformant | `meSeam` interface; `productionSeam` binds `AssembleFromOS`/`NewClientFromOS` |
| Consumed-unchanged `apiclient` seam | ✓ Conformant | `me` adds nothing to `apiclient` |

---

## Non-Behavior Absence

**Status**: Pass (0 of 8 non-behaviors violated)

| Non-behavior (spec.md § Non-Behaviors) | Status | Evidence |
|---|---|---|
| Must not emit raw/structured JSON | ✓ Absent | `formatMe` emits labelled key-value lines; no `json.Marshal`/`Encode` in `me.go` |
| Must not attach token / resolve base URL / open connection / apply timeout itself | ✓ Absent | `me` delegates to `apiclient` (assemble + `Execute`); reads no env/credfile directly |
| Must not interpret non-2xx into a specific API error | ✓ Absent | `classifyClientError` maps every `*ResponseError` → generic APIError; no 401/403/429 special-casing |
| Must not own the full roles surface | ✓ Absent | `Role` is minimal id+name; `formatMe` lists only those |
| Must not paginate / retry / back off on 429 | ✓ Absent | Exactly one `client.Execute`; no retry/loop |
| Must not print/log/expose the token | ✓ Absent | `me.go` never reads `ctx.Cred.Token`; messages are path/status-only; token-never-in-output pinned across all branches |
| Must not prompt interactively | ✓ Absent | `me` has no interactor/prompt path |
| Must not span/switch organizations | ✓ Absent | Single `GET /me` against the one resolved identity |

---

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral scenarios referenced by T005 have their `@wip` removed and run in `TestIdentityReadFeatures`. The 4 `@validation @wip` scenarios correctly retain `@wip` — they are held out from the Builder for independent verification (inspected below), not referenced by an implement task. No stray `@wip` on a should-be-implemented scenario.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation)

These were held out from the implementing pass (`@validation @wip`); traced here independently against the code.

| Scenario (spec.md § Validation Scenarios) | Status | Trace |
|---|---|---|
| me resolves nothing itself | ✓ Satisfied | `me.go` reads no `os.Getenv`, no `.glassfrogrc`, and no `ctx.Cred.Token`; it hands the `--base-url` value to `AssembleFromOS` (009's documented input) and sends via the injected seam. The request descriptor is built only from method/path + the validated `--include` query; identity rides 007's `AuthTransport`. See nuance note below. |
| the token value never appears in produced output | ✓ Satisfied | `formatMe` renders response-side fields only; error messages name path/status, never the header; `me` never reads `Cred.Token`. Pinned by `TestRunMe_*` (token-leak guard in `runMeOver`) and the BDD When step across every scenario. |
| no structured-JSON output leaks in | ✓ Satisfied | `formatMe` produces `actor:`/`organization:`/`access:`/`roles:` labelled lines; no JSON encoder is invoked anywhere in the read path. |
| a non-2xx status is surfaced, not classified | ✓ Satisfied | `classifyClientError` maps any `*ResponseError` → generic `APIError`; `formatClientErrorMessage` reports `status N` with no per-status interpretation. `TestMeCommand_ExitCodesAcrossOutcomes` exercises a 404 → APIError(3). |

**Nuance note (scenario "me resolves nothing itself")**: the implementation reads two flags that are its own mandated surface — `--include` (the request-shaping query, per the Behavioral Accord § Entry) and `--base-url` (handed verbatim to `AssembleFromOS`, the resolve-once input per Assumptions + ADR-2). It does not read any environment variable or credentials file, and re-resolves neither identity nor base URL itself — the connection chain (008/009) and the authenticated transport (007) own that. So the scenario's intent (me delegates rather than re-resolving) holds; the flags it reads are by-design surfaces, not a bypass of the resolution chain. Recorded as Satisfied with this clarification rather than a finding, since there is no gap between spec promise and delivery.

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios are satisfied through inspection. The implementation conforms to its specification: the `me` command issues a single authenticated `GET /me`, projects the reshaped identity (with the opt-in roles embed), and maps Request Execution's typed outcomes onto the 0/1/2/3/6 exit-code surface — without owning transport, identity, base-URL resolution, per-status interpretation, JSON output, pagination/retry, or the token. The full module builds, vets, and tests clean.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The specification loop for 011 is closed.
