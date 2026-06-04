# Specification: Credential Storage

**Feature**: 006-credential-storage
**Role**: Definer
**Tier**: 1 (zero setup)

---

## System Overview

Credential Storage is the *write* counterpart of Credential Discovery (005) within Token Authentication (problem: *Unauthenticated Access* — the CLI has no way to prove it is acting as a specific org + person). Discovery answers "what token are we operating as?" by reading a credentials file; this capability is the single place that *creates* that file. A practitioner (or an AI agent acting for one) supplies a token once; Credential Storage persists it to the `.glassfrogrc` file, in the location and format Discovery later reads, so subsequent invocations need no re-supplied token.

It is the independent sibling Discovery's spec names explicitly: the two share the file name, the location convention (working-directory and home-directory placement), and the file format, but each owns one direction. Storage only ever writes; it never resolves a token for use, never makes an API call, and never decides a process exit code. Because its operator is usually a non-interactive AI agent, storing is deterministic and scriptable — a token can be supplied without a terminal — while still offering an interactive path for a human storing credentials by hand.

---

## Behavioral Accord

### Token input

- When a token is supplied as a command argument, the system uses it as the token to store.
- When no argument is given and a token is piped on standard input, the system reads the token from standard input without echoing it.
- When neither an argument nor piped input is present and the credential environment variable holds a non-empty value, the system persists that value.
- When none of the above sources supplies a token and the session is interactive — that is, standard input is a terminal — the system prompts for the token without echoing the typed characters.
- When more than one source is present, the system resolves a single token in the fixed precedence order: argument, then piped standard input, then the environment variable, then an interactive prompt.
- When the session is non-interactive (standard input is not a terminal) and no source supplies a token, the system reports an error that there is no token to store — it does not prompt and does not exit as a success.

### Location selection

- When no location is specified, the system writes to the credentials file in the home directory.
- When the caller requests the current-directory location, the system writes to the credentials file in the current working directory instead.

### Writing

- When a usable token is resolved and the target location holds no credentials file, the system creates the file holding the token under the file's token key.
- When the system writes the credentials file, it sets owner-only access permissions so the secret is not world- or group-readable.
- When the write succeeds, the system reports where the credentials were stored — the path of the file written — so the operator can confirm the active identity without re-deriving it.

### Existing credentials

- When a credentials file already exists at a candidate location, the system detects the existing token rather than overwriting blindly.
- When an existing token is found and the session is interactive, the system confirms with the operator before changing it, offering to merge (replace the token entry while preserving any other keys in the file) and to choose the target location based on where existing tokens were found (the working-directory file, the home-directory file, or both).
- When an existing token is found and the session is non-interactive, the system reports an error and makes no change, unless the caller supplies an explicit overwrite signal; with that signal it merges into the target location only (the default home file, or the current-directory file when that location was requested). Choosing among working-directory, home, or both is offered only in the interactive confirmation.
- When the system merges into an existing file, it replaces only the token entry and leaves every other key in the file unchanged.

### Error handling

- When the target location cannot be written (for example, permission denied), the system reports a write error naming the location rather than reporting success, and leaves the filesystem unchanged.
- When an existing credentials file must be read to merge but cannot be parsed, the system reports a format error naming the file rather than silently overwriting it and discarding its other contents.

---

## User Scenarios

**In order to** run CLI commands later without re-supplying my token,
**as a** practitioner setting up the CLI for the first time,
**I want to** store my token once to a credentials file the CLI will find automatically.

**In order to** provision credentials inside automation without a terminal,
**as an** AI agent,
**I want to** pipe a token on standard input (or persist one already in the environment) and have it written deterministically.

**In order to** use a project-specific token while working in a particular directory,
**as an** operator who moves between projects,
**I want to** store a token into a current-directory credentials file that takes precedence over my home one.

---

## Non-Behaviors

- The system must not resolve, read, or use a stored token for any request. **Why**: Credential Discovery owns resolution and Request Authentication owns header attachment. A single writer here, a single resolver there — splitting either across two capabilities would let the file contract drift.
- The system must not make any API call to validate the token before storing it. **Why**: Discovery makes no network call either; keeping Storage offline keeps the two siblings symmetric and avoids coupling a local write to remote availability. A bad token surfaces when it is *used*, not when it is stored.
- The system must not print, log, or echo the token value in plaintext — including on input (prompt/stdin) or in the success message. **Why**: the token is a secret; emitting it risks leaking credentials into terminals, logs, or CI transcripts.
- The system must not support multiple tokens, profiles, or per-host entries in the file. **Why**: an API key is scoped to a single org + person (PROJECT constraint), so one stored token is the whole need; multiplexing would add structure with no consumer and break the contract Discovery reads.
- The system must not remove or clear stored credentials in this slice. **Why**: store and remove have different lifecycles and risk profiles; this slice delivers the write that unblocks Discovery, and removal can be added later without reshaping the file contract.
- The system must not decide the canonical process exit code for its outcome. **Why**: Exit-Code Convention (004) owns the category→code mapping. Storage reports the outcome (stored / error); how that maps to an exit code is decided downstream.

---

## Integration Boundaries

- **Credential Discovery (sibling capability, shared contract)**: reads the credentials file Storage writes. The two share the file name (`.glassfrogrc`), the location convention (working-directory and home-directory placement), and the npmrc-style format with a `token` key. Storage only ever writes; Discovery only ever reads.
- **Filesystem (downstream write target)**: Storage creates or updates a credentials file in the home directory or the current working directory, sets owner-only permissions, and must surface unwritable targets (permission denied) and unparseable existing files (format error) rather than failing silently.
- **Standard input / terminal (upstream input)**: Storage reads a token piped on standard input, or prompts for it without echo when the session is interactive and no other source supplied one.
- **Environment (upstream input)**: Storage may read the credential environment variable as one input source to persist.
- **Exit-Code Convention (004, downstream)**: the exit code that accompanies the command reflects the storage outcome, but the classification belongs to that capability, not this one.

