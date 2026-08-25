# Checklist: Identifier Mapping Harvest

**Feature**: 080-identifier-mapping-harvest
**Checked against**: CONSTITUTION.md (v1.1 — 13 principles, I–XIII)
**Artifacts checked**: spec.md, plan.md, interface-cli.md, tasks.md, features/change-targets-unidentifiable/identifier-mapping-harvest.feature (12 scenarios), features/change-targets-unidentifiable/identifier-mapping-completeness.feature (13 scenarios)
**Checks**: 30 (30 pass, 0 fail)
**Generated**: 2026-08-23 (round 4 — re-derived after the K5/H3 remedy: flag-parity assertion added, first-page accord bullet split, both first-page cases given scenarios)

> Source note: no `accords/governance/` directory is deployed in this repo, so this checklist runs **constitution checks only** — the same standing as the sibling specs' checklists. Done-criteria checks are skipped, not failed.

> Calibration note: this feature adds one read leaf, one pure projection package, and one render resource — no build change, no distribution change, no `plugin/` change. Eleven principles were calibrated to that shape. **XII (Standalone Executable)** produced zero applicable checks (no build, distribution, or dependency change; the new package is stdlib-only). **XIII (Self-Contained Operating Surface)** produced zero applicable checks (no file under `plugin/` is touched — the harvest is deliberately absent from every composed-read registry because no operator path consumes it, per the spec's non-behavior). Every principle is phrased as MUST/MUST NOT, so mechanical severity inheritance puts all 28 checks at P0.

> Verification note (round 4 additions): the spec-to-feature title diff was re-run and is empty in both directions (24/24); the runnable/held counts in tasks.md § Scenario inventory were re-counted against the files (9+10 runnable, 3+3 held) and match; all 29 accord bullets were re-walked against the 25 scenarios with full coverage; zero architecture-informed scenarios remain, both having been absorbed into the accord. Round-2 verifications, unchanged: ADR-1's stdlib depth claim was verified empirically this session (go1.26.6: depth 10000 decodes, 10001 fails `exceeded max depth`; a walker of ADR-1's shape survived the max-depth payload). Round-1 verifications, unchanged: the spec-to-feature title diff is empty in both directions (19/19 verbatim); `grammar.json`'s `change_types` set is exactly the 21-member `ProposalChange.type` enum in `spec/glassfrog-api-v5.yaml` (both counted, membership compared), so ADR-3's guard is buildable as specified; `categoryForStatus` in `internal/cli/diagnostic.go` maps a 5xx to `APIError`, confirming the exit-3 assertion in the mid-walk-failure scenario; `internal/render/templates/*.tmpl` are excluded from the whitespace fixers at `.pre-commit-config.yaml:48` and carry no trailing newline (`proposals.full.tmpl` ends `}`), confirming T004's template constraint; `docs/reference/` already carries per-capability files including `change-set-grammar.md` for `proposal grammar`, so T004's new file follows the convention rather than deviating from it.

---

## Summary

| Severity | Count | Pass | Fail |
|---|---|---|---|
| P0 (blocking) | 30 | 30 | 0 |
| P1 (should fix) | 0 | 0 | 0 |
| P2 (consider) | 0 | 0 | 0 |

**Verdict: all 30 checks pass.** Constitution: 30/30. Done-criteria: not run (no accords). Cross-references: folded into constitution checks IV.2–IV.3 (no accord source exists to generate a separate category).

Both round-2 failures were resolved in round 3. The detail of each is retained below as a record of what was fixed and how.

---

## Resolved Checks (were failing in rounds 1–2)

### ✓ III.4 — A non-integral identifier value has a stated, representable outcome *(resolved round 3)*

**Resolution**: excluded and counted, on the shape the H-4 remedy had already established. A value that is not an exact whole number produces no entry, is never truncated toward one, and is counted in `boundary.unrepresentable_identifiers` — which withholds the completeness claim, so the loss reaches the contract rather than passing unnoticed. Chosen over a null-numbered entry (unkeyable against the (element type, number) key the wrong-element control RC-2 rests on) and over a distinct element-type member (which would conflate *what kind of element* with *whether the value could be read*). A companion non-behavior states why this does not contradict the rule against dropping an untypeable identifier: that rule forbids quietness, not exclusion. The scenario now asserts positively — no entry, `unrepresentable_identifiers` as 1, `complete` false, exit 0 — and T001 carries the matching fixture. The round-2 finding follows.

**Round-2 finding (for the record)**:

**Source**: Principle III (Fail Safe, Not Silent) — *"Errors MUST be obvious and recoverable, never hidden"*; anti-pattern *"swallowing errors"*.
**Artifacts**: interface-cli.md § The derived machine document (`number` field); tasks.md T001 Scope; identifier-mapping-harvest.feature § "A non-integral value under an identifier key routes to unknown rather than truncating".

