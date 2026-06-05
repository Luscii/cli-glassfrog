# Specification: Base URL Resolution

**Feature**: 008-base-url-resolution
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Base URL Resolution is the first half of **Connection Resolution**, the capability that addresses the problem *Undefined Connection Settings* — the CLI doesn't know which base URL (or token) to use, or where to read it from. Every API call needs a root URL to hang request paths off of; this capability is the single place that answers "which Glassfrog endpoint are we talking to, right now, in this directory?" It resolves one effective base URL from a fixed precedence chain — command flag, environment variable, config file, built-in default — and hands that URL, together with where it came from, to the request layer.

This slice deliberately covers **only the base URL**. The other half of Connection Resolution — combining the resolved base URL with the token from **Credential Discovery** into the single connection context each request uses — is deferred to a later spec. Base URL Resolution is the sibling of Credential Discovery: it reuses the same `.glassfrogrc` file and the same nearest-wins walk-up, but resolves its own value independently. It is read-only and self-contained — it never writes a config file, never makes an API call, never resolves the token, and never decides a process exit code. Its operator is usually an AI agent acting for a practitioner, so resolution is deterministic and non-interactive, and it always yields a value because a built-in default sits at the end of the chain.

---

## Behavioral Accord

### Resolution order

- When a base URL is requested, the system resolves it from the first available source in a fixed precedence order: the command flag, then the environment variable, then the config file, then the built-in default.
- When the command flag holds a non-empty value, the system uses it and consults no other source.
- When the flag is absent or empty and the environment variable holds a non-empty value, the system uses it and does not read any config file.
- When neither the flag nor the environment variable yields a value, the system searches for a config file starting in the current working directory and ascending through each parent directory to the filesystem root, then the home-directory config file — the same file and walk path Credential Discovery uses. The nearest file that yields a usable base URL wins (nearest-wins).
- When no flag, environment variable, or config file yields a usable base URL, the system uses the built-in default base URL. Base URL Resolution always yields a value — there is no "no base URL found" outcome.

### Reading & extraction

- When a config file is located, the system reads the base URL value held under the file's base-URL key — independently of the token, which has its own key in the same file.
- When a located config file exists and parses but holds no usable base URL value, the system continues searching the remaining sources in precedence order rather than stopping at that file.
- When a base URL is resolved, the system reports it together with the source it came from — the flag, the environment variable, the path of the file it was read from, or the built-in default — so the consumer can surface the active endpoint without re-deriving it.

### Error handling

- When a resolved base URL value is non-empty but not a usable URL — where usable means an absolute URL carrying an `http` or `https` scheme, so a scheme-less host (`api.glassfrog.com`) or a non-`http` scheme (`ftp://…`) is malformed — the system reports a format error naming the source it came from, rather than passing the malformed value downstream or silently falling through to the next source.
- When a candidate config file exists but cannot be read (for example, permission denied), the system reports a read error naming the file rather than silently skipping it and falling through to a different source.
- When a located config file cannot be parsed, the system reports a format error naming the file rather than falling through to the built-in default — a broken config surfaces loudly instead of being mistaken for absence.

---

## User Scenarios

**In order to** point the CLI at a non-production Glassfrog endpoint for a single command,
**as an** operator testing against a staging environment,
**I want to** pass the endpoint as a flag that overrides any stored or default value.

**In order to** run the CLI against the real Glassfrog API without configuring anything first,
**as a** practitioner who just installed the tool,
**I want** a sensible built-in default base URL so commands work out of the box.

**In order to** use a project-specific endpoint across a whole working tree,
**as an** operator who moves between project directories,
**I want** a project-local config file's base URL to take precedence over my home-directory one.

---

## Non-Behaviors

