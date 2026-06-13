# Plan: Subroles Tension Roll-up

**Feature**: 046-subroles-tension-roll-up
**Role**: Shaper
**Inputs**: `specs/046-subroles-tension-roll-up/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent: §300/§297 043 tension reads, §285 042 `tension` group, §211/§205 026 `subroles`, §324/§321 044), `.score/memory/DEPRECATION.md` (no relevant entries), `.score/memory/LEARNINGS.md` (background); the landed code on `main` — `internal/cli/tension.go`, `internal/cli/tension_reads.go`, `internal/glassfrog/tension.go`, `internal/render/render.go`, `internal/paging` — and the sibling plan 043-tension-reads

---

## System Architecture

Subroles Tension Roll-up adds **one read leaf** to the `tension` command group: `glassfrog tension subroles <role-id>` → `GET /roles/{role_id}/subroles/tensions` (`listSubrolesTensions`) — a **paginated roll-up** of the tensions sensed across the anchor role's **direct sub-roles** (one level, not transitive). It is the thinnest member of the tension read family: where Tension Reads (043) had to *add* the plural `tensions` render path, this feature **reuses every seam already landed** and introduces **no new package, model, render resource, validator, transport, pagination, error, or output machinery**. The only genuinely new artifact is the leaf itself (and its BDD feature) — its request differs from `tension list`'s only in the path segment.

The leaf is `ExactArgs(1)`, guard-registered (001), and attached to the `tension` group that `newTensionCommand(seam)` builds in `internal/cli/tension.go` (which already parents `create` (042), `list`/`get` (043), and `update` (044)). It is structurally a clone of the landed `newTensionListCommand` (`internal/cli/tension_reads.go:246`): it carries the same three list flags — `--status` (validated locally via the landed `validateTensionStatus`, sent as `status`), `--first-page`, `--per-page` — walks to completion through `paging.All[Tension]` (016) by default, and renders through the shared flow.

Data flow per invocation (identical to `tension list`'s 042/043 lineage): resolve `--output`/render target first (020/035), validate the one closed-enum input (`--status`) fail-fast where the API would otherwise silently mislead, resolve the connection context once (`AssembleFromOS`, 009), build the `*apiclient.Client` (010/008/007), issue the walk against `/roles/{role_id}/subroles/tensions` (`paging.All`), and render by the resolved format (020): `json`/`yaml` aggregate each page's raw bytes into a `{data:[…]}` document via `aggregateRawData`; `full`/`compact` render the **landed** `tensions` projection (`ResourceTensions` + `TensionsView`, 043). Typed client errors route through the one shared `classifyClientError`/`Diagnose` chain (011/015/031/032) — no new `Outcome`, no new exit code. The leaf-role `404` is surfaced verbatim by that chain.

---

## Architecture Decisions

### ADR-1: Add `tension subroles <role-id>` as a third read verb under 042's `tension` group (conformance to 042 ADR-2 / 043 ADR-1)

**Context**: 042 ADR-2 (§285) established `tension` as a non-runnable group whose future reads/edits each add their own leaf with their own flags; 043 ADR-1 (§300) seated `list`/`get` there, and 044 added `update`. The spec settled the surface in the define session: a distinct `tension subroles <role-id>` verb, **not** a `--subroles` flag on `tension list`. The roll-up keys on a `role_` id (the anchor), exactly like `tension list`, but hits a different endpoint with different "anchor must have sub-roles" semantics.

**Options considered**:
1. **A `--subroles` flag on `tension list`** — one verb spanning two endpoints. Rejected: 043's non-behavior #1 deliberately scoped `tension list` to a role's *own* tensions and declared the roll-up its own capability; overloading the verb forks 043's contract and makes one command's endpoint depend on a flag.
2. **A new `subroles` leaf under the existing `tension` group** — `tension subroles <role-id>`, `ExactArgs(1)`, attached to `newTensionCommand`. Chosen — silent conformance to 042 ADR-2's "each follower adds its own leaf" precedent; one verb = one endpoint.

**Decision**: Option 2. `subroles` is a guard-registered leaf added to the `tension` group beside `create`/`list`/`get`/`update`. The list-only flags (`--status`, `--first-page`, `--per-page`) live on it; it inherits the `--base-url` and `--output`/`-o` root persistent flags (011/020/035). The verb name `subroles` does **not** collide with the top-level `subroles <role-id>` command (026, `listSubroles`): the two live at different command paths (`tension subroles …` vs `subroles …`), so cobra dispatch is unambiguous and the names rhyme intentionally (both are "the direct children of this role").

**Consequences**: `tension` now parents five verbs (`create`/`list`/`get`/`update`/`subroles`). The roll-up sits beside the role's-own-tensions list without merging — a reader picks `list` for one role's tensions and `subroles` for the children's. Future tension verbs continue to add leaves the same way.

### ADR-2: Reuse the landed `Tension` model, `tensions` render path, and `validateTensionStatus` unchanged — add no new model, render resource, or validator

**Context**: `listSubrolesTensions` returns the **same** paginated `{data: [Tension], meta}` shape as `listRoleTensions` (verified in `spec/glassfrog-api-v5.yaml:2625-2646`). 043 already landed the plural list render path — `render.ResourceTensions` + `render.TensionsView` + `tensions.full`/`tensions.compact` (in `builtinResources`, `render.go:358/381`) — and the closed-enum validator `validateTensionStatus` + `supportedTensionStatuses` over the tension set (`unprocessed`/`processed`/`archived`, `tension_reads.go:421/427`). The `Tension` model + `Page[Tension]`/`Document[Tension]` are landed from 042.

**Options considered**:
1. **Add a roll-up-specific render or model** (e.g. grouping tensions by sub-role). Rejected: the endpoint returns a flat `[]Tension`, the same shape `tension list` renders; inventing a grouped projection would fabricate structure the API does not provide (019 anti-fabrication, CONSTITUTION VI) and duplicate a landed render path (011 ADR-1, grow-not-duplicate).
2. **Reuse `ResourceTensions`/`TensionsView`, the `Tension` model, and `validateTensionStatus` exactly as landed** — the roll-up is a different *source* of the same row shape. Chosen — silent conformance to 011 ADR-1, 043 ADR-2/ADR-3, and 044's "new consumer of a landed validator, not a new validator" pattern (§324).

**Decision**: Option 2. No new render resource, no new model, no new validator. `--status` reuses the landed `validateTensionStatus` (a new *consumer* of 043's set — 043's DECISIONS entry already named 044 as the next consumer; 046 is another), rejecting an unsupported value as a `UsageError(2)` naming the value and the supported set **before any context assembly or request** (transport tripwire). `--status` rides (as `status`) only when `Changed()` and non-empty (the 026/034/038/043 optional-flag discipline). The `role_` anchor id passes through unvalidated to the API (042/043 ADR-3, §200).

**Consequences**: 046 is purely additive at the command layer — zero churn in `internal/render`, `internal/glassfrog`, or the validator set. If the spec's tension status enum ever drifts, the one-line change still lives in 043's landed set; 046 inherits it for free. No schema phase, no render phase.

### ADR-3: The roll-up reuses the `tension list` runner shape with the path swapped; the leaf-role `404` is surfaced verbatim, distinct from the empty-list `200`

**Context**: The roll-up differs from `tension list` only in (a) the request path (`/roles/{role_id}/subroles/tensions` vs `/roles/{role_id}/tensions`) and (b) the API's leaf-role behavior: a leaf anchor (no sub-roles) returns **`404`** rather than an empty `200`. The spec settled (define session) that the CLI surfaces that `404` as the shared read failure with **no** "this role has no sub-roles" interpretation — and that it stays distinct from the genuine empty-list success (sub-roles exist but carry no tensions → `200` with `{data: []}`).

**Options considered**:
1. **Special-case the leaf `404`** into a friendly "no sub-roles" empty-list success. Rejected: the CLI is a faithful API surface (VISION Exclusion 1); turning a server `404` into a `0`-exit empty success would hide the wire truth and conflate two genuinely different outcomes (leaf vs childless-but-tensionless).
2. **Surface the `404` through the shared `classifyClientError`/`Diagnose` chain unchanged**, and let the empty `200` render as the landed `no tensions` empty-set line. Chosen — silent conformance to 015/031/032; the two empty-ish outcomes stay distinguishable (non-zero "read failed, status 404" vs `0`-exit empty list).

**Decision**: Option 2. The leaf reuses the `runTensionList` shape (status validation → walk → render → completeness note) with the request path swapped to `/roles/{role_id}/subroles/tensions`. Whether that path swap is expressed by **parameterizing** `runTensionList`/`tensionsConfig` with the path or by a **thin sibling runner** (`runTensionSubroles`) is an interface/tasks-level call; the recommended shape is to parameterize the path since the status filter, paging, completeness, render, and error handling are byte-identical. A `404` (leaf anchor or unknown role id) routes through the shared chain → generic API-error outcome naming the status, non-zero exit; `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)` (a `GET`, so 017 may auto-retry). An empty `200` renders the landed empty-set line and exits `0`.

**Consequences**: No new `Outcome`/`ExitCode`. The Builder must not add a leaf-`404` special case (a Guardian flag if it appears). A transport tripwire and a BDD scenario pin both the leaf-`404`-is-a-failure and the empty-`200`-is-a-success outcomes so they cannot be conflated.

---

## Cross-cutting Concerns

**Dependency on 043 and 042 (both landed)**: verified against `main` — 043 landed at #97 (`tension list`/`tension get`, the `tensions` render path, `validateTensionStatus`), 042 at #91 (the `tension` group via `newTensionCommand`, the `Tension` model, `Document[Tension]`), 044 at #107 (`tension update`). The roll-up reuses all of these unchanged and only extends `newTensionCommand` to attach the fifth leaf, reusing the shared `tensionSeam` (identical to `projectsSeam`; paging is a `paging.All` call in the runner body, not a seam method). 035 (User-Defined Template Output) is landed, so the leaf inherits `-o <template-ref>` support through the shared `resolveRenderTarget`/`writeHuman`/`aggregateRawData`/`output.RenderSuccess` flow for free.

**List completeness** (silent conformance to 025 ADR-3): `tension subroles` defaults to `paging.All` and reduces to `(records, complete)` — walking `Page[json.RawMessage]` for structured output and `Page[Tension]` for human. The `--first-page` opt-out does a single `Execute`; a deliberate opt-out with more pages exits `0` with a one-line "more available — re-run without --first-page" stderr note; a mid-walk failure renders the partial records, writes "incomplete — <cause>" to stderr, and exits non-zero via `classifyClientError(Stop)`. Never silently truncates (CONSTITUTION VI).

**Error handling** (conformance to 011/015/031/032, see ADR-3): typed client errors route through the single shared `classifyClientError`/`Diagnose` chain; the not-authenticated fail-safe (007) refuses at send time; messages name the failure + a next step and never include the token. Failures render format-aware through 032's `reportFailure` chokepoint; partial-walk notes stay on stderr in every format.

**Input validation order**: `--status` is validated (pure, no I/O) ahead of any context assembly, so a transport tripwire can assert no request issued on rejection — the 011/013/038/043 fail-fast discipline. The `role_` id passes through to the API's clean `404`.

**Testing**: pure-unit coverage is inherited (the `Tension` decode, `Page[T]` generics, the `tensions` golden templates, and `validateTensionStatus`'s set are all tested by 042/043). New tests: a `internal/cli` godog suite over a new `features/tension-capture/subroles-tension-roll-up.feature` (alongside the existing tension features) driven by a fake transport returning canned pages, with **transport tripwires** asserting (a) no request when `--status` is unsupported and (b) the leaf-`404` is surfaced as a non-zero failure naming the status, distinct from the empty-`200` success. Reuse the sibling-suite step phrasing (godog matches by text) and never hard-code the validator's supported-set order — it is alphabetically sorted via `supportedTensionStatusNames()`.

**Configuration**: none new. Reuses `--base-url`/`--output`/`-o` (root persistent) and the `--first-page`/`--per-page` list flags (016/025).

---

## Implementation Strategy

Single phase — one cohesive read leaf with no internal dependencies, every seam below it landed:

1. **Command** — `newTensionSubrolesCommand(seam)`: a `tension subroles <role-id>` leaf (`ExactArgs(1)`) modeled on `newTensionListCommand`, registered into `newTensionCommand`'s group. Its runner reuses the `tension list` shape (`--status` via `validateTensionStatus`, `--first-page`/`--per-page`, the 025 completeness logic, the walked-list render via `aggregateRawData`/the `tensions` render key) with the request path set to `/roles/{role_id}/subroles/tensions`. Route failures through `classifyClientError`/`Diagnose` with no leaf-`404` special case.
2. **BDD** — the `subroles-tension-roll-up.feature` suite covering the spec's driving scenarios (roll-up, status filter, full-walk, leaf-`404` failure, no-credential, empty-`200`, unsupported-`--status`, first-page opt-out) plus the two structural tripwires.

Phase 2 depends on Phase 1. No render phase, no validator phase, no schema phase (ADR-2: all reused from 042/043).

---

## Risks

- **Sibling tension specs share `newTensionCommand`** (low likelihood, low impact): 044 (`update`) is landed and 045 (Tension Discard) may add its own leaf to the same group. Mitigation: each spec only *appends* a guard-registered leaf; whichever lands extends `newTensionCommand` and the group's `MustRegister` calls — no shared mutable state, the 001 guard back-stops a malformed group at attach time. 046's base is cut from current `main`.
- **Leaf-`404` mistaken for empty result** (low likelihood, medium impact): a Builder might "helpfully" turn the leaf anchor's `404` into an empty-list success, conflating two distinct outcomes (ADR-3). Mitigation: an explicit BDD scenario + the spec's validation scenario pin the `404`-is-a-failure and empty-`200`-is-a-success outcomes; the Guardian flags any special-casing.
- **Path-swap implementation drift** (low likelihood, low impact): if the leaf clones `runTensionList` instead of parameterizing the path, the two runners could drift (e.g. a future completeness fix lands in one only). Mitigation: ADR-3 recommends parameterizing the path so the status/paging/render/error logic is single-sourced; if a sibling runner is chosen, a test asserts both emit the same render path.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — the exact command/flag spellings (`tension subroles`, `--status`/`--first-page`/`--per-page`), the request-descriptor shape, and whether the runner parameterizes the path or forks a sibling are the **interface** skill's concern. The names used here are the define-session-confirmed surface; treat them as the working contract.
- **Executable scenarios** — the `.feature` file is the **scenarios** skill's output; the Driving Scenarios in spec.md are the source.
- **Task decomposition** — PR-sized units within the single phase are the **tasks** skill's output; the Implementation Strategy above is the input.
- **The `tension` group, the `Tension` model, the `tensions` render path, and `validateTensionStatus`** — owned by 042/043 (landed); this plan reuses them unchanged (ADR-2) and only attaches the roll-up leaf.
- **Transitive/recursive roll-up, the role's-own-tensions list, tension writes, and the proposal write-flow** — out of scope per the spec non-behaviors (043 owns `tension list`; 044/045 own edits/discard).
</content>
</invoke>
