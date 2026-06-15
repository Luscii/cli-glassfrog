# Specification: Operator Orientation

**Feature**: 062-operator-orientation
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Operator Orientation is the **root of the Agent Operating Surface** — the thin, agent-facing layer that rides on top of the Glassfrog CLI. Today an AI agent driving the CLI has no packaged operating knowledge, so it rediscovers how to operate the CLI every session (how to get parseable output, how to page, what exit codes mean, how credentials are supplied) and can mis-drive it. Operator Orientation packages that knowledge so the agent operates the CLI correctly from the first session without rediscovery.

The knowledge is defined as a **Claude plugin** composed of one or more **skills**; the exact skill breakdown is decided during shaping, not here. This spec defines the plugin — its manifest and the orientation content it carries. It does **not** define how the plugin is distributed: publishing it through its own marketplace so an agent environment can discover and install it remains *Operating-Surface Packaging* (the former backlog #70). So this spec **partially** absorbs that item — the plugin definition lands here, its distribution does not. Critically, the layer adds **no API capability of its own** — every fact it packages is about commands the CLI already exposes. It is knowledge and packaging, never a new surface. Two further sibling capabilities build on this root but are **out of scope here**: the Write-Safety Guardrail enforces write-safety, and the operator paths (navigation, tension processing, proposal flows) compose CLI commands into guided journeys — both arrive later as additional skills in the same plugin.

---

## Behavioral Accord

### Plugin definition

- When the plugin is defined, it carries a manifest that identifies it and packages the orientation knowledge as skill content the agent can consult.
- The plugin relies on the CLI's existing credential setup — it introduces no separate credential mechanism of its own.
- When the agent encounters something about driving the CLI it does not already know, it consults the orientation on demand — orientation is reached for when needed, not required to be loaded before the agent's first command.

### Orientation coverage

- When an agent needs per-command or per-flag detail, the orientation points it at the CLI's own built-in help/discovery rather than duplicating it — the orientation packages cross-cutting operating knowledge, not a command catalogue.
- When an agent needs machine-parseable results, the orientation explains how to select a structured output format and what shape to expect, so the agent parses output reliably rather than scraping human-rendered text.
- When a read returns more results than fit in one response, the orientation explains how to detect that more pages exist and how to fetch them.
- When a command exits non-zero, the orientation explains what each exit code means and the appropriate reaction.
- When the agent must authenticate, the orientation explains how the CLI discovers and accepts credentials (the `X-Auth-Token` API key).

### Write-safety as guidance, not enforcement

- When the orientation covers commands that write to the governance record, it describes the write-safety expectation — confirm before writing; on a stale-write (`412`) refusal, re-read and re-confirm rather than blindly retrying — as guidance the agent is told to follow.
- The orientation does not itself gate, confirm, or block any write; enforcing write-safety is the separate Write-Safety Guardrail capability.

### Fidelity to the shipped CLI

- The orientation describes only behavior the CLI actually exposes; it adds no command, flag, or capability beyond what the CLI (and thus the v5 API) already provides.
- When the CLI's command surface changes, the orientation is expected to stay consistent with it — documented behavior that no longer matches the shipped CLI is a defect, not a difference of opinion.

---

## User Scenarios

**In order to** operate the CLI correctly from the first session without rediscovering its surface,
**as an** AI agent,
**I want to** consult packaged orientation covering output formats, pagination, exit codes, credentials, and where to find per-command detail.

**In order to** have a single installable unit that carries the operating knowledge,
**as a** practitioner (or whoever provisions the agent),
**I want to** the operating surface defined as a Claude plugin I can later install into the agent's environment.

**In order to** avoid mis-driving governance writes before the enforcing guardrail exists,
**as an** AI agent,
**I want to** be told the write-safety expectations as part of orientation.

---

## Non-Behaviors

- The orientation must not add any API capability, command, or flag of its own. **Why**: it is knowledge + packaging riding on the CLI; adding capability would break "Bounded by the API surface" and VISION Exclusion 2, turning a guide into a second, drifting surface.
- The orientation must not duplicate the per-command and per-flag detail the CLI's own help already provides. **Why**: duplicated command detail is the largest drift surface — it goes stale the moment a command or flag changes; pointing at the CLI's built-in help keeps that detail single-sourced in the CLI.
- This spec must not define how the plugin is distributed — its own marketplace, publishing, or the install flow. **Why**: distribution is *Operating-Surface Packaging* (#70); defining it here pre-empts that item and couples the plugin's content to a delivery mechanism that should evolve independently.
- The orientation must not enforce, gate, or block writes. **Why**: write-safety gating is the Write-Safety Guardrail; folding enforcement in here blurs the knowledge/guardrail boundary and would duplicate that capability in two places.
- The orientation must not reimplement or duplicate governance or validation logic locally. **Why**: the API is the source of truth (VISION Exclusion 2); local logic inevitably drifts from the spec.
- The orientation must not teach or coach Holacracy practice. **Why**: it orients on *driving the CLI*, not facilitating governance (VISION Exclusion 1); conflating the two pulls the surface toward coaching it explicitly excludes.
- This spec must not define the operator paths (navigation, tension processing, proposal flows). **Why**: those are separate downstream skills added to the same plugin later; this spec establishes the plugin foundation and the orientation knowledge only, and overreaching would pre-empt their shaping.

---

## Integration Boundaries

- **Glassfrog CLI**: the thing being operated. Orientation is read-only knowledge *about* the CLI; it invokes nothing on its own behalf. For per-command detail it defers to the CLI's built-in help. If the CLI changes, orientation must follow.
- **Agent environment / plugin host**: the plugin is defined to be installable into an agent's environment as a Claude plugin. The marketplace that distributes it and the install flow itself are out of scope (#70). Once the plugin is present, orientation is available; if it is absent, the agent simply falls back to rediscovery — nothing breaks.
- **Glassfrog API**: not touched by orientation directly; the CLI mediates all API access. Orientation only points at how credentials for the API are supplied.

---

## Driving Scenarios

### Happy path

**Scenario: The plugin makes orientation consultable**
Given the plugin is present in an agent's environment
When the agent looks for operating knowledge
Then the orientation knowledge is available to consult
And no configuration beyond the CLI's existing credential setup is required.

**Scenario: Getting parseable output by consulting orientation**
Given an agent that needs to read a practitioner's roles
When it consults the orientation for how to obtain machine-parseable results
Then it learns how to select a structured output format and the shape to expect
And it drives the read correctly without scraping human-rendered text.

**Scenario: Paging through a large result set**
Given a list command that returns more results than one response holds
When the agent consults the orientation on pagination
Then it learns how to detect that more pages exist and how to fetch them.

**Scenario: Per-command detail comes from the CLI, not the orientation**
Given an agent that needs the exact flags for one specific command
When it consults the orientation
Then the orientation directs it to the CLI's built-in help for that command
And does not itself enumerate the command's flags.

### Error scenarios

**Scenario: Reacting to a non-zero exit code**
Given a command that has just exited non-zero
When the agent consults the orientation for that exit code
Then it learns what the code means and the appropriate reaction.

**Scenario: Missing credentials**
Given an agent that has not yet supplied a credential
When a command fails for lack of authentication
Then the orientation directs the agent to the CLI's credential setup (the `X-Auth-Token` API key).

### Edge cases

**Scenario: Guidance precedes enforcement on a governance write**
Given the Write-Safety Guardrail does not yet exist
When the agent is about to run a command that writes to the governance record
Then the orientation surfaces the write-safety expectation (confirm first; on a `412` stale-write refusal, re-read and re-confirm)
And it does not block or gate the write itself.

**Scenario: Cross-cutting knowledge drifts from the shipped CLI**
Given the CLI's exit-code or output-format behavior has changed
When the orientation is checked against the shipped CLI
Then the mismatch counts as a defect to fix, not an acceptable difference.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No invented surface**
Given the produced orientation content
When every command, flag, and capability it names is checked against the shipped CLI
Then each one exists in the CLI — the orientation invents no surface the CLI does not expose.

**Scenario: No distribution defined**
Given the plugin definition produced here
When it is inspected for distribution machinery (a marketplace, publishing, or an install flow)
Then none is present — distribution is left to #70.

**Scenario: Guidance, not gating**
Given the orientation's treatment of governance writes
When it is inspected for enforcement behavior
Then it only describes the write-safety expectation and nowhere implements confirmation, gating, or blocking.

**Scenario: No Holacracy coaching**
Given the orientation content
When it is inspected for governance-practice instruction
Then it contains only how to drive the CLI, with no Holacracy coaching or tension interpretation.

---

## Assumptions

- **[ASSUMED] Skill decomposition deferred**: the plugin will comprise one or more skills; whether orientation is a single skill or several is decided during shaping, not in this spec. (The developer confirmed the form is a Claude plugin of skills "defined/shaped later.")
- **[ASSUMED] Packaging partially absorbed**: this spec defines the Claude plugin (its manifest and the orientation skill content); distribution through its own marketplace remains *Operating-Surface Packaging* (#70). (The developer chose to land the plugin definition here but keep the marketplace/distribution in #70.)
- **Drift guard feasibility** (technical): the orientation is hand-authored; an automated CI/test check that flags drift between documented and actual behavior is desirable but may prove only partially feasible or infeasible. Its scope is a planning decision, and shipping with a partial guard — or none — is acceptable. Deferring per-command detail to the CLI's own help keeps the drift surface small. (The developer expressed a preference for a drift check while acknowledging it may be hard.)

---

## Ambiguity Warnings

None remaining — both ambiguities raised during specification (command-surface depth and consumption model) were resolved during clarification. See Clarifications.

---

## Clarifications

### Session 2026-06-15

- **Command-surface depth**: The orientation packages cross-cutting operating knowledge only (output formats, pagination, exit codes, credentials, write-safety) and points the agent at the CLI's own built-in help for per-command/flag detail — the smallest drift surface, aligned with the project's anti-drift value.
- **Consumption model**: Orientation is consulted on demand (the description-triggered skill model), not required to load before the agent's first command.
- **Scope — plugin vs. distribution**: This spec partially absorbs *Operating-Surface Packaging* (#70). It defines the Claude plugin (manifest + orientation skill content) but keeps the marketplace and distribution/install flow in #70.
