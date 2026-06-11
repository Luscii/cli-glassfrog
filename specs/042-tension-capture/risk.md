# Risk: Tension Capture

**Feature**: 042-tension-capture
**Round**: 1
**Date**: 2026-06-11
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. (PROJECT.md has no Regulatory Context, so no IEC 14971 bridge is included.)

This is the CLI's **first write** to live governance data, so the hazards center on creating the *right* record under the *right* identity, exactly once, without leaking the credential. The governance-mutation hazards that dominate the write path generally (CONSTITUTION XI) do **not** apply here: a tension is an *operational* resource (the proposal seed), not governance structure, and it is fully recoverable (a future `tension update`/`delete` exists in the API).

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | Tension recorded against the wrong sensing role (id passed through unvalidated) | spec.md § Behavioral Accord / Non-Behaviors (ADR-3 id pass-through) | Medium | Low | Green | RC-1 | Green |
| H-2 | Request body silently ignored → empty/failed capture (missing `Content-Type`) | plan.md § ADR-1 / Risks | Medium | Low | Green | RC-2 | Green |
| H-3 | Duplicate tension from command-level retry after a post-success network failure (at-least-once) | plan.md § Cross-cutting + interface (non-idempotent POST) | Low | Medium | Green | RC-3 | Green (accepted) |
| H-4 | Tension attributed to the wrong sensing person | spec.md § Non-Behaviors ("must not set `sensed_by`") | Low | Low | Green | RC-4 | Green |
| H-5 | Empty/meaningless tension recorded | spec.md § Behavioral Accord (`--body` required) | Low | Low | Green | RC-5 | Green |
| H-6 | API token leaked in output, error, or request body | interface § auth boundary / CONSTITUTION II | High | Low | Yellow | RC-6 | Yellow (justified) |
| H-7 | A `422`/non-2xx mis-surfaced as success or as an opaque failure | interface-cli.md § Error Communication | Medium | Low | Green | RC-7 | Green |

---

## Hazard Details

### H-1: Wrong sensing role
**Source**: spec.md § Behavioral Accord / Non-Behaviors — the `<role-id>` is escaped but **passed through unvalidated** (plan ADR-3), so a valid-but-unintended `role_…` is accepted by the API.
**Severity**: Medium — a tension misattributed to the wrong sensing role pollutes that role's tension list and could later seed a proposal under the wrong context. Operational and recoverable, not a governance-structure change.
**Probability**: Low — the usual operator is an AI agent that resolves the role id from a prior read; a typo'd-but-valid id is uncommon, and an unknown id 404s cleanly.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-1**: The created tension is rendered with its `role_id` (sensing role) so the operator can verify attribution immediately; an unknown/malformed id surfaces the API's `404`/`422` rather than a silent wrong write.
**Residual Risk**: Green — verification-on-output + clean not-found make a silent misattribution unlikely.

### H-2: Body silently ignored
**Source**: plan.md § ADR-1 / Risks — without `Content-Type: application/json`, the API may ignore the JSON body.
**Severity**: Medium — a tension created with no/empty body, or a confusing `422`.
**Probability**: Low — mitigated by design.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-2**: The request always carries `Content-Type: application/json` (the new `apiclient.Request.ContentType`), pinned by a Phase-1 header-present/absent test (tasks T001).
**Residual Risk**: Green.

### H-3: Duplicate tension (at-least-once)
**Source**: plan.md § Cross-cutting + interface — `createTension` is a non-idempotent `POST` and the spec defines no idempotency key.
**Severity**: Low — a duplicate tension is a recoverable operational item (deletable via the future `tension delete`); it does not corrupt governance structure.
**Probability**: Medium — an agent operator may re-run the command after an ambiguous failure (e.g. a network timeout that occurs *after* the server created the tension but *before* the response reaches the CLI), and there is no server-side dedupe.
**Risk Level**: Green (Low × Medium).
**Controls**:
- **RC-3**: HTTP-level retry is suppressed for the non-idempotent `POST` (§133 `isSafeMethod`), so a `429`/transient does not auto-duplicate; the command returns a clear success/failure exit code so an agent can distinguish a completed capture from a failed one.
**Residual Risk**: Green (accepted) — the post-success-network-failure window is inherent to a non-idempotent create with no idempotency key in the API; it is recoverable, and the future tension reads/delete provide the detection/cleanup path. Documented so the Builder does not add a client-side auto-retry on the create that would convert this into systematic duplication.

