# Interface Accord: User-Defined Template Output — CLI

**Feature**: 035-user-defined-template-output
**Role**: Crafter
**Touchpoint**: CLI
**Plan reference**: System Architecture + ADR-1 (flag-only template recognition), ADR-3 (user-template failures → `UsageError(2)`), ADR-4 (file/stdin source, data-only).

---

This accord pins the **operator-facing surface**: the *widened value set* of 020's `-o`/`--output` flag — a template file path or `stdin` in addition to the four format tokens — what a user template puts on stdout, and how the template-source failures are reported. 035 **adds no command and no flag**: the flag, its short alias, its precedence, and its env/config tiers are all 020's, unchanged. It only widens what a *flag value* may mean, at the flag rung. The Go API (`internal/render` user-template engine, `internal/output` discriminated selection, the `internal/cli` seam + dispatch) is pinned in `interface-spec.md`.

---

## Surface

### Widened `-o` / `--output` flag value set (flag rung only)

The flag is 020's persistent root flag (`--output`, short `-o`), unchanged in registration, scope, and precedence. What changes is the value vocabulary **when the value comes from the flag**:

| Flag value | Interpretation | Owner |
|---|---|---|
| `full` \| `compact` \| `json` \| `yaml` (case-insensitive) | a built-in format | 020 (unchanged) |
| `stdin` | read a user template from piped standard input | **035** |
| any other non-empty value | a path to a user template file (relative paths resolved against the current working directory) | **035** |

- **Reserved names win**: the four format tokens **and** `stdin` are reserved. A file literally named `full`/`json`/`stdin` in the working directory is **not** selected by bare name — name it with a path, e.g. `-o ./stdin`.
- **Flag-only**: the widened vocabulary applies **only to the `-o`/`--output` flag value**. The `GLASSFROG_OUTPUT` env var and the `.glassfrogrc output` key keep 020's contract verbatim — only the four tokens are valid there; any other value is a usage error naming that source (it is **not** treated as a template path). Rationale: a template is shaped to one resource type, so persisting one across heterogeneous reads makes no sense (operator-confirmed).
- **Usage string (shape)** (widened from 020): `Output format — full | compact | json | yaml, a template file path, or "stdin" to read a template from a pipe (overrides GLASSFROG_OUTPUT, the .glassfrogrc output, and the built-in default)`.

### Affected commands

The same result-producing reads 019/020 cover; their synopsis is unchanged (the `-o <value>` slot already exists from 020):

| Command | `-o` template effect |
|---|---|
| `glassfrog me` / `me roles` / `me actions` / `me projects` | success result rendered through the user template instead of a built-in format |
| `glassfrog roles` / `role` / `tree` / `subroles` (025/026) | same — user template renders that command's result value |

Commands that produce no result data (`login`, `help`, `version`) are unaffected by `-o`, exactly as under 020.

### What a user template puts on stdout

| Selection | stdout |
|---|---|
| `-o ./tmpl` (file) | whatever the template emits when executed against the command's result value (see the per-resource Data Vocabulary in `interface-spec.md`) |
| `-o stdin` (pipe) | same — the template read from the pipe, executed against the result value |

035 defines no fixed stdout shape of its own — the operator's template is the shape. The system's only floor is anti-fabrication: a template can only project data the result carried; it cannot make the CLI emit a data value the API did not return (`missingkey=error` fails loud rather than rendering silent fake data).

**Example** (shapes, not literal output):
```
# template file
$ glassfrog me roles -o ./role-ids.tmpl
<the template's output over the MyRolesResponse value>

# piped template
$ printf '{{range .Roles}}{{.ID}}\n{{end}}' | glassfrog me roles -o stdin
<one role id per line — the template's output>

# reserved name wins over a same-named file
$ ls full            # a file named "full" exists here
$ glassfrog me -o full
<the built-in full projection — the file named "full" is NOT read>
```

---

## Interactions

