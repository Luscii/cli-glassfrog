# Risk: API Error Extraction

**Feature**: 015-api-error-extraction
**Round**: 1
**Date**: 2026-06-07
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. (No Regulatory Context either — plain language only, no IEC 14971 bridge.)

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | `ExtractProblem` panics / returns nil on a malformed body → the failure re-opacifies (crash or no error) | spec.md § Behavioral Accord (degradation); plan.md § ADR-2 | High | Medium | Red | RC-1 | Yellow |
| H-2 | `429` not honored → whole organization throttled (carried from 010 H-6) | spec.md § Non-Behaviors; CONSTITUTION X; checklist.md | High | Medium | Red | RC-2 | **Yellow (conditional)** |
| H-3 | Status-derived fallback `Detail` mistaken for the API's own words (fabrication) | plan.md § ADR-2; checklist.md P1 (VIII) | Medium | Medium | Yellow | RC-3 | Green *(controlled — round 2)* |
| H-4 | API-error message gives the cause but no next step → operator can't determine recovery | interface-spec.md § Error Communication; checklist.md P1 (II) | Medium | Medium | Yellow | RC-4 | Green *(controlled — round 2)* |
| H-5 | Classifier ordering bug → 401/403 fall through to `APIError`(3), permission distinction lost | plan.md § ADR-1/ADR-3 / Risks; interface-spec.md | Medium | Medium | Yellow | RC-5 | Green |
| H-6 | Body `status` member overrides the HTTP status → misclassification | spec.md § Behavioral Accord (status authority); plan.md § ADR-2 | Medium | Low | Green | RC-6 | Green |
| H-7 | 401/403 exit-code change (3→4) breaks a caller scripting on exit 3 | tasks.md T002; checklist.md / analyze.md cross-spec | Low | Low | Green | RC-7 | Green |
| H-8 | Surfaced `detail`/`title` leaks unexpected content into output/logs | interface-spec.md § Error Communication | Low | Low | Green | RC-8 | Green |

---

## Hazard Details

### H-1: `ExtractProblem` panics / returns nil on a malformed body

**Source**: spec.md § Behavioral Accord ("never fail to produce a typed error because the body could not be parsed"); plan.md § ADR-2.

**Description**: The whole point of 015 is to make a failed call legible. If `ExtractProblem` is not *total* — a nil-deref on an empty body, a panic on truncated JSON, a nil return on a non-conformant body — then a non-2xx with a junk body (exactly the gateway/proxy case) would crash the CLI or produce no error, re-creating the opacity the capability exists to remove (CONSTITUTION III, Fail Safe).

**Severity**: High — a crash on the error path defeats the capability and violates Fail-Safe.

**Probability**: Medium (inherent) — empty, HTML, and truncated bodies are common from gateways; naive decoding is easy to get subtly wrong.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-1**: `ExtractProblem` is **total by design** — best-effort parse, graceful degradation to a status-derived fallback, raw body preserved, never returns nil and never panics (plan ADR-2). Pinned by the empty / HTML / missing-members table tests (tasks T001) and the held-out `@validation` scenario "Every non-2xx yields a typed error."

**Residual Risk**: Yellow (High × Low) — structurally prevented and test-pinned; residual is a future edit reintroducing a non-total path. The validation test is the regression guard.

### H-2: `429` not honored → whole organization throttled (carried from 010 H-6)

**Source**: spec.md § Non-Behaviors (429 backoff deferred to 017); CONSTITUTION X; checklist.md cross-spec sequencing note.

**Description**: 015 extracts a `429`, classifies it to `RateLimited`(5) (a distinct, scriptable exit code), but does **not** back off — backoff is Rate-Limit Handling (017), which is Should-tier and ranks after the Must-tier reads. So the org-throttle window 010's H-6 identified persists through 015 unchanged: reads built on the seam before 017 lands ignore `Retry-After`, and exceeding the per-org rolling 1-hour limit throttles the **entire organization**.

**Severity**: High — blast radius is the whole org.

**Probability**: Medium (inherent, before 017).

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-2**: 015 never retries (no amplification) and preserves the `429` + rate-limit headers on the wrapped `ResponseError`, so 017 has everything it needs to honor `Retry-After`. 015 cannot itself back off — that is 017's capability by design.

**Residual Risk**: **Yellow (conditional)** (High × Low) — identical justification to 010 H-6: low-volume agent reads, no retry amplification, a self-healing rolling window. **Holds only while read usage stays low-volume.** → **Developer decision (unchanged from 010):** pull 017 forward of / alongside the first reads, or record the low-volume acceptance. 015 does not change this posture (it neither worsens nor resolves it).

### H-3: Status-derived fallback `Detail` mistaken for the API's own words

