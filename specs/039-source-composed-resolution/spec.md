# Specification: Source-Composed Resolution

**Feature**: 039-source-composed-resolution
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Source-Composed Resolution is a **Maintainer-facing mechanism**: a single reusable resolver that walks an ordered list of value sources and returns the first one that yields. Today the CLI re-implements the same precedence chain — command flag, then environment variable, then the nearest `.glassfrogrc` file, then a built-in default — three separate times: once for the auth token (005), once for the base URL (008), and once for the output format (020). Each copy carries its own OS-access seam, its own walk-up skeleton, its own provenance enum, and its own error shape, and the copies can drift. This capability extracts that shared skeleton into one composable resolver so that adding a fourth setting is a matter of *listing sources*, not copying a chain.

The resolver is the dependency root of the *Duplicated Setting Resolution* solution and has no upstream feature edges — it leads the refactor. It is a pure mechanism: it owns no setting of its own and changes no existing call site. A separate slice, **Resolution Call-Site Retrofit (040)**, migrates the token, base-URL, and output-format sites onto it; this spec delivers only the resolver, its source constructors, and their own tests. Because the resolver must stay setting-agnostic, it deliberately leaves value validation (is this a well-formed URL? is this a known format?) to its caller — it resolves *which* source supplied a raw value and *where that value came from*, and hands both back.

---

## Behavioral Accord

### Composition

- When a caller composes a resolver from an ordered list of sources, the system treats list order as precedence: the first listed source is highest priority, the last is lowest.
- When a caller appends an optional trailing default to the list, the system uses it as the backstop that always yields once every preceding source has been exhausted.
- When a caller composes a resolver with no trailing default, the system permits an empty outcome — resolving nothing is a valid result, not an error (the auth-token shape, where absence is normal).

### Walking sources

- When the system resolves a setting, it evaluates the sources in order and stops at the first source that yields a value, returning that value together with its provenance.
- When a source produces no value (an unset environment variable, an unprovided flag, a key absent from every `.glassfrogrc` on the walk, no piped STDIN), the system treats that source as empty and continues to the next source.
- When the system stops at the first yielding source, it does not evaluate any lower-precedence source — resolution is lazy, so a malformed lower-priority config file is never read once a higher source has already won.
- When a source is itself a list of inputs (e.g. a flag source given `["--output", "-o"]`, or an environment source given several variable names), the system walks that inner list in order and yields from the first inner input that produces a value; the provenance identifies which specific input yielded.

### Sources

- When a flag source is evaluated, it yields the flag's value whenever the flag was supplied on the command line — even when that value is empty — and is empty (not yielding) only when the flag was not supplied at all. Presence is determined the way the codebase's flag handling already determines it (supplied vs. not), not by inspecting the value.
- When an environment source is evaluated, it yields the variable's value if the variable is set to a non-empty value, and is empty otherwise.
- When a file source is evaluated, it reads the requested key through the shared `.glassfrogrc` nearest-wins walk-up (the same reader the existing settings use) and yields the value from the nearest file that carries that key; a tokenless or keyless nearer file does not shadow a value lower down.
- When a STDIN source is evaluated, it yields piped input when input is present (non-interactive) and non-empty, and is empty when STDIN is a terminal or carries no data.
- When a single resolver composition includes more than one STDIN source, the system treats that as a programming error and surfaces it loudly rather than silently draining the single stream for the first reader. At most one STDIN source may participate in one resolution.

### Provenance and result shape

- When the system returns a resolved value, it returns one shared result shape carrying the value, its provenance, and — for sources that read external state — the concrete origin (the exact flag name, environment variable, or file path that supplied it).
- When the trailing default supplies the value, the provenance records it as the default rather than as any external source.
- When no source and no default yields, the system returns a result whose provenance records that nothing was found, with no error raised.

### Resolution errors

- When a source cannot complete its read — an unreadable or malformed `.glassfrogrc`, or a failed STDIN read — the system aborts resolution and returns that source's error (verbatim), and does not silently skip past it to a lower source.
- When the system surfaces a resolution error, it surfaces it uniformly as a non-nil `error` return from `Resolve` (callers may `errors.As` typed errors such as `*rcfile.ReadError`/`*rcfile.FormatError`).

---

## User Scenarios

**In order to** add a new configurable setting without copying the flag→env→file→default chain and its OS seam,
**as a** Maintainer,
**I want to** compose a resolver by listing reusable source constructors.

**In order to** give a user a precise error that names where a bad value came from,
**as a** Maintainer,
**I want to** receive the provenance of the winning value alongside the value itself.

**In order to** test resolution deterministically without touching the real environment, filesystem, or terminal,
**as a** Maintainer,
**I want to** have the resolver's OS access sit behind an injectable seam.

---

## Non-Behaviors

- The resolver must not validate the resolved value — no URL well-formedness check, no output-format membership check, no token shape check. **Why**: validation rules are setting-specific; baking one setting's rules into the shared resolver would re-couple it to that setting and defeat the deduplication the capability exists to achieve. The caller validates the returned value using the returned provenance to phrase the error.
- The resolver must not modify, migrate, or wrap any of the three existing resolution sites (token, base URL, output format). **Why**: keeping the mechanism (039) and the migration (040) in separate slices lets the resolver land behavior-neutral and independently reviewable; mixing them would make a regression in either hard to isolate.
- The resolver must not write any file or environment variable, and must not make a network call. **Why**: all three settings it generalizes are read-only resolutions; writing credentials is Credential Storage's job, and resolution reads local state only.
- The resolver must not prompt or solicit input interactively. **Why**: the operator is usually an AI agent, so resolution must be deterministic and reproducible — the same inputs yield the same result on every run. (Reading already-piped STDIN is not a prompt.)
- The resolver must not emit any resolved value into its own diagnostic or log output. **Why**: one of the settings it generalizes is the auth token, a secret; the resolver staying value-silent preserves the existing redaction discipline regardless of which setting it serves.

