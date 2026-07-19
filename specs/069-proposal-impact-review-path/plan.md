# Plan: Proposal Impact Review Path

**Feature**: 069-proposal-impact-review-path
**Role**: Shaper
**Inputs**: spec.md (069), PROJECT.md, DECISIONS.md (457 lines — the 062–068 operating-surface family entries are directly governing), LEARNINGS.md (passive), DEPRECATION.md (no entries touching the operator paths), the shipped plugin artifacts (`plugin/skills/*`, `plugin/agents/*`, `plugin/hooks/gated-commands.txt`), and the CLI command registry (`internal/cli`)

---

## System Architecture

The path ships as two cooperating plugin artifacts plus a registry file and a drift guard — the same shape as its five siblings (064–068), with one structural novelty: the workflow crosses back into the caller's context mid-way, because the operator's response decision sits between the review and the write.

**Components**:

- **`plugin/skills/proposal-impact-review/SKILL.md`** — the thin, discoverable, description-triggered entry point. Owns the WHEN (a circulating proposal awaits the operator's consent response; or the operator wants to know what a proposal would change for them) and the single-sourced WORKFLOW: take the `prp_` id (or surface what is pending via `proposal list`) → delegate the impact review to the reviewer subagent → present the drawn-together impact picture → the operator decides (`no_objection` / `bring_to_meeting` / not yet) → if a response is chosen, issue the gated `proposal respond <prp-id> --response <value>` through 063's confirmed write flow → return the recorded response and its `prr_` id. The skill also owns the two interaction moments (065 pattern: interaction lives in the skill): eliciting the target proposal when only "what's pending on me?" is known, and receiving the operator's response decision after the review.

- **`plugin/agents/proposal-impact-reviewer.md`** — a NEW read-posture subagent that executes the review traversal in its own isolated context and returns only the synthesized impact picture. It reads the circulating proposal and its change set (`proposal get`, optionally `proposal list` to situate), draws the operator's footprint (`me`, `me roles`, `me actions`, `me projects`), reads back specifically-affected governance where the change set touches the operator's footprint (`roles <id>`, `domains <id>`, `policies <id>`), and composes the current-versus-proposed picture — including the honest "does not touch your current governance" case and the footprint-incompleteness qualifier (see Cross-cutting Concerns). It performs **no write of any kind**: its prompt fences every `proposal` write (create/propose/respond/withdraw — 067, 068, and this path's own write), and 063's PreToolUse hook gates any leak as backstop. The reviewer is structurally incapable of answering the proposal it reviews — the record step lives outside it.

- **`plugin/agents/proposal-impact-review-commands.txt`** — the single-sourced composed-leaf registry (the 066/067/068 file pattern): the review reads plus the one gated write, each as a command path (`proposal get`, `proposal list`, `me`, `me roles`, `me actions`, `me projects`, `roles`, `domains`, `policies`, `proposal respond`). Consumed by the agent artifact (its "Composed reads" section names exactly the read leaves), the skill (the respond leaf), and the drift guard.

- **`internal/build/proposal_impact_review_guard_test.go`** — the best-effort drift guard extending the gate-membership-posture pattern (067 ADR-5, 068 ADR-5): every registry leaf resolves in the shipped CLI command registry; `proposal respond` **is** a member of 063's `plugin/hooks/gated-commands.txt`; every other leaf is **not**. Posture consumers across the family: all-out (066), one-in (067), two-in-two-out (068), **one-in-nine-out (069)**.

**Flow**:

```
operator / caller agent
   │  (skill: WHEN + workflow, interaction moments)
   ├─► proposal-impact-reviewer subagent  (isolated context, reads only)
   │      proposal get ── change set, response_summary, deadline
   │      me / me roles / me actions / me projects ── footprint
   │      roles / domains / policies ── affected-governance read-backs
   │      └─► returns: synthesized impact picture (+ incompleteness qualifiers)
   ├─ operator reads the picture, decides (or stops — review stands alone)
   └─► gated write, in caller context, only on an explicit operator choice:
          glassfrog proposal respond <prp-id> --response <value>
          └─ 063 PreToolUse hook → human confirmation → server authorizes
```

No Go production code changes: the only Go change is the guard test. `plugin.json` is untouched (directory auto-discovery, per 066 ADR-2 precedent).

---

## Architecture Decisions

### ADR-1: Both forms — thin skill delegating to a new reviewer subagent

**Context**: The spec defers the delivery form (skill, agent, or both) to shaping. Five siblings (064–068) all resolved this the same way, for reasons that apply with full force here: the path's core promise is a synthesized impact picture rather than raw dumps (the review traversal is the widest read fan-out of any path yet — proposal + four `me` reads + up to three read-backs per affected role), and its central non-behaviors (never decide the response, never write outside the one gated respond) need a structure, not just prose.

**Options considered**:
1. **Skill only** — the caller executes the reads in its own context. Simple, but the raw reads flood the caller's context and "synthesized, not raw" rests on caller discipline; rejected by 064 for exactly this and nothing here differs.
2. **Agent only** — no discoverable, description-triggered entry point; the when/workflow knowledge has no home. Rejected by every sibling.
3. **Both** — skill owns when + workflow + interaction; agent executes the traversal in isolation and returns the picture.

**Decision**: Option 3 — both forms. Silent conformance to 064 ADR-1 through 068 ADR-1 (recorded because the spec explicitly deferred the form). `plugin/skills/proposal-impact-review/SKILL.md` + `plugin/agents/proposal-impact-reviewer.md`, additive siblings under 062 ADR-1's homes; the workflow is single-sourced in the skill and referenced (not restated) by the agent.

**Consequences**: Sixth skill, fifth agent. The `plugin/agents/` type gains another consumer; no restructure. The skill stays thin per the shipped 067/068 owner feedback (when / workflow / delegation / boundaries — no family positioning, no mechanics deferrals).

### ADR-2: A new `proposal-impact-reviewer` agent — not a reuse of the circulator or drafter

**Context**: 066 left open whether later write paths might reuse a prior path's subagent; 067 and 068 each resolved it by minting a new agent, because the shipped sibling's prompt fence forbade exactly the command the new path needed, and eroding a validate-pinned fence forces the sibling's re-validation.

**Options considered**:
1. **Reuse `proposal-circulator`** — its shipped prompt forbids `proposal respond` (a validate-pinned 068 invariant). Reuse erodes that fence and forces 068's re-validation; a union fence blurs both paths' guarantees.
2. **Reuse `proposal-drafter`** — same shape: its fence forbids `proposal propose/respond/withdraw`.
3. **New agent** — declarative agent files are cheap; eroded fences are not.

**Decision**: Option 3 — mint `proposal-impact-reviewer`. Silent conformance to the one-agent-per-path pattern (now uniformly resolved three times for the write-adjacent paths; this closes the family — 069 is the last operator path).

**Consequences**: Each path's agent keeps a fence exactly complementary to its own surface. The reviewer's fence is the strictest yet: it forbids **all four** proposal writes, including this path's own `respond` (see ADR-3).

### ADR-3: Two-phase flow — review in the subagent, decision and the gated respond in the caller context

**Context**: This is the genuinely new structural question. 067 and 068 ran their gated writes *inside* the subagent (hooks fire inside subagent calls; one coherent isolated execution). But 069's workflow is interrupted by design: the spec requires the *operator* to judge the impact and choose the response — an isolated subagent has no dependable channel to ask (the 065 finding), and the review must be complete and presented before the choice exists. Something must come back to the caller between the read fan-out and the write.

**Options considered**:
1. **Two delegations to one dual-mode agent** — the reviewer returns the picture; the skill re-invokes the same agent with the operator-chosen value to run the gated respond inside it. Conforms to 067/068's in-subagent write locus, but the second delegation re-briefs an agent to run one single-token-flag command, buys no isolation (tiny output), and requires the agent to carry a "record mode" — weakening its fence from "never writes" to "writes only when handed a value".
2. **Review in the agent; decision and the gated respond in the skill (caller context)** — the reviewer stays a pure read agent (064/065 fence shape); after the operator decides, the caller issues `glassfrog proposal respond <prp-id> --response <value>` and 063's PreToolUse hook fires in the caller context exactly as it does anywhere.

**Decision**: Option 2 — review in the agent, respond in the skill. **Announced divergence** from 067 ADR-3 / 068 ADR-3's in-subagent write locus, on their own reasoning: the write locus followed the workflow's shape (one coherent isolated execution), and 069's workflow shape is different — the operator decision structurally splits it at exactly the read/write seam. The divergence buys the path's central invariant structurally: the agent that draws the picture **cannot** be the thing that records the answer ("reviews inform, never decide" becomes partly architecture, not just prompt), and the review-without-response case (spec: the review stands alone) is the default exit rather than a suppressed branch. The confirmation guarantee is unchanged — 063 gates the caller's Bash identically, and the response value rides inline on the command line (`--response no_objection`), so the human confirms the exact payload (067's inline-payload principle, trivially satisfied by a one-token value). Confirmation stays layered: the skill narrates the proposal + chosen value before issuing, and 063's structural gate confirms at execution.

**Consequences**: The reviewer agent's fence forbids all proposal writes including `respond`; the skill owns the respond step and passes **only an operator-chosen value** — the skill must never infer or default the response from the review's content, and issues nothing when the operator declines or defers (review-only exit, a first-class outcome). Later paths (there are none — 069 closes the family) would have had a third write-locus precedent to weigh: in-subagent when the workflow is one coherent execution, in-caller when a human decision splits it.

### ADR-4: Pure composition of ten shipped leaves — no convenience command, footprint honesty pinned

**Context**: The spec forbids new commands/flags/capability (VISION Exclusion 2; "knowledge + guardrails, never capability"). The review needs: the proposal and its change set, the operator's footprint, and current-state read-backs for affected governance. All exist: 056 `proposal get`/`proposal list`, 011 `me`, 012 `me roles`, 013 `me actions`, 014 `me projects`, 025 `roles [id]`, 033 `domains <id>`, 034 `policies <id>`, 058 `proposal respond`. One footprint read has a known sharp edge: `me roles` **does not paginate** — when more roles exist than one page, it emits a single stderr incompleteness note and exits 0 (012's shipped shape; LEARNINGS/memory: a flag derived from it must be tri-state, never a silent false).

