# Interface Accord: Output-Aware Failure Rendering — CLI

**Feature**: 032-output-aware-failure-rendering
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-1 (format-aware chokepoint `reportFailure`), ADR-2 (Diagnostic→envelope mapping; `next_step` field), ADR-3 (channel split + command-execution-only scope; partial-walk stays stderr), ADR-4 (kind map; body-when-valid).

---

This accord pins the **operator-facing failure surface**: which channel a command-execution failure lands on under each `--output` format, the shape of the structured error document an agent parses, and the exit code each failure carries. 032 **adds no flag and no command** — it changes only what an *existing* failure puts on stdout/stderr when `json`/`yaml` is the resolved format. It closes the interim gap 020 documented (failures under a structured format kept plain-text). The Go package contract (the `output.ErrorDetail` field and the `internal/cli` mapping/dispatch symbols) is pinned in `interface-spec.md`. The literal envelope field names and channel/exit-code contract are the contract here; usage/message wording is kept consistent with the spec but is an implementation detail.

---

## Surface

### No new flag or command

032 reuses the `--output` flag and the four format values 020 owns (`full` \| `compact` \| `json` \| `yaml`, case-insensitive). It changes the **failure** rendering for the structured values; the human values keep today's behavior byte-for-byte.

### Structured failure document (`json` / `yaml`)

When a command-execution failure occurs and the resolved format is `json` or `yaml`, stdout carries **one** 018 unified error envelope (and nothing else). The envelope is `{ "error": { … } }` with these fields (listed below in field-declaration order — which the JSON render preserves; the YAML render emits keys alphabetically, so YAML consumers should rely on the keys, not their order. Absent fields are omitted, never null-keyed):

| Field | Type | Presence | Source |
|---|---|---|---|
| `message` | string | always | the diagnostic's **cause** (`Diagnostic.Cause`, 031) — a human-readable, token-free description |
| `next_step` | string | when the diagnostic carries one | the diagnostic's **next step** (`Diagnostic.NextStep`, 031) — the recovery action, surfaced as its own parseable key (NEW field; see `interface-spec.md`) |
| `kind` | string | always | the lowercased category token mapped 1:1 from the diagnostic's `Outcome` category (ADR-4): `usage` \| `runtime` \| `network` \| `api` \| `permission` \| `rate-limit` |
| `status` | integer | only for a non-2xx API failure | the HTTP status from the wrapped `*ResponseError` |
| `body` | object/array | only when the API returned a body **and** it is valid JSON | the raw API error body, nested verbatim as structured data (not a quoted string); omitted when absent or not valid JSON (ADR-4) |

**Example** (`--output json`, a `403` with an API body):

```json
{
  "error": {
    "message": "You are not a member of this circle",
    "next_step": "check that your identity has the required role membership or permission for this resource",
    "kind": "permission",
    "status": 403,
    "body": { "type": "about:blank", "title": "Forbidden", "detail": "You are not a member of this circle" }
  }
}
```

**Example** (`--output yaml`, a transport failure — no API payload; YAML keys are emitted alphabetically):

```yaml
error:
  kind: network
  message: 'could not reach the API: connection refused'
  next_step: check connectivity; the API may be unreachable
```

### Human failure line (`full` / `compact`)

Unchanged from today: stderr carries `renderDiagnostic(d)` — the cause alone when there is no next step, else `"<cause> — <next step>"`. stdout stays empty. `full` is the byte-stable default, so a caller that omits `--output` sees exactly today's failure output.

---

## Interactions

**Channel split**: under `json`/`yaml` the envelope is the **sole** content of stdout, so an agent parses stdout with the same parser whether the command succeeded (the result document) or failed (the error envelope) — never a bare-text line interleaved into a structured stream. Under `full`/`compact` the failure stays on stderr and stdout is empty.

