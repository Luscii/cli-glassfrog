# Interface Accord: Agent-Facing Grammar Reference — CLI

**Feature**: 077-agent-facing-grammar-reference
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (generated, committed, embedded artifact; record stays at its 072 home), ADR-3 (structured at generation — the artifact is a real contract), ADR-4 (rendering through the Resource × Format machinery; `json`/`yaml` serialize the structure directly), ADR-5 (regenerate-and-compare guard with named invariants), Integration Design (the `proposal grammar` leaf and the `PROPOSAL_READS` edit).

---

This accord pins the operator-facing surface of the grammar reference: the command, the rendered structure per format, the exit-code envelope, and the write-gate conduct. The one structure defined here does double duty by design: it is both the command's `json`/`yaml` output **and** the committed embedded artifact's payload contract — the generator writes it, the drift guard pins it, the command embeds and renders it. There is no second schema to drift. The serialization/format machinery it rides on is pinned by 018/019/020/035; exit codes by 004 (+ 015/054 extensions).

---

## Surface

### The command

`glassfrog proposal grammar` — a read leaf under the existing `proposal` group.

- **Arguments**: none. Cobra rejects any positional argument as a usage error before command code runs — this is the accord's "change set offered for judgment" refusal: there is no input path, and the failure is usage, never a validity verdict.
- **Command-local flags**: none.
- **Inherited persistent flags**: `--output` participates fully (format selection below). `--base-url` parses but is inert — the command makes no request, so the flag changes nothing. Credential settings are never resolved.
- **Short help**: names the surface honestly — the change-set grammar for proposal changes: the part shapes and placement rules from the published contract, plus verified empirical observations, consulted before assembling. Help text must state that the command judges nothing (the server stays the judge of validity).

### The rendered structure (`json` / `yaml`)

The top-level document has exactly two keys, both always present:

| Key | Type | Content |
|---|---|---|
| `change_types` | array | one entry per change type the published contract enumerates |
| `facts` | array | one entry per live fact in the grammar record; `[]` when the residue is empty — the key is never omitted |

**`change_types[]` entry**:

| Field | Type | Contract |
|---|---|---|
| `type` | string | a member of the contract's `ProposalChange.properties.type.enum`, verbatim (e.g. `CreatePolicy`) |
| `placement` | string | `top-level` \| `nested-only` |
| `wrappers` | array of strings | present **only** when `placement` is `nested-only`: the role-operation parts the type must ride inside — `["CreateRole", "UpdateRole"]` |
| `provenance` | string | `published-contract` |

**`facts[]` entry**:

| Field | Type | Contract |
|---|---|---|
| `id` | string | the record's fact id (`CSG-1`, `CSG-2`, …) — permanent handles, never reused |
| `title` | string | the fact's one-line heading from the record |
| `shape` | string | the shape to use, as the record states it |
| `disposition` | string | the record's closed vocabulary, verbatim: `accepted` \| `rejected` \| `accepted-but-invalid` |
| `symptom` | string | the observable symptom of getting the shape wrong, as the record states it |
| `provenance` | string | `empirical-observation` |

**Provenance tokens** are the two kebab strings above and nothing else — agents compare them literally. Every entry in both arrays carries one; a consumer can always tell a contract-published shape from a verified observation.

**Excluded by design**: the record's per-fact **Evidence** and lineage (**Provenance**) fields do not render in any format. They are maintenance metadata; the lineage text references development-repository history that is meaningless — and would be a leaked repository pointer — on an operator's machine.

**Ordering is deterministic**: `change_types` sorted alphabetically by `type`; `facts` in the record's live-facts manifest order (ascending id). Two invocations of the same binary produce byte-identical structured output.

**Example (`--output json`, abbreviated — the real enum carries every contract type)**:

```json
{
  "change_types": [
    {"type": "CreateAccountability", "placement": "nested-only", "wrappers": ["CreateRole", "UpdateRole"], "provenance": "published-contract"},
    {"type": "CreatePolicy", "placement": "top-level", "provenance": "published-contract"},
    {"type": "UpdateRole", "placement": "top-level", "provenance": "published-contract"}
  ],
  "facts": [
    {
      "id": "CSG-1",
      "title": "An own-circle policy is a top-level CreatePolicy part with no UpdateRole wrapper",
      "shape": "…",
      "disposition": "accepted",
      "symptom": "…",
      "provenance": "empirical-observation"
    },
    {
      "id": "CSG-2",
      "title": "An UpdateRole self-targeting the circle from inside its own governance is accepted at create but returned invalid",
      "shape": "…",
      "disposition": "accepted-but-invalid",
      "symptom": "…",
      "provenance": "empirical-observation"
    }
  ]
}
```

