# Plan: Templated Human Rendering

**Feature**: 019-templated-human-rendering
**Role**: Shaper
**Inputs**: spec.md (019-templated-human-rendering); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: API schema lives in `internal/glassfrog` — 011; each read pairs a pure `run<Read>` + pure `format<Read>` projection renderer behind an injected seam, the reshaped projection is the default and `--output json`/structured output is a deferred future flag — 011 ADR-5, lines 103–104; `internal/apiclient` is transport-only and `apiclient` never imports `cli`; producer-classifies-a-code-free-Outcome / consumer-maps via the single `ExitCode` registry — 002/004/011; `internal/rcfile` is the one-reader pattern, "two readers of one file drift" — 008/LEARNINGS); `.score/memory/LEARNINGS.md` (table-driven change-detector needs a `len`+comma-ok exhaustiveness guard so a dropped entry fails loud — PR #10; godog suites point at their OWN feature file; capture stdout/stderr with a temp file, not `os.Pipe` — PR #10). No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord (Rendering / Full / Compact / Empty-and-absent), three happy + two error + two edge scenarios, three validation scenarios, six reasoned non-behaviors, integration boundaries (011–014, 020, 029), user scenarios, assumptions, no open `[NEEDS CLARIFICATION]`. The three behavioral forks were resolved during `/score:define` (compact test-only until 020; explicit per-command empty line; compact counts nested collections). **One architectural unknown was resolved this session**: the rendering mechanism is Go stdlib `text/template` from day one (Option B), not a Go-renderer interface (Option A) — ADR-1.

---

## System Overview

Templated Human Rendering replaces the four landed per-command projection functions (`formatMe`, `formatMeRoles`, `formatMeActions`, `formatMeProjects`) with a single shared rendering seam: a `text/template`-backed engine in a new `internal/render` package that maps a read's result data to human text through *named templates*. It ships two built-in templates per result type — `full` (field-equivalent to today's projection) and `compact` (a denser, one-line-per-record variant). The seam is the load-bearing piece: 020 Output Format Selection will dispatch to a template by name, and 029 User-Defined Template Output will register caller-supplied templates through the same engine.

The parts:

- **`internal/render` (NEW package — rendering engine + built-in templates)** — owns a parsed `text/template` set whose named templates are the built-ins (`me.full`, `me.compact`, `roles.full`, `roles.compact`, `actions.full`, `actions.compact`, `projects.full`, `projects.compact`), embedded via `//go:embed`. It exposes a single render entry point that, given a *resource key* + *format name* + the result value, executes the matching template into a buffer and returns the text (or a typed error). A small `FuncMap` and `Option("missingkey=error")` carry the Constitution guarantees that pure template text can't express alone (empty-set detection, nested-collection count). The package depends only on `internal/glassfrog` (the result structs it renders) and the stdlib — never on `cli` or `apiclient` (it owns no commands, no transport, no exit codes).
- **The four read commands (MODIFIED `internal/cli/me.go`, `me_roles.go`, `me_actions.go`, `me_projects.go`)** — each drops its inline `formatXxx` projection and instead calls the render seam with its resource key and the **`full`** format (the only format reachable until 020). The pure `run<Read>` orchestration, the injected transport seam, and the error-classification path are unchanged.

```
glassfrog me / me roles / me actions / me projects
  └─ run<Read>(cfg):  … Execute → result value (glassfrog.*Response)
        success → render.Render(resourceKey, render.FormatFull, result) → buffer → stdout   [019]
                    │  text/template set (//go:embed), FuncMap, missingkey=error
                    │  exec error (built-in bug) → RuntimeError(1); nothing written
        error   → classifyClientError(err) → message (unchanged; NOT templated)

internal/render (NEW)      template set {me,roles,actions,projects}×{full,compact}; Render(resource,format,data)→(string,err)
internal/glassfrog (dep)   MeResponse / MyRolesResponse / MyActionsResponse / MyProjectsResponse  (render inputs; unchanged)
internal/cli (modified)    four reads call render.Render(…, render.FormatFull, …) instead of formatXxx
  ── compact templates exist + are tested, but no CLI path selects them until 020
```

