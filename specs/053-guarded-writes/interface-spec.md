# Interface Accord: Guarded Writes — Specification

**Feature**: 053-guarded-writes
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/ADR-2 — the narrow `IfMatch` request field and the conditional `If-Match` send in `Execute` (a code-API boundary; the exported surface is the interface). Produced by Version Capture on Read (052) `Response.Version()`; consumed by each write command's own read-then-write retrofit and by Stale-Write Surfacing (054).

---

The artifact is a single addition to the `internal/apiclient` package: one narrow field on the existing `Request` descriptor, plus the conditional send that turns it into an `If-Match` precondition inside `Execute`. Its consumer is another Go package (each write command's retrofit), so the "invocation surface" is the new struct field, the "input" is the verbatim version token a caller threads in from 052's `Response.Version()`, and the "constraint" is the verbatim-fidelity + empty-as-absent + method-agnostic contract. The signature is the contract the Builder implements and the Verifier tests against; the exact field-name spelling and doc-comment wording are implementation detail to finalize within this shape. This accord adds **no** CLI, API, output, or event surface — those are explicitly out of scope (the spec's non-behaviors); it is the request-side mirror of 042's read-side `ContentType` field.

---

## Surface

### Package

`internal/apiclient` — the field lives on the `Request` struct in `client.go`, beside the existing `ContentType` field; the send lives in `Execute` in `execute.go`, immediately after the `ContentType` header block. No new type; no change to `Response`, the `executor` interface, `RetryExecutor`, `NewClient`, or `buildURL`. No new imports (it uses the `net/http.Request.Header` `Execute` already builds).

### Field

```go
// IfMatch is the optional resource version sent as the request's If-Match
// precondition header by Execute, only when non-empty (mirrors ContentType,
// 042 ADR-1). Empty for every request that is not guarded — the landed reads
// and writes leave it "", so their outbound request carries no If-Match header
// and stays byte-identical; a guarded write sets it to the version a prior
// single-resource read captured (apiclient.Response.Version(), 052) so the
// server refuses a stale write (412 Precondition Failed) instead of overwriting
// it last-write-wins. The value is sent verbatim — no quoting, unquoting,
// weak-validator ("W/…") handling, or normalization — because the precondition
// must echo the server's token byte-for-byte or risk a spurious 412. A narrow
// field, not a general Header bag — this is the deferred If-Match consumer 042
// named; generalize only when a second request header lands.
IfMatch string
```

### Send (in `Execute`)

```go
if req.IfMatch != "" {
    // Set the precondition only when the caller supplied a version (mirrors the
    // ContentType block above). Method-agnostic: depends only on the field, so a
    // DELETE is guarded like a PUT/PATCH. 007's AuthTransport owns only
    // X-Auth-Token, so If-Match is set here on the built request, before Do.
    httpReq.Header.Set("If-Match", req.IfMatch)
}
```

### Example

```go
// A write command's retrofit (NOT delivered by this spec) — illustrative consumer.
resp, err := exec.Execute(reqCtx, readReq, &resource) // single-resource pre-write read
// ... handle err ...
version := resp.Version()                 // "" when the response carried no ETag (052)
writeReq := apiclient.Request{
    Method:      "PATCH",
    Path:        "/tensions/" + id,
    Body:        body,
    ContentType: "application/json",
    IfMatch:     version,                 // "" → unconditional write (no If-Match sent)
}
_, err = exec.Execute(reqCtx, writeReq, &updated)
```

---

## Interactions

**Caller-driven, conditional**: the precondition is attached only when `IfMatch` is non-empty. Nothing sets it implicitly inside `Execute`, and no existing read or write call site is changed — the field zero-values to `""`, so every landed request stays byte-identical until a write command opts in. This mirrors the 052 (capture mechanism) → consumer split and the 039 (mechanism) → 040 (call-site retrofit) precedent.

**Empty-as-absent**: an unset `IfMatch` and an empty `IfMatch` are indistinguishable — both send no `If-Match` header, so the write proceeds unconditionally (last-write-wins, the API's behavior when `If-Match` is omitted). This is the send-side symmetry with 052's `Version()` sentinel: `Version()` returns `""` for "no version captured", and `""` threaded into `IfMatch` sends no precondition.

**Verbatim forwarding**: `Execute` passes `IfMatch` straight to `Header.Set` with no transformation, so a caller that threads `Version()` → `IfMatch` forwards the server's token unchanged — quotes and `W/` prefix preserved — and the precondition matches what the server expects.

**Method-agnostic, independent of `ContentType`**: the send depends only on `IfMatch` being non-empty, regardless of HTTP method, so a guarded `DELETE` (Tension Discard) behaves like a guarded `PUT`/`PATCH`. `If-Match` and `Content-Type` are separate fields set by separate blocks; a JSON-bodied guarded write carries both, neither displacing the other.

**Determinism**: same `Request` → same outbound headers. The send performs no I/O of its own beyond the existing `Do`.

---

## Error Communication

| Condition | Behavior |
|---|---|
| `IfMatch` non-empty | `Execute` sets `If-Match` to the value **verbatim** (quotes and `W/` prefix preserved) on the outbound request |
| `IfMatch` empty / unset | No `If-Match` header is set; the write proceeds unconditionally — **not** an error |
| Server refuses a guarded write (`412 Precondition Failed`) | Returned as the existing generic `*ResponseError{StatusCode: 412, Header, Body}` (`execute.go`); 053 does **not** interpret, classify, or relabel it — existing 004/015 diagnostics and exit codes apply, and distinct surfacing is Stale-Write Surfacing (054) |

Setting a header has no failure mode of its own: `Header.Set` cannot error. 053 adds no new `Outcome`, no new exit code, and emits no diagnostics — a refused write rides the unchanged non-2xx path.

---

## Consistency Notes

- **Sibling interfaces**: none — this feature has only a specification touchpoint. No CLI/API/output/event surface of its own; it is consumed in-process by Go packages (write-command retrofits and 054).
- **Mirror of 052's read-side accord**: 052 added `Response.Version()` (read-side capture); this adds `Request.IfMatch` + its send (write-side precondition). Together they are the capture-then-send pair the Optimistic Concurrency roadmap defines. The verbatim guarantee is stated on both sides so the token survives capture → forward unchanged.
- **Narrow-field idiom (042 ADR-1 / DECISIONS)**: 042 made `ContentType` a narrow field and deferred a general `Header` bag / the `If-Match` field "until a real consumer lands." This accord is that consumer and honors the idiom — it adds the narrow `IfMatch` field, **not** a header bag. The bag waits for a genuine second request header.
- **Single send seam preserved (010 ADR-1)**: the precondition is set inside the one `Execute` every command calls through, not in a forked `ExecuteIfMatch` path — consistent with the existing `ContentType` handling a reviewer already knows.
- **Interface-level, not frozen**: the field name (`IfMatch`), doc-comment wording, and the exact placement of the conditional block within `Execute` are the Builder's to finalize within this shape and the verbatim / empty-as-absent / method-agnostic contract.
