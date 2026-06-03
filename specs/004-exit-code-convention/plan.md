# Plan: Exit-Code Convention

**Feature**: 004-exit-code-convention
**Role**: Shaper
**Inputs**: spec.md (004), PROJECT.md, CONSTITUTION.md (context), DECISIONS.md, LEARNINGS.md; existing code in `internal/cli/` and `main.go`

---

## System Architecture

Exit-Code Convention is the final, thinnest layer of the CLI Skeleton. It does not introduce a new subsystem — it completes a seam the earlier specs deliberately left open: every invocation already flows through `cli.Run` (002), which returns a **code-free `Outcome` category**, and `main.go` already maps that category to a process code with a *documented placeholder* (`success → 0, any error → non-zero`). This feature replaces the placeholder with the canonical, finer-grained mapping and gives the category set its missing member.

Three small parts, all in the existing `internal/cli` package:

```
 main.go
   └─ os.Exit( cli.Main() )                                     [thin wrapper]

 internal/cli : func Main() int   [NEW — extracted, testable entrypoint]
   ├─ cli.Run(cli.Assemble(), os.Args[1:]) → (Outcome, error)   [002, unchanged shape]
   ├─ deferred recover() → returns 1 on panic, writes crash to stderr   [ADR-4]
   └─ returns cli.ExitCode(outcome)                             [NEW: 004 owns this]
         │
         ▼
 internal/cli/exitcode.go  [NEW — the single canonical registry]
   • named code constants 0–6 (the published, frozen convention)
   • ExitCode(Outcome) int — pure category→code lookup, Fail-Safe default = 1
         ▲
         │ category produced by …
 internal/cli/dispatch.go  [Outcome enum — extended]
   • Success, UsageError            (existing producers)
   • RuntimeError                   (NEW — resolves 002's deferral)
```

