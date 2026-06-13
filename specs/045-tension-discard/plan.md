# Plan: Tension Discard

**Feature**: 045-tension-discard
**Role**: Shaper
**Inputs**: `specs/045-tension-discard/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent — 042 ADR-1/2/3, 043 tension-status + read-leaf entries, 044 ADR-1/3), `.score/memory/DEPRECATION.md` (no tension/042/043/044 deprecations), `.score/memory/LEARNINGS.md` (background); existing code in `internal/cli` (`tension.go`, `tension_reads.go`, `me.go` — `reportFailure`/`refineClientError`, `clienterror.go`), `internal/apiclient` (`execute.go` — `Response`/`ResponseError`, `retry.go` — `isSafeMethod`), `internal/render` (`render.go`, `templates/`); the vendored `spec/glassfrog-api-v5.yaml` (`deleteTension`)

---

## System Architecture

Tension Discard is the **soft-delete verb of the `tension` family** — a fourth leaf alongside the landed `create` (042), `list`/`get` (043), and `update` (044). Architecturally it is the leanest write in the family: resolve the output format first (so a bad `--output` fails fast), assemble the connection, build the retrying executor, send exactly one **bodyless** `DELETE /tensions/{id}`, then interpret the outcome. It rides the proven chain end-to-end and adds **no transport surface and no `internal/glassfrog` model at all**: the request has no body (so `ContentType` stays `""`, unchanged from a read), and the success carries no server payload to model.

Two properties of `deleteTension` shape the flow, and both are decided below. First, success is **bodyless** — `Execute(reqCtx, req, nil)` returns a drained `*Response{StatusCode: 204}`; there is nothing to decode, so the command **synthesizes** its own result (the discarded `ten_` id + a discarded marker) and renders it through the same 018/019/020/035 flow the siblings use (ADR-3). Second, the delete is **not REST-strict idempotent** — a re-delete returns `404`, which `Execute` surfaces as a `*ResponseError{StatusCode: 404}` (the generic non-2xx error). Discard intercepts that one status via `errors.As` **before** `reportFailure` and treats it as the success end-state (ADR-2), then surfaces a one-line **advisory on stderr** saying which success it was — discarded a live tension (`204`) or found one already gone (`404`) — while keeping stdout identical (ADR-4).

**Components**:

- **`internal/cli/tension.go`** (extended) — a new `discard <ten-id>` leaf attached to the existing non-runnable `tension` group inside `newTensionCommand` (042 ADR-2), beside `create`/`list`/`get`/`update`. Unlike its siblings it declares **no editable-field flags** — it removes a tension whole — so it carries only the inherited persistent `--base-url`/`--output`. It delegates to a pure `runTensionDiscard(cfg)` over the **landed `tensionSeam`**, so every branch (success-204, success-404, transport, non-2xx, not-authenticated, bad-`--output`) runs offline against a fake transport. The group `Short` widens to name discard alongside the rest.
- **`internal/render`** (extended) — a minimal `TensionDiscardView{ID}` plus a `tension-discard.{full,compact}.tmpl` pair and a `ResourceTensionDiscard` key, so the human path renders the synthesized confirmation through `writeHuman` (035-aware) exactly as the other resources do. (Field/template wording is interface-level.)
- **No `internal/glassfrog` change, no `internal/apiclient` change** — the success has no API model to add (the synthesized result is a command-local construct), and a bodyless `DELETE` needs no new request field (042 already made `Method` a free string and `ContentType` optional). This is the first family member that touches neither package.

**Data flow**: `discard <ten-id>` → resolve render target (020/035); a bad `--output` is a fail-fast `UsageError(2)`, no request → `assemble` (009) → `newClient` (008/007) → `NewRetryExecutor` (017) → one `Execute` `DELETE /tensions/{ten_id}`, **no body, no `Content-Type`, `out == nil`** (010) → branch on the outcome:
- `err == nil` (a `204`): success — note "discarded" for the advisory.
- `errors.As(err, &respErr)` with `respErr.StatusCode == 404`: success — note "already gone" for the advisory (ADR-2).
- any other error: `reportFailure` → `refineClientError`/`classifyClientError` (015/017) maps transport → `NetworkUnavailable(6)`, `401`/`403` → `PermissionError(4)`, `429` → `RateLimited(5)`, other non-2xx → `APIError(3)`, not-authenticated → the shared fail-safe; exit non-zero.

On either success: write the advisory line to stderr (ADR-4), then render the synthesized `{data:{id,discarded}}` result on stdout — machine formats via `output.RenderSuccess` (018), human/`-o template` via `writeHuman` over `TensionDiscardView` (019/035) — and exit `Success(0)`. The `ten_` id is escaped as one path segment (`url.PathEscape`) but passed through unvalidated, so a malformed id reaches the API as a clean `404` (which is then a success — see ADR-2's consequences). Adds no new `Outcome`/`ExitCode`.

---

## Architecture Decisions

### ADR-1: `discard <ten-id>` as a new flagless leaf on the existing `tension` group

**Context**: 042 ADR-2 established a non-runnable `tension` group reserving the namespace for "future reads/edits", with command-specific flags living on each leaf; 043 added `list`/`get` and 044 added `update`. Discard maps to `DELETE /tensions/{id}` — keyed off a *tension* id like `get`/`update` (not the *role* id `create`/`list` use). Unlike every prior leaf, discard has **no editable inputs**: it deletes the whole tension, so it needs no field flags.

**Options considered**:
1. **New `discard <ten-id>` leaf on `newTensionCommand`, no field flags** — `ExactArgs(1)` over the `ten_` id, only the inherited persistent flags, `runTensionDiscard` mirrors the sibling orchestration shape. Conforms to 042 ADR-2's reserved namespace and the structural guard; the smallest possible surface.
2. **A separate top-level `tension-discard` command** — rejected: 042 ADR-2 already claimed `tension` as the verb namespace; a parallel surface would fork the family the group exists to hold (the same rejection 044 ADR-2 recorded).

**Decision**: Option 1. Add `newTensionDiscardCommand(seam)` and attach it in `newTensionCommand` beside the four landed leaves; widen the group `Short` to name discard. The leaf declares no flags, so a stray `--body`/`--status` is a free cobra unknown-flag `UsageError(2)` (the structural guard 034/038/044 rely on). The `ten_` id is escaped as one path segment and passed through unvalidated (no local `^ten_…$` regex), per 042/043/044 ADR-3 (§200 pass-through) — except that an unknown id's `404` is a *success* here, not an error (ADR-2).

**Consequences**: The `tension` family gains its delete verb with a zero-flag surface — the only leaf with no inputs beyond the id. Silent conformance to the recorded namespace precedent (no DECISIONS entry for the placement itself; ADR-2 and ADR-3 carry the novelty). A future `tension` verb still adds its own leaf the same way.

### ADR-2: Treat a `404` as success (idempotent discard), intercepting the `ResponseError` before `reportFailure`

**Context**: The spec's defining decision: discard treats a `404` as success, not a not-found failure. The vendored `deleteTension` is documented as not REST-strict idempotent — the first `DELETE` of a live tension returns `204`, a subsequent one returns `404`, and an already-discarded tension is indistinguishable from a never-existed one ("treat 404-following-204 as success"). A single CLI call issues one `DELETE` and cannot know whether a prior `204` happened, so the choice is binary: treat *every* `404` on this path as success, or as failure. `Execute` surfaces a non-2xx as a generic `*ResponseError{StatusCode, …}` (`execute.go`), and the `RetryExecutor` returns it unchanged for any status other than `429` (`retry.go:164`) — so a `404` arrives at the command as a `ResponseError` it can inspect.

**Options considered**:
1. **Intercept `404` as success** — after `Execute`, `errors.As(err, &respErr)`; if `respErr.StatusCode == http.StatusNotFound`, treat it as the success end-state (the tension is gone) instead of routing to `reportFailure`. Matches the API's own guidance, makes the command retry-safe for the agents that drive it, and the end-state is identical either way.
2. **Treat `404` as `APIError(3)`** — let `404` flow through `reportFailure` like `get`/`update` do. Surfaces a mistyped id, but re-discarding an already-gone tension then fails, contradicting the API guidance, and the command is not retry-safe (a dropped-connection retry of a succeeded delete reports failure).

**Decision**: Option 1. `runTensionDiscard` checks the returned error with `errors.As` for a `*ResponseError`; a `StatusCode` of `http.StatusNotFound` is folded into the success path (advisory "already gone", ADR-4), and **only** `404` is — `401`/`403`/`429`/other non-2xx and transport/not-authenticated all still route to `reportFailure` unchanged. The check keys on the exact status, so a permission `403` on a tension that exists is never swallowed.

**Consequences**: Discard is idempotent and retry-safe; a re-run is always a clean exit `0`. The accepted cost — discarding a mistyped or never-existed id reports success — is a recorded spec non-behavior, softened by the stderr advisory naming the "already gone" case (ADR-4) so the operator still gets a signal. This is the **first command in the codebase that treats a non-2xx as success**; every prior command routes all non-2xx through the shared classifier. Cross-spec relevant (proposals' withdraw/other deletes may face the same idempotency shape) → record in DECISIONS.md.

### ADR-3: Synthesize a command-local result for the bodyless success and route it through 020 (machine + a new human discard view)

**Context**: A successful delete (`204` or the `404`-as-success) carries **no body** — `Execute(reqCtx, req, nil)` drains and returns a bare `*Response`. But the family's contract (and the spec) is to *produce structured data* rendered in the effective format (`full`/`compact`/`json`/`yaml`), so an agent can parse `-o json`. There is no server payload to echo, so the command must construct its own result. The siblings decode `Document[Tension]` and render a `TensionView`; discard has only the id it was given.

**Options considered**:
1. **Synthesize `{data:{id,discarded}}` and route through the existing flow** — build a small command-local result from the id; for a machine format marshal it to `json.RawMessage` and hand it to `output.RenderSuccess` (018); for human/`-o template` render a new minimal `TensionDiscardView` through `writeHuman` (019/035). The result is identical for `204` and `404`.
2. **Silent success / plain-text stdout** — print nothing (or a bare line) on success. Rejected: forks the family's render contract — `-o json`/`-o yaml` would yield no parseable result and `-o <template>` (035) would not apply — exactly the "produce structured data" contract the spec preserves.
3. **Echo nothing on stdout, advisory only** — rejected for the same reason; the spec lists `full`/`compact`/`json`/`yaml` as effective formats for the result, so stdout must carry the rendered result.

**Decision**: Option 1. Add a `TensionDiscardView{ID}` + `ResourceTensionDiscard` + `tension-discard.{full,compact}.tmpl` in `internal/render`; in `runTensionDiscard`, after a success, build the result and dispatch on the resolved render target exactly as the siblings do — `RenderSuccess(machineFmt, raw)` for a built-in machine format, `writeHuman(…, ResourceTensionDiscard, …)` otherwise (buffer-then-write, so a render failure leaves stdout empty and maps to `RuntimeError(1)`). The synthesized result carries **only** the id and a discarded marker — no server-owned fields (e.g. `discarded_at`), which the bodyless response never provided (spec validation scenario). The result is byte-identical for `204` and `404` (the distinction rides stderr — ADR-4).

**Consequences**: Discard keeps the family's structured-output contract despite an empty wire — `-o json/yaml/<template>` all work. This is the **first write whose stdout result is synthesized client-side rather than decoded from the server**; it sets the pattern for any future bodyless-success write (e.g. proposal withdraw). The cost is one new render view + template pair for a one-field result — accepted, because staying on the `writeHuman`/`RenderSuccess` rails is cheaper than a bespoke output path and keeps 035 user-templates working. The exact envelope field names and template wording are interface-level. Cross-spec relevant → record in DECISIONS.md.

### ADR-4: Surface the `204`-vs-`404` distinction as an advisory on stderr, keeping stdout identical

**Context**: Both successes mean the tension is gone, but they differ in what the command *did* — it deleted a live tension (`204`) or found one already gone (`404`). The clarified spec keeps the machine-readable stdout result identical for both (ADR-3) and surfaces the distinction as an advisory note on the diagnostic channel. The codebase already routes diagnostics (failures, retry notices) to stderr via Action Transparency (031/032); a *success* advisory on stderr is the new wrinkle.

**Options considered**:
1. **One-line advisory on stderr, stdout identical** — after determining the outcome, write "discarded tension `<id>`" (`204`) or "tension `<id>` was already discarded — nothing to do" (`404`) to stderr; render the identical synthesized result on stdout; exit `0` either way. Keeps machine output stable and gives the human operator the change/no-change signal.
2. **Put the distinction in the stdout result** — add an `already_discarded` flag to the synthesized JSON. Rejected: the spec mandates stdout be identical for both, and an in-band flag the agent must branch on is more contract than the advisory warrants.
3. **No distinction (uniform success)** — the rejected clarify option; the operator could never tell whether their command changed anything.

**Decision**: Option 1. The advisory is a single `Fprintln` to `cfg.stderr`, gated on which success branch was taken; it is informational (not an error), never includes the token, and does not change the `Success(0)` exit. stdout carries only the synthesized result.

**Consequences**: The operator learns whether the discard was effective without the machine contract drifting; pipelines that read stdout see a stable result and can ignore stderr. Establishes that a success path may write an advisory to stderr (a small extension of the failures-only stderr convention) — feature-local, no DECISIONS entry. A test pins that the `404` branch writes the "already gone" advisory *and* exits `0` with the standard result (no not-found error leaks — spec validation scenario).

---

## Cross-cutting Concerns

**Failure mapping (no new outcomes)**: every failure except `404` routes through the landed `reportFailure` → `refineClientError`/`classifyClientError` chain (`me.go`/`clienterror.go`): not-authenticated → the shared fail-safe (`UsageError(2)` NoCredentials / `RuntimeError(1)` CredentialError); transport → `NetworkUnavailable(6)`; `401`/`403` → `PermissionError(4)`; `429` → `RateLimited(5)`; other non-2xx → `APIError(3)` with the RFC 9457 detail surfaced. `404` alone is diverted to success *before* this chain (ADR-2). The command adds no interpretation and never prints the token.

**Non-idempotent retry (silent conformance to §133)**: `discard` builds the same `NewRetryExecutor` as the rest, but 017's `isSafeMethod` restricts auto-retry-on-`429` to `GET`/`HEAD` (`retry.go:65`), so a `DELETE` `429` surfaces once and is never silently re-sent — discard cannot double-fire. (Re-firing would be harmless anyway given ADR-2, but the gate keeps behavior uniform with `create`/`update`.)

**Output**: success renders through the same buffer-then-write path as the siblings — `output.RenderSuccess` for a built-in machine format, `writeHuman` for human/`-o template` — over the *synthesized* `{data:{id,discarded}}` result (ADR-3), so a render failure leaves stdout empty and maps to `RuntimeError(1)`. The stderr advisory (ADR-4) is written before the stdout render and is independent of format.

**Testing**: the seam injection keeps `runTensionDiscard` pure over a fake transport. Unit tests drive every branch offline — `204` success, `404`-as-success (asserting exit `0`, the standard stdout result, and the "already gone" stderr advisory), `403`/`429`/other non-2xx failure, transport failure, not-authenticated, and a bad `--output` (a fail-fast `UsageError(2)` with a transport tripwire confirming **no request**). A focused test pins that the `DELETE` carries no body and no `Content-Type`. A godog BDD suite mirrors the driving scenarios (the per-command `*_bdd_test.go` convention).

---

## Implementation Strategy

**Single phase — the `tension discard` command.** Add `TensionDiscardView` + `ResourceTensionDiscard` + `tension-discard.{full,compact}.tmpl` to `internal/render`; build `newTensionDiscardCommand` + `runTensionDiscard` in `internal/cli/tension.go` (resolve `--output` → assemble → one bodyless `DELETE /tensions/{id}` with `out == nil` → branch: `204` success / `404`-as-success / `reportFailure`; on success write the stderr advisory and render the synthesized result); attach the leaf in `newTensionCommand` and widen the group `Short`; reuse the landed `tensionSeam`, `resolveRenderTarget`, `writeHuman`, `output.RenderSuccess`, `reportFailure`/`refineClientError`/`classifyClientError`, and `NewRetryExecutor`; add BDD + unit tests. There is **no transport phase** (bodyless `DELETE` needs no `ContentType`) and **no `internal/glassfrog` change** (the result is synthesized client-side). The tasks skill may split the render view/template from the command if it prefers, but there is no cross-phase dependency. (PR-sized.)

---

## Risks

- **`404`-as-success masks a mistyped id** — likelihood expected (typos happen), impact low-to-medium (a wrong id reports success rather than surfacing the typo). Mitigation: accepted per spec non-behavior; the stderr advisory ("already discarded — nothing to do", ADR-4) gives the operator a change/no-change signal, and a genuine permission problem returns `403` (not `404`), so it is never swallowed.
- **The `404` interception is too broad and swallows a real not-found on the wrong path** — likelihood low, impact medium. Mitigation: the `errors.As` check keys on the **exact** `StatusCode == 404` of a `*ResponseError` from the one `DELETE` this command issues; `401`/`403`/`429`/other route to `reportFailure` unchanged. A test pins that only `404` is folded into success.
- **The synthesized result claims a state the server didn't confirm** — likelihood low, impact low. Mitigation: a `204`/`404` *is* the server's confirmation the tension is gone; the result carries only the id + a discarded marker (no fabricated `discarded_at`), pinned by a validation scenario.
- **A success advisory on stderr is mistaken for an error by a caller** — likelihood low, impact low. Mitigation: the advisory is informational and the exit code is `0`; callers keying on the exit code (the documented contract) are unaffected, and stdout carries the stable result.

---

## What This Plan Does Not Cover

- **Protocol-level contract** — the exact flag/help/`Long` text, the synthesized result's envelope field names (`{data:{id,discarded}}`), the advisory message wording for the `204` vs `404` cases, and the discard template content/names are the **interface** skill's concern.
- **Executable scenarios** — Gherkin for the driving scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units within the single phase are the **tasks** skill's concern.
- **Restore / un-discard, proposal cascade, optimistic concurrency (`If-Match`/Clobbered Changes)** — out of scope per the spec's non-behaviors. The API exposes no un-delete, the soft-delete leaves associated proposals in place, and discard sends no `If-Match` (last-write-wins); when Clobbered Changes lands, the write path opts into the shared guard rather than growing its own (042 ADR-1 / 044 deferred exactly that).
