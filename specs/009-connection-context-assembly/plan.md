# Plan: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Role**: Shaper
**Inputs**: spec.md (009-connection-context-assembly); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: `internal/apiclient` is Connection Configuration's home — 008 ADR-1; `BaseURL{Value,Source,Path}` + `BaseURLError` — 008; `auth.Resolution{Token,Source,Path}` code-free shape + `SourceNone`-is-absence — 005; 007's `AuthTransport` wraps Connection Configuration's base transport and consumes an injected `func() (auth.Resolution, error)`, resolve-once via `sync.Once`, typed `AuthError{NoCredentials|CredentialError}`, fail-safe refusal — 007; "producer classifies a code-free outcome, consumer maps it" — 002/004/005/007/008; "inject env/fs seams, the resolver is pure" — 005/008 ADR-5; the generic `.glassfrogrc` walk lives in `internal/rcfile` — 2026-06-06 refactor); `.score/memory/LEARNINGS.md` (relevant: a struct that carries a secret must implement a redacting `String()`; nil-default-vs-fail-loud for injected seams — PR #20; a godog suite must point at its own feature file). No SOUL.md. No DEPRECATION.md (no deprecation filtering applied).

**Readiness**: Must met + Should substantial (integration boundaries name specific systems, user scenarios present, assumptions flagged, error/edge scenarios beyond happy path). Strong foundation. No architectural unknowns required a resolve conversation — package placement, the resolver pattern, the code-free outcome shape, and the secret-hygiene discipline are all fixed by DECISIONS precedent. The one genuine design question — the 007/009 `[ASSUMED]` seam (does 007 read the token from this context or re-resolve from Discovery?) — is settled here as a Shaper decision (ADR-2), determined by the spec's "resolved once per invocation" and the existing `AuthTransport` shape.

---

## System Architecture

Connection Context Assembly is the **consuming half of Connection Configuration** — the half 008 explicitly deferred ("combine base URL + token … is a deferred later spec"). 008 produced `BaseURL{Value,Source,Path}`; 005 produced `auth.Resolution{Token,Source,Path}`; 007 built the `AuthTransport` round-tripper that attaches the token and owns the fail-safe refusal. This slice adds the **single bundle** that pairs those two resolved outcomes — the connection context every request hangs off — and reports whether that bundle is complete, **without deciding anything**.

It is a **transparent aggregator** (spec-confirmed: option A + carry-both). It calls 008's base-URL resolver and 005's credential resolver each exactly once, captures each `(value, error)` outcome verbatim, and packs both — successes *and* errors — into one `ConnectionContext` value. It always produces a context: a missing or broken base URL or credential is carried *inside* the context, never a reason to short-circuit or to fail assembly. It re-resolves nothing (it calls the resolvers; it reads no flag, env, or file itself), transforms nothing (no URL normalization, no header construction), makes no network call, writes nothing, prints nothing, and decides no exit code. The refuse-to-call fail-safe stays where 007 already owns it.

The parts (all additive in `internal/apiclient` — this slice changes no existing file):

- **`ConnectionContext`** — a code-free value object carrying the base-URL outcome (`BaseURL` + a typed base-URL error, one of which is set) and the credential outcome (`auth.Resolution` + a typed credential error). The `auth.Resolution` holds the token, so `ConnectionContext` carries a secret and gets a value-receiver redacting `String()` (LEARNINGS) — its readiness reasons name only safe-to-display source/path strings, never the token.
- **`Assemble(resolveBaseURL, resolveCred)`** — the pure aggregator. Injected resolver funcs (`func() (BaseURL, error)` and `func() (auth.Resolution, error)`); it calls **both** (even when the first errors — carry-both), and returns a `ConnectionContext`. No `error` return: assembly always yields a context.
- **`AssembleFromOS(flagValue)`** — the thin production seam binding the real resolvers (`ResolveBaseURLFromOS(flagValue)` and `auth.Resolve`), parallel to `auth.Resolve` / `ResolveBaseURLFromOS`.
- **Readiness** — derived, not stored: the context is *complete* iff the base URL is present (no base-URL error) **and** a token is present (`Cred.Source != SourceNone`) **and** there is no credential error; otherwise *incomplete*, naming which part is missing or errored. Reporting only — the consumer decides what to do.

