# Plan: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Role**: Shaper
**Inputs**: `specs/074-post-create-validity-read/spec.md`, `PROJECT.md`, `.score/memory/DECISIONS.md`, `.score/memory/LEARNINGS.md`, `.score/memory/DEPRECATION.md`, the landed `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` (CSG-2), and the live read-only probe recorded in the spec's Clarifications. No SOUL.md at the project root.

---

## System Architecture

The feature adds a **second server exchange inside one command**. `runProposalCreate` (`internal/cli/proposal.go`) keeps its existing shape end to end — resolve the render target, require and read `--changes`, apply the type floor, assemble the connection, send one `POST /proposals` — and gains a **read-back stage** that runs only after that POST succeeds: one `GET /proposals/{id}` against the id the create returned, issued through the same `apiclient.RetryExecutor` the reads already use. The read-back's response carries the two undeclared verdict fields (`valid`, `validation_alerts`) that the probe established are present on the single-proposal read and absent from the list read; those, plus the `available_transitions` the declared schema already carries, are the **verdict**.

Three things move through the command after the create returns:

```
POST /proposals ──► created proposal (prp_ id, status, changes …)
                          │
                          ├─ id ──► GET /proposals/{id} ──► verdict {valid, validation_alerts, available_transitions}
                          │                   │
                          │                   └─ failed? ──► verdict unavailable + reason (NOT a command failure)
                          ▼
                   render: machine → the read-back's raw {data} verbatim  (+ stderr provenance advisory)
                           human  → proposal-created view = shared proposal body + verdict block
```

Four components carry the change, each small:

- **`internal/glassfrog` (model)** — the shared `Proposal` struct grows `Valid *bool` and `ValidationAlerts []ValidationAlert`, plus the new `ValidationAlert` type (`severity`, `path`, `message`). Growing the shared model rather than forking one is the standing grow-not-duplicate precedent (011 ADR-1, followed by 055 ADR-4 / 056 ADR-2). The pointer and the nil-able slice are what make "the server said nothing" distinguishable from "the server said no" (ADR-3).
- **`internal/render` (projection)** — a new `ResourceProposalCreated` key with `proposal-created.full.tmpl` / `proposal-created.compact.tmpl`, and a `ProposalCreatedView` that **embeds** the existing `ProposalView` and adds the verdict. The new templates render the proposal body by invoking the existing `proposal.<format>.tmpl` from the single parsed template set, so there is exactly one source for the body (ADR-4).
- **`internal/cli` (orchestration)** — `runProposalCreate` gains the read-back stage and the verdict assembly. This is the only place control flow changes; no other proposal leaf is touched.
- **`internal/apiclient`** — untouched. The read-back is an ordinary bodyless GET through the landed executor, so it inherits 017's safe-method retry (a 429 or 5xx on the read-back is retried; a POST still never is) with no transport change.

Data ownership is unchanged and one-directional: the server owns validity, the CLI reports it, and nothing about the verdict is computed, cached, or persisted locally (VISION Exclusion 2 and Exclusion 4 both hold by construction — there is no second source of truth and no store).

---

## Architecture Decisions

### ADR-1: Obtain the verdict from a post-create read-back of `GET /proposals/{id}`, and distinguish this from 057's rejected pre-read

**Context**: The spec requires the create result to carry the server's verdict on the created draft. The read-only probe (spec Clarifications) established that `GET /proposals/{id}` carries `valid` and `validation_alerts` and that `GET /proposals` carries neither. Whether the create's own 201 body carries them is unobserved and cannot be settled without performing a real governance create in the live organization (spec Assumptions). Meanwhile 057 ADR-3 is an active precedent that reads, in its own words, that `propose` is *server-authorized* — the command "does NOT pre-read the proposal to check `available_transitions`" because "a pre-read forks the server's authority, adds a round-trip, and risks a stale snapshot." A plan that adds a read to a write command has to say why that precedent does not forbid it.

**Options considered**:
1. **Post-create read-back** — after a successful create, issue one `GET /proposals/{id}` and report what it says. Verified route to the verdict; costs one extra request per create.
2. **Trust the create response's own body** — surface `valid`/`validation_alerts` if the 201 carries them. Zero extra requests, but the premise is unverified, and if the 201 omits the fields (or the server computes validity after responding) the capability silently reports nothing on exactly the shape it exists to catch.
3. **No read at all** — document the accepted-but-invalid shape and leave detection to the operator. Free, and already done: CSG-2 records the shape. It is what the org has today, and CSG-2 is the evidence that a documented shape does not stop a human confirming a dead write.

