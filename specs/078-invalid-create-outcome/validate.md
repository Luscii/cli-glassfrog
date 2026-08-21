# Validate: Invalid-Create Outcome

**Feature**: 078-invalid-create-outcome
**Round**: 1 of 3
**Date**: 2026-08-21
**Verdict**: Issues
**Artifacts loaded**: spec.md, plan.md (§ System Architecture, Architecture Decisions, Implementation Strategy), tasks.md (4 of 4 tasks complete), interface-cli.md, features/success-reported-for-a-dead-proposal/invalid-create-outcome.feature, features/success-reported-for-a-dead-proposal/post-create-validity-read.feature, PROJECT.md
**Implementation files**: 12 files under `internal/cli/` and `internal/output/` (5 production, 7 test), plus 2 feature files, 2 `internal/build` guards, 8 documentation surfaces, and 2 plugin skills

---

## Independence Degradation (read first)

This validation was produced by the **same context that implemented the feature**. Score's Principle 4 separation between creation and evaluation is therefore **not** structurally present in this round:

- The four `@validation` scenarios were **not held out** from the implementing agent. They were read in full during T003, when the whole feature file was loaded to build its runner. Their "independent verification" property does not hold for this round.
- Findings below are the product of adversarial re-inspection against the artifacts, verified by command output and an empirical render dump rather than recall — but a blind spot shared between implementation and evaluation would not have been caught.

Both findings recorded below are in prose the implementing context wrote or swept, which is weak evidence that the re-inspection had some independent reach. It is not a substitute for a separate assessor. **Recommendation**: treat the two findings as actionable, and treat the `@validation` results as corroborating rather than independent.

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✗ Fail | 2 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 4 passed, 2 findings. Both findings are prose truthfulness defects in claims about the exit-code convention; neither is a behavioral gap.

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

Every driving scenario in spec.md is concretized as an **active** (untagged) scenario in `invalid-create-outcome.feature` and executes in `TestInvalidCreateOutcomeFeatures` (14 scenarios, 92 steps, all passing).

| Spec driving scenario | Concretized as | Status | Implementation |
|---|---|---|---|
| An invalid draft terminates the create as a failure | An invalid draft fails the create with its own exit code | ✓ Covered | `internal/cli/proposal.go:216` → `invalidCreateFired` → `reportFailure`; diagnostic in `diagnostic.go:renderDiagnostic` |
| A valid draft still succeeds | A valid draft still succeeds | ✓ Covered | `proposal.go:216` guard falls through; `render.NewProposalVerdict` unchanged |
| A valid draft carrying alerts still succeeds | A valid draft carrying alerts still succeeds | ✓ Covered | `invalidCreateFired` reads only `Valid`; alert presence never consulted |
| A missing verdict leaves the create a success | A missing verdict leaves the create a success | ✓ Covered | `readBack.Valid != nil` guard in `proposal.go:343` |
| A failed read-back leaves the create a success | A failed read-back leaves the create a success | ✓ Covered | `unavailableReason == ""` guard in `proposal.go:343` |
| The failure keys on the verdict, not the transitions | The failure keys on the verdict, not the transitions | ✓ Covered | Trigger expression reads no transition field; fixture supplies a non-empty set |
| A machine-readable failure is fully structured | A machine-readable failure carries the structured envelope | ✓ Covered | `proposal.go:184` → `errorEnvelopeFor`; `internal/output/error.go:ErrorDetail` |
| The new code never renumbers an existing one | The invalid-create code is new and renumbers nothing | ✓ Covered | `internal/cli/exitcode.go:40`; BDD registry steps in `invalid_create_outcome_bdd_test.go` |

The three User Scenarios' Rule blocks are all represented, including the compact-format and user-template surfaces the interface accord added beyond the spec's driving set.

---

## Acceptance Criteria

**Status**: Fail (T001, T002, T003 criteria met; T004 has one criterion not fully satisfied — 2 findings)

