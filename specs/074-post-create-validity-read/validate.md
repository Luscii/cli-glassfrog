# Validate: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Round**: 1 of 3
**Date**: 2026-08-08
**Verdict**: Issues
**Artifacts loaded**: `spec.md`, `plan.md`, `tasks.md`, `interface-cli.md`, `interface-spec.md`, `features/success-reported-for-a-dead-proposal/post-create-validity-read.feature`, `PROJECT.md`
**Implementation files**: 6 production files — `internal/glassfrog/proposal.go`, `internal/render/render.go`, `internal/render/usertemplate.go`, `internal/render/templates/proposal-created.{full,compact}.tmpl`, `internal/cli/proposal.go` — plus 4 test files. Commits `bf97834`…`7cfb9bb` (7 tasks, one commit each).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass (9 of 9 covered; 1 wording ambiguity) | 1 |
| Acceptance criteria | ✗ Fail (37 of 38 criteria met) | 1 |
| Interface contract conformance | ✗ Fail (CLI accord conformant; Go accord diverged) | 1 |
| Non-behavior absence | ✓ Pass (11 of 11 absent) | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 4; 1 scoping ambiguity) | 1 |

**Total**: 5 dimensions checked, 3 passed, 4 findings

No dimension was skipped. `go test ./...` (2302 tests), `gofmt -l .`, and `golangci-lint run` are clean; the feature's godog suite executes 15 scenarios / 100 steps.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 covered) — one ambiguity recorded as F-3

Every driving scenario in `spec.md` § Driving Scenarios traces to a code path and to an executing scenario in the feature file.

| Scenario | Status | Implementation |
|---|---|---|
| A valid draft reports its verdict alongside its id | ✓ Covered | `internal/cli/proposal.go:runProposalCreate` (human arm) → `render.NewProposalVerdict` → `proposal-created.full.tmpl` |
| A created-but-invalid draft surfaces the server's refusal | ✓ Covered | Same path; alerts block in `proposal-created.full.tmpl`; read-back's `available_transitions` render the empty set |
| An agent parses the verdict out of machine-readable output | ✓ Covered (see F-3) | `runProposalCreate` machine arm emits the read-back's raw `{data}`; `writeMachineVerdictAdvisory` renders provenance on stderr |
| The read-back cannot reach the server | ✓ Covered | `readBackProposalVerdict` → `readBackVerdictReason` transport arm; outcome stays `Success` |
| The create itself is rejected | ✓ Covered | `reportFailure` returns before the read-back stage; no read is issued |
| The server reports no verdict at all | ✓ Covered | `Valid == nil` arm of `NewProposalVerdict` → "not reported by the server" |
| A valid draft with no available transitions | ✓ Covered | Validity renders in the verdict block; transitions stay a line of the shared body |
| A status that disagrees with the verdict | ✓ Covered | Status renders from the server's document; `NewProposalVerdict` never reads it |
| The read-back exhausts the hour's request budget | ✓ Covered | 429 arm of `readBackVerdictReason` names the exhausted request budget |

---

## Acceptance Criteria

**Status**: Fail (37 of 38 criteria met across 7 checked tasks; 1 finding)

All 7 tasks are checked. Criteria were verified against implementation and tests, not against the commit messages.

| Task | Criteria | Status |
|---|---|---|
| T001 — verdict fields + `ValidationAlert` | 6 | ✓ All met |
| T002 — created view, verdict projection, label mapping | 7 | ✓ All met |
| T003 — resource registration + delegating templates | 7 | ✓ All met |
| T004 — isolated read-back helper | 6 | ✓ All met |
| T005 — human render path | 8 | ✗ 7 met, 1 not strictly met (F-1) |
| T006 — machine render path | 8 | ✓ All met |
| T007 — test reconciliation + BDD suite | 5 | ✓ All met |

Notable criteria confirmed by inspection rather than by assertion count: T003's registry-exhaustiveness test derives its expectation from `len(builtinResources)` rather than a literal; T004's no-error signature is structural (`(glassfrog.Proposal, json.RawMessage, string)`); T006's verbatim-bytes claim is pinned twice — `bytes.Equal` against the wire bytes in `TestReadBackProposalVerdict_RawIsVerbatim`, and against the fixture's own normalization in `TestRunProposalCreate_MachineEmitsReadBackVerbatim`.

