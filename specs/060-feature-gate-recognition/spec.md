# Specification: Feature-Gate Recognition

**Feature**: 060-feature-gate-recognition
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Feature-Gate Recognition is the capability that answers *why* a `403` came back: was it an ordinary permission denial, or was the operation simply **not available on this organization's plan**? It pairs with the Next-tier **Unsignalled Plan Limits** problem — plan/feature-gated endpoints reject with `403` and no clear "not available on your plan" signal. The Glassfrog API serves `403` as a generic RFC 9457 Problem Details document with **no structured field** marking it as a plan-gate (confirmed: the typed error from API Error Extraction (015) carries the authoritative HTTP status, the RFC 9457 type/title/detail members, and provenance metadata — but no field that identifies a plan-gate). So gate-awareness cannot come from the response body — it has to come from **static knowledge of which operation was called**, drawn from the spec's own gate metadata.

It is deliberately narrow — it *recognizes* a suspected plan-gate, nothing more. It takes the typed API error that API Error Extraction (015) already produces, checks whether the failing operation is one the spec marks as plan/feature-gated (the Premium async-proposal write path; the `x-feature-gate: ai_integration` extension is modeled but no command reaches it today), and — only on a `403` — tags the failure as a *possible* plan-limit rejection naming the suspected gate. Because the body is not self-identifying, recognition expresses **possibility, never certainty**: a genuine permission `403` on a gated operation is indistinguishable from a plan-gate `403`, and recognition does not pretend otherwise. It does not render any user-facing message — turning the recognized rejection into an actionable "not available on your plan" diagnostic is the downstream **Plan-Limit Signal** (#61). Its job is to make a plan-gate `403` distinguishable from a generic one.

---

## Behavioral Accord

### Recognizing a plan-gated rejection

- When the typed API error came from an operation the spec marks as plan/feature-gated **and** carries a `403` status, the system recognizes it as a *possible* plan-limit rejection and names the suspected gate (Premium async proposals, or `ai_integration`).
- When recognition fires, it expresses possibility, not certainty — it marks the rejection as one that *may be* a plan limit, never as a confirmed one, because the `403` body cannot distinguish a plan gate from a genuine permission denial.

### Declining to recognize

- When the failing operation is **not** in the known plan-gated set (e.g. a governance read, an identity read), the system does not recognize it as a plan-limit rejection — it stays a generic failure, whatever its status.
- When a plan-gated operation fails with a status **other than** `403` (e.g. a `422` validation rejection, a `412` stale-write refusal), the system does not recognize it as a plan-limit rejection — only `403` carries the plan-gate meaning.

### Independence from the response body

- When recognizing, the system keys solely on the operation's static gate metadata and the HTTP status; it does not inspect the response body to decide whether a `403` is a plan-gate, since the body is not contractually self-identifying.

### Producing only a classification

- When recognition fires, the system produces only a classification of the existing failure (the suspected-plan-limit marker and the named gate), carried alongside the typed error; it renders no user-facing message and changes no exit code — those remain the consuming command's and the downstream signal's concern.

---

## User Scenarios

**In order to** be told "this isn't on your plan" instead of a bare permission error when a proposal write is rejected,
**as a** practitioner whose AI agent drives the CLI,
**I want to** have plan-gated `403`s recognized as distinct from ordinary permission denials.

**In order to** word a plan-limit diagnostic that names the right gate and never falsely tells someone to upgrade,
**as** the downstream Plan-Limit Signal capability,
**I want to** receive a recognized rejection tagged as a *possible* plan limit with the suspected gate named.

---

## Non-Behaviors

- The system must not render a user-facing "not available on your plan" message or any diagnostic text. **Why**: rendering the recognized rejection into actionable output is the downstream Plan-Limit Signal (#61); collapsing recognition and rendering into one capability would couple the two and make the classification untestable on its own.
- The system must not assert certainty that a `403` is a plan limit. **Why**: the `403` body is generic and a genuine permission denial on a gated operation is indistinguishable from a plan-gate; claiming certainty would risk telling a properly-permissioned user to upgrade a plan that is already sufficient.
- The system must not inspect the response body to identify a gate. **Why**: the API does not mark plan-gates in the body, so body heuristics would be unreliable and would drift from the spec's authoritative gate metadata.
- The system must not call the API to probe plan status. **Why**: recognition is a static interpretation of a `403` that already arrived; an extra live capability probe would add latency, cost a request against the rate limit, and exceed the narrow recognition concern.
- The system must not register the deferred `ai_integration` agent/skill operations or the out-of-scope `restricted (non-premium)` custom-field/tag operations as reachable gated commands. **Why**: PROJECT scope defers the agent/skill endpoints and excludes actor/admin operations; no CLI command calls them, so recognizing them would describe behavior the CLI cannot reach.
- The system must not change the HTTP-status-to-exit-code mapping or alter the error chain. **Why**: API Error Extraction (015) and the consuming command own status authority and exit codes (producer-classifies / consumer-maps); recognition is an additive classification, not a remapping.

---

## Integration Boundaries

- **API Error Extraction (015)** *(upstream)*: recognition reads the typed API error it produces — the authoritative `403` status and the operation that failed. It adds a classification; it does not replace or re-wrap the status-authority contract.
- **Static spec gate metadata** *(reference)*: the set of plan-gated operations is derived from the published spec — the Premium-documented async-proposal write operations ("requires async proposals … Returns 403") and the `x-feature-gate: ai_integration` vendor extension — baked in, not fetched at runtime.
- **Plan-Limit Signal (#61)** *(downstream)*: consumes the recognized classification (suspected-plan-limit marker + named gate) to render the actionable diagnostic. Not built here; recognition is its prerequisite.
- **Glassfrog API** *(system actor)*: the origin of the `403`. Recognition never contacts it — it interprets a response that already arrived.

---

## Driving Scenarios

### Happy path

**Scenario: Advancing a draft on a non-Premium org is recognized as a plan limit**
Given the CLI advances a draft proposal into circulation (a Premium async-proposal operation)
When the API rejects it with `403`
Then the failure is recognized as a *possible* plan-limit rejection
And the suspected gate is named as Premium async proposals.

**Scenario: Creating a proposal on a non-Premium org is recognized as a plan limit**
Given the CLI creates a proposal from a tension (a Premium async-proposal operation)
When the API rejects it with `403`
Then the failure is recognized as a *possible* plan-limit rejection naming Premium async proposals.

**Scenario: Recording a response on a non-Premium org is recognized as a plan limit**
Given the CLI records a circle member's response (a Premium async-proposal operation)
When the API rejects it with `403`
Then the failure is recognized as a *possible* plan-limit rejection naming Premium async proposals.

### Error scenarios

**Scenario: A 403 from a non-gated read is not a plan limit**
Given the CLI reads a role (an operation with no plan gate)
When the API rejects it with `403`
Then the failure is not recognized as a plan-limit rejection
And it remains a generic permission denial.

**Scenario: A non-403 failure from a gated operation is not a plan limit**
Given the CLI creates a proposal (a Premium async-proposal operation)
When the API rejects it with `422` because a change is invalid
Then the failure is not recognized as a plan-limit rejection.

### Edge cases

**Scenario: A genuine permission denial on a gated operation is still flagged as possible**
Given the CLI advances a draft proposal (a Premium async-proposal operation)
And the caller lacks permission for a reason unrelated to the plan
When the API rejects it with `403`
Then the failure is recognized as a *possible* plan-limit rejection, not a confirmed one
And recognition does not claim certainty about the cause.

**Scenario: Recognition ignores body content when identifying the gate**
Given a gated operation is rejected with `403`
And the response body carries a `detail` describing some other cause
When recognition runs
Then it keys on the operation's static gate metadata and the status alone
And the body content does not change whether the gate is recognized.

**Scenario: A modeled ai_integration gate has no reachable command today**
Given the `ai_integration` gate kind is modeled in recognition
And no CLI command maps to an `ai_integration`-gated operation
When commands run
Then no invocation triggers `ai_integration` recognition today
And recognition is ready to name that gate if such a command later lands.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The capability names no user-facing diagnostic wording**
Given the recognition capability's surface
When it is inspected
Then it produces only a classification (a suspected-plan-limit marker and a named gate)
And it contains no rendered "not available on your plan" message — that belongs to Plan-Limit Signal.

**Scenario: Possibility is expressed everywhere recognition is described**
Given every place recognition surfaces its result
When the result is read
Then it is framed as a *possible* / suspected plan limit
And nowhere claims a `403` is certainly a plan limit.

---

## Assumptions

- **The call site knows its operation identity**: recognition can associate a failing call with the spec operation that produced it. ([ASSUMED] — informed by the CLI's structure where each command maps to one spec operation; if operation identity is not reachable at the recognition point, the mechanism for supplying it is an architecture concern for the plan.)
- **The gated set explicitly enumerates the Premium async-proposal write operations, including those not yet built**: recognition registers each Premium async-proposal write operation by identity, and that registry already includes the operations whose commands have not yet landed, so each is covered the moment its command issues the request — without a later edit to this capability. ([ASSUMED] — Withdraw Proposal is a Premium async-proposal operation not yet specified or built (BACKLOG #59); because recognition keys on the operation rather than the command, its pre-registered entry covers it on arrival. This is the per-operation registry the plan settles in ADR-2 — not generic family matching, which would wrongly imply a brand-new, unregistered operation is auto-covered.)
- **Gate metadata is static, not fetched**: the plan-gated set is derived from the published spec at build time, not by querying plan status at runtime. (Technical default — consistent with "Spec is the contract" and avoids a live capability probe.)

---

## Ambiguity Warnings

_None. The recognition surface, the gated set, the possibility-not-certainty framing, and the #60/#61 boundary were settled in conversation._
