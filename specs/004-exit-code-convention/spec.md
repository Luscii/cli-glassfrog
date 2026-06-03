# Specification: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Exit-Code Convention is the fourth and final capability of the CLI Skeleton (problem: *No Runnable CLI*). Every invocation of `glassfrog` ends by terminating the process with a numeric exit code — the one machine signal that travels back to whoever launched the CLI after all output is written. This capability owns that signal: the canonical mapping from an outcome's *category* (success, usage error, a specific operational failure, an unexpected internal failure) to a stable process exit code, and the guarantee that **every** termination path emits the right one.

The siblings deliberately reserved this surface. Argument Dispatch (002) classifies an invocation as a success or a usage error but explicitly refuses to emit a code; Help & Version (003) produces help/version text with a success outcome but decides no code. Exit-Code Convention closes that loop. Its operator is usually an AI agent acting for a practitioner: a stable, finely-distinguished exit code lets the agent branch on `$?` — back off on a rate limit, escalate on a permission failure, fix the command on a usage error — and lets CI fail a job on any non-zero result. This directly serves Action Transparency (the operator can always tell what happened) and Fail Safe (a failure is never reported as success).

This capability is a **pure category→code mapper**. It does not decide which category an outcome belongs to — the producer of the outcome classifies it (Argument Dispatch labels success/usage error; the future API-client classifies an API or network result into a failure category). Exit-Code Convention maps that category to its code and emits it.

---

## Behavioral Accord

### Success

- When a command completes its intended work, the process exits with code `0`.
- When dispatch routes a bare group, the root, or a help/version request to a help/listing outcome (002/003), that outcome is a success and the process exits with code `0`.

### Usage error

- When dispatch or argument parsing classifies an invocation as a usage error (an unknown command, or an unknown/missing flag or positional argument), the process exits with code `2`.

### Operational failure (API-facing)

