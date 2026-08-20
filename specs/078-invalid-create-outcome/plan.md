# Plan: Invalid-Create Outcome

**Feature**: 078-invalid-create-outcome
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (precedent: 004, 031, 032, 054, 055, 061, 074), DEPRECATION.md, LEARNINGS.md (background), landed code (`internal/cli/exitcode.go`, `errorenvelope.go`, `proposal.go`, `me.go`, `internal/output/error.go`), `plugin/skills/orientation/SKILL.md`

---

## System Architecture

The feature is a **reclassification at one existing seam**. `runProposalCreate` (055, widened by 074) already obtains everything the failure needs at the moment of decision: the created `prp_` id, the read-back's decoded `Proposal` (with `Valid *bool` and `ValidationAlerts`), and the unavailable-reason when the read-back did not answer. One caveat the implementation must handle: the **human** branch binds the decoded proposal, while the **machine** branch currently discards it (`_, readBackRaw, reason := readBackProposalVerdict(…)`) because it emits raw bytes. That branch must bind it (`readBack, readBackRaw, reason := …`) — the verdict cannot be lifted from the create's own POST document, which 074 established is not a verified carrier of the verdict fields, and a second read is forbidden. 078 inserts one check after `readBackProposalVerdict` on **both** render branches: when the read-back answered (`reason == ""`) and the verdict is an explicit `*Valid == false`, the command stops on its success path and routes a new **typed failure value** through the existing shared failure chokepoint. Every other verdict state falls through to today's behaviour untouched.

The parts:

1. **Outcome + registry arms** — a new `Outcome` constant `InvalidCreate` with arms at **three** switch sites, every one of which has a non-fatal default that would otherwise mask an omission: `codeInvalidCreate = 8` in `exitcode.go`'s const block + `ExitCode` switch (default `1`), the `"invalid-create"` token in `errorenvelope.go`'s `kind()` switch (default `"runtime"`), and the `String()` arm in `dispatch.go` (default `Outcome(%d)`, so an omitted arm renders `Outcome(8)` in every test-failure message and nothing fails). The two table-test *files* (`exitcode_test.go`, `errorenvelope_test.go`) carry three length-guarded table functions between them, and every one of their mirror lists gains the new entry explicitly; `String()` has no table test at all, so its arm is pinned by a task criterion.
2. **Typed failure value** — `*invalidCreateError` in `internal/cli`, carrying `{ProposalID string, Alerts []glassfrog.ValidationAlert}`, with a token-free `Error()`. It is the value `runProposalCreate` hands to `reportFailure` when the trigger fires.
3. **Diagnose arm** — `Diagnose` (031's single total normalizer) gains an `errors.As` arm for `*invalidCreateError`, producing `Diagnostic{Category: InvalidCreate, Cause, NextStep}` plus the carried id and alerts (the 061 `Feature` pattern: the shared `Diagnostic` grows fields only this failure family populates).
4. **Envelope extension** — `output.ErrorDetail` grows `ProposalID string \`json:"proposal_id,omitempty"\`` and `ValidationAlerts []ValidationAlert \`json:"validation_alerts,omitempty"\``, with a neutral `output.ValidationAlert{Severity, Path, Message string}` declared in `internal/output` so the package stays model-free (018's home / populated-in-cli comment pattern, exactly as `NextStep` and `Feature` do). `errorEnvelopeFor` maps the Diagnostic's carried id/alerts into these fields.
5. **Human failure rendering** — `renderDiagnostic` (031) renders the cause, then one line per alert (severity, path, the server's message), then the next step — so the human failure carries every alert without stuffing them into the cause string (which would duplicate into the machine envelope's `message`).
6. **Convention-documenting surfaces** — split by package boundary. **Phase 1** owns everything inside `internal/cli` plus the orientation skill row: the new 078 paragraph in `exitcode.go`'s narrative, the orientation table row (a drift guard couples that row to the constant), and the **five present-tense prose claims the new code falsifies** — `exitcode_test.go`'s "eight frozen exit codes" count and its producers-today enumeration, `dispatch.go`'s matching enumeration, `exitcode.go`'s "the full 0–6 convention is published now", and `exitcode.go`'s "the operational categories (codes 3–6) gain their cases with the future API client" (false since 015). None is covered by a test, and all sit in files Phase 1 already edits. The two genuinely *historical* 0–6 sentences are frozen. **Phase 2** owns every convention-stating surface **outside** `internal/cli`. Those outside-`internal/cli` surfaces are enumerated in tasks.md rather than left to a range-string grep — three of them state the range in backticks or not at all, so a grep for "0–7" misses them: `docs/guides/how-to-read-exit-codes.md` (whose table stops at `6` — it is already missing code `7`, so a grep for "0–7" never finds it), `docs/explanation/how-failures-are-reported.md`, the operator-orientation feature line **and its paired step regex** (godog matches by text), `thenMeaningEachCode`'s `code <= 7` loop bound, the self-containment guard's "exit codes 0–7" string, and one in-surface line in `plugin/skills/proposal-drafting/SKILL.md` (the write-path surface, which says nothing about validity today). The interface artifact (`interface-cli.md`) gains its row in the interface phase.

Control flow on the trigger, machine format: create POST → lift id → read-back → verdict invalid → `reportFailure` writes the **error envelope to stdout** (032's convention) and exits 8. No verbatim proposal document is emitted and no `verdict_source` advisory is written — the envelope subsumes both (ADR-4). Human format: nothing on stdout, diagnostic (cause + alert lines + next step) on stderr, exit 8.

---

## Architecture Decisions

### ADR-1: `InvalidCreate` takes previously-unused code 8, added at all three `Outcome` switch sites

**Context**: The spec pins a new, previously-unused code under 004's extension rule (developer-confirmed at define time). The registry today runs 0–7 (`codeStaleWrite = 7`, 054). A new Outcome touches **three** switch sites — `exitcode.go`'s `ExitCode`, `errorenvelope.go`'s `kind()`, and `dispatch.go`'s `String()` — each with a non-failing default that hides an omission. (074's DECISIONS entry named the first pair; `String()` is the third, found during this spec's pre-implementation guard.)

**Options considered**:
1. **New code 8, arms at all three switch sites** — follows 054's precedent exactly (new category → single registry site per map → previously-unused code → never renumber). Costs one new code consumers must learn.
2. **Reuse an existing failure code (3 or 7)** — no new code, but collapses "the server accepted the write and its result is dead" onto "the server refused the exchange", which the spec's non-behavior forbids.

**Decision**: Option 1. `codeInvalidCreate = 8`, `kind() → "invalid-create"`, `String() → "InvalidCreate"`, enum constant `InvalidCreate` appended to the `Outcome` block (never inserted — the enum's order is load-bearing only for readability, but appending keeps diffs clean). The `exitcode.go` narrative comment gains a **new** 078 paragraph after the 054 one; 054's own sentence ("the first code beyond the originally-published 0–6 band") is a true historical claim about 004's published band and is not edited. Both table tests list the new entry explicitly with the comma-ok + length guards already in place.

**Consequences**: Consumers branching on `$?` gain a distinct signal for accepted-but-dead; no existing branch changes meaning. One coupling is load-bearing for sequencing: `internal/build/operatororientation.go` derives the CLI's published code set by regexing the `exitcode.go` const block and `diffSets` it **bidirectionally** against the backticked digits in `plugin/skills/orientation/SKILL.md`, so the constant and that skill row are one atomic change (T001) rather than code-now/docs-later. The remaining convention-stating surfaces are swept in Phase 2, and they are enumerated individually because at least one of them never states the range as "0–7" at all.

### ADR-2: The failure travels 031's single Diagnose chain via a typed error — no bespoke render path

**Context**: This failure is not an exchange error — the POST and the GET both succeeded. But 031's DECISIONS entry is explicit: failure diagnostics are one cli-side chain and "future error-surface work extends this one chain". 074 ADR-2 rejected `reportFailure` for the read-back *because the envelope carried no `prp_` id* — a gap 078 closes rather than routes around.

**Options considered**:
1. **Typed `*invalidCreateError` + a `Diagnose` arm** — the failure enters the same `reportFailure` → `Diagnose` → `errorEnvelopeFor`/`renderDiagnostic` chain every other failure uses; the id and alerts ride the error value and are extracted by `errors.As`, the same shape as `*ResponseError`'s status/body extraction.
2. **Hand-built `Diagnostic` bypassing `Diagnose`** — skips inventing an error for a non-error, but creates a second classification site, which 031 exists to forbid.
3. **Bespoke failure rendering inside `proposal.go`** — full local control, but fractures 032's single format-aware chokepoint and would re-implement envelope rendering.

**Decision**: Option 1. `runProposalCreate` constructs `&invalidCreateError{ProposalID: id, Alerts: readBack.ValidationAlerts}` and returns `reportFailure(stdout, stderr, rt.format, err)` — the identical call shape both branches already use for exchange failures. `refineClientError` passes it through unchanged (no wrapped `*ResponseError`), and the new `Diagnose` arm classifies it.

**Consequences**: The invalid-create failure renders, formats, and exits exactly like every other failure with zero new rendering machinery. The one oddity — an `error` value for a completed exchange — is confined to the `internal/cli` seam and documented on the type.

### ADR-3: The envelope carries the id and alerts as uniform `omitempty` extensions of `ErrorDetail`, with a neutral alert struct in `internal/output`

**Context**: The spec's clarification (session 2026-08-18) pins envelope-only: the machine failure document carries the `prp_` id and alerts as its own fields and does **not** embed the server proposal document. `internal/output` must stay model-free and transport-free (018's invariant); `glassfrog.ValidationAlert` lives in the model package.

**Options considered**:
1. **Extend `ErrorDetail` with `proposal_id` + `validation_alerts` (`omitempty`), neutral `output.ValidationAlert` mirror struct** — the 061 `Feature` precedent applied twice: declared in 018's home, populated in `errorEnvelopeFor`, absent from every failure that doesn't carry them.
2. **Carry the alerts as `json.RawMessage`** — no mirror struct, but the envelope's fields are typed by convention (`Body` is the one raw slot, reserved for an upstream response body — which this is not: the exchange succeeded).
3. **Embed the server proposal document** — rejected by the spec's clarification: breaks the uniform failure shape.

**Decision**: Option 1. `errorEnvelopeFor` maps the Diagnostic's carried `[]glassfrog.ValidationAlert` into `[]output.ValidationAlert` field-by-field (three strings; `internal/cli` is the one package importing both, exactly the seam comment on `errorEnvelopeFor` describes). The mirror's three fields carry `omitempty` so a key the server omitted stays omitted here too — the success path emits the server document verbatim, and a reconstructed entry must not turn an absent `path` into `""`.

**Consequences**: Every existing failure's envelope is byte-identical (both fields `omitempty`); an agent parses invalid-create like any other failure and finds two extra keys. The mirror struct is a three-string copy — trivial, and the cost of keeping 018's home model-free. One consequence of `omitempty` to carry into the interface: an invalid draft the server attached **no** alerts to still fails, and its envelope omits `validation_alerts` entirely — a zero-length slice is not rendered as `[]`, so no contract may promise an empty array. `ErrorDetail`'s doc comment enumerates the field order and is extended with the two new fields.

### ADR-4: On the invalid-create failure, machine stdout carries the failure envelope — not the server document — and no `verdict_source` advisory is emitted

**Context**: **Announced divergence (narrowing) from 074 ADR-5**, which had the machine path emit the read-back's document verbatim with the advisory on stderr. That decision governed a command that always reported `Success`; 078 carves the `valid: false` state out of it. The spec's clarification forbids embedding the proposal document in the failure, and 032's convention already puts the envelope on stdout for machine-format failures.

**Options considered**:
1. **Envelope on stdout (032 convention), advisory suppressed** — the failure envelope's `kind: "invalid-create"` + `proposal_id` + `validation_alerts` answer everything the advisory answered (the CLI asked; here is the record it read); a success-shaped advisory next to a failure envelope would be two half-signals.
2. **Envelope on stdout + advisory on stderr** — keeps the advisory unconditional, but emits a success-path artifact on a failure path, and the advisory's `read_back: true` alongside `kind: invalid-create` is redundant by construction.

**Decision**: Option 1. The advisory (and the human prose line) remain exactly as landed for the **three** verdict states that stay successes (valid, not reported, unavailable); the fourth — explicit `valid: false` — renders only the failure. 074 ADR-5's rule is narrowed to "the machine path emits the later document verbatim *on the success outcomes*"; nothing else in it changes.

**Consequences**: Distinguishability holds structurally: the three success states are told apart by the emitted document + advisory as today (`data.valid` plus the advisory's `read_back`); the failure is told apart by the envelope and exit 8. Counting note for downstream artifacts: a `valid: true` draft carrying alerts is the *valid* state, not a state of its own — the success states are three, not four. Candidate for `/score:deprecate` against 074 ADR-2's "reports `Success` for all four verdict states" — the not-valid state now fails by design, which 074's entry itself forecast.

### ADR-5: The diagnostic's cause and remedy — resolving the spec's `[ASSUMED]` dead-draft remedy

**Context**: The spec *had* carried an `[ASSUMED]` marker that the remedy depends on whether a discard path exists; this ADR is what settled it, and the spec's Assumptions entry now records the settled answer and points here. Established facts: the CLI exposes no draft-delete (059's `withdraw` moves a *circulating* proposal back to draft — it does not delete a draft); CSG-2's record shows a dead draft is deletable only in the web UI; the grammar reference (`glassfrog proposal grammar`, 077) documents known accepted-but-invalid shapes.

**Options considered**:
1. **Remedy = revise-and-recreate, naming the web UI for cleanup and the grammar reference for known shapes** — honest about what the CLI can and cannot do; actionable without any new capability.
2. **Remedy = point at a future discard command** — names a command the caller cannot run, which the landed advisory design explicitly refuses to do.

**Decision**: Option 1. Cause names the verdict and its provenance: the server reported the created draft not valid (read back after the create). NextStep directs: review the alerts, consult `glassfrog proposal grammar` for documented invalid shapes, create a corrected proposal from the same tension; the invalid draft remains and can be deleted in the GlassFrog web UI. Exact wording is the interface phase's to pin; the elements are decided here.

**Consequences**: The assumption is settled without new capability, and the spec's marker was retired when this ADR landed. If a draft-delete command ever lands, the NextStep text is a one-site edit (the `Diagnose` arm).

---

## Cross-cutting Concerns

**Error handling**: the trigger condition is precisely `reason == "" && readBack.Valid != nil && !*readBack.Valid` — the `*bool` nil-vs-false distinction (074 ADR-3) is the load-bearing check; nil (no verdict stated) stays `Success`. The check sits after `readBackProposalVerdict` on both render branches, **before any stdout write** (buffer-then-write holds — on the machine branch the check replaces the document emission, so a failed create never half-emits). The read-back's own failures keep their landed never-the-command's-failure contract untouched.

**Testing**: the two table tests gain the new entry (the comma-ok + `len` guards there already defend the zero-value/omission traps), and `String()`'s arm — which no table test covers — is pinned by an explicit criterion asserting `"InvalidCreate"` rather than `Outcome(8)`; `Diagnose` gets an arm test (cause, next step, category, carried id/alerts); `errorEnvelopeFor` gets an invalid-create case (fields present, `omitempty` absence on other failures pinned); `runProposalCreate` BDD covers the trigger and every non-trigger verdict state on both branches (fake seam, offline, as landed). The exhaustiveness caveat in `errorenvelope.go`'s comment — the table test cannot see a constant absent from its list — is answered by adding the entry in the same commit as the constant.

**Observability**: no new channels. stdout/stderr split follows 032; nothing writes the token.

**Configuration**: nothing configurable — the spec forbids an opt-out of the failure just as 074 forbade skipping the read-back.

---

## Implementation Strategy

**Phase 1 — the outcome, end to end** (three sequential tasks): `Outcome` constant + all three switch arms + the added comment paragraph + the five falsified in-package prose claims + the coupled orientation table row; `*invalidCreateError` + `Diagnose` arm; `ErrorDetail`/`ValidationAlert` extension + `errorEnvelopeFor` mapping; `renderDiagnostic` alert lines; the two call-site checks in `runProposalCreate` (binding the decoded proposal on the machine branch); the scenario retirement including the runner's `~@deprecated` filter and the six superseded Go unit tests; all tests above. Every task ends with a green suite — the orientation row rides with the constant, and the runner filter rides with the behavior flip. The phase leaves every other command byte-identical.

**Phase 2 — the remaining convention-documenting surfaces** (depends on Phase 1's final wording; the orientation skill already moved in Phase 1). The property to satisfy: *a surface that enumerates the convention lists every published code, and any stated range ends at the highest published code.* Sites are enumerated in tasks.md T004; two of them live in `internal/build` guard code that would keep passing while asserting less. `docs/guides/how-to-read-exit-codes.md` needs code `7` as well as `8` — it never got 054's code. Plugin edits must satisfy the 076 self-containment guard — in-surface names only, no spec numbers.

---

## Risks

- **A masked switch omission** (low likelihood, high impact): all three switches default non-fatally (`1`, `"runtime"`, `Outcome(%d)`), so a missed arm ships a wrong-but-plausible signal rather than failing. Mitigation: the two table tests gain the entry in the same commit, and `String()`'s arm — which has no table test — is pinned by an explicit T001 criterion.
- **Stale convention prose, and a sweep that looks complete but is not** (medium likelihood, medium impact): a range-string grep finds the orientation skill but misses `docs/guides/how-to-read-exit-codes.md` (whose table simply stops at `6`) and `thenMeaningEachCode`'s numeric loop bound — so "no surface says 0–7" can pass while two surfaces still under-state the convention. Mitigation: T004 enumerates every site and states the property rather than the string; the paired feature-line/step-regex edit is called out because godog matches by text.
- **074 scenario collision — confirmed, not hypothetical** (high likelihood if unhandled, high impact): **three** active scenarios in `post-create-validity-read.feature` assert `exit with code 0` against a not-valid read-back (lines 35, 124, 146), and a fourth is held out (`@validation @wip`). They were true of the landed intermediate state and become false the moment this feature lands. The `@deprecate` tag alone does not help: nothing in the repo reads it — every runner filters `~@wip` only. **The collision has a second half in Go**, which no tag filter can reach: `internal/cli/proposal_readback_test.go`'s shared fixture `proposalReadBackBody` *is* the not-valid body, and six tests assert `Success` against it while pinning success-shaped stdout and the stderr advisory. Mitigation (T003, all in the same commit as the behavior flip): retag the three scenarios `@deprecated` **and** add `~@deprecated` to the one runner that loads that file, giving the tag the exclusion its documented lifecycle already promises; **and** re-point or invert those six Go tests, changing the two table-driven tests' shared loop body rather than only adding a row to their case tables. Also flagged for `/score:deprecate` against 074 ADR-2's four-states clause.

---

## What This Plan Does Not Cover

- **Protocol-level contracts** — the exact envelope field spellings in the published interface table, the exit-code row's published text, and the diagnostic's final wording: the interface phase (`interface-cli.md`).
- **Executable scenarios** — the scenarios phase turns the spec's driving scenarios into the feature file, including the 074-collision amendment named in Risks.
- **Task decomposition** — the tasks phase; the two phases above are its seed.
- **A draft-discard command** — deliberately out; ADR-5 designs the remedy around its absence.