---

## Driving Scenarios

### Happy path

**Scenario: Store a token supplied as an argument to the home file**
Given no credentials file exists in the home directory
And a token is supplied as a command argument
When the token is stored
Then the system creates the home-directory credentials file holding the token under the token key
And sets owner-only permissions on the file
And reports the path it wrote
And the token value does not appear in the output.

**Scenario: Store a token piped on standard input to the current directory**
Given the caller requests the current-directory location
And a token is piped on standard input
And no argument is supplied
When the token is stored
Then the system writes the current-directory credentials file holding the piped token.

**Scenario: Persist a token already present in the environment**
Given no argument is supplied and nothing is piped on standard input
And the credential environment variable holds a non-empty value
When the token is stored
Then the system writes that value to the home-directory credentials file.

### Error scenarios

**Scenario: Target location is not writable**
Given the target location cannot be written (permission denied)
And a usable token is supplied
When the token is stored
Then the system reports a write error naming the location
And the filesystem is left unchanged.

**Scenario: Existing credentials file cannot be parsed for a merge**
Given a credentials file already exists at the target location but cannot be parsed
And a usable token is supplied
When the token is stored
Then the system reports a format error naming the file
And does not overwrite it or discard its other contents.

**Scenario: Non-interactive session with no token supplied**
Given the session is non-interactive (standard input is not a terminal)
And no token is supplied as an argument, on standard input, or in the environment
When the token is stored
Then the system reports an error that there is no token to store
And writes nothing.

### Edge cases

**Scenario: Supplied token is blank**
Given the supplied token is empty or only whitespace
When the token is stored
Then the system rejects it as not a usable token
And writes nothing.

**Scenario: Merge preserves other keys in an existing file**
Given a credentials file already exists holding the token key and an unrelated key
And the operator confirms a merge with a new token
When the token is stored
Then the system replaces only the token entry
And leaves the unrelated key unchanged.

**Scenario: Existing token, non-interactive, no overwrite signal**
Given a credentials file already holds a token at the target location
And the session is non-interactive
And no explicit overwrite signal is given
When a new token is stored
Then the system reports an error
And leaves the existing credentials unchanged.

---

## Validation Scenarios

> These are held out from the implementing agent for independent verification.

**Scenario: Stored token round-trips through Discovery**
Given a token is stored to a location
When Credential Discovery later resolves a token from that location
Then it resolves the same token Storage wrote.

**Scenario: The token value never appears in produced output**
Given any storage outcome (stored, write error, or format error)
When the capability's output and diagnostics are inspected
Then the token value itself is never present in plaintext.

**Scenario: A stored file is not world- or group-readable**
Given a token is stored to a new credentials file
When the file's access permissions are inspected
Then only the owning user can read it.

---

## Assumptions

- **Shared file name and format**: the file is `.glassfrogrc`, npmrc-style `key=value` text with the token under a `token` key. (Pinned as the shared contract with Credential Discovery (005), which reads the same file — the two now agree on one contract.)
- **Environment variable name**: the credential environment variable is `GLASSFROG_TOKEN`, matching Discovery. (Same shared contract; adjustable without changing behavior.)
- **Owner-only permissions** `[ASSUMED]`: a newly written credentials file is created with owner-read/write-only permissions (`0600`-style) on platforms that support them. (Standard for a secret-bearing dotfile; surfaced because an operator may have a different expectation on some platforms.)
- **Usable token definition**: a token is "usable" when it is a non-empty, non-whitespace value, matching Discovery's definition. (Avoids persisting an accidentally-blank entry as a real credential.)
- **Home directory**: the operating user's home directory as the platform defines it is the location of the home-directory credentials file. (Standard for a config/credentials dotfile; matches Discovery.)

---

## Ambiguity Warnings

_None remaining — the non-interactive behavior on an existing token (error unless an explicit overwrite signal, then merge), the interactivity definition (a terminal on standard input), and the non-interactive no-token case (report "no token to store") were resolved during the clarify session below. The owner-only file permission remains recorded as `[ASSUMED]` for confirmation during planning; that is a default to confirm, not a behavioral gap._

---

## Clarifications

### Session 2026-06-04

- **Input precedence**: a single token is resolved from the first present source in the order argument → piped standard input → environment variable → interactive prompt; the prompt is used only when the session is interactive and nothing else supplied a token.
- **Write location**: the home-directory credentials file is the default; a flag targets the current-directory file instead. Chosen to mirror the two locations Discovery searches.
- **Existing file (interactive)**: when a credentials file already exists, an interactive Storage session confirms with the operator before changing it — offering to merge (replace the token entry, preserve other keys) and to choose the target location based on where existing tokens were found (working-directory / home / both).
- **Existing file (non-interactive)**: when the session is non-interactive and an existing token is present, Storage reports an error and makes no change unless the caller supplies an explicit overwrite signal; with that signal it merges into the target location only (home by default, or the current-directory file when requested). The working-directory/home/both choice is offered only in the interactive flow.
- **Interactivity & no-token case**: a session is interactive when standard input is a terminal. When the session is non-interactive and no source (argument, piped stdin, or environment variable) supplies a token, Storage reports "no token to store" as an error rather than prompting or exiting as a success.
- **Removal**: out of scope for this slice — store only; clearing credentials is deferred.