| Task | Criteria met | Notes |
|---|---|---|
| T001 | ✓ All | `ExitCode(InvalidCreate)=8`, `kind()="invalid-create"`, `String()="InvalidCreate"` (pinned in `dispatch_test.go`). All six length-guarded mirror lists extended. Header count eight→nine. Orientation row + range landed with the constant, keeping `TestOperatorOrientationDriftGuard` green. Verified against `origin/main`: only `codeInvalidCreate = 8` was added to the const block — no existing constant line changed — and both frozen historical sentences survive byte-identical (`exitcode.go:21`, `exitcode_test.go:9`). |
| T002 | ✓ All | Cause and next step byte-identical to the interface accord (verified by render dump). Envelope carries `kind`/`proposal_id`/`validation_alerts`; `status`/`body`/`feature` absent. Zero-alert case omits `validation_alerts` entirely. The pinned plan-limit `order` slice was not extended; a separate declaration-order assertion covers the invalid-create key set. `internal/output` imports no model or transport package (`go list -deps` confirms: `rcfile`, `resolve`, `output` only). |
| T003 | ✓ All | Both render branches guarded before any stdout write; machine branch binds the decoded proposal. Exchange count pinned at 2 both at the transport (`tr.calls`) and in Gherkin. Three superseded scenarios excluded (sibling suite: 12 of 19). Six superseded Go tests re-pointed or inverted, with branching loop bodies in the two table-driven ones. `DEPRECATION.md` ticked. |
| T004 | ✗ One criterion | The site list was completed (and two omitted sites were found and fixed). The **property** criterion — *"no surface enumerating the convention omits a published code, and no stated range ends below the highest published code"* — is not fully satisfied: see **F-1**. A second, narrower ownership gap is **F-2**. |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

Verified by dumping the actual rendered documents from the shipped code paths and comparing against the accord's pinned examples, rather than by reading the tests.

| Surface | Status | Evidence |
|---|---|---|
| Command — unchanged, no `--allow-invalid`/`--no-fail` | ✓ Conformant | No flag registration added anywhere in the branch diff |
| The four verdict states → outcomes | ✓ Conformant | `invalidCreateFired` maps exactly one state to failure; three fall through |
| stdout — machine formats: envelope keys, order, and strings | ✓ Conformant | Render dump matches `interface-cli.md` lines 43–55 **byte-for-byte**, including key order `message → next_step → kind → proposal_id → validation_alerts` |
| Field contract: presence rules per key | ✓ Conformant | `status`/`body`/`feature` absent; `proposal_id` always present on this failure; `validation_alerts` present only with ≥1 alert |
| `validation_alerts` absent (never `[]`) on a zero-alert invalid draft | ✓ Conformant | Render dump of the zero-alert variant emits neither the key nor an empty array |
| stderr — human formats: cause, alert lines, next step | ✓ Conformant | Render dump matches `interface-cli.md` lines 74–78 byte-for-byte, including the two-space indent and the `\n — ` separator |
| stderr — zero alerts collapses to one line | ✓ Conformant | Render dump shows the single `cause — next step` line, no blank gap |
| stdout — user template: failure bypasses rendering | ✓ Conformant | `resolveRenderTarget` returns `format: output.DefaultFormat` for a template, so the failure takes the human path before `writeHuman`; scenario asserts the template marker never appears |
| Machine-format stderr: "nothing new; retry notices only" | ✓ Conformant | `reportFailure`'s machine arm writes only to stdout |
| Exit codes — the registry becomes 0–8 | ✓ Conformant | `exitcode.go` const block and switch; end-to-end propagation verified below |
| Exchange counts unchanged (2 on the invalid path) | ✓ Conformant | Transport tripwire and Gherkin step |

**End-to-end propagation verified explicitly** (the one link a unit test on `ExitCode` alone would mask): `outcomeToDispatchError`'s `default:` arm (`me.go:353`) wraps any non-Success/UsageError/RuntimeError outcome in `*outcomeError`, and `dispatch.go:264` honours the carried category verbatim. The path is generic — there is no per-category whitelist that `InvalidCreate` could fall out of. Confirmed by the BDD scenarios asserting exit 8 through the full `Run()` entry point.

---

## Non-Behavior Absence

**Status**: Pass (9 of 9 exclusions upheld)

| Non-behavior | Status | Evidence |
|---|---|---|
| Must not raise the failure on anything but explicit `valid: false` | ✓ Upheld | `proposal.go:343` — `unavailableReason == "" && readBack.Valid != nil && !*readBack.Valid`. No other predicate. |
| Must not determine validity locally | ✓ Upheld | The trigger reads only the server's `Valid` field; no status, transition, alert, or change-set term appears in it |
| Must not withhold/delay/replace the created `prp_` id | ✓ Upheld | Both construction sites pass the id; `errorEnvelopeFor` maps it; the cause names it. **Structurally guaranteed non-empty**: an undeterminable id makes `readBackProposalVerdict` short-circuit with a non-empty reason (`proposal.go:325`), so the trigger cannot fire with `ProposalID == ""` — which matters because the field carries `omitempty` and would otherwise vanish |
| Must not reuse an existing failure code | ✓ Upheld | `codeInvalidCreate = 8` is new; `TestExitCodeConstants_Distinct` enforces uniqueness across all nine |
| Must not perform its own read-back or extra request | ✓ Upheld | No request added; `tr.calls == 2` asserted at the transport and in Gherkin |
| Must not change the outcome of any other proposal write | ✓ Upheld | `invalidCreateFired` has exactly two call sites, both in `runProposalCreate`. `proposal_propose.go`, `proposal_withdraw.go`, `proposal_respond.go` are untouched by the branch |
| Must not retry, poll, or wait for validity | ✓ Upheld | No loop, sleep, or retry introduced in `proposal.go` |
| Must not emit a success-shaped document under the failure code, or vice versa | ✓ Upheld | The check precedes every stdout write on both branches; machine stdout carries the envelope only (`"data"` absence asserted); the `verdict_source` advisory is suppressed on the failure and still fires on all three success states |
| Must not embed the full server proposal document in the failure | ✓ Upheld | The envelope carries three scalar fields plus a three-string alert list; no `json.RawMessage` passthrough of the proposal |

