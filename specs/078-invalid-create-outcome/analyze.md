# Analyze: Invalid-Create Outcome

**Feature**: 078-invalid-create-outcome
**Artifacts analyzed**: spec.md, plan.md, interface-cli.md, tasks.md, features/success-reported-for-a-dead-proposal/invalid-create-outcome.feature (new), features/success-reported-for-a-dead-proposal/post-create-validity-read.feature (sibling, tag-only edits), `.score/memory/DECISIONS.md` (three 078 entries), `.score/memory/DEPRECATION.md` (one `[decision]` entry + § Pending Deprecations)
**Checklist context**: loaded — `checklist.md` present (round 2 record: 22 checks, 20 pass, 2 P0 fail, 9 P2 considerations), correlation performed
**Checks**: 16 (11 pass, 5 fail)
**Generated**: 2026-08-20 (round 4 — re-derived from scratch)

One interface file and two feature files, so C4/C6/K2/K5/K6 run once each, with C6/K1/K5 evaluated across all scenarios in both feature files. Every claim about landed code was re-read at the source rather than taken from the artifact that cites it: `internal/cli/{exitcode.go, exitcode_test.go, dispatch.go, errorenvelope.go, errorenvelope_test.go, diagnostic.go, proposal.go, proposal_readback_test.go, post_create_validity_read_bdd_test.go, me.go, proposal_grammar.go}`, `internal/output/{error.go, selection.go}`, `internal/build/{operatororientation.go, operator_orientation_bdd_test.go, operator_orientation_guard_test.go, surface_self_containment_guard_test.go}`, `docs/guides/how-to-read-exit-codes.md`, `docs/explanation/how-failures-are-reported.md`, `plugin/skills/{orientation,proposal-drafting}/SKILL.md`, `features/unequipped-agent-operators/operator-orientation.feature`, and every `Paths:`/`Tags:` declaration under `internal/`. The previous analyze.md's verdicts were not carried forward.

---

## Summary

| Category | Severity | Checks | Pass | Fail |
|---|---|---|---|---|
| Consistency | P0 | 6 | 4 | 2 |
| Completeness | P1 | 6 | 5 | 1 |
| Coherence | P2 | 4 | 2 | 2 |
| **Total** | — | **16** | **11** | **5** |

**Findings**: 2 P0, 1 P1, 7 P2 (10 total across the 5 failing checks).

---

## Changes Since Previous Run

**Previous** (round 3): 3 P0, 3 P1, 6 P2 (12 findings)
**Current** (round 4): 2 P0, 1 P1, 7 P2 (10 findings)

**Resolved** (verified against the current files, not against a fix report):