Rendering operates on the **response-side result structs only** — the token is a request header, never a result field — so the secret-never-emitted rule holds by construction (continuing 011).

---

## Architecture Decisions

### ADR-1: The rendering mechanism is Go stdlib `text/template`; built-in `full`/`compact` are embedded template definitions per result type (resolves the architectural unknown)

**Context**: The spec requires results to render "through a template mechanism … the template seam is what later admits caller-supplied templates (029)." Two realizations were considered. Whichever is chosen is the foundation 020 (selection) and 029 (user templates) build on — costly to reverse.

**Options considered**:
1. **A `Renderer` interface with Go-code built-ins** — `full`/`compact` are Go functions (today's `formatXxx` refactored) behind an interface; 029 later plugs a `text/template` renderer in behind the same interface. Type-safe, minimal churn, Constitution guarantees stay in plain Go — but built-ins and user templates use two different engines, and "template" is interpreted loosely as "named renderer."
2. **`text/template` from day one** — `full`/`compact` are actual template definitions over the result structs; 029's caller-supplied templates reuse the very same engine. One literal template engine for built-ins and the future user-template feature; most faithful to "template." Cost: stringly-typed, render-time (not compile-time) errors, and the data-fidelity rules need template-side guards plus helper funcs.

**Decision**: Option 2 — `text/template`. `internal/render` parses a template set whose named members are the eight built-ins, embedded as files via `//go:embed`. A read renders by resource key + format name. `text/template` (not `html/template`) is correct: CLI output is plain text, not HTML, so no auto-escaping is wanted. It is stdlib, so no dependency is added (CONSTITUTION XII holds).

**Consequences**: 029 becomes a natural extension — a caller-supplied template is just another template parsed into (a clone of) the same set, executed by the same engine, so no second rendering path is introduced. 020 selects a built-in by name. The cost is paid in ADR-3 (data-fidelity guards must live in template text + a FuncMap, pinned by golden tests) and ADR-4 (render-time errors handled by buffer-then-write). This **diverges from 011 ADR-5's** "pure `format<Read>` Go projection renderer per command" precedent — see ADR-2's consequences and the deprecation note in the handoff.

### ADR-2: Rendering lives in a new `internal/render` package, depending only on `internal/glassfrog` and the stdlib

**Context**: The engine and its templates are reused by all four reads now and by 020/029 later; they must live somewhere that both `cli` (the read commands) and a future selection layer can import without a cycle. Precedent: schema in `internal/glassfrog`, transport in `internal/apiclient` (transport-only, never imports `cli`), commands in `internal/cli`. The codebase keeps one home per concern.

**Options considered**:
1. **A new `internal/render` package** — engine + embedded templates + the `FuncMap`. Imports `internal/glassfrog` (the structs it renders); imported by `internal/cli`. Clean rendering-vs-command split; the obvious home for 020/029 to extend.
2. **Put rendering in `internal/cli`** — rejected: `cli` owns commands and exit-code classification, not a reusable rendering engine; 020/029 would import command code, and the templates would sit beside cobra leaves.
3. **Put rendering in `internal/glassfrog`** — rejected: `glassfrog` is leaf *schema* (no behavior); adding template execution would broaden it the way 010 declined to put schema in `apiclient`.

**Decision**: Option 1. `internal/render` is the rendering home. It depends on `internal/glassfrog` (render inputs) and stdlib (`text/template`, `embed`, `bytes`); it must not import `internal/cli` or `internal/apiclient` (mirrors the `apiclient`-never-imports-`cli` layering). The exact exported API (function signature, resource-key/format-name types, `FuncMap` helpers) is interface-level (`/score:interface`).

**Consequences**: A clean four-way layering — `glassfrog` (schema) ← `render` (presentation) ← `cli` (commands); `apiclient` (transport) ← `cli`. 020 adds the `--output` flag in `cli` and calls `render` with a selected format name; 029 extends `render` with caller-supplied templates. This is where the 011 "projection renderer per command" pattern is superseded: the per-command Go formatter moves out of `cli` and becomes a per-result-type template in `render`. *Precedent-setting + divergence — recorded in DECISIONS; a `/score:deprecate` of 011's per-command-formatter note is suggested in the handoff.*

### ADR-3: Built-in templates render over the typed result structs with `missingkey=error` + a minimal `FuncMap`; the Constitution data-fidelity rules are enforced in template text and pinned by golden tests

**Context**: `text/template` renders an absent struct field as its zero value (empty string) and a missing map key as `<no value>` by default — both would violate "present only data the API returned; never fabricate a *data value*" (CONSTITUTION, data fidelity) and the spec's "preserve the landed explicit-absence markers" / "explicit empty line" / "compact counts nested collections" rules (per the 2026-06-08 clarification: a `full` template keeps each landed read's explicit-absence markers — `(none)`, `—`, `(no purpose set)`, `(no role)` — except `me`, which omits its empty roles section; the rule forbids inventing a data value, not these emptiness markers). Pure template text can express *marker-or-omit-when-absent* (`{{if .X}}…{{else}}<marker>{{end}}`) and *enumerate* (`{{range}}`), but empty-set detection and nested-collection counts read cleanly only with helpers.

