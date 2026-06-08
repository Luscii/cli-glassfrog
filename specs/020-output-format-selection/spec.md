# Specification: Output Format Selection

**Feature**: 020-output-format-selection
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Output Format Selection is the **selector and router** of the Output Formatting cluster. It introduces the `--output` flag that picks one of four formats per invocation — **`full`**, **`compact`**, **`json`**, **`yaml`** — resolves the effective format from a fixed precedence chain, and dispatches a command's successful result to the renderer matching that format. It is what finally makes `compact`, `json`, and `yaml` reachable from the command line: Structured Serialization (018) built the JSON/YAML encoders and Templated Human Rendering (019) built the `full`/`compact` templates, but both deliberately deferred *which* format is active to this capability. 020 owns selection and success dispatch; it owns no renderer of its own.

It resolves one effective format from the same precedence shape Base URL Resolution (008) uses — command flag, then environment variable, then config file, then a built-in default — reusing the same `.glassfrogrc` file and nearest-wins walk-up that Credential Discovery (005) and Base URL Resolution (008) already walk, reading its **own** output-format key independently of the token and base-URL keys. Like base-URL resolution, it always yields a format because a built-in default sits at the end of the chain.

It deliberately does **not** render anything itself (018 and 019 own the four renderers), does **not** define the process exit code beyond emitting the conventional usage code for an invalid selector (004 owns exit codes), and does **not** render command *failures* in the selected format — that is Output-Aware Failure Rendering (032), which consumes the format 020 selects. The one failure 020 raises itself is the fail-fast **invalid-selector usage error**, the case 018 explicitly handed to this capability.

---

## Behavioral Accord

### Resolution

- When a command that produces result data runs, the system resolves one effective output format from a fixed precedence order: the `--output` flag, then the output-format environment variable, then the output-format value in the config file, then the built-in default.
- When the flag holds a value, the system uses it and consults no other source.
- When the flag is absent and the environment variable holds a value, the system uses it and does not read any config file.
- When neither the flag nor the environment variable yields a value, the system searches the config file along the same path Base URL Resolution (008) and Credential Discovery (005) walk — the current working directory ascending to the filesystem root, then the home-directory config file as a de-duplicated final fallback — and the nearest file that yields an output-format value wins (nearest-wins).
- When no flag, environment variable, or config file yields a value, the system uses the built-in default, **`full`**, preserving today's standing projection (019). Selection always yields a format — there is no "no format" outcome.
- When the `--output` flag is supplied anywhere on the command path — before or after the subcommand — the system resolves the same effective format; flag position does not change the result.

### Valid formats

- The four valid formats are `full`, `compact`, `json`, and `yaml`, and only these four.
- When a source provides a format value, the system matches it case-insensitively, so `json`, `JSON`, `Json`, and `jSON` all select the JSON format; only the four token names are valid, in any casing.

### Dispatch

- When a format is resolved and the command produces a successful result, the system routes that result to the renderer matching the format: `full` and `compact` to the human templates (019), `json` and `yaml` to the structured encoders (018). The command no longer selects its own output.
- When the same successful result is rendered under different formats, only the rendering differs; the system does not change which fields the command fetched or the result data it produced — selection shapes presentation, not the request.

### Invalid selector

- When a source provides a non-empty value that is not one of the four valid format names (for example `--output=xml`), the system reports a usage error naming the offending value, makes no API request, and runs no command — it fails fast.
- When an invalid value comes from a lower-precedence source (the environment variable or the config file) while higher-precedence sources are absent, the system reports the same usage error naming the source the value came from, rather than silently falling through to the next source — a present-but-invalid value surfaces loudly, mirroring how Base URL Resolution (008) treats a present-but-malformed value.
- When the system reports an invalid-selector usage error, it exits with the conventional usage exit code defined by Exit-Code Convention (004); it defines no new exit code of its own.

---

## User Scenarios

**In order to** parse a command's full result with my tooling,
**as an** AI agent operating the CLI,
**I want to** select `--output json` (or `yaml`) per invocation and receive that format.

**In order to** always get machine-readable output without passing a flag on every call,
**as an** AI agent with a fixed pipeline,
**I want to** set the output format once via an environment variable or config file and have every command honor it.

**In order to** scan a long list quickly when reading as a human,
**as a** practitioner triaging governance,
**I want to** select the `compact` rendering that 019 built but the CLI did not yet expose.

---

## Non-Behaviors

