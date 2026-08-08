# Tasks: Post-Create Validity Read

**Feature**: 074-post-create-validity-read
**Concretization**: Full context
**Inputs**: `plan.md`, `spec.md`, `interface-cli.md`, `interface-spec.md`, `features/success-reported-for-a-dead-proposal/post-create-validity-read.feature`, `PROJECT.md`

---

## Dependency Graph

```
Phase 1: Verdict model (1 task, no dependencies) [Shared]
Phase 2: Create-specific render path (2 tasks, depends on Phase 1) [Shared]
Phase 3: Read-back orchestration (4 tasks, depends on Phases 1, 2) [US1, US2, US3]

7 tasks total | 0 phases parallelizable | Builder: pipeline mode, single builder
```

The phases are strictly sequential — Phase 2's view carries the fields Phase 1 adds, and Phase 3 renders through Phase 2's key. Within Phase 3, T005 and T006 are marked `[P]`: they touch the two arms of the same dispatch and can be written independently once T004 exists, though they land in the same file.

**Story labels** (from spec.md User Scenarios, in order):
- **US1** — the practitioner sees the server's verdict in the create result they approved
- **US2** — the agent reads validity, alerts, and transitions out of the create output it already parses
- **US3** — the agent still receives the `prp_` id when the verdict cannot be obtained

---

## Branching Guidance

**Pipeline mode**: `spec/074-post-create-validity-read/base` → `spec/074-post-create-validity-read/task-1` … `task-7`

Phases 1 and 2 change no user-visible behaviour (nothing reads the new fields until T005/T006), so they merge to base independently and keep the behavioural review concentrated in Phase 3.

---

## Phase 1: Verdict model [Shared]

- [ ] **T001** [Shared] Add the verdict fields and the `ValidationAlert` type to the shared proposal model
  - **Scope**: `internal/glassfrog/proposal.go` — add `Valid *bool` and `ValidationAlerts []ValidationAlert` to the existing `Proposal` struct, and the new `ValidationAlert` type (`Severity`, `Path`, `Message`, each snake_case-tagged). Comments carry why the pointer deviates from the codebase's nullable-as-empty-string convention and that both fields are undeclared in the vendored contract. No method that summarises the verdict is added.
  - **Acceptance criteria**:
    - `valid: true`, `valid: false`, and an absent `valid` decode to a non-nil true, a non-nil false, and nil respectively — asserted as three distinct cases
    - An explicit `"valid": null` decodes to nil (indistinguishable from absent, and that is the intent)
    - `"validation_alerts": []` decodes to a non-nil empty slice while an absent key decodes to nil, and both are asserted
    - A populated alert decodes all three keys, and an alert object carrying an extra unknown key still decodes without error
    - Every existing `Proposal` decode test passes unchanged, and no existing consumer reads the new fields
    - `gofmt -l .` is clean — adding fields re-aligns the sibling struct tags, and that re-alignment belongs in this commit
  - **Dependencies**: None
  - **Plan reference**: Phase 1: Verdict model; ADR-3
  - **Interface references**: `interface-spec.md`: "`internal/glassfrog` — two fields on the existing `Proposal`, one new type"

---

## Phase 2: Create-specific render path [Shared]

- [ ] **T002** [Shared] Add the created-proposal view, the verdict projection, and the label mapping
  - **Scope**: `internal/render/render.go` — add `ProposalCreatedView` (embedding the existing `ProposalView`, plus a `Verdict` field), `ProposalVerdict` (`Validity`, `Alerts`, `Source`), and the pure `NewProposalVerdict(valid *bool, alerts, unavailableReason, id)` mapping. This is the single source of the four state labels; no other package composes them.
  - **Acceptance criteria**:
    - `NewProposalVerdict` maps a non-nil true to `valid`, a non-nil false to `not valid`, nil to `not reported by the server`, and a non-empty reason to `unavailable — <reason>`
    - The same call produces the **compact** label for the same four states — `valid`, `not valid`, `validity not reported`, `validity unavailable` — so both vocabularies have one source and cannot drift
    - The compact label appends ` (N alert(s))` whenever the server stated at least one alert, in **either** validity state; the full label never carries an alert count
    - A non-empty reason wins over any flag value, and carries no alerts — nothing is claimed that the server did not state
    - `Source` names the read-back and the proposal id when a verdict was obtained, and states that none was obtained otherwise
    - The function is pure: no I/O, no clock, no package-level state, and it is tested without a template or a server
    - `ProposalCreatedView` field promotion is asserted — an existing `ProposalView` field path still resolves through the embedded struct
  - **Dependencies**: T001
  - **Plan reference**: Phase 2: Create-specific render path; ADR-3, ADR-4
  - **Interface references**: `interface-spec.md`: "`internal/render` — one resource key, one view, one verdict projection, two templates"

