# Tasks: Legacy Identifier Request

**Feature**: 075-legacy-identifier-request
**Concretization**: Full context (plan + spec + interface-cli.md + interface-spec.md + three feature files)
**Inputs**: plan.md, spec.md, interface-cli.md, interface-spec.md, features/change-targets-unidentifiable/{legacy-identifier-request,legacy-identifier-absence,legacy-identifier-guard}.feature

---

## Dependency Graph

Phase 1: Opt-in plumbing and structured fidelity (2 tasks, no dependencies) [Shared]
Phase 2: Human render (2 tasks, depends on Phase 1) [Shared]
Phase 3: Retirement tripwire (1 task, depends on Phase 1 — parallel with Phase 2) [US4]

5 tasks total | 2 phases parallelizable (2 and 3, once 1 lands) | Builder: implement skill or manual

**Story coverage** — the spec has four user scenarios, and the `[Shared]` labels below are the template's multi-story rule rather than an unassigned default:
- **US1** (practitioner asks a role read) — T002 sends the request; T003 renders the role family.
- **US2** (agent obtains the number from a read it is already making) — T002 (tree and list requests), T003 (tree render), T004 (the operator-template surface).
- **US3** (agent distinguishes same-named roles) — T002 (walked role list), T003 (compact role rendering, where two same-named roles are compared side by side), T004 (actor directory).
- **US4** (maintainer keeps the time-limited nature visible) — T002 owns the help-text constant; T005 anchors it mechanically.

T001 is `[Shared]` because the model layer serves all four. No user scenario is unserved.

---

## Branching Guidance

**Pipeline mode**: `spec/075-legacy-identifier-request/base` → `spec/075-legacy-identifier-request/task-N`. At this feature's size a single branch carrying all three phases is acceptable (plan: phases exist so tasks slice cleanly, not because separate PRs are required).

**Role-based mode**: same structure; 073-circle-routing-rule has a complete artifact set awaiting implementation — coordinate on `internal/build` if both land guard files in the same window (different files, no shared state).

> **Lint gate applies to every task below.** There is **no pre-commit hook in this repo** — verified: no `.pre-commit-config.yaml`, no husky/lefthook, and `.git/hooks/` holds only samples. Linting exists solely as CI's `lint:` job (gofmt + `go vet` + golangci-lint + shellcheck). So `go test ./...` passing locally does **not** mean CI is green, and the gate is stated per task rather than assumed. This is the shape that cost PR #164 a triage round: adding struct fields silently re-aligned neighbouring gofmt columns, tests were green locally, CI's lint leg went red. T001, T003, and T004 all add or reorder struct fields, so they are the live cases.

---

## Scenario Disposition

28 scenarios across the three feature files. **23 executed**, each owned by exactly one task; **5 held**, all for the same reason.

| Scenario | File | Disposition |
|---|---|---|
| A single role read carries the legacy numeric identifier | request | T003 |
| The identity read's structured output carries every number the response carried | request | T002 |
| The identity read's human render shows the actor and organization numbers | request | T004 |
| A failing read is unaffected by the request | request | T002 |
| An unsupported read refuses the flag before any request | request | T002 |
| Without the flag the read is byte-identical to before | request | T002 |
| One tree read yields legacy numbers for a whole subtree | request | T003 |
| An operator template sees the membership number the built-in render omits | request | T004 |
| Every role in a walked list carries its legacy number | request | T002 |
| The actor directory carries a legacy number for every actor | request | T004 |
| Compact output of the role family carries the number as a segment *(outline, 2 examples)* | request | T003 |
| No outbound request addresses a resource by its legacy number | request | **held** — `@validation` |
| An agent-backed actor renders its absence with the agent backing as reason | absence | T004 |
| An agent-backed actor's structured output carries a bare null | absence | T002 |
| A read whose embeds omit the number states it once per group | absence | T003 |
| A read whose embeds carry the number renders it and states nothing | absence | T004 |
| A retired bridge leaves the structured output without the key | absence | T002 |
| A retired bridge leaves the human render showing explicit absence | absence | T003 |
| An integer-bearing legacy number is accepted in either spelling *(outline, 2 examples)* | absence | T003 |
| A non-numeric legacy number fails loudly rather than reading as absent | absence | T003 |
| Structured output distinguishes not-requested from requested-and-absent | absence | **held** — `@validation` |
| Structured output carries the legacy_id key exactly where the response did | absence | **held** — `@validation` |
| The flag exists on exactly the supported reads and nowhere else | guard | **held** — `@validation` |
| The retirement clock is stated in the flag's own help text | guard | **held** — `@validation` |
| The guard derives both sides and fails on a vendored-contract change | guard | T005 |
| The guard anchors the shared help constant's retirement property | guard | T005 |
| A newly mapped operation whose schema omits the field fails until it is probed | guard | T005 |
| A registered exception for an unmapped operation fails as stale | guard | T005 |

