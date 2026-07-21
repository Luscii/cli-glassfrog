# Interface Accord: Operating-Surface Packaging — Specification

**Feature**: 070-operating-surface-packaging
**Role**: Crafter
**Touchpoint**: Specification
**Plan reference**: the marketplace manifest, `glassfrog-setup` skill, and consistency-guard boundaries in plan.md's System Architecture (ADR-1/-2/-3/-4/-5)
**Inputs**: spec.md, plan.md, PROJECT.md; the marketplace-manifest contracts are grounded against real installed marketplaces under the developer's `~/.claude/plugins/marketplaces/` (the private Luscii marketplace, learning-opportunities, claude-plugins-official) — external reference examples, **not** files present in this repository

> The artifacts *are* the interface: a marketplace manifest consumed by the Claude plugin host, a setup skill consumed by agents on demand, and a guard contract consumed by CI. No runtime surface is added — the feature adds no `glassfrog` subcommand, endpoint, or event.

---

## Surface

### Invocation

| Consumer | Entry point | Trigger |
|---|---|---|
| Claude Code plugin host | `.claude-plugin/marketplace.json` (repo root) | `/plugin marketplace add Luscii/cli-glassfrog` reads the manifest; `/plugin install glassfrog@glassfrog` installs the listed plugin from `./plugin` |
| AI agent | `plugin/skills/glassfrog-setup/SKILL.md` | The frontmatter `description` matches a provisioning need — loaded on demand, like every sibling skill |
| CI | `internal/build` guard test | Ordinary `go test ./...` run |

There are no flags or arguments anywhere — none of the three artifacts has a parameterized entry point.

### Structural layout (required files)

```
.claude-plugin/
  marketplace.json                       # the marketplace manifest (required, repo root)
plugin/
  skills/
    glassfrog-setup/
      SKILL.md                           # the setup skill (required)
internal/build/
  operatingsurfacepackaging.go           # path constants + parse helpers (production source)
  operating_surface_packaging_guard_test.go  # best-effort consistency guard
```

