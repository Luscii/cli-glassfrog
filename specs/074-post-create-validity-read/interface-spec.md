# Interface Accord: Post-Create Validity Read — Specification

**Feature**: 074-post-create-validity-read
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-3 (tri-state composed verdict on the shared model), ADR-4 (create-specific render key delegating to the shared template, over a view that embeds it), ADR-6 (id lifted by local decode of the retained raw bytes), ADR-2 (isolated read-back helper returning a reason, never an error). Cross-cutting: the Verdict Assembly and Rendering Design table.

---

This accord pins the **Go API surface** the feature introduces across three packages: the two new decoded fields and one new type in `internal/glassfrog`, the new render key / view / verdict projection / templates in `internal/render`, and the one new orchestration helper in `internal/cli`. The CLI-facing surface — command, output, stderr, exit codes — is in `interface-cli.md`. Following 055's precedent for this same command, the Go surface gets its own accord because the Builder needs the shapes pinned before either the render or the orchestration work can start.

---

## Surface

### `internal/glassfrog` — two fields on the existing `Proposal`, one new type

`Proposal` grows two fields. Field order follows the response's own ordering (after the response counts, alongside the other server-computed state) and every field keeps an explicit snake_case tag — `encoding/json` is case-insensitive but does not bridge underscores, so an untagged `ValidationAlerts` would silently never bind.

```go
// Valid is the server's own verdict on this proposal. It is a POINTER, not a
// bool: the field is NOT declared in spec/glassfrog-api-v5.yaml, so it may be
// absent, and `false` is a legitimate value — a plain bool would make "the
// server said the draft is invalid" indistinguishable from "the server said
// nothing", which is the silent-default trap 074 exists to close. nil means the
// server stated no verdict; it never means valid and never means invalid.
// Observed carried by getProposal and NOT by listProposals (074 probe).
Valid *bool `json:"valid"`

// ValidationAlerts carries the server's blocking and advisory alerts on this
// proposal. Also undeclared in the contract. A nil slice means the key was
// absent or null; a non-nil empty slice means the server stated an empty list —
// both mean "no alerts", and neither is a validity verdict on its own (an entry
// carries its own severity).
ValidationAlerts []ValidationAlert `json:"validation_alerts"`
```

```go
// ValidationAlert is one entry of a proposal's validation_alerts. Undeclared in
// the v5 contract and observed live (074 probe): a three-key object carrying the
// severity, the element path the alert concerns, and the server's own message.
// Typed rather than a free-form map because all three keys are rendered; decoding
// stays forward-compatible, so an added key is ignored rather than fatal.
type ValidationAlert struct {
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}
```

**Not added**: any method that computes a summary verdict from these fields plus status or transitions. The dimensions stay separate at the model layer (ADR-3); a consumer wanting one word derives it at its own layer and owns that judgment.

**Unchanged**: every other `Proposal` field, `ProposalChange` and its `UnmarshalJSON`, `ResponseSummary`, `CreateProposalRequest`/`Body`/`NewCreateProposalRequest`, `ProposalVote`, and the response-input types. Existing consumers of `Proposal` (`proposal get`, `list`, `propose`, `withdraw`) compile and behave identically — they simply never read the new fields.

### `internal/render` — one resource key, one view, one verdict projection, two templates

```go
// ResourceProposalCreated is the create-specific projection (074): the created
// proposal PLUS the server's verdict read back from getProposal. Distinct from
// ResourceProposal, which stays the shared singular projection used by
// proposal get (056), propose (057), and withdraw (059) — the verdict is
// confined to the create result, so it may not ride the shared key. Its
// templates render the body by invoking the shared proposal templates, so there
// is exactly one source for the body's lines.
ResourceProposalCreated Resource = "proposal-created"
```

`ResourceProposalCreated` is appended to `builtinResources` — the second of the two registry sites a new resource has. The existing exhaustiveness test then requires `proposal-created.full.tmpl` and `proposal-created.compact.tmpl` to be present and parseable, which is the intended fail-loud.

