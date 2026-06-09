# Interface Accord: Role Reads — Specification

**Feature**: 025-role-reads
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-2 (grow `Role`, add `RoleDetail` + leaf models in `internal/glassfrog`), ADR-1 (`roles` command shape), ADR-3 (default walk vs `--first-page`), ADR-4 (validate include, pass id through).

---

This accord pins the **Go API surface** Role Reads introduces: the `internal/glassfrog` schema growth the reads decode into (and that the downstream per-role specs reuse), the `internal/cli` command shape (the injected seam, the pure run/validate functions), and the two new `internal/render` template keys. The CLI-facing surface — command, flags, output, exit codes — is in `025/interface-cli.md`. Field names and concrete Go types are a build detail; the **shapes, signatures, and projected fields** are the contract. Everything in `internal/apiclient`, `internal/paging`, `internal/output`, and `internal/render`'s engine is consumed **unchanged**.

---

## Surface

### `internal/glassfrog` — schema growth (ADR-2; conforms to 011 ADR-1)

Plain JSON-tagged structs, tolerant of unknown/extra fields. Leaf package — no transport, no cobra.

| Type | Shape | Notes |
|---|---|---|
| `Role` (grown) | existing `ID`,`Name`,`Purpose`,`Accountabilities`,`Domains` (011/012) **+** `Type`, `ParentRoleID *string`, `HasSubroles bool`, `Flags []string`, `Fillers []Actor`, `Tags []string` | The full `Role` spec shape. The list decodes `Page[Role]` (016). One canonical type — never a second. |
| `RoleDetail` | embeds `Role`; **+** `Assignments []Assignment`, `Subroles []Role`, `ParentRole *Role`, `Policies []Policy`, `Notes []Note`, `Skills []SkillSummary` | The `GET /roles/{id}` body's `data`. Related fields are nil/empty unless `?include`d. Nested `Subroles`/`ParentRole` are plain `Role` → no recursion. |
| `Assignment` | `ID`, `ActorID`, `RoleID`, (focus/election fields decoded, not all projected) | Minimal; reused by Role Reads' `--include=assignments` and future specs. |
| `Policy` | `ID`, `Title`, `Body` (✓ title projected) | Reused by Role Policies (#34). |
| `Note` | `ID`, `Title`, `Body` (✓ title + body projected); `RoleID`/`CreatedAt`/`UpdatedAt` decoded | Per `spec.yaml` `Note` (required `title` + `body`; no `text` field). Reused by future note reads. |
| `SkillSummary` | `ID`, `Name`; **no** `Content` | `?include=skills` returns summaries only; full content via a future `GET /skills/{id}`. |
| `RoleDocument` (or reuse a generic `Document[T]`) | `Data RoleDetail` | The single-object `{data: …}` envelope (distinct from the paginated `Page[T]`). Whichever single-object read lands first may generalize this to `Document[T]`. |

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newRolesCommand` | `(seam rolesSeam) *cobra.Command` | Guard-registered `roles` leaf (`Use:"roles [id]"`, `Args: cobra.MaximumNArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares the list + single flags; reads inherited `--base-url`/`--output`; delegates to `runRoles`. **Replaces** the existing stub `roles` group (its `list`/`get` children are removed); the existing `Assemble()` wiring is updated to this seam-taking constructor. |
| `runRoles` | `(cfg rolesConfig) (Outcome, error)` | Pure over injected values. Branches on whether an id is present: validates flag combos + (single) `--include` + resolves `--output` **before** assembly; then list → `runRolesList`, single → `runRoleGet`. Writes result to `cfg.stdout`, diagnostics/notes to `cfg.stderr`; returns the code-free `Outcome`. |
| `runRolesList` | `(cfg, exec, format) (Outcome, error)` | Walks `GET /roles` to completion in **every** format (the format changes rendering, not fetch depth). Human: `paging.All[Role]` → render `org-roles`. Structured: `paging.All[json.RawMessage]` → aggregate the verbatim per-role bytes into `{data:[…]}` (018 fidelity; the synthesized envelope drops per-page `meta`). `--first-page` opts out to one page in both formats. A first-page failure (no records) reports like any read error; a mid-walk failure renders the partial set + stderr note + non-zero via `classifyClientError(Stop)` — identical signalling in both formats. |
| `runRoleGet` | `(cfg, id string) (RoleDetail, Outcome)` | One `Execute` into a `RoleDetail` document; `?include=` from validated `--include`. |
| `validateRolesInclude` | `(targets []string) error` | Rejects an unsupported `--include` value against `{assignments,subroles,parent_role,policies,notes,skills}` before any request (011 `validateInclude` shape). |
| `validateRolesFlags` | `(hasID bool, flags…) error` | The flag-combination guard: `--include` requires an id; list filters/`--first-page`/`--per-page` forbid an id. Returns a usage error naming the misuse. |
| `rolesSeam` | interface: assemble the `ConnectionContext` from a base-URL value, build an executor (`*apiclient.Client` wrapped by `RetryExecutor`) usable by both a direct `Execute` and `paging.All` | Production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS` + `NewRetryExecutor`; tests bind a fake `Executor` returning canned pages / single responses. |

**No `Outcome`/`ExitCode` edit** — Role Reads reuses the categories 011/015 landed. `renderResult[T]` (020) is reused for output dispatch; both reads register a render key (below).

### `internal/render` additions

| Key | Formats | Data | Notes |
|---|---|---|---|
| `org-roles` | `full`, `compact` | `[]glassfrog.Role` | Org-wide list projection (025 interface-cli Output). **New** key — distinct from the shipped `roles` key/templates `glassfrog me roles` uses (untouched). |
| `role` | `full`, `compact` | `glassfrog.RoleDetail` | Single role + guarded per-`include` sections (omit when not requested; explicit-absence marker when requested-but-empty), via 019's `missingkey=error` + `{{if}}` pattern. **New** singular key — distinct from the shipped `roles`. |

The registry exhaustiveness guard (PR #10 `len`+comma-ok) asserts both keys carry both formats.

### Consumed unchanged (not defined here)

- `apiclient.AssembleFromOS`, `NewClientFromOS`, `(*Client).Execute`, `apiclient.Request`, the typed errors, and `NewRetryExecutor`/`RetryExecutor` (010/017).
- `paging.All[T]`, `paging.Result[T]`, `paging.WithPageSize` (016); `glassfrog.Page[T]`/`Pagination` (016).
- `output.ResolveFormat`/`OutputFormat`/`RenderSuccess`/`ErrorEnvelope` (018/020); `render.Render(key, format, v)` (019); the `cli` `renderResult[T]` dispatch (020).
- `classifyClientError`, `Outcome`, `ExitCode` (011/015).

**Example (shapes, not literal values)**:
```
// single read
if err := validateRolesInclude(includes); err != nil { return UsageError, err }   // before any call
ctx := seam.assemble(baseURLFlag)
ex,  err := seam.executor(ctx)                                                     // *Client wrapped by RetryExecutor
var doc glassfrog.RoleDocument
resp, err := ex.Execute(reqCtx,
              apiclient.Request{Method: "GET", Path: "/roles/" + id,
                Query: url.Values{"include": {strings.Join(includes, ",")}}},        // apiclient.Request.Query is url.Values
              &doc)                                                                 // 2xx → doc.Data; non-2xx → *ResponseError
renderResult("role", resolvedFormat, doc.Data)                                      // 020 dispatch

// list (default walk) — filters is a url.Values built from the validated flags.
// EVERY format walks to completion; the format only changes how the set renders.
req := apiclient.Request{Method: "GET", Path: "/roles", Query: filters}
if machineFmt, ok := format.MachineFormat(); ok {                                   // structured
    res := paging.All[json.RawMessage](reqCtx, ex, req)                             // per-role raw bytes preserved
    doc, _ := aggregateRawRoles(machineFmt, res.Records)                            // {"data":[<raw>,…]} (018 fidelity)
    stdout.Write(doc)
    // res.Stop != nil → reportIncompleteWalk (partial set + stderr note + non-zero)
} else {                                                                            // human
    res := paging.All[glassfrog.Role](reqCtx, ex, req)
    text, _ := render.Render("org-roles", humanFmt, res.Records)
    stdout.Write(text)                                                              // res.Stop != nil → same incomplete note
}
```

---

## Interactions

- **Validate-before-call**: `validateRolesFlags` + `validateRolesInclude` + `output.ResolveFormat` all run before `seam.assemble`, so a misuse or bad include/format costs no network call (a tripwire fake asserts the executor is never invoked on rejection).
- **One executor, two consumers**: the seam builds the executor once; the single read calls `Execute` directly, the list passes the same executor to `paging.All`. Resolution happened at assembly (009); the reads re-resolve nothing and never read `ctx.Cred.Token`.
- **Decode targets**: human list → `Page[Role]` (per page, by the walker); structured list → `Page[json.RawMessage]` (per-role raw bytes preserved across the walk, then aggregated into `{data:[…]}`); single → `RoleDocument` (`{data: RoleDetail}`) for the human path, raw `json.RawMessage` verbatim for the structured path (018 ADR-2).
- **Completeness → exit, format-independent**: both walks render the gathered set and, on a mid-walk failure, write the stderr incomplete note and return the classified non-zero `Outcome` (`reportIncompleteWalk`); a first-page failure (no records) reports like any read error. The `--first-page` opt-out renders one page with `Success` and a stderr note when more exist. Structured and human signal incompleteness the same way — neither relies on in-band `meta`.

---

## Error Communication

`runRoles` returns exactly one code-free `Outcome`; `classifyClientError` maps any typed client error to it (the command maps `Outcome`→dispatch's error channel; `ExitCode` maps `Outcome`→process code at the single registry). The full mapping (incl. local validation rows) is the table in `025/interface-cli.md`. Salient points:

- **Discrimination order**: `*AuthError` matched before `*TransportError` (007's fail-safe must not be mislabelled transport — 010's discipline).
- **No new codes**: `roles` adds no `Outcome`/`ExitCode` case; 015's landed split already supplies `permission`(4)/`rate-limit`(5) for 401/403/429.
- **Exhaustiveness guard**: `classifyClientError`'s table test keeps its `len`+comma-ok completeness check.
- **No secret anywhere**: no message or projection renders the token; the reads never read `ctx.Cred.Token`.

---

## Consistency Notes

- **Schema package** (`internal/glassfrog`, 011 ADR-1): Role Reads grows the shared `Role` and adds `RoleDetail` + the leaf models. The per-role specs (#33/#34/#38) **reuse** `Policy`/`Assignment`/etc. — this accord sets that precedent rather than letting each redefine.
- **Walker** (`016/interface-spec.md`): first consumer of `paging.All` in a command. The default path walks; the `--first-page` opt-out reuses the landed 012–014 single-page-signal shape instead of a `WithMaxPages` cap, preserving 016's `Result.Complete == (Stop==nil)` invariant (plan ADR-3).
- **Output dispatch** (`018`/`019`/`020`): reuses `renderResult[T]` and the two-package (`output`/`render`) split; adds two **new** render keys (`org-roles`, `role`) distinct from the shipped `roles` key (`me roles`), imports neither package into the other.
- **Exit-Code Convention** (`004` + `exitcode.go`): no registry edit; reuses the frozen mapping and 011's `classifyClientError`.
- **Distinct from `me roles`** (012): shares the grown `Role`, not the command — org-wide vs token-scoped.
- **Specification touchpoint** in a project with no `accords/` directory: no cross-spec accord patterns to align against; conforms to the in-repo precedent set by 010/011/014's `interface-spec.md`.
