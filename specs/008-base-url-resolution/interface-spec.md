# Interface Accord: Base URL Resolution — Specification

**Feature**: 008-base-url-resolution
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: Integration Design — "`.glassfrogrc` file format (specification boundary, shared with 005/006)" plus the `--base-url` flag, the `GLASSFROG_BASE_URL` environment input, the built-in default, and the `BaseURL` result the deferred connection-context half consumes.

---

This accord pins the contracts the base-URL half of Connection Configuration exposes: the **configuration inputs** (the `--base-url` flag, the `GLASSFROG_BASE_URL` environment variable, the `.glassfrogrc` `base_url` key, and the built-in default), the **`base_url` addition to the shared `.glassfrogrc` format**, the **"usable" URL contract**, and the **`BaseURL`** value the resolver returns. The three *names* are marked `[ASSUMED]` (CLI conventions, not in the API spec) — reconcile the `base_url` key with Credential Storage (006) before this ships. The built-in **default value is pinned** to `https://glassfrog.com/api/v5` from `spec/glassfrog-api-v5.yaml`. There is no command and no entry point in this slice — resolution is a resolver call, and the `--base-url` flag's cobra *registration* belongs with the future command that triggers API calls — so the *invocation* and *instructional* surfaces are N/A.

---

## Surface

### Configuration inputs

| Input | Form | Precedence | Description |
|---|---|---|---|
| `--base-url` | command flag (value) | 1 (highest) | `[ASSUMED]` The base URL, supplied at invocation. A non-empty value short-circuits all other sources. Its cobra registration is deferred to the consuming command; the resolver accepts the value as an input now. |
| `GLASSFROG_BASE_URL` | environment variable | 2 | `[ASSUMED]` A non-empty value (after trimming) short-circuits the file search. Empty, unset, or whitespace-only → ignored, search proceeds (uniform "usable" rule). |
| `.glassfrogrc` `base_url` | file (`key=value`) | 3 | `[ASSUMED]` Read from the same file Credential Discovery (005) locates: nearest in the current directory's ancestry wins, then the home-directory file as a final fallback (consulted once — not re-read if it already appeared on the walk-up path). |
| built-in default | compile-time constant | 4 (backstop) | **`https://glassfrog.com/api/v5`** — pinned from `spec/glassfrog-api-v5.yaml` (servers `url: /api/v5` resolved against the documented host `https://glassfrog.com`). Always valid by construction; guarantees resolution always yields a value. |

### `.glassfrogrc` `base_url` key `[ASSUMED]`

This slice adds **one key** to the existing `.glassfrogrc` structural contract pinned by 005 — the format, walk-up, comment/blank handling, and `token` key are unchanged and are read through the **same shared parser** (no second reader).

| Line kind | Rule |
|---|---|
| `base_url=<value>` | The base URL. Split on the first `=`, value trimmed. A value empty or whitespace-only after trimming is treated as **no `base_url` present** (the location is skipped, like a tokenless file). |
| `token=<value>` | Unchanged (005). Read by the token resolver only; **never returned** to base-URL callers (secret hygiene). |
| other keys / blanks / `# comments` / malformed lines | Unchanged from the 005 contract. |

**Example `.glassfrogrc`**:
```
# glassfrog config — base_url overrides the built-in default
token=gf_live_abc123
base_url=https://glassfrog.com/api/v5
```
(Only full-line `#` comments are recognized — the shared 005 parser does not strip inline comments, so a trailing `# …` on the `base_url` line would become part of the value.)

**Search locations**, in precedence order (config-file rung only — the flag and env precede it, the default follows):
1. `.glassfrogrc` in the current working directory, then each ancestor up to the filesystem root — **nearest wins**
2. `.glassfrogrc` in the home directory (consulted once; not re-read if already on the walk-up path)

The walk path, nearest-wins, and home-dedupe are identical to 005's `candidateDirs`.

### "Usable" base URL contract `[ASSUMED via clarify 2026-06-04]`

A resolved value is **usable** only when it is an **absolute URL carrying an `http` or `https` scheme**.

| Value | Treatment |
|---|---|
| absolute `http(s)` URL (e.g. `https://api.glassfrog.com/...`) | usable — the source wins |
| empty / whitespace-only | absent — fall through to the next source |
| non-empty, not an absolute `http(s)` URL (scheme-less host `api.glassfrog.com`, non-`http` scheme `ftp://…`, unparseable) | **malformed** — typed error naming the source, no fall-through (see Error Communication) |

The built-in default is a known-valid constant and is **not** re-validated.

### Output contract — `BaseURL` (consumed by the deferred connection-context half)

| Field | Type | Description |
|---|---|---|
| `Value` | string | The resolved base URL. Always set on the success path (a value is always produced). Passed through as given — never normalized (no trailing-slash rewrite, no scheme coercion). |
| `Source` | enum | `Flag`, `Environment`, `File`, or `Default`. |
| `Path` | string | When `Source` is `File`, the path the value was read from. Empty otherwise. |