---

## Interface Contract Conformance

**Status**: Fail (`interface-cli.md` fully conformant; `interface-spec.md` diverged at two points — F-2)

`interface-cli.md` — the operator-facing accord — is conformant on every surface:

| Surface | Status | Evidence |
|---|---|---|
| Command synopsis (no flag added; no opt-out) | ✓ Conformant | `newProposalCreateCommand` declares only `--changes`; grep for `no-verify`/`skip-verdict` returns nothing |
| The four verdict states | ✓ Conformant | `NewProposalVerdict` produces exactly the four full labels and the four compact labels |
| stdout — human `full` | ✓ Conformant | `proposal-created.full.tmpl`; golden tests pin the 16-column alignment, the conditional alerts block, and the source line |
| stdout — human `compact` | ✓ Conformant | One line; `(N alert(s))` appended in either validity state |
| stdout — machine formats | ✓ Conformant | Read-back's raw `{data}` verbatim, create's raw as fallback; no composed envelope |
| stdout — user template | ✓ Conformant (field paths) | `ProposalCreatedView` embeds `ProposalView`; promotion asserted |
| stderr — advisory (both families) | ✓ Conformant | Prose in human formats; `{"verdict_source": …}` in machine formats, absent keys for not-applicable |
| Exit codes (unchanged set) | ✓ Conformant | `Success` in all four verdict states; read-back failures never reach an exit code |

`interface-spec.md` — the Go-surface accord — diverged at two points (F-2): `funcMap` is listed under **Unchanged** but gained an `include` helper, and `proposal-created.compact.tmpl` differs from the pinned sketch. The rendered output still matches `interface-cli.md`, which is why the CLI accord passes and this one does not.

---

## Non-Behavior Absence

