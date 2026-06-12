# Interface Accord: Tension Reads — CLI

**Feature**: 043-tension-reads
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (two verb leaves `tension list <role-id>` + `tension get <ten-id>` under 042's `tension` group; list-only flags structurally enforced), ADR-2 (reuse 042's `Tension` + `Document[Tension]` + singular `tension` render; add only the plural `tensions` list render), ADR-3 (`--status` validated locally via a new `validateTensionStatus` over the tension status set; ids passed through). Completeness: Cross-cutting (reuses 025 ADR-3). Dependency: builds on 042 (the `tension` group, model, singular render).

---

This accord pins the operator-facing tension read surface: the two leaves `tension list` (per-role list) and `tension get` (standalone read) under the `tension` group, their flags, the rendered output, completeness signalling, and exit codes. The `tension` group, the `glassfrog.Tension` model, and the singular `tension` render key are landed by Tension Capture (042); this feature adds the read leaves and the plural `tensions` list render. The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the read counterpart to `tension create` (042) — it surfaces existing tensions on a role and reads one by its `ten_` id.

---

## Surface

### `glassfrog tension` — command group (no action)

The non-runnable group landed by 042 (plan ADR-2): it carries no action and parents its children (`create` from 042; `list` and `get` added here), so `glassfrog tension` with no subcommand prints help. This feature attaches two new leaves to the existing group through the registration guard (001); it does not redefine the group.

### `glassfrog tension list <role-id>` — list a role's tensions

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage`. Reads `GET /roles/{role_id}/tensions` (`listRoleTensions`) and produces the tensions on that role as a list result. Walks to completion by default (see Interactions).

**Synopsis**:
```
glassfrog tension list <ROLE_ID> [--status STATUS] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The role whose tensions to list (`role_…`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**List flags** (declared only on `list`):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--status` | string | — | Filter by tension status, sent as the `status` parameter. **Validated locally** before any request against the tension status set (`unprocessed`, `processed`, `archived`); an unsupported value is a usage error naming the value and the supported set, no API call (plan ADR-3, a new `validateTensionStatus` distinct from the action/project `validateStatus` set). Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); `--status ""` behaves as no filter. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

With no `--status`, every tension on the role is requested.

### `glassfrog tension get <ten-id>` — read a single tension

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /tensions/{id}` (`getTension`) and produces the single tension with full detail.

**Synopsis**:
```
glassfrog tension get <TEN_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `TEN_ID` | string | yes | The tension to read (`ten_…`). Exactly one required (cobra `ExactArgs(1)`). Passed through unvalidated; an unknown id → `404`. |

`get` declares **no list flags**. Passing `--status`, `--first-page`, or `--per-page` to it is rejected by cobra's unknown-flag handling as a usage error before any request — this is how the spec's "filter applies only to the list" is enforced (no hand-rolled cross-combo guard; plan ADR-1).

**Inherited (persistent root) flags**, read by cobra inheritance on both leaves, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*List (`tension list`)* — the **new plural `tensions` render key** (plan ADR-2): `full` renders one block per tension (`<ten_…>  [<status>]  <label|—>`, then the `body` rendered **verbatim** on its own indented line — never truncated or reflowed (CONSTITUTION VI), matching `projects.full` and the singular `tension.full` — and an indented `sensing role: <role_…|—>` line); `compact` renders one line per tension (`<ten_…>  [<status>]  <label | one-line body summary>`). For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each tension's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed roles/domains/policies/projects walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope and **not** a decode-and-re-encode of `[]Tension`.

*Single (`tension get`)* — reuses the **landed singular `tension` render key** (042) unchanged. `full` renders the full single-tension detail:
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
Each nullable field renders through an explicit-absence guard rather than a blank; the free-text `body` is rendered verbatim, never truncated or reflowed (CONSTITUTION VI). `compact` — `<ten_…>  [<status>]  <body>` (the detail block omitted).

**Empty list** (the role carries no tensions, or none matches `--status`): under `full`/`compact`, stdout is exactly the `tensions` template's empty line (`no tensions`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: each leaf has its own `RunE` (no `len(args)` branching). Before any network call, in order: (1) cobra `Args`/flag parsing (a list-only flag on `get` fails here as an unknown flag); (2) `--output` resolution (020); (3) `--status` validation on `list` (the one closed-enum input — `validateTensionStatus`). Output resolution precedes status validation to keep error precedence consistent with the sibling reads — both are pure, pre-assembly checks, so either order keeps the no-request guarantee, but resolving `--output` first means an invalid `--output` is reported even when `--status` is also invalid. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/014/025/038).

**List completeness** (`tension list`; reuses 025 ADR-3 verbatim):
- **Default** — the command walks every page via `paging.All[Tension]` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more tensions exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the tensions gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero**:
  ```
  note: result is incomplete — <cause>; the tensions shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario). The single `tension get` read is unpaginated — no completeness signalling.

