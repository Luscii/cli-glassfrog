# Plan: Tension Capture

**Feature**: 042-tension-capture
**Role**: Shaper
**Inputs**: `specs/042-tension-capture/spec.md`, PROJECT.md, `.score/memory/DECISIONS.md` (precedent), `.score/memory/DEPRECATION.md`, `.score/memory/LEARNINGS.md`; existing code in `internal/cli` (`projects.go`, `authcmd.go`, `registry.go`), `internal/apiclient` (`client.go`, `execute.go`, `retry.go`), `internal/glassfrog` (`document.go`, `projects.go`)

---

## System Architecture

Tension Capture is the CLI's **first write**. Architecturally it is the single-resource read shape (`project <proj-id>` / `policy <pol-id>`) run with a `POST` that carries a body: resolve output format first, validate the closed-enum input fail-fast, assemble the connection, build the retrying executor, send exactly one request, then render the created resource or classify the failure. It rides the proven chain end-to-end and adds the smallest possible new surface to support a request body.

**Components**:

- **`internal/cli/tension.go`** (new) — a non-runnable `tension` group command parenting a `create` leaf (`glassfrog tension create <role-id>`), registered through `Register` (001), mirroring the `auth` / `auth login` group/leaf pattern. The leaf carries `--body`, `--label`, `--meeting-type`, inherits the persistent `--base-url`/`--output`, and delegates to a pure `runTensionCreate(cfg)` over the same seam shape as `projectsSeam` (`assemble` / `newClient` / `sleep` / `resolveFormat`) so every branch runs offline against a fake transport. `runTensionCreate` validates inputs (ADR-3), builds the JSON body, sends one `Execute` to `POST /roles/{role_id}/tensions`, and renders the created tension.
- **`internal/glassfrog/tension.go`** (new) — the `Tension` response model (matching the v5 `Tension` schema: `id`, `type`, `body`, `status`, `role_id`, `sensed_by_id`, `created_at`, `updated_at`, `label`, `meeting_type`, `parent_role_id`, explicit snake_case tags, forward-compatible decoding) plus the request input type encoding the nested `{tension: {body, label?, meeting_type?}}` envelope. Models live here per 011 ADR-1 (grow/add in `glassfrog`, never command-local). The token is never a field — it rides the transport (CONSTITUTION II).
- **`internal/apiclient` `Request`/`Execute`** (extended) — gains the ability to attach a request `Content-Type` so a JSON body is parsed by the API (ADR-1). All existing GET reads are unaffected.
- **`internal/render`** (extended) — a new render resource + view renders the created tension on the human path; the machine path reuses `output.RenderSuccess` over the raw `{data: …}` bytes (silent conformance to the 018/020 single-read pattern). Exact template content is interface-level.

