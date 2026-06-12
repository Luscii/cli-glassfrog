# Interface Accord: Resolution Call-Site Retrofit — Specification

**Feature**: 040-resolution-call-site-retrofit
**Role**: Crafter
**Touchpoint**: Specification (code-API)
**Plan reference**: System Architecture + ADR-1 (surface-stable, map `resolve.Provenance` back), ADR-2 (flag presence threaded), ADR-3 (validate the winner), ADR-4 (per-site OS seam feeds `resolve`)

---

This accord pins the **Go package-API contract** the retrofit changes: the resolver function signatures (which gain a flag-presence input), the `internal/cli` `meSeam.resolveFormat` contract, the `apiclient.AssembleFromOS` signature, and the `resolve.Provenance` → per-domain-type mapping each resolver performs internally. The operator-facing flag-behaviour change is pinned separately in `interface-cli.md`. Consumers are other Go packages, so the "invocation surface" is the exported identifiers and the "configuration surface" is each function's inputs. Signatures are the contract; exact identifier spellings, parameter ordering, and doc-comment wording are the Builder's to finalize within these shapes (as in 039).

**Unchanged surfaces** (the surface-stable guarantee — ADR-1): `auth.Resolution`/`auth.Source`, `apiclient.BaseURL`/`apiclient.BaseURLSource`/`*BaseURLError`, `output.OutputFormat`/`*FormatError`, and rcfile's typed errors keep their fields, members, and message shapes. `resolve.Provenance` is an internal intermediate, never exposed past the three resolver functions. No new package, no new exported type.

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

### Output-format resolver — `internal/output` (signature GAINS presence; pure-core folded)

```go
// The 6-arg pre-fetched-source ResolveFormat core is FOLDED into one composing
// function; getenv stays the package seam var, startDir/homeDir are injected
// (ADR-4). flagPresent is cobra Changed() for --output.
func ResolveFormatFromOS(flagValue string, flagPresent bool, startDir, homeDir string) (OutputFormat, error)
```

