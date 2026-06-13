// Tests for the package generator (build.mjs), spec 037 T003.
//
// The pure shape assertions (os/cpu map, =version pinning, bin placement) need no
// tar; the full generate() test builds a fixture dist/ of 022-named archives and
// asserts the emitted umbrella + four platform package directories.
import { test } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { npmVersion, platformPackageJson, umbrellaPackageJson, generate } from '../build.mjs';

const require = createRequire(import.meta.url);
const platform = require('../lib/platform.js');
const { mkTmp, rmrf, makeArchive, hasTar } = require('./helpers.js');

test('strips the leading v to form the npm version', () => {
	assert.strictEqual(npmVersion('v1.3.0'), '1.3.0');
	assert.strictEqual(npmVersion('v1.4.0-rc.1'), '1.4.0-rc.1');
	assert.strictEqual(npmVersion('1.3.0'), '1.3.0');
});

test('platform package.json carries the os/cpu map, files, and no bin', () => {
	const pkg = platformPackageJson(
		{ os: 'darwin', cpu: 'x64', goos: 'darwin', goarch: 'amd64' },
		'1.3.0',
	);
	assert.strictEqual(pkg.name, '@luscii-healthtech/glassfrog-darwin-x64');
	assert.strictEqual(pkg.version, '1.3.0');
	assert.deepStrictEqual(pkg.os, ['darwin']);
	assert.deepStrictEqual(pkg.cpu, ['x64']);
	assert.deepStrictEqual(pkg.files, ['bin/glassfrog']);
	assert.strictEqual(pkg.bin, undefined, 'platform packages declare no bin');
	assert.strictEqual(pkg.publishConfig.access, 'public');
});

test('umbrella package.json pins all four optional deps to =version', () => {
	const template = JSON.parse(
		fs.readFileSync(path.join(import.meta.dirname, '..', 'package.json'), 'utf8'),
	);
	const pkg = umbrellaPackageJson(template, '1.3.0');
	assert.strictEqual(pkg.name, '@luscii-healthtech/glassfrog');
	assert.strictEqual(pkg.version, '1.3.0');
	assert.deepStrictEqual(pkg.bin, { glassfrog: 'bin/glassfrog' });
	assert.strictEqual(pkg.scripts.postinstall, 'node postinstall.js');
	assert.strictEqual(pkg.scripts.test, undefined, 'dev test script dropped from publish');
	assert.deepStrictEqual(pkg.optionalDependencies, {
		'@luscii-healthtech/glassfrog-darwin-arm64': '=1.3.0',
		'@luscii-healthtech/glassfrog-darwin-x64': '=1.3.0',
		'@luscii-healthtech/glassfrog-linux-arm64': '=1.3.0',
		'@luscii-healthtech/glassfrog-linux-x64': '=1.3.0',
	});
	assert.strictEqual(pkg.publishConfig.access, 'public');
});

test('generate() emits one umbrella + four platform packages from a dist/', { skip: !hasTar() }, () => {
	const ver = '1.3.0';
	const distDir = mkTmp('glassfrog-dist-');
	const outDir = mkTmp('glassfrog-out-');
	try {
		// Build a fixture archive per target at 022's exact asset name.
		for (const t of platform.SUPPORTED) {
			const work = mkTmp('glassfrog-arc-');
			const { archivePath } = makeArchive(work, `#!/bin/sh\necho ${t.os}-${t.cpu}\n`);
			const { archive } = platform.assetNames(ver, t.goos, t.goarch);
			fs.copyFileSync(archivePath, path.join(distDir, archive));
			rmrf(work);
		}

		const result = generate({ version: `v${ver}`, distDir, outDir });
		assert.strictEqual(result.platforms.length, 4);

		// Umbrella shape + copied sources.
		const umb = JSON.parse(fs.readFileSync(path.join(outDir, 'glassfrog', 'package.json'), 'utf8'));
		assert.strictEqual(umb.name, '@luscii-healthtech/glassfrog');
		assert.strictEqual(umb.version, ver);
		assert.ok(fs.existsSync(path.join(outDir, 'glassfrog', 'bin', 'glassfrog')));
		assert.ok(fs.existsSync(path.join(outDir, 'glassfrog', 'postinstall.js')));
		assert.ok(fs.existsSync(path.join(outDir, 'glassfrog', 'lib', 'platform.js')));

		// Each platform package: shape + bundled binary.
		for (const t of platform.SUPPORTED) {
			const dir = path.join(outDir, `glassfrog-${t.os}-${t.cpu}`);
			const pkg = JSON.parse(fs.readFileSync(path.join(dir, 'package.json'), 'utf8'));
			assert.strictEqual(pkg.name, `@luscii-healthtech/glassfrog-${t.os}-${t.cpu}`);
			assert.deepStrictEqual(pkg.os, [t.os]);
			assert.deepStrictEqual(pkg.cpu, [t.cpu]);
			assert.strictEqual(pkg.bin, undefined);
			const bin = path.join(dir, 'bin', 'glassfrog');
			assert.ok(fs.existsSync(bin), `binary bundled for ${t.os}-${t.cpu}`);
			assert.match(fs.readFileSync(bin, 'utf8'), new RegExp(`${t.os}-${t.cpu}`));
		}
	} finally {
		rmrf(distDir);
		rmrf(outDir);
	}
});

test('generate() fails loudly when a target archive is missing', () => {
	const distDir = mkTmp('glassfrog-dist-empty-');
	const outDir = mkTmp('glassfrog-out-');
	try {
		assert.throws(() => generate({ version: 'v1.3.0', distDir, outDir }), /missing release archive/);
	} finally {
		rmrf(distDir);
		rmrf(outDir);
	}
});