- ~~P0 C4-a: interface-cli.md attributed the Phase-1 placement of `exitcode.go`'s narrative paragraph to a drift guard that does not read it~~ → **fixed**. § Consistency Notes now reads *"plan **Phase 1** for the orientation skill row — a drift guard couples that row to the constant itself — and for `exitcode.go`'s own narrative paragraph, which rides with the code it describes"*. Re-verified at the source: `operatororientation.go:212` regexes `code(\w+)\s*=\s*(\d+)` out of `exitcode.go`, `:253` reads the backticked digits out of the skill, `:258` `diffSets` the two bidirectionally. Nothing reads `exitcode.go`'s prose, and the accord no longer claims it does.
- ~~P0 C5-a: plan.md's 074-collision risk and Phase 1 enumeration covered only the Gherkin half~~ → **fixed**. plan.md § Risks now carries *"**The collision has a second half in Go**, which no tag filter can reach … six tests assert `Success` against it … re-point or invert those six Go tests, changing the two table-driven tests' shared loop body rather than only adding a row to their case tables"*, and § Implementation Strategy Phase 1 now lists *"the scenario retirement including the runner's `~@deprecated` filter **and the six superseded Go unit tests**"*. Plan and T003 now agree the collision has two halves.
- ~~P0 C5-b: T003's "Three additionally pin stdout content" sub-count was false and its table-test disposition unexecutable~~ → **fixed**. T003 now reads *"**All six** additionally pin success-shaped output the failure path removes"* and *"for the two table-driven tests it is **not** enough to add an expected-`InvalidCreate` row, because their shared loop body asserts `outcome != Success` → `t.Fatalf` and then asserts success-shaped stdout/stderr **unconditionally** — the loop body itself must branch per case"*. Both loop bodies re-verified: `proposal_readback_test.go:405` and `:656` each `t.Fatalf` on `outcome != Success` and then assert success-shaped stdout/stderr unconditionally.
- ~~P1 K4-a: T001's "the only prose edited" criterion foreclosed two landed present-tense enumerations~~ → **fixed in kind**. T001 now names all four: *"`exitcode_test.go`'s header count (eight → nine); `exitcode_test.go`'s "ExitCode maps the categories that have producers today…" enumeration; `dispatch.go`'s matching enumeration on the `Outcome` block; and `exitcode.go`'s "the full 0–6 convention is published now""*. All four exist at the named symbols (`exitcode_test.go:5`, `:21–27`; `dispatch.go:47–65`; `exitcode.go:8`). **Residual raised as C5-a and H1-a**: the plan was not widened to match, and DECISIONS.md still says the `exitcode.go` sentence is not edited.
- ~~P1 K4-b: T004 omitted the fourth `0–7` site inside `operator_orientation_bdd_test.go`~~ → **fixed in Scope**. T004 now names *"**two** sites inside `thenMeaningEachCode`: its loop bound … **and** the failure message it returns, `"skill does not document exit code %d in the 0–7 convention"`"*. Verified at `:246` and `:248`. A repo-wide sweep for `0–7`/`0-7` across `internal/`, `features/`, `plugin/`, `docs/` returns exactly five occurrences and T001/T004 between them name all five. **Residual raised as H1-b**: the Risk note's count was not updated.
- ~~P1 K5-a: the accord's exchange-count invariant on the invalid path had no scenario~~ → **fixed**. `invalid-create-outcome.feature:30` now carries *"And the create will have been followed by exactly one read of the created proposal"*, byte-identical to `post-create-validity-read.feature:31` and to the sibling runner's regex at `post_create_validity_read_bdd_test.go:121`. T003's criterion is no longer conditional: *"asserted **both** at the transport … **and** in Gherkin … Neither assertion is optional."*
- ~~P2 H1-b: T003 described as pending a tagging already in the file and already ticked~~ → **fixed**. T003 now reads *"(six of them, the compact failure scenario included — see the tracker)"*; the conditional clause is gone. The count six is correct (`@pending-deprecation` at feature lines 22, 34, 48, 107, 118, 154).
- ~~P2 H3-a: the retirement record covered only the Gherkin half~~ → **fixed**. DEPRECATION.md gained note 0: *"**The collision has a Go half too.** Six unit tests in `internal/cli/proposal_readback_test.go` assert `Success` against the shared not-valid fixture; no tag filter reaches them, so they are re-pointed or inverted in the same change (078 tasks.md T003)."* **Residual raised as H1-c**: the notes preamble still says "all four".
- ~~P2 H3-b: the 4→6 pairing note explained only one of the two 1:2 families~~ → **fixed**. Note 4 now reads *"four deprecations, six replacements, because **two of the four are answered twice**"* and names both the structured-output and the compact family. Re-derived independently: 1 + 2 + 2 + 1 = 6, and the tracker's six replacement titles match the six `@pending-deprecation` scenario titles exactly.
- ~~P2 H3-c (first half): T003's reuse list named two step texts the new feature file does not contain~~ → **fixed**. The tension `Given` is gone from the list and the exchange-count `Then` is now in the file. **Residuals raised as H3-a and H3-b**: the list still enumerates two shared steps where the file shares far more, and the user-template `Given` near-miss is still unresolved.
- ~~Plan ADR-3 did not pin the alert mirror's `omitempty`~~ → **fixed**. ADR-3 § Decision now reads *"The mirror's three fields carry `omitempty` so a key the server omitted stays omitted here too … a reconstructed entry must not turn an absent `path` into `""`."* **Residual raised as C4-a**: the interface accord and T002 were not brought into line with it.

**New findings** (not raised in round 3, or raised and only half-closed):

- **C4-a** (new, created by the ADR-3 fix), **C5-a** (new, created by the K4-a fix), **K4-a** (new — a second field-list family), **H1-a** (round 3's H1-a with its polarity reversed by the K4-a fix), **H1-b** and **H1-c** (residues of the K4-b and H3-a fixes), **H3-a** / **H3-b** (residues of the H3-c fix), **H3-c** (round 3's H3-e, unchanged), **H3-d** (new).
- Four of the ten are direct side-effects of the round-3 fix pass: a decision was pinned in the plan without being propagated downstream (C4-a), a task was widened without widening the plan (C5-a) or the decision record (H1-a), and two enumerations gained an item without their sibling counts being re-derived (H1-b, H1-c).

---

## Consistency (P0 — contradiction)

### C1 | spec.md § Integration Boundaries ↔ plan.md § System Architecture — **PASS**
### C2 | spec.md § Behavioral Accord ↔ plan.md § System Architecture — **PASS**
### C3 | spec.md § Non-Behaviors ↔ plan.md § Architecture Decisions — **PASS**

Re-verified the load-bearing pairs: spec Non-Behavior 5 (*"must not perform its own read-back or issue any extra request"*) ↔ plan § System Architecture (*"the verdict cannot be lifted from the create's own POST document … and a second read is forbidden"*) and T003's rebind instruction; spec Non-Behavior 4 (*"must not reuse an existing failure code"*) ↔ ADR-1 Option 2's rejection; spec Non-Behavior 9 (*"must not embed the full server proposal document"*) ↔ ADR-3 Option 3's rejection. Non-Behavior numbering cited by T003 and its criteria (*"Non-Behavior 5"*, twice) is correct against the current spec's nine-item list.

### C4 | plan.md § Architecture Decisions ↔ interface-cli.md § Surface — **FAIL** (1 finding)

**C4-a — plan ADR-3 pins per-field `omitempty` on the alert mirror, so a rendered alert can carry fewer than three keys; interface-cli.md's field contract describes each alert as a fixed three-key object, and T002 carries neither rule.**

plan.md ADR-3 § Decision (the sentence added in the round-3 fix pass):

> The mirror's **three fields carry `omitempty`** so a key the server omitted stays omitted here too — the success path emits the server document verbatim, and **a reconstructed entry must not turn an absent `path` into `""`**.

interface-cli.md § stdout — machine formats, field contract:

