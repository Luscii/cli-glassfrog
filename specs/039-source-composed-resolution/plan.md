# Plan: Source-Composed Resolution

**Feature**: 039-source-composed-resolution
**Role**: Shaper
**Inputs**: spec.md (039), PROJECT.md, `.score/memory/DECISIONS.md` (precedent: §49 inject-roots, §62 base-URL chain, §71 stdin/TTY seam, §74 `internal/rcfile` shared reader, §161/§170 output-format pure-core resolver), DEPRECATION.md (no superseding entries in this domain), LEARNINGS.md (passive)

---

## System Architecture

Source-Composed Resolution introduces one new leaf package, **`internal/resolve`**, that generalizes the precedence skeleton three settings currently re-implement (token `internal/auth`, base URL `internal/apiclient/baseurl.go`, output format `internal/output/format.go`). It owns no setting; it is a mechanism the call sites will adopt in 040.

The package has four parts:

- **`Source`** — a kind-tagged, lazily-evaluated value origin. Constructors produce them: `FromFlags(...)`, `FromEnv(...)`, `FromFile(...)`, `FromStdin(...)`, plus a trailing `Default(value)`. Each carries its `Kind` (so the resolver can introspect without evaluating) and a closure that, when run, returns *empty*, *a value*, or *an error*.
- **`Resolve(sources ...Source)`** — the walk. It evaluates sources in list order (first = highest precedence), stops at the first that yields, and returns a code-free `Resolution{Value, Provenance}` plus an `error`. Evaluation is lazy: once a source yields, no lower source is touched. A source that errs aborts the walk and surfaces the error (no fall-through).
- **`Resolution` / `Provenance`** — the one shared result shape. `Provenance` carries a `Kind` (`Flag`/`Env`/`File`/`Stdin`/`Default`/`None`) and an `Origin` label (the exact flag name, env var, or file path) — enough for a caller to phrase a validation error without the resolver knowing the setting.
- **The OS seam** — `FromEnv`/`FromFile`/`FromStdin` take injected access (an env lookup, the start/home directories for the rcfile walk, a bounded stdin reader + an `isTTY` flag), mirroring the established inject-the-seam pattern. A thin `…FromOS`-style convenience binds the real `os.Getenv` / `os.Getwd` / `os.UserHomeDir` / `os.Stdin`.

Data flow: a caller composes an ordered `[]Source` → `Resolve` walks it → returns the winning raw value + provenance, or a typed resolution error. The file source delegates the nearest-wins walk to `rcfile.Resolve(startDir, homeDir, key)` (no parallel parser — §74). Validation of the returned value is **not** here; it is the caller's job (ADR-3).

```
caller (040 call site)            internal/resolve                 internal/rcfile
  compose []Source  ─────────────▶ Resolve(sources...)
                                     walk, first-yield-wins, lazy
       FromFile(key) ─────────────────────────────────────────────▶ Resolve(dir,home,key)
                                   ◀── Resolution{Value, Provenance} / error
  validate(Value, Provenance) ── caller maps invalid → usage error
```

---

## Architecture Decisions

### ADR-1: Home the resolver in a new `internal/resolve` leaf package

**Context**: The precedence skeleton must live somewhere all three settings (and future ones) can adopt without creating an import cycle. `internal/rcfile` already owns the file-format/walk concern (§74); `internal/auth`, `internal/apiclient`, `internal/output` own their respective settings.

**Options considered**:
1. **Add the resolver to `internal/rcfile`** — reuse the existing shared file package. But rcfile is specifically the *file-format and nearest-wins-walk* concern; folding flag/env/stdin precedence into it pulls unrelated I/O concerns (cobra flag state, stdin, env) into a package whose remit is the `.glassfrogrc` format.
2. **New `internal/resolve` leaf package importing only `internal/rcfile`** — a dedicated home for multi-source precedence. The domain packages import `resolve` in 040; `resolve` imports none of them, so no cycle.

**Decision**: Option 2 — `internal/resolve`. It imports `internal/rcfile` (leaf → leaf) for the file source and nothing else from the project. The domain packages will depend on it in 040; it depends on no domain package, keeping it a clean leaf.

**Consequences**: One new package, no cycle risk. `rcfile` stays focused on the file format. The setting-specific constants (`token` / `base_url` / `output` keys, env var names, flag names) stay in their owning packages and are passed *into* the resolver — `resolve` holds no setting-specific knowledge.

### ADR-2: A kind-tagged Source walked lazily into a code-free Resolution

**Context**: The resolver must walk an ordered source list, skip empty sources, stop at the first that yields, return value + provenance, and enforce an at-most-one-STDIN-source rule (spec, ADR-5). The existing resolvers return code-free results with a `Source` enum (§46, §62, §161) — the established "producer classifies a code-free outcome, consumer maps it" split.

**Options considered**:
1. **Plain `func() (string, bool, error)` sources** — minimal. But a bare closure can't be introspected: the resolver can't count STDIN sources before draining the stream, and provenance kind must be threaded out-of-band.
2. **Kind-tagged `Source` (a struct or interface carrying `Kind` + a lazy `eval` closure)** — the resolver reads `.Kind` without evaluating (for the STDIN guard and for provenance), then runs `eval` only when the walk reaches it.

