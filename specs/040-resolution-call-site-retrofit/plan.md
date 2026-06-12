# Plan: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Role**: Shaper
**Inputs**: spec.md (040), PROJECT.md, 039 spec.md + plan.md (the resolver this slice consumes), `.score/memory/DECISIONS.md` (precedent: §47 auth `Resolution`/`Source`, §61/§62 base-URL chain + `Source` enum, §74 `internal/rcfile` shared reader + "auth keeps `Resolution`/`Source`", §79/§82 connection-context reads `Cred.Source == SourceNone`, §160/§161/§170 output-format pure-core resolver, §164 `--output`/`-o` persistent root flag, §98 `--base-url` persistent root flag), DEPRECATION.md (no superseding entries in this domain), LEARNINGS.md (passive: cobra `Changed()`-not-value, PRs #61/#73)

---

## System Architecture

Resolution Call-Site Retrofit changes no package boundary and adds no package. It rewrites the *innards* of three existing resolvers to compose the `internal/resolve` walk delivered by 039, while leaving every public output type, provenance enum, and typed error exactly as it is (the surface-stable choice — ADR-1). The three sites:

- **Token** — `internal/auth/resolve.go`: `resolve.Resolve(FromEnv, FromFile)` (no flag rung, no default; `KindNone` → the existing `SourceNone` "nothing found" outcome).
- **Base URL** — `internal/apiclient/baseurl.go`: `resolve.Resolve(FromFlags, FromEnv, FromFile, Default)`.
- **Output format** — `internal/output/format.go`: `resolve.Resolve(FromFlags, FromEnv, FromFile, Default)`.

In each, the generic `resolve.Resolution{Value, Provenance{Kind, Origin}}` is mapped back onto the site's existing code-free output (`auth.Resolution{Token,Source,Path}`, `apiclient.BaseURL{Value,Source,Path}`, `output.OutputFormat`) and its typed errors (`*BaseURLError`, `*FormatError`, rcfile's typed errors). The setting-specific value validators (`isUsableURL`, `ParseFormat`) move to run on the *resolved winner* (ADR-3), since `resolve` is setting-agnostic.

This slice carries one deliberate behavioural change: the base-URL and output-format flag rungs adopt `resolve.FromFlags`' **presence** semantics (cobra `Changed()`) in place of today's value-emptiness proxy (ADR-2). That presence bit does not exist at the resolvers today, so it is threaded from the command layer down — the slice's largest mechanical surface.

```
RunE (12 read commands)                          internal/apiclient            internal/resolve
  flag := GetString(FlagBaseURL)
  present := Changed(FlagBaseURL) ──▶ AssembleFromOS(flag, present) ─▶ ResolveBaseURL(Flag{present,flag}, dirs)
                                                                          └▶ resolve.Resolve(FromFlags,FromEnv,FromFile,Default)
                                       map Provenance → BaseURL/BaseURLError      ◀── Resolution{Value,Provenance}
  flag := GetString(FlagOutput)
  present := Changed(FlagOutput) ───▶ seam.resolveFormat(flag, present) ─▶ output.ResolveFormat(Flag{...}, dirs)

  (token: auth.Resolve() — no flag, signature unchanged)
```

**Boundaries this plan crosses** (drive the interface step): (1) a **code-API boundary** — the base-URL/output resolver signatures and the `internal/cli` `meSeam.resolveFormat` contract gain a flag-presence input; (2) a **CLI behaviour boundary** — an explicitly-supplied empty/whitespace `--base-url`/`--output` now fails loud. No new endpoint, no new command, no new file.

---

## Architecture Decisions

### ADR-1: Surface-stable retrofit — keep the per-domain output types and `Source` enums, map `resolve.Provenance` back (diverges from 039 ADR-2's forecast)

**Context**: 039 ADR-2 forecast that 040 would *remove* the three divergent per-domain `Source` enums in favour of one uniform `resolve.Provenance` ("their removal is 040's work"). The 040 spec instead scopes a behaviour-preserving retrofit that keeps every public surface. The recorded precedent supports preservation: DECISIONS §74 ratifies "`internal/auth` keeps the token domain (`Resolution`, `Source`, …)"; 039 itself recorded no DECISIONS entry, so its "remove the enums" line is an unratified forecast.

