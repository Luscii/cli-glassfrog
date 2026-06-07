# Interface Accord: My Projects — Specification

**Feature**: 014-my-projects
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1 (`Project` joins `internal/glassfrog`, reuses `Pagination`/envelope + `validateStatus`), ADR-2 (no `?include`), ADR-3 (injected seam + pure `runMyProjects`/`formatMyProjects`).

---

This accord pins the **Go API surface** My Projects introduces: the `Project` resource model added to `internal/glassfrog` and the `my projects` command's internal shape (the injected seam, the pure `runMyProjects`/`formatMyProjects`). It **reuses** — and does not redefine — 011's `internal/glassfrog` foundation, `classifyClientError`, `Outcome`/`ExitCode`, and persistent `--base-url`; 012's `my` parent, `Pagination`, list envelope, and signal renderer; and 013's `validateStatus` + status set. The CLI-facing surface is in `014/interface-cli.md`; this file is the package-level contract the Builder implements and the Verifier tests against. Field names and concrete Go types are a build detail; the **shapes, signatures, and which fields are projected** are the contract.

---

## Surface

### `internal/glassfrog` (EXTENDED — add `Project`; reuse `Pagination` + envelope)

Plain JSON-tagged structs decoded from API responses. No transport, no cobra, no exit codes. Decoding is **tolerant of unknown/extra fields**.

| Type | Shape (fields → projected?) | Notes |
|---|---|---|
| `Project` (NEW) | `ID` ✓ (`proj_…`), `Status` ✓, `Description` ✓, `RoleID` ✓ (`role_…`, **nullable** — null for non-role-owned projects), `Tags []string` ✓ (when present), `HasSubProjects` ✓, `HasActions` ✓; `IndividualInitiative`, `ParentProjectID` (nullable), `CreatedAt`/`UpdatedAt`, `Link` (nullable), `Note` (nullable) decoded, not projected | The `/me/projects` list item. `ID` is the machine-actionable handle. The `sub_projects`/`actions` embed arrays are **not** modelled (no `?include` on this operation — ADR-2). |
| `Pagination` (REUSED from 012/013) | `PerPage`, `HasNextPage` ✓ (drives the signal), `NextCursor` | Not redefined here. |
| list envelope (REUSED from 012/013) | `Data []Project`, `Meta{ Pagination }` | The `{data, meta.pagination}` body; `Project` decodes through it. |

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newMyProjectsCommand` | `(seam myProjectsSeam) *cobra.Command` | The guard-registered `projects` leaf (`Use:"projects"`, `Args: cobra.NoArgs`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`) attached to the `my` parent (012); local `--status` flag; reads the persistent `--base-url` value; delegates to `runMyProjects`. Wired once under `my` in `Assemble()`. |
| `runMyProjects` | `(cfg myProjectsConfig) (Outcome, error)` | Pure over injected values (the `runMyActions` shape): **reused** `validateStatus` → assemble context → build client → `Execute` → `formatMyProjects` on success / `classifyClientError` on a typed error. Writes the projection to `cfg.stdout`, messages to `cfg.stderr`; returns the code-free `Outcome`. |
| `formatMyProjects` | `(list <glassfrog list-of-Project>) string` | Pure projection renderer (`014/interface-cli.md` defines the fields/order, the empty-result line, and the "more available" signal driven by `Meta.Pagination.HasNextPage`). Unit-tested in isolation. |
| `myProjectsSeam` | interface: assemble the `ConnectionContext` from a base-URL flag value, and build a `*apiclient.Client` over a base `http.RoundTripper` | Production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS`; tests bind a fake transport returning canned `GET /me/projects` responses. Mirrors 013's `myActionsSeam`. |

### Consumed unchanged (not defined here)

- **From `010/interface-spec.md`**: `apiclient.AssembleFromOS`, `apiclient.NewClientFromOS`, `(*Client).Execute`, the `apiclient.Request` descriptor, and the typed errors `*AuthError` / `*TransportError` / `*ResponseError` / `*DecodeError`.
- **From `011/interface-spec.md`**: `classifyClientError(err) Outcome`, the `Outcome` enum, the `ExitCode` mapper (codes `3`/`6`), and the persistent `--base-url` root flag.
- **From `012`**: the `my` parent command, `glassfrog.Pagination`, the list envelope, and the "more results available" signal renderer/convention.
- **From `013/interface-spec.md`**: `validateStatus(status string) error` and the status set. `my projects` calls the same validator — it adds none of its own.

**Example (shapes, not literal values)**:
```
if err := validateStatus(statusFlag); err != nil { return UsageError, err }   // reused from 013, before any I/O
ctx    := seam.assemble(baseURLFlag)
client, err := seam.newClient(ctx)
var list glassfrog.ProjectList                                                // {Data []Project, Meta{Pagination}}  (envelope reused)
resp, err := client.Execute(reqCtx,
               apiclient.Request{Method:"GET", Path:"/me/projects", Query:{"status": statusFlag}},  // Query omitted when --status absent
               &list)
