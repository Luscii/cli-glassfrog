# Interface Accord: Tension Discard — CLI

**Feature**: 045-tension-discard
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (`discard <ten-id>` flagless leaf on 042's `tension` group; id passed through), ADR-2 (`404` treated as SUCCESS — intercept the `ResponseError` before `reportFailure`, only `404`), ADR-3 (synthesize `{data:{id,discarded}}` and route through 018/019/020/035; new `TensionDiscardView` + `tension-discard.{full,compact}.tmpl`; no `glassfrog` model, no transport change), ADR-4 (`204`-vs-`404` advisory on stderr, stdout identical). Retry: Cross-cutting (silent conformance to §133 `isSafeMethod` — `DELETE` not auto-retried on `429`).

---

This accord pins the operator-facing tension-discard surface: the new flagless `discard` leaf under the `tension` group, the bodyless `DELETE`, the synthesized discard result, the `204`-vs-`404` stderr advisory, and exit codes. The `tension` group, the registration guard, format selection/rendering (`018`/`019`/`020`/`035`), the request seam (`010`, incl. the optional `ContentType` 042 added — here left empty), the shared failure chokepoint (`reportFailure`/`classifyClientError`, `015`/`031`/`032`), and the frozen `Outcome`/`ExitCode` registry (`004`/`011`) are all landed; this feature adds **one leaf** and **one minimal render key**, and reuses everything else. The resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the **soft-delete verb of the `tension` family** — it removes one tension by id, and is the first command that treats a non-2xx (`404`) as success and synthesizes its own stdout result for a bodyless response.

---

## Surface

### `glassfrog tension` — command group (no action)

The non-runnable group landed by 042 (plan ADR-2): it carries no action and parents its children (`create` from 042; `list`/`get` from 043; `update` from 044; `discard` added here), so `glassfrog tension` with no subcommand prints help. This feature attaches one new leaf to the existing group through the registration guard (001); it does not redefine the group. The group `Short` widens to name discard alongside the rest (a `Short` that omitted it would make `glassfrog tension --help` misleading).

### `glassfrog tension discard <ten-id>` — soft-delete an existing tension

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `DELETE /tensions/{id}` (`deleteTension`) and produces a synthesized discard result.

**Synopsis**:
```
glassfrog tension discard <TEN_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `TEN_ID` | string | yes | The tension to discard (`ten_…`), used as the `DELETE /tensions/{id}` path id. Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it — and an unknown id's `404` is a **success** here (idempotent discard, plan ADR-2), not an error. |

**Editable-field flags**: **none.** Discard removes a tension whole — there is nothing to edit. It is the only `tension` leaf with no flags of its own; a stray `--body`/`--status`/`--label`/`--meeting-type` is a cobra unknown-flag usage error for free (the structural guard). The leaf exposes no confirmation/`--force`/`--yes` flag (spec non-behavior — the CLI is non-interactive and agent-driven) and no restore/un-discard affordance (the API has no un-delete).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 (widened by 035) | `full` (default) \| `compact` \| `json` \| `yaml`, or a user-template ref (`@file`/`-`). |

**Output** (success, stdout): a **synthesized** discard result — the command has no server body to echo, so it builds the result client-side from the supplied id (plan ADR-3). It is rendered by Output Format Selection (020, widened by 035) in the effective format, and is **byte-identical whether the API answered `204` or `404`** (the distinction rides stderr — see Interactions). The result carries **only** the id and a discarded marker — never a server-owned field (e.g. `discarded_at`), which the bodyless response never provided.

*`json`* / *`yaml`* — the synthesized `{data: …}` document, serialized verbatim (018):
```json
{
  "data": {
    "id": "ten_0123456789abcdef0123456789abcdef",
    "discarded": true
  }
}
```

*`full`* / *`compact`* (or a selected user template, 035) — the human projection (019) over the new `tension-discard` render key:
```
<ten_…>  [discarded]
```
`full` and `compact` render the same single confirmation line (there is nothing more to show); a user template receives the `TensionDiscardView` (`{ID}`). The raw API envelope is never emitted under a human format (there is none).

The field names (`id`, `discarded`), the render key (`tension-discard`), the template layout, and the advisory wording (below) are conventional Crafter choices that realize the spec's confirmed surface — adjustable at build time without changing behavior.

## Interactions

**Dispatch**: the `discard` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here); (2) `--output` resolution (020/035) — a present-but-invalid selector, or an unparseable/empty user-template source, fails fast as a usage error with **no request**. There are no input-field checks (the leaf has no editable flags). Resolving `--output` first keeps error precedence consistent with the siblings; a transport tripwire asserts the no-request guarantee on the bad-`--output` path (per 042/043/044).

**Request**: on valid input, the leaf assembles the connection and sends **one** `Execute` to `DELETE /tensions/{id}` with **no body and no `Content-Type`** (the optional `ContentType` seam 042 added is left empty — bodyless, unchanged from a read) and **`out == nil`** (there is no 2xx body to decode; 010 drains it). There is no walk and no pagination. **No `If-Match` header is sent** (optimistic concurrency is Clobbered Changes — deferred; last-write-wins). The `ten_` id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (plan ADR-1).

**Outcome interpretation** (plan ADR-2/ADR-4): the leaf branches on the `Execute` result —
- **`err == nil`** (a `2xx`, i.e. `204`): success. Write the advisory `discarded tension <ten_…>` to **stderr**.
- **`errors.As(err, &respErr)` with `respErr.StatusCode == 404`**: success (the tension is gone — already discarded or never existed; a single call cannot tell, and the end-state is identical). Write the advisory `tension <ten_…> was already discarded — nothing to do` to **stderr**.
- **any other error**: route to the shared `reportFailure` (see Error Communication).

On either success branch, the leaf then renders the synthesized result on stdout (machine: `output.RenderSuccess`; human/`-o template`: `writeHuman` over `TensionDiscardView`, buffer-then-write) and exits `Success` (0). The stderr advisory is informational, never an error, never includes the token, and does not change the exit code; stdout is identical across the two branches.

**Non-idempotent retry**: `discard` is built on the same `RetryExecutor` as the rest, but 017's `isSafeMethod` restricts `429` auto-retry to `GET`/`HEAD`, so a `DELETE` surfaces a `429` on first occurrence and is **never silently re-sent** (silent conformance to §133). (Re-firing would be harmless given the idempotent `404` handling, but the gate keeps behavior uniform with `create`/`update`.)

**Piping / scripting**: on success, **stdout** carries only the rendered (synthesized) discard result, identical for `204`/`404`; the change-vs-no-change signal is on **stderr**, so a pipeline reading stdout sees a stable result and can ignore stderr. On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 045 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic / output |
|---|---|---|---|---|
| Tension discarded (live tension, `204`) | — | `Success` | 0 | synthesized result on stdout; **stderr advisory** "discarded tension `<ten_…>`" |
| Tension already gone (`404`) | `*ResponseError{404}` — **intercepted as success** (plan ADR-2) | `Success` | 0 | **identical** synthesized result on stdout; **stderr advisory** "tension `<ten_…>` was already discarded — nothing to do" — **no not-found error** |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Not permitted to delete (`401`/`403`) | `*ResponseError` (→ `*ProblemError`, 015) | `PermissionError` | 4 | names the HTTP status + extracted RFC 9457 detail; next step per class |
| Rate limited (`429`) | `*ResponseError` (→ `*ProblemError`, 015) | `RateLimited` | 5 | names `429` + detail; `DELETE` is not auto-retried (§133) |
| Other non-2xx (`4xx`/`5xx`, not `404`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` | 3 | names the HTTP status + detail |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector / unparseable user template | `*output.FormatError` / template error (020/035) | `UsageError` | 2 | names the bad format value + the valid names (or the template parse failure); **no request sent** |
| Render of the synthesized result fails | render error (018/019) | `RuntimeError` | 1 | buffer-then-write leaves stdout empty; token-free |
| Unknown flag, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

`404` is the **only** non-2xx not present as a failure row — it is folded into success above (plan ADR-2), and the interception keys on the exact status, so a `403` on an existing tension is never swallowed. Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The token value never appears in any message.

## Consistency Notes

- **Soft-delete verb on the reserved family** (plan ADR-1): `discard` is the fourth addition under 042's `tension` group, after 043's `list`/`get` and 044's `update`. It is the only leaf with **no flags of its own**, so any field flag is a cobra unknown-flag usage error for free (the structural guard 034/038/042/043/044 rely on). No `tension`-group redefinition; no `role` group.
- **`404` as success — a deliberate family exception** (plan ADR-2): every other `tension` leaf (and every prior command) routes all non-2xx through the shared classifier; discard alone intercepts `404` as the success end-state because `deleteTension` is documented not REST-strict idempotent ("treat 404-following-204 as success") and a single call cannot distinguish already-discarded from never-existed. This makes re-discard retry-safe; the accepted cost (a mistyped id reports success) is a spec non-behavior, softened by the stderr advisory.
- **Synthesized result, not a decoded one** (plan ADR-3): unlike `create`/`get`/`update` (which decode `Document[Tension]` and render the `tension` key), discard has no server body and builds `{data:{id,discarded}}` itself, rendered through the same 018/020/035 flow over a new minimal `tension-discard` render key — so `-o json/yaml/<template>` all work. The result carries no server-owned fields (no `discarded_at`), since none were returned.
- **Change signal on stderr, stable result on stdout** (plan ADR-4): the `204`-vs-`404` distinction is a one-line **stderr** advisory; stdout is byte-identical for both. This is a small extension of the failures-only stderr convention (031/032) — the first command to write a *success* advisory to stderr — chosen so machine consumers see a stable result while a human still learns whether the discard changed anything.
- **No new machinery** (plan System Architecture): discard adds **no** transport surface (bodyless `DELETE`, `ContentType` empty), **no** `glassfrog` model (the result is CLI-side), and **no** new `Outcome`/`ExitCode` — it reuses `apiclient.Request`/`Execute` (010), the `tensionSeam`, `resolveRenderTarget`/`writeHuman`/`output.RenderSuccess` (020/035), and `reportFailure`/`classifyClientError`/the frozen registry (011/015/032). The only net-new surface is the `discard` leaf and the `tension-discard` render key (view + two templates). It is the first family member touching neither `internal/glassfrog` nor `internal/apiclient`.
- **Flag/command spellings** (`tension`, `discard`) and the result/advisory wording resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. No `accords/` directory exists, so there are no cross-spec accord patterns to align against — this conforms to the in-repo CLI precedent set by 042/043/044.
