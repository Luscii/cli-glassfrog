# Specification: Operating-Surface Packaging

**Feature**: 070-operating-surface-packaging
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Operating-Surface Packaging is the **distribution vehicle** for the Agent Operating Surface. Operator Orientation (062) defined the operating surface as a Claude plugin — a manifest, orientation knowledge, operator-path skills, agents, and a write-safety hook — but deliberately left out *how that plugin reaches an agent environment*. This spec closes that gap: it ships a **repo-shipped Claude plugin marketplace** so an agent environment can **discover**, **install**, and **run** the plugin straight from this repository, and it adds a **`glassfrog-setup` skill** that walks a fresh environment from "plugin installed" to "CLI present and authenticated."

The boundary is **distribution only**. The plugin definition, its orientation content, and every operator path already live in 062 and its successors (063–069); this spec adds no API capability, no operating knowledge, and no command surface. It ships a general Claude marketplace that lists the glassfrog plugin today and can list sibling plugins (e.g. a future Holacracy-practice plugin) later, plus a setup skill that leans on the CLI's *existing* install channels and credential mechanism — the CLI stays self-contained; the setup skill reimplements neither. This honors PROJECT scope "Agent operating surface" and the surface's founding constraint: knowledge and packaging, never capability.

---

## Behavioral Accord

### Discovery

- When an agent environment adds this repository as a Claude plugin marketplace, the marketplace is found and it lists the glassfrog operating-surface plugin as an installable entry.
- When the marketplace is inspected, it presents as a **general** Claude marketplace — one that carries the glassfrog plugin now and can list additional plugins later without being restructured.

### Install and run

- When an agent environment installs the glassfrog plugin from the marketplace, the plugin's skills, agents, and write-safety hook become available in that environment.
- When the plugin is installed, it runs against the separately-installed `glassfrog` CLI — packaging introduces no runtime of its own and bundles no binary.

### Setup skill — getting to ready

- When an operator invokes the setup skill, it checks whether the `glassfrog` CLI is present and runnable in the environment.
- When the CLI is absent, the setup skill points the operator at the CLI's existing install channels (the install script, Homebrew tap, or npm wrapper) rather than installing or bundling the binary itself.
- When the CLI is present, the setup skill checks whether a working credential is configured (an authenticated identity read succeeds).
- When no working credential is configured, the setup skill guides the operator through the CLI's existing credential setup (the `X-Auth-Token` API key), introducing no separate credential mechanism.
- When both the presence check and the auth check pass, the setup skill reports the environment as ready to drive the CLI through the operating surface.

### Consistency with what it distributes

- When the marketplace lists the glassfrog plugin, the entry stays consistent with the plugin it points at — the named plugin resolves to a real plugin definition in this repo whose identity matches.
- When the marketplace entry no longer matches the plugin it distributes (a missing plugin, a mismatched name, an unresolvable source), that is a defect to fix, not an acceptable difference.

---

## User Scenarios

**In order to** get the operating surface into my agent's environment without hand-copying files or rediscovering where it lives,
**as a** practitioner (or whoever provisions the agent),
**I want to** add this repo as a Claude marketplace and install the glassfrog plugin from it.

**In order to** start driving the CLI immediately after installing the plugin,
**as an** AI agent,
**I want to** run a setup skill that confirms the CLI is installed and I am authenticated, and tells me exactly what to do when either is missing.

**In order to** add a second operating-surface plugin later without standing up new distribution machinery,
**as a** maintainer,
**I want** the marketplace to be a general one the glassfrog plugin is simply the first entry of.

---

## Non-Behaviors

- Packaging must not define or duplicate the plugin's orientation content, operator paths, or the write-safety hook. **Why**: those are owned by 062–069; redefining them here creates a second source that drifts from the plugin it is meant to distribute.
- Packaging must not add any API capability, command, or flag. **Why**: the surface is knowledge + packaging riding on the CLI (VISION Exclusion 2); adding capability turns a distribution vehicle into a second, drifting surface.
- The setup skill must not reimplement, bundle, or fork the CLI's installation. **Why**: the CLI is self-contained and distributed through its own channels (install script, Homebrew, npm); a second installer would drift from them and split the source of truth for how the binary is delivered.
- The setup skill must not introduce a credential mechanism of its own. **Why**: authentication lives in the CLI's existing `X-Auth-Token` setup; a parallel credential path would fragment where secrets are configured and how they are validated.
- The marketplace must not be locked to a single plugin such that adding a sibling requires restructuring. **Why**: the surface anticipates further plugins (e.g. Holacracy practice); a glassfrog-only marketplace would force a rebuild the moment a second plugin is ready.
- The setup skill must not enforce, gate, or block CLI writes. **Why**: write-safety gating is the Write-Safety Guardrail (063); folding enforcement into setup blurs the knowledge/guardrail boundary and duplicates that capability.
- Packaging must not teach or coach Holacracy practice. **Why**: it distributes a surface for *driving the CLI*, not for facilitating governance (VISION Exclusion 1).

---

## Integration Boundaries

