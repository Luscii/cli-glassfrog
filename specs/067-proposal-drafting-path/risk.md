# Risk: Proposal Drafting Path

**Feature**: 067-proposal-drafting-path
**Round**: 1
**Date**: 2026-07-18
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Degradation flags**: none — full artifact set present. Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. No Regulatory Context in PROJECT.md — regulatory bridge omitted.

---

## Risk Register

| ID | Hazard | Source | Severity | Probability | Controls | Residual |
|---|---|---|---|---|---|---|
| H-1 | The gated create executes without human confirmation | spec § Staying within the operator layer; plan ADR-3; interface § Gated-write nuance | High | Low | RC-1, RC-2, RC-3 | **Yellow** (accepted with justification) |
| H-2 | The payload sent differs from the change set the practitioner was shown | plan ADR-3 consequences + Risk 2; interface § Confirmation contract | Medium | Low | RC-4, RC-5 | Green |
| H-3 | A rejected or failed create is misreported (fabricated `prp_` id or failure-as-success) | spec § Error scenarios; interface § Error Communication | High | Low | RC-6 | **Yellow** (accepted with justification) |
| H-4 | A duplicate draft is opened for a change already in flight | spec § Situating + edge case; plan Risk 5 | Low | Medium | RC-7 | Green |
| H-5 | An incomplete situating walk is treated as the complete picture | spec § Error scenarios; interface § Error Communication | Medium | Low | RC-8 | Green |
| H-6 | Boundary bleed: the path advances/withdraws/responds after the create (068/069 territory) | spec § Non-Behaviors; plan Risk 4 | Medium | Low | RC-9, RC-10 | Green |
| H-7 | Artifact/CLI drift: a named command changes, or the gate registry changes under the path | spec § Staying within the operator layer; plan ADR-5 + Risk 3 | Medium | Medium | RC-11 | Green |
| H-8 | The host does not honour subagents, tool grants, or hooks | plan Risk 6; interface § Error Communication | High | Low | RC-12, RC-3 | **Yellow** (accepted with justification) |
| H-9 | Skill and agent workflow prose diverge | plan Risk 1 | Low | Medium | RC-13 | Green |

**Unacceptable (Red) residual risks: none.**

---

## Hazard Detail

### H-1 — The gated create executes without human confirmation

**Description**: `proposal create` is the write the Write-Safety Guardrail exists to gate. If it ever executed unconfirmed — because the hook did not fire inside the subagent, because the create leaf left 063's gated registry, or because the agent routed the write around the gate — a governance draft would be created without the explicit human confirmation the design promises.
**Severity: High** — an unconfirmed governance write is exactly the failure class 063 was built to prevent; it breaks the operator-layer safety story for the whole write path (plan § System Architecture).
**Probability: Low** — 066 empirically confirmed Claude Code `PreToolUse` hooks fire inside subagent Bash calls (hook input carries `agent_id`); the create is in the shipped `gated-commands.txt` today; and the drift guard makes registry-side regression a build failure.
**Controls**:
- **RC-1**: 063's `PreToolUse` write gate interposes human confirmation on every `proposal create` Bash call, including the subagent's (empirically confirmed coverage).
- **RC-2**: The drift guard asserts `proposal create` **is a member of** 063's gated set — the create silently leaving the registry turns CI red (plan ADR-5).
- **RC-3**: The agent prompt scopes the write surface to the one create, run as a plain `glassfrog` Bash command the hook can see (no indirection), with the change set inline.
**Residual: High × Low = Yellow — accepted.** Justification: the two structural layers (hook + guard) cover the two credible causes; what remains is host-contract risk, tracked separately as H-8. A worst-case unconfirmed create still produces a `draft` — visible, withdrawable, and not yet governance — never an accepted change.

### H-2 — The payload sent differs from the one shown

**Description**: The agent narrates the assembled change set, then runs the create. Shell-quoting damage or an agent error could make the transmitted `--changes` differ from the narration — the practitioner confirms one thing, another is sent.
**Severity: Medium** — the wrong content lands in a *draft* proposal: wrong, but visible, server-validated, and withdrawable before circulation; not a silently applied governance change.
**Probability: Low** — the payload rides in a single inline argument; the confirmation prompt shows the actual command, not the narration.
**Controls**:
- **RC-4**: Inline `--changes` means 063's confirmation displays the exact payload being written — the structural layer confirms the *real* command even if the narration is wrong (plan ADR-3; interface § Confirmation contract).
- **RC-5**: The CLI's client-side floor (valid JSON array, non-empty, `type` per element — 055) rejects a mangled payload before any request, and the server validates the rest.
**Residual: Medium × Low = Green.**

