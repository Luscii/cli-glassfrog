#!/bin/sh
# glassfrog installer — a POSIX one-liner that turns a bare Linux or macOS host
# into one with a working `glassfrog` binary on PATH. No runtime, no package
# manager, no sudo.
#
#   curl -fsSL https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh | sh
#   wget -qO-  https://raw.githubusercontent.com/Luscii/cli-glassfrog/main/install.sh | sh
#
# Configuration is environment-only (a piped `… | sh` cannot take flags). Set it
# on the `sh` invocation (the right side of the pipe), not before `curl`:
#
#   curl -fsSL <url> | GLASSFROG_VERSION=v1.3.0 GLASSFROG_INSTALL_DIR="$HOME/bin" sh
#
#   GLASSFROG_VERSION            release to install; unset → latest STABLE
#                                (pre-releases excluded). A tag with or without
#                                a leading `v`. A pinned tag installs that exact
#                                release, including a pre-release.
#   GLASSFROG_INSTALL_DIR        install directory; unset → ${XDG_BIN_HOME:-$HOME/.local/bin}
#   GLASSFROG_DOWNLOAD_BASE_URL  resolution/download base; unset → https://github.com.
#                                The test/mirror seam. Deliberately DISTINCT from
#                                the CLI's GLASSFROG_BASE_URL (the API endpoint).
#
# It downloads the archive + checksums the Automated Release Pipeline (spec 022)
# attaches to a GitHub Release, verifies the archive's sha256 against the
# checksums file, and only then places the binary. Nothing reaches the install
# dir until verification passes (CONSTITUTION I).
#
# Asset names are OWNED BY spec 022 (.goreleaser.yaml `archives`/`checksum`):
#   archive   = glassfrog_<ver>_<os>_<arch>.tar.gz
#   checksums = glassfrog_<ver>_checksums.txt
# where <ver> is the tag WITHOUT the leading `v`. A change to that name_template
# would 404 these downloads — the Go exec-test fixtures encode these exact names
# so drift breaks a test. Keep this in sync with .goreleaser.yaml.
#
# Exit codes (the installer's OWN small scheme — not the CLI's exitcode.go):
#   0 success | 1 runtime (download/404/extract/fs) | 2 usage/environment
#   (unsupported platform, missing tooling, bad version) | 3 integrity (checksum)
# All diagnostics go to stderr; only the success report goes to stdout.
#
# POSIX-sh discipline: no arrays, no [[ ]], no `pipefail`, no `local`. Function-
# internal variables are prefixed `_` so they cannot clobber a caller's globals
# (POSIX functions share one scope; this is the substitute for `local`).

set -eu

REPO_PATH="Luscii/cli-glassfrog"
BINARY="glassfrog"

# --- diagnostics ------------------------------------------------------------

err() { printf '%s\n' "$*" >&2; }
die_runtime() { err "$*"; exit 1; }
die_usage() { err "$*"; exit 2; }
die_integrity() { err "$*"; exit 3; }

# have COMMAND — true when COMMAND is on PATH.
have() { command -v "$1" >/dev/null 2>&1; }

# --- platform detection (pure) ----------------------------------------------

# detect_platform UNAME_S UNAME_M — echo "<os> <arch>" for a supported target;
# else print an actionable message naming the detected platform and the
# supported set, and return 2. Pure: takes uname output as arguments so it is
# unit-testable without shimming uname.
detect_platform() {
	_os_raw=$1
	_arch_raw=$2
	case "$_os_raw" in
		Darwin) _os=darwin ;;
		Linux) _os=linux ;;
		*) _os= ;;
	esac
	case "$_arch_raw" in
		x86_64 | amd64) _arch=amd64 ;;
		arm64 | aarch64) _arch=arm64 ;;
		*) _arch= ;;
	esac
	if [ -z "$_os" ] || [ -z "$_arch" ]; then
		err "unsupported platform: ${_os_raw}/${_arch_raw}."
		err "glassfrog supports darwin/linux on amd64/arm64."
		return 2
	fi
	printf '%s %s\n' "$_os" "$_arch"
}

# --- asset naming (pure) ----------------------------------------------------

# asset_names VER OS ARCH — echo "<archive> <checksums>". VER is the tag WITHOUT
# the leading `v` (GoReleaser's {{ .Version }}). Pure and unit-testable.
asset_names() {
	_ver=$1
	_os=$2
	_arch=$3
	printf '%s %s\n' \
		"${BINARY}_${_ver}_${_os}_${_arch}.tar.gz" \
		"${BINARY}_${_ver}_checksums.txt"
}