- The system must not resolve, read, or attach the token. **Why**: that is Credential Discovery's job, and combining the two into a connection context is the deferred half of this capability. Pulling token work into this slice would entangle two independent precedence chains and drag forward deferred scope.
- The system must not make any API call or check that the resolved base URL is reachable or live. **Why**: resolution must be offline and deterministic; a network probe would make it slow and non-deterministic, and reachability is the request's concern, not resolution's.
- The system must not write, create, or modify any config file. **Why**: it is strictly a reader; Credential Storage owns writing. Two writers of the same file would split the file contract across capabilities.
- The system must not decide the process exit code or the message shown on error. **Why**: Exit-Code Convention and the consuming command own that classification. Resolution reports the outcome (resolved / error); how that maps to an exit code is decided downstream.
- The system must not normalize, rewrite, or canonicalize the resolved URL — no stripping or adding trailing slashes, no forcing a scheme. **Why**: the CLI is a faithful surface; silently rewriting an operator-supplied endpoint could mask a misconfiguration or break a deliberately unusual endpoint. The value is passed through as given (once validated as usable).
- The system must not prompt interactively for a base URL. **Why**: the operator is usually a non-interactive AI agent; resolution is deterministic and falls back to the built-in default rather than soliciting input.
- The system must not support multiple endpoints, profiles, or per-host entries in the config file. **Why**: an API key is scoped to a single org + person (PROJECT constraint), so one effective base URL is the whole need; multiplexing would add structure with no consumer.

---

## Integration Boundaries

- **Credential Discovery (sibling capability, shared file & walk)**: shares the `.glassfrogrc` file, the location convention (working-directory walk-up to root, then home-directory file), and the file format. Base URL Resolution reads its own base-URL key independently of the token key; an absent key at a location means "nothing here," and the search continues.
- **Credential Storage (sibling capability, shared contract)**: writes the `.glassfrogrc` file. If it later persists a base-URL value, the two must agree on the key name. This slice only ever reads the file.
- **Command-line invocation (upstream input)**: reads the base-URL flag, the highest-precedence source.
- **Environment (upstream input)**: reads the base-URL environment variable; an AI agent's runtime may set it to point at an endpoint without touching disk.
- **Filesystem (upstream input)**: reads config files from the current directory's ancestry and from the home directory. It tolerates missing files (skip) and surfaces unreadable or unparseable ones (error).
- **Glassfrog API v5 specification (reference)**: the built-in default base URL is the v5 API server URL declared in `spec/glassfrog-api-v5.yaml`'s `servers:` block.
- **Connection context / request layer (downstream consumer)**: receives the resolved base URL and its source and uses it as the root for request URLs. This slice produces the value; what consumes it (including pairing it with the token) is downstream.
- **Exit-Code Convention (downstream)**: an error outcome informs the exit code that accompanies a command, but the classification belongs to that capability, not this one.

---

## Driving Scenarios

### Happy path

**Scenario: Flag overrides every other source**
Given the base-URL flag is set to a non-empty, usable URL
And the environment variable and a config file also hold base URLs
When a base URL is requested
Then the system uses the flag's value
And reports the source as the flag
And consults no other source.

**Scenario: Environment variable wins over config file and default**
Given the flag is absent
And the environment variable holds a usable base URL
And a config file also holds a different base URL
When a base URL is requested
Then the system uses the environment variable's value
And does not read any config file.

**Scenario: Nearest config file wins over the home file**
Given the flag and environment variable are absent
And a config file in the current working directory holds a usable base URL
And a home-directory config file holds a different base URL
When a base URL is requested
Then the system uses the current-directory file's value (nearest-wins).

**Scenario: Built-in default when nothing is configured**
Given the flag and environment variable are absent
And no config file holds a usable base URL anywhere in the walk path
When a base URL is requested
Then the system uses the built-in default base URL
And reports the source as the built-in default.

### Error scenarios

**Scenario: A source supplies a malformed base URL**
Given the base-URL flag is set to a non-empty value that is not an `http(s)` URL (for example, a scheme-less host)
When a base URL is requested
Then the system reports a format error naming the flag as the source
And does not fall through to another source or to the default.

