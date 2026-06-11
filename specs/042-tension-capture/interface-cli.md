# Interface Accord: Tension Capture — CLI

**Feature**: 042-tension-capture
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (write-body `Content-Type` seam), ADR-2 (`tension` group + `create` leaf, reserves the namespace), ADR-3 (`--meeting-type` validated locally via a new `validateMeetingType`; `--body` required non-empty; role id passed through). Output: Cross-cutting (single-resource `Document[Tension]`, reuses 018/020). Retry: Cross-cutting (silent conformance to §133 `isSafeMethod`).

---

This accord pins the operator-facing tension-capture surface: the `tension` command group, its `create` leaf, the leaf's flags, the rendered created-tension output, and exit codes. The request seam it calls through is pinned in `010/interface-spec.md` (extended by this feature's `042/interface-spec.md` for the write body); format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the CLI's **first write** — the entry point to the proposal write path; it records a tension and produces it (with its `ten_` id) so a later proposal can reference it as `tension_id`.

---

## Surface

### `glassfrog tension` — command group (no action)

A guard-registered (001), non-runnable group: it carries **no action** and parents at least one child (`create`), so `glassfrog tension` with no subcommand prints help. Mirrors the `auth` / `auth login` group/leaf shape (plan ADR-2). The group is assembled with its child **before** being registered under root, so the registration guard's ">=1 child" rule holds at attach time. The `tension` namespace is reserved for future reads/edits (`tension get`/`list`/`update`/`delete`) — out of scope here (spec non-behaviors).

### `glassfrog tension create <role-id>` — capture a tension

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage` (the leaf owns its messages). Sends `POST /roles/{role_id}/tensions` (`createTension`) and produces the created tension as a single-resource result.

**Synopsis**:
```
glassfrog tension create <ROLE_ID> --body TEXT [--label TEXT] [--meeting-type tactical|governance] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The **sensing** role the tension is captured against (`role_…`; stored by the API as `impacted_role_id`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`/`422`). |

**Write flags** (declared only on `create`):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--body` | string | — | **Required.** The tension text, sent as `tension.body`. A missing flag, or a value that is empty or only whitespace, is a usage error naming `--body` as required, **before any request** (no API call). Free text, passed through verbatim — never truncated, reflowed, or parsed (CONSTITUTION I/VI). |
| `--label` | string | — | Optional short title, sent as `tension.label`. Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); an omitted or empty `--label` is omitted from the body. Free text, passed through. |
| `--meeting-type` | string | — | Optional routing-intent hint, sent as `tension.meeting_type`. **Validated locally** before any request against the closed set (`tactical`, `governance`); an unsupported value is a usage error naming the value and the supported set, no API call (plan ADR-3, conforms to `validateStatus`). Sent only when present and non-empty. Not a meeting binding — captured tensions are not bound to a meeting. |

The leaf does **not** expose `--status` (the API auto-computes it; `archived` is set only via a future `tension update`) nor any `sensed_by`/person flag (the API derives the sensing person from the token).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the created tension is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured `{data: …}` document verbatim (018), `full`/`compact` render the human projection (019, the new `tension` render key). The raw API envelope is never emitted under a human format.

*`full`* — the created tension's detail (the new `tension` render key):
```
<ten_…>  [<status>]
  Body:          <body>
  Label:         <label | (none)>
  Sensing role:  <role_… | (none)>
  Sensed by:     <per_… | (none)>
  Meeting type:  <tactical | governance | (none)>
  Parent role:   <role_… | (none)>
  Created:       <created_at | (unknown)>
  Updated:       <updated_at | (unknown)>
```
Each nullable field (`label`, `role_id`, `sensed_by_id`, `meeting_type`, `parent_role_id`) renders through an explicit-absence guard (`{{if .X}}…{{else}}(none){{end}}`) rather than a blank; the free-text `body` is rendered verbatim, never truncated or reflowed (the policy-`Body`/project-`note` precedent, CONSTITUTION VI). On a successful create the server-computed `status` is typically `unprocessed`.

*`compact`* — `<ten_…>  [<status>]  <body>` (the detail block omitted; the `ten_` id is always present — it is the load-bearing handle a later proposal references).