```go
// ProposalCreatedView is the data the proposal-created templates render. It
// EMBEDS ProposalView, so every field path an existing user template (035) could
// reference on the create — .Proposal.ID, .Proposal.Status, … — still resolves
// through Go's field promotion, and the invoked shared template finds .Proposal
// unchanged.
type ProposalCreatedView struct {
	ProposalView
	Verdict ProposalVerdict
}

// ProposalVerdict is the RENDER projection of the server's verdict: display
// labels resolved in Go, because text/template treats any non-nil pointer as
// truthy and would render a pointer-to-false as valid. Validity is a label for
// ONE dimension, not a roll-up — Alerts render separately, and available
// transitions stay a line of the shared body.
//
// Validity and Compact are two renderings of the SAME four states, both produced
// here so the state vocabulary is single-sourced (plan § Verdict Assembly: the
// compact format carries "a short verdict token and an alert count"). They are not
// interchangeable: the full block can afford the server's reason text, and a
// compact one-liner cannot — appending an arbitrarily long server-derived reason
// behind a 36-character id would destroy the one-line contract.
type ProposalVerdict struct {
	// Validity is the `full` block's label: one of "valid", "not valid",
	// "not reported by the server", or "unavailable — <reason>".
	Validity string
	// Compact is the compact line's label: one of "valid", "not valid",
	// "validity not reported", or "validity unavailable", with " (N alert(s))"
	// appended when the server stated at least one alert — in EITHER validity
	// state, so a favourable verdict carrying an advisory alert stays visible.
	Compact string
	// Alerts is what the server stated; empty renders no alerts block.
	Alerts []glassfrog.ValidationAlert
	// Source is the provenance line's value: the read-back it came from, or an
	// explicit statement that no verdict was obtained.
	Source string
}

// NewProposalVerdict maps the decoded tri-state (valid pointer, alerts, and an
// unavailable reason) onto the display labels. It is the SINGLE source of BOTH
// label vocabularies — the cli package never hand-builds these strings, and no
// template composes one from parts. A non-empty unavailableReason wins: no
// validity is claimed and no alerts are carried, because none were stated by the
// server, so neither label ever carries an alert count in that state.
func NewProposalVerdict(valid *bool, alerts []glassfrog.ValidationAlert, unavailableReason string, id string) ProposalVerdict
```

Template files, both rendering the body through the shared template from the single parsed set:

```gotemplate
{{/* proposal-created.full.tmpl */}}
{{template "proposal.full.tmpl" .}}  Validity:       {{.Verdict.Validity}}
{{if .Verdict.Alerts}}  Alerts ({{len .Verdict.Alerts}}):
{{range .Verdict.Alerts}}    - [{{.Severity}}] {{.Path}}: {{.Message}}
{{end}}{{end}}  Verdict source: {{.Verdict.Source}}
```

```gotemplate
{{/* proposal-created.compact.tmpl */}}
{{trimSpace (include "proposal.compact.tmpl" .)}}  {{.Verdict.Compact}}
```

The compact wrapper delegates through `include`, not `{{template}}`, because `interface-cli.md` contracts the compact form as **one line** and the shared `proposal.compact.tmpl` ends in a newline. `text/template` cannot capture `{{template}}` output, so the shared body has to come back as a string before it can be trimmed. The alternative — a conditional verdict line inside the shared template — is precisely the leak ADR-4 exists to prevent, so the delegation moved into the wrapper instead.

**Changed in `internal/render`, beyond the new view and resource key**:

