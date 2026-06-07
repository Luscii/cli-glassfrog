# Tasks: API Error Extraction

**Feature**: 015-api-error-extraction
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, features/opaque-failures/api-error-extraction.feature

---

## Dependency Graph

Phase 1: `ProblemError` + `ExtractProblem` in `internal/apiclient` (1 task, no phase dependencies) [Shared]
Phase 2: Consumer wiring in `internal/cli` — `PermissionError` + the 401/403 split + detail-surfacing (1 task, depends on Phase 1) [Shared]
Phase 3: Executable acceptance via godog (1 task, depends on Phase 2) [Shared]

3 tasks total | 0 phases parallelizable (linear chain) | Builder: pipeline

> Every task is `[Shared]`: API Error Extraction is one capability serving all three user scenarios (know-why-it-failed / decide-what-to-do-next / usable-error-on-junk) rather than any single one.
>
> **Cross-spec note**: Phase 1 is purely additive in `internal/apiclient` (a new file; no existing file changed) and consumes landed code — 010's `ResponseError{StatusCode, Header, Body}` on main. Phase 2 grows the shared `internal/cli` read-surface helpers 011 introduced (`classifyClientError`, `formatClientErrorMessage`, `reportClientError`, the `Outcome` enum, `ExitCode`) **additively** — the only behavioural change to landed reads (011–014) is that a 401/403 now exits 4 (the code 004 reserved) instead of 3, and every non-2xx message now carries the API's detail. `apiclient` still never imports `internal/cli`; the capability decides no exit code (the consumer maps). No new `.glassfrogrc` key, env var, or flag is introduced.

---

## Branching Guidance

