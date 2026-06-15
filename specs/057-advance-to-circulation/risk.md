# Risk: Advance to Circulation

**Feature**: 057-advance-to-circulation
**Round**: 1
**Date**: 2026-06-15
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md, features/proposal-write-flow/advance-to-circulation.feature
**Acceptability matrix**: Default 3×3 traffic light

---

> ⚠ Using default risk acceptability matrix — PROJECT.md defines no Regulatory Context and no project-level risk acceptability matrix.

This is a pre-implementation risk assessment (no code exists yet, and the sibling `proposal` group/model/render it consumes have not landed). Severity and probability are grounded in the plan's architecture and the interface accord, not in observed runtime behavior. The feature file was loaded to check whether each hazard's control has a behavioral assertion behind it; this is recorded per hazard, not as a separate test-gap section (Step 7 is re-run only).

> **Update (2026-06-15, post-guard):** H-7 was closed after this pass — a "A rate-limited advance is surfaced, not silently retried" scenario was added to advance-to-circulation.feature (also closing analyze K5). H-7's residual is now Green, and H-5 (`429` double-fire) gains a BDD assertion. The register, summary, and traceability below reflect the post-closure state.

This is a **write** to live governance: a successful advance mutates a proposal's state (`draft → proposed_outside_meeting`), fires circle notifications, sets a consent deadline, and records the proposer's implicit `no_objection` — so the hazard surface centers on (a) advancing the *wrong* proposal, (b) reporting a refused/failed advance as success, and (c) the cross-spec coordination of consuming sibling-owned machinery that has not yet landed.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | A valid-but-wrong `prp_` id advances a proposal the operator did not intend — firing circle notifications and starting a consent clock on the wrong proposal | spec.md § Invocation / Integration Boundaries / Non-Behaviors (no confirmation prompt) | Medium | Low | Green | RC-1, RC-2, RC-3 | Green |
| H-2 | A `404` or `422` is folded into success (copying discard's `404`-as-success), reporting a refused/failed advance as a successful circulation | plan.md § ADR-3; interface-cli.md § Error Communication; tasks.md T001/T002 (risk notes) | High | Low | Yellow | RC-4, RC-5, RC-6 | Green |
| H-3 | A client-side pre-check of `available_transitions` is added, forking server authority and acting on a stale snapshot (TOCTOU) | spec.md § Non-Behaviors; plan.md § ADR-3; interface-cli.md § Interactions | Medium | Low | Green | RC-7, RC-8 | Green |
| H-4 | The Premium `403` is given a bespoke "not available on your plan" message — fabricating a plan-gate classification the API did not structurally provide | plan.md § ADR-3; interface-cli.md § Error Communication; spec.md § Failure | Medium | Low | Green | RC-9, RC-10 | Green |
| H-5 | A `POST` `429` is auto-retried and double-fires the advance; a re-fire of a succeeded advance then surfaces a spurious `422` | plan.md § Cross-cutting (Non-idempotent retry) | Low | Low | Green | RC-11, RC-16 | Green |
| H-6 | The sibling `proposal` group/model/singular render (055/056) has not landed when 057 is built — breaking the base, or producing a duplicate divergent model/render | plan.md § Cross-cutting (Sibling coordination) / § Risks; tasks.md § Dependency Graph / Branching | Medium | Medium | Yellow | RC-12, RC-13, RC-14 | Yellow |
| H-7 | The `429`/`401` (and other shared-seam) error surfaces in the interface have no feature scenario, so a wiring regression ships uncaught | interface-cli.md § Error Communication; analyze.md K5 (resolved) | Low | Medium | Yellow | RC-15, RC-16 | Green |

---

## Hazard Details

### H-1: A valid-but-wrong `prp_` id advances the wrong proposal

**Source**: spec.md § Invocation (one `prp_` positional, passed through unvalidated); § Integration Boundaries (a `200` fires notifications and sets the deadline); § Non-Behaviors ("must not prompt for confirmation … the advance is reversible via Withdraw").

**Description**: `propose` is an explicit write keyed off a single caller-supplied `prp_` id, passed through to the API unvalidated and with **no confirmation prompt** (a deliberate non-behavior for agent automation, per CONSTITUTION's conflict-resolution rule). Unlike discard — where a wrong/typo'd id is almost always a harmless `404` no-op — a wrong id here that *happens to be valid, visible, in `draft`, and have `propose` available* will genuinely advance the wrong proposal: notifications fire to that circle, a consent deadline starts, and the proposer's implicit `no_objection` is recorded. The side effects are externally visible.

**Severity**: Medium — the wrong-advance fires real, circle-visible notifications and starts a consent clock, but it is **reversible** (Withdraw Proposal returns a circulating proposal to `draft`, clearing responses and timestamps) and mutates no governance *structure* (the proposal's `changes[]` are not applied until acceptance). No data loss, no structural corruption; blast radius is one proposal and the social cost of a spurious notification.

**Probability**: Low — a *successful* wrong-advance requires a narrow conjunction: the wrong id must be syntactically valid, visible to the caller, in `draft`, and offer `propose`. A typo'd or stale id almost always fails at `404` (not found / not visible) or `422` (not a `draft` / not offered) — which are surfaced as failures (H-2 controls). The id is supplied with explicit write intent, not derived implicitly.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-1**: Explicit write intent — the command is a deliberate write verb taking the id as a required positional; the constitution accepts "the command's existence is the intent" in lieu of a prompt, and the operator/agent chooses the id knowingly (spec § Invocation; CONSTITUTION IX + conflict-resolution).
- **RC-2**: The success echoes the advanced proposal — the `200` renders the full `Proposal` (its id, the new `proposed_outside_meeting` status, the deadline), so the operator/agent can immediately verify *which* proposal advanced and detect a wrong-target (plan ADR-2; interface § Surface; feature "carries the deadline and implicit response").
- **RC-3**: Reversibility — Withdraw Proposal (a named sibling) returns a wrongly-circulated proposal to `draft`, clearing responses and proposed timestamps, so the side effect is recoverable rather than terminal (spec § Non-Behaviors).

**Residual Risk**: Green — the narrow precondition for a *successful* wrong-advance, the echoed proposal for immediate verification, and the Withdraw reversal path keep this acceptable. The no-prompt design is an explicitly accepted non-behavior grounded in the agent-automation contract.

### H-2: A `404`/`422` is folded into success (the inverse-of-discard trap)

**Source**: plan.md § ADR-3 ("it INTERCEPTS NO status … the EXACT OPPOSITE of discard's `errors.As`-for-`404`-as-success divert"); interface-cli.md § Error Communication (`404`/`422` present as failure rows); tasks.md T001/T002 risk notes.

**Description**: The closest implementation template for this command is the discard leaf (045), which *folds its `404` into success* because a soft-delete has an idempotent end-state. An advance does not. If an implementer copies the discard pattern and carries the `errors.As`-for-`404`-as-success divert across — or widens it to `422` — then a refused transition (`422`, not allowed) or an unknown proposal (`404`) would be reported as a *successful circulation*. That is a failure-reported-as-success, the exact anti-pattern CONSTITUTION III (Fail Safe, Not Silent) exists to prevent, and it is materially misleading on a governance write: the operator believes a proposal entered circulation when the server refused.

**Severity**: High — a swallowed `422`/`404` tells the operator/agent an advance succeeded when nothing circulated; a multi-step agent workflow would proceed (e.g. waiting for responses, or recording its own response) on a proposal that is still a draft or does not exist. No advisory, no error — a silent false success on live governance.

**Probability**: Low — the design pins "intercept no status" in plan ADR-3, the interface error table lists `404` and `422` as explicit failure rows, tasks T001/T002 carry it as a load-bearing risk ("do not copy discard's divert"), and a held-out `@validation` scenario asserts both are real failures. The residual probability is an implementer copying a looser predicate during the build or a future transition sibling (Withdraw) reusing a loosened version.

**Risk Level**: Yellow (High × Low).

**Controls**:
- **RC-4**: No-status-interception design — every non-2xx routes through the shared `reportFailure`/`classifyClientError` chokepoint unchanged; there is no `errors.As`-for-status divert at all (plan ADR-3; interface § Error Communication).
- **RC-5**: Pinned negative assertion — the `@validation` scenario "A 404 and a 422 are surfaced as real failures" verifies neither produces a success result and both exit non-zero, independently of the implementing agent (spec § Validation Scenarios; feature file).
- **RC-6**: Cross-spec precedent — the DECISIONS.md entry recording 057's inverse-of-discard stance carries the "treats `404`/`422` as real failures" rule forward to the Withdraw/Response transitions that will reuse this decode-and-render path.

**Residual Risk**: Green — with the design intercepting no status (RC-4), an executable held-out assertion that `404`/`422` fail loudly (RC-5), and the precedent recorded for siblings (RC-6), a fold-to-success regression is caught at validation time rather than in production.

### H-3: A client-side pre-check forks server authority and races a stale snapshot

**Source**: spec.md § Non-Behaviors ("must not pre-read the proposal to inspect `available_transitions`"); plan.md § ADR-3; interface-cli.md § Interactions ("no prior `GET`").

**Description**: `available_transitions` is surfaced by the reads (056), making it tempting to add a "is `propose` allowed?" pre-read before the POST "to be helpful." Doing so (a) forks the server's authority over the transition rule, (b) adds a round-trip, and (c) introduces a TOCTOU race — the proposal could change between the pre-read and the write, so a stale "propose allowed" snapshot green-lights a POST the server then refuses anyway (or, worse, a pre-check that wrongly *blocks* a valid advance).

**Severity**: Medium — the server remains authoritative (it still `422`s a disallowed transition), so a stale pre-check mostly costs a wasted read and a confusing client-side gate; it does not corrupt state. The drift is a correctness/efficiency regression and a forked source of truth, not a data hazard.

**Probability**: Low — the no-pre-check is an explicit spec non-behavior, pinned in plan ADR-3 and the interface, and a `@validation` scenario asserts exactly one request and no prior read.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-7**: Single-request design — the command issues exactly one bodyless `POST` and lets the server's `422` enforce; it builds no pre-read (plan ADR-3; interface § Interactions).
- **RC-8**: Held-out assertion — the `@validation` scenario "The advance issues one request and reads nothing first" verifies a single request to the propose transition and no prior `GET` (spec § Validation Scenarios; feature file).

**Residual Risk**: Green — the design issues one request and the held-out scenario pins it; the server stays the single authority for the transition rule.

### H-4: The Premium `403` is given a fabricated plan-gate message

**Source**: plan.md § ADR-3 ("the Premium `403` gets NO bespoke 'not available on your plan' message"); interface-cli.md § Error Communication; spec.md § Failure.

**Description**: The transition is Premium-gated, so a `403` is expected when async proposals are disabled. But the `403` is a generic RFC 9457 `ProblemDetails` with **no structured field** distinguishing a plan-gate from an ordinary permission denial. If the command adds a bespoke "not available on your plan" message, it fabricates a classification the API did not provide (a guess presented as fact — touching CONSTITUTION VIII, No Fabricated Data) and forks the plan-gate signal that belongs to the separate Plan-Limit Signalling capability, producing inconsistent `403` handling across the write path.

**Severity**: Medium — a fabricated "it's your plan" message could misdirect an operator whose `403` was actually an ordinary permission denial (the two are indistinguishable in the `403` body), wasting troubleshooting on the wrong cause. Bounded to one command's diagnostic text; no state impact.

**Probability**: Low — the design explicitly routes the `403` through the shared classifier as a generic `PermissionError(4)` with no plan-aware branch, the interface table says so, and the feature scenario "A Premium-gated refusal surfaces plainly" asserts no plan-specific message is added.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-9**: Generic-`403` design — the `403` is a plain `PermissionError(4)` via the shared chain; the command adds no plan-aware branch and surfaces only the API's extracted detail (plan ADR-3; interface § Error Communication).
- **RC-10**: Behavioral assertion — the feature scenario "A Premium-gated refusal surfaces plainly" verifies the failure names the HTTP status with **no** "not available on your plan" message (feature file); the dedicated plan-gate signal is deferred to Plan-Limit Signalling, which will refine the `403` centrally (the way 054 refined the `412` for all writes).

**Residual Risk**: Green — no fabricated plan classification is produced; the `403` is handled uniformly with every other refusal, and the scenario pins the absence of a bespoke message.

### H-5: A `POST` `429` double-fires the advance

**Source**: plan.md § Cross-cutting Concerns (Non-idempotent retry — "017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD`").

**Description**: If the shared executor auto-retried a `POST` on `429`, the advance could double-fire. Re-firing a *succeeded* advance returns `422` (`propose` no longer offered), so a naive retry could convert a real success into a surfaced failure; and a future non-idempotent transition copying the pattern could be harmed by a genuine double-application.

**Severity**: Low — given the server's own state machine, a second propose on an advanced proposal is a `422`, not a destructive re-application; the practical impact is a confusing spurious failure, bounded to retry behavior on one command.

**Probability**: Low — `isSafeMethod` (`retry.go:65`) restricts `429` auto-retry to `GET`/`HEAD`, so a `POST` `429` surfaces once and is never silently re-sent; the command reuses the landed `NewRetryExecutor` unchanged and adds no retry surface.

**Risk Level**: Green (Low × Low).

**Controls**:
- **RC-11**: Method-gated auto-retry — the advance rides the existing `RetryExecutor`, whose `isSafeMethod` gate excludes `POST` from `429` auto-retry, so the CLI never double-fires; a `429` maps to `RateLimited(5)` and surfaces on first occurrence (plan § Cross-cutting).
- **RC-16**: Behavioral assertion — the "A rate-limited advance is surfaced, not silently retried" scenario (added to close H-7) asserts the `429` is surfaced once with the `POST` not auto-retried (request count == 1), giving this control a BDD assertion in addition to the unit coverage (feature file; tasks T002).

**Residual Risk**: Green — the CLI cannot silently re-send the `POST`; the inherited, tested seam (017) gates it, and the behavior is now pinned by both a unit test and the rate-limited feature scenario.

### H-6: The sibling `proposal` group/model/render has not landed

**Source**: plan.md § Cross-cutting (Sibling coordination) and § Risks (first bullet); tasks.md § Dependency Graph and § Branching Guidance.

**Description**: 057 is a **pure consumer** of three artifacts owned by Proposal Reads (056, Analyzed) and/or Proposal Creation (055, concurrent): the `proposal` command group (`newProposalCommand` + the `proposalSeam`), the `glassfrog.Proposal` model + `Document[Proposal]` decode, and the singular `proposal` render path. As of writing, **neither sibling has landed on `main`**. If 057 is implemented before either lands, the base is missing all three; and if the first-to-land contract is mishandled, an implementer could create a *second, divergent* `Proposal` model or singular render — violating grow-not-duplicate (DECISIONS 011 ADR-1) and Composition over Monolith (CONSTITUTION V). This is the real-world coordination hazard the tension write leaves (042–045) did not face, because their group/model/render were already landed.

**Severity**: Medium — a missing base breaks the build (caught immediately); a duplicate divergent model/render is a genuine integration defect (two sources of truth for the proposal shape, forked human rendering), but it is contained to wiring + render and would surface in review/merge, not in live governance data.

**Probability**: Medium — landing order across parallel Conductor workspaces is genuinely unknown (056 is Analyzed, 055 concurrent, 057 just shaped); the conditions for the hazard (057 picked up before a sibling lands) are plausible, not remote.

**Risk Level**: Yellow (Medium × Medium).

**Controls**:
- **RC-12**: First-to-land-creates / follower-reuses contract — documented in plan § Cross-cutting and Risks, and in tasks § Branching; the follower rebases and reuses (or grows, never duplicates) the model/render (plan; tasks).
- **RC-13**: Explicit start gate — tasks T001 marks itself "Not startable until a sibling proposal spec has landed the group/model/singular render," with the leads-first fallback (create only the minimal subset) spelled out, so the implementer does not begin against an absent base (tasks § Dependency Graph / T001 Dependencies).
- **RC-14**: Faithful-structured-output backstop — even if model field coverage diverges between siblings, `output.RenderSuccess` emits the raw `{data: Proposal}` bytes, so `-o json/yaml` is correct regardless, bounding any divergence to human render + cobra wiring (plan § Cross-cutting; ADR-2).

**Residual Risk**: Yellow — the coordination contract, the explicit start gate, and the raw-bytes backstop bound the blast radius to wiring + human render, caught at build/merge rather than in production. The residual is the inherent cross-spec sequencing exposure of building a consumer before its provider lands; it is acceptable with the documented justification (the follower path is the expected case, and the gate prevents a premature start) and is the developer's to manage at scheduling time — it does not block the spec.

### H-7: Shared-seam error surfaces lack dedicated scenario coverage

**Source**: interface-cli.md § Error Communication (the `429`, `401`, credential-file, base-URL, and render-failure rows); analyze.md K5 (resolved).

**Description**: The interface accord enumerates the full failure surface. As originally shaped, the feature file carried no scenario for the `429` (rate-limited) path — the one *distinct outcome* (`RateLimited(5)`) that nothing else exercised — nor the `401` path or the credential-file / base-URL / render-failure rows. The sibling write leaf 045 included a rate-limited scenario; 057 did not. A wiring regression in one of these branches could have shipped without a failing BDD scenario. **This hazard has been closed** (see RC-16): a rate-limited scenario was added, covering the distinct `429` outcome and the no-auto-retry behavior.

**Severity**: Low — all are downstream shared-seam reuses (005/008/011/015/017/018/032), landed and validated by their own specs; 057 reuses them unchanged via `reportFailure`/`classifyClientError`/`resolveRenderTarget`. The obligation is present in the design (the interface table enumerates them, tasks T001 maps them to `reportFailure`); the gap was *dedicated* BDD coverage, not implementation obligation.

**Probability**: Medium — these are real code branches; without a scenario, an ordering/wiring mistake (e.g. resolving `--output` after assembly, breaking the no-request guarantee; or a `429` not surfacing once) would not be caught by the behavioral suite. Medium rather than Low because the resolve-first ordering and the non-retry gate are exactly the kind of detail that drifts.

**Risk Level**: Yellow (Low × Medium) — pre-closure.

**Controls**:
- **RC-15**: Unit-test branch coverage — plan § Cross-cutting (Testing) and tasks T001 require offline unit tests for *every* branch, including `429` (surfaced once, not retried), `401`, not-authenticated, credential-file error, and bad-`--output` (with a transport tripwire confirming no request). The branches are covered at the unit level.
- **RC-16**: Rate-limited feature scenario — "A rate-limited advance is surfaced, not silently retried" was added to advance-to-circulation.feature, asserting the `429` surfaces once with the `POST` not auto-retried (request count == 1). This exercises the only *distinct* uncovered outcome (`RateLimited(5)`) at the BDD level. The `401`→`PermissionError(4)` outcome is exercised by the Premium-`403` scenario (one scenario per outcome class, the 045 precedent); the credential-file / base-URL / render-failure rows remain unit-covered (RC-15), as in 045.

**Residual Risk**: Green — RC-16 closes the one distinct-outcome BDD gap (`429`/no-retry); the permission-class outcome is BDD-covered by the `403` scenario, and the remaining shared-seam rows have unit coverage. The original analyze K5 finding is resolved. No residual coverage gap of consequence remains.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 1 | H-6 |
| Green (accepted) | 6 | H-1, H-2, H-3, H-4, H-5, H-7 |

**Unacceptable risks**: None. No residual risk remains Red after controls.

- The defining novel decision of 057 — `404`/`422` as **real failures** (the inverse of discard's `404`-as-success) — generated the highest-attention hazard (**H-2**, false success on a governance write). It reduces cleanly to Green because the design intercepts no status, a held-out scenario asserts both fail loudly, and the precedent is recorded for the sibling transitions.
- **H-7** (shared-seam scenario coverage) was **closed** to Green: a rate-limited (`429`) feature scenario was added (RC-16), covering the only distinct uncovered outcome and pinning the no-auto-retry behavior; the permission-class outcome is BDD-covered by the `403` scenario, and the remaining shared-seam rows keep unit coverage (the 045 pattern). The original analyze K5 finding is resolved.
- **H-6** (sibling coordination) remains the sole Yellow, acceptable with documented justification: the cross-spec sequencing exposure of building a consumer before its provider (055/056) lands. The first-to-land contract, the explicit T001 start gate, and the raw-bytes structured-output backstop bound it to wiring + human render, caught at build/merge. It is the developer's to manage at scheduling time — not a spec defect.
- The write-specific real-world hazards (**H-1** wrong-target advance, **H-3** pre-check race, **H-4** fabricated plan message, **H-5** double-fire) all reduce to Green via the explicit-write-intent, no-pre-check, generic-`403`, and method-gated-retry designs, each with a behavioral or unit assertion behind it (**H-5** now BDD-covered by the rate-limited scenario).

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Invocation / Integration Boundaries / Non-Behaviors |
| H-2 | plan.md § ADR-3; interface-cli.md § Error Communication; tasks.md T001/T002 |
| H-3 | spec.md § Non-Behaviors; plan.md § ADR-3; interface-cli.md § Interactions |
| H-4 | plan.md § ADR-3; interface-cli.md § Error Communication; spec.md § Failure |
| H-5 | plan.md § Cross-cutting Concerns (Non-idempotent retry) |
| H-6 | plan.md § Cross-cutting (Sibling coordination) / § Risks; tasks.md § Dependency Graph / Branching |
| H-7 | interface-cli.md § Error Communication; analyze.md K5 (resolved) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | spec.md § Invocation; CONSTITUTION IX + conflict-resolution — explicit write intent, no prompt |
| RC-2 | H-1 | plan.md § ADR-2; interface § Surface — the `200` echoes the advanced proposal for verification; feature "carries the deadline and implicit response" |
| RC-3 | H-1 | spec.md § Non-Behaviors — Withdraw reverses a wrongly-circulated proposal |
| RC-4 | H-2 | plan.md § ADR-3; interface § Error Communication — no-status-interception, `404`/`422` are failure rows |
| RC-5 | H-2 | spec.md § Validation Scenarios — held-out "A 404 and a 422 are surfaced as real failures"; feature `@validation` |
| RC-6 | H-2 | DECISIONS.md (057 inverse-of-discard entry) — carries the real-failure rule to Withdraw/Response |
| RC-7 | H-3 | plan.md § ADR-3; interface § Interactions — single bodyless POST, no pre-read |
| RC-8 | H-3 | spec.md § Validation Scenarios — held-out "issues one request and reads nothing first"; feature `@validation` |
| RC-9 | H-4 | plan.md § ADR-3; interface § Error Communication — generic `PermissionError(4)`, no plan branch |
| RC-10 | H-4 | feature "A Premium-gated refusal surfaces plainly" — asserts no plan-specific message |
| RC-11 | H-5 | plan.md § Cross-cutting — `isSafeMethod` gate (`retry.go:65`) excludes `POST` from `429` auto-retry |
| RC-12 | H-6 | plan.md § Cross-cutting / § Risks; tasks.md § Branching — first-to-land-creates / follower-reuses |
| RC-13 | H-6 | tasks.md § Dependency Graph / T001 — explicit "not startable until base in place" gate + leads-first fallback |
| RC-14 | H-6 | plan.md § Cross-cutting; ADR-2 — raw-bytes structured output faithful regardless of model coverage |
| RC-15 | H-7 | plan.md § Cross-cutting (Testing); tasks.md T001 — offline unit coverage of every branch incl. `429`/`401`/bad-`--output` tripwire |
| RC-16 | H-5, H-7 | features/proposal-write-flow/advance-to-circulation.feature — "A rate-limited advance is surfaced, not silently retried" (added to close K5/H-7); tasks.md T002 |
