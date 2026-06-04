# Interface Accord: Credential Discovery — Specification

**Feature**: 005-credential-discovery
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: Integration Design — "Credentials file format (specification boundary, shared with Credential Storage 006)", plus the `GLASSFROG_TOKEN` environment input and the `Resolution` contract consumed by Request Authentication (007).

---

This accord pins three contracts the rest of Token Authentication depends on: the **`.glassfrogrc` file format** (read here, written by Credential Storage 006), the **`GLASSFROG_TOKEN` environment variable**, and the **`Resolution`** value Discovery hands to Request Authentication (007). All three are marked `[ASSUMED]` where the name/format is provisional and must be reconciled with Credential Storage before either ships. There is no command or entry point — Discovery is consumed by a resolver call, not invoked from the shell — so the *invocation* and *instructional* surfaces are N/A.

---

## Surface

### Configuration inputs

| Input | Form | Description |
|---|---|---|
| `GLASSFROG_TOKEN` | environment variable | `[ASSUMED]` The token, supplied directly. A value that is **non-empty after trimming** is used as-is and short-circuits all file reads. Empty, unset, or whitespace-only → ignored, file search proceeds (a blank value is treated as absent, matching the file token's usability rule). |
| `.glassfrogrc` | file (`key=value`) | `[ASSUMED]` The credentials file. Searched in the current directory's ancestry (nearest wins) and in the home directory. |

### `.glassfrogrc` structural contract `[ASSUMED]`

A UTF-8 text file of newline-separated lines:

| Line kind | Rule |
|---|---|
| `key=value` | Split on the **first** `=`; key and value are trimmed of surrounding whitespace. Unknown keys are ignored (forward-compatible). |
| `token=<value>` | The credential. A value that is empty or whitespace-only after trimming is treated as **no token present**. |
| blank line | Ignored. |
| `# comment` | A line whose first non-whitespace character is `#` is ignored. |
| any other non-blank, non-comment line **without** an `=` | Makes the file **malformed** (a parse error — see Error Communication). |

**Example `.glassfrogrc`**:
```
# glassfrog credentials
token=gf_live_abc123
```

**Search locations**, in precedence order:
1. `GLASSFROG_TOKEN` (environment)
2. `.glassfrogrc` in the current working directory, then each ancestor directory up to the filesystem root — **nearest wins**
3. `.glassfrogrc` in the home directory (consulted once; if the home directory already appeared in the walk-up chain it is not re-read)

### Output contract — `Resolution` (consumed by Request Authentication 007)

| Field | Type | Description |
|---|---|---|
| `Token` | string | The resolved credential. Present only when `Source` is `Environment` or `File`. |
| `Source` | enum | `Environment`, `File`, or `None`. |
| `Path` | string | When `Source` is `File`, the path the token was read from. Empty otherwise. |

`Source: None` means no credential was found anywhere — a **normal** outcome, not an error. `Source` and `Path` are the only parts of a `Resolution` safe to display; `Token` is a secret and must never be rendered, logged, or placed in an error.

---

## Interactions

**Resolution precedence** (the configuration-precedence rule): environment overrides files; among files, the nearest `.glassfrogrc` walking up from the current directory overrides a more distant one; the home-directory file is the final fallback. The first source that yields a **usable** token (non-empty after trimming) wins.

**Walk-up and skip semantics**:
- The search ascends from the current directory through every parent up to the filesystem root, then falls back to the home file.
- A file that **exists, parses, but has no usable `token`** is *skipped* — the search continues to the next location rather than stopping. (A present-but-tokenless nearest file does not shadow a home file that has a token.)
- A file that is **missing** at a candidate location is simply skipped (not an error).

**Determinism**: for an unchanged environment and filesystem, resolution always returns the same `Token` from the same `Source` — the precedence order is total.

---

## Error Communication

Discovery itself emits no exit code and prints nothing; it returns outcomes for Request Authentication (007) and Exit-Code Convention to map. The contract distinguishes three outcomes:

| Condition | Outcome |
|---|---|
| A usable token found | `Resolution{Token, Source, Path}`, no error |
| No source yields a usable token (env empty/unset, no file, or only tokenless files) | `Resolution{Source: None}`, **no error** — absence is normal; the consumer decides what to do |
| A candidate `.glassfrogrc` exists but cannot be **read** (e.g. permission denied) | **typed read error** naming the file path; the search does **not** silently fall through to another source |
| A candidate `.glassfrogrc` exists but cannot be **parsed** (a non-comment, non-blank line without `=`) | **typed format error** naming the file path; **not** reported as "no credentials found" |

**Secret hygiene** (constraint, enforced): no error message, log line, or other output produced anywhere in resolution contains the token value. Errors carry only the offending **path**. A broken credential fails loud (read/format error); it is never masked as absence.

---

## Consistency Notes

- **Shared with Credential Storage (006)**: the `.glassfrogrc` name and `key=value`/`token` format defined here are the same contract Storage will *write*. The read and write sides share one format module (plan ADR-1/ADR-3), so they cannot drift. Both the file name and `GLASSFROG_TOKEN` are `[ASSUMED]` — **reconcile with 006 before either capability ships**; if 006 revises the format, this accord updates with it.
- **Consumed by Request Authentication (007)**: 007 reads `Resolution`, attaches `Token` as the `X-Auth-Token` header, and reports `Source`/`Path` (never `Token`) as the operator-facing active identity — satisfying Action Transparency without leaking the secret.
- **First specification touchpoint in this project**: prior specs designed CLI touchpoints (003-help-and-version `interface-cli.md`). Discovery deliberately has **no** CLI surface (spec non-behavior), so this is a specification accord, not a CLI one — no inconsistency, just a different boundary type.
- **No `accords/` directory** exists in the project, so there are no established cross-spec accord patterns to align against; this accord stands on its own.
