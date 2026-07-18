# Interface Accord: Proposal Drafting Path — Specification

**Feature**: 067-proposal-drafting-path
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture + ADR-1 (thin skill + subagent), ADR-2 (new `proposal-drafter` agent, no reuse of 066's processor), ADR-3 (gated create inside the subagent, `--changes` inline), ADR-4 (compose shipped proposal/tension commands), ADR-5 (drift guard + gated-membership invariant)
**Inputs**: spec.md, plan.md, PROJECT.md; frontmatter contracts grounded against the shipped sibling artifacts in this repository (`plugin/skills/tension-processing/SKILL.md`, `plugin/agents/tension-processor.md`, `plugin/agents/tension-processing-commands.txt`) and the 064/065/066 accords

> The artifact *is* the interface: two declarative plugin components — a `proposal-drafting` **skill** and a `proposal-drafter` **agent** — consumed by on-demand loading and delegation, not a runtime call. Protocol-level contracts are the two files' frontmatter, their required sections, the confirmation contract around the one gated write, the single-source leaf list the drift guard consumes, and the shape of the draft record the agent returns.

---

## Surface

### Invocation

Two entry points, one delegating to the other (mirroring 064/065/066):

| Consumer | Entry point | Trigger |
|---|---|---|
| AI agent or human user | `plugin/skills/proposal-drafting/SKILL.md` | The skill's frontmatter `description` matches the need "a well-formed tension is ready to become a governance change — assemble the changes and create the draft proposal" — loaded **on demand** |
| The skill (as caller) | `plugin/agents/proposal-drafter.md` | The skill instructs the caller to **delegate** the drafting to this subagent, passing the anchor `ten_` id and the intended change (natural language and/or a prepared change set); the subagent runs in its own context and returns the draft record |

No flags or arguments on either — the anchor tension id and intended change are passed as natural-language input to the agent.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                            # manifest — untouched (auto-discovery)
  skills/
    proposal-drafting/
      SKILL.md                             # when + workflow, delegates to the agent (required)
  agents/                                  # reused component type (064 ADR-2)
    proposal-drafter.md                    # gated-write drafter subagent (required)
    proposal-drafting-commands.txt         # single-source composed-leaf list (ADR-5)
    tension-processor.md                   # 066 — untouched sibling
    governance-navigator.md                # 064 — untouched sibling
    constraint-navigator.md                # 065 — untouched sibling
internal/build/
    proposal_drafting_guard_test.go        # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/skills/orientation/` (062), the sibling path skills (064/065/066), `plugin/agents/` siblings and their leaf lists, and `plugin/hooks/` (063 — including `gated-commands.txt`, which this feature *reads* in its guard but never edits) are untouched. `marketplace.json` remains absent (distribution is #70).

### `plugin.json` changes

None. Skills are auto-discovered from `skills/` and agents from `agents/` (confirmed by 063's `hooks.json`, 064's navigator, 065's and 066's landed artifacts). `version` MAY bump per the surface's pre-1.0 discretion. No existing field is rewritten.

### `SKILL.md` frontmatter

YAML, matching the 062/064/065/066 convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-drafting` |
| `description` | string | yes | **The trigger.** States *when* to reach for it (a well-formed tension in hand, ready to become a governance change; assemble the changes and create the draft proposal, returning its `prp_` id) and that the create is a gated write run through explicit confirmation. Worded to fire on that need and **not** on "capture/refine/retire a tension" (that's 066), "am I allowed to do X" (that's 065), "understand the governance around a concern" (that's 064), or "advance/respond/withdraw a circulating proposal" (that's 068/069) |

### Required sections in `SKILL.md`

The body is thin — *when* + *workflow* + *delegation* + the gated-write note, not a second copy of the steps:

| Section | Must convey |
|---|---|
| When to reach for it | The job: from a ready tension's `ten_` id, assemble the governance changes and create the draft proposal; and the boundaries — not tension *processing* (066), not authority *judgment* (065), not governance *understanding* (064), not *circulation* (068/069) |
| The workflow | The single-sourced steps: anchor `ten_` id → ground the draft (`tension get <ten-id>`) → situate against the proposals already in flight (`proposal list --role-id <circle> --status draft`, paged through the **full** set before judging duplicates; `proposal get <prp-id>` to inspect a candidate match) → assemble the `changes` JSON array (each element an object with a non-empty `type`; verbatim above that floor — no typed builders) → surface anchor + change set for confirmation → create the draft (`proposal create <ten-id> --changes '<inline JSON>'`, a **gated** write 063 confirms) → return the draft record and hand the `prp_` id to the Proposal Circulation Path (068). Situating narrows by circle + `draft` status — the list has **no** tension filter, and the workflow must not imply one |
| Delegation | Instruction to run the `proposal-drafter` subagent for execution so the situating walk and raw output stay out of the caller's context; and what the caller gets back (the draft record). If the agent is absent, the workflow remains usable as guidance (documented degradation) |
| Gated-write note | States that `proposal create` is a **governance write gated by the Write-Safety Guardrail (063)**: the create always runs through the confirmed write flow — the confirmation shows the exact command with the change set inline — and a declined confirmation means no proposal is created. The path's *only* write is the create; it never runs `proposal propose`/`respond`/`withdraw` (068/069, gated by 063 regardless). Points at orientation (062) for output/exit-code/pagination/quoting mechanics rather than restating them |

### `agents/proposal-drafter.md` frontmatter

YAML, matching the shipped sibling agents (`name`, `description`, `tools`, `model`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `proposal-drafter` |
| `description` | string | yes | Gated-write proposal drafter: given a ready tension's `ten_` id and the intended change, grounds the draft in the tension, situates against proposals already in flight, assembles the change set, and creates the draft proposal through the guardrail-confirmed write — returning a draft record carrying the `prp_` id. Never advances, responds, or withdraws; never judges authority. The proposal-drafting skill delegates here |
| `tools` | string list | yes | **Write-capable-but-fenced grant**: `Bash, Read, Grep, Glob` — includes `Bash` (to invoke the `glassfrog` reads and the one gated create); **excludes `Write` and `Edit`** so the agent cannot mutate the workspace — which also blocks change-set temp files, keeping inline the only honest `--changes` source (ADR-3) |
| `model` | string | optional | `inherit` unless a specific tier is chosen |

### Required sections in `agents/proposal-drafter.md`

| Section | Must convey |
|---|---|
| Identity & scope | Who it is (a proposal drafter) and its hard limits: exactly **one** write — `proposal create`, always through 063's gate; **never** `proposal propose`/`respond`/`withdraw`; never a tension write (066's territory); never an authority verdict (065's); never a typed per-change validator (deferred *Unguided Change Construction*) |
| Workflow | Executes the same single-sourced steps the skill names (references them; does not restate a divergent copy) |
| Confirmation contract | Before the create: narrate the anchor tension and the assembled change set. The create passes `--changes` **inline** so 063's confirmation prompt displays the exact payload being written — never a file path or `stdin` that would make the human confirm blind. A declined confirmation is an outcome (`action: declined`), not an error |
| Composed commands | The exact `glassfrog` leaves it may call: `tension get`, `proposal list`, `proposal get` (reads) and `proposal create` (the one gated write) — and only these |
| Output contract | Returns **only** the draft record (below), never a dump of raw command output |

### Draft-record output shape

The agent's return value — the contract the caller consumes. Field names/types are the contract; the rendering is the agent's prose:

| Element | Type | Notes |
|---|---|---|
| draft | Proposal? | The created proposal: `id` (`prp_…`), `status` (`draft`), `tension_id` (`ten_…`), `changes` (the array as accepted). Absent when nothing was created (declined, rejected, or surfaced-existing) |
| anchor | Tension | The tension the draft is grounded in: `id` (`ten_…`), `sensing_role_id` (`role_…`), one-line summary |
| situating | Proposal[] | The in-flight proposals weighed during the duplicate check: each `id`, `status`, one-line summary — the context that made the create a deliberate addition |
| action | string | What was done: `created` \| `surfaced-existing` \| `declined` \| `none` |
| handoff | string? | When a draft exists and is ready to circulate: the `prp_` id to feed the Proposal Circulation Path (068). Absent otherwise |
| notes | string[] | Duplicate/failure/decision notes (e.g. "matching draft already in flight — surfaced prp_…", "create rejected: 403 async proposals not enabled", "situating walk incomplete: page 3 failed", "confirmation declined — nothing created") |

Every listed element carries the id needed to read it again or feed it onward (spec accord: the record bridges back into the CLI). A contract shape, not a serialization format.

### Single-source leaf list (ADR-5)

The composed leaves are written **once** in `plugin/agents/proposal-drafting-commands.txt`, consumed by both the agent's "Composed commands" reference and the drift guard — mirroring 063's `gated-commands.txt`, 064's `composed-reads.txt`, and 066's `tension-processing-commands.txt`. Contract: newline-delimited two-token leaves (`tension get`, `proposal list`, `proposal get`, `proposal create`), matching 063's registry format so membership checks compare like with like. The file's comments state the invariant the guard enforces: every leaf exists in the shipped CLI; `proposal create` **is a member of** 063's gated set; the three reads are **not**. One file, two consumers, no duplicated list.

---

## Interactions

**Invocation-to-output flow**:

1. A caller has a ready tension (066's handoff or a practitioner-identified `ten_` id) and the need matches the skill `description`; the host loads `SKILL.md` on demand.
2. The skill states the workflow and directs the caller to delegate to the `proposal-drafter` subagent, passing the anchor `ten_` id and the intended change.
3. The subagent runs the drafting **in its own context**: `tension get <ten-id>` to ground the draft → `proposal list --role-id <circle> --status draft` (full walk) to situate, `proposal get <prp-id>` to inspect a candidate match → assembles the change set (JSON array, non-empty `type` per element, verbatim above the floor) → narrates anchor + change set → runs `proposal create <ten-id> --changes '<inline JSON>'`.
4. 063's `PreToolUse` hook interposes: the practitioner sees the exact command — payload inline — and confirms or declines. (Hook coverage inside subagent calls was empirically confirmed by 066: hook input carries `agent_id`.) Declined → nothing created, `action: declined`.
5. The subagent returns **only** the draft record — raw command output never leaves its context.
6. The caller receives the record and can act on any element by its id — feeding the `prp_` id to the Proposal Circulation Path (068), re-reading via `proposal get`, or returning to 066 to refine the anchor tension.

**Instructional model**: the skill tells the caller *when to draft and to delegate*; the agent *performs* the ground/situate/assemble/confirm/create and *synthesizes* the record. Neither adds a `glassfrog` subcommand. For per-command flags both defer to `glassfrog proposal <sub> --help` / `glassfrog tension get --help` (062 ADR-3). For output/pagination/exit-code/quoting mechanics both defer to the orientation skill (062).

**Single-sourced workflow**: the steps are written once (in the skill) and referenced by the agent — no second, divergent copy (plan Risk 1).

---

## Error Communication

Specification artifacts fail as **constraint violations** and as **runtime drafting outcomes** the agent surfaces:

| Condition | Behavior |
|---|---|
| `SKILL.md` missing / no `description` | Skill not discoverable — the path is effectively absent; callers fall back to driving `proposal` commands by hand (still gated by 063) |
| `proposal-drafter.md` missing / not registered | Skill's delegation target is absent; drafting cannot run isolated (the workflow remains readable as guidance, degraded — plan Risk 6) |
| Agent `tools` grant omits `Bash` | Agent cannot invoke `glassfrog` at all — misconfiguration, not a safety win |
| Confirmation declined at the 063 gate | `action: declined`, no proposal created, a "confirmation declined" note — an outcome, not an error (spec edge case) |
| Create rejected (`403` Premium refusal, `404` unknown anchor, `422` rejected change set) | Agent surfaces the failure by name in `notes` with `action: none`; **no fabricated** `prp_` id (spec error scenario) |
| Situating list fails mid-walk | Agent surfaces the failure in `notes` and weighs the proposals gathered so far, flagged incomplete — does not abort or invent (spec error scenario; 056 already flags the partial walk) |
| Matching draft already in flight | Agent returns the existing proposal with its id, `action: surfaced-existing`, no create — does not silently open a duplicate (spec edge case) |
| Draft created and ready to circulate | Agent sets `handoff` to the `prp_` id for 068 and stops — never advances, responds, or withdraws (spec edge case + ADR-3) |
| Change set too unwieldy for an inline command line | Agent surfaces it to the caller rather than smuggling it through a hidden file — inline is the only source that keeps the confirmation informative (ADR-3) |
| Drift guard fails (`internal/build` test red) | A composed leaf no longer exists in the CLI, **or** `proposal create` left 063's gated set, **or** a composed read entered it — the truthfulness/gated-invariant contract is broken; fix the artifact (or the claim) |
| Drift guard coverage reduced/omitted | Permitted **only if stated**, never silent — the test names what it does not cover (LEARNINGS: no silent caps) |

**Gated-write nuance (consistent with plan ADR-3)**: the `tools` grant gates *Claude tools*, not individual `glassfrog` subcommands — withholding `Write`/`Edit` blocks workspace mutation but the agent legitimately runs the one gated create through `Bash`. The write boundary is **layered**: (1) the agent prompt scopes execution to the four composed leaves and forbids every other proposal write; (2) 063's `PreToolUse` gate (`plugin/hooks/glassfrog-write-gate.sh`) interposes the human confirmation on `proposal create` — and fires inside subagent Bash calls (confirmed by 066 against Claude Code; hook input carries `agent_id`) — and would gate `propose`/`respond`/`withdraw` as backstop if the prompt fence ever failed. Unlike 066, whose writes pass the gate ungated by design, 067's single write is *supposed to be asked* — a create that executes without a confirmation prompt is itself a defect (the gated-membership drift-guard assertion exists to catch the registry-side cause).

Runtime HTTP error shapes, status rendering, optimistic-concurrency, and rate-limit handling are **N/A** here — they belong to the CLI the agent drives (015/017/031/032) and are surfaced through the exit-code reactions orientation (062) already documents.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (it adds no `glassfrog` subcommand, API, UI, or event surface).
- **Follows the 064/065/066 skill+agent pattern** (plan ADR-1): thin skill delegating to an isolated subagent, `plugin/agents/` home, directory auto-discovery, single-sourced workflow. Silent conformance — 064 carried the original divergence from 062 ADR-2.
- **New agent, not a reuse** (plan ADR-2): 066's `tension-processor` prompt forbids any `proposal …` command — a validate-pinned invariant this feature must not erode. The drafter is a sibling, and 066's artifacts stay untouched.
- **Diverges from 066's write posture** (plan ADR-3, announced): 066 = only the *ungated* tension writes, never a proposal write; 067 = only the one *gated* create, always through the gate, with the change set inline so the confirmation shows the payload. Recorded in DECISIONS.md.
- **Single-sources the leaf list** (plan ADR-5) following the 063/064/066 file precedent — with the guard asserting the **inverse** of 066's disjointness: the write leaf's *membership* in the gated set, and the reads' absence from it.
- **Depends on the orientation skill (062)** for output/pagination/exit-code/quoting mechanics and on 063's write gate for the confirmation — both single-sourced elsewhere, referenced not duplicated. Reads 063's `gated-commands.txt` in the guard; never edits it.
- **Hands off to sibling specs**: the created `prp_` id feeds Proposal Circulation (068); the anchor tension's lifecycle stays with Tension Processing (066); authority questions go to Constraint Discovery (065). None is invoked or implemented here.
- **Frontmatter grounded against shipped repo artifacts** (066's processor, 064's navigator, 065's constraint-navigator) rather than external plugin examples — the in-repo siblings are now the stronger precedent.
