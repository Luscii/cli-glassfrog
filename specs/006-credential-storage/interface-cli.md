# Interface Accord: Credential Storage — CLI

**Feature**: 006-credential-storage
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-2 (the guard-registered leaf delegating to the `internal/auth` writer), ADR-3 (token-source precedence + interactivity), ADR-5 (code-free outcome mapped by Exit-Code Convention).

---

This accord pins the operator-facing command that stores a token into the `.glassfrogrc` file Discovery (005) reads. The file's structural contract is pinned separately in `interface-spec.md`; this file pins the **invocation surface** — the command, its flags, input precedence, interactivity, output, and exit codes.

---

## Surface

### `glassfrog auth` (group)

A new command group holding credential operations. Non-runnable group with a required one-line summary (registration guard: group-has-children, non-empty `Short`). This slice contributes one subcommand (`login`); removal/logout is out of scope.

### `glassfrog auth login [TOKEN]`

Store an API token to the credentials file so later commands authenticate without re-supplying it.

**Synopsis**: `glassfrog auth login [TOKEN] [--cwd] [--overwrite]`

| Argument | Type | Required | Description |
|---|---|---|---|
| `TOKEN` | string | no | The token to store. When omitted, it is taken from piped stdin, then `GLASSFROG_TOKEN`, then an interactive prompt. At most one positional (`MaximumNArgs(1)`). |

| Flag | Type | Default | Description |
|---|---|---|---|
| `--cwd` | bool | false | Write `./.glassfrogrc` in the current working directory instead of the home-directory file. |
| `--overwrite` | bool | false | Replace an existing stored token without prompting. Required to change an existing token in a non-interactive session. |

**Output** (success, stdout) — names the written path, never the token:
```
Stored credentials in <path>
```
where `<path>` is the resolved credentials file (the home-directory `.glassfrogrc`, or `./.glassfrogrc` under `--cwd`).

## Interactions

**Token source precedence** (first present wins): positional `TOKEN` → piped stdin → `GLASSFROG_TOKEN` → interactive prompt. A value that is empty or whitespace-only is **not** a usable token (same rule as 005) and is rejected.

**Interactivity** is determined by whether standard input is a terminal (TTY):
- **stdin is a pipe** (e.g. `echo "$TOKEN" | glassfrog auth login`): the piped value is read as the token (no prompt).
- **stdin is a TTY and no other source supplied a token**: a non-echoing prompt requests the token.
- **stdin is not a TTY and no source supplied a token**: the command reports an error (`no token to store`) — it never hangs on a prompt and never exits success.

**Target location**: the home-directory `.glassfrogrc` by default; `--cwd` writes the current-directory file instead. (These are the two locations Discovery searches.)

**Existing stored token**:
- **Interactive (TTY)**: the command confirms before changing an existing token, offering to merge (replace the `token` entry, preserve other keys) and to choose the target among the current-directory file, the home file, or both.
- **Non-interactive**: the command reports an error and makes no change *unless* `--overwrite` is given; with `--overwrite` it merges into the single target location only (home, or `./.glassfrogrc` under `--cwd`). The current-directory/home/both choice is offered only in the interactive confirmation.

**Scripting**: designed for non-interactive use — `printf '%s' "$TOKEN" | glassfrog auth login --cwd` stores a project-local token with no terminal. The token never appears in output, so command transcripts and CI logs do not leak it.

**Configuration precedence** (mirrors the source precedence above): explicit `TOKEN` argument overrides piped stdin, which overrides `GLASSFROG_TOKEN`, which overrides the prompt.

## Error Communication

Errors are written to **stderr**; the process exit code is the category mapped by Exit-Code Convention (004) — this command emits a code-free outcome category, not a code.

The **Outcome category** column uses the canonical `Outcome` enum names published by 004 / `internal/cli` (`Success`, `UsageError`, `RuntimeError`) so the mapping to Exit-Code Convention is unambiguous. Every error message names both the cause **and** a concrete next step (CONSTITUTION II), and never includes the token value.

| Condition | Outcome category | Exit code (via 004) | stderr message (cause + next step; token never included) |
|---|---|---|---|
| Token stored | `Success` | 0 | — (success line on stdout) |
| No token supplied, non-interactive | `UsageError` | 2 | `no token to store` — supply a token via argument, stdin, or `GLASSFROG_TOKEN` |
| Blank / whitespace-only token | `UsageError` | 2 | names the empty input (not a value) — supply a non-empty token |
| Existing token, non-interactive, no `--overwrite` | `UsageError` | 2 | reports that a credential already exists at `<path>` (the path only, never the stored value) — pass `--overwrite` to replace it |
| Target not writable (permission denied) | `RuntimeError` | 1 | write error naming `<path>` — check write permission on the directory; filesystem unchanged |
| Existing file unparseable (merge) | `RuntimeError` | 1 | format error naming `<path>` — fix or remove the malformed `.glassfrogrc`; file not overwritten |

Exit codes follow the frozen convention pinned by 004 (`0` Success, `1` RuntimeError/internal, `2` UsageError). No new codes are introduced. The token value never appears in any message, prompt echo, or diagnostic.

## Consistency Notes

- **File format** is pinned in the sibling `interface-spec.md` (write side) and must round-trip with 005's `interface-spec.md` (read side) — same `.glassfrogrc`, `key=value`, `token` key. `[ASSUMED]` name, jointly held with 005.
- **`GLASSFROG_TOKEN`** is the same constant Discovery reads (005); reused, not re-declared. Note the dual role: for Discovery it is a runtime override; here it is a persistable source — persisting it is an explicit operator action.
- **Command conventions** follow 001/003 precedent: the `auth` group and `login` leaf register through the fail-loud guard, are wired explicitly in `main`, the leaf declares `Args: cobra.MaximumNArgs(1)`, and both carry a non-empty `Short` so standard cobra help renders for free. No package-global cobra toggles change.
- **Exit-code mapping** is owned by 004 (`interface`/registry); this command only classifies its outcome category, exactly as Argument Dispatch (002) does.
- **No `--token` flag** this slice (spec non-behavior): the token is a positional or comes from stdin/env/prompt. A flag is a possible later addition.
