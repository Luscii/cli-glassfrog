# Tasks: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, interface-cli.md, features/unconsumable-output/user-defined-template-output.feature

---

## Dependency Graph

Phase 1: User-template engine (`internal/render`) (1 task, no phase dependencies) [Shared]
Phase 2: Discriminated selection (`internal/output`) (1 task, no phase dependencies) [Shared]
Phase 3: Seam, dispatch, classifier + read wiring (`internal/cli`) (3 tasks, depends on Phases 1 and 2) [Shared]

5 tasks total | T001 and T002 startable immediately (parallel — different packages) | 2 phases parallelizable | Builder: pipeline

> **Cross-spec dependency — satisfied.** 035 extends three landed seams: `internal/render` (019 — the built-in template set this feature clones; #50, STATUS "Ready"), `internal/output` (020's selection vocabulary + `ResolveFormat` + `FormatError`; STATUS "Complete"), and `internal/cli` (020's `-o`/`--output` persistent flag, the per-command `resolveFormat` seam, `renderResult` dispatch, and the `classifyClientError` chain; STATUS "Complete"). No upstream is concurrent. Phase 1 (clone 019's engine) and Phase 2 (extend 020's resolver) have no dependency on each other and are startable immediately; Phase 3 wires both into the reads.
>
> Every task is `[Shared]`: the engine, the selection, the classifier arm, the dispatch branch, and the read wiring each serve all three user scenarios (render-through-my-file / pipe-a-one-off / supply-my-own-view) rather than decomposing per scenario — the same shape 020 used.

---

## Branching Guidance

**Pipeline mode**: `spec/035-user-defined-template-output/base` → `spec/035-user-defined-template-output/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: 035 is **downstream** of 018/019/020 (all landed) and a sibling consumer alongside 032 Output-Aware Failure Rendering, which consumes the same 020 selection to render *failures* in the active format. 035 widens the `Selection` type 032 will read; the two are not concurrent here, but 032 must account for the user-template arm when it lands (plan Risks / interface-spec 032 boundary). 033/034 have since landed on `main` — their `domains`/`domain` and `policies`/`policy` reads are now among the `--output`-capable commands Phase 3 must wire. 028 remains in progress and touches unrelated surfaces.

---

## Phase 1: User-template engine (`internal/render`) [Shared]

- [ ] **T001** [Shared] [P] Add the caller-template engine to `internal/render` — `UserTemplate` parsed into a clone of the built-in set, `ParseUserTemplate`, `(*UserTemplate).Render`, and the typed `UserTemplateError` (parse vs. execute).
  - **Scope**: In `internal/render`, add `ParseUserTemplate(text string) (*UserTemplate, error)` that parses caller text into a **clone of the built-in set** (`templates.Clone()`), so the user template shares the package `FuncMap` and `Option("missingkey=error")` and may compose a built-in via `{{template "<resource>.<format>.tmpl" .}}`. Add `(*UserTemplate).Render(data any) (string, error)` executing into an in-memory buffer (buffer-then-return; never partial output). Add `UserTemplateError{Stage, Source, Err}` with `Stage ∈ {Parse, Execute}`, `errors.As`-discriminable and **distinct from `*RenderError`**; `Source` is the operator-facing origin (file path or `"stdin"`). No new `FuncMap` helper is added (data-only sandbox holds by construction). The package still imports only `internal/glassfrog` + stdlib — no `cli`/`output`/`os`.
  - **Acceptance criteria**:
    - `ParseUserTemplate` returns a usable `*UserTemplate` for valid text and `(nil, *UserTemplateError{Stage: Parse, …})` for a syntax error — with no data and no I/O.
    - `(*UserTemplate).Render(v)` returns the rendered text for a guarded template, `("", *UserTemplateError{Stage: Execute, …})` for an unguarded reference to an absent field/key under `missingkey=error`, and never writes partial output.
    - A user template can reference a built-in by name (clone shares the set); the `FuncMap` (`trimSpace`/`join`/`indent`) is available and unchanged; no helper exposes file/network/exec.
    - `UserTemplateError` is `errors.As`-discriminable and not assignable from `*RenderError`. The package imports no `cli`/`output`/`os`; `go build`/`vet` clean; unit + golden tests cover parse error, execute error, guarded-absence marker, and built-in composition.
  - **Dependencies**: None — extends 019's landed package. Startable immediately.
  - **Plan reference**: Phase 1; ADR-2 (clone the built-in set — no second engine), ADR-3 (typed parse-vs-execute error).
  - **Interface references**: interface-spec.md — Surface (`UserTemplate`, `ParseUserTemplate`, `(*UserTemplate).Render`, `UserTemplateError`).
  - **Scenario references**: user-defined-template-output.feature: "A template file renders the result", "A template renders an absence marker for a missing field", "A user template introduces no value absent from the source"

## Phase 2: Discriminated selection (`internal/output`) [Shared]

- [ ] **T002** [Shared] [P] Add the discriminated selection to `internal/output` — `TemplateRef`, `Selection`, `ResolveSelection`/`ResolveSelectionFromOS`, with the flag-only template-ref rule and `stdin` added to the reserved words.
  - **Scope**: In `internal/output`, add `TemplateRef{Kind TemplateKind; Path string}` (`TemplateKind ∈ {TemplateFile, TemplateStdin}`) and a `Selection` discriminating a built-in `OutputFormat` from a `*TemplateRef`. Add `ResolveSelection` (pure core over injected sources) and `ResolveSelectionFromOS(flagValue, startDir, homeDir)` that reuse `ResolveFormat`'s precedence (flag → env → file → default) but, **at the flag rung only**, classify a non-empty value: a reserved format token → `OutputFormat`; `stdin` → `TemplateRef{TemplateStdin}`; any other value → `TemplateRef{TemplateFile, value}`. Add `stdin` to a centralized reserved-words set. The env and file rungs are **unchanged** — still four tokens or `*FormatError`, never a `TemplateRef`. `internal/output` carries only the path/marker — no file/stdin I/O, no `internal/render` import.
  - **Acceptance criteria**:
    - A flag value of `full`/`compact`/`json`/`yaml` (any casing) resolves to the matching `OutputFormat`; `stdin` resolves to `TemplateRef{TemplateStdin}`; any other non-empty flag value resolves to `TemplateRef{TemplateFile, value}` (reserved names win — a value equal to a reserved word never becomes a file).
    - A non-token value in `GLASSFROG_OUTPUT` or the `.glassfrogrc output` key still returns `*FormatError` naming that source — it is **never** treated as a template path.
    - Precedence is unchanged (flag wins; absent skips; present-but-invalid at env/file surfaces loudly); all-absent yields `Full`.
    - `internal/output` imports no `internal/render`; pure-core tests run over injected sources with no real `~/.glassfrogrc`; `go build`/`vet` clean.
  - **Dependencies**: None — extends 020's landed package. Startable immediately (parallel with T001).
  - **Plan reference**: Phase 2; ADR-1 (flag-only discriminated selection).
  - **Interface references**: interface-spec.md — Surface (`TemplateRef`, `Selection`, `ResolveSelection`, `ResolveSelectionFromOS`, reserved words); Interactions (flag-only template recognition).
  - **Scenario references**: user-defined-template-output.feature: "A reserved name wins over a same-named file", "A template source is never honored from env or config"

## Phase 3: Seam, dispatch, classifier + read wiring (`internal/cli`) [Shared]

- [ ] **T003** [Shared] [P] Map `*render.UserTemplateError` to `UsageError` in `classifyClientError` — a new `errors.As` arm beside the `*output.FormatError` arm; unit test pins parse and execute stages both classify usage.
  - **Scope**: In `internal/cli` (`clienterror.go`, the `classifyClientError` chain), add an `errors.As` arm mapping `*render.UserTemplateError` (either `Stage`) to `UsageError`, symmetric with the existing `*output.FormatError` and base-URL arms. Token-free message naming the source. No new exit code.
  - **Acceptance criteria**:
    - `classifyClientError(*render.UserTemplateError{Parse})` and `{Execute}` both return `UsageError`; existing arms (FormatError, base-URL, auth, transport, response) are unchanged.
    - No new exit code; 004's convention intact; `go build`/`vet` clean.
  - **Dependencies**: T001 (`*render.UserTemplateError`).
  - **Plan reference**: Phase 3; ADR-3 (user-template failures → `UsageError(2)`).
  - **Interface references**: interface-spec.md — Error Communication; interface-cli.md — Error Communication.
  - **Scenario references**: user-defined-template-output.feature: "A malformed template fails before any request", "An execution failure writes nothing to stdout"

- [ ] **T004** [Shared] [P] Add the user-template arm to the `renderResult` dispatch — when the selection is a prepared `*render.UserTemplate`, decode the typed `*T` and write `tmpl.Render(v)`, buffer-then-write.
  - **Scope**: In `internal/cli/render.go`, extend the generic `renderResult` so that, alongside the structured (`output.RenderSuccess`) and built-in human (`render.Render`) arms, a prepared `*render.UserTemplate` arm decodes the typed `*T`, executes `tmpl.Render(v)`, and writes the result to stdout only on success (buffer-then-write — a `*UserTemplateError` leaves stdout empty and is returned for classification by T003 → `UsageError(2)`). The optional incompleteness note behaves as on the built-in human path.
  - **Acceptance criteria**:
    - Given a prepared `*UserTemplate` and a successful result, the dispatch writes the template's output to stdout.
    - An execution error returns the `*UserTemplateError` with nothing written to stdout (buffer-then-write).
    - The structured and built-in human arms are unchanged; `internal/render`/`internal/output` remain non-importing siblings (only this dispatch imports both); `go build`/`vet` clean.
  - **Dependencies**: T001 (`*render.UserTemplate`), T002 (`Selection`).
  - **Plan reference**: Phase 3; ADR-2/ADR-3 (engine, buffer-then-write, usage classification).
  - **Interface references**: interface-spec.md — Surface (`renderResult` user-template arm), Error Communication.
  - **Scenario references**: user-defined-template-output.feature: "A template file renders the result", "A template piped on stdin renders the result", "An execution failure writes nothing to stdout"

- [ ] **T005** [Shared] Wire the reads to resolve-selection → read+parse-fail-fast → dispatch — widen the seam to `resolveSelection`, add the injected `readTemplateSource` (file via `os.ReadFile` cwd-relative, stdin via the bounded `isTTY`-guarded reader), parse before assembly, and widen the `-o`/`--output` usage string.
  - **Scope**: In `internal/cli`, widen the per-command seam method from `resolveFormat → (output.OutputFormat, error)` to `resolveSelection → (output.Selection, error)` (production binds `output.ResolveSelectionFromOS`; every `--output`-capable read command — the `me`/`me roles`/`me actions`/`me projects` reads plus `roles`/`role`/`tree`/`subroles` (025/026), `domains`/`domain` (033), and `policies`/`policy` (034) — and their test fakes adopt it). Add a seam `readTemplateSource(ref)` that reads a `TemplateFile` via `os.ReadFile` (relative path against the cwd) and a `TemplateStdin` via the bounded, `isTTY`/empty-guarded stdin reader (reusing the 006 `term.IsTerminal` + `readBoundedStdin` shape). In each read: resolve the selection first; if it is a `TemplateRef`, read the source and `render.ParseUserTemplate` it **before** assembly or any request (a missing file / parse error / empty-or-un-piped stdin → reported to stderr, `UsageError(2)`, no request); pass the prepared `*UserTemplate` to T004's dispatch on a successful response. Widen the `root.go` `-o`/`--output` usage string to name a template file path and `stdin`. Built-in format paths are unchanged.
  - **Acceptance criteria**:
    - `-o ./tmpl` and `-o stdin` (with a pipe) render the invoked read's result through the template; `-o full|compact|json|yaml` and the default path behave exactly as under 020.
    - A missing template file, an unparseable template (file or stdin), and `-o stdin` with no pipe / empty stdin each exit 2 with no API request made (tripwire transport asserts no send), naming the source; an execution failure exits 2 after the response with empty stdout.
    - A non-token value in `GLASSFROG_OUTPUT`/`.glassfrogrc output` still exits 2 as 020's invalid-selector error (not a template path).
    - The reads' existing godog/projection suites stay green for the built-in paths; token-never-in-output tests still pass; the seam fakes drive the file/stdin reads off the real filesystem/terminal; `go build`/`vet` clean.
  - **Dependencies**: T002 (`Selection`/`ResolveSelectionFromOS`), T003 (classifier arm), T004 (dispatch arm), T001 (`ParseUserTemplate`).
  - **Plan reference**: Phase 3; ADR-1 (resolve selection), ADR-4 (file/stdin behind the seam, read+parse fail-fast before request).
  - **Interface references**: interface-cli.md — Surface (widened value set, usage string), Interactions (resolve/read/parse before request), Error Communication; interface-spec.md — Surface (`resolveSelection`, `readTemplateSource`), Interactions (I/O at the seam).
  - **Scenario references**: user-defined-template-output.feature: "A template file renders the result", "A missing template file fails fast", "A malformed template fails before any request", "A template piped on stdin renders the result", "Selecting stdin with nothing piped fails fast", "A reserved name wins over a same-named file", "A malformed template is caught before any API call"