There is **no `None`** member — unlike 005's `Resolution`, base-URL resolution always yields a value (the default backstops the chain). `Source` and `Path` are safe to display; the value carries no secret.

---

## Interactions

**Resolution precedence**: the first source that yields a **usable** value wins — `--base-url` flag, then `GLASSFROG_BASE_URL`, then the nearest `.glassfrogrc` `base_url` walking up from the current directory (then the home file), then the built-in default. The flag overrides everything; the env overrides files and the default; among files, nearest wins; the default is the final backstop.

**Walk-up and skip semantics** (config-file rung):
- A file that **exists, parses, but has no usable `base_url`** is *skipped* — the search continues (a tokenless-but-base-URL-less nearest file does not shadow a home file that has one, nor the default).
- A **missing** file at a candidate location is skipped (not an error).
- A whitespace-only flag/env/file value is treated as absent and falls through.

**Always terminates with a value**: there is no "base URL not found" outcome — when no flag, env, or file value is usable, the default is returned with `Source: Default`.

**Determinism**: for an unchanged flag value, environment, and filesystem, resolution always returns the same `Value` from the same `Source`.

---

## Error Communication

The resolver itself emits no exit code and prints nothing; it returns a typed outcome for the consuming command and Exit-Code Convention (004) to map. The contract distinguishes:

| Condition | Outcome |
|---|---|
| A usable value found at any rung | `BaseURL{Value, Source, Path}`, no error |
| No usable flag/env/file value | `BaseURL{Value: <default>, Source: Default}`, **no error** — the default is normal, not a fallback failure |
| A non-empty source (`--base-url`, `GLASSFROG_BASE_URL`, or a file `base_url`) supplies a value that is **not an absolute `http(s)` URL** | **typed format error** (`BaseURLError`) naming the **source** (`flag`, `GLASSFROG_BASE_URL`, or the file path); the search does **not** fall through to another source or the default |
| A candidate `.glassfrogrc` exists but cannot be **read** (e.g. permission denied) | the shared reader's **typed read error** naming the file path; no silent fall-through (reuses 005's `*ReadError`) |
| A candidate `.glassfrogrc` exists but cannot be **parsed** (a non-comment, non-blank line without `=`) | the shared reader's **typed format error** naming the file path; **not** treated as "no `base_url`" (reuses 005's `*FormatError`) |

**Fail loud, not silent** (CONSTITUTION III): a malformed value or a broken file surfaces a typed error naming where it came from — never a quiet fall-through to a lower-precedence source or the default. **No secret in errors**: `BaseURLError` carries only the source label / path; the token is never in scope on a base-URL path.

**Open downstream gap (not resolved here)**: 004's frozen convention has no dedicated "bad configuration" code (code 4 is *API*-side rejection). Which exit code a malformed-base-URL outcome receives is the consuming-command spec's decision — exactly the gap 007 flagged for "cannot authenticate." This accord surfaces only the typed outcome.

---

## Consistency Notes

- **Shared with Credential Discovery (005) & Credential Storage (006)**: the `base_url` key extends the same `.glassfrogrc` contract 005 reads and 006 writes, through the **one shared parser** (plan ADR-3) — no second reader, so the sides can't drift. The key name `base_url`, `GLASSFROG_BASE_URL`, and the `--base-url` flag name are `[ASSUMED]` CLI conventions — **reconcile the file key with 006 before this ships**; if 006 revises the format, this accord updates with it. The default value is pinned (`https://glassfrog.com/api/v5`, from `spec/glassfrog-api-v5.yaml`).
- **Mirrors 005's `Resolution` shape, minus `None`**: `BaseURL{Value, Source, Path}` parallels `Resolution{Token, Source, Path}` and continues the "producer returns a code-free typed outcome, consumer maps it" split (002/004/005/007). The deliberate difference is the absent `None` member and the `Flag`/`Default` source members — base-URL resolution always yields a value.
- **Consumed by the deferred connection-context half**: that half combines `BaseURL.Value` with 005's token to assemble the base `http.Client`/transport that 007's `AuthTransport` wraps. This accord defines the value and its source; assembling the client is out of scope.
- **Secret hygiene preserved across the shared file**: although the base URL is not a secret, the file also holds the token. The reader seam this accord uses returns only `base_url`, never the token (plan ADR-3) — base-URL callers never come into possession of the secret.
- **Second specification touchpoint in this project**: like 005, Base URL Resolution has **no** CLI command surface in this slice (the `--base-url` flag registration is deferred to the consuming command), so this is a specification accord, not a CLI one. No `accords/` directory exists, so there are no cross-spec accord patterns to align against.
