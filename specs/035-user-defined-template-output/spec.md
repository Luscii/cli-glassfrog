# Specification: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

User-Defined Template Output is the capability the Output Formatting cluster was always building toward: it lets an operator render a read command's result through **their own template** instead of one of the four built-in formats. Templated Human Rendering (019) deliberately built its `render` seam so that caller-supplied templates could be admitted later "without re-opening the rendering path"; this feature is that admission. Output Format Selection (020) introduced the `-o` / `--output` flag and reserved four format tokens — `full`, `compact`, `json`, `yaml`; this feature extends only that **flag's** value interpretation so a value that is not a reserved token is treated as a user template source.

The feature serves the CLI's defining context: its operator is usually an AI agent that needs result data shaped a particular way for a downstream pipeline, or a practitioner who wants a bespoke human view. The operator supplies a template two ways: `-o <templateFile>` names a template file, and `-o stdin` reads the template from piped standard input. Reserved format names always win, so the built-in formats are never shadowed by a same-named file. The template renders the **invoked command's** successful result through the existing per-resource seam — the author writes a template against that one command's data shape. This feature owns the recognition of a template source on the flag, reading/parsing the template, and rendering the result through it; it does not own the four built-in renderers (018/019), the precedence chain or the env-var/config tiers (020), failure rendering (032), or exit codes (004).

---

## Behavioral Accord

### Template source recognition (flag only)

- When the `-o` / `--output` **flag** holds one of the four reserved format tokens (`full`, `compact`, `json`, `yaml`), the system selects that built-in renderer exactly as Output Format Selection (020) already does — reserved names win, and a file of the same name does not shadow them.
- When the flag holds the reserved value `stdin`, the system reads a template from piped standard input and renders the result through it.
- When the flag holds any other non-empty value, the system treats that value as a path to a template file, resolving a relative path against the current working directory, and renders the result through the template the file contains.
- The system extends only the **flag's** value interpretation. The environment-variable and config-file tiers of 020's precedence chain are unchanged: a non-reserved value there remains 020's invalid-selector usage error, not a template source.

### Rendering through a user template

- When a user template is selected and the invoked read command produces a successful result, the system renders that result's data through the template and prints the rendered text — the same result data the built-in renderers receive, for the command that was invoked.
- When a user template references a field the result does not carry, the system renders an explicit absence marker for that field rather than a fabricated value — it never substitutes a data value (id, name, status, real field value) the API did not return.
- The template author is responsible for how their template renders an empty or absent value; the system's only floor is that it must not invent a data value the API did not return.

### Errors

- When a named template file does not exist or cannot be read, the system reports a usage error naming the file, makes no API request, and runs no command — it fails fast, mirroring 020's invalid-selector handling.
- When the template (from a file or from stdin) cannot be parsed, the system reports a usage error naming the source, makes no API request, and runs no command — the malformed template is caught before any read.
- When `stdin` is selected but no template is piped (standard input is empty or not a pipe), the system reports a usage error and makes no API request.
- When the system reports any of these usage errors, it exits with the conventional usage exit code defined by Exit-Code Convention (004); it defines no new exit code of its own.

---

## User Scenarios

**In order to** transform a read's result into the exact shape my downstream pipeline expects,
**as an** AI agent operating the CLI,
**I want to** pass `-o <templateFile>` and receive the result rendered through my own template.

**In order to** render a result with a one-off template without writing a file to disk,
**as an** AI agent composing a command on the fly,
**I want to** pipe a template into the command and select it with `-o stdin`.

**In order to** produce a bespoke human-readable view of governance data,
**as a** practitioner with a specific reporting format in mind,
**I want to** supply my own template instead of the built-in `full` / `compact` views.

---

## Non-Behaviors

- The system must not honor a template file path or `stdin` from the environment variable or config-file tiers — template sourcing is flag-only. **Why**: a template is shaped to one resource type, so a single persisted template applied across heterogeneous reads (`me`, `my roles`, `my actions`, `my projects`) would render wrong for most of them; flag-only keeps the template tied to the one invocation it fits and leaves 020's env/config contract untouched.
- The system must not reimplement or alter the four built-in renderers. **Why**: Structured Serialization (018) and Templated Human Rendering (019) own the `full` / `compact` / `json` / `yaml` renderers; this feature adds a parallel path through the same seam, it does not fork the built-ins.
- The system must not render command *failures* (transport or API errors) through a user template. **Why**: Output-Aware Failure Rendering (032) owns failure-content rendering, and 019 already establishes that errors keep their cause-plus-next-step format rather than routing through the render seam; the only failures this feature raises are its own fail-fast usage errors.
- The system must not fabricate, default, or substitute a *data value* the API did not return. **Why**: the Constitution forbids presenting values the API did not return; a user template that references a missing field gets an explicit absence marker, never an invented governance value.
- The system must not let a template execute commands, read other files, or reach the network. **Why**: the template is a presentation surface over already-fetched data; granting it code, filesystem, or network access would turn a rendering input into an arbitrary-execution surface — the sandboxing concern 019 explicitly deferred to this feature.
- The system must not change which fields the invoked command fetches from the API. **Why**: rendering is downstream of the read — the template shapes presentation, not the request, so a field a template wants but the read did not fetch is the read's concern, not the renderer's.
- The system must not modify 020's precedence chain, default, or env-var/config resolution. **Why**: 020 owns selection; this feature only widens the meaning of a non-reserved value at the flag, so the resolution machinery and the four reserved tokens stay 020's.

