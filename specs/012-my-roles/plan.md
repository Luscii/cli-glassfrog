# Plan: My Roles

**Feature**: 012-my-roles
**Role**: Shaper
**Inputs**: spec.md (012-my-roles); PROJECT.md; `.score/memory/DECISIONS.md` (relevant precedent: the producer-classifies / consumer-maps split — 002/004/005/007/008/009; 004's single `ExitCode(Outcome)` registry with frozen codes 0–6; **010's implemented seam** — `Request{Method,Path,Query,Body}`, `NewClient(ctx,base)` / `NewClientFromOS(ctx)`, `(*Client).Execute(reqCtx, req, out any) (*Response,error)`, `*Response{StatusCode,Header}`, and typed `*TransportError`/`*ResponseError{Status,Header,Body}`/`*DecodeError`, `requestTimeout=30s`, `*AuthError` discriminated via `errors.As` first; **011 Identity Read's landed scaffolding** — `me` is a runnable command with a persistent root `--base-url` flag, a shared `classifyClientError(err) Outcome` helper in `internal/cli`, the operational `Outcome` categories `NetworkUnavailable`(6)/`APIError`(3) added at the single registry, and the `internal/glassfrog` schema package whose `Role{ID,Name}` 012 grows; guard-registered cobra leaves wired in `Assemble` — 001; inject seams, fail-fast on nil — 005/008/PR #20); `.score/memory/LEARNINGS.md` (a godog suite points at its own feature file; step helpers return errors; pin frozen values with regression tests; tests never touch the real network or `~/.glassfrogrc`). No SOUL.md.

**Conformance note**: Identity Read (011) landed on `main` first (spec-through-guard) and **owns the shared self-service-read scaffolding**. Per the "first-to-land creates, the other conforms" rule (005/006, 007/008), this plan **conforms to 011's recorded contract**: 012 attaches a `roles` subcommand under 011's runnable `me` command, reuses 011's `classifyClientError` and exit-code mapping, inherits the persistent root `--base-url`, and grows 011's `internal/glassfrog.Role` rather than defining its own type. The exit-code reconciliation (developer decision, 2026-06-07): adopt 011's mapping — the local auth fail-safe maps to **2/1**, not a new `PermissionError`(4).

**Readiness**: Must met + Should substantial. One open `[NEEDS CLARIFICATION]` by design: the *form* of the incompleteness signal (resolved in interface-cli.md as a stderr line). No architectural unknown requires a resolve — the seam (010) is implemented and 011's contract is recorded.

---

## System Architecture

My Roles is a self-service read on the proven, **now-implemented** transport chain: `glassfrog me roles` sends `GET /me/roles` and prints a reshaped projection of the roles the practitioner fills, or fails with a named error and the right exit code. It is **the second command on the self-service-read surface**, attaching to the scaffolding Identity Read (011) established.

It adds **no new transport, identity, resolution, or registry logic** — it conforms to and reuses what 010 and 011 provide:

```
internal/cli

  me command (011, runnable: `glassfrog me [--include roles]`)
    └── roles subcommand (012, this slice)            ── MustRegister under me; Args: NoArgs; Short set
          RunE (thin seam over runMyRoles):
            flag  := root persistent --base-url (011, inherited)         ── apiclient.FlagBaseURL
            ctx   := apiclient.AssembleFromOS(flag)                      ── resolve once (009)
            client, err := apiclient.NewClientFromOS(ctx)                ── build once (010); err = base-URL fail-fast
            resp,  err  := client.Execute(reqCtx,
                             apiclient.Request{Method:"GET", Path:"/me/roles"}, &out)   ── send (010)
                                │
            success ───────────┤ formatMyRoles(out) → stdout; stderr note if out.Meta.Pagination.HasNextPage; Success(0)
            error   ───────────┘ classifyClientError(err) (011, shared) → Outcome → ExitCode (004)
                                   *AuthError{NoCredentials}  → UsageError        → 2
                                   *AuthError{CredentialError}→ RuntimeError       → 1
                                   *TransportError            → NetworkUnavailable → 6
                                   *ResponseError (non-2xx)   → APIError           → 3
                                   *DecodeError               → RuntimeError       → 1
                                   base-URL error             → UsageError         → 2

  internal/glassfrog (011 schema package)
    Role{ID,Name}  ──grown by 012──►  Role{ID,Name,Purpose,Accountabilities[],Domains[]}   (one shared type)
    + Pagination{PerPage,HasNextPage,NextCursor}  (012 adds — shared, reused by 013/014)
    + MyRolesResponse{Data []Role, Meta{Pagination}}                                       (012 adds)
```

Everything 012 touches is additive: a `roles` leaf under `me`, growth of the shared `Role` type plus a `/me/roles` response wrapper, and a pure projection renderer. The boundary is the **CLI invocation surface** (`me roles`, stdout projection, stderr failures, exit codes). The exact projected-field labelling and the incompleteness-signal form are the interface skill's concern.

**Dependencies**: 010 (Request Execution) is **implemented** on `main`. 011's scaffolding (the runnable `me` command, the root `--base-url`, `classifyClientError`, and `internal/glassfrog`) is **specified and recorded but not yet implemented** — 012's implementation is gated on it (Implementation Strategy, Phase 0).

---

## Architecture Decisions

### ADR-1: `me roles` is a `roles` subcommand under 011's runnable `me` command

**Context**: My Roles is a token-scoped self-service read (`GET /me/roles`). 011 landed `me` as a **runnable** command (`glassfrog me [--include roles]` prints the authenticated identity). 012 needs `glassfrog me roles`. The existing `roles` group is the 001 stub for *org-wide* Governance Reads — a different surface.

**Options considered**:
1. **Attach `roles` as a subcommand of 011's runnable `me` command** — `glassfrog me roles` is a `NoArgs` leaf registered through the guard under `me`; `me` keeps its own identity action and gains a child. Matches 011's landed shape and the ROADMAP "/me, my roles" framing; siblings `me actions` (013) / `me projects` (014) attach the same way.
2. **A separate non-runnable `me` group** (012's pre-conformance assumption) — rejected: 011 already made `me` runnable; a second, non-runnable `me` would collide at the registration guard.
3. **A flat top-level `my-roles` command** — rejected: abandons the `me` grouping 011 established and the API's `/me/*` family.

**Decision**: Option 1. `me` is 011's runnable command; 012 registers a `roles` `NoArgs` leaf under it via the guard, with a non-empty `Short`. The leaf's `RunE` is a thin seam delegating to a pure core.

**Consequences**: `me` becomes a command that is **both runnable and a parent** — verify the 001 registration guard permits this (its rules were "leaf-has-action" / "group-has-children"; a runnable parent satisfies both, but confirm the guard doesn't treat them as mutually exclusive — a coordination point with 011 at implementation). `me roles` takes no positional args and no filters (the endpoint offers none beyond paging); stray input is a usage error via `Args: NoArgs`. *Precedent-setting: self-service sub-reads attach as subcommands under the runnable `me` command.*

### ADR-2: Conform to 011's landed scaffolding — reuse, don't recreate

**Context**: 010's plan named "the first consuming command" as the creator of the shared scaffolding (the `me` command, the root `--base-url` flag, the operational `Outcome` categories + `ExitCode` cases, and a response-schema home). **011 landed first** and created that contract: a runnable `me` command, `--base-url` persistent on the root, a shared `classifyClientError(err) Outcome` helper adding `NetworkUnavailable`(6)/`APIError`(3) at the single registry, and the `internal/glassfrog` schema package.

**Options considered**:
1. **012 conforms** — reuses every shared piece (the `me` command, the root flag, `classifyClientError`, `internal/glassfrog`) and adds only its own `roles` leaf, the grown `Role` fields, and its projection. The "other conforms" half of the first-to-land rule (005/006, 007/008).
2. **012 re-creates its own scaffolding** — rejected: a duplicate `me` command collides at the guard; a second classifier/registry edit forks 004's single-registry invariant; a command-local Role type duplicates `internal/glassfrog.Role` (011 explicitly says 012 grows the shared type).

**Decision**: Option 1. 012 adds **no new `Outcome` category and no registry edit** — its outcomes map through 011's `classifyClientError` to categories that already exist after 011 (`NetworkUnavailable`/`APIError`) or pre-date it (`UsageError`/`RuntimeError`/`Success`). It registers no `--base-url` flag (inherited from root). It defines no Role type (grows 011's).

**Consequences**: 012's footprint shrinks to its genuinely-own work. The implementation gate becomes 011's scaffolding existing (Phase 0). If — contrary to current state — 012 were to implement before 011, it would create those pieces *to 011's recorded contract*, not its own design. *Precedent-setting: later self-service reads conform to 011's scaffolding; they reuse `classifyClientError` and grow `internal/glassfrog` types.*

### ADR-3: Reuse 011's `classifyClientError` and exit-code mapping (auth fail-safe → 2/1, no `PermissionError`)

**Context**: 010 hands the command typed code-free errors. 011 introduced the single `classifyClientError(err) Outcome` chain and the mapping that fills 004's reserved codes. 012's pre-conformance ADR-3 proposed routing `*AuthError` to a new `PermissionError`(4); the developer reconciled (2026-06-07) to **adopt 011's mapping** instead — it mirrors the shipped `runLogin` (no-token→2, bad-file→1) and reserves 4/5 for genuine API-side outcomes (015 splits 401/403→4, 017 splits 429→5).

**Decision**: 012 routes every 010 error through 011's shared `classifyClientError`:

| 010 outcome | `Outcome` (via classifyClientError) | Exit code |
|---|---|---|
| `*AuthError{NoCredentials}` (no usable token) | `UsageError` | **2** |
| `*AuthError{CredentialError}` (unreadable credential file) | `RuntimeError` | **1** |
| `*TransportError` (wire / DNS / TLS / timeout) | `NetworkUnavailable` | **6** |
| `*ResponseError` (generic non-2xx, incl. 401/403/429) | `APIError` | **3** |
| `*DecodeError` (2xx body won't parse) | `RuntimeError` | **1** |
| base-URL error from `NewClient` | `UsageError` | **2** |
| success | `Success` | **0** |

No new `Outcome` value and no `ExitCode` edit are introduced by 012 — the categories exist after 011. `PermissionError`(4) and `RateLimited`(5) stay reserved with no producer until 015/017.

**Consequences**: One registry, one classifier, one consistent mapping across `me`, `me roles`, and every later read — by reuse, not duplication. The classifier's exhaustiveness test (011, PR #10 LEARNINGS: `len`+comma-ok) guards the chain; 012 adds no divergent mapping for it to catch. *Conforms to the DECISIONS contract recorded from 011.*

### ADR-4: Decode into the shared `internal/glassfrog` types — grow `Role`, add a `/me/roles` wrapper

**Context**: `Execute` decodes the 2xx body into a caller `out any` (JSON). 011 created `internal/glassfrog` with a minimal `Role{ID,Name}` and recorded that **My Roles grows the SAME type** (never a second Role). The `/me/roles` body is `{ data: [Role], meta: { pagination } }`.

**Options considered**:
1. **Grow the shared `internal/glassfrog.Role`** (add `Purpose`, `Accountabilities []{Description}`, `Domains []{Description}` with JSON tags), define a reusable named `Pagination{ PerPage, HasNextPage, NextCursor }` (the shared list-pagination model 013/014 reuse — created here per the 013 DECISIONS contract, matching the API's `Pagination` schema incl. `next_cursor`), and add a `MyRolesResponse{ Data []Role; Meta{ Pagination Pagination } }` wrapper referencing it; feed it to a pure `formatMyRoles(resp) string` and a pure `incomplete(meta) bool` (reads `HasNextPage`). `NextCursor` is decoded but unused until Pagination (016).
2. **A command-local decode DTO** (012's pre-conformance assumption) — rejected: 011 explicitly homes read models in `internal/glassfrog` as shared types; a local DTO duplicates `Role` and forks the schema.

**Decision**: Option 1. The grown `Role` carries the projected fields (decoding is tolerant of unknown/extra fields, so `me`'s minimal use and `me roles`' fuller use share one type); `MyRolesResponse` is the decode target. `formatMyRoles` renders the projection (name + `role_…` id; `Purpose:`; `Domains:` then `Accountabilities:`, each header always present, `(none)` when empty; `(no purpose set)` for null; `No roles.` for empty). `incomplete` derives from `Meta.Pagination.HasNextPage`.

**Consequences**: Output contains only the projected fields by construction; `me --include roles` and `me roles` stay schema-consistent. The renderer is pure and thin, so labelling stays aligned with 011's projection convention. The incompleteness-signal *form* is interface-level. *Conforms to 011's `internal/glassfrog` decision; feature-local beyond that.*

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — implemented, upstream)**: `NewClientFromOS(ctx)` builds the client once (base-URL fail-fast verbatim); `client.Execute(reqCtx, Request{Method:"GET", Path:"/me/roles"}, &resp)` sends and returns the typed outcome.
- **Connection Context Assembly (009)**: `AssembleFromOS(flagValue)` resolves the context once from the inherited root `--base-url`.
- **Identity Read (011, `internal/cli` + `internal/glassfrog` — the scaffolding this slice conforms to)**: 012 attaches under the runnable `me` command; reuses `classifyClientError`; inherits the persistent root `--base-url`; grows `internal/glassfrog.Role` and adds `MyRolesResponse`.
- **Request Authentication (007, via 010)**: the no-token / broken-credential fail-safe surfaces as `*AuthError`; `classifyClientError` maps `NoCredentials`→2, `CredentialError`→1. The command never reads the token.
- **Exit-Code Convention (004, `internal/cli`)**: outcomes map through 011's `classifyClientError` + the existing `ExitCode` registry; 012 adds no case.
- **Glassfrog API `GET /me/roles`**: inbound read only; non-2xx→`APIError`/3, unreachable→`NetworkUnavailable`/6, undecodable 2xx→`RuntimeError`/1.
- **User / AI agent (CLI surface)**: projection to stdout; failures and the incompleteness note to stderr.

---

## Cross-cutting Concerns

**Testing (CONSTITUTION IV)** — RED-first, hermetic, no real network. Following 011's read-command pattern: a thin cobra `RunE` over an injected seam (prod binds `AssembleFromOS`/`NewClientFromOS` + the real base transport; tests bind a fake `http.RoundTripper` returning canned `/me/roles` JSON) delegating to a pure `runMyRoles(cfg)`; pure `formatMyRoles(MyRolesResponse) string` and `incomplete(meta) bool` unit-tested directly. Branches covered: multi-role projection (name/purpose/domains/accountabilities/minimal-id; fillers/tags/flags absent), empty list (`No roles.`, exit 0), `has_next_page` (stderr note, exit 0), no-token (`*AuthError{NoCredentials}`→exit 2), unreadable credential (`*AuthError{CredentialError}`→exit 1), transport (exit 6), non-2xx (exit 3), undecodable 2xx (exit 1), base-URL error (exit 2, nothing sent), stray arg (`NoArgs`→exit 2). A token-never-in-output test covers stdout+stderr on every branch. Command scenarios become a godog suite pointed at `features/self-service-reads/my-roles.feature` (its own file). Reuse `classifyClientError`'s exhaustiveness test — 012 adds no mapping to it.

**Secret hygiene (CONSTITUTION II)**: the command never reads `ctx.Cred.Token`; output is the projection; the typed errors are token-free (010); the token is never a `Role`/`MyRolesResponse` field (it is a request header).

**Error handling (CONSTITUTION III)**: fail loud at every fork; empty list is a success; the incompleteness note prevents a partial list reading as complete. Every error message names a cause **and** a next step (the resolved checklist P0; interface-cli § Error Communication).

**Command registration (001/002)**: the `roles` leaf registers through the guard under the runnable `me` command (verify runnable-with-children is permitted); `Args: NoArgs`; no package-global cobra toggle changes; `--base-url` is inherited, not re-registered.

---

## Implementation Strategy

Three phases plus an external prerequisite. All 012 work is additive in `internal/cli` and `internal/glassfrog` (plus the feature file).

- **Phase 0 — prerequisites**: 010 (Request Execution) is **implemented** ✓. 011's scaffolding — the runnable `me` command, the persistent root `--base-url`, `classifyClientError`, and `internal/glassfrog` (`Role{ID,Name}`) — is **recorded but not yet implemented**; T-2/T-3 depend on it. *(If 011 implements first, 012 builds on it; if 012 reaches implementation first, it creates those pieces to 011's recorded contract, then 011 conforms.)*
- **Phase 1 — schema growth (T-1)**: grow `internal/glassfrog.Role` with `Purpose`, `Accountabilities []{Description}`, `Domains []{Description}` (JSON-tagged, tolerant of extra fields), define the reusable named `Pagination{PerPage, HasNextPage, NextCursor}` (shared with 013/014, matching the API schema), and add `MyRolesResponse{Data, Meta{Pagination}}` referencing it; pure `formatMyRoles` + `incomplete`, unit-tested. Independent of the command wiring; can start as soon as `internal/glassfrog` exists. *Depends on: 011's `internal/glassfrog` (or creates it to contract).*
- **Phase 2 — the `me roles` command (T-2)**: register the `roles` `NoArgs` leaf under `me`; `RunE` → seam → `runMyRoles` running the build-once flow (`AssembleFromOS(rootFlag)` → `NewClientFromOS` → `Execute GET /me/roles` into `MyRolesResponse`), mapping errors via **011's `classifyClientError`**, projecting to stdout, and writing the incompleteness note to stderr. Hermetic tests over a fake base transport for every branch. *Depends on: Phase 1, 010 (done), 011's `classifyClientError` + runnable `me` + root flag.*
- **Phase 3 — executable acceptance (T-3)**: godog step definitions for `my-roles.feature` with a fake base transport and canned payloads; assert projected fields, absences, the incompleteness note, and per-scenario exit codes (0/1/2/3/6). *Depends on: Phase 2.*

---

## Risks

- **011's scaffolding not yet implemented** (high likelihood, blocks implementation only): 012's Phases 2–3 need the runnable `me` command, `classifyClientError`, the root `--base-url`, and `internal/glassfrog`. *Mitigation*: tracked as Phase 0; 011 is ahead on `main` and will likely implement first; if not, 012 creates the pieces to 011's recorded contract. Phase 1 (schema growth) and the spec/plan are unblocked.
- **`me` runnable-with-children vs the 001 guard** (medium likelihood, low impact): `me` must be a command that both runs (011) and parents `roles` (012); the guard's "leaf-has-action / group-has-children" rules must permit it. *Mitigation*: confirm at implementation with 011; a runnable parent satisfies both predicates, but pin it with a registration test.
- **Token leak through output or an error line** (low likelihood, high impact): a command on the live request path. *Mitigation*: never reads the token; token-never-in-output test across all branches; never a response-model field.
- **Projected-field labelling drifts from 011's projection convention** (medium likelihood, low impact): 011's `formatMe` and 012's `formatMyRoles` should read consistently. *Mitigation*: field selection is spec-fixed; both renderers are pure and reviewed against each other; the shared `Role` type keeps field names aligned.
- **Incompleteness-signal form** (low likelihood, low impact): resolved in interface-cli as a single stderr line at exit 0; the *behavior* is locked in the spec.

---

## What This Plan Does Not Cover

- **010's seam internals** and **011's scaffolding internals** — consumed/conformed-to, not designed here.
- **Following pagination** — Pagination (016); this slice signals incompleteness only.
- **Classifying the non-2xx `*ResponseError`** — API Error Extraction (015) splits 401/403→4 later; **`429` backoff** — Rate-Limit Handling (017) adds `RateLimited`→5.
- **A raw `--output json` mode** — the Unconsumable Output capability (a future cross-cutting flag, likely persistent on the root like `--base-url`).
- **The exact projected-field labelling and the incompleteness-signal form** — interface-cli.md.
- **Executable Gherkin** — `/score:scenarios` produced `features/self-service-reads/my-roles.feature`.
