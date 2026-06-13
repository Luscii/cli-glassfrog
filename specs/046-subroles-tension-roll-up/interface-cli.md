# Interface Accord: Subroles Tension Roll-up — CLI

**Feature**: 046-subroles-tension-roll-up
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (a fifth read leaf `tension subroles <role-id>` under 042's `tension` group; define-session decision: a distinct verb, not a `--subroles` flag on `tension list`), ADR-2 (reuse the landed `Tension` model, `tensions` render path, and `validateTensionStatus` — zero new model/render/validator), ADR-3 (reuse the `tension list` runner shape with the request path swapped to `/roles/{role_id}/subroles/tensions`; leaf-anchor `404` surfaced verbatim, distinct from the empty-list `200`). Completeness: Cross-cutting (reuses 025 ADR-3). Dependency: builds on the landed 042 (`tension` group, model, render) and 043 (the `tensions` list render, `validateTensionStatus`).

---

This accord pins the operator-facing roll-up surface: the single leaf `tension subroles` under the `tension` group, its flags, the rendered output, completeness signalling, and exit codes. The `tension` group, the `glassfrog.Tension` model, the plural `tensions` render key, and `validateTensionStatus` are all landed (042/043); this feature attaches **one** new leaf and reuses everything below it. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`/`035`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the cross-role counterpart to `tension list` (043): where `list` surfaces one role's own tensions, `subroles` rolls up the tensions sensed across that role's **direct sub-roles** (one level, not transitive).

---

## Surface

### `glassfrog tension` — command group (no action)

The non-runnable group landed by 042 (plan ADR-1): it carries no action and parents its children (`create` from 042; `list`/`get` from 043; `update` from 044; `subroles` added here), so `glassfrog tension` with no subcommand prints help. This feature attaches one new leaf to the existing group through the registration guard (001); it does not redefine the group.

### `glassfrog tension subroles <role-id>` — roll up the tensions across a role's direct sub-roles

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage`. Reads `GET /roles/{role_id}/subroles/tensions` (`listSubrolesTensions`) and produces the tensions sensed across the anchor role's direct sub-roles as a list result. Walks to completion by default (see Interactions).

**Synopsis**:
```
glassfrog tension subroles <ROLE_ID> [--status STATUS] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The **anchor** role whose direct sub-roles' tensions to roll up (`role_…`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it. A leaf anchor (no sub-roles) or an unknown/malformed id → `404` (see Error Communication). |

**List flags** (declared on this leaf):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--status` | string | — | Filter by tension status, sent as the `status` parameter. **Validated locally** before any request against the tension status set (`unprocessed`, `processed`, `archived`); an unsupported value is a usage error naming the value and the supported set, no API call. Reuses the landed `validateTensionStatus` (043) — a new *consumer* of that set, not a new validator (plan ADR-2). Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); `--status ""` behaves as no filter. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. Sent on presence, not value (a provided `0`/negative reaches the API rather than being silently dropped). |

With no `--status`, every tension across the direct sub-roles is requested.

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml` \| a user-template ref (035). |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the roll-up walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

The roll-up reuses the **landed plural `tensions` render key** (043) unchanged — no new render artifact (plan ADR-2). `full` renders one block per tension (`<ten_…>  [<status>]  <label|—>`, then the `body` rendered **verbatim** on its own indented line — never truncated or reflowed (CONSTITUTION VI) — and an indented `sensing role: <role_…|—>` line); `compact` renders one line per tension (`<ten_…>  [<status>]  <label | one-line body summary>`). For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each tension's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope.

**Empty list** (the sub-roles exist but carry no tensions, or none matches `--status`): under `full`/`compact`, stdout is exactly the `tensions` template's empty line (`no tensions`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit `{"data": []}`, never `null`.) This `0`-exit empty success is **distinct** from a leaf-anchor `404` failure (Error Communication) — the two empty-ish outcomes never conflate.

## Interactions

**Dispatch**: the leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing; (2) `--output` resolution (020); (3) `--status` validation (the one closed-enum input — `validateTensionStatus`). Output resolution precedes status validation to keep error precedence consistent with the sibling reads — both are pure, pre-assembly checks, so either order keeps the no-request guarantee, but resolving `--output` first means an invalid `--output` is reported even when `--status` is also invalid. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/038/043).

**List completeness** (reuses 025 ADR-3 verbatim):
- **Default** — the command walks every page via `paging.All[Tension]` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more tensions exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the tensions gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero**:
  ```
  note: result is incomplete — <cause>; the tensions shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario).

