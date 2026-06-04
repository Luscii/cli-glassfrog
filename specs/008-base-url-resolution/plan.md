# Plan: Base URL Resolution

**Feature**: 008-base-url-resolution
**Role**: Shaper
**Inputs**: spec.md (008-base-url-resolution), PROJECT.md, `.score/memory/DECISIONS.md` (relevant precedent: Go self-contained binary; `internal/auth` package + the shared `.glassfrogrc` reader and the read/write-share-one-format rule; `Resolution{Token,Source,Path}` code-free shape; "filesystem/env-dependent resolvers inject their roots"; "producer classifies a code-free outcome, consumer maps it"; 007's "Connection Configuration owns base URL/timeouts/retries" and "007 lives in the API-client package `internal/apiclient`, whichever of 007/Connection-Configuration lands first creates it"). `.score/memory/LEARNINGS.md` (relevant: a second reader of the shared file drifts — keep one parser; godog suites must point at their own feature file; the directory-at-path tripwire for fail-loud/no-read tests; usable-value rule applied uniformly across sources; nil-default-vs-fail-loud for injected seams). No SOUL.md.

**Readiness**: Must met + Should substantial (integration boundaries name specific systems, user scenarios present, assumptions flagged, error/edge scenarios beyond happy path). Strong foundation. No architectural unknowns required resolving — package placement, the resolver pattern, and the result shape are all fixed by DECISIONS precedent; this plan reuses them.

---

## System Architecture

Base URL Resolution is the **base-URL half of Connection Configuration** — the sibling capability 007 (Request Authentication) deferred and flagged as an `[ASSUMED]` seam. 007 already established (DECISIONS, PR #20) that *Connection Configuration owns the base URL, timeouts, and retries*, that 007's auth round-tripper wraps *Connection Configuration's base transport*, and that both live in `internal/apiclient` — the package 007 created. This spec adds the first piece of that owner: the resolver that answers "which Glassfrog endpoint are we talking to, right now, in this directory?" and always produces a value.

It is parallel in shape to Credential Discovery (005): a deterministic, injected-roots resolver over a fixed precedence chain that returns a small code-free result. The differences are exactly the two the spec calls out — the chain has a **command flag** at the top and a **built-in default** at the bottom (so there is no "nothing found" outcome), and the value is **validated as an absolute `http(s)` URL** rather than treated as an opaque secret. It reads the file-sourced value from the **same `.glassfrogrc` and the same nearest-wins walk** that 005 uses, via 005's one shared reader — not a second parser.

The parts:

- **Base URL resolver** (`internal/apiclient`) — given the flag value, the environment, a start directory, and a home directory, walks the precedence chain (flag → `GLASSFROG_BASE_URL` → nearest `.glassfrogrc` `base_url` up the tree, then home → built-in default) and returns a **`BaseURL{Value, Source, Path}`**, or a typed error when a non-empty source supplies a malformed URL.
- **`base_url` file read** — reuses `internal/auth`'s single `.glassfrogrc` parser and walk. `auth` exposes a network-free, secret-safe seam that returns *only* the `base_url` value (never the token) from the nearest file. No second `.glassfrogrc` reader is written (LEARNINGS: two readers of one file drift).
- **URL validation** — a resolved non-empty value is *usable* only when it is an absolute URL carrying an `http`/`https` scheme (spec clarification). Whitespace-only is absent (fall through); non-empty-but-not-`http(s)` is a typed `BaseURLError` naming the source, with no fall-through. The built-in default is a compile-time constant, valid by construction, never re-validated.
- **`BaseURL` outcome** — a code-free result mirroring 005's `Resolution`: `Value`, a `Source` enum (`Flag`, `Environment`, `File`, `Default`), and a `Path` (the file when `Source == File`, empty otherwise). It emits no exit code and prints nothing — the consuming command and Exit-Code Convention (004) own those.

```
internal/apiclient                         (created by 007; this spec adds Connection Configuration's base-URL half)

  ResolveBaseURL(flagValue, startDir, homeDir) → (BaseURL, error)
    1. flagValue non-empty?        → validate → BaseURL{Value, Source: Flag}
    2. env GLASSFROG_BASE_URL?     → validate → BaseURL{Value, Source: Environment}
    3. nearest .glassfrogrc base_url up the tree, then home (auth walk) → validate → {Source: File, Path}
    4. else                        → BaseURL{Value: defaultBaseURL, Source: Default}
      · non-empty value, not an absolute http(s) URL → BaseURLError{Source}  (no fall-through, fail loud)
      · whitespace-only / key absent / file absent   → fall through to the next source
      · always yields a value (the default backstops the chain)

  reuses ──► internal/auth: shared .glassfrogrc parser + candidateDirs walk
                exposes a network-free, token-never-returned base_url file read
```

The resolved `BaseURL` is what the *deferred* connection-context half will hand to the base `http.RoundTripper`/`http.Client` that 007's `AuthTransport` wraps. Assembling that client is **not** in this slice.

---

## Architecture Decisions

### ADR-1: Base URL Resolution is Connection Configuration; it lives in `internal/apiclient` and produces the base transport's base URL

**Context**: 007 deferred base-URL ownership to "Connection Configuration" and pinned (DECISIONS) that its auth round-tripper wraps Connection Configuration's base transport, all in `internal/apiclient` — "whichever of 007/Connection-Configuration lands first creates it." 007 landed (PR #20), so `internal/apiclient` exists. This spec is the base-URL half of that owner.

**Options considered**:
1. **`internal/apiclient`** — co-located with the transport it configures and with 007's `AuthTransport` that wraps the resulting base transport. Resolves the `[ASSUMED]` seam 007 flagged.
2. **`internal/auth`** — rejected: `auth` is the credential-file/secret concern and must stay narrow and network-free; a base URL is connection configuration, not a credential. (It still *reads* the shared file via auth — ADR-3 — but the resolution policy is not auth's.)
3. **`internal/cli`** — rejected: this slice registers no command and makes no API call (CONSTITUTION V); it doesn't belong in the command-tree layer.

**Decision**: Option 1 — the resolver lives in `internal/apiclient`, the package 007 created. The `internal/apiclient` name, `[ASSUMED]` while 007 was unbuilt, is now **fixed** (007 shipped it).

**Consequences**: Connection Configuration finally has a home and its first capability. The composition seam 007 flagged is partly settled — this slice produces `BaseURL{Value, Source}`; how that value becomes the base `http.Client`/transport that `AuthTransport` wraps is the deferred connection-context half. *Precedent-setting: Connection Configuration lives in `internal/apiclient`.*

### ADR-2: Precedence is flag → env → file → built-in default; the chain always yields a value

**Context**: The spec fixes the order (command flag > environment variable > config file > built-in default) and that resolution always yields a value. This mirrors 005's env-first walk-up but adds a flag rung on top and a default backstop at the bottom — so, unlike 005, there is no `None`/absent outcome.

**Options considered**:
1. **Ordered short-circuit chain with the default at the end** — the first usable source wins; the file rung reuses 005's `candidateDirs` walk-up + nearest-wins; the built-in default backstops the chain so a value is always produced.
2. **Layered merge across sources** (npm-style) — rejected: a single base URL has nothing to merge; first-usable-wins is the whole semantic (mirrors 005 ADR-2).
3. **No built-in default; report "unset"** — rejected: the spec explicitly requires a default so commands work out of the box, and forbids an "absent" outcome.

**Decision**: Option 1. Flag short-circuits ahead of all else when non-empty; then `GLASSFROG_BASE_URL`; then the nearest `.glassfrogrc` `base_url` up the tree and at home; then the compile-time default. A whitespace-only value or an absent `base_url` key at a location is "nothing here" and falls through; the default guarantees termination with a value.

**Consequences**: Deterministic (same inputs → same `Value`+`Source`). "Usable" must be defined precisely (ADR-4) so a blank value doesn't win and a malformed one doesn't pass. No caller ever has to handle "no base URL."

### ADR-3: Reuse `internal/auth`'s single `.glassfrogrc` reader for the `base_url` key — never a second parser; the token is never returned to base-URL callers

**Context**: 005 established one shared `.glassfrogrc` reader so the read/write sides can't drift, and LEARNINGS records a concrete drift bug from two code paths over one file. The file now carries a second key (`base_url`) beside `token`. CONSTITUTION XII wants no new dependencies. But `auth`'s package invariant is secret hygiene — base-URL callers must not come into possession of the token.

**Options considered**:
1. **Reuse `auth`'s parser + walk through a narrow, secret-safe seam** that returns only the requested `base_url` value (plus its path), never the token. `auth` keeps owning the file format and the walk; `internal/apiclient` owns the precedence policy, validation, default, env var, and flag.
2. **Duplicate a tiny `key=value` reader in `internal/apiclient`** — rejected: a second reader of the same file is exactly the drift class LEARNINGS warns about (a Storage-side change to the format would silently desync this path).
3. **Return the whole parsed key→value map (including `token`) to `apiclient`** — rejected: hands the secret to a non-secret consumer, breaking `auth`'s secret-hygiene invariant.

**Decision**: Option 1. `auth` exposes a network-free base-URL file read over the same `candidateDirs` walk and the same `parseCredentials` step, returning `(value, path, found, error)` for the `base_url` key only — the token never crosses this seam. The `base_url` file-key constant lives in `auth` beside `tokenKey` (it's a `.glassfrogrc` format detail); `GLASSFROG_BASE_URL`, the flag name, and the default URL are `internal/apiclient` constants (connection concerns).

**Consequences**: One parser, one walk, no drift; the token stays out of base-URL code paths. `auth`'s responsibility broadens honestly from "the token" to "the `.glassfrogrc` file" while staying network- and command-free. The three names (`base_url`, `GLASSFROG_BASE_URL`, `--base-url`) stay `[ASSUMED]` pending reconciliation with Credential Storage; the default URL value is pinned from `spec/glassfrog-api-v5.yaml` (`https://glassfrog.com/api/v5`). *Precedent-setting: a second `.glassfrogrc` key is read through the one shared reader, not a parallel parser; non-secret callers never receive the token.*

### ADR-4: Validate the resolved value as an absolute `http(s)` URL; malformed → typed code-free `BaseURLError` naming the source, no fall-through; the default is never re-validated

**Context**: The clarify session pinned "usable" as an absolute URL with an `http`/`https` scheme; a malformed value is reported as a format error naming its source, not passed through or skipped. CONSTITUTION III (fail loud, not silent) and the 002/004/005/007 "producer classifies a code-free outcome, consumer maps it" split both apply.

**Options considered**:
1. **Validate at the resolution boundary; typed code-free `BaseURLError{Source, Path}`** — a non-empty value that is not an absolute `http(s)` URL fails loud naming where it came from, with no fall-through; whitespace-only is absent (fall through); the compile-time default is known-valid and not re-validated.
2. **Pass the value through unvalidated, let the eventual request fail** — rejected: the spec requires resolution to report the malformed value, and a deferred failure loses the source attribution.
3. **`auth`/the file reader validates URL-ness** — rejected: URL validity is a connection-configuration concern, not a `.glassfrogrc` format concern; `auth` only reports the raw string and its path.

**Decision**: Option 1. `internal/apiclient` validates each non-empty candidate (the implementation detail of `net/url` parsing + scheme check is the interface/build concern). A malformed value returns a typed `BaseURLError` that names the source (`flag`, `GLASSFROG_BASE_URL`, or the file path) and never carries anything secret. 007's read/format-error stance and 005's typed errors are the precedent.

**Consequences**: Code-free, mirroring 005/007 — the consuming command maps the error to an exit code. **Open downstream gap (not this slice's to resolve):** 004's frozen convention has no dedicated "bad configuration" code (code 4 is *API*-side rejection); which code a malformed-base-URL outcome receives is for the consuming-command spec / clarify — exactly the gap 007 flagged for "cannot authenticate." This slice surfaces only the typed outcome.

### ADR-5: Inject the flag value, environment, and start/home directories; the resolver is pure

**Context**: CONSTITUTION IV requires hermetic, RED-first tests, and resolution is almost entirely environment/filesystem-dependent. 005 ADR-5 already established injected roots so tests never read the developer's real `~/.glassfrogrc`; the file rung here reuses that injected walk.

**Options considered**:
1. **Inject `flagValue`, the env (through a thin seam), `startDir`, and `homeDir`** into a pure resolver; a small production wrapper binds the real `os` values and the flag. Hermetic, table-driven tests over temp trees; reuses 005's seam.
2. **Read globals inside the resolver** — rejected: non-hermetic, couples tests to real env/CWD, risks reading the real home file (the same reasons 005 rejected it).

**Decision**: Option 1 — injected seams, parallel to 005. The flag value is an input (empty = absent); the production wrapper binds `os.Getenv`/`os.Getwd`/`os.UserHomeDir` and the flag source.

**Consequences**: Every precedence rung and the malformed/empty/absent cases are unit-tested with no I/O, and the no-read guarantees are pinned with 005's directory-at-path tripwire (LEARNINGS). The `--base-url` cobra flag's *registration* is deferred to the consuming command (see Risks / What This Plan Does Not Cover); the resolver accepts the flag *value* now, exactly as 005/007 built resolvers ahead of their consumers.

---

## Integration Design

- **`.glassfrogrc` file format (specification boundary, shared with Credential Discovery 005 & Credential Storage 006)**: this slice adds a second key, `base_url`, read through 005's one shared parser. The key name is `[ASSUMED]`; reconcile with Credential Storage when it gains base-URL support, so the writer and reader agree.
- **`internal/auth` (internal dependency)**: provides the `.glassfrogrc` parser and `candidateDirs` walk; exposes a secret-safe `base_url` file read (value + path, never the token). `auth` stays network- and command-free.
- **Command-line invocation (input)**: the `--base-url` flag is the top-precedence source. Its *value* is an input to the resolver now; its cobra *registration* belongs with the future command that triggers API calls.
- **Environment (input)**: `GLASSFROG_BASE_URL` (`[ASSUMED]`); a non-empty, valid value short-circuits the file search. Whitespace-only / unset falls through (uniform "usable" rule, LEARNINGS).
- **Glassfrog API v5 specification (reference, `spec/glassfrog-api-v5.yaml`)**: the built-in default is the v5 server URL. The spec's `servers:` block declares a *relative* `url: /api/v5` resolved against the documented host `https://glassfrog.com` (the `info.contact.url`), giving the concrete default **`https://glassfrog.com/api/v5`**. (The same spec pins `X-Auth-Token` as the `ApiKeyAuth` header — already a fixed PROJECT constant.)
- **Connection-context half + base transport (downstream, deferred)**: the resolved `BaseURL.Value` becomes the root of the base `http.Client`/`RoundTripper` that 007's `AuthTransport` wraps. This slice produces the value and its source; assembling the client is the deferred half.
- **Exit-Code Convention (004 — downstream)**: a malformed-base-URL outcome maps to a non-zero code, but the classification belongs to the consuming command, not this resolver (ADR-4).

---

## Cross-cutting Concerns

**Error handling (CONSTITUTION III)**: a non-empty source supplying a non-`http(s)` value, and an existing-but-unreadable/unparseable `.glassfrogrc`, both fail loud with a typed error naming the source/path — never a silent fall-through. Absence is handled by the default (a value is always produced), so there is no error channel for "not configured."

**Configuration**: the `base_url` file key is a centralized constant in `internal/auth` beside `tokenKey`; `GLASSFROG_BASE_URL`, the `--base-url` flag name, and the default URL are centralized constants in `internal/apiclient`. The default URL value is **pinned** to `https://glassfrog.com/api/v5` from `spec/glassfrog-api-v5.yaml`; the three *names* (`base_url`, `GLASSFROG_BASE_URL`, `--base-url`) remain `[ASSUMED]` (CLI conventions, not in the API spec), pending reconciliation with Credential Storage (the shared file key).

**Secret hygiene (CONSTITUTION II)**: although the base URL is not a secret, the shared file also holds the token. The `auth` seam this slice uses returns only the `base_url` value, never the token; base-URL code paths never hold the secret. Errors name only the source/path.

**Testing (CONSTITUTION IV)**: RED-first. The pure resolver is exercised over temp directory trees with injected roots (ADR-5) and a controlled `GLASSFROG_BASE_URL`: each precedence rung, nearest-wins, malformed-per-source, whitespace-empty-falls-through, file-without-`base_url`-falls-through, and the default backstop. The "no file read happened" guarantees (e.g. flag/env short-circuit) use the directory-at-path tripwire from LEARNINGS rather than asserting only on output. The driving scenarios become a godog suite **scoped to this spec's own feature file** (LEARNINGS: a suite must point at its own file, not the `features/` directory).

**No command surface (this slice)**: like 005 and 007, this capability registers no cobra command and prints nothing; the `--base-url` flag registration and the command that makes API calls are downstream. The cobra LEARNINGS findings do not apply here.

---

## Implementation Strategy

Three phases, linear. Depends on `internal/auth`'s shared reader and `candidateDirs` (005, landed) and on `internal/apiclient` (007, landed).

- **Phase 1 — `base_url` file read (in `internal/auth`)**: a network-free, secret-safe read returning the `base_url` value (and path) from the nearest `.glassfrogrc` over the existing `candidateDirs` walk and `parseCredentials` step; the token is never returned through it. Add the `base_url` file-key constant beside `tokenKey`. RED-first unit tests: value present, key absent (→ not found, fall through), malformed file (→ `*FormatError`), missing file (→ skip via `os.ErrNotExist`). *Depends on: existing 005 reader.*
- **Phase 2 — base URL resolver + precedence + validation (in `internal/apiclient`)**: `ResolveBaseURL(flagValue, startDir, homeDir) → (BaseURL, error)` implementing flag→env→file→default short-circuit, usable = absolute `http(s)` URL, typed `BaseURLError` naming the source on a malformed non-empty value (no fall-through), whitespace-empty/absent fall-through, and the default backstop; plus the production seam binding `os.Getenv`/`os.Getwd`/`os.UserHomeDir` and the flag value. Result `BaseURL{Value, Source, Path}` with the `Source` enum. RED-first unit tests per rung, malformed-per-source, empty-env-falls-through, file-without-`base_url`-falls-through, default, and determinism; no-read tripwires for the short-circuit rungs. *Depends on: Phase 1.*
- **Phase 3 — executable acceptance**: godog step definitions for the driving scenarios (flag wins, env wins, nearest file wins, default backstop, malformed-source error, unreadable-file error, empty-env falls through, file-without-`base_url` falls through), in `internal/apiclient`'s godog suite pointed at this spec's own feature file under a per-spec `features/` subdirectory. *Depends on: Phase 2.*

---

## Risks

- **Shared `.glassfrogrc` contract drift with Credential Storage** (medium likelihood, medium impact): Storage doesn't yet write `base_url`; if it later writes a different key/shape, reads break. Mitigation: reuse the one shared parser (ADR-3), keep `base_url` an `[ASSUMED]` centralized constant, flag reconciliation in the handoff.
- **Default URL host is derived, not literal** (low likelihood, low impact): `spec/glassfrog-api-v5.yaml` declares a *relative* server `url: /api/v5`, so the host `https://glassfrog.com` is taken from the spec's documented `contact.url`, giving `https://glassfrog.com/api/v5`. Mitigation: the default is a single centralized constant; if Glassfrog publishes an absolute server URL later, change it in one place. Confirm the host with the spec owner if the relative-server convention is load-bearing.
- **Secret leakage through the shared reader** (low likelihood, high impact): a generic key-reader could hand the token to base-URL code. Mitigation: the `auth` seam returns only `base_url` (never the token); ADR-3 makes this a contract, asserted by a test.
- **Non-hermetic tests touching the real home `.glassfrogrc`** (medium likelihood, high impact): a test reading the developer's real file could leak/flake. Mitigation: ADR-5's injected roots (reused from 005); only the production seam reads globals; tripwires pin the no-read guarantees.
- **URL-validity edge cases** (medium likelihood, low impact): scheme-less hosts and non-`http` schemes must be rejected, valid `http(s)` accepted. Mitigation: pin the accept/reject boundary with table tests on the validator (spec clarification is the oracle).
- **`--base-url` flag wiring deferred** (low likelihood, low impact): the future consuming command must feed the flag value into the resolver. Mitigation: the resolver takes the flag value as an injected input now; document the wiring obligation for the consuming command (same build-ahead pattern as 005/007).
- **Exit code for a malformed-base-URL outcome is undefined** (medium likelihood, low impact): 004 has no "bad configuration" code, and code 4 is API-side. Mitigation: stay code-free (typed `BaseURLError`); the classification is a downstream decision flagged for the consuming-command spec / clarify (mirrors 007's open gap). Does not block this slice.

---

## What This Plan Does Not Cover

- **The connection-context half** — combining the resolved base URL with the discovered token and assembling the base `http.Client`/`RoundTripper` that 007's `AuthTransport` wraps. Explicitly the deferred other half of Connection Resolution.
- **Timeouts, retries, response parsing, pagination** — Connection Configuration owns these, but they are not this base-URL slice.
- **Registering the `--base-url` cobra flag** — belongs with the future command that triggers API calls; this slice accepts the flag value as a seam input.
- **The command that makes API calls** — a future spec; this slice has no command surface.
- **The exit code for a malformed-base-URL outcome** — the consuming command + Exit-Code Convention (004); this slice surfaces a typed outcome only (ADR-4, flagged downstream gap).
- **Final `base_url` key / `GLASSFROG_BASE_URL` / `--base-url` names** — `[ASSUMED]` (CLI conventions); reconcile the file key with Credential Storage. (The default URL value is pinned: `https://glassfrog.com/api/v5`, from `spec/glassfrog-api-v5.yaml`.)