- The system must not implement the JSON/YAML encoders or the `full`/`compact` templates. **Why**: Structured Serialization (018) and Templated Human Rendering (019) own the four renderers; 020 is the selector and router that dispatches to them. Re-implementing rendering here would fork the output contracts across capabilities.
- The system must not render command failures (transport errors, API errors) in the selected format. **Why**: Output-Aware Failure Rendering (032) owns failure-content rendering and consumes the format 020 selects; the only failure 020 raises itself is the invalid-selector usage error. Until 032 lands, failures keep today's cause-plus-next-step form even under `json`/`yaml` (see Assumptions) — a documented interim gap, not 020's job to close.
- The system must not define any process exit code beyond emitting the conventional usage code for an invalid selector. **Why**: Exit-Code Convention (004) owns exit codes; output format is otherwise orthogonal to the success/failure signal — an agent reads the rendered output and the exit code as complementary outputs.
- The system must not write, create, or modify any config file. **Why**: it reads its own output-format key only; Credential Storage (006) owns writing `.glassfrogrc`. Two writers of one file would split the file contract.
- The system must not own the `.glassfrogrc` location, walk-up, or parse mechanics. **Why**: Credential Discovery (005) and Base URL Resolution (008) own file discovery; 020 reuses that walk to read its own key, independently of the token and base-URL keys.
- The system must not support multiple formats per invocation, per-command format overrides, profiles, or per-format options. **Why**: one effective format per invocation is the whole need; multiplexing would add structure with no consumer.
- The system must not change which fields a command fetches from the API or the result data it produces. **Why**: selection is downstream of the read — it shapes presentation, not the request, so the fetched data is identical across formats.
- The system must not make an API call or probe anything to resolve the format. **Why**: resolution must be offline and deterministic, exactly as Base URL Resolution (008) requires.

---

## Integration Boundaries

