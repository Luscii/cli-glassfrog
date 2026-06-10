# Interface Accord: Role Policies — Specification

**Feature**: 034-role-policies
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-2 (grow `Policy`, generalize `Document[T]`, refactor `RoleDocument`), ADR-1 (two-command shape), ADR-3 (`--query`/id pass-through, no local validator), ADR-4 (`policies`/`policy` render keys).

---

This accord pins the **Go API surface** Role Policies introduces: the `internal/glassfrog` schema growth the reads decode into, the generalized single-object envelope, the `internal/cli` command symbols, and the two new `internal/render` template keys. The CLI-facing surface — commands, flags, output, exit codes — is in `034/interface-cli.md`. Field names and concrete Go types are a build detail; the **shapes, signatures, and projected fields** are the contract. Everything in `internal/apiclient`, `internal/paging`, `internal/output`, and `internal/render`'s engine is consumed **unchanged**.

---

## Surface

### `internal/glassfrog` — schema growth + envelope generalization (ADR-2; conforms to 011 ADR-1, 016)

Plain JSON-tagged structs, tolerant of unknown/extra fields. Leaf package — no transport, no cobra.

| Type | Shape | Notes |
|---|---|---|
| `Policy` (grown) | existing `ID`, `Title`, `Body` (025) **+** `RoleID string`, `DomainID string`, `CreatedAt string`, `UpdatedAt string` | The full `Policy` spec shape. `role_id`/`domain_id` are nullable in `spec.yaml` — modeled as plain strings (empty = null), mirroring the existing nullable `Body`. One canonical type, grown not duplicated; 025's embedded render reads only `ID`/`Title`/`Body` and is unaffected. |
| `Document[T any]` (new generic) | `Data T` `json:"data"` | The single-object `{data: …}` envelope, the single-read counterpart to the paginated `Page[T]` (016). Generalizes 025's named `RoleDocument` (whose comment invited it). The single policy read decodes `Document[Policy]`. |
| `RoleDocument` (refactored) | `= Document[RoleDetail]` (type alias) or replaced at call sites | 025's `{data: RoleDetail}` envelope becomes an alias of / is replaced by `Document[RoleDetail]`. Keeping it as an alias preserves 025's decode call site and BDD byte-stable (plan Risk 1). |

The list decodes the generic `glassfrog.Page[Policy]` (016) — no new list envelope.

### `internal/cli` additions

