# Interface Accord: Help & Version — CLI

**Feature**: 003-help-and-version
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture (cobra-standard rendering on the assembled root); ADR-1 (standard help rendering), ADR-2 (hide built-ins, keep `--help`), ADR-3 (unified version).

---

## Surface

This accord defines what the caller observes when they ask the CLI to describe itself or report its version. The text is produced by the command framework's standard rendering over the registered command set (001); this accord fixes which invocations produce which kind of output, not the byte-exact layout.

| Invocation | Output |
|---|---|
| `glassfrog --help` *(or bare `glassfrog`, routed by dispatch)* | Root **listing**: the program's summary, then top-level commands and groups — each with its one-line summary — in **alphabetical order** by name. |
| `glassfrog <group> --help` *(or bare `glassfrog <group>`)* | Group **listing**: the group's usage line and summary, then its immediate child commands/subgroups with summaries, alphabetically. Immediate children only — not recursive. |
| `glassfrog <path> --help` *(leaf)* | Leaf **usage**: the command's usage/invocation line, its one-line summary, and its available flags (framework-standard form). |
| `glassfrog --version` | The CLI **version string** (e.g. `glassfrog version 1.2.0`). |
| `glassfrog version` | The **same** version string as `--version` — identical output. |

**Flags and commands this accord introduces:**
- `--help` (persistent flag, every command) — requests help/usage for the command it is attached to.
- `--version` (root flag) — requests the version string.
- `version` (registered leaf command) — requests the version string; output identical to `--version`.

**Built-ins:** the framework's auto-injected `help` and `completion` *commands* are **hidden** — they do not appear in any listing and `glassfrog help` / `glassfrog completion` do not resolve as built-in commands. The `--help` *flag* remains available everywhere.

---

## Interactions

- **Requesting help**: help is reached via the `--help` flag on any command/group/root, or by dispatch (002) routing a bare group or bare root to the listing outcome. There is no standalone `help` command.
- **Listing order**: commands within any listing are alphabetical by name — stable across runs, so a caller (typically an AI agent) can locate a command by name without reading the whole list.
- **Group vs leaf**: a leaf's `--help` shows its usage; a group's `--help` shows the listing of its immediate children (a group's "usage" is the set of subcommands it offers).
- **Version parity**: the `--version` flag and the `version` command read one underlying version value and emit the same line; build tooling sets that value at compile time.
- **Precedence**: when both `--help` and `--version` are supplied, help is produced (help takes precedence).

---

## Error Communication

- **Success surface**: help, listing, and version are normal output on **standard output** with a success outcome; they are not error conditions.
- **Help on an unregistered path** (`glassfrog bogus --help`): this accord renders **nothing** for `bogus` — an unrecognized command is Argument Dispatch's (002) unknown-command contract (message on stderr, pointer to help).
- **`--version` on a non-root command** (`glassfrog roles --version`): not a version request; the unrecognized flag is dispatch/parsing's usage error, not handled here.
- **Exit codes are out of scope**: this accord specifies what text is produced and on which stream, not the process exit code. Mapping any outcome to a code is **Exit-Code Convention**'s (004) contract.

---

## Consistency Notes

- **001 (`interface-cli.md`, `interface-spec.md`)**: 001 deferred "the flag grammar, the output format, and the format of the command listing" to Help & Version — this accord fills exactly that deferral. Listings draw names and summaries from the guard-registered set; the 001 guard guarantees every command has a non-empty summary, so every listed command has text to show.
- **002 (`interface-cli.md`)**: dispatch routes bare-group/root/unknown outcomes; 002 explicitly left "whether cobra's built-in `help`/`completion` are kept or hidden" to this spec — resolved here as **hidden**. Dispatch produces the unknown-command error; this accord renders the help that error points to.
- **004 (Exit-Code Convention, future)**: consumes outcomes and assigns process exit codes; this accord deliberately says nothing about codes.
- **Assumptions**: help/version output goes to **standard output** (stdout), conventional for explicitly-requested help/version (consistent with 002's stderr-for-errors assumption); the binary name is `glassfrog` (001 assumption; not yet fixed in PROJECT.md).
- **Deviation from framework default**: the framework's built-in `help`/`completion` commands are suppressed (ADR-2) rather than left at their defaults — flagged here because it diverges from out-of-the-box cobra behavior, in service of the spec's "no standalone `help` command" and faithful-listing non-behaviors.