- **Structured Serialization (018 — downstream renderer)**: provides the JSON and YAML encoders and the uniform-format / unified-error-envelope contract. 020 selects `json` or `yaml` and routes the successful result through the matching encoder. 018 explicitly delegated the invalid-selector case to 020; 020 owns it.
- **Templated Human Rendering (019 — downstream renderer)**: provides the `full` and `compact` templates. 020 selects among them and dispatches the result; 020 is what first makes `compact` reachable from the command line (019 built it but exposed no selection surface).
- **Credential Discovery (005) / Base URL Resolution (008) — shared file & walk**: 020 reads its own output-format value from the same `.glassfrogrc`, using the same nearest-wins walk-up plus de-duplicated home-directory fallback, independently of the token key (005) and the base-URL key (008).
- **Exit-Code Convention (004)**: the invalid-selector usage error exits with 004's conventional usage code; for all other outcomes the exit code is decided independently of output format.
- **Output-Aware Failure Rendering (032 — downstream dependent)**: consumes the format 020 selects to render failures in that format (human cause-plus-next-step for `full`/`compact`; 018's structured envelope for `json`/`yaml`). 020 establishes the selection; 032 renders the failures.
- **Read commands (011–014, and future reads)**: honor the resolved format and route their result through the selected renderer. Commands that produce no result data (`login`, `help`, `version`) are unaffected by `--output`.
- **Command-line invocation / Environment / Filesystem (upstream inputs)**: the `--output` flag, the output-format environment variable, and the config file value are the three configurable sources of the precedence chain.

---

## Driving Scenarios

### Happy path

**Scenario: --output json selects the JSON encoder**
Given a command produces a successful result
And `--output json` is supplied
When the result is rendered
Then the system routes the result through Structured Serialization's (018) JSON encoder
And the output is the JSON document for that result.

**Scenario: omitting --output selects the default full template**
Given a command produces a successful result
And no `--output` flag, environment variable, or config-file value is present
When the result is rendered
Then the system resolves the built-in default `full`
And routes the result through Templated Human Rendering's (019) `full` template, preserving today's projection.

**Scenario: --output compact makes the compact rendering reachable**
Given a successful `my roles` read returning several roles
And `--output compact` is supplied
When the result is rendered
Then the system routes the result through 019's `compact` template
And each role appears on a single line — the rendering 019 built but the CLI did not previously expose.

### Error scenarios

**Scenario: an unknown format value fails fast**
Given `--output xml` is supplied
When the command is invoked
Then the system reports a usage error naming the invalid value `xml`
And makes no API request and runs no command
And exits with the conventional usage exit code (004).

**Scenario: an invalid value in a lower-precedence source surfaces loudly**
Given the `--output` flag is absent
And the output-format environment variable holds `xml`
When the command is invoked
Then the system reports a usage error naming the environment-variable source and the invalid value
And does not fall through to the config file or the default.

### Edge cases

**Scenario: format matching is case-insensitive**
Given `--output JSON` is supplied
When the result is rendered
Then the system selects the JSON format
And routes the result through the JSON encoder exactly as lowercase `json` would.

**Scenario: the flag overrides the environment variable and config file**
Given `--output json` is supplied
And the output-format environment variable holds `yaml`
And the config file holds `compact`
When the result is rendered
Then the system uses `json` and consults no other source.

**Scenario: the config file supplies the format when flag and environment are absent**
Given the `--output` flag and the environment variable are both absent
And the nearest config file on the walk-up holds `compact`
When the result is rendered
Then the system selects `compact` from that file
And renders the result through the `compact` template.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: each token dispatches to exactly its renderer**
Given the same successful result
When it is rendered once under each of `full`, `compact`, `json`, and `yaml`
Then `full` and `compact` route through 019's templates and `json` and `yaml` route through 018's encoders
And no token routes to a renderer other than its own.

**Scenario: the precedence chain resolves the first available source**
Given different formats present at the flag, the environment variable, and the config file in various combinations, plus the default at the end
When the effective format is resolved
Then the first source in flag → environment → config → default order that yields a value wins
And an absent source is skipped while a present-but-invalid source raises the usage error instead of falling through.

**Scenario: selection changes rendering only, never the fetched data**
Given a command whose result carries a fixed set of fields
When the result is rendered under each of the four formats
Then every format reflects the same underlying result data
And selection alters the encoding or template applied, not which fields the command fetched or produced.

---

## Assumptions

- **Default format is `full`** (confirmed): when no source selects a format, the CLI renders `full`, preserving the standing projection 019 established. This is a decision, not an assumption — recorded here for traceability.
- **Environment-variable name and config key** `[ASSUMED]`: the *behavior* is fixed — a dedicated output-format source sits at the environment tier and at the config tier of the chain, read independently of the token and base-URL keys. The literal environment-variable name and config-file key are interface-level details pinned in plan/interface, exactly as Base URL Resolution (008) leaves its literal names to the interface.
- **Interim failure-rendering gap**: until Output-Aware Failure Rendering (032) lands, command failures keep today's cause-plus-next-step form even when `json`/`yaml` is selected; 018's uniform-format guarantee (every output structured under a structured format) is fully realized only once 032 renders failures in the selected format. This mirrors 019's interim state, where `compact` was built but unreachable until 020 wired selection.
- **Effective format need not be surfaced**: unlike Base URL Resolution (008), which reports its resolved source because the active endpoint carries safety weight, 020 need not surface which source the format came from — the effective format is the whole need. (If a consumer later needs the source, that is an additive concern.)

---

## Ambiguity Warnings

_None remaining — the behavioral forks were resolved during the defining conversation: (1) the default when `--output` is omitted is `full` (preserving 019's standing projection); (2) selection follows the full precedence chain — flag > environment variable > config file > default — mirroring Base URL Resolution (008), with the flag accepted anywhere on the command path and the config value read from the same `.glassfrogrc` walk; (3) format matching is case-insensitive over the four valid tokens, with a short alias for the flag; (4) an invalid selector fails fast with a usage error and the conventional exit code (004), at any source, naming the offending source rather than falling through; and (5) 020 owns selection and success dispatch while Output-Aware Failure Rendering (032) owns rendering failures in the selected format. The remaining `[ASSUMED]` items (environment-variable name, config key) are interface-level naming details, not behavioral gaps._

---

## Clarifications

### Session 2026-06-08

- **Default format**: when `--output` is omitted from every source, the CLI renders `full`, preserving the projection 019 made the standing output. An agent opts into a machine format explicitly.
- **Selection mechanism**: a full precedence chain mirroring Base URL Resolution (008) — `--output` flag, then the output-format environment variable, then the config-file value (same `.glassfrogrc` and nearest-wins walk Credential Discovery and Base URL Resolution use), then the built-in default. The flag is a persistent flag accepted anywhere on the command path (root or alongside the subcommand), resolving to the same effective format.
- **Flag ergonomics**: the flag carries a short alias, and value matching is case-insensitive — `json`, `JSON`, `Json`, `jSON` all select the JSON format — while only the four token names (`full`, `compact`, `json`, `yaml`) are valid in any casing.
- **Invalid-selector handling, extended to all sources**: an invalid format value fails fast with a usage error and the conventional usage exit code (004), making no API request. This applies not only to the flag but to a present-but-invalid value in the environment variable or config file, where the error names the offending source rather than silently falling through — derived from Base URL Resolution's (008) established convention that a present-but-malformed value surfaces loudly instead of falling through to a lower-precedence source.
- **Failure-path boundary**: 020 owns selection, success dispatch, and the invalid-selector usage error; Output-Aware Failure Rendering (032), downstream, owns rendering command failures (transport, API errors) in the selected format. 020 establishes the selection 032 consumes.
