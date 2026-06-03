# Interface Accord: Command Registration — Specification

**Feature**: 001-command-registration
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture — "Registration guard" component and the internal extension surface; ADR-3 (fail-loud guard), ADR-4 (explicit wiring).

---

## Surface

The registration contract is what a Maintainer authors against to add a command. It has two parts: the **command definition** (the data each command carries) and the **registration entry points** (how a command is attached to the tree).

### Command definition

A command is one of two kinds — a **leaf** (runs an action) or a **group** (contains children). Both carry a name and a summary.

| Field | Type | Required | Description |
|---|---|---|---|
| name | string | yes | The single invocation token for this command (e.g. `roles`, `list`). Must be non-empty after trimming surrounding whitespace, and unique among its siblings. |
| summary | string | yes | One-line description, surfaced later by Help & Version. Must be non-empty after trimming. |
| action | function | leaf only | The behavior invoked when this command runs. Stored at registration; executed by Argument Dispatch, not by registration. |
| children | command[] | group only | One or more child commands. A child may itself be a leaf or a group, so groups nest to arbitrary depth. |

A command supplies **exactly one** of `action` (leaf) or a non-empty `children` set (group) — never both, never neither.

### Registration entry points

| Entry point | Signature (contract) | Description |
|---|---|---|
| Register | `Register(parent, child) -> error` | Validates `child` against all registration rules; on success attaches it under `parent` in the known command set; on any violation returns a descriptive error and attaches nothing. |
| MustRegister | `MustRegister(parent, child)` | Convenience wrapper over `Register` for the startup wiring path; aborts the program (panic) on violation instead of returning an error. |

`parent` is the root of the known command set or any already-registered group. Registration of the top-level commands uses the root as `parent`.

---

## Interactions

**Authoring a command**: a Maintainer adds a command in its own package (exposing a constructor that returns the command definition), then adds a single line at the wiring site (`main`) that registers it under its parent. No existing command's package is edited (ADR-4).

**Assembly order**: registration happens once at program initialization, before any invocation is dispatched. A group's children are registered into the group *before* the group is registered under its parent, so the "group must have ≥1 child" rule can be enforced at attach time. Nested groups are built by registering subgroups into parent groups, to arbitrary depth.

**Composition example** (contract-level, order shown):
```
rolesGroup := group("roles", "Read and inspect roles")
MustRegister(rolesGroup, leaf("list", "List roles", listAction))
MustRegister(rolesGroup, leaf("get",  "Show one role", getAction))
MustRegister(root, rolesGroup)        // root now exposes: roles list, roles get
MustRegister(root, leaf("version", "Print the version", versionAction))
```

**Querying**: the known command set resolves a path (`roles list`) to its command, resolves a bare group name (`roles`) to the group node itself, and enumerates every command/group with its name and summary. These reads are consumed by Argument Dispatch and Help & Version (see interface-cli.md).

---

## Error Communication

All violations are **constraint violations surfaced at startup**, before any user command runs. `Register` returns a descriptive error; `MustRegister` panics with the same message.

| Violation | Reported as |
|---|---|
| Name already taken among the parent's existing children | Error naming the conflicting name and the parent's path |
| Name empty or whitespace-only | Error identifying the offending command |
| Summary empty or whitespace-only | Error identifying the offending command |
| Leaf registered without an action | Error identifying the offending command |
| Group registered with no children | Error identifying the offending group |

The error message names the offending command so the Maintainer can locate it. Mapping a startup-abort to a concrete process **exit code is out of scope** — that contract belongs to the Exit-Code Convention capability. This accord guarantees only that a failed registration prevents the CLI from dispatching any command.

---

## Consistency Notes

- **Sibling**: the runtime surface these registrations produce is defined in `interface-cli.md`. The `action` stored here is executed by Argument Dispatch; the `summary` is rendered by Help & Version — this file defines only the authoring/registration contract, not dispatch or help behavior.
- **Stack mapping** (from plan ADR-2): realized over cobra — `name` → `Use` first token, `summary` → `Short`, `action` → `RunE`, `children` → attached subcommands. The guard wraps cobra's attach so cobra's permissive defaults (duplicate names, missing summary) cannot bypass the rules above.
- **First accord**: no `accords/` patterns exist yet — this file sets the project's first interface convention. Later command specs register through these same entry points.