> `validation_alerts` | this failure only, and only when the server attached at least one alert (`omitempty`) | every alert the server attached, **each `{severity, path, message}`** with the server's own values — **the same three keys the server's entries carry**.

The accord's `omitempty` note attaches to the *outer* `validation_alerts` key only ("only when the server attached at least one alert"), and its description of an entry is a fixed triple. Under the plan's decision an entry the server sent without a `path` renders as `{severity, message}` — two keys — which the accord's contract does not describe. An agent written against the accord will index `path` unconditionally.

tasks.md T002 resolves it in neither direction. Its Scope names *"the neutral `output.ValidationAlert{Severity, Path, Message}` with the declared-here/populated-in-cli comment pattern"* — no field tags, in pointed contrast to the same sentence's `"ErrorDetail.ProposalID` + `ErrorDetail.ValidationAlerts` (`omitempty`)"` — and its criteria say only *"(server key spellings `severity`/`path`/`message`)"*. No scenario covers a partial alert entry. A Builder implementing from T002 or the accord ships three non-`omitempty` strings; a Builder implementing from ADR-3 ships three `omitempty` strings. Both satisfy their source and they are not the same envelope.

**Remedy**: pick one. If ADR-3's per-field `omitempty` stands, add a presence row to the accord's field contract (*"each entry carries the keys the server's entry carried; a key the server omitted is absent, not empty-stringed"*) and put the three field tags in T002's Scope beside `ErrorDetail`'s. If it does not, drop the sentence from ADR-3.

### C5 | plan.md § System Architecture / § Implementation Strategy ↔ tasks.md T001 § Scope — **FAIL** (1 finding)

**C5-a — T001 now owns four prose edits inside `internal/cli`; plan.md describes exactly one change to that prose (an added paragraph) and its Phase-2 sweep list contains no `internal/cli` site.**

tasks.md T001 § Acceptance criteria:

> The prose this task **does** edit, because each is a present-tense claim the new code falsifies: `exitcode_test.go`'s header count (eight → nine); `exitcode_test.go`'s "ExitCode maps the categories that have producers today…" enumeration; `dispatch.go`'s matching enumeration on the `Outcome` block; and `exitcode.go`'s "the full 0–6 convention is published now" (**already stale since 054, and owned here because T004's sweep does not reach `internal/cli`**).

plan.md ADR-1 § Decision states the only `internal/cli` prose change the architecture knows about:

> The `exitcode.go` narrative comment **gains a new 078 paragraph** after the 054 one; 054's own sentence … is not edited.

plan.md § System Architecture part 6 and § Implementation Strategy Phase 1 match that scope — *"the added comment paragraph"* — and Phase 2's enumerated site list is entirely outside `internal/cli` (`docs/guides/…`, `docs/explanation/…`, the operator-orientation feature line and step regex, `thenMeaningEachCode`'s loop bound, the self-containment guard string, `proposal-drafting/SKILL.md`). Three of T001's four edits are therefore in no plan phase at all, and the fourth (`exitcode.go:8`) is in a file the plan says gains a paragraph and loses nothing.

T001's own parenthetical states the reason the plan does not cover it — *"T004's sweep does not reach `internal/cli`"* — which is a correct observation about the plan and an admission that the plan has a hole, not a fix for it. This is the mirror image of round 3's C5-a: that finding was closed by widening the plan to match T003; the K4-a fix widened T001 without the same move. A reader taking plan.md as the architecture of record sizes T001 as constant + three arms + one added paragraph + one skill row.

Verified at the source, so the edits themselves are warranted: `exitcode.go:8` reads *"Per ADR-2 the full 0–6 convention is published now"* (present tense, false since 054); `exitcode_test.go:5` reads *"the **eight** frozen exit codes"*; `exitcode_test.go:21–27` enumerates the producer-backed set and stops at 054's StaleWrite; `dispatch.go:47–65` carries the same enumeration on the `Outcome` type doc. Nothing tests any of them.

