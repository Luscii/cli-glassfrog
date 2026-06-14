# Tasks: Actor Read

**Feature**: 049-actor-read
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/actors-disconnected-from-governance/actor-read.feature

---

## Dependency Graph

Phase 1: `internal/glassfrog` `ActorDetail` type (1 task, no phase dependencies) [Shared]
Phase 2: `internal/render` `actor` single-detail key (1 task, depends on Phase 1) [Shared]
Phase 3: Grow the `actors` command for the single read (1 task, depends on Phase 2 + 048's `actors` command) [Shared]
Phase 4: Executable acceptance (1 task, depends on Phase 3) [Shared]

4 tasks total | T001 startable immediately | linear chain (T001 → T002 → T003 → T004) | Builder: pipeline

> Plan-faithful: the plan's three phases map here as schema (T001 — `ActorDetail`), render (T002 — the `actor` single-detail key), and command (T003 — growing `actors` for the single read), plus the executable-acceptance task (T004) the plan's Cross-cutting/testing section calls for. The result is a clean linear chain: T002 needs T001's `ActorDetail` (the render view embeds the footprint), T003 needs T002's `actor` render key, and T004 needs T003's command. Only T001 is startable immediately. Story labels: the spec's three user scenarios are US1 (see what an actor does — their governance footprint, `--include roles`), US2 (drill into an actor by id, `per_`/`agt_`), and US3 (tell a missing actor apart from a failed network). The single grown `actors` command (T003) serves all three, so T003 is `[Shared]`; T001/T002/T004 are `[Shared]` infrastructure.
>
> **T003 grows 048's landed `actors` command.** It widens `internal/cli/actors.go` from `cobra.NoArgs` to `cobra.MaximumNArgs(1)`, branching `RunE` on `len(args)` (plan ADR-1, an announced divergence from 048 ADR-1). **048 is merged on `main`** (`feat(048): Actor Directory — glassfrog actors` #90 — `actors.go` ships with `cobra.NoArgs` + `runActorsList`), so this is an edit against current `main`, not a cross-spec race. **All cross-spec dependencies are landed on main** — 007/009/010/011/015/018/019/020/025/048 with their packages shipped — `internal/glassfrog.Actor` (id/name/kind/timestamps, 011), the full `Role`/`Assignment`/`RoleDetail` (025, `roles.go`), `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the persistent `--base-url`/`--output`/`-o`, the single-resource render dispatch + the singular-detail render keys (`ResourceRole`/`ResourceDomain`/`ResourcePolicy`, 025/033/034) over `internal/output`/`internal/render` (018/019/020), and the reject-unknown local-validator shape (`internal/cli/include.go`).
>
> **Existing main state 049 builds on**: there is **no** `glassfrog.ActorDetail` (T001 adds it — reusing the landed `Actor`/`Role`/`Assignment`), **no** `actor` (singular) render key (T002 adds it), and `actors` is a `cobra.NoArgs` directory list (048; T003 grows it to `MaximumNArgs(1)`). 049 adds a **new `--include` validator** over `{roles, assignments}` but **no** new `Outcome` category, `ExitCode` case, generic type, or root flag. There is **no pagination** — `GET /actors/{id}` is a single resource (plan Cross-cutting).
>
> **The single-actor drill-in of the Actor Reads slice.** `actors <id>` reads `GET /actors/{id}` (the ungated unified endpoint); the directory list is 048, and Actor Assignments (050, the standalone roles-an-actor-fills list) is its own spec — the optional-positional shape forecloses `actors <id> assignments`, so 050 takes a flag/separate surface (plan ADR-1).

---

## Branching Guidance

**Pipeline mode**: `spec/049-actor-read/base` → `spec/049-actor-read/task-1`, `…/task-2`, `…/task-3`, `…/task-4` (one task branch per T-id, merged back into the spec base). Cut the base from current `main` — it already carries 048's `actors` command (#90), which T003 grows. T001/T002 touch only `internal/glassfrog` and `internal/render`.

**Role-based awareness**: parallel Conductor workspaces may carry sibling reads (the actor-reads wave: 048 Actor Directory, 050 Actor Assignments, 047 Role Fillers). 049 touches `internal/glassfrog` (new `ActorDetail`, reusing `Actor`/`Role`/`Assignment` — no growth of the leaf types), `internal/render` (one new singular `actor` key), and `internal/cli` (grows `actors.go` + a new `--include` validator). The only cross-package contract to coordinate is the `actors` command growth shared with 048 (ADR-1). 050 will reuse `Assignment`/`ActorDetail` likewise (additive).

---

## Phase 1: `internal/glassfrog` `ActorDetail` type [Shared]

- [x] **T001** [Shared] Add the `ActorDetail` decode type embedding `Actor` + optional `Roles`/`Assignments` — unit-test the `{data: ActorDetail}` decode; `internal/glassfrog` (new `ActorDetail`) + decode tests — 3 decode tests (bare/roles/assignments), added `ActorDocument` alias
  - **Scope**: In `internal/glassfrog`, add `ActorDetail struct { Actor; Roles []Role; Assignments []Assignment }` — embedding the **unchanged** landed `Actor` (11, `me.go`) and adding two optional embed slices that decode from the `roles`/`assignments` JSON keys (`omitempty`/nil unless `?include`d). Reuse the **full landed** `Role` and `Assignment` (025, `roles.go`) verbatim — define **no** new leaf models and **do not** grow `Actor`/`Role`/`Assignment`. The single read decodes the `{data: ActorDetail}` document wrapper (one object — **not** the `Page[Actor]` list envelope the directory uses). Decoding stays tolerant of unknown/extra fields (011). Schema only — no transport, no cobra, no exit codes; the package keeps importing neither `cli` nor `apiclient`.
  - **Acceptance criteria**:
    - `ActorDetail` decodes a bare actor (`{data:{id,name,kind,…}}`) with `Roles`/`Assignments` nil/empty
    - `ActorDetail` decodes an actor with `roles` embedded — each `Role` carries its full shape (purpose, accountabilities, domains)
    - `ActorDetail` decodes an actor with `assignments` embedded — reusing the landed `Assignment` shape
    - `Actor`, `Role`, and `Assignment` are unchanged (byte-stable); `go build`/`go vet` clean; `internal/glassfrog` imports neither `cli` nor `apiclient`
  - **Dependencies**: None (embeds/reuses the landed `Actor`/`Role`/`Assignment`).
  - **Plan reference**: Phase 1 (Schema); ADR-2 (`ActorDetail` embeds `Actor` + optional `Roles`/`Assignments`, the `RoleDetail` shape; reuse the full `Role`/`Assignment`)
  - **Interface references**: interface-cli.md — Consistency Notes (`ActorDetail`, no leaf-type growth)
  - **Scenario references**: actor-read.feature: "A roles include embeds the actor's governance footprint", "An assignments include embeds the actor's assignments"
  - **Risk**: ⚠️ Embed `Actor`, do not duplicate or grow it (011 ADR-1). ⚠️ Reuse the **full** landed `Role`/`Assignment` (025) — define no second model. ⚠️ The single read decodes a `{data: ActorDetail}` document, not a `Page` envelope.

## Phase 2: `internal/render` `actor` single-detail key [Shared]

- [x] **T002** [Shared] Add the singular `actor` render key + view + templates — golden + registry-guard tests; `internal/render` (new `actor.{full,compact}.tmpl`, `ActorDetailView`, `ResourceActor`) + tests — 9 golden tests, registry guard passes with the new key
  - **Scope**: In `internal/render`, add **one new** singular render key `actor` (the single-actor + footprint detail over `ActorDetail`, via an `ActorDetailView` mirroring the landed singular-detail views). Add the `ResourceActor` constant to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates: `actor.full` — the actor's `<per_…|agt_…>  [<kind>]` line and an indented `Name: <name>`, then a `Roles:` section (present only when embedded) listing each role's name/purpose/accountabilities/domains, and an `Assignments:` section (present only when embedded); `actor.compact` — one line: `<per_…|agt_…>  [<kind>]  <name>` with `roles=<n>`/`assignments=<n>` counts when the embeds are present. Each embed section and each nullable field uses 019's explicit-absence guard (`(no purpose set)`/`(none)` etc.) — rendered only when `?include`d, never inventing a value. Render `name`/`purpose` verbatim — never truncated or reflowed (CONSTITUTION VI). The footprint may reuse the `role` template's accountability/domain fragments or define an actor-framed layout. `ResourceMe` (one actor inside the `me` document) and `ResourceActors` (048's flat list) are **not** reused — touch no existing key or template. Depends on `internal/glassfrog` (`ActorDetail`) + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `actor` key renders both `full` and `compact`; the identity line shows the `per_`/`agt_` id, the `kind` badge, and the name
    - With roles embedded, `full` prints each role's name, purpose, accountabilities, and domains; an embedded role with no purpose renders the absence marker, not a blank
    - With assignments embedded, `full` prints the assignments; with no embed requested, the `Roles:`/`Assignments:` sections are omitted (not printed empty)
    - The registry-exhaustiveness guard passes with the new `actor` key carrying both formats; golden tests pin each template (bare actor; roles footprint; assignments; absent-embed)
    - `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: T001 (`ActorDetail`).
  - **Plan reference**: Phase 2 (Render); ADR-4 (new singular `ResourceActor` key; `ResourceMe`/`ResourceActors` not reused)
  - **Interface references**: interface-cli.md — Output (human `full`/`compact` shapes, footprint sections, absence guards), Consistency Notes (singular render key)
  - **Scenario references**: actor-read.feature: "A roles include embeds the actor's governance footprint", "An assignments include embeds the actor's assignments", "Default output carries no raw API envelope"
  - **Risk**: ⚠️ Add the new singular `actor` key only — do not reuse or fork `ResourceMe`/`ResourceActors`. ⚠️ Explicit-absence/omit handling for the optional embeds; never invent a value (019). ⚠️ Add `ResourceActor` to the guarded set so exhaustiveness still holds. ⚠️ Never truncate/reflow `name`/`purpose` (CONSTITUTION VI).

## Phase 3: Grow the `actors` command for the single read [Shared]

- [x] **T003** [Shared] Grow `actors` to `MaximumNArgs(1)` + the single-read branch + `--include` validator + mode-separation guards — RED-first unit tests for every branch; grows `internal/cli/actors.go` (+ the `--include` validator) — `runActors` dispatcher + `runActorRead`, `validateActorInclude`/`validateActorsFlags`; 048 directory branch preserved
  - **Scope**: Grow `internal/cli/actors.go` (048). Change `Args` from `cobra.NoArgs` to **`cobra.MaximumNArgs(1)`** and branch `RunE` on `len(args)`: **0 args → the existing directory list** (048 `runActorsList`, preserved verbatim), **1 arg → a new `runActorRead`**. Declare `--include` (single-only). Add the **`--include` reject-unknown validator** against `{roles, assignments}` — a per-read set distinct from the role include set (a thin `validateActorInclude` or a parameterization of the landed `include.go` validator; do not regress the existing `--include` message for current callers — plan ADR-3). Add the **mode-separation guards**: a list filter (`--kind`/`--role-id`/`--query`/`--first-page`/`--per-page`) supplied **with** an id → `UsageError(2)` (filters are list-only); `--include` supplied **without** an id → `UsageError(2)` (`--include` is single-only). `runActorRead(cfg) (Outcome, error)`: **resolve `--output` first (020), then the mode-separation guards, then validate `--include`** — all pure and **before** assembly, so a bad format / misplaced flag / bad include is a fail-fast `UsageError(2)` with **no request** (transport tripwire); pass the actor **id through unvalidated** on the path (plan ADR-3 — `getActor` answers a bad id with a clean `404`). Build the `include` query parameter only when `--include` is `Changed()` and non-empty. Issue **exactly one** `Execute` of `GET /actors/{id}?include=…` (via the shared seam over the `RetryExecutor`-wrapped `*Client`) into a `{data: ActorDetail}` target — **no walk, no `paging.All`, no `Page[T]`** (single resource). For `json`/`yaml`, decode the 2xx body into `json.RawMessage` and emit the `{data:…}` document via `output.RenderSuccess` (the single-resource raw-bytes path, **not** `aggregateRawData`); for `full`/`compact`, render `ResourceActor` over an `ActorDetailView`. Route failures through the shared `classifyClientError` (a `404` → `APIError(3)`; `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)`; transport → `NetworkUnavailable(6)`). Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog actors per_abc` issues exactly one `GET /actors/per_abc` and prints the actor's id/name/kind; exits 0 (no page walk, no cursor followed)
    - `glassfrog actors agt_def` reads `GET /actors/agt_def` (the ungated unified endpoint) — never the `ai_integration`-gated `/agents` alias — even with a token lacking `ai_integration`
    - `--include roles` sends `include=roles` and renders the footprint; `--include assignments` sends `include=assignments` and embeds the assignments; `--include roles,assignments` sends both
    - `--include nonsense` is a `UsageError(2)` naming the value + supported set (`assignments, roles`, sorted), **no request sent** (transport tripwire)
    - A list filter with an id (`actors per_abc --kind human`), and `--include` with no id (`actors --include roles`), are each a `UsageError(2)` with **no request sent**
    - `glassfrog actors` with **no id** still lists the directory and exits 0 (048 behavior preserved)
    - An unknown id answered `404` → the read-failed outcome naming the HTTP status, exits non-zero (API-error code); `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); base-URL/`--output` errors → UsageError(2)
    - `-o json`/`yaml` emit the single `{data:…}` document (raw bytes, not re-encoded), `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T002 (the `actor` render key), T001 (`ActorDetail`), and 048's `actors` command (grown, not created). Reuses `classifyClientError` (011/015), the single-resource render dispatch (`renderResult`-style), and the reject-unknown validator shape (`include.go`).
  - **Plan reference**: Phase 3 (Command); ADR-1 (grow `actors` to `MaximumNArgs(1)`; 0 args → directory, 1 arg → single read), ADR-3 (`--include` validated locally; id passed through), ADR-2 (`{data: ActorDetail}` decode); Cross-cutting (no pagination)
  - **Interface references**: interface-cli.md — `actors <id>` Surface, `--include`, mode-separation table, Output, Interactions (validation order, single-request no-walk), Error Communication
  - **Scenario references**: actor-read.feature: "An id reads a single actor", "An agt_ id reads an agent", "An agent read reaches the ungated unified endpoint", "A roles include embeds the actor's governance footprint", "An assignments include embeds the actor's assignments", "An unsupported include is rejected as a usage error", "A footprint include with no id is rejected", "A list filter combined with an id is rejected", "The command with no id still lists the directory", "A missing token fails as a not-authenticated usage error", "An unknown id fails with the API status"
  - **Risk**: ⚠️ `MaximumNArgs(1)` (not `ExactArgs(1)`) — 0 args must still reach the 048 directory branch unchanged; a godog scenario asserts `actors` (no id) still lists. ⚠️ Preserve 048's `runActorsList` verbatim — branch, don't rewrite. ⚠️ Issue exactly ONE `Execute` into `{data: ActorDetail}` — no `paging.All`/`Page[T]`/walk, even with large embedded arrays. ⚠️ Structured output is the single `{data:…}` raw-bytes document (`output.RenderSuccess`), NOT `aggregateRawData` and NOT a decode-and-re-encode of `ActorDetail`. ⚠️ Resolve `--output` first, then mode-separation guards, then validate `--include`, BEFORE assembly, so the tripwire confirms no request on rejection. ⚠️ Validate `--include` against `{roles, assignments}` as a per-read set; pass the id through (025 ADR-4). ⚠️ If parameterizing the landed validator, keep its existing message intact for current callers. ⚠️ Reuse `classifyClientError` — no second `errors.As` chain. ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`.

## Phase 4: Executable acceptance [Shared]

- [x] **T004** [Shared] Make the driving scenarios pass as executable acceptance — new/extended `internal/cli` godog suite over `actor-read.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held — `TestActorReadFeatures` (11 scenarios pass), 4 `@validation` scenarios held `@wip`
  - **Scope**: Add godog step definitions for `features/actors-disconnected-from-governance/actor-read.feature` in a godog suite (e.g. `TestActorReadFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `actors <id>` through the shared seam over a fake base `http.RoundTripper` returning canned `GET /actors/{id}` responses (bare actor; roles-embedded; assignments-embedded; `404`), plus a transport tripwire for the no-request paths (unsupported `--include`; list-filter-with-id; `--include`-without-id) and the single-request/no-walk and ungated-`/actors`-not-`/agents` assertions. Include the `actors` (no id) directory scenario to prove the grown command still lists. Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection / param-assertion / tripwire phrasings from the `me*`/`roles`/`actors`(048) suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` actor-read scenario has an executable, passing path; `@wip` removed from them
    - The `@validation` scenarios keep `@wip`
    - The suite's `Paths` names only `actor-read.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - The single-request/no-walk and ungated-endpoint behaviors are genuinely exercised by the fakes (one recorded request to `/actors/{id}`; no `/agents` request)
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (the grown `actors` command — all behavioral scenarios must be implementable)
  - **Plan reference**: Phase 3 (Command, the single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: actor-read.feature: all behavioral Rule-block scenarios (the `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `actor-read.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings (incl. 048's `actors` bindings) before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the `404`, roles/assignments-embedded, no-request (tripwire), single-request-no-walk, ungated-endpoint, and directory-still-lists fakes so the footprint, drill-in, mode-separation, and 048-preservation scenarios genuinely exercise their paths. ⚠️ Capture stdout/stderr with a temp file, not `os.Pipe` (PR #10 LEARNINGS).