- When an outcome is classified as a permission/authorization failure (the API rejected the caller's authentication or membership, including a premium-gated rejection), the process exits with code `4`.
- When an outcome is classified as rate-limited, the process exits with code `5`.
- When an outcome is classified as network-unavailable (the API could not be reached at all — connection failure, name-resolution failure, or timeout), the process exits with code `6`.
- When an outcome is classified as a general API error not covered by a more specific category, the process exits with code `3`.
- When more than one category could apply, the producer classifies the outcome under the **most specific** one; that category determines the code — a rate-limited outcome exits `5`, not the general API-error `3`.

### Unexpected failure (safety net)

- When the CLI terminates for any reason that matches no known category — including an unanticipated internal failure — the process exits with the internal-error code `1`, and never `0`.
- When the failure is an unanticipated internal crash, the safety net also writes the crash (the failure value and, where available, a trace) to standard error before exiting, so the failure stays diagnosable. **Why**: no other component renders an unrecovered crash, so suppressing it would hide what went wrong — the opposite of Action Transparency.

### Code map

| Category | Code |
|---|---|
| Success | `0` |
| Internal / unexpected error (safety net) | `1` |
| Usage error | `2` |
| General API error | `3` |
| Permission / authorization error | `4` |
| Rate-limited | `5` |
| Network-unavailable | `6` |

### Stability and extension

- Each category maps to exactly one code, and each assigned code maps to exactly one category.
- A single canonical category→code registry, owned by this capability, is the source of truth. Every command references it; a new outcome category is added only there, where it takes a new, previously-unused code. Existing assignments are never reassigned or renumbered, so a consumer's existing branch on `$?` never silently changes meaning, and two categories can never collide on one code.
- Codes reserved by the shell (126, 127, and 128 + N for signals) are never assigned to a CLI category.
- A single invocation terminates with exactly one exit code.

---

## User Scenarios

**In order to** react correctly to a failure without parsing text,
**as an** AI agent acting for a practitioner,
**I want** each failure class to carry a distinct exit code,
so that I can back off on a rate limit, escalate on a permission failure, and fix my own mistake on a usage error.

**In order to** fail a build whenever the CLI did not fully succeed,
**as a** CI pipeline,
**I want** every non-success outcome to exit non-zero,
so that no hidden failure passes the job.

**In order to** add a new command without inventing my own error signalling,
**as a** Maintainer,
**I want** my command's outcomes to map onto the existing code registry,
so that a new command behaves consistently with every other command on day one.

---

## Non-Behaviors

- The system must not render or format an error or outcome *message* — the explanation of what went wrong and the next step. **Why**: Help & Version (003) owns presentation and each command owns its own error text; Exit-Code Convention contributes the numeric process code. Rendering outcome messages here would duplicate and fracture the single owner of output. **Carve-out**: the unexpected-failure safety net may write the crash diagnostic (the failure value and any trace) to standard error — the one exception, because no other component renders an unrecovered crash and suppressing it would violate Action Transparency. This is a diagnostic, never an outcome-category message.
- The system must not decide which category an outcome belongs to, nor which registered command an invocation names. **Why**: the producer classifies the outcome (002 for success/usage error, the API-client for API/network failures); this capability only maps the resulting category to a code. Re-deciding the category here would split classification across two owners and couple the skeleton to the API's error taxonomy.
- The system must not retry, back off, or take any recovery action for a rate-limit or network failure. **Why**: it signals the category via the code so the caller (or the API-client layer, per CONSTITUTION X) decides whether to retry; silently recovering here would hide the failure the code is meant to surface.
- The system must not catch and suppress a failure to force a `0` exit. **Why**: a failure reported as success is exactly the silent failure CONSTITUTION III (Fail Safe) forbids; the safety-net rule routes every unclassified failure to a non-zero code instead.
- The system must not reuse a shell-reserved code (126, 127, 128 + N). **Why**: those collide with shell and signal semantics, corrupting an agent's interpretation of `$?`.

---

## Integration Boundaries

- **Argument Dispatch (002, upstream)**: hands this capability a success-or-usage-error classification for an invocation; it maps that to `0` or `2`.
- **Help & Version (003, sibling)**: produces help/listing/version text as a success outcome; this capability emits `0` alongside it.
- **API client / domain commands (downstream, future)**: classify an API or network outcome into a failure category (mapping concrete API statuses such as 401/403/429 into the abstract categories), then reference the registry to obtain the code. The skeleton defines the convention and the registry now; the API-facing categories are exercised once domain commands exist.
- **Process / shell (external)**: the exit code is the only *outcome* signal this capability emits — CI runners and agent operators read it via `$?`. The sole other thing it may write is the unexpected-failure crash diagnostic to standard error (see the safety-net accord); it emits no other output.

---

## Driving Scenarios

### Happy path

**Scenario: A successful command exits zero**
Given a registered command whose action completes its intended work
When the caller invokes it and it succeeds
Then the process exits with code `0`.

**Scenario: A help/listing outcome exits zero**
Given a `roles` group with subcommands is registered
When the caller invokes `glassfrog roles` with no further token
Then dispatch routes to a help/listing outcome classified as success
And the process exits with code `0`.

**Scenario: Different failure classes carry different codes**
Given one invocation is classified as a usage error and another as rate-limited
When each terminates
Then the first exits with code `2` and the second with code `5`
And an agent can tell the two failures apart from `$?` alone.

### Error scenarios

**Scenario: An unknown command exits the usage code**
Given no command named `rolez` is registered
When the caller invokes `glassfrog rolez`
Then dispatch classifies the outcome as a usage error
And the process exits with code `2`.

**Scenario: A rate-limited request exits the rate-limit code**
Given a command whose outcome the producer classifies as rate-limited
When the process terminates
Then it exits with code `5`
And not the general API-error code `3`.

### Edge cases

**Scenario: The most specific category wins**
Given an outcome the producer could classify as either a permission failure or a general API error
When it is classified under the most specific category, permission failure
Then the process exits with code `4`
And not the general API-error code `3`.

**Scenario: An unexpected internal failure never exits zero**
Given a command that terminates for a reason matching no known category
When the process ends
Then it exits with the internal-error code `1`
And never with `0`.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Codes and categories are one-to-one**
Given the registry of categories and codes
When each category is matched to its code and back
Then no two categories share a code and no category has two codes.

**Scenario: No shell-reserved code is assigned**
Given the assigned set of codes
When each is checked against the shell-reserved values (126, 127, 128 + N)
Then none of the assigned codes falls in that reserved range.

**Scenario: Adding a category never renumbers existing codes**
Given a published set of assigned codes
When a new outcome category is introduced in the registry
Then it takes a previously-unused code
And every existing category keeps the same code it had before.

**Scenario: Specification names no implementation technology**
Given the specification text
When it is scanned for technology names
Then no programming language, framework, library, or concrete data-structure choice appears (the numeric codes are the external contract, not an implementation choice).

---

## Assumptions

- **Binary name**: assumed the CLI is invoked as `glassfrog` and that this capability sets the exit code of that process. (Consistent with 001/002/003.)
- **API-facing categories are forward-looking**: assumed no domain command exists yet in the skeleton, so the API/permission/rate-limit/network categories are defined as convention and registry now, and first exercised when domain commands arrive. (Informed by the feature model: 004 depends only on 002; domain commands are later solutions.)

---

## Ambiguity Warnings

None remaining. The three warnings raised at specification time — HTTP-status→category ownership, code-assignment governance for the extensible set, and the exact numeric values — were resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-03

- **HTTP-status → category ownership**: this capability is a pure category→code mapper. The producer of an outcome classifies it — the future API-client maps concrete API statuses (e.g. 401/403/429) into abstract failure categories; 004 only maps category→code. This mirrors 002's producer-classifies pattern, keeps the skeleton free of API knowledge it does not yet have, and decouples 004 from the API taxonomy and CONSTITUTION X retry behaviour. Captured in the System Overview, the Operational-failure accord (now classification-driven), the Integration Boundaries, and a new Non-Behavior.
- **Code-assignment governance**: a single canonical category→code registry owned by 004 is the source of truth. Every command references it, and a new category is added only there, where the no-renumbering rule is enforced — making collisions impossible by construction. Captured in the Stability and extension accord.
- **Exit-code values**: fixed as `0` success / `1` internal-error / `2` usage / `3` general API error / `4` permission / `5` rate-limit / `6` network-unavailable. `0`/`1`/`2` follow the Unix/CLI-framework convention; `3`–`6` form a contiguous operational band. Captured in the Code map and the Operational-failure accord.
- **Crash-diagnostic carve-out** (resolves the analyze H3 drift): the "render no text / exit code is the only signal" boundary forbids rendering an error or outcome *message* (those belong to cobra / 003 / the command), but the unexpected-failure safety net MAY write the crash (the failure value and any trace) to standard error, because no other component renders an unrecovered crash and suppressing it would violate Action Transparency (II). Captured as a positive behavior in the safety-net accord, a carve-out on the rendering Non-Behavior, and relaxed wording on the Process/shell boundary — aligning the spec with plan ADR-4 and interface-cli.md.
