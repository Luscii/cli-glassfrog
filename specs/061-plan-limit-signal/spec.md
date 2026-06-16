# Specification: Plan-Limit Signal

**Feature**: 061-plan-limit-signal
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Plan-Limit Signal is the **rendering half** of **Plan-Limit Signalling** (problem: *Unsignalled Plan Limits* — plan/flag-gated endpoints reject with `403` and no clear "not available on your plan" signal). It pairs with Feature-Gate Recognition (060), which does the *recognizing*: 060 takes the typed API error and, only on a `403` from a known plan-gated operation, tags the failure as a **possible** plan-limit rejection naming the suspected gate (Premium async proposals) — but renders no message. The two sibling diagnostic capabilities deliberately decline this work too: Diagnostic Normalization (031) and Output-Aware Failure Rendering (032) both explicitly refuse to "translate a `403` into plan/Premium availability guidance," leaving that to *this* problem. This capability closes that gap.

It turns 060's recognized classification into an **actionable diagnostic**: a cause that says the operation may not be available on the organization's plan and **names the gating feature**, plus a next step pointing the caller to verify the plan. Crucially, it produces this in the **same normalized diagnostic shape** every other failure uses (cause, category, next step), so the existing failure renderer (032) carries it to the selected `--output` format with no special-casing — which is why this capability depends on Feature-Gate Recognition (060) and Diagnostic Normalization (031), not on 032. It is legibility polish, not a remapping: the category stays **permission/authorization** and the process exit code is unchanged. And it honors 060's defining constraint — recognition expresses **possibility, never certainty** — so its wording never falsely tells a properly-permissioned caller to upgrade.

---

## Behavioral Accord

### Shaping a recognized plan-limit rejection into an actionable diagnostic

- When a failure has been recognized as a *possible* plan-limit rejection (a typed `403` on a known plan-gated operation, tagged by Feature-Gate Recognition with the suspected gate named), the system produces a diagnostic whose cause states the operation may not be available on the organization's plan and **names the gating feature**, and whose next step points the caller to confirm the plan includes that feature.
- The system produces this diagnostic in the same normalized shape every other failure uses — a cause, a category, and a next step — so the existing failure renderer presents it in the selected output format without a dedicated rendering path.

### Keeping the framing honest — possible, not certain

