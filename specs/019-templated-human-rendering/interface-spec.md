# Interface Accord: Templated Human Rendering — Specification

**Feature**: 019-templated-human-rendering
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4 — the new `internal/render` package: the `text/template`-backed engine, its `Render` entry point keyed by resource + format, the embedded built-in template set ({me,roles,actions,projects}×{full,compact}), the `FuncMap` + `missingkey=error` data-fidelity guards, and the buffer-then-return error contract consumed by the four read commands now and by Output Format Selection (020) / User-Defined Template Output (029) later.

---

This accord pins the **Go API surface of `internal/render`**: the **`Render`** entry point, the **`Resource`/`Format`** key types and their built-in constants, the embedded **template set** and its naming contract, the typed **`RenderError`**, and the **registry-exhaustiveness** guarantee. There is **no command and no entry point** in this capability — rendering is a library function the read commands call — so the *invocation* surface is **N/A**; the `--output` flag that will select a `Format` is **020's** surface, not this one. The capability introduces **no configuration of its own** — no `.glassfrogrc` key, env var, or flag; the built-in templates are compile-time `//go:embed` assets and the resource/format keys are package constants. This is the human-rendering mirror of the cluster: 018 Structured Serialization is the machine-format sibling; this accord is the `text/template` engine the spec's "template mechanism" names.

---

## Surface

### Entry point

| Function | Signature (shape) | Description |
|---|---|---|
| `Render` | `Render(resource Resource, format Format, data any) (string, error)` | Executes the built-in template named `<resource>.<format>` against `data`, into an in-memory buffer, and returns the rendered text on success. Returns `("", *RenderError)` on any failure (unknown resource/format, or a template execution error) — **never partial output** (ADR-4). Pure over its inputs (no I/O, no network, no token); the same `(resource, format, data)` always yields the same string. |

`data` is the read's decoded result value — `glassfrog.MeResponse`, `glassfrog.MyRolesResponse`, `glassfrog.MyActionsResponse`, or `glassfrog.MyProjectsResponse` (the structs the read surface already decodes; unchanged by 019). The template for a given `resource` expects the matching type; a type/template mismatch surfaces as a `*RenderError` (ADR-3/4), never a silent zero-value render.

### Key types

| Type | Shape | Built-in values |
|---|---|---|
| `Resource` | named string | `ResourceMe` (`"me"`), `ResourceRoles` (`"roles"`), `ResourceActions` (`"actions"`), `ResourceProjects` (`"projects"`) — one per read result type |
| `Format` | named string | `FormatFull` (`"full"`), `FormatCompact` (`"compact"`) — the two built-in templates |

These constants are the single source of truth for the keys: the read commands pass them, the template names derive from them (`<resource>.<format>`), and 020 maps its `--output` flag value onto a `Format`. No call site spells a key as a bare literal.

### Template set (structural contract)

| Element | Contract |
|---|---|
| Location | Template files live under `internal/render` and are bundled with `//go:embed` (no runtime file read; CONSTITUTION XII self-containment holds). |
| Naming | Exactly one template named `<resource>.<format>` for every `(Resource × Format)` pair — eight built-ins: `me.full`, `me.compact`, `roles.full`, `roles.compact`, `actions.full`, `actions.compact`, `projects.full`, `projects.compact`. |
| Parsing | Parsed once into a single `text/template` set with `Option("missingkey=error")` and the package `FuncMap`. `text/template`, not `html/template` — CLI output is plain text, no HTML auto-escaping. |
| `FuncMap` | Provides only the helpers the data-fidelity rules need that template syntax can't express inline (e.g. nested-collection count, empty-set detection). Exact helper set is implementation detail; helpers are pure and token-free. |

### Error type — `RenderError`

| Type | Shape | Returned when |
|---|---|---|
| `*RenderError` | `{ Resource Resource; Format Format; Err error }` | The named template is absent (unknown resource/format key) **or** template execution fails (missing key under `missingkey=error`, a `FuncMap` helper error, a type mismatch). `errors.As`-discriminable; `Err` wraps the underlying `text/template` cause. Carries **no token** and no request data — only the keys and the engine's error. A built-in `*RenderError` is a code defect, so the consuming read maps it to `RuntimeError(1)` (ADR-4), like 011's undecodable-body. |

**Example (shapes, not literal output)**:
```
ok:       render.Render(render.ResourceMe, render.FormatFull, meResp)
            → ("actor:        …\norganization: …\naccess:       …\n", nil)
compact:  render.Render(render.ResourceRoles, render.FormatCompact, rolesResp)
            → ("<one line per role …>\n", nil)         // built + tested; no CLI path selects it until 020
empty:    render.Render(render.ResourceProjects, render.FormatFull, emptyProjectsResp)
            → ("no projects\n", nil)                    // explicit empty line, both formats
bad key:  render.Render(render.ResourceMe, render.Format("verbose"), meResp)
            → ("", &render.RenderError{Resource:"me", Format:"verbose", Err:…})
```