**Remedy**: extend plan.md § System Architecture part 6 and § Implementation Strategy Phase 1 to say that Phase 1 also sweeps the present-tense convention enumerations inside `internal/cli` (`exitcode.go`'s ADR-2 sentence, `exitcode_test.go`'s header count and producer enumeration, `dispatch.go`'s `Outcome` doc), and state the Phase-2 property's boundary explicitly (*"Phase 2 covers every convention-stating surface outside `internal/cli`; the in-package ones ride with the constant in T001"*). Change nothing in T001.

### C6 | interface-cli.md § Surface ↔ features/*.feature Given/When/Then — **PASS**

Every field, token, and code a step names is defined in the accord: `kind "invalid-create"`, `proposal_id`, `validation_alerts`, the cause (`message`) and the remedy (`next_step`), `read_back`, `valid`, `available_transitions`, exit `8`, exit `0`, and the non-zero API-error code. No step references a surface the accord does not define, in either feature file.

### Passed (4/6)

spec↔plan boundary alignment, spec↔plan behavioral alignment, spec↔plan non-behavior exclusion, interface↔scenario step coverage.

---

## Completeness (P1 — gap)

### K1 | spec.md § Driving Scenarios / § Validation Scenarios → features/*.feature — **PASS**

All eight driving scenarios and all four validation scenarios have Gherkin equivalents; the file's remaining six are `Proposed:`-tagged interface/plan derivations or success re-pins. 18 scenarios (14 behaviour + 4 `@validation`), counted against the file. The three `Rule:` blocks quote the spec's three User Scenarios verbatim (backticks dropped, as the house convention has it).

### K2 | spec.md § Integration Boundaries → interface-*.md — **PASS**
### K3 | plan.md § Implementation Strategy → tasks.md — **PASS** (Phase 1 → T001–T003, Phase 2 → T004)

### K4 | plan.md ADR-3 (envelope extension) → tasks.md T002 § Scope — **FAIL** (1 finding)

**K4-a — two landed enumerations the envelope extension falsifies are in no task's scope; T002's field-list criterion names the two inside `ErrorDetail`'s doc comment and stops there.**

tasks.md T002 § Acceptance criteria:

> `ErrorDetail`'s doc comment carries **two** spelled-out field lists — the declaration-order enumeration and the "NextStep, Feature, Status, and Body are omitempty" sentence — and both are updated, along with `errorenvelope_test.go`'s matching pinned list.

Both are real and correctly identified (`internal/output/error.go:21–28`; the pinned `order` slice at `errorenvelope_test.go:191–192`). Two sibling enumerations in the same two files are not named by T002 or by any other task, and both go stale the moment the two fields land:

- `internal/cli/errorenvelope.go:52–64` — `errorEnvelopeFor`'s doc comment is a **bulleted field-mapping list**: *"Message ← d.Cause … NextStep ← d.NextStep … Feature ← d.Feature … Kind ← kind(d.Category) … Status ← … Body ← …"*. T002's Scope names the *code* change here (*"`errorEnvelopeFor` mapping the carried id/alerts"*) but not this list, which is the only place the mapping is described in prose and is the exact list a reader consults to learn what the envelope carries.
- `internal/output/error.go:46–47` — `Kind`'s doc comment enumerates the taxonomy: *"the lowercased taxonomy term (always present): usage / runtime / network / api (plus the 015-widened permission / rate-limit)"*. It already omits `stale-write` (054) and will omit `invalid-create`. interface-cli.md states the full set explicitly — *"`"invalid-create"` — joins `usage` / `runtime` / `network` / `api` / `permission` / `rate-limit` / `stale-write`"* — which is exact against `kind()`'s seven arms, so the accord and the landed comment will disagree by two tokens with nothing owning the difference.

This is structurally the same gap K4-a named last round for the exit-code family, in the field-list family. Nothing tests either site: the `len` + comma-ok guards check maps, and the pinned `order` slice checks `ErrorDetail`'s declaration order, not any comment.

**Remedy**: add both to T002's Scope and fold them into the same criterion — *"every spelled-out field or kind enumeration in `internal/output/error.go` and `internal/cli/errorenvelope.go` is updated: `ErrorDetail`'s two field lists, `Kind`'s taxonomy list (which also gains `stale-write`), `errorEnvelopeFor`'s mapping bullets, and `errorenvelope_test.go`'s pinned order list."*

### K5 | interface-cli.md § Interactions → features/*.feature — **PASS**

The accord's four pinned invariants now all have Gherkin behind them: the exchange count (`invalid-create-outcome.feature:30`), the pre-stdout-write ordering (`:55` *"stdout will be empty"* in the compact failure scenario), the template bypass (`:200`), and the per-state exit codes (every scenario's closing `Then`). The exchange-count step is byte-identical to the sibling's, so one definition serves both suites, as T003 now requires.

### K6 | spec.md § User Scenarios → interface-cli.md — **PASS**

### Passed (5/6)

spec driving+validation scenarios→Gherkin, spec boundaries→interface file, plan phases→tasks, interface interactions→scenarios, spec user scenarios→interface coverage.

---

## Coherence (P2 — drift)

### H1 | Cited counts and claims across plan.md, tasks.md, DECISIONS.md, DEPRECATION.md, and landed code — **FAIL** (3 findings)

**H1-a — DECISIONS.md records that `exitcode.go`'s two `0–6` sentences are historical and "not edited"; tasks.md T001 now edits one of them and calls it present-tense. The decision record and the task contradict each other outright.**

`.score/memory/DECISIONS.md`, 078 ADR-1 entry (final sentence):

> `exitcode.go`'s narrative correctly says `0–6` **twice** as a HISTORICAL claim about 004's originally-published band **and is not edited** — the 078 paragraph is appended after 054's.

tasks.md T001 § Acceptance criteria:

> The prose this task **does** edit, **because each is a present-tense claim the new code falsifies**: … and `exitcode.go`'s **"the full 0–6 convention is published now"** (already stale since 054 …).

Both sentences describe the same two occurrences (`exitcode.go:8` and `:20`) and they disagree on both the characterization and the action: DECISIONS says both are historical and neither is touched; T001 says `:8` is present-tense and is edited. Round 3 raised this as H1-a in the opposite polarity (plan and tasks named one sentence, DECISIONS named two); the fix moved tasks.md to the correct reading of the code and left the decision record on the old one. DECISIONS.md is the artifact a future spec consults for 078's precedent, so the stale half is the one that propagates.

Verified at the source: `exitcode.go:8` is *"Per ADR-2 the full 0–6 convention is published now"* — present tense, and false since 054 added code 7. `exitcode.go:19–20` is *"the first code beyond the originally-published 0–6 band"* — genuinely historical and correctly frozen by T001.

