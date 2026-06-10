# Validate: Role Domains

**Feature**: 033-role-domains
**Round**: 1 of 3
**Date**: 2026-06-10
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md, interface-cli.md, features/governance-reads/role-domains.feature, PROJECT.md
**Implementation files**: 6 — `internal/glassfrog/domains.go` (+ `domains_test.go`), `internal/cli/domains.go` (+ `domains_test.go`), `internal/cli/domain.go` (+ `domain_test.go`), `internal/cli/role_domains_bdd_test.go`, `internal/render/render.go` + 4 new templates (`domains.{full,compact}.tmpl`, `domain.{full,compact}.tmpl`), wiring in `internal/cli/app.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 of 3 validation scenarios satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (13 of 13 behavioral scenarios covered)

Every driving scenario has an identifiable code path; all 13 are additionally executable and green in the `TestRoleDomainsFeatures` godog suite (70 steps).

| Scenario | Status | Implementation |
|---|---|---|
| List a role's domains | ✓ Covered | `domains.go:runDomainsList` (human/structured walk via `paging.All[Domain]`) |
| Read a single domain by id | ✓ Covered | `domain.go:runDomainGet` (one `Execute` into `DomainDocument`) |
| Read a single domain with policies embedded | ✓ Covered | `domain.go:runDomainGet` (`?include=policies` → `DomainView` policies section) |
| Search a role's domains | ✓ Covered | `domains.go:domainsQuery` (sets `q` only when trimmed non-blank; rides every walked page) |
| No usable token | ✓ Covered | `runDomains`/`runDomain` → `reportClientError` → `UsageError(2)` |
| API cannot be reached | ✓ Covered | shared classifier → `NetworkUnavailable(6)` |
| Unknown domain id | ✓ Covered | `runDomainGet` → `reportClientError` → `APIError(3)` (names status) |
| Unsupported `--include` rejected (no call) | ✓ Covered | `domain.go:validateIncludeSet(…, {policies})` tripwire |
| `--include` rejected on the list | ✓ Covered | `domains.go:validateDomainsFlags` tripwire |
| Role controls no domains | ✓ Covered | `domains.*.tmpl` → `No domains.`, exit 0 |
| Multi-page walk to completion | ✓ Covered | `domains.go:runDomainsList` default `paging.All` walk |
| First-page opt-out signals more | ✓ Covered | `domains.go:runDomainsFirstPage` + `moreDomainsNote` |
| Mid-walk failure → partial flagged incomplete | ✓ Covered | `domains.go:reportIncompleteDomainsWalk` (non-zero, `incompleteDomainsNote`) |

---

## Acceptance Criteria

**Status**: Pass (4 of 4 tasks checked; criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 — grow `Domain` + `DomainDocument` | ✓ Met | `glassfrog/domains.go` adds `Type`/`RoleID *string`/timestamps/`Policies []Policy` + `DomainDocument`; list rides `Page[Domain]`; 4 decode tests incl. inline-embed no-regression |
| T002 — `domains <role-id>` list | ✓ Met | `domains.go` walk/`--first-page`/`--per-page`/`--query` (q on every page)/`--include` rejected/`domains` render key; 15 unit + 4 golden tests |
| T003 — `domain <dom-id>` single | ✓ Met | `domain.go` one `Execute`, `--include {policies}` validation, list-flags rejected, guarded `Policies` + null-role marker; 12 unit + 6 golden tests |
| T004 — executable acceptance | ✓ Met | `TestRoleDomainsFeatures` (Paths scoped to the one feature file); 13 behavioral pass, 3 `@validation` held |

Error-class routing (`*AuthError`→2, `*TransportError`→6, `*ResponseError`→3/4/5, `*MalformedPageError`→1, `*DecodeError`→3 per the 031 divergence) is delivered by reuse of the shared `classifyClientError`/`refineClientError` (011/015/031) — 033 adds no mapping, consistent with the acceptance criteria's "via the shared classifier" wording. The no-token / transport / 404 branches are additionally exercised directly in the unit suites.

---

## Interface Contract Conformance

**Status**: Pass (2 of 2 command surfaces conformant)

| Surface | Status | Notes |
|---|---|---|
| `glassfrog domains <ROLE_ID>` | ✓ Conformant | `ExactArgs(1)`; `--query/-q`, `--first-page`, `--per-page`; inherits `--base-url`/`-o`; `--include` rejected; full = `<desc> (id)`, compact = `id  <desc>`, empty = `No domains.` |
| `glassfrog domain <DOMAIN_ID>` | ✓ Conformant | `ExactArgs(1)`; `--include` (`{policies}`); rejects `--query`/`--first-page`/`--per-page`; full = desc + id + `Role:` + guarded `Policies:`, compact = `id  desc  role=…` |

Interactions verified: `--output` resolved first, then flag-applicability + `--include` validation before assembly (transport tripwire); search composes with the walk (`q` on every page — unit-pinned); single read is unpaginated and sends no `If-None-Match`; structured formats emit the raw `{data:[…]}` / `{data}` payload (018). Completeness notes match the accord verbatim (`note: more domains exist than shown; re-run without --first-page to fetch all` and `note: result is incomplete — <cause>; the domains shown are a partial set`). No new `Outcome`/`ExitCode`/root flag introduced.

---

## Non-Behavior Absence

**Status**: Pass (6 of 6 exclusions honored)

| Non-behavior | Status | Evidence |
|---|---|---|
| No re-implementation of the inline domains embed | ✓ Absent | `Domain` grown additively; Role/TreeNode embeds render only `Description`, unchanged |
| No org-wide "all domains" list / no role-less list | ✓ Absent | `domains` is `ExactArgs(1)` required role id; `GET /roles/{id}/domains` only |
| No standalone policy read | ✓ Absent | grep confirms no policy command/render key; `--include policies` embeds inline only |
| No raw-JSON default / no private format flag | ✓ Absent | dispatches through Output Format Selection (020); default `full`; no own flag |
| No base-URL/token/header/fail-safe/error-typing/exit-code ownership | ✓ Absent | reuses 005/007/008/015/004; never reads `ctx.Cred.Token`; adds no `Outcome`/`ExitCode` |
| No write/mutation of a domain | ✓ Absent | `GET`-only on both paths |

---

## @wip Lifecycle Completion

**Status**: Pass

The 13 behavioral scenarios referenced by the checked tasks have `@wip` removed and pass in the godog suite. The 3 `@validation` scenarios retain `@wip` by design (held for this skill), which is the correct end state — they are not behavioral-task scenarios.

---

## Validation Scenario Results

**Status**: Satisfied (3 of 3 traced to implementation, independently of the driving-scenario pass)

| Scenario | Status | Trace |
|---|---|---|
| Default output carries no raw API envelope | ✓ Satisfied | Default human format (`full`) renders via `domains.full.tmpl` → `<desc> (id)` per domain; grep confirms the `domain*`/`domains*` human templates contain no `data`/`meta` literals. The raw envelope appears only on the separate `MachineFormat` (json/yaml) path. |
| Embedded-policies view is not a standalone read | ✓ Satisfied | `domain.full.tmpl` nests `Policies:` *under* the domain block via `DomainView.Domain.Policies`; grep confirms no standalone policy command or `ResourcePolicy` render key exists — the #34 surface is not produced here. |
| List incompleteness is never silent | ✓ Satisfied | A mid-walk stop writes `incompleteDomainsNote` to stderr **and** exits non-zero via `classifyClientError(Stop)`; `--first-page` writes `moreDomainsNote`. The explicit signal + cause + non-zero exit means a partial set cannot be read as complete (CONSTITUTION VI). |

No step definitions were registered for the `@validation` phrasings (they remain `@wip`), so verification is inspection-based, grounded by the greps above.

---

## Verdict: Ready

All 5 conformance dimensions pass with zero findings, and all 3 held-out validation scenarios are satisfied through independent inspection. The implementation conforms to the specification: both command surfaces match the interface accord, every driving scenario has a traced (and executably-green) code path, every spec non-behavior is honored, and the `@wip` lifecycle is complete. Build, vet, and the full test suite are clean.

---

## Next Steps

Implementation conforms to the specification. Suggest PR review and merge. The specification loop for 033-role-domains is closed.