**Request**: the command walks `GET /roles/{role_id}/subroles/tensions` (sending `status` only when supplied). The anchor id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (ADR-3). All reads are bodyless `GET`s — the 042 write-body `Content-Type` seam is not used here. The roll-up is **one level only**: the command issues exactly this one paginated read and makes no attempt to recurse into grand-child roles (spec non-behavior + validation scenario).

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 046 reuses `reportFailure` unchanged. The leaf **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Roll-up listed (incl. empty list — sub-roles carry no tensions) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| **Leaf anchor (no sub-roles)** or unknown role id, or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step. **No "this role has no sub-roles" interpretation** is added — the `404` is surfaced as the shared read failure (plan ADR-3, VISION Exclusion 1). |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text surfaced inside the incompleteness note (the partial set is already on stdout) |
| Unsupported `--status` value | — (`validateTensionStatus`) | `UsageError` | 2 | "unsupported --status value \"…\" — supported: archived, processed, unprocessed"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. A `GET` `429` may be auto-retried by 017. The token value never appears in any message.

## Consistency Notes

- **Cross-role counterpart to `tension list`** (plan System Architecture / ADR-1): `tension subroles` is a verb leaf under the `tension` group, the same grammar as `create`/`list`/`get`/`update`. The define session chose a distinct verb over a `--subroles` flag on `tension list` — 043's non-behavior #1 scoped `list` to a role's own tensions and declared the roll-up its own capability, so one verb = one endpoint. The verb name does **not** collide with the top-level `subroles <role-id>` command (026, `listSubroles`): the two live at different command paths (`tension subroles …` vs `subroles …`) and read intentionally rhyme ("the direct children of this role").
- **Reuse-only at the data layer** (plan ADR-2): the command reuses the landed `internal/glassfrog.Tension` model + generics (the walk decodes `Page[json.RawMessage]` (structured) / `Page[Tension]` (human)), the landed plural `tensions` render key (`ResourceTensions`/`TensionsView` + `tensions.full`/`tensions.compact`), and the landed `validateTensionStatus` set — **no new model, render resource, or validator**. This is thinner than 043, which had to *add* the `tensions` render path; 046 only adds a command leaf.
- **Path-swap is the only data difference** (plan ADR-3): the runner reuses the `tension list` shape (status validation → `paging.All` walk → render → completeness note) with the request path set to `/roles/{role_id}/subroles/tensions`. Whether that is expressed by parameterizing `runTensionList`/`tensionsConfig` with the path or by a thin sibling runner is implementation-level; parameterizing keeps the status/paging/render/error logic single-sourced.
- **Leaf-`404` vs empty-`200`** (plan ADR-3): the API answers `404` for a leaf anchor (no sub-roles); the command surfaces it through the shared chain as a non-zero read failure naming the status, with no special-case message. This stays distinct from the genuine empty-list `200` (sub-roles exist but carry no tensions), which renders the `no tensions` line and exits `0`. Both outcomes are pinned by BDD scenarios + a transport tripwire so they cannot conflate.
- **Completeness, errors, config reuse the shared seams unchanged** (plan Cross-cutting): `paging.All` + `--first-page` (016/025 ADR-3), the shared `classifyClientError`/frozen `Outcome`/`ExitCode` registry (011/015), 032's `reportFailure` chokepoint, the 035-widened render flow, the persistent `--base-url`/`--output` flags — all reused exactly as `runTensionList` uses them. Adds no new `Outcome`/`ExitCode`.
- **Flag/command spellings** (`tension subroles`, `--status`/`--first-page`/`--per-page`) resolve the define-session-confirmed surface; they are conventional and adjustable at build time without changing behavior.
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
