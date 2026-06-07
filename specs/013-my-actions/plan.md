# Plan: My Actions

**Feature**: 013-my-actions
**Role**: Shaper
**Inputs**: spec.md (013-my-actions); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: API response models live in a new `internal/glassfrog` schema package, read commands decode into shared schema types — 011 ADR-1; the read surface classifies 010's typed client errors through one `internal/cli` `classifyClientError` into the `Outcome` enum, new categories added once at the registry — 011 ADR-3; local/precondition failures reuse `UsageError`/`RuntimeError`, reserved codes 3–6 are API-side/network outcomes, 015/017 refine `APIError` without renumbering — 011 ADR-4; global connection options are persistent root flags read by inheritance, `--base-url` already registered — 011 ADR-2; each read pairs a pure `run<Read>` + pure `format<Read>` behind an injected seam, projection is the agent-legible default with `--output json` deferred — 011 ADR-5; `internal/apiclient` `NewClientFromOS(ctx)` + `(*Client).Execute(reqCtx, Request, out)` send seam returning `*Response` or typed code-free errors — 010; `internal/rcfile` owns the `.glassfrogrc` read/parse/walk; producer-classifies-a-code-free-outcome / consumer-maps split — 002/004/005/007/008/009/010); `.score/memory/LEARNINGS.md` (inject seams / fail-fast on nil — PR #20; a godog suite points at its OWN feature file; step helpers return errors, never panic; temp-file stderr capture not `os.Pipe`; table-driven change-detector guards `len()`+comma-ok — PR #10); `.score/memory/DEPRECATION.md` (no entry touches the read surface). No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord (Listing / Filtering / Pagination boundary / Empty result / Error handling), three happy-path + two error + two edge scenarios, four validation scenarios, nine non-behaviors, integration boundaries naming every dependency (010/007/011/012/016/004/API), user scenarios, assumptions. Strong foundation. The four behavioral forks were resolved during `/score:define` and the `--status` fork was aligned to 011's `validateInclude` precedent during verification against the shaped 011 artifacts. **No architectural unknowns required a resolve exchange**: 011 already resolved the schema-package home, the classifier, the reserved-code mapping, and the persistent `--base-url` flag; My Roles (012) — the first paginated `/me*` list, in progress in a parallel session — owns the `me` parent command, the `glassfrog.Pagination` type, the list envelope, and the "more results available" signal convention this command reuses. The remaining `[ASSUMED]` items (exact `Action` field set, projection layout, flag spelling, the 012-owned pagination/signal shapes) are interface-level, not behavioral gaps.

---

## System Architecture

My Actions is **a thin additive read on top of two foundations**: Identity Read (011) — which established the `internal/glassfrog` schema package, the shared `classifyClientError`, the operational `Outcome`/`ExitCode` categories, and the persistent `--base-url` root flag — and My Roles (012), the first paginated `/me*` list, which establishes the `me` parent command, the pagination types, the list envelope, and the "more results available" signal. 013 introduces **no new infrastructure**: it adds one resource model (`glassfrog.Action`), one command leaf (`me actions`), its pure orchestration/projection/validation trio, and a shared status-filter validator. It reuses everything else by construction — the same discipline ("one reader, one type, one registry") the project has held since 005.

The command lists the actions owned by roles the authenticated practitioner fills (`GET /me/actions`), fetching **one page** through 010's `Execute` seam, rendering the result through the shared `/me*` list projection with a "more available" signal when the API reports a next page, and validating an optional `--status` filter against the spec's status set **before any request** (the 011 `validateInclude` fail-fast pattern). It interprets nothing about non-2xx responses (API Error Extraction, 015), walks no pages (Pagination, 016), and backs off on no `429` (Rate-Limit Handling, 017).

The parts:

