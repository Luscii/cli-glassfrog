# Specification: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Templated Human Rendering is the human-text half of the Output Formatting cluster. It introduces a *rendering seam* that maps a read command's result data into human-readable text through **named templates**, and ships two built-ins: **`full`** and **`compact`**. The seam is the load-bearing design intent — it is what later admits caller-supplied templates (029 User-Defined Template Output) without re-opening the rendering path. It connects directly to the project's third VISION principle: output shaped for both an AI agent to parse and a practitioner to read.

Today each shipped read (`me`, `my roles`, `my actions`, `my projects`) prints a hardcoded, per-command projection — a multi-line, labelled view with ids always present. This feature replaces those bespoke projections with the shared seam: today's projection becomes each command's `full` template, and `compact` is a new, denser, scan-friendly variant. This feature owns *only* the human-rendering mechanism and its two built-in templates. It does not own the `--output` flag, the choice of which template to apply, or the default when none is selected — that is 020 Output Format Selection. It does not emit JSON or YAML — that is 018 Structured Serialization. Because 020 has not landed, the standing CLI output is the `full` template (preserving the existing projection); `compact` is built and verified through the seam but is **not reachable from the command line** until 020 wires the flag.

---

## Behavioral Accord

### Rendering

- When a read command produces result data, the system renders it to human text by applying a named template, then prints the rendered text — the command no longer formats its own projection inline.
- When no format has been selected (the only mode available until 020), the system renders the `full` template, preserving the output the command produced before this feature.

### Full template

- When result data is rendered with `full`, the output carries every field the result holds — ids, names, kinds, access levels, and any embedded collections — with one labelled field per line.
- When `full` renders an embedded collection that is present (e.g. the roles from `me --include roles`), it enumerates each member with its identifying fields.
- The id values are always surfaced under `full`, because they are the machine-actionable handles an agent uses in follow-up calls.

### Compact template

- When result data is rendered with `compact`, the output is a condensed, scan-friendly view: the essential identifying fields, one line per record, denser than `full`.
- When `compact` renders an embedded collection (e.g. the roles from `me --include roles`), it shows the member *count* on the record's line (e.g. `roles: 3`) rather than enumerating each member, so the record stays a single line.
- The id values are still surfaced under `compact` — density reduces field count, never the actionable handle.

### Empty and absent data

- When a read returns an empty result set (zero records), both templates emit an explicit, per-command empty line (e.g. `no roles`, `no actions`, `no projects`) rather than printing nothing or a fabricated record — a successful-but-empty read must be legible, distinct from a failure.
- When a field's value is absent or blank, a template may render an explicit absence marker that signals the field is empty (e.g. `—`, `(none)`, `(no purpose set)`, `(no role)`) — each `full` template preserves the absence markers of its landed projection. The constraint is narrower: a template must never fabricate a *data value the API did not return* (an invented id, name, status, or real field value). An explicit emptiness marker reports absence; it does not invent data.
- When an embedded collection is empty, each `full` template handles it as its landed projection does — `me` omits the roles section, while `roles` renders the section header with `(none)`; `compact` renders the count (e.g. `roles: 0`) on the record's line.

---

## User Scenarios

**In order to** read a command's result as a human without parsing a machine format,
**as a** practitioner reviewing governance,
**I want** the CLI to render results as labelled, human-readable text.

**In order to** scan a long list of records quickly,
**as an** AI agent or practitioner triaging output,
**I want** a compact, one-line-per-record rendering.

**In order to** see everything a record carries when I need detail,
**as an** operator inspecting a single resource,
**I want** a full rendering that surfaces every field with its ids.

---

## Non-Behaviors

- The system must not read any flag or otherwise choose which template to apply, nor decide the default. **Why**: selection and the default belong to 020 Output Format Selection; choosing here would freeze a CLI surface 020 owns and force its redesign.
- The system must not expose `compact` through any operator-facing mechanism (flag, environment variable, or otherwise) in this feature. **Why**: `compact` becomes reachable only when 020 wires selection; a stop-gap surface here would be a second selection path 020 would have to reconcile or deprecate.
- The system must not emit JSON, YAML, or any machine-serialization format. **Why**: that is 018 Structured Serialization; rendering both here would couple two independently-shippable capabilities and double-commit the output contract.
- The system must not load, resolve, or accept caller-supplied template files. **Why**: that is 029; the seam admits them later, but accepting them now ships an unspecified file-resolution and sandboxing surface ahead of its spec.
- The system must not fabricate, default, or substitute a *data value* the API did not return — an invented id, name, status, or real field value. **Why**: Constitution forbids presenting values the API did not return; a fabricated governance value misleads both the agent and the practitioner. (An explicit emptiness marker such as `—` or `(none)` is not a fabricated value — it reports that a field is absent, it does not invent data.)
- The system must not render error output through a template. **Why**: the cause-plus-next-step error format is owned by the reads (and API Error Extraction); routing it through the human-render seam would blur the success/failure output contract.
- The system must not change which fields a read fetches from the API. **Why**: rendering is downstream of the read — it shapes presentation, not the request, so a missing field is the read's concern, not the renderer's.

---

## Integration Boundaries

- **Read commands (011–014, and future reads)**: supply their parsed result data to the rendering seam and print the text it returns. Each result type has its own `full` and `compact` rendering. If a read fails before producing result data, nothing is rendered.
- **Output Format Selection (020, downstream — not yet built)**: will select among the named templates and dispatch result data to the matching one. This feature exposes the named templates (`full`, `compact`) it dispatches to; until 020 lands, only `full` is reachable from the CLI.
- **User-Defined Template Output (029, future)**: will register caller-supplied templates through the same seam. This feature defines the seam shape but admits only the two built-ins.