### H-3 — A rejected create is misreported

**Description**: The create can fail as the `403` Premium refusal, a `404` unknown anchor, or a `422` rejected change set. If the agent fabricated a `prp_` id or reported failure as success, the caller would hand a phantom id to the Circulation Path (068).
**Severity: High** — a phantom `prp_` id propagates into the downstream write flow and every later step operates on a record that does not exist.
**Probability: Low** — the CLI's exit codes and error envelope surface non-2xx failures unambiguously (015/004, via orientation), and the no-fabrication contract is pinned by a driving scenario and the record shape (`action: none`, `draft` absent).
**Controls**:
- **RC-6**: The record contract makes failure explicit — `action: none`, no `draft` element, the failure named in `notes`; the scenario "A rejected create fabricates no id" pins it.
**Residual: High × Low = Yellow — accepted.** Justification: a single prompt-level contract backed by an executable scenario; the CLI layer beneath it (exit codes, error envelopes) is shipped and tested, and 068's own reads would fail fast on a phantom id.

### H-4 — A duplicate draft is opened

**Description**: Situating narrows by circle + `draft` status because `proposal list` offers no tension filter; a related draft in another circle (or in a non-draft status) can be missed, and a duplicate opened.
**Severity: Low** — a duplicate draft clutters the record but is visible and withdrawable; the server accepts it as legitimate.
**Probability: Medium** — the narrowing is genuinely partial (spec Assumption 5 accepts this).
**Controls**:
- **RC-7**: Full-walk situating over the in-flight set before any create, surfacing matches with their ids so the create is a deliberate addition (spec § Situating; scenario "A matching in-flight draft is surfaced instead of duplicated").
**Residual: Low × Medium = Green.** The limitation is stated in the artifacts rather than hidden — claiming a tension-scoped check would invent surface.

### H-5 — An incomplete walk treated as complete

**Description**: If the situating list fails mid-walk and the partial result were treated as the whole, the duplicate judgment would run on a false picture.
**Severity: Medium** — a missed in-flight draft leads to H-4's outcome with higher confidence attached.
**Probability: Low** — 056 already flags a partial walk as incomplete with its cause; the agent's contract is to relay the flag, pinned by a scenario.
**Controls**:
- **RC-8**: The relayed incomplete flag + the "failed situating walk yields a partial picture" scenario — partial is always labeled partial.
**Residual: Medium × Low = Green.**

### H-6 — Boundary bleed into circulation

**Description**: After a successful create it is tempting to advance the draft (`proposal propose`) — 068's write.
**Severity: Medium** — an unwanted circulation start is a real governance event, but it is itself gated by 063, so a human sees it before it executes.
**Probability: Low** — the prompt fence names the forbidden leaves; the validation scenario asserts absence.
**Controls**:
- **RC-9**: Prompt fence — the agent's one write is `proposal create`; `propose`/`respond`/`withdraw` are named as forbidden (interface § Identity & scope).
- **RC-10**: 063 gates every proposal-write leaf regardless — the backstop confirmation would surface any bleed to the human.
**Residual: Medium × Low = Green.**

### H-7 — Artifact/CLI drift

**Description**: The artifacts name four command leaves and depend on one registry invariant. A CLI rename/removal, or a gate-registry edit, would silently falsify the guidance.
**Severity: Medium** — an agent following stale guidance mis-drives the CLI or (registry side) loses the confirmation property (that branch escalates to H-1).
**Probability: Medium** — surfaces do change across releases; this is the failure mode every sibling guarded.
**Controls**:
- **RC-11**: The best-effort `internal/build` drift guard — composed leaves exist in the registry; `proposal create` ∈ gated set; reads ∉ gated set; all sides source-derived (plan ADR-5).
**Residual: Medium × Medium → Green after control** — the guard converts silent drift into a build failure; what it deliberately does not cover (flags, prose, parser) is stated in the test, not silent.

### H-8 — Host does not honour subagents, grants, or hooks

