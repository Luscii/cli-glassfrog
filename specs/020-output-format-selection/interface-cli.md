# Interface Accord: Output Format Selection — CLI

**Feature**: 020-output-format-selection
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-2 (`--output`/`-o` persistent root flag), ADR-1 (precedence chain mirroring `--base-url`), ADR-3 (dispatch to `internal/render` vs `internal/output`), ADR-4 (invalid selector → `UsageError(2)`).

---

This accord pins the **operator-facing selection surface**: the `--output` flag, the four format values, the resolution precedence, and what each format puts on stdout. 020 **adds one flag and no command** — the existing commands, args, and their flags (`--include` on `me`; `--status` on `me actions`/`me projects`; `--base-url` everywhere) are unchanged. It makes `compact`, `json`, and `yaml` reachable from the command line for the first time. The Go package API (the `internal/output` selection vocabulary + resolver, and the `internal/cli` dispatch) is pinned in `interface-spec.md`. Literal flag/env/key spellings are the contract here; usage strings are kept consistent with the spec but are an implementation detail.

---

## Surface

### New global flag

| Flag | Short | Scope | Values | Default |
|---|---|---|---|---|
| `--output` | `-o` | Persistent on the root command; inherited by every command (mirrors `--base-url`, 011 ADR-2) | `full` \| `compact` \| `json` \| `yaml` (case-insensitive) | `full` |

- Usage string (shape): `Output format — full | compact | json | yaml (overrides GLASSFROG_OUTPUT, the .glassfrogrc output, and the built-in default)`.
- The flag is **accepted anywhere on the command path** — `glassfrog --output json me` and `glassfrog me --output json` resolve identically (cobra interspersed parsing over an inherited persistent flag).
- Value matching is **case-insensitive** over exactly the four tokens: `json`, `JSON`, `Json`, `jSON` all select `json`. Only the four token names are valid in any casing; anything else is a usage error (below).
- The flag appears (inert) on non-result commands (`version`, `auth …`) — the same accepted wart as `--base-url` (011 ADR-2). It has observable effect only on commands that produce result data.

### Affected commands

| Command | Synopsis (unchanged) | `--output` effect |
|---|---|---|
| `glassfrog me` | `me [--include roles] [-o <fmt>] [--base-url …]` | success body routed to the selected renderer |
| `glassfrog me roles` | `me roles [-o <fmt>] [--base-url …]` | success body routed to the selected renderer |
| `glassfrog me actions` | `me actions [--status …] [-o <fmt>] [--base-url …]` | success body routed to the selected renderer |
| `glassfrog me projects` | `me projects [--status …] [-o <fmt>] [--base-url …]` | success body routed to the selected renderer |

### What each format puts on stdout

| Format | Renderer | stdout shape (authoritative source) |
|---|---|---|
| `full` (default) | `internal/render` `FormatFull` | the labelled human projection — **019 interface-cli.md** is authoritative (field-equivalent to each read's pre-019 projection) |
| `compact` | `internal/render` `FormatCompact` | one line per record, ids always present, nested collections as counts — **019 interface-cli.md** is authoritative |
| `json` | `internal/output` `RenderSuccess(JSON, raw)` | a single JSON document of the raw API payload, verbatim — **018 interface-spec.md** is authoritative |
| `yaml` | `internal/output` `RenderSuccess(YAML, raw)` | the identical data as a single YAML document — **018 interface-spec.md** is authoritative |

020 selects and routes; it defines no new stdout shape of its own. The same successful result feeds whichever renderer is chosen — selection changes the rendering, never which fields the command fetched.

---

## Interactions

**Resolution precedence** (ADR-1, mirroring `--base-url` 008): the effective format is the first source that yields a value, otherwise the built-in default:

1. `--output` / `-o` flag value
2. `GLASSFROG_OUTPUT` environment variable
3. `.glassfrogrc` `output` key (nearest-wins up the directory tree, then the home-directory file — the same walk Credential Discovery 005 and Base URL Resolution 008 use)
4. built-in default `full`

A source that is **absent** is skipped to the next rung; a source that is **present but not one of the four tokens** is a usage error naming that source (Error Communication). Resolution always yields a format. The resolved *source* is not surfaced to the operator (unlike `--base-url`, whose active endpoint is reported) — the effective format is the whole need.

**Order of operations**: the format is resolved **before any API request** (the decode target depends on it — `interface-spec.md` / 018 ADR-2), so an invalid selector fails before the network is touched. Resolution is independent of `--base-url` resolution and connection assembly (009).

**Piping / scripting**: `json`/`yaml` are the agent-parseable channels (a single document on stdout, diagnostics on stderr — 018). `full` stays the byte-stable default an existing script already parses, so adding `--output` changes nothing for callers that omit it.

## Error Communication

| Condition | stdout | stderr | Exit code |
|---|---|---|---|
| Valid format, read + render succeed | the rendered document | — | `0` (Success) |
| `--output <bad>` (e.g. `--output xml`) | nothing (no request made) | usage error naming the value, e.g. `unsupported --output value "xml" — supported: full, compact, json, yaml` | `2` (UsageError) |
| `GLASSFROG_OUTPUT` / `.glassfrogrc output` holds an invalid value (flag absent) | nothing | usage error naming the **source** and value, e.g. `unsupported output value "xml" from GLASSFROG_OUTPUT — supported: full, compact, json, yaml` | `2` (UsageError) |
| `.glassfrogrc` unreadable / unparseable while resolving `output` | nothing | the `internal/rcfile` read/format error naming the file (mirrors `--base-url`) | `2` (UsageError) |
| Read succeeds, render fails | nothing (no partial output) | the render error message (token-free) | `1` (RuntimeError) |
| Read fails (transport / API / auth / decode) | nothing | existing cause-plus-next-step message (unchanged) | existing code (1/3/4/5/6) |

- The invalid-selector and resolution errors map to `UsageError(2)` through the shared `classifyClientError` chain — the same class as a malformed `--base-url` (interface-spec.md). **No new exit code**; 004's convention is unchanged.
- **Interim failure-rendering gap** (spec Assumption): until Output-Aware Failure Rendering (032) lands, a *command* failure under `json`/`yaml` keeps today's cause-plus-next-step text on stderr — it is **not** yet wrapped in 018's error envelope. 020 renders only its own invalid-selector usage error; 032 makes failures format-aware. This mirrors how 019 shipped `compact` built-but-unreachable until 020.

## Consistency Notes

- **Pairs with interface-spec.md**: that file pins the `internal/output` selection vocabulary + resolver and the `internal/cli` dispatch; this file pins what the operator types and sees. The per-format stdout shapes are owned by 019 (`full`/`compact`) and 018 (`json`/`yaml`) — this accord references them rather than redefining them.
- **Conforms to `--base-url` (011 ADR-2)**: `--output` is a persistent root flag read by inheritance, name from a centralized constant, inert-on-non-result-commands wart accepted — the same shape, so the two global flags read uniformly.
- **Conforms to 004 / `classifyClientError` (011, 015)**: the usage exit code and the error-class routing reuse the frozen convention and the single classification chain; 020 adds one arm (`*output.FormatError` → `UsageError`), no renumbering.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against.
- **Re: 019's flagged empty-line inconsistency** (`No roles.` / `No actions.` / `no projects`): 020 **does not normalize it**. 019 raised 020 as a possible place to harmonize, but the spec scopes 020 to selection only and assigns empty-line wording to the renderer; normalizing here would change `full` output and edit sibling reads' golden tests, outside 020's selection scope. Left to its owning specs / a dedicated cleanup.
