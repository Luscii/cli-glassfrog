# Interface Accord: Resolution Call-Site Retrofit — Specification

**Feature**: 040-resolution-call-site-retrofit
**Role**: Crafter
**Touchpoint**: Specification (code-API)
**Plan reference**: System Architecture + ADR-1 (surface-stable, map `resolve.Provenance` back), ADR-2 (flag presence threaded), ADR-3 (validate the winner), ADR-4 (per-site OS seam feeds `resolve`)

---

This accord pins the **Go package-API contract** the retrofit changes: the resolver function signatures (which gain a flag-presence input), the `internal/cli` `resolveSelection` seam contract, the `apiclient.AssembleFromOS` signature, and the `resolve.Provenance` → per-domain-type mapping each resolver performs internally. The operator-facing flag-behaviour change is pinned separately in `interface-cli.md`. Consumers are other Go packages, so the "invocation surface" is the exported identifiers and the "configuration surface" is each function's inputs. Signatures are the contract; exact identifier spellings, parameter ordering, and doc-comment wording are the Builder's to finalize within these shapes (as in 039).

**Unchanged surfaces** (the surface-stable guarantee — ADR-1): `auth.Resolution`/`auth.Source`, `apiclient.BaseURL`/`apiclient.BaseURLSource`/`*BaseURLError`, `output.Selection` (the 035 discriminated format-or-template type, which wraps `output.OutputFormat` and carries `*TemplateRef`)/`*FormatError`, and rcfile's typed errors keep their fields, members, and message shapes. `resolve.Provenance` is an internal intermediate, never exposed past the three resolver functions. No new package, no new exported type. **In particular the retrofit must not regress User-Defined Template Output (035): a non-token `--output` value remains a template file path, and `"stdin"` a piped template — only the env/file rungs reject non-tokens.**

---

## Surface

### Token resolver — `internal/auth` (signature UNCHANGED)

```go
// Resolve answers "what token are we acting as, right now, in this directory?"
// Signature unchanged from 005. Internally composes the resolve walk instead of
// the hand-rolled env-then-file chain.
func Resolve() (Resolution, error)
```

Internal composition (no flag rung, no default — the optional-setting shape):

```go
res, err := resolve.Resolve(
    resolve.FromEnv(getenv, envTokenVar),          // GLASSFROG_TOKEN
    resolve.FromFile(startDir, homeDir, tokenKey), // .glassfrogrc "token"
)
```

`startDir`/`homeDir`/`getenv` come from `auth`'s own seam vars (ADR-4), exactly as today.

### Base-URL resolver — `internal/apiclient` (signatures GAIN presence)

```go
// flagPresent is cobra Changed() for --base-url; flagValue is its GetString value.
func ResolveBaseURL(flagValue string, flagPresent bool, startDir, homeDir string) (BaseURL, error)
func ResolveBaseURLFromOS(flagValue string, flagPresent bool) (BaseURL, error)

// Assemble's OS binding threads the same pair through to ResolveBaseURLFromOS.
func AssembleFromOS(flagValue string, flagPresent bool) ConnectionContext
```

Internal composition:

```go
res, err := resolve.Resolve(
    resolve.FromFlags(resolve.Flag{Name: "--" + FlagBaseURL, Present: flagPresent, Value: flagValue}),
    resolve.FromEnv(getenv, EnvVarBaseURL),
    resolve.FromFile(startDir, homeDir, baseURLKey),
    resolve.Default(DefaultBaseURL),
)
```

### Output-selection resolver — `internal/output` (signature GAINS presence)

The output setting resolves to a **`Selection`** (035: a built-in `OutputFormat` *or* a `*TemplateRef` — a template file path / piped `"stdin"`), via `ResolveSelectionFromOS` — **not** the narrower `ResolveFormat`/`OutputFormat`. `ResolveSelectionFromOS` becomes the single composing entry: it composes the *precedence walk* onto `resolve.Resolve` (the flag-rung token-vs-template classification stays). The existing 6-arg pre-fetched-source pure core `ResolveSelection(flagValue, envValue, fileValue, filePath, fileFound, fileErr)` is **folded into `ResolveSelectionFromOS` and removed** — `resolve.Resolve` now fetches env/file via `FromEnv`/`FromFile`, so a pre-fetched-values core no longer fits (mirrors base URL, whose composing core takes dirs and has no pre-fetched variant). The `selection_test.go` cases that drove the 6-arg core move to driving `ResolveSelectionFromOS` over the package `getenv` seam + temp-dir `.glassfrogrc` files (the ADR-4 hermetic harness).