## Interactions

**Dispatch**: the `create` leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing (an unknown flag, or a positional count ≠ 1, fails here); (2) `--output` resolution (020); (3) `--body` non-empty validation; (4) `--meeting-type` validation against the closed set. Resolving `--output` first keeps error precedence consistent with the reads (an invalid `--output` is reported even when another input is also invalid); the input checks are pure and pre-assembly, so any order preserves the no-request guarantee. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/038).

**Request**: on valid input, the leaf marshals the body `{"tension": {"body": …, "label"?: …, "meeting_type"?: …}}` (only the supplied fields; `--label`/`--meeting-type` omitted when absent), and sends **one** `Execute` to `POST /roles/{role_id}/tensions` with `Content-Type: application/json` (plan ADR-1). There is no walk and no pagination — a single request. The role id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (ADR-3). The `201` body `{data: Tension}` decodes into `glassfrog.Document[Tension]`; the structured path serializes the raw payload verbatim.

**Non-idempotent retry**: `create` is built on the same `RetryExecutor` as the reads, but 017's `isSafeMethod` restricts 429 auto-retry to `GET`/`HEAD`, so a `POST` surfaces the `429` on first occurrence and is **never silently re-sent** — capture cannot double-create on a rate-limit retry (silent conformance to §133, no command-side special-casing).

**Piping / scripting**: on success, stdout carries only the rendered created tension. On failure, Output-Aware Failure Rendering (032) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 042 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Tension captured | — | `Success` | 0 | — (created tension on stdout) |
| Missing / empty / whitespace-only `--body` | — (local) | `UsageError` | 2 | "`--body` is required and must not be empty"; **no request sent** |
| Unsupported `--meeting-type` value | — (`validateMeetingType`) | `UsageError` | 2 | "unsupported --meeting-type value \"…\" — supported: governance, tactical"; **no request sent** |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown sensing role (`404`), rejected body/field (`422`), or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted RFC 9457 detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Unknown flag, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. A `422` validation rejection (e.g. the server rejects the body) classifies as `APIError`(3) with the server's detail surfaced — the command adds no interpretation. The token value never appears in any message.

## Consistency Notes

- **First write, reuses the read chain** (plan System Architecture): `create` is the single-resource shape (`project`/`policy` get) run with a `POST` that carries a body. It reuses the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `RetryExecutor` (017), `output.RenderSuccess` (structured) + `renderFn`/`render.Render` (human), and 032's `reportFailure` chokepoint — all unchanged. The only net-new machinery is the write-body `Content-Type` seam (`042/interface-spec.md`, plan ADR-1) and the `tension` model + render key.
- **Group namespace, not a flat command** (plan ADR-2): `tension create` reserves a `tension <verb>` family; the write flags live only on `create`, so a future `tension get` adds its own leaf and passing `--body` to a read would be a cobra unknown-flag usage error for free (the structural guard 034/038 rely on). No `role` group is created.
- **One local validator** (plan ADR-3): `--meeting-type` is the single closed-enum input, validated by a **new `validateMeetingType`** mirroring the shared `validateStatus` (`internal/cli/status.go`) — same fail-fast shape, the meeting-type set sourced from the `spec.yaml` `meeting_type` enum. `--body`'s required-non-empty check is the spec's deliberate fail-fast (a bodyless tension is meaningless); `--label`/`--body` are free text passed through; the role id is passed through to a clean `404`/`422` (§200 principle: validate where the API would silently mislead; pass through where it reports cleanly).
- **No `--status` / no `sensed_by`** (spec non-behaviors): the API owns status (auto-computed) and the sensing person (derived from the token), so the command exposes neither — it must not claim a state or an identity the server owns.
- **Flag/command spellings** (`tension`, `create`, `--body`, `--label`, `--meeting-type`) resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. The plural-list/singular pattern of the reads does not apply — this is a verb leaf under a noun group.
- **Command conventions** follow 001/003: the group and leaf register through the fail-loud guard, are explicitly wired in `main`/`Assemble`, the leaf declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
