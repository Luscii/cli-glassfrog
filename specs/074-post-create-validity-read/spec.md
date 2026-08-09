# Specification: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

`glassfrog proposal create` is the anchor of the governance write path: it turns a captured tension into a draft proposal and prints the draft's `prp_` id so a later step can advance it to circulation. Today a create the server accepts is reported as a success, full stop. The write path's first live exercise showed that is not enough. A change set whose role update self-targeted the circle from inside its own governance was **accepted at create and returned with an id** while the server had already marked the draft not valid, attached a blocking alert, and left it with no available transitions — the shape recorded as **CSG-2** in `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md`. A human had confirmed a gated write; the CLI reported success; the object was dead on arrival and could only be deleted through the web UI. "Created" is not "valid," and nothing in the create result said so.

Post-Create Validity Read closes that gap by **asking the server rather than judging locally**: once a create succeeds, the CLI reads the created draft back through the single-proposal read and surfaces the server's own verdict on it — the `valid` flag, every entry in `validation_alerts`, and the transitions available on it — alongside the returned `prp_` id. A live probe (2026-08-08, read-only, recorded under Clarifications) established that these fields are carried by the **single-proposal read and by no other read**: `proposal list` omits them entirely, so the read-back is the only verified route to the verdict. The same probe found two live drafts in the organization already sitting at `valid: false` with populated alerts, a `draft_with_conflicts` proposal reporting `valid: true`, and valid drafts with an empty transition set — so validity, status, and available transitions are three independent dimensions and none of them can be derived from another. No governance rule is evaluated locally, so VISION Exclusion 2 holds; the read is a published v5 operation, so Principle 1 holds for the *operation* even though the two verdict fields it carries are not declared in the contract.

This specification stops at surfacing the verdict. Turning a created-but-invalid proposal into a reported failure with its own exit code is the sibling capability Invalid-Create Outcome, and recording the dead shape as a documented fact is Change-Set Grammar Facts (072), whose record explicitly defers runtime detection here.

---

## Behavioral Accord

### Read-back

- When a create succeeds, the CLI reads the created draft back from the server by the id the create returned, before reporting the result.
- When a create succeeds, the read-back happens unconditionally — it is never skipped on the basis of anything the CLI observes in the create response.
- When a create fails, no read-back is attempted and the failure is reported exactly as it is reported today.
- When a read-back is performed, the reported result carries the provenance of the verdict — that it came from a read of the named record after the create — in every output format the create supports, not only the human one, and in a form that format's reader can consume.

### Verdict surfacing

- When the read-back reports the draft valid, the result states that the server considers it valid, alongside the `prp_` id and status.
- When the read-back reports the draft not valid, the result states that verdict and carries every validation alert the server attached, each with the server's own message, the element it concerns, and its severity.
- When the read-back carries validation alerts, their presence is reported as their own fact — an alert never stands in for the validity verdict, and a draft the server reports valid while attaching alerts is reported as exactly that.
- When the read-back reports the transitions available on the draft, they are surfaced as their own part of the verdict — an empty set of transitions is never restated as a validity verdict, and a validity verdict is never inferred from the transitions.
- When the draft's status and its validity disagree, both are reported as they stand — the status is never adjusted to match the verdict, and the verdict is never inferred from the status.
- When the read-back carries no validity information at all, the result states that the server reported no verdict — never that the draft is valid, and never that it is not.
- When a machine-readable output format is selected, the verdict travels in the emitted document in the same structured form as the rest of the proposal, so an agent reads it without scraping human text.
- When a machine-readable output format is selected, everything the command reports about the verdict is machine-readable — including its provenance and, when the verdict could not be obtained, the reason and the remedy. Nothing an agent needs in order to tell the four verdict states apart is available only as human prose.

### Read-back failure

- When the read-back cannot be completed — the request does not reach the server, the server refuses it, or the retry policy the reads already apply is exhausted — the result still reports the created proposal's `prp_` id and states that the verdict could not be obtained and why.
- When the read-back fails, the created id is neither withheld nor replaced by the read-back's failure.

---

## User Scenarios

**In order to** find out at the moment of the write that a confirmed governance change produced an object nothing can move forward,
**as a** practitioner whose agent created a proposal on my behalf,
**I want to** see the server's own verdict on the draft in the result of the create I approved.