**Options considered**:
1. **A `proposal impact` / `proposal review` convenience command** — rejected as every sibling rejected its analogue: adds API surface, invites local governance logic.
2. **Compose the shipped leaves; defer flags to `glassfrog <cmd> --help`** — silent conformance to 062 ADR-3 and every sibling's ADR-4.
3. **Also compose `search` (041) / `tree` (026) / `policy` (034 singular)** for deeper affected-governance navigation — rejected as widening: change-set elements carry role ids, so `roles <id>` + role-scoped `domains`/`policies` suffice; deeper navigation is 064's path, and the operator can invoke it separately.

**Decision**: Option 2, with the leaf set fixed at the ten named above. Two review-correctness rules are pinned at plan level (they will become prompt content and validation scenarios): **(a) footprint honesty** — when `me roles` signals incompleteness (the stderr note, or the in-band pagination metadata in structured output), the impact picture must carry that qualifier forward, and the "this does not touch your current governance" verdict must be stated as *"not in the roles visible to this read (list incomplete)"* — never an unqualified negative derived from a known-incomplete list; **(b) reads inform, never decide** — the picture relates change set to footprint but computes no objection verdict and pre-gates no write on `available_transitions` (the 068 no-client-pre-gate invariant, inherited).

**Consequences**: Only Go change is the guard test. The path defers each command's read behavior (who walks pagination, who doesn't) to the command itself — the plan does not blanket-claim "all reads paginate"; it names the one that doesn't and requires the picture to say so.

