# Interface Accord: Operator Orientation — Specification

**Feature**: 062-operator-orientation
**Role**: Crafter
**Touchpoint**: Specification
**Inputs**: spec.md, plan.md, PROJECT.md; the plugin/manifest contracts are grounded against real installed Claude plugins under the developer's `~/.claude/plugins/` (e.g. score, prelude, learning-opportunities) and a Luscii marketplace manifest found there — external reference examples, **not** files present in this repository

> The artifact *is* the interface: a Claude plugin (manifest + one skill) consumed by reading and on-demand loading, not by a runtime call. Protocol-level contracts are the `plugin.json` field set, the `SKILL.md` frontmatter, and the orientation content's required sections.

---

## Surface

### Invocation

The plugin exposes **no CLI-style entry point**. It is consumed two ways:

| Consumer | Entry point | Trigger |
|---|---|---|
| Claude Code plugin host | `plugin/.claude-plugin/plugin.json` | Host loads the manifest and auto-discovers skills under `plugin/skills/` |
| AI agent | `plugin/skills/glassfrog-operator/SKILL.md` | The skill's frontmatter `description` matches an agent need ("how do I drive this CLI / parse output / react to this exit code") — loaded **on demand**, not at session start |

There are no flags or arguments.

### Structural layout (required files)

```
plugin/
  .claude-plugin/
    plugin.json                      # plugin manifest (required)
  skills/
    glassfrog-operator/              # the one orientation skill (working name)
      SKILL.md                       # the orientation content (required)
internal/build/
    operator_orientation_guard_test.go   # best-effort drift guard (companion, not part of the plugin package)
```

`plugin/.claude-plugin/marketplace.json` is **deliberately absent** — distribution is #70.

### `plugin.json` schema

Grounded on the score/prelude manifests (author object form, `keywords` array, **no `skills` array** — skills are auto-discovered from `skills/`).

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | kebab-case plugin id — **`glassfrog-operator`** `[ASSUMED]` |
| `version` | string (semver) | yes | starts at `0.1.0` |
| `description` | string | yes | one line — what the operating surface provides |
| `author` | object `{name, url?}` | recommended | `{ "name": "Luscii" }` to match sibling plugins |
| `keywords` | string[] | optional | e.g. `["glassfrog","cli","holacracy","agent","operator"]` |
| `homepage` / `repository` / `license` | string | optional | follow sibling-plugin convention if added |

No `skills`, `commands`, `agents`, `hooks`, or `mcpServers` keys in this spec — the single skill is discovered by directory convention; the guardrail's potential `hooks/` and the paths' skills are #63 / #64–#69.

### `SKILL.md` frontmatter

YAML frontmatter, matching the score-skill convention (`name` + `description` only):

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | `glassfrog-operator` `[ASSUMED]` |
| `description` | string | yes | **The trigger.** Must state *when* to consult it and name the CLI-driving topics (parsing output, pagination, exit-code reactions, credentials, write-safety) so it fires on the right need and not spuriously |

### Required sections in `SKILL.md` (the orientation content)

The body packages cross-cutting operating knowledge only. Required topic sections:

| Section | Must convey |
|---|---|
| Output for parsing | How to select a structured format and the shape to expect. Format tokens are exactly `full, compact, json, yaml` (source: `internal/output` `supportedFormats`); `json`/`yaml` are the machine-parseable pair |
| Pagination | How to detect that more pages exist and fetch them (source: `internal/paging`) |
| Exit-code reactions | What each code in the **0–7** convention means and the reaction — incl. `StaleWrite`=7 for `412` (source: `internal/cli` `Outcome`→`ExitCode`) |
| Credentials | How the CLI discovers/accepts the `X-Auth-Token` key; points at the existing `glassfrog auth login` — introduces no new credential mechanism |
| Write-safety expectation | Confirm before writing; on a `412` stale-write, re-read and re-confirm rather than blind-retry — stated as **guidance**, with an explicit "this skill does not enforce it" note |
| Driving the command surface | Points the agent at `glassfrog help` / `--help` for per-command and per-flag detail — does **not** catalogue commands |

---

## Interactions

**Invocation-to-output flow** — there is no produced artifact; the "output" is the agent's correctly-driven CLI call:

1. The plugin host reads `plugin.json` and discovers the skill under `skills/`.
2. The skill is registered with its `description` as the trigger surface.
3. While operating, the agent hits something it doesn't know about driving the CLI.
4. The need matches the skill `description`; the host loads `SKILL.md` on demand.
5. The agent reads the relevant section and drives the real `glassfrog` CLI (which calls the Glassfrog API exactly as today).

**Instructional model**: the skill tells the agent *how to operate*, not *what governance to perform*. For anything below cross-cutting (a specific command's flags), it routes to `glassfrog help`. For write operations it states the safety expectation but stops short of acting on it.

---

## Error Communication

Specification artifacts fail as **constraint violations**, not runtime errors:

| Condition | Behavior |
|---|---|
| `plugin.json` missing or malformed | Host cannot load the plugin; no skill is available; the agent falls back to rediscovery (nothing in the CLI breaks) |
| `SKILL.md` missing, empty, or lacking a `description` | Skill is not discoverable / will not trigger — the orientation is effectively absent |
| Drift guard fails (`internal/build` test red in CI) | Signals the skill's **enumerable** facts (format tokens, exit-code numbers/labels, `auth login` existence) diverged from the shipped CLI — the truthfulness contract is broken; fix the skill (or the claim) |
| Drift guard coverage reduced/omitted | Permitted (spec assumption: partial/none acceptable) **only if stated**, never silently — the test must name what it does *not* cover |
| Skill names a command/flag/format the CLI lacks | Invented-surface defect — caught by the anchor test for the enumerable part, by review for the rest |

Runtime error shapes (request/response bodies, HTTP status rendering, rate-limit handling) are **N/A** — the plugin has no runtime surface; those belong to the CLI it describes (specs 015/017/031/032).

---

## Consistency Notes

- **Sibling interface files**: none — this feature has only a specification touchpoint (no API/CLI/UI/events surface; it adds no `glassfrog` subcommand).
- **Follows plugin conventions** observed in `score`/`prelude`: `author` as an object, `keywords` array, and **skill auto-discovery** (no `skills` array in the manifest). Version starts at `0.1.0` per the spec's pre-1.0 surface.
- **Distribution deferred to #70** (plan ADR-5): when #70 lands it adds a `{ name, version, description, source }` entry for this plugin to a marketplace manifest (e.g. the Luscii marketplace seen in installed plugins under `~/.claude/plugins/` — no such file exists in this repo today) or ships the plugin's own marketplace. The shape is known; producing it here is out of scope and is asserted absent by a validation scenario.
- **Deviation from sibling acquisition channels** (npm §318 / homebrew §316): those are CI-*generated/published* binary channels; this plugin is **committed, hand-authored content** (plan ADR-3). Same "repo-shipped acquisition channel" family, different production mode — flagged so #70 doesn't assume a generated path.
- **Naming `[ASSUMED]`**: plugin id and skill name `glassfrog-operator` are working names; reversible (consumers trigger by `description`, not a fixed file path). Adjust freely before implementation.
