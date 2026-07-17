# Interface Accord: Governance Navigation Path — Specification

**Feature**: 064-governance-navigation-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + read-only subagent), ADR-2 (`plugin/agents/` component type), ADR-4 (drift guard), ADR-5 (tool-grant + prompt guardrails)
**Inputs**: spec.md, plan.md, PROJECT.md; the plugin/skill/agent frontmatter contracts are grounded against real installed Claude plugins under `~/.claude/plugins/` (score's `agents/guardian-agent.md`, the official `pr-review-toolkit` agents, the Luscii `example-plugin`) — external reference examples, **not** files in this repository

> The artifact *is* the interface: two declarative plugin components — a `governance-navigation` **skill** and a read-only `governance-navigator` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the `plugin.json` registration, the two files' frontmatter, their required sections, and the shape of the synthesized picture the agent returns.

---

## Surface

### Invocation

Two entry points, one delegating to the other:

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/governance-navigation/SKILL.md` | The skill's frontmatter `description` matches the need "understand the governance around a tension / who fills the roles / what domains & policies bear on it" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/governance-navigator.md` | The skill instructs the caller to **delegate** the traversal to this subagent (via the host's agent-invocation surface), passing the concern; the subagent runs in its own context and returns the synthesized picture |

No flags or arguments on either — the concern is passed as natural-language input to the agent.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                          # manifest — updated only as agent registration requires (ADR-2)
  skills/
    governance-navigation/
      SKILL.md                           # when + workflow, delegates to the agent (required)
  agents/                                # NEW component type (ADR-2)
    governance-navigator.md              # read-only subagent that executes the traversal (required)
internal/build/
    governance_navigation_guard_test.go  # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/agents/` is introduced here; `plugin/skills/orientation/` (062) is untouched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

Additive only. Skills are auto-discovered from `skills/` (062 convention), so no `skills` key is added. **Directory auto-discovery is confirmed by 063** (now landed): `plugin/hooks/hooks.json` is discovered from `plugin/hooks/` with **no** `hooks` array in this repo's `plugin.json`. By the same convention the agent is auto-discovered from `plugin/agents/` — **add no `agents` key**. `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the score-skill / 062 convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `governance-navigation` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (working a tension; needing the roles/fillers/domains/policies around a concern) and that it returns a synthesized picture — worded to fire on that need and **not** on "how do I drive the CLI" (that's orientation) or "am I allowed to do X" (that's 065) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* + *delegation*, not a second copy of the traversal:

| Section | Must convey |
|---|---|
| When to reach for it | The job: work a tension by understanding the governance around a free-form concern; and the boundaries — not tension *capture* (066), not authority *judgment* (065) |
| The workflow | The single-sourced traversal steps: concern → `search` to discover what it touches → read the relevant roles (`roles [id]`) and fillers (`fillers` / `subrole-actors`) → draw in governing `domains` / `policies` → synthesize. Bounded by relevance, not a full-tree walk; where a search or list spans multiple pages it pages through the full set (per the orientation pagination guidance) *before* narrowing, so narrowing never silently drops unfetched pages |
| Delegation | Instruction to run the `governance-navigator` subagent for execution so the raw reads stay out of the caller's context; and what the caller gets back (the synthesized picture) |
| Read-only + surfacing note | States the path only reads and only *surfaces* governing governance — it hands "can I do X?" to 065. Points at orientation (062) for output/exit-code/pagination mechanics rather than restating them |

### `agents/governance-navigator.md` frontmatter

YAML, matching the installed-plugin agent convention (`name`, `description`, `tools`, optional `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `governance-navigator` |
| `description` | string | yes | Read-only governance navigator: given a concern, traverses roles/fillers/domains/policies and returns a synthesized picture. Phrased so the skill (and the host) route traversal work here |
| `tools` | string list | yes | **Read-only grant.** Includes `Bash` (required to invoke `glassfrog` reads) plus `Read`/`Grep`/`Glob` as needed; **excludes `Write` and `Edit`** so the agent cannot mutate the workspace. See Error Communication for the governance-write nuance |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/governance-navigator.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a read-only navigator) and its hard limits: never writes to the governance record, never judges authority |
| Workflow | Executes the same single-sourced traversal the skill names (references it; does not restate a divergent copy) |
| Composed reads | The exact `glassfrog` read leaves it may call: `search`, `roles`, `tree`, `fillers`, `subrole-actors`, `domains`, `policies` — and only these |
| Output contract | Returns **only** the synthesized picture (below), never a dump of raw command output |

### Synthesized-picture output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| roles | Role[] | Each: `id` (always `role_…` — a circle is a role with `type: circle`, keeping a `role_…` id; there is no `circle_` id namespace), `name`, one-line relevance to the concern |
| fillers | per-role list | Each role's actors: `actor_id` (`per_`/`agt_`), `name` |
| domains | Domain[] | Governing domains in view: `id`, `role_id`, brief |
| policies | Policy[] | Governing policies in view: `id`, `role_id`, brief |
| notes | string[] | Narrowing/failure notes (e.g. "results narrowed — refine", "one read failed", "nothing relevant found") |

Every listed element carries the id needed to read it again (spec accord: the synthesis bridges back into the CLI). This is a *contract shape*, not a serialization format — the agent presents it as a readable picture, and each id remains actionable.

---

## Interactions

**Invocation-to-output flow**:

1. A caller (agent or user) senses a governance concern and its need matches the skill `description`; the host loads `SKILL.md` on demand.
2. The skill states the workflow and directs the caller to delegate to the `governance-navigator` subagent, passing the concern.
3. The subagent runs the traversal **in its own context**: `search` on the concern → resolve matched role ids into `roles [id]` → `fillers` / `subrole-actors` for who fills them → `domains` / `policies` for governing governance, following into sub-roles only as far as relevance warrants. Multi-page searches/lists are paged through in full (orientation pagination) before the picture is narrowed, so "most relevant" is chosen over the complete set.
4. The subagent synthesizes and returns **only** the picture (the shape above) — the raw command output never leaves its context.
5. The caller receives the picture and can act on any element by its id (bridging back into the CLI, or into 065/066 as the next step).

**Instructional model**: the skill tells the caller *when to navigate and to delegate*; the agent *performs* the read traversal and *synthesizes*. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog <command> --help` (062 ADR-3). For output/pagination/exit-code mechanics both defer to the orientation skill (062) rather than restating them.

**Single-sourced workflow**: the traversal steps are written once (in the skill) and referenced by the agent — the agent does not carry a second, potentially divergent copy (plan Risk 1).

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime traversal outcomes** the agent surfaces:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to driving reads by hand |
| `governance-navigator.md` missing / not registered | Skill's delegation target is absent; traversal cannot run isolated (the skill's workflow is still readable as guidance, degraded — plan Risk 2) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a read-only win |
| Search matches nothing | Agent returns a picture with an explicit "nothing relevant found — refine the concern" note; **no fabricated** roles/governance (spec error scenario) |
| A read in the traversal fails (e.g. leaf role has no sub-role fillers, or a read errors) | Agent surfaces the failure in `notes` and returns the picture from the reads that succeeded — does not abort or invent (spec error scenario) |
| Over-broad concern (many matches) | Agent pages through the full result set, then returns the most relevant with a "narrowed — refine" note, not every match — narrowing over the complete set, never a silent single-page cap (spec edge case; Constitution VI) |
| Concern is an authority question | Agent surfaces the governing domains/policies and defers the verdict to 065 — never rules on permission (spec edge case + ADR-5) |
| Drift guard fails (`internal/build` test red) | A composed command leaf the artifacts name no longer exists in the CLI — the truthfulness contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Governance-write nuance (consistent with plan ADR-5)**: the `tools` grant gates *Claude tools*, not individual `glassfrog` subcommands, so withholding `Write`/`Edit` blocks workspace mutation but cannot by itself stop a `glassfrog` proposal/tension write issued through `Bash`. Governance-write prevention is therefore **layered**: (1) the agent prompt scopes it to the read leaves above; (2) 063's now-landed `PreToolUse` write-safety hook (`plugin/hooks/glassfrog-write-gate.sh`) gates any proposal-write `Bash` call and passes the navigator's reads through ungated. **Caveat (063 landed):** that hook is registered as a `PreToolUse` matcher on `Bash`; whether the host applies it to a **subagent's** Bash calls is not settled by the plugin. If it does not, layer (2) does not cover the navigator and the read-only property rests on layer (1) (prompt scope) plus the `Write`/`Edit`-withheld grant. Implementation (T001) should confirm subagent hook coverage against the target host and keep the prompt strictly read-only regardless.

Runtime HTTP error shapes, status rendering, and rate-limit handling are **N/A** here — they belong to the CLI the agent drives (015/017/031/032) and are surfaced through the exit-code reactions orientation (062) already documents.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Follows 062's plugin conventions**: `author`/`keywords` object-and-array manifest shape, skill auto-discovery (no `skills` array), hand-authored committed content, and deferral of per-command detail to `--help`. Extends them additively with the first `agents/` component (plan ADR-2).
- **Introduces the skill+agent pattern** (plan ADR-1) — an **announced divergence** from 062 ADR-2's projection that the paths "arrive as sibling skills." Recorded in DECISIONS.md; #65 (Constraint Discovery, also a read path) may reuse this pattern.
- **Consistent with plan ADR-5** on the tool-grant's reach (see Error Communication): both state the layered reality — the tool grant blocks `Write`/`Edit` (workspace mutation), and governance-write prevention is prompt scope + 063's `PreToolUse` hook when present.
- **Depends on the orientation skill (062)** for output/pagination/exit-code mechanics and on 063's write-safety hook for the governance-write guarantee — both single-sourced elsewhere, referenced not duplicated.
- **Agent frontmatter grounded** against installed plugins under `~/.claude/plugins/` (external examples, not repo files); the exact `tools` token list and any host-specific `agents` registration key are confirmed against the target host at build time.
