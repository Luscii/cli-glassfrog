# Interface Accord: Stale-Write Surfacing — CLI

**Feature**: 054-stale-write-surfacing
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-1 (distinct `StaleWrite` Outcome → new exit code `7`, classified by `412` status at the existing registry sites) and ADR-2 (the `412`-specific cause and re-read/retry next step composed at the existing `Diagnose` site). Producer of the `412`: Guarded Writes (053). Code map owner: Exit-Code Convention (004). Diagnostic composer: Diagnostic Normalization (031). Renderer: Output-Aware Failure Rendering (032).

---

This accord adds **one new process exit code** to the published convention and **one new per-status diagnostic** (a cause + next step) to the existing failure-reporting surface. It defines no new command, flag, or output format — a stale write is surfaced through the same `$?` + stderr diagnostic path every API failure already uses. It is the surface half of the Optimistic Concurrency split: Guarded Writes (053) produces the `412 Precondition Failed`; this capability makes that refusal legible and machine-distinguishable.

---

## Surface

### The new exit code

A guarded write the server refuses with `412 Precondition Failed` (a clobber — the resource changed since it was read) now terminates the process with a **new, distinct code `7`**, instead of the generic API-error code `3` it shared with every other uncategorized non-2xx before this capability.

The published, frozen category↔code registry, extended with the new row (the `0`–`6` rows are unchanged from 004):

| Code | Category | Emitted when | Producer |
|---|---|---|---|
| `0` | Success | A command completed, or a group/root/`--version`/help resolved. | Argument Dispatch (002) |
| `1` | Internal error (safety net) | A resolved action failed, or termination matched no more-specific category (incl. panic). | Argument Dispatch / panic-recover |
| `2` | Usage error | Unknown command, or an unknown/missing flag or unexpected positional. | Argument Dispatch (002) |
| `3` | API error | The API returned an error not covered by a more specific category. | API client (011/015) |
| `4` | Permission / authorization error | The API rejected the caller's auth or membership (`401`/`403`). | API Error Extraction (015) |
| `5` | Rate-limited | The API reported the rate limit was exceeded (`429`). | API Error Extraction (015) |
| `6` | Network-unavailable | The API could not be reached at all (connection, DNS, timeout). | API client (011) |
| **`7`** | **Stale-write (precondition failed)** | **A guarded write was refused because the resource changed since it was read (`412`).** | **Stale-Write Surfacing (054)** |

`7` is the next previously-unused value, extending the operational band (`3`–`6` → `7`). It is the first code added beyond 004's originally-published `0`–`6` set — the extension mechanism 004 designed for (new category → single registry site → new unused code → never renumber). No existing code is renumbered or reassigned. A single invocation always exits with exactly one code.

### The diagnostic (stderr)

A `412` outcome is reported as a normalized diagnostic carrying the same three fields every failure carries — a **cause**, a **category** (stale-write), and a **next step** — rendered to stderr per the selected `--output` by Output-Aware Failure Rendering (032), unchanged. Distinct to `412`:

