# Interface Accord: Tension Processing Path — Specification

**Feature**: 066-tension-processing-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + subagent), ADR-2 (`plugin/agents/` reuse), ADR-3 (write-capable-but-fenced subagent), ADR-4 (compose shipped tension commands), ADR-5 (drift guard + ungated-invariant cross-check)
**Inputs**: spec.md, plan.md, PROJECT.md; the plugin/skill/agent frontmatter contracts are grounded against real installed Claude plugins under `~/.claude/plugins/` (score's `agents/guardian-agent.md`, the official `pr-review-toolkit` agents, the Luscii `example-plugin`) and against the sibling 064 accord — external reference examples, **not** files in this repository

> The artifact *is* the interface: two declarative plugin components — a `tension-processing` **skill** and a `tension-processor` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the `plugin.json` registration, the two files' frontmatter, their required sections, the single-source leaf-list the drift guard consumes, and the shape of the tension record the agent returns.

---

## Surface

### Invocation

Two entry points, one delegating to the other (mirroring 064):

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/tension-processing/SKILL.md` | The skill's frontmatter `description` matches the need "turn a voiced tension into a well-formed record — capture it on the sensing role, situate it against what's already sensed, refine or retire it" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/tension-processor.md` | The skill instructs the caller to **delegate** the processing to this subagent (via the host's agent-invocation surface), passing the voiced tension and the sensing role; the subagent runs in its own context and returns the tension record |

No flags or arguments on either — the tension and its sensing role are passed as natural-language input to the agent.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                          # manifest — additive only (ADR-2)
  skills/
    tension-processing/
      SKILL.md                           # when + workflow, delegates to the agent (required)
  agents/                                # reused component type (064 ADR-2)
    tension-processor.md                 # write-capable-but-fenced subagent (required)
    governance-navigator.md              # 064 — untouched sibling
  <single-source leaf list>              # composed tension leaves, consumed by artifacts + drift guard (ADR-5; exact path/format below)
internal/build/
    tension_processing_guard_test.go     # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/skills/orientation/` (062), `plugin/agents/governance-navigator.md` (064), and `plugin/hooks/` (063) are untouched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

Additive only. Skills are auto-discovered from `skills/` (062 convention) and agents from `agents/` (**confirmed** by 063's landed `hooks.json` directory auto-discovery and by 064's navigator) — so **add no `skills` or `agents` key**. `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the score-skill / 062 / 064 convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `tension-processing` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (a practitioner has a tension to act on and wants it recorded/refined/retired on the right role) and that it returns a well-formed tension record carrying its id. Worded to fire on that need and **not** on "understand the governance around a concern" (that's 064), "am I allowed to do X" (that's 065), or "draft/circulate a proposal" (that's 067/068) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* + *delegation*, not a second copy of the steps:

| Section | Must convey |
|---|---|
| When to reach for it | The job: turn a voiced tension into a well-formed record on the sensing role; and the boundaries — not governance *understanding* (064), not authority *judgment* (065), not proposal *drafting/circulation* (067/068) |
| The workflow | The single-sourced steps: voiced tension → situate (`tension list <role-id>` + `tension subroles <role-id>` to see what the role and its sub-roles already sense) → capture on the sensing role (`tension create <role-id>`) → refine (`tension update <ten-id>`) or retire (`tension discard <ten-id>`) → hand the ready `ten_` id to the Proposal Drafting Path (067). Situating over multi-page lists pages through the full set (per orientation pagination) before judging duplicates, so a duplicate check never silently drops unfetched pages |
| Delegation | Instruction to run the `tension-processor` subagent for execution so the situating reads stay out of the caller's context; and what the caller gets back (the tension record) |
| Write-boundary note | States the path performs only *operational* tension writes (ungated per 063) and **never** a proposal write — it hands the ready `ten_` id to 067 and hands "does this need a proposal / am I allowed?" to 065. Points at orientation (062) for output/exit-code/pagination mechanics rather than restating them |

### `agents/tension-processor.md` frontmatter

YAML, matching the installed-plugin agent convention (`name`, `description`, `tools`, optional `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `tension-processor` |
| `description` | string | yes | Tension processor: given a voiced tension and a sensing role, situates it against what's already sensed, captures it, refines or retires it, and returns the record. Phrased so the skill (and the host) route tension-processing work here |
| `tools` | string list | yes | **Write-capable-but-fenced grant.** Includes `Bash` (required to invoke `glassfrog tension` reads *and* writes) plus `Read`/`Grep`/`Glob` as needed; **excludes `Write` and `Edit`** so the agent cannot mutate the workspace. Unlike 064's navigator this is **not** a read-only grant — see Error Communication for the write nuance |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/tension-processor.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a tension processor) and its hard limits: only operational tension writes; **never** a proposal write (create/propose/respond/withdraw), never an authority verdict |
| Workflow | Executes the same single-sourced steps the skill names (references them; does not restate a divergent copy) |
| Composed commands | The exact `glassfrog` leaves it may call: `tension list`, `tension get`, `tension subroles`, `tension create`, `tension update`, `tension discard` — and only these; explicitly **no** `proposal …` command |
| Output contract | Returns **only** the tension record (below), never a dump of raw command output |

### Tension-record output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| tension | Tension | The captured/updated tension in hand: `id` (`ten_…`), `sensing_role_id` (`role_…`), `body`, `label?`, `status` |
| situating | Tension[] | The tensions already sensed on the role and rolled up from its sub-roles that the processor weighed: each `id`, `sensing_role_id`, one-line summary — the context that made the capture/refine a deliberate choice |
| action | string | What was done: `captured` \| `refined` \| `retired` \| `surfaced-existing` \| `none` |
| handoff | string? | When the tension is ready to become a governance change: the `ten_` id to feed the Proposal Drafting Path (067). Absent when not ready |
| notes | string[] | Duplicate/failure/decision notes (e.g. "already sensed — surfaced existing", "roll-up read failed", "capture rejected: unknown role", "discarded as moot") |

Every listed element carries the id needed to read it again or feed it onward (spec accord: the record bridges back into the CLI). This is a *contract shape*, not a serialization format — the agent presents it as a readable record, and each id remains actionable.

### Single-source leaf list (ADR-5)

The composed tension leaves are written **once** in a committed file consumed by both the artifacts' reference and the drift guard — mirroring 063's `plugin/hooks/gated-commands.txt` single-source pattern. Contract: a newline-delimited list of the six `tension` sub-leaves (`create`, `list`, `get`, `update`, `discard`, `subroles`). The exact path (e.g. `plugin/agents/tension-processing-commands.txt`) and whether the leaves are stored as bare sub-verbs or `tension <sub>` pairs are settled at build time (T-tasks); the invariant is one file, two consumers, no duplicated list.

---

## Interactions

**Invocation-to-output flow**:

1. A caller (agent or user) has a tension to act on and its need matches the skill `description`; the host loads `SKILL.md` on demand.
2. The skill states the workflow and directs the caller to delegate to the `tension-processor` subagent, passing the voiced tension and the sensing role.
3. The subagent runs the processing **in its own context**: `tension list <role-id>` + `tension subroles <role-id>` to situate against what's already sensed → if the tension is new, `tension create <role-id>` to capture it → `tension update <ten-id>` to refine, or `tension discard <ten-id>` to retire, as the practitioner directs. Multi-page situating lists are paged through in full (orientation pagination) before judging duplicates.
4. The subagent returns **only** the tension record (the shape above) — the raw command output never leaves its context.
5. The caller receives the record and can act on any element by its id — feeding a ready `ten_` id to the Proposal Drafting Path (067), or handing an authority question to the Constraint Discovery Path (065).

**Instructional model**: the skill tells the caller *when to process a tension and to delegate*; the agent *performs* the situate/capture/refine/retire and *synthesizes* the record. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog tension <sub> --help` (062 ADR-3). For output/pagination/exit-code mechanics both defer to the orientation skill (062) rather than restating them.

**Single-sourced workflow**: the steps are written once (in the skill) and referenced by the agent — the agent does not carry a second, potentially divergent copy (plan Risk 1).

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime processing outcomes** the agent surfaces:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to driving `tension` commands by hand |
| `tension-processor.md` missing / not registered | Skill's delegation target is absent; processing cannot run isolated (the skill's workflow is still readable as guidance, degraded — plan Risk 4) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a safety win |
| Capture rejected (unknown sensing role, blank/whitespace body) | Agent surfaces the usage/API failure by name in `notes` with `action: none`; **no fabricated** `ten_` id (spec error scenario) |
| A situating read fails (e.g. sub-role roll-up errors) | Agent surfaces the failure in `notes` and returns the record from the reads that succeeded — does not abort or invent (spec error scenario) |
| Tension already sensed (duplicate) | Agent returns the existing tension with its id, `action: surfaced-existing`, and a "already sensed" note — does not silently record a duplicate (spec edge case) |
| Tension ready to become a governance change | Agent sets `handoff` to the `ten_` id for 067 and stops — never drafts/creates/circulates the proposal itself (spec edge case + ADR-3) |
| Tension is moot | Agent retires it (`tension discard`), `action: retired` — does not push toward a proposal (spec edge case) |
| Drift guard fails (`internal/build` test red) | A composed tension leaf the artifacts name no longer exists in the CLI, **or** a leaf drifted into/out of 063's gated set — the truthfulness/ungated-invariant contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Write nuance (consistent with plan ADR-3)**: the `tools` grant gates *Claude tools*, not individual `glassfrog` subcommands, so withholding `Write`/`Edit` blocks *workspace* mutation but is **not** a read-only property — the agent legitimately runs `glassfrog tension create/update/discard` through `Bash`, which 063 leaves **ungated** by design. The boundary this path must not cross is a *proposal* write. That fence is **layered**: (1) the agent prompt scopes execution to the six tension leaves above and forbids any `proposal …` command; (2) 063's landed `PreToolUse` write-safety hook (`plugin/hooks/glassfrog-write-gate.sh`) gates any proposal-write `Bash` call and passes the tension writes through ungated. **Caveat (063 landed):** that hook is a `PreToolUse` matcher on `Bash`; whether the host applies it to a **subagent's** Bash calls is not settled by the plugin. If it does not, layer (2) does not cover the processor and the proposal fence rests on layer (1) (prompt scope) — which matters more here than for read-only 064, because this subagent legitimately writes. Implementation (T001) should confirm subagent hook coverage against the target host and keep the prompt strictly scoped regardless.

Runtime HTTP error shapes, status rendering, optimistic-concurrency (`If-Match`/`412`), and rate-limit handling are **N/A** here — they belong to the CLI the agent drives (015/017/031/032/052/054) and are surfaced through the exit-code reactions orientation (062) already documents.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Follows 064's skill+agent pattern** (plan ADR-1): reuses the `plugin/agents/` component type and the thin-skill-delegates-to-subagent shape 064 introduced. This is **silent conformance** to 064, not a re-divergence from 062 ADR-2 — 064 already carried that divergence and recorded it in DECISIONS.md.
- **Diverges from 064's read-only grant** (plan ADR-3): 064's navigator is read-only (`Write`/`Edit` withheld *and* prompt-scoped to read leaves); 066's processor withholds `Write`/`Edit` (no workspace mutation) but is **write-capable** at the `glassfrog` level, fenced to the ungated operational tension leaves. Announced divergence, recorded in DECISIONS.md.
- **Single-sources the leaf list** (plan ADR-5) following 063's `gated-commands.txt` precedent — one file, consumed by both the artifacts and the drift guard, which additionally asserts the tension leaves are **disjoint** from 063's gated proposal-write set (the ungated-writes invariant).
- **Depends on the orientation skill (062)** for output/pagination/exit-code mechanics and on 063's write-safety hook for the proposal-write fence — both single-sourced elsewhere, referenced not duplicated.
- **Hands off to sibling specs**: the ready `ten_` id feeds Proposal Drafting (067); authority questions go to Constraint Discovery (065). Neither is invoked or implemented here.
- **Agent frontmatter grounded** against installed plugins under `~/.claude/plugins/` (external examples, not repo files); the exact `tools` token list and any host-specific `agents` registration key are confirmed against the target host at build time.
