# Learnings

Operational patterns discovered during implementation. Each entry records a non-trivial finding that future specs should account for. Managed by the implement skill via memory-protocol.md.

---

## Findings

### 2026-06-03 — cobra injects `completion` and `help` commands outside the registration guard
(Found during [T004](../../specs/001-command-registration/tasks.md) implementation)

- **Type**: design-issue
- **Location**: `internal/cli` (cobra root command), surfaces in `glassfrog --help` / command listing
- **Severity**: low — expected cobra behavior; no impact on Command Registration's guarantees, but it affects sibling capabilities.
- **Description**: cobra automatically adds built-in `completion` and `help` subcommands to the root. These are framework-provided and do **not** pass through our `Register`/`MustRegister` guard, so they are not subject to the fail-loud rules and they appear in the command listing alongside guard-registered commands.
- **Suggested action**: The **Help & Version** spec (and **Argument Dispatch**) should decide explicitly how to treat cobra's built-ins — list them, hide them, or replace cobra's default help — rather than assuming every visible command came through the guard. If the built-ins are unwanted, disable via `cmd.CompletionOptions.DisableDefaultCmd` and/or `SetHelpCommand`.
