# Interface Accord: Actor Read — CLI

**Feature**: 049-actor-read
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (grow 048's `actors` to `MaximumNArgs(1)`; 0 args → directory list, 1 arg → single read), ADR-2 (`ActorDetail` embeds `Actor` + optional `Roles`/`Assignments`; `{data: ActorDetail}` document), ADR-3 (`--include` validated locally over `{roles, assignments}`; actor id passed through to a clean 404), ADR-4 (new singular `ResourceActor` render key, full+compact). Cross-cutting: no pagination (single resource).

---

This accord pins the operator-facing single-actor surface: the `actors <id>` form of the `actors` command, its `--include` flag, the rendered output and footprint, and exit codes. It is the **single-actor drill-in** of the Actor Reads slice — the by-id read the directory (048) defers to. The directory list (`glassfrog actors`, 0 args) is pinned in `048/interface-cli.md`; this accord pins the **1-arg single read** on the same command. The request seam it calls through is pinned in `010/interface-spec.md`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). Unlike the directory, it walks no pages — `GET /actors/{id}` is a single resource and the `?include` embeds arrive inline.

---

## Surface

### `glassfrog actors <id>` — read one actor with their governance footprint

The `actors` command (guard-registered 001, explicitly-wired) is grown from `cobra.NoArgs` (048) to `cobra.MaximumNArgs(1)`; its `RunE` branches on `len(args)`. With **one positional id** it reads `GET /actors/{id}` (`getActor`) and produces that actor as a single-actor result. With **no positional** it is the directory list (048, unchanged). The positional id accepts both `per_` (human) and `agt_` (agent) prefixes — the same command reads either kind.

**Synopsis**:
```
glassfrog actors <id> [--include LIST] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `<id>` | string | yes (for the single read) | The actor's id — `per_…` (human) or `agt_…` (agent). Passed through on the path verbatim; **not** validated locally — `getActor` answers a malformed/unknown id with a clean `404` (no silent-wrong-results), so the API resolves it (025 ADR-4). Exactly one positional; a second positional is a usage error before any request. |

**Single-read flag** (optional):

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--include` | — | string (comma-separated) | — | Embed related resources inline on the actor, sent as the `include` query parameter. **Validated locally** before any request against the closed set (`roles`, `assignments`); an unsupported value is a usage error naming the value and the supported set, no API call. `--include roles` embeds the actor's governance footprint (each role's name/purpose/accountabilities/domains); `--include assignments` embeds the actor↔role assignments. Multiple values combine (e.g. `--include roles,assignments`). Sent only when present and non-empty. |

**Mode separation** — the directory's filter flags (`--kind`, `--role-id`, `--query`/`-q`, `--first-page`, `--per-page`) are **list-only**; `--include` is **single-only**. Supplying a list filter together with an id, or `--include` with no id, is a usage error before any request (the 025 ADR-1 cross-combo discipline):

| Invocation | Result |
|---|---|
| `actors` (0 args, optional filters) | Directory list (048) |
| `actors <id>` (+ optional `--include`) | Single read (this accord) |
| `actors <id> --kind …` / `--role-id …` / `--query …` / `--first-page` | Usage error (2), no request — filters are list-only |
| `actors --include …` (no id) | Usage error (2), no request — `--include` is single-only |
| `actors <id> <id2>` | Usage error (2), no request — at most one positional |

**Inherited (persistent root) flags**, read by cobra inheritance, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured `{data: …}` document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format.

*Human (`full`/`compact`)* — the **new singular `actor` render key** (plan ADR-4), rendering the `ActorDetail` projection. Distinct from `ResourceActors` (the 048 directory list) and `ResourceMe` (the `me` identity document):
- `full` — the actor's identity, then the footprint when embedded:
  ```
  <per_… | agt_…>  [<kind>]
    Name:  <name>
    Roles:                       (present only with --include roles)
      <role_…>  <role name>
        Purpose:  <purpose | (no purpose set)>
        Accountabilities:  <n> | (none)
        Domains:  <n> | (none)
    Assignments:                 (present only with --include assignments)
      <role_…>  <role name>  <focus | —>
  ```
- `compact` — the identity line, with embed counts when present: `<per_… | agt_…>  [<kind>]  <name>  roles=<n>  assignments=<n>`

The embed sections are rendered **only when `?include`d** — each guarded by an explicit-absence marker (019), never inventing a value. An embedded role with no purpose/accountabilities/domains renders its absence markers (`(no purpose set)` / `(none)`), not blanks. For `json`/`yaml`, the single read emits the actor's raw 2xx bytes as the `{data: …}` document (the single-resource raw-bytes path, 018 ADR-2 — not a decode-and-re-encode of `ActorDetail`).

## Interactions

**Dispatch**: the command's single `RunE` branches on `len(args)`. For the single read (1 arg), before any network call, in order: (1) cobra `Args`/flag parsing (a second positional fails as a `MaximumNArgs(1)` violation; an unknown flag fails here); (2) `--output` resolution (020); (3) mode-separation guards (a list filter with an id, `--include` without an id) → usage error; (4) `--include` validation (the reject-unknown validator over `{roles, assignments}`). `--output` is resolved first so an invalid format is reported even when a mode-separation or `--include` error is also present (the output-first precedence T003 mandates); all are pure, pre-assembly checks, so the order only sets error precedence and never sends a request. Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/025/048).

**Single request, no walk**: the read issues exactly one `Execute` of `GET /actors/{id}?include=…` into a `{data: ActorDetail}` document. There is **no pagination** — `getActor` is a single resource and the `roles`/`assignments` embeds arrive inline in that one response, even when they are large arrays (a held-out validation scenario asserts exactly one request). The `--first-page`/`--per-page` list flags do not apply.