**Source**: plan.md § ADR-2 (fallback `Detail` derived from `http.StatusText`); checklist.md P1 (CONSTITUTION VIII).

**Description**: When the body isn't parseable, `ExtractProblem` fills `Detail` with a status-derived phrase (e.g. "Bad Gateway") placed in the **same field** that otherwise carries the API's own `detail`, with no provenance marker. A consumer or operator can't tell "the API said this" from "the CLI derived this from the status" — a synthesized value presented as real (CONSTITUTION VIII).

**Severity**: Medium — misleading diagnosis; the value is a real-status rendering, not invented governance data, so no governance corruption.

**Probability**: Medium — unparseable bodies are common, so the fallback fires often.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-3** *(controlled — round 2)*: the fallback derives **only from the real HTTP status** (`http.StatusText`), never from invented API content; **and** `ProblemError` now carries a **`DetailSynthesized bool`** provenance marker (true when `Detail` is the fallback, false when it is the API's own). The consumer keys its message on the flag, so a synthesized detail is never rendered as the API's words. (PR #37 triage addressed the checklist VIII P1.)

**Residual Risk**: Green (Medium × Low) — the no-invented-content guarantee plus the explicit provenance marker close the same-field ambiguity; a test pins that a synthesized-detail case renders the fallback wording, not the synthesized text.

### H-4: API-error message gives the cause but no next step

**Source**: interface-spec.md § Error Communication; checklist.md P1 (CONSTITUTION II).

**Description**: The new message surfaces the API's `detail` (what went wrong) but specifies no next-step hint, while II requires "what went wrong **and** the next step" and the sibling message arms (auth/transport/base-URL) all name one. The operating agent sees the cause but not the recovery action — most acute for `PermissionError` (401/403), where "check the token's access / membership" is an obvious, omitted step.

**Severity**: Medium — degraded recovery guidance; the cause is shown, so not opaque.

**Probability**: Medium — every API-error message lacks the next step as currently specified.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-4** *(controlled — round 2)*: the API-error message now appends a next-step hint, at minimum for the permission class (401/403 → "check the token's access / membership"), in `formatClientErrorMessage` (interface-spec.md § Error Communication; tasks T002). The `detail` provides "what went wrong"; the hint provides the next step. (PR #37 triage addressed the checklist II P1.)

**Residual Risk**: Green (Medium × Low) — both halves of II are satisfied and the arm matches its siblings; a test pins that a permission case shows the next-step hint.

### H-5: Classifier ordering bug → 401/403 fall through to `APIError`(3)

**Source**: plan.md § ADR-1/ADR-3 and § Risks; interface-spec.md § Consumer contract changes.

**Description**: `ProblemError` wraps `ResponseError`, so `errors.As(err, &responseErr)` matches a `*ProblemError` too. If `classifyClientError` evaluates the generic `*ResponseError` arm before the status branch, every 401/403 falls through to `APIError`(3) and the permission distinction silently never fires — the operator can't tell "fix your token/permissions" from a generic API error, and the reserved code 4 stays unused.

**Severity**: Medium — wrong recovery guidance and a silently dead split; no data corruption.

**Probability**: Medium (inherent) — the wrapping makes the ordering bug easy.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-5**: branch on status within the `*ProblemError`/`*ResponseError` arm (or check `*ProblemError` first), **before** the generic arm — the same discrimination-order discipline 010/011 apply for `*AuthError` before `*TransportError`. Pinned by a test asserting a 401/403 maps to `PermissionError`(4), not `APIError`(3) (tasks T002 acceptance).

**Residual Risk**: Green (Medium × Low) — explicit and test-pinned.

### H-6: Body `status` member overrides the HTTP status

**Source**: spec.md § Behavioral Accord (status authority); plan.md § ADR-2.

**Description**: If the in-body `status` were treated as authoritative, a body claiming `200`/`401` on a `500`/`403` HTTP response would misclassify the exit code and the message.

**Severity**: Medium.

**Probability**: Low — design pins the HTTP status authoritative.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-6**: `ProblemError.StatusCode` is always set from the wrapped `ResponseError`'s HTTP status; the body `status` is carried as metadata only — **now in an explicit `BodyStatus` field** (added in the PR #37 triage round, which found the metadata had no home on the surface), so the "carried as metadata only" assertion is observable rather than unsatisfiable. Pinned by the status-mismatch driving scenario and the "produced status always equals the HTTP status" validation scenario.

**Residual Risk**: Green — authoritative by construction, metadata exposed via `BodyStatus`, test-pinned.

### H-7: 401/403 exit-code change (3→4) breaks a caller scripting on exit 3

**Source**: tasks.md T002; checklist.md / analyze.md cross-spec observation.

**Description**: The landed reads (011–014) currently exit `3` for 401/403; 015 changes them to `4`. A downstream script or agent branching on exit `3` for "permission denied" would behave differently.

**Severity**: Low — `4` is a published, reserved code (004's frozen convention); agents are expected to handle the documented codes.

**Probability**: Low — the CLI is young, `4` was always the reserved permission code, and few consumers are likely hardcoding `3` for 401/403 this early.

**Risk Level**: Green (Low × Low)

**Controls**:
- **RC-7**: the change takes a **reserved, published** code (not a renumber); it is documented in interface-spec.md's status→exit-code table and DECISIONS.md. *(Visibility, not a safety control:)* surface the 3→4 change in the PR description so a consumer scripting on exit codes sees it — the developer-confirmation already flagged in shape/checklist.

**Residual Risk**: Green (accepted) — intended, reserved-code change; the only action is visibility.

### H-8: Surfaced `detail`/`title` leaks unexpected content into output/logs

**Source**: interface-spec.md § Error Communication (the message now renders API-supplied text).

**Description**: The message now surfaces the API's `detail`/`title`. The token is structurally absent (015 reads only the response side — the `X-Auth-Token` is a request header, never echoed), but API-supplied text is now rendered to stderr.

**Severity**: Low — the rendered text is the API's own reply, not the secret credential.

**Probability**: Low — Glassfrog's RFC 9457 `detail` is a short human summary; the token cannot appear (response-side only).

**Risk Level**: Green (Low × Low)

**Controls**:
- **RC-8**: 015 reads only response-side fields (status/headers/body); it never reads `ctx.Cred.Token`. The message renders `detail`/`title`/`status` only. A token-never-in-output test (mirroring 010's) covers `ProblemError.Error()` and the rendered message.

**Residual Risk**: Green — the token is out of reach by construction.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 2 | H-1, H-2 |
| Green (accepted) | 6 | H-3, H-4, H-5, H-6, H-7, H-8 |

> **Round 2 (PR #37 triage):** H-3 (fallback-detail provenance) and H-4 (missing next-step) moved **Yellow → Green** — their recommended controls (the `DetailSynthesized` marker, the permission next-step hint) are now in the design (RC-3, RC-4). H-6's control gained the explicit `BodyStatus` field. Two residuals remain Yellow.

**Unacceptable risks**: None are Red *after controls*. Two residuals warrant attention:

- **H-1 (totality)** — controlled by design (the total `ExtractProblem` + the `@validation` "every non-2xx yields a typed error" test); Yellow only as the standing reminder that a future edit must not reintroduce a non-total path. Keep the test load-bearing.
- **H-2 (`429` org-throttle)** — the carried-forward **developer decision** from 010 H-6, unchanged by 015: pull Rate-Limit Handling (017) forward of / alongside the reads, or record the low-volume acceptance. 015 neither worsens nor resolves it.

*(H-3, H-4, H-5, H-6 are now Green; H-5's 401→PermissionError test stays load-bearing.)*

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Behavioral Accord (graceful degradation); plan.md § ADR-2 |
| H-2 | spec.md § Non-Behaviors (429 → 017); CONSTITUTION X; checklist.md |
| H-3 | plan.md § ADR-2 (status-derived fallback); checklist.md P1 (VIII) |
| H-4 | interface-spec.md § Error Communication; checklist.md P1 (II) |
| H-5 | plan.md § ADR-1/ADR-3 / § Risks; interface-spec.md § Consumer contract changes |
| H-6 | spec.md § Behavioral Accord (status authority); plan.md § ADR-2 |
| H-7 | tasks.md T002; checklist.md / analyze.md cross-spec observation |
| H-8 | interface-spec.md § Error Communication |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | plan.md § ADR-2 (total, fail-soft `ExtractProblem`); tasks T001 degradation tests + `@validation` totality scenario |
| RC-2 | H-2 | plan.md § Integration Design (no retry; preserve 429 + rate-limit headers for 017) |
| RC-3 | H-3 | plan.md § ADR-2 (fallback derives from real status only) + checklist-recommended provenance marker (pending) |
| RC-4 | H-4 | interface-spec.md § Error Communication (detail surfaced) + checklist-recommended next-step hint (pending) |
| RC-5 | H-5 | plan.md § ADR-1 (check `*ProblemError`/status before the generic arm); tasks T002 401→PermissionError test |
| RC-6 | H-6 | plan.md § ADR-2 (HTTP status authoritative; body status metadata-only); status-mismatch + validation scenarios |
| RC-7 | H-7 | exitcode.go/interface-spec.md (reserved published code 4, not a renumber) + PR-description visibility |
| RC-8 | H-8 | plan.md § Cross-cutting (response-side only; never reads the token); token-never-in-output test |
