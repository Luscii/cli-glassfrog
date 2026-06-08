# Plan: Output Format Selection

**Feature**: 020-output-format-selection
**Role**: Shaper
**Inputs**: spec.md (020-output-format-selection); PROJECT.md; CONSTITUTION.md; `.score/memory/DECISIONS.md` (relevant precedent: `--output` selector shares `internal/output` while human templating is the separate `internal/render` — 018 reconciliation 2026-06-08; `internal/output` is a pure leaf owning the JSON/YAML `Format` enum + `RenderSuccess`/`RenderError`/`ErrorEnvelope` — 018 ADR-1/ADR-4; `internal/render` owns `text/template` `full`/`compact` per result type, `Render(resource,format,data)→(string,err)`, buffer-then-write, render-error→`RuntimeError(1)` — 019 ADR-1..4; `--base-url` is a persistent root flag read by cobra inheritance, name from a centralized `apiclient.FlagBaseURL` constant — 011 ADR-2; base URL resolves on flag→`GLASSFROG_BASE_URL`→`.glassfrogrc base_url`→default and a present-but-malformed value at any rung errors naming its source — 008 ADR-2; the shared `.glassfrogrc` read/parse/walk lives in `internal/rcfile`, consumers read their own key — 005/008 + 2026-06-06 refactor; filesystem/env resolvers inject their roots and bind real OS access in a thin production seam — 005; non-2xx/usage classification is centralized in `classifyClientError`, base-URL errors map to `UsageError(2)` — 011/015); `.score/memory/LEARNINGS.md` (inject seams / fail-fast; a godog suite points at its own feature file; pure functions unit-tested; centralized constants prevent call-site/key drift); `.score/memory/DEPRECATION.md` (the 011 per-command `format<Read>` projection is superseded by 019's `internal/render` seam — 020 builds on the replacement, not the retired pattern; no entry touches selection). No SOUL.md.

---

## System Architecture

Output Format Selection is the **selector and router** that closes the Output Formatting cluster. It introduces the `--output` flag, resolves one effective format from a precedence chain, and dispatches a command's successful result to the renderer that 018 and 019 already built. It writes no encoder and no template of its own.

It threads three landed seams that were deliberately left open for it:
- `internal/output` (018) — the machine branch: `Format{JSON,YAML}`, `RenderSuccess(Format, json.RawMessage)`, `RenderError(Format, ErrorEnvelope)`.
- `internal/render` (019) — the human branch: `Render(resourceKey, renderFormat, data) (string, error)`, with `FormatFull`/`FormatCompact` per result type.
- `internal/rcfile` (005/008) — the shared `.glassfrogrc` reader and nearest-wins walk.

020 adds two things: a **four-format selection vocabulary + precedence resolver** in `internal/output` (the slot 018 reserved), and a **dispatch** in `internal/cli` — the only package that imports both `output` and `render` — that branches the decode target and the success renderer on the resolved format.

```
root command (internal/cli/root.go)
  └─ persistent flag --output / -o   ── inherited by every command (mirrors --base-url, 011 ADR-2)

read command RunE (me / my roles / my actions / my projects)
  │  reads --output flag value (cobra inheritance)
  ▼
output.ResolveFormat(flag, env, rcfile-walk)        [internal/output, NEW — mirrors apiclient.baseurl 008]
  │   flag → GLASSFROG_OUTPUT → .glassfrgrc `output` → default `full`
  │   present-but-invalid at ANY source → *output.FormatError{Source,Value}   ── fail-fast, before any request
  ▼  OutputFormat ∈ {Full, Compact, JSON, YAML}
cli render-dispatch  (internal/cli, NEW shared helper)
  ├─ structured (JSON|YAML):  decode 2xx → json.RawMessage → output.RenderSuccess(fmt, raw) → stdout
  └─ human (Full|Compact):    decode 2xx → typed struct    → render.Render(resourceKey, fmt, v) → stdout
       (failure path stays today's cause+next-step on stderr — 032 owns format-aware failures)
```

Resolution happens **before** the request because the decode target depends on the format (018 ADR-2: `json.RawMessage` when structured, the typed struct when human). It is independent of connection assembly (009) — output format is a presentation concern, not part of the connection context, so it is resolved on the render path, not inside `AssembleFromOS`.

---

## Architecture Decisions

### ADR-1: The four-format selection vocabulary and the precedence resolver live in `internal/output`, mirroring `apiclient.baseurl` resolution

**Context**: 018's reconciliation reserved `internal/output` as the home the `--output` selector would share with the JSON/YAML serializers. 020 must turn a flag/env/config value into one of four formats and resolve it on a precedence chain. PROJECT.md and 008 already establish exactly this resolution shape for `--base-url`.

**Options considered**:
1. **Selection enum + resolver in `internal/output`** — the slot 018 reserved; `OutputFormat{Full,Compact,JSON,YAML}` plus a `ResolveFormat` mirroring `apiclient.baseurl`. Keeps the whole output vocabulary in one leaf; the resolver reuses `internal/rcfile`. Downside: `internal/output` gains an env/filesystem-touching resolver (handled by the 005 inject-seam split).
2. **Resolver in `internal/apiclient` beside `baseurl.go`** — co-locates all precedence resolution. But output format is a *presentation* concern, not a connection one; folding it into the connection layer entangles unrelated concerns and contradicts 018's reservation.
3. **Resolver in `internal/cli`** — closest to the flag, but `cli` owns commands and exit-code classification, not a reusable resolver; 032 and 029 would then import command code to reach the vocabulary.

**Decision**: Option 1. `internal/output` gains `OutputFormat` (the four-valued selection enum) and `ResolveFormat`, structured exactly like `apiclient.baseurl`: a pure core over injected (flag value, env lookup, rcfile walk) plus a thin production seam that binds the real `os.Getenv`/`os.Getwd`/home and the `internal/rcfile` reader (005 inject-roots rule). The chain is `--output` flag → `GLASSFROG_OUTPUT` → `.glassfrogrc` `output` key → built-in default `full`, and it always yields a format. `OutputFormat` exposes `IsStructured()` and a mapping to 018's `output.Format` for the JSON/YAML case; the human-format mapping to `render`'s format names is `cli`'s concern (ADR-3), so `internal/output` never imports `internal/render`.

**Consequences**: One leaf owns the whole output vocabulary; `internal/output` imports `internal/rcfile` (a clean leaf→leaf edge, no cycle). The `output` rcfile key becomes the **third** `.glassfrogrc` key (after `token` 005 and `base_url` 008), read through the one shared reader — never a parallel parser. The exact symbol/type spellings and the literal env-var/key names are interface-level (`/score:interface`), as 008's were.

### ADR-2: `--output` (`-o`) is a persistent flag on the root command, read by cobra inheritance

**Context**: The spec requires the flag accepted anywhere on the command path (before or after the subcommand) and honored by every result-producing read. 011 ADR-2 set this precedent for `--base-url`.

**Options considered**:
1. **Persistent flag on root, name from a centralized `output` constant** — registered once, inherited by every current and future read via `cmd.Flags().GetString(...)`; cobra's interspersed parsing already accepts it on either side of the subcommand. Matches `--base-url` exactly. Cost: the flag appears (inert) on non-result commands (`version`, `auth`) — the same accepted wart as `--base-url`.
2. **Local flag re-registered on each read command** — avoids the inert-on-non-API-commands wart, but every current and future read must re-register, and "anywhere on the path" is harder to honor. Rejected as a divergence from 011 with no benefit.

**Decision**: Option 1. `--output` registers as a persistent flag on the root in `root.go` beside `--base-url`, with the short alias `-o`. Its name comes from a centralized constant (`output.FlagOutput`-style, mirroring `apiclient.FlagBaseURL`) so the resolver rung and the registered flag cannot drift. Result-producing reads read the value via inheritance; the inert appearance on non-result commands is accepted, exactly as for `--base-url`.

**Consequences**: Silent conformance to 011's persistent-flag precedent. `-o` is a new short alias (no existing short flag collides — `--include` is long-only). Reading the flag is a wiring-bug-only failure path (like `me.go`'s existing `GetString(FlagBaseURL)` guard).