**Hold set**: all 5 are `@validation`, held for `/score:validate` — a single reason, not a mixed set. No scenario is inexecutable for process reasons.

**Phase locality**: every executed scenario is greenable within its owning task's phase. Structured-output assertions sit in Phase 1 (T002) because ADR-2 means structured fidelity lands with the query parameter; human-render assertions sit in Phase 2 (T003/T004). The two scenarios that originally straddled this boundary were split, one half per phase.

T001 carries no scenario of its own — it is a model-layer change verified by unit tests, and its behavior is exercised through T003's scenarios. This is stated so its absence from the table reads as deliberate.

---

## Phase 1: Opt-in plumbing and structured fidelity [Shared]

- [ ] **T001** [Shared] [P] Grow the five typed models with the tolerant nullable legacy-identifier field
  - **Scope**: `internal/glassfrog` only — add `LegacyID *int64` with `json:"legacy_id"` to `Role`, `Actor`, `Organization`, `Membership`, `TreeNode`, with a custom unmarshaller accepting **either a JSON integer or a JSON string** (ADR-3 decode tolerance). Decode-only; no encode path, no render change.
  - **Acceptance criteria**:
    - All five structs decode `legacy_id` as integer, **integer-bearing** quoted string (`"14062695"`), explicit null, and absent key to the correct pointer state (value / value / nil / nil)
    - A **non-numeric** string (`"abc"`, `""`) fails the decode — the tolerance accepts a string *spelling of an integer*, not any string. A value that is neither integer nor string likewise fails. A genuine contract break stays loud rather than silently yielding nil
    - The failing cases are covered by tests, not just the accepting ones — a tolerant decoder whose rejection path is untested is where a silent nil enters
    - Decode tests cover all four states on at least `Role` and `Actor`
    - The tolerance is scoped to this field and softens no other decoding
    - `TreeNode` carries a comment recording that the field rests on observed evidence, not the vendored schema (LEARNINGS W1), so a future reader does not remove it as unsupported
    - `gofmt -l .` clean and `go test ./...` green before push — this task adds a field to five structs, the exact edit shape that re-aligns sibling gofmt columns
  - **Dependencies**: None
  - **Plan reference**: Phase 1 — opt-in plumbing and structured fidelity, ADR-3
  - **Interface references**: interface-cli.md: The field (incl. Decode tolerance and the `TreeNode` note)

