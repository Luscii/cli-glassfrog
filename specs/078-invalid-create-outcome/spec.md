# Specification: Invalid-Create Outcome

**Feature**: 078-invalid-create-outcome
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

`glassfrog proposal create` is the anchor of the governance write path. Post-Create Validity Read (074) closed one gap in it: after a create the server accepts, the CLI now reads the created draft back and surfaces the server's own verdict — the `valid` flag, its `validation_alerts`, and the transitions available on it — alongside the returned `prp_` id. But 074 stops at *surfacing*. It deliberately keeps a created-but-invalid proposal a **success** exiting `0`, carrying an unfavourable verdict, because turning that verdict into a reported failure introduces a new outcome and touches the exit-code registry — which 074 named as this sibling capability's whole job.

Invalid-Create Outcome closes the loop. When a create succeeds but 074's read-back reports the server's verdict as **not valid** (`valid: false`), this capability reclassifies the invocation as a **failure**: a new outcome category with its own previously-unused process exit code, rendered through the CLI's normalized failure machinery so the created id and the server's alerts travel as the failure's explanation. This is the runtime answer to the CSG-2 incident recorded in Change-Set Grammar Facts (072) — a human confirmed a gated write, the CLI reported success, and the object was dead on arrival. After this capability, a dead-on-arrival draft exits non-zero, so an agent branching on `$?` and a CI job both see the failure without parsing text. The trigger is only an *explicit* unfavourable verdict: an unknown verdict, an unreachable read-back, or a valid draft that merely carries alerts all stay a success, exactly as 074 reports them. No governance rule is evaluated locally — the verdict is the server's, obtained by 074, so VISION Exclusion 2 holds.

---

## Behavioral Accord

### Outcome classification

- When a create succeeds and the read-back reports the server's verdict as not valid, the invocation terminates as a **failure** in a distinct outcome category, exiting with that category's own process exit code.
- When a create succeeds and the read-back reports the draft valid, the outcome is unchanged — the create remains a success exiting `0`, whether or not the server attached alerts to that valid draft.
- When a create succeeds but no server verdict is available — the read-back returned no validity information, or the read-back could not be completed at all — the outcome is unchanged: a success exiting `0` carrying the unknown-verdict result 074 already reports. The failure outcome is reserved for an explicit unfavourable verdict, never for the absence of one.
- When the server rejects the create itself, the outcome and its exit code are exactly today's — this capability changes nothing on the rejected-create path, and no read-back exists to classify.
- When more than one interpretation could apply, the classification keys on the `valid` flag alone: a `valid: false` draft is a failure even with no alerts and even with an empty transition set, and a `valid: true` draft is a success even when it carries alerts.

### Registry extension

- The invalid-create failure's exit code is a **new, previously-unused code**, added at the single canonical category→code registry Exit-Code Convention (004) owns.
- No existing code is renumbered or reassigned when the new code is added, so a consumer's existing branch on `$?` never silently changes meaning, and the new code collides with no other category.

### Failure rendering

- When the invocation terminates as an invalid-create failure, the reported result still carries the created `prp_` id, so the accepted-but-dead draft stays locatable.
- When the invocation terminates as an invalid-create failure, it carries every validation alert the server attached — each with the server's own message, the element it concerns, and its severity — as the failure's explanation.
- When the invalid-create failure is rendered, it takes the shape of a normalized CLI failure (a stated cause and, where one exists, a remedy) rather than the success rendering of a created proposal carrying a bad verdict — the outcome and its rendering agree.
- When a machine-readable output format is selected, the failure is emitted as the structured failure document: its outcome category, the created `prp_` id, the cause, the remedy where present, and the alerts are all machine-readable, and nothing an agent needs in order to recognize the invalid-create failure is available only as human prose.
- When a machine-readable output format is selected, the failure document carries the created `prp_` id and the server's alerts as its own fields and does not embed the full server proposal document — the failure keeps the uniform shape of every other CLI failure, and an agent that needs the whole created object reads it back with the single-proposal read.

### Consumption

- The classification consumes the verdict Post-Create Validity Read (074) already obtained; this capability performs no read of its own and issues no additional request to reach the verdict.

---

## User Scenarios

**In order to** learn at the moment of the write that a confirmed governance change produced a dead object, without reading command output to find out,
**as an** AI agent driving the governance write path,
**I want** an accepted-but-invalid create to exit with its own failure code, so I can branch on `$?` and stop the sequence.

**In order to** keep a build honest when a proposal it created can never move forward,
**as a** CI pipeline,
**I want** an invalid create to exit non-zero, so the job fails instead of passing on a dead draft.

