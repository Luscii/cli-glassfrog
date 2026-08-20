# Interface Accord: Invalid-Create Outcome — CLI

**Feature**: 078-invalid-create-outcome
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (`InvalidCreate` → exit code 8, arms at all three `Outcome` switch sites), ADR-2 (typed error through the single Diagnose chain), ADR-3 (envelope carries `proposal_id` + `validation_alerts` as uniform `omitempty` extensions), ADR-4 (failure envelope on stdout, no verbatim document, no advisory — announced narrowing of 074 ADR-5), ADR-5 (cause and remedy elements).

---

This accord pins what a caller of `glassfrog proposal create` sees when the server accepts the write but reports the created draft **not valid**: the new exit code, the failure document in each output format, and the diagnostic wording. Everything a caller sees on the three remaining success states is pinned by 074's accord and is byte-identical after this feature.

---

## Surface

### Command — unchanged

```
glassfrog proposal create <tension-id> --changes <json-array|file-path|stdin>
                                       [--output <full|compact|json|yaml|stdin|path-to-template>]
                                       [--base-url <url>]
```

No flag is added, removed, or renamed. In particular there is **no** `--allow-invalid`, `--no-fail`, or equivalent: the spec forbids an opt-out of the failure, so the surface offers none — exactly as 074 offered no opt-out of the read-back.

### The four verdict states and their outcomes

074 defined four verdict states, all reported as `Success`. This feature splits one of them off as a failure, leaving **three success states and one failure state**. The verdict vocabulary is unchanged — what changes is the outcome one of them maps to:

| State | Meaning | Outcome after this feature |
|---|---|---|
| **valid** | the server states the created draft is valid | success, exit `0` — unchanged |
| **not valid** | the server states the created draft is not valid | **failure, exit `8` — this feature** |
| **not reported** | the read-back succeeded but the server stated no validity | success, exit `0` — unchanged |
| **unavailable** | no verdict could be obtained, with a reason | success, exit `0` — unchanged |

The trigger is precise and server-stated: the read-back answered **and** carried an explicit `valid: false`. Nothing else trips it — not an absent `valid`, not a failed read-back, not the presence of alerts on a valid draft, not an empty `available_transitions`. A `valid: true` draft carrying alerts is the *valid* state and succeeds, alerts rendered as 074 pinned them.

### stdout — machine formats (`--output json`, `--output yaml`) on the failure

The shared failure envelope (018/032), in the selected format — **not** the server's proposal document, and **not** accompanied by a `verdict_source` advisory. The envelope gains two keys, present only on this failure:

```json
{
  "error": {
    "message": "the server accepted the create but reports proposal prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b not valid (read back after the create)",
    "next_step": "review the alerts, check \"glassfrog proposal grammar\" for documented invalid shapes, and create a corrected proposal from the same tension; the invalid draft can be deleted in the GlassFrog web UI",
    "kind": "invalid-create",
    "proposal_id": "prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b",
    "validation_alerts": [
      { "severity": "error", "path": "name", "message": "Can't update the Cloud Foundations role during this meeting." }
    ]
  }
}
```

Field contract:

| Key | Presence | Content |
|---|---|---|
| `message` | always | the cause, naming the created `prp_` id and the verdict's provenance (the read-back). Token-free. Does **not** enumerate the alerts — they have their own key. |
| `next_step` | always on this failure | the remedy (ADR-5): review the alerts, consult `glassfrog proposal grammar`, create a corrected proposal from the same tension; the invalid draft is deletable in the GlassFrog web UI. |
| `kind` | always | `"invalid-create"` — joins `usage` / `runtime` / `network` / `api` / `permission` / `rate-limit` / `stale-write`. |
| `proposal_id` | this failure only (`omitempty`) | the created draft's `prp_` id, verbatim. Never withheld on this failure. |
| `validation_alerts` | this failure only, and only when the server attached at least one alert (`omitempty`) | every alert the server attached, each carrying the server's own `severity`, `path`, and `message` values under those same key names. Each key is **present only when the server sent it** — a value the server omitted stays omitted rather than being reconstructed as `""` (the live probe saw all three on every entry, so a partial entry is unobserved, not impossible). An invalid draft with **zero** alerts still fails; the key is then **absent**, not an empty array (`omitempty` omits a zero-length list). Absence means "the server attached none", never "there were none to look for". |
| `status`, `body`, `feature` | absent | no exchange failed and no plan gate fired — the keys don't apply, so they don't appear. |