**Scenario: A config file exists but cannot be read**
Given the flag and environment variable are absent
And the nearest config file exists but is not readable (permission denied)
When a base URL is requested
Then the system reports a read error naming that file
And does not silently fall through to the default.

### Edge cases

**Scenario: Environment variable set but empty**
Given the flag is absent
And the base-URL environment variable is set to an empty or whitespace-only value
And a config file holds a usable base URL
When a base URL is requested
Then the system does not treat the empty environment variable as a value
And proceeds to the config-file search.

**Scenario: A config file is present but holds no base URL**
Given the flag and environment variable are absent
And the nearest config file exists and parses but contains no base-URL entry
And the home-directory config file holds a usable base URL
When a base URL is requested
Then the system continues past the file with no base-URL entry
And uses the home-directory file's value.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Resolution is deterministic**
Given an unchanged environment, filesystem, and set of flags
When a base URL is requested twice
Then both requests resolve the same base URL from the same source.

**Scenario: Resolution always yields a value**
Given the flag and environment variable are absent
And no config file holds a usable base URL
When a base URL is requested
Then the system produces the built-in default base URL
And never reports an "absent" or "not found" outcome.

**Scenario: Resolution performs no writes**
Given any starting filesystem state
When a base URL is requested
Then the filesystem is unchanged afterward — no config file is created or modified.

**Scenario: Resolution makes no network call**
Given any resolution outcome
When a base URL is requested
Then no outbound connection or API call is made during resolution.

---

## Assumptions

- **Config key name** `[ASSUMED]`: the base URL is stored in `.glassfrogrc` under a `base_url` key, alongside the `token` key that Credential Discovery reads. This is a contract shared with Credential Storage; reconcile the key name when Credential Storage gains base-URL support. (Parallels 005's `token` key.)
- **Environment variable name** `[ASSUMED]`: the base-URL environment variable is `GLASSFROG_BASE_URL`, following the conventional `<TOOL>_<SETTING>` shape and paralleling `GLASSFROG_TOKEN`. (Adjustable without changing any behavior.)
- **Flag name** `[ASSUMED]`: the command flag is `--base-url`. (Adjustable without changing any behavior.) Note: unlike Credential Discovery, which deferred its `--token` flag, the flag is in scope here because the feature model lists it as the top-precedence source for the base URL.
- **Built-in default value** `[ASSUMED]`: the default base URL is the Glassfrog API v5 server URL declared in `spec/glassfrog-api-v5.yaml`'s `servers:` block. The exact string is pinned from the spec during planning. (Not yet hardcoded anywhere in the codebase.)
- **"Usable" base URL**: a value is usable when it is non-empty, non-whitespace, and is an absolute URL carrying an `http` or `https` scheme. Whitespace-only values are treated as absent and fall through; a non-empty value that is not an `http(s)` URL (a scheme-less host, or a non-`http` scheme) is malformed and reported as a format error rather than skipped.
- **File and walk path**: the config file, its name, and the walk path (working-directory ascent to the filesystem root, then the home-directory file) are exactly those Credential Discovery uses. (Same path, per the defining conversation.)

---

## Ambiguity Warnings

_None remaining — the URL validity criterion was resolved during the 2026-06-04 clarify session (see Clarifications). The config key name, environment variable, flag name, and built-in default value are recorded as `[ASSUMED]` and must be reconciled with `spec/glassfrog-api-v5.yaml` and Credential Storage during planning; these are coordination items, not behavioral gaps in this spec._

---

## Clarifications

### Session 2026-06-04

- **URL validity criterion**: a base URL is "usable" only when it is an absolute URL carrying an `http` or `https` scheme. A scheme-less host (`api.glassfrog.com`) or a non-`http` scheme (`ftp://…`) is malformed and reported as a format error naming its source, rather than passed through or skipped. Chosen because the v5 API and the built-in default are both `https`, and the project treats the spec as the contract — so the most common typo (a pasted host with no scheme) surfaces loudly at resolution time, consistent with the loud-failure stance for unreadable/unparseable files.