- [ ] **T003** [Shared] Register the `proposal-created` resource and add the two delegating templates
  - **Scope**: `internal/render/render.go` + `internal/render/templates/` — add the `ResourceProposalCreated` const with its doc comment, append it to `builtinResources`, and add `proposal-created.full.tmpl` and `proposal-created.compact.tmpl`. Both render the body by invoking the existing `proposal.<format>.tmpl` from the single parsed set; neither restates a body line. The shared `proposal` templates are not modified.
  - **Acceptance criteria**:
    - The full template renders the shared body followed by the `Validity` line, an `Alerts (N):` block only when alerts are present, and the `Verdict source:` line — labels aligned to the existing 16-column field
    - Each alert line renders its severity, path, and the server's message verbatim
    - The compact template renders the shared compact line plus `.Verdict.Compact` — not the full block's `Validity`, which would put the server's reason text on a one-liner; transitions stay absent from compact, as today
    - All four verdict states render distinctly in both formats
    - A **valid** verdict carrying an alert renders both facts — the validity as `valid` and the alert with its severity, path, and message — so alert presence never reads as an unfavourable verdict
    - `proposal`-keyed output for the same proposal is asserted byte-identical to its current rendering — this is the guard against verdict lines leaking into `proposal get`, `propose`, and `withdraw`
    - The registry-exhaustiveness test passes with the new key, deriving its expectation from `builtinResources` rather than a hard-coded count
  - **Dependencies**: T002
  - **Plan reference**: Phase 2: Create-specific render path; ADR-4
  - **Scenario references**: `post-create-validity-read.feature`: "Scenario: The sibling proposal commands render no verdict"
  - **Interface references**: `interface-cli.md`: "stdout — human `full` format", "stdout — human `compact` format"
  - **Risk**: ⚠️ The shared template is the drift surface ADR-4 routes around — the byte-identical assertion is the criterion that must not be dropped or weakened

---

## Phase 3: Read-back orchestration [US1, US2, US3]

- [ ] **T004** [US3] Add the isolated read-back helper
  - **Scope**: `internal/cli/proposal.go` — add `readBackProposalVerdict(ctx, exec, id) (glassfrog.Proposal, json.RawMessage, string)`: one `GET /proposals/` + `url.PathEscape(id)` through the supplied executor, returning the decoded proposal, the raw bytes, and a reason string. It returns no error: every failure becomes a reason. An empty id short-circuits with the id-undeterminable reason and issues no request.
  - **Acceptance criteria**:
    - A successful read-back returns an empty reason, the decoded proposal, and the raw bytes
    - A wire failure, a non-2xx, a post-retry 429, and an undecodable body each return a distinct reason and no error
    - The 429 reason names the exhausted request budget; the reasons are the ones `interface-cli.md` enumerates
    - An empty id issues zero requests and returns the id-undeterminable reason
    - The helper never calls `reportFailure` and never returns a non-Success outcome — asserted by its signature and by a test that a failing read-back leaves the caller's outcome untouched
    - The path is built with `url.PathEscape`, matching `proposal get`
  - **Dependencies**: T001
  - **Plan reference**: Phase 3: Read-back orchestration; ADR-2
  - **Scenario references**: `post-create-validity-read.feature`: "Scenario: An unreachable read-back still reports the created id", "Scenario: A rate-limited read-back reports the exhausted budget with the created id"
  - **Interface references**: `interface-spec.md`: "`internal/cli` — one helper, one changed call site"

- [ ] **T005** [US1] [P] Wire the read-back into the create's human render path
  - **Scope**: `internal/cli/proposal.go` — in `runProposalCreate`'s human arm, after a successful create: take the id from the already-decoded `Document[Proposal]`, call the helper, build the verdict through `render.NewProposalVerdict`, render through `render.ResourceProposalCreated` over a `ProposalCreatedView`, and write the stderr advisory. The outcome stays `Success` for every verdict state.
  - **Acceptance criteria**:
    - A valid, an invalid, a no-verdict, and an unavailable read-back each render their state and exit 0
    - An invalid created draft prints the id, the `not valid` verdict, the alert with severity and path, and that no transitions are available
    - The stderr advisory names the read on success and, when unavailable, states the cause and the remedy (`glassfrog proposal get <prp_id>`) — except where no id could be determined, which names no remedy
    - In a human format the advisory is the prose line; the prose and the structured form are derived from one value, so they cannot disagree about whether the CLI asked
    - The pre-request rejection paths are unchanged: a missing/blank/unparseable/type-less `--changes` and a bad `--output` still issue zero requests
    - A user-supplied template written against the pre-change view still renders — every field path that resolved before still resolves, and its output is unchanged by the verdict's addition
    - The `compact` selection carries the id, status, change count, the compact validity label, and the alert count on one line
    - A render failure still maps to the internal-error code with stdout left empty
  - **Dependencies**: T003, T004
  - **Plan reference**: Phase 3: Read-back orchestration; ADR-2, ADR-4
  - **Scenario references**: `post-create-validity-read.feature`: "Scenario: A valid created draft reports its verdict with its id", "Scenario: A created-but-invalid draft surfaces the server's refusal", "Scenario: A draft the server states no verdict on is reported as unreported", "Scenario: A valid draft with no transitions keeps the two facts distinct", "Scenario: A conflicted status and a favourable verdict are both reported as given", "Scenario: A valid draft carrying an advisory alert reports both facts", "Scenario: The compact line carries the validity token", "Scenario: A user template written before the verdict still renders"
  - **Interface references**: `interface-cli.md`: "The four verdict states", "stderr"

