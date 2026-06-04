# Checklist: Request Authentication

**Feature**: 007-request-authentication
**Checked against**: CONSTITUTION.md (no `accords/governance/done-*.md` present)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/unauthenticated-access/request-authentication.feature, tasks.md
**Checks**: 9 (9 pass, 0 fail)
**Generated**: 2026-06-04

---

## Summary

All 9 checks pass. Constitution: 9/9. Done-criteria: not run (no accords). Cross-references: not run (no accords).

9 of 12 constitution principles produced applicable checks; 4 are N/A for this request-side, no-command auth middleware that interprets no API response (see Governance Notes). Unlike read-only Discovery (005), 007 sits directly on the API request contract — so **Spec Fidelity (I)** (it attaches the spec-defined `X-Auth-Token`) and **Writes Require Explicit Intent (IX)** (it is method-agnostic and adds only a header) are applicable here.

---

## Constitution Checks: 9/9 passed

### Failures

None.

### Passed (9/9)

**P0** | CONSTITUTION.md I (Spec Fidelity): "MUST map to a Glassfrog API v5 spec operation … MUST NOT invent endpoints, parameters, or behaviors"
→ **interface-spec.md § Surface (Configuration), § Consistency Notes**, **spec.md § Assumptions**: authentication uses exactly the spec-defined `X-Auth-Token` header scheme (PROJECT constraint); 007 invents no auth parameter, endpoint, or behavior — it attaches the contract's own credential header to outgoing calls.

**P0** | CONSTITUTION.md III (Fail Safe, Not Silent): "Errors MUST be obvious and recoverable, never hidden … validate a write before sending"; anti-pattern "a failure condition reported as success"
→ **spec.md § Behavioral Accord > Fail safe**, **interface-spec.md § Error Communication ("No unauthenticated send")**, **features/…/request-authentication.feature**: an absent or broken credential never produces an anonymous request — it ends in a typed `AuthError` with the request unsent, and a broken credential (`CredentialError`) is kept distinct from absence (`NoCredentials`). Pinned by "A missing credential refuses the call", "A broken credential fails loudly without sending", and validation "No request is ever sent unauthenticated".

**P0** | CONSTITUTION.md IV (TDD): "Features MUST be built test-first (RED → GREEN)"
→ **tasks.md § T001, T002**: both mandate RED-first unit tests (the `authorize` mapping's three branches + secret hygiene; the round-tripper over a fake base transport + fake resolver) before implementation.

**P0** | CONSTITUTION.md IV: "user-facing behavior MUST have an executable acceptance scenario before the code"
→ **features/unauthenticated-access/request-authentication.feature**, **tasks.md § T003**: the behavioral `@wip` scenarios (token attached / verbatim / same-identity / resolved-once / missing-credential refusal / broken-credential refusal / active-source reported / diagnostic redaction) exist before implementation; tasks.md T003 turns them into executable acceptance.

**P0** | CONSTITUTION.md V (Composition over Monolith): "modular, independently-testable parts … adding … MUST NOT require changing unrelated ones"
→ **plan.md § ADR-1, ADR-2, ADR-3**: 007 is a `RoundTripper` in the API-client package (not `internal/auth`), composing over Connection Configuration's base transport and consuming Discovery via an injected resolver. It registers no command and edits no existing module; the round-tripper is exercised in isolation with a fake base transport and fake resolver.

**P0** | CONSTITUTION.md VII (Working Software): implementation paired with tests; "validate and build"
→ **tasks.md § T001, T002, T003**: each task pairs implementation with tests; `go build ./...` and `go vet ./...` clean are explicit acceptance criteria.

**P0** | CONSTITUTION.md VIII (No Fabricated Data): "MUST NOT invent, guess, or fill placeholder values" (applied to the credential)
→ **spec.md § Non-Behaviors** ("must not fabricate a token … must not send an unauthenticated request as a fallback"), **scenario "A missing credential refuses the call"**: when no credential is available, 007 refuses the call rather than fabricating or substituting a token.

**P0** | CONSTITUTION.md IX (Writes Require Explicit Intent): "A read-shaped command … MUST NEVER mutate as a side effect"
→ **plan.md § System Architecture**, **interface-spec.md § Interactions**: 007 attaches a header only — it never sets or changes the request method or body, so it cannot turn a read into a write. It initiates no request itself; the command layer decides the operation. No mutation is introduced.

**P0** | CONSTITUTION.md XII (Standalone Executable): "no language runtime … no other software … that must be installed first"
→ **plan.md § ADR-1, Risks**: the round-tripper uses only the Go standard library (`net/http`, `errors`); no third-party dependency is added. The artifact remains a single self-contained binary.

---

## Governance Notes

- **No `done-*` accords found.** Constitution checks ran; done-criteria and cross-reference checks did not. Consider creating, to enable vertical quality checks for later specs:
  - `accords/governance/done-specify.md`, `done-plan.md`, `done-interface.md`, `done-scenarios.md`, `done-tasks.md`
- **Principle II (Action Transparency)**: N/A for the direct operator-facing clause — 007 surfaces nothing to the operator itself and decides no message (spec non-behavior). It hands up machine-parseable material — the active identity (`Source`/`Path`, never the token) and a discriminable typed `AuthError{Kind}` — that the consuming command renders. See Advisory Notes for the next-step cascade.
- **Principle VI (Size-Aware)**: N/A — 007 attaches a header; it handles no API result sets or pagination (Connection Configuration / response handling own those).
- **Principle X (Respect API Limits)**: N/A — 007 interprets no response, so `429` back-off and `If-Match`/`ETag` concurrency are out of its scope (they live with Connection Configuration and the update-issuing commands). See Advisory Notes.
- **Principle XI (Governance via Proposals)**: N/A — 007 exposes no governance-mutating command path.

## Advisory Notes (not severity findings)

These passed (or are out of scope) but are worth the developer's attention:

- **II — the "next step" cascade**: Action Transparency requires every operator-facing error to explain the next step. 005 deferred the "acting as" line and the error's next step to 007; 007 in turn defers the operator *message* to the consuming command (spec non-behavior), providing the discriminable `Kind` (`NoCredentials` vs `CredentialError`) so the command can produce a specific next step ("no token — run the login command / set `GLASSFROG_TOKEN`" vs "credentials file broken at `<path>`"). 007 satisfies its share, but the actual next-step wording now rests with a not-yet-specified consuming command — pin it when that spec is written so the cascade doesn't drop the next step. (This is the same clause a reviewer raised on 006.)
- **Secret hygiene has no dedicated principle**: "the token value never appears in output, logs, or diagnostics" is a genuine security property (especially on the request side, where header logging leaks easily) but doesn't trace cleanly to a single constitution principle, so it is not raised as a severity check. It is pinned by the validation scenarios "The token value never appears in output" and "The token is redacted from request diagnostics" — recommend the reviewer confirm `AuthError` strings and any request tracing carry only the path, never the token.
- **`[ASSUMED]` seam with Connection Configuration (parallel session)**: the composition seam (`RoundTripper` wrapping), the package name (`internal/apiclient` proposed), and the `net/http` substrate are provisional and shared with an unspecified sibling. This is a cross-artifact/coordination concern (analyze's domain), not a constitution check — flagged so it is reconciled before either capability ships. (The upstream dependency, Discovery 005, is now implemented and validated Ready on main, PR #11.)
