# Risk: Tension Update

**Feature**: 044-tension-update
**Round**: 1
**Date**: 2026-06-13
**Artifacts loaded**: spec.md, plan.md, interface-cli.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

---

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md.

PROJECT.md defines no Regulatory Context referencing IEC 14971; the regulatory bridge is omitted and this artifact uses plain language only.

This is a **write** capability that mutates live Holacracy governance data (`PATCH /tensions/{id}`). The dominant hazard theme is the gap between what the operator intends and what persists on the system of record: a clobbered concurrent edit, an unintended status transition, a misread failure, or a leaked secret. The plan reuses landed 042/043 infrastructure (transport `ContentType` seam, `Document[Tension]`, singular render, `tensionSeam`, `validateTensionStatus`, `validateMeetingType`) and adds a small new surface (`TensionUpdateInput`, the `update` leaf, two preconditions). Several real-world hazards are deliberately deferred by design (notably optimistic concurrency / Clobbered Changes) — those are recorded here with their accepted residual risk and the deferral rationale, not rubber-stamped away.

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | Concurrent edit silently clobbered — no `If-Match`/ETag, last-write-wins | spec.md § Non-Behaviors; PROJECT.md § Constraints (Optimistic concurrency) | High | Medium | Red | RC-1 | Yellow |
| H-2 | Unintended status transition persists (e.g. `archived` archives a live tension) | spec.md § Input; interface-cli.md § `--status` | High | Low | Yellow | RC-2, RC-3 | Yellow |
| H-3 | Present-but-empty flag yields a no-op `PATCH` that changes nothing | plan.md § Risks; plan.md § ADR-3 | Medium | Low | Green | RC-4 | Green |
| H-4 | `--status`/`--meeting-type` validator drifts from the spec enum | plan.md § Risks; interface-cli.md § Consistency Notes | Medium | Low | Green | RC-5 | Green |
| H-5 | Body silently dropped if `Content-Type` is absent → 422 / empty-body surprise | plan.md § Risks; interface-spec.md § Content type | High | Low | Yellow | RC-6 | Yellow |
| H-6 | Rate-limit (429) on a non-idempotent `PATCH` retried → duplicate mutation | plan.md § Cross-cutting (non-idempotent retry); interface-cli.md § Non-idempotent retry | High | Low | Yellow | RC-7 | Green |
| H-7 | Token leaked into output, diagnostic, or request body | spec.md § Failure; interface-spec.md § Error Communication | High | Low | Yellow | RC-8 | Yellow |
| H-8 | Unknown/malformed id (404) or rejected field (422) misreported to the operator | spec.md § Failure; interface-cli.md § Error Communication | Medium | Low | Green | RC-9 | Green |
| H-9 | Permission failure (401/403) on a write mis-surfaced as a generic error | spec.md § Failure; interface-cli.md § Error Communication | Medium | Low | Green | RC-9 | Green |
| H-10 | Server's recomputed status diverges from the sent value, confusing the operator | spec.md § Non-Behaviors; plan.md § Risks | Low | High | Yellow | RC-10 | Green |
| H-11 | Partial-update body sends an unintended field, mutating data the operator did not name | spec.md § Input; interface-spec.md § Partial body | High | Low | Yellow | RC-4, RC-11 | Yellow |
| H-12 | Render failure after a successful write leaves the operator unsure whether the edit persisted | plan.md § Cross-cutting (Output) | Medium | Low | Green | RC-12 | Green |

---

## Hazard Details

### H-1: Concurrent edit silently clobbered (last-write-wins)

**Source**: spec.md § Non-Behaviors — "must not send an `If-Match` precondition or otherwise guard against concurrent edits"; PROJECT.md § Constraints — "Mutable resources expose `ETag` headers and accept `If-Match` for optimistic locking; omitting it is last-write-wins."

**Description**: Update sends an unconditional `PATCH` with no `If-Match`. If another actor (or another agent session) edited the same tension between the operator's read and write, the operator's write silently overwrites the intervening change. No error is raised; the data simply diverges from what the operator believed they were editing.

**Severity**: High — this is a live governance record on the system of record; a clobbered status or body edit is a silent data-loss event with no audit signal at the CLI layer. The blast radius is the single tension, but the loss is undetectable to the operator.