**Remedy**: amend the DECISIONS entry to match T001 — *"`exitcode.go` says `0–6` twice: `:20`'s "the first code beyond the originally-published band" is HISTORICAL and frozen; `:8`'s "the full 0–6 convention is published now" is PRESENT-TENSE, already stale since 054, and is corrected here because the Phase-2 sweep does not reach `internal/cli`."*

**H1-b — T004's Risk note still counts three `internal/build` sites after its Scope was widened to four.**

tasks.md T004 § Scope now names four sites inside `internal/build`: the paired step regex (`operator_orientation_bdd_test.go`), `thenMeaningEachCode`'s loop bound, `thenMeaningEachCode`'s failure message, and `surface_self_containment_guard_test.go`'s `knownSafeTokens` entry.

tasks.md T004 § Risk:

> Only **one** of **the three `internal/build` sites**  silently weakens — `thenMeaningEachCode`'s numeric loop bound … **The other two** are stale *text*: the step regex … and `surface_self_containment_guard_test.go`'s string …

The note's count and its enumeration both predate the round-3 fix that added the failure message. Verified at the source: `operator_orientation_bdd_test.go:93` (step regex), `:246` (loop bound), `:248` (failure message), `surface_self_containment_guard_test.go:58` (guard token) — four. The note now under-counts exactly the site it was rewritten to add, which is the failure mode round 3's K4-b named.

**Remedy**: *"Only **one** of the **four** `internal/build` sites silently weakens … The other **three** are stale *text*: the step regex …, `thenMeaningEachCode`'s failure message (which states the range and goes false with it), and `surface_self_containment_guard_test.go`'s string …"*

**H1-c — DEPRECATION.md's notes preamble says "all four" after a fifth note was inserted.**

`.score/memory/DEPRECATION.md` § Pending Deprecations:

> Notes on executing this removal (**all four** verified against the runners during 078's pre-implementation guard):
>
> 0. **The collision has a Go half too.** …
> 1. **Three of the four `@deprecate` scenarios actually execute** …
> 2. **Retagging alone is inert.** …
> 3. **"→ active" does not apply uniformly to the six replacements.** …
> 4. **Pairing is not 1:1 — four deprecations, six replacements** …

Five notes, numbered 0–4, under a preamble that counts four. The round-3 fix inserted note 0 (correctly, to avoid renumbering the citations that reference notes 1–4) and left the count. The claim the preamble makes — that each note was verified against the runners — is also the weakest fit for note 0, whose subject is Go unit tests that no runner reaches.

**Remedy**: *"Notes on executing this removal (**all five** verified against the runners and the Go tests during 078's pre-implementation guard)"*, or drop the count and say *"each verified at the source"*.

### H2 | Detail symmetry across spec↔plan and plan↔tasks — **PASS**

### H3 | Scope alignment across tasks.md, features/*.feature, and interface-cli.md — **FAIL** (4 findings)

**H3-a — T003's sibling-step-reuse rule enumerates two shared step texts; the new feature file reuses twenty-three of the sibling suite's step texts verbatim.**

tasks.md T003 § Scope note — the BDD harness:

> Where a step text already exists in the sibling suite — **the connection-context `Given` and the exchange-count `Then`, both of which this file now uses verbatim** — register the same implementation rather than a near-duplicate

Both named steps are correct. The apposition reads as the shared set, and it is not: derived from `invalid-create-outcome.feature`'s full step inventory against `post_create_validity_read_bdd_test.go:96–153`, twenty-three step texts are byte-identical to a sibling registration —

| Kind | Verbatim-shared step texts |
|---|---|
| `Given` (8) | `a complete connection context with a stored token`; `the created proposal reads back as not valid with the alert "…" and no available transitions`; `the created proposal reads back as not valid with one validation alert`; `the created proposal reads back carrying no validity field`; `the created proposal reads back as valid with no validation alerts`; `the created proposal reads back as valid with one alert of severity "…"`; `the create succeeds but the read of the created proposal cannot reach the server`; `the proposals endpoint rejects the create` |
| `When` (2) | `an agent runs "glassfrog …"`; `an agent creates a proposal with that template selected` |
| `Then` (13) | `the created proposal will be printed with its "…" id and "…" status`; `the created proposal will be printed with its "…" id`; `the result will report the validity as "…"`; `the create will have been followed by exactly one read of the created proposal`; `the command will exit with code (\d+)`; `the command will exit with a non-zero API-error code`; `the result will carry the alert with its severity, path, and message`; `the result will report that the server stated no verdict on the draft`; `the structured result will contain the created proposal's "…" id`; `the structured result will carry "…", "…", and "…"`; `the advisory will be rendered in the selected machine format, carrying "…" as true`; `the compact line will carry the created "…" id, its status, and the change count`; `it will carry the validity token and the alert count` |

Only two of the new file's read-back fixture `Given`s are genuinely new (`… as not valid with no validation alerts`, `… as not valid with one validation alert and a non-empty transition set`), and the two success re-pin scenarios (`:108`, `:119`) are composed **entirely** of sibling step texts. A Builder who takes the pair as the shared set writes twenty-one near-duplicate implementations of steps whose semantics are already fixed in the sibling suite — the drift the rule exists to prevent. Round 3 raised the same clause for naming a step the file does not contain; the fix corrected the entries without correcting the scope.

**Remedy**: replace the apposition with a rule that does not depend on an enumeration — *"Where a step text already exists in the sibling suite (twenty-three of this file's step texts do, including every step of the two success re-pin scenarios), register the same implementation rather than a near-duplicate; diff the two files' step texts before writing any definition."*

**H3-b — the user-template `Given` is a near-miss pair across the two feature files and T003's instruction resolves it in neither direction.**

`invalid-create-outcome.feature:197`:

> And **a user template referencing only proposal fields**

`post-create-validity-read.feature:158`, registered at `post_create_validity_read_bdd_test.go:110`:

> And **a user template referencing only the proposal fields the create rendered before the verdict existed**

The two scenarios then share the following `When` **verbatim** (`an agent creates a proposal with that template selected`, both files), so the pair is one shared step and one divergent one for the same fixture concept. tasks.md T003:

> and **check the user-template `Given` against the sibling's longer wording so the two do not become near-miss variants**

"Check … so the two do not become near-miss variants" does not say which way to resolve it, and the two available resolutions are not equivalent: reusing the sibling's wording means editing the feature file (which T003's Scope does not authorize for this file beyond tag changes), while keeping the short wording means registering a second template-fixture `Given` — the near-duplicate the same sentence forbids two clauses earlier. Round 3's remedy asked for one clause stating that the divergence is deliberate; the fix reworded the instruction instead. The semantics do differ (the sibling's pins pre-verdict field paths; this one only needs a template that is never rendered), so a distinct step is probably right — but no artifact says so.