### ADR-5: Drift guard extends the gate-membership-posture pattern — one-in, nine-out

**Context**: Every operator path carries a best-effort `internal/build` drift guard over a single-sourced leaf registry (062 ADR-4 lineage). Since 067, write-path guards also assert each leaf's *gate-membership posture* against 063's `gated-commands.txt` — "every operator write-path's guard asserts its leaves' gate-membership posture, whichever side of the gate they belong on" (067 ADR-5, generalized by 068).

**Options considered**:
1. **Existence-only guard** (the 064/065 read-path shape) — misses the path's central promise: that `respond` stays confirmed. A read/write swap across the gate would pass.
2. **Existence + per-leaf gate-membership** — `proposal respond` ∈ gated set; each of the nine reads ∉; per-leaf contract-facts, not count-satisfiable (the 067 lesson: anchor the specific gated member, don't over-derive "the single gated leaf").

**Decision**: Option 2 — `internal/build/proposal_impact_review_guard_test.go` over `plugin/agents/proposal-impact-review-commands.txt`, silent conformance to 067/068 ADR-5. The guard asserts: every leaf resolves to a command in the shipped CLI registry (command-path resolution, handling the `me <sub>` two-token paths and bare `me`); `proposal respond` is a member of `plugin/hooks/gated-commands.txt`; every other registry leaf is absent from it; and the agent artifact names each read leaf while the skill names the respond leaf. Explicitly partial (existence + gate-membership, not flags/prose/parser robustness); reductions stated, never silent.

**Consequences**: Pulling `proposal respond` out of 063's gated set — or a read into it — turns CI red; the artifacts cannot silently name a command the CLI lacks or a write the guardrail stopped confirming. Posture consumers now span all four shapes: all-out, one-in, two-in-two-out, one-in-nine-out.

---

## Cross-cutting Concerns

**Failure honesty (inherited spec accord)**: Any composed read failing mid-picture → the reviewer surfaces the failure by name and returns what it gathered, flagged incomplete — never invented data, never an abandoned review. A rejected respond (`403` plan-gate — 061 renders the possibility-framed diagnostic; `404`; `422` including the already-responded case) is surfaced by name from the CLI's own diagnostics; the path adds no retry (a retry is itself a gated write needing fresh confirmation — 063 ADR-5's structural no-blind-retry).

