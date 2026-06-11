# Tasks: Tension Capture

**Feature**: 042-tension-capture
**Concretization**: Full context (plan + spec + interface-cli + interface-spec + scenarios)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/tension-capture/tension-capture.feature

---

## Dependency Graph

Phase 1: Write-body transport seam (1 task, no phase dependencies) [Shared]
Phase 2: The tension create command (4 tasks: T002 model — no deps, parallel with Phase 1; T003 render — depends T002; T004 command — depends Phases 1+T002+T003; T005 acceptance step — depends T004) [US1/US2/US3/US4]

5 tasks total | T001 and T002 startable immediately (parallel) | Builder: pipeline

> Plan-faithful: the plan's **two** phases map here directly — the transport seam (Phase 1: T001) and the command (Phase 2: T002 model / T003 render / T004 command). The godog acceptance work (T005) is the closing **acceptance step of Phase 2**, realizing plan's Cross-cutting testing note — not a separate numbered phase, since plan defines exactly two. **This is the CLI's first write**, so the only genuinely new infrastructure is the write-body `Content-Type` field on `apiclient.Request` (T001, plan ADR-1) — everything else rides the landed read chain. Story labels follow the spec's four user scenarios — US1 (record a tension), US2 (get back the `ten_` id), US3 (attach label + meeting-type), US4 (refuse an empty body). The command (T004) realizes all four in one leaf, so it carries `[US1/US2/US3/US4]`; the seam, model, and render key are shared infrastructure `[Shared]`.
>
> **All cross-spec dependencies are landed on main.** Per STATUS.md, 007/009/010/011/015/016/017/018/019/020/032/034 are Complete with their packages shipped — `apiclient.Request`/`(*Client).Execute`/`AssembleFromOS`/`NewClientFromOS` + the typed errors (010), `NewRetryExecutor` incl. the `isSafeMethod` 429 gate (017), the generic `Document[T]` (034), `classifyClientError`/`refineClientError`/`reportFailure`/`Diagnose`/`Outcome`/`ExitCode` (011/015/031/032), `output.ResolveFormat`/`RenderSuccess` + the `cli` `renderResult[T]` dispatch and the two-package `output`/`render` split (018/019/020), the persistent `--base-url`/`--output`/`-o`, the shared `validateStatus` shape (`internal/cli/status.go`, 013/014) that `validateMeetingType` mirrors, and the registration guard + `Assemble()` wiring (001). The 042 base cuts from current main with no sequencing caveat. **Existing main state 042 builds on**: there is **no** `tension` command, **no** `glassfrog.Tension` model, **no** `tension` render key, and `apiclient.Request` has **no** `ContentType` field (T001 adds it, additively). 042 adds **no** new `Outcome` category, `ExitCode` case, generic envelope type, or root flag.
>
> **First write, reserves the `tension` namespace** (plan ADR-2). The `tension` group + `create` leaf mirror the landed `auth`/`auth login` group/leaf; future tension reads/edits/delete (deferred, spec non-behaviors) will add sibling leaves under the same group.

---

## Branching Guidance