# normalize_tag TAG — ensure a single leading `v` (the download path uses the
# tag as the release publishes it; GoReleaser tags carry `v`).
normalize_tag() {
	case "$1" in
		v*) printf '%s\n' "$1" ;;
		*) printf 'v%s\n' "$1" ;;
	esac
}

# strip_v TAG — drop a leading `v` to form <ver> for the asset name.
strip_v() { printf '%s\n' "${1#v}"; }

# --- tooling detection ------------------------------------------------------

# DOWNLOADER, SHATOOL are set here for download/sha256_of to consume. Probing
# happens before any network access so a bare host fails fast (exit 2).
DOWNLOADER=
SHATOOL=

detect_tooling() {
	if have curl; then
		DOWNLOADER=curl
	elif have wget; then
		DOWNLOADER=wget
	else
		die_usage "no downloader found: install curl or wget."
	fi

	if have sha256sum; then
		SHATOOL=sha256sum
	elif have shasum; then
		SHATOOL=shasum
	elif have openssl; then
		SHATOOL=openssl
	else
		die_usage "no sha256 utility found: install sha256sum, shasum, or openssl."
	fi

	have tar || die_usage "tar is required but was not found."
}

# --- network primitives -----------------------------------------------------

# extract_location — read a Location header (case-insensitive) from piped HTTP
# response headers and echo its value (CR and surrounding space stripped).
extract_location() {
	while IFS= read -r _line; do
		case "$_line" in
			[Ll]ocation:*)
				_loc=${_line#*:}
				_loc=$(printf '%s' "$_loc" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
				printf '%s\n' "$_loc"
				return 0
				;;
		esac
	done
	return 0
}

# resolve_redirect URL — echo the redirect target of URL without following it.
resolve_redirect() {
	_url=$1
	case "$DOWNLOADER" in
		curl) curl -sI "$_url" 2>/dev/null | extract_location ;;
		wget) wget -S --max-redirect=0 -O /dev/null "$_url" 2>&1 | extract_location ;;
	esac
}

# download URL DEST — fetch URL to DEST, failing (non-zero) on a 404/error.
download() {
	_url=$1
	_dest=$2
	case "$DOWNLOADER" in
		curl) curl -fsSL -o "$_dest" "$_url" ;;
		wget) wget -q -O "$_dest" "$_url" ;;
	esac
}

# sha256_of FILE — echo the lowercase hex sha256 of FILE using SHATOOL.
sha256_of() {
	_f=$1
	case "$SHATOOL" in
		sha256sum) sha256sum "$_f" | awk '{print $1}' ;;
		shasum) shasum -a 256 "$_f" | awk '{print $1}' ;;
		openssl) openssl dgst -sha256 "$_f" | awk '{print $NF}' ;;
	esac
}

# --- tag resolution ---------------------------------------------------------

