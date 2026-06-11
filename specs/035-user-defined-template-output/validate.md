# Validate: User-Defined Template Output

**Feature**: 035-user-defined-template-output
**Round**: 1 of 3
**Date**: 2026-06-11
**Verdict**: Ready
**Artifacts loaded**: spec.md, plan.md, tasks.md (5 of 5 complete), interface-spec.md, interface-cli.md, features/unconsumable-output/user-defined-template-output.feature, PROJECT.md
**Implementation files**: 5 changed/added across 3 packages — `internal/render/usertemplate.go`, `internal/output/selection.go`, `internal/cli/{usertemplate.go, render.go, diagnostic.go}` + the seam/wiring edits in `me*.go`, `roles.go`, `tree.go`, `subroles.go`, `domains.go`, `domain.go`, `policies.go`, `projects.go`, `root.go`

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

**Total**: 5 dimensions checked, 5 passed, 0 findings; 3 validation scenarios, 3 satisfied.

---

## Driving Scenario Coverage

**Status**: Pass (6 of 6 driving scenarios covered)

Every driving scenario has an identifiable code path; the 6 non-`@validation` scenarios run as an executable godog suite (`TestUserDefinedTemplateOutputFeatures`, 8 scenarios / 42 steps, all passing — the suite also includes the architecture-informed "execution failure writes nothing to stdout" and a second stdin scenario).

| Scenario | Status | Implementation |
|---|---|---|
| a template file renders the result | ✓ Covered | `resolveRenderTarget` → `readTemplateSource` (TemplateFile) → `ParseUserTemplate`; `writeHuman` via `runMeRoles`→`renderResult` (`me_roles.go`, `render.go`, `usertemplate.go`) |
| a template is read from piped stdin | ✓ Covered | `readTemplateSourceFrom` (TemplateStdin, isTTY/empty-guarded bounded read) → `tmpl.Render` (`usertemplate.go`) |
| a reserved name wins over a same-named file | ✓ Covered | `classifyFlagSelection` checks `ParseFormat` then `stdin` before treating a value as a file (`output/selection.go`) |
| a missing template file fails fast | ✓ Covered | `readTemplateSourceFrom` file branch → `reportTemplateError` → UsageError(2), no request (`usertemplate.go`) |
| a malformed template fails fast before the read | ✓ Covered | `ParseUserTemplate` in `resolveRenderTarget` runs before `assemble`/`newClient` (`usertemplate.go`, per-command `runX`) |
| a guarded template renders an absence marker | ✓ Covered | inherited `Option("missingkey=error")` + author guard; `me` with empty `.Roles` renders the `{{else}}` marker (`render/usertemplate.go`) |
| stdin selected with nothing piped | ✓ Covered | `readTemplateSourceFrom` TTY branch (`!tmplStdinPiped`) → UsageError(2), no request |

---

## Acceptance Criteria

**Status**: Pass (5 of 5 tasks complete, all criteria met)

| Task | Status | Evidence |
|---|---|---|
| T001 user-template engine | ✓ Met | `ParseUserTemplate`/`Render`/`UserTemplateError{Stage}` distinct from `*RenderError`; clone of built-in set; no new FuncMap; 11 unit/golden tests |
| T002 discriminated selection | ✓ Met | `TemplateRef`/`Selection`/`ResolveSelection(FromOS)`; flag-only template-ref rule; `stdin` reserved; env/file → `*FormatError`; 8 pure-core tests |
| T003 classifier arm | ✓ Met | `Diagnose` arm `*render.UserTemplateError → UsageError`; both stages pinned by `TestClassifyClientError_UserTemplateError` |
| T004 dispatch arm | ✓ Met | `renderResult` `tmpl` param + unified `writeHuman` (execute→Usage, defect→Runtime, buffer-then-write); test pins empty stdout + source naming |
| T005 read wiring | ✓ Met | every `--output`-capable seam → `resolveSelection`+`readTemplateSource`; `resolveRenderTarget` fail-fast; `-o` usage string widened; new godog suite |

---

## Interface Contract Conformance

**Status**: Pass (all surfaces present with the specified shapes)

| Surface (interface-spec / interface-cli) | Status | Evidence |
|---|---|---|
| `render.UserTemplate`, `ParseUserTemplate(text) (*UserTemplate, error)`, `(*UserTemplate).Render(any) (string, error)` | ✓ Conformant | `internal/render/usertemplate.go` |
| `render.UserTemplateError{Stage, Source, Err}`, `errors.As`-discriminable, distinct from `*RenderError` | ✓ Conformant | `usertemplate.go`; `TestUserTemplateError_DistinctFromRenderError` |
| `output.TemplateRef{Kind, Path}`, `Selection{Format, Template}` + `AsTemplate()` | ✓ Conformant | `internal/output/selection.go` (matches the documented `Format OutputFormat + Template *TemplateRef`, `Template == nil` ⇒ built-in) |
| `output.ResolveSelection` / `ResolveSelectionFromOS`; reserved words = 4 tokens + `stdin` | ✓ Conformant | `selection.go` (`reservedStdin`) |
| cli seam `resolveSelection` + `readTemplateSource`; `renderResult` user-template arm; `classifyClientError` arm | ✓ Conformant | `internal/cli/{usertemplate.go, render.go, diagnostic.go}` |
| Widened `-o`/`--output` usage string naming a template path + `stdin` | ✓ Conformant | `root.go:53-56` |
| Error Communication table (file-missing/parse/stdin/execute → UsageError(2); execute is the one post-response case) | ✓ Conformant | `reportTemplateError` (pre-request) + `writeHuman`→`classifyClientError` (execute) |