---

## Integration Boundaries

- **Output Format Selection (020 — upstream selector)**: provides the `-o` / `--output` flag and the four reserved format tokens. This feature reads the resolved flag value and, when it is not a reserved token, reinterprets it as a template source. 020's env-var/config tiers and invalid-selector handling are unchanged.
- **Templated Human Rendering (019 — the render seam)**: provides the per-resource `render` seam this feature plugs a user template into. The user template is a new path through the same seam; 019's `full` / `compact` built-ins are untouched.
- **Structured Serialization (018)**: owns the `json` / `yaml` reserved renderers; selecting one of those tokens routes through 018 as before, never through a user template.
- **Read commands (011–014, and future reads)**: a user template renders the invoked command's successful result. Commands that produce no result data (`login`, `help`, `version`) are unaffected. If a read fails before producing result data, nothing is rendered through the template.
- **Filesystem / standard input (upstream inputs)**: a template file is read from the path named on the flag (relative paths from the current working directory); `stdin` reads a template from piped standard input.
- **Exit-Code Convention (004)**: the fail-fast usage errors (missing file, unparseable template, empty stdin) exit with 004's conventional usage code.

---

## Driving Scenarios

### Happy path

**Scenario: a template file renders the result**
Given a successful `my roles` read returning several roles
And `-o ./roles.tmpl` names a readable, parseable template
When the result is rendered
Then the system renders the roles' data through that template
And prints the template's output.

**Scenario: a template is read from piped stdin**
Given a template piped into the command on standard input
And `-o stdin` is supplied to a successful `me` read
When the result is rendered
Then the system reads the template from standard input
And renders the `me` result's data through it.

**Scenario: a reserved name wins over a same-named file**
Given a file named `full` exists in the current working directory
And `-o full` is supplied
When the result is rendered
Then the system selects the built-in `full` template (019)
And does not read the file named `full`.

### Error scenarios

**Scenario: a missing template file fails fast**
Given `-o ./nope.tmpl` names a file that does not exist
When the command is invoked
Then the system reports a usage error naming the file
And makes no API request and runs no command
And exits with the conventional usage exit code (004).

**Scenario: a malformed template fails fast before the read**
Given `-o ./broken.tmpl` names a file whose template cannot be parsed
When the command is invoked
Then the system reports a usage error naming the source
And makes no API request and runs no command.

### Edge cases

**Scenario: a template references a field the result does not carry**
Given a successful read whose result omits an embedded collection
And a template that references that collection
When the result is rendered
Then the system renders an explicit absence marker where the field would be
And substitutes no fabricated data value.

**Scenario: stdin selected with nothing piped**
Given `-o stdin` is supplied
And no template is piped to standard input
When the command is invoked
Then the system reports a usage error
And makes no API request.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: a template source is never honored from env or config**
Given the `-o` flag is absent
And the output-format environment variable or config file holds a non-reserved value such as a file path
When the effective format is resolved
Then the system raises 020's invalid-selector usage error for that source
And does not read or render any template file.

**Scenario: a user template introduces no value absent from the source**
Given any successful result and any user template
When the result is rendered
Then every data value shown (id, name, status, field value) traces to a field the result carried
And none is synthesized (an explicit absence marker is structural, not a data value).

**Scenario: a malformed template is caught before any API call**
Given a user template that cannot be parsed, from a file or from stdin
When the command is invoked
Then the failure is reported before any read is attempted
And no API request is made.

---

## Assumptions

- **Relative paths resolve from the current working directory**: a template-file path on the flag that is not absolute is resolved against the invocation's working directory, the standard CLI convention. (Confirmed during the defining conversation.)
- **`stdin` is a reserved flag value**: like the four format tokens, `stdin` is reserved at the flag, so a file literally named `stdin` is selected only via a path such as `./stdin`. (Follows directly from the reserved-names-win rule the operator confirmed.)
- **Template language `[ASSUMED]`**: the *behavior* is fixed — the operator supplies a template, it renders the invoked command's result data, and it has no code/file/network access. The concrete template syntax and the field vocabulary it exposes are interface/plan-level details, pinned downstream exactly as 020 leaves its literal flag/key names to its interface.

---

## Ambiguity Warnings

None — the behavioral forks (flag overload vs. separate flag, the `stdin` trigger, flag-only vs. full-precedence sourcing, and the error/sandboxing floor) were resolved during the defining conversation.
