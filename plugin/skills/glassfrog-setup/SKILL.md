---
name: glassfrog-setup
description: Provision and verify the environment for driving the glassfrog CLI (the command-line client for the GlassFrog Holacracy API). Use this on any provisioning need - a fresh agent environment, right after installing the glassfrog plugin, a request to "set up glassfrog", a `glassfrog` command-not-found failure, or an authentication failure at the start of a session. It checks that the CLI binary is present and the credential works, directs you to the right fix when either check fails (the CLI's existing install channels for a missing binary; the CLI's own X-Auth-Token setup for a failing credential), re-checks after each fix, and reports the environment ready. Not for how to operate a CLI that already runs - output formats, pagination, exit codes, write-safety, and per-command flags are the orientation skill's territory.
---

# Setting up the glassfrog CLI environment

This skill owns the **journey** from "plugin installed" to "ready to drive the
CLI": check the binary, check the credential, direct each failure to its fix,
verify the fix took, and only then report ready. It installs nothing and stores
nothing of its own — every fix routes to the CLI's existing channels and
mechanisms.

The two failure classes stay distinct end to end: a **missing binary** routes to
the install channels and never to credential guidance; a **failing credential**
routes to the CLI's own credential setup and never to the install channels.

## 1. Presence check

Run an innocuous command that needs no credential:

```
glassfrog --version
```

- It prints a version → the CLI is present. Continue to the auth check.
- The shell reports the command as **not found** → the binary is missing. Go to
  [Missing CLI](#missing-cli-the-install-channels), then **re-run this check**
  before moving on.

## 2. Auth check

Run a low-cost authenticated identity read:

```
glassfrog me
```

- Exit code `0` → the credential works; the output is the identity the CLI
  operates as. Continue to the ready report.
- A non-zero exit → the credential is missing, wrong, or lacks access. Go to
  [Failing credential](#failing-credential-the-clis-own-setup), then **re-run
  this check**. (For what a specific non-zero code means, consult the
  orientation skill's exit-code reference — don't guess from the message.)

## Missing CLI — the install channels

Direct the operator to any one of the CLI's three existing install channels;
they all deliver the same released binary:

- **Install script** (macOS/Linux):

  ```
  curl -fsSL https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh | sh
  ```

- **Homebrew tap**:

  ```
  brew install luscii/cli-glassfrog/glassfrog
  ```

- **npm wrapper** (Node toolchains):

  ```
  npm i -g @luscii-healthtech/glassfrog
  ```

This skill never installs, bundles, or places the binary itself — the channels
own installation. After the fix, **re-run the presence check**: only a passing
re-check moves you forward; a failing re-check routes back here, never to a
ready report.

## Failing credential — the CLI's own setup

The CLI authenticates with an API key sent as the `X-Auth-Token` header, and
the CLI owns that mechanism end to end. Walk the operator through the CLI's own
command:

```
glassfrog auth login
```

This skill introduces **no credential mechanism of its own** — it stores
nothing, validates nothing, and never asks for the key directly; the CLI is the
single place the credential lives. For the full credential precedence and
options, see `glassfrog auth --help` and the orientation skill's credentials
section. After the fix, **re-run the auth check**: a failing re-check routes
back here, never to a ready report.

## Ready report

When the presence check and the auth check have both passed — including any
re-checks after fixes — report the environment **ready to drive the CLI**
through the operating surface: name the CLI version the presence check printed
and the identity the auth check returned.

## Boundary

Setup owns the **journey**: check → fix → verify. The orientation skill owns
the **reference**: how credentials work, what exit codes mean, which output
formats exist. This skill links there and never restates it; per-command detail
comes from `glassfrog <command> --help`.
