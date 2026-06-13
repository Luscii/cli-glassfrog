# Plan: Tension Update

**Feature**: 044-tension-update
**Role**: Shaper
**Inputs**: `specs/044-tension-update/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent — 042 ADR-1/2/3, 043 tension-status entries), `.score/memory/DEPRECATION.md` (no tension/042/043 deprecations), `.score/memory/LEARNINGS.md` (background); existing code in `internal/cli` (`tension.go`, `tension_reads.go`, `status.go`), `internal/glassfrog` (`tension.go`, `document.go`), `internal/apiclient` (`client.go`, `execute.go`, `retry.go`); the vendored `spec/glassfrog-api-v5.yaml` (`updateTension`, shared `TensionInput` schema)

---

## System Architecture

Tension Update is the **edit verb of the `tension` family** — a third leaf alongside the landed `create` (042) and `list`/`get` (043). Architecturally it is `runTensionGet`'s single-resource shape run with a `PATCH` that carries a partial body: resolve the output format first, run the pure precondition checks fail-fast, assemble the connection, build the retrying executor, send exactly one `PATCH /tensions/{id}`, then render the updated tension or classify the failure. It rides the proven chain end-to-end and — unlike 042 — adds **no new transport surface at all**: 042 already added `apiclient.Request.ContentType`, and `Request.Method` is a free string, so a body-bearing `PATCH` is expressible today. The new surface is one command leaf, one request-input model, and the partial-update precondition.

**Components**:

- **`internal/cli/tension.go`** (extended) — a new `update <ten-id>` leaf attached to the existing non-runnable `tension` group inside `newTensionCommand` (042 ADR-2), beside the `create`/`list`/`get` leaves. It carries the editable-field flags `--body`, `--label`, `--status`, `--meeting-type` (all optional; these live only on `update`), inherits the persistent `--base-url`/`--output`, and delegates to a pure `runTensionUpdate(cfg)` over the **landed `tensionSeam`** (`assemble`/`newClient`/`sleep`/`resolveSelection`/`readTemplateSource`) so every branch runs offline against a fake transport. The group's `Short` widens to name the edit alongside capture/reads.
- **`internal/glassfrog/tension.go`** (extended) — a new `TensionUpdateInput` request type encoding the same nested `{tension: {…}}` envelope but with **all four fields `omitempty`, including `status`** (the partial-update shape). This is a sibling of the landed `TensionInputBody`, not a reuse of it (ADR-1). The `Tension` response model, `Document[Tension]`, and the singular render are reused unchanged (011 ADR-1 — grow/reuse, never duplicate).
- **Validation (reused)** — `--status` is validated by the landed `validateTensionStatus`/`supportedTensionStatuses` (043; its DECISIONS entry already names 044 as the next consumer), and `--meeting-type` by the landed `validateMeetingType`/`supportedMeetingTypes` (042). No new validator set is introduced (ADR-2). The two new pure checks are the partial-update precondition and the blank-body-if-supplied check (ADR-3).

**Data flow**: `update <ten-id> [--body|--label|--status|--meeting-type]` → resolve render target (020/035) → (if `--body` supplied) reject a blank body → validate `--status` + `--meeting-type` against their closed sets → require at least one editable field to be sent (all fail-fast, no request) → `assemble` (009) → `newClient` (008/007) → `NewRetryExecutor` (017) → marshal `{tension:{…}}` with only the supplied fields → one `Execute` `PATCH /tensions/{ten_id}` with `Content-Type: application/json` (010, X-Auth-Token via 007's transport) → on `200` decode `Document[Tension]`; machine renders the raw `{data}` verbatim (018), human renders the `tension` view (019/035) → success (0). On any failure, `reportFailure` + `refineClientError`/`classifyClientError` (015/017) map `404`/`422`/transport/not-authenticated to the existing outcomes; no new `Outcome`/`ExitCode`.

---

## Architecture Decisions

### ADR-1: Model the partial-update body as a new `TensionUpdateInput`, not a reuse of capture's `TensionInputBody`

**Context**: The vendored `updateTension` PATCH body references the **same** `TensionInput` schema as `createTension` (`{tension: {body?, label?, status?, meeting_type?}}`, no field required). But the two operations impose different client-side contracts on that wire shape. Capture's landed `TensionInputBody` (042) is deliberately `{Body (no omitempty), Label omitempty, MeetingType omitempty}` with **no `status` field at all** — a documented invariant ("the input must not claim a state … the server owns"; status is server-computed at creation, so the type literally cannot express it). Update is the opposite: status is editable (the API allows the `archived` transition on `PATCH`), `--body` is optional (only checked when supplied), and *every* field rides only when supplied (true partial update). The shared schema must be serialized two different ways.

**Options considered**:
1. **New `TensionUpdateInput` + `TensionUpdateBody`** — a sibling type with all four fields `omitempty` (including `Status`), and a `NewTensionUpdateInput` constructor. Forks one small struct, but each input type then states an honest, single-operation contract and capture's "no status field" invariant stays byte-stable and load-bearing.
2. **Grow the shared `TensionInputBody`** — add `Status string omitempty` and make `Body` `omitempty`. One type for both writes (the 011 ADR-1 "grow not duplicate" reflex), but it erases capture's type-level guarantee that creation cannot send a server-owned `status`, and re-couples two operations whose field contracts genuinely diverge. The "grow not duplicate" precedent governs *response* models (Policy/Domain grown additively); request inputs are constructed per-operation and 042 already chose a status-free shape on purpose.

**Decision**: Option 1 — a new `TensionUpdateInput`. The partial-update body is `{Body omitempty, Label omitempty, Status omitempty, MeetingType omitempty}`; `NewTensionUpdateInput` takes the already-resolved (presence-filtered) field values and relies on `omitempty` to drop the unsupplied ones. Because the command guarantees every *supplied* value is non-empty (a blank `--body` is rejected; `--status`/`--meeting-type` are closed-enum non-empty; `--label` rides only when non-empty), `omitempty` + plain strings faithfully expresses "send only what was supplied" without pointer fields. Capture's `TensionInputBody` is left untouched.

**Consequences**: Capture's documented server-ownership invariant survives intact and 042's wire output is provably unchanged. Update gets an honest partial-update type. Cost: two small input structs share a schema — accepted, because they encode different operation contracts (status presence, body optionality). The `Tension` response model, `Document[Tension]`, and singular render are reused as-is. Cross-spec relevant (the write-path's first *edit* input shape) → record in DECISIONS.md.

### ADR-2: `update <ten-id>` as a new leaf on the existing `tension` group, reusing the landed status/meeting-type validators

**Context**: 042 ADR-2 established a non-runnable `tension` group reserving the namespace for "future reads/edits", with edit-only flags living on their own leaf; 043 added `list`/`get` under it. 043's DECISIONS entry explicitly forecast that "Tension Update 044's `--status`, which also accepts `archived`, reuse[s] this set" (`validateTensionStatus` over `{unprocessed, processed, archived}`), and 042's `validateMeetingType` covers `{tactical, governance}`. The surface is fixed as `glassfrog tension update <ten-id>` over a *tension* id (`PATCH /tensions/{id}`) — unlike `create`/`list`, which key off a *role* id.

**Options considered**:
1. **New `update <ten-id>` leaf on `newTensionCommand`, reusing the two landed validators** — `ExactArgs(1)` over the `ten_` id, edit flags declared only on this leaf, `runTensionUpdate` mirrors `runTensionGet`. Conforms to 042 ADR-2's reserved namespace and the structural flag guard (passing `--body` to `get` is already a cobra unknown-flag usage error); no new validator set.
2. **A separate top-level `tension-update` command or a new `update` noun group** — rejected: 042 ADR-2 already claimed `tension` as the verb namespace, and a parallel surface would fork the family the group exists to hold.

**Decision**: Option 1. Add `newTensionUpdateCommand(seam)` and attach it in `newTensionCommand` beside the three landed leaves; widen the group `Short` to name the edit. `--status` and `--meeting-type` are validated by the **already-landed** `validateTensionStatus` (043) and `validateMeetingType` (042) — silent conformance to active precedent, a new *consumer* of existing sets, not a new pattern or a copied set. The `ten_` id is escaped as one path segment (`url.PathEscape`) and passed through unvalidated (no local `^ten_…$` regex), so an unknown/malformed id surfaces as the API's `404`/`422` via the shared classifier (042/043 ADR-3, the §200 pass-through rule). Adds no new `Outcome`/`ExitCode`.

**Consequences**: The `tension` family gains its edit verb with the smallest surface. The edit flags are leaf-local, so cross-verb misuse stays a free cobra usage error. Both vocabularies remain single-sourced from the spec enum (no second copy to drift). The `ten_`-vs-`role_` id distinction is handled purely by which endpoint the leaf targets — no client-side id-kind check. Conforms to a recorded precedent rather than setting a new one (no DECISIONS entry needed for the namespace itself; ADR-1 and ADR-3 carry the cross-spec novelty).

### ADR-3: Require at least one editable field fail-fast, and reject a blank `--body` when supplied — both presence-aware, before any request

**Context**: The spec adds two preconditions capture did not have. (a) "An update that changes nothing is meaningless": with no editable field flag the command must reject as a usage error and send nothing. (b) A supplied `--body` must not be blank — but, unlike capture, `--body` is **optional** here, so the blank check fires only when the flag is present. Both must run before assembly so a transport tripwire can assert no request was issued. Cobra exposes per-flag `Changed()` (presence) independently of the parsed value — the codebase's settled discipline is to gate on presence, not on a value that may collide with a meaningful default.

**Options considered**:
1. **Presence-derived send-set, fail if empty; blank-body checked only when `--body` is present** — compute the fields that will actually be sent from `Changed()` + non-empty value (capture's `labelSet`/`meetingTypeSet` + `omitempty` rule, extended to body and status); if that set is empty, reject "at least one of --body/--label/--status/--meeting-type is required". Separately, if `--body` is `Changed()` and trims to empty, reject with the specific "a body cannot be blanked" message first. Catches both the no-flags case and the present-but-empty edge (`--label ""` alone) with one invariant: the `PATCH` must carry a field.
2. **Count `Changed()` flags only (ignore values)** — simpler, but `update <id> --label ""` would pass the gate yet marshal an empty `{tension:{}}` body — exactly the meaningless no-op the spec forbids.
3. **Let the server reject an empty body (422)** — rejected: the precondition is knowable locally and the spec mandates a client-side usage error with no request (§113 — validate locally where the API would otherwise be opaque or wasteful).

**Decision**: Option 1. Order: resolve `--output` → (if `--body` `Changed()`) reject a whitespace-only body as `UsageError(2)` naming `--body` → validate `--status` and `--meeting-type` against their closed sets → require the resolved send-set to be non-empty (else `UsageError(2)` naming the four flags) → only then assemble and send. The blank-body check precedes the generic precondition so `update <id> --body "   "` gets the precise "a body cannot be blanked" message rather than the generic one. All checks are pure and pre-assembly, so the no-request-on-rejection invariant holds under a tripwire transport. Presence (`Changed()`) decides what is sent; the value decides validity — consistent with capture and the cobra-presence discipline.

**Consequences**: A no-op update (no flags, or only empty-valued flags) costs no request and exits `UsageError(2)`. A blank supplied body is reported specifically and early. `--label ""`/`--status ""`/`--meeting-type ""` resolve to "no field sent" and therefore do not satisfy the precondition on their own — the spec's "changes nothing is meaningless" intent holds beyond the literal no-flags case. No field-clearing affordance is introduced (spec non-behavior): an empty value is omitted, never sent as null. Cross-spec relevant (the write-path's first *partial-update precondition*) → record in DECISIONS.md.

---

## Cross-cutting Concerns

**Non-idempotent retry (silent conformance to §133)**: `update` builds the same `NewRetryExecutor` as the reads, but 017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD` (verified — `retry.go:65`), so a `PATCH` surfaces a `429` on first occurrence and is **never silently re-sent**. The method gate makes the write safe with no command-side special-casing; a focused test pins that a `PATCH` `429` is surfaced, not retried.

