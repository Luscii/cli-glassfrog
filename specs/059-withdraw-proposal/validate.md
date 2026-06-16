# Validate: Withdraw Proposal

**Feature**: 059-withdraw-proposal
**Round**: 1 of 3
**Date**: 2026-06-15
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/proposal-write-flow/withdraw-proposal.feature, PROJECT.md
**Implementation files**: `internal/cli/proposal_withdraw.go` (new), `internal/cli/proposal.go` (one registration + group `Short` widening); tests `internal/cli/proposal_withdraw_test.go`, `internal/cli/proposal_withdraw_bdd_test.go`

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

**Status**: Pass (11 of 11 behavioral scenarios covered)

All driving scenarios are referenced by the checked tasks (T001 the command, T002 the executable acceptance) and have identifiable code paths in `runProposalWithdraw` / `newProposalWithdrawCommand`. Every behavioral scenario passes in `TestProposalWithdrawFeatures` (11 scenarios, 62 steps).

| Scenario | Status | Implementation |
|---|---|---|
| A circulating proposal is withdrawn back to draft | ✓ Covered | `proposal_withdraw.go` success branch → `writeHuman`/`ProposalView` |
| A disallowed transition fails with the API status (422) | ✓ Covered | `proposal_withdraw.go` → `reportFailure` (no interception) |
| An unknown proposal id fails with the API status (404) | ✓ Covered | `proposal_withdraw.go` → `reportFailure` (no interception) |
| A missing token fails as a not-authenticated usage error | ✓ Covered | shared `reportFailure`/`classifyClientError` fail-safe → `UsageError(2)` |
| A missing proposal id is rejected before any request | ✓ Covered | `cobra.ExactArgs(1)` on the leaf, no request |
| A Premium-gated refusal surfaces plainly (403) | ✓ Covered | `reportFailure` → `PermissionError(4)`, generic |
| A transport failure surfaces network-unavailable | ✓ Covered | `reportFailure` → `NetworkUnavailable(6)` |
| A rate-limited withdraw is surfaced, not silently retried (429) | ✓ Covered | `NewRetryExecutor` + 017 `isSafeMethod` (POST not retried) |
| An invalid output format is rejected before any request | ✓ Covered | `resolveRenderTarget` first, fail-fast `UsageError(2)`, no request |
| The withdrawn proposal is rendered as JSON | ✓ Covered | machine-format branch → `output.RenderSuccess` over raw bytes |
| The withdrawn proposal carries the cleared deadline and updated transitions | ✓ Covered | `ProposalView{Proposal: doc.Data}` → `proposal.full` (`Deadline: (none)`, `Transitions: propose`) |

---

## Acceptance Criteria

