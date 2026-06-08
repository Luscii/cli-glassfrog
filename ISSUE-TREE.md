# Glassfrog CLI — Issue Tree

Problems for the Glassfrog v5 CLI, decomposed into a project foundation, a shared client foundation, and the endpoint commands that are the project's endgoal. Bootstrapped from the 2026-06-03 explore session.

> How can practitioners — and the AI agents acting for them — read and change Glassfrog governance from the command line, faithfully to the v5 API?

---

* Project Foundation — prerequisites for the CLI to exist and run before any command can be built
  * No Runnable CLI — no project skeleton or command framework exists to build any command on
    + affects: Maintainer
  * Runtime-Dependent Distribution — the CLI can't run without a separately-installed runtime, so it won't run where operators need it
    + affects: Practitioner
    + affects: AI agent
    + affects: Maintainer
  * No Automated Pipeline — without CI/CD, every change is linted, tested, triaged, and released by hand, so regressions and inconsistencies can reach main and releases go out unguarded
    + affects: Maintainer
    + related-to: Runtime-Dependent Distribution
    + candidate: PR Checks Workflow — on pull request, run lint + tests and auto-apply administrative labels (pattern: template pull-request.yml)
    + candidate: Release Drafting on Merge — on merge to main, draft a GitHub release (release-drafter, label-driven semver) and re-run tests as a safety check (pattern: template draft-release.yml)
  * Shallow AI Reviews — Copilot reviews PRs without a configured project environment, so its feedback is poorly grounded and misses project-specific context
    + affects: Maintainer
    + candidate: Copilot Review Environment — configure the repo's Copilot environment (build/test/context setup) so its PR reviews are better-grounded
* Client Foundation — cross-cutting concerns every command depends on
  * Unauthenticated Access — the CLI has no way to prove it's acting as a specific org + person, so Glassfrog can't authorize its calls
    + affects: AI agent
    + affects: Practitioner
  * Undefined Connection Settings — the CLI doesn't know which token, organization, or base URL to use, or where to read them from
    + affects: Practitioner
  * Duplicated Setting Resolution — each configurable setting (token, base URL, output format) re-implements the same flag→env→.glassfrogrc→default precedence chain, so adding a setting copies the OS seam, the chain skeleton, the Source enum, and the validation-error shape — and the copies can drift
    + affects: Maintainer
    + related-to: Undefined Connection Settings
    + related-to: Unconsumable Output
    + candidate: Composable setting resolver — resolveSetting([...sources]) takes an ordered list of source providers and returns the first that yields a value, where ARRAY ORDER IS PRECEDENCE (flag, env, file); sources are fromFlags(main, alias), fromEnv(envVar), fromFile(key) with the file source partially applied from retrieveFromSettingsFile(".glassfrogrc") so the file is bound once; a source yielding nothing is skipped, an optional trailing default source backstops the chain (absence stays valid where there is no default, e.g. token), and per-source validation collapses the per-type Source enums + validation errors into one shared shape; retrofit the 3 call sites (token 005, base-url 008, output 020), behavior-preserving with tests staying green; the .glassfrogrc file tier (rcfile.Resolve) is already shared and stays as the file source
  * Unconsumable Output — results aren't shaped for an AI agent to parse reliably or for a human to read
    + affects: AI agent
    + affects: Practitioner
    + candidate: --output flag — caller selects the output format per invocation: human-readable (full/compact), JSON, possibly YAML
  * Opaque Failures — when a call fails, the caller can't tell what went wrong or what to do next
    + affects: AI agent
    + affects: Practitioner
  * Unsignalled Plan Limits — plan/flag-gated endpoints (ai_integration, Premium) reject with 403 and no clear "not available on your plan" signal
    + affects: Practitioner
  * Silent Truncation — large result sets and the org tree get cut off without the caller knowing
    + affects: AI agent
    + affects: Practitioner
  * Getting Throttled — unmanaged request volume trips the per-org hourly rate limit, throttling the whole organization
    + affects: Practitioner
  * Clobbered Changes — concurrent governance edits overwrite each other when writes skip version checks
    + affects: Practitioner
* Endpoint Commands — expose each in-scope v5 operation as a command (the endgoal)
  * Governance Reads — read roles, circles, accountabilities, domains, policies, and projects
    + affects: Practitioner
  * Self-Service Reads — read "what's mine": me, my roles, actions, and projects
    + affects: Practitioner
  * Tension Capture — record a tension as the entry point to a proposal
    + affects: Practitioner
  * Proposal Write-Flow — the multi-step governance write path: create → propose → respond → accepted
    + affects: Practitioner
    + related-to: Unsignalled Plan Limits