**Decision**: Option 1 — post-create read-back. The verdict comes from the surface that is *verified* to carry it, and the read is a published v5 operation (`getProposal`), so VISION Principle 1 holds for the operation even though two of its response fields are undeclared.

**This is not the pre-read 057 rejected, and the difference is directional.** 057's rejected pre-read would have had the client inspect `available_transitions` *before* a write in order to decide whether the server would allow it — the client substituting its own gate for the server's, on a snapshot that could be stale by the time the write landed. This read-back runs *after* the write, asks the server what it thinks of an object that already exists, and reports the answer without acting on it. It cannot fork the server's authority because it makes no decision; it cannot be stale in the way 057 feared because there is no subsequent write whose admissibility it is predicting. 057 ADR-3's reasoning is therefore preserved, not diverged from, and this ADR is the boundary statement that keeps the two legible side by side. The related 053 ADR-2 precedent points the same way: it records that "an extra pre-write read, the read-failure path" belongs in its own spec per write call site rather than in the shared mechanism — which is exactly what this spec is for the create call site.

**Consequences**: `proposal create` costs two requests instead of one on the success path, against a per-organization rolling-hour budget (spec Integration Boundaries). Every existing test that asserts the create's success path issues exactly one exchange must be updated — deliberately, since that count is now part of the specified behavior. The no-request-on-rejection tripwire is untouched: all pre-request rejections still return before any assembly, and a failed create still issues exactly one. If a future live create observation shows the 201 already carries a computed verdict, Option 2 becomes available as a pure optimization: the verdict assembly and both render paths stay exactly as designed, and only the second exchange is removed.

### ADR-2: The read-back's failure is isolated — it never becomes the command's failure, and never withholds the id

**Context**: The create has already succeeded when the read-back runs; the `prp_` id is the load-bearing handle a later step needs (055 ADR-4). The spec is explicit that a read-back failure must still report the id with an explicit "verdict unobtainable" and that a missing verdict must never read as a favourable one. The command's existing failure path is the shared `reportFailure` (032), which renders a failure envelope and returns a non-Success outcome.

**Options considered**:
1. **Isolate the read-back** — its error is captured, converted into a "verdict unavailable" reason carried by the result, and the command still reports the successful create with `Success`.
2. **Route the read-back error through `reportFailure`** — uniform with every other exchange in the codebase. But it would report the *create* as failed when the create succeeded, and the failure envelope carries no `prp_` id — so the operator loses the handle to a draft that exists, which is precisely the CSG-2 aftermath the spec forbids.
3. **Retry until a verdict arrives** — strongest guarantee, but an unbounded wait inside a gated write with a human watching, which the spec's non-behaviors forbid.