**Probability**: Medium — the CLI's typical operator is an AI agent acting on a practitioner's behalf (PROJECT.md § Actors), and agent-driven editing raises the chance of overlapping sessions on the same backlog. The window is real, though most single-operator sessions will not collide.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-1**: The hazard is consciously deferred to a separate, named Client-Foundation capability (**Clobbered Changes**), not ignored. The spec and plan document the last-write-wins behavior explicitly and commit `update` to opt into the shared `If-Match`/ETag guard when Clobbered Changes lands, rather than growing its own. Until then the behavior is documented and matches the API's own default, so the operator is not misled about a guarantee that does not exist.

**Residual Risk**: Yellow — RC-1 does not eliminate the clobber window; it converts an unaccounted hazard into a documented, deliberately-deferred one with a named remediation path. This is acceptable **with justification**: optimistic concurrency is a cross-cutting concern relevant to every write, the project has explicitly sequenced it as its own capability, and the API itself defaults to last-write-wins when `If-Match` is omitted. The developer should note that no clobber detection exists in this command until Clobbered Changes ships — agents editing shared backlogs concurrently can lose writes silently.

### H-2: Unintended status transition persists

**Source**: spec.md § Input — "Unlike capture, status is editable here … (e.g. to `archived`)"; interface-cli.md § `--status` flag.

**Description**: `--status archived` (or any valid transition) moves a tension through its lifecycle. A wrong-but-valid value — e.g. archiving a tension that is still being worked, supplied by an agent that misjudged intent — passes local validation and is sent. The transition persists.

**Severity**: High — archiving removes a tension from the active backlog; for a practitioner this is a meaningful state change to a governance artifact, and the wrong transition is not flagged because it is a valid value.

**Probability**: Low — the value set is closed (`unprocessed`/`processed`/`archived`) and supplied explicitly per-invocation; there is no default that could silently archive. The hazard requires the operator (or agent) to supply a deliberate-but-wrong value.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-2**: Closed-enum validation (`validateTensionStatus`, 043) rejects any value outside the supported set before any request, so a typo or fabricated state never reaches the server.
- **RC-3**: No clear-to-null and no client-side recompute (spec non-behaviors) — the command forwards only the explicit, validated value and never infers a status; there is no implicit transition path.

**Residual Risk**: Yellow — RC-2/RC-3 prevent malformed or inferred transitions but cannot prevent a *valid but unintended* one (archiving the wrong tension). Acceptable with justification: the CLI is a faithful API surface (PROJECT.md § Actors — "the single source of truth the CLI never second-guesses"); intent validation beyond the enum is the operator's responsibility, and no client-side affordance could distinguish a correct archive from a mistaken one without second-guessing the operator.

### H-3: Present-but-empty flag yields a no-op `PATCH`

**Source**: plan.md § Risks — "A present-but-empty flag yields a no-op `PATCH`"; plan.md § ADR-3.

**Description**: An invocation like `update <id> --label ""` could pass a naive flag-count gate yet marshal an empty `{tension:{}}` body — a wasted request that changes nothing, contradicting the spec's "an update that changes nothing is meaningless."

**Severity**: Medium — the consequence is a wasted, side-effect-free request (and rate-limit consumption), not data loss.

**Probability**: Low — ADR-3 keys the precondition on the resolved send-set (presence `Changed()` AND non-empty value), so `--label ""` alone resolves to an empty send-set and is rejected before assembly.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-4**: The at-least-one-field precondition computes the send-set from presence + non-empty value and rejects an empty send-set as a usage error before any request; a transport tripwire pins that no request is issued on this path.

**Residual Risk**: Green — RC-4 closes the no-op path locally and is test-pinned.

### H-4: Validator drifts from the spec enum

**Source**: plan.md § Risks — "Reusing `validateTensionStatus` drifts if the spec enum changes"; interface-cli.md § Consistency Notes.

**Description**: If the vendored spec's tension-status or meeting-type enum changes and the validator's set is a private copy, `update` could accept or reject values inconsistently with the server.

**Severity**: Medium — a drifted validator either rejects a now-valid value (false usage error) or admits a now-invalid one (server 422); neither corrupts data.