**Flow**: a producer classifies an outcome into an `Outcome` category (Argument Dispatch labels `Success`/`UsageError`; a resolved command's own action failure becomes `RuntimeError`). The extracted `cli.Main()` asks the registry for the code and returns it; `main` exits with it. The registry is a *pure mapper* — it never classifies, renders text, retries, or inspects an `error`. It is the single site where a category is bound to a code.

**Forward-looking categories**: the spec's API-facing categories (API error, permission, rate-limit, network-unavailable) have **no producer yet** — no API client exists in the skeleton. Their codes (3–6) are published now as named constants (the frozen convention an agent can rely on), but the live `Outcome` enum grows to include them only when their producer — the future API client — lands and classifies them. This is the extensibility model from the spec's clarification: a new category is added at the one registry site, taking a pre-reserved code, and existing codes are never renumbered.

---

## Architecture Decisions

### ADR-1: 004 is a pure category→code mapper; the `Outcome` enum is the category vocabulary and `ExitCode` is the single registry

**Context**: The spec's clarifications fixed two things: (Q1) 004 stays a pure category→code mapper — producers classify, 004 maps; (Q2) a single canonical category→code registry owned by 004 is the source of truth. The codebase already has an `Outcome` enum in `dispatch.go` described as "the code-free classification" — exactly the category concept — and 002's DECISIONS entry says 004 is "the consumer that will map this category to a process code without re-deriving it from an untyped error."

**Options considered**:
1. **Extend `Outcome` + a `ExitCode(Outcome) int` registry** — the existing enum *is* the category vocabulary; 004 adds the missing category and a single mapping function. One type, one registry, no translation layer.
2. **New `ExitCategory` type + a `Classify(Outcome, error) ExitCategory` translator + `ExitCode(ExitCategory)`** — keeps `Outcome` frozen at two values; introduces a richer parallel type. More types and an extra translation step that re-inspects the `error` — the very thing 002's ADR-2 said 004 should avoid.

**Decision**: Option 1. `Outcome` is the canonical category vocabulary; `ExitCode(Outcome) int` in a new `internal/cli/exitcode.go` is the single registry. Producers return categories; `main` calls `ExitCode`. No error re-inspection, no parallel type.

`ExitCode` is a small total function (a `switch` or package-level table) over the categories with producers, with a **Fail-Safe default arm returning the internal-error code `1`** so any unmapped/future category can never accidentally yield `0`.

**Consequences**: One obvious home for the convention. The mapping is trivially unit-testable. Because the registry is pure, the no-renumbering and uniqueness rules are enforceable in one place. Downside: `Outcome` now spans two source files conceptually (the type in `dispatch.go`, the code mapping in `exitcode.go`) — mitigated by a doc comment cross-reference.

### ADR-2: Publish the full 0–6 code convention as named constants now; grow the live category enum only as producers arrive

**Context**: The spec (Q3) pinned exact values — `0` success, `1` internal, `2` usage, `3` API, `4` permission, `5` rate-limit, `6` network — and froze them by the no-renumbering rule. But only `Success`, `UsageError`, and the new `RuntimeError` have producers in the skeleton; the operational categories are explicitly "forward-looking convention."

**Options considered**:
1. **Pre-populate all seven `Outcome` enum values now** — the enum is the complete published vocabulary; uniqueness tests are total over it. But four enum values would have no producer and no `ExitCode` caller path exercised — dead exported symbols inviting "why is this here" churn, and a misleading signal that the skeleton classifies API failures (it doesn't).
2. **Publish codes as named constants, grow the enum with producers** — define `codeSuccess…codeNetworkUnavailable` (0–6) as the frozen convention in `exitcode.go`; add the `Outcome` value (and its `ExitCode` case) for a category only when its producer exists. Operational constants are documented as reserved-for-the-API-client.
3. **Document the operational band in a comment only** — no constants until needed. Lightest, but the pinned contract is then unenforced and undiscoverable; an agent can't see the frozen values, and the uniqueness/no-reserved checks have nothing to test.

**Decision**: Option 2. The seven code constants are the published, frozen, test-pinned contract; the `Outcome` enum carries only categories with producers (gains `RuntimeError` now). When the API client lands it adds `APIError`/`PermissionError`/`RateLimited`/`NetworkUnavailable` to the enum and the matching `ExitCode` cases, each referencing its already-reserved constant.

**Consequences**: The contract is real and testable today (uniqueness, no shell-reserved range, stable values) without dead enum values. The extension recipe is concrete and one-site. Downside: a small asymmetry — codes exist as constants before their categories exist as enum values; documented explicitly in the registry comment so it reads as intentional, not incomplete.

### ADR-3: Resolve 002's deferred "RuntimeError" as the catch-all internal-error code (1), reclassified at the producer (dispatch)

**Context**: `dispatch.Run`'s default arm currently returns `(Success, err)` for a resolved command whose own action fails, with two tests (`TestRun_RuntimeActionError_IsSuccessCategory`, `TestRun_RuntimeError_NotMisclassifiedAsArgError`) pinning that "Success, RuntimeError deferred to 004." 004 is that consumer. The spec defines no distinct "runtime" code — the only non-success/non-usage code for an unclassified failure is the catch-all `1` (internal/unexpected). In the skeleton there is no API classifier, so a resolved command's failure has no more-specific category.

**Options considered**:
1. **Add `RuntimeError` to `Outcome`, map it to code `1`, reclassify in dispatch (the producer)** — dispatch's default arm returns `RuntimeError` instead of `Success` when the action errored. Honors "producers classify"; `main` needs only the category. Requires updating the two deferral tests (which were written to be updated here).
2. **Leave dispatch returning `(Success, err)`; have `main`/registry treat "Success with non-nil error" as code 1** — no change to dispatch, but it forces `ExitCode` (or `main`) to re-inspect the `error`, reintroducing the error-derivation 002's ADR-2 and ADR-1 above explicitly reject, and leaving `Success` meaning two different things.

**Decision**: Option 1. Add `RuntimeError` to the `Outcome` enum (`String()` updated), map `ExitCode(RuntimeError) = 1`, and change dispatch's default arm so a resolved command whose action returned an error is classified `RuntimeError`. The two deferral tests are updated to assert `RuntimeError` (and that the error still travels via the return). This is the deferral being fulfilled — silent conformance to 002's DECISIONS entry, not a divergence.

**Consequences**: A command failure now exits `1` (never `0`), satisfying Fail Safe end-to-end. `Success` regains a single meaning (ran-and-succeeded, or help/listing). Blast radius is contained to the two named tests plus `String()`; the BDD harness gains exit-code steps but its existing `Success`/`UsageError` assertions are unchanged. The arg-rejection arm (UsageError) and flag-failure arm are untouched.

### ADR-4: Recover from panics at the entrypoint (`cli.Main`) and exit `1`, to guarantee the safety net and avoid the Go panic→exit-2 collision

**Context**: The accord requires that *any* termination matching no known category — including an unanticipated internal failure — exits `1` and never `0`. Go's runtime, however, terminates an **unrecovered panic with exit status `2`** — which collides with our `UsageError = 2`. Without intervention, a nil-deref bug would masquerade to an agent as a usage error.

**Options considered**:
1. **Deferred `recover()` in the extracted `cli.Main()` that returns `1`** — the recover wraps the dispatch; on panic it (optionally logs and) returns `codeInternalError`, which `main` passes to `os.Exit`. Guarantees the spec's "unexpected internal failure → 1, never collides with 2", and stays testable in-process.
2. **Do nothing; accept Go's default panic exit 2** — simplest, but directly violates the accord and silently aliases internal crashes onto the usage code, defeating the finer-grained contract the whole feature exists for.

**Decision**: Option 1. The extracted `cli.Main() int` installs a deferred `recover()` that maps an unrecovered panic to a `codeInternalError` (1) return; `main` is the thin `os.Exit(cli.Main())`. The registry's default arm already returns `1` for unmapped categories; this closes the same guarantee for the panic path that bypasses the category system entirely. Returning the code from `Main()` (rather than calling `os.Exit` inside the recover) lets a test observe the `1` without `os.Exit` terminating the test binary; a subprocess smoke test covers the real `os.Exit` wiring end-to-end.

**Consequences**: The `1` safety net is honored on both paths (unmapped category and panic). An agent never sees a crash as a usage error. Downside: recovering swallows the default Go panic traceback — so the handler must still write the panic value (and ideally a stack) to stderr to preserve Action Transparency (CONSTITUTION II); 004 renders no *category* text, but a crash must remain diagnosable.

---

## Cross-cutting Concerns

**Error rendering stays out of 004**: cobra prints usage errors and a resolved command's `RunE` error; dispatch writes the synthesized nested-unknown-subcommand message (002/LEARNINGS). 004 emits only the numeric code — the panic-recover handler (ADR-4) is the one place 004 writes to stderr, and only to keep a crash diagnosable, never to render a category.

**Testing strategy** — mirrors the established "pin the contract with regression tests" convention (002 pinned `EnablePrefixMatching=false`, 003 pinned `EnableCommandSorting=true`):
- `exitcode_test.go`: pins the **published code constants** — the **values are exactly** `0/1/2/3/4/5/6` (a change-detector, so a future renumber breaks loudly), their **uniqueness** (no two constants share an integer), and that **none falls in the shell-reserved range** (126, 127, 128+N); plus `ExitCode` **mapping tests for the producer-backed categories only** (Success→0, UsageError→2, RuntimeError→1) and the Fail-Safe default→1. Per ADR-2 the operational categories (codes 3–6) have no `Outcome` value until their producer exists, so a full category↔code one-to-one test waits for that producer — until then those codes are pinned at the constant level.
- Update the two deferral tests in `dispatch_test.go` to assert `RuntimeError`.
- BDD: exercise the producer-backed scenarios in-process via the extracted `cli.Main()` (success / help / usage / internal-failure), plus a **subprocess smoke test** for the real `os.Exit` wiring and the panic→1 path; the operational-category scenarios (rate-limit / permission / different-classes) stay `@validation` (held out, pinned at the constant level until their producer lands).

**Configuration**: none. Codes are hardcoded constants — they are the contract, not a setting. The Fail-Safe default (`1`) is also hardcoded.

**Constitution alignment**: III (Fail Safe) — default and panic paths both yield non-zero `1`, a failure is never `0`. II (Action Transparency) — distinct codes per failure class are the machine signal; the panic handler preserves diagnosability.

---

## Implementation Strategy

Single phase (the feature is small and the dependency order within it is linear). Suggested ordering for PR-sized decomposition:

1. **Registry** — add `internal/cli/exitcode.go`: the seven `code*` constants (frozen convention) and `ExitCode(Outcome) int` with the Fail-Safe default. Add `exitcode_test.go` (exact-values, constant uniqueness, no shell-reserved, producer-backed-category mapping). RED→GREEN.
2. **Category** — extend `Outcome` with `RuntimeError`, update `String()`, and reclassify dispatch's default arm (`Success, err` → `RuntimeError, err` when the action errored). Update the two deferral tests in `dispatch_test.go`. RED first (flip the test expectations), then GREEN.
3. **Entrypoint** — extract a testable `cli.Main() int` that maps the outcome via `cli.ExitCode` and recovers a panic to return `1` (writing a stderr diagnostic); reduce `main.go` to `os.Exit(cli.Main())` and replace the placeholder doc comment with the real convention reference. Extracting `Main()` rather than inlining in `main` lets the exit-code and panic paths be exercised in-process (step 4), since `os.Exit` would otherwise terminate the test binary.
4. **Scenarios wiring** — bind the producer-backed exit-code scenarios via `cli.Main()` and add a subprocess smoke test for the `os.Exit`/panic→1 path, against `features/no-runnable-cli.feature`; the operational-category scenarios stay `@validation` until their producer lands.

Steps 1→2 are independent enough to land separately; 3 depends on both (it calls `ExitCode` and relies on `RuntimeError`); 4 depends on 1–3.

---

## Risks

- **Go panic exit-status-2 collides with `UsageError = 2`** — likelihood low per run, but impact high (an internal crash misread as the operator's input error defeats the contract). Mitigation: ADR-4's recover→1. Pin with a note in the recover handler; consider a test that a panicking command exits 1, not 2.
- **Reclassifying dispatch's default arm ripples beyond the two named tests** — medium likelihood. Mitigation: the deferral was designed and test-named for this change; grep `Success` assertions in `dispatch_test.go`/`dispatch_bdd_test.go` before landing (LEARNINGS' "audit both the unit tree and the BDD harness for parity" applies).
- **`main`'s old placeholder mapped *all* errors to exit 1** — usage errors will now exit `2` (intended), so any external consumer or script asserting "non-zero == 1" sees a behavior change. Impact low (no test pins `main`; it is the documented placeholder being replaced), but worth calling out in the PR.
- **Unused operational code constants read as incomplete** — low impact. Mitigation: ADR-2's explicit registry comment framing them as reserved-for-the-API-client, plus the exact-values pin test that gives them a present-day purpose.

---

## What This Plan Does Not Cover

- **The protocol-level exit-code contract table** (the published category↔code surface an agent reads) — that is the **interface** skill's artifact (`interface-cli.md`): the process-exit boundary named in the spec's Integration Boundaries is an external-facing surface.
- **Executable scenarios** — the **scenarios** skill concretizes the spec's driving scenarios into `features/no-runnable-cli.feature` and step bindings.
- **Task decomposition** — the **tasks** skill turns the four implementation steps above into PR-sized units with acceptance criteria.
- **The API client's HTTP-status→category classification** — deferred to the future API-client spec (the producer of categories 3–6); 004 only reserves their codes.
