// Shared platform vocabulary for the npm acquisition channel (spec 037).
//
// Both the launcher (bin/glassfrog) and the postinstall fallback (postinstall.js)
// — and, via createRequire, the generator (build.mjs) — read the SAME supported
// set, os/cpu↔goos/goarch map, package names, asset-name template, and refusal
// wording from here. One copy, so the GoReleaser→npm arch mapping (ADR-6) and the
// 022 asset-name template (ADR-3) cannot drift between the three surfaces.
//
// Zero npm dependencies — Node built-ins only (CONSTITUTION-clean, audit-free).
'use strict';

const fs = require('fs');
const path = require('path');

// The package scope and the published package names (interface-spec § Surface).
const SCOPE = '@luscii-healthtech';
const UMBRELLA = `${SCOPE}/glassfrog`;
const BINARY = 'glassfrog';

// The GitHub repository path the release assets live under (mirrors install.sh,
// spec 027). Used to construct the fallback download URL.
const REPO_PATH = 'Luscii/cli-glassfrog';

// The four supported targets. `os`/`cpu` are npm's vocabulary (process.platform /
// process.arch and the package.json `os`/`cpu` fields); `goos`/`goarch` are
// GoReleaser's, used only to construct the 022 asset names. The map is the whole
// of ADR-6's GoReleaser→npm translation: amd64→x64, arm64→arm64, os unchanged.
const SUPPORTED = [
	{ os: 'darwin', cpu: 'arm64', goos: 'darwin', goarch: 'arm64' },
	{ os: 'darwin', cpu: 'x64', goos: 'darwin', goarch: 'amd64' },
	{ os: 'linux', cpu: 'arm64', goos: 'linux', goarch: 'arm64' },
	{ os: 'linux', cpu: 'x64', goos: 'linux', goarch: 'amd64' },
];

// platformPackageName(os, cpu) — the scoped name of a per-platform package.
function platformPackageName(os, cpu) {
	return `${SCOPE}/${BINARY}-${os}-${cpu}`;
}

// A human-readable list of the four targets for refusal messages.
const TARGET_LIST = SUPPORTED.map((t) => `${t.os}-${t.cpu}`).join(', ');

// detectTarget(platform, arch) — the SUPPORTED entry matching a host, or null.
// platform/arch default to the running host's. npm's `os`/`cpu` vocabulary is
// exactly process.platform/process.arch, so the match is a direct equality.
function detectTarget(platform = process.platform, arch = process.arch) {
	return SUPPORTED.find((t) => t.os === platform && t.cpu === arch) || null;
}

// assetNames(ver, goos, goarch) — the 022 archive + checksums asset names.
// `ver` is the tag WITHOUT the leading `v` (GoReleaser's {{ .Version }}). This is
// the hard coupling to 022's .goreleaser.yaml `name_template`; the test fixtures
// encode these exact names so a template change breaks a test (ADR-3 / Risk R1).
function assetNames(ver, goos, goarch) {
	return {
		archive: `${BINARY}_${ver}_${goos}_${goarch}.tar.gz`,
		checksums: `${BINARY}_${ver}_checksums.txt`,
	};
}

// downloadBase(baseUrl, tag) — the release-download directory URL for a tag.
// Mirrors install.sh: <base>/Luscii/cli-glassfrog/releases/download/<tag>.
function downloadBase(baseUrl, tag) {
	return `${baseUrl}/${REPO_PATH}/releases/download/${tag}`;
}

// placedBinaryPath(pkgRoot) — where the postinstall fallback places the binary
// inside the umbrella package, and where the launcher looks for it when no
// platform package is bundled. A directory the umbrella owns, never published.
function placedBinaryPath(pkgRoot) {
	return path.join(pkgRoot, 'binary', BINARY);
}

// resolveBundledBinary(target, pkgRoot) — the path to the bundled platform
// package's binary if its optional dependency is installed, else null. `paths`
// roots the lookup at the umbrella so an npm-hoisted platform package is found.
function resolveBundledBinary(target, pkgRoot) {
	try {
		return require.resolve(`${platformPackageName(target.os, target.cpu)}/bin/${BINARY}`, {
			paths: [pkgRoot],
		});
	} catch {
		return null;
	}
}

// resolveBinary(platform, arch, pkgRoot) — the runnable binary path for a host:
// the bundled platform package first (the offline primary path), else the
// postinstall-placed binary, else null (unsupported host, or postinstall skipped).
function resolveBinary(platform, arch, pkgRoot) {
	const target = detectTarget(platform, arch);
	if (!target) return null;
	const bundled = resolveBundledBinary(target, pkgRoot);
	if (bundled) return bundled;
	const placed = placedBinaryPath(pkgRoot);
	return fs.existsSync(placed) ? placed : null;
}

// unsupportedMessage(platform, arch) — install-time refusal naming the detected
// platform, the supported set, and a next step (CONSTITUTION II, Action
// Transparency). Mirrors 027's unsupported-platform wording for cross-channel
// consistency.
function unsupportedMessage(platform, arch) {
	return (
		`Unsupported platform: ${platform}/${arch}.\n` +
		`${UMBRELLA} provides binaries for: ${TARGET_LIST}.\n` +
		'Next step: use one of those platforms, or install via the install script ' +
		'(curl … | sh) or Homebrew, which cover the same release.'
	);
}

// noBinaryMessage(platform, arch) — the launcher's runtime backstop (ADR-4). On a
// supported host this means the postinstall was skipped (--ignore-scripts) and no
// platform package was installed; on an unsupported host it is the same refusal as
// install time.
function noBinaryMessage(platform, arch) {
	if (!detectTarget(platform, arch)) {
		return unsupportedMessage(platform, arch);
	}
	return (
		`glassfrog: no runnable binary was found for ${platform}/${arch}.\n` +
		`Supported platforms: ${TARGET_LIST}.\n` +
		'This usually means the package was installed with --ignore-scripts, which skips ' +
		'the postinstall that places the binary.\n' +
		`Next step: reinstall ${UMBRELLA} without --ignore-scripts.`
	);
}

module.exports = {
	SCOPE,
	UMBRELLA,
	BINARY,
	REPO_PATH,
	SUPPORTED,
	TARGET_LIST,
	platformPackageName,
	detectTarget,
	assetNames,
	downloadBase,
	placedBinaryPath,
	resolveBundledBinary,
	resolveBinary,
	unsupportedMessage,
	noBinaryMessage,
};
