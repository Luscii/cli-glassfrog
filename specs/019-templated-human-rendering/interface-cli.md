# Interface Accord: Templated Human Rendering — CLI

**Feature**: 019-templated-human-rendering
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-1/3/4 — the rendered human output the four read commands (`me`, `me roles`, `me actions`, `me projects`) now produce through the `internal/render` seam: the standing `full` format (field-equivalent to each read's pre-019 projection) and the defined-but-not-yet-selectable `compact` format.

---

This accord pins the **rendered stdout surface** of the four read commands under 019. 019 **adds no command and no flag** — the commands, their args, and their flags (`--include` on `me`; `--status` on `me actions`/`me projects`) are unchanged (011–014); it changes only *how their successful result is rendered* (through `internal/render`, see `interface-spec.md`). The `--output` flag that will let an operator pick a format is **020's** surface. Until 020, every read renders **`full`**; **`compact`** is fully defined and tested here but reachable from no operator surface yet (Q1 resolution). JSON/YAML output is **018's** surface, not this one.

---

## Surface

No new commands or flags. The affected commands and their unchanged invocation:

| Command | Synopsis | 019 effect |
|---|---|---|
| `glassfrog me` | `me [--include roles] [--base-url …]` | success body rendered via `render.Render(ResourceMe, FormatFull, …)` |
| `glassfrog me roles` | `me roles [--base-url …]` | success body rendered via `render.Render(ResourceRoles, FormatFull, …)` |
| `glassfrog me actions` | `me actions [--status …] [--base-url …]` | success body rendered via `render.Render(ResourceActions, FormatFull, …)` |
| `glassfrog me projects` | `me projects [--status …] [--base-url …]` | success body rendered via `render.Render(ResourceProjects, FormatFull, …)` |

### `full` format (standing output)

`full` is **field-equivalent to each command's landed pre-019 projection** — same fields, same order, same labels. It is pinned by a golden test capturing current output (ADR-4 / plan Risk "full drifts"); the shapes below describe that contract, the golden is authoritative.

- **`me` full** — labelled identity block; the roles section appears only when `--include roles` was given *and* the response carried roles:
  ```
  actor:        <name> (<kind>) <actor-id>
  organization: <name> (<org-id>)
  access:       <access-level>
  roles:
    - <name> (<role-id>)
  ```
- **`roles` full** — one block per role (blank line between blocks); section headers always render (uniform, agent-parseable), with `    (none)` when a section is empty:
  ```
  <name> (<role-id>)
    Purpose: <purpose | (no purpose set)>
    Domains:
      - <description>
    Accountabilities:
      - <description>
  ```
- **`actions` full** — two lines per action; description falls back to `—` when blank; the `tags:` segment appears only when tags exist:
  ```
  <action-id>  [<status>]  <description | —>
    role: <role-id>   tags: <t1, t2>
  ```
- **`projects` full** — two lines per project; `role` falls back to the no-role marker; `sub-projects`/`actions` render `yes`/`no`:
  ```
  <project-id>  [<status>]  <description | —>
    role: <role-id | (no role)>   sub-projects: <yes|no>   actions: <yes|no>   tags: <…>
  ```

### `compact` format (defined; not operator-selectable until 020)

`compact` is **one line per record**, carrying the essential identifying fields with **ids always present**, and rendering a nested collection as a **count** rather than enumerating it:

| Resource | `compact` line (shape) |
|---|---|
| `me` | `<actor-id>  <name> (<kind>)  org=<org-id>  access=<level>  roles=<N>` — `roles=<N>` present only when `--include roles` was given (the nested collection as a count, vs `full`'s enumeration) |
| `roles` | `<role-id>  <name>  domains=<N>  accountabilities=<N>` — nested sections as counts |
| `actions` | `<action-id>  [<status>]  <description | —>` |
| `projects` | `<project-id>  [<status>]  <description | —>` |

**Example** (`me roles`, three roles):
```
# full (standing)                          # compact (defined, not yet selectable)
Marketing Lead (role_abc)                   role_abc  Marketing Lead  domains=1  accountabilities=2
  Purpose: Reach the right audience         role_def  Facilitator  domains=0  accountabilities=1
  Domains:                                   role_ghi  Rep Link  domains=0  accountabilities=0
    - Brand voice
  Accountabilities:
    - Planning campaigns
    - Measuring reach
…
```

### Empty result (both formats)

When a read returns zero records, the command emits an explicit per-command empty line (no fabricated row, no blank output). The wording is **inherited verbatim from each landed projection** to preserve `full` byte-equivalence:

| Resource | Empty line (full and compact) |
|---|---|
| `roles` | `No roles.` |
| `actions` | `No actions.` |
| `projects` | `no projects` |
| `me` | (no empty case — `me` always resolves a single actor) |

---

## Interactions

**Piping / scripting**: output stays plain text on stdout; `full` is the standing, byte-stable format an existing script already parses (no change). `compact` is denser but not reachable until 020 wires `--output` — so no script can depend on it yet.

**Configuration precedence**: none introduced here. 019 hardcodes `full`; 020 will add `--output` (flag > env > default) and select the `Format`. The read commands keep reading `--base-url`/`--include`/`--status` exactly as before.

**Buffer-then-write**: the command renders the result into a buffer via `render.Render` and writes to stdout only on success (interface-spec.md); on a render error nothing is written to stdout and the command exits non-zero (below). Error output (auth/transport/API failures) is **not** rendered through a template — it keeps its existing cause-plus-next-step format on stderr (spec Non-Behavior).

## Error Communication

| Condition | stdout | stderr | Exit code |
|---|---|---|---|
| Read succeeds, render succeeds | rendered `full` output | — | `0` (Success) |
| Read succeeds, render fails (`*RenderError`) | nothing (no partial output) | the render error message (token-free) | `1` (RuntimeError) |
| Read fails (transport / API / auth / decode) | nothing | existing cause-plus-next-step message (unchanged) | existing code (1/2/3/4/5/6) |

Exit codes are unchanged from 011–014 except that a render failure maps to `RuntimeError(1)` through the existing `Outcome`→`ExitCode` registry — **no new code, no renumbering** (interface-spec.md). The list reads' partial-result incompleteness note (pagination, CONSTITUTION VI) is unchanged: it is rendered output plus a stderr note, orthogonal to which format produced the body.

## Consistency Notes

- **Pairs with interface-spec.md**: that file pins the `internal/render` Go API (`Render`, `Resource`/`Format`, `RenderError`); this file pins what the operator sees. The `full` golden referenced here is the field-equivalence contract the engine must preserve.
- **Preserves the 011–014 output contracts**: `full` reproduces each landed projection field-for-field, so the existing `internal/cli` godog suites and projection assertions stay green — that is the no-regression gate (plan Cross-cutting / Risks). `compact` adds no obligation to those suites; it is tested in `internal/render`.
- **FLAGGED — pre-existing empty-line inconsistency, deliberately not fixed here**: the three list reads emit inconsistent empty lines today — `No roles.` and `No actions.` (capitalized, trailing period) vs `no projects` (lowercase, no period). 019 **inherits each verbatim** to keep `full` byte-equivalent and the landed BDD suites green; normalizing them would change observable output and edit sibling specs' tests, which is out of 019's scope. This is a good candidate for a future cleanup or for 020 to normalize when it touches the output surface. The spec assigns empty-line wording to the renderer (Assumptions), so harmonization is a presentation decision, not a behavioral one.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against; the exit-code mapping follows the frozen 004 convention and the shared `classifyClientError` chain (011).
