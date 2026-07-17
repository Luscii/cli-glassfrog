# Risk: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Round**: 1
**Generated**: 2026-07-17
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix in PROJECT.md)
**Degradation flags**: none — full upstream set present. PROJECT.md declares no Regulatory Context, so no IEC 14971 bridge is included.

> Domain note: this is a **read-only** path, so there is no direct write/clobber hazard. The dominant hazard class is **integrity of understanding** — a picture that is *incomplete* (H-1), *misleading* (H-2), or *stale* (H-3) leads the agent or practitioner to act on a wrong model of governance. Severity ratings weight decision-quality over availability. Two hazards (H-1, H-6) corresponded directly to the checklist Observation on paging-before-narrowing and the analyze K5 scenario-coverage gap; both were closed by the post-guard fixes (RC-3, RC-14) and are now Green.

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Narrowing an over-broad result set *before* paging through it silently drops relevant roles/domains/policies → incomplete governance picture | spec § Driving Scenarios (over-broad edge); checklist Obs 2 (VI); plan R4 | High | Med | Red | RC-1, RC-2, RC-3 | **Green** |
| H-2 | Synthesis misrepresents governance — the agent draws a wrong conclusion from correct reads | spec § System Overview / Behavioral Accord (Synthesis); plan (agent judgment) | High | Med | Red | RC-4, RC-5 | **Yellow** |
| H-3 | Stale picture — reads reflect a moment in time; governance changed since (a proposal was accepted concurrently) | spec § Behavioral Accord (read traversal); PROJECT (API is SoT) | Med | Med | Yellow | RC-6, RC-7 | **Green** |
| H-4 | Prompt scope fails and/or 063's hook is absent → the navigator issues a `glassfrog` governance write via `Bash` from a "read" path (the tool grant blocks `Write`/`Edit` but cannot restrict `glassfrog` subcommands) | interface § Error Communication (governance-write nuance); plan ADR-5, R2 | High | Low | Yellow | RC-8, RC-9, RC-10 | **Yellow** |
| H-5 | A composed read leaf is renamed/removed in the CLI → traversal breaks or the picture misleads | plan ADR-4; interface § Error Communication; analyze | Med | Med | Yellow | RC-11, RC-12 | **Green** |
| H-6 | Agent not registered/discoverable, or the delegation degrades, untested → synthesis-via-isolation lost (raw dumps land in the caller) | analyze K5; interface § Surface / § Error Communication | Med | Med | Yellow | RC-13, RC-14 | **Green** |
| H-7 | A wide traversal fans out many reads → `429` throttling for the whole organization | plan (traversal fan-out); CONSTITUTION X | Med | Low | Green | RC-15, RC-16 | **Green** |
| H-8 | The path answers "can I do X?" itself instead of deferring to Constraint Discovery (065) → a surfacing tool issues a wrong authority verdict | spec § Non-Behaviors / edge scenario; plan ADR-5 | Med | Low | Green | RC-17, RC-18 | **Green** |

No residual risk is **Red**. After the post-guard fixes, H-1 (RC-3: paging-before-narrowing now pinned in spec/interface/scenario) and H-6 (RC-14: registration/discovery + degradation scenarios added) are mitigated to **Green**. Two **Yellow** residuals remain — H-2 (synthesis judgment) and H-4 (external host contract + 063-presence dependency) — both *inherent* and accepted with the justifications below.

---

## Hazard Detail

**H-1 — Narrow-before-page silent truncation.** *Severity High*: an incomplete picture is a false model of governance the agent acts on — the silent-truncation failure Constitution VI exists to prevent. *Probability Medium*: large orgs return multi-page searches/trees, and neither spec nor interface states that narrowing operates over the fully-paged set. Controls: **RC-1** the over-broad scenario *signals* narrowing ("narrowed — refine") rather than dropping silently; **RC-2** the navigator inherits the CLI's pagination via the orientation dependency (062); **RC-3** the skill/agent state that narrowing chooses "most relevant" over the paged-through result set. Residual **Green** — RC-3 is now adopted (spec Traversal accord + interface Surface/Interactions + the over-broad scenario's page-through step pin it), so paging-before-narrowing is fixed in the artifacts, not left to the orientation dependency alone.

**H-2 — Misleading synthesis.** *Severity High* (a wrong conclusion misdirects governance work), *Probability Medium* (LLM synthesis can misweight correct data). Controls: **RC-4** every element carries its id, so the caller can drill into the raw read and verify the synthesis (spec accord; interface output shape) — the picture is *auditable*, not a black box; **RC-5** the no-fabrication scenarios keep the agent from inventing to fill gaps. Residual **Yellow** — synthesis is agent judgment; ids make it checkable but not error-proof. This is inherent to a synthesis feature and accepted.

