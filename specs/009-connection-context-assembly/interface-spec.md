# Interface Accord: Connection Context Assembly — Specification

**Feature**: 009-connection-context-assembly
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: System Architecture + ADR-1/ADR-2 — the `internal/apiclient` `ConnectionContext` value object and the `Assemble` / `AssembleFromOS` aggregator that pairs 008's `BaseURL` outcome with 005's `auth.Resolution` outcome, consumed by Request Execution (010).

---

This accord pins the Go API surface of the connection-context half of Connection Configuration: the **`ConnectionContext`** value the aggregator produces, the **`Assemble`** pure function and its **`AssembleFromOS`** production seam, and the **readiness** contract consumers read. There is **no command and no entry point** in this slice — assembly is a function call, and the `--base-url` flag's cobra registration belongs with the future command that triggers API calls — so the *invocation* and *instructional* surfaces are N/A. The capability has **no configuration of its own**: the flag value threads through `AssembleFromOS`, and the env/file inputs are owned by 008 (`GLASSFROG_BASE_URL`, `.glassfrogrc base_url`) and 005 (`GLASSFROG_TOKEN`, `.glassfrogrc token`); this accord introduces no new configuration keys.

---

## Surface

### Entry points

| Function | Signature (shape) | Description |
|---|---|---|
| `Assemble` | `(resolveBaseURL func() (BaseURL, error), resolveCred func() (auth.Resolution, error)) ConnectionContext` | The pure aggregator. Calls **both** resolvers exactly once, packs each `(value, error)` outcome verbatim, and returns a `ConnectionContext`. **No `error` return** — assembly always yields a context. Both args are required and must be non-nil; a nil resolver is a wiring bug and panics (fail-fast, no nil-default — DECISIONS/PR #20). |
| `AssembleFromOS` | `(flagValue string) ConnectionContext` | The thin production seam: binds `resolveBaseURL` to `func() (BaseURL, error) { return ResolveBaseURLFromOS(flagValue) }` and `resolveCred` to `auth.Resolve`, then delegates to `Assemble`. The `--base-url` flag *value* is an input now; its cobra registration is deferred to the consuming command. Intended to be called **once per invocation** (resolve-once; see Interactions). |

### Output contract — `ConnectionContext`

The single bundle pairing the resolved endpoint with the resolved identity. It carries a secret (the token, inside `Cred`) and therefore renders redacted (see Error Communication / Consistency).

| Field | Type | Description |
|---|---|---|
| `BaseURL` | `BaseURL` (008) | The resolved base URL outcome (`Value`, `Source`, `Path`). Meaningful when `BaseURLErr` is nil. |
| `BaseURLErr` | `error` | The typed base-URL failure carried verbatim from 008 — `*BaseURLError` (malformed value, names the source) or an `internal/rcfile` read/format error (names the file). `nil` on success. Exactly one of {`BaseURL` valid, `BaseURLErr` set}. |
| `Cred` | `auth.Resolution` (005) | The resolved credential outcome (`Token`, `Source`, `Path`). `Source == auth.SourceNone` means **no credential found** (absence, not an error). The `Token` is the secret — read it through the field, never through formatting. |
| `CredErr` | `error` | The typed credential failure carried verbatim from 005 — an `internal/rcfile` read/format error naming the file (path-only). `nil` on success or absence. |

### Readiness accessors

| Method | Shape | Contract |
|---|---|---|
| `Complete()` | `() bool` | `true` **iff** `BaseURLErr == nil` **and** `CredErr == nil` **and** `Cred.Source != auth.SourceNone` — a usable base URL **and** a present token, no errors. |
| `Problems()` | `() []string` | Safe-to-display labels for each missing or errored part, in a stable order (base URL first, then credential); **empty when `Complete()`**. Each label is built only from safe sources — the base-URL error's source label / file path, and the credential `Source`/`Path` (path-only by the 005/007 contract) or a fixed "no credentials found" phrase. **Never contains the token.** (Exact strings are a build detail; the contract is: one entry per incomplete part, naming it, secret-free.) |
| `String()` | `() string` | Value-receiver, **redacting**: renders the base-URL source (or its error label) and the credential `Source`/`Path` and the readiness, reporting the token as present/absent — never verbatim — so `%v`/`%+v`/`%s` cannot leak it. |

**Example (shape, not literal output)**:
```
complete:   ConnectionContext{ baseURL: file (/home/me/.glassfrogrc), credential: file (/home/me/.glassfrogrc), token: <redacted>, complete }
incomplete: ConnectionContext{ baseURL: default, credential: none, token: <none>, incomplete: [credentials: no credentials found] }
both wrong: ConnectionContext{ baseURL: error (--base-url not a valid absolute http(s) URL), credential: error (/etc/.glassfrogrc could not be read), incomplete: [base URL: …, credentials: …] }
```

---

## Interactions

**Assembly flow**: `Assemble` invokes `resolveBaseURL` and `resolveCred` — **both, always, exactly once each** — and never short-circuits on the first error (carry-both). It captures each outcome into the matching field pair and returns the context. Assembly itself reads no flag, environment variable, or file directly and makes no network call or write — all I/O lives inside the injected resolvers (008/005).

**Readiness is derived, reported, never decided**: `Complete()` / `Problems()` compute from the carried fields on demand. A missing or broken base URL or credential makes the context *incomplete* and is named in `Problems()`; it is **not** a reason to refuse a request, fabricate a value, or pick an exit code. The refuse-to-call fail-safe stays in Request Authentication (007), applied at request time.

**Single resolution point (resolve-once)**: both walks happen once, at assembly. Request Execution (010) builds 007's `AuthTransport` with a resolver that **replays** the context's cached credential — `func() (auth.Resolution, error) { return ctx.Cred, ctx.CredErr }` — rather than re-walking Discovery (ADR-2). So `AuthTransport.authorize` still yields the token on the present branch and the typed `AuthError` (`NoCredentials` / `CredentialError`) on the absent/errored branches, the same identity and endpoint apply to every call by construction, and the two walks cannot drift. The command layer calls `AssembleFromOS` once and threads the context; `AuthTransport`'s `sync.Once` backstops the request layer.

**Determinism**: for an unchanged pair of resolver outcomes, `Assemble` returns the same context with the same field values and the same readiness.

---

## Error Communication

`Assemble` **never returns an error and never panics on a resolver failure** — a resolver error is data, captured into `BaseURLErr` / `CredErr` and surfaced through `Complete()` / `Problems()`. (The only panic path is a nil resolver argument — a wiring bug, fail-fast per DECISIONS/PR #20.)

| Condition | Outcome |
|---|---|
| Both parts resolve | `ConnectionContext` with `BaseURLErr == nil`, `CredErr == nil`, `Cred.Source != None`; `Complete() == true`; `Problems()` empty. |
| Credential absent (`auth.SourceNone`, no error) | Context carries the base URL and `Cred.Source == None`; `Complete() == false`; `Problems()` names the missing credential. No refusal, no fabricated token. |
| Base-URL error (malformed value, or unreadable/unparseable file) | `BaseURLErr` carries 008's typed error verbatim (source label or file path); `Complete() == false`; `Problems()` names the base-URL part. |
| Credential error (unreadable/unparseable `.glassfrogrc`) | `CredErr` carries 005's typed read/format error (path-only); `Complete() == false`; `Problems()` names the credential part. |
| **Both** parts report a problem | Both `BaseURLErr` and `CredErr`/absence are carried; `Problems()` lists **both** — no short-circuit on the first. |

**No secret anywhere**: `Problems()`, `String()`, and every carried error are token-free — base-URL errors are source/path-only, credential errors are path-only by the 005/007 contract, and the token is rendered only as present/absent. **Fail loud stays at the producers** (008's malformed-URL `BaseURLError`, 005/rcfile's read/format errors); this slice neither swallows nor re-wraps them.

**Open downstream gap (not resolved here)**: 004's frozen convention has no dedicated "cannot connect / bad configuration" code (code 4 is *API*-side rejection). Which exit code an incomplete context maps to is the consuming command's decision — the same gap 007 and 008 flagged. This accord surfaces only the code-free readiness view.

---

## Consistency Notes

- **Mirrors the 005/008 code-free outcome shape**: `ConnectionContext` aggregates `auth.Resolution{Token,Source,Path}` (005) and `BaseURL{Value,Source,Path}` (008) and continues the "producer returns a typed code-free outcome, consumer maps it" split (002/004/005/007/008). The deliberate addition is the *aggregate readiness* view (`Complete()` / `Problems()`), which neither sibling needed alone.
- **Carries a secret → redacting `String()`**: because `Cred` holds the token, `ConnectionContext` follows the secret-bearing-struct rule established for `auth.Resolution` (DECISIONS/LEARNINGS) — a value-receiver `String()` that redacts the token while showing the safe parts. Pinned by a format test.
- **Resolves the 007/009 `[ASSUMED]` seam**: 007's `AuthTransport` reads the credential from this context (via the replay resolver 010 wires), not from a fresh Discovery walk (ADR-2, DECISIONS). 007's code is unchanged by this slice — a replay thunk satisfies its existing injected-resolver contract.
- **Client assembly is out of scope**: building the base `http` transport rooted at `BaseURL.Value` and wrapping it in `AuthTransport` is Request Execution (010). This accord defines the *context value*; 010 builds the *client* from it (refines 008's forecast).
- **Input/output & side effects**: inputs are the two resolver funcs (or `flagValue` for the OS seam); output is the `ConnectionContext` value. `Assemble` has **no side effects** (no I/O, no writes, no network); all I/O is the injected resolvers'. No new `.glassfrogrc` key, env var, or flag is introduced.
- **Third specification touchpoint in this project**: like 005/007/008, this capability has **no** CLI command surface in this slice, so this is a specification accord, not a CLI one. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
