# Validate: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Round**: 3 of 3
**Date**: 2026-08-09
**Verdict**: Ready
**Artifacts loaded**: `spec.md`, `plan.md`, `tasks.md`, `interface-cli.md`, `interface-spec.md`, `features/success-reported-for-a-dead-proposal/post-create-validity-read.feature`, `PROJECT.md`, previous `validate.md` (Rounds 1–2)
**Implementation files**: 6 production files — `internal/glassfrog/proposal.go`, `internal/render/render.go`, `internal/render/usertemplate.go`, `internal/render/templates/proposal-created.{full,compact}.tmpl`, `internal/cli/proposal.go` — plus 4 test files. Commits `bf97834`…`9a92e62` (7 tasks, 1 review fix, 4 round-3 remediation commits).

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass (9 of 9 covered) | 0 |
| Acceptance criteria | ✓ Pass (38 of 38 criteria met) | 0 |
| Interface contract conformance | ✓ Pass (both accords conformant) | 0 |
| Non-behavior absence | ✓ Pass (11 of 11 absent) | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (4 of 4) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings

No dimension was skipped. `go test ./...` passes across all 12 packages (2315 tests); `gofmt -l .` and `golangci-lint run` are clean; the feature's godog suite executes 15 scenarios / 100 steps.

**Independence caveat**: this round's inspection ran in the same context that produced the Round 1–2 remediation. Principle 4's separation between creation and evaluation is therefore weaker here than in Rounds 1–2, where the findings pre-existed the session. Two mitigations were applied and are recorded because they are the reason this verdict should be trusted at all: every claim added to an accord was re-verified against source rather than against the commit that wrote it, and both falsifiability fixes were confirmed by **mutation** — the guarded behavior was deliberately broken and the tests were observed to fail — rather than by observing them pass. The residual risk is concentrated in what a fresh reader would notice and this one did not; the Round-2 miss recorded below is evidence that such misses are real.

---

## Driving Scenario Coverage

**Status**: Pass (9 of 9 covered)

Re-traced against current code. The Round 1–2 mapping holds. F-3's wording correction removes the last divergence between the scenario text and the code path it describes.

| Scenario | Status | Implementation |
|---|---|---|
| A valid draft reports its verdict alongside its id | ✓ Covered | `runProposalCreate` (human arm) → `render.NewProposalVerdict` → `proposal-created.full.tmpl` |
| A created-but-invalid draft surfaces the server's refusal | ✓ Covered | Same path; alerts block; read-back's `available_transitions` render the empty set |
| An agent parses the verdict out of machine-readable output | ✓ Covered | Machine arm emits the read-back's raw `{data}`; `writeMachineVerdictAdvisory` renders provenance structurally on stderr. `spec.md` now states this shape rather than implying stdout |
| The read-back cannot reach the server | ✓ Covered | `readBackVerdictReason` transport arm; outcome stays `Success` |
| The create itself is rejected | ✓ Covered | `reportFailure` returns before the read-back stage |
| The server reports no verdict at all | ✓ Covered | `Valid == nil` arm → "not reported by the server" |
| A valid draft with no available transitions | ✓ Covered | Validity in the verdict block; transitions stay a line of the shared body |
| A status that disagrees with the verdict | ✓ Covered | Status renders from the server's document; `NewProposalVerdict` never reads it |
| The read-back exhausts the hour's request budget | ✓ Covered | 429 arm names the exhausted request budget |

---

## Acceptance Criteria

**Status**: Pass (38 of 38 criteria met across 7 checked tasks)

All 7 tasks remain checked.

| Task | Criteria | Status |
|---|---|---|
| T001 — verdict fields + `ValidationAlert` | 6 | ✓ All met |
| T002 — created view, verdict projection, label mapping | 7 | ✓ All met |
| T003 — resource registration + delegating templates | 7 | ✓ All met |
| T004 — isolated read-back helper | 6 | ✓ All met |
| T005 — human render path | 8 | ✓ All met |
| T006 — machine render path | 8 | ✓ All met |
| T007 — test reconciliation + BDD suite | 5 | ✓ All met |

T005's sixth criterion — the one Round 2 recorded as over-claimed — has been narrowed to what ADR-4 delivers: every pre-074 field path still resolves and none is removed, renamed, or reshaped, while values reflect the read-back where one was obtained. The criterion now also states the test obligation it implies (the pinning projection must be one on which the two documents disagree), which is what makes it checkable rather than merely true.

T004's criterion "a wire failure, a non-2xx, a post-retry 429, and an undecodable body each return a distinct reason" holds: those four families retain four distinct reasons. The identity guard is a fifth cause that reuses the fourth family's reason by design, and `interface-cli.md` now enumerates it.

---

## Interface Contract Conformance

**Status**: Pass (both accords conformant)

