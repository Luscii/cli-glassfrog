# Interface Accord: Source-Composed Resolution — Specification

**Feature**: 039-source-composed-resolution
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/2/3/4/5 — the `internal/resolve` package API (a code-API boundary; the package's exported surface is the interface)

---

The artifact is a Go package, `internal/resolve`. Its consumers are other Go packages (the 040 call sites), so the "invocation surface" is the package's exported identifiers, the "configuration surface" is the inputs each constructor accepts, and "constraint violations" are the panic and error contracts. Signatures are the contract the Builder implements and the Verifier tests against; exact field ordering and doc-comment wording are implementation detail.

---

## Surface

### Package

`internal/resolve` — imports `internal/rcfile` (file source), `golang.org/x/term` (the `StdinFromOS` TTY check — already in `go.mod`, used by `authlogin_seam.go`), and the standard library; imports no domain package (`auth` / `apiclient` / `output`). The Phase-1 core (types, `Resolve`, value-only sources) is standard-library-only; only the Phase-2 `StdinFromOS` binding pulls `golang.org/x/term`.

### Types

**`SourceKind`** — classifies where a resolved value originated. Also the `Provenance.Kind` vocabulary.

| Member | Meaning |
|---|---|
| `KindNone` | nothing yielded and no default was present (the zero value) |
| `KindFlag` | a command flag supplied the value |
| `KindEnv` | an environment variable supplied the value |
| `KindFile` | a `.glassfrogrc` key supplied the value |
| `KindStdin` | piped standard input supplied the value |
| `KindDefault` | the trailing default supplied the value |

`func (k SourceKind) String() string` — lowercase token (`none`/`flag`/`env`/`file`/`stdin`/`default`), for messages and tests.

**`Provenance`** — where a resolved value came from. All fields are safe to display (never a secret).

| Field | Type | Description |
|---|---|---|
| `Kind` | `SourceKind` | which source kind won |
| `Origin` | `string` | the concrete origin label: the flag name (e.g. `--output`), the env var name (e.g. `GLASSFROG_BASE_URL`), or the resolved file path. Empty for `KindNone`, `KindDefault`, and `KindStdin`. |

**`Resolution`** — the code-free output of `Resolve`.

| Field | Type | Description |
|---|---|---|
| `Value` | `string` | the resolved raw value, verbatim; meaningful only when the returned error is nil and `Found()` is true |
| `Provenance` | `Provenance` | where `Value` came from |

`func (r Resolution) Found() bool` — reports `r.Provenance.Kind != KindNone` (any source, including the default, yielded).

**`Source`** — one origin in the precedence list. Opaque to callers; constructed only via the constructors below. `func (s Source) Kind() SourceKind` exposes its kind (used by `Resolve` for the STDIN guard; available to callers for assertions).

**`Flag`** — one flag input for `FromFlags`.

| Field | Type | Description |
|---|---|---|
| `Name` | `string` | operator-facing label including dashes (e.g. `--output`, `-o`); becomes `Provenance.Origin` when this flag wins |
| `Present` | `bool` | whether the flag was supplied on the command line (cobra `Changed()`) |
| `Value` | `string` | the flag's value |

### Functions

```go
// Resolve walks sources in order (index 0 = highest precedence) and returns the
// first that yields. Evaluation is lazy: once a source yields, no lower source is
// evaluated. A source that errs aborts the walk and returns that error verbatim
// (no fall-through). When no source yields and no Default is present, returns
// Resolution{Provenance: {Kind: KindNone}} with a nil error.
// PANICS if more than one Stdin source is supplied (a composition/wiring bug).
func Resolve(sources ...Source) (Resolution, error)

// FromFlags yields the value of the first Present flag, in argument order. A
// Present flag yields even when its Value is empty (presence-based, per cobra
// Changed()). Provenance.Origin is that flag's Name.
func FromFlags(flags ...Flag) Source

// FromEnv yields the first non-empty value among names, looked up via lookup, in
// argument order. Provenance.Origin is the name that yielded. A value that is
// empty after trimming does not yield.
func FromEnv(lookup func(string) string, names ...string) Source

// FromFile yields the value for key from the nearest .glassfrogrc up the tree,
// via rcfile.Resolve(startDir, homeDir, key). Provenance.Origin is the resolved
// file path. An unreadable/unparseable file errs (verbatim rcfile typed error);
// a missing or key-less file does not yield.
func FromFile(startDir, homeDir, key string) Source

// FromStdin yields trimmed piped input when isTTY is false and the content is
// non-empty after trimming. On a terminal (isTTY true) it never reads and does
// not yield. The read is bounded (maxStdinBytes): input that exceeds the bound
// errs (no silent truncation), and a read failure errs. Provenance.Origin is empty.
func FromStdin(read func() (string, error), isTTY bool) Source

// Default always yields value, with Provenance{Kind: KindDefault}. Place it last.
func Default(value string) Source
```

### OS binding (thin convenience)

Production callers bind the real OS globals; these are thin sugar over the pure constructors (the OS seam stays injected — ADR-4). Tests use the pure constructors directly and never touch these.

```go
// OSRoots returns the directories the file walk searches: the working directory
// (an error if it cannot be determined) and the home directory ("" if it cannot
// be determined — the home fallback is dropped, never a hard failure). Mirrors
// the getwd/userHomeDir preamble the existing *FromOS resolvers share.
func OSRoots() (startDir, homeDir string, err error)

// EnvFromOS is FromEnv(os.Getenv, names...).
func EnvFromOS(names ...string) Source

// StdinFromOS is FromStdin bound to a bounded os.Stdin reader and
// term.IsTerminal(os.Stdin.Fd()).
func StdinFromOS() Source
```

`FromFlags` needs no OS binding (the caller supplies cobra flag state); `Default` needs none. The `maxStdinBytes` cap is a package constant (value `[ASSUMED]`, tuning deferred like 006's `maxPipedTokenBytes`).

### Example

```go
// A base-URL-style composition (flag → env → file → default). The setting's
// names/keys/default come from the caller, not from internal/resolve.
startDir, homeDir, err := resolve.OSRoots()
// ... handle err ...
res, err := resolve.Resolve(
    resolve.FromFlags(resolve.Flag{Name: "--base-url", Present: flagChanged, Value: flagValue}),
    resolve.EnvFromOS("GLASSFROG_BASE_URL"),
    resolve.FromFile(startDir, homeDir, "base_url"),
    resolve.Default("https://app.glassfrog.com/api/v5"),
)
// err != nil  → an unreadable/unparseable .glassfrogrc (verbatim rcfile error)
// res.Value   → the winning raw value (NOT yet validated — caller validates)
// res.Provenance.Origin → "--base-url" | "GLASSFROG_BASE_URL" | "<path>" | "" (default)
```

---

## Interactions

**Precedence by position**: the argument order to `Resolve` is the precedence — index 0 highest. A trailing `Default(...)` is the backstop; omit it for an optional setting (the token shape), where a `KindNone` result is the normal "nothing found" outcome.

**List-walking within a source**: `FromFlags` and `FromEnv` accept several inputs and walk them in order, yielding from the first that does; the winner's identity is reported in `Provenance.Origin` (e.g. `FromFlags(Flag{Name:"--output"...}, Flag{Name:"-o"...})` reports whichever alias was present).

**Yield rules** (what counts as "yields"):
- `FromFlags` — a `Present` flag yields, *even with an empty `Value`* (presence-based).
- `FromEnv` / `FromFile` / `FromStdin` — yield only on a present, non-empty (post-trim) value.
- `Default` — always yields.

**Validation is the caller's** (ADR-3): `Resolve` returns the raw winning `Value` and runs no validator. The caller validates `Value` (URL shape, format enum, …) and, on failure, phrases the error from `Provenance.Origin`. Because resolution is first-*non-empty* (not first-*valid*), an invalid high-precedence value still wins and is rejected by the caller — never silently superseded by a lower source (preserving the present-but-invalid-fails-loud behavior).

**Determinism**: same inputs → same result. No source prompts or blocks; `FromStdin` reads already-piped input only and never waits on a TTY.

---

## Error Communication

| Condition | Behavior |
|---|---|
| No source yields, no `Default` | `Resolution{Provenance:{Kind:KindNone}}`, **nil error** — a valid empty outcome, not a failure |
| `FromFile` hits an unreadable/unparseable `.glassfrogrc` | `Resolve` returns the **verbatim** `rcfile` typed error (`*rcfile.ReadError` / `*rcfile.FormatError`, naming the path); the walk aborts, no fall-through to a lower source |
| `FromStdin` read fails | `Resolve` returns the read error; the walk aborts |
| `FromStdin` input exceeds `maxStdinBytes` | `FromStdin` errs (no silent truncation); `Resolve` returns the error and the walk aborts — satisfies Constitution VI (never silently truncate) |
| More than one `Stdin` source passed to `Resolve` | **panic** (`resolve.Resolve: at most one Stdin source per resolution`) — a wiring bug, consistent with the nil-seam fail-fast convention (PR #20) |
| A value is present but invalid for the setting | **not** an error here — `Resolve` returns it as the winner; the caller validates and reports |

**Uniform handling**: any non-nil `error` from `Resolve` means resolution failed at a source — callers treat it uniformly (abort, map to an exit code) while remaining free to `errors.As` the specific typed error for a tailored message. The resolver itself emits no diagnostics and never formats `Value` into a message (secret hygiene for the token setting).

---

## Consistency Notes

- **Sibling interfaces**: none — this feature has only a specification touchpoint (no API/CLI/UI/events surface of its own; it is consumed in-process by Go packages).
- **`internal/rcfile`**: `FromFile` delegates the nearest-wins walk to `rcfile.Resolve(startDir, homeDir, key)` (DECISIONS §74 — one shared reader, never a parallel parser) and surfaces its typed errors verbatim. `internal/resolve` adds no second parser and owns no `.glassfrogrc` format knowledge.
- **Code-free outcome + Source enum**: `Resolution`/`Provenance` continue the established "producer classifies a code-free outcome, consumer maps it" split (DECISIONS §46/§62/§161); `Provenance` is the *unified* successor to the three divergent per-domain `Source` enums (`auth.Source`, `apiclient.BaseURLSource`, and `output`'s implicit source). Their removal and the call-site adoption are **040 (Resolution Call-Site Retrofit)**, not this slice.
- **Injected seam**: the pure constructors take injected access and the thin `…FromOS`/`OSRoots`/`EnvFromOS`/`StdinFromOS` helpers bind reality — conforming to DECISIONS §49 (inject roots) and §71 (`isTTY` + bounded stdin read), matching `authlogin_seam.go`. No package-global mutable seam is introduced.
- **No `--token` flag**: consistent with auth being env/file-only (005) — a token composition simply omits a `FromFlags` source.
- **Interface-level, not frozen**: exact identifier spellings (`Resolve`, `From*`, `Default`, `Flag`, `SourceKind` members), field ordering, and the `maxStdinBytes` value are the Builder's to finalize within these shapes.
