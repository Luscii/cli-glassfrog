# Specification: Help & Version

**Feature**: 003-help-and-version
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Help & Version is the presentation surface of the CLI Skeleton (problem: *No Runnable CLI*). Command Registration (001) builds the known command set and stores a required one-line summary for every command and group; Argument Dispatch (002) routes a typed invocation to a command, a group, the root, or an unknown-command error. Both capabilities deliberately refuse to render any text for display — they reserved that concern for this capability. Help & Version owns it: turning the registered set into a **command listing**, turning a single command into its **per-command usage**, and reporting the CLI's **version**.

This is the first capability whose entire job is what the caller *reads*. Its operator is usually an AI agent acting for a practitioner, so its output must be stable and predictable — the same invocation produces the same layout every run, letting an agent discover the command surface and confirm which build it is talking to without guessing. It is a read-only consumer: it never registers commands, resolves invocations, or decides exit codes — those belong to Command Registration, Argument Dispatch, and Exit-Code Convention respectively.

---

## Behavioral Accord

### Command listing

- When help is requested for the root (a bare `glassfrog` invocation routed in by dispatch, or `glassfrog --help`), the system produces a listing of every top-level command and group, each shown with its one-line summary.
- When help is requested for a group (a bare group invocation routed in by dispatch, or `glassfrog <group> --help`), the system produces the group's own path and summary followed by a listing of its immediate child commands and subgroups, each with its one-line summary.
- The listing reflects the registered command set faithfully — every registered command at that level appears, none is invented, and the only descriptive text shown is the summary stored at registration.
- Commands within a listing are presented in alphabetical order by name, so repeated invocations yield identical output and a command can be located by name without reading the whole list.

### Per-command usage

- When `--help` is requested on a leaf command (`glassfrog <path> --help`), the system produces that command's usage in the command framework's standard form — its usage/invocation line, its one-line summary, its available flags, and (for a group) its child commands with summaries.
- When `--help` is requested on a group, the usage is the group listing described above — a group's "usage" is the set of subcommands it offers.

### Version

- When the version is requested via the `--version` flag (`glassfrog --version`) or the `version` command (`glassfrog version`), the system produces the CLI's version string.
- Both request forms produce identical output.

### Precedence

- When a single invocation requests both help and version (`glassfrog --help --version`), the system produces help — the help request takes precedence.

---

## User Scenarios

**In order to** discover what the CLI can do without prior knowledge of its commands,
**as an** AI agent operating the CLI,
**I want to** see a listing of every command with its summary.

**In order to** invoke a command correctly,
**as an** AI agent or practitioner,
**I want to** request `<command> --help` and read the command's path and purpose.

**In order to** confirm I am operating the expected build of the CLI,
**as an** operator,
**I want to** request the version and read a single version string.

---

## Non-Behaviors

- The system must not resolve or route invocations to commands. **Why**: routing is Argument Dispatch's (002) concern; Help & Version only renders text once dispatch has determined a help, listing, or version outcome. Duplicating routing here would create two sources of truth for what an invocation means.
- The system must not decide or emit process exit codes. **Why**: Exit-Code Convention (004) owns exit-code classification. This capability produces text only; coupling the numeric code here would pre-empt that capability.
- The system must not produce structured or machine-formatted output (JSON or similar). **Why**: a `--json`/structured output mode is a separate cross-cutting concern, out of scope for the skeleton; bundling it now would expand this capability before the read commands that need it exist.
- The system must not provide a standalone `help` command or subcommand (e.g. `glassfrog help roles`). **Why**: for now the `--help` flag plus dispatch-routed bare group/root listings cover the need; a `help` command is a possible future addition, not part of this slice.
- The system must not introduce new *required* per-command documentation data beyond the one-line summary registration already stores — no new mandatory long-description, example, or flag-help fields. **Why**: the summary is the only description data Command Registration stores (001); mandating richer documentation as new required data would exceed the skeleton's scope. The framework's standard help rendering may still display a command's existing description and its auto-generated flag list.
- The system must not emit build metadata (commit hash, build date) in version output. **Why**: a bare version string is the agreed scope now; enriched version output is deferred until there is demand.
- The system must not register, reorder, or otherwise mutate the command set. **Why**: it is a read-only consumer of Command Registration (001); the registered set is fixed at initialization and must stay the single source of truth.
- The system must not fabricate usage for a path that names no registered command. **Why**: an unrecognized path is dispatch's unknown-command error; rendering invented help would mask the error and mislead the caller.

---

## Integration Boundaries