Every other failure of every command is byte-identical to today: the two new keys are `omitempty` and only this failure populates them.

### stderr — human formats (`full`, `compact`) on the failure

stdout stays empty (the failure convention everywhere in the CLI). stderr carries the diagnostic: the cause, one line per alert, then the next step. Both human formats render it identically.

```
the server accepted the create but reports proposal prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b not valid (read back after the create)
  error name: Can't update the Cloud Foundations role during this meeting.
 — review the alerts, check "glassfrog proposal grammar" for documented invalid shapes, and create a corrected proposal from the same tension; the invalid draft can be deleted in the GlassFrog web UI
```

Alert-line shape: two-space indent, `<severity> <path>: <message>`, one line per alert in the server's order. When the server attached no alerts there is no alert block at all, and the diagnostic collapses to the single `cause — next step` line every other CLI failure renders — not a two-line form with an empty gap. The cause and next-step strings are the **same** strings the machine envelope carries in `message` and `next_step` — one source, two renderings.

### stdout — user template (`--output <path-to-template>`) on the failure

A user template does not render this failure. Failures bypass template rendering everywhere in the CLI (032): a template path is a human format for the *success* document, so on the invalid-create failure the caller gets the stderr diagnostic above and exit `8`, and their template is never invoked. This is pinned so a template author knows the failure path is outside their projection — the three success states remain renderable exactly as 074's accord defines.

### What is no longer emitted on this path

- The server's proposal document (either of the two) — the spec's clarification pins envelope-only; the full object stays one `glassfrog proposal get <prp_id>` away.
- The `verdict_source` advisory — its two facts (the CLI asked; here is the record it read) are subsumed by `kind: "invalid-create"` and `proposal_id`. The advisory continues to fire on the three success states exactly as landed.

---

## Interactions

### The invalid path, end to end

1. Steps 1–4 are byte-identical to 074's accord: format resolution, `--changes` floor, one `POST /proposals` (never retried), one `GET /proposals/{id}` (retry-eligible).
2. The verdict check runs after the read-back and **before any stdout write** — a failed create never half-emits.
3. When the read-back answered with `valid: false`: the failure renders (envelope to stdout in a machine format; diagnostic to stderr in a human format) and the command exits `8`.
4. Every other verdict state proceeds exactly as 074 landed it, exit `0`.

**Exchange counts are unchanged**: 0 on pre-request rejection, 1 on a failed create, exactly 2 on a successful create with an attempted read-back. The failure classification adds no request.

### Reading the outcomes in a machine format

| State | Exit | stdout | stderr |
|---|---|---|---|
| valid | 0 | read-back document (`valid: true`) | advisory `read_back: true` |
| **not valid** | **8** | **failure envelope, `kind: "invalid-create"`** | *(nothing new; retry notices only)* |
| not reported | 0 | document without `valid` | advisory `read_back: true` |
| unavailable | 0 | create's document | advisory `read_back: false` + `reason` |
| *(create itself rejected)* | 3/4/5/6 | failure envelope (unchanged) | — |

No state requires prose-scraping; the failure is the only state with a non-zero exit among verdict-carrying outcomes, and the only one whose stdout carries `kind`.

### What is deliberately absent

- **No retry, poll, or wait for validity** — the verdict is final for this change set; the remedy is a new proposal.
- **No widening** to `propose`, `withdraw`, or `respond` — those fail loudly at the server and have no accepted-but-invalid state.
- **No draft-discard command** — the remedy names the web UI for cleanup (ADR-5); if a discard command ever lands, only the `next_step` string changes.

