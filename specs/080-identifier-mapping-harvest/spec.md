# Specification: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Identifier Mapping Harvest is the second answer to the **Change Targets Unidentifiable from the CLI** problem, and the complement to `Legacy Identifier Request` rather than a substitute for it. A proposal's change can only name an existing governance element by a *legacy numeric identifier* — the number the web UI shows in a URL and the change payloads carry under a `databaseId`-family key. The opt-in read parameter that capability turns on reaches the role, actor, tree, and identity reads only. There are no standalone accountability, domain, or policy endpoints, and a live probe confirmed the number is present on a role while absent from every one of that read's embedded accountabilities and domains. So for those three element types the number stays unreadable through every read the CLI ships.

What does carry them is the organization's own proposal history. The proposal list read returns each proposal's `changes` inline, so one walked call yields every numeric identifier ever authored into a change set, most of them named, and with every name they ever carried intact. A walk of 1139 proposals on 2026-08-05 produced 2916 distinct identifiers across ten key spellings, 81% carrying a human-readable label, and covered 14 of the 15 identifiers a failed drafting session had needed. The mapping is therefore **bootstrapped, not learned** — it does not accumulate across sessions, it is derived in one pass from data the CLI can already read.

The three unreadable element types are what the harvest is *for*, but they are not what it emits. Every numeric identifier the change payloads carry is harvested and typed, roles and circles and people included, because they arrive in the same walked payload and filtering them out would discard data already in hand. Reaching the residue is what makes descent load-bearing: the contract requires the accountability and domain change types to appear as *children* of a role change, never as top-level changes, so the numbers this capability exists for live one or more levels down inside a wrapper.

Two properties shape everything below. First, this is a **live, stateless read**: one invocation walks, projects, and emits, and nothing is written down. Persisting a mapping is a different capability with a different problem behind it, and it is where the project's exclusion of a local data store has to be settled — not here. Second, unlike the read route it complements, this output is a **derived projection, not a faithful echo**: the mapping exists in no API response, so the CLI assembles it. That licence is bounded — the CLI reshapes and groups what the payloads carried, and never invents, decodes, judges, or resolves.

---

## Behavioral Accord

### Walking the history

- When the operator runs the harvest, the CLI walks the organization's proposal history to completion, reading each proposal's change set from the response that lists it.
- When the operator narrows the walk with any filter the proposal list read already offers, the walk honours it and the resulting mapping is reported as partial.
- When the operator opts out of the walk after the first page, the mapping is reported as partial — whether or not more proposals existed, because it was the operator who bounded the walk rather than the history that bounded it.
- When that opted-out first page was not the whole history, the CLI also notes that more proposals exist; when it was the whole history, no such note is written. This is the proposal list read's own note discipline, unchanged.
- When the walk stops on an error after at least one page, the CLI emits the mapping gathered so far, names the cause, reports the mapping as partial, and exits with the failure the proposal list read already produces for that cause.
- When a record in the walked history cannot be read as a proposal, the CLI skips exactly that record, harvests every readable one, counts what it skipped, and reports the count in every output format — one unreadable record never caps the walk, and its loss is never silent.
- When the walk covered the whole history with no filter, no opt-out, and no unreadable record, the mapping is reported as complete.

### Reaching every change

- When a change contains further changes nested inside it, the CLI descends into them and harvests each nested change in its own right, to whatever depth they nest.
- When a nested change is harvested, its element type comes from that nested change's own type, not from the change that wraps it.

### Typing an identifier