**In order to** find and clean up the dead draft the write left behind,
**as a** practitioner whose agent created a proposal on my behalf,
**I want** the failure to still show me the created `prp_` id and the server's reasons.

---

## Non-Behaviors

- The system must not raise the invalid-create failure on anything but an explicit `valid: false` — not on a missing verdict, not on a read-back that could not be completed, not on an empty transition set, and not on the mere presence of alerts on an otherwise-valid draft. **Why**: those are the unknown and valid-with-caveats states 074 established as independent dimensions; failing on absence is the mirror image of reading absence as success — both mislead — and 074 verified live that a `valid: true` draft carrying alerts is genuinely valid and that valid drafts routinely carry empty transition sets.
- The system must not determine validity locally — not from the change-set shape, the draft status, the transition set, or the alerts. **Why**: local governance judgment is VISION Exclusion 2, and obtaining the verdict from the server is 074's job; this capability only maps the verdict 074 already holds to an outcome.
- The system must not withhold, delay, or replace the created `prp_` id because the outcome is a failure. **Why**: the write already happened; a failure that swallows the id leaves an orphan the operator can only locate through the web UI — the exact CSG-2 aftermath, made worse because the id was the one handle the caller could have kept.
- The system must not reuse an existing failure code for the invalid-create outcome. **Why**: the create's transport succeeded, so this is neither a general API error, a permission failure, a rate-limit, nor a stale write; collapsing it onto one of those would make an agent confuse "the server refused the write" with "the server accepted the write but its result is dead," which call for different responses.
- The system must not perform its own read-back or issue any extra request to reach the verdict. **Why**: 074 already reads the created draft once; a second read doubles the create's request cost against the per-organization hourly budget (CONSTITUTION X) for a verdict already in hand.
- The system must not change the outcome of any other proposal write — advancing to circulation, withdrawing, or recording a response. **Why**: create is where an accepted-but-dead object is silently produced; the other writes fail loudly at the server and have no accepted-but-invalid state to detect.
- The system must not retry, poll, or wait for the draft to become valid. **Why**: validity is the server's verdict on the change set as submitted and will not change without a new proposal, so waiting spends budget inside a gated write for a verdict that is already final.
- The system must not emit a success-shaped document under the failure code, or a failure document under a success code, in any output format. **Why**: the exit code and the rendered outcome are one signal (Action Transparency, CONSTITUTION II); disagreeing halves would be a silent failure wearing a success mask in one channel (Fail Safe, CONSTITUTION III).
- The system must not embed the full server proposal document inside the invalid-create failure document. **Why**: a failure that carries a whole passed-through proposal object breaks the uniform failure shape every other CLI failure has, forcing an agent to parse invalid-create differently from every other failure; the created `prp_` id and the alerts are what the agent needs to act, and the full object stays one single-proposal read away.

---

## Integration Boundaries

- **Post-Create Validity Read (074, upstream)**: supplies the server verdict — the `valid` flag and the `validation_alerts` — that this outcome classifies on. 074 obtains the verdict; this capability decides the outcome from it. When 074 reports no verdict or a failed read-back, this capability leaves the outcome a success.
- **Exit-Code Convention (004)**: owns the single canonical category→code registry. This capability adds one new category and its previously-unused code there, under 004's no-renumber extension rule.
- **Diagnostic Normalization (031)**: the failure is rendered through the normalized diagnostic shape (a cause and, where one exists, a remedy), so the invalid-create failure looks like every other CLI failure to its reader.
- **Output-Aware Failure Rendering (032)**: the failure is rendered format-aware — structured when a machine format is selected, human-readable otherwise — so an agent reads the outcome without scraping text.
- **Proposal Creation (055)**: the command whose outcome is being reclassified. Its request path and its success-create output are otherwise unchanged.
- **Glassfrog API v5 — proposal create and single-proposal read**: unchanged and not re-issued. This capability adds no API call; it consumes 074's read of the created draft.
- **Change-Set Grammar Facts (072, complement)**: records CSG-2 — the accepted-but-invalid shape — as a documented shape to avoid. This capability is the runtime outcome for exactly that shape.

---

## Driving Scenarios

### Happy path

**Scenario: An invalid draft terminates the create as a failure**
Given a change set the server accepts but reports not valid
When the operator creates the proposal
Then the invocation terminates as a failure carrying the new invalid-create exit code
And the created `prp_` id is carried in the reported result
And every validation alert the server attached is carried as the failure's explanation

**Scenario: A valid draft still succeeds**
Given a change set the server accepts and reports valid
When the operator creates the proposal
Then the invocation succeeds and exits `0`, exactly as 074 reports it

