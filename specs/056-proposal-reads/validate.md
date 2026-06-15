# Validate: Proposal Reads

**Feature**: 056-proposal-reads
**Round**: 1 of 3
**Date**: 2026-06-15
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/proposal-write-flow/proposal-reads.feature, PROJECT.md
**Implementation files**: `internal/cli/proposal_reads.go` (validator + list + get), `internal/cli/proposal.go` (group wiring), `internal/glassfrog/proposal.go` (model, created by 055), `internal/render/render.go` + `templates/proposal{,s}.{full,compact}.tmpl` (both render keys); tests: `internal/cli/proposal_reads_test.go`, `internal/cli/proposal_reads_bdd_test.go`, `internal/glassfrog/proposal_test.go`, `internal/render/proposal_test.go`

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

Guardian agent definition (`agents/guardian-agent.md`) was not deployed at the expected path — proceeded with the validate SKILL.md alone (reduced character consistency, not a blocked skill). Supplementary test execution was run: `go test ./internal/cli/ ./internal/render/ ./internal/glassfrog/` passes (incl. `TestProposalReadsFeatures` — 11 behavioral scenarios, 57 steps).

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 scenarios covered)

All driving scenarios in spec.md (and their concretized `.feature` forms) have identifiable implementation code paths.

| Scenario | Status | Implementation |
|---|---|---|
| List the proposals visible to the caller | ✓ Covered | `runProposalList` → `runProposalListWalk` `GET /proposals` via `paging.All`; `proposals` render key (`proposal_reads.go:133`) |
| Read a single proposal with full detail | ✓ Covered | `runProposalGet` `GET /proposals/{id}` → `Document[Proposal]` → singular `proposal` render (changes/response_summary/available_transitions) (`proposal_reads.go:412`) |
| Narrow the list to a circle and a status | ✓ Covered | `proposalsQuery` sends `role_id` + `status`; `--status` validated first (`proposal_reads.go:248`) |
| Proposal id does not exist (404) | ✓ Covered | id passed through unvalidated → `reportFailure` via shared classifier; `TestRunProposalGet_UnknownIdSurfacesAPIStatus` |
| No usable credential | ✓ Covered | `*AuthError{NoCredentials}` → `UsageError(2)` via `reportFailure`; `TestRunProposalList_NoCredentialsIsUsageError` |
| No proposals are visible | ✓ Covered | empty walk → `proposals` template `no proposals` line, exit 0; `TestRunProposalList_EmptyIsCleanSuccess` |
| Unsupported status value rejected before any request | ✓ Covered | `validateProposalStatus` pre-assembly; `TestRunProposalList_UnsupportedStatusNoRequest` (tripwire) |
| List filter on the single read is rejected | ✓ Covered | `get` declares no list flags → cobra unknown-flag `UsageError`; `TestProposalGetCommand_RejectsListFlags` (×7) |
| Paginated list with first-page opt-out | ✓ Covered | `runProposalListFirstPage` single page + `moreProposalsNote`, exit 0; `TestRunProposalList_FirstPageSignalsMore` |

---

## Acceptance Criteria