---

## @wip Lifecycle Completion

**Status**: Pass

- `invalid-create-outcome.feature`: 18 scenarios. **14 active** (all `@wip` and all six `@pending-deprecation` tags removed), **4 retain `@validation @wip`** — correctly held out, including the one that was `@validation @pending-deprecation @wip` and kept `@validation @wip` because "active" never applied to it. No `@wip` remains on any non-`@validation` scenario.
- `post-create-validity-read.feature`: the retirement mechanism landed with the retag. Three previously-active `@deprecate` scenarios are now `@deprecated` **and** the runner filters `~@wip && ~@deprecated`, so the tag finally carries an exclusion (12 of 19 scenarios run). Before this change the tag was read by nothing in the repository.
- The Go half of the same collision — six tests no tag filter reaches — was re-pointed or inverted in the same commit.

---

## Validation Scenario Results

**Status**: Satisfied (4 of 4 traced) — **but see the Independence Degradation section: these were not held out from the implementing context.**

| Scenario | Status | Trace |
|---|---|---|
| No failure trigger exists besides the server's explicit verdict | ✓ Satisfied | `invalidCreateFired` (`proposal.go:343`) is the sole trigger — one boolean expression reading only the unavailable reason and `*readBack.Valid`. Two call sites, both in the create. No status, transition-set, alert-count, or change-set term reachable from it. |
| The created id survives every failure path | ✓ Satisfied | Two failure paths (machine `proposal.go:184`, human `proposal.go:217`), both passing the id. Non-emptiness is structural, not incidental: an undeterminable id yields a non-empty unavailable reason, which disables the trigger. |
| The registry stays one-to-one after the new code | ✓ Satisfied | `TestExitCode_ProducerBackedCategories` (9 categories, `len` + comma-ok guards), `TestExitCodeConstants_ExactValues`, `TestExitCodeConstants_Distinct` (no shared code), plus the BDD registry scenario comparing every pre-078 assignment against `ExitCode` after the addition. |
| The failure is distinguishable from every success state | ✓ Satisfied | `TestRunProposalCreate_MachineFourStateTable` branches per state: the failure by exit 8 + `kind: "invalid-create"` + absence of a `data` document + absence of the advisory; the three successes by `data.valid` presence/value + `verdict_source.read_back`. A valid draft carrying alerts is recognizable as the *valid* state (`proposalReadBackValidWithAlertBody`, and the active `Structured output carries the verdict for a valid create` scenario). No human prose is required for any of the four. |

---

## Findings

### F-1: Three surfaces state the exit-code range as "3–7", ending below the highest published code

- **Dimension**: Acceptance criteria (T004)
- **Source**: tasks.md T004 § Acceptance criteria — *"no surface enumerating the convention omits a published code, and no stated range ends below the highest published code — including surfaces that write the range with backticked digits or state no range at all."* The criterion is unqualified; it is not scoped to the enumerated site list.
- **Implementation**:
  - `internal/cli/proposal_grammar.go:83` — `// Exit codes 3–7 (API, permission, rate-limit, network, stale-write) are unproducible by construction: nothing here can produce a response, a 403, a 429, a transport error, or an ETag conflict.`
  - `docs/reference/change-set-grammar.md:230` — `**Exit codes 3–7 are unproducible by this command.**`
  - `docs/reference/proposals.md:280` — "codes `3`–`6` and `8` are unproducible for it because it issues no request." (quoted, not code-spanned: the sentence contains its own backticks)