**In order to** decide the next step without issuing a second command to find out whether the write meant anything,
**as an** AI agent driving the governance write path,
**I want to** read the created draft's validity, its alerts, and its available transitions out of the create output I already parse.

**In order to** keep the handle on a proposal that really was created when the follow-up read fails,
**as an** AI agent,
**I want to** still receive the `prp_` id, together with an explicit statement that the verdict is unknown.

---

## Non-Behaviors

- The CLI must not determine validity locally — not from the shape of the change set, not from the draft's status, not from an empty transition set, not from the presence of alerts, not from any rule of its own. **Why**: local governance judgment is VISION Exclusion 2, and the probe showed every one of those proxies is wrong in practice — a `draft_with_conflicts` proposal reported `valid: true`, and valid drafts carry empty transition sets routinely. A locally-derived verdict would contradict the server's on live data today.
- The CLI must not change the create's outcome or exit code when the server reports the draft not valid — within this specification a created-but-invalid proposal is still reported as a successful create carrying an unfavourable verdict. **Why**: rendering it as a failure introduces a new outcome and touches the exit-code registry, which is the sibling Invalid-Create Outcome's whole job; folding it in here would hide a registry change inside a read-back and make neither reviewable on its own.
- The CLI must not present the `valid` flag or the `validation_alerts` entries as part of the published v5 contract. **Why**: neither field is declared in `spec/glassfrog-api-v5.yaml` — the accepted-but-invalid behaviour is still absent from the spec (LEARNINGS 2026-08-05, S5) — and VISION Principle 1 forbids publishing anything outside the contract as contract; labelling them authoritative would invite trust that breaks silently on the next refresh.
- The CLI must not treat a missing verdict as a favourable one. **Why**: an absent field silently read as "valid" is the original failure wearing a new mask — the caller would believe the write was checked when nothing checked it.
- The CLI must not withhold, delay, or overwrite the created `prp_` id because the read-back failed. **Why**: the write already happened; losing the id leaves an orphan draft the operator can only locate through the web UI, which is exactly the aftermath the CSG-2 incident produced.
- The CLI must not read back after a create the server rejected. **Why**: nothing was written, so the read has no subject; issuing it would produce a misleading not-found and spend rate-limit budget on a record that does not exist.
- The CLI must not offer a way to skip the read-back. **Why**: the failure being prevented is silent, so an opt-out would restore the exact shape of the problem on the path a hurried caller takes; the cost of the extra read is one read.
- The CLI must not poll, wait, or retry beyond the retry policy the existing reads already apply in order to obtain a verdict. **Why**: an unbounded wait sits inside a gated write with a human watching; the command reports what the server says now, and says so when the server says nothing.
- The CLI must not extend the read-back to the other proposal writes — advancing to circulation, withdrawing, or recording a response. **Why**: create is where a dead object is silently produced; widening this to every write would double the request cost of the whole write path and reach into problems those commands own separately.
- The CLI must not make the verdict a rendered part of the proposal list. **Why**: the server does not carry the verdict on the list read at all, so rendering it there would require a per-item detail read — a request per row for a surface that exists to scan many proposals cheaply.
- The CLI must not suppress or filter what the single-proposal read already returns to a caller who asked for a machine format. **Why**: `proposal get --output json` passes the server's document through verbatim, so the verdict fields already reach that caller today; a boundary that confines *rendering* to the create result must not become a reason to start withholding bytes the server sent.

---

## Integration Boundaries

- **Glassfrog API v5 — single-proposal read (`GET /proposals/{id}`, `getProposal`)**: the read-back target, a published operation already exercised by Proposal Reads, and the only verified carrier of the verdict. Reads the created draft; returns `valid` and `validation_alerts` alongside the declared proposal fields. When it is unavailable, the verdict is unobtainable and the created id is reported without it.
- **Glassfrog API v5 — proposal create (`POST /proposals`, `createProposal`)**: unchanged. The request body, the verbatim change set, and the create's own failure reporting are untouched; this capability only adds what happens after a successful create.
- **Glassfrog API v5 — proposal list (`GET /proposals`, `listProposals`)**: named here as a boundary that does *not* carry the verdict. Verified live: the list response omits `valid` and `validation_alerts` for every item, which is why the read-back reads one record rather than trusting a cheaper surface.
- **Rate limits (per-organization rolling hour)**: a create now costs two calls instead of one. A create that succeeds against a nearly-exhausted budget may find the read-back limited — which is a read-back failure, not a create failure.
- **Proposal Creation (055)**: the command surface being widened, and the owner of the create request path and its output-format dispatch.
- **Proposal Reads (056)**: supplies the single-proposal read and the shared proposal model and rendering that the verdict fields would grow into.
- **Write-Safety Guardrail (063)**: the gated confirmation that precedes the create is unchanged — the read-back is a read and needs no confirmation of its own.
- **Invalid-Create Outcome (downstream consumer)**: consumes this verdict to render a created-but-invalid proposal as a failure with an exit code. This capability produces the verdict; that one decides what it costs.
- **Change-Set Grammar Facts (072, complement)**: records the accepted-but-invalid shape as a documented shape to avoid, and explicitly leaves runtime detection of it to this capability, so the two do not duplicate each other.

