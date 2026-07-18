# Plan: Proposal Drafting Path

**Feature**: 067-proposal-drafting-path
**Role**: Shaper
**Inputs**: `specs/067-proposal-drafting-path/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (loaded — precedent from 062/063/064/065/066 and the proposal command family 055/056), `.score/memory/LEARNINGS.md` (passive — drift-guard single-sourcing, guard-artifact sync, BDD step-fidelity), `.score/memory/DEPRECATION.md` (no relevant entries for 067)

---

## System Architecture

Proposal Drafting Path is the fourth **operator path** on the Agent Operating Surface and the first to cross the Write-Safety Guardrail (063) into a *gated* governance write. Like its siblings (064/065/066) it adds no Go CLI code and no API capability — it is a pair of declarative plugin artifacts, hand-authored and committed under the existing top-level `plugin/` tree, plus one best-effort Go drift-guard test anchoring the artifacts to the shipped command surface *and* to 063's gated-command registry.

The path is delivered in **two cooperating pieces**, mirroring 064/065/066:

- **A thin skill** — `plugin/skills/proposal-drafting/SKILL.md`. The discoverable, description-triggered entry point. It carries the *when* (a well-formed tension is ready to become a governance change and a draft proposal should be created from it) and the *workflow* (anchor `ten_` id → read the tension back to ground the draft → situate against the proposals already in flight in the circle → assemble the change set → surface anchor + changes for confirmation → create the draft through the gated write → return the `prp_` id and hand it to the Proposal Circulation Path 068). It delegates execution to the proposal-drafter subagent.

- **A proposal-drafter subagent** — `plugin/agents/proposal-drafter.md`. A **new** agent (not a reuse of 066's `tension-processor` — ADR-2) that runs the workflow in its own isolated context and returns only the drawn-together draft record. Its single write is `glassfrog proposal create` — a write 063's `PreToolUse` hook gates behind human confirmation, and 066's implementation empirically confirmed that hook **fires inside subagent Bash calls**, so the confirmation reaches the practitioner even from isolation (ADR-3). The change set is passed **inline** in the create command so the confirmation prompt displays the exact payload being written. The tool grant withholds `Write`/`Edit`; the prompt fences execution to the composed leaves and forbids every other proposal write (`propose`/`respond`/`withdraw` — those are 068/069's territory, and 063 gates them regardless).

**Flow**: caller arrives with a ready tension's `ten_` id (066's handoff or a practitioner-identified tension) → skill triggers and delegates to the `proposal-drafter` subagent → subagent grounds the draft (`tension get <ten-id>`), situates (`proposal list --role-id … --status draft`, paged in full), helps assemble the `changes` JSON array (each element carrying a non-empty `type`, passed through verbatim — no typed builders) → subagent surfaces the anchor + assembled change set, then runs `glassfrog proposal create <ten-id> --changes '<inline JSON>'` → 063's hook interposes the human confirmation → on success the subagent returns the synthesized draft record carrying the `prp_` id and `draft` status, ready to hand to 068.

Everything the two artifacts name is behaviour the CLI already exposes. The path extends `plugin/` additively — a sibling skill and a sibling agent — exactly as 062 ADR-2 anticipated, no restructure.

---

## Architecture Decisions

### ADR-1: Deliver the path as a thin skill delegating to a drafting subagent (both-form)

**Context**: The spec deferred the delivery form to shaping (spec Assumption 1). 064 established the operator-path pattern as a thin skill + isolated subagent, and 065/066 followed it; 067 shares the same "synthesized record, not raw dumps" promise, plus a stronger reason for isolation: the situating reads (a full-walk proposal list) and the tension read stay out of the caller's context, and the draft record comes back drawn together.

**Options considered**:
1. **Skill only** — execution happens in the caller's context; "synthesized not raw" rests on caller discipline, and the situating walk floods the caller.
2. **Subagent only** — isolation without a discoverable, description-triggered entry point; the *when* + workflow knowledge has no home.
3. **Both — thin skill + subagent** — the skill owns *when* and the single-sourced workflow and delegates; the subagent owns isolated execution and returns the synthesized draft record.

**Decision**: Option 3 — both. Silent conformance to 064 ADR-1 / 065 ADR-1 / 066 ADR-1 (the established operator-path pattern; 064 already carried the divergence from 062 ADR-2's skills-only projection). The workflow is written **once** in the skill; the agent references it rather than restating a divergent copy.

**Consequences**: Discoverability + context isolation; two artifacts to keep coherent (mitigated by single-sourcing the workflow in the skill, as 066 does).

### ADR-2: Mint a new `proposal-drafter` agent — do not reuse or extend 066's `tension-processor`

**Context**: 066's plan explicitly deferred to this spec whether 067 reuses the `tension-processor` subagent ("an agent per skill would be overkill" was noted but left undecided). The shipped `tension-processor` prompt **forbids any `proposal …` command** — that fence is a validate-pinned 066 invariant and one layer of its "operational writes only" guarantee. The developer asked for a recommendation; the Shaper recommended a new agent and the developer accepted.

**Options considered**:
1. **New `proposal-drafter` agent** — each path's prompt fence stays exact: the drafter's one write is `proposal create`, never propose/respond/withdraw; the processor's fence (no `proposal …` at all) is untouched. One more declarative file.
2. **Reuse/extend `tension-processor`** — a single shared write executor, but it requires deleting the processor's no-proposal fence — eroding 066's shipped, validated safety property and forcing 066's re-validation — and a union fence ("tension writes OR proposal create") blurs both paths' guarantees.

**Decision**: Option 1 — a new agent at `plugin/agents/proposal-drafter.md`, sibling skill at `plugin/skills/proposal-drafting/SKILL.md`. Declarative agent files are cheap; eroded fences are not. Silent conformance to the one-agent-per-path pattern (064 navigator, 065 constraint-navigator, 066 processor) and to 062 ADR-1/ADR-2 + 064 ADR-2 for the additive homes (directory auto-discovery; no `plugin.json` `agents` key).

**Consequences**: 066's artifacts and validation stand untouched. The `plugin/agents/` type gains a third consumer. 068 faces the same question for circulation and may revisit; this ADR decides only 067, it does not pre-commit 068. Negative: a fourth agent file whose workflow prose must be kept coherent with its skill (same single-sourcing mitigation as the siblings).

### ADR-3: The gated create runs inside the subagent, with the change set inline — 063's hook is the structural confirmation layer

**Context**: The spec requires the create to run through 063's confirmed write flow, with the assembled change set surfaced before the write. The architectural question is *where* the gated write executes: inside the isolated subagent, or handed back to the caller. 066's implementation confirmed empirically that Claude Code `PreToolUse` hooks fire inside subagent calls (hook input carries `agent_id`), so 063's Bash gate reaches a subagent's `proposal create`. The developer confirmed the in-subagent option.

**Options considered**:
1. **Create inside the subagent, `--changes` inline** — isolation preserved end-to-end; 063's hook interposes the human confirmation at execution time; passing the change set inline in the command line means the confirmation prompt displays the exact payload being written (a file path or `stdin` keyword would hide it).
2. **Agent assembles only; caller executes the create** — keeps the agent read-only, but pushes the write back into the caller's context, loses the isolation benefit, and splits the workflow across two contexts for no safety gain now that hook coverage is confirmed.
3. **Create inside the subagent, `--changes` from a file** — same locus, but the confirmation prompt shows a path, not the payload; the human confirms blind.

**Decision**: Option 1. The subagent surfaces the anchor tension and assembled change set in its narration *before* the write (prompt-level presentation), then runs `glassfrog proposal create <ten-id> --changes '<inline JSON array>'`; 063's hook returns `permissionDecision:"ask"` and the practitioner confirms seeing the exact command — payload included. Confirmation is therefore **layered**: presentation (prompt-level) + the 063 gate (structural, fires regardless of what the agent narrates). The tool grant withholds `Write`/`Edit` (also keeping the agent from writing change-set temp files — inline is the only honest source); the prompt fences the write surface to `proposal create` alone and forbids `proposal propose`/`respond`/`withdraw` (068/069's writes, which 063 gates as backstop).

**Consequences**: This is an **announced divergence** from 066 ADR-3's write posture: 066's writes are deliberately *ungated* (operational tension edits), whereas 067's single write is deliberately *gated* — "only the ungated tension writes" (066) becomes "only the one gated create, always through the gate" (067). Positive — the human sees exactly what will be written, once, at the moment it matters; a rejected confirmation means no proposal is created (spec edge case). Negative — inline JSON meets shell quoting: the change set must be valid single-quoted shell payload; the skill/agent point at the orientation (062) quoting guidance, and a change set too unwieldy for the command line is surfaced to the caller rather than smuggled through a hidden file.

### ADR-4: Compose only already-shipped commands; add no CLI code beyond a drift guard

**Context**: "Knowledge + guardrails, never capability" (PROJECT) and the spec's non-behaviors forbid any new command, flag, or local governance/validation logic. Every command this path composes is shipped: Proposal Creation (055, `proposal create`), Proposal Reads (056, `proposal list` / `proposal get`), Tension Reads (043, `tension get`).

**Options considered**:
1. **Pure composition** — the artifacts drive existing `glassfrog` commands only; the CLI is unchanged.
2. **A convenience `proposal draft` command in the CLI** — rejected on sight: new API-adjacent surface, breaking "Bounded by the API surface" / VISION Exclusion 2, duplicating composition logic the operator layer owns.

**Decision**: Option 1 — pure composition; the only Go change is a test (ADR-5). Silent conformance to 062/064/065/066's ADR pattern; per-command flags are never restated — the artifacts defer to `glassfrog proposal <sub> --help` / `glassfrog tension get --help`. The change-set floor (valid JSON array, non-empty `type` per element) is *described* as what the CLI enforces (055's floor), never re-implemented or extended — no typed per-change builders (deferred *Unguided Change Construction*).

**Consequences**: The drift surface is the set of command leaves the artifacts name, plus one invariant those leaves must satisfy (ADR-5). Situating narrows by circle + `draft` status because that is what `proposal list` exposes — the artifacts must not imply a tension-id filter (spec Assumption 5).

### ADR-5: Drift guard pins the composed leaves and the *gated-membership* invariant — the inverse of 066's disjointness assertion

**Context**: 062–066 each anchor plugin artifacts to the CLI with a best-effort `internal/build` config-guard test, single-sourced (063's `gated-commands.txt`; 066's `tension-processing-commands.txt` + disjointness check). 067's correctness depends on the **opposite** invariant from 066: its write leaf (`proposal create`) must be **a member of** 063's gated set — if it ever left that registry, the "confirmed write flow" promise would silently become an unconfirmed write. Its composed reads (`proposal list`, `proposal get`, `tension get`) must stay **out** of the gated set, or situating would start prompting.

**Options considered**:
1. **Best-effort guard: leaf existence + gate-membership both ways** — assert every composed leaf exists in the CLI's registry; assert `proposal create` ∈ 063's `gated-commands.txt`; assert the composed reads ∉ the gated set. Single-source the composed-leaf list in one committed file consumed by the artifacts' reference and the guard.
2. **Existence-only guard** — cheaper, but leaves the gated invariant (the crux of ADR-3) unguarded — exactly the silent-drift failure this surface is prone to.
3. **No guard** — rejected; drift is the failure mode every sibling guarded.

**Decision**: Option 1 — a best-effort `internal/build` test, with the composed-leaf list single-sourced (066's format: two-token leaves, matching 063's registry format so membership checks compare like with like), asserting existence, gated-membership of the write leaf, and non-membership of the reads. Both sides source-derived, never hard-coded (the guard derives the gated set from `gated-commands.txt` and the live surface from the command registry — a hard-coded copy would be a second source of truth).

Coverage is explicitly partial: existence + gate-membership, not flags (deferred to `--help`), not the synthesized-record prose, not parser robustness. Stated in the test, not silent.

**Consequences**: A renamed/removed composed command fails the build; a change that removes `proposal create` from the gated registry (or gates a composed read) fails the build — the confirmed-write-flow promise is pinned structurally. Together with 066, the pattern generalizes: every operator write-path's guard asserts its leaves' *gate-membership posture*, whichever side of the gate they belong on.

---

## Cross-cutting Concerns

**Failure handling**: The subagent drafts defensively. A create rejected by the server — the `403` Premium refusal, an unknown anchor (`404`), a rejected change set (`422`) — is surfaced by name with nothing created and no fabricated `prp_` id (spec error scenario). A situating list that fails mid-walk yields the proposals gathered so far, flagged incomplete with the cause (056 already guarantees the flag; the agent must relay it, not paper over it). A declined confirmation at the 063 gate means no proposal is created — the agent reports the decline as an outcome, not an error. Exit-code, pagination, and output mechanics are deferred to the orientation skill (062), never restated.

**Confirmation layering**: Presentation (the agent narrates anchor + change set before the write) and structure (063's hook interposes at execution, showing the exact inline command) are independent layers; the structural layer holds even if the narration is wrong. The inline `--changes` decision (ADR-3) is what makes the structural layer *informative* rather than a blind "run this?".

**No new observability/config/state**: The path composes commands. Its only artifacts are the two prose files, the single-source leaf list, and the guard test.

**Testing strategy**: (1) the ADR-5 drift guard (existence + gate-membership both ways); (2) BDD scenarios (Tier 2) exercising the spec's driving/validation scenarios against the artifacts' described behaviour — tension→draft, situating with full paging, file-sourced-assembly guidance, create-rejected, partial situating failure, confirmation-before-crossing, duplicate-in-flight, handoff-to-068, no-invented-surface, assembly-not-typed-construction, drafting-only, not-judging-or-coaching, synthesized-not-raw. As with 064/065/066, the artifacts are declarative prose, so scenario coverage verifies described behaviour and the guard verifies the named surface and its gate posture.

---

## Implementation Strategy

Single phase — one coherent unit, mirroring 065/066:

1. **Author the skill and agent, add the drift guard.** Write `plugin/skills/proposal-drafting/SKILL.md` (when + single-sourced workflow + delegation + the gated-write note), write `plugin/agents/proposal-drafter.md` (write-capable-but-fenced grant: `Bash` in, `Write`/`Edit` out; workflow by reference; composed leaves; draft-record output contract), single-source the composed-leaf list, and add the `internal/build` guard (existence + gated-membership of `proposal create` + non-membership of the reads). The agent is discovered by directory convention from `plugin/agents/` — no `plugin.json` edit. Siblings (`orientation`, `governance-navigation`, `constraint-discovery`, `tension-processing`, `plugin/hooks/`) stay untouched.

The tasks skill may split this into the sibling-standard two PR-sized units (artifacts first, guard second); the only ordering constraint is that the guard pins the leaves the artifacts name.

---

## Risks

- **Skill/agent workflow drift** (medium likelihood, low impact): two artifacts, one workflow. *Mitigation*: single-source the steps in the skill; the agent references them (ADR-1); review covers what the guard cannot.
- **Inline change set meets shell quoting** (medium likelihood, medium impact): governance change sets are JSON with quotes and apostrophes; a mangled inline payload either fails 055's parse floor (safe — usage error, no request) or, worse, parses to something other than what was narrated. *Mitigation*: the artifacts point at 062's quoting guidance; the 063 confirmation shows the exact payload as the last line of defense; a change set too unwieldy for the command line is surfaced to the caller rather than routed through a hidden file (ADR-3).
- **Gate-registry drift** (low likelihood, high impact): if `proposal create` ever left 063's gated set, this path's writes would silently stop being confirmed. *Mitigation*: ADR-5's membership assertion turns that into a build failure.
- **Boundary bleed into circulation (068) or authority judgment (065)** (medium likelihood, low impact): after a successful create it is tempting to advance the draft; before it, to rule on whether a proposal is needed. *Mitigation*: prompt fence forbids `proposal propose/respond/withdraw` and authority verdicts; 063 gates the circulation writes regardless; validation scenarios assert absence.
- **Duplicate check is circle-scoped, not tension-scoped** (low likelihood, low impact): `proposal list` offers no tension filter, so situating can miss a related draft filed in another circle. *Mitigation*: accepted and stated — the artifacts describe the check as circle + `draft` status (spec Assumption 5); claiming more would invent surface.
- **Host doesn't honour subagents/tool grants/hooks** (low likelihood, medium impact): the isolation, the fenced grant, and the in-subagent gate all assume a capable host; 066 confirmed hook-in-subagent coverage for Claude Code specifically. *Mitigation*: the skill degrades to guidance (as 064/065/066 document); #70 owns host targeting.

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the skill/agent frontmatter schemas, the exact fenced tool-grant syntax, the single-source leaf-list file format, the draft-record output shape, and the wording of the confirmation narration. These are the **interface** skill's concern (`interface-spec.md`).
- **Executable scenarios** — the Gherkin realizing the spec's driving/validation scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units and acceptance criteria are the **tasks** skill's concern.
- **Distribution** — publishing/installing the plugin (now four skills + three agents) remains Operating-Surface Packaging (#70).
- **068's agent question** — whether Proposal Circulation mints its own agent or shares this one is 068's decision; ADR-2 decides only 067 and does not pre-generalize `proposal-drafter` into a shared write executor.
- **Typed change construction** — per-`type` builders/validation remain the deferred *Unguided Change Construction* capability; this path assembles and passes through verbatim above 055's floor.