- [ ] **T002** [Shared] [P] Register `--legacy-id` on the four leaves and thread `include_legacy_id=true` through the six read paths
  - **Scope**: `internal/cli` — one shared help constant carrying the retirement caveat; command-local `Flags()` registration on `roles`, `tree`, `actors`, `me`; when set, append `include_legacy_id=true` to the request `url.Values` on both branches of `roles`/`actors`, `tree`, and `me` (walked lists inherit per-page carry from `paging.All`'s query clone — no paging change). BDD step definitions for the scenarios owned here.
  - **Acceptance criteria**:
    - Flag set → outbound query carries `include_legacy_id=true` on all six read paths; every page of a walked list carries it
    - Flag unset → outbound request byte-identical to today (no parameter at all)
    - Registration uses `Flags()`, not `PersistentFlags()` — `me roles`, `me actions`, and `me projects` must keep rejecting the flag
    - `--legacy-id` on any other command fails pre-request with cobra's unknown-flag usage error, exit 2, no request sent
    - All four registrations use the single shared help constant (conveys transition + retirement)
    - Structured output passes through unchanged raw bytes: key where the API sent it, null preserved, no key on embeds that omit it, membership number present on `me`
    - Failure paths unchanged with the flag set
    - `gofmt -l .` clean and `go test ./...` green before push
    - `@wip` removed from the seven scenarios owned here as they pass
  - **Dependencies**: None (parallel with T001 — the structured path decodes raw bytes, not the typed models)
  - **Plan reference**: Phase 1 — opt-in plumbing and structured fidelity, ADR-1, ADR-2
  - **Scenario references**: request.feature: "The identity read's structured output carries every number the response carried", "A failing read is unaffected by the request", "An unsupported read refuses the flag before any request", "Without the flag the read is byte-identical to before", "Every role in a walked list carries its legacy number"; absence.feature: "An agent-backed actor's structured output carries a bare null", "A retired bridge leaves the structured output without the key"
  - **Interface references**: interface-cli.md: The flag, The request, Structured output

## Phase 2: Human render [Shared]

- [ ] **T003** [Shared] [P] Surface the number in the role-family human renders (`roles`, `role`, `tree`)
  - **Scope**: requested-bit threading from the three commands' flag state into their render data (the `Requested`-map pattern), plus `roles.{full,compact}`, `role.{full,compact}`, `tree.{full,compact}` template changes: `legacy_id=<n|—>` compact segments, `Legacy id:` full lines with `(none)` absence, embed-note suffix on `Fillers`/`Assignments`/`Subroles` (role.full) and `Members` (tree.full) headings. View-level deref (TreeRow precedent) so no pointer artifacts render.
  - **Acceptance criteria**:
    - Requested + present renders the number beside the stable id per interface-cli.md's per-template table
    - Requested + absent renders `legacy_id=—` (compact) / `(none)` (full)
    - Tree rows carry the number at **every depth**, not only the root
    - Embedded groups on `role.full` carry the note once per group, never per member; description-only groups (Domains, Accountabilities, Policies, Notes, Skills) and the single-line `Parent role` stay untouched
    - The note's wording states what **this read** carries, never what exists in the system of record
    - Not requested → all six templates render byte-identical to before (regression-pinned)
    - A non-numeric `legacy_id` surfaces as a decode failure, never as the explicit-absence marker — absence and breakage must not look alike
    - `gofmt -l .` clean and `go test ./...` green before push — this task adds fields to render data structs
    - `@wip` removed from the seven scenarios owned here as they pass
  - **Dependencies**: T001, T002
  - **Plan reference**: Phase 2 — human render; Render Design
  - **Scenario references**: request.feature: "A single role read carries the legacy numeric identifier", "One tree read yields legacy numbers for a whole subtree", "Compact output of the role family carries the number as a segment"; absence.feature: "A read whose embeds omit the number states it once per group", "A retired bridge leaves the human render showing explicit absence", "An integer-bearing legacy number is accepted in either spelling", "A non-numeric legacy number fails loudly rather than reading as absent"
  - **Interface references**: interface-cli.md: Human output — shared idioms, Human output — per template (roles/role/tree rows)

- [ ] **T004** [Shared] [P] Surface the number in the actor/me human renders (`actors`, `actor`, `me`)
  - **Scope**: requested-bit threading for `actors`/`me`, plus `actors.{full,compact}`, `actor.{full,compact}`, `me.{full,compact}` template changes: agent-backed absence `(none — agent-backed actor)` on full renders, `[agent]`-adjacent `legacy_id=—` on compact, `me` actor + organization numbers with membership deliberately unrendered, embed-note suffix on `Roles`/`Assignments` (actor.full), and **the number rendered on each embedded role in `me.full` with no embed note** (LEARNINGS W2).
  - **Acceptance criteria**:
    - Agent-backed actor: full render names the agent backing as the absence reason
    - Human `me` shows actor + organization numbers only; `me -o json --legacy-id` carries all three including membership
    - `me --legacy-id --include roles`: each embedded role renders its own number, and **nothing** is stated about embeds lacking it — verified live, this read's embeds do carry the field
    - An operator-supplied template can render the membership number the built-in render omits
    - The actor directory list renders a number per actor row
    - Not-requested renders byte-identical; exit 0 when requested and nothing carried a number, with no diagnostic
    - `gofmt -l .` clean and `go test ./...` green before push — this task adds fields to render data structs
    - `@wip` removed from the five scenarios owned here as they pass
  - **Dependencies**: T001, T002 (parallel with T003 — different templates and call sites)
  - **Plan reference**: Phase 2 — human render; Render Design
  - **Scenario references**: request.feature: "The identity read's human render shows the actor and organization numbers", "An operator template sees the membership number the built-in render omits", "The actor directory carries a legacy number for every actor"; absence.feature: "An agent-backed actor renders its absence with the agent backing as reason", "A read whose embeds carry the number renders it and states nothing"
  - **Interface references**: interface-cli.md: Human output — per template (actors/actor/me rows)

## Phase 3: Retirement tripwire [US4]

- [ ] **T005** [US4] Add the `internal/build` coverage guard with its observed-exception register
  - **Scope**: `internal/build/legacyidcoverage.go` (declared operation→command mapping, observed-exception register, derivation helpers — production source) and `legacyidcoverage_test.go` asserting interface-spec.md's five invariants: S=K set equality against the vendored spec's `IncludeLegacyId` `$ref` sites (structural YAML parsing, no line-grep); every mapped leaf carries a local `legacy-id` flag absent from its children; no other leaf in the tree registers it; all registrations share the one help constant whose text conveys the transition/retirement property (stem-level, not exact prose); and response-schema capability via the register.
  - **Acceptance criteria**:
    - Removing an operation's `IncludeLegacyId` `$ref` from a spec copy fails naming the operation and the paired remedy (drop mapping + registration together)
    - Adding the parameter to a new operation in a spec copy fails naming the coverage decision
    - A leaf registering `legacy-id` outside the mapping, or a mapped leaf missing it, fails naming the leaf
    - Invariant 5: a mapped operation whose schema declares no `legacy_id` and has no registered exception fails; the register's `getRoleTree` entry carries its probe evidence and passes; an exception for an unmapped operation fails as stale
    - The guard **must not** assert bare "every mapped operation's schema declares the field" — that form is permanently unpassable given `TreeNode`, and its only green state is deleting a working read. The reason is written into the guard's comment
    - Guard hard-codes no operation ids or command names outside the declared mapping and register; both sets derived at test time
    - Failure messages name reachable remedies, each checked against sibling invariants
    - `gofmt -l .` clean and `go test ./...` green before push
    - `@wip` removed from the four scenarios owned here as they pass
  - **Dependencies**: T002 (the registration set and help constant must exist)
  - **Plan reference**: Phase 3 — retirement tripwire, ADR-4
  - **Scenario references**: guard.feature: "The guard derives both sides and fails on a vendored-contract change", "The guard anchors the shared help constant's retirement property", "A newly mapped operation whose schema omits the field fails until it is probed", "A registered exception for an unmapped operation fails as stale"
  - **Interface references**: interface-spec.md: Invariants (1–5), Design note on invariant 5, Error Communication
  - **Risk**: ⚠️ Guard over-trigger on refresh noise — parse by structure, mirroring the 072 guard's approach on this file (plan § Risks)
