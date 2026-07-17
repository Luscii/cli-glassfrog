# Tasks: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/governance-navigation-path.feature

---

## Dependency Graph

Phase 1: Navigation path artifacts + drift guard (2 tasks, no phase dependencies; intra-phase: T002 depends on T001) [Shared]

2 tasks total | 0 phases parallelizable | Builder: pipeline (single active spec)

The plan is a single phase (ADR-1/2 land the two artifacts together; ADR-4's guard pins the leaves they name). The two PR-sized units follow the plan's own suggested split: artifacts + registration first, drift guard second.

---

## Branching Guidance

**Pipeline mode**: `spec/064-governance-navigation-path/base` → `spec/064-governance-navigation-path/task-1`, `spec/064-governance-navigation-path/task-2`

T002 depends on T001 (the guard pins the command leaves the artifacts name), so the two task branches land in sequence onto the spec base.

---

## Phase 1: Navigation path artifacts + drift guard [Shared]

- [x] **T001** [Shared] Author the governance-navigation skill and the read-only governance-navigator agent, and register the agent — 13 scenarios un-@wip'd (the traversal, synthesis, boundary, read-only, and registration/degradation behaviors) — skill + agent + single-sourced composed-reads.txt added under plugin/; BDD suite green (13 scenarios); agent auto-discovered from plugin/agents/ (no plugin.json `agents` key), orientation skill untouched, no marketplace.json
  - **Scope**: Add `plugin/skills/governance-navigation/SKILL.md` (the thin, discoverable entry point: when to reach for it, the single-sourced traversal workflow, the delegation-to-the-agent instruction, and the read-only + surfacing-not-judging guardrail note) and `plugin/agents/governance-navigator.md` (the read-only executor: identity/scope, the workflow it runs — *referencing* the skill's workflow, not restating a divergent copy — the composed read leaves, and the synthesized-picture output contract). The agent is discovered by directory convention from `plugin/agents/` — **do not** add an `agents` key to `plugin/.claude-plugin/plugin.json` (063's landed `plugin/hooks/hooks.json` is auto-discovered with no `hooks` array, confirming this plugin uses directory auto-discovery; the manifest stays unchanged). Hand-authored, committed content; adds no CLI code.
  - **Acceptance criteria**:
    - `plugin/skills/governance-navigation/SKILL.md` exists with YAML frontmatter `name` (`governance-navigation`) + `description`; the `description` states *when* to reach for it (working a tension; the roles/fillers/domains/policies around a concern) and that it returns a synthesized picture, worded not to fire on "how do I drive the CLI" (orientation) or "am I allowed to do X" (065)
    - The skill body carries the when / workflow (concern → `search` → `roles [id]` → `fillers`/`subrole-actors` → `domains`/`policies` → synthesize, relevance-bounded) / delegation / read-only + surfacing-not-judging note, pointing at `glassfrog <command> --help` for per-command detail and at the orientation skill (062) for output/pagination/exit-code mechanics rather than restating them
    - The workflow states that where a `search` or list read spans multiple pages, the navigator pages through the full result set (per the orientation pagination guidance) **before** narrowing to "most relevant" — narrowing is a choice over the complete set, never a silent single-page cap (Constitution VI)
    - The agent is registered so the host discovers it and the skill's delegation resolves; if the agent is absent/unregistered, the skill's workflow remains usable as guidance with no CLI command broken (documented degradation)
    - Subagent hook coverage is confirmed against the target host: verify whether 063's landed `PreToolUse` `Bash` gate (`plugin/hooks/glassfrog-write-gate.sh`) also fires for the navigator subagent's Bash calls; regardless of the answer, the agent prompt is strictly read-only (the load-bearing control if the hook does not reach the subagent — see risk.md H-4)
    - `plugin/agents/governance-navigator.md` exists with frontmatter `name` (`governance-navigator`), `description`, and a **read-only `tools` grant** that includes `Bash` (to invoke `glassfrog` reads) and excludes `Write`/`Edit`; the body references the skill's workflow, lists only the composed read leaves, and defines the synthesized-picture output (roles/fillers/domains/policies, each carrying its id; narrowing/failure notes)
    - The workflow steps are single-sourced in the skill and referenced by the agent — no second, divergent copy
    - The artifacts only read and only surface governing governance: they name no write command, and they defer the authority verdict to the Constraint Discovery Path (065); the existing `plugin/skills/orientation/` is untouched and no `marketplace.json` is added (distribution is #70)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (thin skill + read-only subagent), ADR-2 (`plugin/agents/` component type), ADR-3 (compose shipped reads, no CLI code), ADR-5 (read-only tool grant + prompt guardrail)
  - **Scenario references**: governance-navigation-path.feature — "Search a concern surfaces the relevant roles", "The navigator is reachable once the plugin registers it", "A missing navigator degrades the path to guidance", "An over-broad concern is narrowed, not dumped", "An empty search reports nothing found without fabricating", "A failed read yields a partial picture", "The result is a synthesized picture, not raw output", "A relevant role's domains and policies are drawn in", "A circle concern follows into its sub-roles", "An authority question surfaces governance but defers the verdict", "The path surfaces governance without judging authority", "Every element of the picture carries its actionable id", "The path only reads, never writes"
  - **Interface references**: interface-spec.md — Surface (structural layout, `SKILL.md` / agent frontmatter, required sections, synthesized-picture output shape), Interactions (skill→agent delegation flow), Error Communication (empty search, partial failure, authority-question deferral, governance-write nuance)

- [ ] **T002** [Shared] Add the best-effort drift-guard test in `internal/build` — 1 scenario un-@wip'd ("The path names no read the CLI lacks"); pins the composed read leaves, uncovered scope documented in the test, not omitted silently
  - **Scope**: A new `internal/build` test asserting every `glassfrog` read leaf the skill/agent name still exists in the CLI's command registry. Best-effort and explicitly partial — pins the *existence* of the command leaves, not their flags (deferred to `--help`), not the synthesized-picture prose, and not parser robustness. If an anchor proves infeasible to assert, state the reduced coverage rather than dropping it silently.
  - **Acceptance criteria**:
    - Test asserts the command leaves the artifacts compose — `search`, `roles`, `tree`, `fillers`, `subrole-actors`, `domains`, `policies` — each resolve to a real command in the CLI's registry
    - Test fails loudly and names the offending leaf when one no longer exists in the shipped CLI
    - Any leaf or fact deliberately left uncovered is documented in the test, not omitted silently (no silent caps)
    - Reuses the `internal/build` config-guard home/idiom established by 062/063 (063's `write_safety_guardrail_guard_test.go` + `writesafetyguardrail.go` are the concrete model)
    - Single-sources the composed-read leaf list in one file consumed by **both** the agent/skill artifact and the drift test — mirroring 063's `plugin/hooks/gated-commands.txt` (single source for the hook script and its drift tripwire) — rather than duplicating the leaf list in prose and test
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-4 (best-effort drift guard)
  - **Scenario references**: governance-navigation-path.feature — "The path names no read the CLI lacks"
  - **Interface references**: interface-spec.md — Error Communication (drift guard fails → CI red; reduced coverage permitted only if stated)