- `funcMap` gains **one** helper, `include(name string, data any) (string, error)`, which executes a named built-in into a string and returns the engine's error unchanged, so a failure inside the included template still fails the outer render loud. It is pure over its inputs and token-free like every sibling helper. Note the reachability: user templates parse into a clone of the built-in set and therefore **share this FuncMap** (035, ADR-2), so `include` is callable from a caller-authored template. The data-only sandbox still holds by construction — `include` exposes no file, network, or exec surface, and can only render a built-in over data the caller was already handed — but the callable surface is wider than the built-ins, and that is the accord's statement of it.
- `templates` moves from a declaration-time initializer to assignment in `init()`. `include` refers back to the parsed set, so a declaration-time initializer would make `funcMap` ↔ `templates` an initialization cycle. `init` runs after package-level vars, so `template.Must` still panics a parse failure at package init — the build-time-defect property is unchanged.
- `userTemplateBase` (usertemplate.go) is assigned in that same `init()`, because its clone must be taken after `templates` exists.

**Unchanged**: `proposal.full.tmpl`, `proposal.compact.tmpl`, `ProposalView`, `ResourceProposal`, `Render`, `RenderError`, and every other resource key. No conditional verdict line is added to the shared templates.

### `internal/cli` — one helper, one changed call site

```go
// readBackProposalVerdict performs the post-create read-back (074): ONE
// GET /proposals/{id} through the supplied executor, decoding the server's
// verdict. Its failures are NEVER the command's failures — it returns a
// human-readable reason instead of an error, so a failed read-back can never
// withhold the created proposal's id or produce a non-zero exit (ADR-2). The raw
// bytes are returned so the machine path can emit the read-back's own document
// verbatim; they are nil when no read-back produced a body.
//
// Returns (proposal, raw, reason): reason is empty when the read-back answered.
func readBackProposalVerdict(ctx context.Context, exec executor, id string) (glassfrog.Proposal, json.RawMessage, string)
```

```go
// verdictSource is the CLI-owned advisory the create writes to stderr (074). It is
// rendered as prose in a human format and structurally in a machine format,
// following the format-aware diagnostic convention (032). It is a CLI shape, not a
// server shape — which is what keeps 018's verbatim contract untouched: the server's
// document on stdout is never reshaped, and this never rides stdout.
//
// ReadBack answers "did the CLI manage to ask?" — the one question the emitted
// document cannot answer, and therefore the field that makes all four verdict states
// machine-distinguishable. ProposalID is omitted when no id could be determined,
// Reason is omitted when the read-back answered, and Remedy is omitted when there is
// none to name: an absent key means "not applicable", never an empty string.
type verdictSource struct {
	ReadBack   bool   `json:"read_back"`
	ProposalID string `json:"proposal_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Remedy     string `json:"remedy,omitempty"`
}