**Scope — command-execution failures only**: 032 renders failures that arise *after a command runs* (transport, decode, typed-API, and pre-request connection/auth failures — the failures that today reach the shared failure chokepoint). The following keep their existing plain-text paths and are **not** wrapped in the envelope:
- **Usage errors** (unknown command, unknown/missing/invalid flag or positional) — the `dispatch.go` plain-text path; they fail before a command executes in the resolved format.
- **The invalid-selector usage error** (`--output xml`) — owned by 020; the format itself is invalid, so there is nothing valid to render into.

**Piping / scripting**: `glassfrog … --output json 2>/dev/null` yields a parseable document on success or failure; the exit code (below) is the authoritative success/failure signal in every format.

## Error Communication

Per resolved format × failure family — stdout, stderr, and exit code (exit codes are 004's, **unchanged**: 1 Runtime, 3 API, 4 Permission, 5 RateLimited, 6 NetworkUnavailable):

| Failure | `full` / `compact` | `json` / `yaml` | Exit code |
|---|---|---|---|
| Transport failure (unreachable API) | cause+next-step on stderr | `error` envelope on stdout (`kind: network`, no `status`/`body`) | `6` |
| Typed API error with a JSON body (e.g. `403`, `500`) | cause+next-step on stderr | envelope on stdout (`kind` per status, `status`, `body` verbatim) | `4` / `3` (per status) |
| `401`/`403` permission | cause+next-step on stderr | envelope (`kind: permission`, `next_step` = verify token / check membership) | `4` |
| `429` rate-limited (after 017 exhausts retries) | cause+next-step on stderr | envelope (`kind: rate-limit`, `next_step` = wait for the reset window and retry) | `5` |
| Decode failure (undecodable 2xx body) | cause+next-step on stderr | envelope (`kind: api`, no `body`) | `3` |
| Internal-error fallback (unmappable failure, no next step) | cause on stderr (no next step) | envelope (`kind: runtime`, `next_step` **omitted**) | `1` |
| API body present but **not** valid JSON | cause+next-step on stderr | envelope **without** `body` (message+kind+status still rendered) | per status |
| Structured render cannot complete (genuinely un-encodable envelope) | n/a | **nothing** on stdout (buffer-then-write); token-free render error on stderr | `1` (Runtime) |
| Mid-walk failure **with partial data** (paginated reads) | partial document on stdout + incompleteness note on stderr | partial `{ "data": […] }` document on stdout + plain-text incompleteness note on **stderr** | non-zero (per the mid-walk cause) |

- The **partial-walk** row is the one place a structured run emits text on the secondary channel: the partial structured document already occupies stdout, so the incompleteness signal rides stderr (ADR-3) and the exit code carries the failure. A mid-walk failure with **no** records gathered is a clean failure and renders the envelope on stdout like any other.
- The **exit code is identical across formats** — 032 changes presentation, never the 004 code. Rendering and code derive from the same `Diagnose` value, so they cannot disagree.
- No secret (API token, auth header) appears in any rendered failure, in any format — the diagnostic is already token-free (031) and the envelope renderer adds nothing (018).

## Consistency Notes

- **Supersedes 020's interim-gap note** (`020/interface-cli.md` Error Communication, the "Read fails … existing cause-plus-next-step message (unchanged)" row + the "Interim failure-rendering gap" bullet): with 032, a command-execution failure under `json`/`yaml` is now the envelope on stdout, not plain text. 020's own invalid-selector usage error is unaffected (still plain text, 020's).
- **Pairs with `interface-spec.md`**: that file pins the `output.ErrorDetail.NextStep` field + tag, the `internal/cli` `reportFailure` signature, and the `errorEnvelopeFor`/`kind` mapping. This file pins what the operator/agent sees on each channel.
- **Conforms to 018** (`output.ErrorEnvelope`/`RenderError`): 032 reuses the envelope shape and encoder unchanged except for the additive `next_step` field; the "one complete document per channel, never a fragment" guarantee is preserved on the failure path.
- **Conforms to 031** (`Diagnose`): cause, next step, and category all come from the one normalized diagnostic; 032 renders, it does not classify.
- **Conforms to 004** (`classifyClientError`/`ExitCode`): no new exit code; the returned `Outcome` is unchanged from the pre-032 chokepoint.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against.
