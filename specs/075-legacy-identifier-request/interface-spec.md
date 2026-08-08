# Interface Accord: Legacy Identifier Request — Specification (retirement tripwire guard)

**Feature**: 075-legacy-identifier-request
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-4 (parameter-coverage guard deriving the `IncludeLegacyId` operation set from the vendored spec and cross-checking the CLI's registration set).

---

This accord pins the structural contract of the capability's retirement tripwire: a best-effort `internal/build` guard in the 062/072 config-drift family. The guard is a test, not a runtime surface — its "consumers" are the refresh PR that trips it and the maintainer who reads its failure. Invocation surface: N/A beyond `go test ./...` (it is an ordinary test in `internal/build`). Configuration surface: N/A (nothing configurable).

---

## Surface

### Files

| File | Role |
|---|---|
| `internal/build/legacyidcoverage.go` | The declared contract-fact mapping + derivation helpers (production source, per the operator-path BDD convention that drift-guard helpers do not live in `_test.go`) |
| `internal/build/legacyidcoverage_test.go` | The guard test asserting the invariant groups below |

### The declared contract-fact mapping

One Go map literal — operation id (vendored-spec vocabulary) → CLI command leaf (cobra vocabulary):

```go
// operation id → command path whose leaf registers --legacy-id
{"listRoles": "roles", "getRole": "roles", "getRoleTree": "tree",
 "listActors": "actors", "getActor": "actors", "getMe": "me"}
```

This mapping is a **checked-in contract fact, not a second source of truth** (the 067 boundary nuance): the *sets* on each side are derived at test time — the spec side from the vendored YAML, the CLI side from the live command tree — and the map exists only because YAML cannot know cobra names. Operation ids map to four distinct leaves; the map's value set is the registration set.

### Inputs

| Input | Derivation |
|---|---|
| Spec-side set | Parse `spec/glassfrog-api-v5.yaml` by structure (the 072 guard's parsing approach, never line-grep): every operation whose `parameters` include a `$ref` to `#/components/parameters/IncludeLegacyId` |
| CLI-side set | The live root command tree (the 064/065 extractor family): which leaves carry a local `legacy-id` flag |
| Help-text constant | The shared caveat string from `internal/cli`, read from source, not re-declared |

---

## Interactions

Derivation flow, per test run:

1. Parse the vendored spec → set **S** of operation ids referencing `IncludeLegacyId`.
2. Read the declared mapping → key set **K**, value set (commands) **C**.
3. Build the live command tree → set **L** of leaves carrying a local `legacy-id` flag.
4. Assert the invariant groups below over S, K, C, L.

The guard hard-codes no operation ids and no command names outside the declared mapping, and the mapping itself is asserted against both derived sides — deliberate retirement or widening updates the mapping and passes; a partial edit fails loudly (072's manifest lesson).

## Invariants

| # | Invariant | What it protects |
|---|---|---|
| 1 | **S = K** (set equality, both directions) | Parameter removed upstream → fails at the refresh PR (the retirement alarm); parameter added to new operations → fails, surfacing the coverage question |
| 2 | Every command in **C** resolves to a live leaf and that leaf's `legacy-id` flag is **local** (present on the leaf, absent from its children — decisive on `me`, whose subcommands are contract-excluded) | The supported set the CLI offers matches the declared facts |
| 3 | **L = C** — no other leaf in the entire tree registers `legacy-id` | The spec's "exactly the supported reads and nowhere else" validation scenario, held mechanically |
| 4 | Each registration's flag usage string is the one shared constant, and that constant names the property being guarded: it must convey *temporary/transition* and *retirement* (assert on the constant's identity + the two stems, e.g. `transition`/`retire`, not on exact prose) | The retirement clock stays where the spec put it (help text) without freezing wording — the check satisfies its message (guard-message-as-specification lesson: pin the property, not the phrasing) |
| 5 | Every mapped operation's response schema either **declares** `legacy_id` (directly or through `allOf` composition) **or** appears in the declared **observed-exception register** with its probe evidence. The register's entries must be a *subset* of the mapped operations — an exception for an operation nobody maps is stale and fails | Catches a *new* mapped operation whose schema cannot carry the field, without failing on the one where the schema is known-defective. See the design note below — this invariant deliberately is not a bare assertion |

### Design note on invariant 5 — why it is a register, not an assertion

The obvious-looking form of this invariant is *"every mapped operation's response schema declares `legacy_id`."* That form is **unpassable**, and the reason is worth stating so nobody "tightens" it later.

`getRoleTree` references the parameter and returns the field on every node at every depth, but its `TreeNode` schema declares no `legacy_id` and no `additionalProperties` (LEARNINGS W1 — 1336 nodes, depth 6, all integer, zero nulls). The schema is upstream-defective. Under the bare assertion, the guard fails permanently, the vendored file cannot be edited (standing precedent for this artifact), and the only way to reach green is to **drop a fully working read** — so the guard's sole passing state would be the removal of correct behavior.

Hence the register. Its one entry today is:

```go
// operation id → why its schema cannot be trusted, and what was observed instead
{"getRoleTree": "TreeNode omits legacy_id; live probe returns it on all nodes to full depth (LEARNINGS 2026-08-08, W1)"}
```

Each entry carries its probe evidence, so the exception is a recorded observation rather than a silenced check. The subset rule keeps the register honest: if `getRoleTree` ever leaves the mapping, its stale exception fails the build instead of lingering. And a *newly* mapped operation whose schema omits the field still fails invariant 5 until someone probes it and either fixes the mapping or registers the exception — which is exactly the check's purpose, since deriving coverage from `$ref` sites alone is what produced this feature's one P0.

## Error Communication

Failure messages name the condition **and its reachable remedy**, scoped by the nature of the disagreement (the #190 lesson — remedies checked against sibling invariants so every prescribed fix can actually go green):

| Failure | Message names |
|---|---|
| S ≠ K, operation missing from spec side | "`<op>` no longer references IncludeLegacyId in the vendored spec — the bridge is retiring for this read: remove the mapping entry AND the flag registration together (invariants 2/3 will fail until both are gone)" |
| S ≠ K, new spec-side operation | "`<op>` now accepts IncludeLegacyId but is not in the declared mapping — decide coverage: add mapping + registration, or record the exclusion in the mapping's comment" |
| C/L mismatch (invariants 2–3) | which leaf is missing the flag, or which unexpected leaf carries it — remedy: align registration with the mapping (never edit the vendored spec) |
| Invariant 4 | which registration diverged from the shared constant, or which stem the constant lost |
| Invariant 5, schema omits the field and no exception registered | "`<op>`'s response schema does not declare `legacy_id` — probe the operation, then either correct the mapping or register the exception with its probe evidence. Do not edit the vendored spec." |
| Invariant 5, stale register entry | "`<op>` has a registered schema exception but is not in the mapping — remove the stale exception" |

**Explicitly partial, stated in the guard's comment**: the guard cannot see the API *behaviorally* dropping the field while the vendored contract still declares it (numbers stop arriving with every check green). That residue is by spec design — reads degrade silently, the refresh diff is the detection instrument, and this guard is that instrument's mechanical half.

---

## Consistency Notes

- Joins the `internal/build` guard family as the **flag-surface ↔ vendored-contract** anchor (072 added knowledge↔contract, 073 premise tripwire); same file-structure conventions (production-source helpers, one guard file pair, structural YAML parsing).
- The mapping's shape follows the `VerifyRunners`-style checked-in contract-fact pattern, with the property (why each entry exists) in its comment so a future editor re-derives rather than copies.
- Sibling: interface-cli.md defines the flag surface and shared help constant this guard anchors; the constant lives in `internal/cli` and is imported/read, never duplicated here.
- No STATUS/enumeration is hard-coded in prose elsewhere: orientation (062) defers flag detail to `--help`, so no skill artifact needs a matching edit (plan "Does Not Cover").