// newVerdictSource builds the advisory from the read-back's outcome. It is the single
// source of the advisory's content in BOTH renderings — the prose line is derived
// from the same value the structured form encodes, so the two can never disagree
// about whether the CLI asked.
func newVerdictSource(id string, unavailableReason string) verdictSource
```

`runProposalCreate`'s contract is otherwise unchanged: same `proposalCreateConfig`, same signature, same pre-request ordering, same one `POST`. Its success path gains: lift the id (from the already-decoded document in the human path; from a local unmarshal of the retained raw bytes in the machine path — ADR-6), call the helper, build the verdict through `render.NewProposalVerdict`, render through `ResourceProposalCreated` or emit the read-back's raw, and write the stderr advisory.

**Unchanged**: `proposalCreateConfig`, `proposalSeam`, `productionSeam.readChangesSource`, `validateChanges`, `resolveRenderTarget`, `writeHuman`, `reportFailure`, `newProposalCreateCommand` and its flag set, and every other proposal leaf's run function.

---

## Interactions

### Invocation-to-output flow

```
runProposalCreate
  ├─ resolveRenderTarget → require --changes → readChangesSource → validateChanges   (no request yet)
  ├─ assemble → newClient → NewRetryExecutor
  ├─ POST /proposals ─── failure ──► reportFailure (unchanged; NO read-back)
  │        │
  │        └─ success
  │             ├─ machine: raw retained; unmarshal raw → Document[Proposal] → id
  │             └─ human:   already decoded → Document[Proposal] → id
  ├─ readBackProposalVerdict(ctx, exec, id)   ← empty id short-circuits: no request, reason set
  ├─ render.NewProposalVerdict(valid, alerts, reason, id)
  ├─ machine: write read-back raw (or the create's raw when reason != "")
  │   human:  writeHuman(..., ResourceProposalCreated, ProposalCreatedView{…})
  └─ stderr advisory → return Success
```

### Contracts the Builder must hold

- **Nil-vs-false is load-bearing at every hop.** The pointer stays a pointer from decode to `NewProposalVerdict`; no intermediate converts it to a bool. The only place the three cases collapse into text is the label mapping.
- **Empty id short-circuits.** `readBackProposalVerdict` issues no request for an empty id and returns the id-undeterminable reason. This is the only path where the helper produces a reason without an exchange.
- **The retained raw is never re-marshalled.** The machine path unmarshals a *copy* of the bytes to read the id and emits bytes the CLI never reshaped — the 018 verbatim contract.
- **`NewProposalVerdict` is pure.** No I/O, no clock, no formatting of anything but its inputs, so the label vocabulary is unit-testable without a server or a template.
- **The helper takes the executor, not the seam.** It composes the same retrying executor the create used, so it inherits the resolved connection and 017's safe-method retry with no second assembly.

---

## Error Communication

| Failure | Where it surfaces | Go-level shape |
|---|---|---|
| Pre-request rejection | `UsageError`, message on stderr | unchanged — returns before assembly |
| Create failure (any class) | `reportFailure` → format-aware envelope | unchanged; the read-back helper is never called |
| Read-back wire/status/decode failure | the verdict's `unavailable` label + stderr advisory | helper returns a **reason string**, never an error; caller returns `Success` |
| Empty/unliftable created id | same as above | helper short-circuits with a reason; no request |
| Render failure of the new templates | `RuntimeError`(1), stdout left empty | existing `*RenderError` path, buffer-then-write; the new key is registered so an unknown-key error means a registry omission, not a runtime input |
| Undecodable create body (machine path) | `APIError`(3) for the create, per the standing decode classification | the id lift reuses the same decode; a decode failure here is the create's body being unreadable |

The single rule this table encodes: **only the create's own exchange can produce a non-zero exit.** Nothing the read-back does, or fails to do, changes the command's outcome.

---

## Consistency Notes

- **Sibling accord**: `interface-cli.md` pins what these symbols produce — the rendered blocks, the four state labels, the stderr wording, and the exit-code table. The label strings are defined there and produced only by `NewProposalVerdict`, so the two files have one source between them.
- **011 ADR-1 grow-not-duplicate** is followed for the model: new response fields join the existing shared `Proposal` rather than forking a create-specific schema. **055 ADR-4 / 056 ADR-2** established that shared model and are extended, not replaced.
- **055 ADR-4** is diverged from at exactly one point — the create's human render key — and the divergence is announced in plan ADR-4 with the shared template still rendering the body. Everything else in 055's Go surface is untouched.
- **058 ADR-2** is the precedent for a dedicated render path for a distinct concern (`proposal-response`, distinct from `proposal`); `proposal-created` follows it, with the refinement that this one *delegates* to the shared template rather than standing alone.
- **Nullable convention deviation**: the codebase models nullable strings as empty strings (`Proposal.TensionID`, `Tension.RoleID`). `Valid *bool` deviates deliberately, and the field comment carries the reason so a later reader does not "normalize" it into a bool. This is the one place in the model where absence needed its own representation.
- **Registry discipline**: a new `Resource` touches two sites (the const block and `builtinResources`) and requires two template files. The exhaustiveness test derives its expectation from `builtinResources` rather than a hard-coded count, so it fails loudly on a missing template without needing its own number updated.
- **`missingkey=error`** is in force, so every field path the new templates reference must exist on the view. The verdict is a value (not a pointer) on the view for exactly this reason: there is no nil case to guard in the template.
