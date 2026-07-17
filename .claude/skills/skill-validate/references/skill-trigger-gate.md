# Skill-deliverable trigger-precision gate

> Runs skill-creator's description-optimizer as an independent evaluator of a skill's
> invocation contract. Read this before running it: the naive reading of the output is
> wrong. The gate measures **precision** (does the description avoid firing on adjacent
> surfaces the spec excluded?) reliably, and **recall** (does it fire on everything it
> should?) unreliably. Apply decisions key on the held-out split only, and a
> kept-original is a pass.

## How to run it

The optimizer is `skill-creator`'s `run_loop.py`. It splits the eval set (~60% train /
~40% held-out test), evaluates the shipped description by running each query multiple
times (default 3) for a stable trigger rate, proposes rewrites, re-evaluates on both
splits, and iterates (default up to 4). It selects `best_description` by the
**held-out** score to avoid overfitting the train split.

```bash
python -m scripts.run_loop \
  --eval-set <path-to-trigger-eval.json> \
  --skill-path <path-to-skill> \
  --model <model-id-powering-this-session> \
  --max-iterations 4 \
  --verbose
```

Operational notes:
- Use the model id from the current session so triggering matches what users
  experience.
- Expect ~10 parallel `claude -p` workers writing self-cleaning temp files under
  `.claude/commands/` — they clean up as each query finishes.
- Keep the eval set and results under a gitignored path (`.context/skill-review/`) —
  they are evaluation scratch, not shipped artifacts.

## Reading the result — the part that matters

**Precision is the durable signal.** It measures false triggers: does the description
fire on near-miss negatives the spec excluded (adjacent CLI mechanics, "am I allowed
to X" authority questions, raw reads, write requests, sibling features)? High
precision with zero false triggers means the "don't fire on the wrong surface"
boundary holds — which is what a skill-deliverable's conformance actually depends on.

**Recall is harness-noise — do not act on it.** The triggering harness runs each query
in a bare `claude -p` sandbox with a synthetic slash-command and *no project CLI or
org data*. Non-actionable, exploratory queries under-trigger in that sandbox
regardless of wording, so recall reads low and — critically — **flat across every
description, including the pushier rewrites.** Flat recall means the test cannot rank
descriptions on the positive side. It is a property of the sandbox, not a defect in
the description, and leaning keyword-heavy does not move it — it only risks eroding
precision.

Worked example (064 `governance-navigator`, opus-4-8, 4 iterations, 12 train / 8 test):

| | train | test (held-out) |
|---|---|---|
| Original (winner) | 6/12 | 4/8 |
| Rewrites (iter 2–4) | 6/12 | 4/8 |

Precision was 100% on both splits (zero false triggers across every near-miss
negative). Recall was ~6–8% and identical for every description. The loop kept the
original as `best_description`. **Correct outcome: change nothing — the gate passed.**

## The apply rule

- **Held-out win** → verify the winner still satisfies the spec-pinned requirements
  from interface-spec.md (e.g. "name + description only", states *when*, includes the
  synthesized-picture claim, names both hard boundaries), then apply it. Report
  before/after and scores.
- **No held-out win** (the common case; loop returns the original) → change nothing.
  Report it as a passing gate.
- **Never** apply a rewrite that wins only on train, or that is merely "pushier."
  Rewrites that don't generalize to held-out buy nothing and can erode precision.

## Agent / subagent descriptions

The optimizer handles skills only. A subagent description is a routing hint for the
host, not a user trigger, so the loop does not apply. Review it by reasoning instead:
does it name identity, input, output, the hard boundaries, and any delegation link
(e.g. "the X skill delegates traversal here")? Record the reasoned check; do not run
the optimizer against it.

## The caveat to carry into the summary

This gate catches **over-triggering only**. A green result means "the description does
not fire on adjacent surfaces the spec excluded" — not "triggering is fully
validated." Because recall is sandbox-noise, under-triggering is invisible here; if it
ever matters, it needs real invocation data, not this loop. Always state this so a
green gate is not misread as complete triggering coverage.
