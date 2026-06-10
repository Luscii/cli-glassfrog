# Interface Accord: Role Policies — CLI

**Feature**: 034-role-policies
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (two sibling commands `policies <role-id>` + `policy <pol-id>`, list-only flags structurally enforced), ADR-2 (grow `Policy`, generalize `Document[T]`), ADR-3 (`--query` free-text pass-through, id pass-through), ADR-4 (`policy`/`policies` render keys). Completeness: Cross-cutting (reuses 025 ADR-3).

---

This accord pins the operator-facing policy read surface: the two commands `policies` (per-role list) and `policy` (standalone read), their flags, the rendered output, completeness signalling, and exit codes. The request seam they call through is pinned in `010/interface-spec.md`; the walker in `016`; format selection/rendering in `018`/`019`/`020`; the resolved base URL and token arrive pre-assembled in the `ConnectionContext` (009). The Go-facing surface (schema growth, `Document[T]`, cli symbols, render keys) is in `034/interface-spec.md`. Distinct from Role Reads' embedded view (`roles <id> --include=policies`, 025) — this is the *addressable* policy surface 025 deferred here.

---

## Surface

### `glassfrog policies <role-id>` — list a role's policies

A guard-registered (001), explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /roles/{id}/policies` (`listRolePolicies`) and produces the policies governing that role's interior as a list result. Walks to completion by default (see Interactions).

**Synopsis**:
```
glassfrog policies <ROLE_ID> [--query TEXT] [--first-page] [--per-page N] [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `ROLE_ID` | string | yes | The role whose policies to list (`role_…`). Exactly one required; zero or more than one is a usage error (cobra `ExactArgs(1)`), no API call. The id is **not** validated locally; the API resolves it (an unknown/malformed id → `404`). |

**List flags** (declared only on `policies`):

| Flag | Short | Type | Default | Description |
|---|---|---|---|---|
| `--query` | `-q` | string | — | Free-text search, sent verbatim as the endpoint's `q` parameter. Sent only when the flag is **present and non-empty** (`cmd.Flags().Changed`); `--query ""` behaves as no filter. Not validated locally — it is a search string, not a closed enum. |
| `--first-page` | — | bool | false | Opt out of the full walk: fetch only the first page and signal if more exist (see Interactions). |
| `--per-page` | — | int | *(016 default: API max)* | Page size for the walk (016's `WithPageSize`); the API owns the valid range. |

### `glassfrog policy <pol-id>` — read a single policy

A guard-registered, explicitly-wired runnable leaf with `Args: cobra.ExactArgs(1)`. Reads `GET /policies/{id}` (`getPolicy`) and produces the single policy, including its full body.

**Synopsis**:
```
glassfrog policy <POL_ID> [--base-url URL] [-o FORMAT]
```

| Argument | Type | Required | Description |
|---|---|---|---|
| `POL_ID` | string | yes | The policy to read (`pol_…`). Exactly one required (cobra `ExactArgs(1)`). Passed through unvalidated; an unknown id → `404`. |

`policy` declares **no list flags**. Passing `--query`/`-q`, `--first-page`, or `--per-page` to it is rejected by cobra's unknown-flag handling as a usage error before any request — this is how the spec's "the search filter applies only to the list" is enforced (no hand-rolled cross-combo guard; plan ADR-1).

**Inherited (persistent root) flags**, read by cobra inheritance on both commands, not redeclared:

| Flag | Owner | Description |
|---|---|---|
| `--base-url` | 011 | Override API base URL (top rung of 008's precedence chain). |
| `-o`, `--output` | 020 | `full` (default) \| `compact` \| `json` \| `yaml`. |

**Output** (success, stdout): the result is rendered by Output Format Selection (020) in the resolved format — `json`/`yaml` emit the structured document, `full`/`compact` render the human projection (019). The raw API envelope is never emitted under a human format. **Format changes rendering, not fetch depth**: the list walks to completion in every format (and `--first-page` limits all formats to one page); structured output never returns a shorter set than human.

*List (`policies`), `full`* — one block per policy, blocks separated by a blank line:
```
<Policy Title> (pol_0123…)
  Role:   <role_… | (org-level — no role)>
  Domain: <dom_… | (whole-role — no domain)>
  Body:
    <full policy body | (no body set)>
```
*List, `compact`* — one line per policy, following the repo's compact convention (resource id first, then double-space-separated `key=value` fragments): `pol_0123…  <Policy Title>  role=<role_…|—>  domain=<dom_…|—>`.

*Single (`policy`), `full`* — the policy block above with the full body rendered verbatim (never truncated or reflowed — CONSTITUTION VI), plus `created_at`/`updated_at` shown with explicit-absence guards.
*Single, `compact`* — `pol_0123…  <Policy Title>  role=<role_…|—>  domain=<dom_…|—>` (body omitted).

**Empty list** (the role has no policies, or none matches `--query`): under `full`/`compact`, stdout is exactly `No policies.` and the command exits `0` — an empty list is a valid answer, not an error. (Structured formats emit a valid empty list — `{"data": []}` for `json` — never `null`.)

## Interactions

**Dispatch**: each command has its own `RunE` (no `len(args)` branching — the two-command split replaces 025's single-command branch). Before any network call, in order: (1) cobra `Args`/flag parsing (a list-only flag on `policy` fails here as an unknown flag); (2) `--output` resolution (020). Any failure here is a fail-fast usage error and **no request is sent** (a transport tripwire asserts this, per 011/013/025).

**List completeness** (`policies`; reuses 025 ADR-3 verbatim):
- **Default** — the command walks every page via `paging.All[Policy]` (016) and renders the complete set.
- **`--first-page`** — issues a single page request, renders the first page, and if more pages exist writes one line to **stderr**, exiting `0`:
  ```
  note: more policies exist than shown; re-run without --first-page to fetch all
  ```
- **Mid-walk failure** (default walk stops on a transport/API/malformed-paging error) — renders the policies gathered so far, writes one explicit line to **stderr** naming the cause, and exits **non-zero** via `classifyClientError(Stop)`:
  ```
  note: result is incomplete — <cause>; the policies shown are a partial set
  ```
A partial list is therefore never silently presented as complete (CONSTITUTION VI; spec accord + validation scenario). The single `policy` read is unpaginated — no completeness signalling.

**Piping / scripting**: stdout carries only the rendered result (or `No policies.`); all diagnostics — failure messages and both incompleteness notes — go to **stderr**, never interleaved into the stdout projection.

**Configuration precedence**: `--base-url` (008 chain) and `--output` (020 chain: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → `full`) are resolved upstream; the token via 005. No new configuration here.

## Error Communication

Errors go to **stderr**; the process exit code is the category from Exit-Code Convention (004), produced through **011's shared `classifyClientError`**. Both commands **introduce no `Outcome` category and no `ExitCode` case**. Every message names the cause **and** a next step, and never includes the token.

| Condition | Source error (010) | Outcome (via `classifyClientError`) | Exit | stderr message (cause + next step) |
|---|---|---|---|---|
| Policies listed / policy read (incl. empty list) | — | `Success` | 0 | — (result on stdout; incompleteness note on stderr when applicable) |
| No usable token | `*AuthError{NoCredentials}` | `UsageError` | 2 | "not authenticated — run `glassfrog auth login`" |
| Unreadable / malformed credential file | `*AuthError{CredentialError}` | `RuntimeError` | 1 | names the path — "fix or remove the malformed `.glassfrogrc`" |
| Unknown/forbidden role id or policy id, or other non-2xx (401/403/429/4xx/5xx) | `*ResponseError` (→ `*ProblemError`, 015) | `APIError` 3 / `PermissionError` 4 / `RateLimited` 5 | 3/4/5 | names the HTTP status + extracted detail (015), per-class next step (permission: "check the token's access"; rate-limit: "retry later") |
| Could not reach the wire | `*TransportError` | `NetworkUnavailable` | 6 | names the transport failure + host — "check network access and the base URL" |
| 2xx body did not match the expected shape | `*DecodeError` | `APIError` 3 | 3 | "the API response did not match the expected shape — may be an API change; report it" |
| Malformed paging mid-walk (`policies`) | `*MalformedPageError` (016) | `RuntimeError` | 1 | "the API returned malformed pagination — partial set shown" |
| Base-URL configuration error | base-URL error from `NewClient` | `UsageError` | 2 | names the malformed base URL + source |
| Invalid `--output` selector | `*output.FormatError` (020) | `UsageError` | 2 | names the bad format value + the four valid names |
| List-only flag on `policy`, or wrong positional count (zero / >1) | — (cobra) | `UsageError` | 2 | cobra's unknown-flag / arg-count message; no request sent |

Codes `4`/`5` arrive via 015's landed split of `APIError`(3) at the shared classifier; these commands benefit with no edit. A `*DecodeError` (a 2xx body that will not parse) maps to `APIError`(3), not `RuntimeError`(1) — **Diagnostic Normalization (031 ADR-2)** reclassified it as part of consolidating the error→category mapping into the shared `Diagnose` normalizer (which `classifyClientError` now delegates to). Role Policies documents but does not own that mapping, so it inherits the change with no code edit. The token value never appears in any message.

## Consistency Notes

- **Addressable, not embedded**: `glassfrog policies`/`policy` is the addressable policy surface; `glassfrog roles <id> --include=policies` (025) embeds policies inline on a role as a convenience view. The two coexist deliberately (spec Non-Behavior) and share the grown `internal/glassfrog.Policy`, not a command.
- **Two-command split enforces list-only-ness structurally** (plan ADR-1): unlike 025's one-command `len(args)` branch with explicit filter+id guards, `--query`/`--first-page`/`--per-page` simply don't exist on `policy`, so cobra's unknown-flag handling does the rejecting. Both commands are `ExactArgs(1)` (mirrors 026's `subroles <id>`), not `MaximumNArgs(1)`.
- **No local input validation** (plan ADR-3): unlike 011's `validateInclude` / 013's `validateStatus` / 025's `validateRolesInclude`, this feature has no closed-enum input. `--query` is free text passed through as `q`; the id is a free identifier passed through to a clean `404`. (025 ADR-4 principle: validate where the API would silently mislead; pass through where it reports cleanly.)
- **Reuses landed machinery**: the persistent `--base-url` (011) and `--output`/`-o` (020), the shared `classifyClientError` + frozen `Outcome`/`ExitCode` registry (011/015), the `paging.All` walker + `RetryExecutor` (016/017), and the `renderResult[T]` dispatch over `internal/output`/`internal/render` (018/019/020). Adds only the two commands, the `Policy` growth + `Document[T]` generalization, and two **new** render keys — `policies` (list) and `policy` (single).
- **`-q` short alias**: new, no collision (`--include` is long-only, 020's `-o` is taken by output). **Flag/command spellings** (`policies`, `policy`, `--query`/`-q`) resolve the spec's confirmed surface; they are conventional, adjustable at build time without changing behavior.
- **Sets the per-role-read surface precedent** (plan ADR-1): #33 (Role Domains) and #38 (Role Projects) can follow `domains <role-id>`/`domain <dom-id>` and `projects <role-id>`/`project <prj-id>` verbatim. No `role` group is created here.
- **Command conventions** follow 001/003: each leaf registers through the fail-loud guard, is explicitly wired in `main`/`Assemble`, declares its `Args` validator + a non-empty `Short`, sets `SilenceErrors`/`SilenceUsage`, and changes no package-global cobra toggles. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
