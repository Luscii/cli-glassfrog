# Tasks: Base URL Resolution

**Feature**: 008-base-url-resolution
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/undefined-connection-settings/base-url-resolution.feature

---

## Dependency Graph

Phase 1: `base_url` file read in `internal/auth` (1 task, no phase dependencies) [Shared]
Phase 2: Base URL resolver + precedence + validation in `internal/apiclient` (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: base URL resolution is infrastructure serving all three user scenarios (flag-override / default-out-of-the-box / project-local-precedence) rather than any single one.
>
> **Cross-spec note**: this slice reuses Credential Discovery's (005) one shared `.glassfrogrc` parser + `candidateDirs` walk — implemented and validated Ready on main — and lives in `internal/apiclient`, the package Request Authentication (007) created. The `base_url` file key, `GLASSFROG_BASE_URL`, and the `--base-url` flag name are `[ASSUMED]` CLI conventions — reconcile the file key with Credential Storage (006). The built-in default `https://glassfrog.com/api/v5` is a fixed constant — the `/api/v5` path from `spec/glassfrog-api-v5.yaml`, the host inferred from `info.contact.url` (risk H-1).

---

## Branching Guidance

**Pipeline mode**: `spec/008-base-url-resolution/base` → `spec/008-base-url-resolution/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: none active — specs 001–007 are Complete; 008 is the only in-progress spec. The deferred connection-context half of Connection Resolution (combine base URL + token, assemble the `http.Client` that 007's `AuthTransport` wraps) is a later spec, not a concurrent one.

---

## Phase 1: `base_url` file read in `internal/auth` [Shared]

- [x] **T001** [Shared] Add a network-free, secret-safe `base_url` file read to `internal/auth`, reusing the shared parser and walk — RED-first unit tests — 19 unit tests (file reader + walk + secret-hygiene tripwires); generalized `parseCredentials` into a shared `parseFields` pass (token reader unchanged), added `baseURLKey`, `readBaseURLFile`, exported `ResolveBaseURLFile` walk
  - **Scope**: In `internal/auth`, add the `base_url` file-key constant beside `tokenKey`. Add an exported function that resolves the `base_url` value from the nearest `.glassfrogrc` over the **existing** `candidateDirs` walk (current directory → ancestors → home, nearest-wins, home-dedupe) and the **existing** `parseCredentials` step — returning `(value string, path string, found bool, err error)`. The **token is never returned** through this path and never appears in any error (secret hygiene; plan ADR-3). It performs no URL validation (the raw string is returned; validation is Phase 2's concern) and makes no network call. Generalize `parseCredentials` to capture the `base_url` key alongside `token` without widening what the token reader exposes — no second `.glassfrogrc` parser is written.
  - **Acceptance criteria**:
    - The reader returns the nearest file's `base_url` value and that file's path; nearest-wins and home-dedupe match the token walk (reuses `candidateDirs`)
    - A file that exists and parses but has no `base_url` entry → `found=false`; the walk continues to the next location (does not stop at it)
    - A missing file at a candidate location is skipped (unwraps to `os.ErrNotExist`); an unreadable file → the shared `*ReadError` naming the path; an unparseable file → the shared `*FormatError` naming the path — no silent fall-through
    - The token value never appears in the returned value or in any error from this path
    - RED-first unit tests over temp directory trees with injected roots (reuses 005's seam — tests never touch the real home `.glassfrogrc`); `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (builds on 005's reader, on main)
  - **Plan reference**: Phase 1 — `base_url` file read; ADR-3 (reuse the one shared reader; token never returned)
  - **Interface references**: interface-spec.md — `.glassfrogrc` `base_url` key (Surface), the shared-reader read/format errors (Error Communication)
  - **Scenario references**: undefined-connection-settings/base-url-resolution.feature: "The nearest config file wins over the home file", "A config file with no base URL is skipped", "An unreadable config file fails loudly"
  - **Risk**: ⚠️ Secret hygiene — generalizing `parseCredentials` must not let the `base_url` path return or log the token. Add a test asserting the token value never appears in the `base_url` read's value or errors.

## Phase 2: Base URL resolver + precedence + validation in `internal/apiclient` [Shared]

- [x] **T002** [Shared] Implement the precedence resolver (flag → env → file → default) with `http(s)` validation and the code-free `BaseURL` result, RED-first — 18 unit tests (each rung, malformed-per-source, short-circuit tripwires, verbatim pass-through, determinism, URL-validity boundary table, production seam); added `BaseURL`/`BaseURLSource`/`BaseURLError`, centralized `GLASSFROG_BASE_URL`/`base-url`/`DefaultBaseURL` constants, pure `ResolveBaseURL` + `ResolveBaseURLFromOS` seam
  - **Scope**: In `internal/apiclient`, define `BaseURL{Value, Source, Path}` and a `Source` enum (`Flag`, `Environment`, `File`, `Default` — **no `None`**). Add centralized constants: `GLASSFROG_BASE_URL` (env var, `[ASSUMED]`), the `--base-url` flag name (`[ASSUMED]`), and the **pinned** default `https://glassfrog.com/api/v5`. Implement a pure `ResolveBaseURL(flagValue, startDir, homeDir) → (BaseURL, error)`: try the flag, then `GLASSFROG_BASE_URL`, then T001's `base_url` file walk, then the default. **Usable** = an absolute URL with an `http`/`https` scheme; a whitespace-only value is absent (fall through); a non-empty value that is not an `http(s)` URL is a typed code-free `BaseURLError` naming the source (`flag` / `GLASSFROG_BASE_URL` / the file path) with **no fall-through**. The default is known-valid and not re-validated; the chain always yields a value. The value is passed through verbatim (no trailing-slash/scheme normalization). Add a thin production seam binding `os.Getenv`/`os.Getwd`/`os.UserHomeDir` and the flag value. No exit code, no `os.Exit`, no command surface, no `--base-url` cobra registration (deferred to the consuming command).
  - **Acceptance criteria**:
    - Flag set (valid) → `BaseURL{Source: Flag}`, no other source consulted; env set (valid), no flag → `Source: Environment`, no file read; nearest file `base_url` (valid), no flag/env → `Source: File` with the path; nothing usable → `Source: Default` with the pinned default
    - A non-empty value that is not an absolute `http(s)` URL (scheme-less host, non-`http` scheme) at any source → `BaseURLError` naming that source; the resolver does not fall through to a lower source or the default
    - A whitespace-only flag/env/file value, and a file with no `base_url`, fall through; an unreadable/unparseable file surfaces T001's typed error (no silent fall-through to the default)
    - Resolution is deterministic (same flag/env/fs → same `Value` + `Source`); it always returns a value; it performs no writes and makes no network call
    - The token is never in scope on any base-URL path; `BaseURLError` carries only the source label / path
    - RED-first unit tests cover each rung, malformed-per-source (flag, env, file), empty-env-falls-through, file-without-`base_url`-falls-through, default backstop, and determinism; short-circuit rungs use the directory-at-path tripwire (LEARNINGS) so a stray file read fails loudly; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2 — base URL resolver; ADR-1 (lives in `internal/apiclient`), ADR-2 (precedence + always-a-value), ADR-4 (validate + typed code-free error), ADR-5 (injected seams)
  - **Interface references**: interface-spec.md — Configuration inputs + precedence, "usable" URL contract, `BaseURL` output (Surface/Interactions), Error Communication
  - **Scenario references**: undefined-connection-settings/base-url-resolution.feature: "The flag overrides every other source", "A malformed flag value fails loudly", "The built-in default is used when nothing is configured", "The environment variable wins over a config file", "A malformed environment value names the environment variable"
  - **Risk**: ⚠️ The default host is *derived* (spec declares a relative server `url: /api/v5`; host taken from `contact.url` `https://glassfrog.com`) — keep it a single constant. URL-validity boundary (scheme-less vs `http(s)` vs non-`http`) needs explicit accept/reject table tests.

## Phase 3: Executable acceptance [Shared]

- [x] **T003** [Shared] Make the 008 driving scenarios pass as executable acceptance via godog, driving `ResolveBaseURL` over temp dirs with injected roots and a controlled flag/env — 10 behavioral scenarios pass in a new `TestBaseURLFeatures` suite scoped to base-url-resolution.feature; 4 `@validation` scenarios kept `@wip`; both apiclient suites verified reporting independent counts (10 + 8). NB: behavioral scenarios arrived already un-`@wip` from the scenarios skill — nothing to remove
  - **Scope**: Add godog step definitions for `features/undefined-connection-settings/base-url-resolution.feature` (all three Rule blocks), driving `ResolveBaseURL` over temp directory trees with injected roots, a controlled `GLASSFROG_BASE_URL`, and a supplied flag value. Assert flag/env/file/default precedence, nearest-wins, malformed-naming-the-source (flag, env, **and file**), empty-env fall-through, file-without-`base_url` fall-through, unreadable-file error, and the default backstop. Remove `@wip` from the passing behavioral scenarios; keep the four `@validation` scenarios (always-yields-a-value / deterministic / no-writes / no-network) `@wip` (held out for validate). **Suite scoping (LEARNINGS)**: the `internal/apiclient` godog suite now serves two feature files (007's `request-authentication.feature` and this one) — point the suite's `Paths` at the **specific feature files**, not the `features/` directory, so un-wipping one spec's scenarios can't break another suite.
  - **Acceptance criteria**:
    - Every non-`@validation` 008 scenario (flag wins / malformed flag / built-in default / env wins / nearest file wins / no-base_url skip / empty env ignored / unreadable file / malformed env names source / malformed file value names file) has an executable, passing path
    - `@wip` removed from those scenarios; the four `@validation` scenarios keep `@wip`
    - No real network and no real home directory are touched — temp dirs and injected roots only; the suite asserts the filesystem is unchanged and no outbound connection is made
    - The `apiclient` suite's `Paths` names specific feature files (both 007's and 008's); `go build ./...`, `go vet ./...`, and the feature suite run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: undefined-connection-settings/base-url-resolution.feature: all 008 Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — adding a second feature file to the `apiclient` package must keep each suite pointed at specific files (not the directory); verify both suites report their own `N scenarios (N passed)`. Test isolation — fakes/temp dirs must not leak between scenarios.