**Status**: Pass (both checked tasks' criteria met)

- **T001** (`proposal withdraw <prp-id>` command): every criterion verified by 15 unit tests in `proposal_withdraw_test.go` — bodyless `POST /proposals/{id}/withdraw` with `out == &Document[Proposal]` and no prior GET; `-o json` raw `{data: Proposal}` vs human projection; `422`/`404` as `APIError(3)` (not folded to success); generic Premium `403` → `PermissionError(4)`; `401`→4, `429`→`RateLimited(5)` not retried, transport→`NetworkUnavailable(6)`; leaf declares no flags (stray `--force`/`--yes`/`--changes` → `UsageError(2)`); missing/extra positional → `UsageError(2)`, no request; bad `--output` → `UsageError(2)`, no request; id path-escaped; no new `Outcome`/`ExitCode`/model/render key; siblings unchanged; `go build`/`go vet` clean.
- **T002** (executable acceptance): new `TestProposalWithdrawFeatures` godog suite whose `Paths` names only `withdraw-proposal.feature`; 11 behavioral scenarios passing; 4 `@validation` held `@wip`; reused propose-suite step phrasings; helpers return errors. `go test ./...` clean.

---

## Interface Contract Conformance

**Status**: Pass (the single `withdraw` surface conforms)

| Surface element | Status | Evidence |
|---|---|---|
| `proposal withdraw <prp-id>` leaf, `ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage` | ✓ Conformant | `newProposalWithdrawCommand` |
| No flags (no `--force`/`--yes`/`--changes`) | ✓ Conformant | leaf declares none; inherits persistent `--base-url`/`-o` only |
| `POST /proposals/{id}/withdraw`, bodyless, no `Content-Type`, `out == &Document[Proposal]` | ✓ Conformant | `apiclient.Request{Method: POST, Path: …/withdraw}`, no `Body`/`ContentType` |
| No prior `GET`, no `If-Match` | ✓ Conformant | one `Execute`; no `IfMatch` set; tests assert `calls == 1` |
| Dispatch order: args/flags → `--output` resolve (no request on bad value) → assemble → send | ✓ Conformant | `resolveRenderTarget` first, then `assemble`/`newClient`/`Execute` |
| Output: decode + render via singular `proposal` path (no new render key) | ✓ Conformant | `output.RenderSuccess` (machine) / `writeHuman` over `ProposalView` (human) |
| Error table: 422/404→`APIError(3)`, 401/403→`PermissionError(4)` generic, 429→`RateLimited(5)` no retry, transport→`NetworkUnavailable(6)`, usage→2; no status interception | ✓ Conformant | `reportFailure` chokepoint, unchanged |
| Group `Short` widened to name `withdraw` | ✓ Conformant | `newProposalCommand` group `Short` and leaf registration |

---

## Non-Behavior Absence

**Status**: Pass (all 10 exclusions respected)

| Non-behavior | Status | Evidence |
|---|---|---|
| No create/list/get/propose/respond | ✓ Absent | only the `withdraw` leaf is added; siblings untouched |
| No re-edit/re-propose after withdraw | ✓ Absent | single transition; no chained edit/propose call |
| No client-side pre-read of `available_transitions` | ✓ Absent | exactly one request, no prior `GET` (asserted) |
| No `404`/`422` as success | ✓ Absent | every non-2xx routes to `reportFailure`; no `errors.As`-for-404 divert |
| No plan-aware `403` message | ✓ Absent | generic `PermissionError(4)`; tests ban "plan"/"premium"/"upgrade" |
| No `If-Match` / concurrency guard | ✓ Absent | `IfMatch` unset; tests assert no `If-Match` header |
| No narration of destructive side effects | ✓ Absent | renders returned proposal only; tests ban "deleted"/"discard" narration |
| No confirmation / `--force` / `--yes` | ✓ Absent | flagless leaf; stray `--force`/`--yes` → cobra usage error |
| No private output-format flag / fixed raw default | ✓ Absent | reuses 020/035 `resolveRenderTarget` |
| No own base-URL/token/error-typing/exit-code path | ✓ Absent | reuses `assemble`/`newClient`/`reportFailure`/frozen registry |

---

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral scenarios (referenced by checked T002) carry no `@wip` and run under the `~@wip` filter. The 4 `@validation` scenarios remain `@wip` — correct: they are not referenced by any checked task and are held for this validate pass.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation)

The `@validation` step definitions are intentionally not written (held `@wip`), so these are traced by code inspection — the canonical validate baseline — not executed.

| Scenario | Status | Trace |
|---|---|---|
| The withdrawn result is the server's proposal unembellished | ✓ Satisfied | `runProposalWithdraw` renders `doc.Data`/raw bytes directly — no synthesized fields, no side-effect narration; unit test asserts no "deleted" string in output |
| The withdraw output carries no raw API envelope under a human format | ✓ Satisfied | human path takes `writeHuman` over the reshaped `ProposalView`; the raw `{data:…}` branch is reached only for a machine format (`rt.format.MachineFormat()` split); tests assert human output lacks raw `data` and json lacks `Transitions:` labels |
| The withdraw issues one request and reads nothing first | ✓ Satisfied | a single `apiclient.Request` is built and `Execute`d; no prior `GET`; BDD `postedBodylessToWithdrawTransition` and unit tests assert `calls == 1` |
| A 404 and a 422 are surfaced as real failures | ✓ Satisfied | both route through `reportFailure` to `APIError(3)`, exit non-zero, empty stdout; unit tests `…UnknownProposalIsAPIError` / `…DisallowedTransitionIsAPIError` confirm no success result |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 4 held-out validation scenarios trace to clear code paths. The implementation is a faithful structural twin of the landed `propose` leaf on the `/withdraw` sub-path: a flagless, bodyless `POST` that decodes and renders the returned `draft` proposal, intercepts no status (the inverse of discard's `404`-as-success), keeps the Premium `403` generic, never auto-retries the `POST`, and adds no confirmation/`--force` guard — conforming to plan ADR-1 and ADR-2 and to every spec non-behavior. No new `Outcome`/`ExitCode`/model/render key/transport surface was introduced. `go build ./...`, `go vet ./...`, and `go test ./...` are clean.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge (`gh pr create --base main`). The specification loop for 059 is closed.