| Symbol | Signature (shape) | Description |
|---|---|---|
| `newPoliciesCommand` | `(seam policiesSeam) *cobra.Command` | Guard-registered `policies` leaf (`Use:"policies <role-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`, `SilenceErrors`/`SilenceUsage`); declares `--query`/`-q`, `--first-page`, `--per-page`; reads inherited `--base-url`/`--output`; delegates to `runPoliciesList`. Explicitly wired in `Assemble()`. |
| `newPolicyCommand` | `(seam policiesSeam) *cobra.Command` | Guard-registered `policy` leaf (`Use:"policy <pol-id>"`, `Args: cobra.ExactArgs(1)`, non-empty `Short`); declares **no** list flags (so a list-only flag is a cobra unknown-flag usage error); reads inherited flags; delegates to `runPolicyGet`. Explicitly wired. |
| `runPoliciesList` | `(cfg policiesConfig) (Outcome, error)` | Pure over injected values. Resolves `--output` **before** assembly; builds the `q` query from `--query` only when `Changed()` and non-empty; walks `GET /roles/{id}/policies` to completion in **every** format (format changes rendering, not fetch depth). Human: `paging.All[Policy]` → render `policies`. Structured: `paging.All[json.RawMessage]` → aggregate verbatim per-policy bytes into `{data:[…]}` (018 fidelity). `--first-page` opts out to one page in both formats; mid-walk failure renders partial + stderr note + non-zero via `classifyClientError(Stop)`. Writes result to `cfg.stdout`, diagnostics/notes to `cfg.stderr`. |
| `runPolicyGet` | `(cfg policyConfig, id string) (Outcome, error)` | Resolves `--output` before assembly; one `Execute` into a `Document[Policy]` (`{data: Policy}`); the id is escaped as a single path segment (`url.PathEscape`) but passed through unvalidated (ADR-3). Structured `--output` emits the raw payload verbatim; the human path renders the `policy` template over a `render.PolicyView`. |
| `policiesSeam` | interface: assemble the `ConnectionContext` from a base-URL value, build an executor (`*apiclient.Client` wrapped by `RetryExecutor`) usable by both a direct `Execute` and `paging.All` | Production binds `apiclient.AssembleFromOS` + `apiclient.NewClientFromOS` + `NewRetryExecutor`; tests bind a fake `Executor` returning canned pages / single responses. Shared by both commands (same shape as 025's `rolesSeam`). |

**No `validate…Include`/`validate…Flags` function** — this feature has no closed-enum input and the two-command split makes list-only-ness structural (plan ADR-1/ADR-3). **No `Outcome`/`ExitCode` edit** — reuses the categories 011/015 landed. `renderResult[T]` (020) is reused for output dispatch.

### `internal/render` additions

| Key | Formats | Data | Notes |
|---|---|---|---|
| `policies` | `full`, `compact` | `[]glassfrog.Policy` (a `PoliciesView`) | Per-role list projection (034 interface-cli Output). **New** key. Renders the inherited empty-set line `No policies.` |
| `policy` | `full`, `compact` | `glassfrog.Policy` (a `PolicyView`) | Single policy + full body, scope (`RoleID`/`DomainID`) and timestamps with explicit-absence guards (019's `missingkey=error` + `{{if}}` pattern). **New** key. First template to render a long free-text `Body` as primary content — must not truncate or reflow it. |

The registry exhaustiveness guard (PR #10 `len`+comma-ok) asserts both keys carry both formats.

### Consumed unchanged (not defined here)

- `apiclient.AssembleFromOS`, `NewClientFromOS`, `(*Client).Execute`, `apiclient.Request`, the typed errors, and `NewRetryExecutor`/`RetryExecutor` (010/017).
- `paging.All[T]`, `paging.Result[T]`, `paging.WithPageSize` (016); `glassfrog.Page[T]`/`Pagination` (016).
- `output.ResolveFormat`/`OutputFormat`/`RenderSuccess`/`ErrorEnvelope` (018/020); `render.Render(key, format, v)` (019); the `cli` `renderResult[T]` dispatch (020).
- `classifyClientError`, `Outcome`, `ExitCode` (011/015).

**Example (shapes, not literal values)**:
```
// single read
ctx := seam.assemble(baseURLFlag)
ex,  err := seam.executor(ctx)                                                 // *Client wrapped by RetryExecutor
var doc glassfrog.Document[glassfrog.Policy]
resp, err := ex.Execute(reqCtx,
              apiclient.Request{Method: "GET", Path: "/policies/" + url.PathEscape(id)},
              &doc)                                                            // 2xx → doc.Data; non-2xx → *ResponseError
renderResult("policy", resolvedFormat, doc.Data)                              // 020 dispatch

// list (default walk) — q set only when --query is Changed() and non-empty.
req := apiclient.Request{Method: "GET", Path: "/roles/" + url.PathEscape(roleID) + "/policies", Query: query}
if machineFmt, ok := format.MachineFormat(); ok {                             // structured
    res := paging.All[json.RawMessage](reqCtx, ex, req)                       // per-policy raw bytes preserved
    doc, _ := aggregateRaw(machineFmt, res.Records)                          // {"data":[<raw>,…]} (018 fidelity)
    stdout.Write(doc)                                                         // res.Stop != nil → reportIncompleteWalk
} else {                                                                      // human
    res := paging.All[glassfrog.Policy](reqCtx, ex, req)
    text, _ := render.Render("policies", humanFmt, res.Records)
    stdout.Write(text)                                                        // res.Stop != nil → same incomplete note
}
```

---

## Interactions

- **Resolve-before-call**: `output.ResolveFormat` runs before `seam.assemble`, so a bad `--output` (or a list-only flag on `policy`, caught by cobra at parse) costs no network call (a tripwire fake asserts the executor is never invoked on rejection).
- **One executor, two consumers** within `policies`: the seam builds the executor once; the list passes it to `paging.All`. The `policy` read calls `Execute` directly. Resolution happened at assembly (009); the reads re-resolve nothing and never read `ctx.Cred.Token`.
- **Decode targets**: human list → `Page[Policy]` (per page, by the walker); structured list → `Page[json.RawMessage]` (per-policy raw bytes preserved across the walk, then aggregated into `{data:[…]}`); single → `Document[Policy]` (`{data: Policy}`) for the human path, raw `json.RawMessage` verbatim for the structured path (018 ADR-2).
- **Completeness → exit, format-independent**: the `policies` walk renders the gathered set and, on a mid-walk failure, writes the stderr incomplete note and returns the classified non-zero `Outcome` (`reportIncompleteWalk`); `--first-page` renders one page with `Success` + a stderr note when more exist. Structured and human signal incompleteness the same way — neither relies on in-band `meta`.

---

## Error Communication

Each `run…` returns exactly one code-free `Outcome`; `classifyClientError` maps any typed client error to it (the command maps `Outcome`→dispatch's error channel; `ExitCode` maps `Outcome`→process code at the single registry). The full mapping (incl. local/cobra rows) is the table in `034/interface-cli.md`. Salient points:

- **Discrimination order**: `*AuthError` matched before `*TransportError` (007's fail-safe must not be mislabelled transport — 010's discipline).
- **No new codes**: neither command adds an `Outcome`/`ExitCode` case; 015's landed split already supplies `permission`(4)/`rate-limit`(5) for 401/403/429.
- **Exhaustiveness guard**: `classifyClientError`'s table test keeps its `len`+comma-ok completeness check.
- **No secret anywhere**: no message or projection renders the token; the reads never read `ctx.Cred.Token`.

---

## Consistency Notes

- **Schema package** (`internal/glassfrog`, 011 ADR-1): Role Policies **reuses and grows** the shared `Policy` (025 set the precedent that the per-role specs reuse `Policy`/`Assignment`/etc., not redefine) and generalizes the single-object envelope to `Document[T]` — the list-side `Page[T]` (016) generalization applied to single reads. The `Document[T]` refactor touches 025's single-role decode (kept byte-stable via the alias, plan Risk 1).
- **Walker** (`016/interface-spec.md`): `policies` is a new consumer of `paging.All`; the `--first-page` opt-out reuses 025 ADR-3's landed shape, preserving 016's `Result.Complete == (Stop==nil)` invariant.
- **Output dispatch** (`018`/`019`/`020`): reuses `renderResult[T]` and the two-package (`output`/`render`) split; adds two **new** render keys (`policies`, `policy`); imports neither package into the other. Structured json/yaml needs no change (raw-bytes path, 018 ADR-2).
- **Exit-Code Convention** (`004` + `exitcode.go`): no registry edit; reuses the frozen mapping and 011's `classifyClientError`.
- **No local validator** (plan ADR-3): departs from 011/013/025's `validate…` precedent because there is no closed-enum input to guard; list-only-ness is structural (ADR-1).
- **Specification touchpoint** in a project with no `accords/` directory: no cross-spec accord patterns to align against; conforms to the in-repo precedent set by 010/011/014/025's `interface-spec.md`.
