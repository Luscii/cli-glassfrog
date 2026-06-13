# Risk: Tension Discard

**Feature**: 045-tension-discard
**Round**: 1
**Date**: 2026-06-13
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, PROJECT.md, features/tension-capture/tension-discard.feature
**Acceptability matrix**: Default 3×3 traffic light

---

> ⚠ Using default risk acceptability matrix — PROJECT.md defines no Regulatory Context and no project-level risk acceptability matrix.

This is a pre-implementation risk assessment (no code exists yet). Severity and probability are grounded in the plan's architecture and the interface accord, not in observed runtime behavior. The feature file was loaded to check whether each hazard's control has a behavioral assertion behind it; this is recorded per hazard, not as a separate test-gap section (Step 7 is re-run only).

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | A mistyped or never-existed `ten_` id returns `404`, is reported as success, and exits 0 — masking operator error | spec.md § Non-Behaviors; plan.md § Risks | Medium | High | Red | RC-1, RC-2 | Yellow |
| H-2 | The `404`-as-success interception broadens beyond the exact status and silently swallows a real failure (e.g. a `403` permission denial or a not-found from a misrouted path) | plan.md § ADR-2; interface-cli.md § Error Communication | High | Low | Yellow | RC-3, RC-4 | Green |
| H-3 | The synthesized stdout result claims a state the server never confirmed, or fabricates a server-owned field (e.g. `discarded_at`) | spec.md § Validation Scenarios; plan.md § ADR-3 | Medium | Low | Green | RC-5, RC-6 | Green |
| H-4 | The agent-retry idempotency assumption fails: a `DELETE 429` is auto-retried and double-fires, or a non-`404` failure on re-run is not retry-safe | plan.md § Cross-cutting (Non-idempotent retry); spec.md § User Scenarios | Medium | Low | Green | RC-7 | Green |
| H-5 | The success advisory on stderr is mistaken for an error by a caller, or its `204`/`404` wording leaks into the machine-readable stdout result | plan.md § ADR-4; interface-cli.md § Interactions (Piping/scripting) | Low | Low | Green | RC-8, RC-9 | Green |
| H-6 | A non-`404` error surface (malformed credential file, base-URL config error, render failure, invalid `--output`) has no dedicated scenario, so a regression in its handling ships uncaught | interface-cli.md § Error Communication; analyze.md K5 (P1) | Low | Medium | Yellow | RC-10 | Yellow |

---

## Hazard Details

### H-1: `404`-as-success masks a mistyped or never-existed id

**Source**: spec.md § Non-Behaviors ("The system must not treat a `404` as a not-found failure … the accepted cost is that a mistyped id reports success rather than surfacing the typo"); plan.md § Risks (first bullet).

**Description**: Discard treats every `404` on the `DELETE /tensions/{id}` path as success because a single CLI call cannot distinguish an already-discarded tension from one that never existed (the API is documented as not REST-strict idempotent: "treat 404-following-204 as success"). The consequence the spec explicitly accepts: an operator (or agent) who fumbles the `ten_` id — a transposed character, a wrong-resource id, a stale id from another org — gets a green checkmark and exit 0. They believe they discarded a tension; they discarded nothing.

**Severity**: Medium — the wrong outcome is a *false sense of completion*, not data loss or a corrupted write. The intended tension (if a typo) remains live on the active record; the named-but-nonexistent id was never anything. No governance state is mutated incorrectly. Blast radius is confined to the one operator's mental model of one command invocation. It is not Low because an agent driving a multi-step workflow could proceed on the false belief that a cleanup step succeeded, and because the soft-delete is recoverable in spirit but the operator gets no signal that *nothing happened*.

**Probability**: High — typos and stale ids are an everyday occurrence, and the spec's own User Scenarios frame the primary operator as an AI agent issuing ids programmatically (where a malformed-id bug would recur on every call). The plan rates this "likelihood expected (typos happen)." Nothing in the architecture detects the condition; by design it cannot.

**Risk Level**: Red (Medium × High).