- [ ] **T006** [US2] [P] Wire the read-back into the create's machine render path
  - **Scope**: `internal/cli/proposal.go` — in `runProposalCreate`'s machine arm: keep the create's raw bytes untouched, unmarshal a copy into `Document[Proposal]` to lift the id, call the helper, and emit the **read-back's** raw `{data}` verbatim — falling back to the create's raw when no read-back produced a body. Write the stderr advisory. No composed envelope and no CLI-added keys.
  - **Acceptance criteria**:
    - On a successful read-back the emitted document is the read-back's, carrying `valid`, `validation_alerts`, and `available_transitions` as the server sent them
    - On a failed read-back the emitted document is the create's, unchanged from today, and the structured advisory states the verdict was unobtainable and why
    - The stderr advisory is rendered in the **selected machine format** (not prose), carrying `read_back` and, when applicable, `proposal_id`, `reason`, and `remedy`; absent keys mean "not applicable" and no key is ever an empty string
    - The four verdict states are distinguishable from `data.valid` plus `verdict_source.read_back` alone, with no prose parsing — asserted as a four-case table
    - A success body carrying no `prp_` id emits the create's document, issues no read-back, and reports the id-undeterminable reason
    - The emitted bytes are never re-marshalled — asserted by comparing against the fixture's exact bytes
    - Both `json` and `yaml` selections are covered
    - An undecodable create body still classifies as an API error for the create, unchanged
  - **Dependencies**: T003, T004
  - **Plan reference**: Phase 3: Read-back orchestration; ADR-5, ADR-6
  - **Scenario references**: `post-create-validity-read.feature`: "Scenario: Structured output carries the verdict alongside the created id", "Scenario: A create response carrying no id yields no read-back"
  - **Interface references**: `interface-cli.md`: "stdout — machine formats"

- [ ] **T007** [Shared] Reconcile the existing create tests and wire the BDD suite
  - **Scope**: `internal/cli/proposal_test.go` and a new `internal/cli/post_create_validity_read_bdd_test.go` — update every existing create test whose exchange count changed (the success path now performs two exchanges), and add the godog suite over the new feature file with its step definitions, clearing `@wip` per scenario as each passes.
  - **Acceptance criteria**:
    - Exchange counts are asserted explicitly, not incidentally: zero requests on any pre-request rejection, exactly one on a failed create, exactly two on a successful create whose read-back is attempted
    - The rejected-create path asserts that **no** read of any proposal was attempted
    - Step definitions read parameterised values from world state rather than re-deriving them, and apply nil guards symmetrically
    - `@wip` is removed only from scenarios the suite actually executes; the disposition table below is the record of what is executed and what is held
    - `go test ./...` and `gofmt -l .` are both clean
  - **Dependencies**: T005, T006
  - **Plan reference**: Phase 3: Read-back orchestration; Cross-cutting Concerns → Testing strategy
  - **Risk**: ⚠️ Existing create tests that assert a single exchange will fail until updated — that failure is the intended signal, not a regression

### Scenario disposition — `post-create-validity-read.feature`

Nineteen scenarios. Fifteen are executed by T007's suite; four are held. No scenario is inexecutable — the held set is held for one reason only.

| Scenario | Disposition |
|---|---|
| A valid created draft reports its verdict with its id | Executed (T007, via T005) |
| A created-but-invalid draft surfaces the server's refusal | Executed (T007, via T005) |
| A rejected create is reported without a read-back | Executed (T007, via T005) |
| A draft the server states no verdict on is reported as unreported | Executed (T007, via T005) |
| A valid draft with no transitions keeps the two facts distinct | Executed (T007, via T005) |
| A conflicted status and a favourable verdict are both reported as given | Executed (T007, via T005) |
| A valid draft carrying an advisory alert reports both facts | Executed (T007, via T005) |
| Structured output carries the verdict alongside the created id | Executed (T007, via T006) |
| An unobtainable verdict is machine-readable in a machine format | Executed (T007, via T006) |
| The compact line carries the validity token | Executed (T007, via T005) |
| A user template written before the verdict still renders | Executed (T007, via T005) |
| The sibling proposal commands render no verdict | Executed (T007, via T003) |
| An unreachable read-back still reports the created id | Executed (T007, via T004/T005) |
| A rate-limited read-back reports the exhausted budget with the created id | Executed (T007, via T004/T005) |
| A create response carrying no id yields no read-back | Executed (T007, via T006) |
| The reported result names the read that produced the verdict | **Held** — `@validation`, for `/score:validate` |
| No verdict is derived from the change set, status, transitions, or alerts | **Held** — `@validation`, for `/score:validate` |
| The verdict fields are never presented as published contract | **Held** — `@validation`, for `/score:validate` |
| Every verdict state is distinguishable in every output format | **Held** — `@validation`, for `/score:validate` |

The four held scenarios keep their `@validation @wip` tags through implementation; `/score:validate` is where they are exercised, and T007 must not clear their tags.
