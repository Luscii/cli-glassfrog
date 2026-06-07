# Validate: My Actions

**Feature**: 013-my-actions
**Round**: 1 of 3
**Date**: 2026-06-07
**Verdict**: Issues
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (4 of 4 tasks complete), interface-cli.md, interface-spec.md, features/self-service-reads/my-actions.feature, PROJECT.md
**Implementation files**: 5 — `internal/glassfrog/actions.go`, `internal/cli/status.go`, `internal/cli/my_actions.go`, `internal/cli/app.go` (wiring), plus tests (`actions_test.go`, `status_test.go`, `my_actions_test.go`, `my_actions_bdd_test.go`)

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

**Total**: 5 dimensions checked, 4 passed, 1 finding (F-1). 5 of 5 @validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (10 of 10 scenarios covered)

Every driving scenario (spec.md) and architecture-informed behavioral scenario (feature file) referenced by the checked tasks has an identifiable code path, and all 11 behavioral scenarios pass executably via `TestMyActionsFeatures`.

| Scenario | Status | Implementation |
|---|---|---|
| list the practitioner's actions | ✓ Covered | `my_actions.go:runMyActions` (GET /me/actions) → `formatMyActions` |
| filter by a supported status | ✓ Covered | `runMyActions` step 3 sets `Query{"status"}`; `validateStatus` accepts |
| more results than one page | ✓ Covered | `incompleteActions` + `incompleteActionsNote` (stderr); single `Execute` |
| no usable token | ✓ Covered | `classifyClientError` → `*AuthError{NoCredentials}` → UsageError; transport fail-safe |
| API responds with a non-2xx | ✓ Covered | `classifyClientError` → `*ResponseError` → generic APIError |
| no matching actions | ✓ Covered | `formatMyActions` empty-list branch → `No actions.` + Success |
| invalid status rejected before any request | ✓ Covered | `runMyActions` step 1 (`validateStatus` before assemble) |
| network failure (architecture-informed) | ✓ Covered | `*TransportError` → NetworkUnavailable, no retry |
| undecodable response (architecture-informed) | ✓ Covered | `*DecodeError` → RuntimeError |
| malformed base URL / credentials file (architecture-informed) | ✓ Covered | base-URL error → UsageError; `*AuthError{CredentialError}` → RuntimeError |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 checked tasks satisfied)

| Task | Status | Evidence |
|---|---|---|
| T001 — `glassfrog.Action` + envelope decode | ✓ Met | `actions.go`; `actions_test.go` (single-page, multi-page, empty, nullable, unknown-field tolerance) |
| T002 — shared `validateStatus` + status set | ✓ Met | `status.go`; `status_test.go` (7 supported pass, empty passes, unsupported names value+sorted set, pure) |
| T003 — `me actions` command + pure trio | ✓ Met | `my_actions.go`; `my_actions_test.go` (success, filter+query, status-rejection tripwire, has-next signal, empty, every error branch, token-never-in-output, no own `--base-url`) |
| T004 — wiring + godog suite | ✓ Met | `app.go` (one `MustRegister` under `me`); `TestMyActionsFeatures` scoped to its own feature file; 11 behavioral pass, 5 `@validation` held `@wip`; each suite reports its own count |

> Note: T003/T004 acceptance criteria reference "the `my` parent"; the implementation registers under the `me` parent. See F-1 — this is a documented, spec-authorized convention alignment, not an unmet criterion (every behavioral criterion is satisfied).

---

## Interface Contract Conformance

**Status**: Fail (command path non-conformant; all other surfaces conformant — 1 finding)

| Surface | Status | Finding |
|---|---|---|
| Command path (`glassfrog my actions`) | ✗ Non-conformant | F-1 |
| `--status` flag (local, spec-set values, validated pre-request) | ✓ Conformant | — |
| `--base-url` (persistent root flag, inherited, not re-registered) | ✓ Conformant | — |
| Output projection (id/status/description/role/tags; `—` for null; empty-result line; more-available signal) | ✓ Conformant | — |
| Error Communication table (Success/UsageError/RuntimeError/APIError/NetworkUnavailable → 0/2/1/3/6) | ✓ Conformant | — |
| `classifyClientError` reuse, no new exit code, no new classifier branch | ✓ Conformant | — |
| Secret hygiene (never reads `ctx.Cred.Token`; token never rendered) | ✓ Conformant | — |

---

## Non-Behavior Absence

**Status**: Pass (9 of 9 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not walk pagination / fetch beyond first page | ✓ Absent | Single `Execute`; `incompleteActions` only reads `HasNextPage` |
| Must not send an unsupported `--status` | ✓ Absent | `validateStatus` rejects before any request |
| Must not render raw payload / own output shape | ✓ Absent | `formatMyActions` is a reshaped projection; no payload `Marshal` |
| Must not expose `--output json` | ✓ Absent | Only `--status` (local) + inherited `--base-url`; no json flag |
| Must not resolve/read token or base URL, nor attach header | ✓ Absent | Token rides AuthTransport via the seam; command reads only the resolved `--base-url` flag value; no `os.Getenv`/rcfile read |
| Must not interpret a non-2xx into a specific error | ✓ Absent | Generic message names status + generic next step (pinned by `notInterpretedAPIError` step) |
| Must not read the org-wide action surface | ✓ Absent | Path is `/me/actions` |
| Must not mutate actions | ✓ Absent | GET only |
| Must not prompt interactively | ✓ Absent | No prompts; outcomes are typed |

---

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral scenarios (referenced by checked task T004) have had `@wip` removed and pass executably. The 5 `@validation` scenarios retain `@validation @wip` — they are held out for this validate pass and are not referenced by any checked implementation task, so their retained `@wip` is correct (T004 explicitly keeps them held).

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 scenarios traced to implementation)