```
internal/apiclient                          (Connection Configuration; 008 added the base-URL half, this adds the assembly half)

  AssembleFromOS(flagValue)                            ── production seam
    └─ Assemble(
         func(){ ResolveBaseURLFromOS(flagValue) },    ── 008's resolver (flag→env→file→default)
         auth.Resolve,                                 ── 005's resolver (env→file→none)
       ) → ConnectionContext                           ── always returns; carries both outcomes

  ConnectionContext{ BaseURL, BaseURLErr, Cred, CredErr }
       · Complete()  = BaseURLErr==nil && CredErr==nil && Cred.Source != SourceNone
       · reasons()   = safe-to-display labels for each missing/errored part (never the token)
       · String()    = redacting (Cred.Token never rendered)

  consumed by ──► Request Execution (010, future): builds the base http transport rooted at
                  ctx.BaseURL.Value and wraps it in 007's AuthTransport, feeding AuthTransport a
                  resolver that REPLAYS ctx's cached credential — NOT a fresh Discovery walk (ADR-2).
                  007's fail-safe then refuses at request time on absent/errored credential.
```

The connection context is the **single resolution point** for an invocation: both walks happen once, here. 007's `AuthTransport` is built (by 010) over a replay of this context's cached credential, so "the same identity and endpoint apply to every call" becomes structural, and no second resolution can drift from the first.

---

## Architecture Decisions

### ADR-1: `ConnectionContext` is a code-free value object; `Assemble` is a transparent aggregator that always returns a context and carries both sub-outcomes