- When a change carries a numeric identifier under a key spelling that names its element type, the CLI records the identifier under that element type.
- When a change carries a numeric identifier under a bare key that names no element type, the CLI takes the element type from that change's own type — this is how an accountability's number, which no key spelling ever names, becomes typed.
- When neither the key spelling nor the change's type establishes an element type, the CLI records the identifier with its element type reported as unknown, carrying the key spelling and the change type it was found under. It is never dropped and never guessed at.
- When a value under an identifier key is not an exact whole number — a fractional value, or one too large to hold exactly — the CLI records no entry for it and never truncates it toward one. A value the change payloads could never carry is not an identifier, and an entry keyed on it could be neither deduplicated nor matched. The CLI counts what it excluded and reports the count, so the exclusion reaches the completeness claim instead of passing unnoticed.
- When the same number is seen under two different element types across the history, each is a distinct entry. The CLI does not merge them, rank them, or choose between them — the identifier spaces are genuinely separate.
- When the same identifier arrives as text in one change and as a number in another, both are the one identifier and produce one entry.

### What an entry carries

- Every entry carries its element type and its number, and those two together identify it.
- When a change referencing an identifier also carries text naming the element, that text is recorded on the entry as a label, tagged with the payload key it was read from.
- When the text sits under a key the CLI recognises as that change type's naming field, the label is tagged as such. When it sits under any other text-valued key beside the identifier, the label is still recorded and still tagged with its key — so a consumer can tell a name from some other text found nearby, and an unrecognised key never means an identifier silently loses its label.
- When an identifier was named differently over its history, every distinct label it ever carried is recorded. A later name does not replace an earlier one, and none is marked as the identifier's current name.
- When a label was carried in several proposals, it records the most recent proposal that carried it and how many proposals carried it in all.
- When no change ever named an identifier, the entry carries no label, and the CLI neither synthesizes one nor omits the entry.

### Emitting the mapping

- When the harvest completes, its completeness — whole history, or partial and why — is stated in every output format.
- When the whole history was walked, the human render shows a summary: how many identifiers were found of each element type, how many carry no label, and that the walk was complete. It does not list the entries.
- When the walk was narrowed by a filter or an opt-out, the human render lists the entries as well as the summary. The rule keys on whether the operator narrowed the walk, not on how many entries came back, so the same invocation always renders the same way.
- When any structured format is selected, the whole mapping is carried in full — every entry, every label, every tag and provenance — regardless of how the walk ended.
- When the same harvest is run twice over an unchanged history, the two outputs are identical: entry order and label order are deterministic and do not follow the order proposals happened to arrive in.
- When the history yields no identifiers at all, the CLI emits an empty mapping, states that it is empty, and exits successfully.

### Failure

- When the walk fails before any page is gathered — unauthorized, rate-limited, or a plan refusal — the diagnostic and exit code are the ones the proposal list read already produces, and no partial mapping is emitted.
- When the operator supplies a filter value the proposal list read rejects, the harvest refuses on the same terms, before any request is sent.

---

## User Scenarios

**In order to** name an accountability or domain as a change target without a number that no read will give me,
**as an** AI agent drafting a proposal on a practitioner's behalf,
**I want to** obtain the numeric identifiers the organization's own proposal history has already recorded.

**In order to** recognise an element whose name has changed since it was last proposed on,
**as an** AI agent resolving a change target,
**I want to** see every name an identifier ever carried, not only its current one.

**In order to** check for myself that a number is the element I think it is,
**as a** practitioner whose governance work the CLI serves,
**I want to** see the most recent proposal that used each name, so I can read the change that used it.

**In order to** avoid trusting a mapping that silently covers only part of the history,
**as an** AI agent consuming the harvest,
**I want to** be told whether the walk was complete, so a narrowed mapping is never read as an exhaustive one.

---

## Non-Behaviors

