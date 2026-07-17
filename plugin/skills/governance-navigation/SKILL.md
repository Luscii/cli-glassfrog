---
name: governance-navigation
description: Understand the governance around a tension by traversing the roles, fillers, domains, and policies that bear on a free-form concern, returning a synthesized picture rather than raw command dumps. Reach for this whenever a practitioner voices a concern and you need to see which roles it touches, who fills them, and the governing domains and policies — before deciding what to do. It is not about how to drive the CLI mechanically (that is orientation), and it does not judge whether an action is allowed (that is the Constraint Discovery Path). Use it to go from a concern to the governance that shapes it.
---

# Navigating the governance around a concern

This skill is the first **operator path** on the agent operating surface. Its job
is to take a practitioner's free-form concern — usually a **tension**, sensed but
not yet located in the structure — and turn it into a **synthesized picture** of
the governance around it: the relevant roles, who fills them, and the domains and
policies that shape them, each element carrying the id that reads it again.

It composes only reads the `glassfrog` CLI already exposes. It adds no command,
flag, or capability of its own, and it **never writes** — capturing the tension
is the Tension Processing Path's job, and ruling on whether an action is allowed
is the Constraint Discovery Path's job (see [Boundaries](#boundaries)).

## When to reach for it

Reach for governance navigation when you are **working a tension** and need to
understand the governance around it before acting:

- A practitioner voices a concern with **no role in hand** — you need to discover
  which roles, fillers, domains, and policies it touches.
- You have a role and want the **fuller picture** around it: who fills it, what
  domains it controls, and what policies bear on the concern.
- A concern reaches into a **circle** and you need to follow it into the relevant
  sub-roles without walking the whole tree.

Do **not** reach for it to learn *how to drive the CLI* — output formats,
pagination, exit codes, credentials, write-safety mechanics live in the
**orientation** skill. And do **not** use it to answer *"am I allowed to do X?"* —
that authority verdict belongs to the **Constraint Discovery Path** (065). This
path only surfaces the governing governance; it does not judge it.

## The workflow

This is the single source of the traversal steps — the `governance-navigator`
agent runs exactly this workflow; it does not carry a second copy. From a concern:

1. **Search** — run `search` on the concern to discover what governance it
   touches across every model, since the practitioner usually has no role id in
   hand. This is the entry point when nothing is known yet.
2. **Read the roles** — resolve the matched role ids into `roles [id]` (and use
   `tree [id]` when you need a role's structural context within its circle).
3. **See who fills them** — `fillers <role-id>` for the actors filling a role;
   for a circle concern, `subrole-actors <role-id>` rolls up who fills its direct
   sub-roles. Follow into sub-roles only **as far as the concern warrants** —
   stop short of walking the whole tree.
4. **Draw in the governing governance** — `domains <role-id>` for the domains a
   relevant role controls, and `policies <role-id>` for the policies on its
   interior, keeping only those that bear on the concern.
5. **Synthesize** — draw the results together into one picture (roles, fillers,
   domains, policies), each element carrying its id. Present the synthesized
   picture, **not** a concatenation of raw command output.

Traversal is **bounded by relevance**, not a full-tree walk: at each step, narrow
to what actually touches the concern. Where a `search` or a list read spans more
than one page, **page through the full result set first** (the default walk does
this; see the orientation pagination guidance) and *then* narrow to what is most
relevant — narrowing is a choice over the complete set, never a silent
single-page cap. When the picture is narrowed, say so, so the practitioner can
refine the concern.

## Delegation

Run the **`governance-navigator`** agent to execute this workflow. It performs the
traversal in its **own isolated context** and returns **only** the synthesized
picture — so the raw `search`/`roles`/`fillers`/`domains`/`policies` output stays
out of your context and never floods it. You pass the concern as natural-language
input; you get back the picture (roles, fillers, domains, policies, and any
narrowing or failure notes), with every element carrying the id needed to act on
it next — reading it again, or feeding it into the Constraint Discovery (065) or
Tension Processing (066) paths.

If the navigator agent is absent or unregistered, this workflow above still stands
as guidance you can follow by hand — the path degrades to guidance, and no CLI
command is broken by the agent's absence.

## Boundaries

- **Read-only.** This path only reads and only *surfaces* governance. It contains
  no write, confirm, or gate step. Capturing the tension into the record is the
  **Tension Processing Path**'s job (066), not this one.
- **Surfacing, not judging.** When a concern is really an authority question
  ("may I do X?"), surface the governing domains and policies and **defer the
  verdict to the Constraint Discovery Path** (065). Never rule on whether an
  action is permitted.
- **Mechanics live elsewhere.** For output formats, pagination, exit codes,
  credentials, and write-safety, consult the **orientation** skill rather than
  restating them here. For the exact flags of any one command, ask the CLI:
  `glassfrog <command> --help`.