`interface-cli.md` — the operator-facing accord — conformant on every surface: command synopsis with no opt-out flag, the four verdict states, both human formats, machine stdout, the user-template view, the format-aware stderr advisory, and the unchanged exit-code set. The reason-cause table now enumerates **six** causes against the implementation's six, with the two that share a reason string marked as sharing it and the reason given. Verified against `readBackProposalVerdict`: empty id, wire failure, non-2xx, post-retry 429, unmarshal error, and `doc.Data.ID != id`.

`interface-spec.md` — the Go-surface accord — now matches the implementation at the three points Round 2 recorded as diverged:

| Accord claim | Verified against |
|---|---|
| `funcMap` gains one helper, `include(name string, data any) (string, error)` | `internal/render/render.go:671` — returns the engine's error unchanged |
| `include` is reachable from user templates because they share the FuncMap | `usertemplate.go:94` clones `userTemplateBase`, itself a clone of `templates` built with `Funcs(funcMap)`. The data-only sandbox holds: `include` exposes no file, network, or exec surface |
| `templates` and `userTemplateBase` are `init()`-assigned to break the `funcMap` ↔ `templates` cycle | `render.go:722-732` |
| Compact template sketch `{{trimSpace (include "proposal.compact.tmpl" .)}}  {{.Verdict.Compact}}` | `internal/render/templates/proposal-created.compact.tmpl`, byte-for-byte |

`plan.md` ADR-4 carried the same two claims and was corrected with them, though Round 2 did not cite it — see the improvement summary.

---

## Non-Behavior Absence

**Status**: Pass (11 of 11 exclusions absent)

| Non-behavior | Status | Evidence |
|---|---|---|
| No local validity determination | ✓ Absent | `NewProposalVerdict` branches only on `valid *bool` and the unavailable reason; the identity guard compares ids and derives no verdict |
| No outcome/exit-code change when not valid | ✓ Absent | `Success` in every verdict state |
| Fields not presented as published contract | ✓ Absent | No help text, `docs/`, or `plugin/` reference; both confirmed absent from `spec/glassfrog-api-v5.yaml`. Re-verified after this round's accord edits, which discuss `funcMap` and not the verdict fields |
| Missing verdict not treated as favourable | ✓ Absent | `Valid == nil` → "not reported by the server" |
| Created `prp_` id never withheld | ✓ Absent (guarded) | `doc.Data.ID == id` required before a read-back counts as answered. Covered at helper level (5 shape cases incl. `{}`, `{"data":{}}`, and a foreign `prp_9999`), human-arm level, and machine-arm level |
| No read-back after a rejected create | ✓ Absent | `reportFailure` returns before the read-back stage; asserted |
| No opt-out of the read-back | ✓ Absent | No flag added |
| No polling or extra retry | ✓ Absent | One `Execute` through the shared `RetryExecutor` |
| Not extended to other proposal writes | ✓ Absent | Two call sites, both inside `runProposalCreate` |
| Verdict not rendered in the proposal list | ✓ Absent | `ResourceProposals` untouched; the byte-identical guard sets `Valid=false` plus alerts and asserts the shared rendering carries no verdict line — a discriminating guard, not an agreeing one |
| Nothing withheld from `proposal get --output json` | ✓ Absent | `internal/cli/proposal_reads.go` unchanged |

---

## @wip Lifecycle Completion

**Status**: Pass

19 scenarios, 15 executing with `@wip` cleared, 4 `@wip` tags all on `@validation` scenarios held for this skill per `tasks.md` § Scenario disposition. The step line added to the format-scoping validation scenario this round sits inside a `@wip` scenario and is therefore not executed — the suite still reports 15 scenarios / 100 steps, unchanged.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced to implementation)

| Scenario | Status | Trace |
|---|---|---|
| The reported result names the read that produced the verdict | ✓ Satisfied | `full`: `Verdict source: read-back of <prp_id> after create`. `compact`/user template: stderr prose names the read. Machine: `verdict_source.read_back` + `proposal_id` |
| No verdict is derived from the change set, status, transitions, or alerts | ✓ Satisfied | `NewProposalVerdict` reads only `valid` and the reason; neither template consults status or transitions for the validity line; the alert count appends in *either* validity state. The identity guard is an identity comparison, not a validity derivation |
| The verdict fields are never presented as published contract | ✓ Satisfied | Absent from help, `docs/`, `plugin/`, and the vendored contract; field comments state the non-declaration explicitly |
| Every verdict state is distinguishable in every output format | ✓ Satisfied | The four built-in formats are pinned by golden tests (`full`/`compact`) and by the four-case table over `data.valid` + `verdict_source.read_back` (`json`/`yaml`). The caller-authored template form is now explicitly scoped in the scenario, the feature file, and `interface-cli.md` § "stdout — user template": the command promises the verdict is *available*, not rendered, for output it does not compose |

**Observation (not a conformance failure, carried from Rounds 1–2)**: `docs/guides/proposals/how-to-manage-a-governance-proposal.md:37` still describes the create's result as id + status — accurate but incomplete now. `/score:document` owns that surface.

