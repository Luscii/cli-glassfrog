# Plan: Role Fillers

**Feature**: 047-role-fillers
**Role**: Shaper
**Inputs**: spec.md (047), PROJECT.md, DECISIONS.md, LEARNINGS.md (background), DEPRECATION.md

---

## System Architecture

Role Fillers adds the **role-scoped assignment read** to the Actor Reads slice. Architecturally it is the simplest member of the role-scoped read family: a **single** thin cobra command in `internal/cli` that builds a request, hands it to the proven read chain, walks the pages, and renders the result. It introduces **no new package** and **no new transport, pagination, error, or output machinery** — every seam it needs is landed. It is a near-exact sibling of the *list half* of Role Projects (038) and Role Policies (034), minus the singular standalone read those features pair with (the API exposes no `GET /assignments/{id}` — see ADR-1).

One command, `ExactArgs(1)`, guard-registered (001) and explicitly wired in `main`:

- **`glassfrog fillers <role-id>`** → `GET /roles/{role_id}/assignments` (`listRoleAssignments`) — a **paginated list** of the assignments that fill a role. Each assignment carries the embedded actor (the endpoint's default `?include=actor`, so `{id, name, kind}` arrives without a flag) plus the assignment's own `focus` and `elected_until`. Walks to completion through `paging.All` (016) by default, with the shared `--first-page` opt-out. It offers **no filter flags and no `--include`** (the endpoint accepts neither beyond the default include + pagination — ADR-3).

Data flow per invocation (identical to 038/043/048's list path): the command resolves the connection context once (`AssembleFromOS`), builds the `*apiclient.Client` (010), and walks the list — for `json`/`yaml` reading `glassfrog.Page[json.RawMessage]` and aggregating each assignment's raw bytes into a `{data:[…]}` document via `aggregateRawData` (per-page `meta` dropped), for `full`/`compact` reading `glassfrog.Page[glassfrog.Assignment]` and rendering the new `fillers` projection (019) through `writeHuman`. The role id is sent into the path unvalidated; a bad id surfaces as the API's clean non-2xx, classified by the one shared `classifyClientError` chain (011/015). It adds **no new `Outcome` category and no new exit code**.

The only genuinely new artifacts are the **`fillers` command** and a **`fillers` list render path** (`render.FillersView{Data []glassfrog.Assignment}`, the `render.ResourceFillers` registry entry, and the `fillers.full`/`fillers.compact` templates) — mirroring 048's `actors` render-key addition and 043's plural `tensions` list render. The `glassfrog.Assignment` model and the whole read chain (`paging.All`, `aggregateRawData`, `writeHuman`, the connection seam) are reused as-is.

---

## Architecture Decisions

### ADR-1: Expose a single top-level `fillers <role-id>` command — no plural/singular pair

**Context**: The per-role-read surface precedent (034 ADR-1, followed by 033 and 038) is **two** sibling top-level commands — a plural `<noun>s <role-id>` for the role-scoped list and a singular `<noun> <id>` for the standalone read — because those features have two reads keyed on different id kinds. Role Fillers has only **one** read worth surfacing: the API's `/assignments/{id}` path carries only the administrative `PATCH`/`DELETE` (no `GET`), and assignment administration is out of scope per PROJECT.md. So there is no singular standalone read to pair with the list.

**Options considered**:
1. **Reuse the plural/singular pair anyway** (`fillers <role-id>` + `filler <asgn-id>`). Rejected: there is no `GET /assignments/{id}`, so the singular command would have no endpoint to call — inventing one would fabricate API surface, violating "Spec is the contract" (PROJECT.md).
2. **A `role <role-id> fillers` group.** Rejected for the same reason 034/033 rejected it: a group forces sibling per-role reads to land a shared `role` group in lockstep; the slice deliberately did not create one.
3. **A single top-level `fillers <role-id>` command, `ExactArgs(1)`.** Chosen — the role-scoped list half of the 034/038 pattern, standing alone.

**Decision**: Option 3. `fillers` is one guard-registered, explicitly-wired command taking a required positional role id (`cobra.ExactArgs(1)` — omitted/extra positional is a `UsageError(2)` from cobra's own validator, no hand-rolled guard). Because there are no list-only filter flags to register (ADR-3), the cross-combo guard that 034/038 leaned on cobra for is moot here. It inherits the `--base-url` and `--output`/`-o` persistent root flags (011/020) plus the shared `--first-page`/`--per-page` walk flags.

**Consequences**: This is the **first role-scoped list read with no singular sibling** — a documented, endpoint-driven divergence from the 034/038 plural/singular pattern, not a deviation from it. No `role` group is created. The command name `fillers` speaks the practitioner's question while the model/endpoint stay "assignment" (the resource noun); the two never collide because nothing else claims `fillers`. The slice sibling Actor Assignments (050) will read the *actor-scoped* `/actors/{actor_id}/assignments` — the same resource from the other end — as its own command.

### ADR-2: Reuse `glassfrog.Assignment` as-is — no schema growth; add a new `fillers` list render path

**Context**: The `glassfrog.Assignment` model was grown by Role Reads (025, landed) in `internal/glassfrog/roles.go` for the role's `?include=assignments` embed, and its comment explicitly reserves it for "the future standalone assignment reads" — this feature. It already carries `id`, `actor_id`, `role_id`, `focus`, `elected_until`, and the embedded `actor` (`{id, name, kind}`) — the exact shape `listRoleAssignments` returns with its default `include=actor`. 025's embedded view (on `RoleDetail`) surfaces only the actor reference; the spec's decided projection for this read additionally shows `focus` and `elected_until`.

**Options considered**:
1. **Add a fuller `AssignmentDetail` type beside `Assignment`.** Rejected: violates 011 ADR-1 / 025 ADR-2 (one shared schema type, grown not duplicated); the model is already complete — `focus`/`elected_until` are present and merely unrendered by 025's projection.
2. **Reuse `Assignment` unchanged; the list walks `Page[json.RawMessage]` (structured) / `Page[Assignment]` (human), and a new `fillers` render path projects actor + focus + elected_until.** Chosen.

**Decision**: Option 2 — no model change at all (smaller even than 038, which added a singular render). The list walk reads `glassfrog.Page[json.RawMessage]` for structured output (aggregated via `aggregateRawData`, the landed walked-list pattern) and `glassfrog.Page[glassfrog.Assignment]` for human rendering. The new render path is a homogeneous-row list exactly like 048's `actors` and 043's `tensions`: a `render.FillersView{Data []glassfrog.Assignment}`, a `render.ResourceFillers` registry entry (`"fillers"`), and `fillers.full.tmpl` / `fillers.compact.tmpl`.

**Consequences**: No schema phase. The templates surface fields 025's embedded projection does not — `focus` and `elected_until` — both nullable in the spec, so each gets an explicit-absence marker (the established render guard: omit when unset would hide it; render a marker instead). `actor.name`/`actor.kind` render from the default include with no flag. Structured (`json`/`yaml`) output carries the full assignment object regardless of the projection, since it aggregates raw bytes.

### ADR-3: Offer no list filters and no `--include` flag; validate nothing locally; pass the role id through

**Context**: 025 ADR-4 set the input-handling principle — validate closed-enum inputs locally (where a wrong value makes the API silently mislead), pass free identifiers through. Sibling list reads diverge on how much they expose: 038 ADR-3 validates a `--status` enum and passes `--query`/`--tag`; 048 validates `--kind` and passes `--role-id`/`--query`. Role Fillers diverges the *other* way: `listRoleAssignments` accepts **no query filter** at all beyond `include` and the pagination params, and its only non-default `include` value (`role`) is redundant when the caller already supplied the role id.

**Options considered**:
1. **Expose `--include` (actor/role/actor,role) to mirror the endpoint's one query param.** Rejected: `actor` is already the default (and is the whole point of the read), and `role` re-fetches the role the caller named — a knob with no useful setting. It would also need local enum validation for no behavioural gain.
2. **Invent client-side filters (by kind, name, focus).** Rejected: the endpoint offers none, so any narrowing would be the CLI second-guessing the API; actor-shaped filtering already lives on Actor Directory's (048) `actors --role-id --kind --query` surface.
3. **No filter flags, no `--include`; send only the default request + pagination; validate nothing locally.** Chosen.

**Decision**: Option 3 — silent conformance to 034 ADR-3's "validate nothing" stance (Role Policies likewise exposed no closed-enum input). The command sends the role id into the path unvalidated; a malformed id surfaces as the API's clean `404`, classified by the shared chain. The default `include=actor` is relied upon (whether the request sends `include=actor` explicitly or trusts the documented server default is an interface-level detail — both yield the embedded actor).

**Consequences**: The smallest input surface in the read family — no validator, no `Changed()` filter plumbing, no cross-combo guard. List completeness is unchanged: silent conformance to 025 ADR-3 (`paging.All` walk-by-default, `--first-page` opt-out exits 0 + "more available" signal, mid-walk failure renders partial + cause and exits non-zero).

---

## Cross-cutting Concerns

**Error handling**: Reuses the one shared `classifyClientError` chain (011/015) — no usage transport failure, no-credential fail-safe (007), wire failure (network-unavailable), and non-2xx (including `404` for an unknown role id) all flow through the landed `Outcome`/`ExitCode` mapping (004). No new outcome category. A walk that fails mid-stream renders the assignments gathered so far, flagged incomplete with the cause, and exits non-zero (025 ADR-3 / CONSTITUTION VI).

**Output**: Structured output aggregates per-assignment raw bytes via `aggregateRawData` (per-page `meta` dropped); human output renders the `fillers` projection. The command produces structured data and defines no format flag of its own — Output Format Selection (020) resolves `full`/`compact`/`json`/`yaml` (and any `-o <template-ref>`, 035), inherited from the root.

**Testing strategy**: A `fillersSeam` of the same shape as `projectsSeam` (assemble + newClient + sleep + resolveSelection + readTemplateSource) so `productionSeam` satisfies it unchanged and tests drive a fake; BDD scenarios drive the command end-to-end against a stub transport, with a transport tripwire confirming the no-credential path issues no request. The render templates get golden-output coverage for present and absent `focus`/`elected_until`.

## Implementation Strategy

Single phase — the feature is one command plus one render path, all on landed seams, with no schema growth and no new package:

1. Add `render.ResourceFillers`, `render.FillersView`, and the `fillers.full`/`fillers.compact` templates (the only new render artifacts).
2. Add the `fillers` command (`newFillersCommand`) in `internal/cli`: the `fillersSeam`, the walked-list `RunE` (structured via `Page[json.RawMessage]` + `aggregateRawData`; human via `Page[Assignment]` + `writeHuman`), guard-registration (001), and explicit wiring in `main`.
3. BDD scenarios + render golden tests.

No inter-phase dependency worth sequencing beyond "render key exists before the command references it" — the tasks skill may keep this a single PR or split render-path from command at its discretion.

## Risks

- **`focus`/`elected_until` absence rendering** (low likelihood, low impact): both fields are nullable; rendering an empty string instead of an explicit-absence marker would silently misrepresent "no focus / not an elected seat". Mitigation: the spec's explicit-absence behaviour is an Output accord bullet and gets dedicated golden coverage (Cross-cutting / Testing).
- **Name-vs-resource confusion** (low likelihood, low impact): the command is `fillers` but the model/endpoint are "assignment"; a future reader might expect a `glassfrog.Filler` type. Mitigation: ADR-1/ADR-2 record the deliberate split, and the render key (`fillers`) is the only place the user-facing noun appears.

## What This Plan Does Not Cover

- **Protocol-level detail** — the exact request shape (whether `include=actor` is sent explicitly), the `fillers.full`/`fillers.compact` field layout and column choices, and the precise CLI help text are the **interface** skill's concern.
- **Executable scenarios** — the Gherkin for the spec's driving scenarios is the **scenarios** skill's concern.
- **Task decomposition** — whether the render path and command ship as one PR or two is the **tasks** skill's call.
- **Actor Assignments (050)** — the actor-scoped read of the same `Assignment` resource (`/actors/{actor_id}/assignments`) is a separate spec; this plan touches only the role-scoped read.