**Description**: Isolation, the fenced grant, and the in-subagent gate all assume a capable host. On a host without hook support, RC-1 is absent entirely and the write could execute unconfirmed (H-1's worst branch); without subagent support, the path degrades to caller-context guidance.
**Severity: High** — same failure class as H-1 when hooks are absent while writes still execute.
**Probability: Low** — the target host is Claude Code, where coverage is confirmed; distribution/host targeting is owned by Operating-Surface Packaging (#70).
**Controls**:
- **RC-12**: Documented degradation — without the agent, the skill remains guidance only; the gated-write note tells any operator that the create requires the confirmed flow.
- **RC-3**: (shared) the prompt-level narration + inline payload keeps the presentation layer even where the structural layer is missing.
**Residual: High × Low = Yellow — accepted.** Justification: the host contract is external to this feature; #70 owns targeting capable hosts, and the prompt layer plus the CLI's own non-interactive safety (055's validation floor, server permissions) bound the damage to a visible, withdrawable draft.

### H-9 — Skill/agent prose drift

**Description**: Two artifacts describe one workflow; unsynchronized edits could diverge them.
**Severity: Low** — inconsistent guidance, caught at review or by confused output; no write-safety impact.
**Probability: Medium** — plain prose, no automated guard covers it.
**Controls**:
- **RC-13**: The workflow is single-sourced in the skill and referenced by the agent (plan ADR-1); review carries what the guard cannot.
**Residual: Low × Medium = Green.**

---

## Residual Risk Summary

No Red residuals. Three Yellow residuals — H-1 (unconfirmed create), H-3 (misreported create), H-8 (host contract) — each accepted with documented justification; all three share the same bounding property: the worst-case output is a *visible, withdrawable draft*, never an accepted governance change, because acceptance requires the separate circulation flow (068) with its own gated writes. Six Green.

---

## Traceability Index

| Hazard | Source section |
|---|---|
| H-1 | spec § Staying within the operator layer; plan ADR-3; interface § Gated-write nuance |
| H-2 | plan ADR-3 Consequences + Risks (inline quoting); interface § Confirmation contract |
| H-3 | spec § Driving Scenarios (A create is rejected); interface § Error Communication |
| H-4 | spec § Situating + edge case (matching draft); spec Assumption 5; plan Risks (circle-scoped check) |
| H-5 | spec § Driving Scenarios (A situating read fails); interface § Error Communication |
| H-6 | spec § Non-Behaviors (no advance/withdraw/respond); plan Risks (boundary bleed) |
| H-7 | spec § Staying within the operator layer; plan ADR-5 + Risks (gate-registry drift) |
| H-8 | plan Risks (host support); interface § Error Communication (degradation rows) |
| H-9 | plan Risks (workflow drift) |

| Control | Architectural grounding |
|---|---|
| RC-1 | 063 `PreToolUse` gate (`plugin/hooks/glassfrog-write-gate.sh`), subagent coverage confirmed by 066 |
| RC-2 | Drift guard gated-membership assertion (plan ADR-5; T002) |
| RC-3 | Agent prompt: single write surface, inline payload, no indirection (plan ADR-3; T001) |
| RC-4 | Inline `--changes` → confirmation displays the exact payload (plan ADR-3) |
| RC-5 | 055's client-side change-set floor + server validation (shipped CLI) |
| RC-6 | Draft-record contract: `action: none`, no fabricated `draft` (interface § Output contract) |
| RC-7 | Full-walk situating before create (spec § Situating; T001 acceptance criterion) |
| RC-8 | 056's incomplete flag, relayed (interface § Error Communication) |
| RC-9 | Prompt fence naming the forbidden proposal-write leaves (T001) |
| RC-10 | 063 gates all proposal-write leaves as backstop (shipped) |
| RC-11 | `internal/build` drift guard, all sides source-derived (T002) |
| RC-12 | Documented degradation-to-guidance (T001; interface § Error Communication) |
| RC-13 | Single-sourced workflow, agent references the skill (plan ADR-1; T001) |

---

## Scenario Coverage Note

Scenarios pre-exist this risk round (guard runs post-shape in this project). Spot-check: H-1 → "The path routes its one write through the guardrail" + "An unconfirmed create leaves the record untouched"; H-3 → "A rejected create fabricates no id"; H-4 → "A matching in-flight draft is surfaced instead of duplicated"; H-5 → "A failed situating walk yields a partial picture"; H-6 → "The path stops at the created draft"; H-7 → "The path names no command the CLI lacks"; H-8 → "A missing drafter degrades the path to guidance". H-2 and H-9 have no dedicated scenario — H-2's control is structural (the hook shows the real payload; the CLI floor rejects mangled JSON) and H-9's is a review concern; neither is scenario-testable as declarative prose. No test gap requiring action.