---

## Interactions

**Render flow**: the caller passes a resource key, a format, and the decoded result. `Render` looks up `<resource>.<format>` in the parsed set, executes it into a `bytes.Buffer`, and returns `buf.String()` on success or `("", *RenderError)` on failure. Nothing is written to any `io.Writer` by the package — the **caller** writes the returned string to stdout only when `err == nil` (buffer-then-write, ADR-4), so a render failure never leaves partial bytes on stdout.

**Standing vs. selectable formats (Q1 resolution)**: the four reads call `Render` with `FormatFull` only — `full` is the standing CLI output, byte-equivalent to each read's pre-019 projection. `FormatCompact` is fully built and unit-tested but reached from **no** operator surface until 020 wires `--output`; `internal/render` exposes both constants so 020 can select either without a package change.

**Data-fidelity guards (CONSTITUTION — present only API data)**: realized in template text + the `FuncMap`, not by a view-model layer (ADR-3):
- **ids always present** — the template writes the id line unconditionally.
- **absent/blank field** — `{{if .X}}…{{else}}<marker>{{end}}` renders the landed explicit-absence marker (`—`, `(none)`, `(no purpose set)`, `(no role)`), except `me`'s empty roles section, which is omitted as its landed projection does; a missing key never renders as empty / `<no value>`, and no *data value* the API didn't return is invented.
- **explicit empty set** — when the result's record collection is empty, the template emits the per-command empty line, inherited verbatim from the landed projections (`No roles.` / `No actions.` / `no projects` — see interface-cli.md), instead of nothing or a fabricated row.
- **compact counts a nested collection** — a `len`-based helper renders `roles=<N>` on the record's line rather than enumerating, which `full` does.
- **never fabricate** — no template emits a literal default for missing data; `missingkey=error` backstops typos by failing loud.

**Extension seam (020 / 029)**: 020 selects a built-in `Format` by name and removes the hardcoded `FormatFull` at the call sites — no `render` change needed. 029 will register a caller-supplied template (parsed into a clone of the built-in set, executed by the same engine) — so no second rendering path is introduced. 019 ships only the two built-ins and exposes no registration API yet.

**Secret hygiene**: `Render` receives only response-side result structs; the token is an `X-Auth-Token` request header, never a result field, so it cannot appear in rendered output (continuing 011). `RenderError` carries only keys and the engine cause.

---

## Error Communication

`Render` communicates failure through its `error` return only; success carries the full rendered string and a `nil` error.

| Condition | Outcome |
|---|---|
| Template found, executes cleanly | `(renderedText, nil)`. Caller writes `renderedText` to stdout. |
| Unknown `resource`/`format` (no `<resource>.<format>` in the set) | `("", *RenderError{resource, format, …})`. A wiring defect. |
| Execution error (missing key under `missingkey=error`, helper error, type mismatch) | `("", *RenderError{resource, format, Err})`. Nothing rendered. |

**Code-free, consumer-maps**: `internal/render` wires **no exit code** and prints nothing (it must not import `internal/cli`). The consuming read writes the string on success, and on a `*RenderError` returns `RuntimeError` through the existing `Outcome`→`ExitCode` registry — **no new exit code and no renumbering** (ADR-4). A render failure is an internal/contract failure (code 1), parallel to 011's `*DecodeError` handling.

**Registry exhaustiveness (build-time/test guard)**: a package test asserts every `(Resource × Format)` pair resolves to a parsed template — a `len`+comma-ok exhaustiveness check (PR #10 LEARNINGS) so a dropped or misnamed template fails loud, not silently at runtime.

---

## Consistency Notes

- **New package, mirroring the established layering**: `internal/render` joins `internal/glassfrog` (schema), `internal/apiclient` (transport), `internal/rcfile` (config-file) as a single-concern internal package. It depends only on `internal/glassfrog` + stdlib and must **not** import `internal/cli`/`internal/apiclient` — the same "lower layers never import `cli`" rule 010 established for `apiclient`.
- **Supersedes the per-command projection-renderer pattern (011 ADR-5)**: 011 paired each read with a pure `format<Read>` Go projection in `internal/cli`. This accord moves rendering into `internal/render` as per-result-type templates; `formatMe`/`formatMeRoles`/`formatMeActions`/`formatMeProjects` are removed and their output becomes each read's `full` template. The `run<Read>` + injected-transport-seam half of 011 ADR-5 is unchanged. (DECISIONS divergence entry recorded; `/score:deprecate` suggested.)
- **Pairs with interface-cli.md**: this file pins the Go API; the rendered output shapes the operator sees (per command, per format, empty/absent behavior) are in `interface-cli.md`. The `full` golden referenced there is the field-equivalence contract this engine must preserve.
- **Specification touchpoint, like its siblings**: a Go-package accord (as 010/016 are), not a runtime surface. No `accords/` directory exists, so there are no cross-spec accord patterns to align against. Invocation and configuration surfaces are **N/A** (library, no command, no settings).