---

## Non-Behavior Absence

**Status**: Pass (0 violations)

| Non-behavior | Status | Evidence |
|---|---|---|
| No template source from env/config (flag-only) | ✓ Absent | `ResolveSelection` classifies only the flag rung; env/file delegate to `ResolveFormat` (four tokens or `*FormatError`, never a `TemplateRef`) — `selection.go`; `TestResolveSelection_{Env,File}NonTokenIsFormatError` |
| No reimplementation of the four built-in renderers | ✓ Absent | `writeHuman` routes built-ins through the unchanged `renderFn`; structured arm unchanged; the user template is a parallel path, not a fork |
| No rendering of *failures* through a user template | ✓ Absent | `tmpl.Render(v)` is called in exactly one place (`writeHuman`, success only); `reportFailure` takes `rt.format` (DefaultFormat when a template is selected → human cause-plus-next-step) |
| No fabricated/defaulted data value | ✓ Absent | inherited `Option("missingkey=error")`; unguarded missing field → `StageExecute` error, empty stdout — `render/usertemplate.go`; `TestUserTemplate_Render_Missing*` |
| No code/file/network access from a template | ✓ Absent | `render/usertemplate.go` imports only `bytes`/`fmt`/`text/template`; no new FuncMap helper added — data-only sandbox by construction (file/stdin I/O lives behind the cli seam, not in the engine) |
| No change to which fields the read fetches | ✓ Absent | the request (`apiclient.Request`) is built unchanged; the template renders the already-decoded result value downstream of the send |
| No modification of 020's precedence/default/env-config | ✓ Absent | `ResolveSelection` reuses `ResolveFormat` for precedence; only the flag-rung value interpretation widens |

---

## @wip Lifecycle Completion

**Status**: Pass

The 8 implementation scenarios have had `@wip` removed and run green; the 3 `@validation @wip` scenarios remain `@wip` correctly — they are held out for this validation pass and are not referenced by any implementation task. No stray `@wip` on an implemented scenario.

---

## Validation Scenarios

**Status**: Satisfied (3 of 3) — held out from the Builder, traced independently.

| Validation scenario | Status | Independent trace |
|---|---|---|
| a template source is never honored from env or config | ✓ Satisfied | flag-absent path: `ResolveSelection` → `ResolveFormat("", env, file, …)` returns `*FormatError` for a non-token env/config value, never a `TemplateRef` (`selection.go`); pinned by `TestResolveSelection_EnvNonTokenIsFormatError` / `…FileNonTokenIsFormatError`. No `readTemplateSource` call is reachable without a flag-rung `TemplateRef`. |
| a user template introduces no value absent from the source | ✓ Satisfied | the engine projects only fields of the decoded result value; an unguarded missing struct field or map key fails under `missingkey=error` (`StageExecute`, empty output) rather than synthesizing a value — `TestUserTemplate_Render_MissingStructField` / `…_MissingMapKey`. An absence marker is the author's `{{else}}` text (structural), not a data value. |
| a malformed template is caught before any API call (file or stdin) | ✓ Satisfied | `resolveRenderTarget` calls `readTemplateSource` then `ParseUserTemplate` before `assemble`/`newClient`/send, for both `TemplateFile` and `TemplateStdin`; a parse error returns via `reportTemplateError` with no client built. The file case is also pinned by the godog "fails before any request" scenario (transport asserts 0 calls); the stdin case shares the identical pre-request code path. |

---

## Verdict

**Ready.** All 5 tasks are checked, all 5 conformance dimensions pass with zero findings, and all 3 held-out validation scenarios trace to clear code paths. The implementation fulfills the spec's promises: a flag-rung-only template source (file or stdin), rendering the invoked read's success through the same seam as the built-ins, anti-fabrication preserved by the inherited `missingkey=error` guard, a data-only sandbox held by construction (no new FuncMap helper, I/O behind the cli seam), and every template failure mapped to the conventional usage exit code without a new code. The one spec-extending behavior (a post-response *execution* failure spends a request before exiting 2) is the documented, unavoidable consequence of `text/template`'s dynamic typing (plan Risks / ADR-3) and stays within the operator-input-is-usage-error intent.

`go build`, `go vet`, and `go test ./...` are clean; the prior 020/projection suites stay green (a non-token `--output xml` now fails fast as a missing-template-file usage error — same category, no request).

---

## Handoff

Implementation conforms to the specification. Suggest PR review and merge — the specification loop for 035 is closed.
