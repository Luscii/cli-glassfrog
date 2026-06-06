# Risk: Connection Context Assembly

**Feature**: 009-connection-context-assembly
**Round**: 1
**Generated**: 2026-06-06
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Matrix**: default 3×3 traffic-light — no project-level acceptability matrix found in PROJECT.md
**Degradation flags**: none — spec, plan, and interface all present. PROJECT.md has no Regulatory Context, so no IEC 14971 bridge is included.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Pre-control | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | The token leaks when the `ConnectionContext` is rendered/logged (it holds the secret in `Cred`) | spec § Non-Behaviors (secret); plan § Cross-cutting; interface § Error Communication | High | Low | Yellow | RC-1 | **Green** |
| H-2 | Assembly short-circuits on the first resolver error, dropping the other sub-outcome (carry-both broken → partial diagnosis) | spec § Behavioral Accord (carry-forward); plan ADR-1; spec error scenario "both inputs report a problem" | Medium | Medium | Yellow | RC-2 | **Green** |
| H-3 | Assembly usurps 007's fail-safe — refuses/decides on absent/errored credential, splitting the contract | spec § Non-Behaviors ("must not refuse / decide exit code"); plan ADR-2 | Medium | Low | Green | RC-3 | **Green** |
| H-4 | Request Execution (010) re-resolves instead of replaying the context → identity/endpoint drift from what readiness reported | plan ADR-2; interface § Interactions (single resolution point); spec § lifecycle | Medium | Medium | Yellow | RC-4 | **Yellow** |
| H-5 | A readiness reason (`Problems()`) or a carried error leaks the token or a secret-bearing string | interface § Error Communication ("no secret anywhere"); spec § Non-Behaviors | High | Low | Yellow | RC-5 | **Green** |
| H-6 | A nil resolver silently degrades (fabricated/empty outcome) instead of failing loud | plan § Cross-cutting (fail-fast, no nil-default — PR #20); interface § Entry points | Medium | Low | Green | RC-6 | **Green** |

No residual **Red**. One residual **Yellow** (H-4) — acceptable with the documented justification below.

> **Inherited, not re-owned**: the mis-derived-default-host hazard (008 H-1) is *not* a new 009 hazard — this slice carries 008's `BaseURL` verbatim and owns no endpoint derivation. It remains tracked under 008's risk.md until the connection-context half sends real traffic (010).

---

## Hazard Detail

### H-1 — Token leak when the context is rendered
**Severity: High** (secret exposure) — `ConnectionContext` carries `Cred auth.Resolution`, which holds the token; a `%v`/`%+v`/log of the struct could dump it. **Probability: Low** — the design mandates a redacting render, and the spec/interface forbid emitting the token.
- **RC-1**: a **value-receiver redacting `String()`** renders the credential `Source`/`Path` and readiness but reports the token as present/absent — never verbatim — covering `%v`/`%+v`/`%s` (the `auth.Resolution.String()` precedent); a test asserts the token never appears in `%+v`/`String()` (tasks T001 acceptance + risk note).
- **Residual: Green**.

### H-2 — Carry-both broken by a short-circuit
**Severity: Medium** — if `Assemble` returned after the first resolver error, the context would carry only one problem; the operator fixes it, re-runs, and immediately hits the second (extra round-trips, misleading "one problem" picture) — but no governance is corrupted and nothing is sent. **Probability: Medium** — a naive `if err != nil { return }` after the base-URL resolver is an easy implementation slip; the plan and tasks call this out explicitly as the carry-both risk.
- **RC-2**: `Assemble` calls **both** resolvers unconditionally before constructing the context; a **both-resolvers-called tripwire** fake fails the test if the second resolver isn't invoked after the first errors (spec carry-forward accord; plan ADR-1; tasks T002).
- **Residual: Green**.

### H-3 — Assembly usurps 007's fail-safe
**Severity: Medium** — if assembly decided to refuse the request or mapped an exit code, the fail-safe contract would be split across two capabilities; a wrong decision could let an unauthenticated/misrouted call proceed. Blast radius is bounded because 007's `AuthTransport` still refuses at `RoundTrip` on an absent/errored credential (structural). **Probability: Low** — ADR-2 keeps refusal in 007; `Complete()`/`Problems()` are reporting-only and `Assemble` returns no error.
- **RC-3**: assembly produces a code-free context and decides nothing — readiness is a reported view; the refuse-to-call fail-safe stays in 007's `AuthTransport`, which (per ADR-2) replays the context's credential and refuses at request time (spec § Non-Behaviors; plan ADR-2).
- **Residual: Green**.

### H-4 — Replay seam re-resolves → identity/endpoint drift
**Severity: Medium** — if Request Execution (010) re-walks Credential Discovery instead of replaying the context's cached credential, the credential used at request time could differ from the one whose readiness the context reported (e.g., a `.glassfrogrc` edited between assembly and the call) — the operator sees "complete" but the request acts under a different/absent identity. Confusing, not governance-corrupting. **Probability: Medium** — 010 is a future spec; honoring the seam is a coordination obligation, not yet code.
- **RC-4**: `ConnectionContext` carries the full resolved credential so 010 can replay it; ADR-2 (and the `.score/memory/DEPRECATION.md` entry retiring the open seam) pin "010 wires `AuthTransport` with a thunk over `ctx.Cred`/`ctx.CredErr`, not a fresh walk"; `AssembleFromOS` documents once-per-invocation, and `AuthTransport`'s `sync.Once` backstops the request layer.
- **Residual: Yellow** — accepted with justification: the enforcing control lives in 010 (future), not in this slice, so it cannot be unit-pinned here. Tracked via the DECISIONS precedent and the DEPRECATION entry; verify when 010 wires the transport. (Mirrors 008 H-6's future-coordination Yellow.)

### H-5 — Secret leak through a readiness reason or carried error
**Severity: High** (secret exposure) — `Problems()` and the carried errors are operator-facing; if a reason were built from `Cred.Token` or a credential error embedded the token, the secret would leak through the readiness surface (distinct from H-1's struct-render surface). **Probability: Low** — base-URL errors are source-only (008), credential errors are path-only by the 005/007 contract, and `Problems()` is specified to derive only from safe labels.
- **RC-5**: `Problems()` is built **only** from `BaseURLError.Source`/path and the credential `Source`/`Path` (or a fixed "no credentials found" phrase), never `Cred.Token`; carried errors are already source/path-only upstream; a test asserts no token appears in any `Problems()` entry (interface § Error Communication; tasks T001).
- **Residual: Green**.

### H-6 — Nil resolver silently degrades
**Severity: Medium** — a nil-defaulted resolver could fabricate an empty/placeholder outcome and hide a wiring bug, producing a context that misrepresents what was resolved. **Probability: Low** — the resolvers are wired deliberately at integration; the design chooses fail-fast.
- **RC-6**: `Assemble` requires non-nil resolvers and **panics** on nil (fail-fast, no nil-default — the PR #20 LEARNINGS stance); the precondition is documented on the constructor and a nil-panic test pins it (tasks T002).
- **Residual: Green**.

---

## Residual Risk Summary

6 hazards, 6 controls. After controls: 5 Green, 1 Yellow, 0 Red.
- **H-4 (Yellow)**: the replay-seam control lives in Request Execution (010); verify 010 replays the context's credential rather than re-resolving. Tracked via the DECISIONS precedent and the DEPRECATION entry that retired the open 007/009 seam.

The single Yellow does not block this slice — it is a forward-coordination obligation on a future spec, already recorded as precedent. Two High-severity secret-exposure hazards (H-1, H-5) both reduce to Green via the redacting-render and secret-free-reason controls, consistent with the secret-hygiene discipline carried from 005/007.

---

## Traceability Index

**Hazards → source**: H-1 → spec § Non-Behaviors, plan § Cross-cutting · H-2 → spec § Behavioral Accord (carry-forward), plan ADR-1 · H-3 → spec § Non-Behaviors, plan ADR-2 · H-4 → plan ADR-2, interface § Interactions · H-5 → interface § Error Communication, spec § Non-Behaviors · H-6 → plan § Cross-cutting (fail-fast).

**Controls → grounding**: RC-1 → plan Cross-cutting (redacting `String()`, `auth.Resolution` precedent) · RC-2 → ADR-1 (carry-both, tripwire test) · RC-3 → ADR-2 (refusal stays in 007) · RC-4 → ADR-2 + DEPRECATION entry (replay seam) · RC-5 → interface Error Communication (secret-free reasons) · RC-6 → Cross-cutting (fail-fast on nil).

Downstream: tasks reference these controls — RC-1/RC-5 → T001 (redacting `String()` + token-never-in-`Problems()` tests); RC-2/RC-6 → T002 (both-resolvers-called tripwire + nil-panic tests); RC-3 is realized structurally (007 untouched) and RC-4 is a 010 obligation.
