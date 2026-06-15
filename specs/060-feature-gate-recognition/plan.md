# Plan: Feature-Gate Recognition

**Feature**: 060-feature-gate-recognition
**Role**: Shaper
**Inputs**: spec.md (060), PROJECT.md, FEATURE-MODEL.md (Plan-Limit Signalling), DECISIONS.md (24 entries), DEPRECATION.md, LEARNINGS.md; grounded against `internal/apiclient/problem.go` (015), `internal/apiclient/execute.go` (010), `internal/cli/diagnostic.go` (031), `internal/cli/me.go` `reportFailure` (032), `internal/cli/exitcode.go` (004), and the spec's gate metadata in `spec/glassfrog-api-v5.yaml`.

---

## System Architecture

Feature-Gate Recognition is a **single internal component** — a pure recognizer over the spec's static gate metadata, with no command, no request, and no output of its own. It answers one question: *given the operation that failed and the HTTP status that came back, is this a 403 from a known plan/feature-gated operation?* It produces a code-free classification value naming the **suspected** gate; it renders nothing and changes no exit code.

**The gap it fills.** Today every non-2xx funnels through one shared failure site — `reportFailure` (032, `me.go`) → `refineClientError` (the single `*ResponseError`→`*ProblemError` site, 015 ADR-4) → `Diagnose` (031). `Diagnose` keys *purely on HTTP status*: `categoryForStatus(403)` → `PermissionError` (exit 4) and `nextStepForStatus(403)` → a generic "check the identity's role/permissions" hint. It has **no operation awareness**, so a plan-gated 403 (Premium async proposals not enabled) and an ordinary permission 403 (caller isn't a circle member) are indistinguishable. Recognition supplies exactly the operation-gate knowledge `Diagnose` lacks.

**What this feature delivers (and deliberately does not).** This feature builds the recognizer and its static registry, fully tested in isolation. It does **not** wire recognition into `Diagnose`/`reportFailure`, does not thread operation identity through the error chain, and does not alter any rendered message or exit code — that integration plus the gate-aware "not available on your plan" wording is **Plan-Limit Signal (#61)**, which is precisely why #61 (not #60) depends on Diagnostic Normalization (031). #60 depends only on API Error Extraction (015): it interprets the status that 015's typed error already carries. This split keeps every #60 non-behavior intact by construction — there is nothing downstream of the recognizer to change yet.

**Component shape** (placed in `internal/apiclient`, sibling to `problem.go` — recognition is API-response interpretation, the same domain as `ExtractProblem`):

- A `Gate` classification type — a code-free enum modeling the gate *kinds* (`GateNone`, `GatePremiumAsyncProposals`, `GateAIIntegration`).
- A **static gated-operation registry** — entries transcribing the spec's gate metadata: HTTP method + path template + the gate it carries. Only the reachable Premium async-proposal write family is registered (see ADR-3).
- A pure recognizer — `RecognizeFeatureGate(method, path string, status int) Gate` — returning the suspected gate when the status is `403` **and** the operation matches a registered gated entry; `GateNone` otherwise. A non-`403` status from a gated operation, or any status from an unregistered operation, returns `GateNone`.

**Data flow (this feature):** none at runtime — the recognizer is a leaf function exercised by its tests. **Data flow (once #61 lands):** the failed request's method + path reach the recognition call (a threading concern #61 owns), `RecognizeFeatureGate` returns a `Gate`, and #61 maps a non-`None` gate to a plan-limit diagnostic.

---

## Architecture Decisions

### ADR-1: Deliver recognition as a standalone, unwired pure capability — defer all integration to #61

**Context**: #60's spec is recognition-only: it must produce a classification but "render no message," "change no exit code," and "not alter the error chain" (Non-Behaviors). The natural live consumer is `Diagnose` (031), but teaching `Diagnose` the gate-aware branch is itself rendering work, and `Diagnose` is reached via `reportFailure`, which has only `err` in scope — not the request's method/path. The FEATURE-MODEL makes #61 (Plan-Limit Signal) depend on *both* #60 and Diagnostic Normalization (031); #60 depends only on API Error Extraction (015).

**Options considered**:
1. **Wire recognition into `reportFailure`/`Diagnose` now, inertly** — apply the recognizer at the failure site but render nothing different. Disadvantage: requires threading operation identity through the error chain (a #61 need), adds live-path code with no observable effect, and risks tripping the "don't alter the error chain" non-behavior for no benefit.
2. **Wrap the error in a `FeatureGateError` at each gated call site** — mirror the 015 `ProblemError`-wraps-`ResponseError` idiom. Disadvantage: `refineClientError` re-extracts a fresh `*ProblemError` from the wrapped `*ResponseError` and would *discard* an outer wrapper, so the recognition would be silently lost at the single refinement site; making it survive means reworking 015's refinement — scope creep for a Could-Have.
3. **Pure, unwired recognizer + static registry, tested in isolation** — deliver the recognition mechanism and its gate data as a leaf component; let #61 own the threading, the `Diagnose` integration, and the wording.

**Decision**: Option 3. #60 ships `Gate`, the registry, and `RecognizeFeatureGate`, with unit/BDD coverage proving correct classification across the gated set, the non-gated case, and the non-403 case. No call site changes; no error-chain changes.

This keeps the recognizer a genuinely independent, reusable foundation ("buildable now," per the BACKLOG) and honors every #60 non-behavior trivially — there is no rendering, exit-code, or error-chain surface in scope. The BDD scenarios exercise the recognizer at its own boundary (the recognition seam), not a full CLI invocation, consistent with how 015's `ExtractProblem` is verified.

**Consequences**: Recognition is correct and tested but not yet observable end-to-end from the CLI — that arrives with #61. #61 inherits a clear, small contract: source method/path/status at the render site, call `RecognizeFeatureGate`, branch the diagnostic on a non-`None` gate. The threading mechanism (enrich the error with the originating request vs. pass method/path alongside `err`) is left to #61, where it is actually needed (flagged in Risks).

This conforms to established precedent. The 057 and 058 decisions explicitly reserved the Premium `403` for "Plan-Limit Signalling, which will refine the 403 *centrally the way 054 refined the 412* for all writes — NOT per-command." 054 added `StaleWrite` by classifying `412` at the central diagnostic/exit-code sites *status-only* ("the 412 maps to StaleWrite regardless of the command"). The decisive difference — and the reason #60 exists as its own capability — is that a plan-gate `403` is **not** status-only: every `403` is `PermissionError` today, and only the *gated operations* are plan-limits. #60 supplies exactly the operation dimension that 054-style status-classification lacks; #61 then performs the central, 054-shaped site refinement using #60's recognizer. Threading operation identity to that central site is the new cost #61 takes on (054's `412` needed only status, so it had no such cost).

### ADR-2: Key recognition on HTTP method + path template, matched against a static registry

**Context**: Recognition must identify *which* operation produced a 403 using static spec metadata, not the response body (Non-Behavior: the 403 body is not self-identifying). The signals statically available about an operation are its HTTP method and path; `apiclient.Request` already carries `Method` and `Path`, and the spec's gate metadata is documented per path/operation.

**Options considered**:
1. **Per-call-site `Gate` declaration** — each gated command tags its own request with its gate. Disadvantage: scatters the gate knowledge across call sites, and (with ADR-1's no-call-site-change stance) there are no call sites to tag in #60 at all.
2. **`operationId` constants** — introduce a stable operationId per command and key the registry on it. Disadvantage: commands don't carry operationIds today; introducing and threading them is new surface the live consumer (#61) would also have to populate.
3. **Static `(method, path-template)` registry with segment-wise matching** — transcribe the spec's gate metadata as `{method, pathTemplate, gate}` rows; match a concrete path against a template treating `{segment}` as a single-segment wildcard, ignoring query.

**Decision**: Option 3. The registry is the literal, self-documenting transcription of the spec's gate metadata, and `method`+`path` are exactly what a caller already has. The four Premium async-proposal entries:

| Method | Path template | Gate |
|---|---|---|
| POST | `/proposals` | Premium async-proposals |
| POST | `/proposals/{proposal_id}/propose` | Premium async-proposals |
| POST | `/proposals/{proposal_id}/withdraw` | Premium async-proposals |
| POST | `/proposals/{proposal_id}/responses` | Premium async-proposals |

Matching compares the method, then path segments pairwise, where a `{…}` template segment matches any one concrete segment and a literal segment must match exactly; segment count must be equal. Query strings are ignored.

**Consequences**: Adding a future gated operation is a one-row registry edit. Explicit per-operation rows (not a `POST /proposals*` prefix rule) prevent a future non-gated POST under `/proposals` from being mis-recognized. The `withdraw` and `responses` rows are registered now though their commands aren't built yet (Withdraw is BACKLOG #59, Response Recording is 058, in pipeline) — recognition is keyed to the operation, not the command, so it is correct the moment those commands issue the request. The matcher is a small bounded helper; its edge cases (trailing slash, segment count) are pinned by tests.

### ADR-3: Model both gate kinds, register only the reachable Premium async-proposal family

**Context**: The spec carries two relevant gate kinds: the Premium async-proposal write path (in scope — its commands exist or are in pipeline) and the `x-feature-gate: ai_integration` extension on 7 agent/skill operations (deferred per PROJECT scope — no CLI command reaches them). The spec settled this as "model the gate kind generally, but only the proposal writes are reachable today" (define decision A; spec edge-case scenario "a modeled ai_integration gate has no reachable command today").

**Options considered**:
1. **Premium-only** — omit `ai_integration` entirely until those endpoints land. Disadvantage: the recognizer can't name the `ai_integration` gate even in principle; a later command would force a type change, not just a registry edit.
2. **Model both kinds; register both operation sets** — transcribe the 7 `ai_integration` paths too. Disadvantage: registers deferred, out-of-scope surface that no caller can ever hit today, contradicting PROJECT scope and adding rows that can't be exercised end-to-end.
3. **Model both kinds; register only the reachable Premium family** — `GateAIIntegration` exists in the type, but the registry holds only the four Premium async-proposal rows.

**Decision**: Option 3. `Gate` includes `GateAIIntegration` so the recognizer is *ready* to name that gate, but no registry row references it. When an `ai_integration`-gated command later lands (the PROJECT "Deferred" condition), recognition extends by adding registry rows — no type change.

**Consequences**: The recognizer's vocabulary matches the spec's gate model; its *active* recognition matches the project's current scope. The "ready but unregistered" state is directly the spec's edge-case scenario, and is pinned by a test asserting no registered row carries `GateAIIntegration` today (so the boundary can't drift silently).

### ADR-4: Express possibility, not certainty

**Context**: Because the 403 body is not self-identifying, a genuine permission 403 on a gated operation is indistinguishable from a plan-gate 403 (spec Non-Behavior + edge-case scenario). Recognition keys only on operation + status, so it can only ever say a 403 *may be* a plan limit.

**Options considered**:
1. **Assert a recognized 403 *is* a plan limit** — simpler downstream wording. Disadvantage: would let #61 falsely tell a properly-permissioned caller to upgrade an already-sufficient plan.
2. **Encode possibility in the contract** — the recognizer's result means "this 403 is *consistent with* a known plan gate," never "confirmed"; documented on the function and the `Gate` type, and asserted by the edge-case test (a gated-operation 403 that is really a permission denial still returns the suspected gate, and the contract names it as possible).

**Decision**: Option 2. A non-`None` `Gate` is defined as a *suspicion*, not a verdict. The naming and doc comments carry this explicitly so #61 words the diagnostic as possibility ("may not be available on your plan") and never as certainty.

**Consequences**: #61 must phrase its diagnostic conditionally. The recognizer never needs to distinguish plan-gate from permission-denial 403s (it provably can't), so its contract stays honest and its tests assert the possibility framing rather than an impossible certainty.

---

## Cross-cutting Concerns

**Error handling**: The recognizer is total and pure — it takes primitives (`method, path, status`), never returns an error, never panics, and performs no I/O. An unrecognized or malformed input simply yields `GateNone`. This mirrors `ExtractProblem`'s total/pure discipline (015).

**Secret hygiene (CONSTITUTION II)**: The recognizer reads only method, path, and status — never the token, never the response body. No new surface can leak the `X-Auth-Token`.

**Testing strategy**: Table-driven unit tests over the matcher and recognizer: each gated row recognized on 403; each gated row *not* recognized on a non-403 (422, 412, 200); a non-gated operation (e.g. `GET /roles/{id}`, `POST` to an unrelated path) never recognized; path-template edge cases (segment count mismatch, wildcard substitution, query ignored); and the ADR-3 guard (no registered row is `GateAIIntegration`). BDD `.feature` scenarios from the spec map onto the recognizer's boundary. A change-detector test over the registry guards against silent loss of a gated row (per the LEARNINGS comma-ok / length-guard pattern for zero-valued map lookups).

**Configuration**: The gated set is hardcoded static metadata transcribed from the spec — not configurable and not fetched at runtime (consistent with "Spec is the contract" and the spec's no-live-probe Non-Behavior).

---

## Implementation Strategy

**Single phase.** One component, no dependencies beyond the already-shipped 015 surface it conceptually interprets. The work is: define the `Gate` type, define the static registry, implement the path-template matcher and `RecognizeFeatureGate`, and cover all three with tests. The tasks skill should produce one PR-sized unit (optionally splitting the matcher helper from the recognizer if it eases review, but they are naturally one change).

---

## Risks

- **Integration debt deferred to #61** (likely, low impact): #60 ships an unwired recognizer, so its value is unrealized until #61 threads operation identity to the render site and renders the diagnostic. Mitigation: ADR-1 documents the exact contract #61 consumes; #61's FEATURE-MODEL dependency on #60 makes the hand-off explicit. The recognizer is independently tested, so the foundation is proven even while dormant.
- **Threading method/path to the recognition call is non-trivial for #61** (possible, medium impact): `reportFailure`/`Diagnose` currently receive only `err`; the originating request's method/path are not in the error chain. #61 must either enrich `ResponseError`/`ProblemError` with the request or pass method/path alongside `err`. Mitigation: flagged here so #61 plans for it; #60's recognizer takes primitives precisely so it is agnostic to whichever threading approach #61 picks.
- **Path-template matcher fragility** (possible, low impact): a concrete path with a trailing slash, extra segment, or query could mis-match. Mitigation: explicit segment-count + segment-wise matching with query ignored, pinned by edge-case tests.
- **Registry drift from the spec** (possible, low impact): the gated set is hand-transcribed, so a future spec revision could add a gated operation the registry misses, or the `ai_integration` boundary could be crossed silently. Mitigation: explicit per-operation rows, the ADR-3 guard test, and a registry change-detector; the gate metadata's spec provenance is documented at the registry.

---

## What This Plan Does Not Cover

- **Wiring recognition into the live failure path, and the "not available on your plan" diagnostic** — owned by Plan-Limit Signal (#61), along with its dependency on Diagnostic Normalization (031) and the threading of operation identity to the render site.
- **Registering the deferred `ai_integration` agent/skill operations** — added when those commands land (PROJECT "Deferred" condition); #60 only models the gate kind.
- **Executable scenario placement** — the scenarios skill turns the spec's driving scenarios into `.feature` files at the recognizer's boundary.
- **Task decomposition** — the tasks skill breaks the single phase into the concrete implementation unit.
- **Protocol/interface contracts** — none: this feature has no external-facing or specification boundary (no command, no API call, no output, no declarative artifact), so the interface skill has nothing to design here.