**Status**: Pass (11 of 11 exclusions absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No local validity determination | ✓ Absent | `NewProposalVerdict` branches only on `valid *bool` and the unavailable reason; alerts affect only the compact count suffix; status and transitions are never read |
| No outcome/exit-code change when not valid | ✓ Absent | `Success` returned in every verdict state; asserted in both render arms |
| Fields not presented as published contract | ✓ Absent | No help text, doc, or skill references either field; model comments state the opposite ("NOT declared in `spec/glassfrog-api-v5.yaml`") |
| Missing verdict not treated as favourable | ✓ Absent | `Valid == nil` → "not reported by the server"; tri-state decode pinned with three distinct cases |
| Created `prp_` id never withheld | ✓ Absent | Id printed in all four states; machine path always emits a document |
| No read-back after a rejected create | ✓ Absent | `reportFailure` returns before the read-back stage; `TestRunProposalCreate_FailedCreateIssuesNoReadBack` asserts no GET follows |
| No opt-out of the read-back | ✓ Absent | No flag added |
| No polling or extra retry | ✓ Absent | One `Execute` through the shared `RetryExecutor`; no loop, no sleep of its own |
| Not extended to other proposal writes | ✓ Absent | Two call sites, both inside `runProposalCreate`; `propose`/`withdraw`/`respond` files unchanged since `f1e7ba1` |
| Verdict not rendered in the proposal list | ✓ Absent | `ResourceProposals` untouched; `TestRender_ProposalKeyed_UnchangedByVerdictFields` pins byte-identical output |
| Nothing withheld from `proposal get --output json` | ✓ Absent | `internal/cli/proposal_reads.go` unchanged |

---

## @wip Lifecycle Completion

**Status**: Pass

The feature file holds 19 scenarios. 15 have their `@wip` cleared and execute in `TestPostCreateValidityReadFeatures` (15 scenarios / 100 steps). The 4 remaining `@wip` tags are all on `@validation`-tagged scenarios, which `tasks.md` § Scenario disposition explicitly holds for this skill — T007's criteria state they must not be cleared. No stray `@wip` remains on any scenario referenced by a checked task.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 4 traced to implementation; 1 scoping ambiguity — F-4)

| Scenario | Status | Trace |
|---|---|---|
| The reported result names the read that produced the verdict | ✓ Satisfied | `full`: `Verdict source: read-back of <prp_id> after create`. `compact`/user template: the stderr prose line names the read. Machine: `verdict_source.read_back` + `proposal_id`. A reader absent from the write can tell a second read produced it in every format. |
| No verdict is derived from the change set, status, transitions, or alerts | ✓ Satisfied | `NewProposalVerdict` (inspected in full) reads only `valid` and the unavailable reason. Neither template consults `.Proposal.Status` or `.Proposal.AvailableTransitions` for the validity line. Alert count appends to the compact label in *either* validity state, so it never stands in for the verdict. |
| The verdict fields are never presented as published contract | ✓ Satisfied | `valid`/`validation_alerts` appear in no help text, no `docs/`, no `plugin/` skill or agent. Both are confirmed absent from `spec/glassfrog-api-v5.yaml`, and the field comments say so explicitly. |
| Every verdict state is distinguishable in every output format | ✓ Satisfied for the four built-in formats; ambiguous for the user-template form | `full` (four distinct `Validity` lines) and `compact` (four distinct labels) pinned by golden tests; `json`/`yaml` pinned by the four-case table over `data.valid` + `verdict_source.read_back`. See F-4 for the user-template form. |

**Observation (not a conformance failure)**: `docs/guides/proposals/how-to-manage-a-governance-proposal.md:37` still describes the create's result as "the created proposal with its `prp_` id and `draft` status" — accurate but now incomplete, since the create also reports a verdict and writes a stderr advisory. No artifact promised a documentation update and no task covered one; `/score:document` owns that surface.

---

## Findings

### F-1: T005's "output is unchanged" criterion is over-claimed, and its test cannot detect the difference

- **Dimension**: Acceptance criteria
- **Source**: `tasks.md` § T005 — "A user-supplied template written against the pre-change view still renders — every field path that resolved before still resolves, **and its output is unchanged by the verdict's addition**"
- **Implementation**: `internal/cli/proposal.go:runProposalCreate` (human arm) — `proposal := doc.Data; if reason == "" { proposal = readBack }`; test at `internal/cli/proposal_readback_test.go:TestRunProposalCreate_PreChangeUserTemplateStillRenders`
- **Gap**: When the read-back answers, the human arm renders the **read-back's** proposal, not the create's. Field paths all still resolve (the first half of the criterion holds), but *values* now come from the second document. A pre-074 user template referencing `.Proposal.AvailableTransitions`, `.Proposal.Status`, `.Proposal.UpdatedAt`, or the response counts can therefore print something different than it did before. The pinning test projects only `.Proposal.ID`, `.Proposal.Status`, and `len .Proposal.Changes` against fixtures that agree on all three — while the same fixtures disagree on transitions (`["propose"]` in the create body vs `["propose","withdraw"]` in the read-back). The criterion is asserted only in the case where it cannot fail.

  The document swap is not itself a defect: `plan.md` § Verdict Assembly sources available transitions from the read-back, and the driving scenario "A created-but-invalid draft surfaces the server's refusal" requires the empty transition set that only the read-back reports. The gap is that the criterion claims more than the design delivers, and the test does not discriminate. Resolution is to narrow the criterion (field paths resolve; values reflect the read-back where one was obtained) and re-point the test at fixtures that differ — not to change the render.

### F-2: `interface-spec.md`'s Go-surface accord no longer matches the implementation

- **Dimension**: Interface contract conformance
- **Source**: `interface-spec.md` § Surface — "**Unchanged**: `proposal.full.tmpl`, `proposal.compact.tmpl`, `ProposalView`, `ResourceProposal`, **`funcMap`**, `Render`, `RenderError` …"; and the pinned compact template `{{template "proposal.compact.tmpl" .}}  {{.Verdict.Compact}}`
- **Implementation**: `internal/render/render.go` (`funcMap` gained `include`; `templates` and `userTemplateBase` moved into `init()`); `internal/render/templates/proposal-created.compact.tmpl` reads `{{trimSpace (include "proposal.compact.tmpl" .)}}  {{.Verdict.Compact}}`
- **Gap**: Three stated-unchanged items changed. The shared compact template ends with a trailing newline and `{{template}}` output cannot be captured or trimmed, so the pinned sketch would have put the verdict on a second line — breaking the one-line compact contract that `interface-cli.md` § "stdout — human `compact` format" pins with five examples. The implementation chose the user-facing accord over the Go sketch, which is the right precedence, and `LEARNINGS.md` records the decision. What remains is that `interface-spec.md` now misdescribes the surface a future reader would implement or review against. Two specifics deserve to land in the accord: `include` is reachable from user templates (035), so the FuncMap surface a caller-authored template may call is wider than any artifact documents (it is pure and data-only, so the 035 sandbox holds); and `templates`/`userTemplateBase` are now `init()`-assigned to break the resulting declaration-time initialization cycle.

### F-3: The machine-output driving scenario reads as though provenance rides the emitted document

- **Dimension**: Driving scenario coverage
- **Source**: `spec.md` § Driving Scenarios — "Then the emitted document carries the validity verdict … / **And it carries the provenance of the verdict** — that a read of that record produced it"
- **Implementation**: `internal/cli/proposal.go:writeMachineVerdictAdvisory` — provenance is a `{"verdict_source": …}` document on **stderr**; stdout carries the server's document verbatim with nothing added
- **Gap**: Read literally, "it" is the emitted document from the preceding line, and the implementation does not put provenance there — `plan.md` ADR-5 explicitly rejected composing an envelope, because inventing a response shape would break the 018 verbatim contract. Three downstream artifacts settle the reading the implementation follows: ADR-5, `interface-cli.md` § stderr, and the feature file's own step ("the advisory will be rendered in the selected machine format"). `spec.md` § Behavioral Accord is also consistent with it ("the *reported result* carries the provenance … in every output format"). So this is a wording ambiguity in one scenario line, not a behavioral gap — but the spec is the top of the chain, and a reader who starts there will expect an in-document field. Worth one clarifying edit so the four artifacts read the same way.

### F-4: Validation scenario "every output format" does not say whether the user-template form is in scope

- **Dimension**: Validation scenarios
- **Source**: feature file § "Every verdict state is distinguishable in every output format" — "Given each output format the create supports"; `interface-cli.md` § Command synopsis enumerates **six** output forms, including a template file path
- **Implementation**: `internal/cli/proposal.go` (human arm) — a user template renders whatever its author wrote; the stderr advisory that accompanies it is the human **prose** line, which is byte-identical across `valid`, `not valid`, and `not reported` (all three are `read_back: true`)
- **Gap**: For the four built-in formats the scenario holds and is pinned by tests. For a caller-authored template that does not reference `.Verdict`, three of the four states are indistinguishable — stdout says nothing about validity and the prose advisory says the same thing in all three. The CLI cannot control what a caller's template prints, and `interface-cli.md` § "stdout — user template" takes the position that the verdict is *available* rather than rendered — which is arguably the most the command can promise. The artifacts do not state which reading governs, so the scenario cannot be marked unambiguously satisfied. Resolution is a scoping sentence — either the scenario names the built-in formats, or `interface-cli.md` states that the user-template form satisfies distinguishability by making the verdict available on the view.

---

## Verdict: Issues

4 findings across 3 conformance dimensions and the validation-scenario set. All four are incremental and none touches the feature's behavior on the built-in formats:

- **F-1** is the only finding with a code-adjacent component, and even there the correct move is to narrow an over-claimed acceptance criterion and re-point one test at discriminating fixtures — the render behavior it describes is required by a driving scenario.
- **F-2**, **F-3**, and **F-4** are artifact corrections: an interface accord that drifted from the implementation it governs, one ambiguous scenario line, and one unscoped "every format" claim.

No behavioral section is unimplemented, no interface surface is missing, and every `@validation` scenario traced to real code — so this is not Not Ready. The implementation delivers what the specification promised on the surfaces the specification pins.

---

## Next Steps

4 findings to address. Suggest:

1. **F-1** via `/score:implement` — narrow T005's criterion and re-point `TestRunProposalCreate_PreChangeUserTemplateStillRenders` at fixtures whose create and read-back documents differ, so the assertion can fail if the promise breaks.
2. **F-2** by updating `interface-spec.md` — move `funcMap` out of the Unchanged list, document `include` (including its reachability from user templates) and the `init()` assignment, and correct the compact template's pinned content.
3. **F-3** and **F-4** by a small `spec.md` / feature-file clarification — either directly or via `/score:clarify` if the scoping question in F-4 deserves a recorded decision.

Then re-validate (`/score:validate 074`) to close the loop. The findings are recorded here rather than fixed; the developer owns which of them to act on before the PR.
