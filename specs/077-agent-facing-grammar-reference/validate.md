# Validate: Agent-Facing Grammar Reference

**Feature**: 077-agent-facing-grammar-reference
**Round**: 2 of 3
**Date**: 2026-08-10
**Verdict**: Ready
**Artifacts loaded**: spec.md (amended after round 1), plan.md (§ System Architecture, Architecture Decisions, Integration Design, Cross-cutting Concerns, Risks — amended), tasks.md (6 of 6 tasks complete — amended), interface-cli.md (amended), `features/unguided-change-construction/agent-facing-grammar-reference.feature`, PROJECT.md
**Implementation files**: 11 — `internal/grammar/` (artifact.go, grammar.go, grammar.json, gen/main.go), `internal/build/` (grammarartifact.go, grammarfacts.go, writesafetyguardrail.go), `internal/cli/` (proposal_grammar.go, proposal.go), `internal/render/` (grammar.go, render.go, templates/grammar.{full,compact}.tmpl), `plugin/hooks/glassfrog-write-gate.sh`, `docs/reference/change-set-grammar.md`

Round 1's record is preserved in full below. Four artifacts were amended between
the rounds, so every dimension was re-inspected against the amended text rather
than carried forward — a round-1 trace against superseded wording proves nothing
about the current state.

---

# Round 2 (2026-08-10) — Ready

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 3) | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings

One advisory note is recorded below (A-1). It is not a finding: it concerns the
precision of one artifact clause relative to its amended sibling in the same
artifact, not anything the implementation does or fails to do.

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

Re-executed at the current head: **12 scenarios, 52 steps, 0 failures**. The
concretized feature file is unchanged since round 1; the spec's driving scenarios
are also unchanged (the round-1 amendments touched § Non-Behaviors and § Validation
Scenarios only), so round 1's per-scenario traces stand and were confirmed green.

## Acceptance Criteria

**Status**: Pass (6 of 6 tasks complete, all criteria met)

| Task | Status | Change since round 1 |
|---|---|---|
| T001 Generator + committed artifact | ✓ Met | — |
| T002 Embed + typed accessor | ✓ Met | — |
| T003 Drift guard | ✓ **Now met** | The spec's nested-only-membership trigger gained its red case (`TestGrammarArtifactGuardCatchesANestedOnlyMembershipShift`). Verified by execution, and verified *red* with the vocabulary-half comparison disabled — where it fails with the misleading "the ENCODING diverged" message, which is the wrong-file blame the assertions guard against. |
| T004 Render resource + templates | ✓ Met | — |
| T005 Command + both guardrail edits | ✓ Met | The criterion itself was amended (malformed credential *value*). Re-verified against the amended text, including the **no-credential-file-present** case over the real settings-file walk — an empty directory resolves to `full` and the full reference renders at exit 0. Round 1 had traced only the no-token-context variant. |
| T006 Reference documentation | ✓ Met | — |

## Interface Contract Conformance

**Status**: Pass (9 of 9 surface clauses conformant)

| Surface clause (interface-cli.md) | Status | Change since round 1 |
|---|---|---|
| `glassfrog proposal grammar` — read leaf under the `proposal` group | ✓ Conformant | — |
| Arguments: none; cobra rejects any positional as usage | ✓ Conformant | Clause amended to name the inherited template source as a non-exception; implementation matches |
| Command-local flags: none | ✓ Conformant | — |
| Inherited `--output` participates fully; `--base-url` parses and is inert | ✓ Conformant | — |
| Short help names the surface honestly | ✓ **Now conformant** | The Short carries all four elements the clause names — the change-set grammar, *before assembling*, *contract-published* types with placement, *verified empirical observations* — at 121 characters, inside the CLI's existing 126 maximum. A Short-specific assertion and a length bound now pin it; the pre-existing help sweep was satisfied by either field and could not have caught it. |
| Rendered structure: two keys, field tables, provenance tokens, deterministic order | ✓ Conformant | — |
| Human formats (`full`/`compact`) content requirements | ✓ Conformant | — |
| Error communication: 0 / 2 / 1, codes 3–7 unproducible | ✓ Conformant | — |
| Credential-free conduct | ✓ Conformant | Clause amended to a malformed credential *value*, with the shared-file boundary stated. Both halves pinned by named tests. |

The `json` envelope deviation remains a recorded interface decision, not a finding
(round 1 § Interface Contract Conformance).

