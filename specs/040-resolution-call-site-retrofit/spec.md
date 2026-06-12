# Specification: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Resolution Call-Site Retrofit migrates the CLI's three hand-rolled setting-precedence chains onto the single composable resolver delivered by Source-Composed Resolution (039). Today three call sites each re-implement the same flag→env→`.glassfrogrc`→default skeleton: the auth token (005, `internal/auth`), the base URL (008, `internal/apiclient`), and the output format (020, `internal/output`). Each copy carries its own walk, its own whitespace-trim rule, and its own precedence ordering, and the copies can drift. This slice replaces each chain's *innards* with a composition of `resolve` sources, deletes the duplicated skeleton, and maps the generic `resolve.Resolution` back onto each site's existing public output type — so the resolver becomes the one place precedence is implemented.

This is the consuming half of the *Duplicated Setting Resolution* solution: 039 built the resolver and changed no call site; 040 moves the call sites onto it. The retrofit is **behaviour-preserving at every public surface** — output types (`auth.Resolution`, `apiclient.BaseURL`, `output.OutputFormat`), provenance enums, and typed errors (`*BaseURLError`, `*FormatError`, rcfile's typed errors) stay exactly as they are, so downstream consumers (007 Request Authentication, `assemble.go`, the cli read commands) are untouched. It carries **one deliberate behaviour change**: the base-URL and output-format flag rungs switch from value-emptiness to presence (cobra `Changed()`) as the test for "the flag was supplied", aligning them with the resolver's flag semantics and fixing a long-standing inconsistency (PRs #61/#73). Setting-specific knowledge — flag/env names, file keys, defaults, and value validators — stays at each call site; `resolve` remains setting-agnostic.

---

## Behavioral Accord

### Token resolution (005) — unchanged behaviour

- When the token is resolved, the system consults the `GLASSFROG_TOKEN` environment variable first and the nearest `.glassfrogrc` `token` key second, returning the first that yields — the existing env-then-file precedence, now expressed as a `resolve` composition with no flag rung and no default.
- When no source yields a token, the system returns the "nothing found" outcome (`auth.Resolution` with `SourceNone`) and no error — absence is a normal result, not a failure.
- When a `.glassfrogrc` on the token walk is unreadable or unparseable, the system aborts with rcfile's typed error naming the file and does not fall through to a lower source.
- When the system returns a resolved token, it maps the resolver's provenance back onto `auth.Resolution` (env → `SourceEnvironment`, file → `SourceFile` with `Path`) and never formats, logs, or places the token value in any message or error.

### Base URL resolution (008)

- When the base URL is resolved, the system walks the `--base-url` flag, then `GLASSFROG_BASE_URL`, then the nearest `.glassfrogrc` `base_url` key, then the built-in default, returning the first that yields — the existing precedence, now a `resolve` composition.
- When the `--base-url` flag was supplied on the command line, the system treats it as the winning source by its **presence** (cobra `Changed()`), even when its value is empty or whitespace, and validates that value; when the flag was not supplied, the system falls through to the environment.
- When the environment variable or a file value is whitespace-only, the system treats it as absent and falls through to the next source (unchanged trim rule).
- When the winning source supplies a value that is not an absolute http(s) URL, the system fails loud with a `*BaseURLError` naming that source and does not fall through to a lower source.
- When no flag, environment, or file value yields, the system returns the built-in default (`SourceDefault`), which is valid by construction and never re-validated — so resolution always yields a base URL.
- When the system returns a resolved base URL, it maps the resolver's provenance back onto `apiclient.BaseURL` (flag/env/file/default → the existing `BaseURLSource` members, with `Path` for the file rung).

### Output format resolution (020)

- When the output format is resolved, the system walks the `--output` (and `-o`) flag, then `GLASSFROG_OUTPUT`, then the nearest `.glassfrogrc` `output` key, then the built-in default (`full`), returning the first that yields — the existing precedence, now a `resolve` composition.
- When the `--output`/`-o` flag was supplied on the command line, the system treats it as the winning source by its **presence**, even when its value is empty or whitespace, and parses that value; when the flag was not supplied, the system falls through to the environment.
- When the environment variable or a file value is whitespace-only, the system treats it as absent and falls through.
- When the winning source supplies a value that is not one of `full | compact | json | yaml`, the system fails loud with a `*FormatError` naming that source and the offending value, and does not fall through.
- When no source yields, the system returns the built-in default `full`, so resolution always yields a format.

### Shared-mechanism adoption (Maintainer-facing)

- When a maintainer reads any of the three resolvers after the retrofit, they find precedence expressed as an ordered list of `resolve` sources, not a hand-rolled walk — the duplicated trim/walk/precedence skeleton is gone from each call site.
- When a maintainer adds a future setting, the per-setting work is listing sources and writing a small mapping to a public type, because the precedence walk, the lazy short-circuit, and the file walk now live once in `resolve`.

---

## User Scenarios

**In order to** maintain one precedence implementation instead of three drifting copies,
**as a** Maintainer,
**I want to** express each setting's resolution as a composition of shared `resolve` sources.

**In order to** get a loud, immediate error when I explicitly pass an empty or malformed `--base-url` or `--output` rather than having it silently ignored,
**as a** CLI user,
**I want to** the flag I supplied to be honoured as the winning source by presence, not quietly dropped when its value is blank.

**In order to** keep relying on the token, base-URL, and output-format outputs exactly as before,
**as a** downstream consumer (Request Authentication, connection assembly, the read commands),
**I want to** the retrofit to leave every public output type, provenance enum, and typed error unchanged.

---

## Non-Behaviors

- The system must not change any public output type, provenance enum, or typed error shape (`auth.Resolution`/`Source`, `apiclient.BaseURL`/`BaseURLSource`/`*BaseURLError`, `output.OutputFormat`/`*FormatError`). **Why**: consumers (007, `assemble.go`, the cli render path) depend on these surfaces; the retrofit is an internal mechanism swap, and changing a surface would turn a refactor into a breaking change.
- The system must not move setting-specific constants — flag names, env var names, `.glassfrogrc` keys, default values — into `internal/resolve`. **Why**: `resolve` owns no setting (039 ADR-1); centralising these would re-couple the generic walk to the domain and reintroduce the cycle 039 avoided.
- The system must not add value validation (URL well-formedness, format-token membership) to `internal/resolve`. **Why**: `resolve` resolves *which* source won and *where* the value came from, not whether it is well-formed (039 ADR-3); validation stays at the call site that owns the value's meaning.
- The system must not introduce a STDIN source at any of the three call sites. **Why**: none of token, base URL, or output format reads piped input; wiring an unused reader would add a consumable-stream hazard for no behaviour.
- The system must not give the token resolver a flag rung or a trailing default, nor change its env→file precedence. **Why**: 005's contract is that a missing token is a normal `SourceNone` outcome and there is no `--token` flag; adding either would change observable token behaviour.
- The system must not re-validate or normalize the built-in default base URL or default format. **Why**: both are valid by construction (008/020) and backstop the chain; re-validating them adds a failure mode that cannot occur.
- The system must not alter the `.glassfrogrc` format, the nearest-wins walk, or rcfile's typed read/format errors. **Why**: the one shared file walk (DECISIONS §74) is reused verbatim through `resolve.FromFile`; a second parser or a changed error shape would fork the file contract.

---

## Integration Boundaries

- **`internal/resolve` (039)**: each call site composes `resolve.Resolve` over `FromFlags`/`FromEnv`/`FromFile`/`Default` and reads back a `resolve.Resolution{Value, Provenance}`. Inbound dependency only; `resolve` does not import the call sites.
- **`internal/rcfile`**: the file rung is delegated to `resolve.FromFile`, which wraps the existing `rcfile.Resolve` nearest-wins walk. Typed `*ReadError`/`*FormatError` propagate unchanged.
- **Request Authentication (007) / connection assembly (`assemble.go`)**: consume `auth.Resolution` and `apiclient.BaseURL` exactly as today — no change expected.
- **The cli read commands (`me`, `roles`, `domains`, …)**: supply the `--base-url` and `--output` flag inputs. After the retrofit they pass the flag's **presence** (`cmd.Flags().Changed(...)`) alongside its value, where today they pass only the value string.

---

## Driving Scenarios

### Happy path

**Scenario: Token resolved from the environment**
Given `GLASSFROG_TOKEN` is set to a non-empty value
When the token is resolved
Then the system returns an `auth.Resolution` with that token and `Source` = environment
And no `.glassfrogrc` is read.

**Scenario: Base URL resolved from a supplied flag**
Given `--base-url https://example.com/api/v5` is supplied
When the base URL is resolved
Then the system returns a `BaseURL` with that value and `Source` = flag
And the environment and `.glassfrogrc` are not consulted.

**Scenario: Output format falls through to the built-in default**
Given no `--output` flag, no `GLASSFROG_OUTPUT`, and no `.glassfrogrc` `output` key
When the output format is resolved
Then the system returns `full` with `Source` = default.

**Scenario: Base URL falls through flag→env→file→default**
Given the `--base-url` flag is not supplied and `GLASSFROG_BASE_URL` is unset
And the nearest `.glassfrogrc` carries `base_url = https://team.example.com/api/v5`
When the base URL is resolved
Then the system returns that file value with `Source` = file and the file's path.

### Error scenarios

**Scenario: Malformed base URL from the flag fails loud**
Given `--base-url not-a-url` is supplied
When the base URL is resolved
Then the system returns a `*BaseURLError` naming `--base-url`
And no lower-precedence source is consulted.

**Scenario: Unparseable `.glassfrogrc` on the output walk fails loud**
Given the `--output` flag is not supplied and `GLASSFROG_OUTPUT` is unset
And the nearest `.glassfrogrc` is malformed
When the output format is resolved
Then the system returns rcfile's typed format error naming the file
And the built-in default is not used.

### Edge cases

**Scenario: Explicitly empty `--base-url` is honoured by presence and fails loud**
Given `--base-url ""` (or whitespace) is supplied on the command line
When the base URL is resolved
Then the system treats the flag as the winning source by its presence
And returns a `*BaseURLError` naming `--base-url`
And does not fall through to the environment.

**Scenario: Whitespace-only `GLASSFROG_OUTPUT` is treated as absent and falls through**
Given the `--output` flag is not supplied and `GLASSFROG_OUTPUT` is set to `"   "`
And no `.glassfrogrc` `output` key is present
When the output format is resolved
Then the system treats the environment value as absent
And returns the built-in default `full`.

**Scenario: No token anywhere is a normal empty outcome**
Given `GLASSFROG_TOKEN` is unset and no `.glassfrogrc` carries a `token` key
When the token is resolved
Then the system returns `auth.Resolution` with `Source` = none and no error.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: No precedence skeleton remains at a call site**
Given the retrofit is complete
When a reader inspects `internal/auth/resolve.go`, `internal/apiclient/baseurl.go`, and `internal/output/format.go`
Then precedence is expressed as a `resolve` source composition
And none of the three re-implements the env-trim / file-walk / source-ordering skeleton by hand.

**Scenario: Public surfaces are byte-for-byte stable for consumers**
Given the retrofit is complete
When a consumer reads `auth.Resolution`, `apiclient.BaseURL`, `output.OutputFormat`, and the typed errors
Then their fields, provenance members, and error message shapes are unchanged from before the retrofit.

**Scenario: The flag-semantics change is the only observable behaviour difference**
Given the same environment and `.glassfrogrc` as before the retrofit
When every documented resolution path is exercised
Then the only differing observable outcome is that an explicitly-supplied empty/whitespace `--base-url` or `--output` now fails loud instead of falling through.

---

## Assumptions

- **Flag presence is threaded from the command layer**: the cli read commands can pass `cmd.Flags().Changed(...)` to the base-URL and output-format resolvers, since sibling flags in the same `RunE` already read `Changed()`. (Confirmed by inspection of `domain.go`/`me.go`; the resolver signatures gain a presence input.)
- **Provenance origin labels align**: `resolve.Provenance.Origin` (e.g. `--base-url`, `GLASSFROG_BASE_URL`, the file path) supplies the same operator-facing source labels the existing typed errors already use. (Informed by 039's source constructors and the current `*BaseURLError`/`*FormatError` source strings.)
- **The token's `resolve.Resolution` is mapped, never formatted**: the generic `resolve.Resolution` has no redacting `String()`, so the token-path adapter maps it into `auth.Resolution` (which does redact) without formatting the intermediate. (Technical assumption; preserves 005 secret hygiene.)

---

## Ambiguity Warnings

_None — the one behavioural decision (flag presence vs value-emptiness) was resolved in the defining conversation in favour of presence-based (`Changed()`) semantics._
