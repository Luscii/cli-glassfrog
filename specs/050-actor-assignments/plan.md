# Plan: Actor Assignments

**Feature**: 050-actor-assignments
**Role**: Shaper
**Inputs**: spec.md (050), PROJECT.md, DECISIONS.md, LEARNINGS.md (background), DEPRECATION.md

---

## System Architecture

Actor Assignments adds the **actor-scoped assignment read** to the Actor Reads slice — the exact mirror of Role Fillers (047), reading the one `Assignment` resource from the actor end. Architecturally it is a **single** thin cobra command in `internal/cli` that builds a request, hands it to the proven read chain, walks the pages, and renders the result. It introduces **no new package** and **no new transport, pagination, error, or output machinery** — every seam it needs is landed. It is a near-exact sibling of the *list half* of Role Projects (038) / Role Policies (034), and the structural twin of `fillers` (047), minus the singular standalone read those features pair with (the API exposes no `GET /assignments/{id}` — see ADR-1).

One command, `ExactArgs(1)`, guard-registered (001) and explicitly wired in `main`:

- **`glassfrog assignments <actor-id>`** → `GET /actors/{actor_id}/assignments` (`listActorAssignments`) — a **paginated list** of the assignments an actor holds. Each assignment carries the embedded **role** (the endpoint's default `?include=role`, so `{id, type, name, purpose, parent_role_id}` arrives without a flag) plus the assignment's own `focus` and `elected_until`. Walks to completion through `paging.All` (016) by default, with the shared `--first-page` opt-out. It offers **no filter flags and no `--include`** (the endpoint accepts neither beyond the default include + pagination — ADR-3).

Data flow per invocation (identical to 038/043/047/048's list path): the command resolves the connection context once (`AssembleFromOS`), builds the `*apiclient.Client` (010), and walks the list — for `json`/`yaml` reading `glassfrog.Page[json.RawMessage]` and aggregating each assignment's raw bytes into a `{data:[…]}` document via `aggregateRawData` (per-page `meta` dropped), for `full`/`compact` reading `glassfrog.Page[glassfrog.Assignment]` and rendering the new `assignments` projection (019) through `writeHuman`. The actor id is sent into the path unvalidated; a bad id surfaces as the API's clean non-2xx (`404`), classified by the one shared `classifyClientError` chain (011/015). It adds **no new `Outcome` category and no new exit code**.

The genuinely new artifacts are the **`assignments` command**, an **`assignments` list render path** (`render.AssignmentsView{Data []glassfrog.Assignment}`, the `render.ResourceAssignments` registry entry, and the `assignments.full`/`assignments.compact` templates — mirroring 048's `actors` render-key addition), and a **small additive growth of `glassfrog.Assignment`** to carry the embedded `role` object the actor-end default include returns (ADR-2). The whole read chain (`paging.All`, `aggregateRawData`, `writeHuman`, the connection seam) is reused as-is.

---

## Architecture Decisions

### ADR-1: Expose a single top-level `assignments <actor-id>` command — no plural/singular pair

**Context**: The per-id-read surface precedent (034 ADR-1, followed by 033/038) is **two** sibling top-level commands — a plural list and a singular standalone read keyed on different id kinds. Like Role Fillers (047 ADR-1), Actor Assignments has only **one** read worth surfacing: the API's `/assignments/{id}` path carries only the administrative `PATCH`/`DELETE` (no `GET`), and assignment administration is out of scope per PROJECT.md. So there is no singular standalone read to pair with the list. This is the actor-end mirror of 047, which DECISIONS (047 ADR-1) anticipated: "Actor Assignments (050) reads the same `Assignment` resource from the actor end … and faces the same single-read-absence."

**Options considered**:
1. **Reuse the plural/singular pair anyway** (`assignments <actor-id>` + a singular `assignment <asgn-id>`). Rejected: there is no `GET /assignments/{id}`, so the singular command would have no endpoint to call — inventing one would fabricate API surface, violating "Spec is the contract" (PROJECT.md).
2. **An `actor <actor-id> assignments` group.** Rejected for the same reason 034/033/047 rejected a per-id group: it forces sibling actor-scoped reads to land a shared `actor` group in lockstep; the slice deliberately did not create one (the actor surface is `actors`, 048, flag-only discovery — no group).
3. **A single top-level `assignments <actor-id>` command, `ExactArgs(1)`.** Chosen — the actor-scoped list, standing alone, structurally identical to `fillers` (047).

**Decision**: Option 3. `assignments` is one guard-registered, explicitly-wired command taking a required positional actor id (`cobra.ExactArgs(1)` — omitted/extra positional is a `UsageError(2)` from cobra's own validator, no hand-rolled guard). Because there are no list-only filter flags to register (ADR-3), there is no cross-combo guard. It inherits the `--base-url` and `--output`/`-o` persistent root flags (011/020) plus the shared `--first-page`/`--per-page` walk flags.

**Consequences**: The second read in the slice with no singular sibling (after 047) — endpoint-driven, not a deviation from the 034/038 pattern. The command name `assignments` is the practitioner-facing noun paired with 047's `fillers`; the model/endpoint stay the "assignment" resource. No `actor` group is created. Adds no new `Outcome`/`ExitCode`.

### ADR-2: Grow `glassfrog.Assignment` to carry the embedded `role` object — additive, plus a new `assignments` list render path

**Context**: The `glassfrog.Assignment` model (in `internal/glassfrog/roles.go`, grown by Role Reads 025 and reserved for "the future standalone assignment reads") already carries `id`, `actor_id`, `role_id`, `focus`, `elected_until`, and the embedded `actor` (`{id, name, kind}`). The role-end read (047) reuses it **as-is** because `/roles/{id}/assignments` defaults to `include=actor` — the embed it needs is already present. The **actor-end** read defaults to `include=role`, returning an embedded **role** object (`{id, type, name, purpose, parent_role_id}` per the spec) that the model **does not yet carry**. DECISIONS (047 ADR-2) forecast "050 reuses `Assignment` likewise"; that forecast was precise about the *resource* (one shared `Assignment`, not a new type) but anticipated the embed was already present — for the actor end it is not. 048 ADR (DECISIONS §280) already anticipated the actor-side reads "may grow `glassfrog.Actor`/related models in place (011 ADR-1) for the footprint embeds the directory omits."

**Options considered**:
1. **Add a separate `ActorAssignment`/`AssignmentDetail` type.** Rejected: violates 011 ADR-1 / 025 ADR-2 (one shared schema type, grown not duplicated). The list/single shapes are otherwise identical; a second type would fork the model for one embed.
2. **Reuse `Assignment` unchanged and skip the role projection.** Rejected: the spec's whole point is projecting the *filled role* (name/id + purpose/parent context) the default `role` include returns; dropping it would render an assignment with no human-legible role — only a bare `role_id`.
3. **Grow `Assignment` in place — add an embedded `Role` field (`{id, type, name, purpose, parent_role_id}`) decoded from `?include=role`, present only when the API embeds it; add a new `assignments` list render path.** Chosen.

**Decision**: Option 3 — additive growth, silent conformance to 011 ADR-1 / 025 ADR-2 (grow not duplicate). `Assignment` gains an embedded `Role` struct tagged `json:"role"`, mirroring the existing embedded `Actor` block; nullable `purpose`/`parent_role_id` are modeled as plain strings like the existing nullable `focus`/`elected_until`. The growth is **additive and forward-compatible** — 025's `?include=assignments` embed and 047's `fillers` projection read only the actor block, so the new `role` field decodes unused on those paths (the established 012→025 forward-compatible pattern). The new render path is a homogeneous-row list exactly like 047's `fillers` and 048's `actors`: a `render.AssignmentsView{Data []glassfrog.Assignment}`, a `render.ResourceAssignments` registry entry (`"assignments"`), and `assignments.full.tmpl` / `assignments.compact.tmpl`.

**Consequences**: A minimal schema phase (one embedded struct), where 047 had none — the documented difference between the two ends of the same resource. The templates surface the filled role (name + id, with the purpose/parent context the include carries) alongside `focus` and `elected_until`; the two nullable assignment fields each get an explicit-absence marker (the established render guard — render a marker, never omit-silently or invent). `role.name`/`role.id` render from the default include with no flag. Structured (`json`/`yaml`) output carries the full assignment object regardless of the human projection, since it aggregates raw bytes (018 ADR-2).

### ADR-3: Offer no list filters and no `--include` flag; validate nothing locally; pass the actor id through

**Context**: 025 ADR-4 set the input-handling principle — validate closed-enum inputs locally, pass free identifiers through. Sibling list reads diverge on how much they expose: 038/048 validate a `--status`/`--kind` enum and pass free filters; Role Fillers (047 ADR-3) diverges the *other* way because `listRoleAssignments` accepts no query filter beyond `include` + pagination. `listActorAssignments` is identical: **no query filter** beyond `include` and the pagination params, and its only non-default `include` value (`actor`) is redundant when the caller already supplied the actor id.

**Options considered**:
1. **Expose `--include` (role/actor/role,actor) to mirror the endpoint's one query param.** Rejected: `role` is already the default (and is the whole point of the read), and `actor` re-fetches the actor the caller named — a knob with no useful setting, needing local enum validation for no behavioural gain.
2. **Invent client-side filters (by role type, name, focus).** Rejected: the endpoint offers none, so any narrowing would be the CLI second-guessing the API.
3. **No filter flags, no `--include`; send only the default request + pagination; validate nothing locally.** Chosen.

**Decision**: Option 3 — silent conformance to 047 ADR-3 / 034 ADR-3's "validate nothing" stance. The command sends the actor id into the path unvalidated; a malformed or unknown id surfaces as the API's clean `404`, classified by the shared chain. The default `include=role` is relied upon (whether the request sends `include=role` explicitly or trusts the documented server default is an interface-level detail — both yield the embedded role).

**Consequences**: The same smallest-input surface as 047 — no validator, no `Changed()` filter plumbing, no cross-combo guard. List completeness is unchanged: silent conformance to 025 ADR-3 (`paging.All` walk-by-default, `--first-page` opt-out exits 0 + "more available" signal, mid-walk failure renders partial + cause and exits non-zero).

---

## Cross-cutting Concerns

**Error handling**: Reuses the one shared `classifyClientError` chain (011/015) — no-credential fail-safe (007), wire failure (network-unavailable), and non-2xx (including `404` for an unknown actor id) all flow through the landed `Outcome`/`ExitCode` mapping (004). No new outcome category. A walk that fails mid-stream renders the assignments gathered so far, flagged incomplete with the cause, and exits non-zero (025 ADR-3 / CONSTITUTION VI).

**Output**: Structured output aggregates per-assignment raw bytes via `aggregateRawData` (per-page `meta` dropped); human output renders the `assignments` projection. The command produces structured data and defines no format flag of its own — Output Format Selection (020) resolves `full`/`compact`/`json`/`yaml` (and any `-o <template-ref>`, 035), inherited from the root.

**Testing strategy**: An `assignmentsSeam` of the same shape as `fillersSeam`/`projectsSeam` (assemble + newClient + sleep + resolveSelection + readTemplateSource) so `productionSeam` satisfies it unchanged and tests drive a fake; BDD scenarios drive the command end-to-end against a stub transport, with a transport tripwire confirming the no-credential path issues no request. The render templates get golden-output coverage for present and absent `focus`/`elected_until`, and for the embedded role (with and without `purpose`/`parent_role_id`). The new embedded `Role` decode gets a model-level test that a `?include=role` body populates it and that an actor-end body without it (and the role-end `fillers`/025 paths) leaves it zero-valued without error.

## Implementation Strategy

Two small, naturally-ordered concerns — the model grows before the render path and command reference it, but the work is light enough that the tasks skill may keep it one PR or split model+render from command at its discretion:

1. Grow `glassfrog.Assignment` with the embedded `Role` struct (additive), and add a model decode test.
2. Add `render.ResourceAssignments`, `render.AssignmentsView`, and the `assignments.full`/`assignments.compact` templates.
3. Add the `assignments` command (`newAssignmentsCommand`) in `internal/cli`: the `assignmentsSeam`, the walked-list `RunE` (structured via `Page[json.RawMessage]` + `aggregateRawData`; human via `Page[Assignment]` + `writeHuman`), guard-registration (001), and explicit wiring in `main`.
4. BDD scenarios + render golden tests.

The only inter-step dependency worth naming is "model + render key exist before the command references them."

## Risks

- **`focus`/`elected_until`/role-context absence rendering** (low likelihood, low impact): `focus`, `elected_until`, and the role's `purpose`/`parent_role_id` are nullable; rendering an empty string instead of an explicit-absence marker would silently misrepresent absence. Mitigation: the spec's explicit-absence behaviour is an Output accord bullet and gets dedicated golden coverage (Cross-cutting / Testing).
- **Embedded-role growth perturbing existing decode paths** (low likelihood, low impact): adding a `role` field to the shared `Assignment` could in principle affect 025's `?include=assignments` embed or 047's `fillers` projection. Mitigation: the growth is additive and the new field decodes unused on those paths (the 012→025 forward-compatible pattern); a model test asserts the role-end / 025 bodies leave it zero-valued.
- **Name-vs-resource confusion** (low likelihood, low impact): the command is `assignments` but the model/endpoint are the "assignment" resource read from the *actor* end; a reader might expect it to be symmetric with `fillers`. Mitigation: ADR-1/ADR-2 record the deliberate actor-end framing, and the render key (`assignments`) is the only place the user-facing noun appears.

## What This Plan Does Not Cover

- **Protocol-level detail** — the exact request shape (whether `include=role` is sent explicitly), the `assignments.full`/`assignments.compact` field layout and column choices, the embedded-role field selection in the human projection, and the precise CLI help text are the **interface** skill's concern.
- **Executable scenarios** — the Gherkin for the spec's driving scenarios is the **scenarios** skill's concern.
- **Task decomposition** — whether the model growth, render path, and command ship as one PR or two is the **tasks** skill's call.
- **Role Fillers (047)** — the role-scoped read of the same `Assignment` resource (`/roles/{role_id}/assignments`) is a separate spec; this plan touches only the actor-scoped read and the additive model growth it requires.