**Probability**: Low — the set is single-sourced from the vendored spec enum and reused (not copied) across `list`/`create`/`update`, so one change updates all consumers at one site.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-5**: Single-sourcing the status/meeting-type sets from the vendored spec enum and reusing the landed validators (043/042) as new consumers — no second copy to drift.

**Residual Risk**: Green — RC-5 removes the duplication that would cause drift; the residual is the general staleness of the vendored spec itself, which is a project-wide concern outside this feature.

### H-5: Body silently dropped without `Content-Type`

**Source**: plan.md § Risks — "Body silently ignored without `Content-Type`"; interface-spec.md § Content type rides the descriptor.

**Description**: A body-bearing `PATCH` whose request omits `Content-Type: application/json` could have its body ignored by the server, producing a 422 or an empty-body surprise — the operator believes they edited a field that was never sent.

**Severity**: High — a dropped body means the intended edit silently does not persist; if it surfaces as 422 the operator at least sees a failure, but an ignored-body 200 would be a silent no-op presented as success.

**Probability**: Low — `runTensionUpdate` always sets `ContentType: application/json` (042 ADR-1), and 042's header-present/absent test already pins the mechanism.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-6**: The `update` request always carries the `ContentType` field (042's reused seam), and the landed 042 test pins that the header is present on a body-bearing write.

**Residual Risk**: Yellow — RC-6 is a strong, test-backed control; residual is the dependence on 042's mechanism remaining correct across future transport changes. The plan should ensure a focused 044-level assertion (or explicit reliance on 042's test) that the `PATCH` body rides with the header. Acceptable with justification: the mechanism is landed and pinned; the residual is regression-only.

### H-6: 429 on a non-idempotent `PATCH` retried into a duplicate mutation

**Source**: plan.md § Cross-cutting (non-idempotent retry, §133); interface-cli.md § Non-idempotent retry.

**Description**: The retry executor auto-retries 429s. If a `PATCH` were retried, a non-idempotent write could be applied twice (duplicate mutation), or at least re-sent against the operator's intent.

**Severity**: High — a duplicated governance mutation is a real-world integrity hazard for a write.

**Probability**: Low — 017's `isSafeMethod` restricts auto-retry-on-429 to `GET`/`HEAD` (`retry.go:65`), so a `PATCH` is never auto-retried; the 429 surfaces on first occurrence.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-7**: The landed `isSafeMethod` gate makes `PATCH` non-retryable with no command-side special-casing; the plan commits a focused test pinning that a `PATCH` 429 is surfaced, not retried.

**Residual Risk**: Green — RC-7 structurally prevents the retry (method gate, not a per-command opt-out) and is test-pinned. The architectural safeguard reduces probability to effectively nil for this path, so residual is Green.

### H-7: Token leaked into output, diagnostic, or body

**Source**: spec.md § Failure — "never includes the token"; interface-spec.md § Error Communication — "No secret anywhere."

**Description**: A write command handles a credential; a leak into stdout, a diagnostic message, or the request body would expose the `X-Auth-Token` (PROJECT.md § Constraints — one key = one org + person).

**Severity**: High — a leaked token grants the caller's full org-scoped permissions.

**Probability**: Low — `runTensionUpdate` never reads `ctx.Cred.Token`, the request body carries no credential, and the auth header is owned by the 007 transport seam; the failure path reuses `reportFailure` which is built to exclude the token.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-8**: The command never reads the token field; the auth header is attached by the 007 transport seam outside the command; no message or projection renders the token; the body envelope (`{tension:{…}}`) carries only field values.

**Residual Risk**: Yellow — RC-8 covers the known surfaces; residual is the general possibility of an unanticipated diagnostic path including a token-bearing value. Acceptable with justification: the command structurally never holds the token, and the no-secret discipline is a landed, project-wide invariant. A scenario asserting no token appears in any output/diagnostic would close the residual.

### H-8: 404/422 misreported

**Source**: spec.md § Failure; interface-cli.md § Error Communication.

**Description**: An unknown/malformed id (404) or a rejected field/value (422) must be reported with the HTTP status and a next step. A misclassification or a swallowed status would leave the operator unable to tell why the edit failed.