**Remedy**: settle it in T003 — *"the user-template `Given` is deliberately its own phrasing: the sibling's pins pre-074 field paths, this one needs only a template that is never rendered, so it gets its own definition"* — or align the feature file's wording with the sibling's and reuse the definition.

**H3-c — the zero-alert human diagnostic layout is pinned only in tasks.md T002; interface-cli.md leaves it open on a detail plan.md assigns to the interface phase, and no scenario exercises it.**

plan.md § What This Plan Does Not Cover:

> **Protocol-level contracts** — the exact envelope field spellings …, and **the diagnostic's final wording**: the interface phase (`interface-cli.md`).

interface-cli.md § stderr — human formats states the with-alerts shape and the no-alert exclusion, but not the no-alert *layout*:

> Alert-line shape: two-space indent, `<severity> <path>: <message>`, one line per alert in the server's order. **When the server attached no alerts, no alert lines appear between cause and next step.**

tasks.md T002 § Acceptance criteria settles it, and nowhere else does:

> An invalid draft with **zero** alerts renders the human diagnostic **exactly as every other single-line failure does (`cause — next step`, no alert block and no stray blank line)**; with one or more alerts the alert lines sit between them.

The accord's worked example puts `" — <next step>"` on its own line after the alert line, so a reader of the accord alone could conform either way — one `cause — next step` line, or a two-line form with a bare `" — next step"` continuation. Landed `renderDiagnostic` (`internal/cli/diagnostic.go:330–336`) returns `d.Cause + " — " + d.NextStep`, so T002's reading is the one that matches the chain it extends — which makes the accord's silence the defect, not T002's decision. No scenario pins it either: the only zero-alert scenario (`invalid-create-outcome.feature:60`) selects `--output json`. Unchanged from round 3 (H3-e).

**Remedy**: move T002's sentence into interface-cli.md § stderr, and add a human-format zero-alert assertion — a `full`/`compact` variant of the zero-alert scenario, or a second `Then` on it.

**H3-d — the feature file's Rule 2 quotes the CI User Scenario ("exit non-zero, so the job fails") and carries only exit-`0` success scenarios.** *(SUSPECTED)*

`invalid-create-outcome.feature:162–186`:

> Rule: Fail the build when the write produced a dead draft
>   # … as a CI pipeline, / # I want an invalid create to exit non-zero, so the job fails instead of passing on a dead draft.
>
>   Scenario: A valid draft still succeeds … And the command will exit with code 0
>   Scenario: A valid draft carrying alerts still succeeds … And the command will exit with code 0

Neither of the Rule's two scenarios asserts a non-zero exit; both assert exit `0` on a *valid* draft. Every scenario that does exercise the CI-relevant behaviour (`exit with code 8`) sits under Rule 1, *"Branch on the exit code instead of parsing output"* — the AI-agent User Scenario. So spec § User Scenarios' second entry is realized, but not under the Rule that quotes it, and the Rule as titled has no demonstrating scenario.

Marked SUSPECTED because there is a defensible reading: the two scenarios are the negative controls for the rule (a *valid* draft must not fail the build), and Rule-block placement is a scenarios-phase judgement, not a contract. But the placement reads as an accident of grouping — the failure scenarios are all in Rule 1 and the successes split across Rules 1 and 2 — and the tell is that Rule 2's own title describes behaviour nothing under it asserts.

**Remedy**: either move one failure scenario (or add a CI-shaped one) under Rule 2, or state in the Rule's placement why the success scenarios belong there.

### H4 | plan.md phase structure ↔ tasks.md phase references — **PASS**

Phase 1 (T001→T002→T003, strictly sequential) and Phase 2 (T004, depends on T003) match plan § Implementation Strategy in ordering, grouping, and dependency direction. Task counts agree (3 + 1 = 4). No task references a phase the plan does not define. Scenario ownership across the three Phase-1 tasks is complete and non-overlapping: T001 owns the two registry scenarios, T002 the envelope-uniformity and zero-alerts scenarios, T003 *"every behavior scenario except"* those — with the explicit clause *"all of which this task's harness must still execute"* closing the gap between reference and execution. The four `@validation` scenarios are unowned by design (held out, `@wip` retained).

