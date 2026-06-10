# Plan: Output-Aware Failure Rendering

**Feature**: 032-output-aware-failure-rendering
**Role**: Shaper
**Inputs**: spec.md (032); PROJECT.md; `.score/memory/DECISIONS.md` (81 entries — relevant precedent: 031 `Diagnose(err)→Diagnostic{Category,Cause,NextStep}` and "032 reads the whole value to render per `--output`, sourcing status/body from the wrapped `*ResponseError` per 018"; 020 `OutputFormat.IsStructured()/MachineFormat()` + "Consumed by 032"; 018 `ErrorEnvelope`/`ErrorDetail`/`RenderError` with "the typed-error→envelope mapping lives at the cli/020 boundary"; render.go is the single importer of both `internal/output` and `internal/render`; buffer-then-write→RuntimeError(1)); `.score/memory/LEARNINGS.md` (table-driven exit-code detectors, temp-file stderr capture, no-silent-failure in helpers); `.score/memory/DEPRECATION.md` (no entry touches the failure-render surface). No SOUL.md.

---

## System Architecture

The CLI already produces, on a command-execution failure, two values 032 needs and a third it preserves:

- a **normalized diagnostic** — `Diagnose(err) → Diagnostic{Category, Cause, NextStep}` (031, `internal/cli/diagnostic.go`), total over every failure family;
- a **resolved output format** — `output.OutputFormat` (020), with `MachineFormat() (output.Format, bool)` distinguishing the structured members (`json`/`yaml`) from the human members (`full`/`compact`);
- the **unified error envelope** machinery — `output.ErrorEnvelope{Error ErrorDetail}` + `output.RenderError(Format, ErrorEnvelope)` (018), which already encodes a complete JSON/YAML failure document or returns a render error, never a fragment.

