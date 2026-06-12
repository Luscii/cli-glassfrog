# Risk: Resolution Call-Site Retrofit

**Feature**: 040-resolution-call-site-retrofit
**Round**: 1
**Date**: 2026-06-12
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, interface-cli.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light (no project-level matrix found in PROJECT.md)
**Degradation flags**: none — full upstream artifact set present

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | Validation-relocation regression: moving `isUsableURL`/`ParseFormat` to the winner changes which values pass/fail, validates the default, or validates before resolution | plan ADR-3, Risks | Medium | Medium | Yellow | RC-1 | Green |
| H-2 | Provenance-label mismatch: `Origin` doesn't reproduce the exact `*BaseURLError`/`*FormatError` source labels, drifting operator-facing error text | plan Risks; interface-spec mapping | Low | Low | Green | RC-2 | Green |
| H-3 | Incomplete presence threading: a missed `RunE`/seam site keeps value-emptiness, so the flag-semantics fix is inconsistent across commands | plan Risks; interface-spec RunE plumbing | Medium | Low | Green | RC-3 | Green |
| H-4 | `Changed()` misreports presence on inherited persistent flags: an empty `--base-url`/`--output` not detected (silent fall-through) or an unsupplied flag falsely "supplied" (spurious fail-loud) | plan Risks; interface-cli | Medium | Low | Green | RC-4 | Green |
| H-5 | Token secret leak via the generic `resolve.Resolution` (no redacting `String()`) if the token path formats/logs the intermediate before mapping to `auth.Resolution` | plan Cross-cutting (Secret hygiene); spec | **High** | Low | Yellow | RC-5 | Yellow |
| H-6 | Connection-readiness drift: a wrong token `KindNone→SourceNone` mapping makes `ConnectionContext.Complete()`/`Ready()` misjudge "authenticated" | plan ADR-1; apiclient consumers (context.go) | Medium | Low | Green | RC-6 | Green |

No residual **Red** risks. One residual **Yellow** (H-5) — acceptable with the documented justification below.

---

## Hazard Detail

### H-1 — Validation-relocation regression
**Description**: 039 ADR-3 moved validation out of the resolver; 040 relocates `isUsableURL`/`ParseFormat` to run on the resolved winner. A regression could accept a malformed value (→ a doomed/wrong API call) or reject a valid one (→ a blocked command), or validate the default (which is valid by construction and must not be re-validated).
**Severity — Medium**: blast radius is a single command invocation; outcomes are an operator-visible connection/usage error, recoverable, not data loss.
**Probability — Medium**: this is the load-bearing change 039's risk register flagged; the relocation is hand-written per site.
**Controls**: **RC-1** — each resolver carries its existing suite forward green, plus explicit regression tests for no-fall-through-on-malformed-winner and default-unvalidated (tasks T002/T003 acceptance criteria). Reduces probability to Low → **Green**.

### H-2 — Provenance-label mismatch
**Description**: the typed errors phrase their `Source` from `Provenance.Origin`; if a passed-in flag/env name or the file path doesn't reproduce `--base-url`/`GLASSFROG_BASE_URL`/`--output`/`GLASSFROG_OUTPUT`/path exactly, operator-facing error text drifts (an Action-Transparency concern).
**Severity — Low**: cosmetic error-text change; the failure is still surfaced and recoverable.
**Probability — Low**: 039's tests already pinned the three label forms for this purpose.
**Controls**: **RC-2** — assert the mapped labels against the pre-retrofit error strings (tasks T002/T003). Residual **Green**.

### H-3 — Incomplete presence threading
**Description**: the presence bit threads through every read-command `RunE` (13 today) + all the `resolveSelection` seam declarations (11 today); a missed site would silently keep value-emptiness, so `--base-url ""` fails loud on some commands but falls through on others.
**Severity — Medium**: inconsistent cross-command behaviour → operator confusion; not data loss.
**Probability — Low**: the signature change is deliberately overload-free (plan Risks), so every un-threaded call site fails to compile.
**Controls**: **RC-3** — overload-free signatures (compiler-enforced completeness) + a `RunE`-level presence test. Residual **Green**.

### H-4 — `Changed()` misreports presence on inherited persistent flags
**Description**: the behaviour change depends on `cmd.Flags().Changed(name)` being correct for the inherited persistent root flags. If it under-reports, an explicit empty flag silently falls through (the bug 040 set out to fix persists); if it over-reports, an unsupplied flag spuriously fails loud, blocking legitimate default use.
**Severity — Medium**: a spurious fail-loud blocks a valid invocation; a missed presence reverts to the prior silent behaviour. Recoverable, single-invocation.
**Probability — Low**: the same `cmd.Flags()` flagset already serves `GetString` for these flags, so `Changed` reads the same parsed state.
**Controls**: **RC-4** — `RunE`-level tests asserting presence for supplied / unsupplied / `--flag=` / flag-on-either-side-of-subcommand (tasks T002/T003). Residual **Green**.

