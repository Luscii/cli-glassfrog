# Interface Accord: Subrole Filler Roll-up — CLI

**Feature**: 051-subrole-filler-roll-up
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (a distinct role-keyed read leaf `ExactArgs(1)`, NOT a `subroles` subcommand of the positional-bearing `actors` command; the 026/050 own-command precedent), ADR-2 (reuse the landed `Actor` model, `actors` render path, and `validateKind` — zero new model/render/validator; surface `--kind` only), ADR-3 (reuse the `runActorsListWalk` runner shape with the request path swapped to `/roles/{role_id}/subroles/actors`; leaf-anchor `404` surfaced verbatim, distinct from the empty-list `200`; one level only). Completeness: Cross-cutting (reuses 025 ADR-3). Dependency: builds on the landed 048 (`actors` command, `Actor` model, `ResourceActors` render path, `validateKind`) and mirrors the landed 046 roll-up.

---

This accord pins the operator-facing roll-up surface: the single role-keyed leaf, its flags, the rendered output, completeness signalling, and exit codes. The `glassfrog.Actor` model, the plural `actors` render key, and `validateKind` are all landed (048); this feature adds **one** new command leaf and reuses everything below it. The request seam it calls through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`/`035`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). This is the cross-role counterpart to Actor Directory (048): where `actors --role-id <id>` surfaces the actors filling **one** role, this rolls up the actors filling that role's **direct sub-roles** (one level, not transitive). It is the actor-shaped twin of Subroles Tension Roll-up (046).

---

## Surface

### `glassfrog subrole-actors <role-id>` — roll up the actors filling a role's direct sub-roles

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`, a non-empty `Short`, and `SilenceErrors`/`SilenceUsage`. Reads `GET /roles/{role_id}/subroles/actors` (`listSubrolesActors`) and produces the actors filling the anchor role's direct sub-roles as a list result. Walks to completion by default (see Interactions). It is its **own** top-level read leaf (plan ADR-1) — **not** `actors subroles …`: the landed `actors` command is a runnable, positional-bearing leaf (049 grows it to an optional positional), so hosting a subcommand under it would force the runnable-parent-with-children shape the codebase's optional-positional discipline avoids (025 ADR-1). It stands as its own command exactly as `subroles <role-id>` (026) stands beside `roles` and `assignments <actor-id>` (050) stands beside `actors`.

**Synopsis**:
```
glassfrog subrole-actors <ROLE_ID> [--kind KIND] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The **anchor** role whose direct sub-roles' fillers to roll up (`role_…`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it. A leaf anchor (no sub-roles) or an unknown/malformed id → `404` (see Error Communication). |

**List flags** (declared on this leaf):

| Flag | Type | Default | Description |
|---|---|---|---|
| `--kind` | string | — | Filter by actor type, sent as the `kind` parameter. **Validated locally** before any request against the closed actor-kind set (`human`, `agent`); an unsupported value is a usage error naming the value and the supported set, no API call. Reuses the landed `validateKind` (048) — a new *consumer* of that set, not a new validator (plan ADR-2). Sent only when present and non-empty (`cmd.Flags().Changed` AND non-empty); `--kind ""` behaves as no filter. |
| `--first-page` | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. Sent on presence, not value (a provided `0`/negative reaches the API rather than being silently dropped). |

With no `--kind`, every actor filling the direct sub-roles is requested. The endpoint offers **no** other query filter — there is deliberately no `--role-id` and no `--query` here (those are 048's directory filters over `/actors`; `listSubrolesActors` accepts only `kind` + pagination — spec non-behavior).

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml` \| a user-template ref (035). |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the roll-up walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

The roll-up reuses the **landed plural `actors` render key** (048) unchanged — no new render artifact (plan ADR-2). Every row is the same `glassfrog.Actor` shape (homogeneous):
- `full` — one block per actor:
  ```
  <per_… | agt_…>  [<kind>]
    Name:  <name>
  ```
- `compact` — one line per actor: `<per_… | agt_…>  [<kind>]  <name>`

The `id` prefix (`per_`/`agt_`) and the `kind` badge let the operator tell a person from an agent and carry the id forward (e.g. into the 049 actor drill-in). For `json`/`yaml`, the walked list emits the **aggregated `{data:[…]}` document** built from each actor's raw bytes (per-page `meta` dropped) via `aggregateRawData` — the landed walked-list pattern (walk `Page[json.RawMessage]`), **not** a single page's envelope and **not** a decode-and-re-encode of `[]Actor`.

The rows are **bare actor records** — `id`/`name`/`kind` (the `/subroles/actors` shape; cross-linked assignments excluded). The assignment fields `focus`/`elected_until` are **not** projected — that is the assignment-shaped read of Role Fillers (047), not this roll-up (spec non-behavior + validation scenario).

**Empty list** (the sub-roles exist but carry no fillers, or none matches `--kind`): under `full`/`compact`, stdout is exactly the `actors` template's empty line (`no actors`) and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit `{"data": []}`, never `null`.) This `0`-exit empty success is **distinct** from a leaf-anchor `404` failure (Error Communication) — the two empty-ish outcomes never conflate.

## Interactions

**Dispatch**: the leaf has its own `RunE`. Before any network call, in order: (1) cobra `Args`/flag parsing; (2) `--output` resolution (020); (3) `--kind` validation (the one closed-enum input — `validateKind`). Output resolution precedes kind validation to keep error precedence consistent with the sibling reads — both are pure, pre-assembly checks, so either order keeps the no-request guarantee, but resolving `--output` first means an invalid `--output` is reported even when `--kind` is also invalid. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/048).

