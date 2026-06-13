// Tests for the postinstall fallback (postinstall.js), spec 037 T002.
//
// Driven against a localhost fixture server via the GLASSFROG_DOWNLOAD_BASE_URL
// seam (here, the baseUrl opt) — no network, no real registry. Fixtures encode
// 022's exact asset names, so a name_template drift breaks these tests (Risk R1).
'use strict';

const { test } = require('node:test');
const assert = require('node:assert');
const fs = require('fs');
const path = require('path');
const platform = require('../lib/platform.js');
const { postinstall, InstallError } = require('../postinstall.js');
const { mkTmp, rmrf, makeArchive, startReleaseServer, hasTar } = require('./helpers.js');

const VER = '1.3.0';
const TAG = `v${VER}`;
// The host target (the suite runs on a supported platform: darwin/linux).
const TARGET = platform.detectTarget();
const NAMES = platform.assetNames(VER, TARGET.goos, TARGET.goarch);

// makeUmbrella() — a temp umbrella package root carrying its stamped version.
function makeUmbrella() {
	const root = mkTmp('glassfrog-postinstall-');
	fs.writeFileSync(
		path.join(root, 'package.json'),
		JSON.stringify({ name: platform.UMBRELLA, version: VER }),
	);
	return root;
}

// serveRelease(assets) — start the fixture server for this version's tag.
function serveRelease(assets) {
	return startReleaseServer({ tag: TAG, assets });
}

test('happy fallback downloads, verifies, and places the binary', { skip: !hasTar() }, async () => {
	const work = mkTmp('glassfrog-fixture-');
	const { archivePath, sha256 } = makeArchive(work, '#!/bin/sh\necho real-binary\n');
	const archiveBytes = fs.readFileSync(archivePath);
	const server = await serveRelease({
		[NAMES.archive]: archiveBytes,
		[NAMES.checksums]: `${sha256}  ${NAMES.archive}\n`,
	});
	const root = makeUmbrella();
	try {
		const result = await postinstall({ baseUrl: server.baseUrl, pkgRoot: root, version: VER });
		assert.strictEqual(result.placed, true);
		const placed = platform.placedBinaryPath(root);
		assert.ok(fs.existsSync(placed), 'binary placed');
		assert.match(fs.readFileSync(placed, 'utf8'), /real-binary/);
		assert.ok(fs.statSync(placed).mode & 0o111, 'placed binary is executable');
	} finally {
		await server.close();
		rmrf(work);
		rmrf(root);
	}
});

test('bundled platform package is a no-op with no network', async () => {
	const root = makeUmbrella();
	// Install a fake matching platform package so the bundled binary resolves.
	const pkgName = platform.platformPackageName(TARGET.os, TARGET.cpu);
	const pkgDir = path.join(root, 'node_modules', pkgName, 'bin');
	fs.mkdirSync(pkgDir, { recursive: true });
	fs.writeFileSync(path.join(pkgDir, 'glassfrog'), '#!/bin/sh\necho bundled\n');
	fs.writeFileSync(
		path.join(root, 'node_modules', pkgName, 'package.json'),
		JSON.stringify({ name: pkgName, version: VER }),
	);
	const server = await serveRelease({});
	try {
		const result = await postinstall({ baseUrl: server.baseUrl, pkgRoot: root, version: VER });
		assert.strictEqual(result.reason, 'bundled');
		assert.strictEqual(server.hits.count, 0, 'no network on the bundled path');
		assert.ok(!fs.existsSync(platform.placedBinaryPath(root)), 'nothing placed in binary/');
	} finally {
		await server.close();
		rmrf(root);
	}
});

test('checksum mismatch aborts before placing the binary', { skip: !hasTar() }, async () => {
	const work = mkTmp('glassfrog-fixture-');
	const { archivePath } = makeArchive(work, '#!/bin/sh\necho tampered\n');
	const archiveBytes = fs.readFileSync(archivePath);
	const server = await serveRelease({
		[NAMES.archive]: archiveBytes,
		// A deliberately wrong checksum entry.
		[NAMES.checksums]: `${'0'.repeat(64)}  ${NAMES.archive}\n`,
	});
	const root = makeUmbrella();
	try {
		await assert.rejects(
			postinstall({ baseUrl: server.baseUrl, pkgRoot: root, version: VER }),
			(err) => err instanceof InstallError && /integrity check failed/.test(err.message),
		);
		assert.ok(!fs.existsSync(platform.placedBinaryPath(root)), 'no binary placed on mismatch');
	} finally {
		await server.close();
		rmrf(work);
		rmrf(root);
	}
});

test('unsupported platform refuses and places nothing', async () => {
	const root = makeUmbrella();
	try {
		await assert.rejects(
			postinstall({ platform: 'win32', arch: 'x64', pkgRoot: root, version: VER }),
			(err) =>
				err instanceof InstallError &&
				/Unsupported platform: win32\/x64/.test(err.message) &&
				/darwin-arm64, darwin-x64, linux-arm64, linux-x64/.test(err.message),
		);
		assert.ok(!fs.existsSync(platform.placedBinaryPath(root)), 'nothing placed');
	} finally {
		rmrf(root);
	}
});

test('missing tar fails before any download or placement', async () => {
	const root = makeUmbrella();
	const emptyBin = mkTmp('glassfrog-nopath-');
	const savedPath = process.env.PATH;
	let serverHit = false;
	const server = await startReleaseServer({ tag: TAG, assets: {} });
	// Hijack hits to detect any network attempt.
	const origHits = server.hits;
	try {
		process.env.PATH = emptyBin; // no tar resolvable
		await assert.rejects(
			postinstall({ baseUrl: server.baseUrl, pkgRoot: root, version: VER }),
			(err) => err instanceof InstallError && /tar was not found/.test(err.message),
		);
		serverHit = origHits.count > 0;
		assert.strictEqual(serverHit, false, 'no download attempted before the tar probe');
		assert.ok(!fs.existsSync(platform.placedBinaryPath(root)), 'nothing placed');
	} finally {
		process.env.PATH = savedPath;
		await server.close();
		rmrf(root);
		rmrf(emptyBin);
	}
});