**Failure mapping (no new outcomes)**: every failure routes through the landed `reportFailure` → `refineClientError`/`classifyClientError` chain. Not-authenticated → the shared fail-safe (`UsageError(2)` NoCredentials / `RuntimeError(1)` CredentialError); transport → `NetworkUnavailable(6)`; non-2xx via 015's `ExtractProblem` — `404` (unknown tension id) and `422` (rejected field/value) → `APIError(3)` with the RFC 9457 detail surfaced, `401`/`403` → `PermissionError(4)`, `429` → `RateLimited(5)`. The command adds no interpretation and never prints the token.

**Output**: the `200 {data: Tension}` decodes into the existing generic `Document[Tension]` (no per-resource envelope — §143/§254). Machine formats emit the raw `{data}` via `output.RenderSuccess`; the human path renders the landed singular `tension` view (the server's re-computed status is whatever the response carries — the command claims no authority over the final state, spec non-behavior). Buffer-then-write so a render failure leaves stdout empty and maps to `RuntimeError(1)`.

**Testing**: the seam injection keeps `runTensionUpdate` pure over a fake transport — unit tests drive happy/usage/failure branches offline; a transport tripwire pins "no request issued" for every rejection path (no-field, blank body, unsupported status, unsupported meeting-type); a godog BDD suite mirrors the driving scenarios (the per-command `*_bdd_test.go` convention). A focused marshalling test asserts that only supplied fields appear in the `{tension:{…}}` body and that no `If-Match` header is sent.

---

## Implementation Strategy

**Single phase — the `tension update` command.** Add `TensionUpdateInput` + `NewTensionUpdateInput` to `internal/glassfrog/tension.go`; add the partial-update precondition and blank-body-if-supplied checks; build `newTensionUpdateCommand` + `runTensionUpdate` in `internal/cli/tension.go` (one `Execute` `PATCH /tensions/{id}`, presence-aware body marshalling, render/classify); attach the leaf in `newTensionCommand` and widen the group `Short`; reuse the landed `validateTensionStatus`/`validateMeetingType`, `Document[Tension]`, `TensionView`, render resource, `reportFailure`, classifier, and `tensionSeam`; add BDD + unit tests. There is no transport phase — 042's `ContentType` already serves `PATCH`. The tasks skill may split the model/input from the command if it prefers, but there is no cross-phase dependency. (PR-sized.)

---

## Risks

- **A present-but-empty flag yields a no-op `PATCH`** — likelihood low once ADR-3 lands, impact medium (a wasted request that changes nothing, contradicting the spec). Mitigation: the precondition keys on the resolved send-set (presence + non-empty), so `--label ""` alone is rejected before any request; a tripwire test pins it.
- **Reusing `validateTensionStatus` drifts if the spec enum changes** — likelihood low, impact medium. Mitigation: the set is single-sourced from the vendored spec enum (043's recorded property); update reuses it rather than copying, so a spec change updates both reads and update at one site.
- **Body silently ignored without `Content-Type`** — likelihood very low (already mitigated), impact high (a `422`/empty-body surprise). Mitigation: `runTensionUpdate` always sets `ContentType: application/json` (042 ADR-1), and 042's header-present/absent test already pins the mechanism.
- **Server status recompute differs from the sent value** — likelihood expected (the API re-runs auto-computation on save), impact none by design. Mitigation: spec non-behavior — the command forwards the validated value and renders whatever the server returns, claiming no authority over the final state.

---

## What This Plan Does Not Cover

- **Protocol-level contract** — exact flag spellings, help/`Long` text, the request/response field mapping, the `Content-Type` constant, and the precise usage-error message wording are the **interface** skill's concern.
- **Executable scenarios** — Gherkin for the driving scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units within the single phase are the **tasks** skill's concern.
- **Tension soft-delete (Discard, 045), optimistic concurrency (`If-Match`/Clobbered Changes), and field-clearing-to-null** — out of scope per the spec's non-behaviors. This plan deliberately sends no `If-Match` and offers no clear-to-empty affordance; when Clobbered Changes lands, `update` opts into the shared guard rather than growing its own header mechanism (042 ADR-1 deferred exactly that generalization).