### H-5 — Token secret leak via the generic `resolve.Resolution`
**Description**: `auth.Resolution` redacts its token in `String()`, but the generic `resolve.Resolution` (the new intermediate the token path now flows through) has **no** redacting `String()`. If that intermediate is ever formatted (`%v`/`%+v`), logged, or placed in an error before being mapped into `auth.Resolution.Token`, the credential leaks (CONSTITUTION II secret hygiene).
**Severity — High**: credential exposure is the worst outcome in this domain — it undermines the whole auth model.
**Probability — Low**: the token path maps `res.Value` straight into `auth.Resolution` and never formats the intermediate; `resolve` itself emits no diagnostics (039 ADR-3).
**Controls**: **RC-5** — the token path maps `resolve.Resolution.Value` into `auth.Resolution.Token` immediately and never formats/logs the intermediate; `auth.Resolution`'s redaction is the backstop; a test asserts that formatting the token-resolution output (`%+v`) contains no token value (tasks T001). Residual **Yellow** — High×Low.
**Acceptance rationale**: a residual Yellow is accepted with justification — the credential never reaches a formatting site, the redaction backstop is preserved, and a regression test pins the no-leak property. Reducing severity below High is not possible (the value is a secret by nature); the control drives probability to Low, which is the acceptable corner of the High-severity row.

### H-6 — Connection-readiness drift
**Description**: Option A (ADR-1) keeps the per-domain enums precisely so `ConnectionContext.Complete()`/`Ready()`/status (which branch on `Cred.Source == auth.SourceNone`) need not change. The residual risk is the token resolver's own `KindNone→SourceNone` mapping: if wrong, a none-found could map to a non-None source (Complete() true with no token → an unauthenticated request attempt) or vice versa (a valid session judged not-ready).
**Severity — Medium**: a false "ready" leads to a request that 007 still refuses at send time (the refuse-to-call fail-safe is untouched), so the worst case is a clearer-elsewhere failure, not a sent-unauthenticated request.
**Probability — Low**: the mapping is a direct one-to-one (`KindNone→SourceNone`); the surface-stable choice deliberately leaves the consumer logic untouched.
**Controls**: **RC-6** — the provenance→`auth.Source` mapping test pins `KindNone→SourceNone` (and env/file rows); the `internal/apiclient` suite (Complete()/Ready()) is carried forward green (tasks T001). Residual **Green**.

---

## Residual Risk Summary

- **Red (unacceptable)**: none.
- **Yellow (acceptable with justification)**: H-5 (token secret hygiene through the new generic-resolution intermediate) — justified above; control RC-5 drives probability to Low and adds a no-leak regression test.
- **Green (accepted)**: H-1, H-2, H-3, H-4, H-6.

The single elevated hazard is credential hygiene, which is inherent to routing the token through a generic (non-redacting) resolver — the load-bearing control is to map-and-never-format, with the `auth.Resolution` redaction as backstop. This is the finding that justified running risk: it is not surfaced by checklist/analyze (which found the artifacts internally consistent), but it is the one real-world safety hazard the retrofit introduces.

---

## Traceability Index

| ID | Traces to |
|---|---|
| H-1 | plan.md § ADR-3, § Risks; spec.md § Behavioral Accord (validation) |
| H-2 | plan.md § Risks; interface-spec.md § Provenance → per-domain-type mapping |
| H-3 | plan.md § Risks; interface-spec.md § RunE plumbing, § Consistency Notes (compiler-enforced) |
| H-4 | plan.md § ADR-2, § Risks; interface-cli.md § Interactions |
| H-5 | plan.md § Cross-cutting (Secret hygiene); spec.md § Non-Behaviors (secret hygiene); interface-spec.md § Error Communication |
| H-6 | plan.md § ADR-1; interface-spec.md § Provenance → per-domain-type mapping |
| RC-1 | tasks.md T002/T003 acceptance criteria (regression tests) |
| RC-2 | tasks.md T002/T003 (label assertions); 039 pinned-label tests |
| RC-3 | plan.md § Risks (overload-free signatures); tasks.md T002/T003 (RunE presence test) |
| RC-4 | tasks.md T002/T003 (RunE presence tests) |
| RC-5 | plan.md § Cross-cutting (Secret hygiene); tasks.md T001 (no-leak test) |
| RC-6 | tasks.md T001 (mapping test); carried-forward internal/apiclient suite |

---

## Test Gap Analysis

First invocation — no test gap analysis. On re-run, RC-1…RC-6 can be cross-referenced against `.feature` scenarios; note that RC-5 (token no-leak) is currently a unit-level control with no behavioural `.feature` scenario, consistent with the token path having no operator-facing surface of its own.