- **Claude plugin host / marketplace mechanism**: consumes the repo-shipped marketplace manifest to discover, install, and run the plugin. Packaging produces the manifest; the host performs add/install/run. If the host is absent, nothing is installed and the surface simply isn't available — the CLI itself is unaffected.
- **The glassfrog plugin (062–069)**: the artifact being distributed. The marketplace entry points at the plugin definition already in this repo; the setup skill is a new skill added to that same plugin. Packaging references the plugin; it does not redefine its content.
- **The glassfrog CLI and its distribution channels (install script, Homebrew tap, npm wrapper)**: the runtime the plugin drives. The setup skill *checks* for the CLI and *points at* these channels when it is missing; it never installs the binary. The CLI stays self-contained.
- **The CLI's credential setup (`X-Auth-Token`)**: the authentication the setup skill verifies and guides toward. The setup skill reads through the CLI to confirm an authenticated identity; it stores and validates no credential of its own.

---

## Driving Scenarios

### Happy path

**Scenario: Discovering the plugin through the marketplace**
Given an agent environment with a Claude plugin host
When the environment adds this repository as a plugin marketplace
Then the marketplace is found
And it lists the glassfrog operating-surface plugin as an installable entry.

**Scenario: Installing and running the plugin**
Given the repository has been added as a marketplace
When the environment installs the glassfrog plugin from it
Then the plugin's skills, agents, and write-safety hook become available in that environment.

**Scenario: A ready environment reported ready**
Given the plugin is installed, the `glassfrog` CLI is present, and a working credential is configured
When the operator invokes the setup skill
Then it confirms both the CLI presence and the authenticated identity
And reports the environment as ready to drive the CLI.

### Error scenarios

**Scenario: The CLI is not installed**
Given the plugin is installed but the `glassfrog` CLI is not present in the environment
When the operator invokes the setup skill
Then it reports the CLI as missing
And directs the operator to the CLI's existing install channels
And it does not attempt to install or bundle the binary itself.

**Scenario: No working credential is configured**
Given the `glassfrog` CLI is present but no working credential is configured
When the operator invokes the setup skill
Then the auth check fails
And the skill guides the operator through the CLI's existing `X-Auth-Token` credential setup
And it introduces no credential mechanism of its own.

### Edge cases

**Scenario: The marketplace entry drifts from the plugin**
Given the marketplace lists the glassfrog plugin
When the entry no longer resolves to a matching plugin definition in the repo
Then the mismatch counts as a defect to fix, not an acceptable difference.

**Scenario: A second plugin is added to the marketplace**
Given the marketplace ships with the glassfrog plugin as its only entry
When a sibling operating-surface plugin is later added
Then it is listed as an additional entry
And the marketplace requires no restructuring to carry it.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Distribution only — no new surface**
Given the packaging artifacts produced here
When they are inspected for orientation content, operator paths, commands, or API capability
Then none is present — every operating fact still lives in the plugin (062–069), and packaging adds only the marketplace and the setup skill.

**Scenario: The CLI stays self-contained**
Given the setup skill
When it is inspected for how it handles a missing CLI or a missing credential
Then it only checks and points at the CLI's existing install channels and credential setup — it installs no binary and stores no credential of its own.

**Scenario: The marketplace is general, not glassfrog-locked**
Given the marketplace manifest
When it is inspected for whether it can carry more than one plugin
Then its shape admits additional plugin entries without restructuring.

**Scenario: The marketplace matches what it ships**
Given the marketplace entry for the glassfrog plugin
When the named plugin and its source are resolved against the repo
Then they point at the real plugin definition and its identity matches.

---

## Assumptions

- **[ASSUMED] Marketplace mechanism is the Claude plugin marketplace**: distribution is a repo-shipped `.claude-plugin/marketplace.json`-style manifest that a Claude plugin host consumes via add/install, with `source` resolving to the in-repo `plugin/` definition. (Follows FEATURE-MODEL's "repo-shipped, its own marketplace" and the existing `plugin/.claude-plugin/plugin.json`; the exact manifest fields are a shaping/planning detail.)
- **[ASSUMED] Setup skill ships inside the plugin**: `glassfrog-setup` is a new skill added to the same plugin the marketplace distributes, not a standalone artifact. (The developer scoped it as part of this packaging work; the operating surface is delivered as one plugin.)
- **Auth check via an identity read** (technical): the setup skill confirms authentication by having the CLI perform a low-cost authenticated read (identity/`me`) and observing success, rather than inspecting stored secrets directly. (Leans on the CLI as the source of truth for whether a credential works, matching the surface's "never reimplement CLI behavior" value; the precise command is a planning detail.)

---

## Ambiguity Warnings

None remaining — the three boundary questions raised during specification (CLI-prerequisite handling, marketplace generality, and the consistency guard) were resolved during the conversation. See Clarifications.

---

## Clarifications

### Session 2026-07-21

- **CLI prerequisite**: The CLI stays self-contained and is *not* installed by packaging. Instead, a `glassfrog-setup` skill performs a presence check and an auth check, and — when either fails — points the operator at the CLI's existing install channels and `X-Auth-Token` credential setup. Presence-check and auth-check are explicitly desired.
- **Marketplace scope**: A **general** Claude marketplace, not a glassfrog-dedicated one. It ships the glassfrog plugin as its first entry and is intended to list future sibling plugins (e.g. a Holacracy-practice plugin) without restructuring.
- **Consistency guard**: In scope as a **light consistency non-behavior** — the marketplace entry must resolve to the plugin it names and stay consistent with it; a drifted or unresolvable entry is a defect.
