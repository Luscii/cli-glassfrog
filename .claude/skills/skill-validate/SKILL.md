---
name: skill-validate
description: Validates a Claude Skill deliverable before PR — runs the trigger-precision gate (skill-creator's description-optimizer) as independent verification, applies a description only if it wins on the held-out split, then hands off to score:validate for conformance. Use this instead of /score:validate whenever the thing being validated is itself a Claude Skill (a produced SKILL.md), when the user wants to check a skill's description won't over-trigger on adjacent surfaces, or when running post-implementation validation on a skill-deliverable.
---

# Skill-Validate

`skill-validate` is the skill-deliverable variant of Score's `validate`. When the
thing you built is *itself a Claude Skill* — a `SKILL.md` with a triggering
description — plain `validate` inspects whether the description exists as written, but
not whether it *behaves*: whether it fires on the right surface and stays quiet on the
adjacent ones the spec excluded. This wrapper adds that empirical check, then defers
to `validate` for the rest of conformance.

It is a workflow wrapper in the same family as `guard`, `define`, and `shape`: it runs
one gate, then loads and follows the next skill. It does **not** re-implement
`validate` — conformance inspection stays `validate`'s job.

**Order is deliberate: gate first, then validate.** The gate may *apply* an improved
description, and `validate` should bless the final shipped state, not a pre-mutation
one.

**When to use plain `/score:validate` instead:** the deliverable is ordinary code, not
a skill. This wrapper is only for skill-deliverables.

---

## Process

### Step 1: Confirm the deliverable is a Claude Skill

Identify the target spec (accept a spec reference as an argument, same as `validate`).
Confirm the implementation produced a `SKILL.md`, and that interface-spec.md (if
present) defines a triggering/invocation surface.

If it is not a skill-deliverable: "This is the skill-deliverable validator. For
ordinary code, run `/score:validate` directly." Stop.

### Step 2: Locate the eval set and baseline

The gate needs a trigger eval set (should-trigger / should-not-trigger queries) and a
baseline (no-skill or the previous version). The eval set should come from the
`scenarios` stage for skill-deliverables; keep it and results under a gitignored path
(`.context/skill-review/`).

- If an eval set exists: proceed to Step 3.
- If none exists: "No trigger eval set found — the gate needs should-trigger /
  should-not-trigger queries (the `scenarios` stage should emit these for
  skill-deliverables). I can skip the gate and run `/score:validate` on inspection
  alone, or help build an eval set first." Do **not** fabricate an eval set inline as
  part of a validation pass — a hand-made set inside the gate defeats the
  independent-evaluator purpose.

### Step 3: Run the trigger-precision gate

Load `references/skill-trigger-gate.md` before running — it covers how to run the
optimizer and, more importantly, how to read the result. The naive reading ("it
proposed a pushier description, apply it") is wrong here.

Run skill-creator's description-optimizer as an independent evaluator. Then:

- **Read precision, not recall.** Precision (no false triggers on the near-miss
  negatives the spec excluded) is the durable signal. Recall is harness-noise — the
  triggering sandbox runs bare `claude -p` with no project CLI/org data, so
  exploratory queries under-trigger regardless of wording, and recall reads flat
  across every description including the rewrites. Do not act on recall.
- **Apply rule — held-out win only.** If a proposal beats the shipped description on
  the *held-out (test)* split, verify the winner still satisfies the spec-pinned
  requirements from interface-spec.md, then apply it and report before/after + scores.
  If nothing beats the shipped description on held-out (the common case — the loop
  returns the original as `best_description`), **that is a passing gate, not a null
  result.** Change nothing. Never apply a train-only or merely-pushier rewrite; it
  buys nothing and can erode precision.
- **Agent/subagent descriptions are out of scope.** The optimizer handles skills only.
  A subagent description is a host routing hint, not a user trigger — review it by
  reasoning (identity / input / output / hard boundaries / delegation link), don't run
  the optimizer against it.
- **Record the outcome** and always state the over-triggering-only caveat: a green
  gate means "does not over-trigger on excluded surfaces," not "triggering is fully
  validated." Optionally write a short `skill-trigger-gate.md` into the spec directory
  so the numbers and the applied/kept decision are durable alongside `validate.md`.

### Step 4: Hand off to score:validate

Invoke `/score:validate` against the same spec (via the Skill tool). This is the
wrapper's composition step — `validate` runs its five conformance dimensions and
@validation scenarios against the **final** (possibly updated) SKILL.md, and produces
`validate.md` with the verdict. Do not duplicate that inspection here.

### Step 5: Combined summary

Present both parts together:
1. **Trigger gate**: precision (the signal that matters), the held-out result,
   applied-or-kept decision with before/after if applied, and the
   over-triggering-only caveat. If skipped/unavailable, say so.
2. **Conformance**: `validate`'s verdict (Ready / Issues / Not Ready) and its findings.

---

## Non-behaviors (Never Do)

- Do not apply a proposed description that fails to beat the shipped one on the
  held-out split — a train-split or "it's pushier" win is not a result.
- Do not treat a kept-original gate as a failure or a null result — it is a passing
  precision check.
- Do not run the description-optimizer against an agent/subagent description — reason
  about routing-hint descriptions instead.
- Do not read a green gate as full triggering coverage — it verifies over-triggering
  only (recall is harness-noise).
- Do not fabricate an eval set inside the validation pass — that defeats independent
  evaluation.
- Do not re-implement `validate`'s conformance inspection — defer to `/score:validate`.

---

## Reference Index

| File | What it contains |
|---|---|
| `references/skill-trigger-gate.md` | How to run skill-creator's description-optimizer, how to read precision (durable) vs recall (harness-noise), the held-out-win apply rule, the agent-description carve-out, and the over-triggering-only caveat. Load before Step 3. |