**Status**: Pass (6 of 6 tasks checked; criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 model | ✓ Met | `glassfrog.Proposal`/`ProposalChange`(free-form `Fields`)/`ResponseSummary` (created by 055); follower added `Page[Proposal]` decode + reflect anti-attribution tests (`proposal_test.go`) |
| T002 render | ✓ Met | grown singular `proposal.full`/`.compact` (changes by type via `changeProps`, proposed/deadline/accepted, expected/recv, response total); new plural `proposals` key + `ProposalsView` + `ResourceProposals`, both in guarded `builtinResources`; `TestRegistry_AllBuiltinsResolve` passes |
| T003 validator | ✓ Met | `validateProposalStatus` + single-sourced `supportedProposalStatuses` (incl. `draft_with_conflicts`) in `proposal_reads.go`, not shared `status.go`; 3 unit tests |
| T004 list | ✓ Met | `proposal list` `cobra.NoArgs`, five filters (`--status` local, four passed through), `--first-page`/`--per-page` completeness; ~11 unit tests + positional-reject test |
| T005 get | ✓ Met | `proposal get <prp-id>` `cobra.ExactArgs(1)`, no list flags; raw `{data: Proposal}` for machine / singular render for human; 6 unit tests |
| T006 BDD | ✓ Met | `TestProposalReadsFeatures` Paths → `proposal-reads.feature` only; 11 behavioral pass, 4 `@validation` held `@wip` |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface | Status | Evidence |
|---|---|---|
| `proposal` group (non-runnable, parents create/list/get) | ✓ Conformant | `newProposalCommand` attaches `create` (055) + `list` + `get`; `TestProposalCommand_GroupRegistersUnderGuard` |
| `proposal list` (`cobra.NoArgs`, global; 7 list flags) | ✓ Conformant | `newProposalListCommand`; `--status`/`--role-id`/`--proposer-id`/`--proposed-after`/`--accepted-after`/`--first-page`/`--per-page` declared only here |
| `proposal get <prp-id>` (`cobra.ExactArgs(1)`, no list flags) | ✓ Conformant | `newProposalGetCommand`; list flags rejected by cobra unknown-flag handling (structural guard) |
| Output: list → aggregated `{data:[…]}` (machine) / `proposals` projection (human); single → raw `{data: Proposal}` / `proposal` projection | ✓ Conformant | `aggregateRawData`/`output.RenderSuccess` + `writeHuman`; empty list → `no proposals`, exit 0 |
| Error table (Auth→2, Transport→6, ResponseError→3/4/5, base-URL/`--output`→2, unsupported `--status`→2, cobra→2) | ✓ Conformant | routed through `reportFailure`/`classifyClientError`; no new `Outcome`/`ExitCode` (lines 130, 406) |
| Validation order: `--output` resolved, then `--status`, before assembly | ✓ Conformant | `runProposalList` resolves render target then validates status before `assemble` |

---

## Non-Behavior Absence

**Status**: Pass (8 of 8 non-behaviors absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No create/advance/withdraw/respond | ✓ Absent | read path issues only `http.MethodGet` (`proposal_reads.go:168,424`); no POST/PUT/DELETE |
| Must not act on `available_transitions` | ✓ Absent | transitions rendered (`Transitions:` line) but no transition request is ever constructed |
| No per-person response attribution | ✓ Absent | `ResponseSummary` is exactly 3 int counts (reflect guard); templates surface only Total/NoObjection/BringToMeeting; structured = raw server bytes (API exposes aggregate only) |
| Must not interpret `changes[]` | ✓ Absent | `ProposalChange.Fields` is `map[string]any` free-form; `changeProps` renders verbatim compact JSON, no per-type schema |
| `--status` only a filter, never a status write | ✓ Absent | `--status` sent as a query parameter only; no status field written |
| No raw-JSON default / own format flag | ✓ Absent | no `--output`/`--format` declared by the reads; routes through 020 |
| No base-URL/token/header/exit-code reinvention | ✓ Absent | reuses `resolveRenderTarget`/`assemble`/`newClient`/`classifyClientError`; never reads `ctx.Cred.Token` |
| No interpret/summarize/advise | ✓ Absent | commands produce structured proposal data only; no advisory text |

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 scenarios traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| The read surface never reaches into the write verbs | ✓ Satisfied | `list`/`get` issue only GET; `available_transitions` printed, never invoked; help (`Short`/`Long`) advertises no mutation — `get`'s Long states transitions are "printed but never invoked — advancing a proposal is a separate write command" |
| No per-person response attribution is reconstructed | ✓ Satisfied | every format shows only aggregate counts; `ResponseSummary` carries no per-person field (reflect guard `TestResponseSummary_AggregateOnly`); no template field attributes a response to an individual |
| An unsupported status costs no request | ✓ Satisfied | `validateProposalStatus` runs before `assemble`/`newClient`; transport tripwire `TestRunProposalList_UnsupportedStatusNoRequest` confirms `calls == 0` |
| Output is structured, not pre-rendered | ✓ Satisfied | reads define no format flag of their own; supply structured data (`ProposalsView`/`ProposalView` human, `json.RawMessage` machine); all four formats render from the same fetched result via 020 |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 4 held-out `@validation` scenarios are traced to clear implementation code paths. The implementation conforms to its specification — including the follower-grows-not-duplicates coordination with Proposal Creation (055): the shared `proposal` group, `glassfrog.Proposal` model, and singular `proposal` render key were reused and grown, never duplicated, and structured output stays faithful regardless of model coverage.

One observation (not a finding — outside validate's behavioral conformance scope): growing the shared singular `proposal.full`/`.compact` to the 056 detail block also changed `proposal create`'s (055) human output, dropping the prior `Created:`/`Updated:` lines in favor of the interface-pinned fields. This is the intended grow-not-duplicate behavior; structured `-o json`/`yaml` for create is unchanged. Worth a glance in PR review for anyone who pinned 055's exact human output downstream.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The 4 `@validation` scenarios remain `@wip` in the feature file by design — they are held-out verification, satisfied here by inspection rather than by an executable un-`@wip` path.
