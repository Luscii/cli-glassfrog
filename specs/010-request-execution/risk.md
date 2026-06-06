# Risk: Request Execution

**Feature**: 010-request-execution
**Round**: 1
**Date**: 2026-06-06
**Artifacts loaded**: spec.md, plan.md, interface-spec.md, PROJECT.md
**Acceptability matrix**: Default 3×3 traffic light

> ⚠ Using default risk acceptability matrix — no project-level matrix found in PROJECT.md. (No Regulatory Context either — plain language only, no IEC 14971 bridge.)

---

## Risk Register

| H-ID | Hazard | Source | Severity | Probability | Risk Level | Controls | Residual Risk |
|---|---|---|---|---|---|---|---|
| H-1 | Token leaks into request logs / error output | spec.md § Non-Behaviors | High | Medium | Red | RC-1 | Yellow |
| H-2 | 007's `AuthError` mislabeled as a transport error → wrong exit code | plan.md § ADR-4 / Risks | Medium | Medium | Yellow | RC-2 | Green |
| H-3 | Non-2xx response treated as success / decoded into the target | spec.md § Behavioral Accord | High | Medium | Red | RC-3 | Yellow |
| H-4 | Response body not closed → fd / connection leak | plan.md § Cross-cutting / Risks | Medium | Medium | Yellow | RC-4 | Green |
| H-5 | Hung connection blocks forever (no timeout) | spec.md § Driving Scenarios (edge) | Medium | High | Red | RC-5 | Green |
| H-6 | `429` not honored → whole organization throttled | spec.md § Non-Behaviors; CONSTITUTION X | High | Medium | Red | RC-6 | **Yellow (conditional)** |
| H-7 | Base-URL error not refused → request sent to a bad/unintended endpoint | spec.md § Behavioral Accord | Medium | Low | Green | RC-7 | Green |
| H-8 | Decode failure returns a fabricated zero-valued target | spec.md § Behavioral Accord; CONSTITUTION VIII | Medium | Low | Green | RC-8 | Green |
| H-9 | Unbounded response body read into memory | interface-spec.md § Error outcomes | Low | Low | Green | — | Green (accepted) |

---

## Hazard Details

### H-1: Token leaks into request logs / error output

**Source**: spec.md § Non-Behaviors — "must not print, log, or expose the token value."

**Description**: The request path is exactly where secrets leak — request tracing, verbose logging, or an error string that echoes a header could expose the `X-Auth-Token`, compromising the org+person credential.

**Severity**: High — a leaked token is an org-wide credential compromise (CONSTITUTION II secret hygiene).

**Probability**: Medium (inherent) — without discipline, the request layer is the most likely leak site.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-1**: 010 never reads `ctx.Cred.Token` — the replay thunk handed to `AuthTransport` is the token's only path, and 007 owns header attachment. No 010 error type carries the token (`TransportError` = network cause; `ResponseError` = the *response* side; `DecodeError` = parse cause). Pinned by the "token never appears in any output" validation scenario.

**Residual Risk**: Yellow (High × Low) — with RC-1 the token is structurally out of 010's reach; residual is the general possibility of a future edit introducing a log line. Acceptable with the pinned token-never-in-output test as the regression guard.

### H-2: 007's `AuthError` mislabeled as a transport error

**Source**: plan.md § ADR-4 and § Risks.

**Description**: `AuthTransport` returns its `*AuthError` (no/broken credential) as the `error` from `Do`, the same channel a genuine wire failure uses. If `Execute` wraps indiscriminately, a "cannot authenticate" becomes a "network unavailable," misdirecting the agent's recovery (retry the network vs. fix credentials) and the exit code.

**Severity**: Medium — wrong diagnosis and exit code; no data corruption.

**Probability**: Medium (inherent) — an easy mistake when both arrive as `error`.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-2**: `errors.As(&AuthError)` runs **before** any wrapping; the `AuthError` is propagated unchanged, everything else becomes `TransportError`. Pinned by the no-token-context scenario asserting the returned error is `*AuthError`, not `*TransportError`.

**Residual Risk**: Green (Medium × Low) — the discrimination is explicit and test-pinned.

### H-3: Non-2xx response treated as success / decoded into the target

**Source**: spec.md § Behavioral Accord (non-2xx short-circuit).

