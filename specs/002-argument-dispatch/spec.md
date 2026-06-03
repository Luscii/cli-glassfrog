# Specification: Argument Dispatch

**Feature**: 002-argument-dispatch
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Argument Dispatch is the runtime counterpart to Command Registration (001). Where registration builds the known command set, dispatch takes a raw invocation — the tokens a caller types after `glassfrog` — and resolves them against that set to route to the correct command, or reports that no such command exists. It is what makes registered commands actually runnable.

This capability is the second of the CLI Skeleton (problem: *No Runnable CLI*), and the first whose behavior the end caller — an AI agent acting for a practitioner, or the practitioner directly — observes on every invocation. Command Registration's spec explicitly reserved this surface for dispatch: deciding which registered command a typed invocation names, and what happens when it names none. Dispatch only *reads* the command set; it never changes it, renders help, or decides exit codes — those belong to Command Registration, Help & Version, and Exit-Code Convention respectively.

---

## Behavioral Accord

### Resolution

- When the invocation names a registered command path exactly (a top-level name, or a group name followed by the names of its descendants), dispatch routes to that command and the command's action runs.
- When the invocation names a registered group with no further subcommand, dispatch resolves to the group and routes to a help/listing outcome — the group has no action of its own.
- When the invocation has no tokens at all (`glassfrog` alone), dispatch resolves to the root and routes to the same help/listing outcome.

### Matching

- Matching is exact at every level: a token resolves only if it equals a registered name at that level. Dispatch does not match by prefix or abbreviation, so a token that is not an exact registered name never routes to a longer command.

### Unknown command

- When the first token that does not match any registered command at its level is encountered, dispatch reports an unknown-command error that names the unrecognized token and points the caller to help. Where a close registered name exists, it may suggest it ("did you mean …") on a best-effort basis — the suggestion is not a guaranteed part of the contract. This is a usage-error outcome — dispatch classifies it as such; the concrete process exit code is Exit-Code Convention's concern.

### Invalid input

- When the invocation includes a flag or positional argument that the resolved command does not accept, dispatch reports a usage error and the command does not run. Unexpected input is never silently ignored.

---

## User Scenarios

**In order to** run a registered command,
**as an** AI agent acting for a practitioner,
**I want to** type its full path and have the invocation routed to that command and executed.

**In order to** recover quickly from a typo,
**as an** operator,
**I want** an unknown command to tell me which token wasn't recognized and how to get help.

**In order to** discover what a group offers,
**as an** operator,
**I want to** type the group name alone to surface its available subcommands.

---

## Non-Behaviors

- The system must not render help text or the command listing itself. **Why**: dispatch only *routes* a group or unknown invocation to a help/usage outcome; Help & Version owns presentation. Rendering here would duplicate and pre-empt that capability.
- The system must not define or emit process exit codes. **Why**: dispatch classifies an outcome (success vs usage error); Exit-Code Convention encodes that classification into a process code. Emitting codes here would fracture the single owner of process-lifecycle signaling.
- The system must not register, add, or modify commands. **Why**: the command set is Command Registration's domain; dispatch is strictly read-only over the registered tree, so the set it routes against is always the one registration built.
- The system must not match commands by prefix or abbreviation. **Why**: exact matching keeps routing predictable for AI agents and prevents resolution from silently changing when a new sibling command is later registered.
- The system must not perform a command's actual work (API calls, governance reads). **Why**: dispatch hands control to the resolved command; what that command does is its own spec's concern. Dispatch ends at routing.

---

## Integration Boundaries

- **Command Registration (001, upstream)**: dispatch reads the known command set to resolve a typed path. When the set is empty, no invocation resolves except the root help outcome.
- **Help & Version (sibling)**: dispatch routes group, root, and unknown-command outcomes toward a help/usage rendering; Help & Version produces the actual text.
- **Exit-Code Convention (sibling)**: dispatch labels each outcome's category (success or usage error); Exit-Code Convention maps the category to a process exit code.
- **No external systems**: dispatch is entirely in-process and touches no network, file, or API boundary.