These were held out from the Builder. Traced independently against the code; existing unit tests provide supplementary confidence (they assert the same invariants but are distinct from the behavioral godog suite, which skips `@wip`).

| Scenario | Status | Trace |
|---|---|---|
| The command resolves nothing itself | ✓ Satisfied | `runMyActions` does I/O only through the injected seam (assemble/newClient/Execute); reads no env var or credentials file for token/base URL — token rides 007's AuthTransport. `--status` is the command's own request-shaping input, not a token/base-URL re-resolution. |
| Output is the reshaped projection, not structured JSON | ✓ Satisfied | `formatMyActions` emits labelled projection lines; no `json.Marshal` of the payload; no `--output json` flag exists. (`TestFormatMyActions_*`) |
| The token never appears in any output | ✓ Satisfied | Command never references `ctx.Cred.Token`; `formatMyActions` renders response-side fields only; token-leak assertion runs on every branch (`runMyActionsOver`, BDD `run` helper). |
| An unsupported status costs no request | ✓ Satisfied | `validateStatus` runs before `seam.assemble`/`newClient`; tripwire test confirms `transport.calls == 0` and `!assembleCalled`/`!newClientCalled` (`TestRunMyActions_UnsupportedStatusRejectedBeforeAnyRequest`). |
| Exactly one page request is made | ✓ Satisfied | Single `Execute`; `incompleteActions` reads `HasNextPage` without fetching; `tr.calls == 1` asserted on the has-next branch (`TestRunMyActions_HasNextPageSignalsMoreOnStderr`). |

---

## Findings

### F-1: CLI command path is `me actions`, interface accord specifies `my actions`

- **Dimension**: Interface contract conformance
- **Source**: `interface-cli.md § Surface > Command` (`glassfrog my actions [--status <status>]`; "A leaf under the `my` parent command (introduced by My Roles, 012)"); same `my actions` / `my` parent wording in `interface-spec.md` and `tasks.md` T003/T004.
- **Implementation**: `internal/cli/app.go` (`MustRegister(meCmd, newMyActionsCommand(...))` — attached to the `me` parent); `internal/cli/my_actions.go:newMyActionsCommand` (`Use: "actions"`). Invocation is `glassfrog me actions`; `glassfrog my actions` resolves to "unknown command".
- **Gap**: The interface artifacts describe a `my` parent command that does not exist. My Roles (012) — which the 013 artifacts name as the introducer of that parent — actually shipped as `me roles` (a leaf under the runnable `me` command, commit 959be80). The implementation correctly mirrors that established convention as `me actions`.
- **Severity / nature**: Low — **documentation drift, not a code defect**. spec.md (the behavioral authority) explicitly marks the command surface `[ASSUMED]` and states it "is pinned alongside My Roles (012) ... the behavior ... is fixed regardless of the final spelling" (§ Assumptions). Since 012 pinned `me <noun>`, `me actions` is the spec-authorized spelling; every behavioral contract (flags, projection, error mapping, exit codes, secret hygiene) conforms. The divergence lives only in the derived `interface-cli.md`/`interface-spec.md`/`tasks.md` prose, which predates 012's actual naming. Recorded during implementation in `.score/memory/LEARNINGS.md` (2026-06-07 naming-drift entry).
- **Suggested resolution** (developer's decision — Guardian does not decide): align the stale prose in `interface-cli.md`, `interface-spec.md`, and `tasks.md` (`my actions`/`my` parent → `me actions`/`me` parent) — a doc-only change the spec's `[ASSUMED]` clause already authorizes. No code change is warranted unless a deliberate, project-wide rename of the whole self-service-read group (`me roles` + `me actions` + future `me projects`) to a `my` parent is intended, which would be a separate spec touching 011/012/014.

---

## Verdict: Issues

1 finding in 1 dimension (interface contract conformance). It is incremental and, by the spec's own `[ASSUMED]` command-surface clause, resolvable as a documentation alignment rather than a code change — every behavioral and validation check passes. All 10 driving/behavioral scenarios are covered (and pass executably), all 4 task acceptance criteria are met, all 9 non-behaviors are absent, the `@wip` lifecycle is correct, and all 5 held-out `@validation` scenarios trace to implementation.

The implementation conforms to spec.md (the behavioral authority); it diverges only from the literal command-path wording in the derived interface/tasks prose, which went stale relative to My Roles (012)'s actual `me <noun>` shape.

---

## Next Steps

1 finding (F-1) to address — a documentation alignment, not a code fix. Suggested: update the `my actions` / `my` parent references in `interface-cli.md`, `interface-spec.md`, and `tasks.md` to `me actions` / `me` parent (authorized by spec.md § Assumptions), then re-validate. The developer owns whether to do this before or after merge; the finding is recorded transparently here and (per the user's request) in the PR description, so shipping as-is is a conscious, visible choice rather than a hidden gap.
