---
name: constraint-discovery
description: Learn what governs an action a practitioner wants to take — whether it is within their own authority, falls under another role's domain, is shaped by a policy, or is a governance change that needs a proposal — by surfacing and characterizing the governing domains and policies from the glassfrog governance record, returned as a synthesized picture. Reach for this when the question is "what constrains this wanted action?" or "is this mine to do?". It is not for understanding the governance around a tension or seeing who fills a role (that is governance-navigation), and not for how to drive the CLI mechanically (that is orientation). It surfaces what the record says; it never computes a permission verdict.
---

# Discovering what constrains a wanted action

This skill is the second **operator path** on the agent operating surface — the
mirror of governance navigation. Its job is to take an action a practitioner
wants to take and turn it into a **synthesized picture** of what governs it:
the governing domains, the binding policies, the role(s) that hold them, and a
**characterization** of the authority situation drawn from the governance
record — each element carrying the id that reads it again.

It composes only the reads the `glassfrog` CLI already exposes. It adds no
command, flag, or capability of its own, it **never writes**, and it never
computes a permission verdict from local rules — it surfaces and characterizes
what the record says, and when the record does not clearly answer, it says so
rather than guessing.

## When to reach for it

Reach for constraint discovery when a practitioner wants to **do** something
and you need to know what constrains it before acting:

- Is the action **within the operator's own authority** — governed by a domain
  that a role the caller fills holds?
- Does it fall under **another role's domain**, so it needs that role's
  permission or a proposal?
- Is it **shaped by a policy** that grants or limits it?
- Is it a **change to governance structure**, which goes through a proposal?

Do **not** reach for it to work a tension or to see which roles a concern
touches and who fills them — that is the **governance-navigation** skill's job.
Do **not** reach for it to learn *how to drive the CLI* — output formats,
pagination, exit codes, credentials, and write-safety mechanics live in the
**orientation** skill.

## The workflow

This is the single source of the traversal steps — the `constraint-navigator`
agent runs exactly this workflow (steps 2–6); it does not carry a second copy.
Step 1, the clarify branch, lives here in the skill and only here.

1. **Clarify when the action is too vague** (in the skill, caller context —
   see the next section). Only a well-formed, searchable action goes past this
   step.
2. **Search** — run `search` on the wanted action to discover the domains,
   policies, and roles it touches across every model. This is the entry point
   when nothing is known yet.
3. **Read the owning roles** — resolve the matched ids into `roles [id]` (and
   `tree [id]` for a role's structural context within its circle).
4. **Draw in the governing governance** — `domains <role-id>` for the domains
   a role controls, `policies <role-id>` for the policies on its interior (and
   `policy <pol-id>` when the search matches a policy directly), keeping only
   what bears on the action.
5. **Tell your own from another's** — read the caller's own roles with
   `me roles` to learn whether a governing domain is held by a role the caller
   fills. `me roles` does **not** follow pagination and can return an
   incomplete list (it emits an incompleteness note on stderr): when it
   signals incompleteness, mark the own-role finding **uncertain** — never a
   definite "not yours".
6. **Characterize and synthesize** — draw the results together into one
   picture and characterize the authority situation from the record (see the
   agent's output contract): within the caller's own authority, under another
   role's domain, shaped by a policy, a governance change that needs a
   proposal, or — when the match is ambiguous — the record does not clearly
   answer.

Where a **paging-capable** read (`search`, `roles`, `domains`, `policies`)
spans more than one page, **page through the full result set first** (the
default walk does this; see the orientation pagination guidance) and *then*
narrow to the most relevant governing constraints — narrowing is a choice over
the complete set, never a silent single-page cap. When the picture is
narrowed, say so, so the practitioner can refine the action.

## Clarify when the action is too vague

When the wanted action is described too vaguely to search for the governance
that would constrain it, **this skill — not the agent — asks the operator to
sharpen it**, via the host's structured ask mechanism (an
`AskUserQuestion`-style prompt), **before delegating**. It never guesses a
meaning and traverses on the guess. If the operator declines to sharpen the
action, the path stops here — no traversal runs and no action is fabricated;
the `constraint-navigator` is never invoked on a guess. Interaction lives in
the skill because the agent runs isolated and non-interactive; only a
well-formed action is delegated.

## Delegation

Run the **`constraint-navigator`** agent to execute the traversal (workflow
steps 2–6). It performs the reads in its **own isolated context** and returns
**only** the synthesized picture — so the raw `search`/`roles`/`domains`/
`policies`/`me roles` output stays out of your context and never floods it.
You pass the well-formed action as natural-language input; you get back the
picture (the governing domains and policies, the roles that hold them, the
characterization, and any narrowing, failure, or uncertainty notes), with
every element carrying the id needed to act on it next — reading it again,
finding whom to ask (`fillers <role-id>`, or the governance-navigation path
for the fuller picture of people around a role), or feeding the proposal paths
when a proposal is the next step.

If the navigator agent is absent or unregistered, the workflow above still
stands as guidance you can follow by hand — the path degrades to guidance, and
no CLI command is broken by the agent's absence.

## Boundaries

- **Read-only.** This path only reads. It contains no write, confirm, or gate
  step, and it names no write command. (065 is a read path and drives no
  write, so it does not depend on the write-safety `PreToolUse` gate (063);
  regardless, the navigator's prompt is strictly read-only.)
- **Surface and characterize, never rule.** The path surfaces the governing
  domains and policies **drawn from the record** and characterizes the
  authority situation as read there. It never computes a permission verdict
  from local rules, it reimplements no permission rules, and it never
  fabricates an authority ruling under uncertainty — when the record does not
  clearly answer, it says so and surfaces what it found.
- **Mechanics live elsewhere.** For output formats, pagination, exit codes,
  credentials, and write-safety, consult the **orientation** skill rather than
  restating them here. For the exact flags of any one command, ask the CLI:
  `glassfrog <command> --help`.