## Non-Behavior Absence

**Status**: Pass (6 of 6 non-behaviors absent)

The first non-behavior was amended after round 1 and now reads: "no argument or
flag of its own takes one … The output-template source every read inherits is not
an exception: it renders a caller's template and evaluates nothing, so it is not a
change-set input path."

Against that text the implementation conforms exactly, and the boundary is pinned
rather than asserted: `TestProposalGrammar_APipedChangeSetIsRenderedAsATemplateNotJudged`
executes the piped-change-set path and asserts the text renders verbatim, that no
validity vocabulary appears on either stream, and that the caller's template
*replaces* the built-in rendering. Nothing parses the bytes as governance, compares
them against the grammar, or produces a verdict.

The other five non-behaviors were re-checked and remain absent; round 1's evidence
(provenance tokens on every entry, no per-type field list, no routing/identifier
content, no write path, no drafting-path rewiring) is unchanged.

## @wip Lifecycle Completion

**Status**: Pass

Three `@wip` tags remain, all carrying `@validation` — correctly held out. No
scenario referenced by a checked task retains `@wip`.

## Validation Scenario Results

**Status**: Satisfied (3 of 3)

Re-traced independently at the current head, not carried forward from round 1.

| Scenario | Status | Trace |
|---|---|---|
| The rendered vocabulary equals the contract's | ✓ Satisfied | Set comparison re-run against the vendored contract: **21 rendered types = 21 contract types, 6 nested-only = 6 nested-only, zero missing and zero extra in either direction**. Every `wrappers` member is itself an enum member. |
| No judgment path exists | ✓ Satisfied | The operative clause — "no invocation evaluates, filters, or scores a change set" — holds by construction and is now pinned by execution on the one path that consumes caller bytes. See advisory A-1 on the scenario's closing clause. |
| One source for the residue | ✓ Satisfied **literally** | The scenario was amended to "renders from the record, through whatever generated projection the binary carries" and "no fact's text is **hand-maintained** outside the record". Both are literally true: all four rendered fields of both facts are byte-identical to the record parser's output, the artifact is machine-generated and header-marked, and a hand edit to a fact's symptom is rejected by the guard naming the record. Round 1's reliance on plan ADR-1's reading is no longer needed. |

## Advisory

### A-1: `Then its only effect is rendering knowledge` was not amended alongside its sibling clause

- **Type**: advisory — artifact wording, no implementation gap
- **Location**: `spec.md:150` (§ Validation Scenarios, "No judgment path exists") and its concretized twin at `features/…/agent-facing-grammar-reference.feature:79`
- **Why this is not a finding**: round 1's F-1 cited two sites — § Non-Behaviors and this closing clause. The fix amended the former and left the latter, so spec.md now states the same boundary at two precision levels: the non-behavior explicitly acknowledges the inherited template source, while this clause still asserts that rendering knowledge is the only effect. For the one invocation where a caller's template ignores the grammar data, the effect is rendering *that template*. The implementation conforms to the amended non-behavior, and the spec has now told a reader how to read the boundary, so the clause reads as a summary of the effect class (a rendering, never a judgment or a mutation) rather than an independent claim.
- **Recommended resolution**: whenever spec.md is next edited, scope the clause the way the non-behavior now is — e.g. "its only effect is rendering, never judging". The concretized twin is the Verifier's artifact; validate does not modify `.feature` scenario content, so aligning it is a scenarios-stage touch-up. **No implementation change is recommended, and this does not block merge.**
- **Pattern worth noting**: this is the third instance in this feature of one wording fix leaving a sibling copy behind (F-1's four code/doc sites, F-3's plan quotes, and now this clause). The lesson is recorded in `.score/memory/LEARNINGS.md` — sweep the phrase, not the file.

## Verdict: Ready

All 5 conformance dimensions pass and all 3 validation scenarios are satisfied. The
implementation conforms to its specification.

**Why not Issues**, stated so the boundary is auditable: the verdict tree gives
Issues when a conformance check fails. None does. A-1 is not a failed check — it
describes one artifact clause being less precise than its amended sibling in the
same artifact, which is cross-artifact wording consistency rather than
implementation conformance, and the implementation matches every artifact under the
spec's own now-explicit reading. Recording it as a finding would report an
implementation gap that does not exist.