# resolve_tag — echo the release tag (carrying `v`) to install. Default: follow
# the `releases/latest` redirect (GitHub excludes pre-releases by definition).
# Pinned: use GLASSFROG_VERSION verbatim. Validation of a pinned value happens
# before this is called.
resolve_tag() {
	_pinned=${GLASSFROG_VERSION:-}
	if [ -n "$_pinned" ]; then
		normalize_tag "$_pinned"
		return 0
	fi
	_loc=$(resolve_redirect "${BASE_URL}/${REPO_PATH}/releases/latest")
	if [ -z "$_loc" ]; then
		die_runtime "could not resolve the latest stable release (no redirect target)."
	fi
	# The redirect points at …/releases/tag/<tag>; the tag is the last path
	# segment.
	_tag=${_loc##*/}
	if [ -z "$_tag" ] || [ "$_tag" = "$_loc" ]; then
		die_runtime "could not parse a release tag from: ${_loc}"
	fi
	printf '%s\n' "$_tag"
}

# --- verification -----------------------------------------------------------

# verify_checksum ARCHIVE CHECKSUMS NAME — return 0 when ARCHIVE's sha256 equals
# the entry for NAME in the CHECKSUMS file; else return 3. Caller treats a
# non-zero return as an integrity abort. Reads the filesystem only — no network.
verify_checksum() {
	_archive=$1
	_sums=$2
	_name=$3
	# GoReleaser checksums lines: "<hash>  <filename>"; $2 is the filename.
	_expected=$(awk -v n="$_name" '$2 == n {print $1; exit}' "$_sums")
	if [ -z "$_expected" ]; then
		err "no checksum entry for ${_name} in the checksums file."
		return 3
	fi
	_actual=$(sha256_of "$_archive")
	if [ "$_expected" != "$_actual" ]; then
		err "integrity check failed for ${_name}: expected ${_expected}, got ${_actual}."
		return 3
	fi
}

# --- install ----------------------------------------------------------------

# install_binary SRC DIR — place SRC at DIR/<binary>, creating DIR if absent and
# overwriting any existing binary (the upgrade path). The mv is the only
# mutation of the user's environment and happens last.
install_binary() {
	_src=$1
	_dir=$2
	if ! mkdir -p "$_dir"; then
		die_usage "install directory is not creatable: ${_dir}"
	fi
	if ! mv -f "$_src" "${_dir}/${BINARY}"; then
		die_runtime "failed to install the binary into ${_dir}"
	fi
	chmod +x "${_dir}/${BINARY}" 2>/dev/null || true
}

# check_path DIR — if DIR is not on $PATH, print (to stdout) the exact line to
# add it. Never edits a shell profile.
check_path() {
	_dir=$1
	case ":${PATH}:" in
		*":${_dir}:"*) ;;
		*)
			printf '%s is not on your PATH. Add it with:\n' "$_dir"
			# The literal `$PATH` is intentional: this line is meant to be
			# pasted into the user's shell, where $PATH expands at that time.
			# shellcheck disable=SC2016
			printf '  export PATH="%s:$PATH"\n' "$_dir"
			;;
	esac
}

# --- orchestration ----------------------------------------------------------

main() {
	BASE_URL=${GLASSFROG_DOWNLOAD_BASE_URL:-https://github.com}
	install_dir=${GLASSFROG_INSTALL_DIR:-${XDG_BIN_HOME:-$HOME/.local/bin}}

	# Validate a pinned version up front (exit 2): reject anything that could not
	# be a tag (whitespace, slashes, shell-significant characters).
	pinned=${GLASSFROG_VERSION:-}
	if [ -n "$pinned" ]; then
		case "$pinned" in
			*[!0-9A-Za-z.+_-]*) die_usage "GLASSFROG_VERSION '${pinned}' is not a valid version tag." ;;
		esac
	fi

	# Platform (exit 2 on an unsupported target).
	if ! platform=$(detect_platform "$(uname -s)" "$(uname -m)"); then
		exit 2
	fi
	os=${platform% *}
	arch=${platform#* }

	# Tooling (exit 2 before any download if a category is missing).
	detect_tooling

	# Resolve the release and derive the deterministic asset names.
	tag=$(resolve_tag)
	ver=$(strip_v "$tag")
	names=$(asset_names "$ver" "$os" "$arch")
	archive=${names% *}
	checksums=${names#* }

	dl_base="${BASE_URL}/${REPO_PATH}/releases/download/${tag}"

	# Everything fetched/verified/extracted happens in a private temp dir; an
	# EXIT trap removes it on every path. Nothing touches the install dir until
	# the checksum verifies.
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT INT TERM HUP

	if ! download "${dl_base}/${checksums}" "${tmp}/${checksums}"; then
		die_runtime "failed to download the checksums file: ${dl_base}/${checksums}"
	fi
	if ! download "${dl_base}/${archive}" "${tmp}/${archive}"; then
		die_runtime "failed to download the release archive: ${dl_base}/${archive} (does version ${tag} exist?)"
	fi

	if ! verify_checksum "${tmp}/${archive}" "${tmp}/${checksums}" "$archive"; then
		exit 3
	fi

	if ! tar -xzf "${tmp}/${archive}" -C "$tmp"; then
		die_runtime "failed to extract ${archive}"
	fi
	if [ ! -f "${tmp}/${BINARY}" ]; then
		die_runtime "the archive did not contain the expected ${BINARY} binary."
	fi

	install_binary "${tmp}/${BINARY}" "$install_dir"

	printf 'Installed %s %s to %s\n' "$BINARY" "$tag" "${install_dir}/${BINARY}"
	check_path "$install_dir"
}

# Run main only when executed, not when a test sources this file to exercise
# individual functions. A sourcing harness sets GLASSFROG_INSTALL_LIB=1.
if [ "${GLASSFROG_INSTALL_LIB:-}" != "1" ]; then
	main "$@"
fi
