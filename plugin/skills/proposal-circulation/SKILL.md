---
name: proposal-circulation
description: Move an already-created proposal through the consent lifecycle — advancing a draft into circulation, monitoring where a circulating proposal stands on its way toward acceptance, or withdrawing a circulating proposal back to draft for amendment — through the guardrail-confirmed bodyless transitions (both writes gated, each confirmed independently), returning a circulation record carrying the prp_ id. Reach for this whenever a draft proposal is ready to enter circulation, a circulating proposal needs monitoring, or a circulating proposal must be pulled back to draft. It is not for assembling changes or creating the draft (that is the Proposal Drafting Path), it never records a no-objection or bring-to-meeting response (that is the response side), it is not for capturing, refining, or retiring a tension (that is the Tension Processing Path), it does not judge whether an action is allowed or needs a proposal (that is the Constraint Discovery Path), and it does not explain the governance around a concern (that is governance navigation).
---

# Circulating a proposal through the consent lifecycle

This skill takes a proposal that **already exists** — carrying its `prp_` id — and
moves or watches it through the consent lifecycle: **advancing** the draft into
circulation, **monitoring** where the circulating proposal stands, or
**withdrawing** it back to draft for amendment, returning a drawn-together
circulation record.

It composes only commands the `glassfrog` CLI already exposes. It adds no command,
flag, or capability of its own. Its **two** writes are the gated `proposal propose`
and `proposal withdraw`; every other write stays out of this path (see
[Gated-writes note](#gated-writes-note)).

## When to reach for it

Reach for proposal circulation when a proposal's `prp_` id is in hand and the job
is one of three circulation acts: a **draft proposal is ready to enter
circulation**, a **circulating proposal needs monitoring** for progress toward
acceptance, or a **circulating proposal must be withdrawn back to draft** so it
can be amended.

Do **not** reach for it to *assemble changes or create the draft* — that is the
**Proposal Drafting Path** (067); a created draft's `prp_` id is handed *to* this
path, not drafted here. Do not use it to *record a no-objection or
bring-to-meeting response* — that is the **response side** (069). Do not use it to
*capture, refine, or retire a tension* — that is the **Tension Processing Path**
(066). Do not use it to answer *"am I allowed to do X?"* or *"does this need a
proposal?"* — that authority verdict belongs to the **Constraint Discovery Path**
(065). And do not use it to *understand the governance around a concern* — that
traversal is the **governance-navigation** skill's job (064).

## The workflow

This is the single source of the circulation steps — the `proposal-circulator`
agent runs exactly this workflow; it does not carry a second copy. From the
proposal's `prp_` id and the intent (advance / monitor / withdraw):

1. **Ground** — `proposal get <prp-id>` reads the proposal exactly as the server
   returns it: its `status`, `response_summary`, `response_deadline`, and
   `available_transitions` — the picture every circulation act starts from.
2. **Situate (where relevant)** — when the circle's in-flight picture matters to
   the intent, `proposal list` surfaces the proposals already circulating. Where
   the relevant in-flight walk spans more than one page, **page through the full
   result set** (the default walk does this; see the orientation pagination
   guidance) — never a silent single-page cap.
3. **Act by intent**:
   - **Advance** — narrate the proposal and what `propose` will do, then run
     `proposal propose <prp-id>` — a **gated** governance write 063 confirms. On
     success the server returns the proposal in `proposed_outside_meeting`,
     carrying the server-set `response_deadline` and the proposer's implicit
     `no_objection` in the `response_summary`.
   - **Monitor** — draw the picture together from the reads: `response_summary`,
     `response_deadline`, and `available_transitions`, exactly as the server
     returned them. No write. The path **computes no acceptance** — the
     proposal's `status` is the signal.
   - **Withdraw** — narrate the proposal and what `withdraw` will do, then run
     `proposal withdraw <prp-id>` — the second **gated** write. On success the
     server returns the proposal back in `draft`, with
     `proposed_at`/`response_deadline` cleared and prior responses deleted —
     reflected in the returned record.
4. **Return the circulation record** and hand off: after a withdraw, hand the
   `prp_` id back to the **Proposal Drafting Path** (067) for re-editing; consent
   responses (`no_objection` / `bring_to_meeting`) belong to the **response side**
   (069), never this path.

The reads **inform, never gate**: the `available_transitions` snapshot is
narration for the proposer — it shows where the proposal stands — **never a
client-side precondition**. Issue the intended transition and let the server
authorize: a `422` refusal is surfaced **plainly**, and the call is never
pre-gated on the read snapshot. The server is the only transition authority.

For the exact flags of any one command, ask the CLI:
`glassfrog proposal <sub> --help`.

## Delegation

Run the **`proposal-circulator`** agent to execute this workflow. It performs the
grounding read, the monitoring walk, and the gated transitions in its **own
isolated context** and returns **only** the circulation record — so the raw
`proposal get`/`proposal list` output stays out of your context and never floods
it. You pass the `prp_` id and the intent (natural language); you get back the
circulation record (the proposal exactly as the server last returned it, the
in-flight proposals it was situated against, the action taken, the `prp_` handoff
id after a withdraw, and notes), with every element carrying the id needed to act
on it next.

If the circulator agent is absent or unregistered, the workflow above still
stands as guidance you can **follow by hand** — the path degrades to guidance,
and no CLI command is broken by the agent's absence.

## Gated-writes note

This path's **only** writes are `proposal propose` and `proposal withdraw`, and
each is a **governance write gated by the Write-Safety Guardrail (063)**: every
transition runs through the **confirmed write flow**. Both transitions are
**bodyless** — the confirmed command line (`glassfrog proposal propose prp_…` /
`glassfrog proposal withdraw prp_…`) *is* the complete payload, so the
confirmation shows everything that will run and nothing is hidden. Each
transition **confirms independently**: a session that advances and later
withdraws crosses the gate twice, and the confirmations are **never batched or
pre-authorized**. A **declined** confirmation means **no transition** happens.

This path performs **no other write**. It never runs `proposal create` —
assembling and creating the draft is the **Proposal Drafting Path** (067) — and
never `proposal respond` — recording a response is the **response side** (069);
063 gates those writes regardless. It never performs a tension write (that is the
Tension Processing Path, 066), it does not rule on whether the change is within
authority (that is the Constraint Discovery Path, 065), and it does not advise on
governance craft or coach Holacracy practice.

That write fence holds in layers: the circulator agent's prompt is scoped to the
four composed leaves and forbids any other `proposal` write, and 063's
`PreToolUse` write gate (`plugin/hooks/glassfrog-write-gate.sh`) interposes the
human confirmation on `proposal propose` and `proposal withdraw` — and would gate
any other proposal write regardless. In the target host, `PreToolUse` hooks fire
for a subagent's Bash calls too (the hook input carries an `agent_id` inside a
subagent call, confirmed by 066), so the gate reaches the circulator.

For output shapes, pagination mechanics, and exit-code reactions, see the
orientation skill (062) — this path does not restate them.