---

## Driving Scenarios

### Happy path

**Scenario: A valid draft reports its verdict alongside its id**
Given a change set the server accepts and considers valid
When the operator creates the proposal
Then the result carries the created `prp_` id and its draft status
And it states that the server considers the draft valid
And it carries the transitions the server reports as available on it

**Scenario: A created-but-invalid draft surfaces the server's refusal**
Given a change set whose role update self-targets the circle from inside its own governance
When the operator creates the proposal
Then the result carries the created `prp_` id
And it states that the server considers the draft not valid
And it carries each validation alert the server attached, with the server's own message, the element it concerns, and its severity
And it states that no transitions are available on the draft

**Scenario: An agent parses the verdict out of machine-readable output**
Given a machine-readable output format is selected
When the operator creates the proposal
Then the emitted document carries the validity verdict, the validation alerts, and the available transitions alongside the id
And the reported result carries the provenance of the verdict — that a read of that record produced it — in a machine-readable form alongside the emitted document, which stays the server's own
And no part of the verdict is available only in human-formatted text

### Error scenarios

**Scenario: The read-back cannot reach the server**
Given the create succeeds and returns a `prp_` id
And the follow-up read cannot be completed
When the result is reported
Then it carries the created `prp_` id
And it states that the server's verdict could not be obtained, and why
And it does not describe the draft as either valid or not valid

**Scenario: The create itself is rejected**
Given the server rejects the change set
When the result is reported
Then the failure is reported exactly as a rejected create is reported today
And no read-back of any proposal is attempted

### Edge cases

**Scenario: The server reports no verdict at all**
Given the create succeeds
And the read-back returns the draft carrying no validity information
When the result is reported
Then it states that the server reported no verdict on the draft
And it describes the draft as neither valid nor not valid

**Scenario: A valid draft with no available transitions**
Given the create succeeds
And the read-back reports the draft valid while reporting no available transitions
When the result is reported
Then both the favourable validity verdict and the empty transition set are surfaced as distinct parts of the verdict
And neither is restated as the other

**Scenario: A status that disagrees with the verdict**
Given the create succeeds
And the read-back reports a status suggesting conflicts while reporting the draft valid
When the result is reported
Then the status and the validity verdict are both reported as the server gave them
And neither is adjusted to agree with the other

**Scenario: The read-back exhausts the hour's request budget**
Given the create consumes the last of the organization's hourly request budget
And the read-back is refused as rate-limited after the retry policy the reads already apply is exhausted
When the result is reported
Then it carries the created `prp_` id
And it states that the verdict could not be obtained because the request budget was exhausted

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Provenance of the reported result is legible**
Given a reader of the create result who was not present for the write
When they read a result carrying a verdict
Then they can tell which part of it the server returned about the created draft and that a second read produced it

**Scenario: No local validity derivation appears anywhere**
Given the whole specification
When it is read for any instruction to decide validity from the change set, the status, the transition set, the alerts, or any local rule
Then no such instruction is present, and every verdict statement is attributed to the server

**Scenario: Every verdict state is distinguishable in every output format**
Given each output format the create itself composes — the two human formats and the two machine formats
When the server states a favourable verdict, states an unfavourable verdict, states no verdict, and the read-back fails, in turn
Then all four are distinguishable from one another in that format without inference
And in a machine format none of the four requires reading human prose to identify
And under a caller-authored template the four states are *available* to be distinguished rather than rendered, because the command does not compose that output — a template that never references the verdict cannot be made to show it

