# Tasks: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/undefined-connection-settings/connection-context-assembly.feature

---

## Dependency Graph

Phase 1: `ConnectionContext` value object + readiness + redacting `String()` in `internal/apiclient` (1 task, no phase dependencies) [Shared]
Phase 2: `Assemble` + `AssembleFromOS` aggregator in `internal/apiclient` (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance via godog (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: connection-context assembly is infrastructure serving all three user scenarios (pair-endpoint-and-identity / see-what's-ready / assemble-once-reuse) rather than any single one.
>
> **Cross-spec note**: this slice is purely additive — it changes no existing file. It consumes landed code: 008's `ResolveBaseURLFromOS` + `BaseURL`/`BaseURLError` and 005's `auth.Resolve` + `auth.Resolution`, both in/under `internal/apiclient` and `internal/auth` on main. It resolves the 007/009 `[ASSUMED]` seam (DECISIONS): the connection context is the single resolution point, and Request Execution (010) will replay the context's cached credential into 007's existing `AuthTransport` rather than re-resolving — so **007's code is untouched here** (the wiring lives in 010). Building the `http.Client` / base transport from the context is **also** 010's, not this slice's (refines 008's forecast). No new `.glassfrogrc` key, env var, or flag is introduced.

---

## Branching Guidance

**Pipeline mode**: `spec/009-connection-context-assembly/base` → `spec/009-connection-context-assembly/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: none active — specs 001–008 are Complete; 009 is the only in-progress spec. Request Execution (010) and the future command that triggers API calls are later specs, not concurrent ones.

---

## Phase 1: `ConnectionContext` value object + readiness + redacting `String()` [Shared]

- [ ] **T001** [Shared] Add the `ConnectionContext` value object pairing 008's `BaseURL` outcome with 005's `auth.Resolution` outcome, with derived readiness and a redacting `String()` — RED-first unit tests
  - **Scope**: In `internal/apiclient` (a new `context.go`), define `ConnectionContext{ BaseURL BaseURL; BaseURLErr error; Cred auth.Resolution; CredErr error }` — carrying 008's base-URL outcome (value or `BaseURLErr`) and 005's credential outcome (`Cred`, with `Source == auth.SourceNone` meaning absent, or `CredErr`). Add `Complete() bool` (= `BaseURLErr == nil && CredErr == nil && Cred.Source != auth.SourceNone`). Add `Problems() []string` returning safe-to-display labels for each missing/errored part — base URL first, then credential — empty when complete, each built **only** from the base-URL error's source/path and the credential `Source`/`Path` (or a fixed "no credentials found" phrase), **never** the token. Add a **value-receiver** redacting `String()` that renders the base-URL source (or its error label), the credential `Source`/`Path`, and the readiness, reporting the token as present/absent — never verbatim. No I/O, no network, no writes.
  - **Acceptance criteria**:
    - `Complete()` is true only when there is a usable base URL **and** a present token (`Source != None`) **and** neither error is set; false when any part is missing or errored
    - `Problems()` is empty when complete; otherwise has one entry per incomplete part, naming it, in stable order (base URL, then credential); every entry is secret-free
    - `String()` renders the safe parts and reports the token as present/absent; `%v`/`%+v`/`%s`/`String()` never contain the token value (mirrors `auth.Resolution.String()`)
    - RED-first unit tests: complete; credential-absent; base-URL-error; credential-error; both-errored; and a redaction test asserting a real token value never appears in `%+v`, `String()`, or any `Problems()` entry; `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (builds on 005's `auth.Resolution` and 008's `BaseURL`/`BaseURLError`, on main)
  - **Plan reference**: Phase 1; ADR-1 (code-free value object); Cross-cutting Concerns (secret hygiene — redacting `String()`)
  - **Interface references**: interface-spec.md — Output contract `ConnectionContext`, Readiness accessors (Surface), Error Communication
  - **Scenario references**: connection-context-assembly.feature: "The token is redacted when the context is rendered", "No credentials still assembles a context carrying the absence" (readiness naming)
  - **Risk**: ⚠️ Secret hygiene — `ConnectionContext` holds the token (inside `Cred`). The redacting `String()` and a "token never appears in `Problems()`/`String()`/`%+v`" test are load-bearing (the `auth.Resolution` precedent); `Problems()` must derive only from safe source/path labels.

## Phase 2: `Assemble` + `AssembleFromOS` aggregator [Shared]

- [ ] **T002** [Shared] Implement the transparent aggregator: call both resolvers once, carry both outcomes, always return a context — plus the production seam — RED-first unit tests
  - **Scope**: In `internal/apiclient`, add the pure `Assemble(resolveBaseURL func() (BaseURL, error), resolveCred func() (auth.Resolution, error)) ConnectionContext`. It calls **both** resolvers exactly once, **does not short-circuit** when the first returns an error (carry-both), packs each `(value, error)` into the matching `ConnectionContext` fields, and returns the context with **no `error` return** — assembly always yields a context, never refuses, decides no exit code, and reads no flag/env/file itself. Both args are required and must be non-nil; a nil resolver is a wiring bug and panics (fail-fast — no nil-default, per DECISIONS/PR #20; document the precondition). Add `AssembleFromOS(flagValue string) ConnectionContext` binding `resolveBaseURL` to `ResolveBaseURLFromOS(flagValue)` and `resolveCred` to `auth.Resolve`, documented as **once per invocation**.
  - **Acceptance criteria**:
    - `Assemble` calls **both** resolvers even when the base-URL resolver returns an error (carry-both) — pinned by a tripwire fake that records invocation and fails if the second resolver isn't called
    - Each base-URL × credential outcome combination is packed correctly into the context: complete; credential-absent; base-URL-error; credential-error; **both** problems present
    - `Assemble` never returns an error; it is deterministic (same resolver outcomes → same context with the same readiness and sources)
    - A nil resolver argument panics (fail-fast); the precondition is documented on the constructor
    - `AssembleFromOS` binds `ResolveBaseURLFromOS`/`auth.Resolve` and delegates to `Assemble`
    - RED-first unit tests over fake resolvers for each outcome combination, the both-resolvers-called tripwire, determinism, and the nil-resolver panic; `go build`/`go vet` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-1 (always-returns-a-context, carry-both, no decide), ADR-2 (carries the token, single resolution point); Cross-cutting (injected seams, fail-fast on nil)
  - **Interface references**: interface-spec.md — Entry points (`Assemble`/`AssembleFromOS`), Interactions (assembly flow, carry-both, determinism), Error Communication (never returns an error; nil panic)
  - **Scenario references**: connection-context-assembly.feature: "A complete context pairs the resolved base URL and token", "A built-in-default base URL with a token still completes", "A base-URL error and an absent credential are surfaced together", "A base-URL error is carried while a present token is kept intact"
  - **Risk**: ⚠️ Carry-both must not short-circuit on the first error — a naive `if err != nil { return }` after the base-URL resolver would skip the credential walk and drop its outcome; pin with the tripwire fake. Resist the nil-default reflex on the injected resolvers (keep fail-fast — PR #20).

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the 009 driving scenarios pass as executable acceptance via godog, driving `Assemble` over fake resolver outcomes in a new suite scoped to this spec's own feature file
  - **Scope**: Add godog step definitions for `features/undefined-connection-settings/connection-context-assembly.feature` (all three Rule blocks), in a **new** `internal/apiclient` godog suite (`TestConnectionContextFeatures`) whose `Paths` names **only** that feature file — the package now has three suites (`TestFeatures` → 007, `TestBaseURLFeatures` → 008, this → 009), each pointed at its specific file (LEARNINGS: a suite must point at its own file, never the `features/` directory). Drive `Assemble` over **fake** base-URL/credential resolver outcomes (and render the context for the redaction scenario); assert complete/incomplete readiness with the named missing/errored part, carry-both (both surfaced, no short-circuit), the token kept intact when the base URL errors, a carried credential error naming the file, redaction, and the assemble-once/reuse behavior. **Reuse existing step phrasing** where an assertion already exists — grep the package's `sc.Step(` registrations before adding new bindings; reserve new phrasings for genuinely new assertions. Remove `@wip` from the 8 behavioral scenarios; keep the 4 `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 009 scenario (complete-context / default+token completes / no-credentials carries absence / both-problems surfaced / credential-error-carried / base-URL-error-with-token / token-redacted / one-context-across-calls) has an executable, passing path
    - `@wip` removed from those scenarios; the four `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `connection-context-assembly.feature`; all three `apiclient` suites run and report their own independent `N scenarios (N passed)` counts
    - No real network and no real home/filesystem are touched — fake resolver outcomes only; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: connection-context-assembly.feature: all 009 behavioral Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — a third feature file in the `apiclient` package must keep every suite pointed at specific files (not the directory), or un-wipping one spec's scenarios breaks another suite; verify all three report their own counts. Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings (LEARNINGS).