---

## Error Communication

### Exit codes — the registry becomes 0–8

| Code | Outcome | Reached from this command |
|---|---|---|
| 0 | success | the create succeeded and the verdict is *valid*, *not reported*, or *unavailable* |
| 1 | internal error | a render failure, or an unmarshalable request body |
| 2 | usage error | missing/blank `--changes`, unreadable change source, type-floor violation, bad `--output`, wrong positional count |
| 3 | API error | the **create** returned a generic non-2xx, or an undecodable create body |
| 4 | permission error | the **create** returned 401/403 (includes the Premium gate) |
| 5 | rate limited | the **create** returned 429 after retries |
| 6 | network unavailable | the **create** could not reach the wire |
| 7 | stale write | not reachable from this command (no `If-Match` on a create) |
| **8** | **invalid create** | **the server accepted the write but reports the created draft not valid** |

Code 8 is previously unused; no existing code is renumbered or changes meaning (004's extension rule, as 054 exercised it for 7). Codes 126/127/128+N remain unassigned. An agent's reaction to `8`: do **not** blind-retry the same change set — the draft exists but is dead; read the alerts, revise, create a new proposal.

### The read-back's failures still never reach an exit code

074's table of unavailable-reasons is unchanged in text and in consequence: every read-back failure resolves to the *unavailable* state, exit `0`. Failing on an unobtainable verdict would fail on absence, which the spec forbids as the mirror image of reading absence as success.

### What a failed create looks like

Still unchanged: the create's own rejection renders through the shared failure path with no read-back and exits 3/4/5/6. The invalid-create failure is distinguishable from it in every format — by exit code, by `kind`, and by the presence of `proposal_id` (a rejected create has no id; an invalid create always carries one).

---

## Consistency Notes

- **074 (Post-Create Validity Read)** is the accord this one amends. Its exit-code table's row 0 (the create succeeds "including when the verdict is `not valid`, `not reported`, or `unavailable`", with the prose below the table stating that **an invalid create still exits 0**) and ADR-2's "Success for all four verdict states" described the landed intermediate state and explicitly forecast this feature as the change. Per the 031-deprecation precedent, 074's spec-directory artifacts stay at their historical wording; the **live** convention surfaces are swept instead (below). A `/score:deprecate` entry against 074 ADR-2's four-states clause is flagged in the plan.
- **Announced narrowing of 074 ADR-5**: "the machine path emits the later document verbatim + advisory" now governs the success outcomes only. On the failure, 032's landed convention takes over (envelope on stdout). 018 is not violated — no server document is reshaped; one is *withheld from stdout* on a failure, which is what every failure already does.
- **Envelope extension follows 061's `Feature` precedent**: declared in `internal/output` (018's home) with the populated-in-cli comment pattern, `omitempty`, absent from every failure that doesn't carry it. The alert entries reuse the server's own key spellings (`severity`, `path`, `message`) so an agent that already parses the success document's `validation_alerts` parses these identically.
- **Convention-stating surfaces swept in the same change** (plan **Phase 1** for the orientation skill row — a drift guard couples that row to the constant itself — and for `exitcode.go`'s own narrative paragraph, which rides with the code it describes; plan **Phase 2** for the rest): `plugin/skills/orientation/SKILL.md` ("a fixed range, 0–7" and its table gain code 8 — self-containment lexicon respected: no spec numbers), `docs/guides/how-to-read-exit-codes.md`, `docs/explanation/how-failures-are-reported.md` if it enumerates kinds, and `exitcode.go`'s narrative comment (a 078 paragraph after the 054 one).
- **Vocabulary**: the state words stay 074's four, with *not valid* now also carrying the outcome name **invalid create** (`kind: "invalid-create"`, exit 8). The kind token and the exit-code row use the same hyphenation everywhere: `invalid-create` as token, "invalid create" as prose.
