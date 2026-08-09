# Validate: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Round**: 2 of 3
**Date**: 2026-08-09
**Verdict**: Issues
**Artifacts loaded**: `spec.md`, `plan.md`, `tasks.md`, `interface-cli.md`, `interface-spec.md`, `features/success-reported-for-a-dead-proposal/post-create-validity-read.feature`, `PROJECT.md`, previous `validate.md` (Round 1)
**Implementation files**: 6 production files — `internal/glassfrog/proposal.go`, `internal/render/render.go`, `internal/render/usertemplate.go`, `internal/render/templates/proposal-created.{full,compact}.tmpl`, `internal/cli/proposal.go` — plus 4 test files. Commits `bf97834`…`70ca2ea` (7 tasks + 1 review fix).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass (9 of 9 covered; 1 wording ambiguity) | 1 |
| Acceptance criteria | ✗ Fail (37 of 38 criteria met) | 1 |
| Interface contract conformance | ✗ Fail (CLI accord conformant except one table; Go accord diverged) | 2 |
| Non-behavior absence | ✓ Pass (11 of 11 absent) | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 4; 1 scoping ambiguity) | 1 |

**Total**: 5 dimensions checked, 3 passed, 5 findings (4 carried forward, 1 new)

No dimension was skipped. `go test ./...` passes across all 11 packages; `gofmt -l .` and `golangci-lint run` are clean; the feature's godog suite executes 15 scenarios / 100 steps.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 covered) — one ambiguity carried forward as F-3

Re-traced against current code. The Round 1 mapping holds unchanged; the read-back helper's new identity guard narrows *when* a read-back counts as answered but changes no scenario's outcome.

| Scenario | Status | Implementation |
|---|---|---|
| A valid draft reports its verdict alongside its id | ✓ Covered | `runProposalCreate` (human arm) → `render.NewProposalVerdict` → `proposal-created.full.tmpl` |
| A created-but-invalid draft surfaces the server's refusal | ✓ Covered | Same path; alerts block; read-back's `available_transitions` render the empty set |
| An agent parses the verdict out of machine-readable output | ✓ Covered (see F-3) | Machine arm emits the read-back's raw `{data}`; `writeMachineVerdictAdvisory` renders provenance on stderr |
| The read-back cannot reach the server | ✓ Covered | `readBackVerdictReason` transport arm; outcome stays `Success` |
| The create itself is rejected | ✓ Covered | `reportFailure` returns before the read-back stage |
| The server reports no verdict at all | ✓ Covered | `Valid == nil` arm → "not reported by the server" |
| A valid draft with no available transitions | ✓ Covered | Validity in the verdict block; transitions stay a line of the shared body |
| A status that disagrees with the verdict | ✓ Covered | Status renders from the server's document; `NewProposalVerdict` never reads it |
| The read-back exhausts the hour's request budget | ✓ Covered | 429 arm names the exhausted request budget |

---

## Acceptance Criteria

**Status**: Fail (37 of 38 criteria met across 7 checked tasks; 1 finding carried forward)

All 7 tasks remain checked. Criteria re-verified against implementation and tests.

| Task | Criteria | Status |
|---|---|---|
| T001 — verdict fields + `ValidationAlert` | 6 | ✓ All met |
| T002 — created view, verdict projection, label mapping | 7 | ✓ All met |
| T003 — resource registration + delegating templates | 7 | ✓ All met |
| T004 — isolated read-back helper | 6 | ✓ All met |
| T005 — human render path | 8 | ✗ 7 met, 1 not strictly met (F-1) |
| T006 — machine render path | 8 | ✓ All met |
| T007 — test reconciliation + BDD suite | 5 | ✓ All met |

T004's criterion "a wire failure, a non-2xx, a post-retry 429, and an undecodable body each return a distinct reason" still holds: those four families retain four distinct reasons. The identity guard added since Round 1 is a *fifth* cause that deliberately reuses the fourth family's reason — see F-5, which is an accord-completeness finding rather than a criterion failure.

---

## Interface Contract Conformance

**Status**: Fail (2 findings — one carried forward, one new)

`interface-cli.md` — the operator-facing accord — remains conformant on every surface checked in Round 1 (command synopsis with no opt-out flag, the four verdict states, both human formats, machine stdout, the user-template view, the format-aware stderr advisory, and the unchanged exit-code set). One table within it is now incomplete: the cause enumeration behind the reason vocabulary (F-5).

`interface-spec.md` — the Go-surface accord — still diverges at the two points recorded in Round 1: `funcMap` is listed under **Unchanged** while carrying an `include` helper, and `proposal-created.compact.tmpl` differs from the pinned sketch (F-2). Verified still present: one `include` entry in `render.go`, and `interface-spec.md:133` still lists `funcMap` as unchanged.