**H-3 — Stale picture.** *Severity Medium*, *Probability Medium*. Controls: **RC-6** ids let the caller re-read the current state at any point; **RC-7** any downstream *write* acts through the proposal path, which carries optimistic-concurrency stale-write surfacing (054), so acting on a stale read is caught at write time, not silently applied. Residual **Green** — read staleness is inherent to any read and is caught where it matters (the write boundary).

**H-4 — Prompt-scope failure / 063 hook does not cover the subagent.** *Severity High* (a governance write from a read path violates IX), *Probability Low*. The tool grant only blocks *workspace* mutation (`Write`/`Edit`); a `glassfrog` write can still be issued via `Bash`, so the governance-write controls are primarily prompt scope and 063's hook, not the tool grant. Controls: **RC-8** the agent's `tools` grant withholds `Write`/`Edit` (blocks workspace mutation, not `glassfrog` subcommands); **RC-9** the agent prompt scopes execution to the read leaves (the primary governance-write control); **RC-10** 063's `PreToolUse` `Bash` gate — now landed (`plugin/hooks/glassfrog-write-gate.sh` + `gated-commands.txt`) — gates any `glassfrog` proposal-write `Bash` call and `allow()`s the navigator's reads ungated. Residual **Yellow** — with 063 landed the backstop concretely exists, **but** it is registered as a `PreToolUse` matcher on the *main* agent's `Bash`; whether the host also applies it to a **subagent's** `Bash` calls is a host-semantics question not settled by the plugin (see the new consideration below). If it does not, RC-10 does not cover the navigator and governance-write prevention rests on RC-9 (prompt scope) with RC-8 blocking only workspace writes. The prompt scope is the floor; the hook is a backstop of uncertain reach over the subagent.

**H-5 — Read-leaf drift.** *Severity Medium* (a broken read yields a partial/failed picture, surfaced not silent), *Probability Medium* (the CLI evolves). Controls: **RC-11** the best-effort `internal/build` drift guard pins the named leaves and fails the build when one leaves the CLI (plan ADR-4, tasks T002); **RC-12** the partial-failure scenario surfaces a failed read and returns the picture from the reads that succeeded, never inventing. Residual **Green** — build tripwire + runtime partial-failure handling compound.

**H-6 — Registration/degradation untested.** *Severity Medium* (the path degrades to skill-as-guidance — the raw dumps then land in the caller's context, losing the synthesis-via-isolation value; not an integrity loss), *Probability Medium* (first agent in the surface; no scenario covers registration/discovery — the analyze K5 gap). Controls: **RC-13** the interface documents the degraded-to-guidance fallback; **RC-14** the registration/discovery + missing-agent-degradation scenarios (the analyze K5 fix). Residual **Green** — RC-14 is now adopted ("The navigator is reachable once the plugin registers it" and "A missing navigator degrades the path to guidance" scenarios, un-`@wip`'d by T001), so the surface is verified rather than assumed.

**H-7 — Rate-limit amplification.** *Severity Medium* (org-wide throttling for the rate-limit window), *Probability Low*. Controls: **RC-15** relevance-bounded traversal (stop short of walking the whole tree — spec Assumption 3, plan); **RC-16** a `429` mid-traversal is handled by the partial-picture failure path plus the CLI's `429` backoff (017), reached via orientation. Residual **Green**.

**H-8 — Authority overreach.** *Severity Medium* (a surfacing tool issuing a permission verdict misleads and blurs the 064/065 boundary), *Probability Low*. Controls: **RC-17** the prompt-level guardrail hands "can I do X?" to the Constraint Discovery Path (065); **RC-18** the "surfacing not judging" validation scenario and the "authority question defers" edge scenario pin the boundary. Residual **Green**.

---

## Controls Index

