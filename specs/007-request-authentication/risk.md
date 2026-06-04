# Risk: Request Authentication

**Feature**: 007-request-authentication
**Round**: 1
**Date**: 2026-06-04
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light — no project-level matrix found in PROJECT.md
**Regulatory bridge**: none — PROJECT.md defines no Regulatory Context

---

## Risk Register

| H-ID | Hazard | Source | Sev | Prob | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | An unauthenticated request escapes to the API (base transport reached without `X-Auth-Token`) | spec Non-Behaviors; validation "No request is ever sent unauthenticated"; plan ADR-1 | High | Low | Yellow | RC-1 | Green |
| H-2 | Seam mismatch — the auth round-tripper is not actually composed in front of Connection Configuration's base transport | plan ADR-1/Risks; interface `[ASSUMED]` seam | High | Medium | Red | RC-2 | Yellow |
| H-3 | The token leaks into request logs, diagnostics, or an error string | spec Non-Behaviors; interface Error Communication | High | Low | Yellow | RC-3 | Green |
| H-4 | A broken credential is masked as "no credentials" (or vice versa), giving the wrong next step | spec error scenarios; interface Error Communication | Medium | Low | Green | RC-4 | Green |
| H-5 | A cannot-authenticate failure is reported as success (exits 0) — downstream misclassifies the typed error | plan ADR-4 (exit-code gap); CONSTITUTION III | High | Medium | Red | RC-5 | Yellow |
| H-6 | The token is altered in transit (trimmed/re-encoded), producing a wrong or invalid identity | spec edge "Token is attached verbatim" | Medium | Low | Green | RC-6 | Green |
| H-7 | Contract/build-order drift with Discovery (005) — `auth.Resolution` shape diverges | plan ADR-2/Risks; depends-on edge | High | Low | Yellow | RC-7 | Green |
| H-8 | Stale cached identity — the resolve-once cache serves an outdated credential | plan Configuration (resolve-once); interface identity lifetime | Low | Low | Green | RC-8 | Green |
| H-9 | An API token rejection (`401`/`403`) surfaces with no recoverable next step — response-side owner unspecified | spec Non-Behaviors (no response interpretation); interface Error Communication | Medium | Medium | Yellow | RC-9 | Yellow |

No residual risk remains **Red**. Three residual **Yellow** risks (H-2, H-5, H-9) are acceptable with the documented justifications below. (H-7 closed to Green after Discovery 005 landed on main — see its detail.)

---

## Hazard Detail

### H-1 — Unauthenticated request escapes
**Severity High** — an anonymous call could act as the wrong (or no) identity and return misleading governance data, or mutate as an unintended caller. **Probability Low** — the round-tripper only delegates to the base transport on the authenticated branch.
**Controls**: **RC-1** the `RoundTripper` reaches the base transport *only* after setting `X-Auth-Token`; absence or error returns an `AuthError` with the request unsent — structurally, not by caller discipline. Pinned by the validation scenario "No request is ever sent unauthenticated".
**Residual Green** — the guarantee is structural *given the round-tripper is in the request path* (that placement is H-2).

### H-2 — Seam mismatch with Connection Configuration
**Severity High** — if Connection Configuration sends through its base transport without 007's wrapper in front, every call goes out unauthenticated (or, if double-wrapped, the header is set twice); the structural guarantee in H-1 is voided. **Probability Medium** — the seam is `[ASSUMED]`, Connection Configuration is being modelled in a parallel session, and neither is built yet.
**Controls**: **RC-2** the auth layer is a `RoundTripper` composable over *any* base transport, so the wiring adapts cheaply; the composition seam and package name are reconciled with Connection Configuration before integration; whichever capability lands first creates the shared package.
**Residual Yellow** — initial Red reduced by the composable design, but cross-spec reconciliation is an open coordination action until both ship. Acceptable with that action tracked.