**Controls**:
- **RC-1**: The stderr advisory distinguishes the two successes — a `204` emits "discarded tension `<id>`" while a `404` emits "tension `<id>` was already discarded — nothing to do." The operator/agent receives a change-vs-no-change signal on the diagnostic channel even though stdout is identical, so the "nothing happened" case is observable rather than silent (plan ADR-4, interface § Interactions).
- **RC-2**: A genuine permission problem on an existing tension returns `403`, not `404`, and routes to `reportFailure` unchanged (RC-3/RC-4), so the masking is bounded to the truly-gone / never-existed case — it does not extend to "you lacked rights to see it."

**Residual Risk**: Yellow — RC-1 converts a silent success into an *advisory-bearing* success: a human reading stderr, or an agent that inspects stderr, can tell a no-op apart from a real discard. The residual exposure is for a consumer that reads only stdout and the exit code (the documented machine contract), which by design is identical for both successes — that consumer cannot tell a typo from a real discard. This is an explicitly accepted spec non-behavior (the API offers no way to distinguish the two, so no control can fully eliminate it without contradicting the idempotency decision). Acceptable with the documented justification already recorded in spec.md § Non-Behaviors and plan.md § Risks; the advisory is the agreed mitigation.

### H-2: The `404` interception broadens and swallows a real failure

**Source**: plan.md § ADR-2 ("only `404` is — `401`/`403`/`429`/other non-2xx … all still route to `reportFailure` unchanged"); interface-cli.md § Error Communication ("`404` is the **only** non-2xx not present as a failure row").

**Description**: This is the first command in the codebase to fold a non-2xx response into success. The interception must key on *exactly* `StatusCode == 404`. If the predicate ever widened — a range check, a `>= 400 && not 5xx`, a string-match on "not found", or catching a sibling not-found from a different path — a permission denial (`403`), a misrouted not-found, or another 4xx could be silently reported as a successful discard. That is a failure-reported-as-success, the exact anti-pattern Fail-Safe-Not-Silent (CONSTITUTION III) exists to prevent.

**Severity**: High — a swallowed `403` tells the operator a discard succeeded when the server actively refused it; the tension remains live and the operator believes it is gone, with no advisory and no error. Unlike H-1 (where the end-state genuinely is "gone"), here the end-state is "still present and you were denied," so the false success is materially misleading and could mask a real authorization or routing defect across the whole write family that copies this pattern.

**Probability**: Low — the plan, interface, and tasks all pin the predicate to the exact status via `errors.As(err, &respErr)` on `respErr.StatusCode == http.StatusNotFound`, and ADR-2's chosen option explicitly rejects a broader test. Tasks T002 and T003 each carry an acceptance criterion that the interception keys on `404` only and that `403`/`429`/other route normally. The probability is the residual risk that an implementer widens the predicate during a future refactor, or that a sibling bodyless-delete (the plan names proposal withdraw as a follow-on) copies a looser version.

**Risk Level**: Yellow (High × Low).

**Controls**:
- **RC-3**: Exact-status interception — the design requires `errors.As` against a typed `*ResponseError` with `StatusCode == http.StatusNotFound` (the constant), not a range or a substring, so only `404` diverts and every other non-2xx flows through the shared `reportFailure`/`classifyClientError` chokepoint unchanged (plan ADR-2, interface § Error Communication).
- **RC-4**: A pinned negative test — T002/T003 require asserting that a `403` on an existing tension is *not* folded into success and a `429` is surfaced (not swallowed). The feature file backs this: "A refused permission fails with the API status" and "A rate-limited discard is surfaced, not silently retried" exercise the boundary directly, so a broadening regression fails an executable scenario.

**Residual Risk**: Green — with RC-3 pinning the predicate to a single typed status and RC-4 giving it a behavioral assertion that a sibling status (`403`/`429`) must still fail loudly, a broadening of the interception is caught at test time rather than in production. The cross-spec corollary (a future bodyless-delete copying a looser predicate) is flagged in ADR-2's consequences as a DECISIONS.md entry, which carries the exact-status requirement forward.

### H-3: The synthesized result claims an unconfirmed state or fabricates a server field

**Source**: spec.md § Validation Scenarios ("it does not fabricate server-owned fields (e.g. a `discarded_at` timestamp) that no response body provided"); plan.md § ADR-3; plan.md § Risks (third bullet).