---

## Verdict: Ready

All 5 conformance dimensions pass. All 4 validation scenarios satisfied. The implementation conforms to its specification, and the specification now describes the implementation.

The five Round-2 findings are resolved: four were artifact corrections, and the fifth (F-1) was an over-claimed acceptance criterion whose remedy was to narrow the claim and make the two tests behind it falsifiable. No behavior changed in this round — the only production-source edit is one comment narrowed to match the accord it cites.

---

## Changes Since Previous Run

**Round**: 3 (previous: Round 2, verdict Issues)

### Resolved (5 of 5 findings)

- **F-1** (Rounds 1–2): T005's "output is unchanged" criterion — **resolved**, at two sites. The criterion in `tasks.md` is narrowed to the field-path promise ADR-4 actually delivers, with the value behavior stated explicitly. `TestRunProposalCreate_PreChangeUserTemplateStillRenders` is re-pointed at `.Proposal.AvailableTransitions` — the field on which the create and read-back fixtures disagree — and split into two cases asserting which document supplied the values (read-back answered / read-back failed). The second site is recorded below.
- **F-2** (Rounds 1–2): `interface-spec.md` Go-surface drift — **resolved**. `funcMap` moved out of the Unchanged list; the `include` helper, its reachability from user templates, and the `init()` assignment are documented; the compact template sketch matches the file. `plan.md` ADR-4 corrected with it.
- **F-3** (Rounds 1–2): spec driving-scenario provenance wording — **resolved**. The scenario now states that the reported result carries provenance in a machine-readable form alongside the emitted document, which stays the server's own.
- **F-4** (Rounds 1–2): validation-scenario format scoping — **resolved**. The scenario, the feature file, and `interface-cli.md` now agree that the four-state guarantee covers the formats the command composes, and that a caller-authored template gets availability rather than rendering.
- **F-5** (Round 2): the reason-cause table omitted the identity/shape mismatch — **resolved**. The table enumerates six causes, states why two share a reason string (the vocabulary is closed on purpose), and names the different follow-ups the two suggest. The helper's own comment was narrowed from "every failure family gets a distinct reason" to the exchange families it actually maps.

### New (0 findings)

### Closed outside the finding list — a Round 2 miss

F-1 had a **second site** that Rounds 1 and 2 did not identify. The feature scenario *"A user template written before the verdict still renders"* — which **executes**, unlike the four held scenarios — asserted *"the template's output will be unchanged by the verdict's addition"*, and its step definition was rigged to make that true: the When overrode the read-back with `available_transitions: ["propose"]`, identical to the create response, under a comment stating this *"keeps the template's inputs identical to the pre-074 rendering"*. The projection therefore rendered the same string whichever document was substituted, and the step could not fail. An inert assertion was running green on every PR.

Found while sweeping F-1 during this round's inspection, fixed in `9a92e62`: the step now asserts the read-back's values are what the paths carry, the fixture carries `["propose","withdraw"]` against the create's `["propose"]`, and the template projects the transition count. Verified falsifiable by the same mutation probe used at the unit level.

Recorded here rather than folded into F-1's resolution line, because the process lesson matters: **a finding about a non-discriminating test fixture should trigger a search for sibling fixtures rigged the same way, not just a fix at the cited line.** Round 2 cited one test by name; the identical defect one layer down went uncited for two rounds. `.score/memory/LEARNINGS.md` should carry this alongside the Round-1 miss it rhymes with — both are cases where the enumerated evidence was checked and the unenumerated sibling was not.

### Round 1–2 record (preserved)

| Dimension | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| Driving scenario coverage | ✓ Pass (1 ambiguity) | ✓ Pass (1 ambiguity) | ✓ Pass |
| Acceptance criteria | ✗ Fail (1) | ✗ Fail (1) | ✓ Pass |
| Interface contract conformance | ✗ Fail (1) | ✗ Fail (2) | ✓ Pass |
| Non-behavior absence | ✓ Pass — *on incomplete evidence* | ✓ Pass (guard added) | ✓ Pass |
| @wip lifecycle completion | ✓ Pass | ✓ Pass | ✓ Pass |
| Validation scenarios | ✓ Satisfied (3 of 4) | ✓ Satisfied (3 of 4) | ✓ Satisfied (4 of 4) |

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.

Two items are deliberately left open and belong to other owners:

1. `docs/guides/proposals/how-to-manage-a-governance-proposal.md:37` describes the create's result as id + status — still accurate, now incomplete. `/score:document` owns it.
2. The Round-2 miss above is worth a `LEARNINGS.md` entry on sibling-fixture sweeps, alongside the clean-unmarshal entry from the Round-1 miss.

The independence caveat at the head of this document applies to the verdict: a reviewer who did not write this round's changes is the missing check, and PR review is where it belongs.