- The CLI must not write, cache, or accumulate the mapping across invocations. **Why**: a stored mapping is a local data store and a second source of truth, which the project excludes outright; it would also make a stale number look authoritative long after the governance element it named was removed. Persistence belongs to the separate problem of what an operating surface keeps between sessions, where that exclusion gets settled deliberately.
- The CLI must not resolve, match, or rank an element against the mapping. **Why**: matching a description to an identifier is the sibling capability's whole job, and it has to report ambiguity rather than choose — folding a matcher in here would bury that discipline inside a read.
- The CLI must not record or infer which role a nested element sits on, even though the wrapping change carries that role's number. **Why**: which role holds an accountability is a live governance fact, readable from the role itself; a proposal from two years ago says only that it was nested there *then*, so carrying it forward would present history as current structure — the same staleness the no-store boundary exists to prevent.
- The CLI must not use a harvested number to address a resource, assemble a change, or send any write. **Why**: this capability's entire contribution is making the numbers readable; consuming one is the drafting path's act, and it happens behind the write-safety confirmation this read never touches.
- The CLI must not synthesize, decode, derive, or infer a number that no change carried. **Why**: the stable identifiers are randomly generated and encode nothing about the number, so a fabricated value would silently address the wrong governance element in a write — the worst outcome this whole problem area exists to prevent.
- The CLI must not present a filtered, opted-out, or error-truncated mapping as a complete one. **Why**: a consumer that treats a partial mapping as exhaustive concludes an identifier does not exist when it merely was not walked to, which is a confidently wrong answer rather than a missing one.
- The CLI must not drop an identifier it cannot type, nor guess a type for it. **Why**: an untypeable number is still the number a change used, so it is kept as an unknown-typed entry; dropping it would make the mapping quietly incomplete, and guessing would put an invented element type in front of a write. This governs an identifier the CLI cannot *type* — not a value it cannot *represent*, which the bullet below governs instead.
- The CLI must not exclude anything from the mapping silently — neither a record it cannot read nor a value that is not an exact whole number. **Why**: exclusion is sometimes the right answer, because a fractional value could never be a change payload's identifier and an entry keyed on one would serve no consumer. What the no-dropping rule above guards against is *quietness*, not exclusion. So every exclusion is counted and carried in the completeness claim, and a mapping that lost something never reports itself complete — which is what keeps these two rules complementary rather than contradictory.
- The CLI must not discard text it does not recognise as a naming field. **Why**: the change payloads carry no per-type field schema, so an unrecognised key is as likely to be a new change type as it is to be noise — dropping its text would leave an identifier silently unlabelled and indistinguishable from one the history genuinely never named, which is the distinction the downstream matcher depends on.
- The CLI must not collapse two element types sharing a number into one entry. **Why**: the legacy identifier spaces are separate per element type, so a merged entry asserts an equivalence the record does not contain and would hand a resolver the wrong element.
- The CLI must not prefer, promote, or mark one label as an identifier's current name. **Why**: the history says which names were used and when they were proposed, not which is in force today; naming one current would be a judgement the record does not support, and a match may legitimately be made on an old name.
- The CLI must not verify that a harvested identifier still exists, still names what it named, or is valid to use. **Why**: the server is the judge of a change target, and a local liveness check would be governance logic the CLI does not own — the mapping reports what the history recorded, nothing more.
- The CLI must not request the opt-in legacy identifier on any read, or blend a read's numbers into the mapping. **Why**: this capability's source is the proposal history alone; mixing sources would make the mapping's provenance unreportable and couple it to a facility with an expiry date.
- The CLI must not change what the proposal list read returns, or how it renders. **Why**: the harvest is a new projection over the same walk, and a read that other capabilities already depend on must not shift underneath them.

---

## Integration Boundaries

- **Glassfrog API v5 — proposal history** *(upstream)*: the proposal list operation returns each proposal's change set inline, which is the harvest's only source. Data flows one way. When it is unavailable, rate-limited, or refused by plan, the harvest fails on exactly the terms that read already fails on.
- **Pagination** *(internal)*: the harvest is a walked read and inherits the walk's completeness discipline — the same boundary signalling, first-page opt-out, and partial-on-error behavior the proposal list read has.
- **Output selection and serialization** *(internal, downstream)*: the mapping flows into whichever format was selected, including operator-supplied templates. The structured formats carry the whole mapping; the human render is where it is summarized.
- **Identifier Resolution by Content Match** *(downstream consumer)*: turns the harvested numbers into an answer by matching description text against the mapping. It owns declining to choose when a label maps to several identifiers or an entry carries none, and it is the consumer the label tagging serves — a label read from a recognised naming field is stronger evidence than text found under an unrecognised key. The harvest owns reporting both faithfully rather than smoothing them together.
- **Legacy Identifier Request** *(sibling, complement)*: covers the role, actor, tree, and identity reads through an opt-in read parameter that carries a retirement clock. The harvest covers what no read reaches. Neither replaces the other, and the harvest deliberately does not consume it.
- **Identifier Prompt Before Assembly** *(sibling, the floor)*: asking the operator for a number before assembling remains the unconditional fallback beneath both identifier routes, for the identifiers this harvest finds unlabelled or ambiguous.

