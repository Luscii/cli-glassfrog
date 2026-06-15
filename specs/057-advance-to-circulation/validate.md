# Validate: Advance to Circulation

**Feature**: 057-advance-to-circulation
**Round**: 1 of 3
**Date**: 2026-06-15
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/proposal-write-flow/advance-to-circulation.feature, PROJECT.md
**Implementation files**: `internal/cli/proposal_propose.go` (command + pure run), `internal/cli/proposal.go` (group wiring), `internal/cli/proposal_propose_test.go` (16 unit tests), `internal/cli/proposal_advance_bdd_test.go` (godog suite), `internal/render/templates/proposal.full.tmpl` + `internal/render/proposal_test.go` (shared render grown by one line)

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

**Status**: Pass (9 of 9 behavioral scenarios covered)

Every behavioral scenario has an identifiable code path in `runProposalPropose` and an executable godog binding in `proposal_advance_bdd_test.go` (11 scenarios, 62 steps, passing).

| Scenario | Status | Implementation |
|---|---|---|
| Advance a draft proposal into circulation | ✓ Covered | `proposal_propose.go:runProposalPropose` (bodyless `POST /proposals/{id}/propose`, decode + render); BDD `postedBodylessToProposeTransition` |
| Advanced proposal rendered as JSON | ✓ Covered | `runProposalPropose` machine branch → `output.RenderSuccess(raw)`; BDD `printedAsJSON`, unit `TestRunProposalPropose_StructuredJSONEmitsRawPayload` |
| Result reflects server-set deadline & implicit response | ✓ Covered | human branch → `writeHuman(ProposalView)`; `proposal.full.tmpl` renders `Deadline` + `Responses`; BDD `carriesResponseDeadline`/`reflectsImplicitNoObjection` |
| Transition not allowed is a failure (422) | ✓ Covered | error branch → `reportFailure` (no interception); unit `…_DisallowedTransitionIsAPIError` |
| Proposal id does not exist (404) | ✓ Covered | error branch → `reportFailure`; unit `…_UnknownProposalIsAPIError` (stdout empty, exit 3) |
| No credential surfaces not-authenticated | ✓ Covered | `reportFailure` not-authenticated fail-safe; unit `…_NoCredentialsIsUsageError` (exit 2, no request) |
| Missing proposal id rejected before request | ✓ Covered | `cobra.ExactArgs(1)`; unit `…_RequiresExactlyOneArg` (no request) |
| Premium not enabled (403) plain refusal | ✓ Covered | `reportFailure` → `PermissionError`; unit `…_PremiumDeniedIsPermission` (no plan message) |
| Transport failure → network-unavailable | ✓ Covered | `reportFailure` → `NetworkUnavailable`; unit `…_TransportErrorIsNetworkUnavailable` |

(Architecture-informed behaviors — `429` surfaced once and not auto-retried, invalid `--output` fail-fast — are also covered: `…_RateLimitSurfacedNotRetried`, `…_BadOutputUsageErrorNoRequest`.)

## Acceptance Criteria

**Status**: Pass (T001, T002 — both checked, all criteria met)

**T001** (the command): every criterion has unit-test evidence in `proposal_propose_test.go` — the one bodyless `POST` with no prior `GET`/no `Content-Type`/no `If-Match` and a `200` rendering exit 0; `-o json` raw `{data: Proposal}` verbatim; `404`/`422` as `APIError(3)` failures naming the status with empty stdout; the Premium `403` as a generic `PermissionError(4)` (no plan message); `429` as `RateLimited(5)` with a single request (no auto-retry); transport → `NetworkUnavailable(6)`; missing/extra positional and stray flag → `UsageError(2)` with no request; bad `--output` → `UsageError(2)` with no request; no token in output; all branches offline. No new `Outcome`/`ExitCode`/model/render key/root flag added; the sibling `create` leaf code is unchanged.

**T002** (executable acceptance): the new `TestProposalProposeFeatures` suite names only `advance-to-circulation.feature`, reports its own count (11 scenarios / 62 steps passing), and the 4 `@validation` scenarios remain `@wip`. `go build ./...`, `go vet ./...`, and the full suite run clean.

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface (interface-cli.md) | Status | Implementation |
|---|---|---|
| `proposal` group (no action, widened `Short` to name `propose`) | ✓ Conformant | `proposal.go:newProposalCommand` attaches `propose`; group `Short` widened |
| `proposal propose <prp-id>` — `ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`, **no flags** | ✓ Conformant | `proposal.go:newProposalProposeCommand` |
| Bodyless `POST /proposals/{id}/propose`, `out == &Document[Proposal]`, no prior `GET`, no `If-Match` | ✓ Conformant | `runProposalPropose` (single `Execute`, `Method: POST`, no `Body`/`ContentType`/`IfMatch`) |
| Output: `full` shows lifecycle timestamps incl. `response_deadline`; `json`/`yaml` raw verbatim | ✓ Conformant | `proposal.full.tmpl` (grown with the `Deadline` line); machine branch emits raw bytes |
| Error table (`404`/`422`→`APIError 3`; `401`/`403`→`PermissionError 4`; `429`→`RateLimited 5`; transport→6; bad `--output`/cobra→`UsageError 2`) | ✓ Conformant | all routed through the shared `reportFailure`/`classifyClientError`, no interception |
| No new `Outcome`/`ExitCode` | ✓ Conformant | none added |

