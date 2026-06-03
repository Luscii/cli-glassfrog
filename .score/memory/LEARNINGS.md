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

### 2026-06-03 — cobra rejects unknown subcommands only at the root, not under nested groups
(Found during [T002](../../specs/002-argument-dispatch/tasks.md) implementation)

- **Type**: design-issue
- **Location**: `internal/cli/dispatch.go` (`Run`), surfaces for any `glassfrog <group> <typo>` invocation.
- **Severity**: medium — without handling, an unknown subcommand under a group fails open (prints the group's help, returns no error), violating the dispatch contract that every unmatched token is a `UsageError`.
- **Description**: cobra's default arg validator (`legacyArgs`) only emits an `unknown command` error when the command has **no parent** (the root). A non-runnable nested group (e.g. `roles`) given an unknown subcommand (`lst`) returns `flag.ErrHelp` *before* arg validation runs, so cobra prints the group's help and `Execute` returns `nil`. Setting `Args: cobra.NoArgs` on the group does **not** fix this — `Find` skips `legacyArgs` once `Args` is non-nil, and the `!Runnable() → flag.ErrHelp` early return still pre-empts `ValidateArgs`. Our registration guard also forbids giving a group a `RunE` (a command is leaf-with-action XOR group-with-children), so the group cannot be made runnable to intercept the args. Dispatch therefore detects the swallowed token itself: after `ExecuteC`, when the executed node is a non-runnable group and `executed.Flags().Args()` (cobra's own flag-stripped positional remainder) is non-empty, `Run` synthesizes `unknown command "<token>" for "<path>"` and classifies `UsageError`. **Because cobra returned `nil` here it printed no error of its own, so `Run` must also write the synthesized error (plus a `Run '<path> --help' for usage.` pointer) to `executed.ErrOrStderr()` — otherwise the operator sees only the group help and never which token was unrecognized (PR #4 review, round 1).** This stderr write belongs in `Run`, not the entrypoint: blanket-printing returned errors in `main.go` would double-print the errors cobra already emits itself (root unknown command, unknown flag).
- **Suggested action**: **Help & Version (003)** owns help-text rendering and the keep/hide of built-ins; if it customizes the help command or the not-runnable behavior, re-verify this swallow-detection still fires. **Exit-Code Convention (004)** maps the `UsageError` category — note that the nested-group unknown-subcommand path produces `UsageError` via a *synthesized* error (not a cobra-native one), so any error-type inspection in 004 must not assume a cobra sentinel.

### 2026-06-03 — cobra leaf commands accept arbitrary positional args unless they declare an `Args` validator
(Found during PR #4 review of [002-argument-dispatch](../../specs/002-argument-dispatch/tasks.md))

- **Type**: design-issue
- **Location**: leaf command constructors (`version.go`, `roles.go`); classification in `internal/cli/dispatch.go` (`Run`).
- **Severity**: medium — without an `Args` validator a leaf silently ignores extra positional tokens (`glassfrog version extra` runs the action and drops `extra`), violating the Invalid-input accord ("unexpected input is never silently ignored; the command does not run").
- **Description**: a cobra command with no `Args` field defaults to `ArbitraryArgs`, accepting any positional tokens and passing them to the action. Each leaf must declare what it accepts: leaves with no positional arguments set `Args: cobra.NoArgs`; a leaf taking a fixed argument (e.g. a future `roles get <role-id>`) sets `cobra.ExactArgs(n)`. The acceptance contract is per-command, so dispatch can't enforce it generically — but it must *classify* the rejection. cobra runs `ValidateArgs` before the action, so when `Run`'s runnable-command arm sees a non-nil `err`, it re-checks `executed.ValidateArgs(executed.Flags().Args())`: non-nil means the command refused an unexpected arg (`UsageError`, action never ran); nil means the action ran and the error is its own runtime failure (`Success`, with the error returned — `RuntimeError` deferred to 004).
- **Suggested action**: every new leaf must declare an `Args` validator matching its positional contract. Consider whether the **Command Registration** guard should *require* an explicit `Args` validator on leaves (fail-loud) so this can't be forgotten — today it is a convention, not an enforced rule.