- When wording the cause, the system frames the plan limit as **possible**, not certain — it surfaces that the same `403` may instead be an ordinary permission denial, because a `403` cannot distinguish the two. It never states that the organization's plan is definitely insufficient.
- The system never instructs the caller to upgrade or pay as the certain, sole remedy. The next step is to **verify** the plan includes the gating feature (and, by extension, that the caller's permissions are sufficient), not an assertion that an upgrade is required.

### Naming the gate without inventing specifics

- When recognition named a gate, the diagnostic names **that same gate** in human-meaningful terms (e.g. "Premium async proposals"); it does not name a gate recognition did not identify, nor downgrade it to a generic "a feature."
- The system does not fabricate a plan name, a price, an upgrade URL, or any remedy detail the spec does not supply; the next step stays a verifiable action (confirm the plan includes the named feature), never an invented path.

### Rendering across the selected output format

- When the resolved output format is a human format (`full`/`compact`), the recognized plan-limit diagnostic appears as the cause-plus-next-step line, on the same channel and in the same form as every other rendered failure.
- When the resolved output format is a structured format (`json`/`yaml`), the diagnostic is carried in the unified error envelope, and the **named gate appears as its own distinct, parseable element** — not folded *only* into the prose cause (which also names the gate) — so an agent can read which feature is gated programmatically without parsing prose.
- The plan-limit diagnostic carries the same facts in every format (the plan-limit cause, the named gate, the next step). Selection changes only how the failure is presented, never which facts it contains.

### Staying in its lane

- The system acts only on failures Feature-Gate Recognition has tagged as possible plan limits; a `403` (or any failure) recognition has **not** tagged is left to the generic diagnostic unchanged — no plan-limit wording leaks onto it.
- The system keeps the failure's category **permission/authorization** and does not change the process exit code; it replaces the cause and next step for the recognized case but never remaps the category or emits a code.
- The system does not recognize plan gates itself, inspect the response body, or call the API to check plan status; it consumes the classification recognition already produced from static gate metadata and the HTTP status.

---

## User Scenarios

**In order to** be told "this may not be on your plan" and what to check, instead of a bare permission error when a proposal write is rejected,
**as a** practitioner whose AI agent drives the CLI,
**I want** a recognized plan-limit `403` surfaced as an actionable diagnostic that names the gating feature.

**In order to** branch on a plan limit programmatically rather than parse it out of prose,
**as an** AI agent operating the CLI with `--output json`,
**I want** the gating feature surfaced as a distinct, parseable element of the failure envelope.

**In order to** trust the signal and not be sent to upgrade a plan that is already sufficient,
**as an** operator (human or agent),
**I want** the plan-limit wording framed as a possibility, never a certain insufficiency.

---

## Non-Behaviors

- The system must not assert certainty that the plan is insufficient, nor instruct the caller to upgrade as the sole, certain remedy. **Why**: 060 establishes that a genuine permission `403` is indistinguishable from a plan gate; telling a correctly-permissioned caller to upgrade would be confidently wrong (CONSTITUTION VIII) and could send them down a needless, paid path.
- The system must not recognize plan gates itself or re-derive the gated set. **Why**: Feature-Gate Recognition (060) is the single recognizer; duplicating its gate metadata here would let the two drift, and recognition deliberately keeps the gated set in one place.
- The system must not fabricate a plan name, price, upgrade URL, or any remedy detail the spec does not supply. **Why**: CONSTITUTION VIII (No Fabricated Data) — an invented upgrade path rendered as clean output is more dangerous than an honest "verify your plan includes this feature."
- The system must not change the failure's category or the process exit code. **Why**: this is legibility polish; Exit-Code Convention (004) is the single category→code mapper and Diagnostic Normalization (031) owns the category — remapping here would risk two paths disagreeing on the code for one `403`.
- The system must not implement the output-format rendering or define the error envelope's shape. **Why**: Output-Aware Failure Rendering (032) renders failures per format and Structured Serialization (018) owns the envelope and encoders; this capability produces a normalized diagnostic and lets 032 render it, so a second renderer cannot drift from the first.
- The system must not inspect the response body or call the API to confirm plan status. **Why**: recognition already keyed on static gate metadata and the HTTP status; a live probe would add latency, cost a request against the rate limit, and the `403` body is not contractually self-identifying anyway.
- The system must not produce a plan-limit message for a gate no command reaches today (the modeled `ai_integration` gate). **Why**: PROJECT scope defers the agent/skill endpoints and no CLI command triggers that gate; messaging an unreachable gate would describe behavior the CLI cannot produce.

---

## Integration Boundaries

- **Feature-Gate Recognition (060)** *(upstream)*: supplies the recognized classification — the *possible* plan-limit marker and the named gate — riding on the typed `403` API error. This capability reads that classification; it never re-recognizes or re-derives the gated set.
- **Diagnostic Normalization (031)** *(peer / upstream)*: owns the one normalized diagnostic shape (cause, category, next step) and deliberately declines plan interpretation. This capability supplies the plan-limit cause and next step for the recognized case, in that same shape, keeping the permission/authorization category.
- **Output-Aware Failure Rendering (032)** *(downstream renderer)*: renders the diagnostic in the resolved `--output` format; the distinct gate element rides in 018's envelope under the structured formats. This capability produces the value; 032 renders it, which is why this capability does not depend on 032 directly.
- **Structured Serialization (018)** *(reference)*: provides the unified error envelope that carries the named-gate element. This capability maps the gate into the envelope; 018 owns the envelope shape and the encoders.
- **Exit-Code Convention (004)** *(unchanged)*: the category stays permission/authorization, so the exit code is identical to a generic `403`. This capability touches neither the category nor the code.

---

## Driving Scenarios

### Happy path

**Scenario: Advancing a draft on a non-Premium org surfaces an actionable plan-limit diagnostic**
Given the CLI advances a draft proposal into circulation (a Premium async-proposal operation)
And the failure was recognized as a possible plan-limit rejection naming Premium async proposals
When the diagnostic is produced under the default `full` format
Then the cause states the operation may not be available on the organization's plan and names Premium async proposals
And the next step points the caller to confirm the plan includes Premium async proposals.

**Scenario: Creating a proposal surfaces the gate, framed as a possibility**
Given the CLI creates a proposal from a tension (a Premium async-proposal operation)
And the failure was recognized as a possible plan-limit rejection naming Premium async proposals
When the diagnostic is produced
Then it names Premium async proposals as the gating feature
And it frames the plan limit as possible, noting the same `403` may instead be a permission denial.

**Scenario: The gating feature is a distinct, parseable element under json**
Given a recognized plan-limit rejection naming Premium async proposals
When the diagnostic is rendered with `--output json`
Then the unified error envelope carries the named gate as its own distinct, parseable element
And the gate name is not folded only into the prose cause text (an agent can read it without parsing prose).

### Error scenarios

**Scenario: A 403 that was not recognized keeps the generic diagnostic**
Given the CLI reads a role and the API rejects it with `403`
And the failure was not recognized as a plan-limit rejection
When the diagnostic is produced
Then it is the generic permission/authorization diagnostic
And no plan-limit wording and no gate name appear.

**Scenario: A non-403 failure on a gated operation gets no plan-limit wording**
Given the CLI creates a proposal (a Premium async-proposal operation)
And the API rejects it with `422` because a change is invalid (so recognition did not fire)
When the diagnostic is produced
Then it carries no plan-limit wording and names no gate.

### Edge cases

**Scenario: A genuine permission denial on a gated operation is still hedged, never asserted**
Given the CLI advances a draft proposal (a Premium async-proposal operation)
And the caller actually lacks permission for a reason unrelated to the plan
And recognition flagged it as a possible plan-limit rejection (it cannot tell the two apart)
When the diagnostic is produced
Then the wording frames the plan limit as possible and notes it may be a permission issue
And it never states the plan is certainly insufficient nor instructs the caller to upgrade.

**Scenario: The exit code is unchanged by the plan-limit wording**
Given the same recognized `403`, rendered once under `full` and once under `json`
When each invocation terminates
Then both terminate with the permission/authorization exit code under Exit-Code Convention (004)
And only the rendered presentation differs.

**Scenario: A modeled ai_integration gate produces no message today**
Given the `ai_integration` gate kind is modeled in recognition
And no CLI command maps to an `ai_integration`-gated operation
When commands run
Then no invocation produces an `ai_integration` plan-limit message today
And the capability is ready to name that gate if such a command later lands.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Possibility is preserved everywhere the signal renders**
Given every place the capability surfaces a plan-limit diagnostic
When each is read
Then it frames the plan limit as possible and names the gate
And nowhere asserts the plan is certainly insufficient nor instructs the caller to upgrade as the sole remedy.

**Scenario: No fabricated remedy detail in the rendered diagnostic**
Given a recognized plan-limit rejection
When the diagnostic is rendered under each format
Then it names no plan price, no upgrade URL, and no plan name beyond the gating feature recognition supplied
And the next step is a verifiable action, not an invented path.

**Scenario: No implementation leakage in the artifact**
Given the produced specification
When it is reviewed
Then it names only observable behavior (the diagnostic's facts, the gate element, which format renders where) and prescribes no language, type system, or internal data layout.

---

## Assumptions

- **The signal rides the existing diagnostic pipeline**: this capability produces a value in Diagnostic Normalization's (031) normalized shape so Output-Aware Failure Rendering (032) renders it in the selected format with no dedicated rendering path — which is why the dependency is 060 + 031, not 032. ([ASSUMED] — informed by the 031/032 boundary, where 031 produces the diagnostic value and 032 renders it; if the pipeline cannot carry a plan-limit-specialized diagnostic, the mechanism for shaping it is an architecture concern for the plan.)
- **The named gate is a distinct element of 018's envelope**: under structured formats the gate is surfaced as its own parseable element of Structured Serialization's (018) unified error envelope, alongside the cause/`message` and the next-step element 032 added. (The exact field name is an interface-level detail pinned alongside 018's envelope shape, not a behavioral gap.)
- **The gate's human-meaningful name comes from recognition**: the gate string the diagnostic names (e.g. "Premium async proposals") is the human-meaningful name Feature-Gate Recognition (060) supplies; the exact wording is an interface-level detail, not invented here.

---

## Ambiguity Warnings

_None. The wording stance (possible, not certain — never "upgrade required"), the unchanged permission/authorization category and exit code, naming only the recognized gate without fabricated remedy detail, the distinct parseable gate element under structured formats, and the Premium-only-reachable / `ai_integration`-dormant gate scope were all settled in conversation._