---

## Driving Scenarios

### Happy path

**Scenario: An accountability's number is recovered from inside a role change**
Given a proposal whose role change carries a nested accountability change
When the operator runs the harvest
Then the mapping carries an entry typed as an accountability for the nested change's numeric identifier
And the entry carries the text that nested change used to name the accountability
And the entry is not typed by the role change that wrapped it

**Scenario: A walked history yields the whole mapping in one pass**
Given an organization whose proposal history spans more than one page
When the operator runs the harvest
Then the walk covers every page of the history
And the mapping reports itself as covering the whole history

**Scenario: An identifier carries every name it ever had**
Given two proposals that named the same numeric identifier differently
When the operator runs the harvest
Then the entry for that identifier carries both names
And each name records the most recent proposal that carried it and how many proposals carried it
And neither name is marked as the identifier's current one

**Scenario: The same run twice produces the same output**
Given an unchanged proposal history
When the operator runs the harvest twice
Then the two outputs are identical
And the entries and the labels within each entry appear in the same order both times

### Error scenarios

**Scenario: A walk that fails partway still yields what it gathered**
Given a history walk that fails after at least one page has been read
When the operator runs the harvest
Then the mapping gathered so far is emitted
And the mapping reports itself as partial and names the cause
And the command exits with the failure that walk already produces

**Scenario: A walk that fails immediately yields no mapping**
Given a caller whose credential the API refuses
When the operator runs the harvest
Then no mapping is emitted
And the diagnostic and exit code are the ones the proposal list read already produces

**Scenario: An unsupported filter value is refused before any request**
Given a filter value the proposal list read does not accept
When the operator runs the harvest with that value
Then the CLI refuses and names the unsupported value
And no request reaches the API

### Edge cases

**Scenario: An identifier that cannot be typed is kept, not guessed at**
Given a change whose key spelling names no element type and whose change type establishes none either
When the operator runs the harvest
Then the mapping carries that identifier with its element type reported as unknown
And the entry names the key spelling and the change type it was found under
And no element type is inferred for it

**Scenario: Text under an unrecognised key is kept as a tagged label**
Given a change that names its element under a key the CLI does not recognise as that change type's naming field
When the operator runs the harvest
Then the entry carries that text as a label tagged with the key it was read from
And the label is distinguishable from one read from a recognised naming field
And the identifier is not left unlabelled

**Scenario: One number seen as two element types stays two entries**
Given a history in which the same number appears once as a circle and once as a role
When the operator runs the harvest
Then the mapping carries two entries, one per element type
And neither entry is merged into or preferred over the other

**Scenario: A number written as text and as a number is one identifier**
Given a history in which the same identifier appears as text in one change and as a number in another
When the operator runs the harvest
Then the mapping carries one entry for it
And that entry carries the labels from both changes

**Scenario: An identifier no change ever named is still in the mapping**
Given a change that references a numeric identifier without carrying any text naming it
When the operator runs the harvest
Then the mapping carries an entry for that identifier with no label
And the CLI does not synthesize a name for it

**Scenario: An unreadable record is skipped and counted, not the end of the walk**
Given a walked history in which one record cannot be read as a proposal
When the operator runs the harvest
Then the mapping carries the identifiers from every readable proposal
And the output reports one record as unreadable and the mapping as not complete
And the read exits successfully

