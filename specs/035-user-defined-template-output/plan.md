# Plan: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, `.score/memory/DECISIONS.md` (precedent grep: 019/020 render+selection, "029 clones the same set"), `.score/memory/DEPRECATION.md` (render-failure→RuntimeError(1) fault axis); existing code `internal/render`, `internal/output`, `internal/cli` (render dispatch, root flag, per-command resolveFormat seam)

---

## System Architecture

User-Defined Template Output is a *thin extension across the three existing Output-Formatting layers* — it adds no new package. It plugs into the seam 019 built and 020 selects through, exactly as both specs anticipated ("029 registers caller-supplied templates into a clone of the same set, no second engine"). The change is to widen, at one rung, what a resolved output selection can be, and to add one rendering path through the already-parsed template engine.

Three layers, each keeping its established role:

```
  -o / --output flag value
        │
        ▼
  internal/output ── ResolveSelection (extends 020's ResolveFormat)
        │            flag rung: reserved token → OutputFormat (unchanged)
        │                       "stdin"        → TemplateRef{Stdin}      ┐ flag-only
        │                       any other value→ TemplateRef{File, path} ┘
        │            env / file rungs: UNCHANGED — only the 4 tokens, else *FormatError
        ▼
  internal/cli ──── per-command seam + renderResult dispatch
        │   • reads the template source (os.ReadFile | injected stdin reader)
        │   • parses it FIRST, before any API request (fail-fast → UsageError 2)
        │   • on a successful read, renders the typed result through the prepared template
        ▼
  internal/render ─ UserTemplate: caller text parsed into a CLONE of the built-in
                    set (same funcMap, same Option("missingkey=error")); execute
                    against the resource's existing data value; buffer-then-write
```

**Data flow inside a read command** (e.g. `me roles`) becomes:

1. Read the inherited `--base-url` and `--output` flag values (unchanged).
2. **Resolve the render selection** (flag-first). A built-in `OutputFormat`, or — only when the *flag* carried a non-reserved value — a `TemplateRef` (a file path, or the `stdin` marker). Env/config still resolve to one of the four tokens or a `*FormatError`.
3. **If a `TemplateRef`**: read its bytes (file from disk / injected stdin) and **parse** them into a `UserTemplate` — *before* assembly or any request. A missing file, unparseable template, or empty/un-piped stdin fails fast as `UsageError(2)`, no API call.
4. Assemble the connection and build the client (unchanged).
5. Send the one request; on success, render the typed result either through the built-in renderer (json/yaml/full/compact, unchanged) **or** through the prepared `UserTemplate`.

The render selection is computed once per invocation; the template is read+parsed once, up front; execution happens once, after a successful response. `internal/render` stays a pure leaf (text + data in, text out), `internal/output` stays free of template/file I/O, and the file/stdin reads live behind the cli seam — the same three-way split the cluster already uses.

---

## Architecture Decisions

### ADR-1: Template-source recognition is a flag-only extension of 020's resolver, returning a discriminated selection