**Pipeline mode**: `spec/042-tension-capture/base` → `spec/042-tension-capture/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base). Cut the base from current main (all dependencies landed).

**Role-based awareness**: parallel Conductor workspaces may carry sibling specs (035/039/041 are Analyzed per STATUS). T001 makes an **additive** change to the shared `internal/apiclient.Request` (new `ContentType` field) — backward-compatible (existing reads pass `""`), but it is the one cross-package contract change here, so a sibling touching `internal/apiclient` should be aware. T002–T004 add new files (`internal/glassfrog/tension.go`, `internal/cli/tension.go`) and one new `internal/render` key — no edits to existing read commands.

---

## Phase 1: Write-body transport seam [Shared]

- [ ] **T001** [Shared] [P] Add the write-body `Content-Type` capability to `apiclient.Request` + `Execute` — header-present/absent tests; `internal/apiclient` (`client.go` field + `execute.go` set) + tests
  - **Scope**: In `internal/apiclient`, add an **additive** field `ContentType string` to `Request` (beside `Method`/`Path`/`Query`/`Body`). In `(*Client).Execute`, after `http.NewRequestWithContext` and **before** `c.httpClient.Do`, set the header only when non-empty: `if req.ContentType != "" { httpReq.Header.Set("Content-Type", req.ContentType) }`. No other behavior changes — the response path, error taxonomy (`*TransportError`/`*ResponseError`/`*DecodeError`/`*AuthError`), and the single-`Do` / always-close-body discipline are untouched. `RetryExecutor` (017) forwards `Request` unchanged, so the field rides through with no edit there. Chose the narrow field over a general `Header http.Header` bag (plan ADR-1) — generalize when a second header (`If-Match`, deferred) has a real consumer.
  - **Acceptance criteria**:
    - A `Request` with `ContentType: "application/json"` produces an outbound request whose `Content-Type` header is `application/json`
    - A `Request` with `ContentType: ""` produces an outbound request with **no** `Content-Type` header set — the existing GET reads are byte-identical (a test pins this)
    - No change to the response decode, error wrapping, or body-close behavior; existing `apiclient` tests stay green
    - `go build ./...` / `go vet ./...` clean
  - **Dependencies**: None.
  - **Plan reference**: Phase 1 (Write-body transport seam); ADR-1 (narrow `ContentType` field)
  - **Interface references**: interface-spec.md — `internal/apiclient` write-body content type
  - **Risk**: ⚠️ Additive only — do not change the descriptor's existing fields or any read's behavior (empty `ContentType` → no header). ⚠️ Set the header on the built `*http.Request` before `Do`, not in the auth transport (007 owns only `X-Auth-Token`). ⚠️ Resist the general `Header` map — narrow field per ADR-1.

## Phase 2: The tension create command [US1/US2/US3/US4]

- [ ] **T002** [Shared] [P] Add the `glassfrog.Tension` model + the create request-input shape — decode/encode round-trip tests; new `internal/glassfrog/tension.go` + tests
  - **Scope**: New `internal/glassfrog/tension.go`. Add the `Tension` response model matching the v5 `Tension` schema with explicit snake_case JSON tags, tolerant of unknown/extra fields: `ID` (`ten_…`), `Type` (`tension`), `Body`, `Status` (`unprocessed`/`processed`/`archived`), `RoleID` (`role_…`, nullable→empty), `SensedByID` (`per_…`, nullable→empty), `CreatedAt`, `UpdatedAt`, `Label` (nullable), `MeetingType` (`tactical`/`governance`/null), `ParentRoleID` (`role_…`, nullable) — nullable fields as plain strings (empty = null), mirroring `Policy.Body`/`Project.RoleID`. Add the create request-input shape encoding the nested `{ "tension": { "body": …, "label"?: …, "meeting_type"?: … } }` envelope: `body` always serialized; `label`/`meeting_type` use `omitempty` so an absent field sends nothing. **No `status`** field in the input (server auto-computes) and **no `sensed_by`** field (server derives from the token). Leaf package — imports no transport, no cobra, no `crypto`. The token is never a field (CONSTITUTION II). The create response decodes into the landed generic `Document[Tension]` — add no new envelope type.
  - **Acceptance criteria**:
    - A `{data: Tension}` body decodes into `Document[Tension]`; every field binds (snake_case tags verified — an untagged `RoleID` would silently never bind to `role_id`); a null `role_id`/`sensed_by_id`/`label`/`meeting_type`/`parent_role_id` decodes to empty string
    - Marshalling the input with body only emits `{"tension":{"body":"…"}}` (no `label`/`meeting_type`/`status`/`sensed_by` keys); with label + meeting-type emits all three under `tension`
    - Unknown/extra response fields decode cleanly (forward-compatible)
    - `internal/glassfrog` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: None (references the landed `Document[T]`, 034).
  - **Plan reference**: Phase 2 (the command — model); System Architecture (`internal/glassfrog`); conforms to 011 ADR-1
  - **Interface references**: interface-spec.md — `internal/glassfrog` tension schema + request input
  - **Risk**: ⚠️ Explicit snake_case tags on every field (encoding/json does not bridge underscores). ⚠️ `omitempty` on `label`/`meeting_type` so an absent flag sends no field; never send `status` or `sensed_by`. ⚠️ New type — do NOT grow an existing model (no read shares the tension shape). ⚠️ Nullable-as-empty-string convention, matching the landed models.

- [ ] **T003** [Shared] Add the `internal/render` `tension` key + `TensionView` + templates — golden + registry-guard tests; `internal/render` (new `tension.{full,compact}.tmpl`, `TensionView`, `ResourceTension`) + tests
  - **Scope**: In `internal/render`, add **one new** render key `tension` (single, data `glassfrog.Tension` via a `TensionView{Tension}`, mirroring the landed `PolicyView{Policy}`/`ProjectView`). Add the `ResourceTension` constant to `builtinResources` so the registry-exhaustiveness guard covers it (PR #10 `len`+comma-ok shape). Add two `//go:embed` templates: `tension.full` — `<ten_…>  [<status>]` then an indented block (`Body`, `Label`, `Sensing role`, `Sensed by`, `Meeting type`, `Parent role`, `Created`, `Updated`) rendering the free-text `body` **verbatim — never truncated or reflowed** (CONSTITUTION VI), with 019's absence-guard discipline (`{{if …}}…{{else}}(none){{end}}`) on the nullable `label`/`role_id`/`sensed_by_id`/`meeting_type`/`parent_role_id`; `tension.compact` — `<ten_…>  [<status>]  <body>`. Depends only on `internal/glassfrog` + stdlib; must not import `cli`/`apiclient`.
  - **Acceptance criteria**:
    - The `tension` key renders both `full` and `compact`; `full` shows the id, status, the verbatim body, and the label/role/sensed-by/meeting-type/parent/timestamps
    - A null `label`/`role_id`/`sensed_by_id`/`meeting_type`/`parent_role_id` renders its explicit-absence marker (`(none)`), never `<no value>` or an invented value
    - A long free-text `body` is rendered verbatim (neither truncated nor reflowed)
    - The registry-exhaustiveness guard passes with the new `tension` key carrying both formats; golden tests pin each template
    - `internal/render` imports neither `cli` nor `apiclient`; `go build`/`go vet` clean
  - **Dependencies**: T002 (the view references `glassfrog.Tension`).
  - **Plan reference**: Phase 2 (the command — render); System Architecture (`internal/render`)
  - **Interface references**: interface-cli.md — Output (`full`/`compact` shapes); interface-spec.md — `internal/render` additions
  - **Risk**: ⚠️ Add the single key only; touch no existing key/template. ⚠️ Never truncate/reflow `body` (CONSTITUTION VI). ⚠️ Explicit-absence guards for every nullable field; never invent a value. ⚠️ Add `ResourceTension` to the guarded set so exhaustiveness still holds.

