# Interface Accord: Credential Storage — Specification

**Feature**: 006-credential-storage
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: ADR-1 (writer joins `internal/auth`, round-trips with the shared format module), ADR-4 (line-preserving merge, atomic write, owner-only permissions), Integration Design (the `.glassfrogrc` specification boundary shared with Discovery 005).

---

This accord pins the **write side** of the shared `.glassfrogrc` contract — the file Storage produces and Discovery (005) consumes. The structural format itself (line kinds, the `token` key, comment/blank rules, search locations) is the canonical contract in **`005-credential-discovery/interface-spec.md`**; this file does not restate it. It pins only what writing adds on top: what an existing file looks like after a store, the at-rest guarantees, and what the writer refuses to do. The binding test is a **round-trip**: a token written here resolves back through 005's reader.

There is no separate entry point here — the write is invoked through the `glassfrog auth login` command pinned in `interface-cli.md`. This accord describes the file artifact and the writer's structural guarantees.

---

## Surface

### Written artifact: `.glassfrogrc`

The writer produces a file conforming to 005's `.glassfrogrc` structural contract (`key=value` lines, `token=<value>`, `#` comments, blank lines; UTF-8; split on first `=`). Reference: `005-credential-discovery/interface-spec.md`. `[ASSUMED]` name/format, jointly held with 005 — changes once, in the shared `internal/auth` constants.

**Write outcomes by prior state**:

| Prior state of target path | Result |
|---|---|
| File absent | File created containing a `token=<value>` line. |
| File present, no `token` key | A `token=<value>` line is appended; all existing lines preserved. |
| File present, has `token` key | The existing `token=` line's value is replaced in place; all other lines (including comments and ordering) preserved. |
| File present, unparseable | **No write** — a format error is reported (see Error Communication). |

**At-rest guarantees**:
- The written file's permissions are owner read/write only (`0600`) `[ASSUMED]` — POSIX permission bits; best-effort on platforms without them. The secret is never present in a more-permissive intermediate file (see Interactions — atomic write).
- The `token` value is the only credential written; no profiles, hosts, or additional token entries are introduced (single-token contract, 005).

## Interactions

**Merge model** (preserve-others): a store replaces *only* the `token` entry. Every other line — unknown `key=value` pairs, comments, blank lines, and their order — survives unchanged. A hand-edited `.glassfrogrc` keeps its comments after a re-store.

**Parse-validate before write**: the existing file is validated through the shared reader first. A file that parses (even with no `token` key) proceeds to the line-preserving rewrite; a file that does not parse aborts with a format error and **no write**.

**Atomic write** (invocation-to-output): the new content is written to a temporary file in the **same directory** as the target, given owner-only permissions before any token bytes are written, then renamed over the target. A failure at any point leaves the original file (or its absence) unchanged — a store is all-or-nothing.

**Round-trip contract**: a token written by this writer, at the home or current-directory location, must be the exact token 005's resolver returns when reading that location. This is the contract that keeps the read and write sides from drifting; it is the writer's primary acceptance test.

## Error Communication

| Condition | Behavior |
|---|---|
| Existing target file is unparseable (a non-comment, non-blank line without `=`) | Report a format error naming the path; do not overwrite or discard the file's contents. |
| Target path not writable (permission denied, unwritable directory) | Report a write error naming the path; leave the filesystem unchanged (no temp file left behind). |
| Supplied token is empty / whitespace-only | Rejected upstream by the command as not a usable token; the writer is not invoked. |

No error message, log line, or diagnostic contains the token value — only paths. (Secret hygiene, mirroring 005; CONSTITUTION II/III.)

## Consistency Notes

- **Canonical format lives in 005**: this accord references `005-credential-discovery/interface-spec.md` rather than duplicating the line-kind table. The two are one jointly-held `[ASSUMED]` contract; the shared `internal/auth` format module is the single implementation (plan ADR-1).
- **Command surface** is the sibling `interface-cli.md` (`glassfrog auth login`); this file is the artifact/structural side of the same feature.
- **Round-trip with the reader** is the cross-capability invariant — pinned here, tested as the writer's acceptance criterion, and noted in 005's reconciliation flag.
- No `accords/` directory exists in the project; there are no prior specification-accord patterns to align against beyond 005's sibling file.