**Data flow**: `create <role-id> --body …` → resolve format (020) → validate `--meeting-type` + non-empty `--body` (fail-fast, no request) → assemble (009) → `NewClient` (008/007) → `NewRetryExecutor` (017) → marshal `{tension:{…}}` → one `Execute` `POST /roles/{id}/tensions` with `Content-Type: application/json` (010, X-Auth-Token via 007's transport) → on 201 decode `Document[Tension]`; machine renders raw `{data}` (018), human renders the tension view (019) → success (0). On any failure, `reportFailure` + `classifyClientError`/`refineClientError` (015/017) map status to the existing outcomes; no new `Outcome`/`ExitCode`.

---

## Architecture Decisions

### ADR-1: Carry the write body's `Content-Type` on the request descriptor

**Context**: Every landed command is a bodyless `GET`; `apiclient.Request` has `Method`, `Path`, `Query`, and `Body io.Reader` but **no header field**, and `Execute` sets no `Content-Type`. The Glassfrog v5 `createTension` body is JSON; without `Content-Type: application/json` the API may ignore the body and answer `422` (a silent-wrong-result class the codebase explicitly designs against — see §113/§200). The first write must thread a request content type to the wire.

**Options considered**:
1. **Narrow `ContentType string` field on `Request`** — `Execute` sets the header only when non-empty. Minimal; reads pass `""` and are byte-identical; the one knob the write needs, no more.
2. **General `Header http.Header` map on `Request`** — `Execute` merges it onto the outbound request. More flexible (would also serve a future `If-Match`), but adds a general mechanism with one real consumer today; `If-Match` is the explicitly-deferred Clobbered Changes capability, and the spec's non-behaviors forbid `If-Match` here.
3. **Command-side `RoundTripper`** — wrap the client to inject the header. Heavier, splits request shape across two layers, and fights the "the `Request` descriptor is the per-call shape" model in `client.go`.

**Decision**: Option 1 — a narrow `ContentType string` on `Request`. `Execute` does `if req.ContentType != "" { httpReq.Header.Set("Content-Type", req.ContentType) }` after building the request and before `Do`. The tension write sets it to `application/json`; every read leaves it empty and is unchanged.

This conforms to the codebase's anti-speculation idiom (008/010 deferred defaults; 017 caps `[ASSUMED]` until tuned) — add the knob a real consumer needs, not a general header bag for hypothetical writes. When a second request header earns its place (e.g. `If-Match` for Clobbered Changes), generalizing `ContentType` into `Header http.Header` is a contained refactor with that consumer to ground it.

**Consequences**: The first write-body mechanism is established and minimal. The reads' transport behavior is provably unchanged (empty content type → no header set). A future per-request header need will revisit this field rather than extend it speculatively now. Sets a precedent worth recording for the rest of the write path.

### ADR-2: `tension create <role-id>` as a group + leaf, reserving the `tension` namespace

**Context**: The spec fixes the surface as `glassfrog tension create <role-id>`, deliberately leaving room for future `tension get/list/update` siblings. The registration guard (001) requires a command be *either* a leaf (with an action) *or* a group (with ≥1 child), never both. The codebase has two surface idioms: runnable-parent-with-children (`me`, `me roles`) and non-runnable group (`auth`, `auth login`).

**Options considered**:
1. **Non-runnable `tension` group + `create` leaf** — `tension` parents `create`; `tension` alone prints help (no action). Matches `auth`/`auth login` exactly and the registration guard's group rule; the `tension` namespace is reserved for reads/edits without re-litigating the surface.
2. **Flat `tension-create <role-id>` leaf** — a single hyphenated command, no group. Fewer moving parts now, but forecloses a clean `tension <verb>` family and reads awkwardly beside the noun-led reads.

**Decision**: Option 1 — a non-runnable `tension` group parenting a `create` leaf, both assembled and attached through `Register` (the group is built with its child before being registered under root, so the guard's ">=1 child" rule holds at attach time). This realizes the spec's stated intent to leave room for siblings and sets the namespace precedent the future tension reads/edits will conform to.

**Consequences**: `tension` with no subcommand prints help (group behavior). The list-only/write-only flags live on `create`, so a future `tension get` adds its own leaf with its own flags — passing `--body` to a read would be a cobra unknown-flag usage error for free (the structural guard 034/038 rely on). No `role` group is created (consistent with 026/033/034/038). Cross-spec relevant → record in DECISIONS.md.

### ADR-3: Validate `--meeting-type` and a non-empty `--body` fail-fast; pass the role id through

**Context**: `create` takes one closed-enum input (`--meeting-type` ∈ {`tactical`, `governance`}), one required free-text input (`--body`), one optional free-text input (`--label`), and the path role id. The codebase has a settled rule (§113/§200): validate locally where the API would otherwise silently mislead; pass through where it reports cleanly.

**Options considered**:
1. **Local fail-fast for the closed enum + required body; pass the id through** — a pure `validateMeetingType` (mirroring `validateStatus`) rejects an unsupported `--meeting-type` as `UsageError(2)` naming the value and the supported set; a missing or trim-empty `--body` is rejected as `UsageError(2)` naming `--body`; both run before assembly with a transport tripwire asserting nothing was sent. The role id is escaped as one path segment and passed through to the API's clean `404`/`422`.
2. **API pass-through for everything** — send whatever is given and surface the `422`. Rejected for `--meeting-type`: an opaque `APIError(3)` is indistinguishable from a real failure, and the enum is knowable locally (§113's exact rationale).

**Decision**: Option 1. `validateMeetingType` conforms to the `validateStatus`/`validateInclude` precedent (silent conformance — a new validator over the meeting-type set sourced from the spec enum, not a new pattern). The required-non-empty `--body` check is the spec's deliberate fail-fast: a bodyless tension is meaningless, so reject it before spending a request rather than relying on the API's `422` (consistent with resolving inputs that are knowable client-side). `--body` is treated as empty when it trims to nothing (spec assumption). The role id is a free identifier the endpoint reports cleanly as `404`, so it passes through (§200) — no local id-shape regex.

**Consequences**: Both pure checks run pre-assembly, so the no-request-on-rejection invariant holds (tripwire transport in tests). Error precedence follows the sibling reads: resolve `--output` first, then validate inputs. Adds no new `Outcome`. The meeting-type set tracks the spec enum (the §113 "set tracks the spec" property).

---

## Cross-cutting Concerns

**Non-idempotent retry (silent conformance to §133)**: `create` uses the same `NewRetryExecutor` as the reads, but 017's `isSafeMethod` already restricts auto-retry-on-429 to `GET`/`HEAD` — a `POST` surfaces the `429` on first occurrence and is **never silently re-sent**, so capture cannot double-create on a rate-limit retry. The executor is still built uniformly (it also carries the `sleep`/progress seam); the method gate makes the write safe with no command-side special-casing.

**Failure mapping (no new outcomes)**: every failure routes through the landed `reportFailure` → `refineClientError`/`classifyClientError` chain. Not-authenticated → the shared fail-safe (`UsageError(2)` NoCredentials / `RuntimeError(1)` CredentialError, per the landed 011/012 mapping); transport → network-unavailable; non-2xx via 015's `ExtractProblem` — `404` (unknown sensing role) and `422` (rejected body/field) classify as `APIError(3)` with the RFC 9457 detail surfaced, `401`/`403` → `PermissionError(4)`, `429` → `RateLimited(5)`. The command adds no interpretation and never prints the token.

**Output**: the 201 `{data: Tension}` decodes into the existing generic `Document[Tension]` (no per-resource envelope — §143/§254). Machine formats emit the raw `{data}` via `output.RenderSuccess`; the human path renders a tension view that surfaces the `ten_` id (the load-bearing handle a later proposal references). Buffer-then-write so a render failure leaves stdout empty and maps to `RuntimeError(1)`.

**Testing**: the seam injection keeps `runTensionCreate` pure over a fake transport — unit tests drive happy/usage/failure branches offline; a transport tripwire pins "no request issued" for both rejection paths; a godog BDD suite mirrors the driving scenarios (matching the per-command `*_bdd_test.go` convention). The new `apiclient.Request.ContentType` gets a focused test asserting the header is set for a non-empty value and absent for the empty default (so the reads' unchanged behavior is pinned).

---

## Implementation Strategy

**Phase 1 — Write-body transport seam**: extend `apiclient.Request` with `ContentType` and have `Execute` set the header when non-empty; test header-present/absent. Small, self-contained, and a prerequisite for the command. (PR-sized.)

**Phase 2 — The `tension create` command**: add the `glassfrog.Tension` model + request input type; add `validateMeetingType` + the non-empty `--body` check; build `tension.go` (group + leaf, seam, `runTensionCreate`, body marshalling, one `Execute`, render/classify); add the render resource + view; wire the group under root through `Register`; BDD + unit tests. Depends on Phase 1. (PR-sized; the tasks skill may split the model/render from the command if it prefers.)

---

## Risks

- **Body silently ignored without `Content-Type`** — likelihood low once ADR-1 lands, impact high (a `422`/empty-body surprise). Mitigation: ADR-1 always sends `application/json` for the write, and a Phase-1 test pins the header.
- **Double-create on retry** — likelihood very low (already mitigated), impact high (duplicate governance seeds). Mitigation: §133's `isSafeMethod` gate; a test asserting a `POST` 429 is surfaced, not retried, guards against regression.
- **`meeting_type` as routing hint vs. meeting binding** — the spec assumes `--meeting-type` is a stored categorization, not a meeting attachment (the endpoint says captures are "not bound to a meeting"). Low impact; if the API treats it otherwise, the assumption is documented and the field can be dropped without touching the chain.

---

## What This Plan Does Not Cover

- **Protocol-level contract** — exact flag spellings, help text, the request/response field mapping, and the `Content-Type` constant location are the **interface** skill's concern.
- **Executable scenarios** — Gherkin for the driving scenarios is the **scenarios** skill's concern.
- **Task decomposition** — PR-sized units within the two phases are the **tasks** skill's concern.
- **Tension reads/edits/delete, list-subroles, and the proposal write-flow** — deferred to their own specs (spec non-behaviors); this plan reserves the `tension` namespace (ADR-2) but designs only `create`.
- **Optimistic concurrency (`If-Match`)** — out of scope (Clobbered Changes); ADR-1 deliberately does not build the general header mechanism it would need.
