# Specification: Command Registration

**Feature**: 001-command-registration
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

The Glassfrog CLI is a command-line surface over the Glassfrog v5 API. Before any individual command — read roles, capture a tension, submit a proposal — can exist, the CLI needs a way to know which commands it offers and how they are organized. **Command Registration** is that foundation: the mechanism by which a command (its name, a one-line summary, and the action it performs) is declared and becomes part of the CLI's known command set. Commands may be organized into **groups** (e.g. `roles`, `proposals`) that contain subcommands (`roles list`, `roles get`), mirroring the domain's resource-and-verb shape.

This capability serves the **Maintainer** — the person extending the CLI. Its defining property is modularity: adding a new command attaches it without editing unrelated commands or a central, hand-maintained list. The registered set is the single source of truth that the sibling capabilities (Argument Dispatch, Help & Version) later read; Command Registration itself only builds and guards that set.

---

## Behavioral Accord

### Registration

- When a leaf command is registered with a name, a one-line summary, and an action, it becomes part of the CLI's known command set.
- When a group is registered with a name, a one-line summary, and one or more children, the group and each child become known, addressable by their full path (e.g. `roles list`). A child may itself be a leaf command or another group, so groups may nest to any finite depth (e.g. `proposals changes add`).
- When a command or group is registered alongside existing ones, the existing commands are unaffected — registering one command never requires or causes changes to unrelated ones.
- Registration occurs during CLI initialization, before any user invocation is handled.

### Lookup & enumeration

- When the known command set is queried for a path (a top-level name, or a group name followed by the names of its descendants), the matching command or group is returned if it exists, and nothing is returned if it does not.
- When the known command set is queried for a group name on its own (no descendant path), the group node is returned. A group has no action of its own; resolving to it exposes its child commands so dispatch and Help & Version can act on them.
- When the known command set is enumerated, every registered command and group is listed with its name and summary.

### Validation & errors

- When a name is registered that is already taken at the same level (same parent), registration fails with an error naming the conflicting name, before any user command runs.
- When a command or group is registered with an empty or whitespace-only name, registration fails with an error.
- When a command or group is registered with an empty or whitespace-only summary, registration fails with an error.
- When a leaf command is registered without an action, registration fails with an error.
- When a group is registered with no children, registration fails with an error.

---

## User Scenarios

**In order to** add a new command without risking the ones already there,
**as a** Maintainer,
**I want to** register my command in isolation
so that it attaches to the CLI without editing unrelated commands or a central dispatch list.

**In order to** mirror the API's resource-and-verb structure,
**as a** Maintainer,
**I want to** register commands inside named groups
so that related operations (e.g. all `roles` reads) live together and are addressed as `roles <verb>`.

**In order to** catch mistakes before users hit them,
**as a** Maintainer,
**I want to** have a duplicate or malformed registration fail at startup
so that a colliding command name never silently shadows another.

---

## Non-Behaviors

- The system must not parse invocation arguments or decide which command the user typed. **Why**: that is Argument Dispatch's responsibility; Command Registration only builds the set dispatch reads. Folding parsing in here couples the source-of-truth structure to input handling and makes both harder to evolve.
- The system must not render help, usage, or the command listing for display. **Why**: Help & Version owns presentation. Registration stores the one-line summary as data; formatting it is a separate concern that would otherwise bloat this capability.
- The system must not execute a registered command's action. **Why**: invoking the action is dispatch's job at runtime. Keeping registration execution-free preserves the clean split between the static command set and runtime behavior.
- The system must not define or emit process exit codes. **Why**: Exit-Code Convention owns process-lifecycle signaling; registration has no knowledge of how a command run ends.
- The system must not discover or load commands dynamically at runtime (plugins, directory scanning), nor allow registration after dispatch has begun. **Why**: the command set is fixed at initialization; runtime mutation would make the known set unpredictable for dispatch and help, and plugin loading is far beyond the skeleton's purpose.

---

## Integration Boundaries

