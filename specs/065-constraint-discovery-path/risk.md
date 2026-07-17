# Risk: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Round**: 1
**Generated**: 2026-07-18
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix in PROJECT.md)
**Degradation flags**: none — full upstream set present. PROJECT.md declares no Regulatory Context, so no IEC 14971 bridge is included.

> Domain note: this is a **read-only** path, so there is no direct write/clobber hazard. The dominant hazard class is **integrity of authority judgment** — a path whose whole purpose is to tell an operator what governs an action is most dangerous when it *rules* instead of *surfacing* (H-1) or *misattributes* whose authority is in play (H-2). Severity ratings weight decision-quality over availability. Two hazards corresponded directly to the analyze findings and were closed by the post-guard fixes: H-2 (own-vs-other) held the analyze **K5** scenario-coverage gap, now closed by the own-role authority scenario; H-7's drift control held the analyze **H3** enumeration drift, now resolved by naming `me roles` in plan ADR-2. Both are Green after the fixes.

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | The path computes a permission verdict from local logic, or fabricates a ruling when the record is unclear → the operator acts on an ungrounded "you may/may not" answer | spec § Non-Behaviors; plan ADR-4; CONSTITUTION VIII; VISION Exclusion 2 | High | Med | Red | RC-1, RC-2, RC-3, RC-4 | **Yellow** |
| H-2 | Own-vs-other misattribution — the `me roles` read is stale/failed/omitted → an action under the caller's own domain is framed as "another role's" (or vice versa), giving a wrong authority picture | interface § Surface (`owned_by_caller`/`me roles`); spec § Behavioral Accord (Characterization); analyze K5 | Med | Med | Yellow | RC-5, RC-6, RC-7 | **Green** |
| H-3 | Clarify-when-vague fails — the skill guesses a vague action instead of asking → the navigator traverses a wrong action and returns a confidently-wrong picture | spec § Behavioral Accord (Entry); plan ADR-3; interface § Interactions | Med | Low | Yellow | RC-8, RC-9 | **Green** |
| H-4 | Narrowing an over-broad result set *before* paging through it silently drops governing domains/policies → incomplete constraint picture | spec § Driving Scenarios (over-broad edge); CONSTITUTION VI; plan R6 | High | Med | Red | RC-10, RC-11, RC-12 | **Green** |
| H-5 | Misleading synthesis — the agent draws a wrong characterization from correct reads | spec § System Overview / Characterization; plan (agent judgment) | High | Med | Red | RC-3, RC-13 | **Yellow** |
| H-6 | Stale picture — reads reflect a moment in time; governance changed since | spec § Behavioral Accord (read traversal); PROJECT (API is SoT) | Med | Med | Yellow | RC-14, RC-15 | **Green** |
| H-7 | A composed read leaf is renamed/removed in the CLI → traversal breaks or the picture misleads | plan ADR-2; interface § Error Communication; analyze H3 | Med | Med | Yellow | RC-16, RC-17 | **Green** |
| H-8 | Agent not registered/discoverable, or the delegation degrades, untested → synthesis-via-isolation lost (raw dumps land in the caller) | interface § Surface / § Error Communication | Med | Low | Green | RC-18, RC-19 | **Green** |
| H-9 | The read path issues a `glassfrog` governance write via `Bash` (tool grant blocks `Write`/`Edit` but not `glassfrog` subcommands) | interface § Error Communication (governance-write nuance); plan ADR-4 | High | Low | Yellow | RC-20, RC-21, RC-22 | **Yellow** |
| H-10 | A wide traversal fans out many reads → `429` throttling for the whole organization | plan (traversal fan-out); CONSTITUTION X | Med | Low | Green | RC-23, RC-24 | **Green** |

No residual risk is **Red**. After the post-guard fixes (H-2 → Green via the own-role scenario; H-7's H3 caveat resolved), three **Yellow** residuals remain — H-1 (rules-vs-surfaces, the path's defining hazard), H-5 (synthesis judgment), and H-9 (external host contract + subagent hook reach) — all *inherent* and accepted with the justifications below.

---

## Hazard Detail

**H-1 — Local ruling / fabricated verdict.** *Severity High*: an ungrounded authority answer, delivered by a tool that reads authoritative, is exactly the No-Fabricated-Data (VIII) and Local-governance-logic (VISION Exclusion 2) failure — it misdirects governance action. *Probability Medium*: an LLM navigator asked "am I allowed?" is strongly tempted to answer rather than surface. Controls: **RC-1** the prompt-level surface-and-characterize-never-rule guardrail (plan ADR-4); **RC-2** the `characterization` field's first-class "the record does not clearly answer" value, so uncertainty has a legitimate output that isn't a guess; **RC-3** every element carries its id → the characterization is auditable against the raw domain/policy read; **RC-4** the "surface-not-rule" and "no-fabricated-ruling-under-uncertainty" validation scenarios pin the boundary. Residual **Yellow** — the guarantee is prompt-level (reading a policy is legitimate, so it cannot be tool-enforced); RC-1–RC-4 make ruling *visibly wrong* and give uncertainty an honest outlet, but cannot structurally prevent an LLM from over-asserting. Inherent to the feature and accepted.