```go
// getenv stays the package seam var, startDir/homeDir are injected (ADR-4 — the
// cli productionSeam.resolveSelection derives and passes them). flagPresent is
// cobra Changed() for --output. Returns the discriminated Selection (035), not a
// bare OutputFormat. The 6-arg ResolveSelection(flag, env, file, path, found, err)
// pure core is folded in here and removed.
func ResolveSelectionFromOS(flagValue string, flagPresent bool, startDir, homeDir string) (Selection, error)
```

Internal composition (single `--output` Flag — cobra merges `-o`, so `Origin` is `--output`, matching today's `*FormatError` label):

```go
res, err := resolve.Resolve(
    resolve.FromFlags(resolve.Flag{Name: "--" + FlagOutput, Present: flagPresent, Value: flagValue}),
    resolve.FromEnv(getenv, EnvVarOutput),
    resolve.FromFile(startDir, homeDir, outputKey),
    resolve.Default(FormatFull.String()), // "full"; classified back to DefaultFormat
)
// then interpret the winner into a Selection (035):
//   - flag-rung winner (KindFlag) → classifyFlagSelection(res.Value): a reserved
//     token → that OutputFormat; "stdin" → TemplateRef{TemplateStdin}; any other
//     non-empty value → TemplateRef{TemplateFile, Path}
//   - env/file/default winner       → ParseFormat(res.Value) → a built-in format,
//     NEVER a template (templates are reachable only from the flag — 035 ADR-1)
```

### `internal/cli` seam contract — `resolveSelection` (GAINS presence)

```go
// Was: resolveSelection(flagValue string) (output.Selection, error)
// Now (all the per-read-command resolveSelection seam declarations
// + the productionSeam impl):
resolveSelection(flagValue string, flagPresent bool) (output.Selection, error)
```

The `productionSeam` impl derives `startDir`/`homeDir` and delegates to `output.ResolveSelectionFromOS(flagValue, flagPresent, startDir, homeDir)`. The companion `readTemplateSource(ref output.TemplateRef) (string, error)` seam method (035) is **unchanged** — the retrofit touches only how the `Selection` is resolved, not how a chosen template is read. Test seams bind a fake over the new `resolveSelection` signature.

### `internal/cli` RunE plumbing (every read command)

Each `RunE` reads both the value and the presence bit for the flags it forwards:

```go
baseURL, _   := cmd.Flags().GetString(apiclient.FlagBaseURL) // GetString returns (string, error)
baseURLSet   := cmd.Flags().Changed(apiclient.FlagBaseURL)    // Changed returns bool
outputFlag, _ := cmd.Flags().GetString(output.FlagOutput)
outputSet    := cmd.Flags().Changed(output.FlagOutput)
// … baseURL/baseURLSet → AssembleFromOS;  outputFlag/outputSet → seam.resolveSelection
```

`Changed()` is correct on these inherited persistent root flags — the same `cmd.Flags()` flagset already serves their `GetString` (ADR-2).

---

## Interactions

### Provenance → per-domain-type mapping (the adapter each resolver performs — ADR-1)

| Setting | `resolve.Provenance.Kind` | Maps to | Notes |
|---|---|---|---|
| Token | `KindEnv` / `KindFile` / `KindNone` | `auth.SourceEnvironment` / `auth.SourceFile` (Path = `Origin`) / `auth.SourceNone` | `Token = res.Value`; no flag, no default |
| Base URL | `KindFlag` / `KindEnv` / `KindFile` / `KindDefault` | `apiclient.SourceFlag` / `SourceEnvironment` / `SourceFile` (Path = `Origin`) / `SourceDefault` | `Value = res.Value` |
| Output | `KindFlag` / `KindEnv` / `KindFile` / `KindDefault` | a `Selection`: flag winner → `classifyFlagSelection` (format **or** `*TemplateRef`); env/file winner → `ParseFormat` (format only); default → `Selection{DefaultFormat}` | the `Source` lives only in `*FormatError`; templates only from the flag (035) |

`Provenance.Origin` supplies the exact existing error/Path labels: `--base-url`/`GLASSFROG_BASE_URL`/file-path and `--output`/`GLASSFROG_OUTPUT`/file-path (039 pinned these labels for this purpose).

### Interpretation at the winner (ADR-3)

After `resolve.Resolve` returns with a **nil** error, the call site interprets/validates `res.Value` **only when `res.Provenance.Kind != KindDefault`** (the default is valid by construction):

- Base URL: `if Kind != KindDefault && !isUsableURL(res.Value) { return &BaseURLError{Source: res.Provenance.Origin} }`
- Output: classify the winner into a `Selection`. The **flag** winner (`KindFlag`) goes through `classifyFlagSelection` — a reserved token → that format, `"stdin"` → a stdin template, any other non-empty value → a template **file path** (035), so a non-token flag value is *not* an error. The **env/file** winner (`KindEnv`/`KindFile`) goes through `ParseFormat` and a non-token fails loud with `&FormatError{Source: res.Provenance.Origin, Value: res.Value}` (templates are flag-only).

Because `resolve` returns the first **yielding** (not first **valid**) source, an invalid high-precedence value still wins and is rejected/interpreted here — never silently superseded by a lower source (the present-but-invalid-fails-loud guarantee, no fall-through).

### Yield semantics at the flag rung (the behaviour change — ADR-2)

`resolve.FromFlags` yields whenever `Present` is true, **even with an empty/whitespace `Value`**. So a supplied `--base-url`/`--output` always wins its rung; an unsupplied flag (`Present == false`) does not yield and the walk falls through to env. Env/file rungs keep their non-empty-after-trim yield rule (`FromEnv`/`FromFile`), so a whitespace-only env/file value still falls through (unchanged). For `--base-url` the supplied empty value is validated and fails loud; for `--output` the supplied empty value reaches `classifyFlagSelection` (035) and resolves to a degenerate empty selection that fails loud through the selection/template path rather than falling through — the precise empty-`--output` outcome is a clarify-level follow-up for the implementer, not pinned here.

---

## Error Communication

| Condition | Resolver behaviour |
|---|---|
| `resolve.Resolve` returns a typed `rcfile` error (`*ReadError`/`*FormatError`) | returned **verbatim** before any value validation; no fall-through (unchanged from today) |
| Base URL winner present but invalid | `*BaseURLError` naming `res.Provenance.Origin`; no fall-through (ADR-3) |
| Output env/file winner is a non-token | `*FormatError` naming `res.Provenance.Origin`; no fall-through (a flag-rung non-token is a template, not an error — 035) |
| Token: nothing yields | `auth.Resolution{Source: SourceNone}`, **nil error** (unchanged) |
| Base URL / output: nothing yields | the `Default` rung wins; `SourceDefault` / `Selection{DefaultFormat}`, nil error (unchanged) |
| `auth.Resolve` token mapping | `res.Value` → `Resolution.Token`; the `resolve.Resolution` is **never** formatted/logged (secret hygiene — `resolve.Resolution` has no redacting `String()`; `auth.Resolution` does) |

The two distinct error classes (resolution-error vs. value-invalidity) and their exit-code mapping at the `internal/cli` boundary are unchanged — `classifyClientError` keeps its existing `*BaseURLError`/`*FormatError` → `UsageError(2)` arms. No new `Outcome`, no new exit code.

---

## Consistency Notes

- **Pairs with `interface-cli.md`**: that file pins the operator-facing flag-semantics change; this file pins the Go contracts behind it. The two are the same change seen from the two sides of the seam.
- **Consumes 039 (`interface-spec.md`)**: uses `resolve.Resolve`/`FromFlags`/`FromEnv`/`FromFile`/`Default` and the `resolve.Flag{Name,Present,Value}` shape exactly as 039 specified; this slice adds the call-site adapters 039 deferred to 040.
- **Surface-stable (ADR-1)** — diverges from 039 ADR-2's forecast that 040 removes the per-domain `Source` enums. They are kept and mapped onto; the enum-unification is deferred (DECISIONS 2026-06-12; a `/score:deprecate` of the forecast is suggested).
- **Per-site OS seam (ADR-4 / DECISIONS §49/§71)**: each `…FromOS` entrypoint sources the real dirs/env exactly as today and feeds it into `resolve`'s injected constructors; `resolve`'s own `…FromOS` helpers are unused here. The seam shape differs: `auth` and `apiclient` own `getenv`/`getwd`/`userHomeDir` and derive their own dirs, while `internal/output` owns only `getenv` and `ResolveSelectionFromOS` *receives* `startDir`/`homeDir` from the cli `productionSeam.resolveSelection`. Tests stay hermetic (temp dirs + the package `getenv` var), never the real `~/.glassfrogrc`.
- **Consumes 035 (User-Defined Template Output)**: the output setting resolves through `output.Selection`/`ResolveSelection`/`resolveSelection` (035 widened 020), so the retrofit composes the precedence walk inside `ResolveSelection` and preserves its flag-rung token-vs-template classification and the `readTemplateSource` seam — it does not revert to the bare `OutputFormat`/`ResolveFormat` surface.
- **Compiler-enforced threading**: the presence-bearing signatures change with **no** default-value overload, so every un-threaded call site fails to compile — the safeguard against a missed `RunE` silently keeping value-emptiness behaviour (plan Risks).
- **No `accords/` directory** exists, so there are no cross-spec package-API accord patterns to align against.
