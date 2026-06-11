# Validate: Cross-Model Search

**Feature**: 041-cross-model-search
**Round**: 1 of 3
**Date**: 2026-06-11
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md (§ System Architecture), tasks.md (3 of 3 tasks complete), interface-cli.md, features/undiscoverable-governance/cross-model-search.feature, PROJECT.md
**Implementation files**: `internal/glassfrog/search.go` (+ `search_test.go`), `internal/cli/search.go` (+ `search_test.go`, `cross_model_search_bdd_test.go`), `internal/cli/include.go` (`validateClosedFlagSet`), `internal/cli/app.go` (wiring), `internal/render/render.go` (`ResourceSearch`, `SearchView`/`SearchRow`/`NewSearchView`, + `search_test.go`), `internal/render/templates/search.{full,compact}.tmpl`

---

## Conformance Summary

| Dimension | Status | Findings |
|---|---|---|
| Driving scenario coverage | ✓ Pass | 0 |
| Acceptance criteria | ✓ Pass | 0 |
| Interface contract conformance | ✓ Pass | 0 |
| Non-behavior absence | ✓ Pass | 0 |
| @wip lifecycle completion | ✓ Pass | 0 |
| **Validation scenarios** | ✓ Satisfied | 0 |

**Total**: 5 dimensions checked, 5 passed, 0 findings; 5 of 5 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (11 of 11 scenarios covered)

All driving scenarios from spec.md § Driving Scenarios have identifiable code paths, each exercised by a passing godog scenario (`TestCrossModelSearchFeatures`, 13 scenarios / 68 steps) and/or unit test.

| Scenario | Status | Implementation |
|---|---|---|
| Search across all resource types | ✓ Covered | `search.go:runSearch`→`runSearchList`→`paging.All[SearchResult]`; `searchQuery` sets `query`, omits `types` |
| Scope a search to specific types | ✓ Covered | `searchQuery` sets `types=role,project`; `bdd:onlyRoleAndProjectPrinted` |
| Each result carries the bridge | ✓ Covered | `render.NewSearchView` surfaces type/id/title/excerpt/rank + Role line; `search.full.tmpl` |
| Multi-word websearch query forwarded verbatim | ✓ Covered | `searchQuery` `q.Set("query", cfg.query)` (no transform); quote-aware `splitArgs` in suite |
| No usable token | ✓ Covered | `runSearch`→`reportFailure`→`classifyClientError` (AuthError{NoCredentials}→UsageError 2) |
| Query the API rejects as malformed (400) | ✓ Covered | walk Stop on 400→`reportFailure`→APIError(3); names status |
| A search that matches nothing | ✓ Covered | empty `Records`→`search` template prints `No results.`, exit 0 |
| Unsupported `--types` rejected without an API call | ✓ Covered | `validateTypes` before assembly; transport tripwire (`tr.calls==0`) |
| A missing query is a usage error | ✓ Covered | cobra `ExactArgs(1)`→`Run` arg-validator branch→UsageError(2), no request |
| Multi-page walk to completion | ✓ Covered | `paging.All` walks all pages; `bdd:everyPageWalked` (2 calls, both pages printed) |
| First-page opt-out stops at one page, signals more | ✓ Covered | `runSearchFirstPage` single `Execute`, `moreSearchNote` on `HasNextPage`, exit 0 |

---

## Acceptance Criteria

**Status**: Pass (3 of 3 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — flat `SearchResult` via `Page[SearchResult]` | ✓ Met | `glassfrog/search.go`; 5 decode tests (snake-case binding, nullable excerpt/role_id, float rank, empty page, unknown-field tolerance) pass |
| T002 — `search` command, `--types` validation, walk + `--first-page` + `--per-page`, render, wiring | ✓ Met | `cli/search.go`; 21 unit tests cover every branch incl. `per_page=100` default, query+types on every page, full error mapping; 6 render goldens; wired in `app.go` |
| T003 — driving scenarios as executable acceptance | ✓ Met | `cross_model_search_bdd_test.go`; 13 behavioral scenarios pass, `@validation` held |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces conformant)

