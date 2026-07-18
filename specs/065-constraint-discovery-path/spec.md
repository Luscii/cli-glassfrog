# Specification: Constraint Discovery Path

**Feature**: 065-constraint-discovery-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Constraint Discovery Path is a read-only **operator path** on the Agent Operating Surface — a guided composition riding on top of the Glassfrog CLI, built on the root that *Operator Orientation* (062) established. It is the mirror of the *Governance Navigation Path* (064): where 064 starts from a free-form *concern* and returns a picture of the governance around it, this path starts from something the operator wants to **do** — a concrete action — and surfaces the domains and policies that govern *that action*, so the operator learns whether it is within their own role's authority to take, falls under another role's domain (and so needs that role's permission), is shaped by a policy they must observe, or is a change to governance itself (and so needs a proposal).

The path is knowledge, not capability. It composes already-shipped reads — Cross-Model Search (041) to discover which models a wanted action touches, and the role-interior Role Domains and Role Policies reads for the governance that constrains it — and adds no command, flag, or governance logic of its own. It is strictly read-only.

Critically, the path **surfaces** the governing governance drawn from the record; it does **not** compute a permission verdict from local Holacracy rules. The API and the governance record are the source of truth (VISION Exclusion 2); reimplementing "what's allowed" locally would drift from that record and second-guess it. So when the record clearly shows a domain owns the action or a policy binds it, the path names that situation; and when the record does not clearly answer, the path says so and surfaces what it found rather than guessing a ruling. The exact delivery form — a plugin skill, a read-only agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a practitioner voices a free-form description of something they want to do (an action, in the loose sense), the path takes that action as its starting point — no pre-existing structure or id is required.
- When the action already names a specific role or domain the practitioner has in mind, the path may begin from that role directly rather than searching first.
- When the wanted action is too vague to locate the governance that would constrain it, the path asks the operator a clarifying question to sharpen the action before traversing, rather than guessing what they meant.

### Discovery

- When the governing governance is not yet known, the path uses cross-model search over the wanted action to discover which roles, domains, and policies it touches, then follows each result's id into the matching role-domains and role-policies reads.
- When a search or list read that supports paging spans more than one page, the path pages through the full result set before choosing what is most relevant — so narrowing is a relevance choice made over the complete set, never a silent drop of unfetched pages.
- When a list read cannot be paged to completion — the caller's own-roles read does not follow pagination and signals when its list is incomplete — the path treats that result as possibly-incomplete: it surfaces the incompleteness as uncertainty and must not conclude the action falls outside the caller's authority from a roles list that may be missing entries. An unconfirmed own-role is reported as uncertain, never as a definite "another role's domain".

### Characterization

- When the traversal is complete, the path surfaces the authority situation drawn from the record: which domain (if any) governs the action and whose role holds it, which policies (if any) shape or limit it, and whether the action is a change to governance structure that goes through a proposal by default.
- When a domain that governs the action belongs to a role other than the operator's, the path surfaces that the action falls under that role's authority — so the operator sees they would need that role's permission or a proposal, without the path ruling on the outcome.
- When no domain in view governs the action and no policy in view limits it, the path surfaces that the record shows nothing constraining it — it reports the absence of a constraint, it does not manufacture a "you are permitted" verdict.
- When the picture is presented, each element carries the id needed to read it again or act on it, so the surfacing bridges back into the CLI's own commands rather than being a dead end.

### Surfacing, not ruling

- The path surfaces the governing domains and policies as the governance in view; it does not compute whether the wanted action is permitted, forbidden, or requires a proposal from any local rule of its own.
- When the record does not clearly answer whether the action is constrained (for example, no domain plainly owns it, or the match is ambiguous), the path says so and surfaces what it found — it never fabricates an authority ruling to fill the gap.
- The path must not reimplement or duplicate Glassfrog's permission or validation rules locally — the API, per the spec, is the source of truth for what is valid.

### Read-only and fidelity

- The path only reads; it never writes to the governance record, and it composes only reads the CLI already exposes — it invents no command, flag, or capability.
- When the CLI's read surface changes, the path is expected to stay consistent with it; guidance that names a read the CLI no longer offers is a defect, not a difference of opinion.

---

## User Scenarios

**In order to** know whether an action I want to take is mine to take, needs someone else's permission, or needs a proposal — without hand-assembling a set of domain and policy reads,
**as an** AI agent,
**I want to** follow a guided path from a wanted action to the governing domains and policies that constrain it.

**In order to** decide how to proceed before I act,
**as a** practitioner (served through the agent),
**I want to** see which domain governs my action, which policies shape it, and whether it needs a proposal — drawn from the record, not from the tool's own opinion.

**In order to** move from understanding a constraint to acting on it without losing my place,
**as an** AI agent,
**I want to** have each surfaced domain, policy, and role carry the id that reads it again or feeds the next path.

---

## Non-Behaviors

- The path must not write to the governance record or drive any write command. **Why**: it is a read traversal; requesting permission, drafting a proposal, or capturing a tension are separate write paths (the proposal paths, 066) with their own guardrail, and folding a write in here would bypass the Write-Safety Guardrail.
- The path must not add any command, flag, or API capability of its own. **Why**: it is a guided composition of existing reads (knowledge + guardrails, never capability); inventing a surface would break "Bounded by the API surface" and VISION Exclusion 2.
- The path must not reimplement or duplicate Glassfrog's permission or validation rules locally, nor compute an allow/deny verdict from its own logic. **Why**: the API is the source of truth (VISION Exclusion 2); local permission logic drifts from the record the CLI faithfully surfaces and second-guesses it — the path surfaces the governing governance, it does not rule on it.
- The path must not fabricate an authority ruling when the record does not clearly answer. **Why**: a guessed "you are / are not allowed" is worse than no answer — it reads as authoritative while being ungrounded; the path must instead say what is unclear and surface what it found.
- The path must not judge or interpret the *substance* of the tension behind the action, or coach Holacracy practice. **Why**: it navigates the record to surface constraints on an action, not facilitate governance (VISION Exclusion 1); interpreting the tension is out of scope entirely.
- The path must not dump raw, unsynthesized command output as its result. **Why**: raw dumps are exactly the rediscovery burden the operating surface exists to remove; the value is the drawn-together picture of what constrains the action, not a concatenation of domain and policy reads.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing read commands (search, role domains, role policies, role reads) and defers to the CLI's built-in help for their exact flags. If a read changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Governance Navigation Path (064)**: the sibling read path. 064 surfaces the governance around a free-form *concern*; this path scopes to a wanted *action* and surfaces what constrains it. The two share composed reads and the read-only, non-ruling stance, but neither reimplements the other's traversal.
- **Glassfrog API**: never touched directly — the CLI mediates every read, and the API enforces the caller's read permissions. The path only ever sees what the caller is allowed to read, and defers to the API/record as the source of truth for what is valid.

---

## Driving Scenarios

### Happy path

**Scenario: A wanted action falls under another role's domain**
Given a practitioner's free-form action and no role or domain in hand
When the path searches the governance record for what the action touches
Then it surfaces the domain that governs the action and the role that holds it
And it surfaces that the action falls under that role's authority, so it would need that role's permission or a proposal
And each element is carried with the id needed to read it again.

**Scenario: A wanted action is shaped by a policy**
Given an action the path has located in the governance record
When the path traverses the policies bearing on it
Then it surfaces the policy that grants or limits the action
And presents it as the constraint the operator must observe, drawn together with any governing domain, not as separate dumps.

**Scenario: A wanted action that nothing in the record constrains**
Given an action for which no domain in view governs it and no policy in view limits it
When the path completes
Then it surfaces that the record shows nothing constraining the action
And it reports that absence plainly rather than asserting the operator is permitted.

### Error scenarios

**Scenario: A read in the discovery fails**
Given a traversal where one read fails (for example, a role-policies read errors)
When the path continues
Then it surfaces what the failure was and returns the picture assembled from the reads that succeeded
And it does not invent the missing piece or abandon the whole picture.

**Scenario: The record does not clearly answer**
Given an action for which the match is ambiguous — no domain plainly owns it, or several partial matches conflict
When the path completes
Then it says the record does not clearly answer and surfaces what it found
And it does not fabricate an authority ruling to resolve the ambiguity.

### Edge cases

**Scenario: The wanted action is too vague to locate its governance**
Given an action described too vaguely to search for the governing domains and policies
When the path begins
Then it asks the operator a clarifying question to sharpen the action
And it does not guess a meaning and traverse on the guess.

**Scenario: An over-broad action matches many models**
Given an action so broad that search returns many roles, domains, and policies
When the path assembles the picture
Then it presents the most relevant governing constraints rather than dumping every match
And it chooses them over the full paged-through result set, not a single page
And makes clear the picture is narrowed, so the practitioner can refine.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced constraint-discovery-path content
When every command and read it composes is checked against the shipped CLI
Then each one exists — the path invents no read the CLI does not expose.

**Scenario: Read-only throughout**
Given the path content
When it is inspected for any write, confirm, or gate step
Then none is present — the path only reads.

**Scenario: Surfacing, not ruling**
Given the path's treatment of the wanted action
When it is inspected for a permission verdict computed from local logic
Then none is present — it surfaces the governing domains and policies drawn from the record and nowhere reimplements permission rules or rules on whether the action is allowed.

**Scenario: No fabricated ruling under uncertainty**
Given the path's handling of an action the record does not clearly constrain
When its result is inspected
Then it states what is unclear and surfaces what it found, and nowhere asserts a "permitted" or "forbidden" verdict it cannot ground in the record.

**Scenario: Synthesized, not raw**
Given the path's result
When it is inspected against raw command output
Then it is a drawn-together picture of what constrains the action, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a read-only agent, or both is decided during shaping. (Mirrors how 062 deferred its skill decomposition and how 064 deferred its own form.)
- **[ASSUMED] Entry is a free-form action**: the path starts from a voiced action and searches to discover what governs it; it does not require any pre-existing structure or id. (The developer confirmed this entry model.)
- **[ASSUMED] Clarifying question when the action is vague**: when the wanted action is too under-specified to search, the path asks the operator to sharpen it before traversing. The developer noted a preference to surface such a question through a structured ask-the-operator mechanism (e.g. an `ask_user_question`-style tool); the exact mechanism is a delivery detail for shaping, so the accord fixes only the behavior — the path asks rather than guesses.
- **Composed reads are already shipped** (technical): Cross-Model Search (041), Role Domains, and Role Policies all exist in the CLI today, so the path composes them rather than waiting on new reads. (Grounded in the FEATURE-MODEL dependency list for this capability.)
- **Traversal depth is relevance-bounded** (technical): how far the traversal follows into a role's interior governance is guided by the action's relevance rather than a fixed level, consistent with "synthesized picture rather than raw dumps." (Informed default; the path is guidance, so depth is judgment the operator applies, not a fixed algorithm.)

---

## Ambiguity Warnings

None remaining — the delivery form, the entry point, the behavior when the action is vague, and the surface-vs-rule boundary were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-17

- **Surface, do not rule**: The path surfaces the governing domains and policies drawn from the record and characterizes the authority situation (own authority / another role's domain / policy-shaped / needs a proposal); it does not compute a permission verdict from local Holacracy logic. When the record does not clearly answer, it says so and surfaces what it found rather than guessing a verdict. This reconciles 064's "065 is the authority judgment" shorthand with the FEATURE-MODEL's "never reimplements permission rules locally": the judgment is the framing of the surfaced record, not local rule evaluation.
- **Entry point**: The path starts from a free-form action description, searches to discover the governance that constrains it, then surfaces it. When the action is too vague to search, the path asks the operator a clarifying question (preferably through a structured ask-the-operator mechanism) rather than guessing.
- **Boundary with Governance Navigation Path (064)**: 064 surfaces the governance around a free-form concern (working a tension); this path scopes to a wanted action and surfaces what constrains it (am I limited by a domain or policy?). Both are read-only and non-ruling.