---

## Driving Scenarios

### Happy path

**Scenario: full preserves the identity projection**
Given a successful `me` read returning an actor, organization, and membership
When the result is rendered with the `full` template
Then the output carries the actor's id, name, and kind, the organization's id and name, and the access level, one labelled field per line
And the output matches the projection the `me` command produced before this feature.

**Scenario: compact renders a list one line per record**
Given a successful `my roles` read returning several roles
When the result is rendered with the `compact` template
Then each role appears on a single line carrying its essential identifying fields
And each line still surfaces the role's id.

**Scenario: full enumerates an embedded collection**
Given a successful `me --include roles` read whose response carries roles
When the result is rendered with the `full` template
Then the identity fields are rendered
And each embedded role is enumerated with its id and name.

### Error scenarios

**Scenario: a failed read is not rendered through a template**
Given a `my actions` read that fails with a transport or API error
When the command reports the failure
Then the error keeps its existing cause-plus-next-step format
And no template is applied to it.

**Scenario: a missing field is omitted, never fabricated**
Given a successful `me` read whose result carries no embedded roles
When the result is rendered with the full template
Then the roles section is omitted rather than rendered as an empty heading
And no fabricated actor, organization, or role value stands in for data the API did not return.

### Edge cases

**Scenario: an empty result set is legible, not blank**
Given a successful `my projects` read returning zero projects
When the result is rendered with either template
Then the output emits an explicit per-command empty line (`no projects`)
And it prints neither nothing nor a fabricated row.

**Scenario: compact counts a nested collection that full expands**
Given a successful `me --include roles` read carrying three roles
When the result is rendered with `compact`
Then the nested role collection is shown as a count (`roles: 3`) on the actor's line rather than enumerated as `full` does
And the rendering still surfaces the actor's id.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: full is field-equivalent to the pre-feature projection**
Given each shipped read (`me`, `my roles`, `my actions`, `my projects`)
When rendered with `full`
Then the surfaced fields match what the command projected before this feature, with no field dropped or added.

**Scenario: no template introduces a value absent from the source**
Given any result data and either built-in template
When the result is rendered
Then every *data value* shown (id, name, status, field value) traces to a field the result carried — none is synthesized (explicit emptiness markers such as `—` / `(none)` are structural, not data values).

**Scenario: full and compact render the same record set**
Given the same list result
When rendered with `full` and again with `compact`
Then both account for exactly the same records — neither drops nor adds a record relative to the other.

---

## Assumptions

- **Standing template until 020**: Until Output Format Selection lands, the reads render `full` as their standing output, preserving today's projection; `compact` is built and unit-verified through the seam but not reachable from the CLI. (Confirmed during clarification — 020 owns selection and the default, so this feature must not introduce a flag or other selection surface.)
- **Per-result-type templates**: The seam is a shared rendering interface, but `full`/`compact` are realized per result type (each read's result has its own pair) rather than as one universal template over all results. (Each shipped read already has its own distinct projection.)
- **Empty-line wording follows the command noun**: The explicit empty line uses the read's own noun (`no roles`, `no actions`, `no projects`); the exact phrasing is a presentation detail the renderer owns. (Confirmed during clarification — same line under both templates.)

---

## Ambiguity Warnings

None — all warnings raised during specification were resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-07

- **Interim reachability of `compact`**: `compact` is built and verified through the seam but is not exposed through any operator-facing mechanism in this feature — the standing CLI output stays `full`, and `compact` becomes selectable only when 020 wires the `--output` flag. Added a non-behavior forbidding an interim selection surface and made the System Overview boundary definite.
- **Empty top-level result**: A zero-record read emits an explicit per-command empty line (e.g. `no roles`) under both templates, rather than printing nothing or a fabricated row, so a successful-but-empty read is legible and distinct from a failure. Added an "Empty and absent data" accord group and sharpened the empty-result edge-case scenario.
- **Compact rendering of nested collections**: Under `compact`, an embedded collection (e.g. `me --include roles`) is rendered as a member count on the record's line (`roles: 3`) rather than enumerated, preserving the one-line-per-record contract; `full` continues to enumerate. Added the rule to the Compact-template accord and updated the corresponding edge-case scenario.

### Session 2026-06-08 (post-shape)

- **Absent/blank field rendering vs. field-equivalence (resolved a spec-vs-landed-projection conflict surfaced during shape)**: The earlier accord said `full` must *omit* an absent field with "no placeholder, null, or dash", but `full` is also required to be field-equivalent to the landed projections, which deliberately render `(none)`, `—`, `(no purpose set)`, and `(no role)`. Resolved in favour of field-equivalence: a `full` template **preserves its landed explicit-absence markers**, which are structural emptiness indicators, not fabricated data. The anti-fabrication rule was narrowed to forbid only inventing a *data value the API did not return* (id, name, status, real field value). Reframed the "Empty and absent data" accord group and the matching Non-Behavior, rewrote the "a missing field is omitted, never fabricated" driving scenario around the genuinely-omitted case (`me`'s empty roles embed), and clarified the "no value absent from the source" validation scenario to mean data values, not structural markers. (Empty-embed handling stays per-read: `me` omits its roles section, `roles` renders `(none)` — each matching its landed projection.)
