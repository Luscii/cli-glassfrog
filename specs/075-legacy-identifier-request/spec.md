# Specification: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Legacy Identifier Request is the Must-tier answer to the **Change Targets Unidentifiable from the CLI** problem. A proposal's change can only name an existing role, circle, or person by a *legacy numeric identifier* — the number the web UI shows in a URL like `/orgnav/roles/11079492` and the change payloads carry as `databaseId`. Until now no CLI read exposed that number, so every write touching existing governance sent the operator out to the browser to read it off a URL mid-flow. That gap cost a full drafting run.

The API grew the answer. Six reads the CLI already ships accept an opt-in request for the number: the role list and single-role read, the role tree, the actor list and single-actor read, and the identity read. When asked, each resource in the response additionally carries its numeric identifier — verified live to be the same number the web UI shows and the change payloads consume. This capability turns that on: an explicit opt-in on exactly those six reads, surfacing the number alongside the stable identifier in whichever output format was selected.

The number is **a bridge with an expiry date**. The contract declares it a transition-only v3→v5 aid that retires when the v3 API retires, states plainly that integrations should not durably depend on it, and returns nothing at all for actors backed by agents. So this capability is shaped to *ask and report*, never to assume: off unless requested, honest about absence, and never a substitute for the stable identifier the CLI addresses resources by. When the bridge eventually stops answering, reads keep working and the numbers simply stop arriving — which is the behavior the fallback path (asking the operator for the number before assembling a change) is designed to catch.

One principle settles every question about where the number appears: **the structured output is a faithful echo of the response the API returned, and the human render is where the CLI curates and explains.** Structured output neither filters a number the API sent nor synthesizes one it did not; the human render may omit a number that serves no operator purpose and may say in words why one is absent.

---

## Behavioral Accord

### Requesting

- When the operator asks for the legacy numeric identifier on the role list, the single-role read, the role tree, the actor list, the single-actor read, or the identity read, the CLI requests it from the API as part of that read.
- When the operator does not ask for it, the CLI requests nothing extra and the read behaves and renders exactly as it does today.
- When the read walks more than one page, the request for the number applies to every page of the walk, so no page comes back without it while others carry it.
- When the operator asks for it on a read that does not support it, the CLI refuses before sending any request and names the option it does not accept.

### Surfacing — structured output

- When the number was requested, every resource in the structured output carries the number exactly as the response carried it for that resource: an integer where the API sent one, an explicit null where the API sent null, and no key at all for a resource the API did not carry it on.
- When the number was not requested, no key for it appears on any resource.

### Surfacing — human output

- When the number was requested, the human render shows it beside the stable identifier of each resource that is the read's own subject.
- When the identity read is asked for the number, the human render shows the numbers for the caller's own actor and for the organization — the two that a change can consume and that appear beside a stable identifier in that render.
- When the number was not requested, the human render is unchanged.

### Absence

- When a resource that is the read's own subject came back without a number, the human render shows that resource's number as explicitly absent, distinguishable from a resource whose number was never requested.
- When the resource is an actor backed by an agent, the human render names that agent backing as the reason no number exists. The structured output carries a plain null and nothing more, because the actor's kind is already in the payload and a consumer reads the reason from it.
- When related resources are embedded inside a read and that read's embeds do not carry the number, the human render states once, alongside that embedded group, that this read does not carry the number for embedded resources — it does not repeat an absence marker per embedded resource, and it does not let the omission read as "these resources have no number in the system of record".
- When a read's embedded resources *do* carry the number, the human render shows it on each of them as it would for any subject, and states nothing about embeds. Whether a given read's embeds carry it is a per-read observed fact, not a rule the CLI infers.
- When the number was requested and no resource in the response carried one, the read still succeeds: the CLI reports absence as above, emits no diagnostic, and exits successfully.

### Failure

- When the read itself fails — not found, unauthorized, rate-limited, or a plan refusal — the existing diagnostic and exit code apply unchanged, and asking for the number changes nothing about that outcome.

---

## User Scenarios

**In order to** target an existing role in a governance change without breaking flow to read a number off a browser URL,
**as a** practitioner whose governance work the CLI serves,
**I want to** ask a role read for the numeric identifier the change payload needs.

**In order to** assemble a change set in one pass instead of failing into a not-found and going out to the web UI,
**as an** AI agent drafting a proposal on a practitioner's behalf,
**I want to** obtain a change target's numeric identifier from a read I am already making.

**In order to** tell two same-named roles apart when choosing a change target,
**as an** AI agent resolving an ambiguous target,
**I want to** see each candidate's numeric identifier next to its stable identifier, so the choice is made on an identifier rather than on a name that does not distinguish them.

**In order to** avoid building a durable dependence on a facility that is going to be withdrawn,
**as a** maintainer of the CLI,
**I want to** have the number's opt-in, nullable, and time-limited nature stated where an operator meets it, so nobody mistakes it for a permanent part of the contract.

