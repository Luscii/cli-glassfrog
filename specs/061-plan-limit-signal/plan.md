# Plan: Plan-Limit Signal

**Feature**: 061-plan-limit-signal
**Role**: Shaper
**Inputs**: spec.md (061), PROJECT.md, FEATURE-MODEL.md (Plan-Limit Signalling), DECISIONS.md, DEPRECATION.md, LEARNINGS.md; the 060 plan (the recognizer contract this consumes); grounded against `internal/apiclient/execute.go` (010 — `Execute`, `ResponseError`), `internal/apiclient/problem.go` (015 — `ProblemError`, `ExtractProblem`), `internal/cli/diagnostic.go` (031 — `Diagnose`, `Diagnostic`, `categoryForStatus`, `nextStepForStatus`, `renderDiagnostic`), `internal/cli/me.go` (032 — `reportFailure`, `refineClientError`), `internal/cli/errorenvelope.go` (032 — `errorEnvelopeFor`, `kind`), `internal/output/error.go` (018 — `ErrorEnvelope`, `ErrorDetail`), and `internal/cli/exitcode.go` (004).

---

## System Architecture

Plan-Limit Signal is the **rendering half** of Plan-Limit Signalling: it turns Feature-Gate Recognition's (060) code-free classification into the actionable *"may not be available on your plan"* diagnostic that Diagnostic Normalization (031) and Output-Aware Failure Rendering (032) both deliberately refused to produce. It adds **no new component** — it is a set of small, surgical edits along the one existing failure chain so that a recognized plan-gate `403` carries gate-aware wording everywhere a failure is already rendered.

**The chain it edits (today).** Every command-execution failure funnels through one chokepoint: `reportFailure` (`me.go`, 032) → `refineClientError` (the single `*ResponseError`→`*ProblemError` site, 015 ADR-4) → `Diagnose` (`diagnostic.go`, 031), which sets the `Diagnostic{Category, Cause, NextStep}` that both renderers consume — `renderDiagnostic` for the human `full`/`compact` stderr line, and `errorEnvelopeFor` → `output.RenderError` for the structured `json`/`yaml` envelope on stdout. `Diagnose` keys a `403` *purely on status*: `categoryForStatus(403)` → `PermissionError` (exit 4) and `nextStepForStatus(403)` → a generic "check the identity's role/permission" hint. It has **no operation awareness** — exactly the gap 060's `RecognizeFeatureGate(method, path, status) Gate` fills, and exactly the gap this feature wires in.

**The one missing input — operation identity.** `RecognizeFeatureGate` needs the failed request's HTTP method and path; the typed error chain does not carry them today (`ResponseError` holds only `StatusCode`, `Header`, `Body`; `ProblemError` wraps it and adds the RFC 9457 members). This feature supplies that input by **enriching the error at its source** — `Execute` constructs the `*ResponseError` with `req.Method`/`req.Path` already in scope (ADR-1) — so the method/path ride the error through the unchanged chain to `Diagnose`, where recognition runs.

**What this feature delivers.**
- `ResponseError` gains `Method` and `Path`, populated in `Execute` (the threading input — additive, every other consumer unaffected).
- `Diagnose` (031, the central site) runs `RecognizeFeatureGate` on a `403` whose wrapped `ResponseError` matches a gated operation, and on a non-`None` gate refines `Cause`/`NextStep` to the possibility-framed plan-limit wording naming the gate — **keeping `Category = PermissionError`, so the exit code is unchanged** (ADR-2). The recognized gate is carried on the `Diagnostic` so both renderers present it consistently (ADR-3).
- The structured envelope (`ErrorDetail`, 018) gains a distinct, `omitempty` `feature` element, mapped by `errorEnvelopeFor` from the `Diagnostic`'s carried gate (ADR-4) — so an agent reads the gating feature programmatically, while the human line carries it in the cause prose.

Because the diagnostic is shaped *inside* `Diagnose`, 032's existing renderer carries it to the selected format **for free** — which is precisely why this feature depends on 060 + 031, not on 032.

**Data flow.** Request fails → `Execute` returns `*ResponseError{StatusCode, Header, Body, Method, Path}` → `refineClientError` wraps it as `*ProblemError` (unchanged) → `Diagnose` matches `*ProblemError`, reads the wrapped `*ResponseError`'s `Method`/`Path` via `errors.As`, calls `RecognizeFeatureGate` → on a non-`None` gate, sets plan-limit `Cause`/`NextStep`/`Feature` (category unchanged) → `renderDiagnostic` (human) / `errorEnvelopeFor` (structured) present it in the selected format.

---

## Architecture Decisions

### ADR-1: Thread operation identity by enriching the typed error at its source, not by widening the failure-site signature