Nothing inside `plugin/` changes except the one added skill directory — `plugin/.claude-plugin/` stays marketplace-free (062's "deliberately absent" holds for that path; the marketplace lives at the repo root, where a plugin host looks when the *repo* is added as a marketplace).

### `marketplace.json` schema

Grounded on the installed Luscii / learning-opportunities / claude-plugins-official marketplaces.

| Field | Type | Required | Notes |
|---|---|---|---|
| `$schema` | string | recommended | `https://anthropic.com/claude-code/marketplace.schema.json` (as in learning-opportunities and the official directory) |
| `name` | string | yes | **`glassfrog`** (ADR-1) — the identity in `/plugin install glassfrog@glassfrog` |
| `owner` | object `{name, email?}` | yes | `{ "name": "Luscii" }`, matching `plugin.json`'s `author` |
| `metadata` | object `{description?, version?}` | optional | may carry a one-line marketplace description; no marketplace-level version is required |
| `plugins` | array | yes | list-shaped (ADR-2) — one entry today, siblings appended later |

**The glassfrog entry** (`plugins[0]`):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `glassfrog` — must equal `plugin/.claude-plugin/plugin.json` `name` (guarded) |
| `source` | string | yes | `./plugin` — relative path that must resolve to the directory containing the plugin manifest (guarded) |
| `description` | string | yes | **verbatim-equal** to `plugin.json`'s `description` (guarded) — the checkable form of ADR-2's "consistent" |
| `version` | — | **absent** | deliberately no pin: the source is in-repo, so the installed version is the checkout's; `plugin.json` stays the single version source (ADR-2) |

### `SKILL.md` frontmatter (`glassfrog-setup`)

YAML frontmatter matching the sibling-skill convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `glassfrog-setup` |
| `description` | string | yes | **The trigger.** Must fire on provisioning needs — a fresh agent environment, the plugin just installed, "set up glassfrog", a `glassfrog` command-not-found failure, or an authentication failure at session start — and must not fire on ordinary operating questions (orientation's territory) |

### Required sections in `SKILL.md` (the setup journey)

| Section | Must convey |
|---|---|
| Presence check | Instruct the agent to run an existing innocuous command (`glassfrog --version`-style) and interpret: runs → CLI present; command not found → missing-CLI fix |
| Auth check | Instruct the agent to run a low-cost authenticated identity read (`glassfrog me`) and interpret: exit 0 → authenticated; non-zero → failing-credential fix (exit-code semantics defer to orientation) |
| Missing-CLI fix | Name the three shipped install channels — install script, Homebrew tap, npm wrapper — with their canonical invocations **sourced from README's Installation section**; never a bundled binary or a bespoke install path |
| Failing-credential fix | Walk the operator toward the CLI's existing `X-Auth-Token` setup; point at the CLI/orientation for the mechanism — the skill stores and validates nothing itself |
| Ready report | When both checks pass, state the environment ready to drive the CLI through the operating surface |
| Boundary note | Setup owns the *journey* (check → fix → verify); orientation owns the *reference* (credentials, exit codes, output formats) — setup links, never restates |

### Guard contract (`internal/build`)

`operatingsurfacepackaging.go` exports the path constants (`.claude-plugin/marketplace.json`, `plugin/.claude-plugin/plugin.json`, `plugin/skills/glassfrog-setup/SKILL.md`) and the parse helpers — production source, not `_test.go`, per the family convention. The guard test asserts:

| Assertion | Both sides derived from |
|---|---|
| The manifest parses and a `plugins` entry named `glassfrog` **exists** (never "is the only entry" — generality is preserved) | `marketplace.json` |
| The entry's `source` resolves to a directory containing the plugin manifest | `marketplace.json` ↔ filesystem |
| Entry `name` equals plugin `name`; entry `description` equals plugin `description` | `marketplace.json` ↔ `plugin.json` |
| The entry carries no `version` key | `marketplace.json` (contract-fact of ADR-2) |
| The setup skill file exists with non-empty frontmatter `name` (`glassfrog-setup`) and `description` | `SKILL.md` |
| Enumerable setup facts anchor: the three channel names appear in the skill; the auth-check leaf (`me`) resolves in the CLI command registry | `SKILL.md` ↔ registry/README |

Explicitly partial: it verifies in-repo consistency and enumerable facts, not the host's marketplace schema, prose quality, or flag detail — and any coverage reduction must be stated in the test, never silent.

---

## Interactions

**Install flow** (discover → install → run):

1. The operator (or agent) runs `/plugin marketplace add Luscii/cli-glassfrog`; the host reads the repo-root manifest and registers the `glassfrog` marketplace.
2. `/plugin install glassfrog@glassfrog` resolves the entry's `./plugin` source and installs the plugin.
3. The host auto-discovers the plugin's skills, agents, and hook exactly as it does today for a locally-present plugin — packaging changes nothing downstream of install.

**Setup journey** (the skill's instructional model):

1. The skill triggers on a provisioning need.
2. Presence check → on failure, the missing-CLI fix (three channels), then **re-check** — the journey loops until the check passes or the operator stops.
3. Auth check → on failure, the failing-credential fix, then re-check.
4. Both green → ready report. The two failure classes stay distinct end-to-end: missing binary never routes to credential guidance, and vice versa.

**Second-plugin flow** (future): a sibling plugin is one appended `plugins[]` entry (in-repo relative path or cross-repo source). No other file changes; the guard's existence-based lookup is unaffected by additional entries.

---

## Error Communication

Specification artifacts fail as **constraint violations**, not runtime errors:

| Condition | Behavior |
|---|---|
| `marketplace.json` missing or malformed | The host cannot add the marketplace; nothing installs. The guard fails first in CI (parse assertion) |
| Entry `source` does not resolve to the plugin | Install fails at the host; guard red (resolution assertion) — the spec's "defect, not difference" accord |
| Entry `name`/`description` drifts from `plugin.json` | Guard red (equality assertions) |
| A `version` key appears on the entry | Guard red — the single-version-source contract broke |
| `SKILL.md` missing, or frontmatter lacks `description` | The setup skill never triggers — setup is effectively absent; guard red (frontmatter assertion) |
| Setup skill names a channel or command the repo/CLI lacks | Enumerable-fact anchor red for the guarded part; review catches the rest |
| Guard coverage reduced or omitted | Permitted only if stated in the test — never silently |

Runtime error shapes are **N/A** — the CLI's own exit codes and failure rendering (015/031/032) are consumed by the setup skill's instructions, not redefined.

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint.
- **Follows the installed Luscii-marketplace form** (owner object, `plugins` array, relative `./` sources) and adopts `$schema` from the learning-opportunities/official examples.
- **Flagged deviation — no per-entry `version`**: the private Luscii marketplace pins a version on every entry; this manifest deliberately does not (ADR-2). Reasoning: those entries version plugins vendored *into* the marketplace repo, whereas here the source is the same checkout the marketplace ships in — a pin would be a second version source drifting from `plugin.json` on every bump.
- **Supersedes 062's entry-shape projection**: 062's Consistency Notes sketched the future marketplace entry as `{name, version, description, source}`; ADR-2 drops `version` from that shape. The other three fields land as projected.
- **Guard extends the family pattern** (§417/§426/§438): helpers in production source, both sides derived at test time, explicitly-partial coverage with stated reductions — plus one new anchor type: identity equality across two committed manifests.
- **Skill conventions**: frontmatter is `name` + `description` only; the trigger-precision expectation mirrors orientation's (fire on the need, not spuriously) with the added negative boundary against orientation's own territory.
