# Interface Accord: Plan-Limit Signal — CLI

**Feature**: 061-plan-limit-signal
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-2 (central `Diagnose` refinement, category/exit-code unchanged), ADR-3 (possibility-framed wording; gate carried on the `Diagnostic`), ADR-4 (distinct `feature` envelope element).

---

This accord pins the **operator-facing plan-limit failure surface**: how a recognized plan-gate `403` reads under each `--output` format, the new structured envelope element that names the gating feature, and the exit code it carries. 061 **adds no flag and no command** — it changes only the *wording* and *envelope content* of an existing `403` failure when that `403` came from a known plan-gated operation. It builds directly on 032's failure-rendering surface, adding one envelope key and gate-aware human wording. The Go-package contract (the `output.ErrorDetail.Feature` field, the `Diagnostic.Feature` field, the `apiclient.ResponseError.Method/Path` fields, and the `Diagnose` integration) is pinned in `interface-spec.md`. **The contract here is: the `feature` key (name, presence, source), the gate-aware human line naming the gate and framing possibility, and the unchanged exit code.** The exact cause/next-step wording strings are kept consistent with the spec but are an implementation detail.

---

## Surface

### No new flag or command

061 reuses the `--output` flag and the four format values 020 owns (`full` | `compact` | `json` | `yaml`). It changes only what a recognized plan-gate `403` renders; every other failure — and every non-recognized `403` — renders byte-for-byte as it does today.

### New structured envelope element — `feature` (`json` / `yaml`)

When a command-execution failure is a **recognized plan-limit `403`** (a `403` from an operation Feature-Gate Recognition (060) marks as plan-gated) and the resolved format is `json`/`yaml`, the 018 unified error envelope carries one **additional** field, `feature`, alongside 032's existing fields:

| Field | Type | Presence | Source |
|---|---|---|---|
| `message` | string | always | the diagnostic's **cause** — now the possibility-framed plan-limit cause naming the gate |
| `next_step` | string | when present | the recovery action — now "verify the plan includes <feature>" wording |
| `feature` | string | **only for a recognized plan-limit `403`** | the gating feature's display name (e.g. `Premium async proposals`) — the recognized `Gate`'s display name (ADR-3); a distinct, parseable element, **not** folded into `message` (ADR-4) |
| `kind` | string | always | the category token — **`permission`**, unchanged (the `403` stays `PermissionError`) |
| `status` | integer | non-2xx | `403` |
| `body` | object/array | when the API returned a valid-JSON body | the raw `403` body verbatim |

Field declaration order: `message` → `next_step` → `feature` → `kind` → `status` → `body` (JSON preserves it; YAML emits keys alphabetically — rely on the keys, not their order). `feature` is `omitempty`, so **every non-plan-limit failure renders the exact 032 envelope, byte-stable** — the key appears only for a recognized plan-limit `403`.

**Example** (`--output json`, advancing a draft on a non-Premium org):

```json
{
  "error": {
    "message": "the operation may not be available on your organization's plan — it requires Premium async proposals; a 403 may instead mean your identity lacks permission",
    "next_step": "verify your organization's plan includes Premium async proposals, or that your identity has permission for this operation",
    "feature": "Premium async proposals",
    "kind": "permission",
    "status": 403,
    "body": { "type": "about:blank", "title": "Forbidden", "detail": "Forbidden" }
  }
}
```

### Gate-aware human failure line (`full` / `compact`)

Unchanged channel and shape from 032: stderr carries `renderDiagnostic(d)` — `"<cause> — <next step>"`. For a recognized plan-limit `403`, the cause and next step are the gate-aware ones, so the line **names the gating feature and frames it as a possibility** rather than the generic "check the identity's role/permission" hint:

```
the operation may not be available on your organization's plan — it requires Premium async proposals; a 403 may instead mean your identity lacks permission — verify your organization's plan includes Premium async proposals, or that your identity has permission for this operation
```

The gate name lives in the cause prose here (the human line has no distinct field); the distinct `feature` element exists only in the structured envelope. stdout stays empty under the human formats.

---

## Interactions

**When the signal fires**: only when (a) the failure is a `403`, and (b) `RecognizeFeatureGate` matches the failed operation to a registered plan-gated entry (today: the four Premium async-proposal writes — `proposal create`/`propose`/`withdraw`/`respond`). A `403` from any other operation, and any non-`403` from a gated operation, render exactly as they do today — no `feature` key, generic permission/other wording.

**Possibility, never certainty**: the wording never asserts the plan is insufficient and never tells the caller to upgrade — because a genuine permission `403` on a gated operation is indistinguishable from a plan gate (060 ADR-4). The next step is to *verify*, not to *upgrade*.

**Piping / scripting**: `glassfrog proposal propose <id> --output json` yields a parseable envelope on which an agent can branch on `error.feature` to detect a plan limit programmatically; the exit code (below) remains the authoritative failure signal.

## Error Communication

| Failure | `full` / `compact` | `json` / `yaml` | Exit code |
|---|---|---|---|
| Recognized plan-limit `403` (gated op) | gate-aware cause+next-step on stderr (names the feature, frames possibility) | envelope on stdout with `kind: permission`, `feature: <name>`, `status: 403`, `body` when valid JSON | **`4`** (unchanged) |
| Non-recognized `403` (non-gated op) | today's generic permission cause+next-step on stderr | today's envelope — **no `feature` key** | `4` |
| Non-`403` from a gated op (e.g. `422`, `412`) | today's wording for that status | today's envelope — **no `feature` key** | per status (3 / 7 / …) |

- **The exit code is unchanged** — a recognized plan-limit `403` is still `PermissionError` → exit `4`, identical across formats. 061 refines wording and adds the `feature` element; it never remaps the category or the code (ADR-2). Rendering and code still derive from the one `Diagnose` value, so they cannot disagree.
- No secret (API token, auth header) appears in any rendered plan-limit failure — the cause/next-step are static gate-aware literals + the gate display name, and the envelope carries only response-side facts (031/032/018).

## Consistency Notes

- **Extends `032/interface-cli.md`**: 061 adds one `omitempty` envelope key (`feature`) and gate-aware wording to the failure surface 032 owns; the channel split, exit-code-across-formats invariant, and "one complete document per channel" guarantee are unchanged. Mirrors 032's own additive `next_step` extension.
- **Pairs with `interface-spec.md`**: that file pins the Go symbols — `ResponseError.Method/Path`, `Diagnostic.Feature`, `ErrorDetail.Feature` field/tag, the `Diagnose`↔`RecognizeFeatureGate` integration, and the gate→display-name mapping. This file pins what the operator/agent sees on each channel.
- **Conforms to 060** (`RecognizeFeatureGate`): the gate identity and the possibility-not-certainty framing come from the recognizer; 061 renders, it does not re-recognize.
- **Conforms to 031** (`Diagnose`): the gate-aware cause/next-step/feature all come from the one normalized diagnostic; 061 shapes that diagnostic centrally, it does not fork a renderer.
- **Conforms to 004**: no new exit code; the recognized plan-limit `403` keeps `PermissionError` → `4`.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against.