---

## Non-Behavior Absence

**Status**: Pass (11 of 11 exclusions absent)

Re-verified all eleven. The material change since Round 1 is the fifth row, which Round 1 marked Absent on incomplete evidence — see the Improvement Summary.

| Non-behavior | Status | Evidence |
|---|---|---|
| No local validity determination | ✓ Absent | `NewProposalVerdict` branches only on `valid *bool` and the unavailable reason; the new identity guard compares ids and derives no verdict — a rejected read-back yields a zero proposal plus a reason, rendering as `unavailable` |
| No outcome/exit-code change when not valid | ✓ Absent | `Success` in every verdict state |
| Fields not presented as published contract | ✓ Absent | No help text, doc, or skill references either field; both confirmed absent from `spec/glassfrog-api-v5.yaml` |
| Missing verdict not treated as favourable | ✓ Absent | `Valid == nil` → "not reported by the server" |
| **Created `prp_` id never withheld** | ✓ Absent (now guarded) | Previously held only on the four reason-bearing failure paths. A 2xx decoding to a zero or foreign proposal produced an *empty* reason, so both arms substituted that document — the human body rendered `  []` and the machine arm emitted the empty document, dropping the id. Closed in `399b357` by requiring `doc.Data.ID == id`; regressions at helper, human-arm, and machine-arm level |
| No read-back after a rejected create | ✓ Absent | `reportFailure` returns before the read-back stage; asserted |
| No opt-out of the read-back | ✓ Absent | No flag added |
| No polling or extra retry | ✓ Absent | One `Execute` through the shared `RetryExecutor` |
| Not extended to other proposal writes | ✓ Absent | Two call sites, both inside `runProposalCreate` |
| Verdict not rendered in the proposal list | ✓ Absent | `ResourceProposals` untouched; byte-identical guard test |
| Nothing withheld from `proposal get --output json` | ✓ Absent | `internal/cli/proposal_reads.go` unchanged |

---

## @wip Lifecycle Completion

**Status**: Pass

Unchanged from Round 1: 19 scenarios, 15 executing with `@wip` cleared, 4 `@wip` tags all on `@validation` scenarios held for this skill per `tasks.md` § Scenario disposition.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 4 traced to implementation; 1 scoping ambiguity — F-4)

Each scenario was re-traced independently against current code rather than inherited from Round 1.

| Scenario | Status | Trace |
|---|---|---|
| The reported result names the read that produced the verdict | ✓ Satisfied | `full`: `Verdict source: read-back of <prp_id> after create`. `compact`/user template: stderr prose names the read. Machine: `verdict_source.read_back` + `proposal_id` |
| No verdict is derived from the change set, status, transitions, or alerts | ✓ Satisfied | `NewProposalVerdict` reads only `valid` and the reason; neither template consults status or transitions for the validity line; the alert count appends in *either* validity state. The new identity guard is an identity comparison, not a validity derivation |
| The verdict fields are never presented as published contract | ✓ Satisfied | Absent from help, `docs/`, `plugin/`, and the vendored contract; field comments state the non-declaration explicitly |
| Every verdict state is distinguishable in every output format | ✓ Satisfied for the four built-in formats; ambiguous for the user-template form | `full`/`compact` pinned by golden tests; `json`/`yaml` pinned by the four-case table over `data.valid` + `verdict_source.read_back`. See F-4 |

**Observation (not a conformance failure, carried from Round 1)**: `docs/guides/proposals/how-to-manage-a-governance-proposal.md:37` still describes the create's result as id + status — accurate but incomplete now. `/score:document` owns that surface.

---

## Findings

### F-1 *(carried forward from Round 1, unchanged)*: T005's "output is unchanged" criterion is over-claimed, and its test cannot detect the difference

- **Dimension**: Acceptance criteria
- **Source**: `tasks.md` § T005 — "A user-supplied template written against the pre-change view still renders — every field path that resolved before still resolves, **and its output is unchanged by the verdict's addition**"
- **Implementation**: `internal/cli/proposal.go:runProposalCreate` — `proposal := doc.Data; if reason == "" { proposal = readBack }`; test `TestRunProposalCreate_PreChangeUserTemplateStillRenders`
- **Gap**: When the read-back answers, the human arm renders the **read-back's** proposal. Field paths all still resolve, but values now come from the second document, so a pre-074 template referencing `.Proposal.AvailableTransitions`, `.Proposal.Status`, `.Proposal.UpdatedAt`, or the response counts can print something different. The pinning test projects only `.Proposal.ID`, `.Proposal.Status`, and `len .Proposal.Changes` against fixtures that agree on all three — while those same fixtures disagree on transitions (`["propose"]` in the create body vs `["propose","withdraw"]` in the read-back). Verified still present in Round 2: both the swap and the non-discriminating fixture are unchanged.

  The swap is not the defect — `plan.md` § Verdict Assembly sources transitions from the read-back, and the invalid-draft driving scenario needs the empty set only the read-back reports. Resolution is to narrow the criterion (field paths resolve; values reflect the read-back where one was obtained) and re-point the test at fixtures that differ.