**Options considered**:
1. **Raw result structs + `Option("missingkey=error")` + a small `FuncMap`** — templates render the `glassfrog.*Response` values directly; `missingkey=error` makes any truly-missing key fail loud (no silent `<no value>`); `{{if .X}}…{{else}}<marker>{{end}}` renders the landed explicit-absence marker for a blank field (and `me`'s empty roles section is omitted, matching its landed projection); `{{range}}` enumerates under `full`; a `len`/`count` helper drives the compact nested-count and the explicit empty line. Minimal layers; the templates read close to the data.
2. **Per-type view models** — precompute a presentation struct per result type (counts, "isEmpty", formatted lines) and template over that. Cleaner templates, but adds a parallel model layer and moves logic out of the template the spec wanted to be the locus of presentation.

**Decision**: Option 1 — render over the typed structs with `missingkey=error` and a minimal `FuncMap`. The data-fidelity guarantees are realized as: **ids always present** = the template writes the id line unconditionally; **absent/blank field** = `{{if .X}}…{{else}}<marker>{{end}}` renders the landed explicit-absence marker (`(none)`, `—`, `(no purpose set)`, `(no role)`), except `me`'s empty roles section, which is omitted as its landed projection does; **empty result set** = `{{if not .Data}}no <resource>{{else}}…{{end}}` (the explicit per-command empty line, Q-empty resolution); **compact counts a nested collection** = a `count` helper (`roles={{len .Roles}}`) instead of `{{range}}`; **never fabricate** = no template invents a *data value* the API didn't return (id/name/status/field value), and `missingkey=error` backstops typos — explicit emptiness markers report absence and are not fabricated data (2026-06-08 clarification). A FuncMap helper is added only where a guarantee can't be expressed inline.

**Consequences**: Presentation logic lives in the template text (the spec's intent) with a thin helper layer. The guarantees are not compiler-enforced, so they are pinned by **golden/unit tests** per result type per format (ADR-4 / Cross-cutting): `full` golden = the captured pre-019 output (field-equivalence), absent/blank field → landed explicit-absence marker (or omission, e.g. `me`'s roles section), empty set → explicit line, compact nested → count. The exact `FuncMap` helper set and template text are interface/implementation detail.

### ADR-4: Render into a buffer then write; a render-time error is an internal bug → `RuntimeError(1)`; the standing CLI format is `full`, with `compact` built-and-tested but unselectable until 020

**Context**: `text/template` execution can fail partway through, having already written bytes. Emitting partial output then erroring would violate "fail safe, not silent" (CONSTITUTION III) and corrupt agent-parsed output. Separately, the clarify session fixed (Q1) that `compact` ships built and verified but is reachable from no operator surface until 020 wires `--output`; the standing output must remain `full`, byte-equivalent to today.

**Options considered**:
1. **Render to a `bytes.Buffer`, check the error, then write to stdout only on success** — on a render error nothing reaches stdout; the command maps the error to `RuntimeError(1)` (a built-in template that fails to execute is a code defect, like 011's `*DecodeError`→`RuntimeError`). The read always asks for `full`; `compact` templates are parsed and unit-tested but no CLI call selects them.
2. **Stream template execution straight to stdout** — rejected: a mid-execution failure leaves partial output on stdout (Fail-Safe violation) and the exit code then contradicts what was printed.

**Decision**: Option 1. `render.Render` executes into a buffer and returns `(string, error)`; the command writes the string to stdout only when `err == nil`, else routes the error to `RuntimeError(1)` through the existing `Outcome`→`ExitCode` path (no new exit code). The four reads pass the constant `full` format; `compact` is exercised directly by `internal/render` tests, satisfying Q1's "built and verified but not CLI-reachable."

**Consequences**: No partial output is ever emitted; a built-in-template defect fails loud at code 1, consistent with 011's undecodable-body handling. The standing output stays `full` and is pinned byte-for-byte against the pre-019 projection by golden tests, so the rewire is provably non-regressive. `compact`'s correctness does not depend on 020 landing — its tests stand alone. When 020 arrives it supplies the format name and removes the hardcoded `full` at the call sites; no `render` change is needed.

---

## Cross-cutting Concerns

**Data fidelity / anti-fabrication (CONSTITUTION — present only data the API returned)**: `Option("missingkey=error")` + `{{if .X}}…{{else}}<marker>{{end}}` guards + no invented data values in any template; golden tests assert a blank field renders its landed marker (`—`/`(none)`/`(no purpose set)`/`(no role)`) — or is omitted, as `me`'s roles section is — is never rendered as `<value>`/empty from a missing key, and that every *data value* shown traces to a result field. This is the spec's central guarantee and gets the most test weight.

**Error handling (CONSTITUTION III — fail safe, not silent)**: buffer-then-write (ADR-4) means a render failure prints nothing and exits 1; success prints the complete buffer. The error path of the read (transport/API/auth) is untouched and is **not** routed through `render` (spec Non-Behavior: error output is not templated).

**Testing (CONSTITUTION IV — RED-first, hermetic)**: templates are pure (`data → string`), so `internal/render` is unit-tested with no network: per result type, `full` against a captured golden, `compact` for one-line-per-record + nested count + ids-present, both for empty-set (explicit line) and absent/blank field (landed marker, or omission for `me`'s roles section). A **registry exhaustiveness test** asserts every read resource key has both a `full` and a `compact` template registered (`len`+comma-ok guard, PR #10 LEARNINGS — a dropped template fails loud, not silently). The four reads' existing `internal/cli` godog suites and projection unit tests continue to pass unchanged — that *is* the no-regression pin for `full`. Capture uses a temp file, not `os.Pipe` (PR #10).

**Secret hygiene (CONSTITUTION II)**: `render` only ever receives response-side result structs; the token is a request header, not a result field, so it cannot appear in rendered output (continuing 011). Pinned by the reads' existing token-never-in-output tests, which still cover the now-templated success path.

**Configuration**: built-in templates are compile-time `//go:embed` assets — no runtime configuration (029 introduces caller-supplied templates; 020 introduces selection). Resource keys and format names are centralized constants in `internal/render`, so call sites and templates can't drift (mirrors the rcfile one-source-of-truth discipline).

---

## Implementation Strategy

Two phases. The upstream dependency (`internal/glassfrog` result structs) is **landed on main** — all four reads are implemented (STATUS: 011–014 Complete), so the result types and their current projections exist to render and to golden-capture.

- **Phase 1 — `internal/render` engine + the eight built-in templates**: create the package; embed the template files; parse them into one set with `missingkey=error` + the `FuncMap`; expose the buffer-then-render entry point (ADR-2/3/4). Author `full` (field-equivalent to each read's current `formatXxx`) and `compact` (one line per record; ids present; nested collections as a count) for `me`, `roles`, `actions`, `projects`. RED-first unit tests: full goldens captured from the current four projections; compact behavior; empty-set explicit line; absent/blank-field markers (and `me`'s empty-roles omission); the registry exhaustiveness guard. *Depends on: `internal/glassfrog` (landed). No CLI change yet — `compact` is verified here, independent of 020.*
- **Phase 2 — rewire the four reads to render through the seam with `full`**: in each of `me.go`, `me_roles.go`, `me_actions.go`, `me_projects.go`, replace the inline `formatXxx(resp)` + `Fprint` with `render.Render(resourceKey, full, resp)` → buffer → stdout, mapping a render error to `RuntimeError(1)`; remove the now-dead `formatXxx` functions and their pure unit tests (superseded by the golden tests in Phase 1). The reads' existing BDD suites must stay green (output unchanged) — the regression gate. *Depends on: Phase 1. Naturally splits into four PR-sized units (one per read) for the tasks skill.*

---

## Risks

- **`text/template` silently renders absent data as a zero value, fabricating output** (medium likelihood, high impact — it's the spec's core guarantee): an unguarded `{{.Field}}` prints empty for an absent field; a typo'd key prints `<no value>`. *Mitigation*: `Option("missingkey=error")` (loud on missing keys) + mandatory `{{if .X}}…{{else}}<marker>{{end}}` guards on optional fields + golden tests asserting a blank field renders its landed marker (or omits, for `me` roles) and never a fabricated data value. Reviewed as the highest-weight test area.
- **Partial output on a mid-render failure** (low likelihood, high impact): streaming execution could leave bytes on stdout before erroring. *Mitigation*: buffer-then-write (ADR-4); stdout is written only on `err == nil`; a render error exits 1 with nothing printed.
- **`full` drifts from the pre-019 projection, regressing shipped output** (medium likelihood, medium impact): re-expressing four hand-written projections as templates risks subtle layout changes. *Mitigation*: golden tests capture the current output verbatim and assert equality after the rewire; the reads' existing BDD/projection tests stay green as a second gate.
- **`compact` rots untested because no CLI path reaches it until 020** (medium likelihood, medium impact): an unreachable surface tends to lose coverage. *Mitigation*: `compact` is fully unit-tested in `internal/render` independent of CLI reachability (ADR-4) — its correctness never depended on 020.
- **Template/format keys drift from result types** (low likelihood, medium impact): stringly-typed resource/format keys could mismatch a result type. *Mitigation*: centralized key constants + the registry exhaustiveness test (every read has both formats); `missingkey=error` and a render-time "no such template" error fail loud.
- **Divergence from 011's per-command-formatter precedent confuses future reads** (low likelihood, low impact): later reads might still add a `formatXxx` out of habit. *Mitigation*: the divergence ADR + a DECISIONS entry establish the seam as the pattern; a `/score:deprecate` of the 011 note is suggested so the superseded precedent is explicit.

---

## What This Plan Does Not Cover

- **The `--output` flag, format selection, and the default-when-omitted** — 020 Output Format Selection; 019 hardcodes `full` at the call sites and exposes the named templates 020 will select among.
- **JSON / YAML machine output** — 018 Structured Serialization (a sibling renderer, not a human template).
- **Caller-supplied template files and their resolution/sandboxing** — 029 User-Defined Template Output, which extends the same engine.
- **The exact exported API of `internal/render`** — function signature, resource-key/format-name types, `FuncMap` helper set, and the literal template text/layout — `/score:interface` pins the CLI and specification boundaries.
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into a feature file under `features/`.
- **Per-read task breakdown** — `/score:tasks` decomposes Phase 2 into one PR-sized unit per read.
