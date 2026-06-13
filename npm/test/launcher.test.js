// Unit tests for the launcher (bin/glassfrog), spec 037 T001.
//
// The launcher is exec'd as a real child process against a stub binary placed at
// the postinstall location, so argv passthrough, exit-code propagation, and
// signal re-raising are observed end-to-end — no real install, no network.
'use strict';

const { test, before, after } = require('node:test');
const assert = require('node:assert');
const path = require('path');
const {
	mkTmp,
	rmrf,
	copyUmbrellaSources,
	writeStubBinary,
	runLauncher,
} = require('./helpers.js');

let pkgRoot;

before(() => {
	pkgRoot = mkTmp('glassfrog-launcher-');
	copyUmbrellaSources(pkgRoot);
	// Place a stub binary where the postinstall fallback would (binary/glassfrog),
	// so the launcher resolves it via the placed-path branch.
	writeStubBinary(path.join(pkgRoot, 'binary', 'glassfrog'));
});

after(() => rmrf(pkgRoot));

test('forwards all arguments to the binary unchanged', () => {
	const r = runLauncher(pkgRoot, ['roles', 'list', '--json', '--limit', '5']);
	assert.strictEqual(r.status, 0);
	assert.match(r.stdout, /ARGS:\["roles","list","--json","--limit","5"\]/);
});

test('exits with the binary exit code unchanged', () => {
	for (const code of [0, 1, 2, 6, 42]) {
		const r = runLauncher(pkgRoot, ['--exit', String(code)]);
		assert.strictEqual(r.status, code, `expected exit ${code}`);
		assert.strictEqual(r.signal, null);
	}
});

test('re-raises a terminating signal from the binary', () => {
	const r = runLauncher(pkgRoot, ['--signal']);
	// A signal-terminated child makes the launcher die by the same signal: the
	// spawn result carries signal (not status).
	assert.strictEqual(r.signal, 'SIGTERM');
	assert.strictEqual(r.status, null);
});

test('refuses clearly and exits non-zero when no binary resolves (backstop)', () => {
	const empty = mkTmp('glassfrog-launcher-empty-');
	copyUmbrellaSources(empty);
	// No binary/ placed and no platform package installed → the backstop fires.
	try {
		const r = runLauncher(empty, ['--version']);
		assert.notStrictEqual(r.status, 0);
		assert.match(r.stderr, /no runnable binary|Unsupported platform/);
		assert.match(r.stderr, /darwin-arm64, darwin-x64, linux-arm64, linux-x64/);
		assert.match(r.stderr, /--ignore-scripts|install script|Homebrew/);
		assert.strictEqual(r.stdout, '', 'launcher writes nothing to stdout of its own');
	} finally {
		rmrf(empty);
	}
});