**Piping / scripting**: on success, stdout carries only the rendered result. On failure, Output-Aware Failure Rendering (032, landed) routes by format: structured (`json`/`yaml`) failures emit the 018 unified error envelope on **stdout** (so an agent parses success and failure the same way), while human (`full`/`compact`) failures write the diagnostic to **stderr**.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

The process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`** (delegating to 031's `Diagnose`). Failure rendering is **format-aware** via the landed Output-Aware Failure Rendering chokepoint (032, `reportFailure`): for `json`/`yaml` the 018 unified error envelope is written to **stdout**; for `full`/`compact` the diagnostic (cause + next step) is written to **stderr**. 049 reuses `reportFailure` unchanged. The command **introduces no `Outcome` category and no `ExitCode` case**. Every diagnostic names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | Diagnostic — cause + next step (`full`/`compact` → stderr; `json`/`yaml` → 018 envelope on stdout) |
|---|---|---|---|---|
| Actor read (incl. with embeds) | — | `Success` | 0 | — (result on stdout) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | cause "not authenticated"; next step "run `glassfrog auth login` or set GLASSFROG_TOKEN" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | cause names the credentials file; next step "fix or re-create the credentials file with `glassfrog auth login`" |
| Unknown / malformed id (`404`), or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | cause names the transport failure; next step "check connectivity; the API may be unreachable" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | cause "the API response did not match the expected shape"; next step "this may be an API change; report it (`<decode error>`)" |
| Unsupported `--include` value | — (the reject-unknown validator) | `UsageError` | 2 | "unsupported --include value \"…\" — supported: assignments, roles"; no request sent |
| List filter with an id, or `--include` with no id | — (the mode-separation guard) | `UsageError` | 2 | names the misplaced flag (filters are list-only / `--include` is single-only); no request sent |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| More than one positional, or unknown flag | — (cobra) | `UsageError` | 2 | cobra's `MaximumNArgs(1)` / unknown-flag message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; this command benefits with no edit. The supported `--include` set is named **alphabetically sorted** in the rejection message (`assignments, roles`), matching the landed validator convention. The token value never appears in any message.

## Consistency Notes

- **One command, two modes** (plan ADR-1): `actors` grows from 048's `cobra.NoArgs` to `cobra.MaximumNArgs(1)`, the actor analogue of `roles`/`roles <id>` (025 ADR-1) — 0 args → directory list (048, preserved verbatim), 1 arg → single read (this accord). This is an announced divergence from 048 ADR-1's `cobra.NoArgs`; 048 is merged on `main` (#90), so 049 grows the already-landed command (no first-to-land race). **Foreclosure**: cobra cannot tell an actor id from a subcommand under an optional positional, so `actors <id> assignments` is impossible — Actor Assignments (050) takes a flag/separate-command surface, distinct from this accord's `--include assignments` embed.
- **Validate `--include` locally; pass the id through** (plan ADR-3): `--include` is the closed-enum input, validated against `{roles, assignments}` (reject-unknown, fail-fast, transport tripwire) — the landed `include.go` shape, as a **per-read set** distinct from the role include set (026 two-validator precedent); a shared set would accept `roles`/`subroles`/`policies` the actor endpoint silently drops. Whether it lands as a thin `validateActorInclude` or a parameterization of the landed validator (which hard-codes a set/message) is a build-time factoring detail — the behavior is fixed. The actor id is passed through (`getActor` documents only 401/404 — clean not-found, no silent-wrong-results: 025 ADR-4 "validate where the API would silently mislead; pass through where it reports cleanly").
- **`ActorDetail`, no schema growth of leaf types** (plan ADR-2): the single read decodes `{data: ActorDetail}` where `ActorDetail` embeds the unchanged `glassfrog.Actor` (011/048) + optional `Roles []Role`/`Assignments []Assignment` — the `RoleDetail` shape (025 ADR-2). The embedded `Role` (full shape) and `Assignment` are the landed 025 types, not new leaf models; the directory list keeps decoding `Page[Actor]`. The embed fields live on the detail type, not on `Actor` (which `me`/the directory still decode bare).
- **Singular render key** (plan ADR-4): the only new render artifact is the **`actor`** key — an `ActorDetailView` (or equivalent), the `ResourceActor` registry entry added to `builtinResources` (so the exhaustiveness guard covers it), and the `actor.full`/`actor.compact` templates. The established singular-detail shape (`ResourceRole`/`ResourceDomain`/`ResourcePolicy`) and the natural sibling of 048's plural `ResourceActors`. The footprint render may reuse the `role` template's accountability/domain fragments or define an actor-framed layout — a build-time detail; explicit-absence guards (019) keep absent embeds from rendering invented values. Structured `json`/`yaml` output needs no change (018 ADR-2 serializes raw bytes).
- **No pagination** (cross-cutting): unlike the directory (048) and every list read (025/026/038/041), the single read issues one `Execute` and follows no cursor — the `getMe`/`getRole` single-resource shape (011/025). `Page[T]`/`paging.All` are not instantiated; `--first-page`/`--per-page` do not apply.
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the single-resource render dispatch (`renderResult[T]`-style: the human projection for `full`/`compact`, the raw `{data:…}` document for `json`/`yaml` — not the walked-list `aggregateRawData` path), and the landed failure chain (031's `Diagnose` rendered format-aware by 032's `reportFailure`). Adds only the single-read branch on `actors`, the `--include` validator, the `ActorDetail` decode, and the `actor` render path.
- **Flag/command spellings** (`actors <id>`, `--include`) resolve the spec's confirmed surface (the spec flagged the `--include` shape `[ASSUMED]`); comma-separated values mirror `roles <id> --include policies,subroles` (025). No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