**Severity**: Medium — a misreported failure costs the operator a diagnostic round-trip; no data is corrupted (the write did not apply).

**Probability**: Low — failures route through the landed `reportFailure` → `refineClientError`/`classifyClientError` chain (015), which names the HTTP status and surfaces the RFC 9457 detail; the command adds no interpretation.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-9**: The shared classifier (015) maps 404/422 to `APIError(3)` with the status and extracted detail surfaced, and the exhaustiveness guard (`len`+comma-ok) on the classifier table keeps the mapping complete.

**Residual Risk**: Green — RC-9 is landed and exhaustiveness-guarded.

### H-9: Permission failure (401/403) mis-surfaced

**Source**: spec.md § Failure; interface-cli.md § Error Communication; PROJECT.md § Constraints (Premium-gated, single-identity permissions).

**Description**: A write may be refused for permission (401/403) — the caller's membership does not allow the edit, or a premium-gated path returns 403. Surfacing this as a generic API error would obscure the real cause.

**Severity**: Medium — a mislabelled permission error misdirects the operator's remediation; no data effect.

**Probability**: Low — 015's landed split routes 401/403 to `PermissionError(4)` distinctly from the generic `APIError(3)`; `update` benefits with no edit.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-9**: The shared classifier distinguishes `PermissionError(4)` (401/403) and `RateLimited(5)` (429) from the generic `APIError(3)`, surfacing the right outcome and exit code.

**Residual Risk**: Green — covered by the landed, exhaustiveness-guarded classifier.

### H-10: Server-recomputed status diverges from sent value

**Source**: spec.md § Non-Behaviors — "must not perform its own status auto-computation, nor reconcile … against the server's recompute"; plan.md § Risks.

**Description**: The API re-runs status auto-computation on save, so the value returned may differ from `--status`. An operator who expected the sent value to stick could be confused.

**Severity**: Low — by design the command renders whatever the server returns; there is no data hazard, only a possible expectation mismatch.

**Probability**: High — the API is documented to recompute on save, so divergence is expected behavior, not an error.

**Risk Level**: Yellow (Low × High)

**Controls**:
- **RC-10**: The command renders the server's returned tension verbatim and claims no authority over the final status (spec non-behavior); the operator always sees the persisted truth, not a client prediction.

**Residual Risk**: Green — RC-10 makes the rendered status the authoritative server value, eliminating any false client-side claim. The only residual is operator expectation, addressable by interface/help wording; no system hazard remains. Reduced to Green: the matrix yellow (Low×High) reflects frequency, but rendering the true server state removes the actual harm.

### H-11: Partial-update body sends an unintended field

**Source**: spec.md § Input — "the system sends only those fields (partial update)"; interface-spec.md § Partial body, presence-faithful.

**Description**: If the send-set logic were wrong, a field the operator did not name could ride in the body and mutate data unintentionally (the inverse of H-3) — e.g. a stale or defaulted value sent as if supplied.

**Severity**: High — sending an unnamed field overwrites server data the operator did not intend to touch.

**Probability**: Low — the send-set is computed from `Changed()` + non-empty value and marshalled with `omitempty`; only supplied, non-empty fields ride. The plan commits a marshalling test asserting only supplied fields appear and no `If-Match` is sent.

**Risk Level**: Yellow (High × Low)

**Controls**:
- **RC-4**: Presence-aware send-set construction (`Changed()` + non-empty) ensures only supplied fields are eligible.
- **RC-11**: A focused marshalling test asserts the `{tension:{…}}` body carries only the supplied fields (and no `If-Match` header).

**Residual Risk**: Yellow — RC-4/RC-11 are strong and test-pinned; residual is regression risk in the presence-detection logic. Acceptable with justification: the cobra-presence discipline is a settled, repeatedly-reviewed codebase pattern and the marshalling assertion guards exactly this surface. (See also the codebase learning that presence must be gated on `Changed()`, not value.)

### H-12: Render failure after a successful write

**Source**: plan.md § Cross-cutting (Output) — "Buffer-then-write so a render failure leaves stdout empty and maps to `RuntimeError(1)`."

