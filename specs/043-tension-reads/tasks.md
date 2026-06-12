# Tasks: Tension Reads

**Feature**: 043-tension-reads
**Concretization**: Full context (plan + spec + interface-cli + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, features/tension-capture/tension-reads.feature

---

## Dependency Graph

Phase 1: `internal/render` plural `tensions` key (1 task, reuses 042's landed `Tension` model) [Shared]
Phase 2: `validateTensionStatus` + tension status set (1 task, no dependencies) [US3]
Phase 3: The two read leaves (2 tasks: T003 list depends on Phases 1+2; T004 get depends on 042's landed singular render + T003 scaffold) [US1/US3/US4 + US2]
Phase 4: Executable acceptance (1 task, depends on Phase 3) [Shared]

5 tasks total | T001 and T002 startable immediately (parallel) | Builder: pipeline

> Plan-faithful: the plan's four ordered steps map here as render plural `tensions` (T001), validator (T002), commands (T003 list / T004 get), and BDD (T005). The plan reuses 042's `internal/glassfrog.Tension` model unchanged (ADR-2) and instantiates the landed generics `Page[Tension]` (016) and `Document[Tension]` (042/034) — **no schema phase**. The **list** leaf (T003) needs the new plural `tensions` render key (T001) and the new `validateTensionStatus` (T002); the **get** leaf (T004) needs 042's **singular** `tension` render key and the shared seam scaffold (T003). Story labels follow the spec's four user scenarios — US1 (list a role's tensions), US2 (read one tension's detail), US3 (narrow by status), US4 (trust the list is whole). T003 serves US1/US3/US4 → `[Shared]`; T004 is cleanly `[US2]`; T002 is the status-filter machinery → `[US3]`.
>
> **✅ 042 (Tension Capture) has landed on `main` (#91) and this task list is verified against the implementation.** Present and reused as-is: `internal/cli/tension.go` (the `tension` group built by `newTensionCommand(seam)`, parenting the `create` leaf via `MustRegister`), `internal/glassfrog/tension.go` (the `Tension` model + `Document[Tension]` decode; fields `ID/Type/Body/Status/RoleID/SensedByID/CreatedAt/UpdatedAt/Label/MeetingType/ParentRoleID`), the singular `tension` render key (`render.ResourceTension` in `builtinResources`) + `render.TensionView{Tension}` + the `tension.{full,compact}.tmpl` templates. **043 attaches `list`/`get` leaves to the existing group** (extend `newTensionCommand` to `MustRegister` the two new leaves) and **reuses 042's `tensionSeam`** (which is identical to `projectsSeam` — paging is done via `paging.All` in the command body, not the seam — so it serves the list walk and the single read unchanged).
>
> **035 (User-Defined Template Output) also landed, widening the output flow** that 042/038 now use: the reads resolve through `resolveRenderTarget(seam, outputFlag, stderr)` (not the older `resolveFormat`), render the human path through `writeHuman(stdout, stderr, rt.tmpl, ResourceX, rt.format, view)` (not a bare `renderFn`), and the machine path through `output.RenderSuccess` (single) / `aggregateRawData` (list). The seam carries `resolveSelection`/`readTemplateSource`, so **043's reads inherit `-o <template-ref>` user-template support for free** by following the `runProjectsList`/`runProjectGet`/`runTensionCreate` pattern. T003/T004 follow this verbatim.
>
> **All other cross-spec dependencies are landed on main.** 007/009/010/011/015/016/017/018/020/025/031/035/038/042 are Complete with their packages shipped — `Page[T]`/`paging.All`, the generic `Document[T]`, `RetryExecutor`, `classifyClientError`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031), the persistent `--base-url`/`--output`/`-o`, the 035-widened render target + `writeHuman`/`aggregateRawData`/`output.RenderSuccess` machinery, and the tension foundation (042). 043 adds **no** new `Outcome` category, `ExitCode` case, generic type, or root flag. **`validateTensionStatus` lives with the tension command code (next to 042's `validateMeetingType` in the tension file), NOT in `status.go`** — 042 established that tension-domain validators live with the tension code, not in the shared action/project `status.go`.
>
> **The tension read surface, counterpart to Tension Capture (042).** `tension create` writes (042); 043 reads `GET /roles/{role_id}/tensions` (list) + `GET /tensions/{id}` (get) as verb leaves under the same `tension` group — the structural twin of Role Projects (038), in the verb grammar 042 opened (plan ADR-1).

---

## Branching Guidance

**Pipeline mode**: `spec/043-tension-reads/base` → `spec/043-tension-reads/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

**Base branch**: 042 has landed on `main` (#91), so cut the 043 base from current `main` — the `tension` group, `Tension` model, and singular `tension` render are all present. No sequencing caveat remains.

**Role-based awareness**: parallel Conductor workspaces may carry sibling tension specs (044 update / 045 discard / 046 subroles roll-up follow). 043 attaches two leaves to 042's `tension` group (extending `newTensionCommand`), adds one plural render key in `internal/render`, and adds a new tension-status validator in the tension command file — it makes **no** change to `internal/glassfrog` (042's `Tension` model reused as-is, plan ADR-2) and does not touch the shared `status.go`.

---

## Phase 1: `internal/render` plural `tensions` key [Shared]

- [x] **T001** [Shared] [P] Add the plural `tensions` list render key + `TensionsView` + templates — golden + registry-guard tests; `internal/render` — 2 scenarios, 5 golden tests (new `tensions.{full,compact}.tmpl`, `TensionsView`, `ResourceTensions`)
  - **Scope**: In `internal/render`, add **one new** render key `tensions` (list, data `[]glassfrog.Tension` via a `TensionsView`, mirroring the landed `ProjectsView`). Add the `ResourceTensions` constant to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates: `tensions.full` — one block per tension (`<ten_…>  [<status>]  <label|—>`, then the `body` on its own indented line rendered **verbatim — never truncated or reflowed** (CONSTITUTION VI), then an indented `sensing role: <role_…|—>` line); `tensions.compact` — one line per tension (`<ten_…>  [<status>]  <label | one-line body summary>`). Both render the empty-set line `no tensions` for an empty slice. Use 019's absence-guard discipline (`{{if eq (trimSpace …) ""}}…`). The **singular** `tension` key (042) is reused unchanged by the `get` leaf — touch no existing key or template. Depends on `internal/glassfrog` (the `Tension` model from 042) + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `tensions` key renders both `full` and `compact`; `full` shows id, status, label, body, and sensing role per tension
    - An empty slice renders `no tensions` (not `<no value>`, not a blank); a null/empty `label` and a null `role_id` render their explicit-absence markers
    - A long free-text `body` is rendered verbatim (neither truncated nor reflowed)
    - The registry-exhaustiveness guard passes with the new `tensions` key carrying both formats; golden tests pin each template
    - The singular `tension` key/templates (042) are unchanged; `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: 042's `glassfrog.Tension` model (landed on main, #91) — the `TensionsView` references `[]glassfrog.Tension`. Startable immediately.
  - **Plan reference**: Phase 1 (Render); ADR-2 (reuse 042's singular `tension` key, add the plural `tensions` key)
  - **Interface references**: interface-cli.md — Output (list `tensions`, `full`/`compact` shapes), Consistency Notes (render keys)
  - **Scenario references**: tension-reads.feature: "A role's tensions are listed", "A role with no tensions is a clean success"
  - **Risk**: ⚠️ Add the plural key only — reuse 042's landed singular `tension` key (`ResourceTension` + `TensionView` + `tension.{full,compact}.tmpl`), touch no existing template. ⚠️ Never truncate/reflow `body` (CONSTITUTION VI). ⚠️ Explicit-absence guards for every nullable field; never invent a value. ⚠️ Add `ResourceTensions` to the guarded `builtinResources` set so exhaustiveness still holds.

## Phase 2: `validateTensionStatus` + tension status set [US3]

- [x] **T002** [US3] [P] Add `validateTensionStatus` + the tension status set in the tension command file — unit tests; a NEW set distinct from `validateStatus` — in new tension_reads.go, 3 unit tests
  - **Scope**: In the tension command file (the new `tension_reads.go`, or `tension.go` beside 042's `validateMeetingType`/`supportedMeetingTypes` — **tension-domain validators live with the tension code, NOT in the shared `status.go`**, per 042's just-landed precedent), add a pure `validateTensionStatus(s string) error` mirroring the landed `validateMeetingType`/`validateStatus` shape, over a **new** tension status set (`unprocessed`, `processed`, `archived`) in a single-sourced `supportedTensionStatuses` map + a sorted `supportedTensionStatusNames()` helper (the `supportedMeetingTypes`/`supportedMeetingTypeNames` shape from 042). An empty string passes (no filter); an unsupported non-empty value returns an error naming the value and the supported set (sorted, like `validateMeetingType`'s message). **Do not reuse `validateStatus`** — its action/project vocabulary (`current`/`completed`/…) is wrong for tensions (plan ADR-3). Pure, no I/O; imports nothing new.
  - **Acceptance criteria**:
    - `validateTensionStatus("unprocessed")`, `("processed")`, `("archived")`, and `("")` all return nil
    - `validateTensionStatus("open")` returns an error naming the unsupported value and listing the supported set (`archived`, `processed`, `unprocessed`)
    - The set is single-sourced (one map/var); adding a status is a one-line change
    - The shared `status.go` (`validateStatus`/`supportedActionStatuses`) and 042's `validateMeetingType`/`supportedMeetingTypes` are untouched; `go build`/`go vet` clean; unit tests pin the supported and rejected cases
  - **Dependencies**: None — a pure new validator in the tension command code (mirrors 042's `validateMeetingType`). Startable immediately against current main.
  - **Plan reference**: Phase 2 (Validator); ADR-3 (validate the closed-enum `--status` locally via a new validator)
  - **Interface references**: interface-cli.md — `--status` flag (validated set), Consistency Notes (one local validator, a new set)
  - **Scenario references**: tension-reads.feature: "An unsupported status is rejected as a usage error", "A rejected status issues no request" (the validator is the mechanism)
  - **Risk**: ⚠️ A new set, NOT a reuse of `validateStatus` — reusing it would accept invalid tension statuses and reject valid ones (a correctness bug, plan ADR-3). ⚠️ Empty string must pass (it means "no filter"). ⚠️ Keep the set single-sourced. ⚠️ Place it with the tension code (next to 042's `validateMeetingType`), NOT in the shared `status.go`.

## Phase 3: The two read leaves [US1/US3/US4 + US2]

- [x] **T003** [Shared] Add the `tension list <role-id>` command (reusing 042's `tensionSeam`) + `--status`/`--first-page`/`--per-page` + completeness + wiring under 042's `tension` group — RED-first unit tests for every branch — runTensionList in tension_reads.go, wired via newTensionCommand; 17 unit tests (walk/empty/status/first-page/per-page/mid-walk/classify/bad-output/structured)
  - **Scope**: New `internal/cli/tension_reads.go` (or extend 042's `tension.go`). `newTensionListCommand(seam tensionSeam) *cobra.Command`: a guard-registered leaf (`Use:"list <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), declares `--status`, `--first-page`, `--per-page`; reads the persistent `--base-url`/`--output`; delegates to pure `runTensionList`. **Reuse 042's existing `tensionSeam`** (`assemble`/`newClient`/`sleep`/`resolveSelection`/`readTemplateSource` — identical to `projectsSeam`; paging is a `paging.All` call in the command body, not a seam method, so no new seam is needed); prod passes `productionSeam{}` from `Assemble`, tests bind a fake. `runTensionList(cfg) (Outcome, error)` follows `runProjectsList` verbatim (035-widened): call `resolveRenderTarget(cfg.seam, cfg.outputFlag, cfg.stderr)` → `rt` and **validate `--status` via `validateTensionStatus`** (T002) **before** assembly, so a bad template/format or bad status is a fail-fast `UsageError(2)` with no request; build the `status` query parameter only when `Changed()` and non-empty; default path walks `GET /roles/{role_id}/tensions` to completion using the landed two-track list pattern (NOT `renderResult[T]`): for the machine format (`rt.format.MachineFormat()` ok), walk `paging.All[json.RawMessage]` then `aggregateRawData(machineFmt, res.Records)` into the `{data:[…]}` document (per-record raw bytes preserved, per-page `meta` dropped); for the human path, walk `paging.All[glassfrog.Tension]` then `writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTensions, rt.format, view)` over a `TensionsView` (T001) — `writeHuman` carries 035 user-template support, so `-o <template-ref>` works for free. `--first-page` does one `Execute` into the corresponding `Page[…]` and writes a "more tensions exist" stderr note (exit 0) when `HasNextPage`; a mid-walk `Result.Stop` renders the partial set, writes an "incomplete — <cause>" stderr note, and exits non-zero via `classifyClientError(Stop)`; `--per-page` sets `WithPageSize`. The role id is passed through (`url.PathEscape`, no local validation — plan ADR-3). Attach the leaf to 042's `tension` group by extending `newTensionCommand` to `MustRegister` it (the group already parents `create`). Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog tension list role_0123` walks every page (`paging.All`) and prints the projection of all the role's tensions; exits 0
    - A role with no tensions prints the `tensions` empty line (`no tensions`) and exits 0
    - `--status unprocessed` sends `status=unprocessed`; an omitted/empty `--status` sends nothing
    - `--status open` (unsupported) is a `UsageError(2)` naming the value + supported set, **no request sent** (transport tripwire)
    - `--first-page` against a multi-page list prints one page, writes a "more tensions exist" stderr note, exits 0
    - A mid-walk failure prints the partial set, writes an "incomplete — <cause>" stderr note, exits non-zero (classified from `Stop`)
    - `*AuthError{NoCredentials}` → UsageError(2); `*TransportError` → NetworkUnavailable(6); `*ResponseError` → APIError(3)/PermissionError(4)/RateLimited(5); base-URL/`--output` errors → UsageError(2); `-o json`/`yaml` emit the aggregated `{data:[…]}` document, `full`/`compact` the human projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`tensions` render key), T002 (`validateTensionStatus`); 042's `tension` group + `Tension` model (landed on main, #91).
  - **Plan reference**: Phase 3 (Commands); ADR-1 (verb leaf under 042's group, `ExactArgs(1)`, list-only flags), ADR-3 (`--status` validated locally; id passed through); Cross-cutting (completeness reuses 025 ADR-3)
  - **Interface references**: interface-cli.md — `tension list` Surface, list flags, Output, Interactions (validation order + completeness), Error Communication
  - **Scenario references**: tension-reads.feature: "A role's tensions are listed", "A role with no tensions is a clean success", "A missing token fails as a not-authenticated usage error", "The tension list is narrowed by a supported status", "An unsupported status is rejected as a usage error", "A rejected status issues no request", "The first-page opt-out stops at one page and signals more", "A mid-walk failure yields a partial set flagged incomplete"
  - **Risk**: ⚠️ Reuse `validateTensionStatus`(T002)/`classifyClientError`/`resolveRenderTarget`+`aggregateRawData`+`writeHuman`/`paging.All`/`RetryExecutor` — inline no second copy of the page loop, render branch, or `errors.As` chain (follow `runProjectsList` verbatim). ⚠️ Reuse 042's existing `tensionSeam` — do not define a parallel `tensionsSeam`. ⚠️ Structured output is the aggregated `{data:[…]}` over `Page[json.RawMessage]`, NOT a decode-and-re-encode of `[]Tension` and NOT a single page's raw envelope. ⚠️ `--status` sent only when `Changed()` and non-empty. ⚠️ Resolve the render target (`resolveRenderTarget`) and validate `--status` BEFORE assembly so the tripwire confirms no request on rejection. ⚠️ Never silently truncate — every incomplete path writes an explicit stderr note (CONSTITUTION VI). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Attach to 042's group by extending `newTensionCommand` — don't create a second `tension` group. ⚠️ Never read `ctx.Cred.Token`.

- [x] **T004** [US2] Add the `tension get <ten-id>` single-read command — `runTensionGet` + `Document[Tension]` decode + the singular `tension` render dispatch — RED-first unit tests — runTensionGet in tension_reads.go, wired via newTensionCommand; 5 unit tests (detail/unknown-id/structured/list-flag tripwire/ExactArgs)
  - **Scope**: In `internal/cli/tension_reads.go`, add `newTensionGetCommand(seam tensionSeam) *cobra.Command` (reusing 042's `tensionSeam`): a guard-registered leaf (`Use:"get <ten-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`), declares **no** list flags (so `--status`/`--first-page`/`--per-page` are cobra unknown-flag usage errors — the structural list-only guard, plan ADR-1); reads the persistent `--base-url`/`--output`; delegates to pure `runTensionGet`. `runTensionGet(cfg, id) (Outcome, error)` follows `runProjectGet`/`runTensionCreate` verbatim (035-widened): `resolveRenderTarget(...)` before assembly; with the id `url.PathEscape`-d and passed through unvalidated (plan ADR-3) so an unknown id surfaces the API's non-2xx via the shared classifier. For the machine format (`rt.format.MachineFormat()` ok), `Execute` into a raw `json.RawMessage` and emit the `{data: Tension}` body verbatim via `output.RenderSuccess` (018); for the human path, `Execute` into `glassfrog.Document[glassfrog.Tension]` and render via `writeHuman(cfg.stdout, cfg.stderr, rt.tmpl, render.ResourceTension, rt.format, render.TensionView{Tension: doc.Data})` — the **singular `tension` render key from 042** (the exact shape `runTensionCreate` already uses; `writeHuman` carries 035 user-template support, NOT `renderResult[T]`). Attach the leaf to 042's `tension` group by extending `newTensionCommand` to `MustRegister` it. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog tension get ten_0123` reads the tension and prints its status, body, and sensing role; exits 0
    - An unknown `ten_ffff` surfaces the API status (APIError(3)/PermissionError(4)) — the id is not validated locally
    - A list flag on `get` (`--status`/`--first-page`/`--per-page`) is a `UsageError(2)` via cobra's unknown-flag handling, **no request sent** (transport tripwire across the flags)
    - `-o json`/`yaml` emit the raw single-tension payload; `full`/`compact` render the singular `tension` human projection (042's key); `-o <template-ref>` renders through a 035 user template
    - No new `Outcome`/`ExitCode`/root flag; no token in any output; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T003 (the shared `tensionSeam` reference + file scaffold); 042's singular `tension` render key + `tension` group + `Document[Tension]` (landed on main, #91).
  - **Plan reference**: Phase 3 (Commands); ADR-1 (verb leaf, list-only-ness structural), ADR-3 (id pass-through), ADR-2 (reuse `Document[Tension]` + 042's singular render)
  - **Interface references**: interface-cli.md — `tension get` Surface, Output (single), Error Communication
  - **Scenario references**: tension-reads.feature: "A single tension is read with full detail", "An unknown tension id fails with the API status", "The list filter is rejected on the single read"
  - **Risk**: ⚠️ Declare no list flags on `get` — that's how list-only-ness is enforced (no hand-rolled cross-combo guard; plan ADR-1). ⚠️ Pass the id through (no regex gate); the API 404s cleanly (plan ADR-3). ⚠️ Reuse 042's singular `tension` render key — do not add a second singular key. ⚠️ Render the detail faithfully (no truncation). ⚠️ Never read `ctx.Cred.Token`.

## Phase 4: Executable acceptance [Shared]

- [ ] **T005** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `tension-reads.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/tension-capture/tension-reads.feature` in a **new** `internal/cli` godog suite (e.g. `TestTensionReadsFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `tension list`/`tension get` through the shared seam over a fake base `http.RoundTripper` returning canned `GET /roles/{role_id}/tensions` (single-page, multi-page, mid-walk error, empty, status-filtered) and `GET /tensions/{id}` (found, 404) responses, plus a transport tripwire for the no-request paths (unsupported `--status`; list flag on `get`). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 3 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / projection phrasings from the `me*`/`roles`/`policies`/`projects`/`tension-capture` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` tension-reads scenario has an executable, passing path; `@wip` removed from them
    - The 3 `@validation` scenarios keep `@wip`
    - The new suite's `Paths` names only `tension-reads.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T003 (`tension list`), T004 (`tension get`) — all behavioral scenarios must be implementable
  - **Plan reference**: Phase 4 (BDD); System Architecture (attach to 042's group); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: tension-reads.feature: all behavioral Rule-block scenarios (the 3 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `tension-reads.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ Cover the mid-walk-error, multi-page, empty-list, status-filtered, and no-request (tripwire) fakes so the completeness, empty, filter, and rejection scenarios genuinely exercise their paths.