**Description**: If a 4xx/5xx body were decoded into the success target, the caller would act on a non-success payload as if valid — wrong governance reads (CONSTITUTION III/VIII).

**Severity**: High — the agent acts on bad data.

**Probability**: Medium (inherent) — naive `Do`-then-decode would do exactly this.

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-3**: a status-class check short-circuits every non-2xx to a generic `ResponseError{status,headers,body}` **before** any decode; the body is never decoded into the success target. Pinned by the "a non-2xx body is never decoded into the success target" validation scenario.

**Residual Risk**: Yellow (High × Low) — structurally prevented and test-pinned; residual is a future change moving the decode ahead of the status check. The validation test is the guard.

### H-4: Response body not closed → fd / connection leak

**Source**: plan.md § Cross-cutting Concerns and § Risks.

**Description**: The non-2xx and decode-error branches are the easy ones to forget to close; over a long agent session the fd / connection pool starves.

**Severity**: Medium — resource exhaustion, not data loss.

**Probability**: Medium (inherent) — a classic Go branch-specific omission.

**Risk Level**: Yellow (Medium × Medium)

**Controls**:
- **RC-4**: a single `defer resp.Body.Close()` immediately after a non-nil response, before branching, covering every path. Exercised by the every-branch `httptest` tests.

**Residual Risk**: Green (Medium × Low).

### H-5: Hung connection blocks forever (no timeout)

**Source**: spec.md § Driving Scenarios (edge — "hung connection fails on the request timeout").