`yaml` is the same document in YAML serialization.

### The human formats (`full` / `compact`)

Content requirements, not byte contracts (the golden render tests pin bytes):

- **`full`** must present: the change-type vocabulary with each type's placement; the nesting rule (nested-only types and their wrappers) stated once; every fact with its title, shape, disposition, and symptom; and a visible provenance marking that separates contract-published content from empirical observations. When `facts` is empty, `full` states that no empirical residue is currently recorded — the section never silently disappears.
- **`compact`** must present: every type with its placement class in condensed form, and each fact as a one-line summary carrying id, disposition, and title. The same empty-residue statement applies.

### The embedded artifact

The committed artifact wraps the rendered structure in a maintenance envelope:

```json
{
  "generated": "<do-not-edit marker naming the regeneration step>",
  "grammar": { "change_types": [...], "facts": [...] }
}
```

The command renders the `grammar` value and nothing else — the envelope's marker is repo-facing and never reaches output. The drift guard pins the whole file (byte-equality against in-memory regeneration) and requires the marker (ADR-5 invariant). The `grammar` value **is** the structure defined above; there is exactly one definition.

---

## Interactions

- **Consultation flow**: one invocation returns the whole reference — no pagination, no `--per-page`, no partial views, no state. The intended sequence (an assembler runs it before building a change set) is packaged by the downstream consultation capability; nothing in this command tracks whether consultation happened.
- **Format selection**: the house chain (`--output` flag → env → `.glassfrogrc` → default) selects among `full`, `compact`, `json`, `yaml`, or a template-file reference. A user template (`-o <file>`) applies over the grammar structure exactly as templates apply over read payloads (035 semantics, including the template-forces-full-path behavior).
- **Credential-free conduct**: the command works before `glassfrog auth login` has ever run, with no credential file present, and with a malformed one — token resolution is never invoked, not invoked-and-ignored.
- **Offline conduct**: no network access is attempted; output is identical with and without connectivity.
- **Write-gate conduct**: the operating surface's write gate passes `proposal grammar` ungated — `grammar` joins `list` and `get` in the hook's recognized-read set (`PROPOSAL_READS`). The proposal surface's checked-in expectation (`expectedProposalSurface`) gains the leaf in the same change; that expectation is what fails the build on an unclassified leaf, while the script's own conduct on an unrecognized subcommand is to fail closed and ask. Both edits ship with the command (plan § Integration Design).

---

## Error Communication

| Condition | Outcome | Exit code |
|---|---|---|
| Reference rendered | Success | 0 |
| Any positional argument (including a change set offered for checking) | UsageError — cobra's usage message; no validity language | 2 |
| Unknown flag | UsageError | 2 |
| Embedded artifact fails to decode (build-guaranteed impossible; corrupt build) | RuntimeError — CLI-internal fault | 1 |

Codes 3–7 (API, permission, rate-limit, network, stale-write) are **unproducible** by this command — a contract fact, not an aspiration: no request path exists to produce them. Usage errors follow every command's existing conduct (stderr + usage text); nothing bespoke.

---

## Consistency Notes

- **Deviation — `json` is not a server-envelope mirror.** Every API read's `json` output reproduces the server's response envelope; this command has no response to mirror, so `json` serializes the embedded structure directly. That is the pass-through spirit applied to a knowledge read: the structure *is* the raw data. Recorded as a deliberate deviation so no reviewer "fixes" it toward `{data: …}`.
- **One schema, three consumers.** The structure above is the artifact payload the generator writes, the shape the guard regenerates and compares, and the command's structured output. A change to it is a contract edit reviewed against all three.
- **Wrapper derivation left to tasks** (within plan ADR-2's nothing-hand-maintained discipline): the `wrappers` pair is stated in the contract's `ProposalChange` description prose. If generation-time prose-parsing proves brittle, the sanctioned fallback is a checked-in contract fact set-compare-guarded against the vendored spec — the 067/075 nuance (sets derived, cross-vocabulary mappings declared). The output shape is pinned either way.
- **Upstream record contract (072)**: the record's five labelled per-fact fields are the source; this accord consumes `title`, `shape`, `disposition`, `symptom` and deliberately drops `Evidence` and lineage. The disposition vocabulary is the record's, verbatim — this accord introduces no third spelling.
- **No `accords/` directory exists** in this repository — there are no cross-feature accord files to check against; the house conventions cited here live in the prior interface accords (004, 018/019/020, 035) and the exit-code registry.
- **Sibling interface files**: none — the CLI touchpoint is this feature's only external surface; the artifact contract is pinned here rather than in a separate `interface-spec.md` because it is the same structure as the output (see "One schema, three consumers").