### Passed (2/4)

Detail symmetry, phase coverage.

---

## Cross-artifact facts verified at the source

Recorded because several artifacts assert them and a reader should not have to re-derive them.

- **Three `Outcome` switch sites, all with non-fatal defaults** — `ExitCode` (`exitcode.go:44`, default `codeInternalError`), `kind` (`errorenvelope.go:24`, default `"runtime"`), `String` (`dispatch.go:114`, default `fmt.Sprintf("Outcome(%d)", int(o))`). plan, tasks, interface, and DECISIONS all say three. ✔
- **Six mirror lists, three length-guarded table functions, two files** — `exitcode_test.go`: `publishedCodes:10`, `outcomeCodes:34`, `want:46` (len guard `:56`), `want:88` (len guard `:93`); `errorenvelope_test.go`: `cases:21`, `allOutcomes:53` (len guard `:54`). `String()` has no table test. T001's enumeration is exact. ✔
- **`exitcode_test.go:5` says "the **eight** frozen exit codes"** — T001's header-count criterion is exact. ✔
- **T003's six-test list is complete and its dispositions are exact** — the only `runProposalCreate`-driving users of the not-valid fixture `proposalReadBackBody` (`:21`, `"valid":false` with one alert) are `:374`, `:432`, `:461`, `:593`, `:643`, `:786`; the other uses (`:43`, `:59`, `:155`) drive `readBackProposalVerdict` directly. Both table tests' shared loop bodies `t.Fatalf` on `outcome != Success` (`:405`, `:656`) and then assert success-shaped stdout/stderr unconditionally, so a per-case branch is genuinely required. `TestRunProposalCreate_PreChangeUserTemplateStillRenders` drives `proposalReadBackValidBody` (`:504`) and a network error (`:511`) — "needs no change" is exact. `proposalReadBackValidBody` exists at `:351` as the re-point target. ✔
- **The machine branch discards the decoded proposal** — `proposal.go:177` `_, readBackRaw, reason := readBackProposalVerdict(…)`; the human branch binds it at `:207`. T003's rebind instruction is necessary and correctly stated. ✔
- **The failure chain accepts a non-`ResponseError` value unchanged** — `refineClientError` (`me.go:276–286`) returns `err` untouched when no `*ResponseError` is reachable, so plan ADR-2's *"`refineClientError` passes it through unchanged"* holds. `reportFailure` (`me.go:252–268`) returns `d.Category`, so `Diagnose`'s arm is what produces exit 8; in a machine format it writes **only** stdout, so interface-cli.md's *"(nothing new; retry notices only)"* stderr cell is exact. ✔
- **The drift guard is fully derived — no code→name table needs an `8` entry** — `operatororientation.go:212` regexes `code(\w+)\s*=\s*(\d+)` out of `exitcode.go`, `:253` reads the backticked digits out of the skill, `:258` `diffSets` them bidirectionally; the `(7 -> "StaleWrite")` in the `ExitCodes` doc comment (`:181`) is an example, not an enumeration. T001's *"the guard reads only the backticked digit"* is exact. `plugin/skills/orientation/SKILL.md:67` says *"a fixed range, 0–7"* and its table carries `` `6` `` and `` `7` `` in backticks. ✔
- **All five repo-wide `0–7` occurrences are owned** — `plugin/skills/orientation/SKILL.md:67` (T001), `features/unequipped-agent-operators/operator-orientation.feature:45` and `internal/build/operator_orientation_bdd_test.go:93` (byte-identical, T004), `operator_orientation_bdd_test.go:248` (T004), `internal/build/surface_self_containment_guard_test.go:58` (T004). The numeric loop bound `for code := 0; code <= 7` is at `:246`. No sixth site exists. ✔
- **T004's non-range site characterizations are exact** — `docs/guides/how-to-read-exit-codes.md`'s table ends at `` `6` `` with no `7` row, carries `*(reserved — see note)*` on 3 and `*(reserved)*` on 4–6, and the *"no command produces them yet"* prose at `:37`. `docs/explanation/how-failures-are-reported.md:10` scopes the format-aware chain to *"a transport, API, or decode error"* and does **not** enumerate kind tokens, so T004's conditional *"if the document enumerates kinds"* is correct. `plugin/skills/proposal-drafting/SKILL.md` references orientation only for pagination (`:48`) and says nothing about validity. ✔
- **The remedy names a command that exists** — `glassfrog proposal grammar` is landed (`internal/cli/proposal_grammar.go:135`), so ADR-5's and the accord's `next_step` points at a runnable read. ✔
- **One runner, exact path, `~@wip` only — and no runner filters `~@deprecated`** — `post_create_validity_read_bdd_test.go:31` `Paths` names only the sibling feature by exact path, `:32` `Tags: "~@wip"`. Every one of the 60 `Tags:` declarations under `internal/` filters `~@wip` (four add a positive tag); none mentions `@deprecated`. No Go file anywhere references `invalid-create-outcome.feature`, and no `InvalidCreate`/`invalid-create` symbol exists yet — this is a spec-only branch, so `@wip` on the new file's behaviour scenarios is correct. ✔
- **Tag and pairing counts** — `invalid-create-outcome.feature`: 18 scenarios (14 behaviour + 4 `@validation`), 6 `@pending-deprecation` at `:22`, `:34`, `:48`, `:107`, `:118`, `:154` (5 behaviour + 1 `@validation`). `post-create-validity-read.feature`: 4 `@deprecate` at `:35`, `:124`, `:146` (active, each asserting `exit with code 0` against a not-valid read-back) and `:205` (`@validation @wip @deprecate`). plan § Risks' line citations (35, 124, 146) are current. DEPRECATION.md's checklist names all ten by exact title with no extras and no omissions, and note 4's pairing (1 + 2 + 2 + 1 = 6) re-derives correctly from the two files. ✔
- **`omitempty` / empty-array agreement on the outer key** — spec § Failure rendering, plan ADR-3 § Consequences, interface § field contract, T002's third criterion, and `invalid-create-outcome.feature:66` all say `validation_alerts` is **absent**, never `[]`, when the server attached none. No artifact promises an empty array. (The per-*entry* field presence is C4-a.) ✔
- **Success-state count** — "three success states" in spec § Validation Scenarios, plan ADR-4 (twice), interface § four verdict states and § Consistency Notes, DECISIONS, DEPRECATION, and `invalid-create-outcome.feature:158`. No artifact says four success states, and every artifact that counts them repeats that a `valid: true` draft carrying alerts is the *valid* state. ✔
- **Kind-token set** — `kind()` has seven arms (`usage`, `runtime`, `network`, `api`, `permission`, `rate-limit`, `stale-write`); interface-cli.md's `kind` row enumerates exactly those seven as what `invalid-create` joins. The accord is exact against the switch; `ErrorDetail.Kind`'s doc comment is not (K4-a). ✔
- **The three Rule blocks quote the spec's User Scenarios verbatim** — `invalid-create-outcome.feature:17–19`, `:163–165`, `:189–191` against spec § User Scenarios, word for word (backticks dropped per house convention). No reworded Connextra text. ✔