**Description**: Without a timeout, a connection that accepts but never responds blocks the CLI indefinitely, stalling the operating agent (Go's default client has no timeout — PR #20 LEARNINGS).

**Severity**: Medium — automation stalls; no data corruption.

**Probability**: High (inherent) — no timeout means *any* hung peer hangs forever.

**Risk Level**: Red (Medium × High)

**Controls**:
- **RC-5**: a client-level `requestTimeout` bounds the single attempt; a hung-connection test asserts a `TransportError` on timeout.

**Residual Risk**: Green (Medium × Low) — bounded by construction. Residual is only timeout-value tuning (the `[ASSUMED]` constant), which is non-blocking.

### H-6: `429` not honored → whole organization throttled

**Source**: spec.md § Non-Behaviors (no retry/backoff — deferred to 017); CONSTITUTION X (Respect API Limits); checklist.md cross-spec sequencing note.

**Description**: 010 surfaces a `429` as a generic `ResponseError` carrying the rate-limit headers but does **not** back off — backoff is Rate-Limit Handling (017). In the BACKLOG, 017 is Should-tier and ranks **after** the Must-tier reads (011–014). So reads built on 010 before 017 lands would ignore `Retry-After`. Exceeding the per-org rolling 1-hour limit throttles the **entire organization** (CONSTITUTION X rationale).

**Severity**: High — the blast radius is the whole org, not one caller.

**Probability**: Medium (inherent, ignoring 010's controls).

**Risk Level**: Red (High × Medium)

**Controls**:
- **RC-6**: 010 makes **exactly one attempt and never retries**, so it cannot create a retry storm that amplifies throttling; and it **surfaces the 429 + rate-limit headers** on `ResponseError`, giving 017 everything it needs to honor `Retry-After`. (010 cannot *itself* back off — that is 017's capability by design.)

**Residual Risk**: **Yellow (conditional)** (High × Low) — justified for the initial read slice: agent-driven governance reads are low-volume, one-attempt-no-retry means no amplification, and the limit is a rolling 1-hour window that self-heals. **The justification holds only while read usage stays low-volume.** If high-volume or looping reads ship on 010 **before** 017, probability rises to Medium and residual becomes **Red**. → **Developer decision required** (see Residual Risk Summary): either pull 017 forward of / alongside the first reads, or accept this Yellow with the low-volume justification documented.

### H-7: Base-URL error not refused → request sent to a bad/unintended endpoint

**Source**: spec.md § Behavioral Accord (base-URL fail-fast).

**Description**: If a carried base-URL error (malformed config) were not refused, 010 could send to an unintended endpoint or surface a confusing wire error that masks the real config problem.

**Severity**: Medium.

**Probability**: Low — `NewClient` returns `ctx.BaseURLErr` and builds nothing by design.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-7**: `NewClient` refuses on `ctx.BaseURLErr` before constructing or sending; pinned by the "base-URL problem is refused before sending" scenario.

**Residual Risk**: Green — fail-fast at construction.

### H-8: Decode failure returns a fabricated zero-valued target

**Source**: spec.md § Behavioral Accord (decode error); CONSTITUTION VIII (No Fabricated Data).

**Description**: If a 2xx body that doesn't match the target were swallowed, the caller would receive a zero-valued struct presented as real data.

**Severity**: Medium.

**Probability**: Low — design surfaces a `DecodeError` rather than a zero value.

**Risk Level**: Green (Medium × Low)

**Controls**:
- **RC-8**: an undecodable 2xx body returns a `DecodeError`, never a zero-valued target; pinned by the "undecodable success body is surfaced as a decode error" scenario.

**Residual Risk**: Green.

### H-9: Unbounded response body read into memory

**Source**: interface-spec.md § Error outcomes (`ResponseError.Body []byte`); spec decode-into-target.

**Description**: 010 reads the full non-2xx body into `[]byte` and decodes the full 2xx body; a very large response would spike memory.

**Severity**: Low — bounded in practice by the API's paginated responses (CONSTITUTION VI; per-page limits).

**Probability**: Low — Glassfrog responses are paginated and size-limited per page.

**Risk Level**: Green (Low × Low)

**Controls**: — (accepted). Pagination (016) governs result-set size; a per-response read of one page is bounded.

**Residual Risk**: Green (accepted) — revisit only if a non-paginated large-payload endpoint is added.

---

## Residual Risk Summary

| Level | Count | Hazards |
|---|---|---|
| Red (unacceptable) | 0 | — |
| Yellow (justified) | 3 | H-1, H-3, H-6 |
| Green (accepted) | 6 | H-2, H-4, H-5, H-7, H-8, H-9 |

**Unacceptable risks**: None are Red *after controls*. **One residual demands an explicit decision — H-6** (`429`/org-throttle): it is Yellow **only while read usage stays low-volume**, and becomes **Red** if high-volume or looping reads ship on 010 before Rate-Limit Handling (017). Recommended: schedule 017 close to the first reads (011–014), or record the low-volume acceptance for the initial slice. H-1 and H-3 are Yellow guarded by their held-out `@validation` regression tests (token-never-in-output; non-2xx-never-decoded) — keep those tests load-bearing.

---

## Traceability Index

### Hazards

| ID | Source |
|---|---|
| H-1 | spec.md § Non-Behaviors (secret hygiene) |
| H-2 | plan.md § ADR-4 / § Risks |
| H-3 | spec.md § Behavioral Accord (non-2xx short-circuit) |
| H-4 | plan.md § Cross-cutting Concerns / § Risks |
| H-5 | spec.md § Driving Scenarios — edge (timeout) |
| H-6 | spec.md § Non-Behaviors (no backoff → 017); CONSTITUTION X |
| H-7 | spec.md § Behavioral Accord (base-URL fail-fast) |
| H-8 | spec.md § Behavioral Accord (decode error); CONSTITUTION VIII |
| H-9 | interface-spec.md § Error outcomes (`ResponseError.Body`) |

### Controls

| ID | Mitigates | Grounding |
|---|---|---|
| RC-1 | H-1 | plan.md § Cross-cutting (secret hygiene — 010 never reads the token; replay thunk only) |
| RC-2 | H-2 | plan.md § ADR-4 (`errors.As(*AuthError)` before wrapping) |
| RC-3 | H-3 | plan.md § ADR-3 (status-class short-circuit to generic `ResponseError`, no decode) |
| RC-4 | H-4 | plan.md § Cross-cutting (single `defer resp.Body.Close()` on every branch) |
| RC-5 | H-5 | plan.md § ADR-4 (client-level `requestTimeout`, one bounded attempt) |
| RC-6 | H-6 | plan.md § ADR-3/ADR-4 (one attempt, no retry; surface 429 + rate-limit headers for 017) |
| RC-7 | H-7 | plan.md § ADR-2 (`NewClient` base-URL fail-fast) |
| RC-8 | H-8 | plan.md § ADR-3 (`DecodeError` rather than a zero-valued target) |