### ADR-3: `internal/cli` owns the dispatch; a shared generic render-dispatch selects decode target and renderer, so the four reads don't duplicate the branch

**Context**: `internal/output` (machine) and `internal/render` (human) are deliberately separate leaves; neither imports the other. Only `internal/cli` imports both. Each of the four reads currently decodes its typed struct and renders one way; under 020 each must branch structured-vs-human on the decode target and the success renderer.

**Options considered**:
1. **A shared generic dispatch in `internal/cli`** — `renderResult[T](w, format, resourceKey, exec, reqCtx, req)`: when `format.IsStructured()` it decodes into `json.RawMessage` and calls `output.RenderSuccess`; otherwise it decodes into `T` and calls `render.Render(resourceKey, humanFormat, v)`. The four reads delegate; the branch lives in one place.
2. **Per-read inline branching** — each read writes its own structured/human `if`. Rejected: four near-identical branches, the kind of duplication 019's centralized keys and 016's generic `All[T]` were created to avoid.

**Decision**: Option 1. A single generic dispatch in `internal/cli` owns decode-target selection (018 ADR-2) and success routing; `internal/output` and `internal/render` stay non-importing siblings, and `cli` maps `OutputFormat`'s human values to `render`'s format names at this one site. Each read passes its resource key and result type and delegates.