**Description**: Both success responses are bodyless — there is no server payload to echo — so the command builds its own stdout result, `{data:{id,discarded}}`. The hazard is twofold: (a) the result asserts `discarded: true` for a state the server did not return in a body, and (b) a future implementer, wanting a "richer" result, adds a fabricated server-owned field such as a `discarded_at` timestamp that the API never sent. Fabricated data violates No-Fabricated-Data (CONSTITUTION VIII).

**Severity**: Medium — a fabricated `discarded_at` would be a plausible-looking lie an agent could persist or act on downstream; a misclaimed `discarded` marker could assert completion that did not occur. Bounded to the one synthesized result; no write or cascade. Not High because the `discarded: true` marker is the command's own attestation of the action it took, and a `204`/`404` *is* the server's confirmation that the tension is gone — so the claim is grounded, not invented; the danger is only in over-reaching it.

**Probability**: Low — ADR-3 explicitly scopes the result to "only the id and a discarded marker — no server-owned fields," the interface accord repeats it ("never a server-owned field … which the bodyless response never provided"), the `TensionDiscardView` is specified with a single `{ID}` field (no timestamp field exists to populate), and a held-out `@validation` scenario ("The synthesized result claims nothing the server did not return") asserts the negative. The architecture gives the implementer no server field to copy.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-5**: Minimal view shape — `TensionDiscardView{ID}` carries exactly one field, the caller-supplied id; there is no struct member for a server-owned value, so a `discarded_at` cannot be populated without a deliberate (and review-visible) schema change (plan ADR-3, tasks T001).
- **RC-6**: A held-out validation assertion — the `@validation` scenario "The synthesized result claims nothing the server did not return" verifies the rendered result carries the id + discarded marker and *no* server-owned field, independently of the implementing agent (spec § Validation Scenarios, feature file).

**Residual Risk**: Green — RC-5 makes fabrication a structural non-option (no field to fill) and RC-6 independently verifies the negative. The `discarded: true` marker is grounded in the server's `204`/`404` (the confirmation that the tension is gone), so it is an attestation, not a fabrication. Accepted.

### H-4: Agent-retry idempotency assumption fails

**Source**: plan.md § Cross-cutting Concerns (Non-idempotent retry — "017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD`"); spec.md § User Scenarios (the agent "I want … a re-run of the same command stay safe rather than fail").

**Description**: The design's safety story for its primary operator (an AI agent) rests on two assumptions: (a) re-running discard is safe because a re-delete's `404` is folded into success (H-1/ADR-2), and (b) the CLI does not *itself* silently re-fire a `DELETE`. If a `DELETE 429` were auto-retried by the shared executor, the command could double-fire a non-idempotent write; and if a non-`404` failure path were not genuinely retry-safe, an agent's reflexive retry could compound an error rather than converge.

**Severity**: Medium — a double-fired `DELETE` is largely harmless *given* ADR-2 (the second fire returns `404`, folded to success), so the practical impact is low; but the assumption is load-bearing for the agent contract, and a future write that copies the pattern without the soft-delete's forgiveness could be harmed by a double-fire. Bounded to retry behavior on one command.

**Probability**: Low — `isSafeMethod` (`retry.go:65`) restricts `429` auto-retry to `GET`/`HEAD`, so a `DELETE` `429` surfaces once and is never silently re-sent; discard reuses the landed `NewRetryExecutor` unchanged and adds no retry surface. The behavior is inherited from a landed, tested seam (017), not newly built here.

**Risk Level**: Green (Medium × Low).

**Controls**:
- **RC-7**: Method-gated auto-retry — discard rides the existing `RetryExecutor`, whose `isSafeMethod` gate excludes `DELETE` from `429` auto-retry, so the CLI never double-fires the delete; a `429` maps to `RateLimited(5)` and is surfaced on first occurrence. The feature scenario "A rate-limited discard is surfaced, not silently retried" pins this (plan § Cross-cutting, interface § Interactions, feature file).