**Context**: The spec reuses `-o`/`--output` (020's flag) so a non-reserved value names a template source, but *only at the flag rung* — env var and config file keep 020's four-token-or-`FormatError` behavior (the operator confirmed flag-only sourcing, because a template is shaped to one resource type). The precedence short-circuit ("flag present → flag wins, consult nothing lower") lives inside `output.ResolveFormat`. Recognition must not fork that precedence logic.

**Options considered**:
1. **Intercept in `internal/cli` before calling the resolver** — cli inspects the flag value, peels off template sources, and only delegates the rest to `output.ResolveFormat`. Keeps `internal/output` untouched, but re-implements the "flag wins" short-circuit in cli, forking precedence across two places.
2. **Extend `output` to return a discriminated `Selection`** — `ResolveFormat`'s flag-rung branch emits a `TemplateRef` for a non-reserved, non-empty value instead of a `*FormatError`; the env/file branches are unchanged. Precedence stays in one place; `output` carries only the path string / a `stdin` marker (no file read, no `render` import).

**Decision**: Option 2. `internal/output` owns the selection vocabulary (020 ADR), and DECISIONS records it as "consumed by 029 (user templates)" — extending it here is the precedent-honoring home. A new resolution entry point (e.g. `ResolveSelection`) returns a discriminated result: a built-in `OutputFormat`, or a `TemplateRef{Kind: File|Stdin, Path string}`. The `TemplateRef` outcome is reachable **only from the flag rung**; the env and config rungs call the existing four-token path and still produce a `*FormatError` for any non-token value. `output` holds only the path/marker — it performs no I/O and does not import `internal/render`.

**Consequences**: Precedence and "present-but-invalid surfaces loudly" stay centralized in `output`. The per-command `resolveFormat` seam signature widens from `(output.OutputFormat, error)` to the discriminated selection — a mechanical change at every `--output`-capable read site (the `me*` reads plus `roles`/`role`/`tree`/`subroles`, `domains`/`domain`, `policies`/`policy`) and their fakes. `internal/output` still never imports `internal/render`. 032 (failure rendering), which consumes 020's selection, will see the wider selection type and must decide how a failure renders when a user template was selected (noted as a downstream boundary, not resolved here).

### ADR-2: A user template is parsed into a clone of the built-in set — no second engine

**Context**: 019 built one `text/template` set (`Option("missingkey=error")`, a pure `funcMap` of `trimSpace`/`join`/`indent`), and DECISIONS commits 029 to "a clone of the same set (no second engine)." The user template needs the same helpers and the same data-fidelity guard, and may want to compose built-in templates.

**Options considered**:
1. **A fresh `text/template` per user template** — parse the caller text in isolation. Simple, but forks engine configuration (funcMap, missingkey policy) into a second place that can drift from the built-ins, and can't reference built-in templates.
2. **Parse into a clone of the built-in set** — `templates.Clone()` then parse the caller text into the clone. Same funcMap, same `Option("missingkey=error")` by construction; the user template can `{{template "me.full.tmpl" .}}` to compose a built-in.

**Decision**: Option 2. `internal/render` gains a `UserTemplate` type and two entry points: parse caller text into a clone of the base set (returns a parse error), and execute it against a data value (buffer-then-write, returns text or an execution error). Configuration is shared by construction — the anti-fabrication guard (`missingkey=error`: a missing map key fails loud, never renders silent fake data) applies to user templates identically to built-ins. The pure `funcMap` is reused unchanged; no `env`/`exec`/`readFile`-style helper is added, so the data-only sandbox holds by construction (see ADR-4).

**Consequences**: One engine, one configuration, one funcMap — no drift. A user template may reference built-in named templates. Absence-marker rendering is the *author's* responsibility via the same `{{if .X}}…{{else}}—{{end}}` guards the built-ins use; the system's floor is anti-fabrication, not automatic marker injection (the spec's "renders an absence marker" case is the guarded-template case). A truly-missing key or struct field in an unguarded user template surfaces as an execution error (classified by ADR-3), never as a fabricated value.

### ADR-3: All user-template failures map to UsageError(2), via a typed render error

**Context**: The spec classifies the pre-call cases (missing file, unparseable template, empty/un-piped stdin) as fail-fast usage errors with the conventional usage exit code (004). A template that parses but fails at *execution* (an unguarded reference to an absent struct field/map key) can only be detected after the response is in hand — text/template is dynamically typed, so field existence isn't knowable at parse time. The existing fault axis (DEPRECATION 2026-06-10): exit 3 = "the API's fault", exit 1 = "our fault" (a built-in `*RenderError` is a code defect), exit 2 = usage.

**Options considered**:
1. **Execution failures → RuntimeError(1)** — reuse 019's built-in render-failure mapping. But exit 1 means "the CLI's own defect"; a user template failing is the *operator's* input, not a CLI bug — it would mis-signal fault.
2. **All user-template failures → UsageError(2)** — both parse (pre-call) and execution (post-call) failures are the operator's template to fix, symmetric with how a malformed `--output` value (`*FormatError`) and a malformed `--base-url` already map to `UsageError(2)`.

**Decision**: Option 2. `internal/render`'s user-template entry points return a typed, `errors.As`-discriminable error (distinct from the built-in `*RenderError`) that the cli classifier maps to `UsageError(2)` — a new arm symmetric with the existing `*output.FormatError → UsageError` arm. Parse failures are caught up front (fail-fast, no request); execution failures are caught post-response and still exit 2.

**Consequences**: Uniform "anything wrong with the operator's template is a usage error (2)" — distinct from built-in render defects (1) and API faults (3), keeping the fault axis sharp. The one wrinkle: an *execution* failure occurs after a successful API call, so a request was spent before the exit-2 — acceptable, and unavoidable given text/template's dynamic typing. The spec only explicitly covered the pre-call fail-fast cases; this ADR extends the same classification to the post-call execution case the spec left implicit (flagged in Risks).

### ADR-4: File and stdin reads live behind the command seam; the data-only sandbox is preserved by construction

**Context**: Reading a template file from disk and reading a piped template from stdin are I/O. The cluster's leaves are pure: `internal/render` and `internal/output` perform no I/O, and interactive/stdin input is injected through the cli seam (DECISIONS 2026-06-04: "interactive-input commands inject the stdin/TTY/env/dir seam"). The spec also forbids the template from executing commands, reading other files, or reaching the network.

**Options considered**:
1. **Read inside `internal/render` or `internal/output`** — convenient, but breaks the leaf purity both packages hold and bypasses the injected-seam testability precedent.
2. **Read behind the cli seam** — the per-command seam (which already injects `assemble`/`newClient`/`resolveFormat`, and elsewhere stdin/TTY) gains the template-source read: `os.ReadFile` for a file path, the injected stdin reader for `stdin`. The bytes flow to `render`'s parse entry.

**Decision**: Option 2. The seam reads the source (file via `os.ReadFile`, resolving a relative path against the current working directory; stdin via the injected reader with an `isTTY`/empty check). The data-only sandbox is preserved *by construction*: `text/template` has no file/network/exec primitives, and the reused `funcMap` adds none — so a user template can only project the data it is handed. The seam reads exactly the one named template file and nothing the template asks for.

**Consequences**: `render` and `output` stay pure leaves and fully unit-testable; the file/stdin reads are injectable, so the fail-fast cases (missing file, empty stdin, not-a-pipe) are testable off the real filesystem and terminal. Empty or un-piped stdin under `-o stdin` is detected at the seam and becomes `UsageError(2)` before any request. No sandboxing code is needed because no escape primitive is ever exposed.

---

## Cross-cutting Concerns

**Error handling**: One fail-fast point (template read+parse, before assembly) and one post-response point (template execution); both route the typed user-template error to `UsageError(2)` (ADR-3). Buffer-then-write is preserved — a user-template execution failure leaves stdout empty, never a partial render. Messages are token-free (the token is a request header, never in scope on the render path — continuing 019/011) and name the source (the file path, or `stdin`) and the underlying template error.

**Configuration**: The `-o`/`--output` flag is unchanged structurally (still the persistent root `StringP`, short `-o`); only its *usage string* widens to mention a template file / `stdin`, and its accepted value set widens at the flag rung. No new flag, env var, or config key is added — flag-only sourcing (ADR-1) means the env var and `.glassfrogrc output` key keep their four-token contract verbatim.

**Testing**: `internal/render` user-template parse/execute is pure and golden/unit-testable (parse error, execution error, absence-marker-via-guard, built-in composition). `internal/output` selection discrimination is pure (flag→TemplateRef; env/file non-token→FormatError). The cli seam's file/stdin reads are injected, so fail-fast cases are driven over fakes; BDD scenarios cover the full path (file template renders, stdin template renders, reserved-name-wins, missing file, malformed template, empty stdin) with no real network or `~/.glassfrogrc`.

## Implementation Strategy

Three phases. Phases 1 and 2 are independent pure-leaf changes; Phase 3 integrates them at the cli layer.

- **Phase 1 — `internal/render` user-template engine**: add the `UserTemplate` type, parse-into-clone and execute entry points, and the typed user-template error (parse vs. execute). Pure, no cli/output dependency. Unit + golden tests. (ADR-2.)
- **Phase 2 — `internal/output` selection discrimination**: add the `TemplateRef` type and the `ResolveSelection` entry point that returns a built-in format or a flag-only `TemplateRef`; env/file rungs unchanged. Pure. Unit tests over hand-built sources. (ADR-1.)
- **Phase 3 — `internal/cli` wiring**: widen the per-command `resolveFormat` seam to the discriminated selection; add the seam's file/stdin read (injected); insert the read+parse fail-fast step before assembly; add the user-template branch to `renderResult`; add the user-template-error → `UsageError(2)` classifier arm; update the `-o`/`--output` usage string. Migrate every `--output`-capable read command (the `me*` reads plus `roles`/`role`/`tree`/`subroles`, `domains`/`domain`, `policies`/`policy`) and their fakes. BDD scenarios. (ADR-3, ADR-4.) Depends on Phases 1 and 2.

## Risks

- **Reserved-name shadowing** *(low likelihood, low impact)*: an operator who has a file named `full`/`json`/`stdin` and expects `-o full` to read it gets the built-in instead. Mitigation: reserved-names-win is specified and documented, with the `./full` escape; the interface usage string names the reserved set.
- **Post-call execution-error classification** *(medium likelihood, low impact)*: the spec explicitly covered only the pre-call fail-fast cases; an unguarded user template that parses but references an absent field fails *after* the API call and exits `UsageError(2)` (ADR-3). A request is spent before the exit. This is unavoidable (text/template dynamic typing) and the spec's intent (operator-input failure = usage error) is preserved — but it is an extension of the spec's letter and should be confirmed if the post-call timing matters to a consumer.
- **Seam signature churn** *(medium likelihood, low impact)*: widening `resolveFormat` from `(OutputFormat, error)` to the discriminated selection touches every `--output`-capable read command — a dozen across the `me*`, `roles`/`role`/`tree`/`subroles`, `domains`/`domain`, and `policies`/`policy` reads — and their test fakes. Mitigation: it is a mechanical, compiler-enforced change; the discriminated type is the single new shape every site adopts.
- **032 boundary** *(low likelihood, medium impact)*: Output-Aware Failure Rendering (032, not yet landed) consumes 020's selection to render *failures* in the selected format. A user template is a success-only renderer; 032 must decide how a failure renders when a user template was selected (e.g. fall back to the human cause-plus-next-step form). Noted as a downstream boundary; 035 does not render failures through a user template (non-behavior), so it does not pre-empt 032's choice.

---

## What This Plan Does Not Cover

- **Protocol-level surface** (interface skill): the exact `ResolveSelection`/`TemplateRef` signatures, the `UserTemplate` API and error type names, the widened seam interface, the precise `-o`/`--output` usage string, and — importantly — the **per-resource data vocabulary a template author writes against** (which fields each resource's data value exposes: `glassfrog.MeResponse`, `MyRolesResponse`, `RoleView`, `TreeView`, etc.). The template *language* is Go `text/template` (inherited from 019, not a new choice); the field contract per resource is the interface skill's to pin in `interface-spec.md`.
- **Executable scenarios** (scenarios skill): the Gherkin for the driving scenarios.
- **Task decomposition** (tasks skill): PR-sized units across the three phases.
- **Failure rendering under a selected user template** (032): out of scope here; this feature renders successes only and keeps failures in their existing cause-plus-next-step form.