out := formatMyProjects(list)                                                 // projection + "more available" when list.Meta.Pagination.HasNextPage
```

---

## Interactions

- **Validate-first (reused)**: the shared `validateStatus` runs before `seam.assemble`, so an unsupported `--status` costs no network call (a tripwire fake asserts the transport is never invoked on rejection).
- **Build-once / send-once flow**: `runMyProjects` calls `seam.assemble` once and `seam.newClient` once, then `Execute` once. `my projects` re-resolves nothing and never reads `ctx.Cred.Token`.
- **Decode target**: `Execute` receives `&list` (the envelope); on 2xx the body decodes into it, on non-2xx it is left untouched and a `*ResponseError` is returned (010's decode-or-skip contract).
- **First page only**: exactly one `Execute` call; `formatMyProjects` reads `list.Meta.Pagination.HasNextPage` to decide whether to append the signal. No second page is fetched (Pagination, 016).
- **No `include`**: no `?include` is ever added; `has_sub_projects`/`has_actions` are projected as presence signals (ADR-2).
- **Classification reuse**: `classifyClientError` is the one place 010's typed errors become an `Outcome`; `runMyProjects` calls it rather than inlining its own chain.
- **Seam injection**: production wires `newMyProjectsCommand(productionSeam{})` under `my`; tests bind a fake `http.RoundTripper`, so every branch runs offline and off the real `~/.glassfrogrc`.

---

## Error Communication

`runMyProjects` returns exactly one code-free `Outcome` per invocation; the shared `classifyClientError` maps the API client's typed error to it. The mapping is 011's, unchanged:

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

- **Discrimination order**: `*AuthError` is matched before `*TransportError`, preserving 010/011's discipline.
- **No new mapping to guard**: 014 adds no `Outcome`/`ExitCode` case and no validator, so 011's classifier table test and 013's `validateStatus` tests still cover the shared logic; 014's tests assert `runMyProjects` returns the expected `Outcome` per branch and that the reused `validateStatus` rejects before any request (transport tripwire).
- **No secret anywhere**: none of the messages or the projection renders the token; `my projects` never reads `ctx.Cred.Token`. Pinned across success and every error branch.
- **Fail-safe**: `ExitCode`'s `default→1` backstops any unmapped `Outcome`.

---

## Consistency Notes

- **Twin of My Actions (`013`)**: identical command/seam/projection/error shape; the same shared `validateStatus`. Differences: resource `Project`, the `has_sub_projects`/`has_actions` presence fields, and no `--include` (ADR-2). This is the strongest evidence 013's shared validator + projection convention were designed for the siblings.
- **Reuses 011 + 012 unchanged**: the `glassfrog` package, `classifyClientError`, `Outcome`/`ExitCode`, `--base-url` (011); the `my` parent, `Pagination`, envelope, signal (012). 014 adds no exit code, no classifier branch, no validator, no flag registration.
- **No `?include` for `/me/projects`**: the operation documents no `include` parameter; `Project`'s `sub_projects`/`actions` embed arrays are not modelled, and no `--include` flag exists (ADR-2). Embedding is offered only where the operation documents it (`me --include roles`, 011).
- **No new configuration of its own**: `my projects` introduces no env var or `.glassfrogrc` key; connection config is owned upstream (008/011/005). `--status` is a request-shaping flag.
- **Specification touchpoint**: a package-API accord extending the read surface; conforms to 011/013 precedent. No `accords/` directory exists, so there are no cross-spec accord patterns to align against beyond the DECISIONS precedent.
