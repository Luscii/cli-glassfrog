# Risk: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Round**: 1 (H-5 amended 2026-08-08 after the format-aware advisory landed — see H-5)
**Date**: 2026-08-08
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Hazards**: 10 | **Controls**: 23 | **Unacceptable residual (Red)**: 1

> **Matrix**: using the default 3×3 traffic-light matrix — PROJECT.md defines no risk acceptability matrix.
> **Regulatory bridge**: omitted — PROJECT.md declares no Regulatory Context.
> **Order note**: risk ran here via `/score:guard --pre`, after scenarios and tasks rather than in its canonical position (plan → interface → **risk** → scenarios). The formal test-gap section is a re-run artifact and is skipped, but where a control's strength depends on an existing scenario or task criterion, that grounding is named inline.

---

## Risk Register

| H | Hazard | Source | Sev | Prob | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | The undeclared verdict fields disappear upstream and the feature reports "no verdict" forever, undetected | spec § Assumptions; plan ADR-1 | High | Med | RC-1, RC-2 | 🟡 Yellow |
| H-2 | Validity is computed after the create returns, so the read-back reports a favourable or absent verdict on a draft the server later marks invalid | plan § Risks 2; spec § Clarifications | High | Low | RC-3, RC-4 | 🟡 Yellow |
| H-3 | The doubled request count exhausts the organization's hourly budget | spec § Integration Boundaries; plan § Risks 4 | Low | Low | RC-5, RC-6 | 🟢 Green |
| H-4 | An invalid create exits 0, so an exit-code-gated caller proceeds as though governance changed | spec § Non-Behaviors 2; plan § Risks 5 | High | High | RC-7, RC-8*, RC-9* | 🔴 **Red** |
| H-5 | In a machine format, "verdict unavailable" is indistinguishable from "no verdict stated" | interface-cli.md § machine formats; plan ADR-5 | Med | Med | RC-10, RC-23 | 🟢 Green |
| H-6 | The verdict is rendered for a different proposal than the one created | plan ADR-6; interface-spec.md § Interactions | High | Low | RC-11, RC-12, RC-13 | 🟢 Green |
| H-7 | Verdict lines leak onto `proposal get` / `propose` / `withdraw`, changing output other consumers parse | plan § Risks 3; ADR-4 | Med | Low | RC-14, RC-15, RC-16 | 🟢 Green |
| H-8 | The create's own 201 already carries the verdict, making the second exchange pure waste | spec § Assumptions; plan § Risks 1 | Low | Med | RC-17 | 🟢 Green |
| H-9 | A user template written against the pre-change view breaks when the create's view type changes | plan ADR-4 § Consequences | Med | Low | RC-18, RC-19 | 🟢 Green |
| H-10 | A render failure on the new template loses the created id from stdout on an otherwise successful write | plan § Verdict Assembly; interface-spec.md § Consistency Notes | High | Low | RC-20, RC-21, RC-22 | 🟡 Yellow |

`*` RC-8 and RC-9 are **not yet in place** — see H-4.

---

## Hazard Detail

### H-4 | An invalid create exits 0 — 🔴 unacceptable residual

**Hazard**: The server accepts a create, marks the draft invalid, and the CLI exits `0`. A caller that decides "did this work?" from the exit code concludes the governance change landed and proceeds — chaining `proposal propose` against a draft with no available transitions, or reporting success to the practitioner.

**Severity: High.** This is the original problem's automation half, undiminished. The CSG-2 incident cost a human-confirmed gated write, a dead draft, and web-UI cleanup; an agent that cannot see the failure repeats it at machine speed. The spec's own System Overview names this consequence.

**Probability: High.** It is not a failure mode — it is the specified behavior (spec Non-Behavior 2). Any exit-code-gated caller hits it on every invalid create. It is made concrete by this repo shipping the agent guidance that teaches exit-code interpretation, so the operator most likely to gate on the code is the one the project itself packages.