**Pipeline mode**: `spec/015-api-error-extraction/base` → `spec/015-api-error-extraction/task-1`, `…/task-2`, `…/task-3` (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: none active — specs 001–011 are Complete; 012–014 are Analyzed (pre-implementation). 015 depends only on 010 (landed on main) and 011's shared `internal/cli` helpers (landed). Pagination (016) and Rate-Limit Handling (017) are later specs that build on this capability, not concurrent ones.

---

## Phase 1: `ProblemError` + `ExtractProblem` in `internal/apiclient` [Shared]

- [ ] **T001** [Shared] Add the typed `ProblemError` (wrapping 010's `ResponseError`) and the pure, total `ExtractProblem` that parses RFC 9457 Problem Details, degrades gracefully, and keeps the HTTP status authoritative — RED-first unit tests
  - **Scope**: In `internal/apiclient` (a new file, e.g. `problem.go`), define `ProblemError` carrying the authoritative `StatusCode int` (from the wrapped `*ResponseError`), `Type`/`Title`/`Detail string`, a **`DetailSynthesized bool`** provenance marker, a **`BodyStatus *int`** (the body's own `status` member when present; nil otherwise — metadata only), and the wrapped `*ResponseError` (with `Error() string` naming the status + detail, and `Unwrap() error` returning the `*ResponseError` so `errors.As(err, &ResponseError)` still matches and `Body`/`Header` stay reachable). Add `ExtractProblem(re *ResponseError) *ProblemError`: best-effort-decode `re.Body` as an RFC 9457 Problem Details object (`type`/`title`/`status`/`detail`), **not gated on `Content-Type`**; set `StatusCode` from `re.StatusCode` (authoritative — a disagreeing in-body `status` member is never promoted, but is captured in `BodyStatus`); on a parseable body, populate `Type`/`Title`/`Detail` and set `DetailSynthesized = false` (and `BodyStatus` if the body carried `status`); on an empty / non-JSON / HTML / member-missing body, set a fallback `Detail` derived from the status (e.g. `http.StatusText(re.StatusCode)`) **with `DetailSynthesized = true`** and leave the raw body on the wrapped value. The function is **total** — it never returns an error and never panics. It reads only the response-side `ResponseError`; it never reads the request token.
  - **Acceptance criteria**:
    - A valid Problem Details body yields a `ProblemError` carrying `StatusCode` plus the extracted `Type`/`Title`/`Detail`, with `DetailSynthesized == false`
    - A body with RFC 9457 extension members surfaces only `Type`/`Title`/`Detail` as fields; the raw body remains reachable via the wrapped `*ResponseError`
    - An empty, non-JSON, HTML, or member-missing body yields a `ProblemError` with a status-derived fallback `Detail`, **`DetailSynthesized == true`**, and the raw body preserved — `ExtractProblem` never returns nil and never panics
    - Parsing is **not** conditioned on the response `Content-Type`
    - When the body's `status` member disagrees with `re.StatusCode`, the `ProblemError.StatusCode` equals `re.StatusCode` (HTTP authoritative); the body's `status` never overrides but **is captured in `BodyStatus`** (so the disagreement is observable as metadata)
    - `errors.As` discriminates `*ProblemError`, and `errors.As(err, &ResponseError)` still matches the wrapped value; no `ProblemError` output renders a token-shaped value
    - RED-first table-driven unit tests over crafted `ResponseError` values (valid / extensions / empty / HTML / missing-members / status-mismatch); `go build ./...` and `go vet ./...` clean
  - **Dependencies**: None (builds on 010's `ResponseError`, on main)
  - **Plan reference**: Phase 1; ADR-1 (`ProblemError` wraps `ResponseError`, pure `ExtractProblem`), ADR-2 (graceful degradation + HTTP-status-authoritative); Cross-cutting (secret hygiene — response-side only; total/fail-soft parse)
  - **Interface references**: interface-spec.md — Entry points (`ExtractProblem`), Output contract `ProblemError` (Surface), Interactions (best-effort parse, status authority, builds on 010's seam)
  - **Scenario references**: api-error-extraction.feature: "A valid Problem Details body is extracted", "A 404 surfaces the API's own detail", "Extension members are preserved without being promoted", "The HTTP status wins over a disagreeing body status", "An empty body degrades to the HTTP status", "A non-JSON gateway body degrades gracefully", "Every non-2xx yields a typed error"
  - **Risk**: ⚠️ Keep `ExtractProblem` **total** — a junk/empty body must degrade, never panic or return nil (CONSTITUTION III; the empty/HTML/missing-members rows pin this). ⚠️ Do not promote the body's `status` member over the HTTP status (fork 2). ⚠️ Do not gate parsing on `Content-Type` (spec Non-Behavior). ⚠️ Response-side only — never read `ctx.Cred.Token` / the request header.

## Phase 2: Consumer wiring in `internal/cli` [Shared]

- [ ] **T002** [Shared] Add `PermissionError`, split 401/403 → permission(4) in the shared classifier, refine once at the chokepoint, and surface the API's detail in the operator message — RED-first unit tests
  - **Scope**: Grow 011's shared `internal/cli` read-surface helpers (no parallel path): (1) add `PermissionError` to the `Outcome` enum (`dispatch.go`) with its `String()` arm; (2) add `case PermissionError: return codePermissionError` to `ExitCode` (`exitcode.go`) — the constant `codePermissionError = 4` already exists; keep the pure-mapper shape and the `default → 1` fail-safe; renumber nothing; (3) in `classifyClientError` (`clienterror.go`), branch the API error on status — **401 or 403 → `PermissionError`**, everything else (incl. **429**) → `APIError` — checked on the `*ProblemError`/`*ResponseError` status **before** the bare generic `*ResponseError` arm (the wrapping in T001 means an unordered `*ResponseError` arm would otherwise swallow it); (4) in `reportClientError` (`me.go`), refine a `*ResponseError` into a `*ProblemError` via `apiclient.ExtractProblem` **once** (guard against double-refinement when the error is already a `*ProblemError`) so the typed error travels up the chain and message+category are computed from the same value; (5) in `formatClientErrorMessage` (`me.go`), add a `*ProblemError` arm that renders the API's `Detail` (and `Title` where useful) **when `DetailSynthesized == false`**, falling back to the existing `"the API returned a non-2xx response: status N"` wording **when `DetailSynthesized == true`** (key on the provenance flag, **not** on `Detail` emptiness — the fallback always fills `Detail`), and **append a next-step hint** so the arm matches its siblings and CONSTITUTION II ("…and the next step") — at minimum for the `PermissionError` class (401/403): "check the token's access / membership". Token-free throughout (response-side fields only).
  - **Acceptance criteria**:
    - `classifyClientError` maps a 401 and a 403 API error to `PermissionError`; a 429 and a 500 still map to `APIError`; the classifier table test grows with these rows and keeps its `len`+comma-ok exhaustiveness guard (PR #10)
    - `ExitCode(PermissionError)` returns `codePermissionError` (4); `exitcode_test.go` asserts the now-live mapping; no existing code is renumbered
    - A 401/403 is classified `PermissionError` even though `*ProblemError` unwraps to `*ResponseError` (the status branch / `*ProblemError` check precedes the generic `*ResponseError` arm) — pinned by a test
    - `reportClientError` returns a `*ProblemError` for a non-2xx (refined once, no double-refinement); `formatClientErrorMessage` renders the API's `Detail` when `DetailSynthesized == false` and the "status N" fallback when `true`, and appends a next-step hint (≥ for 401/403); a test asserts a synthesized-detail case shows the fallback wording (not the synthesized text) and a permission case shows the next-step hint
    - The surfaced message is token-free; existing 011–014 read tests still pass (only 401/403 change exit code, 3 → 4)
    - RED-first unit tests for the classifier rows, the exit-code mapping, and the message (detail-surfaced + fallback); `go build ./...` and `go vet ./...` clean
  - **Dependencies**: T001
  - **Plan reference**: Phase 2; ADR-3 (consumer-side split, `apiclient` decides no exit code), ADR-4 (refine once at `reportClientError`, surface detail with fallback); Cross-cutting (error handling, secret hygiene)
  - **Interface references**: interface-spec.md — Consumer contract changes, Status → Outcome → exit-code mapping (Surface), Error Communication
  - **Scenario references**: api-error-extraction.feature: "The API detail appears in the failure message", "An authorization failure exits with the permission code", "A 403 is carried generically", "A 429 is extracted without backoff"
  - **Risk**: ⚠️ Classifier discrimination order — branch on status / check `*ProblemError` **before** the generic `*ResponseError` arm, or every 401/403 falls through to `APIError`(3) and the split silently never fires (test a 401 → `PermissionError`). ⚠️ 429 stays `APIError`(3) here — do **not** map it to rate-limit(5) (that is 017's split); pin 429 → `APIError`. ⚠️ Keep the change additive — only 401/403 change category; confirm 011–014's existing tests still pass.

## Phase 3: Executable acceptance [Shared]

- [ ] **T003** [Shared] Make the 015 driving scenarios pass as executable acceptance via godog, driving the extraction + consumer wiring over a fake transport in a suite scoped to this spec's own feature file
  - **Scope**: Add godog step definitions for `features/opaque-failures/api-error-extraction.feature` (all three Rule blocks), in a **new** `internal/cli` godog suite whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory — the package already has the `me`/identity-read suite pattern). Drive the extraction-level scenarios directly through `apiclient.ExtractProblem` over crafted `ResponseError` values, and the consumer-observable scenarios (detail-in-message, 401/403 → exit 4) through the read-command path over a **fake** base `http.RoundTripper` returning canned non-2xx responses (the `meSeam`/`me_test.go` fake-transport pattern). Step helpers return errors, never panic (LEARNINGS). **Reuse existing step phrasing** where an assertion already exists — grep the package's `sc.Step(` registrations before adding bindings. Remove `@wip` from the behavioral scenarios; keep the `@validation` scenarios `@wip` (held out for validate).
  - **Acceptance criteria**:
    - Every non-`@validation` 015 scenario (valid-extraction / 404-detail / 429-no-backoff / 403-generic / extension-preserved / status-authority / detail-in-message / authorization-exits-4 / empty-body-degrades / non-JSON-degrades) has an executable, passing path
    - `@wip` removed from those scenarios; the four `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `api-error-extraction.feature`; every `internal/cli` suite runs and reports its own independent `N scenarios (N passed)` counts
    - No real network (fake base / crafted `ResponseError` only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T002
  - **Plan reference**: Phase 3 — Executable acceptance; Cross-cutting Concerns (testing strategy)
  - **Scenario references**: api-error-extraction.feature: all 015 behavioral Rule-block scenarios
  - **Risk**: ⚠️ Suite scoping — keep every `internal/cli` suite pointed at its specific feature file (not the directory), or un-wipping one spec's scenarios breaks another suite; verify each reports its own counts. ⚠️ Step-vocabulary — grep existing `sc.Step(` registrations and match phrasing before writing new bindings (LEARNINGS); step helpers return errors, never panic.
