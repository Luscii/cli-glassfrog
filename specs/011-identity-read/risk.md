# Risk: Identity Read

**Feature**: 011-identity-read
**Round**: 1
**Generated**: 2026-06-07
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: default 3×3 traffic-light — no project-level matrix found in PROJECT.md
**Degradation flags**: none (full upstream artifact set present). Note: 010 (Request Execution) is the upstream code dependency, now implemented and landed on main (#30); building 011 from current main satisfies it. This is a build-sequencing concern captured in plan.md/tasks.md, not a domain hazard.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk | Controls | Residual |
|---|---|---|---|---|---|---|---|
| H-1 | The token leaks into output, logs, or an error message | spec Non-Behaviors; plan Cross-cutting/Risks; interface auth boundary | High | Low | Yellow | RC-1, RC-2, RC-3 | Green |
| H-2 | A generic non-2xx hides which failure it is (auth rejection vs rate-limit vs server error) | spec Non-Behaviors; interface error table; plan ADR-4 | Medium | Medium | Yellow | RC-4, RC-5 | Yellow |
| H-3 | A `429` without backoff contributes to org-wide throttling | spec Non-Behaviors; plan (defers to 017); CONSTITUTION X | Medium | Low | Green | RC-6, RC-7 | Green |
| H-4 | A wrong/inferred base-URL host sends the token to the wrong (or hostile) endpoint | 008 risk H-1; interface `--base-url`; spec Integration Boundaries | High | Low | Yellow | RC-8, RC-9, RC-1 | Yellow |
| H-5 | A malformed `.glassfrogrc` (CredentialError) is indistinguishable from other internal errors and lacks a next step | interface error table; analyze K5; checklist P1 | Low | Medium | Green | RC-10, RC-11 | Green |
| H-6 | Tolerant decode renders a missing projected field as blank, misleading the agent | plan ADR-1; CONSTITUTION VIII | Low | Low | Green | RC-12, RC-13 | Green |
| H-7 | `--include` validation drifts from the spec enum (rejects a valid target, or lets an unknown one reach the API) | plan ADR-5/Assumptions; interface `--include` | Low | Low | Green | RC-14 | Green |

**No Red residuals.** Two Yellow residuals (H-2, H-4) are acceptable with the documented justifications below; the rest reduce to Green.

---

## Hazard Detail

### H-1 — Token leakage in output, logs, or errors
- **Source**: spec.md Non-Behaviors ("must not print, log, or expose the token"); plan.md Cross-cutting (secret hygiene) and Risks; interface-spec.md Error Communication ("No secret anywhere").
- **Severity — High**: the token is an org+person credential; exposure enables impersonation against live governance data.
- **Probability — Low**: structurally prevented — `me` never reads `ctx.Cred.Token`; the token's only path is 010's replay thunk into 007's `AuthTransport`; the projection renders response-side fields only.
- **Controls**: **RC-1** `me` never reads `ctx.Cred.Token` (no code path touches it). **RC-2** the projection renders only response-side fields (the `X-Auth-Token` request header is never echoed). **RC-3** a token-never-in-output test asserts the token is absent across success and every error branch.
- **Residual — Green**: the structural guarantee plus the regression test reduce High×Low to acceptable.

### H-2 — Generic non-2xx obscures the failure kind
- **Source**: spec.md Non-Behaviors (no per-status interpretation); interface-cli.md exit-code table (every non-2xx → exit 3); plan.md ADR-4.
- **Severity — Medium**: an agent that can't tell a `401` (re-authenticate) from a `429` (back off) from a `5xx` (transient) may retry uselessly, fail to re-auth, or escalate wrongly.
- **Probability — Medium**: `401`/`403`/`429` are routine API responses.
- **Controls**: **RC-4** the failure message carries the HTTP status code, so the operator/agent can read "401" / "429" even before classification exists. **RC-5** API Error Extraction (015) and Rate-Limit Handling (017) will split `APIError`(3) into permission(4) and rate-limited(5) without renumbering — a roadmap commitment.
- **Residual — Yellow (acceptable, justified)**: the status code is surfaced now; finer machine-classification is a deliberate deferral to 015/017 (spec fork 3). Surfacing the raw status is sufficient for a human or a status-code-aware agent in the interim.

### H-3 — `429` without backoff contributes to throttling
- **Source**: spec.md Non-Behaviors (no retry/backoff); plan.md (defers to 017); CONSTITUTION X.
- **Severity — Medium**: sustained ignored `429`s can throttle the whole org.
- **Probability — Low**: `me` makes exactly one request per invocation and never retries, so it cannot itself drive a retry storm; a single `/me` rarely triggers `429`.
- **Controls**: **RC-6** exactly one attempt, no retry loop (010's bounded `Do`). **RC-7** a `429` is surfaced as a failure, so the caller stops rather than looping.
- **Residual — Green**: a one-shot read cannot cause throttling on its own; honoring `Retry-After`/backoff is 017's scope.

### H-4 — Wrong/inferred base-URL host
- **Source**: 008 risk H-1 (the default host `https://glassfrog.com` is *inferred* from `info.contact.url`, not normatively specified); interface-cli.md `--base-url`; spec.md Integration Boundaries.
- **Severity — High**: if the resolved endpoint is wrong or attacker-controlled, the request — carrying `X-Auth-Token` — goes to the wrong host (credential exfiltration or wrong governance data).
- **Probability — Low**: the default is a fixed compiled constant; overriding it requires an explicit flag/env/`.glassfrogrc` value the operator controls, and 008 rejects a non-`http(s)` value.
- **Controls**: **RC-8** the base URL must be an absolute `http(s)` URL or it fails loud (008 `BaseURLError`). **RC-9** the token is sent only to the resolved endpoint, which defaults to the known constant. **RC-1** (the token is handled only via the replay thunk).
- **Residual — Yellow (acceptable, justified)**: the inferred-host concern is inherited from 008 (its risk H-1) and is the operator's configuration responsibility; the default constant and the `http(s)` validation bound it. Re-examine if the spec ever publishes a normative host.

### H-5 — Malformed `.glassfrogrc` indistinguishable / no next step
- **Source**: interface-cli.md error table (`CredentialError` → exit 1); analyze.md K5 (no scenario); checklist.md P1 (no next step).
- **Severity — Low**: operator confusion only — no governance data is harmed.
- **Probability — Medium**: a hand-edited credentials file can be malformed.
- **Controls**: **RC-10** the error names the offending file path (007/`rcfile` `FormatError`, secret-safe). **RC-11** *(implemented this session)* interface-cli.md now specifies the next-step message ("fix or re-create the file with `glassfrog auth login`") and `identity-read.feature` now has the acceptance scenario "A malformed credentials file fails the read loudly" for the `CredentialError` path.
- **Residual — Green**: low impact; the checklist P1 and analyze K5 that fed this hazard are now resolved.

### H-6 — Blank projected field from tolerant decode
- **Source**: plan.md ADR-1 (tolerant decode; "only projected fields required to be present"); CONSTITUTION VIII (No Fabricated Data).
- **Severity — Low**: a blank is not a fabricated value, but a blank id could mislead a follow-up call.
- **Probability — Low**: `MeResponse`'s projected fields (actor id/name/kind, org id/name, access level) are spec-required.
- **Controls**: **RC-12** decode tolerates unknown fields but never synthesizes a value (VIII holds). **RC-13** *(consider)* treat a missing required-by-projection field as a representation error rather than rendering a silent blank.
- **Residual — Green**.

### H-7 — `--include` validation drift
- **Source**: plan.md ADR-5/Assumptions; interface-cli.md `--include`.
- **Severity — Low**: a valid target wrongly rejected, or an unknown target reaching the API (which would 4xx).
- **Probability — Low**: the enum is `{roles}` only today; tracking the spec is a one-line change.
- **Controls**: **RC-14** `validateInclude` is sourced from the spec's `include` set, so it tracks the contract.
- **Residual — Green**.

---

## Residual Risk Summary

- **Unacceptable (Red)**: none.
- **Acceptable with justification (Yellow)**: H-2 (generic non-2xx — status surfaced, finer classification deferred to 015/017) and H-4 (inferred base-URL host — inherited from 008, bounded by the default constant + `http(s)` validation).
- **Accepted (Green)**: H-1, H-3, H-5, H-6, H-7.

The dominant safety properties for a read that carries a secret across an auth boundary — no token leakage (H-1) and no wrong-host exfiltration (H-4) — are both bounded by structural controls. No hazard blocks implementation.

---

## Traceability Index

**Hazards → source**
- H-1 → spec Non-Behaviors (token); plan Cross-cutting/Risks; interface-spec Error Communication
- H-2 → spec Non-Behaviors (no per-status interpretation); interface-cli exit-code table; plan ADR-4
- H-3 → spec Non-Behaviors (no retry); plan "What This Plan Does Not Cover"; CONSTITUTION X
- H-4 → DECISIONS/008 risk H-1 (inferred host); interface-cli `--base-url`; spec Integration Boundaries
- H-5 → interface-cli/-spec error tables; analyze K5; checklist P1
- H-6 → plan ADR-1 (tolerant decode); CONSTITUTION VIII
- H-7 → plan ADR-5/Assumptions; interface-cli `--include`

**Controls → grounding** (downstream tasks/scenarios may reference these RC-N IDs)
- RC-1/RC-2/RC-3 → plan Cross-cutting (secret hygiene); spec Validation "token never in output" scenario
- RC-4/RC-5 → plan ADR-4; interface error tables; 015/017 roadmap
- RC-6/RC-7 → 010 one-bounded-attempt; spec Non-Behaviors
- RC-8/RC-9 → 008 `BaseURLError` validation; plan Integration Design
- RC-10/RC-11 → 007/`rcfile` `FormatError`; checklist P1 + analyze K5 (recommended additions)
- RC-12/RC-13 → plan ADR-1 / Data Model Design
- RC-14 → plan ADR-5 / Assumptions (include set tracks the spec enum)

---

## Regulatory Bridge

Not applicable — PROJECT.md declares no Regulatory Context (no IEC 14971 mapping required). Plain-language risk assessment only.