**Options considered**:
1. **Unify on `resolve.Provenance` (honour 039's forecast)** — delete `auth.Source`/`apiclient.BaseURLSource`, expose `resolve.Provenance` from the resolvers. But this ripples into `internal/apiclient` consumers: `Identity` embeds `auth.Source` (transport.go), and `ConnectionContext.Complete()`/`Ready()`/status rendering branch on `Cred.Source == auth.SourceNone` (context.go, auth.go). 039's own risk register flags "behavioural drift during 040's retrofit" as high-impact, and this is exactly the readiness logic it warns about.
2. **Surface-stable — keep the per-domain types, derive them from `resolve` internally** — the resolvers compose `resolve.Resolve` and map `resolve.Provenance.Kind` → their existing enum members. Consumers (007/009/010) are untouched.

**Decision**: Option 2. Each resolver becomes a thin adapter over `resolve`: compose sources, take the winner, map provenance back onto the existing code-free type. The per-domain enums survive as the consumer-facing vocabulary; `resolve.Provenance` is an internal intermediate, never exposed past the three resolver functions. This was confirmed with the developer during planning.

**Consequences**: The type-change blast radius is confined to the three resolver files; the connection-readiness logic is untouched, sidestepping the high-impact drift risk. The cost is that 039's "one uniform provenance vocabulary" end-state is not reached — the enum-unification remains available as a later cleanup spec. This is an **announced divergence** from 039 ADR-2's forecast; a `/score:deprecate` note is suggested at handoff so the forecast is formally retired rather than left dangling.

### ADR-2: Flag presence via cobra `Changed()` replaces value-emptiness at the base-URL and output-format flag rungs

**Context**: Today `ResolveBaseURL`/`ResolveFormat` receive only `flagValue string` and treat `strings.TrimSpace(flagValue) != ""` as "the flag was supplied" — so `--base-url ""` or `--output "  "` is silently treated as absent and falls through. `resolve.FromFlags` is presence-based (cobra `Changed()`; an explicit `--flag=` counts). LEARNINGS records the `Changed()`-not-value rule as re-violated in PRs #61/#73. The developer chose to fix the inconsistency in this retrofit (spec decision B).

**Options considered**:
1. **Preserve value-emptiness** — construct `resolve.Flag{Present: strings.TrimSpace(flagValue) != ""}`. Zero observable change, but bakes the value-as-presence quirk into the new wiring and contradicts `resolve`'s documented flag semantics.
2. **Presence-based** — thread the real `cmd.Flags().Changed(name)` bit and construct `resolve.Flag{Name, Present, Value}`. An explicitly-supplied empty/whitespace flag now yields and is validated (failing loud); an unsupplied flag falls through.

**Decision**: Option 2. The presence bit is threaded from each `RunE` (`cmd.Flags().Changed(apiclient.FlagBaseURL)` / `Changed(output.FlagOutput)`) through `AssembleFromOS` and the `meSeam.resolveFormat` contract into the resolvers' `FromFlags`. `Changed()` reports correctly on these inherited persistent root flags (verified: the same `cmd.Flags()` flagset already serves `GetString` for them).

**Consequences**: One observable behaviour change — an explicit empty/whitespace `--base-url`/`--output` fails loud instead of falling through (covered by edge-case scenarios in the spec). The presence threading is the slice's broadest mechanical change: `AssembleFromOS`/`ResolveBaseURLFromOS`/`ResolveBaseURL` and `ResolveFormatFromOS`/`ResolveFormat` gain a presence input, the `meSeam.resolveFormat` signature changes across its 10 declarations + production impl, and the 12 read-command `RunE`s pass `Changed()`. The token path is unaffected (no `--token` flag).

### ADR-3: Validation runs on the resolved winner at the call site; the default is never validated

**Context**: 039 ADR-3 relocated value validation from the resolver to the call site, observing that the observable present-but-invalid-fails-loud behaviour (§170) is preserved by construction because `resolve` returns the first *yielding* (not first *valid*) source. 040 is where that relocation actually happens, and 039's risk register names it the load-bearing change for regression.

**Options considered**:
1. **Validate inside a wrapper before mapping** — fine, but must replicate the per-rung "skip the default" carve-out and the no-fall-through guarantee carefully.
2. **Validate the single returned winner, keyed off `Provenance.Kind`** — `resolve.Resolve` already short-circuits at the first yield, so the returned value *is* the winner; validate it unless `Kind == KindDefault` (valid by construction).

**Decision**: Option 2. After `resolve.Resolve` returns (nil error), the call site runs `isUsableURL(res.Value)` / `ParseFormat(res.Value)` when `res.Provenance.Kind != KindDefault`; on failure it returns `&BaseURLError{Source: res.Provenance.Origin}` / `&FormatError{Source: res.Provenance.Origin, Value: res.Value}`. `resolve.Provenance.Origin` supplies exactly the existing label forms — `--base-url`/`GLASSFROG_BASE_URL`/file-path and `--output`/`GLASSFROG_OUTPUT`/file-path — because the call site passes those names into `FromFlags`/`FromEnv` and `FromFile` sets Origin to the resolved path (039's tests pinned these labels for this purpose).

**Consequences**: No fall-through past a malformed winner (the walk already stopped; the caller rejects rather than re-walking). The default is returned unvalidated, matching "valid by construction, never re-validated." A resolution error from `resolve` (rcfile typed `*ReadError`/`*FormatError`, surfaced verbatim with no fall-through) is returned before any value validation, preserving the two distinct error classes (§170, 039 cross-cutting).

### ADR-4: Each call site keeps its own OS seam and feeds it into `resolve`'s injected constructors

**Context**: Each resolver owns a package-level `var getenv/getwd/userHomeDir` seam so its tests run hermetically over temp dirs and a controlled environment, never the developer's real `~/.glassfrogrc` (§49, §71, CONSTITUTION IV). 039 also provides `…FromOS` binding helpers. The retrofit could route through either.

**Options considered**:
1. **Route through `resolve`'s own `FromOS` binding helpers** — fewer seam vars, but moves the OS-binding ownership out of each domain package and would make the secret-bearing token path depend on `resolve`'s binding surface.
2. **Keep each site's existing seam; pass it into the injected constructors** — `resolve.FromEnv(getenv, name)`, `resolve.FromFile(startDir, homeDir, key)` with `startDir`/`homeDir` derived by the site's existing `getwd`/`userHomeDir`, exactly as today.

**Decision**: Option 2. The `…FromOS` entrypoints (`auth.Resolve`, `ResolveBaseURLFromOS`, `ResolveFormatFromOS`) keep deriving the real dirs/env through their own seam vars and pass them into `resolve`'s constructors; `resolve.FromFile` delegates to the same `rcfile.Resolve` the sites call today. `resolve`'s `FromOS` helpers are not used by these three sites.

**Consequences**: The existing hermetic test pattern (set the package `getenv` var, point at a temp-dir rcfile) carries forward with minimal churn; no test reads the real home. The token path gains no dependency on `resolve`'s binding helpers, keeping the secret concern self-contained. The exact testable-seam shape per site (e.g. whether `output`'s pre-fetched-source pure core is retained or folded) is interface-level.

---

## Cross-cutting Concerns

**Secret hygiene**: `resolve.Resolution` has no redacting `String()`; the token path maps `res.Value` into `auth.Resolution.Token` (which *does* redact via its `String()`) and never formats or logs the intermediate `resolve.Resolution`. `resolve.FromFile` over the `token` key returns only that key's value (rcfile, §74), so a non-token resolution never possesses the token. Provenance (`Kind`/`Origin`) is safe to display by construction.

**Error handling**: Three classes stay distinct and behaviour-equivalent to today. *Resolution errors* (unreadable/malformed `.glassfrogrc`) propagate verbatim from `resolve.Resolve` and abort before validation, no fall-through (§170). *Value-invalidity* is the call site's `*BaseURLError`/`*FormatError` on the winner (ADR-3). *Wiring errors* (multiple STDIN) cannot occur — no site composes a STDIN source.

**Testing strategy**: Each resolver carries its existing suite forward green (the behaviour-preservation contract), plus: new edge-case tests for the presence change (explicit empty/whitespace `--base-url`/`--output` → typed error; unsupplied flag → fall-through); provenance-mapping tests (each `resolve.Kind` → the right per-domain enum member and label); and the no-fall-through-on-malformed-winner regression. cli-layer tests assert each `RunE` passes the real `Changed()` bit. Tests stay hermetic (temp dirs, controlled env) per ADR-4.

**Layering**: No import direction changes. The three domain packages add an import of `internal/resolve` (leaf → leaf, no cycle — 039 ADR-1); `internal/apiclient` and `internal/output` keep their existing imports; `internal/cli` still imports `apiclient`/`output` and now reads `Changed()` (a cobra call it already makes for sibling flags).

---

## Implementation Strategy

Three independent retrofits sharing one mechanism. Each ships as its own reviewable PR carrying the relevant test suite forward green. All three depend only on 039 (landed). Phases 2 and 3 both edit the read-command `RunE` files (different lines — base-URL vs. output plumbing); that shared-edit surface is a merge-order coordination point, not a hard dependency, so they may land in either order.

**Phase 1 — Token retrofit (`internal/auth`)**: Rewrite `resolve(startDir, homeDir)` to `resolve.Resolve(FromEnv(getenv, envTokenVar), FromFile(startDir, homeDir, tokenKey))`; map `KindEnv→SourceEnvironment`, `KindFile→SourceFile`(+Path), `KindNone→SourceNone`. `auth.Resolve()`'s signature is unchanged; no cli plumbing. Lowest risk, fully self-contained — good first slice.

**Phase 2 — Base-URL retrofit + presence threading (`internal/apiclient`, `internal/cli`)**: Rewrite `ResolveBaseURL` to compose `resolve` with a `FromFlags(Flag{Name:"--base-url", Present, Value})` rung; relocate `isUsableURL` to the winner (ADR-3); map provenance → `BaseURL`/`BaseURLError`. Thread presence: `ResolveBaseURL`/`ResolveBaseURLFromOS`/`AssembleFromOS` gain the presence input; each read command's `RunE` passes `cmd.Flags().Changed(apiclient.FlagBaseURL)`.

**Phase 3 — Output-format retrofit + presence threading (`internal/output`, `internal/cli`)**: Rewrite `ResolveFormat`/`ResolveFormatFromOS` to compose `resolve` with a presence-based `FromFlags` rung (single `--output` flag; cobra merges `-o`, so one Flag, Origin `--output` — matching today's label); relocate `ParseFormat` to the winner; map provenance → `OutputFormat`/`FormatError`. Update the `meSeam.resolveFormat` contract (10 declarations + production impl) and the `RunE`s to pass `Changed(output.FlagOutput)`.

---

## Risks

- **Behavioural drift in validation relocation** (medium likelihood, high impact — inherited from 039): moving `isUsableURL`/`ParseFormat` to the winner could regress the no-fall-through-on-malformed guarantee or accidentally validate the default. Mitigation: carry each existing resolver suite forward green and add the explicit no-fall-through-on-malformed-winner + default-unvalidated regression tests before refactoring.
- **Provenance-label mismatch breaks error text** (low likelihood, medium impact): if a passed-in `FromFlags`/`FromEnv` name or `FromFile`'s path doesn't reproduce the exact `*BaseURLError`/`*FormatError` `Source` label, error messages change. Mitigation: 039's tests already pinned the three label forms; assert the mapped labels against the pre-retrofit strings.
- **`Changed()` semantics on inherited persistent flags** (low likelihood, medium impact): if `cmd.Flags().Changed(name)` reported differently from `GetString(name)` for inherited persistent root flags, presence would be wrong. Mitigation: verified the same flagset serves both; a `RunE`-level test asserts presence for supplied / unsupplied / `--flag=` cases.
- **Incomplete presence threading** (medium likelihood, medium impact): 12 `RunE` sites + the 10-declaration seam contract are a broad mechanical surface; a missed site silently keeps value-emptiness behaviour. Mitigation: change the resolver/seam signatures first so the compiler flags every un-threaded call site (no default-value overload that would let an old call compile).

---

## What This Plan Does Not Cover

- **Exact Go signatures and the per-site testable-seam shape** — the precise spelling of the presence parameter (a `resolve.Flag`, a `(value, present)` pair, or a small struct), and whether `output`'s pre-fetched-source pure core is retained or folded. These are the interface skill's concern (this is a code-API boundary feature).
- **Executable scenarios** — the scenarios skill concretizes the spec's driving scenarios into `.feature` files.
- **Task decomposition** — the tasks skill breaks the three phases into PR-sized units.
- **The enum-unification end-state** — removing the per-domain `Source` enums in favour of `resolve.Provenance` (039 ADR-2's forecast) is explicitly deferred (ADR-1); it is a candidate for a later cleanup spec and a `/score:deprecate` note.