**Decision**: Option 2 — a kind-tagged `Source`. `Resolve(sources ...Source) (Resolution, error)` walks in order; each source's `eval` returns one of {empty → continue, value → stop and return with provenance, error → stop and surface}. The result is a code-free `Resolution{Value string, Provenance}` where `Provenance{Kind, Origin}`, continuing the code-free-outcome precedent. A trailing `Default(value)` always yields with `Kind: Default`; when no source and no default yields, the result is `Provenance{Kind: None}` with a nil error (a valid empty outcome — the token shape, §46).

**Consequences**: Provenance is uniform across all settings, replacing the three divergent per-domain `Source` enums (their removal is 040's work, not this slice's). The kind tag enables the STDIN guard (ADR-5). Lazy evaluation means a lower-precedence source that would error is never reached once a higher source yields — matching the spec's lazy guarantee.

### ADR-3: Value validation stays at the call site; the resolver returns raw value + provenance

**Context**: Today each resolver validates inline — `isUsableURL` in `baseurl.go`, `ParseFormat` in `format.go` — and a present-but-invalid value fails loud with no fall-through (§170). The spec (clarified) places validation at the call site so the resolver stays setting-agnostic.

**Options considered**:
1. **Resolver accepts a per-setting validator** — keeps the fail-loud-no-fall-through logic centralized. But it re-couples the shared mechanism to setting-specific rules (URL shape, format enum), the exact coupling this feature exists to remove.
2. **Resolver returns the raw winning value + provenance; caller validates** — the resolver stays purely about *which source won* and *where the value came from*.

**Decision**: Option 2. `Resolve` picks the first source that yields a (non-empty, for value-only sources) value and returns it verbatim with provenance; it runs no validator. The **observable** present-but-invalid-fails-loud behavior (§170) is preserved by construction: the highest-precedence non-empty source wins, and the caller validates *that* winner using the returned `Provenance.Origin` to phrase the error (`--base-url`, `GLASSFROG_BASE_URL`, the file path). Because resolution is first-non-empty (not first-*valid*), an invalid high-precedence value still wins and is then rejected by the caller — never silently superseded by a lower source.

**Consequences**: The resolver has zero setting knowledge. 040 must relocate `isUsableURL` / `ParseFormat` to their call sites and validate the resolved value there — explicitly noted for 040. The resolver's own error surface narrows to *resolution* errors (unreadable/malformed `.glassfrogrc`, stdin read failure), not value validation — these pass through verbatim as typed errors (the rcfile typed errors already exist). The default value is returned unvalidated, matching today's "default is valid by construction, never re-validated."

### ADR-4: Sources take an injected OS seam; a thin binding supplies the real os globals

**Context**: The current resolvers read process globals through package-level `var getenv = os.Getenv` (etc.) and inject `startDir`/`homeDir` into the pure walk (§49, §71). The composable resolver's env/file/stdin sources also touch the OS and must stay hermetically testable.

**Options considered**:
1. **Package-level `var` seams in `internal/resolve`** — matches the literal shape of the three current resolvers. But package globals are process-wide mutable state shared across all settings resolving concurrently in one process, and they don't compose with per-source list-walking (e.g. a `FromEnv` over several names).
2. **Inject the seam into the source constructors** — `FromEnv(lookup, names...)`, `FromFile(startDir, homeDir, key)` (delegating to `rcfile.Resolve`), `FromStdin(read, isTTY)`; a thin `FromOS`-style helper binds the real `os.Getenv` / dirs / a bounded `os.Stdin` reader + `term.IsTerminal`.

**Decision**: Option 2 — inject per source. This conforms to the deeper precedent (§49 inject-roots, §71 "a pure resolver takes the candidate sources … + startDir/homeDir; a thin production seam binds real stdin/TTY/os.Getenv/dirs"), and the stdin source reuses §71's bounded-read + `isTTY` shape directly (`authlogin_seam.go`). Tests construct sources over fake lookups / temp dirs / in-memory stdin, so no suite reads the developer's real environment.

**Consequences**: Every branch is hermetically testable without process-global mutation. The pure constructors carry no `os` dependency in their core logic; only the thin binding helper imports `os`/`term`. The stdin source bounds its read (the `maxPipedTokenBytes` precedent) so a runaway pipe can't exhaust memory.

### ADR-5: More than one STDIN source in a composition is a programming error → panic

**Context**: STDIN is a single consumable stream. The spec forbids more than one STDIN source per resolution and requires it be "surfaced loudly." This is a composition (wiring) mistake by a developer, not a runtime input condition.

**Options considered**:
1. **Return a typed error from `Resolve`** — testable without `recover`, but overloads `Resolve`'s error (which represents *runtime* resolution failures like an unreadable file) with a *wiring* fault, blurring the two for callers.
2. **Panic with a clear message** — matches the codebase's established fail-fast-on-wiring-bug convention (PR #20 / §: `NewClient`, `NewRetryExecutor`, `Assemble` all panic on nil seams).

