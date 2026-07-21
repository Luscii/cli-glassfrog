# Glassfrog CLI

A command-line surface over the [Glassfrog](https://www.glassfrog.com/) API v5 —
read your Holacracy governance record (roles, circles, accountabilities, and the
rest) and act on it through proposals. Built to be driven by AI agents as well
as by hand. `glassfrog` ships as a single self-contained executable: no runtime,
no package manager required.

## Installation

Install the latest stable release with one command (macOS or Linux):

```sh
curl -fsSL https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh | sh
```

No `curl`? Use `wget`:

```sh
wget -qO- https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh | sh
```

The script detects your platform, downloads the matching release archive from
GitHub Releases, **verifies its sha256 checksum**, and installs the `glassfrog`
binary into a per-user directory — **no `sudo`, no shell-profile edits**. If the
install directory is not on your `PATH`, the script prints the exact
`export PATH=…` line to add it. Re-running the one-liner upgrades in place (and
pinning an older version downgrades).

### Supported platforms

macOS and Linux, on `amd64` (`x86_64`) and `arm64` (`aarch64`). Windows is not
supported. An unsupported platform is refused with a clear message and nothing
is installed.

### Configuration

The installer is configured through environment variables. Because the command
is piped into `sh`, set them on the **`sh` invocation** (the right side of the
pipe), not before `curl`:

```sh
curl -fsSL https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh \
  | GLASSFROG_VERSION=v1.3.0 GLASSFROG_INSTALL_DIR="$HOME/bin" sh
```

| Variable | Default | Description |
|---|---|---|
| `GLASSFROG_VERSION` | _(latest stable)_ | Release to install — a tag with or without the leading `v` (`v1.3.0` or `1.3.0`). Unset installs the latest **stable** release (pre-releases excluded); pinning a tag installs that exact release, including a pre-release. |
| `GLASSFROG_INSTALL_DIR` | `${XDG_BIN_HOME:-$HOME/.local/bin}` | Directory to install into. Created if absent. Must be writable by you — the script never escalates privileges. |
| `GLASSFROG_DOWNLOAD_BASE_URL` | `https://github.com` | Base URL for release resolution and downloads (a mirror / enterprise / test seam). |

> **Note:** `GLASSFROG_DOWNLOAD_BASE_URL` configures where the **installer**
> fetches release archives from. It is deliberately distinct from the CLI's own
> `GLASSFROG_BASE_URL`, which overrides the **API** endpoint the installed
> `glassfrog` talks to. Exporting one does not affect the other.

## Install via npm

If you work in a Node toolchain, install the same binary through npm. The
`@luscii-healthtech/glassfrog` package resolves and runs the matching platform
binary: npm installs only the platform package for your OS and CPU, and a
zero-dependency launcher execs that binary with arguments and exit codes passed
straight through.

Run it once, without installing:

```sh
npx @luscii-healthtech/glassfrog --version
```

Install globally:

```sh
npm i -g @luscii-healthtech/glassfrog
glassfrog --version
```

Pin a specific version (a stable release or a pre-release):

```sh
npm i -g @luscii-healthtech/glassfrog@1.3.0
npm i -g @luscii-healthtech/glassfrog@1.4.0-rc.1
```

Supported platforms match the install script: macOS and Linux on `x64`
(`amd64`) and `arm64`. An unsupported platform (e.g. Windows) is refused at
install with a clear message and nothing runnable is placed. When the matching
platform package is available the install needs **no network**; otherwise a
postinstall fallback downloads the release archive, **verifies its sha256
checksum**, and only then places the binary.

> The npm package, the [install script](#installation) above, and the Homebrew
> tap are independent acquisition channels for the **same** released binary —
> pick whichever fits your environment. Maintainers: the npm channel publishes
> over GitHub OIDC trusted publishing, which requires a
> [one-time per-package setup](scripts/npm-trusted-publishers.md) before the
> first release.

## Install the agent operating surface

If an AI agent drives the CLI for you, the repo also ships a Claude plugin — the
**agent operating surface**: orientation knowledge, operator-path skills, and a
write-safety hook that equip an agent to drive `glassfrog` correctly and safely.
In a Claude plugin host (e.g. Claude Code), add this repository as a plugin
marketplace, then install the plugin from it:

```
/plugin marketplace add Luscii/cli-glassfrog
/plugin install glassfrog@glassfrog
```

The plugin is knowledge and guardrails only — it adds no command and no API
capability. The **CLI itself is a prerequisite**: install the `glassfrog` binary
first through any of the channels above (the [install script](#installation),
[Homebrew](docs/guides/installation/how-to-install.md), or [npm](#install-via-npm)).
After installing the plugin, the `glassfrog-setup` skill checks that the binary
is present and authenticated and directs you to the right fix when either is
missing. See [How to install the operating surface](docs/guides/agent-operators/how-to-install-the-operating-surface.md).

## Documentation

See [`docs/`](docs/) for guides, reference, and explanation, organised along the
[Diátaxis](https://diataxis.fr/) framework.