- **Argument Dispatch (sibling capability)**: reads the known command set to resolve a typed invocation to a registered command. Command Registration produces; dispatch consumes. When the set is empty, dispatch has nothing to resolve.
- **Help & Version (sibling capability)**: enumerates the known command set to produce the command listing and per-command usage, consuming names and summaries only.
- **No external systems**: this capability is entirely internal to the CLI process and touches no network, file, or API boundary.

---

## Driving Scenarios

### Happy path

**Scenario: Register a top-level leaf command**
Given an empty command set
When a leaf command `version` is registered with a summary and an action
Then querying the set for `version` returns that command
And enumerating the set includes `version` with its summary.

**Scenario: Register a group with subcommands**
Given an empty command set
When a group `roles` is registered with a summary and subcommands `list` and `get`
Then querying for the path `roles list` returns the list command
And querying for `roles get` returns the get command
And enumerating the set includes the `roles` group and both subcommands.

**Scenario: Look up a group on its own**
Given the group `roles` is registered with subcommands `list` and `get`
When the set is queried for `roles` with no descendant path
Then the `roles` group node is returned
And its child commands `list` and `get` are reachable through it.

**Scenario: Register unrelated commands independently**
Given a command set already containing the `roles` group
When a `proposals` group is registered
Then both `roles` and `proposals` are present in the set
And the `roles` group is unchanged.

### Error scenarios

**Scenario: Duplicate top-level name**
Given the name `roles` is already registered at the top level
When another command is registered under the name `roles` at the top level
Then registration fails with an error naming `roles`
And it fails before any user command runs.

**Scenario: Empty command name**
Given any registration state
When a command is registered with an empty or whitespace-only name
Then registration fails with an error.

**Scenario: Command without a summary**
Given any registration state
When a command or group is registered with an empty or whitespace-only summary
Then registration fails with an error.

**Scenario: Leaf command without an action**
Given any registration state
When a leaf command is registered with no action
Then registration fails with an error.

### Edge cases

**Scenario: Same subcommand name under different groups**
Given the groups `roles` and `proposals` are registered
When each group registers a `get` subcommand
Then both succeed, because `get` is unique within its own parent
And `roles get` and `proposals get` resolve independently.

**Scenario: A group nested within a group**
Given a group `proposals` is registered
When a subgroup `changes` containing the subcommand `add` is registered under it
Then the path `proposals changes add` resolves to the add command.

**Scenario: Group with no children**
Given any registration state
When a group is registered with zero children
Then registration fails with an error, because an empty group is unreachable.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Lookup is predictable from registration alone**
Given only this specification
When a reader registers a group `roles` with a subcommand `list`
Then they can state that `roles list` resolves to the list command, that `roles` alone resolves to the group, and that enumerating the set lists the `roles` group — without consulting any code.

**Scenario: No implementation leakage**
Given the specification text
When it is scanned for technology names
Then no programming language, framework, library, or concrete data-structure choice appears.

**Scenario: Each excluded concern names its owner**
Given the Non-Behaviors section
When each non-behavior is read
Then it names the sibling capability that owns the excluded concern.

---

## Assumptions

- **Binary name**: assumed the CLI is invoked as `glassfrog`, and examples use that name; the registration mechanism itself is independent of the binary name. (Informed by the project name.)
- **Initialization-time registration**: assumed all commands are registered once during program startup, producing a fixed command set. (Technical default; matches the no-runtime-mutation boundary above.)
- **Nesting depth**: confirmed during clarification — groups may nest to arbitrary finite depth (a group within a group, e.g. `proposals changes add`). The common case remains a single level.

---

## Ambiguity Warnings

None remaining. The two warnings raised at specification time — nesting depth and whether a summary is required at registration — were resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-03

- **Nesting depth**: Groups may nest to arbitrary finite depth — a group's children may themselves be groups (e.g. `proposals changes add`). The registry carries a recursive tree, not a single level of grouping.
- **Summary at registration**: A one-line summary is required for every command and group. A missing or whitespace-only summary fails registration at startup, alongside the existing missing-name and missing-action errors. This guarantees Help & Version always has a description to show.
- **Bare group lookup**: Querying a group name on its own resolves to the group node itself (which has no action) so dispatch and Help & Version can reach and list its child commands — rather than treating a bare group name as a miss.