**Footprint incompleteness (the tri-state rule)**: The one silent-cap hazard in the composed set is `me roles` (ADR-4a). The reviewer's prompt must instruct reading the incompleteness signal (stderr note in human formats; in-band pagination metadata in `json`/`yaml`) and qualifying both the footprint and any "no impact on you" conclusion. This is the memory-pinned lesson (065/#155): a derived judgment over a possibly-incomplete list is *uncertain*, never a confident negative.

**Testing strategy**: BDD feature file(s) under the established godog suite exercising the declarative artifacts (the 064-established content-inspection pattern: whitespace-normalized copy for content assertions, raw for structure; helpers in production source, not `_test.go`), plus the drift guard. Validation scenarios from the spec (held out) will check: no invented surface, the single gated respond routed through the guardrail, reviews-inform-never-decide, no circulation/creation step, no authority verdict or coaching, synthesized-not-raw.

**Configuration**: None — all artifacts are committed static content; the registry file is the only machine-read input, consumed at build/test time.

---

## Implementation Strategy

Single phase, mirroring 067/068's landed shape (one implementation PR after the spec-artifact PR):

1. `plugin/agents/proposal-impact-review-commands.txt` — the registry (ten leaves, header documenting the one-in-nine-out posture).
2. `plugin/skills/proposal-impact-review/SKILL.md` — thin skill: when / workflow (id → delegate review → present picture → operator decides → gated respond or review-only exit) / delegation / boundaries (advance-withdraw → 068, drafting → 067, authority → 065; response value always operator-chosen).
3. `plugin/agents/proposal-impact-reviewer.md` — the reviewer: composed reads, synthesis instructions (impact picture, current-vs-proposed, footprint-honesty qualifier), the all-proposal-writes fence, failure honesty.
4. `internal/build/proposal_impact_review_guard_test.go` — the drift guard per ADR-5.
5. BDD coverage for the drift-guard-adjacent invariants per the family's established test shape.

No inter-phase dependencies; the artifacts reference each other, so they land together.

---

## Risks

- **Fence erosion by future edits** (low likelihood, high impact): the reviewer's all-writes fence and the skill's operator-chosen-value rule are prompt-level — a later edit could soften them. Mitigation: validation scenarios pin both; the drift guard pins the gate posture structurally.
- **`me roles` incompleteness silently dropped in synthesis** (medium likelihood — it is the subtlest instruction in the prompt; medium impact — a false "no impact on you"): mitigation: ADR-4a makes it a named plan rule, a prompt section, and a validation scenario, not a nice-to-have.
- **Divergent write locus confuses family maintenance** (low): 069 is the only path whose gated write runs in caller context. Mitigation: ADR-3 records the reasoning in DECISIONS.md; the skill and agent each state the locus explicitly.
- **Change-set → footprint matching is judgment, not mechanics** (medium/low): change elements reference governance by id/type; mapping them onto the operator's roles is the reviewer's synthesis judgment and cannot be drift-guarded. Mitigation: explicitly partial guard (stated), synthesized-not-raw validation scenario, and the fail-toward-surfacing default (when unsure whether a change touches the operator, show it).

---

## What This Plan Does Not Cover

- **Protocol-level artifact contracts** — the skill/agent frontmatter, section structure, registry line grammar, and guard-test assertions are the interface skill's concern (this is a specification boundary: the feature produces declarative artifacts with structural contracts).
- **Executable scenarios** — the scenarios skill concretizes the spec's driving scenarios into the feature-file suite.
- **Task decomposition** — the tasks skill splits the single phase into PR-sized units.
- **Distribution** — Operating-Surface Packaging (070) owns marketplace/publishing/install; nothing here adds distribution machinery.
- **Typed change-set builders / richer diffing** — the current-vs-proposed picture is drawn from the record by the reviewer's judgment; a mechanical governance-diff engine would be local governance logic (VISION Exclusion 2) and stays out.
