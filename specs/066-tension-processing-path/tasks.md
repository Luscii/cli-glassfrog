# Tasks: Tension Processing Path

**Feature**: 066-tension-processing-path
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/tension-processing-path.feature

---

## Dependency Graph

Phase 1: Tension-processing path artifacts + drift guard (2 tasks, no phase dependencies; intra-phase: T002 depends on T001) [Shared]

2 tasks total | 0 phases parallelizable | Builder: pipeline (single active spec)

The plan is a single phase (ADR-1/2/3 land the two artifacts together; ADR-5's guard pins the leaves they name and the ungated invariant). The two PR-sized units follow the plan's own suggested split: artifacts + registration first, drift guard second.

---

## Branching Guidance

**Pipeline mode**: `spec/066-tension-processing-path/base` → `spec/066-tension-processing-path/task-1`, `spec/066-tension-processing-path/task-2`

T002 depends on T001 (the guard pins the command leaves the artifacts name and asserts their disjointness from 063's gated set), so the two task branches land in sequence onto the spec base.

---

## Phase 1: Tension-processing path artifacts + drift guard [Shared]

- [ ] **T001** [Shared] Author the tension-processing skill and the write-capable tension-processor agent, single-source the composed-leaf list, and register the agent — 14 scenarios un-@wip'd (capture, refine, retire, situate, duplicate, partial-failure, capture-rejected, handoff, synthesis, and the operational-writes-only / guardrail-boundary / processing-not-judging boundaries, plus registration/degradation) — skill + agent + single-sourced leaf-list added under plugin/; BDD suite green (14 scenarios); agent auto-discovered from plugin/agents/ (no plugin.json `agents` key), orientation skill / navigator agent / hooks untouched, no marketplace.json
  - **Scope**: Add `plugin/skills/tension-processing/SKILL.md` (the thin, discoverable entry point: when to reach for it, the single-sourced workflow, the delegation-to-the-agent instruction, and the write-boundary note) and `plugin/agents/tension-processor.md` (the executor: identity/scope, the workflow it runs — *referencing* the skill's workflow, not restating a divergent copy — the composed tension leaves, and the tension-record output contract). Single-source the composed-leaf list in one committed file. The agent is discovered by directory convention from `plugin/agents/` — **do not** add an `agents` key to `plugin/.claude-plugin/plugin.json` (063's landed `plugin/hooks/hooks.json` is auto-discovered with no `hooks` array, and 064's navigator confirmed the same; the manifest stays unchanged). Hand-authored, committed content; adds no CLI code.
  - **Acceptance criteria**:
    - `plugin/skills/tension-processing/SKILL.md` exists with YAML frontmatter `name` (`tension-processing`) + `description`; the `description` states *when* to reach for it (a practitioner has a tension to act on and wants it recorded/refined/retired on the right role) and that it returns a well-formed tension record carrying its id, worded not to fire on "understand the governance around a concern" (064), "am I allowed to do X" (065), or "draft/circulate a proposal" (067/068)
    - The skill body carries the when / workflow (voiced tension → situate via `tension list <role-id>` + `tension subroles <role-id>` → capture via `tension create <role-id>` → refine via `tension update <ten-id>` or retire via `tension discard <ten-id>` → hand the `ten_` id to 067) / delegation / write-boundary note, pointing at `glassfrog tension <sub> --help` for per-command detail and at the orientation skill (062) for output/pagination/exit-code mechanics rather than restating them
    - The workflow states that where a situating list read spans multiple pages, the processor pages through the full result set (per the orientation pagination guidance) **before** judging duplicates — a duplicate check over the complete set, never a silent single-page cap (Constitution VI)
    - The agent is registered so the host discovers it and the skill's delegation resolves; if the agent is absent/unregistered, the skill's workflow remains usable as guidance with no CLI command broken (documented degradation)
    - Subagent hook coverage is confirmed against the target host: verify whether 063's landed `PreToolUse` `Bash` gate (`plugin/hooks/glassfrog-write-gate.sh`) also fires for the tension-processor subagent's Bash calls; regardless of the answer, the agent prompt is strictly scoped to the six tension leaves and forbids any `proposal …` command (the load-bearing proposal fence if the hook does not reach the subagent — see risk.md; matters more than 064 because this subagent legitimately writes)
    - `plugin/agents/tension-processor.md` exists with frontmatter `name` (`tension-processor`), `description`, and a **write-capable-but-fenced `tools` grant** that includes `Bash` (to invoke `glassfrog tension` reads and writes) and excludes `Write`/`Edit` (no workspace mutation); the body references the skill's workflow, lists only the composed tension leaves, and defines the tension-record output (tension/situating/action/handoff/notes, each carrying its id)
    - The artifacts perform only operational tension writes and name no `proposal …` command; they hand the ready `ten_` id to 067 and any authority question to 065; the workflow steps are single-sourced in the skill and referenced by the agent — no second, divergent copy
    - The existing `plugin/skills/orientation/`, `plugin/agents/governance-navigator.md` (064), and `plugin/hooks/` (063) are untouched, and no `marketplace.json` is added (distribution is #70)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (thin skill + subagent), ADR-2 (`plugin/agents/` reuse), ADR-3 (write-capable-but-fenced grant + prompt fence), ADR-4 (compose shipped tension commands, no CLI code)
  - **Scenario references**: tension-processing-path.feature — "A voiced tension is captured on the sensing role", "A captured tension is refined without recapturing", "A moot tension is retired rather than pushed to a proposal", "A rejected capture fabricates no id", "The processor is reachable once the plugin registers it", "A missing processor degrades the path to guidance", "The result is a synthesized tension record, not raw output", "A tension is situated against existing and rolled-up tensions", "A duplicate surfaces the existing tension instead of recording a second", "A failed situating read yields a partial record", "A ready tension is handed off without drafting a proposal", "The path performs only operational tension writes", "The path stays on the ungated operational side of the guardrail", "The path processes the tension without judging authority or coaching"
  - **Interface references**: interface-spec.md — Surface (structural layout, `SKILL.md` / agent frontmatter, required sections, single-source leaf list, tension-record output shape), Interactions (skill→agent delegation flow), Error Communication (capture rejected, situating failure, duplicate, ready-handoff, moot-discard, write nuance)

- [ ] **T002** [Shared] Add the best-effort drift-guard test in `internal/build` — 1 scenario un-@wip'd ("The path names no command the CLI lacks"); pins the composed tension leaves AND asserts they are disjoint from 063's gated proposal-write set, uncovered scope documented in the test, not omitted silently — TestTensionProcessingDriftGuard (composed⊆live registry + disjoint-from-063 + both sides source-derived, not hard-coded) + unit tests proving it's not fail-open; partial coverage (flags, prose, parser) stated in the test
  - **Scope**: A new `internal/build` test asserting every `glassfrog tension` leaf the skill/agent name still exists in the CLI's command registry **and** that those leaves are disjoint from 063's gated proposal-write set (the ungated-writes invariant ADR-3 depends on). Best-effort and explicitly partial — pins the *existence* of the command leaves and their *gate-membership*, not their flags (deferred to `--help`), not the tension-record prose, and not parser robustness. If an anchor proves infeasible to assert, state the reduced coverage rather than dropping it silently.
  - **Acceptance criteria**:
    - Test asserts the tension leaves the artifacts compose — `tension list`, `tension get`, `tension subroles`, `tension create`, `tension update`, `tension discard` — each resolve to a real command in the CLI's registry, reading the composed set from the single-source leaf-list file (T001), not a hard-coded copy
    - Test asserts those composed tension leaves are **disjoint** from 063's gated proposal-write set (source-derived from 063's `gated-commands.txt`), so a future change that pulls a tension leaf into the gated set (or a proposal leaf into the composed set) fails the build
    - Test fails loudly and names the offending leaf when one no longer exists in the shipped CLI or violates the disjointness invariant
    - Any leaf or fact deliberately left uncovered is documented in the test, not omitted silently (no silent caps)
    - Reuses the `internal/build` config-guard home/idiom established by 062/063/064 (064's `governance_navigation_guard_test.go` and 063's `write_safety_guardrail_guard_test.go` + `gated-commands.txt` single-source are the concrete models)
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-5 (best-effort drift guard + ungated-invariant cross-check)
  - **Scenario references**: tension-processing-path.feature — "The path names no command the CLI lacks"
  - **Interface references**: interface-spec.md — Error Communication (drift guard fails → CI red; reduced coverage permitted only if stated), Surface (single-source leaf list)