**Scenario: A valid draft carrying alerts still succeeds**
Given a change set the server accepts and reports valid while attaching alerts to it
When the operator creates the proposal
Then the invocation succeeds and exits `0`
And the alerts are surfaced as 074 reports them, not as a failure

### Error scenarios

**Scenario: A missing verdict leaves the create a success**
Given the create succeeds
And the read-back returns the draft carrying no validity information
When the result is reported
Then the invocation succeeds and exits `0` carrying the no-verdict result
And the invalid-create failure is not raised

**Scenario: A failed read-back leaves the create a success**
Given the create succeeds and returns a `prp_` id
And the read-back cannot be completed
When the result is reported
Then the invocation succeeds and exits `0` carrying the unknown-verdict result 074 reports
And the invalid-create failure is not raised

### Edge cases

**Scenario: The failure keys on the verdict, not the transitions**
Given the create succeeds
And the read-back reports the draft not valid while reporting no available transitions
When the result is reported
Then the failure is raised on the `valid: false` verdict
And the outcome would be identical had the invalid draft reported a non-empty transition set

**Scenario: A machine-readable failure is fully structured**
Given a machine-readable output format is selected
And the server reports the created draft not valid
When the invocation terminates as an invalid-create failure
Then the emitted failure document carries the outcome category, the created `prp_` id, the cause, and the alerts
And nothing an agent needs in order to recognize the invalid-create failure is available only as human prose

**Scenario: The new code never renumbers an existing one**
Given the exit-code registry with its existing assigned codes
When the invalid-create category is added to it
Then the category takes a previously-unused code
And every existing category keeps the code it had before

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The failure keys only on the explicit server verdict**
Given the whole specification
When it is read for any instruction to raise the invalid-create failure from the draft status, the transition set, the presence of alerts, or the change-set shape
Then no such instruction is present, and every failure trigger is attributed to the server's `valid: false`

**Scenario: The created id survives every failure path**
Given each path on which the invocation terminates as an invalid-create failure
When the result is reported
Then the created `prp_` id is present in it

**Scenario: The new code is one-to-one and never reassigned**
Given the exit-code registry after the invalid-create category is added
When each category is matched to its code and back
Then the invalid-create category maps to exactly one previously-unused code
And no existing category's code has changed

**Scenario: The failure is distinguishable from every success state in a machine format**
Given a machine-readable output format
When the server reports the draft not valid, reports it valid, reports it valid with alerts, reports no verdict, and the read-back fails, in turn
Then the invalid-create failure is distinguishable from each of the three success states — valid, no verdict, and unavailable — without inference
And a valid draft carrying alerts is recognizable as the valid state rather than a state of its own
And none of these requires reading human prose to identify

---

## Assumptions

- **A new previously-unused exit code** (technical, confirmed with the developer in the define session): the invalid-create outcome takes a new code beyond the currently-assigned band rather than reusing an existing failure code, following 004's extension rule (new category → single registry site → previously-unused code → never renumber).
- **The verdict source is 074's read-back** (technical, verified): the `valid` flag and the `validation_alerts` come from Post-Create Validity Read, which established the single-proposal read as the only carrier of the verdict; this capability adds no request of its own.
- **The trigger is `valid: false` only** (behavioral, confirmed with the developer): a missing verdict and a failed read-back both leave the outcome a success, and a `valid: true` draft carrying alerts is a success. (Confirmed in the define session; grounded in 074's live finding that validity, status, transitions, and alert-presence are independent dimensions.)
- **The dead draft's remedy** (settled during the architecture phase): the normalized diagnostic points the operator at revising and re-creating, and names the GlassFrog web UI for removing the dead draft. The question this carried at specification time — whether the CLI exposes a discard path — was resolved against the landed surface: it does not (`withdraw` moves a *circulating* proposal back to draft rather than deleting a draft), so the remedy names only what the operator can actually do. Recorded in plan ADR-5; the exact wording is pinned in the interface accord.

---

## Ambiguity Warnings

None remaining. The one warning raised at specification time — whether the machine-format failure document should embed the full server proposal document — was resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-08-18

- **Machine-format failure payload — envelope-only vs. embedded proposal document**: on the invalid-create failure path the machine output carries the created `prp_` id and the server's alerts as fields of the normalized failure document, and does **not** embed the full server proposal document the success path emits. The failure keeps the uniform shape of every other CLI failure so an agent parses invalid-create like any other failure; the id and alerts are what the agent needs to act, and the whole created object stays one single-proposal read (`glassfrog proposal get <prp_id>`) away. Captured in the Failure-rendering accord and a new Non-Behavior. (The alternative — embedding the proposal object for a no-second-call convenience — was rejected because it makes the failure shape non-uniform.)
