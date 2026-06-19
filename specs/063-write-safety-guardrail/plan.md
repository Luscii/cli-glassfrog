# Plan: Write-Safety Guardrail

**Feature**: 063-write-safety-guardrail
**Role**: Shaper
**Inputs**: spec.md, PROJECT.md, DECISIONS.md (targeted precedent grep — §398/§399 059's deferral of destructive-write confirmation to *this* guardrail "centrally, across the write path, not per-command"; §409/§411 062 plugin home + `plugin/hooks/` extension point + additive sibling growth; §414 hand-authored-content / defer-to-`--help`; §417 best-effort `internal/build` drift guard; §406 061 recognizer-keys-on-operation + central-refinement shape), DEPRECATION.md (no relevant entries)

---

## System Architecture

Like Operator Orientation (062), this feature adds **no code to the Go CLI** and introduces no API capability. It lives entirely in the **operator layer** — the Claude plugin under `plugin/` that 062 established — and it enforces a gate on commands the CLI already exposes.

The enforcement seam is a **Claude Code `PreToolUse` hook** bundled in the plugin. When the agent operator is about to run a shell command (the `Bash` tool), the hook fires *before* the command executes, inspects the command, and — when it recognizes a governance write on the proposal write path — returns a permission decision that routes the call to the plugin host's **human-confirmation** prompt. The practitioner approves or declines; the agent cannot self-authorize. Reads and operational tension edits are recognized as not-gated and pass straight through.

Three parts:

1. **The gate hook** — a self-contained, dependency-free `PreToolUse` hook script under `plugin/hooks/`, registered through the plugin's hook configuration. It reads the tool-call JSON on stdin, and for a `Bash` call it parses the command line to decide whether the invocation is a proposal-path governance write.

2. **The gated-command registry** — a single, machine-readable list of the proposal write leaves the hook gates (`proposal create`, `proposal propose`, `proposal respond`, `proposal withdraw`). The hook reads it to classify a command; the drift test (below) reads the *same* source so the two never disagree. Tension edits (`tension create/update/discard`) and all reads are deliberately absent from the registry.

3. **The drift tripwire** — a best-effort test in `internal/build` (the project's existing config-guard home) that asserts the CLI's `proposal` subcommand surface still matches what the registry assumes, so a newly-added or renamed proposal write command can't silently ship ungated.

Data flow: there is no new runtime path inside the CLI. The flow is `agent → Bash(glassfrog proposal …) → host fires PreToolUse hook → hook classifies → ask (human confirms) → command runs → CLI → API`, exactly as today for everything the hook lets through unchanged. The re-read-and-re-confirm cycle on a stale write (`412`, exit `7`) is the same loop: the failed write's retry is itself a proposal-path write, so the hook gates it again — a blind retry can't slip past.

---

## Architecture Decisions

### ADR-1: Enforce at the operator layer with a `PreToolUse` hook — not a skill, and not a CLI prompt/flag

**Context**: The spec demands *enforcement* — "gate," "the agent cannot self-authorize," "never reached without asking." Two precedents bound the solution. §399 (059-withdraw-proposal) explicitly **rejected** CLI-level confirmation (an interactive prompt or a `--force`/`--yes` flag) because it "breaks the non-interactive, agent-driven contract — an agent cannot answer a prompt — and would be the first write to gate itself, forking the uniform write surface," and deferred destructive-write confirmation to *this* guardrail to "gate destructive writes ACROSS the write path centrally, not per-command." §409 (062 ADR-1) reserved `plugin/hooks/` as the guardrail's likely home. The spec's `[ASSUMED] delivered as a skill` was flagged as a shaping decision.

**Options considered**:
1. **Skill-only (instructional)** — a `SKILL.md` that tells the agent to confirm before writes. Matches the spec's assumption and 062's pattern, but a skill is *guidance the agent may ignore* — it cannot enforce, so it fails the spec's "cannot self-authorize" requirement and barely differs from 062's existing write-safety guidance.
2. **CLI-level gate** (`--force` / interactive prompt in the write commands) — enforced at the source, but this is exactly what §399 rejected: it forks the uniform, non-interactive CLI write surface and an agent can't answer a prompt.
3. **Operator-layer `PreToolUse` hook** — the plugin host intercepts the agent's `Bash` call before it runs and can route it to a human confirmation. Genuine, central, per-write-path enforcement that lives off the CLI entirely.

**Decision**: Option 3 — a `PreToolUse` hook in the plugin. It is the only option that *enforces* (the agent physically cannot run the command until the host's decision resolves), it gates centrally across the write path rather than per-command (honoring §399), and it keeps the CLI's write surface uniform and untouched. This resolves the spec's `[ASSUMED] skill` to **a hook** — the orientation *skill* (062) already carries the write-safety *guidance*; this capability adds the *enforcement* the skill could never provide.

**Consequences**: The deliverable is a hook script + its registration, not a skill. Enforcement depends on the agent running inside a plugin host that supports `PreToolUse` hooks (see R2). In a headless/autonomous session with no human to answer, a gated write blocks pending approval — that is the guardrail working as intended (VISION 2), not a defect. The spec's "delivered as a skill" assumption is superseded and must be reconciled (noted for handoff).

**Constitution reconciliation** (II Action Transparency / IX Writes Require Explicit Intent — "When Principles Conflict", row 2): the constitution resolves that *intent is expressed by the explicit write command itself, not by an interactive prompt, so an agent can act without a human in the loop*. That resolution is **CLI-scoped** — it keeps the **CLI's** write surface non-interactive so an agent can drive it (the same contract §399 preserved). This gate adds a human checkpoint at the **operator/host layer** and leaves the CLI's command-is-intent contract untouched: an agent driving the bare CLI (no plugin) still acts without a prompt. The operator-layer gate therefore realizes VISION principle 2 *without* breaching the constitution's CLI-scoped resolution, so no amendment is required. If the team instead reads that resolution as binding on the whole operating surface (not just the CLI), a constitution amendment with a version bump is the prerequisite before this lands — a decision for the developer, surfaced by the pre-implementation guard's P1.

### ADR-2: Gate by returning `permissionDecision: "ask"` — reuse the host's human checkpoint

**Context**: Clarify resolved the confirmation model to **A**: a mandatory human-in-the-loop checkpoint per governance write; the agent cannot self-authorize. The plugin host's `PreToolUse` hook protocol lets a hook return a permission decision of `allow`, `deny`, or `ask`.

**Options considered**:
1. **`deny` + conversational instruction** — block the call and tell the agent to ask the practitioner in chat, then re-issue. Works, but it relies on the agent faithfully relaying and interpreting a free-text exchange — softer, and it re-introduces agent judgment into the gate.
2. **`ask` (host confirmation prompt)** — the host surfaces the exact command to the human and runs it only on explicit approval. The decision is the human's, made against the literal command; the agent neither answers nor can bypass it.

**Decision**: Option 2 — return `permissionDecision: "ask"` with a reason that names the write (command, target, change-bearing flags) for the human to review. This *is* the human-in-the-loop checkpoint Q1=A chose, realized through the host's built-in confirmation UI rather than an invented one.

**Consequences**: The "surface what I'm about to do, in terms the practitioner can review" behavior is satisfied by the host rendering the command + the hook's reason string. No bespoke confirmation surface is built. On decline, the host blocks the write and the record is unchanged. The reason string's content (what it names) is an interface-level contract.

### ADR-3: Recognize governance writes with a static command-leaf registry, fail-closed within the `proposal` namespace

**Context**: There is no pre-execution signal from the CLI about whether a command writes (adding one would be new CLI capability — forbidden). The hook must classify from the `Bash` command string alone. §406 (061) set the "recognizer keys on the operation, classified centrally" pattern; 060's static gate registry is the same shape.

**Options considered**:
1. **Heuristic match** (e.g., any command containing `propose`/`withdraw`) — brittle: false-positives on unrelated commands and reads, false-negatives on aliases.
2. **Static registry of proposal write leaves** — parse the command to find the `glassfrog` invocation and its subcommand path, then match against an explicit set `{proposal create, proposal propose, proposal respond, proposal withdraw}`. Conservative and legible.

**Decision**: Option 2 — a static registry of the four proposal-write leaves (matching Q2=B's proposal-path-only scope). The recognizer resolves the `glassfrog` token (handling an absolute path, an env-var prefix, and arguments) and the subcommand path, then gates iff the path is a registered write leaf. Within the `proposal` namespace it is **fail-closed**: an *unrecognized* `proposal` subcommand is gated rather than waved through, so a future write leaf is safe-by-default until the registry is updated. Non-`glassfrog` commands, reads, and `tension` edits pass.

**Consequences**: The gate is precise about the proposal write path and never blocks reads or tension edits. The registry is the one place that encodes "what is a governance write," which makes it the drift surface ADR-4 guards. Robust command-string parsing (pipes, chaining, quoting) is an implementation concern with integrity stakes (see R1).

### ADR-4: Single-source the registry and guard drift with a best-effort `internal/build` tripwire

**Context**: If the CLI gains or renames a proposal write command and the hook's registry isn't updated, a governance write ships **ungated** — an integrity hole. §417 (062 ADR-4) established a best-effort `internal/build` config-drift test (extending §175/§203/§309/§316) as the project's idiom for exactly this class of risk.

**Options considered**:
1. **Rely on review** — cheapest, but review misses surface changes, and the cost here is a silent integrity gap.
2. **Auto-classify CLI commands as read/write** — would need the CLI to declare write-ness (new capability — forbidden) or a fragile heuristic that re-introduces the drift it's meant to catch.
3. **Tripwire over the `proposal` subcommand set** — a test that enumerates the CLI's `proposal` subcommands and fails when that set changes against a checked-in expectation, forcing a conscious human reclassification + registry update.

**Decision**: Option 3 — a best-effort tripwire in `internal/build`. The gated-command registry is stored once in a machine-readable form both the hook and the test consume; the test asserts (a) the registry's leaves still exist on the CLI's `proposal` command and (b) the `proposal` subcommand surface hasn't grown/renamed beyond the expectation without the registry being updated.

**Consequences**: A new or renamed proposal write command breaks the build until someone classifies it and updates the single registry — closing the silent-ungated-write hole. Like 062 ADR-4 it is explicitly *partial* (it pins the enumerable surface, not the hook's parsing robustness); a reduction in coverage is acceptable but must be stated, never silent (LEARNINGS: no silent caps).

### ADR-5: Do not re-author the `412` re-read guidance — enforce re-confirmation on the retry instead

**Context**: The spec's stale-write accord says the guardrail re-reads and re-confirms rather than blind-retrying. The *re-read* is operator behavior; 062's orientation skill already states the `412` re-read-and-re-confirm expectation as guidance (§414 — orientation owns this, and content must not be duplicated).

**Options considered**:
1. **Re-author the re-read protocol in a 063 skill** — duplicates guidance 062 already single-sources, creating a drift surface between two copies.
2. **Lean on 062's guidance + let the hook enforce re-confirmation** — the hook gates *every* proposal-path write, so a retry after a `412` is gated again; "no blind retry" holds structurally, and the re-read step stays single-sourced in orientation.

**Decision**: Option 2. 063 adds enforcement (the hook), not new guidance. "No blind retry" is enforced because the retry is itself a gated write requiring fresh human confirmation; the re-read remains 062's orientation guidance.

**Consequences**: 063 ships no new skill content; the orientation skill is the single home for the re-read protocol. The hook does not itself perform or verify the re-read (it can't inject state) — it guarantees the retry can't run un-confirmed. If a future need arises to *enforce* the re-read specifically, that is a later, separate refinement.

---

## Plugin Structure (Specification Boundary)

This feature's boundary is a **specification boundary** plus a **host-integration boundary** — it produces declarative artifacts an external consumer (the Claude Code plugin host) loads and executes:

- **What it produces**: a `PreToolUse` hook script under `plugin/hooks/`, the plugin hook-registration entry that wires it to the `Bash` tool, the single gated-command registry, and the `internal/build` drift test.
- **What the consumer expects**: a hook the host invokes on `PreToolUse` with the documented tool-call JSON on stdin, returning the documented permission-decision JSON.
- **Invocation surface**: the host fires the hook; there is no CLI-level invocation surface and no new `glassfrog` command or flag.

Protocol-level detail — the exact hook-registration schema, the stdin/stdout JSON contract (`hookSpecificOutput.permissionDecision` plus the human-facing message field, whose exact spelling is host-specific and is pinned in `interface-spec.md`), the registry file format and location, and the precise content of the confirmation reason string — is the **interface** skill's concern (`interface-spec.md`).

---

## Cross-cutting Concerns

- **Integrity / fail-safe** — the dominant concern. A false-negative (an ungated governance write) is an integrity failure; ADR-3's fail-closed-within-`proposal` stance and ADR-4's drift tripwire both bias toward gating. A false-positive (gating a read) is mere friction and is avoided by the precise registry, never by relaxing the gate.
- **Drift** — handled by ADR-4 (single-sourced registry + `internal/build` tripwire), reusing the §417/§175 idiom. Partial by design; reductions stated, not silent.
- **Testing strategy** — three kinds: (1) the `internal/build` drift tripwire (ADR-4); (2) recognizer unit tests over command-string variants — bare/absolute-path/env-prefixed `glassfrog`, each gated leaf, each ungated tension edit, reads, unrecognized `proposal` subcommand (fail-closed), and non-`glassfrog` commands; (3) a static check that the hook is well-formed and registered against the host's documented schema. There is no CLI runtime behavior to integration-test — enforcement lives in the host.
- **Configuration** — none added to the CLI. The hook introduces no credential mechanism (writes still authenticate via the CLI's existing `auth login`).
- **Boundary discipline** — the hook must add no CLI command/flag/capability and must not re-validate the change (the API stays source of truth); it only classifies-and-gates. Reinforced by spec non-behaviors and validation scenarios.

---

## Implementation Strategy

**Phase 1 — Recognizer + registry + hook.** Create the gated-command registry (the four proposal write leaves), the `PreToolUse` hook script that parses a `Bash` command, classifies it against the registry (fail-closed within `proposal`), and returns `ask` for a governance write / `allow`-equivalent (pass-through) otherwise, and the plugin hook-registration entry wiring it to the `Bash` tool. Self-contained; depends only on the interface contract for the registration schema and JSON shapes. This is the bulk of the work.

**Phase 2 — Drift tripwire.** Add the best-effort `internal/build` test asserting the registry's leaves exist on the CLI's `proposal` command and the `proposal` subcommand surface matches the checked-in expectation. Depends on Phase 1 (the registry must exist to anchor against). If anchoring proves harder than expected, reduce scope and state the reduction.

The phases are PR-separable (gate first, guard second); tasks decomposes the specifics.

---

## Risks

- **R1 — Recognizer evasion / mis-parse** (medium likelihood, high impact). A governance write reaches the shell in a form the parser doesn't recognize (chained `&&`/`;`/pipes, quoting, an alias, a wrapper script), so it ships ungated — an integrity hole; or a read is mis-gated (friction). *Mitigation*: ADR-3 resolves the `glassfrog` token and subcommand path rather than substring-matching, is fail-closed within `proposal`, and is covered by recognizer unit tests over command-string variants. Residual exotic-invocation evasion is accepted and noted, not silently assumed away.
- **R2 — Host hook contract is version-specific** (medium likelihood, medium impact). The `PreToolUse` registration schema and the `permissionDecision` protocol depend on the Claude Code plugin host's current conventions (the same external-contract risk as 062 R2). *Mitigation*: interface pins the contract against the host's documented format at design time; treat it as an external contract to revisit if the host changes.
- **R3 — Enforcement absent outside a hook-supporting host** (certain where true, medium impact). If the plugin (or its hooks) isn't installed, or the agent drives the CLI outside a `PreToolUse`-capable host, writes fall back to 062's unguarded, guidance-only behavior. *Mitigation*: this is the documented fallback (spec Integration Boundaries); the guardrail strengthens a present host, it cannot retrofit absent infrastructure. Distribution that gets the hook installed is #70's concern.
- **R4 — Drift guard is partial** (certain, low–medium impact). The tripwire pins the enumerable command surface, not the hook's parsing robustness (R1). *Mitigation*: accepted and stated per 062 ADR-4's precedent; the partiality is explicit, never presented as total coverage.

---

## What This Plan Does Not Cover

- **Exact host/registry contracts** — the hook-registration schema, the stdin/stdout JSON contract, the registry file format/location, and the confirmation reason-string content are the **interface** skill's output (`interface-spec.md`).
- **The `412` re-read guidance** — single-sourced in 062's orientation skill (ADR-5); this plan adds enforcement, not that guidance.
- **Distribution / installation** — getting the plugin (and its hook) onto an agent's host is **Operating-Surface Packaging (#70)**.
- **Spec-assumption reconciliation** — the spec's `[ASSUMED] delivered as a skill` is resolved here to *a hook* (ADR-1); FEATURE-MODEL's gated-set enumeration was already flagged in the spec for reconciliation. Both are noted for the developer, not silently edited.
- **Executable scenarios** and **task decomposition** — the **scenarios** and **tasks** skills.