**Residual Risk**: Green — the CLI cannot silently re-send the `DELETE`, and an *operator/agent*-initiated re-run is the safe case the idempotent-`404` decision (ADR-2) is built for. The combination of "no silent CLI retry" + "`404`-as-success on re-run" makes the agent-retry assumption sound. Accepted.

### H-5: Success advisory on stderr mistaken for an error, or leaks into stdout

**Source**: plan.md § ADR-4 (advisory on stderr, stdout identical); plan.md § Risks (fourth bullet); interface-cli.md § Interactions (Piping/scripting).

**Description**: This is the first command to write a *success* advisory to stderr — a small extension of the failures-only stderr convention (031/032). Two failure modes: (a) a caller that treats any stderr output as an error flag could misclassify a successful discard as failed; (b) the `204`/`404` distinction could leak into the machine-readable stdout result, breaking the contract that stdout is byte-identical for both successes.

**Severity**: Low — the documented contract is exit-code + stdout; a caller keying on those (the correct contract) is unaffected. A stderr-as-error heuristic is a caller-side mis-assumption, not a defect in this command. Leakage into stdout would break the identical-result contract but is a rendering bug confined to one command's output.

**Probability**: Low — the advisory is specified as a single informational `Fprintln` to stderr with exit `0`; the design explicitly keeps stdout identical for `204`/`404` (the distinction "rides stderr"), and ADR-4 rejected the alternative of an in-band `already_discarded` flag. The feature scenarios assert stderr carries the discarded / already-gone note while exit is `0` and stdout carries the standard result.

**Risk Level**: Green (Low × Low).

**Controls**:
- **RC-8**: Channel separation by contract — the change-vs-no-change signal is confined to stderr; stdout carries only the rendered synthesized result, byte-identical for `204` and `404`, and the exit code is `0` for both. Machine consumers read stdout + exit code and can ignore stderr (plan ADR-4, interface § Interactions/Piping).
- **RC-9**: Advisory-not-error discipline — the stderr line is informational, never routed through the error/`reportFailure` path, never includes the token, and does not change the exit code; the `404` validation scenario asserts "no not-found error reaches the user" and the note is "advisory information" with exit `0` (spec § Validation Scenarios, feature file).

**Residual Risk**: Green — the machine contract (stdout + exit code) is stable and well-specified; the residual is a caller-side stderr-as-error heuristic, which is outside this command's control and contradicts the documented contract. Accepted.

### H-6: Non-`404` error surfaces lack dedicated scenario coverage

**Source**: interface-cli.md § Error Communication (13 outcome rows); analyze.md K5 (P1) — four interface-defined error surfaces have no Gherkin scenario.

**Description**: The interface accord defines thirteen outcome surfaces. The feature file carries behavioral scenarios for nine. Four have no dedicated scenario: malformed/unreadable credential file (`*AuthError{CredentialError}` → `RuntimeError(1)`), base-URL configuration error (→ `UsageError(2)`), render-of-synthesized-result failure (→ `RuntimeError(1)`), and present-but-invalid `--output` selector (→ `UsageError(2)`, no request). A regression in any of these paths could ship without a failing scenario to catch it. This is the one open P1 carried over from the analyze pass.

**Severity**: Low — all four are downstream-shared seams (005/008/011/015/018/032) that are landed and validated by their own specs, not discard-specific logic. Discard reuses them unchanged via `reportFailure`/`classifyClientError`/`resolveRenderTarget`; the obligation to handle them is present in the design (the interface table enumerates them and tasks T002 maps them to `reportFailure`). The gap is in *dedicated* scenario coverage, not in implementation obligation. The bad-`--output` no-request guarantee is additionally covered by the T002/T003 transport tripwire even without a Gherkin scenario.

**Probability**: Medium — the four paths are real code branches the command must wire; without a scenario, a wiring mistake (e.g. resolving `--output` *after* assembly, breaking the no-request guarantee; or a render failure not buffer-then-written) would not be caught by the behavioral suite. Medium rather than Low because these are exactly the kind of precedence/ordering details that drift (the no-request-before-render ordering is called out as load-bearing in tasks T002).

**Risk Level**: Yellow (Low × Medium).

