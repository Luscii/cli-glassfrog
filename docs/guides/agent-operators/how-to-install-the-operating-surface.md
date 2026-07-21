<!-- Source: docs authored during Score implement
     Capability: Agent Operating Surface
     Contributing specs: 070-operating-surface-packaging
     Artifacts: .claude-plugin/marketplace.json, plugin/skills/glassfrog-setup/SKILL.md, interface-spec.md, features/unequipped-agent-operators/operating-surface-packaging.feature
     Generated: 2026-07-21
     Type: how-to -->

# How to install the agent operating surface

This guide is for whoever provisions an AI agent to drive the `glassfrog` CLI —
or for the agent itself. It shows how to get the **agent operating surface** (the
Claude plugin that 062–069 built: orientation, the operator-path skills, and the
write-safety hook) into an agent environment from this repository, and how to go
from "plugin installed" to "ready to drive the CLI". For how to *operate* once
installed, see [How to operate the glassfrog CLI safely as an agent](how-to-operate-safely.md).

## Prerequisites

- A Claude plugin host (e.g. Claude Code) in the agent's environment.
- The `glassfrog` **CLI is a prerequisite, not part of the plugin** — the plugin
  is knowledge and guardrails riding on top of the binary. Install it through
  any of the CLI's existing channels: the
  [install script](../../../README.md#installation), the
  [Homebrew tap](../installation/how-to-install.md), or the
  [npm wrapper](../../../README.md#install-via-npm). (The setup skill below
  will catch a missing binary and point you here.)

## 1. Add the repository as a plugin marketplace

The repo ships a Claude plugin marketplace named `glassfrog` at
`.claude-plugin/marketplace.json`. Register it in the host:

```
/plugin marketplace add Luscii/cli-glassfrog
```

The host reads the manifest from the repository and lists the `glassfrog`
plugin as an installable entry.

## 2. Install the glassfrog plugin

```
/plugin install glassfrog@glassfrog
```

(The plugin is named `glassfrog`; the marketplace is too — hence
`glassfrog@glassfrog`.) The entry's source points at the repo's `plugin/`
directory, so the installed version is whatever the checkout carries — there is
no separate version to pick. The host auto-discovers the plugin's skills,
agents, and the write-safety hook; nothing else needs configuring.

## 3. Go from installed to ready with the setup skill

A fresh environment usually still lacks the binary, the credential, or both.
The plugin's `glassfrog-setup` skill owns that journey: it checks the CLI is
present and the credential works, directs you to the right fix when either
fails (the CLI's install channels for a missing binary; the CLI's own
`glassfrog auth login` for a failing credential), re-checks after each fix, and
reports the environment ready when both checks pass. Invoke it on any
provisioning need — right after install, on a `glassfrog: command not found`,
or on an authentication failure at session start.

## Verify

- `/plugin marketplace add Luscii/cli-glassfrog` registers a marketplace named
  `glassfrog` listing one plugin, also `glassfrog`.
- After `/plugin install glassfrog@glassfrog`, the orientation and operator-path
  skills are available and proposal writes are gated by the write-safety hook.
- The setup skill reports ready once `glassfrog --version` runs and an
  authenticated identity read succeeds.

## Troubleshooting

**The host cannot find the marketplace.** The manifest lives at
`.claude-plugin/marketplace.json` in the **repo root** (not under `plugin/`).
Make sure the marketplace add names this repository, `Luscii/cli-glassfrog`.

**The listed plugin doesn't match what installs.** The marketplace entry and
the plugin's own manifest (`plugin/.claude-plugin/plugin.json`) must agree — a
mismatch is a **defect to fix**, not a tolerable difference. A consistency
guard in `internal/build` fails the build when the entry's name, description,
or source drifts from the plugin definition, or when a version pin appears on
the entry (the version is single-sourced in `plugin.json`).

**Installed, but `glassfrog` commands fail.** That is exactly the setup skill's
territory (step 3): a missing binary routes to the install channels, a failing
credential routes to `glassfrog auth login` — the plugin never installs a
binary or stores a credential of its own.