| Surface element (interface-cli.md) | Status | Implementation |
|---|---|---|
| `search <QUERY>` runnable leaf, `ExactArgs(1)`, non-empty `Short`, `SilenceErrors/Usage` | ✓ Conformant | `newSearchCommand` |
| `--types` (comma-separated, reject-unknown, sent only when set) | ✓ Conformant | `validateTypes`; `searchQuery` omits when empty |
| `--first-page`, `--per-page` (1–100, API owns range) | ✓ Conformant | flags + `searchPageSizeFor` |
| Inherited `--base-url`, `-o/--output` (not redeclared) | ✓ Conformant | read via `apiclient.FlagBaseURL`/`output.FlagOutput` |
| `full` row layout (`[type] title (id)  rank R` / `Excerpt: …\|—` / conditional `Role:`) | ✓ Conformant | `search.full.tmpl` (golden-pinned) |
| `compact` row layout (`[type]  id  title  rank=R`) | ✓ Conformant | `search.compact.tmpl` (golden-pinned) |
| Empty result → exactly `No results.`, exit 0 | ✓ Conformant | template `{{if not .Rows}}No results.` |
| Completeness notes (more-exist / incomplete on stderr; walk-by-default) | ✓ Conformant | `moreSearchNote`, `incompleteSearchNote`, `reportIncompleteSearchWalk` |
| Error table → `classifyClientError`; no new `Outcome`/`ExitCode`/root flag | ✓ Conformant | reuses landed classifier; no new enum/flag added (verified) |

---

## Non-Behavior Absence

**Status**: Pass (no excluded behavior present)

| Non-behavior (spec.md) | Status | Evidence |
|---|---|---|
| Must not parse/rewrite/escape/validate query syntax | ✓ Absent | `searchQuery` `q.Set("query", cfg.query)` — no Split/Escape/Trim/Replace on query |
| Must not re-rank/re-sort/de-dup/filter results | ✓ Absent | `NewSearchView` preserves input order; no `sort`/`Slice` in the search render path |
| Must not auto-fetch the resource per result | ✓ Absent | only `Path: "/search"` requests; no per-result reads |
| Must not invent `title`/`excerpt`, nor count omitted | ✓ Absent | null/blank excerpt → `—` marker, null role_id omits Role line; no fabrication |
| Must not resolve base-URL/token/header/typing/exit-codes itself | ✓ Absent | reuses `assemble`/`newClient`/`classifyClientError`; no token field read |
| Must not emit raw API JSON as fixed default nor define a private format flag | ✓ Absent | human default renders the `search` projection via `renderResult`-style dispatch; no private `--output` |
| Must not write/mutate/capture from a hit | ✓ Absent | GET-only; no write path |

---

## @wip Lifecycle Completion

**Status**: Pass

The 13 behavioral scenarios referenced by checked task T003 have had `@wip` removed and pass in `TestCrossModelSearchFeatures`. The 5 `@validation @wip` scenarios remain held — correctly, as they are reserved for this independent verification step, not referenced by an implement task for un-tagging.

---

## Validation Scenario Results

**Status**: Satisfied (5 of 5 traced to implementation)

Traced independently against code paths (not assumed from the driving-scenario pass). The feature file carries 5 `@validation` scenarios (spec.md § Validation Scenarios lists 4; the feature adds the architecture-informed "A rejected type issues no request").

| Scenario | Status | Trace |
|---|---|---|
| The query reaches the API byte-for-byte | ✓ Satisfied | `searchQuery` sets `query` verbatim; `TestRunSearch_QueryForwardedByteForByte` asserts `lastQuery.Get("query")` equals input incl. operators |
| Default output carries no raw API envelope | ✓ Satisfied | default `full` → `renderFn(ResourceSearch, …)` over `NewSearchView`; the raw `{data,meta}` path is reached only under `json`/`yaml` (`MachineFormat`) |
| A rejected type issues no request | ✓ Satisfied | `validateTypes` runs in `runSearch` step 2, before `assemble`; `TestRunSearch_UnsupportedTypeRejectedBeforeRequest` asserts `tr.calls==0` |
| The rendered order matches the API's relevance order | ✓ Satisfied | `paging.All` appends pages in API order; `NewSearchView` preserves it; `TestNewSearchView_PreservesInputOrder` pins a non-rank-sorted input |
| A partial result set cannot be read as complete | ✓ Satisfied | mid-walk `res.Stop != nil` → `reportIncompleteSearchWalk` writes `incomplete — <cause>` to stderr + non-zero exit; partial set on stdout is flagged |

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings. All 5 validation scenarios are independently traced to clear code paths. The implementation conforms to the specification: the query is forwarded verbatim, the API's relevance order is preserved, `--types` is validated locally before any request, the result walks to completion by default (with a `--first-page` opt-out and mid-walk incompleteness signalling), nullable fields render as absent rather than fabricated, and the command reuses the landed transport/auth/pagination/error/output machinery without adding a new `Outcome`, `ExitCode`, or output flag. No excluded behavior is present.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge.

The 5 `@validation` scenarios remain `@wip` in the feature file by design (they are this skill's held-out verification, traced here by inspection rather than executed as un-`@wip` godog steps). Leaving them tagged keeps `TestCrossModelSearchFeatures` reporting only the 13 behavioral scenarios; the conformance evidence for the held-out set lives in this validate.md. If the team's convention is to also land executable `@validation` step definitions, that is a follow-up, not a conformance gap.
