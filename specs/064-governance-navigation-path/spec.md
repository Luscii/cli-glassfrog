# Specification: Governance Navigation Path

**Feature**: 064-governance-navigation-path
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Governance Navigation Path is the first **operator path** on the Agent Operating Surface — the guided compositions that ride on top of the Glassfrog CLI, built on the root that *Operator Orientation* (062) established. Where orientation teaches an agent *how to drive* the CLI, a path teaches it *how to accomplish a governance job* by composing commands the CLI already exposes. This path covers the highest-value read job: working a tension. An agent starts from a free-form concern a practitioner has voiced and, without a new API capability, traverses the governance record to a synthesized picture — the roles the concern touches, who fills them, and the domains and policies that shape them — instead of dumping raw command output the agent must then untangle.

The path is knowledge, not capability. It composes already-shipped reads — Cross-Model Search (041) to discover what a free-form concern touches, Role Reads for the roles themselves, Role Fillers (047) for who holds them, and the role-interior reads for domains and policies — and adds no command, flag, or governance logic of its own. It is strictly read-only; capturing the tension is the separate Tension Processing Path (066), and judging whether a wanted action is within authority is the separate Constraint Discovery Path (065). The exact delivery form — a plugin skill, a read-only governance-navigator agent, or both — is decided during shaping; this spec fixes the behavior.

---

## Behavioral Accord

### Entry

- When a practitioner voices a free-form concern (a tension in the loose sense), the path takes that concern as its starting point — no pre-existing tension id is required.
- When the concern already names a specific role the practitioner has in mind, the path can begin from that role directly rather than searching first.

### Traversal

- When the relevant role is not yet known, the path uses cross-model search over the concern to discover which roles, policies, and domains it touches, then follows each result's id into the matching read.
- When a role is in hand, the path reads that role and who fills it, and — where relevant to the concern — the domains it controls and the policies on its interior.
- When a matched role is a circle (contains sub-roles), the path may follow into the sub-roles and their fillers as far as the concern warrants, bounded by relevance rather than walking the whole tree.
- When a search or list read spans more than one page, the path pages through the full result set before choosing what is most relevant — so narrowing is a relevance choice made over the complete set, never a silent drop of unfetched pages.

### Synthesis

- When the traversal is complete, the path returns a synthesized picture — the relevant roles, their fillers, and the governing domains and policies drawn together — rather than a sequence of raw command dumps.
- When the picture is presented, each element carries the id needed to read it again or act on it, so the synthesis bridges back into the CLI's own commands rather than being a dead end.

### Read-only and fidelity

- The path only reads; it never writes to the governance record, and it composes only reads the CLI already exposes — it invents no command, flag, or capability.
- When the CLI's read surface changes, the path is expected to stay consistent with it; guidance that names a read the CLI no longer offers is a defect, not a difference of opinion.

### Surfacing, not judging

- When the traversal surfaces domains and policies around a concern, the path presents them as the governing governance in view — it does not rule on whether a wanted action is permitted; that authority judgment is the Constraint Discovery Path (065).

---

## User Scenarios

**In order to** work a tension without hand-assembling a dozen reads and untangling raw output,
**as an** AI agent,
**I want to** follow a guided path from the practitioner's concern to a synthesized picture of the governance around it.

**In order to** understand who and what my concern touches before I decide what to do,
**as a** practitioner (served through the agent),
**I want to** see the relevant roles, their fillers, and the domains and policies that shape them, drawn together.

**In order to** move from understanding to action without losing my place,
**as an** AI agent,
**I want to** have each element of the picture carry the id that reads it again or feeds the next path.

---

## Non-Behaviors

- The path must not write to the governance record or drive any write command. **Why**: it is a read traversal; capturing a tension or drafting a proposal are separate write paths (066, and the proposal paths) with their own guardrail, and folding a write in here would bypass the Write-Safety Guardrail.
- The path must not add any command, flag, or API capability of its own. **Why**: it is a guided composition of existing reads (knowledge + guardrails, never capability); inventing a surface would break "Bounded by the API surface" and VISION Exclusion 2.
- The path must not judge whether a wanted action is within the practitioner's authority or requires a proposal. **Why**: that evaluative read is the Constraint Discovery Path (065); this path surfaces the governing governance, it does not rule on it — overlapping the two would duplicate the judgment in two places.
- The path must not reimplement or duplicate governance or permission logic locally. **Why**: the API is the source of truth (VISION Exclusion 2); local logic drifts from the spec and second-guesses the record the CLI faithfully surfaces.
- The path must not teach or coach Holacracy practice — interpreting the tension or advising on governance craft. **Why**: it navigates the record to work a tension, not facilitate it (VISION Exclusion 1).
- The path must not dump raw, unsynthesized command output as its result. **Why**: raw dumps are exactly the rediscovery burden the operating surface exists to remove; the value is the drawn-together picture, and returning raw output would make the path indistinguishable from the agent calling the commands itself.
- This spec must not define the plugin's distribution or the path's exact delivery form. **Why**: distribution is Operating-Surface Packaging (070) and the skill/agent decomposition is a shaping decision; fixing them here pre-empts work that should evolve independently.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being driven. The path composes the CLI's existing read commands (search, role reads, fillers, domains, policies) and defers to the CLI's built-in help for their exact flags. If a read changes, the path must follow.
- **Operator Orientation (062) / the plugin**: this path is added to the same Claude plugin and assumes the orientation knowledge (output formats, pagination, exit codes) is available; it builds on that root rather than repeating it.
- **Glassfrog API**: never touched directly — the CLI mediates every read, and the API enforces the caller's read permissions. The path only ever sees what the caller is allowed to read.

