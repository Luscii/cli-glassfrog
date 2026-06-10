# Specification: Diagnostic Normalization

**Feature**: 031-diagnostic-normalization
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Diagnostic Normalization is the root of **Diagnostic Reporting** (problem: *Opaque Failures* — when a call fails, the caller can't tell what went wrong or what to do next). The CLI surfaces failures from several places in distinct, family-specific shapes: **transport errors** and **decode errors** from Request Execution (010), **typed API errors** from API Error Extraction (015), and **usage errors** from Argument Dispatch (002). This capability collapses all of them into **one normalized diagnostic** that always carries the same three things: a **cause** (a human-meaningful explanation of what went wrong), a **category** (which kind of failure it is), and — where one exists — the **next step** the caller can take to resolve it.

It is the failure-legibility half of CONSTITUTION II (every error must explain what went wrong and the next step) and III (failures must be obvious, never silent). It sits at the centre of three siblings without doing their jobs: it is the producer-side **classifier** that Exit-Code Convention (004) — the sole category→code mapper — expects from "the API client," and it produces the diagnostic value that Output-Aware Failure Rendering (32) renders per `--output`. It does not print, choose an output format, emit a process exit code, retry, re-parse raw response bodies, or interpret a `403` as a plan-availability signal. Its single job is to make every failure legible in one consistent, actionable shape.

---

## Behavioral Accord

### Normalizing a failure into one diagnostic

- When handed any recognized CLI failure — a transport error, a decode error, a typed API error, or a usage error — the system produces exactly one normalized diagnostic carrying a cause, a category, and (where one exists) a next step, in the same shape regardless of which family the failure came from.
- When handed a failure it does not recognize as one of these three families, the system does not normalize it; it leaves the failure to Exit-Code Convention's (004) safety net, which exits with the internal-error code and writes the trace — the system never synthesizes a catch-all internal diagnostic of its own.
- When handed a successful outcome, the system produces no diagnostic and leaves the success untouched — only failures are normalized.

### Classifying the category

- When the failure is a transport error (the API could not be reached: connection refused, name-resolution failure, TLS failure, or timeout), the system classifies it as **network-unavailable**.
- When the failure is a decode error (a `2xx` response whose body could not be decoded into the caller's target), the system classifies it as a **general API error** — the call succeeded at the wire but the API returned an unreadable shape — so it stays distinguishable from a genuine internal failure.
- When the failure is a typed API error, the system classifies it from its authoritative HTTP status: `401`/`403` → **permission/authorization**; `429` → **rate-limited**; any other `4xx`/`5xx` → **general API error**.
- When the failure is a usage error (an unknown command, or an unknown/missing flag or positional argument), the system classifies it as a **usage error**.
- When more than one category could apply, the system classifies under the **most specific** one — a `429` is rate-limited, not a general API error — so the category is unambiguous for downstream consumers.
- The category vocabulary the system assigns is exactly the failure taxonomy Exit-Code Convention (004) maps to a process code; the system assigns the category and never emits the code itself.

### Composing the cause

- When the typed API error carries the API's own error detail (its `detail`, falling back to `title`), the system uses that text as the cause, so the operator reads the API's human-meaningful explanation rather than a status number alone.
- When the API supplied no readable detail or title, the system derives a cause from the HTTP status rather than inventing one; it never fabricates a specific cause the API did not give.
- When the failure is a transport error, the cause names the transport failure that occurred (the connection could not be established, name resolution failed, TLS failed, or the request timed out).
- When the failure is a decode error, the cause names that the API responded but its body could not be read as expected.
- When the failure is a usage error, the cause names what was wrong with the invocation — which token was unrecognized, or which flag or positional argument was missing or invalid — carried through from the dispatch outcome.

### Composing the next step

- When a category has a known, generally-applicable recovery, the system attaches it as the next step:
  - a `401` permission/authorization failure points the caller to verify the configured API token;
  - a `403` permission/authorization failure points the caller that its identity may lack the required role membership or permission for the resource;
  - a rate-limited failure points the caller to wait for the rate-limit window to reset (per the rate-limit headers carried on the error) and then retry;
  - a usage error points the caller to the command's help;
  - a network-unavailable failure points the caller to check connectivity and the configured API endpoint.
- When no reliable next step exists (for example, a general API error whose only signal is the API's own detail), the system omits the next step rather than guessing one, so the caller is never sent down a misleading path.

### Reporting the diagnostic — staying in its lane

- The system reports the normalized diagnostic to the calling command; it does not print it, render it in any `--output` format, or decide the process exit code — those belong to Output-Aware Failure Rendering (32) and Exit-Code Convention (004).
- The system does not re-parse a raw response body; it consumes the already-typed API error from API Error Extraction (015) and reads only what that typed error carries.
- The system does not retry, back off, or wait on any failure; a `429` reaches it only after Rate-Limit Handling (017) has exhausted its retry budget, and the system simply classifies the surfaced `429` as rate-limited.
- The system does not translate a `403` into plan- or feature-availability guidance ("not available on your plan"); that is the *Unsignalled Plan Limits* problem's concern. It classifies a `403` as permission/authorization and leaves plan-specific interpretation to that capability.

---

## User Scenarios

**In order to** decide what to do after a failure instead of decoding a status and body myself,
**as an** AI agent driving a command,
**I want** every failure handed back as one diagnostic carrying a cause, a category, and (where one exists) a next step.

**In order to** write one failure-handling path instead of one per failure source,
**as a** maintainer building endpoint commands,
**I want** transport, decode, API, and usage failures collapsed into a single consistent diagnostic shape.

**In order to** branch reliably on the kind of failure (back off, fix input, escalate, give up),
**as an** operator (human or agent),
**I want** every failure to carry a category drawn from one fixed vocabulary.

---

## Non-Behaviors

- The system must not print the diagnostic or render it in any output format. **Why**: rendering per `--output` is Output-Aware Failure Rendering's (32) job; duplicating it here would split one rendering contract across two capabilities and let them drift.
- The system must not emit or decide the process exit code. **Why**: Exit-Code Convention (004) is the single category→code mapper; this capability supplies the category, and a second emitter would risk two paths disagreeing on the code.
- The system must not synthesize a catch-all "internal error" diagnostic for failures it doesn't recognize. **Why**: an unrecognized failure is most likely an unanticipated internal crash, which 004's safety net already renders (internal-error code + trace); a competing internal-diagnostic path here could mask that trace and split one safety net across two owners.
- The system must not retry, back off, or sleep on any failure. **Why**: Rate-Limit Handling (017) owns the `429` retry within bounded caps; reintroducing waits here would double the delay and reopen the unbounded-wait hazard that capability exists to close.
- The system must not re-parse the raw response body to extract the cause. **Why**: API Error Extraction (015) already produced the typed error; re-parsing would duplicate that work and could disagree with the extraction the rest of the system trusts.
- The system must not translate a `403` into plan/Premium availability guidance. **Why**: that is the *Unsignalled Plan Limits* problem; conflating it here would attach plan advice to ordinary permission failures and mislead the caller.
- The system must not fabricate a cause or a next step the failure doesn't support. **Why**: CONSTITUTION VIII (No Fabricated Data) and II — a confidently wrong cause or next step is worse than an honest, status-derived one.

---

## Integration Boundaries

- **Request Execution (010)** *(upstream)*: source of transport errors and decode errors. The system reads the typed outcome and sends nothing.
- **API Error Extraction (015)** *(upstream)*: source of typed API errors (authoritative status, `detail`/`title`/`type`, raw body, headers). The system reads the typed error and never re-parses the body.
- **Argument Dispatch (002)** *(upstream)*: source of usage errors (unknown command, invalid/missing flag or positional). The system reads the classified usage outcome.
- **Exit-Code Convention (004)** *(downstream)*: consumes the assigned category to emit a process exit code, and owns the safety-net code for any failure the system does not normalize. The system supplies the category, not the code.
- **Output-Aware Failure Rendering (32)** *(downstream)*: consumes the normalized diagnostic and renders it per the selected `--output` format. The system produces the value, not the rendering.

---

## Driving Scenarios

### Happy path

**Scenario: Transport failure normalized to network-unavailable**
Given a command's request failed with a transport error (connection refused)
When the failure is normalized
Then the diagnostic's category is network-unavailable
And the cause names that the API could not be reached
And the next step points the caller to check connectivity and the configured endpoint

**Scenario: Permission failure carries the API's own detail**
Given a typed API error with HTTP status `403` and a `detail` of "You are not a member of this circle"
When the failure is normalized
Then the category is permission/authorization
And the cause is the API's `detail` text
And the next step points the caller that its identity may lack the required membership or permission

**Scenario: Usage error normalized from dispatch**
Given a usage error reporting the unknown command "rolez"
When the failure is normalized
Then the category is usage error
And the cause names the unrecognized token
And the next step points the caller to the command's help

### Error scenarios

**Scenario: API error with no readable detail derives its cause from the status**
Given a typed API error with HTTP status `500` and no `detail` or `title`
When the failure is normalized
Then the category is general API error
And the cause is derived from the HTTP status rather than invented
And no fabricated next step is attached when none reliably applies

**Scenario: Undecodable 2xx body normalized to general API error**
Given a decode error from Request Execution (a `2xx` response whose body could not be decoded into the caller's target)
When the failure is normalized
Then the category is general API error
And the cause names that the API responded but its body could not be read as expected

**Scenario: Rate-limited surfaced after retries are exhausted**
Given a typed API error with HTTP status `429` surfaced after Rate-Limit Handling (017) exhausted its retry budget
When the failure is normalized
Then the category is rate-limited
And the next step points the caller to wait for the rate-limit window to reset and retry
And the system attaches no additional wait or retry of its own

### Edge cases

**Scenario: A success is never normalized**
Given a successful (2xx) outcome reaches the normalizer
When it is processed
Then no diagnostic is produced and the success passes through untouched

**Scenario: Most-specific category wins on an overlapping status**
Given a typed API error whose status `429` is also, broadly, a non-2xx "API error" status
When the failure is normalized
Then the category is rate-limited, not general API error

**Scenario: An unrecognized failure falls through to the safety net**
Given a failure value the system does not recognize as a transport, decode, typed-API, or usage failure
When it reaches the normalizer
Then no diagnostic is produced and the failure is left to Exit-Code Convention's safety net (internal-error code + trace)

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: One consistent shape across every failure family**
Given one failure of each family — a transport error, a typed API error, and a usage error
When each is normalized
Then each diagnostic exposes the same three fields (cause, category, next step) and each category is drawn from the one fixed vocabulary

**Scenario: The 403 boundary holds — no plan guidance leaks in**
Given a typed API error with HTTP status `403`
When the failure is normalized
Then the diagnostic contains no plan- or Premium-availability language, confirming the *Unsignalled Plan Limits* boundary is intact

**Scenario: No implementation leakage in the artifact**
Given the produced specification
When it is reviewed
Then it names only the diagnostic's observable fields (cause, category, next step) and prescribes no language, type system, or internal data layout

---

## Assumptions

- **Category vocabulary alignment**: Assumed the category vocabulary is exactly Exit-Code Convention's failure taxonomy — usage error, general API error, permission/authorization, rate-limited, network-unavailable, and internal/unexpected. (Informed by 004 being the sole category→code mapper, which names "the API client" as the classifier that supplies the category.)
- **One failure in, one diagnostic out**: Assumed the normalizer handles a single terminal failure per command invocation, not an aggregation of multiple failures. (Informed by the CLI's one-invocation-one-outcome shape across the sibling specs.)

---

## Ambiguity Warnings

None remaining — the three ambiguities from the initial draft (decode-error category, 401-vs-403 next-step granularity, internal/unexpected coverage) were resolved in the 2026-06-10 clarification session. See **Clarifications**.

---

## Clarifications

### Session 2026-06-10

- **Internal/unexpected coverage**: Diagnostic Normalization covers only the three known failure families (transport/decode, typed-API, usage). A failure it does not recognize falls through to Exit-Code Convention's (004) safety net (internal-error code + trace); this capability does not synthesize a catch-all internal diagnostic of its own.
- **Decode-error category**: A `2xx` response whose body cannot be decoded is classified as a **general API error** — the call succeeded at the wire but the API returned an unreadable shape — keeping it distinguishable from a genuine internal failure.
- **Permission next-step granularity**: `401` and `403` share the permission/authorization category but carry distinct next steps — `401` → verify the configured token; `403` → the identity may lack the required membership/permission.
- **Rate-limited next-step**: A `429` surfaced after Rate-Limit Handling (017) exhausts its retries carries a next step — wait for the rate-limit window to reset (per the rate-limit headers on the error) and retry.
