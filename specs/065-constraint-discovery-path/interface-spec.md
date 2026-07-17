# Interface Accord: Constraint Discovery Path — Specification

**Feature**: 065-constraint-discovery-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + read-only subagent), ADR-2 (compose shipped reads + drift guard), ADR-3 (clarify-when-vague lives in the skill), ADR-4 (surface-and-characterize, never locally rule)
**Inputs**: spec.md, plan.md, PROJECT.md; the plugin/skill/agent frontmatter contracts are grounded against 064's landed `governance-navigation` skill + `governance-navigator` agent (this repo) and, for shape, real installed Claude plugins under `~/.claude/plugins/` — the latter are external reference examples, **not** files in this repository

> The artifact *is* the interface: two declarative plugin components — a `constraint-discovery` **skill** and a read-only `constraint-navigator` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the `plugin.json` (auto-)registration, the two files' frontmatter, their required sections, the clarify-when-vague interaction the skill owns, and the shape of the synthesized picture the agent returns.

---

## Surface

### Invocation

Two entry points, one delegating to the other:

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/constraint-discovery/SKILL.md` | The skill's frontmatter `description` matches the need "is this action within my authority / does it fall under another role's domain / is it shaped by a policy / does it need a proposal?" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/constraint-navigator.md` | After the skill has a well-formed action (clarifying with the operator first if the action is too vague), it instructs the caller to **delegate** the read traversal to this subagent (via the host's agent-invocation surface); the subagent runs in its own context and returns the synthesized picture |

No flags or arguments on either — the wanted action is passed as natural-language input. The skill's one interactive step (clarify-when-vague, ADR-3) uses the **host's structured ask-the-operator mechanism** (an `AskUserQuestion`-style tool) in the caller's context; the exact tool is host-provided (see Consistency Notes).

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                          # manifest — no component-array edit needed (auto-discovery, ADR-1)
  skills/
    constraint-discovery/
      SKILL.md                           # when + workflow + clarify-when-vague, delegates to the agent (required)
  agents/
    constraint-navigator.md              # read-only subagent that executes the read traversal (required)
internal/build/
    constraint_discovery_guard_test.go   # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/skills/constraint-discovery/` and `plugin/agents/constraint-navigator.md` are new sibling files; nothing 062/064 shipped (the `orientation` / `governance-navigation` skills, the `governance-navigator` agent) is touched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

Additive only, and in practice **none required**. Skills and agents are auto-discovered from `skills/` and `agents/` respectively — confirmed by 063's `hooks.json` (no `hooks` array) and 064's landed `governance-navigator` agent (no `agents` array). **Add no `skills`/`agents` key.** `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the 062/064 skill convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `constraint-discovery` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (you want to know whether a wanted action is within your authority, falls under another role's domain, is shaped by a policy, or needs a proposal) and that it returns a synthesized picture of the governing governance. Worded to fire on that authority/constraint need and **not** on "work a tension / who fills this role" (that's 064 governance-navigation) or "how do I drive the CLI" (that's 062 orientation) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* (incl. the clarify branch) + *delegation*, not a second copy of the traversal:

| Section | Must convey |
|---|---|
| When to reach for it | The job: find out whether a wanted action is within the operator's authority, under another role's domain, policy-shaped, or a governance change needing a proposal; and the boundaries — not working a tension in general (064), not driving the CLI (062), not writing anything |
| The workflow | The single-sourced steps: wanted action → **if too vague to search, clarify with the operator first (ADR-3)** → `search` to discover the domains/policies/roles it touches → read the owning role(s) (`roles [id]`/`tree`) and their `domains`/`policies` (and `policy [pol-id]` when search returns a policy directly) → optionally read the caller's own roles (`me roles`) to tell "your own" from "another role's" → characterize the authority situation → synthesize. Where a search or list spans multiple pages it pages through the full set (orientation pagination) **before** narrowing, so narrowing never silently drops unfetched pages |
| Clarify-when-vague | The skill (not the agent) detects a too-vague action and asks the operator to sharpen it via the host's structured ask mechanism, **before** delegating; if the operator declines to sharpen, the path stops here with no traversal (ADR-3) — it never guesses an action |
| Delegation | Instruction to run the `constraint-navigator` subagent for the read traversal so the raw reads stay out of the caller's context; and what the caller gets back (the synthesized picture with its characterization) |
| Surface-not-rule + read-only note | States the path only reads and only **surfaces + characterizes** the governing governance from the record — it never computes a permission verdict from local rules, and when the record is unclear it says so rather than guessing. Points at orientation (062) for output/exit-code/pagination mechanics rather than restating them |

### `agents/constraint-navigator.md` frontmatter

YAML, matching 064's `governance-navigator` convention (`name`, `description`, `tools`, optional `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `constraint-navigator` |
| `description` | string | yes | Read-only constraint navigator: given a well-formed wanted action, traverses the governing domains/policies (and the owning roles, and the caller's own roles) and returns a synthesized picture that **characterizes** the authority situation without ruling. Phrased so the skill (and host) route the read traversal here |
| `tools` | string list | yes | **Read-only grant.** Includes `Bash` (required to invoke `glassfrog` reads) plus `Read`/`Grep`/`Glob` as needed; **excludes `Write` and `Edit`** so the agent cannot mutate the workspace. See Error Communication for the governance-write nuance. **No interactive/ask tool** — the agent is non-interactive (ADR-3) |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/constraint-navigator.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a read-only constraint navigator) and its hard limits: never writes to the governance record, never computes a permission verdict from local rules, never fabricates a ruling under uncertainty, never asks the operator (interaction is the skill's job) |
| Workflow | Executes the same single-sourced traversal the skill names (references it; does not restate a divergent copy), on a well-formed action it is handed |
| Composed reads | The exact `glassfrog` read leaves it may call: `search`, `roles`, `tree`, `domains`, `policies`, `policy`, `me roles` — and only these |
| Output contract | Returns **only** the synthesized picture (below), never a dump of raw command output, and never an allow/deny verdict |

### Synthesized-picture output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| action | string | The well-formed action the picture is about (echoed for the operator's confirmation) |
| domains | Domain[] | Governing domains in view: `id`, `role_id` (the role that holds it — always `role_…`; a circle is a role with `type: circle`, no `circle_` id namespace), `role_name`, brief, `owned_by_caller` (bool — whether `role_id` is among the caller's own roles, from `me roles`) |
| policies | Policy[] | Binding policies in view: `id` (`pol_…`), `role_id`, brief of what it grants/limits |
| characterization | string | The authority situation **drawn from the record**, one of: within the caller's own authority (no other-role domain governs it, no policy limits it) · falls under another role's domain (names the role + id) → needs that role's permission or a proposal · shaped by a policy (names it) → observe it · a change to governance structure → goes through a proposal by default · **the record does not clearly answer** (with what was found). Never an allow/deny verdict computed from local rules |
| notes | string[] | Narrowing / failure / uncertainty notes (e.g. "results narrowed — refine", "one read failed", "nothing constraining found", "ambiguous — record does not clearly answer") |

Every listed element carries the id needed to read it again or act on it (spec accord: the synthesis bridges back into the CLI). This is a *contract shape*, not a serialization format — the agent presents it as a readable picture, and each id remains actionable (e.g. `fillers <role_id>` to find whom to ask, or 064/066 as the next step).

---

## Interactions

**Invocation-to-output flow**:

1. A caller (agent or user) has an action a practitioner wants to take and its need matches the skill `description`; the host loads `SKILL.md` on demand.
2. **Clarify-when-vague (skill, caller context — ADR-3)**: if the action is too vague to search for its governing governance, the skill asks the operator to sharpen it via the host's structured ask mechanism. If the operator sharpens it, continue with the well-formed action; if they decline, the path stops here — no action is fabricated and the agent is never invoked.
3. The skill directs the caller to delegate the well-formed action to the `constraint-navigator` subagent.
4. The subagent runs the traversal **in its own context**: `search` on the action → resolve matched ids into `roles [id]` / `domains <role_id>` / `policies <role_id>` (or `policy <pol_id>` when search returns a policy directly) → read the caller's own roles (`me roles`) to mark `owned_by_caller` → follow into sub-roles (`tree`) only as far as relevance warrants. Multi-page searches/lists are paged through in full (orientation pagination) before the picture is narrowed.
5. The subagent synthesizes and returns **only** the picture (the shape above) with its `characterization` — the raw command output never leaves its context.
6. The caller receives the picture and can act on any element by its id (bridging back into the CLI, into 064 for the people around it, or into 066 / the proposal paths if a proposal is the next step).

**Instructional model**: the skill tells the caller *when to reach for constraint discovery, when to clarify, and to delegate*; the agent *performs* the read traversal and *synthesizes + characterizes*. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog <command> --help` (062 ADR-3). For output/pagination/exit-code mechanics both defer to the orientation skill (062).

**Single-sourced workflow**: the traversal steps are written once (in the skill) and referenced by the agent — the agent does not carry a second, potentially divergent copy (plan Risk 2). The clarify branch lives only in the skill and has no agent counterpart.

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime traversal outcomes** the agent surfaces:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to reading domains/policies by hand |
| `constraint-navigator.md` missing / not registered | Skill's delegation target is absent; the read traversal cannot run isolated (the skill's workflow is still readable as guidance, degraded — plan Risk 4) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a read-only win |
| Action too vague to search | The **skill** asks the operator to sharpen it; if declined, the path stops with no traversal and no fabricated action (spec edge case; ADR-3) |
| Search matches nothing | Agent returns a picture with an explicit "nothing constraining found — refine the action" note; **no fabricated** domains/policies/ruling (spec happy/edge case) |
| A read in the traversal fails (e.g. a `policies` read errors) | Agent surfaces the failure in `notes` and returns the picture from the reads that succeeded — does not abort or invent (spec error scenario) |
| Record does not clearly answer (no domain plainly owns it, conflicting partial matches) | `characterization` = "the record does not clearly answer" **with what was found**; **never** a fabricated authority ruling (spec error scenario + ADR-4) |
| Over-broad action (many matches) | Agent pages through the full result set, then returns the most relevant governing constraints with a "narrowed — refine" note, not every match — never a silent single-page cap (spec edge case; Constitution VI) |
| Drift guard fails (`internal/build` test red) | A composed command leaf the artifacts name no longer exists in the CLI — the truthfulness contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Governance-write nuance (consistent with plan ADR-4)**: the `tools` grant gates *Claude tools*, not individual `glassfrog` subcommands, so withholding `Write`/`Edit` blocks workspace mutation but cannot by itself stop a `glassfrog` write issued through `Bash`. 065 is a **read** path — it drives no write — so this is a defense-in-depth note, not a live path: (1) the agent prompt scopes it to the read leaves above; (2) 063's landed `PreToolUse` write-safety hook gates any proposal-write `Bash` call and passes the navigator's reads through ungated. **Caveat (as in 064):** whether the host applies that hook to a **subagent's** Bash calls is not settled by the plugin; if it does not, the read-only property rests on layer (1) plus the `Write`/`Edit`-withheld grant. Since 065 issues only reads, it does not depend on the hook.

Runtime HTTP error shapes, status rendering, and rate-limit handling are **N/A** here — they belong to the CLI the agent drives (015/017/031/032) and are surfaced through the exit-code reactions orientation (062) already documents.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Reuses 064's skill+agent pattern** (plan ADR-1) — the second read path, following the pattern 064's DECISIONS entry named 065 as a reuser of. Silent conformance; no new divergence from 062 ADR-2 beyond the one 064 already announced.
- **Introduces the clarify-in-skill interaction** (plan ADR-3) — the one shape 064's navigator did not have. The interaction lives in the **skill** (caller context) because an isolated subagent has no dependable channel to prompt the operator; the agent stays non-interactive. Recorded in DECISIONS.md as the reusable "interaction in the skill, isolated execution in the agent" pattern for later paths.
- **Composed-reads deviation from the FEATURE-MODEL dep list** *(flagged)*: the FEATURE-MODEL lists 065's deps as Operator Orientation + Cross-Model Search + Role Domains + Role Policies. This accord additionally names two shipped reads the spec's behaviour requires: **Role Reads** (`roles`/`tree`, to read the role that *holds* a governing domain) and **`me roles`** (the caller's own roles, to mark `owned_by_caller` and so tell "your own authority" from "another role's domain" per the spec's Characterization accord). Both are already-shipped self-service reads and both only *surface facts from the record* — they do not add API surface and do not reimplement permission logic (VISION Exclusion 2 holds). Role Fillers is still **not** composed; the picture names the owning `role_id` and hands whom-to-ask to `fillers`/064.
- **Consistent with plan ADR-4** on the surface-not-rule guarantee: the `characterization` field surfaces the authority situation from the record and its "record does not clearly answer" value is a first-class outcome, never a fabricated verdict. Verified by a held-out validation scenario, not a tool grant.
- **`domains`/`policies`/`policy` are top-level commands** (`domains <id>`, `policies <role-id>`, `policy <pol-id>`) — not `roles` subcommands, and there is no `roles get`; the drift guard pins the real leaves, avoiding the invented-leaf trap (same care 064 took).
- **Depends on the orientation skill (062)** for output/pagination/exit-code mechanics — single-sourced there, referenced not duplicated.
- **Frontmatter grounded** against 064's landed artifacts in this repo and installed plugins under `~/.claude/plugins/` (external examples, not repo files); the exact `tools` token list and the host's structured ask-the-operator tool name are confirmed against the target host at build time.