**Violation**: The three artifacts route a non-integral or out-of-range value to *"unknown handling"*, and no artifact says what that is. The two available readings are mutually exclusive and both collide with a stated contract:

- **Emit an unknown-typed entry** — but `number` is contractually *"(JSON integer, always)"*, and `12345.5` has no integer representation. The entry cannot be formed.
- **Exclude the identifier** — but spec.md § Non-Behaviors states *"The CLI must not drop an identifier it cannot type"*, whose stated reason (*"dropping it makes the mapping quietly incomplete in a way completeness reporting would not catch"*) applies with equal force to an identifier it cannot *represent*.

The scenario cannot discriminate: it asserts only the negative (`no entry will carry the number 12345`) plus `exit code 0`, which both readings satisfy. So the case is untested in either direction, and the failure mode — an identifier silently vanishing from a mapping that reports itself complete — is exactly the hidden-loss shape III forbids.

**Note**: the underlying hazard is real and correctly identified — this check faults the *unresolved* remedy, not the decision to guard the case. Plan.md § Risks names float64 normalization as a high-impact risk and calls for the fixture; the fixture cannot be written until the outcome is stated.

**Remedy** (spec + interface, one edit): decide whether an unrepresentable value produces (a) an entry whose `number` is absent or null with the raw text preserved for the operator, (b) an excluded identifier counted in a new `unrepresentable` count so the loss is reported rather than silent, or (c) a distinct element-type member. Then rewrite the scenario to assert the chosen positive outcome, not only the absence of a truncated value.

### ✓ IV.1 — The empty-mapping behavior has an acceptance scenario *(resolved round 3)*

**Resolution**: added as a spec edge case and a `@wip` scenario in the completeness feature, mirroring the sibling read's *"An empty visible set is a clean success"* — the human "none" idiom, the `total: 0` / empty-`entries` document, and exit 0 all asserted. T003 now references it. All 28 accord bullets have covering scenarios. The round-2 finding follows.

**Round-2 finding (for the record)**:

**Source**: Principle IV (Test-Driven Development) — *"User-facing behavior MUST have an executable acceptance scenario before the code that satisfies it."*
**Artifacts**: spec.md § Behavioral Accord → Emitting the mapping (final bullet); interface-cli.md § Human render (final bullet); features/change-targets-unidentifiable/*.feature.

**Violation**: The accord states *"When the history yields no identifiers at all, the CLI emits an empty mapping, states that it is empty, and exits successfully."* The interface pins the render (*"the existing 'none' idiom (`no identifiers found in the proposal history`), exit 0"*) and the document shape (`total: 0`, `entries: []`). Neither feature file contains a scenario for it. All 27 accord bullets across the six groups were re-walked against the 23 scenarios (round 2 — the new unreadable-record bullet IS covered); this remains the only bullet with no covering scenario.

It is a distinct code path — the empty-render idiom and the zero-entry document are not exercised by any other scenario — and the sibling read already has the equivalent coverage (`proposal-reads.feature`: *"An empty visible set is a clean success"*), so the precedent exists and this spec dropped it. T003's acceptance criteria list the empty case in neither the scenario set nor the decode test.

**Remedy**: add one `@wip` scenario to identifier-mapping-completeness.feature (a history with proposals but no change payloads carrying identifiers → empty `entries`, `total: 0`, the "none" idiom in the human render, exit 0), and add it to T003's scenario references and its runnable count.

---

## Constitution Checks

### I. Spec Fidelity — 4/4 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| I.1 | Every request the harvest issues maps to a spec operation | interface-cli.md § Interactions; validation scenario "No harvested number is ever sent back to the API" | ✓ `GET /proposals` (`listProposals`) is the only request; the validation scenario asserts it |
| I.2 | Every query parameter sent is defined by the spec for that operation | interface-cli.md § Surface flags table | ✓ status, role_id, proposer_id, proposed_after, accepted_after, per_page, cursor — the `listProposals` set `proposal list` already sends |
| I.3 | The derived document is not published as spec-authoritative | interface-cli.md § The derived machine document | ✓ states *"its shape is this accord's contract, not the API's"*; ADR-5 pins the class boundary |
| I.4 | The change-type table derives from the vendored contract, not an invented list | plan.md ADR-3; tasks.md T001 | ✓ guard derives from `grammar.Load()`; verified `grammar.json` change_types (21) == `ProposalChange.type` enum (21), exact membership match |

### II. Action Transparency — 3/3 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| II.1 | The harvest emits machine-parseable output | interface-cli.md § The derived machine document | ✓ `{data: …}` in json and yaml |
| II.2 | Every failure names a cause and a next step | interface-cli.md § Error Communication | ✓ every row routes through the 031 diagnostic path or a usage error naming the value and the supported set |
| II.3 | The operator can tell what coverage the answer has, in every format | spec.md accord; interface-cli.md § field contract | ✓ `complete` + three `boundary` dimensions in-document, stated in every format; stderr notes are additive, not the only signal |

### III. Fail Safe, Not Silent — 6/6 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| III.1 | No error is swallowed | interface-cli.md § Error Communication | ✓ mid-walk stop carries cause in-document, a stderr note, and a classified non-zero exit |
| III.2 | No failure condition is reported as success | Error Communication table | ✓ verified against `categoryForStatus` — a 5xx classifies `APIError`/3, matching the scenario's assertion |
| III.3 | An identifier the CLI cannot type is never dropped | spec.md § Non-Behaviors; harvest.feature "An untypeable identifier…" | ✓ `unknown` vocabulary member with `origins`; scenario asserts the entry exists |
| III.4 | An identifier the CLI cannot represent has a stated, testable outcome | spec.md accord (Typing) + § Non-Behaviors; interface-cli.md `number` + `unrepresentable_identifiers`; harvest.feature "A non-whole-number value is excluded and counted…"; tasks.md T001 fixture | ✓ *(resolved round 3 — see Resolved Checks)* |
| III.6 | *(round 3)* The no-dropping rule and the counted-exclusion rule are simultaneously satisfiable — neither can be honoured only by violating the other | spec.md § Non-Behaviors (the two adjacent bullets) | ✓ the rules are on different axes and say so: an identifier that cannot be **typed** is kept as `unknown`; a value that cannot be **represented** is excluded and counted. No input satisfies one by breaching the other, and the second bullet states the distinction explicitly so a later reader cannot collapse them |
| III.5 | *(round 2)* A record the CLI cannot read is skipped, counted, and reported — never a capped walk, never silent | spec.md accord (Walking, new bullet); interface-cli.md `unreadable_records`; completeness.feature "An unreadable record is skipped and counted…"; tasks.md T001 fixture criterion | ✓ the H-4 remedy: fault isolated at the record, loss carried in the completeness contract, run exits 0, fixture + scenario both pin it |

### IV. Test-Driven Development — 4/4 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| IV.1 | Every accord behavior has an acceptance scenario | spec.md accord (29 bullets) vs 25 scenarios | ✓ 29/29 covered |
| IV.2 | Every spec scenario has a feature scenario with a verbatim `# Source:` title | both feature files | ✓ title diff empty in both directions (24 spec titles, 24 matched) |
| IV.3 | Architecture-informed scenarios are marked as proposed with their source | both feature files | ✓ vacuously — none remain. Both proposals (the float64 exclusion in round 3, the first-page opt-out in round 4) became spec-derived when the accord absorbed their behavior, which is the correct end state for a proposal that survives review rather than a lost finding |
| IV.4 | Tasks sequence tests ahead of the code that satisfies them | tasks.md T001–T004 | ✓ T001/T002 land fixture-pinned pure logic; T003 flips `@wip` only when the scenarios pass |

### V. Composition over Monolith — 2/2 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| V.1 | Adding the harvest forces no edit to an unrelated command | plan.md § System Architecture; spec.md § Non-Behaviors; validation scenario "The proposal list read is unchanged by the harvest" | ✓ new file + new package; the non-behavior is a hard boundary and T003 restates it as an acceptance criterion |
| V.2 | The projection is independently testable | plan.md ADR-2; tasks.md T001 | ✓ `internal/harvest` is pure — T001's acceptance forbids `net/http`, `os`, and cli/render imports |

### VI. Size-Aware by Design — 3/3 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| VI.1 | The walk pages to completion by default | spec.md accord; interface-cli.md § Interactions | ✓ inherits the 056 walk |
| VI.2 | No partial result is presented as complete | spec.md § Non-Behaviors; validation scenario "Exactly the completed unfiltered walk claims completeness" | ✓ four walk endings enumerated, three report partial |
| VI.3 | The boundary signal distinguishes its causes | interface-cli.md § `boundary` | ✓ composed dimensions (narrowed / first_page / stopped) reported independently, never collapsed into one enum |

### VII. Working Software — 2/2 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| VII.1 | Every task pairs implementation with its tests | tasks.md T001–T004 | ✓ each task's acceptance names the fixtures or scenarios that prove it |
| VII.2 | Every task's acceptance includes the build/lint/test gate | tasks.md T001–T004 | ✓ all four name `go test`, `gofmt -l .`, and `golangci-lint run ./...` |

### VIII. No Fabricated Data — 3/3 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| VIII.1 | No value in the mapping is synthesized | spec.md § Non-Behaviors | ✓ numbers and labels come from change payloads; `counts` are aggregates over returned data, not invented values |
| VIII.2 | No label is synthesized for an unlabelled identifier | harvest.feature "An identifier no change ever named appears without a label" | ✓ asserts the empty array and the absence of synthesis |
| VIII.3 | No element type is guessed | spec.md § Non-Behaviors; plan.md ADR-3 | ✓ `unknown` is the stated fallback; the empirical key table is deliberately unguarded *because* it degrades to a visible unknown rather than a silent guess |

### IX. Writes Require Explicit Intent — 1/1 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| IX.1 | The read issues no mutating request | validation scenario "No harvested number is ever sent back to the API"; spec.md § Non-Behaviors | ✓ only the history walk; a non-behavior forbids assembling or sending any change from a harvested number |

### X. Respect API Limits — 1/1 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| X.1 | The harvest introduces no new retry loop and honours the existing 429 handling | plan.md § System Architecture, § Cross-cutting Concerns | ✓ rides `apiclient.NewRetryExecutor` on the same walk `proposal list` performs; no concurrency added. `If-Match` is N/A (read-only) |

### XI. Governance via Proposals — 1/1 pass

| # | Check | Artifact | Result |
|---|---|---|---|
| XI.1 | The harvest exposes no governance-mutation path | spec.md § Non-Behaviors; interface-cli.md § Error Communication | ✓ read-only; consuming a harvested number is explicitly the drafting path's act, behind the write-safety gate this command never touches |

### XII. Standalone Executable — 0 applicable checks

No build, distribution, packaging, or dependency change. `internal/harvest` is stdlib-only. Nothing in this feature can violate XII.

### XIII. Self-Contained Operating Surface — 0 applicable checks

No file under `plugin/` is touched. The harvest is deliberately absent from every composed-read registry — no operator path consumes it, and the spec's non-behavior keeps consumption with the drafting path. `surfaceselfcontainment.go`'s walk is unaffected.

---

## Governance Infrastructure Notes

- **No `accords/governance/` directory** — done-criteria checks (done-specify, done-plan, done-interface, done-scenarios, done-tasks) cannot be generated. Consider creating them to enable per-skill quality checks; until then every Score spec in this repo runs constitution-only checks.
- **Two principles produced zero applicable checks** (XII, XIII), each with the reason recorded above rather than silently omitted.
- **No structural mismatches** — CONSTITUTION.md v1.1 uses a consistent principle/rationale/detection structure throughout, and every principle is phrased MUST/MUST NOT, so severity inheritance is uniform at P0.

---

## Improvement Summary (round 2 vs round 1)

Previous run: 28 checks, 2 P0 fail (III.4, IV.1). Current run: 29 checks, 2 P0 fail (III.4, IV.1 — both unchanged; neither was in the H-4/H-5 remedy's scope).

- **Added and passing**: III.5 — the unreadable-record isolation introduced by the amendment, with its accord bullet, interface dimension, scenario, and fixture criterion all in agreement.
- **Strengthened without a check change**: VI.2/VI.3 now rest on a four-dimension boundary; V.2's purity claim now includes the projection's own decode (`Harvest([]json.RawMessage)` keeps `internal/harvest` free of the typed model while still importing nothing beyond stdlib).
- **Still failing**: III.4 (the non-integral value's outcome is still "unknown handling" with no representable shape) and IV.1 (the empty mapping still has no scenario). Both remedies are unchanged from round 1.

---

## Improvement Summary (round 3 vs round 2)

Previous run: 29 checks, 2 P0 fail (III.4, IV.1). Current run: 30 checks, 0 fail.

- **Resolved**: III.4 (unrepresentable values now excluded, counted, and positively asserted) and IV.1 (empty-mapping scenario added; 28/28 accord bullets covered).
- **Added and passing**: III.6 — the two adjacent non-behaviors were checked for mutual reachability, because a rule pair where one can only be honoured by breaching the other is a defect nothing else detects.
- **Changed without a check change**: IV.3 now covers one architecture-informed scenario rather than two — the float64 scenario became spec-derived when the accord absorbed its behavior, which is the correct migration rather than a lost proposal.
- **Constitution coverage unchanged**: XII and XIII still produce zero applicable checks, for the reasons recorded above.

---

## Improvement Summary (round 4 vs round 3)

Previous run: 30 checks, 0 fail. Current run: 30 checks, 0 fail. No check changed verdict; the round closed the two sibling findings analyze had raised (K5, H3) and strengthened two checks' evidence.

- **Strengthened**: IV.1 (29/29 accord bullets covered after the first-page bullet split produced two conditions, each with its own scenario) and IV.3 (no architecture-informed scenarios remain — both were absorbed into the accord).
- **Newly detectable, not newly checked**: the flag-parity invariant the accord declares now has a derive-both-sides assertion in T003. This is an analyze finding (K5), not a constitution check, but it removes the *"an accord asserts what no guard checks"* condition that VII's spirit disfavours.