**Request**: `list` walks `GET /roles/{role_id}/tensions` (sending `status` only when supplied); `get` issues **one** `Execute` to `GET /tensions/{id}` and decodes `glassfrog.Document[Tension]`. The ids are escaped as single path segments (`url.PathEscape`) but passed through unvalidated (ADR-3). All reads are bodyless `GET`s — the 042 write-body `Content-Type` seam is not used here.

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 043 reuses `reportFailure` unchanged. Both leaves **introduce no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Tensions listed / tension read (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown role id (`list`) or tension id (`get`), or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk (`list`) | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text — "malformed paging: the cursor did not advance at page N" — surfaced inside the incompleteness note (the partial set is already on stdout) |
| Unsupported `--status` value (`list`) | — (`validateTensionStatus`) | `UsageError` | 2 | "unsupported --status value \"…\" — supported: archived, processed, unprocessed"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| List-only flag on `get`, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; these commands benefit with no edit. The token value never appears in any message.

## Consistency Notes

- **Read counterpart to `tension create`** (plan System Architecture): `tension list`/`tension get` are the structural twin of Role Projects' `projects`/`project` (038), run as verb leaves under the `tension` group rather than a plural/singular noun pair — because 042 already claimed `tension` as a group with a `create` leaf, and a bare `tension <ten-id>` would collide with subcommand dispatch (the cobra constraint 038 ADR-1 cites). They reuse the persistent `--base-url` (011) and `--output`/`-o` (020, widened by 035 to also accept a user-template ref), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), the 035-widened render flow — `resolveRenderTarget` then `aggregateRawData` (structured) / `writeHuman` over the projection (human) for the list, and `output.RenderSuccess` (structured) / `writeHuman` (human) for the single read — and 032's `reportFailure` chokepoint, all unchanged (the exact pattern 042's `runTensionCreate` and 038's `runProjectsList`/`runProjectGet` use).
- **Verb leaves under the existing group enforce list-only-ness structurally** (plan ADR-1): `--status`/`--first-page`/`--per-page` simply don't exist on `get`, so cobra's unknown-flag handling does the rejecting. Both leaves are `ExactArgs(1)` (mirrors 026's `subroles <id>`, 033/034/038's pairs, 042's `create`).
- **One local validator, a new set** (plan ADR-3): `--status` is the single closed-enum input, validated by a **new `validateTensionStatus`** over the tension status set (`unprocessed`, `processed`, `archived`), placed **with the tension command code** beside 042's landed `validateMeetingType` (NOT in the shared `internal/cli/status.go`, which owns the action/project `validateStatus`/`supportedActionStatuses`) — and **not** reusing `validateStatus`, whose vocabulary (`current`/`completed`/…) is wrong for tensions. The `role_`/`ten_` ids are free identifiers passed through to a clean `404`. (025 ADR-4 principle: validate where the API would silently mislead; pass through where it reports cleanly.)
- **No schema growth, reuses 042's model + generics** (plan ADR-2): reuses `internal/glassfrog.Tension` (added by 042) unchanged and instantiates the landed generics: the list walks `Page[json.RawMessage]` (structured) / `Page[Tension]` (human), the single read decodes `Document[Tension]`. No model edit.
- **Render keys** (plan ADR-2): the **single `get` reuses the landed singular `tension` key** (042) unchanged; the only new render artifacts are the plural **`tensions`** key — a `TensionsView` view struct (mirroring `ProjectsView`), the `ResourceTensions` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and the `tensions.full`/`tensions.compact` templates. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes). This is the plural/singular mirror of 038, with roles reversed (038 added the singular; 043 adds the plural).
- **Depends on 042** (plan Cross-cutting / Risks): the `tension` group, the `Tension` model, and the singular `tension` render are 042's; 042 is sequenced first. If 043 lands first, its contingency stands up the minimal group + model + singular render and 042 then adds only `create` — a tasks-stage branch point, not a contract change.
- **Flag/command spellings** (`tension list`, `tension get`, `--status`) resolve the spec's confirmed surface; they are conventional and adjustable at build time without changing behavior. The plural-list/singular noun pattern of 038 does not apply — these are verb leaves under a noun group (the 042 grammar).
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