---

## Non-Behaviors

- The CLI must not request the number by default, or on any read where the operator did not ask. **Why**: the contract keeps it off by default precisely to keep the default read contract free of legacy coupling — and since the facility retires, a read that always carries the number teaches every consumer to depend on something that will disappear.
- The CLI must not offer the request on any read the contract excludes — cross-model search, the aggregated context reads, the role's filler/subrole/domain/policy/project reads, and the caller's own roles, actions, and projects reads. **Why**: an unrecognized request parameter is not rejected by the API, so an option offered there would look honoured and return nothing — a silent lie is worse than a refusal that names the option.
- The CLI must not make the request a persistent default settable through the environment or a configuration file. **Why**: a persisted default is indistinguishable from always-on, which is exactly the durable dependence the contract warns against; the opt-in must be a per-invocation act.
- The CLI must not synthesize, decode, guess, or derive a number that the response did not carry. **Why**: the stable identifiers are randomly generated, so they encode nothing about the number — a fabricated value would silently address the wrong governance element in a write, which is the worst outcome this whole problem area exists to prevent.
- The CLI must not filter a number out of the structured output, or add a key the response did not carry. **Why**: structured output is what a machine consumer parses; curating it makes the CLI an interpreter of the contract rather than a surface over it, and a consumer could no longer tell what the API actually returned.
- The CLI must not retain, cache, or accumulate numbers across invocations. **Why**: building a stored mapping is a separate capability with its own scope (the residue no read exposes); folding a cache in here would couple a live read to an index and make a stale number look authoritative.
- The CLI must not present the number as the resource's identifier, accept it as input to address a resource, or substitute it for the stable identifier anywhere. **Why**: the number retires and the stable identifier does not; a read or write keyed on the number would break on the day the bridge is withdrawn.
- The CLI must not validate, normalize, range-check, or otherwise judge the number it received. **Why**: the server owns this value; interpreting it would put local logic between the operator and the system of record for no gain, and would risk rejecting a number the API considers valid.
- The CLI must not treat an absent or null number as a failure, a warning, or a reason to exit non-zero. **Why**: absence is legitimate in three distinct ways — agent-backed actors have no legacy record, embedded resources never carry one, and the parameter will one day be silently ignored — so failing would break reads that are working correctly.
- The CLI must not attempt to detect, announce, or infer that the facility has retired. **Why**: it genuinely cannot distinguish "the bridge is gone" from "these particular resources have no number", so any signal would be a false alarm on a working read; noticing an upstream contract change belongs to the check that diffs the vendored contract, which is the only reliable drift signal.
- The CLI must not assemble, validate, or send any proposal change using the number it surfaced. **Why**: consuming the number belongs to the drafting path and its pre-assembly gate; this capability's whole contribution is making the number readable.

---

## Integration Boundaries

- **Glassfrog API v5** *(upstream)*: six operations accept an opt-in request for the legacy numeric identifier — the role list, the single role, the role tree, the actor list, the single actor, and the identity read. Data flows one way: the CLI asks, each resource in the response carries the number or a null. The retirement of this facility is expected to present as numbers no longer arriving rather than as a rejected request, since the API does not reject request parameters it no longer recognizes.
- **Output selection and serialization** *(internal, downstream)*: the number flows into whichever format was selected. The structured formats echo the response's own shape; the human formats and operator-supplied templates render what the human accord above describes.
- **Pagination** *(internal)*: walked reads carry the request on every page, so a multi-page result is uniform.
- **Vendored-contract drift check** *(sibling)*: owns noticing that the facility retired or changed. This capability deliberately emits no retirement signal of its own and relies on that check instead.
- **Governance write path** *(downstream consumer)*: the drafting path consumes the number as a change target. When no number is obtainable, the unconditional fallback — asking the operator for it before assembling — remains the floor beneath this capability rather than being replaced by it.

---

## Driving Scenarios

### Happy path

**Scenario: A single role read carries its numeric identifier**
Given a role that exists in the organization
When the operator reads that role and asks for the legacy numeric identifier
Then the output carries the role's numeric identifier alongside its stable identifier
And the numeric identifier is the same number the web UI shows for that role

**Scenario: Every role in a walked list carries its numeric identifier**
Given an organization whose role list spans more than one page
When the operator lists roles and asks for the legacy numeric identifier
Then every role in the aggregated result carries its numeric identifier
And no page is missing the numeric identifier while another page has it

**Scenario: One tree read yields a whole subtree's numeric identifiers**
Given a circle with sub-roles beneath it
When the operator reads the role tree and asks for the legacy numeric identifier
Then every row in the tree carries its numeric identifier