- **Gap**: `proposal grammar` issues no request and creates nothing, so code `8` is unproducible by it for exactly the reason `3`–`7` are. All three sentences purport to enumerate what the command cannot produce, and all three now omit a published code. The third site is the more serious of the three: it was **edited by T004** and left at `3`–`6` **and** `8`, reproducing inside the sweep the precise defect T004's own scope note warned about for `how-to-read-exit-codes.md` (*"Adding 8 alone would ship a guide in which 8 exists but 7 does not"*). The three sites now state the same fact in three different and mutually inconsistent forms. The honest form for all three is `3`–`8`, with `invalid-create` added to the parenthetical category list and a matching clause in the "nothing here can produce…" enumeration. No behavioural impact — these are contract-fact comments and prose.

### F-2: Five prose sites in `internal/cli` name only two of the six categories that reach the exit-code convention via the dispatch error channel

- **Dimension**: Acceptance criteria (T001 / plan Phase 1 ownership)
- **Source**: plan.md § System Architecture item 6 — *"**Phase 1** owns everything inside `internal/cli`"* for convention-documenting surfaces; tasks.md T001 asserts the in-package set is exactly *"the five present-tense prose claims the new code falsifies"*.
- **Implementation**:
  - `internal/cli/dispatch.go:32` — `so Exit-Code Convention maps it to the right code (3/6)`
  - `internal/cli/dispatch.go:261` — `dispatch cannot re-derive (APIError, NetworkUnavailable) by returning`
  - `internal/cli/dispatch.go:263` — `maps it (3/6) rather than the RuntimeError(1) catch-all.`
  - `internal/cli/me.go:342` — `the operational categories (APIError, NetworkUnavailable)`
  - `internal/cli/me.go:343` — `carrying the category so Exit-Code Convention maps them to 3/6`
- **Gap**: `outcomeToDispatchError`'s `default:` arm routes **six** categories through `*outcomeError` today — `APIError`(3), `PermissionError`(4), `RateLimited`(5), `NetworkUnavailable`(6), `StaleWrite`(7), and now `InvalidCreate`(8) — but all five comments present a two-element set and the code pair "(3/6)" as complete. The claims were already incomplete before this feature (false since 015 added 4/5 and 054 added 7); 078 adds a sixth member. T001's enumeration of "five prose claims" in `internal/cli` therefore undercounts, and `me.go` is not in any task's file set at all, so no task would have reached it. Severity is low — the code is generic and correct, and the mechanism it describes is not category-specific — but these are the sentences a reader consults to learn which outcomes survive dispatch, and one of them states code numbers. Verified: `errors.As` on `*outcomeError` and the `default:` arm impose no whitelist, so the comments understate the code rather than the code understating the comments.

---

## Verdict: Issues

2 findings, both in the acceptance-criteria dimension, both prose truthfulness defects in claims about the exit-code convention. Neither is a behavioural gap:

- All 8 driving scenarios are covered by active, passing scenarios.
- Every interface surface is conformant, verified byte-for-byte against the accord's pinned JSON and stderr examples by dumping the shipped render paths.
- All 9 non-behaviors are upheld, two of them structurally rather than incidentally (the non-empty id, the single trigger).
- The `@wip` lifecycle is complete, and the deprecation retirement landed with a working exclusion mechanism in the same commit as the behaviour flip.
- All 4 validation scenarios trace to code — with the independence caveat recorded above.

Both findings are incremental and fixable in one implement round: eight lines of comment and prose across five files. F-1 is the more substantive of the two because one of its three sites was introduced by this change and because the three sites now disagree with each other. F-2 documents pre-existing staleness that this feature's declared Phase-1 ownership covered but its enumeration did not reach.

The gaps do **not** meet the Not Ready bar: no conformance dimension is wholly unimplemented, no interface surface is missing, and no validation scenario revealed an absent code path.

---

## Handoff

Fix via `/score:implement`, then re-validate. Suggested grouping — one commit, since both findings are the same class of defect:

1. **F-1** — bring all three "unproducible" sentences to `3`–`8` and add `invalid-create` to the category list in `proposal_grammar.go:83` and the "nothing here can produce…" clause. Re-derive the sentence rather than appending `and 8`; the three sites should also agree in form.
2. **F-2** — re-derive the five dispatch-channel comments to describe the category set open-endedly (the `default:` arm carries *every* operational category) rather than naming two of six and a "(3/6)" code pair. Phrasing that enumerates will go stale again on the next outcome; phrase it by the mechanism.

Both are prose-only; the existing test suite should remain green without modification, which is itself worth confirming rather than assuming.

**Note for the next cycle**: this round's independence was compromised (same context implemented and validated). If a second round runs, consider invoking it from a fresh session, or via the `score:guardian-agent` subagent, so the `@validation` scenarios recover their held-out property.
