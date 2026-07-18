# Tasks: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/unequipped-agent-operators/constraint-discovery-path.feature

---

## Dependency Graph

Phase 1: Constraint discovery path artifacts + drift guard (2 tasks, no phase dependencies; intra-phase: T002 depends on T001) [Shared]

2 tasks total | 0 phases parallelizable | Builder: pipeline (single active spec)

The plan is a single phase (ADR-1/3 land the two artifacts together; ADR-2's guard pins the read leaves they name). The two PR-sized units follow the plan's own suggested split: artifacts + registration first, drift guard second — the same shape 064 used for the sibling read path.

---

## Branching Guidance

**Pipeline mode**: `spec/065-constraint-discovery-path/base` → `spec/065-constraint-discovery-path/task-1`, `spec/065-constraint-discovery-path/task-2`

T002 depends on T001 (the guard pins the command leaves the artifacts name), so the two task branches land in sequence onto the spec base.

---

## Phase 1: Constraint discovery path artifacts + drift guard [Shared]

- [x] **T001** [Shared] Author the constraint-discovery skill (with the clarify-when-vague step) and the read-only constraint-navigator agent, and register the agent — 16 scenarios un-@wip'd (traversal, clarify-when-vague, own-vs-other characterization, incomplete-roles uncertainty, surface-not-rule, no-fabricated-ruling, boundary, read-only, registration/degradation) — skill + agent + single-sourced composed-reads list added under plugin/; BDD suite green; agent auto-discovered from plugin/agents/ (no plugin.json `agents` key), the orientation (062) and governance-navigation (064) artifacts untouched, no marketplace.json — done: 16 scenarios un-@wip'd and green, single source is `plugin/agents/constraint-discovery-composed-reads.txt`, no findings
  - **Scope**: Add `plugin/skills/constraint-discovery/SKILL.md` (the thin, discoverable entry point: when to reach for it, the single-sourced workflow *including the clarify-when-vague branch the skill owns*, the delegation-to-the-agent instruction, and the read-only + surface-and-characterize-never-rule guardrail note) and `plugin/agents/constraint-navigator.md` (the read-only, **non-interactive** executor: identity/scope, the workflow it runs — *referencing* the skill's workflow, not restating a divergent copy — the composed read leaves, and the synthesized-picture output contract with its `characterization` field). The agent is discovered by directory convention from `plugin/agents/` — **do not** add an `agents` key to `plugin/.claude-plugin/plugin.json` (064's landed `governance-navigator` agent and 063's `hooks.json` confirm directory auto-discovery; the manifest stays unchanged). Hand-authored, committed content; adds no CLI code.
  - **Acceptance criteria**:
    - `plugin/skills/constraint-discovery/SKILL.md` exists with YAML frontmatter `name` (`constraint-discovery`) + `description`; the `description` states *when* to reach for it (learning whether a wanted action is within the operator's authority, under another role's domain, policy-shaped, or needs a proposal) and that it returns a synthesized picture, worded not to fire on "work a tension / who fills this role" (064) or "how do I drive the CLI" (062)
    - The skill body carries the when / workflow / clarify-when-vague / delegation / surface-not-rule + read-only note, pointing at `glassfrog <command> --help` for per-command detail and at the orientation skill (062) for output/pagination/exit-code mechanics rather than restating them
    - The workflow states the clarify-when-vague step lives in the **skill** (caller context): when the wanted action is too vague to search, the skill asks the operator to sharpen it via the host's structured ask mechanism **before** delegating; if the operator declines, the path stops with no traversal and no fabricated action; the agent is never invoked on a guess (ADR-3)
    - The workflow states that where a **paging-capable** read (`search`, `roles`, `domains`, `policies`) spans multiple pages, the navigator pages through the full result set (per the orientation pagination guidance) **before** narrowing to "most relevant" — narrowing is a choice over the complete set, never a silent single-page cap (Constitution VI)
    - The workflow states that `me roles` does **not** follow pagination and can return an incomplete list (it emits an incompleteness note): when it signals incompleteness the navigator marks `owned_by_caller` **uncertain** (never a definite `false`) and surfaces that uncertainty, so a possibly-missing own-role is never misattributed as "another role's domain" (verified by the incomplete-roles-list scenario)
    - The agent is registered so the host discovers it and the skill's delegation resolves; if the agent is absent/unregistered, the skill's workflow remains usable as guidance with no CLI command broken (documented degradation)
    - Subagent hook coverage note carried from 064: 065 is a read path and drives no write, so it does not depend on 063's `PreToolUse` gate; regardless, the agent prompt is strictly read-only
    - `plugin/agents/constraint-navigator.md` exists with frontmatter `name` (`constraint-navigator`), `description`, and a **read-only `tools` grant** that includes `Bash` (to invoke `glassfrog` reads) and excludes `Write`/`Edit` (and includes no interactive/ask tool — the agent is non-interactive); the body references the skill's workflow, lists only the composed read leaves, and defines the synthesized-picture output
    - The synthesized-picture output contract includes the `characterization` field surfacing the authority situation drawn from the record — **composed** (not a single mutually-exclusive value) from: the domain finding (held by a role the caller fills → within their own authority · held by another role, named + id → needs its permission or a proposal · no domain governs it → the absence reported plainly, not a "permitted" verdict); **plus** any policies that shape the action, surfaced **even when it is within the caller's own domain** (policies compose with the domain finding); **plus** whether it is a governance change → needs a proposal; with **the record does not clearly answer** (with what was found) when the match is ambiguous — and never an allow/deny verdict computed from local logic; every element (`domains`, `policies`, owning `role_id`) carries its actionable id
    - The workflow steps are single-sourced in the skill and referenced by the agent — no second, divergent copy; the clarify branch lives only in the skill
    - The artifacts only read and only surface + characterize governing governance from the record: they name no write command, reimplement no permission rules, and fabricate no ruling under uncertainty; the existing `plugin/skills/orientation/`, `plugin/skills/governance-navigation/`, and `plugin/agents/governance-navigator.md` are untouched and no `marketplace.json` is added (distribution is #70)
  - **Dependencies**: None
  - **Plan reference**: Phase 1; ADR-1 (thin skill + read-only subagent), ADR-3 (clarify-when-vague in the skill), ADR-4 (surface-and-characterize, never locally rule; read-only layering)
  - **Scenario references**: constraint-discovery-path.feature — "A wanted action under another role's domain is surfaced with its owner", "An action under the caller's own role's domain is within their authority", "An incomplete roles list surfaces uncertainty, not a false 'not yours'", "A too-vague action is clarified by the skill before any traversal", "An over-broad action is narrowed, not dumped", "A failed read yields a partial picture", "The navigator is reachable once the plugin registers it", "A missing navigator degrades the path to guidance", "A policy that shapes the action is surfaced as a constraint to observe", "An unconstrained action surfaces the absence without asserting permission", "An ambiguous record is reported as unclear, not resolved by a guess", "The path surfaces and characterizes without ruling from local logic", "Under uncertainty the path says so rather than fabricating a verdict", "The result is a synthesized picture, not raw output", "Every element of the picture carries its actionable id", "The path only reads, never writes"
  - **Interface references**: interface-spec.md — Surface (structural layout, `SKILL.md` / agent frontmatter, required sections, synthesized-picture output shape with `characterization`), Interactions (clarify-when-vague branch + skill→agent delegation flow), Error Communication (too-vague clarify, empty search, partial failure, record-unclear, governance-write nuance)

- [x] **T002** [Shared] Add the best-effort drift-guard test in `internal/build` — 1 scenario un-@wip'd ("The path names no read the CLI lacks"); pins the composed read leaves against the shipped CLI, both sides source-derived (not hard-coded), with partial coverage documented in the test, not omitted silently — done: `constraint_discovery_guard_test.go` + `LiveMeSubcommands`/`CheckConstraintDrift` (the `me roles` leaf anchors via the `me` subcommand surface), unit tests prove the guard is not fail-open, no findings
  - **Scope**: A new `internal/build` test asserting every `glassfrog` read leaf the skill/agent name still exists in the CLI's command registry. Best-effort and explicitly partial — pins the *existence* of the command leaves, not their flags (deferred to `--help`), not the synthesized-picture prose, the `characterization` wording, or parser robustness. If an anchor proves infeasible to assert, state the reduced coverage rather than dropping it silently.
  - **Acceptance criteria**:
    - Test asserts the command leaves the artifacts compose — `search`, `roles`, `tree`, `domains`, `policies`, `policy`, and `me roles` — each resolve to a real command in the CLI's registry (`domains`/`policies`/`policy` are top-level commands, not `roles` subcommands, and `me roles` is the `roles` subcommand of `me`)
    - Test fails loudly and names the offending leaf when one no longer exists in the shipped CLI
    - Both the expected leaf set and the live command set are derived from source — the guard does not hard-code the list it guards (it reads the single-sourced composed-reads list the artifact also consumes)
    - Any leaf or fact deliberately left uncovered is documented in the test, not omitted silently (no silent caps)
    - Reuses the `internal/build` config-guard home/idiom established by 062/063/064 (064's `governance_navigation` drift guard + single-sourced `composed-reads.txt` are the concrete model)
    - Single-sources the composed-read leaf list in one file consumed by **both** the agent/skill artifact and the drift test — mirroring 064's `composed-reads.txt` and 063's `gated-commands.txt` — rather than duplicating the leaf list in prose and test
  - **Dependencies**: T001
  - **Plan reference**: Phase 1; ADR-2 (best-effort drift guard, single-sourced leaf list)
  - **Scenario references**: constraint-discovery-path.feature — "The path names no read the CLI lacks"
  - **Interface references**: interface-spec.md — Error Communication (drift guard fails → CI red; reduced coverage permitted only if stated)
