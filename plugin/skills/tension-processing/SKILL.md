---
name: tension-processing
description: Turn a practitioner's voiced tension into a well-formed tension record on the role that senses it — situating it against what the role and its sub-roles already sense, capturing it, refining it, or retiring it — and return the record carrying its ten_ id, ready to hand onward. Reach for this whenever a practitioner has a tension to act on and wants it recorded, refined, or retired on the right role. It is not for understanding the governance around a concern (that is governance navigation), it does not judge whether an action is allowed or needs a proposal (that is the Constraint Discovery Path), and it never drafts or circulates a proposal (that is the Proposal Drafting and Circulation Paths).
---

# Processing a tension into a well-formed record

This skill is the second **operator path** on the agent operating surface — the
write-side counterpart to governance navigation. Its job is to take a
practitioner's **voiced tension** and the **sensing role** it belongs to and work
it into a **well-formed tension record**: situated against what is already
sensed, captured on the right role, refined or retired as the practitioner
directs, and carrying the `ten_` id that reads it again or feeds the next path.

It composes only the operational `tension` commands the `glassfrog` CLI already
exposes — the six leaves listed in `tension-processing-commands.txt` (under
`plugin/agents/`, the single source of the composed leaves). It adds no
command, flag, or capability of its own, and it never crosses into a governance
write (see [Write boundary](#write-boundary)).

## When to reach for it

Reach for tension processing when a practitioner has a **tension to act on**:

- A gap or issue has been voiced and needs to be **captured** on the role that
  senses it, as a deliberate addition to what is already on the record.
- A tension already on the record needs a **clearer body or better routing** —
  refine it in place rather than recapturing it as a second entry.
- A tension has become **moot** and should be retired from the record rather
  than pushed toward a proposal.

Do **not** reach for it to *understand the governance around a concern* — that
traversal is the **governance-navigation** skill's job. Do not use it to
answer *"am I allowed to do X?"* or *"does this need a proposal?"* — that
authority verdict belongs to the **Constraint Discovery Path**. And do not
use it to *draft or circulate a proposal* — a ready tension is handed to the
**Proposal Drafting Path**; this path stops at the record.

## The workflow

This is the single source of the processing steps — the `tension-processor`
agent runs exactly this workflow; it does not carry a second copy. From a voiced
tension and its sensing role:

1. **Situate** — see what is already sensed before adding to it:
   `tension list <role-id>` for the tensions already on the sensing role, and
   `tension subroles <role-id>` for the tensions rolled up from its direct
   sub-roles. Where a situating list read spans more than one page, **page
   through the full result set** (the default walk does this; see the
   orientation pagination guidance) **before judging duplicates** — the
   duplicate check is a judgment over the complete set, never a silent
   single-page cap.
2. **Check for a duplicate** — if the voiced tension matches one already on the
   record, **surface the existing tension with its id** and let the practitioner
   refine that one; do not silently record a second.
3. **Capture** — `tension create <role-id>` records the tension on the sensing
   role, as a deliberate addition made in view of what is already sensed. The
   created tension's `ten_` id is the handle every later step uses.
4. **Refine or retire** — `tension update <ten-id>` sharpens the body or
   routing of a tension already on the record (no recapture); `tension discard
   <ten-id>` retires a tension the practitioner decided is moot rather than
   pushing it toward a proposal.
5. **Hand off** — when the tension is ready to become a governance change,
   hand its `ten_` id to the **Proposal Drafting Path**. Drafting,
   creating, or circulating the proposal is that path's job, never this one's.

For the exact flags of any one command, ask the CLI:
`glassfrog tension <sub> --help`. For output formats, pagination, and exit-code
reactions, consult the **orientation** skill rather than restating them
here.

## Delegation

Run the **`tension-processor`** agent to execute this workflow. It performs the
situating reads and the tension writes in its **own isolated context** and
returns **only** the tension record — so the raw `tension list`/`tension
subroles` output stays out of your context and never floods it. You pass the
voiced tension and the sensing role as natural-language input; you get back the
record (the tension in hand, the situating tensions it was weighed against, the
action taken, any handoff id, and notes), with every element carrying the id
needed to act on it next.

If the processor agent is absent or unregistered, the workflow above still
stands as guidance you can follow by hand — the path degrades to guidance, and
no CLI command is broken by the agent's absence.

## Write boundary

This path performs only **operational tension writes** — `tension create`,
`tension update`, `tension discard` — which the Write-Safety Guardrail
leaves **ungated** by design: they are operational edits, not governance writes,
so they run without a confirmation gate and this path must not invent one.

It **never performs a proposal write**. It never creates, proposes, responds
to, or withdraws a proposal — the ready `ten_` id is handed to the Proposal
Drafting Path, and whether a tension *needs* a proposal (or whether the
practitioner is allowed to act) is a question for the Constraint Discovery Path. This path processes the tension; it does not rule on whether the
governance record permits an action, and it does not advise on governance craft
or coach Holacracy practice.

That proposal fence holds in layers: the processor agent's prompt is scoped to
the six tension leaves and forbids any `proposal` command, and the guardrail's
`PreToolUse` write gate (`plugin/hooks/glassfrog-write-gate.sh`) is the backstop
that would gate any proposal write regardless — in the target host, `PreToolUse`
hooks fire for a subagent's Bash calls too (the hook input carries an
`agent_id` inside a subagent call), so the gate reaches the processor.