The missing wire is the **failure-render dispatch**: the point that, given the resolved format and the diagnostic, routes a failure to the human surface (cause-plus-next-step on stderr, today's behavior) or the structured surface (the 018 envelope on stdout). That point already exists in skeleton form as the **single failure chokepoint** `reportClientError(stderr, err)` in `internal/cli`, called from ~19 command-execution failure sites (the four `renderResult[T]` reads via `render.go`, plus the paginated `roles`/`subroles`/`tree` reads and `me`). 032 makes that one chokepoint format-aware and threads the resolved format and stdout to it. The envelope-mapping logic lives in `internal/cli` — the only package that imports both `internal/output` (the envelope/encoder) and `internal/apiclient` (the wrapped `*ResponseError` carrying status and raw body); `internal/output` stays a transport-free leaf, exactly as 018 requires.

```
command-execution failure (transport / decode / typed-API / pre-request)
        │
        ▼
  reportFailure(stdout, stderr, format, err)        ← the one chokepoint (was reportClientError)
        │  refineClientError(err) ; d := Diagnose(err)
        │
        ├─ human  (full/compact): renderDiagnostic(d) ──────────────► stderr   (unchanged; stdout empty)
        │
        └─ structured (json/yaml): build ErrorEnvelope from d + wrapped *ResponseError
                                   output.RenderError(machineFmt, env) ─► stdout (sole document)
        │
        ▼ returns d.Category → Exit-Code Convention (004) maps it to the process code (unchanged)
```

A distinct, parallel reporter — `reportIncompleteWalk` / `reportIncompleteSubrolesWalk` — handles the **partial-data** case (a mid-walk failure after some pages already rendered to stdout). It stays a stderr note in **every** format and is deliberately *not* routed through 032 (see ADR-3): a partial structured document already occupies stdout, and a second document there would break 018's one-document-per-channel guarantee.

---

## Architecture Decisions

### ADR-1: Extend the single failure chokepoint into a format-aware reporter, rather than add a parallel renderer

**Context**: Failures currently reach one helper, `reportClientError(stderr io.Writer, err error) (Outcome, error)` — the chokepoint 031 delegates `Diagnose` to. It writes `renderDiagnostic(d)` to stderr regardless of format (the documented interim gap, 020/019). 032 must render in the selected format. The spec scopes 032 to *command-execution* failures (transport, decode, typed-API, pre-request), which is exactly the set that funnels through this chokepoint — usage errors take the separate `dispatch.go` plain-text path, and the invalid-selector error is 020's.

**Options considered**:
1. **Extend the one chokepoint** — evolve `reportClientError` into `reportFailure(stdout, stderr io.Writer, format output.OutputFormat, err error)`, thread the resolved `format` + `cfg.stdout` to all ~19 call sites. One rendering contract; the compiler enforces that no site is missed (signature change → every un-updated site fails to build).
2. **Wrap at the `renderResult` level only** — make only the generic success-dispatch carry format-aware failures. Smaller diff, but the paginated `roles`/`subroles`/`tree` reads bypass `renderResult` and would keep plain-text failures under `json`/`yaml` — a silent half-fix.

**Decision**: Option 1. The chokepoint is the established single-source-of-truth seam (031's `Diagnose`, 011's `classifyClientError`); making it format-aware keeps one contract and reaches every command-execution failure. `format` and `cfg.stdout` are already in scope at every site (each read's run function takes `format`; every `cfg` carries `stdout`), so threading is mechanical.

**Consequences**: ~19 call-site edits, all mechanical (`reportClientError(cfg.stderr, err)` → `reportFailure(cfg.stdout, cfg.stderr, format, err)`). The signature change makes the Go compiler the completeness backstop — an un-threaded site is a build error, not a silent plain-text leak. The three category-only callers (`Diagnose(err).Category` in the partial-result and output-format paths) are unaffected — they don't render, they only read the category.

### ADR-2: Own the Diagnostic→ErrorEnvelope mapping in `internal/cli`; extend `output.ErrorDetail` with an `omitempty` `next_step` field

**Context**: 018's envelope (`output.ErrorDetail{Message, Kind, Status, Body}`) carries only a human `message` today. The 032 clarification (2026-06-10) decided the next step must be surfaced under structured formats as **its own parseable element**, distinct from the cause — the operator is usually an AI agent reading the recovery action programmatically. The mapping needs `Diagnose`'s output (cli) and the wrapped `*ResponseError`'s status + raw body (apiclient); `internal/output` must import neither (018's leaf invariant).

**Options considered**:
1. **Distinct `next_step` field on `output.ErrorDetail`** (`json:"next_step,omitempty"`), populated by a cli-side mapper from `Diagnostic.NextStep`. Machine-actionable; additive and omitempty, so 018's existing envelope tests are unaffected.
2. **Fold the next step into `message`** (`"cause — next step"`, mirroring `renderDiagnostic`) — no envelope change, but the next step isn't independently parseable; rejected by the clarification.

**Decision**: Option 1. Add `NextStep string \`json:"next_step,omitempty"\`` to `output.ErrorDetail`. A new cli-side helper (e.g. `errorEnvelopeFor(err) output.ErrorEnvelope`) maps: `Message ← d.Cause`; `NextStep ← d.NextStep` (omitted when empty — the internal-error fallback and bare general-API errors carry none); `Kind ← kind(d.Category)` (ADR-4); `Status`/`Body ← errors.As(err, *ResponseError)` (omitted when absent). The helper lives in `internal/cli` (the only importer of both `internal/output` and `internal/apiclient`), keeping `internal/output` transport-free.

**Consequences**: 018's envelope gains one optional field — additive, no existing structured-success or structured-error test changes shape. The cli↔output mapping that 018's `ErrorDetail` doc anticipated ("kind from classifyClientError, status/body from `*ResponseError`") now exists. The field name `next_step` is the interface-level spelling (interface-spec.md pins it alongside 018's envelope shape).

### ADR-3: Structured envelope → stdout; human text → stderr; command-execution failures only; the partial-walk incompleteness note stays on stderr in every format

**Context**: Two channel questions. (a) Where does each format's failure land? (b) The paginated reads can fail **mid-walk after some pages already rendered**; under a structured format the partial `{data:[…]}` document is already on stdout. 018 guarantees one complete document per channel.

**Options considered**:
1. **Clean failures format-aware; partial-walk incompleteness stays a stderr note in all formats.** The nothing-on-stdout chokepoint (`reportClientError`) renders structured→stdout / human→stderr; the partial-data reporters (`reportIncompleteWalk`/`reportIncompleteSubrolesWalk`) keep their stderr note unchanged because stdout already holds the partial structured document. The exit code still signals the failure.
2. **Envelope the incompleteness too** — emit an error envelope on stdout even when a partial document is already there. Breaks 018's one-document-per-channel; an agent parsing stdout sees two documents. Rejected.
3. **Suppress the partial document under structured formats and emit only the envelope.** Loses data the walk already gathered — worse for the agent. Rejected.

**Decision**: Option 1. Structured envelope → stdout (the channel an agent parses, so success and failure read the same way); human cause-plus-next-step → stderr (today's behavior, byte-preserved). 032's surface is exactly the existing clean-failure chokepoint; the partial-data reporters are out of scope and keep their stderr note in every format. Usage errors (`dispatch.go`) and the invalid-selector error (020) stay plain-text — out of scope per spec.

**Consequences**: The existing code's separation of "clean failure, nothing on stdout" (`reportClientError`) from "partial data already on stdout" (`reportIncompleteWalk`) is the seam — 032 changes the former, leaves the latter. No two-documents-on-stdout hazard. Under structured formats a paginated read that fails mid-walk with partial data writes the partial `{data:[…]}` on stdout plus a plain-text incompleteness note on stderr and a non-zero exit — the structured contract is met for the data, the incompleteness rides the secondary channel. (Documented as a known nuance in Cross-cutting Concerns.)

### ADR-4: Envelope `kind` is a 1:1 map from the `Outcome` category; the raw body is included only when it is valid JSON

**Context**: `Diagnostic.Category` is the `Outcome` enum (`Success, UsageError, RuntimeError, NetworkUnavailable, APIError, PermissionError, RateLimited`). 018's `ErrorDetail.Kind` expects a lowercased taxonomy token. The wrapped `*ResponseError.Body` is raw bytes that 018 nests as **structured** data (`json.RawMessage`), and `RenderError` returns an error (no document) if `Body` is not valid JSON.

**Options considered**:
1. **1:1 kind map + body-when-valid.** A small total mapper `kind(Outcome) string` (`usage`/`runtime`/`network`/`api`/`permission`/`rate-limit`), and include `Body` only when `json.Valid(raw)` — otherwise omit it (the envelope still carries message/kind/status). Keeps the render total for realistic inputs; a malformed API body never suppresses the diagnostic.
2. **Pass the body raw and let `RenderError` fail on non-JSON.** A non-JSON error body would fail the whole structured render → RuntimeError(1), hiding the cause behind an internal error. Rejected — the diagnostic is more valuable than the raw body.

**Decision**: Option 1. `kind(Outcome)` is a net-new total mapper in `internal/cli` (a `switch` with a default to `runtime`/internal, mirroring the table-driven exhaustiveness discipline — LEARNINGS PR #10). `Body` is set only when `json.Valid`. The buffer-then-write safety net is retained: should `RenderError` still fail (a genuinely impossible-to-encode envelope), nothing partial reaches stdout and the outcome maps to `RuntimeError(1)` — matching the spec's "a structured render that cannot complete" scenario and 018/019/020's existing pattern.

**Consequences**: The structured render is effectively total for real failures (message + kind always present; status/body optional). The kind map is one more exhaustiveness-guarded table — a future `Outcome` value must be added here or it falls to the safe default. The body-validity gate means an agent never loses the cause to a malformed upstream body.

---

## Cross-cutting Concerns

**Error handling / the render's own failure**: structured rendering follows the established buffer-then-write contract — `RenderError` returns a complete document or an error; on error nothing reaches stdout and the outcome is `RuntimeError(1)`. The failure-render path never itself becomes a silent failure (LEARNINGS: no-silent-failure in helpers).

**Token safety**: the rendered failure must never carry the API token or auth header. This is preserved for free — `Diagnose` already produces token-free `Cause`/`NextStep` (017/031), and `output.RenderError` "adds nothing to the values it is handed." No new redaction logic; a regression test asserts no secret in any of the four renders (spec validation scenario).

**Exit code**: unchanged. `reportFailure` returns `d.Category` exactly as `reportClientError` did; 004 maps it. Rendering and code stay derived from the same `Diagnose` value, so they cannot disagree (the `reportClientError` invariant, preserved).

**The partial-walk nuance** (ADR-3): under a structured format, a mid-walk failure with partial data yields a partial structured document on stdout + a plain-text incompleteness note on stderr + non-zero exit. This is the one place a structured run emits non-structured text on the secondary channel; it is deliberate (the alternative loses data or doubles documents on stdout) and documented for the scenarios/validate steps.

**Testing strategy**: unit-test the pure mapper (`errorEnvelopeFor` / `kind`) — every `Outcome`→kind, next_step present/absent, status/body present/absent, body-valid vs body-invalid (omitted). Behavioral BDD over the chokepoint: each format × {transport, typed-API-with-body, internal-fallback} asserting channel (stdout vs stderr), document validity, next-step presence, and the unchanged exit code. The signature change to `reportFailure` is its own compile-time completeness check across the ~19 sites.

---

## Implementation Strategy

**Phase 1 — Envelope mapping (pure, no call-site churn).** Add `NextStep string \`json:"next_step,omitempty"\`` to `output.ErrorDetail`. Add the cli-side `kind(Outcome) string` mapper and `errorEnvelopeFor(err error) output.ErrorEnvelope` (refine once, `Diagnose`, map cause/next_step/kind, extract status/body from the wrapped `*ResponseError` with `json.Valid` gating the body). Unit-test the mapper exhaustively. No behavior changes yet — the new helper is unused.

**Phase 2 — Make the chokepoint format-aware and thread the format.** Evolve `reportClientError` into `reportFailure(stdout, stderr io.Writer, format output.OutputFormat, err error) (Outcome, error)`: structured → `output.RenderError(machineFmt, errorEnvelopeFor(err))` to stdout (buffer-then-write); human → `renderDiagnostic(d)` to stderr (unchanged). Update all ~19 clean-failure call sites to pass `cfg.stdout` + `format`. Leave `reportIncompleteWalk`/`reportIncompleteSubrolesWalk` untouched (ADR-3). BDD scenarios over the four formats × failure families.

Phase 2 depends on Phase 1 (`errorEnvelopeFor` must exist). The split keeps the pure mapping testable in isolation before the mechanical threading lands.

---

## Risks

- **A missed call site leaks plain text under `json`/`yaml`** (low likelihood, medium impact). Mitigation: the `reportFailure` signature change makes every un-updated site a compile error — the type system enforces completeness. No grep-for-sites risk.
- **Body extraction lost across refinement** (medium likelihood, low impact). The refined `*ProblemError` must still expose the raw `*ResponseError.Body` via `errors.As` (it `Unwrap`s to it per diagnostic.go). Mitigation: a unit test asserts the body survives refinement into the envelope; if a future refine drops it, the test fails.
- **Extending 018's `ErrorDetail` couples 032 to 018's struct** (low likelihood, low impact). The field is additive + omitempty, so 018's envelope tests are unaffected. Mitigation: keep the field declaration in `internal/output` (018's home) and the *population* in `internal/cli`, so ownership stays clean.
- **The partial-walk structured nuance surprises a consumer** (low likelihood, low impact). An agent could parse the partial stdout document as a complete success. Mitigation: documented in Cross-cutting Concerns and surfaced in a validation scenario; the non-zero exit code is the authoritative failure signal.

---

## What This Plan Does Not Cover

- **Protocol-level shapes** — the literal `next_step` field name, the exact `kind` token spellings, and the envelope's field ordering are the interface skill's concern (interface-spec.md, alongside 018's envelope contract).
- **Executable scenarios** — the Gherkin for the four-formats × failure-families matrix and the channel/exit-code assertions are the scenarios skill's output.
- **Task decomposition** — the PR-sized split of Phase 1 / Phase 2 is the tasks skill's output.
- **Usage-error and invalid-selector rendering** — explicitly out of scope (spec non-behaviors); they keep their plain-text paths in `dispatch.go` / 020.
- **Changing the success path, the pagination walk, or 018's encoders/019's templates** — untouched; 032 is purely the failure-side wire.
