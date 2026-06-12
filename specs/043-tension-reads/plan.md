# Plan: Tension Reads

**Feature**: 043-tension-reads
**Role**: Shaper
**Inputs**: `specs/043-tension-reads/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md` (background); the sibling plans 042-tension-capture (the `tension` namespace, `Tension` model, singular render) and 038-role-projects (the list-by-role + read-by-id read pair); existing code in `internal/cli`, `internal/glassfrog`, `internal/render`, `internal/paging`

---

## System Architecture

Tension Reads adds the **read pair for tensions** — the structural twin of Role Projects (038): a paginated role-scoped list plus a standalone read-by-id, both thin cobra leaves that build a request, hand it to the proven read chain, and render the result. It introduces **no new package** and **no new transport, pagination, error, or output machinery** — every seam it needs is landed.

Two leaves, each `ExactArgs(1)`, guard-registered (001) and attached to the **`tension` group command that 042 lands** (042 ADR-2 reserved the namespace and named these reads as the verbatim follower — "a future `tension get` adds its own leaf with its own flags"):

- **`glassfrog tension list <role-id>`** → `GET /roles/{role_id}/tensions` (`listRoleTensions`) — a **paginated list** of the tensions on a role. Walks to completion through `paging.All[Tension]` (016) by default, with a `--first-page` opt-out and one optional server-side filter: `--status` (validated locally against the tension status set, sent as `status`).
- **`glassfrog tension get <ten-id>`** → `GET /tensions/{id}` (`getTension`) — a **single tension** with full detail, decoded from a `{data: Tension}` document.

