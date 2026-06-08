# Tasks: Output Format Selection

**Feature**: 020-output-format-selection
**Concretization**: Full context (plan + spec + interface + scenarios)
**Inputs**: plan.md, spec.md, interface-spec.md, interface-cli.md, features/unconsumable-output/output-format-selection.feature

---

## Dependency Graph

Phase 1: Selection vocabulary + resolver (`internal/output`) (2 tasks, no phase dependencies) [Shared]
Phase 2: Flag wiring + dispatch (`internal/cli`) (4 tasks, depends on Phase 1 — and on 018 + 019 being landed on main) [Shared]

6 tasks total | T001 startable immediately (depends only on `internal/rcfile`, landed) | Builder: pipeline

> **Cross-spec dependency — satisfied.** 020 is the selector/router that consumes two seams built by other specs: `internal/output` (018 Structured Serialization — `Format`, `RenderSuccess`) and `internal/render` (019 Templated Human Rendering — `Render`, `FormatFull`/`FormatCompact`, the `Resource` keys). **Both are landed on main (#49/#50)** — the four reads already render through `render.Render`, so all tasks are unblocked. Phase 1 (the `internal/output` selection vocabulary + resolver) had no upstream dependency; Phase 2's dispatch (T005/T006) consumes the now-landed seams, and T006 rewires the *post-019* read commands (which call `render.Render`, not `formatXxx`). (plan Risks; DECISIONS 2026-06-08.)
>
> Every task is `[Shared]` except the resolver: selection, the flag, the classifier arm, and the dispatch each serve all three user scenarios (select-per-invocation / set-once-via-env-or-config / reach-compact) rather than decomposing per scenario. T002 is labelled `[US2]` because the env/config rungs are the "set the format once" story.

---

## Branching Guidance

**Pipeline mode**: `spec/020-output-format-selection/base` → `spec/020-output-format-selection/task-1`, `…/task-2`, … (one task branch per T-id, merged back into the spec base).

**Parallel-spec awareness**: 020 closes the Output Formatting cluster. It is **downstream** of 018 (machine JSON/YAML) and 019 (human full/compact) — both must land before Phase 2. It is **upstream** of 032 Output-Aware Failure Rendering (consumes the selected format to render failures) and 029 User-Defined Template Output (registers caller templates into the same `internal/render` engine). Neither downstream spec is concurrent with this one.

---

## Phase 1: Selection vocabulary + resolver (`internal/output`) [Shared]

- [ ] **T001** [Shared] Add the four-format selection vocabulary and its constants to `internal/output`
  - **Scope**: In the existing `internal/output` package (created by 018), add the `OutputFormat` enum (`Full`, `Compact`, `JSON`, `YAML`) — the *selection* vocabulary, distinct from 018's machine-encoder `Format{JSON,YAML}`. Add `ParseFormat(string) (OutputFormat, error)` matching the four tokens case-insensitively (only those four valid in any casing); `(OutputFormat).IsStructured() bool` (true for JSON/YAML); `(OutputFormat).MachineFormat() (Format, bool)` mapping a structured selection to 018's `Format` (false for the human formats). Centralize the constants `FlagOutput = "output"`, `EnvVarOutput = "GLASSFROG_OUTPUT"`, the unexported `.glassfrogrc` key `"output"`, and `DefaultFormat = Full`. `internal/output` must not import `internal/render` or `internal/cli`.
  - **Acceptance criteria**:
    - `ParseFormat` returns the matching `OutputFormat` for each of `full`/`compact`/`json`/`yaml` and for mixed casing (`JSON`, `Json`, `jSON` → JSON); any other non-empty value returns a non-nil error.
    - `IsStructured()` is true for `JSON`/`YAML` and false for `Full`/`Compact`; `MachineFormat()` returns (`Format.JSON`/`Format.YAML`, true) for the structured values and (_, false) for the human values.
    - `DefaultFormat` is `Full`; the flag/env/key constants exist as the single source of truth.
    - The package still imports no `cli`/`render`; `go build ./...` and `go vet ./...` are clean.
  - **Dependencies**: None (extends 018's package surface; needs no resolver or renderer yet). Startable immediately.
  - **Plan reference**: Phase 1; ADR-1 (selection vocabulary in `internal/output`).
  - **Interface references**: interface-spec.md — Surface (`OutputFormat`, `ParseFormat`, `IsStructured`, `MachineFormat`, constants).
  - **Scenario references**: output-format-selection.feature: "An uppercase selector selects the same format", "Each format routes to exactly its renderer"

- [ ] **T002** [US2] Add `ResolveFormat` + `FormatError` + the OS seam, mirroring `apiclient.baseurl`
  - **Scope**: In `internal/output`, add `ResolveFormat` as a pure core over injected sources resolving the chain flag → env → `.glassfrogrc` `output` key → `DefaultFormat`, plus `ResolveFormatFromOS(flagValue, startDir, homeDir string) (OutputFormat, error)` binding the real `os.Getenv` and the `internal/rcfile` nearest-wins walk (the 005 inject-roots split, the `apiclient.baseurl` shape). Distinguish *absent at a source* (skip to next rung) from *present-but-invalid* (`*FormatError{Source, Value}`, naming `--output` / `GLASSFROG_OUTPUT` / the file path). Surface `internal/rcfile` read/format errors while reading the `output` key. Always yields a format when no source errors.
  - **Acceptance criteria**:
    - Flag wins over env and config; env wins over config; the nearest config file wins over the home file; all-absent yields `Full`.
    - A present-but-invalid value at the flag, the env var, or a config file returns `*FormatError` naming that source and the offending value — never a fall-through to a lower rung.
    - An unreadable/unparseable `.glassfrogrc` encountered while reading the `output` key surfaces the `internal/rcfile` read/format error naming the file.
    - The `output` key is read through the shared `internal/rcfile` reader (no parallel parser); the `token` value is never returned. Pure-core tests run over injected sources with no real `~/.glassfrogrc` read; `go build`/`vet` clean.
  - **Dependencies**: T001 (the vocabulary + constants). `internal/rcfile` is landed on main.
  - **Plan reference**: Phase 1; ADR-1 (resolver mirrors `apiclient.baseurl`), ADR-4 (present-but-invalid errors naming source).
  - **Interface references**: interface-spec.md — Surface (`ResolveFormat`, `ResolveFormatFromOS`, `FormatError`); Interactions (precedence, resolve-before-request).
  - **Scenario references**: output-format-selection.feature: "The flag overrides the environment variable and config file", "The config file supplies the format when flag and environment are absent", "An invalid environment value fails, naming its source", "An unreadable config file fails resolution as a usage error", "Resolution takes the first available source"

## Phase 2: Flag wiring + dispatch (`internal/cli`) [Shared]

- [ ] **T003** [Shared] [P] Register the persistent `--output` / `-o` flag on the root command
  - **Scope**: In `internal/cli/root.go`, register `--output` as a persistent flag on the root beside `--base-url`, with the short alias `-o`, name from `output.FlagOutput`, and a usage string naming the four values and the override order. Inherited by every command via cobra; inert on non-result commands (the accepted `--base-url` wart). No read-command edits in this task.
  - **Acceptance criteria**:
    - `glassfrog --help` (and each command's help) lists `--output`/`-o` with the four values; `glassfrog --output json me` and `glassfrog me --output json` both parse the value (inherited persistent flag, interspersed parsing).
    - The flag name is read from `output.FlagOutput` (no string literal at the registration site); `go build`/`vet` clean.
  - **Dependencies**: T001 (`output.FlagOutput`).
  - **Plan reference**: Phase 2; ADR-2 (`--output`/`-o` persistent root flag).
  - **Interface references**: interface-cli.md — Surface (new global flag).
  - **Scenario references**: output-format-selection.feature: "A json selector routes the result to the JSON encoder" (flag-side), "A compact selector renders one line per record"

- [ ] **T004** [Shared] [P] Map `*output.FormatError` to `UsageError` in `classifyClientError`
  - **Scope**: In `internal/cli` (the `classifyClientError` chain, `clienterror.go`), add an `errors.As` arm mapping `*output.FormatError` to `UsageError`, symmetric with the existing base-URL arms, so an invalid selector's category and message agree and the exit code is the conventional usage code (2). Token-free message.
  - **Acceptance criteria**:
    - `classifyClientError(*output.FormatError)` returns `UsageError`; the existing base-URL/auth/transport/response arms are unchanged.
    - No new exit code is introduced; 004's convention is intact; `go build`/`vet` clean.
  - **Dependencies**: T002 (`*output.FormatError`).
  - **Plan reference**: Phase 2; ADR-4 (invalid selector → `UsageError(2)`).
  - **Interface references**: interface-spec.md — Error Communication; interface-cli.md — Error Communication.
  - **Scenario references**: output-format-selection.feature: "An unknown selector value fails before any request", "An invalid environment value fails, naming its source"

- [ ] **T005** [Shared] [P] Add the shared generic render-dispatch in `internal/cli`
  - **Scope**: Add a generic dispatch (e.g. `renderResult[T any]`) in `internal/cli` that, given a resolved `OutputFormat`, selects the decode target and the success renderer: structured (`IsStructured()`) → decode `*json.RawMessage`, write `output.RenderSuccess(machineFmt, raw)`; human → decode `*T`, write `render.Render(resourceKey, humanFmt, v)`, mapping `OutputFormat`'s `Full`/`Compact` to `render.FormatFull`/`FormatCompact` at this one site. A render error from either path maps to `RuntimeError(1)` (buffer-then-write; nothing partial on stdout). This is the single importer of both `internal/output` and `internal/render`.
  - **Acceptance criteria**:
    - For the same result, `json`/`yaml` route through `output.RenderSuccess` and `full`/`compact` through `render.Render`; no format routes to the other renderer (the "each format routes to exactly its renderer" validation).
    - A render error from either renderer yields exit 1 with no partial stdout.
    - `internal/output` and `internal/render` do not import each other; only this dispatch imports both; `go build`/`vet` clean.
  - **Dependencies**: T001, T002; **and landed on main**: 018 `internal/output` (`Format`, `RenderSuccess`) + 019 `internal/render` (`Render`, `FormatFull`/`FormatCompact`, `Resource` keys).
  - **Plan reference**: Phase 2; ADR-3 (`cli` owns the dispatch; `output`/`render` non-importing siblings).
  - **Interface references**: interface-spec.md — Surface (render dispatch); Interactions (decode-target selection, package layering).
  - **Scenario references**: output-format-selection.feature: "Each format routes to exactly its renderer", "Selection changes rendering only, not the fetched data", "A json selector routes the result to the JSON encoder"
  - **Note**: Cross-spec — consumes the 018 + 019 seams, both now landed on main (#49/#50); no longer gated.

- [ ] **T006** [Shared] Route the four reads through the dispatch and resolve the format fail-fast
  - **Scope**: In `internal/cli/me.go`, `me_roles.go`, `me_actions.go`, `me_projects.go`, resolve the format first via `output.ResolveFormatFromOS(flagValue, …)` (reading the inherited `--output` value), reporting a resolution error to stderr and returning `UsageError` **before** connection assembly or any request (the `validateInclude` fail-fast shape); then delegate the success path to T005's dispatch with the read's `render.Resource` key and result type, replacing the post-019 hardcoded `render.Render(…, FormatFull, …)` call. Output-format resolution stays out of `AssembleFromOS` (009). Command failures keep today's cause-plus-next-step on stderr (032 owns format-aware failures).
  - **Acceptance criteria**:
    - `--output json|yaml` on any of the four reads emits the structured document; `--output compact` emits the compact rendering; omitting `--output` (and no env/config) emits `full`, byte-equivalent to today.
    - An invalid `--output` (or env/config) value exits 2 with no API request made (tripwire transport asserts no send), naming the offending source.
    - The reads' existing godog/projection suites stay green for the default `full` path; the token-never-in-output tests still pass; `go build`/`vet` clean.
  - **Dependencies**: T003, T004, T005 (and, transitively, landed 018 + 019).
  - **Plan reference**: Phase 2; ADR-3 (dispatch), ADR-4 (fail-fast resolve, independent of connection assembly).
  - **Interface references**: interface-cli.md — Surface (affected commands), Interactions (resolution precedence, order of operations), Error Communication; interface-spec.md — Interactions (resolve before request).
  - **Scenario references**: output-format-selection.feature: "A json selector routes the result to the JSON encoder", "Omitting the selector renders the default full format", "A compact selector renders one line per record", "An unknown selector value fails before any request", "An invalid environment value fails, naming its source"
