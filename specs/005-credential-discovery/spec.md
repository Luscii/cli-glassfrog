# Specification: Credential Discovery

**Feature**: 005-credential-discovery
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Credential Discovery is the resolution capability of Token Authentication (problem: *Unauthenticated Access* — the CLI has no way to prove it is acting as a specific org + person, so Glassfrog can't authorize its calls). Every authenticated CLI action needs an `X-Auth-Token`; this capability is the single place that answers "what token are we operating as, right now, in this directory?" It locates and reads the credentials a practitioner stored earlier (or an AI agent supplied at runtime), resolves one token, and hands that token — together with where it came from — to **Request Authentication**, which attaches it to outgoing calls.

It is the dependency root of the auth solution: Request Authentication cannot resolve a token without it. It is read-only and self-contained — it never writes a credentials file (that is **Credential Storage**'s job, an independent sibling that writes the file this capability reads), never makes an API call, and never decides a process exit code. Its operator is usually an AI agent acting for a practitioner, so resolution is deterministic and non-interactive: the same environment and filesystem yield the same token from the same source on every run, and a missing token is reported rather than solicited.

---

## Behavioral Accord

### Resolution order

- When a token is requested, the system resolves it from the first available source in a fixed precedence order: the environment variable, then the nearest credentials file found by walking up from the current directory, then the home-directory credentials file.
- When the environment variable holds a non-empty value, the system uses it as the token and does not read any credentials file.
- When the environment variable is unset or empty, the system searches for a credentials file starting in the current working directory and ascending through each parent directory up to the filesystem root; the nearest file that yields a usable token wins (nearest-wins).
- When no usable credentials file is found anywhere in the walk-up chain, the system falls back to the credentials file in the home directory.

### Reading & extraction

- When a credentials file is located, the system reads the token value held under the file's token key.
- When a located file exists and parses but holds no usable token, the system continues searching the remaining sources in precedence order rather than stopping at that file.
- When a token is resolved, the system reports the token together with the source it came from — the environment variable, or the path of the file it was read from — so the consumer can surface the active identity without re-deriving it.

### Absence

- When no source yields a usable token, the system reports that no credentials were found. It does not invent a token, prompt for one, or treat absence as an error of its own — the consuming command decides what to do about it.

### Error handling

- When a candidate credentials file exists but cannot be read (for example, permission denied), the system reports a read error naming the file rather than silently skipping it and falling through to a different source.
- When a located credentials file cannot be parsed, the system reports a format error naming the file rather than reporting "no credentials found" — a broken credential surfaces loudly instead of being mistaken for absence.

---

## User Scenarios

**In order to** run CLI commands without re-supplying my token on every invocation,
**as a** practitioner who stored credentials once,
**I want** the CLI to find and use my stored token automatically.

**In order to** operate the CLI in automation without persisting a file to disk,
**as an** AI agent,
**I want** to supply the token through an environment variable that overrides any stored file.

**In order to** use a different token while working inside a particular project,
**as an** operator who moves between project directories,
**I want** a project-local credentials file to take precedence over my home-directory one.

---

## Non-Behaviors

- The system must not write, create, or modify any credentials file. **Why**: Discovery is strictly a reader; Credential Storage owns writing. Two writers of the same file would split the file contract across two capabilities and let each drift from the other.
- The system must not attach the token to requests or make any API call. **Why**: Request Authentication owns header attachment and the network boundary. Discovery resolves the value only; coupling the transport here would entangle resolution with request handling.
- The system must not decide the process exit code or the message shown when no token is found. **Why**: Exit-Code Convention and the consuming command own that classification. Discovery reports the outcome (resolved / absent / error); how that maps to an exit code is decided downstream.
- The system must not print, log, or otherwise expose the token value in plaintext output. **Why**: the token is a secret; emitting it risks leaking credentials into terminals, logs, or CI transcripts.
- The system must not prompt interactively for a token. **Why**: the operator is usually a non-interactive AI agent; blocking on a prompt would hang automation. Missing credentials are reported, not solicited.
- The system must not support multiple tokens, profiles, or per-host entries in the file. **Why**: an API key is scoped to a single org + person (PROJECT constraint), so a single resolved token is the whole need; multiplexing would add structure with no consumer.
- The system must not accept a token via a command-line flag as part of this slice. **Why**: the environment variable plus the credentials file cover the need; a `--token` flag is a possible later addition, and adding it now would widen the resolution surface before any command requires it.

---

## Integration Boundaries

- **Credential Storage (sibling capability, shared contract)**: writes the credentials file that Discovery reads. The two share the file name, the location convention (working-directory and home-directory placement), and the file format. Discovery only ever reads it; if the file is absent, Discovery treats that location as "nothing here" and continues.
- **Request Authentication (downstream consumer)**: receives the resolved token and its source, or the "no credentials" / error outcome, and uses it to attach the `X-Auth-Token` header. Discovery produces the value; Request Authentication owns what happens next.
- **Environment (upstream input)**: Discovery reads the credential environment variable; an AI agent's runtime may set it to inject a token without touching disk.
- **Filesystem (upstream input)**: Discovery reads files from the current directory's ancestry and from the home directory. It must tolerate missing files (skip) and surface unreadable or unparseable ones (error).
- **Exit-Code Convention (downstream)**: the exit code that ultimately accompanies a command reflects the discovery outcome, but the classification belongs to that capability, not this one.

---

## Driving Scenarios

### Happy path

**Scenario: Environment variable overrides any stored file**
Given the credential environment variable is set to a non-empty token
And a home-directory credentials file also holds a token
When a token is requested
Then the system uses the environment variable's token
And reports the source as the environment variable
And does not read any credentials file.

**Scenario: Nearest credentials file wins over the home file**
Given a credentials file in the current working directory holds a token
And a home-directory credentials file holds a different token
And the environment variable is unset
When a token is requested
Then the system uses the current-directory file's token (nearest-wins).

**Scenario: Walk-up finds an ancestor's credentials file**
Given the environment variable is unset
And there is no credentials file in the current working directory
And a credentials file two directories up holds a token
And there is no credentials file in the home directory
When a token is requested from the current directory
Then the system uses the ancestor file's token.

**Scenario: Home-directory file as the final fallback**
Given the environment variable is unset
And no credentials file exists anywhere in the current directory's ancestry
And the home-directory credentials file holds a token
When a token is requested
Then the system uses the home-directory file's token.

### Error scenarios

**Scenario: A credentials file exists but cannot be read**
Given the nearest credentials file exists but is not readable (permission denied)
When a token is requested
Then the system reports a read error naming that file
And does not silently fall through to another source.

**Scenario: A credentials file cannot be parsed**
Given the nearest credentials file exists but cannot be parsed
When a token is requested
Then the system reports a format error naming that file
And does not report "no credentials found".

### Edge cases

**Scenario: No credentials anywhere**
Given the environment variable is unset
And no credentials file exists in the current directory's ancestry or the home directory
When a token is requested
Then the system reports that no credentials were found
And does not fabricate a token or raise an error of its own.

**Scenario: Environment variable set but empty**
Given the credential environment variable is set to an empty value
And a credentials file holds a token
When a token is requested
Then the system does not treat the empty environment variable as a token
And proceeds to the credentials-file search.

**Scenario: A file is present but holds no token**
Given the nearest credentials file exists and parses but contains no token entry
And the home-directory credentials file holds a token
When a token is requested
Then the system continues past the tokenless file
And uses the home-directory file's token.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Resolution is deterministic**
Given an unchanged environment and filesystem
When a token is requested twice
Then both requests resolve the same token from the same source.

**Scenario: The token value never appears in produced output**
Given any resolution outcome (resolved, absent, read error, or format error)
When the capability's output and diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: Discovery performs no writes**
Given any starting filesystem state
When a token is requested
Then the filesystem is unchanged afterward — no credentials file is created or modified.

---

## Assumptions

- **Credentials file name and format** `[ASSUMED]`: the file is named `.glassfrogrc` and uses npmrc-style `key=value` text with the token stored under a `token` key. This is a contract shared with Credential Storage, which writes the same file. The developer delegated the choice; it was made to keep the npmrc parallel the feature description invoked. (Surface for confirmation when Credential Storage is specified so both capabilities agree on one contract.)
- **Environment variable name** `[ASSUMED]`: the credential environment variable is `GLASSFROG_TOKEN`, following the conventional `<TOOL>_TOKEN` shape. (Adjustable without changing any behavior.)
- **Home directory**: the operating user's home directory as the platform defines it is the location of the home-directory credentials file. (Standard for a config/credentials dotfile.)
- **Walk-up ceiling**: the ancestor search stops at the filesystem root. (Matches npm/git walk-up behavior, which the nearest-wins model was based on.)
- **Usable token definition**: a token is "usable" when its source provides a non-empty value under the token key; whitespace-only values are treated as absent. (Avoids resolving an accidentally-blank entry as a real credential.)

---

## Ambiguity Warnings

_None remaining — the resolution strategy (walk-up), environment-variable precedence (env-first), single-token scope, and the shared file contract were resolved during the defining conversation. The file name/format and environment-variable name are recorded as `[ASSUMED]` and must be reconciled with Credential Storage during planning; this is a coordination item, not a behavioral gap in this spec._

---

## Clarifications

### Session 2026-06-03

- **Search strategy**: the working-directory search is a walk-up — ascend from the current directory through each parent to the filesystem root, nearest file wins — rather than checking only the current directory. Chosen to match the npmrc model the feature description invoked, so a token can apply across a whole project subtree.
- **Environment variable**: an environment variable participates in resolution and takes precedence over any file (env-first). Chosen because the usual operator is an AI agent that may inject the token at runtime without persisting a file.
- **Single token**: the credentials file holds exactly one token; no profiles, hosts, or multiple entries. Follows the single-org+person-per-key constraint.
- **File contract**: the credentials file is `.glassfrogrc` in npmrc-style `key=value` form with a `token` key (recommended and pinned at the developer's request, marked `[ASSUMED]` for reconciliation with Credential Storage).
