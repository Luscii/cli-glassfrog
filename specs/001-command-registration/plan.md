# Plan: Command Registration

**Feature**: 001-command-registration
**Role**: Shaper
**Inputs**: spec.md (001-command-registration), PROJECT.md, CONSTITUTION.md (Principle XII referenced), BACKLOG.md / FEATURE-MODEL.md (skeleton context). No SOUL.md; no `.score/memory/` existed at plan time (this plan's post-step creates `DECISIONS.md`) — first planning run.

---

## System Architecture

Command Registration is the foundation of the Glassfrog CLI — the very first code in a repository that currently holds only discovery documents and the vendored API spec. The CLI is built as a **self-contained Go binary** (CONSTITUTION Principle XII: runs on a clean host with only OS + network, no pre-installed runtime). It uses the **cobra** command framework, whose command tree *is* the command registry the spec describes.

The parts:

- **Command definition** — the unit a Maintainer authors: a name, a one-line summary, and either an action (leaf) or one-or-more children (group). Realized as a `*cobra.Command` (`Use` = name, `Short` = summary, `RunE` = action) assembled inside each command's own package, so adding a command touches only that package.
- **Registration guard** — a thin wrapper (`Register(parent, child)` / `MustRegister`) that validates a command against the spec's fail-loud rules and then attaches it to its parent in the cobra tree. This is the single source of truth for *registration validity*; it is the spec's behavioral owner.
- **Root command (the registry tree)** — cobra's root command is the top of the known command set; the tree of commands beneath it is "the known command set." Path lookup, enumeration, and a bare-group resolving to itself are cobra tree behaviors the guard does not re-implement.

**Control flow**: at startup, `main` wires the tree by calling the guard for each command (explicit assembly — see ADR-4). Every guard call validates before attaching. If any registration violates a rule, the guard returns an error and startup aborts *before* cobra parses or dispatches any invocation — satisfying the spec's "fails before any user command runs." Once the tree is assembled, cobra owns parsing, dispatch, and help rendering — the concerns the sibling capabilities (Argument Dispatch, Help & Version) build on. Registration performs no network or file I/O.

```
main (explicit wiring)
  └─ Register(root, rolesGroup)        ← guard: validate → AddCommand
       └─ Register(rolesGroup, listCmd)
       └─ Register(rolesGroup, getCmd)
  └─ Register(root, versionCmd)
        ▲
        │ on any violation: return error, abort startup (no dispatch)
   cobra tree == known command set  → dispatch & help read it
```

---

## Architecture Decisions

### ADR-1: Build the CLI in Go as a self-contained executable

**Context**: CONSTITUTION Principle XII requires a self-contained executable that assumes no pre-installed runtime or interpreter — only host OS plus network. PROJECT.md names no stack; this is the project's foundational language decision, inherited by every later spec. The CLI is I/O-bound: an HTTP surface over the Glassfrog v5 API with JSON shaping and dual (machine/human) output.

**Options considered**:
1. **Go** — compiles to a single self-contained executable, trivial cross-compilation to every OS/arch (with `CGO_ENABLED=0` to avoid cgo where supported), the most mature nested-subcommand CLI ecosystem (cobra). Gentle curve; minor downside is it's a new language for the team.
2. **Rust** — also a single self-contained executable, excellent CLI library (clap). Strong safety guarantees, but the borrow-checker tax and slower iteration buy little for an I/O-bound API wrapper.
3. **Compiled TypeScript (Bun/Deno)** — keeps the team in TS; satisfies XII only by bundling a ~50–100MB runtime into the artifact, which is the opposite spirit of "just the binary."

**Decision**: Option 1 — Go. It is the best fit for a standalone, network-bound API-client CLI: native single-binary output is exactly what Principle XII wants, and cobra is the de-facto standard for the nested-command shape this project needs (`kubectl`, `gh`, `docker`).

In practice: `go build` produces `glassfrog`; binaries are cross-compiled per target via `GOOS`/`GOARCH`, with `CGO_ENABLED=0` avoiding cgo for portable cross-compilation where supported. The artifact is self-contained — it assumes no pre-installed runtime, satisfying XII; "fully static" linking is a platform-specific outcome (e.g. Linux/musl), not a universal guarantee, so it is not relied on as the criterion.

**Consequences**: Native compliance with Principle XII (the clean-environment detection test passes with a small binary). The team adopts Go (interfaces, goroutines, errors-as-values) — a real but gentle learning step. Rust's deeper guarantees and TS code-sharing are both forgone; revisiting the language later would mean a full rewrite. *Precedent-setting for the whole project.*

### ADR-2: Adopt cobra as the command framework; its command tree is the registry

**Context**: The spec requires nested command groups to arbitrary depth, lookup by path, enumeration, and a bare group resolving to itself. Argument Dispatch and Help & Version (sibling capabilities) build directly on the registered set.

**Options considered**:
1. **Adopt cobra** — a mature framework that natively models nested commands, path resolution, help, and (later) completions. Dispatch and help come essentially for free as thin layers. Downside: a dependency, and its defaults don't match the spec's fail-loud rules.
2. **Hand-roll the registry** — full control over the registration model and semantics, zero CLI dependency, but Argument Dispatch and Help & Version must be built from scratch, re-inventing well-trodden wheels.
3. **urfave/cli** — another established Go CLI library; capable, but a smaller ecosystem and less ubiquitous than cobra for deeply-nested command trees.

**Decision**: Option 1 — cobra. Its command tree directly realizes the spec's "known command set," and the sibling capabilities become thin layers rather than separate subsystems, accelerating the rest of the skeleton.

**Consequences**: Large reduction in code for dispatch and help. The skeleton is coupled to cobra; if it later proves limiting, migration has cost. cobra does **not** enforce the spec's fail-loud rules (it tolerates duplicate names and missing summaries), which is precisely why ADR-3 exists. *Precedent-setting: later command specs register into the cobra tree via the guard.*

### ADR-3: Enforce fail-loud registration through a guard wrapper, not cobra defaults

**Context**: The spec mandates that duplicate sibling names, empty/whitespace names, empty/whitespace summaries, leaves without an action, and groups without children all fail at registration time, before any user command runs. cobra's `AddCommand` enforces none of these.

**Options considered**:
1. **Rely on cobra defaults** — least code, but silently violates the spec (collisions shadow, missing summaries ship). Rejected.
2. **Guard wrapper** — a small `Register`/`MustRegister` function that validates each command against all five rules and only then calls `AddCommand`; on violation it returns/panics with an error naming the offending command, aborting startup.
3. **Fork or patch cobra** — maximal control, unjustifiable maintenance burden for five validation rules.

**Decision**: Option 2 — a guard wrapper. It keeps the spec's behavioral rules in our own legible, unit-testable code (aligning with CONSTITUTION Principle I — legible, traceable actions) while leaving tree structure, lookup, and help to cobra.

In practice: `Register(parent, child)` checks trimmed-non-empty name, trimmed-non-empty summary, leaf-has-action, group-has-≥1-child, and no name collision among `parent`'s existing children; then attaches. A `MustRegister` variant converts a violation into a startup panic for the wiring path. Errors are surfaced before `Execute()` is ever called.

**Consequences**: The fail-loud contract is owned, tested, and independent of cobra's evolving defaults. Every command must register *through* the guard — calling `AddCommand` directly is the bypass to watch for in review. The guard is a pure function over (parent, child), trivially unit-testable RED-first.

### ADR-4: Wire commands explicitly in `main`, not via package `init()` side effects

**Context**: Go packages can self-register via `init()` import side effects (a common CLI pattern), or commands can be assembled explicitly by a wiring function. The spec requires deterministic, startup-time registration; CONSTITUTION Principle I values legible, traceable actions.

**Options considered**:
1. **Package `init()` auto-registration** — commands register themselves on import; adding a command is just an import. Convenient, but ordering is implicit and the command set becomes hard to trace (registration is scattered and order-dependent on import graph).
2. **Explicit wiring in `main`** — a single assembly point calls the guard for each command. Adding a command adds one wiring line plus its package; the full command set is readable in one place.

**Decision**: Option 2 — explicit wiring. Deterministic ordering, a single legible source of truth for what the CLI offers, and clean failure surfacing (the guard's error aborts the assembly function before dispatch).

**Consequences**: Adding a command requires one line at the wiring site in addition to the command's own package — a deliberate, traceable touch rather than an invisible import side effect. The "without touching unrelated commands" property holds: the wiring site grows by one line; no existing command is edited. *Precedent-setting for how all later commands attach.*

---

## Cross-cutting Concerns

**Error handling**: Registration is fail-loud. The guard returns an error (or `MustRegister` panics) naming the offending command and the rule violated; the wiring function propagates it so startup aborts before `cobra.Execute()`. The mapping of that abort to a concrete process exit code is **Exit-Code Convention's** concern, not this feature's — registration's only obligation is that no user command runs after a failed registration.

**Testing strategy** (CONSTITUTION Principle: test-first): the guard is a pure function over (parent, child) → error, so each fail-loud rule and the happy path get a RED-first unit test with no cobra execution needed. The assembled tree is assertable by path lookup and enumeration, which lets the spec's driving scenarios become executable acceptance scenarios over the registry.

**Configuration**: Nothing is configurable here — the command set is fixed at build/initialization time (per the spec's no-runtime-mutation, no-plugin-discovery non-behaviors).

**Observability**: Out of scope for an in-process registry; registration does no logging beyond the guard's error text.

---

## Implementation Strategy

Three phases with a strict dependency chain; each is PR-sized.

- **Phase 1 — Go module + root command skeleton**: `go mod init`, add the cobra dependency, create the root command and `main` entrypoint, confirm `go build` emits a runnable `glassfrog` binary that prints root help. This is the minimal bootstrap needed to *host* registration — not the full CLI. *Depends on: nothing.*
- **Phase 2 — Registration guard**: implement `Register`/`MustRegister` with all five fail-loud rules and RED-first unit tests; this is the spec's core. *Depends on: Phase 1 (needs the cobra types and module).*
- **Phase 3 — Exercise nested registration**: wire a `version` leaf and a sample nested group (e.g. `roles list` / `roles get`) through the guard to prove arbitrary-depth registration, bare-group resolution, and collision rejection end-to-end, turning the driving scenarios into executable acceptance. *Depends on: Phase 2.*

---

## Risks

- **cobra semantics diverge from the spec at the edges** (medium likelihood, medium impact): cobra's path resolution returns the deepest-matched command plus remaining args rather than a clean "not found," and a bare group shows help by virtue of having no `RunE`. The guard and Phase-3 acceptance tests must confirm bare-group-resolves-to-itself behaves as specified; the precise "unknown path" semantics belong to Argument Dispatch and are noted, not resolved, here.
- **Guard bypass via direct `AddCommand`** (medium likelihood, medium impact): nothing in cobra prevents a future contributor from calling `AddCommand` directly and skipping the fail-loud rules. Mitigation: make `Register` the only documented path, cover it in review, and consider a lint/test that asserts no direct `AddCommand` calls outside the guard.
- **cobra coupling** (low likelihood, medium impact): the whole skeleton leans on cobra (ADR-2). If it proves limiting, dispatch/help would need re-homing. Accepted as a deliberate trade for velocity.

---

## What This Plan Does Not Cover

- **Protocol-level CLI surface** — exact flag names, the `glassfrog` invocation grammar, help-text formatting, and the `Register` function signature/contract are the **interface** skill's concern (this feature has a CLI boundary and an internal extension surface).
- **Executable scenarios** — turning the spec's driving scenarios into `.feature` files is the **scenarios** skill's job.
- **Task decomposition** — breaking the three phases into PR units with acceptance criteria is the **tasks** skill's job.
- **Argument parsing/dispatch, help rendering, exit codes** — deliberately deferred to the sibling skeleton capabilities; this plan only ensures the registry they consume exists and is guarded.
