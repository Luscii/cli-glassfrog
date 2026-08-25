# Interface Accord: Identifier Mapping Harvest — CLI

**Feature**: 080-identifier-mapping-harvest
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (structural descent), ADR-2 (`internal/harvest` projection over raw records; command as thin 056-shaped orchestrator), ADR-3 (typing tables + grammar-derived drift guard), ADR-4 (recognized naming set + tagged fallback, own-object collection), ADR-5 (derived `{data: …}` document), ADR-6 (one render resource, narrowed-keyed branch).

---

This accord pins the operator-facing surface of the harvest: the command and its flags, the derived machine document's field contract, the human render's summary-vs-listing rule, the stderr notes, and the exit behavior. The walk, format-selection, and pagination machinery it rides on is pinned by 016/020/035/056; exit codes by 004 (+ 015/054/078 extensions). Nothing here consumes a harvested number — resolution is the sibling capability's future concern, and the drafting path's after that.

---

## Surface

### `glassfrog proposal identifiers` — harvest the legacy numeric identifiers from the proposal history

Verb leaf on the existing `proposal` group (registration guard; the group is already a parent). `cobra.NoArgs` — the read is organization-global, scoped only by flags.

**Short**: `Harvest the legacy numeric identifiers recorded in the organization's proposal history`

**Flags** (all command-local; names, value shapes, and help wording mirror `proposal list` — same five server-side filters, same walk controls):

| Flag | Type | Default | Contract |
|---|---|---|---|
| `--status` | string | (absent) | Validated against the one closed proposal-status set (`supportedProposalStatuses`) before any request; empty means no filter. Presence of a non-empty value marks the mapping **narrowed**. |
| `--role-id` | string | (absent) | Free value passed through as `role_id`. Non-empty ⇒ narrowed. |
| `--proposer-id` | string | (absent) | Free value passed through as `proposer_id`. Non-empty ⇒ narrowed. |
| `--proposed-after` | string | (absent) | Free value passed through as `proposed_after`. Non-empty ⇒ narrowed. |
| `--accepted-after` | string | (absent) | Free value passed through as `accepted_after`. Non-empty ⇒ narrowed. |
| `--first-page` | bool | `false` | Single-page opt-out. Sets the document's `first_page` boundary dimension regardless of whether more pages existed; the more-exist stderr note is emitted only when the API reports a next page (the `proposal list` note discipline). |
| `--per-page` | int | (absent) | Page size, presence-keyed (`Changed()`), passed through unclamped — an out-of-range value surfaces the API's rejection. |

Inherited persistent flags (`--base-url`, `--output`) behave exactly as on every read. There is no way to persist any harvest behavior through the environment or an rc-file key.

### The derived machine document

For `-o json` / `-o yaml`: a **CLI-authored derived document** — the first of its class (plan ADR-5) — in the standard `{data: …}` envelope. It is not an echo of any server response; its shape is this accord's contract, not the API's.

```json
{
  "data": {
    "complete": false,
    "boundary": { "narrowed": true, "first_page": false, "stopped": null, "unreadable_records": 0, "unrepresentable_identifiers": 0 },
    "counts": {
      "total": 2916,
      "unlabelled": 553,
      "by_element_type": { "accountability": 913, "circle": 41, "domain": 187, "person": 305, "policy": 96, "role": 1349, "role_note": 0, "unknown": 25 }
    },
    "entries": [
      {
        "element_type": "accountability",
        "number": 31763476,
        "labels": [
          {
            "text": "Delivering the roadmap for the platform",
            "key": "description",
            "recognized": true,
            "last_proposal_id": "prp_9919982d…",
            "proposal_count": 3
          }
        ]
      },
      {
        "element_type": "unknown",
        "number": 55512345,
        "origins": [ { "key": "widgetDatabaseId", "change_type": "MoveItems" } ],
        "labels": []
      }
    ]
  }
}
```

**Field contract**:

- `complete` (bool, always present): `true` iff every `boundary` dimension is clear — not narrowed, not first-page, `stopped` null, `unreadable_records` zero, and `unrepresentable_identifiers` zero. The single authoritative completeness claim; the stderr notes are courtesies, this field is the contract.
- `boundary` (object, always present — all-clear on a complete walk, so parsers see one stable shape):
  - `narrowed` (bool): any of the five filters carried a non-empty value.
  - `first_page` (bool): the `--first-page` opt-out was taken.
  - `stopped` (string or `null`): `null`, or the cause text when the walk stopped on an error after gathering at least one page.
  - `unreadable_records` (integer, always present, `0` on a clean walk): how many walked records could not be read as proposals and were skipped — the projection harvests every readable record and counts the rest; one bad record never caps the walk.
  - `unrepresentable_identifiers` (integer, always present, `0` on a clean walk): how many values under an identifier key were not exact whole numbers and were therefore excluded rather than truncated. These are **composed dimensions, not an enum** — a filtered walk can also stop, a stopped walk can also have skipped a record; each dimension reports independently.
- `counts` (object, always present): `total` (entry count), `unlabelled` (entries with an empty `labels` array), `by_element_type` (every vocabulary member present, zero included, in the fixed order below). Derived from `entries`; the two never disagree.
- `entries` (array, always present, `[]` on an empty mapping). Each entry:
  - `element_type` (string): closed snake_case vocabulary — `accountability`, `circle`, `domain`, `person`, `policy`, `role`, `role_note`, `unknown`. Per-row table assignments (which change type or key spelling yields which member) are implementation-decided under the ADR-3 drift guard with per-row provenance comments; this accord pins the vocabulary, the spelling, and the discipline that anything unassignable is `unknown`, never dropped, never guessed.
  - `number` (JSON integer, always): normalized from either observed source spelling (string or number). A value that is not an exact whole number produces **no entry at all** — it is never truncated toward one and never carried as a null-numbered entry, because entries are keyed on (element type, number) and an unkeyed entry could be neither deduplicated nor matched. The exclusion is counted in `boundary.unrepresentable_identifiers`, so it withholds the completeness claim rather than passing unnoticed.
  - `origins` (array of `{key, change_type}`): present **only** when `element_type` is `unknown` — the key spelling(s) and enclosing change type(s) the number was found under, deduplicated, first-appearance order. Absent on typed entries.
  - `labels` (array, always present): each label is `{text, key, recognized, last_proposal_id, proposal_count}` — the label text; the payload key it was read from; whether that key is the recognized naming field for the enclosing change type; the id of the most recent proposal (by the proposal's `created_at`) that carried this text; and how many proposals carried it. Deduplicated per entry by `(text, key)`.

**Ordering** (deterministic, pinned): entries by `element_type` alphabetically (`accountability` … `unknown` — the fallback lands last), then `number` ascending. Labels within an entry in first-appearance-in-walk order. `by_element_type` keys in the same fixed alphabetical order.

### Human render

One render resource (`ResourceIdentifierMapping`), compact and full templates, branching on the view's narrowed state (plan ADR-6):

- **Whole-history walk** (complete, or stopped-on-error with no filter/opt-out): **summary only** — a completeness line, per-element-type counts (zero rows omitted in the render), and the unlabelled count; when records were skipped as unreadable or values excluded as unrepresentable, the summary states how many of each. No entry listing; the structured formats and operator templates are the full-mapping surface.
- **Narrowed walk** (any filter or `--first-page`): summary **plus** the entry listing — per entry: element type, number, and each label's text with its recognized-or-key tag; an unlabelled entry shows the existing absence idiom; an `unknown` entry names its origin key and change type.
- **Empty mapping**: the existing "none" idiom (`no identifiers found in the proposal history`), exit 0.
- The listing rule keys on *narrowing*, never on entry count — the same invocation always renders the same way.

### Operator templates

`-o <template-ref>` addresses the full view struct: everything the machine document carries plus the narrowed state. Entries are **always present in the view**, including on a whole-history walk where the built-in template shows only the summary — the curation is the built-in renders' behavior, not a data restriction (the 075 template principle).

---

## Interactions

1. Output format resolves first (020/035): a bad `--output` selector or unreadable template fails as a usage error before anything else.
2. `--status` validates second, pure and pre-request: an unsupported value is a usage error naming the value and the supported set; no request is sent (tripwire-pinned).
3. Connection assembly, client, retrying executor — the 056 shape unchanged.
4. The walk: `GET /proposals` as raw records (`paging.All[json.RawMessage]` — the machinery `proposal list`'s machine path already runs; every output format consumes the projection). Default walks to completion; `--first-page` fetches one page; `--per-page` sizes pages.
5. Projection: `harvest.Harvest(rawRecords)` — pure; it decodes each record itself, skipping and counting any that will not read as a proposal. The command layer then attaches walk-ending completeness (which it alone knows) and merges the projection's unreadable count into the boundary.
6. Render per resolved format. Machine: the derived document. Human: summary or summary+listing. Template: the view.

The harvest issues no request other than the history walk — no per-identifier reads, no legacy-id opt-in on any read, ever.

---

## Error Communication

No new Outcome, no new exit code, no new envelope kind. The 0–7 convention (+ 8 from 078) is referenced, not extended.

| Condition | stdout | stderr | Exit |
|---|---|---|---|
| Success — complete walk | document/render, `complete: true` | — | 0 |
| Success — filtered and/or `--first-page` | document/render, `complete: false`, dimensions set | more-exist note only if the API reported a next page under `--first-page` | 0 |
| Walk stops on error after ≥1 page | partial document/render, `stopped` carries the cause | harvest-worded incomplete note (sibling of `incompleteProposalsWalkNote`): names the cause, states the mapping is partial | classified non-zero of the stop cause |
| Walk completes but N records were unreadable | document/render, `complete: false`, `unreadable_records: N` | unreadable-records note naming the count | 0 — a reported degradation, not a failure (plan § Cross-cutting Concerns) |
| Walk completes but N identifier values were unrepresentable | document/render, `complete: false`, `unrepresentable_identifiers: N` | unrepresentable-values note naming the count | 0 — same reported-degradation posture |
| Walk fails before any page (auth, plan refusal, transport, rate-limit) | failure per format-aware rendering (032) | diagnostic per 031 | classified non-zero — exactly `proposal list`'s |
| Unsupported `--status` value | — | usage error naming value + supported set | 2 |
| Invalid `--output` / template source | — | usage error | 2 |
| Empty history / empty mapping | empty document (`total: 0`, `entries: []`) or "none" idiom | — | 0 |

New stderr note constants (harvest-worded, defined beside the command):
- incomplete-walk: `note: result is incomplete — %s; the identifier mapping shown is partial`
- more-exist (first-page): `note: more proposals exist than harvested; re-run without --first-page for the whole history`
- unreadable-records: `note: %d record(s) in the walked history could not be read as proposals and were skipped; the mapping may be missing their identifiers`
- unrepresentable-values: `note: %d value(s) under an identifier key were not whole numbers and were excluded; the mapping may be missing those identifiers`

The document's `boundary`/`complete` fields carry the same facts in every format — a machine consumer never needs stderr to know the mapping is partial.

---

## Consistency Notes

- **Deviation from the verbatim-server-document invariant (018), by design**: every prior machine output echoes a server document; this one is CLI-authored because the mapping exists in no response. The class boundary is pinned here and in DECISIONS.md: *derived documents exist only where no server document does*. No echo command may cite this accord to reshape a response.
- **`proposal list` is untouched**: same walk, same filters, same notes discipline — but no flag, field, or rendering of `proposal list` changes. Shared helpers are reused only where they need no change; wording-divergent pieces (the two notes) are new constants, not edits.
- **Flag parity with `proposal list` is deliberate and load-bearing**: the five filters keep their exact names, value shapes, and pass-through semantics so an operator's knowledge transfers; drift between the two flag sets is a defect against this accord — and a **detectable** one: a parity assertion derives both flag sets from the two constructed commands and fails on any divergence in name, value type, or default (tasks T003). The accord states no invariant that nothing checks.
- **The composed `boundary` object follows the composed-status precedent** (065-era): five independent dimensions (narrowed, first-page, stopped, unreadable-records, unrepresentable-identifiers) reported independently, never collapsed into one exclusive enum value.
- **No credential-free path**: unlike `proposal grammar` (077), the harvest is a real walked read — it authenticates, retries, and rate-limits like every other read.
- **Sibling accord, not consumed**: `--legacy-id` (075) and this harvest are complements; neither surface references the other's output, and the harvest never sends `include_legacy_id`.