**Decision**: Option 1 — isolate. The read-back is a *best-effort second exchange whose failure is correlated with nothing the caller can act on*, so it is handled the way a diagnostic on a failure path must be handled: it may not become the failure. Concretely, `runProposalCreate` calls the read-back through a helper that returns `(verdict, unavailableReason)` and never an error the caller propagates; every path out of it leads to reporting the created proposal. The command's outcome after a successful create is `Success` regardless of the verdict's content or availability — including when the server reports `valid: false`, which this spec deliberately does not turn into a failure (that is the sibling Invalid-Create Outcome's job, which adds a new Outcome and exit code at both registry sites).

**Consequences**: A caller can distinguish four states on the success path — valid, not valid, no verdict reported by the server, and verdict unobtainable (with a reason). Exit code 0 covers all four in this spec, which is a knowingly incomplete answer to the underlying problem: an invalid create still exits 0 until the sibling lands. The rate-limit interaction is benign by construction — a 429 on the read-back after retries is a verdict-unavailable reason, not a failed create.

### ADR-3: Model the verdict as a composed, tri-state record — `Valid *bool` plus a nil-able typed alert slice — never an enum and never derived

**Context**: The probe returned four independently-varying facts about live proposals: a `draft_with_conflicts` proposal with `valid: true`; plain `draft` proposals with `valid: false` and populated `validation_alerts`; valid drafts with `available_transitions: []`; and each alert as an object carrying `severity`, `path`, and `message`. The spec forbids inferring validity from status, transitions, or the presence of alerts, and forbids reading an absent verdict as favourable. Neither `valid` nor `validation_alerts` is declared in `spec/glassfrog-api-v5.yaml`, so both may be absent from any given response.

**Options considered**:
1. **Composed record with a pointer flag** — `Valid *bool` (nil = the server said nothing), `ValidationAlerts []ValidationAlert` (nil = absent, empty = present-and-empty), and the existing `AvailableTransitions` read as its own dimension.
2. **A plain `bool`** — simpler to render, but `false` would mean both "the server says invalid" and "the field was absent," collapsing the exact distinction the spec calls the original failure wearing a new mask.
3. **A single derived status enum** — one value like `valid` / `invalid` / `blocked` computed from the flag, the alerts, and the transitions. Renders compactly, but it is a local governance judgment (VISION Exclusion 2) and the probe already disproves the derivations it would rest on.

**Decision**: Option 1 — a composed record. The fields are added to the shared `glassfrog.Proposal` (grow-not-duplicate), decoded forward-compatibly like every sibling field. `ValidationAlert` is a typed struct — `Severity`, `Path`, `Message`, each with an explicit snake_case tag — rather than a free-form map, because the probe observed a stable three-key shape and the spec requires rendering all three; a forward-compatible decode still tolerates extra keys. The verdict presented to the render layer is a small struct holding the flag, the alerts, the transitions, and the unavailable-reason — four fields that vary independently, with no computed roll-up anywhere.

**Consequences**: `Valid *bool` is the one nullable-as-pointer field in a model whose convention is nullable-as-empty-string (`Proposal.TensionID`, `Tension.RoleID`). That divergence is deliberate and belongs in the field's comment: an empty string is a fine stand-in for an absent id because no id is ever legitimately empty, whereas `false` is a legitimate value of `valid`, so the absent case needs its own representation. `missingkey=error` in the render engine means the templates must guard the pointer explicitly rather than relying on a zero value. Any future consumer that wants a single-word summary must compute it at its own layer and own that judgment — this model will not hand one over.

### ADR-4: Render the human path through a create-specific `proposal-created` view that delegates the proposal body to the shared `proposal` template

**Context**: The spec confines the *rendered* verdict to the create result: `proposal list` cannot carry it without a request per row, and `proposal get`'s render is not to be changed. But `internal/render` has one singular `proposal` template pair, shared by create (055 ADR-4), `proposal get` (056), `propose` (057 ADR-2), and `withdraw` (059 ADR-2). Verdict lines added there would appear on all four. Separately, 055 ADR-4 is an active precedent recording that the create's human path renders "the SHARED singular proposal view (with 056)".

**Options considered**:
1. **A create-specific `proposal-created` resource whose templates invoke the shared `proposal` template for the body** and append the verdict block. One source for the body; the verdict reaches exactly one command.
2. **Conditional verdict lines in the shared `proposal` template** — no new resource key. But `proposal get` reads the very endpoint that carries the fields, so the condition would be true there and the verdict would render on `get`, `propose`, and `withdraw` — the widening the spec's clarification excluded.
3. **A standalone `proposal-created` template that restates the body's lines** — full control, and a second copy of eleven body lines that drifts the first time the shared one changes.

**Decision**: Option 1. `templates` is a single `template.Must(...ParseFS(...))` set, so `proposal-created.full.tmpl` can render `{{template "proposal.full.tmpl" .}}` and then emit the verdict; `ProposalCreatedView` embeds `ProposalView`, and Go's field promotion makes the embedded `.Proposal` resolve inside the invoked template unchanged. The compact pair mirrors this with one difference forced by the one-line contract: the shared compact template ends in a newline, and `text/template` cannot capture `{{template}}` output, so the compact wrapper delegates through an `include` helper it can wrap in `trimSpace`. That widens `funcMap` by one pure helper and moves the `templates` assignment into `init()` (the helper refers back to the parsed set, which would otherwise be an initialization cycle) — both recorded in `interface-spec.md`, including the fact that user templates share the FuncMap and can therefore call `include`. Adding a resource touches the two registry sites the engine already has — the `Resource` const block and `builtinResources` — and the existing exhaustiveness test then requires both template files to exist, which is the intended fail-loud.

**Announced divergence from 055 ADR-4** (active precedent, origin `055-proposal-creation`): the create's human render is no longer keyed to `ResourceProposal`. The divergence is deliberately narrow — the shared singular template still renders the proposal body, byte for byte, and the change is that the create wraps it rather than calling it directly. 055 ADR-4's reason for sharing (don't fork the 019/020 human-view contract; don't duplicate the body) is honoured by the delegation. What changes is only that the create's output is a superset. This is a candidate for `/score:deprecate` so a later reader of 055 ADR-4 is not surprised by the call site.

**Consequences**: A user-supplied template (035) at the create call site now renders over `ProposalCreatedView` rather than `ProposalView`; because the embedded fields are promoted, every field path an existing user template could reference still resolves, so no user template breaks. What is *not* promised is that the values are unchanged: where the read-back answered, the rendered proposal is the read-back's (§ Verdict Assembly), so a path like `.Proposal.AvailableTransitions` reflects the second document — deliberately, since the invalid-draft scenario needs the empty set only the read-back reports. `proposal get`, `propose`, and `withdraw` keep `ResourceProposal` and their output is unchanged — which the scenarios should pin, because the shared template is the drift surface.

### ADR-5: In a machine format, emit the read-back's raw `{data}` verbatim, with the create's raw as the fallback, and carry provenance on stderr

**Context**: 018 is a verbatim contract: a structured `--output` emits the server's document as bytes, unreshaped. After this change the command holds *two* server documents for the same object — the create's 201 and the read-back's 200. The spec requires an agent to read the verdict structurally from machine output and requires the verdict's provenance to be carried in every output format.

**Options considered**:
1. **Emit the read-back's raw document; fall back to the create's raw when there is no read-back; put the provenance on stderr.** One server document on stdout, verbatim, and it is the later and richer of the two (it is the only one verified to carry the verdict).
2. **Compose an envelope carrying both documents plus a provenance field** — provenance fully in-band. But it invents a response shape no contract declares and breaks the verbatim contract every other command upholds, which is the drift 018 exists to prevent.
3. **Emit the create's raw document and put the verdict on stderr** — preserves "the create's response is what you asked for." Rejected: the primary consumer is an agent parsing stdout, and putting the verdict out-of-band for that consumer defeats the capability (VISION Principle 3).

**Decision**: Option 1. On stdout the caller gets one verbatim server document; when the read-back succeeded, that document is the read-back's and carries `valid` and `validation_alerts` inline, which is *itself* the structural signal that a read-back produced it. The provenance sentence — that the verdict was read back from the named proposal after the create — rides stderr, which is format-independent and therefore satisfies "in every output format" without touching the verbatim stream. This follows 045's precedent of a stderr advisory that disambiguates otherwise-indistinguishable success outcomes, and keeps the guidance that stdout stays machine-clean.

**Amendment (2026-08-08, after `/score:guard --pre`)**: as first written, this decision left the advisory as prose in every format, which meant a document with no `valid` key could not be told apart from a read-back that never answered without reading human text — contradicting the spec's own accord. The first proposed remedy was to narrow the accord's claim. That was rejected: the accord stated the intent, and the intent was right. **The advisory is format-aware instead**, following the landed diagnostic-rendering precedent (032) that already renders a failure as a structured envelope when a machine format is selected, on the principle that "a failure reads the same way as a success" on the channel an agent parses. Concretely: in a human format the advisory is the prose line it always was; in a machine format it is a small structured document carrying the provenance, and, when the verdict could not be obtained, the reason and the remedy. It rides **stderr** rather than stdout because stdout is occupied by the server's own document on this path — and a CLI-owned diagnostic may have a CLI-owned shape, which is exactly the line 018 draws: server documents are never reshaped, CLI diagnostics were always ours to shape (032 invents an envelope of its own for the same reason). All four verdict states are therefore machine-distinguishable without parsing prose, and 018's verbatim contract is untouched.

**Consequences**: An agent that diffs the two documents will not see the create's own body at all when the read-back succeeds; if that body ever carried something the read-back's does not, it would be lost. Both are the same `Proposal` schema and the probe found the detail read to be a strict superset in practice, so this is recorded as a risk rather than a guard. When the read-back could not be performed the emitted document is the create's, and the stderr advisory states the verdict was unobtainable and why — so the machine path never emits a document *claiming* a verdict it does not have, and never emits nothing at all.

### ADR-6: Obtain the read-back id by decoding the create's response locally — never by a second create-side request

**Context**: The human path already decodes the create's 201 into `Document[Proposal]`, so the id is in hand. The machine path deliberately does not decode: it executes into a `json.RawMessage` and hands those bytes to `output.RenderSuccess`. But the read-back needs the created id, and the path is built from it.

**Options considered**:
1. **Decode the machine path's raw bytes locally into `Document[Proposal]`** to lift the id out, then read back and emit the read-back's raw. No extra request; the raw bytes are retained untouched for the fallback.
2. **Decode into a minimal purpose-built id-only struct** — marginally cheaper, and a second decode shape for the same document that has to be kept in step with the model.
3. **Switch the machine path to the decode-and-re-marshal path** — one code path for both formats, at the cost of re-serializing the server's document, which is exactly what 018's verbatim contract forbids.

**Decision**: Option 1. `glassfrog.Document[Proposal]` already exists and decodes forward-compatibly, so lifting the id costs one local unmarshal and no request. The raw bytes are kept as-is for the fallback emission, so verbatim holds. The path is built exactly as `proposal get` builds it — `"/proposals/" + url.PathEscape(id)` — reusing the landed escaping rather than re-deriving it.

**Consequences**: There is one new failure mode with no server involvement: the create returned 2xx bytes from which no `prp_` id can be lifted (an undecodable or id-less body). That cannot produce a read-back, so it reports the create's document with a verdict-unavailable reason naming the cause, rather than fabricating a path from an empty id and issuing a request that would 404. This needs a scenario; it is the machine-path twin of the human path's existing decode-error handling.

---

## Verdict Assembly and Rendering Design

The verdict is assembled once and consumed by both render paths, so the two cannot drift apart in what they consider "the verdict."

| Verdict dimension | Source | Absent means |
|---|---|---|
| Validity flag | read-back `valid` | the server stated no verdict — reported as such, never as valid, never as invalid |
| Validation alerts | read-back `validation_alerts` | no alerts stated; an empty array is reported as no alerts, which is distinct from an absent field only in that the field was present |
| Available transitions | read-back `available_transitions` (declared) | reported as none available — never restated as a validity verdict |
| Unavailable reason | the read-back's own failure, or an unliftable id | the read-back was performed and answered |

Rendering per path:

- **Human (`full`)** — the shared proposal body, then a verdict block: the validity line (valid / not valid / not reported by the server, or unobtainable with its reason), each alert on its own line with severity, path, and message, and a provenance line naming the read. The `Valid` pointer is guarded explicitly in the template because the engine runs `missingkey=error` and a nil dereference would otherwise surface as a render error.
- **Human (`compact`)** — the shared compact body plus a short verdict token and an alert count, keeping the one-line shape the compact format promises.
- **Machine (`json`/`yaml`)** — the read-back's raw `{data}` verbatim on stdout (ADR-5); the advisory rendered **in the selected format** on stderr, carrying the provenance and any unavailable reason plus its remedy. `read_back` is the field that answers what the document cannot: whether the CLI managed to ask.
- **User template (035)** — renders over `ProposalCreatedView`; every existing field path still resolves through the embedded view (ADR-4).

A render failure on either path stays what it is today: a `*RenderError` mapped to `RuntimeError`(1) with stdout left empty, buffer-then-write.

---

## Cross-cutting Concerns

**Error handling.** Three failure classes are now distinct and must stay so. A pre-request rejection (missing `--changes`, bad source, floor violation, invalid `--output`) returns before assembly with no request at all — unchanged, and the existing transport tripwire still proves it. A failed create routes through `reportFailure` exactly as today, with no read-back attempted. A failed read-back is not a command failure at all (ADR-2): it produces a reason string, and the command reports the created proposal with `Success`.

**Observability.** stderr gains one advisory line on the create success path: the verdict's provenance, or the reason it is unavailable. 017's retry notices already write there, so a read-back that is retried is visible without new plumbing. Nothing new is logged about the change set, and no token or request body ever reaches either stream (CONSTITUTION II holds unchanged — the verdict carries no secret material).

**Configuration.** Nothing new is configurable. The spec forbids an opt-out of the read-back, so no flag is added; `--output` and `--base-url` behave as they do today, and the read-back inherits the resolved connection rather than resolving its own.

**Testing strategy.** The architecture implies four test surfaces. Model decoding needs the tri-state cases explicitly — `valid: true`, `valid: false` with populated alerts, `valid` absent, `validation_alerts: []` versus absent — because that is the distinction ADR-3 exists to preserve and a value-typed decode would pass a naive test. Render needs both new templates against all four verdict states, plus an assertion that `proposal`-keyed output is byte-identical to today for `get`/`propose`/`withdraw` (the shared-template drift surface ADR-4 creates). Orchestration needs a two-exchange fake seam: create-then-read on the success path, create-only on the create-failure path, and read-back-fails on the isolation path. And the registry exhaustiveness test picks up the new resource for free, provided both template files land with it.

---

## Implementation Strategy

Three phases, strictly ordered — each is independently reviewable and the later ones cannot be written before the earlier ones exist.

**Phase 1 — Verdict model.** Grow `glassfrog.Proposal` with `Valid *bool` and `ValidationAlerts []ValidationAlert`; add the `ValidationAlert` type. Decode tests cover the tri-state matrix. Nothing else changes: every existing consumer of the model compiles and behaves identically, because no template or command reads the new fields yet. Depends on nothing.

**Phase 2 — Create-specific render path.** Add `ResourceProposalCreated` to the const block and to `builtinResources`, the `ProposalCreatedView` embedding `ProposalView` plus the verdict, and the two delegating templates. Render tests cover the four verdict states in both formats and pin the unchanged `proposal`-keyed output. Depends on Phase 1 for the verdict fields the view carries.

**Phase 3 — Read-back orchestration.** Add the read-back stage to `runProposalCreate`: lift the id (decoded in both paths, per ADR-6), issue the GET through the existing executor, assemble the verdict, dispatch to the new render path in the human case and to the read-back's raw bytes in the machine case, and emit the stderr advisory. Update the existing create tests whose exchange count changes, and add the isolation and id-unliftable cases. Depends on both earlier phases.

The phase boundaries are also the natural PR boundaries: Phase 1 and 2 are additive and invisible to the CLI's behavior, so only Phase 3 changes what a user sees — which keeps the behavioral review concentrated in one place.

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| The create's 201 already carries a computed verdict, making the read-back a needless request on every create | Medium — unobserved, and only a real governance create settles it | Low — one wasted read per create; the surfacing behavior is correct either way | ADR-1 records the fallback explicitly: if a live create shows the verdict in the 201, delete the second exchange and keep every other layer. The first real create after this ships should have its raw 201 body inspected and the finding recorded in LEARNINGS. |
| Validity is computed asynchronously, so an immediate read-back reports no verdict (or a stale favourable one) for a draft the server later marks invalid | Low-medium — a sibling aggregate on this resource was observed lagging by hours (LEARNINGS 2026-08-05, F8) | High — the capability would report reassuringly on exactly the shape it exists to catch | The design reports what the server says at read time and never infers; "no verdict" is a first-class reported state, so the failure mode is visible rather than silent. Bounded re-reading was considered and excluded by the spec; if lag is observed in practice, that exclusion is the thing to revisit. |
| Verdict lines leak onto `proposal get` / `propose` / `withdraw` through the shared template | Low — the design routes around it | Medium — silently widens two read commands the spec confined | ADR-4 keeps the verdict in a separate resource key; the render tests pin `proposal`-keyed output as unchanged, which fails loudly if a later change moves verdict lines into the shared body. |
| The doubled request cost pushes a create over the hourly budget that would previously have fit | Low | Low — surfaces as a verdict-unavailable reason, never as a lost write | ADR-2's isolation makes this benign by construction; the reason names the exhausted budget so the operator knows to re-read rather than re-create. |
| An invalid create still exits 0 until the sibling outcome lands, so an automated caller keyed on exit codes still treats a dead proposal as success | High — it is the specified scope | Medium — the human-visible half of the problem is solved, the machine-gate half is not | Stated as scope, not hidden: the spec's non-behavior and ADR-2 both name it. Invalid-Create Outcome is the dependent backlog item and should follow closely; shipping this without it is an improvement, not a fix. |

---

## What This Plan Does Not Cover

- **Protocol-level detail** — the exact flag surface, stderr advisory wording, template line formats, view field names, and the verdict's rendered vocabulary are the interface skill's concern. This plan fixes the shape and the layering, not the strings.
- **Executable scenarios** — the spec's eight driving scenarios and four validation scenarios become feature files in the scenarios step; the two new cases this plan surfaced (an unliftable id in the machine path, and `proposal`-keyed output staying unchanged) belong there too.
- **The outcome and exit code for an invalid create** — deliberately excluded by the spec. It adds a new `Outcome` and a new exit code, which touches both registry sites (`exitcode.go` and the error-envelope kind switch) and is the sibling Invalid-Create Outcome's whole content.
- **Settling whether the create response carries the verdict** — requires a real governance create in the live organization, which produces a draft that may need web-UI cleanup. Carried as ADR-1's fallback and the first risk above, not resolved here.
- **Any change to the write-safety guardrail (063)** — the gated leaf is still `proposal create`; the read-back adds no command surface, so the guardrail's positional-subcommand match and its contract facts are untouched.
- **The other proposal writes** — `propose`, `withdraw`, and `respond` are out of scope by the spec's non-behavior and are not touched by any phase here.
