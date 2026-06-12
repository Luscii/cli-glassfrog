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

## Documentation

See [`docs/`](docs/) for guides, reference, and explanation, organised along the
[Diátaxis](https://diataxis.fr/) framework.