- **`internal/glassfrog` (extended)** — add `Action` (`ID`, `Description`, `Status`, `RoleID`, `Tags`, … decoded; the projected subset pinned at interface). Reuse `glassfrog.Pagination` and the list envelope My Roles (012) introduces as the first paginated read; 013 does not define a second pagination type. Schema only — no transport, no cobra, no exit codes (011 ADR-1).
- **`internal/cli/my_actions.go` (NEW)** — `newMyActionsCommand(seam)`: a guard-registered cobra leaf (`Use:"actions"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) attached to the **`me` parent command** (012). Carries the local `--status` flag; reads the persistent `--base-url` value (011 ADR-2); delegates to a pure `runMyActions(cfg)`.
- **`runMyActions` + `formatMyActions` + `validateStatus` (pure, in `my_actions.go`)** — `validateStatus` rejects an unsupported `--status` value before any request (usage error, fail-fast); `runMyActions` assembles the context, builds the client, calls `Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me/actions", Query:{status?}}, &list)` with `&list` a `glassfrog`-list-of-`Action` decode target, and on success calls `formatMyActions(list)` to render the shared list projection plus the "more available" signal; on a typed error it calls the shared `classifyClientError` and writes a token-free message. `formatMyActions` is a pure renderer (testable in isolation).
- **Shared `validateStatus` + status set (NEW, reused by 014)** — the `{archived, cancelled, completed, current, scheduled, someday, waiting}` set sourced from the spec enum, and the pure validator that rejects anything outside it. Introduced here (the first status-filtered read) and reused by My Projects (014); the same "introduce-once, reuse" shape 011 used for `Role`/`classifyClientError`.

```
glassfrog me actions [--status <s>]        (cobra leaf under `me` (012), internal/cli/my_actions.go)
  │  read persistent --base-url value (root flag, 011 ADR-2)
  │  validateStatus(flag) ─ unsupported value → UsageError, no request (fail-fast)   [ADR-2]
  └─ runMyActions(cfg):
       ctx    := AssembleFromOS(baseURL)                 ── 009, resolve-once
       client := NewClientFromOS(ctx)                    ── 010; base-URL fail-fast lives here
       resp, err := client.Execute(reqCtx,
                      apiclient.Request{Method:"GET", Path:"/me/actions", Query:{status?}},
                      &list)                              ── Request is 010's; &list : glassfrog list-of-Action (decode target)
         ├─ success    → formatMyActions(list) → stdout (projection + "more available" when HasNextPage) ; Success(0)
         ├─ *AuthError{NoCredentials}   → UsageError(2)
         ├─ *AuthError{CredentialError} → RuntimeError(1)
         ├─ *TransportError             → NetworkUnavailable(6)
         ├─ *ResponseError (non-2xx)    → APIError(3)              (generic; 015 refines 401/403→4, 429→5)
         └─ *DecodeError                → RuntimeError(1)
                                          │
              classifyClientError(err) ──┘  (internal/cli, 011 ADR-3 — REUSED, not re-added)
              ExitCode(Outcome) ───────────► process code   (004 registry, codes added by 011)

internal/glassfrog (extended)       + Action{…}        ; reuses Pagination + list envelope (012)
internal/apiclient (010, consumed)  AssembleFromOS, NewClientFromOS, (*Client).Execute, typed errors  ── unchanged
internal/cli (011, consumed)        classifyClientError, Outcome, ExitCode, persistent --base-url      ── REUSED, unchanged
```

The token takes exactly one path — 010's replay thunk into 007's `AuthTransport`. `me actions` **never reads `ctx.Cred.Token`**; its projection renders only response-side fields, so secret hygiene holds structurally.

---

## Architecture Decisions

### ADR-1: `Action` joins `internal/glassfrog`; pagination types and the list envelope are reused from My Roles (012), not redefined

**Context**: `me actions` decodes the `GET /me/actions` body — a paginated envelope `{data: [Action], meta: {pagination}}` (`spec/glassfrog-api-v5.yaml` `Action` + `Pagination`). 011 set the precedent that API response models live in `internal/glassfrog` (ADR-1). My Roles (012) is the first paginated `/me*` read, so it introduces the `Pagination` struct and the list envelope shape; 013/014 are the second and third paginated reads.

**Options considered**:
1. **Add `Action` to `internal/glassfrog`; reuse 012's `Pagination` + list envelope** — one pagination type and one envelope shape across every list read, mirroring how `Role` is shared. 013 adds only its resource model.
2. **Define a 013-local pagination/envelope type** — rejected: duplicates `Pagination` across 012/013/014, the exact drift the project avoids (one type, one place); three list reads with three envelope shapes is a maintenance trap.
3. **Put pagination in `internal/apiclient`** — rejected: 010 is transport-only and the envelope is response *schema*; it belongs in `glassfrog` (011 ADR-1).

**Decision**: Option 1. Add `Action` to `internal/glassfrog`; reuse `glassfrog.Pagination` and the list envelope My Roles (012) introduces. If 013 is built before 012 lands, it introduces the shared `Pagination`/envelope (still in `glassfrog`, still shared) and 012 consumes it — whichever paginated read lands first creates the shared types, the others reuse (the 005/006/007 "first-to-land creates the shared package" pattern). Decoding is tolerant of unknown/extra fields. The projected `Action` subset and the envelope field/tag shapes are interface-level.

**Consequences**: One pagination type, one envelope, across all list reads. 014 (My Projects) reuses both plus `Action` is its sibling resource model (`Project`). *Precedent-setting: paginated list reads share one `glassfrog.Pagination` + one list-envelope shape; each adds only its resource model.*

### ADR-2: `--status` is validated against the spec's status set before any request, via a shared pure `validateStatus` (mirrors 011 `validateInclude`)

**Context**: The spec (fork 2, aligned during verification) fixes that an unsupported `--status` is a usage error caught locally before any request — naming the unsupported value and the supported set — rather than passed through to the API. This mirrors 011's `validateInclude`, which rejects an unsupported `--include` target before assembly. The status enum (`archived, cancelled, completed, current, scheduled, someday, waiting`) is identical for My Actions and My Projects (014), and `GET /me/actions` does not even document a `400` for a bad status, so pass-through could silently return wrong results.

**Options considered**:
1. **A shared pure `validateStatus([]string) error` over a named status set sourced from the spec enum, run before context assembly** — same shape and fail-fast timing as `validateInclude`; introduced here, reused by 014. An unsupported value → `UsageError`(2), no request (a transport tripwire asserts nothing was sent).
2. **Pass `--status` through to `?status=` and let the API reject it** — rejected during verification: returns an opaque `APIError`(3) indistinguishable from a real failure, and is undocumented for `/me/actions`; local validation gives the agent operator a usage error and the supported set to self-correct.
3. **Validate inside `runMyActions` after assembly** — rejected: wastes a connection assembly on a value already known to be invalid; 011's precedent validates before any I/O.

**Decision**: Option 1. A pure `validateStatus` checks the flag value against the spec's status set and returns a usage error naming the unsupported value and the supported set, before assembly or request. Introduced in 013, reused verbatim by 014. Whether it lives in `my_actions.go` or a small shared `internal/cli` helper is an interface detail; the set tracks the spec enum (vendored at `spec/glassfrog-api-v5.yaml`).

**Consequences**: Consistent with 011's flag-validation precedent; the agent gets an actionable usage error. The set is a one-line change if the spec adds a status. *Precedent-setting: spec-enumerated filter flags on the read surface are validated locally against the spec set before any request (the `validateInclude` shape), reusing one validator across siblings.*

### ADR-3: First page only, with a "more results available" signal following My Roles (012)'s convention; no paging walk

**Context**: The spec fixes (fork 1) that the command fetches one page and surfaces a "more available" signal, deferring multi-page traversal to Pagination (016). 010's `Execute` returns exactly one response; the envelope's `meta.pagination` carries `has_next_page` and `next_cursor`. My Roles (012), the first paginated read, establishes how the signal is rendered.

**Options considered**:
1. **One `Execute` call; `formatMyActions` renders the page and appends the "more available" signal (per 012's convention) when `pagination.has_next_page` is true** — narrow, defers the walk to 016, follows the established signal shape.
2. **Walk all pages now** — rejected: duplicates Pagination (016)'s contract across every list read and hides the boundary this command is meant to signal (spec Non-Behavior).
3. **Render the page silently with no signal** — rejected: a truncated list indistinguishable from a complete one is exactly the "silent truncation" the signal exists to prevent (spec user scenario 3).

**Decision**: Option 1. `runMyActions` makes one `Execute` call; `formatMyActions` renders the page and, when `has_next_page` is true, appends the "more available" signal in the form My Roles (012) establishes (reused, not reinvented). No `--cursor`/`--per-page` flags are added — those arrive with Pagination (016).

**Consequences**: A narrow, honest read; 016 later owns the walk for every list read uniformly. *Reuses 012's signal convention; if 012 lands a different signal shape, 013 tracks it (Risk).* 

### ADR-4: A guard-registered `actions` leaf under the `me` parent command, with an injected seam and a pure `runMyActions`/`formatMyActions`/`validateStatus` trio

**Context**: CONSTITUTION IV requires hermetic, RED-first tests with no real network. 011 ADR-5 established the pattern: a thin cobra `RunE` over an injected seam, delegating to a pure `run*` over injected values, with a pure `format*` renderer and pre-request validation. The dedicated `/me*` reads are grouped under a `me` parent command (011 refers to "the dedicated `my roles` read"); My Roles (012) introduces that parent.

**Options considered**:
1. **`newMyActionsCommand(seam)` attached to the `me` parent (012), + pure `runMyActions(cfg)` + pure `formatMyActions(list) string` + pure `validateStatus`** — the seam binds `AssembleFromOS`/`NewClientFromOS` (and an injected base `http.RoundTripper`) in production and fakes in tests, so every branch (success, more-available, empty, non-2xx, transport, no-token, decode-fail, base-URL-error, status-rejection) runs offline. Mirrors `me` exactly.
2. **A monolithic `RunE` calling the real OS seams** — rejected: not hermetically testable, violates CONSTITUTION IV and the 005/006/011 injected-seam precedent.
3. **A top-level `my-actions` command instead of a `me actions` leaf** — rejected: 011 implies the `my <noun>` grouping; a flat command forks the surface 012 establishes.

**Decision**: Option 1. The seam shape mirrors 011's `meSeam`. `newMyActionsCommand` registers `actions` under the `me` parent via the `MustRegister` guard (001). If the `me` parent does not yet exist (012 not landed), 013 creates it idempotently and 012 reuses it — first sibling to land creates the parent. `validateStatus` runs before assembly; `formatMyActions` renders the list projection + signal.

**Consequences**: Every outcome is exercised offline; `me actions` never touches the real network or `~/.glassfrogrc` in tests. The exact seam interface, `--status` flag spelling, decode type names, projection layout, and `me`-parent ownership are interface-level. *Mild precedent: each dedicated `/me*` read is a leaf under the `me` parent, pairing a pure `run<Read>`/`format<Read>`/validator behind an injected seam.*

---

## Data Model Design

`internal/glassfrog` (schema only; JSON-tagged structs; field names/tags pinned at `/score:interface`):

- **`Action`** — the `/me/actions` list item: `ID` (`actn_` — the machine-actionable handle), `Description` (nullable), `Status` (the status enum), `RoleID` (`role_` — owning role), `Tags []string`; `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, and optional `Permissions`/`TriggerEvent`/`Note` decoded. The projected subset (likely id, description, status, role, tags) is pinned at interface; decoding is tolerant of unknown/extra fields.
- **`Pagination`** (reused from 012) — `PerPage`, `HasNextPage`, `NextCursor` (present when `HasNextPage`). 013 does not redefine it.
- **List envelope** (reused from 012) — `{Data []Action, Meta{Pagination}}`. The concrete envelope (a generic `List[T]` or a per-resource struct) is 012's interface decision; 013 decodes `Action` through it.

The token is never a field here (it is a request header). Secret hygiene holds by construction.

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — upstream dependency, landed on main)**: `me actions` calls `NewClientFromOS(ctx)` then `(*Client).Execute(reqCtx, Request{Method:"GET", Path:"/me/actions", Query:{status?}}, &list)`. Consumes 010's `Client`/`Execute`/`Request`/`Response` and typed errors unchanged; adds nothing to `apiclient`.
- **Identity Read (011, `internal/cli` + `internal/glassfrog` — upstream dependency, shaped not yet built)**: `me actions` reuses the `internal/glassfrog` package, the shared `classifyClientError`, the `Outcome`/`ExitCode` categories (codes 3/6), and the persistent `--base-url` flag — all introduced by 011. 013 adds none of these; it depends on 011 being implemented first.
- **My Roles (012, `internal/cli` + `internal/glassfrog` — upstream dependency, in progress in a parallel session)**: `me actions` reuses the `me` parent command, the `glassfrog.Pagination` type, the list envelope, and the "more results available" signal convention 012 introduces as the first paginated read. 013 attaches its `actions` leaf to 012's parent.
- **Glassfrog API `GET /me/actions` (`listMyActions`, system actor)**: returns `{data: [Action], meta: {pagination}}` on `200`, with an optional `status` filter; `401`/`404` per the spec. `me actions` decodes the 200 envelope; non-200 arrives as 010's generic `*ResponseError`.
- **My Projects (014 — downstream sibling/twin)**: reuses `validateStatus` + the status set (013 introduces them), the list-projection convention, and the same seam/command shape over `GET /me/projects` and `glassfrog.Project`.
- **Pagination (016) / API Error Extraction (015) / Rate-Limit Handling (017) — downstream siblings**: 016 later owns walking past the first page for every list read; 015/017 refine `APIError`(3) by status without renumbering. 013 surfaces the boundary and the generic outcome they build on.
- **Exit-Code Convention (004, `internal/cli`)**: outcomes map through the single `ExitCode` registry; 013 adds no codes (011 added 3/6).

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: `me actions` **never reads `ctx.Cred.Token`** — the token's only path is 010's replay thunk into 007's `AuthTransport`. `formatMyActions` renders response-side fields only; `classifyClientError` messages name outcomes/paths, never the token. Pinned by a test asserting no output or error renders the token across success and every error branch, and that the command never references `ctx.Cred.Token`.

**Error handling (CONSTITUTION III)**: fail loud at every fork — an unsupported `--status` is refused before any request; a base-URL error refuses at `NewClientFromOS` (no doomed send); a wire failure is `NetworkUnavailable`(6); a non-2xx is never treated as success (generic `APIError`(3)); a 2xx body that won't decode is a loud `RuntimeError`(1), never a zero-valued projection. Every outcome maps through the single `ExitCode` registry; the `default→1` fail-safe backstops any unmapped category.

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network or `~/.glassfrogrc`. The injected seam binds a fake base `http.RoundTripper` returning canned `GET /me/actions` responses, so every branch runs offline: 200 single page → projection; 200 with `has_next_page` → projection + "more available" signal; 200 empty `data` → success with an empty list; 200 with `?status=current` → request carried the filter; non-2xx → `APIError`(3); transport error → `NetworkUnavailable`(6); no-token context → `UsageError`(2); broken-cred context → `RuntimeError`(1); undecodable 200 → `RuntimeError`(1); base-URL-error context → `UsageError`(2); unsupported `--status` → `UsageError`(2) with no request issued (a tripwire fake asserts the transport was never called). `formatMyActions` and `validateStatus` are unit-tested pure. The driving scenarios become a **new godog suite** pointed at its **own** feature file `features/self-service-reads/my-actions.feature` (LEARNINGS — never the whole `features/` dir); step helpers return errors, never panic; stderr/stdout capture uses a temp file, not `os.Pipe` (PR #10 LEARNINGS).

---

## Implementation Strategy

Three phases, mostly linear. **Upstream dependencies**: 010 is **landed on main**; Identity Read (011) and My Roles (012) are **shaped/in-progress but not yet on main**, and 013 reuses their foundations (the `glassfrog` package + `classifyClientError` + `Outcome`/`ExitCode` codes + `--base-url` from 011; the `me` parent + `Pagination` + list envelope + signal from 012). 013 should be cut from a main that carries 011 and 012; the tasks dependency graph keeps this explicit.

- **Phase 1 — `glassfrog.Action` (+ shared pagination/envelope if 012 has not landed)**: add `Action` with JSON tags and tolerant decode; reuse `glassfrog.Pagination` + the list envelope (or introduce them as shared types if 012 is not yet present). RED-first unit tests: a `GET /me/actions` fixture decodes into the envelope (single page, multi-page with `next_cursor`, empty `data`); extra/unknown fields are ignored. *Depends on: nothing built (leaf schema); coordinates with 012 on the shared types.*
- **Phase 2 — `validateStatus` + the status set**: the pure validator and the spec-sourced set. RED-first: accepts each supported status; rejects an unsupported value naming it and the set; empty/absent filter is valid. *Depends on: nothing (pure).* 
- **Phase 3 — `me actions` command: `newMyActionsCommand` + `runMyActions` + `formatMyActions` + seam + wiring**: the cobra leaf under the `me` parent, the injected seam (prod binds `AssembleFromOS`/`NewClientFromOS`), the pure `runMyActions` orchestration (validate status → assemble → build client → execute → render/classify), the pure `formatMyActions` projection + signal, and one `MustRegister` wiring line under `me`. RED-first unit tests over the fake-transport seam for every branch (Cross-cutting / Testing); godog step definitions for the driving scenarios in the new suite. *Depends on: Phase 1 (decode target), Phase 2 (validator), 010 implemented, and 011 + 012 implemented (the reused `classifyClientError`, `--base-url`, `me` parent, pagination/signal).*

---

## Risks

- **013 built before 011 and/or 012 land** (medium likelihood, high impact): Phase 3 reuses 011's `classifyClientError`/`Outcome`/`ExitCode`/`--base-url` and 012's `me` parent/`Pagination`/envelope/signal; building before they exist forces 013 to either stub or duplicate them. *Mitigation*: cut 013 from a main carrying 011 and 012; the tasks graph marks the dependency. If 013 must lead, it creates the shared types in `glassfrog`/the `me` parent idempotently and the later sibling reuses them (first-to-land-creates, 005/006/007 pattern) — but the preferred order is 010 → 011 → 012 → 013.
- **012's pagination/signal shape differs from the assumed convention** (medium likelihood, medium impact): 013 reuses 012's `Pagination`, envelope, and "more available" signal, designed here against an assumed shape since 012 is not yet on main. *Mitigation*: the spec marks the signal shape `[ASSUMED]` and "tracks 012"; 013's `formatMyActions` isolates the signal rendering, so adapting to 012's final shape is a localized change. Confirmed acceptable with the developer during verification.
- **Token leak through the projection or an error** (low likelihood, high impact): `me actions` sits on the read path. *Mitigation*: never reads `ctx.Cred.Token`; `formatMyActions` renders response-side fields only; classifier messages are path/outcome-only; pinned by a token-never-in-output test across success + every error branch (CONSTITUTION II).
- **Generic non-2xx → code 3 coarse until 015/017** (medium likelihood, low impact): every non-2xx maps to `APIError`(3). *Mitigation*: code 3 is the residual "general API error" bucket by design (004/011 ADR-4); 015/017 split it without renumbering. Forward-compatible.
- **`--status` set drifts from the spec enum** (low likelihood, low impact): hard-coding the seven statuses could lag a spec addition. *Mitigation*: the set is sourced from the spec's `status` enum; adding a value is a one-line change tracking the spec (an `[ASSUMED]`, planning-adjustable detail).

---

## What This Plan Does Not Cover

- **Multi-page traversal** — Pagination (016) owns walking past the first page for every list read; 013 fetches one page and signals more exist.
- **`--output json` (structured output)** — a deferred, cross-cutting capability (a future persistent root flag per 011); `me actions` ships the reshaped projection only (a Non-Behavior).
- **Per-status API error interpretation** (401/403→permission, 429→rate-limit, error-detail extraction) — API Error Extraction (015) and Rate-Limit Handling (017), which split `APIError`(3) without renumbering.
- **The `internal/glassfrog` package foundation, `classifyClientError`, the `Outcome`/`ExitCode` categories, the persistent `--base-url` flag** — introduced by Identity Read (011); 013 reuses them unchanged.
- **The `me` parent command, the `Pagination` type, the list envelope, the "more available" signal convention** — introduced by My Roles (012) as the first paginated read; 013 reuses them.
- **The exact Go API shapes** — the seam interface, `--status` flag spelling, `Action`/envelope struct fields and JSON tags, and the projection/signal layout — `/score:interface` pins these (the CLI + specification boundaries).
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/self-service-reads/my-actions.feature`.
- **010's `apiclient` implementation** — a separate spec (010), implemented and landed on main; 013 consumes it unchanged.