**Decision**: Option 2 — panic. `Resolve` inspects source kinds before walking; if more than one has `Kind: Stdin`, it panics with a message naming the misuse (e.g. `resolve.Resolve: at most one Stdin source per resolution`). This keeps `Resolve`'s returned `error` exclusively about runtime resolution failures, consistent with the nil-seam fail-fast precedent. A test asserts the panic.

**Consequences**: A genuine wiring bug fails immediately and unambiguously at the call site during development, never silently draining the stream for the first reader. Runtime resolution errors and wiring bugs stay cleanly separated. Because no setting consumes STDIN yet, this guard protects a future 040+ consumer rather than any current path.

---

## Cross-cutting Concerns

**Error handling**: Two error classes stay distinct. *Resolution errors* (unreadable/malformed `.glassfrogrc` via `rcfile`, a stdin read failure) are returned verbatim as typed errors from `Resolve` and abort the walk with no fall-through — preserving §170's loud-surfacing. *Wiring errors* (multiple STDIN sources) panic (ADR-5). *Value-invalidity* is neither — it is deferred to the caller (ADR-3).

**Secret hygiene**: The resolver returns the raw value (which, for the token setting, is a secret) to its caller but never logs or formats values itself — it emits no diagnostics. The file source delegates to `rcfile.Resolve`, which returns only the requested key's value (§74), so a non-token resolution never comes into possession of the token. The `Resolution`/`Provenance` types expose only `Value` (read by field, never auto-formatted into messages by the resolver) plus the safe-to-display `Kind`/`Origin`.

**Configuration**: The resolver hardcodes nothing setting-specific. Flag names, env var names, the `.glassfrogrc` key, and the default value are all caller-supplied inputs to the constructors.

**Testing strategy**: Pure constructors over injected lookups / temp dirs / in-memory readers make every branch (each source yields/empty/errs; lazy short-circuit; default backstop; none-found; list-walk within a source; the STDIN-guard panic) hermetic. A regression test pins the lazy guarantee (a malformed lower-precedence file source is never evaluated when a higher source yields). Tests assert provenance `Origin` for each kind so 040 can rely on the labels.

---

## Implementation Strategy

Small, single-package feature. Two phases keep PRs reviewable; phase 2 depends on phase 1's types.

**Phase 1 — Core walk + value-only sources**: Define `Source`, `SourceKind`, `Provenance`, `Resolution`; implement `Resolve` (ordered walk, first-yield-wins, lazy, none-found outcome, STDIN-count guard panic); implement the sources that need no OS I/O — `FromFlags(...)` (presence-based yield, even empty value; list-walk over aliases), `FromEnv(lookup, names...)` (non-empty yield; list-walk), `Default(value)`. Unit tests for precedence, lazy short-circuit, list-walk, none-found, and the panic guard.

**Phase 2 — I/O sources + OS binding**: Implement `FromFile(startDir, homeDir, key)` (delegates to `rcfile.Resolve`, surfaces its typed errors verbatim) and `FromStdin(read, isTTY)` (bounded read; yields only when piped and non-empty). Add the thin `…FromOS` binding helper(s) that supply real `os.Getenv` / `os.Getwd` / `os.UserHomeDir` / bounded `os.Stdin` + `term.IsTerminal`. Tests over temp dirs and in-memory stdin; resolution-error and stdin-read-failure paths.

Both phases ship behind no call-site change — the three existing resolvers are untouched (040's work).

---

## Risks

- **The provenance shape is insufficient for 040's error messages** (low likelihood, medium impact): if `Provenance.Origin` doesn't carry exactly what the current `BaseURLError.Source` / `FormatError.Source` labels need (`--base-url`, `GLASSFROG_BASE_URL`, the file path), 040's behavior-preservation breaks. Mitigation: tests assert the `Origin` label for each kind against the three current label forms before 040 starts.
- **Over-generalization for a single future consumer** (medium likelihood, low impact): `FromStdin` and the list-walking sources have no current consumer, so the API could miss what a real consumer needs. Mitigation: the spec/Definer chose to build them now; keep the surface minimal and let 040 (and the first real STDIN consumer) drive any extension rather than speculating further.
- **Behavioral drift during 040's retrofit** (medium likelihood, high impact): moving validation to the call site (ADR-3) is where a regression could slip in — e.g. forgetting to validate the default, or validating before resolution. Mitigation: 040 carries the existing resolver test suites forward green; this plan flags the validation-relocation explicitly so 040 treats it as the load-bearing change.

---

## What This Plan Does Not Cover

- **The exact Go signatures, type spellings, and constructor names** — `interface` vs `struct` for `Source`, the precise field names, the `…FromOS` helper's shape. These are the interface skill's concern (this is a specification/code-API boundary feature: it produces a reusable package API).
- **Executable scenarios** — the scenarios skill concretizes the spec's driving scenarios into `.feature` files.
- **Task decomposition** — the tasks skill breaks the two phases into PR-sized units.
- **The call-site retrofit** — migrating token / base URL / output onto the resolver, and relocating `isUsableURL` / `ParseFormat` to those call sites, is **040 (Resolution Call-Site Retrofit)**, deliberately out of scope here.