**Scenario: A first-page opt-out reports partial even when that page was the whole history**
Given a proposal history that fits on a single page
When the operator opts out of the walk after the first page
Then the mapping is reported as partial
And no note about further proposals is written
And the read exits successfully

**Scenario: A first-page opt-out on a longer history says more proposals exist**
Given a proposal history spanning more than one page
When the operator opts out of the walk after the first page
Then the mapping is reported as partial
And the output notes that more proposals exist than were harvested
And the read exits successfully

**Scenario: A value that is not a whole number is excluded and counted, never truncated**
Given a change carrying a fractional value under an identifier key
When the operator runs the harvest
Then no entry carries a truncated form of that value, and no entry is created for it
And the output reports one identifier as unrepresentable and the mapping as not complete
And the read exits successfully

**Scenario: A history carrying no identifiers is an empty mapping, not a failure**
Given a proposal history whose changes carry no numeric identifiers at all
When the operator runs the harvest
Then the mapping is empty and the output says so
And the read exits successfully

**Scenario: A narrowed walk lists its entries where a whole-history walk summarizes**
Given an operator who narrows the walk with a filter the proposal list read offers
When the operator runs the harvest and reads the human render
Then the mapping reports itself as partial and states that the walk was narrowed
And the human render lists the entries
And the same harvest run without the filter renders a summary instead, with no entries listed

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Nothing is written to disk**
Given a harvest run to completion
When the filesystem is compared before and after
Then no mapping, index, or cache file has been created or modified
And a second run reads the history again rather than reading anything back

**Scenario: The mapping's completeness claim can be trusted**
Given every way a walk can end — completed clean, filtered, opted out after the first page, stopped on an error, completed past an unreadable record, and completed past an unrepresentable value
When each resulting output is inspected in every format
Then exactly the clean completed unfiltered walk reports itself as covering the whole history
And each of the other five reports itself as partial

**Scenario: No harvested number is ever sent back to the API**
Given the CLI's full outbound request surface
When every request path, query parameter, and request body is inspected
Then no request addresses a resource by a legacy numeric identifier
And the harvest issues no request other than the history walk

**Scenario: The proposal list read is unchanged**
Given the proposal list read's output before and after this capability exists
When the two are compared in every format
Then they are identical
And no option the harvest introduces appears on the proposal list read

**Scenario: The human summary's counts account for every entry**
Given a whole-history harvest whose human render shows a summary
When the summary's per-element-type counts and its unlabelled count are compared against the structured output of the same run
Then every entry in the structured output is accounted for by exactly one element-type count
And the unlabelled count matches the number of entries carrying no label

**Scenario: A reader can tell an unlabelled identifier from an untypeable one**
Given a mapping containing an identifier no change ever named and an identifier whose element type could not be established
When a consumer inspects the two entries
Then the unlabelled entry carries its element type and no label
And the untypeable entry carries its labels, if any, and an element type reported as unknown
And neither is reported as the other

---

## Assumptions

