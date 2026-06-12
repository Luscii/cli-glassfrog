# Interface Accord: Resolution Call-Site Retrofit — CLI

**Feature**: 040-resolution-call-site-retrofit
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-2 (flag presence via cobra `Changed()` replaces value-emptiness)

---

This accord pins the **one operator-facing behaviour change** the retrofit makes: how an explicitly-supplied **empty or whitespace-only** `--base-url` or `--output` is treated. It adds **no flag and no command** — `--base-url` (008/011) and `--output`/`-o` (020) keep their names, scope, values, precedence, default, and inherited-persistent-root-flag shape. The Go contracts behind this change are pinned in `interface-spec.md`. Everything not stated below is unchanged from 008/020.

---

## Surface

No surface additions. The affected flags, unchanged in every other respect:

| Flag | Short | Precedence chain | Default |
|---|---|---|---|
| `--base-url` | — | flag → `GLASSFROG_BASE_URL` → `.glassfrogrc base_url` → built-in default | `https://app.glassfrog.com/api/v5` |
| `--output` | `-o` | flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` → built-in default | `full` |

The token (`GLASSFROG_TOKEN` / `.glassfrogrc token`) has **no flag** and is unaffected by this accord.

---

## Interactions

### The change: the flag rung is now presence-based, not value-based

"Was the flag supplied?" is now decided by cobra `Changed()` (presence) rather than by whether its value is non-empty after trimming. This affects only the case where the flag is supplied with an empty or whitespace-only value:

| Invocation | Before (value-emptiness) | After (presence) |
|---|---|---|
| `--base-url https://x/api` | flag wins, used | flag wins, used (unchanged) |
| `--base-url ""` or `--base-url "   "` | treated as absent → **falls through** to env/file/default | flag is **supplied** → wins its rung → validated → **fails loud** (not a usable URL) |
| (no `--base-url`) | falls through | falls through (unchanged) |
| `--output json` | flag wins, used | flag wins, used (unchanged) |
| `--output ""` / `-o "  "` | treated as absent → **falls through** | flag is **supplied** → wins → parsed → **fails loud** (not a valid format token) |
| (no `--output`) | falls through | falls through (unchanged) |

**Environment and file rungs are unchanged**: a whitespace-only `GLASSFROG_BASE_URL` / `GLASSFROG_OUTPUT`, or a whitespace-only `.glassfrogrc` value, is still treated as absent and falls through. The presence change applies to the **flag rung only** — the rung where the operator's explicit act of typing the flag is the signal.

Rationale: typing `--base-url ""` is an explicit instruction; honouring it (and reporting it as invalid) is more truthful than silently ignoring it and connecting to the default endpoint. This aligns the two flag rungs with the project's `Changed()`-not-value convention and with `resolve`'s flag semantics (`interface-spec.md`).

---

## Error Communication

The new failures reuse the existing typed errors and exit code — **no new exit code, no new message shape**:

| Condition | stdout | stderr | Exit code |
|---|---|---|---|
| `--base-url ""` / `--base-url "  "` (supplied, empty/whitespace) | nothing (no request) | `base URL from --base-url is not a valid absolute http(s) URL` (the existing `*BaseURLError` text) | `2` (UsageError) |
| `--output ""` / `-o "  "` (supplied, empty/whitespace) | nothing (no request) | the existing `*FormatError` text naming `--output` and the value, e.g. `unsupported output value "" from --output — supported: full, compact, json, yaml` | `2` (UsageError) |
| Whitespace-only env/file value (flag absent) | — | — falls through to the next rung (unchanged) | unchanged |
| Every other path (valid value, malformed non-empty value, unreadable `.glassfrogrc`, …) | unchanged | unchanged | unchanged |

The errors map to `UsageError(2)` through the same `classifyClientError` arms (`*BaseURLError` / `*output.FormatError` → `UsageError`) that already exist (011/015/020) — the empty-flag case simply reaches them one rung earlier than an absent flag would.

---

## Consistency Notes

- **Refines 008 (`--base-url`) and 020 (`interface-cli.md`, `--output`)**: same flags, same precedence, same exit code; only the empty/whitespace-supplied-flag outcome changes (absent-via-fall-through → present-and-rejected). The valid-value and present-but-malformed paths are byte-identical.
- **Pairs with `interface-spec.md`**: the presence bit (`cmd.Flags().Changed(...)`) is threaded from each `RunE` through `AssembleFromOS` / the `resolveFormat` seam into the resolvers' `FromFlags` rung — the code contract for this behaviour.
- **Conforms to 004 / `classifyClientError`**: reuses the frozen `UsageError(2)` convention and the single classification chain; adds no arm and renumbers nothing.
- **Token is out of scope**: no `--token` flag exists, so the presence change touches only `--base-url` and `--output`.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against.