**List completeness** (reuses 025 ADR-3 verbatim, as 046/048):
- **Default** — the command walks every page via `paging.All[Actor]` (016) and renders the complete set. The `kind` filter is carried on **every** page request, not just the first.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more actors exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the actors gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the actors shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario).

**Request**: the command walks `GET /roles/{role_id}/subroles/actors` (sending `kind` only when supplied). The anchor id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (ADR-3). All reads are bodyless `GET`s. The roll-up is **one level only**: the command issues exactly this one paginated read and makes no attempt to recurse into grand-child roles (spec non-behavior + validation scenario).

**Piping / scripting**: on success, stdout carries only the rendered result (or the empty line). On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout**, while human (`full`/`compact`) failures write the diagnostic to **stderr**. The two incompleteness notes stay on **stderr** in every format — a partial `{data:[…]}` document already occupies stdout (032's one-document-per-channel rule).

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 051 reuses `reportFailure` unchanged. The leaf **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Roll-up listed (incl. empty list — sub-roles carry no fillers) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| **Leaf anchor (no sub-roles)** or unknown role id, or other non-2xx (`401`/`403`/`429`/`4xx`/`5xx`) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step. **No "this role has no sub-roles" interpretation** is added — the `404` is surfaced as the shared read failure (plan ADR-3, VISION Exclusion 1). |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Malformed paging mid-walk | `*MalformedPageError` (016) | `RuntimeError` | 1 | the walker's own text surfaced inside the incompleteness note (the partial set is already on stdout) |
| Unsupported `--kind` value | — (`validateKind`) | `UsageError` | 2 | "unsupported --kind value \"…\" — supported: agent, human"; no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| Wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. A `GET` `429` may be auto-retried by 017. The token value never appears in any message.

## Consistency Notes

- **Cross-role counterpart to the directory's `--role-id` filter** (plan System Architecture / ADR-1): `subrole-actors <role-id>` rolls up the actors across a role's **direct sub-roles**, where `actors --role-id <id>` (048) lists the actors filling **one** role. It is the actor-shaped twin of `tension subroles <role-id>` (046) and the same one-level roll-up family as 026's `subroles`.
- **Own command, not a subcommand of `actors`** (plan ADR-1): 046's `tension subroles` rhyme does not transfer — `tension` is a non-runnable group, while `actors` is a runnable, positional-bearing leaf (049 grows it to an optional positional). Hosting a `subroles` subcommand under it would create the runnable-parent-with-children shape 025 ADR-1's foreclosure avoids and an ordering dependency on the pending 049/050; a distinct command (the 026/050 shape) removes both. Because a distinct command cannot be spelled as two words, the spelling is a single token: **`subrole-actors`** is pinned (names the bare-actor return shape and the `/subroles/actors` endpoint). The alternative **`subrole-fillers`** (naming the "Subrole Filler Roll-up" capability) is equally valid and adjustable at build time without changing behavior — `subrole-actors` is chosen to avoid implying the assignment-shaped projection of 047's `fillers`.
- **Actor shape, not assignment shape** (plan ADR-2): the endpoint returns bare `Actor` records (cross-linked assignments excluded), the same shape `GET /actors` returns — so rows carry `id`/`name`/`kind` and **not** `focus`/`elected_until`. The assignment-shaped read of a single role's fillers is Role Fillers (047, `fillers <role-id>`); this roll-up surfaces the actors, not the filling relationship.
- **Reuse-only at the data layer** (plan ADR-2): the command reuses the landed `internal/glassfrog.Actor` model + generics (the walk decodes `Page[json.RawMessage]` (structured) / `Page[Actor]` (human)), the landed plural `actors` render key (`ResourceActors`/`ActorsView` + `actors.full`/`actors.compact`), and the landed `validateKind` set — **no new model, render resource, or validator**. This is thinner than 048, which had to *add* the `actors` render path; 051 only adds a command leaf.
- **Only `--kind` is surfaced** (plan ADR-2): `listSubrolesActors` accepts only `kind` + pagination — no `role_id`, no `q` — so the directory's `--role-id`/`--query` filters are deliberately absent here (a flag with no endpoint setting would mislead). `--kind` reuses the landed `validateKind` (reject-unknown against `{human, agent}`, fail-fast, transport tripwire — the `validateClosedFlagSet` shape; the message names `--kind`).
- **Path-swap is the only data difference** (plan ADR-3): the runner reuses the landed `runActorsListWalk` shape (kind validation → `paging.All` walk → render → completeness note) with the request path set to `/roles/{role_id}/subroles/actors` and the role id supplied from the positional. Whether that is expressed by parameterizing `runActorsListWalk` with the path + base query or by a thin sibling runner is implementation-level; parameterizing keeps the kind/paging/render/error logic single-sourced.
- **Leaf-`404` vs empty-`200`** (plan ADR-3): the API answers `404` for a leaf anchor (no sub-roles); the command surfaces it through the shared chain as a non-zero read failure naming the status, with no special-case message. This stays distinct from the genuine empty-list `200` (sub-roles exist but carry no fillers), which renders the `no actors` line and exits `0`. Both outcomes are pinned by BDD scenarios + a transport tripwire so they cannot conflate.
- **Completeness, errors, config reuse the shared seams unchanged** (plan Cross-cutting): `paging.All` + `--first-page` (016/025 ADR-3), the shared `classifyClientError`/frozen `Outcome`/`ExitCode` registry (011/015), 032's `reportFailure` chokepoint, the 035-widened render flow, the persistent `--base-url`/`--output` flags — all reused exactly as `runActorsListWalk` uses them. Adds no new `Outcome`/`ExitCode`.
- **Command conventions** follow 001/003: the leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator (`ExactArgs(1)`) + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