- **Precedence (020, unchanged)**: `-o`/`--output` flag → `GLASSFROG_OUTPUT` → `.glassfrogrc output` (nearest-wins walk) → built-in default `full`. The flag short-circuits all lower rungs. A template source can arise **only** at the flag rung; the env/config rungs resolve to one of the four tokens or fail.
- **Accepted anywhere on the path (020, unchanged)**: `glassfrog -o ./t.tmpl me roles` and `glassfrog me roles -o ./t.tmpl` resolve identically (cobra interspersed parsing over the inherited persistent flag).
- **Resolve, read, and parse before any request (extends 020's "resolve format first")**: when the flag selects a template source, the CLI reads the source (file from disk, or the piped stdin) and **parses** the template before assembling the connection or sending — so a missing file, malformed template, or empty/un-piped stdin fails fast with no API request. The template is **executed** only after a successful response.
- **Piping**: `-o stdin` consumes standard input as the template; the read commands take no other stdin input, so there is no conflict. The pipe must carry a template — an interactive terminal (no pipe) or an empty pipe under `-o stdin` is a usage error (below).
- **Composing built-ins**: because the user template is parsed into a clone of the built-in set, it may invoke a built-in by name, e.g. `{{template "roles.full.tmpl" .}}` (interface-spec.md / ADR-2).

## Error Communication

| Condition | stdout | stderr | Exit code | Request made? |
|---|---|---|---|---|
| Template (file or stdin) renders successfully | the template's output | — | `0` (Success) | Yes |
| `-o ./nope.tmpl` — file missing / unreadable | nothing | usage error naming the file | `2` (UsageError) | No |
| `-o ./broken.tmpl` / piped template — fails to **parse** | nothing | usage error naming the source (file path or `stdin`) and the parse cause | `2` (UsageError) | No |
| `-o stdin` — no pipe (interactive TTY) or empty pipe | nothing | usage error: no template piped to standard input | `2` (UsageError) | No |
| Template parses, fails at **execution** (unguarded reference to an absent field) | nothing (buffer-then-write) | usage error naming the source and the execution cause | `2` (UsageError) | **Yes** (post-response) |
| `GLASSFROG_OUTPUT` / `.glassfrogrc output` holds a non-token value (flag absent) | nothing | 020's usage error naming the source — **not** treated as a template path | `2` (UsageError) | No |
| Read fails (transport/API/auth/decode) | nothing | existing cause-plus-next-step message (unchanged) | existing (1/3/4/5/6) | — |

- All template-source failures map to `UsageError(2)` through the shared `classifyClientError` chain — the same class as a malformed `--output` token (020) and a malformed `--base-url`. **No new exit code**; 004's convention is unchanged.
- Messages are token-free (the token is a request header, never on the render path) and name the source and a corrective next step.
- The post-response execution case is the one path that spends a request before exiting `2` — unavoidable because `text/template` field existence is only knowable at execution; the operator-input-failure-is-usage-error intent is preserved (interface-spec.md / plan Risks).

## Consistency Notes

- **Builds directly on 020's flag** (`interface-cli.md`, 020): same flag, short alias, scope, precedence, and "accepted anywhere on the path" — 035 only widens the value set at the flag rung and leaves the env/config tiers' four-token contract untouched. Reading 020 and 035 together, `-o` resolves to: a built-in format (any rung) or a user template (flag only).
- **Per-format stdout for the built-in tokens is unchanged**: `full`/`compact` → 019 interface-cli.md; `json`/`yaml` → 018 interface-spec.md. 035 adds the user-template stdout, owned by the operator's template.
- **Conforms to 004 / `classifyClientError`**: reuses the frozen usage exit code and the single classification chain; 035 adds one arm (`*render.UserTemplateError` → `UsageError`), beside 020's `*FormatError` arm — no renumbering.
- **Fail-fast mirrors 020/`validateInclude`**: resolution, source read, and parse happen before the network is touched, so a bad template never triggers a doomed request.
- **032 boundary (downstream)**: 032 (failure rendering) will decide how a *failure* renders when a user template was selected; 035 renders successes only, and command failures keep today's cause-plus-next-step form on stderr.
- **No `accords/` directory** exists, so there are no cross-spec CLI accord patterns to align against beyond the sibling interface files (018/019/020/025/026).
