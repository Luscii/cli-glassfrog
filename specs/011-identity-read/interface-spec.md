# Interface Accord: Identity Read — Specification

**Feature**: 011-identity-read
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (new `internal/glassfrog` schema package), ADR-3 (the `Outcome`/`ExitCode` extension + shared `classifyClientError`), ADR-4 (error→category mapping), ADR-5 (injected seam + pure `runMe`/`formatMe`/`validateInclude`).

---

This accord pins the **Go API surface** Identity Read introduces: the new `internal/glassfrog` schema package the read surface decodes into, the `internal/cli` extension that gives 004's reserved codes a producer (the `Outcome` enum + `ExitCode` cases + the shared `classifyClientError`), and the `me` command's internal shape (the injected seam, the pure `runMe`/`formatMe`/`validateInclude`). The CLI-facing surface — the command, flags, projection, and exit codes — is in `011/interface-cli.md`; this file is the package-level contract the Builder implements and the Verifier tests against. Field names and concrete Go types are a build detail; the **shapes, signatures, and which fields are projected** are the contract.

---

## Surface

### `internal/glassfrog` (NEW package — API schema, leaf)

Plain JSON-tagged structs decoded from API responses. No transport, no cobra, no exit codes — a leaf package both `internal/cli` and `internal/apiclient` may import without a cycle (ADR-1). Decoding is **tolerant of unknown/extra fields** (forward-compatible with API additions).