**Context**: The spec confirms (option A + carry-both) a transparent aggregator: it combines 008's base-URL outcome and 005's credential outcome, carries both forward — including errors and absence — reports readiness, but does **not** refuse a request, decide an exit code, or re-resolve. 008 deferred this "connection-context half" to a later spec; this is it. The producer-classifies/consumer-maps split (002/004/005/007/008) and Fail-Safe (CONSTITUTION III — surface loudly, don't degrade) both apply.

**Options considered**:
1. **Value object + pure aggregator that always returns a `ConnectionContext`** — `Assemble` calls both injected resolvers once, packs each `(value, error)` verbatim, and returns a context with no `error` of its own; problems live inside the context. Readiness is a derived view. Mirrors 005/008's code-free outcome shape.
2. **Aggregator that returns `(ConnectionContext, error)`, erroring when a part is missing/broken** — rejected: it would make assembly *decide* that an incomplete bundle is a failure, duplicating 007's fail-safe and splitting that contract across two capabilities (spec Non-Behavior). It also breaks carry-both — an early base-URL error would mask an also-absent credential.
3. **A gate that refuses to build a context without a token** — rejected: the spec's option-B, explicitly declined; it duplicates 007's refusal decision.

**Decision**: Option 1. `Assemble(resolveBaseURL, resolveCred) ConnectionContext` invokes **both** resolvers (no short-circuit on the first error), stores `BaseURL`+`BaseURLErr` and `Cred`+`CredErr`, and returns the context. Readiness is computed on demand from those fields. Assembly never returns an error, never refuses, never maps an exit code.

**Decision works in practice**: a base-URL error and an absent credential coexist in one context, both reportable in a single pass (spec error scenario "both inputs report a problem"). The consuming command reads the context, asks `Complete()`, and — if incomplete — surfaces the named reasons and lets 007's fail-safe / Exit-Code Convention classify. Client assembly (the base `http` transport rooted at `BaseURL.Value`, wrapped in `AuthTransport`) is **not** here — it moves to Request Execution (010). This refines 008's forecast, which bundled "assemble the `http.Client`" into the connection-context half; assembly produces the *context value*, 010 builds the *client* from it.

**Consequences**: A trivially-testable pure function with no I/O. Callers never handle an "assembly failed" path — only a "context incomplete" view. The context's `Complete()`/reasons API shape is interface-level (deferred to `/score:interface`). Because the context holds `auth.Resolution` (a secret), it must redact on render (Cross-cutting). *Precedent-setting.*

### ADR-2: The context carries the full credential outcome (token included) and is the single resolution point; Request Execution replays it into 007's `AuthTransport` rather than re-resolving — resolving the 007/009 `[ASSUMED]` seam

**Context**: Both 007 and this spec recorded an `[ASSUMED]` seam: does Request Authentication read the token from the connection context, or directly from Credential Discovery? The spec says the context "combines the resolved base URL **with the discovered token**" and that one context is "assembled once per invocation and reused." 007's `AuthTransport` already takes an injected `func() (auth.Resolution, error)` and caches the first result via `sync.Once`.

**Options considered**:
1. **Context carries the full `auth.Resolution` (incl. token); 010 wires `AuthTransport` with a resolver that replays the context's cached credential** — one resolution per invocation, at assembly; `AuthTransport` consumes the already-resolved outcome. The seam between 007 and Connection Configuration becomes: 007 reads from the context.
2. **Context carries only a safe `Identity` (Source/Path, no token); `AuthTransport` re-resolves from Discovery at request time** — rejected: two independent walks (one for the context's readiness, one for the token) can drift, and it contradicts "resolved once per invocation"; it also re-reads the filesystem after assembly already did.
3. **Context carries no credential at all; readiness is base-URL-only** — rejected: the spec explicitly pairs the token into the context and reports credential presence/absence as part of readiness.

**Decision**: Option 1. `ConnectionContext` carries `Cred auth.Resolution` (and `CredErr error`). The context is the single resolution point. Request Execution (010) will build `NewAuthTransport(baseTransport, func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr })` — `AuthTransport`'s existing `authorize` then yields the token on the present branch and the typed `AuthError` (`NoCredentials` / `CredentialError`) on the absent/errored branches, so the **fail-safe refusal still fires in 007**, at request time, exactly as before. This slice writes only the context + assembler; the wiring lives in 010, so 007's code is untouched here.

**Consequences**: Resolution happens once; identity and endpoint are stable across every call by construction. The token lives in the context — hence the redacting `String()` (Cross-cutting). 007's `AuthTransport` needs no change: a replay thunk satisfies its injected-resolver contract, and its `sync.Once` simply caches an already-final value. **Resolves the 007/009 `[ASSUMED]` seam** (consider `/score:deprecate` to retire the open seam note). *Precedent-setting: the connection context is the single resolution point; 010 replays it into `AuthTransport`; client assembly is 010's, not the context's.*

---

## Integration Design

- **Base URL Resolution (008, `internal/apiclient` — upstream)**: `Assemble` calls `ResolveBaseURLFromOS(flagValue)` (prod) / an injected `func() (BaseURL, error)` (test). A `*BaseURLError` or an `internal/rcfile` read/format error is captured into `BaseURLErr` and carried, never retried. The `--base-url` flag *value* is threaded through `AssembleFromOS(flagValue)`; its cobra registration remains the future command's job.
- **Credential Discovery (005, `internal/auth` — upstream)**: `Assemble` calls `auth.Resolve` (prod) / an injected `func() (auth.Resolution, error)` (test). `Source: None` (absent) and a typed read/format error are both carried verbatim — absence is not an error here.
- **Request Authentication (007, `internal/apiclient` — sibling consumer)**: consumes the context's credential at request time via the replay resolver 010 wires (ADR-2). 007 keeps the fail-safe refusal and the `AuthError` taxonomy; this slice neither attaches the header nor refuses.
- **Request Execution / API Client (010, future — downstream)**: builds the base `http` transport rooted at `ctx.BaseURL.Value`, wraps it in `AuthTransport`, and sends. Owns the `http.Client` assembly this plan deliberately excludes.
- **Exit-Code Convention (004 — downstream)**: an incomplete context informs a non-zero exit code, but the classification belongs to the consuming command. Same open gap 007/008 flagged: 004 has no local "bad configuration / cannot-connect" code (code 4 is API-side). This slice surfaces a code-free readiness view only.

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: `ConnectionContext` carries `Cred.Token`. Following the established secret-bearing-struct rule (LEARNINGS, `auth.Resolution.String()`), it gets a **value-receiver `String()`** that renders the base-URL source, the credential `Source`/`Path`, and readiness, but reports the token as present/absent — never verbatim — so `%v`/`%+v`/`%s` cannot leak it. Readiness reasons are built from `BaseURLError.Source` (a label) and the credential `Source`/`Path` (path-only by the 005/007 contract) — never the token. Pinned by a test asserting `%+v`/`String()` omit the token but keep the safe parts.

**Error handling (CONSTITUTION III)**: assembly itself cannot fail — it always returns a context. The *carried* errors fail loud at their own producers (008's malformed-URL `BaseURLError`, 005/rcfile's read/format errors); this slice neither swallows nor re-wraps them, only surfaces them through the context and the readiness reasons. Fail-Safe (refuse to call) stays in 007.

**Testing (CONSTITUTION IV)**: RED-first, hermetic. `Assemble` is pure over its two injected resolver funcs, so every combination is table-tested with **no I/O**: complete; credential-absent → incomplete naming credential; base-URL-error → incomplete naming base URL; both-errored → both reasons present (carry-both); determinism (same inputs → same context). A **tripwire** resolver that records invocation pins "both resolvers are always called" even when the first returns an error (the negative property the carry-both behavior depends on — LEARNINGS: assert non-events with a tripwire, not just output). Redaction is pinned with a token-in-context format test. The driving scenarios become a godog suite **scoped to this spec's own feature file** (LEARNINGS) — `features/undefined-connection-settings/connection-context-assembly.feature`, a third suite in `internal/apiclient` beside `TestBaseURLFeatures` and `TestFeatures`.

**No command surface (this slice)**: like 005/007/008, this capability registers no cobra command and prints nothing; the cobra LEARNINGS findings do not apply. The `--base-url` flag value is an injected input via `AssembleFromOS`.

**Resolve-once discipline**: "one context per invocation" is a calling-convention guarantee — the command layer calls `AssembleFromOS` once and threads the context. Documented on `AssembleFromOS`; `AuthTransport`'s own `sync.Once` backstops accidental repeat resolution at the request layer.

---

## Implementation Strategy

Three phases, linear. Depends only on landed code: 008 (`ResolveBaseURLFromOS`, `BaseURL`, `BaseURLError`) and 005 (`auth.Resolve`, `auth.Resolution`). No existing file is modified — the slice is purely additive (a new `context.go` + tests in `internal/apiclient`).

- **Phase 1 — `ConnectionContext` + readiness + redacting `String()`**: define the value object (`BaseURL`/`BaseURLErr`, `Cred`/`CredErr`), the derived `Complete()` and the safe readiness-reasons accessor, and the value-receiver redacting `String()`. RED-first unit tests: complete, each single-part-incomplete, both-incomplete, redaction (token never in `%+v`/`String()`/reasons). *Depends on: 005/008 types (landed).*
- **Phase 2 — `Assemble` + `AssembleFromOS`**: the pure aggregator over two injected resolver funcs (calls both, carry-both, always returns a context) and the thin production seam binding `ResolveBaseURLFromOS(flagValue)` + `auth.Resolve`. RED-first unit tests over fake resolvers for every base-URL × credential outcome combination, plus the determinism case and the **both-resolvers-called tripwire** (even when the base-URL resolver errors). Fail-fast on nil resolvers (no nil-default — LEARNINGS/PR #20). *Depends on: Phase 1.*
- **Phase 3 — executable acceptance**: godog step definitions for the driving scenarios (complete; built-in-default + token still complete; one context across multiple calls; no-credentials carries absence; both-problems surfaced; base-URL error with token present; token redacted), in a new `internal/apiclient` godog suite pointed at this spec's own feature file. Reuse existing step vocabulary where the assertion already exists (LEARNINGS — grep the package's `sc.Step(` registrations before adding new phrasings). *Depends on: Phase 2.*

---

## Risks

- **Seam mismatch with future Request Execution (010)** (medium likelihood, medium impact): if 010 re-resolves the credential from Discovery instead of replaying the context, two walks can drift and "resolved once" breaks. Mitigation: ADR-2 pins replay-from-context and the context carries the full `auth.Resolution`; record it as precedent so 010 inherits the seam.
- **Secret leakage through the context** (low likelihood, high impact): the context holds the token, one `%+v` away from a log line. Mitigation: value-receiver redacting `String()` and a format test (the `auth.Resolution` precedent); readiness reasons are source/path-only.
- **A readiness reason leaks a token-bearing string** (low likelihood, high impact): reasons must be built only from safe labels. Mitigation: reasons derive from `BaseURLError.Source` and credential `Source`/`Path` (path-only by contract), never from `Cred.Token`; pinned by test.
- **Exit code for an incomplete context is undefined** (medium likelihood, low impact): 004 has no "cannot connect / bad config" code. Mitigation: stay code-free (readiness view only); the classification is the consuming-command's decision — the same open gap 007/008 flagged. Does not block this slice.
- **Spec's "one context per invocation" relies on caller discipline** (low likelihood, low impact): repeated `AssembleFromOS` calls would re-resolve. Mitigation: document the once-per-invocation convention; `AuthTransport`'s `sync.Once` backstops the request layer.

---

## What This Plan Does Not Cover

- **Building the `http.Client` / base transport** rooted at the resolved base URL and wrapping 007's `AuthTransport` — Request Execution (010). This plan refines 008's forecast: assembly produces the context *value*; 010 builds the *client* from it.
- **Wiring `AuthTransport`'s resolver to the context** — the replay seam (ADR-2) is *specified* here but *realized* in 010; 007's code is untouched by this slice.
- **The exact `ConnectionContext` API** — field names, the `Complete()`/reasons method signatures, the readiness type — the interface skill (`/score:interface`) pins these.
- **Executable Gherkin** — the scenarios skill turns the driving scenarios into the feature file.
- **The `--base-url` flag registration and the command that triggers API calls** — a future command spec; this slice accepts the flag value as a seam input.
- **The exit code for an incomplete context** — the consuming command + Exit-Code Convention (004); this slice surfaces a code-free readiness view only.
