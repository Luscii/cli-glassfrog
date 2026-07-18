# Risk: Tension Processing Path

**Feature**: 066-tension-processing-path
**Round**: 1
**Generated**: 2026-07-18
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix in PROJECT.md)
**Degradation flags**: none — full upstream set present. PROJECT.md declares no Regulatory Context, so no IEC 14971 bridge is included.

> Domain note: unlike its read-only sibling 064, this is a **write** path, so it carries a real write hazard class — a clobbered concurrent edit (H-3), a misplaced operational write (H-4), and, most acutely, the subagent **crossing into a proposal write** the Write-Safety Guardrail (063) exists to gate (H-2). Because the subagent is write-capable by design (063 leaves operational tension edits ungated), the proposal fence is *not* a tool-grant guarantee — it rests on prompt scope plus 063's hook, whose reach over a subagent's Bash is unsettled. H-2 is therefore the dominant, and only Yellow, residual. Severity ratings weight governance integrity over availability. Two hazards (H-1 paging, H-6 drift-invariant) tie to the checklist Observation and the ADR-5 disjointness guard; both are closed to Green.

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Judging a duplicate over an un-paged situating set silently misses a tension on a later page → a duplicate tension is recorded | spec § Situating / edge (duplicate); checklist Obs 2 (VI); plan R? | Med | Med | Yellow | RC-1, RC-2, RC-3 | **Green** |
| H-2 | Prompt scope fails and/or 063's hook does not reach the subagent → the processor issues a `glassfrog proposal` write from the tension path (the tool grant blocks `Write`/`Edit` but not `glassfrog` subcommands, and this subagent legitimately writes) | interface § Error Communication (write nuance); plan ADR-3, R2 | High | Low | Yellow | RC-4, RC-5, RC-6 | **Yellow** |
| H-3 | A `tension update` clobbers a concurrent edit (last-write-wins without `If-Match`) | CONSTITUTION X; interface § Error Communication (`If-Match`/`412` N/A here) | Med | Low | Green | RC-7 | **Green** |
| H-4 | The processor captures on the wrong sensing role, or misidentifies the "existing" duplicate, recording/editing the wrong record | spec § Behavioral Accord (Situating/Capture); plan (agent judgment) | Med | Med | Yellow | RC-8, RC-9 | **Green** |
| H-5 | A composed `tension` leaf is renamed/removed in the CLI → the workflow breaks or misleads | plan ADR-4/5; interface § Error Communication; analyze | Med | Med | Yellow | RC-10, RC-11 | **Green** |
| H-6 | A tension leaf drifts into 063's gated set (or a proposal leaf into 066's composed set) → 066's ungated writes start prompting, or a proposal write is wrongly executed as "operational" | plan ADR-3/ADR-5 (ungated invariant) | Med | Low | Green | RC-12 | **Green** |
| H-7 | Agent not registered/discoverable, or delegation degrades, untested → synthesis-via-isolation lost (raw dumps land in the caller) | analyze K5; interface § Surface / § Error Communication | Med | Med | Yellow | RC-13, RC-14 | **Green** |
| H-8 | The path judges "this needs a proposal" or "you're allowed" instead of deferring to Constraint Discovery (065) | spec § Non-Behaviors / validation scenario; plan ADR-3 | Med | Low | Green | RC-15, RC-16 | **Green** |
| H-9 | The path coaches Holacracy craft (interprets the tension, advises whether it is well-formed governance) — VISION Exclusion 1 | spec § Non-Behaviors / validation scenario | Low | Low | Green | RC-17 | **Green** |

