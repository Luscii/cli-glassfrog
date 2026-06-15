# Risk: Withdraw Proposal

**Feature**: 059-withdraw-proposal
**Round**: 1
**Date**: 2026-06-15
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md, features/proposal-write-flow/withdraw-proposal.feature
**Acceptability matrix**: Default 3×3 traffic light

---

> ⚠ Using default risk acceptability matrix — PROJECT.md defines no Regulatory Context and no project-level risk acceptability matrix.

This is a pre-implementation risk assessment (no code exists yet for this command, though all the machinery it consumes — the `proposal` group/model/render and the near-identical `propose` leaf — is landed on `main`). Severity and probability are grounded in the plan's architecture and the interface accord, not in observed runtime behavior. The feature file was loaded to check whether each hazard's control has a behavioral assertion behind it; this is recorded per hazard, not as a separate test-gap section (Step 7 is re-run only).

This is a **destructive write** to live governance: a successful withdraw mutates a proposal's state (circulating → `draft`), **deletes all of its existing consent responses**, and clears its `proposed_at`/`response_deadline`. That destructiveness is what distinguishes 059's hazard surface from its `propose` mirror (057): where a wrong advance is reversible (by withdraw itself), a wrong **withdraw destroys cast consent responses that nothing restores** — re-proposing requires the circle to respond again. The hazard surface therefore centers on (a) withdrawing the *wrong* proposal and irreversibly discarding its responses, (b) reporting a refused/failed withdraw as success, and (c) the design choice to ship this destructive write with **no confirmation guard** (deferred to Write-Safety Guardrail). Unlike 057, there is **no sibling-coordination hazard** — every dependency is landed.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | A valid-but-wrong `prp_` id withdraws a proposal the operator did not intend — **irreversibly deleting its cast consent responses** and clearing its deadline, with no confirmation prompt to catch it | spec.md § Invocation / Integration Boundaries / Non-Behaviors (no confirmation); plan.md § ADR-2 | High | Low | Yellow | RC-1, RC-2, RC-3 | Yellow |
| H-2 | A `404` or `422` is folded into success (copying discard's `404`-as-success), reporting a refused/failed withdraw as a successful return-to-draft | plan.md § ADR-1; interface-cli.md § Error Communication; tasks.md T001/T002 (risk notes) | High | Low | Yellow | RC-4, RC-5, RC-6 | Green |
| H-3 | A client-side pre-check of `available_transitions` is added, forking server authority and acting on a stale snapshot (TOCTOU) | spec.md § Non-Behaviors; plan.md § ADR-1; interface-cli.md § Interactions | Medium | Low | Green | RC-7, RC-8 | Green |
| H-4 | The Premium `403` is given a bespoke "not available on your plan" message — fabricating a plan-gate classification the API did not structurally provide | plan.md § ADR-1; interface-cli.md § Error Communication; spec.md § Failure | Medium | Low | Green | RC-9, RC-10 | Green |
| H-5 | A `POST` `429` is auto-retried and double-fires the withdraw; a re-fire of a succeeded withdraw then surfaces a spurious `422` | plan.md § Cross-cutting (Non-idempotent retry) | Low | Low | Green | RC-11, RC-12 | Green |

---

## Hazard Details

### H-1: A valid-but-wrong `prp_` id withdraws the wrong proposal, destroying its responses

**Source**: spec.md § Invocation (one `prp_` positional, passed through unvalidated); § Integration Boundaries (a `200` deletes responses and clears timestamps server-side); § Non-Behaviors ("must not prompt for confirmation or require a `--force`/`--yes` flag"); plan.md § ADR-2 (destructive but unconfirmed).

**Description**: `withdraw` is an explicit write keyed off a single caller-supplied `prp_` id, passed through to the API unvalidated and with **no confirmation prompt and no `--force` flag** (a deliberate non-behavior for agent automation, per CONSTITUTION's conflict-resolution rule, captured in plan ADR-2). A wrong id that *happens to be valid, visible, and currently circulating with `withdraw` available* will genuinely withdraw the wrong proposal — and the server then **deletes that proposal's existing `proposal_responses`** and clears its `proposed_at`/`response_deadline`. Unlike a wrong `propose` (057 H-1), which is reversible *by withdraw*, a wrong `withdraw` has **no inverse that restores the deleted responses**: the circle members' cast `no_objection`/`bring_to_meeting` votes are gone, and re-proposing requires them to respond again.

**Severity**: High — the wrong-withdraw irreversibly destroys cast consent responses (governance work that members performed), forcing the circle to re-respond if the proposal is re-proposed. It mutates no governance *structure* (the proposal's `changes[]` are untouched and the proposal itself survives in `draft`), and the cost is recoverable by re-collection rather than permanent structural corruption — but destroyed votes are real data loss, and there is no second checkpoint (no prompt) to catch the mistake before it lands. Rated High to honestly reflect that withdraw is more dangerous than its `propose` mirror.

**Probability**: Low — a *successful* wrong-withdraw requires a narrow conjunction: the wrong id must be syntactically valid, visible to the caller, currently circulating (`proposed_outside_meeting`/`escalated`), and offer `withdraw`. A typo'd or stale id almost always fails at `404` (not found / not visible) or `422` (not circulating / `withdraw` not offered) — surfaced as failures (H-2 controls). The id is supplied with explicit write intent, not derived implicitly.

**Risk Level**: Yellow (High × Low).

**Controls**:
- **RC-1**: Explicit write intent — the command is a deliberate destructive write verb taking the id as a required positional; the constitution accepts "the command's existence is the intent" in lieu of a prompt, and the operator/agent chooses the id knowingly (spec § Invocation; CONSTITUTION IX + conflict-resolution; plan ADR-2).
- **RC-2**: The success echoes the resulting `draft` proposal — the `200` renders the full `Proposal` (its id, the reset `draft` status, the cleared deadline, the now-zeroed `response_summary`), so the operator/agent can immediately verify *which* proposal was withdrawn and detect a wrong-target after the fact (plan ADR-1; interface § Surface; feature "carries the cleared deadline and updated transitions").
- **RC-3**: Deferred operator-layer guard — plan ADR-2 names **Write-Safety Guardrail** as the capability that will add destructive-write confirmation across the write path centrally; until it lands, the residual is explicitly accepted (the agent-automation contract precludes an inline prompt here). The deletion is recoverable by re-collecting responses (the proposal survives in `draft`).

**Residual Risk**: Yellow — acceptable with documented justification. The irreversibility of the response deletion keeps severity at High, so controls (which reduce probability and aid after-the-fact detection) cannot drop it to Green. The no-confirmation design is a constitution-endorsed, deliberate non-behavior for agent automation, and the named future control (Write-Safety Guardrail, plan ADR-2) is where the second checkpoint will live. This is the developer's accepted trade-off — the uniform non-interactive write surface over an inline prompt that would break agent operation — not a spec defect. It does not block implementation, but it is the one hazard worth the developer's explicit sign-off.

### H-2: A `404`/`422` is folded into success (the inverse-of-discard trap)

**Source**: plan.md § ADR-1 ("it intercepts no status … the inverse of discard's `404`-as-success"); interface-cli.md § Error Communication (`404`/`422` present as failure rows); tasks.md T001/T002 risk notes.

**Description**: The closest *idempotency* template, the discard leaf (045), *folds its `404` into success* because a soft-delete has an idempotent end-state. A withdraw does not. If an implementer copies discard's `errors.As`-for-`404`-as-success divert — or widens it to `422` — then a refused transition (`422`, not allowed) or an unknown proposal (`404`) would be reported as a *successful return-to-draft*. That is a failure-reported-as-success, the exact anti-pattern CONSTITUTION III (Fail Safe, Not Silent) exists to prevent, and it is materially misleading on a governance write. The correct template is the landed `propose` leaf (`proposal_propose.go`), which intercepts no status.

**Severity**: High — a swallowed `422`/`404` tells the operator/agent a withdraw succeeded when nothing changed; a multi-step agent workflow would proceed (e.g. editing the "draft" and re-proposing) on a proposal that is still circulating or does not exist. No advisory, no error — a silent false success on live governance.

**Probability**: Low — the design pins "intercept no status" in plan ADR-1, the interface error table lists `404`/`422` as explicit failure rows, tasks T001/T002 carry it as a load-bearing risk ("copy `proposal_propose.go`, not `tension.go`'s discard path"), and a held-out `@validation` scenario asserts both are real failures. The residual probability is an implementer copying a looser predicate during the build.

**Risk Level**: Yellow (High × Low).

**Controls**:
- **RC-4**: No-status-interception design — every non-2xx routes through the shared `reportFailure`/`classifyClientError` chokepoint unchanged; there is no `errors.As`-for-status divert at all, mirroring the landed `propose` leaf (plan ADR-1; interface § Error Communication).
- **RC-5**: Pinned negative assertion — the `@validation` scenario "A 404 and a 422 are surfaced as real failures" verifies neither produces a success result and both exit non-zero, independently of the implementing agent (spec § Validation Scenarios; feature file).
- **RC-6**: Cross-spec precedent — the DECISIONS.md entries recording 057's inverse-of-discard stance (§395/§396) and 059's own ADR-2 entry carry the "treats `404`/`422` as real failures" rule, and 059 explicitly conforms to the landed `propose` template rather than the discard one.

**Residual Risk**: Green — with the design intercepting no status (RC-4), an executable held-out assertion that `404`/`422` fail loudly (RC-5), and the landed-twin precedent (RC-6), a fold-to-success regression is caught at validation time rather than in production.

### H-3: A client-side pre-check forks server authority and races a stale snapshot

**Source**: spec.md § Non-Behaviors ("must not pre-read the proposal to inspect `available_transitions`"); plan.md § ADR-1; interface-cli.md § Interactions ("no prior `GET`").

**Description**: `available_transitions` is surfaced by the reads (056), making it tempting to add a "is `withdraw` allowed?" pre-read before the POST "to be helpful." Doing so (a) forks the server's authority over the transition rule, (b) adds a round-trip, and (c) introduces a TOCTOU race — the proposal could change between the pre-read and the write, so a stale "withdraw allowed" snapshot green-lights a POST the server then refuses anyway (or a pre-check that wrongly *blocks* a valid withdraw).

**Severity**: Medium — the server remains authoritative (it still `422`s a disallowed transition), so a stale pre-check mostly costs a wasted read and a confusing client-side gate; it does not corrupt state. The drift is a correctness/efficiency regression and a forked source of truth, not a data hazard.

**Probability**: Low — the no-pre-check is an explicit spec non-behavior, pinned in plan ADR-1 and the interface, and a `@validation` scenario asserts exactly one request and no prior read.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-7**: Single-request design — the command issues exactly one bodyless `POST` and lets the server's `422` enforce; it builds no pre-read (plan ADR-1; interface § Interactions).
- **RC-8**: Held-out assertion — the `@validation` scenario "The withdraw issues one request and reads nothing first" verifies a single request to the withdraw transition and no prior `GET` (spec § Validation Scenarios; feature file).

**Residual Risk**: Green — the design issues one request and the held-out scenario pins it; the server stays the single authority for the transition rule.

### H-4: The Premium `403` is given a fabricated plan-gate message

**Source**: plan.md § ADR-1 ("the Premium `403` … generic, no plan-gate special-casing"); interface-cli.md § Error Communication; spec.md § Failure.

**Description**: The transition is Premium-gated, so a `403` is expected when async proposals are disabled. But the `403` is a generic RFC 9457 `ProblemDetails` with **no structured field** distinguishing a plan-gate from an ordinary permission denial. If the command adds a bespoke "not available on your plan" message, it fabricates a classification the API did not provide (a guess presented as fact — touching CONSTITUTION VIII, No Fabricated Data) and forks the plan-gate signal that belongs to the separate Plan-Limit Signalling capability.

**Severity**: Medium — a fabricated "it's your plan" message could misdirect an operator whose `403` was actually an ordinary permission denial (the two are indistinguishable in the `403` body), wasting troubleshooting on the wrong cause. Bounded to one command's diagnostic text; no state impact.

**Probability**: Low — the design explicitly routes the `403` through the shared classifier as a generic `PermissionError(4)` with no plan-aware branch, the interface table says so, and the feature scenario "A Premium-gated refusal surfaces plainly" asserts no plan-specific message.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-9**: Generic-`403` design — the `403` is a plain `PermissionError(4)` via the shared chain; the command adds no plan-aware branch and surfaces only the API's extracted detail (plan ADR-1; interface § Error Communication).
- **RC-10**: Behavioral assertion — the feature scenario "A Premium-gated refusal surfaces plainly" verifies the failure names the HTTP status with **no** "not available on your plan" message (feature file); the dedicated plan-gate signal is deferred to Plan-Limit Signalling.

**Residual Risk**: Green — no fabricated plan classification is produced; the `403` is handled uniformly with every other refusal, and the scenario pins the absence of a bespoke message.

### H-5: A `POST` `429` double-fires the withdraw

**Source**: plan.md § Cross-cutting Concerns (Non-idempotent retry — "017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD`").

**Description**: If the shared executor auto-retried a `POST` on `429`, the withdraw could double-fire. Re-firing a *succeeded* withdraw returns `422` (`withdraw` no longer offered, the proposal is already `draft`), so a naive retry could convert a real success into a surfaced failure.

**Severity**: Low — given the server's state machine, a second withdraw on an already-`draft` proposal is a `422`, not a destructive re-application (the responses are already gone from the first withdraw); the practical impact is a confusing spurious failure, bounded to retry behavior on one command.

**Probability**: Low — `isSafeMethod` (`retry.go:65`) restricts `429` auto-retry to `GET`/`HEAD`, so a `POST` `429` surfaces once and is never silently re-sent; the command reuses the landed `NewRetryExecutor` unchanged and adds no retry surface.

**Risk Level**: Green (Low × Low).

**Controls**:
- **RC-11**: Method-gated auto-retry — the withdraw rides the existing `RetryExecutor`, whose `isSafeMethod` gate excludes `POST` from `429` auto-retry, so the CLI never double-fires; a `429` maps to `RateLimited(5)` and surfaces on first occurrence (plan § Cross-cutting).
- **RC-12**: Behavioral assertion — the "A rate-limited withdraw is surfaced, not silently retried" scenario asserts the `429` is surfaced once with the `POST` not auto-retried (request count == 1), giving this control a BDD assertion in addition to the unit coverage (feature file; tasks T002).

**Residual Risk**: Green — the CLI cannot silently re-send the `POST`; the inherited, tested seam (017) gates it, and the behavior is pinned by both a unit test and the rate-limited feature scenario.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 1 | H-1 |
| Green (accepted) | 4 | H-2, H-3, H-4, H-5 |

**Unacceptable risks**: None. No residual risk remains Red after controls.

- The defining hazard of 059 — **H-1**, a wrong-target withdraw that **irreversibly destroys cast consent responses** with no confirmation guard — is the sole Yellow. It is acceptable with documented justification: the no-confirmation design is a constitution-endorsed deliberate non-behavior for agent automation (an inline prompt would break the agent operator), the success echoes the resulting `draft` proposal for after-the-fact verification, the deletion is recoverable by re-collection (the proposal survives), and the named future control (Write-Safety Guardrail, plan ADR-2) is where the operator-layer second checkpoint will live. This is the developer's explicit accepted trade-off and the one hazard worth their sign-off before implementing — it does not block the spec.
- **H-2** (false success on a refused withdraw) reduces cleanly to Green because the design intercepts no status (conforming to the landed `propose` twin, not the discard idempotency path), a held-out scenario asserts both `404` and `422` fail loudly, and the precedent is recorded.
- The remaining write-specific hazards (**H-3** pre-check race, **H-4** fabricated plan message, **H-5** double-fire) all reduce to Green via the no-pre-check, generic-`403`, and method-gated-retry designs, each with a behavioral or unit assertion behind it.
- **Unlike its `propose` mirror (057), 059 has no sibling-coordination hazard** — the `proposal` group/model/render and the `propose` leaf template are all landed on `main`, so the Yellow that dominated 057's register (H-6) does not arise here. The rate-limit error surface is covered by a feature scenario from the outset (no transient scenario-gap hazard either).

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Invocation / Integration Boundaries / Non-Behaviors; plan.md § ADR-2 |
| H-2 | plan.md § ADR-1; interface-cli.md § Error Communication; tasks.md T001/T002 |
| H-3 | spec.md § Non-Behaviors; plan.md § ADR-1; interface-cli.md § Interactions |
| H-4 | plan.md § ADR-1; interface-cli.md § Error Communication; spec.md § Failure |
| H-5 | plan.md § Cross-cutting Concerns (Non-idempotent retry) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | spec.md § Invocation; CONSTITUTION IX + conflict-resolution; plan ADR-2 — explicit destructive-write intent, no prompt |
| RC-2 | H-1 | plan.md § ADR-1; interface § Surface — the `200` echoes the resulting `draft` proposal for verification; feature "carries the cleared deadline and updated transitions" |
| RC-3 | H-1 | plan.md § ADR-2 — Write-Safety Guardrail named as the deferred operator-layer confirmation; deletion recoverable by re-collection |
| RC-4 | H-2 | plan.md § ADR-1; interface § Error Communication — no-status-interception (the landed `propose` twin), `404`/`422` are failure rows |
| RC-5 | H-2 | spec.md § Validation Scenarios — held-out "A 404 and a 422 are surfaced as real failures"; feature `@validation` |
| RC-6 | H-2 | DECISIONS.md (057 §395/§396 inverse-of-discard; 059 ADR-2 entry) — real-failure rule + landed-twin conformance |
| RC-7 | H-3 | plan.md § ADR-1; interface § Interactions — single bodyless POST, no pre-read |
| RC-8 | H-3 | spec.md § Validation Scenarios — held-out "issues one request and reads nothing first"; feature `@validation` |
| RC-9 | H-4 | plan.md § ADR-1; interface § Error Communication — generic `PermissionError(4)`, no plan branch |
| RC-10 | H-4 | feature "A Premium-gated refusal surfaces plainly" — asserts no plan-specific message |
| RC-11 | H-5 | plan.md § Cross-cutting — `isSafeMethod` gate (`retry.go:65`) excludes `POST` from `429` auto-retry |
| RC-12 | H-5 | features/proposal-write-flow/withdraw-proposal.feature — "A rate-limited withdraw is surfaced, not silently retried"; tasks.md T002 |