### H-3 — Token leaks into output
**Severity High** — a leaked credential in a request log, CI transcript, or error is a real compromise; the request side is where header logging leaks most easily. **Probability Low** — the token is confined to the header value and excluded from every other sink.
**Controls**: **RC-3** the token is set only via the request header API, never logged; `AuthError` (including `CredentialError`) carries only the path; request diagnostics redact or omit the `X-Auth-Token` value. Pinned by the validation scenario "The token value never appears in output" and "The token is redacted from request diagnostics".
**Residual Green** — comprehensive sink coverage plus two validation scenarios.

### H-4 — Broken vs absent credential conflated
**Severity Medium** — telling the operator "no credentials" when the file is actually broken (or vice versa) sends them down the wrong recovery path. **Probability Low** — the design models the two outcomes as distinct types.
**Controls**: **RC-4** a discriminable `AuthError{Kind}` — `NoCredentials` vs `CredentialError`; `CredentialError` wraps Discovery's typed error naming the path. Pinned by "A broken credential fails loudly without sending".
**Residual Green**.

### H-5 — Auth failure reported as success
**Severity High** — a cannot-authenticate outcome that exits `0` is a Fail-Safe violation (CONSTITUTION III): the operator/agent believes the command succeeded when no authenticated call was made. **Probability Medium** — 004's frozen convention has no dedicated code for a *local* "cannot authenticate" precondition (code 4 is reserved for API-side rejection), and the mapping rests with a not-yet-specified consuming command.
**Controls**: **RC-5** 007 returns a typed, discriminable `AuthError` (never a nil/success outcome), forcing the consuming command to classify it via `errors.As` and map it to a non-zero code; the missing exit-code category is flagged as a behavioral gap for the consuming-command spec / `/score:clarify`.
**Residual Yellow** — 007's part (never returning success on failure) is solid; the actual non-zero mapping depends on a downstream spec that does not yet exist. Acceptable with the exit-code-category decision tracked for that spec. *Related: the next-step wording for the operator message is the same cascade noted in checklist advisory II.*

### H-6 — Token altered in transit
**Severity Medium** — a trimmed or re-encoded token authenticates as a different (or invalid) identity, causing silent rejection or wrong-identity action. **Probability Low** — the design attaches the value verbatim.
**Controls**: **RC-6** the header value equals the resolved token exactly — no trim, no re-encode. Pinned by "The token is attached verbatim".
**Residual Green**.