**Scenario: The identity read curates the human render and echoes the structured output**
Given a caller authenticated as a human actor
When the operator runs the identity read and asks for the legacy numeric identifier
Then the structured output carries every numeric identifier the response carried, including the membership's
And the human render shows the numbers for the caller's own actor and for the organization
And the human render does not show the membership's number

### Error scenarios

**Scenario: The request is refused on a read that does not support it**
Given a read the contract excludes from carrying the numeric identifier
When the operator asks that read for the legacy numeric identifier
Then the CLI refuses before sending any request
And the diagnostic names the option the read does not accept
And no request reaches the API

**Scenario: A failing read is unaffected by the request**
Given a role identifier that does not exist
When the operator reads that role and asks for the legacy numeric identifier
Then the CLI reports the not-found exactly as it does without the request
And the exit code is the one that read already produces

### Edge cases

**Scenario: An agent-backed actor has no legacy number**
Given an actor backed by an agent
When the operator reads that actor and asks for the legacy numeric identifier
Then the human render shows the number as explicitly absent and names the actor's agent backing as the reason
And the structured output carries an explicit null and no reason field
And the read succeeds

**Scenario: A read whose embedded resources carry no numeric identifier says so once per group**
Given a role with sub-roles and fillers, on a read whose embedded resources do not carry the number
When the operator reads that role, asks for the legacy numeric identifier, and asks for the sub-roles to be embedded
Then the role itself carries its numeric identifier
And the structured output carries no numeric-identifier key on any embedded sub-role or filler, exactly as the response did not
And the human render states once alongside each embedded group that this read does not carry the number for them
And no absence marker is repeated per embedded resource

**Scenario: A read whose embedded resources do carry the numeric identifier renders it**
Given the identity read, whose embedded roles carry the number
When the operator runs it, asks for the legacy numeric identifier, and asks for the roles to be embedded
Then each embedded role carries its own numeric identifier
And the human render states nothing about embedded resources lacking the number

**Scenario: Nothing comes back when the bridge no longer answers**
Given an API that no longer honours the request for the legacy numeric identifier
When the operator reads a role and asks for the legacy numeric identifier
Then the read reports the number as absent
And no diagnostic is emitted and nothing claims the facility has retired
And the read exits successfully

**Scenario: Not asking leaves the read byte-for-byte as it was**
Given a role that exists in the organization
When the operator reads that role without asking for the legacy numeric identifier
Then no field, line, or key for a numeric identifier appears in any output format
And the output is identical to what the read produced before this capability existed

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The retirement clock is stated where an operator meets the option**
Given the CLI's own help for a read that supports the request
When a reader consults it
Then the help states that the numeric identifier is a temporary transition aid that will be withdrawn
And it does not present the number as a stable or permanent identifier

**Scenario: A machine consumer can tell "not asked for" from "asked for and absent"**
Given two structured outputs of the same read — one that asked for the numeric identifier and one that did not
When a parser compares them
Then the output that did not ask carries no key for the numeric identifier at all
And the output that asked carries the key for every resource the response carried it on, holding either a number or an explicit null

**Scenario: The structured output's keys match what the API returned**
Given any read that asked for the numeric identifier, and the response the API returned for it
When each resource's numeric-identifier key is compared against that response
Then the structured output carries the key exactly where the response did and nowhere else
And no resource carries a synthesized number, a filtered-away number, or an added explanatory field

**Scenario: The option exists on exactly the supported reads and nowhere else**
Given the CLI's full command surface
When every read command's options are enumerated
Then exactly the role list, single role, role tree, actor list, single actor, and identity reads offer the request
And no other command offers it, including the caller's own roles, actions, and projects reads

**Scenario: The number is never used to address a resource**
Given the CLI's outbound requests across the whole read and write surface
When every request path and identifier argument is inspected
Then no request addresses a resource by its legacy numeric identifier
And every resource is addressed by its stable identifier

---

## Assumptions