- **The subcommand's spelling is already decided** *(developer decision)*: the harvest is `glassfrog proposal identifiers` — a subcommand of the proposal group, because its source is the proposal history. Recorded here so the interface accord adopts it rather than re-litigating it; the behavioral accord above deliberately does not rest on the spelling.
- **Scope is every element type the change payloads carry, not only the unreadable residue** *(developer decision)*: the problem this serves is the accountability, domain, and policy residue no read exposes, and the committed solution line names those three. The harvest nonetheless emits the role, circle, and person numbers too, because they arrive in the same walked payload and filtering them out would discard data already in hand. **The committed capability line still names three element types and needs widening to match.**
- **The filters mirror the proposal list read's** *(developer decision)*: the harvest offers the same narrowing options that read offers, rather than a set of its own, and any narrowing marks the mapping partial. Chosen over offering no filters — a walk of the whole history is expensive, and an operator who knows the proposal they want should not pay for all of them.
- **No enclosing context on a nested entry** *(developer decision)*: the wrapping role change carries the role's own number, so an accountability entry *could* record which role it was nested under, for free. Deliberately not done: which role holds an accountability is a live governance fact to read from the role, and a proposal's nesting is only evidence of where it sat at that time. Chosen over recording it, and over recording only the most recent nesting.
- **Labels are a recognised naming set plus tagged fallback** *(adopted on the specifier's recommendation, not a developer choice)*: the CLI knows a naming field per change type and tags labels read from it as such, and records any other text found beside the identifier as a label tagged with its own key. The alternatives each destroy a distinction the downstream matcher needs — a recognised-set-only rule degrades silently when the API grows a change type, leaving an entry indistinguishable from one the history never named; an any-text rule makes a rationale a name and feeds false matches into a matcher that keys on description text. **Say so and this becomes either alternative** — the tagging is the whole cost of keeping both readings available.
- **"Most recent" means most recently created** *(technical)*: a label's recorded proposal is the latest by the proposal's own creation time. Proposals carry several timestamps; creation is the one every proposal has regardless of how far it travelled.
- **The human render's rule keys on narrowing, not on size** *(developer decision)*: a whole-history walk summarizes and a narrowed walk lists, so the same invocation always renders the same way and no size threshold has to be guessed at. A small organization's whole-history walk therefore summarizes even though it would fit on screen — chosen over a threshold, which would make the render unpredictable across organizations and over time.
- **Element typing keys on the change type, not only the key spelling** *(verified, not assumed)*: ten key spellings were observed in a live harvest, and none of them names an accountability — an accountability's number arrives under the bare key on a nested accountability change. So the change's own type is load-bearing for the one element type the problem most needs, not a fallback.
- **Accountability and domain changes are nested, not top-level** *(verified, not assumed)*: the contract states that the six accountability and domain change types must appear as children of a role change. Descent is therefore the only route to the residue this capability exists for, not a robustness measure.
- **The identifier spaces are separate per element type** *(verified, not assumed)*: circles were observed occupying a five-to-six-digit space while roles and accountabilities occupy an eight-digit one, and no sampled change carried both a stable and a numeric identifier for the same target. Keying entries by element type and number together is therefore a property of the record, not a defensive choice.
- **Both spellings of a number are the one identifier** *(verified, not assumed)*: the same payload was observed carrying the identifier as text at the change level and as a number in its nested children, so accepting both is required rather than tolerant.
- **The output is a derived projection, not an echo** *(technical)*: the sibling read route holds structured output to a faithful echo of the response. That principle cannot apply here — the mapping exists in no response — so the CLI groups, types, and deduplicates. The bound is that it reshapes only what the payloads carried.
- **[ASSUMED] Ordering is by element type then number, labels by first appearance in the walk**: determinism is required by the accord; this particular order is an informed default. If a different order serves a consumer better, only the order changes, not the content.

---

## Ambiguity Warnings

None remaining — the two questions raised during specification (which text in a change counts as a label, and what the human render does with a mapping of this size) were both resolved during clarification. See Clarifications.

---

## Clarifications

### Session 2026-08-22

- **Descent is the load-bearing path, and it was missing**: the accord described only top-level changes, while the contract requires the accountability and domain change types to appear as *children* of a role change. The residue this capability exists for was therefore unreachable as specified. The harvest now descends to any depth, and a nested change is typed by its own type rather than by the change that wraps it. (System Overview; new *Reaching every change* accord group; happy-path driving scenario; Assumptions.)
- **A nested entry records no enclosing context**: the wrapping role change carries the role's own number, so an accountability entry could cheaply say which role it was nested under. Deliberately not recorded — which role holds an accountability is a live governance fact to read from the role, and a proposal's nesting only says where it sat at that time. Chosen over recording it, and over recording only the most recent nesting. (Non-Behaviors; Assumptions.)
- **What text counts as a label**: the CLI recognises a naming field per change type and tags labels read from it, and records any other text beside the identifier as a label tagged with its own key. Chosen over a recognised-set-only rule, which degrades silently when the API grows a change type and leaves an entry indistinguishable from one the history never named; and over an any-text rule, which makes a rationale a name and feeds false matches into a matcher that keys on description text. Adopted on the specifier's recommendation rather than as a developer choice, and flagged as such in Assumptions. (Behavioral Accord — *What an entry carries*; Non-Behaviors; edge-case driving scenario; Integration Boundaries.)
- **Provenance is bounded**: a label records the most recent proposal that carried it and how many proposals carried it in all, rather than every proposal — a common name used in hundreds of proposals would otherwise make its entry enormous. (Behavioral Accord — *What an entry carries*; User Scenarios; happy-path driving scenario.)
- **The human render summarizes a whole-history walk and lists a narrowed one**: the observed harvest produced 2916 entries, so a full render serves nobody. The rule keys on whether the operator narrowed the walk, not on entry count, so the same invocation always renders the same way and no threshold has to be guessed at. Structured formats always carry the whole mapping regardless. (Behavioral Accord — *Emitting the mapping*; edge-case driving scenario; new validation scenario reconciling the summary's counts against the structured output; Assumptions.)

### Session 2026-08-23 (amendment during `/score:guard --pre`)

- **An unreadable record degrades the mapping instead of capping the walk**: risk review (H-4) found that decoding the walk through the typed proposal model let one malformed change element anywhere in the history fail its whole page, and the walk's stop-and-retain contract would then cap the harvest at the pages before it — permanently, with no operator remedy, in the one feature whose value is that a single walk bootstraps the whole mapping. The accord now isolates the fault at the record: an unreadable record is skipped, counted, and reported in every format, the mapping is not reported as complete, and the read still succeeds. The loss is visible where the completeness claim lives, which is what keeps a degraded mapping from being read as an exhaustive one. (Behavioral Accord — Walking the history; edge-case driving scenario; completeness validation scenario widened to five endings.)

- **An unrepresentable value is excluded and counted, not forced into an entry**: the guard round (checklist III.4) found that routing a non-integral value to "unknown handling" named no outcome the contract could hold — an unknown-typed entry still needs the whole number its key is built from, and the covering scenario asserted only that nothing was truncated, so it passed under either reading and tested neither. Resolved on the shape the unreadable-record remedy had already established one finding earlier: exclude it, count it, let the count withhold the completeness claim. Chosen over an entry with an absent number (unkeyable, undeduplicable, unmatchable — it would break the (element type, number) keying the wrong-element control rests on) and over a distinct element-type member (which would conflate *what kind of element* with *whether the value could be read*). A companion non-behavior states why excluding here does not contradict the rule against dropping an untypeable identifier: that rule forbids quietness, not exclusion. (Behavioral Accord — Typing an identifier; Non-Behaviors; edge-case driving scenario; completeness validation scenario widened to six endings.)
- **The empty mapping gained the scenario its accord bullet always implied**: the accord promised an empty mapping, a statement that it is empty, and a successful exit, and the interface pinned both the document (`total: 0`, `entries: []`) and the render idiom — but no scenario exercised either, the only accord bullet in the feature with no coverage (checklist IV.1). Added as an edge case, mirroring the sibling read's *"An empty visible set is a clean success"*. (Edge-case driving scenario.)

- **The first-page bullet was split so each consequence carries its own condition**: as one sentence it read as a flat conjunction — opting out makes the mapping partial *and* notes that more proposals exist — while the interface fires the note only when the API reports a next page. The trailing *"exactly as the proposal list read does"* resolved it correctly, but only for a reader who already knew the sibling's note discipline, and the surface reading contradicted the scenario governing it (analyze H3). Now two bullets: the partial marking is unconditional because the operator bounded the walk, and the note is conditional on the page not having been the whole history. Both cases gained a driving scenario, which also retired the last architecture-informed proposal by absorbing it into the accord. (Behavioral Accord — Walking the history; two edge-case driving scenarios.)
