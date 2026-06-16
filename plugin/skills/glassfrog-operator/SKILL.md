---
name: glassfrog-operator
description: Operating knowledge for driving the glassfrog CLI (the command-line client for the GlassFrog Holacracy API) as an AI agent. Consult this whenever you are about to run, or have just run, a `glassfrog` command and need to know how to get machine-parseable output, page through a result set that spans more than one response, interpret a non-zero exit code, set up the `X-Auth-Token` credential, observe write-safety before changing the governance record, or find the exact flags for one specific command. Reach for it before scraping human-rendered text, guessing what an exit code means, or retrying a refused write — even when the task does not name the CLI explicitly.
---

# Driving the glassfrog CLI

This skill is operating knowledge for an agent driving the `glassfrog` CLI — the
command-line client for the GlassFrog Holacracy API. It covers the cross-cutting
things that are the same across every command: how to get output you can parse,
how to page, what an exit code means, how to authenticate, and how to write
safely. It is **not** a command catalogue — for what any single command does and
which flags it takes, ask the CLI itself (see [Per-command detail](#per-command-detail-comes-from-the-cli)).

Everything here describes behaviour the CLI already exposes. The skill adds no
command, flag, or capability of its own — if something below ever stops matching
the shipped CLI, treat that as a defect to fix here, not a difference of opinion.

## Output you can parse

By default the CLI renders output for a human to read. When you are going to
**parse** the result, do not scrape that human text — ask for a structured
format instead, so you get a stable shape rather than prose that can be reworded
at any time.

Select the format with the `--output` flag (short: `-o`). The supported tokens
are exactly: full, compact, json, yaml.

- `full` and `compact` are the two **human** projections (labelled detail vs. a
  denser one-line-per-record listing). Use these when a person will read the
  output, not when you will parse it.
- `json` and `yaml` are the two **machine-parseable** formats. They emit the
  API's response as structured data you can decode directly.

So when you need to read, say, a practitioner's roles for downstream processing,
pass `--output json` and decode the result — never parse the `full`/`compact`
human rendering:

```
glassfrog me roles --output json
```

You can also set the format once via the `GLASSFROG_OUTPUT` environment variable
or the `.glassfrogrc` `output` key; the `--output` flag wins over both. For which
formats a specific command supports and any per-command shape, consult its
`--help`.

## Pagination

A list that is larger than one API page is handled for you: list commands **page
to completion by default**, walking every page and returning the full set in a
single invocation. So in the normal case you simply get the complete list — there
is nothing extra to do, and nothing silently truncated.

When you instead want just the first page — and want to *know whether more would
have followed* — pass `--first-page`. It fetches only the first page and signals
whether further results exist, so you can detect a multi-page result without
walking all of it. Use `--per-page` to influence the page size of the walk (the
API owns the valid range).

To fetch the subsequent pages, just drop `--first-page`: the default walk gathers
every page to the end. Not every command paginates, and the exact flags live with
each command — confirm support and spelling with `glassfrog <command> --help`.

## Exit codes and how to react

Every invocation sets a process exit code you should read from `$?`. The
convention is a fixed range, 0–7. A non-zero code is the CLI telling you *what
kind* of thing went wrong, which determines the right reaction — react to the
code, don't re-parse the message:

| Code | Meaning | React by |
|---|---|---|
| `0` | Success — the command ran and its action succeeded (or a help/listing/`--version` resolved). | Proceed; parse the output. |
| `1` | Internal error — a resolved action failed unexpectedly, a panic, or an unmapped category. | Treat as a bug or a transient fault; capture the message, retry once, escalate if it persists. |
| `2` | Usage error — unknown command, unknown/missing flag, or an unexpected argument. | Fix the invocation; check `glassfrog <command> --help` for the correct form. |
| `3` | API error — a generic non-2xx response from the API. | Read the error; it is usually a bad request or a server-side condition. Adjust the request or report it. |
| `4` | Permission error — the API rejected the call as an auth/membership failure (401/403). | Check the credential (see [Credentials](#credentials)) and that the account has access. |
| `5` | Rate limited — the API rate limit was exceeded (429). | Back off and retry later. The CLI already retries 429 internally; exit `5` means it gave up — wait longer. |
| `6` | Network unavailable — the API could not be reached at the wire (connection/DNS/TLS/timeout). | Check connectivity and retry; the request never reached the API. |
| `7` | Stale write — a guarded write was refused because the resource changed since it was read (412). | Do **not** blind-retry. Re-read the resource, re-confirm the change still makes sense, then re-submit. See [Write-safety](#write-safety). |

## Credentials

The CLI authenticates to the GlassFrog API with an API key, sent as the
`X-Auth-Token` header on every request. If a command fails for lack of
authentication (exit code `4`), the credential is missing or wrong.

Set it up with the CLI's own command:

```
glassfrog auth login
```

That is the whole mechanism — there is no separate credential store to configure
for this skill. The CLI discovers the key from the credential you supply; for the
full precedence (flag, environment, config file) and the options on the login
command, see `glassfrog auth --help`. Introduce no other credential mechanism.

## Write-safety

> This section is **guidance**, not enforcement. This skill describes what safe
> writing looks like; it does not gate, confirm, or block any write — you remain
> responsible for following it. (A separate guardrail capability will enforce it
> later.)

Some commands change the governance record — governance changes flow through
**proposals**, and some operational items can be edited directly. Writes are not
reads: a mistaken write mutates shared state others rely on. So before running a
command that writes, **confirm the change is intended** rather than firing it
speculatively.

Guarded writes are checked against the version of the resource you read. If the
resource changed in between, the write is refused with a `412` (exit code `7`) —
a stale-write refusal. When that happens, **re-read the resource, re-confirm that
your change still makes sense against the new state, and only then re-submit**.
Never blindly retry the same write: the refusal means your basis for the change
is out of date, and retrying as-is risks clobbering someone else's change.

## Per-command detail comes from the CLI

This skill deliberately does **not** list commands or enumerate their flags —
that catalogue lives in the CLI itself and would only drift if duplicated here.
When you need the exact flags, arguments, or behaviour of one specific command,
ask the CLI directly:

```
glassfrog <command> --help
```

That is the single source of truth for per-command detail; trust it over any
remembered flag. (Note: the CLI exposes help through the `--help` *flag* only —
there is no `help` subcommand, so `glassfrog <command> --help`, never
`glassfrog help <command>`.)