**Consequences**: One place to change when a fifth format or a new read lands. The dispatch is the natural consumer of `render.Render`'s `(string, error)` and `output.RenderSuccess`'s `([]byte, error)` — both surface a render error mapped to `RuntimeError(1)` via the existing classifier (019 ADR-4 / 018's render-failure contract), so a buffered failure prints nothing and exits 1 (fail-safe, CONSTITUTION III). The exact dispatch signature and the human-format mapping are interface-level.

### ADR-4: An invalid or unreadable selector fails fast as `UsageError(2)` before any request, naming its source

**Context**: 018 explicitly handed 020 the invalid-selector case (`--output=xml`). The spec extends fail-fast to a present-but-invalid value at any source and to unreadable/unparseable `.glassfrogrc`, naming the source — the same loud-surfacing 008 applies to a malformed `base_url`.

**Options considered**:
1. **Resolve-then-validate fail-fast in the read orchestration, mapped to `UsageError`** — `ResolveFormat` returns `*output.FormatError{Source,Value}` (and surfaces `internal/rcfile` read/format errors) before assembly or request; the read reports it to stderr and returns `UsageError`, exactly as `validateInclude` rejects a bad `--include` and as base-URL errors map to `UsageError`. No wasted API call.
2. **Fall through an invalid lower-precedence value to the next source** — rejected: it silently masks a misconfiguration (a typo'd `GLASSFROG_OUTPUT` would be ignored rather than corrected), contradicting the spec and 008's present-but-malformed rule.

**Decision**: Option 1. `ResolveFormat` distinguishes *absent at a source* (skip to next) from *present-but-invalid at a source* (`*output.FormatError` naming the source and value). The read resolves the format first, before connection assembly and the request; on a format error it writes the token-free message to stderr and returns `UsageError`, which the established `Outcome`→`ExitCode` path maps to code 2 (004). `classifyClientError` gains an arm mapping `*output.FormatError` to `UsageError`, symmetric with its base-URL arms, so the category and message stay consistent.

**Consequences**: An invalid selector never reaches the network; the exit code is the conventional usage code with no new code introduced. Case-insensitive matching over the four tokens is part of parsing (`JSON`/`Json`/`jSON` → JSON); only the four token names are valid in any casing. Resolution errors join base-URL errors as the usage-class configuration failures the read surfaces before sending.

---

## Cross-cutting Concerns

**Error handling & fail-safe (CONSTITUTION III)**: Two failure surfaces. (1) Selector resolution failure → fail-fast `UsageError(2)` before any request (ADR-4). (2) A render failure from `output.RenderSuccess`/`render.Render` → the dispatch writes nothing to stdout and maps to `RuntimeError(1)` via the existing classifier (018/019 buffer-then-write contract). **Command** failures (transport, API, auth) under a structured format are *not* routed through `output.RenderError` by 020 — that is Output-Aware Failure Rendering (032). Until 032 lands, those failures keep today's cause-plus-next-step form on stderr even under `json`/`yaml`: a documented interim gap (spec Assumption), mirroring how 019 shipped `compact` built-but-unreachable until 020.

**Secret hygiene (CONSTITUTION II)**: 020 routes only response-side result data and the format value; the token is a request header, never a result field, so it cannot appear in any rendered output. The `output` rcfile key is read through the shared reader, which never returns the `token` value to non-secret callers (008's rule). The existing token-never-in-output tests continue to cover the dispatched success path.

**Configuration**: The effective format is resolved at invocation from flag/env/rcfile (ADR-1); there is no other runtime configuration. The flag name, env-var name, and rcfile key are centralized constants so call sites, the resolver rungs, and the rcfile reader cannot drift (the rcfile one-source-of-truth discipline). Per the spec, the resolved source is not surfaced to the consumer (unlike base URL, where the active endpoint carries safety weight).

**Testing (CONSTITUTION IV)**: `ResolveFormat`'s pure core is unit-tested hermetically over injected flag/env/rcfile values — every precedence and present-but-invalid branch offline, never reading the developer's real `~/.glassfrogrc` (005 inject-roots). The `cli` dispatch is exercised over a fake transport returning canned bodies, asserting that each of the four formats routes to its renderer and that the same result data feeds every format. End-to-end `--output json|yaml|compact` on a real read becomes reachable for the first time here (018/019 verified their renderers at the component level); the four reads' existing golden/projection tests gate that `full` (the default) stays byte-equivalent.

---

## Implementation Strategy

Two phases. The upstream seams (`internal/output` 018, `internal/render` 019) must be **landed on main** before Phase 2 can wire end-to-end — see Risks; this mirrors how 019 was gated on the landed reads.

**Phase 1 — Selection vocabulary + resolver (`internal/output`).** Add `OutputFormat{Full,Compact,JSON,YAML}`, case-insensitive parsing, `IsStructured()`, and the mapping to 018's `output.Format`. Implement `ResolveFormat` as a pure core over injected (flag, env, rcfile walk) plus the production OS/rcfile seam, mirroring `apiclient.baseurl`: flag → `GLASSFROG_OUTPUT` → `.glassfrogrc` `output` → default `full`; `*output.FormatError{Source,Value}` on a present-but-invalid value at any source; surface `internal/rcfile` read/format errors. Centralize the flag/env/key constants. Pure unit tests for every precedence and error branch. *(Depends on `internal/rcfile`, landed; independent of 019.)*

**Phase 2 — Flag wiring + dispatch (`internal/cli`).** Register the persistent `--output`/`-o` flag on the root (ADR-2). Add the shared generic render-dispatch (ADR-3) and route the four reads (`me`, `my roles`, `my actions`, `my projects`) through it: resolve the format first (fail-fast on error, ADR-4), branch the decode target, and call `output.RenderSuccess` (structured) or `render.Render` (human). Add the `*output.FormatError` → `UsageError` arm to `classifyClientError`. Behavioral tests over a fake transport: each format routes correctly; `full` default is byte-equivalent to today; invalid selector at flag/env/rcfile fails fast with code 2 and no request. *(Depends on Phase 1, on landed 018 `internal/output` renderers, and on landed 019 `internal/render` reads.)*

---

## Risks

- **020 is gated on both 018 and 019 landing on main** (high likelihood, medium impact): the dispatch imports `output.RenderSuccess` (018) and `render.Render` (019), and Phase 2 rewires the post-019 read commands. On this branch neither is merged yet (the reads still carry inline `formatXxx`). *Mitigation*: Phase 1 (`internal/output` vocabulary + resolver) depends on neither and can proceed; Phase 2 sequences after both merge. The tasks skill should mark the cross-spec dependency and branch accordingly.
- **The four reads each acquire a structured-vs-human branch** (medium likelihood, medium impact): if the dispatch is not shared, the branch duplicates four times and drifts. *Mitigation*: ADR-3's single generic dispatch; a registry-style test that every read routes all four formats keeps the surface uniform (the 019 exhaustiveness pattern).
- **`full` default drifts when reads are rewired through the dispatch** (low likelihood, medium impact): re-routing the default path risks changing shipped output. *Mitigation*: the reads' existing golden/projection tests assert `full` stays byte-equivalent; `full` remains the resolved default when no source selects a format.
- **An invalid value silently falls through instead of erroring** (low likelihood, medium impact): a resolver that treats present-but-invalid as absent would mask a typo'd env var or config and contradict the spec. *Mitigation*: ADR-4 distinguishes absent (skip) from present-but-invalid (error naming source), pinned by per-source error tests, mirroring 008.

---

## What This Plan Does Not Cover

- **Protocol-level surface detail** — the exact `--output` flag spelling and usage string, the literal `GLASSFROG_OUTPUT` env-var name and `.glassfrogrc` `output` key, the `OutputFormat` type/symbol spellings, the dispatch signature, and the precise usage-error wording for an invalid selector → `/score:interface` (CLI and specification boundaries).
- **Format-aware failure rendering** — rendering command failures (transport, API, auth) as 018's envelope under `json`/`yaml`, and as cause-plus-next-step under `full`/`compact` → Output-Aware Failure Rendering (032). 020 establishes the selection 032 consumes and renders only its own invalid-selector usage error.
- **Caller-supplied templates** — accepting a template file as a format → User-Defined Template Output (029), which extends the same `internal/render` engine.
- **Executable scenarios and task decomposition** → `/score:scenarios` and `/score:tasks`.
