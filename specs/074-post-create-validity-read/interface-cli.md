# Interface Accord: Post-Create Validity Read — CLI

**Feature**: 074-post-create-validity-read
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: ADR-1 (post-create read-back of `GET /proposals/{id}`), ADR-2 (read-back failure isolated; `Success` for all four verdict states), ADR-4 (create-specific human render delegating to the shared `proposal` template), ADR-5 (machine path emits the read-back's raw `{data}` verbatim; provenance on stderr), ADR-6 (id lifted by local decode). Cross-cutting: the Verdict Assembly and Rendering Design table.

---

This accord pins what a caller of `glassfrog proposal create` sees after this feature lands: the command surface (unchanged), the four verdict states and their rendered vocabulary in each output format, the stderr advisory, and the exit codes (unchanged). The Go-level surface — model fields, render keys, view types, template files — is in `interface-spec.md`.

---

## Surface

### Command — unchanged

```
glassfrog proposal create <tension-id> --changes <json-array|file-path|stdin>
                                       [--output <full|compact|json|yaml|stdin|path-to-template>]
                                       [--base-url <url>]
```

No flag is added, removed, or renamed by this feature. In particular there is **no** `--no-verify`, `--skip-verdict`, `--verify`, or equivalent: the spec forbids an opt-out of the read-back, so the surface offers none. `--changes` remains required with its three sources (reserved keyword `stdin`, an existing regular file, inline JSON); `--output` and `--base-url` remain the inherited persistent flags.

`--output` accepts the four reserved format tokens (`full`, `compact`, `json`, `yaml` — any casing), the reserved `stdin` for a template read from standard input, or any other value as a template file path (035). All six forms are listed above because this accord specifies output for **each** of them, and the compact section below documents rendering under a token the synopsis must therefore name.

What changes is **what the command prints on success**, and that on the success path it performs a second server exchange.

### The four verdict states

Every successful create reports exactly one of these. They are the vocabulary the whole accord is written against.

| State | Meaning | Reached when |
|---|---|---|
| **valid** | the server states the created draft is valid | read-back returned `valid: true` |
| **not valid** | the server states the created draft is not valid | read-back returned `valid: false` |
| **not reported** | the read-back succeeded but the server stated no validity at all | `valid` absent or null in the read-back |
| **unavailable** | no verdict could be obtained, with a reason | the read-back failed, or no `prp_` id could be lifted from the create response |

`not reported` and `unavailable` are distinct and must stay distinct: the first is the server declining to say, the second is the CLI failing to ask. Neither is ever rendered as `valid` or as `not valid`.

### stdout — human `full` format

The created proposal's body renders exactly as it does today (same lines, same order, same explicit-absence markers — the shared singular `proposal` rendering), followed by a verdict block. Labels align to the existing 16-column label field.

```
prp_5e647e6847b74d0aa1b0bd5c2e2f9a11  [draft]
  Tension:        ten_8b871c1d4f3e4a5c9d2e1f0a7b6c5d4e
  Circle:         role_c470a1b2c3d4e5f60718293a4b5c6d7e
  Proposer:       agt_1f2e3d4c5b6a79880123456789abcdef
  Proposed:       (none)
  Deadline:       (none)
  Accepted:       (none)
  Responses:      0 total — 0 no-objection, 0 bring-to-meeting
  Expected/recv:  0 / 0
  Transitions:    propose, withdraw
  Changes (1):
    - [CreatePolicy] name=Deploy windows
  Validity:       valid
  Verdict source: read-back of prp_5e647e6847b74d0aa1b0bd5c2e2f9a11 after create
```

The `Validity` line's value is one of `valid`, `not valid`, `not reported by the server`, or `unavailable — <reason>`.

An `Alerts` block appears **only** when the read-back returned at least one entry in `validation_alerts`, and mirrors the existing `Changes (N):` shape. Each entry renders its severity, the element path it concerns, and the server's message verbatim:

```
  Validity:       not valid
  Alerts (1):
    - [error] name: Can't update the Cloud Foundations role during this meeting.
  Verdict source: read-back of prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b after create
```

When the verdict is unavailable, no `Alerts` block is printed — there are no server-stated alerts to print — and the source line says so:

```
  Validity:       unavailable — the proposal could not be read back (network unreachable)
  Verdict source: none — the created proposal is reported from the create response
```

`Transitions` is **not** part of the verdict block. It is already a line in the shared body and stays there, which is how the accord keeps the transitions dimension visibly separate from the validity dimension rather than folded into it.

### stdout — human `compact` format

One line, as today, with a **compact validity label** appended. Transitions were never in the compact line and are not added here — `compact`'s contract is a scannable summary, and the load-bearing signal (validity) is what this feature adds to it.

The compact label is a **distinct, shorter rendering of the same four states**, not the `full` block's `Validity` value:

| State | `full` block's `Validity` | compact label |
|---|---|---|
| valid | `valid` | `valid` |
| not valid | `not valid` | `not valid` |
| not reported | `not reported by the server` | `validity not reported` |
| unavailable | `unavailable — <reason>` | `validity unavailable` |

`(N alert(s))` is appended whenever the server stated at least one alert, in **either** validity state — so a favourable verdict carrying an advisory alert is visible on the compact line too.

Both labels come from the one mapping function (`interface-spec.md` § Surface), so the four state words are single-sourced and cannot drift apart. What differs is only how much each format spends on them: `full` carries the server's reason text, `compact` cannot — appending an arbitrarily long server-derived reason behind a 36-character id would destroy the one-line contract this section exists to keep.

The id is rendered in full, as the shared compact template already renders it — the examples below are not eliding it.

```
prp_5e647e6847b74d0aa1b0bd5c2e2f9a11  [draft]  1 change(s)  0 responses  valid
prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b  [draft]  1 change(s)  0 responses  not valid (1 alert)
prp_9f8e7d6c5b4a39281706f5e4d3c2b1a0  [draft]  1 change(s)  0 responses  valid (1 alert)
prp_a1b2c3d4e5f6708192a3b4c5d6e7f801  [draft]  2 change(s)  0 responses  validity not reported
prp_e5f6071829a3b4c5d6e7f80192a3b4c5  [draft]  1 change(s)  0 responses  validity unavailable
```

### stdout — machine formats (`--output json`, `--output yaml`)

One server document, verbatim, exactly as every other command emits (018). Which document depends on whether the read-back happened:

- **read-back succeeded** → the read-back's `{data: Proposal}`. It carries `valid`, `validation_alerts`, and `available_transitions` inline, so the verdict is structurally present without any CLI-composed wrapper.
- **read-back did not happen or failed** → the create's own `{data: Proposal}`, unchanged from today's behaviour.

No composed envelope, no added keys, no CLI-invented fields. The document's own content is the signal: a document carrying `valid` came from the read-back.

**A document with no `valid` key is ambiguous on its own** — the server may have declined to state a verdict, or the read-back may never have answered. The emitted document is identical in both cases, because in neither did a server state a verdict for the CLI to pass through. That ambiguity is resolved by the **structured advisory** below, not by adding a key to the server's document: when a machine format is selected the advisory is itself machine-readable, so an agent distinguishes all four states without parsing prose. The server document stays verbatim; the CLI's own diagnostic carries the CLI's own shape.

```json
{
  "data": {
    "id": "prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b",
    "type": "proposal",
    "status": "draft",
    "tension_id": "ten_8b871c1d4f3e4a5c9d2e1f0a7b6c5d4e",
    "changes": [ … ],
    "response_summary": { "total": 0, "no_objection": 0, "bring_to_meeting": 0 },
    "available_transitions": [],
    "expected_response_count": 12,
    "received_response_count": 0,
    "valid": false,
    "validation_alerts": [
      { "severity": "error", "path": "name", "message": "Can't update the Cloud Foundations role during this meeting." }
    ],
    "created_at": "2026-08-08T13:59:04Z",
    "updated_at": "2026-08-08T13:59:04Z"
  }
}
```

*(Abbreviated: `changes` elided. Every other key is as the server returned it — the CLI adds and removes nothing.)*

### stdout — user template (`--output <path-to-template>`)

Renders over the create-specific view. Every field path an existing user template could reference on the create still resolves (the view embeds the one it replaces), and the verdict is additionally available. Field paths are pinned in `interface-spec.md`.

### stderr — the verdict advisory

One advisory on the create success path, in **every** output format — this is how the verdict's provenance, and any reason it is unavailable, reaches a caller whose stdout is a verbatim server document.

The advisory is **format-aware**, following the landed convention that a diagnostic renders structurally when a machine format is selected (032: "a failure reads the same way as a success" on the channel an agent parses). It is token-free and carries no request body.

**Human formats (`full`, `compact`)** — one prose line:

| Situation | stderr line |
|---|---|
| Verdict obtained | `the validity verdict was read back from proposal <prp_id> after the create` |
| Read-back failed | `could not read proposal <prp_id> back to obtain its validity verdict: <reason>; the proposal was created — run "glassfrog proposal get <prp_id>" to read its verdict` |
| No id could be lifted | `could not determine the created proposal's id from the create response, so no validity verdict was obtained; the create response is reported above` |

The read-back-failed line names the remedy, not just the cause: the verdict is still obtainable, the command to obtain it is the one the CLI already ships, and the id it needs is in the output the caller just received. The no-id line names no remedy because there is none to name — without an id there is nothing to re-read, and that is the honest thing to say rather than pointing at a command the caller cannot run.

**Machine formats (`json`, `yaml`)** — the same three situations, rendered in the selected format on stderr. This is the shape that makes all four verdict states machine-distinguishable: `read_back` answers "did the CLI manage to ask?", which is exactly the question the emitted document cannot answer.

Verdict obtained:

```json
{"verdict_source": {"read_back": true, "proposal_id": "prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b"}}
```

Read-back failed:

```json
{"verdict_source": {"read_back": false, "proposal_id": "prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b",
                    "reason": "the read-back was rate limited (the request budget was exhausted)",
                    "remedy": "glassfrog proposal get prp_c76cdbf1a2934e5d8b7a6c5d4e3f2a1b"}}
```

No id could be lifted:

```json
{"verdict_source": {"read_back": false, "reason": "the created proposal's id could not be determined"}}
```

*(Shown one-per-line for readability; the emitted document's exact whitespace follows the existing structured-render helpers.)* `proposal_id` is omitted when no id could be determined, and `remedy` is omitted when there is none — an absent key means "not applicable," never an empty string, so nothing is fabricated. `reason` is absent when `read_back` is `true`.

**Reading the four states in a machine format**:

| State | stdout `data.valid` | stderr `verdict_source.read_back` |
|---|---|---|
| valid | `true` | `true` |
| not valid | `false` | `true` |
| not reported | absent | `true` |
| unavailable | absent | `false` (with `reason`) |

Existing stderr output is unchanged in both format families: 017's retry notices still appear as they do today (and may now appear for the read-back, since a GET is retry-eligible).

Failure diagnostics still route through the shared reporting path, unchanged.

---

## Interactions

### The success path, end to end

1. `--output` is resolved first, then `--changes` is required, read, and floored — all before any request, so the no-request-on-rejection tripwire is unchanged.
2. One `POST /proposals` is sent. It is never auto-retried.
3. On success, the created proposal's id is taken from the response body (no extra request).
4. One `GET /proposals/{id}` is sent for that id, retry-eligible like every other read.
5. The verdict is assembled from `valid`, `validation_alerts`, and `available_transitions`.
6. stdout gets the rendered result; stderr gets the provenance advisory.
7. The command exits `0`.

**Exchange counts** are part of the contract: 0 requests on any pre-request rejection; exactly 1 on a failed create; exactly 2 on a successful create whose read-back is attempted (plus any retries the existing policy performs). Nothing in this feature issues a third.

### What is deliberately absent

- **No polling or waiting.** One read-back attempt, subject only to the retry policy the reads already apply. If the server states no verdict, that is reported, not waited out.
- **No pre-read before the create.** The create is sent without first reading anything; server authorization is unchanged (057 ADR-3's boundary holds).
- **No confirmation for the read-back.** The write-safety guardrail still gates `proposal create` itself; a read needs no gate of its own and none is added.
- **No verdict on the sibling commands.** `proposal get`, `proposal list`, `proposal propose`, `proposal withdraw`, and `proposal respond` produce byte-identical output to today. `proposal get --output json` continues to pass the server's document through verbatim, which already includes `valid` and `validation_alerts` — this feature neither adds that nor starts withholding it.

---

## Error Communication

### Exit codes — unchanged set, unchanged meanings

| Code | Outcome | Reached from this command |
|---|---|---|
| 0 | success | the create succeeded — **including when the verdict is `not valid`, `not reported`, or `unavailable`** |
| 1 | internal error | a render failure (built-in or user template), or an unmarshalable request body |
| 2 | usage error | missing/blank `--changes`, unreadable change source, type-floor violation, bad `--output`, wrong positional count |
| 3 | API error | the **create** returned a generic non-2xx, or an undecodable create body |
| 4 | permission error | the **create** returned 401/403 (includes the Premium gate) |
| 5 | rate limited | the **create** returned 429 after retries |
| 6 | network unavailable | the **create** could not reach the wire |
| 7 | stale write | not reachable from this command (no `If-Match` on a create) |

No code is added and none changes meaning. The single most important row is the first: **an invalid create still exits 0.** That is this feature's stated scope boundary, not an oversight — turning it into a failure exit is the dependent Invalid-Create Outcome capability, which adds a new outcome at both registry sites.

### The read-back's failures never reach an exit code

Every read-back failure resolves to the `unavailable` verdict state with a reason, and the command still exits `0`. The reasons a caller may see, each rendered into both the `Validity` line and the stderr advisory:

| Cause | Reason text shape |
|---|---|
| Wire failure | `the proposal could not be read back (<network cause>)` |
| Non-2xx on the read-back | `the read-back was refused (<status-derived cause>)` |
| 429 after retries | `the read-back was rate limited (the request budget was exhausted)` |
| Undecodable read-back body | `the read-back response could not be read` |
| No id in the create response | `the created proposal's id could not be determined` |

These are reasons, not diagnostics: they do not route through the shared failure envelope, they never replace the created id, and they never produce a non-zero exit. A caller that needs the verdict after seeing `unavailable` re-runs `glassfrog proposal get <prp_id>` — the id it needs is in the output it just received.

### What a failed create looks like

Unchanged in every respect. The failure renders through the shared format-aware failure path, no read-back is attempted, and the exit code is whatever the create's own classification yields (3/4/5/6). A caller cannot distinguish today's failed create from tomorrow's.

---

## Consistency Notes

- **Sibling accord**: `interface-spec.md` pins the Go surface this accord's output is produced by — the two new model fields, the alert type, the new render key and view, and the two template files. Field paths quoted here for the user-template case are defined there.
- **055 (Proposal Creation)** owns the command, its flags, the change-source resolution, and the type floor. This accord changes none of them; it extends only what the command prints on success and adds the second exchange. The one departure from 055's interface is the human render key at the create call site, which the plan records as an announced divergence from 055 ADR-4 — the shared template still renders the body.
- **056 (Proposal Reads)** owns `GET /proposals/{id}`, whose path construction and escaping the read-back reuses rather than re-deriving. `proposal get`'s own output is unchanged by this accord.
- **045 (Tension Discard)** is the precedent for a stderr advisory that disambiguates otherwise-indistinguishable success outcomes; the provenance line follows it. **059 (Withdraw Proposal)** deliberately added *no* advisory because it had one success outcome — the contrast is exactly the point: this command now has four.
- **018 / 020 / 035** are upheld unchanged: a machine format emits one server document verbatim, format selection is resolved before any request, and a user template renders over the command's view. The verdict never introduces a CLI-composed JSON shape.
- **Exit-code convention (004, extended by 054 to 0–7)**: this feature adds nothing to the registry. The "0–6" band in the older header comment is already stale at `codeStaleWrite = 7`; nothing here depends on that comment.
- **Vocabulary**: the four state words (`valid`, `not valid`, `not reported`, `unavailable`) are the same in the compact line, the full block, and the stderr advisory. The `full` and compact **renderings** of those states differ in length — `full` carries the server's reason, compact carries an alert count instead — but both are produced by the one mapping function, so the states themselves cannot diverge. They are presentation labels for one dimension, validity, and are not a roll-up of alerts or transitions, which stay separately rendered.