---

## Driving Scenarios

### Happy path

**Scenario: From a concern to the roles that touch it**
Given a practitioner's free-form concern and no role in hand
When the path searches the governance record for what the concern touches
Then it surfaces the relevant roles and who fills them
And each is carried with the id needed to read it again.

**Scenario: Drawing in the governing domains and policies**
Given a role the path has identified as relevant to the concern
When the path traverses that role's governance
Then it draws in the domains it controls and the policies on its interior that bear on the concern
And presents them as part of one picture, not separate dumps.

**Scenario: A circle concern follows into its sub-roles**
Given the concern touches a role that is a circle
When the path judges the sub-roles relevant
Then it follows into those sub-roles and their fillers as far as the concern warrants
And stops short of walking the whole tree.

### Error scenarios

**Scenario: The concern matches nothing**
Given a concern for which cross-model search returns no results
When the path completes
Then it reports that nothing relevant was found and suggests refining the concern
And it does not fabricate roles or governance that the record does not contain.

**Scenario: A read in the traversal fails**
Given a traversal where one read fails (for example, a leaf role has no sub-role fillers to roll up, or a read errors)
When the path continues
Then it surfaces what the failure was and returns the picture assembled from the reads that succeeded
And it does not invent the missing piece or abandon the whole picture.

### Edge cases

**Scenario: An over-broad concern matches many models**
Given a concern so broad that search returns many roles, policies, and domains
When the path assembles the picture
Then it presents the most relevant results rather than dumping every match
And it chooses them over the full paged-through result set, not a single page
And makes clear the picture is narrowed, so the practitioner can refine.

**Scenario: The concern is really an authority question**
Given a concern phrased as "am I allowed to do X?"
When the path surfaces the domains and policies that govern X
Then it presents that governing governance
And it hands the authority judgment to the Constraint Discovery Path (065) rather than ruling on permission itself.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced navigation-path content
When every command and read it composes is checked against the shipped CLI
Then each one exists — the path invents no read the CLI does not expose.

**Scenario: Read-only throughout**
Given the path content
When it is inspected for any write, confirm, or gate step
Then none is present — the path only reads.

**Scenario: Surfacing, not judging**
Given the path's treatment of domains and policies
When it is inspected for an authority or permission verdict
Then it only surfaces the governing governance and nowhere rules on whether an action is allowed.

**Scenario: Synthesized, not raw**
Given the path's result
When it is inspected against raw command output
Then it is a drawn-together picture, not a concatenation of unsynthesized dumps.

---

## Assumptions

- **[ASSUMED] Delivery form deferred to shaping**: the path is specified behaviorally; whether it ships as a plugin skill, a read-only governance-navigator agent, or both is decided during shaping. (The developer confirmed the form is deferred, mirroring how 062 deferred its skill decomposition.)
- **[ASSUMED] Entry is a free-form concern**: the path starts from a voiced concern and searches to discover what it touches; it does not require an already-captured `ten_` tension id, and capturing the tension is the separate Tension Processing Path (066). (The developer confirmed this entry model.)
- **Traversal depth is relevance-bounded** (technical): how deep the traversal follows sub-roles and interior governance is guided by the concern's relevance rather than a fixed level, consistent with "synthesized picture rather than raw dumps." (Informed default; the path is guidance, so depth is judgment the operator applies, not a fixed algorithm.)
- **Composed reads are already shipped** (technical): Cross-Model Search (041), Role Reads, Role Fillers (047), and the role-interior domain/policy reads all exist in the CLI today, so the path composes them rather than waiting on new reads. (Grounded in the FEATURE-MODEL dependency list for this capability.)

---

## Ambiguity Warnings

None remaining — the delivery form, the entry point, and the boundary with the Constraint Discovery Path (065) were all resolved during specification. See Clarifications.

---

## Clarifications

### Session 2026-07-17

- **Delivery form**: Deferred to shaping. The spec fixes behavior; whether the path is a plugin skill, a read-only governance-navigator agent, or both is a shaping decision (mirrors 062).
- **Entry point**: The path starts from a free-form concern, searches to discover what it touches, then traverses. It does not require a captured tension id — capture is the separate Tension Processing Path (066).
- **Boundary with Constraint Discovery Path (065)**: This path *surfaces* the governing domains and policies as part of the picture around a tension; the authority question ("can I do X, or does it need a proposal?") is 065. 064 shows the governance; 065 judges it.