Internal composition (single `--output` Flag — cobra merges `-o`, so `Origin` is `--output`, matching today's `*FormatError` label):

```go
res, err := resolve.Resolve(
    resolve.FromFlags(resolve.Flag{Name: "--" + FlagOutput, Present: flagPresent, Value: flagValue}),
    resolve.FromEnv(getenv, EnvVarOutput),
    resolve.FromFile(startDir, homeDir, outputKey),
    resolve.Default(FormatFull.String()), // "full"; parsed back, or short-circuited as DefaultFormat
)
```

### `internal/cli` seam contract — `meSeam.resolveFormat` (GAINS presence)

```go
// Was: resolveFormat(flagValue string) (output.OutputFormat, error)
// Now (all 10 seam declarations + the productionSeam impl):
resolveFormat(flagValue string, flagPresent bool) (output.OutputFormat, error)
```

The `productionSeam` impl derives `startDir`/`homeDir` and delegates to `output.ResolveFormatFromOS(flagValue, flagPresent, startDir, homeDir)`. Test seams bind a fake over the new signature.

### `internal/cli` RunE plumbing (12 read commands)

Each `RunE` reads both the value and the presence bit for the flags it forwards:

```go
baseURL      := cmd.Flags().GetString(apiclient.FlagBaseURL)
baseURLSet   := cmd.Flags().Changed(apiclient.FlagBaseURL)
outputFlag   := cmd.Flags().GetString(output.FlagOutput)
outputSet    := cmd.Flags().Changed(output.FlagOutput)
// … baseURL/baseURLSet → AssembleFromOS;  outputFlag/outputSet → seam.resolveFormat
```

`Changed()` is correct on these inherited persistent root flags — the same `cmd.Flags()` flagset already serves their `GetString` (ADR-2).

---

## Interactions

### Provenance → per-domain-type mapping (the adapter each resolver performs — ADR-1)

| Setting | `resolve.Provenance.Kind` | Maps to | Notes |
|---|---|---|---|
| Token | `KindEnv` / `KindFile` / `KindNone` | `auth.SourceEnvironment` / `auth.SourceFile` (Path = `Origin`) / `auth.SourceNone` | `Token = res.Value`; no flag, no default |
| Base URL | `KindFlag` / `KindEnv` / `KindFile` / `KindDefault` | `apiclient.SourceFlag` / `SourceEnvironment` / `SourceFile` (Path = `Origin`) / `SourceDefault` | `Value = res.Value` |
| Output | `KindFlag` / `KindEnv` / `KindFile` / `KindDefault` | the `OutputFormat` from `ParseFormat(res.Value)`; default → `DefaultFormat` | the `Source` lives only in `*FormatError` |

`Provenance.Origin` supplies the exact existing error/Path labels: `--base-url`/`GLASSFROG_BASE_URL`/file-path and `--output`/`GLASSFROG_OUTPUT`/file-path (039 pinned these labels for this purpose).

### Validation at the winner (ADR-3)

After `resolve.Resolve` returns with a **nil** error, the call site validates `res.Value` **only when `res.Provenance.Kind != KindDefault`** (the default is valid by construction):

- Base URL: `if Kind != KindDefault && !isUsableURL(res.Value) { return &BaseURLError{Source: res.Provenance.Origin} }`
- Output: `if Kind != KindDefault { f, err := ParseFormat(res.Value); if err != nil { return &FormatError{Source: res.Provenance.Origin, Value: res.Value} } }`

Because `resolve` returns the first **yielding** (not first **valid**) source, an invalid high-precedence value still wins and is rejected here — never silently superseded by a lower source (the present-but-invalid-fails-loud guarantee, no fall-through).

### Yield semantics at the flag rung (the behaviour change — ADR-2)

`resolve.FromFlags` yields whenever `Present` is true, **even with an empty/whitespace `Value`**. So a supplied `--base-url`/`--output` always wins its rung and is validated (failing loud on empty/whitespace); an unsupplied flag (`Present == false`) does not yield and the walk falls through to env. Env/file rungs keep their non-empty-after-trim yield rule (`FromEnv`/`FromFile`), so a whitespace-only env/file value still falls through (unchanged).

---

## Error Communication

| Condition | Resolver behaviour |
|---|---|
| `resolve.Resolve` returns a typed `rcfile` error (`*ReadError`/`*FormatError`) | returned **verbatim** before any value validation; no fall-through (unchanged from today) |
| Winner is present but invalid (base URL / output) | `*BaseURLError` / `*FormatError` naming `res.Provenance.Origin`; no fall-through (ADR-3) |
| Token: nothing yields | `auth.Resolution{Source: SourceNone}`, **nil error** (unchanged) |
| Base URL / output: nothing yields | the `Default` rung wins; `SourceDefault` / `DefaultFormat`, nil error (unchanged) |
| `auth.Resolve` token mapping | `res.Value` → `Resolution.Token`; the `resolve.Resolution` is **never** formatted/logged (secret hygiene — `resolve.Resolution` has no redacting `String()`; `auth.Resolution` does) |

The two distinct error classes (resolution-error vs. value-invalidity) and their exit-code mapping at the `internal/cli` boundary are unchanged — `classifyClientError` keeps its existing `*BaseURLError`/`*FormatError` → `UsageError(2)` arms. No new `Outcome`, no new exit code.

---

## Consistency Notes

- **Pairs with `interface-cli.md`**: that file pins the operator-facing flag-semantics change; this file pins the Go contracts behind it. The two are the same change seen from the two sides of the seam.
- **Consumes 039 (`interface-spec.md`)**: uses `resolve.Resolve`/`FromFlags`/`FromEnv`/`FromFile`/`Default` and the `resolve.Flag{Name,Present,Value}` shape exactly as 039 specified; this slice adds the call-site adapters 039 deferred to 040.
- **Surface-stable (ADR-1)** — diverges from 039 ADR-2's forecast that 040 removes the per-domain `Source` enums. They are kept and mapped onto; the enum-unification is deferred (DECISIONS 2026-06-12; a `/score:deprecate` of the forecast is suggested).
- **Per-site OS seam (ADR-4 / DECISIONS §49/§71)**: each `…FromOS` entrypoint keeps its own `getenv`/`getwd`/`userHomeDir` seam and feeds it into `resolve`'s injected constructors; `resolve`'s own `…FromOS` helpers are unused here. Tests stay hermetic (temp dirs + the package `getenv` var), never the real `~/.glassfrogrc`.
- **Compiler-enforced threading**: the presence-bearing signatures change with **no** default-value overload, so every un-threaded call site fails to compile — the safeguard against a missed `RunE` silently keeping value-emptiness behaviour (plan Risks).
- **No `accords/` directory** exists, so there are no cross-spec package-API accord patterns to align against.