| Type | Shape (fields → projected?) | Notes |
|---|---|---|
| `MeResponse` | `Actor` (✓), `Organization` (✓), `Membership` (✓), `Roles []Role` (✓ when present) | The `GET /me` body. `Roles` is populated only when `?include=roles` was requested; empty/absent otherwise. |
| `Actor` | `ID` ✓, `Name` ✓, `Kind` ✓ (`human`\|`agent`); `CreatedAt`/`UpdatedAt` decoded, not projected | `ID` carries the `per_`/`agt_` prefix — the machine-actionable handle. |
| `Organization` | `ID` ✓, `Name` ✓ | `ID` is `org_…`. |
| `Membership` | `AccessLevel` ✓ (`admin`\|`normal`); `ID`/`ActorID`/`OrganizationID` decoded, not projected | — |
| `Role` | `ID` ✓, `Name` ✓ | **Minimal** — the `--include roles` embed. My Roles (012) grows THIS type to the full role shape (accountabilities, domains, assignments); never a second type. |

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `Outcome` (extended) | adds `NetworkUnavailable`, `APIError` to the existing enum (`dispatch.go`) | The operational categories 004 reserved; their producer is now this command. `String()` gains the two names. |
| `ExitCode` (extended) | `(Outcome) int` — adds cases `NetworkUnavailable→6`, `APIError→3` | Stays a **pure mapper** (never inspects an error); `default→codeInternalError(1)` fail-safe unchanged. |
| `classifyClientError` | `(err error) Outcome` | The **single** `errors.As` chain mapping the API client's typed errors (010) → `Outcome`. Reused verbatim by 012–017. Maps per the table in Error Communication. |
| `newMeCommand` | `(seam meSeam) *cobra.Command` | The guard-registered `me` leaf (`Use:"me"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); local `--include` flag; reads the persistent `--base-url` value; delegates to `runMe`. Wired once in `Assemble()`. |
| `runMe` | `(cfg meConfig) (Outcome, error)` | Pure over injected values (the `runLogin` shape): `validateInclude` → assemble context → build client → `Execute` → `formatMe` on success / `classifyClientError` on a typed error. Writes the projection to `cfg.stdout`, messages to `cfg.stderr`; returns the code-free `Outcome` the command maps onto dispatch's error channel. |
| `formatMe` | `(me glassfrog.MeResponse, includeRoles bool) string` | Pure projection renderer (`011/interface-cli.md` defines the surfaced fields/order). Unit-tested in isolation. |
| `validateInclude` | `(targets []string) error` | Rejects an unsupported `--include` target against the spec's `include` set (today `{roles}`), returning a usage error naming the unsupported target — **before** any context assembly or request. |
| `meSeam` | interface: assemble the `ConnectionContext` from a base-URL flag value, and build a `*apiclient.Client` over a base `http.RoundTripper` | Production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS` (real base transport); tests bind a fake transport returning canned `GET /me` responses (ADR-5). Exact method set is a build detail. |

### Consumed unchanged (not defined here)

- `apiclient.AssembleFromOS(flagValue) ConnectionContext`, `apiclient.NewClientFromOS(ctx) (*Client, error)`, `(*Client).Execute(reqCtx, Request, out any) (*Response, error)`, the `apiclient.Request` descriptor, and the typed errors `*AuthError` / `*TransportError` / `*ResponseError` / `*DecodeError` — all from `010/interface-spec.md`. `me` adds nothing to `apiclient`.

**Example (shapes, not literal values)**:
```
// production seam: assemble once, build once, send once.
ctx    := seam.assemble(baseURLFlag)                       // apiclient.AssembleFromOS
client, err := seam.newClient(ctx)                          // apiclient.NewClientFromOS → err is ctx.BaseURLErr when endpoint unusable
var me glassfrog.MeResponse
resp, err := client.Execute(reqCtx,
               apiclient.Request{Method:"GET", Path:"/me", Query:{"include":"roles"}},  // Query omitted when --include absent
               &me)                                          // 2xx → me populated; non-2xx → *ResponseError; etc.
out := formatMe(me, includeRoles)                            // projection string
```

---

## Interactions

- **Build-once / send-once flow**: `runMe` calls `seam.assemble` once and `seam.newClient` once, then `Execute` once. Resolution already happened at assembly (009), so identity + endpoint are stable; `me` re-resolves nothing and never reads `ctx.Cred.Token`.
- **Decode target**: `Execute` receives `&me` (a `*glassfrog.MeResponse`); on 2xx the body decodes into it, on non-2xx it is left untouched and a `*ResponseError` is returned (010's decode-or-skip contract).
- **`--include` first**: `validateInclude` runs before `seam.assemble`, so an unsupported target costs no network call (a tripwire fake asserts the transport is never invoked on rejection).
- **Classification reuse**: `classifyClientError` is the one place the read surface turns 010's typed errors into an `Outcome`; 012–017 call it rather than inlining their own `errors.As` chains.
- **Seam injection**: the `me` command is constructed with a seam (`newMeCommand(productionSeam{})` in `Assemble()`); tests construct it with a fake seam binding a fake `http.RoundTripper`, so every branch runs offline and off the real `~/.glassfrogrc`.

---

## Error Communication

`runMe` returns exactly one code-free `Outcome` per invocation; `classifyClientError` maps the API client's typed error to it (the command then maps `Outcome`→error-channel for dispatch, and `ExitCode` maps `Outcome`→process code at the single registry):

| Input error (from 010) | `classifyClientError` → `Outcome` | `ExitCode` |
|---|---|---|
| `nil` (2xx success) | `Success` | `0` |
| `*AuthError{NoCredentials}` | `UsageError` | `2` |
| `*AuthError{CredentialError}` | `RuntimeError` | `1` |
| base-URL error (from `newClient`) | `UsageError` | `2` |
| `*DecodeError` | `RuntimeError` | `1` |
| `*ResponseError` (generic non-2xx) | `APIError` | `3` |
| `*TransportError` | `NetworkUnavailable` | `6` |
| (validation, before any request) unsupported `--include` | `UsageError` | `2` |

- **Discrimination order**: `*AuthError` is matched (via `errors.As`) before `*TransportError`, since 007's fail-safe surfaces as the error from `Do` and must not be mislabelled transport (010's discipline, preserved here).
- **Exhaustiveness guard**: `classifyClientError`'s table test asserts each typed error maps to its expected `Outcome` with a `len`+comma-ok-style completeness check, so a dropped or added mapping fails loud, not silently (PR #10 LEARNINGS).
- **No secret anywhere**: none of the messages or the projection renders the token; `me` never reads `ctx.Cred.Token`. Pinned across success and every error branch.
- **Fail-safe**: `ExitCode`'s `default→1` still backstops any future/unmapped `Outcome`.

---

## Consistency Notes

- **Request Execution (`010/interface-spec.md`)**: this accord defines the *first consumer* of 010's seam — the decode target (`glassfrog.MeResponse`) and the classification (`classifyClientError`) 010 deliberately deferred to "the first consuming command." `me` consumes `Client`/`Execute`/`Request`/errors unchanged.
- **Exit-Code Convention (`004/interface-cli.md` + `exitcode.go`)**: the `Outcome`/`ExitCode` extension happens at 004's single registry; codes `3`/`6` get their producer, codes `0`/`1`/`2` are reused, none is renumbered. 015/017 later add cases that split `APIError`(3) into permission(4)/rate-limited(5) without renumbering.
- **Credential Storage (`006`)**: the `meSeam` + pure `runMe` mirror 006's `loginSeam` + `runLogin`; the `NoCredentials`→Usage / `CredentialError`→Runtime mapping mirrors 006's outcome shape.
- **New `internal/glassfrog` package**: the first API-schema package (ADR-1). It depends on nothing internal (leaf), so `cli` and `apiclient` import it without a cycle. 012–017 add resource models here and reuse `Role`.
- **No new configuration of its own**: `me` introduces no new env var or `.glassfrogrc` key; `--base-url`/`GLASSFROG_BASE_URL`/`base_url` are owned upstream (008), `GLASSFROG_TOKEN`/`token` by 005. `--include` is a request-shaping flag, not configuration.
- **Fifth specification touchpoint in this project** (after 005/007/008/009/010): a package-API accord. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
