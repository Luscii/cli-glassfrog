// Unit tests for the shared platform library (lib/platform.js), spec 037.
//
// Pure-function coverage that needs no install and no network: the os/cpu↔goos/
// goarch map (ADR-6), the 022 asset-name template (ADR-3 / Risk R1), and the
// supported/unsupported detection that drives both refusal surfaces (ADR-5).
'use strict';

const { test } = require('node:test');
const assert = require('node:assert');
const platform = require('../lib/platform.js');

test('detects exactly the four supported targets', () => {
	assert.deepStrictEqual(
		platform.SUPPORTED.map((t) => `${t.os}-${t.cpu}`).sort(),
		['darwin-arm64', 'darwin-x64', 'linux-arm64', 'linux-x64'],
	);
});

test('maps npm os/cpu to GoReleaser goos/goarch (amd64↔x64, arm64↔arm64)', () => {
	assert.deepStrictEqual(platform.detectTarget('darwin', 'x64'), {
		os: 'darwin',
		cpu: 'x64',
		goos: 'darwin',
		goarch: 'amd64',
	});
	assert.deepStrictEqual(platform.detectTarget('linux', 'arm64'), {
		os: 'linux',
		cpu: 'arm64',
		goos: 'linux',
		goarch: 'arm64',
	});
});

test('returns null for unsupported platforms and arches', () => {
	assert.strictEqual(platform.detectTarget('win32', 'x64'), null);
	assert.strictEqual(platform.detectTarget('linux', 'ia32'), null);
	assert.strictEqual(platform.detectTarget('freebsd', 'arm64'), null);
});

test('builds the 022 asset names from the tag-without-v version', () => {
	// Pins 022's .goreleaser.yaml name_template: a drift breaks this test.
	assert.deepStrictEqual(platform.assetNames('1.4.0', 'linux', 'amd64'), {
		archive: 'glassfrog_1.4.0_linux_amd64.tar.gz',
		checksums: 'glassfrog_1.4.0_checksums.txt',
	});
	assert.deepStrictEqual(platform.assetNames('1.4.0-rc.1', 'darwin', 'arm64'), {
		archive: 'glassfrog_1.4.0-rc.1_darwin_arm64.tar.gz',
		checksums: 'glassfrog_1.4.0-rc.1_checksums.txt',
	});
});

test('constructs the release download base like install.sh (027)', () => {
	assert.strictEqual(
		platform.downloadBase('https://github.com', 'v1.4.0'),
		'https://github.com/Luscii/cli-glassfrog/releases/download/v1.4.0',
	);
});

test('names platform packages under the @luscii-healthtech scope', () => {
	assert.strictEqual(
		platform.platformPackageName('darwin', 'arm64'),
		'@luscii-healthtech/glassfrog-darwin-arm64',
	);
	assert.strictEqual(platform.UMBRELLA, '@luscii-healthtech/glassfrog');
});

test('unsupported message names the platform, the four targets, and a next step', () => {
	const msg = platform.unsupportedMessage('win32', 'x64');
	assert.match(msg, /win32\/x64/);
	assert.match(msg, /darwin-arm64, darwin-x64, linux-arm64, linux-x64/);
	assert.match(msg, /Next step:/);
});

test('backstop message advises --ignore-scripts recovery on a supported host', () => {
	const msg = platform.noBinaryMessage('linux', 'x64');
	assert.match(msg, /--ignore-scripts/);
	assert.match(msg, /reinstall/);
	// On an unsupported host the backstop falls through to the unsupported refusal.
	assert.match(platform.noBinaryMessage('win32', 'x64'), /Unsupported platform/);
});