### H-4: Wrong sensing person
**Source**: spec.md § Non-Behaviors — "must not infer, set, or override `sensed_by`."
**Severity**: Low — the API derives the person from the token, so a wrong attribution would require a wrong token, not a CLI defect.
**Probability**: Low.
**Risk Level**: Green.
**Controls**:
- **RC-4**: The CLI never sends a `sensed_by`/person field; the request body carries only `tension.{body,label?,meeting_type?}`. The sensing person is the token's identity by construction.
**Residual Risk**: Green.

### H-5: Empty/meaningless tension
**Source**: spec.md § Behavioral Accord — a bodyless tension is meaningless.
**Severity**: Low.
**Probability**: Low.
**Risk Level**: Green.
**Controls**:
- **RC-5**: `--body` is validated non-empty (after `strings.TrimSpace`) **before any request** (fail-fast `UsageError(2)`); the API's `422` is a backstop.
**Residual Risk**: Green.

### H-6: Token leakage
**Source**: interface § auth boundary + CONSTITUTION II — the write authenticates with the `X-Auth-Token`.
**Severity**: High — a leaked credential is a serious exposure.
**Probability**: Low — structurally controlled.
**Risk Level**: Yellow (High × Low).
**Controls**:
- **RC-6**: The token rides 007's transport header only — it is never a request-body field and never a model field; `reportFailure`/the diagnostics never print it; `runTensionCreate` never reads `ctx.Cred.Token`. (tasks T002/T004 acceptance forbid the token in any output or the request body.)
**Residual Risk**: Yellow (justified) — high-severity by nature, but the structural "token only in the transport header" invariant (inherited from every landed read) drives probability to Low. Acceptable with the documented control; the BDD/unit tests assert no token appears in output.

### H-7: Mis-surfaced API error
**Source**: interface-cli.md § Error Communication — `404` (unknown role), `422` (rejected body/field), `401/403/429`.
**Severity**: Medium — a write failure reported as success would mislead the agent into thinking a tension was captured when it was not.
**Probability**: Low — reuses the landed, tested failure chain.
**Risk Level**: Green (Medium × Low).
**Controls**:
- **RC-7**: Every non-2xx routes through `reportFailure` → `classifyClientError`/`ExtractProblem` (015), surfacing the HTTP status + RFC 9457 detail and exiting non-zero; a `422` classifies as `APIError(3)` with the server's detail. No outcome adds its own interpretation.
**Residual Risk**: Green.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 1 | H-6 |
| Green (accepted) | 6 | H-1, H-2, H-3, H-4, H-5, H-7 |

**Unacceptable risks**: None. All residual risks are Green or the single justified Yellow (H-6, token hygiene — controlled structurally).

**Worth the developer's attention**: H-3 (at-least-once duplicate) is Green and recoverable, but the note matters — **do not add a client-side auto-retry on the `POST`**; doing so would turn an inherent edge case into systematic duplication. RC-3's suppression of POST retry is load-bearing.

---

## Traceability Index

**Hazards → source**:
- H-1 → spec.md § Behavioral Accord / Non-Behaviors (ADR-3 id pass-through)
- H-2 → plan.md § ADR-1, § Risks
- H-3 → plan.md § Cross-cutting (Non-idempotent retry), interface-cli.md § Interactions
- H-4 → spec.md § Non-Behaviors (`sensed_by`)
- H-5 → spec.md § Behavioral Accord (`--body` required)
- H-6 → interface-cli.md § Error Communication, CONSTITUTION II
- H-7 → interface-cli.md § Error Communication

**Controls → grounding**:
- RC-1 → render output echoes `role_id` (interface-cli § Output); API 404 (§200 id pass-through)
- RC-2 → `apiclient.Request.ContentType` (plan ADR-1; tasks T001)
- RC-3 → `isSafeMethod` POST no-retry (§133; interface-spec § Retry); exit-code outcome (004)
- RC-4 → no `sensed_by` field in the request (interface-spec § request input)
- RC-5 → pre-request `--body` validation (spec § Invocation; tasks T004)
- RC-6 → token-only-in-transport invariant (007); `reportFailure` token-free (032); tasks T002/T004 acceptance
- RC-7 → `reportFailure`/`classifyClientError`/`ExtractProblem` (015/032)