**Context**: `RecognizeFeatureGate` needs the failed request's method + path, but the error chain that reaches `Diagnose` carries neither. The 060 plan flagged this threading as #61's cost and left the mechanism open. `reportFailure` — the chokepoint where `Diagnose` runs — receives only `err` and is called from ~100 sites across every command file.

**Options considered**:
1. **Widen `reportFailure(…, err)` to `reportFailure(…, method, path, err)`** — pass method/path alongside the error at each call site. Disadvantage: a mechanical edit to ~100 call sites (and their fakes), most of which don't even have the method/path handy (they hold a domain error from a walker/seam, not the raw request); huge churn for a Could-Have, and it puts request metadata on a signature that is otherwise purely about presentation.
2. **Wrap the error in a `FeatureGateError` at each gated call site** — tag the gate where the request is issued. Disadvantage: the single `refineClientError` site re-extracts a fresh `*ProblemError` from the wrapped `*ResponseError` and would *discard* an outer wrapper (015 ADR-4) — the tag would be silently lost; making it survive means reworking 015's refinement. (Rejected for the same reason in 060 ADR-1.)
3. **Add `Method` + `Path` to `ResponseError`, set in `Execute`** — the request (`req.Method`, `req.Path`) is in scope exactly where `Execute` builds the `*ResponseError`. The fields ride the error through `ProblemError` (which wraps it) to `Diagnose` with zero call-site changes; `Diagnose` reads them via `errors.As(err, &responseErr)` even when `err` is the refined `*ProblemError` (it unwraps to the `ResponseError`).

**Decision**: Option 3. Enrich `ResponseError` at the source. The threading becomes two additive struct fields and one assignment in `Execute`; the entire downstream chain — `refineClientError`, `Diagnose`, `reportFailure`, all ~100 call sites — is untouched.

**Consequences**: `ResponseError` now also describes *which operation* failed, not just *how* — a small widening of its role (it already carries status/headers/body for 015/017 to interpret; method/path are the request-side complement). `Error()` is unchanged, so every existing message and golden test stays byte-identical; the new fields zero-value to `""` so any `ResponseError` built elsewhere (none today) degrades to `GateNone`. `Diagnose` gains a second `errors.As` to reach the wrapped `ResponseError` for method/path — the same pattern `errorEnvelopeFor` already uses to reach the status/body. Recognition stays agnostic to the threading: it takes primitives (060 ADR-1), so this choice doesn't bind it.

### ADR-2: Refine the plan-limit `403` centrally in `Diagnose`, 054-shaped — category and exit code unchanged

**Context**: 057, 058, and 060 all explicitly reserved the Premium `403` to be refined "centrally the way 054 refined the `412` for all writes — NOT per-command" (DECISIONS.md §396/§402). 054 added `StaleWrite` by classifying `412` *status-only* at the central `categoryForStatus`/`nextStepForStatus` sites. A plan-gate `403` differs in one way (060 ADR-1): it is **not** status-only — every `403` is `PermissionError` today and only the *gated operations* are plan-limits — so the central refinement needs the operation dimension 060 supplies, but the *site* is the same central one.