---

## Integration Boundaries

- **Command-line flags**: a flag source reads cobra flag state (whether a flag was supplied, and its value), passed in by the caller. The resolver does not register flags itself.
- **Process environment**: an environment source reads variables through the injectable OS seam. Unset or empty variables read as empty.
- **`.glassfrogrc` file**: a file source reads a single key through the shared nearest-wins walk-up reader. A missing file is normal (empty); an unreadable or malformed file is a resolution error naming the path.
- **STDIN**: a STDIN source reads piped input through the seam. A terminal (no pipe) or empty stream reads as empty; a read failure is a resolution error.

---

## Driving Scenarios

### Happy path

**Scenario: Highest-precedence source wins**
Given a resolver composed of a flag source, an environment source, a file source, and a trailing default
And the flag was supplied with a value
When the setting is resolved
Then the system returns the flag's value
And the provenance identifies the flag as the origin

**Scenario: Resolution falls through empty sources**
Given the same resolver
And the flag was not supplied but the environment variable is set to a non-empty value
When the setting is resolved
Then the system skips the empty flag source and returns the environment value
And the provenance identifies the environment variable as the origin

**Scenario: Trailing default backstops an otherwise-empty chain**
Given the same resolver
And no flag, environment variable, or `.glassfrogrc` key supplies a value
When the setting is resolved
Then the system returns the default value
And the provenance records the value as coming from the default

### Error scenarios

**Scenario: Malformed config file fails loud rather than falling through**
Given a resolver whose file source reads `.glassfrogrc`
And the nearest `.glassfrogrc` is unreadable or malformed
And no higher-precedence source yielded
When the setting is resolved
Then the system surfaces a resolution error naming the file path
And it does not silently skip to a lower-precedence source

**Scenario: STDIN read failure is surfaced uniformly**
Given a resolver whose highest-precedence source reads STDIN
And reading the piped STDIN fails
When the setting is resolved
Then the system surfaces a resolution error in the same shape used for a config-file failure

### Edge cases

**Scenario: No source and no default yields a valid empty result**
Given a resolver composed of an environment source and a file source with no trailing default
And neither the variable nor any `.glassfrogrc` carries the key
When the setting is resolved
Then the system returns a result whose provenance records that nothing was found
And no error is raised

**Scenario: A list-valued source yields from its first matching input**
Given a flag source composed over the inputs `["--output", "-o"]`
And `--output` was not supplied but `-o` was supplied with a value
When the setting is resolved
Then the system returns the `-o` value
And the provenance identifies `-o` as the specific input that yielded

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: The resolver names no concrete setting**
Given the resolver and its source constructors
When a reviewer reads their contract
Then nothing references the token, base URL, or output format by name — the mechanism is setting-agnostic and would serve a fourth setting unchanged.

**Scenario: Provenance is rich enough to reproduce existing error labels**
Given a resolved value
When the caller phrases a validation error from the provenance alone
Then it can produce the same origin labels the current resolvers use (`--base-url`, `GLASSFROG_BASE_URL`, the file path) without the resolver knowing which setting it served.

**Scenario: No resolved value leaks into diagnostics**
Given a resolution that returns a value (including a token)
When the resolver's own output is inspected
Then no resolved value appears in it.

---

## Assumptions

- **Yield semantics differ between flags and the value-only sources**: a flag source yields whenever the flag was supplied (presence), even with an empty value; the environment, file, and STDIN sources yield only on a present, non-empty value. (Resolved in clarify — flags follow the codebase's `Changed()`-based convention, while the value-only sources cannot distinguish unset from set-empty and so key on non-emptiness.)
- **STDIN is non-interactive only**: a STDIN source yields only when input is piped (STDIN is not a terminal); on a terminal it yields empty rather than blocking for input. (Informed by the deterministic-non-interactive constraint — the resolver must never wait on a human.)
- **OS seam follows the existing pattern**: environment, working-directory, home-directory, and STDIN access sit behind injectable function values (the pattern the three current resolvers already use), not a new interface. (Technical — preserves the hermetic test approach already in the codebase.)
- **`fromStdin` ships without a current consumer**: none of the three retrofit targets reads STDIN today, but the source is built now so the resolver's source set is complete when 040 migrates. (Per the FEATURE-MODEL and the Definer decision; the contention question this raised is resolved by the at-most-one-STDIN-source constraint in the Behavioral Accord.)

---

## Ambiguity Warnings

None remaining — both warnings from the initial draft (STDIN contention, flag yield semantics) were resolved during clarification. See Clarifications.

---

## Clarifications

### Session 2026-06-11

- **Flag yield semantics**: A flag source yields whenever the flag was *supplied* (presence, per cobra `Changed()`), even with an empty value — `--flag ""` wins and the caller validates it. This aligns with the codebase's established "validate on `Changed()`, not value" convention. The environment, file, and STDIN sources continue to yield only on a present, non-empty value, since they cannot distinguish unset from set-empty.
- **STDIN contention**: A single resolver composition may include at most one STDIN source. A second STDIN source is a programming error surfaced loudly, rather than silently draining the single stream for the first reader. No buffering machinery is introduced — no setting consumes STDIN yet.