Note: interface-cli.md § Output explicitly lists `response_deadline` among `full`'s fields. The landed 055 `proposal.full` template omitted it; the implementation grew the shared template by one guarded line to make the surface conformant (recorded in tasks.md T002 + LEARNINGS). This is the interface-required behavior, not an extra capability — no new render key, model, or `ProposalView` field.

## Non-Behavior Absence

**Status**: Pass (9 of 9 exclusions respected)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not create/list/get/withdraw/respond | ✓ Absent | leaf issues only `POST …/propose`; no other endpoint |
| Must not pre-read `available_transitions` | ✓ Absent | exactly one `Execute`, no `GET` (validation scenario 1) |
| Must not choose backend dispatch path | ✓ Absent | plain bodyless `propose`, no path/param choice |
| Must not treat `404`/`422` as success | ✓ Absent | no status interception — every non-2xx → `reportFailure` |
| Must not give Premium `403` plan-aware messaging | ✓ Absent | generic `PermissionError(4)`; unit + BDD assert no plan message |
| Must not send `If-Match` | ✓ Absent | `Request` sets no `IfMatch`; tests assert `lastIfMatch == ""` |
| Must not interpret/narrate side effects | ✓ Absent | renders only the returned proposal; BDD `noNotificationNarration` |
| Must not prompt for confirmation / `--force` | ✓ Absent | no such flag declared |
| Must not emit raw JSON as fixed default nor own format flag | ✓ Absent | dispatches via 020 `resolveRenderTarget`; no private flag |

## @wip Lifecycle Completion

**Status**: Pass

The 11 behavioral scenarios referenced by checked tasks have had `@wip` removed and pass under the `~@wip` filter. The 4 `@validation` scenarios correctly retain `@wip` (held for this skill; not referenced as behavioral by a checked task) — `advance-to-circulation.feature:125,133,147,156`.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| The advance issues one request and reads nothing first | ✓ Satisfied | `runProposalPropose` runs exactly one `exec.Execute` (machine/human branches are mutually exclusive); no `GET`/`MethodGet` in the file. Unit & BDD assert `calls == 1` |
| A 404 and a 422 are real failures | ✓ Satisfied | both route through `reportFailure` with no interception → `APIError(3)`, exit non-zero, empty stdout; no non-2xx folded to success (the inverse of discard's `404`-as-success) |
| The result is the server's proposal, unembellished | ✓ Satisfied | machine → raw `{data: Proposal}` bytes verbatim (`RenderSuccess`); human → `ProposalView` over model fields only (`status`, `response_deadline`, `response_summary`, `available_transitions`, `changes`). No synthesized fields, no side-effect narration. The added `Deadline` line surfaces the server's `response_deadline` — a returned field this scenario explicitly expects rendered, not fabrication |
| Output is structured, not pre-rendered | ✓ Satisfied | the command supplies structured data (`Document[Proposal]`/raw bytes) and declares no format flag; all four formats dispatch from the same result via 020. Human `full` shows the reshaped projection with no raw `data` envelope (BDD `printedAsJSON` asserts no `Transitions:` label leaks into JSON, and the human path emits no `"data"` envelope) |

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios are satisfied through inspection and corroborated by the passing test suites (`go build`/`go vet`/`go test ./...` clean). The implementation conforms to the specification: the `propose` transition is a pure consumer of the sibling-landed `proposal` group/model/render path, issues exactly one bodyless server-authorized `POST` with no client pre-check, treats `404`/`422` as real failures, keeps the Premium `403` generic, and renders the returned proposal unembellished in every format.

One implementation note (not a finding): the shared `proposal.full` template was grown by one guarded `Deadline:` line to surface `response_deadline`. This was required for interface conformance (§ Output) and the "carries the deadline" behavioral scenario, and is exactly what validation scenario 3 expects rendered. It widens the shared render across the sibling `proposal create`/`get` `full` output (an additive, accurate line) — flagged for reviewer awareness, recorded in `.score/memory/LEARNINGS.md`.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The reviewer's attention is drawn to the one shared-render growth (the `Deadline:` line in `proposal.full.tmpl`), which also affects `proposal create`/`get` `full` output — additive and interface-required, but a shared surface worth a glance.
