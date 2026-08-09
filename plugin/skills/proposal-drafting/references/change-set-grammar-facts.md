> **Empirical record.** Every fact below is observed server behavior, captured
> from live proposal writes against the Glassfrog API. None of it is part of the
> published Glassfrog API v5 contract — where the
> contract already speaks, this record cites it rather than restating it. Treat
> these as verified observations that may change when the server changes, not as
> authoritative reference. This is the residue the contract does not carry; a
> shape the contract absorbs is retired from here, not duplicated.

# Change-Set Grammar Facts

- **Owner**: the proposal-drafting skill. This record lives in that skill's
  `references/` directory as the single source; any other skill that needs it
  consumes it through a symbolic link into its own directory, never a copy.
- **Contract citations**: the published Glassfrog API v5 contract — cited by
  schema and property name only, never by line number and never by restated values.
- **Live facts**: CSG-1, CSG-2

## Contract citations

Where the published contract already carries a shape, this record cites it here
rather than recording it as an empirical fact. These are pointers into the
contract, not restatements of it.

- **Change-type vocabulary**: the full set of governance command types is the
  enum at `ProposalChange.properties.type.enum`. This record names individual
  types (e.g. `CreatePolicy`, `UpdateRole`) only as they appear in a fact's
  Shape; every such name must exist in that enum. No count is pinned here — the
  enum is the source, and a refresh that grows or shrinks it does not falsify
  this record.
- **Nested-only rule**: the accountability and domain change types must appear
  as children of a `CreateRole` or `UpdateRole` part, never as top-level
  proposal changes. This rule and its type list are carried in the description
  of the `ProposalChange` schema. Nested-only types: `CreateAccountability`,
  `UpdateAccountability`, `RemoveAccountability`, `CreateDomain`, `UpdateDomain`,
  `RemoveDomain`. The list is reproduced here — under that explicit label, so
  the guard can set-compare it against the contract — but the rule's content
  stays the contract's.

## CSG-1 — An own-circle policy is a top-level `CreatePolicy` part with no `UpdateRole` wrapper

- **Shape**: a proposal that creates or changes a circle's own policy carries a
  **top-level** `CreatePolicy` part — `{"type": "CreatePolicy", …}` sitting
  directly in the `changes[]` array, not wrapped in an `UpdateRole`. The web UI
  generates exactly this top-level form.
- **Disposition**: accepted.
- **Symptom**: wrapping the policy change as a child of an `UpdateRole` part is
  refused; the accepted shape is the unwrapped top-level `CreatePolicy`, which is
  the form the web UI itself emits. An assembler that reaches for an `UpdateRole`
  wrapper by analogy with accountability/domain edits builds a change set the
  server will not take.
- **Evidence**: `prp_ebe2815f…` — live payload `"changes": [{"type": "CreatePolicy", …}]`.
- **Provenance**: supersedes the provisional field note of 2026-08-05 (F5).

## CSG-2 — An `UpdateRole` self-targeting the circle from inside its own governance is accepted at create but returned invalid

- **Shape**: an `UpdateRole` part whose target is the circle itself, proposed
  from inside that circle's own governance.
- **Disposition**: accepted-but-invalid.
- **Symptom**: the server **accepts the create** and returns a created `prp_`
  id, but the proposal comes back `valid: false` with a blocking alert (e.g.
  *"Can't update the … role during this meeting."*) and `available_transitions:
  []`. A returned `prp_` id here is **not** a successful governance change — the
  draft is dead on arrival and can be removed only via the web UI. A consumer
  must not read the created id as a valid change; "created" is not "valid".
- **Evidence**: `prp_c76cd6bf…` — created, returned `valid: false`, deleted via the web UI.
- **Provenance**: supersedes the provisional field note of 2026-08-05 (F6).
