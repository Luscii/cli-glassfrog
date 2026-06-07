# Plan: My Projects

**Feature**: 014-my-projects
**Role**: Shaper
**Inputs**: spec.md (014-my-projects); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: API models live in `internal/glassfrog`, read commands decode into shared schema types — 011 ADR-1; the read surface classifies 010's typed errors through one `classifyClientError` into the `Outcome` enum — 011 ADR-3; local/precondition failures reuse `UsageError`/`RuntimeError`, reserved codes 3–6 are API-side/network outcomes, 015/017 refine `APIError` without renumbering — 011 ADR-4; global connection options are persistent root flags read by inheritance — 011 ADR-2; pure `run<Read>`+`format<Read>` behind an injected seam, projection default with `--output json` deferred — 011 ADR-5; paginated list reads share one `glassfrog.Pagination` + one list-envelope shape, each adds only its resource model — 013 ADR-1; spec-enumerated filter flags validated locally before any request, reusing one validator across siblings — 013 ADR-2; 010's `NewClientFromOS`/`Execute` send seam; producer-classifies/consumer-maps — 002/004/005/007/008/009/010); `.score/memory/LEARNINGS.md` (inject seams / fail-fast on nil — PR #20; a godog suite points at its OWN feature file; step helpers return errors; temp-file stderr capture; table-driven detector guards `len()`+comma-ok — PR #10); `.score/memory/DEPRECATION.md` (no entry touches the read surface). No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord (Listing / Filtering / Pagination boundary / Empty result / Error handling), three happy-path + two error + two edge scenarios, four validation scenarios, ten non-behaviors, integration boundaries naming every dependency (010/007/011/012/013/016/004/API), user scenarios, assumptions. Strong foundation. The four behavioral forks were resolved during `/score:define`, with `--status` aligned to 011's `validateInclude` precedent during verification. **No architectural unknowns required a resolve exchange**: this spec is the structural twin of My Actions (013) and reuses everything 013 and its upstreams (011/012) establish; the only project-specific deltas are the `Project` resource model and the explicit non-behavior that `/me/projects` offers no `?include=` embedding. The remaining `[ASSUMED]` items (exact `Project` field set, projection layout, flag spelling, the 012-owned pagination/signal shapes) are interface-level, not behavioral gaps.

---

## System Architecture

My Projects is **the structural twin of My Actions (013)** over a different resource. It lists the projects owned by roles the authenticated practitioner fills (`GET /me/projects`), fetching one page through 010's `Execute` seam, rendering the result through the shared `/me*` list projection with a "more available" signal, and validating an optional `--status` filter against the spec's status set **before any request** — using the **same `validateStatus` 013 introduces** (the status enum is identical for both endpoints). It interprets nothing about non-2xx responses (015), walks no pages (016), and backs off on no `429` (017).

It reuses three foundations and adds almost nothing new: from Identity Read (011) the `internal/glassfrog` package, `classifyClientError`, the `Outcome`/`ExitCode` categories (codes 3/6), and the persistent `--base-url` flag; from My Roles (012) the `my` parent command, the `glassfrog.Pagination` type, the list envelope, and the "more available" signal; from My Actions (013) the shared `validateStatus` + status set and the list-projection/seam shape. 014 adds one resource model (`glassfrog.Project`), one command leaf (`my projects`), and its pure orchestration/projection.

The parts:

- **`internal/glassfrog` (extended)** — add `Project` (`ID`, `Description`, `Status`, `RoleID` (nullable — null for non-role-owned projects), `Tags`, `HasSubProjects`, `HasActions`, … decoded; projected subset pinned at interface). Reuse `glassfrog.Pagination` and the list envelope (012/013). Schema only.
- **`internal/cli/my_projects.go` (NEW)** — `newMyProjectsCommand(seam)`: a guard-registered cobra leaf (`Use:"projects"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) attached to the **`my` parent (012)**. Carries the local `--status` flag; reads the persistent `--base-url` value; delegates to a pure `runMyProjects(cfg)`.
- **`runMyProjects` + `formatMyProjects` (pure, in `my_projects.go`)** — `runMyProjects` calls the shared `validateStatus` (013), assembles the context, builds the client, calls `Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me/projects", Query:{status?}}, &list)`, and on success calls `formatMyProjects(list)` to render the projection + signal; on a typed error it calls the shared `classifyClientError`. `formatMyProjects` is a pure renderer (testable in isolation).
- **Reused, not added** — `validateStatus` + the status set (013), `classifyClientError` + `Outcome`/`ExitCode` + `--base-url` (011), the `my` parent + `Pagination` + envelope + signal (012).

```
glassfrog my projects [--status <s>]       (cobra leaf under `my` (012), internal/cli/my_projects.go)
  │  read persistent --base-url value (root flag, 011 ADR-2)
  │  validateStatus(flag) ─ unsupported value → UsageError, no request (REUSED from 013)   [ADR-1]
  └─ runMyProjects(cfg):
       ctx    := AssembleFromOS(baseURL)                 ── 009, resolve-once
       client := NewClientFromOS(ctx)                    ── 010; base-URL fail-fast lives here
       resp, err := client.Execute(reqCtx,
                      apiclient.Request{Method:"GET", Path:"/me/projects", Query:{status?}},
                      &list)                              ── Request is 010's; &list : glassfrog list-of-Project (decode target)
         ├─ success    → formatMyProjects(list) → stdout (projection + "more available" when HasNextPage) ; Success(0)
         ├─ *AuthError{NoCredentials}   → UsageError(2)
         ├─ *AuthError{CredentialError} → RuntimeError(1)
         ├─ *TransportError             → NetworkUnavailable(6)
         ├─ *ResponseError (non-2xx)    → APIError(3)              (generic; 015 refines 401/403→4, 429→5)
         └─ *DecodeError                → RuntimeError(1)
                                          │
              classifyClientError(err) ──┘  (internal/cli, 011 — REUSED)
              ExitCode(Outcome) ───────────► process code   (004 registry, codes added by 011)

internal/glassfrog (extended)       + Project{…}       ; reuses Pagination + list envelope (012/013)
internal/apiclient (010, consumed)  AssembleFromOS, NewClientFromOS, (*Client).Execute, typed errors  ── unchanged
internal/cli (011/013, consumed)    classifyClientError, Outcome, ExitCode, --base-url, validateStatus  ── REUSED, unchanged
```

The token takes exactly one path — 010's replay thunk into 007's `AuthTransport`. `my projects` **never reads `ctx.Cred.Token`**; its projection renders only response-side fields, so secret hygiene holds structurally.

---

## Architecture Decisions

### ADR-1: `Project` joins `internal/glassfrog`; `Pagination`/envelope and `validateStatus` are reused, not redefined

**Context**: `my projects` decodes the `GET /me/projects` body — the same paginated envelope `{data: [Project], meta: {pagination}}` as My Actions, with the same `?status=` filter and identical status enum (`spec/glassfrog-api-v5.yaml` `Project` + `Pagination`). 013 ADR-1/ADR-2 already established that paginated reads share one `Pagination`/envelope and that the spec-enumerated `--status` flag is validated by a shared `validateStatus`.

**Options considered**:
1. **Add `Project` to `internal/glassfrog`; reuse the shared `Pagination`/envelope (012/013) and the shared `validateStatus` (013)** — 014 adds only its resource model and command; everything else is reuse, matching the "one type, one validator, one place" discipline.
2. **Re-introduce a 014-local pagination type or a second status validator** — rejected: duplicates types/logic that 013 already made shared; the exact drift the project avoids.

**Decision**: Option 1. Add `Project` to `internal/glassfrog`; reuse `glassfrog.Pagination`, the list envelope, and `validateStatus` + the status set. Decoding is tolerant of unknown/extra fields. The projected `Project` subset and envelope shapes are interface-level.

**Consequences**: 014 is almost entirely reuse — the strongest evidence that 013's shared types and validator were designed for the siblings. *Conforms to 013 ADR-1/ADR-2 (silent precedent conformance); no new cross-spec precedent.*

### ADR-2: No `?include=` embedding — `/me/projects` offers no include parameter

**Context**: The `Project` schema carries `sub_projects` and `actions` arrays "only present when `?include=…` is requested," and `has_sub_projects`/`has_actions` booleans. But the `/me/projects` operation in the spec exposes **no `include` parameter** (only `per_page`, `cursor`, `status`). Identity Read (011) supports `--include roles` because `/me` documents the `include` parameter; `/me/projects` does not.

**Options considered**:
1. **No `--include` flag; project the `has_sub_projects`/`has_actions` booleans as presence signals** — faithful to the endpoint's contract; the booleans already tell the reader whether children exist without fetching them.
2. **Add a `--include` flag anyway (sub_projects/actions)** — rejected: the operation does not accept `include`, so the API would ignore or reject it; offering a flag the contract does not support violates "spec is the contract" (PROJECT) and forks the surface from what `/me/projects` actually does.

**Decision**: Option 1. `my projects` exposes no `--include` flag. The projection surfaces `has_sub_projects`/`has_actions` as booleans so the reader knows children exist; fetching them is out of scope (a future capability would need a project-detail read, not a list embed).

**Consequences**: A narrower, contract-faithful command than `me` — the only structural difference from My Actions. *Feature-specific (no cross-spec precedent): embedding is offered only where the operation documents an `include` parameter.*

### ADR-3: A guard-registered `projects` leaf under the `my` parent, with an injected seam and a pure `runMyProjects`/`formatMyProjects` (mirrors 013 ADR-4)

**Context**: CONSTITUTION IV requires hermetic, RED-first tests with no real network. 013 ADR-4 (itself following 011 ADR-5) established the leaf-under-`my` + injected-seam + pure-`run`/`format` pattern for the paginated reads.

**Options considered**:
1. **`newMyProjectsCommand(seam)` under the `my` parent (012), + pure `runMyProjects(cfg)` + pure `formatMyProjects(list) string`, reusing the shared `validateStatus`** — every branch runs offline over a fake transport; identical shape to `my actions`.
2. **A monolithic `RunE` / a top-level `my-projects` command** — rejected for the same reasons as 013 ADR-4 (not hermetic; forks the `my` grouping).

**Decision**: Option 1. The seam mirrors 013's `myActionsSeam`. `newMyProjectsCommand` registers `projects` under the `my` parent via the `MustRegister` guard; if the parent does not yet exist (012 not landed), it is created idempotently and the later sibling reuses it.

**Consequences**: Every outcome is exercised offline; the command is a near-mechanical twin of `my actions`. The seam interface, `--status` spelling, decode type names, and projection layout are interface-level. *Conforms to 013 ADR-4 (silent precedent conformance).*

---

## Data Model Design

`internal/glassfrog` (schema only; JSON-tagged structs; field names/tags pinned at `/score:interface`):

- **`Project`** — the `/me/projects` list item: `ID` (`proj_…` — the machine-actionable handle), `Description`, `Status` (the status enum), `RoleID` (`role_…`, **nullable** — null for non-role-owned projects), `Tags []string`, `HasSubProjects` ✓, `HasActions` ✓ (presence signals, projected); `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, `Link` (nullable), `Note` (nullable) decoded. The `sub_projects`/`actions` embed arrays are **not** modelled for this read (no `?include` on the operation — ADR-2). The projected subset (likely id, status, description, role, has-children flags, tags) is pinned at interface; decoding tolerates unknown/extra fields.
- **`Pagination`** (reused from 012/013) — `PerPage`, `HasNextPage`, `NextCursor`. Not redefined.
- **List envelope** (reused from 012/013) — `{Data []Project, Meta{Pagination}}`. `Project` decodes through it.

The token is never a field here. Secret hygiene holds by construction.

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — upstream dependency, landed on main)**: `my projects` calls `NewClientFromOS(ctx)` then `(*Client).Execute(reqCtx, Request{Method:"GET", Path:"/me/projects", Query:{status?}}, &list)`. Consumes 010's surface unchanged.
- **Identity Read (011, `internal/cli` + `internal/glassfrog` — upstream dependency, shaped not built)**: reuses the `glassfrog` package, `classifyClientError`, `Outcome`/`ExitCode` (codes 3/6), and the persistent `--base-url` flag. 014 adds none of these.
- **My Roles (012, `internal/cli` + `internal/glassfrog` — upstream dependency, in progress)**: reuses the `my` parent, `glassfrog.Pagination`, the list envelope, and the "more available" signal convention.
- **My Actions (013, `internal/cli` + `internal/glassfrog` — upstream dependency/twin)**: reuses the shared `validateStatus` + status set and the list-projection/seam shape. 014 differs only in resource (`Project`) and the no-`include` non-behavior.
- **Glassfrog API `GET /me/projects` (`listMyProjects`, system actor)**: returns `{data: [Project], meta: {pagination}}` on `200`, with an optional `status` filter; documents `400`/`401`/`404`. `my projects` decodes the 200 envelope; non-200 arrives as 010's generic `*ResponseError`.
- **Pagination (016) / API Error Extraction (015) / Rate-Limit Handling (017) — downstream siblings**: as for 013 — 016 owns the walk; 015/017 refine `APIError`(3) without renumbering.
- **Exit-Code Convention (004, `internal/cli`)**: outcomes map through the single `ExitCode` registry; 014 adds no codes.

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: `my projects` **never reads `ctx.Cred.Token`** — the token's only path is 010's replay thunk into 007's `AuthTransport`. `formatMyProjects` renders response-side fields only; `classifyClientError` messages name outcomes/paths, never the token. Pinned by a token-never-in-output test across success and every error branch.

**Error handling (CONSTITUTION III)**: fail loud at every fork — an unsupported `--status` is refused before any request; a base-URL error refuses at `NewClientFromOS`; a wire failure is `NetworkUnavailable`(6); a non-2xx is never treated as success (generic `APIError`(3)); a 2xx body that won't decode is a loud `RuntimeError`(1). Every outcome maps through the single `ExitCode` registry; the `default→1` fail-safe backstops any unmapped category.

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network or `~/.glassfrogrc`. The injected seam binds a fake base `http.RoundTripper` returning canned `GET /me/projects` responses, so every branch runs offline: 200 single page → projection; 200 with `has_next_page` → projection + "more available" signal; 200 empty `data` → success with an empty list; 200 with `?status=current` → request carried the filter; non-2xx → `APIError`(3); transport error → `NetworkUnavailable`(6); no-token context → `UsageError`(2); broken-cred context → `RuntimeError`(1); undecodable 200 → `RuntimeError`(1); base-URL-error context → `UsageError`(2); unsupported `--status` → `UsageError`(2) with no request issued (tripwire). `formatMyProjects` is unit-tested pure; `validateStatus` is already pinned by 013's tests (014 adds a thin reuse test). The driving scenarios become a **new godog suite** pointed at its **own** feature file `features/self-service-reads/my-projects.feature` (LEARNINGS); step helpers return errors, never panic; stderr/stdout capture uses a temp file, not `os.Pipe` (PR #10 LEARNINGS).

---

## Implementation Strategy

Three phases, mostly linear, mirroring 013. **Upstream dependencies**: 010 is **landed on main**; 011 and 012 are **shaped/in-progress but not yet on main**; 013 (My Actions) introduces the shared `validateStatus` + status set that 014 reuses. 014 should be cut from a main that carries 010, 011, 012, and 013's `validateStatus`.

- **Phase 1 — `glassfrog.Project`**: add `Project` with JSON tags and tolerant decode; reuse `glassfrog.Pagination` + the list envelope. RED-first unit tests: a `GET /me/projects` fixture decodes into the envelope (single page, multi-page with `next_cursor`, empty `data`); a null `role_id` decodes; `has_sub_projects`/`has_actions` decode; extra/unknown fields are ignored. *Depends on: nothing built (leaf schema); reuses 012/013's `Pagination`/envelope.*
- **Phase 2 — `my projects` command: `newMyProjectsCommand` + `runMyProjects` + `formatMyProjects` + seam + wiring**: the cobra leaf under the `my` parent, the injected seam (prod binds `AssembleFromOS`/`NewClientFromOS`), the pure `runMyProjects` orchestration (reuse `validateStatus` → assemble → build client → execute → render/classify), the pure `formatMyProjects` projection + signal, and one `MustRegister` wiring line under `my`. RED-first unit tests over the fake-transport seam for every branch; godog step definitions for the driving scenarios in the new suite. *Depends on: Phase 1 (decode target), and 010 + 011 + 012 + 013's `validateStatus` implemented.*

(Two phases rather than three: 014 introduces no new validator — it reuses 013's — so the only new code is the resource model and the command.)

---

## Risks

- **014 built before 011/012/013 land** (medium likelihood, high impact): Phase 2 reuses 011's `classifyClientError`/`Outcome`/`ExitCode`/`--base-url`, 012's `my` parent/`Pagination`/envelope/signal, and 013's `validateStatus`. *Mitigation*: cut 014 from a main carrying 011, 012, and 013; the tasks graph marks the dependency. If 014 leads 013, it must own the shared `validateStatus` (the first status-filtered read to land creates it) and 013 reuses — coordinate so only one copy exists. Preferred order: 010 → 011 → 012 → 013 → 014.
- **012's pagination/signal shape differs from the assumed convention** (medium likelihood, medium impact): same as 013 — `formatMyProjects` isolates the signal rendering so adapting is localized; the spec marks the signal `[ASSUMED]` / "tracks 012".
- **Token leak through the projection or an error** (low likelihood, high impact): *Mitigation*: never reads `ctx.Cred.Token`; renders response-side fields only; pinned by a token-never-in-output test across every branch.
- **`Project` over-modelled with the `?include` embeds** (low likelihood, low impact): the schema carries `sub_projects`/`actions` arrays, but `/me/projects` offers no `include`. *Mitigation*: ADR-2 — do not model or offer the embeds for this read; project the `has_*` booleans instead.
- **Generic non-2xx → code 3 coarse until 015/017** (medium likelihood, low impact): same as 013; code 3 is the residual bucket, split later without renumbering.

---

## What This Plan Does Not Cover

- **Multi-page traversal** — Pagination (016); 014 fetches one page and signals more exist.
- **`?include=sub_projects`/`actions` embedding** — not offered by the `/me/projects` operation (ADR-2); a future project-detail read, not this list.
- **`--output json` (structured output)** — a deferred, cross-cutting capability (a future persistent root flag per 011); `my projects` ships the reshaped projection only (a Non-Behavior).
- **Per-status API error interpretation** — API Error Extraction (015) and Rate-Limit Handling (017), which split `APIError`(3) without renumbering.
- **The reused foundations** — `internal/glassfrog` + `classifyClientError` + `Outcome`/`ExitCode` + `--base-url` (011); the `my` parent + `Pagination` + envelope + signal (012); `validateStatus` + the status set (013). 014 reuses them unchanged.
- **The exact Go API shapes** — the seam interface, `--status` flag spelling, `Project`/envelope struct fields and JSON tags, and the projection/signal layout — `/score:interface` pins these.
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/self-service-reads/my-projects.feature`.
- **010's `apiclient` implementation** — a separate spec (010), landed on main; 014 consumes it unchanged.