**Controls**:
- **RC-10**: Unit-test branch coverage — plan § Cross-cutting (Testing) and tasks T002 require offline unit tests for *every* branch, including not-authenticated, credential-file error, bad-`--output` (with a transport tripwire confirming no request), and render failure. The branches are covered at the unit level even where a Gherkin scenario is absent; the bad-`--output` no-request guarantee specifically has a tripwire assertion.

**Residual Risk**: Yellow — RC-10's unit coverage exercises the four paths, so the gap is narrowed to *dedicated executable-acceptance* coverage (the BDD layer), not total absence of test. The residual is the well-understood trade recorded in analyze.md K5 (these are shared-seam reuses, not discard-specific logic). Acceptable with the documented justification; the developer may close it by adding Gherkin scenarios for the four surfaces (most cheaply the bad-`--output` no-request case, since its ordering is load-bearing) via `/score:scenarios`, but it does not block implementation.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 2 | H-1, H-6 |
| Green (accepted) | 4 | H-2, H-3, H-4, H-5 |

**Unacceptable risks**: None. No residual risk remains Red after controls.

- **H-1** (Red → Yellow) and **H-6** (Yellow, unchanged) are acceptable with documented justification:
  - **H-1**: The `404`-as-success masking is an explicitly accepted spec non-behavior — the API offers no way to distinguish an already-discarded tension from a never-existed one, so no control can eliminate it without reversing the idempotency decision the spec made deliberately. The stderr advisory (RC-1) is the agreed mitigation and gives the operator/agent a change-vs-no-change signal.
  - **H-6**: The four uncovered error surfaces are landed shared-seam reuses with unit-level branch coverage (RC-10); the gap is dedicated BDD scenarios, not implementation obligation. Recorded as P1 in analyze.md. The developer can close it via `/score:scenarios` but it does not block implementation.
- The defining novel decision of 045 — `404`-as-success — generated the two highest-attention hazards (**H-1** masking, **H-2** over-broadening). H-2 reduces cleanly to Green because the exact-status predicate and its negative test (`403`/`429` still fail loudly) are pinned in plan, interface, tasks, and the feature file. H-1 cannot reduce below Yellow by design, and that Yellow is the developer's to accept (it already is, in the spec).

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Non-Behaviors; plan.md § Risks |
| H-2 | plan.md § ADR-2; interface-cli.md § Error Communication |
| H-3 | spec.md § Validation Scenarios; plan.md § ADR-3 |
| H-4 | plan.md § Cross-cutting Concerns (Non-idempotent retry); spec.md § User Scenarios |
| H-5 | plan.md § ADR-4; interface-cli.md § Interactions (Piping/scripting) |
| H-6 | interface-cli.md § Error Communication; analyze.md K5 (P1) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | plan.md § ADR-4 — `204`/`404` stderr advisory; interface § Interactions |
| RC-2 | H-1 | plan.md § ADR-2 — permission denial returns `403` not `404`, routes to `reportFailure` |
| RC-3 | H-2 | plan.md § ADR-2 — exact `errors.As` on `StatusCode == http.StatusNotFound`; interface § Error Communication |
| RC-4 | H-2 | tasks.md T002/T003 acceptance — pinned negative test; feature file "refused permission" / "rate-limited … not silently retried" |
| RC-5 | H-3 | plan.md § ADR-3 — single-field `TensionDiscardView{ID}`; tasks.md T001 |
| RC-6 | H-3 | spec.md § Validation Scenarios — held-out "claims nothing the server did not return"; feature file `@validation` |
| RC-7 | H-4 | plan.md § Cross-cutting — `isSafeMethod` gate (`retry.go:65`) excludes `DELETE` from `429` auto-retry; feature "not silently retried" |
| RC-8 | H-5 | plan.md § ADR-4 — stdout byte-identical, signal confined to stderr; interface § Interactions/Piping |
| RC-9 | H-5 | spec.md § Validation Scenarios — `404` leaks no error, advisory is informational, exit 0; feature file `@validation` |
| RC-10 | H-6 | plan.md § Cross-cutting (Testing); tasks.md T002 — offline unit coverage of every branch incl. bad-`--output` no-request tripwire |