- [ ] **T004** [US1/US2/US3/US4] Add the `tension create <role-id>` command — `tension` group + `create` leaf + `tensionSeam` + `validateMeetingType` + body marshal + `Execute` + render dispatch + wiring — RED-first unit tests for every branch; new `internal/cli/tension.go` + `validateMeetingType` + `Assemble()` wiring
  - **Scope**: New `internal/cli/tension.go`. `newTensionCommand(seam tensionSeam) *cobra.Command`: a guard-registered **non-runnable group** (`Use:"tension"`, non-empty `Short`, no `RunE`), built with its `create` child attached **before** registration under root (so the guard's ">=1 child" rule holds). `newTensionCreateCommand(seam tensionSeam) *cobra.Command`: a guard-registered leaf (`Use:"create <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`), declares `--body`/`--label`/`--meeting-type`; reads the persistent `--base-url`/`--output`; delegates to pure `runTensionCreate`. Define `tensionSeam` (assemble `ConnectionContext`, build the `RetryExecutor`-wrapped `*Client`, `sleep`, `resolveFormat` — the `projectsSeam` shape; prod binds `AssembleFromOS`/`NewClientFromOS`/`NewRetryExecutor`, tests bind a fake `Execute` + tripwire). Add `validateMeetingType(value string) error` mirroring `validateStatus` (empty = no constraint; a non-empty value outside `{tactical, governance}` → a `UsageError`-bound error naming the value + supported set; set sourced from the `spec.yaml` `meeting_type` enum). `runTensionCreate(cfg) (Outcome, error)`: resolve `--output` (020) **first**, then validate `--body` non-empty after `strings.TrimSpace` (else `UsageError(2)` naming `--body`, **no request**) and `--meeting-type` (else `UsageError(2)`, **no request**); assemble, build the executor, marshal `{tension:{…}}` (label/meeting-type only when `Changed()` and non-empty), and send **one** `Execute` `POST /roles/{role_id}/tensions` with `ContentType: "application/json"` (T001), the id `url.PathEscape`-d and passed through unvalidated (plan ADR-3). For `json`/`yaml`, `Execute` into a raw `json.RawMessage` and emit `{data: Tension}` verbatim via `output.RenderSuccess` (018); for `full`/`compact`, `Execute` into `Document[Tension]` and render via `renderFn(ResourceTension, TensionView{doc.Data})` (the single-read shape, mirroring `runProjectGet`). On any failure route through `reportFailure` (032). Wire `MustRegister(root, newTensionCommand(...))` in `Assemble()`. Adds no new `Outcome`/`ExitCode`. Never read `ctx.Cred.Token`.
  - **Acceptance criteria**:
    - `glassfrog tension create role_0123 --body "…"` POSTs `{"tension":{"body":"…"}}` with `Content-Type: application/json` and prints the created tension (incl. `ten_` id, server status); exits 0
    - `--label "…" --meeting-type governance` adds `label` + `meeting_type=governance` to the body; an omitted `--label`/`--meeting-type` sends no such field
    - A missing `--body`, or a `--body` that trims to empty, is a `UsageError(2)` naming `--body`, **no request sent** (transport tripwire)
    - `--meeting-type weekly` (unsupported) is a `UsageError(2)` naming the value + supported set, **no request sent** (tripwire)
    - `glassfrog tension` (no subcommand) prints help (group, no action); the registration guard accepts the group (≥1 child) and the leaf (action)
    - `*AuthError{NoCredentials}` → UsageError(2); `*AuthError{CredentialError}` → RuntimeError(1); `*TransportError` → NetworkUnavailable(6); `*ResponseError` (404 unknown role / 422 rejected body / 401/403/429) → APIError(3)/PermissionError(4)/RateLimited(5) via the shared classifier; base-URL/`--output` errors → UsageError(2)
    - A `POST` is never auto-retried on 429 (the 017 `isSafeMethod` gate) — a rate-limited capture surfaces on first occurrence (no duplicate)
    - `-o json`/`yaml` emit the raw `{data: Tension}` payload; `full`/`compact` render the `tension` projection
    - No new `Outcome`/`ExitCode`/root flag; no token in any output or request body; all branches run offline; `go build`/`go vet` clean
  - **Dependencies**: T001 (`ContentType`), T002 (`Tension` model + input shape), T003 (`tension` render key).
  - **Plan reference**: Phase 2 (the command); ADR-2 (group + leaf), ADR-3 (`validateMeetingType`, required-non-empty `--body`, id pass-through); Cross-cutting (single-resource `Document[Tension]`, no new outcomes, §133 retry)
  - **Interface references**: interface-cli.md — Surface (group + `create`), write flags, Interactions (validation order), Error Communication; interface-spec.md — `internal/cli` additions
  - **Scenario references**: tension-capture.feature: "A tension is captured against the sensing role", "A missing token fails as a not-authenticated usage error", "An unknown sensing role fails with the API status", "A rate-limited capture is surfaced, not silently re-sent", "The created tension's id is present in structured output", "A tension is captured with a label and a meeting-type", "An unsupported meeting-type is rejected as a usage error", "A missing body is rejected as a usage error", "A whitespace-only body is rejected as empty"
  - **Risk**: ⚠️ Reuse `classifyClientError`/`reportFailure`/`RetryExecutor`/`output.RenderSuccess`/`renderFn` — inline no second copy of the error chain, retry loop, or render branch. ⚠️ `validateMeetingType` mirrors `validateStatus` — a sibling validator, NOT a second copy of any set. ⚠️ Resolve `--output` and validate `--body`/`--meeting-type` BEFORE assembly so the tripwire confirms no request on rejection. ⚠️ Set `ContentType: "application/json"` on the request (T001) or the API may ignore the body → 422. ⚠️ Marshal label/meeting-type only when `Changed()` and non-empty; never send `status`/`sensed_by`. ⚠️ Build the group with its child before registering it (guard's ">=1 child" rule). ⚠️ Capture stdout/stderr in tests with a temp file, not `os.Pipe` (PR #10 LEARNINGS). ⚠️ Never read `ctx.Cred.Token`; no token in the request body.

## Phase 2 — acceptance step: Executable acceptance [Shared]

- [ ] **T005** [Shared] Make the driving scenarios pass as executable acceptance — new `internal/cli` godog suite over `tension-capture.feature`; un-`@wip` the behavioral scenarios, keep `@validation` held
  - **Scope**: Add godog step definitions for `features/tension-capture/tension-capture.feature` in a **new** `internal/cli` godog suite (e.g. `TestTensionCaptureFeatures`) whose `Paths` names **only** that feature file (LEARNINGS: a suite points at its own file, never the `features/` directory). Drive `tension create` through the shared seam over a fake base `http.RoundTripper` returning canned `POST /roles/{role_id}/tensions` responses (201 with a created tension, 404 unknown role, 429 rate-limit), plus a transport tripwire for the no-request paths (missing/blank `--body`, unsupported `--meeting-type`, missing token). Remove `@wip` from the spec-derived + architecture-informed behavioral scenarios; keep the 2 `@validation` scenarios `@wip` (held for validate). Grep existing `sc.Step(` registrations and reuse shared exit-code / stderr-substring / structured-output / no-request phrasings from the `me*`/`roles`/`projects` suites before writing new bindings; step helpers return errors, never panic.
  - **Acceptance criteria**:
    - Every non-`@validation` tension-capture scenario has an executable, passing path; `@wip` removed from them
    - The 2 `@validation` scenarios keep `@wip`
    - The 429 fake proves the capture is surfaced once and not retried (no duplicate POST); the tripwire fakes prove the no-request rejections
    - The new suite's `Paths` names only `tension-capture.feature`; all `internal/cli` godog suites run and report their own independent scenario counts
    - No real network (fake base transport / loopback only) and no real home/filesystem are touched; `go build ./...`, `go vet ./...`, and the feature suites run clean
  - **Dependencies**: T004 (the command must exist for the behavioral scenarios to be implementable).
  - **Plan reference**: Phase 2 (the command); System Architecture (single `Assemble()` wiring site); Cross-cutting (testing)
  - **Interface references**: interface-cli.md — Surface, Interactions, Error Communication
  - **Scenario references**: tension-capture.feature: all behavioral Rule-block scenarios (the 2 `@validation` scenarios stay held for validate)
  - **Risk**: ⚠️ Suite scoping — point the suite at `tension-capture.feature` only (not the directory); verify it reports its own count. ⚠️ Reuse shared step phrasings before writing new bindings; step helpers return errors, never panic (LEARNINGS). ⚠️ The 429 fake must assert exactly one POST (no retry on the non-idempotent method) so the no-double-create scenario genuinely exercises §133. ⚠️ Cover the 201, 404, 429, and no-request (tripwire) fakes so the happy, unknown-role, rate-limit, and rejection scenarios each exercise their paths.