Data flow per invocation (identical to 038's 033/034 lineage): resolve `--output` format first (020), validate the one closed-enum input (`--status`) fail-fast where the API would otherwise silently mislead, resolve the connection context once (`AssembleFromOS`, 009), build the `*apiclient.Client` (010/008/007), issue the request (list → `paging.All`; single → one `Execute`), and render by the resolved format (020): the **single** read emits the raw `{data: Tension}` bytes (018) or the human `tension` template (019, the singular render 042 lands); the **list** walks pages — for `json`/`yaml` aggregating each tension's raw bytes into a `{data:[…]}` document via `aggregateRawData` (per-page `meta` dropped), for `full`/`compact` rendering the new `tensions` projection (019). Typed client errors route through the one shared `classifyClientError` chain (011/015) — no new `Outcome` category, no new exit code.

The only genuinely new artifacts are: the **`tension list` / `tension get` leaves**, a **plural `tensions` list render path** (`TensionsView`, the `ResourceTensions` registry entry, and the `tensions.full`/`tensions.compact` templates), and a **`validateTensionStatus`** check over the tension status set. The `glassfrog.Tension` model, the single-object `Document[Tension]` instantiation, and the **singular `tension` render path** are reused as-is from Tension Capture (042) — this feature is a follower on 042's foundation the way 038 followed 014's `Project` model.

---

## Architecture Decisions

### ADR-1: Add `tension list <role-id>` and `tension get <ten-id>` as leaves under 042's `tension` group (conformance to 042 ADR-2)

**Context**: 042 ADR-2 established the `tension` namespace as a non-runnable group parenting a `create` leaf, and explicitly set the precedent that "the future tension reads/edits will conform to" — "a future `tension get` adds its own leaf with its own flags — passing `--body` to a read would be a cobra unknown-flag usage error for free." Tension Reads has two reads keyed on **different id kinds**: a `role_` id selects the per-role list, a `ten_` id selects one tension. The spec fixed the surface as `tension list` / `tension get` (continuing the verb grammar 042 opened, rather than the plural/singular noun pair 038 used) precisely because `tension` is already a group with a `create` sub-verb and cobra cannot tell a bare `tension <ten-id>` from a subcommand name.

**Options considered**:
1. **Plural/singular noun pair** (`tensions <role-id>` / `tension <ten-id>`), as 038 did for projects. Rejected: 042 already claimed `tension` as a group with the `create` leaf, so a bare `tension <ten-id>` would collide with subcommand dispatch — the exact cobra constraint 038 ADR-1 cites. The verb grammar is forced by the shipped write verb.
2. **Two verb leaves under the existing group** — `tension list <role-id>` + `tension get <ten-id>`, both `ExactArgs(1)`, attached to 042's `tension` group. Chosen — silent conformance to 042 ADR-2.

**Decision**: Option 2. `list` and `get` are guard-registered leaves added to the `tension` group 042 builds. The **list-only flags (`--status`, `--first-page`, `--per-page`) live only on `list`** — passing any to `get` is rejected by cobra's own unknown-flag handling as a `UsageError(2)` before assembly, so the spec's "filter on the single read is a usage error" needs no hand-rolled guard (the structural guard 034/038/042 rely on). No `role` group is created (consistent with 026/033/034/038).

**Consequences**: `tension` now parents `create` (042), `list`, and `get`. The three share the namespace without collision; each future edit verb (Tension Update 044, Tension Discard 045) adds its own leaf the same way. The `--base-url` and `--output`/`-o` persistent root flags (011/020) are inherited by all leaves.

### ADR-2: Reuse 042's `Tension` model, `Document[Tension]`, and the singular `tension` render path; add only the plural `tensions` list render path

**Context**: 042 lands `internal/glassfrog/tension.go` (the `Tension` model matching the v5 schema — `id`, `body`, `status`, `role_id`, `sensed_by_id`, `label`, `meeting_type`, `created_at`, `updated_at`, forward-compatible decoding), uses the generic `Document[Tension]` envelope for the created tension, and adds a **singular** `tension` render resource (`ResourceTension` + `tension.full`/`tension.compact`). `getTension` returns "full detail" of the **same** `Tension` schema, so the single read needs no new model and no new singular render. But there is **no plural `tensions` render path** — 042 only created the singular (`builtinResources` today has `ResourceProjects`/`ResourceProject` as the plural/singular precedent; the tension equivalent will have only the singular after 042).

**Options considered**:
1. **Add a fuller list type beside `Tension`.** Rejected: violates 011 ADR-1 (one shared schema type, grown not duplicated); the model is already complete and the list returns the same shape.
2. **Reuse `Tension` unchanged; the list walks `Page[json.RawMessage]` (structured) / `Page[Tension]` (human), the single read decodes `Document[Tension]`, and add only the plural `tensions` render path.** Chosen — mirrors 038 ADR-2 + ADR-4, with the plural/singular roles reversed (038 added the singular because 014 had the list; here 043 adds the list because 042 has the singular).

**Decision**: Option 2. No model change. Add the plural list render path: a `TensionsView` struct in `internal/render`, the `ResourceTensions` registry entry added to `builtinResources` (so the exhaustiveness guard covers it — the PR #10 registry shape), and two embedded templates — `tensions.full` (per-tension: `ten_` id, status, label, body summary, sensing role) and `tensions.compact` (id + status + label/body summary), with the empty-set line (`no tensions`). The single `get` read reuses 042's `tension.full`/`tension.compact` unchanged. Structured `json`/`yaml` reuses the landed machinery: the **single** read serializes the raw `{data: Tension}` verbatim (018, via `output.RenderSuccess`); the **list** walks each page as `json.RawMessage` and emits the aggregated `{data:[…]}` document via `aggregateRawData` — the roles/domains/policies/projects walked-list pattern.

**Consequences**: One new resource key and two new templates, mirroring 038's `project` addition exactly. The incomplete-walk stderr note lives on the command, not the template (025 ADR-3 / 032). No schema phase (model reused from 042).

### ADR-3: Validate the closed-enum `--status` locally via a new `validateTensionStatus`; pass the ids through

**Context**: 025 ADR-4 set the input-handling principle (also applied by 038 ADR-3 and 042 ADR-3): validate closed-enum inputs locally where a wrong value makes the API silently return wrong results; pass free identifiers through where the API reports cleanly. `listRoleTensions` offers a **closed `status` enum** — but a **different set** from the action/project statuses `validateStatus` (`internal/cli/status.go`) covers: tensions are `unprocessed`, `processed`, `archived`. The `role_`/`ten_` ids are free identifiers the endpoints answer with a clean `404`.

**Options considered**:
1. **Reuse `validateStatus`.** Rejected: its set (`archived`, `cancelled`, `completed`, `current`, …) is the action/project vocabulary; reusing it would accept invalid tension statuses (`current`) and reject valid ones (`unprocessed`, `processed`) — a correctness bug.
2. **Pass `--status` through to the API.** Rejected: an unsupported status is silently ignored or mishandled, returning wrong results indistinguishable from "no matches" — the closed-enum hazard 025 ADR-4 validates against locally.
3. **A new `validateTensionStatus` over the tension status set, placed with the tension command code (next to 042's landed `validateMeetingType`); pass the ids through.** Chosen — silent conformance to the validate-closed-enum-locally pattern, a new validator instance over a new set, in the same location 042 ADR-3 placed `validateMeetingType` (the tension command file, NOT the shared `status.go`).

**Decision**: Option 3. A pure `validateTensionStatus` (mirroring 042's landed `validateMeetingType` — `supportedTensionStatuses` map + a sorted `supportedTensionStatusNames()` helper) over the tension status set, placed in the tension command file (`tension_reads.go`, beside `validateMeetingType` in `tension.go`), rejects an unsupported value as a `UsageError(2)` naming the value and the supported set **before any context assembly or request** (a transport tripwire asserts nothing was sent). `--status` is sent (as `status`) only when `Changed()` and non-empty (the 026/034/038 optional-flag discipline); the positional id passes through unvalidated and a bad id surfaces as the API's clean `404`, classified by the shared chain. **Not in `status.go`**: 042 established that tension-domain validators live with the tension code, not in the shared action/project `status.go` (which owns `validateStatus`/`supportedActionStatuses`).

**Consequences**: A new single-sourced tension-status set with the tension command code (the `supportedMeetingTypes` shape from 042); adding a tension status remains a one-line change tracking the vendored `spec/glassfrog-api-v5.yaml`. The shared `status.go` is untouched. Future tension edits (Tension Update 044's `--status`, which also accepts `archived`) reuse this set. No new `Outcome`/`ExitCode`.

---

## Cross-cutting Concerns

**Dependency on 042 (landed) — verified against the implementation**: This plan builds on 042's tension foundation, now **landed on `main` (#91)**: the `tension` group command built by `newTensionCommand(seam)` in `internal/cli/tension.go` (ADR-1; 043 extends it to `MustRegister` the `list`/`get` leaves), the `glassfrog.Tension` model + `Document[Tension]` decode in `internal/glassfrog/tension.go`, and the singular `tension` render resource — `render.ResourceTension` (in `builtinResources`) + `render.TensionView{Tension}` + `tension.{full,compact}.tmpl` (ADR-2). Confirmed against the merged code: the `Tension` fields (`ID/Type/Body/Status/RoleID/SensedByID/CreatedAt/UpdatedAt/Label/MeetingType/ParentRoleID`) cover the list projection and single-read detail; 042's `tensionSeam` is **identical to `projectsSeam`** (`assemble`/`newClient`/`sleep`/`resolveSelection`/`readTemplateSource`) so 043 reuses it for both reads (paging is a `paging.All` call in the command body, not a seam method). **035 (User-Defined Template Output) also landed**, widening the render flow 042/038 use — `resolveRenderTarget` + `writeHuman(…, rt.tmpl, …)` (human) and `output.RenderSuccess`/`aggregateRawData` (machine); 043's reads follow `runProjectsList`/`runProjectGet`/`runTensionCreate` verbatim and inherit `-o <template-ref>` user-template support for free.

**List completeness** (silent conformance to 025 ADR-3): `tension list` defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Tension]` for human; the `--first-page` opt-out does a single `Execute`. A deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial records, writes "incomplete — <cause>" to stderr, and exits non-zero. Never silently truncates (CONSTITUTION VI). Adds no `Outcome`/`ExitCode`.

**Error handling**: identical to every read since 011 — typed client errors route through the single shared `classifyClientError`/`refineClientError` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. Failures render format-aware through 031's `Diagnose` and 032's `reportFailure` chokepoint — structured failures emit the 018 error envelope on stdout, human failures write the diagnostic on stderr; partial-walk notes stay on stderr in every format. No new exit codes. A `404` is the unknown role id (list) or unknown tension id (get); `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)` (a `GET`, so 017 may auto-retry).

**Input validation order**: `--status` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/014/038 fail-fast discipline. `get` declares no `--status`, so the cross-read rejection is cobra's unknown-flag handling (ADR-1).

**Testing**: pure-unit coverage is mostly inherited (the `Tension` decode and `Document[T]`/`Page[T]` generics are tested by 042 and the earlier reads). New tests: golden tests for the two `tensions` list templates (incl. the empty-set line and a multi-status list); `validateTensionStatus` unit tests (supported set passes, empty passes, unsupported names the value + set); a `internal/cli` godog suite over `features/tension-capture/tension-reads.feature` (the tension problem directory, alongside 042's `tension-capture.feature`) driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when `get` is given `--status` (unknown-flag `UsageError`) and (b) no request when `--status` is unsupported.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025).

---

## Implementation Strategy

Single phase — one cohesive read pair with no internal dependencies, every seam below it landed (modulo the 042 prerequisite). Suggested ordering for task decomposition:

1. **Render** — add the plural `tensions` list path: `TensionsView` struct, `ResourceTensions` registry entry (in `builtinResources`), and the `tensions.full`/`tensions.compact` templates; golden tests. (The singular `tension` render path is reused from 042 unchanged.)
2. **Validator** — add `validateTensionStatus` + the tension status set in the tension command file (beside 042's `validateMeetingType`, not the shared `status.go`); unit tests. (Tiny; independent of Phase 1.)
3. **Commands** — `tension list <role-id>` (list: `paging.All[Tension]` + `--status` via `validateTensionStatus`, `--first-page`/`--per-page`, the 025 completeness logic) and `tension get <ten-id>` (single: `Document[Tension]`), guard-registered and attached to 042's `tension` group; route through the walked-list render pattern (`aggregateRawData` for structured / projection for human; the single read via `output.RenderSuccess` / the 042 `tension` render key) and `classifyClientError`. Depends on Phases 1 + 2.
4. **BDD** — the `tension-reads.feature` suite covering the spec's driving scenarios + the two structural tripwires (`--status` on `get`; unsupported `--status`). Depends on Phase 3.

Phases 1 and 2 are independent and can run in parallel; Phase 3 needs both; Phase 4 needs Phase 3. No schema phase (ADR-2: model reused from 042).

---

## Risks

- **042 sequencing — RESOLVED** (was: 042 lands after 043): 042 has **landed on `main` (#91)**, so the `tension` group, `Tension` model, and singular `tension` render are present and the plan is verified against them. 043's base is cut from current `main`; no contingency is needed. (Residual coordination: sibling tension specs 044/045/046 will also attach leaves to the same `tension` group — whichever lands extends `newTensionCommand`.)
- **Tension status enum drift** (low likelihood, low impact): the tension status set is single-sourced with the tension command code (the `supportedMeetingTypes` shape from 042), against the spec enum (`unprocessed`, `processed`, `archived`). Mitigation: a spec change is a one-line update tracking the vendored `spec/glassfrog-api-v5.yaml`; the validator's unit test pins the set.
- **Tension `body` may contain long free text** (low likelihood, low impact): the `tensions` list templates summarize the body (`compact` especially); the singular `tension.full` (042) renders it faithfully (`text/template`, no truncation — CONSTITUTION VI). No reflow.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — exact command/flag spellings, the `TensionsView` field names, the request-descriptor shape, and the template text are the **interface** skill's concern. The names used here (`tension list`/`tension get`, `--status`) are the developer-confirmed surface; treat them as the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units within the single phase are the **tasks** skill's output; the Implementation Strategy above is the input.
- **The `tension` group construction and the `Tension` model** — owned by 042 (landed, #91); this plan reuses them unchanged (ADR-2) and only extends `newTensionCommand` to attach the read leaves.
- **Tension edits/discard, subroles roll-up, and the proposal write-flow** — deferred to their own specs (044/045/046 and Proposal Write-Flow), per the spec non-behaviors.
