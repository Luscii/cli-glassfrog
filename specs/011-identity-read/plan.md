# Plan: Identity Read

**Feature**: 011-identity-read
**Role**: Shaper
**Inputs**: spec.md (011-identity-read); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: `internal/apiclient` is Connection Configuration's home — base URL/timeouts/transport — 008/010; 009 `AssembleFromOS(flagValue) ConnectionContext` resolves once per invocation; 010 `NewClientFromOS(ctx)` + `(*Client).Execute(reqCtx, Request, out)` is the send seam returning a `*Response` or typed code-free `*AuthError`/`*TransportError`/`*ResponseError`/`*DecodeError`/base-URL error; 004 `internal/cli/exitcode.go` is the single `ExitCode(Outcome)` registry with frozen codes 0–6, codes 3–6 reserved for "the first consuming command (011)" to add the `Outcome` categories + cases; the producer-classifies-a-code-free-outcome / consumer-maps split — 002/004/005/007/008/009/010; cobra leaf + injected seam + pure `run*` over injected values — 006 `runLogin`/`loginSeam`; `MustRegister` guard + single `Assemble()` wiring site — 001; `FlagBaseURL="base-url"` already a constant in `apiclient`); `.score/memory/LEARNINGS.md` (relevant: inject seams / fail-fast on nil — PR #20; a godog suite points at its OWN feature file; godog step helpers return errors, never panic; capture stderr/stdout with a temp file not `os.Pipe`; a table-driven change-detector must guard `len()`+comma-ok so a dropped zero-valued entry fails loud — PR #10; `os.Pipe`-free capture). No SOUL.md.

**Readiness**: Must met + Should substantial — behavioral accord (Entry / Reading / Output / Failure), three happy-path + two error + three edge scenarios, four validation scenarios, eight non-behaviors, integration boundaries naming every dependency (010/007/009/008/API/004/012), user scenarios, assumptions. Strong foundation. The three behavioral forks were resolved during `/score:define` (opt-in `--include roles`; reshaped projection with `--output json` deferred; generic-outcome-mapped-via-004 failures). **Two architectural unknowns were resolved in this session's resolve phase**: (Q1) the API response models live in a new `internal/glassfrog` package; (Q2) `--base-url` registers as a persistent flag on the root command. The remaining `[ASSUMED]` items (exact command name, flag/projection shape, decode type names) are interface-level, not behavioral gaps.

---

## System Architecture

Identity Read is **the CLI's first end-to-end read** — the `me` command that finally turns the assembled chain into a runnable command. The prior slices built every part but no consumer: 008 resolves the base URL, 005/007 the token + authenticated transport, 009 assembles the `ConnectionContext`, 010 builds the `Client` and the `Execute` send seam. 011 is the first command to **call through that seam to the live API**, decode the `GET /me` response, and surface a result to the operator. It is the smallest slice that proves the whole stack works against the real Glassfrog API, and it sets the shape every later read (My Roles 012, My Actions 013, My Projects 014) follows.

It is also the slice that closes three gaps the prior slices each forecast and deferred to "the first consuming command": (1) the **`--base-url` flag registration** that triggers a real call; (2) the **operational `Outcome` categories + `ExitCode` cases** (codes 3–6) that 004 reserved but left producer-less; and (3) the first **API response model** to decode into. 011 is therefore not purely additive — it adds the persistent `--base-url` flag to the root and extends the `Outcome` enum + `ExitCode` registry (two existing files), alongside new files.

The parts:

- **`internal/glassfrog` (NEW package — API schema)** — the Go structs the read surface decodes into: `MeResponse{Actor, Organization, Membership, Roles []Role}`, `Actor{ID, Name, Kind}`, `Organization{ID, Name}`, `Membership{AccessLevel, …}`, and a minimal `Role{ID, Name}` (the `?include=roles` embed; the full role shape grows with My Roles 012). This package holds *schema only* — no transport, no cobra, no exit codes. It is the shared home so `me --include roles` and `my roles` (012) decode the **same** `Role` type (ADR-1).
- **`internal/cli/me.go` (NEW)** — `newMeCommand(seam)`: a guard-registered cobra leaf (`Use:"me"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) carrying the `--include` flag. Its `RunE` reads the persistent `--base-url` value, delegates to a pure `runMe(cfg)` over injected values (the `runLogin` pattern), and maps the returned code-free `Outcome` onto dispatch's error channel.
- **`runMe` + `formatMe` + `validateInclude` (pure, in `me.go`)** — `validateInclude` rejects an unsupported `--include` target *before* any request (usage error, fail-fast); `runMe` assembles the context, builds the client, calls `Execute(reqCtx, apiclient.Request{Method:"GET", Path:"/me", Query:{include}}, &me)` (the `Request` descriptor is 010's; the `&me` decode target is `glassfrog.MeResponse`), and on success calls `formatMe(me, includeRoles)` to render the reshaped projection; on a typed error it calls the shared classifier and writes a token-free message. `formatMe` is a pure `MeResponse → string` renderer (testable in isolation; the projection-per-read convention 012–014 reuse).
- **`internal/cli` client-error classifier (NEW shared helper) + `Outcome`/`ExitCode` extension (MODIFIED)** — a `classifyClientError(err) (Outcome, …)` that `errors.As`-discriminates 010's typed errors into `Outcome` categories at the **single registry site** (ADR-3/4). Adds `NetworkUnavailable` and `APIError` to the `Outcome` enum (`dispatch.go`) and their cases to `ExitCode` (`exitcode.go`), taking reserved codes 6 and 3.
- **Persistent `--base-url` flag on the root (MODIFIED `app.go`/`root.go`)** — registered once on the root command, inherited by every API command (ADR-2). Its value flows into `AssembleFromOS(flagValue)`.

```
glassfrog me [--include roles]            (cobra leaf, internal/cli/me.go, wired once in Assemble())
  │  read persistent --base-url value (root flag, ADR-2)
  │  validateInclude(flag) ─ unsupported target → UsageError, no request (fail-fast)   [ADR-5]
  └─ runMe(cfg):
       ctx    := AssembleFromOS(baseURL)                 ── 009, resolve-once
       client := NewClientFromOS(ctx)                    ── 010; base-URL fail-fast lives here
         │   (err ≠ nil → base-URL error → classify → UsageError(2))
       resp, err := client.Execute(reqCtx,
                      apiclient.Request{Method:"GET", Path:"/me", Query:{include?}},
                      &me)                                ── Request is 010's; &me : glassfrog.MeResponse (decode target)
         ├─ success    → formatMe(me, includeRoles) → stdout ; Success(0)
         ├─ *AuthError{NoCredentials}   → UsageError(2)   "not authenticated — run glassfrog auth login"
         ├─ *AuthError{CredentialError} → RuntimeError(1)  broken .glassfrogrc
         ├─ *TransportError             → NetworkUnavailable(6)   [NEW Outcome]
         ├─ *ResponseError (non-2xx)    → APIError(3)             [NEW Outcome, generic; 015 refines 401/403→4, 429→5]
         └─ *DecodeError                → RuntimeError(1)
                                          │
              classifyClientError(err) ──┘  (internal/cli, errors.As chain, single registry; ADR-3/4)
              ExitCode(Outcome) ───────────► process code   (004, the one mapper)

internal/glassfrog (NEW)            MeResponse{Actor,Organization,Membership,Roles[]Role}, Role{ID,Name}  (decode targets; Request is 010's)
internal/apiclient (010, consumed)  AssembleFromOS, NewClientFromOS, (*Client).Execute, typed errors  ── unchanged
internal/cli (004, extended)        Outcome += NetworkUnavailable, APIError ; ExitCode += cases 6, 3
```

The token takes exactly one path — the replay thunk 010 hands to 007's `AuthTransport`. `me` **never reads `ctx.Cred.Token`**; its projection renders only response-side fields (the API's reply), so the secret-never-emitted rule holds structurally.

---

## Architecture Decisions

### ADR-1: API response models live in a new `internal/glassfrog` schema package (resolves Q1)

**Context**: `me` must decode the `GET /me` body into a Go value (`MeResponse` = actor + organization + membership, plus an optional embedded `roles` array — `spec/glassfrog-api-v5.yaml` `MeResponse`/`Actor`/`Organization`/`Membership`/`Role`). The `Role` type is shared from day one: `me --include roles` decodes it *and* My Roles (012) decodes it. Where the read surface's response models live is precedent-setting for 012–017 (Action, Project, Proposal, …). 010 scoped `internal/apiclient` as transport-only ("`apiclient` never imports `internal/cli`"); `internal/cli` owns commands, not domain types.

**Options considered**:
1. **A new `internal/glassfrog` schema package** — Actor, Organization, Membership, MeResponse, Role, … as plain structs with JSON tags; no transport, no cobra. The read commands decode into these; 012–017 grow the package. Clean transport-vs-schema split; one shared `Role`.
2. **Put the models in `internal/apiclient`** — common in Go API clients, but broadens 010's deliberately transport-only package into transport+schema, and every endpoint model would pile into the connection package.
3. **Define models locally in `internal/cli` per command** — rejected: `cli` would own domain types (a layering smell), and `Role` would duplicate across `me` and `my roles` (012), the exact drift the project avoids (one reader, one type).

**Decision**: Option 1. A new `internal/glassfrog` package holds the API resource structs. `me` decodes `GET /me` into `glassfrog.MeResponse`; the embedded roles use `glassfrog.Role` (minimal `ID`+`Name` now; My Roles 012 grows it to the full role shape — same type, one place). The package depends on nothing internal (it is leaf schema), so both `cli` and `apiclient` may import it without a cycle.

**Consequences**: A clean three-way layering — `glassfrog` (schema) ← `cli` (commands) and `apiclient` (transport), neither of which owns domain types it shouldn't. 012–014 add their resource models here and reuse `Role`. The exact struct/field/JSON-tag shape is interface-level (`/score:interface`). *Precedent-setting: API response models live in `internal/glassfrog`; the read commands decode into shared schema types, never command-local duplicates.*

### ADR-2: `--base-url` registers as a persistent flag on the root command (resolves Q2)

**Context**: `me` is the first command to trigger a real API call, so it must register the `--base-url` flag whose value feeds `AssembleFromOS(flagValue)` (008's highest-precedence rung; `FlagBaseURL="base-url"` is already an `apiclient` constant). 010 explicitly left this registration "to the first consuming command." Every future API command (012–014, proposals) needs the same flag; the base URL is a connection-wide concern, not a per-command one.

**Options considered**:
1. **A persistent flag on the root command** — registered once in `Assemble()`/root construction, inherited by every current and future subcommand. One registration, no duplication; base URL is global connection config. Wart: it also appears (inert) on `version`/`auth`.
2. **A local flag on `me`** (or a shared `addBaseURLFlag(cmd)` registrar each API command calls) — keeps the flag off non-API commands, but every API command must opt in, and the flag definition risks drifting across call sites.

**Decision**: Option 1. Register `--base-url` as a persistent flag on the root in the wiring layer; `me` (and 012–017) read its resolved value via cobra's flag inheritance and pass it to `AssembleFromOS`. The flag name/usage string come from the existing `apiclient.FlagBaseURL` constant so the precedence-chain rung and the registered flag can't drift.

**Consequences**: One flag definition for the whole CLI; adding an API command needs no flag wiring. `Assemble()`/root construction is no longer flag-free — a regression for 003's help tests (root help now shows a global-flags section), so those tests are updated in the same change (it is allowed: 003's narrowed non-behavior forbids only new *required* documentation data, and a persistent flag with a description is optional). The inert appearance on `version`/`auth` is accepted. *Precedent-setting: global connection options (starting with `--base-url`) are persistent root flags; API commands read them by inheritance, not by re-registering.*

### ADR-3: `me` is the first consuming command — it adds the operational `Outcome` categories + `ExitCode` cases and the shared client-error classifier at the single registry

**Context**: 004 froze codes 0–6 and reserved 3–6 for "the first consuming command" to populate when its producer exists; 010 produces typed code-free errors (`*TransportError`, generic non-2xx `*ResponseError`, `*DecodeError`) and propagates 007's `*AuthError`, but wires no exit code (it has no command). 011 is that first command. The classification must live at `internal/cli`'s single `ExitCode(Outcome)` registry (the producer-classifies / consumer-maps spine), and be reusable by 012–017 (all reads classify the same client errors).

**Options considered**:
1. **Add `NetworkUnavailable` + `APIError` to the `Outcome` enum and their `ExitCode` cases, and a shared `internal/cli` `classifyClientError(err) Outcome` that `errors.As`-discriminates 010's typed errors** — the read commands call the one classifier; the enum + registry grow once at the canonical site. Reused verbatim by 012–017.
2. **Each read command inlines its own `errors.As` chain and maps to codes directly** — rejected: bypasses the `Outcome`→`ExitCode` registry (002/004's spine), duplicates the chain across 012–017, and invites drift in which error maps to which code.
3. **Push classification back into `apiclient`** — rejected: `apiclient` must not import `internal/cli` (010 DECISIONS); the `Outcome` enum + exit codes are `cli`'s, and the producer/consumer split is the whole point.

**Decision**: Option 1. Extend the `Outcome` enum (`dispatch.go`) with `NetworkUnavailable` and `APIError`; add their cases to `ExitCode` (`exitcode.go`) taking reserved codes 6 and 3. A new `classifyClientError(err) Outcome` (a small `internal/cli` helper) is the single `errors.As` chain mapping 010's typed errors → `Outcome`; `runMe` calls it, and 012–017 reuse it. `ExitCode` stays a pure mapper (it never inspects the error) — the classifier produces the category, `ExitCode` maps it.

**Consequences**: The codes 004 reserved get their producer, at the one site, reused by the whole read surface. `ExitCode`'s `default → codeInternalError(1)` fail-safe still backstops any future category. Because the classifier is table-shaped, its tests follow the PR #10 LEARNINGS guard (assert each typed error maps to its expected `Outcome`, with a `len`+comma-ok-style exhaustiveness check so a dropped mapping fails loud, not silently). *Precedent-setting: the read surface classifies 010's typed client errors through one `internal/cli` helper into the `Outcome` enum; new categories are added once, at the registry, never inlined per command.*

### ADR-4: The error→code mapping reuses existing categories where the semantics already exist; only transport and generic non-2xx take new reserved codes (resolves spec fork 3)

**Context**: 004's reserved codes are 3 (general API error), 4 (permission), 5 (rate-limit), 6 (network-unavailable). The spec fixed (fork 3) that `me` surfaces 010's *generic* outcome and maps via 004 — **no per-status interpretation** (401/403→permission, 429→rate-limit is API Error Extraction 015 / Rate-Limit Handling 017, not built yet). DECISIONS repeatedly flag that 004 has no local "cannot authenticate" or "bad configuration" code — "the consuming-command spec must decide." 011 decides.

**Options considered**:
1. **Reuse existing categories for the local/precondition failures; take reserved codes only for the two outcomes 011 genuinely produces now** — `*TransportError`→`NetworkUnavailable`(6); generic non-2xx `*ResponseError`→`APIError`(3, the "general API error" bucket); `*AuthError{NoCredentials}`→`UsageError`(2) ("not authenticated — run `glassfrog auth login`"); `*AuthError{CredentialError}`→`RuntimeError`(1) (malformed `.glassfrogrc`); `*DecodeError`→`RuntimeError`(1) (the API returned 2xx but the body didn't parse — an internal/contract failure); base-URL error→`UsageError`(2) (the operator supplied a bad endpoint via flag/env/file — a correctable input).
2. **Invent finer codes now** (a dedicated "not authenticated" code, a "bad config" code) — rejected: 004's convention is frozen and codes 3–6 are spec-assigned; adding a code is a 004 change, and the spec deferred finer API-side classification to 015/017. Reusing 1/2 for the local preconditions avoids premature codes while staying honest.
3. **Map generic non-2xx to RuntimeError(1)** — rejected: a non-2xx is the *API* answering, not an internal crash; code 3 ("general API error") is exactly its bucket, and mapping it there lets 015 later split 401/403→4 and 429→5 **without renumbering** (3 stays the residual general-API bucket).

**Decision**: Option 1, with the mapping table above. `NoCredentials` → `UsageError` mirrors `runLogin`'s "no token to store" → `UsageError`; `CredentialError`/`DecodeError` → `RuntimeError` mirror `runLogin`'s `FormatError` → `RuntimeError`. Only `NetworkUnavailable`(6) and `APIError`(3) are new.

**Consequences**: Minimal new surface, no premature codes, forward-compatible with 015/017 (they add `Outcome` cases that split `APIError` by status; code 3 is never renumbered). The "not authenticated" path gives the operator an actionable message at code 2. *Precedent-setting: local/precondition client failures (no credential, broken cred file, undecodable body, bad base URL) reuse `UsageError`/`RuntimeError`; reserved codes 3–6 are for genuine API-side / network outcomes; 015/017 refine `APIError` later without renumbering.*

### ADR-5: A guard-registered cobra leaf with an injected seam, a pure `runMe`, a pure `formatMe` projection renderer, and request-time `--include` validation

**Context**: CONSTITUTION IV requires hermetic, RED-first tests with no real network; the spec requires a reshaped projection (not JSON — `--output json` deferred) and rejecting an unsupported `--include` target *before* any request (fail-fast usage error, the 002 invalid-input convention). 006's `runLogin`/`loginSeam` established the pattern: a thin cobra `RunE` over an injected seam, delegating to a pure `run*` over plain injected values.

**Options considered**:
1. **`newMeCommand(seam)` + pure `runMe(cfg)` + pure `formatMe(MeResponse) string` + pure `validateInclude([]string) error`** — the seam binds `AssembleFromOS`/`NewClientFromOS` (and the injected base `http.RoundTripper`) in production and fakes in tests, so every branch (success-decode, non-2xx, transport, no-token, decode-fail, base-URL-error, include-rejection, roles-embed, empty-roles) runs offline. `formatMe` is unit-tested in isolation; `validateInclude` runs before context assembly.
2. **A monolithic `RunE` that calls the real OS seams directly** — rejected: not hermetically testable (hits the real network/filesystem/`~/.glassfrogrc`), violating CONSTITUTION IV and the 005/006 injected-seam precedent.

**Decision**: Option 1. The seam shape mirrors `loginSeam`. To inject a fake transport, the seam exposes client construction over an injected base `http.RoundTripper` (production binds the real one via `NewClientFromOS`; tests bind a fake returning canned `GET /me` responses). `validateInclude` checks the flag values against the spec's `include` set (today `{roles}`) and returns a usage error naming the unsupported target before any assembly/request. `formatMe` renders the projection (actor id/name/kind, organization id/name, membership access level; a roles section listing id+name when embedded, omitted when none).

**Consequences**: Every outcome is exercised offline; `me` never touches the real network or the developer's `~/.glassfrogrc` in tests. The exact seam interface, `--include` flag spelling (value vs. repeatable), `Request`/`MeResponse` type names, and projection layout (labels, ordering) are interface-level. *Mild precedent: each read command pairs a pure `run<Read>` + a pure `format<Read>` projection renderer behind an injected seam; the projection is the agent-legible default, with `--output json` a future cross-cutting flag.*

---

## Data Model Design

`internal/glassfrog` (schema only; JSON-tagged structs decoded from the API; field names/tags pinned at `/score:interface`):

- **`MeResponse`** — the `GET /me` body: `Actor`, `Organization`, `Membership`, and `Roles []Role` (present only when `?include=roles` was requested; empty/absent otherwise).
- **`Actor`** — `ID` (`per_`/`agt_` prefix — the machine-actionable handle), `Name`, `Kind` (`human`|`agent`). (`created_at`/`updated_at` decoded but not projected.)
- **`Organization`** — `ID` (`org_`), `Name`.
- **`Membership`** — `AccessLevel` (`admin`|`normal`); other fields (`id`, `actor_id`, `organization_id`) decoded, only access level projected.
- **`Role`** — minimal `ID` (`role_`/`circle_`), `Name` for the embed. The full role shape (accountabilities, domains, …) is **My Roles (012)'s** growth of this same type — 011 defines only what the embed projects, and 012 extends it without a second type.

Decoding is tolerant of unknown/extra fields (forward-compatible with API additions); only the projected fields are required to be present for a successful render. The token is never a field here (it is a request header, not a response field) — secret hygiene holds by construction.

---

## Integration Design

- **Request Execution (010, `internal/apiclient` — upstream dependency, landed on main #30)**: `me` calls `NewClientFromOS(ctx)` then `(*Client).Execute(reqCtx, Request{Method:"GET", Path:"/me", Query:{include?}}, &me)`. 011 consumes 010's `Client`/`Execute`/`Request`/`Response` and the typed errors **unchanged**; it adds nothing to `apiclient`. *010's `apiclient` code is on main; building 011 from main means Phase 3 compiles directly against the real seam — see Implementation Strategy and Risks.*
- **Connection Context Assembly (009) / Base URL Resolution (008) / Request Authentication (007) — transitive**: `AssembleFromOS(baseURLFlag)` resolves the context once; the token reaches the wire only through 007's `AuthTransport` (010's replay thunk). `me` reads the `--base-url` flag value and passes it in; it re-resolves nothing and never reads the token.
- **Glassfrog API `GET /me` (system actor)**: returns `MeResponse` on `200` (+ embedded `roles` when `?include=roles`); `401` for an unusable token, `404` per the spec. `me` decodes the 200 body into `glassfrog.MeResponse`; non-200 arrives as 010's generic `*ResponseError`.
- **Exit-Code Convention (004, `internal/cli` — extended here)**: `me` adds `NetworkUnavailable`/`APIError` to the `Outcome` enum and their `ExitCode` cases (ADR-3/4), and classifies 010's typed errors through the shared `classifyClientError`. `apiclient` still does not import `internal/cli`.
- **My Roles (012 — downstream sibling)**: reuses `glassfrog.Role` (grows it to the full shape) and the shared `classifyClientError`; `me --include roles` is the API's convenience embed, not a second roles surface.
- **API Error Extraction (015) / Rate-Limit Handling (017 — downstream siblings)**: later refine `APIError`(3) by splitting 401/403→permission(4) and 429→rate-limit(5) as new `Outcome` cases at the same registry — without renumbering code 3.

---

## Cross-cutting Concerns

**Secret hygiene (CONSTITUTION II)**: `me` **never reads `ctx.Cred.Token`** — the token's only path is 010's replay thunk into 007's `AuthTransport`. The projection (`formatMe`) renders **response-side** fields only (actor/org/membership from the API's reply); the `X-Auth-Token` *request* header is not echoed in any response field. The classifier's messages name outcomes/paths, never the token (mirrors 007/009/010). Pinned by a test asserting no `me` output or error renders the token across success and every error branch, and that `me`'s code never references `ctx.Cred.Token`.

**Error handling (CONSTITUTION III)**: fail loud at every fork — an unsupported `--include` is refused before any request; a base-URL error refuses at `NewClientFromOS` (no doomed send); a wire failure is `NetworkUnavailable`(6); a non-2xx is never treated as success (generic `APIError`(3)); a 2xx body that won't decode is a loud `RuntimeError`(1), never a zero-valued projection. Every outcome maps through the single `ExitCode` registry; the `default→1` fail-safe backstops any unmapped category.

**Testing (CONSTITUTION IV)**: RED-first, hermetic, no real network or real `~/.glassfrogrc`. The injected seam (ADR-5) binds a fake base `http.RoundTripper` returning canned `GET /me` responses, so every branch runs offline: 200 decoded → projection; 200 with `?include=roles` → roles listed; 200 roles-requested-but-empty → roles section omitted; non-2xx → `APIError`(3); transport error → `NetworkUnavailable`(6); no-token context → `*AuthError{NoCredentials}` → `UsageError`(2); broken-cred context → `RuntimeError`(1); undecodable 200 → `*DecodeError` → `RuntimeError`(1); base-URL-error context → `UsageError`(2); unsupported `--include` → `UsageError`(2) with no request issued (a tripwire fake asserts the transport was never called). `formatMe` and `validateInclude` are unit-tested pure. `classifyClientError` gets a table test with an exhaustiveness guard (PR #10 LEARNINGS — `len`+comma-ok so a dropped mapping fails loud). The driving scenarios become a **new `internal/cli` godog suite** (the `credstorage_bdd_test.go` pattern) pointed at its **own** feature file `features/self-service-reads/identity-read.feature` (LEARNINGS — never the whole `features/` dir); step helpers return errors, never panic; stderr/stdout capture uses a temp file, not `os.Pipe` (PR #10 LEARNINGS).

**Help/version regression (003)**: the new persistent `--base-url` root flag changes root help output (a global-flags section appears). The 003 help-and-version tests that assert root help content are updated in the same change; the flag is optional documentation data, so the narrowed 003 non-behavior is not violated (ADR-2).

---

## Implementation Strategy

Four phases, mostly linear. **Upstream dependency (satisfied)**: 010 (`NewClientFromOS`, `Execute`, `Request`, `Response`, `*TransportError`/`*ResponseError`/`*DecodeError`, propagated `*AuthError`) is **landed on main (#30)**, so Phase 3 compiles against the real `apiclient` seam as long as 011 is built from current main. Phases 1–2 have **no** 010 dependency and can proceed independently in any case.

- **Phase 1 — `internal/glassfrog` schema package**: define `MeResponse`, `Actor`, `Organization`, `Membership`, `Role` (minimal) with JSON tags; tolerant decode of unknown fields. RED-first unit tests: a `GET /me` fixture decodes into the structs (with and without the `roles` embed); extra/unknown fields are ignored. *Depends on: nothing (leaf schema).*
- **Phase 2 — `Outcome`/`ExitCode` extension + `classifyClientError` + persistent `--base-url` flag**: add `NetworkUnavailable`/`APIError` to the `Outcome` enum and `ExitCode` cases (codes 6/3); add the shared `classifyClientError(err) Outcome` (`errors.As` chain over 010's typed error types, now on main — T002 references the types, not 010's runtime behavior, so it stays unit-testable in isolation); register `--base-url` as a persistent root flag in the wiring layer and update the affected 003 help tests + the exit-code pin tests. RED-first: classifier table test with exhaustiveness guard; `ExitCode` pins for the two new codes; root-help reflects the new global flag. *Depends on: 010's error type names (interface-spec); does not need 010's behavior.*
- **Phase 3 — `me` command: `newMeCommand` + `runMe` + `formatMe` + `validateInclude` + seam**: the cobra leaf, the injected seam (prod binds `AssembleFromOS`/`NewClientFromOS`), the pure `runMe` orchestration (validate include → assemble → build client → execute → render/classify), the pure `formatMe` projection, and `validateInclude`. RED-first unit tests over the fake-transport seam for every branch (Cross-cutting / Testing). *Depends on: Phase 1 (decode target), Phase 2 (classifier + flag), and 010 implemented (compiles against `Execute`).*
- **Phase 4 — wiring + executable acceptance**: one `MustRegister(root, newMeCommand(productionSeam{}))` line in `Assemble()`; godog step definitions for the driving scenarios in the new `internal/cli` suite pointed at `features/self-service-reads/identity-read.feature`. *Depends on: Phase 3.*

---

## Risks

- **011 built from a base that predates 010** (low likelihood, medium impact): Phase 3 cannot compile against `apiclient.Execute` unless 010's code is present. *Mitigation*: 010 is **landed on main (#30)** — building 011 from current main satisfies the dependency; this spec's branch is already rebased on it. Phases 1–2 (schema, exit-code extension, classifier, flag) have no 010 dependency regardless. The tasks dependency graph keeps the relationship explicit so the Builder cuts the 011 base from a 010-bearing main.
- **Token leak through the projection or an error** (low likelihood, high impact): `me` sits on the read path. *Mitigation*: `me` never reads `ctx.Cred.Token`; `formatMe` renders response-side fields only; classifier messages are path/outcome-only; pinned by a token-never-in-output test across success + every error branch (CONSTITUTION II).
- **Generic non-2xx → code 3 re-litigated by 015/017** (medium likelihood, low impact): mapping every non-2xx to `APIError`(3) is coarse until 015/017 split 401/403→4 and 429→5. *Mitigation*: code 3 is the residual "general API error" bucket by design (004); 015/017 add finer `Outcome` cases at the same registry **without renumbering** 3. Forward-compatible (ADR-4).
- **Persistent root flag destabilizes 003 help tests / dispatch** (medium likelihood, low impact): adding `--base-url` to the root changes help output and the global flag set. *Mitigation*: update the affected 003 tests in the same change; the flag is optional documentation data (003's narrowed non-behavior permits it); cobra handles persistent-flag inheritance, so dispatch/exact-match behavior is unaffected (pin with the existing help regression tests).
- **`--include` validation drifts from the spec enum** (low likelihood, low impact): hard-coding `{roles}` could lag a future spec addition. *Mitigation*: `validateInclude` checks against a named set sourced from the spec's `include` enum; adding a target is a one-line change tracking the spec (an `[ASSUMED]`, planning-adjustable detail, not a behavior change).

---

## What This Plan Does Not Cover

- **`--output json` (structured/verbatim output)** — a future cross-cutting capability the spec deferred; `me` ships the reshaped projection only (a Non-Behavior).
- **Per-status API error interpretation** (401/403→permission, 429→rate-limit, error-detail extraction) — API Error Extraction (015) and Rate-Limit Handling (017), which add finer `Outcome` cases that split `APIError`(3) without renumbering.
- **Pagination** — `/me` is a single resource; Pagination (016) serves the list reads (012–014).
- **The full `Role` shape** (accountabilities, domains, assignments) — My Roles (012) grows `glassfrog.Role`; `me --include roles` projects only id+name.
- **The exact Go API shapes** — the seam interface, `--include` flag spelling, `Request`/`MeResponse`/`Role` struct fields and JSON tags, and the projection layout — `/score:interface` pins these (the CLI + specification boundaries).
- **Executable Gherkin** — `/score:scenarios` turns the driving scenarios into `features/self-service-reads/identity-read.feature`.
- **010's `apiclient` implementation** — a separate spec (010) already designed through its pre-implementation guard; 011 consumes it unchanged.