### F-2 *(carried forward from Round 1, unchanged)*: `interface-spec.md`'s Go-surface accord no longer matches the implementation

- **Dimension**: Interface contract conformance
- **Source**: `interface-spec.md:133` — "**Unchanged**: … `funcMap` …"; and the pinned compact template `{{template "proposal.compact.tmpl" .}}  {{.Verdict.Compact}}`
- **Implementation**: `internal/render/render.go` (`funcMap` carries `include`; `templates` and `userTemplateBase` assigned in `init()`); `internal/render/templates/proposal-created.compact.tmpl` reads `{{trimSpace (include "proposal.compact.tmpl" .)}}  {{.Verdict.Compact}}`
- **Gap**: Three stated-unchanged items changed. The implementation correctly preferred `interface-cli.md`'s one-line compact contract over the Go sketch, and `LEARNINGS.md` records why; what remains is that `interface-spec.md` misdescribes the surface a future reader would implement or review against. Two specifics belong in the accord: `include` is reachable from user templates (035) — pure and data-only, so the sandbox holds, but the callable surface is wider than any artifact documents — and the `init()` assignment that breaks the resulting initialization cycle.

### F-3 *(carried forward from Round 1, unchanged)*: The machine-output driving scenario reads as though provenance rides the emitted document

- **Dimension**: Driving scenario coverage
- **Source**: `spec.md` § Driving Scenarios — "Then the emitted document carries the validity verdict … / **And it carries the provenance of the verdict**"
- **Implementation**: `writeMachineVerdictAdvisory` — provenance is a `{"verdict_source": …}` document on **stderr**; stdout carries the server's document with nothing added
- **Gap**: Read literally, "it" is the emitted document, and the implementation deliberately does not put provenance there (ADR-5 rejected composing an envelope, to preserve 018's verbatim contract). ADR-5, `interface-cli.md` § stderr, the feature file's own step, and `spec.md` § Behavioral Accord ("the *reported result* carries the provenance") all settle the reading the implementation follows. One clarifying edit to the scenario line would make all four artifacts read the same way.

### F-4 *(carried forward from Round 1, unchanged)*: Validation scenario "every output format" does not scope the user-template form

- **Dimension**: Validation scenarios
- **Source**: feature file § "Every verdict state is distinguishable in every output format" — "Given each output format the create supports"; `interface-cli.md` enumerates **six** output forms including a template file path
- **Implementation**: human arm — a user template renders whatever its author wrote; the accompanying stderr advisory is the human **prose** line, byte-identical across `valid`, `not valid`, and `not reported` (all three are `read_back: true`)
- **Gap**: For the four built-in formats the scenario holds and is test-pinned. For a caller-authored template that does not reference `.Verdict`, three of the four states are indistinguishable. `interface-cli.md` § "stdout — user template" takes the position that the verdict is *available* rather than rendered, which is arguably the most the command can promise; the artifacts do not state which reading governs. Resolution is a scoping sentence in the scenario or the accord.

### F-5 *(new in Round 2)*: The reason-cause table omits the identity/shape mismatch the new guard introduced

- **Dimension**: Interface contract conformance
- **Source**: `interface-cli.md` § "The read-back's failures never reach an exit code" — a five-row table mapping each **cause** to its reason text, with `the read-back response could not be read` attributed to exactly one cause: "Undecodable read-back body"
- **Implementation**: `internal/cli/proposal.go:readBackProposalVerdict` — two sites now return that reason: the unmarshal-error branch (the enumerated cause) and the `doc.Data.ID != id` guard added in `399b357`, which fires for a 2xx that decodes cleanly into a zero or foreign proposal
- **Gap**: The implementation now has **six** causes against the accord's five. Reusing the existing reason string was the right call — it keeps the reason *vocabulary* closed, which the accord and the four-state machine contract both depend on — but a reader consulting the table to interpret `the read-back response could not be read` will conclude the body failed to parse, when it may have parsed perfectly and simply not been the requested proposal. That matters operationally: the two causes suggest different follow-ups (a malformed response vs. a proxy or gateway returning the wrong record). Resolution is one row (or a widened cause cell) in the table — no code change, and no new reason string.

---

## Verdict: Issues

5 findings across 3 conformance dimensions and the validation-scenario set — 4 carried forward unchanged from Round 1, 1 new.

The distribution matters more than the count: **four of the five are artifact corrections**, and the fifth (F-1) is an over-claimed acceptance criterion whose remedy is to narrow the claim and sharpen one test, not to change behavior. No behavioral section is unimplemented, no interface surface is missing, every driving scenario traces to code, and all eleven non-behaviors are absent — including the id-retention guarantee that Round 1 marked Absent on incomplete evidence and that is now genuinely enforced.

This is not Not Ready: the gaps are incremental and none reveals a missing implementation. It is not Ready either, because two accords now misdescribe the code they govern and one criterion claims more than the design delivers.

---

## Changes Since Previous Run

**Round**: 2 (previous: Round 1, verdict Issues)

### Resolved (0 of 4 findings)

None. No artifact edits were made between rounds; the only code change was the review fix below.

### Remaining (4 findings)

- **F-1** (Round 1): T005's "output is unchanged" criterion — **still present**, verified unchanged. The document swap and the non-discriminating test fixture are both as Round 1 found them.
- **F-2** (Round 1): `interface-spec.md` Go-surface drift — **still present**. `include` remains in `funcMap`; `interface-spec.md:133` still lists `funcMap` as unchanged.
- **F-3** (Round 1): spec driving-scenario provenance wording — **still present**. No artifact edit.
- **F-4** (Round 1): validation-scenario format scoping — **still present**. No artifact edit.

### New (1 finding)

- **F-5** (Round 2): the reason-cause table omits the identity/shape mismatch — **new**, introduced by the Round 1→2 code change. A consequence of the fix below, not a defect in it.

### Closed outside the finding list — a Round 1 miss

Round 1 marked the non-behavior *"must not withhold, delay, or overwrite the created `prp_` id because the read-back failed"* as **Absent**. That assessment was incomplete. It traced the guarantee through the four reason-bearing failure paths and never asked what happens where **no reason is produced at all**: a 2xx read-back carrying `{}`, `{"data":{}}`, or a document for another proposal unmarshals into `Document[Proposal]` without error, so the empty reason made both call sites treat it as answered and substitute a zero-valued proposal. The human body rendered `  []` — id and status gone — and the machine arm emitted the empty document, dropping the created `prp_` id from stdout entirely.

Found by external review (Copilot, PR #196), reproduced, and closed in `399b357` by requiring the decoded document to carry the requested id — which also rejects a foreign record whose verdict and bytes would otherwise be reported as this create's. Regressions added at helper, human-arm, and machine-arm level.

Recorded here rather than quietly corrected in the table above, because the process lesson outlives the bug: **a guarantee stated absolutely must be traced through the success path, not only through the enumerated failure modes** — the success path is where a silent substitution hides. `.score/memory/LEARNINGS.md` carries the full entry.

### Round 1 record (preserved)

| Dimension | Round 1 Status | Round 2 Status |
|---|---|---|
| Driving scenario coverage | ✓ Pass (1 ambiguity) | ✓ Pass (1 ambiguity) |
| Acceptance criteria | ✗ Fail (1) | ✗ Fail (1) |
| Interface contract conformance | ✗ Fail (1) | ✗ Fail (2) |
| Non-behavior absence | ✓ Pass — *on incomplete evidence* | ✓ Pass (guard added) |
| @wip lifecycle completion | ✓ Pass | ✓ Pass |
| Validation scenarios | ✓ Satisfied (3 of 4) | ✓ Satisfied (3 of 4) |

---

## Next Steps

5 findings to address, four of them artifact edits:

1. **F-1** via `/score:implement` — narrow T005's criterion and re-point `TestRunProposalCreate_PreChangeUserTemplateStillRenders` at fixtures whose create and read-back documents differ, so the assertion can fail if the promise breaks.
2. **F-2** and **F-5** by updating the interface accords — move `funcMap` out of `interface-spec.md`'s Unchanged list and document `include` plus the `init()` assignment; add the identity/shape-mismatch cause to `interface-cli.md`'s reason table.
3. **F-3** and **F-4** by a small `spec.md` / feature-file clarification, or via `/score:clarify` if F-4's scoping question deserves a recorded decision.

**Round 3 is the last round.** A third consecutive Issues verdict triggers the hard stop, at which point the remaining findings get assessed as spec problems versus implementation problems and handed back for a decision. Since four of these five are one-line artifact corrections, resolving them before the next run is the cheap path. The findings are recorded here rather than fixed — the developer owns which to act on before merge.