**Scenario: No undeclared field is presented as contract**
Given the specification's treatment of the `valid` flag and the `validation_alerts` entries
When it is read for a claim that either is part of the published v5 contract
Then no such claim is present

---

## Assumptions

- **The verdict route is the single-proposal read** (technical, verified): the verdict is obtained by reading the created draft back, because the single-proposal read is the only surface observed to carry `valid` and `validation_alerts` — the list read omits them. (Established by the read-only probe recorded under Clarifications, not inferred.)
- **The create response's own verdict fields are unobserved** `[ASSUMED]`: the create response is a third serializer, and whether it already carries `valid` and `validation_alerts` at create time is unknown — settling it requires performing a real governance create and reading its raw body, which produces a draft in the live organization. The read-back is specified on the strength of the surface that *is* verified. (If a future observation shows the create response carries a computed verdict, the read-back becomes an optimization to remove rather than a behaviour to change — the surfacing accord above is unaffected either way. Flagged for the plan phase.)
- **Alert entries carry a message, a path, and a severity** (technical): each `validation_alerts` entry was observed as an object with the server's human-readable `message`, the `path` of the element it concerns, and a `severity`. (Two live invalid drafts, both `severity: error`; a non-error severity is possible, which is why alert presence is specified as distinct from the verdict.)
- **No new transport behaviour** (technical): the read-back travels the same connection, credential, and read-retry policy the existing reads use. (It is an ordinary published read; nothing about it needs its own transport rules.)
- **No new output machinery** (technical): the verdict is surfaced through the output-format selection the create already honours — verbatim in a machine format, templated for a human. (Existing format selection and templating already cover both paths.)

---

## Clarifications

### Session 2026-08-08

- **How the verdict is obtained**: settled by a read-only live probe before planning rather than by assumption. `GET /proposals/{id}` carries `valid` (boolean) and `validation_alerts` (array of `{severity, path, message}`); `GET /proposals` omits both for every item. Two live drafts in the organization sit at `valid: false` with populated alerts, so the unfavourable state is reachable without reproducing the CSG-2 shape. The read-back is therefore specified as the verdict's route, and the field spellings are pinned rather than assumed. The probe also disproved three candidate local proxies for validity: a `draft_with_conflicts` proposal reported `valid: true`, valid drafts routinely carry an empty `available_transitions`, and status moves independently of the flag. The one part the probe could not reach — whether the create response itself carries the fields — needs a real governance write to settle and is carried as a flagged assumption for the plan phase instead.
- **Scope of the surfacing**: the rendered verdict is confined to the create result, not grown into every proposal the CLI shows. The list read cannot carry it without a request per row, which settles the question for `proposal list` on the server's terms. For `proposal get`, the boundary is about *rendering* only — that command's machine output already passes the server's document through verbatim, verdict fields included, and this specification does not start withholding them.
- **What the read-back's visibility requires**: the verdict's provenance is carried in the reported result itself, in every output format, rather than disclosed only in command help. A command that makes two server calls where the caller named one must say so where the caller is already looking (VISION Principle 4).

### Session 2026-08-08 (post-guard)

- **How the machine-format claim is honoured**: a quality gate found that two of the four verdict states — the server stating no verdict, and the read-back never answering — produced an identical emitted document, leaving a prose stderr line as the only way to tell them apart. That contradicted this accord's promise that an agent reads the verdict without scraping human text. The first remedy considered was to **narrow the promise** to what the design then did. It was rejected: the accord stated the intent, the intent was right, and narrowing it would have preserved the gap the accord exists to close. The design was widened instead — the advisory became **format-aware**, following the landed convention that a diagnostic is rendered structurally when a machine format is selected. A CLI-owned diagnostic may carry a CLI-owned shape; what may never be reshaped is the server's own document, and that stays verbatim. All four states are now machine-distinguishable in every format. The accord, the validation scenario, the feature file, plan ADR-5, and `interface-cli.md` moved together.
- **Where the remedy for an unobtainable verdict lives**: in the advisory itself, not only in the accord. A read-back that fails now names the command that obtains the verdict (`glassfrog proposal get <prp_id>`) alongside the cause, because the verdict is still obtainable and the id it needs is in the output the caller just received. The one case with no remedy — no id could be determined — names none, rather than pointing at a command the caller cannot run.