---

## Driving Scenarios

### Happy path

**Scenario: Route a nested leaf command**
Given a `roles` group with a `list` subcommand is registered
When the caller invokes `glassfrog roles list`
Then dispatch routes to the `list` command
And the `list` command's action runs.

**Scenario: Route a top-level leaf command**
Given a `version` command is registered at the top level
When the caller invokes `glassfrog version`
Then dispatch routes to `version`
And its action runs.

**Scenario: Bare group surfaces its subcommands**
Given a `roles` group with `list` and `get` subcommands is registered
When the caller invokes `glassfrog roles` with no further token
Then dispatch resolves to the `roles` group
And routes to a help/listing outcome showing `list` and `get`
And the outcome is a success.

### Error scenarios

**Scenario: Unknown top-level command**
Given no command named `rolez` is registered
When the caller invokes `glassfrog rolez`
Then dispatch reports an unknown-command error naming `rolez`
And points the caller to help
And classifies the outcome as a usage error.

**Scenario: Unknown subcommand under a known group**
Given a `roles` group with a `list` subcommand is registered
When the caller invokes `glassfrog roles lst`
Then dispatch reports an unknown-command error naming `lst`
And may suggest `list` as the closest match
And classifies the outcome as a usage error.

**Scenario: Unexpected flag is rejected**
Given a `roles list` command is registered
When the caller invokes `glassfrog roles list --bogus`
Then dispatch reports a usage error naming the unexpected `--bogus`
And the `list` command does not run.

### Edge cases

**Scenario: A prefix does not resolve to a longer command**
Given `roles` is the only registered command beginning with `ro`
When the caller invokes `glassfrog ro list`
Then dispatch does not route to `roles`
And reports an unknown-command error naming `ro`.

**Scenario: Empty invocation resolves to root help**
Given any registered command set
When the caller invokes `glassfrog` with no tokens
Then dispatch resolves to the root
And routes to a help/listing outcome
And the outcome is a success.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No routing depends on abbreviation**
Given this specification
When every routing scenario is read
Then each names a full, exact command path — none relies on a prefix or abbreviation resolving.

**Scenario: Each non-behavior names its owning capability**
Given the Non-Behaviors section
When each non-behavior is read
Then it names the capability that owns the excluded concern (Command Registration, Help & Version, or Exit-Code Convention).

**Scenario: Specification names no implementation technology**
Given the specification text
When it is scanned for technology names
Then no programming language, framework, library, or concrete data-structure choice appears.

---

## Assumptions

- **Binary name**: assumed the CLI is invoked as `glassfrog`; dispatch operates on the tokens following it. (Informed by the project name and 001.)
- **Shared command set**: assumed dispatch resolves against the same command set Command Registration (001) builds — dispatch adds no commands of its own. (Technical default; the dependency is explicit in the feature model.)
- **Exact matching confirmed**: prefix/abbreviation matching is excluded by decision, not assumption (recorded as a non-behavior).

---

## Ambiguity Warnings

None remaining. The two warnings raised at specification time — handling of unexpected input, and whether the "did you mean" suggestion is required — were resolved during clarification (see Clarifications).

---

## Clarifications

### Session 2026-06-03

- **Unexpected input is rejected, not tolerated**: an unknown flag or unexpected positional argument is a usage error and the command does not run. This reverses the specify-time "tolerate" answer once the conflict with CONSTITUTION III (Fail Safe — errors never hidden) and II (Action Transparency — the operator must be able to tell what happened) was surfaced; it also matches the command framework's default behavior. Captured in the Behavioral Accord's new "Invalid input" section and the "Unexpected flag is rejected" scenario.
- **"Did you mean" suggestion is best-effort**: the unknown-command contract requires naming the unrecognized token and pointing to help; a closest-match suggestion may accompany it when one exists but is not a guaranteed, tested part of the contract.