**Independent corroboration worth recording**: between the rounds, an automated
reviewer (Copilot, PR #204) flagged five items on this feature. Three were the same
defects round 1 found — the Short help's missing provenances (F-4), the docs'
"takes no input" over-claim (F-1), and the absent nested-only-membership red case
(F-2) — reached without seeing the validation pass. One (the credential over-claim
surviving in the command's own comments) showed round 1's fix had been applied
incompletely. One was new and outside round 1's scope (a leaf named "advance" in a
sibling doc's overview). Two independent passes converging on the same three
defects is evidence the round-1 findings were real rather than manufactured; the
fourth is evidence that a wording fix needs a phrase sweep, not a file edit.

## Changes Since Previous Run

**Round**: 2 (previous: Round 1 — Issues, 4 findings)

### Resolved (4 of 4 findings)

- **F-1** (Non-behavior absence) — *resolved by amendment, implementation deliberately unchanged*. spec.md § Non-Behaviors now scopes the claim to "no argument or flag of its own takes one" and names the inherited template source as a non-exception. The same absolute phrasing was swept from interface-cli.md, the command's package comment, its step comment, plan.md, tasks.md, two test comments, and the reference doc. The behaviour is now pinned by execution rather than asserted.
- **F-2** (Acceptance criteria, T003) — *resolved*. The nested-only-membership red case exists, derives both sides from source, and was verified red before being trusted.
- **F-3** (Validation scenario, "One source for the residue") — *resolved by amendment*. spec.md was aligned to the phrasing the scenarios stage had already concretized; plan.md ADR-1 and Risk R1 were updated so their quotes are not misquotes. The scenario is now satisfied literally, without recourse to ADR-1's reading. Round 1 also **overstated this finding's scope** — the feature file's concretization was already correct — and that correction is recorded in the Resolution Log below.
- **F-4** (Interface contract conformance) — *resolved*. The Short carries the accord's four elements within the house length range, pinned by a Short-specific assertion and a length bound.

### Remaining (0 findings)

### New (0 findings; 1 advisory)

- **A-1** — the un-amended closing clause of the "No judgment path exists" scenario. Advisory, not a finding: see above.

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge — PR
[#204](https://github.com/Luscii/cli-glassfrog/pull/204), 9 CI checks green, all
review threads resolved.

Two optional touch-ups, neither blocking: A-1's clause wording (spec.md and its
feature-file twin), and the three `@validation` scenarios remain `@wip` by design —
they are inexecutable as written (they inspect the landed surface rather than drive
it) and were traced by inspection here.

---

# Round 1 (2026-08-10) — Issues

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✗ Fail | 1 |
| Interface contract conformance | ✗ Fail | 1 |
| Non-behavior absence | ✗ Fail (1 ambiguous) | 1 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied (3 of 3; 1 under a recorded reading) | 1 |

**Total**: 5 dimensions checked, 2 passed, 4 findings

No finding is a behavioral gap. Every driving scenario, Conduct accord, and Sync
accord clause was traced to a working code path, and each trace below was verified
by execution rather than by reading. Two findings are implementation-side and small
(F-2, F-4); two are spec-wording matters where the implementation is correct and the
decision was already recorded upstream (F-1, F-3).

---

## Driving Scenario Coverage

**Status**: Pass (8 of 8 scenarios covered)

All eight spec driving scenarios were concretized into the feature file and execute
green (12 scenarios, 52 steps, 0 failures).

| Scenario (spec.md § Driving Scenarios) | Status | Implementation |
|---|---|---|
| An assembler reads the grammar before building | ✓ Covered | `internal/cli/proposal_grammar.go:runProposalGrammar`; verified 21 types + 2 facts rendered with no request |
| Provenance is visible on every shape | ✓ Covered | `internal/grammar/artifact.go` (token constants); every entry carries one — verified over the shipped artifact |
| The grammar renders without credentials | ✓ Covered | `runProposalGrammar` takes `selectionSeam` only; `assembleCalled`/`newClientCalled`/transport-call assertions all zero |
| A change set offered for judgment is refused as usage, not judged | ✓ Covered | `cobra.NoArgs`; exit 2 with no validity vocabulary (4 positional shapes swept) |
| The contract vocabulary drifts from the rendering | ✓ Covered | `internal/build/grammarartifact.go:CheckGrammarArtifact`; verified red on an enum change **and** on a nested-only membership shift |
| A recorded fact retires | ✓ Covered | `grammarFactEntries` over the manifest; `CreatePolicy` verified to remain in the contract-derived layer |
| "Accepted" is not "valid" survives the rendering | ✓ Covered | CSG-2 renders `accepted-but-invalid`; symptom states a returned `prp_` id is not a successful change |
| No live residue | ✓ Covered | `grammar.{full,compact}.tmpl` `{{else}}` arm; golden-pinned in both formats |

**Content accord — the record *gaining* a fact.** spec.md § Content requires the
rendering to follow the record when it "gains or retires a fact". Only retirement
carries a scenario, so the gain path was traced separately: a synthesized CSG-3
(disposition `rejected`) flows through `BuildGrammarFromSources` into the rendered
residue in manifest order, with Evidence and lineage excluded, and the guard goes
red with two findings naming the record until regeneration lands. The gain path
conforms and additionally exercises the third disposition value the shipped record
does not currently use.

---

## Acceptance Criteria

**Status**: Fail (6 of 6 tasks complete; 1 finding)

Every checked task's criteria were inspected. All are met except one divergence
class named by the spec but absent from T003's list.

| Task | Status | Note |
|---|---|---|
| T001 Generator + committed artifact | ✓ Met | Both halves derived; wrapper-pair decision recorded in a code comment at `specWrapperRE`; empty manifest → `facts: []` verified; byte-determinism pinned |
| T002 Embed + typed accessor | ✓ Met | Envelope unreachable through `Load()` (marker/path strings asserted absent from the payload); decode failure returns an error across 5 corrupt shapes |
| T003 Drift guard | ✗ Partial | Five named divergence classes each have a red case; the spec's sixth trigger has none — **F-2** |
| T004 Render resource + templates | ✓ Met | Both templates end in `0x7d` (no trailing newline); golden tests pin both formats plus the empty-residue variant |
| T005 Command + both guardrail edits | ✓ Met | `TestWriteSafetyRegistryDriftGuard` green; gate passes `proposal grammar` ungated; `TestOperatingSurfaceSelfContainment` green (no new development-repository reference in `plugin/`) |
| T006 Reference documentation | ✓ Met | Formats, structured keys, token vocabularies, and the 0/2/1 envelope with 3–7 stated unproducible; cross-cutting `-o` facts verified against `internal/cli/root.go` |

---

## Interface Contract Conformance

**Status**: Fail (7 of 8 surface clauses conformant, 1 finding)

| Surface clause (interface-cli.md) | Status | Finding |
|---|---|---|
| `glassfrog proposal grammar` — read leaf under the `proposal` group | ✓ Conformant | — |
| Arguments: none; cobra rejects any positional as usage | ✓ Conformant | — |
| Command-local flags: none | ✓ Conformant | — |
| Inherited `--output` participates fully; `--base-url` parses and is inert | ✓ Conformant | — |
| Short help names the surface honestly | ✗ Non-conformant | F-4 |
| Rendered structure: two keys, field tables, provenance tokens, deterministic order | ✓ Conformant | — |
| Human formats (`full`/`compact`) content requirements | ✓ Conformant | — |
| Error communication: 0 / 2 / 1, codes 3–7 unproducible | ✓ Conformant | — |
| Write-gate conduct: `PROPOSAL_READS` + `expectedProposalSurface` in the same change | ✓ Conformant | — |

**Recorded deviation, not a finding.** The accord's Consistency Notes deliberately
break the `{data: …}` envelope convention for `json`/`yaml` ("Recorded as a
deliberate deviation so no reviewer 'fixes' it toward `{data: …}`"). The
implementation matches the accord, and the deviation is documented in
`docs/reference/change-set-grammar.md`. Re-litigating a recorded interface decision
is outside this dimension's lane; noted here for traceability only.

---

## Non-Behavior Absence

**Status**: Fail (5 of 6 non-behaviors clean, 1 ambiguous finding)

| Non-behavior (spec.md § Non-Behaviors) | Status | Finding |
|---|---|---|
| Must not accept a change set to judge | ✗ Ambiguous | F-1 |
| Must not present an empirical fact as contract-authoritative | ✓ Absent | — |
| Must not invent per-type field guidance | ✓ Absent | — |
| Must not carry routing or identifier facts | ✓ Absent | — |
| Must not replace or retire the recorded grammar facts | ✓ Absent | — |
| Must not rewire the drafting path's consultation step | ✓ Absent | — |

Evidence for the five clean rows:

- **Provenance marking**: every `change_types` entry carries `published-contract`
  and every `facts` entry `empirical-observation`; the guard fails the build on a
  missing or wrong token on either side.
- **Per-type field guidance**: the rendered type entry carries exactly `type`,
  `placement`, `wrappers`, `provenance` — no field list anywhere.
- **Routing / identifier facts**: the rendered payload contains none of
  `circle_id`, `route`, `routing`, `anchor circle`, `identifier resolution`,
  `legacy`, `numeric id`, `ten_`, `role_`.
- **Record ownership**: the command performs no write; the record stays at its 072
  home and `GrammarFactsPath` is unchanged.
- **Drafting path**: the only file touched under `plugin/` is
  `glassfrog-write-gate.sh`, a one-word addition to `PROPOSAL_READS` — which
  spec.md § Integration Boundaries explicitly requires "in the same change that
  ships the command". No `plugin/skills/proposal-drafting/` file was modified.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced; V3 under a reading recorded in plan ADR-1)

These were held out from the Builder. Each was traced independently by execution,
not by reusing the driving-scenario pass.

| Scenario | Status | Trace |
|---|---|---|
| The rendered vocabulary equals the contract's | ✓ Satisfied | Independent set comparison against `ProposalChange.properties.type.enum` and the nested-only rule, read from the vendored contract at assertion time: **21 rendered = 21 contract, 6 nested-only = 6 contract, zero missing and zero extra on either side**. Every `wrappers` member is itself an enum member. |
| No judgment path exists | ✓ Satisfied, with a caveat | No invocation evaluates, filters, or scores a change set: there is no parse of a change set, no validity predicate, and no comparison of caller input against the grammar anywhere in `proposal_grammar.go`. Caveat on "its only effect is rendering knowledge" — see **F-1**. |
| One source for the residue | ✓ Satisfied under ADR-1's reading | All four rendered fields of both facts are **byte-identical** to the record parser's output; the committed artifact is byte-identical to a fresh derivation; a hand edit to a fact's symptom is rejected by the guard, naming the record as the diverged half. The literal clause "no fact's text lives a second life outside it" is not literally true — see **F-3**. |

---

## Findings

### F-1: A change set piped to `-o stdin` succeeds (exit 0) and is echoed back

- **Dimension**: Non-behavior absence (also caveats validation scenario "No judgment path exists")
- **Source**: spec.md § Non-Behaviors — "The command must not accept a change set to judge — **no input path for one exists**, and an attempt fails as a usage error, never as a validity verdict"; `features/…/agent-facing-grammar-reference.feature` — "its only effect will be rendering knowledge"
- **Implementation**: `internal/cli/proposal_grammar.go:runProposalGrammar` → `resolveRenderTarget` → `internal/cli/usertemplate.go:readTemplateSourceFrom` (the 035 `TemplateStdin` branch)
- **Gap**: The natural attempt conforms exactly — `glassfrog proposal grammar changes.json` exits 2 with no validity language. But the inherited user-template path is an input path that consumes piped bytes. Verified by execution: piping `[{"type":"CreatePolicy","name":"x"}]` with `-o stdin` **exits 0 and prints the change set back verbatim** (a JSON document with no `{{` is a valid `text/template` that renders itself). The grammar is not rendered at all, so "its only effect is rendering knowledge" does not hold for that invocation, and an agent could misread exit 0 as acceptance.

  **Ambiguity, stated rather than assumed**: the *excluded capability* is absent — nothing evaluates, filters, or scores the change set; the bytes are treated as a template, not as governance. This is also unmodified house behavior on every read (`cat anything | glassfrog roles -o stdin` behaves identically), and rejecting stdin templates on this one command would violate the Conduct accord's "renders in it the way other reads do". So the gap is in the non-behavior's absolute phrasing ("no input path for one exists"), not in the code.

  **Recommended resolution**: a spec wording change — scope the clause to a change-set input path, or note the inherited template source as an acknowledged non-judging input. **No implementation change is recommended.** The developer owns this call.

### F-2: The Sync accord's nested-only-membership trigger has no red-case test

- **Dimension**: Acceptance criteria (T003)
- **Source**: spec.md § Sync — "When the vendored contract's change-type vocabulary changes — a type added, removed, or renamed, **or the nested-only membership shifting** — and the rendered reference no longer matches it, the repository's merge-gating verification run fails"; tasks.md T003 acceptance criteria list five divergence classes, the contract one being "vendored-spec enum change without regeneration"
- **Implementation**: `internal/build/grammarartifact_guard_test.go` — `TestGrammarArtifactGuardCatchesAContractRefreshWithoutRegeneration` perturbs the enum by appending a member; no test perturbs nested-only membership with the enum unchanged
- **Gap**: The behavior conforms — verified by execution. Dropping `CreateDomain` from the contract's nested-only list while leaving the enum intact makes a fresh derivation reclassify it as `top-level` with no `wrappers`, and `CheckGrammarArtifact` goes red naming the contract-derived half and the regeneration remedy. What is missing is the guard *for the guard*: the spec names membership shift as a distinct trigger, and T003's class list did not carry it, so no red case pins it. A future refactor of the placement derivation could silently stop catching this trigger with every test still green.

  **Recommended resolution**: one added red case in `grammarartifact_guard_test.go` perturbing nested-only membership with the enum unchanged, asserting the finding names the contract half. Fixable in an implement round.

### F-3: The residue's text exists in two committed files, which the held-out scenario's literal wording excludes

- **Dimension**: Validation scenario ("One source for the residue")
- **Source**: spec.md § Validation Scenarios — "each renders from the record **And no fact's text lives a second life outside it**"; spec.md § Sync — "no second copy of a fact's text exists **to drift** from the record"
- **Implementation**: `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` (the record) and `internal/grammar/grammar.json` (the generated artifact) — verified: CSG-1's symptom text is present in exactly these two files
- **Gap**: Literally read, the validation scenario is not satisfied: a second copy of each fact's text is committed. Read as the Sync accord phrases it ("to drift"), it is satisfied: there is exactly one editable source, the artifact is header-marked generated, and drift is mechanically impossible — a hand edit to the artifact's fact text is rejected by the guard naming the record, and a record edit without regeneration is rejected too. Both were verified by execution.

  This is not an undiscovered gap. plan.md ADR-1 records the drift-impossibility reading explicitly and says "a validator reading the spec alone should be pointed here", and plan Risk R1 predicts this exact challenge, prescribing "a spec wording touch-up, not an architecture change". The architecture was developer-confirmed at resolve.

  **Recommended resolution**: align the validation scenario's wording with the Sync accord's ("no second copy … to drift from the record"), or add the ADR-1 pointer to the scenario. **No implementation change is recommended.**

### F-4: Short help omits the provenance framing the accord names for it

- **Dimension**: Interface contract conformance
- **Source**: interface-cli.md § Surface > The command — "**Short help**: names the surface honestly — the change-set grammar for proposal changes: the part shapes and placement rules **from the published contract**, plus **verified empirical observations**, **consulted before assembling**"
- **Implementation**: `internal/cli/proposal_grammar.go` — `Short: "Show the change-set grammar: the change types, where each may appear, and the known dead shapes"`
- **Gap**: The Short names the shapes and placement rules but not the two provenance standings the accord enumerates for it: it does not say the vocabulary comes *from the published contract*, and "the known dead shapes" does not convey *verified empirical observations*. "Consulted before assembling" and "judges nothing" are absent from Short but present in Long, which satisfies the bullet's second sentence ("Help text must state…"); the provenance framing appears in Long only.

  This matters because `glassfrog proposal --help` shows only the Short, so an agent scanning the subcommand list gets no signal that the reference mixes two standings — which is the feature's second user scenario.

  **Tradeoff, stated so it can be dismissed knowingly**: cobra Shorts are one-line list entries, and the accord's full sentence would make this one unwieldy. A middle option exists (e.g. naming "contract types and verified observations"). Fixable in one line, or accept as-is with the accord amended.

---

## Verdict: Issues

4 findings across 4 dimensions. None is a behavioral gap: all 8 driving scenarios,
every Conduct clause, both Sync clauses, and all three held-out validation scenarios
trace to working, execution-verified code paths, and 5 of 6 non-behaviors are cleanly
absent.

Two findings are implementation-side and incremental — F-2 (one missing red case for
a trigger the spec names and the code already handles) and F-4 (one line of help
text). Both are fixable in a single implement round.

Two findings are spec-wording matters where the implementation is correct and should
not change — F-1 (the non-behavior's "no input path exists" is absolute, while the
inherited, non-judging template source consumes stdin) and F-3 (the validation
scenario's literal wording versus the Sync accord's "to drift" phrasing, already
anticipated and resolved by plan ADR-1 / Risk R1). Recommending code changes for
either would break the Conduct accord's format uniformity or reverse a
developer-confirmed architecture decision.

The gaps are incremental, not fundamental, so this is Issues rather than Not Ready.

---

## Resolution Log (fix round, post-round-1)

Appended after round 1 rather than replacing it — the findings above stay as the
record of what round 1 found. This section records only what was done about them. It
is not a round-2 verdict; re-run `/score:validate` for that.

| Finding | Disposition | Change |
|---|---|---|
| F-1 | Wording amended; behavior deliberately unchanged | spec.md § Non-Behaviors now scopes the claim to "no argument or flag of its own takes one" and states that the inherited output-template source is not an exception. The same absolute phrasing was swept from interface-cli.md, `proposal_grammar.go`, two test comments, and the reference doc — all four had repeated it. Pinned by `TestProposalGrammar_APipedChangeSetIsRenderedAsATemplateNotJudged`, which asserts the piped text renders verbatim, that no validity vocabulary appears, and that the caller's template *replaces* the built-in rendering (the fact that makes "its only effect is rendering knowledge" imprecise for that one invocation). |
| F-2 | Fixed | `TestGrammarArtifactGuardCatchesANestedOnlyMembershipShift` added: one type leaves the contract's nested-only set with the **enum untouched**, and the guard must name the contract half and not the record half. Both sides derive from source (the shifted set is the real set minus one of its own members), so no type name is hard-coded. Verified red before trusting it — with the vocabulary-half comparison disabled it fails, and fails with the misleading "the ENCODING diverged" message, which is exactly the wrong-file blame the test guards against. |
| F-3 | Wording amended; architecture unchanged, as plan R1 prescribed | **Round 1 overstated this finding's scope.** The concretized scenario in the feature file already read "renders from the record **through the generated artifact**" and "no fact's text will be **hand-maintained** outside the record" — both literally satisfied. Only spec.md retained "lives a second life outside it". spec.md was aligned to the scenarios stage's existing phrasing; plan.md ADR-1 and Risk R1 quoted the old text and were updated so they no longer misquote, with a note that the recorded decision is unchanged. |
| F-4 | Fixed | The Short now carries all four elements the accord names for it — the change-set grammar, *before assembling*, *contract-published* types with placement, and *verified empirical observations* — at 121 characters, inside the CLI's existing 126-character maximum. `TestProposalGrammar_HelpStatesItInformsAndNeverValidates` gained a Short-specific assertion (the help-text sweep above it is satisfied by either field, so it could not have caught this) plus a length bound. |

**Correction to round 1**: F-3's finding text attributed the literal wording to "the
held-out scenario" generally. That was wrong — the feature file's concretization was
already correct, and only spec.md carried the absolute phrasing. The finding's
substance (two committed copies of the fact text) stands; its scope was narrower than
reported.

**Not changed, deliberately**: the `json` envelope deviation (recorded in the
accord's Consistency Notes) and the credential-file boundary recorded in
`.score/memory/LEARNINGS.md` — the latter is an accord over-claim whose fix is also a
wording change, but it was reported as a LEARNINGS entry during implementation rather
than as a round-1 finding, and it is listed here so it is not lost: **interface-cli.md
§ Interactions still promises the command works "with a malformed one [credential
file]", which is not achievable** because the credential shares `.glassfrogrc` with
the `output` setting. Suggested wording: "a malformed credential *value*".

---

## Next Steps (as recorded at round 1)

Superseded by the Resolution Log above — kept as round 1's handoff record.

4 findings to address. Suggested split:

1. **Implement round** (F-2, F-4): add the nested-only-membership red case; decide
   the Short help wording. Then re-validate (`/score:validate`).
2. **Developer decision** (F-1, F-3): both are spec-wording touch-ups with no code
   change recommended. F-3's resolution is already prescribed by plan Risk R1. If
   you accept the current wording as-is, say so in the PR so the next validation
   round does not re-raise them.

Nothing here blocks PR review. If you would rather ship and reword later, F-2 is the
only finding with a durability cost — it leaves a spec-named divergence trigger
unpinned by any test.