**Options considered**:
1. **Per-command `403` interception** — each gated command (`proposal create`/`propose`/`withdraw`/`respond`) catches its own `403` and renders plan-limit wording. Disadvantage: scatters the wording across four+ commands, drifts from the landed "no status interception, route through the shared `reportFailure`" stance every proposal write took (057/058/060), and re-fires the per-command hazard the central chokepoint exists to avoid.
2. **Central refinement in `Diagnose`** — in the existing `*ProblemError` arm, when the status is `403` and `RecognizeFeatureGate` (on the wrapped `ResponseError`'s method/path) returns a non-`None` gate, replace `Cause`/`NextStep` with the plan-limit wording and record the gate; otherwise fall through to today's generic permission diagnostic. Category stays `PermissionError`.

**Decision**: Option 2. One edit in `Diagnose`'s `*ProblemError` arm covers every gated operation — including `withdraw`/`responses`, whose commands may not all be built yet, because recognition keys on the operation, not the command (060 ADR-2). **The `403`→`PermissionError` mapping and the exit code are unchanged**: this refines only the human-facing `Cause` and `NextStep` (and the carried gate), exactly as the spec's "legibility polish, category/exit-code unchanged" requires. `categoryForStatus(403)` is *not* touched.

**Consequences**: Completes the 054/060 template — an operation-aware refinement split into a tested recognizer (060) + a central diagnostic-site edit (here). The branch lives at the single point where `Diagnose` already has the refined error, so the cause/category/next-step still come from one matched value and cannot drift (the invariant `Diagnose`'s doc-comment protects). A non-gated `403`, and any non-`403` on a gated op, are provably untouched (recognition returns `GateNone`).

### ADR-3: Carry the recognized gate on the `Diagnostic`, classified once; 061 maps the gate kind to its display name and possibility-framed wording

**Context**: Both renderers must present the plan-limit failure consistently (human line on stderr; structured envelope on stdout), and 060's `Gate` is a *code-free enum kind* (`GatePremiumAsyncProposals`, `GateAIIntegration`) — not a human string. Someone must (a) decide where the gate lives so both renderers read one value, and (b) map the kind to its human-meaningful name and the possibility-framed wording. The spec's assumption is that "the gate's human-meaningful name comes from recognition" — i.e. it is *determined by which gate recognition identified* — with the exact wording an interface-level detail.

**Options considered**:
1. **Re-run `RecognizeFeatureGate` inside `errorEnvelopeFor`** — let the structured renderer recognize independently. Disadvantage: two classification sites that can disagree; violates the landed "`Diagnose` computes once, both renderers read the one `Diagnostic`" contract (`errorenvelope.go`).
2. **Add a `Feature` field to `Diagnostic`, set once by `Diagnose`** — `Diagnose` records the recognized gate's display name on the `Diagnostic`; `renderDiagnostic` weaves it into the `Cause` prose (no new human field needed — human output is just cause-plus-next-step), and `errorEnvelopeFor` maps it to the distinct envelope element. The kind→display-name + wording mapping is a small table owned **here** (061's rendering concern), keeping 060 a pure code-free classifier.

**Decision**: Option 2. The recognized gate becomes part of the one normalized `Diagnostic`, classified exactly once at the central site. 061 owns the `Gate`-kind→display-name (`GatePremiumAsyncProposals` → "Premium async proposals") and the possibility-framed wording; 060 stays code-free. This reconciles the spec assumption: the *name is determined by* recognition's output (the identified gate), even though the human string literal lives in 061's rendering layer.

**Wording (possibility, never certainty — spec decision 1A, 060 ADR-4)**: the `Cause` states the operation *may not be* available on the organization's plan, names the gating feature, and notes the same `403` may instead be a permission issue; the `NextStep` points the caller to **verify** the plan includes that feature, never to upgrade as a certain remedy. No plan name, price, or upgrade URL is fabricated (CONSTITUTION VIII) — the display name is the only added specific, and it comes from the recognized gate.

**Consequences**: A future gate kind needs one display-name/wording entry here; a guard test (PR #10 exhaustiveness shape) asserts every non-`None` gate kind has a display name so the mapping can't silently drop one. The human line gains no structural field — the gate is in its prose; only the structured envelope adds a distinct element (ADR-4).

### ADR-4: Surface the gate as an `omitempty` distinct element of 018's error envelope

**Context**: Under `json`/`yaml`, the spec requires the gating feature as its **own distinct, parseable element** — not folded *only* into the prose `message` (which also names the gate) — so an agent reads it programmatically. 018's `ErrorDetail` carries `message`/`next_step`/`kind`/`status`/`body`; 032 already extended it once (the `next_step` key) following exactly this pattern.

**Options considered**:
1. **Fold the gate into `message`** — no schema change. Disadvantage: violates the spec's distinct-element requirement; an agent would have to parse the gate back out of prose.
2. **Add a `feature` field to `ErrorDetail`, `omitempty`** — a new key carrying the display name, absent for every non-plan-limit failure (so the shared envelope shape is preserved — the field that doesn't apply is simply absent, never null-keyed), populated by `errorEnvelopeFor` from the `Diagnostic`'s carried gate.

**Decision**: Option 2. Mirror the landed `next_step` extension: declare the field in `internal/output` (018's home), populate it in `internal/cli`'s `errorEnvelopeFor` (keeping `internal/output` classification-free). The exact JSON key (`feature` is the working name) is pinned at the interface step alongside the envelope shape.

**Consequences**: The envelope grows one optional key; existing failures render byte-identically (the key is absent under `omitempty`). The field name is the only new structural contract — pinned in interface-spec. YAML key order is unaffected (RenderError emits YAML keys alphabetically through a map; rely on the key, not its position — `error.go`).

---

## Cross-cutting Concerns

**Error handling**: The added path is total and side-effect-free. `RecognizeFeatureGate` never errors (060 ADR-1); a `GateNone` result (non-gated op, or a `403` that doesn't match) falls through to today's exact generic permission diagnostic. `Diagnose` stays total and never panics.

**Secret hygiene (CONSTITUTION II)**: The new inputs are the request **method** and **path** — request metadata, never the `X-Auth-Token` (which rides 007's `AuthTransport`, not `Request`/`ResponseError`). The display name and wording are static literals. No new surface can leak the token; the envelope stays secret-free by construction.

**Backward compatibility**: All struct changes are additive (`ResponseError.Method/Path`, `Diagnostic.Feature`, `ErrorDetail.Feature`) and zero-value to `""`/absent. Every non-plan-limit failure — and every existing test and golden projection — renders byte-identically. `Error()` strings are unchanged.

**Testing strategy**: BDD `.feature` scenarios from the spec drive the failure path at the `Diagnose`/render boundary (the diagnostic_normalization / output-aware-failure-rendering BDD shape). Unit tests: each of the four gated operations on a `403` → plan-limit cause naming the gate + verify-plan next step + `Feature` set + category still `PermissionError`; a non-gated `403` → unchanged generic permission diagnostic, no `Feature`; a non-`403` on a gated op → unchanged; the human line carries the gate in prose with no certainty claim; the structured envelope carries `feature` as a distinct key and omits it for non-plan-limit failures; the gate-kind→display-name exhaustiveness guard (PR #10 `len`+comma-ok shape). The possibility framing is asserted (no "upgrade required" / no certain-insufficiency wording).

**Configuration**: None. The gated set is 060's static registry; the display-name/wording table is static literals — not configurable, not fetched (consistent with "Spec is the contract" and the no-live-probe non-behavior).

---

## Implementation Strategy

**Single phase, unblocked — 060 has landed.** This feature consumes `RecognizeFeatureGate` + the `Gate` type from Feature-Gate Recognition (060), now **merged to `main`** (#142): `RecognizeFeatureGate(method, path string, status int) Gate`, the `GateNone`/`GatePremiumAsyncProposals`/`GateAIIntegration` kinds, and `Gate.String()` (kebab-case, for logs) are in `internal/apiclient`. 061's display name is the human-prose form ("Premium async proposals"), distinct from `String()`'s `"premium-async-proposals"`.

The work, naturally one or two PR-sized units the tasks skill will decompose:
1. **Threading** — add `Method`/`Path` to `ResponseError`; set them in `Execute` (`apiclient`).
2. **Central refinement** — add the `Feature` field to `Diagnostic`; in `Diagnose`'s `*ProblemError` arm, recognize a gated `403` (reading the wrapped `ResponseError`'s method/path) and refine `Cause`/`NextStep`/`Feature` on a non-`None` gate; add the gate-kind→display-name + wording mapping with its exhaustiveness guard (`cli`).
3. **Structured surface** — add the `omitempty` `feature` field to `ErrorDetail` (`output`); map it in `errorEnvelopeFor` (`cli`).

These are tightly coupled (the field flows source→classifier→envelope) and may ship as one PR; the tasks skill may split (1) from (2)+(3) if it eases review.

---

## Risks

- **060 dependency — landed, resolved** (was: sequencing risk): 061 depends on 060's recognizer, which has **merged to `main`** (#142). The consumed contract (`RecognizeFeatureGate(method, path, status) Gate`, the `Gate` kinds, `Gate.String()`) is as 060's plan fixed it, so 061 is unblocked with no remaining sequencing risk.
- **`ResponseError` field-widening reaches a broad consumer set** (possible, low impact): `ResponseError` is read by 015/017 and the envelope. Mitigation: fields are additive and zero-valued; `Error()` unchanged; existing tests prove byte-identical behavior for every non-plan-limit failure.
- **Display-name / wording drift from 060's `Gate` enum** (possible, low impact): the kind→name table is hand-maintained against 060's enum. Mitigation: the exhaustiveness guard test asserts every non-`None` gate kind has a display name, so a new kind without one fails loud (PR #10 shape).
- **Over-asserting wording** (possible, medium impact if wrong): the diagnostic must never tell a permissioned caller to upgrade. Mitigation: spec decision 1A + 060 ADR-4 fix the possibility framing; a validation scenario asserts no certainty/upgrade language; the wording is a static literal reviewed once.

---

## What This Plan Does Not Cover

- **The recognizer and its registry** — owned by Feature-Gate Recognition (060); this feature only consumes `RecognizeFeatureGate`.
- **The `ai_integration` gate's reachable wording** — `GateAIIntegration` is modeled-but-unregistered (060 ADR-3); no command reaches it, so 061 maps it for readiness but it produces no message today (spec edge case). When such a command lands, the display name is already in place.
- **Exact key/field/wording spellings** — `ErrorDetail`'s JSON key (`feature`), the `Diagnostic.Feature` field, and the cause/next-step strings are pinned at the interface step.
- **Output-format mechanics** — Output-Aware Failure Rendering (032) and Structured Serialization (018) own the rendering and envelope encoding; 061 shapes the diagnostic and lets them render it.
- **Executable scenario placement / task decomposition** — the scenarios and tasks skills.