- **Cause** — identifies the failure as a precondition failure caused by the resource changing since it was read. When the API supplied its own error `detail`/`title`, that text is surfaced (the API's own words win, as for every typed API error); when it did not, the cause is derived from the `412` status and names the precondition failure / changed-since-read — never invented.
- **Next step** — tells the operator to **re-read the resource to obtain its current version, then retry the write**. This replaces the misleading generic step (*"the API rejected the read; check that the token has access…"*) a `412` previously inherited from the generic bucket.

The exact cause and next-step wording is the Builder's to finalize within this contract.

---

## Interactions

- **Reading the result**: a CI runner or AI-agent operator inspects `$?` after the invocation. `$? == 7` means specifically "the resource changed under me" — distinguishable from any other API error without parsing text.
- **Branching example** (illustrative, not a mandated UX):
  ```
  glassfrog tension update ten_123 --body "…"
  case $? in
    0) : ;;                       # success
    7) reread_and_retry ten_123 ;; # stale write — re-read for the current version, then retry
    4) : ;;                        # permission — escalate
    *) : ;;                        # other failure
  esac
  ```
- **Producer-classifies model**: the producer classifies the outcome's category; the registry maps category→code and never re-derives it. The category is assigned from the **surfaced `412` status alone** — never from whether this CLI sent an `If-Match`, which command failed, or which resource it targeted — exactly as `401`/`403`/`429` are classified by status.
- **Most-specific category wins**: `412` is classified as stale-write, not the generic API error — so it exits `7`, not `3`.
- **Extension**: this is the new category added at the single registry site; existing codes are never renumbered, so a consumer's existing branch on `0`–`6` keeps its meaning across releases.

---

## Error Communication

| Condition | Exit code | Cause | Next step |
|---|---|---|---|
| Guarded write refused, `412`, API supplied a `detail`/`title` | `7` (stale-write) | The API's own detail/title, surfaced as the cause | Re-read the resource for its current version, then retry the write |
| Guarded write refused, `412`, no readable API detail | `7` (stale-write) | Status-derived: a precondition failure — the resource changed since it was read | Re-read the resource for its current version, then retry the write |
| Any other non-2xx (`404`, `500`, …) | unchanged (`3`, or its specific code) | unchanged | unchanged |
| `401`/`403` / `429` | unchanged (`4` / `5`) | unchanged | unchanged |

- **Never zero on failure**: a refused guarded write exits non-zero (`7`), never `0` (Fail Safe, CONSTITUTION III).
- **No fabrication**: the cause prefers the API's own words and otherwise states only the precondition-failed / changed-since-read semantics the `412` status defines (CONSTITUTION VIII). The next step is the actionable recovery, not a guess.
- **This capability renders no text itself** — it supplies the category, cause, and next step into the diagnostic that 032 renders and the code `7` that 004's registry emits. No new stderr writer is introduced.
- **No new shell-reserved code**: `7` is outside the shell-reserved range (`126`, `127`, `128 + N`), so `$?` stays unambiguous.

---

## Consistency Notes

- **Exit-Code Convention (`004/interface-cli.md`)**: 004 published the `0`–`6` registry and the extension rule — "a new outcome category is added at the single registry site, taking a new previously-unused code; existing codes are never renumbered." This accord is the first to exercise that rule with a code beyond the original band; `7` is anticipated growth, not a contract break.
- **API Error Extraction (015) split precedent**: 015 split the generic `APIError` into `PermissionError`(4)/`RateLimited`(5) by classifying `401/403`/`429` at the per-status classifier. This accord is the identical move for `412` → stale-write(7) — same site, same status-driven shape a reviewer already knows.
- **Diagnostic Normalization (031)**: 031 owns the cause and per-status next-step composition. This accord adds a `412` next-step arm (re-read/retry) and a `412`-aware cause beside the existing `401`/`403`/`429` arms; every other status's cause and next step are untouched.
- **Output-Aware Failure Rendering (032)**: renders the stale-write diagnostic per `--output` with no change — the diagnostic keeps its three-field (cause/category/next-step) shape, so the renderer needs no new case.
- **Guarded Writes (053, producer)**: 053's accord states a refused guarded write "rides the unchanged non-2xx path … distinct surfacing is Stale-Write Surfacing (054)." This accord is that surfacing — it consumes the `412` 053 produces and gives it the distinct code/cause/next-step, closing the capture (052) → send (053) → surface (054) chain.
- **No sibling interface types**: this feature has only a CLI touchpoint. It adds no API, output-format, event, or specification surface, and no cross-package code-API surface (the change is internal to `internal/cli`'s classification and registry).
- **Interface-level, not frozen**: the published code value (`7`), the category name (stale-write), and the re-read/retry next-step intent are the contract; the exact cause and next-step wording and the internal symbol names are the Builder's to finalize within this shape.
