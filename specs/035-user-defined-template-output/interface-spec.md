# Interface Accord: User-Defined Template Output — Specification

**Feature**: 035-user-defined-template-output
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (flag-only discriminated selection in `internal/output`), ADR-2 (user template parsed into a clone of 019's set — no second engine), ADR-3 (user-template failures → `UsageError(2)` via a typed error), ADR-4 (file/stdin read behind the cli seam; data-only sandbox by construction).

---

This accord pins three Go contracts, each extending an existing package: the **user-template engine** added to `internal/render` (019's package), the **discriminated selection** added to `internal/output` (020's package), and the **seam + dispatch** wiring in `internal/cli`. It also pins the **template-author data vocabulary** — which decoded value a user template renders against per resource. No new package, no new command, no new flag, env var, or config key is introduced; the only operator surface is the *widened value set* of 020's `-o`/`--output` flag, pinned in `interface-cli.md`. Concrete symbol/field spellings are a build detail; the **shapes, signatures, the discriminated-selection semantics, the parse-vs-execute error split, and the per-resource data value** are the contract.

---

## Surface

### `internal/render` — user-template engine (NEW symbols added to 019's package)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `UserTemplate` | opaque type wrapping a parsed `*text/template.Template` | A caller-supplied template, parsed into a **clone of the built-in set** (`templates.Clone()`), so it shares the package `FuncMap` and `Option("missingkey=error")` and may compose a built-in via `{{template "me.full.tmpl" .}}` (ADR-2). |
| `ParseUserTemplate` | `(text string) (*UserTemplate, error)` | Parses `text` into a clone of the built-in set. Returns `(nil, *UserTemplateError{Stage: Parse, …})` on a syntax error — **no I/O, no data needed**, so the caller can call this before any API request (fail-fast). |
| `(*UserTemplate).Render` | `(data any) (string, error)` | Executes the template against `data` into an in-memory buffer; returns the rendered text on success, `("", *UserTemplateError{Stage: Execute, …})` on an execution error — **never partial output** (buffer-then-return, mirroring `Render`'s ADR-4 contract). Pure over `data`: no I/O, no network, no token. |

`data` is the read's decoded result value — the **same value the built-in `Render(resource, …)` receives** for that resource (see Data Vocabulary). The author writes their template against that value's fields. `text/template` is data-only by construction: it has no file/network/exec primitive, and the reused `FuncMap` adds none (only `trimSpace`/`join`/`indent`), so a user template can only project the data it is handed (ADR-4 — the spec's sandbox non-behavior holds without sandboxing code).

### Error type — `UserTemplateError`

| Type | Shape | Returned when |
|---|---|---|
| `*UserTemplateError` | `{ Stage Stage; Source string; Err error }` where `Stage ∈ {Parse, Execute}` | A caller template fails to **parse** (`Stage: Parse`, pre-request) or fails at **execution** (`Stage: Execute`, post-response — e.g. an unguarded reference to an absent struct field/map key under `missingkey=error`). `errors.As`-discriminable and **distinct from `*RenderError`** (a built-in render failure is a code defect → `RuntimeError(1)`; a user-template failure is operator input → `UsageError(2)`, ADR-3). `Source` is the operator-facing origin (the file path, or `"stdin"`); `Err` wraps the `text/template` cause. Carries **no token** and no request data. |

### `internal/output` — discriminated selection (NEW symbols added to 020's package)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `TemplateRef` | `{ Kind TemplateKind; Path string }` where `TemplateKind ∈ {TemplateFile, TemplateStdin}` | Names a user-template source. `Path` is the file path for `TemplateFile`; empty (ignored) for `TemplateStdin`. Carries only the path/marker — **no file read, no `internal/render` import**. |
| `Selection` | discriminated result: a built-in `OutputFormat` **or** a `TemplateRef` | What resolution now yields. A small struct or sum exposing which arm is set (e.g. `Format OutputFormat` + `Template *TemplateRef`, with `Template == nil` meaning a built-in format). |
| `ResolveSelection` | pure core over injected sources → `(Selection, error)` | Like `ResolveFormat`, but the **flag rung** may yield a `TemplateRef` (see Interactions). Returns `*FormatError` only from the env/file rungs (unchanged) and the rcfile read error unchanged. |
| `ResolveSelectionFromOS` | `(flagValue, startDir, homeDir string) (Selection, error)` | Thin production seam binding the real `os.Getenv` + `internal/rcfile` walk, mirroring `ResolveFormatFromOS`. |

**Reserved flag words** (centralized constants, the single source of truth shared with `interface-cli.md`): the four `ParseFormat` tokens (`full`/`compact`/`json`/`yaml`) **plus** `stdin`. Any non-empty flag value outside this reserved set is a `TemplateRef{TemplateFile, value}`.

### `internal/cli` — seam + dispatch (CHANGED)

| Symbol | Signature (shape) | Description |
|---|---|---|
| seam method | `resolveSelection(flagValue string) (output.Selection, error)` (widened from `resolveFormat → (OutputFormat, error)`) | Production seam binds `output.ResolveSelectionFromOS`; every `--output`-capable read command (the `me*` reads plus `roles`/`role`/`tree`/`subroles`, `domains`/`domain`, `policies`/`policy`) and their fakes adopt the wider return. |
| seam method | `readTemplateSource(ref output.TemplateRef) (string, error)` | Reads the template text: `os.ReadFile(ref.Path)` for a file (relative path resolved against the **current working directory**); the injected bounded-stdin reader for `TemplateStdin`, with an `isTTY`/empty check (reuses the `term.IsTerminal` + `readBoundedStdin` seam established by Credential Storage 006). A missing/unreadable file, or empty/un-piped stdin, returns an error mapped to `UsageError(2)`. |
| render-dispatch | `renderResult[T]` gains a user-template arm | When the selection is a prepared `*render.UserTemplate`: decode the typed `*T`, then write `tmpl.Render(v)`; buffer-then-write (a `*UserTemplateError` leaves stdout empty). Built-in arms (json/yaml via `output.RenderSuccess`, full/compact via `render.Render`) are unchanged. |
| `classifyClientError` arm | (extends the existing function) | Adds `*render.UserTemplateError` → `UsageError`, symmetric with the existing `*output.FormatError` and base-URL arms. **No new exit code.** |

### Consumed unchanged (not defined here)

- **From `019/interface-spec.md`**: the parsed built-in template set, `FuncMap`, `Option("missingkey=error")`, the buffer-then-write contract, `render.Resource` keys, and `render.Render`. `ParseUserTemplate` clones this set.
- **From `020/interface-spec.md`**: `OutputFormat`, `ParseFormat`, `FlagOutput`/`EnvVarOutput`/`outputKey`, `*FormatError`, the precedence resolver shape, and the `classifyClientError` chain. `ResolveSelection` wraps/extends `ResolveFormat`; the env/file rungs keep the four-token contract verbatim.
- **From `004`/`011`**: the `Outcome`→`ExitCode` registry and `UsageError(2)`.

### Data Vocabulary (template-author structural contract)

A user template renders against the **same decoded value the built-in template for that resource receives** — there is no new data shape. The author writes their template against that value's fields (defined by `internal/glassfrog` and each read's `interface-spec.md`):

| `render.Resource` | Data value a template renders against | Field contract source |
|---|---|---|
| `me` | `glassfrog.MeResponse` | 011 / 019 interface-spec.md |
| `roles` | `glassfrog.MyRolesResponse` | 012 / 019 |
| `actions` | `glassfrog.MyActionsResponse` | 013 / 019 |
| `projects` | `glassfrog.MyProjectsResponse` | 014 / 019 |
| `org-roles` | `[]glassfrog.Role` (025) | 025 interface-spec.md |
| `role` | `render.RoleView` (`Detail` + `Requested`) | 025 |
| `tree` | `render.TreeView` (`Rows` + `Requested`) | 026 |
| `subroles` | `render.SubrolesView` (`Children` + `Requested`) | 026 |
| `domains` | `render.DomainsView` (the role's gathered domains) | 033 interface-spec.md |
| `domain` | `render.DomainView` (`Domain` + `Requested`) | 033 |
| `policies` | `render.PoliciesView` (`[]glassfrog.Policy`) | 034 interface-spec.md |
| `policy` | `render.PolicyView` (one `glassfrog.Policy`) | 034 |

This table tracks the built-in `render.Resource` set, which grows as new reads land; every key here renders the invoked read's decoded value. The author is responsible for absence/empty rendering via the same `{{if .X}}…{{else}}—{{end}}` guards the built-ins use; the engine's floor is anti-fabrication (`missingkey=error` fails loud, never silent fake data — it does not auto-inject markers).

**Example (shapes, not literal values)**:
```
// resolve (internal/cli RunE), flag-first:
sel, err := seam.resolveSelection(outputFlag)        // OutputFormat | TemplateRef
if err != nil { return reportFormatResolutionError(stderr, err) }   // *FormatError → UsageError(2)

// if a user template was selected — read+parse BEFORE assembly (fail-fast):
if ref, ok := sel.AsTemplate(); ok {
    text, e := seam.readTemplateSource(ref)          // missing file / empty stdin → UsageError(2), no request
    if e != nil { return reportTemplateError(stderr, e) }
    tmpl, pe := render.ParseUserTemplate(text)       // *UserTemplateError{Parse} → UsageError(2), no request
    if pe != nil { return reportTemplateError(stderr, pe) }
    // ... assemble, build client, send; then:
    s, re := tmpl.Render(v)                          // *UserTemplateError{Execute} → UsageError(2)
    // re → UsageError(2) (buffer-then-write); else write s to stdout
}
```

---

## Interactions

- **Flag-only template recognition (ADR-1)**: `ResolveSelection` consults `--output` flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `DefaultFormat`, the same precedence as `ResolveFormat`. At the **flag rung only**, a non-empty value is classified: a reserved token → that `OutputFormat`; `stdin` → `TemplateRef{TemplateStdin}`; any other value → `TemplateRef{TemplateFile, value}`. The flag short-circuits all lower rungs (flag wins). The **env and file rungs are unchanged** — they parse the four tokens or produce `*FormatError`; they never yield a `TemplateRef`. So precedence stays centralized in `internal/output`, and a template source is reachable only from the command line.
- **Read+parse before request (extends 020 ADR-4)**: 020 resolved the *format* before any request; 035 extends the fail-fast window — when a `TemplateRef` is selected, the command reads the source and **parses** it before assembling the connection or sending. A missing file, unparseable template, or empty/un-piped stdin reports to stderr and returns `UsageError(2)` with **no API request**. The template is executed only after a successful response.
- **Engine sharing (ADR-2)**: `ParseUserTemplate` clones the built-in set, so the user template shares the funcMap, the `missingkey=error` guard, and the built-in named templates. One engine, one configuration — no drift, and no second rendering path.
- **I/O at the seam (ADR-4)**: `internal/render` and `internal/output` stay pure leaves (no file/stdin I/O, no `os`); the file read (`os.ReadFile`, cwd-relative) and stdin read (bounded, `isTTY`-guarded) live behind the cli seam, injected so the fail-fast cases are testable off the real filesystem and terminal.
- **Secret hygiene**: a user template receives only response-side result structs (the token is an `X-Auth-Token` request header, never a result field — continuing 011/019); `UserTemplateError` carries only the source label and the engine cause.

## Error Communication

| Failure | Origin | `cli` mapping | Exit | Request made? |
|---|---|---|---|---|
| Template file missing / unreadable | seam `readTemplateSource` (`os.ReadFile` err) | `reportTemplateError` → `UsageError` | `2` | No |
| `-o stdin` with no pipe (TTY) or empty stdin | seam `readTemplateSource` (isTTY/empty) | → `UsageError` | `2` | No |
| Template fails to parse | `*render.UserTemplateError{Parse}` | `classifyClientError` → `UsageError` | `2` | No |
| Template parses, fails at execution | `*render.UserTemplateError{Execute}` | `classifyClientError` → `UsageError` | `2` | **Yes** (post-response) |
| Present-but-invalid value at env/config (flag absent) | `*output.FormatError` (unchanged) | `UsageError` | `2` | No |
| Read fails (transport/API/auth/decode) | existing typed errors | unchanged (`reportClientError`) | 1/3/4/5/6 | — |

- All user-template failures map to `UsageError(2)` — distinct from a built-in `*RenderError` (`RuntimeError(1)`, a code defect) and an API fault (`APIError(3)`). The post-response execution case spends a request before exiting `2` (unavoidable — `text/template` is dynamically typed, so field existence is not knowable at parse time); the spec covered only the pre-request fail-fast cases explicitly (noted in plan Risks).
- Messages are token-free and name the source (the file path or `stdin`) plus the underlying template error.

## Consistency Notes

- **Extends `internal/render` (019) and `internal/output` (020), bridged by `internal/cli`**: realizes the "029 (user templates)" consumer both specs named. 019 reserved the clone-the-set extension seam; 020 recorded the selection vocabulary as "consumed by 029". The two renderer leaves stay non-importing; `internal/output` gains the `TemplateRef` selection but performs no template I/O and does not import `internal/render`.
- **Conforms to the inject-seam precedent (005/006)**: the file/stdin reads reuse the established `isTTY` + `readBoundedStdin` shape (Credential Storage 006), so the fail-fast cases are hermetically testable.
- **Conforms to 004 / `classifyClientError` (011/015/020)**: adds one arm (`*render.UserTemplateError` → `UsageError`) beside 020's `*FormatError` arm — no new exit code, no renumbering.
- **Pairs with interface-cli.md**: that file pins the operator-facing widened `-o` value set and what a user template puts on stdout; this file pins the Go API and the per-resource data vocabulary that realizes it.
- **032 boundary (downstream)**: Output-Aware Failure Rendering consumes 020's selection; the wider `Selection` type means 032 must decide how a *failure* renders when a user template was selected. 035 renders successes only (failures keep today's cause-plus-next-step form), so it does not pre-empt 032's choice.
- **No `accords/` directory** exists; the only cross-spec contracts are the sibling interface-spec files (019/020/025/026), referenced above.
