# Checklist: Request Execution

**Feature**: 010-request-execution
**Checked against**: CONSTITUTION.md (done-* accords not found — see Governance Notes)
**Artifacts checked**: spec.md, plan.md, interface-spec.md, features/no-shared-api-client/request-execution.feature, tasks.md
**Checks**: 11 (11 pass, 0 fail)
**Generated**: 2026-06-06

---

## Summary

All 11 checks pass. Constitution: 11/11. Done-criteria: not checked (no accords). Cross-reference: not checked (no accords).

One constitution principle (XI Governance via Proposals) produced no applicable check — the slice is a generic transport seam with no command and no governance-mutating path. Three principles (VI Size-Aware, X Respect API Limits, II Action Transparency) were calibrated to the API-client seam; see Calibration notes. A cross-spec **sequencing observation** on X (429 backoff lands in 017, after the Must-tier reads) is recorded in Governance Notes — it is not a 010 artifact defect.

---

## Constitution Checks: 11/11 passed

### Calibration notes

- **II. Action Transparency** — this slice has no command surface and returns typed values, so "report the spec operation + target resource, machine-parseable" is calibrated to "`Execute` returns a structured `Response{StatusCode, Header}` on success and typed errors (`TransportError`/`ResponseError{StatusCode,Header,Body}`/`DecodeError`, plus the propagated `AuthError`) that name the failure cause"; the *operation + resource id* line and the user-facing *next-step* message + exit code are owned by the consuming command (011), which knows it called e.g. `GET /me` and maps the typed error to a code (deferred, code-free split — consistent with 007/008/009). *Secret-hygiene nuance:* the seam must not leak the token — 010 never reads `ctx.Cred.Token` (the replay thunk is its only path) and no error carries it; this is a cross-artifact invariant for analyze to confirm, enforced by design here.
- **III. Fail Safe, Not Silent** — "validate a write / no partial state" is N/A (010 issues the request but owns no multi-step write; reads don't mutate). The live concern — "no swallowed error, no failure reported as success" — is met: base-URL error refuses before sending, a wire failure is a `TransportError`, a non-2xx is short-circuited to `ResponseError` (never treated as success), an undecodable 2xx is a loud `DecodeError` (not a zero-valued target), exactly one attempt (no silent retry), and the response body is always closed.
- **VI. Size-Aware by Design** — calibrated to the single-request seam: 010 returns **one complete response** (it reads the full body for non-2xx into `ResponseError.Body`, decodes the full 2xx body into the caller's target) and **exposes `Response.Header`** so paging boundaries are detectable. It never silently truncates. Paging *across* pages is Pagination (016), which builds on the headers this seam surfaces.
- **X. Respect API Limits** — calibrated: the detection target ("a retry loop that ignores `429`/`Retry-After`") cannot trip because 010 makes **exactly one attempt and never retries**; it surfaces a `429` as a `ResponseError` carrying the rate-limit headers so Rate-Limit Handling (017) can honor them. `If-Match`/`ETag` optimistic concurrency is an *update*-command concern, not this read-transport seam's. (Cross-spec sequencing of 017 — see Governance Notes.)

### Passed (11/11)

- **P0** | CONSTITUTION I (Spec Fidelity): invents no endpoints, parameters, or behaviors — **pass**. spec.md System Overview + Non-Behaviors: 010 is a **method-agnostic transport** that sends whatever `Request{Method, Path, …}` a caller supplies and joins it onto 008's spec-traced base URL; it defines no endpoint of its own. Per-operation spec fidelity is each consuming command's check (011+).
- **P0** | CONSTITUTION II (Action Transparency): outcomes are structured and machine-parseable — **pass**. interface-spec.md Surface (`Response{StatusCode, Header}`) + Error outcomes; the consuming command adds the operation+resource line (calibration above).
- **P0** | CONSTITUTION II (Action Transparency): failures name the cause — **pass**. interface-spec.md § Error Communication: `TransportError` wraps the network cause, `ResponseError` carries status+headers+body, `DecodeError` wraps the parse cause, `AuthError` is propagated; none carries the token.
- **P0** | CONSTITUTION III (Fail Safe, Not Silent): every fork fails loud, nothing swallowed or mis-reported as success — **pass**. spec Behavioral Accord + plan ADR-3/ADR-4 + Cross-cutting: base-URL refusal, typed transport error, non-2xx short-circuit, decode error, one-attempt-no-retry, body always closed.
- **P0** | CONSTITUTION IV (TDD): test-first + executable acceptance — **pass**. tasks T001–T003 each specify RED-first unit tests; features/no-shared-api-client/request-execution.feature exists with executable acceptance scenarios (Phase 3 / T003); 4 `@validation` scenarios held out.
- **P0** | CONSTITUTION V (Composition over Monolith): modular shared client, no unrelated edits — **pass**. plan System Architecture + tasks: 010 **is** the "shared API client" V names; purely additive in `internal/apiclient`, changes no existing file, 007's `AuthTransport` untouched (replay-thunk wiring only). Commands (011–017) compose over `(*Client).Execute` without entanglement; `apiclient` does not import `internal/cli`.
- **P0** | CONSTITUTION VI (Size-Aware by Design): no silent truncation — **pass** (calibrated). interface-spec.md Output contract + plan ADR-3: one complete response returned, `Response.Header` exposed for 016; full non-2xx body carried raw.
- **P0** | CONSTITUTION VII (Working Software): impl + tests + build per increment — **pass**. tasks bundle implementation with RED-first tests; acceptance criteria require `go build ./...` and `go vet ./...` clean.
- **P0** | CONSTITUTION VIII (No Fabricated Data): presents only what the API returned — **pass**. plan ADR-3 + spec: the 2xx body is decoded into the caller's target; a non-decodable body is a `DecodeError` rather than a fabricated zero-valued target; non-2xx carries the raw body; no defaulting or synthesis. Strongly aligned (the decode-error-not-zero-value and no-forced-decode design directly serves VIII).
- **P0** | CONSTITUTION IX (Writes Require Explicit Intent): no mutation as a side effect — **pass** (calibrated). 010 is a method-agnostic transport with **no command path**; it issues no write of its own and has no read/get/list path that emits POST/PATCH/DELETE. The read-never-mutates gating is expressed by the consuming command passing the method; 010 introduces no implicit mutation.
- **P0** | CONSTITUTION XII (Standalone Executable): no new dependency — **pass**. plan/tasks/interface use only the standard library (`net/http`, `context`, `encoding/json`, `errors`, `io`) and existing internal packages (`internal/apiclient`, `internal/auth`); no third-party dependency is introduced.

---

## Governance Notes

*(Separate from feature quality findings.)*

- **No `done-*` accords found** (`accords/governance/` does not exist). Done-criteria and cross-reference checks were not generated. Consider creating `accords/governance/done-{specify,plan,interface,scenarios,tasks}.md` to enable layered done-criteria checks across the pipeline. (Same project-level gap recorded for 008/009 — infrastructure, not a 010 finding.)
- **XI. Governance via Proposals** produced no applicable check — 010 is a generic transport seam: no command surface, and it mutates no governance (it sends whatever a future command supplies; the proposal-gating lives on the governance-write commands, not the transport).
- **Cross-spec sequencing — X (Respect API Limits) and the 429 backoff** *(observation, not a 010 defect; carried to analyze/risk)*: 010 deliberately defers `429` backoff to Rate-Limit Handling (017) and only **surfaces** the 429 + rate-limit headers. In the BACKLOG, 017 is Should-tier and ranks **after** the Must-tier reads (011–014). So a window exists where reads built on 010 would run **without** honoring `Retry-After` until 017 lands. 010's artifacts are correct for 010's scope (it enables backoff by carrying the headers and never silently retries); the sequencing is a roadmap concern the Guardian should weigh — consider pulling 017 forward, or accepting the one-attempt-no-backoff behavior for the initial read slice.
- **Secret-hygiene** (token never emitted) is enforced by design (010 never reads `ctx.Cred.Token`; the replay thunk is the only path; no error carries the token) rather than a single CONSTITUTION principle; recorded here as a cross-artifact invariant for analyze to confirm.
