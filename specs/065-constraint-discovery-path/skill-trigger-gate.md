# Skill Trigger Gate: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Date**: 2026-07-18
**Gate**: Skipped — no trigger eval set exists for this spec
**Description decision**: Shipped description kept (no optimizer run)

## Why skipped

The trigger-precision gate needs a should-trigger / should-not-trigger eval set
produced independently of the implementing context (the scenarios stage emits
these for skill-deliverables). No such set exists for 065: the archived
define/shape context (`constraint-discovery-path`) carries none, and 064's
`trigger-eval.json` targets the governance-navigation skill, not this one.
Fabricating a set inside the validation pass — in the same session that authored
the description — would defeat the independent-evaluator purpose, so the gate
was skipped rather than self-graded.

## What this means

- **Not verified empirically**: that `constraint-discovery`'s description avoids
  firing on adjacent surfaces (064's tension-work, 062's CLI mechanics, raw
  reads, write requests). The description carries explicit negative wording for
  both siblings, and 064's gate showed 100% precision for that pattern, but 065
  has no measured number.
- **Follow-up**: build a trigger eval set in an independent context (~20 queries,
  positives from the spec's authority questions; near-miss negatives from 064/062
  surfaces and write requests), keep it under `.context/skill-review/`, and run
  skill-creator's `run_loop.py`. Apply a rewrite only on a held-out win.
- Even with a green gate this would verify **over-triggering only** — recall is
  harness-noise in the bare sandbox.

## Agent description (reasoned review — optimizer out of scope for agents)

`constraint-navigator`'s description is a host routing hint, reviewed by
reasoning per the gate reference: it names identity (read-only constraint
navigator), input (a well-formed wanted action), output (synthesized picture
characterizing the authority situation, every element id-carrying), the hard
boundaries (never writes; never computes a permission verdict from local
rules), and the delegation link ("The constraint-discovery skill delegates
traversal here"). All five elements present — pass.
