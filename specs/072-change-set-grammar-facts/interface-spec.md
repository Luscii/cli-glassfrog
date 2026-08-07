# Interface Accord: Change-Set Grammar Facts — Specification

**Feature**: 072-change-set-grammar-facts
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: plan.md System Architecture — "The record" and "The guard" (ADR-1 home/ownership, ADR-2 form, ADR-3 guard coupling)

---

## Surface

### Invocation

N/A — the record has no entry point. It is consumed by reading: loaded by future consumer skills (#75/#77), grepped by future builder scoping (#83), and parsed by the guard at CI. Nothing invokes it.

### Configuration

N/A — the record has no configurable parameters. Its path is a shared constant in `internal/build` (see Guard Coupling).

### The record file

**Path**: `plugin/skills/proposal-drafting/references/change-set-grammar-facts.md` — owned by the proposal-drafting skill (plan ADR-1); the first `references/` directory in the shipped plugin.

**Required file anatomy, in order**:

| # | Element | Structural requirement |
|---|---|---|
| 1 | Empirical marker | A leading blockquote (`> …`) before any heading, containing the phrase `Empirical record` and stating that every fact is observed server behavior and none of it is part of the published Glassfrog API v5 contract. This is the guard's marker anchor. |
| 2 | Document header | `# Change-Set Grammar Facts` title, followed by three field lines: an **Owner** line naming the proposal-drafting skill and the symlink consumption rule, a **Contract citations** line naming `spec/glassfrog-api-v5.yaml`, and the **Live facts** manifest (row 3). |
| 3 | Live-facts manifest | A header field line `**Live facts**: <ID>[, <ID>…]` declaring the fact IDs the record currently carries — e.g. `**Live facts**: CSG-1, CSG-2`. This is the record's own declaration of its membership, and the reason the guard needs no hard-coded fact list (plan ADR-3). **Invariant: set-equality with the actual fact sections** — listing order is document order by convention, not a checked rule. It sits in the header, not in the contract-citations section: that section describes the *published contract*, while the manifest describes *this record's* content. |
| 4 | Contract-citations section | A `## ` section that **cites, never restates**, what the published contract already carries: (a) the change-type enum, anchored as `ProposalChange.properties.type.enum`; (b) the nested-only rule, anchored to `ProposalChange`'s schema description, carrying the cited type list explicitly (currently the six accountability/domain types) so the guard can set-compare it. Anchors are schema/property names — never line numbers, never restated enum values beyond the citation lists, and never a pinned count. |
| 5 | Fact sections | One `## ` section per recorded fact, headed `## CSG-<n> — <title>`; the heading is the guard's parse anchor for both the ID and the title. Exactly the facts the manifest declares, no more and no fewer. At landing: `CSG-1` and `CSG-2`. **IDs are allocated monotonically and never reused** — a retired ID stays retired, because DEPRECATION.md entries and any downstream consumer treat the ID as a permanent handle. A record with zero fact sections is not a valid record (see Error Communication condition 3). |

**Per-fact structural contract** — each fact section carries five required labelled fields, as bold-labelled list items (`- **<Label>**: …`); labels are the machine anchors, layout around them is free:

| Field | Type | Requirement |
|---|---|---|
| Shape | prose + inline JSON fragment | The accepted or offending change-set part shape, concrete enough to assemble from. Every change-type name mentioned must exist in the cited enum. |
| Disposition | closed vocabulary | Exactly one of `accepted`, `rejected`, `accepted-but-invalid`. |
| Symptom | prose | The observable consequence of misusing the shape — what the caller sees, not internal behavior. |
| Evidence | identifier(s) | The live object(s) the fact was verified against (`prp_…` ids, abbreviated as captured). |
| Provenance | reference | The LEARNINGS entry the fact supersedes (2026-08-05, F5 or F6). |

**Fact content at landing**:

| ID | Title (section heading after the ID) | Disposition |
|---|---|---|
| CSG-1 | An own-circle policy is a top-level `CreatePolicy` part with no `UpdateRole` wrapper | `accepted` (the wrapped alternative's refusal rides in the Symptom) |
| CSG-2 | An `UpdateRole` self-targeting the circle from inside its own governance is accepted at create but returned invalid (`valid: false`, blocking alert, `available_transitions: []`) | `accepted-but-invalid` |

### Guard coupling

**Files** (family convention — derivation helpers in production source, assertions in the test):

| File | Contents |
|---|---|
| `internal/build/grammarfacts.go` | The record's path constant; parsers deriving every side — record side (marker presence, the **live-facts manifest**, fact section IDs and titles, field labels, disposition values, cited type names, nested-only citation list) and spec side (`ProposalChange.properties.type.enum` values, the nested-only type set from the schema description). |
| `internal/build/grammarfacts_guard_test.go` | The guard test asserting the invariants below. |

Every side is derived from its file at test time. The guard hard-codes **no enum values, no type names, and no fact IDs** (plan ADR-3) — the expected fact set comes from the record's own manifest, which is what lets a retirement pass instead of tripping the guard.

---

## Interactions

**Consumer flows** (all read-only; none ship in this feature):

- **Pre-assembly consultation** (#75/#77): the drafting path loads the record before building a `changes[]` array — finds the accepted shape to use (CSG-1) and the dead shape to avoid (CSG-2) without a refused round-trip. The record is local to the owning skill; no symlink needed for this primary path.
- **Builder scoping** (#83): Go tooling greps fact sections for verified change-type names to bound which typed builders exist.
- **Second skill-consumer** (future, none planned): consumes via a symbolic link into its own skill directory, added by the spec that adds the consumption — never a copy (plan ADR-1 rule).

**Maintenance flow** — retirement (plan ADR-4) is a **deliberate three-part act**, all three parts in one change:

1. **delete** the fact section (never marked retired in place);
2. **drop its ID from the `Live facts` manifest** — omit this and the guard fails, which is the design: retirement is explicit, never a side effect of an edit;
3. **record the supersession** via `/score:deprecate`, naming what retired and what absorbed it.

The retired ID is not reused. When the last fact retires, the **record itself is deleted** rather than kept as an empty shell — a zero-fact record fails the guard (condition 3). Git history is the changelog. The same flow lands this feature: one `/score:deprecate` entry supersedes LEARNINGS 2026-08-05 F5/F6.

**Guard flow**: CI parses record and vendored spec on every build; a refresh that moves the enum or the nested-only set fails at the record, forcing re-derivation or retirement before the change lands.

---

## Error Communication

The guard is the record's only enforcement surface. Each violation fails the build loudly, naming the invariant, the offending element, **and which resolution path applies** — the design admits exactly two legitimate answers to a citation failure (re-derive the citation, or retire the fact), so the message says which rather than leaving the reader to infer it from the plan (plan ADR-3). Never silently, never partially: the first failure does not suppress reporting of the rest where the test can continue.

| # | Condition | Failure message names | Resolution path named |
|---|---|---|---|
| 1 | The manifest declares a fact ID with no matching `## CSG-<n>` section | the declared-but-absent ID | Complete the retirement (drop the ID) — or restore the section if the deletion was unintended |
| 2 | A fact section's ID is absent from the manifest | the present-but-undeclared ID | Add the ID to the manifest — or finish the deletion if the fact was meant to retire |
| 3 | The record has zero fact sections | that the record is empty | Delete the record and record the supersession; an empty record is not a valid state |
| 4 | A required field is missing or empty in a fact | the fact ID and field label | Supply the field |
| 5 | A Disposition value outside the closed vocabulary | the fact ID and the unrecognized value | Use one of `accepted` / `rejected` / `accepted-but-invalid` |
| 6 | A change-type name mentioned in the record is absent from the spec's `ProposalChange.properties.type.enum` | the type name and the citation anchor | Re-derive the citation, or retire the fact that names it |
| 7 | The record's nested-only citation list is not set-equal to the spec description's nested-only set | both sets, so the drift is readable from the failure | Re-derive the citation, or retire the fact it supports |
| 8 | The empirical marker blockquote is absent or missing its required phrase | the marker requirement | Restore the marker |

Conditions 1 and 2 are the two directions of the manifest invariant, and they are deliberately distinct: together they make a **partial** retirement fail while a **complete** one passes. This replaces an earlier contract that required `CSG-1` and `CSG-2` by name — which would have failed the build on every legitimate retirement.

**Degradation**: none — the record has no optional inputs. **Explicitly partial** (stated, not silent): the guard cannot detect the spec *semantically* absorbing CSG-2 (accepted-but-invalid is behavior prose, not a schema key); that residue belongs to the refresh-diff review and is named in the guard's comment.

---

## Consistency Notes

- **No sibling interface files** — the feature has one touchpoint; there is no interface-cli.md (the CLI is untouched) and no interface-api.md (no API surface is added).
- **Extends, does not deviate from, the single-source + guard idiom**: the `plugin/agents/*-commands.txt` pattern (one machine-readable source consumed by artifact and drift test) applied to a markdown knowledge record; guard file naming follows `internal/build` family convention (§461, §465, §470 precedents).
- **The manifest is a self-consistency check, not tamper-proofing**: it catches a *partial* edit (section removed, manifest entry left behind, or the reverse) but not a thorough-but-wrong one — an edit that removes both halves passes. That is inherent to any in-repo scheme and is accepted, not solved (plan R4); what protects the record beyond it is that retirement takes a visible second edit in the diff plus a deprecation entry. Stated here so no later reader mistakes the invariant for a stronger guarantee than it is.
- **Citation-list nuance**: the contract-citations section carries the six nested-only type names *as a citation list* — this is deliberate and is not a restatement: the rule's content (what nested-only means, where children ride) stays the spec's; the list exists so the guard can set-compare record against spec, which is exactly the retirement tripwire the Maintenance accord requires.
- **Empirical marker vs. docs conventions**: reference files in this repo open with a `> …` blockquote describing the file (e.g. the Score references); the marker reuses that shape, adding the guard-anchored `Empirical record` phrase.
- **No accords/ directory exists** in this repo; project conventions come from PROJECT.md and DECISIONS.md precedent, both honored above.