| RC-ID | Control (assessment level) | Mitigates |
|---|---|---|
| RC-1 | Over-broad narrowing is signaled ("narrowed — refine"), never silent | H-1 |
| RC-2 | Navigator inherits the CLI's pagination via the orientation (062) dependency | H-1 |
| RC-3 | State in the skill/agent that narrowing chooses "most relevant" over the *paged-through* set (checklist Obs 2 fix) | H-1 |
| RC-4 | Every picture element carries its id → the synthesis is auditable against the raw read | H-2 |
| RC-5 | No-fabrication scenarios: empty/partial results never invent | H-2 |
| RC-6 | Ids let the caller re-read current state | H-3 |
| RC-7 | Downstream writes carry optimistic-concurrency stale-write surfacing (054) | H-3 |
| RC-8 | Agent `tools` grant withholds Write/Edit (no local mutation) | H-4 |
| RC-9 | Agent prompt scopes execution to the read leaves | H-4 |
| RC-10 | 063 `PreToolUse` `Bash` gate (landed) gates proposal-write calls, allows reads — backstop of uncertain reach over a *subagent's* Bash | H-4 |
| RC-11 | Best-effort `internal/build` drift guard pins the composed read leaves | H-5 |
| RC-12 | Partial-failure scenario surfaces a failed read, returns the succeeded reads | H-5, H-7 |
| RC-13 | Interface documents the degraded-to-guidance fallback when the agent is absent | H-6 |
| RC-14 | Add a registration/discovery + missing-agent-degradation scenario (analyze K5 fix) | H-6 |
| RC-15 | Relevance-bounded traversal (no full-tree walk) | H-7 |
| RC-16 | `429` mid-traversal handled by partial-picture path + CLI 017 backoff | H-7 |
| RC-17 | Prompt-level guardrail defers the authority verdict to 065 | H-8 |
| RC-18 | "Surfacing not judging" + "authority question defers" scenarios | H-8 |

## Traceability Index

- **H-1** → spec.md § Driving Scenarios (over-broad edge); checklist.md Observation 2 (Constitution VI); plan.md § Risks (R4)
- **H-2** → spec.md § System Overview, § Behavioral Accord (Synthesis)
- **H-3** → spec.md § Behavioral Accord; PROJECT.md (API is source of truth)
- **H-4** → interface-spec.md § Error Communication (governance-write nuance); plan.md § ADR-5, § Risks (R2)
- **H-5** → plan.md § ADR-4; interface-spec.md § Error Communication; analyze.md
- **H-6** → analyze.md K5; interface-spec.md § Surface, § Error Communication
- **H-7** → plan.md (traversal fan-out); CONSTITUTION.md X
- **H-8** → spec.md § Non-Behaviors, § Driving Scenarios (authority-question edge); plan.md § ADR-5
- **RC-1–RC-18** → grounded in spec Behavioral Accord/Non-Behaviors, plan ADR-3/4/5 + Risks, interface Surface/Error Communication, tasks T001/T002

## Residual Risk Summary

8 hazards, 18 controls. **0 Red**, **2 Yellow** (H-2, H-4), **6 Green** (H-1, H-3, H-5, H-6, H-7, H-8) — *after the post-guard fixes*. The two actionable residuals from the initial run, **H-1** (narrow-before-page, checklist Obs 2) and **H-6** (registration/degradation untested, analyze K5), were closed by adopting RC-3 and RC-14. The two remaining **Yellow** residuals are inherent and accepted: **H-2** (synthesis is agent judgment — mitigated by every element carrying its auditable id) and **H-4** (the read-only guarantee leans on prompt scope, with 063's now-landed write hook as a backstop of uncertain reach over the subagent). There is no unacceptable (Red) residual — the path is safe to build.

## Post-063-Landing Note (subagent hook coverage)

063's Write-Safety Guardrail implementation landed on `main` (PR #150) and is now on this branch: `plugin/hooks/hooks.json` registers a `PreToolUse` matcher on `Bash` running `glassfrog-write-gate.sh`, which gates the four `proposal` write leaves (`gated-commands.txt`) and `allow()`s everything else — including all of the navigator's reads. Two concrete impacts on 064:

- **Confirmed (positive)**: the RC-10 backstop is no longer hypothetical, and the navigator's composed reads pass the gate ungated (no friction) because they are not `proposal` writes.
- **New open question (feeds H-4)**: the hook is registered against the **main** agent's `Bash`. Whether a Claude Code plugin `PreToolUse` hook also fires for a **subagent's** `Bash` tool calls is not settled by the plugin definition. If it does *not*, the `governance-navigator` subagent's Bash calls bypass 063's gate, so 064's "must not write" rests on the agent prompt (RC-9) plus the `Write`/`Edit`-withheld tool grant (RC-8) — not on the 063 backstop. **Action for implementation (T001):** confirm subagent hook coverage against the target host; if uncovered, keep the navigator's prompt strictly read-only and treat RC-9 as the load-bearing control. This does not raise the residual above Yellow (Probability stays Low — the prompt scope holds by default), but it sharpens where the guarantee actually comes from.