**Controls**:
- **RC-7** *(in place)* — the verdict is present in stdout in every output format, so a caller that reads output rather than the exit code can gate correctly. Grounded in the spec's accord and both render paths.
- **RC-8** *(not in place)* — the dependent Invalid-Create Outcome capability (backlog #76) renders an invalid create as a failure with its own exit code. Specified but unbuilt; this spec deliberately excludes it because it adds an `Outcome` at both registry sites.
- **RC-9** *(not in place)* — the packaged agent guidance could instruct reading the verdict rather than the exit code after a create. Not in this spec's scope, and not currently written anywhere.

**Residual risk: Red.** RC-7 reduces severity from High to Medium — the information exists, it is simply not where an exit-code-gated caller looks. Medium × High is Red under the default matrix.

**Acceptance rationale — and the recommendation**: this is a *known, documented, deliberately scoped* gap rather than an oversight; the spec, ADR-2, plan Risk 5, and checklist P2-1 all name it. But "documented" is not a control, and the matrix says Red must be reduced. Two ways to reduce it:

1. **Sequence #76 immediately after this**, so the window in which an invalid create exits 0 is measured in one PR rather than left open. This is the cleaner fix and the one the dependency graph already implies.
2. **Add RC-9 within reach of this spec** — one sentence in the agent-facing guidance telling the operator to read the verdict after a create, not the exit code. Cheap, and it reduces probability rather than severity.

Shipping this feature alone is still a strict improvement over today (a human reading output now sees the truth). The Red is about what an *agent* sees, and it stays Red until one of the two above lands.

### H-1 | The undeclared fields disappear upstream — 🟡 Yellow

**Hazard**: `valid` and `validation_alerts` are absent from `spec/glassfrog-api-v5.yaml`. If the server stops sending them, every create reports "the server stated no verdict" — indefinitely, and without anything noticing.

**Severity: High.** The feature's purpose evaporates while the command still looks healthy. The operator sees a benign-looking state that means "unchecked."

**Probability: Medium.** Two recorded precedents make this more than theoretical: the vendored contract sat three months stale with nobody noticing (LEARNINGS S7 — `info.version` did not move across +15 operations, and the vendor changelog listed neither of the two changes that mattered), and the platform does retire transition-only surfaces on purpose (S3, `include_legacy_id`). Undeclared fields carry no compatibility promise at all.

**Controls**:
- **RC-1** *(in place)* — graceful degradation: absence is a first-class reported state, never a fabricated verdict. The feature cannot start lying if the fields vanish; it can only go quiet. Grounded in the tri-state model (ADR-3) and asserted by T001's nil-decode criterion.
- **RC-2** *(in place, partial)* — the field comments and the spec's non-behavior record that both fields are undeclared, so the next reader knows the dependency is unguaranteed rather than assuming a contract.

**Residual risk: Yellow.** Severity drops to Medium because the failure is safe rather than false. **Detection is the gap**: nothing distinguishes "the server stopped sending `valid`" from "the server declined to state a verdict for this proposal," so the disappearance would be invisible. LEARNINGS S7's own suggested action — a drift check that diffs the vendored contract rather than trusting `info.version` — is the natural home for that detection and is not this spec's work. Worth noting that a contract *diff* would not catch this either, since the fields were never in the contract to disappear from.

### H-2 | Validity may lag the create — 🟡 Yellow

**Hazard**: If the server computes validity asynchronously, an immediate read-back sees either no verdict or a provisional favourable one on a draft it will shortly mark invalid.

**Severity: High.** The favourable-provisional case is the worst outcome the feature can produce: it actively reassures on the exact shape it exists to catch, which is worse than the status quo where nothing is claimed.

**Probability: Low.** A sibling aggregate on this same resource was observed lagging by hours (LEARNINGS F8, `response_summary`), which is what raises the question. Against that: the probe found `valid` and `validation_alerts` populated and internally consistent on every draft read, including two invalid ones, with no sign of a provisional state. No evidence of lag at create — but also no evidence against it, because settling that needs a real create.

**Controls**:
- **RC-3** *(in place)* — no inference: the CLI reports what the server says at read time and never derives a verdict, so a lagging server produces "not reported," not a fabricated pass.
- **RC-4** *(in place, by exclusion)* — the spec forbids polling and bounded re-reading, so the reported verdict is always a single honest observation rather than a guess about convergence.

**Residual risk: Yellow.** RC-3 covers the absent case fully; it does *not* cover a provisional `valid: true`, for which no control exists and none is possible without a second read. **This is the highest-value thing to verify on the first real create after this ships**: capture the raw 201 and the raw read-back, compare, and record the finding. It settles H-2 and H-8 together.

### H-5 | Machine-format states collapse — 🟢 Green (reduced from Yellow, 2026-08-08)

**Hazard**: An agent parsing stdout sees a document with no `valid` key in two different situations — the server declined to state a verdict, and the CLI never managed to ask. It cannot tell which, so it cannot tell whether the write was checked.

**Severity: Medium.** The agent may treat an unchecked write as checked-and-unreported. It does not produce a false favourable verdict, which is what keeps this below High.

**Probability: Medium.** Read-back failures are ordinary operational events — a transient network fault or an exhausted budget is enough.

**Controls**:
- **RC-10** *(in place)* — the advisory names the cause and the remedy, and states that the created id stands. Present in every output selection.
- **RC-23** *(in place, added 2026-08-08)* — the advisory is **format-aware**: in a machine format it renders structurally, carrying `read_back`, which answers the one question the emitted document cannot — did the CLI manage to ask? All four verdict states resolve from `data.valid` plus `verdict_source.read_back`, with no prose parsing.

**Residual risk: Green.** RC-23 removes the collapse rather than documenting it. Probability of a read-back failure is unchanged at Medium — that is an operational fact, not a design property — but the *consequence* is no longer that an agent cannot tell whether the write was checked, because the advisory says so in a form the agent already parses. This hazard was the same defect checklist CH-4 and analyze C2 raised; the risk view's contribution was showing that the triggering condition is ordinary rather than rare, which is what argued against the cheaper fix of narrowing the spec's claim.

### H-10 | A render failure loses the id — 🟡 Yellow

**Hazard**: The render engine runs `missingkey=error`. If the new template references a field path that does not resolve, a successful create renders nothing and exits 1 — the created proposal exists, and its id is nowhere in stdout.

**Severity: High.** The id-loss consequence the spec forbids for read-back failures arrives here by a different door: the write happened, and the operator has no handle on it.

**Probability: Low.** The design removes the nil from the template's reach on purpose — labels are resolved in Go (RC-20) precisely because a `*bool` renders as truthy regardless of value, and the verdict is a value rather than a pointer on the view (RC-21) so there is no nil case to guard.

**Controls**:
- **RC-20** *(in place)* — display labels resolved in Go; templates see only strings and slices.
- **RC-21** *(in place)* — `Verdict` is a value field on the view, so no nil dereference path exists.
- **RC-22** *(in place, pre-existing)* — buffer-then-write leaves stdout empty rather than half-written, and T003 asserts all four states render in both formats.

**Residual risk: Yellow.** Note that this exposure is **pre-existing, not introduced**: a render failure on the shared `proposal` template already loses the id today (055). The new template widens the surface by two more field paths. Reducing it below Yellow would mean printing the id on the render-failure path, which contradicts buffer-then-write — a project-level convention, not this feature's call. Flagged so the tradeoff is visible rather than resolved here.

### Green hazards — brief

- **H-3** *(rate budget)*: RC-5 isolation means a rate-limited read-back can never fail the create; RC-6 is the existing backoff, and one scenario covers the post-retry case. One extra GET per create, against a governance rhythm of a handful of creates per day.
- **H-6** *(wrong proposal's verdict)*: RC-11 — the id comes only from the create's own response body, never from user input, a cache, or a store (the project maintains none). RC-12 — the emitted machine document is the read-back's own body carrying its own id, so it is self-verifying. RC-13 — the provenance line names the proposal the verdict came from.
- **H-7** *(leak onto sibling commands)*: RC-14 separate render key; RC-15 the byte-identical assertion in T003, which turns the guarantee into a test rather than an intention; RC-16 the registry exhaustiveness test.
- **H-8** *(redundant exchange)*: RC-17 — ADR-1 records the exact fallback, so discovering the 201 carries a verdict costs one deletion rather than a redesign. Severity is Low because the wasted call is one GET and the read-back's document is the later of the two if they ever disagreed.
- **H-9** *(user template breakage)*: RC-18 embedding rather than replacing the view, so field promotion preserves every existing path; RC-19 the compatibility assertion in T002/T005. Analyze K5 notes no *scenario* holds this promise — the control is real but tested one layer lower than the promise is made.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| 🔴 Red (unacceptable) | 1 | H-4 |
| 🟡 Yellow (acceptable with justification) | 3 | H-1, H-2, H-10 |
| 🟢 Green (accepted) | 6 | H-3, H-5, H-6, H-7, H-8, H-9 |

**One unacceptable residual.** H-4 is not a design defect — it is the boundary the spec drew, and the artifacts name it consistently. What the matrix adds is that "documented" does not discharge it: the recommended reduction is to sequence backlog #76 immediately after this feature, or to add the one-sentence agent guidance (RC-9) that tells the operator to read the verdict rather than the exit code.

**Two hazards resolve on evidence rather than design.** H-2 and H-8 are both settled by capturing the raw 201 and the raw read-back on the first real create after this ships, and recording the comparison. That single observation retires H-8, and either retires H-2 or turns it into a Red that needs a bounded re-read.

**One hazard has no detection path.** H-1's control makes the failure safe but silent. If the verdict quietly stops arriving, nothing in the system says so.

---

## Traceability Index

**Hazards → source**

| H | Source section |
|---|---|
| H-1 | spec.md § Assumptions ("The create response's own verdict fields are unobserved", "verdict route"); spec.md § Non-Behaviors 3; plan.md ADR-1 |
| H-2 | plan.md § Risks (row 2); spec.md § Clarifications ("How the verdict is obtained") |
| H-3 | spec.md § Integration Boundaries (rate limits); plan.md § Risks (row 4) |
| H-4 | spec.md § Non-Behaviors (row 2); plan.md ADR-2 § Consequences; plan.md § Risks (row 5) |
| H-5 | interface-cli.md § "stdout — machine formats" and § "stderr — the verdict advisory"; plan.md ADR-5 § Amendment |
| H-6 | plan.md ADR-6; interface-spec.md § Interactions ("Empty id short-circuits") |
| H-7 | plan.md § Risks (row 3); plan.md ADR-4 |
| H-8 | spec.md § Assumptions; plan.md § Risks (row 1); plan.md ADR-1 § Consequences |
| H-9 | plan.md ADR-4 § Consequences; interface-cli.md § "stdout — user template" |
| H-10 | plan.md § Verdict Assembly and Rendering Design; interface-spec.md § Consistency Notes (`missingkey=error`) |

**Controls → architectural grounding**

| RC | Grounded in |
|---|---|
| RC-1, RC-3 | ADR-3 tri-state model; T001 nil-decode criterion |
| RC-2 | ADR-3 field comments; spec Non-Behavior 3 |
| RC-4 | spec Non-Behaviors (no polling beyond the existing read-retry policy) |
| RC-5 | ADR-2 isolated read-back |
| RC-6 | Landed safe-method retry policy (017), inherited unchanged |
| RC-7 | spec Behavioral Accord § Verdict surfacing; both render paths |
| RC-8 | Invalid-Create Outcome (backlog #76) — **not in place** |
| RC-9 | Packaged agent guidance — **not in place, out of this spec's scope** |
| RC-10 | interface-cli.md § stderr; ADR-5 |
| RC-23 | ADR-5 amendment (format-aware advisory, 032's precedent extended to a success-path advisory); interface-spec.md `verdictSource` |
| RC-11, RC-12, RC-13 | ADR-6 local id lift; ADR-5 verbatim emission; interface-cli.md § stderr |
| RC-14, RC-15, RC-16 | ADR-4 separate resource key; T003 byte-identical + exhaustiveness criteria |
| RC-17 | ADR-1 § Consequences (documented fallback) |
| RC-18, RC-19 | ADR-4 embedded view; T002/T005 compatibility criteria |
| RC-20, RC-21 | plan § Verdict Assembly (labels resolved in Go; verdict as a value) |
| RC-22 | Buffer-then-write convention (pre-existing, 018/019) |