### H-7 — Contract/build-order drift with Discovery (005)
**Severity High** — if the consumed `auth.Resolution` shape diverges from what 005 produces, resolution breaks and authentication silently fails. **Probability Low** — 005 is now implemented and validated **Ready** (PR #11), and its shipped contract matches 007's assumptions exactly: `Resolution{Token, Source, Path}`, the `Source` enum (`SourceNone`/`SourceEnvironment`/`SourceFile`), `func Resolve() (Resolution, error)` (which fits ADR-3's injected-resolver signature), and the path-only typed `FormatError`/`ReadError` for `CredentialError` to wrap. This is no longer two unbuilt specs converging on a shape.
**Controls**: **RC-7** 007 imports the now-real `auth.Resolution` and binds `auth.Resolve` in production; the dependency direction is one-way (client → `internal/auth`); 007's mapping and round-tripper are still unit-tested with a fake resolver, and any future shape change updates one site.
**Residual Green** — the dependency is built, validated, and contract-confirmed; only the (already-clean) composition with Connection Configuration remains open, tracked under H-2.

### H-8 — Stale cached identity
**Severity Low** — within a single short-lived CLI invocation the identity is meant to be fixed (single org + person per key), so a stale cache is the intended semantics rather than a fault. **Probability Low** — a credential change mid-invocation is rare and out of the single-identity model.
**Controls**: **RC-8** the resolve-once cache is scoped to one CLI invocation; 005 resolution is deterministic, so the cached identity equals a re-resolved one for the invocation's lifetime.
**Residual Green**.

### H-9 — API token rejection has no recoverable next step
**Severity Medium** — a `401`/`403` from the API (expired/revoked/wrong-scope token) needs a clear "re-authenticate / check your token" next step; 007 correctly does not interpret responses, but the owner of response-side auth handling (Connection Configuration / a future command spec) is not yet specified, so a rejection could surface as a generic API error. **Probability Medium** — token expiry/revocation is a routine real-world condition and the handling owner is currently unassigned.
**Controls**: **RC-9** 007 scopes response interpretation out cleanly (request-side only — spec Non-Behavior), keeping the boundary unambiguous; the response-side `401`/`403` recovery message is flagged as an ownership item for Connection Configuration / the consuming-command spec.
**Residual Yellow** — not a 007 defect (correct scoping), but a real-world gap in the end-to-end auth story until the response-side owner is specified. Acceptable with the ownership item tracked.

---

## Residual Risk Summary

| Residual | Count | Hazards |
|---|---|---|
| Green | 6 | H-1, H-3, H-4, H-6, H-7, H-8 |
| Yellow | 3 | H-2, H-5, H-9 |
| Red | 0 | — |

The three Yellow residuals share a theme: each depends on a condition outside 007's own logic — the composition seam with a parallel sibling (H-2), the exit-code mapping in a not-yet-specified command (H-5), and the response-side rejection owner (H-9). 007's own controls are in place for each; none is unacceptable. (H-7, the 005 dependency, closed to Green now that 005 is implemented and validated Ready with a matching contract.) The highest-leverage follow-ups are reconciling the Connection Configuration seam (H-2) and deciding the cannot-authenticate exit-code category (H-5) when the consuming-command spec is written.

---

## Traceability Index

**Hazards → source**:
- H-1 → spec.md § Non-Behaviors (no unauthenticated fallback); § Validation Scenarios; plan.md ADR-1
- H-2 → plan.md ADR-1 + § Risks; interface-spec.md § Consistency Notes (`[ASSUMED]` seam)
- H-3 → spec.md § Non-Behaviors (secret hygiene); interface-spec.md § Error Communication
- H-4 → spec.md § Behavioral Accord > Fail safe; interface-spec.md § Error Communication
- H-5 → plan.md ADR-4 (exit-code gap) + § Risks; CONSTITUTION.md III (Fail Safe)
- H-6 → spec.md § Driving Scenarios (Edge — verbatim)
- H-7 → plan.md ADR-2 + § Risks; FEATURE-MODEL depends-on edge (005)
- H-8 → plan.md § Cross-cutting Concerns (resolve-once); interface-spec.md § Surface (identity lifetime)
- H-9 → spec.md § Non-Behaviors (no `401`/`403` interpretation); interface-spec.md § Error Communication

**Controls → architectural grounding**:
- RC-1 → plan ADR-1 (RoundTripper delegates only on the authenticated branch)
- RC-2 → plan ADR-1/ADR-2 (composable RoundTripper; seam + package reconciled before integration)
- RC-3 → plan § Cross-cutting Concerns (secret hygiene); interface Error Communication
- RC-4 → plan ADR-4 (typed `AuthError{Kind}`); interface Error Communication
- RC-5 → plan ADR-4 (typed code-free error; consuming command maps via `errors.As`)
- RC-6 → spec Edge / interface Interactions (verbatim attach)
- RC-7 → plan ADR-3 (injected resolver) + ADR-2 (consume 005's pinned `Resolution`)
- RC-8 → plan § Cross-cutting Concerns (resolve-once cache, deterministic resolution)
- RC-9 → spec Non-Behavior / interface Error Communication (response interpretation scoped out)

Downstream: the `.feature` validation scenarios already realize RC-1 (no-unauthenticated-send), RC-3 (token-never-in-output / diagnostic redaction), and RC-6 (verbatim). RC-2, RC-5, and RC-9 carry cross-spec coordination actions (seam, exit-code category, response-side owner) to track before the end-to-end auth path ships; RC-7's 005 dependency is now satisfied (005 implemented & validated Ready, contract confirmed matching).