No residual risk is **Red**. One **Yellow** residual remains — **H-2** (the proposal-write fence leans on prompt scope + 063's hook of uncertain subagent reach) — inherent to a write-capable subagent and accepted with the justification below. All other residuals are Green.

---

## Hazard Detail

**H-1 — Duplicate via judge-before-page.** *Severity Medium* (a duplicate tension clutters the record and can mislead a later reader, but is operational and correctable), *Probability Medium* (a busy role returns multi-page tension lists). Controls: **RC-1** the situating step pages through the full result set before judging duplicates; **RC-2** the processor inherits the CLI's pagination via the orientation dependency (062); **RC-3** the duplicate-surface scenario returns the existing tension by id rather than recording a second. Residual **Green** — RC-1 is now pinned in the spec Situating accord, the situating scenario ("will page through the full result set before judging duplicates"), and T001, closing the checklist Obs 2 gap.

**H-2 — Proposal-write escape (dominant).** *Severity High*: a `glassfrog proposal` write issued from the tension path is a governance write reached without the practitioner's explicit confirmation — exactly what 063 (and Constitution IX/XI) exist to prevent. *Probability Low*: the prompt scopes execution to the six tension leaves and forbids `proposal …`, and 063's hook backstops it. Unlike 064, the tool grant is **not** a control here — the subagent legitimately runs `glassfrog` writes via `Bash`, so nothing structurally blocks a `proposal` write; the fence is prompt + hook. Controls: **RC-4** the agent prompt scopes execution to the tension leaves and explicitly forbids any `proposal …` command (the load-bearing control); **RC-5** 063's `PreToolUse` `Bash` gate gates the four proposal-write leaves and passes tension writes through ungated; **RC-6** ADR-5's drift guard keeps the composed set disjoint from 063's gated set, so a proposal leaf can never silently enter 066's scope. Residual **Yellow** — RC-5's reach over a *subagent's* Bash is unsettled (see the note below); if it does not cover the subagent, the fence rests on RC-4 (prompt scope). Inherent to a write-capable subagent; accepted, with T001 confirming subagent hook coverage.

**H-3 — Update clobber.** *Severity Medium* (a clobbered concurrent tension edit loses a colleague's change), *Probability Low* (concurrent edits to the same tension are uncommon). Controls: **RC-7** the path composes the CLI's `tension update`, which carries the CLI's optimistic-concurrency / stale-write surfacing (Guarded Writes 053, Stale-Write Surfacing 054) — the path adds no update logic and does not bypass `If-Match`; interface marks `If-Match`/`412` handling N/A here (owned by the CLI). Residual **Green** — the concurrency control lives in the composed command, not reimplemented.

**H-4 — Wrong-record operational write.** *Severity Medium* (a misplaced or duplicate tension is operational noise, correctable by `tension update`/`discard`, not governance-structure damage), *Probability Medium* (agent judgment selects the sensing role and the duplicate match). Controls: **RC-8** the situating step surfaces existing tensions with their ids so the choice is made against the record, not blind; **RC-9** capture returns the created tension with its `ten_`/`sensing_role_id` so the caller can verify placement, and the no-fabrication scenarios keep the agent from inventing. Residual **Green** — auditable ids + a correctable, low-blast-radius operation (the API also enforces who may sense/edit).

**H-5 — Leaf drift.** *Severity Medium* (a broken tension command yields a partial/failed record, surfaced not silent), *Probability Medium* (the CLI evolves). Controls: **RC-10** the best-effort `internal/build` drift guard pins the composed leaves and fails the build when one leaves the CLI (plan ADR-5, tasks T002); **RC-11** the situating-failure scenario surfaces a failed read and returns the record from the reads that succeeded, never inventing. Residual **Green** — build tripwire + runtime partial-failure handling compound.

**H-6 — Ungated-invariant drift.** *Severity Medium* (a wrongly-gated tension write is friction; a wrongly-executed proposal write would be a guardrail bypass — but the guard catches both at build), *Probability Low* (requires an un-guarded edit to either registry). Controls: **RC-12** ADR-5's drift guard asserts the composed tension leaves are disjoint from 063's `gated-commands.txt` and fails the build on violation. Residual **Green** — the invariant H-2/H-3 depend on is pinned by a build tripwire.

**H-7 — Registration/degradation untested.** *Severity Medium* (the path degrades to skill-as-guidance — the situating reads then land raw in the caller's context, losing synthesis-via-isolation; not an integrity loss), *Probability Medium* (second agent in the surface; the analyze K5 coverage angle). Controls: **RC-13** the interface documents the degraded-to-guidance fallback; **RC-14** the registration/discovery + missing-agent-degradation scenarios, un-`@wip`'d by T001. Residual **Green** — the surface is verified rather than assumed.

**H-8 — Authority overreach.** *Severity Medium* (a processing tool issuing a proposal-need or permission verdict misleads and blurs the 066/065 boundary), *Probability Low*. Controls: **RC-15** the prompt-level guardrail hands "does this need a proposal / am I allowed?" to the Constraint Discovery Path (065); **RC-16** the "processing, not judging" validation scenario pins the boundary. Residual **Green**.

**H-9 — Coaching bleed.** *Severity Low* (advice on governance craft is out of scope but not harmful to the record), *Probability Low*. Controls: **RC-17** the prompt guardrail and the "not coaching" clause of the "processing, not judging or coaching" validation scenario keep the path to navigating the record, not facilitating (VISION Exclusion 1). Residual **Green**.

---

## Controls Index

| RC-ID | Control (assessment level) | Mitigates |
|---|---|---|
| RC-1 | Situating pages through the full result set before judging duplicates (checklist Obs 2 fix) | H-1 |
| RC-2 | Processor inherits the CLI's pagination via the orientation (062) dependency | H-1 |
| RC-3 | Duplicate-surface scenario returns the existing tension by id, never a second record | H-1 |
| RC-4 | Agent prompt scopes execution to the six tension leaves and forbids any `proposal …` command | H-2 |
| RC-5 | 063 `PreToolUse` `Bash` gate gates proposal-write calls, passes tension writes ungated — backstop of uncertain reach over a subagent's Bash | H-2 |
| RC-6 | ADR-5 drift guard keeps the composed set disjoint from 063's gated set (no proposal leaf can enter 066's scope) | H-2, H-6 |
| RC-7 | The composed `tension update` carries the CLI's optimistic-concurrency / stale-write surfacing (053/054); the path adds no update logic | H-3 |
| RC-8 | Situating surfaces existing tensions with ids so placement/duplicate choices are made against the record | H-4 |
| RC-9 | Capture returns the created tension with `ten_`/`sensing_role_id`; no-fabrication scenarios | H-4 |
| RC-10 | Best-effort `internal/build` drift guard pins the composed tension leaves | H-5 |
| RC-11 | Situating-failure scenario surfaces a failed read, returns the succeeded reads | H-5 |
| RC-12 | Drift guard asserts composed leaves disjoint from 063's gated set (build tripwire) | H-6 |
| RC-13 | Interface documents the degraded-to-guidance fallback when the agent is absent | H-7 |
| RC-14 | Registration/discovery + missing-agent-degradation scenarios (analyze K5 angle) | H-7 |
| RC-15 | Prompt-level guardrail defers the proposal-need / authority verdict to 065 | H-8 |
| RC-16 | "Processing, not judging" validation scenario | H-8 |
| RC-17 | Prompt guardrail + "not coaching" validation scenario clause | H-9 |

---

## Traceability Index

- **H-1** → spec.md § Situating (paging + duplicate); checklist.md Observation 2 (Constitution VI)
- **H-2** → interface-spec.md § Error Communication (write nuance); plan.md § ADR-3, § Risks (063-hook risk)
- **H-3** → CONSTITUTION.md X; interface-spec.md § Error Communication (`If-Match`/`412` N/A)
- **H-4** → spec.md § Behavioral Accord (Situating/Capture)
- **H-5** → plan.md § ADR-4/ADR-5; interface-spec.md § Error Communication; analyze.md
- **H-6** → plan.md § ADR-3, § ADR-5 (ungated invariant); § Risks
- **H-7** → analyze.md K5; interface-spec.md § Surface, § Error Communication
- **H-8** → spec.md § Non-Behaviors, § Validation Scenarios; plan.md § ADR-3
- **H-9** → spec.md § Non-Behaviors, § Validation Scenarios (VISION Exclusion 1)
- **RC-1–RC-17** → grounded in spec Behavioral Accord/Non-Behaviors, plan ADR-3/4/5 + Risks, interface Surface/Error Communication, tasks T001/T002

---

## Residual Risk Summary

9 hazards, 17 controls. **0 Red**, **1 Yellow** (H-2), **8 Green**. The two artifact-actionable items — **H-1** (judge-before-page, checklist Obs 2, closed by RC-1) and **H-6** (ungated-invariant drift, pinned by RC-12/RC-6) — are Green. The genuine write hazards new to this path — **H-3** (update clobber) and **H-4** (wrong-record write) — are Green because the concurrency control lives in the composed CLI command and a misplaced operational tension is auditable and correctable. The single **Yellow** residual, **H-2**, is inherent: a write-capable subagent's proposal fence cannot be a tool-grant guarantee, so it rests on prompt scope (RC-4) with 063's hook (RC-5) as a backstop of uncertain reach over the subagent. There is no unacceptable (Red) residual — the path is safe to build, with T001 tasked to confirm subagent hook coverage.

## Post-063-Landing Note (subagent hook coverage)

063's Write-Safety Guardrail landed on `main` (PR #150): `plugin/hooks/hooks.json` registers a `PreToolUse` matcher on `Bash` running `glassfrog-write-gate.sh`, which gates the four `proposal` write leaves (`gated-commands.txt`) and `allow()`s everything else — including all of 066's operational tension writes (they are not `proposal` writes). Two impacts on 066, sharper than for read-only 064 because this subagent legitimately writes:

- **Confirmed (positive)**: the RC-5 backstop concretely exists, and 066's tension writes pass the gate ungated (no friction) — the correct behaviour per 063's deliberate tension-edit carve-out.
- **Open question (feeds H-2)**: the hook is registered against the **main** agent's `Bash`. Whether it also fires for a **subagent's** `Bash` calls is not settled by the plugin definition. If it does *not*, the `tension-processor` subagent's `Bash` calls bypass 063's gate, so the "no proposal write" fence rests on RC-4 (prompt scope) alone. **Action for implementation (T001):** confirm subagent hook coverage against the target host; if uncovered, keep the processor's prompt strictly scoped to the tension leaves and treat RC-4 as the load-bearing control. This does not raise the residual above Yellow (Probability stays Low — the prompt scope holds by default), but it names where the guarantee actually comes from.