---

## Checklist Correlation

- Checklist **CH-10** (P0, no runner for the new feature file) and **CH-13** (P0, Go tests pin the old outcome) remain closed; both were re-verified mechanically this round rather than accepted from the fix report. CH-13's residue from round 3 (**C5-b**) is now closed too — T003's sub-count and disposition are both correct against the landed file.
- Checklist **P2-2** (T001's narrative-freeze criterion) correlates with this round's **C5-a** and **H1-a**. The task side is now right; the plan and the decision record are the two artifacts left behind.
- Checklist **P2-4** (exchange-count step) is closed — see K5.
- Checklist **P2-3** (zero-alert human diagnostic shape) correlates with **H3-c**, unchanged: tasks.md pins the shape, interface-cli.md still does not, and the zero-alert scenario is still machine-format only.
- Checklist **P2-6** (stale line citations) stays closed; the only line citations remaining in any 078 artifact are plan § Risks' `35, 124, 146`, all three current.
- Checklist **P2-1** (`stdin` listed as an `--output` value in interface-cli.md § Command) is **not a finding**. Re-verified at the source: `internal/output/selection.go` declares `const reservedStdin = "stdin"` and `internal/cli/root.go`'s `--output` help text documents it, so `stdin` is a real selector. The accord is correct; the earlier checklist entry was measuring against `format.go`'s `supportedFormats` literal, which is the *format* vocabulary, not the selector vocabulary. Recorded here so it is not re-raised a fourth time.
- No checklist finding was re-evaluated; each is referenced for correlation only. `checklist.md` itself is a round-2 record and is stale against the current artifacts.

---

## Governance Notes

- **Full artifact set present** — spec.md, plan.md, one interface file, two feature files, tasks.md, checklist.md. No check was skipped for a missing artifact.
- **Memory artifacts included beyond the matrix** — `.score/memory/DECISIONS.md` (three 078 entries) and `.score/memory/DEPRECATION.md` (one `[decision]` entry + § Pending Deprecations) were folded into the H1 and H3 evaluations, because in this feature they carry claims the spec-directory artifacts also make. The relationship matrix does not name them; treat H1-a and H1-c as extensions of H1 rather than as separate check types.
- **Landed-code claims are in scope** — this feature reclassifies an outcome at an existing seam, so most cross-artifact claims are claims about `internal/cli`, `internal/output`, and `internal/build`. Every one was re-read at the source; the list is under § Cross-artifact facts verified at the source.
- **Sibling-artifact drift not evaluated** — 074's own spec-directory artifacts are deliberately left at their historical wording (interface § Consistency Notes, per the 031-deprecation precedent). Analyze did not check 074's artifacts against 078's; that is the deprecation record's job and it is discharged in `.score/memory/DEPRECATION.md`.
- **Two round-3 findings are half-closed rather than open or fixed** — K4-b (Scope fixed, Risk note not: H1-b) and H3-a (note added, preamble count not: H1-c). Both were re-derived from the files, not from the previous analyze.md.
- **Advisory only** — two P0 findings are recorded; neither blocks. C4-a is the one whose failure mode reaches a caller (two Builders reading two artifacts ship two different envelopes); C5-a and H1-a are documentation-of-record defects whose cost lands on the next spec that cites 078.
