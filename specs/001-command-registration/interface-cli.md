# Interface Accord: Command Registration — CLI

**Feature**: 001-command-registration
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture — the `glassfrog` binary's command tree (cobra root); the CLI boundary noted in "What This Plan Does Not Cover".

---

## Surface

The binary is invoked as `glassfrog`. Command Registration does not define any *specific* end-user commands — those arrive with later specs — it defines the **shape of the invocation surface** that registered commands produce:

- A **registered leaf** is reachable at `glassfrog <path…>` (the sequence of names from the root to the leaf, e.g. `glassfrog roles list`); invoking it runs that leaf's action.
- A **registered group** is reachable at `glassfrog <path…>` ending on the group (e.g. `glassfrog roles`); resolving to a group does not run an action — it resolves to the group node so its child commands can be listed.
- Paths nest to arbitrary depth (`glassfrog proposals changes add`), mirroring the registered tree.

> Scope: the **flag grammar**, the **output format**, and the **format of the command listing** are owned by Help & Version; **process exit codes** are owned by Exit-Code Convention. They are out of scope for this accord (plan, "What This Plan Does Not Cover").

---

## Interactions

| Invocation | Resolves to |
|---|---|
| `glassfrog <group>` (no trailing leaf) | The group node — Help & Version lists its child commands with their summaries. |
| `glassfrog <group> <subcommand>` | The leaf command — its action runs. |
| `glassfrog <group> <subgroup> <leaf>` | The nested leaf — arbitrary depth resolves the same way. |

The guarantee this accord makes is narrow: every path that was **registered** (interface-spec.md) resolves to its command or group, and a bare group self-resolves rather than erroring. How a typed invocation is matched to a path, and what happens for an *unregistered* path, are Argument Dispatch's contract.

---

## Error Communication

- **Unregistered path**: not this accord's concern — Argument Dispatch owns the "unknown command" contract.
- **Registration failure at startup**: if any command fails registration (see interface-spec.md), the binary aborts before dispatching anything — there is no partial command tree exposed to the user. The concrete exit code is Exit-Code Convention's contract.

---

## Consistency Notes

- **Sibling**: this runtime surface is produced entirely by the registrations defined in `interface-spec.md`; the two files describe the same tree from the consumer side (here) and the author side (there).
- **Deferred surfaces**: output format, command-listing format, and exit codes are deliberately deferred to Help & Version and Exit-Code Convention per the plan — this accord intentionally says nothing about them to avoid pre-empting those capabilities' contracts.
- **Assumption**: the binary name is `glassfrog` (spec assumption; not yet fixed in PROJECT.md).