- **The opt-in's spelling is already decided** *(developer decision)*: the option is spelled `--legacy-id` — the CLI's own short naming rather than a verbatim echo of the API's parameter name. Recorded here so the interface accord adopts it rather than re-litigating it; the behavioral accord above deliberately does not rest on the spelling.
- **Transitional caveat lives in help text only** *(developer decision)*: the number's opt-in, nullable, and time-limited nature is stated in the option's own help text. It is deliberately *not* accompanied by a diagnostic note when the option is used, nor by a "transitional" label in the rendered output. (Chosen explicitly over both alternatives during specification: the operator asked for the number, so the output stays clean and the caveat lives where the option is discovered.)
- **The requested number appears in every format that renders the read's subject** *(technical)*: structured, human, and operator-supplied template output alike. (An opt-in that a format silently dropped would defeat the request the operator made; the exact line shape in the terse human format is an architecture concern, not a behavioral one.)
- **Explicit absence uses the CLI's existing absence idiom** *(technical)*: the human renders already have a settled way of showing "this resource has none of that"; explicit absence reuses it rather than introducing a new marker.
- **Operator-supplied templates follow the structured shape** *(technical)*: a template addresses the decoded read result, so it sees the number wherever the structured output carries it. The curation described for the human render is the built-in renders' behavior, not a restriction the CLI imposes on a template the operator wrote.
- **The number and the change payload's identifier are one identifier space** *(verified, not assumed)*: the number a read returns and the number a change payload carries have been cross-checked as the same value under two spellings, so a number read here is directly usable as a change target.
- **All six reads' coverage is observed, not inferred from the contract** *(verified 2026-08-08, LEARNINGS W1–W4)*: every operation was probed live. The role tree returns the number on every node to full depth even though its response schema omits the field — the schema is defective, and a decision made from the contract alone would have dropped a working read. Which reads' *embedded* resources carry the number is likewise per-read observed: `me`'s embedded roles do, and the single-role read's embeds do not, for the same underlying schema.
- **[ASSUMED] Agent-backed actors return no number**: taken from the contract, not observed — the organization currently has no agent-backed actors (all 164 are human, list complete), so the absence-reason rendering is exercised by fixture only. If it turns out an agent-backed actor returns something other than null, only the reason text is affected, not the absence behavior. (LEARNINGS W6.)

---

## Ambiguity Warnings

None remaining — the three questions raised during specification (how embedded resources show an absent number, whether the identity read surfaces the membership number, and who owns noticing that the facility retired) were all resolved during clarification. See Clarifications.

---

## Clarifications

### Session 2026-08-08

- **One principle settles three of the four questions**: the structured output is a faithful echo of the API's response shape; the human render is where the CLI curates and explains. Added to the System Overview, split into two Surfacing groups in the behavioral accord, and protected by a new non-behavior forbidding both filtering a number out of structured output and adding a key the response did not carry.
- **Embedded resources**: the structured output omits the numeric-identifier key on embedded resources exactly as the API's response does; the human render states once, alongside each embedded group, that this read does not carry the number for embedded resources. Chosen over repeating an absence marker per embedded resource (uniform but noisy on a large embed) and over stating the rule only in help text (a reader of one response could not tell why the embed had none). The accord previously committed to the per-resource marking while the ambiguity warning called the choice open — that contradiction is resolved. (Behavioral Accord — Absence; edge-case driving scenario.)

### Session 2026-08-08 (amendment during `/score:guard --pre`)

- **The embed rule is per-read and observed, not global**: a live probe of all six reads found that `me`'s embedded roles **do** carry the number while the single-role read's embeds do not — the same `Role` schema behaving differently per endpoint (LEARNINGS W2/W3). The Absence accord had stated the exclusion as a blanket rule, which would have shipped a false sentence in `me`'s rendered output. It now has two bullets: one for reads whose embeds omit the number, one for reads whose embeds carry it, with the per-read determination named as observed fact. The structured path needed no change — the faithful-echo principle already mirrors whatever arrived rather than asserting a rule, which is the second time that principle has absorbed a contract surprise for free.
- **The role tree stays in scope on observed evidence**: its response schema omits `legacy_id`, but the runtime returns it on every node to full depth (LEARNINGS W1). Recorded as an assumption-turned-verification rather than left resting on the `$ref` enumeration that originally justified including it. Recorded here because the amendment came from a guard-stage probe, not from a specification gap: the spec's behavior was right, and its *grounding* was not.
- **Decode tolerance for the field**: the developer chose to accept both an integer and a string spelling for this one field rather than fail a read over an optional extra. Every observed value is an integer (LEARNINGS W5), so this is deliberate cheap insurance against a transitional field's looseness, not a fix for an observed defect.
- **The identity read's membership number**: the structured output carries all three numbers the response carries — actor, organization, and membership — while the human render shows only the actor's and the organization's, the two a change can consume and the two that appear beside a stable identifier in that render. The accord previously named two while the Assumptions said three; both now agree. (Behavioral Accord — Surfacing; happy-path driving scenario.)
- **Why an agent-backed actor has no number**: the human render names the agent backing as the reason; the structured output carries a plain null and no reason field, because the actor's kind is already in the payload and a machine consumer reads the reason from it rather than from a field the contract does not define. (Behavioral Accord — Absence; edge-case driving scenario.)
- **Retirement detection**: this capability owes no signal. The CLI cannot distinguish "the bridge retired" from "these particular resources have no number", so any signal would be a false alarm on a working read. Detection belongs entirely to the check that diffs the vendored contract — the recorded finding that neither the contract's version field nor its changelog is a usable drift signal makes that the only reliable instrument. Recorded as a non-behavior and as a sibling integration boundary. (Non-Behaviors; Integration Boundaries; edge-case driving scenario.)