- **Command Registration (001, upstream)**: Help & Version reads command names, group structure, and one-line summaries from the known command set. When the set is empty, only the root listing outcome is reachable, and it shows no commands.
- **Argument Dispatch (002, upstream)**: dispatch routes the root outcome, group outcome, and explicit `--help`/`--version` requests toward this capability, which produces the text. The unknown-command error and its pointer to help are dispatch's own; Help & Version renders nothing for an unrecognized path.
- **Exit-Code Convention (004, downstream)**: this capability produces the help/listing/version text; the process exit code that accompanies it is classified by Exit-Code Convention.
- **Caller (AI agent / practitioner)**: explicit help and version requests are produced as the command's successful output (standard output), in a stable layout the caller can read or parse line-by-line.

---

## Driving Scenarios

### Happy path

**Scenario: Root listing shows all top-level commands**
Given a command set with a `version` command and a `roles` group, each registered with a summary
When the caller invokes `glassfrog --help`
Then the system lists `version` and `roles`, each with its one-line summary.

**Scenario: Group help lists its subcommands**
Given a `roles` group containing `list` and `get`, each with a summary
When the caller invokes `glassfrog roles --help`
Then the system shows the `roles` path and summary
And lists `list` and `get`, each with its summary.

**Scenario: Leaf command usage**
Given a `roles get` command registered with the summary "Show one role"
When the caller invokes `glassfrog roles get --help`
Then the system shows the full path `roles get` and the summary "Show one role".

**Scenario: Version via flag and via command match**
Given the CLI is built with version `1.2.0`
When the caller invokes `glassfrog --version` and separately `glassfrog version`
Then both produce the same output containing `1.2.0`.

### Error scenarios

**Scenario: Help requested for an unregistered command**
Given no command named `bogus` is registered
When the caller invokes `glassfrog bogus --help`
Then the system produces no usage text for `bogus`
And the unknown-command outcome is left to Argument Dispatch.

**Scenario: Version flag on a non-root command is not a version request**
Given a `roles` group with no `--version` flag of its own
When the caller invokes `glassfrog roles --version`
Then the system does not produce version output
And the unrecognized flag is left to dispatch/parsing as a usage error.

### Edge cases

**Scenario: Root help on an empty command set**
Given no commands have been registered
When the caller invokes `glassfrog --help`
Then the system produces a coherent listing that names no commands rather than failing.

**Scenario: Group help lists only immediate children**
Given a `proposals` group whose child `changes` is itself a group containing `add`
When the caller invokes `glassfrog proposals --help`
Then the system lists `changes` with its summary
And does not recurse into `changes` to list `add`.

**Scenario: Both help and version requested**
Given the CLI is built with a version string
When the caller invokes `glassfrog --help --version`
Then the system produces help output, not version output.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Help introduces no new required registration data**
Given the command framework's standard help rendering
When help or usage is produced
Then every command-specific description shown comes from data the command already declares (its summary or an existing description), with no new mandatory registration field introduced.

**Scenario: Output is deterministic across runs**
Given an unchanged command set
When the same help or listing invocation is run twice
Then the two outputs are byte-for-byte identical.

**Scenario: No exit-code or routing logic leaks into rendering**
Given the produced spec and any later implementation
When the help/version surface is inspected
Then it neither selects a process exit code nor decides which command an invocation names.

---

## Assumptions

- **Version string is build-supplied**: the version value is stamped into the CLI at build time; how it is injected is an implementation detail this spec does not constrain. (Standard practice for a compiled CLI; the project stack is Go + cobra.)
- **Flag recognition is provided upstream**: identifying that an invocation carries `--help` or `--version` is handled by the dispatch/CLI framework; Help & Version is responsible for the rendering once the request is identified. (Follows the 001/002 boundary where dispatch routes and this capability renders.)
- **Version-unset fallback**: when no version was stamped into the build, version output shows a clear placeholder rather than an empty string. (Avoids silent empty output that an operator could misread.) `[ASSUMED]`

---

## Ambiguity Warnings

_None remaining — the listing-order question was resolved during clarification (see Clarifications)._

---

## Clarifications

### Session 2026-06-03

- **Listing order**: commands within a listing are ordered alphabetically by name. Chosen over registration order because it is the most predictable for the AI-agent operator (a command can be found by name without reading the whole list) and matches the conventional CLI framework default.
- **Help-text rendering**: usage/help uses the command framework's standard rendering (usage line, summary, available flags, child commands) rather than a minimal path+summary-only form. Non-behavior E was narrowed accordingly: it now forbids only *new required* documentation data, not the framework's display of existing descriptions and auto-generated flag lists. Chosen to keep Help & Version a thin layer over the framework's defaults instead of a custom presentation template. The bare-version-string, no-`--json`, no-standalone-`help`-command, and faithful-listing (no invented commands) boundaries are unchanged; cobra's built-in `help` and `completion` *commands* are still hidden (the `--help` flag remains).
