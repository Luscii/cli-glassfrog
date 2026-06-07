# Interface Accord: My Actions — Specification

**Feature**: 013-my-actions
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (`Action` joins `internal/glassfrog`, reuses 012's `Pagination`/envelope), ADR-2 (shared `validateStatus`), ADR-3 (first page + signal), ADR-4 (injected seam + pure `runMyActions`/`formatMyActions`).

---

This accord pins the **Go API surface** My Actions introduces: the `Action` resource model added to `internal/glassfrog`, the shared `validateStatus` + status set, and the `me actions` command's internal shape (the injected seam, the pure `runMyActions`/`formatMyActions`). It **reuses** — and does not redefine — 011's `internal/glassfrog` foundation, `classifyClientError`, `Outcome`/`ExitCode`, and persistent `--base-url`, and 012's `me` parent command, `Pagination` type, list envelope, and signal renderer. The CLI-facing surface (command, flags, projection, exit codes) is in `013/interface-cli.md`; this file is the package-level contract the Builder implements and the Verifier tests against. Field names and concrete Go types are a build detail; the **shapes, signatures, and which fields are projected** are the contract.

---

## Surface

### `internal/glassfrog` (EXTENDED — add `Action`; reuse `Pagination` + envelope)

Plain JSON-tagged structs decoded from API responses. No transport, no cobra, no exit codes (011 ADR-1). Decoding is **tolerant of unknown/extra fields** (forward-compatible with API additions).

| Type | Shape (fields → projected?) | Notes |
|---|---|---|
| `Action` (NEW) | `ID` ✓ (`actn_…`), `Status` ✓, `Description` ✓ (nullable → `—`), `RoleID` ✓ (`role_…`), `Tags []string` ✓ (when present); `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, `Permissions`, `TriggerEvent`, `Note` decoded, not projected | The `/me/actions` list item. `ID` is the machine-actionable handle. |
| `Pagination` (REUSED from 012) | `PerPage`, `HasNextPage` ✓ (drives the signal), `NextCursor` | Not redefined here. If 013 lands before 012, it creates this type in `glassfrog` and 012 reuses it (first-to-land-creates). |
| list envelope (REUSED from 012) | `Data []Action`, `Meta{ Pagination }` | The `{data, meta.pagination}` body. Whether it is a generic `List[T]` or a per-resource struct is 012's decision; `Action` decodes through it. |

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `validateStatus` (NEW, shared with 014) | `(status string) error` | Rejects a non-empty `--status` value outside the spec's status set (`{archived, cancelled, completed, current, scheduled, someday, waiting}`), returning a usage error naming the value and the supported set — **before** any context assembly or request. An empty/absent value passes (no constraint). The set is sourced from the spec enum. |
| `newMyActionsCommand` | `(seam meSeam) *cobra.Command` | The guard-registered `actions` leaf (`Use:"actions"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) attached to the `me` parent (012); local `--status` flag; reads the persistent `--base-url` value; delegates to `runMyActions`. Wired once under `me` in `Assemble()`. |
| `runMyActions` | `(cfg myActionsConfig) (Outcome, error)` | Pure over injected values (the `runMe` shape): `validateStatus` → assemble context → build client → `Execute` → `formatMyActions` on success / `classifyClientError` on a typed error. Writes the projection to `cfg.stdout`, messages to `cfg.stderr`; returns the code-free `Outcome` the command maps onto dispatch's error channel. |
| `formatMyActions` | `(list <glassfrog list-of-Action>) string` | Pure projection renderer (`013/interface-cli.md` defines the fields/order, the empty-result line, and the "more available" signal driven by `Meta.Pagination.HasNextPage`). Unit-tested in isolation. |
| seam (REUSED — 011's `meSeam`) | interface: assemble the `ConnectionContext` from a base-URL flag value, and build a `*apiclient.Client` over a base `http.RoundTripper` | `me actions` **reuses 011's `meSeam`** rather than defining its own `myActionsSeam` — its needs are identical to `me`/`me roles` (assemble + newClient). Production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS`; tests bind a fake transport returning canned `GET /me/actions` responses (ADR-4). |

### Consumed unchanged (not defined here)

- **From `010/interface-spec.md`**: `apiclient.AssembleFromOS(flagValue)`, `apiclient.NewClientFromOS(ctx)`, `(*Client).Execute(reqCtx, Request, out)`, the `apiclient.Request` descriptor, and the typed errors `*AuthError` / `*TransportError` / `*ResponseError` / `*DecodeError`. `me actions` adds nothing to `apiclient`.
- **From `011/interface-spec.md`**: `classifyClientError(err) Outcome`, the `Outcome` enum (incl. `NetworkUnavailable`/`APIError`), the `ExitCode` mapper (codes `3`/`6`), and the persistent `--base-url` root flag (`apiclient.FlagBaseURL`). `me actions` reuses these; it adds no `Outcome` case and no `ExitCode` case.
- **From `012`**: the `me` parent command, `glassfrog.Pagination`, the list envelope, and the "more results available" signal renderer/convention.

**Example (shapes, not literal values)**:
```
// production seam: validate, assemble once, build once, send once.
if err := validateStatus(statusFlag); err != nil { return UsageError, err }   // before any I/O
ctx    := seam.assemble(baseURLFlag)                                          // apiclient.AssembleFromOS
client, err := seam.newClient(ctx)                                            // apiclient.NewClientFromOS → err is ctx.BaseURLErr when endpoint unusable
var list glassfrog.MyActionsResponse                                          // {Data []Action, Meta{Pagination}}  (envelope shape reused from 012)
resp, err := client.Execute(reqCtx,
               apiclient.Request{Method:"GET", Path:"/me/actions", Query:{"status": statusFlag}},  // Query omitted when --status absent
               &list)                                                         // 2xx → list populated; non-2xx → *ResponseError; etc.
out := formatMyActions(list)                                                  // projection + "more available" when list.Meta.Pagination.HasNextPage
```

---

## Interactions

- **Validate-first**: `validateStatus` runs before `seam.assemble`, so an unsupported `--status` costs no network call (a tripwire fake asserts the transport is never invoked on rejection) — the 011 `validateInclude` discipline.
- **Build-once / send-once flow**: `runMyActions` calls `seam.assemble` once and `seam.newClient` once, then `Execute` once. Resolution already happened at assembly (009); `me actions` re-resolves nothing and never reads `ctx.Cred.Token`.
- **Decode target**: `Execute` receives `&list` (the envelope); on 2xx the body decodes into it, on non-2xx it is left untouched and a `*ResponseError` is returned (010's decode-or-skip contract).
- **First page only**: exactly one `Execute` call; `formatMyActions` reads `list.Meta.Pagination.HasNextPage` to decide whether to append the signal. No second page is fetched (Pagination, 016).
- **Classification reuse**: `classifyClientError` is the one place 010's typed errors become an `Outcome`; `runMyActions` calls it rather than inlining its own `errors.As` chain (011 ADR-3).
- **Seam injection**: the command is constructed with a seam (`newMyActionsCommand(productionSeam{})` under `me` in `Assemble()`); tests construct it with a fake seam binding a fake `http.RoundTripper`, so every branch runs offline and off the real `~/.glassfrogrc`.

---

## Error Communication

`runMyActions` returns exactly one code-free `Outcome` per invocation; the shared `classifyClientError` maps the API client's typed error to it (the command then maps `Outcome`→error-channel for dispatch, and `ExitCode` maps `Outcome`→process code at the single registry). The mapping is 011's, unchanged:

| Input | `Outcome` | `ExitCode` |
|---|---|---|
| `validateStatus` rejection (before any request) | `UsageError` | `2` |
| `nil` (2xx success, incl. empty list) | `Success` | `0` |
| `*AuthError{NoCredentials}` | `UsageError` | `2` |
| `*AuthError{CredentialError}` | `RuntimeError` | `1` |
| base-URL error (from `newClient`) | `UsageError` | `2` |
| `*DecodeError` | `RuntimeError` | `1` |
| `*ResponseError` (generic non-2xx) | `APIError` | `3` |
| `*TransportError` | `NetworkUnavailable` | `6` |

- **Discrimination order**: `*AuthError` is matched (via `errors.As`) before `*TransportError`, preserving 010/011's discipline so 007's fail-safe is never mislabelled transport.
- **No new mapping to guard**: 013 adds no `Outcome`/`ExitCode` case, so 011's exhaustiveness table test still covers the classifier; 013's tests assert `runMyActions` returns the expected `Outcome` per branch and that `validateStatus` rejects before any request (transport tripwire).
- **No secret anywhere**: none of the messages or the projection renders the token; `me actions` never reads `ctx.Cred.Token`. Pinned across success and every error branch.
- **Fail-safe**: `ExitCode`'s `default→1` backstops any unmapped `Outcome`.

---

## Consistency Notes

- **Reuses 011 unchanged**: `me actions` is the first command to *consume* 011's read-surface foundation (the `glassfrog` package, `classifyClientError`, `Outcome`/`ExitCode`, `--base-url`) without extending it — proof the 011 surface is reusable as designed. It adds no exit code and no classifier branch.
- **Reuses 012's list foundation**: the `me` parent, `Pagination`, the envelope, and the signal renderer. The `actions` leaf is a sibling of the `roles` leaf. If 013 lands before 012, it creates the shared `Pagination`/envelope/`me`-parent in place and 012 reuses them (first-to-land-creates; 005/006/007 pattern).
- **`validateStatus` introduced here, reused by 014**: the spec-sourced status set and the pure validator. 014 (My Projects) calls the same validator; the set is identical for both endpoints. Whether it lives in `my_actions.go` or a small shared helper is a build detail; the contract is "validate before any request, reuse one set."
- **No new configuration of its own**: `me actions` introduces no env var or `.glassfrogrc` key; `--base-url`/`GLASSFROG_BASE_URL`/`base_url` are owned upstream (008/011), `GLASSFROG_TOKEN`/`token` by 005. `--status` is a request-shaping flag, not configuration.
- **Specification touchpoint**: a package-API accord, extending the read surface 011 established. No `accords/` directory exists, so there are no cross-spec accord patterns to align against beyond the DECISIONS precedent.