**Description**: The write succeeds server-side, but rendering the response fails. If partial output were emitted, the operator could not tell whether the edit persisted.

**Severity**: Medium — the edit did persist; the hazard is operator uncertainty, not data loss.

**Probability**: Low — buffer-then-write ensures a render failure leaves stdout empty and maps cleanly to `RuntimeError(1)`.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-12**: Buffer-then-write — the rendered result is only written to stdout once rendering succeeds; a render failure yields empty stdout and a `RuntimeError(1)` diagnostic.

**Residual Risk**: Green — RC-12 makes the failure unambiguous (no partial output). Residual is the operator still needing a follow-up read to confirm the persisted state, which is inherent to a non-transactional write and outside this command's scope.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 5 | H-1, H-2, H-5, H-7, H-11 |
| Green (accepted) | 7 | H-3, H-4, H-6, H-8, H-9, H-10, H-12 |

**Unacceptable risks**: None remain Red after controls. **H-1 (concurrent-edit clobber)** is the most significant residual: it sits at Red before controls and is reduced to Yellow only by the deliberate-deferral control (RC-1), not by an active guard. The developer should be explicit that **no clobber detection exists in this command until the Clobbered Changes capability ships** — concurrent edits to the same tension can silently lose a write. This is consistent with the API's own last-write-wins default and PROJECT.md's stated concurrency model, but it is a real, undetectable data-loss window that the deferral makes a conscious, documented choice rather than an oversight.

**Hazards lacking an adequate active control**: 1 — H-1. Its only control is documentation + deferral (RC-1); there is no in-command mechanism that detects or prevents the clobber. Every other hazard has an active, mostly landed-and-test-pinned control.

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Non-Behaviors; PROJECT.md § Constraints (Optimistic concurrency) |
| H-2 | spec.md § Input; interface-cli.md § `--status` flag |
| H-3 | plan.md § Risks; plan.md § ADR-3 |
| H-4 | plan.md § Risks; interface-cli.md § Consistency Notes |
| H-5 | plan.md § Risks; interface-spec.md § Content type rides the descriptor |
| H-6 | plan.md § Cross-cutting Concerns (non-idempotent retry); interface-cli.md § Non-idempotent retry |
| H-7 | spec.md § Failure; interface-spec.md § Error Communication |
| H-8 | spec.md § Failure; interface-cli.md § Error Communication |
| H-9 | spec.md § Failure; interface-cli.md § Error Communication; PROJECT.md § Constraints |
| H-10 | spec.md § Non-Behaviors; plan.md § Risks |
| H-11 | spec.md § Input; interface-spec.md § Partial body, presence-faithful |
| H-12 | plan.md § Cross-cutting Concerns (Output) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | spec.md § Non-Behaviors — Clobbered Changes deferral; PROJECT.md § Constraints — last-write-wins is the API default |
| RC-2 | H-2 | plan.md § ADR-2 — reuses landed `validateTensionStatus` (043) closed-enum validation |
| RC-3 | H-2 | spec.md § Non-Behaviors — no client recompute, no clear-to-null |
| RC-4 | H-3, H-11 | plan.md § ADR-3 — presence-aware send-set (`Changed()` + non-empty); transport tripwire |
| RC-5 | H-4 | interface-cli.md § Consistency Notes — single-sourced enum, reused validators |
| RC-6 | H-5 | plan.md § ADR-1 / Cross-cutting — `ContentType` seam (042), header test pinned |
| RC-7 | H-6 | plan.md § Cross-cutting — `isSafeMethod` 429 gate (`retry.go:65`); focused PATCH-429 test |
| RC-8 | H-7 | interface-spec.md § Error Communication — command never reads `ctx.Cred.Token`; 007 owns the auth header |
| RC-9 | H-8, H-9 | interface-cli.md § Error Communication — landed `classifyClientError`/015 split, exhaustiveness-guarded |
| RC-10 | H-10 | spec.md § Non-Behaviors — renders server-returned status verbatim, claims no authority |
| RC-11 | H-11 | plan.md § Testing — marshalling test asserts only supplied fields and no `If-Match` |
| RC-12 | H-12 | plan.md § Cross-cutting (Output) — buffer-then-write, render failure → `RuntimeError(1)` |
