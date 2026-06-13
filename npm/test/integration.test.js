// Cross-unit / end-to-end test for the npm channel (spec 037 T004).
//
// No single unit owns this: it drives a full fallback install (postinstall places
// the binary from the fixture server) and then runs the launcher against the
// placed binary, asserting argument + exit-code passthrough. Hermetic — a
// localhost fixture server via the GLASSFROG_DOWNLOAD_BASE_URL seam, no network.
'use strict';

const { test } = require('node:test');
const assert = require('node:assert');
const fs = require('fs');
const path = require('path');
const platform = require('../lib/platform.js');
const { postinstall } = require('../postinstall.js');
const {
	mkTmp,
	rmrf,
	copyUmbrellaSources,
	makeArchive,
	startReleaseServer,
	runLauncher,
	hasTar,
} = require('./helpers.js');

const VER = '1.3.0';
const TAG = `v${VER}`;
const TARGET = platform.detectTarget();
const NAMES = platform.assetNames(VER, TARGET.goos, TARGET.goarch);

// A passthrough stub binary: echoes its arguments and honours `fail` → exit 7.
const STUB_BODY = ['#!/bin/sh', 'echo "ARGS:$*"', 'if [ "$1" = "fail" ]; then exit 7; fi', 'exit 0', ''].join('\n');

test('fallback install then launcher exec passes args and exit code through', { skip: !hasTar() }, async () => {
	const work = mkTmp('glassfrog-e2e-arc-');
	const { archivePath, sha256 } = makeArchive(work, STUB_BODY);
	const archiveBytes = fs.readFileSync(archivePath);
	const server = await startReleaseServer({
		tag: TAG,
		assets: {
			[NAMES.archive]: archiveBytes,
			[NAMES.checksums]: `${sha256}  ${NAMES.archive}\n`,
		},
	});

	const root = mkTmp('glassfrog-e2e-pkg-');
	copyUmbrellaSources(root);
	fs.writeFileSync(
		path.join(root, 'package.json'),
		JSON.stringify({ name: platform.UMBRELLA, version: VER }),
	);

	try {
		// Install (fallback path): downloads, verifies, places.
		const result = await postinstall({
			baseUrl: server.baseUrl,
			pkgRoot: root,
			version: VER,
			log: () => {},
		});
		assert.strictEqual(result.placed, true, 'fallback placed the binary');

		// Launch: arguments forwarded, exit 0 on success.
		const ok = runLauncher(root, ['roles', 'list', '--json']);
		assert.strictEqual(ok.status, 0);
		assert.match(ok.stdout, /ARGS:roles list --json/);

		// Launch: the binary's non-zero exit code propagates unchanged.
		const failed = runLauncher(root, ['fail']);
		assert.strictEqual(failed.status, 7);
	} finally {
		await server.close();
		rmrf(work);
		rmrf(root);
	}
});