**H-2 — Own-vs-other misattribution.** *Severity Medium* (a wrong own/other framing sends the operator to the wrong next step — self-act vs seek-permission), *Probability Medium* (`me roles` is one more read that can fail/stale, and the own-role branch is untested — analyze K5). Controls: **RC-5** the navigator reads `me roles` to mark `owned_by_caller` from the record (not a guess); **RC-6** the owning `role_id` is always surfaced with its id, so the operator can verify ownership independently of the tool's framing; **RC-7** the partial-failure handling surfaces a failed `me roles` read rather than silently assuming ownership. Residual **Green** *(after guard fix)* — RC-6 makes the attribution checkable, and the **analyze K5 gap is now closed**: the "An action under the caller's own role's domain is within their authority" scenario pins the `owned_by_caller = true` branch, moving RC-5/RC-7 from documented to verified. The `me roles` read can still fail at runtime, but that path is surfaced (RC-7), not silent.

**H-3 — Clarify-when-vague failure.** *Severity Medium* (traversing a guessed action yields a picture about the wrong thing), *Probability Low* (the clarify step is explicit and the too-vague scenario pins it). Controls: **RC-8** the clarify-when-vague step lives in the skill (caller context), which asks before delegating (plan ADR-3, interface Interactions); **RC-9** the too-vague scenario asserts the skill asks and does not traverse on a guess, and stops if the operator declines. Residual **Green** — the interaction is placed where a channel to the operator exists and is scenario-pinned.

**H-4 — Narrow-before-page silent truncation.** *Severity High* (an incomplete constraint picture is the silent-truncation failure VI exists to prevent), *Probability Medium* (large orgs return multi-page searches). Controls: **RC-10** the over-broad scenario *signals* narrowing ("narrowed — refine"); **RC-11** the navigator inherits the CLI's pagination via the orientation (062) dependency; **RC-12** the spec Discovery accord + interface + the over-broad scenario state narrowing chooses "most relevant" over the *paged-through* set. Residual **Green** — pinned in the artifacts from the outset (065 carried 064's post-guard fix forward rather than repeating the gap).

**H-5 — Misleading synthesis.** *Severity High*, *Probability Medium* (LLM synthesis can misweight correct data). Controls: **RC-3** every element carries its id → the characterization is auditable against the raw read; **RC-13** the no-fabrication scenarios keep the agent from inventing to fill gaps. Residual **Yellow** — synthesis is agent judgment; ids make it checkable but not error-proof. Inherent and accepted.

**H-6 — Stale picture.** *Severity Medium*, *Probability Medium*. Controls: **RC-14** ids let the caller re-read current state at any point; **RC-15** any downstream *write* acts through the proposal path, which carries optimistic-concurrency stale-write surfacing (054), so acting on a stale read is caught at write time. Residual **Green**.

**H-7 — Read-leaf drift.** *Severity Medium* (a broken read yields a partial/failed picture, surfaced not silent), *Probability Medium* (the CLI evolves). Controls: **RC-16** the best-effort `internal/build` drift guard pins the named leaves and fails the build when one leaves the CLI (plan ADR-2, tasks T002); **RC-17** the partial-failure scenario surfaces a failed read and returns the picture from the reads that succeeded. Residual **Green** — *the analyze H3 caveat is now resolved*: plan ADR-2's enumeration names `me roles` alongside the other leaves, so the plan, the single-sourced guard list, and the artifacts agree — the guard pins exactly what the artifacts compose.

**H-8 — Registration/degradation untested.** *Severity Medium* (degrades to skill-as-guidance — raw dumps then land in the caller; not an integrity loss), *Probability Low*. Controls: **RC-18** the interface documents the degraded-to-guidance fallback; **RC-19** the registration/discovery + missing-agent-degradation scenarios (present in the feature from the outset). Residual **Green**.

**H-9 — Governance write from a read path.** *Severity High* (a write from a read path violates IX), *Probability Low* — lower than 064's equivalent because 065 composes **no** write leaf at all; a governance write would require the navigator to invent a `proposal`/`tension` write command outside its listed reads. Controls: **RC-20** the `tools` grant withholds `Write`/`Edit` (blocks workspace mutation); **RC-21** the agent prompt scopes execution to the read leaves (the primary control); **RC-22** 063's landed `PreToolUse` `Bash` gate backstops any `glassfrog` write. Residual **Yellow** — same inherent host uncertainty as 064: whether 063's hook fires for a *subagent's* Bash is not settled by the plugin; if not, RC-21 (prompt scope) is load-bearing. Probability is Low (065 drives no write by design), so this stays Yellow, not Red.

**H-10 — Rate-limit amplification.** *Severity Medium*, *Probability Low*. Controls: **RC-23** relevance-bounded traversal (spec Assumption); **RC-24** a `429` mid-traversal is handled by the partial-picture path plus the CLI's `429` backoff (017/031), reached via orientation. Residual **Green**.

