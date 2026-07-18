# Plan: Proposal Circulation Path

**Feature**: 068-proposal-circulation-path
**Role**: Shaper
**Inputs**: `specs/068-proposal-circulation-path/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (loaded — precedent from the operator-path family 062/063/064/065/066/067 and the proposal command family 056/057/059), `.score/memory/LEARNINGS.md` (passive — drift-guard single-sourcing, gate-membership anchoring, guard-artifact sync), `.score/memory/DEPRECATION.md` (no relevant entries for 068)

---

## System Architecture

Proposal Circulation Path is the fifth **operator path** on the Agent Operating Surface and the second to cross the Write-Safety Guardrail (063) — this time **twice**: both of its writes, the `propose` and `withdraw` transitions, sit in 063's gated registry. Like its siblings (064/065/066/067) it adds no Go CLI code and no API capability — it is a pair of declarative plugin artifacts, hand-authored and committed under the existing top-level `plugin/` tree, plus one best-effort Go drift-guard test anchoring the artifacts to the shipped command surface *and* to 063's gated-command registry.

The path is delivered in **two cooperating pieces**, mirroring 064/065/066/067:

- **A thin skill** — `plugin/skills/proposal-circulation/SKILL.md`. The discoverable, description-triggered entry point. It carries the *when* (a draft proposal is ready to circulate, a circulating proposal needs monitoring, or a circulating proposal needs pulling back for amendment) and the *workflow* (anchor `prp_` id → read the proposal back to ground the act → depending on intent: advance the draft through the gated `propose`, monitor the circulation picture, or withdraw through the gated `withdraw` → hand off: a withdrawn draft's `prp_` id back to the Proposal Drafting Path (067) for re-editing; consent responses to the response side (069)). It delegates execution to the proposal-circulator subagent.

- **A proposal-circulator subagent** — `plugin/agents/proposal-circulator.md`. A **new** agent (not a reuse of 067's `proposal-drafter` — ADR-2, developer-confirmed) that runs the workflow in its own isolated context and returns only the drawn-together circulation picture. Its writes are exactly `glassfrog proposal propose <prp-id>` and `glassfrog proposal withdraw <prp-id>` — both leaves 063's `PreToolUse` hook gates behind human confirmation, and the hook fires inside subagent Bash calls (confirmed empirically in 066, relied on in 067), so each confirmation reaches the practitioner even from isolation. Both transitions are **bodyless** — the command line *is* the full payload (`proposal propose prp_…`), so the 063 confirmation prompt is inherently complete; 067's inline-payload problem does not arise (ADR-3). The tool grant withholds `Write`/`Edit`; the prompt fences execution to the composed leaves and forbids every other proposal write (`create` is 067's, `respond` is 069's — and 063 gates those regardless).

**Flow**: caller arrives with a `prp_` id (067's draft handoff, or a practitioner-identified proposal in either state) and an intent → skill triggers and delegates to the `proposal-circulator` subagent → subagent grounds the act (`proposal get <prp-id>`: status, `response_summary`, `response_deadline`, `available_transitions`) and, where relevant, situates against the circle's in-flight proposals (`proposal list`, paged in full) → for an advance or withdraw, the subagent narrates the proposal and the intended transition, then runs the bodyless gated command → 063's hook interposes the human confirmation → on success the subagent returns the server's proposal exactly as returned (advanced: `proposed_outside_meeting` with `response_deadline` and the implicit `no_objection`; withdrawn: back in `draft` with timestamps cleared), synthesized into the circulation picture carrying the `prp_` id. The reads **inform, never gate**: the agent surfaces `available_transitions` so the proposer knows what the server currently offers, but it never turns that snapshot into a client-side precondition — it issues the transition and surfaces the server's `422` plainly (the 057/059 no-pre-gate discipline, held at prompt level).

Everything the two artifacts name is behaviour the CLI already exposes. The path extends `plugin/` additively — a sibling skill and a sibling agent — exactly as 062 ADR-2 anticipated, no restructure.

---

## Architecture Decisions

### ADR-1: Deliver the path as a thin skill delegating to a circulation subagent (both-form)

**Context**: The spec deferred the delivery form to shaping (spec Assumption 1). 064 established the operator-path pattern as a thin skill + isolated subagent; 065, 066, and 067 followed it. 068 shares the same "synthesized picture, not raw dumps" promise and the same isolation benefit: the monitoring reads (a `proposal get` plus a possible full-walk `proposal list`) stay out of the caller's context, and the circulation picture comes back drawn together.

**Options considered**:
1. **Skill only** — execution happens in the caller's context; "synthesized not raw" rests on caller discipline, and the monitoring walk floods the caller.
2. **Subagent only** — isolation without a discoverable, description-triggered entry point; the *when* + workflow knowledge has no home.
3. **Both — thin skill + subagent** — the skill owns *when* and the single-sourced workflow and delegates; the subagent owns isolated execution and returns the synthesized circulation picture.

**Decision**: Option 3 — both. Silent conformance to 064 ADR-1 / 065 ADR-1 / 066 ADR-1 / 067 ADR-1 (the established operator-path pattern). The workflow is written **once** in the skill; the agent references it rather than restating a divergent copy. No 065-style interaction branch is needed in the skill: the path's inputs (a `prp_` id and an intent) are concrete on arrival, and the one human interaction that matters — confirming each gated transition — is structural (063's hook), not conversational.

**Consequences**: Discoverability + context isolation; two artifacts to keep coherent (mitigated by single-sourcing the workflow in the skill, as the siblings do).

### ADR-2: Mint a new `proposal-circulator` agent — do not reuse or extend 067's `proposal-drafter`

**Context**: 066 and 067 each explicitly left the agent-reuse question to the next path; 067 ADR-2 states "068 faces the same question for circulation and may revisit; this ADR decides only 067." The shipped `proposal-drafter` prompt **forbids `proposal propose`/`respond`/`withdraw`** — that fence is a validate-pinned 067 invariant and one layer of its "only the one gated create" guarantee. The developer asked the question be surfaced; the Shaper recommended a new agent and the developer confirmed.

**Options considered**:
1. **New `proposal-circulator` agent** — each path's prompt fence stays exact: the circulator's writes are `proposal propose` + `proposal withdraw`, never `create`/`respond`; the drafter's fence (no propose/respond/withdraw) is untouched. One more declarative file.
2. **Reuse/extend `proposal-drafter`** — a single proposal-write executor, but it requires deleting the drafter's fence — eroding 067's shipped, validated safety property and forcing 067's re-validation — and a union fence ("create OR propose OR withdraw") blurs both paths' guarantees and pre-builds the shared write executor 066/067 each declined to pre-generalize.

**Decision**: Option 1 — a new agent at `plugin/agents/proposal-circulator.md`, sibling skill at `plugin/skills/proposal-circulation/SKILL.md`. Declarative agent files are cheap; eroded fences are not — the same reasoning that resolved the 066→067 reuse question. Silent conformance to the one-agent-per-path pattern (064 navigator, 065 constraint-navigator, 066 processor, 067 drafter) and to 062 ADR-1/ADR-2 + 064 ADR-2 for the additive homes (directory auto-discovery; no `plugin.json` `agents` key).

**Consequences**: 067's artifacts and validation stand untouched. The `plugin/agents/` type gains a fourth consumer. 069 faces the same question for the response side and may revisit; this ADR decides only 068. Negative: a fifth agent file whose workflow prose must be kept coherent with its skill (same single-sourcing mitigation as the siblings).

### ADR-3: Both gated transitions run inside the subagent as bodyless commands — 063's hook is the structural confirmation layer, and the command line is inherently the full payload

**Context**: The spec requires both `propose` and `withdraw` to run through 063's confirmed write flow, with the proposal surfaced before each write. 067 ADR-3 settled the *where* (inside the subagent — hooks fire in subagent Bash calls) and fought a real battle over the *what* (inline `--changes` so the confirmation shows the payload). 068's transitions carry **no body** — the entire write is `glassfrog proposal propose <prp-id>` or `glassfrog proposal withdraw <prp-id>`.

**Options considered**:
1. **Transitions inside the subagent** — isolation preserved end-to-end; 063's hook interposes at execution; the confirmation prompt shows the complete command, which for a bodyless transition *is* the complete payload.
2. **Agent monitors only; caller executes the transitions** — keeps the agent write-free, but pushes the writes back into the caller's context, splits the workflow across two contexts, and gains nothing: hook coverage inside subagents is confirmed, and there is no payload-visibility concern to trade against.

**Decision**: Option 1. Silent conformance to 067 ADR-3's write locus; the payload-source dimension that made 067's decision hard is absent here. Confirmation is **layered**, as in 067: the subagent narrates the proposal (id, status, what the transition will do) *before* the write (prompt-level presentation), and 063's hook interposes structurally at execution — the practitioner sees `glassfrog proposal propose prp_…` and that line omits nothing. A declined confirmation means no transition happens (spec edge case); the agent reports the decline as an outcome, not an error. Two writes in one path also means **two independent confirmations** when a session both advances and later withdraws — each transition is its own gated act; the path never batches or pre-authorizes them.

**Consequences**: Positive — the human confirms exactly what will run, per transition, with nothing hidden; no shell-quoting risk (067's inline-JSON concern does not arise). Negative — none beyond the friction inherent to two gated acts, which is the spec's explicit design ("exactly two gated governance writes… each always through the guardrail's confirmed flow").

### ADR-4: Compose only already-shipped commands; reads inform, never gate — held at prompt level

**Context**: "Knowledge + guardrails, never capability" (PROJECT) and the spec's non-behaviors forbid any new command, flag, or local transition-authorization logic. Every command this path composes is shipped: Advance to Circulation (057, `proposal propose`), Withdraw Proposal (059, `proposal withdraw`), Proposal Reads (056, `proposal get` / `proposal list`). 057 and 059 each carry a validate-pinned invariant — *no client-side pre-gating on `available_transitions`* — and 068 is the first operator path that reads that field while also invoking the transitions it advertises, so the temptation to pre-gate lives here.

**Options considered**:
1. **Pure composition, with the no-pre-gate discipline written into the prompt fence** — the agent reads to narrate, issues the transition regardless of what the snapshot said, and surfaces the server's `422` plainly.
2. **A convenience `proposal circulate` command in the CLI** — rejected on sight: new API-adjacent surface, breaking "Bounded by the API surface" / VISION Exclusion 2.
3. **Composition with a client-side transition check** ("only issue `propose` if the read showed it available") — rejected: forks the server's authority onto a stale snapshot, exactly what 057/059's non-behaviors forbid.

**Decision**: Option 1 — pure composition; the only Go change is a test (ADR-5). Silent conformance to 062/064/065/066/067's composition ADRs; per-command flags are never restated — the artifacts defer to `glassfrog proposal <sub> --help`. The **reads-inform-never-gate** rule is a prompt-level guarantee (like 064's surface-not-judge and 065's no-local-verdict): it cannot be tool-enforced, because reading the proposal is legitimate and required; it is verified by a held-out validation scenario. The monitoring picture surfaces `response_summary` / `response_deadline` / `available_transitions` exactly as the server returned them — the agent computes no acceptance and narrates no side effects the record doesn't show.

**Consequences**: The drift surface is the set of command leaves the artifacts name, plus their gate-membership posture (ADR-5). Monitoring situates by circle + status when the in-flight picture is relevant, because that is what `proposal list` exposes (no tension filter — same acknowledged limit as 067).

### ADR-5: Drift guard pins the composed leaves and *both* writes' gated-membership — extending 067's gate-posture pattern to a two-write path

**Context**: 062–067 each anchor plugin artifacts to the CLI with a best-effort `internal/build` config-guard test, single-sourced. 067 generalized the pattern: *every operator write-path's guard asserts its leaves' gate-membership posture, whichever side of the gate they belong on.* 068's correctness depends on that posture twice over: its write leaves (`proposal propose`, `proposal withdraw`) must each be **members of** 063's gated set — if either ever left the registry, the "confirmed write flow" promise for that transition would silently become an unconfirmed write. Its composed reads (`proposal get`, `proposal list`) must stay **out** of the gated set, or monitoring would start prompting.

**Options considered**:
1. **Best-effort guard: leaf existence + gate-membership both ways** — assert every composed leaf exists in the CLI's registry; assert `proposal propose` ∈ and `proposal withdraw` ∈ 063's `gated-commands.txt`; assert the composed reads ∉ the gated set. Single-source the composed-leaf list in one committed file consumed by the artifacts' reference and the guard.
2. **Existence-only guard** — cheaper, but leaves the two gated invariants (the crux of ADR-3) unguarded.
3. **No guard** — rejected; drift is the failure mode every sibling guarded.

**Decision**: Option 1 — a best-effort `internal/build` test, with the composed-leaf list single-sourced (the sibling format: two-token leaves, matching 063's registry format so membership checks compare like with like), asserting existence, gated-membership of **both** write leaves, and non-membership of the reads. Both sides source-derived, never hard-coded (the guard derives the gated set from `gated-commands.txt` and the live surface from the command registry). Per the 067-vintage boundary nuance: the *sets* stay derived, but which specific leaves must sit on which side of the gate is this path's checked-in contract-fact — the guard names `proposal propose` and `proposal withdraw` as the required gated members and `proposal get`/`proposal list` as the required ungated reads, cross-checked against source, so a swap (a read wandering into the gated set, a write wandering out) cannot satisfy the guard by count alone.

Coverage is explicitly partial: existence + gate-membership, not flags (deferred to `--help`), not the synthesized-picture prose, not the no-pre-gate discipline (prompt-level, validation-scenario-verified). Stated in the test, not silent.

**Consequences**: A renamed/removed composed command fails the build; a change that removes either transition from the gated registry (or gates a composed read) fails the build — both confirmed-write-flow promises are pinned structurally. The gate-posture pattern now has consumers on three postures: all-out (066), one-in (067), two-in-two-out (068).

---

## Cross-cutting Concerns

**Failure handling**: The subagent circulates defensively. A transition rejected by the server — the `403` Premium refusal, an unknown proposal (`404`), a transition not currently allowed (`422`) — is surfaced by name with nothing transitioned and no fabricated state (spec error scenario); a `422` on `propose` or `withdraw` is a real refusal, never absorbed (the 057/059 invariant, relayed not re-derived). A monitoring read that fails mid-walk yields what was gathered so far, flagged incomplete with the cause (056 already guarantees the flag; the agent relays it). A declined confirmation at the 063 gate means no transition — reported as an outcome, not an error. Exit-code, pagination, and output mechanics are deferred to the orientation skill (062), never restated.

**Confirmation layering**: Presentation (the agent narrates the proposal and intended transition before each write) and structure (063's hook interposes at execution, showing the complete bodyless command) are independent layers; the structural layer holds even if the narration is wrong. Each of the two transitions confirms independently — no batching, no pre-authorization.

**No new observability/config/state**: The path composes commands. Its only artifacts are the two prose files, the single-source leaf list, and the guard test.

**Testing strategy**: (1) the ADR-5 drift guard (existence + gate-membership both ways, two writes in, two reads out); (2) BDD scenarios (Tier 2) exercising the spec's driving/validation scenarios against the artifacts' described behaviour — advance-with-confirmation, monitor-drawn-together, withdraw-and-hand-back-to-067, transition-rejected, partial monitoring failure, confirmation-before-crossing, reads-inform-never-gate, response-belongs-to-069, no-invented-surface, circulation-only, not-judging-or-coaching, synthesized-not-raw. As with the siblings, the artifacts are declarative prose, so scenario coverage verifies described behaviour and the guard verifies the named surface and its gate posture.

---

## Implementation Strategy

Single phase — one coherent unit, mirroring 065/066/067:

1. **Author the skill and agent, add the drift guard.** Write `plugin/skills/proposal-circulation/SKILL.md` (when + single-sourced workflow + delegation + the two-gated-writes note), write `plugin/agents/proposal-circulator.md` (write-capable-but-fenced grant: `Bash` in, `Write`/`Edit` out; workflow by reference; composed leaves; circulation-picture output contract), single-source the composed-leaf list, and add the `internal/build` guard (existence + gated-membership of `proposal propose` and `proposal withdraw` + non-membership of `proposal get`/`proposal list`). The agent is discovered by directory convention from `plugin/agents/` — no `plugin.json` edit. Siblings (`orientation`, `governance-navigation`, `constraint-discovery`, `tension-processing`, `proposal-drafting`, `plugin/hooks/`) stay untouched.

The tasks skill may split this into the sibling-standard two PR-sized units (artifacts first, guard second); the only ordering constraint is that the guard pins the leaves the artifacts name.

---

## Risks

- **Skill/agent workflow drift** (medium likelihood, low impact): two artifacts, one workflow. *Mitigation*: single-source the steps in the skill; the agent references them (ADR-1); review covers what the guard cannot.
- **Gate-registry drift** (low likelihood, high impact): if `proposal propose` or `proposal withdraw` ever left 063's gated set, that transition would silently stop being confirmed. *Mitigation*: ADR-5's per-leaf membership assertions turn either drift into a build failure.
- **Pre-gating creep** (medium likelihood, medium impact): the agent reads `available_transitions` and then invokes transitions — the natural drift is "only offer `propose` when the read showed it," quietly forking the server's authority onto a stale snapshot. *Mitigation*: the reads-inform-never-gate rule is an explicit prompt fence (ADR-4), and a held-out validation scenario inspects for a client-side gate; the server's `422` remains the surfaced authority.
- **Boundary bleed into drafting (067) or responses (069)** (medium likelihood, low impact): after a withdraw it is tempting to re-edit the change set; while monitoring, to record a `no_objection`. *Mitigation*: prompt fence forbids `proposal create`/`respond`; 063 gates them regardless; validation scenarios assert absence; the handoffs (back to 067, out to 069) are named workflow steps.
- **Two-write confirmation fatigue** (low likelihood, low impact): a session that advances, amends, and re-advances crosses the gate repeatedly, and a fatigued practitioner may rubber-stamp. *Mitigation*: accepted — per-transition confirmation is the spec's explicit design; the narration layer keeps each prompt informative rather than rote.
- **Host doesn't honour subagents/tool grants/hooks** (low likelihood, medium impact): the isolation, the fenced grant, and the in-subagent gates all assume a capable host; 066 confirmed hook-in-subagent coverage for Claude Code specifically. *Mitigation*: the skill degrades to guidance (as the siblings document); #70 owns host targeting.

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the skill/agent frontmatter schemas, the exact fenced tool-grant syntax, the single-source leaf-list file format, the circulation-picture output shape, and the wording of the confirmation narration. These are the **interface** skill's concern (`interface-spec.md`).
- **Executable scenarios** — the Gherkin realizing the spec's driving/validation scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units and acceptance criteria are the **tasks** skill's concern.
- **Distribution** — publishing/installing the plugin (now five skills + four agents) remains Operating-Surface Packaging (#70).
- **069's agent question** — whether the response side mints its own agent or shares one is 069's decision; ADR-2 decides only 068 and does not pre-generalize `proposal-circulator` into a shared transition executor.
- **Plan-limit signalling** — the `403` a transition can meet renders through the CLI's own 061 plan-gate diagnostic; the path relays it and adds no plan-aware interpretation of its own.
