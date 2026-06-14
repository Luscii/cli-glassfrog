# Interface Accord: Version Capture on Read — Specification

**Feature**: 052-version-capture-on-read
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/ADR-2 — the version-capture accessor on `apiclient.Response` (a code-API boundary; the exported surface is the interface). Consumed in-process by Guarded Writes (053).

---

The artifact is a single addition to the `internal/apiclient` package: an accessor on the existing `Response` type that hands back the resource version a single-resource read carried. Its consumer is another Go package (053's guarded-write call site), so the "invocation surface" is the exported method, the "input" is the `*Response` the caller already holds from `Execute`, and the "constraint" is the verbatim-fidelity contract. The signature is the contract the Builder implements and the Verifier tests against; the exact method name spelling and doc-comment wording are implementation detail to finalize within this shape. This accord adds **no** CLI, API, output, or request-side surface — those are explicitly out of scope (the spec's non-behaviors; the `If-Match` request field is 053's).

---

## Surface

### Package

`internal/apiclient` — the addition lives alongside the `Response` type in `execute.go`. No new imports (it reads the `net/http.Header` the `Response` already holds). No new type; no change to `Request`, `Execute`, the `executor` interface, or `RetryExecutor`.

### Method

```go
// Version returns the resource version captured from this read response — the
// ETag header, verbatim — for a later guarded write (053) to send as If-Match.
// It returns the header value exactly as the server stated it: no unquoting, no
// weak-validator ("W/…") stripping, no normalization. When the response carries
// no ETag, it returns "" (the "no version captured" sentinel). Header lookup is
// case-insensitive (net/http canonicalization), so ETag/Etag/etag all match.
func (r *Response) Version() string
```

The value is derived purely from `r.Header` — `Version()` stores nothing, mutates nothing, sends nothing, and renders nothing. It is meaningful only on a successful read: a failed read returns an error and no `*Response`, so there is no receiver to call it on.

### Example

```go
// 053 (Guarded Writes) — illustrative consumer, NOT delivered by this spec.
resp, err := exec.Execute(reqCtx, readReq, &resource) // single-resource pre-write read
// ... handle err ...
version := resp.Version()      // "" when the response carried no ETag
// 053 then sends `version` as the If-Match header on the write, when non-empty.
```

---

## Interactions

**Caller-driven, lazy**: capture happens only when a consumer calls `Version()` on a read's `*Response`. Nothing captures eagerly inside `Execute`, and no existing read call site is changed — single reads keep discarding `*Response` until 053 opts in. This mirrors the 039 (mechanism) → 040 (call-site retrofit) split.

**Single-resource only, by construction**: `Version()` reports whatever `ETag` the one response carried. There is no list-walk seam — the pagination walker / `aggregateRawData` path never produces a single `*Response` for a consumer to query — so collection reads yield no per-resource version. The single-resource discipline is upheld by *where* the accessor is called (053's single-resource pre-read), and this spec ships no call site that violates it.

**Empty-as-absent**: an absent `ETag` and an empty `ETag` both return `""`. A consumer reads `""` as "no version to guard with" and proceeds last-write-wins (the API's behavior when `If-Match` is omitted). The two cases are intentionally indistinguishable — neither is a usable precondition.

**Determinism**: same `Response` → same value. `Version()` performs no I/O and never blocks.

---

## Error Communication

| Condition | Behavior |
|---|---|
| Response carries an `ETag` | Returns the header value **verbatim** (quotes and `W/` prefix preserved) |
| Response carries no `ETag` | Returns `""` — a valid "nothing captured" outcome, **not** an error |
| Read failed (non-2xx, transport, decode) | No `*Response` exists; `Version()` is never reached — existing diagnostics and exit codes (004/015) are untouched |

`Version()` has no failure mode of its own: it cannot error or panic, because `http.Header.Get` returns `""` for any absent key. It emits no diagnostics.

---

## Consistency Notes

- **Sibling interfaces**: none — this feature has only a specification touchpoint. No CLI/API/output/event surface of its own; it is consumed in-process by a Go package (053).
- **Reuses the existing `Response.Header` seam**: `Response` already exposes `Header http.Header` "for the sibling capabilities" (Rate-Limit Handling 017 reads rate-limit headers off the non-2xx path). `Version()` is the read-side ETag reader over that same already-present surface — no new transport state, consistent with the existing header-consumer pattern.
- **Narrow-field / anti-speculation idiom (DECISIONS, 042)**: 042 kept `Request.ContentType` a narrow field and deferred a general `Header` bag / the `If-Match` field "until a real consumer lands." This accord honors that — it adds **no** `If-Match` field to `Request`; the request-side change belongs to 053, the consumer that grounds it. 052 is read-side only.
- **Verbatim fidelity is fixed here**: the no-normalization guarantee is pinned at the capture point so 053 can forward the value unchanged; a downstream consumer that unquotes or normalizes would risk a spurious 412 (a tested-against hazard, plan Risk 2).
- **Interface-level, not frozen**: the method name (`Version` vs `ETag`), doc-comment wording, and whether it lives as a method or a small free function are the Builder's to finalize within this shape and the verbatim/empty-sentinel contract.
