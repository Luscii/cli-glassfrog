# Interface Accord: Output Format Selection — Specification

**Feature**: 020-output-format-selection
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (selection vocabulary + resolver in `internal/output`, mirroring `apiclient.baseurl`), ADR-3 (`internal/cli` dispatch; `output` and `render` non-importing siblings), ADR-4 (`*FormatError` → `UsageError(2)`, resolution independent of connection assembly).

---

This accord pins two Go contracts: the **selection vocabulary + precedence resolver** 020 adds to the existing `internal/output` package (created by 018), and the **render-dispatch seam** in `internal/cli` that routes a command's result to the renderer the resolved format names. 020 owns selection and success dispatch; it consumes 018's encoders and 019's templates unchanged. Concrete Go symbol/field spellings are a build detail; the **shapes, signatures, format identifiers, precedence semantics, and the literal flag/env/key names** are the contract.

---

## Surface

### `internal/output` — selection vocabulary + resolver (NEW symbols added to 018's package)

| Symbol | Signature (shape) | Description |
|---|---|---|
| `OutputFormat` | enum: `Full`, `Compact`, `JSON`, `YAML` | The four-valued *selection* vocabulary. Distinct from 018's `Format{JSON,YAML}` (the machine-encoder enum); `OutputFormat` is what flag/env/config resolve to. |
| `DefaultFormat` | `OutputFormat` = `Full` | The built-in default when no source selects a format (preserves 019's standing projection). |
| `ParseFormat` | `(s string) (OutputFormat, error)` | Case-insensitive over exactly `full`/`compact`/`json`/`yaml`. Returns a non-nil error (used to build `*FormatError`) for any other non-empty value, in any casing. |
| `(OutputFormat).IsStructured` | `() bool` | `true` for `JSON`/`YAML` (the `internal/output` branch), `false` for `Full`/`Compact` (the `internal/render` branch). |
| `(OutputFormat).MachineFormat` | `() (Format, bool)` | Maps a structured `OutputFormat` to 018's `Format` (`JSON`→`Format.JSON`, `YAML`→`Format.YAML`); the bool is `false` for the human formats. Keeps the json/yaml mapping inside `internal/output`; the human→`render` mapping lives in `cli` (ADR-3). |
| `ResolveFormat` | pure core over injected sources → `(OutputFormat, error)` | Resolves the precedence chain (see Interactions). Returns `DefaultFormat` when every source is absent; `*FormatError` when a source is present-but-invalid; an `internal/rcfile` read/format error when the config file can't be read while resolving the `output` key. |
| `ResolveFormatFromOS` | `(flagValue, startDir, homeDir string) (OutputFormat, error)` | Thin production seam binding the real `os.Getenv` and the `internal/rcfile` walk to `ResolveFormat`'s injected sources (the 005 inject-roots split). Tests call `ResolveFormat` with hand-built sources; no suite reads the real `~/.glassfrogrc`. |
| `FormatError` | struct: `Source string`, `Value string`; `Error() string` | The present-but-invalid-value error, naming the source (`--output`, `GLASSFROG_OUTPUT`, or the `.glassfrogrc` path) and the offending value. Mapped to `UsageError` by `cli` (Error Communication). |

**Constants** (the literal names the spec marked `[ASSUMED]`, pinned here — mirroring `apiclient.FlagBaseURL`/`EnvVarBaseURL`/`baseURLKey`):

| Constant (shape) | Value |
|---|---|
| `FlagOutput` | `"output"` (the `--output` long name; `-o` short alias is registered at the flag site) |
| `EnvVarOutput` | `"GLASSFROG_OUTPUT"` |
| `outputKey` (unexported) | `"output"` — the `.glassfrogrc` key, the third key after `token` (005) and `base_url` (008) |
| source labels | `"--output"` / `"GLASSFROG_OUTPUT"` / the resolved file path — carried in `FormatError.Source` |

### `internal/cli` — render dispatch (NEW)

| Symbol | Signature (shape) | Description |
|---|---|---|
| render-dispatch | generic over the result type, e.g. `renderResult[T any](w io.Writer, f OutputFormat, resourceKey render.Resource, exec Executor, reqCtx, req) (Outcome, error)` | Selects the decode target and the success renderer from `f`: structured → decode `*json.RawMessage`, write `output.RenderSuccess(machineFmt, raw)`; human → decode `*T`, write `render.Render(resourceKey, humanFmt, v)`. The single site that imports **both** `internal/output` and `internal/render` and maps `OutputFormat`'s human values to `render`'s `FormatFull`/`FormatCompact`. The four reads delegate to it. |
| `classifyClientError` arm | (extends the existing function) | Adds `*output.FormatError` → `UsageError`, symmetric with the existing base-URL arms, so an invalid selector's category and message agree. |

### Consumed unchanged (not defined here)

- **From `018/interface-spec.md`**: `output.Format{JSON,YAML}`, `RenderSuccess(Format, json.RawMessage) ([]byte, error)`, `RenderError(Format, ErrorEnvelope) ([]byte, error)`, `ErrorEnvelope`/`ErrorDetail`. 020 calls `RenderSuccess` for the structured success path; `RenderError` is reserved for 032.
- **From `019/interface-spec.md`**: `render.Render(resource, format, data) (string, error)`, `render.FormatFull`/`render.FormatCompact`, the `render.Resource` keys (`me`/`roles`/`actions`/`projects`), and the buffer-then-write / render-error→`RuntimeError(1)` contract.
- **From `008` (`internal/apiclient/baseurl.go`)**: the resolution shape `ResolveFormat` mirrors — flag → env → `.glassfrogrc` key → default, present-but-invalid errors naming its source — and `internal/rcfile`'s shared reader + nearest-wins walk that reads the `output` key.
- **From `010/011`**: `(*Client).Execute(reqCtx, req, out any)` accepting a `*json.RawMessage` (structured) or a typed `*glassfrog` struct (human) as `out`; `classifyClientError(err) Outcome` and the `Outcome`→`ExitCode` registry (004).

**Example (shapes, not literal values)**:
```
// in a read command's RunE:
fv, _ := cmd.Flags().GetString(output.FlagOutput)          // inherited persistent flag (-o)
fmtSel, err := output.ResolveFormatFromOS(fv, cwd, home)   // flag→GLASSFROG_OUTPUT→.glassfrogrc output→Full
if err != nil { report(stderr, err); return UsageError }   // *FormatError | rcfile err → UsageError(2), no request

// dispatch (internal/cli), after building the executor:
if mf, ok := fmtSel.MachineFormat(); ok {                  // json | yaml
    var raw json.RawMessage
    if _, e := exec.Execute(reqCtx, req, &raw); e != nil { return reportClientError(stderr, e) }
    doc, rerr := output.RenderSuccess(mf, raw)             // 018
    // rerr → RuntimeError(1); else write doc to stdout
} else {                                                    // full | compact
    var v glassfrog.MeResponse
    if _, e := exec.Execute(reqCtx, req, &v); e != nil { return reportClientError(stderr, e) }
    s, rerr := render.Render(render.ResourceMe, humanFormat(fmtSel), v)  // 019
    // rerr → RuntimeError(1); else write s to stdout
}
```

---

## Interactions

- **Precedence (ADR-1)**: `ResolveFormat` consults `--output` flag → `GLASSFROG_OUTPUT` → `.glassfrocrc` `output` key (nearest-wins walk via `internal/rcfile`) → `DefaultFormat` (`Full`). The first source yielding a value wins; an **absent** source is skipped, a **present-but-invalid** source produces `*FormatError` (no fall-through). Always yields a format.
- **Resolve before request (ADR-4 / 018 ADR-2)**: the command resolves the format first — the decode target depends on it. On any resolution error the command reports to stderr and returns `UsageError` before assembling the connection or sending, so no doomed request is made (the `validateInclude` fail-fast shape).
- **Decode-target selection**: `IsStructured()` chooses `*json.RawMessage` (verbatim 2xx capture, 018 ADR-2) vs the typed `*glassfrog` struct (the human projection input, 019). The choice is `cli`'s; `output.RenderSuccess` and `render.Render` each only encode what they are handed.
- **Package layering (ADR-3)**: `internal/output` (machine + selection vocabulary) and `internal/render` (human templates) do not import each other; `internal/cli` is the only importer of both and owns the `OutputFormat`→`render.Format` mapping. `internal/output` imports `internal/rcfile` (leaf→leaf) for the config rung.
- **Independent of connection assembly (009)**: output-format resolution is not part of `AssembleFromOS`; it is a presentation concern resolved on the render path. A base-URL error and a format error are independent usage-class failures.

## Error Communication

| Failure | Origin | `cli` mapping | Exit |
|---|---|---|---|
| Present-but-invalid value at any source | `*output.FormatError{Source,Value}` | `classifyClientError` → `UsageError` | `2` |
| `.glassfrogrc` unreadable / unparseable while reading `output` | `*rcfile.ReadError` / `*rcfile.FormatError` | existing base-URL-symmetric arms → `UsageError` | `2` |
| Structured/human render fails | `output.RenderSuccess` / `render.Render` error | `RuntimeError` (buffer-then-write; nothing partial on stdout) | `1` |
| Command failure (transport/API/auth/decode) | existing typed errors | unchanged (`reportClientError`) — **not** routed through `output.RenderError` by 020 (032's scope) | 1/3/4/5/6 |

- `*FormatError` messages are token-free (they carry only a source label and the offending format value) and name a corrective next step consistent with `--base-url`'s wording.
- The secret never appears in any rendered output or resolution error: 020 routes only response-side result data and the format value; the `output` rcfile key is read through the shared reader that never returns `token` to non-secret callers (008's rule).

## Consistency Notes

- **Extends `internal/output` (018), sibling to `internal/render` (019)**: claims the selector slot 018 reserved; the two renderers stay non-importing leaves, `cli` bridges them. The structured path calls 018's `RenderSuccess`; the human path calls 019's `Render` — neither contract changes.
- **Resolver mirrors `apiclient.baseurl` (008)**: same chain shape, same present-but-invalid-errors-naming-source rule, same `internal/rcfile` shared reader and nearest-wins walk, same pure-core-plus-OS-seam split (005). The `output` key is the third `.glassfrogrc` key.
- **Pairs with interface-cli.md**: that file pins the operator-facing flag, values, precedence, and per-format stdout; this file pins the Go API that realizes it.
- **`RenderError` (018) is intentionally unused here**: 020 does not render command failures in the active format — that is 032. The symbol is consumed-unchanged context, not part of 020's surface, so the json/yaml error path stays today's stderr text until 032.
- **No `accords/` directory** exists; the only cross-spec contracts are the sibling interface-spec files (008/011/018/019), referenced above.
