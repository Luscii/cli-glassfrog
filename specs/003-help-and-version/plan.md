# Plan: Help & Version

**Feature**: 003-help-and-version
**Role**: Shaper
**Inputs**: spec.md (003-help-and-version, post-clarify), PROJECT.md, `.score/memory/DECISIONS.md` (read at 6 entries — Go, cobra-is-registry, registration guard, explicit wiring, cobra resolution, outcome classification; this plan's post-step appends two more, bringing it to 8), `.score/memory/LEARNINGS.md` (cobra injects `help`/`completion` outside the guard — explicitly deferred to this spec). No SOUL.md; no DEPRECATION.md. Existing source: `internal/cli/{root,version,app,registry,roles}.go`, `main.go` (001 implemented; 002 planned but not yet implemented).

---

## System Architecture

Help & Version is not a new subsystem — it is **configuration of the cobra root** that `Assemble()` already builds (DECISIONS.md: "cobra's command tree *is* the registry; sibling capabilities become thin layers over cobra"). cobra natively renders a command listing, per-command usage, and version output; this feature tunes that rendering to match the spec and removes the framework artifacts the spec forbids. The three behavioral surfaces map directly onto cobra mechanisms:

- **Command listing** (root/group) and **per-command usage** — cobra's built-in help, triggered by the `--help` flag or by dispatch resolving to a bare group/root. cobra enumerates child commands with their `Short` summaries, sorted alphabetically (its default). The amended spec accepts cobra's **standard** help rendering (usage line, summary, flags section, child listing), so no custom template is written.
- **Version** — both `glassfrog --version` and `glassfrog version` produce one identical string. The existing `version` leaf command (`internal/cli/version.go`) already prints `version` (default `"0.0.0-dev"`, overridable via `-ldflags -X …cli.version=`). This feature wires cobra's root `Version` field + a version template so the `--version` flag prints the **same** string through the same source of truth.
- **Framework built-ins** — cobra auto-adds a `help` command and a `completion` command outside the registration guard (LEARNINGS.md). The spec forbids a standalone `help` command and requires the listing to show only registered commands, so both built-in **commands** are hidden; the `--help` **flag** is retained.

```
Assemble()                         (001: builds the guarded cobra tree)
  └─ configureHelpAndVersion(root) (003: this feature)
        ├─ root.Version = version            → enables --version flag
        ├─ root.SetVersionTemplate(…)        → --version prints the same string as `version` cmd
        ├─ root.SetHelpCommand({Hidden:true})       → `help` token no longer resolves (empty-Use placeholder, no resolvable token); --help flag kept
        └─ root.CompletionOptions.DisableDefaultCmd = true → removes `completion`
   cobra renders: listing / usage (alphabetical, EnableCommandSorting=true) / version
```

Control flow is unchanged from 001: `main` calls `Assemble()` and executes. This feature only adds a configuration pass applied to the root before execution. No command package is edited; no network or file I/O.

---

## Architecture Decisions

### ADR-1: Realize Help & Version as cobra's standard rendering, not a custom presentation layer

**Context**: The spec requires a command listing, per-command usage, and version output. DECISIONS.md establishes that sibling capabilities are thin layers over cobra. During planning the spec's original non-behavior E (no flags section, no long descriptions) would have forced a custom help template; the developer chose to **accept cobra's standard help rendering**, and the spec was amended (clarify, 2026-06-03) to narrow E to "no new *required* documentation data."

**Options considered**:
1. **Use cobra's default help/usage rendering** — zero custom presentation code; the listing, usage line, flags section, and alphabetical sorting come from the framework. Output matches what `kubectl`/`gh` users expect. Downside: output shape is cobra's, not bespoke.
2. **Custom minimal help template** — `SetHelpTemplate`/`SetUsageTemplate` emitting only path + summary + child listing. Honors the original strict E, but re-implements presentation and diverges from the thin-layer precedent for marginal benefit.

**Decision**: Option 1 — cobra's standard rendering. It conforms to the cobra-thin-layer precedent (DECISIONS.md) and the amended spec. Help & Version's code is configuration, not a renderer.

In practice: nothing is rendered by hand. `--help` (and dispatch's bare-group/root routing) invoke cobra help; commands need only their `Short` summary (guaranteed non-empty by the 001 guard) to appear correctly. Alphabetical order is cobra's default (`cobra.EnableCommandSorting == true`) — kept, not configured.

**Consequences**: Minimal new code; help stays consistent with 001/002's framework choice. Output shape is coupled to cobra's templates (acceptable per ADR-2 of 001). Alphabetical ordering depends on a cobra **package-global** (`EnableCommandSorting`) staying `true` — a regression test must pin it (mirrors how 002 pinned `EnablePrefixMatching == false`). The root's existing `Long` text will now surface in `--help` output — acceptable under the amended spec. *Precedent-setting: later command specs get standard help for free by setting `Short`.*

### ADR-2: Hide cobra's built-in `help` and `completion` commands; keep the `--help` flag

**Context**: cobra injects a `help` command and a `completion` command that bypass the registration guard and appear in the listing (LEARNINGS.md, deferred here). The spec's non-behaviors forbid a standalone `help` command and require the listing to show only registered commands ("none is invented"). `completion` is outside the skeleton's three surfaces and the deferred-scope (PROJECT.md defers AI-agent/operational extensions, and shell completion is not in scope).

**Options considered**:
1. **Hide both built-in commands, keep the `--help` flag** — replace cobra's default help command with a hidden **empty-`Use`** placeholder (`SetHelpCommand(&cobra.Command{Hidden: true})`) so it stays out of the listing *and* introduces no resolvable command token; `Hidden:true` **alone** only hides a command from listings — it does not stop `glassfrog help` from resolving, and a *named* placeholder (e.g. `Use: "__help_disabled"`) would itself become a resolvable invented path, which is why the empty-`Use` form is required. `CompletionOptions.DisableDefaultCmd = true` removes `completion`. The `--help` flag is unaffected.
2. **Keep cobra's built-ins** — least code, but a `help` command violates the no-standalone-`help` non-behavior and both appear in the listing as un-guarded "invented" commands.
3. **Hide `completion` only, keep `help`** — still violates the no-standalone-`help` non-behavior.

**Decision**: Option 1 — hide both commands, retain the `--help` flag. This satisfies "no standalone `help` command" and "listing shows only registered commands" without losing the flag-driven help the spec is built around.

**Consequences**: The listing is faithful to the guarded set. Shell completion is not offered — a deliberate skeleton-scope exclusion; if completion is wanted later, re-enabling is a one-line reversal. Hidden-command configuration lives at the root, applied once in the configuration pass. A test should assert that `glassfrog help` and `glassfrog completion` do **not** resolve (unknown command, not merely hidden), that neither appears in the listing, and that the `--help` flag still renders. *Precedent-setting: the CLI treats framework built-ins as hidden by default; future specs that want a built-in surfaced must opt in explicitly.*

### ADR-3: Unify `--version` and the `version` command on a single version string

**Context**: The spec requires `glassfrog --version` and `glassfrog version` to produce **identical** output. `version.go` already owns the `version` variable and the `version` command; cobra's `--version` flag is enabled by setting `rootCmd.Version`.

**Options considered**:
1. **Single source of truth** — `rootCmd.Version = version` plus a version template that prints the same string the `version` command prints (the **bare** value — the template must override cobra's default `Name version X` form), both reading the one package-level `version` var. One value, two entry points, guaranteed identical.
2. **Independent renderers** — let the flag and the command format separately. Risks drift between the two outputs, directly threatening the "identical output" requirement.

**Decision**: Option 1 — both entry points read the same `version` var and emit the same line. The `version` command stays guard-registered (001 precedent); the flag is configured on the root.

**Consequences**: Identical output is structural, not coincidental. The build-time `-ldflags -X` override flows to both. The `"0.0.0-dev"` default already satisfies the spec's `[ASSUMED]` version-unset fallback (a clear placeholder, never empty). A test asserts the flag output and command output match — and specifically that neither carries cobra's default `glassfrog version …` prefix. *Feature-local — not recorded as cross-spec precedent.*

---

## Cross-cutting Concerns

**Error handling**: Help and version are success outputs; they go to standard output and do not error in normal use. Requesting help on an unregistered path (`glassfrog bogus --help`) is an unknown-command outcome owned by Argument Dispatch (002) — this feature renders nothing for it. The numeric exit code that accompanies any of this is Exit-Code Convention's (004) concern; this feature only produces text.

**Precedence**: When both `--help` and `--version` are present, help wins (spec). This is cobra's default behavior (the help flag short-circuits before version) — relied on, not implemented, and pinned by a test.

**Configuration**: Nothing is runtime-configurable. The `version` string is set at build time via `-ldflags`. Help rendering, sorting, and built-in hiding are fixed at initialization.

**Testing strategy** (CONSTITUTION IV, test-first): unit/acceptance tests exercise the assembled root: (a) root and group `--help` list expected commands with summaries, alphabetically; (b) `--version` flag and `version` command emit identical output; (c) neither `help` nor `completion` command appears in the listing, while `--help` still renders; (d) regression test pins `cobra.EnableCommandSorting == true`; (e) `--help --version` yields help. The spec's driving scenarios become the godog BDD outer loop (following 001/002's pattern).

---

## Implementation Strategy

Two phases, linear; each PR-sized.

- **Phase 1 — Root configuration pass**: add a `configureHelpAndVersion(root)` step (called from `Assemble()` after wiring) that sets `root.Version`, the version template (sharing `version.go`'s string), hides the `help` command, and disables `completion`. RED-first unit tests for: `--version`/`version` parity, built-ins absent from listing, `--help` still renders, sorting global pinned. *Depends on: nothing beyond 001 (already implemented).*
- **Phase 2 — Executable acceptance**: godog step definitions for the 003 driving scenarios (root listing, group listing, leaf usage, version-via-flag-and-command, help-on-unregistered-path renders nothing, empty-set root help, group lists only immediate children, `--help --version` precedence), turning them into executable acceptance. *Depends on: Phase 1.*

---

## Risks

- **Alphabetical ordering depends on a cobra package-global** (low likelihood, medium impact): `cobra.EnableCommandSorting` is process-global and defaults `true`; if any future code sets it `false`, the alphabetical-listing behavior breaks silently. Mitigation: a regression test asserting sorted output (directly parallels 002's `EnablePrefixMatching` pin).
- **Hiding the `help` command via a hidden placeholder** (low likelihood, low impact): cobra has no first-class "no help command" switch; the idiom is `SetHelpCommand` with a `Hidden` command. A cobra upgrade could change this. Mitigation: a test asserting `glassfrog help` does not resolve as a built-in command and `--help` still works.
- **Version output drift** (low likelihood, medium impact): if a future change formats the flag template differently from the `version` command, the "identical output" requirement breaks. Mitigation: both read one `version` var; a test asserts byte-equality of the two outputs.

---

## What This Plan Does Not Cover

- **Exit codes / stream-to-code mapping** — this feature produces text; **Exit-Code Convention (004)** maps outcomes to process codes.
- **Routing of bare-group/root/unknown invocations** — **Argument Dispatch (002)** resolves and routes; this feature only renders once cobra reaches a help/version outcome.
- **Protocol-level CLI surface** (exact flag spellings, the usage grammar, version-string format contract) — the **interface** skill's concern.
- **Executable scenarios** and **task decomposition** — the **scenarios** and **tasks** skills.
- **Structured/`--json` output, build metadata in version, a standalone `help` command, shell completion** — all out of scope per the spec's non-behaviors; noted so downstream skills don't re-open them.