---

## Controls Index

| RC-ID | Control (assessment level) | Mitigates |
|---|---|---|
| RC-1 | Prompt-level surface-and-characterize-never-rule guardrail | H-1 |
| RC-2 | `characterization` has a first-class "record does not clearly answer" value (uncertainty ≠ a guessed verdict) | H-1 |
| RC-3 | Every picture element carries its id → characterization is auditable against the raw read | H-1, H-5 |
| RC-4 | "Surface-not-rule" + "no-fabricated-ruling" validation scenarios | H-1 |
| RC-5 | Navigator reads `me roles` to mark `owned_by_caller` from the record | H-2 |
| RC-6 | Owning `role_id` surfaced with its id → ownership independently verifiable | H-2 |
| RC-7 | Partial-failure handling surfaces a failed `me roles` read (no silent ownership assumption) | H-2 |
| RC-8 | Clarify-when-vague step in the skill (caller context) asks before delegating | H-3 |
| RC-9 | Too-vague scenario: skill asks, does not traverse on a guess, stops if declined | H-3 |
| RC-10 | Over-broad narrowing is signaled ("narrowed — refine"), never silent | H-4 |
| RC-11 | Navigator inherits the CLI's pagination via orientation (062) | H-4 |
| RC-12 | Narrowing chooses "most relevant" over the *paged-through* set (spec/interface/scenario) | H-4 |
| RC-13 | No-fabrication scenarios: empty/partial results never invent | H-5 |
| RC-14 | Ids let the caller re-read current state | H-6 |
| RC-15 | Downstream writes carry optimistic-concurrency stale-write surfacing (054) | H-6 |
| RC-16 | Best-effort `internal/build` drift guard pins the composed read leaves (incl. `me roles`) | H-7 |
| RC-17 | Partial-failure scenario surfaces a failed read, returns the succeeded reads | H-7, H-10 |
| RC-18 | Interface documents the degraded-to-guidance fallback | H-8 |
| RC-19 | Registration/discovery + missing-agent-degradation scenarios | H-8 |
| RC-20 | Agent `tools` grant withholds Write/Edit (no workspace mutation) | H-9 |
| RC-21 | Agent prompt scopes execution to the read leaves | H-9 |
| RC-22 | 063 `PreToolUse` `Bash` gate (landed) backstops proposal-writes — reach over a subagent's Bash uncertain | H-9 |
| RC-23 | Relevance-bounded traversal (no full-tree walk) | H-10 |
| RC-24 | `429` mid-traversal handled by partial-picture path + CLI 017/031 backoff | H-10 |

## Traceability Index

- **H-1** → spec.md § Non-Behaviors; plan.md § ADR-4; CONSTITUTION.md VIII; VISION Exclusion 2
- **H-2** → interface-spec.md § Surface (`owned_by_caller`/`me roles`); spec.md § Behavioral Accord (Characterization); analyze.md K5
- **H-3** → spec.md § Behavioral Accord (Entry); plan.md § ADR-3; interface-spec.md § Interactions
- **H-4** → spec.md § Driving Scenarios (over-broad edge); CONSTITUTION.md VI; plan.md § Risks
- **H-5** → spec.md § System Overview, § Behavioral Accord (Characterization)
- **H-6** → spec.md § Behavioral Accord; PROJECT.md (API is source of truth)
- **H-7** → plan.md § ADR-2; interface-spec.md § Error Communication; analyze.md H3
- **H-8** → interface-spec.md § Surface, § Error Communication
- **H-9** → interface-spec.md § Error Communication (governance-write nuance); plan.md § ADR-4
- **H-10** → plan.md (traversal fan-out); CONSTITUTION.md X
- **RC-1–RC-24** → grounded in spec Behavioral Accord/Non-Behaviors, plan ADR-1/2/3/4 + Risks, interface Surface/Interactions/Error Communication, tasks T001/T002

## Residual Risk Summary

10 hazards, 24 controls. **0 Red**, **3 Yellow** (H-1, H-5, H-9), **7 Green** (H-2, H-3, H-4, H-6, H-7, H-8, H-10) — *after the post-guard fixes*. The two actionable items from the initial run were closed: **H-2** (own-vs-other) went to Green when the own-role authority scenario pinned the `owned_by_caller = true` branch (analyze K5 fix), and **H-7**'s drift control was firmed up when plan ADR-2's enumeration was reconciled to name `me roles` (analyze H3 fix). The three remaining **Yellow** residuals are inherent and accepted: the defining hazard **H-1** (rule-instead-of-surface — mitigated to visibly-auditable-and-honest by RC-1–RC-4 but not structurally preventable, exactly as the "surface, not rule" discipline anticipates), **H-5** (synthesis is agent judgment — mitigated by every element carrying its auditable id), and **H-9** (the read-only guarantee leans on prompt scope, with 063's landed write hook a backstop of uncertain reach over the subagent). There is no unacceptable (Red) residual — the path is safe to build.
